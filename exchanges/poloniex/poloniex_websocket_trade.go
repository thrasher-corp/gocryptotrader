package poloniex

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WsCreateOrder create an order for an account.
func (e *Exchange) WsCreateOrder(ctx context.Context, arg *PlaceOrderRequest) (*OrderIDResponse, error) {
	if err := validateOrderRequest(arg); err != nil {
		return nil, err
	}
	var resp []*OrderIDResponse
	if err := SendWebsocketRequest(ctx, e, "createOrder", arg, resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	if len(resp) != 1 {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, common.ErrInvalidResponse)
	}
	return resp[0], nil //nolint:nilness // resp is guaranteed to be non-nil and contain exactly one element
}

// WsCancelMultipleOrdersByIDs batch cancel one or many active orders in an account by IDs through the websocket stream.
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
	var resp []*WsCancelOrderResponse
	if err := SendWebsocketRequest(ctx, e, "cancelOrders", params, resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// WsCancelTradeOrders batch cancel all orders in an account.
func (e *Exchange) WsCancelTradeOrders(ctx context.Context, symbols []string, accountTypes []AccountType) ([]*WsCancelOrderResponse, error) {
	args := make(map[string]any)
	if len(symbols) > 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	var resp []*WsCancelOrderResponse
	if err := SendWebsocketRequest(ctx, e, "cancelAllOrders", args, resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// SendWebsocketRequest sends a websocket request through the private connection
func SendWebsocketRequest[T hasError](ctx context.Context,
	e *Exchange,
	event string,
	arg any,
	response []T,
) error {
	conn, err := e.Websocket.GetConnection(connSpotPrivate)
	if err != nil {
		return err
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

	e.spotSubMtx.Lock()
	result, err := conn.SendMessageReturnResponse(ctx, sWebsocketPrivateEPL, input.ID, input)
	e.spotSubMtx.Unlock()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(result, &WebsocketResponse{Data: &response}); err != nil {
		return err
	}
	if response == nil {
		return common.ErrNoResponse
	}

	return checkForErrorInSliceResponse(response)
}

func checkForErrorInSliceResponse[T hasError](slice []T) error {
	var err error
	for _, v := range slice {
		if e := v.Error(); e != nil {
			err = common.AppendError(err, e)
		}
	}
	return err
}
