package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Websocket request operation types
const (
	WsCreate = "order.create"
	WsAmend  = "order.amend"
	WsCancel = "order.cancel"

	OutboundTradeConnection = "PRIVATE_TRADE"
)

// CreateOrderThroughWebsocket creates an order through the websocket connection
func (by *Bybit) CreateOrderThroughWebsocket(ctx context.Context, arg *PlaceOrderParams) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var ret *OrderResponse
	return ret, by.SendWebsocketRequest(ctx, WsCreate, arg, &ret)
}

// AmendOrderThroughWebsocket amends an order through the websocket connection
func (by *Bybit) AmendOrderThroughWebsocket(ctx context.Context, arg *AmendOrderParams) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var ret *OrderResponse
	return ret, by.SendWebsocketRequest(ctx, WsAmend, arg, &ret)
}

// CancelOrderThroughWebsocket cancels an order through the websocket connection
func (by *Bybit) CancelOrderThroughWebsocket(ctx context.Context, arg *CancelOrderParams) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var ret *OrderResponse
	return ret, by.SendWebsocketRequest(ctx, WsCancel, arg, &ret)
}

// SendWebsocketRequest sends a request to the exchange through the websocket connection
func (by *Bybit) SendWebsocketRequest(ctx context.Context, op string, argument, ret any) error {
	conn, err := by.Websocket.GetConnection(OutboundTradeConnection)
	if err != nil {
		return err
	}

	out := struct {
		RequestID string            `json:"reqId"`
		Header    map[string]string `json:"header"`
		Operation string            `json:"op"`
		Arguments []any             `json:"args"`
	}{
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		Header:    map[string]string{"X-BAPI-TIMESTAMP": strconv.FormatInt(time.Now().UnixMilli(), 10)},
		Operation: op,
		Arguments: []any{argument},
	}

	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, out.RequestID, out)
	if err != nil {
		return err
	}

	response := struct {
		RequestID    string            `json:"reqId"`
		RetCode      int64             `json:"retCode"`
		RetMsg       string            `json:"retMsg"`
		Operation    string            `json:"op"`
		Data         any               `json:"data"`
		Header       map[string]string `json:"header"`
		ConnectionID string            `json:"connId"`
	}{
		Data: ret,
	}

	if err := json.Unmarshal(resp, &response); err != nil {
		return err
	}

	if response.RetCode != 0 {
		return fmt.Errorf("code:%d, info:%v message:%s", response.RetCode, retCode[response.RetCode], response.RetMsg)
	}

	return nil
}

// retCode is a map of error codes to their respective messages, as they are not supplied through the API.
var retCode = map[int64]string{
	10404: "1. op type is not found; 2. category is not correct/supported",
	10429: "System level frequency protection",
	20006: "reqId is duplicated",
	10016: "1. internal server error; 2. Service is restarting",
	10019: "ws trade service is restarting, do not accept new request, but the request in the process is not affected. You can build new connection to be routed to normal service",
}
