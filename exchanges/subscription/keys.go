package subscription

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// MatchableKey interface should be implemented by Key types which want a more complex matching than a simple key equality check
// The Subscription method allows keys to compare against keys of other types
type MatchableKey interface {
	Match(MatchableKey) bool
	GetSubscription() *Subscription
	String() string
}

// ExactKey is key type for subscriptions where all the pairs in a Subscription must match exactly
type ExactKey struct {
	*Subscription
}

var _ MatchableKey = ExactKey{} // Enforce ExactKey must implement MatchableKey

// GetSubscription returns the underlying subscription
func (k ExactKey) GetSubscription() *Subscription {
	return k.Subscription
}

// String implements Stringer; returns the Asset, Channel and Pairs
// Does not provide concurrency protection on the subscription it points to
func (k ExactKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised ExactKey"
	}
	p := s.Pairs.Format(currency.PairFormat{Uppercase: true, Delimiter: "/"})
	return fmt.Sprintf("%s %s %s", s.Channel, s.Asset, p.Join())
}

// Match implements MatchableKey
// Returns true if the key fields exactly matches the subscription, including all Pairs
// Does not check QualifiedChannel or Params
func (k ExactKey) Match(eachKey MatchableKey) bool {
	if eachKey == nil {
		return false
	}
	eachSub := eachKey.GetSubscription()
	return eachSub != nil &&
		eachSub.Channel == k.Channel &&
		eachSub.Asset == k.Asset &&
		eachSub.Pairs.Equal(k.Pairs) &&
		eachSub.Levels == k.Levels &&
		eachSub.Interval == k.Interval
}

// IgnoringPairsKey is a key type for finding subscriptions to group together for requests
type IgnoringPairsKey struct {
	*Subscription
}

var _ MatchableKey = IgnoringPairsKey{} // Enforce IgnoringPairsKey must implement MatchableKey

// GetSubscription returns the underlying subscription
func (k IgnoringPairsKey) GetSubscription() *Subscription {
	return k.Subscription
}

// String implements Stringer; returns the asset and Channel name but no pairs
func (k IgnoringPairsKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised IgnoringPairsKey"
	}
	return fmt.Sprintf("%s %s", s.Channel, s.Asset)
}

// Match implements MatchableKey
func (k IgnoringPairsKey) Match(eachKey MatchableKey) bool {
	if eachKey == nil {
		return false
	}
	eachSub := eachKey.GetSubscription()

	return eachSub != nil &&
		eachSub.Channel == k.Channel &&
		eachSub.Asset == k.Asset &&
		eachSub.Levels == k.Levels &&
		eachSub.Interval == k.Interval
}

// IgnoringAssetKey is a key type for finding subscriptions to group together for requests
type IgnoringAssetKey struct {
	*Subscription
}

var _ MatchableKey = IgnoringAssetKey{} // Enforce IgnoringAssetKey must implement MatchableKey

// GetSubscription returns the underlying subscription
func (k IgnoringAssetKey) GetSubscription() *Subscription {
	return k.Subscription
}

// String implements Stringer; returns the asset and Channel name but no pairs
func (k IgnoringAssetKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised IgnoringAssetKey"
	}
	return fmt.Sprintf("%s %s", s.Channel, s.Pairs)
}

// Match implements MatchableKey
func (k IgnoringAssetKey) Match(eachKey MatchableKey) bool {
	if eachKey == nil {
		return false
	}
	eachSub := eachKey.GetSubscription()

	return eachSub != nil &&
		eachSub.Channel == k.Channel &&
		eachSub.Pairs.Equal(k.Pairs) &&
		eachSub.Levels == k.Levels &&
		eachSub.Interval == k.Interval
}

// ChannelKey is a key type for finding a single subscription by its channel, this will match first found.
// For use with exchange websocket method GetSubscription.
type ChannelKey struct {
	*Subscription
}

var _ MatchableKey = ChannelKey{} // Enforce ChannelKey must implement MatchableKey

// MustChannelKey is a helper function to create a ChannelKey from a subscription channel
func MustChannelKey(channel string) ChannelKey {
	if channel == "" {
		panic("channel must not be empty")
	}
	return ChannelKey{Subscription: &Subscription{Channel: channel}}
}

// Match implements MatchableKey
func (k ChannelKey) Match(eachKey MatchableKey) bool {
	return k.Subscription.Channel == eachKey.GetSubscription().Channel
}

// GetSubscription returns the underlying subscription
func (k ChannelKey) GetSubscription() *Subscription {
	return k.Subscription
}
