package bitstamp

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
	"github.com/thrasher-corp/gocryptotrader/common/key"
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

// GetWebsocketToken returns a token and user ID for subscribing to private websocket channels
// The token is only valid for a short period and must be used immediately
func (e *Exchange) GetWebsocketToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	var resp *WebsocketTokenResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/websockets_token/", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("error fetching websocket token: %w", err)
	}
	return resp, nil
}

const (
	bitstampWSURL = "wss://ws.bitstamp.net" //nolint:gosec // G101 false positive on a public URL
	// hbInterval keeps traffic flowing for the connection monitor when the subscribed channels are quiet
	hbInterval = 8 * time.Second
	// websocketConnection is the message filter of the connection setup which every connection is made from
	websocketConnection = "websocket"

	privateChannelPrefix = "private-"
)

// Websocket channels
// diff_order_book is not supported: its documented reconciliation with a REST snapshot keeps levels of orders which
// are created and deleted between two broadcasts, which leaves the book crossed
const (
	channelLiveTrades         = "live_trades"
	channelOrderbook          = "order_book"
	channelDetailOrderbook    = "detail_order_book"
	channelFundingRate        = "funding_rate"
	channelAnnouncements      = "announcements"
	channelMyOrders           = "my_orders"
	channelMyTrades           = "my_trades"
	channelLiveOrderTrades    = "live_order_trades"
	channelMySettlements      = "my_settlements"
	channelMyLiquidations     = "my_liquidations"
	channelMyTokenSettlements = "my_token_settlements"
	// channelSelfTrades subscribes to the private live trades channel, which only carries self trades
	channelSelfTrades = "self_trades"
)

var (
	errParsingWSPair       = errors.New("unable to parse currency pair from channel")
	errMalformedChannel    = errors.New("malformed channel name")
	errWebsocketError      = errors.New("websocket error")
	errOrderIDMissing      = errors.New("order event missing an order ID")
	errMalformedOrderEvent = errors.New("malformed order event")
	errOrderEventRecovery  = errors.New("unable to recover missed order events")
	errOrderEventsSkipped  = errors.New("missed order events skipped")
	errStaleOrderEvent     = errors.New("order event is not after the last one handled")
	errOrderIDOverflow     = errors.New("order ID exceeds the maximum orderbook level ID")
	errOrderbookSeed       = errors.New("unable to load the REST orderbook")

	hbMsg = []byte(`{"event":"bts:heartbeat"}`)
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.OrderbookChannel: channelOrderbook,
	subscription.AllTradesChannel: channelLiveTrades,
	subscription.MyOrdersChannel:  channelMyOrders,
	subscription.MyTradesChannel:  channelMyTrades,
	channelDetailOrderbook:        channelDetailOrderbook,
	channelFundingRate:            channelFundingRate,
	channelAnnouncements:          channelAnnouncements,
	channelLiveOrderTrades:        channelLiveOrderTrades,
	channelSelfTrades:             channelLiveTrades,
	channelMySettlements:          channelMySettlements,
	channelMyLiquidations:         channelMyLiquidations,
	channelMyTokenSettlements:     channelMyTokenSettlements,
}

// accountChannels are channels which are not specific to a market; subscriptions to them should use an empty asset
var accountChannels = []string{channelAnnouncements, channelMySettlements, channelMyLiquidations, channelMyTokenSettlements}

// privateChannels are the subscription channels which can only be subscribed to with a token
var privateChannels = []string{
	subscription.MyOrdersChannel, subscription.MyTradesChannel, channelLiveOrderTrades, channelSelfTrades,
	channelMySettlements, channelMyLiquidations, channelMyTokenSettlements,
}

// WebsocketConnect connects to the websocket server and keeps the connection's traffic flowing with heartbeats
func (e *Exchange) WebsocketConnect(ctx context.Context, conn websocket.Connection) error {
	var dialer gws.Dialer
	if err := conn.Dial(ctx, &dialer, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     hbMsg,
		Delay:       hbInterval,
	})
	return nil
}

func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	event, err := jsonparser.GetUnsafeString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w `event`: %w", common.ErrParsingWSField, err)
	}

	switch event {
	case "bts:heartbeat":
		return e.retryKeptOrderEvents(ctx)
	case "bts:subscription_succeeded", "bts:unsubscription_succeeded":
		return handleWSSubscription(conn, event, respRaw)
	case "bts:error":
		return handleWSError(respRaw)
	case "bts:request_reconnect":
		go func() {
			if err := e.Websocket.Shutdown(); err != nil { // Connection monitor will reconnect
				log.Errorf(log.WebsocketMgr, "%s failed to shutdown websocket: %v", e.Name, err)
			}
		}()
		return nil
	}

	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}
	name, market, private, err := splitChannel(channel)
	if err != nil {
		return err
	}
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w `data`: %w", common.ErrParsingWSField, err)
	}

	switch {
	case name == channelAnnouncements && !private:
		return e.handleWSAnnouncement(ctx, event, data)
	case name == channelMySettlements && private:
		return e.handleWSSettlement(ctx, event, data)
	case name == channelMyLiquidations && private:
		var alert WebsocketLiquidationAlert
		if err := json.Unmarshal(data, &alert); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &alert)
	case name == channelMyTokenSettlements && private:
		var settlement WebsocketTokenSettlement
		if err := json.Unmarshal(data, &settlement); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &settlement)
	case market == "":
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}

	pair, a, err := e.marketPair(market)
	if err != nil {
		return err
	}
	switch {
	case name == channelOrderbook && !private:
		return e.handleWSOrderbook(ctx, data, pair, a)
	case name == channelDetailOrderbook && !private:
		return e.handleWSDetailOrderbook(ctx, data, pair, a)
	case name == channelLiveTrades && !private:
		return e.handleWSTrade(data, pair, a)
	case name == channelLiveTrades && private:
		return e.handleWSSelfTrade(ctx, data, pair, a)
	case name == channelFundingRate && !private:
		return e.handleWSFundingRate(ctx, data, pair, a)
	case name == channelMyOrders && private:
		return e.handleWSAccountOrder(ctx, respRaw, event, data, pair, a)
	case (name == channelMyTrades || name == channelLiveOrderTrades) && private:
		return e.handleWSMyTrade(data, pair, a)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

// handleWSSubscription delivers a subscription's confirmation to the request awaiting it on the connection it was sent on
func handleWSSubscription(conn websocket.Connection, event string, respRaw []byte) error {
	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}
	op := strings.TrimSuffix(strings.TrimPrefix(event, "bts:"), "scription_succeeded")
	return conn.RequireMatchWithData(op+":"+channel, respRaw)
}

// handleWSError returns the error reported by the server
// Errors do not identify the request or channel which caused them, so failed subscriptions time out
func handleWSError(respRaw []byte) error {
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w `data`: %w", common.ErrParsingWSField, err)
	}
	var wsErr WebsocketError
	if err := json.Unmarshal(data, &wsErr); err != nil {
		return err
	}
	if wsErr.Code != 0 {
		return fmt.Errorf("%w %d: %s", errWebsocketError, wsErr.Code, wsErr.Message)
	}
	return fmt.Errorf("%w: %s", errWebsocketError, wsErr.Message)
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return normaliseSubscriptions(e.Features.Subscriptions).ExpandTemplates(e)
}

// normaliseSubscriptions returns subscriptions with the assets of account channels cleared, since those channels are
// not specific to a market and would otherwise be subscribed to once per asset, and with private channels marked as
// authenticated, since they cannot be subscribed to without a token
func normaliseSubscriptions(l subscription.List) subscription.List {
	normalised := make(subscription.List, len(l))
	for i, s := range l {
		normalised[i] = s
		if s.QualifiedChannel != "" {
			continue
		}
		accountChannel := isAccountChannelName(subscriptionNames[s.Channel]) && s.Asset != asset.Empty
		private := slices.Contains(privateChannels, s.Channel) && !s.Authenticated
		if !accountChannel && !private {
			continue
		}
		c := s.Clone()
		if accountChannel {
			c.Asset = asset.Empty
			c.Pairs = nil
		}
		if private {
			c.Authenticated = true
		}
		normalised[i] = c
	}
	return normalised
}

// ValidateSubscriptions implements subscription.ListValidator
// The order book and detail order book channels both load the order book of their market, one aggregated by price and
// the other holding each order, so the subscriptions generated for a market must not include both
func (e *Exchange) ValidateSubscriptions(l subscription.List) error {
	books := make(map[key.PairAsset]string)
	for _, s := range l {
		// A disabled subscription is never subscribed to, so it cannot conflict
		if !s.Enabled || !isOrderbookSubscription(s) {
			continue
		}
		name := subscriptionNames[s.Channel]
		for _, p := range s.Pairs {
			k := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: s.Asset}
			if existing, ok := books[k]; ok && existing != name {
				return fmt.Errorf("%w for %s %s between %s and %s", subscription.ErrExclusiveSubscription, p, s.Asset, existing, name)
			}
			books[k] = name
		}
	}
	return nil
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isAccountChannel": isAccountChannel,
		// The pairs a subscription names are not formatted for requests, as the enabled pairs are
		"marketSymbol": formatMarketSymbol,
	}).Parse(subTplText)
}

// Subscribe sends websocket messages on a connection to receive data from a list of channels
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubsWithCreds(ctx, conn, subs, "sub")
}

// Unsubscribe sends websocket messages on a connection to stop receiving data from a list of channels
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubsWithCreds(ctx, conn, subs, "unsub")
}

// manageSubsWithCreds subscribes to or unsubscribes from channels
// Subscribing to private channels needs a new token, as a token cannot be reused for later subscriptions, whereas
// unsubscribing only needs the user ID which names the private channels, which is kept from the last token
func (e *Exchange) manageSubsWithCreds(ctx context.Context, conn websocket.Connection, subs subscription.List, op string) error {
	subs = normaliseSubscriptions(subs)
	var errs error
	var creds *WebsocketTokenResponse
	if slices.ContainsFunc(subs, isPrivateSubscription) {
		if userID := e.websocketUserID.Load(); op == "unsub" && userID != 0 {
			creds = &WebsocketTokenResponse{UserID: userID}
		} else {
			creds, errs = e.GetWebsocketToken(ctx)
			if creds != nil {
				e.websocketUserID.Store(creds.UserID)
			}
		}
	}
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, s subscription.List) error { return e.manageSubs(ctx, conn, s, op, creds) }, 1))
}

// isPrivateSubscription returns whether a subscription needs a token, as it is either marked as authenticated or is to
// a private channel
// Expanded subscriptions are not marked by normaliseSubscriptions, since a marked copy would no longer be the
// subscription that the websocket manager tracks
func isPrivateSubscription(s *subscription.Subscription) bool {
	return s.Authenticated || slices.Contains(privateChannels, s.Channel)
}

func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, subs subscription.List, op string, creds *WebsocketTokenResponse) error {
	subs, errs := subs.ExpandTemplates(e)
	for _, s := range subs {
		if op == "sub" && isOrderbookSubscription(s) {
			// A market whose REST orderbook cannot be loaded gets its book when it next changes, so it is still subscribed
			if err := e.seedOrderbook(ctx, s); err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, conn.GetURL(), errSend, err)
				}
			}
		}
		req := WebsocketEventRequest{
			Event: "bts:" + op + "scribe",
			Data: WebsocketEventData{
				Channel: s.QualifiedChannel,
			},
		}
		if isPrivateSubscription(s) {
			if creds == nil {
				return request.ErrAuthRequestFailed
			}
			req.Data.Channel = privateChannelPrefix + req.Data.Channel + "-" + strconv.FormatUint(creds.UserID, 10)
			req.Data.Auth = creds.Token
		}
		_, err := conn.SendMessageReturnResponse(ctx, request.Unset, op+":"+req.Data.Channel, req)
		if err == nil {
			if op == "sub" {
				err = e.Websocket.AddSuccessfulSubscriptions(conn, s)
			} else {
				err = e.Websocket.RemoveSubscriptions(conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

// splitChannel returns the name and market of a channel, and whether it is private
// Private channels are suffixed with the user ID, and market symbols never contain an underscore
func splitChannel(channel string) (name, market string, private bool, err error) {
	if after, ok := strings.CutPrefix(channel, privateChannelPrefix); ok {
		idx := strings.LastIndexByte(after, '-')
		if idx <= 0 {
			return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
		}
		if _, err := strconv.ParseUint(after[idx+1:], 10, 64); err != nil {
			return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
		}
		channel, private = after[:idx], true
	}
	if isAccountChannelName(channel) {
		return channel, "", private, nil
	}
	idx := strings.LastIndexByte(channel, '_')
	if idx <= 0 || idx == len(channel)-1 {
		return "", "", false, fmt.Errorf("%w: %q", errMalformedChannel, channel)
	}
	return channel[:idx], channel[idx+1:], private, nil
}

// marketPair returns the currency pair and asset type of a channel market symbol
func (e *Exchange) marketPair(market string) (currency.Pair, asset.Item, error) {
	a := asset.Spot
	if strings.HasSuffix(market, strings.ToLower(perpetualMarketSuffix)) {
		a = asset.PerpetualContract
	}
	pair, err := e.MatchSymbolWithAvailablePairs(market, a, false)
	if err != nil {
		return currency.EMPTYPAIR, asset.Empty, fmt.Errorf("%w %q: %w", errParsingWSPair, market, err)
	}
	return pair, a, nil
}

// channelName converts global channel names to exchange specific ones
// panics if name is not supported, so should be called within a recover chain
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

// isAccountChannel returns whether a subscription is to a channel which is not specific to a market
func isAccountChannel(s *subscription.Subscription) bool {
	return isAccountChannelName(subscriptionNames[s.Channel])
}

func isAccountChannelName(name string) bool {
	return slices.Contains(accountChannels, name)
}

const subTplText = `
{{- if isAccountChannel $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- channelName $.S }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- with $name := channelName $.S }}
			{{- range $p := $pairs -}}
				{{- $name -}} _ {{- marketSymbol $p -}}
				{{ $.PairSeparator }}
			{{- end -}}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- end }}
`
