package bitfinex

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// wsMaxChannelsPerConnection is how many channels Bitfinex lets one connection subscribe to
	wsMaxChannelsPerConnection = 25
	// wsConfFlags asks Bitfinex to append its time to every event and to send order book checksums
	wsConfFlags = wsFlagTimestamp | wsFlagChecksum
	// wsFlagTimestamp appends the time Bitfinex sent an event to it
	wsFlagTimestamp = 1 << 15
	// wsFlagChecksum sends a checksum of each order book's top 25 levels every few seconds
	wsFlagChecksum = 1 << 17

	// wsPublicConnection and wsAccountConnection filter the public connections, which carry the market data
	// subscriptions, from the account connection, which authenticates and receives the account's events
	wsPublicConnection  = "public"
	wsAccountConnection = "account"

	// wsAccountChannelID is the channel the account's events arrive on once the connection authenticates
	wsAccountChannelID = 0

	// wsAuthNonceAttempts is how many times authentication is tried while Bitfinex refuses its nonce
	wsAuthNonceAttempts = 3

	// wsRenewRetryDelay is how long a channel's renewal waits before its first retry, doubling to wsRenewMaxRetryDelay
	wsRenewRetryDelay    = time.Second
	wsRenewMaxRetryDelay = time.Minute

	wsInfoServerRestart    = 20051
	wsInfoMaintenanceStart = 20060
	wsInfoMaintenanceEnd   = 20061

	wsBookChannel    = "book"
	wsTickerChannel  = "ticker"
	wsTradesChannel  = "trades"
	wsCandlesChannel = "candles"
	// wsStatusChannel streams a derivative pair's status, with its funding, mark price and open interest
	wsStatusChannel = "status"

	// CandlesPeriodKey configures the Candles aggregated period for MarginFunding in subscription.Subscription.Params
	CandlesPeriodKey = "_period"
)

var (
	errWebsocketMalformedFrame   = errors.New("malformed websocket frame")
	errWebsocketEventFailed      = errors.New("websocket event failed")
	errWebsocketChannelUnknown   = errors.New("websocket channel unknown")
	errWebsocketAuthentication   = errors.New("websocket authentication failed")
	errWebsocketUnsubscribe      = errors.New("websocket unsubscribe failed")
	errWebsocketChannelDropped   = errors.New("websocket channel unsubscribed during its renewal")
	errSubscriptionParamReserved = errors.New("subscription parameter is reserved")
	errSubscriptionNotSinglePair = errors.New("subscription must have exactly one pair")
	errStatusChannelAsset        = errors.New("status channel streams derivative pairs only")
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.All},
	{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.All},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Margin, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.MarginFunding, Interval: kline.OneMin, Params: map[string]any{CandlesPeriodKey: "p30"}},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.USDTMarginedFutures, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.CoinMarginedFutures, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Levels: 100, Params: map[string]any{"prec": "R0"}},
	{Enabled: true, Channel: wsStatusChannel, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: wsStatusChannel, Asset: asset.CoinMarginedFutures},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    wsTickerChannel,
	subscription.OrderbookChannel: wsBookChannel,
	subscription.CandlesChannel:   wsCandlesChannel,
	subscription.AllTradesChannel: wsTradesChannel,
}

// setupWebsocket sets the websocket up with public connections, which carry the market data subscriptions, up to
// Bitfinex's channel limit each, and an account connection, which authenticates and receives the account's events
func (e *Exchange) setupWebsocket(exch *config.Exchange) error {
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: wsMaxChannelsPerConnection,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
	}); err != nil {
		return err
	}
	publicURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	accountURL, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   publicURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.wsConnect,
		GenerateSubscriptions: e.generateSubscriptions,
		Subscriber:            e.wsSubscribe,
		Unsubscriber:          e.wsUnsubscribe,
		Handler:               e.wsHandleData,
		MessageFilter:         wsPublicConnection,
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   accountURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.wsConnect,
		Authenticate:          e.wsAuthenticate,
		GenerateSubscriptions: e.generateAccountSubscriptions,
		Subscriber:            e.wsSubscribeAccount,
		Unsubscriber:          e.wsUnsubscribeAccount,
		Handler:               e.wsHandleData,
		MessageFilter:         wsAccountConnection,
	})
}

// wsConnect dials a connection and configures it to send event times and order book checksums. The connection's
// reader starts once it returns, so the configuration's reply is checked as it arrives; Bitfinex applies it before the
// requests that follow it
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	e.wsChannels.pruneClosed()
	return conn.SendJSONMessage(ctx, request.Unset, &wsConfRequest{Event: "conf", Flags: wsConfFlags})
}

// wsAuthenticate authenticates the account connection, which then receives the account's events on channel 0. A
// failure leaves the public connections up and turns authenticated websocket use off, as GCT did before connections
// were managed separately
func (e *Exchange) wsAuthenticate(ctx context.Context, conn websocket.Connection) error {
	e.wsWalletCalcs.reset()
	if err := e.wsSendAuth(ctx, conn); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket authentication failed, authenticated websocket use is off: %v", e.Name, err)
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	return nil
}

// wsSendAuth sends the authentication request and checks its reply. Bitfinex keeps one nonce per key across its REST
// and websocket connections, so authenticating takes its nonce from the counter REST requests are signed with; a REST
// request signed after it can still reach Bitfinex first, so a nonce Bitfinex refuses is retried with a fresh one
func (e *Exchange) wsSendAuth(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	for attempt := 1; ; attempt++ {
		n := e.Requester.GetNonce(nonce.UnixNano).String()
		payload := "AUTH" + n
		signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(payload), []byte(creds.Secret))
		if err != nil {
			return err
		}
		resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, wsAuthSignature, &wsAuthRequest{
			Event:       "auth",
			APIKey:      creds.Key,
			AuthPayload: payload,
			AuthNonce:   n,
			AuthSig:     hex.EncodeToString(signature),
		})
		if err != nil {
			return err
		}
		var auth wsEvent
		if err := json.Unmarshal(resp, &auth); err != nil {
			return err
		}
		switch {
		case auth.Status == "OK":
			return nil
		case auth.Code != wsErrorAuthNonce || attempt == wsAuthNonceAttempts:
			return fmt.Errorf("%w: %s (code %d)", errWebsocketAuthentication, auth.Message, auth.Code)
		}
	}
}

// generateSubscriptions returns the public connections' subscriptions from the configured ones. Spot and margin trade
// the same markets, so a margin subscription asking for a spot one's channel is carried by it, as Bitfinex refuses a
// connection's second subscription to a channel
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	// Account events need no subscription, so a configured authenticated subscription has nothing to subscribe to
	subs = subs.Public()
	spot := make(map[string]*subscription.Subscription)
	for _, s := range subs {
		if s.Asset == asset.Spot {
			spot[s.QualifiedChannel] = s
		}
	}
	shared := make(map[string]*subscription.Subscription)
	subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool {
		if s.Asset != asset.Margin {
			return false
		}
		spotSub, ok := spot[s.QualifiedChannel]
		if ok {
			shared[s.QualifiedChannel] = spotSub
		}
		return ok
	})
	for _, s := range e.wsSharedChannels.set(shared) {
		// A margin book its spot channel no longer updates is stale until a snapshot of its own; one shared again then
		// has its channel renewed for it
		if s.Channel == subscription.OrderbookChannel {
			_ = e.Websocket.Orderbook.InvalidateOrderbook(s.Pairs[0], asset.Margin)
		}
	}
	return subs, nil
}

// generateAccountSubscriptions returns the account channel, which authenticating subscribes the account connection
// to, when authenticated websocket use is available; without it there is no account connection
func (e *Exchange) generateAccountSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	return subscription.List{{Channel: subscription.MyAccountChannel, Authenticated: true, Asset: asset.All}}, nil
}

// wsSubscribeAccount records the account channel, which Bitfinex subscribes an authenticated connection to
func (e *Exchange) wsSubscribeAccount(_ context.Context, conn websocket.Connection, subs subscription.List) error {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: the account connection is not authenticated", websocket.ErrSubscriptionFailure)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// wsUnsubscribeAccount forgets the account channel, which lasts as long as the connection
func (e *Exchange) wsUnsubscribeAccount(_ context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{"subToMap": subToMap}).Parse(subTplText)
}

// subToMap returns the subscribe request fields of a subscription to one pair
func subToMap(s *subscription.Subscription, a asset.Item, p currency.Pair) (map[string]any, error) {
	c := s.Channel
	if name, ok := subscriptionNames[s.Channel]; ok {
		c = name
	}
	req := map[string]any{"channel": c}
	var fundingPeriod string
	for k, v := range s.Params {
		switch k {
		case CandlesPeriodKey:
			period, ok := v.(string)
			if !ok {
				return nil, common.GetTypeAssertError("string", v, "subscription.CandlesPeriodKey")
			}
			fundingPeriod = ":" + period
		case "key", "symbol", "len", "event", "subId":
			// The request sets these itself, so a parameter would be overwritten without notice
			return nil, fmt.Errorf("%w: %s", errSubscriptionParamReserved, k)
		default:
			req[k] = v
		}
	}
	if s.Levels != 0 {
		req["len"] = strconv.Itoa(s.Levels)
	}
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	switch c {
	case wsCandlesChannel:
		req["key"] = "trade:" + s.Interval.Short() + ":" + symbol + fundingPeriod
	case wsStatusChannel:
		if !isDerivativeAsset(a) {
			return nil, fmt.Errorf("%w: %s %s", errStatusChannelAsset, a, p)
		}
		req["key"] = "deriv:" + symbol
	default:
		req["symbol"] = symbol
	}
	return req, nil
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- range $p := $pairs  -}}
		{{- subToMap $.S $asset $p | mustToJson }}
		{{- $.PairSeparator }}
	{{- end -}}
	{{ $.AssetSeparator }}
{{- end -}}
`

// wsSubscribe subscribes to the channels at once, as each request awaits its own confirmation, recording each
// Bitfinex confirms
func (e *Exchange) wsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		if err := e.wsSubscribeChannel(ctx, conn, l[0]); err != nil {
			return fmt.Errorf("%w: %s", err, l[0])
		}
		return nil
	}, 1)
}

// wsSubscribeChannel subscribes to one channel, recording its subscription once Bitfinex confirms it
func (e *Exchange) wsSubscribeChannel(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	if err := e.wsSendSubscribe(ctx, conn, s); err != nil {
		return err
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, s)
}

// wsSendSubscribe sends a subscription's request, which is keyed by its ID until Bitfinex confirms it, when the reader
// maps the channel ID it is given before reading the channel's first event
func (e *Exchange) wsSendSubscribe(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	if len(s.Pairs) != 1 {
		return errSubscriptionNotSinglePair
	}
	req := map[string]any{}
	if err := json.Unmarshal([]byte(s.QualifiedChannel), &req); err != nil {
		return err
	}
	subID := e.MessageID()
	req["event"] = "subscribe"
	req["subId"] = subID
	// The request stays pending until Bitfinex replies, which the reader handles, so that a confirmation arriving after
	// the wait for it ended, by timing out or by its context ending, still maps its channel; a request never sent is
	// forgotten with its connection
	e.wsChannels.addPending(subID, conn, s)
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, wsSubscribeSignature(subID), req)
	if err != nil {
		return err
	}
	var event wsEvent
	if err := json.Unmarshal(resp, &event); err != nil {
		return err
	}
	if event.Event == "subscribed" {
		return nil
	}
	// Bitfinex refuses a second subscription to a channel, naming it; one already carrying an equal subscription,
	// confirmed after its request stopped waiting, is this subscription
	if event.Code == wsErrorDuplicate && e.wsChannels.adopt(conn, event.ChannelID, s) {
		return nil
	}
	return fmt.Errorf("%w: %s (code %d)", websocket.ErrSubscriptionFailure, event.Message, event.Code)
}

// wsUnsubscribe unsubscribes from the channels at once
func (e *Exchange) wsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		if err := e.wsUnsubscribeChannel(ctx, conn, l[0]); err != nil {
			return fmt.Errorf("%w: %s", err, l[0])
		}
		return nil
	}, 1)
}

// wsUnsubscribeChannel unsubscribes from one channel, forgetting it once Bitfinex confirms
func (e *Exchange) wsUnsubscribeChannel(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	e.wsRenewMu.RLock()
	defer e.wsRenewMu.RUnlock()
	channelID, ok := e.wsChannels.idOf(conn, s)
	if !ok {
		// Bitfinex never confirmed the channel, or a renewal left it unsubscribed, so it is not subscribed; a
		// confirmation still to come is unsubscribed when it arrives
		e.wsChannels.unpark(conn, s)
		e.wsChannels.dropPending(conn, s)
		return e.Websocket.RemoveSubscriptions(conn, s)
	}
	if err := e.wsSendUnsubscribe(ctx, conn, channelID); err != nil {
		return err
	}
	e.wsChannels.remove(conn, channelID)
	return e.Websocket.RemoveSubscriptions(conn, s)
}

// wsSendUnsubscribe sends a channel's unsubscribe request
func (e *Exchange) wsSendUnsubscribe(ctx context.Context, conn websocket.Connection, channelID int64) error {
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, wsUnsubscribeSignature(channelID), &wsUnsubscribeRequest{Event: "unsubscribe", ChannelID: channelID})
	if err != nil {
		return err
	}
	var event wsEvent
	if err := json.Unmarshal(resp, &event); err != nil {
		return err
	}
	// A channel Bitfinex no longer has is as unsubscribed as one it removes
	if event.Event != "unsubscribed" && event.Code != wsErrorNotSubscribed && event.Code != wsErrorUnsubscribeInvalid {
		return fmt.Errorf("%w: %s (code %d)", errWebsocketUnsubscribe, event.Message, event.Code)
	}
	return nil
}

// resubscribeBook renews the channel of a book that no longer matches Bitfinex's, which sends a fresh snapshot. It runs
// apart from the connection's reader, which must read the replies, and retries until the channel is renewed, the
// manager unsubscribes it or the connection ends
func (e *Exchange) resubscribeBook(ctx context.Context, conn websocket.Connection, s *subscription.Subscription, k key.PairAsset, resync uint64) {
	e.Websocket.Wg.Go(func() {
		for delay := wsRenewRetryDelay; ; delay = min(2*delay, wsRenewMaxRetryDelay) {
			err := e.wsRenewChannel(ctx, conn, s)
			if err == nil {
				// The fresh snapshot ends the book's resync
				return
			}
			if errors.Is(err, errWebsocketChannelDropped) {
				break
			}
			log.Errorf(log.WebsocketMgr, "%s error resubscribing %s: %v", e.Name, s, err)
			if !e.wsRenewWait(ctx, conn, delay) {
				break
			}
			// A confirmation arriving after its request stopped waiting brings the snapshot, ending the resync
			if !e.wsBookResync.active(k, resync) {
				return
			}
		}
		e.wsBookResync.abandon(k, resync)
	})
}

// wsRenewChannel unsubscribes a subscription's channel and subscribes it again, leaving the manager's subscription as
// it is. A channel an earlier attempt left unsubscribed is subscribed, and one the manager has unsubscribed meanwhile
// is left alone. Bitfinex counts a connection's channels, so the channel is unsubscribed first
func (e *Exchange) wsRenewChannel(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	e.wsRenewMu.Lock()
	defer e.wsRenewMu.Unlock()
	if channelID, ok := e.wsChannels.channelOf(conn, s); ok {
		if err := e.wsSendUnsubscribe(ctx, conn, channelID); err != nil {
			return err
		}
		e.wsChannels.park(conn, channelID)
	} else if !e.wsChannels.isParked(conn, s) {
		return fmt.Errorf("%w: %s", errWebsocketChannelDropped, s)
	}
	return e.wsSendSubscribe(ctx, conn, s)
}

// wsRenewWait waits to retry a channel's renewal, reporting false once the connection or the websocket has ended
func (e *Exchange) wsRenewWait(ctx context.Context, conn websocket.Connection, delay time.Duration) bool {
	if isConnectionClosed(conn) {
		return false
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-e.Websocket.ShutdownC:
		return false
	case <-timer.C:
		return !isConnectionClosed(conn)
	}
}

// wsHandleData routes a frame by its form: an object is an event, and an array is a channel's update, channel 0
// carrying the account's events
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	respRaw = bytes.TrimSpace(respRaw)
	if len(respRaw) == 0 {
		return fmt.Errorf("%w: empty frame", errWebsocketMalformedFrame)
	}
	switch respRaw[0] {
	case '{':
		return e.wsHandleEvent(ctx, conn, respRaw)
	case '[':
		channelID, err := jsonparser.GetInt(respRaw, "[0]")
		if err != nil {
			return fmt.Errorf("%w: %w: %.256s", errWebsocketMalformedFrame, err, respRaw)
		}
		if channelID == wsAccountChannelID {
			return e.wsHandleAccountEvent(ctx, conn, respRaw)
		}
		return e.wsHandleChannelEvent(ctx, conn, channelID, respRaw)
	}
	return fmt.Errorf("%w: %.256s", errWebsocketMalformedFrame, respRaw)
}

// wsHandleEvent handles the object events Bitfinex sends: replies to requests and information about the connection
func (e *Exchange) wsHandleEvent(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var event wsEvent
	if err := json.Unmarshal(respRaw, &event); err != nil {
		return err
	}
	switch event.Event {
	case "subscribed":
		// The channel is mapped before the subscriber hears, so its first event, which follows, is not dropped. A
		// confirmation arriving after its request stopped waiting maps its channel all the same
		mapped, err := e.wsChannels.confirm(event.SubscriptionID, conn, event.ChannelID)
		if err != nil {
			return err
		}
		if !mapped {
			// The subscription was unsubscribed before Bitfinex confirmed it, so its channel is unsubscribed in turn,
			// without waiting, as the reader reads the reply
			return conn.SendJSONMessage(ctx, request.Unset, &wsUnsubscribeRequest{Event: "unsubscribe", ChannelID: event.ChannelID})
		}
		conn.IncomingWithData(wsSubscribeSignature(event.SubscriptionID), respRaw)
		return nil
	case "unsubscribed":
		if e.wsChannels.drained(conn, event.ChannelID) {
			return nil
		}
		return conn.RequireMatchWithData(wsUnsubscribeSignature(event.ChannelID), respRaw)
	case "error":
		switch {
		case event.SubscriptionID != "":
			e.wsChannels.removePending(event.SubscriptionID)
			if conn.IncomingWithData(wsSubscribeSignature(event.SubscriptionID), respRaw) {
				return nil
			}
			// A refusal arriving after its request stopped waiting
			return fmt.Errorf("%w: %s (code %d)", websocket.ErrSubscriptionFailure, event.Message, event.Code)
		case event.ChannelID != 0:
			// Bitfinex no longer having a draining channel leaves nothing to unsubscribe
			if e.wsChannels.drained(conn, event.ChannelID) {
				return nil
			}
			return conn.RequireMatchWithData(wsUnsubscribeSignature(event.ChannelID), respRaw)
		}
		return fmt.Errorf("%w: %s (code %d)", errWebsocketEventFailed, event.Message, event.Code)
	case "auth":
		return conn.RequireMatchWithData(wsAuthSignature, respRaw)
	case "conf":
		if event.Status != "OK" {
			return fmt.Errorf("%w: conf flags %d: %s", errWebsocketEventFailed, wsConfFlags, respRaw)
		}
		return nil
	case "info":
		return e.wsHandleInfo(ctx, &event)
	case "pong":
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

// wsHandleInfo relays the information events about the platform, other than the version Bitfinex sends on connecting
func (e *Exchange) wsHandleInfo(ctx context.Context, event *wsEvent) error {
	switch event.Code {
	case 0:
		return nil
	case wsInfoServerRestart:
		// Bitfinex closes the connection, which reconnects
		log.Warnf(log.WebsocketMgr, "%s websocket server is restarting: %s", e.Name, event.Message)
	case wsInfoMaintenanceStart:
		log.Warnf(log.WebsocketMgr, "%s platform maintenance has started, orders cannot be placed or cancelled: %s", e.Name, event.Message)
	case wsInfoMaintenanceEnd:
		log.Infof(log.WebsocketMgr, "%s platform maintenance has ended: %s", e.Name, event.Message)
	}
	return e.Websocket.DataHandler.Send(ctx, &PlatformInfoEvent{Code: event.Code, Message: event.Message})
}

// wsSubscribeSignature matches a subscribe request with its reply
func wsSubscribeSignature(subID string) string {
	return "subscribe:" + subID
}

// wsUnsubscribeSignature matches an unsubscribe request with its reply
func wsUnsubscribeSignature(channelID int64) string {
	return "unsubscribe:" + strconv.FormatInt(channelID, 10)
}

// isConnectionClosed reports whether a connection reports itself disconnected; the manager's connections do
var isConnectionClosed = func(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return ok && !c.IsConnected()
}
