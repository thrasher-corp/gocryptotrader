package subscription

import (
	"slices"
)

// List is a container of subscription pointers
type List []*Subscription

// Strings returns a sorted slice of subscriptions
func (l List) Strings() []string {
	s := make([]string, len(l))
	for i := range l {
		s[i] = l[i].String()
	}
	slices.Sort(s)
	return s
}

// GroupPairs groups subscriptions which are identical apart from the Pair
func (l List) GroupPairs() (n List) {
	for len(l) > 0 {
		s := l[0]
		key := &IgnoringPairsKey{s}
		n = append(n, s)
		// Remove the first element, and any which match it
		l = slices.DeleteFunc(l[1:], func(eachSub *Subscription) bool {
			if m, ok := eachSub.Key.(MatchableKey); ok {
				if key.Match(m) {
					s.AddPairs(eachSub.Pairs...)
					return true
				}
			}
			return false
		})
	}
	return
}
