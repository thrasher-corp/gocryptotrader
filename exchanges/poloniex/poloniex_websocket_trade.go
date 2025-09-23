package poloniex

import (
	"context"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// WsCreateOrder create an order for an account.
func (e *Exchange) WsCreateOrder(arg *PlaceOrderParams) (*PlaceOrderResponse, error) {
	if *arg == (PlaceOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	var resp []PlaceOrderResponse
	err := e.SendWebsocketRequest("createOrder", arg, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrInvalidResponse
	} else if resp[0].Code != 0 {
		return nil, fmt.Errorf("error code: %d message: %s", resp[0].Code, resp[0].Message)
	}
	return &resp[0], nil
}

// WsCancelMultipleOrdersByIDs batch cancel one or many active orders in an account by IDs through the websocket stream.
func (e *Exchange) WsCancelMultipleOrdersByIDs(args *OrderCancellationParams) ([]WsCancelOrderResponse, error) {
	if args == nil {
		return nil, common.ErrNilPointer
	}
	if len(args.ClientOrderIDs) == 0 && len(args.OrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []WsCancelOrderResponse
	return resp, e.SendWebsocketRequest("cancelOrders", args, &resp)
}

// WsCancelAllTradeOrders batch cancel all orders in an account.
func (e *Exchange) WsCancelAllTradeOrders(symbols, accountTypes []string) ([]WsCancelOrderResponse, error) {
	args := make(map[string][]string)
	if len(symbols) > 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	var resp []WsCancelOrderResponse
	return resp, e.SendWebsocketRequest("cancelAllOrders", args, &resp)
}

// SendWebsocketRequest represents a websocket request through the authenticated connections.
func (e *Exchange) SendWebsocketRequest(event string, arg, response any) error {
	if !e.Websocket.IsConnected() || !e.Websocket.CanUseAuthenticatedEndpoints() {
		return websocket.ErrWebsocketNotEnabled
	}
	input := &struct {
		ID     string `json:"id"`
		Event  string `json:"event"`
		Params any    `json:"params"`
	}{
		ID:     strconv.FormatInt(e.Websocket.Conn.GenerateMessageID(false), 10),
		Event:  event,
		Params: arg,
	}
	result, err := e.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, input.ID, input)
	if err != nil {
		return err
	}

	resp := &WebsocketResponse{
		Data: response,
	}
	err = json.Unmarshal(result, &resp)
	if err != nil {
		return err
	}
	return nil
}
