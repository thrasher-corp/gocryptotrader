package deribit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// The message filters of the websocket's connections, which the manager's GetConnection returns a connection by
const (
	// WebsocketRequestConnection filters the connection the WebsocketAPI methods are sent over, which Deribit advises
	// keeping apart from market data, so a burst of notifications does not hold up an order
	WebsocketRequestConnection = "requests"
	// WebsocketSubscriptionConnection filters the connections the manager spreads subscriptions over
	WebsocketSubscriptionConnection = "subscriptions"
)

const (
	// websocketHeartbeatInterval has Deribit send a heartbeat every 15 seconds, so a stale connection is noticed well
	// within the traffic timeout
	websocketHeartbeatInterval = 15 * time.Second
	// maximumChannelsPerConnection is the most channels a connection is subscribed to, the most Deribit takes in one
	// subscribe request, so each batch of subscriptions the manager gives a connection fits one request of each kind
	maximumChannelsPerConnection = 500

	wsMethodSubscription = "subscription"
	wsMethodHeartbeat    = "heartbeat"
	// wsHeartbeatTestRequest is the heartbeat type that Deribit closes the connection for unless a public/test request
	// answers it
	wsHeartbeatTestRequest = "test_request"

	// Channel intervals; raw is only available to an authenticated connection, and agg2 aggregates about a second
	intervalRaw       = "raw"
	intervalHundredMS = "100ms"
	intervalAgg2      = "agg2"

	// Values a kind or currency channel parameter takes to select every kind or currency
	channelAny = "any"

	// Book channel notification types
	bookSnapshot = "snapshot"
	bookChange   = "change"
	bookDelete   = "delete"
)

// Deribit's subscription channels, which a subscription names, other than through a standard channel, to subscribe to
const (
	ChannelPlatformState                    = "platform_state"
	ChannelPlatformPublicMethodsState       = "platform_state.public_methods_state"
	ChannelAnnouncements                    = "announcements"
	ChannelBook                             = "book"
	ChannelTicker                           = "ticker"
	ChannelIncrementalTicker                = "incremental_ticker"
	ChannelPerpetual                        = "perpetual"
	ChannelQuote                            = "quote"
	ChannelTrades                           = "trades"
	ChannelBlockTradeConfirmations          = "block_trade_confirmations"
	ChannelPriceIndex                       = "deribit_price_index"
	ChannelPriceRanking                     = "deribit_price_ranking"
	ChannelPriceStatistics                  = "deribit_price_statistics"
	ChannelVolatilityIndex                  = "deribit_volatility_index"
	ChannelMarkPriceOptions                 = "markprice.options"
	ChannelEstimatedExpirationPrice         = "estimated_expiration_price"
	ChannelChartTrades                      = "chart.trades"
	ChannelInstrumentCreation               = "instrument.creation"
	ChannelInstrumentState                  = "instrument.state"
	ChannelUserMarketMakerProtectionTrigger = "user.mmp_trigger"
	ChannelUserPortfolio                    = "user.portfolio"
	ChannelUserTrades                       = "user.trades"
	ChannelUserComboTrades                  = "user.combo_trades"
	ChannelUserOrders                       = "user.orders"
	ChannelUserChanges                      = "user.changes"
	ChannelUserAccessLog                    = "user.access_log"
	ChannelUserLock                         = "user.lock"
	ChannelUserLiquidation                  = "user.liquidation"
	ChannelUserIsolatedLiquidation          = "user.isolated.liquidation"
	ChannelUserLiquiditySupportProgram      = "user.lsp"
	ChannelBlockRFQMaker                    = "block_rfq.maker"
	ChannelBlockRFQTaker                    = "block_rfq.taker"
	ChannelBlockRFQMakerQuotes              = "block_rfq.maker.quotes"
	ChannelBlockRFQTrades                   = "block_rfq.trades"
)

var (
	errNoTickerSnapshot            = errors.New("incremental ticker change received before its snapshot")
	errUnsupportedInstrumentFormat = errors.New("unsupported instrument format")
	errChannelParameter            = errors.New("invalid channel parameter")
	errChannelNeedsAuthentication  = errors.New("channel needs an authenticated subscription")
	errChannelNeedsInstrument      = errors.New("channel needs an asset and pairs")
	errOrderbookOutOfSequence      = errors.New("orderbook change out of sequence")
	errNoLongerResubscribing       = errors.New("subscription is no longer resubscribing")
	errMalformedNotification       = errors.New("malformed notification")
)

// websocketChannel describes how a Deribit channel is qualified
type websocketChannel struct {
	// instrument channels may be qualified per pair by its instrument name, for a subscription with an asset
	instrument bool
	// kindCurrency channels may be qualified by the kind and currency parameters, each any when unset, for a
	// subscription without an asset; kindRequired ones need an instrument kind, as Deribit refuses any
	kindCurrency bool
	kindRequired bool
	// interval channels end with the subscription's interval: 100ms, raw (one millisecond) or agg2 (one second)
	interval bool
	// resolution channels end with the subscription's interval as a candle resolution
	resolution bool
	// indexName channels are qualified by the index_name parameter, such as btc_usd
	indexName bool
	// currency channels are qualified by the currency parameter, any when unset; optionalCurrency channels only when it
	// is set
	currency         bool
	optionalCurrency bool
	// private channels need an authenticated subscription
	private bool
}

// websocketChannels lists every Deribit channel a subscription may name
var websocketChannels = map[string]websocketChannel{
	ChannelPlatformState:                    {},
	ChannelPlatformPublicMethodsState:       {},
	ChannelAnnouncements:                    {},
	ChannelBook:                             {instrument: true, interval: true},
	ChannelTicker:                           {instrument: true, interval: true},
	ChannelIncrementalTicker:                {instrument: true},
	ChannelPerpetual:                        {instrument: true, interval: true},
	ChannelQuote:                            {instrument: true},
	ChannelTrades:                           {instrument: true, kindCurrency: true, kindRequired: true, interval: true},
	ChannelBlockTradeConfirmations:          {optionalCurrency: true, private: true},
	ChannelPriceIndex:                       {indexName: true},
	ChannelPriceRanking:                     {indexName: true},
	ChannelPriceStatistics:                  {indexName: true},
	ChannelVolatilityIndex:                  {indexName: true},
	ChannelMarkPriceOptions:                 {indexName: true},
	ChannelEstimatedExpirationPrice:         {indexName: true},
	ChannelChartTrades:                      {instrument: true, resolution: true},
	ChannelInstrumentCreation:               {kindCurrency: true},
	ChannelInstrumentState:                  {kindCurrency: true},
	ChannelUserMarketMakerProtectionTrigger: {indexName: true, private: true},
	ChannelUserPortfolio:                    {currency: true, private: true},
	ChannelUserTrades:                       {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserComboTrades:                  {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserOrders:                       {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserChanges:                      {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserAccessLog:                    {private: true},
	ChannelUserLock:                         {private: true},
	ChannelUserLiquidation:                  {private: true},
	ChannelUserIsolatedLiquidation:          {private: true},
	ChannelUserLiquiditySupportProgram:      {private: true},
	ChannelBlockRFQMaker:                    {currency: true, private: true},
	ChannelBlockRFQTaker:                    {currency: true, private: true},
	ChannelBlockRFQMakerQuotes:              {currency: true, private: true},
	ChannelBlockRFQTrades:                   {currency: true},
}

// websocketStandardChannels maps standard channels to the Deribit channels that serve them
var websocketStandardChannels = map[string]string{
	subscription.TickerChannel:    ChannelTicker,
	subscription.OrderbookChannel: ChannelBook,
	subscription.AllTradesChannel: ChannelTrades,
	subscription.CandlesChannel:   ChannelChartTrades,
	subscription.MyOrdersChannel:  ChannelUserOrders,
	subscription.MyTradesChannel:  ChannelUserTrades,
	subscription.MyAccountChannel: ChannelUserPortfolio,
}

// websocketChannelsByLength lists the channel names longest first, so a notification's channel is matched to the most
// specific name it starts with
var websocketChannelsByLength = func() []string {
	names := slices.Collect(maps.Keys(websocketChannels))
	slices.SortFunc(names, func(a, b string) int { return cmp.Or(len(b)-len(a), strings.Compare(a, b)) })
	return names
}()

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneDay},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, // Raw is available for authenticated users
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
}

// WebsocketAPI calls Deribit's methods over the websocket's request connection rather than REST; a method with a REST
// form takes that form's name and arguments
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the methods called over the websocket's request connection
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// WebsocketConnect dials one of the websocket's connections and asks Deribit to send it heartbeats; the manager reads a
// connection only once it is dialled, so the heartbeat request waits for its answer in the background
func (e *Exchange) WebsocketConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	go func() {
		// The connection still works without heartbeats, so a failure is only logged
		if err := e.setWebsocketHeartbeat(ctx, conn); err != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket heartbeat could not be set: %s", e.Name, err)
		}
	}()
	return nil
}

// setWebsocketHeartbeat asks Deribit to send a connection heartbeats
func (e *Exchange) setWebsocketHeartbeat(ctx context.Context, conn websocket.Connection) error {
	const method = "public/set_heartbeat"
	var resp string
	if err := e.sendWebsocketRequest(ctx, conn, nonMatchingEPL, method, &SetHeartbeatRequest{Interval: websocketHeartbeatInterval.Seconds()}, &resp); err != nil {
		return err
	}
	return matchOKResponse(method, resp)
}

// connectRequestConnection dials the request connection as WebsocketConnect dials any other. The manager dials it first
// in every cycle of connections, whether connecting, reconnecting or flushing, so it lets the cycle log its connections
// in whenever authenticated websocket support is enabled, and a failed login stops authenticated requests only until the
// websocket connects again
func (e *Exchange) connectRequestConnection(ctx context.Context, conn websocket.Connection) error {
	e.Websocket.SetCanUseAuthenticatedEndpoints(e.IsWebsocketAuthenticationSupported())
	return e.WebsocketConnect(ctx, conn)
}

// websocketAuthenticate logs in a connection the manager connects while authenticated requests are allowed. A failed
// login is logged and stops authenticated requests, leaving the connection to serve public requests and channels
func (e *Exchange) websocketAuthenticate(ctx context.Context, conn websocket.Connection) error {
	if err := e.wsLogin(ctx, conn); err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket authentication failed: %s", e.Name, err)
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	return nil
}

// wsLogin authenticates a connection with a client signature, so the secret is not sent
func (e *Exchange) wsLogin(ctx context.Context, conn websocket.Connection) error {
	if !e.IsWebsocketAuthenticationSupported() {
		return exchange.ErrAuthenticationSupportNotEnabled
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	arg, err := NewClientSignatureRequest(creds, time.Now(), e.Requester.GetNonce(nonce.UnixNano).String(), "")
	if err != nil {
		return err
	}
	if _, err := e.authenticateConnection(ctx, conn, arg); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// requestConnection returns the connection the WebsocketAPI methods are sent over
func (e *Exchange) requestConnection(method string) (websocket.Connection, error) {
	conn, err := e.Websocket.GetConnection(WebsocketRequestConnection)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, method)
	}
	return conn, nil
}

// SendWebsocketRequest sends a JSON-RPC request over the websocket's request connection and decodes its result; a
// private method needs the connection to be authenticated
func (e *Exchange) SendWebsocketRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	if isPrivateMethod(method) && !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: %s", errWebsocketNotAuthenticated, method)
	}
	conn, err := e.requestConnection(method)
	if err != nil {
		return err
	}
	return e.sendWebsocketRequest(ctx, conn, epl, method, params, result)
}

// sendWebsocketRequest sends a JSON-RPC request over a connection and decodes its result; a private method needs the
// connection to be authenticated
func (e *Exchange) sendWebsocketRequest(ctx context.Context, conn websocket.Connection, epl request.EndpointLimit, method string, params, result any) error {
	private := isPrivateMethod(method)
	if private && !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: %s", errWebsocketNotAuthenticated, method)
	}
	if !websocketConnected(conn) {
		return fmt.Errorf("%w: %s", websocket.ErrNotConnected, method)
	}
	if private {
		var err error
		if params, err = authorisedParams(ctx, params); err != nil {
			return err
		}
	}
	return retryRequest(ctx, func() error {
		if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
			return err
		}
		req := &JSONRPCRequest{JSONRPC: jsonRPCVersion, ID: e.MessageSequence(), Method: method, Params: params}
		raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
		if err != nil {
			return err
		}
		resp := &JSONRPCResponse{Result: result}
		var challenged privateResult
		if private {
			challenged.result = result
			resp.Result = &challenged
		}
		if err := json.Unmarshal(raw, resp); err != nil {
			return err
		}
		if resp.Error != nil {
			return resp.Error
		}
		if challenged.challenge != nil {
			return challenged.challenge
		}
		return nil
	})
}

// websocketConnected reports whether a connection is connected; the manager dials a new connection each time, so one it
// has shut down stays closed
func websocketConnected(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return ok && c.IsConnected()
}

// wsHandleData passes a response to the request awaiting it on the connection, answers a heartbeat test request and
// processes a subscription notification
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var envelope WebsocketEnvelope
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return fmt.Errorf("%s could not parse websocket data: %w: %s", e.Name, err, respRaw)
	}
	if envelope.ID != 0 {
		return conn.RequireMatchWithData(envelope.ID, respRaw)
	}
	switch envelope.Method {
	case wsMethodHeartbeat:
		if envelope.Params.Type == wsHeartbeatTestRequest {
			go e.wsAnswerHeartbeat(ctx, conn)
		}
		return nil
	case wsMethodSubscription:
		return e.wsHandleNotification(ctx, conn, envelope.Params.Channel, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// wsAnswerHeartbeat answers a heartbeat test request with a public/test request on the connection it arrived on, without
// which Deribit closes that connection
func (e *Exchange) wsAnswerHeartbeat(ctx context.Context, conn websocket.Connection) {
	var resp *TestConnectionResponse
	if err := e.sendWebsocketRequest(ctx, conn, nonMatchingEPL, "public/test", nil, &resp); err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket heartbeat test request could not be answered: %s", e.Name, err)
	}
}

// wsHandleNotification passes a subscription notification to its channel's handler; a book's is resubscribed on the
// connection it arrived on when it cannot be applied
func (e *Exchange) wsHandleNotification(ctx context.Context, conn websocket.Connection, channel string, respRaw []byte) error {
	switch notificationChannelName(channel) {
	case ChannelBook:
		return e.wsHandleOrderbook(ctx, conn, channel, respRaw)
	case ChannelIncrementalTicker:
		return e.wsHandleIncrementalTicker(ctx, channel, respRaw)
	case ChannelChartTrades:
		return e.wsHandleCandle(ctx, channel, respRaw)
	case ChannelTicker:
		return e.wsHandleTicker(ctx, respRaw)
	case ChannelTrades:
		return e.wsHandleTrades(ctx, respRaw)
	case ChannelUserOrders:
		return e.wsHandleOrders(ctx, channel, respRaw)
	case ChannelUserTrades, ChannelUserComboTrades:
		return e.wsHandleUserTrades(ctx, respRaw)
	case ChannelPerpetual:
		return wsRelay[WebsocketPerpetualInterest](ctx, e, respRaw)
	case ChannelQuote:
		return e.wsHandleQuote(ctx, respRaw)
	case ChannelPriceIndex:
		return wsRelay[WebsocketPriceIndex](ctx, e, respRaw)
	case ChannelPriceRanking:
		return wsRelay[[]WebsocketPriceRanking](ctx, e, respRaw)
	case ChannelPriceStatistics:
		return wsRelay[WebsocketPriceStatistics](ctx, e, respRaw)
	case ChannelVolatilityIndex:
		return wsRelay[WebsocketVolatilityIndex](ctx, e, respRaw)
	case ChannelMarkPriceOptions:
		return wsRelay[[]WebsocketMarkPriceOption](ctx, e, respRaw)
	case ChannelEstimatedExpirationPrice:
		return wsRelay[WebsocketEstimatedExpirationPrice](ctx, e, respRaw)
	case ChannelInstrumentState:
		return wsRelay[WebsocketInstrumentState](ctx, e, respRaw)
	case ChannelPlatformState:
		return wsRelay[WebsocketPlatformState](ctx, e, respRaw)
	case ChannelPlatformPublicMethodsState:
		return wsRelay[WebsocketPublicMethodsState](ctx, e, respRaw)
	case ChannelUserLock:
		return wsRelay[WebsocketUserLock](ctx, e, respRaw)
	case ChannelUserLiquidation, ChannelUserIsolatedLiquidation:
		return wsRelay[WebsocketUserLiquidation](ctx, e, respRaw)
	case ChannelUserMarketMakerProtectionTrigger:
		return wsRelay[WebsocketMarketMakerProtectionTrigger](ctx, e, respRaw)
	case ChannelAnnouncements:
		return wsRelay[WebsocketAnnouncement](ctx, e, respRaw)
	case ChannelInstrumentCreation:
		return wsRelay[WebsocketInstrumentCreation](ctx, e, respRaw)
	case ChannelUserPortfolio:
		return e.wsHandlePortfolio(ctx, respRaw)
	case ChannelUserChanges:
		return wsRelay[WebsocketUserChanges](ctx, e, respRaw)
	case ChannelUserAccessLog:
		return wsRelay[AccessLogEntry](ctx, e, respRaw)
	case ChannelUserLiquiditySupportProgram:
		return wsRelay[WebsocketLiquiditySupportProgramEvent](ctx, e, respRaw)
	case ChannelBlockTradeConfirmations:
		return wsRelay[PendingBlockTradeResponse](ctx, e, respRaw)
	case ChannelBlockRFQMaker, ChannelBlockRFQTaker:
		return wsRelay[BlockRFQResponse](ctx, e, respRaw)
	case ChannelBlockRFQMakerQuotes:
		return wsRelay[WebsocketBlockRFQQuotes](ctx, e, respRaw)
	case ChannelBlockRFQTrades:
		return wsRelay[BlockRFQTrade](ctx, e, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// notificationChannelName returns the Deribit channel a notification's qualified channel belongs to, or an empty string
// for an unknown channel
func notificationChannelName(channel string) string {
	for _, name := range websocketChannelsByLength {
		if channel == name || strings.HasPrefix(channel, name+".") {
			return name
		}
	}
	return ""
}

// decodeNotification decodes a subscription notification's data into the value data points to
func decodeNotification(respRaw []byte, data any) error {
	notification := WebsocketNotification{Params: WebsocketNotificationParams{Data: data}}
	if err := json.Unmarshal(respRaw, &notification); err != nil {
		return fmt.Errorf("%w: %w", errMalformedNotification, err)
	}
	return nil
}

// wsRelay decodes a notification's data into its exported type and relays it
func wsRelay[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	data := new(T)
	if err := decodeNotification(respRaw, data); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

// generateSubscriptions returns the configured subscriptions with their channels qualified, leaving out those Deribit
// refuses for the spot it routes to Coinbase Exchange
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := e.expandSubscriptions(e.Features.Subscriptions)
	if err != nil {
		return nil, err
	}
	if slices.ContainsFunc(subs, isRoutedSpotSensitive) {
		e.coinbaseRoutedSpotMtx.RLock()
		loaded := e.coinbaseRoutedSpot != nil
		e.coinbaseRoutedSpotMtx.RUnlock()
		// Only the spot instruments say which spot is routed, and no spot pairs may have been fetched yet. They are
		// fetched over REST, as the manager generates subscriptions holding the lock that finding the websocket's request
		// connection waits for
		if !loaded {
			instruments, err := e.getInstruments(context.TODO(), &InstrumentsRequest{Currency: anyCurrency, Kind: KindSpot}, false)
			if err == nil {
				_, err = e.tradablePairs(asset.Spot, instruments)
			}
			if err != nil {
				return nil, fmt.Errorf("loading the spot instruments Deribit routes to Coinbase Exchange: %w", err)
			}
		}
	}
	return slices.DeleteFunc(subs, e.isUnstreamedSubscription), nil
}

// expandSubscriptions qualifies subscriptions' channels and keys each by its qualified channel. A subscription already
// qualified comes back as it is, and may be held, so it is keyed only when it is not keyed by its qualified channel
// already, as the subscription store reads a held subscription's key without its lock
func (e *Exchange) expandSubscriptions(subs subscription.List) (subscription.List, error) {
	expanded, err := subs.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	for _, s := range expanded {
		if key, ok := s.EnsureKeyed().(qualifiedChannelKey); !ok || key.Subscription != s {
			s.SetKey(qualifiedChannelKey{s})
		}
	}
	return expanded, nil
}

// qualifiedChannelKey identifies a subscription by its qualified channel, as channels such as an index's differ only by
// a parameter the default key leaves out
type qualifiedChannelKey struct {
	*subscription.Subscription
}

// Match reports whether a subscription has the same qualified channel
func (k qualifiedChannelKey) Match(eachKey subscription.MatchableKey) bool {
	if eachKey == nil {
		return false
	}
	s := eachKey.GetSubscription()
	return s != nil && s.QualifiedChannel == k.QualifiedChannel
}

// GetSubscription returns the subscription the key identifies
func (k qualifiedChannelKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

// String returns the qualified channel
func (k qualifiedChannelKey) String() string {
	if k.Subscription == nil {
		return "Uninitialised qualifiedChannelKey"
	}
	return k.QualifiedChannel
}

// isUnstreamedSubscription reports whether Deribit refuses a subscription for the spot it routes to Coinbase Exchange: a
// routed pair's trades or candles, or spot trades by kind for every currency or a routed pair's base or quote
func (e *Exchange) isUnstreamedSubscription(sub *subscription.Subscription) bool {
	switch {
	case isSpotTradesOrCandles(sub):
		return e.isCoinbaseRoutedSpot(formatPairString(asset.Spot, sub.Pairs[0]))
	case isSpotKindTrades(sub):
		ccy, err := channelCurrency(sub.Params, true)
		return err == nil && e.isCoinbaseRoutedSpotCurrency(ccy)
	default:
		return false
	}
}

// isRoutedSpotSensitive reports whether Deribit refuses a subscription when spot it stands for is routed to Coinbase
// Exchange
func isRoutedSpotSensitive(sub *subscription.Subscription) bool {
	return isSpotTradesOrCandles(sub) || isSpotKindTrades(sub)
}

// isSpotTradesOrCandles reports whether a subscription is to the trades or candles of one spot pair
func isSpotTradesOrCandles(sub *subscription.Subscription) bool {
	if sub.Asset != asset.Spot || len(sub.Pairs) != 1 {
		return false
	}
	name, err := websocketChannelName(sub)
	return err == nil && (name == ChannelTrades || name == ChannelChartTrades)
}

// isSpotKindTrades reports whether a subscription is to the trades of spot by kind, without an asset
func isSpotKindTrades(sub *subscription.Subscription) bool {
	if sub.Asset != asset.Empty {
		return false
	}
	name, err := websocketChannelName(sub)
	if err != nil || name != ChannelTrades {
		return false
	}
	kind, err := channelParameter(sub.Params, "kind", channelAny)
	return err == nil && strings.EqualFold(kind, KindSpot)
}

// GetSubscriptionTemplate returns the channel qualification template, which qualifies an instrument channel per pair
// for a subscription with an asset and any other channel once
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":       websocketChannelName,
		"perInstrument":     func(name string) bool { return websocketChannels[name].instrument },
		"qualifyInstrument": qualifyInstrumentChannel,
		"qualify":           qualifyChannel,
	}).Parse(websocketSubscriptionTemplate)
}

// websocketSubscriptionTemplate qualifies a subscription per pair for instrument channels and once otherwise
const websocketSubscriptionTemplate = `
{{- $name := channelName $.S }}
{{- if and $.S.Asset (perInstrument $name) }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- range $pair := $pairs }}
			{{- qualifyInstrument $.S $asset $pair }}
			{{- $.PairSeparator }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- qualify $.S }}
{{- end }}
`

// websocketChannelName returns the Deribit channel a subscription refers to: a standard channel's Deribit channel, or
// the channel it names
func websocketChannelName(sub *subscription.Subscription) (string, error) {
	if sub == nil {
		return "", common.ErrNilPointer
	}
	if name, ok := websocketStandardChannels[sub.Channel]; ok {
		return name, nil
	}
	if _, ok := websocketChannels[sub.Channel]; ok {
		return sub.Channel, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, sub.Channel)
}

// qualifyInstrumentChannel returns a subscription's channel for one of its pairs
func qualifyInstrumentChannel(sub *subscription.Subscription, a asset.Item, pair currency.Pair) (string, error) {
	name, channel, err := validateChannelSubscription(sub)
	if err != nil {
		return "", err
	}
	qualified := name + "." + formatPairString(a, pair)
	if name == ChannelBook && sub.Levels != 0 {
		group, err := bookGroup(sub.Params)
		if err != nil {
			return "", err
		}
		switch sub.Levels {
		case 1, 10, 20:
		default:
			return "", fmt.Errorf("%w: %d, as a grouped book sends 1, 10 or 20 levels", subscription.ErrInvalidLevel, sub.Levels)
		}
		qualified += "." + group + "." + strconv.Itoa(sub.Levels)
	}
	return appendChannelInterval(qualified, sub, &channel)
}

// qualifyChannel returns a subscription's channel when it is not qualified per pair
func qualifyChannel(sub *subscription.Subscription) (string, error) {
	name, channel, err := validateChannelSubscription(sub)
	if err != nil {
		return "", err
	}
	qualified := name
	switch {
	case channel.kindCurrency:
		kind, err := channelParameter(sub.Params, "kind", channelAny)
		if err != nil {
			return "", err
		}
		if channel.kindRequired && strings.EqualFold(kind, channelAny) {
			return "", fmt.Errorf("%w: %s needs an instrument kind, as Deribit refuses %s", errChannelParameter, name, kind)
		}
		ccy, err := channelCurrency(sub.Params, true)
		if err != nil {
			return "", err
		}
		qualified += "." + kind + "." + ccy
	case channel.instrument:
		return "", fmt.Errorf("%w: %s", errChannelNeedsInstrument, name)
	case channel.indexName:
		index, err := channelParameter(sub.Params, "index_name", "")
		if err != nil {
			return "", err
		}
		qualified += "." + index
	case channel.currency:
		ccy, err := channelCurrency(sub.Params, true)
		if err != nil {
			return "", err
		}
		qualified += "." + ccy
	case channel.optionalCurrency:
		ccy, err := channelCurrency(sub.Params, false)
		if err != nil {
			return "", err
		}
		if ccy != "" {
			qualified += "." + ccy
		}
	}
	return appendChannelInterval(qualified, sub, &channel)
}

// validateChannelSubscription returns the Deribit channel a subscription refers to, checking a private channel is
// subscribed authenticated
func validateChannelSubscription(sub *subscription.Subscription) (string, websocketChannel, error) {
	name, err := websocketChannelName(sub)
	if err != nil {
		return "", websocketChannel{}, err
	}
	channel := websocketChannels[name]
	if channel.private && !sub.Authenticated {
		return "", websocketChannel{}, fmt.Errorf("%w: %s", errChannelNeedsAuthentication, name)
	}
	return name, channel, nil
}

// appendChannelInterval appends the subscription's interval to a channel that takes one; Deribit rejects a whole
// subscribe request holding a raw channel from an unauthenticated connection, so a raw channel must be subscribed
// authenticated
func appendChannelInterval(qualified string, sub *subscription.Subscription, channel *websocketChannel) (string, error) {
	switch {
	case channel.resolution:
		resolution, err := chartResolution(sub.Interval)
		if err != nil {
			return "", err
		}
		return qualified + "." + resolution, nil
	case channel.interval:
		var interval string
		switch sub.Interval {
		case kline.HundredMilliseconds:
			interval = intervalHundredMS
		case kline.ThousandMilliseconds:
			interval = intervalAgg2
		case kline.Raw:
			if !sub.Authenticated {
				return "", fmt.Errorf("%w: %s raw interval", errChannelNeedsAuthentication, qualified)
			}
			interval = intervalRaw
		default:
			return "", fmt.Errorf("%w: %s, as Deribit aggregates over 100ms, 1s or raw", kline.ErrUnsupportedInterval, sub.Interval)
		}
		return qualified + "." + interval, nil
	}
	return qualified, nil
}

// chartResolution returns the chart.trades resolution of an interval
func chartResolution(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.TenMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.ThreeHour, kline.SixHour, kline.TwelveHour:
		return strconv.FormatInt(int64(interval.Duration()/time.Minute), 10), nil
	case kline.OneDay:
		return "1D", nil
	default:
		return "", fmt.Errorf("%w: %s, as chart.trades serves 1m to 12h and 1D", kline.ErrUnsupportedInterval, interval)
	}
}

// bookGroup returns a grouped book's price grouping from the group parameter, none when unset; a number, as
// configuration supplies it, is formatted as Deribit names it
func bookGroup(params map[string]any) (string, error) {
	var group string
	switch value := params["group"].(type) {
	case nil:
		return "none", nil
	case string:
		group = value
	case float64:
		group = strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		group = strconv.Itoa(value)
	default:
		return "", fmt.Errorf("%w: group %v", errChannelParameter, value)
	}
	switch group {
	case "none", "1", "2", "5", "10", "25", "100", "250":
		return group, nil
	default:
		return "", fmt.Errorf("%w: group %q", errChannelParameter, group)
	}
}

// channelParameter returns a string channel parameter, or the default when it is unset; a parameter without a default
// is required
func channelParameter(params map[string]any, name, defaultValue string) (string, error) {
	switch value := params[name].(type) {
	case nil:
		if defaultValue == "" {
			return "", fmt.Errorf("%w: %s is required", errChannelParameter, name)
		}
		return defaultValue, nil
	case string:
		if value == "" || strings.Contains(value, ".") {
			return "", fmt.Errorf("%w: %s %q", errChannelParameter, name, value)
		}
		return value, nil
	default:
		return "", fmt.Errorf("%w: %s %v", errChannelParameter, name, value)
	}
}

// channelCurrency returns the currency parameter, upper case as Deribit names currencies, or any; when required an
// unset currency is any, otherwise empty
func channelCurrency(params map[string]any, required bool) (string, error) {
	var ccy string
	switch value := params["currency"].(type) {
	case nil:
		if required {
			return channelAny, nil
		}
		return "", nil
	case string:
		ccy = value
	case currency.Code:
		ccy = value.String()
	default:
		return "", fmt.Errorf("%w: currency %v", errChannelParameter, value)
	}
	if strings.EqualFold(ccy, channelAny) {
		return channelAny, nil
	}
	if ccy == "" || strings.Contains(ccy, ".") {
		return "", fmt.Errorf("%w: currency %q", errChannelParameter, ccy)
	}
	return strings.ToUpper(ccy), nil
}

// Subscribe subscribes a connection to channels, through private/subscribe for private channels, refusing a channel held
// already, unless it is held resubscribing, and a book channel for an instrument with another book channel held or
// requested
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	e.subscriptionMtx.Lock()
	defer e.subscriptionMtx.Unlock()
	subs, err := e.expandSubscriptions(subs)
	if err != nil {
		return err
	}
	subs, errs := e.refuseHeldSubscriptions(subs)
	subs, bookErrs := e.refuseBookConflicts(subs)
	errs = common.AppendError(errs, bookErrs)
	errs = common.AppendError(errs, e.handleSubscription(ctx, conn, true, false, subs.Public()))
	return common.AppendError(errs, e.handleSubscription(ctx, conn, true, true, subs.Private()))
}

// Unsubscribe unsubscribes a connection from channels, through private/unsubscribe for private channels
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	e.subscriptionMtx.Lock()
	defer e.subscriptionMtx.Unlock()
	subs, err := e.expandSubscriptions(subs)
	if err != nil {
		return err
	}
	errs := e.handleSubscription(ctx, conn, false, false, subs.Public())
	return common.AppendError(errs, e.handleSubscription(ctx, conn, false, true, subs.Private()))
}

// handleSubscription subscribes a connection to or unsubscribes it from qualified channels, then stores or removes each
// channel Deribit acknowledged and reports the rest
func (e *Exchange) handleSubscription(ctx context.Context, conn websocket.Connection, subscribe, private bool, subs subscription.List) error {
	if len(subs) == 0 {
		return nil
	}
	acknowledged, err := e.sendSubscriptionRequest(ctx, conn, subscribe, private, subs.QualifiedChannels())
	if err != nil {
		return err
	}
	var errs error
	for _, s := range subs {
		if !slices.Contains(acknowledged, s.QualifiedChannel) {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errUnacknowledgedSubscription, s.QualifiedChannel))
			continue
		}
		if subscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, s))
		} else {
			errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
		}
	}
	return errs
}

// sendSubscriptionRequest subscribes a connection to or unsubscribes it from qualified channels, through
// private/subscribe or private/unsubscribe for private channels, and returns the channels Deribit acknowledged
func (e *Exchange) sendSubscriptionRequest(ctx context.Context, conn websocket.Connection, subscribe, private bool, channels []string) ([]string, error) {
	method, epl := "public/unsubscribe", nonMatchingEPL
	switch {
	case subscribe && private:
		method, epl = "private/subscribe", subscribeEPL
	case subscribe:
		method, epl = "public/subscribe", subscribeEPL
	case private:
		method = "private/unsubscribe"
	}
	var acknowledged []string
	return acknowledged, e.sendWebsocketRequest(ctx, conn, epl, method, &SubscriptionRequest{Channels: channels}, &acknowledged)
}

// refuseHeldSubscriptions returns the subscriptions less each one held already, other than one held resubscribing, and
// an error for each one left out. The manager checks a request for subscriptions held before it subscribes, but not
// under subscriptionMtx, so a resubscription or another request may subscribe one meanwhile
func (e *Exchange) refuseHeldSubscriptions(subs subscription.List) (subscription.List, error) {
	allowed := make(subscription.List, 0, len(subs))
	var errs error
	for _, s := range subs {
		if held := e.Websocket.GetSubscription(s); held != nil && (held != s || s.State() != subscription.ResubscribingState) {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrDuplicate, s.QualifiedChannel))
			continue
		}
		allowed = append(allowed, s)
	}
	return allowed, errs
}

// refuseBookConflicts returns the subscriptions less each book subscription whose instrument has another book channel
// held, or requested in the same list, and an error for each one left out: every book channel of an instrument writes
// its one stored book, which a change from one channel then fails to follow, or a snapshot from another replaces at
// its own depth
func (e *Exchange) refuseBookConflicts(subs subscription.List) (subscription.List, error) {
	held := make(map[string]string)
	for _, s := range e.Websocket.GetSubscriptions() {
		if instrument, ok := bookInstrument(s.QualifiedChannel); ok {
			held[instrument] = s.QualifiedChannel
		}
	}
	requested := make(map[string][]string)
	for _, s := range subs {
		if instrument, ok := bookInstrument(s.QualifiedChannel); ok && !slices.Contains(requested[instrument], s.QualifiedChannel) {
			requested[instrument] = append(requested[instrument], s.QualifiedChannel)
		}
	}
	allowed := make(subscription.List, 0, len(subs))
	var errs error
	for _, s := range subs {
		instrument, ok := bookInstrument(s.QualifiedChannel)
		switch {
		case !ok:
			allowed = append(allowed, s)
		case held[instrument] != "" && held[instrument] != s.QualifiedChannel:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s, as %s is subscribed", subscription.ErrExclusiveSubscription, s.QualifiedChannel, held[instrument]))
		case len(requested[instrument]) > 1:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s, as %s are requested together", subscription.ErrExclusiveSubscription, s.QualifiedChannel, strings.Join(requested[instrument], " and ")))
		default:
			allowed = append(allowed, s)
		}
	}
	return allowed, errs
}

// bookInstrument returns the instrument a qualified book channel is for
func bookInstrument(channel string) (string, bool) {
	rest, ok := strings.CutPrefix(channel, ChannelBook+".")
	if !ok {
		return "", false
	}
	instrument, _, ok := strings.Cut(rest, ".")
	if !ok {
		return "", false
	}
	return instrument, true
}
