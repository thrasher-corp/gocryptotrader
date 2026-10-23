package gate

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

// WebsocketSpotSubmitOrders submits orders via the websocket connection in one request, which takes up to 10 orders for
// each of up to 4 currency pairs
func (e *Exchange) WebsocketSpotSubmitOrders(ctx context.Context, orders ...*CreateOrderRequest) ([]*WebsocketOrderResponse, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}

	pairs := make(currency.Pairs, len(orders))
	for i := range orders {
		if orders[i].Text == "" {
			// API requires Text field, or it will be rejected
			orders[i].Text = "t-" + strconv.FormatInt(e.messageIDSeq.IncrementAndGet(), 10)
		}
		if orders[i].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if orders[i].Side != order.Buy.Lower() && orders[i].Side != order.Sell.Lower() {
			return nil, order.ErrSideIsInvalid
		}
		if orders[i].Amount == 0 {
			return nil, order.ErrAmountIsInvalid
		}
		if orders[i].Type == "limit" && orders[i].Price == 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
		pairs[i] = orders[i].CurrencyPair
	}
	if err := checkSpotBatchLimits(pairs); err != nil {
		return nil, err
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

	var resp *WebsocketOrderResponse
	if err := e.SendWebsocketRequest(ctx, spotCancelSingleOrderEPL, "spot.order_cancel", asset.Spot, &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}, &resp, 1); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// WebsocketSpotCancelAllOrdersByIDs cancels multiple orders via the websocket
func (e *Exchange) WebsocketSpotCancelAllOrdersByIDs(ctx context.Context, o []WebsocketOrderBatchRequest) ([]*WebsocketCancellAllResponse, error) {
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

	var resp []*WebsocketCancellAllResponse
	return resp, e.SendWebsocketRequest(ctx, spotCancelBatchOrdersEPL, "spot.order_cancel_ids", asset.Spot, o, &resp, 2)
}

// WebsocketSpotCancelAllOrdersByPair cancels all orders for a specific pair
func (e *Exchange) WebsocketSpotCancelAllOrdersByPair(ctx context.Context, pair currency.Pair, side order.Side, account string) ([]*WebsocketOrderResponse, error) {
	if !pair.IsEmpty() && side == order.UnknownSide {
		// This case will cancel all orders for every pair, this can be introduced later
		return nil, fmt.Errorf("'%v' %w while pair is set", side, order.ErrSideIsInvalid)
	}

	sideStr := ""
	if side != order.UnknownSide {
		sideStr = side.Lower()
	}

	var resp []*WebsocketOrderResponse
	return resp, e.SendWebsocketRequest(ctx, spotCancelAllOpenOrdersEPL, "spot.order_cancel_cp", asset.Spot, &WebsocketCancelRequest{
		Pair:    pair,
		Side:    sideStr,
		Account: account,
	}, &resp, 1)
}

// WebsocketSpotAmendOrder amends an order via the websocket connection
func (e *Exchange) WebsocketSpotAmendOrder(ctx context.Context, amend *WebsocketAmendOrderRequest) (*WebsocketOrderResponse, error) {
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
		return nil, fmt.Errorf("%w: amount or price must be set", order.ErrAmountIsInvalid)
	}
	var resp *WebsocketOrderResponse
	return resp, e.SendWebsocketRequest(ctx, spotAmendOrderEPL, "spot.order_amend", asset.Spot, amend, &resp, 1)
}

// WebsocketSpotListOrders lists spot orders via the websocket connection
func (e *Exchange) WebsocketSpotListOrders(ctx context.Context, arg *WebsocketSpotListOrdersRequest) ([]*WebsocketOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	switch arg.Status {
	case statusOpen:
		if arg.Pair.IsEmpty() {
			return nil, fmt.Errorf("%w: required for open orders", currency.ErrCurrencyPairEmpty)
		}
	case statusFinished:
	case "":
		return nil, errStatusNotSet
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidOrderStatus, arg.Status)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := &websocketSpotListOrdersParams{
		Status:  arg.Status,
		Page:    arg.Page,
		Limit:   arg.Limit,
		Account: arg.Account,
		Side:    arg.Side,
	}
	if !arg.Pair.IsEmpty() {
		params.Pair = arg.Pair.String()
	}
	if !arg.StartTime.IsZero() {
		params.From = unixSeconds(arg.StartTime)
	}
	if !arg.EndTime.IsZero() {
		params.To = unixSeconds(arg.EndTime)
	}
	var resp []*WebsocketOrderResponse
	return resp, e.SendWebsocketRequest(ctx, spotGetOrdersEPL, "spot.order_list", asset.Spot, params, &resp, 1)
}

// WebsocketSpotGetOrderStatus gets the status of an order via the websocket connection
func (e *Exchange) WebsocketSpotGetOrderStatus(ctx context.Context, orderID string, pair currency.Pair, account string) (*WebsocketOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *WebsocketOrderResponse
	return resp, e.SendWebsocketRequest(ctx, spotGetOrdersEPL, "spot.order_status", asset.Spot, &WebsocketOrderRequest{OrderID: orderID, Pair: pair.String(), Account: account}, &resp, 1)
}
