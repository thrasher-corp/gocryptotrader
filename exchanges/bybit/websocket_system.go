package bybit

import (
	"context"
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// SystemStatusConnection is the message filter of the system status connection
const SystemStatusConnection = "SYSTEM_STATUS"

const systemStatusPath = "/v5/public/misc/status"

// SystemStatusConnectionSetup returns the setup of a connection to the system status stream at url, which pushes
// platform maintenance and service incident notices (System Status). The connection subscribes only when the
// configured subscriptions include the system.status channel, so it is not made otherwise
func (e *Exchange) SystemStatusConnectionSetup(url string) *websocket.ConnectionSetup {
	checkTimeout, maxLimit := exchange.DefaultWebsocketResponseCheckTimeout, exchange.DefaultWebsocketResponseMaxLimit
	if e.Config != nil {
		checkTimeout, maxLimit = e.Config.WebsocketResponseCheckTimeout, e.Config.WebsocketResponseMaxLimit
	}
	return &websocket.ConnectionSetup{
		URL:                   url,
		ResponseCheckTimeout:  checkTimeout,
		ResponseMaxLimit:      maxLimit,
		Connector:             e.WebsocketConnect,
		GenerateSubscriptions: e.generateSystemStatusSubscriptions,
		Subscriber:            e.SystemStatusSubscribe,
		Unsubscriber:          e.SystemStatusUnsubscribe,
		Handler:               e.wsHandleSystemStatus,
		MessageFilter:         SystemStatusConnection,
	}
}

// generateSystemStatusSubscriptions returns the enabled system.status subscription, if any
func (e *Exchange) generateSystemStatusSubscriptions() (subscription.List, error) {
	for _, s := range e.Features.Subscriptions.Enabled() {
		if s.Channel == chanSystemStatus {
			sub := s.Clone()
			sub.QualifiedChannel = chanSystemStatus
			return subscription.List{sub}, nil
		}
	}
	return nil, nil
}

// SystemStatusSubscribe subscribes the system status connection to subs
func (e *Exchange) SystemStatusSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.sendCommandSubscriptions(ctx, conn, subscribeOperation, qualifyByChannelName(subs))
}

// SystemStatusUnsubscribe unsubscribes the system status connection from subs
func (e *Exchange) SystemStatusUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.sendCommandSubscriptions(ctx, conn, unsubscribeOperation, qualifyByChannelName(subs))
}

// wsHandleSystemStatus handles a message from the system status stream
func (e *Exchange) wsHandleSystemStatus(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return err
	}
	switch topic {
	case "":
		return e.handleNoTopicWebsocketResponse(ctx, conn, respRaw)
	case chanSystemStatus:
		return relayWebsocketResponse[WebsocketSystemStatusUpdate](ctx, e, respRaw)
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}
