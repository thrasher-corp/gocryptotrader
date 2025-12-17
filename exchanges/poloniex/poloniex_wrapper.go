package poloniex

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errChainsNotFound = errors.New("chains not found")

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

const (
	connSpotPublic     = "spot:public"
	connSpotPrivate    = "spot:private"
	connFuturesPublic  = "futures:public"
	connFuturesPrivate = "futures:private"
)

// SetDefaults sets default settings for poloniex
func (e *Exchange) SetDefaults() {
	e.Name = "Poloniex"
	e.Enabled = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for a, ps := range assetPairStores {
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
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
		Subscriptions: defaultSubscriptions.Clone(),
	}
	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	if e.API.Endpoints == nil {
		e.API.Endpoints = e.NewEndpoints()
		if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
			exchange.RestSpot:                apiURL,
			exchange.WebsocketSpot:           websocketURL,
			exchange.WebsocketPrivate:        privateWebsocketURL,
			exchange.WebsocketFutures:        futuresWebsocketPublicURL,
			exchange.WebsocketFuturesPrivate: futuresWebsocketPrivateURL,
		}); err != nil {
			log.Errorln(log.ExchangeSys, err)
		}
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
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

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: exch,
		FillsFeed:      e.Features.Enabled.FillsFeed,
		TradeFeed:      e.Features.Enabled.TradeFeed,
		Features:       &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}
	wsSpot, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		URL:                   wsSpot,
		RateLimit:             request.NewWeightedRateLimitByDuration(2 * time.Millisecond),
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Handler:               e.wsHandleData,
		Connector:             e.wsConnect,
		MessageFilter:         connSpotPublic,
	}); err != nil {
		return err
	}
	wsSpotPrivate, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		URL:                   wsSpotPrivate,
		RateLimit:             request.NewWeightedRateLimitByDuration(2 * time.Millisecond),
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generatePrivateSubscriptions,
		Handler:               e.wsHandleData,
		Connector:             e.wsConnect,
		MessageFilter:         connSpotPrivate,
		Authenticate:          e.authenticateSpotAuthConn,
	}); err != nil {
		return err
	}
	wsFutures, err := e.API.Endpoints.GetURL(exchange.WebsocketFutures)
	if err != nil {
		return err
	}
	// Futures Public Connection
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsFutures,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		RateLimit:             request.NewWeightedRateLimitByDuration(2 * time.Millisecond),
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Handler:               e.wsFuturesHandleData,
		Subscriber:            e.SubscribeFutures,
		Unsubscriber:          e.UnsubscribeFutures,
		GenerateSubscriptions: e.generateFuturesSubscriptions,
		Connector:             e.wsConnect,
		MessageFilter:         connFuturesPublic,
	}); err != nil {
		return err
	}

	wsFuturesPrivate, err := e.API.Endpoints.GetURL(exchange.WebsocketFuturesPrivate)
	if err != nil {
		return err
	}
	// Futures Private Connection
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsFuturesPrivate,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		RateLimit:             request.NewWeightedRateLimitByDuration(2 * time.Millisecond),
		Handler:               e.wsFuturesHandleData,
		Subscriber:            e.SubscribeFutures,
		Unsubscriber:          e.UnsubscribeFutures,
		GenerateSubscriptions: e.generateFuturesPrivateSubscriptions,
		Connector:             e.wsConnect,
		MessageFilter:         connFuturesPrivate,
		Authenticate:          e.authenticateFuturesAuthConn,
		Authenticated:         true,
	})
}

// FetchTradablePairs returns a list of the exchange's tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Spot:
		resp, err := e.GetSymbols(ctx)
		if err != nil {
			return nil, err
		}

		pairs := make([]currency.Pair, 0, len(resp))
		for _, symbolDetail := range resp {
			if !strings.EqualFold(symbolDetail.State, "NORMAL") {
				continue
			}
			pairs = append(pairs, symbolDetail.Symbol)
		}
		return pairs, nil
	case asset.Futures:
		instruments, err := e.GetFuturesAllProducts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(instruments))
		for _, productInfo := range instruments {
			if !strings.EqualFold(productInfo.Status, "Open") {
				continue
			}
			pairs = append(pairs, productInfo.Symbol)
		}
		return pairs, nil
	}
	return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	for _, assetType := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, assetType)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assetType, false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		ticks, err := e.GetTickers(ctx)
		if err != nil {
			return err
		}
		for _, tick := range ticks {
			if err := ticker.ProcessTicker(&ticker.Price{
				ExchangeName: e.Name,
				AssetType:    assetType,
				Pair:         tick.Symbol,
				Last:         tick.MarkPrice.Float64(),
				Low:          tick.Low.Float64(),
				Ask:          tick.Ask.Float64(),
				Bid:          tick.Bid.Float64(),
				High:         tick.High.Float64(),
				QuoteVolume:  tick.Amount.Float64(),
				Volume:       tick.Quantity.Float64(),
			}); err != nil {
				return err
			}
		}
	case asset.Futures:
		ticks, err := e.GetFuturesMarket(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		for _, tick := range ticks {
			if err := ticker.ProcessTicker(&ticker.Price{
				ExchangeName: e.Name,
				AssetType:    assetType,
				Pair:         tick.Symbol,
				LastUpdated:  tick.EndTime.Time(),
				Volume:       tick.Quantity.Float64(),
				BidSize:      tick.BestBidSize.Float64(),
				Bid:          tick.BestBidPrice.Float64(),
				AskSize:      tick.BestAskSize.Float64(),
				Ask:          tick.BestAskPrice.Float64(),
			}); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(context.Context, currency.Pair, asset.Item) (*ticker.Price, error) {
	return nil, common.ErrFunctionNotSupported
}

func orderbookLevelFromSlice(data []types.Number) orderbook.Levels {
	obs := make(orderbook.Levels, len(data)/2)
	for i := range obs {
		obs[i].Price = data[i*2].Float64()
		obs[i].Amount = data[i*2+1].Float64()
	}
	return obs
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
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
		orderbookNew, err := e.GetOrderbook(ctx, fPair, 0, 150)
		if err != nil {
			return nil, err
		}
		book.Bids = orderbookLevelFromSlice(orderbookNew.Bids)
		book.Asks = orderbookLevelFromSlice(orderbookNew.Asks)
	case asset.Futures:
		orderbookNew, err := e.GetFuturesOrderBook(ctx, fPair, 0, 150)
		if err != nil {
			return nil, err
		}
		book.Bids = orderbookNew.Bids.Levels()
		book.Asks = orderbookNew.Asks.Levels()
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	if err := book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, fPair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	switch assetType {
	case asset.Spot, asset.Futures:
		accountBalance, err := e.GetSubAccountBalances(ctx)
		if err != nil {
			return nil, err
		}
		subAccts := accounts.SubAccounts{}
		for _, subAccountBalances := range accountBalance {
			a := accounts.NewSubAccount(stringToAccountType(subAccountBalances.AccountType), subAccountBalances.AccountID)
			for _, bal := range subAccountBalances.Balances {
				a.Balances.Set(bal.Currency, accounts.Balance{
					Currency:               bal.Currency,
					Total:                  bal.AvailableBalance.Float64(),
					Hold:                   bal.Hold.Float64(),
					Free:                   bal.Available.Float64(),
					AvailableWithoutBorrow: bal.AvailableBalance.Float64(),
				})
			}
			subAccts = subAccts.Merge(a)
		}
		return subAccts, e.Accounts.Save(ctx, subAccts, true)
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	end := time.Now()
	walletActivity, err := e.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "")
	if err != nil {
		return nil, err
	}
	depositsLen := len(walletActivity.Deposits)
	resp := make([]exchange.FundingHistory, depositsLen+len(walletActivity.Withdrawals))
	for i, walletDeposit := range walletActivity.Deposits {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          walletDeposit.Status,
			Timestamp:       walletDeposit.Timestamp.Time(),
			Amount:          walletDeposit.Amount.Float64(),
			Currency:        walletDeposit.Currency.String(),
			CryptoToAddress: walletDeposit.Address,
			CryptoTxID:      walletDeposit.TransactionID,
			TransferType:    "deposit",
		}
	}

	for i, walletWithdrawal := range walletActivity.Withdrawals {
		resp[depositsLen+i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          walletWithdrawal.Status,
			Timestamp:       walletWithdrawal.Timestamp.Time(),
			Currency:        walletWithdrawal.Currency.String(),
			Amount:          walletWithdrawal.Amount.Float64(),
			Fee:             walletWithdrawal.Fee.Float64(),
			CryptoToAddress: walletWithdrawal.Address,
			CryptoTxID:      walletWithdrawal.TransactionID,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	end := time.Now()
	withdrawals, err := e.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "withdrawals")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals.Withdrawals))
	for _, walletWithdrawal := range withdrawals.Withdrawals {
		if !c.Equal(walletWithdrawal.Currency) {
			continue
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          walletWithdrawal.Status,
			Timestamp:       walletWithdrawal.Timestamp.Time(),
			Currency:        walletWithdrawal.Currency.String(),
			Amount:          walletWithdrawal.Amount.Float64(),
			Fee:             walletWithdrawal.Fee.Float64(),
			CryptoToAddress: walletWithdrawal.Address,
			CryptoTxID:      walletWithdrawal.TransactionID,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	fPair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		tradeData, err := e.GetTrades(ctx, fPair, 0)
		if err != nil {
			return nil, err
		}
		for _, td := range tradeData {
			side, err := order.StringToOrderSide(td.TakerSide)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          td.ID,
				Exchange:     e.Name,
				CurrencyPair: fPair,
				AssetType:    assetType,
				Side:         side,
				Price:        td.Price.Float64(),
				Amount:       td.Amount.Float64(),
				Timestamp:    td.Timestamp.Time(),
			})
		}
	case asset.Futures:
		futuresExecutions, err := e.GetFuturesExecution(ctx, fPair, 0)
		if err != nil {
			return nil, err
		}
		for _, fExec := range futuresExecutions {
			side, err := order.StringToOrderSide(fExec.Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(fExec.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: fPair,
				AssetType:    assetType,
				Side:         side,
				Price:        fExec.Price.Float64(),
				Amount:       fExec.Amount.Float64(),
				Timestamp:    fExec.CreationTime.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	fPair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		tradeData, err := e.GetTrades(ctx, fPair, 1000)
		if err != nil {
			return nil, err
		}
		for _, td := range tradeData {
			if td.CreateTime.Time().After(timestampEnd) ||
				td.CreateTime.Time().Before(timestampStart) {
				continue
			}
			side, err := order.StringToOrderSide(td.TakerSide)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          td.ID,
				Exchange:     e.Name,
				CurrencyPair: fPair,
				AssetType:    assetType,
				Side:         side,
				Price:        td.Price.Float64(),
				Amount:       td.Amount.Float64(),
				Timestamp:    td.CreateTime.Time(),
			})
		}
	case asset.Futures:
		tradeData, err := e.GetFuturesExecution(ctx, fPair, 0)
		if err != nil {
			return nil, err
		}
		for _, fExecInfo := range tradeData {
			side, err := order.StringToOrderSide(fExecInfo.Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(fExecInfo.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: fPair,
				AssetType:    assetType,
				Side:         side,
				Price:        fExecInfo.Price.Float64(),
				Amount:       fExecInfo.Amount.Float64(),
				Timestamp:    fExecInfo.CreationTime.Time(),
			})
		}
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	if s.AssetType == asset.Spot {
		var stpMode string
		switch s.TimeInForce {
		case order.PostOnly:
			stpMode = "EXPIRE_MAKER"
		case order.GoodTillCancel:
			stpMode = "EXPIRE_TAKER"
		}
		switch s.Type {
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			var trackingDistance string
			if s.Type|order.TrailingStop == order.TrailingStop {
				trackingDistance = strconv.FormatFloat(s.TrackingValue, 'f', -1, 64)
				if s.TrackingMode == order.Percentage {
					trackingDistance += "%"
				}
			}

			sOrder, err := e.CreateSmartOrder(ctx, &SmartOrderRequest{
				Symbol:         s.Pair,
				Type:           OrderType(s.Type),
				Side:           s.Side,
				AccountType:    AccountType(s.AssetType),
				Price:          s.Price,
				StopPrice:      s.TriggerPrice,
				Quantity:       s.Amount,
				Amount:         s.QuoteAmount,
				ClientOrderID:  s.ClientOrderID,
				TimeInForce:    TimeInForce(s.TimeInForce),
				TrailingOffset: trackingDistance,
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(sOrder.ID)
		case order.Market, order.Limit, order.LimitMaker:
			response, err := e.PlaceOrder(ctx, &PlaceOrderRequest{
				AccountType:             AccountType(s.AssetType),
				Symbol:                  s.Pair,
				Price:                   s.Price,
				Quantity:                s.Amount,
				Amount:                  s.QuoteAmount,
				AllowBorrow:             false,
				Type:                    OrderType(s.Type),
				Side:                    s.Side,
				TimeInForce:             TimeInForce(s.TimeInForce),
				ClientOrderID:           s.ClientOrderID,
				SelfTradePreventionMode: stpMode,
				SlippageTolerance:       "0",
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(response.ID)
		default:
			return nil, fmt.Errorf("%w: %v", order.ErrUnsupportedOrderType, s.Type)
		}
	}
	side := "BUY"
	if s.Side.IsShort() {
		side = "SELL"
	}
	var stpMode string
	switch s.TimeInForce {
	case order.PostOnly:
		stpMode = "EXPIRE_MAKER"
	case order.GoodTillCancel:
		stpMode = "EXPIRE_TAKER"
	}
	switch s.Type {
	case order.Market, order.Limit, order.LimitMaker:
	default:
		return nil, fmt.Errorf("%w: %v", order.ErrUnsupportedOrderType, s.Type)
	}
	var positionSide order.Side
	if s.MarginType != margin.Unset {
		positionSide = s.Side.Position()
	}
	response, err := e.PlaceFuturesOrder(ctx, &FuturesOrderRequest{
		ClientOrderID:           s.ClientOrderID,
		Side:                    side,
		MarginMode:              MarginMode(s.MarginType),
		PositionSide:            positionSide,
		Symbol:                  s.Pair,
		OrderType:               OrderType(s.Type),
		ReduceOnly:              s.ReduceOnly,
		TimeInForce:             TimeInForce(s.TimeInForce),
		Price:                   s.Price,
		Size:                    s.Amount,
		SelfTradePreventionMode: stpMode,
	})
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(response.OrderID)
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, action.AssetType)
	}
	switch action.Type {
	case order.Market, order.Limit, order.LimitMaker:
		resp, err := e.CancelReplaceOrder(ctx, &CancelReplaceOrderRequest{
			OrderID:           action.OrderID,
			ClientOrderID:     action.ClientOrderID,
			Price:             action.Price,
			Quantity:          action.Amount,
			AmendedType:       action.Type.String(),
			TimeInForce:       TimeInForce(action.TimeInForce),
			SlippageTolerance: 0,
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
		oResp, err := e.CancelReplaceSmartOrder(ctx, &CancelReplaceSmartOrderRequest{
			OrderID:          action.OrderID,
			NewClientOrderID: action.ClientOrderID,
			Price:            action.Price,
			StopPrice:        action.TriggerPrice,
			Amount:           action.Amount,
			AmendedType:      OrderType(action.Type),
			ProceedOnFailure: !action.TimeInForce.Is(order.ImmediateOrCancel),
			TimeInForce:      TimeInForce(action.TimeInForce),
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
		return nil, fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, action.Type)
	}
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	var err error
	switch o.AssetType {
	case asset.Spot:
		switch o.Type {
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			_, err = e.CancelSmartOrderByID(ctx, o.OrderID, o.ClientOrderID)
		default:
			_, err = e.CancelOrderByID(ctx, o.OrderID, o.ClientOrderID)
		}
	case asset.Futures:
		_, err = e.CancelFuturesOrder(ctx, &CancelOrderRequest{Symbol: o.Pair, OrderID: o.OrderID, ClientOrderID: o.ClientOrderID})
	default:
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, common.ErrEmptyParams
	}
	type assetAndType struct {
		aType asset.Item
		oType order.Type
		pair  currency.Pair // only futures orders need grouping by pair
	}
	type orderAndClientSuppliedIDs struct {
		orderIDs       []string
		clientOrderIDs []string
	}
	assetAndTypeToIDsMap := make(map[assetAndType]*orderAndClientSuppliedIDs)
	for i := range o {
		var pair currency.Pair
		if o[i].AssetType == asset.Futures {
			if o[i].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			pair = o[i].Pair
		} else if o[i].AssetType != asset.Spot {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, o[i].AssetType)
		}
		keyInfo := assetAndType{o[i].AssetType, o[i].Type, pair}
		if assetAndTypeToIDsMap[keyInfo] == nil {
			assetAndTypeToIDsMap[keyInfo] = &orderAndClientSuppliedIDs{
				orderIDs:       []string{},
				clientOrderIDs: []string{},
			}
		}
		switch {
		case o[i].OrderID != "":
			assetAndTypeToIDsMap[keyInfo].orderIDs = append(assetAndTypeToIDsMap[keyInfo].orderIDs, o[i].OrderID)
		case o[i].ClientOrderID != "":
			assetAndTypeToIDsMap[keyInfo].clientOrderIDs = append(assetAndTypeToIDsMap[keyInfo].clientOrderIDs, o[i].ClientOrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for k, v := range assetAndTypeToIDsMap {
		if k.aType == asset.Spot {
			switch k.oType {
			case order.Market, order.Limit, order.AnyType, order.UnknownType:
				cancelledOrders, err := e.CancelOrdersByIDs(ctx, v.orderIDs, v.clientOrderIDs)
				if err != nil {
					return nil, err
				}
				for _, co := range cancelledOrders {
					if slices.Contains(v.orderIDs, co.OrderID) {
						resp.Status[co.OrderID] = co.State + " " + co.Message
					} else {
						resp.Status[co.ClientOrderID] = co.State + " " + co.Message
					}
				}
			case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
				cancelledOrders, err := e.CancelMultipleSmartOrders(ctx, &CancelOrdersRequest{
					OrderIDs:       v.orderIDs,
					ClientOrderIDs: v.clientOrderIDs,
				})
				if err != nil {
					return nil, err
				}
				for _, co := range cancelledOrders {
					if slices.Contains(v.orderIDs, co.OrderID) {
						resp.Status[co.OrderID] = co.State + " " + co.Message
					} else {
						resp.Status[co.ClientOrderID] = co.State + " " + co.Message
					}
				}
			default:
				return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, k.oType.String())
			}
		} else {
			cancelledOrders, err := e.CancelMultipleFuturesOrders(ctx, &CancelFuturesOrdersRequest{
				Symbol:         k.pair,
				OrderIDs:       v.orderIDs,
				ClientOrderIDs: v.clientOrderIDs,
			})
			if err != nil {
				return nil, err
			}
			for _, co := range cancelledOrders {
				cancellationStatus := order.Cancelled.String()
				if co.Code != 200 && co.Code != 0 {
					cancellationStatus = "Failed"
				}
				if slices.Contains(v.orderIDs, co.OrderID) {
					resp.Status[co.OrderID] = cancellationStatus
				} else {
					resp.Status[co.ClientOrderID] = cancellationStatus
				}
			}
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, cancelOrd *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	if cancelOrd == nil {
		return cancelAllOrdersResponse, common.ErrNilPointer
	}
	var pairs currency.Pairs
	if !cancelOrd.Pair.IsEmpty() {
		fPair, err := e.FormatExchangeCurrency(cancelOrd.Pair, cancelOrd.AssetType)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		pairs = append(pairs, fPair)
	}
	switch cancelOrd.AssetType {
	case asset.Spot:
		switch cancelOrd.Type {
		case order.TrailingStop, order.TrailingStopLimit, order.StopLimit, order.Stop:
			var orderTypes []OrderType
			if cancelOrd.Type != order.UnknownType {
				orderTypes = append(orderTypes, OrderType(cancelOrd.Type))
			}
			resp, err := e.CancelSmartOrders(ctx, pairs, nil, orderTypes)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for _, co := range resp {
				cancelAllOrdersResponse.Status[co.OrderID] = co.State
			}
		default:
			if e.Websocket.IsConnected() && e.Websocket.CanUseAuthenticatedEndpoints() && e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				wsResponse, err := e.WsCancelTradeOrders(ctx, pairs.Strings(), []AccountType{AccountType(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for _, wco := range wsResponse {
					cancelAllOrdersResponse.Status[strconv.FormatUint(wco.OrderID, 10)] = wco.State
				}
			} else {
				resp, err := e.CancelTradeOrders(ctx, pairs.Strings(), []AccountType{AccountType(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for _, co := range resp {
					if co.Code == 0 || co.Code == 200 {
						cancelAllOrdersResponse.Status[co.OrderID] = co.State
					} else {
						cancelAllOrdersResponse.Status[co.OrderID] = "Failed"
					}
				}
			}
		}
	case asset.Futures:
		result, err := e.CancelFuturesOrders(ctx, cancelOrd.Pair, cancelOrd.Side.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for _, co := range result {
			cancelAllOrdersResponse.Status[co.OrderID] = order.Cancelled.String()
			if co.Code != 200 && co.Code != 0 {
				cancelAllOrdersResponse.Status[co.OrderID] = "Failed"
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w: %q", asset.ErrNotSupported, cancelOrd.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	switch assetType {
	case asset.Spot:
		trades, err := e.GetTradesByOrderID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		orderTrades := make([]order.TradeHistory, len(trades))
		for i, td := range trades {
			oType, err := order.StringToOrderType(td.Type)
			if err != nil {
				return nil, err
			}
			orderTrades[i] = order.TradeHistory{
				Exchange:  e.Name,
				TID:       td.ID,
				FeeAsset:  td.FeeCurrency.String(),
				Price:     td.Price.Float64(),
				Total:     td.Amount.Float64(),
				Timestamp: td.CreateTime.Time(),
				Amount:    td.Quantity.Float64(),
				Fee:       td.FeeAmount.Float64(),
				Side:      td.Side,
				Type:      oType,
			}
		}
		resp, err := e.GetOrder(ctx, orderID, "")
		if err != nil {
			smartOrders, err := e.GetSmartOrderDetails(ctx, orderID, "")
			switch {
			case err != nil:
				return nil, err
			case len(smartOrders) == 0:
				return nil, order.ErrOrderNotFound
			case len(smartOrders) != 1:
				return nil, fmt.Errorf("%w: too may smart orders returned", common.ErrInvalidResponse)
			}
			s := smartOrders[0]
			oType, err := order.StringToOrderType(s.Type)
			if err != nil {
				return nil, err
			}
			return &order.Detail{
				Side:          s.Side,
				Pair:          s.Symbol,
				Exchange:      e.Name,
				Trades:        orderTrades,
				OrderID:       s.ID,
				TimeInForce:   s.TimeInForce,
				ClientOrderID: s.ClientOrderID,
				Price:         s.Price.Float64(),
				QuoteAmount:   s.Amount.Float64(),
				Date:          s.CreateTime.Time(),
				LastUpdated:   s.UpdateTime.Time(),
				Amount:        s.Quantity.Float64(),
				Type:          oType,
				Status:        orderStateFromString(s.State),
				AssetType:     stringToAccountType(s.AccountType),
			}, nil
		}
		oType, err := order.StringToOrderType(resp.Type)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:                resp.Price.Float64(),
			Amount:               resp.Quantity.Float64(),
			AverageExecutedPrice: resp.AveragePrice.Float64(),
			QuoteAmount:          resp.Amount.Float64(),
			ExecutedAmount:       resp.FilledQuantity.Float64(),
			RemainingAmount:      resp.Quantity.Float64() - resp.FilledAmount.Float64(),
			Cost:                 resp.FilledQuantity.Float64() * resp.AveragePrice.Float64(),
			Side:                 resp.Side,
			Exchange:             e.Name,
			OrderID:              resp.ID,
			ClientOrderID:        resp.ClientOrderID,
			Type:                 oType,
			Status:               orderStateFromString(resp.State),
			AssetType:            stringToAccountType(resp.AccountType),
			Date:                 resp.CreateTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			Pair:                 pair,
			Trades:               orderTrades,
			TimeInForce:          resp.TimeInForce,
		}, nil
	case asset.Futures:
		fResults, err := e.GetFuturesOrderHistory(ctx, pair, "", "", "", orderID, "", "", time.Time{}, time.Time{}, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(fResults) != 1 {
			return nil, order.ErrOrderNotFound
		}
		orderDetail := fResults[0]
		oType, err := order.StringToOrderType(orderDetail.OrderType)
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
			Exchange:             e.Name,
			ClientOrderID:        orderDetail.ClientOrderID,
			Type:                 oType,
			Side:                 orderDetail.Side,
			Status:               orderStateFromString(orderDetail.State),
			AssetType:            asset.Futures,
			Date:                 orderDetail.CreationTime.Time(),
			LastUpdated:          orderDetail.UpdateTime.Time(),
			Pair:                 orderDetail.Symbol,
			TimeInForce:          orderDetail.TimeInForce,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
}

// orderStateFromString returns an order.Status instance from a string representation
func orderStateFromString(orderState string) order.Status {
	switch strings.ToUpper(orderState) {
	case "NEW":
		return order.New
	case "FAILED":
		return order.Rejected
	case "FILLED":
		return order.Filled
	case "CANCELED":
		return order.Cancelled
	case "PENDING_CANCEL":
		return order.PendingCancel
	case "PARTIALLY_CANCELED":
		return order.PartiallyCancelled
	case "PARTIALLY_FILLED":
		return order.PartiallyFilled
	default:
		return order.UnknownStatus
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	depositAddrs, err := e.GetDepositAddresses(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	// Some coins use a main address, so we must use this in conjunction with the returned
	// deposit address to produce the full deposit address and payment-id
	currencyDetail, err := e.GetCurrency(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	for _, networkDetail := range currencyDetail.NetworkList {
		if networkDetail.CurrencyType == "address-payment-id" && networkDetail.DepositAddress != "" && (networkDetail.Blockchain == "" || networkDetail.Blockchain != chain) {
			paymentID, ok := depositAddrs[cryptocurrency.Upper().String()]
			if !ok {
				newAddr, err := e.NewCurrencyDepositAddress(ctx, cryptocurrency)
				if err != nil {
					return nil, err
				}
				paymentID = newAddr
			}
			return &deposit.Address{
				Address: networkDetail.DepositAddress,
				Tag:     paymentID,
				Chain:   networkDetail.Blockchain,
			}, nil
		}
	}

	var chainName string
	address, ok := depositAddrs[cryptocurrency.Upper().String()]
	if !ok {
		for _, networkDetail := range currencyDetail.NetworkList {
			if !networkDetail.DepositEnable {
				return nil, fmt.Errorf("deposits and withdrawals for %v are currently disabled", cryptocurrency.Upper().String())
			}

			newAddr, err := e.NewCurrencyDepositAddress(ctx, cryptocurrency)
			if err != nil {
				return nil, err
			}
			address = newAddr
			chainName = networkDetail.Blockchain
		}
	}
	return &deposit.Address{
		Address: address,
		Chain:   chainName,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.WithdrawCurrency(ctx, &WithdrawCurrencyRequest{
		Coin:       withdrawRequest.Currency,
		Network:    withdrawRequest.Crypto.Chain,
		Address:    withdrawRequest.Crypto.Address,
		Amount:     withdrawRequest.Amount,
		AddressTag: withdrawRequest.Crypto.AddressTag,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: e.Name,
		ID:   strconv.FormatUint(v.WithdrawRequestID, 10),
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	var samplePair currency.Pair
	if len(req.Pairs) == 1 {
		samplePair = req.Pairs[0]
	}
	var sideString string
	switch {
	case req.Side.IsLong():
		sideString = order.Buy.String()
	case req.Side.IsShort():
		sideString = order.Sell.String()
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		resp, err := e.GetOpenOrders(ctx, samplePair, sideString, "", req.FromOrderID, 0)
		if err != nil {
			return nil, err
		}
		for _, td := range resp {
			if len(req.Pairs) != 0 && !req.Pairs.Contains(td.Symbol, true) {
				continue
			}
			oType, err := order.StringToOrderType(td.Type)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Type:        oType,
				OrderID:     td.ID,
				Side:        td.Side,
				Amount:      td.Amount.Float64(),
				Date:        td.CreateTime.Time(),
				Price:       td.Price.Float64(),
				Pair:        td.Symbol,
				Exchange:    e.Name,
				TimeInForce: td.TimeInForce,
			})
		}
	case asset.Futures:
		fOrders, err := e.GetCurrentFuturesOrders(ctx, samplePair, sideString, "", "", 0, 0, 0)
		if err != nil {
			return nil, err
		}
		for _, fOrder := range fOrders {
			if len(req.Pairs) != 0 && !req.Pairs.Contains(fOrder.Symbol, true) {
				continue
			}
			var oState order.Status
			switch fOrder.State {
			case "NEW":
				oState = order.Active
			case "PARTIALLY_FILLED":
				oState = order.PartiallyFilled
			default:
				continue
			}
			var mType margin.Type
			switch fOrder.MarginMode {
			case "ISOLATED":
				mType = margin.Isolated
			case "CROSS":
				mType = margin.Multi
			}
			oType, err := order.StringToOrderType(fOrder.OrderType)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Type:            oType,
				OrderID:         fOrder.OrderID,
				Side:            fOrder.Side,
				Amount:          fOrder.Size.Float64(),
				Date:            fOrder.CreationTime.Time(),
				Price:           fOrder.Price.Float64(),
				Exchange:        e.Name,
				Pair:            fOrder.Symbol,
				ReduceOnly:      fOrder.ReduceOnly.Bool(),
				Leverage:        fOrder.Leverage.Float64(),
				ExecutedAmount:  fOrder.ExecQuantity.Float64(),
				RemainingAmount: fOrder.Size.Float64() - fOrder.ExecQuantity.Float64(),
				ClientOrderID:   fOrder.ClientOrderID,
				Status:          oState,
				AssetType:       req.AssetType,
				LastUpdated:     fOrder.UpdateTime.Time(),
				MarginType:      mType,
				FeeAsset:        fOrder.FeeCurrency,
				Fee:             fOrder.FeeAmount.Float64(),
				TimeInForce:     fOrder.TimeInForce,
			})
		}
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

func accountTypeString(assetType asset.Item) string {
	switch assetType {
	case asset.Spot:
		return "SPOT"
	case asset.Futures:
		return "FUTURES"
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
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.AssetType == asset.Spot {
		switch req.Type {
		case order.Market, order.Limit, order.UnknownType, order.AnyType:
			oTypeString, err := orderTypeString(req.Type)
			if err != nil {
				return nil, err
			}
			resp, err := e.GetOrdersHistory(ctx, &OrdersHistoryRequest{
				AccountType: accountTypeString(req.AssetType),
				OrderType:   oTypeString,
				Side:        req.Side,
				Limit:       100,
				StartTime:   req.StartTime,
				EndTime:     req.EndTime,
			})
			if err != nil {
				return nil, err
			}
			orders := make([]order.Detail, 0, len(resp))
			for _, tOrder := range resp {
				if len(req.Pairs) != 0 && !req.Pairs.Contains(tOrder.Symbol, true) {
					continue
				}
				oType, err := order.StringToOrderType(tOrder.Type)
				if err != nil {
					return nil, err
				}
				assetType, err := asset.New(tOrder.AccountType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					OrderID:              strconv.FormatUint(tOrder.ID, 10),
					Side:                 tOrder.Side,
					Amount:               tOrder.Amount.Float64(),
					ExecutedAmount:       tOrder.FilledAmount.Float64(),
					Price:                tOrder.Price.Float64(),
					AverageExecutedPrice: tOrder.AveragePrice.Float64(),
					Pair:                 tOrder.Symbol,
					Type:                 oType,
					Exchange:             e.Name,
					QuoteAmount:          tOrder.Amount.Float64() * tOrder.AveragePrice.Float64(),
					RemainingAmount:      tOrder.Quantity.Float64() - tOrder.FilledQuantity.Float64(),
					ClientOrderID:        tOrder.ClientOrderID,
					Status:               orderStateFromString(tOrder.State),
					AssetType:            assetType,
					Date:                 tOrder.CreateTime.Time(),
					LastUpdated:          tOrder.UpdateTime.Time(),
					TimeInForce:          tOrder.TimeInForce,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return req.Filter(e.Name, orders), nil
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			oTypeString, err := orderTypeString(req.Type)
			if err != nil {
				return nil, err
			}
			smartOrders, err := e.GetSmartOrderHistory(ctx,
				&OrdersHistoryRequest{
					Symbol:      currency.EMPTYPAIR,
					AccountType: accountTypeString(req.AssetType),
					OrderType:   oTypeString,
					Side:        req.Side,
					Limit:       100,
					StartTime:   req.StartTime,
					EndTime:     req.EndTime,
				},
			)
			if err != nil {
				return nil, err
			}
			orders := make([]order.Detail, 0, len(smartOrders))
			for _, smartOrder := range smartOrders {
				if len(req.Pairs) != 0 && !req.Pairs.Contains(smartOrder.Symbol, true) {
					continue
				}
				assetType, err := asset.New(smartOrder.AccountType)
				if err != nil {
					return nil, err
				}
				oType, err := order.StringToOrderType(smartOrder.Type)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					Side:          smartOrder.Side,
					Amount:        smartOrder.Amount.Float64(),
					Price:         smartOrder.Price.Float64(),
					TriggerPrice:  smartOrder.StopPrice.Float64(),
					Pair:          smartOrder.Symbol,
					Type:          oType,
					Exchange:      e.Name,
					OrderID:       smartOrder.ID,
					ClientOrderID: smartOrder.ClientOrderID,
					Status:        orderStateFromString(smartOrder.State),
					AssetType:     assetType,
					Date:          smartOrder.CreateTime.Time(),
					LastUpdated:   smartOrder.UpdateTime.Time(),
					TimeInForce:   smartOrder.TimeInForce,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return orders, nil
		default:
			return nil, fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, req.Type)
		}
	}
	oTypeString, err := orderTypeString(req.Type)
	if err != nil {
		return nil, err
	}
	orderHistory, err := e.GetFuturesOrderHistory(ctx, currency.EMPTYPAIR, oTypeString, req.Side.String(), "", "", "", "", req.StartTime, req.EndTime, 0, 100)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(orderHistory))
	for _, fOrder := range orderHistory {
		if len(req.Pairs) != 0 && !req.Pairs.Contains(fOrder.Symbol, true) {
			continue
		}
		oType, err := order.StringToOrderType(fOrder.OrderType)
		if err != nil {
			return nil, err
		}
		detail := order.Detail{
			Side:            fOrder.Side,
			Amount:          fOrder.Quantity.Float64(),
			ExecutedAmount:  fOrder.ExecutedAmount.Float64(),
			Price:           fOrder.Price.Float64(),
			Pair:            fOrder.Symbol,
			Type:            oType,
			Exchange:        e.Name,
			RemainingAmount: fOrder.Quantity.Float64() - fOrder.ExecutedAmount.Float64(),
			OrderID:         fOrder.OrderID,
			ClientOrderID:   fOrder.ClientOrderID,
			Status:          orderStateFromString(fOrder.State),
			AssetType:       asset.Futures,
			Date:            fOrder.CreationTime.Time(),
			LastUpdated:     fOrder.UpdateTime.Time(),
			TimeInForce:     fOrder.TimeInForce,
		}
		detail.InferCostsAndTimes()
		orders = append(orders, detail)
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		resp, err := e.GetCandlesticks(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for i, candleData := range resp {
			timeSeries[i] = kline.Candle{
				Time:   candleData.StartTime.Time(),
				Open:   candleData.Open.Float64(),
				High:   candleData.High.Float64(),
				Low:    candleData.Low.Float64(),
				Close:  candleData.Close.Float64(),
				Volume: candleData.Quantity.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		resp, err := e.GetFuturesKlineData(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for i, fCandle := range resp {
			timeSeries[i] = kline.Candle{
				Time:   fCandle.StartTime.Time(),
				Open:   fCandle.OpeningPrice.Float64(),
				Close:  fCandle.ClosingPrice.Float64(),
				High:   fCandle.HighestPrice.Float64(),
				Low:    fCandle.LowestPrice.Float64(),
				Volume: fCandle.BaseAmount.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	}

	return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	switch a {
	case asset.Spot:
		for i := range req.RangeHolder.Ranges {
			resp, err := e.GetCandlesticks(ctx,
				req.RequestFormatted,
				req.ExchangeInterval,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			for _, candleData := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candleData.StartTime.Time(),
					Open:   candleData.Open.Float64(),
					High:   candleData.High.Float64(),
					Low:    candleData.Low.Float64(),
					Close:  candleData.Close.Float64(),
					Volume: candleData.Quantity.Float64(),
				})
			}
		}
	case asset.Futures:
		for i := range req.RangeHolder.Ranges {
			resp, err := e.GetFuturesKlineData(ctx,
				req.RequestFormatted,
				interval,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
				500,
			)
			if err != nil {
				return nil, err
			}
			for _, fCandle := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   fCandle.StartTime.Time().UTC(),
					Open:   fCandle.OpeningPrice.Float64(),
					Close:  fCandle.ClosingPrice.Float64(),
					High:   fCandle.HighestPrice.Float64(),
					Low:    fCandle.LowestPrice.Float64(),
					Volume: fCandle.BaseAmount.Float64(),
				})
			}
		}
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencyDetail, err := e.GetCurrency(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if len(currencyDetail.NetworkList) == 0 {
		return nil, fmt.Errorf("%w for currency %v", errChainsNotFound, cryptocurrency)
	}
	chains := make([]string, len(currencyDetail.NetworkList))
	for i, cryptoNetwork := range currencyDetail.NetworkList {
		chains[i] = cryptoNetwork.Blockchain
	}
	return chains, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetSystemTimestamp(ctx)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, assetType asset.Item) ([]futures.Contract, error) {
	if !assetType.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if assetType != asset.Futures {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	contracts, err := e.GetFuturesAllProducts(ctx, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts))
	for i, productInfo := range contracts {
		ct := futures.Quarterly
		if strings.HasSuffix(productInfo.Symbol.Quote.String(), "PERP") {
			ct = futures.Perpetual
		}
		resp[i] = futures.Contract{
			Type:               ct,
			Exchange:           e.Name,
			SettlementCurrency: productInfo.SettlementCurrency,
			MarginCurrency:     productInfo.SettlementCurrency,
			Asset:              assetType,
			Name:               productInfo.Symbol,
			StartDate:          productInfo.ListingDate.Time(),
			IsActive:           strings.EqualFold(productInfo.Status, "OPEN"),
			Status:             productInfo.Status,
			MaxLeverage:        productInfo.Leverage.Float64(),
			SettlementType:     futures.Linear,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil || *r == (fundingrate.LatestRateRequest{}) {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrEmptyParams)
	}
	if !r.Pair.IsEmpty() {
		is, err := e.IsPerpetualFutureCurrency(r.Asset, r.Pair)
		if err != nil {
			return nil, err
		} else if !is {
			return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
	}
	contracts, err := e.GetFuturesHistoricalFundingRates(ctx, r.Pair, time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	timeChecked := time.Now()
	resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
	for _, fFundingRate := range contracts {
		var isPerp bool
		isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, fFundingRate.Symbol)
		if err != nil {
			return nil, err
		} else if !isPerp {
			continue
		}
		rate := fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     fFundingRate.Symbol,
			LatestRate: fundingrate.Rate{
				Time: fFundingRate.FundingRateSettleTime.Time(),
				Rate: decimal.NewFromFloat(fFundingRate.FundingRate.Float64()),
			},
			TimeOfNextRate: fFundingRate.NextFundingTime.Time(),
			TimeChecked:    timeChecked,
		}
		rate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fFundingRate.NextFundingTime.Time(),
			Rate: decimal.NewFromFloat(fFundingRate.NextPredictedFundingRate.Float64()),
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	return a == asset.Futures && strings.HasSuffix(cp.Quote.String(), "PERP"), nil
}

var priceScaleMultipliers = []float64{1, 0.1, 0.01, 0.001, 0.0001, 0.00001, 0.000001, 0.0000001, 0.00000001, 0.000000001, 0.0000000001, 0.00000000001, 0.000000000001, 0.0000000000001, 0.00000000000001, 0.000000000000001}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	if a == asset.Spot {
		instruments, err := e.GetSymbols(ctx)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(instruments))
		for i, symbolDetail := range instruments {
			l[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, symbolDetail.Symbol),
				MinimumBaseAmount:       symbolDetail.SymbolTradeLimit.MinQuantity.Float64(),
				MinimumQuoteAmount:      symbolDetail.SymbolTradeLimit.MinAmount.Float64(),
				MaximumBaseAmount:       symbolDetail.SymbolTradeLimit.MaxQuantity.Float64(),
				MaximumQuoteAmount:      symbolDetail.SymbolTradeLimit.MaxAmount.Float64(),
				PriceStepIncrementSize:  priceScaleMultipliers[symbolDetail.SymbolTradeLimit.PriceScale],
				AmountStepIncrementSize: priceScaleMultipliers[symbolDetail.SymbolTradeLimit.QuantityScale],
				QuoteStepIncrementSize:  priceScaleMultipliers[symbolDetail.SymbolTradeLimit.AmountScale],
			}
		}
		return limits.Load(l)
	}

	instruments, err := e.GetFuturesAllProducts(ctx, currency.EMPTYPAIR)
	if err != nil {
		return err
	}
	l := make([]limits.MinMaxLevel, len(instruments))
	for i, productInfo := range instruments {
		l[i] = limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, productInfo.Symbol),
			MinPrice:                productInfo.MinPrice.Float64(),
			MaxPrice:                productInfo.MaxPrice.Float64(),
			PriceStepIncrementSize:  productInfo.TickSize.Float64(),
			AmountStepIncrementSize: productInfo.LotSize.Float64(),
			MinimumBaseAmount:       productInfo.MinQuantity.Float64(),
			MaximumBaseAmount:       productInfo.MaxQuantity.Float64(),
			MinimumQuoteAmount:      productInfo.MinSize.Float64(),
			MarketMinQty:            productInfo.MinQuantity.Float64(),
			MarketMaxQty:            productInfo.MarketMaxQty.Float64(),
		}
	}
	return limits.Load(l)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	pairFormat, err := e.GetPairFormat(a, true)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		return mainURL + tradeSpotPath + pairFormat.Format(cp), nil
	case asset.Futures:
		return mainURL + tradeFuturesPath + pairFormat.Format(cp), nil
	default:
		return "", fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

// WebsocketSubmitOrder submits an order to the exchange via a websocket connection
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	if s.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, s.AssetType)
	}
	switch s.Type {
	case order.Market, order.Limit, order.LimitMaker:
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, s.Type)
	}
	var stpMode string
	switch s.TimeInForce {
	case order.PostOnly:
		stpMode = "EXPIRE_MAKER"
	case order.GoodTillCancel:
		stpMode = "EXPIRE_TAKER"
	}
	response, err := e.WsCreateOrder(ctx, &PlaceOrderRequest{
		Symbol:                  s.Pair,
		Price:                   s.Price,
		Quantity:                s.Amount,
		Amount:                  s.QuoteAmount,
		AllowBorrow:             false,
		Type:                    OrderType(s.Type),
		Side:                    s.Side,
		TimeInForce:             TimeInForce(s.TimeInForce),
		ClientOrderID:           s.ClientOrderID,
		SelfTradePreventionMode: stpMode,
	})
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(response.ID)
}

// WebsocketCancelOrder cancels an order via the websocket connection
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, req *order.Cancel) error {
	if req.OrderID == "" && req.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if err := req.Validate(req.StandardCancel()); err != nil {
		return err
	}
	_, err := e.WsCancelMultipleOrdersByIDs(ctx, []string{req.OrderID}, []string{req.ClientOrderID})
	return err
}

// orderTypeString return a string representation of order type
func orderTypeString(oType order.Type) (string, error) {
	switch oType {
	case order.Market, order.Limit, order.LimitMaker:
		return oType.String(), nil
	case order.StopLimit:
		return "STOP_LIMIT", nil
	case order.TrailingStopLimit:
		return "TRAILING_STOP_LIMIT", nil
	case order.AnyType, order.UnknownType:
		return "", nil
	}
	return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, oType)
}
