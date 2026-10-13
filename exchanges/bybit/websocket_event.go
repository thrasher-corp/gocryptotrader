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
)

// eventPublicURL is the event trading public stream; Bybit's docs omit it, and it was found by probing
const eventPublicURL = "wss://stream.bybit.com/v5/public/event"

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

// EventPublicConnectionSetup returns the setup of the event trading public connection. It generates subscriptions
// only for configured event channels, so it connects only when a user subscribes to one
func (e *Exchange) EventPublicConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   eventPublicURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.generateEventPublicSubscriptions,
		Subscriber:            e.eventPublicSubscribe,
		Unsubscriber:          e.eventPublicUnsubscribe,
		Handler:               e.wsHandleEventPublicData,
	}
}

// generateEventPublicSubscriptions returns the enabled subscriptions to event public channels, qualified by topic
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
			return nil, err
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
		return e.relayFrame(ctx, respRaw, new(EventOrderbookFrame))
	case strings.HasPrefix(topic, "publicTrade."):
		return e.relayFrame(ctx, respRaw, new(EventPublicTradeFrame))
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
		return e.relayFrame(ctx, respRaw, new(EventPositionFrame))
	case ChannelEventExecution, chanExecution:
		return e.relayFrame(ctx, respRaw, new(EventExecutionFrame))
	case ChannelEventOrder, chanOrder:
		return e.relayFrame(ctx, respRaw, new(EventOrderFrame))
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// isEventContractFrame returns whether a private frame carries event contract data, which the generic execution and
// order topics share with the other categories
func isEventContractFrame(respRaw []byte) bool {
	category, err := jsonparser.GetString(respRaw, "data", "[0]", "category")
	return err == nil && category == cEvent
}
