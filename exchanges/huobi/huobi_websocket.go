package huobi

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/common/crypto"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/asset"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	baseWSURL = "wss://api.huobi.pro"

	wsMarketURL   = baseWSURL + "/ws"
	wsMarketKline = "market.%s.kline.1min"
	wsMarketDepth = "market.%s.depth.step0"
	wsMarketTrade = "market.%s.trade.detail"

	wsAccountsOrdersEndPoint = "/ws/v1"
	wsAccountsList           = "accounts.list"
	wsOrdersList             = "orders.list"
	wsOrdersDetail           = "orders.detail"
	wsAccountsOrdersURL      = baseWSURL + wsAccountsOrdersEndPoint
	wsAccountListEndpoint    = wsAccountsOrdersEndPoint + "/" + wsAccountsList
	wsOrdersListEndpoint     = wsAccountsOrdersEndPoint + "/" + wsOrdersList
	wsOrdersDetailEndpoint   = wsAccountsOrdersEndPoint + "/" + wsOrdersDetail

	wsDateTimeFormatting = "2006-01-02T15:04:05"

	signatureMethod  = "HmacSHA256"
	signatureVersion = "2"
	requestOp        = "req"
	authOp           = "auth"
)

// Instantiates a communications channel between websocket connections
var comms = make(chan WsMessage, 1)

// WsConnect initiates a new websocket connection
func (h *HUOBI) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer

	if h.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(h.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	err := h.wsDial(&dialer)
	if err != nil {
		return err
	}
	err = h.wsAuthenticatedDial(&dialer)
	if err != nil {
		log.Errorf("%v - authenticated dial failed: %v", h.Name, err)
	}
	err = h.wsLogin()
	if err != nil {
		log.Errorf("%v - authentication failed: %v", h.Name, err)
	}

	go h.WsHandleData()
	h.GenerateDefaultSubscriptions()

	return nil
}

func (h *HUOBI) wsDial(dialer *websocket.Dialer) error {
	var err error
	var conStatus *http.Response
	h.WebsocketConn, conStatus, err = dialer.Dial(wsMarketURL, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", wsMarketURL, conStatus, conStatus.StatusCode, err)
	}
	go h.wsMultiConnectionFunnel(h.WebsocketConn, wsMarketURL)
	return nil
}

func (h *HUOBI) wsAuthenticatedDial(dialer *websocket.Dialer) error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	var err error
	var conStatus *http.Response
	h.AuthenticatedWebsocketConn, conStatus, err = dialer.Dial(wsAccountsOrdersURL, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", wsAccountsOrdersURL, conStatus, conStatus.StatusCode, err)
	}
	go h.wsMultiConnectionFunnel(h.AuthenticatedWebsocketConn, wsAccountsOrdersURL)
	return nil
}

// wsMultiConnectionFunnel manages data from multiple endpoints and passes it to a channel
func (h *HUOBI) wsMultiConnectionFunnel(ws *websocket.Conn, url string) {
	h.Websocket.Wg.Add(1)
	defer h.Websocket.Wg.Done()
	for {
		select {
		case <-h.Websocket.ShutdownC:
			return
		default:
			_, resp, err := ws.ReadMessage()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			h.Websocket.TrafficAlert <- struct{}{}
			b := bytes.NewReader(resp)
			gReader, err := gzip.NewReader(b)
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			unzipped, err := ioutil.ReadAll(gReader)
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			err = gReader.Close()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			comms <- WsMessage{Raw: unzipped, URL: url}
		}
	}
}

// WsHandleData handles data read from the websocket connection
func (h *HUOBI) WsHandleData() {
	h.Websocket.Wg.Add(1)
	defer h.Websocket.Wg.Done()
	for {
		select {
		case <-h.Websocket.ShutdownC:
			return
		case resp := <-comms:
			if h.Verbose {
				log.Debugf(log.SubSystemExchSys, "%v: %v: %v", h.Name, resp.URL, string(resp.Raw))
			}
			switch resp.URL {
			case wsMarketURL:
				h.wsHandleMarketData(resp)
			case wsAccountsOrdersURL:
				h.wsHandleAuthenticatedData(resp)
			}
		}
	}
}

func (h *HUOBI) wsHandleAuthenticatedData(resp WsMessage) {
	var init WsAuthenticatedDataResponse
	err := common.JSONDecode(resp.Raw, &init)
	if err != nil {
		h.Websocket.DataHandler <- err
		return
	}
	if init.ErrorCode > 0 {
		if init.ErrorMessage == "api-signature-not-valid" {
			h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		h.Websocket.DataHandler <- fmt.Errorf("%v %v Websocket error %v %s",
			h.Name,
			resp.URL,
			init.ErrorCode,
			init.ErrorMessage)
		return
	}
	if init.Ping != 0 {
		err = h.WebsocketConn.WriteJSON(`{"pong":1337}`)
		if err != nil {
			log.Error(log.SubSystemExchSys, err)
		}
		return
	}

	if init.Op == "sub" {
		if h.Verbose {
			log.Debugf("%v: %v: Successfully subscribed to %v", h.Name, resp.URL, init.Topic)
		}
		return
	}

	switch {
	case strings.EqualFold(init.Op, authOp):
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		var response WsAuthenticatedDataResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, "accounts"):
		var response WsAuthenticatedAccountsResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.Contains(init.Topic, "orders") &&
		strings.Contains(init.Topic, "update"):
		var response WsAuthenticatedOrdersUpdateResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.Contains(init.Topic, "orders"):
		var response WsAuthenticatedOrdersResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, wsAccountsList):
		var response WsAuthenticatedAccountsListResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, wsOrdersList):
		var response WsAuthenticatedOrdersListResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, wsOrdersDetail):
		var response WsAuthenticatedOrderDetailResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	}
}

func (h *HUOBI) wsHandleMarketData(resp WsMessage) {
	var init WsResponse
	err := common.JSONDecode(resp.Raw, &init)
	if err != nil {
		h.Websocket.DataHandler <- err
		return
	}
	if init.Status == "error" {
		h.Websocket.DataHandler <- fmt.Errorf("%v %v Websocket error %s %s",
			h.Name,
			resp.URL,
			init.ErrorCode,
			init.ErrorMessage)
		return
	}
	if init.Subscribed != "" {
		return
	}
	if init.Ping != 0 {
		err = h.WebsocketConn.WriteJSON(`{"pong":1337}`)
		if err != nil {
			log.Error(log.SubSystemExchSys, err)
		}
		return
	}

	switch {
	case strings.Contains(init.Channel, "depth"):
		var depth WsDepth
		err := common.JSONDecode(resp.Raw, &depth)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := strings.Split(depth.Channel, ".")
		h.WsProcessOrderbook(&depth, data[1])
	case strings.Contains(init.Channel, "kline"):
		var kline WsKline
		err := common.JSONDecode(resp.Raw, &kline)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := strings.Split(kline.Channel, ".")
		h.Websocket.DataHandler <- exchange.KlineData{
			Timestamp:  time.Unix(0, kline.Timestamp),
			Exchange:   h.GetName(),
			AssetType:  "SPOT",
			Pair:       currency.NewPairFromString(data[1]),
			OpenPrice:  kline.Tick.Open,
			ClosePrice: kline.Tick.Close,
			HighPrice:  kline.Tick.High,
			LowPrice:   kline.Tick.Low,
			Volume:     kline.Tick.Volume,
		}
	case strings.Contains(init.Channel, "trade"):
		var trade WsTrade
		err := common.JSONDecode(resp.Raw, &trade)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := strings.Split(trade.Channel, ".")
		h.Websocket.DataHandler <- exchange.TradeData{
			Exchange:     h.GetName(),
			AssetType:    "SPOT",
			CurrencyPair: currency.NewPairFromString(data[1]),
			Timestamp:    time.Unix(0, trade.Tick.Timestamp),
		}
	}
}

// WsProcessOrderbook processes new orderbook data
func (h *HUOBI) WsProcessOrderbook(ob *WsDepth, symbol string) error {
	var bids []orderbook.Item
	for _, data := range ob.Tick.Bids {
		bidLevel := data.([]interface{})
		bids = append(bids, orderbook.Item{Price: bidLevel[0].(float64),
			Amount: bidLevel[0].(float64)})
	}

	var asks []orderbook.Item
	for _, data := range ob.Tick.Asks {
		askLevel := data.([]interface{})
		asks = append(asks, orderbook.Item{Price: askLevel[0].(float64),
			Amount: askLevel[0].(float64)})
	}

	p := currency.NewPairFromString(symbol)

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = p

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, h.GetName(), false)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Exchange: h.GetName(),
		Asset:    asset.Spot,
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HUOBI) GenerateDefaultSubscriptions() {
	var channels = []string{wsMarketKline, wsMarketDepth, wsMarketTrade}
	var subscriptions []exchange.WebsocketChannelSubscription
	if h.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, "orders.%v", "orders.%v.update")
		subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
			Channel: "accounts",
		})
	}
	enabledCurrencies := h.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			channel := fmt.Sprintf(channels[i], enabledCurrencies[j].Lower().String())
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  channel,
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBI) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	if strings.Contains(channelToSubscribe.Channel, "orders.") ||
		strings.Contains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("sub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	subscription, err := common.JSONEncode(WsRequest{Subscribe: channelToSubscribe.Channel})
	if err != nil {
		return err
	}
	return h.wsSend(subscription)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	if strings.Contains(channelToSubscribe.Channel, "orders.") ||
		strings.Contains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("unsub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	subscription, err := common.JSONEncode(WsRequest{Unsubscribe: channelToSubscribe.Channel})
	if err != nil {
		return err
	}
	return h.wsSend(subscription)
}

// WsSend sends data to the websocket server
func (h *HUOBI) wsSend(data []byte) error {
	h.wsRequestMtx.Lock()
	defer h.wsRequestMtx.Unlock()
	if h.Verbose {
		log.Debugf(log.SubSystemExchSys, "%v sending message to websocket %s", h.Name, string(data))
	}
	return h.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

func (h *HUOBI) wsLogin() error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticationRequest{
		Op:               authOp,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
	}
	hmac := h.wsGenerateSignature(timestamp, wsAccountsOrdersEndPoint)
	request.Signature = crypto.Base64Encode(hmac)
	err := h.wsAuthenticatedSend(request)
	if err != nil {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

func (h *HUOBI) wsAuthenticatedSend(request interface{}) error {
	h.wsRequestMtx.Lock()
	defer h.wsRequestMtx.Unlock()
	encodedRequest, err := common.JSONEncode(request)
	if err != nil {
		return err
	}
	if h.Verbose {
		log.Debugf("%v sending Authenticated message to websocket %s", h.Name, string(encodedRequest))
	}
	return h.AuthenticatedWebsocketConn.WriteMessage(websocket.TextMessage, encodedRequest)
}

func (h *HUOBI) wsGenerateSignature(timestamp, endpoint string) []byte {
	values := url.Values{}
	values.Set("AccessKeyId", h.API.Credentials.Key)
	values.Set("SignatureMethod", signatureMethod)
	values.Set("SignatureVersion", signatureVersion)
	values.Set("Timestamp", timestamp)
	host := "api.huobi.pro"
	payload := fmt.Sprintf("%s\n%s\n%s\n%s",
		"GET", host, endpoint, values.Encode())
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(h.API.Credentials.Secret))
}

func (h *HUOBI) wsAuthenticatedSubscribe(operation, endpoint, topic string) error {
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedSubscriptionRequest{
		Op:               operation,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            topic,
	}
	hmac := h.wsGenerateSignature(timestamp, endpoint)
	request.Signature = crypto.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetAccountsList(pair currency.Pair) error {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authenticated cannot get accounts list", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedAccountsListRequest{
		Op:               requestOp,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsAccountsList,
		Symbol:           pair,
	}
	hmac := h.wsGenerateSignature(timestamp, wsAccountListEndpoint)
	request.Signature = crypto.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetOrdersList(accountID int64, pair currency.Pair) error {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authenticated cannot get orders list", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedOrdersListRequest{
		Op:               requestOp,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersList,
		AccountID:        accountID,
		Symbol:           pair.Lower(),
		States:           "submitted,partial-filled",
	}
	hmac := h.wsGenerateSignature(timestamp, wsOrdersListEndpoint)
	request.Signature = crypto.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetOrderDetails(orderID string) error {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authenticated cannot get order details", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedOrderDetailsRequest{
		Op:               requestOp,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersDetail,
		OrderID:          orderID,
	}
	hmac := h.wsGenerateSignature(timestamp, wsOrdersDetailEndpoint)
	request.Signature = crypto.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}
