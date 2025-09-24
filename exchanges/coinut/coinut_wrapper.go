package coinut

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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

// SetDefaults sets current default values
func (e *Exchange) SetDefaults() {
	e.Name = "COINUT"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresClientID = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				SubmitOrders:      true,
				UserTradeHistory:  true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				GetOrders:              true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				UserTradeHistory:       true,
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoViaWebsiteOnly |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinutAPIURL,
		exchange.WebsocketSpot: coinutWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets the current exchange configuration
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinutWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.GenerateDefaultSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(33 * time.Millisecond),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	var resp Instruments
	var err error
	if e.Websocket.IsConnected() {
		resp, err = e.WsGetInstruments(ctx)
	} else {
		resp, err = e.GetInstruments(ctx)
	}
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(resp.Instruments))
	var pair currency.Pair
	for _, instrument := range resp.Instruments {
		if len(instrument) == 0 {
			return nil, errors.New("invalid data received")
		}
		e.instrumentMap.Seed(instrument[0].Base+instrument[0].Quote, instrument[0].InstrumentID)
		pair, err = currency.NewPairFromStrings(instrument[0].Base, instrument[0].Quote)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	if err := e.UpdatePairs(pairs, asset.Spot, false); err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (subAccts accounts.SubAccounts, err error) {
	var bal *UserBalance
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		if bal, err = e.wsGetAccountBalance(ctx); err != nil {
			return nil, err
		}
	} else {
		if bal, err = e.GetUserBalance(ctx); err != nil {
			return nil, err
		}
	}
	subAccts = accounts.SubAccounts{&accounts.SubAccount{
		AssetType: assetType,
		Balances: accounts.CurrencyBalances{
			currency.BCH:  {Currency: currency.BCH, Total: bal.BCH},
			currency.BTC:  {Currency: currency.BTC, Total: bal.BTC},
			currency.BTG:  {Currency: currency.BTG, Total: bal.BTG},
			currency.CAD:  {Currency: currency.CAD, Total: bal.CAD},
			currency.ETC:  {Currency: currency.ETC, Total: bal.ETC},
			currency.ETH:  {Currency: currency.ETH, Total: bal.ETH},
			currency.LCH:  {Currency: currency.LCH, Total: bal.LCH},
			currency.LTC:  {Currency: currency.LTC, Total: bal.LTC},
			currency.MYR:  {Currency: currency.MYR, Total: bal.MYR},
			currency.SGD:  {Currency: currency.SGD, Total: bal.SGD},
			currency.USD:  {Currency: currency.USD, Total: bal.USD},
			currency.XMR:  {Currency: currency.XMR, Total: bal.XMR},
			currency.ZEC:  {Currency: currency.ZEC, Total: bal.ZEC},
			currency.USDT: {Currency: currency.USDT, Total: bal.USDT},
		},
	}}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	err := e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	instID := e.instrumentMap.LookupID(fPair.String())
	if instID == 0 {
		return nil, errors.New("unable to lookup instrument ID")
	}
	var tick Ticker
	tick, err = e.GetInstrumentTicker(ctx, instID)
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Last:         tick.Last,
		High:         tick.High24,
		Low:          tick.Low24,
		Bid:          tick.HighestBuy,
		Ask:          tick.LowestSell,
		Volume:       tick.Volume24,
		Pair:         p,
		LastUpdated:  tick.Timestamp.Time(),
		ExchangeName: e.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	err := e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return book, err
	}

	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	instID := e.instrumentMap.LookupID(fPair.String())
	if instID == 0 {
		return book, errLookupInstrumentID
	}

	orderbookNew, err := e.GetInstrumentOrderbook(ctx, instID, 200)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Buy))
	for x := range orderbookNew.Buy {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Buy[x].Quantity,
			Price:  orderbookNew.Buy[x].Price,
		}
	}

	book.Asks = make(orderbook.Levels, len(orderbookNew.Sell))
	for x := range orderbookNew.Sell {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Sell[x].Quantity,
			Price:  orderbookNew.Sell[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	currencyID := e.instrumentMap.LookupID(p.String())
	if currencyID == 0 {
		return nil, errLookupInstrumentID
	}
	var tradeData Trades
	tradeData, err = e.GetTrades(ctx, currencyID)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData.Trades))
	for i := range tradeData.Trades {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData.Trades[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatInt(tradeData.Trades[i].TransactionID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData.Trades[i].Price,
			Amount:       tradeData.Trades[i].Quantity,
			Timestamp:    tradeData.Trades[i].Timestamp.Time(),
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	err := o.Validate(e.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	if _, err = strconv.Atoi(o.ClientID); err != nil {
		return nil, fmt.Errorf("%s - ClientID must be a number, received: %s", e.Name, o.ClientID)
	}

	var orderID string
	status := order.New
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var response *order.Detail
		response, err = e.wsSubmitOrder(ctx, &WsSubmitOrderParameters{
			Currency: o.Pair,
			Side:     o.Side,
			Amount:   o.Amount,
			Price:    o.Price,
		})
		if err != nil {
			return nil, err
		}
		orderID = response.OrderID
	} else {
		err = e.loadInstrumentsIfNotLoaded(ctx)
		if err != nil {
			return nil, err
		}

		var fPair currency.Pair
		fPair, err = e.FormatExchangeCurrency(o.Pair, asset.Spot)
		if err != nil {
			return nil, err
		}

		currencyID := e.instrumentMap.LookupID(fPair.String())
		if currencyID == 0 {
			return nil, errLookupInstrumentID
		}

		var APIResponse any
		var clientIDInt uint64
		clientIDInt, err = strconv.ParseUint(o.ClientID, 10, 32)
		if err != nil {
			return nil, err
		}
		APIResponse, err = e.NewOrder(ctx,
			currencyID,
			o.Amount,
			o.Price,
			o.Side.IsLong(),
			uint32(clientIDInt))
		if err != nil {
			return nil, err
		}
		responseMap, ok := APIResponse.(map[string]any)
		if !ok {
			return nil, errors.New("unable to type assert responseMap")
		}
		orderType, ok := responseMap["reply"].(string)
		if !ok {
			return nil, errors.New("unable to type assert orderType")
		}
		switch orderType {
		case "order_rejected":
			return nil, fmt.Errorf("clientOrderID: %v was rejected: %v", o.ClientID, responseMap["reasons"])
		case "order_filled":
			orderIDResp, ok := responseMap["order_id"].(float64)
			if !ok {
				return nil, errors.New("unable to type assert orderID")
			}
			orderID = strconv.FormatFloat(orderIDResp, 'f', -1, 64)
			status = order.Filled
		case "order_accepted":
			orderIDResp, ok := responseMap["order_id"].(float64)
			if !ok {
				return nil, errors.New("unable to type assert orderID")
			}
			orderID = strconv.FormatFloat(orderIDResp, 'f', -1, 64)
		}
	}
	resp, err := o.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	err := e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return err
	}
	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	fPair, err := e.FormatExchangeCurrency(o.Pair, asset.Spot)
	if err != nil {
		return err
	}

	currencyID := e.instrumentMap.LookupID(fPair.String())

	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp *CancelOrdersResponse
		resp, err = e.wsCancelOrder(ctx, &WsCancelOrderParameters{Currency: o.Pair, OrderID: orderIDInt})
		if err != nil {
			return err
		}
		if len(resp.Status) >= 1 && resp.Status[0] != "OK" {
			return errors.New(e.Name + " - Failed to cancel order " + o.OrderID)
		}
	} else {
		if currencyID == 0 {
			return errLookupInstrumentID
		}
		_, err = e.CancelExistingOrder(ctx, currencyID, orderIDInt)
		if err != nil {
			return err
		}
	}

	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	req := make([]CancelOrders, 0, len(o))
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			return nil, order.ErrClientOrderIDNotSupported
		case o[i].OrderID != "":
			currencyID := e.instrumentMap.LookupID(o[i].Pair.String())
			oid, err := strconv.ParseInt(o[i].OrderID, 10, 64)
			if err != nil {
				return nil, err
			}
			req = append(req, CancelOrders{
				InstrumentID: currencyID,
				OrderID:      oid,
			})
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	results, err := e.CancelOrders(ctx, req)
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	for i := range results.Results {
		resp.Status[strconv.FormatInt(results.Results[i].OrderID, 10)] = results.Results[i].Status
	}
	return resp, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, details *order.Cancel) (order.CancelAllResponse, error) {
	if err := details.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var cancelAllOrdersResponse order.CancelAllResponse
	err := e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}
	cancelAllOrdersResponse.Status = make(map[string]string)
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		openOrders, err := e.wsGetOpenOrders(ctx, details.Pair.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var ordersToCancel []WsCancelOrderParameters
		for i := range openOrders.Orders {
			var fPair currency.Pair
			fPair, err = e.FormatExchangeCurrency(details.Pair, asset.Spot)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if openOrders.Orders[i].InstrumentID == e.instrumentMap.LookupID(fPair.String()) {
				ordersToCancel = append(ordersToCancel, WsCancelOrderParameters{
					Currency: details.Pair,
					OrderID:  openOrders.Orders[i].OrderID,
				})
			}
		}
		resp, err := e.wsCancelOrders(ctx, ordersToCancel)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range resp.Results {
			if openOrders.Orders[i].Status[0] != "OK" {
				cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders.Orders[i].OrderID, 10)] = strings.Join(openOrders.Orders[i].Status, ",")
			}
		}
	} else {
		var allTheOrders []OrderResponse
		ids := e.instrumentMap.GetInstrumentIDs()
		for x := range ids {
			fPair, err := e.FormatExchangeCurrency(details.Pair, asset.Spot)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if ids[x] == e.instrumentMap.LookupID(fPair.String()) {
				openOrders, err := e.GetOpenOrders(ctx, ids[x])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				allTheOrders = append(allTheOrders, openOrders.Orders...)
			}
		}

		var allTheOrdersToCancel []CancelOrders
		for i := range allTheOrders {
			cancelOrder := CancelOrders{
				InstrumentID: allTheOrders[i].InstrumentID,
				OrderID:      allTheOrders[i].OrderID,
			}
			allTheOrdersToCancel = append(allTheOrdersToCancel, cancelOrder)
		}

		if len(allTheOrdersToCancel) > 0 {
			resp, err := e.CancelOrders(ctx, allTheOrdersToCancel)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			for i := range resp.Results {
				if resp.Results[i].Status != "OK" {
					cancelAllOrdersResponse.Status[strconv.FormatInt(resp.Results[i].OrderID, 10)] = resp.Results[i].Status
				}
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	err = e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	var currenciesToCheck []string
	if len(req.Pairs) == 0 {
		for i := range req.Pairs {
			var fPair currency.Pair
			fPair, err = e.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
			if err != nil {
				return nil, err
			}
			currenciesToCheck = append(currenciesToCheck, fPair.String())
		}
	} else {
		for k := range e.instrumentMap.Instruments {
			currenciesToCheck = append(currenciesToCheck, k)
		}
	}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for x := range currenciesToCheck {
			var openOrders *WsUserOpenOrdersResponse
			openOrders, err = e.wsGetOpenOrders(ctx, currenciesToCheck[x])
			if err != nil {
				return nil, err
			}
			for i := range openOrders.Orders {
				var p currency.Pair
				p, err = currency.NewPairFromString(currenciesToCheck[x])
				if err != nil {
					return nil, err
				}

				var fPair currency.Pair
				fPair, err = e.FormatExchangeCurrency(p, asset.Spot)
				if err != nil {
					return nil, err
				}

				var side order.Side
				side, err = order.StringToOrderSide(openOrders.Orders[i].Side)
				if err != nil {
					return nil, err
				}

				orders = append(orders, order.Detail{
					Exchange:        e.Name,
					OrderID:         strconv.FormatInt(openOrders.Orders[i].OrderID, 10),
					Pair:            fPair,
					Side:            side,
					Date:            openOrders.Orders[i].Timestamp.Time(),
					Status:          order.Active,
					Price:           openOrders.Orders[i].Price,
					Amount:          openOrders.Orders[i].Quantity,
					ExecutedAmount:  openOrders.Orders[i].Quantity - openOrders.Orders[i].OpenQuantity,
					RemainingAmount: openOrders.Orders[i].OpenQuantity,
				})
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			var curr currency.Pair
			curr, err = e.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
			if err != nil {
				return nil, err
			}
			instrumentsToUse = append(instrumentsToUse,
				e.instrumentMap.LookupID(curr.String()))
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = e.instrumentMap.GetInstrumentIDs()
		}

		var pairs currency.Pairs
		pairs, err = e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}

		var format currency.PairFormat
		format, err = e.GetPairFormat(asset.Spot, true)
		if err != nil {
			return nil, err
		}

		for x := range instrumentsToUse {
			var openOrders GetOpenOrdersResponse
			openOrders, err = e.GetOpenOrders(ctx, instrumentsToUse[x])
			if err != nil {
				return nil, err
			}
			for y := range openOrders.Orders {
				curr := e.instrumentMap.LookupInstrument(instrumentsToUse[x])
				var p currency.Pair
				p, err = currency.NewPairFromFormattedPairs(curr, pairs, format)
				if err != nil {
					return nil, err
				}

				var side order.Side
				side, err = order.StringToOrderSide(openOrders.Orders[y].Side)
				if err != nil {
					return nil, err
				}

				orders = append(orders, order.Detail{
					OrderID:  strconv.FormatInt(openOrders.Orders[y].OrderID, 10),
					Amount:   openOrders.Orders[y].Quantity,
					Price:    openOrders.Orders[y].Price,
					Exchange: e.Name,
					Side:     side,
					Date:     openOrders.Orders[y].Timestamp.Time(),
					Pair:     p,
				})
			}
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	err = e.loadInstrumentsIfNotLoaded(ctx)
	if err != nil {
		return nil, err
	}
	var allOrders []order.Detail
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for i := range req.Pairs {
			for j := int64(0); ; j += 100 {
				var trades *WsTradeHistoryResponse
				trades, err = e.wsGetTradeHistory(ctx, req.Pairs[i], j, 100)
				if err != nil {
					return allOrders, err
				}
				for x := range trades.Trades {
					curr := e.instrumentMap.LookupInstrument(trades.Trades[x].InstrumentID)
					var p currency.Pair
					p, err = currency.NewPairFromString(curr)
					if err != nil {
						return nil, err
					}

					var side order.Side
					side, err = order.StringToOrderSide(trades.Trades[x].Side)
					if err != nil {
						return nil, err
					}

					detail := order.Detail{
						Exchange:        e.Name,
						OrderID:         strconv.FormatInt(trades.Trades[x].OrderID, 10),
						Pair:            p,
						Side:            side,
						Date:            trades.Trades[x].Timestamp.Time(),
						Status:          order.Filled,
						Price:           trades.Trades[x].Price,
						Amount:          trades.Trades[x].Quantity,
						ExecutedAmount:  trades.Trades[x].Quantity - trades.Trades[x].OpenQuantity,
						RemainingAmount: trades.Trades[x].OpenQuantity,
					}
					detail.InferCostsAndTimes()
					allOrders = append(allOrders, detail)
				}
				if len(trades.Trades) < 100 {
					break
				}
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			var curr currency.Pair
			curr, err = e.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
			if err != nil {
				return nil, err
			}

			instrumentID := e.instrumentMap.LookupID(curr.String())
			if instrumentID > 0 {
				instrumentsToUse = append(instrumentsToUse, instrumentID)
			}
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = e.instrumentMap.GetInstrumentIDs()
		}

		var pairs currency.Pairs
		pairs, err = e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}

		var format currency.PairFormat
		format, err = e.GetPairFormat(asset.Spot, true)
		if err != nil {
			return nil, err
		}

		for x := range instrumentsToUse {
			var orders TradeHistory
			orders, err = e.GetTradeHistory(ctx, instrumentsToUse[x], -1, -1)
			if err != nil {
				return nil, err
			}
			for y := range orders.Trades {
				curr := e.instrumentMap.LookupInstrument(instrumentsToUse[x])
				var p currency.Pair
				p, err = currency.NewPairFromFormattedPairs(curr, pairs, format)
				if err != nil {
					return nil, err
				}

				var side order.Side
				side, err = order.StringToOrderSide(orders.Trades[y].Order.Side)
				if err != nil {
					return nil, err
				}

				allOrders = append(allOrders, order.Detail{
					OrderID:  strconv.FormatInt(orders.Trades[y].Order.OrderID, 10),
					Amount:   orders.Trades[y].Order.Quantity,
					Price:    orders.Trades[y].Order.Price,
					Exchange: e.Name,
					Side:     side,
					Date:     orders.Trades[y].Order.Timestamp.Time(),
					Pair:     p,
				})
			}
		}
	}
	return req.Filter(e.Name, allOrders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	return e.wsAuthenticate(ctx)
}

func (e *Exchange) loadInstrumentsIfNotLoaded(ctx context.Context) error {
	if !e.instrumentMap.IsLoaded() {
		if e.Websocket.IsConnected() {
			_, err := e.WsGetInstruments(ctx)
			if err != nil {
				return err
			}
		} else {
			err := e.SeedInstruments(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
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
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = ""
	return tradeBaseURL + cp.Upper().String() + "/", nil
}
