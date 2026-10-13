package bybit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Websocket connection message filters
const (
	OutboundTradeConnection  = "PRIVATE_TRADE"
	InboundPrivateConnection = "PRIVATE"
)

// WebsocketAuthenticateTradeConnection authenticates the trade connection (Websocket Trade Guideline: Authentication)
func (e *Exchange) WebsocketAuthenticateTradeConnection(ctx context.Context, conn websocket.Connection) error {
	args, err := e.websocketAuthenticationArguments(ctx)
	if err != nil {
		return err
	}
	// The request carries no request ID, so the reply carries none either and is matched on its operation
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, authOperation, &WebsocketTradeRequest{Operation: authOperation, Arguments: args})
	if err != nil {
		return err
	}
	var resp WebsocketTradeResponse[[]types.Time]
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp.ReturnCode != 0 {
		return fmt.Errorf("%w: %w", errWebsocketAuthFailed, &APIError{Code: resp.ReturnCode, Message: resp.ReturnMessage})
	}
	return nil
}

// wsHandleTradeData routes replies on the trade connection to the requests waiting on them
func (e *Exchange) wsHandleTradeData(conn websocket.Connection, respRaw []byte) error {
	requestID, err := jsonparser.GetString(respRaw, "reqId")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return err
	}
	if requestID != "" {
		return conn.RequireMatchWithData(requestID, respRaw)
	}
	op, err := jsonparser.GetString(respRaw, "op")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return err
	}
	switch op {
	case authOperation:
		return conn.RequireMatchWithData(authOperation, respRaw)
	case "pong":
		return nil
	}
	return fmt.Errorf("%w for trade: %s", errUnhandledStreamData, respRaw)
}

// WSCreateOrder places an order over the trade connection (Websocket Trade Guideline: Create Order). It returns once
// Bybit accepts the request; the order's status, fills and fees follow on the private order and execution streams
func (e *Exchange) WSCreateOrder(ctx context.Context, r *PlaceOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.create", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketOrderRequest(ctx, "order.create", payload, epl)
}

// WSAmendOrder amends an order over the trade connection (Websocket Trade Guideline: Amend Order). It returns once Bybit
// accepts the request; the amended order follows on the private order stream
func (e *Exchange) WSAmendOrder(ctx context.Context, r *AmendOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.amend", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketOrderRequest(ctx, "order.amend", payload, epl)
}

// WSCancelOrder cancels an order over the trade connection (Websocket Trade Guideline: Cancel Order). It returns once
// Bybit accepts the request; the cancelled order follows on the private order stream
func (e *Exchange) WSCancelOrder(ctx context.Context, r *CancelOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.cancel", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketOrderRequest(ctx, "order.cancel", payload, epl)
}

// WSCreateBatchOrders places a batch of orders over the trade connection (Websocket Trade Guideline: Batch Create
// Order). Each result reports the outcome of its order; a rejected order has a non-zero Code. The orders' updates
// follow on the private order stream
func (e *Exchange) WSCreateBatchOrders(ctx context.Context, r *BatchPlaceOrderRequest) ([]WebsocketBatchOrderResult, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.create-batch", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketBatchOrderRequest(ctx, "order.create-batch", r.Category, len(r.Request), payload, epl)
}

// WSAmendBatchOrders amends a batch of orders over the trade connection (Websocket Trade Guideline: Batch Amend
// Order). Each result reports the outcome of its order; a rejected order has a non-zero Code. The orders' updates
// follow on the private order stream
func (e *Exchange) WSAmendBatchOrders(ctx context.Context, r *BatchAmendOrderRequest) ([]WebsocketBatchOrderResult, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.amend-batch", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketBatchOrderRequest(ctx, "order.amend-batch", r.Category, len(r.Request), payload, epl)
}

// WSCancelBatchOrders cancels a batch of orders over the trade connection (Websocket Trade Guideline: Batch Cancel
// Order). Each result reports the outcome of its order; a rejected order has a non-zero Code. The orders' updates
// follow on the private order stream
func (e *Exchange) WSCancelBatchOrders(ctx context.Context, r *BatchCancelOrderRequest) ([]WebsocketBatchOrderResult, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	epl, err := websocketOrderEPL("order.cancel-batch", r.Category)
	if err != nil {
		return nil, err
	}
	payload, err := r.Payload()
	if err != nil {
		return nil, err
	}
	return e.sendWebsocketBatchOrderRequest(ctx, "order.cancel-batch", r.Category, len(r.Request), payload, epl)
}

// websocketOrderEPL returns the rate limit of a websocket order entry operation for a category, which draws on the
// budget of the matching REST operation. Websocket order entry does not support spread trading, so any category but
// spot, linear, inverse and option is refused before a request is sent
func websocketOrderEPL(operation, category string) (request.EndpointLimit, error) {
	switch category {
	case cSpot, cLinear, cInverse, cOption:
		return getWSRateLimitEPLByCategory(operation, category)
	case "":
		return 0, errCategoryNotSet
	}
	return 0, fmt.Errorf("%w: websocket order entry for category %q", common.ErrFunctionNotSupported, category)
}

// sendWebsocketOrderRequest sends an order request over the trade connection and returns the order IDs from Bybit's
// acceptance of it
func (e *Exchange) sendWebsocketOrderRequest(ctx context.Context, op string, args any, epl request.EndpointLimit) (*OrderResponse, error) {
	conn, err := e.Websocket.GetConnection(OutboundTradeConnection)
	if err != nil {
		return nil, err
	}
	resp, err := sendWebsocketTradeRequest[OrderResponse](ctx, e, conn, op, args, epl)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// sendWebsocketBatchOrderRequest sends a batch order request over the trade connection and returns each order's result
// with its outcome
func (e *Exchange) sendWebsocketBatchOrderRequest(ctx context.Context, op, category string, orders int, args any, epl request.EndpointLimit) ([]WebsocketBatchOrderResult, error) {
	outbound, err := e.Websocket.GetConnection(OutboundTradeConnection)
	if err != nil {
		return nil, err
	}
	if err := e.takeBatchOrders(ctx, epl, category, orders); err != nil {
		return nil, err
	}
	resp, err := sendWebsocketTradeRequest[WebsocketBatchOrderAcknowledgements](ctx, e, outbound, op, args, epl)
	if err != nil {
		return nil, err
	}
	if len(resp.Data.List) != len(resp.RetExtInfo.List) {
		return nil, fmt.Errorf("%w: %d orders, %d outcomes", errBatchOutcomeMismatch, len(resp.Data.List), len(resp.RetExtInfo.List))
	}
	results := make([]WebsocketBatchOrderResult, len(resp.Data.List))
	for i := range resp.Data.List {
		results[i] = WebsocketBatchOrderResult{
			Category:    resp.Data.List[i].Category,
			Symbol:      resp.Data.List[i].Symbol,
			OrderID:     resp.Data.List[i].OrderID,
			OrderLinkID: resp.Data.List[i].OrderLinkID,
			CreateAt:    resp.Data.List[i].CreateAt.Time(),
			Code:        resp.RetExtInfo.List[i].Code,
			Message:     resp.RetExtInfo.List[i].Message,
		}
	}
	return results, nil
}

// takeBatchOrders takes one request from a websocket batch's budget for each order beyond the first, which the request
// itself takes: Bybit counts every order of a linear, inverse or spot batch against the operation's budget, but an
// options batch once
func (e *Exchange) takeBatchOrders(ctx context.Context, epl request.EndpointLimit, category string, orders int) error {
	if category == cOption {
		return nil
	}
	for range orders - 1 {
		if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
			return fmt.Errorf("rate limiting batch orders: %w", err)
		}
	}
	return nil
}

// sendWebsocketTradeRequest sends a request on the trade connection and returns its reply, or the error it reports
func sendWebsocketTradeRequest[T any](ctx context.Context, e *Exchange, conn websocket.Connection, op string, args any, epl request.EndpointLimit) (*WebsocketTradeResponse[T], error) {
	req := &WebsocketTradeRequest{
		RequestID: e.MessageID(),
		Header:    &WebsocketTradeRequestHeader{Timestamp: strconv.FormatInt(time.Now().UnixMilli(), 10)},
		Operation: op,
		Arguments: []any{args},
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, epl, req.RequestID, req)
	if err != nil {
		return nil, err
	}
	var resp WebsocketTradeResponse[T]
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return nil, err
	}
	if resp.ReturnCode != 0 {
		return nil, &APIError{Code: resp.ReturnCode, Message: resp.ReturnMessage}
	}
	return &resp, nil
}
