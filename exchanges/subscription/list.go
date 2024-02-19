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

// PruneNil removes in place any nil Subscriptions from the list
func (l *List) PruneNil() {
	n := slices.Clip(slices.DeleteFunc(*l, func(s *Subscription) bool { return s == nil }))
	*l = n
}
