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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	errBatchSliceEmpty  = errors.New("batch cannot be empty")
	errNoOrdersToCancel = errors.New("no orders to cancel")
	errEdgeCaseIssue    = errors.New("edge case issue")
	errChannelEmpty     = errors.New("channel cannot be empty")
)

// GetWebsocketRoute returns the route for a websocket request, this is a POC
// for the websocket wrapper.
func (g *Gateio) GetWebsocketRoute(a asset.Item) (string, error) {
	switch a {
	case asset.Spot:
		return gateioWebsocketEndpoint, nil
	default:
		return "", common.ErrNotYetImplemented
	}
}

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

	tn := time.Now()
	msg := "api\n" + channel + "\n" + "\n" + strconv.FormatInt(tn.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(creds.Secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return nil, err
	}
	signature := hex.EncodeToString(mac.Sum(nil))

	payload := WebsocketPayload{
		RequestID: strconv.FormatInt(tn.UnixNano(), 10),
		APIKey:    creds.Key,
		Signature: signature,
		Timestamp: strconv.FormatInt(tn.Unix(), 10),
	}

	request := WebsocketRequest{Time: tn.Unix(), Channel: channel, Event: "api", Payload: payload}

	resp, err := conn.SendMessageReturnResponse(ctx, request.Payload.RequestID, request)
	if err != nil {
		return nil, err
	}

	var inbound WebsocketAPIResponse
	err = json.Unmarshal(resp, &inbound)
	if err != nil {
		return nil, err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		err := json.Unmarshal(inbound.Data, &wsErr.Errors)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	var result WebsocketLoginResponse
	return &result, json.Unmarshal(inbound.Data, &result)
}

// WebsocketOrderPlace places an order via the websocket connection. You can
// send multiple orders in a single request. But only for one asset route.
// So this can only batch spot orders or futures orders, not both.
func (g *Gateio) WebsocketOrderPlace(ctx context.Context, batch []WebsocketOrder, a asset.Item) ([]WebsocketOrderResponse, error) {
	if len(batch) == 0 {
		return nil, errBatchSliceEmpty
	}

	for i := range batch {
		if batch[i].Text == "" {
			// For some reason the API requires a text field, or it will be
			// rejected in the second response. This is a workaround.
			// +1 index for uniqueness in batch, when clock hasn't updated yet.
			// TODO: Remove and use common counter.
			batch[i].Text = "t-" + strconv.FormatInt(time.Now().UnixNano()+int64(i), 10)
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

	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
	}

	if len(batch) == 1 {
		var singleResponse WebsocketOrderResponse
		err = g.SendWebsocketRequest(ctx, "spot.order_place", route, batch[0], &singleResponse, 2)
		return []WebsocketOrderResponse{singleResponse}, err
	}

	var resp []WebsocketOrderResponse
	err = g.SendWebsocketRequest(ctx, "spot.order_place", route, batch, &resp, 2)
	return resp, err
}

// WebsocketOrderCancel cancels an order via the websocket connection
func (g *Gateio) WebsocketOrderCancel(ctx context.Context, orderID string, pair currency.Pair, account string, a asset.Item) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
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
	err = g.SendWebsocketRequest(ctx, "spot.order_cancel", route, params, &resp, 1)
	return &resp, err
}

// WebsocketOrderCancelAllByIDs cancels multiple orders via the websocket
func (g *Gateio) WebsocketOrderCancelAllByIDs(ctx context.Context, o []WebsocketOrderCancelRequest, a asset.Item) ([]WebsocketCancellAllResponse, error) {
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

	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
	}

	var resp []WebsocketCancellAllResponse
	err = g.SendWebsocketRequest(ctx, "spot.order_cancel_ids", route, o, &resp, 2)
	return resp, err
}

// WebsocketOrderCancelAllByPair cancels all orders for a specific pair
func (g *Gateio) WebsocketOrderCancelAllByPair(ctx context.Context, pair currency.Pair, side order.Side, account string, a asset.Item) ([]WebsocketOrderResponse, error) {
	if !pair.IsEmpty() && side == order.UnknownSide {
		return nil, fmt.Errorf("%w: side cannot be unknown when pair is set as this will purge *ALL* open orders", errEdgeCaseIssue)
	}

	sideStr := ""
	if side != order.UnknownSide {
		sideStr = side.Lower()
	}

	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
	}

	params := &WebsocketCancelParam{
		Pair:    pair,
		Side:    sideStr,
		Account: account,
	}

	var resp []WebsocketOrderResponse
	err = g.SendWebsocketRequest(ctx, "spot.order_cancel_cp", route, params, &resp, 1)
	return resp, err
}

// WebsocketOrderAmend amends an order via the websocket connection
func (g *Gateio) WebsocketOrderAmend(ctx context.Context, amend *WebsocketAmendOrder, a asset.Item) (*WebsocketOrderResponse, error) {
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

	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
	}

	var resp WebsocketOrderResponse
	err = g.SendWebsocketRequest(ctx, "spot.order_amend", route, amend, &resp, 1)
	return &resp, err
}

// WebsocketGetOrderStatus gets the status of an order via the websocket connection
func (g *Gateio) WebsocketGetOrderStatus(ctx context.Context, orderID string, pair currency.Pair, account string, a asset.Item) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	route, err := g.GetWebsocketRoute(a)
	if err != nil {
		return nil, err
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
	err = g.SendWebsocketRequest(ctx, "spot.order_status", route, params, &resp, 1)
	return &resp, err
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, channel, route string, params, result any, expectedResponses int) error {
	paramPayload, err := json.Marshal(params)
	if err != nil {
		return err
	}

	conn, err := g.Websocket.GetOutboundConnection(route)
	if err != nil {
		return err
	}

	tn := time.Now()
	request := &WebsocketRequest{
		Time:    tn.Unix(),
		Channel: channel,
		Event:   "api",
		Payload: WebsocketPayload{
			// This request ID associated with the payload is the match to the
			// response.
			RequestID:    strconv.FormatInt(tn.UnixNano(), 10),
			RequestParam: paramPayload,
			Timestamp:    strconv.FormatInt(tn.Unix(), 10),
		},
	}

	responses, err := conn.SendMessageReturnResponses(ctx, request.Payload.RequestID, request, expectedResponses, InspectPayloadForAck)
	if err != nil {
		return err
	}

	if len(responses) == 0 {
		return fmt.Errorf("no responses received")
	}

	var inbound WebsocketAPIResponse
	// The last response is the one we want to unmarshal, the other is just
	// an ack. If the request fails on the ACK then we can unmarshal the error
	// from that as the next response won't come anyway.
	endResponse := responses[len(responses)-1]

	err = json.Unmarshal(endResponse, &inbound)
	if err != nil {
		return err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		err = json.Unmarshal(inbound.Data, &wsErr)
		if err != nil {
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