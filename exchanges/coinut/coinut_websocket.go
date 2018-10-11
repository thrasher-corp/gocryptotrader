package coinut

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
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

type wsRequest struct {
	Request   string `json:"request"`
	SecType   string `json:"sec_type,omitempty"`
	InstID    int64  `json:"inst_id,omitempty"`
	TopN      int64  `json:"top_n,omitempty"`
	Subscribe bool   `json:"subscribe"`
	Nonce     int64  `json:"nonce"`
}

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
			channels["comms"] <- resp
		}
	}
}

type wsResponse struct {
	Reply string `json:"reply"`
}

// WsHandleData handles read data
func (c *COINUT) WsHandleData() {
	c.Websocket.Wg.Add(1)
	defer c.Websocket.Wg.Done()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		case resp := <-channels["comms"]:
			var incoming wsResponse
			err := common.JSONDecode(resp, &incoming)
			if err != nil {
				log.Fatal(err)
			}

			switch incoming.Reply {
			case "hb":
				channels["hb"] <- resp

			case "inst_tick":
				var ticker WsTicker
				err := common.JSONDecode(resp, &ticker)
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
				err := common.JSONDecode(resp, &orderbooksnapshot)
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
				err := common.JSONDecode(resp, &orderbookUpdate)
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
				err := common.JSONDecode(resp, &tradeSnap)
				if err != nil {
					log.Fatal(err)
				}

			case "inst_trade_update":
				var tradeUpdate WsTradeUpdate
				err := common.JSONDecode(resp, &tradeUpdate)
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

			default:
				log.Fatal("Edge case:", string(resp))
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
	channels["comms"] = make(chan []byte, 1)
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

var orderbookCache []orderbook.Base
var obMtx sync.Mutex

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
	newOrderbook.LastUpdated = time.Now()

	obMtx.Lock()
	orderbookCache = append(orderbookCache, newOrderbook)
	obMtx.Unlock()

	return nil
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(ob WsOrderbookUpdate) error {
	obMtx.Lock()
	defer obMtx.Unlock()

	for x := range orderbookCache {
		if orderbookCache[x].CurrencyPair == instrumentListByCode[ob.InstID] {
			if ob.Side == "buy" {
				for y := range orderbookCache[x].Bids {
					if orderbookCache[x].Bids[y].Price == ob.Price {
						if ob.Volume == 0 {
							orderbookCache[x].Bids = append(orderbookCache[x].Bids[:y],
								orderbookCache[x].Bids[y+1:]...)
							return nil
						}
						orderbookCache[x].Bids[y].Amount = ob.Volume
						return nil
					}
				}
				orderbookCache[x].Bids = append(orderbookCache[x].Bids,
					orderbook.Item{
						Amount: ob.Volume,
						Price:  ob.Price,
					})
				return nil
			}

			for y := range orderbookCache[x].Asks {
				if orderbookCache[x].Asks[y].Price == ob.Price {
					if ob.Volume == 0 {
						orderbookCache[x].Asks = append(orderbookCache[x].Asks[:y],
							orderbookCache[x].Asks[y+1:]...)
						return nil
					}
					orderbookCache[x].Asks[y].Amount = ob.Volume
					return nil
				}
			}
			orderbookCache[x].Bids = append(orderbookCache[x].Asks,
				orderbook.Item{
					Amount: ob.Volume,
					Price:  ob.Price,
				})
			return nil
		}
	}
	return errors.New("coinut.go error - currency pair not found")
}

type wsHeartbeatResp struct {
	Nonce  int64         `json:"nonce"`
	Reply  string        `json:"reply"`
	Status []interface{} `json:"status"`
}

// WsTicker defines the resp for ticker updates from the websocket connection
type WsTicker struct {
	HighestBuy   float64 `json:"highest_buy,string"`
	InstID       int64   `json:"inst_id"`
	Last         float64 `json:"last,string"`
	LowestSell   float64 `json:"lowest_sell,string"`
	OpenInterest float64 `json:"open_interest,string"`
	Reply        string  `json:"reply"`
	Timestamp    int64   `json:"timestamp"`
	TransID      int64   `json:"trans_id"`
	Volume       float64 `json:"volume,string"`
	Volume24H    float64 `json:"volume24,string"`
}

// WsOrderbookSnapshot defines the resp for orderbook snapshot updates from
// the websocket connection
type WsOrderbookSnapshot struct {
	Buy       []WsOrderbookData `json:"buy"`
	Sell      []WsOrderbookData `json:"sell"`
	InstID    int64             `json:"inst_id"`
	Nonce     int64             `json:"nonce"`
	TotalBuy  float64           `json:"total_buy,string"`
	TotalSell float64           `json:"total_sell,string"`
	Reply     string            `json:"reply"`
	Status    []interface{}     `json:"status"`
}

// WsOrderbookData defines singular orderbook data
type WsOrderbookData struct {
	Count  int64   `json:"count"`
	Price  float64 `json:"price,string"`
	Volume float64 `json:"qty,string"`
}

// WsOrderbookUpdate defines orderbook update response from the websocket
// connection
type WsOrderbookUpdate struct {
	Count    int64   `json:"count"`
	InstID   int64   `json:"inst_id"`
	Price    float64 `json:"price,string"`
	Volume   float64 `json:"qty,string"`
	TotalBuy float64 `json:"total_buy,string"`
	Reply    string  `json:"reply"`
	Side     string  `json:"side"`
	TransID  int64   `json:"trans_id"`
}

// WsTradeSnapshot defines Market trade response from the websocket
// connection
type WsTradeSnapshot struct {
	Nonce  int64         `json:"nonce"`
	Reply  string        `json:"reply"`
	Status []interface{} `json:"status"`
	Trades []WsTradeData `json:"trades"`
}

// WsTradeData defines market trade data
type WsTradeData struct {
	Price     float64 `json:"price,string"`
	Volume    float64 `json:"qty,string"`
	Side      string  `json:"side"`
	Timestamp int64   `json:"timestamp"`
	TransID   int64   `json:"trans_id"`
	// Where's instrument ID?
}

// WsTradeUpdate defines trade update response from the websocket connection
type WsTradeUpdate struct {
	InstID    int64   `json:"inst_id"`
	Price     float64 `json:"price,string"`
	Reply     string  `json:"reply"`
	Side      string  `json:"side"`
	Timestamp int64   `json:"timestamp"`
	TransID   int64   `json:"trans_id"`
	// Where's volume?
}

// WsInstrumentList defines instrument list
type WsInstrumentList struct {
	Spot   map[string][]WsSupportedCurrency `json:"SPOT"`
	Nonce  int64                            `json:"nonce"`
	Reply  string                           `json:"inst_list"`
	Status []interface{}                    `json:"status"`
}

// WsSupportedCurrency defines supported currency on the exchange
type WsSupportedCurrency struct {
	Base          string `json:"base"`
	InstID        int64  `json:"inst_id"`
	DecimalPlaces int64  `json:"decimal_places"`
	Quote         string `json:"quote"`
}
