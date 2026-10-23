package hyperliquid

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	// websocketHandshakeTimeout bounds a dial, which holds the manager's lock while it retries the account connection
	websocketHandshakeTimeout = 10 * time.Second
	// websocketMessageInterval spaces requests within the limit of 2000 messages a minute
	websocketMessageInterval = 30 * time.Millisecond
	// maximumWebsocketSubscriptions is Hyperliquid's per-IP subscription limit, which the public and account
	// connections' subscriptions count against together
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

// websocketConnectionRole is the message filter of a connection setup
type websocketConnectionRole string

// The public connection carries market feeds and WebsocketAPI posts, and the account connection the account feeds
const (
	websocketPublicConnection  websocketConnectionRole = "public"
	websocketAccountConnection websocketConnectionRole = "account"
)

var (
	errWebsocketAssetMismatch        = errors.New("websocket market asset mismatch")
	errWebsocketFeedUnknown          = errors.New("websocket subscription has no subscribed feed")
	errWebsocketParameter            = errors.New("invalid websocket subscription parameter")
	errWebsocketPost                 = errors.New("websocket post request failed")
	errWebsocketServer               = errors.New("websocket server error")
	errWebsocketSubscription         = errors.New("websocket subscription acknowledgement error")
	errWebsocketSubscriptionsLimit   = errors.New("websocket subscriptions would exceed Hyperliquid's limit of 1000 for an IP address")
	errWebsocketAccountFeedsDisabled = errors.New("authenticated websocket support is disabled")
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

// connectPublicWebsocket dials the public connection, which carries the market feeds and WebsocketAPI posts
func (e *Exchange) connectPublicWebsocket(ctx context.Context, conn websocket.Connection) error {
	if err := e.connectWebsocket(ctx, conn); err != nil {
		return err
	}
	e.websocketConnectionsMu.Lock()
	e.websocketPostConnection = conn
	e.websocketConnectionsMu.Unlock()
	return nil
}

// connectAccountWebsocket dials the account connection. Hyperliquid's account feeds are public and keyed by address, so
// authenticated websocket use only means this connection is up
func (e *Exchange) connectAccountWebsocket(ctx context.Context, conn websocket.Connection) error {
	if err := e.connectWebsocket(ctx, conn); err != nil {
		return err
	}
	e.websocketConnectionsMu.Lock()
	e.websocketAccountConnection = conn
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	e.websocketConnectionsMu.Unlock()
	return nil
}

func (e *Exchange) connectWebsocket(ctx context.Context, connection websocket.Connection) error {
	if !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if connection == nil {
		return common.ErrNilPointer
	}
	if err := connection.Dial(ctx, &gws.Dialer{HandshakeTimeout: websocketHandshakeTimeout}, http.Header{}, nil); err != nil {
		return fmt.Errorf("dialling %s: %w", connection.GetURL(), err)
	}
	connection.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"method":"ping"}`),
		Delay:       websocketPingInterval,
	})
	return nil
}

// publicWebsocketDisconnected fails the public connection's outstanding subscription operations; when it is the current
// public connection, WebsocketAPI posts stop using it and the trade replays it awaited are no longer expected
func (e *Exchange) publicWebsocketDisconnected(_ context.Context, conn websocket.Connection) {
	_ = e.failWebsocketPending(conn, websocket.ErrNotConnected)
	e.websocketConnectionsMu.Lock()
	current := e.websocketPostConnection == conn
	if current {
		e.websocketPostConnection = nil
	}
	e.websocketConnectionsMu.Unlock()
	if current {
		e.clearTradeReplays()
	}
}

// accountWebsocketDisconnected fails the account connection's outstanding subscription operations; when it is the
// current account connection, account feeds stop, and as userEvents does not resend the fills sent while it is down, the
// orders' fills may then be incomplete
func (e *Exchange) accountWebsocketDisconnected(_ context.Context, conn websocket.Connection) {
	_ = e.failWebsocketPending(conn, websocket.ErrNotConnected)
	e.websocketConnectionsMu.Lock()
	current := e.websocketAccountConnection == conn
	if current {
		e.websocketAccountConnection = nil
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	e.websocketConnectionsMu.Unlock()
	if current {
		e.uncoverWebsocketOrders()
	}
}

// wsHandleData handles a message, settling subscription operations on the connection that sent them; an error names
// the channel whose message failed, as the data handler relays it without further context
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, raw []byte) error {
	if string(raw) == websocketConnectionEstablished {
		return nil
	}
	var envelope WebsocketMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("%s decoding websocket message: %w", e.Name, err)
	}
	if err := e.wsHandleChannel(ctx, conn, &envelope, raw); err != nil {
		return fmt.Errorf("%s %s message: %w", e.Name, envelope.Channel, err)
	}
	return nil
}

// wsHandleChannel passes a message to its channel's handler, relaying a message of an unknown channel as unhandled
func (e *Exchange) wsHandleChannel(ctx context.Context, conn websocket.Connection, envelope *WebsocketMessage, raw []byte) error {
	switch envelope.Channel {
	case wsChannelSubscriptionResponse:
		return e.wsHandleSubscriptionResponse(conn, envelope.Data)
	case wsChannelPost:
		return e.wsHandlePostResponse(conn, envelope.Data)
	case wsChannelPong:
		return nil
	case wsChannelError:
		return e.wsHandleError(conn, envelope.Data)
	case wsChannelAllMids:
		return wsRelay[WebsocketAllMids](ctx, e, envelope.Data)
	case wsChannelNotification:
		return wsRelay[WebsocketNotification](ctx, e, envelope.Data)
	case wsChannelWebData3:
		return wsRelay[WebsocketWebData3](ctx, e, envelope.Data)
	case wsChannelTWAPStates:
		return wsRelay[WebsocketTWAPStates](ctx, e, envelope.Data)
	case wsChannelClearinghouseState:
		return wsRelay[WebsocketClearinghouseState](ctx, e, envelope.Data)
	case wsChannelOpenOrders:
		return wsRelay[WebsocketOpenOrders](ctx, e, envelope.Data)
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
		return wsRelay[WebsocketBestBidOffer](ctx, e, envelope.Data)
	case wsChannelSpotState:
		return wsRelay[WebsocketSpotState](ctx, e, envelope.Data)
	case wsChannelAllDEXsClearinghouseState:
		return wsRelay[WebsocketAllDEXsClearinghouseState](ctx, e, envelope.Data)
	case wsChannelAllDEXsAssetContexts:
		return wsRelay[WebsocketAllDEXsAssetContexts](ctx, e, envelope.Data)
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
func (e *Exchange) wsHandleSubscriptionResponse(conn websocket.Connection, raw []byte) error {
	var response *WebsocketSubscriptionResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return fmt.Errorf("%w: %w", errWebsocketSubscription, err)
	}
	if response == nil {
		return fmt.Errorf("%w: %w", errWebsocketSubscription, common.ErrNoResponse)
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
	_, err := e.completeWebsocketPending(conn, response.Method, &response.Subscription)
	return err
}

// completeWebsocketPending applies the server's completion of a pending operation to the subscription store; it reports
// whether an operation was pending, as a late or unknown completion is ignored
func (e *Exchange) completeWebsocketPending(conn websocket.Connection, method string, sub *WebsocketSubscription) (bool, error) {
	key := websocketPendingKey{connection: conn, subscription: *sub}
	e.websocketPendingMu.Lock()
	if method == wsMethodUnsubscribe {
		delete(e.websocketUnheldAggregatedFills, key)
	}
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
	abandoned := pending.abandoned
	if abandoned && key.subscription.Type == wsChannelUserFills && key.subscription.AggregateByTime {
		if e.websocketUnheldAggregatedFills == nil {
			e.websocketUnheldAggregatedFills = make(map[websocketPendingKey]struct{})
		}
		e.websocketUnheldAggregatedFills[key] = struct{}{}
	}
	e.websocketPendingMu.Unlock()
	if abandoned {
		// The feed is now active, but its subscription was rolled back when abandoned. Unsubscribing would stop a
		// configured feed until the next flush or reconnection, so the feed is left active and unheld: a later subscribe
		// of it takes it over through "Already subscribed". Until then it is not checked against exclusive variants, and
		// counts towards the subscription limit only through the room kept for a configured account feed not yet held
		return true, nil
	}

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

// failWebsocketPendingOperation fails and rolls back one pending operation; it reports whether the operation was pending.
// An abandoned subscribe is only removed, as it was failed and rolled back when abandoned
func (e *Exchange) failWebsocketPendingOperation(conn websocket.Connection, method string, sub *WebsocketSubscription, cause error) bool {
	key := websocketPendingKey{connection: conn, subscription: *sub}
	e.websocketPendingMu.Lock()
	pending := e.websocketPending[key]
	if pending == nil || pending.method != method {
		e.websocketPendingMu.Unlock()
		return false
	}
	delete(e.websocketPending, key)
	abandoned := pending.abandoned
	e.websocketPendingMu.Unlock()
	if !abandoned {
		pending.done <- common.AppendError(cause, e.rollbackWebsocketPending(pending))
	}
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

// failWebsocketPending fails and rolls back every operation outstanding on a connection. Abandoned subscribes, already
// rolled back, are dropped only when the connection is lost, as until then their feeds may be active
func (e *Exchange) failWebsocketPending(conn websocket.Connection, cause error) error {
	lost := errors.Is(cause, websocket.ErrNotConnected)
	e.websocketPendingMu.Lock()
	pending := make([]*websocketPendingOperation, 0, len(e.websocketPending))
	for key, operation := range e.websocketPending {
		if key.connection == conn {
			if operation.abandoned && !lost {
				continue
			}
			delete(e.websocketPending, key)
			if !operation.abandoned {
				pending = append(pending, operation)
			}
		}
	}
	if lost {
		for key := range e.websocketUnheldAggregatedFills {
			if key.connection == conn {
				delete(e.websocketUnheldAggregatedFills, key)
			}
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

// abandonWebsocketPending fails a sent operation the server has not settled in time; if the acknowledgement handler
// already took the operation, its outcome is returned instead. Hyperliquid may still acknowledge a subscribe and keep
// sending its feed, so an abandoned subscribe stays pending, holding the feed's slot and refusing other operations on
// it, until Hyperliquid answers it or the connection drops
func (e *Exchange) abandonWebsocketPending(key *websocketPendingKey, pending *websocketPendingOperation, cause error) error {
	if pending.method != wsMethodSubscribe {
		return e.abortWebsocketPending(key, pending, cause)
	}
	e.websocketPendingMu.Lock()
	if e.websocketPending[*key] != pending {
		e.websocketPendingMu.Unlock()
		return <-pending.done
	}
	pending.abandoned = true
	e.websocketPendingMu.Unlock()
	return common.AppendError(cause, e.rollbackWebsocketPending(pending))
}

// abandonedWebsocketSubscriptions returns the subscriptions of the abandoned subscribes still pending, each holding a
// slot its feed may take
func (e *Exchange) abandonedWebsocketSubscriptions() subscription.List {
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	var abandoned subscription.List
	for _, pending := range e.websocketPending {
		if pending.abandoned {
			abandoned = append(abandoned, pending.subscription)
		}
	}
	return abandoned
}

// wsHandleError settles the operation an error message answers, which the message identifies by quoting its
// subscription or request; any other error fails every operation pending on the connection
func (e *Exchange) wsHandleError(conn websocket.Connection, raw []byte) error {
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
			if settled, err := e.completeWebsocketPending(conn, wsMethodSubscribe, &sub); settled {
				return err
			}
		}
		return cause
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorAlreadyUnsubscribed); ok {
		// The feed is inactive, so the unsubscribe succeeded
		if sub, err := decodeWebsocketSubscription(quoted); err == nil {
			if settled, err := e.completeWebsocketPending(conn, wsMethodUnsubscribe, &sub); settled {
				return err
			}
		}
		return cause
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorInvalidSubscription); ok {
		if sub, err := decodeWebsocketSubscription(quoted); err == nil && e.failWebsocketPendingOperation(conn, wsMethodSubscribe, &sub, cause) {
			return cause
		}
		return e.failWebsocketPending(conn, cause)
	}
	if quoted, ok := strings.CutPrefix(message, wsErrorParseFailure); ok {
		var quotedRequest WebsocketQuotedRequest
		if err := json.Unmarshal([]byte(quoted), &quotedRequest); err == nil {
			switch quotedRequest.Method {
			case wsMethodPost:
				response, err := json.Marshal(&WebsocketPostResponse{ID: quotedRequest.ID, Response: WebsocketPostResponseBody{Type: wsPostTypeError, Payload: raw}})
				if err == nil && conn.IncomingWithData(quotedRequest.ID, response) {
					return nil
				}
				return cause
			case wsMethodSubscribe, wsMethodUnsubscribe:
				if e.failWebsocketPendingOperation(conn, quotedRequest.Method, &quotedRequest.Subscription, cause) {
					return cause
				}
			}
		}
	}
	return e.failWebsocketPending(conn, cause)
}

// decodeWebsocketSubscription decodes the canonical subscription an error message quotes
func decodeWebsocketSubscription(quoted string) (WebsocketSubscription, error) {
	var sub WebsocketSubscription
	if err := json.Unmarshal([]byte(quoted), &sub); err != nil {
		return WebsocketSubscription{}, err
	}
	return sub, nil
}

// wsHandlePostResponse delivers a post response to the request awaiting its ID
func (e *Exchange) wsHandlePostResponse(conn websocket.Connection, raw []byte) error {
	var response *WebsocketPostResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("decoding post response: %w", common.ErrNoResponse)
	}
	return conn.RequireMatchWithData(response.ID, raw)
}

func (e *Exchange) wsHandleCandle(ctx context.Context, raw []byte) error {
	var update CandleResponse
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
	var update WebsocketL2Book
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
	var updates []RecentTradeResponse
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
	if len(e.websocketTradeReplays) != 1 {
		return
	}
	for coin := range e.websocketTradeReplays {
		// The replay ends the coin's history, so a replay after a reconnection relays every trade it holds
		if _, ok := e.websocketWatermarks.trades[coin]; !ok {
			if e.websocketWatermarks.trades == nil {
				e.websocketWatermarks.trades = make(map[string]websocketPosition)
			}
			e.websocketWatermarks.trades[coin] = websocketPosition{}
		}
	}
	clear(e.websocketTradeReplays)
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
func (e *Exchange) unrelayedTrades(updates []RecentTradeResponse) []RecentTradeResponse {
	coin := updates[0].Coin
	replay := e.consumeTradeReplay(coin)
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	if e.websocketWatermarks.trades == nil {
		e.websocketWatermarks.trades = make(map[string]websocketPosition)
	}
	return unrelayedEntries(e.websocketWatermarks.trades, coin, updates, replay, func(update *RecentTradeResponse) websocketPosition {
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
func (e *Exchange) clearWebsocketWatermark(feed *WebsocketSubscription) {
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
	var updates []WebsocketOrder
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
func (e *Exchange) applyWebsocketOrderFill(source *FillResponse, mapping *pairMapping, a asset.Item, side order.Side, now time.Time) (order.Detail, bool) {
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
	var event WebsocketUserEvent
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
	var update WebsocketUserFills
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	feed := websocketFillsFeed{channel: wsChannelUserFills, user: strings.ToLower(update.User)}
	fills := unrelayedFills(e, feed, update.Fills, update.IsSnapshot, func(f *FillResponse) websocketPosition {
		return websocketPosition{time: f.Time.Time(), tradeID: f.TradeID}
	})
	return e.relayFills(ctx, fills, nil, !e.aggregatesUserFills(feed.user))
}

// aggregatesUserFills reports whether the userFills feed held for a user, or else one acknowledged after its subscribe
// was abandoned, combines each order's fills within a block; a combined fill carries the trade ID of one of them, so it
// cannot be added up with the fills userEvents sends
func (e *Exchange) aggregatesUserFills(user string) bool {
	for _, sub := range e.Websocket.GetSubscriptions() {
		if key, ok := sub.EnsureKeyed().(*websocketSubscriptionKey); ok && key.feed.Type == wsChannelUserFills && key.feed.User == user {
			return key.feed.AggregateByTime
		}
	}
	e.websocketPendingMu.Lock()
	defer e.websocketPendingMu.Unlock()
	for key := range e.websocketUnheldAggregatedFills {
		if key.subscription.User == user {
			return true
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
func (e *Exchange) relayFills(ctx context.Context, fills []FillResponse, sliceFills []TWAPSliceFillResponse, toOrders bool) error {
	toFillsFeed := e.IsFillsFeedEnabled()
	if !toOrders && !toFillsFeed {
		return nil
	}
	orders := make([]order.Detail, 0, len(fills)+len(sliceFills))
	data := make([]fill.Data, 0, len(fills)+len(sliceFills))
	now := time.Now()
	relay := func(source *FillResponse) error {
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
func (e *Exchange) convertFill(source *FillResponse, p currency.Pair, a asset.Item, side order.Side) fill.Data {
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
	update := new(WebsocketUserFundings)
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
	update := new(WebsocketUserNonFundingLedgerUpdates)
	if err := json.Unmarshal(raw, update); err != nil {
		return err
	}
	if len(update.NonFundingLedgerUpdates) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, update)
}

func (e *Exchange) wsHandlePerpetualTicker(ctx context.Context, raw []byte) error {
	var update WebsocketPerpetualAssetContext
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
	var update WebsocketSpotAssetContext
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
	var update WebsocketUserTWAPSliceFills
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	feed := websocketFillsFeed{channel: wsChannelUserTWAPSliceFills, user: strings.ToLower(update.User)}
	sliceFills := unrelayedFills(e, feed, update.TWAPSliceFills, update.IsSnapshot, func(f *TWAPSliceFillResponse) websocketPosition {
		return websocketPosition{time: f.Fill.Time.Time(), tradeID: f.Fill.TradeID}
	})
	return e.relayFills(ctx, nil, sliceFills, true)
}

// wsHandleUserTWAPHistory relays TWAP status changes, dropping a message without changes
func (e *Exchange) wsHandleUserTWAPHistory(ctx context.Context, raw []byte) error {
	update := new(WebsocketUserTWAPHistory)
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
	var updates []WebsocketOutcomeMetaUpdate
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
	var contexts map[string]WebsocketFastAssetContext
	if err := json.Unmarshal(decompressed, &contexts); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, contexts)
}

// WebsocketAPI posts Hyperliquid's info requests and signed actions over the websocket connection rather than REST; an
// action with a REST form takes that form's name and arguments
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the info requests and signed actions posted over the websocket connection
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// PostInfo sends an info request body, such as an InfoRequest, over the public websocket connection and decodes the
// info endpoint's response into result
func (w WebsocketAPI) PostInfo(ctx context.Context, payload, result any) error {
	if payload == nil || result == nil {
		return common.ErrNilPointer
	}
	response, err := w.e.sendWebsocketPost(ctx, wsPostTypeInfo, payload)
	if err != nil {
		return err
	}
	var info *WebsocketPostInfoResponse
	if err := json.Unmarshal(response, &info); err != nil {
		return fmt.Errorf("decoding info post payload: %w", err)
	}
	if info == nil {
		return fmt.Errorf("decoding info post payload: %w", common.ErrNoResponse)
	}
	if err := json.Unmarshal(info.Data, result); err != nil {
		return fmt.Errorf("decoding %s info response: %w", info.Type, err)
	}
	return nil
}

// PlaceOrders places a batch of orders as Exchange.PlaceOrders does, posting the order action over the websocket
func (w WebsocketAPI) PlaceOrders(ctx context.Context, arg *PlaceOrdersRequest) ([]OrderActionStatusResponse, error) {
	return w.e.placeOrders(ctx, arg, true)
}

// CancelOrders cancels a batch of orders by order ID as Exchange.CancelOrders does, posting the cancel action over the
// websocket
func (w WebsocketAPI) CancelOrders(ctx context.Context, arg *CancelOrdersRequest) ([]CancelActionStatusResponse, error) {
	return w.e.cancelOrders(ctx, arg, true)
}

// CancelOrdersByClientOrderID cancels a batch of orders by client order ID as Exchange.CancelOrdersByClientOrderID
// does, posting the cancel action over the websocket
func (w WebsocketAPI) CancelOrdersByClientOrderID(ctx context.Context, arg *CancelOrdersByClientOrderIDRequest) ([]CancelActionStatusResponse, error) {
	return w.e.cancelOrdersByClientOrderID(ctx, arg, true)
}

// ModifySingleOrder replaces one order as Exchange.ModifySingleOrder does, posting the modify action over the websocket
func (w WebsocketAPI) ModifySingleOrder(ctx context.Context, arg *ModifyOrderRequest) error {
	return w.e.modifySingleOrder(ctx, arg, true)
}

// ModifyOrders replaces a batch of orders as Exchange.ModifyOrders does, posting the batch modify action over the
// websocket
func (w WebsocketAPI) ModifyOrders(ctx context.Context, arg *ModifyOrdersRequest) ([]OrderActionStatusResponse, error) {
	return w.e.modifyOrders(ctx, arg, true)
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
	e.websocketConnectionsMu.Lock()
	connection := e.websocketPostConnection
	e.websocketConnectionsMu.Unlock()
	if connection == nil {
		return nil, websocket.ErrNotConnected
	}
	id := e.websocketPostID.Add(1)
	raw, err := connection.SendMessageReturnResponse(ctx, request.Unset, id, &WebsocketPostRequest{
		Method:  wsMethodPost,
		ID:      id,
		Request: WebsocketPostRequestBody{Type: requestType, Payload: payload},
	})
	if err != nil {
		return nil, fmt.Errorf("posting %s request: %w", requestType, err)
	}
	var response *WebsocketPostResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decoding post response: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("decoding post response: %w", common.ErrNoResponse)
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

// expandWebsocketSubscriptions expands one connection's share of the configured subscriptions, the account feeds or the
// rest, and keys each by the feed it selects. Each share is expanded and validated apart, so the account feeds' errors
// leave the public connection alone; the configuration replaces the held subscriptions, so it is validated on its own
// rather than beside the feeds a flush is about to remove
func (e *Exchange) expandWebsocketSubscriptions(account bool) (subscription.List, error) {
	configured := slices.DeleteFunc(slices.Clone(e.Features.Subscriptions), func(s *subscription.Subscription) bool {
		return s != nil && s.Authenticated != account
	})
	subs, err := configured.ExpandTemplates(websocketTemplateExpander{e})
	if validationErr := e.ValidateSubscriptions(subs); validationErr != nil {
		return nil, common.AppendError(err, validationErr)
	}
	keyWebsocketSubscriptions(subs)
	return subs, err
}

// generatePublicSubscriptions returns the configured market feeds, which the public connection subscribes, first noting
// the account feeds they leave room for, as the public connection subscribes before the account connection. The market
// feeds past that room are left out, so that a flush removes held ones to make way for account feeds the configuration
// adds
func (e *Exchange) generatePublicSubscriptions() (subscription.List, error) {
	reserved := e.reserveAccountFeeds(context.TODO())
	subs, err := e.expandWebsocketSubscriptions(false)
	if err != nil && !errors.Is(err, websocket.ErrSubscriptionPartial) {
		return subs, err
	}
	if room := max(maximumWebsocketSubscriptions-reserved, 0); len(subs) > room {
		var leftOut int
		var refused error
		subs, leftOut, refused = e.capMarketFeeds(subs, room)
		if leftOut != 0 {
			err = common.AppendError(err, fmt.Errorf("%w: account feeds need %d of Hyperliquid's %d slots, which leaves out %d of the market feeds",
				websocket.ErrSubscriptionPartial, reserved, maximumWebsocketSubscriptions, leftOut))
		}
		if refused != nil {
			err = common.AppendError(err, fmt.Errorf("%w: market feeds left out: %w", websocket.ErrSubscriptionPartial, refused))
		}
	}
	return subs, err
}

// capMarketFeeds returns the market feeds that fit in room slots, leaving out the feeds configured last, with how many it
// left out and why it refused the entries whose form the public connection would refuse. A feed configured twice takes
// one slot, and its later entries are kept or left out with its first. A refused entry is left out too, as its key can
// still match a held feed, which the flush would then keep in a slot the account feeds need
func (e *Exchange) capMarketFeeds(subs subscription.List, room int) (kept subscription.List, leftOut int, refused error) {
	kept = make(subscription.List, 0, min(len(subs), room))
	counted := make(map[websocketFeedIdentity]struct{}, room)
	left := make(map[websocketFeedIdentity]struct{})
	for _, sub := range subs {
		if _, err := e.checkWebsocketFeed(sub); err != nil {
			refused = common.AppendError(refused, fmt.Errorf("%s: %w", sub, err))
			continue
		}
		// A form checkWebsocketFeed accepts always has an identity
		id, _ := newWebsocketSubscriptionKey(sub).identity()
		if _, ok := counted[id]; ok {
			kept = append(kept, sub)
			continue
		}
		if len(counted) == room {
			left[id] = struct{}{}
			continue
		}
		counted[id] = struct{}{}
		kept = append(kept, sub)
	}
	return kept, len(left), refused
}

// reserveAccountFeeds notes each account feed the account connection subscribes when it connects, for market feeds to
// leave room for in Hyperliquid's subscription limit, and returns how many it noted: a feed configured twice is noted
// once, and one whose form the account connection would refuse is not noted. The market or DEX a feed names is checked
// only when it is subscribed, so that no request delays the public connection, and a feed naming one that does not
// exist keeps its slot
func (e *Exchange) reserveAccountFeeds(ctx context.Context) int {
	var reserved []*websocketSubscriptionKey
	if e.accountFeedsAvailable(ctx) == nil {
		configured, err := e.expandWebsocketSubscriptions(true)
		// Any generation error but a partial one fails the account connection, which then subscribes nothing
		if err != nil && !errors.Is(err, websocket.ErrSubscriptionPartial) {
			configured = nil
		}
		for _, sub := range configured {
			if _, err := e.checkWebsocketFeed(sub); err != nil {
				continue
			}
			feed := newWebsocketSubscriptionKey(sub)
			if slices.ContainsFunc(reserved, func(k *websocketSubscriptionKey) bool { return k.Match(feed) }) {
				continue
			}
			reserved = append(reserved, feed)
		}
	}
	e.websocketPendingMu.Lock()
	e.websocketAccountFeeds = reserved
	e.websocketPendingMu.Unlock()
	return len(reserved)
}

// awaitedAccountFeeds returns how many of the reserved account feeds are not held, which a market feed leaves room for;
// none while account feeds are unavailable. A held account feed that is not reserved keeps its slot even when the flush
// under way will remove it, as the account connection flushes after the public one and a market feed given that slot
// would starve a reserved feed whenever the removal does not happen; the market feed waits for the next flush instead
func (e *Exchange) awaitedAccountFeeds(ctx context.Context, held subscription.List) int {
	if e.accountFeedsAvailable(ctx) != nil {
		return 0
	}
	e.websocketPendingMu.Lock()
	reserved := e.websocketAccountFeeds
	e.websocketPendingMu.Unlock()
	var awaited int
	for _, feed := range reserved {
		if !slices.ContainsFunc(held, func(sub *subscription.Subscription) bool {
			return sub.Authenticated && feed.Match(newWebsocketSubscriptionKey(sub))
		}) {
			awaited++
		}
	}
	return awaited
}

// generateAccountSubscriptions returns the configured account feeds, which the account connection subscribes when
// account feeds are available
func (e *Exchange) generateAccountSubscriptions() (subscription.List, error) {
	if err := e.accountFeedsAvailable(context.TODO()); err != nil {
		// Authenticated websocket support is off by default, which leaves the account feeds out without a warning
		if !errors.Is(err, errWebsocketAccountFeedsDisabled) {
			log.Warnf(log.WebsocketMgr, "%s account websocket feeds disabled: %s", e.Name, err)
		}
		return nil, nil
	}
	return e.expandWebsocketSubscriptions(true)
}

// accountFeedsAvailable returns an error unless authenticated websocket support is enabled and an account address is
// configured, without which the account feeds cannot be subscribed
func (e *Exchange) accountFeedsAvailable(ctx context.Context) error {
	if !e.IsWebsocketAuthenticationSupported() {
		return errWebsocketAccountFeedsDisabled
	}
	_, err := e.getWatchAddress(ctx)
	return err
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

// SubscribeToWebsocketChannels subscribes to feeds, keying each by the feed it selects first, as the manager refuses a
// subscription matching a held one and would otherwise compare them without their parameters; account feeds go to the
// account connection and the rest to the public connection
func (e *Exchange) SubscribeToWebsocketChannels(channels subscription.List) error {
	ctx := context.TODO()
	if slices.Contains(channels, nil) {
		return common.ErrNilPointer
	}
	var errs error
	if unavailable := e.accountFeedsAvailable(ctx); unavailable != nil {
		channels = slices.DeleteFunc(slices.Clone(channels), func(s *subscription.Subscription) bool {
			if s.Authenticated {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", unavailable, s))
			}
			return s.Authenticated
		})
	}
	expanded, err := channels.ExpandTemplates(websocketTemplateExpander{e})
	errs = common.AppendError(errs, err)
	held := e.Websocket.GetSubscriptions()
	// Expansion passes subscriptions already expanded through without validating them, so all are validated here
	if err := e.ValidateSubscriptions(slices.Concat(expanded, held)); err != nil {
		return common.AppendError(errs, err)
	}
	if total := len(held) + len(expanded); total > maximumWebsocketSubscriptions {
		return common.AppendError(errs, fmt.Errorf("%w: %d held and %d requested", errWebsocketSubscriptionsLimit, len(held), len(expanded)))
	}
	keyWebsocketSubscriptions(expanded)
	return common.AppendError(errs, e.manageWebsocketChannels(ctx, expanded, e.Websocket.SubscribeToSetup))
}

// UnsubscribeToWebsocketChannels unsubscribes from feeds; a subscription is found by the feed it selects, so
// unsubscribing a variant that is not held leaves the held variants subscribed
func (e *Exchange) UnsubscribeToWebsocketChannels(channels subscription.List) error {
	if slices.Contains(channels, nil) {
		return common.ErrNilPointer
	}
	expanded, errs := channels.ExpandTemplates(websocketTemplateExpander{e})
	held := make(subscription.List, 0, len(expanded))
	for _, sub := range expanded {
		feed := newWebsocketSubscriptionKey(sub)
		if sub = e.Websocket.GetSubscription(feed); sub == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrNotFound, feed))
			continue
		}
		held = append(held, sub)
	}
	return common.AppendError(errs, e.manageWebsocketChannels(context.TODO(), held, e.Websocket.UnsubscribeFromSetup))
}

// manageWebsocketChannels passes account feeds to the account connection setup and the rest to the public one
func (e *Exchange) manageWebsocketChannels(ctx context.Context, subs subscription.List, manage func(context.Context, any, subscription.List) error) error {
	var errs error
	for _, role := range []websocketConnectionRole{websocketPublicConnection, websocketAccountConnection} {
		account := role == websocketAccountConnection
		if roleSubs := slices.DeleteFunc(slices.Clone(subs), func(s *subscription.Subscription) bool { return s.Authenticated != account }); len(roleSubs) != 0 {
			errs = common.AppendError(errs, manage(ctx, role, roleSubs))
		}
	}
	return errs
}

// newWebsocketSubscriptionKey keys a subscription by the feed it selects
func newWebsocketSubscriptionKey(sub *subscription.Subscription) *websocketSubscriptionKey {
	options, err := websocketFeedOptions(sub)
	return &websocketSubscriptionKey{Subscription: sub, options: options, valid: err == nil}
}

// ValidateSubscriptions refuses two variants of a feed whose messages do not say which variant sent them: two l2Book
// variants of one market would replace each other's book in the shared store, and two userFills variants would relay
// every fill twice, once combined with the order's other fills of its block. An aggregated userFills is refused beside
// userEvents for the same reason, as userEvents relays every fill too
func (e *Exchange) ValidateSubscriptions(subs subscription.List) error {
	var books []*websocketSubscriptionKey
	var userFills, userEvents *websocketSubscriptionKey
	for _, sub := range subs {
		variant := newWebsocketSubscriptionKey(sub)
		if !variant.valid || sub.Authenticated != websocketFeeds[variant.options.Type].user {
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
			if userEvents != nil && variant.options.AggregateByTime {
				return fmt.Errorf("%w: %s and %s", subscription.ErrExclusiveSubscription, userEvents, variant)
			}
			userFills = variant
		case wsSubscriptionUserEvents:
			if userFills != nil && userFills.options.AggregateByTime {
				return fmt.Errorf("%w: %s and %s", subscription.ErrExclusiveSubscription, userFills, variant)
			}
			userEvents = variant
		}
	}
	return nil
}

// websocketSubscribe subscribes to feeds on a connection, completing only once the server acknowledges each one
func (e *Exchange) websocketSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	// Feeds validated before others were subscribed reach here too, as when the manager reconnects the account
	// connection with its configured feeds before a runtime subscription's, so they are validated against those held,
	// and against those of abandoned subscribes, which may still be active
	if err := e.ValidateSubscriptions(slices.Concat(subs, e.Websocket.GetSubscriptions(), e.abandonedWebsocketSubscriptions())); err != nil {
		return err
	}
	var errs error
	for _, sub := range subs {
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, conn, wsMethodSubscribe, sub))
	}
	return errs
}

// websocketUnsubscribe unsubscribes from held feeds on a connection, completing only once the server acknowledges each
// one
func (e *Exchange) websocketUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	var errs error
	for _, sub := range subs {
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, conn, wsMethodUnsubscribe, sub))
	}
	return errs
}

// manageWebsocketSubscription sends a subscription operation and waits for its acknowledgement, rolling the local
// subscription state back if the server rejects it, the connection drops or it times out
// An unsubscribe repeats the feed its subscription subscribed, as Hyperliquid requires, rather than rebuilding it from
// metadata that no longer lists a delisted market or from a configured address that has since changed
func (e *Exchange) manageWebsocketSubscription(ctx context.Context, connection websocket.Connection, method string, sub *subscription.Subscription) error {
	if sub == nil || connection == nil {
		return common.ErrNilPointer
	}
	var payload WebsocketSubscription
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
	previousState := sub.State()
	if method == wsMethodSubscribe {
		feed := newWebsocketSubscriptionKey(sub)
		feed.feed = payload
		// The store files a subscription under its key, so re-keying a stored subscription would strand its entry
		if e.Websocket.GetSubscription(feed) != nil {
			return fmt.Errorf("%w: %s", subscription.ErrDuplicate, feed)
		}
		// Held subscriptions include those awaiting their acknowledgement, so both connections' count against the limit;
		// a market feed also leaves room for the account feeds not yet held, which the account connection subscribes
		// when it connects
		// An abandoned subscribe's feed may be active, so it counts as held, and as an account feed it takes the room kept
		// for that feed unless it was subscribed for an address no longer watched
		held := e.Websocket.GetSubscriptions()
		abandoned := e.abandonedWebsocketSubscriptions()
		var reserved int
		if !sub.Authenticated {
			watched, _ := e.getWatchAddress(ctx) // awaitedAccountFeeds keeps no room without a watched address
			reserved = e.awaitedAccountFeeds(ctx, slices.Concat(held, slices.DeleteFunc(slices.Clone(abandoned), func(s *subscription.Subscription) bool {
				key, ok := s.EnsureKeyed().(*websocketSubscriptionKey)
				return ok && key.feed.User != "" && key.feed.User != watched
			})))
		}
		if occupied := len(held) + len(abandoned); occupied+reserved >= maximumWebsocketSubscriptions {
			return fmt.Errorf("%w: %d held and %d kept for account feeds", errWebsocketSubscriptionsLimit, occupied, reserved)
		}
		sub.SetKey(feed)
		if err := e.Websocket.AddSubscriptions(connection, sub); err != nil {
			return err
		}
	} else if err := sub.SetState(subscription.UnsubscribingState); err != nil {
		return err
	}
	key := websocketPendingKey{connection: connection, subscription: payload}
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

	err := e.awaitWebsocketPending(ctx, connection, &key, pending, &WebsocketSubscriptionRequest{Method: method, Subscription: payload})
	if err == nil && method == wsMethodUnsubscribe {
		e.clearWebsocketWatermark(&payload)
	}
	return err
}

// awaitWebsocketPending sends a subscription request and waits for the server to settle its pending operation
func (e *Exchange) awaitWebsocketPending(ctx context.Context, connection websocket.Connection, key *websocketPendingKey, pending *websocketPendingOperation, req *WebsocketSubscriptionRequest) error {
	if err := connection.SendJSONMessage(ctx, request.Unset, req); err != nil {
		return e.abortWebsocketPending(key, pending, fmt.Errorf("sending %s request: %w", req.Method, err))
	}
	timer := time.NewTimer(e.WebsocketResponseMaxLimit)
	defer timer.Stop()
	select {
	case err := <-pending.done:
		return err
	case <-ctx.Done():
		return e.abandonWebsocketPending(key, pending, ctx.Err())
	case <-timer.C:
		return e.abandonWebsocketPending(key, pending, fmt.Errorf("%w: %s", websocket.ErrSignatureTimeout, pending.subscription))
	}
}

// websocketSubscriptionPayload builds the feed a subscription refers to, validating every coin, DEX and option first,
// because Hyperliquid closes the connection, dropping every feed on it, for a subscription it cannot validate
func (e *Exchange) websocketSubscriptionPayload(ctx context.Context, sub *subscription.Subscription) (WebsocketSubscription, error) {
	payload, err := e.checkWebsocketFeed(sub)
	if err != nil {
		return WebsocketSubscription{}, err
	}
	feed := websocketFeeds[payload.Type]
	if feed.user {
		if payload.User, err = e.getWatchAddress(ctx); err != nil {
			return WebsocketSubscription{}, err
		}
	}
	if feed.coin {
		mapping, err := e.getPairMapping(ctx, sub.Pairs[0], sub.Asset)
		if err != nil {
			return WebsocketSubscription{}, err
		}
		payload.Coin = mapping.coin
	}
	if payload.DEX != "" {
		if err := e.validateWebsocketDEX(ctx, payload.DEX); err != nil {
			return WebsocketSubscription{}, err
		}
	}
	return payload, nil
}

// checkWebsocketFeed returns the options of the feed a subscription refers to once the checks that need no request pass:
// its options, its authenticated flag against the feed's kind, and its pair and asset; the market's coin and the DEX's
// registration are left to websocketSubscriptionPayload
func (e *Exchange) checkWebsocketFeed(sub *subscription.Subscription) (WebsocketSubscription, error) {
	payload, err := websocketFeedOptions(sub)
	if err != nil {
		return WebsocketSubscription{}, err
	}
	feed := websocketFeeds[payload.Type]
	if sub.Authenticated != feed.user {
		return WebsocketSubscription{}, fmt.Errorf("%w: %s must set authenticated to %t", subscription.ErrNotSupported, payload.Type, feed.user)
	}
	switch {
	case feed.coin:
		if len(sub.Pairs) != 1 {
			return WebsocketSubscription{}, subscription.ErrNotSinglePair
		}
		if sub.Pairs[0].IsEmpty() {
			return WebsocketSubscription{}, currency.ErrCurrencyPairEmpty
		}
		if !e.SupportsAsset(sub.Asset) {
			return WebsocketSubscription{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, sub.Asset)
		}
		if payload.Type == wsChannelActiveAssetData && sub.Asset != asset.PerpetualContract {
			return WebsocketSubscription{}, fmt.Errorf("%w: %s serves perpetual contracts only", asset.ErrNotSupported, payload.Type)
		}
	case sub.Asset != asset.Empty:
		return WebsocketSubscription{}, fmt.Errorf("%w: %s does not take an asset", subscription.ErrNotSupported, payload.Type)
	}
	return payload, nil
}

// websocketFeedOptions returns the Hyperliquid subscription type a subscription refers to and the options its interval,
// levels and parameters select, as its payload sends them; it reads only what the type takes, as Hyperliquid would
// ignore the rest, and leaves the coin and user, which the market and account supply
func websocketFeedOptions(sub *subscription.Subscription) (WebsocketSubscription, error) {
	name, err := websocketChannelName(sub)
	if err != nil {
		return WebsocketSubscription{}, err
	}
	options := WebsocketSubscription{Type: name}
	if websocketFeeds[name].dex {
		if options.DEX, err = websocketDEXName(sub.Params); err != nil {
			return WebsocketSubscription{}, err
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
		return WebsocketSubscription{}, err
	}
	return options, nil
}

// websocketOrderbookOptions sets an l2Book's options: 5 levels select the fast book, and the nSigFigs and mantissa
// parameters aggregate it
func websocketOrderbookOptions(sub *subscription.Subscription, payload *WebsocketSubscription) error {
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
