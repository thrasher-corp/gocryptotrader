package coinut

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
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
			resp, err := c.WsReadData()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}

			var incoming wsResponse
			err = common.JSONDecode(resp.Raw, &incoming)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}

			switch incoming.Reply {
			case "hb":
				channels["hb"] <- resp.Raw

			case "inst_tick":
				var ticker WsTicker
				err := common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				c.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  time.Unix(0, ticker.Timestamp),
					Exchange:   c.GetName(),
					AssetType:  assets.AssetTypeSpot,
					HighPrice:  ticker.HighestBuy,
					LowPrice:   ticker.LowestSell,
					ClosePrice: ticker.Last,
					Quantity:   ticker.Volume,
				}

			case "inst_order_book":
				var orderbooksnapshot WsOrderbookSnapshot
				err := common.JSONDecode(resp.Raw, &orderbooksnapshot)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				err = c.WsProcessOrderbookSnapshot(&orderbooksnapshot)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				currencyPair := instrumentListByCode[orderbooksnapshot.InstID]

				c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Exchange: c.GetName(),
					Asset:    assets.AssetTypeSpot,
					Pair:     currency.NewPairFromString(currencyPair),
				}

			case "inst_order_book_update":
				var orderbookUpdate WsOrderbookUpdate
				err := common.JSONDecode(resp.Raw, &orderbookUpdate)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				err = c.WsProcessOrderbookUpdate(&orderbookUpdate)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				currencyPair := instrumentListByCode[orderbookUpdate.InstID]

				c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Exchange: c.GetName(),
					Asset:    assets.AssetTypeSpot,
					Pair:     currency.NewPairFromString(currencyPair),
				}

			case "inst_trade":
				var tradeSnap WsTradeSnapshot
				err := common.JSONDecode(resp.Raw, &tradeSnap)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

			case "inst_trade_update":
				var tradeUpdate WsTradeUpdate
				err := common.JSONDecode(resp.Raw, &tradeUpdate)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				currencyPair := instrumentListByCode[tradeUpdate.InstID]

				c.Websocket.DataHandler <- exchange.TradeData{
					Timestamp:    time.Unix(tradeUpdate.Timestamp, 0),
					CurrencyPair: currency.NewPairFromString(currencyPair),
					AssetType:    assets.AssetTypeSpot,
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

	return int64(c.Nonce.Get())
}

// WsSetInstrumentList fetches instrument list and propagates a local cache
func (c *COINUT) WsSetInstrumentList() error {
	req, err := common.JSONEncode(wsRequest{
		Request: "inst_list",
		SecType: "spot",
		Nonce:   c.GetNonce(),
	})

	if err != nil {
		return err
	}

	err = c.WebsocketConn.WriteMessage(websocket.TextMessage, req)
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
	pairs := c.GetEnabledPairs(assets.AssetTypeSpot)

	for _, p := range pairs {
		ticker := wsRequest{
			Request:   "inst_tick",
			InstID:    instrumentListByString[p.String()],
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

		ob := wsRequest{
			Request:   "inst_order_book",
			InstID:    instrumentListByString[p.String()],
			Subscribe: true,
			Nonce:     c.GetNonce(),
		}

		objson, err := common.JSONEncode(ob)
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
	newOrderBook.AssetType = assets.AssetTypeSpot
	newOrderBook.LastUpdated = time.Now()

	return c.Websocket.Orderbook.LoadSnapshot(&newOrderBook, c.GetName(), false)
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(ob *WsOrderbookUpdate) error {
	p := currency.NewPairFromString(instrumentListByCode[ob.InstID])

	if ob.Side == exchange.BuyOrderSide.ToLower().ToString() {
		return c.Websocket.Orderbook.Update([]orderbook.Item{
			{Price: ob.Price, Amount: ob.Volume}},
			nil,
			p,
			time.Now(),
			c.GetName(),
			assets.AssetTypeSpot)
	}

	return c.Websocket.Orderbook.Update([]orderbook.Item{
		{Price: ob.Price, Amount: ob.Volume}},
		nil,
		p,
		time.Now(),
		c.GetName(),
		assets.AssetTypeSpot)
}
