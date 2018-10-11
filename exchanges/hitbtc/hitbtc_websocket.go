package hitbtc

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

const (
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
)

var channels map[string]chan []byte

// WsConnect starts a new connection with the websocket API
func (h *HitBTC) WsConnect() error {
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

	var err error
	h.WebsocketConn, _, err = dialer.Dial(hitbtcWebsocketAddress, http.Header{})
	if err != nil {
		return err
	}

	channels = make(map[string]chan []byte, 1)
	channels["readData"] = make(chan []byte, 1)

	go h.WsReadData()
	go h.WsHandleData()

	err = h.WsSubscribe()
	if err != nil {
		return err
	}

	return nil
}

// WsSubscribe subscribes to the relevant channels
func (h *HitBTC) WsSubscribe() error {
	enabledPairs := h.GetEnabledCurrencies()
	for _, p := range enabledPairs {
		pF := exchange.FormatExchangeCurrency(h.GetName(), p)

		tickerSubReq, err := common.JSONEncode(WsNotification{
			JSONRPCVersion: rpcVersion,
			Method:         "subscribeTicker",
			Params:         params{Symbol: pF.String()},
		})
		if err != nil {
			return err
		}

		err = h.WebsocketConn.WriteMessage(websocket.TextMessage, tickerSubReq)
		if err != nil {
			return nil
		}

		orderbookSubReq, err := common.JSONEncode(WsNotification{
			JSONRPCVersion: rpcVersion,
			Method:         "subscribeOrderbook",
			Params:         params{Symbol: pF.String()},
		})
		if err != nil {
			return err
		}

		err = h.WebsocketConn.WriteMessage(websocket.TextMessage, orderbookSubReq)
		if err != nil {
			return nil
		}

		tradeSubReq, err := common.JSONEncode(WsNotification{
			JSONRPCVersion: rpcVersion,
			Method:         "subscribeTrades",
			Params:         params{Symbol: pF.String()},
		})
		if err != nil {
			return err
		}

		err = h.WebsocketConn.WriteMessage(websocket.TextMessage, tradeSubReq)
		if err != nil {
			return nil
		}
	}
	return nil
}

// WsReadData reads from the websocket connection
func (h *HitBTC) WsReadData() {
	h.Websocket.Wg.Add(1)
	defer func() {
		err := h.WebsocketConn.Close()
		if err != nil {
			h.Websocket.DataHandler <- fmt.Errorf("hitbtc_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		h.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-h.Websocket.ShutdownC:
			return

		default:
			_, resp, err := h.WebsocketConn.ReadMessage()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}

			h.Websocket.TrafficAlert <- struct{}{}
			channels["readData"] <- resp
		}
	}
}

// WsHandleData handles websocket data
func (h *HitBTC) WsHandleData() {
	h.Websocket.Wg.Add(1)
	defer h.Websocket.Wg.Done()

	for {
		select {
		case <-h.Websocket.ShutdownC:

		case resp := <-channels["readData"]:
			var init capture
			err := common.JSONDecode(resp, &init)
			if err != nil {
				log.Fatal(err)
			}

			if init.Error.Message != "" || init.Error.Code != 0 {
				h.Websocket.DataHandler <- fmt.Errorf("hitbtc.go error - Code: %d, Message: %s",
					init.Error.Code,
					init.Error.Message)
				continue
			}

			if init.Result {
				continue
			}

			switch init.Method {
			case "ticker":
				var ticker WsTicker
				err := common.JSONDecode(resp, &ticker)
				if err != nil {
					log.Fatal(err)
				}

				ts, err := time.Parse(time.RFC3339, ticker.Params.Timestamp)
				if err != nil {
					log.Fatal(err)
				}

				h.Websocket.DataHandler <- exchange.TickerData{
					Exchange:  h.GetName(),
					AssetType: "SPOT",
					Pair:      pair.NewCurrencyPairFromString(ticker.Params.Symbol),
					Quantity:  ticker.Params.Volume,
					Timestamp: ts,
					OpenPrice: ticker.Params.Open,
					HighPrice: ticker.Params.High,
					LowPrice:  ticker.Params.Low,
				}

			case "snapshotOrderbook":
				var obSnapshot WsOrderbook
				err := common.JSONDecode(resp, &obSnapshot)
				if err != nil {
					log.Fatal(err)
				}

				err = h.WsProcessOrderbookSnapshot(obSnapshot)
				if err != nil {
					log.Fatal(err)
				}

				h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Exchange: h.GetName(),
					Asset:    "SPOT",
					Pair:     pair.NewCurrencyPairFromString(obSnapshot.Params.Symbol),
				}

			case "updateOrderbook":
				var obUpdate WsOrderbook
				err := common.JSONDecode(resp, &obUpdate)
				if err != nil {
					log.Fatal(err)
				}

				h.WsProcessOrderbookUpdate(obUpdate)

			case "snapshotTrades":
				var tradeSnapshot WsTrade
				err := common.JSONDecode(resp, &tradeSnapshot)
				if err != nil {
					log.Fatal(err)
				}

			case "updateTrades":
				var tradeUpdates WsTrade
				err := common.JSONDecode(resp, &tradeUpdates)
				if err != nil {
					log.Fatal(err)
				}

			default:
				log.Fatal("edge case: ", string(resp))
			}
		}
	}
}

var orderbookCache []orderbook.Base
var obMtx sync.Mutex

// WsProcessOrderbookSnapshot processes a full orderbook snapshot to a local cache
func (h *HitBTC) WsProcessOrderbookSnapshot(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 || len(ob.Params.Ask) == 0 {
		return errors.New("hitbtc.go error - no orderbooks to process")
	}

	var bids []orderbook.Item
	for _, bid := range ob.Params.Bid {
		bids = append(bids, orderbook.Item{Amount: bid.Size, Price: bid.Price})
	}

	var asks []orderbook.Item
	for _, ask := range ob.Params.Ask {
		asks = append(asks, orderbook.Item{Amount: ask.Size, Price: ask.Price})
	}

	var newOrderbook orderbook.Base
	newOrderbook.Asks = asks
	newOrderbook.Bids = bids
	newOrderbook.AssetType = "SPOT"
	newOrderbook.CurrencyPair = ob.Params.Symbol
	newOrderbook.LastUpdated = time.Now()
	newOrderbook.Pair = pair.NewCurrencyPairFromString(ob.Params.Symbol)

	obMtx.Lock()
	orderbookCache = append(orderbookCache, newOrderbook)
	obMtx.Unlock()

	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (h *HitBTC) WsProcessOrderbookUpdate(ob WsOrderbook) {
	go func() {
		if len(ob.Params.Bid) == 0 && len(ob.Params.Ask) == 0 {
			return
		}

		if len(ob.Params.Bid) > 0 {
			for _, bid := range ob.Params.Bid {
				err := h.WsUpdateBid(bid.Price, bid.Size, ob.Params.Symbol)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		if len(ob.Params.Bid) > 0 {
			for _, ask := range ob.Params.Ask {
				err := h.WsUpdateAsk(ask.Price, ask.Size, ob.Params.Symbol)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: h.GetName(),
			Asset:    "SPOT",
			Pair:     pair.NewCurrencyPairFromString(ob.Params.Symbol),
		}
	}()
}

// WsUpdateBid updates bid side on orderbook cache
func (h *HitBTC) WsUpdateBid(target, volume float64, symbol string) error {
	obMtx.Lock()
	defer obMtx.Unlock()

	for x := range orderbookCache {
		if orderbookCache[x].CurrencyPair == symbol {
			for y := range orderbookCache[x].Bids {
				if orderbookCache[x].Bids[y].Price == target {
					if volume == 0 {
						orderbookCache[x].Bids = append(orderbookCache[x].Bids[:y],
							orderbookCache[x].Bids[y+1:]...)
						return nil
					}
					orderbookCache[x].Bids[y].Amount = volume
					return nil
				}
			}

			if volume == 0 {
				return nil
			}

			orderbookCache[x].Bids = append(orderbookCache[x].Bids,
				orderbook.Item{
					Price:  target,
					Amount: volume})
			return nil
		}
	}
	return errors.New("hitbtc.go error - symbol not found")
}

// WsUpdateAsk updates ask side on orderbook cache
func (h *HitBTC) WsUpdateAsk(target, volume float64, symbol string) error {
	obMtx.Lock()
	defer obMtx.Unlock()

	for x := range orderbookCache {
		if orderbookCache[x].CurrencyPair == symbol {
			for y := range orderbookCache[x].Asks {
				if orderbookCache[x].Asks[y].Price == target {
					if volume == 0 {
						orderbookCache[x].Asks = append(orderbookCache[x].Asks[:y],
							orderbookCache[x].Asks[y+1:]...)
						return nil
					}
					orderbookCache[x].Asks[y].Amount = volume
					return nil
				}
			}
			if volume == 0 {
				return nil
			}

			orderbookCache[x].Asks = append(orderbookCache[x].Asks,
				orderbook.Item{
					Price:  target,
					Amount: volume})
			return nil
		}
	}
	return errors.New("hitbtc.go error - symbol not found")
}

type capture struct {
	Method string `json:"method"`
	Result bool   `json:"result"`
	Error  struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WsRequest defines a request obj for the JSON-RPC and gets a websocket
// response
type WsRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
	ID     interface{} `json:"id"`
}

// WsNotification defines a notification obj for the JSON-RPC this does not get
// a websocket response
type WsNotification struct {
	JSONRPCVersion string      `json:"jsonrpc"`
	Method         string      `json:"method"`
	Params         interface{} `json:"params"`
}

type params struct {
	Symbol string `json:"symbol"`
}

// WsTicker defines websocket ticker feed return params
type WsTicker struct {
	Params struct {
		Ask         float64 `json:"ask,string"`
		Bid         float64 `json:"bid,string"`
		Last        float64 `json:"last,string"`
		Open        float64 `json:"open,string"`
		Low         float64 `json:"low,string"`
		High        float64 `json:"high,string"`
		Volume      float64 `json:"volume,string"`
		VolumeQuote float64 `json:"volumeQuote,string"`
		Timestamp   string  `json:"timestamp"`
		Symbol      string  `json:"symbol"`
	} `json:"params"`
}

// WsOrderbook defines websocket orderbook feed return params
type WsOrderbook struct {
	Params struct {
		Ask []struct {
			Price float64 `json:"price,string"`
			Size  float64 `json:"size,string"`
		} `json:"ask"`
		Bid []struct {
			Price float64 `json:"price,string"`
			Size  float64 `json:"size,string"`
		} `json:"bid"`
		Symbol   string `json:"symbol"`
		Sequence int64  `json:"sequence"`
	} `json:"params"`
}

// WsTrade defines websocket trade feed return params
type WsTrade struct {
	Params struct {
		Data []struct {
			ID        int64   `json:"id"`
			Price     float64 `json:"price,string"`
			Quantity  float64 `json:"quantity,string"`
			Side      string  `json:"side"`
			Timestamp string  `json:"timestamp"`
		} `json:"data"`
		Symbol string `json:"symbol"`
	} `json:"params"`
}
