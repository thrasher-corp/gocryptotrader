package coinut

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const coinutWebsocketURL = "wss://wsapi.coinut.com"
const coinutWebsocketRateLimit = 30

var nNonce map[int64]string
var channels map[string]chan []byte
var instrumentListByString map[string]int64
var instrumentListByCode map[int64]string
var populatedList bool

// NOTE for speed considerations
// wss://wsapi-as.coinut.com
// wss://wsapi-na.coinut.com
// wss://wsapi-eu.coinut.com

// WsConnect intiates a websocket connection
func (c *COINUT) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go c.WsHandleData()

	if !populatedList {
		instrumentListByString = make(map[string]int64)
		instrumentListByCode = make(map[int64]string)
		err = c.WsSetInstrumentList()
		if err != nil {
			return err
		}
		populatedList = true
	}
	c.wsAuthenticate()
	c.GenerateDefaultSubscriptions()

	// define bi-directional communication
	channels = make(map[string]chan []byte)
	channels["hb"] = make(chan []byte, 1)

	return nil
}

// WsHandleData handles read data
func (c *COINUT) WsHandleData() {
	c.Websocket.Wg.Add(1)

	defer func() {
		c.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		default:
			resp, err := c.WebsocketConn.ReadMessage()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}
			c.Websocket.TrafficAlert <- struct{}{}

			if strings.HasPrefix(string(resp.Raw), "[") {
				var incoming []wsResponse
				err = common.JSONDecode(resp.Raw, &incoming)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				for i := range incoming {
					if incoming[i].Nonce > 0 {
						c.WebsocketConn.AddResponseWithID(incoming[i].Nonce, resp.Raw)
						break
					}
					var individualJSON []byte
					individualJSON, err = common.JSONEncode(incoming[i])
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.wsProcessResponse(individualJSON)
				}

			} else {
				var incoming wsResponse
				err = common.JSONDecode(resp.Raw, &incoming)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				c.wsProcessResponse(resp.Raw)
			}

		}
	}
}

func (c *COINUT) wsProcessResponse(resp []byte) {
	var incoming wsResponse
	err := common.JSONDecode(resp, &incoming)
	if err != nil {
		c.Websocket.DataHandler <- err
		return
	}
	switch incoming.Reply {
	case "hb":
		channels["hb"] <- resp
	case "inst_tick":
		var ticker WsTicker
		err := common.JSONDecode(resp, &ticker)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := instrumentListByCode[ticker.InstID]
		c.Websocket.DataHandler <- wshandler.TickerData{
			Timestamp: time.Unix(0, ticker.Timestamp),
			Pair:      currency.NewPairFromString(currencyPair),
			Exchange:  c.GetName(),
			AssetType: asset.Spot,
			High:      ticker.HighestBuy,
			Low:       ticker.LowestSell,
			Close:     ticker.Last,
			Volume:    ticker.Volume,
		}

	case "inst_order_book":
		var orderbooksnapshot WsOrderbookSnapshot
		err := common.JSONDecode(resp, &orderbooksnapshot)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		err = c.WsProcessOrderbookSnapshot(&orderbooksnapshot)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := instrumentListByCode[orderbooksnapshot.InstID]
		c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Exchange: c.GetName(),
			Asset:    asset.Spot,
			Pair:     currency.NewPairFromString(currencyPair),
		}
	case "inst_order_book_update":
		var orderbookUpdate WsOrderbookUpdate
		err := common.JSONDecode(resp, &orderbookUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		err = c.WsProcessOrderbookUpdate(&orderbookUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := instrumentListByCode[orderbookUpdate.InstID]
		c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Exchange: c.GetName(),
			Asset:    asset.Spot,
			Pair:     currency.NewPairFromString(currencyPair),
		}
	case "inst_trade":
		var tradeSnap WsTradeSnapshot
		err := common.JSONDecode(resp, &tradeSnap)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}

	case "inst_trade_update":
		var tradeUpdate WsTradeUpdate
		err := common.JSONDecode(resp, &tradeUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := instrumentListByCode[tradeUpdate.InstID]
		c.Websocket.DataHandler <- wshandler.TradeData{
			Timestamp:    time.Unix(tradeUpdate.Timestamp, 0),
			CurrencyPair: currency.NewPairFromString(currencyPair),
			AssetType:    asset.Spot,
			Exchange:     c.GetName(),
			Price:        tradeUpdate.Price,
			Side:         tradeUpdate.Side,
		}
	default:
		if incoming.Nonce > 0 {
			c.WebsocketConn.AddResponseWithID(incoming.Nonce, resp)
			return
		}
		c.Websocket.DataHandler <- fmt.Errorf("%v unhandled websocket response: %s", c.Name, resp)
	}
}

// GetNonce returns a nonce for a required request
func (c *COINUT) GetNonce() int64 {
	if c.Nonce.Get() == 0 {
		c.Nonce.Set(time.Now().Unix())
	} else {
		c.Nonce.Inc()
	}

	return int64(c.Nonce.Get())
}

// WsSetInstrumentList fetches instrument list and propagates a local cache
func (c *COINUT) WsSetInstrumentList() error {
	request := wsRequest{
		Request: "inst_list",
		SecType: strings.ToUpper(asset.Spot.String()),
		Nonce:   c.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(request.Nonce, request)
	if err != nil {
		return err
	}
	var list WsInstrumentList
	err = common.JSONDecode(resp, &list)
	if err != nil {
		return err
	}
	for currency, data := range list.Spot {
		instrumentListByString[currency] = data[0].InstID
		instrumentListByCode[data[0].InstID] = currency
	}
	if len(instrumentListByString) == 0 || len(instrumentListByCode) == 0 {
		return errors.New("instrument lists failed to populate")
	}
	return nil
}

// WsProcessOrderbookSnapshot processes the orderbook snapshot
func (c *COINUT) WsProcessOrderbookSnapshot(ob *WsOrderbookSnapshot) error {
	var bids []orderbook.Item
	for i := range ob.Buy {
		bids = append(bids, orderbook.Item{
			Amount: ob.Buy[i].Volume,
			Price:  ob.Buy[i].Price,
		})
	}

	var asks []orderbook.Item
	for i := range ob.Sell {
		asks = append(asks, orderbook.Item{
			Amount: ob.Sell[i].Volume,
			Price:  ob.Sell[i].Price,
		})
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = currency.NewPairFromString(instrumentListByCode[ob.InstID])
	newOrderBook.AssetType = asset.Spot

	return c.Websocket.Orderbook.LoadSnapshot(&newOrderBook, false)
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(update *WsOrderbookUpdate) error {
	p := currency.NewPairFromString(instrumentListByCode[update.InstID])
	bufferUpdate := &wsorderbook.WebsocketOrderbookUpdate{
		CurrencyPair: p,
		UpdateID:     update.TransID,
		AssetType:    asset.Spot,
	}
	if strings.EqualFold(update.Side, exchange.BuyOrderSide.ToLower().ToString()) {
		bufferUpdate.Bids = []orderbook.Item{{Price: update.Price, Amount: update.Volume}}
	} else {
		bufferUpdate.Asks = []orderbook.Item{{Price: update.Price, Amount: update.Volume}}
	}
	return c.Websocket.Orderbook.Update(bufferUpdate)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *COINUT) GenerateDefaultSubscriptions() {
	var channels = []string{"inst_tick", "inst_order_book"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	enabledCurrencies := c.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	c.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (c *COINUT) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := wsRequest{
		Request:   channelToSubscribe.Channel,
		InstID:    instrumentListByString[channelToSubscribe.Currency.String()],
		Subscribe: true,
		Nonce:     c.WebsocketConn.GenerateMessageID(false),
	}
	return c.WebsocketConn.SendMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *COINUT) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := wsRequest{
		Request:   channelToSubscribe.Channel,
		InstID:    instrumentListByString[channelToSubscribe.Currency.String()],
		Subscribe: false,
		Nonce:     c.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(subscribe.Nonce, subscribe)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response["status"].([]interface{})[0] != "OK" {
		return fmt.Errorf("%v unsubscribe failed for channel %v", c.Name, channelToSubscribe.Channel)
	}
	return nil
}

func (c *COINUT) wsAuthenticate() error {
	if !c.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", c.Name)
	}
	timestamp := time.Now().Unix()
	nonce := c.WebsocketConn.GenerateMessageID(false)
	payload := fmt.Sprintf("%v|%v|%v", c.API.Credentials.ClientID, timestamp, nonce)
	hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(c.API.Credentials.Key))
	loginRequest := struct {
		Request   string `json:"request"`
		Username  string `json:"username"`
		Nonce     int64  `json:"nonce"`
		Hmac      string `json:"hmac_sha256"`
		Timestamp int64  `json:"timestamp"`
	}{
		Request:   "login",
		Username:  c.API.Credentials.ClientID,
		Nonce:     nonce,
		Hmac:      crypto.HexEncodeToString(hmac),
		Timestamp: timestamp,
	}

	resp, err := c.WebsocketConn.SendMessageReturnResponse(loginRequest.Nonce, loginRequest)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response["status"].([]interface{})[0] != "OK" {
		c.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return fmt.Errorf("%v failed to authenticate", c.Name)
	}
	c.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

func (c *COINUT) wsGetAccountBalance() (*WsGetAccountBalanceResponse, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}
	accBalance := wsRequest{
		Request: "user_balance",
		Nonce:   c.WebsocketConn.GenerateMessageID(false),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(accBalance.Nonce, accBalance)
	if err != nil {
		return nil, err
	}
	var response WsGetAccountBalanceResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Status[0] != "OK" {
		return &response, fmt.Errorf("%v get account balance failed", c.Name)
	}
	return &response, nil
}

func (c *COINUT) wsSubmitOrder(order *WsSubmitOrderParameters) (*WsStandardOrderResponse, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}
	currency := c.FormatExchangeCurrency(order.Currency, asset.Spot).String()
	var orderSubmissionRequest WsSubmitOrderRequest
	orderSubmissionRequest.Request = "new_order"
	orderSubmissionRequest.Nonce = c.WebsocketConn.GenerateMessageID(false)
	orderSubmissionRequest.InstID = instrumentListByString[currency]
	orderSubmissionRequest.Qty = order.Amount
	orderSubmissionRequest.Price = order.Price
	orderSubmissionRequest.Side = string(order.Side)

	if order.OrderID > 0 {
		orderSubmissionRequest.OrderID = order.OrderID
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(orderSubmissionRequest.Nonce, orderSubmissionRequest)
	if err != nil {
		return nil, err
	}
	var standardOrder WsStandardOrderResponse
	standardOrder, err = c.wsStandardiseOrderResponse(resp)
	if err != nil {
		return nil, err
	}
	if standardOrder.Status[0] != "OK" {
		return &standardOrder, fmt.Errorf("%v order submission failed. %v", c.Name, standardOrder)
	}
	if len(standardOrder.Reasons) > 0 && standardOrder.Reasons[0] != "" {
		return &standardOrder, fmt.Errorf("%v order submission failed. %v", c.Name, standardOrder.Reasons[0])
	}
	return &standardOrder, nil
}

func (c *COINUT) wsStandardiseOrderResponse(resp []byte) (WsStandardOrderResponse, error) {
	var response WsStandardOrderResponse
	var incoming wsResponse
	err := common.JSONDecode(resp, &incoming)
	if err != nil {
		return response, err
	}
	switch incoming.Reply {
	case "order_accepted":
		var orderAccepted WsOrderAcceptedResponse
		err := common.JSONDecode(resp, &orderAccepted)
		if err != nil {
			return response, err
		}
		response = WsStandardOrderResponse{
			InstID:      orderAccepted.InstID,
			Nonce:       orderAccepted.Nonce,
			OpenQty:     orderAccepted.OpenQty,
			OrderID:     orderAccepted.OrderID,
			OrderType:   orderAccepted.Reply,
			Price:       orderAccepted.OrderPrice,
			Qty:         orderAccepted.Qty,
			Side:        orderAccepted.Side,
			Status:      orderAccepted.Status,
			TransID:     orderAccepted.TransID,
			ClientOrdID: orderAccepted.ClientOrdID,
		}
	case "order_filled":
		var orderFilled WsOrderFilledResponse
		err := common.JSONDecode(resp, &orderFilled)
		if err != nil {
			return response, err
		}
		response = WsStandardOrderResponse{
			InstID:      orderFilled.Order.InstID,
			Nonce:       orderFilled.Nonce,
			OpenQty:     orderFilled.Order.OpenQty,
			OrderID:     orderFilled.Order.OrderID,
			OrderType:   orderFilled.Reply,
			Price:       orderFilled.Order.Price,
			Qty:         orderFilled.Order.Qty,
			Side:        orderFilled.Order.Side,
			Status:      orderFilled.Status,
			TransID:     orderFilled.TransID,
			ClientOrdID: orderFilled.Order.ClientOrdID,
		}
	case "order_rejected":
		var orderRejected WsOrderRejectedResponse
		err := common.JSONDecode(resp, &orderRejected)
		if err != nil {
			return response, err
		}
		response = WsStandardOrderResponse{
			InstID:      orderRejected.InstID,
			Nonce:       orderRejected.Nonce,
			OpenQty:     orderRejected.OpenQty,
			OrderID:     orderRejected.OrderID,
			OrderType:   orderRejected.Reply,
			Price:       orderRejected.Price,
			Qty:         orderRejected.Qty,
			Side:        orderRejected.Side,
			Status:      orderRejected.Status,
			TransID:     orderRejected.TransID,
			ClientOrdID: orderRejected.ClientOrdID,
			Reasons:     orderRejected.Reasons,
		}
	}
	return response, nil
}

func (c *COINUT) wsSubmitOrders(orders []WsSubmitOrderParameters) ([]WsStandardOrderResponse, []error) {
	var errors []error
	var ordersResponse []WsStandardOrderResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		errors = append(errors, fmt.Errorf("%v not authorised to submit orders", c.Name))
		return nil, errors
	}
	orderRequest := WsSubmitOrdersRequest{}
	for i := range orders {
		currency := c.FormatExchangeCurrency(orders[i].Currency, asset.Spot).String()
		orderRequest.Orders = append(orderRequest.Orders,
			WsSubmitOrdersRequestData{
				Qty:         orders[i].Amount,
				Price:       orders[i].Price,
				Side:        string(orders[i].Side),
				InstID:      instrumentListByString[currency],
				ClientOrdID: i + 1,
			})
	}

	orderRequest.Nonce = c.WebsocketConn.GenerateMessageID(false)
	orderRequest.Request = "new_orders"
	resp, err := c.WebsocketConn.SendMessageReturnResponse(orderRequest.Nonce, orderRequest)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	var incoming []interface{}
	err = common.JSONDecode(resp, &incoming)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	for i := range incoming {
		var individualJSON []byte
		individualJSON, err = common.JSONEncode(incoming[i])
		if err != nil {
			errors = append(errors, err)
			continue
		}
		standardOrder, err := c.wsStandardiseOrderResponse(individualJSON)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if standardOrder.Status[0] != "OK" {
			errors = append(errors, fmt.Errorf("%v order submission failed. %v", c.Name, standardOrder))
			continue
		}
		if len(standardOrder.Reasons) > 0 && standardOrder.Reasons[0] != "" {
			errors = append(errors, fmt.Errorf("%v order submission failed for currency %v and orderID %v, message %v ",
				c.Name,
				instrumentListByCode[standardOrder.InstID],
				standardOrder.OrderID,
				standardOrder.Reasons[0]))

			continue
		}
		ordersResponse = append(ordersResponse, standardOrder)
	}

	return ordersResponse, errors
}

func (c *COINUT) wsGetOpenOrders(p currency.Pair) error {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authorised to get open orders", c.Name)
	}
	currency := c.FormatExchangeCurrency(p, asset.Spot).String()
	var openOrdersRequest WsGetOpenOrdersRequest
	openOrdersRequest.Request = "user_open_orders"
	openOrdersRequest.Nonce = c.WebsocketConn.GenerateMessageID(false)
	openOrdersRequest.InstID = instrumentListByString[currency]

	resp, err := c.WebsocketConn.SendMessageReturnResponse(openOrdersRequest.Nonce, openOrdersRequest)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response["status"].([]interface{})[0] != "OK" {
		return fmt.Errorf("%v get open orders failed for currency %v",
			c.Name,
			p)
	}
	return nil
}

func (c *COINUT) wsCancelOrder(cancellation WsCancelOrderParameters) error {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authorised to cancel order", c.Name)
	}
	currency := c.FormatExchangeCurrency(cancellation.Currency, asset.Spot).String()
	var cancellationRequest WsCancelOrderRequest
	cancellationRequest.Request = "cancel_order"
	cancellationRequest.InstID = instrumentListByString[currency]
	cancellationRequest.OrderID = cancellation.OrderID
	cancellationRequest.Nonce = c.WebsocketConn.GenerateMessageID(false)

	resp, err := c.WebsocketConn.SendMessageReturnResponse(cancellationRequest.Nonce, cancellationRequest)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response["status"].([]interface{})[0] != "OK" {
		return fmt.Errorf("%v order cancellation failed for currency %v and orderID %v, message %v",
			c.Name,
			cancellation.Currency,
			cancellation.OrderID,
			response["status"])
	}
	return nil
}

func (c *COINUT) wsCancelOrders(cancellations []WsCancelOrderParameters) (*WsCancelOrdersResponse, []error) {
	var errors []error
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errors
	}
	cancelOrderRequest := WsCancelOrdersRequest{}
	for i := range cancellations {
		currency := c.FormatExchangeCurrency(cancellations[i].Currency, asset.Spot).String()
		cancelOrderRequest.Entries = append(cancelOrderRequest.Entries, WsCancelOrdersRequestEntry{
			InstID:  instrumentListByString[currency],
			OrderID: cancellations[i].OrderID,
		})
	}

	cancelOrderRequest.Request = "cancel_orders"
	cancelOrderRequest.Nonce = c.WebsocketConn.GenerateMessageID(false)
	resp, err := c.WebsocketConn.SendMessageReturnResponse(cancelOrderRequest.Nonce, cancelOrderRequest)
	if err != nil {
		return nil, []error{err}
	}
	var response WsCancelOrdersResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, []error{err}
	}
	if response.Status[0] != "OK" {
		return &response, []error{err}
	}
	for i := range response.Results {
		if response.Results[i].Status != "OK" {
			errors = append(errors, fmt.Errorf("%v order cancellation failed for currency %v and orderID %v, message %v",
				c.Name,
				instrumentListByCode[response.Results[i].InstID],
				response.Results[i].OrderID,
				response.Results[i].Status))
		}
	}
	return &response, errors
}

func (c *COINUT) wsGetTradeHistory(p currency.Pair, start, limit int64) error {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v not authorised to get trade history", c.Name)
	}
	currency := c.FormatExchangeCurrency(p, asset.Spot).String()
	var request WsTradeHistoryRequest
	request.Request = "trade_history"
	request.InstID = instrumentListByString[currency]
	request.Nonce = c.WebsocketConn.GenerateMessageID(false)
	request.Start = start
	request.Limit = limit

	resp, err := c.WebsocketConn.SendMessageReturnResponse(request.Nonce, request)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response["status"].([]interface{})[0] != "OK" {
		return fmt.Errorf("%v get trade history failed for %v",
			c.Name,
			request)
	}
	return nil
}
