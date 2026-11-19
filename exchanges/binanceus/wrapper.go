package binanceus

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
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

// SetDefaults sets the basic defaults for Binance.US
func (e *Exchange) SetDefaults() {
	e.Name = "Binanceus"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	if err := e.SetAssetPairStore(asset.Spot, currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, asset.Spot, err)
	}

	e.Features = exchange.Features{
		Subscriptions: defaultSubscriptions.Clone(),
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				TradeFee:              true,
				CryptoDepositFee:      true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				TradeFetching:          true,
				KlineFetching:          true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
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
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   apiURL,
		exchange.RestSpotSupplementary:      apiURL,
		exchange.WebsocketSpot:              websocketStreamsURL,
		exchange.WebsocketSpotSupplementary: websocketStreamsURL,
		exchange.WebsocketTrade:             websocketAPIURL,
	}); err != nil {
		log.Errorf(log.ExchangeSys, "%s setting default endpoints error %v", e.Name, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
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
	if err := e.Websocket.Setup(e.websocketManagerSetup(exch)); err != nil {
		return err
	}
	e.orderbookSync = e.newOrderbookSync(e.fetchOrderbookSnapshot, orderbookmanager.DefaultWSOrderbookUpdateTimeDelay, orderbookmanager.DefaultWSOrderbookUpdateDeadline)
	streamsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	streams := e.streamConnectionSetup(exch)
	streams.URL = streamsURL
	if err := e.Websocket.SetupNewConnection(streams); err != nil {
		return err
	}
	// A WebSocket API connection kept from an earlier setup must not serve requests once this one connects
	e.wsAPI.set(nil, 0)
	// The WebSocket API connection carries the User Data Stream and the WebsocketAPI methods. The websocket fails to
	// connect when any of its connections does, so the connection is set up only with authenticated websocket support
	// enabled, leaving the market data streams independent of the WebSocket API otherwise
	if !e.IsWebsocketAuthenticationSupported() {
		return nil
	}
	apiURL, err := e.API.Endpoints.GetURL(exchange.WebsocketTrade)
	if err != nil {
		return err
	}
	api := e.websocketAPIConnectionSetup(exch)
	api.URL = apiURL
	return e.Websocket.SetupNewConnection(api)
}

var errInvalidOrderID = errors.New("invalid order ID")

// parseOrderID converts an order ID into Binance.US's numeric form; an empty ID is 0, which the endpoints leave out
func parseOrderID(orderID string) (uint64, error) {
	if orderID == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidOrderID, orderID, err)
	}
	return id, nil
}

// orderListIDPrefix starts the order ID of each OCO order list, as Binance.US numbers order lists apart from orders, so
// that an ID says which of the two it names
const orderListIDPrefix = "list-"

// formatOrderListID returns the order ID of the OCO order list with ID orderListID
func formatOrderListID(orderListID uint64) string {
	return orderListIDPrefix + strconv.FormatUint(orderListID, 10)
}

// parseSpotOrderID converts an order ID into Binance.US's numeric form, an OCO order list's into its order list ID, and
// reports whether it is an order list's
func parseSpotOrderID(orderID string) (id uint64, isList bool, err error) {
	listID, isList := strings.CutPrefix(orderID, orderListIDPrefix)
	if !isList {
		id, err = parseOrderID(orderID)
		return id, false, err
	}
	if id, err = strconv.ParseUint(listID, 10, 64); err != nil {
		return 0, true, fmt.Errorf("%w %q: %w", errInvalidOrderID, orderID, err)
	}
	return id, true, nil
}

// remainingAmount returns an order's unfilled amount, in decimal arithmetic so it is as exact as Binance.US's amounts.
// Quantity self-trade prevention has prevented can no longer fill, so an open or filled order leaves it out, while a
// cancelled or expired order reports all it did not execute
func remainingAmount(original, executed, prevented types.Number, status order.Status) float64 {
	remaining := original.Decimal().Sub(executed.Decimal())
	if status == order.Filled || !status.IsInactive() {
		remaining = remaining.Sub(prevented.Decimal())
	}
	return remaining.InexactFloat64()
}

// averagePrice returns the average price of an order's fills from the quote amount and quantity they total, in decimal
// arithmetic so it is as exact as Binance.US's amounts
func averagePrice(cumulativeQuote, executed types.Number) float64 {
	if executed <= 0 {
		return 0
	}
	return cumulativeQuote.Decimal().Div(executed.Decimal()).InexactFloat64()
}

// spotOrderTypeString returns the order type of t. A post only limit order is a LIMIT_MAKER order, Binance.US's post
// only type
func spotOrderTypeString(t order.Type, tif order.TimeInForce) (OrderType, error) {
	switch t {
	case order.Limit:
		if tif.Is(order.PostOnly) {
			return OrderTypeLimitMaker, nil
		}
		return OrderTypeLimit, nil
	case order.LimitMaker:
		return OrderTypeLimitMaker, nil
	case order.Market:
		return OrderTypeMarket, nil
	case order.StopMarket:
		return OrderTypeStopLoss, nil
	case order.StopLimit:
		return OrderTypeStopLossLimit, nil
	case order.TakeProfitMarket:
		return OrderTypeTakeProfit, nil
	case order.TakeProfitLimit:
		return OrderTypeTakeProfitLimit, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// stringToSpotOrderType converts an order type, the inverse of spotOrderTypeString; STOP_LOSS and TAKE_PROFIT orders
// become market orders when they trigger
func stringToSpotOrderType(orderType OrderType) (order.Type, error) {
	switch orderType {
	case OrderTypeLimit:
		return order.Limit, nil
	case OrderTypeMarket:
		return order.Market, nil
	case OrderTypeStopLoss:
		return order.StopMarket, nil
	case OrderTypeStopLossLimit:
		return order.StopLimit, nil
	case OrderTypeTakeProfit:
		return order.TakeProfitMarket, nil
	case OrderTypeTakeProfitLimit:
		return order.TakeProfitLimit, nil
	case OrderTypeLimitMaker:
		return order.LimitMaker, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// spotTimeInForce returns the time in force of an order of type t. Binance.US reports a LIMIT_MAKER order, its post only
// type, as good till cancel, which reads as post only, as SubmitOrder reports it
func spotTimeInForce(t order.Type, tif order.TimeInForce) order.TimeInForce {
	if t == order.LimitMaker {
		return order.PostOnly
	}
	return tif
}

// oneCancelsTheOtherContingencyType is the contingency type of OCO order lists
const oneCancelsTheOtherContingencyType = "OCO"

// orderListStatus converts an OCO order list's listOrderStatus
func orderListStatus(status string) (order.Status, error) {
	switch status {
	case "EXECUTING":
		return order.Active, nil
	case "ALL_DONE":
		return order.Closed, nil
	case "REJECT":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, status)
	}
}

// orderListDetail converts an OCO order list, which GCT tracks as one order identified by its order list ID, tagged as
// an order list's; the list queries report neither the side nor the amounts of its orders. Values that cannot be
// converted are left unset and returned as the error
func (e *Exchange) orderListDetail(orderListID uint64, listClientOrderID, listOrderStatus string, transactionTime time.Time, pair currency.Pair) (order.Detail, error) {
	status, err := orderListStatus(listOrderStatus)
	return order.Detail{
		Exchange:      e.Name,
		OrderID:       formatOrderListID(orderListID),
		ClientOrderID: listClientOrderID,
		Type:          order.OCO,
		Status:        status,
		AssetType:     asset.Spot,
		Date:          transactionTime,
		LastUpdated:   transactionTime,
		Pair:          pair,
	}, err
}

// FetchTradablePairs returns the symbols that trade on Binance.US's spot market
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	info, err := e.GetExchangeInfo(ctx, nil)
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, 0, len(info.Symbols))
	for i := range info.Symbols {
		if s := &info.Symbols[i]; s.Status == string(SymbolStatusTrading) && s.IsSpotTradingAllowed {
			pairs = append(pairs, currency.NewPair(s.BaseAsset, s.QuoteAsset))
		}
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	return pairs.Format(format), nil
}

// UpdateTradablePairs updates the exchange's available pairs and stores them in the exchange's config
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

// UpdateTickers updates the tickers of every available pair from one request for every symbol's 24hr ticker
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	// A request without symbols returns every ticker, at the weight of one naming 101 or more symbols
	stats, err := e.GetPriceChangeStats(ctx, &PriceChangeStatsRequest{})
	if err != nil {
		return err
	}
	prices := make([]ticker.Price, 0, len(stats))
	for i := range stats {
		pair, ok, err := e.tickerPair(stats[i].Symbol)
		if err != nil {
			return err
		}
		if ok {
			prices = append(prices, e.spotTickerPrice(&stats[i], pair))
		}
	}
	_, err = ticker.ProcessBatch(prices)
	return err
}

// tickerPair returns the available pair of a ticker's symbol, which the ticker is stored under, and false for a symbol
// without one: ticker lists include symbols that are not trading
func (e *Exchange) tickerPair(symbol string) (currency.Pair, bool, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Spot, false)
	switch {
	case err == nil:
		return pair, true, nil
	case errors.Is(err, currency.ErrPairNotFound):
		return currency.EMPTYPAIR, false, nil
	default:
		return currency.EMPTYPAIR, false, err
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	stats, err := e.GetPriceChangeStats(ctx, &PriceChangeStatsRequest{Symbols: currency.Pairs{p}})
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
	}
	price := e.spotTickerPrice(&stats[0], p)
	if err := ticker.ProcessTicker(&price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// spotTickerPrice converts a 24hr ticker. Close is the previous close, as the ticker stream reports it
func (e *Exchange) spotTickerPrice(t *PriceChangeStatsResponse, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                       t.LastPrice.Float64(),
		LastSize:                   t.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: t.WeightedAveragePrice.Float64(),
		High:                       t.HighPrice.Float64(),
		Low:                        t.LowPrice.Float64(),
		Bid:                        t.BidPrice.Float64(),
		BidSize:                    t.BidQuantity.Float64(),
		Ask:                        t.AskPrice.Float64(),
		AskSize:                    t.AskQuantity.Float64(),
		BaseVolume:                 t.Volume.Float64(),
		QuoteVolume:                t.QuoteVolume.Float64(),
		Open:                       t.OpenPrice.Float64(),
		PercentChange24Hour:        t.PriceChangePercent.Float64(),
		Close:                      t.PreviousClosePrice.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  asset.Spot,
		LastUpdated:                t.CloseTime.Time(),
	}
}

// websocketBookFreshness is how recently before UpdateOrderbook makes its request the websocket must have updated a book
// that an order book stream maintains for the REST snapshot to leave it in place: replacing a book its diff depth stream
// maintains makes the stream's next event fetch a new snapshot
const websocketBookFreshness = 15 * time.Second

// UpdateOrderbook updates and returns the orderbook for a currency pair. While an order book stream maintains the pair's
// book, a book the websocket updated within websocketBookFreshness before the request was made, or since, which the
// REST snapshot may predate, is left in place and returned instead of the REST snapshot
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	requested := time.Now()
	resp, err := e.GetOrderBook(ctx, &OrderBookRequest{Symbol: p, Limit: 1000})
	if err != nil {
		return nil, err
	}
	// The order book has no timestamp, so processing stamps it
	book := &orderbook.Book{
		Bids:              resp.Bids.Levels(),
		Asks:              resp.Asks.Levels(),
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		LastUpdateID:      resp.LastUpdateID,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	if e.orderbookStreamed(p, a) {
		book.DeferToWebsocketWithin = websocketBookFreshness + time.Since(requested)
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances. For the configured account, whose User Data Stream the WebSocket
// API connection subscribes, a currency the stream reported at or after the account update time the snapshot shows
// keeps its stored balance, as keepStreamedSpotBalances explains, and a snapshot older than the latest one stored is not
// stored, with the stored balances returned in its place
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, assetType)
	}
	resp, err := e.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	subAccount := accounts.NewSubAccount(assetType, "")
	for i := range resp.Balances {
		free := resp.Balances[i].Free.InexactFloat64()
		locked := resp.Balances[i].Locked.InexactFloat64()
		subAccount.Balances.Set(resp.Balances[i].Asset, accounts.Balance{
			Total: free + locked,
			Hold:  locked,
			Free:  free,
		})
	}
	subAccts := accounts.SubAccounts{subAccount}
	asOf := resp.UpdateTime.Time()
	var stale bool
	if err := e.withBalanceStore(func() error {
		if stale = e.isStaleSpotSnapshot(ctx, asOf); stale {
			return nil
		}
		ordered := e.keepStreamedSpotBalances(ctx, subAccount, asOf)
		if err := e.Accounts.Save(ctx, subAccts, true); err != nil {
			return err
		}
		if ordered {
			e.recordBalanceSnapshot(asOf)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if !stale {
		return subAccts, nil
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return e.Accounts.SubAccounts(creds, asset.Spot)
}

// isStaleSpotSnapshot reports whether a REST snapshot of the configured account, which shows its account update time
// asOf, is older than the latest one stored: polls overlapping each other can be answered out of order, and the older
// one must not replace the newer one's balances
func (e *Exchange) isStaleSpotSnapshot(ctx context.Context, asOf time.Time) bool {
	if asOf.IsZero() {
		return false
	}
	creds, err := e.GetCredentials(ctx)
	return err == nil && e.isConfiguredAccount(creds) && asOf.Before(e.balanceSnapshotTime())
}

// GetRecentTrades returns the most recent trades for a currency pair
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	trades, err := e.GetMostRecentTrades(ctx, &RecentTradeRequest{Symbol: p, Limit: 1000})
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for i, t := range trades {
		resp[i] = trade.Data{
			TID:          strconv.FormatUint(t.ID, 10),
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         tradeTakerSide(t.IsBuyerMaker),
			Price:        t.Price.Float64(),
			Amount:       t.Quantity.Float64(),
			Timestamp:    t.Time.Time(),
		}
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// tradeTakerSide returns the side of a trade's taker, which sold when the buyer made the market
func tradeTakerSide(isBuyerMaker bool) order.Side {
	if isBuyerMaker {
		return order.Sell
	}
	return order.Buy
}

// GetHistoricTrades returns the aggregate trades of a currency pair from start to end
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	// A limit no window reaches pages through the aggregate trades until the end of the window
	trades, err := e.GetAggregatedTrades(ctx, &AggregatedTradeRequest{
		Symbol:    p,
		StartTime: start,
		EndTime:   end,
		Limit:     math.MaxUint64,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching %s aggregate trades: %w", p, err)
	}
	resp := make([]trade.Data, len(trades))
	for i, t := range trades {
		resp[i] = trade.Data{
			TID:          strconv.FormatUint(t.AggregateTradeID, 10),
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         tradeTakerSide(t.IsBuyerMaker),
			Price:        t.Price.Float64(),
			Amount:       t.Quantity.Float64(),
			Timestamp:    t.Timestamp.Time(),
		}
	}
	return resp, nil
}

var (
	errAmbiguousOrderListID       = errors.New("ID names both an order and an order list")
	errOrderListIDRequiresOCOType = errors.New("an order list ID pages order lists, which only the OCO type lists")
	errTriggerPriceRequired       = errors.New("trigger price is required")
	errTradeFeeNotFound           = errors.New("trade fee not found")
	errQuoteAmountWithAmount      = errors.New("an order takes a base amount or a quote amount, not both")
	errOrderOptionUnsupported     = errors.New("order option not supported")
	errNegativeStopLimitPrice     = errors.New("stop limit price cannot be negative")
)

// canUseWebsocketAPIForOrders reports whether authenticated order requests can go through the WebSocket API: not once it
// refused the configured account's key, nor for other credentials, as a refusal closes the connection all requests share
func (e *Exchange) canUseWebsocketAPIForOrders(ctx context.Context) bool {
	creds, err := e.GetCredentials(ctx)
	return err == nil && e.isConfiguredAccount(creds) && !e.websocketAPIKeyRejected.Load() && e.WebsocketAPI().IsConnected() && e.Websocket.CanUseAuthenticatedWebsocketForWrapper()
}

// overWebsocketAPI reports whether a request naming an order or order list by its ID, byID, or by its client ID,
// byClientID, goes through the WebSocket API. The WebSocket API uses one ID and ignores the other when both are set,
// while REST checks that they name the same order, so such a request goes over REST
func (e *Exchange) overWebsocketAPI(ctx context.Context, byID, byClientID bool) bool {
	return (!byID || !byClientID) && e.canUseWebsocketAPIForOrders(ctx)
}

// logOrderIssue logs a value of an order that could not be converted, which leaves that value unset: a listing still
// returns the order, and a placement that Binance.US accepted is not reported as failed
func (e *Exchange) logOrderIssue(orderID string, err error) {
	log.Warnf(log.ExchangeSys, "%s %s order %s: %v", e.Name, asset.Spot, orderID, err)
}

// orderSideString returns the side Binance.US takes for an order's direction
func orderSideString(side order.Side) (Side, error) {
	switch {
	case side.IsLong():
		return SideBuy, nil
	case side.IsShort():
		return SideSell, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// resolveTriggerOrderType returns the limit form of a bare stop or take profit order when it has a price and the market
// form otherwise, since Stop and TakeProfit leave open which of Binance.US's two order types is meant
func resolveTriggerOrderType(t order.Type, price float64) order.Type {
	switch t {
	case order.Stop:
		if price > 0 {
			return order.StopLimit
		}
		return order.StopMarket
	case order.TakeProfit:
		if price > 0 {
			return order.TakeProfitLimit
		}
		return order.TakeProfitMarket
	default:
		return t
	}
}

// submittedOrderType returns the type an order is placed as: a bare stop or take profit resolves to its limit or market
// form, and a post only limit order is placed as LIMIT_MAKER, which every later read reports
func submittedOrderType(s *order.Submit) order.Type {
	t := resolveTriggerOrderType(s.Type, s.Price)
	if t == order.Limit && s.TimeInForce.Is(order.PostOnly) {
		return order.LimitMaker
	}
	return t
}

// timeInForceString returns the time in force a limit priced order is sent with: GTC unless another is requested
func timeInForceString(tif order.TimeInForce) (TimeInForce, error) {
	switch tif {
	case order.UnknownTIF, order.GoodTillCancel:
		return TimeInForceGoodTillCancel, nil
	case order.ImmediateOrCancel:
		return TimeInForceImmediateOrCancel, nil
	case order.FillOrKill:
		return TimeInForceFillOrKill, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
}

// spotOrderValues holds the values an order is sent with, whichever route places it
type spotOrderValues struct {
	side                                           Side
	orderType                                      OrderType
	timeInForce                                    TimeInForce
	price, stopPrice, quantity, quoteOrderQuantity float64
}

// newSpotOrderValues works out the values of an order. Only limit priced types take a price and a time in force, stop
// and take profit types trigger at TriggerPrice, and a market order without an amount spends or receives QuoteAmount
func newSpotOrderValues(s *order.Submit) (*spotOrderValues, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := spotOrderTypeString(resolveTriggerOrderType(s.Type, s.Price), s.TimeInForce)
	if err != nil {
		return nil, err
	}
	if err := checkUntimedSpotTimeInForce(orderType, s.TimeInForce); err != nil {
		return nil, err
	}
	v := &spotOrderValues{side: side, orderType: orderType}
	switch orderType {
	case OrderTypeLimit, OrderTypeStopLossLimit, OrderTypeTakeProfitLimit:
		if v.timeInForce, err = timeInForceString(s.TimeInForce); err != nil {
			return nil, err
		}
		fallthrough
	case OrderTypeLimitMaker:
		if s.Price <= 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
		v.price = s.Price
	}
	switch orderType {
	case OrderTypeStopLoss, OrderTypeStopLossLimit, OrderTypeTakeProfit, OrderTypeTakeProfitLimit:
		if s.TriggerPrice <= 0 {
			return nil, errTriggerPriceRequired
		}
		v.stopPrice = s.TriggerPrice
	}
	if orderType == OrderTypeMarket && s.Amount == 0 {
		v.quoteOrderQuantity = s.QuoteAmount
	} else {
		v.quantity = s.Amount
	}
	if v.quantity <= 0 && v.quoteOrderQuantity <= 0 {
		return nil, fmt.Errorf("%w: %s orders take a base amount", order.ErrAmountIsInvalid, orderType)
	}
	return v, nil
}

// checkUntimedSpotTimeInForce refuses a time in force that an order of a type Binance.US takes without one would not
// keep, as it goes unsent: a LIMIT_MAKER order rests until it is cancelled, and only as a maker, a STOP_LOSS or
// TAKE_PROFIT order waits until it is cancelled or triggers a market order, and a MARKET order expires what it cannot
// fill at once, as an IOC order does. A MARKET order also takes GTC, as Binance.US reports market orders with it
func checkUntimedSpotTimeInForce(orderType OrderType, tif order.TimeInForce) error {
	accepted := order.GoodTillCancel
	switch orderType {
	case OrderTypeLimitMaker:
		accepted |= order.PostOnly
	case OrderTypeMarket:
		accepted |= order.ImmediateOrCancel
	case OrderTypeStopLoss, OrderTypeTakeProfit:
	default:
		return nil
	}
	if tif&^accepted != 0 {
		return fmt.Errorf("%w: %s on %s orders", order.ErrUnsupportedTimeInForce, tif, orderType)
	}
	return nil
}

// checkSpotSubmitOptions refuses what a Binance.US spot order cannot carry, rather than placing the order without it: a
// margin type other than cash, a quote amount beside a base amount, a trigger price on a type placed without one, a
// trigger on a rising price where the order cannot take one, and options none of its order types take. An OCO order
// list's stop loss limit price is the price of its stop limit order. A falling trigger cannot be refused, as it is
// StopDirection's zero value
func checkSpotSubmitOptions(s *order.Submit) error {
	if s.MarginType != margin.Unset && s.MarginType != margin.NoMargin {
		return fmt.Errorf("%w: %s for a %s order", margin.ErrMarginTypeUnsupported, s.MarginType, s.AssetType)
	}
	if s.Amount > 0 && s.QuoteAmount > 0 {
		return errQuoteAmountWithAmount
	}
	risk := s.RiskManagementModes
	if s.Type == order.OCO {
		risk.StopLoss.Enabled, risk.StopLoss.LimitPrice = false, 0
	}
	switch {
	case s.ReduceOnly:
		return fmt.Errorf("%w: reduce only", errOrderOptionUnsupported)
	case s.AutoBorrow, s.AutoRepay:
		return fmt.Errorf("%w: borrowing", errOrderOptionUnsupported)
	case !s.EndTime.IsZero():
		return fmt.Errorf("%w: an end time", errOrderOptionUnsupported)
	case s.TriggerPriceType != order.LastPrice:
		return fmt.Errorf("%w: trigger price type %s", errOrderOptionUnsupported, s.TriggerPriceType)
	case s.TrackingMode != order.UnknownTrackingMode, s.TrackingValue != 0, s.LimitTrackingMode != order.UnknownTrackingMode, s.LimitTrackingValue != 0:
		return fmt.Errorf("%w: price tracking", errOrderOptionUnsupported)
	case s.SlippageTolerance != 0:
		return fmt.Errorf("%w: slippage tolerance", errOrderOptionUnsupported)
	case s.TriggerPrice != 0 && (s.Type == order.Limit || s.Type == order.LimitMaker || s.Type == order.Market):
		return fmt.Errorf("%w: a trigger price on a %s order", errOrderOptionUnsupported, s.Type)
	case s.StopDirection == order.StopUp && risingTriggerUnsupported(s):
		return fmt.Errorf("%w: a trigger on a rising price for a %s %s order", errOrderOptionUnsupported, s.Side, s.Type)
	case risk != (order.RiskManagementModes{}):
		return fmt.Errorf("%w: take profit or stop loss", errOrderOptionUnsupported)
	}
	return nil
}

// risingTriggerUnsupported reports whether an order of s's type and side cannot trigger as the price rises to its stop
// price. Binance.US takes a stop price above the market price, and so triggers as the price rises, for a stop loss buy,
// an OCO order list buy's stop loss and a take profit sell; other stop and take profit orders trigger as it falls, and
// limit, limit maker and market orders have no trigger. A type Binance.US does not place is refused as such later
func risingTriggerUnsupported(s *order.Submit) bool {
	switch s.Type {
	case order.Stop, order.StopLimit, order.StopMarket, order.OCO:
		return !s.Side.IsLong()
	case order.TakeProfit, order.TakeProfitLimit, order.TakeProfitMarket:
		return !s.Side.IsShort()
	case order.Limit, order.LimitMaker, order.Market:
		return true
	default:
		return false
	}
}

// SubmitOrder submits a new order through the WebSocket API when it can be used for orders and REST otherwise, asking
// for the FULL reply, which reports the order's status and fills. A bare Stop or TakeProfit order is placed as the limit
// type when Price is set and the market type otherwise, and a post only limit order as LIMIT_MAKER. An OCO order list is
// a LIMIT_MAKER order at Price and a STOP_LOSS order triggered at TriggerPrice, a STOP_LOSS_LIMIT order when
// RiskManagementModes.StopLoss has a limit price, and is identified by its order list ID, tagged as an order list's
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if s.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	if err := checkSpotSubmitOptions(s); err != nil {
		return nil, err
	}
	if t := submittedOrderType(s); t != s.Type {
		// The response reports the type placed, which the order store and type filters match exactly
		resolved := *s
		resolved.Type = t
		s = &resolved
	}
	var resp *order.SubmitResponse
	var err error
	if s.Type == order.OCO {
		resp, err = e.submitOneCancelsTheOtherOrder(ctx, s)
	} else {
		resp, err = e.submitSpotOrder(ctx, s)
	}
	if err != nil {
		return nil, err
	}
	if s.Type == order.LimitMaker {
		// Reads of a LIMIT_MAKER order report it as post only, whichever post only time in force it was asked for
		resp.TimeInForce = order.PostOnly
	}
	return resp, nil
}

// submitSpotOrder places an order
func (e *Exchange) submitSpotOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	v, err := newSpotOrderValues(s)
	if err != nil {
		return nil, err
	}
	placed, err := e.newOrder(ctx, &NewOrderRequest{
		Symbol:               s.Pair,
		Side:                 v.side,
		Type:                 v.orderType,
		TimeInForce:          v.timeInForce,
		Quantity:             v.quantity,
		QuoteOrderQuantity:   v.quoteOrderQuantity,
		Price:                v.price,
		NewClientOrderID:     s.ClientOrderID,
		StopPrice:            v.stopPrice,
		NewOrderResponseType: NewOrderResponseTypeFull,
	}, e.canUseWebsocketAPIForOrders(ctx))
	if err != nil {
		return nil, err
	}
	return e.spotSubmitResponse(s, placed)
}

// spotSubmitResponse completes a submit response with what Binance.US reports about the placed order
func (e *Exchange) spotSubmitResponse(s *order.Submit, placed *NewOrderResponse) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(placed.OrderID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Date = placed.TransactTime.Time()
	resp.LastUpdated = resp.Date
	if resp.Status, err = stringToOrderStatus(placed.Status); err != nil {
		e.logOrderIssue(resp.OrderID, err)
	}
	resp.RemainingAmount = remainingAmount(placed.OriginalQuantity, placed.ExecutedQuantity, placed.PreventedQuantity, resp.Status)
	if resp.Amount > 0 && placed.ExecutedQuantity > 0 && placed.ExecutedQuantity < placed.OriginalQuantity && resp.Status.IsInactive() {
		// A submit response carries no executed amount, so an order Binance.US finished after executing part of it is
		// reported New for the order manager to settle from a later query. A quote sized market order keeps Binance.US's
		// status, since the order manager cannot treat an order without an amount as active
		resp.Status = order.New
	}
	if placed.ExecutedQuantity > 0 {
		resp.Cost = placed.CumulativeQuoteQuantity.Float64()
		resp.AverageExecutedPrice = averagePrice(placed.CumulativeQuoteQuantity, placed.ExecutedQuantity)
	}
	if len(placed.Fills) == 0 {
		return resp, nil
	}
	resp.Trades = make([]order.TradeHistory, len(placed.Fills))
	for i := range placed.Fills {
		resp.Trades[i] = order.TradeHistory{
			Price:     placed.Fills[i].Price.Float64(),
			Amount:    placed.Fills[i].Quantity.Float64(),
			Fee:       placed.Fills[i].Commission.Float64(),
			Exchange:  e.Name,
			TID:       strconv.FormatUint(placed.Fills[i].TradeID, 10),
			Type:      s.Type,
			Side:      s.Side,
			Timestamp: resp.Date,
			FeeAsset:  placed.Fills[i].CommissionAsset.String(),
			Total:     fillQuoteTotal(placed.Fills[i].Price, placed.Fills[i].Quantity),
		}
	}
	return resp, nil
}

// fillQuoteTotal returns a fill's quote total, its price times its quantity truncated to eight decimal places, as
// Binance.US reports a trade's quote quantity
func fillQuoteTotal(price, quantity types.Number) float64 {
	return price.Decimal().Mul(quantity.Decimal()).Truncate(8).InexactFloat64()
}

// newOneCancelsTheOtherOrderRequest works out an OCO order list's request: a LIMIT_MAKER order at Price and a STOP_LOSS
// order triggered at TriggerPrice, which is a STOP_LOSS_LIMIT order when the stop loss risk management mode has a limit
// price. ClientOrderID identifies the list, so Binance.US names the legs, which their order list ID ties to the list
func newOneCancelsTheOtherOrderRequest(s *order.Submit) (*OneCancelsTheOtherOrderRequest, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	if s.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if s.TriggerPrice <= 0 {
		return nil, errTriggerPriceRequired
	}
	req := &OneCancelsTheOtherOrderRequest{
		Symbol:            s.Pair,
		ListClientOrderID: s.ClientOrderID,
		Side:              side,
		Quantity:          s.Amount,
		Price:             s.Price,
		StopPrice:         s.TriggerPrice,
	}
	limitPrice := s.RiskManagementModes.StopLoss.LimitPrice
	if limitPrice < 0 {
		return nil, errNegativeStopLimitPrice
	}
	if limitPrice > 0 {
		req.StopLimitPrice = limitPrice
		// The limit maker order is post only by type, so the time in force can only be the stop limit order's
		if req.StopLimitTimeInForce, err = timeInForceString(s.TimeInForce &^ order.PostOnly); err != nil {
			return nil, err
		}
	} else if s.TimeInForce&^(order.GoodTillCancel|order.PostOnly) != 0 {
		// Neither the limit maker order nor the stop loss order takes a time in force, so the list's would go unsent
		return nil, fmt.Errorf("%w: %s on OCO order lists without a stop limit order", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	}
	return req, nil
}

// submitOneCancelsTheOtherOrder places an OCO order list. A sell's limit maker order rests above the last price and its
// stop loss order below; a buy's the other way round
func (e *Exchange) submitOneCancelsTheOtherOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	req, err := newOneCancelsTheOtherOrderRequest(s)
	if err != nil {
		return nil, err
	}
	list, err := e.newOneCancelsTheOtherOrder(ctx, req, e.canUseWebsocketAPIForOrders(ctx))
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(formatOrderListID(list.OrderListID))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = list.ListClientOrderID
	resp.Date = list.TransactionTime.Time()
	resp.LastUpdated = resp.Date
	if resp.Status, err = orderListStatus(list.ListOrderStatus); err != nil {
		e.logOrderIssue(resp.OrderID, err)
	}
	return resp, nil
}

// ModifyOrder is not supported: Binance.US amends an order only by cancelling it and placing a new one under a new
// order ID, which CancelExistingOrderAndSendNewOrder does with the outcome of both halves
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its order ID or client order ID, or an OCO order list by its order list ID, tagged as
// an order list's, or list client order ID. A request without a type, as the RPC server and order manager send, with an
// untagged ID is cancelled as the pair's order or OCO order list with it, as untaggedOrderList finds
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return err
	}
	if o.AssetType != asset.Spot {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, o.AssetType)
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	orderID, isList, err := parseSpotOrderID(o.OrderID)
	if err != nil {
		return err
	}
	cancelList := func(orderListID *uint64, listClientOrderID string) error {
		_, err := e.cancelOneCancelsTheOtherOrder(ctx, &CancelOrderListRequest{Symbol: o.Pair, OrderListID: orderListID, ListClientOrderID: listClientOrderID}, e.overWebsocketAPI(ctx, orderListID != nil, listClientOrderID != ""))
		return cancelError(err)
	}
	if isList {
		return cancelList(&orderID, o.ClientOrderID)
	}
	if o.Type == order.OCO {
		return cancelList(untaggedID(orderID), o.ClientOrderID)
	}
	if isUntypedRequest(o.Type) {
		list, err := e.untaggedOrderList(ctx, o.Pair, orderID, o.ClientOrderID)
		if err != nil {
			return err
		}
		if list != nil {
			if err := cancelList(&list.orderListID, ""); err != nil {
				return fmt.Errorf("error cancelling order list: %w", err)
			}
			return nil
		}
	}
	_, err = e.cancelExistingOrder(ctx, &CancelOrderRequest{Symbol: o.Pair, OrderID: orderID, OriginalClientOrderID: o.ClientOrderID}, e.overWebsocketAPI(ctx, orderID != 0, o.ClientOrderID != ""))
	return cancelError(err)
}

// isUntypedRequest reports whether a cancellation names no order type, as the RPC server's and the order manager's do,
// so its ID may be an order list's
func isUntypedRequest(t order.Type) bool {
	return t == order.UnknownType || t == order.AnyType
}

// isUnknownOrder reports whether Binance.US rejected a request because no order or order list of the request has its
// ID
func isUnknownOrder(err error) bool {
	if errors.Is(err, order.ErrOrderNotFound) {
		return true
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == -2011 &&
		(strings.HasPrefix(apiErr.Message, "Unknown order") || strings.HasPrefix(apiErr.Message, "Order list does not exist"))
}

// cancelError returns err, wrapping order.ErrOrderNotFound if the error isUnknownOrder and doesn't already wrap it
func cancelError(err error) error {
	if isUnknownOrder(err) && !errors.Is(err, order.ErrOrderNotFound) {
		return fmt.Errorf("%w: %w", order.ErrOrderNotFound, err)
	}
	return err
}

// foundOrderList identifies an OCO order list a lookup found
type foundOrderList struct {
	orderListID uint64
}

// untaggedID returns an order ID without the order list ID prefix as an order list ID: nil for 0, which an empty order
// ID parses to, as order lists named that way are named by their client ID alone
func untaggedID(orderID uint64) *uint64 {
	if orderID == 0 {
		return nil
	}
	return &orderID
}

// untaggedOrderList returns pair's OCO order list with each ID an untyped cancellation names, its order ID, which has no
// order list ID prefix, as the order list ID and its client order ID as the list client order ID, or nil when pair has
// none. Binance.US numbers order lists apart from orders and callers can hold its order list IDs, so such an ID may name
// either kind, and errAmbiguousOrderListID is returned when pair has an order with it too, naming the order's other ID
// as what tells the two apart. Lookups draw on the cancel pool, so the request they precede does not wait on market
// data
func (e *Exchange) untaggedOrderList(ctx context.Context, pair currency.Pair, orderID uint64, clientOrderID string) (*foundOrderList, error) {
	list, err := e.spotOrderList(ctx, untaggedID(orderID), clientOrderID, pair, cancelLookupRate)
	if errors.Is(err, order.ErrOrderNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	found := foundOrderList{orderListID: list.OrderListID}
	// The order is looked up by the ID the order list was: the order ID when there is one
	id, otherID := strconv.FormatUint(orderID, 10), "client order ID alone"
	if orderID == 0 {
		id, otherID = clientOrderID, "order ID"
	} else {
		clientOrderID = ""
	}
	if _, err = e.spotOrder(ctx, pair, orderID, clientOrderID, cancelLookupRate); err != nil && !errors.Is(err, order.ErrOrderNotFound) {
		err = fmt.Errorf("error fetching order: %w", err)
	}
	switch {
	case err == nil:
		return nil, fmt.Errorf("%w: %s on %s %s; name the order by its %s, or the order list as %s", errAmbiguousOrderListID, id, asset.Spot, pair, otherID, formatOrderListID(found.orderListID))
	case errors.Is(err, order.ErrOrderNotFound):
		return &found, nil
	default:
		return nil, err
	}
}

// CancelBatchOrders is not supported: Binance.US cancels orders one at a time, or every order of a symbol at once,
// which CancelAllOrders does
func (e *Exchange) CancelBatchOrders(context.Context, []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels the open orders and order lists of the request's pair, or of every symbol with open orders,
// reporting each cancelled order's status; errors of individual symbols are returned together after every symbol has
// been tried. Binance.US cancels a symbol's orders of every side and type together, so a cancellation of one side's or
// one type's orders is refused. The listing and cancellations draw on the cancel pool, so they do not wait on market
// data
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	if req.Side != order.UnknownSide && req.Side != order.AnySide {
		return resp, fmt.Errorf("%w: cancelling the orders of one side, %s", common.ErrFunctionNotSupported, req.Side)
	}
	if req.Type != order.UnknownType && req.Type != order.AnyType {
		return resp, fmt.Errorf("%w: cancelling the orders of one type, %s", common.ErrFunctionNotSupported, req.Type)
	}
	if req.AssetType != asset.Spot {
		return resp, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	useWebsocket := e.canUseWebsocketAPIForOrders(ctx)
	pairs := currency.Pairs{req.Pair}
	if req.Pair.IsEmpty() {
		open, err := e.openOrders(ctx, currency.EMPTYPAIR, cancelAllListingRate, useWebsocket)
		if err != nil {
			return resp, err
		}
		var symbols []string
		for i := range open {
			if !slices.Contains(symbols, open[i].Symbol) {
				symbols = append(symbols, open[i].Symbol)
			}
		}
		if pairs, err = e.symbolPairs(symbols); err != nil {
			return resp, err
		}
	}
	var errs error
	for _, p := range pairs {
		cancelled, err := e.cancelAllOpenOrderOnSymbol(ctx, p, useWebsocket)
		if err != nil {
			errs = common.AppendError(errs, cancelAllPairError(p, err))
			continue
		}
		for i := range cancelled {
			if len(cancelled[i].OrderReports) == 0 {
				resp.Add(strconv.FormatUint(cancelled[i].OrderID, 10), cancelled[i].Status)
			}
			for j := range cancelled[i].OrderReports {
				resp.Add(strconv.FormatUint(cancelled[i].OrderReports[j].OrderID, 10), cancelled[i].OrderReports[j].Status)
			}
		}
	}
	return resp, errs
}

// symbolPairs returns the pairs of symbols Binance.US reports, to address them in requests: the matching available
// pair, or for a symbol without one a pair that formats back to the same symbol
func (e *Exchange) symbolPairs(symbols []string) (currency.Pairs, error) {
	pairs := make(currency.Pairs, len(symbols))
	for i := range symbols {
		pair, err := e.MatchSymbolWithAvailablePairs(symbols[i], asset.Spot, false)
		if err != nil {
			if pair, err = currency.NewPairFromString(symbols[i]); err != nil {
				return nil, err
			}
		}
		pairs[i] = pair
	}
	return pairs, nil
}

// cancelAllPairError returns the error of cancelling every open order of a pair, or nil when Binance.US rejected the
// request because the pair had no open order left to cancel
func cancelAllPairError(p currency.Pair, err error) error {
	if isUnknownOrder(err) {
		return nil
	}
	return fmt.Errorf("error cancelling %s orders: %w", p, err)
}

// GetOrderInfo returns an order's details. An order list ID is looked up as an OCO order list's. Binance.US numbers order
// lists apart from orders and callers can hold its order list IDs, so an untagged ID is looked up as both, and
// errAmbiguousOrderListID is returned when pair has both
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if a != asset.Spot {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	id, isList, err := parseSpotOrderID(orderID)
	if err != nil {
		return nil, err
	}
	if isList {
		list, err := e.spotOrderList(ctx, &id, "", pair, queryOrderListRate)
		if err != nil {
			return nil, err
		}
		return e.orderListInfo(list, pair)
	}
	o, orderErr := e.spotOrder(ctx, pair, id, "", queryOrderRate)
	if orderErr != nil && !errors.Is(orderErr, order.ErrOrderNotFound) {
		return nil, orderErr
	}
	list, listErr := e.spotOrderList(ctx, untaggedID(id), "", pair, queryOrderListRate)
	switch {
	case listErr != nil && !errors.Is(listErr, order.ErrOrderNotFound):
		return nil, listErr
	case listErr == nil && orderErr == nil:
		return nil, fmt.Errorf("%w: %s on %s %s; look the order list up as %s", errAmbiguousOrderListID, orderID, asset.Spot, pair, formatOrderListID(list.OrderListID))
	case listErr == nil:
		return e.orderListInfo(list, pair)
	case orderErr != nil:
		return nil, common.AppendError(orderErr, listErr)
	}
	d, err := e.spotOrderDetail(o, pair)
	if err != nil {
		return nil, fmt.Errorf("error converting %s order %s: %w", asset.Spot, orderID, err)
	}
	return &d, nil
}

// spotOrder returns pair's order orderID or, when orderID is 0, clientOrderID, drawing on f
func (e *Exchange) spotOrder(ctx context.Context, pair currency.Pair, orderID uint64, clientOrderID string, f request.EndpointLimit) (*TradeOrderResponse, error) {
	o, err := e.queryOrder(ctx, pair, clientOrderID, orderID, f, e.overWebsocketAPI(ctx, orderID != 0, clientOrderID != ""))
	return o, cancelError(err)
}

// spotOrderList returns pair's OCO order list with orderListID, unless it is nil, and listClientOrderID, unless it is
// empty, drawing on f. order.ErrOrderNotFound is returned when pair has no such OCO order list
func (e *Exchange) spotOrderList(ctx context.Context, orderListID *uint64, listClientOrderID string, pair currency.Pair, f request.EndpointLimit) (*OneCancelsTheOtherOrderResponse, error) {
	list, err := e.getOneCancelsTheOtherOrders(ctx, &QueryOrderListRequest{OrderListID: orderListID, OriginalClientOrderID: listClientOrderID}, f, e.overWebsocketAPI(ctx, orderListID != nil, listClientOrderID != ""))
	if err != nil {
		return nil, cancelError(fmt.Errorf("error fetching order list: %w", err))
	}
	formatted, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	// The order list lookups take no symbol, and other contingency types, such as OTO lists, are not OCO orders, whose
	// orders are found by their own order IDs
	if list.Symbol != formatted || list.ContingencyType != oneCancelsTheOtherContingencyType {
		return nil, fmt.Errorf("%w: no %s OCO order list has the ID", order.ErrOrderNotFound, formatted)
	}
	if orderListID != nil && list.OrderListID != *orderListID || listClientOrderID != "" && list.ListClientOrderID != listClientOrderID {
		return nil, fmt.Errorf("%w: no %s OCO order list has each ID", order.ErrOrderNotFound, formatted)
	}
	return list, nil
}

// orderListInfo converts pair's OCO order list as GetOrderInfo returns it
func (e *Exchange) orderListInfo(list *OneCancelsTheOtherOrderResponse, pair currency.Pair) (*order.Detail, error) {
	d, err := e.orderListDetail(list.OrderListID, list.ListClientOrderID, list.ListOrderStatus, list.TransactionTime.Time(), pair)
	if err != nil {
		return nil, fmt.Errorf("error converting %s order list %d: %w", asset.Spot, list.OrderListID, err)
	}
	return &d, nil
}

// orderPair returns an order's pair: the pair its orders were listed for or, when every symbol's orders were listed, the
// available pair matching its symbol, else the requested pair with that symbol, so a requested pair keeps its orders
// without taking those of an available pair sharing its symbol
func (e *Exchange) orderPair(listed currency.Pair, symbol string, requested map[string]currency.Pair) (currency.Pair, error) {
	if !listed.IsEmpty() {
		return listed, nil
	}
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Spot, false)
	if err == nil {
		return pair, nil
	}
	if p, ok := requested[symbol]; ok {
		return p, nil
	}
	return currency.EMPTYPAIR, err
}

// appendListedOrder adds an order of a listing to orders. An order whose symbol is neither an available nor a requested
// pair is left out, and an order with values that cannot be converted is kept without them, each with a logged warning,
// so one odd order does not fail the listing
func (e *Exchange) appendListedOrder(orders []order.Detail, orderID string, listed currency.Pair, symbol string, requested map[string]currency.Pair, convert func(currency.Pair) (order.Detail, error)) []order.Detail {
	pair, err := e.orderPair(listed, symbol, requested)
	if err != nil {
		e.logOrderIssue(orderID, err)
		return orders
	}
	d, err := convert(pair)
	if err != nil {
		e.logOrderIssue(orderID, err)
	}
	return append(orders, d)
}

// orderVars holds an order's side, status, type and time in force as GCT represents them
type orderVars struct {
	side        order.Side
	status      order.Status
	orderType   order.Type
	timeInForce order.TimeInForce
}

// spotOrderVars converts an order's side, status, type and time in force. Empty values are left unset, and a value that
// cannot be converted is left unset with its error joined into the returned error, so a listing can keep the order and
// log the error while a single order query returns it
func spotOrderVars(side Side, status string, orderType OrderType, timeInForce TimeInForce) (orderVars, error) {
	var vars orderVars
	var errs error
	if side != "" {
		var err error
		vars.side, err = order.StringToOrderSide(string(side))
		errs = common.AppendError(errs, err)
	}
	if status != "" {
		var err error
		vars.status, err = stringToOrderStatus(status)
		errs = common.AppendError(errs, err)
	}
	if orderType != "" {
		var err error
		vars.orderType, err = stringToSpotOrderType(orderType)
		errs = common.AppendError(errs, err)
	}
	if timeInForce != "" {
		var err error
		vars.timeInForce, err = order.StringToTimeInForce(string(timeInForce))
		errs = common.AppendError(errs, err)
	}
	return vars, errs
}

// setFilledCost sets an order's cost and average price from the quote amount its fills total, which Binance.US reports
// as negative for historical orders whose total is not available
func setFilledCost(d *order.Detail, cumulativeQuote, executed types.Number) {
	if executed <= 0 || cumulativeQuote < 0 {
		return
	}
	d.Cost = cumulativeQuote.Float64()
	d.CostAsset = d.Pair.Quote
	d.AverageExecutedPrice = averagePrice(cumulativeQuote, executed)
}

// spotOrderDetail converts an order as the order queries return it. Values that cannot be converted are left unset and
// returned as the error
func (e *Exchange) spotOrderDetail(o *TradeOrderResponse, pair currency.Pair) (order.Detail, error) {
	vars, err := spotOrderVars(o.Side, o.Status, o.Type, o.TimeInForce)
	d := order.Detail{
		TimeInForce:     spotTimeInForce(vars.orderType, vars.timeInForce),
		Price:           o.Price.Float64(),
		Amount:          o.OriginalQuantity.Float64(),
		TriggerPrice:    o.StopPrice.Float64(),
		QuoteAmount:     o.OriginalQuoteOrderQuantity.Float64(),
		ExecutedAmount:  o.ExecutedQuantity.Float64(),
		RemainingAmount: remainingAmount(o.OriginalQuantity, o.ExecutedQuantity, o.PreventedQuantity, vars.status),
		Exchange:        e.Name,
		OrderID:         strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:   o.ClientOrderID,
		Type:            vars.orderType,
		Side:            vars.side,
		Status:          vars.status,
		AssetType:       asset.Spot,
		Date:            o.Time.Time(),
		LastUpdated:     o.UpdateTime.Time(),
		Pair:            pair,
	}
	setFilledCost(&d, o.CumulativeQuoteQuantity, o.ExecutedQuantity)
	return d, err
}

// openOrdersAllSymbolsFrom is the number of symbols from which listing every symbol's open orders in one request, at a
// weight of 40, weighs less than listing each symbol's at 3
const openOrdersAllSymbolsFrom = 14

// GetActiveOrders returns the open orders of the requested pairs, or of every symbol when none are requested. OCO order
// lists are listed as orders identified by their order list ID, tagged as an order list's, when the OCO type is
// requested, while their orders are listed among the open orders otherwise
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	requested, err := e.requestedPairsBySymbol(req.Pairs)
	if err != nil {
		return nil, err
	}
	useWebsocket := e.canUseWebsocketAPIForOrders(ctx)
	var orders []order.Detail
	if req.Type == order.OCO {
		lists, err := e.getOpenOneCancelsTheOtherList(ctx, useWebsocket)
		if err != nil {
			return nil, err
		}
		orders = e.appendOrderLists(orders, lists, requested)
		return e.filterOrders(req, orders), nil
	}
	pairs := req.Pairs
	if len(pairs) == 0 || len(pairs) >= openOrdersAllSymbolsFrom {
		pairs = currency.Pairs{currency.EMPTYPAIR}
	}
	for _, p := range pairs {
		open, err := e.openOrders(ctx, p, openOrdersAllRate, useWebsocket)
		if err != nil {
			return nil, err
		}
		for i := range open {
			o := &open[i]
			orders = e.appendListedOrder(orders, strconv.FormatUint(o.OrderID, 10), p, o.Symbol, requested, func(pair currency.Pair) (order.Detail, error) {
				return e.spotOrderDetail(o, pair)
			})
		}
	}
	return e.filterOrders(req, orders), nil
}

// requestedPairsBySymbol returns the requested pairs by their symbols, so a listing of every symbol's orders can give an
// order whose symbol no available pair has the requested pair with that symbol
func (e *Exchange) requestedPairsBySymbol(pairs currency.Pairs) (map[string]currency.Pair, error) {
	bySymbol := make(map[string]currency.Pair, len(pairs))
	for _, p := range pairs {
		symbol, err := e.FormatSymbol(p, asset.Spot)
		if err != nil {
			return nil, err
		}
		bySymbol[symbol] = p
	}
	return bySymbol, nil
}

// appendOrderLists adds the OCO order lists among lists to orders, each as an order identified by its order list ID,
// tagged as an order list's, of the available pair matching its symbol or else the requested pair with it
func (e *Exchange) appendOrderLists(orders []order.Detail, lists []OneCancelsTheOtherOrderResponse, requested map[string]currency.Pair) []order.Detail {
	for i := range lists {
		l := &lists[i]
		if l.ContingencyType != oneCancelsTheOtherContingencyType {
			continue
		}
		orders = e.appendListedOrder(orders, formatOrderListID(l.OrderListID), currency.EMPTYPAIR, l.Symbol, requested, func(pair currency.Pair) (order.Detail, error) {
			return e.orderListDetail(l.OrderListID, l.ListClientOrderID, l.ListOrderStatus, l.TransactionTime.Time(), pair)
		})
	}
	return orders
}

// filterOrders applies req's filters to orders. A bare Stop or TakeProfit type also covers the limit and market forms
// Binance.US's stop and take profit orders read back as, since SubmitOrder places a bare type as one of them
func (e *Exchange) filterOrders(req *order.MultiOrderRequest, orders []order.Detail) order.FilteredOrders {
	if req.Type != order.Stop && req.Type != order.TakeProfit {
		return req.Filter(e.Name, orders)
	}
	orders = slices.DeleteFunc(slices.Clone(orders), func(d order.Detail) bool { return d.Type&req.Type != req.Type })
	anyType := *req
	anyType.Type = order.AnyType
	return anyType.Filter(e.Name, orders)
}

// orderHistorySpan is the longest window an order or order list history request takes: Binance.US does not document
// one, so Binance's 24 hours for the same endpoints is kept
const orderHistorySpan = 24 * time.Hour

// GetOrderHistory returns the requested pairs' orders of every status: orders from FromOrderID on, or OCO order lists
// from an order list ID on, which needs the OCO type, filtered to the request's time window, or the orders within the
// request's time window, requested a day at a time. OCO order lists are listed when the OCO type is requested, for
// every symbol, as the order list history takes no symbol
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	fromID, isList, err := parseSpotOrderID(req.FromOrderID)
	if err != nil {
		return nil, err
	}
	if isList && req.Type != order.OCO {
		return nil, fmt.Errorf("%w: %q", errOrderListIDRequiresOCOType, req.FromOrderID)
	}
	var orders []order.Detail
	if req.Type == order.OCO {
		requested, err := e.requestedPairsBySymbol(req.Pairs)
		if err != nil {
			return nil, err
		}
		lists, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, orderHistorySpan, 1000, func(start, end time.Time) ([]OneCancelsTheOtherOrderResponse, error) {
			return e.GetAllOneCancelsTheOtherOrders(ctx, &AllOrderListsRequest{FromID: fromID, StartTime: start, EndTime: end, Limit: 1000})
		}, func(l *OneCancelsTheOtherOrderResponse) time.Time { return l.TransactionTime.Time() }, func(l *OneCancelsTheOtherOrderResponse) uint64 { return l.OrderListID })
		if err != nil {
			return nil, err
		}
		return e.filterOrders(req, e.appendOrderLists(orders, lists, requested)), nil
	}
	for _, p := range req.Pairs {
		history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, orderHistorySpan, 1000, func(start, end time.Time) ([]TradeOrderResponse, error) {
			return e.AllOrders(ctx, &AllOrdersRequest{Symbol: p, OrderID: fromID, StartTime: start, EndTime: end, Limit: 1000})
		}, func(o *TradeOrderResponse) time.Time { return o.Time.Time() }, func(o *TradeOrderResponse) uint64 { return o.OrderID })
		if err != nil {
			return nil, err
		}
		for i := range history {
			o := &history[i]
			orders = e.appendListedOrder(orders, strconv.FormatUint(o.OrderID, 10), p, o.Symbol, nil, func(pair currency.Pair) (order.Detail, error) {
				return e.spotOrderDetail(o, pair)
			})
		}
	}
	return e.filterOrders(req, orders), nil
}

// pagedOrderHistory fetches the orders between start and end in spans of at most maxSpan, paging forward by time within
// a span while full pages of limit orders come back. A window holding more than limit orders returns its oldest, but a
// span whose first page is full is also paged back from that page's earliest order, in case it held the span's most
// recent orders, and the orders that adds go before the span's others. Each page starts where the previous one ended,
// or when paging back ends where it started, inclusive, so orders sharing that millisecond are kept, and orders already
// fetched are dropped by ID; only orders beyond a full page within one millisecond, or within a span's first or last
// two, cannot be reached. Spans stop once the next would start after now, which holds no orders yet and fails the
// requests' validation. Without a time window one request returns Binance.US's default, its most recent orders. A
// request from an order ID is one request without a window, which returns the orders from that ID on: the endpoints
// let a window override the ID, or refuse the two together, so the window is applied to the orders returned, either
// bound alone included. Each page is put in time order first, as the order list history lists its most recent order
// lists first
func pagedOrderHistory[T any](start, end time.Time, fromID uint64, maxSpan time.Duration, limit int, fetchPage func(start, end time.Time) ([]T, error), timeOf func(*T) time.Time, idOf func(*T) uint64) ([]T, error) {
	fetch := func(start, end time.Time) ([]T, error) {
		page, err := fetchPage(start, end)
		slices.SortStableFunc(page, func(a, b T) int { return timeOf(&a).Compare(timeOf(&b)) })
		return page, err
	}
	if fromID != 0 {
		if !start.IsZero() && !end.IsZero() && start.After(end) {
			return nil, common.ErrStartAfterEnd
		}
		orders, err := fetch(time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		// Binance.US compares whole milliseconds, which is how it would have applied the window
		start, end = start.Truncate(time.Millisecond), end.Truncate(time.Millisecond)
		inWindow := orders[:0]
		for i := range orders {
			at := timeOf(&orders[i])
			if (start.IsZero() || !at.Before(start)) && (end.IsZero() || !at.After(end)) {
				inWindow = append(inWindow, orders[i])
			}
		}
		return inWindow, nil
	}
	if start.IsZero() || end.IsZero() {
		return fetch(start, end)
	}
	// Binance.US takes times in milliseconds, so paging works in whole ones, as order times are; a window within one
	// millisecond cannot be paged and is one request
	startMilli, endMilli := start.Truncate(time.Millisecond), end.Truncate(time.Millisecond)
	if !startMilli.Before(endMilli) {
		return fetch(start, end)
	}
	start, end = startMilli, endMilli
	var orders []T
	seen := make(map[uint64]struct{})
	unseen := func(page []T) []T {
		var fresh []T
		for i := range page {
			id := idOf(&page[i])
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				fresh = append(fresh, page[i])
			}
		}
		return fresh
	}
	for spanStart := start; ; {
		spanEnd := end
		if maxSpan > 0 && spanStart.Add(maxSpan).Before(end) {
			spanEnd = spanStart.Add(maxSpan)
		}
		spanFirst := len(orders)
		// earliest is the time of a full first page's earliest order; the first full page is the first page, as one that
		// is not full ends the paging
		var earliest time.Time
		for pageStart := spanStart; ; {
			page, err := fetch(pageStart, spanEnd)
			if err != nil {
				return nil, err
			}
			fresh := unseen(page)
			orders = append(orders, fresh...)
			if len(page) < limit {
				break
			}
			if earliest.IsZero() {
				earliest = timeOf(&page[0])
			}
			next := timeOf(&page[len(page)-1])
			if next.Before(pageStart) {
				next = pageStart
			}
			if len(fresh) == 0 {
				// A full page of orders already fetched cannot reach the rest of its millisecond, so paging moves past it
				next = next.Add(time.Millisecond)
			}
			if !next.Before(spanEnd) {
				// A window must start before it ends, so the span's last millisecond gets one more page from the
				// millisecond before it
				last := spanEnd.Add(-time.Millisecond)
				if !pageStart.Before(last) {
					break
				}
				next = last
			}
			pageStart = next
		}
		if !earliest.IsZero() {
			for pageEnd := earliest; ; {
				if !spanStart.Before(pageEnd) {
					// A window must start before it ends, so the span's first millisecond gets a page with the one after it
					pageEnd = spanStart.Add(time.Millisecond)
				}
				page, err := fetch(spanStart, pageEnd)
				if err != nil {
					return nil, err
				}
				fresh := unseen(page)
				orders = slices.Insert(orders, spanFirst, fresh...)
				if len(page) < limit || !spanStart.Add(time.Millisecond).Before(pageEnd) {
					break
				}
				pageEnd = timeOf(&page[0])
				if len(fresh) == 0 {
					// A full page of orders already fetched cannot reach the rest of its millisecond, so paging moves before it
					pageEnd = pageEnd.Add(-time.Millisecond)
				}
			}
		}
		if !spanEnd.Before(end) || spanEnd.After(time.Now()) {
			return orders, nil
		}
		spanStart = spanEnd
	}
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetFee returns an estimate of fee based on type of transaction. Trade fees are at the account's commission rates for
// the pair, and withdrawal fees those of the coin's default network
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		rate, err := e.tradeFeeRate(ctx, feeBuilder.Pair, feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
		fee = rate * feeBuilder.PurchasePrice * feeBuilder.Amount
	case exchange.CryptocurrencyWithdrawalFee:
		var err error
		if fee, err = e.cryptocurrencyWithdrawalFee(ctx, feeBuilder.Pair.Base); err != nil {
			return 0, err
		}
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	return max(fee, 0), nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.001 * price * amount
}

// tradeFeeRate returns the account's maker or taker fee rate on a pair, which Binance.US reports as a fraction of the
// trade's value
func (e *Exchange) tradeFeeRate(ctx context.Context, pair currency.Pair, isMaker bool) (float64, error) {
	symbol, err := e.formatSymbol(pair)
	if err != nil {
		return 0, err
	}
	fees, err := e.GetTradeFee(ctx, pair)
	if err != nil {
		return 0, err
	}
	for i := range fees {
		if fees[i].Symbol != symbol {
			continue
		}
		if isMaker {
			return fees[i].MakerCommission.Float64(), nil
		}
		return fees[i].TakerCommission.Float64(), nil
	}
	return 0, fmt.Errorf("%w: %s", errTradeFeeNotFound, symbol)
}

// ValidateAPICredentials validates current credentials used for wrapper
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
	candles, err := e.klineCandles(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.klineCandles(ctx, req.RequestFormatted, req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, req.RangeHolder.Limit)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// klineCandles returns up to limit candles of a pair from start to end
func (e *Exchange) klineCandles(ctx context.Context, pair currency.Pair, interval kline.Interval, start, end time.Time, limit uint64) ([]kline.Candle, error) {
	resp, err := e.GetSpotKline(ctx, &KlinesRequest{
		Symbol:    pair,
		Interval:  interval,
		StartTime: start,
		EndTime:   end,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, len(resp))
	for i, c := range resp {
		candles[i] = kline.Candle{
			Time:   c.OpenTime.Time().UTC(),
			Open:   c.OpenPrice.Float64(),
			High:   c.HighPrice.Float64(),
			Low:    c.LowPrice.Float64(),
			Close:  c.ClosePrice.Float64(),
			Volume: c.Volume.Float64(),
		}
	}
	return candles, nil
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	l, err := e.FetchExchangeLimits(ctx, a)
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return limits.Load(l)
}

// Symbol filter types the execution limits are taken from
const (
	priceFilter              = "PRICE_FILTER"
	percentPriceFilter       = "PERCENT_PRICE"
	percentPriceBySideFilter = "PERCENT_PRICE_BY_SIDE"
	lotSizeFilter            = "LOT_SIZE"
	minNotionalFilter        = "MIN_NOTIONAL"
	notionalFilter           = "NOTIONAL"
	icebergPartsFilter       = "ICEBERG_PARTS"
	marketLotSizeFilter      = "MARKET_LOT_SIZE"
	maxNumOrdersFilter       = "MAX_NUM_ORDERS"
	maxNumAlgoOrdersFilter   = "MAX_NUM_ALGO_ORDERS"
)

// FetchExchangeLimits returns the order execution limits of the symbols that trade on the spot market
func (e *Exchange) FetchExchangeLimits(ctx context.Context, a asset.Item) ([]limits.MinMaxLevel, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	info, err := e.GetExchangeInfo(ctx, nil)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(info.Symbols))
	for i := range info.Symbols {
		s := &info.Symbols[i]
		if !s.IsSpotTradingAllowed {
			continue
		}
		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, a, currency.NewPair(s.BaseAsset, s.QuoteAsset)),
		}
		for j := range s.Filters {
			f := &s.Filters[j]
			switch f.FilterType {
			case priceFilter:
				mml.MinPrice = f.MinPrice.Float64()
				mml.MaxPrice = f.MaxPrice.Float64()
				mml.PriceStepIncrementSize = f.TickSize.Float64()
			case percentPriceFilter:
				narrowPriceBand(&mml, f.MultiplierUp.Float64(), f.MultiplierDown.Float64(), f.AveragePriceMinutes)
			case percentPriceBySideFilter:
				// The limits hold one price band for both sides, so they take the widest
				narrowPriceBand(&mml, max(f.BidMultiplierUp.Float64(), f.AskMultiplierUp.Float64()), min(f.BidMultiplierDown.Float64(), f.AskMultiplierDown.Float64()), f.AveragePriceMinutes)
			case lotSizeFilter:
				mml.MinimumBaseAmount = f.MinQuantity.Float64()
				mml.MaximumBaseAmount = f.MaxQuantity.Float64()
				mml.AmountStepIncrementSize = f.StepSize.Float64()
			case minNotionalFilter, notionalFilter:
				mml.MinNotional = f.MinNotional.Float64()
			case icebergPartsFilter:
				mml.MaxIcebergParts = boundedInt64(f.Limit)
			case marketLotSizeFilter:
				mml.MarketMinQty = f.MinQuantity.Float64()
				mml.MarketMaxQty = f.MaxQuantity.Float64()
				mml.MarketStepIncrementSize = f.StepSize.Float64()
			case maxNumOrdersFilter:
				mml.MaxTotalOrders = boundedInt64(f.MaxNumberOrders)
			case maxNumAlgoOrdersFilter:
				mml.MaxAlgoOrders = boundedInt64(f.MaxNumberAlgoOrders)
			}
		}
		l = append(l, mml)
	}
	return l, nil
}

// narrowPriceBand narrows a symbol's price band to a price band filter's: a symbol can have both PERCENT_PRICE and
// PERCENT_PRICE_BY_SIDE, and an order must pass each
func narrowPriceBand(mml *limits.MinMaxLevel, up, down float64, averagePriceMinutes uint64) {
	if mml.MultiplierUp == 0 || up < mml.MultiplierUp {
		mml.MultiplierUp = up
	}
	mml.MultiplierDown = max(mml.MultiplierDown, down)
	mml.AveragePriceMinutes = boundedInt64(averagePriceMinutes)
}

// boundedInt64 returns a count as an int64, capped at the largest int64
func boundedInt64(count uint64) int64 {
	return int64(min(count, math.MaxInt64))
}

// GetServerTime returns the current exchange server time
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	if a != asset.Spot {
		return time.Time{}, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	resp, err := e.GetExchangeServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// GetFuturesContractDetails is not supported: Binance.US trades spot only
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates is not supported: Binance.US trades spot only
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
		return "", err
	}
	symbol, err := e.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	return tradeBaseURL + symbol, nil
}

// GetAccountFundingHistory is not supported: Binance.US's crypto deposit and withdrawal histories take one coin per
// request, so listing every coin's would make a request for each coin it lists
func (e *Exchange) GetAccountFundingHistory(context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// fundingHistoryPageLimit is the most deposit or withdrawal records one request returns
const fundingHistoryPageLimit = 1000

// GetWithdrawalsHistory returns a coin's withdrawals of the last 90 days, the history's default window, requesting pages
// until one is not full
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var withdrawals []WithdrawalRecordResponse
	for {
		page, err := e.WithdrawalHistory(ctx, &WithdrawalHistoryRequest{Coin: c, Offset: uint64(len(withdrawals)), Limit: fundingHistoryPageLimit})
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, page...)
		if len(page) < fundingHistoryPageLimit {
			break
		}
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		// Withdrawal records carry no transaction ID, so the withdrawal's ID is its transfer ID alone
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawalStatusString(withdrawals[i].Status),
			TransferID:      withdrawals[i].ID,
			Timestamp:       withdrawals[i].ApplyTime.Time(),
			Currency:        withdrawals[i].Coin.String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].TransactionFee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[i].Address,
			CryptoChain:     withdrawals[i].Network,
		}
	}
	return resp, nil
}

// withdrawalStatusString returns the documented name of a withdrawal status, or the code of an undocumented one
func withdrawalStatusString(status uint64) string {
	switch status {
	case 0:
		return "Email sent"
	case 1:
		return "Cancelled"
	case 2:
		return "Awaiting approval"
	case 3:
		return "Rejected"
	case 4:
		return "Processing"
	case 5:
		return "Failure"
	case 6:
		return "Completed"
	default:
		return strconv.FormatUint(status, 10)
	}
}

// GetDepositAddress returns a deposit address for a specified currency, on its default network when chain is empty
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := e.GetDepositAddressForCurrency(ctx, cryptocurrency, chain)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: addr.Address,
		Tag:     addr.Tag,
		Chain:   chain,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted. Binance.US requires the network,
// so a request without a chain is sent on the coin's default network
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	network := withdrawRequest.Crypto.Chain
	if network == "" {
		coinNetwork, err := e.defaultNetwork(ctx, withdrawRequest.Currency)
		if err != nil {
			return nil, err
		}
		network = coinNetwork.Network
	}
	resp, err := e.WithdrawCrypto(ctx, &WithdrawRequest{
		Coin:            withdrawRequest.Currency,
		WithdrawOrderID: withdrawRequest.ClientOrderID,
		Network:         network,
		Address:         withdrawRequest.Crypto.Address,
		AddressTag:      withdrawRequest.Crypto.AddressTag,
		Amount:          withdrawRequest.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ID}, nil
}

// WithdrawFiatFunds is not supported: Binance.US withdraws fiat by payment method to a linked payment account, which a
// withdrawal request does not name; WithdrawFiat takes both
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported, as WithdrawFiatFunds is not
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

var errDefaultNetworkNotFound = errors.New("default network not found")

// coinInfo returns a coin's balances and networks
func (e *Exchange) coinInfo(ctx context.Context, c currency.Code) (*CoinInfoResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	coins, err := e.GetAssetFeesAndWalletStatus(ctx)
	if err != nil {
		return nil, err
	}
	for i := range coins {
		if coins[i].Coin.Equal(c) {
			return &coins[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyNotFound, c)
}

// defaultNetwork returns a coin's default network
func (e *Exchange) defaultNetwork(ctx context.Context, c currency.Code) (*CoinNetwork, error) {
	coin, err := e.coinInfo(ctx, c)
	if err != nil {
		return nil, err
	}
	for i := range coin.NetworkList {
		if coin.NetworkList[i].IsDefault {
			return &coin.NetworkList[i], nil
		}
	}
	return nil, fmt.Errorf("%w for %s", errDefaultNetworkNotFound, c)
}

// cryptocurrencyWithdrawalFee returns the fee for withdrawing a coin on its default network
func (e *Exchange) cryptocurrencyWithdrawalFee(ctx context.Context, c currency.Code) (float64, error) {
	network, err := e.defaultNetwork(ctx, c)
	if err != nil {
		return 0, err
	}
	return network.WithdrawFee.Float64(), nil
}

// GetAvailableTransferChains returns the networks a coin can be deposited on
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coin, err := e.coinInfo(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	var chains []string
	for i := range coin.NetworkList {
		if coin.NetworkList[i].DepositEnable {
			chains = append(chains, coin.NetworkList[i].Network)
		}
	}
	return chains, nil
}
