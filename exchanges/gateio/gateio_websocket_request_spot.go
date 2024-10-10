package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	errBatchSliceEmpty  = errors.New("batch cannot be empty")
	errNoOrdersToCancel = errors.New("no orders to cancel")
	errEdgeCaseIssue    = errors.New("edge case issue")
	errChannelEmpty     = errors.New("channel cannot be empty")
)

// WebsocketLogin authenticates the websocket connection
func (g *Gateio) WebsocketLogin(ctx context.Context, conn stream.Connection, channel string) (*WebsocketLoginResponse, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, conn)
	}

	if channel == "" {
		return nil, errChannelEmpty
	}

	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	tn := time.Now().Unix()
	msg := "api\n" + channel + "\n" + "\n" + strconv.FormatInt(tn, 10)
	mac := hmac.New(sha512.New, []byte(creds.Secret))
	if _, err = mac.Write([]byte(msg)); err != nil {
		return nil, err
	}
	signature := hex.EncodeToString(mac.Sum(nil))

	payload := WebsocketPayload{
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		APIKey:    creds.Key,
		Signature: signature,
		Timestamp: strconv.FormatInt(tn, 10),
	}

	req := WebsocketRequest{Time: tn, Channel: channel, Event: "api", Payload: payload}

	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.Payload.RequestID, req)
	if err != nil {
		return nil, err
	}

	var inbound WebsocketAPIResponse
	if err := json.Unmarshal(resp, &inbound); err != nil {
		return nil, err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		if err := json.Unmarshal(inbound.Data, &wsErr.Errors); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	var result WebsocketLoginResponse
	return &result, json.Unmarshal(inbound.Data, &result)
}

// WebsocketOrderPlaceSpot places an order via the websocket connection. You can
// send multiple orders in a single request. But only for one asset route.
// So this can only batch spot orders or futures orders, not both.
func (g *Gateio) WebsocketOrderPlaceSpot(ctx context.Context, batch []WebsocketOrder) ([]WebsocketOrderResponse, error) {
	if len(batch) == 0 {
		return nil, errBatchSliceEmpty
	}

	for i := range batch {
		if batch[i].Text == "" {
			// For some reason the API requires a text field, or it will be
			// rejected in the second response. This is a workaround.
			batch[i].Text = "t-" + strconv.FormatInt(g.Counter.IncrementAndGet(), 10)
		}
		if batch[i].CurrencyPair == "" {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if batch[i].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if batch[i].Amount == "" {
			return nil, errInvalidAmount
		}
		if batch[i].Type == "limit" && batch[i].Price == "" {
			return nil, errInvalidPrice
		}
	}

	if len(batch) == 1 {
		var singleResponse WebsocketOrderResponse
		err := g.SendWebsocketRequest(ctx, "spot.order_place", asset.Spot, batch[0], &singleResponse, 2)
		return []WebsocketOrderResponse{singleResponse}, err
	}

	var resp []WebsocketOrderResponse
	err := g.SendWebsocketRequest(ctx, "spot.order_place", asset.Spot, batch, &resp, 2)
	return resp, err
}

// WebsocketOrderCancelSpot cancels an order via the websocket connection
func (g *Gateio) WebsocketOrderCancelSpot(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &struct {
		OrderID string `json:"order_id"` // This requires order_id tag
		Pair    string `json:"pair"`
		Account string `json:"account,omitempty"`
	}{
		OrderID: orderID,
		Pair:    pair.String(),
		Account: account,
	}

	var resp WebsocketOrderResponse
	err := g.SendWebsocketRequest(ctx, "spot.order_cancel", asset.Spot, params, &resp, 1)
	return &resp, err
}

// WebsocketOrderCancelAllByIDsSpot cancels multiple orders via the websocket
func (g *Gateio) WebsocketOrderCancelAllByIDsSpot(ctx context.Context, o []WebsocketOrderCancelRequest) ([]WebsocketCancellAllResponse, error) {
	if len(o) == 0 {
		return nil, errNoOrdersToCancel
	}

	for i := range o {
		if o[i].OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if o[i].Pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
	}

	var resp []WebsocketCancellAllResponse
	err := g.SendWebsocketRequest(ctx, "spot.order_cancel_ids", asset.Spot, o, &resp, 2)
	return resp, err
}

// WebsocketOrderCancelAllByPairSpot cancels all orders for a specific pair
func (g *Gateio) WebsocketOrderCancelAllByPairSpot(ctx context.Context, pair currency.Pair, side order.Side, account string) ([]WebsocketOrderResponse, error) {
	if !pair.IsEmpty() && side == order.UnknownSide {
		return nil, fmt.Errorf("%w: side cannot be unknown when pair is set as this will purge *ALL* open orders", errEdgeCaseIssue)
	}

	sideStr := ""
	if side != order.UnknownSide {
		sideStr = side.Lower()
	}

	params := &WebsocketCancelParam{
		Pair:    pair,
		Side:    sideStr,
		Account: account,
	}

	var resp []WebsocketOrderResponse
	return resp, g.SendWebsocketRequest(ctx, "spot.order_cancel_cp", asset.Spot, params, &resp, 1)
}

// WebsocketOrderAmendSpot amends an order via the websocket connection
func (g *Gateio) WebsocketOrderAmendSpot(ctx context.Context, amend *WebsocketAmendOrder) (*WebsocketOrderResponse, error) {
	if amend == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, amend)
	}

	if amend.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	if amend.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if amend.Amount == "" && amend.Price == "" {
		return nil, fmt.Errorf("%w: amount or price must be set", errInvalidAmount)
	}

	var resp WebsocketOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, "spot.order_amend", asset.Spot, amend, &resp, 1)
}

// WebsocketGetOrderStatusSpot gets the status of an order via the websocket connection
func (g *Gateio) WebsocketGetOrderStatusSpot(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &struct {
		OrderID string `json:"order_id"` // This requires order_id tag
		Pair    string `json:"pair"`
		Account string `json:"account,omitempty"`
	}{
		OrderID: orderID,
		Pair:    pair.String(),
		Account: account,
	}

	var resp WebsocketOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, "spot.order_status", asset.Spot, params, &resp, 1)
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, channel string, connSignature, params, result any, expectedResponses int) error {
	paramPayload, err := json.Marshal(params)
	if err != nil {
		return err
	}

	conn, err := g.Websocket.GetOutboundConnection(connSignature)
	if err != nil {
		return err
	}

	tn := time.Now().Unix()
	req := &WebsocketRequest{
		Time:    tn,
		Channel: channel,
		Event:   "api",
		Payload: WebsocketPayload{
			// This request ID associated with the payload is the match to the
			// response.
			RequestID:    strconv.FormatInt(conn.GenerateMessageID(false), 10),
			RequestParam: paramPayload,
			Timestamp:    strconv.FormatInt(tn, 10),
		},
	}

	responses, err := conn.SendMessageReturnResponses(ctx, request.Unset, req.Payload.RequestID, req, expectedResponses, InspectPayloadForAck)
	if err != nil {
		return err
	}

	if len(responses) == 0 {
		return errors.New("no responses received")
	}

	var inbound WebsocketAPIResponse
	// The last response is the one we want to unmarshal, the other is just
	// an ack. If the request fails on the ACK then we can unmarshal the error
	// from that as the next response won't come anyway.
	endResponse := responses[len(responses)-1]

	if err := json.Unmarshal(endResponse, &inbound); err != nil {
		return err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		if err := json.Unmarshal(inbound.Data, &wsErr); err != nil {
			return err
		}
		return fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	to := struct {
		Result any `json:"result"`
	}{
		Result: result,
	}

	return json.Unmarshal(inbound.Data, &to)
}

// InspectPayloadForAck checks the payload for an ack, it returns true if the
// payload does not contain an ack. This will force the cancellation of further
// waiting for responses.
func InspectPayloadForAck(data []byte) bool {
	return !strings.Contains(string(data), "ack")
}
