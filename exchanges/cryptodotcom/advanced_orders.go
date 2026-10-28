package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// maxAdvancedOrderListLength is the most orders an advanced order list request takes
	maxAdvancedOrderListLength = 10
	// maxAdvancedOrderHistoryLimit is the most orders an advanced order history request returns
	maxAdvancedOrderHistoryLimit = 100
)

var (
	errInvalidReferencePrice       = errors.New("reference price must be greater than zero")
	errInvalidReferencePriceType   = errors.New("invalid reference price type")
	errNewPriceRequired            = errors.New("new price required")
	errOrderListTooShort           = errors.New("order list too short")
	errInvalidContingencyOrders    = errors.New("invalid order types for the contingency type")
	errAttachIsolationReduceOnly   = errors.New("an order attached to a position must be REDUCE_ONLY")
	errAttachIsolationConflict     = errors.New("an order attached to a position cannot attach to an order or set an isolation ID")
	errInvalidCancelOrderType      = errors.New("invalid cancel order type")
	errInvalidPositionType         = errors.New("invalid position type")
	errPositionTypeCancelOrderType = errors.New("position type cannot be combined with a LIMIT or TRIGGER cancel order type")
)

// CreateAdvancedOrder places a trigger order. Crypto.com answers on accepting the request, before it places the order;
// the user.advanced.order channel reports the order once placed
func (e *Exchange) CreateAdvancedOrder(ctx context.Context, req *CreateAdvancedOrderRequest) (*OrderIDResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/create-order", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// AmendAdvancedOrder amends a trigger order, and can turn a trigger limit order into a trigger market order or back.
// Crypto.com answers on accepting the request, before it amends the order; the user.advanced.order channel reports the
// amendment
func (e *Exchange) AmendAdvancedOrder(ctx context.Context, req *AmendAdvancedOrderRequest) (*OrderIDResponse, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/amend-order", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateOCO places a one-cancels-the-other strategy, which Crypto.com offers on spot instruments only: a LIMIT order
// and a trigger order, either of which executing, even partly, cancels the other. Crypto.com answers on accepting the
// request, before it places the orders
func (e *Exchange) CreateOCO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := ocoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/create-oco", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOCO cancels an OCO's orders by its list ID; Crypto.com answers on accepting the request, before it cancels them
func (e *Exchange) CancelOCO(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-oco", params, nil)
}

// CreateOTO places a one-triggers-the-other strategy: a LIMIT or MARKET order and a trigger order, which takes effect
// once the first order has fully executed. Crypto.com places it as a SPOT_ATTACH on a spot instrument, which
// CreateAdvancedOrder can attach a further trigger order to, and as a DERIV_ATTACH on a derivative. It answers on
// accepting the request, before it places the orders
func (e *Exchange) CreateOTO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := otoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/create-oto", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOTO cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID, whether it holds two orders or three, as
// CancelOTOCO does; Crypto.com answers on accepting the request, before it cancels them
func (e *Exchange) CancelOTO(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-oto", params, nil)
}

// CreateOTOCO places a one-triggers-a-one-cancels-the-other strategy: a LIMIT or MARKET order, a stop loss and a take
// profit, which take effect once the first order has fully executed. Crypto.com places it as a SPOT_ATTACH on a spot
// instrument, whose stop loss and take profit cancel each other, and as a DERIV_ATTACH on a derivative, whose stop
// loss and take profit are independent. It answers on accepting the request, before it places the orders
func (e *Exchange) CreateOTOCO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := otocoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/create-otoco", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOTOCO cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID, whether it holds two orders or three, as
// CancelOTO does; Crypto.com answers on accepting the request, before it cancels them
func (e *Exchange) CancelOTOCO(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-otoco", params, nil)
}

// CancelAdvancedOrder cancels one order of an advanced order, such as a SPOT_ATTACH's stop loss, leaving the rest of it
// in place. Crypto.com answers on accepting the request, before it cancels the order; the user.advanced.order channel
// reports the cancel
func (e *Exchange) CancelAdvancedOrder(ctx context.Context, req *AdvancedOrderIDRequest) (*OrderIDResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-order", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllAdvancedOrders cancels the advanced orders a request selects; Crypto.com answers on accepting the request,
// before it cancels them
func (e *Exchange) CancelAllAdvancedOrders(ctx context.Context, req *CancelAllAdvancedOrdersRequest) error {
	if err := req.validate(); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-all-orders", req, nil)
}

// GetAdvancedOpenOrders returns the open orders of advanced orders, an entry for each order of an OCO, SPOT_ATTACH or
// DERIV_ATTACH
func (e *Exchange) GetAdvancedOpenOrders(ctx context.Context, req *GetAdvancedOpenOrdersRequest) ([]OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *dataResponse[OrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/get-open-orders", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetAdvancedOrderDetail returns one order of an advanced order
func (e *Exchange) GetAdvancedOrderDetail(ctx context.Context, req *AdvancedOrderIDRequest) (*OrderResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var resp *OrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/get-order-detail", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAdvancedOrderHistory returns orders of advanced orders from the order history; Crypto.com recommends the
// user.advanced.order channel for following orders, and the history for recovering what it missed
func (e *Exchange) GetAdvancedOrderHistory(ctx context.Context, req *GetAdvancedOrderHistoryRequest) ([]OrderResponse, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[OrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/get-order-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// CreateAdvancedOrderList places up to 10 trigger orders, returning each order's outcome by its index in orders.
// Crypto.com answers on accepting the requests, before it places the orders; the user.advanced.order channel reports
// them once placed
func (e *Exchange) CreateAdvancedOrderList(ctx context.Context, orders []CreateAdvancedOrderRequest) ([]OrderListEntryResponse, error) {
	params, err := createAdvancedOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/create-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAdvancedOrderList cancels up to 10 orders of advanced orders, returning each cancel's outcome by its index in
// orders. Crypto.com answers on accepting the requests, before it cancels the orders; the user.advanced.order channel
// reports the cancels
func (e *Exchange) CancelAdvancedOrderList(ctx context.Context, orders []AdvancedOrderIDRequest) ([]OrderListEntryResponse, error) {
	params, err := cancelAdvancedOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/cancel-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAdvancedOrderListDetail returns the orders of OCOs, SPOT_ATTACHes and DERIV_ATTACHes by their list IDs
func (e *Exchange) GetAdvancedOrderListDetail(ctx context.Context, listIDs []string) ([]OrderResponse, error) {
	if len(listIDs) == 0 || slices.Contains(listIDs, "") {
		return nil, errListIDRequired
	}
	var resp *dataResponse[OrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/advanced/get-order-list-detail", &advancedOrderListDetailParams{ListIDs: listIDs}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// validate checks a trigger order has what Crypto.com requires of one, and that an order attached to a position is
// REDUCE_ONLY and attached to nothing else
func (r *CreateAdvancedOrderRequest) validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if !isTriggerOrderType(r.OrderType) {
		return fmt.Errorf("%w: %q is not a trigger order type", errInvalidOrderType, r.OrderType)
	}
	if err := validateAdvancedOrder(r.InstrumentName, r.Side, r.OrderType, r.Price, r.ReferencePrice, r.ReferencePriceType); err != nil {
		return err
	}
	if r.Quantity <= 0 {
		return errInvalidQuantity
	}
	if err := validateOrderInstructions(r.ClientOrderID, r.TimeInForce, r.ExecutionInstructions, r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID); err != nil {
		return err
	}
	if r.AttachIsolationID == "" {
		return nil
	}
	if !slices.Contains(r.ExecutionInstructions, ExecutionInstructionReduceOnly) {
		return errAttachIsolationReduceOnly
	}
	if r.AttachOrderID != "" || r.IsolationID != "" {
		return errAttachIsolationConflict
	}
	return nil
}

// validate checks an order of an OCO, OTO or OTOCO has what Crypto.com requires of its type; a MARKET, STOP_LOSS or
// TAKE_PROFIT buy may spend a notional in place of a quantity
func (r *AdvancedOrderLegRequest) validate() error {
	if err := validateAdvancedOrder(r.InstrumentName, r.Side, r.OrderType, r.Price, r.ReferencePrice, r.ReferencePriceType); err != nil {
		return err
	}
	if err := validateOrderInstructions(r.ClientOrderID, r.TimeInForce, r.ExecutionInstructions, r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID); err != nil {
		return err
	}
	if r.Quantity > 0 {
		return nil
	}
	switch r.OrderType {
	case OrderTypeMarket, OrderTypeStopLoss, OrderTypeTakeProfit:
		if r.Side == SideBuy && r.Notional > 0 {
			return nil
		}
	}
	return errInvalidQuantity
}

// validateAdvancedOrder checks what trigger orders and the orders of an OCO, OTO or OTOCO need alike: an instrument, a
// side, a known order type, a limit price for LIMIT, STOP_LIMIT and TAKE_PROFIT_LIMIT orders, a reference price for
// trigger orders, and a known reference price type when one is set
func validateAdvancedOrder(instrumentName, side, orderType string, price, referencePrice types.Number, referencePriceType string) error {
	if instrumentName == "" {
		return errInstrumentNameRequired
	}
	if side != SideBuy && side != SideSell {
		return fmt.Errorf("%w: %q", errInvalidSide, side)
	}
	switch orderType {
	case OrderTypeLimit, OrderTypeStopLimit, OrderTypeTakeProfitLimit:
		if price <= 0 {
			return fmt.Errorf("%w: a %s order needs a limit price", errInvalidPrice, orderType)
		}
	case OrderTypeMarket, OrderTypeStopLoss, OrderTypeTakeProfit:
	default:
		return fmt.Errorf("%w: %q", errInvalidOrderType, orderType)
	}
	if isTriggerOrderType(orderType) && referencePrice <= 0 {
		return fmt.Errorf("%w: a %s order needs a reference price", errInvalidReferencePrice, orderType)
	}
	switch referencePriceType {
	case "", ReferencePriceTypeMarkPrice, ReferencePriceTypeIndexPrice, ReferencePriceTypeLastPrice:
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidReferencePriceType, referencePriceType)
}

// isTriggerOrderType reports whether an order type is a trigger order's, which Crypto.com places once its reference
// price is reached
func isTriggerOrderType(orderType string) bool {
	switch orderType {
	case OrderTypeStopLoss, OrderTypeStopLimit, OrderTypeTakeProfit, OrderTypeTakeProfitLimit:
		return true
	}
	return false
}

// params validates an amendment and returns its params
func (r *AmendAdvancedOrderRequest) params() (*amendAdvancedOrderParams, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return nil, errOrderIDRequired
	}
	if r.NewQuantity <= 0 {
		return nil, errInvalidQuantity
	}
	if r.NewPrice == nil {
		return nil, errNewPriceRequired
	}
	return &amendAdvancedOrderParams{
		OrderID:           r.OrderID,
		ClientOrderID:     r.ClientOrderID,
		NewQuantity:       r.NewQuantity,
		NewPrice:          r.NewPrice.String(),
		NewReferencePrice: r.NewReferencePrice,
	}, nil
}

// contingencyOrderCounts counts an OCO's, OTO's or OTOCO's orders by type: LIMIT, MARKET, stop loss (STOP_LOSS or
// STOP_LIMIT) and take profit (TAKE_PROFIT or TAKE_PROFIT_LIMIT) orders
type contingencyOrderCounts struct {
	limit, market, stopLoss, takeProfit int
}

// validateContingencyOrders checks an OCO, OTO or OTOCO has its number of orders, each valid, and counts them by type
func validateContingencyOrders(orders []AdvancedOrderLegRequest, contingencyType string, count int) (*contingencyOrderCounts, error) {
	switch {
	case len(orders) == 0:
		return nil, errOrderListEmpty
	case len(orders) < count:
		return nil, fmt.Errorf("%w: an %s takes %d orders, not %d", errOrderListTooShort, contingencyType, count, len(orders))
	case len(orders) > count:
		return nil, fmt.Errorf("%w: an %s takes %d orders, not %d", errOrderListTooLong, contingencyType, count, len(orders))
	}
	var counts contingencyOrderCounts
	for i := range orders {
		if err := orders[i].validate(); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		switch orders[i].OrderType {
		case OrderTypeLimit:
			counts.limit++
		case OrderTypeMarket:
			counts.market++
		case OrderTypeStopLoss, OrderTypeStopLimit:
			counts.stopLoss++
		default:
			counts.takeProfit++
		}
	}
	return &counts, nil
}

// ocoParams validates an OCO's orders, a LIMIT order and a trigger order, which compares its reference price with the
// mark price only, and returns its params
func ocoParams(orders []AdvancedOrderLegRequest) (*advancedOrderListParams[AdvancedOrderLegRequest], error) {
	counts, err := validateContingencyOrders(orders, ContingencyTypeOCO, 2)
	if err != nil {
		return nil, err
	}
	if counts.limit != 1 || counts.stopLoss+counts.takeProfit != 1 {
		return nil, fmt.Errorf("%w: an OCO takes a LIMIT order and a STOP_LOSS, STOP_LIMIT, TAKE_PROFIT or TAKE_PROFIT_LIMIT order", errInvalidContingencyOrders)
	}
	for i := range orders {
		if isTriggerOrderType(orders[i].OrderType) && orders[i].ReferencePriceType != "" && orders[i].ReferencePriceType != ReferencePriceTypeMarkPrice {
			return nil, fmt.Errorf("%w: an OCO's trigger order compares the mark price only, not %s", errInvalidReferencePriceType, orders[i].ReferencePriceType)
		}
	}
	return &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: orders}, nil
}

// otoParams validates an OTO's orders, a LIMIT or MARKET order and a trigger order, and returns its params
func otoParams(orders []AdvancedOrderLegRequest) (*advancedOrderListParams[AdvancedOrderLegRequest], error) {
	counts, err := validateContingencyOrders(orders, ContingencyTypeOTO, 2)
	if err != nil {
		return nil, err
	}
	if counts.limit+counts.market != 1 || counts.stopLoss+counts.takeProfit != 1 {
		return nil, fmt.Errorf("%w: an OTO takes a LIMIT or MARKET order and a STOP_LOSS, STOP_LIMIT, TAKE_PROFIT or TAKE_PROFIT_LIMIT order", errInvalidContingencyOrders)
	}
	return &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: orders}, nil
}

// otocoParams validates an OTOCO's orders, a LIMIT or MARKET order, a stop loss and a take profit, and returns its
// params
func otocoParams(orders []AdvancedOrderLegRequest) (*advancedOrderListParams[AdvancedOrderLegRequest], error) {
	counts, err := validateContingencyOrders(orders, ContingencyTypeOTOCO, 3)
	if err != nil {
		return nil, err
	}
	if counts.limit+counts.market != 1 || counts.stopLoss != 1 || counts.takeProfit != 1 {
		return nil, fmt.Errorf("%w: an OTOCO takes a LIMIT or MARKET order, a STOP_LOSS or STOP_LIMIT order and a TAKE_PROFIT or TAKE_PROFIT_LIMIT order", errInvalidContingencyOrders)
	}
	return &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: orders}, nil
}

// newAdvancedListIDParams returns the params naming an OCO, OTO or OTOCO by its list ID
func newAdvancedListIDParams(listID string) (*advancedListIDParams, error) {
	if listID == "" {
		return nil, errListIDRequired
	}
	return &advancedListIDParams{ListID: listID}, nil
}

// validate checks an advanced order is identified by its order ID or its client order ID
func (r *AdvancedOrderIDRequest) validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return errOrderIDRequired
	}
	return nil
}

// validate checks a cancel-all request's order and position types, and that a position type comes with its
// instrument and without a LIMIT or TRIGGER order type
func (r *CancelAllAdvancedOrdersRequest) validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch r.OrderType {
	case "", CancelOrderTypeLimit, CancelOrderTypeTrigger, CancelOrderTypeAll:
	default:
		return fmt.Errorf("%w: %q", errInvalidCancelOrderType, r.OrderType)
	}
	switch r.PositionType {
	case "":
		return nil
	case PositionTypeCrossMargin, PositionTypeIsolatedMargin:
	default:
		return fmt.Errorf("%w: %q", errInvalidPositionType, r.PositionType)
	}
	if r.InstrumentName == "" {
		return fmt.Errorf("%w: a position type needs one", errInstrumentNameRequired)
	}
	if r.OrderType == CancelOrderTypeLimit || r.OrderType == CancelOrderTypeTrigger {
		return fmt.Errorf("%w: %s", errPositionTypeCancelOrderType, r.OrderType)
	}
	return nil
}

// params validates an order history request and returns its params, its times in the Unix nanoseconds Crypto.com
// recommends for paging
func (r *GetAdvancedOrderHistoryRequest) params() (*advancedOrderHistoryParams, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	if r.Limit > maxAdvancedOrderHistoryLimit {
		return nil, fmt.Errorf("%w: limit %d exceeds the maximum of %d", errInvalidCount, r.Limit, maxAdvancedOrderHistoryLimit)
	}
	startTime, err := unixNanoParam(r.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := unixNanoParam(r.EndTime)
	if err != nil {
		return nil, err
	}
	return &advancedOrderHistoryParams{InstrumentName: r.InstrumentName, StartTime: startTime, EndTime: endTime, Limit: r.Limit}, nil
}

// createAdvancedOrderListParams validates an advanced order list's trigger orders and returns its params
func createAdvancedOrderListParams(orders []CreateAdvancedOrderRequest) (*advancedOrderListParams[CreateAdvancedOrderRequest], error) {
	if err := validateAdvancedOrderListLength(len(orders)); err != nil {
		return nil, err
	}
	for i := range orders {
		if err := orders[i].validate(); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
	}
	return &advancedOrderListParams[CreateAdvancedOrderRequest]{OrderList: orders}, nil
}

// cancelAdvancedOrderListParams validates the orders an advanced cancel list identifies and returns its params
func cancelAdvancedOrderListParams(orders []AdvancedOrderIDRequest) (*advancedOrderListParams[AdvancedOrderIDRequest], error) {
	if err := validateAdvancedOrderListLength(len(orders)); err != nil {
		return nil, err
	}
	for i := range orders {
		if err := orders[i].validate(); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
	}
	return &advancedOrderListParams[AdvancedOrderIDRequest]{OrderList: orders}, nil
}

// validateAdvancedOrderListLength checks an advanced order list holds at least one order and at most
// maxAdvancedOrderListLength
func validateAdvancedOrderListLength(length int) error {
	if length == 0 {
		return errOrderListEmpty
	}
	if length > maxAdvancedOrderListLength {
		return fmt.Errorf("%w: %d orders exceed the maximum of %d", errOrderListTooLong, length, maxAdvancedOrderListLength)
	}
	return nil
}
