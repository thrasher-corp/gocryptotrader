package bybit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// eventPublicPath is the path of the event trading public stream; Bybit's docs omit it, and it was found by probing
const eventPublicPath = "/v5/public/event"

// Event trading websocket channels. The public channels are subscribed on the event connection, and a subscription
// names its event contract symbol in Params["symbol"]; an orderbook subscription takes its depth from Levels, 25 by
// default. The private channels are Bybit's event topic names and travel on the private connection
const (
	ChannelEventOrderbook   = "event.orderbook"
	ChannelEventPublicTrade = "event.publicTrade"
	ChannelEventPosition    = "position.event"
	ChannelEventExecution   = "execution.event"
	ChannelEventOrder       = "order.event"
)

// cEvent is the category of event contract executions and orders, which the generic execution and order topics also
// carry
const cEvent = "event"

// EventPublicConnectionSetup returns the setup of the event trading public connection to url. It generates
// subscriptions only for configured event channels, so it connects only when a user subscribes to one
func (e *Exchange) EventPublicConnectionSetup(exch *config.Exchange, url string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   url,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WebsocketConnect,
		GenerateSubscriptions: e.generateEventPublicSubscriptions,
		Subscriber:            e.eventPublicSubscribe,
		Unsubscriber:          e.eventPublicUnsubscribe,
		Handler:               e.wsHandleEventPublicData,
	}
}

// generateEventPublicSubscriptions returns the enabled subscriptions to event public channels, qualified by topic. A
// subscription without a usable symbol is left out with a warning, since Setup disables the configured ones
func (e *Exchange) generateEventPublicSubscriptions() (subscription.List, error) {
	var subs subscription.List
	for _, s := range e.Features.Subscriptions.Enabled() {
		var prefix string
		switch s.Channel {
		case ChannelEventOrderbook:
			prefix = "orderbook." + strconv.Itoa(topicOrderbookDepth(s)) + "."
		case ChannelEventPublicTrade:
			prefix = "publicTrade."
		default:
			continue
		}
		symbol, err := subscriptionSymbol(s)
		if err != nil {
			// Setup disables the configured subscriptions without a symbol, so this one was added since; it is left out
			// rather than allowed to stop every connection
			log.Warnf(log.WebsocketMgr, "%s skipped a subscription: %v", e.Name, err)
			continue
		}
		subs = append(subs, topicSubscription(s, prefix+symbol))
	}
	return subs, nil
}

// eventPublicSubscribe subscribes to topics on the event public connection
func (e *Exchange) eventPublicSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageCommandSubscriptions(ctx, conn, "subscribe", subs)
}

// eventPublicUnsubscribe unsubscribes from topics on the event public connection
func (e *Exchange) eventPublicUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageCommandSubscriptions(ctx, conn, "unsubscribe", subs)
}

// wsHandleEventPublicData relays the event public connection's frames as decoded. Event contracts have no asset or
// currency pair in the package, and their orderbook has one side of payout ratios, so the frames are not processed into
// the orderbook or trade stores
func (e *Exchange) wsHandleEventPublicData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return handleCommandFrame(conn, respRaw)
		}
		return err
	}
	switch {
	case strings.HasPrefix(topic, "orderbook."):
		return e.relayFrame(ctx, respRaw, new(WebsocketEventOrderbookUpdate))
	case strings.HasPrefix(topic, "publicTrade."):
		return e.relayFrame(ctx, respRaw, new(WebsocketEventPublicTradeUpdate))
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// wsHandleEventPrivateData relays the private connection's event contract position, execution and order frames as
// decoded. Event executions and orders arrive on their event topics and, with category event, on the generic
// execution and order topics
func (e *Exchange) wsHandleEventPrivateData(ctx context.Context, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil {
		return err
	}
	switch topic {
	case ChannelEventPosition:
		return e.relayFrame(ctx, respRaw, new(WebsocketEventPositionUpdate))
	case ChannelEventExecution, chanExecution:
		return e.relayFrame(ctx, respRaw, new(WebsocketEventExecutionUpdate))
	case ChannelEventOrder, chanOrder:
		return e.relayFrame(ctx, respRaw, new(WebsocketEventOrderUpdate))
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// isEventContractFrame returns whether a private frame carries event contract data, which the generic execution and
// order topics share with the other categories
func isEventContractFrame(respRaw []byte) bool {
	category, err := jsonparser.GetString(respRaw, "data", "[0]", "category")
	return err == nil && category == cEvent
}
