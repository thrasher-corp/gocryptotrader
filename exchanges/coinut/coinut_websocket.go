package coinut

import (
	"errors"
	"fmt"
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

const coinutWebsocketURL = "wss://wsapi.coinut.com"
const coinutWebsocketRateLimit = 30 * time.Millisecond

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
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var Dialer websocket.Dialer

	if c.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(c.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		Dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	c.WebsocketConn, _, err = Dialer.Dial(c.Websocket.GetWebsocketURL(),
		http.Header{})

	if err != nil {
		return err
	}

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

	go c.WsHandleData()
	return nil
}

// WsReadData reads data from the websocket connection
func (c *COINUT) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := c.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	c.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
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
			resp, err := c.WsReadData()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}

			if strings.HasPrefix(string(resp.Raw), "[") {
				var incoming []wsResponse
				err = common.JSONDecode(resp.Raw, &incoming)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				for i := range incoming {
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
	case "login":
		var login WsLoginResponse
		err := common.JSONDecode(resp, &login)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- login
	case "hb":
		channels["hb"] <- resp
	case "inst_tick":
		var ticker WsTicker
		err := common.JSONDecode(resp, &ticker)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- exchange.TickerData{
			Timestamp:  time.Unix(0, ticker.Timestamp),
			Exchange:   c.GetName(),
			AssetType:  "SPOT",
			HighPrice:  ticker.HighestBuy,
			LowPrice:   ticker.LowestSell,
			ClosePrice: ticker.Last,
			Quantity:   ticker.Volume,
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
		c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: c.GetName(),
			Asset:    "SPOT",
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
		c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: c.GetName(),
			Asset:    "SPOT",
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
		c.Websocket.DataHandler <- exchange.TradeData{
			Timestamp:    time.Unix(tradeUpdate.Timestamp, 0),
			CurrencyPair: currency.NewPairFromString(currencyPair),
			AssetType:    "SPOT",
			Exchange:     c.GetName(),
			Price:        tradeUpdate.Price,
			Side:         tradeUpdate.Side,
		}
	case "user_balance":
		var userBalance WsUserBalanceResponse
		err := common.JSONDecode(resp, &userBalance)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- userBalance
	case "new_order":
		var newOrder WsNewOrderResponse
		err := common.JSONDecode(resp, &newOrder)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- newOrder
	case "order_accepted":
		var orderAccepted WsOrderAcceptedResponse
		err := common.JSONDecode(resp, &orderAccepted)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- orderAccepted
	case "order_filled":
		var orderFilled WsOrderFilledResponse
		err := common.JSONDecode(resp, &orderFilled)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- orderFilled
	case "order_rejected":
		var orderRejected WsOrderRejectedResponse
		err := common.JSONDecode(resp, &orderRejected)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- orderRejected
	case "user_open_orders":
		var openOrders WsUserOpenOrdersResponse
		err := common.JSONDecode(resp, &openOrders)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- openOrders
	case "trade_history":
		var tradeHistory WsTradeHistoryResponse
		err := common.JSONDecode(resp, &tradeHistory)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- tradeHistory
	case "cancel_orders":
		var cancelOrders WsCancelOrdersResponse
		err := common.JSONDecode(resp, &cancelOrders)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- cancelOrders
	case "cancel_order":
		var cancelOrder WsCancelOrderResponse
		err := common.JSONDecode(resp, &cancelOrder)
		if err != nil {
			c.Websocket.DataHandler <- err
			return
		}
		c.Websocket.DataHandler <- cancelOrder
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
	err := c.wsSend(wsRequest{
		Request: "inst_list",
		SecType: "SPOT",
		Nonce:   c.GetNonce(),
	})
	if err != nil {
		return err
	}

	_, resp, err := c.WebsocketConn.ReadMessage()
	if err != nil {
		return err
	}

	c.Websocket.TrafficAlert <- struct{}{}

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
	for _, bid := range ob.Buy {
		bids = append(bids, orderbook.Item{
			Amount: bid.Volume,
			Price:  bid.Price,
		})
	}

	var asks []orderbook.Item
	for _, ask := range ob.Sell {
		asks = append(asks, orderbook.Item{
			Amount: ask.Volume,
			Price:  ask.Price,
		})
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = currency.NewPairFromString(instrumentListByCode[ob.InstID])
	newOrderBook.AssetType = "SPOT"

	return c.Websocket.Orderbook.LoadSnapshot(&newOrderBook, c.GetName(), false)
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(ob *WsOrderbookUpdate) error {
	p := currency.NewPairFromString(instrumentListByCode[ob.InstID])

	if ob.Side == "buy" {
		return c.Websocket.Orderbook.Update([]orderbook.Item{
			{Price: ob.Price, Amount: ob.Volume}},
			nil,
			p,
			time.Now(),
			c.GetName(),
			"SPOT")
	}

	return c.Websocket.Orderbook.Update([]orderbook.Item{
		{Price: ob.Price, Amount: ob.Volume}},
		nil,
		p,
		time.Now(),
		c.GetName(),
		"SPOT")
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *COINUT) GenerateDefaultSubscriptions() {
	var channels = []string{"inst_tick", "inst_order_book"}
	subscriptions := []exchange.WebsocketChannelSubscription{}
	enabledCurrencies := c.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	c.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (c *COINUT) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscribe := wsRequest{
		Request:   channelToSubscribe.Channel,
		InstID:    instrumentListByString[channelToSubscribe.Currency.String()],
		Subscribe: true,
		Nonce:     c.GetNonce(),
	}
	return c.wsSend(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *COINUT) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscribe := wsRequest{
		Request:   channelToSubscribe.Channel,
		InstID:    instrumentListByString[channelToSubscribe.Currency.String()],
		Subscribe: false,
		Nonce:     c.GetNonce(),
	}
	return c.wsSend(subscribe)
}

// WsSend sends data to the websocket server
func (c *COINUT) wsSend(data interface{}) error {
	c.wsRequestMtx.Lock()
	defer c.wsRequestMtx.Unlock()

	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	if c.Verbose {
		log.Debugf("%v sending message to websocket %v", c.Name, string(json))
	}
	// Basic rate limiter
	time.Sleep(coinutWebsocketRateLimit)
	return c.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}

func (c *COINUT) wsAuthenticate() error {
	timestamp := time.Now().Unix()
	nonce := c.GetNonce()
	payload := fmt.Sprintf("%v|%v|%v", c.ClientID, timestamp, nonce)
	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(c.APIKey))
	loginRequest := struct {
		Request   string `json:"request"`
		Username  string `json:"username"`
		Nonce     int64  `json:"nonce"`
		Hmac      string `json:"hmac_sha256"`
		Timestamp int64  `json:"timestamp"`
	}{
		Request:   "login",
		Username:  c.ClientID,
		Nonce:     nonce,
		Hmac:      common.HexEncodeToString(hmac),
		Timestamp: timestamp,
	}

	return c.wsSend(loginRequest)
}

func (c *COINUT) wsGetAccountBalance() error {
	nonce := c.GetNonce()
	loginRequest := wsRequest{
		Request: "user_balance",
		Nonce:   nonce,
	}
	return c.wsSend(loginRequest)
}

func (c *COINUT) wsSubmitOrder(order *WsSubmitOrderParameters) error {
	order.Currency.Delimiter = ""
	currency := order.Currency.Upper().String()
	nonce := c.GetNonce()
	var orderSubmissionRequest WsSubmitOrderRequest
	orderSubmissionRequest.Request = "new_order"
	orderSubmissionRequest.Nonce = nonce
	orderSubmissionRequest.InstID = instrumentListByString[currency]
	orderSubmissionRequest.Qty = order.Amount
	orderSubmissionRequest.Price = order.Price
	orderSubmissionRequest.Side = string(order.Side)

	if order.OrderID > 0 {
		orderSubmissionRequest.OrderID = order.OrderID
	}
	return c.wsSend(orderSubmissionRequest)
}

func (c *COINUT) wsSubmitOrders(orders []WsSubmitOrderParameters) error {
	if len(orders) > 1000 {
		return fmt.Errorf("%v cannot submit more than 1000 orders", c.Name)
	}

	orderRequest := WsSubmitOrdersRequest{}
	for i := range orders {
		orders[i].Currency.Delimiter = ""
		currency := orders[i].Currency.Upper().String()
		orderRequest.Orders = append(orderRequest.Orders,
			WsSubmitOrdersRequestData{
				Qty:         orders[i].Amount,
				Price:       orders[i].Price,
				Side:        string(orders[i].Side),
				InstID:      instrumentListByString[currency],
				ClientOrdID: i + 1,
			})
	}

	orderRequest.Nonce = c.GetNonce()
	orderRequest.Request = "new_orders"
	return c.wsSend(orderRequest)
}

func (c *COINUT) wsGetOpenOrders(p currency.Pair) error {
	nonce := c.GetNonce()
	p.Delimiter = ""
	currency := p.Upper().String()
	var openOrdersRequest WsGetOpenOrdersRequest
	openOrdersRequest.Request = "user_open_orders"
	openOrdersRequest.Nonce = nonce
	openOrdersRequest.InstID = instrumentListByString[currency]

	return c.wsSend(openOrdersRequest)
}

func (c *COINUT) wsCancelOrder(cancellation WsCancelOrderParameters) error {
	nonce := c.GetNonce()
	cancellation.Currency.Delimiter = ""
	currency := cancellation.Currency.Upper().String()
	var cancellationRequest WsCancelOrderRequest
	cancellationRequest.Request = "cancel_order"
	cancellationRequest.InstID = instrumentListByString[currency]
	cancellationRequest.OrderID = cancellation.OrderID
	cancellationRequest.Nonce = nonce

	return c.wsSend(cancellationRequest)
}

func (c *COINUT) wsCancelOrders(cancellations []WsCancelOrderParameters) error {
	cancelOrderRequest := WsCancelOrdersRequest{}
	for i := range cancellations {
		cancellations[i].Currency.Delimiter = ""
		currency := cancellations[i].Currency.Upper().String()
		cancelOrderRequest.Entries = append(cancelOrderRequest.Entries, WsCancelOrdersRequestEntry{
			InstID:  instrumentListByString[currency],
			OrderID: cancellations[i].OrderID,
		})
	}

	nonce := c.GetNonce()
	cancelOrderRequest.Request = "cancel_orders"
	cancelOrderRequest.Nonce = nonce
	return c.wsSend(cancelOrderRequest)
}

func (c *COINUT) wsGetTradeHistory(p currency.Pair, start, limit int64) error {
	nonce := c.GetNonce()
	p.Delimiter = ""
	currency := p.Upper().String()

	var request WsTradeHistoryRequest
	request.Request = "trade_history"
	request.InstID = instrumentListByString[currency]
	request.Nonce = nonce
	request.Start = start
	request.Limit = limit

	return c.wsSend(request)
}
