package coinut

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const coinutWebsocketURL = "wss://wsapi.coinut.com"

var nNonce map[int64]string
var channels map[string]chan []byte
var instrumentListByString map[string]int64
var instrumentListByCode map[int64]string
var populatedList bool

// NOTE for speed considerations
// wss://wsapi-as.coinut.com
// wss://wsapi-na.coinut.com
// wss://wsapi-eu.coinut.com

// WsReadData reads data from the websocket conection
func (c *COINUT) WsReadData() {
	c.Websocket.Wg.Add(1)

	defer func() {
		err := c.WebsocketConn.Close()
		if err != nil {
			c.Websocket.DataHandler <- fmt.Errorf("coinut_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		c.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		default:
			_, resp, err := c.WebsocketConn.ReadMessage()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}

			c.Websocket.TrafficAlert <- struct{}{}
			c.Websocket.Intercomm <- exchange.WebsocketResponse{Raw: resp}
		}
	}
}

// WsHandleData handles read data
func (c *COINUT) WsHandleData() {
	c.Websocket.Wg.Add(1)
	defer c.Websocket.Wg.Done()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		case resp := <-c.Websocket.Intercomm:
			var incoming wsResponse
			err := common.JSONDecode(resp.Raw, &incoming)
			if err != nil {
				log.Fatal(err)
			}

			switch incoming.Reply {
			case "hb":
				channels["hb"] <- resp.Raw

			case "inst_tick":
				var ticker WsTicker
				err := common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					log.Fatal(err)
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
				err := common.JSONDecode(resp.Raw, &orderbooksnapshot)
				if err != nil {
					log.Fatal(err)
				}

				err = c.WsProcessOrderbookSnapshot(orderbooksnapshot)
				if err != nil {
					log.Fatal(err)
				}

				currencyPair := instrumentListByCode[orderbooksnapshot.InstID]

				c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Exchange: c.GetName(),
					Asset:    "SPOT",
					Pair:     pair.NewCurrencyPairFromString(currencyPair),
				}

			case "inst_order_book_update":
				var orderbookUpdate WsOrderbookUpdate
				err := common.JSONDecode(resp.Raw, &orderbookUpdate)
				if err != nil {
					log.Fatal(err)
				}

				err = c.WsProcessOrderbookUpdate(orderbookUpdate)
				if err != nil {
					log.Fatal(err)
				}

				currencyPair := instrumentListByCode[orderbookUpdate.InstID]

				c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Exchange: c.GetName(),
					Asset:    "SPOT",
					Pair:     pair.NewCurrencyPairFromString(currencyPair),
				}

			case "inst_trade":
				var tradeSnap WsTradeSnapshot
				err := common.JSONDecode(resp.Raw, &tradeSnap)
				if err != nil {
					log.Fatal(err)
				}

			case "inst_trade_update":
				var tradeUpdate WsTradeUpdate
				err := common.JSONDecode(resp.Raw, &tradeUpdate)
				if err != nil {
					log.Fatal(err)
				}

				currencyPair := instrumentListByCode[tradeUpdate.InstID]

				c.Websocket.DataHandler <- exchange.TradeData{
					Timestamp:    time.Unix(tradeUpdate.Timestamp, 0),
					CurrencyPair: pair.NewCurrencyPairFromString(currencyPair),
					AssetType:    "SPOT",
					Exchange:     c.GetName(),
					Price:        tradeUpdate.Price,
					Side:         tradeUpdate.Side,
				}
			}
		}
	}
}

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

	err = c.WsSubscribe()
	if err != nil {
		return err
	}

	// define bi-directional communication
	channels = make(map[string]chan []byte)
	channels["hb"] = make(chan []byte, 1)

	go c.WsReadData()
	go c.WsHandleData()

	return nil
}

// GetNonce returns a nonce for a required request
func (c *COINUT) GetNonce() int64 {
	if c.Nonce.Get() == 0 {
		c.Nonce.Set(time.Now().Unix())
	} else {
		c.Nonce.Inc()
	}

	return c.Nonce.Get()
}

// WsSetInstrumentList fetches instrument list and propagates a local cache
func (c *COINUT) WsSetInstrumentList() error {
	request, err := common.JSONEncode(wsRequest{
		Request: "inst_list",
		SecType: "SPOT",
		Nonce:   c.GetNonce(),
	})

	if err != nil {
		return err
	}

	err = c.WebsocketConn.WriteMessage(websocket.TextMessage, request)
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

// WsSubscribe subscribes to websocket streams
func (c *COINUT) WsSubscribe() error {
	pairs := c.GetEnabledCurrencies()

	for _, p := range pairs {
		ticker := wsRequest{
			Request:   "inst_tick",
			InstID:    instrumentListByString[p.Pair().String()],
			Subscribe: true,
			Nonce:     c.GetNonce(),
		}

		tickjson, err := common.JSONEncode(ticker)
		if err != nil {
			return err
		}

		err = c.WebsocketConn.WriteMessage(websocket.TextMessage, tickjson)
		if err != nil {
			return err
		}

		orderbook := wsRequest{
			Request:   "inst_order_book",
			InstID:    instrumentListByString[p.Pair().String()],
			Subscribe: true,
			Nonce:     c.GetNonce(),
		}

		objson, err := common.JSONEncode(orderbook)
		if err != nil {
			return err
		}

		err = c.WebsocketConn.WriteMessage(websocket.TextMessage, objson)
		if err != nil {
			return err
		}
	}
	return nil
}

// WsProcessOrderbookSnapshot processes the orderbook snapshot
func (c *COINUT) WsProcessOrderbookSnapshot(ob WsOrderbookSnapshot) error {
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

	var newOrderbook orderbook.Base
	newOrderbook.Asks = asks
	newOrderbook.Bids = bids
	newOrderbook.CurrencyPair = instrumentListByCode[ob.InstID]
	newOrderbook.Pair = pair.NewCurrencyPairFromString(instrumentListByCode[ob.InstID])
	newOrderbook.AssetType = "SPOT"
	newOrderbook.LastUpdated = time.Now()

	return c.Websocket.Orderbook.LoadSnapshot(newOrderbook, c.GetName())
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(ob WsOrderbookUpdate) error {
	p := pair.NewCurrencyPairFromString(instrumentListByCode[ob.InstID])

	if ob.Side == "buy" {
		return c.Websocket.Orderbook.Update([]orderbook.Item{
			orderbook.Item{Price: ob.Price, Amount: ob.Volume}},
			nil,
			p,
			time.Now(),
			c.GetName(),
			"SPOT")
	}

	return c.Websocket.Orderbook.Update([]orderbook.Item{
		orderbook.Item{Price: ob.Price, Amount: ob.Volume}},
		nil,
		p,
		time.Now(),
		c.GetName(),
		"SPOT")
}
