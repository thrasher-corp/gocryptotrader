package poloniex

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var assetPairStores = map[asset.Item]currency.PairStore{
	asset.Futures: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
	asset.Spot: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	for a, ps := range assetPairStores {
		if err := p.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing `%s` default asset formats: %s", p.Name, a, err)
		}
	}

	p.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 500,
			},
		},
	}
	var err error
	p.Requester, err = request.New(p.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.API.Endpoints = p.NewEndpoints()
	err = p.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      poloniexAPIURL,
		exchange.WebsocketSpot: poloniexWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.Websocket = stream.NewWebsocket()
	p.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	p.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	p.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}
	err = p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := p.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = p.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            poloniexWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             p.WsConnect,
		Subscriber:            p.Subscribe,
		Unsubscriber:          p.Unsubscribe,
		GenerateSubscriptions: p.generateSubscriptions,
		Features:              &p.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}
	err = p.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  poloniexWebsocketAddress,
		RateLimit:            request.NewWeightedRateLimitByDuration(500 * time.Millisecond),
	})
	if err != nil {
		return err
	}
	return p.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  poloniexPrivateWebsocketAddress,
		RateLimit:            request.NewWeightedRateLimitByDuration(500 * time.Millisecond),
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Spot, asset.Margin:
		resp, err := p.GetSymbolInformation(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}

		pairs := make([]currency.Pair, 0, len(resp))
		for x := range resp {
			if strings.EqualFold(resp[x].State, "PAUSE") {
				continue
			}
			var pair currency.Pair
			pair, err = currency.NewPairFromString(resp[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
		return pairs, nil
	case asset.Futures:
		instruments, err := p.GetV3FuturesAllProductInfo(ctx, "")
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(instruments))
		var cp currency.Pair
		for i := range instruments {
			if !strings.EqualFold(instruments[i].Status, "Open") {
				continue
			}
			println("instruments[i].Symbol: ", instruments[i].Symbol)
			cp, err = currency.NewPairFromString(instruments[i].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(ctx context.Context, forceUpgrade bool) error {
	enabledAssets := p.GetAssetTypes(true)
	for _, assetType := range enabledAssets {
		pairs, err := p.FetchTradablePairs(ctx, assetType)
		if err != nil {
			return err
		}
		err = p.UpdatePairs(pairs, assetType, false, forceUpgrade)
		if err != nil {
			return err
		}
	}
	return p.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (p *Poloniex) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	switch assetType {
	case asset.Spot:
		ticks, err := p.GetTickers(ctx)
		if err != nil {
			return err
		}
		for i := range ticks {
			pair, err := currency.NewPairFromString(ticks[i].Symbol)
			if err != nil {
				return err
			}
			if !enabledPairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				AssetType:    assetType,
				Pair:         pair,
				ExchangeName: p.Name,
				Last:         ticks[i].MarkPrice.Float64(),
				Low:          ticks[i].Low.Float64(),
				Ask:          ticks[i].Ask.Float64(),
				Bid:          ticks[i].Bid.Float64(),
				High:         ticks[i].High.Float64(),
				QuoteVolume:  ticks[i].Amount.Float64(),
				Volume:       ticks[i].Quantity.Float64(),
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		ticks, err := p.GetV3FuturesMarketInfo(context.Background(), "")
		if err != nil {
			return err
		}
		for i := range ticks {
			pair, err := currency.NewPairDelimiter(ticks[i].Symbol, currency.UnderscoreDelimiter)
			if err != nil {
				return err
			}
			if !enabledPairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				AssetType:    assetType,
				Pair:         pair,
				ExchangeName: p.Name,
				LastUpdated:  ticks[i].EndTime.Time(),
				Volume:       ticks[i].Quantity.Float64(),
				BidSize:      ticks[i].BestBidSize.Float64(),
				Bid:          ticks[i].BestBidPrice.Float64(),
				AskSize:      ticks[i].BestAskSize.Float64(),
				Ask:          ticks[i].BestAskPrice.Float64(),
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(ctx context.Context, currencyPair currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := p.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(p.Name, currencyPair, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	err := p.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        p.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: p.CanVerifyOrderbook,
	}
	switch assetType {
	case asset.Spot:
		var orderbookNew *OrderbookData
		orderbookNew, err = p.GetOrderbook(ctx, pair, 0, 0)
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids)/2)
		for y := range book.Bids {
			book.Bids[y].Price = orderbookNew.Bids[y*2].Float64()
			book.Bids[y].Amount = orderbookNew.Bids[y*2+1].Float64()
		}
		book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks)/2)
		for y := range book.Asks {
			book.Asks[y].Price = orderbookNew.Asks[y*2].Float64()
			book.Asks[y].Amount = orderbookNew.Asks[y*2+1].Float64()
		}
	case asset.Futures:
		var orderbookNew *FuturesV3Orderbook
		orderbookNew, err = p.GetV3FuturesOrderBook(ctx, pair.String(), 0, 0)
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids))
		for y := range book.Bids {
			book.Bids[y].Price = orderbookNew.Bids[y][0].Float64()
			book.Bids[y].Amount = orderbookNew.Bids[y][1].Float64()
		}
		book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks))
		for y := range book.Asks {
			book.Asks[y].Price = orderbookNew.Asks[y][0].Float64()
			book.Asks[y].Amount = orderbookNew.Asks[y][1].Float64()
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(p.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	switch assetType {
	case asset.Spot, asset.Futures:
		accountBalance, err := p.GetSubAccountBalances(ctx)
		if err != nil {
			return response, err
		}
		subAccounts := make([]account.SubAccount, len(accountBalance))
		for i := range accountBalance {
			subAccount := account.SubAccount{
				ID:        accountBalance[i].AccountID,
				AssetType: stringToAccountType(accountBalance[i].AccountType),
			}
			currencyBalances := make([]account.Balance, len(accountBalance[i].Balances))
			for x := range accountBalance[i].Balances {
				currencyBalances[x] = account.Balance{
					Currency:               currency.NewCode(accountBalance[i].Balances[x].Currency),
					Total:                  accountBalance[i].Balances[x].AvailableBalance.Float64(),
					Hold:                   accountBalance[i].Balances[x].Hold.Float64(),
					Free:                   accountBalance[i].Balances[x].Available.Float64(),
					AvailableWithoutBorrow: accountBalance[i].Balances[x].AvailableBalance.Float64(),
				}
			}
			subAccounts[i] = subAccount
		}
		response = account.Holdings{
			Exchange: p.Name,
			Accounts: subAccounts,
		}
		creds, err := p.GetCredentials(ctx)
		if err != nil {
			return response, err
		}
		return response, account.Process(&response, creds)
	default:
		return response, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	end := time.Now()
	walletActivity, err := p.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(walletActivity.Deposits))
	for i := range walletActivity.Deposits {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    p.Name,
			Status:          walletActivity.Deposits[i].Status,
			Timestamp:       walletActivity.Deposits[i].Timestamp.Time(),
			Currency:        walletActivity.Deposits[i].Currency,
			Amount:          walletActivity.Deposits[i].Amount.Float64(),
			CryptoToAddress: walletActivity.Deposits[i].Address,
			CryptoTxID:      walletActivity.Deposits[i].TransactionID,
			TransferType:    "deposit",
		}
	}
	for i := range walletActivity.Withdrawals {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    p.Name,
			Status:          walletActivity.Withdrawals[i].Status,
			Timestamp:       walletActivity.Withdrawals[i].Timestamp.Time(),
			Currency:        walletActivity.Withdrawals[i].Currency,
			Amount:          walletActivity.Withdrawals[i].Amount.Float64(),
			Fee:             walletActivity.Withdrawals[i].Fee.Float64(),
			CryptoToAddress: walletActivity.Withdrawals[i].Address,
			CryptoTxID:      walletActivity.Withdrawals[i].TransactionID,
			TransferType:    "withdrawals",
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (p *Poloniex) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	end := time.Now()
	withdrawals, err := p.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "withdrawals")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals.Withdrawals))
	for i := range withdrawals.Withdrawals {
		if !c.Equal(currency.NewCode(withdrawals.Withdrawals[i].Currency)) {
			continue
		}
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals.Withdrawals[i].Status,
			Timestamp:       withdrawals.Withdrawals[i].Timestamp.Time(),
			Currency:        withdrawals.Withdrawals[i].Currency,
			Amount:          withdrawals.Withdrawals[i].Amount.Float64(),
			Fee:             withdrawals.Withdrawals[i].Fee.Float64(),
			CryptoToAddress: withdrawals.Withdrawals[i].Address,
			CryptoTxID:      withdrawals.Withdrawals[i].TransactionID,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (p *Poloniex) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		var tradeData []Trade
		tradeData, err = p.GetTrades(ctx, pair, 0)
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[i].TakerSide)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].Timestamp.Time(),
			})
		}
	case asset.Futures:
		var tradeData []V3FuturesExecutionInfo
		tradeData, err = p.GetV3FuturesExecutionInfo(ctx, pair.String(), 0)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].CreationTime.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	err = p.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (p *Poloniex) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		var tradeData []Trade
		tradeData, err = p.GetTrades(ctx, pair, 1000)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			if tradeData[i].CreateTime.Time().After(timestampEnd) ||
				tradeData[i].CreateTime.Time().Before(timestampStart) {
				continue
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].TakerSide)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].CreateTime.Time(),
			})
		}
	case asset.Futures:
		var tradeData []V3FuturesExecutionInfo
		tradeData, err = p.GetV3FuturesExecutionInfo(ctx, pair.String(), 0)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].CreationTime.Time(),
			})
		}
	}
	if err := p.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s == nil || *s == (order.Submit{}) {
		return nil, common.ErrEmptyParams
	}
	var err error
	s.Pair, err = p.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	oTypeString, err := OrderTypeString(s.Type)
	if err != nil {
		return nil, err
	}
	var tif string
	tif, err = TimeInForceString(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		var smartOrder bool
		var response *PlaceOrderResponse
		switch s.Type {
		case order.Stop, order.StopLimit, order.TrailingStop:
			smartOrder = true
		case order.Limit, order.Market, order.LimitMaker, order.UnknownType:
		default:
			return nil, fmt.Errorf("%v order type %v is not supported", order.ErrTypeIsInvalid, s.Type)
		}
		if smartOrder {
			var sOrder *PlaceOrderResponse
			sOrder, err = p.CreateSmartOrder(ctx, &SmartOrderRequestParam{
				Symbol:        s.Pair,
				Type:          oTypeString,
				Side:          s.Side.String(),
				AccountType:   accountTypeString(s.AssetType),
				Price:         s.Price,
				StopPrice:     s.TriggerPrice,
				Quantity:      s.Amount,
				ClientOrderID: s.ClientOrderID,
				TimeInForce:   tif,
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(sOrder.ID)
		}
		tif, err = TimeInForceString(s.TimeInForce)
		if err != nil {
			return nil, err
		}
		arg := &PlaceOrderParams{
			Symbol:        s.Pair,
			Price:         s.Price,
			Amount:        s.Amount,
			AllowBorrow:   false,
			Type:          oTypeString,
			Side:          s.Side.String(),
			TimeInForce:   tif,
			ClientOrderID: s.ClientOrderID,
		}
		if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			response, err = p.WsCreateOrder(arg)
		} else {
			response, err = p.PlaceOrder(ctx, arg)
		}
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(response.ID)
	case asset.Futures:
		var side string
		var positionSide string
		switch s.Side {
		case order.Sell:
			side = "SELL"
		case order.Buy:
			side = "BUY"
		case order.Long:
			side = "BUY"
			positionSide = "LONG"
		case order.Short:
			side = "SELL"
			positionSide = "SHORT"
		}
		var marginMode string
		switch s.MarginType {
		case margin.Multi:
			marginMode = "CROSS"
		case margin.Isolated:
			marginMode = "ISOLATED"
		}
		var stpMode string
		switch s.TimeInForce {
		case order.PostOnly:
			stpMode = "EXPIRE_MAKER"
		case order.GoodTillCancel:
			stpMode = "EXPIRE_TAKER"
		default:
			stpMode = ""
		}
		response, err := p.PlaceV3FuturesOrder(ctx, &FuturesParams{
			ClientOrderID:           s.ClientOrderID,
			Side:                    side,
			PositionSide:            positionSide,
			Symbol:                  s.Pair.String(),
			OrderType:               oTypeString,
			ReduceOnly:              s.ReduceOnly,
			TimeInForce:             tif,
			Price:                   s.Price,
			Size:                    s.Amount,
			MarginMode:              marginMode,
			SelfTradePreventionMode: stpMode,
		})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(response.OrderID)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if action == nil {
		return nil, common.ErrNilPointer
	}
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, action.AssetType)
	}
	tif, err := TimeInForceString(action.TimeInForce)
	if err != nil {
		return nil, err
	}
	switch action.Type {
	case order.Market, order.Limit, order.LimitMaker:
		resp, err := p.CancelReplaceOrder(ctx, &CancelReplaceOrderParam{
			orderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			Price:         action.Price,
			Quantity:      action.Amount,
			AmendedType:   action.Type.String(),
			TimeInForce:   tif,
		})
		if err != nil {
			return nil, err
		}
		modResp, err := action.DeriveModifyResponse()
		if err != nil {
			return nil, err
		}
		modResp.OrderID = resp.ID
		return modResp, nil
	case order.Stop, order.StopLimit:
		oTypeString, err := OrderTypeString(action.Type)
		if err != nil {
			return nil, err
		}
		oResp, err := p.CancelReplaceSmartOrder(ctx, &CancelReplaceSmartOrderParam{
			orderID:          action.OrderID,
			ClientOrderID:    action.ClientOrderID,
			Price:            action.Price,
			StopPrice:        action.TriggerPrice,
			Amount:           action.Amount,
			AmendedType:      oTypeString,
			ProceedOnFailure: !action.TimeInForce.Is(order.ImmediateOrCancel),
			TimeInForce:      tif,
		})
		if err != nil {
			return nil, err
		}
		modResp, err := action.DeriveModifyResponse()
		if err != nil {
			return nil, err
		}
		modResp.OrderID = oResp.ID
		return modResp, nil
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, action.Type)
	}
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	var err error
	switch o.AssetType {
	case asset.Spot:
		switch o.Type {
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			_, err = p.CancelSmartOrderByID(ctx, o.OrderID, o.ClientOrderID)
		default:
			_, err = p.CancelOrderByID(ctx, o.OrderID)
		}
	case asset.Futures:
		_, err = p.CancelV3FuturesOrder(ctx, &CancelOrderParams{Symbol: o.Pair.String(), OrderID: o.OrderID, ClientOrderID: o.ClientOrderID})
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (p *Poloniex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	orderIDs := make([]string, 0, len(o))
	clientOrderIDs := make([]string, 0, len(o))
	assetType := o[0].AssetType
	pairString := o[0].Pair.String()
	commonOrderType := o[0].Type
	for i := range o {
		switch o[i].AssetType {
		case asset.Spot, asset.Futures:
		default:
			return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
		}
		if assetType != o[i].AssetType {
			return nil, errOrderAssetTypeMismatch
		}
		if !slices.Contains([]order.Type{order.Market, order.Limit, order.AnyType, order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit, order.UnknownType}, o[i].Type) {
			return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, commonOrderType.String())
		}
		if commonOrderType != o[i].Type {
			commonOrderType = order.AnyType
		}
		switch {
		case o[i].ClientOrderID != "":
			clientOrderIDs = append(clientOrderIDs, o[i].ClientOrderID)
		case o[i].OrderID != "":
			orderIDs = append(orderIDs, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
		if assetType == asset.Futures {
			if o[i].Pair.String() == "" {
				return nil, currency.ErrSymbolStringEmpty
			} else if pairString != o[i].Pair.String() {
				return nil, errPairStringMismatch
			}
		}
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	if assetType == asset.Spot {
		switch commonOrderType {
		case order.Market, order.Limit, order.AnyType, order.UnknownType:
			if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				wsCancelledOrders, err := p.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: orderIDs, ClientOrderIDs: clientOrderIDs})
				if err != nil {
					return nil, err
				}
				for i := range wsCancelledOrders {
					if wsCancelledOrders[i].ClientOrderID != "" {
						resp.Status[wsCancelledOrders[i].ClientOrderID] = wsCancelledOrders[i].State + " " + wsCancelledOrders[i].Message
						continue
					}
					orderID := strconv.FormatInt(wsCancelledOrders[i].OrderID, 10)
					resp.Status[orderID] = wsCancelledOrders[i].State + " " + wsCancelledOrders[i].Message
				}
			} else {
				cancelledOrders, err := p.CancelMultipleOrdersByIDs(ctx, &OrderCancellationParams{OrderIDs: orderIDs, ClientOrderIDs: clientOrderIDs})
				if err != nil {
					return nil, err
				}
				for i := range cancelledOrders {
					if cancelledOrders[i].ClientOrderID != "" {
						resp.Status[cancelledOrders[i].ClientOrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
						continue
					}
					resp.Status[cancelledOrders[i].OrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
				}
			}
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			cancelledOrders, err := p.CancelMultipleSmartOrders(ctx, &OrderCancellationParams{
				OrderIDs:       orderIDs,
				ClientOrderIDs: clientOrderIDs,
			})
			if err != nil {
				return nil, err
			}
			for i := range cancelledOrders {
				if cancelledOrders[i].ClientOrderID != "" {
					resp.Status[cancelledOrders[i].ClientOrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
					continue
				}
				resp.Status[cancelledOrders[i].OrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
			}
		default:
			return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, commonOrderType.String())
		}
	} else {
		cancelledOrders, err := p.CancelMultipleV3FuturesOrders(ctx, &CancelOrdersParams{
			Symbol:         pairString,
			OrderIDs:       orderIDs,
			ClientOrderIDs: clientOrderIDs,
		})
		if err != nil {
			return nil, err
		}
		resp.Status = map[string]string{}
		for x := range cancelledOrders {
			if cancelledOrders[x].Code == 200 {
				resp.Status[cancelledOrders[x].OrderID] = "Cancelled"
			}
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(ctx context.Context, cancelOrd *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	if cancelOrd == nil {
		return cancelAllOrdersResponse, common.ErrNilPointer
	}
	var err error
	var pairs currency.Pairs
	if !cancelOrd.Pair.IsEmpty() {
		pairs = append(pairs, cancelOrd.Pair)
	}
	var resp []CancelOrderResponse
	switch cancelOrd.AssetType {
	case asset.Spot:
		switch cancelOrd.Type {
		case order.TrailingStop, order.TrailingStopLimit, order.StopLimit, order.Stop:
			pairsString := []string{}
			if !cancelOrd.Pair.IsEmpty() {
				pairsString = append(pairsString, cancelOrd.Pair.String())
			}
			orderTypes := []string{}
			oTypeString, err := OrderTypeString(order.StopLimit)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if cancelOrd.Type != order.UnknownType {
				orderTypes = append(orderTypes, oTypeString)
			}
			resp, err = p.CancelAllSmartOrders(ctx, pairsString, nil, orderTypes)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for x := range resp {
				cancelAllOrdersResponse.Status[resp[x].OrderID] = resp[x].State
			}
		default:
			if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				var wsResponse []WsCancelOrderResponse
				wsResponse, err = p.WsCancelAllTradeOrders(pairs.Strings(), []string{accountTypeString(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for x := range wsResponse {
					cancelAllOrdersResponse.Status[strconv.FormatInt(wsResponse[x].OrderID, 10)] = wsResponse[x].State
				}
			} else {
				resp, err = p.CancelAllTradeOrders(ctx, pairs.Strings(), []string{accountTypeString(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for x := range resp {
					cancelAllOrdersResponse.Status[resp[x].OrderID] = resp[x].State
				}
			}
		}
	case asset.Futures:
		var result []FuturesV3OrderIDResponse
		result, err = p.CancelAllV3FuturesOrders(ctx, cancelOrd.Pair.String(), cancelOrd.Side.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for x := range result {
			if result[x].Code == 200 {
				cancelAllOrdersResponse.Status[result[x].OrderID] = "cancelled"
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, cancelOrd.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (p *Poloniex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		trades, err := p.GetTradesByOrderID(ctx, orderID)
		if err != nil && !strings.Contains(err.Error(), "Order not found") {
			return nil, err
		}
		orderTrades := make([]order.TradeHistory, len(trades))
		var oType order.Type
		var oSide order.Side
		for i := range trades {
			oType, err = order.StringToOrderType(trades[i].Type)
			if err != nil {
				return nil, err
			}
			oSide, err = order.StringToOrderSide(trades[i].Side)
			if err != nil {
				return nil, err
			}
			orderTrades[i] = order.TradeHistory{
				Price:     trades[i].Price.Float64(),
				Amount:    trades[i].Quantity.Float64(),
				Fee:       trades[i].FeeAmount.Float64(),
				Exchange:  p.Name,
				TID:       trades[i].ID,
				Type:      oType,
				Side:      oSide,
				Timestamp: trades[i].CreateTime.Time(),
				FeeAsset:  trades[i].FeeCurrency,
				Total:     trades[i].Amount.Float64(),
			}
		}
		var smartOrders []SmartOrderDetail
		resp, err := p.GetOrderDetail(ctx, orderID, "")
		if err != nil {
			smartOrders, err = p.GetSmartOrderDetail(ctx, orderID, "")
			if err != nil {
				return nil, err
			} else if len(smartOrders) == 0 {
				return nil, order.ErrOrderNotFound
			}
			var dPair currency.Pair
			var oStatus order.Status
			if len(smartOrders) > 0 {
				dPair, err = currency.NewPairFromString(smartOrders[0].Symbol)
				if err != nil {
					return nil, err
				} else if !pair.IsEmpty() && !dPair.Equal(pair) {
					return nil, fmt.Errorf("order with ID %s expected a symbol %v, but got %v", orderID, pair, dPair)
				}
				oStatus, err = order.StringToOrderStatus(smartOrders[0].State)
				if err != nil {
					return nil, err
				}
				oSide, err = order.StringToOrderSide(smartOrders[0].Side)
				if err != nil {
					return nil, err
				}
				return &order.Detail{
					Price:         smartOrders[0].Price.Float64(),
					Amount:        smartOrders[0].Quantity.Float64(),
					QuoteAmount:   smartOrders[0].Amount.Float64(),
					Exchange:      p.Name,
					OrderID:       smartOrders[0].ID,
					ClientOrderID: smartOrders[0].ClientOrderID,
					Type:          StringToOrderType(smartOrders[0].Type),
					Side:          oSide,
					Status:        oStatus,
					AssetType:     stringToAccountType(smartOrders[0].AccountType),
					Date:          smartOrders[0].CreateTime.Time(),
					LastUpdated:   smartOrders[0].UpdateTime.Time(),
					Pair:          dPair,
					Trades:        orderTrades,
					TimeInForce:   smartOrders[0].TimeInForce,
				}, nil
			}
		}
		oType, err = order.StringToOrderType(resp.Type)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(resp.State)
		if err != nil {
			return nil, err
		}
		oSide, err = order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:                resp.Price.Float64(),
			Amount:               resp.Quantity.Float64(),
			AverageExecutedPrice: resp.AvgPrice.Float64(),
			QuoteAmount:          resp.Amount.Float64(),
			ExecutedAmount:       resp.FilledQuantity.Float64(),
			RemainingAmount:      resp.Quantity.Float64() - resp.FilledAmount.Float64(),
			Cost:                 resp.FilledQuantity.Float64() * resp.AvgPrice.Float64(),
			Exchange:             p.Name,
			OrderID:              resp.ID,
			ClientOrderID:        resp.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            stringToAccountType(resp.AccountType),
			Date:                 resp.CreateTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			Pair:                 pair,
			Trades:               orderTrades,
			TimeInForce:          resp.TimeInForce,
		}, nil
	case asset.Futures:
		fResults, err := p.GetV3FuturesOrderHistory(ctx, "", "", "", "", orderID, "", "", time.Time{}, time.Time{}, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(fResults) != 1 {
			return nil, order.ErrOrderNotFound
		}
		orderDetail := &fResults[0]
		dPair, err := currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return nil, err
		} else if !pair.IsEmpty() && !dPair.Equal(pair) {
			return nil, fmt.Errorf("order with ID %s expected a symbol %v, but got %v", orderID, pair, dPair)
		}
		oType, err := order.StringToOrderType(orderDetail.OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(orderDetail.State)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:                orderDetail.Price.Float64(),
			Amount:               orderDetail.Quantity.Float64(),
			AverageExecutedPrice: orderDetail.AveragePrice.Float64(),
			QuoteAmount:          orderDetail.AveragePrice.Float64() * orderDetail.ExecQuantity.Float64(),
			ExecutedAmount:       orderDetail.ExecQuantity.Float64(),
			RemainingAmount:      orderDetail.Quantity.Float64() - orderDetail.ExecQuantity.Float64(),
			OrderID:              orderDetail.OrderID,
			Exchange:             p.Name,
			ClientOrderID:        orderDetail.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 orderDetail.CreationTime.Time(),
			LastUpdated:          orderDetail.UpdateTime.Time(),
			Pair:                 dPair,
			TimeInForce:          orderDetail.TimeInForce,
		}, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	depositAddrs, err := p.GetDepositAddresses(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	// Some coins use a main address, so we must use this in conjunction with the returned
	// deposit address to produce the full deposit address and payment-id
	currencies, err := p.GetCurrencyInformation(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	coinParams, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
	}

	var address, paymentID string
	if coinParams.Type == "address-payment-id" && coinParams.DepositAddress != "" {
		paymentID, ok = (*depositAddrs)[cryptocurrency.Upper().String()]
		if !ok {
			newAddr, err := p.NewCurrencyDepositAddress(ctx, cryptocurrency)
			if err != nil {
				return nil, err
			}
			paymentID = newAddr
		}
		return &deposit.Address{
			Address: coinParams.DepositAddress,
			Tag:     paymentID,
			Chain:   coinParams.ParentChain,
		}, nil
	}

	address, ok = (*depositAddrs)[cryptocurrency.Upper().String()]
	if !ok {
		if len(coinParams.ChildChains) > 1 && chain != "" && !slices.Contains(coinParams.ChildChains, chain) {
			return nil, fmt.Errorf("currency %s has %v chains available, one of these must be specified",
				cryptocurrency,
				coinParams.ChildChains)
		}

		coinParams, ok = currencies[cryptocurrency.Upper().String()]
		if !ok {
			return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
		}
		if coinParams.WalletDepositState != "ENABLED" {
			return nil, fmt.Errorf("deposits and withdrawals for %v are currently disabled", cryptocurrency.Upper().String())
		}

		newAddr, err := p.NewCurrencyDepositAddress(ctx, cryptocurrency)
		if err != nil {
			return nil, err
		}
		address = newAddr
	}
	return &deposit.Address{
		Address: address,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if withdrawRequest == nil {
		return nil, withdraw.ErrRequestCannotBeNil
	}
	if withdrawRequest.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if withdrawRequest.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	v, err := p.WithdrawCurrency(ctx, &WithdrawCurrencyParam{
		Currency: withdrawRequest.Currency.String() + withdrawRequest.Crypto.Chain,
		Address:  withdrawRequest.Crypto.Address,
		Amount:   withdrawRequest.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: p.Name,
		ID:   strconv.FormatInt(v.WithdrawRequestID, 10),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!p.AreCredentialsValid(ctx) || p.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var samplePair currency.Pair
	if len(req.Pairs) == 1 {
		samplePair = req.Pairs[0]
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		resp, err := p.GetOpenOrders(ctx, samplePair, req.Side.String(), "", req.FromOrderID, 0)
		if err != nil {
			return nil, err
		}
		for a := range resp {
			var symbol currency.Pair
			symbol, err = currency.NewPairFromString(resp[a].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && req.Pairs.Contains(symbol, true) {
				continue
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(resp[a].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(resp[a].Type)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Type:        oType,
				OrderID:     resp[a].ID,
				Side:        orderSide,
				Amount:      resp[a].Amount.Float64(),
				Date:        resp[a].CreateTime.Time(),
				Price:       resp[a].Price.Float64(),
				Pair:        symbol,
				Exchange:    p.Name,
				TimeInForce: resp[a].TimeInForce,
			})
		}
	case asset.Futures:
		oTypeString, err := OrderTypeString(req.Type)
		if err != nil {
			return nil, err
		}
		fOrders, err := p.GetV3FuturesOrderHistory(context.Background(), samplePair.String(), oTypeString, req.Side.String(), "", "", "", "", req.StartTime, req.EndTime, 0, 0)
		if err != nil {
			return nil, err
		}
		for a := range fOrders {
			var symbol currency.Pair
			symbol, err = currency.NewPairFromString(fOrders[a].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && req.Pairs.Contains(symbol, true) {
				continue
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(fOrders[a].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(fOrders[a].OrderType)
			if err != nil {
				return nil, err
			}
			var oState order.Status
			switch fOrders[a].State {
			case "NEW":
				oState = order.Active
			case "PARTIALLY_FILLED":
				oState = order.PartiallyFilled
			default:
				continue
			}
			var mType margin.Type
			switch fOrders[a].MarginMode {
			case "ISOLATED":
				mType = margin.Isolated
			case "CROSS":
				mType = margin.Multi
			}
			orders = append(orders, order.Detail{
				Type:            oType,
				OrderID:         fOrders[a].OrderID,
				Side:            orderSide,
				Amount:          fOrders[a].Size.Float64(),
				Date:            fOrders[a].CreationTime.Time(),
				Price:           fOrders[a].Price.Float64(),
				Pair:            symbol,
				Exchange:        p.Name,
				ReduceOnly:      fOrders[a].ReduceOnly,
				Leverage:        fOrders[a].Leverage.Float64(),
				ExecutedAmount:  fOrders[a].ExecQuantity.Float64(),
				RemainingAmount: fOrders[a].Size.Float64() - fOrders[a].ExecQuantity.Float64(),
				ClientOrderID:   fOrders[a].ClientOrderID,
				Status:          oState,
				AssetType:       req.AssetType,
				LastUpdated:     fOrders[a].UpdateTime.Time(),
				MarginType:      mType,
				FeeAsset:        currency.NewCode(fOrders[a].FeeCurrency),
				Fee:             fOrders[a].FeeAmount.Float64(),
				TimeInForce:     fOrders[a].TimeInForce,
			})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(p.Name, orders), nil
}

func accountTypeString(assetType asset.Item) string {
	switch assetType {
	case asset.Spot:
		return "SPOT"
	case asset.Futures:
		return "FUTURE"
	default:
		return ""
	}
}

func stringToAccountType(assetType string) asset.Item {
	switch assetType {
	case "SPOT":
		return asset.Spot
	case "FUTURES":
		return asset.Futures
	default:
		return asset.Empty
	}
}

// GetOrderHistory retrieves account order information
// can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot:
		if req.Side != order.Buy && req.Side != order.Sell {
			return nil, fmt.Errorf("%w: %v", order.ErrSideIsInvalid, req.Side)
		}
		switch req.Type {
		case order.Market, order.Limit, order.UnknownType, order.AnyType:
			oTypeString, err := OrderTypeString(req.Type)
			if err != nil {
				return nil, err
			}
			resp, err := p.GetOrdersHistory(ctx, currency.EMPTYPAIR, accountTypeString(req.AssetType), oTypeString, req.Side.String(), "", "", 0, 100, req.StartTime, req.EndTime, false)
			if err != nil {
				return nil, err
			}
			var oSide order.Side
			var oType order.Type
			orders := make([]order.Detail, 0, len(resp))
			for i := range resp {
				var pair currency.Pair
				pair, err = currency.NewPairFromString(resp[i].Symbol)
				if err != nil {
					return nil, err
				}
				if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
					continue
				}
				oSide, err = order.StringToOrderSide(resp[i].Side)
				if err != nil {
					return nil, err
				}
				oType, err = order.StringToOrderType(resp[i].Type)
				if err != nil {
					return nil, err
				}
				var assetType asset.Item
				assetType, err = asset.New(resp[i].AccountType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					Side:                 oSide,
					Amount:               resp[i].Amount.Float64(),
					ExecutedAmount:       resp[i].FilledAmount.Float64(),
					Price:                resp[i].Price.Float64(),
					AverageExecutedPrice: resp[i].AvgPrice.Float64(),
					Pair:                 pair,
					Type:                 oType,
					Exchange:             p.Name,
					QuoteAmount:          resp[i].Amount.Float64() * resp[i].AvgPrice.Float64(),
					RemainingAmount:      resp[i].Quantity.Float64() - resp[i].FilledQuantity.Float64(),
					OrderID:              resp[i].ID,
					ClientOrderID:        resp[i].ClientOrderID,
					Status:               order.Filled,
					AssetType:            assetType,
					Date:                 resp[i].CreateTime.Time(),
					LastUpdated:          resp[i].UpdateTime.Time(),
					TimeInForce:          resp[i].TimeInForce,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return req.Filter(p.Name, orders), nil
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			oTypeString, err := OrderTypeString(req.Type)
			if err != nil {
				return nil, err
			}
			smartOrders, err := p.GetSmartOrderHistory(ctx, currency.EMPTYPAIR, accountTypeString(req.AssetType),
				oTypeString, req.Side.String(), "", "", 0, 100, req.StartTime, req.EndTime, false)
			if err != nil {
				return nil, err
			}
			var oSide order.Side
			var oType order.Type
			orders := make([]order.Detail, 0, len(smartOrders))
			for i := range smartOrders {
				var pair currency.Pair
				pair, err = currency.NewPairFromString(smartOrders[i].Symbol)
				if err != nil {
					return nil, err
				}
				if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
					continue
				}
				oSide, err = order.StringToOrderSide(smartOrders[i].Side)
				if err != nil {
					return nil, err
				}
				oType, err = order.StringToOrderType(smartOrders[i].Type)
				if err != nil {
					return nil, err
				}
				assetType, err := asset.New(smartOrders[i].AccountType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					Side:          oSide,
					Amount:        smartOrders[i].Amount.Float64(),
					Price:         smartOrders[i].Price.Float64(),
					TriggerPrice:  smartOrders[i].StopPrice.Float64(),
					Pair:          pair,
					Type:          oType,
					Exchange:      p.Name,
					OrderID:       smartOrders[i].ID,
					ClientOrderID: smartOrders[i].ClientOrderID,
					Status:        order.Filled,
					AssetType:     assetType,
					Date:          smartOrders[i].CreateTime.Time(),
					LastUpdated:   smartOrders[i].UpdateTime.Time(),
					TimeInForce:   smartOrders[i].TimeInForce,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return orders, nil
		default:
			return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, req.Type)
		}
	case asset.Futures:
		oTypeString, err := OrderTypeString(req.Type)
		if err != nil {
			return nil, err
		}
		orderHistory, err := p.GetV3FuturesOrderHistory(ctx, "", oTypeString, req.Side.String(), "", "", "", "", req.StartTime, req.EndTime, 0, 100)
		if err != nil {
			return nil, err
		}
		var oSide order.Side
		var oType order.Type
		orders := make([]order.Detail, 0, len(orderHistory))
		for i := range orderHistory {
			var pair currency.Pair
			pair, err = currency.NewPairFromString(orderHistory[i].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
				continue
			}
			oSide, err = order.StringToOrderSide(orderHistory[i].Side)
			if err != nil {
				return nil, err
			}
			oType, err = order.StringToOrderType(orderHistory[i].OrderType)
			if err != nil {
				return nil, err
			}
			detail := order.Detail{
				Side:            oSide,
				Amount:          orderHistory[i].Quantity.Float64(),
				ExecutedAmount:  orderHistory[i].ExecutedAmount.Float64(),
				Price:           orderHistory[i].Price.Float64(),
				Pair:            pair,
				Type:            oType,
				Exchange:        p.Name,
				RemainingAmount: orderHistory[i].Quantity.Float64() - orderHistory[i].ExecutedAmount.Float64(),
				OrderID:         orderHistory[i].OrderID,
				ClientOrderID:   orderHistory[i].ClientOrderID,
				Status:          order.Filled,
				AssetType:       asset.Futures,
				Date:            orderHistory[i].CreationTime.Time(),
				LastUpdated:     orderHistory[i].UpdateTime.Time(),
				TimeInForce:     orderHistory[i].TimeInForce,
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
		return req.Filter(p.Name, orders), nil
	default:
		return nil, fmt.Errorf("%w asset type %v", asset.ErrNotSupported, req.AssetType)
	}
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (p *Poloniex) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := p.UpdateAccountInfo(ctx, assetType)
	return p.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := p.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		resp, err := p.GetCandlesticks(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for x := range resp {
			timeSeries[x] = kline.Candle{
				Time:   resp[x].StartTime.Time(),
				Open:   resp[x].Open.Float64(),
				High:   resp[x].High.Float64(),
				Low:    resp[x].Low.Float64(),
				Close:  resp[x].Close.Float64(),
				Volume: resp[x].Quantity.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		resp, err := p.GetV3FuturesKlineData(ctx, req.RequestFormatted.String(), req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for x := range resp {
			timeSeries[x] = kline.Candle{
				Time:   resp[x].EndTime.Time(),
				Open:   resp[x].OpeningPrice.Float64(),
				High:   resp[x].HighestPrice.Float64(),
				Low:    resp[x].LowestPrice.Float64(),
				Volume: resp[x].BaseAmount.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	}

	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := p.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	switch a {
	case asset.Spot:
		for i := range req.RangeHolder.Ranges {
			resp, err := p.GetCandlesticks(ctx,
				req.RequestFormatted,
				req.ExchangeInterval,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   resp[x].StartTime.Time(),
					Open:   resp[x].Open.Float64(),
					High:   resp[x].High.Float64(),
					Low:    resp[x].Low.Float64(),
					Close:  resp[x].Close.Float64(),
					Volume: resp[x].Quantity.Float64(),
				})
			}
		}
	case asset.Futures:
		for i := range req.RangeHolder.Ranges {
			resp, err := p.GetV3FuturesKlineData(ctx,
				req.RequestFormatted.String(),
				interval,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
				500,
			)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   resp[x].EndTime.Time(),
					Open:   resp[x].OpeningPrice.Float64(),
					High:   resp[x].HighestPrice.Float64(),
					Low:    resp[x].LowestPrice.Float64(),
					Volume: resp[x].BaseAmount.Float64(),
				})
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (p *Poloniex) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencies, err := p.GetV2FuturesCurrencyInformation(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if len(currencies.NetworkList) == 0 {
		return nil, fmt.Errorf("%w for currency %v", errChainsNotFound, cryptocurrency)
	}
	chains := make([]string, len(currencies.NetworkList))
	for a := range currencies.NetworkList {
		chains[a] = currencies.NetworkList[a].Blockchain
	}
	return chains, nil
}

// GetServerTime returns the current exchange server time.
func (p *Poloniex) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	sysServerTime, err := p.GetSystemTimestamp(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return sysServerTime.ServerTime.Time(), nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (p *Poloniex) GetFuturesContractDetails(ctx context.Context, assetType asset.Item) ([]futures.Contract, error) {
	if !assetType.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if assetType != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	contracts, err := p.GetV3FuturesAllProductInfo(ctx, "")
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts))
	for i := range contracts {
		var cp currency.Pair
		cp, err = currency.NewPairFromString(contracts[i].Symbol)
		if err != nil {
			return nil, err
		}
		settleCurr := currency.NewCode(contracts[i].SettlementCurrency)
		var ct futures.ContractType
		if strings.HasSuffix(contracts[i].Symbol, "PERP") {
			ct = futures.Perpetual
		} else {
			ct = futures.Quarterly
		}
		resp[i] = futures.Contract{
			Exchange:             p.Name,
			Name:                 cp,
			SettlementCurrencies: currency.Currencies{settleCurr},
			MarginCurrency:       settleCurr,
			Asset:                assetType,
			StartDate:            contracts[i].ListingDate.Time(),
			IsActive:             contracts[i].Status == "OPEN",
			Status:               contracts[i].Status,
			MaxLeverage:          contracts[i].Leverage.Float64(),
			SettlementType:       futures.Linear,
			Type:                 ct,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (p *Poloniex) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil || *r == (fundingrate.LatestRateRequest{}) {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrEmptyParams)
	}
	var pairString string
	if !r.Pair.IsEmpty() {
		is, err := p.IsPerpetualFutureCurrency(r.Asset, r.Pair)
		if err != nil {
			return nil, err
		} else if !is {
			return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
		pairString = r.Pair.String()
	}
	contracts, err := p.GetV3FuturesHistoricalFundingRates(ctx, pairString, time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	timeChecked := time.Now()
	resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
	for i := range contracts {
		var cp currency.Pair
		cp, err = currency.NewPairFromString(contracts[i].Symbol)
		if err != nil {
			return nil, err
		}
		var isPerp bool
		isPerp, err = p.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		} else if !isPerp {
			continue
		}
		rate := fundingrate.LatestRateResponse{
			Exchange: p.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: contracts[i].FundingRateSettleTime.Time(),
				Rate: decimal.NewFromFloat(contracts[i].FundingRate.Float64()),
			},
			TimeOfNextRate: contracts[i].NextFundingTime.Time(),
			TimeChecked:    timeChecked,
		}
		if r.IncludePredictedRate {
			rate.PredictedUpcomingRate = fundingrate.Rate{
				Time: contracts[i].NextFundingTime.Time(),
				Rate: decimal.NewFromFloat(contracts[i].NextPredictedFundingRate.Float64()),
			}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (p *Poloniex) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	switch {
	case a == asset.Futures && strings.HasSuffix(cp.Quote.String(), "PERP"):
		return true, nil
	default:
		return false, nil
	}
}

// UpdateOrderExecutionLimits updates order execution limits
func (p *Poloniex) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !p.SupportsAsset(a) {
		return fmt.Errorf("%w asset: %v", asset.ErrNotSupported, a)
	}
	instruments, err := p.GetSymbolInformation(ctx, currency.EMPTYPAIR)
	if err != nil {
		return err
	}
	limits := make([]order.MinMaxLevel, len(instruments))
	for x := range instruments {
		pair, err := currency.NewPairFromString(instruments[x].Symbol)
		if err != nil {
			return err
		}
		limits[x] = order.MinMaxLevel{
			Pair:                    pair,
			Asset:                   a,
			PriceStepIncrementSize:  instruments[x].SymbolTradeLimit.PriceScale,
			MinimumBaseAmount:       instruments[x].SymbolTradeLimit.MinQuantity.Float64(),
			MinimumQuoteAmount:      instruments[x].SymbolTradeLimit.MinAmount.Float64(),
			AmountStepIncrementSize: instruments[x].SymbolTradeLimit.AmountScale,
			QuoteStepIncrementSize:  instruments[x].SymbolTradeLimit.QuantityScale,
		}
	}
	return p.LoadLimits(limits)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (p *Poloniex) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := p.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return poloniexAPIURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		cp.Delimiter = ""
		return poloniexAPIURL + tradeFutures + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// OrderTypeString return a string representation of order type
func OrderTypeString(oType order.Type) (string, error) {
	switch oType {
	case order.Market, order.Limit, order.LimitMaker:
		return oType.String(), nil
	case order.StopLimit:
		return "STOP_LIMIT", nil
	case order.AnyType, order.UnknownType:
		return "", nil
	}
	return "", fmt.Errorf("%w: order type %v", order.ErrUnsupportedOrderType, oType)
}

// StringToOrderType returns an order.Type instance from string
func StringToOrderType(oTypeString string) order.Type {
	switch oTypeString {
	case "STOP":
		return order.Stop
	case "STOP_LIMIT":
		return order.StopLimit
	case "TRAILING_STOP":
		return order.TrailingStop
	case "TRAILING_STOP_LIMIT":
		return order.TrailingStopLimit
	default:
		return order.Limit
	}
}

// TimeInForceString return a string representation of time-in-force value
func TimeInForceString(tif order.TimeInForce) (string, error) {
	if tif.Is(order.GoodTillCancel) {
		return order.GoodTillCancel.String(), nil
	}
	if tif.Is(order.FillOrKill) {
		return order.FillOrKill.String(), nil
	}
	if tif.Is(order.ImmediateOrCancel) {
		return order.ImmediateOrCancel.String(), nil
	}
	if tif == order.UnsetTIF {
		return "", nil
	}
	return "", fmt.Errorf("%w: TimeInForce value %v is not supported", order.ErrInvalidTimeInForce, tif)
}
