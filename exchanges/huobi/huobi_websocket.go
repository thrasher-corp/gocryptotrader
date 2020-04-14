package huobi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	baseWSURL = "wss://api.huobi.pro"

	wsMarketURL    = baseWSURL + "/ws"
	wsMarketKline  = "market.%s.kline.1min"
	wsMarketDepth  = "market.%s.depth.step0"
	wsMarketTrade  = "market.%s.trade.detail"
	wsMarketTicker = "market.%s.detail"

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
var comms = make(chan WsMessage)

// WsConnect initiates a new websocket connection
func (h *HUOBI) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := h.wsDial(&dialer)
	if err != nil {
		return err
	}
	err = h.wsAuthenticatedDial(&dialer)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authenticated dial failed: %v\n", h.Name, err)
	}
	err = h.wsLogin()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", h.Name, err)
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}

	go h.wsReadData()
	h.GenerateDefaultSubscriptions()

	return nil
}

func (h *HUOBI) wsDial(dialer *websocket.Dialer) error {
	err := h.WebsocketConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	go h.wsFunnelConnectionData(h.WebsocketConn, wsMarketURL)
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
	go h.wsFunnelConnectionData(h.AuthenticatedWebsocketConn, wsAccountsOrdersURL)
	return nil
}

// wsFunnelConnectionData manages data from multiple endpoints and passes it to a channel
func (h *HUOBI) wsFunnelConnectionData(ws *wshandler.WebsocketConnection, url string) {
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

// wsReadData receives and passes on websocket messages for processing
func (h *HUOBI) wsReadData() {
	h.Websocket.Wg.Add(1)
	defer h.Websocket.Wg.Done()
	for {
		select {
		case <-h.Websocket.ShutdownC:
			return
		case resp := <-comms:
			err := h.wsHandleData(resp.Raw)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
		}
	}
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "submitted":
		return order.New, nil
	case "canceled":
		return order.Cancelled, nil
	case "partial-filled":
		return order.PartiallyFilled, nil
	case "partial-canceled":
		return order.PartiallyCancelled, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func stringToOrderSide(side string) (order.Side, error) {
	switch {
	case strings.Contains(side, "buy"):
		return order.Buy, nil
	case strings.Contains(side, "sell"):
		return order.Sell, nil
	}

	return order.UnknownSide, errors.New(side + " not recognised as order side")
}

func stringToOrderType(oType string) (order.Type, error) {
	switch {
	case strings.Contains(oType, "limit"):
		return order.Limit, nil
	case strings.Contains(oType, "market"):
		return order.Market, nil
	}

	return order.UnknownType, errors.New(oType + " not recognised as order type")
}

func (h *HUOBI) wsHandleData(respRaw []byte) error {
	var init WsResponse
	err := json.Unmarshal(respRaw, &init)
	if err != nil {
		return err
	}
	if init.Subscribed != "" ||
		init.UnSubscribed != "" ||
		init.Op == "sub" ||
		init.Op == "unsub" {
		// TODO handle subs
		return nil
	}
	if init.Ping != 0 {
		h.sendPingResponse(init.Ping)
		return nil
	}
	if init.ErrorMessage == "api-signature-not-valid" {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return errors.New(h.Name + " - invalid credentials. Authenticated requests disabled")
	}
	if init.ClientID > 0 {
		if h.WebsocketConn.IsIDWaitingForResponse(init.ClientID) {
			h.WebsocketConn.SetResponseIDAndData(init.ClientID, respRaw)
			return nil
		}
	}

	switch {
	case strings.EqualFold(init.Op, authOp):
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		err := json.Unmarshal(respRaw, &init)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- init

	case strings.EqualFold(init.Topic, "accounts"):
		var response WsAuthenticatedAccountsResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- response

	case strings.Contains(init.Topic, "orders") &&
		strings.Contains(init.Topic, "update"):
		var response WsAuthenticatedOrdersUpdateResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		data := strings.Split(response.Topic, ".")
		if len(data) < 2 {
			return errors.New(h.Name + " - currency could not be extracted from response")
		}
		orderID := strconv.FormatInt(response.Data.OrderID, 10)
		var oSide order.Side
		oSide, err = stringToOrderSide(response.Data.OrderType)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var oType order.Type
		oType, err = stringToOrderType(response.Data.OrderType)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var oStatus order.Status
		oStatus, err = stringToOrderStatus(response.Data.OrderState)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(data[1])
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- &order.Detail{
			Price:           response.Data.Price,
			Amount:          response.Data.UnfilledAmount + response.Data.FilledAmount,
			ExecutedAmount:  response.Data.FilledAmount,
			RemainingAmount: response.Data.UnfilledAmount,
			Exchange:        h.Name,
			ID:              orderID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       a,
			LastUpdated:     time.Unix(response.TS*1000, 0),
			Pair:            p,
		}

	case strings.Contains(init.Topic, "orders"):
		var response WsOldOrderUpdate
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- response
	case strings.Contains(init.Channel, "depth"):
		var depth WsDepth
		err := json.Unmarshal(respRaw, &depth)
		if err != nil {
			return err
		}

		data := strings.Split(depth.Channel, ".")
		err = h.WsProcessOrderbook(&depth, data[1])
		if err != nil {
			return err
		}
	case strings.Contains(init.Rep, "kline"):
		var kline wsKLineResponseThing
		err := json.Unmarshal(respRaw, &kline)
		if err != nil {
			return err
		}
		var curr = strings.Split(init.Rep, ".")
		for i := range kline.Data {
			h.Websocket.DataHandler <- wshandler.KlineData{
				Timestamp: time.Now(),
				Exchange:  h.Name,
				AssetType: asset.Spot,
				Pair: currency.NewPairFromFormattedPairs(curr[1],
					h.GetEnabledPairs(asset.Spot), h.GetPairFormat(asset.Spot, true)),
				OpenPrice:  kline.Data[i].Open,
				ClosePrice: kline.Data[i].Close,
				HighPrice:  kline.Data[i].High,
				LowPrice:   kline.Data[i].Low,
				Volume:     kline.Data[i].Volume,
			}
		}
	case strings.Contains(init.Channel, "kline"):
		var kline WsKline
		err := json.Unmarshal(respRaw, &kline)
		if err != nil {
			return err
		}
		data := strings.Split(kline.Channel, ".")
		h.Websocket.DataHandler <- wshandler.KlineData{
			Timestamp: time.Unix(0, kline.Timestamp*int64(time.Millisecond)),
			Exchange:  h.Name,
			AssetType: asset.Spot,
			Pair: currency.NewPairFromFormattedPairs(data[1],
				h.GetEnabledPairs(asset.Spot), h.GetPairFormat(asset.Spot, true)),
			OpenPrice:  kline.Tick.Open,
			ClosePrice: kline.Tick.Close,
			HighPrice:  kline.Tick.High,
			LowPrice:   kline.Tick.Low,
			Volume:     kline.Tick.Volume,
		}
	case strings.Contains(init.Channel, "trade.detail"):
		var trade WsTrade
		err := json.Unmarshal(respRaw, &trade)
		if err != nil {
			return err
		}
		data := strings.Split(trade.Channel, ".")
		h.Websocket.DataHandler <- wshandler.TradeData{
			Exchange:  h.Name,
			AssetType: asset.Spot,
			CurrencyPair: currency.NewPairFromFormattedPairs(data[1],
				h.GetEnabledPairs(asset.Spot), h.GetPairFormat(asset.Spot, true)),
			Timestamp: time.Unix(0, trade.Tick.Timestamp*int64(time.Millisecond)),
		}
	case strings.Contains(init.Channel, "detail"),
		strings.Contains(init.Rep, "detail"):
		var wsTicker WsTick
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		var data []string
		if wsTicker.Channel != "" {
			data = strings.Split(wsTicker.Channel, ".")
		}
		if wsTicker.Rep != "" {
			data = strings.Split(wsTicker.Rep, ".")
		}
		h.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: h.Name,
			Open:         wsTicker.Tick.Open,
			Close:        wsTicker.Tick.Close,
			Volume:       wsTicker.Tick.Amount,
			QuoteVolume:  wsTicker.Tick.Volume,
			High:         wsTicker.Tick.High,
			Low:          wsTicker.Tick.Low,
			LastUpdated:  time.Unix(0, wsTicker.Timestamp*int64(time.Millisecond)),
			AssetType:    asset.Spot,
			Pair: currency.NewPairFromFormattedPairs(data[1],
				h.GetEnabledPairs(asset.Spot), h.GetPairFormat(asset.Spot, true)),
		}
	default:
		h.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: h.Name + wshandler.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

func (h *HUOBI) sendPingResponse(pong int64) {
	err := h.WebsocketConn.SendJSONMessage(WsPong{Pong: pong})
	if err != nil {
		log.Error(log.ExchangeSys, err)
	}
}

// WsProcessOrderbook processes new orderbook data
func (h *HUOBI) WsProcessOrderbook(update *WsDepth, symbol string) error {
	p := currency.NewPairFromFormattedPairs(symbol,
		h.GetEnabledPairs(asset.Spot),
		h.GetPairFormat(asset.Spot, true))

	var bids, asks []orderbook.Item
	for i := range update.Tick.Bids {
		bids = append(bids, orderbook.Item{
			Price:  update.Tick.Bids[i][0].(float64),
			Amount: update.Tick.Bids[i][1].(float64),
		})
	}

	for i := range update.Tick.Asks {
		asks = append(asks, orderbook.Item{
			Price:  update.Tick.Asks[i][0].(float64),
			Amount: update.Tick.Asks[i][1].(float64),
		})
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = p
	newOrderBook.AssetType = asset.Spot
	newOrderBook.ExchangeName = h.Name

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Pair:     p,
		Exchange: h.Name,
		Asset:    asset.Spot,
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HUOBI) GenerateDefaultSubscriptions() {
	var channels = []string{wsMarketKline, wsMarketDepth, wsMarketTrade, wsMarketTicker}
	var subscriptions []wshandler.WebsocketChannelSubscription
	if h.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, "orders.%v", "orders.%v.update")
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: "accounts",
		})
	}
	enabledCurrencies := h.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			channel := fmt.Sprintf(channels[i], enabledCurrencies[j].Lower().String())
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channel,
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBI) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	if strings.Contains(channelToSubscribe.Channel, "orders.") ||
		strings.Contains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("sub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	return h.WebsocketConn.SendJSONMessage(WsRequest{Subscribe: channelToSubscribe.Channel})
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	if strings.Contains(channelToSubscribe.Channel, "orders.") ||
		strings.Contains(channelToSubscribe.Channel, "accounts") {
		return h.wsAuthenticatedSubscribe("unsub", wsAccountsOrdersEndPoint+channelToSubscribe.Channel, channelToSubscribe.Channel)
	}
	return h.WebsocketConn.SendJSONMessage(WsRequest{Unsubscribe: channelToSubscribe.Channel})
}

func (h *HUOBI) wsGenerateSignature(timestamp, endpoint string) []byte {
	values := url.Values{}
	values.Set("AccessKeyId", h.API.Credentials.Key)
	values.Set("SignatureMethod", signatureMethod)
	values.Set("SignatureVersion", signatureVersion)
	values.Set("Timestamp", timestamp)
	host := "api.huobi.pro"
	payload := fmt.Sprintf("%s\n%s\n%s\n%s",
		http.MethodGet, host, endpoint, values.Encode())
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(h.API.Credentials.Secret))
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
	err := h.AuthenticatedWebsocketConn.SendJSONMessage(request)
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
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            topic,
	}
	hmac := h.wsGenerateSignature(timestamp, endpoint)
	request.Signature = crypto.Base64Encode(hmac)
	return h.AuthenticatedWebsocketConn.SendJSONMessage(request)
}

func (h *HUOBI) wsGetAccountsList() (*WsAuthenticatedAccountsListResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get accounts list", h.Name)
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	request := WsAuthenticatedAccountsListRequest{
		Op:               requestOp,
		AccessKeyID:      h.API.Credentials.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsAccountsList,
	}
	hmac := h.wsGenerateSignature(timestamp, wsAccountListEndpoint)
	request.Signature = crypto.Base64Encode(hmac)
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedAccountsListResponse
	err = json.Unmarshal(resp, &response)
	return &response, err
}

func (h *HUOBI) wsGetOrdersList(accountID int64, pair currency.Pair) (*WsAuthenticatedOrdersResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get orders list", h.Name)
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
		Symbol:           h.FormatExchangeCurrency(pair, asset.Spot).String(),
		States:           "submitted,partial-filled",
	}
	hmac := h.wsGenerateSignature(timestamp, wsOrdersListEndpoint)
	request.Signature = crypto.Base64Encode(hmac)
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedOrdersResponse
	err = json.Unmarshal(resp, &response)
	return &response, err
}

func (h *HUOBI) wsGetOrderDetails(orderID string) (*WsAuthenticatedOrderDetailResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get order details", h.Name)
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
	request.ClientID = h.AuthenticatedWebsocketConn.GenerateMessageID(true)
	resp, err := h.AuthenticatedWebsocketConn.SendMessageReturnResponse(request.ClientID, request)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedOrderDetailResponse
	err = json.Unmarshal(resp, &response)
	return &response, err
}
