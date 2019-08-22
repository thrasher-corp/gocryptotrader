package hitbtc

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
	rateLimit              = 20
)

var requestID nonce.Nonce

// WsConnect starts a new connection with the websocket API
func (h *HitBTC) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := h.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go h.WsHandleData()
	err = h.wsLogin()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", h.Name, err)
	}

	h.GenerateDefaultSubscriptions()

	return nil
}

// WsHandleData handles websocket data
func (h *HitBTC) WsHandleData() {
	h.Websocket.Wg.Add(1)

	defer func() {
		h.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-h.Websocket.ShutdownC:
			return

		default:
			resp, err := h.WebsocketConn.ReadMessage()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}
			h.Websocket.TrafficAlert <- struct{}{}

			var init capture
			err = common.JSONDecode(resp.Raw, &init)
			if err != nil {
				h.Websocket.DataHandler <- err
				continue
			}
			if init.Error.Code == 1002 {
				h.Websocket.SetCanUseAuthenticatedEndpoints(false)
			}
			if init.ID > 0 {
				h.WebsocketConn.AddResponseWithID(init.ID, resp.Raw)
				continue
			}
			if init.Error.Message != "" || init.Error.Code != 0 {
				h.Websocket.DataHandler <- fmt.Errorf("hitbtc.go error - Code: %d, Message: %s",
					init.Error.Code,
					init.Error.Message)
				continue
			}
			if _, ok := init.Result.(bool); ok {
				continue
			}
			if init.Method != "" {
				h.handleSubscriptionUpdates(resp, init)
			} else {
				h.handleCommandResponses(resp, init)
			}
		}
	}
}

func (h *HitBTC) handleSubscriptionUpdates(resp wshandler.WebsocketResponse, init capture) {
	switch init.Method {
	case "ticker":
		var ticker WsTicker
		err := common.JSONDecode(resp.Raw, &ticker)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		ts, err := time.Parse(time.RFC3339, ticker.Params.Timestamp)
		if err != nil {
			h.Websocket.DataHandler <- err
			return
		}
		h.Websocket.DataHandler <- wshandler.TickerData{
			Exchange:  h.GetName(),
			AssetType: asset.Spot,
			Pair:      currency.NewPairFromString(ticker.Params.Symbol),
			Quantity:  ticker.Params.Volume,
			Timestamp: ts,
			OpenPrice: ticker.Params.Open,
			HighPrice: ticker.Params.High,
			LowPrice:  ticker.Params.Low,
		}
	case "snapshotOrderbook":
		var obSnapshot WsOrderbook
		err := common.JSONDecode(resp.Raw, &obSnapshot)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		err = h.WsProcessOrderbookSnapshot(obSnapshot)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
	case "updateOrderbook":
		var obUpdate WsOrderbook
		err := common.JSONDecode(resp.Raw, &obUpdate)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.WsProcessOrderbookUpdate(obUpdate)
	case "snapshotTrades":
		var tradeSnapshot WsTrade
		err := common.JSONDecode(resp.Raw, &tradeSnapshot)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
	case "updateTrades":
		var tradeUpdates WsTrade
		err := common.JSONDecode(resp.Raw, &tradeUpdates)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
	case "activeOrders":
		var activeOrders WsActiveOrdersResponse
		err := common.JSONDecode(resp.Raw, &activeOrders)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- activeOrders
	case "report":
		var reportData WsReportResponse
		err := common.JSONDecode(resp.Raw, &reportData)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
		h.Websocket.DataHandler <- reportData
	}
}

func (h *HitBTC) handleCommandResponses(resp wshandler.WebsocketResponse, init capture) {
	switch resultType := init.Result.(type) {
	case map[string]interface{}:
		switch resultType["reportType"].(string) {
		case "new":
			var response WsSubmitOrderSuccessResponse
			err := common.JSONDecode(resp.Raw, &response)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
			h.Websocket.DataHandler <- response
		case "canceled":
			var response WsCancelOrderResponse
			err := common.JSONDecode(resp.Raw, &response)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
			h.Websocket.DataHandler <- response
		case "replaced":
			var response WsReplaceOrderResponse
			err := common.JSONDecode(resp.Raw, &response)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
			h.Websocket.DataHandler <- response
		}
	case []interface{}:
		if len(resultType) == 0 {
			h.Websocket.DataHandler <- fmt.Sprintf("No data returned. ID: %v", init.ID)
			return
		}
		data := resultType[0].(map[string]interface{})
		if _, ok := data["clientOrderId"]; ok {
			var response WsActiveOrdersResponse
			err := common.JSONDecode(resp.Raw, &response)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
			h.Websocket.DataHandler <- response
		} else if _, ok := data["available"]; ok {
			var response WsGetTradingBalanceResponse
			err := common.JSONDecode(resp.Raw, &response)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
			h.Websocket.DataHandler <- response
		}
	}
}

// WsProcessOrderbookSnapshot processes a full orderbook snapshot to a local cache
func (h *HitBTC) WsProcessOrderbookSnapshot(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 || len(ob.Params.Ask) == 0 {
		return errors.New("hitbtc.go error - no orderbooks to process")
	}

	var bids []orderbook.Item
	for i := range ob.Params.Bid {
		bids = append(bids, orderbook.Item{Amount: ob.Params.Bid[i].Size, Price: ob.Params.Bid[i].Price})
	}

	var asks []orderbook.Item
	for i := range ob.Params.Ask {
		asks = append(asks, orderbook.Item{Amount: ob.Params.Ask[i].Size, Price: ob.Params.Ask[i].Price})
	}

	p := currency.NewPairFromString(ob.Params.Symbol)

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.AssetType = asset.Spot
	newOrderBook.Pair = p

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, false)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    asset.Spot,
		Pair:     p,
	}

	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (h *HitBTC) WsProcessOrderbookUpdate(update WsOrderbook) error {
	if len(update.Params.Bid) == 0 && len(update.Params.Ask) == 0 {
		return errors.New("hitbtc_websocket.go error - no data")
	}

	var bids, asks []orderbook.Item
	for i := range update.Params.Bid {
		bids = append(bids, orderbook.Item{Price: update.Params.Bid[i].Price, Amount: update.Params.Bid[i].Size})
	}

	for i := range update.Params.Ask {
		asks = append(asks, orderbook.Item{Price: update.Params.Ask[i].Price, Amount: update.Params.Ask[i].Size})
	}

	p := currency.NewPairFromString(update.Params.Symbol)
	err := h.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
		Asks:         asks,
		Bids:         bids,
		CurrencyPair: p,
		UpdateID:     update.Params.Sequence,
		AssetType:    asset.Spot,
	})
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    asset.Spot,
		Pair:     p,
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HitBTC) GenerateDefaultSubscriptions() {
	var channels = []string{"subscribeTicker", "subscribeOrderbook", "subscribeTrades", "subscribeCandles"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	if h.Websocket.CanUseAuthenticatedEndpoints() {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: "subscribeReports",
		})
	}
	enabledCurrencies := h.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HitBTC) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := WsNotification{
		Method: channelToSubscribe.Channel,
	}
	if channelToSubscribe.Currency.String() != "" {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
		}
	}
	if strings.EqualFold(channelToSubscribe.Channel, "subscribeTrades") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Limit:  100,
		}
	} else if strings.EqualFold(channelToSubscribe.Channel, "subscribeCandles") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Period: "M30",
			Limit:  100,
		}
	}

	return h.WebsocketConn.SendMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HitBTC) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	unsubscribeChannel := strings.Replace(channelToSubscribe.Channel, "subscribe", "unsubscribe", 1)
	subscribe := WsNotification{
		JSONRPCVersion: rpcVersion,
		Method:         unsubscribeChannel,
		Params: params{
			Symbol: channelToSubscribe.Currency.String(),
		},
	}
	if strings.EqualFold(unsubscribeChannel, "unsubscribeTrades") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Limit:  100,
		}
	} else if strings.EqualFold(unsubscribeChannel, "unsubscribeCandles") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Period: "M30",
			Limit:  100,
		}
	}

	return h.WebsocketConn.SendMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HitBTC) wsLogin() error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	nonce := fmt.Sprintf("%v", time.Now().Unix())
	hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(nonce), []byte(h.API.Credentials.Secret))
	request := WsLoginRequest{
		Method: "login",
		Params: WsLoginData{
			Algo:      "HS256",
			PKey:      h.API.Credentials.Key,
			Nonce:     nonce,
			Signature: crypto.HexEncodeToString(hmac),
		},
	}

	err := h.WebsocketConn.SendMessage(request)
	if err != nil {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// wsPlaceOrder sends a websocket message to submit an order
func (h *HitBTC) wsPlaceOrder(pair currency.Pair, side string, price, quantity float64) (*WsSubmitOrderSuccessResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	id := h.WebsocketConn.GenerateMessageID(false)
	request := WsSubmitOrderRequest{
		Method: "newOrder",
		Params: WsSubmitOrderRequestData{
			ClientOrderID: id,
			Symbol:        pair,
			Side:          strings.ToLower(side),
			Price:         price,
			Quantity:      quantity,
		},
		ID: id,
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsSubmitOrderSuccessResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsCancelOrder sends a websocket message to cancel an order
func (h *HitBTC) wsCancelOrder(clientOrderID string) (*WsCancelOrderResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsCancelOrderRequest{
		Method: "cancelOrder",
		Params: WsCancelOrderRequestData{
			ClientOrderID: clientOrderID,
		},
		ID: h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsCancelOrderResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsReplaceOrder sends a websocket message to replace an order
func (h *HitBTC) wsReplaceOrder(clientOrderID string, quantity, price float64) (*WsReplaceOrderResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "cancelReplaceOrder",
		Params: WsReplaceOrderRequestData{
			ClientOrderID:   clientOrderID,
			RequestClientID: fmt.Sprintf("%v", time.Now().Unix()),
			Quantity:        quantity,
			Price:           price,
		},
		ID: h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsReplaceOrderResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetActiveOrders sends a websocket message to get all active orders
func (h *HitBTC) wsGetActiveOrders() (*WsActiveOrdersResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "getOrders",
		Params: WsReplaceOrderRequestData{},
		ID:     h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsActiveOrdersResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetTradingBalance sends a websocket message to get trading balance
func (h *HitBTC) wsGetTradingBalance() (*WsGetTradingBalanceResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "getTradingBalance",
		Params: WsReplaceOrderRequestData{},
		ID:     h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetTradingBalanceResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetCurrencies sends a websocket message to get trading balance
func (h *HitBTC) wsGetCurrencies(currencyItem currency.Code) (*WsGetCurrenciesResponse, error) {
	request := WsGetCurrenciesRequest{
		Method: "getCurrency",
		Params: WsGetCurrenciesRequestParameters{
			Currency: currencyItem,
		},
		ID: h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetCurrenciesResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (h *HitBTC) wsGetSymbols(currencyItem currency.Pair) (*WsGetSymbolsResponse, error) {
	request := WsGetSymbolsRequest{
		Method: "getSymbol",
		Params: WsGetSymbolsRequestParameters{
			Symbol: currencyItem,
		},
		ID: h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetSymbolsResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (h *HitBTC) wsGetTrades(currencyItem currency.Pair, limit int64, sort, by string) (*WsGetTradesResponse, error) {
	request := WsGetTradesRequest{
		Method: "getTrades",
		Params: WsGetTradesRequestParameters{
			Symbol: currencyItem,
			Limit:  limit,
			Sort:   sort,
			By:     by,
		},
		ID: h.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := h.WebsocketConn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetTradesResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}
