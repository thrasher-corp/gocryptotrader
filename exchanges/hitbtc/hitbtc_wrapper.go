package hitbtc

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

// SetDefaults sets default settings for hitbtc
func (e *Exchange) SetDefaults() {
	e.Name = "HitBTC"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				ModifyOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				SubmitOrder:            true,
				CancelOrder:            true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals:  true,
				DateRanges: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.SevenDay},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      apiURL,
		exchange.WebsocketSpot: hitbtcWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
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
		DefaultURL:            hitbtcWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
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
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}

	symbols, err := e.GetSymbolsDetailed(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(symbols))
	for i, s := range symbols {
		// s.QuoteCurrency is actually settlement currency, so trim the base currency to get the real quote currency
		if pairs[i], err = currency.NewPairFromStrings(s.BaseCurrency, strings.TrimPrefix(s.ID, s.BaseCurrency)); err != nil {
			return nil, err
		}
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

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := e.GetTickers(ctx)
	if err != nil {
		return err
	}

	for x := range tick {
		var pair currency.Pair
		var enabled bool
		pair, enabled, err = e.MatchSymbolCheckEnabled(tick[x].Symbol, a, false)
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
		}

		if !enabled {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick[x].Last,
			High:         tick[x].High,
			Low:          tick[x].Low,
			Bid:          tick[x].Bid,
			Ask:          tick[x].Ask,
			Volume:       tick[x].Volume,
			QuoteVolume:  tick[x].VolumeQuote,
			Open:         tick[x].Open,
			Pair:         pair,
			LastUpdated:  tick[x].Timestamp,
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              c,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	fPair, err := e.FormatExchangeCurrency(c, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := e.GetOrderbook(ctx, fPair.String(), 1000)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, c, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	resp, err := e.GetBalances(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp {
		subAccts[0].Balances.Set(resp[i].Currency, accounts.Balance{
			Total: resp[i].Available + resp[i].Reserved,
			Hold:  resp[i].Reserved,
			Free:  resp[i].Available,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	// TODO supported in v3 API
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// TODO supported in v3 API
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return e.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ts := timestampStart
	var resp []trade.Data
	limit := 1000
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = e.GetTrades(ctx,
			p.String(),
			"",
			"",
			ts.UnixMilli(),
			timestampEnd.UnixMilli(),
			int64(limit),
			0)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			if tradeData[i].Timestamp.Before(timestampStart) || tradeData[i].Timestamp.After(timestampEnd) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Quantity,
				Timestamp:    tradeData[i].Timestamp,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeData[i].Timestamp) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeData[i].Timestamp
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	err := o.Validate(e.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	if e.Websocket.IsConnected() && e.Websocket.CanUseAuthenticatedEndpoints() {
		var response *WsSubmitOrderSuccessResponse
		response, err = e.wsPlaceOrder(ctx, o.Pair, o.Side.String(), o.Amount, o.Price)
		if err != nil {
			return nil, err
		}
		orderID = response.ID
		if response.Result.CumQuantity == o.Amount {
			status = order.Filled
		}
	} else {
		var fPair currency.Pair
		fPair, err = e.FormatExchangeCurrency(o.Pair, o.AssetType)
		if err != nil {
			return nil, err
		}

		var response OrderResponse
		response, err = e.PlaceOrder(ctx,
			fPair.String(),
			o.Price,
			o.Amount,
			o.Type.Lower(),
			o.Side.Lower())
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(response.OrderNumber, 10)
		if o.Type == order.Market {
			status = order.Filled
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

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = e.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	resp, err := e.CancelAllExistingOrders(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp {
		if resp[i].Status != "canceled" {
			cancelAllOrdersResponse.Status[strconv.FormatInt(resp[i].ID, 10)] = fmt.Sprintf("Could not cancel order %v. Status: %v",
				resp[i].ID,
				resp[i].Status)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	resp, err := e.GetActiveOrderByClientOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	format, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	pair = pair.Format(format)

	var side order.Side
	side, err = order.StringToOrderSide(resp.Side)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		OrderID:  resp.ID,
		Amount:   resp.Quantity,
		Exchange: e.Name,
		Price:    resp.Price,
		Date:     resp.CreatedAt,
		Side:     side,
		Pair:     pair,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, ccy currency.Code, _, _ string) (*deposit.Address, error) {
	resp, err := e.GetDepositAddresses(ctx, ccy.String())
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: resp.Address,
		Tag:     resp.PaymentID,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: common.IsEnabled(v),
	}, err
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
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		var resp []OrderHistoryResponse
		resp, err = e.GetOpenOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(allOrders))
	for i := range allOrders {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[i].Symbol,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(allOrders[i].Side)
		if err != nil {
			return nil, err
		}
		orders[i] = order.Detail{
			OrderID:  allOrders[i].ID,
			Amount:   allOrders[i].Quantity,
			Exchange: e.Name,
			Price:    allOrders[i].Price,
			Date:     allOrders[i].CreatedAt,
			Side:     side,
			Pair:     symbol,
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

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		var resp []OrderHistoryResponse
		resp, err = e.GetOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(allOrders))
	for i := range allOrders {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(allOrders[i].Symbol,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(allOrders[i].Side)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		var status order.Status
		status, err = order.StringToOrderStatus(allOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		detail := order.Detail{
			OrderID:              allOrders[i].ID,
			Amount:               allOrders[i].Quantity,
			ExecutedAmount:       allOrders[i].CumQuantity,
			RemainingAmount:      allOrders[i].Quantity - allOrders[i].CumQuantity,
			Exchange:             e.Name,
			Price:                allOrders[i].Price,
			AverageExecutedPrice: allOrders[i].AvgPrice,
			Date:                 allOrders[i].CreatedAt,
			LastUpdated:          allOrders[i].UpdatedAt,
			Side:                 side,
			Status:               status,
			Pair:                 pair,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(e.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	return e.wsLogin(ctx)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// formatExchangeKlineInterval returns Interval to exchange formatted string
func formatExchangeKlineInterval(in kline.Interval) (string, error) {
	switch in {
	case kline.OneMin:
		return "M1", nil
	case kline.ThreeMin:
		return "M3", nil
	case kline.FiveMin:
		return "M5", nil
	case kline.FifteenMin:
		return "M15", nil
	case kline.ThirtyMin:
		return "M30", nil
	case kline.OneHour:
		return "H1", nil
	case kline.FourHour:
		return "H4", nil
	case kline.OneDay:
		return "D1", nil
	case kline.OneWeek:
		return "D7", nil
	case kline.OneMonth:
		return "1M", nil
	}
	return "", fmt.Errorf("%w %v", kline.ErrInvalidInterval, in)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	formattedInterval, err := formatExchangeKlineInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}

	data, err := e.GetCandles(ctx,
		req.RequestFormatted.String(),
		strconv.FormatUint(req.RequestLimit, 10),
		formattedInterval,
		req.Start,
		req.End)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(data))
	for x := range data {
		timeSeries[x] = kline.Candle{
			Time:   data[x].Timestamp,
			Open:   data[x].Open,
			High:   data[x].Max,
			Low:    data[x].Min,
			Close:  data[x].Close,
			Volume: data[x].Volume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	formattedInterval, err := formatExchangeKlineInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for y := range req.RangeHolder.Ranges {
		var data []ChartData
		data, err = e.GetCandles(ctx,
			req.RequestFormatted.String(),
			strconv.FormatUint(req.RequestLimit, 10),
			formattedInterval,
			req.RangeHolder.Ranges[y].Start.Time,
			req.RangeHolder.Ranges[y].End.Time)
		if err != nil {
			return nil, err
		}

		for i := range data {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   data[i].Timestamp,
				Open:   data[i].Open,
				High:   data[i].Max,
				Low:    data[i].Min,
				Close:  data[i].Close,
				Volume: data[i].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
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
	cp.Delimiter = "-to-"
	switch a {
	case asset.Spot:
		return tradeBaseURL + cp.Lower().String(), nil
	case asset.Futures:
		return tradeBaseURL + tradeFutures + cp.Lower().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
