package deribit

import (
	"context"
	"errors"
	"fmt"
)

// maximumSubscriptionLabelLength is the longest label Deribit adds to private channel notifications
const maximumSubscriptionLabelLength = 16

var (
	errChannelsRequired           = errors.New("at least one channel is required")
	errSubscriptionLabelTooLong   = errors.New("subscription label must be at most 16 characters")
	errUnacknowledgedSubscription = errors.New("channel was not acknowledged")
)

// Subscribe subscribes the connection to public channels and returns the channels Deribit subscribed; Deribit only
// accepts it over the websocket
func (w WebsocketAPI) Subscribe(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errChannelsRequired
	}
	var resp []string
	return resp, w.e.SendWebsocketRequest(ctx, subscribeEPL, "public/subscribe", &subscriptionParams{Channels: channels}, &resp)
}

// SubscribePrivate subscribes the authenticated connection to public and private channels, adding an optional label of
// up to 16 characters to private channel notifications, and returns the channels Deribit subscribed; Deribit only
// accepts it over the websocket
func (w WebsocketAPI) SubscribePrivate(ctx context.Context, arg *PrivateSubscribeRequest) ([]string, error) {
	if arg == nil || len(arg.Channels) == 0 {
		return nil, errChannelsRequired
	}
	if len(arg.Label) > maximumSubscriptionLabelLength {
		return nil, fmt.Errorf("%w: %q", errSubscriptionLabelTooLong, arg.Label)
	}
	var resp []string
	return resp, w.e.SendWebsocketRequest(ctx, subscribeEPL, "private/subscribe", arg, &resp)
}

// Unsubscribe unsubscribes the connection from public channels and returns the channels Deribit unsubscribed, which
// leaves out channels that were invalid or not subscribed; Deribit only accepts it over the websocket
func (w WebsocketAPI) Unsubscribe(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errChannelsRequired
	}
	var resp []string
	return resp, w.e.SendWebsocketRequest(ctx, nonMatchingEPL, "public/unsubscribe", &subscriptionParams{Channels: channels}, &resp)
}

// UnsubscribePrivate unsubscribes the authenticated connection from public and private channels and returns the
// channels Deribit unsubscribed; Deribit only accepts it over the websocket
func (w WebsocketAPI) UnsubscribePrivate(ctx context.Context, channels []string) ([]string, error) {
	if len(channels) == 0 {
		return nil, errChannelsRequired
	}
	var resp []string
	return resp, w.e.SendWebsocketRequest(ctx, nonMatchingEPL, "private/unsubscribe", &subscriptionParams{Channels: channels}, &resp)
}

// UnsubscribeAll unsubscribes the connection from every channel; Deribit only accepts it over the websocket
func (w WebsocketAPI) UnsubscribeAll(ctx context.Context) error {
	return w.e.sendOKRequest(ctx, true, nonMatchingEPL, "public/unsubscribe_all", nil)
}

// UnsubscribeAllPrivate unsubscribes the authenticated connection from every channel; Deribit only accepts it over the
// websocket
func (w WebsocketAPI) UnsubscribeAllPrivate(ctx context.Context) error {
	return w.e.sendOKRequest(ctx, true, nonMatchingEPL, "private/unsubscribe_all", nil)
}
