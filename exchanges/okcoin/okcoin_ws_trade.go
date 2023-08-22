package okcoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WsPlaceOrder place trade order through the websocket channel.
func (o *Okcoin) WsPlaceOrder(arg *PlaceTradeOrderParam) (*TradeOrderResponse, error) {
	err := arg.validateTradeOrderParameter()
	if err != nil {
		return nil, err
	}
	var resp []TradeOrderResponse
	err = o.SendWebsocketRequest("order", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].SCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].SCode, resp[0].SMsg)
	}
	return &resp[0], nil
}

// WsPlaceMultipleOrder place orders in batches through the websocket stream. Maximum 20 orders can be placed per request. Request parameters should be passed in the form of an array.
func (o *Okcoin) WsPlaceMultipleOrder(args []PlaceTradeOrderParam) ([]TradeOrderResponse, error) {
	var err error
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, 0 length place order requests", errNilArgument)
	}
	for x := range args {
		err = args[x].validateTradeOrderParameter()
		if err != nil {
			return nil, err
		}
	}
	var resp []TradeOrderResponse
	return resp, o.SendWebsocketRequest("batch-orders", args, &resp, true)
}

// WsCancelTradeOrder cancels a single trade order through the websocket stream.
func (o *Okcoin) WsCancelTradeOrder(arg *CancelTradeOrderRequest) (*TradeOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errOrderIDOrClientOrderIDRequired
	}
	var resp []TradeOrderResponse
	err := o.SendWebsocketRequest("cancel-order", &arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].SCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].SCode, resp[0].SMsg)
	}
	return &resp[0], nil
}

// WsCancelMultipleOrders cancel incomplete orders in batches through the websocket stream. Maximum 20 orders can be canceled per request.
// Request parameters should be passed in the form of an array.
func (o *Okcoin) WsCancelMultipleOrders(args []CancelTradeOrderRequest) ([]TradeOrderResponse, error) {
	var err error
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, 0 length place order requests", errNilArgument)
	}
	for x := range args {
		err = args[x].validate()
		if err != nil {
			return nil, err
		}
	}
	var resp []TradeOrderResponse
	return resp, o.SendWebsocketRequest("batch-cancel-orders", args, &resp, true)
}

// WsAmendOrder amends an incomplete order through the websocket connection
func (o *Okcoin) WsAmendOrder(arg *AmendTradeOrderRequestParam) (*AmendTradeOrderResponse, error) {
	err := arg.validate()
	if err != nil {
		return nil, err
	}
	var resp []AmendTradeOrderResponse
	err = o.SendWebsocketRequest("amend-order", &arg, &resp, true)
	if err != nil {
		if len(resp) > 0 && resp[0].StatusCode != "0" && resp[0].StatusCode != "" {
			return nil, fmt.Errorf("%w, code: %s msg: %s", err, resp[0].StatusCode, resp[0].StatusMessage)
		}
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].StatusCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].StatusCode, resp[0].StatusMessage)
	}
	return &resp[0], nil
}

// WsAmendMultipleOrder amends multiple trade orders.
func (o *Okcoin) WsAmendMultipleOrder(args []AmendTradeOrderRequestParam) ([]AmendTradeOrderResponse, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, please provide at least one trade order amendment request", errNilArgument)
	}
	for x := range args {
		err := args[x].validate()
		if err != nil {
			return nil, err
		}
	}
	var resp []AmendTradeOrderResponse
	return resp, o.SendWebsocketRequest("batch-amend-orders", args, &resp, true)
}

// SendWebsocketRequest send a request through the websocket connection.
func (o *Okcoin) SendWebsocketRequest(operation string, data, result interface{}, authenticated bool) error {
	switch {
	case !o.Websocket.IsEnabled():
		return errors.New(stream.WebsocketNotEnabled)
	case !o.Websocket.IsConnected():
		return stream.ErrNotConnected
	case !o.Websocket.CanUseAuthenticatedEndpoints() && authenticated:
		return errors.New("websocket connection not authenticated")
	}
	req := &struct {
		ID        string      `json:"id"`
		Operation string      `json:"op"`
		Arguments interface{} `json:"args"`
	}{
		ID:        strconv.FormatInt(o.Websocket.Conn.GenerateMessageID(false), 10),
		Operation: operation,
	}
	if reflect.TypeOf(data).Kind() == reflect.Slice {
		req.Arguments = data
	} else {
		req.Arguments = []interface{}{data}
	}
	var byteData []byte
	var err error
	// TODO: ratelimits for websocket
	if authenticated {
		byteData, err = o.Websocket.AuthConn.SendMessageReturnResponse(req.ID, req)
	} else {
		byteData, err = o.Websocket.Conn.SendMessageReturnResponse(req.ID, req)
	}
	if err != nil {
		return err
	}
	response := struct {
		ID        string      `json:"id"`
		Operation string      `json:"op"`
		Data      interface{} `json:"data"`
		Code      string      `json:"code"`
		Message   string      `json:"msg"`
	}{
		Data: &result,
	}
	err = json.Unmarshal(byteData, &response)
	if err != nil {
		return err
	}
	if response.Code != "" && response.Code != "0" && response.Code != "1" && response.Code != "2" {
		if response.Message == "" {
			response.Message = websocketErrorCodes[response.Code]
		}
		return fmt.Errorf("%s websocket error code: %s message: %s", o.Name, response.Code, response.Message)
	}
	return nil
}
