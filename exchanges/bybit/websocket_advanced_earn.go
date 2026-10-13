package bybit

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// advancedEarnPublic serves the Advanced Earn product quotes
const advancedEarnPublic = "wss://stream.bybit.com/v5/public/fp"

// Advanced Earn offer topics; configure a subscription with one of these channels to receive its offers
const (
	AdvancedEarnDualAssetsOffersChannel    = "earn.dualassets.offers"
	AdvancedEarnSmartLeverageOffersChannel = "earn.smartleverage.offers"
	AdvancedEarnDiscountBuyOffersChannel   = "earn.discountbuy.offers"
	AdvancedEarnDoubleWinOffersChannel     = "earn.doublewin.offers"
)

// AdvancedEarnOffersConnection is the message filter of the Advanced Earn offers connection, for finding it with
// Websocket.GetConnection
const AdvancedEarnOffersConnection = "ADVANCED_EARN_OFFERS"

// Advanced Earn stream errors
var (
	errUnhandledAdvancedEarnMessage     = errors.New("unhandled message on the Advanced Earn stream")
	errAdvancedEarnSubscriptionRejected = errors.New("subscription rejected by the Advanced Earn stream")
)

// AdvancedEarnConnectionSetup returns the setup of the public Advanced Earn offers connection. It subscribes only to
// the Advanced Earn channels the exchange configuration enables, none by default, so the connection is not made
// unless a user asks for offers
func (e *Exchange) AdvancedEarnConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   advancedEarnPublic,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.generateAdvancedEarnSubscriptions,
		Subscriber:            e.AdvancedEarnSubscribe,
		Unsubscriber:          e.AdvancedEarnUnsubscribe,
		Handler:               e.wsHandleAdvancedEarnData,
		MessageFilter:         AdvancedEarnOffersConnection,
	}
}

// generateAdvancedEarnSubscriptions returns the enabled Advanced Earn subscriptions in the exchange's configuration
func (e *Exchange) generateAdvancedEarnSubscriptions() (subscription.List, error) {
	var subs subscription.List
	for _, s := range e.Features.Subscriptions {
		if s.Enabled && isAdvancedEarnChannel(s.Channel) {
			subs = append(subs, s.Clone())
		}
	}
	return subs, nil
}

// isAdvancedEarnChannel reports whether a channel is an Advanced Earn offers topic, which only the fp stream serves
func isAdvancedEarnChannel(channel string) bool {
	switch channel {
	case AdvancedEarnDualAssetsOffersChannel, AdvancedEarnSmartLeverageOffersChannel, AdvancedEarnDiscountBuyOffersChannel, AdvancedEarnDoubleWinOffersChannel:
		return true
	}
	return false
}

// AdvancedEarnSubscribe subscribes to Advanced Earn offer topics
func (e *Exchange) AdvancedEarnSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageAdvancedEarnSubscriptions(ctx, conn, "subscribe", subs)
}

// AdvancedEarnUnsubscribe unsubscribes from Advanced Earn offer topics
func (e *Exchange) AdvancedEarnUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageAdvancedEarnSubscriptions(ctx, conn, "unsubscribe", subs)
}

// manageAdvancedEarnSubscriptions sends one request per topic and carries on past a rejected topic: the stream rejects
// a whole request with "Params Error" without naming the topic at fault, so batching would let one bad topic fail the
// rest. The topics take no parameters, so each subscription's channel is its topic
func (e *Exchange) manageAdvancedEarnSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		req := &AdvancedEarnSubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: []string{s.Channel}}
		if err := e.sendAdvancedEarnSubscription(ctx, conn, req); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s: %w", operation, s.Channel, err))
			continue
		}
		var err error
		if operation == "unsubscribe" {
			err = e.Websocket.RemoveSubscriptions(conn, s)
		} else {
			err = e.Websocket.AddSuccessfulSubscriptions(conn, s)
		}
		errs = common.AppendError(errs, err)
	}
	return errs
}

// sendAdvancedEarnSubscription sends a subscription request and checks the stream accepted it
func (e *Exchange) sendAdvancedEarnSubscription(ctx context.Context, conn websocket.Connection, req *AdvancedEarnSubscriptionRequest) error {
	raw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
	if err != nil {
		return err
	}
	var reply AdvancedEarnWebsocketMessage
	if err := json.Unmarshal(raw, &reply); err != nil {
		return err
	}
	if !reply.Success {
		return fmt.Errorf("%w: %s", errAdvancedEarnSubscriptionRejected, reply.ReturnMessage)
	}
	return nil
}

// wsHandleAdvancedEarnData routes a frame of the Advanced Earn offers stream: replies to the requests awaiting them,
// and each topic's offers to the data handler
func (e *Exchange) wsHandleAdvancedEarnData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var msg AdvancedEarnWebsocketMessage
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	switch msg.Topic {
	case "":
		switch msg.Operation {
		case "subscribe", "unsubscribe":
			return conn.RequireMatchWithData(msg.RequestID, respRaw)
		case "ping", "pong":
			return nil
		}
	case AdvancedEarnDualAssetsOffersChannel:
		var offers []DualAssetsOffer
		if err := json.Unmarshal(msg.Data, &offers); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, offers)
	case AdvancedEarnSmartLeverageOffersChannel:
		var offers []SmartLeverageOffer
		if err := json.Unmarshal(msg.Data, &offers); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, offers)
	case AdvancedEarnDiscountBuyOffersChannel:
		var offers []DiscountBuyOffer
		if err := json.Unmarshal(msg.Data, &offers); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, offers)
	case AdvancedEarnDoubleWinOffersChannel:
		var offers []DoubleWinOffer
		if err := json.Unmarshal(msg.Data, &offers); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, offers)
	}
	return fmt.Errorf("%w: %s", errUnhandledAdvancedEarnMessage, respRaw)
}
