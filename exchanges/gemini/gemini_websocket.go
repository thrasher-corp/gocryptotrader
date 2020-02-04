// Package gemini exchange documentation can be found at
// https://docs.sandbox.gemini.com
package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	geminiWebsocketEndpoint        = "wss://api.gemini.com/v1/"
	geminiWebsocketSandboxEndpoint = "wss://api.sandbox.gemini.com/v1/"
	geminiWsEvent                  = "event"
	geminiWsMarketData             = "marketdata"
	geminiWsOrderEvents            = "order/events"
)

// Instantiates a communications channel between websocket connections
var comms = make(chan ReadData)
var responseMaxLimit time.Duration
var responseCheckTimeout time.Duration

// WsConnect initiates a websocket connection
func (g *Gemini) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if g.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(g.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	go g.wsReadData()
	err := g.WsSecureSubscribe(&dialer, geminiWsOrderEvents)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", g.Name, err)
	}
	return g.WsSubscribe(&dialer)
}

// WsSubscribe subscribes to the full websocket suite on gemini exchange
func (g *Gemini) WsSubscribe(dialer *websocket.Dialer) error {
	enabledCurrencies := g.GetEnabledPairs(asset.Spot)
	for i := range enabledCurrencies {
		val := url.Values{}
		val.Set("heartbeat", "true")
		endpoint := fmt.Sprintf("%s%s/%s?%s",
			g.API.Endpoints.WebsocketURL,
			geminiWsMarketData,
			enabledCurrencies[i].String(),
			val.Encode())
		connection := &wshandler.WebsocketConnection{
			ExchangeName:         g.Name,
			URL:                  endpoint,
			Verbose:              g.Verbose,
			ResponseCheckTimeout: responseCheckTimeout,
			ResponseMaxLimit:     responseMaxLimit,
		}
		err := connection.Dial(dialer, http.Header{})
		if err != nil {
			return fmt.Errorf("%v Websocket connection %v error. Error %v",
				g.Name, endpoint, err)
		}
		go g.wsFunnelConnectionData(connection, enabledCurrencies[i])
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
	PayloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%v sendAuthenticatedHTTPRequest: Unable to JSON request", g.Name)
	}

	endpoint := g.API.Endpoints.WebsocketURL + url
	PayloadBase64 := crypto.Base64Encode(PayloadJSON)
	hmac := crypto.GetHMAC(crypto.HashSHA512_384, []byte(PayloadBase64), []byte(g.API.Credentials.Secret))
	headers := http.Header{}
	headers.Add("Content-Length", "0")
	headers.Add("Content-Type", "text/plain")
	headers.Add("X-GEMINI-PAYLOAD", PayloadBase64)
	headers.Add("X-GEMINI-APIKEY", g.API.Credentials.Key)
	headers.Add("X-GEMINI-SIGNATURE", crypto.HexEncodeToString(hmac))
	headers.Add("Cache-Control", "no-cache")

	g.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
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
	go g.wsFunnelConnectionData(g.AuthenticatedWebsocketConn, currency.Pair{})
	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gemini) wsFunnelConnectionData(ws *wshandler.WebsocketConnection, c currency.Pair) {
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

// wsReadData handles all the websocket data coming from the websocket
// connection
func (g *Gemini) wsReadData() {
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
			err := g.wsHandleData(resp.Raw, resp.Currency)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

func (g *Gemini) wsHandleData(respRaw []byte, curr currency.Pair) error {
	// only order details are sent in arrays
	if strings.HasPrefix(string(respRaw), "[") {
		var result []WsOrderResponse
		err := json.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}

		for i := range result {
			oSide, err := order.StringToOrderSide(result[i].Side)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
			g.Websocket.DataHandler <- &order.Detail{
				HiddenOrder:     result[i].IsHidden,
				Price:           result[i].Price,
				Amount:          result[i].OriginalAmount,
				ExecutedAmount:  result[i].ExecutedAmount,
				RemainingAmount: result[i].RemainingAmount,
				Exchange:        g.Name,
				ID:              result[i].OrderID,
				Type:            responseToOrderType(result[i].OrderType),
				Side:            oSide,
				Status:          responseToStatus(result[i].Type),
				AssetType:       asset.Spot,
				Date:            time.Unix(0, result[i].Timestampms*int64(time.Millisecond)),
				Pair:            currency.NewPairFromString(result[i].Symbol),
			}
		}
		return nil
	}
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return fmt.Errorf("%v Error: %v, Raw: %v", g.Name, err, string(respRaw))
	}
	if _, ok := result["type"]; ok {
		switch result["type"] {
		case "subscription_ack":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
		case "unsubscribe":
			var result wsUnsubscribeResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
		case "initial":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
		case "heartbeat":
			var result WsHeartbeatResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
		case "update":
			if curr.IsEmpty() {
				return fmt.Errorf("%v - unhandled data %s",
					g.Name, respRaw)
			}
			var marketUpdate WsMarketUpdateResponse
			err := json.Unmarshal(respRaw, &marketUpdate)
			if err != nil {
				return err
			}
			g.wsProcessUpdate(marketUpdate, curr)
		case "candles_1m_updates",
		"candles_5m_updates",
		"candles_15m_updates",
		"candles_30m_updates",
		"candles_1h_updates",
		"candles_6h_updates",
		"candles_1d_updates":
			var candle wsCandleResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			for i := range candle.Changes {
				g.Websocket.DataHandler <- wshandler.KlineData{
					Timestamp:  time.Unix(int64(candle.Changes[i][0])*1000, 0),
					Pair:       curr,
					AssetType:  asset.Spot,
					Exchange:   g.Name,
					Interval:   result["type"].(string),
					OpenPrice:  candle.Changes[i][1],
					ClosePrice: candle.Changes[i][4],
					HighPrice:  candle.Changes[i][2],
					LowPrice:   candle.Changes[i][3],
					Volume:     candle.Changes[i][5],
				}
			}

		default:
			return fmt.Errorf("%v Unhandled websocket message %s", g.Name, respRaw)
		}
	} else if _, ok := result["result"]; ok {
		switch result["result"].(string) {
		case "error":
			if _, ok := result["reason"]; ok {
				if _, ok := result["message"]; ok {
					return errors.New(result["reason"].(string) + " - " + result["message"].(string))
				}
			}
			return fmt.Errorf("%v Unhandled websocket message %s", g.Name, respRaw)
		default:
			return fmt.Errorf("%v Unhandled websocket message %s", g.Name, respRaw)
		}
	}
	return nil
}

func responseToStatus(response string) order.Status {
	switch response {
	case "accepted":
		return order.New
	case "booked":
		return order.Active
	case "filled":
		return order.Filled
	case "cancelled":
		return order.Cancelled
	case "cancel_rejected":
		return order.Rejected
	case "closed":
		return order.Filled
	default:
		return order.UnknownStatus
	}
}

func responseToOrderType(response string) order.Type {
	switch response {
	case "exchange limit", "auction-only limit", "indication-of-interest limit":
		return order.Limit
		// block trades are conducted off order-book, so their type is market, but would be considered a hidden trade
	case "market buy", "market sell", "block_trade":
		return order.Market
	default:
		return order.UnknownType
	}
}

// wsProcessUpdate handles order book data
func (g *Gemini) wsProcessUpdate(result WsMarketUpdateResponse, pair currency.Pair)  {
	if result.Timestamp == 0 && result.TimestampMS == 0 {
		var bids, asks []orderbook.Item
		for i := range result.Events {
			if result.Events[i].Reason != "initial" {
				g.Websocket.DataHandler <- errors.New("gemini_websocket.go orderbook should be snapshot only")
				continue
			}
			if result.Events[i].Side == "ask" {
				asks = append(asks, orderbook.Item{
					Amount: result.Events[i].Remaining,
					Price:  result.Events[i].Price,
				})
			} else {
				bids = append(bids, orderbook.Item{
					Amount: result.Events[i].Remaining,
					Price:  result.Events[i].Price,
				})
			}
		}
		var newOrderBook orderbook.Base
		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.AssetType = asset.Spot
		newOrderBook.Pair = pair
		newOrderBook.ExchangeName = g.Name
		err := g.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
		if err != nil {
			g.Websocket.DataHandler <- err
			return
		}
		g.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: pair,
			Asset:    asset.Spot,
			Exchange: g.Name}
	} else {
		var asks, bids []orderbook.Item
		for i := range result.Events {
			switch result.Events[i].Type {
			case "trade":
				g.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    time.Unix(0, result.Timestamp),
					CurrencyPair: pair,
					AssetType:    asset.Spot,
					Exchange:     g.Name,
					Price:        result.Events[i].Price,
					Amount:       result.Events[i].Amount,
					Side:         result.Events[i].MakerSide,
				}
			case "change":
				item := orderbook.Item{
					Amount: result.Events[i].Remaining,
					Price:  result.Events[i].Price,
				}
				if strings.EqualFold(result.Events[i].Side, order.Ask.String()) {
					asks = append(asks, item)
				} else {
					bids = append(bids, item)
				}
			default:
				g.Websocket.DataHandler <- fmt.Errorf("%s - Unhandled websocket update: %+v", g.Name, result)
			}
		}
		if len(asks) == 0 && len(bids) == 0 {
			return
		}
			err := g.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
				Asks:       asks,
				Bids:       bids,
				Pair:       pair,
				UpdateTime: time.Unix(0, result.TimestampMS),
				Asset:      asset.Spot,
			})
			if err != nil {
				g.Websocket.DataHandler <- fmt.Errorf("%v %v", g.Name, err)
			}
			g.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: pair,
				Asset:    asset.Spot,
				Exchange: g.Name}
	}
}
