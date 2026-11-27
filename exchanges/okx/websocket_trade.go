package okx

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errMassCancelFailed       = errors.New("mass cancel failed")
	errLockIntervalOutOfRange = errors.New("lock interval out of range")
)

// maxWebsocketBatchOrders is the documented maximum of orders a batch place,
// cancel or amend operation takes
const maxWebsocketBatchOrders = 20

// PlaceOrder places an order through the private websocket connection
func (w WebsocketAPI) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*OrderActionResponse, error) {
	if err := validateWebsocketPlace(arg); err != nil {
		return nil, err
	}
	var resp []*OrderActionResponse
	if err := w.e.sendWebsocketRequest(ctx, privateConnection, placeOrderEPL, "order", []PlaceOrderRequest{*arg}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// PlaceMultipleOrders places up to 20 orders in one request through the
// private websocket connection. A partial success returns every order's
// result beside errPartialSuccess.
func (w WebsocketAPI) PlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequest) ([]*OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}
	if len(args) > maxWebsocketBatchOrders {
		return nil, fmt.Errorf("%w, cannot place more than %d orders", errExceedLimit, maxWebsocketBatchOrders)
	}
	for i := range args {
		if err := validateWebsocketPlace(&args[i]); err != nil {
			return nil, err
		}
	}
	epl := batchEndpointLimit(placeOrderEPL, placeMultipleOrdersEPL, len(args))
	var resp []*OrderActionResponse
	return resp, w.e.sendWebsocketRequest(ctx, privateConnection, epl, "batch-orders", args, &resp)
}

// validateWebsocketPlace checks an order carries the fields every placement
// needs and its instrument's ID code, which the websocket operations identify
// instruments by in place of the instrument ID, and no take profit or stop
// loss, which OKX's websocket order operation does not take
func validateWebsocketPlace(arg *PlaceOrderRequest) error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if err := arg.validateOrder(); err != nil {
		return err
	}
	if arg.InstrumentIDCode == 0 {
		return fmt.Errorf("%w: %s", errMissingInstrumentIDCode, arg.InstrumentID)
	}
	if len(arg.AttachAlgoOrders) != 0 {
		return fmt.Errorf("%w: OKX's websocket order operation takes none", errAttachedOrderUnsupported)
	}
	return nil
}

// CancelSingleOrder cancels an order through the private websocket connection
func (w WebsocketAPI) CancelSingleOrder(ctx context.Context, arg *CancelOrderRequest) (*OrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if err := validateWebsocketCancel(arg); err != nil {
		return nil, err
	}
	var resp []*OrderActionResponse
	if err := w.e.sendWebsocketRequest(ctx, privateConnection, cancelOrderEPL, "cancel-order", []CancelOrderRequest{*arg}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// CancelMultipleOrders cancels up to 20 orders in one request through the
// private websocket connection. A partial success returns every order's
// result beside errPartialSuccess.
func (w WebsocketAPI) CancelMultipleOrders(ctx context.Context, args []CancelOrderRequest) ([]*OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}
	if len(args) > maxWebsocketBatchOrders {
		return nil, fmt.Errorf("%w, cannot cancel more than %d orders", errExceedLimit, maxWebsocketBatchOrders)
	}
	for i := range args {
		if err := validateWebsocketCancel(&args[i]); err != nil {
			return nil, err
		}
	}
	epl := batchEndpointLimit(cancelOrderEPL, cancelMultipleOrdersEPL, len(args))
	var resp []*OrderActionResponse
	return resp, w.e.sendWebsocketRequest(ctx, privateConnection, epl, "batch-cancel-orders", args, &resp)
}

// validateWebsocketCancel checks a cancellation carries an order ID and its
// instrument's ID code, which the websocket operations identify instruments by
// in place of the instrument ID
func validateWebsocketCancel(arg *CancelOrderRequest) error {
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.InstrumentIDCode == 0 {
		return fmt.Errorf("%w: %s", errMissingInstrumentIDCode, arg.InstrumentID)
	}
	return nil
}

// AmendOrder amends an order through the private websocket connection
func (w WebsocketAPI) AmendOrder(ctx context.Context, arg *AmendOrderRequest) (*OrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if err := validateWebsocketAmend(arg); err != nil {
		return nil, err
	}
	var resp []*OrderActionResponse
	if err := w.e.sendWebsocketRequest(ctx, privateConnection, amendOrderEPL, "amend-order", []AmendOrderRequest{*arg}, &resp); err != nil {
		return nil, err
	}
	return singleItem(resp)
}

// AmendMultipleOrders amends up to 20 orders in one request through the
// private websocket connection. A partial success returns every order's
// result beside errPartialSuccess.
func (w WebsocketAPI) AmendMultipleOrders(ctx context.Context, args []AmendOrderRequest) ([]*OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}
	if len(args) > maxWebsocketBatchOrders {
		return nil, fmt.Errorf("%w, cannot amend more than %d orders", errExceedLimit, maxWebsocketBatchOrders)
	}
	for i := range args {
		if err := validateWebsocketAmend(&args[i]); err != nil {
			return nil, err
		}
	}
	epl := batchEndpointLimit(amendOrderEPL, amendMultipleOrdersEPL, len(args))
	var resp []*OrderActionResponse
	return resp, w.e.sendWebsocketRequest(ctx, privateConnection, epl, "batch-amend-orders", args, &resp)
}

// validateWebsocketAmend checks an amendment carries the fields every
// amendment needs and its instrument's ID code, which the websocket operations
// identify instruments by in place of the instrument ID, and no take profit or
// stop loss, which OKX's websocket amend operation does not take
func validateWebsocketAmend(arg *AmendOrderRequest) error {
	if err := arg.validateAmendment(); err != nil {
		return err
	}
	if arg.InstrumentIDCode == 0 {
		return fmt.Errorf("%w: %s", errMissingInstrumentIDCode, arg.InstrumentID)
	}
	if len(arg.AttachAlgoOrders) != 0 {
		return fmt.Errorf("%w: OKX's websocket amend operation takes none", errAttachedOrderUnsupported)
	}
	return nil
}

// CancelAllMMPOrders cancels all the market maker protection pending orders
// of an option instrument family through the private websocket connection.
// It needs the portfolio margin mode and the MMP privilege.
func (w WebsocketAPI) CancelAllMMPOrders(ctx context.Context, arg *MassCancelOrderRequest) error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentType == "" {
		return fmt.Errorf("%w, instrument type can not be empty", errInvalidInstrumentType)
	}
	if arg.InstrumentFamily == "" {
		return errInstrumentFamilyRequired
	}
	if arg.LockInterval > 10000 {
		return fmt.Errorf("%w: %d milliseconds is above 10000", errLockIntervalOutOfRange, arg.LockInterval)
	}
	var resps []*ResultResponse
	if err := w.e.sendWebsocketRequest(ctx, privateConnection, massCancelMMPOrderEPL, "mass-cancel", []MassCancelOrderRequest{*arg}, &resps); err != nil {
		return err
	}
	resp, err := singleItem(resps)
	if err != nil {
		return err
	}
	if !resp.Result {
		return errMassCancelFailed
	}
	return nil
}
