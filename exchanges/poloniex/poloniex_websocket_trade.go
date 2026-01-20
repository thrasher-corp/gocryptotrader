package poloniex

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WsCreateOrder create an order for an account.
func (e *Exchange) WsCreateOrder(ctx context.Context, arg *PlaceOrderRequest) (*WsOrderIDResponse, error) {
	if err := validateOrderRequest(arg); err != nil {
		return nil, err
	}
	resp, err := SendWebsocketRequest[*WsOrderIDResponse](ctx, e, "createOrder", arg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	if len(resp) != 1 || resp[0] == nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, common.ErrInvalidResponse)
	}
	return resp[0], nil
}

// WsCancelMultipleOrdersByIDs batch cancels one or many active orders in an account by their IDs through the websocket stream.
func (e *Exchange) WsCancelMultipleOrdersByIDs(ctx context.Context, orderIDs, clientOrderIDs []string) ([]*WsCancelOrderResponse, error) {
	if len(clientOrderIDs) == 0 && len(orderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string][]string)
	if len(clientOrderIDs) > 0 {
		params["clientOrderIds"] = clientOrderIDs
	}
	if len(orderIDs) > 0 {
		params["orderIds"] = orderIDs
	}
	resp, err := SendWebsocketRequest[*WsCancelOrderResponse](ctx, e, "cancelOrders", params)
	if err != nil {
		// Return resp, which contains the full response including both
		// successful and failed cancellation attempts.
		return resp, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// WsCancelTradeOrders batch cancels all orders in an account.
func (e *Exchange) WsCancelTradeOrders(ctx context.Context, symbols []string, accountTypes []AccountType) ([]*WsCancelOrderResponse, error) {
	args := make(map[string]any)
	if len(symbols) > 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	resp, err := SendWebsocketRequest[*WsCancelOrderResponse](ctx, e, "cancelAllOrders", args)
	if err != nil {
		// Return resp, which contains the full response including both
		// successful and failed cancellation attempts.
		return resp, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// SendWebsocketRequest sends a websocket request through the private connection
func SendWebsocketRequest[T hasError](ctx context.Context,
	e *Exchange,
	event string,
	arg any,
) (response []T, err error) {
	conn, err := e.Websocket.GetConnection(connSpotPrivate)
	if err != nil {
		return nil, err
	}

	input := &struct {
		ID     string `json:"id"`
		Event  string `json:"event"`
		Params any    `json:"params"`
	}{
		ID:     e.MessageID(),
		Event:  event,
		Params: arg,
	}

	// Locks due to the endpoint not returning identifying packets so outbound subs and inbound responses can be matched.
	e.spotSubMtx.Lock()
	result, err := conn.SendMessageReturnResponse(ctx, sWebsocketPrivateEPL, input.ID, input)
	e.spotSubMtx.Unlock()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &WebsocketResponse{Data: &response}); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, common.ErrNoResponse
	}

	return response, checkForErrorInSliceResponse(response)
}
