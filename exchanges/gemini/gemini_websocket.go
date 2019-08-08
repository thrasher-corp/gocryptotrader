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
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/connection"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/monitor"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/ob"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
var responseMaxLimit time.Duration
var responseCheckTimeout time.Duration

// WsConnect initiates a websocket connection
func (g *Gemini) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(monitor.WebsocketNotEnabled)
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
	err := g.WsSecureSubscribe(&dialer, geminiWsOrderEvents)
	if err != nil {
		log.Errorf("%v - authentication failed: %v", g.Name, err)
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
		connection := &connection.WebsocketConnection{
			ExchangeName:         g.Name,
			URL:                  endpoint,
			Verbose:              g.Verbose,
			ResponseCheckTimeout: responseCheckTimeout,
			ResponseMaxLimit:     responseMaxLimit,
		}
		err := connection.Dial(dialer, http.Header{})
		if err != nil {
			return fmt.Errorf("%v Websocket connection %v error. Error %v", g.Name, endpoint, err)
		}
		go g.WsReadData(connection, c)
		if len(enabledCurrencies)-1 == i {
			return nil
		}
	}
	return nil
}

// WsSecureSubscribe will connect to Gemini's secure endpoint
func (g *Gemini) WsSecureSubscribe(dialer *websocket.Dialer, url string) error {
	if !g.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", g.Name)
	}
	payload := WsRequestPayload{
		Request: fmt.Sprintf("/v1/%v", url),
		Nonce:   time.Now().UnixNano(),
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

	g.AuthenticatedWebsocketConn = &connection.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  endpoint,
		Verbose:              g.Verbose,
		ResponseCheckTimeout: responseCheckTimeout,
		ResponseMaxLimit:     responseMaxLimit,
	}
	err = g.AuthenticatedWebsocketConn.Dial(dialer, headers)
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", g.Name, endpoint, err)
	}
	go g.WsReadData(g.AuthenticatedWebsocketConn, currency.Pair{})
	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gemini) WsReadData(ws *connection.WebsocketConnection, c currency.Pair) {
	g.Websocket.Wg.Add(1)
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.ShutdownC:
			return
		default:
			resp, err := ws.ReadMessage()
			if err != nil {
				g.Websocket.DataHandler <- err
				return
			}
			g.Websocket.TrafficAlert <- struct{}{}
			comms <- ReadData{Raw: resp.Raw, Currency: c}
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
					g.Websocket.DataHandler <- fmt.Errorf("%v - unhandled data %s",
						g.Name, resp.Raw)
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
				g.Websocket.DataHandler <- fmt.Errorf("%v - unhandled data %s",
					g.Name, resp.Raw)
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
		newOrderBook.AssetType = orderbook.Spot
		newOrderBook.Pair = pair
		err := g.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
			g.GetName(),
			false)
		if err != nil {
			g.Websocket.DataHandler <- err
			return
		}
		g.Websocket.DataHandler <- monitor.WebsocketOrderbookUpdate{Pair: pair,
			Asset:    orderbook.Spot,
			Exchange: g.GetName()}
	} else {
		var asks, bids []orderbook.Item
		for i := 0; i < len(result.Events); i++ {
			if result.Events[i].Type == "trade" {
				g.Websocket.DataHandler <- monitor.TradeData{
					Timestamp:    time.Now(),
					CurrencyPair: pair,
					AssetType:    orderbook.Spot,
					Exchange:     g.Name,
					EventTime:    result.Timestamp,
					Price:        result.Events[i].Price,
					Amount:       result.Events[i].Amount,
					Side:         result.Events[i].MakerSide,
				}
			} else {
				item := orderbook.Item{
					Amount: result.Events[i].Remaining,
					Price:  result.Events[i].Price,
				}
				if result.Events[i].Side == "ask" {
					asks = append(asks, item)
				} else {
					bids = append(bids, item)
				}
			}
		}
		err := g.Websocket.Orderbook.Update(&ob.WebsocketOrderbookUpdate{
			Asks:          asks,
			Bids:          bids,
			CurrencyPair:  pair,
			UpdateTime:    time.Unix(0, result.TimestampMS),
			ExchangeName:  g.Name,
			AssetType:     orderbook.Spot,
			SortBuffer:    true,
			BufferEnabled: true,
		})
		if err != nil {
			g.Websocket.DataHandler <- fmt.Errorf("%v %v", g.Name, err)
		}
		g.Websocket.DataHandler <- monitor.WebsocketOrderbookUpdate{Pair: pair,
			Asset:    orderbook.Spot,
			Exchange: g.GetName()}
	}
}
