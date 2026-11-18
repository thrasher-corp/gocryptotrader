package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"text/template"
	"time"

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
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Advanced Trade channels a subscription names, other than through a standard channel, to subscribe to
const (
	// ChannelStatus streams products' trading status
	ChannelStatus = "status"
	// ChannelTickerBatch streams products' tickers every five seconds, without their best bid and ask
	ChannelTickerBatch = "ticker_batch"
	// ChannelFuturesBalanceSummary streams the account's US derivatives balances and margin; it needs an authenticated
	// subscription
	ChannelFuturesBalanceSummary = "futures_balance_summary"
)

const (
	// The websocket's connections, which their messages are matched to: Advanced Trade's market data connections and
	// its user connection, and the gateway's public streams and its private connections, which also run its JSON-RPC
	// methods
	websocketMarketDataConnection = "market data"
	websocketUserConnection       = "user"
	derivativesStreamsConnection  = "derivatives streams"
	derivativesPrivateConnection  = "derivatives private"

	// websocketMaxSubscriptionsPerConnection is the most products an Advanced Trade connection's level2 channel takes,
	// counting each USDC alias apart from its USD product; every subscription holds one product or instrument, so each
	// connection is held to it
	websocketMaxSubscriptionsPerConnection = 30
	// websocketCloseTimeout bounds sending the close frame that has the venue close a connection for reconnection
	websocketCloseTimeout = 5 * time.Second

	// Advanced Trade's channel names, as requests and messages carry them
	wsChannelHeartbeats   = "heartbeats"
	wsChannelTicker       = "ticker"
	wsChannelCandles      = "candles"
	wsChannelMarketTrades = "market_trades"
	wsChannelLevel2       = "level2"
	// wsChannelLevel2Data is the channel level2 messages arrive on
	wsChannelLevel2Data = "l2_data"
	wsChannelUser       = "user"
	// wsChannelSubscriptions is the channel of the message confirming a subscribe or unsubscribe request, which lists
	// all of the connection's subscriptions
	wsChannelSubscriptions = "subscriptions"

	wsRequestSubscribe   = "subscribe"
	wsRequestUnsubscribe = "unsubscribe"
	// wsMessageError is the type of the message refusing a request, which names neither the request nor its channel
	wsMessageError = "error"

	wsEventSnapshot = "snapshot"
	wsEventUpdate   = "update"
)

var (
	errWebsocketRequestRefused         = errors.New("websocket request refused")
	errWebsocketSubscribeUnconfirmed   = errors.New("subscription not confirmed")
	errWebsocketUnsubscribeUnconfirmed = errors.New("unsubscription not confirmed")
	errWebsocketSequenceGap            = errors.New("websocket messages dropped")
	errWebsocketStaleMessage           = errors.New("websocket message older than the last received")
	errWebsocketChannelUnknown         = errors.New("websocket channel unknown")
	errWebsocketUnknownProduct         = errors.New("websocket product unknown")
	errWebsocketEventTypeUnknown       = errors.New("websocket event type unknown")
	errChannelNeedsPairs               = errors.New("channel needs an asset and pairs")
)

// advancedTradeChannel describes how an Advanced Trade channel is subscribed
type advancedTradeChannel struct {
	// name is the channel's name in requests
	name string
	// account channels need a JWT and stream on the user connection, for all of the account's products at once; the
	// others take one subscription per product
	account bool
}

// advancedTradeChannels lists every channel a subscription to an Advanced Trade asset may name
var advancedTradeChannels = map[string]advancedTradeChannel{
	subscription.TickerChannel:    {name: wsChannelTicker},
	ChannelTickerBatch:            {name: ChannelTickerBatch},
	subscription.CandlesChannel:   {name: wsChannelCandles},
	subscription.AllTradesChannel: {name: wsChannelMarketTrades},
	subscription.OrderbookChannel: {name: wsChannelLevel2},
	ChannelStatus:                 {name: ChannelStatus},
	subscription.MyAccountChannel: {name: wsChannelUser, account: true},
	ChannelFuturesBalanceSummary:  {name: ChannelFuturesBalanceSummary, account: true},
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.FiveMin},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: ChannelStatus},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// setupWebsocket sets the websocket up with its connections: Advanced Trade's market data connections, up to its level2
// limit each, and its user connection, which carries the account channels, then the gateway's public streams and its
// private connections, which carry its private channels and its JSON-RPC methods
func (e *Exchange) setupWebsocket(exch *config.Exchange) error {
	if err := e.Websocket.Setup(e.websocketManagerSetup(exch)); err != nil {
		return err
	}
	setups, err := e.websocketConnectionSetups(exch)
	if err != nil {
		return err
	}
	for _, setup := range setups {
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// websocketManagerSetup returns the websocket's manager setup
func (e *Exchange) websocketManagerSetup(exch *config.Exchange) *websocket.ManagerSetup {
	return &websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: websocketMaxSubscriptionsPerConnection,
		RateLimitDefinitions:                   e.rateLimits,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
	}
}

// websocketConnectionSetups returns the websocket's connection setups, each with its configured URL
func (e *Exchange) websocketConnectionSetups(exch *config.Exchange) ([]*websocket.ConnectionSetup, error) {
	setups := []struct {
		url   exchange.URL
		setup *websocket.ConnectionSetup
	}{
		{url: exchange.WebsocketSpot, setup: &websocket.ConnectionSetup{
			Connector:             e.wsConnect,
			GenerateSubscriptions: e.generateMarketDataSubscriptions,
			Subscriber:            e.wsSubscribe,
			Unsubscriber:          e.wsUnsubscribe,
			Handler:               e.wsHandleData,
			MessageFilter:         websocketMarketDataConnection,
		}},
		{url: exchange.WebsocketPrivate, setup: &websocket.ConnectionSetup{
			Connector:             e.wsConnect,
			GenerateSubscriptions: e.generateUserSubscriptions,
			Subscriber:            e.wsSubscribe,
			Unsubscriber:          e.wsUnsubscribe,
			Handler:               e.wsHandleData,
			MessageFilter:         websocketUserConnection,
		}},
		{url: exchange.WebsocketFutures, setup: &websocket.ConnectionSetup{
			Connector:             e.wsDerivativesConnect,
			GenerateSubscriptions: e.generateDerivativesStreamsSubscriptions,
			Subscriber:            e.wsDerivativesSubscribe,
			Unsubscriber:          e.wsDerivativesUnsubscribe,
			Handler:               e.wsHandleDerivatives,
			MessageFilter:         derivativesStreamsConnection,
		}},
		{url: exchange.WebsocketFuturesPrivate, setup: &websocket.ConnectionSetup{
			Connector:             e.wsDerivativesConnect,
			Authenticate:          e.wsDerivativesAuthenticate,
			GenerateSubscriptions: e.generateDerivativesPrivateSubscriptions,
			Subscriber:            e.wsDerivativesSubscribe,
			Unsubscriber:          e.wsDerivativesUnsubscribe,
			Handler:               e.wsHandleDerivatives,
			MessageFilter:         derivativesPrivateConnection,
		}},
	}
	connections := make([]*websocket.ConnectionSetup, len(setups))
	for i, c := range setups {
		u, err := e.API.Endpoints.GetURL(c.url)
		if err != nil {
			return nil, err
		}
		c.setup.URL = u
		c.setup.ResponseCheckTimeout = exch.WebsocketResponseCheckTimeout
		c.setup.ResponseMaxLimit = exch.WebsocketResponseMaxLimit
		connections[i] = c.setup
	}
	return connections, nil
}

// generateSubscriptions returns the configured subscriptions with their channels qualified, those carried by the
// connections keep selects. A subscription to a channel that is not qualified per pair streams once whatever asset it
// names, so it is held without an asset or pairs, once; a USDC pair other than USDT-USDC and EURC-USDC streams as its
// USD pair, which Coinbase subscribes once for both, so a channel streaming both is subscribed once, for the USD pair
func (e *Exchange) generateSubscriptions(keep func(*subscription.Subscription) bool) (subscription.List, error) {
	expanded, err := e.Features.Subscriptions.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	subs := make(subscription.List, 0, len(expanded))
	streams := make(map[string]bool, len(expanded))
	for _, s := range expanded {
		if !keep(s) {
			continue
		}
		if !subscriptionPerPair(s, s.Asset) {
			s.Asset, s.Pairs = asset.Empty, nil
			if streams[s.QualifiedChannel] {
				continue
			}
			streams[s.QualifiedChannel] = true
			subs = append(subs, s)
			continue
		}
		stream := s.QualifiedChannel + " " + e.websocketStreamProduct(s.Asset, s.Pairs[0])
		if streams[stream] {
			continue
		}
		streams[stream] = true
		subs = append(subs, s)
	}
	return subs, nil
}

// generateMarketDataSubscriptions returns the subscriptions Advanced Trade's market data connections carry
func (e *Exchange) generateMarketDataSubscriptions() (subscription.List, error) {
	return e.generateSubscriptions(func(s *subscription.Subscription) bool {
		ch, ok := advancedTradeChannelOf(s)
		return ok && !ch.account
	})
}

// generateUserSubscriptions returns the subscriptions Advanced Trade's user connection carries
func (e *Exchange) generateUserSubscriptions() (subscription.List, error) {
	return e.generateSubscriptions(func(s *subscription.Subscription) bool {
		ch, ok := advancedTradeChannelOf(s)
		return ok && ch.account
	})
}

// websocketStreamProduct returns the product a pair streams as: a USDC alias streams as the USD product it trades on
// the same book as. Product data names the aliases once tradable pairs are fetched; until then the documented rule
// names them
func (e *Exchange) websocketStreamProduct(a asset.Item, p currency.Pair) string {
	if a == asset.Spot {
		if canonical, ok := e.pairAliases.canonical(p); ok {
			p = canonical
		} else if p.Quote.Equal(currency.USDC) && !p.Base.Equal(currency.USDT) && !p.Base.Equal(currency.EURC) {
			p = currency.NewPair(p.Base, currency.USD)
		}
	}
	return p.Format(productIDFormat).String()
}

// productIDFormat formats a pair as an Advanced Trade product ID
var productIDFormat = currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}

// advancedTradeChannelOf returns the Advanced Trade channel an expanded subscription streams on, if any
func advancedTradeChannelOf(s *subscription.Subscription) (advancedTradeChannel, bool) {
	if isDerivativesAsset(s.Asset) {
		return advancedTradeChannel{}, false
	}
	ch, ok := advancedTradeChannels[s.Channel]
	if !ok || s.QualifiedChannel != ch.name {
		return advancedTradeChannel{}, false
	}
	return ch, true
}

// isDerivativesAsset reports whether an asset trades on the Global Derivatives gateway rather than Advanced Trade
func isDerivativesAsset(a asset.Item) bool {
	return a == asset.USDCMarginedFutures || a == asset.CoinMarginedFutures
}

// GetSubscriptionTemplate returns the channel qualification template: each asset's channel is qualified by the venue
// trading the asset, per pair for a channel taking pairs, and an asset-less subscription by the venue serving its
// channel
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"perPair":     subscriptionPerPair,
		"qualifyPair": qualifyPairChannel,
		"qualify":     qualifyChannel,
	}).Parse(websocketSubscriptionTemplate)
}

// websocketSubscriptionTemplate qualifies a subscription's channel for each of its assets, per pair for a channel
// taking pairs and once otherwise; an asset the channel does not stream for an asset.All subscription generates nothing
const websocketSubscriptionTemplate = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- if perPair $.S $asset }}
		{{- range $pair := $pairs }}
			{{- qualifyPair $.S $asset $pair }}
			{{- $.PairSeparator }}
		{{- end }}
	{{- else }}
		{{- qualify $.S $asset }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}
`

// subscriptionPerPair reports whether a subscription's channel is qualified per pair for an asset
func subscriptionPerPair(s *subscription.Subscription, a asset.Item) bool {
	switch {
	case a == asset.Empty:
		return false
	case isDerivativesAsset(a):
		name, err := derivativesChannelName(s)
		return err == nil && derivativesChannels[name].instrument
	default:
		ch, ok := advancedTradeChannels[s.Channel]
		return ok && !ch.account
	}
}

// qualifyPairChannel returns a subscription's channel for one of an asset's pairs: the Advanced Trade channel's name,
// as each subscription to one carries its product as its pair, or the gateway channel named for the instrument
func qualifyPairChannel(s *subscription.Subscription, a asset.Item, p currency.Pair) (string, error) {
	if isDerivativesAsset(a) {
		return qualifyDerivativesInstrumentChannel(s, p)
	}
	ch := advancedTradeChannels[s.Channel]
	if ch.name == wsChannelCandles && s.Interval != 0 && s.Interval != kline.FiveMin {
		return "", fmt.Errorf("%w: %s, as Advanced Trade candles are five minutes", kline.ErrUnsupportedInterval, s.Interval)
	}
	return ch.name, nil
}

// qualifyChannel returns a subscription's channel when it is not qualified per pair. An asset.All subscription to a
// channel an asset's venue does not stream generates nothing for that asset, and a heartbeat subscription nothing at
// all, as every connection subscribes to its heartbeats
func qualifyChannel(s *subscription.Subscription, a asset.Item) (string, error) {
	if s.Channel == subscription.HeartbeatChannel {
		return "", nil
	}
	if a == asset.Empty {
		if ch, ok := advancedTradeChannels[s.Channel]; ok {
			if !ch.account {
				return "", fmt.Errorf("%w: %s", errChannelNeedsPairs, s.Channel)
			}
			return ch.name, nil
		}
		return qualifyDerivativesChannel(s)
	}
	if isDerivativesAsset(a) {
		if _, err := derivativesChannelName(s); err != nil {
			return unstreamedChannel(s, err)
		}
		return qualifyDerivativesChannel(s)
	}
	ch, ok := advancedTradeChannels[s.Channel]
	if !ok {
		return unstreamedChannel(s, fmt.Errorf("%w: %s %s", subscription.ErrNotSupported, a, s.Channel))
	}
	return ch.name, nil
}

// unstreamedChannel returns nothing for an asset.All subscription to a channel one of its assets does not stream, and
// the error for a subscription naming that asset
func unstreamedChannel(s *subscription.Subscription, err error) (string, error) {
	if s.Asset == asset.All {
		return "", nil
	}
	return "", err
}

// wsConnect dials an Advanced Trade connection, within Coinbase's limit of 8 connections a second per IP
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := e.Requester.InitiateRateLimit(ctx, websocketConnectEPL); err != nil {
		return err
	}
	e.wsConnections.pruneClosed()
	return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
}

// wsSubscribe subscribes an Advanced Trade connection to channels, with one request for each channel's products. A new
// connection first subscribes to its heartbeats, which keep it open while its channels are quiet; Coinbase closes a
// connection that subscribes to nothing within 5 seconds
func (e *Exchange) wsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	state := e.wsConnections.get(conn)
	state.requestMu.Lock()
	defer state.requestMu.Unlock()
	var errs error
	if !state.heartbeats {
		// A JWT is optional for heartbeats; Coinbase recommends one for the user connection's reliability
		authenticated := slices.ContainsFunc(subs, func(s *subscription.Subscription) bool { return s.Authenticated })
		if _, err := e.wsSendRequest(ctx, conn, wsRequestSubscribe, wsChannelHeartbeats, nil, authenticated); err != nil {
			errs = fmt.Errorf("%w: %s", err, wsChannelHeartbeats)
		} else {
			state.heartbeats = true
		}
	}
	for _, channelSubs := range groupByChannel(subs) {
		errs = common.AppendError(errs, e.wsManageChannel(ctx, conn, wsRequestSubscribe, channelSubs))
	}
	return errs
}

// wsUnsubscribe unsubscribes an Advanced Trade connection from channels, with one request for each channel's products
func (e *Exchange) wsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	state := e.wsConnections.get(conn)
	state.requestMu.Lock()
	defer state.requestMu.Unlock()
	var errs error
	for _, channelSubs := range groupByChannel(subs) {
		errs = common.AppendError(errs, e.wsManageChannel(ctx, conn, wsRequestUnsubscribe, channelSubs))
	}
	return errs
}

// groupByChannel groups subscriptions by their qualified channel, in the order the channels first appear
func groupByChannel(subs subscription.List) []subscription.List {
	var groups []subscription.List
	index := make(map[string]int)
	for _, s := range subs {
		i, ok := index[s.QualifiedChannel]
		if !ok {
			i = len(groups)
			index[s.QualifiedChannel] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], s)
	}
	return groups
}

// wsManageChannel subscribes to or unsubscribes from one channel's subscriptions in one request, then records each
// the reply confirms and reports the others
func (e *Exchange) wsManageChannel(ctx context.Context, conn websocket.Connection, op string, subs subscription.List) error {
	ch, ok := advancedTradeChannelOf(subs[0])
	if !ok {
		return fmt.Errorf("%w: %s", errWebsocketChannelUnknown, subs[0].QualifiedChannel)
	}
	var productIDs []string
	if !ch.account {
		productIDs = make([]string, len(subs))
		for i, s := range subs {
			productIDs[i] = s.Pairs[0].Format(productIDFormat).String()
		}
	}
	confirmation, err := e.wsSendRequest(ctx, conn, op, ch.name, productIDs, ch.account)
	if err != nil {
		return fmt.Errorf("%w: %s %s", err, op, subs)
	}
	var errs error
	for _, s := range subs {
		listed := e.confirmationLists(confirmation, ch.name, s)
		switch {
		case op == wsRequestSubscribe && listed:
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, s))
		case op == wsRequestSubscribe:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errWebsocketSubscribeUnconfirmed, s))
		case listed:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errWebsocketUnsubscribeUnconfirmed, s))
		default:
			errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
		}
	}
	return errs
}

// confirmationLists reports whether a confirmation lists a subscription: an account channel by name, and a product
// channel's subscription by its product or the product it streams as, which Coinbase lists in its place
func (e *Exchange) confirmationLists(confirmation *WebsocketSubscriptionsEvent, channel string, s *subscription.Subscription) bool {
	products, ok := confirmation.Subscriptions[channel]
	if !ok || len(s.Pairs) == 0 {
		return ok
	}
	return slices.Contains(products, s.Pairs[0].Format(productIDFormat).String()) || slices.Contains(products, e.websocketStreamProduct(s.Asset, s.Pairs[0]))
}

// wsSendRequest sends a subscribe or unsubscribe request and returns the subscriptions its reply lists. Coinbase's
// reply names neither the request nor its channel, so the connection's requests are sent one at a time; an
// authenticated request carries a fresh JWT, as each expires after two minutes
func (e *Exchange) wsSendRequest(ctx context.Context, conn websocket.Connection, op, channel string, productIDs []string, authenticated bool) (*WebsocketSubscriptionsEvent, error) {
	// A request its caller stopped waiting for before it was sent has no reply that could arrive late
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req := &WebsocketWireRequest{Type: op, Channel: channel, ProductIDs: productIDs}
	if authenticated {
		jwt, _, err := e.GetJWT(ctx, "")
		if err != nil {
			return nil, err
		}
		req.JWT = jwt
	}
	resp, err := conn.SendMessageReturnResponse(ctx, websocketMessageEPL, wsChannelSubscriptions, req)
	if err != nil {
		if (errors.Is(err, websocket.ErrSignatureTimeout) || ctx.Err() != nil) && isConnected(conn) {
			// The reply can still arrive, after the request stopped waiting for it, and be taken for the next request's,
			// so a connection still up is closed, for the manager to reconnect and subscribe afresh; the close outlives
			// the request's context
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), websocketCloseTimeout)
			defer cancel()
			e.wsCloseForReconnection(closeCtx, conn, websocketMessageEPL, "request unanswered")
		}
		return nil, err
	}
	var reply *WebsocketSubscriptionsResponse
	if err := json.Unmarshal(resp, &reply); err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, fmt.Errorf("%w: %s", common.ErrNoResponse, resp)
	}
	if reply.Type == wsMessageError {
		return nil, fmt.Errorf("%w: %s", errWebsocketRequestRefused, reply.Message)
	}
	if len(reply.Events) == 0 {
		return nil, fmt.Errorf("%w: %s", common.ErrNoResponse, resp)
	}
	return &reply.Events[len(reply.Events)-1], nil
}

// wsHandleData checks an Advanced Trade message's sequence, then routes it to its request or its channel's handler
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var envelope WebsocketEnvelope
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return fmt.Errorf("%s could not parse websocket data: %w: %s", e.Name, err, respRaw)
	}
	if envelope.Type == wsMessageError {
		if conn.IncomingWithData(wsChannelSubscriptions, respRaw) {
			return nil
		}
		return fmt.Errorf("%w: %s", errWebsocketRequestRefused, envelope.Message)
	}
	gap, stale := e.wsConnections.get(conn).sequence(envelope.Sequence)
	if stale {
		// Coinbase allows messages to arrive out of order; one older than the last is dropped, as acting on it would undo
		// a newer one, and the gap it left has already been resynchronised
		return fmt.Errorf("%w: %s sequence %d", errWebsocketStaleMessage, envelope.Channel, envelope.Sequence)
	}
	var errs error
	if gap {
		e.wsResynchronise(ctx, conn)
		errs = fmt.Errorf("%w before %s sequence %d", errWebsocketSequenceGap, envelope.Channel, envelope.Sequence)
	}
	return common.AppendError(errs, e.wsHandleChannel(ctx, conn, &envelope, respRaw))
}

// wsHandleChannel passes a message to its channel's handler, or a confirmation to the request awaiting it
func (e *Exchange) wsHandleChannel(ctx context.Context, conn websocket.Connection, envelope *WebsocketEnvelope, respRaw []byte) error {
	switch envelope.Channel {
	case wsChannelSubscriptions:
		// A confirmation no request awaits answers one that stopped waiting, and is already superseded
		conn.IncomingWithData(wsChannelSubscriptions, respRaw)
		return nil
	case wsChannelHeartbeats:
		return nil
	case wsChannelTicker, ChannelTickerBatch:
		return e.wsHandleTicker(ctx, envelope)
	case wsChannelCandles:
		return e.wsHandleCandles(ctx, envelope)
	case wsChannelMarketTrades:
		return e.wsHandleMarketTrades(ctx, envelope)
	case wsChannelLevel2Data:
		return e.wsHandleLevel2(ctx, conn, envelope)
	case ChannelStatus:
		return e.wsHandleStatus(ctx, envelope)
	case wsChannelUser:
		return e.wsHandleUser(ctx, envelope)
	case ChannelFuturesBalanceSummary:
		return e.wsHandleFuturesBalanceSummary(ctx, envelope)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// wsResynchronise resubscribes a connection's stateful subscriptions after it dropped messages, in the background so
// the reader goes on to read the replies: its order books, invalidated until new snapshots arrive, and its account
// channels, whose snapshots replace what was missed. Coinbase numbers a connection's messages, not its channels', so a
// gap could have been in any of them
func (e *Exchange) wsResynchronise(ctx context.Context, conn websocket.Connection) {
	var resubscribe subscription.List
	for _, s := range conn.Subscriptions().List() {
		ch, ok := advancedTradeChannelOf(s)
		switch {
		case !ok:
		case ch.name == wsChannelLevel2:
			e.wsInvalidateBooks(s)
			resubscribe = append(resubscribe, s)
		case ch.account:
			resubscribe = append(resubscribe, s)
		}
	}
	e.wsResubscribe(ctx, conn, resubscribe, websocketMessageEPL)
}

// wsResubscribe resubscribes a connection's subscriptions in the background, so the reader goes on to read the replies,
// leaving out those another resubscription holds. They are claimed before the reader moves on, as the manager marks
// them resubscribing only once the goroutine runs. A failed resubscription leaves its subscriptions half-way, never
// resubscribed or unsubscribed, so the connection is closed for the manager to reconnect and subscribe afresh: the
// venue answers a close by closing, which the manager takes as a fault
func (e *Exchange) wsResubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List, epl request.EndpointLimit) {
	state := e.wsConnections.get(conn)
	claimed := state.claimResubscription(subs)
	if len(claimed) == 0 {
		return
	}
	go func() {
		defer state.releaseResubscription(claimed)
		err := e.Websocket.ResubscribeFromConnection(ctx, conn, claimed)
		if err == nil || ctx.Err() != nil || !isConnected(conn) {
			return
		}
		log.Errorf(log.WebsocketMgr, "%s websocket could not resubscribe %s, reconnecting: %v", e.Name, claimed, err)
		e.wsCloseForReconnection(ctx, conn, epl, "resubscription failed")
	}()
}

// wsCloseForReconnection asks the venue to close a connection, which it answers by closing it; the manager takes that
// as a fault and reconnects, subscribing afresh, where a connection shut down here would not be reconnected
func (e *Exchange) wsCloseForReconnection(ctx context.Context, conn websocket.Connection, epl request.EndpointLimit, reason string) {
	if err := conn.SendRawMessage(ctx, epl, gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, reason)); err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket could not be closed for reconnection: %v", e.Name, err)
	}
}

// websocketConnectionState holds an Advanced Trade connection's own state
type websocketConnectionState struct {
	// requestMu sends the connection's requests one at a time, as Coinbase's replies do not name their request; it
	// guards heartbeats
	requestMu  sync.Mutex
	heartbeats bool

	sequenceMu sync.Mutex
	last       uint64
	started    bool

	// resubscribing holds the subscriptions a resubscription has claimed
	resubscribingMu sync.Mutex
	resubscribing   map[*subscription.Subscription]bool
}

// claimResubscription claims the subscriptions no resubscription holds, returning them
func (s *websocketConnectionState) claimResubscription(subs subscription.List) subscription.List {
	s.resubscribingMu.Lock()
	defer s.resubscribingMu.Unlock()
	if s.resubscribing == nil {
		s.resubscribing = make(map[*subscription.Subscription]bool)
	}
	claimed := make(subscription.List, 0, len(subs))
	for _, sub := range subs {
		if !s.resubscribing[sub] {
			s.resubscribing[sub] = true
			claimed = append(claimed, sub)
		}
	}
	return claimed
}

// releaseResubscription releases subscriptions a resubscription claimed
func (s *websocketConnectionState) releaseResubscription(subs subscription.List) {
	s.resubscribingMu.Lock()
	defer s.resubscribingMu.Unlock()
	for _, sub := range subs {
		delete(s.resubscribing, sub)
	}
}

// isResubscribing reports whether a resubscription holds a subscription
func (s *websocketConnectionState) isResubscribing(sub *subscription.Subscription) bool {
	s.resubscribingMu.Lock()
	defer s.resubscribingMu.Unlock()
	return s.resubscribing[sub]
}

// sequence records a message's sequence number, reporting a gap when messages before it were dropped, and a message
// older than the last as stale; a connection's first message starts its sequence
func (s *websocketConnectionState) sequence(n uint64) (gap, stale bool) {
	s.sequenceMu.Lock()
	defer s.sequenceMu.Unlock()
	switch {
	case !s.started:
	case n <= s.last:
		return false, true
	case n != s.last+1:
		gap = true
	}
	s.started, s.last = true, n
	return gap, false
}

// websocketConnections holds each Advanced Trade connection's state
type websocketConnections struct {
	m      sync.Mutex
	states map[websocket.Connection]*websocketConnectionState
}

// get returns a connection's state, starting it for a new connection
func (c *websocketConnections) get(conn websocket.Connection) *websocketConnectionState {
	c.m.Lock()
	defer c.m.Unlock()
	if c.states == nil {
		c.states = make(map[websocket.Connection]*websocketConnectionState)
	}
	state, ok := c.states[conn]
	if !ok {
		state = new(websocketConnectionState)
		c.states[conn] = state
	}
	return state
}

// pruneClosed forgets the state of connections that have closed, which the manager replaces with new ones
func (c *websocketConnections) pruneClosed() {
	c.m.Lock()
	defer c.m.Unlock()
	for conn := range c.states {
		if !isConnected(conn) {
			delete(c.states, conn)
		}
	}
}
