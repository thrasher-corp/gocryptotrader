package mexc

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Mexc
func (e *Exchange) SetDefaults() {
	e.Name = "MEXC"
	e.Enabled = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	if err := e.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: ""},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	if err := e.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	if err := e.DisableAssetWebsocketSupport(asset.Futures); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, asset.Futures, err)
	}
	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				KlineFetching:     true,
				AccountInfo:       true,
				SubmitOrder:       true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				KlineFetching:     true,
				AccountInfo:       true,
				SubmitOrder:       true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: false,
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
		Subscriptions: defaultSubscriptions.Clone(),
	}
	var err error
	e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              spotAPIURL,
		exchange.RestFutures:           contractAPIURL,
		exchange.WebsocketSpot:         spotWebsocketURL,
		exchange.WebsocketUSDTMargined: futuresWebsocketURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	if err := e.SetupDefaults(exch); err != nil {
		return err
	}
	spotWSURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	futuresWSURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDTMargined)
	if err != nil {
		return err
	}
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   spotWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      time.Second * 3,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Handler: func(_ context.Context, _ websocket.Connection, incoming []byte) error {
			return e.WsHandleData(incoming)
		},
		MessageFilter: asset.Spot,
		Authenticated: true,
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   futuresWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      time.Second * 3,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		Connector:             e.WsFuturesConnect,
		Subscriber:            e.SubscribeFutures,
		Unsubscriber:          e.UnsubscribeFutures,
		GenerateSubscriptions: e.generateFuturesSubscriptions,
		Handler: func(_ context.Context, _ websocket.Connection, incoming []byte) error {
			return e.WsHandleFuturesData(incoming)
		},
		Authenticate:  e.wsAuth,
		MessageFilter: asset.Futures,
		Authenticated: true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	pairFormat, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		result, err := e.GetSymbols(ctx, nil)
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(result.Symbols))
		for i := range result.Symbols {
			if result.Symbols[i].Status.Int64() != 1 {
				continue
			}
			pair, err := currency.NewPairFromStrings(result.Symbols[i].BaseAsset, result.Symbols[i].QuoteAsset)
			if err != nil {
				return nil, err
			}
			currencyPairs = append(currencyPairs, pair.Format(pairFormat))
		}
		return currencyPairs, nil
	case asset.Futures:
		result, err := e.GetFuturesContracts(ctx, "")
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(result.Data))
		for i := range result.Data {
			switch result.Data[i].State {
			case 2, 3, 4:
				continue
			}
			pair, err := currency.NewPairFromStrings(result.Data[i].BaseCoin, result.Data[i].QuoteCoin)
			if err != nil {
				return nil, err
			}
			currencyPairs = append(currencyPairs, pair.Format(pairFormat))
		}
		return currencyPairs, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := e.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assetTypes[x], false); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		pairString := pFormat.Format(p)
		tickers, err := e.Get24HourTickerPriceChangeStatistics(ctx, []string{pairString})
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers {
			if tickers[t].Symbol != pairString {
				continue
			}
			found = true
			if err := ticker.ProcessTicker(&ticker.Price{
				Pair:         p,
				ExchangeName: e.Name,
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
			}); err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	case asset.Futures:
		pairString := pFormat.Format(p)
		tickers, err := e.GetContractTickers(ctx, pairString)
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers.Data {
			if tickers.Data[t].Symbol != pairString {
				continue
			}
			found = true
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         p,
				ExchangeName: e.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			}); err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		tickers, err := e.Get24HourTickerPriceChangeStatistics(ctx, []string{})
		if err != nil {
			return err
		}
		for t := range tickers {
			pair, err := currency.NewPairFromString(tickers[t].Symbol)
			if err != nil {
				return err
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				ExchangeName: e.Name,
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
			}); err != nil {
				return err
			}
		}
	case asset.Futures:
		tickers, err := e.GetContractTickers(ctx, "")
		if err != nil {
			return err
		}
		for t := range tickers.Data {
			pair, err := currency.NewPairFromString(tickers.Data[t].Symbol)
			if err != nil {
				return err
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			}); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (e *Exchange) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(e.Name, p, assetType)
	if err != nil {
		return e.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (e *Exchange) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	ob, err := orderbook.Get(e.Name, pair, assetType)
	if err != nil {
		return e.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              fPair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch assetType {
	case asset.Spot:
		result, err := e.GetOrderbook(ctx, fPair.String(), 1000)
		if err != nil {
			return book, err
		}
		book.Bids = result.Bids.Levels()
		book.Asks = result.Asks.Levels()
		if err := book.Process(); err != nil {
			return book, err
		}
	case asset.Futures:
		result, err := e.GetContractOrderbook(ctx, fPair.String(), 1000)
		if err != nil {
			return nil, err
		}
		book.Bids = result.Bids.Levels()
		book.Asks = result.Asks.Levels()
		if err := book.Process(); err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (e *Exchange) UpdateAccountInfo(ctx context.Context, _ asset.Item) (account.Holdings, error) {
	resp := account.Holdings{
		Exchange: e.Name,
	}
	accountInfo, err := e.GetAccountInformation(ctx)
	if err != nil {
		return resp, err
	}

	currBalances := make([]account.Balance, len(accountInfo.Balances))
	for b := range accountInfo.Balances {
		currBalances[b] = account.Balance{
			Currency: currency.NewCode(accountInfo.Balances[b].Asset),
			Total:    accountInfo.Balances[b].Free.Float64() + accountInfo.Balances[b].Locked.Float64(),
			Hold:     accountInfo.Balances[b].Locked.Float64(),
		}
	}
	var assetType asset.Item
	switch accountInfo.AccountType {
	case "SPOT":
		assetType = asset.Spot
	case "FUTURES":
		assetType = asset.Futures
	}
	resp.Accounts = []account.SubAccount{
		{
			AssetType:  assetType,
			Currencies: currBalances,
		},
	}

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}

	err = account.Process(&resp, creds)
	if err != nil {
		return account.Holdings{}, err
	}
	return resp, nil
}

func accountStatusToString(status int64) string {
	switch status {
	case 1:
		return "SMALL"
	case 2:
		return "TIME_DELAY"
	case 3:
		return "LARGE_DELAY"
	case 4:
		return "PENDING"
	case 5:
		return "SUCCESS"
	case 6:
		return "AUDITING"
	case 7:
		return "REJECTED"
	}
	return ""
}

func withdrawalStatusToString(withdrawalStatus int64) string {
	switch withdrawalStatus {
	case 1:
		return "APPLY"
	case 2:
		return "AUDITING"
	case 3:
		return "WAIT"
	case 4:
		return "PROCESSING"
	case 5:
		return "WAIT_PACKAGING"
	case 6:
		return "WAIT_CONFIRM"
	case 7:
		return "SUCCESS"
	case 8:
		return "FAILED"
	case 9:
		return "CANCEL"
	case 10:
		return "MANUAL"
	}
	return ""
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	result, err := e.GetFundDepositHistory(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(result))
	for a := range result {
		resp[a] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          accountStatusToString(result[a].Status),
			TransferID:      result[a].TransactionID,
			Timestamp:       result[a].ConfirmTimes.Time(),
			Currency:        result[a].Coin,
			Amount:          result[a].Amount.Float64(),
			CryptoToAddress: result[a].Address,
			TransferType:    "diposit",
		}
	}
	withdrawals, err := e.GetWithdrawalHistory(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	resp = make([]exchange.FundingHistory, len(withdrawals))
	for w := range withdrawals {
		resp[w] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalStatusToString(withdrawals[w].Status),
			TransferID:      withdrawals[w].TransactionID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin,
			Amount:          withdrawals[w].Amount.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalHistory(ctx, c, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for w := range withdrawals {
		resp[w] = exchange.WithdrawalHistory{
			Status:          withdrawalStatusToString(withdrawals[w].Status),
			TransferID:      withdrawals[w].TransactionID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin,
			Amount:          withdrawals[w].Amount.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := e.GetRecentTradesList(ctx, p.String(), 0)
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
				Exchange:     e.Name,
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
		result, err := e.GetContractTransactionData(ctx, p.String(), 0)
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
				Exchange:     e.Name,
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
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Futures:
		result, err := e.GetContractTransactionData(ctx, p.String(), 0)
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
				Exchange:     e.Name,
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
		result, err := e.GetAggregatedTrades(ctx, p.String(), startTime, endTime, 0)
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
				Exchange:     e.Name,
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
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	serverTime, err := e.GetSystemTime(ctx)
	return serverTime.Time(), err
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s == nil {
		return nil, order.ErrSubmissionIsNil
	}
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		orderTypeString, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(s.Type, s.TimeInForce)
		if err != nil {
			return nil, err
		}
		result, err := e.NewOrder(ctx, s.Pair.String(), s.ClientOrderID, s.Side.String(), orderTypeString, s.Amount, s.QuoteAmount, s.Price)
		if err != nil {
			return nil, err
		}
		orderType, tif, err := e.StringToOrderTypeAndTimeInForce(result.Type)
		if err != nil {
			return nil, err
		}
		orderSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(result.Symbol)
		if err != nil {
			return nil, err
		}
		var ordStatus order.Status
		if result.Status != "" {
			ordStatus, err = order.StringToOrderStatus(result.Status)
			if err != nil {
				return nil, err
			}
		}
		return &order.SubmitResponse{
			Pair:                 cp,
			Exchange:             e.Name,
			Type:                 orderType,
			Side:                 orderSide,
			AssetType:            asset.Spot,
			Leverage:             s.Leverage,
			ReduceOnly:           s.ReduceOnly,
			AverageExecutedPrice: s.Price,
			Status:               ordStatus,
			QuoteAmount:          s.QuoteAmount,
			OrderID:              result.OrderID,
			ClientOrderID:        result.ClientOrderID,
			Price:                result.Price.Float64(),
			Amount:               result.OrigQty.Float64(),
			LastUpdated:          result.TransactTime.Time(),
			RemainingAmount:      result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			TimeInForce:          tif,
		}, nil
	case asset.Futures:
		var oTypeInt string
		switch s.Type {
		case order.Limit:
			if order.PostOnly.Is(s.TimeInForce) {
				oTypeInt = "2"
			} else {
				oTypeInt = "1"
			}
		case order.Market:
			switch s.TimeInForce {
			case order.ImmediateOrCancel:
				oTypeInt = "3"
			case order.FillOrKill:
				oTypeInt = "4"
			default:
				oTypeInt = "5"
			}
		case order.Chase,
			order.TrailingStop:
			oTypeInt = "6"
		}
		orderID, err := e.PlaceFuturesOrder(ctx, &PlaceFuturesOrderParams{
			Symbol:          s.Pair.String(),
			Price:           s.Price,
			Volume:          s.Amount,
			Leverage:        int64(s.Leverage),
			Side:            s.Side,
			OrderType:       oTypeInt,
			MarginType:      s.MarginType,
			ExternalOrderID: s.ClientOrderID,
			StopLossPrice:   s.RiskManagementModes.StopLoss.Price,
			TakeProfitPrice: s.RiskManagementModes.TakeProfit.LimitPrice,
			ReduceOnly:      s.ReduceOnly,
		})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(strconv.FormatInt(orderID, 10))
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot:
		_, err := e.CancelTradeOrder(ctx, ord.Pair.String(), ord.OrderID, ord.ClientOrderID, "")
		return err
	case asset.Futures:
		var err error
		if ord.OrderID != "" {
			_, err = e.CancelOrdersByID(ctx, ord.Pair.String(), ord.OrderID)
		} else {
			_, err = e.CancelOrderByClientOrderID(ctx, ord.Pair.String(), ord.ClientOrderID)
		}
		return err
	}
	return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, ord.AssetType)
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(orderCancellation.StandardCancel()); err != nil {
		return order.CancelAllResponse{}, err
	}
	resp := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	var err error
	switch orderCancellation.AssetType {
	case asset.Spot:
		orderCancellation.Pair, err = e.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		result, err := e.CancelAllOpenOrdersBySymbol(ctx, orderCancellation.Pair.String())
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		for r := range result {
			resp.Status[result[r].OrderID] = "cancelled"
		}
		return resp, nil
	case asset.Futures:
		var symbolString string
		if !orderCancellation.Pair.IsEmpty() {
			symbolString, err = e.FormatSymbol(orderCancellation.Pair, orderCancellation.AssetType)
			if err != nil {
				return order.CancelAllResponse{}, err
			}
		}
		result, err := e.CancelAllOpenOrders(ctx, symbolString)
		if err != nil {
			return resp, err
		}
		for r := range result {
			resp.Status[result[r].ExternalOrderID] = "cancelled"
		}
		return resp, nil
	default:
		return order.CancelAllResponse{}, asset.ErrNotSupported
	}
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	pairFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		if pair.IsEmpty() {
			return nil, currency.ErrSymbolStringEmpty
		}
		result, err := e.GetOrderByID(ctx, pairFormat.Format(pair), "", orderID)
		if err != nil {
			return nil, err
		}
		oType, tif, err := e.StringToOrderTypeAndTimeInForce(result.Type)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		var oStatus order.Status
		if result.Status != "" {
			oStatus, err = order.StringToOrderStatus(result.Status)
			if err != nil {
				return nil, err
			}
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
			Exchange:             e.Name,
			OrderID:              result.OrderID,
			ClientOrderID:        result.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Spot,
			LastUpdated:          result.TransactTime.Time(),
			Pair:                 cp,
			TimeInForce:          tif,
		}, nil
	case asset.Futures:
		result, err := e.GetOrderByOrderID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		var oType order.Type
		var tif order.TimeInForce
		switch result.OrderType {
		case 1:
			oType = order.Limit
		case 2:
			tif = order.PostOnly
			oType = order.Limit
		case 3:
			oType = order.Market
			tif = order.ImmediateOrCancel
		case 4:
			oType = order.Market
			tif = order.FillOrKill
		case 5:
			oType = order.Market
		case 6:
			oType = order.Chase
		}
		var fee float64
		switch result.OrderType {
		case 1, 2:
			fee = result.MakerFee
		case 3, 4, 5, 6:
			fee = result.TakerFee
		}
		var oSide order.Side
		switch result.Side {
		case 1, 4:
			oSide = order.Buy
		case 2, 3:
			oSide = order.Sell
		}
		var oStatus order.Status
		switch result.State {
		case 1:
			oStatus = order.Pending
		case 2:
			oStatus = order.New
		case 3:
			oStatus = order.Filled
		case 4:
			oStatus = order.Cancelled
		case 5:
			oStatus = order.Closed
		}
		var mType margin.Type
		switch result.OpenType {
		case 1:
			mType = margin.Isolated
		case 2:
			mType = margin.Multi
		}
		cp, err := currency.NewPairFromString(result.Symbol)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			TimeInForce:          tif,
			Leverage:             result.Leverage,
			Price:                result.Price,
			Amount:               result.Volume,
			TriggerPrice:         result.TakeProfitPrice,
			AverageExecutedPrice: result.DealAvgPrice,
			ExecutedAmount:       result.DealVol,
			RemainingAmount:      result.Volume - result.DealVol,
			Fee:                  fee,
			FeeAsset:             currency.NewCode(result.FeeCurrency),
			Exchange:             e.Name,
			OrderID:              strconv.FormatInt(result.OrderID, 10),
			ClientOrderID:        result.ExternalOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Futures,
			LastUpdated:          result.UpdateTime.Time(),
			Pair:                 cp,
			MarginType:           mType,
		}, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", order.ErrAssetNotSet, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, code currency.Code, _, chain string) (*deposit.Address, error) {
	result, err := e.GetDepositAddressOfCoin(ctx, code, chain)
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
func (e *Exchange) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details order.FilteredOrders
		for p := range getOrdersRequest.Pairs {
			result, err := e.GetOpenOrders(ctx, pairFormat.Format(getOrdersRequest.Pairs[p]))
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
					Exchange:             e.Name,
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
			result, err := e.GetUserCurrentPendingOrder(ctx, pairFormat.Format(getOrdersRequest.Pairs[p]), 0, 0)
			if err != nil {
				return nil, err
			}
			for od := range result.Data {
				var oType order.Type
				var tif order.TimeInForce
				switch result.Data[od].OrderType {
				case 1:
					oType = order.Limit
					tif = order.GoodTillCancel
				case 2:
					oType = order.Limit
					tif = order.PostOnly
				case 3:
					oType = order.Market
					tif = order.ImmediateOrCancel
				case 4:
					oType = order.Market
					tif = order.FillOrKill
				case 5:
					oType = order.Market
				case 6:
					oType = order.Chase
				}
				// order direction 1 open long,2 close short,3 open short, 4 close long
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
					Exchange:             e.Name,
					OrderID:              result.Data[od].ExternalOrderID,
					ClientOrderID:        result.Data[od].ExternalOrderID,
					Type:                 oType,
					Side:                 oSide,
					Status:               oStatus,
					AssetType:            asset.Futures,
					LastUpdated:          result.Data[od].UpdateTime.Time(),
					Pair:                 getOrdersRequest.Pairs[p],
					TimeInForce:          tif,
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
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		var symbolString string
		if len(getOrdersRequest.Pairs) == 1 {
			symbolString = pairFormat.Format(getOrdersRequest.Pairs[0])
		}
		result, err := e.GetAllOrders(ctx, symbolString, getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0)
		if err != nil {
			return nil, err
		}
		orderDetails := make(order.FilteredOrders, len(result))
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
			orderDetails[r] = order.Detail{
				Price:                result[r].Price.Float64(),
				Amount:               result[r].OrigQty.Float64(),
				AverageExecutedPrice: result[r].Price.Float64(),
				QuoteAmount:          result[r].CummulativeQuoteQty.Float64(),
				ExecutedAmount:       result[r].ExecutedQty.Float64(),
				RemainingAmount:      result[r].OrigQty.Float64() - result[r].ExecutedQty.Float64(),
				Exchange:             e.Name,
				OrderID:              result[r].OrderID,
				ClientOrderID:        result[r].ClientOrderID,
				Type:                 oType,
				Side:                 oSide,
				Status:               oStatus,
				AssetType:            asset.Spot,
				LastUpdated:          result[r].TransactTime.Time(),
			}
		}
		return orderDetails, nil
	case asset.Futures:
		var symbolString string
		if len(getOrdersRequest.Pairs) == 1 {
			symbolString = pairFormat.Format(getOrdersRequest.Pairs[0])
		}
		result, err := e.GetAllUserHistoricalOrders(ctx, symbolString, "", "", getOrdersRequest.Side.String(), getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
		if err != nil {
			return nil, err
		}
		pairStrings := getOrdersRequest.Pairs.Strings()
		orderDetails := make(order.FilteredOrders, len(result.Data))
		for r := range result.Data {
			if len(pairStrings) > 1 && slices.Contains(pairStrings, result.Data[r].Symbol) {
				continue
			}
			cp, err := currency.NewPairFromString(result.Data[r].Symbol)
			if err != nil {
				return nil, err
			}
			var marginType margin.Type
			switch result.Data[r].OpenType {
			case 1:
				marginType = margin.Isolated
			case 2:
				marginType = margin.Multi
			}

			var oType order.Type
			var tif order.TimeInForce
			switch result.Data[r].OrderType {
			case 1:
				oType = order.Limit
				tif = order.GoodTillCancel
			case 2:
				oType = order.Limit
				tif = order.PostOnly
			case 3:
				oType = order.Market
				tif = order.ImmediateOrCancel
			case 4:
				oType = order.Market
				tif = order.FillOrKill
			case 5:
				oType = order.Market
			case 6:
				oType = order.Chase
			}
			// order direction 1 open long,2 close short,3 open short, 4 close long
			var oSide order.Side
			switch result.Data[r].Side {
			case 1, 4:
				oSide = order.Long
			case 2, 3:
				oSide = order.Short
			}
			// TODO: fix again
			var oStatus order.Status
			switch result.Data[r].State {
			case 1:
				oStatus = order.Active
			case 2:
				oStatus = order.AutoDeleverage
			case 3:
				oStatus = order.Closed
			}
			var fee float64
			switch result.Data[r].OrderType {
			case 1, 2:
				fee = result.Data[r].MakerFee
			case 3, 4, 5, 6:
				fee = result.Data[r].TakerFee
			}
			orderDetails[r] = order.Detail{
				Exchange:             e.Name,
				TimeInForce:          tif,
				Leverage:             result.Data[r].Leverage,
				Price:                result.Data[r].Price,
				Amount:               result.Data[r].Volume,
				TriggerPrice:         result.Data[r].TakeProfitPrice,
				AverageExecutedPrice: result.Data[r].DealAvgPrice,
				ExecutedAmount:       result.Data[r].DealVol,
				RemainingAmount:      result.Data[r].Volume - result.Data[r].DealVol,
				Fee:                  fee,
				FeeAsset:             currency.NewCode(result.Data[r].FeeCurrency),
				OrderID:              strconv.FormatInt(result.Data[r].OrderID, 10),
				ClientOrderID:        result.Data[r].ExternalOrderID,
				Type:                 oType,
				Side:                 oSide,
				Status:               oStatus,
				AssetType:            asset.Futures,
				LastUpdated:          result.Data[r].UpdateTime.Time(),
				Pair:                 cp,
				MarginType:           marginType,
			}
		}
		return orderDetails, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		if feeBuilder.IsMaker {
			return 0., nil
		}
		return 0.0005, nil
	case exchange.CryptocurrencyTradeFee:
		result, err := e.GetSymbolTradingFee(ctx, feeBuilder.Pair.String())
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

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	pair, err = e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		result, err := e.GetCandlestick(ctx, pair.String(), intervalString, start, end, 0)
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
		result, err := e.GetContractsCandlestickData(ctx, pair.String(), req.ExchangeInterval, start, end)
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
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pFormat, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
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
			result, err := e.GetCandlestick(ctx,
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
			result, err := e.GetContractsCandlestickData(ctx, pFormat.Format(pair), req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
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
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	contracts, err := e.GetFuturesContracts(ctx, "")
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts.Data))
	for a := range contracts.Data {
		cp, err := e.MatchSymbolWithAvailablePairs(contracts.Data[a].Symbol, item, true)
		if err != nil {
			return nil, err
		}
		var contractType futures.ContractType
		if strings.HasSuffix(contracts.Data[a].DisplayNameEn, "PERPETUAL") {
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
			Exchange:             e.Name,
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
func (e *Exchange) IsPerpetualFutureCurrency(assetType asset.Item, pair currency.Pair) (bool, error) {
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	if assetType != asset.Futures {
		return false, futures.ErrNotFuturesAsset
	}
	result, err := e.GetFuturesContracts(context.Background(), pair.String())
	if err != nil {
		return false, err
	}
	if len(result.Data) == 0 {
		return false, common.ErrNoResults
	}
	return strings.HasSuffix(result.Data[0].DisplayNameEn, "PERPETUAL"), nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if !e.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%s %w", r.Asset, asset.ErrNotSupported)
	}
	pFmt, err := e.CurrencyPairs.GetFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	cp := r.Pair.Format(pFmt)
	fundingRates, err := e.GetContractFundingPrice(ctx, cp.String())
	if err != nil {
		return nil, err
	}
	return []fundingrate.LatestRateResponse{
		{
			Exchange: e.Name,
			Asset:    asset.Futures,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(fundingRates.Data.FundingRate),
				Time: fundingRates.Data.Timestamp.Time(),
			},
			TimeOfNextRate: fundingRates.Data.NextSettleTime.Time(),
			TimeChecked:    time.Now(),
		},
	}, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		result, err := e.GetSymbols(ctx, nil)
		if err != nil {
			return err
		}
		pairFormat, err := e.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(result.Symbols))
		for a := range result.Symbols {
			pair, err := currency.NewPairFromStrings(result.Symbols[a].BaseAsset, result.Symbols[a].QuoteAsset)
			if err != nil {
				return err
			}
			l[a] = limits.MinMaxLevel{
				Key:                    key.NewExchangeAssetPair(e.Name, assetType, pair.Format(pairFormat)),
				PriceStepIncrementSize: result.Symbols[a].QuoteAmountPrecision.Float64(),
				QuoteStepIncrementSize: result.Symbols[a].QuoteAmountPrecision.Float64(),
				MaximumQuoteAmount:     result.Symbols[a].MaxQuoteAmount.Float64(),
				MinimumBaseAmount:      result.Symbols[a].BaseSizePrecision.Float64(),
			}
		}
		if err := limits.Load(l); err != nil {
			return err
		}
	case asset.Futures:
		result, err := e.GetFuturesContracts(ctx, "")
		if err != nil {
			return err
		}
		pairFormat, err := e.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(result.Data))
		for a := range l {
			pair, err := currency.NewPairFromString(result.Data[a].Symbol)
			if err != nil {
				return err
			}
			l[a] = limits.MinMaxLevel{
				Key:                    key.NewExchangeAssetPair(e.Name, assetType, pair.Format(pairFormat)),
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
		if err := limits.Load(l); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}
