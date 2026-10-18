package htx

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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		if operation, channel, ok := strings.Cut(id, ":"); ok && (operation == wsSubOp || operation == wsUnsubOp) {
			return e.wsHandleSubscriptionResponse(conn, operation, channel, respRaw)
		}
		var matched bool
		if conn != nil {
			matched = conn.IncomingWithData(id, respRaw)
		} else {
			matched = e.Websocket.Match.IncomingWithData(id, respRaw)
		}
		if matched {
			return nil
		}
	}
	if cid, err := jsonparser.GetString(respRaw, "cid"); err == nil && conn != nil && conn.IncomingWithData(cid, respRaw) {
		return nil
	}

	if pingValue, err := jsonparser.GetInt(respRaw, "ping"); err == nil {
		return e.wsHandleV1ping(ctx, conn, pingValue)
	}

	if action, err := jsonparser.GetString(respRaw, "action"); err == nil {
		switch action {
		case "ping":
			return e.wsHandleV2ping(ctx, conn, respRaw)
		case wsSubOp, wsUnsubOp:
			channel, err := jsonparser.GetString(respRaw, "ch")
			if err != nil {
				return fmt.Errorf("%w 'ch': %w", common.ErrParsingWSField, err)
			}
			return e.wsHandleSubscriptionResponse(conn, action, channel, respRaw)
		case wsRequestOp:
			ch, err := jsonparser.GetString(respRaw, "ch")
			if err != nil {
				return fmt.Errorf("%w 'ch': %w", common.ErrParsingWSField, err)
			}
			return conn.RequireMatchWithData(ch, respRaw)
		}
	}

	if operation, err := jsonparser.GetString(respRaw, "op"); err == nil {
		switch operation {
		case wsAuthChannel:
			return conn.RequireMatchWithData(wsAuthChannel, respRaw)
		case "ping":
			return e.wsHandleFuturesPing(ctx, conn, respRaw)
		case wsSubOp, wsUnsubOp:
			topic, err := jsonparser.GetString(respRaw, "topic")
			if err != nil {
				return fmt.Errorf("%w 'topic': %w", common.ErrParsingWSField, err)
			}
			return e.wsHandleSubscriptionResponse(conn, operation, topic, respRaw)
		case "notify":
			topic, err := jsonparser.GetString(respRaw, "topic")
			if err != nil {
				return fmt.Errorf("%w 'topic': %w", common.ErrParsingWSField, err)
			}
			s := e.getFuturesPrivateSubscription(conn, topic)
			if s == nil {
				return fmt.Errorf("%w: %q", subscription.ErrNotFound, topic)
			}
			return e.wsHandleChannelMsgs(ctx, s, respRaw)
		}
	}

	if err := getErrResp(respRaw); err != nil {
		return err
	}

	if ch, err := jsonparser.GetString(respRaw, "ch"); err == nil {
		var s *subscription.Subscription
		if conn == nil {
			s = e.Websocket.GetSubscription(ch)
		} else {
			s = e.Websocket.GetSubscription(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: ch}}, connection: conn})
		}
		if s == nil {
			return fmt.Errorf("%w: %q", subscription.ErrNotFound, ch)
		}
		return e.wsHandleChannelMsgs(ctx, s, respRaw)
	}

	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
		Message: e.Name + websocket.UnhandledMessage + string(respRaw),
	})
}

// wsHandleSubscriptionResponse updates registration before the reader can process the next push.
func (e *Exchange) wsHandleSubscriptionResponse(conn websocket.Connection, operation, channel string, raw []byte) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	key := wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: channel}}, connection: conn}
	if sub := e.Websocket.GetSubscription(key); sub != nil {
		if responseErr := getErrResp(raw); responseErr != nil {
			if operation == wsSubOp {
				if err := e.Websocket.RemoveSubscriptions(conn, sub); err != nil {
					return err
				}
			}
		} else if operation == wsUnsubOp {
			if err := e.Websocket.RemoveSubscriptions(conn, sub); err != nil {
				return err
			}
		} else if sub.State() != subscription.SubscribedState {
			if err := sub.SetState(subscription.SubscribedState); err != nil {
				return err
			}
		}
	}
	return conn.RequireMatchWithData(operation+":"+channel, raw)
}

// Match scopes wire-topic lookups to their connection; ordinary ExactKey lookups
// still compare subscription settings during a refresh.
func (k wsSubscriptionKey) Match(other subscription.MatchableKey) bool {
	candidate, ok := other.(wsSubscriptionKey)
	return ok && k.ExactKey != nil && k.Subscription != nil && candidate.ExactKey != nil && candidate.Subscription != nil &&
		k.connection == candidate.connection && k.QualifiedChannel == candidate.QualifiedChannel
}

func (e *Exchange) wsHandleChannelMsgs(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	if s.Authenticated && s.Asset != asset.Spot {
		return e.wsHandleFuturesPrivateMessage(ctx, s, respRaw)
	}
	switch s.Channel {
	case subscription.TickerChannel:
		return e.wsHandleTickerMsg(ctx, s, respRaw)
	case subscription.OrderbookChannel:
		return e.wsHandleOrderbookMsg(ctx, s, respRaw)
	case subscription.CandlesChannel:
		return e.wsHandleCandleMsg(ctx, s, respRaw)
	case subscription.AllTradesChannel:
		return e.wsHandleAllTradesMsg(ctx, s, respRaw)
	case subscription.MyAccountChannel:
		return e.wsHandleMyAccountMsg(ctx, respRaw)
	case subscription.MyOrdersChannel:
		return e.wsHandleMyOrdersMsg(ctx, s, respRaw)
	case subscription.MyTradesChannel:
		return e.wsHandleMyTradesMsg(ctx, s, respRaw)
	}
	return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, s.Channel)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":       channelName,
		"isWildcardChannel": isWildcardChannel,
		"interval":          e.FormatExchangeKlineInterval,
	}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		conn, err := e.getWebsocketConnection(l[0])
		if err != nil {
			return err
		}
		return e.manageSubs(ctx, conn, wsSubOp, l)
	}, 1))
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		conn, err := e.getWebsocketConnection(l[0])
		if err != nil {
			return err
		}
		return e.manageSubs(ctx, conn, wsUnsubOp, l)
	}, 1))
}

func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, op string, subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrBatchingNotSupported
	}
	s := subs[0]
	// Delivered topics no longer exist at HTX, so only their local registration needs removing.
	if op == wsUnsubOp && s.Asset == asset.Futures && !s.Authenticated && len(s.Pairs) == 1 && isExpiredDeliveryContract(s.Pairs[0], time.Now()) {
		return e.Websocket.RemoveSubscriptions(conn, s)
	}
	var req any
	switch {
	case s.Authenticated && s.Asset == asset.Spot:
		req = wsReq{Action: op, Channel: s.QualifiedChannel}
	case s.Authenticated && s.Asset == asset.USDTMarginedFutures:
		contractCode := "*"
		if s.Channel == subscription.MyAccountChannel {
			contractCode = ""
		}
		req = wsV5FuturesSubscriptionRequest{
			Operation:    op,
			Topic:        s.QualifiedChannel,
			ContractCode: contractCode,
		}
	case s.Authenticated:
		req = wsFuturesSubscriptionRequest{Operation: op, Topic: s.QualifiedChannel}
	default:
		if op == wsSubOp {
			// Set the id to the channel so that V1 errors can make it back to us
			req = wsSubReq{ID: wsSubOp + ":" + s.QualifiedChannel, Sub: s.QualifiedChannel}
		} else {
			req = wsSubReq{ID: wsUnsubOp + ":" + s.QualifiedChannel, Unsub: s.QualifiedChannel}
		}
	}
	if op == wsSubOp {
		s.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: s}, connection: conn})
		if err := e.Websocket.AddSubscriptions(conn, s); err != nil {
			return fmt.Errorf("%w: %s; error: %w", websocket.ErrSubscriptionFailure, s, err)
		}
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, request.Unset, op+":"+s.QualifiedChannel, req)
	if err == nil {
		err = getErrResp(respRaw)
	}
	if err != nil {
		if op == wsSubOp && e.Websocket.GetSubscription(s.EnsureKeyed()) == s {
			if removeErr := e.Websocket.RemoveSubscriptions(conn, s); removeErr != nil {
				err = common.AppendError(err, removeErr)
			}
		}
		return fmt.Errorf("%s: %w", s, err)
	}
	if e.Verbose && op == wsSubOp {
		log.Debugf(log.ExchangeSys, "%s Subscribed to %s", e.Name, s)
	}
	return nil
}

func stringToOrderSide(side string) (order.Side, error) {
	switch {
	case strings.Contains(side, "buy"):
		return order.Buy, nil
	case strings.Contains(side, "sell"):
		return order.Sell, nil
	}

	return order.UnknownSide, fmt.Errorf("%w: %s", errUnrecognisedOrderSide, side)
}

func stringToOrderType(oType string) (order.Type, error) {
	switch {
	case strings.Contains(oType, "stop-limit"):
		return order.StopLimit, nil
	case oType == "buy-ioc" || oType == "sell-ioc":
		return order.Limit, nil
	case strings.Contains(oType, orderPriceTypeLimit):
		return order.Limit, nil
	case strings.Contains(oType, "market"):
		return order.Market, nil
	}

	return order.UnknownType, fmt.Errorf("%w: %s", errUnrecognisedOrderType, oType)
}

func getErrResp(msg []byte) error {
	var errCode string
	errMsg, _ := jsonparser.GetString(msg, "err-msg")
	value, valueType, _, err := jsonparser.Get(msg, "err-code")
	switch err {
	case nil:
		switch valueType {
		case jsonparser.String, jsonparser.Number:
			errCode = string(value)
		default:
			return fmt.Errorf("%w 'err-code': unexpected JSON type %s from message: %s", common.ErrParsingWSField, valueType, msg)
		}
	case jsonparser.KeyPathNotFoundError: // Look for a V2 error
		errCodeInt, err := jsonparser.GetInt(msg, "code")
		if errCodeInt == 200 || errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w 'code': %w from message: %s", common.ErrParsingWSField, err, msg)
		}
		errCode = strconv.Itoa(int(errCodeInt))
		errMsg, _ = jsonparser.GetString(msg, "message")
	}
	if errCode != "" && errCode != "0" {
		return fmt.Errorf("%w: %s (%s)", errAPIResponse, errMsg, errCode)
	}
	return nil
}

// channelName converts global channel Names used in config of channel input into exchange channel names
// returns the name unchanged if no match is found
func channelName(s *subscription.Subscription, p ...currency.Pair) string {
	if s.Authenticated && s.Asset != asset.Spot {
		if s.Asset == asset.USDTMarginedFutures {
			switch s.Channel {
			case subscription.MyOrdersChannel:
				return "orders"
			case wsTradeUpdatesChannel:
				return "trade"
			case wsExecutionDetailsChannel:
				return "trade_detail"
			case wsPositionsChannel:
				return "positions"
			case subscription.MyAccountChannel:
				return "account"
			case subscription.MyTradesChannel:
				return "match_orders"
			case wsTriggerOrdersChannel:
				return "algo_orders"
			default:
				panic(subscription.ErrUseConstChannelName)
			}
		}
		switch s.Channel {
		case subscription.MyOrdersChannel:
			return "orders.*"
		case subscription.MyTradesChannel:
			return "matchOrders.*"
		case subscription.MyAccountChannel:
			return "accounts.*"
		case wsPositionsChannel:
			return "positions.*"
		case wsTriggerOrdersChannel:
			return "trigger_order.*"
		default:
			panic(subscription.ErrUseConstChannelName)
		}
	}
	n, ok := subscriptionNames[s.Channel]
	if !ok {
		panic(subscription.ErrUseConstChannelName)
	}
	if strings.Contains(n, "%s") {
		return fmt.Sprintf(n, p[0])
	}
	return n
}

func isWildcardChannel(s *subscription.Subscription) bool {
	return s.Authenticated ||
		s.Channel == subscription.MyTradesChannel ||
		s.Channel == subscription.MyOrdersChannel ||
		s.Channel == subscription.MyAccountChannel
}

// wsConnect establishes an HTX websocket connection for the asset selected by its connection setup.
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
}

// generateSubscriptionsForAsset isolates configured subscriptions for one websocket connection.
func (e *Exchange) generateSubscriptionsForAsset(a asset.Item, private bool) (subscription.List, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return subscription.List{}, nil
		}
		return nil, err
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	var unsupported error
	for _, sub := range e.Features.Subscriptions {
		if sub.Enabled && sub.Asset == a && sub.Authenticated == private {
			if sub.Channel == wsFundingRateChannel || sub.Channel == wsFundingRateTopic {
				unsupported = common.AppendError(unsupported, fmt.Errorf("%w: %w: %s", websocket.ErrSubscriptionPartial, common.ErrNotYetImplemented, sub.Channel))
				continue
			}
			cloned := sub.Clone()
			if a == asset.Futures && !private {
				pairs := cloned.Pairs
				if len(pairs) == 0 {
					var err error
					pairs, err = e.GetEnabledPairs(a)
					if err != nil {
						return nil, err
					}
				}
				for _, pair := range pairs {
					if isExpiredDeliveryContract(pair, time.Now()) {
						continue
					}
					delimiter := ""
					if slices.Contains(validContractExpiryCodes, pair.Quote.Upper().String()) {
						delimiter = "_"
					}
					topicSub := cloned.Clone()
					topicSub.Pairs = currency.Pairs{currency.NewPairWithDelimiter(pair.Base.String(), pair.Quote.String(), delimiter)}
					expanded, errs := subscription.List{topicSub}.ExpandTemplates(e)
					if errs != nil {
						return nil, errs
					}
					for _, expandedSub := range expanded {
						expandedSub.Pairs = currency.Pairs{pair}
					}
					subs = append(subs, expanded...)
				}
				continue
			}
			subs = append(subs, cloned)
		}
	}
	expanded, err := subs.ExpandTemplates(e)
	return expanded, common.AppendError(unsupported, err)
}

// isExpiredDeliveryContract recognises dated contracts after their 08:00 UTC delivery; aliases keep their moving venue semantics.
func isExpiredDeliveryContract(pair currency.Pair, now time.Time) bool {
	if len(pair.Quote.String()) != 6 {
		return false
	}
	delivery, err := time.Parse("060102", pair.Quote.String())
	return err == nil && !now.Before(delivery.Add(8*time.Hour))
}

// getWebsocketConnection resolves legacy subscription calls onto the asset-specific connection.
func (e *Exchange) getWebsocketConnection(sub *subscription.Subscription) (websocket.Connection, error) {
	switch sub.Asset {
	case asset.Spot:
		if sub.Authenticated {
			if e.Websocket.AuthConn != nil {
				return e.Websocket.AuthConn, nil
			}
			return e.Websocket.GetConnection(exchange.WebsocketPrivate)
		}
		if e.Websocket.Conn != nil {
			return e.Websocket.Conn, nil
		}
		return e.Websocket.GetConnection(exchange.WebsocketSpot)
	case asset.Futures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketFutures)
	case asset.CoinMarginedFutures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketCoinMarginedPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketCoinMargined)
	case asset.USDTMarginedFutures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketUSDTMargined)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, sub.Asset)
	}
}

// subscribeConnection subscribes through the connection selected by the websocket manager.
func (e *Exchange) subscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsSubOp, batch)
	}, 1))
}

// unsubscribeConnection unsubscribes through the connection selected by the websocket manager.
func (e *Exchange) unsubscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsUnsubOp, batch)
	}, 1))
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 0}, // Aggregation Levels; 0 is no depth aggregation
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    wsMarketDetailChannel,
	subscription.CandlesChannel:   wsCandlesChannel,
	subscription.OrderbookChannel: wsOrderbookChannel,
	subscription.AllTradesChannel: wsTradesChannel,
	subscription.MyTradesChannel:  wsMyTradesChannel,
	subscription.MyOrdersChannel:  wsMyOrdersChannel,
	subscription.MyAccountChannel: wsMyAccountChannel,
	wsFundingRateChannel:          wsFundingRateTopic,
	wsFundingRateTopic:            wsFundingRateTopic,
}

var defaultFuturesSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsTradeUpdatesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsExecutionDetailsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
}

const subTplText = `
{{- if $.S.Asset }}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- if isWildcardChannel $.S }}
			{{- channelName $.S -}}
		{{- else }}
			{{- range $p := $pairs }}
				{{- channelName $.S $p -}}
				{{- if eq $.S.Channel "candles" -}} . {{- interval $.S.Interval }}{{ end }}
				{{- if eq $.S.Channel "orderbook" -}} .step {{- $.S.Levels }}{{ end }}
				{{ $.PairSeparator }}
			{{- end }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else -}}
	{{ channelName $.S }}
{{- end }}
`

const (
	wsAuthChannel        = "auth"
	wsDateTimeFormatting = "2006-01-02T15:04:05"
	signatureMethod      = "HmacSHA256"
	wsSubOp              = "sub"
	wsUnsubOp            = "unsub"
	wsFundingRateChannel = "fundingRate"
	wsFundingRateTopic   = "public.%s.funding_rate"
)

const (
	wsFuturesURL             = "wss://api.hbdm.com/ws"
	wsFuturesPrivateURL      = "wss://api.hbdm.com/notification"
	wsCoinMarginedURL        = "wss://api.hbdm.com/swap-ws"
	wsCoinMarginedPrivateURL = "wss://api.hbdm.com/swap-notification"
	wsUSDTMarginedURL        = "wss://api.hbdm.com/linear-swap-ws"
	wsUSDTMarginedPrivateURL = "wss://api.hbdm.com/ws/v5/notification"
	wsUSDTMarginedTradeURL   = "wss://api.hbdm.com/linear-swap-trade"
)
