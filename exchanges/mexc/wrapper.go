package mexc

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

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
				// /api/v3/klines returns at most 500 rows whatever limit is asked for, so each range
				// GetKlineExtendedRequest carves must be sized at 500; a larger limit fills a range half
				// way and zero-pads the remainder silently.
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}
	var err error
	e.Requester, err = request.New(
		e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      spotAPIURL,
		exchange.WebsocketSpot: spotWebsocketURL,
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
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: exch,
		Features:       &e.Features.Supports.WebsocketCapabilities,
		// MEXC caps a single spot websocket connection at 30 subscriptions.
		MaxWebsocketSubscriptionsPerConnection: 30,
		DefaultURL:                             spotWebsocketURL,
		RunningURL:                             spotWebsocketURL,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   spotWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      time.Second * 3,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Handler:               e.WsHandleData,
		MessageFilter:         asset.Spot,
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
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
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
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
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
			// Resolve the concatenated symbol against the known pairs. A naive split guesses the
			// base/quote boundary and gets it wrong for any symbol whose split is ambiguous
			// (measured against the live catalogue: 1376 of 2067 spot symbols, e.g. METALUSDT was
			// read as MET/ALUSDT). MatchSymbolWithAvailablePairs looks the symbol up instead. The
			// 24h endpoint returns every listed symbol, so a symbol we do not track (delisted between
			// a catalogue refresh and this poll) is skipped rather than failing the whole update or,
			// as the naive split did, inventing a mis-split pair.
			pair, err := e.MatchSymbolWithAvailablePairs(tickers[t].Symbol, assetType, false)
			if err != nil {
				continue
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
	default:
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
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
		result, err := e.GetOrderbook(ctx, fPair, 1000)
		if err != nil {
			return book, err
		}
		book.Bids = result.Bids.Levels()
		book.Asks = result.Asks.Levels()
		// The venue carries its own timestamp and update id; the same correction as wsSendTime on the
		// ws side. Without them Process falls back to time.Now() and a zero update id.
		book.LastUpdated = result.Timestamp.Time()
		book.LastUpdateID = result.LastUpdateID
		if err := book.Process(); err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// accountTypeMatches reports whether the account type the exchange returned is the one asked for.
// The comparison is case-insensitive: the exchange reports "SPOT" while asset.Item renders "spot",
// so a direct comparison never matched and every spot balance request returned ErrNotSupported
// instead of the balances.
func accountTypeMatches(reported string, assetType asset.Item) bool {
	return assetType == asset.Empty || strings.EqualFold(reported, assetType.String())
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	accountInfo, err := e.GetAccountInformation(ctx)
	if err != nil {
		return nil, err
	}
	if !accountTypeMatches(accountInfo.AccountType, assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}

	subAccount := accounts.SubAccount{
		AssetType: assetType,
		Balances:  make(accounts.CurrencyBalances, len(accountInfo.Balances)),
	}
	for b := range accountInfo.Balances {
		ccy := currency.NewCode(accountInfo.Balances[b].Asset)
		// MEXC reports free (available) and locked (frozen) per asset. The account layer keeps the
		// three views independently: Total = free + locked, Hold = locked, Free = free. Free was left
		// unset, so an asset with free=10/locked=3 reported Total=13/Hold=3/Free=0 - a consumer
		// reading available balance saw nothing.
		subAccount.Balances[ccy] = accounts.Balance{
			Currency: ccy,
			Free:     accountInfo.Balances[b].Free.Float64(),
			Hold:     accountInfo.Balances[b].Locked.Float64(),
			Total:    accountInfo.Balances[b].Free.Float64() + accountInfo.Balances[b].Locked.Float64(),
		}
	}

	subAccounts := accounts.SubAccounts{&subAccount}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
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
	withdrawals, err := e.GetWithdrawalHistory(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	depositsLen := len(result)
	resp := make([]exchange.FundingHistory, depositsLen+len(withdrawals))
	for a := range result {
		resp[a] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          accountStatusToString(result[a].Status),
			TransferID:      result[a].TransactionID,
			Timestamp:       result[a].InsertTime.Time(),
			Currency:        result[a].Coin.String(),
			Amount:          result[a].Amount.Float64(),
			CryptoToAddress: result[a].Address,
			TransferType:    "diposit",
		}
	}
	for w := range withdrawals {
		resp[depositsLen+w] = exchange.FundingHistory{
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
		result, err := e.GetRecentTradesList(ctx, p, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result))
		for t := range result {
			side := order.Buy
			if result[t].IsBuyerMaker { // the buyer was the maker, so the taker sold
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
	case asset.Spot:
		result, err := e.GetAggregatedTrades(ctx, p, startTime, endTime, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result))
		for t := range result {
			oSide := order.Buy
			if result[t].MakerBuyer { // the buyer was the maker, so the taker sold
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
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
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
		result, err := e.NewOrder(ctx, s.Pair, s.ClientOrderID, s.Side.String(), orderTypeString, s.Amount, s.QuoteAmount, s.Price)
		if err != nil {
			return nil, err
		}
		if result.ClientOrderID == "" {
			// If the ACK omits the client id, report the id sent with the request.
			result.ClientOrderID = s.ClientOrderID
		}
		orderType, tif, err := e.StringToOrderTypeAndTimeInForce(result.Type)
		if err != nil {
			return nil, err
		}
		orderSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		var ordStatus order.Status
		switch {
		case result.Status != "":
			ordStatus, err = orderStatusFromString(result.Status)
			if err != nil {
				return nil, err
			}
		case result.OrderID != "":
			// MEXC's create-order ACK omits status; a populated OrderID from a successful NewOrder
			// means the order was placed, so report New to keep WasOrderPlaced() true instead of
			// UnknownStatus. The sweep resolves the real lifecycle status (FILLED/PARTIALLY_FILLED/…)
			// from GetOrderInfo afterwards.
			ordStatus = order.New
		}
		return &order.SubmitResponse{
			// s.Pair is already in exchange format; the response symbol is concatenated and a naive
			// split mis-reads most MEXC symbols (METALUSDT read as MET/ALUSDT).
			Pair:            s.Pair,
			Exchange:        e.Name,
			Type:            orderType,
			Side:            orderSide,
			AssetType:       asset.Spot,
			Leverage:        s.Leverage,
			ReduceOnly:      s.ReduceOnly,
			Status:          ordStatus,
			QuoteAmount:     s.QuoteAmount,
			OrderID:         result.OrderID,
			ClientOrderID:   result.ClientOrderID,
			Price:           result.Price.Float64(),
			Amount:          result.OrigQty.Float64(),
			LastUpdated:     result.TransactTime.Time(),
			RemainingAmount: result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			TimeInForce:     tif,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if ord.AssetType != asset.Spot {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	// MEXC expects the symbol without a delimiter (BTCUSDT). Passing ord.Pair through unformatted
	// sent BTC-USDT when the pair carried a delimiter, which the exchange rejects. An empty pair is
	// left to CancelTradeOrder to reject, preserving its ErrSymbolStringEmpty contract.
	pair := ord.Pair
	if !pair.IsEmpty() {
		var err error
		pair, err = e.FormatExchangeCurrency(pair, ord.AssetType)
		if err != nil {
			return err
		}
	}
	_, err := e.CancelTradeOrder(ctx, pair, ord.OrderID, ord.ClientOrderID, "")
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(context.Context, []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// This is a symbol-wide cancel: it cancels every open order for the pair and takes no order id,
	// so StandardCancel() (which requires an OrderID) must not gate it - it rejected a valid
	// symbol-wide request with order.ErrOrderIDNotSet.
	if err := orderCancellation.Validate(); err != nil {
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
		result, err := e.CancelAllOpenOrdersBySymbol(ctx, orderCancellation.Pair)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		for r := range result {
			resp.Status[result[r].OrderID] = "cancelled"
		}
		return resp, nil
	default:
		return order.CancelAllResponse{}, asset.ErrNotSupported
	}
}

// averageExecutedPrice returns the average fill price of a REST order, cummulativeQuoteQty over
// executedQty. The price field is not the average: it is the limit on a limit order, and a filled
// market order reports a price that differs from its fills.
func averageExecutedPrice(o *OrderDetail) float64 {
	executed := o.ExecutedQty.Float64()
	if executed <= 0 {
		return 0
	}
	return o.CummulativeQuoteQty.Float64() / executed
}

// tradesForOrder fetches the fills of a spot order and maps them to domain trade records plus the
// aggregated commission. MEXC charges commission per fill in an asset the venue chooses (base,
// quote, or the MX discount token), so the fee currency is read from the fill and never assumed;
// when fills disagree on the asset the aggregate currency is left unset. It is best-effort: a
// myTrades failure must not sink the order lookup, so callers pass through the base order.
func (e *Exchange) tradesForOrder(ctx context.Context, pair currency.Pair, orderID string) (trades []order.TradeHistory, totalFee float64, feeAsset currency.Code) {
	// MEXC can report an order as filled a moment before its fills surface in myTrades, so a single
	// immediate lookup sometimes finds nothing for a just-completed order. Retry once with a short
	// gap rather than polling repeatedly. Best-effort throughout: the order lookup still returns
	// without commission on failure, but the reason is named rather than dropped silently. The limit
	// is set to the documented maximum (1000) so an order with more than the default page of fills
	// does not undercount its commission.
	var fills []*AccountTrade
	for attempt := 0; ; attempt++ {
		var err error
		fills, err = e.GetAccountTradeList(ctx, pair, orderID, time.Time{}, time.Time{}, 1000)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s: myTrades lookup failed for order %s (%s): %v", e.Name, orderID, pair, err)
			return nil, 0, currency.EMPTYCODE
		}
		if len(fills) > 0 || attempt >= 1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, 0, currency.EMPTYCODE
		case <-time.After(time.Second):
		}
	}
	if len(fills) == 0 {
		log.Warnf(log.ExchangeSys, "%s: myTrades returned no fills for order %s (%s); commission not materialised", e.Name, orderID, pair)
		return nil, 0, currency.EMPTYCODE
	}
	trades = make([]order.TradeHistory, 0, len(fills))
	uniformFee := true
	for _, f := range fills {
		side := order.Buy
		if !f.IsBuyer {
			side = order.Sell
		}
		fillAsset := currency.NewCode(f.CommissionAsset)
		totalFee += f.Commission.Float64()
		switch {
		case feeAsset.IsEmpty():
			feeAsset = fillAsset
		case !feeAsset.Equal(fillAsset):
			uniformFee = false
		}
		trades = append(trades, order.TradeHistory{
			Price:     f.Price.Float64(),
			Amount:    f.Quantity.Float64(),
			Fee:       f.Commission.Float64(),
			Exchange:  e.Name,
			TID:       f.ID,
			Side:      side,
			Timestamp: f.Time.Time(),
			IsMaker:   f.IsMaker,
			FeeAsset:  f.CommissionAsset,
			Total:     f.QuoteQuantity.Float64(),
		})
	}
	if !uniformFee {
		// Commissions charged in different assets cannot be summed into a single figure: the total
		// would be a meaningless mix of currencies with a stale label. Report no aggregate fee and no
		// currency; each fill's own commission and asset are still carried on the TradeHistory records.
		totalFee = 0
		feeAsset = currency.EMPTYCODE
	}
	var breakdown strings.Builder
	for i, f := range fills {
		if i > 0 {
			breakdown.WriteByte(' ')
		}
		fmt.Fprintf(&breakdown, "%v/%q", f.Commission.Float64(), f.CommissionAsset)
	}
	log.Debugf(log.ExchangeSys, "%s: order %s myTrades fills=%d totalFee=%v feeAsset=%q uniform=%v [%s]",
		e.Name, orderID, len(fills), totalFee, feeAsset.String(), uniformFee, breakdown.String())
	return trades, totalFee, feeAsset
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
		result, err := e.GetOrderByID(ctx, pair.Format(pairFormat), "", orderID)
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
			oStatus, err = orderStatusFromString(result.Status)
			if err != nil {
				return nil, err
			}
		}
		// The Query Order response carries time and updateTime but no transactTime (that field only
		// exists on the New Order response), so LastUpdated must come from updateTime and fall back to
		// the creation time when the order is still open (updateTime null decodes to the zero time).
		lastUpdated := result.UpdateTime.Time()
		if lastUpdated.IsZero() {
			lastUpdated = result.Time.Time()
		}
		detail := &order.Detail{
			Price:       result.Price.Float64(),
			Amount:      result.OrigQty.Float64(),
			QuoteAmount: result.CummulativeQuoteQty.Float64(),
			// Cost is the quote actually spent (cumulative filled value); the rpc server maps
			// Detail.Cost to the proto cost field, so a market order's real executed cost reaches
			// the caller instead of a zero. Price alone is the protective limit, not the average.
			Cost:                 result.CummulativeQuoteQty.Float64(),
			AverageExecutedPrice: averageExecutedPrice(result),
			TriggerPrice:         result.StopPrice.Float64(),
			ExecutedAmount:       result.ExecutedQty.Float64(),
			RemainingAmount:      result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			Exchange:             e.Name,
			OrderID:              result.OrderID,
			ClientOrderID:        result.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Spot,
			Date:                 result.Time.Time(),
			LastUpdated:          lastUpdated,
			// pair is the pair the caller asked for; the response symbol is concatenated and a naive
			// split mis-reads most MEXC symbols (METALUSDT read as MET/ALUSDT).
			Pair:        pair.Format(pairFormat),
			TimeInForce: tif,
		}
		// Enrich with the venue's commission facts when the order actually filled. The Query Order
		// response carries no commission, so the fee and its currency come from myTrades keyed by
		// this order. Gate on cumulative quote value rather than executedQty: MEXC reports
		// executedQty=0 on some filled limit orders, so executedQty is not a reliable "has fills".
		if result.CummulativeQuoteQty.Float64() > 0 {
			if trades, fee, feeAsset := e.tradesForOrder(ctx, pair.Format(pairFormat), orderID); len(trades) > 0 {
				detail.Trades = trades
				detail.Fee = fee
				detail.FeeAsset = feeAsset
				// MEXC returns executedQty=0 on some filled limit orders while still reporting a
				// non-zero cummulativeQuoteQty and returning the fills in myTrades. When executedQty is
				// zero the executed amount, remaining amount and average price are derived from the
				// fills, the venue's own record of what actually traded, instead of the zero field.
				if result.ExecutedQty.Float64() == 0 {
					var executed float64
					for i := range trades {
						executed += trades[i].Amount
					}
					if executed > 0 {
						detail.ExecutedAmount = executed
						detail.RemainingAmount = result.OrigQty.Float64() - executed
						detail.AverageExecutedPrice = result.CummulativeQuoteQty.Float64() / executed
					}
				}
			}
		}
		return detail, nil
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
	if len(result) == 0 {
		return nil, deposit.ErrAddressNotFound
	}
	// Without a pinned network the venue returns one address per network; take the first. The
	// destination tag arrives as memo (the field table documents memo, not tag), so read memo and
	// fall back to tag.
	tag := result[0].Memo
	if tag == "" {
		tag = result[0].Tag
	}
	return &deposit.Address{
		Address: result[0].Address,
		Tag:     tag,
		Chain:   result[0].Network,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// orderDetailFromRESTOrder maps a spot REST order record to a domain order.Detail. It is the single
// mapping shared by GetActiveOrders and GetOrderHistory: both previously omitted the pair and the
// order's real timestamps, and GetOrderHistory parsed the type with the generic order.StringToOrderType,
// which does not know MEXC's IMMEDIATE_OR_CANCEL/FILL_OR_KILL/LIMIT_MAKER types and failed the whole
// query, returning nothing.
func (e *Exchange) orderDetailFromRESTOrder(o *OrderDetail, fallbackPair currency.Pair) (order.Detail, error) {
	// A symbol that has left the available pairs (delisted with a working order, or a catalogue not
	// yet refreshed) must not sink the whole listing; fall back to the pair the caller asked for
	// instead of failing, the same way UpdateTickers skips an untracked symbol rather than erroring.
	pair, err := e.MatchSymbolWithAvailablePairs(o.Symbol, asset.Spot, false)
	if err != nil {
		pair = fallbackPair
	}
	oType, tif, err := e.StringToOrderTypeAndTimeInForce(o.Type)
	if err != nil {
		return order.Detail{}, err
	}
	oSide, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	var oStatus order.Status
	if o.Status != "" {
		oStatus, err = orderStatusFromString(o.Status)
		if err != nil {
			return order.Detail{}, err
		}
	}
	// MEXC returns updateTime:null on an open (still-working) order, which decodes to the zero time.
	// Fall back to the creation time so LastUpdated is never the zero time.
	lastUpdated := o.UpdateTime.Time()
	if lastUpdated.IsZero() {
		lastUpdated = o.Time.Time()
	}
	return order.Detail{
		Price:                o.Price.Float64(),
		Amount:               o.OrigQty.Float64(),
		AverageExecutedPrice: averageExecutedPrice(o),
		TriggerPrice:         o.StopPrice.Float64(),
		QuoteAmount:          o.CummulativeQuoteQty.Float64(),
		// Cost is the quote actually spent (cumulative filled value), mapped to the proto cost
		// field by the rpc server; without it a market order reports a zero cost to the caller.
		Cost:            o.CummulativeQuoteQty.Float64(),
		ExecutedAmount:  o.ExecutedQty.Float64(),
		RemainingAmount: o.OrigQty.Float64() - o.ExecutedQty.Float64(),
		Exchange:        e.Name,
		OrderID:         o.OrderID,
		ClientOrderID:   o.ClientOrderID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       asset.Spot,
		Date:            o.Time.Time(),
		LastUpdated:     lastUpdated,
		Pair:            pair,
		TimeInForce:     tif,
	}, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details []order.Detail
		for p := range getOrdersRequest.Pairs {
			result, err := e.GetOpenOrders(ctx, getOrdersRequest.Pairs[p].Format(pairFormat))
			if err != nil {
				return nil, err
			}
			for r := range result {
				detail, err := e.orderDetailFromRESTOrder(result[r], getOrdersRequest.Pairs[p].Format(pairFormat))
				if err != nil {
					return nil, err
				}
				details = append(details, detail)
			}
		}
		// The request's side, type and time filters were ignored; apply them here so a caller asking
		// for only buys or only limit orders is not handed the full open-order set.
		return getOrdersRequest.Filter(e.Name, details), nil
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details []order.Detail
		for p := range getOrdersRequest.Pairs {
			pair := getOrdersRequest.Pairs[p].Format(pairFormat)
			result, err := e.GetAllOrders(ctx, pair, getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0)
			if err != nil {
				return nil, err
			}
			for r := range result {
				detail, err := e.orderDetailFromRESTOrder(result[r], pair)
				if err != nil {
					return nil, err
				}
				details = append(details, detail)
			}
		}
		// The request's side and type filters were ignored; apply them here so a caller asking for
		// only sells or only limit orders is not handed the full history.
		return getOrdersRequest.Filter(e.Name, details), nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	// GetFeeByType returns the absolute fee amount, not the rate. The amount is rate * price * quantity;
	// returning the bare rate reported e.g. 0.0005 as if it were the fee. The offline branch is the
	// same calculation against a fixed worst-case rate, used when no credentials are available to ask
	// the exchange for the account's own schedule.
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		if feeBuilder.IsMaker {
			return 0., nil
		}
		return 0.0005 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyTradeFee:
		result, err := e.GetSymbolTradingFee(ctx, feeBuilder.Pair)
		if err != nil {
			return 0, err
		}
		rate := result.Data.TakerCommission
		if feeBuilder.IsMaker {
			rate = result.Data.MakerCommission
		}
		return rate * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyWithdrawalFee:
	case exchange.CryptocurrencyDepositFee:
	case exchange.InternationalBankDepositFee:
	}
	return 0, nil
}

// candlesFromCandlestick maps the exchange's candlestick rows to kline candles. The candle is stamped
// with its open time (the interval start), which is this repository's kline convention: the exchange
// also reports a close time, and stamping the candle with that shifted every candle forward by one
// interval. It is the single mapping shared by GetHistoricCandles and GetHistoricCandlesExtended.
func candlesFromCandlestick(result []*CandlestickData) []kline.Candle {
	candles := make([]kline.Candle, len(result))
	for c := range result {
		candles[c] = kline.Candle{
			Open:   result[c].OpenPrice.Float64(),
			High:   result[c].HighPrice.Float64(),
			Low:    result[c].LowPrice.Float64(),
			Close:  result[c].ClosePrice.Float64(),
			Volume: result[c].Volume.Float64(),
			Time:   result[c].OpenTime.Time(),
		}
	}
	return candles
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
		result, err := e.GetCandlestick(ctx, pair, intervalString, start, end, 0)
		if err != nil {
			return nil, err
		}
		return req.ProcessResponse(candlesFromCandlestick(result))
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
			result, err := e.GetCandlestick(
				ctx,
				pair.Format(pFormat),
				intervalString,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			timeSeries = append(timeSeries, candlesFromCandlestick(result)...)
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
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
			// quoteAmountPrecision is the minimum quote order amount (measured live for METALUSDT:
			// "1" USDT), not a price step; the price tick is 10^-quotePrecision (quotePrecision=5 =>
			// 0.00001) and the base amount step is 10^-baseAssetPrecision. The previous mapping put
			// the min-notional value ("1") into both the price and quote step, quantizing prices to
			// whole units.
			l[a] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, assetType, pair.Format(pairFormat)),
				PriceStepIncrementSize:  math.Pow(10, -result.Symbols[a].QuotePrecision),
				AmountStepIncrementSize: math.Pow(10, -result.Symbols[a].BaseAssetPrecision),
				QuoteStepIncrementSize:  math.Pow(10, -result.Symbols[a].QuoteAssetPrecision),
				MinimumQuoteAmount:      result.Symbols[a].QuoteAmountPrecision.Float64(),
				MinNotional:             result.Symbols[a].QuoteAmountPrecision.Float64(),
				MaximumQuoteAmount:      result.Symbols[a].MaxQuoteAmount.Float64(),
				MinimumBaseAmount:       result.Symbols[a].BaseSizePrecision.Float64(),
			}
		}
		if err := limits.Load(l); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// orderStatusFromString converts a MEXC order status into the common order.Status.
//
// MEXC reports "PARTIALLY_CANCELED", a spelling the shared order.StringToOrderStatus parser does
// not know. That spelling belongs to this venue's adapter rather than to the shared parser:
// extending the shared vocabulary would change behaviour for every exchange for the sake of one.
// All other spellings are delegated to the shared parser unchanged.
func orderStatusFromString(status string) (order.Status, error) {
	if status == "PARTIALLY_CANCELED" {
		return order.PartiallyCancelled, nil
	}
	return order.StringToOrderStatus(status)
}
