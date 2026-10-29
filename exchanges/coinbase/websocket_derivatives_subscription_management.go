package coinbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// derivativesMaximumSubscriptionLabelLength is the longest label the gateway adds to private channels' notifications
const derivativesMaximumSubscriptionLabelLength = 16

var (
	errDerivativesChannelsRequired         = errors.New("at least one gateway channel is required")
	errDerivativesSubscriptionLabelTooLong = errors.New("gateway subscription label must be at most 16 characters")
)

// Subscribe subscribes the gateway's public streams to public channels and returns the channels the gateway
// subscribed, leaving out an instrument it does not list; the gateway accepts it over the websocket only, on its
// streams. The manager's subscriptions are made through the websocket's subscription methods; these add channels the
// manager does not track
func (w WebsocketAPI) Subscribe(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errDerivativesChannelsRequired
	}
	var resp []string
	return resp, w.sendStreamsRequest(ctx, derivativesSubscribeEPL, "public/subscribe", &DerivativesSubscriptionWireRequest{Channels: channels}, &resp)
}

// SubscribePrivate subscribes the gateway's private connection to channels, adding a label of up to 16 characters to
// private channels' notifications, and returns the channels the gateway subscribed; the gateway accepts it over the
// websocket only
func (w WebsocketAPI) SubscribePrivate(ctx context.Context, req *DerivativesPrivateSubscribeRequest) ([]string, error) {
	if req == nil || len(req.Channels) == 0 {
		return nil, errDerivativesChannelsRequired
	}
	if len(req.Label) > derivativesMaximumSubscriptionLabelLength {
		return nil, fmt.Errorf("%w: %q", errDerivativesSubscriptionLabelTooLong, req.Label)
	}
	var resp []string
	return resp, w.SendRequest(ctx, derivativesSubscribeEPL, "private/subscribe", req, &resp)
}

// Unsubscribe unsubscribes the gateway's public streams from channels and returns the channels the gateway names in
// its reply, which can include channels that were not subscribed; the gateway accepts it over the websocket only, on
// its streams
func (w WebsocketAPI) Unsubscribe(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errDerivativesChannelsRequired
	}
	var resp []string
	return resp, w.sendStreamsRequest(ctx, derivativesNonMatchingEPL, "public/unsubscribe", &DerivativesSubscriptionWireRequest{Channels: channels}, &resp)
}

// UnsubscribePrivate unsubscribes the gateway's private connection from channels and returns the channels the gateway
// unsubscribed; the gateway accepts it over the websocket only
func (w WebsocketAPI) UnsubscribePrivate(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errDerivativesChannelsRequired
	}
	var resp []string
	return resp, w.SendRequest(ctx, derivativesNonMatchingEPL, "private/unsubscribe", &DerivativesSubscriptionWireRequest{Channels: channels}, &resp)
}

// UnsubscribeAll unsubscribes the gateway's public streams from every channel, including the manager's subscriptions;
// the gateway accepts it over the websocket only, on its streams
func (w WebsocketAPI) UnsubscribeAll(ctx context.Context) error {
	var resp string
	if err := w.sendStreamsRequest(ctx, derivativesNonMatchingEPL, "public/unsubscribe_all", nil, &resp); err != nil {
		return err
	}
	if resp != derivativesResultOK {
		return fmt.Errorf("%w: public/unsubscribe_all returned %q", errDerivativesUnexpectedResult, resp)
	}
	return nil
}

// UnsubscribeAllPrivate unsubscribes the gateway's private connection from every channel, including the manager's
// subscriptions; the gateway accepts it over the websocket only
func (w WebsocketAPI) UnsubscribeAllPrivate(ctx context.Context) error {
	return w.e.GlobalDerivatives().sendOKRequest(ctx, true, "private/unsubscribe_all", nil)
}

// sendStreamsRequest sends a JSON-RPC request over the gateway's public streams, which take its public subscription
// methods; the websocket connects them while it carries a public gateway channel
func (w WebsocketAPI) sendStreamsRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	conn, err := w.e.Websocket.GetConnection(derivativesStreamsConnection)
	if err != nil {
		return err
	}
	return w.sendRequestOnConnection(ctx, conn, epl, method, params, result)
}
