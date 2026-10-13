package bybit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const rfqPublicURL = "wss://stream.bybit.com/v5/public/rfq"

// RFQ trading websocket channels, named as Bybit's topics. The public channel is subscribed on the RFQ connection and
// the private channels travel on the private connection
const (
	ChannelRFQPublicTrades = "rfq.open.public.trades"
	ChannelRFQQuotes       = "rfq.open.quotes"
	ChannelRFQTrades       = "rfq.open.trades"
	ChannelRFQInquiries    = "rfq.open.rfqs"
)

// RFQPublicConnectionSetup returns the setup of the RFQ public connection. It generates a subscription only when the
// public trade channel is configured, so it connects only when a user subscribes to it
func (e *Exchange) RFQPublicConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   rfqPublicURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.generateRFQPublicSubscriptions,
		Subscriber:            e.rfqPublicSubscribe,
		Unsubscriber:          e.rfqPublicUnsubscribe,
		Handler:               e.wsHandleRFQPublicData,
	}
}

// generateRFQPublicSubscriptions returns the enabled subscription to the RFQ public trade channel
func (e *Exchange) generateRFQPublicSubscriptions() (subscription.List, error) {
	var subs subscription.List
	for _, s := range e.Features.Subscriptions.Enabled() {
		if s.Channel == ChannelRFQPublicTrades {
			subs = append(subs, topicSubscription(s, ChannelRFQPublicTrades))
		}
	}
	return subs, nil
}

// rfqPublicSubscribe subscribes to topics on the RFQ public connection
func (e *Exchange) rfqPublicSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageRFQSubscriptions(ctx, conn, "subscribe", subs)
}

// rfqPublicUnsubscribe unsubscribes from topics on the RFQ public connection
func (e *Exchange) rfqPublicUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageRFQSubscriptions(ctx, conn, "unsubscribe", subs)
}

// manageRFQSubscriptions subscribes or unsubscribes topics in batches on the RFQ public connection, which echoes each
// request's ID in its reply. A failed batch does not stop the batches after it; the error names every topic that
// failed
func (e *Exchange) manageRFQSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, batch := range common.Batch(subs, 10) {
		topics := batch.QualifiedChannels()
		req := &SubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: topics}
		resp, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s: %w", operation, strings.Join(topics, ", "), err))
			continue
		}
		var reply SubscriptionResponse
		if err := json.Unmarshal(resp, &reply); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if !reply.Success {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", e.applyTopicResults(conn, operation, batch, topics), reply.ReturnMessage))
			continue
		}
		errs = common.AppendError(errs, e.applyTopicResults(conn, operation, batch, nil))
	}
	return errs
}

// wsHandleRFQPublicData relays the RFQ public connection's trade frames as decoded and passes subscription replies to
// the requests awaiting them
func (e *Exchange) wsHandleRFQPublicData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	switch {
	case err == nil && topic == ChannelRFQPublicTrades:
		return e.relayFrame(ctx, respRaw, new(RFQPublicTradeFrame))
	case err == nil:
		return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
	case !errors.Is(err, jsonparser.KeyPathNotFoundError):
		return err
	}
	var reply SubscriptionResponse
	if err := json.Unmarshal(respRaw, &reply); err != nil {
		return err
	}
	switch reply.Operation {
	case "subscribe", "unsubscribe":
		return conn.RequireMatchWithData(reply.RequestID, respRaw)
	case "pong":
		return nil
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, respRaw)
}

// wsHandleRFQPrivateData relays the private connection's RFQ quote, trade and inquiry frames as decoded
func (e *Exchange) wsHandleRFQPrivateData(ctx context.Context, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil {
		return err
	}
	switch topic {
	case ChannelRFQQuotes:
		return e.relayFrame(ctx, respRaw, new(RFQQuoteFrame))
	case ChannelRFQTrades:
		return e.relayFrame(ctx, respRaw, new(RFQTradeFrame))
	case ChannelRFQInquiries:
		return e.relayFrame(ctx, respRaw, new(RFQInquiryFrame))
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}
