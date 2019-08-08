package huobi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/connection"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/monitor"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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

	loginDelay = 50 * time.Millisecond
	rateLimit  = 20
)

// Instantiates a communications channel between websocket connections
var comms = make(chan WsMessage, 1)

// WsConnect initiates a new websocket connection
func (h *HUOBI) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(monitor.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
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
	err := h.WebsocketConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	go h.wsMultiConnectionFunnel(h.WebsocketConn, wsMarketURL)
	return nil
}

func (h *HUOBI) wsAuthenticatedDial(dialer *websocket.Dialer) error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	err := h.AuthenticatedWebsocketConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	go h.wsMultiConnectionFunnel(h.AuthenticatedWebsocketConn, wsAccountsOrdersURL)
	return nil
}

// wsMultiConnectionFunnel manages data from multiple endpoints and passes it to a channel
func (h *HUOBI) wsMultiConnectionFunnel(ws *connection.WebsocketConnection, url string) {
	h.Websocket.Wg.Add(1)
	defer h.Websocket.Wg.Done()
	for {
		select {
		case <-h.Websocket.ShutdownC:
			return
		default:
			resp, err := ws.ReadMessage()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			h.Websocket.TrafficAlert <- struct{}{}
			comms <- WsMessage{Raw: resp.Raw, URL: url}
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
	if init.Ping != 0 {
		err = h.WebsocketConn.SendMessage(WsPong{Pong: init.Ping})
		if err != nil {
			log.Error(err)
		}
		return
	}
	if init.ErrorMessage == "api-signature-not-valid" {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if init.Op == "sub" {
		if h.Verbose {
			log.Debugf("%v: %v: Successfully subscribed to %v", h.Name, resp.URL, init.Topic)
		}
		return
	}
	if init.ClientID > 0 {
		h.AuthenticatedWebsocketConn.AddResponseWithID(init.ClientID, resp.Raw)
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
		err = h.WebsocketConn.SendMessage(WsPong{Pong: init.Ping})
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
		h.Websocket.DataHandler <- monitor.KlineData{
			Timestamp:  time.Unix(0, kline.Timestamp),
			Exchange:   h.GetName(),
			AssetType:  orderbook.Spot,
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
		h.Websocket.DataHandler <- monitor.TradeData{
			Exchange:     h.GetName(),
			AssetType:    orderbook.Spot,
			CurrencyPair: currency.NewPairFromString(data[1]),
			Timestamp:    time.Unix(0, trade.Tick.Timestamp),
		}
	}
}

// WsProcessOrderbook processes new orderbook data
func (h *HUOBI) WsProcessOrderbook(update *WsDepth, symbol string) error {
	p := currency.NewPairFromString(symbol)
	var bids, asks []orderbook.Item
	for i := 0; i < len(update.Tick.Bids); i++ {
		bidLevel := update.Tick.Bids[i].([]interface{})
		bids = append(bids, orderbook.Item{Price: bidLevel[0].(float64),
			Amount: bidLevel[0].(float64)})
	}
	for i := 0; i < len(update.Tick.Asks); i++ {
		askLevel := update.Tick.Asks[i].([]interface{})
		asks = append(asks, orderbook.Item{Price: askLevel[0].(float64),
			Amount: askLevel[0].(float64)})
	}
	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = p
	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, h.GetName(), true)
	if err != nil {
		return err
	}
	h.Websocket.DataHandler <- monitor.WebsocketOrderbookUpdate{
		Pair:     p,
		Exchange: h.GetName(),
		Asset:    orderbook.Spot,
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HUOBI) GenerateDefaultSubscriptions() {
	var channels = []string{wsMarketKline, wsMarketDepth, wsMarketTrade}
	var subscriptions []monitor.WebsocketChannelSubscription
	if h.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, "orders.%v", "orders.%v.update")
		subscriptions = append(subscriptions, monitor.WebsocketChannelSubscription{
			Channel: "accounts",
		})
	}
	enabledCurrencies := h.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			channel := fmt.Sprintf(channels[i], enabledCurrencies[j].Lower().String())
			subscriptions = append(subscriptions, monitor.WebsocketChannelSubscription{
				Channel:  channel,
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBI) Subscribe(channelToSubscribe monitor.WebsocketChannelSubscription) error {
	if common.StringContains(channelToSubscribe.Channel, "orders.") ||
		common.StringContains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("sub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	return h.WebsocketConn.SendMessage(WsRequest{Subscribe: channelToSubscribe.Channel})
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(channelToSubscribe monitor.WebsocketChannelSubscription) error {
	if common.StringContains(channelToSubscribe.Channel, "orders.") ||
		common.StringContains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("unsub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	return h.WebsocketConn.SendMessage(WsRequest{Unsubscribe: channelToSubscribe.Channel})
}

func (h *HUOBI) wsGenerateSignature(timestamp, endpoint string) []byte {
	values := url.Values{}
	values.Set("AccessKeyId", h.APIKey)
	values.Set("SignatureMethod", signatureMethod)
	values.Set("SignatureVersion", signatureVersion)
	values.Set("Timestamp", timestamp)
	host := "api.huobi.pro"
	payload := fmt.Sprintf("%s\n%s\n%s\n%s",
		"GET", host, endpoint, values.Encode())
	return common.GetHMAC(common.HashSHA256, []byte(payload), []byte(h.APISecret))
}

func (h *HUOBI) wsLogin() error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticationRequest{
		Op:               authOp,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
	}
	hmac := h.wsGenerateSignature(timestamp, wsAccountsOrdersEndPoint)
	request.Signature = common.Base64Encode(hmac)
	err := h.AuthenticatedWebsocketConn.SendMessage(request)
	if err != nil {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}

	time.Sleep(loginDelay)
	return nil
}

func (h *HUOBI) wsAuthenticatedSubscribe(operation, endpoint, topic string) error {
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedSubscriptionRequest{
		Op:               operation,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            topic,
	}
	hmac := h.wsGenerateSignature(timestamp, endpoint)
	request.Signature = common.Base64Encode(hmac)
	return h.AuthenticatedWebsocketConn.SendMessage(request)
}

func (h *HUOBI) wsGetAccountsList(pair currency.Pair) (*WsAuthenticatedAccountsListResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get accounts list", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedAccountsListRequest{
		Op:               requestOp,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsAccountsList,
		Symbol:           pair,
	}
	hmac := h.wsGenerateSignature(timestamp, wsAccountListEndpoint)
	request.Signature = common.Base64Encode(hmac)
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedAccountsListResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (h *HUOBI) wsGetOrdersList(accountID int64, pair currency.Pair) (*WsAuthenticatedOrdersResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get orders list", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedOrdersListRequest{
		Op:               requestOp,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersList,
		AccountID:        accountID,
		Symbol:           pair.Lower(),
		States:           "submitted,partial-filled",
	}
	hmac := h.wsGenerateSignature(timestamp, wsOrdersListEndpoint)
	request.Signature = common.Base64Encode(hmac)
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedOrdersResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (h *HUOBI) wsGetOrderDetails(orderID string) (*WsAuthenticatedOrderDetailResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get order details", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedOrderDetailsRequest{
		Op:               requestOp,
		AccessKeyID:      h.APIKey,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersDetail,
		OrderID:          orderID,
	}
	hmac := h.wsGenerateSignature(timestamp, wsOrdersDetailEndpoint)
	request.Signature = common.Base64Encode(hmac)
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedOrderDetailResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}
