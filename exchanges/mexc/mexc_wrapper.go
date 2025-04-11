package mexc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	websocketResponseMaxLimit = time.Second * 3
)

// SetDefaults sets the basic defaults for Mexc
func (me *MEXC) SetDefaults() {
	me.Name = "MEXC"
	me.Enabled = true
	me.Verbose = true
	me.API.CredentialsValidator.RequiresKey = true
	me.API.CredentialsValidator.RequiresSecret = true

	err := me.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: ""},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = me.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	me.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
	}
	me.Requester, err = request.New(me.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	me.API.Endpoints = me.NewEndpoints()
	me.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      spotAPIURL,
		exchange.RestFutures:   contractAPIURL,
		exchange.WebsocketSpot: wsURL,
	})
	me.Websocket = websocket.NewManager()
	me.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	me.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	me.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (me *MEXC) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		me.SetEnabled(false)
		return nil
	}
	err = me.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := me.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = me.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            wsURL,
		RunningURL:            wsRunningEndpoint,
		Connector:             me.WsConnect,
		Subscriber:            me.Subscribe,
		Unsubscriber:          me.Unsubscribe,
		GenerateSubscriptions: me.generateSubscriptions,
		Features:              &me.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}
	if err := me.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     websocketResponseMaxLimit,
		RateLimit:            request.NewRateLimitWithWeight(time.Second, 2, 1),
	}); err != nil {
		return err
	}
	return nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (me *MEXC) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		result, err := me.GetSymbols(ctx, nil)
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(result.Symbols))
		for i := range result.Symbols {
			if result.Symbols[i].Status.Int64() != 1 {
				continue
			}
			pair, err := currency.NewPairFromString(result.Symbols[i].Symbol)
			if err != nil {
				return nil, err
			}
			currencyPairs = append(currencyPairs, pair)
		}
		return currencyPairs, nil
	case asset.Futures:
		result, err := me.GetFuturesContracts(ctx, "")
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(result.Data))
		for i := range result.Data {
			switch result.Data[i].State {
			case 3, 4:
				continue
			}
			pair, err := currency.NewPairFromString(result.Data[i].Symbol)
			if err != nil {
				return nil, err
			}
			currencyPairs = append(currencyPairs, pair)
		}
		return currencyPairs, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (me *MEXC) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := me.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := me.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		err = me.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (me *MEXC) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pFormat, err := me.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		pairString := pFormat.Format(p)
		tickers, err := me.Get24HourTickerPriceChangeStatistics(ctx, []string{pairString})
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers {
			if tickers[t].Symbol != pairString {
				continue
			}
			found = true
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         p,
				ExchangeName: me.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				Volume:       tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			})
			if err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	case asset.Futures:
		pairString := pFormat.Format(p)
		tickers, err := me.GetContractTickers(ctx, pairString)
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers.Data {
			if tickers.Data[t].Symbol != pairString {
				continue
			}
			found = true
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         p,
				ExchangeName: me.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			})
			if err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return ticker.GetTicker(me.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (me *MEXC) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		tickers, err := me.Get24HourTickerPriceChangeStatistics(ctx, []string{})
		if err != nil {
			return err
		}
		for t := range tickers {
			pair, err := currency.NewPairFromString(tickers[t].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				ExchangeName: me.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				Volume:       tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		tickers, err := me.GetContractTickers(ctx, "")
		if err != nil {
			return err
		}
		for t := range tickers.Data {
			pair, err := currency.NewPairFromString(tickers.Data[t].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         pair,
				ExchangeName: me.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (me *MEXC) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(me.Name, p, assetType)
	if err != nil {
		return me.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (me *MEXC) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(me.Name, pair, assetType)
	if err != nil {
		return me.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (me *MEXC) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	book := &orderbook.Base{
		Exchange:        me.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: me.CanVerifyOrderbook,
	}
	pFormat, err := me.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := me.GetOrderbook(ctx, pFormat.Format(pair), 1000)
		if err != nil {
			return book, err
		}

		book.Bids = make([]orderbook.Tranche, len(result.Bids))
		for x := range result.Bids {
			book.Bids[x] = orderbook.Tranche{
				Price:  result.Bids[x][0].Float64(),
				Amount: result.Bids[x][1].Float64(),
			}
		}
		book.Asks = make([]orderbook.Tranche, len(result.Asks))
		for x := range result.Asks {
			book.Asks[x] = orderbook.Tranche{
				Price:  result.Asks[x][0].Float64(),
				Amount: result.Asks[x][1].Float64(),
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
		return orderbook.Get(me.Name, pair, assetType)
	case asset.Futures:
		result, err := me.GetContractDepthInformation(ctx, pFormat.Format(pair), 1000)
		if err != nil {
			return nil, err
		}
		book.Bids = make([]orderbook.Tranche, len(result.Bids))
		for x := range result.Bids {
			book.Bids[x] = orderbook.Tranche{
				Price:  result.Bids[x].Price,
				Amount: result.Bids[x].Amount,
			}
		}
		book.Asks = make([]orderbook.Tranche, len(result.Asks))
		for x := range result.Asks {
			book.Asks[x] = orderbook.Tranche{
				Price:  result.Asks[x].Price,
				Amount: result.Asks[x].Amount,
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
		return orderbook.Get(me.Name, pair, assetType)
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (me *MEXC) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	resp := account.Holdings{
		Exchange: me.Name,
	}
	subAccounts, err := me.GetSubAccountList(ctx, "", false, 0, 0)
	if err != nil {
		return resp, err
	}
	resp.Accounts = make([]account.SubAccount, len(subAccounts.SubAccounts))
	for sacc := range subAccounts.SubAccounts {
		accAssets, err := me.GetSubAccountAsset(ctx, subAccounts.SubAccounts[sacc].SubAccount, assetType)
		if err != nil {
			return resp, err
		}
		currBalances := make([]account.Balance, len(accAssets.Balances))
		for b := range accAssets.Balances {
			currBalances = append(currBalances, account.Balance{
				Currency: currency.NewCode(accAssets.Balances[b].Asset),
				Total:    accAssets.Balances[b].Free.Float64() + accAssets.Balances[b].Locked.Float64(),
				Hold:     accAssets.Balances[b].Locked.Float64(),
			})
		}
		resp.Accounts = append(resp.Accounts, account.SubAccount{
			ID:         subAccounts.SubAccounts[sacc].SubAccount,
			AssetType:  assetType,
			Currencies: currBalances,
		})
	}
	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (me *MEXC) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := me.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(me.Name, creds, assetType)
	if err != nil {
		return me.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (me *MEXC) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	var err error
	var resp []exchange.FundingHistory
	result, err := me.GetFundDepositHistory(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	for a := range result {
		var statusString string
		switch result[a].Status {
		case 1:
			statusString = "SMALL"
		case 2:
			statusString = "TIME_DELAY"
		case 3:
			statusString = "LARGE_DELAY"
		case 4:
			statusString = "PENDING"
		case 5:
			statusString = "SUCCESS"
		case 6:
			statusString = "AUDITING"
		case 7:
			statusString = "REJECTED"
		}
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    me.Name,
			Status:          statusString,
			TransferID:      result[a].TransactionID,
			Timestamp:       result[a].ConfirmTimes.Time(),
			Currency:        result[a].Coin,
			Amount:          result[a].Amount.Float64(),
			CryptoToAddress: result[a].Address,
			TransferType:    "diposit",
		})
	}
	withdrawals, err := me.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	for w := range withdrawals {
		var wdrStatus string
		switch withdrawals[w].Status {
		case 1:
			wdrStatus = "APPLY"
		case 2:
			wdrStatus = "AUDITING"
		case 3:
			wdrStatus = "WAIT"
		case 4:
			wdrStatus = "PROCESSING"
		case 5:
			wdrStatus = "WAIT_PACKAGING"
		case 6:
			wdrStatus = "WAIT_CONFIRM"
		case 7:
			wdrStatus = "SUCCESS"
		case 8:
			wdrStatus = "FAILED"
		case 9:
			wdrStatus = "CANCEL"
		case 10:
			wdrStatus = "MANUAL"
		}
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    me.Name,
			Status:          wdrStatus,
			TransferID:      withdrawals[w].TransactionID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin,
			Amount:          withdrawals[w].Amount.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			TransferType:    "withdrawal",
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (me *MEXC) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	var resp []exchange.WithdrawalHistory
	withdrawals, err := me.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	for w := range withdrawals {
		var wdrStatus string
		switch withdrawals[w].Status {
		case 1:
			wdrStatus = "APPLY"
		case 2:
			wdrStatus = "AUDITING"
		case 3:
			wdrStatus = "WAIT"
		case 4:
			wdrStatus = "PROCESSING"
		case 5:
			wdrStatus = "WAIT_PACKAGING"
		case 6:
			wdrStatus = "WAIT_CONFIRM"
		case 7:
			wdrStatus = "SUCCESS"
		case 8:
			wdrStatus = "FAILED"
		case 9:
			wdrStatus = "CANCEL"
		case 10:
			wdrStatus = "MANUAL"
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          wdrStatus,
			TransferID:      withdrawals[w].TransactionID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin,
			Amount:          withdrawals[w].Amount.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			TransferType:    "withdrawal",
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (me *MEXC) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := me.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := me.GetRecentTradesList(ctx, p.String(), 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result))
		for t := range result {
			var side order.Side
			if result[t].IsBuyerMaker {
				side = order.Buy
			} else {
				side = order.Sell
			}
			resp[t] = trade.Data{
				TID:          result[t].ID,
				Exchange:     me.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        result[t].Price.Float64(),
				Amount:       result[t].Quantity.Float64(),
				Timestamp:    result[t].Time.Time(),
			}
		}
		return resp, nil
	case asset.Futures:
		result, err := me.GetContractTransactionData(ctx, p.String(), 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result.Data))
		for t := range result.Data {
			var oSide order.Side
			switch result.Data[t].DealType {
			case 1:
				oSide = order.Long
			case 2:
				oSide = order.Short
			}
			resp[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         oSide,
				Price:        result.Data[t].TransactionPrice,
				Amount:       result.Data[t].Quantity,
				Timestamp:    result.Data[t].TransactionTime.Time(),
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w: asset type %v", asset.ErrNotSupported, assetType)
	}
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (me *MEXC) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	p, err := me.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Futures:
		result, err := me.GetContractTransactionData(ctx, p.String(), 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result.Data))
		for t := range result.Data {
			var oSide order.Side
			switch result.Data[t].DealType {
			case 1:
				oSide = order.Long
			case 2:
				oSide = order.Short
			}
			resp[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         oSide,
				Price:        result.Data[t].TransactionPrice,
				Amount:       result.Data[t].Quantity,
				Timestamp:    result.Data[t].TransactionTime.Time(),
			}
		}
		return resp, nil
	case asset.Spot:
		result, err := me.GetAggregatedTrades(ctx, p.String(), startTime, endTime, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result))
		for t := range result {
			var oSide order.Side
			if result[t].MakerBuyer {
				oSide = order.Buy
			} else {
				oSide = order.Sell
			}
			resp[t] = trade.Data{
				TID:          result[t].LastTradeID,
				Exchange:     me.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         oSide,
				Price:        result[t].Price.Float64(),
				Amount:       result[t].Quantity.Float64(),
				Timestamp:    result[t].Timestamp.Time(),
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// GetServerTime returns the current exchange server time.
func (me *MEXC) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	serverTime, err := me.GetSystemTime(ctx)
	return serverTime.Time(), err
}

// SubmitOrder submits a new order
func (me *MEXC) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(me.GetTradingRequirements()); err != nil {
		return nil, err
	}
	// When an order has been submitted you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.SubmitResponse if you think they are applicable.
	// resp, err := s.DeriveSubmitResponse( /*newOrderID*/)
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.Date = exampleTime // e.g. If this is supplied by the exchanges API.
	// return resp, nil
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (me *MEXC) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// When an order has been modified you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.ModifyResponse if you think they are applicable.
	// resp, err := action.DeriveModifyResponse()
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.OrderID = maybeANewOrderID // e.g. If this is supplied by the exchanges API.
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (me *MEXC) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (me *MEXC) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (me *MEXC) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (me *MEXC) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	pairFormat, err := me.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := me.GetOrderByID(ctx, pairFormat.Format(pair), "", orderID)
		if err != nil {
			return nil, err
		}
		oType, err := me.StringToOrderType(result.Type)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(result.Status)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(result.Symbol)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:                result.Price.Float64(),
			Amount:               result.CummulativeQuoteQty.Float64(),
			ContractAmount:       result.OrigQty.Float64(),
			AverageExecutedPrice: result.Price.Float64(),
			ExecutedAmount:       result.ExecutedQty.Float64(),
			RemainingAmount:      result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			Exchange:             me.Name,
			OrderID:              result.OrderID,
			ClientOrderID:        result.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Spot,
			LastUpdated:          result.TransactTime.Time(),
			Pair:                 cp,
		}, nil
	case asset.Futures:
		return nil, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", order.ErrAssetNotSet, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (me *MEXC) GetDepositAddress(ctx context.Context, c currency.Code, _ string, chain string) (*deposit.Address, error) {
	result, err := me.GenerateDepositAddress(ctx, currency.BTC, "")
	if err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, deposit.ErrAddressNotFound
	}
	return &deposit.Address{
		Address: result[0].Address,
		Tag:     result[0].Tag,
		Chain:   result[0].Network,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (me *MEXC) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details order.FilteredOrders
		for p := range getOrdersRequest.Pairs {
			result, err := me.GetOpenOrders(ctx, getOrdersRequest.Pairs[p].String())
			if err != nil {
				return nil, err
			}
			for r := range result {
				var oStatus order.Status
				switch result[r].Status {
				case "NEW":
					oStatus = order.New
				case "FILLED":
					oStatus = order.Filled
				case "PARTIALLY_FILLED":
					oStatus = order.PartiallyFilled
				case "CANCELED":
					oStatus = order.Cancelled
				case "PARTIALLY_CANCELED":
					oStatus = order.PartiallyCancelled
				}
				oSide, err := order.StringToOrderSide(result[r].Side)
				if err != nil {
					return nil, err
				}
				oType, err := order.StringToOrderType(result[r].Type)
				if err != nil {
					return nil, err
				}
				details = append(details, order.Detail{
					Price:                result[r].Price.Float64(),
					Amount:               result[r].OrigQty.Float64(),
					AverageExecutedPrice: result[r].Price.Float64(),
					QuoteAmount:          result[r].CummulativeQuoteQty.Float64(),
					ExecutedAmount:       result[r].ExecutedQty.Float64(),
					RemainingAmount:      result[r].OrigQty.Float64() - result[r].ExecutedQty.Float64(),
					Exchange:             me.Name,
					OrderID:              result[r].OrderID,
					ClientOrderID:        result[r].ClientOrderID,
					Type:                 oType,
					Side:                 oSide,
					Status:               oStatus,
					AssetType:            asset.Spot,
					LastUpdated:          result[r].TransactTime.Time(),
				})
			}
		}
		return details, nil
	case asset.Futures:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details order.FilteredOrders
		for p := range getOrdersRequest.Pairs {
			result, err := me.GetUserCurrentPendingOrder(ctx, getOrdersRequest.Pairs[p].String(), 0, 0)
			if err != nil {
				return nil, err
			}
			for od := range result.Data {
				var oType order.Type
				switch result.Data[od].OrderType {
				case 1:
					oType = order.Limit
				case 2:
					oType = order.PostOnly
				case 3:
					oType = order.ImmediateOrCancel
				case 4:
					oType = order.FillOrKill
				case 5:
					oType = order.Market
				case 6:
					oType = order.Chase
				}
				// order direction 1open long,2close short,3open short, 4 close long
				var oSide order.Side
				switch result.Data[od].Side {
				case 1, 4:
					oSide = order.Long
				case 2, 3:
					oSide = order.Short
				}
				// TODO: fix again
				var oStatus order.Status
				switch result.Data[od].State {
				case 1:
					oStatus = order.Active
				case 2:
					oStatus = order.AutoDeleverage
				case 3:
					oStatus = order.Closed
				}
				details = append(details, order.Detail{
					Leverage:             result.Data[od].Leverage,
					Price:                result.Data[od].Price,
					Amount:               result.Data[od].Volume,
					ContractAmount:       result.Data[od].DealVol,
					AverageExecutedPrice: result.Data[od].DealAvgPrice,
					QuoteAmount:          result.Data[od].DealAvgPrice * result.Data[od].DealVol,
					ExecutedAmount:       result.Data[od].Volume - result.Data[od].DealVol,
					RemainingAmount:      result.Data[od].Volume - result.Data[od].DealVol,
					FeeAsset:             currency.NewCode(result.Data[od].FeeCurrency),
					Exchange:             me.Name,
					OrderID:              result.Data[od].ExternalOrderID,
					ClientOrderID:        result.Data[od].ExternalOrderID,
					Type:                 oType,
					Side:                 oSide,
					Status:               oStatus,
					AssetType:            asset.Futures,
					LastUpdated:          result.Data[od].UpdateTime.Time(),
					Pair:                 getOrdersRequest.Pairs[p],
				})
			}
		}
		return details, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (me *MEXC) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (me *MEXC) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		if feeBuilder.IsMaker {
			return 0., nil
		}
		return 0.0005, nil
	case exchange.CryptocurrencyTradeFee:
		result, err := me.GetSymbolTradingFee(ctx, feeBuilder.Pair.String())
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			return result.Data.MakerCommission, nil
		}
		return result.Data.TakerCommission, nil
	case exchange.CryptocurrencyWithdrawalFee:
	case exchange.CryptocurrencyDepositFee:
	case exchange.InternationalBankDepositFee:
	}
	return 0, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
func (me *MEXC) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := me.UpdateAccountInfo(ctx, assetType)
	return me.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (me *MEXC) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	pair, err = me.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	req, err := me.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		result, err := me.GetCandlestick(ctx, pair.String(), intervalString, start, end, 0)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(result))
		for c := range result {
			timeSeries[c] = kline.Candle{
				Time:   result[c].CloseTime.Time(),
				Open:   result[c].OpenPrice.Float64(),
				High:   result[c].HighPrice.Float64(),
				Low:    result[c].LowPrice.Float64(),
				Close:  result[c].ClosePrice.Float64(),
				Volume: result[c].Volume.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		result, err := me.GetContractsCandlestickData(ctx, pair.String(), req.ExchangeInterval, start, end)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(result.Data.ClosePrice))
		for i := range result.Data.ClosePrice {
			timeSeries[i] = kline.Candle{
				Open:   result.Data.ClosePrice[i],
				Time:   result.Data.Time[i].Time(),
				High:   result.Data.HighPrice[i],
				Low:    result.Data.LowPrice[i],
				Close:  result.Data.ClosePrice[i],
				Volume: result.Data.Volume[i],
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (me *MEXC) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pFormat, err := me.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	req, err := me.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		intervalString, err := intervalToString(interval)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			result, err := me.GetCandlestick(ctx,
				pFormat.Format(pair),
				intervalString,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			for c := range result {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   result[c].CloseTime.Time(),
					Open:   result[c].OpenPrice.Float64(),
					High:   result[c].HighPrice.Float64(),
					Low:    result[c].LowPrice.Float64(),
					Close:  result[c].ClosePrice.Float64(),
					Volume: result[c].Volume.Float64(),
				})
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			result, err := me.GetContractsCandlestickData(ctx, pFormat.Format(pair), req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for i := range result.Data.ClosePrice {
				timeSeries = append(timeSeries, kline.Candle{
					Open:   result.Data.ClosePrice[i],
					Time:   result.Data.Time[i].Time(),
					High:   result.Data.HighPrice[i],
					Low:    result.Data.LowPrice[i],
					Close:  result.Data.ClosePrice[i],
					Volume: result.Data.Volume[i],
				})
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (me *MEXC) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	contracts, err := me.GetFuturesContracts(ctx, "")
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts.Data))
	for a := range contracts.Data {
		cp, err := currency.NewPairFromString(contracts.Data[a].Symbol)
		if err != nil {
			return nil, err
		}
		var contractType futures.ContractType
		switch {
		case strings.HasSuffix(contracts.Data[a].DisplayNameEn, "PERPETUAL"):
			contractType = futures.Perpetual
		}
		var contractStatus string
		switch contracts.Data[a].State {
		case 0:
			contractStatus = "enabled"
		case 1:
			contractStatus = "delivery"
		case 2:
			contractStatus = "completed"
		case 3:
			contractStatus = "offline"
		case 4:
			contractStatus = "pause"
		}
		resp[a] = futures.Contract{
			Exchange:             me.Name,
			Name:                 cp,
			Asset:                item,
			SettlementCurrencies: []currency.Code{currency.NewCode(contracts.Data[a].SettleCoin)},
			Type:                 contractType,
			MaxLeverage:          contracts.Data[a].MaxLeverage,
			IsActive:             contracts.Data[a].State == 0,
			Status:               contractStatus,
			Multiplier:           contracts.Data[a].MinVol,
		}
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
// differs by exchange
func (me *MEXC) IsPerpetualFutureCurrency(assetType asset.Item, pair currency.Pair) (bool, error) {
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	if assetType != asset.Futures {
		return false, futures.ErrNotFuturesAsset
	}
	result, err := me.GetFuturesContracts(context.Background(), pair.String())
	if err != nil {
		return false, err
	}
	return strings.HasSuffix(result.Data[0].DisplayNameEn, "PERPETUAL"), nil
}

// GetLatestFundingRates returns the latest funding rates data
func (me *MEXC) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if !me.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%s %w", r.Asset, asset.ErrNotSupported)
	}
	isPerpetual, err := me.IsPerpetualFutureCurrency(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	if !isPerpetual {
		return nil, fmt.Errorf("%w '%s'", futures.ErrNotPerpetualFuture, r.Pair)
	}
	pFmt, err := me.CurrencyPairs.GetFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	cp := r.Pair.Format(pFmt)
	fundingRates, err := me.GetContractFundingPrice(ctx, cp.String())
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, 1)
	resp[0] = fundingrate.LatestRateResponse{
		Exchange: me.Name,
		Asset:    asset.Futures,
		Pair:     cp,
		LatestRate: fundingrate.Rate{
			Rate: decimal.NewFromFloat(fundingRates.Data.FundingRate),
			Time: fundingRates.Data.Timestamp.Time(),
		},
		TimeOfNextRate: fundingRates.Data.NextSettleTime.Time(),
		TimeChecked:    time.Now(),
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (me *MEXC) UpdateOrderExecutionLimits(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		result, err := me.GetSymbols(ctx, nil)
		if err != nil {
			return err
		}
		limits := make([]order.MinMaxLevel, len(result.Symbols))
		for a := range result.Symbols {
			pair, err := currency.NewPairFromString(result.Symbols[a].Symbol)
			if err != nil {
				return err
			}
			limits[a] = order.MinMaxLevel{
				Pair:                   pair,
				Asset:                  assetType,
				PriceStepIncrementSize: result.Symbols[a].QuoteAmountPrecision.Float64(),
				QuoteStepIncrementSize: result.Symbols[a].QuoteAmountPrecision.Float64(),
				MaximumQuoteAmount:     result.Symbols[a].MaxQuoteAmount.Float64(),
				MinimumBaseAmount:      result.Symbols[a].BaseSizePrecision.Float64(),
			}
		}
		err = me.LoadLimits(limits)
		if err != nil {
			return err
		}
	case asset.Futures:
		result, err := me.GetFuturesContracts(ctx, "")
		if err != nil {
			return err
		}
		limits := make([]order.MinMaxLevel, len(result.Data))
		for a := range limits {
			pair, err := currency.NewPairFromString(result.Data[a].Symbol)
			if err != nil {
				return err
			}
			limits[a] = order.MinMaxLevel{
				Pair:                   pair,
				Asset:                  assetType,
				PriceStepIncrementSize: result.Data[a].PriceScale,
				MinimumBaseAmount:      result.Data[a].MinVol,
				MaxTotalOrders: func() int64 {
					if len(result.Data[a].MaxNumOrders) > 0 {
						return result.Data[a].MaxNumOrders[0]
					}
					return 0
				}(),
				MarketMaxQty: result.Data[a].MaxVol,
			}
		}
		err = me.LoadLimits(limits)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}
