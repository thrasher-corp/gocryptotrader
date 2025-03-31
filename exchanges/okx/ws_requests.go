package okx

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidWebsocketRequest = errors.New("invalid websocket request")
	errOperationFailed         = errors.New("operation failed")
	errPartialSuccess          = errors.New("bulk operation partially succeeded")
	errMassCancelFailed        = errors.New("mass cancel failed")
)

// WSPlaceOrder submits an order
func (ok *Okx) WSPlaceOrder(ctx context.Context, arg *PlaceOrderRequestParam) (*OrderData, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, placeOrderEPL, id, "order", []PlaceOrderRequestParam{*arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WSPlaceMultipleOrder submits multiple orders
func (ok *Okx) WSPlaceMultipleOrder(ctx context.Context, args []PlaceOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}

	for i := range args {
		if err := args[i].Validate(); err != nil {
			return nil, err
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	err := ok.SendAuthenticatedWebsocketRequest(ctx, placeMultipleOrdersEPL, id, "batch-orders", args, &resp)
	if err != nil {
		if batchError := getOrderStatusErrors(resp); batchError != nil {
			err = fmt.Errorf("%w: %w", err, batchError)
		}
	}
	return resp, err
}

// WSCancelOrder cancels an order
func (ok *Okx) WSCancelOrder(ctx context.Context, arg *CancelOrderRequestParam) (*OrderData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, cancelOrderEPL, id, "cancel-order", []CancelOrderRequestParam{*arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WSCancelMultipleOrder cancels multiple orders
func (ok *Okx) WSCancelMultipleOrder(ctx context.Context, args []CancelOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}

	for i := range args {
		if args[i].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[i].OrderID == "" && args[i].ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	err := ok.SendAuthenticatedWebsocketRequest(ctx, cancelMultipleOrdersEPL, id, "batch-cancel-orders", args, &resp)
	if err != nil {
		if batchError := getOrderStatusErrors(resp); batchError != nil {
			err = fmt.Errorf("%w: %w", err, batchError)
		}
	}
	return resp, err
}

// WSAmendOrder amends an order
func (ok *Okx) WSAmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, amendOrderEPL, id, "amend-order", []AmendOrderRequestParams{*arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WSAmendMultipleOrders amends multiple orders
func (ok *Okx) WSAmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}

	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []OrderData
	err := ok.SendAuthenticatedWebsocketRequest(ctx, amendMultipleOrdersEPL, id, "batch-amend-orders", args, &resp)
	if err != nil {
		if batchError := getOrderStatusErrors(resp); batchError != nil {
			err = fmt.Errorf("%w: %w", err, batchError)
		}
	}
	return resp, err
}

// WSMassCancelOrders cancels all MMP pending orders of an instrument family. Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) WSMassCancelOrders(ctx context.Context, args []CancelMassReqParam) error {
	if len(args) == 0 {
		return fmt.Errorf("%T: %w", args, order.ErrSubmissionIsNil)
	}

	for x := range args {
		if args[x].InstrumentType == "" {
			return fmt.Errorf("%w, instrument type can not be empty", errInvalidInstrumentType)
		}
		if args[x].InstrumentFamily == "" {
			return errInstrumentFamilyRequired
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []struct {
		Result bool `json:"result"`
	}
	err := ok.SendAuthenticatedWebsocketRequest(ctx, amendOrderEPL, id, "mass-cancel", args, &resp)
	if err != nil {
		return err
	}

	if len(resp) == 0 {
		return common.ErrNoResponse
	}

	if len(resp) > 1 {
		return fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if !resp[0].Result {
		return errMassCancelFailed
	}

	return nil
}

// WSPlaceSpreadOrder submits a spread order
func (ok *Okx) WSPlaceSpreadOrder(ctx context.Context, arg *SpreadOrderParam) (*SpreadOrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []SpreadOrderResponse
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, placeSpreadOrderEPL, id, "sprd-order", []SpreadOrderParam{*arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WSAmendSpreadOrder amends a spread order
func (ok *Okx) WSAmendSpreadOrder(ctx context.Context, arg *AmendSpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewPrice == 0 && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []SpreadOrderResponse
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, amendSpreadOrderEPL, id, "sprd-amend-order", []AmendSpreadOrderParam{*arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WsCancelSpreadOrder cancels an incomplete spread order through the websocket connection.
func (ok *Okx) WsCancelSpreadOrder(ctx context.Context, orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	arg := make(map[string]string)
	if orderID != "" {
		arg["ordId"] = orderID
	}
	if clientOrderID != "" {
		arg["clOrdId"] = clientOrderID
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []SpreadOrderResponse
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, cancelSpreadOrderEPL, id, "sprd-cancel-order", []map[string]string{arg}, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return nil, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return nil, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return &resp[0], nil
}

// WSCancelAllSpreadOrders cancels all spread orders and return success message through the websocket channel.
func (ok *Okx) WSCancelAllSpreadOrders(ctx context.Context, spreadID string) (bool, error) {
	arg := make(map[string]string, 1)
	if spreadID != "" {
		arg["sprdId"] = spreadID
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var resp []ResponseResult
	if err := ok.SendAuthenticatedWebsocketRequest(ctx, cancelAllSpreadOrderEPL, id, "sprd-mass-cancel", []map[string]string{arg}, &resp); err != nil {
		return false, err
	}

	if len(resp) == 0 {
		return false, common.ErrNoResponse
	}

	if len(resp) > 1 {
		return false, fmt.Errorf("expected 1 response, received %d", len(resp))
	}

	if resp[0].StatusCode > 0 {
		return false, getStatusError(resp[0].StatusCode, resp[0].StatusMessage)
	}

	return resp[0].Result, nil
}

// SendAuthenticatedWebsocketRequest sends a websocket request to the server
func (ok *Okx) SendAuthenticatedWebsocketRequest(ctx context.Context, epl request.EndpointLimit, id, operation string, payload, result any) error {
	if operation == "" || payload == nil {
		return errInvalidWebsocketRequest
	}

	outbound := &struct {
		ID        string `json:"id"`
		Operation string `json:"op"`
		Arguments any    `json:"args"`
		// TODO: Add ExpTime to the struct, the struct should look like this:
		// ExpTime  string `json:"expTime,omitempty"` so a deadline can be set
		// Request effective deadline. Unix timestamp format in milliseconds, e.g. 1597026383085
	}{
		ID:        id,
		Operation: operation,
		Arguments: payload,
	}

	incoming, err := ok.Websocket.AuthConn.SendMessageReturnResponse(ctx, epl, id, outbound)
	if err != nil {
		return err
	}

	intermediary := struct {
		ID        string `json:"id"`
		Operation string `json:"op"`
		Code      int64  `json:"code,string"`
		Message   string `json:"msg"`
		Data      any    `json:"data"`
		InTime    string `json:"inTime"`
		OutTime   string `json:"outTime"`
	}{
		Data: result,
	}

	if err := json.Unmarshal(incoming, &intermediary); err != nil {
		return err
	}

	switch intermediary.Code {
	case 0:
		return nil
	case 1:
		return errOperationFailed
	case 2:
		return errPartialSuccess
	default:
		return getStatusError(intermediary.Code, intermediary.Message)
	}
}

func getOrderStatusErrors(resp []OrderData) error {
	var errs error
	for i := range resp {
		if err := resp[i].Error(); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("index: `%d` error: `%v`", i+1, err))
		}
	}
	return errs
}
