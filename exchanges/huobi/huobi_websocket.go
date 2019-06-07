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
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	huobiSocketIOAddress       = "wss://api.huobi.pro/ws"
	huobiAccountsOrdersAddress = "wss://api.huobi.pro/ws/v1"
	wsMarketKline              = "market.%s.kline.1min"
	wsMarketDepth              = "market.%s.depth.step0"
	wsMarketTrade              = "market.%s.trade.detail"
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
	if h.AuthenticatedAPISupport {
		err = h.wsAuthenticatedDial(&dialer)
		if err != nil {
			return err
		}
		err = h.wsLogin()
		if err != nil {
			return err
		}
	}

	go h.WsHandleData()
	h.GenerateDefaultSubscriptions()

	return nil
}

func (h *HUOBI) wsDial(dialer *websocket.Dialer) error {
	var err error
	var conStatus *http.Response
	h.WebsocketConn, conStatus, err = dialer.Dial(huobiSocketIOAddress, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", huobiSocketIOAddress, conStatus, conStatus.StatusCode, err)
	}
	go h.wsMultiConnectionFunnel(h.WebsocketConn, huobiSocketIOAddress)
	return nil
}

func (h *HUOBI) wsAuthenticatedDial(dialer *websocket.Dialer) error {
	var err error
	var conStatus *http.Response
	h.AuthenticatedWebsocketConn, conStatus, err = dialer.Dial(huobiAccountsOrdersAddress, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", huobiAccountsOrdersAddress, conStatus, conStatus.StatusCode, err)
	}
	go h.wsMultiConnectionFunnel(h.AuthenticatedWebsocketConn, huobiAccountsOrdersAddress)
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
				log.Debugf("%v: %v: %v", h.Name, resp.URL, string(resp.Raw))
			}
			switch resp.URL {
			case huobiSocketIOAddress:
				h.wsHandleMarketData(resp)
			case huobiAccountsOrdersAddress:
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
			log.Error(err)
		}
		return
	}

	if init.Op == "sub" {
		if h.Verbose {
			log.Debugf("%v: %v: Successfully subscribed to %v", h.Name, resp.URL, init.Topic)
		}
	}

	switch {
	case strings.EqualFold(init.Op, "auth"):
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
	case common.StringContains(init.Topic, "orders") &&
		common.StringContains(init.Topic, "update"):
		var response WsAuthenticatedOrdersUpdateResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case common.StringContains(init.Topic, "orders"):
		var response WsAuthenticatedOrdersResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, "accounts.list"):
		var response WsAuthenticatedAccountsListResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, "orders.list"):
		var response WsAuthenticatedOrdersListResponse
		err := common.JSONDecode(resp.Raw, &response)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- response
	case strings.EqualFold(init.Topic, "orders.detail"):
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
			log.Error(err)
		}
		return
	}

	switch {
	case common.StringContains(init.Channel, "depth"):
		var depth WsDepth
		err := common.JSONDecode(resp.Raw, &depth)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := common.SplitStrings(depth.Channel, ".")
		h.WsProcessOrderbook(&depth, data[1])
	case common.StringContains(init.Channel, "kline"):
		var kline WsKline
		err := common.JSONDecode(resp.Raw, &kline)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := common.SplitStrings(kline.Channel, ".")
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
	case common.StringContains(init.Channel, "trade"):
		var trade WsTrade
		err := common.JSONDecode(resp.Raw, &trade)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		data := common.SplitStrings(trade.Channel, ".")
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
		Asset:    "SPOT",
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HUOBI) GenerateDefaultSubscriptions() {
	var channels = []string{wsMarketKline, wsMarketDepth, wsMarketTrade}
	var subscriptions []exchange.WebsocketChannelSubscription
	if h.AuthenticatedAPISupport {
		channels = append(channels, "orders.%v", "orders.%v.update")
		subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
			Channel: "accounts",
		})
	}
	enabledCurrencies := h.GetEnabledCurrencies()
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
	if common.StringContains(channelToSubscribe.Channel, "orders.") ||
		common.StringContains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("sub", "/ws/v1/"+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	subscription, err := common.JSONEncode(WsRequest{Subscribe: channelToSubscribe.Channel})
	if err != nil {
		return err
	}
	return h.wsSend(subscription)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	if common.StringContains(channelToSubscribe.Channel, "orders.") ||
		common.StringContains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("unsub", "/ws/v1/"+channelToSubscribe.Channel, channelToSubscribe.Channel)
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
		log.Debugf("%v sending message to websocket %s", h.Name, string(data))
	}
	return h.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}
func (h *HUOBI) wsLogin() error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	request := WsAuthenticationRequest{
		Op:               "auth",
		AccessKeyID:      h.APIKey,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        timestamp,
	}
	hmac := h.wsGenerateSignature(timestamp, "/ws/v1")
	request.Signature = common.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
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
	values.Set("AccessKeyId", h.APIKey)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", timestamp)
	host := "api.huobi.pro"
	payload := fmt.Sprintf("%s\n%s\n%s\n%s",
		"GET", host, endpoint, values.Encode())
	return common.GetHMAC(common.HashSHA256, []byte(payload), []byte(h.APISecret))
}

func (h *HUOBI) wsAuthenticatedSubscribe(operation, endpoint, topic string) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	request := WsAuthenticatedSubscriptionRequest{
		Op:               operation,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        timestamp,
		Topic:            topic,
	}
	hmac := h.wsGenerateSignature(timestamp, endpoint)
	request.Signature = common.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetAccountsList(pair currency.Pair) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	request := WsAuthenticatedAccountsListRequest{
		Op:               "req",
		AccessKeyID:      h.APIKey,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        timestamp,
		Topic:            "accounts.list",
		Symbol:           pair,
	}
	hmac := h.wsGenerateSignature(timestamp, "/ws/v1/accounts.list")
	request.Signature = common.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetOrdersList(accountID int64, pair currency.Pair) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	request := WsAuthenticatedOrdersListRequest{
		Op:               "req",
		AccessKeyID:      h.APIKey,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        timestamp,
		Topic:            "orders.list",
		AccountID:        accountID,
		Symbol:           pair.Lower(),
		States:           "submitted,partial-filled",
	}
	hmac := h.wsGenerateSignature(timestamp, "/ws/v1/orders.list")
	request.Signature = common.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}

func (h *HUOBI) wsGetOrderDetails(orderID string) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	request := WsAuthenticatedOrderDetailsRequest{
		Op:               "req",
		AccessKeyID:      h.APIKey,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        timestamp,
		Topic:            "orders.detail",
		OrderID:          orderID,
	}
	hmac := h.wsGenerateSignature(timestamp, "/ws/v1/orders.detail")
	request.Signature = common.Base64Encode(hmac)
	return h.wsAuthenticatedSend(request)
}
