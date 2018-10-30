package hitbtc

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

const (
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
)

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

	go h.WsHandleData()

	return h.WsSubscribe()
}

// WsSubscribe subscribes to the relevant channels
func (h *HitBTC) WsSubscribe() error {
	enabledPairs := h.GetEnabledPairs(assets.AssetTypeSpot)
	for _, p := range enabledPairs {
		pF := h.FormatExchangeCurrency(p, assets.AssetTypeSpot)

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
func (h *HitBTC) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := h.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	h.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

// WsHandleData handles websocket data
func (h *HitBTC) WsHandleData() {
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
			resp, err := h.WsReadData()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}

			var init capture
			err = common.JSONDecode(resp.Raw, &init)
			if err != nil {
				h.Websocket.DataHandler <- err
				continue
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
				err := common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				ts, err := time.Parse(time.RFC3339, ticker.Params.Timestamp)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				h.Websocket.DataHandler <- exchange.TickerData{
					Exchange:  h.GetName(),
					AssetType: assets.AssetTypeSpot,
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
					continue
				}

				err = h.WsProcessOrderbookSnapshot(obSnapshot)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

			case "updateOrderbook":
				var obUpdate WsOrderbook
				err := common.JSONDecode(resp.Raw, &obUpdate)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				h.WsProcessOrderbookUpdate(obUpdate)

			case "snapshotTrades":
				var tradeSnapshot WsTrade
				err := common.JSONDecode(resp.Raw, &tradeSnapshot)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

			case "updateTrades":
				var tradeUpdates WsTrade
				err := common.JSONDecode(resp.Raw, &tradeUpdates)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}
			}
		}
	}
}

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

	p := currency.NewPairFromString(ob.Params.Symbol)

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.AssetType = assets.AssetTypeSpot
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.Pair = p

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, h.GetName(), false)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     p,
	}

	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (h *HitBTC) WsProcessOrderbookUpdate(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 && len(ob.Params.Ask) == 0 {
		return errors.New("hitbtc_websocket.go error - no data")
	}

	var bids, asks []orderbook.Item
	for _, bid := range ob.Params.Bid {
		bids = append(bids, orderbook.Item{Price: bid.Price, Amount: bid.Size})
	}

	for _, ask := range ob.Params.Ask {
		asks = append(asks, orderbook.Item{Price: ask.Price, Amount: ask.Size})
	}

	p := currency.NewPairFromString(ob.Params.Symbol)

	err := h.Websocket.Orderbook.Update(bids, asks, p, time.Now(), h.GetName(), assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     p,
	}
	return nil
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
