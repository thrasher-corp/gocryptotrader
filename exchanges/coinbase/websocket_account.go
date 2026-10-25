package coinbase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	// orderSizeLookupBatchSize is the most orders one lookup of the sizes the user channel leaves out names
	orderSizeLookupBatchSize = 50
	// orderSizeLifetime is how long an order's recorded size is kept, since it was recorded or the user channel last
	// reported the order live, once the record holds orderSizesPruneThreshold orders, as an order whose close the user
	// channel never reported would otherwise stay; an order still live after it is looked up again
	orderSizeLifetime        = 30 * 24 * time.Hour
	orderSizesPruneThreshold = 1000
	// closedOrderSizeLifetime is how long the size an order closed with is kept once the record holds
	// orderSizesPruneThreshold closed orders, for a report of the order that comes after its close
	closedOrderSizeLifetime = time.Hour
	// orderSizeTolerance is the relative difference within which two sizes of an order are the same, as one summed from
	// the user channel's quantities can differ from the one Coinbase holds by the floats' rounding
	orderSizeTolerance = 1e-12
)

var (
	errUnknownProductType = errors.New("unknown product type")
	errUnknownTimeInForce = errors.New("unknown time in force")
)

// wsHandleUser relays the user channel's orders, and its futures positions where the account has them enabled; open
// orders arrive in snapshots of up to 50, then each change in an update. A live order whose size is not yet known, or
// whose recorded size a report disputes, is looked up, and relayed again with its size
func (e *Exchange) wsHandleUser(ctx context.Context, envelope *WebsocketEnvelope) error {
	var events []WebsocketUserEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var errs error
	var reported []order.Detail
	// The orders are relayed in order with a lookup's, so a lookup's report cannot follow a newer one
	e.orderSizes.relayMu.Lock()
	defer e.orderSizes.relayMu.Unlock()
	for i := range events {
		details := make([]order.Detail, 0, len(events[i].Orders))
		for j := range events[i].Orders {
			detail, err := e.websocketOrderDetail(&events[i].Orders[j], envelope)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			size, _ := openOrderSize(&events[i].Orders[j])
			e.orderSizes.observe(detail, size)
			reported = append(reported, *detail)
			details = append(details, *detail)
		}
		if len(details) != 0 {
			errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, details))
		}
		positions := &events[i].Positions
		if len(positions.PerpetualFuturesPositions) != 0 || len(positions.ExpiringFuturesPositions) != 0 {
			errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, positions))
		}
	}
	e.wsLookUpOrderSizes(ctx, latestLive(reported))
	return errs
}

// latestLive returns the orders whose latest report in a message is live, in the order reported; a message can report
// an order more than once, and its lookup relays the report it began with
func latestLive(reported []order.Detail) []order.Detail {
	latest := make(map[string]int, len(reported))
	for i := range reported {
		latest[reported[i].OrderID] = i
	}
	var live []order.Detail
	for i := range reported {
		if latest[reported[i].OrderID] == i && isLiveOrderStatus(reported[i].Status) {
			live = append(live, reported[i])
		}
	}
	return live
}

// wsLookUpOrderSizes lists the reported live orders whose sizes are unknown or disputed, in the background so the reader
// goes on, and relays the channel's latest report of each with its size, which is recorded for the order's later
// updates. The lookup takes only the size from the list, whose state can be older than the channel's; an order the
// channel reports closed meanwhile is not relayed again, and an order already being looked up is left to that lookup
func (e *Exchange) wsLookUpOrderSizes(ctx context.Context, reported []order.Detail) {
	claimed, started := e.orderSizes.claim(reported)
	if len(claimed) == 0 {
		return
	}
	go func() {
		defer e.orderSizes.release(claimed, started)
		for batch := range slices.Chunk(claimed, orderSizeLookupBatchSize) {
			orders, err := e.listOrders(ctx, &ListOrdersRequest{OrderIDs: batch})
			if err != nil {
				if ctx.Err() == nil {
					_ = e.Websocket.DataHandler.Send(ctx, fmt.Errorf("%s websocket could not look up the sizes of orders %q: %w", e.Name, batch, err))
				}
				return
			}
			e.relayLookedUpOrders(ctx, orders)
		}
	}()
}

// relayLookedUpOrders relays the channel's latest reports of orders looked up with their sizes, holding the user
// channel's relays meanwhile, so that a report the channel relays next follows them. An order an open report since its
// lookup began gave another size is looked up again, as the list may predate an edit the report shows
func (e *Exchange) relayLookedUpOrders(ctx context.Context, orders []Order) {
	e.orderSizes.relayMu.Lock()
	defer e.orderSizes.relayMu.Unlock()
	details := make([]order.Detail, 0, len(orders))
	var again []order.Detail
	for i := range orders {
		values := orderConfigurationValues(orders[i].OrderConfiguration)
		d, outcome := e.orderSizes.complete(orders[i].OrderID, values.baseSize, values.quoteSize)
		switch outcome {
		case lookupRelay:
			details = append(details, d)
		case lookupAgain:
			again = append(again, d)
		}
	}
	if len(details) != 0 {
		_ = e.Websocket.DataHandler.Send(ctx, details)
	}
	e.wsLookUpOrderSizes(ctx, again)
}

// isLiveOrderStatus reports whether an order with a status can still trade
func isLiveOrderStatus(s order.Status) bool {
	return s != order.UnknownStatus && !s.IsInactive()
}

// isClosedOrderStatus reports whether an order with a status has closed; an unknown status is neither live nor closed
func isClosedOrderStatus(s order.Status) bool {
	return s != order.UnknownStatus && s.IsInactive()
}

// advancedTradeOrderSizes holds the sizes of the account's live Advanced Trade orders by order ID, each in the currency
// it was placed in, the orders being looked up, and the sizes the orders it held closed with
type advancedTradeOrderSizes struct {
	m       sync.Mutex
	sizes   map[string]advancedTradeOrderSize
	pending map[string]*orderSizeLookup
	closed  map[string]closedOrderSize
	// stamps orders the record's writes and the lists and lookups that read sizes, as clock readings can be equal
	stamps uint64
	// relayMu orders the user channel's relays of orders and a lookup's
	relayMu sync.Mutex
}

// orderSizeLookup is a lookup of an order's size, the stamp it began at, the user channel's latest report of the order,
// and the size of an order sized in the base currency that the channel's reports of it open have since given
type orderSizeLookup struct {
	started  uint64
	latest   order.Detail
	reported float64
}

// advancedTradeOrderSize is an order's size, in its base or quote currency, when it was recorded, its stamp, and whether
// a user channel report disputes it
type advancedTradeOrderSize struct {
	base, quote float64
	recorded    time.Time
	stamp       uint64
	disputed    bool
}

// closedOrderSize is the base size an order closed with, and when its close was recorded
type closedOrderSize struct {
	base   float64
	closed time.Time
}

// listStamp is when an order list began: its stamp orders the list's sizes against sizes recorded since, and its
// time is recorded as theirs
type listStamp struct {
	order uint64
	begun time.Time
}

// stamp returns the stamp of an order list begun now
func (s *advancedTradeOrderSizes) stamp() listStamp {
	s.m.Lock()
	defer s.m.Unlock()
	s.stamps++
	return listStamp{order: s.stamps, begun: time.Now()}
}

// set records an order's size
func (s *advancedTradeOrderSizes) set(orderID string, base, quote float64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.prune()
	if s.sizes == nil {
		s.sizes = make(map[string]advancedTradeOrderSize)
	}
	s.stamps++
	s.sizes[orderID] = advancedTradeOrderSize{base: base, quote: quote, recorded: time.Now(), stamp: s.stamps}
}

// prune forgets the sizes kept longest once the record holds many, as it does the sizes closed orders closed with; the
// caller holds s.m
func (s *advancedTradeOrderSizes) prune() {
	now := time.Now()
	if len(s.sizes) >= orderSizesPruneThreshold {
		for id, size := range s.sizes {
			if now.Sub(size.recorded) > orderSizeLifetime {
				delete(s.sizes, id)
			}
		}
	}
	if len(s.closed) >= orderSizesPruneThreshold {
		for id, size := range s.closed {
			if now.Sub(size.closed) > closedOrderSizeLifetime {
				delete(s.closed, id)
			}
		}
	}
}

// setListed records an order's size as an order list begun at a stamp reported it, unless a size was recorded since,
// such as by an edit, which is newer than the list's. A dispute stands, as the report raising it can be newer than the
// list
func (s *advancedTradeOrderSizes) setListed(orderID string, base, quote float64, listed listStamp) {
	s.m.Lock()
	defer s.m.Unlock()
	size, ok := s.sizes[orderID]
	if ok && size.stamp > listed.order {
		return
	}
	s.prune()
	if s.sizes == nil {
		s.sizes = make(map[string]advancedTradeOrderSize)
	}
	s.sizes[orderID] = advancedTradeOrderSize{base: base, quote: quote, recorded: listed.begun, stamp: listed.order, disputed: size.disputed}
}

// get returns an order's recorded size
func (s *advancedTradeOrderSizes) get(orderID string) (advancedTradeOrderSize, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	size, ok := s.sizes[orderID]
	return size, ok
}

// remove forgets a closed order's size, keeping the base size it closed with, when the record held the order, for a
// report of it that comes after its close
func (s *advancedTradeOrderSizes) remove(orderID string, base float64) {
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.sizes[orderID]; !ok {
		return
	}
	delete(s.sizes, orderID)
	if !(base > 0) {
		return
	}
	s.prune()
	if s.closed == nil {
		s.closed = make(map[string]closedOrderSize)
	}
	s.closed[orderID] = closedOrderSize{base: base, closed: time.Now()}
}

// closedSize returns the base size a closed order closed with, or zero when none was kept
func (s *advancedTradeOrderSizes) closedSize(orderID string) float64 {
	s.m.Lock()
	defer s.m.Unlock()
	return s.closed[orderID].base
}

// claim begins looking up the orders reported that are not being looked up and have no size recorded, or a disputed
// one, and returns their IDs and the stamp the lookup began at
func (s *advancedTradeOrderSizes) claim(reported []order.Detail) (claimed []string, started uint64) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.pending == nil {
		s.pending = make(map[string]*orderSizeLookup)
	}
	s.stamps++
	for i := range reported {
		id := reported[i].OrderID
		size, recorded := s.sizes[id]
		_, pending := s.pending[id]
		if (!recorded || size.disputed) && !pending {
			s.pending[id] = &orderSizeLookup{started: s.stamps, latest: reported[i]}
			claimed = append(claimed, id)
		}
	}
	return claimed, s.stamps
}

// dispute marks an order's recorded size as disputed by a user channel report, so that the order is looked up
func (s *advancedTradeOrderSizes) dispute(orderID string) {
	s.m.Lock()
	defer s.m.Unlock()
	if size, ok := s.sizes[orderID]; ok {
		size.disputed = true
		s.sizes[orderID] = size
	}
}

// observe keeps the user channel's latest report of an order being looked up, and the size it gives the order were it
// sized in the base currency, when it gives one. A live report keeps a recorded size as if recorded anew, so that it is
// not pruned as an order whose close went unreported
func (s *advancedTradeOrderSizes) observe(d *order.Detail, reported float64) {
	s.m.Lock()
	defer s.m.Unlock()
	if size, ok := s.sizes[d.OrderID]; ok && isLiveOrderStatus(d.Status) {
		size.recorded = time.Now()
		s.sizes[d.OrderID] = size
	}
	if lookup, ok := s.pending[d.OrderID]; ok {
		lookup.latest = *d
		if reported > 0 {
			lookup.reported = reported
		}
	}
}

// lookupOutcome is what a completed lookup does with the order it looked up
type lookupOutcome uint8

const (
	// lookupDrop relays nothing, as the order was not being looked up or the channel reported it closed meanwhile
	lookupDrop lookupOutcome = iota
	// lookupRelay relays the channel's latest report of the order with its size
	lookupRelay
	// lookupAgain looks the order up again, as an open report since the lookup began gave another size than the list
	lookupAgain
)

// complete ends an order's lookup with the size listed, returning the user channel's latest report of the order and
// whether to relay it with its size, drop it, as the channel reported the order closed meanwhile, or look the order up
// again. A size recorded since the lookup began, by an edit, is newer than the one listed; an open report since then
// that gives a base sized order another size than the list's means one of them predates an edit, so neither is taken
func (s *advancedTradeOrderSizes) complete(orderID string, base, quote float64) (order.Detail, lookupOutcome) {
	s.m.Lock()
	defer s.m.Unlock()
	lookup, ok := s.pending[orderID]
	if !ok || isClosedOrderStatus(lookup.latest.Status) {
		return order.Detail{}, lookupDrop
	}
	delete(s.pending, orderID)
	if size, ok := s.sizes[orderID]; ok && size.stamp > lookup.started {
		base, quote = size.base, size.quote
	} else {
		if quote == 0 && base > 0 && lookup.reported > 0 && !sameOrderSize(lookup.reported, base) {
			// Disputed, so that its repeat claims the order though a list recorded its size meanwhile
			if size, ok := s.sizes[orderID]; ok {
				size.disputed = true
				s.sizes[orderID] = size
			}
			return lookup.latest, lookupAgain
		}
		s.prune()
		if s.sizes == nil {
			s.sizes = make(map[string]advancedTradeOrderSize)
		}
		s.stamps++
		s.sizes[orderID] = advancedTradeOrderSize{base: base, quote: quote, recorded: time.Now(), stamp: s.stamps}
	}
	d := lookup.latest
	d.Amount, d.QuoteAmount = base, quote
	if base > 0 {
		d.RemainingAmount = max(base-d.ExecutedAmount, 0)
	}
	return d, lookupRelay
}

// release ends the lookup of orders begun at a stamp, leaving a lookup of one begun since, such as one looking it up
// again
func (s *advancedTradeOrderSizes) release(orderIDs []string, started uint64) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, id := range orderIDs {
		if lookup, ok := s.pending[id]; ok && lookup.started == started {
			delete(s.pending, id)
		}
	}
}

// websocketOrderDetail returns a user channel order as an order detail. The channel reports an order's remaining
// quantity in the currency it was sized in, base or quote, without saying which, and not its size, so the detail takes
// its size from the order's recorded size, when there is one, and reports the base amount executed. A closed order's
// close time is when it ended, or else when the channel reported it closed; a live order's end time is when it expires
func (e *Exchange) websocketOrderDetail(o *WebsocketUserOrder, envelope *WebsocketEnvelope) (*order.Detail, error) {
	a, err := productTypeAsset(o.ProductType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	pair, err := e.productPair(o.ProductID, a)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	side, err := order.StringToOrderSide(o.OrderSide)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	orderType, err := standardOrderType(o.OrderType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	status, err := standardOrderStatus(o.Status, o.CumulativeQuantity.Float64())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	tif, err := standardTimeInForce(o.TimeInForce, o.PostOnly.Bool())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	var amount, remaining, quoteAmount float64
	if size, ok := e.orderSizes.get(o.OrderID); ok {
		amount, quoteAmount = size.base, size.quote
		switch {
		case amount > 0 && quoteAmount == 0:
			remaining = o.LeavesQuantity.Float64()
			// An edit made elsewhere, or one reported before its response, changes the size an open order's
			// quantities give, but so does a report older than an edit's response, so the order is looked up
			if reported, ok := openOrderSize(o); ok && !sameOrderSize(reported, amount) {
				e.orderSizes.dispute(o.OrderID)
			}
			// A filled order that executed more than its recorded size was enlarged by an edit made elsewhere; one that
			// executed less keeps its size, as Coinbase can mark an order filled that leaves dust unfilled
			if executed := o.CumulativeQuantity.Float64(); o.Status == OrderStatusFilled && executed > amount && resizableOrderType(o.OrderType) {
				amount = executed
			}
		case amount > 0:
			// An order sized in the quote currency reports what remains of its quote size, so what remains of its base
			// size is what it has not executed
			remaining = max(amount-o.CumulativeQuantity.Float64(), 0)
		}
	}
	if amount == 0 && o.Status == OrderStatusFilled {
		// An order filled without a base size recorded, as one first reported filled, such as one placed elsewhere that
		// filled at once, or one sized in the quote currency alone, takes the base amount it executed as its size. One
		// reported filled once its close is recorded, as by a repeated report or a list that read it closed, keeps the
		// size it closed with where that is larger, as Coinbase can mark an order filled leaving dust unfilled
		amount = max(e.orderSizes.closedSize(o.OrderID), o.CumulativeQuantity.Float64())
	}
	var closeTime time.Time
	if isClosedOrderStatus(status) {
		e.orderSizes.remove(o.OrderID, amount)
		closeTime = envelope.Timestamp
		if !o.EndTime.IsZero() && !o.EndTime.After(closeTime) {
			closeTime = o.EndTime
		}
	}
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.ClientOrderID,
		AccountID:            o.RetailPortfolioID,
		Pair:                 pair,
		AssetType:            a,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		Price:                o.LimitPrice.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		Amount:               amount,
		QuoteAmount:          quoteAmount,
		RemainingAmount:      remaining,
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.CumulativeQuantity.Float64(),
		Cost:                 o.FilledValue.Float64(),
		Fee:                  o.TotalFees.Float64(),
		Date:                 o.CreationTime,
		CloseTime:            closeTime,
		LastUpdated:          envelope.Timestamp,
	}, nil
}

// resizableOrderType reports whether an edit can change an Advanced Trade order type's size, which only limit orders,
// whether stop limit, bracket or take profit/stop loss, take
func resizableOrderType(orderType string) bool {
	switch orderType {
	case OrderTypeLimit, OrderTypeStopLimit, OrderTypeBracket, OrderTypeTakeProfitStopLoss:
		return true
	}
	return false
}

// openOrderSize returns the size an order sized in the base currency has by a user channel report of it open: its
// executed and remaining quantities, which the channel gives in its base currency. It reports false for an order type
// an edit cannot resize, whose quantities need not give its size, for a report of any other status, as a pending
// order's or a queued edit's remaining quantity can predate the order's size, and for one with nothing remaining.
// Summing the quantities as decimals gives the size Coinbase holds, where their floats' sum can be off by its rounding
func openOrderSize(o *WebsocketUserOrder) (float64, bool) {
	executed, remaining := o.CumulativeQuantity.Float64(), o.LeavesQuantity.Float64()
	if !resizableOrderType(o.OrderType) || o.Status != OrderStatusOpen || !(remaining > 0) || !(executed >= 0) || math.IsInf(remaining, 0) || math.IsInf(executed, 0) {
		return 0, false
	}
	return decimal.MustFromFloat(executed).Add(decimal.MustFromFloat(remaining)).InexactFloat64(), true
}

// sameOrderSize reports whether two sizes of an order are the same, within orderSizeTolerance
func sameOrderSize(a, b float64) bool {
	return math.Abs(a-b) <= orderSizeTolerance*math.Max(math.Abs(a), math.Abs(b))
}

// productTypeAsset returns the asset an Advanced Trade product type trades as
func productTypeAsset(productType string) (asset.Item, error) {
	switch productType {
	case ProductTypeSpot:
		return asset.Spot, nil
	case ProductTypeFuture:
		return asset.Futures, nil
	default:
		return asset.Empty, fmt.Errorf("%w: %q", errUnknownProductType, productType)
	}
}

// standardOrderType returns an Advanced Trade order type as an order type: an attached take profit/stop loss order is a
// one cancels the other order, and a roll's steps and a scaled order have no order type of their own
func standardOrderType(orderType string) (order.Type, error) {
	switch orderType {
	case OrderTypeLimit:
		return order.Limit, nil
	case OrderTypeMarket:
		return order.Market, nil
	case OrderTypeStopLimit:
		return order.StopLimit, nil
	case OrderTypeStop:
		return order.Stop, nil
	case OrderTypeBracket:
		return order.Bracket, nil
	case OrderTypeTWAP:
		return order.TWAP, nil
	case OrderTypeLiquidation:
		return order.Liquidation, nil
	case OrderTypeTakeProfitStopLoss:
		return order.OCO, nil
	case "", OrderTypeUnknown, OrderTypeRollOpen, OrderTypeRollClose, OrderTypeScaled:
		return order.UnknownType, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// standardOrderStatus returns an Advanced Trade order status as an order status; an open or cancelled order that has
// started filling is partially filled, and an order whose edit is queued stays open
func standardOrderStatus(status string, executed float64) (order.Status, error) {
	switch status {
	case OrderStatusPending, OrderStatusQueued:
		return order.Pending, nil
	case OrderStatusOpen, OrderStatusEditQueued:
		if executed > 0 {
			return order.PartiallyFilled, nil
		}
		return order.Open, nil
	case OrderStatusFilled:
		return order.Filled, nil
	case OrderStatusCancelQueued:
		return order.PendingCancel, nil
	case OrderStatusCancelled:
		if executed > 0 {
			return order.PartiallyFilledCancelled, nil
		}
		return order.Cancelled, nil
	case OrderStatusExpired:
		return order.Expired, nil
	case OrderStatusFailed:
		return order.Rejected, nil
	case OrderStatusUnknown:
		return order.UnknownStatus, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", order.ErrUnsupportedStatusType, status)
	}
}

// standardTimeInForce returns an Advanced Trade time in force as a time in force, with post only where it is set
func standardTimeInForce(timeInForce string, postOnly bool) (order.TimeInForce, error) {
	var tif order.TimeInForce
	switch timeInForce {
	case TimeInForceGoodUntilCancelled:
		tif = order.GoodTillCancel
	case TimeInForceGoodUntilDateTime:
		tif = order.GoodTillTime
	case TimeInForceImmediateOrCancel:
		tif = order.ImmediateOrCancel
	case TimeInForceFillOrKill:
		tif = order.FillOrKill
	case "", TimeInForceUnknown:
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %q", errUnknownTimeInForce, timeInForce)
	}
	if postOnly {
		tif |= order.PostOnly
	}
	return tif, nil
}

// wsHandleFuturesBalanceSummary relays the account's US derivatives balances
func (e *Exchange) wsHandleFuturesBalanceSummary(ctx context.Context, envelope *WebsocketEnvelope) error {
	var events []WebsocketFuturesBalanceSummaryEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var errs error
	for i := range events {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, &events[i].FCMBalanceSummary))
	}
	return errs
}
