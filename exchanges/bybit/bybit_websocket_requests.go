package bybit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Websocket request operation types
const (
	OutboundTradeConnection  = "PRIVATE_TRADE"
	InboundPrivateConnection = "PRIVATE"
)

// WSCreateOrder creates an order through the websocket connection
func (e *Exchange) WSCreateOrder(ctx context.Context, r *PlaceOrderRequest) (*WebsocketOrderDetails, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(r.Category)
	if err != nil {
		return nil, err
	}
	if r.OrderLinkID == "" {
		r.OrderLinkID = uuid.Must(uuid.NewV7()).String()
	}
	return e.sendWebsocketTradeRequest(ctx, "order.create", r.OrderLinkID, r, epl)
}

// WSAmendOrder amends an order through the websocket connection
func (e *Exchange) WSAmendOrder(ctx context.Context, r *AmendOrderRequest) (*WebsocketOrderDetails, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(r.Category)
	if err != nil {
		return nil, err
	}
	if r.OrderLinkID == "" {
		r.OrderLinkID = uuid.Must(uuid.NewV7()).String()
	}
	return e.sendWebsocketTradeRequest(ctx, "order.amend", r.OrderLinkID, r, epl)
}

// WSCancelOrder cancels an order through the websocket connection
func (e *Exchange) WSCancelOrder(ctx context.Context, r *CancelOrderRequest) (*WebsocketOrderDetails, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	epl, err := getWSRateLimitEPLByCategory(r.Category)
	if err != nil {
		return nil, err
	}
	if r.OrderLinkID == "" {
		r.OrderLinkID = uuid.Must(uuid.NewV7()).String()
	}
	return e.sendWebsocketTradeRequest(ctx, "order.cancel", r.OrderLinkID, r, epl)
}

// sendWebsocketTradeRequest sends a trade request to the exchange through the websocket connection
func (e *Exchange) sendWebsocketTradeRequest(ctx context.Context, op, orderLinkID string, payload any, limit request.EndpointLimit) (*WebsocketOrderDetails, error) {
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

	// Set up a listener to wait for the response to come back from the inbound connection. The request is sent through
	// the outbound trade connection, the response can come back through the inbound private connection before the
	// outbound connection sends its acknowledgement.
	ch, err := inbound.MatchReturnResponses(ctx, orderLinkID, 1)
	if err != nil {
		return nil, err
	}

	requestID := e.MessageID()
	outResp, err := outbound.SendMessageReturnResponse(ctx, limit, requestID, WebsocketGeneralPayload{
		RequestID: requestID,
		Header:    map[string]string{"X-BAPI-TIMESTAMP": strconv.FormatInt(time.Now().UnixMilli(), 10)},
		Operation: op,
		Arguments: []any{payload},
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

	inResp := <-ch // Blocking read is acceptable; channel has a built in timeout already
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
	20003: "too frequent requests under the same session",
	10403: "exceed IP rate limit. 3000 requests per second per IP",
}
