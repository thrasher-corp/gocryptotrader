package kucoin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// kucoinFuturesWebsocketURL is the futures connection's starting URL; every connect dials the endpoint its token
	// names
	kucoinFuturesWebsocketURL = "wss://ws-api-futures.kucoin.com/"

	// Spot public channels
	marketTickerChannel           = "/market/ticker" // /market/ticker:{symbol},... or /market/ticker:all
	marketOrderbookDepth1Channel  = "/spotMarket/level1"
	marketOrderbookDepth5Channel  = "/spotMarket/level2Depth5"
	marketOrderbookDepth50Channel = "/spotMarket/level2Depth50"
	marketOrderbookChannel        = "/market/level2"
	callAuctionOrderbookChannel   = "/callauction/level2Depth50"
	callAuctionDataChannel        = "/callauction/callauctionData"
	marketCandlesChannel          = "/market/candles" // /market/candles:{symbol}_{interval}
	marketMatchChannel            = "/market/match"
	marketSnapshotChannel         = "/market/snapshot" // /market/snapshot:{symbol} or /market/snapshot:{market}

	// Spot and margin private channels
	spotOrderV2Channel        = "/spotMarket/tradeOrdersV2"
	spotOrderV1Channel        = "/spotMarket/tradeOrders"
	accountBalanceChannel     = "/account/balance"
	spotMarketAdvancedChannel = "/spotMarket/advancedOrders"

	// Margin public channels
	indexPriceIndicatorChannel = "/indicator/index"
	markPriceIndicatorChannel  = "/indicator/markPrice"

	// Margin private channels
	marginPositionChannel         = "/margin/position"
	isolatedMarginPositionChannel = "/margin/isolatedPosition"

	// Futures public channels
	futuresTickerV2Channel         = "/contractMarket/tickerV2"
	futuresTickerV1Channel         = "/contractMarket/ticker"
	futuresOrderbookDepth5Channel  = "/contractMarket/level2Depth5"
	futuresOrderbookDepth50Channel = "/contractMarket/level2Depth50"
	futuresOrderbookChannel        = "/contractMarket/level2"
	futuresLimitCandleChannel      = "/contractMarket/limitCandle" // /contractMarket/limitCandle:{symbol}_{interval}
	futuresExecutionChannel        = "/contractMarket/execution"
	futuresInstrumentChannel       = "/contract/instrument"
	futuresAnnouncementChannel     = "/contract/announcement"
	futuresSnapshotChannel         = "/contractMarket/snapshot"

	// Futures private channels
	futuresTradeOrderChannel               = "/contractMarket/tradeOrders"
	futuresAccountBalanceEventChannel      = "/contractAccount/wallet"
	futuresPositionAllChannel              = "/contract/positionAll"
	futuresPositionChannel                 = "/contract/position"
	futuresMarginModeChannel               = "/contract/marginMode"
	futuresCrossLeverageChannel            = "/contract/crossLeverage"
	futuresStopOrdersLifecycleEventChannel = "/contractMarket/advancedOrders"

	spotConnection    = "spot"
	futuresConnection = "futures"

	// defaultPingDelay is half the documented ping interval, used when a server names none
	defaultPingDelay = 9 * time.Second
)

var (
	errNoInstanceServer          = errors.New("no websocket instance server")
	errWebsocketDialFailed       = errors.New("websocket dial failed")
	errWelcomeNotReceived        = errors.New("websocket welcome message not received")
	errUnhandledTopic            = errors.New("unhandled websocket topic")
	errUnhandledSubject          = errors.New("unhandled websocket subject")
	errUnhandledMessageType      = errors.New("unhandled websocket message type")
	errSubscriptionRejected      = errors.New("websocket subscription rejected")
	errUnsupportedCandleInterval = errors.New("candle interval not supported by the futures websocket")
	errOrderbookFedByUTA         = errors.New("orderbook depth channel left out for a pair a UTA orderbook feeds")
)

var intervalMapReverseLookup = func() map[string]kline.Interval {
	m := make(map[string]kline.Interval, len(intervalMap))
	for k, v := range intervalMap {
		m[v] = k
	}
	return m
}()

// pingDelay returns the delay between pings for a server's ping interval in milliseconds. The server only needs a ping
// every interval, so pinging twice as often allows for network delay; an interval of zero, or one too long for a
// time.Duration, leaves the default
func pingDelay(intervalMilliseconds uint64) time.Duration {
	if intervalMilliseconds == 0 || intervalMilliseconds > math.MaxInt64/uint64(time.Millisecond) {
		return defaultPingDelay
	}
	return time.Duration(intervalMilliseconds) * time.Millisecond / 2
}

// IntervalFromString returns a kline.Interval from string input.
func IntervalFromString(interval string) (kline.Interval, error) {
	klineInterval, ok := intervalMapReverseLookup[interval]
	if ok {
		return klineInterval, nil
	}
	return 0, fmt.Errorf("%w interval: %v", kline.ErrInvalidInterval, interval)
}

var subscriptionNames = map[asset.Item]map[string]string{
	asset.Futures: {
		subscription.TickerChannel:    futuresTickerV2Channel,
		subscription.OrderbookChannel: futuresOrderbookDepth5Channel, // This does not require a REST request to get the orderbook.
		subscription.CandlesChannel:   futuresLimitCandleChannel,
		subscription.AllTradesChannel: futuresExecutionChannel,
	},
	asset.All: {
		subscription.TickerChannel:    marketTickerChannel,
		subscription.OrderbookChannel: marketOrderbookDepth5Channel, // This does not require a REST request to get the orderbook.
		subscription.CandlesChannel:   marketCandlesChannel,
		subscription.AllTradesChannel: marketMatchChannel,
	},
}

// leaveExplicitFuturesTickers leaves each futures pair an explicit tickerV2 subscription feeds to that subscription. A
// generic futures ticker subscribes the same topic, and KuCoin keeps one subscription per topic, so removing either
// would silence the other; the generic ticker keeps its companion topics. Expansion gives each generic ticker a pair of
// its own, but a pre-expanded one can name several, so each of its symbols gives way alone
func (e *Exchange) leaveExplicitFuturesTickers(subs subscription.List) (subscription.List, error) {
	explicit := make(map[string]struct{})
	for _, s := range subs {
		if s.Channel != futuresTickerV2Channel {
			continue
		}
		_, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
		for symbol := range strings.SplitSeq(symbols, ",") {
			explicit[symbol] = struct{}{}
		}
	}
	if len(explicit) == 0 {
		return subs, nil
	}
	kept := subs[:0]
	for _, s := range subs {
		channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
		if s.Channel != subscription.TickerChannel || s.Asset != asset.Futures || channel != futuresTickerV2Channel {
			kept = append(kept, s)
			continue
		}
		var remaining []string
		var symbolCount int
		for symbol := range strings.SplitSeq(symbols, ",") {
			symbolCount++
			if _, fed := explicit[symbol]; !fed {
				remaining = append(remaining, symbol)
			}
		}
		switch {
		case len(remaining) == 0:
			continue
		case len(remaining) < symbolCount:
			format, err := e.GetPairFormat(asset.Futures, true)
			if err != nil {
				return nil, err
			}
			s = s.Clone()
			s.QualifiedChannel = channel + ":" + strings.Join(remaining, ",")
			s.Pairs = slices.DeleteFunc(s.Pairs, func(pair currency.Pair) bool {
				return !slices.Contains(remaining, pair.Format(format).String())
			})
		}
		kept = append(kept, s)
	}
	return kept, nil
}

// futuresTickerCompanions are the topics a futures ticker subscription also carries: tickerV2 pushes only the best bid
// and ask, Ticker V1 pushes the last fill and the symbol snapshot pushes the 24 hour statistics
var futuresTickerCompanions = []string{futuresTickerV1Channel, futuresSnapshotChannel}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Margin, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Futures, Channel: futuresTradeOrderChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: futuresStopOrdersLifecycleEventChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: futuresAccountBalanceEventChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Margin, Channel: marginPositionChannel, Authenticated: true},
	{Enabled: true, Channel: accountBalanceChannel, Authenticated: true},
}

// GetPublicToken returns a token and the servers for a public spot and margin websocket connection (Get Public Token -
// Classic Spot/Margin)
func (e *Exchange) GetPublicToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	return e.getPublicToken(ctx, exchange.RestSpot, getPublicTokenEPL)
}

// GetPrivateToken returns a token and the servers for a private spot and margin websocket connection (Get Private Token
// - Classic Spot/Margin)
func (e *Exchange) GetPrivateToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	var resp *WebsocketTokenResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getPrivateTokenEPL, http.MethodPost, "/v1/bullet-private", nil, &resp)
}

// GetFuturesPublicToken returns a token and the servers for a public futures websocket connection (Get Public Token -
// Classic Futures)
func (e *Exchange) GetFuturesPublicToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	return e.getPublicToken(ctx, exchange.RestFutures, getFuturesPublicTokenEPL)
}

// GetFuturesPrivateToken returns a token and the servers for a private futures websocket connection (Get Private Token -
// Classic Futures)
func (e *Exchange) GetFuturesPrivateToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	var resp *WebsocketTokenResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesPrivateTokenEPL, http.MethodPost, "/v1/bullet-private", nil, &resp)
}

// getPublicToken posts the unsigned token request, which SendHTTPRequest cannot send as it only sends GET requests
func (e *Exchange) getPublicToken(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit) (*WebsocketTokenResponse, error) {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return nil, err
	}
	var resp *WebsocketTokenResponse
	result := &Response{Data: &resp}
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpointPath + "/v1/bullet-public",
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, result.GetError()
}

// newSpotConnectionSetup returns the setup of the Classic spot and margin websocket connection
func (e *Exchange) newSpotConnectionSetup(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		RateLimit:             request.NewRateLimitWithWeight(time.Second*10, 100, 1), // See: https://www.kucoin.com/docs-new/rate-limit#websocket-rate-limit
		URL:                   wsURL,
		Connector:             e.WebsocketConnectSpot,
		GenerateSubscriptions: e.generateSpotSubscriptions,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		MessageFilter:         spotConnection,
	}
}

// newFuturesConnectionSetup returns the setup of the Classic futures websocket connection, which has its own host and
// tokens. It is optional, so a futures outage leaves the spot connection up while the manager retries it
func (e *Exchange) newFuturesConnectionSetup(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		RateLimit:             request.NewRateLimitWithWeight(time.Second*10, 100, 1), // See: https://www.kucoin.com/docs-new/rate-limit#websocket-rate-limit
		URL:                   wsURL,
		Connector:             e.WebsocketConnectFutures,
		GenerateSubscriptions: e.generateFuturesSubscriptions,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		OnDisconnect:          e.invalidateOrderbooks,
		MessageFilter:         futuresConnection,
		Optional:              true,
	}
}

// orderbookPushKind is what storing an orderbook push without an error says of it
type orderbookPushKind uint8

const (
	// orderbookSnapshot is stored
	orderbookSnapshot orderbookPushKind = iota
	// orderbookDelta is stored, or skipped as one the book holds
	orderbookDelta
	// orderbookIncrement is stored, or queued by the update manager for its resync
	orderbookIncrement
)

// feedOrderbook stores an orderbook push a connection sent through store, and records it for the connection's
// disconnect hook, holding the lock the hook invalidates under throughout, so neither another connection's push nor the
// hook comes between the two. A push stored, which leaves the book holding its update ID where it held another or was
// unavailable, or a snapshot stored with an update ID the book held already, holds the connection's data, so the book is
// no other connection's from then. A rejected push, or a skipped one the book already held, changes nothing. An
// increment the update manager queued makes the book the connection's too, as its resync loads the book for the
// connection, while the connection whose data the book holds keeps it
func (e *Exchange) feedOrderbook(conn websocket.Connection, pair currency.Pair, a asset.Item, updateID int64, kind orderbookPushKind, store func() error) error {
	e.orderbookFeedsMu.Lock()
	defer e.orderbookFeedsMu.Unlock()
	before, beforeErr := e.Websocket.Orderbook.LastUpdateID(pair, a)
	err := store()
	if conn == nil {
		return err
	}
	after, afterErr := e.Websocket.Orderbook.LastUpdateID(pair, a)
	// A snapshot is stored unless rejected, before it is relayed, so one whose relay alone failed was stored
	stored := kind == orderbookSnapshot && (err == nil || errors.Is(err, stream.ErrChannelBufferFull))
	held := afterErr == nil && after == updateID && (beforeErr != nil || before != after || stored)
	if !held && (kind != orderbookIncrement || err != nil) {
		return err
	}
	k := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}
	if held {
		for other, fed := range e.orderbookFeeds {
			if other != conn {
				delete(fed, k)
			}
		}
	}
	if e.orderbookFeeds == nil {
		e.orderbookFeeds = make(map[websocket.Connection]map[key.PairAsset]struct{})
	}
	if e.orderbookFeeds[conn] == nil {
		e.orderbookFeeds[conn] = make(map[key.PairAsset]struct{})
	}
	e.orderbookFeeds[conn][k] = struct{}{}
	return err
}

// invalidateOrderbooks invalidates the orderbooks recorded for a connection once its reader stops. The futures and UTA
// public connections are optional, so the manager stays connected while one is down and the engine goes on waiting for
// those books on the websocket rather than reading them over REST; invalidated, they read as unavailable rather than
// as current until a connection feeds them again. A book another connection also feeds reads as unavailable until
// that connection's next push is stored, and a level 2 resync already under way can still load its REST snapshot
// afterwards, which is dated as of its request
func (e *Exchange) invalidateOrderbooks(_ context.Context, conn websocket.Connection) {
	e.orderbookFeedsMu.Lock()
	defer e.orderbookFeedsMu.Unlock()
	for k := range e.orderbookFeeds[conn] {
		// A book the store no longer holds has nothing to invalidate, which is the only error
		_ = e.Websocket.Orderbook.InvalidateOrderbook(k.Pair(), k.Asset)
	}
	delete(e.orderbookFeeds, conn)
}

// WebsocketConnectSpot connects to the Classic spot and margin websocket, with a private token when authenticated
func (e *Exchange) WebsocketConnectSpot(ctx context.Context, conn websocket.Connection) error {
	getToken := e.GetPublicToken
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		getToken = e.GetPrivateToken
	}
	token, err := getToken(ctx)
	if err != nil {
		return err
	}
	return e.connectWebsocket(ctx, conn, token, e.websocketDialer())
}

// WebsocketConnectFutures connects to the Classic futures websocket, with a private token when authenticated
func (e *Exchange) WebsocketConnectFutures(ctx context.Context, conn websocket.Connection) error {
	getToken := e.GetFuturesPublicToken
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		getToken = e.GetFuturesPrivateToken
	}
	token, err := getToken(ctx)
	if err != nil {
		return err
	}
	return e.connectWebsocket(ctx, conn, token, e.websocketDialer())
}

func (e *Exchange) websocketDialer() *gws.Dialer {
	return &gws.Dialer{HandshakeTimeout: e.Config.HTTPTimeout, Proxy: http.ProxyFromEnvironment}
}

// connectWebsocket dials the server a token names and waits for its welcome, as the connection only takes requests
// once the welcome arrives
func (e *Exchange) connectWebsocket(ctx context.Context, conn websocket.Connection, token *WebsocketTokenResponse, dialer *gws.Dialer) error {
	if err := common.NilGuard(token); err != nil {
		return err
	}
	if len(token.InstanceServers) == 0 {
		return errNoInstanceServer
	}
	server := token.InstanceServers[0]
	conn.SetURL(server.Endpoint)
	connectID := e.MessageID()
	values := url.Values{}
	values.Set("token", token.Token)
	values.Set("connectId", connectID)
	if err := conn.Dial(ctx, dialer, nil, values); err != nil {
		return fmt.Errorf("%w: %w", errWebsocketDialFailed, err)
	}
	welcome, err := readWebsocketMessage(ctx, conn, e.WebsocketResponseMaxLimit)
	if err == nil {
		err = checkWelcome(welcome, connectID)
	}
	if err != nil {
		return err
	}
	// The connect ID is a UUID, which needs no JSON escaping
	ping := []byte(`{"id":"` + connectID + `","type":"ping"}`)
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       pingDelay(server.PingInterval),
		Message:     ping,
		MessageType: gws.TextMessage,
	})
	return nil
}

// readWebsocketMessage reads one message before the connection's reader starts, giving up after the timeout
func readWebsocketMessage(ctx context.Context, conn websocket.Connection, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = exchange.DefaultWebsocketResponseMaxLimit
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	received := make(chan websocket.Response, 1)
	go func() { received <- conn.ReadMessage() }()
	select {
	case resp := <-received:
		if resp.Raw == nil {
			return nil, fmt.Errorf("%w: connection closed", errWelcomeNotReceived)
		}
		return resp.Raw, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %w", errWelcomeNotReceived, ctx.Err())
	}
}

// checkWelcome returns an error unless the message is the welcome for the connect ID
func checkWelcome(data []byte, connectID string) error {
	var msg WebsocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if msg.Type != "welcome" || msg.ID != connectID {
		return fmt.Errorf("%w: %s", errWelcomeNotReceived, data)
	}
	return nil
}

// wsHandleData routes a message from a Classic websocket connection
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respData []byte) error {
	var msg WebsocketMessage
	if err := json.Unmarshal(respData, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case "ack", "error":
		return conn.RequireMatchWithData(msg.ID, respData)
	case "welcome", "pong":
		return nil
	case "message", "":
		// Some private pushes omit the type, so a frame without one is a push when it names a topic
		if msg.Topic != "" {
			return e.processPush(ctx, conn, &msg)
		}
	}
	return fmt.Errorf("%w %q: %s", errUnhandledMessageType, msg.Type, respData)
}

// processPush sends a topic push to the handler for its channel
func (e *Exchange) processPush(ctx context.Context, conn websocket.Connection, msg *WebsocketMessage) error {
	channel, symbols, _ := strings.Cut(msg.Topic, ":")
	switch channel {
	case marketTickerChannel:
		if symbols == "all" {
			// All Tickers pushes name the symbol in the subject
			symbols = msg.Subject
		}
		return e.processTicker(ctx, msg.Data, symbols)
	case marketOrderbookDepth1Channel:
		return e.processOrderbookLevel1(ctx, msg.Data, symbols)
	case marketOrderbookDepth5Channel, marketOrderbookDepth50Channel:
		return e.processOrderbook(ctx, msg.Data, symbols, channel)
	case marketOrderbookChannel:
		return e.processSpotOrderbookIncrement(ctx, msg.Data, symbols)
	case callAuctionOrderbookChannel:
		return e.processCallAuctionOrderbook(ctx, msg.Data, symbols)
	case callAuctionDataChannel:
		return e.processCallAuctionData(ctx, msg.Data)
	case marketCandlesChannel:
		return e.processCandlesticks(ctx, msg.Data, symbols)
	case marketMatchChannel:
		return e.processTradeData(msg.Data, symbols)
	case marketSnapshotChannel:
		return e.processMarketSnapshot(ctx, msg.Data)
	case spotOrderV2Channel, spotOrderV1Channel:
		return e.processOrderChange(ctx, msg.Data)
	case accountBalanceChannel:
		return e.processAccountBalance(ctx, msg.Data)
	case spotMarketAdvancedChannel:
		return e.processStopOrder(ctx, msg.Data)
	case indexPriceIndicatorChannel:
		return processPayload[WebsocketMarginIndexPrice](ctx, e, msg.Data)
	case markPriceIndicatorChannel:
		return processPayload[WebsocketMarginMarkPrice](ctx, e, msg.Data)
	case marginPositionChannel:
		return e.processCrossMarginPosition(ctx, msg)
	case isolatedMarginPositionChannel:
		return e.processIsolatedMarginPosition(ctx, msg)
	case futuresTickerV2Channel:
		return e.processFuturesTickerV2(ctx, msg.Data)
	case futuresTickerV1Channel:
		return e.processFuturesTickerV1(ctx, msg.Data)
	case futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
		return e.processFuturesOrderbookSnapshot(ctx, conn, msg.Data, symbols)
	case futuresOrderbookChannel:
		return e.processFuturesOrderbookIncrement(ctx, conn, msg.Data, symbols)
	case futuresLimitCandleChannel:
		return e.processFuturesKline(ctx, msg.Data, symbols)
	case futuresExecutionChannel:
		return e.processFuturesExecution(msg.Data)
	case futuresInstrumentChannel:
		return e.processFuturesInstrument(ctx, msg, symbols)
	case futuresAnnouncementChannel:
		return e.processFuturesAnnouncement(ctx, msg)
	case futuresSnapshotChannel:
		return e.processFuturesSnapshot(ctx, msg.Data)
	case futuresTradeOrderChannel:
		return e.processFuturesOrderChange(ctx, msg.Data)
	case futuresAccountBalanceEventChannel:
		return e.processFuturesWallet(ctx, msg)
	case futuresPositionAllChannel, futuresPositionChannel:
		return e.processFuturesPosition(ctx, msg)
	case futuresMarginModeChannel:
		return processPayload[WebsocketFuturesMarginMode](ctx, e, msg.Data)
	case futuresCrossLeverageChannel:
		return processPayload[WebsocketFuturesCrossLeverage](ctx, e, msg.Data)
	case futuresStopOrdersLifecycleEventChannel:
		return e.processFuturesStopOrder(ctx, msg.Data)
	}
	return fmt.Errorf("%w: %q", errUnhandledTopic, msg.Topic)
}

// processPayload decodes a push into its type and relays it
func processPayload[T any](ctx context.Context, e *Exchange, data []byte) error {
	resp := new(T)
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// unhandledSubject returns the error for a push whose subject its topic does not document
func unhandledSubject(msg *WebsocketMessage) error {
	return fmt.Errorf("%w %q on topic %q", errUnhandledSubject, msg.Subject, msg.Topic)
}

// mergeTicker applies a push to a copy of the stored ticker of a pair, or to a new ticker when none is stored, so a push
// that reports only some fields keeps the others the store already holds
func (e *Exchange) mergeTicker(pair currency.Pair, a asset.Item, apply func(*ticker.Price)) (*ticker.Price, error) {
	t, err := ticker.GetTicker(e.Name, pair, a)
	if err != nil {
		if !errors.Is(err, ticker.ErrTickerNotFound) {
			return nil, err
		}
		t = &ticker.Price{ExchangeName: e.Name, Pair: pair, AssetType: a}
	}
	apply(t)
	return t, nil
}

// processMergedTicker merges a push onto the stored ticker of a pair, stores it and relays it
func (e *Exchange) processMergedTicker(ctx context.Context, pair currency.Pair, a asset.Item, apply func(*ticker.Price)) error {
	e.tickerMergeMu.Lock()
	t, err := e.mergeTicker(pair, a, apply)
	if err == nil {
		err = ticker.ProcessTicker(t)
	}
	e.tickerMergeMu.Unlock()
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, t)
}

// futuresPair returns the futures pair of a symbol
func (e *Exchange) futuresPair(symbol string) (currency.Pair, error) {
	return e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, false)
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.manageSubscriptions(ctx, conn, subscriptions, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.manageSubscriptions(ctx, conn, subscriptions, "unsubscribe")
}

func (e *Exchange) manageSubscriptions(ctx context.Context, conn websocket.Connection, subs subscription.List, operation string) error {
	outbound := collapseSubscriptionList(subs)

	var errs error
	for assoc, s := range outbound {
		req := &WebsocketSubscriptionRequest{
			ID:             e.MessageID(),
			Type:           operation,
			Topic:          s.QualifiedChannel,
			PrivateChannel: s.Authenticated,
			Response:       true,
		}
		respRaw, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
		if err != nil {
			return err
		}

		var resp *WebsocketSubscriptionResponse
		if err := json.Unmarshal(respRaw, &resp); err != nil {
			return err
		}
		if resp == nil {
			return fmt.Errorf("%w: %s %s", common.ErrNoResponse, operation, s.QualifiedChannel)
		}

		switch resp.Type {
		case "error":
			var msg string
			if err := json.Unmarshal(resp.Data, &msg); err != nil {
				msg = "unknown error"
			}
			errs = common.AppendError(errs, fmt.Errorf("%w: %s %s (%d)", errSubscriptionRejected, s.QualifiedChannel, msg, resp.Code))
		case "ack":
			if operation == "unsubscribe" {
				err = e.Websocket.RemoveSubscriptions(conn, *assoc...)
			} else {
				err = e.Websocket.AddSuccessfulSubscriptions(conn, *assoc...)
				if e.Verbose {
					log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s", e.Name, s.Channel)
				}
			}
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		default:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s from %s", errInvalidMsgType, resp.Type, respRaw))
		}
	}
	return errs
}

// collapseSubscriptionList merges per-pair subscriptions into KuCoin-compatible
// channel batches, capped at 100 entries per request. The returned map preserves
// the original subscriptions in each batch so subscribe and unsubscribe handling
// can still update connection state for the individual inputs.
func collapseSubscriptionList(subs subscription.List) map[*subscription.List]*subscription.Subscription {
	m := make(map[string][]*subscription.Subscription)
	for _, s := range subs {
		parts := strings.Split(s.QualifiedChannel, ":")
		m[parts[0]] = append(m[parts[0]], s)
	}

	result := make(map[*subscription.List]*subscription.Subscription)
	for qChan, group := range m {
		for _, batch := range common.Batch(group, 100) {
			s := batch[0].Clone()
			s.Pairs = nil
			suffixes := make([]string, 0, len(batch))
			for _, sub := range batch {
				s.Pairs = append(s.Pairs, sub.Pairs...)
				parts := strings.SplitN(sub.QualifiedChannel, ":", 2)
				if len(parts) == 2 && parts[1] != "" {
					suffixes = append(suffixes, parts[1])
				}
			}
			s.QualifiedChannel = qChan
			if len(suffixes) != 0 {
				s.QualifiedChannel += ":" + strings.Join(suffixes, ",")
			}
			assoc := subscription.List(batch)
			result[&assoc] = s
		}
	}
	return result
}

// generateSpotSubscriptions returns the subscriptions the spot and margin connection carries
func (e *Exchange) generateSpotSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	return e.leaveUTAOrderbookPairs(slices.DeleteFunc(subs, isFuturesSubscription), utaSpotPublicConnection)
}

// generateFuturesSubscriptions returns the subscriptions the futures connection carries. The futures websocket pushes no
// six hour candles, so a configuration asking for them loses only those subscriptions, reported as a partial failure
func (e *Exchange) generateFuturesSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	var errs error
	subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool {
		if !isFuturesSubscription(s) {
			return true
		}
		if s.Interval == kline.SixHour && channelName(s, s.Asset) == futuresLimitCandleChannel {
			errs = common.AppendError(errs, fmt.Errorf("%w: %w: %s", websocket.ErrSubscriptionPartial, errUnsupportedCandleInterval, s))
			return true
		}
		return false
	})
	subs, err = e.leaveUTAOrderbookPairs(subs, utaFuturesPublicConnection)
	return subs, common.AppendError(errs, err)
}

// leaveUTAOrderbookPairs leaves the pairs a UTA orderbook on utaConnection feeds out of the explicit Classic depth
// channel subscriptions of the same market. Both would write one book, and each depth channel push, a snapshot, would
// reset the UTA orderbook's sequence; the UTA orderbook carries the book whole, so the depth channel gives way, reported
// as a partial failure, as the configuration asked for both
func (e *Exchange) leaveUTAOrderbookPairs(subs subscription.List, utaConnection string) (subscription.List, error) {
	// UTA subscriptions that cannot be generated feed no books, and their own connection reports the error
	utaSubs, _ := e.generateUTASubscriptions(utaConnection)
	fed := make(map[string]struct{})
	for _, s := range utaSubs {
		if s.Channel == UTAOrderbookChannel {
			_, symbol, _ := strings.Cut(s.QualifiedChannel, ":")
			fed[symbol] = struct{}{}
		}
	}
	if len(fed) == 0 {
		return subs, nil
	}
	var errs error
	kept := subs[:0]
	for _, s := range subs {
		switch s.Channel {
		case marketOrderbookDepth1Channel, marketOrderbookDepth5Channel, marketOrderbookDepth50Channel, futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
		default:
			kept = append(kept, s)
			continue
		}
		channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
		var remaining, given []string
		for symbol := range strings.SplitSeq(symbols, ",") {
			if _, ok := fed[symbol]; ok {
				given = append(given, symbol)
				continue
			}
			remaining = append(remaining, symbol)
		}
		if len(given) == 0 {
			kept = append(kept, s)
			continue
		}
		errs = common.AppendError(errs, fmt.Errorf("%w: %w: %s %s", websocket.ErrSubscriptionPartial, errOrderbookFedByUTA, channel, strings.Join(given, ",")))
		if len(remaining) == 0 {
			continue
		}
		s = s.Clone()
		s.QualifiedChannel = channel + ":" + strings.Join(remaining, ",")
		if len(s.Pairs) != 0 {
			formatAsset := s.Asset
			if formatAsset == asset.All || formatAsset == asset.Empty {
				// A pre-expanded subscription can name no single asset, so its channel's market gives the format
				formatAsset = asset.Spot
				if strings.HasPrefix(channel, "/contract") {
					formatAsset = asset.Futures
				}
			}
			format, err := e.GetPairFormat(formatAsset, true)
			if err != nil {
				return nil, err
			}
			s.Pairs = slices.DeleteFunc(s.Pairs, func(pair currency.Pair) bool {
				return !slices.Contains(remaining, pair.Format(format).String())
			})
		}
		kept = append(kept, s)
	}
	return kept, errs
}

// isFuturesSubscription returns whether a subscription belongs on the futures connection
func isFuturesSubscription(s *subscription.Subscription) bool {
	switch s.Asset {
	case asset.Futures:
		return true
	case asset.Empty:
		// Assetless channels are account channels, and the futures ones are all named under /contract
		return strings.HasPrefix(channelName(s, s.Asset), "/contract")
	default:
		return false
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := slices.DeleteFunc(slices.Clone(e.Features.Subscriptions), isUTASubscription).ExpandTemplates(e)
	if err != nil {
		return subs, err
	}
	if subs, err = e.leaveExplicitFuturesTickers(addFuturesTickerCompanions(subs)); err != nil {
		return nil, err
	}
	// Depth channels reload the whole book on every push, which would clobber a realtime book for the same pair, so
	// the realtime upgrade leaves pairs an explicit depth channel already feeds to that channel. Symbols are matched as
	// the depth topic spells them, so a topic KuCoin will not serve, such as a config-format pair or a candle suffix,
	// keeps its realtime feed.
	depthFed := make(map[bool]map[string]struct{}, 2)
	depthTopics := make(map[string]struct{})
	for _, s := range subs {
		switch s.Channel {
		case marketOrderbookDepth1Channel, marketOrderbookDepth5Channel, marketOrderbookDepth50Channel, futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
			isFutures := s.Channel == futuresOrderbookDepth5Channel || s.Channel == futuresOrderbookDepth50Channel
			if depthFed[isFutures] == nil {
				depthFed[isFutures] = make(map[string]struct{})
			}
			channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
			for symbol := range strings.SplitSeq(symbols, ",") {
				depthFed[isFutures][symbol] = struct{}{}
				depthTopics[channel+":"+symbol] = struct{}{}
			}
		}
	}
	// A UTA orderbook writes the same book as the Classic channels for its pair and asset, so the generic orderbook
	// leaves its pairs to it too. UTA subscriptions that cannot be generated feed no books, and their own connection
	// reports the error; one left out as a partial failure leaves the others generated
	for _, connection := range []string{utaSpotPublicConnection, utaFuturesPublicConnection} {
		utaSubs, _ := e.generateUTASubscriptions(connection)
		for _, s := range utaSubs {
			if s.Channel != UTAOrderbookChannel {
				continue
			}
			isFutures := s.Asset == asset.Futures
			if depthFed[isFutures] == nil {
				depthFed[isFutures] = make(map[string]struct{})
			}
			_, symbol, _ := strings.Cut(s.QualifiedChannel, ":")
			depthFed[isFutures][symbol] = struct{}{}
		}
	}
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		// KuCoin keeps one subscription per topic, so a generic orderbook on an explicit depth channel's topic would
		// silence it when the realtime upgrade unsubscribes the generic; the explicit channel carries that topic alone.
		// A generic orderbook on another depth of a pair another channel feeds would load the same book, so it gives
		// way to that channel too. Expansion gives each pair a topic of its own, but a pre-expanded orderbook can name
		// several, so each of its symbols gives way alone
		kept := subs[:0]
		for _, s := range subs {
			if s.Channel != subscription.OrderbookChannel {
				kept = append(kept, s)
				continue
			}
			channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
			var remaining []string
			var symbolCount int
			for symbol := range strings.SplitSeq(symbols, ",") {
				symbolCount++
				_, shared := depthTopics[channel+":"+symbol]
				_, fed := depthFed[s.Asset == asset.Futures][symbol]
				if !shared && !fed {
					remaining = append(remaining, symbol)
				}
			}
			switch {
			case len(remaining) == 0:
				continue
			case len(remaining) < symbolCount:
				s = s.Clone()
				s.QualifiedChannel = channel + ":" + strings.Join(remaining, ",")
				if len(s.Pairs) != 0 {
					formatAsset := s.Asset
					if formatAsset == asset.All || formatAsset == asset.Empty {
						formatAsset = asset.Spot
					}
					format, err := e.GetPairFormat(formatAsset, true)
					if err != nil {
						return nil, err
					}
					s.Pairs = slices.DeleteFunc(s.Pairs, func(pair currency.Pair) bool {
						return !slices.Contains(remaining, pair.Format(format).String())
					})
				}
			}
			kept = append(kept, s)
		}
		return kept, nil
	}
	// Resolve authenticated orderbooks after expansion so the realtime feed is reflected in
	// subscription reconciliation keys and does not retain the public depth feed interval.
	configured := subscription.NewStore()
	seen := subscription.NewStore()
	resolved := make(subscription.List, 0, len(subs))
	for _, s := range subs {
		if s.Channel != subscription.OrderbookChannel {
			resolved = append(resolved, s)
			continue
		}
		if err := configured.Add(s.Clone()); err != nil {
			return nil, err
		}
		s = s.Clone()
		channel := marketOrderbookChannel
		formatAsset := s.Asset
		if s.Asset == asset.Futures {
			channel = futuresOrderbookChannel
		} else if formatAsset == asset.All || formatAsset == asset.Empty {
			formatAsset = asset.Spot
		}
		format, err := e.GetPairFormat(formatAsset, true)
		if err != nil {
			return nil, err
		}
		if fed := depthFed[s.Asset == asset.Futures]; len(fed) != 0 && len(s.Pairs) != 0 {
			s.Pairs = slices.DeleteFunc(s.Pairs, func(pair currency.Pair) bool {
				_, ok := fed[pair.Format(format).String()]
				return ok
			})
			if len(s.Pairs) == 0 {
				continue
			}
		}
		s.Channel = channel
		s.QualifiedChannel = channel + ":" + s.Pairs.Format(format).Join()
		s.Interval = 0
		s.Levels = 0
		// A fresh store given a subscription can only report a duplicate, whose feed is already resolved
		if err := seen.Add(s); err != nil {
			continue
		}
		resolved = append(resolved, s)
	}
	return resolved, nil
}

// addFuturesTickerCompanions adds the companion topics of each futures ticker subscription, leaving out any topic
// another subscription already carries because KuCoin keeps one subscription per topic
func addFuturesTickerCompanions(subs subscription.List) subscription.List {
	topics := make(map[string]struct{}, len(subs))
	for _, s := range subs {
		topics[s.QualifiedChannel] = struct{}{}
	}
	out := slices.Clone(subs)
	for _, s := range subs {
		if s.Channel != subscription.TickerChannel || s.Asset != asset.Futures {
			continue
		}
		_, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
		for _, channel := range futuresTickerCompanions {
			topic := channel + ":" + symbols
			if _, ok := topics[topic]; ok {
				continue
			}
			topics[topic] = struct{}{}
			c := s.Clone()
			c.Channel = channel
			c.QualifiedChannel = topic
			out = append(out, c)
		}
	}
	return out
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").
		Funcs(template.FuncMap{
			"channelName":             channelName,
			"formatSubscriptionPairs": e.formatSubscriptionPairs,
			"mergeMarginPairs":        e.mergeMarginPairs,
			"isSymbolChannel":         isSymbolChannel,
			"channelInterval":         channelInterval,
			"joinPairsWithInterval":   joinPairsWithInterval,
			"batch":                   common.Batch[currency.Pairs],
		}).
		Parse(subTplText)
}

// CalculateAssets returns the available asset types for a currency pair
func (e *Exchange) CalculateAssets(topic string, cp currency.Pair) ([]asset.Item, error) {
	switch {
	case cp.Quote.Equal(currency.USDTM), strings.HasPrefix(topic, "/contract"):
		if err := e.CurrencyPairs.IsAssetEnabled(asset.Futures); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Futures}, nil
	case strings.HasPrefix(topic, "/margin"), strings.HasPrefix(topic, "/index"):
		if err := e.CurrencyPairs.IsAssetEnabled(asset.Margin); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Margin}, nil
	default:
		resp := make([]asset.Item, 0, 2)
		spotEnabled, err := e.IsPairEnabled(cp, asset.Spot)
		if err != nil && !errors.Is(err, currency.ErrCurrencyNotFound) {
			return nil, err
		}
		if spotEnabled {
			resp = append(resp, asset.Spot)
		}
		marginEnabled, err := e.IsPairEnabled(cp, asset.Margin)
		if err != nil && !errors.Is(err, currency.ErrCurrencyNotFound) {
			return nil, err
		}
		if marginEnabled {
			resp = append(resp, asset.Margin)
		}
		return resp, nil
	}
}

// checkSubscriptions looks for any backwards incompatibilities with missing assets
// This should be unnecessary and removable by 2025
func (e *Exchange) checkSubscriptions() error {
	upgraded := false
	for _, s := range e.Config.Features.Subscriptions {
		if s.Asset != asset.Empty || isUTASubscription(s) {
			continue
		}
		upgraded = true
		s.Channel = strings.TrimSuffix(s.Channel, ":%s")
		switch s.Channel {
		case subscription.TickerChannel, subscription.OrderbookChannel:
			s.Asset = asset.All
		case subscription.AllTradesChannel:
			for _, d := range defaultSubscriptions {
				if d.Channel == s.Channel {
					e.Config.Features.Subscriptions = append(e.Config.Features.Subscriptions, d)
				}
			}
		case futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel, futuresAccountBalanceEventChannel:
			s.Asset = asset.Futures
		case marginPositionChannel:
			s.Asset = asset.Margin
		}
	}
	before := len(e.Config.Features.Subscriptions)
	var replacements subscription.List
	e.Config.Features.Subscriptions = slices.DeleteFunc(e.Config.Features.Subscriptions, func(s *subscription.Subscription) bool {
		switch s.Channel {
		case marketOrderbookChannel, futuresOrderbookChannel:
			if s.Enabled {
				assets := asset.Items{s.Asset}
				if s.Asset == asset.All || s.Asset == asset.Empty {
					assets = e.GetAssetTypes(true)
				}
				for _, assetType := range assets {
					replacements = append(replacements, &subscription.Subscription{
						Enabled: true, Channel: subscription.OrderbookChannel, Asset: assetType,
						Pairs: slices.Clone(s.Pairs), Interval: kline.HundredMilliseconds,
					})
				}
			}
			return true
		case futuresOrderbookDepth50Channel, futuresTickerV2Channel:
			// The legacy defaults carried these without an asset, and the generic orderbook and ticker for asset.All
			// replaced them; a futures entry subscribes to the channel itself
			return s.Asset != asset.Futures
		case "/margin/fundingBook", // Deprecated and removed
			"/margin/loan": // Retired with margin lending V1, whose lend orders it reported
			return true
		case subscription.AllTradesChannel:
			return s.Asset == asset.Empty
		}
		return false
	})
	upgraded = upgraded || before != len(e.Config.Features.Subscriptions)
	for _, replacement := range replacements {
		var coveredPairs currency.Pairs
		covered := false
		for i, sub := range e.Config.Features.Subscriptions {
			if !sub.Enabled || sub.Channel != subscription.OrderbookChannel || (sub.Asset != asset.All && sub.Asset != replacement.Asset) {
				continue
			}
			// A generic with a candle interval keeps the suffix on its public topics, where KuCoin sends nothing. It only
			// covers the legacy feed under websocket authentication, which can change after this migration is saved.
			if _, err := IntervalToString(sub.Interval); err == nil {
				continue
			}
			if sub.Authenticated && !e.API.AuthenticatedWebsocketSupport {
				if len(sub.Pairs) == 0 && sub.Asset == replacement.Asset {
					sub = sub.Clone()
					sub.Authenticated = false
					e.Config.Features.Subscriptions[i] = sub
					covered = true
					break
				}
				continue
			}
			if len(sub.Pairs) == 0 {
				covered = true
				break
			}
			coveredPairs = coveredPairs.Add(sub.Pairs...)
		}
		if covered {
			continue
		}
		if len(replacement.Pairs) != 0 && len(coveredPairs) != 0 {
			format, err := e.GetPairFormat(replacement.Asset, true)
			if err != nil {
				return fmt.Errorf("cannot migrate %s orderbook pair coverage: %w", replacement.Asset, err)
			}
			coveredSymbols := make(map[string]struct{}, len(coveredPairs))
			for _, pair := range coveredPairs.Format(format) {
				coveredSymbols[pair.String()] = struct{}{}
			}
			replacement.Pairs = slices.DeleteFunc(replacement.Pairs, func(pair currency.Pair) bool {
				_, ok := coveredSymbols[pair.Format(format).String()]
				return ok
			})
			if len(replacement.Pairs) == 0 {
				continue
			}
		}
		e.Config.Features.Subscriptions = append(e.Config.Features.Subscriptions, replacement)
	}
	if upgraded {
		e.Features.Subscriptions = e.Config.Features.Subscriptions.Enabled()
	}
	return nil
}

// formatSubscriptionPairs spells every asset's pairs in its request format, as a topic must name its symbols the way
// KuCoin spells them even when a subscription lists them in the config format; an orderbook also gives way on the
// futures pairs a more specific orderbook subscription claims
func (e *Exchange) formatSubscriptionPairs(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) (string, error) {
	if pairs, ok := ap[asset.Futures]; ok && s.Channel == subscription.OrderbookChannel && (s.Asset == asset.All || s.Asset == asset.Futures) {
		ap[asset.Futures] = e.filterAssetFeedPairs(s, asset.Futures, pairs)
	}
	for assetType, pairs := range ap {
		// Assetless private channels have no pair format to look up
		if assetType == asset.Empty {
			continue
		}
		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return "", err
		}
		ap[assetType] = pairs.Format(format)
	}
	return "", nil
}

// channelName returns the correct channel name for the asset
func channelName(s *subscription.Subscription, a asset.Item) string {
	if byAsset, hasAsset := subscriptionNames[a]; hasAsset {
		if name, ok := byAsset[s.Channel]; ok {
			return name
		}
	}
	if allAssets, hasAll := subscriptionNames[asset.All]; hasAll {
		if name, ok := allAssets[s.Channel]; ok {
			return name
		}
	}
	return s.Channel
}

// mergeMarginPairs assigns each pair to one subscription on KuCoin's shared spot and margin feeds.
func (e *Exchange) mergeMarginPairs(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) string {
	if strings.HasPrefix(s.Channel, "/margin") {
		return ""
	}
	switch s.Asset {
	case asset.All:
		spotPairs, spotEnabled := ap[asset.Spot]
		marginPairs, marginEnabled := ap[asset.Margin]
		if spotEnabled {
			if marginEnabled {
				spotPairs = spotPairs.Add(marginPairs...)
				ap[asset.Margin] = currency.Pairs{}
			}
			ap[asset.Spot] = e.filterSharedFeedPairs(s, spotPairs)
		} else if marginEnabled {
			ap[asset.Margin] = e.filterSharedFeedPairs(s, marginPairs)
		}
	case asset.Spot:
		pairs := ap[asset.Spot]
		if len(s.Pairs) == 0 {
			for _, other := range e.Features.Subscriptions {
				if other.Asset == asset.Margin && e.sharedFeedCandidate(other, s) {
					pairs = pairs.Add(e.sharedFeedPairs(other)...)
				}
			}
		}
		ap[asset.Spot] = e.filterSharedFeedPairs(s, pairs)
	case asset.Margin:
		ap[asset.Margin] = e.filterSharedFeedPairs(s, ap[asset.Margin])
	}
	return ""
}

func (e *Exchange) filterAssetFeedPairs(s *subscription.Subscription, assetType asset.Item, pairs currency.Pairs) currency.Pairs {
	score := e.sharedFeedPriority(s)
	claimed := make(map[[2]*currency.Item]struct{})
	for _, other := range e.Features.Subscriptions {
		if other == s || other.QualifiedChannel != "" || other.Authenticated && !e.Websocket.CanUseAuthenticatedEndpoints() ||
			(other.Asset != asset.All && other.Asset != assetType) || !sameSharedFeed(other, s) {
			continue
		}
		if otherScore := e.sharedFeedPriority(other); otherScore > score || otherScore == score && other.Asset != s.Asset && other.Asset == assetType {
			for _, pair := range e.subscriptionPairsForAsset(other, assetType) {
				claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}] = struct{}{}
			}
		}
	}
	return slices.DeleteFunc(common.SortStrings(slices.Clone(pairs)), func(pair currency.Pair) bool {
		_, ok := claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}]
		return ok
	})
}

func (e *Exchange) subscriptionPairsForAsset(s *subscription.Subscription, assetType asset.Item) currency.Pairs {
	if len(s.Pairs) != 0 {
		return s.Pairs
	}
	pairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return nil
	}
	return pairs
}

func (e *Exchange) filterSharedFeedPairs(s *subscription.Subscription, pairs currency.Pairs) currency.Pairs {
	score := e.sharedFeedPriority(s)
	claimed := make(map[[2]*currency.Item]struct{})
	for _, other := range e.Features.Subscriptions {
		if other == s || !e.sharedFeedCandidate(other, s) {
			continue
		}
		if otherScore := e.sharedFeedPriority(other); otherScore > score || otherScore == score && other.Asset != s.Asset && sharedAssetPriority(other.Asset) > sharedAssetPriority(s.Asset) {
			for _, pair := range e.sharedFeedPairs(other) {
				claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}] = struct{}{}
			}
		}
	}
	return slices.DeleteFunc(common.SortStrings(slices.Clone(pairs)), func(pair currency.Pair) bool {
		_, ok := claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}]
		return ok
	})
}

func (e *Exchange) sharedFeedCandidate(candidate, reference *subscription.Subscription) bool {
	if candidate.QualifiedChannel != "" || candidate.Authenticated && !e.Websocket.CanUseAuthenticatedEndpoints() ||
		(candidate.Asset != asset.All && candidate.Asset != asset.Spot && candidate.Asset != asset.Margin) || !sameSharedFeed(candidate, reference) {
		return false
	}
	if candidate.Asset == asset.All {
		return e.CurrencyPairs.IsAssetEnabled(asset.Spot) == nil || e.CurrencyPairs.IsAssetEnabled(asset.Margin) == nil
	}
	return e.CurrencyPairs.IsAssetEnabled(candidate.Asset) == nil
}

func (e *Exchange) sharedFeedPairs(s *subscription.Subscription) currency.Pairs {
	if len(s.Pairs) != 0 {
		return s.Pairs
	}
	var pairs currency.Pairs
	for _, assetType := range []asset.Item{asset.Spot, asset.Margin} {
		if s.Asset != asset.All && s.Asset != assetType {
			continue
		}
		enabled, err := e.GetEnabledPairs(assetType)
		if err == nil {
			pairs = pairs.Add(enabled...)
		}
	}
	if s.Asset == asset.Spot {
		for _, other := range e.Features.Subscriptions {
			if other.Asset == asset.Margin && e.sharedFeedCandidate(other, s) {
				pairs = pairs.Add(e.sharedFeedPairs(other)...)
			}
		}
	}
	return pairs
}

func (e *Exchange) sharedFeedPriority(s *subscription.Subscription) int {
	priority := 0
	if len(s.Pairs) != 0 {
		priority += 2
	}
	if s.Asset != asset.All {
		priority++
	}
	// A public orderbook topic keeps any candle suffix, where KuCoin sends nothing, so that entry ranks below every entry
	// with a served topic and yields the pairs they share; the realtime rewrite drops the suffix, so auth leaves ranks alone.
	if s.Channel == subscription.OrderbookChannel && !e.Websocket.CanUseAuthenticatedEndpoints() {
		if _, err := IntervalToString(s.Interval); err == nil {
			priority -= 4
		}
	}
	return priority
}

func sharedAssetPriority(a asset.Item) int {
	switch a {
	case asset.Spot:
		return 2
	case asset.Margin:
		return 1
	default:
		return 0
	}
}

// isSymbolChannel returns if the channel expects receive a symbol
func isSymbolChannel(s *subscription.Subscription) bool {
	switch channelName(s, s.Asset) {
	case spotOrderV2Channel, spotOrderV1Channel, accountBalanceChannel, spotMarketAdvancedChannel, marginPositionChannel,
		futuresAnnouncementChannel, futuresTradeOrderChannel, futuresAccountBalanceEventChannel, futuresPositionAllChannel,
		futuresMarginModeChannel, futuresCrossLeverageChannel, futuresStopOrdersLifecycleEventChannel:
		return false
	}
	return true
}

// channelInterval returns the channel interval if it has one
func channelInterval(s *subscription.Subscription) string {
	switch channelName(s, s.Asset) {
	case marketCandlesChannel, futuresLimitCandleChannel:
		if i, err := IntervalToString(s.Interval); err == nil {
			return i
		}
	}
	return ""
}

func sameSharedFeed(a, b *subscription.Subscription) bool {
	if a.Channel != b.Channel {
		return false
	}
	if a.Channel == subscription.OrderbookChannel {
		return true
	}
	if a.Levels != b.Levels {
		return false
	}
	aInterval, _ := IntervalToString(a.Interval)
	bInterval, _ := IntervalToString(b.Interval)
	return aInterval == bInterval
}

// joinPairsWithInterval returns a list of currency pair symbols joined by comma
// If the subscription has a viable interval it's appended after each symbol
func joinPairsWithInterval(b currency.Pairs, s *subscription.Subscription) string {
	out := make([]string, len(b))
	suffix, err := IntervalToString(s.Interval)
	if err == nil {
		suffix = "_" + suffix
	}
	for i, p := range b {
		out[i] = p.String() + suffix
	}
	return strings.Join(out, ",")
}

const subTplText = `
{{- mergeMarginPairs $.S $.AssetPairs }}
{{- formatSubscriptionPairs $.S $.AssetPairs }}
{{- if isSymbolChannel $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- with $name := channelName $.S $asset }}
			{{- if and (eq $name "/market/ticker") (gt (len $pairs) 10) }}
				{{- $name -}} :all
				{{- with $i := channelInterval $.S }}_{{ $i }}{{ end }}
				{{- $.BatchSize }} {{- len $pairs }}
			{{- else }}
				{{- range $b := batch $pairs 1 }}
					{{- $name -}} : {{- joinPairsWithInterval $b $.S }}
					{{- $.PairSeparator }}
				{{- end }}
				{{- $.BatchSize -}} 1
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- channelName $.S $.S.Asset }}
{{- end }}
`
