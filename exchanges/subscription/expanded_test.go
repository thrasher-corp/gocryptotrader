package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// TestListExpandBoundary exercises the Expand entry point of the manager boundary type
func TestListExpandBoundary(t *testing.T) {
	t.Parallel()

	e := newMockEx()
	e.tpl = "subscriptions.tmpl"

	x, err := List{{Channel: "single-channel"}}.Expand(e)
	require.NoError(t, err, "Expand must not error")
	require.Equal(t, 1, x.Len(), "a mono-channel template must expand to one subscription")
	assert.Equal(t, "single-channel", x.List()[0].QualifiedChannel, "Expand should qualify the channel")

	// A template which fans out to one subscription per enabled pair must report the expanded count
	x, err = List{{Channel: "expand-pairs", Asset: asset.All, Levels: 1}}.Expand(e)
	require.NoError(t, err, "Expand must not error on a fan-out template")
	require.Greater(t, x.Len(), 1, "template fan-out must be visible after expansion")
	for _, s := range x.List() {
		assert.NotEmpty(t, s.QualifiedChannel, "every expanded subscription should be qualified")
	}
}

// TestNewExpandedListValidates ensures the boundary invariant is enforced at construction
func TestNewExpandedListValidates(t *testing.T) {
	t.Parallel()

	s := &Subscription{Channel: "ticker", QualifiedChannel: "ticker-spot"}
	x, err := NewExpandedList(List{s})
	require.NoError(t, err, "a fully qualified list must be accepted")
	assert.Equal(t, 1, x.Len())
	assert.Same(t, s, x.List()[0], "validation should not replace subscription instances")

	_, err = NewExpandedList(List{{Channel: "ticker"}})
	assert.ErrorIs(t, err, ErrNotExpanded, "an unexpanded template should be rejected")

	_, err = NewExpandedList(List{nil})
	assert.ErrorIs(t, err, common.ErrNilPointer, "a nil entry should be rejected")

	_, err = NewExpandedList(List{s, s})
	assert.ErrorIs(t, err, ErrNotExpanded, "a duplicated instance should be rejected")

	x, err = NewExpandedList(List{})
	require.NoError(t, err, "an empty list is valid")
	assert.True(t, x.Empty(), "an empty list should report Empty")
	assert.Empty(t, x.Strings(), "an empty list should have no strings")
}

// TestExpandedListFilterDropsWithoutReplacing ensures filtering only drops entries and keeps identity
func TestExpandedListFilterDropsWithoutReplacing(t *testing.T) {
	t.Parallel()

	a := &Subscription{Channel: "a", QualifiedChannel: "a"}
	b := &Subscription{Channel: "b", QualifiedChannel: "b"}
	x, err := NewExpandedList(List{a, b})
	require.NoError(t, err)

	f := x.Filter(func(s *Subscription) bool { return s != b })
	require.Equal(t, 1, f.Len(), "filter must drop excluded subscriptions")
	assert.Same(t, a, f.List()[0], "filter should retain the instance of kept subscriptions")
	assert.Equal(t, 2, x.Len(), "filter should not mutate the source list")
}
