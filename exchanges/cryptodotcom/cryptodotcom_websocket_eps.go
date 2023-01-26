package cryptodotcom

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// WsGetInstruments retrives information on all supported instruments through the public websocket connection.
func (cr *Cryptodotcom) WsGetInstruments() (*InstrumentList, error) {
	var resp *InstrumentList
	return resp, cr.SendWebsocketRequest(publicInstruments, nil, &resp, false)
}

// WsRetriveTrades fetches the public trades for a particular instrument through the websocket stream.
func (cr *Cryptodotcom) WsRetriveTrades(instrumentName string) (*TradesResponse, error) {
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params := make(map[string]interface{})
	params["instrument_name"] = instrumentName
	var resp *TradesResponse
	return resp, cr.SendWebsocketRequest(publicTrades, params, &resp, false)
}

// SendWebsocketRequest pushed a request data through the websocket data for authenticated and public messages.
func (cr *Cryptodotcom) SendWebsocketRequest(method string, arg map[string]interface{}, result interface{}, authenticated bool) error {
	val := reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr {
		return errors.New("response must to be pointer instance")
	}
	if authenticated && !cr.Websocket.CanUseAuthenticatedEndpoints() {
		return errors.New("can't send authenticated websocket request")
	}
	timestamp := time.Now()
	var id int64
	if authenticated {
		id = cr.Websocket.AuthConn.GenerateMessageID(false)
	} else {
		id = cr.Websocket.Conn.GenerateMessageID(false)
	}
	req := &WsRequestPayload{
		ID:     id,
		Method: method,
		Nonce:  timestamp.UnixMilli(),
		Params: arg,
	}
	var payload []byte
	var err error
	if authenticated {
		payload, err = cr.Websocket.AuthConn.SendMessageReturnResponse(req.ID, req)
	} else {
		val, _ := json.Marshal(req)
		println(string(val))
		payload, err = cr.Websocket.Conn.SendMessageReturnResponse(req.ID, req)
	}
	if err != nil {
		return err
	}
	response := &WSRespData{
		Result: &result,
	}
	err = json.Unmarshal(payload, response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		mes := fmt.Sprintf("error code: %d Message: %s", response.Code, response.Message)
		if response.DetailCode != "0" && response.DetailCode != "" {
			mes = fmt.Sprintf("%s Detail: %s %s", mes, response.DetailCode, response.DetailMessage)
		}
		return errors.New(mes)
	}
	return nil
}
