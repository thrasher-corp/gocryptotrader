package bybit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Websocket request operation types
const (
	OutboundTradeConnection  = "PRIVATE_TRADE"
	InboundPrivateConnection = "PRIVATE"
)

// WSCreateOrder creates an order through the websocket connection
func (e *Exchange) WSCreateOrder(ctx context.Context, arg *PlaceOrderRequest) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	return e.SendWebsocketRequest(ctx, "order.create", arg, epl)
}

// setOrderLinkID the order link ID if not already populated
func (r *PlaceOrderRequest) setOrderLinkID(id string) string {
	if r.OrderLinkID == "" {
		r.OrderLinkID = id
	}
	return r.OrderLinkID
}

// WSAmendOrder amends an order through the websocket connection
func (e *Exchange) WSAmendOrder(ctx context.Context, arg *AmendOrderRequest) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	return e.SendWebsocketRequest(ctx, "order.amend", arg, epl)
}

// setOrderLinkID the order link ID if not already populated
func (r *AmendOrderRequest) setOrderLinkID(id string) string {
	if r.OrderLinkID == "" {
		r.OrderLinkID = id
	}
	return r.OrderLinkID
}

// WSCancelOrder cancels an order through the websocket connection
func (e *Exchange) WSCancelOrder(ctx context.Context, arg *CancelOrderRequest) (*WebsocketOrderDetails, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	return e.SendWebsocketRequest(ctx, "order.cancel", arg, epl)
}

// setOrderLinkID the order link ID if not already populated
func (r *CancelOrderRequest) setOrderLinkID(id string) string {
	if r.OrderLinkID == "" {
		r.OrderLinkID = id
	}
	return r.OrderLinkID
}

// SendWebsocketRequest sends a request to the exchange through the websocket connection
func (e *Exchange) SendWebsocketRequest(ctx context.Context, op string, payload orderLinkIDSetter, limit request.EndpointLimit) (*WebsocketOrderDetails, error) {
	// Get the outbound and inbound connections to send and receive the request. This makes sure both are live before
	// sending the request.
	outbound, err := e.Websocket.GetConnection(OutboundTradeConnection)
	if err != nil {
		return nil, err
	}
	inbound, err := e.Websocket.GetConnection(InboundPrivateConnection)
	if err != nil {
		return nil, err
	}

	tn := time.Now()
	requestID := strconv.FormatInt(outbound.GenerateMessageID(false), 10)

	// Sets OrderLinkID to the outbound payload so that the response can be matched to the request in the inbound connection.
	orderLinkID := argument.setOrderLinkID(strconv.FormatInt(tn.UnixNano(), 10) + requestID) // UnixNano is used to reduce the chance of an ID clash

	// Set up a listener to wait for the response to come back from the inbound connection. The request is sent through
	// the outbound trade connection, the response can come back through the inbound private connection before the
	// outbound connection sends its acknowledgement.
	wait, err := inbound.MatchReturnResponses(ctx, argumentID, 1)
	if err != nil {
		return nil, err
	}

	outResp, err := outbound.SendMessageReturnResponse(ctx, limit, requestID, WebsocketGeneralPayload{
		RequestID: requestID,
		Header:    map[string]string{"X-BAPI-TIMESTAMP": strconv.FormatInt(tn.UnixMilli(), 10)},
		Operation: op,
		Arguments: []any{argument},
	})
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

	inResp := <-wait // Blocking read okay; wait channel has a built in timeout already
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

var retCode = map[int64]string{
	10404: "either op type is not found or category is not correct/supported",
	10429: "request exceeds rate limit",
	20006: "reqId is duplicated",
	10016: "internal server error",
	10019: "ws trade service is restarting, please reconnect",
}
