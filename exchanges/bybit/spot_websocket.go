package bybit

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func (e *Exchange) handleSpotSubscription(ctx context.Context, conn websocket.Connection, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := e.handleSubscriptions(conn, operation, channelsToSubscribe)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		response, err := conn.SendMessageReturnResponse(ctx, request.Unset, payload.RequestID, payload)
		if err != nil {
			return err
		}
		var resp SubscriptionResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.ReturnMessage)
		}
		if operation == "unsubscribe" {
			err = e.Websocket.RemoveSubscriptions(conn, payload.associatedSubs...)
		} else {
			err = e.Websocket.AddSubscriptions(conn, payload.associatedSubs...)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SpotSubscribe sends a websocket message to receive data from the channel
func (e *Exchange) SpotSubscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleSpotSubscription(ctx, conn, "subscribe", channelsToSubscribe)
}

// SpotUnsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) SpotUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSpotSubscription(ctx, conn, "unsubscribe", channelsToUnsubscribe)
}
