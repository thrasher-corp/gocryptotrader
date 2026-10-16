package hyperliquid

import (
	"bytes"
	"cmp"
	"compress/flate"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// websocketPingInterval keeps quiet connections inside the server's 60 second idle timeout
	websocketPingInterval = 25 * time.Second
	// accountWebsocketProbeTimeout bounds each probe of the websocket endpoint while the account connection is down
	accountWebsocketProbeTimeout = 10 * time.Second
	// maximumAccountWebsocketProbeInterval keeps failed probes well within the 30 new connections a minute that
	// Hyperliquid accepts from an address
	maximumAccountWebsocketProbeInterval = time.Minute
	// websocketMessageInterval spaces requests within the limit of 2000 messages a minute
	websocketMessageInterval = 30 * time.Millisecond
	// maximumWebsocketSubscriptions is Hyperliquid's per-IP subscription limit; the manager counts the public and account
	// connections' subscriptions against it together
	maximumWebsocketSubscriptions = 1000
	// maximumFastAssetContextsSize bounds a decompressed fastAssetCtxs message; a full snapshot is about 82 KB
	maximumFastAssetContextsSize = 16 << 20
	// maximumWebsocketOrders bounds the orders whose fills are added up at twice Hyperliquid's 5000 open order limit
	maximumWebsocketOrders = 10000
	// websocketOrderLifetime is how long an order whose rows and fills stop arriving is remembered
	websocketOrderLifetime = 24 * time.Hour

	wsMethodSubscribe   = "subscribe"
	wsMethodUnsubscribe = "unsubscribe"
	wsMethodPost        = "post"
	wsPostTypeInfo      = "info"
	wsPostTypeAction    = "action"
	wsPostTypeError     = "error"

	// Subscription types, which name their data channels except userEvents
	wsChannelAllMids                     = "allMids"
	wsChannelNotification                = "notification"
	wsChannelWebData3                    = "webData3"
	wsChannelTWAPStates                  = "twapStates"
	wsChannelClearinghouseState          = "clearinghouseState"
	wsChannelOpenOrders                  = "openOrders"
	wsChannelCandle                      = "candle"
	wsChannelOrderbook                   = "l2Book"
	wsChannelTrades                      = "trades"
	wsChannelOrderUpdates                = "orderUpdates"
	wsSubscriptionUserEvents             = "userEvents"
	wsChannelUserFills                   = "userFills"
	wsChannelUserFundings                = "userFundings"
	wsChannelUserNonFundingLedgerUpdates = "userNonFundingLedgerUpdates"
	wsChannelActiveAssetContext          = "activeAssetCtx"
	wsChannelActiveAssetData             = "activeAssetData"
	wsChannelUserTWAPSliceFills          = "userTwapSliceFills"
	wsChannelUserTWAPHistory             = "userTwapHistory"
	wsChannelBestBidOffer                = "bbo"
	wsChannelSpotState                   = "spotState"
	wsChannelAllDEXsClearinghouseState   = "allDexsClearinghouseState"
	wsChannelAllDEXsAssetContexts        = "allDexsAssetCtxs"
	wsChannelOutcomeMetaUpdates          = "outcomeMetaUpdates"
	wsChannelFastAssetContexts           = "fastAssetCtxs"

	// Data channels that do not share their subscription's name
	wsChannelUserEvents             = "user"
	wsChannelActiveSpotAssetContext = "activeSpotAssetCtx"

	wsChannelSubscriptionResponse = "subscriptionResponse"
	wsChannelPost                 = "post"
	wsChannelPong                 = "pong"
	wsChannelError                = "error"
	// websocketConnectionEstablished is a plain text greeting the server once sent on connection
	websocketConnectionEstablished = "Websocket connection established."

	// Error message prefixes that identify the request they answer
	wsErrorAlreadySubscribed   = "Already subscribed: "
	wsErrorAlreadyUnsubscribed = "Already unsubscribed: "
	wsErrorInvalidSubscription = "Invalid subscription "
	wsErrorParseFailure        = "Error parsing JSON into valid websocket request: "
)

var (
	errWebsocketAssetMismatch = errors.New("websocket market asset mismatch")
	errWebsocketFeedUnknown   = errors.New("websocket subscription has no subscribed feed")
	errWebsocketParameter     = errors.New("invalid websocket subscription parameter")
	errWebsocketPost          = errors.New("websocket post request failed")
	errWebsocketServer        = errors.New("websocket server error")
	errWebsocketSubscription  = errors.New("websocket subscription acknowledgement error")
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
}

// websocketFeed describes the parameters a Hyperliquid subscription type takes
type websocketFeed struct {
	// coin feeds are per market, taking the subscription's single pair and its asset
	coin bool
	// user feeds are per account, taking the configured account address, and use the authenticated connection
	user bool
	// dex feeds take an optional perpetual DEX name from the subscription's dex parameter
	dex bool
}

// websocketFeeds lists every Hyperliquid subscription type, which a subscription may name as its channel
var websocketFeeds = map[string]websocketFeed{
	wsChannelAllMids:                     {dex: true},
	wsChannelNotification:                {user: true},
	wsChannelWebData3:                    {user: true},
	wsChannelTWAPStates:                  {user: true, dex: true},
	wsChannelClearinghouseState:          {user: true, dex: true},
	wsChannelOpenOrders:                  {user: true, dex: true},
	wsChannelCandle:                      {coin: true},
	wsChannelOrderbook:                   {coin: true},
	wsChannelTrades:                      {coin: true},
	wsChannelOrderUpdates:                {user: true},
	wsSubscriptionUserEvents:             {user: true},
	wsChannelUserFills:                   {user: true},
	wsChannelUserFundings:                {user: true},
	wsChannelUserNonFundingLedgerUpdates: {user: true},
	wsChannelActiveAssetContext:          {coin: true},
	wsChannelActiveAssetData:             {coin: true, user: true},
	wsChannelUserTWAPSliceFills:          {user: true},
	wsChannelUserTWAPHistory:             {user: true},
	wsChannelBestBidOffer:                {coin: true},
	wsChannelSpotState:                   {user: true},
	wsChannelAllDEXsClearinghouseState:   {user: true},
	wsChannelAllDEXsAssetContexts:        {},
	wsChannelOutcomeMetaUpdates:          {},
	wsChannelFastAssetContexts:           {},
}

// websocketStandardChannels maps standard channels to the Hyperliquid subscription types that serve them; userEvents
// serves myTrades because it carries TWAP slice fills, which userFills omits, and sends no snapshot to replay
var websocketStandardChannels = map[string]string{
	subscription.TickerChannel:    wsChannelActiveAssetContext,
	subscription.OrderbookChannel: wsChannelOrderbook,
	subscription.AllTradesChannel: wsChannelTrades,
	subscription.CandlesChannel:   wsChannelCandle,
	subscription.MyOrdersChannel:  wsChannelOrderUpdates,
	subscription.MyTradesChannel:  wsSubscriptionUserEvents,
}

// WsConnect connects the public stream and, when an account address is configured, a stream for its account feeds
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	// Taken before dialling, as the backoff is for this attempt alone and a failed one must not leave it to a later
	// connection
	backoff := time.Duration(e.accountProbeInterval.Swap(0))
	if err := e.connectWebsocket(ctx, e.Websocket.Conn); err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx, e.Websocket.Conn, false)

	if !e.IsWebsocketAuthenticationSupported() {
		return nil
	}
	if _, err := e.getWatchAddress(ctx); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Warnf(log.WebsocketMgr, "%s account websocket feeds disabled: %s", e.Name, err)
		return nil
	}
	if err := e.connectWebsocket(ctx, e.Websocket.AuthConn); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Warnf(log.WebsocketMgr, "%s account websocket connection failed, retrying: %s", e.Name, err)
		interval := max(cmp.Or(e.Config.ConnectionMonitorDelay, config.DefaultConnectionMonitorDelay), backoff)
		go e.awaitAccountWebsocket(ctx, e.Websocket.AuthConn.GetURL(), e.Websocket.GetProxyAddress(), e.Websocket.ShutdownC, interval)
		return nil
	}
	// Hyperliquid's account feeds are public and keyed by address, so authenticated here only means an address is set
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx, e.Websocket.AuthConn, true)
	return nil
}

func (e *Exchange) connectWebsocket(ctx context.Context, connection websocket.Connection) error {
	if connection == nil {
		return common.ErrNilPointer
	}
	if err := connection.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return fmt.Errorf("dialling %s: %w", connection.GetURL(), err)
	}
	connection.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"method":"ping"}`),
		Delay:       websocketPingInterval,
	})
	return nil
}

// awaitAccountWebsocket probes the websocket endpoint after the account connection failed to dial, keeping the public
// feeds up, and shuts the websocket down once a probe connects, so the connection monitor reconnects it with the account
// feeds; it stops when the websocket shuts down. Probes start after the given interval, which doubles after each failed
// probe up to a minute. A probe that connects hands the doubled interval to the reconnection it causes, so an account
// dial that fails again straight after backs off further rather than starting over. The endpoint, proxy and shutdown
// channel are passed in, as reading them later would race with the manager changing them
func (e *Exchange) awaitAccountWebsocket(ctx context.Context, endpoint, proxy string, shutdown <-chan struct{}, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-shutdown:
			return
		case <-timer.C:
		}
		if err := probeWebsocket(ctx, endpoint, proxy); err != nil {
			if e.Verbose {
				log.Debugf(log.WebsocketMgr, "%s account websocket probe failed: %s", e.Name, err)
			}
			interval = nextAccountWebsocketProbeInterval(interval)
			timer.Reset(interval)
			continue
		}
		select {
		case <-shutdown:
			return
		default:
		}
		// Handed over before shutting down, as the connection monitor can reconnect before Shutdown returns
		backoff := int64(nextAccountWebsocketProbeInterval(interval))
		e.accountProbeInterval.Store(backoff)
		if err := e.Websocket.Shutdown(); err != nil {
			// A shutdown or reconnection already under way also leaves the connection monitor to reconnect, though not
			// because of this probe, so the backoff is withdrawn unless that reconnection has taken it already
			e.accountProbeInterval.CompareAndSwap(backoff, 0)
			if e.Verbose {
				log.Debugf(log.WebsocketMgr, "%s websocket shutdown for the account feeds: %s", e.Name, err)
			}
		}
		return
	}
}

// nextAccountWebsocketProbeInterval doubles the interval between probes of the websocket endpoint, up to a minute unless
// the interval was longer already
func nextAccountWebsocketProbeInterval(interval time.Duration) time.Duration {
	return max(interval, min(interval*2, maximumAccountWebsocketProbeInterval))
}

// probeWebsocket dials a websocket endpoint, through a proxy when one is set, then closes the connection normally
func probeWebsocket(ctx context.Context, endpoint, proxy string) error {
	dialer := gws.Dialer{HandshakeTimeout: accountWebsocketProbeTimeout}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxyURL)
	}
	conn, resp, err := dialer.DialContext(ctx, endpoint, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return err
	}
	_ = conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, ""), time.Now().Add(accountWebsocketProbeTimeout))
	return conn.Close()
}

// wsReadData relays a connection's messages until it closes, then fails its outstanding subscription operations; the
// replays the public connection awaited will not arrive, and userEvents does not resend the fills sent while the account
// connection is down, so the orders' fills may then be incomplete
func (e *Exchange) wsReadData(ctx context.Context, connection websocket.Connection, authenticated bool) {
	defer e.Websocket.Wg.Done()
	for {
		message := connection.ReadMessage()
		if message.Raw == nil {
			_ = e.failWebsocketPending(authenticated, websocket.ErrNotConnected)
			if authenticated {
				e.uncoverWebsocketOrders()
			} else {
				e.clearTradeReplays()
			}
			return
		}
		if err := e.wsHandleData(ctx, message.Raw, authenticated); err != nil {
			if sendErr := e.Websocket.DataHandler.Send(ctx, err); sendErr != nil {
				log.Errorf(log.WebsocketMgr, "%s websocket data handler: %s; source error: %s", e.Name, sendErr, err)
			}
		}
	}
}

// wsHandleData handles a message, settling subscription operations on the connection that sent them; an error names
// the channel whose message failed, as the data handler relays it without further context
func (e *Exchange) wsHandleData(ctx context.Context, raw []byte, authenticated bool) error {
	if string(raw) == websocketConnectionEstablished {
		return nil
	}
	var envelope WsResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("%s decoding websocket message: %w", e.Name, err)
	}
	if err := e.wsHandleChannel(ctx, &envelope, raw, authenticated); err != nil {
		return fmt.Errorf("%s %s message: %w", e.Name, envelope.Channel, err)
	}
	return nil
}

// wsHandleChannel passes a message to its channel's handler, relaying a message of an unknown channel as unhandled
func (e *Exchange) wsHandleChannel(ctx context.Context, envelope *WsResponse, raw []byte, authenticated bool) error {
	switch envelope.Channel {
	case wsChannelSubscriptionResponse:
		return e.wsHandleSubscriptionResponse(envelope.Data, authenticated)
	case wsChannelPost:
		return e.wsHandlePostResponse(envelope.Data)
	case wsChannelPong:
		return nil
	case wsChannelError:
		return e.wsHandleError(envelope.Data, authenticated)
	case wsChannelAllMids:
		return wsRelay[WsAllMids](ctx, e, envelope.Data)
	case wsChannelNotification:
		return wsRelay[WsNotification](ctx, e, envelope.Data)
	case wsChannelWebData3:
		return wsRelay[WsWebData3](ctx, e, envelope.Data)
	case wsChannelTWAPStates:
		return wsRelay[WsTWAPStates](ctx, e, envelope.Data)
	case wsChannelClearinghouseState:
		return wsRelay[WsClearinghouseState](ctx, e, envelope.Data)
	case wsChannelOpenOrders:
		return wsRelay[WsOpenOrders](ctx, e, envelope.Data)
	case wsChannelCandle:
		return e.wsHandleCandle(ctx, envelope.Data)
	case wsChannelOrderbook:
		return e.wsHandleOrderbook(ctx, envelope.Data)
	case wsChannelTrades:
		return e.wsHandleTrades(envelope.Data)
	case wsChannelOrderUpdates:
		return e.wsHandleOrderUpdates(ctx, envelope.Data)
	case wsChannelUserEvents:
		return e.wsHandleUserEvent(ctx, envelope.Data)
	case wsChannelUserFills:
		return e.wsHandleUserFills(ctx, envelope.Data)
	case wsChannelUserFundings:
		return e.wsHandleUserFundings(ctx, envelope.Data)
	case wsChannelUserNonFundingLedgerUpdates:
		return e.wsHandleUserNonFundingLedgerUpdates(ctx, envelope.Data)
	case wsChannelActiveAssetContext:
		return e.wsHandlePerpetualTicker(ctx, envelope.Data)
	case wsChannelActiveSpotAssetContext:
		return e.wsHandleSpotTicker(ctx, envelope.Data)
	case wsChannelActiveAssetData:
		return wsRelay[ActiveAssetDataResponse](ctx, e, envelope.Data)
	case wsChannelUserTWAPSliceFills:
		return e.wsHandleUserTWAPSliceFills(ctx, envelope.Data)
	case wsChannelUserTWAPHistory:
		return e.wsHandleUserTWAPHistory(ctx, envelope.Data)
	case wsChannelBestBidOffer:
		return wsRelay[WsBestBidOffer](ctx, e, envelope.Data)
	case wsChannelSpotState:
		return wsRelay[WsSpotState](ctx, e, envelope.Data)
	case wsChannelAllDEXsClearinghouseState:
		return wsRelay[WsAllDEXsClearinghouseState](ctx, e, envelope.Data)
	case wsChannelAllDEXsAssetContexts:
		return wsRelay[WsAllDEXsAssetContexts](ctx, e, envelope.Data)
	case wsChannelOutcomeMetaUpdates:
		return e.wsHandleOutcomeMetaUpdates(ctx, envelope.Data)
	case wsChannelFastAssetContexts:
		return e.wsHandleFastAssetContexts(ctx, envelope.Data)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(raw),
		})
	}
}

// wsRelay decodes a message into its exported type and relays it
func wsRelay[T any](ctx context.Context, e *Exchange, raw []byte) error {
	update := new(T)
	if err := json.Unmarshal(raw, update); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, update)
}

// wsHandleSubscriptionResponse completes the subscription operation an acknowledgement matches
func (e *Exchange) wsHandleSubscriptionResponse(raw []byte, authenticated bool) error {
	var response WsSubscriptionResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return fmt.Errorf("%w: %w", errWebsocketSubscription, err)
	}
	if (response.Method != wsMethodSubscribe && response.Method != wsMethodUnsubscribe) || strings.TrimSpace(response.Subscription.Type) == "" {
		return fmt.Errorf("%w: invalid response method %q or subscription", errWebsocketSubscription, response.Method)
	}
	if response.Method == wsMethodSubscribe && (response.Subscription.Type == wsSubscriptionUserEvents ||
		response.Subscription.Type == wsChannelUserTWAPSliceFills ||
		(response.Subscription.Type == wsChannelUserFills && !response.Subscription.AggregateByTime)) {
		// A newly started fills feed may have missed execution, including when its acknowledgement arrives after the
		// local operation timed out. Keep the fills for deduplication and recovery, but require a row to prove coverage.
		e.uncoverWebsocketOrders()
	}
	if response.Subscription.Type == wsChannelTrades {
		// Hyperliquid sends a trades subscription's replay of recent trades straight after acknowledging it, even when
		// subscriptions are sent together, so awaiting the replay from here keeps one replay awaited at a time
		if response.Method == wsMethodSubscribe {
			e.expectTradeReplay(response.Subscription.Coin)
		} else {
			e.clearTradeReplay(response.Subscription.Coin)
		}
	}
	_, err := e.completeWebsocketPending(authenticated, response.Method, &response.Subscription)
	return err
}

// completeWebsocketPending applies the server's completion of a pending operation to the subscription store; it reports
// whether an operation was pending, as a late or unknown completion is ignored
func (e *Exchange) completeWebsocketPending(authenticated bool, method string, sub *WsSubscription) (bool, error) {
	key := websocketPendingKey{authenticated: authenticated, subscription: *sub}
	e.websocketPendingMu.Lock()
	pending := e.websocketPending[key]
	if pending == nil {
		e.websocketPendingMu.Unlock()
		return false, nil
	}
	if pending.method != method {
		e.websocketPendingMu.Unlock()
		return true, fmt.Errorf("%w: expected %s, got %s", errWebsocketSubscription, pending.method, method)
	}
	delete(e.websocketPending, key)
	e.websocketPendingMu.Unlock()

	var err error
	switch method {
	case wsMethodSubscribe:
		err = pending.subscription.SetState(subscription.SubscribedState)
	case wsMethodUnsubscribe:
		err = e.Websocket.RemoveSubscriptions(pending.connection, pending.subscription)
	}
	if err != nil {
		err = common.AppendError(err, e.rollbackWebsocketPending(pending))
	}
	pending.done <- err
	return true, err
}

// failWebsocketPendingOperation fails and rolls back one pending operation; it reports whether the operation was pending
func (e *Exchange) failWebsocketPendingOperation(authenticated bool, method string, sub *WsSubscription, cause error) bool {
	key := websocketPendingKey{authenticated: authenticated, subscription: *sub}
	e.websocketPendingMu.Lock()
	pending := e.websocketPending[key]
	if pending == nil || pending.method != method {
		e.websocketPendingMu.Unlock()
		return false
	}
	delete(e.websocketPending, key)
	e.websocketPendingMu.Unlock()
	pending.done <- common.AppendError(cause, e.rollbackWebsocketPending(pending))
	return true
}

// rollbackWebsocketPending restores the local subscription state of an operation the server did not complete
func (e *Exchange) rollbackWebsocketPending(pending *websocketPendingOperation) error {
	if pending == nil || pending.connection == nil || pending.subscription == nil {
		return common.ErrNilPointer
	}
	switch pending.method {
	case wsMethodSubscribe:
		return e.Websocket.RemoveSubscriptions(pending.connection, pending.subscription)
	case wsMethodUnsubscribe:
		if pending.subscription.State() == pending.previousState {
			return nil
		}
		return pending.subscription.SetState(pending.previousState)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, pending.method)
	}
}

// failWebsocketPending fails and rolls back every operation outstanding on a connection
func (e *Exchange) failWebsocketPending(authenticated bool, cause error) error {
	e.websocketPendingMu.Lock()
	pending := make([]*websocketPendingOperation, 0, len(e.websocketPending))
	for key, operation := range e.websocketPending {
		if key.authenticated == authenticated {
			delete(e.websocketPending, key)
			pending = append(pending, operation)
		}
	}
	e.websocketPendingMu.Unlock()

	result := cause
	for i := range pending {
		rollbackErr := e.rollbackWebsocketPending(pending[i])
		pending[i].done <- common.AppendError(cause, rollbackErr)
		result = common.AppendError(result, rollbackErr)
	}
	return result
}

// abortWebsocketPending fails an operation that could not be sent or timed out; if the acknowledgement handler already
// took the operation, its outcome is returned instead
func (e *Exchange) abortWebsocketPending(key *websocketPendingKey, pending *websocketPendingOperation, cause error) error {
	e.websocketPendingMu.Lock()
	if e.websocketPending[*key] != pending {
		e.websocketPendingMu.Unlock()
		return <-pending.done
	}
	delete(e.websocketPending, *key)
	e.websocketPendingMu.Unlock()
	return common.AppendError(cause, e.rollbackWebsocketPending(pending))
}

// wsHandleError settles the operation an error message answers, which the message identifies by quoting its
// subscription or request; any other error fails every operation pending on the connection
func (e *Exchange) wsHandleError(raw []byte, authenticated bool) error {
	var message string
	if err := json.Unmarshal(raw, &message); err != nil {
		message = strings.TrimSpace(string(raw))
	}
	if message == "" || message == "null" {
		message = "unspecified error"
	}
	cause := fmt.Errorf("%w: %s", errWebsocketServer, message)
	if quoted, ok := strings.CutPrefix(message, wsErrorAlreadySubscribed); ok {
		// The feed is active, so the subscribe succeeded, though without replaying recent trades
		if sub, err := decodeWebsocketSubscription(quoted); err == nil {
			if settled, err := e.completeWebsocketPending(authenticated, wsMethodSubscribe, &sub); settled {
				return err
			}
		}
		return cause
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorAlreadyUnsubscribed); ok {
		// The feed is inactive, so the unsubscribe succeeded
		if sub, err := decodeWebsocketSubscription(quoted); err == nil {
			if settled, err := e.completeWebsocketPending(authenticated, wsMethodUnsubscribe, &sub); settled {
				return err
			}
		}
		return cause
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorInvalidSubscription); ok {
		if sub, err := decodeWebsocketSubscription(quoted); err == nil && e.failWebsocketPendingOperation(authenticated, wsMethodSubscribe, &sub, cause) {
			return cause
		}
		return e.failWebsocketPending(authenticated, cause)
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorParseFailure); ok {
		var quotedRequest WsQuotedRequest
		if err := json.Unmarshal([]byte(quoted), &quotedRequest); err == nil {
			switch quotedRequest.Method {
			case wsMethodPost:
				response, err := json.Marshal(&WsPostResponse{ID: quotedRequest.ID, Response: WsPostResponseBody{Type: wsPostTypeError, Payload: raw}})
				if err == nil && e.Websocket.Match.IncomingWithData(quotedRequest.ID, response) {
					return nil
				}
				return cause
			case wsMethodSubscribe, wsMethodUnsubscribe:
				if e.failWebsocketPendingOperation(authenticated, quotedRequest.Method, &quotedRequest.Subscription, cause) {
					return cause
				}
			}
		}
	}
	return e.failWebsocketPending(authenticated, cause)
}

// decodeWebsocketSubscription decodes the canonical subscription an error message quotes
func decodeWebsocketSubscription(quoted string) (WsSubscription, error) {
	var sub WsSubscription
	if err := json.Unmarshal([]byte(quoted), &sub); err != nil {
		return WsSubscription{}, err
	}
	return sub, nil
}

// wsHandlePostResponse delivers a post response to the request awaiting its ID
func (e *Exchange) wsHandlePostResponse(raw []byte) error {
	var response WsPostResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	return e.Websocket.Match.RequireMatchWithData(response.ID, raw)
}

func (e *Exchange) wsHandleCandle(ctx context.Context, raw []byte) error {
	var update Candle
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	interval, err := parseInterval(update.Interval)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Asset:    a,
		Pair:     mapping.pair,
		Interval: interval,
		Candles:  []kline.Candle{convertCandle(&update)},
	})
}

// wsHandleOrderbook loads a book snapshot; the feed pushes full snapshots rather than incremental updates
func (e *Exchange) wsHandleOrderbook(ctx context.Context, raw []byte) error {
	var update WsL2Book
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	book, err := e.convertL2Book(&update.L2BookResponse, mapping.pair, a)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// wsHandleTrades relays the trades that convert and reports the rest; of the batch of recent trades Hyperliquid replays
// on subscribing, it relays only the trades a reconnect missed
func (e *Exchange) wsHandleTrades(raw []byte) error {
	var updates []RecentTrade
	if err := json.Unmarshal(raw, &updates); err != nil {
		return err
	}
	if len(updates) == 0 {
		e.consumeEmptyTradeReplay()
		return nil
	}
	updates = e.unrelayedTrades(updates)
	trades := make([]trade.Data, 0, len(updates))
	var errs error
	for i := range updates {
		mapping, a, err := e.lookupPairMappingByCoin(updates[i].Coin)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("trade %d: %w", i, err))
			continue
		}
		converted, err := e.convertTrade(&updates[i], mapping.pair, a)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("trade %d: %w", i, err))
			continue
		}
		trades = append(trades, converted)
	}
	if len(trades) > 0 {
		errs = common.AppendError(errs, e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), trades...))
	}
	return errs
}

// expectTradeReplay marks a coin whose trades subscription was acknowledged, as its next trades message replays recent
// trades
func (e *Exchange) expectTradeReplay(coin string) {
	e.websocketPendingMu.Lock()
	if e.websocketTradeReplays == nil {
		e.websocketTradeReplays = make(map[string]struct{})
	}
	e.websocketTradeReplays[coin] = struct{}{}
	e.websocketPendingMu.Unlock()
}

// consumeTradeReplay reports whether a coin's trades message is the replay its subscription expects, clearing the mark
func (e *Exchange) consumeTradeReplay(coin string) bool {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	if _, ok := e.websocketTradeReplays[coin]; !ok {
		return false
	}
	delete(e.websocketTradeReplays, coin)
	return true
}

// consumeEmptyTradeReplay clears the replay mark an empty trades message answers, which a market without recent trades
// replays; the message names no coin, but as each replay follows its acknowledgement, one mark at most awaits it
func (e *Exchange) consumeEmptyTradeReplay() {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	if len(e.websocketTradeReplays) == 1 {
		clear(e.websocketTradeReplays)
	}
}

// clearTradeReplay removes a coin's replay mark when its subscription is removed
func (e *Exchange) clearTradeReplay(coin string) {
	e.websocketPendingMu.Lock()
	delete(e.websocketTradeReplays, coin)
	e.websocketPendingMu.Unlock()
}

// clearTradeReplays removes every replay mark when the public connection closes, as the replays will not arrive
func (e *Exchange) clearTradeReplays() {
	e.websocketPendingMu.Lock()
	clear(e.websocketTradeReplays)
	e.websocketPendingMu.Unlock()
}

// unrelayedTrades returns the trades of a coin's message that were not relayed before, consuming the coin's replay mark
func (e *Exchange) unrelayedTrades(updates []RecentTrade) []RecentTrade {
	coin := updates[0].Coin
	replay := e.consumeTradeReplay(coin)
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	if e.websocketWatermarks.trades == nil {
		e.websocketWatermarks.trades = make(map[string]websocketPosition)
	}
	return unrelayedEntries(e.websocketWatermarks.trades, coin, updates, replay, func(update *RecentTrade) websocketPosition {
		return websocketPosition{time: update.Time.Time(), tradeID: update.TradeID}
	})
}

// unrelayedEntries returns the entries of a feed's message that were not relayed before and moves the feed's watermark to
// the newest entry: every entry of a live message, the entries of a replay after the watermark, and none of a replay
// that arrives before the feed has a watermark, as on its first subscription, since that replay predates it
func unrelayedEntries[K comparable, T any](watermarks map[K]websocketPosition, key K, entries []T, replay bool, position func(*T) websocketPosition) []T {
	watermark, ok := watermarks[key]
	newest := watermark
	unrelayed := entries
	if replay {
		unrelayed = make([]T, 0, len(entries))
	}
	for i := range entries {
		entry := position(&entries[i])
		if replay && ok && entry.after(watermark) {
			unrelayed = append(unrelayed, entries[i])
		}
		if entry.after(newest) {
			newest = entry
		}
	}
	watermarks[key] = newest
	return unrelayed
}

// clearWebsocketWatermark forgets the watermark of a feed that was unsubscribed, so subscribing it again relays no
// history, as a first subscription does
func (e *Exchange) clearWebsocketWatermark(feed *WsSubscription) {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	switch feed.Type {
	case wsChannelTrades:
		delete(e.websocketWatermarks.trades, feed.Coin)
	case wsChannelUserFills, wsChannelUserTWAPSliceFills:
		delete(e.websocketWatermarks.fills, websocketFillsFeed{channel: feed.Type, user: feed.User})
	}
}

// wsHandleOrderUpdates relays the updates that convert, with the cost and average price of the fills that cover an
// order's executed size, and reports the rest; orderUpdates sends a row only when an order's status changes, so a partial
// fill reaches the order stream from the fills feeds. Outcome orders are skipped, as no supported asset holds them
func (e *Exchange) wsHandleOrderUpdates(ctx context.Context, raw []byte) error {
	var updates []WsOrder
	if err := json.Unmarshal(raw, &updates); err != nil {
		return err
	}
	orders := make([]order.Detail, 0, len(updates))
	var errs error
	now := time.Now()
	for i := range updates {
		if isOutcomeCoin(updates[i].Order.Coin) {
			continue
		}
		// A position TP/SL order's rows hold zero sizes, as it closes the whole position when triggered; they are relayed as
		// converted, since the engine keeps the size it learnt from REST rather than take a zero amount
		detail, err := e.convertWsOrder(&updates[i])
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order update %d: %w", i, err))
			continue
		}
		e.applyWebsocketOrderRow(updates[i].Order.OrderID, &detail, now)
		orders = append(orders, detail)
	}
	if len(orders) > 0 {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, orders))
	}
	return errs
}

// applyWebsocketOrderRow records an order's row and completes the update relaying it: it adds the cost and average price
// of the row's executed size when the order's fills sum to it, or reports the order's cumulative execution when fills
// arrived ahead of the row. A final row forgets the order, unless the order's fills have yet to reach its executed size,
// as when the row is the first seen of an order that executed something
func (e *Exchange) applyWebsocketOrderRow(orderID uint64, update *order.Detail, now time.Time) {
	final := update.Status.IsInactive()
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	orders := &e.websocketWatermarks.orders
	if final && update.ExecutedAmount == 0 && orders.byOrderID[orderID] == nil {
		return
	}
	record := orders.touch(orderID, now)
	// Fills are deduplicated and positive, so fills summing to a row's executed size are all of the order's fills; a first
	// row showing nothing executed precedes every fill of the order, including fills that arrived ahead of it
	matched := math.Abs(record.executed-update.ExecutedAmount) <= record.sizeTolerance
	if matched || !record.rowSeen && update.ExecutedAmount == 0 {
		record.covered = true
	}
	record.rowSeen = true
	record.originalSize, record.limitPrice, record.placed = update.Amount, update.Price, update.Date
	switch {
	case matched:
		if update.ExecutedAmount > 0 {
			update.Cost, update.AverageExecutedPrice = record.cost, record.cost/record.executed
			record.setFee(update)
		}
	case record.executed > update.ExecutedAmount:
		if !final && record.covered && record.originalSize > 0 {
			record.setFillState(update)
		}
	case final:
		// The fills reaching the row's executed size will be all of the order's fills, even if the record misses earlier ones
		kept := *update
		record.final = &kept
		return
	}
	if final {
		orders.remove(record)
	}
}

// applyWebsocketOrderFill adds a fill to its order's record and returns the update relaying it to the order stream, or
// false for a fill already relayed. The update reports the order's cumulative execution only when the order's fills are
// known to include every execution and no final row awaits more of them, and otherwise only the trade, so it never
// lowers an executed size the engine learnt elsewhere
func (e *Exchange) applyWebsocketOrderFill(source *Fill, mapping *pairMapping, a asset.Item, side order.Side, now time.Time) (order.Detail, bool) {
	price, size, fee := source.Price.Float64(), source.Size.Float64(), source.Fee.Float64()
	filled := source.Time.Time().UTC()
	// An order not seen in a row yet is dated by its fill, which is when it was placed if its fill preceded its placement
	// row; the engine keeps the date it first learns of an order
	update := order.Detail{
		Exchange:      e.Name,
		OrderID:       strconv.FormatUint(source.OrderID, 10),
		ClientOrderID: source.ClientOrderID,
		Side:          side,
		AssetType:     a,
		Date:          filled,
		LastUpdated:   filled,
		Pair:          mapping.pair,
		Trades: []order.TradeHistory{{
			Price:     price,
			Amount:    size,
			Fee:       fee,
			Exchange:  e.Name,
			TID:       strconv.FormatUint(source.TradeID, 10),
			Side:      side,
			Timestamp: filled,
			IsMaker:   !source.Crossed,
			FeeAsset:  source.FeeToken.String(),
			Total:     price * size,
		}},
	}
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	orders := &e.websocketWatermarks.orders
	record := orders.touch(source.OrderID, now)
	if _, relayed := record.tradeIDs[source.TradeID]; relayed {
		return order.Detail{}, false
	}
	if record.tradeIDs == nil {
		record.tradeIDs = make(map[uint64]struct{})
	}
	record.tradeIDs[source.TradeID] = struct{}{}
	record.executed += size
	// Converting rounds the product first, so an architecture that fuses the multiply and add sums the same cost
	record.cost += float64(price * size)
	switch {
	case record.feeUnknown:
	case record.feeAsset.IsEmpty() || record.feeAsset.Equal(source.FeeToken):
		record.fee, record.feeAsset = record.fee+fee, source.FeeToken
	default:
		// A spot fill pays its fee in the token it receives but takes a maker rebate in the token it gives, so a spot
		// order's fees and rebates can be in different tokens, which one total cannot hold
		record.feeUnknown = true
	}
	// Sizes are whole multiples of the market's size step, so sums within half a step of each other are equal
	record.sizeTolerance = 0.5 / math.Pow(10, float64(mapping.sizeDecimals))
	if !record.placed.IsZero() {
		update.Date = record.placed
	}
	if record.final != nil {
		if math.Abs(record.executed-record.final.ExecutedAmount) > record.sizeTolerance {
			return update, true
		}
		final := *record.final
		final.Cost, final.AverageExecutedPrice = record.cost, record.cost/record.executed
		record.setFee(&final)
		final.Trades = update.Trades
		if filled.After(final.LastUpdated) {
			final.LastUpdated = filled
		}
		orders.remove(record)
		return final, true
	}
	if record.covered && record.originalSize > 0 {
		record.setFillState(&update)
	}
	return update, true
}

// setFillState makes an update report an order's cumulative execution from its fills, unless the fills exceed the order's
// size, as they then cannot be the fills its rows describe
func (r *websocketOrder) setFillState(update *order.Detail) {
	remaining := r.originalSize - r.executed
	if remaining < -r.sizeTolerance {
		return
	}
	update.Price, update.Amount = r.limitPrice, r.originalSize
	update.ExecutedAmount, update.Cost, update.AverageExecutedPrice = r.executed, r.cost, r.cost/r.executed
	r.setFee(update)
	update.Status, update.RemainingAmount = order.PartiallyFilled, remaining
	if remaining <= r.sizeTolerance {
		update.Status, update.RemainingAmount = order.Filled, 0
	}
}

// setFee makes an update report the fees of the order's fills, unless they were paid in more than one token
func (r *websocketOrder) setFee(update *order.Detail) {
	if !r.feeUnknown {
		update.Fee, update.FeeAsset = r.fee, r.feeAsset
	}
}

// uncoverWebsocketOrders marks every order's fills as possibly incomplete, so no update reports an order's cumulative
// execution until a row shows its fills cover its executed size again
func (e *Exchange) uncoverWebsocketOrders() {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	for record := e.websocketWatermarks.orders.oldest; record != nil; record = record.newer {
		record.covered = false
	}
}

// touch returns an order's record, starting a new one when it is absent or untouched for its lifetime, and marks it
// touched now, then forgets the least recently touched records beyond the limit and those untouched for their lifetime
func (o *websocketOrders) touch(orderID uint64, now time.Time) *websocketOrder {
	record := o.byOrderID[orderID]
	if record != nil && now.Sub(record.touched) > websocketOrderLifetime {
		o.remove(record)
		record = nil
	}
	if record == nil {
		if o.byOrderID == nil {
			o.byOrderID = make(map[uint64]*websocketOrder)
		}
		record = &websocketOrder{orderID: orderID}
		o.byOrderID[orderID] = record
	} else {
		o.unlink(record)
	}
	record.touched = now
	record.older = o.newest
	if o.newest != nil {
		o.newest.newer = record
	} else {
		o.oldest = record
	}
	o.newest = record
	for o.oldest != record && (len(o.byOrderID) > maximumWebsocketOrders || now.Sub(o.oldest.touched) > websocketOrderLifetime) {
		o.remove(o.oldest)
	}
	return record
}

// remove forgets an order's record
func (o *websocketOrders) remove(record *websocketOrder) {
	o.unlink(record)
	delete(o.byOrderID, record.orderID)
}

// unlink takes a record out of the order in which records were touched
func (o *websocketOrders) unlink(record *websocketOrder) {
	if record.older != nil {
		record.older.newer = record.newer
	} else {
		o.oldest = record.newer
	}
	if record.newer != nil {
		record.newer.older = record.older
	} else {
		o.newest = record.older
	}
	record.older, record.newer = nil, nil
}

// wsHandleUserEvent relays a user event: fills and TWAP slice fills as trades of their orders and as fills, and the
// other events as themselves
func (e *Exchange) wsHandleUserEvent(ctx context.Context, raw []byte) error {
	var event WsUserEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return err
	}
	switch {
	case len(event.Fills) > 0:
		return e.relayFills(ctx, event.Fills, nil, true)
	case len(event.TWAPSliceFills) > 0:
		return e.relayFills(ctx, nil, event.TWAPSliceFills, true)
	case event.Funding != nil:
		return e.Websocket.DataHandler.Send(ctx, event.Funding)
	case event.Liquidation != nil:
		return e.Websocket.DataHandler.Send(ctx, event.Liquidation)
	case len(event.NonUserCancel) > 0:
		return e.Websocket.DataHandler.Send(ctx, event.NonUserCancel)
	default:
		return nil
	}
}

// wsHandleUserFills relays the fills that convert and reports the rest; of the snapshot sent on each subscription, it
// relays only the fills a reconnect missed
func (e *Exchange) wsHandleUserFills(ctx context.Context, raw []byte) error {
	var update WsUserFills
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	feed := websocketFillsFeed{channel: wsChannelUserFills, user: strings.ToLower(update.User)}
	fills := unrelayedFills(e, feed, update.Fills, update.IsSnapshot, func(f *Fill) websocketPosition {
		return websocketPosition{time: f.Time.Time(), tradeID: f.TradeID}
	})
	return e.relayFills(ctx, fills, nil, !e.aggregatesUserFills(feed.user))
}

// aggregatesUserFills reports whether the userFills feed held for a user combines each order's fills within a block; a
// combined fill carries the trade ID of one of them, so it cannot be added up with the fills userEvents sends
func (e *Exchange) aggregatesUserFills(user string) bool {
	for _, sub := range e.Websocket.GetSubscriptions() {
		if key, ok := sub.EnsureKeyed().(*websocketSubscriptionKey); ok && key.feed.Type == wsChannelUserFills && key.feed.User == user {
			return key.feed.AggregateByTime
		}
	}
	return false
}

// unrelayedFills returns the fills of an account fills feed's message that were not relayed before
func unrelayedFills[T any](e *Exchange, feed websocketFillsFeed, fills []T, snapshot bool, position func(*T) websocketPosition) []T {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	if e.websocketWatermarks.fills == nil {
		e.websocketWatermarks.fills = make(map[websocketFillsFeed]websocketPosition)
	}
	return unrelayedEntries(e.websocketWatermarks.fills, feed, fills, snapshot, position)
}

// relayFills relays fills and TWAP slice fills as trades of their orders to the order stream, when toOrders is set, and
// to the fills feed, when it is enabled, reporting those whose coin or side cannot convert; outcome fills are skipped, as
// no supported asset holds them
func (e *Exchange) relayFills(ctx context.Context, fills []Fill, sliceFills []TWAPSliceFill, toOrders bool) error {
	toFillsFeed := e.IsFillsFeedEnabled()
	if !toOrders && !toFillsFeed {
		return nil
	}
	orders := make([]order.Detail, 0, len(fills)+len(sliceFills))
	data := make([]fill.Data, 0, len(fills)+len(sliceFills))
	now := time.Now()
	relay := func(source *Fill) error {
		if isOutcomeCoin(source.Coin) {
			return nil
		}
		mapping, a, err := e.lookupPairMappingByCoin(source.Coin)
		if err != nil {
			return err
		}
		side, err := parseSide(source.Side)
		if err != nil {
			return err
		}
		if toOrders {
			if update, ok := e.applyWebsocketOrderFill(source, &mapping, a, side, now); ok {
				orders = append(orders, update)
			}
		}
		if toFillsFeed {
			data = append(data, e.convertFill(source, mapping.pair, a, side))
		}
		return nil
	}
	var errs error
	for i := range fills {
		if err := relay(&fills[i]); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("fill %d: %w", i, err))
		}
	}
	for i := range sliceFills {
		if err := relay(&sliceFills[i].Fill); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("TWAP %d slice fill %d: %w", sliceFills[i].TWAPID, i, err))
		}
	}
	if len(orders) > 0 {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, orders))
	}
	if len(data) > 0 {
		errs = common.AppendError(errs, e.Websocket.Fills.Update(data...))
	}
	return errs
}

// convertFill converts a fill of a market, identified by its trade ID
func (e *Exchange) convertFill(source *Fill, p currency.Pair, a asset.Item, side order.Side) fill.Data {
	tradeID := strconv.FormatUint(source.TradeID, 10)
	return fill.Data{
		ID:            tradeID,
		Timestamp:     source.Time.Time().UTC(),
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  p,
		Side:          side,
		OrderID:       strconv.FormatUint(source.OrderID, 10),
		ClientOrderID: source.ClientOrderID,
		TradeID:       tradeID,
		Price:         source.Price.Float64(),
		Amount:        source.Size.Float64(),
	}
}

// wsHandleUserFundings relays funding payments; Hyperliquid can send a message without payments, which is dropped
func (e *Exchange) wsHandleUserFundings(ctx context.Context, raw []byte) error {
	update := new(WsUserFundings)
	if err := json.Unmarshal(raw, update); err != nil {
		return err
	}
	if len(update.Fundings) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, update)
}

// wsHandleUserNonFundingLedgerUpdates relays ledger updates; Hyperliquid sends a message without updates at each
// funding payment, which is dropped
func (e *Exchange) wsHandleUserNonFundingLedgerUpdates(ctx context.Context, raw []byte) error {
	update := new(WsUserNonFundingLedgerUpdates)
	if err := json.Unmarshal(raw, update); err != nil {
		return err
	}
	if len(update.NonFundingLedgerUpdates) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, update)
}

func (e *Exchange) wsHandlePerpetualTicker(ctx context.Context, raw []byte) error {
	var update WsPerpetualAssetContext
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	if a != asset.PerpetualContract {
		return fmt.Errorf("%w: expected %s, got %s for %s", errWebsocketAssetMismatch, asset.PerpetualContract, a, update.Coin)
	}
	price := e.perpetualTickerPrice(mapping.pair, &update.Context)
	if err := ticker.ProcessTicker(price); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, price)
}

func (e *Exchange) wsHandleSpotTicker(ctx context.Context, raw []byte) error {
	var update WsSpotAssetContext
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	if a != asset.Spot {
		return fmt.Errorf("%w: expected %s, got %s for %s", errWebsocketAssetMismatch, asset.Spot, a, update.Coin)
	}
	price := e.spotTickerPrice(mapping.pair, &update.Context)
	if err := ticker.ProcessTicker(price); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, price)
}

// wsHandleUserTWAPSliceFills relays TWAP slice fills as trades of their orders and as fills; of the snapshot sent on
// each subscription, it relays only the slice fills a reconnect missed
func (e *Exchange) wsHandleUserTWAPSliceFills(ctx context.Context, raw []byte) error {
	var update WsUserTWAPSliceFills
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	feed := websocketFillsFeed{channel: wsChannelUserTWAPSliceFills, user: strings.ToLower(update.User)}
	sliceFills := unrelayedFills(e, feed, update.TWAPSliceFills, update.IsSnapshot, func(f *TWAPSliceFill) websocketPosition {
		return websocketPosition{time: f.Fill.Time.Time(), tradeID: f.Fill.TradeID}
	})
	return e.relayFills(ctx, nil, sliceFills, true)
}

// wsHandleUserTWAPHistory relays TWAP status changes, dropping a message without changes
func (e *Exchange) wsHandleUserTWAPHistory(ctx context.Context, raw []byte) error {
	update := new(WsUserTWAPHistory)
	if err := json.Unmarshal(raw, update); err != nil {
		return err
	}
	if len(update.History) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, update)
}

// wsHandleOutcomeMetaUpdates relays outcome and question changes, dropping a message without changes
func (e *Exchange) wsHandleOutcomeMetaUpdates(ctx context.Context, raw []byte) error {
	var updates []WsOutcomeMetaUpdate
	if err := json.Unmarshal(raw, &updates); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, updates)
}

// wsHandleFastAssetContexts relays the mark and mid prices of every coin, which arrive as a base64 string of raw
// DEFLATE compressed JSON; the first message after subscribing holds every coin and later messages only the changes
func (e *Exchange) wsHandleFastAssetContexts(ctx context.Context, raw []byte) error {
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return err
	}
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()
	decompressed, err := io.ReadAll(io.LimitReader(reader, maximumFastAssetContextsSize+1))
	if err != nil {
		return err
	}
	if len(decompressed) > maximumFastAssetContextsSize {
		return fmt.Errorf("%w: fastAssetCtxs message exceeds %d bytes", errWebsocketServer, maximumFastAssetContextsSize)
	}
	var contexts map[string]WsFastAssetContext
	if err := json.Unmarshal(decompressed, &contexts); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, contexts)
}

// WsPostInfo sends an info request body, such as an InfoRequest, over the public websocket connection and decodes the
// info endpoint's response into result
func (e *Exchange) WsPostInfo(ctx context.Context, payload, result any) error {
	if payload == nil || result == nil {
		return common.ErrNilPointer
	}
	response, err := e.sendWebsocketPost(ctx, wsPostTypeInfo, payload)
	if err != nil {
		return err
	}
	var info WsPostInfoPayload
	if err := json.Unmarshal(response, &info); err != nil {
		return fmt.Errorf("decoding info post payload: %w", err)
	}
	if err := json.Unmarshal(info.Data, result); err != nil {
		return fmt.Errorf("decoding %s info response: %w", info.Type, err)
	}
	return nil
}

// WsPlaceOrders places a batch of orders as PlaceOrders does, posting the order action over the websocket
func (e *Exchange) WsPlaceOrders(ctx context.Context, arg *PlaceOrdersRequest) ([]OrderActionStatus, error) {
	return e.placeOrders(ctx, arg, true)
}

// WsCancelOrders cancels a batch of orders by order ID as CancelOrders does, posting the cancel action over the websocket
func (e *Exchange) WsCancelOrders(ctx context.Context, arg *CancelOrdersRequest) ([]CancelActionStatus, error) {
	return e.cancelOrders(ctx, arg, true)
}

// WsCancelOrdersByClientOrderID cancels a batch of orders by client order ID as CancelOrdersByClientOrderID does,
// posting the cancel action over the websocket
func (e *Exchange) WsCancelOrdersByClientOrderID(ctx context.Context, arg *CancelOrdersByClientOrderIDRequest) ([]CancelActionStatus, error) {
	return e.cancelOrdersByClientOrderID(ctx, arg, true)
}

// WsModifySingleOrder replaces one order as ModifySingleOrder does, posting the modify action over the websocket
func (e *Exchange) WsModifySingleOrder(ctx context.Context, arg *ModifyOrderRequest) error {
	return e.modifySingleOrder(ctx, arg, true)
}

// WsModifyOrders replaces a batch of orders as ModifyOrders does, posting the batch modify action over the websocket
func (e *Exchange) WsModifyOrders(ctx context.Context, arg *ModifyOrdersRequest) ([]OrderActionStatus, error) {
	return e.modifyOrders(ctx, arg, true)
}

// sendWebsocketAction posts a signed action over the public websocket connection and unwraps its status envelope
func (e *Exchange) sendWebsocketAction(ctx context.Context, payload *SignedActionRequest) (json.RawMessage, error) {
	response, err := e.sendWebsocketPost(ctx, wsPostTypeAction, payload)
	if err != nil {
		return nil, err
	}
	var resp *ExchangeActionResponse
	if err := json.Unmarshal(response, &resp); err != nil {
		return nil, fmt.Errorf("decoding action response: %w", err)
	}
	return actionResponseData(resp)
}

// sendWebsocketPost posts an info request or a signed action, as requestType says, and returns its response payload; a
// rejected request returns the server's error
func (e *Exchange) sendWebsocketPost(ctx context.Context, requestType string, payload any) (json.RawMessage, error) {
	connection := e.Websocket.Conn
	if connection == nil {
		return nil, websocket.ErrNotConnected
	}
	id := e.websocketPostID.Add(1)
	raw, err := connection.SendMessageReturnResponse(ctx, request.Unset, id, &WsPostRequest{
		Method:  wsMethodPost,
		ID:      id,
		Request: WsPostRequestBody{Type: requestType, Payload: payload},
	})
	if err != nil {
		return nil, fmt.Errorf("posting %s request: %w", requestType, err)
	}
	var response WsPostResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decoding post response: %w", err)
	}
	switch response.Response.Type {
	case requestType:
		return response.Response.Payload, nil
	case wsPostTypeError:
		var message string
		if err := json.Unmarshal(response.Response.Payload, &message); err != nil {
			message = string(response.Response.Payload)
		}
		return nil, fmt.Errorf("%w: %s", errWebsocketPost, message)
	default:
		return nil, fmt.Errorf("%w: unexpected response type %q", errWebsocketPost, response.Response.Type)
	}
}

// websocketChannelName returns the Hyperliquid subscription type a subscription's channel refers to: a standard
// channel's type, or the type the channel names
func websocketChannelName(sub *subscription.Subscription) (string, error) {
	if sub == nil {
		return "", common.ErrNilPointer
	}
	if name, ok := websocketStandardChannels[sub.Channel]; ok {
		return name, nil
	}
	if _, ok := websocketFeeds[sub.Channel]; ok {
		return sub.Channel, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, sub.Channel)
}

// GetSubscriptionTemplate returns the channel qualification template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName": websocketChannelName,
		"perPair":     func(name string) bool { return websocketFeeds[name].coin },
		"interval": func(value kline.Interval) string {
			formatted, _ := formatInterval(value) // Subscription validation rejects an interval Hyperliquid does not serve
			return formatted
		},
	}).Parse(websocketSubscriptionTemplate)
}

// generateSubscriptions expands the configured subscriptions and keys each by the feed it selects; the configuration
// replaces the held subscriptions, so it is validated on its own rather than beside the feeds a flush is about to remove
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := e.Features.Subscriptions.ExpandTemplates(websocketTemplateExpander{e})
	if validationErr := e.ValidateSubscriptions(subs); validationErr != nil {
		return nil, common.AppendError(err, validationErr)
	}
	keyWebsocketSubscriptions(subs)
	return subs, err
}

// keyWebsocketSubscriptions keys each subscription by the feed it selects, as the manager finds held subscriptions by the
// keys of the subscriptions it is given
func keyWebsocketSubscriptions(subs subscription.List) {
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		// A subscription keyed by its feed may be held already, and re-keying it would strand its entry in the store
		if _, keyed := sub.EnsureKeyed().(*websocketSubscriptionKey); !keyed {
			sub.SetKey(newWebsocketSubscriptionKey(sub))
		}
	}
}

// SubscribeToWebsocketChannels subscribes to feeds through the websocket manager, keying each by the feed it selects
// first, as the manager refuses a subscription matching a held one and would otherwise compare them without their
// parameters
func (e *Exchange) SubscribeToWebsocketChannels(channels subscription.List) error {
	if slices.Contains(channels, nil) {
		return common.ErrNilPointer
	}
	keyWebsocketSubscriptions(channels)
	return e.Base.SubscribeToWebsocketChannels(channels)
}

// UnsubscribeToWebsocketChannels unsubscribes from feeds through the websocket manager, keying each by the feed it selects
// first, as the manager looks each up among the held subscriptions and would otherwise compare them without their
// parameters
func (e *Exchange) UnsubscribeToWebsocketChannels(channels subscription.List) error {
	if slices.Contains(channels, nil) {
		return common.ErrNilPointer
	}
	keyWebsocketSubscriptions(channels)
	return e.Base.UnsubscribeToWebsocketChannels(channels)
}

// newWebsocketSubscriptionKey keys a subscription by the feed it selects
func newWebsocketSubscriptionKey(sub *subscription.Subscription) *websocketSubscriptionKey {
	options, err := websocketFeedOptions(sub)
	return &websocketSubscriptionKey{Subscription: sub, options: options, valid: err == nil}
}

// ValidateSubscriptions refuses two variants of a feed whose messages do not say which variant sent them: two l2Book
// variants of one market would replace each other's book in the shared store, and two userFills variants would relay
// every fill twice, once combined with the order's other fills of its block
func (e *Exchange) ValidateSubscriptions(subs subscription.List) error {
	var books []*websocketSubscriptionKey
	var userFills *websocketSubscriptionKey
	for _, sub := range subs {
		variant := newWebsocketSubscriptionKey(sub)
		if !variant.valid {
			// Subscribing reports the subscription's error
			continue
		}
		switch variant.options.Type {
		case wsChannelOrderbook:
			for _, first := range books {
				if first.options != variant.options && first.Asset == sub.Asset && slices.ContainsFunc(sub.Pairs, func(p currency.Pair) bool { return first.Pairs.Contains(p, true) }) {
					return fmt.Errorf("%w: %s and %s", subscription.ErrExclusiveSubscription, first, variant)
				}
			}
			books = append(books, variant)
		case wsChannelUserFills:
			if userFills != nil && userFills.options != variant.options {
				return fmt.Errorf("%w: %s and %s", subscription.ErrExclusiveSubscription, userFills, variant)
			}
			userFills = variant
		}
	}
	return nil
}

// Subscribe subscribes to feeds, completing only once the server acknowledges each one
func (e *Exchange) Subscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	if slices.Contains(subscriptions, nil) {
		return common.ErrNilPointer
	}
	expanded, errs := subscriptions.ExpandTemplates(e)
	for _, sub := range expanded {
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, wsMethodSubscribe, sub))
	}
	return errs
}

// Unsubscribe unsubscribes from feeds, completing only once the server acknowledges each one; a subscription is found by
// the feed it selects, so unsubscribing a variant that is not held leaves the held variants subscribed
func (e *Exchange) Unsubscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	if slices.Contains(subscriptions, nil) {
		return common.ErrNilPointer
	}
	expanded, errs := subscriptions.ExpandTemplates(websocketTemplateExpander{e})
	for _, sub := range expanded {
		feed := newWebsocketSubscriptionKey(sub)
		held := e.Websocket.GetSubscription(feed)
		if held == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrNotFound, feed))
			continue
		}
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, wsMethodUnsubscribe, held))
	}
	return errs
}

// manageWebsocketSubscription sends a subscription operation and waits for its acknowledgement, rolling the local
// subscription state back if the server rejects it, the connection drops or it times out
// An unsubscribe repeats the feed its subscription subscribed, as Hyperliquid requires, rather than rebuilding it from
// metadata that no longer lists a delisted market or from a configured address that has since changed
func (e *Exchange) manageWebsocketSubscription(ctx context.Context, method string, sub *subscription.Subscription) error {
	if sub == nil {
		return common.ErrNilPointer
	}
	var payload WsSubscription
	switch method {
	case wsMethodSubscribe:
		var err error
		if payload, err = e.websocketSubscriptionPayload(ctx, sub); err != nil {
			return err
		}
	case wsMethodUnsubscribe:
		key, ok := sub.EnsureKeyed().(*websocketSubscriptionKey)
		if !ok {
			return fmt.Errorf("%w: %s", errWebsocketFeedUnknown, sub)
		}
		payload = key.feed
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, method)
	}
	connection := e.Websocket.Conn
	if sub.Authenticated {
		connection = e.Websocket.AuthConn
	}
	if connection == nil {
		return common.ErrNilPointer
	}
	previousState := sub.State()
	if method == wsMethodSubscribe {
		feed := newWebsocketSubscriptionKey(sub)
		feed.feed = payload
		// The store files a subscription under its key, so re-keying a stored subscription would strand its entry
		if e.Websocket.GetSubscription(feed) != nil {
			return fmt.Errorf("%w: %s", subscription.ErrDuplicate, feed)
		}
		sub.SetKey(feed)
		if err := e.Websocket.AddSubscriptions(connection, sub); err != nil {
			return err
		}
	} else if err := sub.SetState(subscription.UnsubscribingState); err != nil {
		return err
	}
	key := websocketPendingKey{authenticated: sub.Authenticated, subscription: payload}
	pending := &websocketPendingOperation{
		method:        method,
		connection:    connection,
		subscription:  sub,
		previousState: previousState,
		done:          make(chan error, 1),
	}
	e.websocketPendingMu.Lock()
	if e.websocketPending == nil {
		e.websocketPending = make(map[websocketPendingKey]*websocketPendingOperation)
	}
	if e.websocketPending[key] != nil {
		e.websocketPendingMu.Unlock()
		return common.AppendError(fmt.Errorf("%w: operation already pending for %s", errWebsocketSubscription, sub), e.rollbackWebsocketPending(pending))
	}
	e.websocketPending[key] = pending
	e.websocketPendingMu.Unlock()

	err := e.awaitWebsocketPending(ctx, connection, &key, pending, &WsSubscriptionRequest{Method: method, Subscription: payload})
	if err == nil && method == wsMethodUnsubscribe {
		e.clearWebsocketWatermark(&payload)
	}
	return err
}

// awaitWebsocketPending sends a subscription request and waits for the server to settle its pending operation
func (e *Exchange) awaitWebsocketPending(ctx context.Context, connection websocket.Connection, key *websocketPendingKey, pending *websocketPendingOperation, req *WsSubscriptionRequest) error {
	if err := connection.SendJSONMessage(ctx, request.Unset, req); err != nil {
		return e.abortWebsocketPending(key, pending, fmt.Errorf("sending %s request: %w", req.Method, err))
	}
	timer := time.NewTimer(e.WebsocketResponseMaxLimit)
	defer timer.Stop()
	select {
	case err := <-pending.done:
		return err
	case <-ctx.Done():
		return e.abortWebsocketPending(key, pending, ctx.Err())
	case <-timer.C:
		return e.abortWebsocketPending(key, pending, fmt.Errorf("%w: %s", websocket.ErrSignatureTimeout, pending.subscription))
	}
}

// websocketSubscriptionPayload builds the feed a subscription refers to, validating every coin, DEX and option first,
// because Hyperliquid closes the connection, dropping every feed on it, for a subscription it cannot validate
func (e *Exchange) websocketSubscriptionPayload(ctx context.Context, sub *subscription.Subscription) (WsSubscription, error) {
	payload, err := websocketFeedOptions(sub)
	if err != nil {
		return WsSubscription{}, err
	}
	feed := websocketFeeds[payload.Type]
	if sub.Authenticated != feed.user {
		return WsSubscription{}, fmt.Errorf("%w: %s must set authenticated to %t", subscription.ErrNotSupported, payload.Type, feed.user)
	}
	if feed.user {
		if payload.User, err = e.getWatchAddress(ctx); err != nil {
			return WsSubscription{}, err
		}
	}
	switch {
	case feed.coin:
		if len(sub.Pairs) != 1 {
			return WsSubscription{}, subscription.ErrNotSinglePair
		}
		if payload.Type == wsChannelActiveAssetData && sub.Asset != asset.PerpetualContract {
			return WsSubscription{}, fmt.Errorf("%w: %s serves perpetual contracts only", asset.ErrNotSupported, payload.Type)
		}
		mapping, err := e.getPairMapping(ctx, sub.Pairs[0], sub.Asset)
		if err != nil {
			return WsSubscription{}, err
		}
		payload.Coin = mapping.coin
	case sub.Asset != asset.Empty:
		return WsSubscription{}, fmt.Errorf("%w: %s does not take an asset", subscription.ErrNotSupported, payload.Type)
	}
	if payload.DEX != "" {
		if err := e.validateWebsocketDEX(ctx, payload.DEX); err != nil {
			return WsSubscription{}, err
		}
	}
	return payload, nil
}

// websocketFeedOptions returns the Hyperliquid subscription type a subscription refers to and the options its interval,
// levels and parameters select, as its payload sends them; it reads only what the type takes, as Hyperliquid would
// ignore the rest, and leaves the coin and user, which the market and account supply
func websocketFeedOptions(sub *subscription.Subscription) (WsSubscription, error) {
	name, err := websocketChannelName(sub)
	if err != nil {
		return WsSubscription{}, err
	}
	options := WsSubscription{Type: name}
	if websocketFeeds[name].dex {
		if options.DEX, err = websocketDEXName(sub.Params); err != nil {
			return WsSubscription{}, err
		}
	}
	switch name {
	case wsChannelCandle:
		options.Interval, err = formatInterval(sub.Interval)
	case wsChannelOrderbook:
		err = websocketOrderbookOptions(sub, &options)
	case wsChannelUserFills:
		options.AggregateByTime, err = websocketBoolParameter(sub.Params, "aggregateByTime")
	case wsChannelSpotState:
		options.IgnorePortfolioMargin, err = websocketBoolParameter(sub.Params, "ignorePortfolioMargin")
	}
	if err != nil {
		return WsSubscription{}, err
	}
	return options, nil
}

// websocketOrderbookOptions sets an l2Book's options: 5 levels select the fast book, and the nSigFigs and mantissa
// parameters aggregate it
func websocketOrderbookOptions(sub *subscription.Subscription, payload *WsSubscription) error {
	switch sub.Levels {
	case 0, 20:
	case 5:
		payload.Fast = true
	default:
		return fmt.Errorf("%w: %d, as l2Book sends 20 levels or 5 fast levels", subscription.ErrInvalidLevel, sub.Levels)
	}
	var err error
	if payload.NumberOfSignificantFigures, err = websocketUintParameter(sub.Params, "nSigFigs"); err != nil {
		return err
	}
	if payload.Mantissa, err = websocketUintParameter(sub.Params, "mantissa"); err != nil {
		return err
	}
	return validateBookAggregation(payload.NumberOfSignificantFigures, payload.Mantissa)
}

// websocketDEXName returns a subscription's dex parameter, which names a perpetual DEX; an absent or empty parameter is
// the first DEX
func websocketDEXName(params map[string]any) (string, error) {
	switch value := params["dex"].(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	default:
		return "", fmt.Errorf("%w: dex %v", errWebsocketParameter, value)
	}
}

// validateWebsocketDEX checks a builder-deployed perpetual DEX is registered
func (e *Exchange) validateWebsocketDEX(ctx context.Context, dex string) error {
	names, err := e.getPerpetualDEXNames(ctx)
	if err != nil {
		return err
	}
	// The first DEX's name is empty, which a builder-deployed DEX's name never is
	if !slices.Contains(names, dex) {
		return fmt.Errorf("%w: %q", errInvalidPerpetualDEX, dex)
	}
	return nil
}

// websocketUintParameter returns a non-negative whole number subscription parameter, which configuration supplies as a
// JSON number; an absent parameter is zero
func websocketUintParameter(params map[string]any, name string) (uint64, error) {
	switch value := params[name].(type) {
	case nil:
		return 0, nil
	case uint64:
		return value, nil
	case int:
		if value >= 0 {
			return uint64(value), nil
		}
	case float64:
		if value >= 0 && value <= math.MaxUint32 && value == math.Trunc(value) {
			return uint64(value), nil
		}
	}
	return 0, fmt.Errorf("%w: %s %v", errWebsocketParameter, name, params[name])
}

// websocketBoolParameter returns a boolean subscription parameter; an absent parameter is false
func websocketBoolParameter(params map[string]any, name string) (bool, error) {
	switch value := params[name].(type) {
	case nil:
		return false, nil
	case bool:
		return value, nil
	default:
		return false, fmt.Errorf("%w: %s %v", errWebsocketParameter, name, value)
	}
}

// websocketSubscriptionTemplate qualifies a subscription per pair for market feeds and once otherwise
const websocketSubscriptionTemplate = `
{{- $name := channelName $.S }}
{{- if $.S.Asset }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if perPair $name }}
			{{- range $pair := $pairs }}
				{{- $name -}} : {{- $asset -}} : {{- $pair -}}
				{{- if $.S.Interval -}} : {{- interval $.S.Interval -}}{{- end -}}
				{{ $.PairSeparator }}
			{{- end }}
		{{- else }}
			{{- $name -}} : {{- $asset }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else -}}
	{{ $name }}
{{- end }}
`
