package cryptodotcom

import (
	"context"
)

// CreateAdvancedOrder places a trigger order over the websocket, as Exchange.CreateAdvancedOrder does over REST
func (w WebsocketAPI) CreateAdvancedOrder(ctx context.Context, req *CreateAdvancedOrderRequest) (*OrderIDResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-order", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AmendAdvancedOrder amends a trigger order over the websocket, as Exchange.AmendAdvancedOrder does over REST
func (w WebsocketAPI) AmendAdvancedOrder(ctx context.Context, req *AmendAdvancedOrderRequest) (*OrderIDResponse, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/amend-order", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateOneCancelsTheOtherOrder places a one-cancels-the-other strategy over the websocket, as
// Exchange.CreateOneCancelsTheOtherOrder does over REST
func (w WebsocketAPI) CreateOneCancelsTheOtherOrder(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := oneCancelsTheOtherParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-oco", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelOneCancelsTheOtherOrder cancels an OCO's orders by its list ID over the websocket, as
// Exchange.CancelOneCancelsTheOtherOrder does over REST
func (w WebsocketAPI) CancelOneCancelsTheOtherOrder(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-oco", params, nil)
}

// CreateOneTriggersTheOtherOrder places a one-triggers-the-other strategy over the websocket, as
// Exchange.CreateOneTriggersTheOtherOrder does over REST
func (w WebsocketAPI) CreateOneTriggersTheOtherOrder(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := oneTriggersTheOtherParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-oto", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelOneTriggersTheOtherOrder cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID over the websocket, as
// Exchange.CancelOneTriggersTheOtherOrder does over REST
func (w WebsocketAPI) CancelOneTriggersTheOtherOrder(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-oto", params, nil)
}

// CreateOneTriggersAOneCancelsTheOtherOrder places a one-triggers-a-one-cancels-the-other strategy over the websocket,
// as Exchange.CreateOneTriggersAOneCancelsTheOtherOrder does over REST
func (w WebsocketAPI) CreateOneTriggersAOneCancelsTheOtherOrder(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := oneTriggersAOneCancelsTheOtherParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-otoco", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelOneTriggersAOneCancelsTheOtherOrder cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID over the
// websocket, as Exchange.CancelOneTriggersAOneCancelsTheOtherOrder does over REST
func (w WebsocketAPI) CancelOneTriggersAOneCancelsTheOtherOrder(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-otoco", params, nil)
}

// CancelAdvancedOrder cancels one order of an advanced order over the websocket, as Exchange.CancelAdvancedOrder does
// over REST
func (w WebsocketAPI) CancelAdvancedOrder(ctx context.Context, req *AdvancedOrderIDRequest) (*OrderIDResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-order", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelAllAdvancedOrders cancels the advanced orders a request selects over the websocket, as
// Exchange.CancelAllAdvancedOrders does over REST
func (w WebsocketAPI) CancelAllAdvancedOrders(ctx context.Context, req *CancelAllAdvancedOrdersRequest) error {
	if err := req.validate(); err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-all-orders", req, nil)
}

// CreateAdvancedOrderList places up to 10 trigger orders over the websocket, as Exchange.CreateAdvancedOrderList does
// over REST
func (w WebsocketAPI) CreateAdvancedOrderList(ctx context.Context, orders []CreateAdvancedOrderRequest) ([]OrderListEntryResponse, error) {
	params, err := createAdvancedOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-order-list", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelAdvancedOrderList cancels up to 10 orders of advanced orders over the websocket, as
// Exchange.CancelAdvancedOrderList does over REST
func (w WebsocketAPI) CancelAdvancedOrderList(ctx context.Context, orders []AdvancedOrderIDRequest) ([]OrderListEntryResponse, error) {
	params, err := cancelAdvancedOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-order-list", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
