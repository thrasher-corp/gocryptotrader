package gateio

import (
	"context"
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
)

var (
	errOrdersEmpty      = errors.New("orders cannot be empty")
	errNoOrdersToCancel = errors.New("no orders to cancel")
	errChannelEmpty     = errors.New("channel cannot be empty")
)

// WebsocketSpotSubmitOrder submits an order via the websocket connection
func (g *Gateio) WebsocketSpotSubmitOrder(ctx context.Context, order *WebsocketOrder) ([]WebsocketOrderResponse, error) {
	return g.WebsocketSpotSubmitOrders(ctx, []WebsocketOrder{*order})
}

// WebsocketSpotSubmitOrders submits orders via the websocket connection. You can
// send multiple orders in a single request. But only for one asset route.
func (g *Gateio) WebsocketSpotSubmitOrders(ctx context.Context, orders []WebsocketOrder) ([]WebsocketOrderResponse, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}

	for i := range orders {
		if orders[i].Text == "" {
			// API requires Text field, or it will be rejected
			orders[i].Text = "t-" + strconv.FormatInt(g.Counter.IncrementAndGet(), 10)
		}
		if orders[i].CurrencyPair == "" {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if orders[i].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if orders[i].Amount == "" {
			return nil, errInvalidAmount
		}
		if orders[i].Type == "limit" && orders[i].Price == "" {
			return nil, errInvalidPrice
		}
	}

	if len(orders) == 1 {
		var singleResponse WebsocketOrderResponse
		return []WebsocketOrderResponse{singleResponse}, g.SendWebsocketRequest(ctx, spotPlaceOrderEPL, "spot.order_place", asset.Spot, orders[0], &singleResponse, 2)
	}
	var resp []WebsocketOrderResponse
	return resp, g.SendWebsocketRequest(ctx, spotBatchOrdersEPL, "spot.order_place", asset.Spot, orders, &resp, 2)
}

// WebsocketSpotCancelOrder cancels an order via the websocket connection
func (g *Gateio) WebsocketSpotCancelOrder(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}

	var resp WebsocketOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, spotCancelSingleOrderEPL, "spot.order_cancel", asset.Spot, params, &resp, 1)
}

// WebsocketSpotCancelAllOrdersByIDs cancels multiple orders via the websocket
func (g *Gateio) WebsocketSpotCancelAllOrdersByIDs(ctx context.Context, o []WebsocketOrderBatchRequest) ([]WebsocketCancellAllResponse, error) {
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
	return resp, g.SendWebsocketRequest(ctx, spotCancelBatchOrdersEPL, "spot.order_cancel_ids", asset.Spot, o, &resp, 2)
}

// WebsocketSpotCancelAllOrdersByPair cancels all orders for a specific pair
func (g *Gateio) WebsocketSpotCancelAllOrdersByPair(ctx context.Context, pair currency.Pair, side order.Side, account string) ([]WebsocketOrderResponse, error) {
	if !pair.IsEmpty() && side == order.UnknownSide {
		// This case will cancel all orders for every pair, this can be introduced later
		return nil, fmt.Errorf("'%v' %w while pair is set", side, order.ErrSideIsInvalid)
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
	return resp, g.SendWebsocketRequest(ctx, spotCancelAllOpenOrdersEPL, "spot.order_cancel_cp", asset.Spot, params, &resp, 1)
}

// WebsocketSpotAmendOrder amends an order via the websocket connection
func (g *Gateio) WebsocketSpotAmendOrder(ctx context.Context, amend *WebsocketAmendOrder) (*WebsocketOrderResponse, error) {
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
	return &resp, g.SendWebsocketRequest(ctx, spotAmendOrderEPL, "spot.order_amend", asset.Spot, amend, &resp, 1)
}

// WebsocketSpotGetOrderStatus gets the status of an order via the websocket connection
func (g *Gateio) WebsocketSpotGetOrderStatus(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}

	var resp WebsocketOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, spotGetOrdersEPL, "spot.order_status", asset.Spot, params, &resp, 1)
}

// funnelResult is used to unmarshal the result of a websocket request back to the required caller type
type funnelResult struct {
	Result any `json:"result"`
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, epl request.EndpointLimit, channel string, connSignature, params, result any, expectedResponses int) error {
	paramPayload, err := json.Marshal(params)
	if err != nil {
		return err
	}

	conn, err := g.Websocket.GetConnection(connSignature)
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

	responses, err := conn.SendMessageReturnResponsesWithInspector(ctx, epl, req.Payload.RequestID, req, expectedResponses, wsRespAckInspector{})
	if err != nil {
		return err
	}

	if len(responses) == 0 {
		return common.ErrNoResponse
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

	return json.Unmarshal(inbound.Data, &funnelResult{Result: result})
}

type wsRespAckInspector struct{}

// IsFinal checks the payload for an ack, it returns true if the payload does not contain an ack.
// This will force the cancellation of further waiting for responses.
func (wsRespAckInspector) IsFinal(data []byte) bool {
	return !strings.Contains(string(data), "ack")
}
