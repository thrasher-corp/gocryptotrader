package subscription

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// DefaultKey is the fallback key for AddSuccessfulSubscriptions
type DefaultKey struct {
	Channel string
	Pair    currency.Pair
	Asset   asset.Item
}

// State tracks the status of a subscription channel
type State uint8

const (
	UnknownState       State = iota // UnknownState subscription state is not registered, but doesn't imply Inactive
	SubscribingState                // SubscribingState means channel is in the process of subscribing
	SubscribedState                 // SubscribedState means the channel has finished a successful and acknowledged subscription
	UnsubscribingState              // UnsubscribingState means the channel has started to unsubscribe, but not yet confirmed
)

// Subscription container for streaming subscriptions
type Subscription struct {
	Key     any
	Channel string
	Pair    currency.Pair
	Asset   asset.Item
	Params  map[string]interface{}
	State   State
}

// String implements the Stringer interface for Subscription, giving a human representation of the subscription
func (s *Subscription) String() string {
	return fmt.Sprintf("%s %s %s", s.Channel, s.Asset, s.Pair)
}

// EnsureKeyed sets the default key on a channel if it doesn't have one
// Returns key for convenience
func (s *Subscription) EnsureKeyed() any {
	if s.Key == nil {
		s.Key = DefaultKey{
			Channel: s.Channel,
			Asset:   s.Asset,
			Pair:    s.Pair,
		}
	}
	return s.Key
}
