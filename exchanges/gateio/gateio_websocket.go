package gateio

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	gateioWebsocketEndpoint = "wss://ws.gate.io/v3/"
	gatioWsMethodPing       = "ping"
)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if g.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(g.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	g.WebsocketConn, _, err = dialer.Dial(g.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return err
	}

	go g.WsHandleData()

	return g.WsSubscribe()
}

// WsSubscribe subscribes to the full websocket suite on ZB exchange
func (g *Gateio) WsSubscribe() error {
	enabled := g.GetEnabledPairs(assets.AssetTypeSpot)

	for _, c := range enabled {
		ticker := WebsocketRequest{
			ID:     1337,
			Method: "ticker.subscribe",
			Params: []interface{}{c.String()},
		}

		err := g.WebsocketConn.WriteJSON(ticker)
		if err != nil {
			return err
		}

		trade := WebsocketRequest{
			ID:     1337,
			Method: "trades.subscribe",
			Params: []interface{}{c.String()},
		}

		err = g.WebsocketConn.WriteJSON(trade)
		if err != nil {
			return err
		}

		depth := WebsocketRequest{
			ID:     1337,
			Method: "depth.subscribe",
			Params: []interface{}{c.String(), 30, "0.1"},
		}

		err = g.WebsocketConn.WriteJSON(depth)
		if err != nil {
			return err
		}

		kline := WebsocketRequest{
			ID:     1337,
			Method: "kline.subscribe",
			Params: []interface{}{c.String(), 1800},
		}

		err = g.WebsocketConn.WriteJSON(kline)
		if err != nil {
			return err
		}
	}

	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gateio) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := g.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	g.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

// WsHandleData handles all the websocket data coming from the websocket
// connection
func (g *Gateio) WsHandleData() {
	g.Websocket.Wg.Add(1)

	defer func() {
		err := g.WebsocketConn.Close()
		if err != nil {
			g.Websocket.DataHandler <- fmt.Errorf("gateio_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		g.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-g.Websocket.ShutdownC:
			return

		default:
			resp, err := g.WsReadData()
			if err != nil {
				g.Websocket.DataHandler <- err
				continue
			}

			var result WebsocketResponse
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				g.Websocket.DataHandler <- err
				continue
			}

			if result.Error.Code != 0 {
				g.Websocket.DataHandler <- fmt.Errorf("gateio_websocket.go error %s",
					result.Error.Message)
				continue
			}

			switch {
			case common.StringContains(result.Method, "ticker"):
				var ticker WebsocketTicker
				var c string
				err = common.JSONDecode(result.Params[1], &ticker)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				g.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairFromString(c),
					AssetType:  assets.AssetTypeSpot,
					Exchange:   g.GetName(),
					ClosePrice: ticker.Close,
					Quantity:   ticker.BaseVolume,
					OpenPrice:  ticker.Open,
					HighPrice:  ticker.High,
					LowPrice:   ticker.Low,
				}

			case common.StringContains(result.Method, "trades"):
				var trades []WebsocketTrade
				var c string
				err = common.JSONDecode(result.Params[1], &trades)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				for _, trade := range trades {
					g.Websocket.DataHandler <- exchange.TradeData{
						Timestamp:    time.Now(),
						CurrencyPair: currency.NewPairFromString(c),
						AssetType:    assets.AssetTypeSpot,
						Exchange:     g.GetName(),
						Price:        trade.Price,
						Amount:       trade.Amount,
						Side:         trade.Type,
					}
				}

			case common.StringContains(result.Method, "depth"):
				var IsSnapshot bool
				var c string
				var data = make(map[string][][]string)
				err = common.JSONDecode(result.Params[0], &IsSnapshot)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[2], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[1], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				var asks, bids []orderbook.Item

				askData, askOk := data["asks"]
				for _, ask := range askData {
					amount, _ := strconv.ParseFloat(ask[1], 64)
					price, _ := strconv.ParseFloat(ask[0], 64)
					asks = append(asks, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}

				bidData, bidOk := data["bids"]
				for _, bid := range bidData {
					amount, _ := strconv.ParseFloat(bid[1], 64)
					price, _ := strconv.ParseFloat(bid[0], 64)
					bids = append(bids, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}

				if !askOk && !bidOk {
					g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access ask or bid data")
				}

				if IsSnapshot {
					if !askOk {
						g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access ask data")
					}

					if !bidOk {
						g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access bid data")
					}

					var newOrderBook orderbook.Base
					newOrderBook.Asks = asks
					newOrderBook.Bids = bids
					newOrderBook.AssetType = assets.AssetTypeSpot
					newOrderBook.LastUpdated = time.Now()
					newOrderBook.Pair = currency.NewPairFromString(c)

					err = g.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
						g.GetName(),
						false)
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				} else {
					err = g.Websocket.Orderbook.Update(asks,
						bids,
						currency.NewPairFromString(c),
						time.Now(),
						g.GetName(),
						assets.AssetTypeSpot)
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				}

				g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Pair:     currency.NewPairFromString(c),
					Asset:    assets.AssetTypeSpot,
					Exchange: g.GetName(),
				}

			case common.StringContains(result.Method, "kline"):
				var data []interface{}
				err = common.JSONDecode(result.Params[0], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				open, _ := strconv.ParseFloat(data[1].(string), 64)
				closePrice, _ := strconv.ParseFloat(data[2].(string), 64)
				high, _ := strconv.ParseFloat(data[3].(string), 64)
				low, _ := strconv.ParseFloat(data[4].(string), 64)
				volume, _ := strconv.ParseFloat(data[5].(string), 64)

				g.Websocket.DataHandler <- exchange.KlineData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairFromString(data[7].(string)),
					AssetType:  assets.AssetTypeSpot,
					Exchange:   g.GetName(),
					OpenPrice:  open,
					ClosePrice: closePrice,
					HighPrice:  high,
					LowPrice:   low,
					Volume:     volume,
				}
			}
		}
	}
}
