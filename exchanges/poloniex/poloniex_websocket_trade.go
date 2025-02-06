package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WsCreateOrder create an order for an account.
func (p *Poloniex) WsCreateOrder(arg *PlaceOrderParams) (*PlaceOrderResponse, error) {
	if *arg == (PlaceOrderParams{}) {
		return nil, errNilArgument
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	var resp []PlaceOrderResponse
	err := p.SendWebsocketRequest("createOrder", arg, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errInvalidResponse
	} else if resp[0].Code != 0 {
		return nil, fmt.Errorf("error code: %d message: %s", resp[0].Code, resp[0].Message)
	}
	return &resp[0], nil
}

// WsCancelMultipleOrdersByIDs batch cancel one or many active orders in an account by IDs through the websocket stream.
func (p *Poloniex) WsCancelMultipleOrdersByIDs(args *OrderCancellationParams) ([]WsCancelOrderResponse, error) {
	if args == nil {
		return nil, errNilArgument
	}
	if len(args.ClientOrderIDs) == 0 && len(args.OrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []WsCancelOrderResponse
	return resp, p.SendWebsocketRequest("cancelOrders", args, &resp)
}

// WsCancelAllTradeOrders batch cancel all orders in an account.
func (p *Poloniex) WsCancelAllTradeOrders(symbols, accountTypes []string) ([]WsCancelOrderResponse, error) {
	args := make(map[string][]string)
	if len(symbols) > 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	var resp []WsCancelOrderResponse
	return resp, p.SendWebsocketRequest("cancelAllOrders", args, &resp)
}

// SendWebsocketRequest represents a websocket request through the authenticated connections.
func (p *Poloniex) SendWebsocketRequest(event string, arg, response interface{}) error {
	if !p.Websocket.IsConnected() || !p.Websocket.CanUseAuthenticatedEndpoints() {
		return stream.ErrWebsocketNotEnabled
	}
	input := &struct {
		ID     string      `json:"id"`
		Event  string      `json:"event"`
		Params interface{} `json:"params"`
	}{
		ID:     strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
		Event:  event,
		Params: arg,
	}
	result, err := p.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, input.ID, input)
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
