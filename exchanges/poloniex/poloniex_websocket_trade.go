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
func (e *Exchange) WsCreateOrder(arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
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
		return nil, fmt.Errorf("%w: error code: %d message: %s", common.ErrNoResponse, resp[0].Code, resp[0].Message)
	}
	return &resp[0], nil
}

// WsCancelMultipleOrdersByIDs batch cancel one or many active orders in an account by IDs through the websocket stream.
func (e *Exchange) WsCancelMultipleOrdersByIDs(args *CancelOrdersRequest) ([]WsCancelOrderResponse, error) {
	if args == nil {
		return nil, common.ErrNilPointer
	}
	if len(args.ClientOrderIDs) == 0 && len(args.OrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []WsCancelOrderResponse
	err := e.SendWebsocketRequest("cancelOrders", args, &resp)
	if err != nil {
		return nil, err
	}
	for r := range resp {
		if resp[r].Code != 0 {
			return nil, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp[r].Code, resp[r].Message)
		}
	}
	return resp, nil
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
	err := e.SendWebsocketRequest("cancelAllOrders", args, &resp)
	if err != nil {
		return nil, err
	}
	for r := range resp {
		if resp[r].Code != 0 {
			return nil, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp[r].Code, resp[r].Message)
		}
	}
	return resp, nil
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
	conn, err := e.Websocket.GetConnection(connSpotPrivate)
	if err != nil {
		return err
	}
	result, err := conn.SendMessageReturnResponse(context.Background(), request.Auth, input.ID, input)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(result, &WebsocketResponse{
		Data: response,
	}); err != nil {
		return err
	}
	return nil
}
