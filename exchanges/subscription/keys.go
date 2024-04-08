package subscription

// AllPairsKey is key type for subscriptions where all the pairs in a Subscription must match exactly
type AllPairsKey struct {
	*Subscription
}

var _ MatchableKey = AllPairsKey{} // Enforce AllPairsKey must implement MatchableKey

// Match implements MatchableKey
// Returns true if the key fields exactly matches the subscription, including all Pairs
func (a AllPairsKey) Match(eachSubKey any) bool {
	eachSub, ok := eachSubKey.(*Subscription)
	if !ok {
		return false
	}

	switch {
	case eachSub.Channel != a.Channel,
		eachSub.Asset != a.Asset,
		eachSub.Pairs.ContainsAll(a.Pairs, true) != nil,
		a.Pairs.ContainsAll(eachSub.Pairs, true) != nil,
		eachSub.Levels != a.Levels,
		eachSub.Interval != a.Interval:
		return false
	}

	return true
}

// PointerKey is key type for subscriptions where we know the exact sub we want to remove
// This is useful during Unsubscribe, when you might have concurrently reduced 2 subscriptions to having no Pairs
type PointerKey struct {
	subscription *Subscription
}

var _ MatchableKey = PointerKey{} // Enforce PointerKey must implement MatchableKey

// Match implements MatchableKey
// Returns true if the key is the same pointer address
func (p PointerKey) Match(eachSubKey any) bool {
	eachSub, ok := eachSubKey.(*Subscription)
	if !ok {
		return false
	}
	return eachSub == p.subscription
}
