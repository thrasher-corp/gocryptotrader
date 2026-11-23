package okx

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errCancelAllSpreadOrdersFailed = errors.New("cancel all spread orders failed")

// PlaceSpreadOrder places a spread order through the business websocket
// connection which logs in
func (w WebsocketAPI) PlaceSpreadOrder(ctx context.Context, arg *PlaceSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp []*SpreadOrderResponse
	if err := w.e.sendWebsocketRequest(ctx, businessLoginConnection, placeSpreadOrderEPL, "sprd-order", []PlaceSpreadOrderRequest{*arg}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// AmendSpreadOrder amends a spread order through the business websocket
// connection which logs in
func (w WebsocketAPI) AmendSpreadOrder(ctx context.Context, arg *AmendSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewPrice == nil && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}
	var resp []*SpreadOrderResponse
	if err := w.e.sendWebsocketRequest(ctx, businessLoginConnection, amendSpreadOrderEPL, "sprd-amend-order", []AmendSpreadOrderRequest{*arg}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// CancelSpreadOrder cancels a spread order through the business websocket
// connection which logs in
func (w WebsocketAPI) CancelSpreadOrder(ctx context.Context, orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*SpreadOrderResponse
	if err := w.e.sendWebsocketRequest(ctx, businessLoginConnection, cancelSpreadOrderEPL, "sprd-cancel-order", []WebsocketCancelSpreadOrderArgument{{OrderID: orderID, ClientOrderID: clientOrderID}}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// CancelAllSpreadOrders cancels the orders of a spread, or of every spread
// when spreadID is empty, through the business websocket connection which
// logs in
func (w WebsocketAPI) CancelAllSpreadOrders(ctx context.Context, spreadID string) error {
	var resps []*ResultResponse
	if err := w.e.sendWebsocketRequest(ctx, businessLoginConnection, cancelAllSpreadOrderEPL, "sprd-mass-cancel", []WebsocketCancelAllSpreadOrdersArgument{{SpreadID: spreadID}}, &resps); err != nil {
		return err
	}
	resp, err := singleItem(resps)
	if err != nil {
		return err
	}
	if !resp.Result {
		return errCancelAllSpreadOrdersFailed
	}
	return nil
}
