// Package gemini exchange documentation can be found at
// https://docs.sandbox.gemini.com
package gemini

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
	geminiWebsocketEndpoint = "wss://api.gemini.com/v1/marketdata/%s?%s"
	geminiWsEvent           = "event"
	geminiWsMarketData      = "marketdata"
)

// Instantiates a communications channel between websocket connections
var comms = make(chan ReadData, 1)

// WsConnect initiates a websocket connection
func (g *Gemini) WsConnect() error {
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

	go g.WsHandleData()

	return g.WsSubscribe(&dialer)
}

// WsSubscribe subscribes to the full websocket suite on gemini exchange
func (g *Gemini) WsSubscribe(dialer *websocket.Dialer) error {
	enabledCurrencies := g.GetEnabledPairs(assets.AssetTypeSpot)
	for i, c := range enabledCurrencies {
		val := url.Values{}
		val.Set("heartbeat", "true")

		endpoint := fmt.Sprintf(g.Websocket.GetWebsocketURL(),
			c.String(),
			val.Encode())

		conn, _, err := dialer.Dial(endpoint, http.Header{})
		if err != nil {
			return err
		}

		go g.WsReadData(conn, c, geminiWsMarketData)

		if len(enabledCurrencies)-1 == i {
			return nil
		}

		time.Sleep(5 * time.Second) // rate limiter, limit of 12 requests per
		// minute
	}
	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gemini) WsReadData(ws *websocket.Conn, c currency.Pair, feedType string) {
	g.Websocket.Wg.Add(1)

	defer func() {
		err := ws.Close()
		if err != nil {
			g.Websocket.DataHandler <- fmt.Errorf("gemini_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		g.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-g.Websocket.ShutdownC:
			return

		default:
			_, resp, err := ws.ReadMessage()
			if err != nil {
				g.Websocket.DataHandler <- err
				return
			}

			g.Websocket.TrafficAlert <- struct{}{}
			comms <- ReadData{Raw: resp, Currency: c, FeedType: feedType}
		}
	}

}

// WsHandleData handles all the websocket data coming from the websocket
// connection
func (g *Gemini) WsHandleData() {
	g.Websocket.Wg.Add(1)
	defer g.Websocket.Wg.Done()

	for {
		select {
		case <-g.Websocket.ShutdownC:
			return

		case resp := <-comms:
			switch resp.FeedType {
			case geminiWsEvent:

			case geminiWsMarketData:
				var result Response
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				switch result.Type {
				case "update":
					if result.Timestamp == 0 && result.TimestampMS == 0 {
						var bids, asks []orderbook.Item
						for _, event := range result.Events {
							if event.Reason != "initial" {
								g.Websocket.DataHandler <- errors.New("gemini_websocket.go orderbook should be snapshot only")
								continue
							}

							if event.Side == "ask" {
								asks = append(asks, orderbook.Item{
									Amount: event.Remaining,
									Price:  event.Price,
								})
							} else {
								bids = append(bids, orderbook.Item{
									Amount: event.Remaining,
									Price:  event.Price,
								})
							}
						}

						var newOrderBook orderbook.Base
						newOrderBook.Asks = asks
						newOrderBook.Bids = bids
						newOrderBook.AssetType = assets.AssetTypeSpot
						newOrderBook.LastUpdated = time.Now()
						newOrderBook.Pair = resp.Currency

						err := g.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
							g.GetName(),
							false)
						if err != nil {
							g.Websocket.DataHandler <- err
							break
						}

						g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{Pair: resp.Currency,
							Asset:    assets.AssetTypeSpot,
							Exchange: g.GetName()}

					} else {
						for _, event := range result.Events {
							if event.Type == "trade" {
								g.Websocket.DataHandler <- exchange.TradeData{
									Timestamp:    time.Now(),
									CurrencyPair: resp.Currency,
									AssetType:    assets.AssetTypeSpot,
									Exchange:     g.GetName(),
									EventTime:    result.Timestamp,
									Price:        event.Price,
									Amount:       event.Amount,
									Side:         event.MakerSide,
								}

							} else {
								var i orderbook.Item
								i.Amount = event.Remaining
								i.Price = event.Price
								if event.Side == "ask" {
									err := g.Websocket.Orderbook.Update(nil,
										[]orderbook.Item{i},
										resp.Currency,
										time.Now(),
										g.GetName(),
										assets.AssetTypeSpot)
									if err != nil {
										g.Websocket.DataHandler <- err
									}
								} else {
									err := g.Websocket.Orderbook.Update([]orderbook.Item{i},
										nil,
										resp.Currency,
										time.Now(),
										g.GetName(),
										assets.AssetTypeSpot)
									if err != nil {
										g.Websocket.DataHandler <- err
									}
								}
							}
						}

						g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{Pair: resp.Currency,
							Asset:    assets.AssetTypeSpot,
							Exchange: g.GetName()}
					}

				case "heartbeat":

				default:
					g.Websocket.DataHandler <- fmt.Errorf("gemini_websocket.go - unhandled data %s",
						resp.Raw)
				}
			}
		}
	}
}
