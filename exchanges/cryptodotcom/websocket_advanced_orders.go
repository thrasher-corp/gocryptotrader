package cryptodotcom

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
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
	if resp == nil {
		return nil, common.ErrNoResponse
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
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateOCO places a one-cancels-the-other strategy over the websocket, as Exchange.CreateOCO does over REST
func (w WebsocketAPI) CreateOCO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := ocoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-oco", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOCO cancels an OCO's orders by its list ID over the websocket, as Exchange.CancelOCO does over REST
func (w WebsocketAPI) CancelOCO(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-oco", params, nil)
}

// CreateOTO places a one-triggers-the-other strategy over the websocket, as Exchange.CreateOTO does over REST
func (w WebsocketAPI) CreateOTO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := otoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-oto", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOTO cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID over the websocket, as Exchange.CancelOTO
// does over REST
func (w WebsocketAPI) CancelOTO(ctx context.Context, listID string) error {
	params, err := newAdvancedListIDParams(listID)
	if err != nil {
		return err
	}
	return w.e.SendWebsocketRequest(ctx, "private/advanced/cancel-oto", params, nil)
}

// CreateOTOCO places a one-triggers-a-one-cancels-the-other strategy over the websocket, as Exchange.CreateOTOCO does
// over REST
func (w WebsocketAPI) CreateOTOCO(ctx context.Context, orders []AdvancedOrderLegRequest) (*AdvancedOrderListIDResponse, error) {
	params, err := otocoParams(orders)
	if err != nil {
		return nil, err
	}
	var resp *AdvancedOrderListIDResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/advanced/create-otoco", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOTOCO cancels a SPOT_ATTACH's or DERIV_ATTACH's orders by its list ID over the websocket, as
// Exchange.CancelOTOCO does over REST
func (w WebsocketAPI) CancelOTOCO(ctx context.Context, listID string) error {
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
	if resp == nil {
		return nil, common.ErrNoResponse
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
	if resp == nil {
		return nil, common.ErrNoResponse
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
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
