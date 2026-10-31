package cryptodotcom

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// CreateOrder creates a LIMIT or MARKET order over the websocket, as Exchange.CreateOrder does over REST
func (w WebsocketAPI) CreateOrder(ctx context.Context, r *CreateOrderRequest) (*OrderIDResponse, error) {
	params, err := newCreateOrderParams(r)
	if err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/create-order", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// AmendOrder amends an order's price and quantity over the websocket, as Exchange.AmendOrder does over REST
func (w WebsocketAPI) AmendOrder(ctx context.Context, r *AmendOrderRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/amend-order", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelExistingOrder cancels an order over the websocket, as Exchange.CancelExistingOrder does over REST
func (w WebsocketAPI) CancelExistingOrder(ctx context.Context, r *CancelOrderRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/cancel-order", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllExistingOrders cancels the orders of an instrument, or of every instrument, over the websocket, as
// Exchange.CancelAllExistingOrders does over REST
func (w WebsocketAPI) CancelAllExistingOrders(ctx context.Context, r *CancelAllOrdersRequest) error {
	if err := r.validate(); err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/cancel-all-orders", r, nil)
}

// ClosePosition closes a position over the websocket, as Exchange.ClosePosition does over REST
func (w WebsocketAPI) ClosePosition(ctx context.Context, r *ClosePositionRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/close-position", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateOrderList creates a list of 1 to 10 orders over the websocket, as Exchange.CreateOrderList does over REST
func (w WebsocketAPI) CreateOrderList(ctx context.Context, orders []CreateOrderRequest) ([]OrderListEntryResponse, error) {
	params, err := newCreateOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/create-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOrderList cancels a list of 1 to 10 orders over the websocket, as Exchange.CancelOrderList does over REST
func (w WebsocketAPI) CancelOrderList(ctx context.Context, orders []CancelOrderListEntryRequest) ([]OrderListEntryResponse, error) {
	params, err := newCancelOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/cancel-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SendBatchOrderRequest cancels and creates orders of one instrument in one request, which only the websocket offers.
// Crypto.com processes the cancels before the creates, each independently of the others, and confirms the request
// with each operation's outcome before the orders are cancelled or placed; a refused operation is reported in its
// entry's code and message rather than as an error
func (w WebsocketAPI) SendBatchOrderRequest(ctx context.Context, r *BatchOrderRequest) (*BatchOrderResponse, error) {
	params, err := newBatchOrderParams(r)
	if err != nil {
		return nil, err
	}
	var resp *BatchOrderResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/batch-order-request", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
