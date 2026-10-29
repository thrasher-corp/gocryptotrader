package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	// websocketPingInterval follows Bybit's recommendation; without pings, the public streams close a connection that
	// receives no data after a minute, and the option stream after about 30 seconds
	websocketPingInterval = 20 * time.Second

	spotPublic       = "wss://stream.bybit.com/v5/public/spot"
	linearPublic     = "wss://stream.bybit.com/v5/public/linear"  // USDT and USDC perpetuals and futures
	inversePublic    = "wss://stream.bybit.com/v5/public/inverse" // Inverse perpetuals and futures
	optionPublic     = "wss://stream.bybit.com/v5/public/option"  // USDT and USDC options
	websocketPrivate = "wss://stream.bybit.com/v5/private"
	websocketTrade   = "wss://stream.bybit.com/v5/trade"

	// mainnetStreamOrigin serves the system status, Advanced Earn, spread, RFQ and event streams the regional hosts lack,
	// and testnetStreamOrigin those their testnet hosts lack
	mainnetStreamOrigin = "wss://stream.bybit.com"
	testnetStreamOrigin = "wss://stream-testnet.bybit.com"

	// euStreamHost is the stream host of EU accounts, which Bybit's docs give a system status stream of their own
	euStreamHost = "stream.bybit.eu"

	// subscriptionBatchSize is the most topics spot accepts in one request; the other streams accept more, but share the
	// limit so a rejected request costs at most ten topics a resend
	subscriptionBatchSize = 10

	// publicConnectionArgsLimit is the most args, or topics, Bybit lets one connection to the spot or futures public
	// streams subscribe to from 26 Nov 2026; the option stream allows 2000 and the spread stream any number
	publicConnectionArgsLimit = 100

	subscribeOperation   = "subscribe"
	unsubscribeOperation = "unsubscribe"
	authOperation        = "auth"

	// commandResponseType marks option and system status replies, which echo no request ID and so are matched on it
	commandResponseType = "COMMAND_RESP"

	snapshotType = "snapshot"
)

// Public, private and system topic names
const (
	chanOrderbook      = "orderbook"
	chanRPIOrderbook   = "orderbook.rpi"
	chanFullOrderbook  = "orderbook.full"
	chanPublicTrade    = "publicTrade"
	chanPublicTicker   = "tickers"
	chanKline          = "kline"
	chanAllLiquidation = "allLiquidation"
	chanInsurancePool  = "insurance"
	chanPriceLimit     = "priceLimit"
	chanADLAlert       = "adlAlert"

	chanPositions     = "position"
	chanExecution     = "execution"
	chanFastExecution = "execution.fast"
	chanOrder         = "order"
	chanWallet        = "wallet"
	chanGreeks        = "greeks"
	chanStrategy      = "strategy"

	chanSystemStatus = "system.status"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    chanPublicTicker,
	subscription.OrderbookChannel: chanOrderbook,
	subscription.AllTradesChannel: chanPublicTrade,
	subscription.MyOrdersChannel:  chanOrder,
	subscription.MyWalletChannel:  chanWallet,
	subscription.MyTradesChannel:  chanExecution,
	subscription.CandlesChannel:   chanKline,
}

// Errors from subscribing, authenticating and handling pushes on the websocket streams
var (
	errUnhandledStreamData        = errors.New("unhandled stream data")
	errUnsupportedCategory        = errors.New("unsupported category")
	errWebsocketAuthFailed        = errors.New("websocket authentication failed")
	errContractGroupNotSupported  = errors.New("topic has no contract group for asset")
	errOptionTradesNotOneBaseCoin = errors.New("option trade subscriptions must share one base coin")
	errNoStreamOrigin             = errors.New("websocket URL has no scheme and host")
	errOrderbookDepthConflict     = errors.New("orderbook subscriptions of a symbol must share one depth")
)

// commandResponseMu lets one option or system status request at a time wait for its reply, since the replies carry no
// request ID to tell them apart
var commandResponseMu sync.Mutex

// streamOrigin returns the scheme and host of a websocket URL
func streamOrigin(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errNoStreamOrigin, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: %q", errNoStreamOrigin, rawURL)
	}
	return u.Scheme + "://" + u.Host, nil
}

// regionalStreamHosts are the stream hosts of Bybit's regional sites: those its docs give accounts registered with them,
// and the UAE site's, whose REST host api.bybit.ae the docs name. They serve the spot, futures and option streams, but
// answer 404 for the public streams without a configurable endpoint, apart from the EU host's system status stream.
// Their stream-testnet hosts answer most of those with 504 Gateway Timeout
var regionalStreamHosts = []string{
	"stream.bybit.ae", euStreamHost, "stream.bybit.id", "stream.bybit.kz", "stream.bybit.tr", "stream.bybitgeorgia.ge",
	"stream.manepa.jp", "stream.spark-fintech.com",
}

// auxiliaryStreamURL returns the URL of the public stream at path, one without a configurable endpoint. It follows the
// configured spot stream to testnet or another server, but stays on mainnet when that is one of Bybit's regional hosts,
// and on testnet when it is one of their testnet hosts, apart from the EU hosts' system status stream
func auxiliaryStreamURL(spotURL, path string) (string, error) {
	origin, err := streamOrigin(spotURL)
	if err != nil {
		return "", err
	}
	if u, err := url.Parse(origin); err == nil {
		host, fallback := strings.ToLower(u.Hostname()), mainnetStreamOrigin
		if site, ok := strings.CutPrefix(host, "stream-testnet."); ok {
			host, fallback = "stream."+site, testnetStreamOrigin
		}
		if slices.Contains(regionalStreamHosts, host) && (host != euStreamHost || path != systemStatusPath) {
			return fallback + path, nil
		}
	}
	return origin + path, nil
}

// WebsocketConnect connects to a websocket stream and starts its heartbeat
func (e *Exchange) WebsocketConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(wsSubscriptionEPL, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op":"ping"}`),
		Delay:       websocketPingInterval,
	})
	return nil
}

// WebsocketAuthenticatePrivateConnection authenticates the private stream connection (Connect: Authentication)
func (e *Exchange) WebsocketAuthenticatePrivateConnection(ctx context.Context, conn websocket.Connection) error {
	req, err := e.GetAuthenticationPayload(ctx, e.MessageID())
	if err != nil {
		return err
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
	if err != nil {
		return err
	}
	var resp *SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%w: %w", errWebsocketAuthFailed, common.ErrNoResponse)
	}
	if !resp.Success {
		return fmt.Errorf("%w: %s", errWebsocketAuthFailed, resp.ReturnMessage)
	}
	return nil
}

// GetAuthenticationPayload returns the request that authenticates a private stream connection (Connect: Authentication)
func (e *Exchange) GetAuthenticationPayload(ctx context.Context, requestID string) (*AuthenticationRequest, error) {
	args, err := e.websocketAuthenticationArguments(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthenticationRequest{RequestID: requestID, Operation: authOperation, Arguments: args}, nil
}

// websocketAuthenticationArguments returns the API key, expiry and signature that authenticate a private stream or
// trade connection
func (e *Exchange) websocketAuthenticationArguments(ctx context.Context) ([]any, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(time.Hour * 6).UnixMilli()
	return []any{creds.Key, expires, getSign("GET/realtime"+strconv.FormatInt(expires, 10), creds.Secret)}, nil
}

// sendSubscriptions sends subscribe or unsubscribe requests for subs in batches and records the subscriptions Bybit
// accepts, and those sharing a topic another subscription holds without a request. Bybit rejects a whole request when
// one of its topics is invalid and names that topic, so a rejected batch is resent without the topics it names and the
// batches after it are still sent; every rejected topic is reported in the returned error
func (e *Exchange) sendSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	subs, marks, errs := e.shareTopics(conn, operation, subs)
	defer e.keepHolders(marks)
	for _, batch := range topicBatches(subs, subscriptionBatchSize) {
		errs = common.AppendError(errs, e.sendSubscriptionBatch(ctx, conn, operation, batch))
	}
	return errs
}

func (e *Exchange) sendSubscriptionBatch(ctx context.Context, conn websocket.Connection, operation string, batch subscription.List) error {
	var errs error
	for len(batch) != 0 {
		req := &SubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: uniqueTopics(batch)}
		respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
		if err != nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err))
		}
		var resp *SubscriptionResponse
		if err := json.Unmarshal(respRaw, &resp); err != nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err))
		}
		if resp == nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), common.ErrNoResponse))
		}
		if resp.Success {
			return common.AppendError(errs, e.recordSubscriptions(conn, operation, batch))
		}
		var rejected subscription.List
		rejected, batch = splitRejectedSubscriptions(batch, resp.ReturnMessage)
		errs = common.AppendError(errs, fmt.Errorf("%w: %s %s rejected: %s", websocket.ErrSubscriptionFailure, operation, strings.Join(uniqueTopics(rejected), ","), resp.ReturnMessage))
	}
	return errs
}

// splitRejectedSubscriptions returns the subscriptions whose topics a rejection message names, and the rest. A message
// that names none of the topics rejects them all
func splitRejectedSubscriptions(batch subscription.List, message string) (rejected, remaining subscription.List) {
	named := strings.FieldsFunc(message, func(r rune) bool {
		return r == '[' || r == ']' || r == ',' || r == ':' || unicode.IsSpace(r)
	})
	for _, s := range batch {
		if slices.Contains(named, s.QualifiedChannel) {
			rejected = append(rejected, s)
		} else {
			remaining = append(remaining, s)
		}
	}
	if len(rejected) == 0 {
		return batch, nil
	}
	return rejected, remaining
}

// sendCommandSubscriptions sends subscribe or unsubscribe requests for subs on the option and system status streams,
// whose replies list the topics applied and rejected, and records those sharing a topic another subscription holds
// without a request
func (e *Exchange) sendCommandSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	subs, marks, errs := e.shareTopics(conn, operation, subs)
	defer e.keepHolders(marks)
	for _, batch := range topicBatches(subs, subscriptionBatchSize) {
		errs = common.AppendError(errs, e.sendCommandSubscriptionBatch(ctx, conn, operation, batch))
	}
	return errs
}

func (e *Exchange) sendCommandSubscriptionBatch(ctx context.Context, conn websocket.Connection, operation string, batch subscription.List) error {
	req := &SubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: uniqueTopics(batch)}
	commandResponseMu.Lock()
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, commandResponseType, req)
	commandResponseMu.Unlock()
	if err != nil {
		return fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err)
	}
	var resp *SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err)
	}
	if resp == nil {
		return fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), common.ErrNoResponse)
	}
	var accepted, rejected subscription.List
	for _, s := range batch {
		if !resp.Success || slices.Contains(resp.Data.FailTopics, s.QualifiedChannel) {
			rejected = append(rejected, s)
		} else {
			accepted = append(accepted, s)
		}
	}
	var errs error
	if len(accepted) != 0 {
		errs = e.recordSubscriptions(conn, operation, accepted)
	}
	if len(rejected) != 0 {
		errs = common.AppendError(errs, fmt.Errorf("%w: %s %s rejected", websocket.ErrSubscriptionFailure, operation, strings.Join(uniqueTopics(rejected), ",")))
	}
	return errs
}

// recordSubscriptions stores subscriptions Bybit subscribed, recording the connection they hold their topics on, or
// removes ones it unsubscribed
func (e *Exchange) recordSubscriptions(conn websocket.Connection, operation string, subs subscription.List) error {
	e.topicMu.Lock()
	defer e.topicMu.Unlock()
	return e.recordSubscriptionsLocked(conn, operation, subs)
}

// recordSubscriptionsLocked is recordSubscriptions for a caller holding topicMu
func (e *Exchange) recordSubscriptionsLocked(conn websocket.Connection, operation string, subs subscription.List) error {
	if operation == unsubscribeOperation {
		// The manager may still store a holder subscribed on another connection too, so its record here is dropped now
		// rather than when the manager no longer stores it. A key naming several holders leaves which one the manager
		// removes to the manager, and that one's record to holdersLocked
		for _, s := range subs {
			var named []*subscription.Subscription
			for h, holder := range e.topicHolders {
				if holder.conn == conn && namesSubscription(s, h) {
					named = append(named, h)
				}
			}
			if len(named) == 1 {
				delete(e.topicHolders, named[0])
			}
		}
		return e.Websocket.RemoveSubscriptions(conn, subs...)
	}
	if e.topicHolders == nil {
		e.topicHolders = make(map[*subscription.Subscription]topicHolder, len(subs))
	}
	for _, s := range subs {
		e.topicHolders[s] = topicHolder{conn: conn}
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// topicHolder is the connection a subscription holds its topic on, and the requests unsubscribing it in flight, while
// which other requests do not count it
type topicHolder struct {
	conn    websocket.Connection
	leaving []uint64
}

// topicMarks are the holders a request marked leaving, the request's own mark, and the orderbook depths it asks for
type topicMarks struct {
	request uint64
	holders subscription.List
	depths  []bookDepth
}

// bookDepth is an orderbook depth of a symbol on a stream, every connection to which loads the symbol's one book
type bookDepth struct {
	stream, symbol, depth string
}

// shareTopics records at once the subscriptions in subs whose topic another subscription on the connection holds, and
// returns the rest to be sent, with the leaving marks it set for keepHolders. Bybit ends a topic on its first
// unsubscribe, refuses a second subscribe on the linear and inverse streams and leaves it unanswered on the spot
// stream, so a topic that subscriptions share, as option trade subscriptions for one base coin do, is subscribed with
// the first of them and unsubscribed with the last; resubscribing one of them sends nothing while another holds the
// topic. Each subscription unsubscribed takes the holder it names, itself or an equal copy, off its topic, so
// subscriptions unsubscribed together leave their topic to be unsubscribed, and a holder whose unsubscribe is sent is
// marked leaving, so that a request made while it is in flight does not count it. Requests are not held back for one
// another, so two subscribes of a topic sent at once can still both go out. Every orderbook depth of a symbol loads the
// one book the symbol has, so a subscribe to a depth is refused while a connection to the stream holds another depth
// of the symbol, even one being unsubscribed, which can fail, a request in flight asks for one, or an earlier
// subscription in the request does
func (e *Exchange) shareTopics(conn websocket.Connection, operation string, subs subscription.List) (pending subscription.List, marks topicMarks, err error) {
	e.topicMu.Lock()
	defer e.topicMu.Unlock()
	holders := e.holdersLocked(conn)
	held := make(map[string]int, len(holders))
	for _, h := range holders {
		if len(e.topicHolders[h].leaving) == 0 {
			held[h.QualifiedChannel]++
		}
	}
	var stream string
	var depths map[string][]string // The orderbook depths of each symbol the stream holds or requests ask for
	if operation == subscribeOperation && slices.ContainsFunc(subs, func(s *subscription.Subscription) bool {
		_, _, ok := orderbookDepthTopic(s.QualifiedChannel)
		return ok
	}) {
		stream = conn.GetURL()
		depths = e.bookDepthsLocked(stream)
	}
	named := make([]*subscription.Subscription, len(subs))
	if operation == unsubscribeOperation {
		for i, s := range subs {
			j := slices.IndexFunc(holders, func(h *subscription.Subscription) bool { return namesSubscription(s, h) })
			if j < 0 {
				continue
			}
			named[i] = holders[j]
			if len(e.topicHolders[named[i]].leaving) == 0 {
				held[named[i].QualifiedChannel]--
			}
			holders = slices.Delete(holders, j, j+1)
		}
	}
	e.topicRequests++
	marks.request = e.topicRequests
	var shared subscription.List
	for i, s := range subs {
		if depth, symbol, ok := orderbookDepthTopic(s.QualifiedChannel); ok && operation == subscribeOperation {
			if j := slices.IndexFunc(depths[symbol], func(d string) bool { return d != depth }); j >= 0 {
				err = common.AppendError(err, fmt.Errorf("%w: %w: %s conflicts with depth %s", websocket.ErrSubscriptionFailure, errOrderbookDepthConflict, s.QualifiedChannel, depths[symbol][j]))
				continue
			}
			depths[symbol] = []string{depth}
		}
		if held[s.QualifiedChannel] > 0 {
			shared = append(shared, s)
			continue
		}
		pending = append(pending, s)
		if depth, symbol, ok := orderbookDepthTopic(s.QualifiedChannel); ok && operation == subscribeOperation {
			d := bookDepth{stream: stream, symbol: symbol, depth: depth}
			if e.bookDepths == nil {
				e.bookDepths = make(map[bookDepth]int)
			}
			e.bookDepths[d]++
			marks.depths = append(marks.depths, d)
		}
		if h := named[i]; h != nil {
			holder := e.topicHolders[h]
			holder.leaving = append(holder.leaving, marks.request)
			e.topicHolders[h] = holder
			marks.holders = append(marks.holders, h)
		}
	}
	if len(shared) != 0 {
		err = common.AppendError(err, e.recordSubscriptionsLocked(conn, operation, shared))
	}
	return pending, marks, err
}

// orderbookDepthTopic returns the depth and symbol of an orderbook topic, orderbook.{depth}.{symbol}, whose pushes are
// loaded into the symbol's book, and false for any other topic, such as the RPI and full depth orderbook topics
func orderbookDepthTopic(topic string) (depth, symbol string, ok bool) {
	name, rest, _ := strings.Cut(topic, ".")
	if name != chanOrderbook {
		return "", "", false
	}
	depth, symbol, _ = strings.Cut(rest, ".")
	if _, err := strconv.ParseUint(depth, 10, 64); err != nil || symbol == "" {
		return "", "", false
	}
	return depth, symbol, true
}

// bookDepthsLocked returns the orderbook depths of each symbol that subscriptions on the connections to stream hold,
// those being unsubscribed included, and that requests in flight on them ask for. topicMu must be held
func (e *Exchange) bookDepthsLocked(stream string) map[string][]string {
	depths := make(map[string][]string)
	add := func(symbol, depth string) {
		if !slices.Contains(depths[symbol], depth) {
			depths[symbol] = append(depths[symbol], depth)
		}
	}
	for s, h := range e.topicHolders {
		if depth, symbol, ok := orderbookDepthTopic(s.QualifiedChannel); ok && h.conn.GetURL() == stream {
			add(symbol, depth)
		}
	}
	for d := range e.bookDepths {
		if d.stream == stream {
			add(d.symbol, d.depth)
		}
	}
	return depths
}

// keepHolders clears a request's leaving marks once its unsubscriptions are over, so a holder whose unsubscription
// failed, which the websocket manager still stores, is counted again unless another request is unsubscribing it, and
// releases the orderbook depths the request asked for, which the subscriptions it made now hold
func (e *Exchange) keepHolders(marks topicMarks) {
	if len(marks.holders) == 0 && len(marks.depths) == 0 {
		return
	}
	e.topicMu.Lock()
	defer e.topicMu.Unlock()
	for _, d := range marks.depths {
		e.bookDepths[d]--
		if e.bookDepths[d] <= 0 {
			delete(e.bookDepths, d)
		}
	}
	for _, s := range marks.holders {
		h, ok := e.topicHolders[s]
		if !ok {
			continue
		}
		if i := slices.Index(h.leaving, marks.request); i >= 0 {
			h.leaving = slices.Delete(h.leaving, i, i+1)
			e.topicHolders[s] = h
		}
	}
}

// holdersLocked returns the subscriptions recorded on conn that the websocket manager stores, and drops the records of
// those it no longer stores, as after a reconnect. topicMu must be held
func (e *Exchange) holdersLocked(conn websocket.Connection) subscription.List {
	if len(e.topicHolders) == 0 {
		return nil
	}
	stored := make(map[*subscription.Subscription]bool)
	for _, s := range e.Websocket.GetSubscriptions() {
		stored[s] = true
	}
	var holders subscription.List
	for s, h := range e.topicHolders {
		switch {
		case !stored[s]:
			delete(e.topicHolders, s)
		case h.conn == conn:
			holders = append(holders, s)
		}
	}
	return holders
}

// namesSubscription reports whether s names the stored subscription h as the websocket manager's store looks one up:
// by being h, or by a key matching h's. It runs under topicMu, so a custom key's Match must not subscribe or
// unsubscribe, as it must not where the manager's stores call it under their own locks
func namesSubscription(s, h *subscription.Subscription) bool {
	if s == h {
		return true
	}
	key := s.EnsureKeyed()
	if k, ok := key.(subscription.MatchableKey); ok {
		stored, ok := h.EnsureKeyed().(subscription.MatchableKey)
		return ok && k.Match(stored)
	}
	return key == h.EnsureKeyed()
}

// topicBatches splits subs into batches of at most size topics, keeping the subscriptions that share a topic in one
// batch so that each topic is sent once
func topicBatches(subs subscription.List, size int) []subscription.List {
	var groups []subscription.List
	group := make(map[string]int, len(subs))
	for _, s := range subs {
		i, ok := group[s.QualifiedChannel]
		if !ok {
			i = len(groups)
			group[s.QualifiedChannel] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], s)
	}
	batches := common.Batch(groups, size)
	merged := make([]subscription.List, len(batches))
	for i, batch := range batches {
		merged[i] = slices.Concat(batch...)
	}
	return merged
}

// uniqueTopics returns the topics of subs once each, in order; option trade subscriptions for one base coin share a topic
func uniqueTopics(subs subscription.List) []string {
	topics := make([]string, 0, len(subs))
	for _, s := range subs {
		if !slices.Contains(topics, s.QualifiedChannel) {
			topics = append(topics, s.QualifiedChannel)
		}
	}
	return topics
}

// handleNoTopicWebsocketResponse routes replies to subscribe, unsubscribe, auth and ping requests
func (e *Exchange) handleNoTopicWebsocketResponse(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var resp *SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	switch {
	case resp == nil:
		// A null frame answers no request, so it is relayed below as an unknown frame
	case resp.Type == commandResponseType:
		if !conn.IncomingWithData(commandResponseType, respRaw) && len(resp.Data.FailTopics) != 0 {
			return fmt.Errorf("%w: %s rejected", websocket.ErrSubscriptionFailure, strings.Join(resp.Data.FailTopics, ","))
		}
		return nil
	case resp.Operation == subscribeOperation, resp.Operation == unsubscribeOperation, resp.Operation == authOperation:
		if resp.RequestID == "" {
			return nil
		}
		return conn.RequireMatchWithData(resp.RequestID, respRaw)
	case resp.Operation == "ping", resp.Operation == "pong":
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: string(respRaw)})
}

// generateSubscriptions returns the configured spot subscriptions for the spot connection; each other public connection
// generates its own
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	var configured subscription.List
	for _, s := range e.Features.Subscriptions {
		if (s.Asset == asset.Spot || s.Asset == asset.All) && !dedicatedConnectionChannel(s.Channel) {
			configured = append(configured, s)
		}
	}
	expanded, err := configured.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	subs := make(subscription.List, 0, len(expanded))
	for _, s := range expanded {
		if s.Asset == asset.Spot {
			subs = append(subs, s)
		}
	}
	return subs, nil
}

// dedicatedConnectionChannel reports whether a channel belongs to a stream other than spot's: the spread, event, RFQ,
// Advanced Earn and system status streams, public or private
func dedicatedConnectionChannel(channel string) bool {
	switch channel {
	case ChannelSpreadOrderbook, ChannelSpreadPublicTrade, ChannelSpreadTicker, ChannelSpreadOrder, ChannelSpreadExecution,
		ChannelEventOrderbook, ChannelEventPublicTrade, ChannelEventPosition, ChannelEventExecution, ChannelEventOrder,
		ChannelRFQPublicTrades, ChannelRFQQuotes, ChannelRFQTrades, ChannelRFQInquiries,
		AdvancedEarnDualAssetsOffersChannel, AdvancedEarnSmartLeverageOffersChannel, AdvancedEarnDiscountBuyOffersChannel,
		AdvancedEarnDoubleWinOffersChannel, chanSystemStatus:
		return true
	}
	return false
}

// GetSubscriptionTemplate returns the template that expands spot subscriptions into topics
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isSymbolChannel":  isSymbolChannel,
		"intervalToString": intervalToString,
		"orderbookLevels":  orderbookLevels,
	}).Parse(subTplText)
}

// channelName converts global channel names to exchange specific names
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	return s.Channel
}

// isSymbolChannel returns whether the channel accepts a symbol parameter
func isSymbolChannel(name string) bool {
	switch name {
	case chanPositions, chanExecution, chanOrder, chanWallet, chanInsurancePool, chanADLAlert, chanSystemStatus:
		return false
	}
	return true
}

const subTplText = `
{{ with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if isSymbolChannel $name }}
			{{- range $p := $pairs }}
				{{- $name -}} .
				{{- if eq $name "orderbook" -}} {{- orderbookLevels $.S $asset -}} . {{- end }}
				{{- if eq $name "kline" -}} {{- intervalToString $.S.Interval -}} . {{- end }}
				{{- $p }}
				{{- $.PairSeparator }}
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- end }}
`

// symbol returns the symbol Bybit names a pair by for an asset
func (e *Exchange) symbol(p currency.Pair, a asset.Item) (string, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return "", err
	}
	// USDC futures are named with a dash, unlike USDC perpetuals, which the request format serves
	if a == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
		pairFmt.Delimiter = currency.DashDelimiter
	}
	return pairFmt.Format(p), nil
}

// matchPairAssetFromResponse returns the currency pair and asset type based on the category and symbol. Used with a dedicated
// auth connection where multiple asset type changes are piped through a single connection.
func (e *Exchange) matchPairAssetFromResponse(category, symbol string) (currency.Pair, asset.Item, error) {
	assets := make([]asset.Item, 0, 2)
	switch category {
	case cSpot:
		assets = append(assets, asset.Spot)
	case cInverse:
		assets = append(assets, asset.CoinMarginedFutures)
	case cLinear:
		assets = append(assets, asset.USDTMarginedFutures, asset.USDCMarginedFutures)
	case cOption:
		assets = append(assets, asset.Options)
	default:
		return currency.EMPTYPAIR, 0, fmt.Errorf("incoming symbol %q %w: %q", symbol, errUnsupportedCategory, category)
	}
	for _, a := range assets {
		cp, err := e.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return currency.EMPTYPAIR, 0, fmt.Errorf("%w for %s symbol %q", err, category, symbol)
			}
			continue
		}
		return cp, a, nil
	}
	return currency.EMPTYPAIR, 0, currency.ErrPairNotFound
}

// hasPotentialDelimiter returns if the asset has a potential delimiter on the pairs being returned.
func hasPotentialDelimiter(a asset.Item) bool {
	return a == asset.Options || a == asset.USDCMarginedFutures
}

// stringToInterval returns a kline.Interval instance from string.
func stringToInterval(s string) (kline.Interval, error) {
	interval, okay := stringToIntervalMap[s]
	if okay {
		return interval, nil
	}
	return 0, kline.ErrInvalidInterval
}
