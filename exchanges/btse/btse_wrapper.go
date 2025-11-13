package btse

import (
	"context"
	"errors"
	"fmt"
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

// Private Errors
var (
	errInvalidPairSymbol = errors.New("invalid currency pair symbol")
)

// SetDefaults sets the basic defaults for BTSE
func (e *Exchange) SetDefaults() {
	e.Name = "BTSE"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Spot {
			ps.RequestFormat.Delimiter = currency.DashDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				TickerBatching:      true,
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
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
				FundingRateFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching: true,
				TradeFetching:     true,
				Subscribe:         true,
				Unsubscribe:       true,
				GetOrders:         true,
				GetOrder:          true,
			},
			WithdrawPermissions: exchange.NoAPIWithdrawalMethods,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportsRestBatch:  true,
					SupportedViaTicker: true,
				},
			},
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
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 300,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      btseAPIURL,
		exchange.RestFutures:   btseAPIURL,
		exchange.WebsocketSpot: btseWebsocket,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
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

	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            btseWebsocket,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	m, err := e.GetMarketSummary(ctx, "", a == asset.Spot)
	if err != nil {
		return nil, err
	}
	var errs error
	pairs := make(currency.Pairs, 0, len(m))
	for _, marketInfo := range m {
		if pair, err := marketInfo.Pair(); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", marketInfo.Symbol, err))
		} else {
			pairs = append(pairs, pair)
		}
	}
	return pairs, errs
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	a := e.GetAssetTypes(false)
	for i := range a {
		pairs, err := e.FetchTradablePairs(ctx, a[i])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a[i], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	tickers, err := e.GetMarketSummary(ctx, "", a == asset.Spot)
	if err != nil {
		return err
	}
	var errs error
	for x := range tickers {
		pair, err := currency.NewPairFromString(tickers[x].Symbol)
		if err == nil {
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				Ask:          tickers[x].LowestAsk,
				Bid:          tickers[x].HighestBid,
				Low:          tickers[x].Low24Hr,
				Last:         tickers[x].Last,
				Volume:       tickers[x].Volume,
				High:         tickers[x].High24Hr,
				OpenInterest: tickers[x].OpenInterest,
				ExchangeName: e.Name,
				AssetType:    a,
			})
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	symbol, err := e.FormatSymbol(p, a)
	if err != nil {
		return nil, err
	}
	ticks, err := e.GetMarketSummary(ctx, symbol, a == asset.Spot)
	if err != nil {
		return nil, err
	}
	if len(ticks) != 1 {
		return nil, errors.New("market_summary should return 1 tick for a single ticker")
	}
	err = ticker.ProcessTicker(&ticker.Price{
		Pair:         p,
		Ask:          ticks[0].LowestAsk,
		Bid:          ticks[0].HighestBid,
		Low:          ticks[0].Low24Hr,
		Last:         ticks[0].Last,
		Volume:       ticks[0].Volume,
		High:         ticks[0].High24Hr,
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
	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}
	a, err := e.FetchOrderbook(ctx, fPair.String(), 0, 0, 0, assetType == asset.Spot)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, 0, len(a.BuyQuote))
	for x := range a.BuyQuote {
		if e.orderbookFilter(a.BuyQuote[x].Price, a.BuyQuote[x].Size) {
			continue
		}
		book.Bids = append(book.Bids, orderbook.Level{
			Price:  a.BuyQuote[x].Price,
			Amount: a.BuyQuote[x].Size,
		})
	}
	book.Asks = make(orderbook.Levels, 0, len(a.SellQuote))
	for x := range a.SellQuote {
		if e.orderbookFilter(a.SellQuote[x].Price, a.SellQuote[x].Size) {
			continue
		}
		book.Asks = append(book.Asks, orderbook.Level{
			Price:  a.SellQuote[x].Price,
			Amount: a.SellQuote[x].Size,
		})
	}
	book.Asks.SortAsks()
	book.Pair = p
	book.Exchange = e.Name
	book.Asset = assetType
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	balances, err := e.GetWalletInformation(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range balances {
		subAccts[0].Balances.Set(balances[i].Currency, accounts.Balance{
			Total: balances[i].Total,
			Hold:  balances[i].Total - balances[i].Available,
			Free:  balances[i].Available,
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

	const limit = 500
	var tradeData []Trade
	tradeData, err = e.GetTrades(ctx,
		p.String(),
		time.Time{}, time.Time{},
		0, 0, limit,
		false,
		assetType == asset.Spot)
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		side, err := order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatInt(tradeData[i].SerialID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Time.Time(),
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
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	r, err := e.CreateOrder(ctx,
		s.ClientID, 0.0,
		false,
		s.Price,
		s.Side.String(),
		s.Amount, 0, 0,
		fPair.String(),
		goodTillCancel,
		0.0,
		s.TriggerPrice,
		"",
		s.Type.String())
	if err != nil {
		return nil, err
	}

	var orderID string
	if len(r) > 0 {
		orderID = r[0].OrderID
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

	fPair, err := e.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}

	_, err = e.CancelExistingOrder(ctx, o.OrderID, fPair.String(), o.ClientOrderID)
	if err != nil {
		return err
	}

	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
// If product ID is sent, all orders of that specified market will be cancelled
// If not specified, all orders of all markets will be cancelled
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var resp order.CancelAllResponse

	fPair, err := e.FormatExchangeCurrency(orderCancellation.Pair,
		orderCancellation.AssetType)
	if err != nil {
		return resp, err
	}

	allOrders, err := e.CancelExistingOrder(ctx, "", fPair.String(), "")
	if err != nil {
		return resp, err
	}

	resp.Status = make(map[string]string)
	for x := range allOrders {
		if allOrders[x].Status == orderCancelled {
			resp.Status[allOrders[x].OrderID] = order.Cancelled.String()
		}
	}
	return resp, nil
}

func orderIntToType(i int) order.Type {
	switch i {
	case 77:
		return order.Market
	case 76:
		return order.Limit
	default:
		return order.UnknownType
	}
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	o, err := e.GetOrders(ctx, "", orderID, "")
	if err != nil {
		return nil, err
	}

	var od order.Detail
	if len(o) == 0 {
		return nil, errors.New("no orders found")
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	for i := range o {
		if o[i].OrderID != orderID {
			continue
		}

		side := order.Buy
		if strings.EqualFold(o[i].Side, order.Ask.String()) {
			side = order.Sell
		}

		od.Pair, err = currency.NewPairDelimiter(o[i].Symbol,
			format.Delimiter)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetOrderInfo unable to parse currency pair: %s\n",
				e.Name,
				err)
		}
		od.Exchange = e.Name
		od.Amount = o[i].Size
		od.OrderID = o[i].OrderID
		od.Date = o[i].Timestamp.Time()
		od.Side = side

		od.Type = orderIntToType(o[i].OrderType)

		od.Price = o[i].Price
		if od.Status, err = order.StringToOrderStatus(o[i].OrderState); err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}

		th, err := e.TradeHistory(ctx,
			"",
			time.Time{}, time.Time{},
			0, 0, 0,
			false,
			"", orderID)
		if err != nil {
			return nil, fmt.Errorf("unable to get order fills for orderID %s", orderID)
		}

		for i := range th {
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(th[i].Side)
			if err != nil {
				return nil, err
			}
			od.Trades = append(od.Trades, order.TradeHistory{
				Timestamp: th[i].Timestamp.Time(),
				TID:       th[i].TradeID,
				Price:     th[i].Price,
				Amount:    th[i].Size,
				Exchange:  e.Name,
				Side:      orderSide,
				Fee:       th[i].FeeAmount,
			})
		}
	}
	return &od, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
	address, err := e.GetWalletAddress(ctx, c.String())
	if err != nil {
		return nil, err
	}

	extractor := func(addr string) (string, string) {
		if strings.Contains(addr, ":") {
			split := strings.Split(addr, ":")
			return split[0], split[1]
		}
		return addr, ""
	}

	if len(address) == 0 {
		addressCreate, err := e.CreateWalletAddress(ctx, c.String())
		if err != nil {
			return nil, err
		}
		if len(addressCreate) != 0 {
			addr, tag := extractor(addressCreate[0].Address)
			return &deposit.Address{
				Address: addr,
				Tag:     tag,
			}, nil
		}
		return nil, errors.New("address not found")
	}
	addr, tag := extractor(address[0].Address)
	return &deposit.Address{
		Address: addr,
		Tag:     tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	amountToString := strconv.FormatFloat(withdrawRequest.Amount, 'f', 8, 64)
	resp, err := e.WalletWithdrawal(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		amountToString)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: e.Name,
		ID:   resp.WithdrawID,
	}, nil
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
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("no pair provided")
	}

	var orders []order.Detail
	for x := range req.Pairs {
		formattedPair, err := e.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		resp, err := e.GetOrders(ctx, formattedPair.String(), "", "")
		if err != nil {
			return nil, err
		}

		format, err := e.GetPairFormat(asset.Spot, false)
		if err != nil {
			return nil, err
		}

		for i := range resp {
			side := order.Buy
			if strings.EqualFold(resp[i].Side, order.Ask.String()) {
				side = order.Sell
			}

			status, err := order.StringToOrderStatus(resp[i].OrderState)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}

			p, err := currency.NewPairDelimiter(resp[i].Symbol,
				format.Delimiter)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s GetActiveOrders unable to parse currency pair: %s\n",
					e.Name,
					err)
			}

			openOrder := order.Detail{
				Pair:            p,
				Exchange:        e.Name,
				Amount:          resp[i].Size,
				ExecutedAmount:  resp[i].FilledSize,
				RemainingAmount: resp[i].Size - resp[i].FilledSize,
				OrderID:         resp[i].OrderID,
				Date:            resp[i].Timestamp.Time(),
				Side:            side,
				Price:           resp[i].Price,
				Status:          status,
				Type:            orderIntToType(resp[i].OrderType),
			}

			fills, err := e.TradeHistory(ctx,
				"",
				time.Time{}, time.Time{},
				0, 0, 0,
				false,
				"", resp[i].OrderID)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s: Unable to get order fills for orderID %s",
					e.Name,
					resp[i].OrderID)
				continue
			}

			for i := range fills {
				var orderSide order.Side
				orderSide, err = order.StringToOrderSide(fills[i].Side)
				if err != nil {
					return nil, err
				}
				openOrder.Trades = append(openOrder.Trades, order.TradeHistory{
					Timestamp: fills[i].Timestamp.Time(),
					TID:       fills[i].TradeID,
					Price:     fills[i].Price,
					Amount:    fills[i].Size,
					Exchange:  e.Name,
					Side:      orderSide,
					Fee:       fills[i].FeeAmount,
				})
			}
			orders = append(orders, openOrder)
		}
	}
	return req.Filter(e.Name, orders), nil
}

func matchType(input int, required order.Type) bool {
	if (required == order.AnyType) || (input == 76 && required == order.Limit) || input == 77 && required == order.Market {
		return true
	}
	return false
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	if len(getOrdersRequest.Pairs) == 0 {
		var err error
		getOrdersRequest.Pairs, err = e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
	}
	orderDeref := *getOrdersRequest
	for x := range orderDeref.Pairs {
		fPair, err := e.FormatExchangeCurrency(orderDeref.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		currentOrder, err := e.GetOrders(ctx, fPair.String(), "", "")
		if err != nil {
			return nil, err
		}
		for y := range currentOrder {
			if !matchType(currentOrder[y].OrderType, orderDeref.Type) {
				continue
			}
			orderStatus, err := order.StringToOrderStatus(currentOrder[y].OrderState)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(currentOrder[y].Side)
			if err != nil {
				return nil, err
			}
			tempOrder := order.Detail{
				OrderID:              currentOrder[y].OrderID,
				ClientID:             currentOrder[y].ClOrderID,
				Exchange:             e.Name,
				Price:                currentOrder[y].Price,
				AverageExecutedPrice: currentOrder[y].AverageFillPrice,
				Amount:               currentOrder[y].Size,
				ExecutedAmount:       currentOrder[y].FilledSize,
				RemainingAmount:      currentOrder[y].Size - currentOrder[y].FilledSize,
				Date:                 currentOrder[y].Timestamp.Time(),
				Side:                 orderSide,
				Status:               orderStatus,
				Pair:                 orderDeref.Pairs[x],
			}
			tempOrder.InferCostsAndTimes()
			resp = append(resp, tempOrder)
		}
	}
	return getOrdersRequest.Filter(e.Name, resp), nil
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

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval formats kline interval to exchange requested type
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Minutes(), 'f', 0, 64)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.Futures:
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	intervalInt, err := strconv.Atoi(e.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}

	candles, err := e.GetOHLCV(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End.Add(-req.ExchangeInterval.Duration()), // End time is inclusive, so we need to subtract the interval.
		intervalInt,
		a)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   time.Unix(int64(candles[x][0]), 0),
			Open:   candles[x][1],
			High:   candles[x][2],
			Low:    candles[x][3],
			Close:  candles[x][4],
			Volume: candles[x][5],
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.Futures:
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	intervalInt, err := strconv.Atoi(e.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, req.Size())
	for i := range req.RangeHolder.Ranges {
		var candles OHLCV
		candles, err = e.GetOHLCV(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[i].Start.Time,
			req.RangeHolder.Ranges[i].End.Time,
			intervalInt,
			a)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   time.Unix(int64(candles[x][0]), 0),
				Open:   candles[x][1],
				High:   candles[x][2],
				Low:    candles[x][3],
				Close:  candles[x][4],
				Volume: candles[x][5],
			}
		}
	}

	return req.ProcessResponse(timeSeries)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := e.GetCurrentServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.ISO, nil
}

// ExponentPairs returns a map of symbol names which have a Exponent equivalent
// e.g. PIT-USD will be returned if M_PIT-USD exists, and SATS-USD if K_SATS-USD exists
func (m *MarketSummary) ExponentPairs() (map[string]bool, error) {
	pairs := map[string]bool{}
	var errs error
	for _, s := range *m {
		if s.Active && s.HasLiquidity() {
			if symbol, err := s.StripExponent(); err != nil {
				errs = common.AppendError(errs, err)
			} else if symbol != "" {
				pairs[symbol] = true
			}
		}
	}
	return pairs, errs
}

// StripExponent returns the symbol without a exponent prefix; e.g. B_, M_, K_
// Returns an empty string if no exponent prefix is found
// Errors if there's too many underscores, or if the exponent is not recognised
func (m *MarketPair) StripExponent() (string, error) {
	parts := strings.Split(m.Symbol, "_")
	switch len(parts) {
	case 1:
		return "", nil
	case 2:
		switch parts[0] {
		case "B", "M", "K":
			return parts[1], nil
		}
	}
	return "", errInvalidPairSymbol
}

// Pair returns the currency Pair for a MarketPair
func (m *MarketPair) Pair() (currency.Pair, error) {
	baseCurr := m.Base
	var quoteStr string
	if m.Futures {
		if baseCurr.String() == "TRUMPSOL" { // Only base currency which is different to the rest
			baseCurr = currency.TRUMP
			quoteStr = strings.TrimPrefix(m.Symbol, baseCurr.String())
		} else {
			// Quote field is the settlement currency, create the quote currency from the symbol
			s := strings.Split(m.Symbol, m.Base.String())
			if len(s) <= 1 {
				return currency.EMPTYPAIR, errInvalidPairSymbol
			}
			quoteStr = s[1]
		}
	} else {
		s := strings.Split(m.Symbol, currency.DashDelimiter)
		if len(s) != 2 {
			return currency.EMPTYPAIR, errInvalidPairSymbol
		}
		baseCurr = currency.NewCode(s[0])
		quoteStr = s[1]
	}
	return currency.NewPair(baseCurr, currency.NewCode(quoteStr)), nil
}

// GetMarketSummary returns filtered market pair details; Specifically:
//   - Pairs which aren't active are removed
//   - Pairs which don't have liquidity are removed
//   - OTC pairs only traded on web UI are removed
//   - Pairs with an exponent counterpart pair are removed
//     BTSE lists M_ symbols for very small pairs, in millions. For those listings, we want to take the M_ listing in preference
//     to the native listing, since they're often going to appear as locked markets due to size (bid == ask, e.g. 0.0000000003)
func (e *Exchange) GetMarketSummary(ctx context.Context, symbol string, spot bool) (MarketSummary, error) {
	m, err := e.GetRawMarketSummary(ctx, symbol, spot)
	if err != nil {
		return m, err
	}
	ePairs, err := m.ExponentPairs()
	if err != nil {
		return m, err
	}
	filtered := make(MarketSummary, 0, len(m))
	for _, l := range m {
		if !l.Active || !l.HasLiquidity() || (spot && !l.IsMarketOpenToSpot) { // Skip OTC assets only tradable on web UI
			continue
		}
		if ePairs[l.Symbol] { // Skip pair with an exponent sibling
			continue
		}
		if !spot {
			// BTSE API for futures does not return futures field at all, and the docs show it coming back as false
			// Much easier for our data flow if we can trust this field
			l.Futures = true
		}
		filtered = append(filtered, l)
	}
	return filtered, nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	marketSummary, err := e.GetMarketSummary(ctx, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(marketSummary))
	for i := range marketSummary {
		// Quote field is the settlement currency, create the quote currency from the symbol
		quote := currency.NewCode(marketSummary[i].Symbol[len(marketSummary[i].Base.String()):])
		cp := currency.NewPair(marketSummary[i].Base, quote)
		startTime := marketSummary[i].OpenTime.Time()
		endTime := marketSummary[i].CloseTime.Time()
		ct := futures.Perpetual
		if marketSummary[i].TimeBasedContract {
			if endTime.Sub(startTime) > kline.OneMonth.Duration() {
				ct = futures.Quarterly
			} else {
				ct = futures.Monthly
			}
		}
		contractSettlementType := futures.LinearOrInverse
		if marketSummary[i].AvailableSettlement.Contains(currency.USD) {
			contractSettlementType = futures.Linear
		}
		var rate fundingrate.Rate
		if marketSummary[i].FundingRate > 0 {
			rate = fundingrate.Rate{
				Rate: decimal.NewFromFloat(marketSummary[i].FundingRate),
				Time: time.Now().Truncate(time.Hour),
			}
		}
		resp[i] = futures.Contract{
			Exchange:                       e.Name,
			Name:                           cp,
			Underlying:                     currency.NewPair(marketSummary[i].Base, marketSummary[i].Quote),
			Asset:                          item,
			SettlementCurrency:             currency.USDT,
			AdditionalSettlementCurrencies: marketSummary[i].AvailableSettlement,
			StartDate:                      startTime,
			EndDate:                        endTime,
			SettlementType:                 contractSettlementType,
			IsActive:                       marketSummary[i].Active,
			Type:                           ct,
			LatestRate:                     rate,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}

	format, err := e.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := format.Format(r.Pair)
	rates, err := e.GetMarketSummary(ctx, fPair, false)
	if err != nil {
		return nil, err
	}

	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	for i := range rates {
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = e.MatchSymbolCheckEnabled(rates[i].Symbol, r.Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		tt := time.Now().Truncate(time.Hour)
		resp = append(resp, fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: time.Now().Truncate(time.Hour),
				Rate: decimal.NewFromFloat(rates[i].FundingRate),
			},
			TimeOfNextRate: tt.Add(time.Hour),
			TimeChecked:    time.Now(),
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	return a == asset.Futures && p.Quote.Equal(currency.PFC), nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	summary, err := e.GetMarketSummary(ctx, "", a == asset.Spot)
	if err != nil {
		return err
	}
	var errs error
	l := make([]limits.MinMaxLevel, 0, len(summary))
	for _, marketInfo := range summary {
		p, err := marketInfo.Pair()
		if err != nil {
			errs = common.AppendError(err, fmt.Errorf("%s: %w", p, err))
			continue
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, p),
			MinimumBaseAmount:       marketInfo.MinOrderSize,
			MaximumBaseAmount:       marketInfo.MaxOrderSize,
			AmountStepIncrementSize: marketInfo.MinSizeIncrement,
			MinPrice:                marketInfo.MinValidPrice,
			PriceStepIncrementSize:  marketInfo.MinPriceIncrement,
		})
	}
	if err = limits.Load(l); err != nil {
		errs = common.AppendError(errs, err)
	}
	return errs
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	tickers, err := e.GetMarketSummary(ctx, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(tickers))
	for i := range tickers {
		var symbol currency.Pair
		var enabled bool
		symbol, enabled, err = e.MatchSymbolCheckEnabled(tickers[i].Symbol, asset.Futures, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !enabled {
			continue
		}
		var appendData bool
		for j := range k {
			if k[j].Pair().Equal(symbol) {
				appendData = true
				break
			}
		}
		if len(k) > 0 && !appendData {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, symbol),
			OpenInterest: tickers[i].OpenInterest,
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
