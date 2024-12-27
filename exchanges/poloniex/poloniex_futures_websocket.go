package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	futuresWebsocketPrivateURL = "wss://ws.poloniex.com/ws/v3/private"
	futuresWebsocketPublicURL  = "wss://ws.poloniex.com/ws/v3/public"
)

const (
	cnlFuturesSymbol           = "symbol"
	cnlFuturesOrderbookLvl2    = "book_lv2"
	cnlFuturesOrderbook        = "book"
	cnlFuturesCandles          = "candles"
	cnlFuturesTickers          = "tickers"
	cnlFuturesTrades           = "trades"
	cnlFuturesIndexPrice       = "index_price"
	cnlFuturesMarkPrice        = "mark_price"
	cnlFuturesMarkPriceCandles = "mark_price_candles"
	cnlFuturesIndexCandles     = "index_candles"
	cnlFuturesFundingRate      = "funding_rate"

	cnlFuturesPrivatePositions = "positions"
	cnlFuturesPrivateOrders    = "orders"
	cnlFuturesPrivateTrades    = "trade"
	cnlFuturesAccount          = "account"
)

var defaultFuturesChannels = []string{
	cnlFuturesTickers,
	cnlFuturesOrderbook,
	cnlFuturesCandles,
}

// WsFuturesConnect establishes a websocket connection to the futures websocket server.
func (p *Poloniex) WsFuturesConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := p.Websocket.SetWebsocketURL(futuresWebsocketPublicURL, false, false)
	if err != nil {
		return err
	}
	err = p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	p.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		Delay:       time.Second * 15,
		Message:     []byte(`{"type":"ping"}`),
		MessageType: websocket.TextMessage,
	})
	if p.Websocket.CanUseAuthenticatedEndpoints() {
		err = p.AuthConnect()
		if err != nil {
			p.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	p.Websocket.Wg.Add(1)
	go p.wsFuturesReadData(p.Websocket.Conn)
	return nil
}

// AuthConnect establishes a websocket and authenticates to futures private websocket
func (p *Poloniex) AuthConnect() error {
	creds, err := p.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = p.Websocket.SetWebsocketURL(futuresWebsocketPrivateURL, false, false)
	if err != nil {
		return err
	}
	err = p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	p.Websocket.AuthConn.SetupPingHandler(request.Unset, stream.PingHandler{
		Delay:       time.Second * 15,
		Message:     []byte(`{"type":"ping"}`),
		MessageType: websocket.TextMessage,
	})
	timestamp := time.Now().UnixMilli()
	signatureStrings := "GET\n/ws\nsignTimestamp=" + strconv.FormatInt(timestamp, 10)

	var hmac []byte
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signatureStrings),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	data, err := p.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, "auth", &SubscriptionPayload{
		Event:   "subscribe",
		Channel: []string{"auth"},
		Params: map[string]any{"key": creds.Key,
			"signTimestamp": timestamp,
			"signature":     crypto.Base64Encode(hmac),
		},
	})
	if err != nil {
		return err
	}
	var resp *AuthenticationResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if !resp.Data.Success {
		return fmt.Errorf("authentication failed with status code: %s", resp.Data.Message)
	}
	p.Websocket.Wg.Add(1)
	go p.wsFuturesReadData(p.Websocket.AuthConn)
	return nil
}

// wsFuturesReadData handles data from the websocket connection for futures instruments subscriptions.
func (p *Poloniex) wsFuturesReadData(conn stream.Connection) {
	defer p.Websocket.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := p.wsFuturesHandleData(resp.Raw)
		if err != nil {
			p.Websocket.DataHandler <- fmt.Errorf("%s: %w", p.Name, err)
		}
	}
}

func (p *Poloniex) wsFuturesHandleData(respRaw []byte) error {
	var result *FuturesSubscriptionResp
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result.Channel {
	case "auth":
		if !p.Websocket.Match.IncomingWithData("auth", respRaw) {
			return fmt.Errorf("could not match data with %s %s", "auth", respRaw)
		}
		return nil
	case cnlFuturesSymbol:
		return nil
	case cnlFuturesOrderbookLvl2:
		return nil
	case cnlFuturesOrderbook:
		return nil
	case cnlFuturesCandles:
		return nil
	case cnlFuturesTickers:
		return nil
	case cnlFuturesTrades:
		return nil
	case cnlFuturesIndexPrice:
		return nil
	case cnlFuturesMarkPrice:
		return nil
	case cnlFuturesMarkPriceCandles:
		return nil
	case cnlFuturesIndexCandles:
		return nil
	case cnlFuturesFundingRate:
		return nil
	case cnlFuturesPrivatePositions:
		return nil
	case cnlFuturesPrivateOrders:
		return nil
	case cnlFuturesPrivateTrades:
		return nil
	case cnlFuturesAccount:
		return nil
	default:
		p.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: p.Name + stream.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", p.Name, string(respRaw))
	}
}

// ------------------------------------------------------------------------------------------------

// GenerateFuturesDefaultSubscriptions adds default subscriptions to futures websockets.
func (p *Poloniex) GenerateFuturesDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	channels := defaultFuturesChannels
	subscriptions := subscription.List{}
	for i := range channels {
		switch channels[i] {
		case cnlFuturesPrivatePositions,
			cnlFuturesPrivateOrders,
			cnlFuturesPrivateTrades,
			cnlFuturesAccount:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:       channels[i],
				Asset:         asset.Futures,
				Authenticated: true,
			})
		case cnlFuturesSymbol,
			cnlFuturesOrderbookLvl2,
			cnlFuturesOrderbook,
			cnlFuturesCandles,
			cnlFuturesTickers,
			cnlFuturesTrades,
			cnlFuturesIndexPrice,
			cnlFuturesMarkPrice,
			cnlFuturesMarkPriceCandles,
			cnlFuturesIndexCandles,
			cnlFuturesFundingRate:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
				Asset:   asset.Futures,
				Pairs:   enabledPairs,
			})
		}
	}
	return subscriptions, nil
}

func (p *Poloniex) handleFuturesSubscriptions(operation string, subscs subscription.List) []FuturesSubscriptionInput {
	payloads := []FuturesSubscriptionInput{}
	for x := range subscs {
		if len(subscs[x].Pairs) == 0 {
			input := FuturesSubscriptionInput{
				ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
				Type:  operation,
				Topic: subscs[x].Channel,
			}
			payloads = append(payloads, input)
		} else {
			for i := range subscs[x].Pairs {
				input := FuturesSubscriptionInput{
					ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
					Type:  operation,
					Topic: subscs[x].Channel,
				}
				if !subscs[x].Pairs[x].IsEmpty() {
					input.Topic += ":" + subscs[x].Pairs[i].String()
				}
				payloads = append(payloads, input)
			}
		}
	}
	return payloads
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (p *Poloniex) SubscribeFutures(subs subscription.List) error {
	payloads := p.handleFuturesSubscriptions("subscribe", subs)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.AddSuccessfulSubscriptions(p.Websocket.Conn, subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (p *Poloniex) UnsubscribeFutures(unsub subscription.List) error {
	payloads := p.handleFuturesSubscriptions("unsubscribe", unsub)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.RemoveSubscriptions(p.Websocket.Conn, unsub...)
}
