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

// GroupPairs groups subscriptions which are identical apart from the Pairs
// The returned List contains cloned Subscriptions, and the original Subscriptions are left alone
func (l List) GroupPairs() (n List) {
	src := slices.Clone(l)
	for len(src) > 0 {
		s := src[0].Clone()
		key := &IgnoringPairsKey{s}
		n = append(n, s)
		// Remove the first element, and any which match it
		src = slices.DeleteFunc(src[1:], func(eachSub *Subscription) bool {
			if key.Match(&IgnoringPairsKey{eachSub}) {
				s.AddPairs(eachSub.Pairs...)
				return true
			}
			return false
		})
	}
	return
}
