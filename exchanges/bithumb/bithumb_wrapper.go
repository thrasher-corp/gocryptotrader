package bithumb

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
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

var errNotEnoughPairs = errors.New("at least one currency is required to fetch order history")

// SetDefaults sets the basic defaults for Bithumb
func (e *Exchange) SetDefaults() {
	e.Name = "Bithumb"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.location, err = time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Errorf(log.ExchangeSys, "Bithumb unable to load time location: %s", err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoWithdrawal:    true,
				FiatDeposit:         true,
				FiatWithdraw:        true,
				GetOrder:            true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				TradeFee:            true,
				FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
				KlineFetching:       true,
			},
			Websocket: true,
			WebsocketCapabilities: protocol.Features{
				TradeFetching:     true,
				TickerFetching:    true,
				OrderbookFetching: true,
				Subscribe:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					// NOTE: The supported time intervals below are returned
					// offset to the Asia/Seoul time zone. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. Therefore the below intervals
					// are constructed from hourly candles.
					// kline.IntervalCapacity{Interval: kline.SixHour},
					// kline.IntervalCapacity{Interval: kline.TwelveHour},
					// kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 1500,
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
		exchange.WebsocketSpot: wsEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup takes in the supplied exchange configuration details and sets params
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

	ePoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            wsEndpoint,
		RunningURL:            ePoint,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(time.Second),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	currencies, err := e.GetTradablePairs(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(currencies))
	for x := range currencies {
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(currencies[x], "KRW")
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
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
	tickers, err := e.GetAllTickers(ctx)
	if err != nil {
		return err
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	for i := range pairs {
		curr := pairs[i].Base.String()
		t, ok := tickers[curr]
		if !ok {
			return fmt.Errorf("enabled pair %s [%s] not found in returned ticker map %v",
				pairs[i], pairs, tickers)
		}
		p, err := e.FormatExchangeCurrency(pairs[i], a)
		if err != nil {
			return err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			High:         t.MaxPrice,
			Low:          t.MinPrice,
			Volume:       t.UnitsTraded24Hr,
			Open:         t.OpeningPrice,
			Close:        t.ClosingPrice,
			Pair:         p,
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
	curr := p.Base.String()

	orderbookNew, err := e.GetOrderBook(ctx, curr)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Data.Bids))
	for i := range orderbookNew.Data.Bids {
		book.Bids[i] = orderbook.Level{
			Amount: orderbookNew.Data.Bids[i].Quantity,
			Price:  orderbookNew.Data.Bids[i].Price,
		}
	}

	book.Asks = make(orderbook.Levels, len(orderbookNew.Data.Asks))
	for i := range orderbookNew.Data.Asks {
		book.Asks[i] = orderbook.Level{
			Amount: orderbookNew.Data.Asks[i].Quantity,
			Price:  orderbookNew.Data.Asks[i].Price,
		}
	}

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	bal, err := e.GetAccountBalance(ctx, "ALL")
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for k, totalAmount := range bal.Total {
		hold, ok := bal.InUse[k]
		if !ok {
			return subAccts, fmt.Errorf("currency %s missing from InUse balances", k)
		}
		avail, ok := bal.Available[k]
		if !ok {
			avail = totalAmount - hold
		}
		subAccts[0].Balances.Set(currency.NewCode(k), accounts.Balance{
			Total: totalAmount,
			Hold:  hold,
			Free:  avail,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	transactions, err := e.GetUserTransactions(ctx, 0, 0, 3, c, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(transactions.Data))
	for i := range transactions.Data {
		resp[i] = exchange.WithdrawalHistory{
			Timestamp: transactions.Data[i].TransferDate.Time(),
			Currency:  transactions.Data[i].OrderCurrency.String(),
			Amount:    transactions.Data[i].Amount,
			Fee:       transactions.Data[i].Fee,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tradeData, err := e.GetTransactionHistory(ctx, p.String())
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData.Data))
	for i := range tradeData.Data {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData.Data[i].Type)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData.Data[i].Price,
			Amount:       tradeData.Data[i].UnitsTraded,
			Timestamp:    tradeData.Data[i].TransactionDate.Time(),
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
// TODO: Fill this out to support limit orders
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	var orderID string
	if s.Side.IsLong() {
		var result MarketBuy
		result, err = e.MarketBuyOrder(ctx, fPair, s.Amount)
		if err != nil {
			return nil, err
		}
		orderID = result.OrderID
	} else if s.Side.IsShort() {
		var result MarketSell
		result, err = e.MarketSellOrder(ctx, fPair, s.Amount)
		if err != nil {
			return nil, err
		}
		orderID = result.OrderID
	}
	return s.DeriveSubmitResponse(orderID)
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

	_, err := e.CancelTrade(ctx, o.Side.String(), o.OrderID, o.Pair.Base.String())
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	var allOrders []OrderData
	currs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range currs {
		orders, err := e.GetOrders(ctx, "", orderCancellation.Side.String(), 100, time.Time{}, currs[i].Base, currency.EMPTYCODE)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		allOrders = append(allOrders, orders.Data...)
	}

	for i := range allOrders {
		_, err := e.CancelTrade(ctx,
			orderCancellation.Side.String(),
			allOrders[i].OrderID,
			orderCancellation.Pair.Base.String())
		if err != nil {
			cancelAllOrdersResponse.Status[allOrders[i].OrderID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, _ asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	orders, err := e.GetOrders(ctx, orderID, "", 0, time.Time{}, pair.Base, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	for i := range orders.Data {
		if orders.Data[i].OrderID != orderID {
			continue
		}
		orderDetail := order.Detail{
			Amount:          orders.Data[i].Units,
			Exchange:        e.Name,
			ExecutedAmount:  orders.Data[i].Units - orders.Data[i].UnitsRemaining,
			OrderID:         orders.Data[i].OrderID,
			Date:            orders.Data[i].OrderDate.Time(),
			Price:           orders.Data[i].Price,
			RemainingAmount: orders.Data[i].UnitsRemaining,
			Pair:            pair,
		}

		switch orders.Data[i].Type {
		case "bid":
			orderDetail.Side = order.Buy
		case "ask":
			orderDetail.Side = order.Sell
		}

		return &orderDetail, nil
	}
	return nil, fmt.Errorf("%w %v", order.ErrOrderNotFound, orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addr, err := e.GetWalletAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: addr.Data.WalletAddress,
		Tag:     addr.Data.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.WithdrawCrypto(ctx,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Currency.String(),
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.Message,
		Status: v.Status,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if math.Trunc(withdrawRequest.Amount) != withdrawRequest.Amount {
		return nil, errors.New("currency KRW does not support decimal places")
	}
	if !withdrawRequest.Currency.Equal(currency.KRW) {
		return nil, fmt.Errorf("only KRW supported, received '%v'", withdrawRequest.Currency)
	}
	bankDetails := strconv.FormatFloat(withdrawRequest.Fiat.Bank.BankCode, 'f', -1, 64) +
		"_" + withdrawRequest.Fiat.Bank.BankName
	resp, err := e.RequestKRWWithdraw(ctx,
		bankDetails,
		withdrawRequest.Fiat.Bank.AccountNumber,
		int64(withdrawRequest.Amount))
	if err != nil {
		return nil, err
	}
	if resp.Status != "0000" {
		return nil, errors.New(resp.Message)
	}

	return &withdraw.ExchangeResponse{
		Status: resp.Status,
	}, nil
}

// WithdrawFiatFundsToInternationalBank is not supported as Bithumb only withdraws KRW to South Korean banks
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

	if len(req.Pairs) == 0 {
		return nil, errNotEnoughPairs
	}

	format, err := e.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for x := range req.Pairs {
		var resp Orders
		resp, err = e.GetOrders(ctx, "", "", 1000, time.Time{}, req.Pairs[x].Base, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data {
			if resp.Data[i].Status != "placed" {
				continue
			}

			orderDetail := order.Detail{
				Amount:          resp.Data[i].Units,
				Exchange:        e.Name,
				ExecutedAmount:  resp.Data[i].Units - resp.Data[i].UnitsRemaining,
				OrderID:         resp.Data[i].OrderID,
				Date:            resp.Data[i].OrderDate.Time(),
				Price:           resp.Data[i].Price,
				RemainingAmount: resp.Data[i].UnitsRemaining,
				Status:          order.Active,
				Pair: currency.NewPairWithDelimiter(resp.Data[i].OrderCurrency,
					resp.Data[i].PaymentCurrency,
					format.Delimiter),
			}

			switch resp.Data[i].Type {
			case "bid":
				orderDetail.Side = order.Buy
			case "ask":
				orderDetail.Side = order.Sell
			}

			orders = append(orders, orderDetail)
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
		return nil, errNotEnoughPairs
	}

	format, err := e.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for x := range req.Pairs {
		var resp Orders
		resp, err = e.GetOrders(ctx, "", "", 1000, time.Time{}, req.Pairs[x].Base, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data {
			if resp.Data[i].Status == "placed" {
				continue
			}

			orderDetail := order.Detail{
				Amount:          resp.Data[i].Units,
				ExecutedAmount:  resp.Data[i].Units - resp.Data[i].UnitsRemaining,
				RemainingAmount: resp.Data[i].UnitsRemaining,
				Exchange:        e.Name,
				OrderID:         resp.Data[i].OrderID,
				Date:            resp.Data[i].OrderDate.Time(),
				Price:           resp.Data[i].Price,
				Pair: currency.NewPairWithDelimiter(resp.Data[i].OrderCurrency,
					resp.Data[i].PaymentCurrency,
					format.Delimiter),
			}

			switch resp.Data[i].Type {
			case "bid":
				orderDetail.Side = order.Buy
			case "ask":
				orderDetail.Side = order.Sell
			}

			orderDetail.InferCostsAndTimes()
			orders = append(orders, orderDetail)
		}
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
	return in.Short()
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	candles, err := e.GetCandleStick(ctx, req.RequestFormatted.String(), e.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, len(candles.Data))
	for x := range candles.Data {
		if candles.Data[x].Timestamp.Time().Before(req.Start) {
			continue
		}
		if candles.Data[x].Timestamp.Time().After(req.End) {
			break
		}
		timeSeries = append(timeSeries, kline.Candle{
			Time:   candles.Data[x].Timestamp.Time(),
			Open:   candles.Data[x].Open.Float64(),
			High:   candles.Data[x].High.Float64(),
			Low:    candles.Data[x].Low.Float64(),
			Close:  candles.Data[x].Close.Float64(),
			Volume: candles.Data[x].Volume.Float64(),
		})
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.CurrencyPairs.IsAssetSupported(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	l, err := e.FetchExchangeLimits(ctx)
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return limits.Load(l)
}

// UpdateCurrencyStates updates currency states for exchange
func (e *Exchange) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	status, err := e.GetAssetStatusAll(ctx)
	if err != nil {
		return err
	}

	payload := make(map[currency.Code]currencystate.Options)
	for coin, options := range status.Data {
		payload[currency.NewCode(coin)] = currencystate.Options{
			Withdraw: convert.BoolPtr(options.WithdrawalStatus == 1),
			Deposit:  convert.BoolPtr(options.DepositStatus == 1),
		}
	}
	return e.States.UpdateAll(a, payload)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}
