package okx

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// okxBusinessWebsocketURL
	okxBusinessWebsocketURL = "wss://ws.okx.com:8443/ws/v5/business"
)

var (
	// defaultBusinessSubscribedChannels list of channels which are subscribed by default
	defaultBusinessSubscribedChannels = []string{
		okxSpreadPublicTrades,
		okxSpreadOrderbook,
		okxSpreadPublicTicker,

		channelPublicStrucBlockTrades,
		channelPublicBlockTrades,
		channelBlockTickers,
	}

	// defaultBusinessAuthChannels list of authenticated channels
	defaultBusinessAuthChannels = []string{
		okxSpreadOrders,
		okxSpreadTrades,
	}
)

// WsConnectBusiness connects to a business websocket channel.
func (e *Exchange) WsConnectBusiness(ctx context.Context) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	e.Websocket.Conn.SetURL(okxBusinessWebsocketURL)
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx, e.Websocket.Conn)
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			e.Websocket.GetWebsocketURL())
	}
	e.Websocket.Conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		err = e.WsSpreadAuth(ctx)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsSpreadAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (e *Exchange) WsSpreadAuth(ctx context.Context) error {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", e.Name)
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ts := time.Now().Unix()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(ts, 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	args := []WebsocketLoginData{
		{
			APIKey:     creds.Key,
			Passphrase: creds.ClientID,
			Timestamp:  ts,
			Sign:       base64.StdEncoding.EncodeToString(hmac),
		},
	}
	return e.SendAuthenticatedWebsocketRequest(ctx, request.Unset, "login-response", operationLogin, args, nil)
}

// GenerateDefaultBusinessSubscriptions returns a list of default subscriptions to business websocket.
func (e *Exchange) GenerateDefaultBusinessSubscriptions() ([]subscription.Subscription, error) {
	var subs []string
	var subscriptions []subscription.Subscription
	subs = append(subs, defaultBusinessSubscribedChannels...)
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		subs = append(subs, defaultBusinessAuthChannels...)
	}
	for c := range subs {
		switch subs[c] {
		case okxSpreadOrders,
			okxSpreadTrades,
			okxSpreadOrderbookLevel1,
			okxSpreadOrderbook,
			okxSpreadPublicTrades,
			okxSpreadPublicTicker:
			pairs, err := e.GetEnabledPairs(asset.Spread)
			if err != nil {
				return nil, err
			}
			for p := range pairs {
				subscriptions = append(subscriptions, subscription.Subscription{
					Channel: subs[c],
					Asset:   asset.Spread,
					Pairs:   []currency.Pair{pairs[p]},
				})
			}
		case channelPublicBlockTrades,
			channelBlockTickers:
			pairs, err := e.GetEnabledPairs(asset.PerpetualSwap)
			if err != nil {
				return nil, err
			}
			for p := range pairs {
				subscriptions = append(subscriptions, subscription.Subscription{
					Channel: subs[c],
					Asset:   asset.PerpetualSwap,
					Pairs:   []currency.Pair{pairs[p]},
				})
			}
		default:
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: subs[c],
			})
		}
	}
	return subscriptions, nil
}

// BusinessSubscribe sends a websocket subscription request to several channels to receive data.
func (e *Exchange) BusinessSubscribe(ctx context.Context, channelsToSubscribe subscription.List) error {
	return e.handleBusinessSubscription(ctx, operationSubscribe, channelsToSubscribe)
}

// BusinessUnsubscribe sends a websocket unsubscription request to several channels to receive data.
func (e *Exchange) BusinessUnsubscribe(ctx context.Context, channelsToUnsubscribe subscription.List) error {
	return e.handleBusinessSubscription(ctx, operationUnsubscribe, channelsToUnsubscribe)
}

// handleBusinessSubscription sends a subscription and unsubscription information thought the business websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (e *Exchange) handleBusinessSubscription(ctx context.Context, operation string, subscriptions subscription.List) error {
	wsSubscriptionReq := WSSubscriptionInformationList{Operation: operation}
	var channels subscription.List
	var authChannels subscription.List
	var err error
	for i := 0; i < len(subscriptions); i++ {
		arg := SubscriptionInfo{
			Channel: subscriptions[i].Channel,
		}

		switch arg.Channel {
		case okxSpreadOrders, okxSpreadTrades, okxSpreadOrderbookLevel1, okxSpreadOrderbook, okxSpreadPublicTrades, okxSpreadPublicTicker:
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.SpreadID = subscriptions[i].Pairs[0].String()
		case channelPublicBlockTrades, channelBlockTickers:
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.InstrumentID = subscriptions[i].Pairs[0]
		}

		if strings.HasPrefix(arg.Channel, candle) || strings.HasPrefix(arg.Channel, indexCandlestick) || strings.HasPrefix(arg.Channel, markPrice) {
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.InstrumentID = subscriptions[i].Pairs[0]
		}

		if ifAny, ok := subscriptions[i].Params["instFamily"]; ok {
			if arg.InstrumentFamily, ok = ifAny.(string); !ok {
				return common.GetTypeAssertError("string", ifAny, "instFamily")
			}
		}

		var chunk []byte
		channels = append(channels, subscriptions[i])
		wsSubscriptionReq.Arguments = append(wsSubscriptionReq.Arguments, arg)
		chunk, err = json.Marshal(wsSubscriptionReq)
		if err != nil {
			return err
		}
		if len(chunk) > maxConnByteLen {
			i--
			err = e.Websocket.Conn.SendJSONMessage(ctx, request.UnAuth, wsSubscriptionReq)
			if err != nil {
				return err
			}
			if operation == operationUnsubscribe {
				err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, channels...)
			} else {
				err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, channels...)
			}
			if err != nil {
				return err
			}
			channels = subscription.List{}
			wsSubscriptionReq.Arguments = []SubscriptionInfo{}
			continue
		}
	}
	err = e.Websocket.Conn.SendJSONMessage(ctx, request.UnAuth, wsSubscriptionReq)
	if err != nil {
		return err
	}

	if operation == operationUnsubscribe {
		channels = append(channels, authChannels...)
		err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, channels...)
	} else {
		channels = append(channels, authChannels...)
		err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, channels...)
	}
	return err
}
