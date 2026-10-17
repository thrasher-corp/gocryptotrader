package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net" //nolint:gosec // G101 false positive on a public URL
	// hbInterval keeps traffic flowing for the connection monitor when the subscribed channels are quiet
	hbInterval = 8 * time.Second

	privateChannelPrefix = "private-"
)

// Websocket channels
// diff_order_book is not supported: its documented reconciliation with a REST snapshot keeps levels of orders which
// are created and deleted between two broadcasts, which leaves the book crossed
const (
	channelLiveTrades         = "live_trades"
	channelOrderbook          = "order_book"
	channelDetailOrderbook    = "detail_order_book"
	channelFundingRate        = "funding_rate"
	channelAnnouncements      = "announcements"
	channelMyOrders           = "my_orders"
	channelMyTrades           = "my_trades"
	channelLiveOrderTrades    = "live_order_trades"
	channelMySettlements      = "my_settlements"
	channelMyLiquidations     = "my_liquidations"
	channelMyTokenSettlements = "my_token_settlements"
	// channelSelfTrades subscribes to the private live trades channel, which only carries self trades
	channelSelfTrades = "self_trades"
)

// Order events of the private orders channel
const (
	orderEventCreated      = "order_created"
	orderEventChanged      = "order_changed"
	orderEventDeleted      = "order_deleted"
	orderEventReplaced     = "order_replaced"
	orderEventStopActive   = "stop_active"
	orderEventStopInactive = "stop_inactive"
)

// maxOrderEventRecoveryFailures is the number of failed recoveries in a row after which missed order events are given
// up on, so that a recovery which keeps failing does not hold back every later event
const maxOrderEventRecoveryFailures = 3

var (
	errParsingWSPair      = errors.New("unable to parse currency pair from channel")
	errMalformedChannel   = errors.New("malformed channel name")
	errWebsocketError     = errors.New("websocket error")
	errOrderIDMissing     = errors.New("order event missing an order ID")
	errOrderEventRecovery = errors.New("unable to recover missed order events")
	errOrderEventsSkipped = errors.New("missed order events skipped")
	errStaleOrderEvent    = errors.New("order event is not after the last one handled")
	errOrderIDOverflow    = errors.New("order ID exceeds the maximum orderbook level ID")

	hbMsg = []byte(`{"event":"bts:heartbeat"}`)
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.OrderbookChannel: channelOrderbook,
	subscription.AllTradesChannel: channelLiveTrades,
	subscription.MyOrdersChannel:  channelMyOrders,
	subscription.MyTradesChannel:  channelMyTrades,
	channelDetailOrderbook:        channelDetailOrderbook,
	channelFundingRate:            channelFundingRate,
	channelAnnouncements:          channelAnnouncements,
	channelLiveOrderTrades:        channelLiveOrderTrades,
	channelSelfTrades:             channelLiveTrades,
	channelMySettlements:          channelMySettlements,
	channelMyLiquidations:         channelMyLiquidations,
	channelMyTokenSettlements:     channelMyTokenSettlements,
}

// accountChannels are channels which are not specific to a market; subscriptions to them should use an empty asset
var accountChannels = []string{channelAnnouncements, channelMySettlements, channelMyLiquidations, channelMyTokenSettlements}

// privateChannels are the subscription channels which can only be subscribed to with a token
var privateChannels = []string{
	subscription.MyOrdersChannel, subscription.MyTradesChannel, channelLiveOrderTrades, channelSelfTrades,
	channelMySettlements, channelMyLiquidations, channelMyTokenSettlements,
}

// WebsocketConnect connects to a websocket feed
func (e *Exchange) WebsocketConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	ctx := context.TODO()
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}, nil)
	if err != nil {
		return err
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
	}
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     hbMsg,
		Delay:       hbInterval,
	})

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()

	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := e.wsHandleData(ctx, resp.Raw); err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
	}
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	event, err := jsonparser.GetUnsafeString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w `event`: %w", common.ErrParsingWSField, err)
	}

	switch event {
	case "bts:heartbeat":
		return nil
	case "bts:subscription_succeeded", "bts:unsubscription_succeeded":
		return e.handleWSSubscription(event, respRaw)
	case "bts:error":
		return handleWSError(respRaw)
	case "bts:request_reconnect":
		go func() {
			if err := e.Websocket.Shutdown(); err != nil { // Connection monitor will reconnect
				log.Errorf(log.WebsocketMgr, "%s failed to shutdown websocket: %v", e.Name, err)
			}
		}()
		return nil
	}

	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}
	name, market, private, err := splitChannel(channel)
	if err != nil {
		return err
	}
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w `data`: %w", common.ErrParsingWSField, err)
	}

	switch {
	case name == channelAnnouncements && !private:
		return e.handleWSAnnouncement(ctx, event, data)
	case name == channelMySettlements && private:
		return e.handleWSSettlement(ctx, event, data)
	case name == channelMyLiquidations && private:
		var alert WebsocketLiquidationAlert
		if err := json.Unmarshal(data, &alert); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &alert)
	case name == channelMyTokenSettlements && private:
		var settlement WebsocketTokenSettlement
		if err := json.Unmarshal(data, &settlement); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &settlement)
	case market == "":
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}

	pair, a, err := e.marketPair(market)
	if err != nil {
		return err
	}
	switch {
	case name == channelOrderbook && !private:
		return e.handleWSOrderbook(ctx, data, pair, a)
	case name == channelDetailOrderbook && !private:
		return e.handleWSDetailOrderbook(ctx, data, pair, a)
	case name == channelLiveTrades && !private:
		return e.handleWSTrade(data, pair, a)
	case name == channelLiveTrades && private:
		return e.handleWSSelfTrade(ctx, data, pair, a)
	case name == channelFundingRate && !private:
		return e.handleWSFundingRate(ctx, data, pair, a)
	case name == channelMyOrders && private:
		return e.handleWSAccountOrder(ctx, respRaw, event, data, pair, a)
	case (name == channelMyTrades || name == channelLiveOrderTrades) && private:
		return e.handleWSMyTrade(ctx, data, pair, a)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

func (e *Exchange) handleWSSubscription(event string, respRaw []byte) error {
	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}
	op := strings.TrimSuffix(strings.TrimPrefix(event, "bts:"), "scription_succeeded")
	return e.Websocket.Match.RequireMatchWithData(op+":"+channel, respRaw)
}

// handleWSError returns the error reported by the server
// Errors do not identify the request or channel which caused them, so failed subscriptions time out
func handleWSError(respRaw []byte) error {
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w `data`: %w", common.ErrParsingWSField, err)
	}
	var wsErr WebsocketError
	if err := json.Unmarshal(data, &wsErr); err != nil {
		return err
	}
	if wsErr.Code != 0 {
		return fmt.Errorf("%w %d: %s", errWebsocketError, wsErr.Code, wsErr.Message)
	}
	return fmt.Errorf("%w: %s", errWebsocketError, wsErr.Message)
}

func (e *Exchange) handleWSOrderbook(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	var ob WebsocketOrderbook
	if err := json.Unmarshal(data, &ob); err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              ob.Bids.Levels(),
		Asks:              ob.Asks.Levels(),
		Pair:              pair,
		LastUpdated:       ob.Microtimestamp.Time(),
		Asset:             a,
		Exchange:          e.Name,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	filterOrderbookZeroBidPrice(book)
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// handleWSDetailOrderbook loads a snapshot of the top orders of an orderbook, which repeats the price of each order at
// a price level
func (e *Exchange) handleWSDetailOrderbook(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	var ob WebsocketDetailOrderbook
	if err := json.Unmarshal(data, &ob); err != nil {
		return err
	}
	bids, err := orderLevels(ob.Bids)
	if err != nil {
		return err
	}
	asks, err := orderLevels(ob.Asks)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Pair:              pair,
		LastUpdated:       ob.Microtimestamp.Time(),
		Asset:             a,
		Exchange:          e.Name,
		PriceDuplication:  true,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	filterOrderbookZeroBidPrice(book)
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// orderLevels returns orderbook levels which each hold an order
func orderLevels(levels []OrderbookLevel) (orderbook.Levels, error) {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		if levels[i].OrderID > math.MaxInt64 {
			return nil, fmt.Errorf("%w: %d", errOrderIDOverflow, levels[i].OrderID)
		}
		l[i] = orderbook.Level{Price: levels[i].Price, Amount: levels[i].Amount, ID: int64(levels[i].OrderID)}
	}
	return l, nil
}

func (e *Exchange) handleWSTrade(data []byte, pair currency.Pair, a asset.Item) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var t WebsocketTrade
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		Timestamp:    t.Microtimestamp.Time(),
		CurrencyPair: pair,
		AssetType:    a,
		Exchange:     e.Name,
		Price:        t.Price.Float64(),
		Amount:       t.Amount.Float64(),
		Side:         t.Side.Side(),
		TID:          strconv.FormatUint(t.ID, 10),
	})
}

func (e *Exchange) handleWSSelfTrade(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	selfTrade := &WebsocketSelfTrade{Pair: pair, AssetType: a}
	if err := json.Unmarshal(data, selfTrade); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, selfTrade)
}

// handleWSFundingRate relays the funding rate, and the funding update with its mark and index prices
func (e *Exchange) handleWSFundingRate(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	rate := &WebsocketFundingRate{Pair: pair, AssetType: a}
	if err := json.Unmarshal(data, rate); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, websocket.FundingData{
		Timestamp:    rate.Timestamp.Time(),
		CurrencyPair: pair,
		AssetType:    a,
		Exchange:     e.Name,
		Rate:         rate.FundingRate.Float64(),
	}); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, rate)
}

// handleWSAccountOrder processes an event for one of the account's orders
// Each event names the event before it, so events missed by a reconnection or a server restart are recovered through
// the REST API before the event is processed. Events are relayed in order, so an event whose recovery fails is held
// back, and the next event recovers it along with the events missed before it
func (e *Exchange) handleWSAccountOrder(ctx context.Context, respRaw []byte, event string, data []byte, pair currency.Pair, a asset.Item) error {
	var o OrderEventData
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	eventID, eventErr := jsonparser.GetString(respRaw, "event_id")
	preEventID, preEventType, _, preEventErr := jsonparser.Get(respRaw, "pre_event_id")
	if eventErr != nil || eventID == "" || preEventErr != nil {
		return e.processOrderEvent(ctx, event, &o, pair, a)
	}
	orderSource, _ := jsonparser.GetString(respRaw, "order_source")
	key := pair.String() + "|" + a.String() + "|" + orderSource
	e.orderEventMu.Lock()
	checkpoint, ok := e.orderEventCheckpoints[key]
	e.orderEventMu.Unlock()
	var errs error
	switch {
	case ok && checkpoint.eventID == eventID:
		// A repeated event has nothing to recover before it, and is only relayed again if it was refused
		if !checkpoint.pending {
			return nil
		}
	case ok && (checkpoint.pending || preEventType == jsonparser.Null || string(preEventID) != checkpoint.eventID):
		// A null preceding event follows a server restart, which may have dropped any number of events
		failures := checkpoint.failures
		checkpoint, errs = e.recoverOrderEvents(ctx, pair, a, orderSource, checkpoint, eventID)
		if errors.Is(errs, errOrderEventRecovery) {
			// The API refuses a recovery range when the event is not after the checkpoint, so it is dropped rather than
			// relayed over the newer order state
			if errors.Is(errs, errInvalidEventIDRange) {
				return fmt.Errorf("%w: %s for %s %s", errStaleOrderEvent, eventID, pair, a)
			}
			// The event is held back until the events missed before it are recovered, unless recovery keeps failing or
			// the API key lacks permission for it, as every later event would be held back too
			checkpoint.failures = failures + 1
			if checkpoint.failures < maxOrderEventRecoveryFailures && !errors.Is(errs, errNoPermission) {
				e.setOrderEventCheckpoint(key, checkpoint)
				return errs
			}
			errs = common.AppendError(errs, fmt.Errorf("%w for %s %s before event %s", errOrderEventsSkipped, pair, a, eventID))
		}
	}
	err := e.processOrderEvent(ctx, event, &o, pair, a)
	e.setOrderEventCheckpoint(key, orderEventCheckpoint{eventID: eventID, pending: isOrderEventRetryable(err)})
	return common.AppendError(errs, err)
}

// recoverOrderEvents processes the account's order events after a checkpoint and before the current event, and returns
// the checkpoint of the last event handled
// The API returns events from the since ID inclusive to the until ID exclusive, at most 1000 at a time. The error wraps
// errOrderEventRecovery when recovery stopped before reaching the current event
func (e *Exchange) recoverOrderEvents(ctx context.Context, pair currency.Pair, a asset.Item, orderSource string, checkpoint orderEventCheckpoint, untilID string) (orderEventCheckpoint, error) {
	req := &OrderEventsRequest{Pair: pair, OrderSource: orderSource, SinceID: checkpoint.eventID, UntilID: untilID}
	processed := map[string]bool{checkpoint.eventID: !checkpoint.pending}
	var errs error
	for {
		events, err := e.GetAccountOrderData(ctx, req)
		if err != nil {
			return checkpoint, common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderEventRecovery, pair, a, err))
		}
		var added bool
		for i := range events {
			if processed[events[i].EventID] {
				continue
			}
			processed[events[i].EventID] = true
			added = true
			err := e.processOrderEvent(ctx, events[i].Event, &events[i].Data, pair, a)
			if isOrderEventRetryable(err) {
				return orderEventCheckpoint{eventID: events[i].EventID, pending: true}, common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderEventRecovery, pair, a, err))
			}
			errs = common.AppendError(errs, err)
			checkpoint = orderEventCheckpoint{eventID: events[i].EventID}
		}
		if !added || len(events) < maxResultLimit {
			return checkpoint, errs
		}
		req.SinceID = events[len(events)-1].EventID
	}
}

// setOrderEventCheckpoint stores the last private order event handled for a market and order source
func (e *Exchange) setOrderEventCheckpoint(key string, checkpoint orderEventCheckpoint) {
	e.orderEventMu.Lock()
	defer e.orderEventMu.Unlock()
	if e.orderEventCheckpoints == nil {
		e.orderEventCheckpoints = make(map[string]orderEventCheckpoint)
	}
	e.orderEventCheckpoints[key] = checkpoint
}

// isOrderEventRetryable returns whether an order event which failed processing is retried by the next recovery
// Only events the data handler refused are retried, since an event without an order ID is recovered without one too
func isOrderEventRetryable(err error) bool {
	return err != nil && !errors.Is(err, errOrderIDMissing)
}

// processOrderEvent relays an order event as order details; a replacement also reports the replaced order as cancelled,
// in the same payload
func (e *Exchange) processOrderEvent(ctx context.Context, event string, o *OrderEventData, pair currency.Pair, a asset.Item) error {
	if o.ID == 0 && o.ClientOrderID == "" {
		return fmt.Errorf("%w: %s %+v", errOrderIDMissing, event, o)
	}

	amount := o.AmountAtCreate.Float64()
	remaining := o.RemainingAmount.Float64()
	// AmountTraded only holds the amount executed by this event, so the total executed amount is derived instead
	executed := amount - remaining

	var status order.Status
	switch event {
	case orderEventCreated, orderEventReplaced:
		status = order.New
	case orderEventChanged:
		status = order.Open
		if executed > 0 {
			status = order.PartiallyFilled
		}
	case orderEventDeleted:
		switch {
		case remaining == 0 && amount > 0:
			status = order.Filled
		case executed > 0:
			status = order.PartiallyCancelled
		default:
			status = order.Cancelled
		}
	case orderEventStopActive:
		status = order.Active
	case orderEventStopInactive:
		status = order.Cancelled
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: fmt.Sprintf("%s%s%s %+v", e.Name, websocket.UnhandledMessage, event, o)})
	}

	orderType, tif := orderTypeFromSubtypeID(o.OrderSubtype)
	details := []order.Detail{{
		Price:           o.Price.Float64(),
		Amount:          amount,
		RemainingAmount: remaining,
		ExecutedAmount:  executed,
		Exchange:        e.Name,
		ClientOrderID:   o.ClientOrderID,
		Type:            orderType,
		TimeInForce:     tif,
		Side:            o.Side.Side(),
		Status:          status,
		AssetType:       a,
		Date:            o.Microtimestamp.Time(),
		LastUpdated:     o.Microtimestamp.Time(),
		Pair:            pair,
		ReduceOnly:      o.ReduceOnly,
		TriggerPrice:    triggerPrice(o.StopPrice.Float64(), o.ActivationPrice.Float64()),
	}}
	if o.ID != 0 {
		details[0].OrderID = strconv.FormatUint(o.ID, 10)
	}
	if o.TradeAccountID != 0 {
		details[0].AccountID = strconv.FormatUint(o.TradeAccountID, 10)
	}
	if event == orderEventReplaced && o.OriginalOrderID != 0 {
		// A replacement order has a new ID, so the replaced order is reported as cancelled
		details = append(details, order.Detail{
			Exchange:    e.Name,
			OrderID:     strconv.FormatUint(o.OriginalOrderID, 10),
			Status:      order.Cancelled,
			AssetType:   a,
			Pair:        pair,
			LastUpdated: o.Microtimestamp.Time(),
		})
	}
	if len(details) == 1 {
		return e.Websocket.DataHandler.Send(ctx, &details[0])
	}
	// Both orders are relayed in one payload, so a refused replacement is retried without repeating either of them
	return e.Websocket.DataHandler.Send(ctx, details)
}

// orderTypeFromSubtypeID returns the order type and time in force of a websocket order subtype
func orderTypeFromSubtypeID(subtype uint8) (order.Type, order.TimeInForce) {
	switch subtype {
	case 0:
		return order.Limit, order.GoodTillCancel
	case 1, 2, 7:
		return order.Market, order.UnknownTIF
	case 3:
		return order.Limit, order.GoodTillDay
	case 4:
		return order.Limit, order.ImmediateOrCancel
	case 5:
		return order.Limit, order.PostOnly
	case 6:
		return order.Limit, order.FillOrKill
	case 8:
		return order.Limit, order.GoodTillTime
	case 20:
		return order.StopMarket, order.UnknownTIF
	case 21:
		return order.TakeProfitMarket, order.UnknownTIF
	case 22:
		return order.StopLimit, order.UnknownTIF
	case 23:
		return order.TakeProfit | order.Limit, order.UnknownTIF
	case 24, 25:
		return order.TrailingStop, order.UnknownTIF
	case 26, 27:
		return order.TrailingStopLimit, order.UnknownTIF
	default:
		return order.UnknownType, order.UnknownTIF
	}
}

// handleWSMyTrade processes an execution of one of the account's orders
// The execution is relayed as a fill when the fills feed is enabled and as an order update holding the trade, which
// carries the trade's fee to the order
func (e *Exchange) handleWSMyTrade(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	var t WebsocketMyTrade
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	side, err := order.StringToOrderSide(t.Side)
	if err != nil {
		return err
	}
	var orderID, tradeID string
	if t.OrderID != 0 {
		orderID = strconv.FormatUint(t.OrderID, 10)
	}
	if t.ID != 0 {
		tradeID = strconv.FormatUint(t.ID, 10)
	}
	if e.IsFillsFeedEnabled() {
		if err := e.Websocket.Fills.Update(fill.Data{
			Timestamp:     t.Microtimestamp.Time(),
			Exchange:      e.Name,
			AssetType:     a,
			CurrencyPair:  pair,
			Side:          side,
			OrderID:       orderID,
			ClientOrderID: t.ClientOrderID,
			TradeID:       tradeID,
			Price:         t.Price.Float64(),
			Amount:        t.Amount.Float64(),
		}); err != nil {
			return err
		}
	}
	// The order store matches trades by ID, so an execution without both IDs cannot be added to its order
	if orderID == "" || tradeID == "" {
		return nil
	}
	feeAsset := settlementCurrency(pair, a)
	d := &order.Detail{
		Exchange:        e.Name,
		OrderID:         orderID,
		ClientOrderID:   t.ClientOrderID,
		AssetType:       a,
		Pair:            pair,
		Side:            side,
		RemainingAmount: t.OrderRemainingAmount.Float64(),
		LastUpdated:     t.Microtimestamp.Time(),
		Trades: []order.TradeHistory{{
			TID:       tradeID,
			Price:     t.Price.Float64(),
			Amount:    t.Amount.Float64(),
			Fee:       t.Fee.Float64(),
			Exchange:  e.Name,
			Side:      side,
			Timestamp: t.Microtimestamp.Time(),
			FeeAsset:  feeAsset.String(),
		}},
	}
	if t.TradeAccountID != 0 {
		d.AccountID = strconv.FormatUint(t.TradeAccountID, 10)
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

func (e *Exchange) handleWSSettlement(ctx context.Context, event string, data []byte) error {
	settlement := &WebsocketSettlement{Event: event}
	if err := json.Unmarshal(data, settlement); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, settlement)
}

func (e *Exchange) handleWSAnnouncement(ctx context.Context, event string, data []byte) error {
	announcement := &WebsocketAnnouncement{Event: event}
	if err := json.Unmarshal(data, announcement); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, announcement)
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return normaliseSubscriptions(e.Features.Subscriptions).ExpandTemplates(e)
}

// normaliseSubscriptions returns subscriptions with the assets of account channels cleared, since those channels are
// not specific to a market and would otherwise be subscribed to once per asset, and with private channels marked as
// authenticated, since they cannot be subscribed to without a token
func normaliseSubscriptions(l subscription.List) subscription.List {
	normalised := make(subscription.List, len(l))
	for i, s := range l {
		normalised[i] = s
		if s.QualifiedChannel != "" {
			continue
		}
		accountChannel := isAccountChannelName(subscriptionNames[s.Channel]) && s.Asset != asset.Empty
		private := slices.Contains(privateChannels, s.Channel) && !s.Authenticated
		if !accountChannel && !private {
			continue
		}
		c := s.Clone()
		if accountChannel {
			c.Asset = asset.Empty
			c.Pairs = nil
		}
		if private {
			c.Authenticated = true
		}
		normalised[i] = c
	}
	return normalised
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isAccountChannel": isAccountChannel,
	}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from a list of channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubsWithCreds(ctx, subs, "sub")
}

// Unsubscribe sends a websocket message to stop receiving data from a list of channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubsWithCreds(ctx, subs, "unsub")
}

// manageSubsWithCreds subscribes to or unsubscribes from channels
// Subscribing to private channels needs a new token, as a token cannot be reused for later subscriptions, whereas
// unsubscribing only needs the user ID which names the private channels, which is kept from the last token
func (e *Exchange) manageSubsWithCreds(ctx context.Context, subs subscription.List, op string) error {
	subs = normaliseSubscriptions(subs)
	var errs error
	var creds *WebsocketTokenResponse
	if slices.ContainsFunc(subs, isPrivateSubscription) {
		if userID := e.websocketUserID.Load(); op == "unsub" && userID != 0 {
			creds = &WebsocketTokenResponse{UserID: userID}
		} else {
			creds, errs = e.GetWebsocketToken(ctx)
			if creds != nil {
				e.websocketUserID.Store(creds.UserID)
			}
		}
	}
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, s subscription.List) error { return e.manageSubs(ctx, s, op, creds) }, 1))
}

// isPrivateSubscription returns whether a subscription needs a token, as it is either marked as authenticated or is to
// a private channel
// Expanded subscriptions are not marked by normaliseSubscriptions, since a marked copy would no longer be the
// subscription that the websocket manager tracks
func isPrivateSubscription(s *subscription.Subscription) bool {
	return s.Authenticated || slices.Contains(privateChannels, s.Channel)
}

func (e *Exchange) manageSubs(ctx context.Context, subs subscription.List, op string, creds *WebsocketTokenResponse) error {
	subs, errs := subs.ExpandTemplates(e)
	for _, s := range subs {
		req := WebsocketEventRequest{
			Event: "bts:" + op + "scribe",
			Data: WebsocketEventData{
				Channel: s.QualifiedChannel,
			},
		}
		if isPrivateSubscription(s) {
			if creds == nil {
				return request.ErrAuthRequestFailed
			}
			req.Data.Channel = privateChannelPrefix + req.Data.Channel + "-" + strconv.FormatUint(creds.UserID, 10)
			req.Data.Auth = creds.Token
		}
		_, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, op+":"+req.Data.Channel, req)
		if err == nil {
			if op == "sub" {
				err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
			} else {
				err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

// splitChannel returns the name and market of a channel, and whether it is private
// Private channels are suffixed with the user ID, and market symbols never contain an underscore
func splitChannel(channel string) (name, market string, private bool, err error) {
	if after, ok := strings.CutPrefix(channel, privateChannelPrefix); ok {
		idx := strings.LastIndexByte(after, '-')
		if idx <= 0 {
			return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
		}
		if _, err := strconv.ParseUint(after[idx+1:], 10, 64); err != nil {
			return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
		}
		channel, private = after[:idx], true
	}
	if isAccountChannelName(channel) {
		return channel, "", private, nil
	}
	idx := strings.LastIndexByte(channel, '_')
	if idx <= 0 || idx == len(channel)-1 {
		return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
	}
	return channel[:idx], channel[idx+1:], private, nil
}

// marketPair returns the currency pair and asset type of a channel market symbol
func (e *Exchange) marketPair(market string) (currency.Pair, asset.Item, error) {
	a := asset.Spot
	if strings.HasSuffix(market, strings.ToLower(perpetualMarketSuffix)) {
		a = asset.PerpetualContract
	}
	pair, err := e.MatchSymbolWithAvailablePairs(market, a, false)
	if err != nil {
		return currency.EMPTYPAIR, asset.Empty, fmt.Errorf("%w %q: %w", errParsingWSPair, market, err)
	}
	return pair, a, nil
}

// channelName converts global channel names to exchange specific ones
// panics if name is not supported, so should be called within a recover chain
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

// isAccountChannel returns whether a subscription is to a channel which is not specific to a market
func isAccountChannel(s *subscription.Subscription) bool {
	return isAccountChannelName(subscriptionNames[s.Channel])
}

func isAccountChannelName(name string) bool {
	return slices.Contains(accountChannels, name)
}

const subTplText = `
{{- if isAccountChannel $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- channelName $.S }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- with $name := channelName $.S }}
			{{- range $p := $pairs -}}
				{{- $name -}} _ {{- $p -}}
				{{ $.PairSeparator }}
			{{- end -}}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- end }}
`
