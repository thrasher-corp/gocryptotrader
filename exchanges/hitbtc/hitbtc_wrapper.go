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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := h.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	h.Features = exchange.Features{
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

	h.Requester, err = request.New(h.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.API.Endpoints = h.NewEndpoints()
	err = h.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      apiURL,
		exchange.WebsocketSpot: hitbtcWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.Websocket = websocket.NewManager()
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}
	err = h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := h.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            hitbtcWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             h.WsConnect,
		Subscriber:            h.Subscribe,
		Unsubscriber:          h.Unsubscribe,
		GenerateSubscriptions: h.generateSubscriptions,
		Features:              &h.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return h.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HitBTC) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	symbols, err := h.GetSymbolsDetailed(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(symbols))
	for x := range symbols {
		index := strings.Index(symbols[x].ID, symbols[x].QuoteCurrency)
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(symbols[x].ID[:index], symbols[x].ID[index:])
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HitBTC) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := h.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = h.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return h.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (h *HitBTC) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := h.GetTickers(ctx)
	if err != nil {
		return err
	}

	for x := range tick {
		var pair currency.Pair
		var enabled bool
		pair, enabled, err = h.MatchSymbolCheckEnabled(tick[x].Symbol, a, false)
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
			ExchangeName: h.Name,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := h.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(h.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := h.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          h.Name,
		Pair:              c,
		Asset:             assetType,
		ValidateOrderbook: h.ValidateOrderbook,
	}
	fPair, err := h.FormatExchangeCurrency(c, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := h.GetOrderbook(ctx, fPair.String(), 1000)
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
	return orderbook.Get(h.Name, c, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = h.Name
	accountBalance, err := h.GetBalances(ctx)
	if err != nil {
		return response, err
	}

	currencies := make([]account.Balance, 0, len(accountBalance))
	for i := range accountBalance {
		currencies = append(currencies, account.Balance{
			Currency: currency.NewCode(accountBalance[i].Currency),
			Total:    accountBalance[i].Available + accountBalance[i].Reserved,
			Hold:     accountBalance[i].Reserved,
			Free:     accountBalance[i].Available,
		})
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: currencies,
	})

	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (h *HitBTC) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	// TODO supported in v3 API
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (h *HitBTC) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// TODO supported in v3 API
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (h *HitBTC) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return h.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (h *HitBTC) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = h.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ts := timestampStart
	var resp []trade.Data
	limit := 1000
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = h.GetTrades(ctx,
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
				Exchange:     h.Name,
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

	err = h.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	err := o.Validate(h.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	if h.Websocket.IsConnected() && h.Websocket.CanUseAuthenticatedEndpoints() {
		var response *WsSubmitOrderSuccessResponse
		response, err = h.wsPlaceOrder(ctx, o.Pair, o.Side.String(), o.Amount, o.Price)
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(response.ID, 10)
		if response.Result.CumQuantity == o.Amount {
			status = order.Filled
		}
	} else {
		var fPair currency.Pair
		fPair, err = h.FormatExchangeCurrency(o.Pair, o.AssetType)
		if err != nil {
			return nil, err
		}

		var response OrderResponse
		response, err = h.PlaceOrder(ctx,
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

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HitBTC) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HitBTC) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (h *HitBTC) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (h *HitBTC) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HitBTC) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	resp, err := h.CancelAllExistingOrders(ctx)
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
func (h *HitBTC) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := h.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	resp, err := h.GetActiveOrderByClientOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	format, err := h.GetPairFormat(assetType, true)
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
		Exchange: h.Name,
		Price:    resp.Price,
		Date:     resp.CreatedAt,
		Side:     side,
		Pair:     pair,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HitBTC) GetDepositAddress(ctx context.Context, currency currency.Code, _, _ string) (*deposit.Address, error) {
	resp, err := h.GetDepositAddresses(ctx, currency.String())
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
func (h *HitBTC) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := h.Withdraw(ctx,
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
func (h *HitBTC) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !h.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HitBTC) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
		resp, err = h.GetOpenOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := h.GetPairFormat(asset.Spot, false)
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
			Exchange: h.Name,
			Price:    allOrders[i].Price,
			Date:     allOrders[i].CreatedAt,
			Side:     side,
			Pair:     symbol,
		}
	}
	return req.Filter(h.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HitBTC) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
		resp, err = h.GetOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := h.GetPairFormat(asset.Spot, false)
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
			log.Errorf(log.ExchangeSys, "%s %v", h.Name, err)
		}
		var status order.Status
		status, err = order.StringToOrderStatus(allOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", h.Name, err)
		}
		detail := order.Detail{
			OrderID:              allOrders[i].ID,
			Amount:               allOrders[i].Quantity,
			ExecutedAmount:       allOrders[i].CumQuantity,
			RemainingAmount:      allOrders[i].Quantity - allOrders[i].CumQuantity,
			Exchange:             h.Name,
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
	return req.Filter(h.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (h *HitBTC) AuthenticateWebsocket(ctx context.Context) error {
	return h.wsLogin(ctx)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (h *HitBTC) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := h.UpdateAccountInfo(ctx, assetType)
	return h.CheckTransientError(err)
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
func (h *HitBTC) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := h.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	formattedInterval, err := formatExchangeKlineInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}

	data, err := h.GetCandles(ctx,
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
func (h *HitBTC) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := h.GetKlineExtendedRequest(pair, a, interval, start, end)
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
		data, err = h.GetCandles(ctx,
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
func (h *HitBTC) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (h *HitBTC) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (h *HitBTC) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (h *HitBTC) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := h.CurrencyPairs.IsPairEnabled(cp, a)
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
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
