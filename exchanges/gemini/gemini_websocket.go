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
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	geminiWebsocketEndpoint        = "wss://api.gemini.com/v1/"
	geminiWebsocketSandboxEndpoint = "wss://api.sandbox.gemini.com/v1/"
	geminiWsEvent                  = "event"
	geminiWsMarketData             = "marketdata"
	geminiWsOrderEvents            = "order/events"
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
	if g.AuthenticatedAPISupport {
		err := g.WsSecureSubscribe(&dialer, geminiWsOrderEvents)
		if err != nil {
			return err
		}
	}
	return g.WsSubscribe(&dialer)
}

// WsSubscribe subscribes to the full websocket suite on gemini exchange
func (g *Gemini) WsSubscribe(dialer *websocket.Dialer) error {
	enabledCurrencies := g.GetEnabledCurrencies()
	for i, c := range enabledCurrencies {
		val := url.Values{}
		val.Set("heartbeat", "true")
		endpoint := fmt.Sprintf("%s%s/%s?%s",
			g.WebsocketURL,
			geminiWsMarketData,
			c.String(),
			val.Encode())
		conn, conStatus, err := dialer.Dial(endpoint, http.Header{})
		if err != nil {
			return fmt.Errorf("%v %v %v Error: %v", endpoint, conStatus, conStatus.StatusCode, err)
		}
		go g.WsReadData(conn, c)
		if len(enabledCurrencies)-1 == i {
			return nil
		}
		time.Sleep(time.Minute) // rate limiter, limit of 1 requests per
		// minute
	}
	return nil
}

// WsSecureSubscribe will connect to Gemini's secure endpoint
func (g *Gemini) WsSecureSubscribe(dialer *websocket.Dialer, url string) error {
	payload := WsRequestPayload{
		Request: fmt.Sprintf("/v1/%v", url),
		Nonce:   int64(g.GetNonce(false)),
	}
	PayloadJSON, err := common.JSONEncode(payload)
	if err != nil {
		return fmt.Errorf("%v sendAuthenticatedHTTPRequest: Unable to JSON request", g.Name)
	}

	endpoint := fmt.Sprintf("%v%v", g.WebsocketURL, url)
	PayloadBase64 := common.Base64Encode(PayloadJSON)
	hmac := common.GetHMAC(common.HashSHA512_384, []byte(PayloadBase64), []byte(g.APISecret))
	headers := http.Header{}
	headers.Add("Content-Length", "0")
	headers.Add("Content-Type", "text/plain")
	headers.Add("X-GEMINI-PAYLOAD", PayloadBase64)
	headers.Add("X-GEMINI-APIKEY", g.APIKey)
	headers.Add("X-GEMINI-SIGNATURE", common.HexEncodeToString(hmac))
	headers.Add("Cache-Control", "no-cache")

	conn, conStatus, err := dialer.Dial(endpoint, headers)
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", endpoint, conStatus, conStatus.StatusCode, err)
	}
	go g.WsReadData(conn, currency.Pair{})
	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gemini) WsReadData(ws *websocket.Conn, c currency.Pair) {
	g.Websocket.Wg.Add(1)
	defer g.Websocket.Wg.Done()
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
			comms <- ReadData{Raw: resp, Currency: c}
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
			// Gemini likes to send empty arrays
			if string(resp.Raw) == "[]" {
				continue
			}
			var result map[string]interface{}
			err := common.JSONDecode(resp.Raw, &result)
			if err != nil {
				g.Websocket.DataHandler <- fmt.Errorf("%v Error: %v, Raw: %v", g.Name, err, string(resp.Raw))
				continue
			}
			if g.Verbose {
				log.Debugf("MESSAGE: %v", string(resp.Raw))
			}
			switch result["type"] {
			case "subscription_ack":
				var result WsSubscriptionAcknowledgementResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "initial":
				var result WsSubscriptionAcknowledgementResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "accepted":
				var result WsActiveOrdersResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "booked":
				var result WsOrderBookedResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "fill":
				var result WsOrderFilledResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "cancelled":
				var result WsOrderCancelledResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "closed":
				var result WsOrderClosedResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "heartbeat":
				var result WsHeartbeatResponse
				err := common.JSONDecode(resp.Raw, &result)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- result
			case "update":
				if resp.Currency.IsEmpty() {
					g.Websocket.DataHandler <- fmt.Errorf("gemini_websocket.go - unhandled data %s",
						resp.Raw)
					continue
				}
				var marketUpdate WsMarketUpdateResponse
				err := common.JSONDecode(resp.Raw, &marketUpdate)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.wsProcessUpdate(marketUpdate, resp.Currency)
			default:
				g.Websocket.DataHandler <- fmt.Errorf("gemini_websocket.go - unhandled data %s",
					resp.Raw)
			}
		}
	}
}

// wsProcessUpdate handles order book data
func (g *Gemini) wsProcessUpdate(result WsMarketUpdateResponse, pair currency.Pair) {
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
		newOrderBook.AssetType = "SPOT"
		newOrderBook.Pair = pair

		err := g.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
			g.GetName(),
			false)
		if err != nil {
			g.Websocket.DataHandler <- err
			return
		}

		g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{Pair: pair,
			Asset:    "SPOT",
			Exchange: g.GetName()}
	} else {
		for _, event := range result.Events {
			if event.Type == "trade" {
				g.Websocket.DataHandler <- exchange.TradeData{
					Timestamp:    time.Now(),
					CurrencyPair: pair,
					AssetType:    "SPOT",
					Exchange:     g.Name,
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
						pair,
						time.Now(),
						g.GetName(),
						"SPOT")
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				} else {
					err := g.Websocket.Orderbook.Update([]orderbook.Item{i},
						nil,
						pair,
						time.Now(),
						g.GetName(),
						"SPOT")
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				}
			}
		}

		g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{Pair: pair,
			Asset:    "SPOT",
			Exchange: g.GetName()}
	}
}
