package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Message filters telling the websocket's connections apart: market data, user channels, and the trade connection the
// WebsocketAPI methods are sent over
const (
	websocketMarketDataConnection = "market"
	websocketUserConnection       = "user"
	websocketTradeConnection      = "trade"
)

const (
	websocketMethodAuth             = "public/auth"
	websocketMethodHeartbeat        = "public/heartbeat"
	websocketMethodRespondHeartbeat = "public/respond-heartbeat"
	websocketMethodSubscribe        = "subscribe"
	websocketMethodUnsubscribe      = "unsubscribe"

	// defaultWebsocketConnectDelay is how long a connection waits after dialling before it sends anything, as
	// Crypto.com pro-rates a connection's first second of requests by when in the second it opened
	defaultWebsocketConnectDelay = time.Second
	// websocketMaxSubscriptionsPerConnection is how many channels Crypto.com subscribes on one connection before it
	// answers 40107, maximum subscriptions exceeded
	websocketMaxSubscriptionsPerConnection = 400
	// websocketMaxChannelBytes is how many bytes of channel names, with their quotes and commas, a subscribe or unsubscribe
	// request holds: Crypto.com does not answer a request over 4096 bytes, and the rest of a request takes under 200
	websocketMaxChannelBytes = 3900

	// defaultBookDepth is the depth of a book subscription that names none; Crypto.com serves books 10 or 50 levels deep
	defaultBookDepth = 50
	// bookSnapshotInterval is the interval a book subscription names for a full snapshot every 500 milliseconds, in place
	// of a snapshot followed by deltas
	bookSnapshotInterval = kline.Interval(500 * time.Millisecond)
	// underlyingIndexSuffix ends the name of a derivative's underlying index, such as BTCUSD-INDEX
	underlyingIndexSuffix = "-INDEX"
)

// CancelOnDisconnectScopeConnection cancels the orders a connection placed when it disconnects, the only scope
// Crypto.com offers
const CancelOnDisconnectScopeConnection = "CONNECTION"

var (
	errWebsocketNotAuthenticated = errors.New("websocket connection is not authenticated")
	errMalformedWebsocketMessage = errors.New("malformed websocket message")
	errScopeRequired             = errors.New("scope required")
	errNoUnderlyingIndex         = errors.New("instrument has no underlying index")
	errInvalidInteger            = errors.New("invalid integer")
)

// defaultSubscriptions are the market data GoCryptoTrader keeps for every enabled pair: tickers, books 50 levels deep
// with deltas every 100 milliseconds, trades and one minute candles. Each takes one of the 400 subscriptions a connection
// carries for every enabled pair, so the mark, index, funding and settlement channels are left to configuration.
// Authenticated, they also keep the order manager, the fills feed and the account store current with the account's
// orders, advanced orders, trades, balances and positions, each one subscription for every instrument; the transactions,
// account risk and position balance channels repeat what those report, and the OTC channels serve only quote requests
var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Levels: defaultBookDepth, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: AdvancedOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Channel: PositionsChannel, Authenticated: true},
}

// WebsocketAPI calls Crypto.com's methods over the websocket's trade connection rather than REST; a method with a REST
// form takes that form's name and arguments
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the methods called over the websocket's trade connection
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// websocketRequest is a request sent over the websocket, whose ID its responses echo
type websocketRequest struct {
	// ID is an int64, as the connection matches it with the IDs of the messages Crypto.com sends, which are -1 for pushes
	ID        int64           `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	Nonce     uint64          `json:"nonce,omitempty"`
	APIKey    string          `json:"api_key,omitempty"`
	Signature string          `json:"sig,omitempty"`
}

// websocketResponse is a websocket message's envelope; a subscription's acknowledgement names its channel in Channel,
// and a channel's data comes in Result
type websocketResponse struct {
	Method  string          `json:"method"`
	Code    types.Number    `json:"code"`
	Message string          `json:"message"`
	Channel string          `json:"channel"`
	Result  json.RawMessage `json:"result"`
}

// err returns the error a response reports, or nil when it reports none
func (r *websocketResponse) err() error {
	if r.Code == 0 {
		return nil
	}
	return &APIError{Code: r.Code.Int64(), Message: r.Message}
}

// subscribedChannel returns the channel a subscription's answer names: its channel, else its result's subscription,
// which names the channel of an acknowledgement carrying the channel's first data; an answer refusing the request as a
// whole names none
func (r *websocketResponse) subscribedChannel() string {
	if r.Channel != "" {
		return r.Channel
	}
	channel, err := jsonparser.GetString(r.Result, "subscription")
	if err != nil {
		return ""
	}
	return channel
}

// subscribeInspector inspects the answers to a subscribe request, one for each of its channels, for an answer refusing
// the request as a whole, after which Crypto.com answers none of its channels
type subscribeInspector struct{}

// IsFinal reports whether an answer to a subscribe request refuses the request as a whole
func (subscribeInspector) IsFinal(data []byte) bool {
	var resp websocketResponse
	return json.Unmarshal(data, &resp) == nil && resp.err() != nil && resp.subscribedChannel() == ""
}

// setupWebsocket sets the websocket up with its connections: market data, up to the subscription limit each, the user
// channels connection and the trade connection, which carries the WebsocketAPI methods and is dialled whether or not
// any channel is subscribed
func (e *Exchange) setupWebsocket(exch *config.Exchange) error {
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: websocketMaxSubscriptionsPerConnection,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
	}); err != nil {
		return err
	}
	marketDataURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	userURL, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}
	tradeURL, err := e.API.Endpoints.GetURL(exchange.WebsocketTrade)
	if err != nil {
		return err
	}
	for _, setup := range []*websocket.ConnectionSetup{
		{
			URL:                   marketDataURL,
			Connector:             e.wsConnect,
			GenerateSubscriptions: e.generateMarketDataSubscriptions,
			Subscriber:            e.wsSubscribe,
			Unsubscriber:          e.wsUnsubscribe,
			Handler:               e.wsHandleData,
			MessageFilter:         websocketMarketDataConnection,
			ConnectionRateLimiter: func() *request.RateLimiterWithWeight {
				return request.NewRateLimitWithWeight(time.Second, websocketMarketDataRate, 1)
			},
		},
		{
			URL:                   userURL,
			Connector:             e.wsConnect,
			Authenticate:          e.wsAuthenticate,
			GenerateSubscriptions: e.generateUserSubscriptions,
			Subscriber:            e.wsSubscribe,
			Unsubscriber:          e.wsUnsubscribe,
			Handler:               e.wsHandleData,
			MessageFilter:         websocketUserConnection,
			ConnectionRateLimiter: func() *request.RateLimiterWithWeight {
				return request.NewRateLimitWithWeight(time.Second, websocketUserRate, 1)
			},
		},
		{
			URL:                      tradeURL,
			Connector:                e.wsConnect,
			Authenticate:             e.wsAuthenticate,
			Handler:                  e.wsHandleData,
			MessageFilter:            websocketTradeConnection,
			SubscriptionsNotRequired: true,
			ConnectionRateLimiter: func() *request.RateLimiterWithWeight {
				return request.NewRateLimitWithWeight(time.Second, websocketUserRate, 1)
			},
		},
	} {
		setup.ResponseCheckTimeout = exch.WebsocketResponseCheckTimeout
		setup.ResponseMaxLimit = exch.WebsocketResponseMaxLimit
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// wsConnect dials a connection and waits the connect delay before anything is sent on it
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	if e.websocketConnectDelay <= 0 {
		return nil
	}
	t := time.NewTimer(e.websocketConnectDelay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// wsAuthenticate authenticates a user API connection with a public/auth request, whose signature covers no params
func (e *Exchange) wsAuthenticate(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	id := e.nextWebsocketRequestID()
	nonce := requestNonce()
	signature, err := signRequest(creds.Secret, websocketMethodAuth, uint64(id), creds.Key, "", nonce) //nolint:gosec // Request IDs count up from 1
	if err != nil {
		return err
	}
	req := &websocketRequest{ID: id, Method: websocketMethodAuth, Nonce: nonce, APIKey: creds.Key, Signature: signature}
	data, err := conn.SendMessageReturnResponse(ctx, request.Unset, id, req)
	if err != nil {
		return err
	}
	return decodeWebsocketResponse(data, nil)
}

// SetCancelOnDisconnect has Crypto.com cancel the orders the trade connection placed if it disconnects
func (w WebsocketAPI) SetCancelOnDisconnect(ctx context.Context, scope string) (*CancelOnDisconnectResponse, error) {
	if scope == "" {
		return nil, errScopeRequired
	}
	var resp *CancelOnDisconnectResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/set-cancel-on-disconnect", &cancelOnDisconnectRequest{Scope: scope}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCancelOnDisconnect returns the scope of the trade connection's cancel on disconnect
func (w WebsocketAPI) GetCancelOnDisconnect(ctx context.Context) (*CancelOnDisconnectResponse, error) {
	var resp *CancelOnDisconnectResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/get-cancel-on-disconnect", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SendWebsocketRequest sends a private method's request over the authenticated trade connection, with params, which
// may be nil, as its params object, and decodes its result. A websocket not connected is reported as such before it is
// reported as not authenticated, which follows from it
func (e *Exchange) SendWebsocketRequest(ctx context.Context, method string, params, result any) error {
	conn, err := e.Websocket.GetConnection(websocketTradeConnection)
	if err != nil {
		return err
	}
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: %s", errWebsocketNotAuthenticated, method)
	}
	return e.sendConnectionRequest(ctx, conn, method, params, result)
}

// sendConnectionRequest sends a method's request over conn, with params, which may be nil, as its params object, and
// decodes its result
func (e *Exchange) sendConnectionRequest(ctx context.Context, conn websocket.Connection, method string, params, result any) error {
	encodedParams, err := encodeParams(params)
	if err != nil {
		return err
	}
	id := e.nextWebsocketRequestID()
	req := &websocketRequest{ID: id, Method: method, Params: encodedParams, Nonce: requestNonce()}
	data, err := conn.SendMessageReturnResponse(ctx, request.Unset, id, req)
	if err != nil {
		return err
	}
	return decodeWebsocketResponse(data, result)
}

// decodeWebsocketResponse returns the error a websocket response reports, else decodes its result into result
func decodeWebsocketResponse(data []byte, result any) error {
	var resp websocketResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if err := resp.err(); err != nil {
		return err
	}
	if result == nil || len(resp.Result) == 0 {
		return nil
	}
	return json.Unmarshal(resp.Result, result)
}

// nextWebsocketRequestID returns a new websocket request ID, which a request's responses echo
func (e *Exchange) nextWebsocketRequestID() int64 {
	return e.websocketRequestID.Add(1)
}

// websocketMessageID returns a websocket message's ID, which Crypto.com sends as a JSON number or string; -1, or 0 when
// the message has none, marks a channel's data
func websocketMessageID(data []byte) (int64, error) {
	value, dataType, _, err := jsonparser.Get(data, "id")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("%w: %w", errMalformedWebsocketMessage, err)
	case dataType == jsonparser.Number, dataType == jsonparser.String:
		id, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: id %s: %w", errMalformedWebsocketMessage, value, err)
		}
		return id, nil
	default:
		return 0, fmt.Errorf("%w: id %s", errMalformedWebsocketMessage, value)
	}
}

// wsHandleData answers a heartbeat, routes a response to the request awaiting it and hands a channel's data to its
// handler; a subscription's acknowledgement can carry the channel's first data, so it is handled both ways. A response
// arriving after its request stopped waiting is dropped, as nothing awaits it
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, data []byte) error {
	method, err := jsonparser.GetUnsafeString(data, "method")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return fmt.Errorf("%w: %w", errMalformedWebsocketMessage, err)
	}
	id, err := websocketMessageID(data)
	if err != nil {
		return err
	}
	switch {
	case method == websocketMethodHeartbeat:
		return conn.SendJSONMessage(ctx, request.Unset, &websocketRequest{ID: id, Method: websocketMethodRespondHeartbeat})
	case id > 0:
		conn.IncomingWithData(id, data)
		if method == websocketMethodSubscribe {
			return e.wsHandleChannelData(ctx, conn, data)
		}
		return nil
	case method == websocketMethodSubscribe:
		return e.wsHandleChannelData(ctx, conn, data)
	default:
		return nil
	}
}

// wsHandleChannelData hands a channel's data to the handler of its channel; an acknowledgement carries none
func (e *Exchange) wsHandleChannelData(ctx context.Context, conn websocket.Connection, data []byte) error {
	result, dataType, _, err := jsonparser.Get(data, "result")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), err == nil && dataType == jsonparser.Null:
		return nil
	case err != nil:
		return fmt.Errorf("%w: %w", errMalformedWebsocketMessage, err)
	}
	var push websocketChannelPush
	if err := json.Unmarshal(result, &push); err != nil {
		return fmt.Errorf("%w: %w", errMalformedWebsocketMessage, err)
	}
	switch push.Channel {
	case channelBook:
		return e.wsHandleBook(ctx, &push)
	case channelBookUpdate:
		return e.wsHandleBookUpdate(ctx, conn, &push)
	case channelTicker:
		return e.wsHandleTicker(ctx, &push)
	case channelTrade:
		return e.wsHandleTrades(&push)
	case channelCandlestick:
		return e.wsHandleCandles(ctx, &push)
	case IndexPriceChannel:
		return e.wsHandleIndexPrice(ctx, &push)
	case MarkPriceChannel:
		return e.wsHandleMarkPrice(ctx, &push)
	case SettlementPriceChannel:
		return e.wsHandleSettlementPrice(ctx, &push)
	case FundingRateChannel:
		return e.wsHandleFundingRate(ctx, &push)
	case EstimatedFundingRateChannel:
		return e.wsHandleEstimatedFundingRate(ctx, &push)
	case channelUserOrder, AdvancedOrdersChannel, channelUserAdvancedOrder:
		return e.wsHandleUserOrders(ctx, &push)
	case channelUserTrade:
		return e.wsHandleUserTrades(ctx, &push)
	case TransactionsChannel:
		return e.wsHandleTransactions(ctx, &push)
	case channelUserBalance:
		return e.wsHandleUserBalance(ctx, &push)
	case PositionsChannel:
		return e.wsHandlePositions(ctx, &push)
	case AccountRiskChannel:
		return e.wsHandleAccountRisk(ctx, &push)
	case PositionBalanceChannel:
		return e.wsHandlePositionBalance(ctx, &push)
	case OTCQuoteRequestsChannel:
		return e.wsHandleOTCQuoteRequests(ctx, &push)
	case OTCQuotesChannel:
		return e.wsHandleOTCQuotes(ctx, &push)
	case OTCDealsChannel:
		return e.wsHandleOTCDeals(ctx, &push)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(data)})
}

// decodePushData decodes the data a channel's push holds into v, naming the channel when it cannot, as a decoding error
// names only the type decoded into
func decodePushData(push *websocketChannelPush, v any) error {
	if err := json.Unmarshal(push.Data, v); err != nil {
		return fmt.Errorf("%w: %s data: %w", errMalformedWebsocketMessage, push.Channel, err)
	}
	return nil
}

// GetSubscriptionTemplate returns a template naming the channel of each pair, of each asset, a market data subscription
// covers, or the user channels an authenticated subscription covers
func (e *Exchange) GetSubscriptionTemplate(s *subscription.Subscription) (*template.Template, error) {
	if err := common.NilGuard(s); err != nil {
		return nil, err
	}
	if s.Authenticated {
		return template.New("user.tmpl").Funcs(template.FuncMap{
			"userChannel":             userChannel,
			"userChannelByInstrument": userChannelByInstrument,
			"userInstrumentChannel":   e.userInstrumentChannel,
		}).Parse(userSubscriptionTemplate)
	}
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelServesAsset": channelServesAsset,
		"qualifiedChannel":   e.qualifiedChannel,
	}).Parse(subscriptionTemplate)
}

// userSubscriptionTemplate names a user channel: an order or trade channel for each pair of the asset a subscription
// names, else the account's channel once
const userSubscriptionTemplate = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- if userChannelByInstrument $.S $asset }}
		{{- range $pair := $pairs }}
			{{- userInstrumentChannel $.S $asset $pair }}
			{{- $.PairSeparator }}
		{{- end }}
	{{- else }}
		{{- userChannel $.S }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}`

// subscriptionTemplate names a channel for each pair of each asset a subscription's channel serves
const subscriptionTemplate = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- if channelServesAsset $.S $asset }}
		{{- range $pair := $pairs }}
			{{- qualifiedChannel $.S $asset $pair }}
			{{- $.PairSeparator }}
		{{- end }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}`

// channelServesAsset reports whether a subscription's channel serves an asset, erring for a channel Crypto.com does not
// serve, and for an asset the subscription names that its channel does not serve
func channelServesAsset(s *subscription.Subscription, a asset.Item) (bool, error) {
	var serves bool
	switch s.Channel {
	case subscription.TickerChannel, subscription.OrderbookChannel, subscription.AllTradesChannel, subscription.CandlesChannel, MarkPriceChannel:
		serves = a == asset.Spot || a == asset.Margin || a == asset.PerpetualSwap || a == asset.Futures
	case IndexPriceChannel:
		serves = a == asset.PerpetualSwap || a == asset.Futures
	case FundingRateChannel, EstimatedFundingRateChannel:
		serves = a == asset.PerpetualSwap
	case SettlementPriceChannel:
		serves = a == asset.Futures
	default:
		return false, fmt.Errorf("%w: %q", subscription.ErrNotSupported, s.Channel)
	}
	if !serves && s.Asset != asset.All {
		return false, fmt.Errorf("%w: %s channel for asset %q", subscription.ErrNotSupported, s.Channel, a)
	}
	return serves, nil
}

// qualifiedChannel returns the channel Crypto.com names for a subscription's channel to a pair of an asset
func (e *Exchange) qualifiedChannel(s *subscription.Subscription, a asset.Item, p currency.Pair) (string, error) {
	instrument, err := e.FormatSymbol(p, a)
	if err != nil {
		return "", err
	}
	switch s.Channel {
	case subscription.TickerChannel:
		return channelTicker + "." + instrument, nil
	case subscription.OrderbookChannel:
		if _, _, err := bookUpdates(s); err != nil {
			return "", err
		}
		depth, err := bookDepth(s)
		if err != nil {
			return "", err
		}
		return channelBook + "." + instrument + "." + strconv.Itoa(depth), nil
	case subscription.AllTradesChannel:
		return channelTrade + "." + instrument, nil
	case subscription.CandlesChannel:
		timeframe, err := websocketCandleTimeframe(s.Interval)
		if err != nil {
			return "", err
		}
		return channelCandlestick + "." + timeframe + "." + instrument, nil
	case IndexPriceChannel:
		// A derivative's name is its underlying, a dash and its expiry or PERP, and the name of its underlying index, which
		// get-instruments gives as underlying_symbol, is its underlying followed by -INDEX
		i := strings.LastIndexByte(instrument, '-')
		if i <= 0 {
			return "", fmt.Errorf("%w: %s", errNoUnderlyingIndex, instrument)
		}
		return IndexPriceChannel + "." + instrument[:i] + underlyingIndexSuffix, nil
	case MarkPriceChannel, SettlementPriceChannel, FundingRateChannel, EstimatedFundingRateChannel:
		return s.Channel + "." + instrument, nil
	default:
		return "", fmt.Errorf("%w: %q", subscription.ErrNotSupported, s.Channel)
	}
}

// bookDepth returns the depth of a book subscription: the levels it names, which Crypto.com serves 10 or 50 of, else 50
func bookDepth(s *subscription.Subscription) (int, error) {
	switch s.Levels {
	case 0:
		return defaultBookDepth, nil
	case 10, 50:
		return s.Levels, nil
	default:
		return 0, fmt.Errorf("%w %d: Crypto.com serves books 10 or 50 levels deep", subscription.ErrInvalidLevel, s.Levels)
	}
}

// bookUpdates returns the book_subscription_type and book_update_frequency a book subscription is requested with, by
// the interval it names: deltas every 10 or 100 milliseconds, by default 100, or a full snapshot every 500
func bookUpdates(s *subscription.Subscription) (subscriptionType string, frequency uint64, err error) {
	switch s.Interval {
	case 0, kline.HundredMilliseconds:
		return BookSubscriptionTypeSnapshotAndUpdate, 100, nil
	case kline.TenMilliseconds:
		return BookSubscriptionTypeSnapshotAndUpdate, 10, nil
	case bookSnapshotInterval:
		return BookSubscriptionTypeSnapshot, 500, nil
	default:
		return "", 0, fmt.Errorf("%w %s: Crypto.com sends book deltas every 10 or 100 milliseconds, or snapshots every 500", subscription.ErrInvalidInterval, s.Interval)
	}
}

// generateMarketDataSubscriptions returns the subscriptions the market data connections carry, each of Crypto.com's
// channels once; each push is relayed to every asset its instrument trades as
func (e *Exchange) generateMarketDataSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions(false)
	if err != nil {
		return nil, err
	}
	return distinctChannels(subs), nil
}

// generateUserSubscriptions returns the subscriptions the user channels connection carries, each of Crypto.com's
// channels once; expanding them leaves them all out unless the connection can be authenticated
func (e *Exchange) generateUserSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions(true)
	if err != nil {
		return nil, err
	}
	return distinctChannels(subs), nil
}

// distinctChannels returns subs with each of Crypto.com's channels subscribed once: spot and margin trade the same
// instruments, so their subscriptions name the same channels, the account channels are the same for every asset, and an
// instrument has the one book, so the first of its book subscriptions is kept whatever its depth or updates
func distinctChannels(subs subscription.List) subscription.List {
	distinct := make(subscription.List, 0, len(subs))
	channels := make(map[string]struct{}, len(subs))
	for _, s := range subs {
		channel := s.QualifiedChannel
		if i := strings.LastIndexByte(channel, '.'); s.Channel == subscription.OrderbookChannel && i > 0 {
			// A book channel ends with its depth, after a dot, which no instrument name holds
			channel = channel[:i]
		}
		if _, ok := channels[channel]; !ok {
			channels[channel] = struct{}{}
			distinct = append(distinct, s)
		}
	}
	return distinct
}

// generateSubscriptions returns the configured subscriptions with their channels qualified, the user channels when
// authenticated is set and the market data channels otherwise; each kind is expanded apart, so that a subscription the
// one kind's connections cannot serve does not stop the other's
func (e *Exchange) generateSubscriptions(authenticated bool) (subscription.List, error) {
	configured := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, s := range e.Features.Subscriptions {
		if s.Authenticated == authenticated {
			configured = append(configured, s)
		}
	}
	subs, err := configured.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

// subscriptionRequest is the params of a subscribe or unsubscribe request
type subscriptionRequest struct {
	Channels             []string `json:"channels"`
	BookSubscriptionType string   `json:"book_subscription_type,omitempty"`
	BookUpdateFrequency  uint64   `json:"book_update_frequency,omitempty"`
}

// wsSubscribe subscribes to channels, with requests for each book_subscription_type and book_update_frequency their
// subscriptions need, as Crypto.com applies a request's to every book channel it names, each naming as many as it holds
func (e *Exchange) wsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	type batch struct {
		params *subscriptionRequest
		subs   subscription.List
	}
	var batches []*batch
	var errs error
	for _, s := range subs {
		var params subscriptionRequest
		if s.Channel == subscription.OrderbookChannel {
			var err error
			if params.BookSubscriptionType, params.BookUpdateFrequency, err = bookUpdates(s); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", s.QualifiedChannel, err))
				continue
			}
		}
		i := slices.IndexFunc(batches, func(b *batch) bool {
			return b.params.BookSubscriptionType == params.BookSubscriptionType && b.params.BookUpdateFrequency == params.BookUpdateFrequency
		})
		if i == -1 {
			i = len(batches)
			batches = append(batches, &batch{params: &params})
		}
		batches[i].subs = append(batches[i].subs, s)
	}
	for _, b := range batches {
		for _, list := range channelLists(b.subs) {
			params := *b.params
			params.Channels = qualifiedChannels(list)
			errs = common.AppendError(errs, e.wsSubscribeBatch(ctx, conn, &params, list))
		}
	}
	return errs
}

// channelLists splits subscriptions into the lists one subscribe or unsubscribe request holds
func channelLists(subs subscription.List) []subscription.List {
	var lists []subscription.List
	var start, channelBytes int
	for i, s := range subs {
		size := len(s.QualifiedChannel) + len(`"",`)
		if channelBytes+size > websocketMaxChannelBytes {
			lists = append(lists, subs[start:i])
			start, channelBytes = i, 0
		}
		channelBytes += size
	}
	return append(lists, subs[start:])
}

// qualifiedChannels returns the channels subscriptions name
func qualifiedChannels(subs subscription.List) []string {
	channels := make([]string, len(subs))
	for i, s := range subs {
		channels[i] = s.QualifiedChannel
	}
	return channels
}

// wsSubscribeBatch sends a subscribe request for channels and waits for each channel's answer, or for an answer refusing
// the request as a whole, keeping each subscription answered without an error
func (e *Exchange) wsSubscribeBatch(ctx context.Context, conn websocket.Connection, params *subscriptionRequest, subs subscription.List) error {
	byChannel := make(map[string]*subscription.Subscription, len(subs))
	for _, s := range subs {
		byChannel[s.QualifiedChannel] = s
	}
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	id := e.nextWebsocketRequestID()
	req := &websocketRequest{ID: id, Method: websocketMethodSubscribe, Params: encodedParams, Nonce: requestNonce()}
	responses, err := conn.SendMessageReturnResponsesWithInspector(ctx, request.Unset, id, req, len(params.Channels), subscribeInspector{})
	if err != nil {
		return err
	}
	var errs error
	for _, data := range responses {
		var resp websocketResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		channel := resp.subscribedChannel()
		if err := resp.err(); err != nil {
			if channel == "" {
				// Crypto.com refused the request as a whole, answering none of its channels
				return common.AppendError(errs, fmt.Errorf("subscribing %d channels: %w", len(params.Channels), err))
			}
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", channel, err))
			continue
		}
		if channel == "" {
			errs = common.AppendError(errs, fmt.Errorf("%w: subscription acknowledgement names no channel", errMalformedWebsocketMessage))
			continue
		}
		s, ok := byChannel[channel]
		if !ok {
			continue
		}
		delete(byChannel, channel)
		if err := e.Websocket.AddSuccessfulSubscriptions(conn, s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// wsUnsubscribe unsubscribes from channels, with as many requests as they take
func (e *Exchange) wsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	if len(subs) == 0 {
		return nil
	}
	var errs error
	for _, list := range channelLists(subs) {
		errs = common.AppendError(errs, e.wsUnsubscribeBatch(ctx, conn, list))
	}
	return errs
}

// wsUnsubscribeBatch sends an unsubscribe request for channels, which Crypto.com acknowledges once, and removes them
func (e *Exchange) wsUnsubscribeBatch(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	if err := e.sendConnectionRequest(ctx, conn, websocketMethodUnsubscribe, &subscriptionRequest{Channels: qualifiedChannels(subs)}, nil); err != nil {
		return err
	}
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}
