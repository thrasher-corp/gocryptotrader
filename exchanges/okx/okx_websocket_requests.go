package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var errInvalidWebsocketRequest = errors.New("invalid websocket request")

// WsPlaceOrder places an order thought the websocket connection stream, and returns a SubmitResponse and error message.
func (ok *Okx) WsPlaceOrder(ctx context.Context, arg *PlaceOrderRequestParam) (*OrderData, error) {
	err := arg.Validate()
	if err != nil {
		return nil, err
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var orderResp []OrderData
	err = ok.SendAuthenticatedWebsocketRequest(ctx, id, "order", []PlaceOrderRequestParam{*arg}, &orderResp)
	if err != nil {
		return nil, err
	}

	if len(orderResp) != 1 {
		return nil, errNoValidResponseFromServer
	}

	if orderResp[0].SCode > 0 {
		return nil, fmt.Errorf("error code: %d message: %s", orderResp[0].SCode, orderResp[0].SMessage)
	}

	return &orderResp[0], nil
}

// WsPlaceMultipleOrder creates an order through the websocket stream.
func (ok *Okx) WsPlaceMultipleOrder(ctx context.Context, args []PlaceOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, common.ErrNilPointer)
	}

	for x := range args {
		err := args[x].Validate()
		if err != nil {
			return nil, err
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	// Opted to not check scode here as some orders may be successful and some
	// may not. So return everything to the caller to handle.
	var orderResp []OrderData
	return orderResp, ok.SendAuthenticatedWebsocketRequest(ctx, id, "batch-orders", args, &orderResp)
}

// WsCancelOrder websocket function to cancel a trade order
func (ok *Okx) WsCancelOrder(ctx context.Context, arg CancelOrderRequestParam) (*OrderData, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var orderResp []OrderData
	err := ok.SendAuthenticatedWebsocketRequest(ctx, id, "cancel-order", []CancelOrderRequestParam{arg}, &orderResp)
	if err != nil {
		return nil, err
	}

	if len(orderResp) != 1 {
		return nil, errNoValidResponseFromServer
	}

	if orderResp[0].SCode > 0 {
		return nil, fmt.Errorf("error code: %d message: %s", orderResp[0].SCode, orderResp[0].SMessage)
	}
	return &orderResp[0], nil
}

// WsCancelMultipleOrder cancel multiple order through the websocket channel.
func (ok *Okx) WsCancelMultipleOrder(ctx context.Context, args []CancelOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, common.ErrNilPointer)
	}

	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientOrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	// Opted to not check scode here as some orders may be successful and some
	// may not. So return everything to the caller to handle.
	var orderResp []OrderData
	return orderResp, ok.SendAuthenticatedWebsocketRequest(ctx, id, "batch-cancel-orders", args, &orderResp)
}

// WsAmendOrder method to amend trade order using a request thought the websocket channel.
func (ok *Okx) WsAmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	var orderResp []OrderData
	err := ok.SendAuthenticatedWebsocketRequest(ctx, id, "amend-order", []AmendOrderRequestParams{*arg}, &orderResp)
	if err != nil {
		return nil, err
	}

	if len(orderResp) != 1 {
		return nil, errNoValidResponseFromServer
	}

	if orderResp[0].SCode > 0 {
		return nil, fmt.Errorf("error code: %d message: %s", orderResp[0].SCode, orderResp[0].SMessage)
	}

	return &orderResp[0], nil
}

// WsAmendMultipleOrders a request through the websocket connection to amend multiple trade orders.
func (ok *Okx) WsAmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%T: %w", args, common.ErrNilPointer)
	}

	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}

	id := strconv.FormatInt(ok.Websocket.AuthConn.GenerateMessageID(false), 10)

	// Opted to not check scode here as some orders may be successful and some
	// may not. So return everything to the caller to handle.
	var orderResp []OrderData
	return orderResp, ok.SendAuthenticatedWebsocketRequest(ctx, id, "batch-amend-orders", args, &orderResp)
}

// SendAuthenticatedWebsocketRequest sends a websocket request to the server
// TODO: Websocket request endpoint rate limits are shared with their REST API counterparts. A future update should
// access request level rate limits and update the rate limiter accordingly.
func (ok *Okx) SendAuthenticatedWebsocketRequest(ctx context.Context, id, operation string, payload, response any) error {
	if id == "" || operation == "" || payload == nil || response == nil {
		return errInvalidWebsocketRequest
	}

	outbound := &struct {
		ID        string      `json:"id"`
		Operation string      `json:"op"`
		Arguments interface{} `json:"args"`
	}{
		ID:        id,
		Operation: operation,
		Arguments: payload,
	}

	if ok.Verbose {
		if payload, err := json.Marshal(outbound); err != nil {
			log.Debugf(log.ExchangeSys, "%s sending outbound request via websocket: %v", ok.Name, string(payload))
		}
	}

	incoming, err := ok.Websocket.AuthConn.SendMessageReturnResponse(ctx, id, outbound)
	if err != nil {
		return err
	}

	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "%s received incoming request response via websocket: %v", ok.Name, string(incoming))
	}

	intermediary := struct {
		ID        string `json:"id"`
		Operation string `json:"op"`
		Code      int    `json:"code,string"`
		Message   string `json:"msg"`
		Data      any    `json:"data"` // pointer to type
		InTime    string `json:"inTime"`
		OutTime   string `json:"outTime"`
	}{
		Data: response,
	}

	err = json.Unmarshal(incoming, &intermediary)
	if err != nil {
		return err
	}

	if intermediary.Code > 1 {
		return fmt.Errorf("error code: %d message: %s", intermediary.Code, intermediary.Message)
	}

	return nil
}
