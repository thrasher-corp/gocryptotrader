package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Websocket request operation types
const (
	WsCreate = "order.create"
	WsAmend  = "order.amend"
	WsCancel = "order.cancel"

	OutboundTradeConnection  = "PRIVATE_TRADE"
	InboundPrivateConnection = "PRIVATE"
)

// CreateOrderThroughWebsocket creates an order through the websocket connection
// NOTE: When a SPOT order gets matched the fee is deducted from the purchased balance. e.g. if you buy 0.001 BTC depending
// on fee rate you will receive a sub 0.0009... BTC amount. Fees are returned in this response which can then be deducted.
// This does not apply to other asset types as it uses a settlement.
func (by *Bybit) CreateOrderThroughWebsocket(ctx context.Context, arg *PlaceOrderParams) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	return by.SendWebsocketRequest(ctx, WsCreate, arg)
}

// AmendOrderThroughWebsocket amends an order through the websocket connection
func (by *Bybit) AmendOrderThroughWebsocket(ctx context.Context, arg *AmendOrderParams) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	return by.SendWebsocketRequest(ctx, WsAmend, arg)
}

// CancelOrderThroughWebsocket cancels an order through the websocket connection
func (by *Bybit) CancelOrderThroughWebsocket(ctx context.Context, arg *CancelOrderParams) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	return by.SendWebsocketRequest(ctx, WsCancel, arg)
}

// WebsocketOrderDetails is the order details from the websocket response.
type WebsocketOrderDetails struct {
	Category             string        `json:"category"`
	OrderID              string        `json:"orderId"`
	OrderLinkID          string        `json:"orderLinkId"`
	IsLeverage           string        `json:"isLeverage"` // Whether to borrow. Unified spot only. 0: false, 1: true; Classic spot is not supported, always 0
	BlockTradeID         string        `json:"blockTradeId"`
	Symbol               string        `json:"symbol"` // Undelimitered so inbuilt string used.
	Price                types.Number  `json:"price"`
	Qty                  types.Number  `json:"qty"`
	Side                 order.Side    `json:"side"`
	PositionIdx          int           `json:"positionIdx"`
	OrderStatus          string        `json:"orderStatus"`
	CreateType           string        `json:"createType"`
	CancelType           string        `json:"cancelType"`
	RejectReason         string        `json:"rejectReason"` // Classic spot is not supported
	AvgPrice             types.Number  `json:"avgPrice"`
	LeavesQty            types.Number  `json:"leavesQty"`   // The remaining qty not executed. Classic spot is not supported
	LeavesValue          types.Number  `json:"leavesValue"` // The remaining value not executed. Classic spot is not supported
	CumExecQty           types.Number  `json:"cumExecQty"`
	CumExecValue         types.Number  `json:"cumExecValue"`
	CumExecFee           types.Number  `json:"cumExecFee"`
	ClosedPnl            types.Number  `json:"closedPnl"`
	FeeCurrency          currency.Code `json:"feeCurrency"` // Trading fee currency for Spot only.
	TimeInForce          string        `json:"timeInForce"`
	OrderType            string        `json:"orderType"`
	StopOrderType        string        `json:"stopOrderType"`
	OcoTriggerBy         string        `json:"ocoTriggerBy"` // UTA Spot: add new response field ocoTriggerBy, and the value can be OcoTriggerByUnknown, OcoTriggerByTp, OcoTriggerBySl
	OrderIv              types.Number  `json:"orderIv"`
	MarketUnit           string        `json:"marketUnit"`   // The unit for qty when create Spot market orders for UTA account. baseCoin, quoteCoin
	TriggerPrice         types.Number  `json:"triggerPrice"` // Trigger price. If stopOrderType=TrailingStop, it is activate price. Otherwise, it is trigger price
	TakeProfit           types.Number  `json:"takeProfit"`
	StopLoss             types.Number  `json:"stopLoss"`
	TpslMode             string        `json:"tpslMode"` // TP/SL mode, Full: entire position for TP/SL. Partial: partial position tp/sl. Spot does not have this field, and Option returns always ""
	TakeProfitLimitPrice types.Number  `json:"tpLimitPrice"`
	StopLossLimitPrice   types.Number  `json:"slLimitPrice"`
	TpTriggerBy          string        `json:"tpTriggerBy"`
	SlTriggerBy          string        `json:"slTriggerBy"`
	TriggerDirection     int           `json:"triggerDirection"` // Trigger direction. 1: rise, 2: fall
	TriggerBy            string        `json:"triggerBy"`
	LastPriceOnCreated   types.Number  `json:"lastPriceOnCreated"`
	ReduceOnly           bool          `json:"reduceOnly"`
	CloseOnTrigger       bool          `json:"closeOnTrigger"`
	PlaceType            string        `json:"placeType"` // 	Place type, option used. iv, price
	SmpType              string        `json:"smpType"`
	SmpGroup             int           `json:"smpGroup"`
	SmpOrderID           string        `json:"smpOrderId"`
	CreatedTime          types.Time    `json:"createdTime"`
	UpdatedTime          types.Time    `json:"updatedTime"`
}

// WebsocketConfirmation is the initial response from the websocket connection
type WebsocketConfirmation struct {
	RequestID              string            `json:"reqId"`
	RetCode                int64             `json:"retCode"`
	RetMsg                 string            `json:"retMsg"`
	Operation              string            `json:"op"`
	RequestAcknowledgement OrderResponse     `json:"data"`
	Header                 map[string]string `json:"header"`
	ConnectionID           string            `json:"connId"`
}

// WebsocketOrderResponse is the response from an order request through the websocket connection
type WebsocketOrderResponse struct {
	ID           string                  `json:"id"`
	Topic        string                  `json:"topic"`
	CreationTime types.Time              `json:"creationTime"`
	Data         []WebsocketOrderDetails `json:"data"`
}

// WebsocketGeneralPayload is the general payload for websocket requests
type WebsocketGeneralPayload struct {
	RequestID string            `json:"reqId"`
	Header    map[string]string `json:"header"`
	Operation string            `json:"op"`
	Arguments []any             `json:"args"`
}

// SendWebsocketRequest sends a request to the exchange through the websocket connection
func (by *Bybit) SendWebsocketRequest(ctx context.Context, op string, argument any) (*WebsocketOrderDetails, error) {
	// Get the outbound and inbound connections to send and receive the request. This makes sure both are live before
	// sending the request.
	outbound, err := by.Websocket.GetConnection(OutboundTradeConnection)
	if err != nil {
		return nil, err
	}
	inbound, err := by.Websocket.GetConnection(InboundPrivateConnection)
	if err != nil {
		return nil, err
	}

	out := WebsocketGeneralPayload{
		RequestID: strconv.FormatInt(outbound.GenerateMessageID(false), 10),
		Header:    map[string]string{"X-BAPI-TIMESTAMP": strconv.FormatInt(time.Now().UnixMilli(), 10)},
		Operation: op,
		Arguments: []any{argument},
	}

	outResp, err := outbound.SendMessageReturnResponse(ctx, request.Unset, out.RequestID, out)
	if err != nil {
		return nil, err
	}

	var confirmation WebsocketConfirmation
	if err := json.Unmarshal(outResp, &confirmation); err != nil {
		return nil, err
	}

	if confirmation.RetCode != 0 {
		return nil, fmt.Errorf("code:%d, info:%v message:%s", confirmation.RetCode, retCode[confirmation.RetCode], confirmation.RetMsg)
	}

	wait, err := inbound.MatchReturnResponses(ctx, confirmation.RequestAcknowledgement.OrderID, 1)
	if err != nil {
		return nil, err
	}

	// Wait for the response to come back from the inbound connection.
	inResp := <-wait
	if inResp.Err != nil {
		return nil, inResp.Err
	}

	if len(inResp.Responses) != 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(inResp.Responses))
	}

	var ret WebsocketOrderResponse
	if err := json.Unmarshal(inResp.Responses[0], &ret); err != nil {
		return nil, err
	}

	if len(ret.Data) != 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(ret.Data))
	}

	if ret.Data[0].RejectReason != "EC_NoError" {
		return nil, fmt.Errorf("order rejected: %s", ret.Data[0].RejectReason)
	}

	return &ret.Data[0], nil
}

// retCode is a map of error codes to their respective messages, as they are not supplied through the API.
var retCode = map[int64]string{
	10404: "1. op type is not found; 2. category is not correct/supported",
	10429: "System level frequency protection",
	20006: "reqId is duplicated",
	10016: "1. internal server error; 2. Service is restarting",
	10019: "ws trade service is restarting, do not accept new request, but the request in the process is not affected. You can build new connection to be routed to normal service",
}