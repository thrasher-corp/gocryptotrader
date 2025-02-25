package okx

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
func (ok *Okx) WsConnectBusiness() error {
	if !ok.Websocket.IsEnabled() || !ok.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	ok.Websocket.Conn.SetURL(okxBusinessWebsocketURL)
	err := ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	ok.Websocket.Wg.Add(1)
	go ok.wsReadData(ok.Websocket.Conn)
	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ok.Websocket.GetWebsocketURL())
	}
	ok.Websocket.Conn.SetupPingHandler(request.UnAuth, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	if ok.Websocket.CanUseAuthenticatedEndpoints() {
		err = ok.WsSpreadAuth(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsSpreadAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ok *Okx) WsSpreadAuth(ctx context.Context) error {
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ok.Name)
	}
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timeUnix := time.Now()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(timeUnix.Unix(), 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64Sign := crypto.Base64Encode(hmac)
	wsReq := WebsocketEventRequest{
		Operation: operationLogin,
		Arguments: []WebsocketLoginData{
			{
				APIKey:     creds.Key,
				Passphrase: creds.ClientID,
				Timestamp:  timeUnix.Unix(),
				Sign:       base64Sign,
			},
		},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.UnAuth, wsReq)
	if err != nil {
		return err
	}
	timer := time.NewTimer(ok.WebsocketResponseCheckTimeout)
	randomID, err := common.GenerateRandomString(16)
	if err != nil {
		return fmt.Errorf("%w, generating random string for incoming websocket response failed", err)
	}
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:    randomID,
		Chan:  wsResponse,
		Event: operationLogin,
	}
	ok.WsRequestSemaphore <- 1
	defer func() {
		<-ok.WsRequestSemaphore
	}()
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Event == operationLogin && data.StatusCode == "0" {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return nil
			} else if data.Event == "error" &&
				(data.StatusCode == "60022" || data.StatusCode == "60009") {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
				return fmt.Errorf("authentication failed with error: %v", ErrorCodes[data.StatusCode])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				wsReq.Operation)
		}
	}
}

// GenerateDefaultBusinessSubscriptions returns a list of default subscriptions to business stream.
func (ok *Okx) GenerateDefaultBusinessSubscriptions() ([]subscription.Subscription, error) {
	var subs []string
	var subscriptions []subscription.Subscription
	subs = append(subs, defaultBusinessSubscribedChannels...)
	if ok.Websocket.CanUseAuthenticatedEndpoints() {
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
			pairs, err := ok.GetEnabledPairs(asset.Spread)
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
			pairs, err := ok.GetEnabledPairs(asset.PerpetualSwap)
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
func (ok *Okx) BusinessSubscribe(channelsToSubscribe subscription.List) error {
	return ok.handleBusinessSubscription(operationSubscribe, channelsToSubscribe)
}

// BusinessUnsubscribe sends a websocket unsubscription request to several channels to receive data.
func (ok *Okx) BusinessUnsubscribe(channelsToUnsubscribe subscription.List) error {
	return ok.handleBusinessSubscription(operationUnsubscribe, channelsToUnsubscribe)
}

// handleBusinessSubscription sends a subscription and unsubscription information thought the business websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (ok *Okx) handleBusinessSubscription(operation string, subscriptions subscription.List) error {
	wsSubscriptionReq := WSSubscriptionInformationList{Operation: operation}
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	var channels subscription.List
	var authChannels subscription.List
	var err error
	for i := 0; i < len(subscriptions); i++ {
		arg := SubscriptionInfo{
			Channel: subscriptions[i].Channel,
		}
		var instrumentID, instrumentFamily, spreadID string
		switch arg.Channel {
		case okxSpreadOrders,
			okxSpreadTrades,
			okxSpreadOrderbookLevel1,
			okxSpreadOrderbook,
			okxSpreadPublicTrades,
			okxSpreadPublicTicker:
			spreadID = subscriptions[i].Pairs[0].String()
		case channelPublicBlockTrades,
			channelBlockTickers:
			instrumentID = subscriptions[i].Pairs[0].String()
		}
		instrumentFamilyInterface, okay := subscriptions[i].Params["instFamily"]
		if okay {
			instrumentFamily, _ = instrumentFamilyInterface.(string)
		}

		arg.InstrumentFamily = instrumentFamily
		arg.SpreadID = spreadID
		arg.InstrumentID = instrumentID

		var chunk []byte
		channels = append(channels, subscriptions[i])
		wsSubscriptionReq.Arguments = append(wsSubscriptionReq.Arguments, arg)
		chunk, err = json.Marshal(wsSubscriptionReq)
		if err != nil {
			return err
		}
		if len(chunk) > maxConnByteLen {
			i--
			err = ok.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, wsSubscriptionReq)
			if err != nil {
				return err
			}
			if operation == operationUnsubscribe {
				err = ok.Websocket.RemoveSubscriptions(ok.Websocket.Conn, channels...)
			} else {
				err = ok.Websocket.AddSuccessfulSubscriptions(ok.Websocket.Conn, channels...)
			}
			if err != nil {
				return err
			}
			channels = subscription.List{}
			wsSubscriptionReq.Arguments = []SubscriptionInfo{}
			continue
		}
	}
	err = ok.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, wsSubscriptionReq)
	if err != nil {
		return err
	}

	if operation == operationUnsubscribe {
		channels = append(channels, authChannels...)
		err = ok.Websocket.RemoveSubscriptions(ok.Websocket.Conn, channels...)
	} else {
		channels = append(channels, authChannels...)
		err = ok.Websocket.AddSuccessfulSubscriptions(ok.Websocket.Conn, channels...)
	}
	return err
}
