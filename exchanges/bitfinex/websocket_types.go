package bitfinex

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// wsAuthSignature matches the auth request with its reply, which carries no ID; a connection sends it once, before
	// any other request awaiting a reply
	wsAuthSignature = "auth"

	// wsErrorUnsubscribeInvalid and wsErrorNotSubscribed are the errors Bitfinex replies to unsubscribing from a
	// channel it does not have: the first for a channel ID it never gave, the second for one no longer subscribed
	wsErrorUnsubscribeInvalid = 10400
	wsErrorNotSubscribed      = 10401
	// wsErrorDuplicate is the error Bitfinex replies to a connection's second subscription to a channel, naming it
	wsErrorDuplicate = 10301
	// wsErrorAuthNonce is the error Bitfinex replies to an authentication whose nonce it refuses
	wsErrorAuthNonce = 10114

	// marginInfoKeyBase and marginInfoKeySymbol key the account's margin summary and its buying power on a trading pair
	marginInfoKeyBase   = "base"
	marginInfoKeySymbol = "sym"
)

// wsConfRequest configures a connection's flags
type wsConfRequest struct {
	Event string `json:"event"`
	Flags uint64 `json:"flags"`
}

// wsAuthRequest authenticates a connection
type wsAuthRequest struct {
	Event       string `json:"event"`
	APIKey      string `json:"apiKey"`
	AuthPayload string `json:"authPayload"`
	AuthNonce   string `json:"authNonce"`
	AuthSig     string `json:"authSig"`
}

// wsUnsubscribeRequest unsubscribes a channel
type wsUnsubscribeRequest struct {
	Event     string `json:"event"`
	ChannelID uint64 `json:"chanId"`
}

// wsEvent is an object event: a reply to a conf, auth, subscribe or unsubscribe request, an error, or information
type wsEvent struct {
	Event          string `json:"event"`
	Status         string `json:"status"`
	Code           uint64 `json:"code"`
	Message        string `json:"msg"`
	ChannelID      uint64 `json:"chanId"`
	SubscriptionID string `json:"subId"`
}

// PlatformInfoEvent is information Bitfinex sends about the platform: its server restarting, so connections will
// reconnect, or maintenance starting or ending, during which orders can be neither placed nor cancelled
type PlatformInfoEvent struct {
	Code    uint64
	Message string
}

// WebsocketDerivativeStatus is a derivative pair's status, as its status channel streams it
type WebsocketDerivativeStatus struct {
	Pair      currency.Pair
	AssetType asset.Item
	Status    DerivativeStatusResponse
}

// wsDataFrame is a channel event carrying its data alone: [CHANNEL_ID, DATA, TIME]
type wsDataFrame[T any] struct {
	ChannelID uint64     `tuple:"0"`
	Data      T          `tuple:"1"`
	Time      types.Time `tuple:"2"`
}

// UnmarshalJSON decodes a channel event array
func (f *wsDataFrame[T]) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// wsTypedFrame is a channel event carrying a type before its data: [CHANNEL_ID, TYPE, DATA, TIME]
type wsTypedFrame[T any] struct {
	ChannelID uint64     `tuple:"0"`
	Type      string     `tuple:"1"`
	Data      T          `tuple:"2"`
	Time      types.Time `tuple:"3"`
}

// UnmarshalJSON decodes a typed channel event array
func (f *wsTypedFrame[T]) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// wsChannelKey identifies a channel, which Bitfinex numbers per connection
type wsChannelKey struct {
	conn websocket.Connection
	id   uint64
}

// wsPendingSubscription is a connection's subscription awaiting Bitfinex's confirmation, or its channel's renewal
type wsPendingSubscription struct {
	conn websocket.Connection
	sub  *subscription.Subscription
}

// wsChannelRegistry maps each connection's channel IDs to the subscriptions they carry. Subscriptions keep their own
// keys, which the manager matches when it compares the subscriptions it wants with those it has
type wsChannelRegistry struct {
	mu       sync.RWMutex
	channels map[wsChannelKey]*subscription.Subscription
	pending  map[string]wsPendingSubscription
	// parked holds the subscriptions whose channels a renewal has unsubscribed and not yet subscribed again
	parked []wsPendingSubscription
	// dropped holds the requests of subscriptions unsubscribed before Bitfinex confirmed them, and draining the
	// channels such a confirmation brought, which are unsubscribed in turn
	dropped  map[string]struct{}
	draining map[wsChannelKey]struct{}
}

// addPending records a subscription sent with a request ID
func (r *wsChannelRegistry) addPending(subID string, conn websocket.Connection, s *subscription.Subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == nil {
		r.pending = make(map[string]wsPendingSubscription)
	}
	r.pending[subID] = wsPendingSubscription{conn: conn, sub: s}
}

// removePending forgets a subscription request, as Bitfinex has replied to it or never received it
func (r *wsChannelRegistry) removePending(subID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pending, subID)
	delete(r.dropped, subID)
}

// dropPending marks the requests of a connection's subscriptions equal to s as no longer wanted, so that a confirmation
// still to come is unsubscribed rather than mapped
func (r *wsChannelRegistry) dropPending(conn websocket.Connection, s *subscription.Subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()
	match := parkedMatch(conn, s)
	for subID, p := range r.pending {
		if match(p) {
			if r.dropped == nil {
				r.dropped = make(map[string]struct{})
			}
			r.dropped[subID] = struct{}{}
		}
	}
}

// confirm maps the channel ID Bitfinex gave a subscription request to its subscription, however late the confirmation,
// reporting whether it did: the channel of a request dropped meanwhile drains instead
func (r *wsChannelRegistry) confirm(subID string, conn websocket.Connection, channelID uint64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pending[subID]
	if !ok || p.conn != conn {
		return false, fmt.Errorf("%w: subscription %q confirmed as channel %d", errWebsocketChannelUnknown, subID, channelID)
	}
	delete(r.pending, subID)
	k := wsChannelKey{conn: conn, id: channelID}
	if _, ok := r.dropped[subID]; ok {
		delete(r.dropped, subID)
		if r.draining == nil {
			r.draining = make(map[wsChannelKey]struct{})
		}
		r.draining[k] = struct{}{}
		return false, nil
	}
	if r.channels == nil {
		r.channels = make(map[wsChannelKey]*subscription.Subscription)
	}
	r.channels[k] = p.sub
	r.parked = slices.DeleteFunc(r.parked, func(parked wsPendingSubscription) bool { return parked == p })
	return true, nil
}

// isDraining reports whether a connection's channel is draining, its subscription unsubscribed before Bitfinex
// confirmed it
func (r *wsChannelRegistry) isDraining(conn websocket.Connection, channelID uint64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.draining[wsChannelKey{conn: conn, id: channelID}]
	return ok
}

// drained forgets a draining channel Bitfinex has unsubscribed, reporting whether it was draining
func (r *wsChannelRegistry) drained(conn websocket.Connection, channelID uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := wsChannelKey{conn: conn, id: channelID}
	_, ok := r.draining[k]
	delete(r.draining, k)
	return ok
}

// adopt maps a connection's channel to s when it carries a subscription equal to s, reporting whether it does: Bitfinex
// refuses a second subscription to a channel it confirmed after the first's request stopped waiting
func (r *wsChannelRegistry) adopt(conn websocket.Connection, channelID uint64, s *subscription.Subscription) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := wsChannelKey{conn: conn, id: channelID}
	carried, ok := r.channels[k]
	if !ok || !(subscription.ExactKey{Subscription: s}).Match(subscription.ExactKey{Subscription: carried}) {
		return false
	}
	r.channels[k] = s
	r.parked = slices.DeleteFunc(r.parked, func(parked wsPendingSubscription) bool { return parked.conn == conn && parked.sub == s })
	return true
}

// get returns the subscription a connection's channel carries, or nil
func (r *wsChannelRegistry) get(conn websocket.Connection, channelID uint64) *subscription.Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channels[wsChannelKey{conn: conn, id: channelID}]
}

// idOf returns the ID of the connection's channel carrying a subscription
func (r *wsChannelRegistry) idOf(conn websocket.Connection, s *subscription.Subscription) (uint64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := subscription.ExactKey{Subscription: s}
	for k, sub := range r.channels {
		if k.conn == conn && key.Match(subscription.ExactKey{Subscription: sub}) {
			return k.id, true
		}
	}
	return 0, false
}

// channelOf returns the ID of the connection's channel carrying the subscription itself, rather than one equal to it,
// as a renewal follows its own subscription
func (r *wsChannelRegistry) channelOf(conn websocket.Connection, s *subscription.Subscription) (uint64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, sub := range r.channels {
		if k.conn == conn && sub == s {
			return k.id, true
		}
	}
	return 0, false
}

// remove forgets a connection's channel
func (r *wsChannelRegistry) remove(conn websocket.Connection, channelID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.channels, wsChannelKey{conn: conn, id: channelID})
}

// park forgets a connection's channel, which a renewal has unsubscribed, keeping its subscription until the renewal
// subscribes it again
func (r *wsChannelRegistry) park(conn websocket.Connection, channelID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := wsChannelKey{conn: conn, id: channelID}
	if s, ok := r.channels[k]; ok {
		delete(r.channels, k)
		r.parked = append(r.parked, wsPendingSubscription{conn: conn, sub: s})
	}
}

// isParked reports whether a renewal has unsubscribed the connection's channel carrying the subscription itself, and
// not yet subscribed it again
func (r *wsChannelRegistry) isParked(conn websocket.Connection, s *subscription.Subscription) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Contains(r.parked, wsPendingSubscription{conn: conn, sub: s})
}

// unpark forgets the subscriptions equal to s whose channels a renewal has unsubscribed, as they are no longer wanted
func (r *wsChannelRegistry) unpark(conn websocket.Connection, s *subscription.Subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parked = slices.DeleteFunc(r.parked, parkedMatch(conn, s))
}

// parkedMatch matches a parked subscription of a connection equal to s
func parkedMatch(conn websocket.Connection, s *subscription.Subscription) func(wsPendingSubscription) bool {
	key := subscription.ExactKey{Subscription: s}
	return func(p wsPendingSubscription) bool {
		return p.conn == conn && key.Match(subscription.ExactKey{Subscription: p.sub})
	}
}

// pruneClosed forgets the channels and requests of closed connections, whose replacements Bitfinex numbers afresh
func (r *wsChannelRegistry) pruneClosed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.channels {
		if isConnectionClosed(k.conn) {
			delete(r.channels, k)
		}
	}
	for subID, p := range r.pending {
		if isConnectionClosed(p.conn) {
			delete(r.pending, subID)
			delete(r.dropped, subID)
		}
	}
	for k := range r.draining {
		if isConnectionClosed(k.conn) {
			delete(r.draining, k)
		}
	}
	r.parked = slices.DeleteFunc(r.parked, func(p wsPendingSubscription) bool { return isConnectionClosed(p.conn) })
}

// wsSharedChannels holds the spot subscriptions whose channels carry margin's data too, by their requests: spot and
// margin trade the same markets, so a pair enabled for both has one channel
type wsSharedChannels struct {
	requests atomic.Pointer[map[string]*subscription.Subscription]
}

// set replaces the shared channels with those the subscriptions were last generated with, returning the spot
// subscriptions whose channels no longer carry margin's data
func (w *wsSharedChannels) set(requests map[string]*subscription.Subscription) []*subscription.Subscription {
	previous := w.requests.Swap(&requests)
	if previous == nil {
		return nil
	}
	var dropped []*subscription.Subscription
	for request, s := range *previous {
		if _, ok := requests[request]; !ok {
			dropped = append(dropped, s)
		}
	}
	return dropped
}

// carriesMargin reports whether a subscription's channel carries margin's data as well as spot's
func (w *wsSharedChannels) carriesMargin(s *subscription.Subscription) bool {
	if s.Asset != asset.Spot {
		return false
	}
	requests := w.requests.Load()
	if requests == nil {
		return false
	}
	_, ok := (*requests)[s.QualifiedChannel]
	return ok
}

// BalanceInfoResponse is the value of the account's assets, which the websocket sends as it changes
type BalanceInfoResponse struct {
	AssetsUnderManagement float64 `tuple:"0"`
	// NetAssetsUnderManagement is AssetsUnderManagement less the account's liabilities
	NetAssetsUnderManagement float64 `tuple:"1"`
}

// UnmarshalJSON decodes a balance info array
func (b *BalanceInfoResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// wsNotificationSignature returns the signature matching a notification with the input request it answers, from the
// ID of the order or offer it carries, or the client order ID a new order is placed with. A request whose
// notification identifies none is sent alone, and matched by the notification's type
func wsNotificationSignature(n *Notification[json.RawMessage]) string {
	var position string
	switch n.Type {
	case wsOrderNewRequest:
		position = "[2]"
	case wsOrderUpdateRequest, wsOrderCancelRequest, wsFundingOfferCancelRequest:
		position = "[0]"
	case wsOrderCancelMultiRequest, wsFundingOfferNewRequest:
		return n.Type
	default:
		return ""
	}
	id, err := jsonparser.GetInt(n.Data, position)
	if err != nil {
		return ""
	}
	return wsInputSignature(n.Type, strconv.FormatInt(id, 10))
}

// wsInputSignature returns the signature matching an input request with its notification
func wsInputSignature(notificationType, id string) string {
	return notificationType + ":" + id
}
