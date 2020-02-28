package coinut

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	coinutWebsocketURL       = "wss://wsapi.coinut.com"
	coinutWebsocketRateLimit = 30
)

var (
	channels map[string]chan []byte
)

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

	if !c.instrumentMap.IsLoaded() {
		_, err = c.WsGetInstruments()
		if err != nil {
			return err
		}
	}
	err = c.wsAuthenticate()
	if err != nil {
		c.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Error(log.WebsocketMgr, err)
	}
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
				c.Websocket.ReadMessageErrors <- err
				return
			}
			c.Websocket.TrafficAlert <- struct{}{}

			if strings.HasPrefix(string(resp.Raw), "[") {
				var incoming []wsResponse
				err = json.Unmarshal(resp.Raw, &incoming)
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
					individualJSON, err = json.Marshal(incoming[i])
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.wsProcessResponse(individualJSON)
				}
			} else {
				var incoming wsResponse
				err = json.Unmarshal(resp.Raw, &incoming)
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
	err := json.Unmarshal(resp, &incoming)
	if err != nil {
		c.Websocket.DataHandler <- err
		return
	}
	switch incoming.Reply {
	case "hb":
		channels["hb"] <- resp
	case "inst_tick":
		var wsTicker WsTicker
		err := json.Unmarshal(resp, &wsTicker)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}

		currencyPair := c.instrumentMap.LookupInstrument(wsTicker.InstID)
		c.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: c.Name,
			Volume:       wsTicker.Volume24,
			QuoteVolume:  wsTicker.Volume24Quote,
			Bid:          wsTicker.HighestBuy,
			Ask:          wsTicker.LowestSell,
			High:         wsTicker.High24,
			Low:          wsTicker.Low24,
			Last:         wsTicker.Last,
			LastUpdated:  time.Unix(0, wsTicker.Timestamp),
			AssetType:    asset.Spot,
			Pair: currency.NewPairFromFormattedPairs(currencyPair,
				c.GetEnabledPairs(asset.Spot),
				c.GetPairFormat(asset.Spot, true)),
		}

	case "inst_order_book":
		var orderbooksnapshot WsOrderbookSnapshot
		err := json.Unmarshal(resp, &orderbooksnapshot)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		err = c.WsProcessOrderbookSnapshot(&orderbooksnapshot)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := c.instrumentMap.LookupInstrument(orderbooksnapshot.InstID)
		c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Exchange: c.Name,
			Asset:    asset.Spot,
			Pair: currency.NewPairFromFormattedPairs(currencyPair,
				c.GetEnabledPairs(asset.Spot),
				c.GetPairFormat(asset.Spot, true)),
		}
	case "inst_order_book_update":
		var orderbookUpdate WsOrderbookUpdate
		err := json.Unmarshal(resp, &orderbookUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		err = c.WsProcessOrderbookUpdate(&orderbookUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := c.instrumentMap.LookupInstrument(orderbookUpdate.InstID)
		c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Exchange: c.Name,
			Asset:    asset.Spot,
			Pair: currency.NewPairFromFormattedPairs(currencyPair,
				c.GetEnabledPairs(asset.Spot),
				c.GetPairFormat(asset.Spot, true)),
		}
	case "inst_trade":
		var tradeSnap WsTradeSnapshot
		err := json.Unmarshal(resp, &tradeSnap)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}

	case "inst_trade_update":
		var tradeUpdate WsTradeUpdate
		err := json.Unmarshal(resp, &tradeUpdate)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		currencyPair := c.instrumentMap.LookupInstrument(tradeUpdate.InstID)
		c.Websocket.DataHandler <- wshandler.TradeData{
			Timestamp: time.Unix(tradeUpdate.Timestamp, 0),
			CurrencyPair: currency.NewPairFromFormattedPairs(currencyPair,
				c.GetEnabledPairs(asset.Spot),
				c.GetPairFormat(asset.Spot, true)),
			AssetType: asset.Spot,
			Exchange:  c.Name,
			Price:     tradeUpdate.Price,
			Side:      tradeUpdate.Side,
		}
	default:
		if incoming.Nonce > 0 {
			c.WebsocketConn.AddResponseWithID(incoming.Nonce, resp)
			return
		}
		c.Websocket.DataHandler <- fmt.Errorf("%v unhandled websocket response: %s", c.Name, resp)
	}
}

// WsGetInstruments fetches instrument list and propagates a local cache
func (c *COINUT) WsGetInstruments() (Instruments, error) {
	var list Instruments
	request := wsRequest{
		Request: "inst_list",
		SecType: strings.ToUpper(asset.Spot.String()),
		Nonce:   getNonce(),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(request.Nonce, request)
	if err != nil {
		return list, err
	}
	err = json.Unmarshal(resp, &list)
	if err != nil {
		return list, err
	}
	for curr, data := range list.Instruments {
		c.instrumentMap.Seed(curr, data[0].InstID)
	}
	if len(c.instrumentMap.GetInstrumentIDs()) == 0 {
		return list, errors.New("instrument list failed to populate")
	}
	return list, nil
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
	newOrderBook.Pair = currency.NewPairFromFormattedPairs(
		c.instrumentMap.LookupInstrument(ob.InstID),
		c.GetEnabledPairs(asset.Spot),
		c.GetPairFormat(asset.Spot, true),
	)
	newOrderBook.AssetType = asset.Spot
	newOrderBook.ExchangeName = c.Name

	return c.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(update *WsOrderbookUpdate) error {
	p := currency.NewPairFromFormattedPairs(
		c.instrumentMap.LookupInstrument(update.InstID),
		c.GetEnabledPairs(asset.Spot),
		c.GetPairFormat(asset.Spot, true),
	)
	bufferUpdate := &wsorderbook.WebsocketOrderbookUpdate{
		Pair:     p,
		UpdateID: update.TransID,
		Asset:    asset.Spot,
	}
	if strings.EqualFold(update.Side, order.Buy.Lower()) {
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
		Request: channelToSubscribe.Channel,
		InstID: c.instrumentMap.LookupID(c.FormatExchangeCurrency(channelToSubscribe.Currency,
			asset.Spot).String()),
		Subscribe: true,
		Nonce:     getNonce(),
	}
	return c.WebsocketConn.SendJSONMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *COINUT) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := wsRequest{
		Request: channelToSubscribe.Channel,
		InstID: c.instrumentMap.LookupID(c.FormatExchangeCurrency(channelToSubscribe.Currency,
			asset.Spot).String()),
		Subscribe: false,
		Nonce:     getNonce(),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(subscribe.Nonce, subscribe)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = json.Unmarshal(resp, &response)
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
	nonce := getNonce()
	payload := c.API.Credentials.ClientID + "|" +
		strconv.FormatInt(timestamp, 10) + "|" +
		strconv.FormatInt(nonce, 10)
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
	err = json.Unmarshal(resp, &response)
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

func (c *COINUT) wsGetAccountBalance() (*UserBalance, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}
	accBalance := wsRequest{
		Request: "user_balance",
		Nonce:   getNonce(),
	}
	resp, err := c.WebsocketConn.SendMessageReturnResponse(accBalance.Nonce, accBalance)
	if err != nil {
		return nil, err
	}
	var response UserBalance
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Status[0] != "OK" {
		return &response, fmt.Errorf("%v get account balance failed", c.Name)
	}
	return &response, nil
}

func (c *COINUT) wsSubmitOrder(o *WsSubmitOrderParameters) (*WsStandardOrderResponse, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}
	curr := c.FormatExchangeCurrency(o.Currency, asset.Spot).String()
	var orderSubmissionRequest WsSubmitOrderRequest
	orderSubmissionRequest.Request = "new_order"
	orderSubmissionRequest.Nonce = getNonce()
	orderSubmissionRequest.InstID = c.instrumentMap.LookupID(curr)
	orderSubmissionRequest.Qty = o.Amount
	orderSubmissionRequest.Price = o.Price
	orderSubmissionRequest.Side = string(o.Side)

	if o.OrderID > 0 {
		orderSubmissionRequest.OrderID = o.OrderID
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
	err := json.Unmarshal(resp, &incoming)
	if err != nil {
		return response, err
	}
	switch incoming.Reply {
	case "order_accepted":
		var orderAccepted WsOrderAcceptedResponse
		err := json.Unmarshal(resp, &orderAccepted)
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
		err := json.Unmarshal(resp, &orderFilled)
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
		err := json.Unmarshal(resp, &orderRejected)
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
		curr := c.FormatExchangeCurrency(orders[i].Currency, asset.Spot).String()
		orderRequest.Orders = append(orderRequest.Orders,
			WsSubmitOrdersRequestData{
				Qty:         orders[i].Amount,
				Price:       orders[i].Price,
				Side:        string(orders[i].Side),
				InstID:      c.instrumentMap.LookupID(curr),
				ClientOrdID: i + 1,
			})
	}

	orderRequest.Nonce = getNonce()
	orderRequest.Request = "new_orders"
	resp, err := c.WebsocketConn.SendMessageReturnResponse(orderRequest.Nonce, orderRequest)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	var incoming []interface{}
	err = json.Unmarshal(resp, &incoming)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	for i := range incoming {
		var individualJSON []byte
		individualJSON, err = json.Marshal(incoming[i])
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
				c.instrumentMap.LookupInstrument(standardOrder.InstID),
				standardOrder.OrderID,
				standardOrder.Reasons[0]))

			continue
		}
		ordersResponse = append(ordersResponse, standardOrder)
	}

	return ordersResponse, errors
}

func (c *COINUT) wsGetOpenOrders(curr string) (*WsUserOpenOrdersResponse, error) {
	var response *WsUserOpenOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get open orders", c.Name)
	}
	var openOrdersRequest WsGetOpenOrdersRequest
	openOrdersRequest.Request = "user_open_orders"
	openOrdersRequest.Nonce = getNonce()
	openOrdersRequest.InstID = c.instrumentMap.LookupID(curr)

	resp, err := c.WebsocketConn.SendMessageReturnResponse(openOrdersRequest.Nonce, openOrdersRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get open orders failed for currency %v",
			c.Name,
			curr)
	}
	return response, nil
}

func (c *COINUT) wsCancelOrder(cancellation *WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var response *CancelOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to cancel order", c.Name)
	}
	curr := c.FormatExchangeCurrency(cancellation.Currency, asset.Spot).String()
	var cancellationRequest WsCancelOrderRequest
	cancellationRequest.Request = "cancel_order"
	cancellationRequest.InstID = c.instrumentMap.LookupID(curr)
	cancellationRequest.OrderID = cancellation.OrderID
	cancellationRequest.Nonce = getNonce()

	resp, err := c.WebsocketConn.SendMessageReturnResponse(cancellationRequest.Nonce, cancellationRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v order cancellation failed for currency %v and orderID %v, message %v",
			c.Name,
			cancellation.Currency,
			cancellation.OrderID,
			response.Status[0])
	}
	return response, nil
}

func (c *COINUT) wsCancelOrders(cancellations []WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var err error
	var response *CancelOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, err
	}
	var cancelOrderRequest WsCancelOrdersRequest
	for i := range cancellations {
		curr := c.FormatExchangeCurrency(cancellations[i].Currency, asset.Spot).String()
		cancelOrderRequest.Entries = append(cancelOrderRequest.Entries, WsCancelOrdersRequestEntry{
			InstID:  c.instrumentMap.LookupID(curr),
			OrderID: cancellations[i].OrderID,
		})
	}

	cancelOrderRequest.Request = "cancel_orders"
	cancelOrderRequest.Nonce = getNonce()
	resp, err := c.WebsocketConn.SendMessageReturnResponse(cancelOrderRequest.Nonce, cancelOrderRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	return response, err
}

func (c *COINUT) wsGetTradeHistory(p currency.Pair, start, limit int64) (*WsTradeHistoryResponse, error) {
	var response *WsTradeHistoryResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get trade history", c.Name)
	}
	curr := c.FormatExchangeCurrency(p, asset.Spot).String()
	var request WsTradeHistoryRequest
	request.Request = "trade_history"
	request.InstID = c.instrumentMap.LookupID(curr)
	request.Nonce = getNonce()
	request.Start = start
	request.Limit = limit

	resp, err := c.WebsocketConn.SendMessageReturnResponse(request.Nonce, request)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get trade history failed for %v",
			c.Name,
			request)
	}
	return response, nil
}
