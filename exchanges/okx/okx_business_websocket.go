package okx

import (
	"context"
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	// okxBusinessWebsocketURL
	okxBusinessWebsocketURL = "wss://ws.okx.com:8443/ws/v5/business"

	businessConnection = "business"
)

var (
	// defaultBusinessSubscribedChannels list of channels which are subscribed by default
	defaultBusinessSubscribedChannels = []string{
		okxSpreadPublicTrades,
		okxSpreadOrderbook,
		okxSpreadPublicTicker,

		channelPublicStrucBlockTrades,
		channelPublicBlockTrades,
		channelBlockTickers,
	}

	// defaultBusinessAuthChannels list of authenticated channels
	defaultBusinessAuthChannels = []string{
		okxSpreadOrders,
		okxSpreadTrades,
	}
)

// GenerateDefaultBusinessSubscriptions returns a list of default subscriptions to business websocket.
func (e *Exchange) GenerateDefaultBusinessSubscriptions() (subscription.List, error) {
	var subs []string
	var subscriptions []*subscription.Subscription
	subs = append(subs, defaultBusinessSubscribedChannels...)
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		subs = append(subs, defaultBusinessAuthChannels...)
	}
next:
	for c := range subs {
		switch subs[c] {
		case okxSpreadOrders,
			okxSpreadTrades,
			okxSpreadOrderbookLevel1,
			okxSpreadOrderbook,
			okxSpreadPublicTrades,
			okxSpreadPublicTicker:
			pairs, err := e.GetEnabledPairs(asset.Spread)
			if err != nil {
				if errors.Is(err, asset.ErrNotEnabled) {
					continue next
				}
				return nil, err
			}
			for p := range pairs {
				subscriptions = append(subscriptions, &subscription.Subscription{
					Channel: subs[c],
					Asset:   asset.Spread,
					Pairs:   []currency.Pair{pairs[p]},
				})
			}
		case channelPublicBlockTrades,
			channelBlockTickers:
			pairs, err := e.GetEnabledPairs(asset.PerpetualSwap)
			if err != nil {
				if errors.Is(err, asset.ErrNotEnabled) {
					continue next
				}
				return nil, err
			}
			for p := range pairs {
				subscriptions = append(subscriptions, &subscription.Subscription{
					Channel: subs[c],
					Asset:   asset.PerpetualSwap,
					Pairs:   []currency.Pair{pairs[p]},
				})
			}
		default:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: subs[c],
			})
		}
	}
	return subscriptions, nil
}

// BusinessSubscribe sends a websocket subscription request to several channels to receive data.
func (e *Exchange) BusinessSubscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleBusinessSubscription(ctx, conn, operationSubscribe, channelsToSubscribe)
}

// BusinessUnsubscribe sends a websocket unsubscription request to several channels to receive data.
func (e *Exchange) BusinessUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleBusinessSubscription(ctx, conn, operationUnsubscribe, channelsToUnsubscribe)
}

// handleBusinessSubscription sends a subscription and unsubscription information through the business websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (e *Exchange) handleBusinessSubscription(ctx context.Context, conn websocket.Connection, operation string, subscriptions subscription.List) error {
	wsSubscriptionReq := WSSubscriptionInformationList{Operation: operation}
	var channels subscription.List
	var authChannels subscription.List
	for i := 0; i < len(subscriptions); i++ {
		arg := SubscriptionInfo{Channel: subscriptions[i].Channel}

		switch arg.Channel {
		case okxSpreadOrders, okxSpreadTrades, okxSpreadOrderbookLevel1, okxSpreadOrderbook, okxSpreadPublicTrades, okxSpreadPublicTicker:
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.SpreadID = subscriptions[i].Pairs[0].String()
		case channelPublicBlockTrades, channelBlockTickers:
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.InstrumentID = subscriptions[i].Pairs[0]
		}

		if strings.HasPrefix(arg.Channel, candle) || strings.HasPrefix(arg.Channel, indexCandlestick) || strings.HasPrefix(arg.Channel, markPrice) {
			if len(subscriptions[i].Pairs) != 1 {
				return currency.ErrCurrencyPairEmpty
			}
			arg.InstrumentID = subscriptions[i].Pairs[0]
		}

		if ifAny, ok := subscriptions[i].Params["instFamily"]; ok {
			if arg.InstrumentFamily, ok = ifAny.(string); !ok {
				return common.GetTypeAssertError("string", ifAny, "instFamily")
			}
		}

		channels = append(channels, subscriptions[i])
		wsSubscriptionReq.Arguments = append(wsSubscriptionReq.Arguments, arg)
		chunk, err := json.Marshal(wsSubscriptionReq)
		if err != nil {
			return err
		}
		if len(chunk) > maxConnByteLen {
			// remove last addition
			channels = channels[:len(channels)-1]
			wsSubscriptionReq.Arguments = wsSubscriptionReq.Arguments[:len(wsSubscriptionReq.Arguments)-1]
			i--
			if err := conn.SendJSONMessage(ctx, websocketRequestEPL, wsSubscriptionReq); err != nil {
				return err
			}
			if operation == operationUnsubscribe {
				err = e.Websocket.RemoveSubscriptions(conn, channels...)
			} else {
				err = e.Websocket.AddSuccessfulSubscriptions(conn, channels...)
			}
			if err != nil {
				return err
			}
			channels = subscription.List{}
			wsSubscriptionReq.Arguments = []SubscriptionInfo{}
			continue
		}
	}
	if err := conn.SendJSONMessage(ctx, websocketRequestEPL, wsSubscriptionReq); err != nil {
		return err
	}

	if operation == operationUnsubscribe {
		channels = append(channels, authChannels...)
		return e.Websocket.RemoveSubscriptions(conn, channels...)
	}
	channels = append(channels, authChannels...)
	return e.Websocket.AddSuccessfulSubscriptions(conn, channels...)
}
