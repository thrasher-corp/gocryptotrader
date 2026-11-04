package subscription

import (
	"errors"
	"fmt"
	"slices"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// ErrNotExpanded reports a list which does not satisfy the expanded boundary invariant
var ErrNotExpanded = errors.New("subscription list is not expanded")

// ExpandedList is the internal contract for runtime subscriptions which have already been expanded.
//
// A plain List carries both configuration templates (QualifiedChannel == "") and expanded runtime
// subscriptions (QualifiedChannel != ""), so it cannot tell the websocket manager which stage it
// holds. ExpandedList can only be constructed through Expand or NewExpandedList, both of which
// validate the invariant, so an unexpanded list cannot silently reach the strict manager boundary.
//
// A struct with an unexported field is used rather than a named slice type: Go permits conversion
// between named slice types which share an underlying type, so a caller could cast an unexpanded
// List to a named slice type and defeat the boundary. See #2457.
//
// The boundary pipeline is:
//
//	Generate -> Expand -> Validate -> Route/batch -> Subscribe
//
// Expansion runs once, at the boundary, before the subscribe call and before connection capacity is
// allocated; the manager owns it, driven by the exchange-supplied expander (or by the generator, when
// the generator already expanded its output). Subscription identity is preserved from Expand through
// to the exchange subscriber so that reconciliation against the subscription store keeps working.
//
// A filter stage (for external hooks such as a trading bot narrowing its own channels) sits between
// Expand and Validate in the wider design, but is deliberately not wired yet; see the reserved-hook
// note on the manager boundary. See #2446 and #2372 for the ownership rules of expansion results.
type ExpandedList struct {
	list List
}

// Expand expands any templates in l for the exchange, then validates the result as an ExpandedList.
// It is the single expansion entry point for the manager boundary.
func (l List) Expand(e IExchange) (ExpandedList, error) {
	expanded, err := l.ExpandTemplates(e)
	if err != nil {
		return ExpandedList{}, err
	}
	return NewExpandedList(expanded)
}

// NewExpandedList validates an already-expanded list and wraps it for the manager boundary.
// It does not expand anything, so callers which already expanded (for example exchange generators
// which call ExpandTemplates themselves) can adopt the boundary without expanding twice.
//
// The invariant is: no nil entries, every entry has a populated QualifiedChannel, and no entry
// instance appears more than once. An already-expanded list is safe to pass here because
// ExpandTemplates leaves qualified entries untouched.
func NewExpandedList(l List) (ExpandedList, error) {
	seen := make(map[*Subscription]struct{}, len(l))
	for _, s := range l {
		if s == nil {
			return ExpandedList{}, fmt.Errorf("%w: list contains a nil subscription", common.ErrNilPointer)
		}
		if s.QualifiedChannel == "" {
			return ExpandedList{}, fmt.Errorf("%w: %s", ErrNotExpanded, s)
		}
		if _, ok := seen[s]; ok {
			return ExpandedList{}, fmt.Errorf("%w: duplicate subscription instance %s", ErrNotExpanded, s)
		}
		seen[s] = struct{}{}
	}
	return ExpandedList{list: l}, nil
}

// List returns the underlying list for callers which still take a List.
// It returns the same underlying slice without copying, so callers must not mutate it or replace its
// instances; use Clone when an independent copy is required.
func (x ExpandedList) List() List {
	return x.list
}

// Len returns the number of expanded subscriptions, which is the exchange subscription unit count
// used for connection capacity accounting.
func (x ExpandedList) Len() int {
	return len(x.list)
}

// Empty returns true when the list holds no expanded subscriptions
func (x ExpandedList) Empty() bool {
	return len(x.list) == 0
}

// Strings returns a sorted slice of subscription strings
func (x ExpandedList) Strings() []string {
	return x.list.Strings()
}

// Filter returns an ExpandedList containing only the subscriptions for which fn returns true.
// It only drops entries; retained entries keep their identity so that custom keys and reconciliation
// state survive the filter. fn must not replace or mutate the subscription it is given.
// It is the building block for the reserved external filter hook at the manager boundary, which is
// not wired yet; a maintainer can use it when the hook is introduced. See #2457.
func (x ExpandedList) Filter(fn func(*Subscription) bool) ExpandedList {
	n := make(List, 0, len(x.list))
	for _, s := range x.list {
		if fn(s) {
			n = append(n, s)
		}
	}
	return ExpandedList{list: slices.Clip(n)}
}

// TODO(#2446): add a key-preserving Clone once cloneWithKeys lands. Subscription.Clone clears the
// key, which would break reconciliation, so the manager deliberately relies on passing the same
// instances through the boundary rather than cloning them.
