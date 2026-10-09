package kraken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errClientOrderIDWithUserReference = errors.New("client order ID cannot be sent with a user reference")
	errOrderIdentifierRequired        = errors.New("one kind of order identifier is required")
	errBatchSize                      = errors.New("invalid batch size")
	errInvalidTimeout                 = errors.New("invalid timeout")
)

// WsAddOrder calls add_order, placing an order on the private websocket connection. Its updates are streamed on the
// executions channel
func (e *Exchange) WsAddOrder(ctx context.Context, req *WsAddOrderRequest) (*WsAddOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	orderParams, err := req.Order.params()
	if err != nil {
		return nil, err
	}
	params := &wsAddOrderParams{
		wsOrderParams: *orderParams,
		Symbol:        e.displaySymbol(req.Pair),
		Validate:      req.Validate,
	}
	if !req.Deadline.IsZero() {
		params.Deadline = req.Deadline.UTC().Format(time.RFC3339Nano)
	}
	resp, err := e.wsPrivateRequest(ctx, "add_order", params)
	if err != nil {
		return nil, err
	}
	result := &WsAddOrderResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// WsEditOrder calls edit_order, replacing a live order with a new order carrying the changed parameters and a new
// order ID. WsAmendOrder changes an order in place, keeping its ID and, where possible, its queue priority
func (e *Exchange) WsEditOrder(ctx context.Context, req *WsEditOrderRequest) (*WsEditOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := &wsEditOrderParams{
		OrderID:            req.OrderID,
		Symbol:             e.displaySymbol(req.Pair),
		OrderQuantity:      req.Quantity,
		LimitPrice:         req.LimitPrice,
		DisplayQuantity:    req.DisplayQuantity,
		FeePreference:      req.FeePreference,
		PostOnly:           req.PostOnly,
		ReduceOnly:         req.ReduceOnly,
		Triggers:           req.Triggers,
		OrderUserReference: req.UserReference,
		Validate:           req.Validate,
	}
	if !req.Deadline.IsZero() {
		params.Deadline = req.Deadline.UTC().Format(time.RFC3339Nano)
	}
	resp, err := e.wsPrivateRequest(ctx, "edit_order", params)
	if err != nil {
		return nil, err
	}
	result := &WsEditOrderResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// WsAmendOrder calls amend_order, changing an order's quantity, display quantity, limit price or trigger price in place,
// keeping its identifiers and, where possible, its queue priority. A quantity below the filled quantity cancels the
// remainder
func (e *Exchange) WsAmendOrder(ctx context.Context, req *WsAmendOrderRequest) (*WsAmendOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if (req.OrderID == "") == (req.ClientOrderID == "") {
		return nil, fmt.Errorf("%w: send either the order ID or the client order ID", errOrderIdentifierRequired)
	}
	params := &wsAmendOrderParams{
		OrderID:          req.OrderID,
		ClientOrderID:    req.ClientOrderID,
		OrderQuantity:    req.Quantity,
		DisplayQuantity:  req.DisplayQuantity,
		LimitPrice:       req.LimitPrice,
		LimitPriceType:   req.LimitPriceType,
		PostOnly:         req.PostOnly,
		TriggerPrice:     req.TriggerPrice,
		TriggerPriceType: req.TriggerPriceType,
	}
	if !req.Pair.IsEmpty() {
		params.Symbol = e.displaySymbol(req.Pair)
	}
	if !req.Deadline.IsZero() {
		params.Deadline = req.Deadline.UTC().Format(time.RFC3339Nano)
	}
	resp, err := e.wsPrivateRequest(ctx, "amend_order", params)
	if err != nil {
		return nil, err
	}
	result := &WsAmendOrderResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// WsCancelOrders calls cancel_order, cancelling orders by one kind of identifier. Kraken replies for each order on its
// own, so each result reports its order's outcome
func (e *Exchange) WsCancelOrders(ctx context.Context, req *WsCancelOrdersRequest) ([]WsCancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var kinds, expected int
	for _, n := range []int{len(req.OrderIDs), len(req.ClientOrderIDs), len(req.UserReferences)} {
		if n != 0 {
			kinds++
			expected = n
		}
	}
	if kinds != 1 {
		return nil, fmt.Errorf("%w: send order IDs, client order IDs or user references", errOrderIdentifierRequired)
	}
	params := &wsCancelOrderParams{
		OrderIDs:            req.OrderIDs,
		ClientOrderIDs:      req.ClientOrderIDs,
		OrderUserReferences: req.UserReferences,
	}
	resps, err := e.wsPrivateRequests(ctx, "cancel_order", params, expected)
	if err != nil {
		return nil, err
	}
	results := make([]WsCancelOrderResponse, len(resps))
	for i := range resps {
		results[i].TimeIn, results[i].TimeOut = resps[i].TimeIn, resps[i].TimeOut
		if !resps[i].Success {
			results[i].Error = &APIError{Errors: []string{resps[i].Error}}
			continue
		}
		if err := json.Unmarshal(resps[i].Result, &results[i]); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// WsCancelAllOrders calls cancel_all, cancelling every open order, including untriggered ones
func (e *Exchange) WsCancelAllOrders(ctx context.Context) (*WsCancelAllOrdersResponse, error) {
	resp, err := e.wsPrivateRequest(ctx, "cancel_all", &wsTokenParams{})
	if err != nil {
		return nil, err
	}
	result := &WsCancelAllOrdersResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// WsCancelAllOrdersAfter calls cancel_all_orders_after, Kraken's dead man's switch: every order is cancelled unless
// another request resets the timer within timeout, which is sent in whole seconds and must be under a day. A zero
// timeout disables the switch. Kraken suggests a 60 second timeout reset every 15 to 30 seconds
func (e *Exchange) WsCancelAllOrdersAfter(ctx context.Context, timeout time.Duration) (*WsCancelAllOrdersAfterResponse, error) {
	if timeout < 0 || (timeout > 0 && timeout < time.Second) || timeout >= 24*time.Hour {
		return nil, fmt.Errorf("%w: %s must be 0 or from a second to under a day", errInvalidTimeout, timeout)
	}
	resp, err := e.wsPrivateRequest(ctx, "cancel_all_orders_after", &wsCancelAfterParams{Timeout: uint64(timeout / time.Second)})
	if err != nil {
		return nil, err
	}
	result := &WsCancelAllOrdersAfterResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// WsBatchAddOrders calls batch_add, placing 2 to 15 orders on one pair. Kraken rejects the whole batch when an order
// fails validation, while an order failing pre-match checks such as funding is rejected alone. The results are in the
// orders' order
func (e *Exchange) WsBatchAddOrders(ctx context.Context, req *WsBatchAddOrdersRequest) ([]WsAddOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Orders) < 2 || len(req.Orders) > 15 {
		return nil, fmt.Errorf("%w: %d orders, 2 to 15 are allowed", errBatchSize, len(req.Orders))
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := &wsBatchAddParams{
		Symbol:   e.displaySymbol(req.Pair),
		Orders:   make([]wsOrderParams, len(req.Orders)),
		Validate: req.Validate,
	}
	for i := range req.Orders {
		o, err := req.Orders[i].params()
		if err != nil {
			return nil, err
		}
		params.Orders[i] = *o
	}
	if !req.Deadline.IsZero() {
		params.Deadline = req.Deadline.UTC().Format(time.RFC3339Nano)
	}
	resp, err := e.wsPrivateRequest(ctx, "batch_add", params)
	if err != nil {
		return nil, err
	}
	var results []WsAddOrderResponse
	if err := json.Unmarshal(resp.Result, &results); err != nil {
		return nil, err
	}
	for i := range results {
		results[i].TimeIn, results[i].TimeOut = resp.TimeIn, resp.TimeOut
	}
	return results, nil
}

// WsBatchCancelOrders calls batch_cancel, cancelling 2 to 50 orders by order ID or user reference, and by client order
// ID
func (e *Exchange) WsBatchCancelOrders(ctx context.Context, req *WsBatchCancelOrdersRequest) (*WsBatchCancelOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if n := len(req.Orders) + len(req.ClientOrderIDs); n < 2 || n > 50 {
		return nil, fmt.Errorf("%w: %d orders, 2 to 50 are allowed", errBatchSize, n)
	}
	resp, err := e.wsPrivateRequest(ctx, "batch_cancel", &wsBatchCancelParams{Orders: req.Orders, ClientOrderIDs: req.ClientOrderIDs})
	if err != nil {
		return nil, err
	}
	result := &WsBatchCancelOrdersResponse{TimeIn: resp.TimeIn, TimeOut: resp.TimeOut}
	return result, json.Unmarshal(resp.Result, result)
}

// params validates an order and returns its request parameters
func (o *WsOrder) params() (*wsOrderParams, error) {
	if o.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if o.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if o.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if o.ClientOrderID != "" && o.UserReference != 0 {
		return nil, errClientOrderIDWithUserReference
	}
	if o.TimeInForce == "gtd" && o.ExpireTime.IsZero() {
		return nil, fmt.Errorf("%w: GTD orders need an expiry", order.ErrInvalidTimeInForce)
	}
	p := &wsOrderParams{
		OrderType:           o.OrderType,
		Side:                o.Side,
		OrderQuantity:       o.Quantity,
		LimitPrice:          o.LimitPrice,
		LimitPriceType:      o.LimitPriceType,
		Triggers:            o.Triggers,
		TimeInForce:         o.TimeInForce,
		Margin:              o.Margin,
		PostOnly:            o.PostOnly,
		ReduceOnly:          o.ReduceOnly,
		ClientOrderID:       o.ClientOrderID,
		OrderUserReference:  o.UserReference,
		Conditional:         o.Conditional,
		DisplayQuantity:     o.DisplayQuantity,
		FeePreference:       o.FeePreference,
		SelfTradePrevention: o.SelfTradePrevention,
		CashOrderQuantity:   o.CashOrderQuantity,
		SenderSubID:         o.SenderSubID,
	}
	if !o.EffectiveTime.IsZero() {
		p.EffectiveTime = o.EffectiveTime.UTC().Format(time.RFC3339)
	}
	if !o.ExpireTime.IsZero() {
		p.ExpireTime = o.ExpireTime.UTC().Format(time.RFC3339)
	}
	return p, nil
}

// wsPrivateRequest sends a request on the private connection with the websocket token added, returning its reply, or
// the error a failed reply carries
func (e *Exchange) wsPrivateRequest(ctx context.Context, method string, params wsTokenSetter) (*wsResponse, error) {
	resps, err := e.wsPrivateRequests(ctx, method, params, 1)
	if err != nil {
		return nil, err
	}
	if !resps[0].Success {
		return nil, &APIError{Errors: []string{resps[0].Error}}
	}
	return &resps[0], nil
}

// wsPrivateRequests sends a request on the private connection with the websocket token added and returns its expected
// number of replies
func (e *Exchange) wsPrivateRequests(ctx context.Context, method string, params wsTokenSetter, expected int) ([]wsResponse, error) {
	conn, err := e.Websocket.GetConnection(wsPrivateConnection)
	if err != nil {
		return nil, err
	}
	token, err := e.websocketToken(ctx, false)
	if err != nil {
		return nil, err
	}
	params.setToken(token)
	req := &wsRequest{Method: method, Params: params, RequestID: e.MessageSequence()}
	raw, err := conn.SendMessageReturnResponses(ctx, request.Unset, req.RequestID, req, expected)
	if err != nil {
		return nil, err
	}
	resps := make([]wsResponse, len(raw))
	for i := range raw {
		if err := json.Unmarshal(raw[i], &resps[i]); err != nil {
			return nil, err
		}
	}
	return resps, nil
}
