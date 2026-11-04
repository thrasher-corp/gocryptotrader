package kucoin

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// utaTradeAuthSignature matches the reply to the UTA trade connection's authentication message, which carries no ID
const utaTradeAuthSignature = "uta-trade-auth"

// AuthenticateUTATradeWebsocket signs in the UTA trade connection. Its reply carries no ID: a welcome frame confirms
// the sign-in, while a code and message reject it before KuCoin closes the connection
func (e *Exchange) AuthenticateUTATradeWebsocket(ctx context.Context, conn websocket.Connection) error {
	req, err := e.utaAuthRequest(ctx)
	if err != nil {
		return err
	}
	reply, err := conn.SendMessageReturnResponse(ctx, request.Unset, utaTradeAuthSignature, req)
	if err != nil {
		return err
	}
	var resp *UTAWebsocketTradeAuthResponse
	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%w: %w", errUTAAuthenticationFailed, common.ErrNoResponse)
	}
	if resp.Data != "welcome" {
		return fmt.Errorf("%w: code %s: %s", errUTAAuthenticationFailed, resp.Code, resp.Message)
	}
	return nil
}

// wsHandleUTATradeData routes a frame from the UTA trade connection: order operation replies by their ID, and
// authentication replies, which carry none, to the pending authentication
func (e *Exchange) wsHandleUTATradeData(_ context.Context, conn websocket.Connection, respData []byte) error {
	var msg UTAWebsocketTradeStatus
	if err := json.Unmarshal(respData, &msg); err != nil {
		return err
	}
	switch {
	case msg.Operation == "pong":
		return nil
	case msg.ID != "":
		return conn.RequireMatchWithData(msg.ID, respData)
	}
	return conn.RequireMatchWithData(utaTradeAuthSignature, respData)
}

// WebsocketAddUTAOrder places a spot, margin or futures order from the unified trading account over the UTA trade connection
// (Add Order)
func (e *Exchange) WebsocketAddUTAOrder(ctx context.Context, r *PlaceUTAOrderRequest) (*UTAWebsocketOrderResponse, error) {
	args, err := r.Payload()
	if err != nil {
		return nil, err
	}
	resp := new(UTAWebsocketOrderResponse)
	if err := e.sendUTATradeRequest(ctx, placeUTAOrderEPL, "uta.order", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WebsocketCancelUTAOrder cancels a unified trading account order over the UTA trade connection (Cancel Order)
func (e *Exchange) WebsocketCancelUTAOrder(ctx context.Context, r *CancelUTAOrderRequest) (*UTAWebsocketOrderResponse, error) {
	args, err := r.Payload()
	if err != nil {
		return nil, err
	}
	resp := new(UTAWebsocketOrderResponse)
	if err := e.sendUTATradeRequest(ctx, cancelUTAOrderEPL, "uta.cancel", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WebsocketAmendUTAOrder changes the price, size or take profit and stop loss trigger prices of a unified trading account
// futures order over the UTA trade connection (Amend Order)
func (e *Exchange) WebsocketAmendUTAOrder(ctx context.Context, r *AmendUTAOrderRequest) (*UTAWebsocketAmendOrderResponse, error) {
	args, err := r.Payload()
	if err != nil {
		return nil, err
	}
	resp := new(UTAWebsocketAmendOrderResponse)
	if err := e.sendUTATradeRequest(ctx, amendUTAOrderEPL, "uta.amend", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// sendUTATradeRequest sends an order operation over the UTA trade connection and decodes its reply into result,
// returning an error when KuCoin rejects the operation
func (e *Exchange) sendUTATradeRequest(ctx context.Context, epl request.EndpointLimit, operation string, args map[string]any, result utaTradeReply) error {
	conn, err := e.Websocket.GetConnection(utaTradeConnection)
	if err != nil {
		return err
	}
	// Websocket order operations share the UTA trading pool with their REST twins
	if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
		return err
	}
	req := &UTAWebsocketTradeRequest{ID: e.utaMessageID(), Operation: operation, Arguments: args}
	reply, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(reply, result); err != nil {
		return err
	}
	if status := result.tradeStatus(); status.Code != "200000" {
		return fmt.Errorf("%w: %s code %s: %s", errUTARequestRejected, operation, status.Code, status.Message)
	}
	return nil
}
