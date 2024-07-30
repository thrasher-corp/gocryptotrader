package subscription

import (
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// EqualLists is a utility function to compare subscription lists and show a pretty failure message
// It overcomes the verbose depth of assert.ElementsMatch spewConfig
// Duplicate of exchange/subscription/subscription:equalList
func EqualLists(tb testing.TB, a, b subscription.List) bool {
	tb.Helper()
	for _, sub := range append(a, b...) {
		sub.Key = &StrictKey{&subscription.ExactKey{Subscription: sub}}
	}
	s, err := subscription.NewStoreFromList(a)
	require.NoError(tb, err, "NewStoreFromList must not error")
	added, missing := s.Diff(b)
	if len(added) > 0 || len(missing) > 0 {
		fail := "Differences:"
		if len(added) > 0 {
			fail = fail + "\n + " + strings.Join(added.Strings(), "\n + ")
		}
		if len(missing) > 0 {
			fail = fail + "\n - " + strings.Join(missing.Strings(), "\n - ")
		}
		assert.Fail(tb, fail, "Subscriptions should be equal")
		return false
	}
	return true
}

// StrictKey is key type for subscriptions where all the pairs, QualifiedChannel and Params in a Subscription must match exactly
type StrictKey struct {
	*subscription.ExactKey
}

var _ subscription.MatchableKey = StrictKey{} // Enforce StrictKey must implement MatchableKey

// Match implements MatchableKey
// Returns true if the key fields exactly matches the subscription, including all Pairs, QualifiedChannel and Params
func (k StrictKey) Match(eachKey subscription.MatchableKey) bool {
	if !k.ExactKey.Match(eachKey) {
		return false
	}
	eachSub := eachKey.GetSubscription()
	return eachSub.QualifiedChannel == k.QualifiedChannel &&
		maps.Equal(eachSub.Params, k.Params)
}

// String implements Stringer; returns the Asset, Channel and Pairs
// Does not provide concurrency protection on the subscription it points to
func (k StrictKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised StrictKey"
	}
	return s.QualifiedChannel + " " + subscription.ExactKey{Subscription: s}.String()
}
