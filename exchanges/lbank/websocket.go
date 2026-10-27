package lbank

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
)

const (
	// wsPublicConnection and wsAccountConnection filter the connections carrying the market data and the account's
	// orders and balances; the account channels need a subscribe key, which each account connection is given
	wsPublicConnection  = "public"
	wsAccountConnection = "account"

	// wsMaxSubscriptionsPerConnection spreads the subscriptions over connections. LBank documents no limit, and one
	// connection took 300 subscriptions when probed
	wsMaxSubscriptionsPerConnection = 200

	// wsPingDelay keeps a quiet connection's traffic flowing; LBank pings about once a minute and closes a connection
	// that leaves a ping unanswered for a minute
	wsPingDelay = 20 * time.Second

	wsTickerChannel      = "tick"
	wsDepthChannel       = "depth"
	wsTradeChannel       = "trade"
	wsKlineChannel       = "kbar"
	wsOrderUpdateChannel = "orderUpdate"
	wsAssetUpdateChannel = "assetUpdate"

	// wsDefaultDepth is the depth an order book subscription without levels takes, the deepest LBank documents. LBank
	// documents 10, 50 and 100 only, but streamed other depths, such as 7, when probed
	wsDefaultDepth = 100
)

var (
	errWebsocketError         = errors.New("websocket error")
	errMalformedWebsocketTime = errors.New("malformed websocket time")
	errUnsupportedChannel     = errors.New("unsupported websocket channel")
	errSubscriptionPairs      = errors.New("subscription must name exactly one pair")
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: wsDefaultDepth},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    wsTickerChannel,
	subscription.OrderbookChannel: wsDepthChannel,
	subscription.AllTradesChannel: wsTradeChannel,
	subscription.CandlesChannel:   wsKlineChannel,
	subscription.MyOrdersChannel:  wsOrderUpdateChannel,
	subscription.MyAccountChannel: wsAssetUpdateChannel,
}

// accountChannels are the channels carrying the account's orders and balances, which LBank streams for every pair at
// once
var accountChannels = []string{wsOrderUpdateChannel, wsAssetUpdateChannel}

// websocketKlineSlots holds the candle intervals LBank streams. It also streams day, week and month candles, which start
// at midnight UTC+8
var websocketKlineSlots = map[kline.Interval]string{
	kline.OneMin:     "1min",
	kline.FiveMin:    "5min",
	kline.FifteenMin: "15min",
	kline.ThirtyMin:  "30min",
	kline.OneHour:    "1hr",
	kline.FourHour:   "4hr",
	kline.OneDay:     "day",
	kline.OneWeek:    "week",
	kline.OneMonth:   "month",
}

// setupWebsocket sets the websocket up with connections carrying the market data and an account connection, made only
// when authenticated websocket use is available, carrying the account's orders and balances
func (e *Exchange) setupWebsocket(exch *config.Exchange) error {
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: wsMaxSubscriptionsPerConnection,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
	}); err != nil {
		return err
	}
	wsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.wsConnect,
		GenerateSubscriptions: e.generatePublicSubscriptions,
		Subscriber:            e.wsSubscribe,
		Unsubscriber:          e.wsUnsubscribe,
		Handler:               e.wsHandleData,
		MessageFilter:         wsPublicConnection,
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.wsConnect,
		GenerateSubscriptions: e.generateAccountSubscriptions,
		Subscriber:            e.wsSubscribe,
		Unsubscriber:          e.wsUnsubscribe,
		Handler:               e.wsHandleData,
		MessageFilter:         wsAccountConnection,
	})
}

// wsConnect dials a connection and pings it so a quiet connection still shows traffic
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	ping, err := json.Marshal(&websocketHeartbeatRequest{Action: "ping", Ping: "gct"})
	if err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     ping,
		Delay:       wsPingDelay,
	})
	return nil
}

// generatePublicSubscriptions returns the configured subscriptions to the market data channels
func (e *Exchange) generatePublicSubscriptions() (subscription.List, error) {
	subs, err := normaliseSubscriptions(e.Features.Subscriptions).ExpandTemplates(e)
	return slices.DeleteFunc(subs, isAccountSubscription), err
}

// generateAccountSubscriptions returns the configured subscriptions to the account channels, none unless authenticated
// websocket use is available
func (e *Exchange) generateAccountSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	subs, err := normaliseSubscriptions(e.Features.Subscriptions).ExpandTemplates(e)
	return slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return !isAccountSubscription(s) }), err
}

// normaliseSubscriptions returns the subscriptions with the asset and pairs of account channels cleared, since LBank
// streams those channels for every pair at once, so they are subscribed to once rather than once per pair, and are
// unaffected by pairs being enabled
func normaliseSubscriptions(l subscription.List) subscription.List {
	normalised := make(subscription.List, len(l))
	for i, s := range l {
		normalised[i] = s
		if s.QualifiedChannel != "" || !slices.Contains(accountChannels, subscriptionNames[s.Channel]) {
			continue
		}
		c := s.Clone()
		c.Asset = asset.Empty
		c.Pairs = nil
		c.Authenticated = true
		normalised[i] = c
	}
	return normalised
}

// isAccountSubscription reports whether an expanded subscription is to an account channel, going by its qualified
// channel rather than its Authenticated flag, which a configured subscription can leave out
func isAccountSubscription(s *subscription.Subscription) bool {
	return slices.Contains(accountChannels, s.QualifiedChannel)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isAccountChannel": isAccountChannel,
		"qualifiedChannel": e.qualifiedChannel,
	}).Parse(subTplText)
}

// channelName returns a subscription's LBank channel name
func channelName(s *subscription.Subscription) (string, error) {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel)
}

// isAccountChannel reports whether a subscription is to an account channel
func isAccountChannel(s *subscription.Subscription) bool {
	return slices.Contains(accountChannels, subscriptionNames[s.Channel])
}

// qualifiedChannel returns a market data subscription's qualified channel for one pair: the channel name, its depth or
// candle interval, and the pair's symbol, such as depth:100:btc_usdt
func (e *Exchange) qualifiedChannel(s *subscription.Subscription, a asset.Item, p currency.Pair) (string, error) {
	name, err := channelName(s)
	if err != nil {
		return "", err
	}
	symbol, err := e.FormatSymbol(p, a)
	if err != nil {
		return "", err
	}
	switch name {
	case wsDepthChannel:
		return name + ":" + strconv.Itoa(depthLevels(s)) + ":" + symbol, nil
	case wsKlineChannel:
		slot, ok := websocketKlineSlots[s.Interval]
		if !ok {
			return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, s.Interval)
		}
		return name + ":" + slot + ":" + symbol, nil
	}
	return name + ":" + symbol, nil
}

// depthLevels returns the depth an order book subscription takes
func depthLevels(s *subscription.Subscription) int {
	if s.Levels > 0 {
		return s.Levels
	}
	return wsDefaultDepth
}

const subTplText = `
{{- if isAccountChannel $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- channelName $.S }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- range $p := $pairs -}}
			{{- qualifiedChannel $.S $asset $p }}
			{{- $.PairSeparator }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- end }}
`

// wsSubscribe subscribes to channels. LBank confirms no subscription, so each is recorded once its request is sent; it
// reports a pair it doesn't trade, or a subscribe key it doesn't recognise, in an error frame the handler relays. The
// account channels take a subscribe key of the connection's own, created with its first account subscription, so a key
// LBank refuses fails only those subscriptions, which the next connect or flush retries, and leaves the market data up.
// Starting a key's renewer reads the manager's ShutdownC and adds to its Wg, which would race a Shutdown without the
// manager's lock, so the account channels are subscribed only by the manager's connects and flushes, which hold it, and
// not through SubscribeToChannels or the resubscribe methods
func (e *Exchange) wsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	if !slices.ContainsFunc(subs, isAccountChannel) {
		return e.manageSubscriptions(ctx, conn, "subscribe", subs)
	}
	e.subscribeKeys.account.Lock()
	defer e.subscribeKeys.account.Unlock()
	var errs error
	if e.subscribeKeys.get(conn) == "" {
		errs = e.startSubscribeKey(ctx, conn, defaultSubscribeKeyTimings)
	}
	return common.AppendError(errs, e.manageSubscriptions(ctx, conn, "subscribe", subs))
}

// wsUnsubscribe unsubscribes from channels, which LBank confirms no more than subscriptions
func (e *Exchange) wsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	if slices.ContainsFunc(subs, isAccountChannel) {
		e.subscribeKeys.account.Lock()
		defer e.subscribeKeys.account.Unlock()
	}
	return e.manageSubscriptions(ctx, conn, "unsubscribe", subs)
}

func (e *Exchange) manageSubscriptions(ctx context.Context, conn websocket.Connection, action string, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		req, err := e.subscriptionRequest(conn, action, s)
		if err == nil {
			err = conn.SendJSONMessage(ctx, request.Unset, req)
		}
		if err == nil && req.SubscribeKey != "" {
			// The key an account channel went out with tells a replacement key to subscribe the channel again
			if action == "subscribe" {
				e.subscribeKeys.markSent(conn, req.Subscribe, req.SubscribeKey)
			} else {
				e.subscribeKeys.forgetSent(conn, req.Subscribe)
			}
		}
		if err == nil {
			if action == "subscribe" {
				err = e.Websocket.AddSuccessfulSubscriptions(conn, s)
			} else {
				err = e.Websocket.RemoveSubscriptions(conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s: %w", action, s.QualifiedChannel, err))
		}
	}
	return errs
}

// subscriptionRequest returns the request subscribing to or unsubscribing from a subscription's channel
func (e *Exchange) subscriptionRequest(conn websocket.Connection, action string, s *subscription.Subscription) (*websocketSubscribeRequest, error) {
	name, ok := subscriptionNames[s.Channel]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnsupportedChannel, s.Channel)
	}
	req := &websocketSubscribeRequest{Action: action, Subscribe: name}
	if slices.Contains(accountChannels, name) {
		key := e.subscribeKeys.get(conn)
		if key == "" {
			return nil, errSubscribeKeyRequired
		}
		req.SubscribeKey = key
		if name == wsOrderUpdateChannel {
			req.Pair = "all"
		}
		return req, nil
	}
	if len(s.Pairs) != 1 {
		return nil, fmt.Errorf("%w: %d pairs", errSubscriptionPairs, len(s.Pairs))
	}
	symbol, err := e.FormatSymbol(s.Pairs[0], s.Asset)
	if err != nil {
		return nil, err
	}
	req.Pair = symbol
	switch name {
	case wsDepthChannel:
		req.Depth = strconv.Itoa(depthLevels(s))
	case wsKlineChannel:
		slot, ok := websocketKlineSlots[s.Interval]
		if !ok {
			return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, s.Interval)
		}
		req.Kline = slot
	}
	return req, nil
}

// wsHandleData routes a frame to the handler of its type
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var msg websocketMessage
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	switch {
	case msg.Action == "ping":
		return conn.SendJSONMessage(ctx, request.Unset, &websocketHeartbeatRequest{Action: "pong", Pong: msg.Ping})
	case msg.Action == "pong":
		return nil
	case msg.Status == "error":
		// LBank appends the caller's public IP to its messages, which is left out so it isn't logged
		message, _, _ := strings.Cut(msg.Message, ",remoteIp")
		return fmt.Errorf("%w: %s", errWebsocketError, message)
	}
	switch msg.Type {
	case wsTickerChannel:
		return e.wsProcessTicker(ctx, &msg)
	case wsDepthChannel:
		return e.wsProcessDepth(ctx, &msg)
	case wsTradeChannel:
		return e.wsProcessTrade(&msg)
	case wsKlineChannel:
		return e.wsProcessKline(ctx, &msg)
	case wsOrderUpdateChannel:
		return e.wsProcessOrderUpdate(ctx, &msg)
	case wsAssetUpdateChannel:
		return e.wsProcessAssetUpdate(ctx, &msg)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

// matchSpotPair returns the available spot pair a frame's symbol names
func (e *Exchange) matchSpotPair(symbol string) (currency.Pair, error) {
	return e.MatchSymbolWithAvailablePairs(symbol, asset.Spot, true)
}
