package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	deribitWebsocketAddress = "wss://www.deribit.com/ws" + deribitAPIVersion
	rpcVersion              = "2.0"
	rateLimit               = 20
	errAuthFailed           = 1002
)

// WsConnect starts a new connection with the websocket API
func (d *Deribit) WsConnect() error {
	if !d.Websocket.IsEnabled() || !d.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := d.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		println("Failed to connect ")
		return err
	}

	d.Websocket.Wg.Add(1)
	go d.wsReadData()

	if d.Websocket.CanUseAuthenticatedEndpoints() {
		err = d.wsLogin(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", d.Name, err)
		}

		d.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	return nil
}

func (d *Deribit) wsLogin(ctx context.Context) error {
	if !d.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", d.Name)
	}
	creds, err := d.GetCredentials(ctx)
	if err != nil {
		return err
	}
	d.Websocket.SetCanUseAuthenticatedEndpoints(true)

	data := ""
	n := d.Requester.GetNonce(true)
	strTS := strconv.FormatInt(time.Now().Unix()*1000, 10)
	str2Sign := fmt.Sprintf("%s\n%s\n%s", strTS, n.String(), data)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(str2Sign),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	request := WsRequest{
		JSONRPCVersion: rpcVersion,
		Method:         "/public/auth",
		ID:             d.Websocket.Conn.GenerateMessageID(false),
		Params: map[string]interface{}{
			"grant_type": "client_signature",
			"client_id":  creds.Key,
			"timestamp":  strTS,
			"nonce":      n.String(),
			"data":       "",
			"signature":  crypto.HexEncodeToString(hmac),
		},
	}

	resp, err := d.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		println("Authentication failed: ", err.Error())
		d.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}

	var response wsLoginResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return fmt.Errorf("%v %v", d.Name, err)
	}
	if response.Error != nil && (response.Error.Code > 0 || response.Error.Message != "") {
		return fmt.Errorf("%v Error:%v Message:%v", d.Name, response.Error.Code, response.Error.Message)
	}

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (d *Deribit) wsReadData() {
	defer d.Websocket.Wg.Done()

	for {
		resp := d.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		err := d.wsHandleData(resp.Raw)
		if err != nil {
			d.Websocket.DataHandler <- err
		}
	}
}

func (d *Deribit) wsHandleData(respRaw []byte) error {
	var response wsResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return fmt.Errorf("%s - err %s could not parse websocket data: %s",
			d.Name,
			err,
			respRaw)
	}

	id := response.ID
	if id > 0 && !d.Websocket.Match.IncomingWithData(id, respRaw) {
		return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %d", id)
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (d *Deribit) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (d *Deribit) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (d *Deribit) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return nil
}

// wsPlaceOrder sends a websocket message to submit an order
func (d *Deribit) wsPlaceOrder(instrumentName, orderType, label, timeInForce, trigger, advanced string, amount, price, maxShow, triggerPrice float64, postOnly, rejectPostOnly, reduceOnly, mmp bool) (*PrivateTradeData, error) {
	if !d.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", d.Name)
	}

	params := map[string]interface{}{
		"instrument_name": instrumentName,
		"amount":          strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if orderType != "" {
		params["type"] = orderType
	}
	if price != 0 {
		params["price"] = strconv.FormatFloat(amount, 'f', -1, 64)
	}
	if label != "" {
		params["label"] = label
	}
	if timeInForce != "" {
		params["time_in_force"] = timeInForce
	}
	if maxShow != 0 {
		params["max_show"] = strconv.FormatFloat(maxShow, 'f', -1, 64)
	}
	postOnlyStr := falseStr
	if postOnly {
		postOnlyStr = trueStr
	}
	params["post_only"] = postOnlyStr
	rejectPostOnlyStr := falseStr
	if rejectPostOnly {
		rejectPostOnlyStr = trueStr
	}
	params["reject_post_only"] = rejectPostOnlyStr
	reduceOnlyStr := falseStr
	if reduceOnly {
		reduceOnlyStr = trueStr
	}
	params["reduce_only"] = reduceOnlyStr
	mmpStr := falseStr
	if mmp {
		mmpStr = trueStr
	}
	params["mmp"] = mmpStr
	if triggerPrice != 0 {
		params["trigger_price"] = strconv.FormatFloat(triggerPrice, 'f', -1, 64)
	}
	if trigger != "" {
		params["trigger"] = trigger
	}
	if advanced != "" {
		params["advanced"] = advanced
	}

	id := d.Websocket.Conn.GenerateMessageID(false)
	request := WsRequest{
		JSONRPCVersion: rpcVersion,
		Method:         "/private/buy",
		ID:             id,
		Params:         params,
	}
	resp, err := d.Websocket.Conn.SendMessageReturnResponse(id, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", d.Name, err)
	}
	var response wsSubmitOrderResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", d.Name, err)
	}
	if response.Error != nil && (response.Error.Code > 0 || response.Error.Message != "") {
		return nil, fmt.Errorf("%v Error:%v Message:%v", d.Name, response.Error.Code, response.Error.Message)
	}
	return response.Result, nil
}
