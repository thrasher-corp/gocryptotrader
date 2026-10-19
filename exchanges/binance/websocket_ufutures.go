package binance

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// USDⓈ-M and options share the routed fstream hosts; each product gets its own connections so their handlers never
// see the other's payloads
const (
	fstreamPublicURL  = "wss://fstream.binance.com/public/stream"
	fstreamMarketURL  = "wss://fstream.binance.com/market/stream"
	fstreamPrivateURL = "wss://fstream.binance.com/private/stream"

	usdtmPublicFilter  = "usd-m-public"
	usdtmMarketFilter  = "usd-m-market"
	usdtmPrivateFilter = "usd-m-private"

	userDataStreamChannel = "userDataStream"

	derivativesSubscriptionBatchSize  = 50
	userDataKeepaliveInterval         = 30 * time.Minute
	derivativesOrderbookSnapshotLimit = 1000

	wsListSubscriptionsMethod = "LIST_SUBSCRIPTIONS"
	wsSetPropertyMethod       = "SET_PROPERTY"
)

var (
	errUnhandledStreamEvent    = errors.New("unhandled stream event")
	errUnknownSymbolType       = errors.New("unknown symbol type")
	errListenKeyEmpty          = errors.New("listen key empty")
	errOrderbookSyncNotSetUp   = errors.New("websocket order book synchronisation not set up")
	errStreamRequestRejected   = errors.New("stream request rejected")
	errUnexpectedStreamResult  = errors.New("unexpected stream request result")
	errUnsupportedStreamOption = errors.New("unsupported stream option")
	errListenKeyExpired        = errors.New("listen key expired")
	errUpdateIDOutOfRange      = errors.New("order book update ID out of range")
)

// futuresKlineIntervals are the candle intervals the USDⓈ-M and COIN-M kline streams accept
var futuresKlineIntervals = []kline.Interval{
	kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour,
	kline.FourHour, kline.SixHour, kline.EightHour, kline.TwelveHour, kline.OneDay, kline.ThreeDay, kline.OneWeek,
	kline.OneMonth,
}

// WsDerivativesConnect dials a USDⓈ-M, COIN-M or options stream connection at its configured URL. Gorilla answers the
// pings the servers send, every 3 minutes for USDⓈ-M and COIN-M and every 5 minutes for options
func (e *Exchange) WsDerivativesConnect(ctx context.Context, conn websocket.Connection) error {
	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	if err := conn.Dial(ctx, &dialer, http.Header{}, nil); err != nil {
		return fmt.Errorf("error dialling derivatives stream: %w", err)
	}
	conn.SetupPingHandler(wsConnectionMessageRate, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	return nil
}

// generateUFuturesPublicSubscriptions returns the USDⓈ-M /public stream subscriptions, its order books
func (e *Exchange) generateUFuturesPublicSubscriptions() (subscription.List, error) {
	return e.generateDerivativesSubscriptions(asset.USDTMarginedFutures, isOrderbookSubscription)
}

// generateUFuturesMarketSubscriptions returns the USDⓈ-M /market stream subscriptions, all but its order books
func (e *Exchange) generateUFuturesMarketSubscriptions() (subscription.List, error) {
	return e.generateDerivativesSubscriptions(asset.USDTMarginedFutures, func(s *subscription.Subscription) bool { return !isOrderbookSubscription(s) })
}

// isOrderbookSubscription reports whether a subscription streams an order book
func isOrderbookSubscription(s *subscription.Subscription) bool {
	return s.Channel == subscription.OrderbookChannel
}

// generateUFuturesUserDataSubscriptions returns the USDⓈ-M user data stream subscription when it can be used
func (e *Exchange) generateUFuturesUserDataSubscriptions() (subscription.List, error) {
	return e.generateUserDataSubscriptions(asset.USDTMarginedFutures)
}

// generateDerivativesSubscriptions expands an asset's subscriptions that belong on a connection into one subscription
// per pair: the subscription's pairs, or every enabled pair. A disabled asset generates nothing, so its connection is
// skipped rather than failing every other connection, and a subscription that cannot be expanded, or an order book
// subscription left out for conflicting with another, is reported as partial generation
func (e *Exchange) generateDerivativesSubscriptions(a asset.Item, belongs func(*subscription.Subscription) bool) (subscription.List, error) {
	if !e.GetAssetTypes(true).Contains(a) {
		return subscription.List{}, nil
	}
	enabled, err := e.GetEnabledPairs(a)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionPartial, a, err)
	}
	var subs subscription.List
	var errs error
	for _, t := range e.Features.Subscriptions {
		if t.Asset != a || !belongs(t) {
			continue
		}
		stream, err := e.derivativesStreamName(a, t)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		pairs := t.Pairs
		if len(pairs) == 0 {
			pairs = enabled
		}
		for _, p := range pairs {
			symbol, err := e.FormatSymbol(p, a)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			s := t.Clone()
			s.Pairs = currency.Pairs{p}
			// Stream names are lower case
			s.QualifiedChannel = strings.ToLower(symbol) + "@" + stream
			subs = append(subs, s)
		}
	}
	subs, conflicts := oneOrderbookSubscription(subs)
	if err := common.AppendError(errs, conflicts); err != nil {
		return subs, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionPartial, a, err)
	}
	return subs, nil
}

// derivativesStreamName returns the stream name after the symbol for a subscription template
func (e *Exchange) derivativesStreamName(a asset.Item, s *subscription.Subscription) (string, error) {
	switch s.Channel {
	case subscription.TickerChannel:
		if a == asset.Options {
			return "optionTicker", nil
		}
		return "ticker", nil
	case subscription.AllTradesChannel:
		if a == asset.Options {
			return "optionTrade", nil
		}
		return "aggTrade", nil
	case subscription.CandlesChannel:
		intervals := futuresKlineIntervals
		if a == asset.Options {
			intervals = optionsKlineIntervals
		}
		if !slices.Contains(intervals, s.Interval) {
			return "", fmt.Errorf("%w: %s candles %s", kline.ErrUnsupportedInterval, a, s.Interval)
		}
		return "kline_" + e.FormatExchangeKlineInterval(s.Interval), nil
	case subscription.OrderbookChannel:
		var levels string
		switch s.Levels {
		case 0:
		case 5, 10, 20:
			levels = strconv.Itoa(s.Levels)
		default:
			return "", fmt.Errorf("%w: %s order book levels %d", errUnsupportedStreamOption, a, s.Levels)
		}
		switch s.Interval {
		case kline.HundredMilliseconds, kline.FiveHundredMilliseconds:
			return "depth" + levels + "@" + s.Interval.Short(), nil
		case 0:
			// USDⓈ-M and COIN-M push every 250ms without a speed suffix; options require one
			if a != asset.Options {
				return "depth" + levels, nil
			}
		}
		return "", fmt.Errorf("%w: %s order book update speed %s", errUnsupportedStreamOption, a, s.Interval)
	default:
		return "", fmt.Errorf("%w: %s %q", subscription.ErrNotSupported, a, s.Channel)
	}
}

// SubscribeDerivatives subscribes to USDⓈ-M, COIN-M or options streams
func (e *Exchange) SubscribeDerivatives(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.manageDerivativesSubscriptions(ctx, conn, wsSubscribeMethod, l)
	}, derivativesSubscriptionBatchSize)
}

// UnsubscribeDerivatives unsubscribes from USDⓈ-M, COIN-M or options streams
func (e *Exchange) UnsubscribeDerivatives(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.manageDerivativesSubscriptions(ctx, conn, wsUnsubscribeMethod, l)
	}, derivativesSubscriptionBatchSize)
}

// manageDerivativesSubscriptions sends one SUBSCRIBE or UNSUBSCRIBE request and records the outcome in the store
func (e *Exchange) manageDerivativesSubscriptions(ctx context.Context, conn websocket.Connection, method string, subs subscription.List) error {
	if method == wsSubscribeMethod {
		if err := e.Websocket.AddSubscriptions(conn, subs...); err != nil {
			return err
		}
	} else if err := subs.SetStates(subscription.UnsubscribingState); err != nil {
		return err
	}
	streams := subs.QualifiedChannels()
	params := make([]any, len(streams))
	for i := range streams {
		params[i] = streams[i]
	}
	if err := e.sendDerivativesStreamCommand(ctx, conn, method, params); err != nil {
		err = fmt.Errorf("%w: %w; streams: %s", websocket.ErrSubscriptionFailure, err, strings.Join(streams, ", "))
		if method == wsSubscribeMethod {
			err = common.AppendError(err, e.Websocket.RemoveSubscriptions(conn, subs...))
		}
		return err
	}
	if method == wsSubscribeMethod {
		return subs.SetStates(subscription.SubscribedState)
	}
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}

// sendDerivativesStreamRequest sends a request on a derivatives stream connection and returns its result. The servers
// close the connection after rejecting a request
func (e *Exchange) sendDerivativesStreamRequest(ctx context.Context, conn websocket.Connection, method string, params []any) (json.RawMessage, error) {
	req := &DerivativesStreamRequest{Method: method, Params: params, ID: e.MessageSequence()}
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsConnectionMessageRate, req.ID, req)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesStreamResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%w: %s code %d: %s", errStreamRequestRejected, method, resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// sendDerivativesStreamCommand sends a request whose only successful result is null
func (e *Exchange) sendDerivativesStreamCommand(ctx context.Context, conn websocket.Connection, method string, params []any) error {
	result, err := e.sendDerivativesStreamRequest(ctx, conn, method, params)
	if err != nil {
		return err
	}
	if len(result) != 0 && string(result) != "null" {
		return fmt.Errorf("%w for %s: %s", errUnexpectedStreamResult, method, result)
	}
	return nil
}

// ListSubscriptions returns the streams a derivatives stream connection is subscribed to
func (e *Exchange) ListSubscriptions(ctx context.Context, conn websocket.Connection) ([]string, error) {
	result, err := e.sendDerivativesStreamRequest(ctx, conn, wsListSubscriptionsMethod, nil)
	if err != nil {
		return nil, err
	}
	var streams []string
	return streams, json.Unmarshal(result, &streams)
}

// SetProperty sets a property of a derivatives stream connection. Binance only supports "combined", which this
// package's handlers require to stay true
func (e *Exchange) SetProperty(ctx context.Context, conn websocket.Connection, property string, value any) error {
	return e.sendDerivativesStreamCommand(ctx, conn, wsSetPropertyMethod, []any{property, value})
}

// generateUserDataSubscriptions returns a user data stream subscription for an asset when authenticated websocket use
// is possible. The listen key is fetched when subscribing, so none is kept where subscriptions are listed
func (e *Exchange) generateUserDataSubscriptions(a asset.Item) (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() || !e.GetAssetTypes(true).Contains(a) || !e.AreCredentialsValid(context.Background()) {
		return subscription.List{}, nil
	}
	return subscription.List{{Channel: userDataStreamChannel, Asset: a, Authenticated: true}}, nil
}

// SubscribeUserData subscribes a private connection to the user data stream of each subscription's asset and keeps
// its listen key alive. Binance treats a listen key as a stream name on combined connections
func (e *Exchange) SubscribeUserData(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		if err := e.subscribeUserData(ctx, conn, s); err != nil {
			errs = common.AppendError(errs, err)
			if !errors.Is(err, asset.ErrNotSupported) {
				e.retryUserDataSubscription(ctx, conn, s.Asset, false, func(attemptCtx context.Context) (bool, error) {
					return e.resubscribeUserData(attemptCtx, ctx, conn, s)
				})
			}
		}
	}
	return errs
}

// resubscribeUserData subscribes a private connection to the user data stream a subscription failed to, while the
// stream is still wanted and no other subscription has been made to it, reporting whether it is done trying. The
// attempt is made under ctx, which ends with the retry, and the listen key is kept alive under keepAliveCtx, as when
// the connection first subscribed
func (e *Exchange) resubscribeUserData(ctx, keepAliveCtx context.Context, conn websocket.Connection, failed *subscription.Subscription) (done bool, err error) {
	if e.Websocket.GetSubscription(failed) != nil {
		return true, nil
	}
	if wanted, err := e.generateUserDataSubscriptions(failed.Asset); err != nil || len(wanted) == 0 {
		return true, err
	}
	s := failed.Clone()
	if err := e.subscribeUserDataKeptAlive(ctx, keepAliveCtx, conn, s); err != nil {
		return false, err
	}
	// Connecting records the subscriptions it makes against the connection, which tells a flush the connection is in use
	return true, conn.Subscriptions().Add(s)
}

func (e *Exchange) subscribeUserData(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	return e.subscribeUserDataKeptAlive(ctx, ctx, conn, s)
}

// subscribeUserDataKeptAlive subscribes under ctx and keeps the listen key alive under keepAliveCtx
func (e *Exchange) subscribeUserDataKeptAlive(ctx, keepAliveCtx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	if err := e.Websocket.AddSubscriptions(conn, s); err != nil {
		return err
	}
	listenKey, err := e.startUserDataStream(ctx, s.Asset)
	if err == nil {
		err = e.sendDerivativesStreamCommand(ctx, conn, wsSubscribeMethod, []any{listenKey})
	}
	if err != nil {
		return common.AppendError(fmt.Errorf("%w: %s user data stream: %w", websocket.ErrSubscriptionFailure, s.Asset, err), e.Websocket.RemoveSubscriptions(conn, s))
	}
	if err := s.SetState(subscription.SubscribedState); err != nil {
		return err
	}
	e.keepUserDataStreamAlive(keepAliveCtx, conn, s)
	return nil
}

// UnsubscribeUserData unsubscribes a private connection from its user data streams. The account's active listen key
// is fetched again because a renewal may have replaced the one first subscribed; it is not closed, since every
// client of the account shares it
func (e *Exchange) UnsubscribeUserData(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.UnsubscribingState); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		listenKey, err := e.startUserDataStream(ctx, s.Asset)
		if err == nil {
			err = e.sendDerivativesStreamCommand(ctx, conn, wsUnsubscribeMethod, []any{listenKey})
		}
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s user data stream: %w", websocket.ErrSubscriptionFailure, s.Asset, err))
			continue
		}
		errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
	}
	return errs
}

// startUserDataStream returns the account's active listen key for an asset, creating one when none is active
func (e *Exchange) startUserDataStream(ctx context.Context, a asset.Item) (string, error) {
	var listenKey string
	switch a {
	case asset.USDTMarginedFutures:
		resp, err := e.StartUFuturesUserDataStream(ctx)
		if err != nil {
			return "", err
		}
		listenKey = resp.ListenKey
	case asset.CoinMarginedFutures:
		resp, err := e.StartCFuturesUserDataStream(ctx)
		if err != nil {
			return "", err
		}
		listenKey = resp.ListenKey
	case asset.Options:
		resp, err := e.StartOptionsUserDataStream(ctx)
		if err != nil {
			return "", err
		}
		listenKey = resp.ListenKey
	default:
		return "", fmt.Errorf("%w: %s user data stream", asset.ErrNotSupported, a)
	}
	if listenKey == "" {
		return "", fmt.Errorf("%w: %s", errListenKeyEmpty, a)
	}
	return listenKey, nil
}

// keepaliveUserDataStream extends the validity of the account's listen key for an asset
func (e *Exchange) keepaliveUserDataStream(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.USDTMarginedFutures:
		_, err := e.KeepaliveUFuturesUserDataStream(ctx)
		return err
	case asset.CoinMarginedFutures:
		_, err := e.KeepaliveCFuturesUserDataStream(ctx)
		return err
	case asset.Options:
		return e.KeepaliveOptionsUserDataStream(ctx)
	default:
		return fmt.Errorf("%w: %s user data stream", asset.ErrNotSupported, a)
	}
}

// keepUserDataStreamAlive extends a user data stream's listen key every 30 minutes, as a key expires 60 minutes after
// its last keepalive, until the websocket shuts down or the stream is unsubscribed
func (e *Exchange) keepUserDataStreamAlive(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) {
	shutdown := e.Websocket.ShutdownC
	interval := cmp.Or(e.userDataRetry.keepalive, userDataKeepaliveInterval)
	e.Websocket.Wg.Go(func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-shutdown:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				if !e.keepaliveUserDataSubscription(ctx, conn, s) {
					return
				}
			}
		}
	})
}

// keepaliveUserDataSubscription extends the listen key of a subscribed user data stream and reports whether the
// stream is still subscribed. A failed keepalive means the key has expired, so the stream is renewed with a new one
func (e *Exchange) keepaliveUserDataSubscription(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) bool {
	if state := s.State(); state != subscription.SubscribedState && state != subscription.ResubscribingState {
		return false
	}
	if err := e.keepaliveUserDataStream(ctx, s.Asset); err != nil {
		e.renewUserDataStream(ctx, conn, s.Asset, err)
	}
	return true
}

// renewUserDataStream subscribes a connection to a new listen key after the old one expired; without it no further
// user data arrives
func (e *Exchange) renewUserDataStream(ctx context.Context, conn websocket.Connection, a asset.Item, reason error) {
	listenKey, err := e.startUserDataStream(ctx, a)
	if err == nil {
		err = e.sendDerivativesStreamCommand(ctx, conn, wsSubscribeMethod, []any{listenKey})
	}
	if err == nil {
		return
	}
	err = fmt.Errorf("%s user data stream renewal after %w failed: %w", a, reason, err)
	if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
		log.Errorf(log.WebsocketMgr, "%s %s: %s", e.Name, errSend, err)
	}
}

// A failed user data stream subscription is tried again after userDataRetryDelay, then after twice the last wait each
// time, up to userDataRetryMaxDelay
const (
	userDataRetryDelay    = 5 * time.Second
	userDataRetryMaxDelay = 5 * time.Minute
)

// userDataRetries tracks, by asset, the connection a user data stream subscription is being tried again on, so that an
// asset has one retry at a time and a retry on a replaced connection cannot end its successor's
type userDataRetries struct {
	mu      sync.Mutex
	running map[asset.Item]websocket.Connection
	// delay is the first wait before a retry, userDataRetryDelay when zero
	delay time.Duration
	// keepalive is the interval between the keepalives of a subscribed stream's listen key, userDataKeepaliveInterval
	// when zero
	keepalive time.Duration
}

// retryUserDataSubscription subscribes asset a's user data stream on conn in the background, at once or after the first
// retry delay, and again after a longer wait each time subscribe fails, until subscribe reports it is done, the
// connection goes down, or the websocket shuts down. Nothing else asks again while the connection stays up, which can
// be until Binance's daily disconnection, leaving the account's balances and orders to polls alone. It must be called
// while the websocket's wait group counts a running routine, such as a connection's reader
func (e *Exchange) retryUserDataSubscription(ctx context.Context, conn websocket.Connection, a asset.Item, immediately bool, subscribe func(context.Context) (done bool, err error)) {
	r := &e.userDataRetry
	r.mu.Lock()
	if r.running[a] == conn {
		r.mu.Unlock()
		return
	}
	if r.running == nil {
		r.running = make(map[asset.Item]websocket.Connection)
	}
	r.running[a] = conn
	delay := cmp.Or(r.delay, userDataRetryDelay)
	r.mu.Unlock()
	var wait time.Duration
	if !immediately {
		wait = delay
	}
	shutdown := e.Websocket.ShutdownC
	e.Websocket.Wg.Go(func() {
		defer func() {
			r.mu.Lock()
			if r.running[a] == conn {
				delete(r.running, a)
			}
			r.mu.Unlock()
		}()
		// An attempt under way when the websocket shuts down is cancelled rather than holding the shutdown up
		retryCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-shutdown:
				cancel()
			case <-retryCtx.Done():
			}
		}()
		timer := time.NewTimer(wait)
		defer timer.Stop()
		for {
			select {
			case <-retryCtx.Done():
				return
			case <-timer.C:
			}
			if !isConnected(conn) {
				return
			}
			done, err := subscribe(retryCtx)
			if err != nil {
				e.reportUserDataStreamError(retryCtx, a, err)
			}
			if done {
				return
			}
			wait = nextUserDataRetryWait(wait, delay)
			timer.Reset(wait)
		}
	})
}

// nextUserDataRetryWait returns the wait before the next attempt at a user data stream subscription after a failed one
// that followed a wait of wait: the first retry delay after an immediate attempt, and twice the last wait after that,
// up to userDataRetryMaxDelay
func nextUserDataRetryWait(wait, delay time.Duration) time.Duration {
	if wait == 0 {
		return delay
	}
	return min(wait*2, userDataRetryMaxDelay)
}

// fetchDerivativesOrderbook returns a USDⓈ-M, COIN-M or options REST order book as deep as the asset's snapshots, which
// keeps options books, shallow but costly when deep, at a weight of 1
func (e *Exchange) fetchDerivativesOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	return e.fetchDerivativesOrderbookDepth(ctx, p, a, orderbookSnapshotLimit(a))
}

// fetchDerivativesOrderbookDepth returns a USDⓈ-M, COIN-M or options REST order book of up to limit levels a side
func (e *Exchange) fetchDerivativesOrderbookDepth(ctx context.Context, p currency.Pair, a asset.Item, limit uint64) (*orderbook.Book, error) {
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	var (
		bids, asks      orderbook.LevelsArrayPriceAmount
		lastUpdateID    uint64
		transactionTime types.Time
	)
	switch a {
	case asset.USDTMarginedFutures:
		resp, err := e.UFuturesOrderbook(ctx, p, limit)
		if err != nil {
			return nil, err
		}
		bids, asks, lastUpdateID, transactionTime = resp.Bids, resp.Asks, resp.LastUpdateID, resp.TransactionTime
	case asset.CoinMarginedFutures:
		resp, err := e.GetFuturesOrderbook(ctx, p, limit)
		if err != nil {
			return nil, err
		}
		bids, asks, lastUpdateID, transactionTime = resp.Bids, resp.Asks, resp.LastUpdateID, resp.TransactionTime
	case asset.Options:
		resp, err := e.GetEOptionsOrderbook(ctx, p, limit)
		if err != nil {
			return nil, err
		}
		bids, asks, lastUpdateID, transactionTime = resp.Bids, resp.Asks, resp.LastUpdateID, resp.TransactionTime
	default:
		return nil, fmt.Errorf("%w: %s order book synchronisation", asset.ErrNotSupported, a)
	}
	if lastUpdateID > math.MaxInt64 {
		return nil, fmt.Errorf("%w: %d", errUpdateIDOutOfRange, lastUpdateID)
	}
	book.Bids = bids.Levels()
	book.Asks = asks.Levels()
	book.LastUpdateID = int64(lastUpdateID)
	book.LastUpdated = transactionTime.Time()
	return book, nil
}

// checkDerivativesPendingUpdate applies the USDⓈ-M, COIN-M and options local order book procedure to buffered diff
// events. Binance chains events by pu, the final update ID of the previous event, so firstUpdateID carries pu+1.
// Events ending before the snapshot are dropped; the first applied event must not start after it. The documented
// U <= lastUpdateId test becomes pu < lastUpdateId+1, as pu < U, which also accepts a snapshot taken between two
// events: such a snapshot equals the book at pu
func checkDerivativesPendingUpdate(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error) {
	if update.UpdateID < lastUpdateID {
		return true, nil
	}
	if firstUpdateID > lastUpdateID+1 {
		return false, fmt.Errorf("%w: snapshot update ID %d precedes previous final update ID %d of %s %s", orderbookmanager.ErrOrderbookSnapshotOutdated, lastUpdateID, firstUpdateID-1, update.Asset, update.Pair)
	}
	return false, nil
}

// processDerivativesDepthUpdate passes a USDⓈ-M, COIN-M or options diff depth event to the order book
// synchronisation. Binance chains these events by pu instead of U, so pu+1 stands in for the first update ID the
// synchronisation checks against the book's last one
func (e *Exchange) processDerivativesDepthUpdate(ctx context.Context, previousFinalUpdateID int64, update *orderbook.Update) error {
	return e.processOrderbookUpdate(ctx, previousFinalUpdateID+1, update)
}

// isPartialDepthStream reports whether a depth stream pushes top-of-book snapshots, <symbol>@depth<levels>, rather
// than diffs, <symbol>@depth
func isPartialDepthStream(stream string) bool {
	_, name, _ := strings.Cut(stream, "@")
	levels, ok := strings.CutPrefix(name, "depth")
	return ok && levels != "" && levels[0] >= '0' && levels[0] <= '9'
}

// matchDerivativesPair resolves a stream symbol to an available pair and reports whether it is enabled. Pairs store
// the symbol without its first delimiter: perpetual USDⓈ-M symbols have none, while delivery contracts, COIN-M
// symbols (BTCUSD_PERP is BTCUSD/PERP) and options symbols split at it
func (e *Exchange) matchDerivativesPair(symbol string, a asset.Item) (pair currency.Pair, enabled bool, err error) {
	pair, err = e.MatchSymbolWithAvailablePairs(symbol, a, true)
	if err != nil {
		return currency.EMPTYPAIR, false, err
	}
	enabled, err = e.CurrencyPairs.IsPairEnabled(pair, a)
	return pair, enabled, err
}

// futuresSymbolTypeAsset returns the product of a payload from its symbol type, or the connection's asset when the
// payload has none. The all-market streams and both hosts carry merged USDⓈ-M and COIN-M content
func futuresSymbolTypeAsset(symbolType uint64, connAsset asset.Item) (asset.Item, error) {
	switch symbolType {
	case 0:
		return connAsset, nil
	case 1:
		return asset.USDTMarginedFutures, nil
	case 2:
		return asset.CoinMarginedFutures, nil
	default:
		return asset.Empty, fmt.Errorf("%w: %d", errUnknownSymbolType, symbolType)
	}
}

// ownProductEntries keeps the entries of a merged all-market array that belong to the connection's product. Both
// hosts deliver the same merged content, so each connection keeps its own product to avoid processing entries twice
func ownProductEntries[T any](entries []T, symbolType func(*T) uint64, connAsset asset.Item) []T {
	kept := entries[:0]
	for i := range entries {
		if a, err := futuresSymbolTypeAsset(symbolType(&entries[i]), connAsset); err == nil && a == connAsset {
			kept = append(kept, entries[i])
		}
	}
	return kept
}

// decodeStreamFrame splits a combined stream frame into its stream name, payload and event type. An array payload
// takes the event type of its first element, so an empty array has none
func decodeStreamFrame(respRaw []byte) (stream string, data []byte, event string, err error) {
	stream, err = jsonparser.GetString(respRaw, "stream")
	if err != nil {
		return "", nil, "", fmt.Errorf("%w: frame without a stream: %w", errUnhandledStreamEvent, err)
	}
	data, dataType, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return "", nil, "", fmt.Errorf("%w: %q frame without data: %w", errUnhandledStreamEvent, stream, err)
	}
	eventData := data
	if dataType == jsonparser.Array {
		// jsonparser reports an unreadable first element as missing, so emptiness is checked first to tell an empty
		// array from a malformed one
		if len(bytes.TrimSpace(data[1:len(data)-1])) == 0 {
			return stream, data, "", nil
		}
		if eventData, _, _, err = jsonparser.Get(data, "[0]"); err != nil {
			return "", nil, "", fmt.Errorf("%w: %q frame with an invalid array: %w", errUnhandledStreamEvent, stream, err)
		}
	}
	event, err = jsonparser.GetString(eventData, "e")
	if err != nil {
		return "", nil, "", fmt.Errorf("%w: %q frame without an event type: %w", errUnhandledStreamEvent, stream, err)
	}
	return stream, data, event, nil
}

// wsHandleUFuturesData routes USDⓈ-M /public and /market stream frames
func (e *Exchange) wsHandleUFuturesData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.handleFuturesStream(ctx, conn, respRaw, asset.USDTMarginedFutures)
}

// handleFuturesStream routes a USDⓈ-M or COIN-M market data frame. Request replies carry an id; stream frames are
// routed on their event type
func (e *Exchange) handleFuturesStream(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		return conn.RequireMatchWithData(id, respRaw)
	}
	stream, data, event, err := decodeStreamFrame(respRaw)
	if err != nil || event == "" {
		return err
	}
	if stream == "tradingSession" {
		return sendDecoded[FuturesTradingSession](ctx, e, data)
	}
	switch event {
	case "bookTicker":
		return e.processFuturesBookTicker(ctx, stream, data, a)
	case "depthUpdate":
		return e.processFuturesDepth(ctx, stream, data, a)
	case "aggTrade":
		return e.processFuturesAggregateTrade(data, a)
	case "forceOrder":
		return e.processFuturesLiquidation(ctx, stream, data, a)
	case "24hrMiniTicker":
		return e.processFuturesMiniTickers(ctx, data, a)
	case "24hrTicker":
		return e.processFuturesTickers(ctx, data, a)
	case "compositeIndex":
		return sendDecoded[FuturesCompositeIndex](ctx, e, data)
	case "continuous_kline":
		return sendDecoded[FuturesContinuousKline](ctx, e, data)
	case "contractInfo":
		return e.processFuturesContractInfo(ctx, data, a)
	case "kline":
		return e.processFuturesKline(ctx, data, a)
	case "markPriceUpdate":
		return e.processFuturesMarkPrices(ctx, data, a)
	case "assetIndexUpdate":
		return e.processFuturesAssetIndices(ctx, data)
	case "IndexUpdate", "indexPriceUpdate": // IndexUpdate is live; the docs still name indexPriceUpdate
		return sendDecoded[CFuturesIndexPrice](ctx, e, data)
	case "indexPrice_kline", "markPrice_kline":
		return sendDecoded[CFuturesPriceKline](ctx, e, data)
	default:
		return fmt.Errorf("%w %q on stream %q", errUnhandledStreamEvent, event, stream)
	}
}

// sendDecoded relays a payload decoded into its own type; it suits streams GCT has no common type for
func sendDecoded[T any](ctx context.Context, e *Exchange, data []byte) error {
	var resp *T
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// sendDecodedArray relays an array payload decoded into a slice of its element type
func sendDecodedArray[T any](ctx context.Context, e *Exchange, data []byte) error {
	var resp []T
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesBookTicker relays best bid and ask updates. They never touch the order book, which the depth streams
// own. !bookTicker carries both products, so each connection relays its own
func (e *Exchange) processFuturesBookTicker(ctx context.Context, stream string, data []byte, a asset.Item) error {
	var resp *FuturesBookTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if stream == "!bookTicker" {
		if ta, err := futuresSymbolTypeAsset(resp.SymbolType, a); err != nil || ta != a {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesDepth applies a depth event: diffs through the REST synchronised order book, partial depth as a
// snapshot of the top levels. RPI depth includes RPI orders, which the order book excludes, so it is relayed as is
func (e *Exchange) processFuturesDepth(ctx context.Context, stream string, data []byte, a asset.Item) error {
	var resp *FuturesDepthUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if strings.Contains(stream, "@rpiDepth") {
		return e.Websocket.DataHandler.Send(ctx, resp)
	}
	da, err := futuresSymbolTypeAsset(resp.SymbolType, a)
	if err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, da)
	if err != nil || !enabled {
		return err
	}
	if isPartialDepthStream(stream) {
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Bids:              resp.Bids.Levels(),
			Asks:              resp.Asks.Levels(),
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             da,
			LastUpdated:       resp.TransactionTime.Time(),
			LastPushed:        resp.EventTime.Time(),
			LastUpdateID:      resp.FinalUpdateID,
			ValidateOrderbook: e.ValidateOrderbook,
		})
	}
	return e.processDerivativesDepthUpdate(ctx, resp.PreviousFinalUpdateID, &orderbook.Update{
		UpdateID:   resp.FinalUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		LastPushed: resp.EventTime.Time(),
		Asset:      da,
		Bids:       resp.Bids.Levels(),
		Asks:       resp.Asks.Levels(),
		Pair:       pair,
		AllowEmpty: true,
	})
}

// processFuturesAggregateTrade relays an aggregate trade of an enabled pair
func (e *Exchange) processFuturesAggregateTrade(data []byte, a asset.Item) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp FuturesAggregateTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	ta, err := futuresSymbolTypeAsset(resp.SymbolType, a)
	if err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, ta)
	if err != nil || !enabled {
		return err
	}
	side := order.Buy
	if resp.IsBuyerMaker {
		side = order.Sell
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		TID:          strconv.FormatUint(resp.AggregateTradeID, 10),
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    ta,
		Side:         side,
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(), // COIN-M counts contracts, as its candle volumes and last trade sizes do
		Timestamp:    resp.TradeTime.Time(),
	})
}

// processFuturesLiquidation relays a market liquidation. These are other traders' orders, so they are not order
// details for the order manager. !forceOrder@arr carries both products, so each connection relays its own
func (e *Exchange) processFuturesLiquidation(ctx context.Context, stream string, data []byte, a asset.Item) error {
	var resp *FuturesLiquidationOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if strings.HasPrefix(stream, "!") {
		if la, err := futuresSymbolTypeAsset(resp.Order.SymbolType, a); err != nil || la != a {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesContractInfo relays a contract update of the connection's product; both hosts carry both products
func (e *Exchange) processFuturesContractInfo(ctx context.Context, data []byte, a asset.Item) error {
	var resp *FuturesContractInfo
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if ca, err := futuresSymbolTypeAsset(resp.SymbolType, a); err != nil || ca != a {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesMarkPrices relays mark price updates; arrays are filtered to the connection's product
func (e *Exchange) processFuturesMarkPrices(ctx context.Context, data []byte, a asset.Item) error {
	if data[0] != '[' {
		return sendDecoded[FuturesMarkPrice](ctx, e, data)
	}
	var resp []FuturesMarkPrice
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	resp = ownProductEntries(resp, func(m *FuturesMarkPrice) uint64 { return m.SymbolType }, a)
	if len(resp) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesAssetIndices relays asset index updates, an object for <assetSymbol>@assetIndex and an array for
// !assetIndex@arr
func (e *Exchange) processFuturesAssetIndices(ctx context.Context, data []byte) error {
	if data[0] == '[' {
		return sendDecodedArray[FuturesAssetIndex](ctx, e, data)
	}
	return sendDecoded[FuturesAssetIndex](ctx, e, data)
}

// processFuturesTickers processes 24 hour tickers of enabled pairs into the ticker store and relays them. The array
// form is the merged all-market stream, filtered to the connection's product
func (e *Exchange) processFuturesTickers(ctx context.Context, data []byte, a asset.Item) error {
	if data[0] != '[' {
		var resp FuturesTicker
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		ta, err := futuresSymbolTypeAsset(resp.SymbolType, a)
		if err != nil {
			return err
		}
		pair, enabled, err := e.matchDerivativesPair(resp.Symbol, ta)
		if err != nil || !enabled {
			return err
		}
		p := e.futuresTickerPrice(&resp, pair, ta)
		return e.processAndSendTicker(ctx, &p)
	}
	var resp []FuturesTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	resp = ownProductEntries(resp, func(t *FuturesTicker) uint64 { return t.SymbolType }, a)
	prices := make([]ticker.Price, 0, len(resp))
	for i := range resp {
		if pair, enabled, err := e.matchDerivativesPair(resp[i].Symbol, a); err == nil && enabled {
			prices = append(prices, e.futuresTickerPrice(&resp[i], pair, a))
		}
	}
	return e.processAndSendTickers(ctx, prices)
}

// futuresTickerPrice converts a 24 hour ticker. COIN-M counts v in contracts and sends the base asset volume as q, so
// it has no quote volume
func (e *Exchange) futuresTickerPrice(t *FuturesTicker, pair currency.Pair, a asset.Item) ticker.Price {
	p := ticker.Price{
		Last:                       t.LastPrice.Float64(),
		LastSize:                   t.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: t.WeightedAveragePrice.Float64(),
		High:                       t.HighPrice.Float64(),
		Low:                        t.LowPrice.Float64(),
		Open:                       t.OpenPrice.Float64(),
		PercentChange24Hour:        t.PriceChangePercent.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  a,
		LastUpdated:                t.EventTime.Time(),
	}
	if a == asset.CoinMarginedFutures {
		p.BaseVolume = t.TotalTradedQuoteAssetVolume.Float64()
		return p
	}
	p.BaseVolume = t.TotalTradedBaseAssetVolume.Float64()
	p.QuoteVolume = t.TotalTradedQuoteAssetVolume.Float64()
	return p
}

// processFuturesMiniTickers processes mini tickers of enabled pairs into the ticker store and relays them. The array
// form is the merged all-market stream, filtered to the connection's product
func (e *Exchange) processFuturesMiniTickers(ctx context.Context, data []byte, a asset.Item) error {
	if data[0] != '[' {
		var resp FuturesMiniTicker
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}
		ta, err := futuresSymbolTypeAsset(resp.SymbolType, a)
		if err != nil {
			return err
		}
		pair, enabled, err := e.matchDerivativesPair(resp.Symbol, ta)
		if err != nil || !enabled {
			return err
		}
		p := e.futuresMiniTickerPrice(&resp, pair, ta)
		return e.processAndSendTicker(ctx, &p)
	}
	var resp []FuturesMiniTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	resp = ownProductEntries(resp, func(t *FuturesMiniTicker) uint64 { return t.SymbolType }, a)
	prices := make([]ticker.Price, 0, len(resp))
	for i := range resp {
		if pair, enabled, err := e.matchDerivativesPair(resp[i].Symbol, a); err == nil && enabled {
			prices = append(prices, e.futuresMiniTickerPrice(&resp[i], pair, a))
		}
	}
	return e.processAndSendTickers(ctx, prices)
}

// futuresMiniTickerPrice converts a mini ticker, whose close is the last price. COIN-M counts v in contracts and sends
// the base asset volume as q, so it has no quote volume
func (e *Exchange) futuresMiniTickerPrice(t *FuturesMiniTicker, pair currency.Pair, a asset.Item) ticker.Price {
	p := ticker.Price{
		Last:         t.ClosePrice.Float64(),
		High:         t.HighPrice.Float64(),
		Low:          t.LowPrice.Float64(),
		Open:         t.OpenPrice.Float64(),
		Pair:         pair,
		ExchangeName: e.Name,
		AssetType:    a,
		LastUpdated:  t.EventTime.Time(),
	}
	if a == asset.CoinMarginedFutures {
		p.BaseVolume = t.TotalTradedQuoteAssetVolume.Float64()
		return p
	}
	p.BaseVolume = t.TotalTradedBaseAssetVolume.Float64()
	p.QuoteVolume = t.TotalTradedQuoteAssetVolume.Float64()
	return p
}

// processFuturesKline relays a candle of an enabled pair
func (e *Exchange) processFuturesKline(ctx context.Context, data []byte, a asset.Item) error {
	var resp FuturesKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, a)
	if err != nil || !enabled {
		return err
	}
	interval, err := formatToInterval(resp.Kline.Interval)
	if err != nil {
		return err
	}
	var validationIssues string
	if !resp.Kline.IsClosed {
		validationIssues = kline.PartialCandle
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:             resp.Kline.StartTime.Time(),
			Open:             resp.Kline.OpenPrice.Float64(),
			High:             resp.Kline.HighPrice.Float64(),
			Low:              resp.Kline.LowPrice.Float64(),
			Close:            resp.Kline.ClosePrice.Float64(),
			Volume:           resp.Kline.Volume.Float64(),
			ValidationIssues: validationIssues,
		}},
	})
}

// wsHandleUFuturesUserData routes USDⓈ-M user data stream frames
func (e *Exchange) wsHandleUFuturesUserData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.handleFuturesUserData(ctx, conn, respRaw, asset.USDTMarginedFutures)
}

// handleFuturesUserData routes a USDⓈ-M or COIN-M user data frame. Its stream name is the listen key, so frames are
// routed on their event type and errors never quote the frame
func (e *Exchange) handleFuturesUserData(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		return conn.RequireMatchWithData(id, respRaw)
	}
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w: %s user data frame without data: %w", errUnhandledStreamEvent, a, err)
	}
	event, err := jsonparser.GetString(data, "e")
	if err != nil {
		return fmt.Errorf("%w: %s user data frame without an event type: %w", errUnhandledStreamEvent, a, err)
	}
	switch event {
	case "MARGIN_CALL":
		return sendDecoded[FuturesMarginCall](ctx, e, data)
	case "ACCOUNT_UPDATE":
		return e.processFuturesAccountUpdate(ctx, data, a)
	case "ORDER_TRADE_UPDATE":
		return e.processFuturesOrderTradeUpdate(ctx, data, a)
	case "ACCOUNT_CONFIG_UPDATE":
		return sendDecoded[FuturesAccountConfigUpdate](ctx, e, data)
	case "TRADE_LITE":
		return sendDecoded[FuturesTradeLite](ctx, e, data)
	case "CONDITIONAL_ORDER_TRIGGER_REJECT":
		return sendDecoded[FuturesConditionalOrderTriggerReject](ctx, e, data)
	case "STRATEGY_UPDATE":
		return sendDecoded[FuturesStrategyUpdate](ctx, e, data)
	case "GRID_UPDATE":
		return sendDecoded[FuturesGridUpdate](ctx, e, data)
	case "ALGO_UPDATE":
		return e.processFuturesAlgoUpdate(ctx, data, a)
	case "listenKeyExpired":
		return e.processListenKeyExpired(ctx, conn, data, a)
	default:
		return fmt.Errorf("%w %q on the %s user data stream", errUnhandledStreamEvent, event, a)
	}
}

// processListenKeyExpired relays the expiry and renews the stream in the background, as the reply to the new
// subscription arrives through this same handler
func (e *Exchange) processListenKeyExpired(ctx context.Context, conn websocket.Connection, data []byte, a asset.Item) error {
	var resp *ListenKeyExpired
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	e.Websocket.Wg.Go(func() {
		e.renewUserDataStream(ctx, conn, a, errListenKeyExpired)
	})
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// userDataPair returns the pair of a user data symbol. A COIN-M or options symbol listed since the available pairs were
// last updated gets the pair FetchTradablePairs stores for it; a USDⓈ-M perpetual symbol cannot be split into its
// assets without exchange information, so an unknown USDⓈ-M symbol has none
func (e *Exchange) userDataPair(symbol string, a asset.Item) (currency.Pair, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, a, true)
	if !errors.Is(err, currency.ErrPairNotFound) {
		return pair, err
	}
	var delimiter string
	switch a {
	case asset.USDTMarginedFutures:
		// A delivery contract's symbol splits at its delivery date; a perpetual's base and quote need exchange information
		if !strings.Contains(symbol, currency.UnderscoreDelimiter) {
			return currency.EMPTYPAIR, err
		}
		delimiter = currency.UnderscoreDelimiter
	case asset.CoinMarginedFutures:
		delimiter = currency.UnderscoreDelimiter
	case asset.Options:
		delimiter = currency.DashDelimiter
	default:
		return currency.EMPTYPAIR, err
	}
	pair, splitErr := currency.NewPairDelimiter(symbol, delimiter)
	if splitErr != nil {
		return currency.EMPTYPAIR, fmt.Errorf("%w: %w", err, splitErr)
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return pair.Format(format), nil
}

// processFuturesAccountUpdate stores changed balances and relays them with the changed positions. The event carries
// wallet balances only, so the margin held from the last REST snapshot is kept and the free balance follows the
// wallet until a REST refresh reports the margin the change took or released. A position whose symbol has no pair is
// left out, so it cannot hold back the others
func (e *Exchange) processFuturesAccountUpdate(ctx context.Context, data []byte, a asset.Item) error {
	var resp FuturesAccountUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp.UpdateData.Balances) > 0 {
		codes := make([]currency.Code, len(resp.UpdateData.Balances))
		for i := range resp.UpdateData.Balances {
			codes[i] = resp.UpdateData.Balances[i].Asset
		}
		subAccount := accounts.NewSubAccount(a, "")
		if err := e.storeBalanceEvent(ctx, a, resp.TransactionTime.Time(), codes, func() error {
			for _, b := range resp.UpdateData.Balances {
				balance, err := e.Accounts.UpdateBalance(ctx, "", a, b.Asset, func(balance *accounts.Balance) {
					// Unrealised profit can lift the free balance above the wallet balance, which the hold cannot show, so
					// an unchanged wallet balance keeps the free balance REST reported
					if total := b.WalletBalance.Float64(); total != balance.Total {
						balance.Total = total
						balance.Free = max(total-balance.Hold, 0)
					}
				})
				if err != nil {
					return err
				}
				subAccount.Balances.Set(b.Asset, balance)
			}
			return nil
		}); err != nil {
			return err
		}
		if err := e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount}); err != nil {
			return err
		}
	}
	positions := make([]futures.Position, 0, len(resp.UpdateData.Positions))
	for i := range resp.UpdateData.Positions {
		p := &resp.UpdateData.Positions[i]
		pair, err := e.userDataPair(p.Symbol, a)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s %s account update: skipping a position: %v", e.Name, a, err)
			continue
		}
		position := futuresPosition(e.Name, a, pair, p.PositionSide, p.PositionAmount.Decimal(), resp.TransactionTime.Time())
		position.OpeningPrice = p.EntryPrice.Decimal()
		position.PositionMargin = p.IsolatedWallet.Decimal()
		position.RealisedPNL = p.AccumulatedRealised.Decimal()
		position.UnrealisedPNL = p.UnrealisedPNL.Decimal()
		positions = append(positions, position)
	}
	if len(positions) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

// balanceRefreshInterval is the least time between the REST balance refreshes of an asset that its user data stream
// balance events start, so a burst of events costs one refresh at the end of each interval
const balanceRefreshInterval = time.Second

// balanceRefreshAttempts bounds the refreshes an asset makes for its latest balance event while REST lags the user data
// stream, reporting a currency as last updated before that event or with another wallet balance
const balanceRefreshAttempts = 5

// balanceRefresher runs the REST balance refreshes the USDⓈ-M, COIN-M and options user data stream balance events start,
// one at a time for each asset. The events carry no available balance, and a fill changes the margin held, which only
// a REST snapshot reports
type balanceRefresher struct {
	mu      sync.Mutex
	running map[asset.Item]bool
	// pending marks an asset whose balances changed again while its refresh ran, so it is refreshed once more
	pending map[asset.Item]bool
	// eventTimes holds, by asset and currency, the transaction time of the latest stream event to report the
	// currency's balance, which a REST snapshot must not predate
	eventTimes map[asset.Item]map[*currency.Item]time.Time
	// snapshotTimes holds, by asset, the account update time of the latest REST snapshot stored for the configured
	// account where Binance gives one for the whole account, which a stream event must not predate
	snapshotTimes map[asset.Item]time.Time
	// interval is the time between refreshes of an asset, balanceRefreshInterval when zero
	interval time.Duration
	// storeMu keeps a balance event from being stored while a REST snapshot is accepted, by a refresh or a poll, so a
	// snapshot is judged against every event stored before it and no event is stored between its checks and its store
	storeMu sync.Mutex
}

// withBalanceStore runs f holding the balance store lock, which events hold to store their balances and refreshes and
// polls hold to accept a REST snapshot; REST requests are made outside it
func (e *Exchange) withBalanceStore(f func() error) error {
	e.balanceRefresh.storeMu.Lock()
	defer e.balanceRefresh.storeMu.Unlock()
	return f()
}

// storeBalanceEvent records a user data stream balance event and stores its balances with store, under the balance
// store lock, then refreshes the available balances the event does not carry. The refresh starts once the balances are
// stored, so its request follows the stores and its snapshot is not taken as older than them
func (e *Exchange) storeBalanceEvent(ctx context.Context, a asset.Item, transactionTime time.Time, codes []currency.Code, store func() error) error {
	if err := e.withBalanceStore(func() error {
		e.recordBalanceEvent(a, transactionTime, codes...)
		return store()
	}); err != nil {
		return err
	}
	e.refreshBalances(ctx, a)
	return nil
}

// recordBalanceEvent records when a user data stream balance event changed an asset's balances, which a REST snapshot
// of them must not predate
func (e *Exchange) recordBalanceEvent(a asset.Item, transactionTime time.Time, codes ...currency.Code) {
	r := &e.balanceRefresh
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.eventTimes == nil {
		r.eventTimes = make(map[asset.Item]map[*currency.Item]time.Time)
	}
	if r.eventTimes[a] == nil {
		r.eventTimes[a] = make(map[*currency.Item]time.Time)
	}
	for _, c := range codes {
		if transactionTime.After(r.eventTimes[a][c.Item]) {
			r.eventTimes[a][c.Item] = transactionTime
		}
	}
}

// recordBalanceSnapshot records the account update time of a REST snapshot of an asset's balances stored for the
// configured account
func (e *Exchange) recordBalanceSnapshot(a asset.Item, asOf time.Time) {
	r := &e.balanceRefresh
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.snapshotTimes == nil {
		r.snapshotTimes = make(map[asset.Item]time.Time)
	}
	if asOf.After(r.snapshotTimes[a]) {
		r.snapshotTimes[a] = asOf
	}
}

// balanceSnapshotTime returns the account update time of the latest REST snapshot of an asset's balances stored for the
// configured account
func (e *Exchange) balanceSnapshotTime(a asset.Item) time.Time {
	r := &e.balanceRefresh
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotTimes[a]
}

// balanceEventTimes returns the transaction time of the latest user data stream event to report each of an asset's
// currencies
func (e *Exchange) balanceEventTimes(a asset.Item) map[*currency.Item]time.Time {
	r := &e.balanceRefresh
	r.mu.Lock()
	defer r.mu.Unlock()
	return maps.Clone(r.eventTimes[a])
}

// refreshBalances refreshes an asset's available balances over REST once a user data stream balance event has stored
// its balances, or has the refresh running for it go once more. Refreshes run while the websocket is connected, or
// connecting, as a private connection streams while the manager dials the connections after it
func (e *Exchange) refreshBalances(ctx context.Context, a asset.Item) {
	r := &e.balanceRefresh
	r.mu.Lock()
	defer r.mu.Unlock()
	if !e.Websocket.IsConnected() && !e.Websocket.IsConnecting() {
		return
	}
	if r.running[a] {
		r.pending[a] = true
		return
	}
	if r.running == nil {
		r.running = make(map[asset.Item]bool)
		r.pending = make(map[asset.Item]bool)
	}
	r.running[a] = true
	// The refresh outlives the event: it stops when the websocket disconnects, and a later event's refresh must not end
	// with this event's context
	go e.runBalanceRefresh(context.WithoutCancel(ctx), a, cmp.Or(r.interval, balanceRefreshInterval))
}

// runBalanceRefresh refreshes an asset's available balances, then once more after each interval in which they changed
// again, or in which REST still lagged the stream or the refresh failed, at most balanceRefreshAttempts times for one
// event, while the websocket is connected or connecting; a refresh already under way when it disconnects finishes its
// request
func (e *Exchange) runBalanceRefresh(ctx context.Context, a asset.Item, interval time.Duration) {
	r := &e.balanceRefresh
	for attempts := 1; ; attempts++ {
		// Checked under the lock events take to ask for another refresh, so one made after a reconnection starts its own
		// rather than being cleared here
		r.mu.Lock()
		if !e.Websocket.IsConnected() && !e.Websocket.IsConnecting() {
			r.running[a], r.pending[a] = false, false
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
		lagging, err := e.refreshAvailableBalances(ctx, a)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s %s balance refresh after a user data stream balance update failed: %v", e.Name, a, err)
		}
		time.Sleep(interval)
		r.mu.Lock()
		again := r.pending[a]
		if again {
			// Another event changed the balances, so its refresh has attempts of its own
			attempts = 0
		} else {
			again = (lagging || err != nil) && attempts < balanceRefreshAttempts
		}
		r.pending[a] = false
		r.running[a] = again
		r.mu.Unlock()
		if !again {
			return
		}
	}
}

// refreshAvailableBalances stores the available balances and margin held that REST reports for an asset, for each
// currency nothing has stored since the request: a balance an event or a REST poll stored after it is newer. The user
// data stream reports each wallet balance as it changes, so a currency REST reports as last updated before the latest
// event to report it, or without an update time, is skipped as predating that event, and REST's available balance is
// only stored beside the wallet balance REST reports with it: a snapshot that disagrees predates an event, or an event
// still on its way will start another refresh. A currency without a balance takes REST's. It reports whether a currency
// the stream reported was skipped as predating its event or for disagreeing, so that REST is asked again. The snapshot
// is accepted under the balance store lock
func (e *Exchange) refreshAvailableBalances(ctx context.Context, a asset.Item) (lagging bool, err error) {
	requested := time.Now()
	subAccount, updated, _, err := e.accountBalances(ctx, a)
	if err != nil {
		return false, err
	}
	err = e.withBalanceStore(func() error {
		eventTimes := e.balanceEventTimes(a)
		var errs error
		for c, fetched := range subAccount.Balances {
			eventTime, streamed := eventTimes[c.Item]
			if streamed && updated[c.Item].Before(eventTime) {
				lagging = true
				continue
			}
			// A balance kept is left as it was, as storing it would stamp it as current
			_, err := e.Accounts.UpdateBalanceIf(ctx, "", a, c, func(stored *accounts.Balance) bool {
				if stored.UpdatedAt.After(requested) {
					return false
				}
				if stored.UpdatedAt.IsZero() || stored.Total == fetched.Total {
					stored.Total, stored.Hold, stored.Free = fetched.Total, fetched.Hold, fetched.Free
					return true
				}
				// Asking again cannot reconcile a currency the stream does not report
				lagging = lagging || streamed
				return false
			})
			errs = common.AppendError(errs, err)
		}
		return errs
	})
	return lagging, err
}

// keepStreamedBalances puts the stored balance in place of each currency a REST poll's snapshot reports as last updated
// before the latest user data stream event to report it, as refreshes skip it: the snapshot was taken before the event
// whose wallet balance is stored, however late the poll stores it, and storing the stored balance unchanged keeps its
// update time, which refreshes requested before the poll compare against. A currency the snapshot leaves out keeps its
// stored balance while its latest event is newer than every update the snapshot shows and than the snapshot's own time,
// asOf, where Binance gives one, rather than being removed by a snapshot that may predate it; a snapshot showing no
// update or time therefore removes none. The user data streams are the configured account's, so a poll of another
// account's credentials is left as it is. The caller holds the balance store lock until the snapshot is stored
func (e *Exchange) keepStreamedBalances(ctx context.Context, a asset.Item, subAccount *accounts.SubAccount, updated map[*currency.Item]time.Time, asOf time.Time) {
	eventTimes := e.balanceEventTimes(a)
	if len(eventTimes) == 0 {
		return
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return
	}
	if streamed := e.GetDefaultCredentials(); streamed == nil || *creds != *streamed {
		return
	}
	newest := asOf
	reported := make(map[*currency.Item]currency.Code, len(subAccount.Balances))
	for c := range subAccount.Balances {
		reported[c.Item] = c
		if updated[c.Item].After(newest) {
			newest = updated[c.Item]
		}
	}
	for item, eventTime := range eventTimes {
		c, ok := reported[item]
		switch {
		case ok && !updated[item].Before(eventTime), !ok && !eventTime.After(newest):
			continue
		case !ok:
			c = item.Currency()
		}
		if stored, err := e.Accounts.GetBalance("", creds, a, c); err == nil {
			subAccount.Balances.Set(c, stored)
		}
	}
}

// futuresPosition builds a position from its signed size. Hedge mode reports the side; one-way mode, BOTH, signs the
// amount
func futuresPosition(exchangeName string, a asset.Item, pair currency.Pair, positionSide string, amount decimal.Decimal, updated time.Time) futures.Position {
	p := futures.Position{
		Exchange:    exchangeName,
		Asset:       a,
		Pair:        pair,
		LatestSize:  amount.Abs(),
		Status:      order.Open,
		LastUpdated: updated,
	}
	switch {
	case positionSide == "LONG", positionSide != "SHORT" && amount.IsPositive():
		p.OpeningDirection = order.Long
	case positionSide == "SHORT", amount.IsNegative():
		p.OpeningDirection = order.Short
	}
	p.LatestDirection = p.OpeningDirection
	if amount.IsZero() {
		p.Status = order.Closed
		p.CloseDate = updated
	}
	return p
}

// processFuturesOrderTradeUpdate relays an order update as an order detail. The event carries no creation time, so
// the order is dated by the update, which order stores keep from the first update they see. A fill's commission is
// that fill's alone, so it is reported with the fill as the order's trade rather than as the order's fee
func (e *Exchange) processFuturesOrderTradeUpdate(ctx context.Context, data []byte, a asset.Item) error {
	var resp FuturesOrderTradeUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	o := &resp.Order
	pair, err := e.userDataPair(o.Symbol, a)
	if err != nil {
		log.Warnf(log.WebsocketMgr, "%s %s order update: skipping order %d: %v", e.Name, a, o.OrderID, err)
		return nil
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return err
	}
	orderType, err := derivativesOrderType(o.OrderType)
	if err != nil {
		return err
	}
	status, err := derivativesOrderStatus(o.OrderStatus)
	if err != nil {
		return err
	}
	tif, err := derivativesTimeInForce(o.TimeInForce)
	if err != nil {
		return err
	}
	d := &order.Detail{
		TimeInForce:          tif,
		ReduceOnly:           o.IsReduceOnly,
		Price:                o.OriginalPrice.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.OrderFilledAccumulatedQuantity.Float64(),
		RemainingAmount:      remainingAmount(o.OriginalQuantity, o.OrderFilledAccumulatedQuantity),
		FeeAsset:             o.CommissionAsset,
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            a,
		Date:                 o.OrderTradeTime.Time(),
		LastUpdated:          o.OrderTradeTime.Time(),
		Pair:                 pair,
	}
	if a == asset.USDTMarginedFutures {
		// Matches the quote total the order queries report as the cost; a COIN-M order's cost is in the base asset,
		// which needs the contract size the event lacks
		d.Cost = o.AveragePrice.Decimal().Mul(o.OrderFilledAccumulatedQuantity.Decimal()).InexactFloat64()
	}
	if orderType == order.TrailingStop || o.OriginalOrderType == "TRAILING_STOP_MARKET" {
		// Binance documents the stop price as meaningless for trailing stops, which trigger at their activation price;
		// the original type still names a trailing stop once it has triggered as another type
		d.TriggerPrice = o.ActivationPrice.Float64()
	}
	// CALCULATED is a liquidation's execution, which fills the order as a trade does
	if o.ExecutionType == "TRADE" || o.ExecutionType == "CALCULATED" {
		d.Trades = []order.TradeHistory{{
			Price:     o.LastFilledPrice.Float64(),
			Amount:    o.OrderLastFilledQuantity.Float64(),
			Fee:       o.Commission.Float64(),
			Exchange:  e.Name,
			TID:       strconv.FormatUint(o.TradeID, 10),
			Type:      orderType,
			Side:      side,
			Timestamp: o.OrderTradeTime.Time(),
			IsMaker:   o.IsMaker,
			FeeAsset:  o.CommissionAsset.String(),
		}}
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

// processFuturesAlgoUpdate relays an ALGO_UPDATE event, then its algo order as an order detail identified by the algo
// ID, the ID SubmitOrder returns for conditional orders, with the status and type conversions of the REST algo order
// queries. The detail carries no trades: the updates of the order a triggered algo order places report its fills
func (e *Exchange) processFuturesAlgoUpdate(ctx context.Context, data []byte, a asset.Item) error {
	var resp FuturesAlgoUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, &resp); err != nil {
		return err
	}
	o := &resp.Order
	pair, err := e.userDataPair(o.Symbol, a)
	if err != nil {
		log.Warnf(log.WebsocketMgr, "%s %s algo update: skipping the order detail of algo order %d: %v", e.Name, a, o.AlgoID, err)
		return nil
	}
	vars, err := compatibleOrderVars(a, o.Side, "", o.OrderType, o.TimeInForce)
	if err != nil {
		return err
	}
	status, err := algoOrderStatus(o.AlgoStatus)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		TimeInForce:          vars.TimeInForce,
		ReduceOnly:           o.IsReduceOnly,
		Price:                o.OrderPrice.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.TriggerPrice.Float64(),
		AverageExecutedPrice: o.AverageFillPrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.AlgoID, 10),
		ClientOrderID:        o.ClientAlgoID,
		Type:                 vars.OrderType,
		Side:                 vars.Side,
		Status:               status,
		AssetType:            a,
		Date:                 resp.TransactionTime.Time(),
		LastUpdated:          resp.TransactionTime.Time(),
		Pair:                 pair,
	})
}

// derivativesOrderType converts a USDⓈ-M, COIN-M or options order type, matching the REST order conversions. STOP
// and TAKE_PROFIT are Binance's stop and take profit limit orders
func derivativesOrderType(orderType string) (order.Type, error) {
	switch orderType {
	case "LIMIT":
		return order.Limit, nil
	case marketOrderType:
		return order.Market, nil
	case "STOP":
		return order.StopLimit, nil
	case "STOP_MARKET":
		return order.StopMarket, nil
	case "TAKE_PROFIT":
		return order.TakeProfitLimit, nil
	case "TAKE_PROFIT_MARKET":
		return order.TakeProfitMarket, nil
	case "TRAILING_STOP_MARKET":
		return order.TrailingStop, nil
	case "LIQUIDATION":
		return order.Liquidation, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// derivativesOrderStatus converts a USDⓈ-M, COIN-M or options order status. EXPIRED_IN_MATCH means self-trade
// prevention expired the order, which closes it like any expiry (order.STP is neither an active nor an inactive status);
// NEW_INSURANCE and NEW_ADL mark liquidation orders
func derivativesOrderStatus(status string) (order.Status, error) {
	switch status {
	case "EXPIRED_IN_MATCH":
		return order.Expired, nil
	case "NEW_INSURANCE":
		return order.Liquidated, nil
	case "NEW_ADL":
		return order.AutoDeleverage, nil
	default:
		// The order package's own error cannot be matched outside it
		s, err := order.StringToOrderStatus(status)
		if err != nil {
			return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, status)
		}
		return s, nil
	}
}

// derivativesTimeInForce converts a time in force. GTX, Binance's post only time in force, and RPI orders are post
// only, as SubmitOrder reports a post only order, and GTE_GTC, the time in force of close position orders, keeps an
// order until it is cancelled or the position it closes is gone
func derivativesTimeInForce(timeInForce string) (order.TimeInForce, error) {
	switch timeInForce {
	case "GTX", "RPI":
		return order.PostOnly, nil
	case "GTE_GTC":
		return order.GoodTillCancel, nil
	default:
		return order.StringToTimeInForce(timeInForce)
	}
}

// processAndSendTicker stores a ticker before relaying it, so a ticker that fails processing is never relayed
func (e *Exchange) processAndSendTicker(ctx context.Context, p *ticker.Price) error {
	if err := ticker.ProcessTicker(p); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, p)
}

// processAndSendTickers stores a batch of tickers and relays only those that were processed
func (e *Exchange) processAndSendTickers(ctx context.Context, prices []ticker.Price) error {
	processed, err := ticker.ProcessBatch(prices)
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}
