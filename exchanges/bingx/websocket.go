package bingx

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Exchange-specific subscription channels, whose payloads are relayed as their own types. Mark prices stream on the
// perpetual connections only
const (
	WebsocketLastPriceChannel  = "lastPrice"
	WebsocketBookTickerChannel = "bookTicker"
	WebsocketMarkPriceChannel  = "markPrice"
)

const (
	websocketSpotFilter                = "spot"
	websocketSpotAccountFilter         = "spot-account"
	websocketSwapFilter                = "usdt-margined"
	websocketSwapAccountFilter         = "usdt-margined-account"
	websocketCoinMarginedFilter        = "coin-margined"
	websocketCoinMarginedAccountFilter = "coin-margined-account"

	// BingX allows 200 topics a connection
	websocketMaxSubscriptionsPerConnection = 200

	// The USDT-M servers send a text heartbeat; spot and coin-M send WebsocketPing
	websocketSwapPing = "Ping"
	websocketSwapPong = "Pong"

	// A listen key is valid for 60 minutes after its creation or last extension, and BingX recommends extending it
	// every 30
	websocketListenKeyValidity          = 60 * time.Minute
	websocketListenKeyKeepaliveInterval = 30 * time.Minute

	// A failed extension is tried again after websocketListenKeyExtensionRetryDelay, then after twice the last wait
	// each time, while the attempt comes at least that first delay before the key expires
	websocketListenKeyExtensionRetryDelay = time.Minute

	// A failed listen key request makes the next attempt wait websocketListenKeyRetryDelay, and each failure after
	// it twice the last wait, up to websocketListenKeyRetryMaxDelay
	websocketListenKeyRetryDelay    = 5 * time.Second
	websocketListenKeyRetryMaxDelay = 5 * time.Minute
)

var (
	errWebsocketUnhandledMessage              = errors.New("unhandled websocket message")
	errWebsocketUnhandledEvent                = errors.New("unhandled websocket event")
	errWebsocketErrorMessage                  = errors.New("websocket error message")
	errWebsocketRequestRejected               = errors.New("websocket request rejected")
	errWebsocketConnectionUnknown             = errors.New("websocket connection not dialled by this exchange")
	errWebsocketConnectionRetired             = errors.New("websocket connection retired after an unanswered request")
	errWebsocketUnsupportedLevels             = errors.New("unsupported order book levels")
	errWebsocketOrderbookSubscriptionConflict = errors.New("order book subscription conflicts with another for the same book")
	errWebsocketInvalidSymbol                 = errors.New("invalid symbol")
	errWebsocketAccountDial                   = errors.New("unable to connect to the account data stream")
	errWebsocketListenKeyUnavailable          = errors.New("listen key unavailable")
	errWebsocketListenKeyBackoff              = errors.New("listen key requests waiting after a failure")
	errWebsocketListenKeyExtension            = errors.New("listen key extension failed")
	errWebsocketListenKeyExpired              = errors.New("listen key expired")
)

// websocketKlineIntervals are the candle intervals every product streams
var websocketKlineIntervals = []kline.Interval{
	kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour,
	kline.FourHour, kline.SixHour, kline.EightHour, kline.TwelveHour, kline.OneDay, kline.ThreeDay, kline.OneWeek,
	kline.OneMonth,
}

// defaultSubscriptions are the subscriptions of each asset: market data, each expanded to the asset's enabled pairs,
// and the account streams, which need authenticated websocket use
var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// websocketState is what the websocket connections share: each dialled connection's generation and session, the order
// book each connection feeds, the listen key keepers and the recovery in progress
type websocketState struct {
	mu          sync.Mutex
	generation  uint64
	connections map[websocket.Connection]*websocketConnection
	// sessions holds the websocket sessions being watched for their teardown
	sessions map[<-chan struct{}]bool
	// keepers holds the listen key keeper of each REST base URL
	keepers    map[string]*websocketListenKeyKeeper
	keyRetries map[string]websocketListenKeyRetry
	recovering <-chan struct{}
	// reconnectAt is when the flush bringing back the account streams left out while listen key requests wait runs,
	// zero when none is due; reconnects numbers the flushes scheduled, so a superseded one does not run
	reconnectAt time.Time
	reconnects  uint64
	// background counts the recoveries, scheduled flushes and balance refreshes, which run apart from the manager's
	// wait group
	background sync.WaitGroup

	booksMu sync.Mutex
	books   map[key.PairAsset]websocketBookFeed

	orderRelay perpetualOrderRelay

	// keepaliveInterval is the time between listen key extensions, websocketListenKeyKeepaliveInterval when zero
	keepaliveInterval time.Duration
	// keyValidity is how long a listen key lasts after its last extension, websocketListenKeyValidity when zero
	keyValidity time.Duration
	// extensionRetryDelay is the first wait after a failed extension, websocketListenKeyExtensionRetryDelay when zero
	extensionRetryDelay time.Duration
	// retryDelay is the first wait after a failed listen key request, websocketListenKeyRetryDelay when zero
	retryDelay time.Duration
}

// websocketConnection is the state of a connection a Connector dialled
type websocketConnection struct {
	generation uint64
	// session is the manager's ShutdownC when the connection was dialled, closed when the connection's websocket
	// session is torn down
	session <-chan struct{}
	// gate holds the request in flight on a coin-M market connection, whose error replies carry no id; it is nil on
	// other connections
	gate chan struct{}
	// credentials are those of the listen key an account connection was dialled with, whose account its events report;
	// they are nil on market connections
	credentials *accounts.Credentials
	// account is the asset whose product's account events an account connection streams, asset.Empty on market
	// connections
	account asset.Item
	// fees totals the fees of the open orders a perpetual account connection streams, whose events report each fill's
	// fee alone
	fees perpetualOrderFeeTotals
	// heldOrders holds a perpetual account connection's order events while it reads its open orders, so their fills add
	// to the totals the read starts; holding is set until the read ends, and held keeps the events in their order
	heldOrders sync.Mutex
	holding    bool
	held       [][]byte

	mu               sync.Mutex
	inFlightID       string
	inFlightDataType string
	retired          bool
}

// websocketBookFeed is the stream feeding an order book and the connection generation it arrives on
type websocketBookFeed struct {
	conn       websocket.Connection
	generation uint64
	stream     string
}

// websocketListenKeyKeeper keeps the listen key of a REST host's account connections alive for one websocket session
type websocketListenKeyKeeper struct {
	host       string
	ePath      exchange.URL
	key        string
	interval   time.Duration
	validity   time.Duration
	retryDelay time.Duration
	session    <-chan struct{}
	// credentials are those the key was created for, whose account the key's connections stream and which extend it
	credentials *accounts.Credentials
	// conns are the connections dialled with the key, guarded by websocketState.mu
	conns []websocket.Connection
	done  chan struct{}
}

// websocketListenKeyRetry is the wait a REST host's listen key requests are under after failing
type websocketListenKeyRetry struct {
	wait time.Duration
	next time.Time
}

// wsConnectionSetups returns the connection setups of each product's market and account data, in the order the
// manager connects them. A generator returns nothing for a disabled asset, or for an account connection without
// authenticated websocket use, so the manager skips that connection. Spot account streams take subscription
// requests; the perpetual account connections are pushed every event, so their subscriptions are recorded alone
func (e *Exchange) wsConnectionSetups(exch *config.Exchange) ([]*websocket.ConnectionSetup, error) {
	if err := common.NilGuard(exch); err != nil {
		return nil, err
	}
	products := []struct {
		asset                       asset.Item
		websocketURL, restURL       exchange.URL
		marketFilter, accountFilter string
		serialRequests              bool
		marketHandler               func(context.Context, websocket.Connection, []byte) error
		accountHandler              func(context.Context, websocket.Connection, []byte) error
		accountSubscriber           func(context.Context, websocket.Connection, subscription.List) error
		accountUnsubscriber         func(context.Context, websocket.Connection, subscription.List) error
	}{
		{asset.Spot, exchange.WebsocketSpot, exchange.RestSpot, websocketSpotFilter, websocketSpotAccountFilter, false, e.wsHandleSpotData, e.wsHandleSpotAccountData, e.wsSubscribe, e.wsUnsubscribe},
		{asset.USDTMarginedFutures, exchange.WebsocketUSDTMargined, exchange.RestUSDTMargined, websocketSwapFilter, websocketSwapAccountFilter, false, e.wsHandleSwapData, e.wsHandleSwapAccountData, e.wsTrackPushedSubscriptions, e.wsUntrackPushedSubscriptions},
		// Coin-M requests go one at a time, as coin-M error replies carry no id
		{asset.CoinMarginedFutures, exchange.WebsocketCoinMargined, exchange.RestCoinMargined, websocketCoinMarginedFilter, websocketCoinMarginedAccountFilter, true, e.wsHandleCoinMarginedData, e.wsHandleCoinMarginedAccountData, e.wsTrackPushedSubscriptions, e.wsUntrackPushedSubscriptions},
	}
	setups := make([]*websocket.ConnectionSetup, 0, len(products)*2)
	for _, p := range products {
		wsURL, err := e.API.Endpoints.GetURL(p.websocketURL)
		if err != nil {
			return nil, err
		}
		setups = append(setups,
			&websocket.ConnectionSetup{
				MessageFilter: p.marketFilter,
				URL:           wsURL,
				Connector: func(ctx context.Context, conn websocket.Connection) error {
					return e.wsConnectMarket(ctx, conn, p.serialRequests)
				},
				GenerateSubscriptions: func() (subscription.List, error) { return e.wsGenerateMarketSubscriptions(p.asset) },
				Subscriber:            e.wsSubscribe,
				Unsubscriber:          e.wsUnsubscribe,
				Handler:               p.marketHandler,
				ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			},
			&websocket.ConnectionSetup{
				MessageFilter: p.accountFilter,
				URL:           wsURL,
				Connector: func(ctx context.Context, conn websocket.Connection) error {
					return e.wsConnectAccount(ctx, conn, p.asset, p.restURL)
				},
				GenerateSubscriptions: func() (subscription.List, error) { return e.wsGenerateAccountSubscriptions(p.asset, p.restURL) },
				Subscriber:            p.accountSubscriber,
				Unsubscriber:          p.accountUnsubscriber,
				Handler:               p.accountHandler,
				ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			})
	}
	return setups, nil
}

// wsDialer returns the dialer of every connection
func (e *Exchange) wsDialer() *gws.Dialer {
	return &gws.Dialer{HandshakeTimeout: e.Config.HTTPTimeout, Proxy: http.ProxyFromEnvironment}
}

// wsConnectMarket dials a market data connection. The servers send heartbeats, which the handlers answer
func (e *Exchange) wsConnectMarket(ctx context.Context, conn websocket.Connection, serialRequests bool) error {
	if err := conn.Dial(ctx, e.wsDialer(), http.Header{}, nil); err != nil {
		return fmt.Errorf("%s unable to connect to the market data streams: %w", e.Name, err)
	}
	e.wsTrackConnection(conn, e.Websocket.ShutdownC, serialRequests, asset.Empty, nil)
	return nil
}

// wsConnectAccount dials the account data connection of asset a's product with the listen key of the REST host serving
// the product's account endpoints, ePath. The key is passed as a dial query value, keeping it out of the connection's
// URL, and out of the error a failed dial returns. The connection streams the account of the credentials the key was
// created for, which it keeps, so the balances its events store and refresh are that account's whatever the exchange's
// default credentials become. Once a perpetual product's account connection is up, its balances are refreshed for the
// connection's session, fetching the changes made while it was disconnected, which include those of a refresh the end
// of the previous session cut off, and its open orders are read to start their fee totals
func (e *Exchange) wsConnectAccount(ctx context.Context, conn websocket.Connection, a asset.Item, ePath exchange.URL) error {
	session := e.Websocket.ShutdownC
	keeper, err := e.wsListenKey(ctx, ePath, session)
	if err != nil {
		return err
	}
	if err := conn.Dial(ctx, e.wsDialer(), http.Header{}, url.Values{"listenKey": {keeper.key}}); err != nil {
		return fmt.Errorf("%w: %s", errWebsocketAccountDial, wsRedact(err.Error(), keeper.key))
	}
	state := e.wsTrackConnection(conn, session, false, a, keeper.credentials)
	e.wsAddKeeperConnection(keeper, conn)
	if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
		ctx = accounts.DeployCredentialsToContext(ctx, keeper.credentials)
		e.refreshBalances(ctx, a, session)
		e.wsReadOpenOrderFees(ctx, state)
	}
	return nil
}

// wsReadOpenOrderFees reads the open orders of a perpetual account connection's account once the connection is up, so
// the fee totals of the orders placed before it start from the fees each had paid rather than from the first fill it
// streams. The connection's order events are held until the read ends and then handled in turn, so their fills add to
// the totals the read starts, while its other frames, heartbeats included, are handled as they arrive. The read ends
// with the connection's session, and the manager waits for it before starting another, so the held events are relayed
// before that session's. The read starts the connection's totals although the connection closes first, and within the
// session a held event that a connection dialled since has taken its order past is dropped (perpetualOrderRelay)
func (e *Exchange) wsReadOpenOrderFees(ctx context.Context, state *websocketConnection) {
	state.heldOrders.Lock()
	state.holding = true
	state.heldOrders.Unlock()
	// The read outlives the connection attempt, whose context ends with it, and ends with the session instead
	ctx, cancel := context.WithCancel(context.WithValue(context.WithoutCancel(ctx), perpetualOrderFeeReadOwnerKey{}, state))
	e.Websocket.Wg.Add(1)
	e.wsState.background.Go(func() {
		defer e.Websocket.Wg.Done()
		defer cancel()
		go func() {
			select {
			case <-state.session:
				cancel()
			case <-ctx.Done():
			}
		}()
		if _, err := e.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: state.account, Side: order.AnySide, Type: order.AnyType}); err != nil {
			log.Errorf(log.WebsocketMgr, "%s %s open order fee read failed: %v", e.Name, state.account, err)
		}
		e.wsReleaseOrderEvents(ctx, state)
	})
}

// wsRedact replaces a secret in an error message
func wsRedact(message, secret string) string {
	if secret == "" {
		return message
	}
	return strings.ReplaceAll(message, secret, "[redacted]")
}

// wsTrackConnection records a dialled connection under a new generation, an account connection with the asset whose
// product's account events it streams and the credentials of its listen key, first forgetting connections no longer
// connected, such as those a failed connection attempt closed, and invalidating the books they fed. The session the
// connection belongs to is watched so that its connections' books are invalidated once it is torn down
func (e *Exchange) wsTrackConnection(conn websocket.Connection, session <-chan struct{}, serialRequests bool, account asset.Item, credentials *accounts.Credentials) *websocketConnection {
	s := &e.wsState
	s.mu.Lock()
	if s.connections == nil {
		s.connections = make(map[websocket.Connection]*websocketConnection)
		s.sessions = make(map[<-chan struct{}]bool)
	}
	gone := make(map[uint64]bool)
	for c, state := range s.connections {
		if !wsIsConnected(c) {
			gone[state.generation] = true
			delete(s.connections, c)
		}
	}
	s.generation++
	state := &websocketConnection{generation: s.generation, session: session, credentials: credentials, account: account}
	if serialRequests {
		state.gate = make(chan struct{}, 1)
	}
	s.connections[conn] = state
	if session != nil && !s.sessions[session] {
		s.sessions[session] = true
		e.Websocket.Wg.Go(func() {
			<-session
			e.wsEndSession(session)
		})
	}
	s.mu.Unlock()
	if len(gone) != 0 {
		e.wsInvalidateBooks(gone)
	}
	return state
}

// wsEndSession forgets the connections of a torn down session and invalidates the books they fed
func (e *Exchange) wsEndSession(session <-chan struct{}) {
	s := &e.wsState
	s.mu.Lock()
	ended := make(map[uint64]bool)
	for c, state := range s.connections {
		if state.session == session {
			ended[state.generation] = true
			delete(s.connections, c)
		}
	}
	delete(s.sessions, session)
	if s.recovering == session {
		s.recovering = nil
	}
	s.mu.Unlock()
	e.wsInvalidateBooks(ended)
}

// wsConnection returns the state of a connection a Connector dialled
func (e *Exchange) wsConnection(conn websocket.Connection) (*websocketConnection, error) {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.connections[conn]; ok {
		return state, nil
	}
	return nil, errWebsocketConnectionUnknown
}

// wsAccountConnections returns the account connections up for asset a's product and the account of creds, which that
// account's order events on the product arrive on
func (e *Exchange) wsAccountConnections(a asset.Item, creds *accounts.Credentials) []*websocketConnection {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	var states []*websocketConnection
	for conn, state := range s.connections {
		if state.account == a && state.credentials.Equal(creds) && wsIsConnected(conn) {
			states = append(states, state)
		}
	}
	return states
}

// wsIsConnected reports whether a connection is up, taking one that cannot tell as up
func wsIsConnected(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return !ok || c.IsConnected()
}

// wsBookKey returns the book an order book subscription feeds
func wsBookKey(s *subscription.Subscription) (key.PairAsset, error) {
	if len(s.Pairs) != 1 {
		return key.PairAsset{}, fmt.Errorf("%w: %s", subscription.ErrNotSinglePair, s)
	}
	return key.PairAsset{Base: s.Pairs[0].Base.Item, Quote: s.Pairs[0].Quote.Item, Asset: s.Asset}, nil
}

// wsClaimBook records conn as the feed of the book an order book subscription streams. A book another live stream
// feeds is refused, as their partial depth pushes would keep replacing each other's books
func (e *Exchange) wsClaimBook(conn websocket.Connection, generation uint64, s *subscription.Subscription) error {
	k, err := wsBookKey(s)
	if err != nil {
		return err
	}
	st := &e.wsState
	st.booksMu.Lock()
	defer st.booksMu.Unlock()
	if feed, ok := st.books[k]; ok && feed.stream != s.QualifiedChannel && wsIsConnected(feed.conn) {
		return fmt.Errorf("%w: %s is fed by %s", errWebsocketOrderbookSubscriptionConflict, s.QualifiedChannel, feed.stream)
	}
	if st.books == nil {
		st.books = make(map[key.PairAsset]websocketBookFeed)
	}
	st.books[k] = websocketBookFeed{conn: conn, generation: generation, stream: s.QualifiedChannel}
	return nil
}

// wsReleaseBook forgets the feed of the book an order book subscription streamed on conn and invalidates the book, as
// nothing updates it any more
func (e *Exchange) wsReleaseBook(conn websocket.Connection, s *subscription.Subscription) {
	k, err := wsBookKey(s)
	if err != nil {
		return
	}
	st := &e.wsState
	st.booksMu.Lock()
	defer st.booksMu.Unlock()
	if feed, ok := st.books[k]; ok && feed.conn == conn && feed.stream == s.QualifiedChannel {
		delete(st.books, k)
		// A book whose first push never arrived has nothing to invalidate
		_ = e.Websocket.Orderbook.InvalidateOrderbook(k.Pair(), k.Asset)
	}
}

// wsInvalidateBooks forgets the feeds of the books the connection generations given fed and invalidates the books
func (e *Exchange) wsInvalidateBooks(generations map[uint64]bool) {
	st := &e.wsState
	st.booksMu.Lock()
	defer st.booksMu.Unlock()
	for k, feed := range st.books {
		if generations[feed.generation] {
			delete(st.books, k)
			_ = e.Websocket.Orderbook.InvalidateOrderbook(k.Pair(), k.Asset)
		}
	}
}

// wsLoadBook loads a partial depth push of a stream as its book's snapshot when the stream feeds the book on the
// connection the push arrived on, so a push from a replaced connection, or of a stream since unsubscribed, cannot
// reload the book
func (e *Exchange) wsLoadBook(ctx context.Context, conn websocket.Connection, stream string, book *orderbook.Book) error {
	st := &e.wsState
	st.mu.Lock()
	state, ok := st.connections[conn]
	st.mu.Unlock()
	if !ok {
		return nil
	}
	k := key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}
	st.booksMu.Lock()
	defer st.booksMu.Unlock()
	if feed, ok := st.books[k]; !ok || feed.conn != conn || feed.generation != state.generation || feed.stream != stream {
		return nil
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// wsSortLevels sorts a partial depth push's levels, bids highest first and asks lowest first, as spot and coin-M asks
// arrive highest first
func wsSortLevels(bids, asks orderbook.Levels) {
	slices.SortFunc(bids, func(a, b orderbook.Level) int { return cmp.Compare(b.Price, a.Price) })
	slices.SortFunc(asks, func(a, b orderbook.Level) int { return cmp.Compare(a.Price, b.Price) })
}

// wsFrameTime returns the time in milliseconds a frame carries under name, or the receipt time when it carries none
func wsFrameTime(respRaw []byte, name ...string) time.Time {
	if ms, err := jsonparser.GetInt(respRaw, name...); err == nil && ms > 0 {
		return time.UnixMilli(ms)
	}
	return time.Now()
}

// wsGenerateMarketSubscriptions expands the market data subscriptions of an asset, and the all-assets ones, into one
// subscription a stream. A disabled asset generates nothing, so its connection is skipped. Identical streams are kept
// once, before the manager counts them against a connection's capacity. A subscription that cannot be expanded, or an
// order book subscription left out for conflicting with another, is reported as partial generation, which leaves the
// others to subscribe
func (e *Exchange) wsGenerateMarketSubscriptions(a asset.Item) (subscription.List, error) {
	if !e.GetAssetTypes(true).Contains(a) || !e.IsAssetWebsocketSupported(a) {
		return subscription.List{}, nil
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, s := range e.Features.Subscriptions {
		if s.Authenticated || (s.Asset != a && s.Asset != asset.All) {
			continue
		}
		c := s.Clone()
		c.Asset = a
		subs = append(subs, c)
	}
	expanded, err := subs.ExpandTemplates(e)
	expanded, conflicts := wsOneOrderbookSubscription(wsDistinctSubscriptions(expanded))
	if err = common.AppendError(err, conflicts); err != nil {
		return expanded, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionPartial, a, err)
	}
	return expanded, nil
}

// wsGenerateAccountSubscriptions returns an asset's account data subscriptions, the configured MyOrders and MyAccount
// ones, when authenticated websocket use is possible. Nothing is returned while listen key requests to the REST host
// at ePath wait after a failure, so the other connections connect without the account streams, which a scheduled
// flush brings back once the wait ends. A perpetual account connection is pushed every event, so its subscriptions
// keep the connection while they are wanted
func (e *Exchange) wsGenerateAccountSubscriptions(a asset.Item, ePath exchange.URL) (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() || !e.GetAssetTypes(true).Contains(a) || !e.AreCredentialsValid(context.Background()) || e.wsListenKeyWaiting(ePath) {
		return subscription.List{}, nil
	}
	subs := make(subscription.List, 0, 2)
	var errs error
	for _, s := range e.Features.Subscriptions {
		if !s.Authenticated || (s.Asset != a && s.Asset != asset.All) {
			continue
		}
		stream, err := wsAccountStreamName(a, s.Channel)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		c := s.Clone()
		c.Asset = a
		c.Pairs = nil
		c.QualifiedChannel = stream
		subs = append(subs, c)
	}
	subs = wsDistinctSubscriptions(subs)
	if errs != nil {
		return subs, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionPartial, a, errs)
	}
	return subs, nil
}

// wsAccountStreamName returns the stream an account data channel subscribes to on spot, and the event it stands for
// on the perpetual account connections, where nothing is subscribed
func wsAccountStreamName(a asset.Item, channel string) (string, error) {
	switch channel {
	case subscription.MyOrdersChannel:
		if a == asset.Spot {
			return "spot.executionReport", nil
		}
		return "ORDER_TRADE_UPDATE", nil
	case subscription.MyAccountChannel:
		return "ACCOUNT_UPDATE", nil
	default:
		return "", fmt.Errorf("%w: %s %q", subscription.ErrNotSupported, a, channel)
	}
}

// wsDistinctSubscriptions keeps the first subscription to each stream, as an all-assets subscription and an asset's own
// can expand to the same one, which BingX would be sent twice and the subscription store refuses to hold twice
func wsDistinctSubscriptions(subs subscription.List) subscription.List {
	seen := make(map[string]bool, len(subs))
	kept := make(subscription.List, 0, len(subs))
	for _, s := range subs {
		if seen[s.QualifiedChannel] {
			continue
		}
		seen[s.QualifiedChannel] = true
		kept = append(kept, s)
	}
	return kept
}

// wsOneOrderbookSubscription keeps one order book subscription for each book, the first, returning the others as
// errors: a pair and asset have one stored book, which a second partial depth stream would keep replacing
func wsOneOrderbookSubscription(subs subscription.List) (subscription.List, error) {
	feeds := make(map[key.PairAsset]string)
	kept := make(subscription.List, 0, len(subs))
	var errs error
	for _, s := range subs {
		if s.Channel != subscription.OrderbookChannel {
			kept = append(kept, s)
			continue
		}
		k, err := wsBookKey(s)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if feed, ok := feeds[k]; ok {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s is fed by %s", errWebsocketOrderbookSubscriptionConflict, s.QualifiedChannel, feed))
			continue
		}
		feeds[k] = s.QualifiedChannel
		kept = append(kept, s)
	}
	return kept, errs
}

// websocketSubscriptionTemplate is the market data stream template, parsed once
var websocketSubscriptionTemplate = sync.OnceValues(func() (*template.Template, error) {
	return template.New("subscriptions.tmpl").Funcs(template.FuncMap{"stream": wsStreamName}).Parse(websocketSubscriptionTemplateText)
})

// websocketSubscriptionTemplateText names each pair's stream, <symbol>@<stream>
const websocketSubscriptionTemplateText = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- $stream := stream $.S $asset }}
	{{- range $pair := $pairs }}
		{{- $pair }}@{{ $stream }}
		{{- $.PairSeparator }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}
`

// GetSubscriptionTemplate returns the market data subscription template
func (e *Exchange) GetSubscriptionTemplate(*subscription.Subscription) (*template.Template, error) {
	return websocketSubscriptionTemplate()
}

// wsStreamName returns the stream a market data subscription takes after the symbol for an asset. Spot names its candle
// intervals in full, and partial depth streams name their levels; swap keeps the server's push interval for them
func wsStreamName(s *subscription.Subscription, a asset.Item) (string, error) {
	switch s.Channel {
	case subscription.TickerChannel:
		return "ticker", nil
	case subscription.AllTradesChannel:
		return "trade", nil
	case subscription.CandlesChannel:
		format := intervalToString
		if a == asset.Spot {
			format = spotWebsocketIntervalToString
		}
		interval, err := format(s.Interval)
		if err != nil {
			return "", err
		}
		return "kline_" + interval, nil
	case subscription.OrderbookChannel:
		switch s.Levels {
		case 5, 10, 20, 50, 100:
			return "depth" + strconv.Itoa(s.Levels), nil
		default:
			return "", fmt.Errorf("%w: %d", errWebsocketUnsupportedLevels, s.Levels)
		}
	case WebsocketLastPriceChannel, WebsocketBookTickerChannel:
		return s.Channel, nil
	case WebsocketMarkPriceChannel:
		if a != asset.Spot {
			return s.Channel, nil
		}
	}
	return "", fmt.Errorf("%w: %s %q", subscription.ErrNotSupported, a, s.Channel)
}

// wsParseInterval returns the interval a stream names, as format writes it
func wsParseInterval(s string, format func(kline.Interval) (string, error)) (kline.Interval, error) {
	for _, interval := range websocketKlineIntervals {
		if name, err := format(interval); err == nil && name == s {
			return interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, s)
}

// websocketUTCPlus8 is the time zone the spot and coin-M candle streams divide time in
var websocketUTCPlus8 = time.FixedZone("UTC+8", 8*60*60)

// wsUTCPlus8PeriodStart returns the start of the candle period a time falls in, in the time's location, as the spot and
// coin-M candle streams divide time: an interval's periods follow on from 1970-01-02 00:00 UTC+8, weeks from Monday
// 1970-01-05 00:00 UTC+8 and months from the 1st, all in UTC+8
func wsUTCPlus8PeriodStart(t time.Time, interval kline.Interval) time.Time {
	if interval == kline.OneMonth {
		local := t.In(websocketUTCPlus8)
		return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, websocketUTCPlus8).In(t.Location())
	}
	origin := (16 * time.Hour).Milliseconds()
	if interval == kline.OneWeek {
		origin += (3 * 24 * time.Hour).Milliseconds()
	}
	length := interval.Duration().Milliseconds()
	ms := t.UnixMilli()
	return time.UnixMilli(ms - ((ms-origin)%length+length)%length).In(t.Location())
}

// wsCandleValidation flags a candle as partial while asOf precedes its end: its start plus the interval, or for a month
// candle the start of the next month in zone, the time zone its stream divides time in. A spot candle's own end time is
// not used, as a month candle's is the end of its first minute
func wsCandleValidation(asOf, start time.Time, interval kline.Interval, zone *time.Location) string {
	end := start.Add(interval.Duration())
	if interval == kline.OneMonth {
		end = start.In(zone).AddDate(0, 1, 0)
	}
	if asOf.Before(end) {
		return kline.PartialCandle
	}
	return ""
}

// wsSubscribe subscribes to each subscription with a request of its own, so a refusal fails that subscription alone
func (e *Exchange) wsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.wsSubscribeOne(ctx, conn, l[0])
	}, 1)
}

// wsSubscribeOne subscribes to a stream and records the outcome in the subscription store. An unanswered subscription
// may still have subscribed the stream, which the stores no longer hold, so the connection is retired, as it is for an
// unanswered unsubscription: the websocket reconnects and subscribes again to what is wanted
func (e *Exchange) wsSubscribeOne(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	state, err := e.wsConnection(conn)
	if err != nil {
		return err
	}
	// Stored first, so a subscription the store already holds, and whose stream may feed a book, is refused before its
	// book is claimed
	if err := e.Websocket.AddSubscriptions(conn, s); err != nil {
		return err
	}
	if s.Channel == subscription.OrderbookChannel {
		if err := e.wsClaimBook(conn, state.generation, s); err != nil {
			return common.AppendError(fmt.Errorf("%w: %w", websocket.ErrSubscriptionFailure, err), e.Websocket.RemoveSubscriptions(conn, s))
		}
	}
	if err := e.wsSendRequest(ctx, conn, state, WebsocketRequestTypeSubscribe, s.QualifiedChannel); err != nil {
		e.wsReleaseBookClaim(conn, s)
		if wsRequestUnanswered(ctx, err) {
			e.wsRequestRecovery(state.session, fmt.Errorf("%w: %s: %w", errWebsocketConnectionRetired, s.QualifiedChannel, err))
		}
		return common.AppendError(fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionFailure, s.QualifiedChannel, err), e.Websocket.RemoveSubscriptions(conn, s))
	}
	return s.SetState(subscription.SubscribedState)
}

// wsReleaseBookClaim releases the book an order book subscription fed on conn, once it has been unsubscribed or failed
// to subscribe
func (e *Exchange) wsReleaseBookClaim(conn websocket.Connection, s *subscription.Subscription) {
	if s.Channel == subscription.OrderbookChannel {
		e.wsReleaseBook(conn, s)
	}
}

// wsUnsubscribe unsubscribes from each subscription with a request of its own
func (e *Exchange) wsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.wsUnsubscribeOne(ctx, conn, l[0])
	}, 1)
}

// wsUnsubscribeOne unsubscribes from a stream, invalidating an order book it fed, and removes it from the store. A
// refused unsubscription leaves the subscription subscribed, in both stores and with any book it feeds, so it can be
// tried again. An unanswered one may still have stopped the stream, so the subscription stays subscribed in both stores
// while the book it fed is invalidated, and the connection is retired: the websocket reconnects and subscribes again
// to what is wanted
func (e *Exchange) wsUnsubscribeOne(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) error {
	state, err := e.wsConnection(conn)
	if err != nil {
		return err
	}
	if err := s.SetState(subscription.UnsubscribingState); err != nil {
		return err
	}
	if err := e.wsSendRequest(ctx, conn, state, WebsocketRequestTypeUnsubscribe, s.QualifiedChannel); err != nil {
		if wsRequestUnanswered(ctx, err) {
			e.wsReleaseBookClaim(conn, s)
			e.wsRequestRecovery(state.session, fmt.Errorf("%w: %s: %w", errWebsocketConnectionRetired, s.QualifiedChannel, err))
		}
		return common.AppendError(fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionFailure, s.QualifiedChannel, err), s.SetState(subscription.SubscribedState))
	}
	e.wsReleaseBookClaim(conn, s)
	return e.Websocket.RemoveSubscriptions(conn, s)
}

// wsTrackPushedSubscriptions records the subscriptions of a perpetual account connection as subscribed. BingX pushes
// every event to the connection without a request, so the subscriptions only keep it while they are wanted
func (e *Exchange) wsTrackPushedSubscriptions(_ context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// wsUntrackPushedSubscriptions removes the subscriptions of a perpetual account connection, which a flush closes once
// none are left
func (e *Exchange) wsUntrackPushedSubscriptions(_ context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}

// wsSendRequest sends a subscription or unsubscription request and returns the refusal its reply reports, if any
func (e *Exchange) wsSendRequest(ctx context.Context, conn websocket.Connection, state *websocketConnection, requestType WebsocketRequestType, dataType string) error {
	req := &WebsocketSubscriptionRequest{ID: e.MessageID(), RequestType: requestType, DataType: dataType}
	var respRaw []byte
	var err error
	if state.gate != nil {
		respRaw, err = e.wsSendSerialRequest(ctx, conn, state, req)
	} else {
		respRaw, err = conn.SendMessageReturnResponse(ctx, websocketSubscriptionRate, req.ID, req)
	}
	if err != nil {
		return err
	}
	var resp *WebsocketSubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%w: %s", errWebsocketRequestRejected, respRaw)
	}
	if resp.Code != 0 {
		return fmt.Errorf("%w: %w", errWebsocketRequestRejected, &APIError{Code: resp.Code, Message: resp.Message})
	}
	return nil
}

// wsSendSerialRequest sends a request on a coin-M market connection once the request before it has its reply,
// recording it as the request in flight for a reply without an id. A request left without a reply retires the
// connection, as the reply may still arrive and be taken for a later request's: later requests fail at once and the
// websocket reconnects
func (e *Exchange) wsSendSerialRequest(ctx context.Context, conn websocket.Connection, state *websocketConnection, req *WebsocketSubscriptionRequest) ([]byte, error) {
	select {
	case state.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-state.gate }()
	state.mu.Lock()
	if state.retired {
		state.mu.Unlock()
		return nil, errWebsocketConnectionRetired
	}
	state.inFlightID, state.inFlightDataType = req.ID, req.DataType
	state.mu.Unlock()

	respRaw, err := conn.SendMessageReturnResponse(ctx, websocketSubscriptionRate, req.ID, req)

	state.mu.Lock()
	state.inFlightID, state.inFlightDataType = "", ""
	unanswered := wsRequestUnanswered(ctx, err)
	if unanswered {
		state.retired = true
	}
	state.mu.Unlock()
	if unanswered {
		e.wsRequestRecovery(state.session, fmt.Errorf("%w: %s: %w", errWebsocketConnectionRetired, req.DataType, err))
	}
	return respRaw, err
}

// wsRequestUnanswered reports whether a request sent with ctx was left without a reply, by timing out, by ctx ending or
// by its write failing on the network, which leaves unknown whether BingX acted on it. A deadline error while ctx is
// still live is the rate limiter refusing to wait past the deadline, before anything was sent
func wsRequestUnanswered(ctx context.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return false
	}
	var netErr net.Error
	return errors.Is(err, websocket.ErrSignatureTimeout) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr)
}

// wsRouteCoinMarginedError routes a coin-M error reply, which carries no id, to the request in flight when its message
// names the request's symbol, stream or the part of the stream after its underscore, and reports it otherwise
func (e *Exchange) wsRouteCoinMarginedError(conn websocket.Connection, respRaw []byte, code int64) error {
	msg, _ := jsonparser.GetString(respRaw, "msg")
	if state, err := e.wsConnection(conn); err == nil && state.gate != nil {
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.inFlightID != "" && wsErrorNamesRequest(msg, state.inFlightDataType) {
			return conn.RequireMatchWithData(state.inFlightID, respRaw)
		}
	}
	return fmt.Errorf("%w: %w", errWebsocketErrorMessage, &APIError{Code: code, Message: msg})
}

// wsErrorNamesRequest reports whether an error message names a request's symbol, as BTC-USD or BTC_USD, its stream, or
// the part of its stream after an underscore, such as the 1min of kline_1min
func wsErrorNamesRequest(msg, dataType string) bool {
	symbol, stream, _ := strings.Cut(dataType, "@")
	names := []string{symbol, strings.ReplaceAll(symbol, "-", "_"), stream}
	if _, suffix, ok := strings.Cut(stream, "_"); ok {
		names = append(names, suffix)
	}
	for _, name := range names {
		if name != "" && strings.Contains(msg, name) {
			return true
		}
	}
	return false
}

// wsRequestRecovery shuts the websocket down for the connection monitor to connect every connection again, the account
// connections with a new listen key, reporting whether it started to. A session is recovered once, and a session
// already torn down is left alone. The shutdown runs apart from the manager's wait group and from any reader, as
// shutting down waits for both. The manager's API cannot check the session and shut down in one step, so a
// reconnection made between the two is shut down too, and the connection monitor connects it again
func (e *Exchange) wsRequestRecovery(session <-chan struct{}, reason error) bool {
	s := &e.wsState
	s.mu.Lock()
	if session == nil || s.recovering == session {
		s.mu.Unlock()
		return false
	}
	s.recovering = session
	s.background.Add(1)
	s.mu.Unlock()
	log.Warnf(log.WebsocketMgr, "%s websocket: reconnecting: %v", e.Name, reason)
	go func() {
		defer s.background.Done()
		select {
		case <-session:
			return
		default:
		}
		if err := e.Websocket.Shutdown(); err != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket: unable to shut down to reconnect: %v", e.Name, err)
		}
	}()
	return true
}

// wsSession returns the session a connection belongs to, or nil for a connection already forgotten
func (e *Exchange) wsSession(conn websocket.Connection) <-chan struct{} {
	if state, err := e.wsConnection(conn); err == nil {
		return state.session
	}
	return nil
}

// wsListenKey returns the keeper of the listen key the account connections to a product's REST host use in this
// session, creating the key and its keeper on the first request, for the credentials ctx names then. A key refused for
// the credentials turns authenticated websocket use off, so the next connection attempt connects the public streams
// alone; other failures make the requests that follow wait, longer after each failure, and schedule the flush that
// connects the account streams once the wait ends
func (e *Exchange) wsListenKey(ctx context.Context, ePath exchange.URL, session <-chan struct{}) (*websocketListenKeyKeeper, error) {
	host, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return nil, err
	}
	s := &e.wsState
	s.mu.Lock()
	if k := s.keepers[host]; k != nil && k.session == session {
		s.mu.Unlock()
		return k, nil
	}
	if r, ok := s.keyRetries[host]; ok && time.Now().Before(r.next) {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: next attempt in %s", errWebsocketListenKeyBackoff, time.Until(r.next).Round(time.Millisecond))
	}
	s.mu.Unlock()

	creds, err := e.GetCredentials(ctx)
	var listenKey string
	if err == nil {
		// Only the key's own request takes these credentials: a reconnect scheduled after a failure resolves the
		// credentials current when it runs
		listenKey, err = e.generateListenKey(accounts.DeployCredentialsToContext(ctx, creds), ePath)
	}
	if err != nil {
		if wsCredentialsRefused(err) {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		} else {
			e.wsScheduleAccountReconnect(ctx, e.wsRecordListenKeyFailure(host))
		}
		return nil, fmt.Errorf("%w: %w", errWebsocketListenKeyUnavailable, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keyRetries, host)
	if k := s.keepers[host]; k != nil && k.session == session {
		return k, nil
	}
	if s.keepers == nil {
		s.keepers = make(map[string]*websocketListenKeyKeeper)
	}
	k := &websocketListenKeyKeeper{
		host:        host,
		ePath:       ePath,
		key:         listenKey,
		credentials: creds,
		interval:    cmp.Or(s.keepaliveInterval, websocketListenKeyKeepaliveInterval),
		validity:    cmp.Or(s.keyValidity, websocketListenKeyValidity),
		retryDelay:  cmp.Or(s.extensionRetryDelay, websocketListenKeyExtensionRetryDelay),
		session:     session,
		done:        make(chan struct{}),
	}
	s.keepers[host] = k
	e.Websocket.Wg.Go(func() { e.wsKeepListenKeyAlive(ctx, k) })
	return k, nil
}

// wsCredentialsRefused reports whether BingX refused a request for its credentials. request.ErrAuthRequestFailed cannot
// tell, as the requester adds it to every failed authenticated request
func wsCredentialsRefused(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case 100001, 100412, 100413, 100419:
		return true
	default:
		return false
	}
}

// wsRecordListenKeyFailure makes a REST host's listen key requests wait after a failure, returning when the wait ends:
// the first retry delay, then twice the last wait each time, up to websocketListenKeyRetryMaxDelay
func (e *Exchange) wsRecordListenKeyFailure(host string) time.Time {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keyRetries == nil {
		s.keyRetries = make(map[string]websocketListenKeyRetry)
	}
	r := s.keyRetries[host]
	if r.wait == 0 {
		r.wait = cmp.Or(s.retryDelay, websocketListenKeyRetryDelay)
	} else {
		r.wait = min(r.wait*2, websocketListenKeyRetryMaxDelay)
	}
	r.next = time.Now().Add(r.wait)
	s.keyRetries[host] = r
	return r.next
}

// wsListenKeyWaiting reports whether listen key requests to the REST host at ePath wait after a failure
func (e *Exchange) wsListenKeyWaiting(ePath exchange.URL) bool {
	host, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return false
	}
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.keyRetries[host]
	return ok && time.Now().Before(r.next)
}

// wsListenKeyWaitEnd returns when the earliest wait of listen key requests still waiting ends, or zero when none waits
func (e *Exchange) wsListenKeyWaitEnd() time.Time {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	var earliest time.Time
	now := time.Now()
	for _, r := range s.keyRetries {
		if r.next.After(now) && (earliest.IsZero() || r.next.Before(earliest)) {
			earliest = r.next
		}
	}
	return earliest
}

// wsScheduleAccountReconnect flushes the subscriptions at a time, so the account streams left out while listen key
// requests wait connect again once the wait ends, without the other connections closing. A flush due no later absorbs
// the request, and a flush leaving account streams out again schedules the next. It runs apart from the manager's wait
// group, as a flush takes the lock a teardown holds while it waits for that group
func (e *Exchange) wsScheduleAccountReconnect(ctx context.Context, at time.Time) {
	s := &e.wsState
	s.mu.Lock()
	if !s.reconnectAt.IsZero() && !at.Before(s.reconnectAt) {
		s.mu.Unlock()
		return
	}
	s.reconnectAt = at
	s.reconnects++
	scheduled := s.reconnects
	s.background.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.background.Done()
		t := time.NewTimer(time.Until(at))
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
		}
		s.mu.Lock()
		if s.reconnects != scheduled {
			s.mu.Unlock()
			return
		}
		s.reconnectAt = time.Time{}
		s.mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		// A websocket not connected takes the account streams when the connection monitor connects it
		if err := e.Websocket.FlushChannels(ctx); err != nil && !errors.Is(err, websocket.ErrNotConnected) && !errors.Is(err, websocket.ErrWebsocketNotEnabled) {
			log.Errorf(log.WebsocketMgr, "%s websocket: unable to connect the account streams: %v", e.Name, err)
		}
		if next := e.wsListenKeyWaitEnd(); !next.IsZero() {
			e.wsScheduleAccountReconnect(ctx, next)
		}
	}()
}

// wsAddKeeperConnection records a connection dialled with a keeper's key, forgetting those no longer connected
func (e *Exchange) wsAddKeeperConnection(k *websocketListenKeyKeeper, conn websocket.Connection) {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	k.conns = append(slices.DeleteFunc(k.conns, func(c websocket.Connection) bool { return !wsIsConnected(c) }), conn)
}

// wsKeeperInUse reports whether any connection dialled with a keeper's key is still connected
func (e *Exchange) wsKeeperInUse(k *websocketListenKeyKeeper) bool {
	s := &e.wsState
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(k.conns, wsIsConnected)
}

// wsKeepListenKeyAlive extends a listen key each interval, with the credentials it was created for, until its session is
// torn down or none of the connections using it is connected. A failed extension is tried again, after longer waits,
// while the key remains valid; once no attempt fits before the key expires, the websocket reconnects and the account
// connections take a new key
func (e *Exchange) wsKeepListenKeyAlive(ctx context.Context, k *websocketListenKeyKeeper) {
	defer e.wsStopKeeper(k)
	ctx, cancel := context.WithCancel(accounts.DeployCredentialsToContext(ctx, k.credentials))
	defer cancel()
	// Ends an extension in progress when the session is torn down, as the teardown waits for the keeper
	go func() {
		select {
		case <-k.session:
			cancel()
		case <-ctx.Done():
		}
	}()
	// The key was created just before its keeper started
	extended := time.Now()
	var wait time.Duration
	t := time.NewTimer(k.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if !e.wsKeeperInUse(k) {
			return
		}
		err := e.extendListenKey(ctx, k.ePath, k.key)
		if err == nil {
			extended, wait = time.Now(), 0
			t.Reset(k.interval)
			continue
		}
		if ctx.Err() != nil {
			return
		}
		err = fmt.Errorf("%w: %s", errWebsocketListenKeyExtension, wsRedact(err.Error(), k.key))
		var retry bool
		if wait, retry = wsExtensionRetryWait(extended, time.Now(), wait, k.retryDelay, k.validity); retry {
			log.Warnf(log.WebsocketMgr, "%s websocket: trying again in %s: %v", e.Name, wait, err)
			t.Reset(wait)
			continue
		}
		e.wsRequestRecovery(k.session, err)
		return
	}
}

// wsExtensionRetryWait returns the wait before trying a failed extension again, firstWait after the first failure and
// twice the last wait after each later one, and whether the attempt comes at least firstWait before a key extended at
// extended expires
func wsExtensionRetryWait(extended, now time.Time, lastWait, firstWait, validity time.Duration) (time.Duration, bool) {
	wait := firstWait
	if lastWait > 0 {
		wait = lastWait * 2
	}
	return wait, now.Add(wait).Before(extended.Add(validity - firstWait))
}

// wsStopKeeper forgets a keeper that has stopped
func (e *Exchange) wsStopKeeper(k *websocketListenKeyKeeper) {
	s := &e.wsState
	s.mu.Lock()
	if s.keepers[k.host] == k {
		delete(s.keepers, k.host)
	}
	s.mu.Unlock()
	close(k.done)
}

// wsHandleControlFrame answers heartbeats and routes request replies to their requests, reporting whether the frame was
// either. Heartbeats are answered before anything else, on a rate limit of their own, so subscription requests never
// hold them up
func (e *Exchange) wsHandleControlFrame(ctx context.Context, conn websocket.Connection, respRaw []byte) (bool, error) {
	if bytes.Equal(respRaw, []byte(websocketSwapPing)) {
		return true, conn.SendRawMessage(ctx, websocketHeartbeatRate, gws.TextMessage, []byte(websocketSwapPong))
	}
	if _, err := jsonparser.GetString(respRaw, "ping"); err == nil {
		var ping WebsocketPing
		if err := json.Unmarshal(respRaw, &ping); err != nil {
			return true, err
		}
		pong, err := json.Marshal(&WebsocketPongRequest{Pong: ping.Ping, Time: ping.Time})
		if err != nil {
			return true, err
		}
		return true, conn.SendRawMessage(ctx, websocketHeartbeatRate, gws.TextMessage, pong)
	}
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		return true, conn.RequireMatchWithData(id, respRaw)
	}
	return false, nil
}

// wsMarketFrame splits a market data frame into the symbol and stream its dataType names and its data. A frame with a
// non-zero code is an error the server reports outside any request
func wsMarketFrame(respRaw []byte) (symbol, stream string, data []byte, err error) {
	if code, codeErr := jsonparser.GetInt(respRaw, "code"); codeErr == nil && code != 0 {
		msg, _ := jsonparser.GetString(respRaw, "msg")
		return "", "", nil, fmt.Errorf("%w: %w", errWebsocketErrorMessage, &APIError{Code: code, Message: msg})
	}
	dataType, err := jsonparser.GetString(respRaw, "dataType")
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
	}
	symbol, stream, ok := strings.Cut(dataType, "@")
	if !ok {
		return "", "", nil, fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
	}
	data, _, _, err = jsonparser.Get(respRaw, "data")
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
	}
	return symbol, stream, data, nil
}

// wsAccountEvent returns an account data frame's event type and payload, unwrapping a payload sent in a data member,
// as spot order events are. Its errors never quote the frame, which may carry the listen key
func wsAccountEvent(respRaw []byte) (event string, data []byte, err error) {
	if code, codeErr := jsonparser.GetInt(respRaw, "code"); codeErr == nil && code != 0 {
		msg, _ := jsonparser.GetString(respRaw, "msg")
		return "", nil, fmt.Errorf("%w: %w", errWebsocketErrorMessage, &APIError{Code: code, Message: msg})
	}
	data = respRaw
	if payload, dataType, _, dataErr := jsonparser.Get(respRaw, "data"); dataErr == nil && dataType == jsonparser.Object {
		data = payload
	}
	event, err = jsonparser.GetString(data, "e")
	if err != nil {
		return "", nil, fmt.Errorf("%w: account data frame without an event type", errWebsocketUnhandledMessage)
	}
	return event, data, nil
}

// wsRelay relays a payload decoded into its own type, for streams GCT has no common type for
func wsRelay[T any](ctx context.Context, e *Exchange, data []byte) error {
	var resp *T
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// wsProcessTicker stores a ticker and relays it once stored, so a ticker that fails processing is never relayed
func (e *Exchange) wsProcessTicker(ctx context.Context, p *ticker.Price) error {
	if err := ticker.ProcessTicker(p); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, p)
}

// wsTakerSide returns the side that took a trade: a buyer that was the maker means the seller took the trade
func wsTakerSide(isBuyerMaker bool) order.Side {
	if isBuyerMaker {
		return order.Sell
	}
	return order.Buy
}

// wsSplitSymbol splits a symbol at its last dash, as a few spot bases contain one
func wsSplitSymbol(symbol string) (base, quote string, err error) {
	i := strings.LastIndexByte(symbol, '-')
	if i <= 0 || i == len(symbol)-1 {
		return "", "", fmt.Errorf("%w: %q", errWebsocketInvalidSymbol, symbol)
	}
	return symbol[:i], symbol[i+1:], nil
}

// wsMatchPair returns the available pair of a market data symbol and whether it is enabled
func (e *Exchange) wsMatchPair(symbol string, a asset.Item) (pair currency.Pair, enabled bool, err error) {
	base, quote, err := wsSplitSymbol(symbol)
	if err != nil {
		return currency.EMPTYPAIR, false, err
	}
	return e.MatchSymbolCheckEnabled(base+quote, a, false)
}

// wsAccountPair returns the pair of an account event's symbol: the available pair, or the pair the symbol names for a
// symbol listed since the available pairs were last updated, as an account's orders and positions are relayed whatever
// pairs are enabled
func (e *Exchange) wsAccountPair(symbol string, a asset.Item) (currency.Pair, error) {
	base, quote, err := wsSplitSymbol(symbol)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	if pair, err := e.MatchSymbolWithAvailablePairs(base+quote, a, false); err == nil {
		return pair, nil
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return currency.NewPairWithDelimiter(base, quote, format.Delimiter).Format(format), nil
}

// wsProcessListenKeyExpired relays the expiry of an account connection's listen key and reconnects the websocket, as
// the account connections need a new key
func (e *Exchange) wsProcessListenKeyExpired(ctx context.Context, conn websocket.Connection, data []byte) error {
	var resp *WebsocketListenKeyExpired
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	e.wsRequestRecovery(e.wsSession(conn), errWebsocketListenKeyExpired)
	return e.Websocket.DataHandler.Send(ctx, resp)
}
