package gateio

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errOrdersEmpty      = errors.New("orders cannot be empty")
	errNoOrdersToCancel = errors.New("no orders to cancel")
	errChannelEmpty     = errors.New("channel cannot be empty")
)

// authenticateSpot sends an authentication message to the websocket connection
func (e *Exchange) authenticateSpot(ctx context.Context, conn websocket.Connection) error {
	return e.websocketLogin(ctx, conn, "spot.login")
}

// WebsocketSpotSubmitOrder submits an order via the websocket connection
func (e *Exchange) WebsocketSpotSubmitOrder(ctx context.Context, o *CreateOrderRequest) (*WebsocketOrderResponse, error) {
	resps, err := e.WebsocketSpotSubmitOrders(ctx, o)
	if err != nil {
		return nil, err
	}
	if len(resps) != 1 {
		return nil, common.ErrInvalidResponse
	}
	return resps[0], nil
}

// WebsocketSpotSubmitOrders submits orders via the websocket connection. You can
// send multiple orders in a single request. But only for one asset route.
func (e *Exchange) WebsocketSpotSubmitOrders(ctx context.Context, orders ...*CreateOrderRequest) ([]*WebsocketOrderResponse, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}

	for i := range orders {
		if orders[i].Text == "" {
			// API requires Text field, or it will be rejected
			orders[i].Text = "t-" + strconv.FormatInt(e.messageIDSeq.IncrementAndGet(), 10)
		}
		if orders[i].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if orders[i].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if orders[i].Amount == 0 {
			return nil, errInvalidAmount
		}
		if orders[i].Type == "limit" && orders[i].Price == 0 {
			return nil, errInvalidPrice
		}
	}

	if len(orders) == 1 {
		var singleResponse *WebsocketOrderResponse
		return []*WebsocketOrderResponse{singleResponse}, e.SendWebsocketRequest(ctx, spotPlaceOrderEPL, "spot.order_place", asset.Spot, orders[0], &singleResponse, 2)
	}
	var resp []*WebsocketOrderResponse
	return resp, e.SendWebsocketRequest(ctx, spotBatchOrdersEPL, "spot.order_place", asset.Spot, orders, &resp, 2)
}

// WebsocketSpotCancelOrder cancels an order via the websocket connection
func (e *Exchange) WebsocketSpotCancelOrder(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}

	var resp WebsocketOrderResponse
	return &resp, e.SendWebsocketRequest(ctx, spotCancelSingleOrderEPL, "spot.order_cancel", asset.Spot, params, &resp, 1)
}

// WebsocketSpotCancelAllOrdersByIDs cancels multiple orders via the websocket
func (e *Exchange) WebsocketSpotCancelAllOrdersByIDs(ctx context.Context, o []WebsocketOrderBatchRequest) ([]WebsocketCancellAllResponse, error) {
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
	return resp, e.SendWebsocketRequest(ctx, spotCancelBatchOrdersEPL, "spot.order_cancel_ids", asset.Spot, o, &resp, 2)
}

// WebsocketSpotCancelAllOrdersByPair cancels all orders for a specific pair
func (e *Exchange) WebsocketSpotCancelAllOrdersByPair(ctx context.Context, pair currency.Pair, side order.Side, account string) ([]WebsocketOrderResponse, error) {
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
	return resp, e.SendWebsocketRequest(ctx, spotCancelAllOpenOrdersEPL, "spot.order_cancel_cp", asset.Spot, params, &resp, 1)
}

// WebsocketSpotAmendOrder amends an order via the websocket connection
func (e *Exchange) WebsocketSpotAmendOrder(ctx context.Context, amend *WebsocketAmendOrder) (*WebsocketOrderResponse, error) {
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
	return &resp, e.SendWebsocketRequest(ctx, spotAmendOrderEPL, "spot.order_amend", asset.Spot, amend, &resp, 1)
}

// WebsocketSpotGetOrderStatus gets the status of an order via the websocket connection
func (e *Exchange) WebsocketSpotGetOrderStatus(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	params := &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}

	var resp WebsocketOrderResponse
	return &resp, e.SendWebsocketRequest(ctx, spotGetOrdersEPL, "spot.order_status", asset.Spot, params, &resp, 1)
}
