package websocket

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// TestToExpandedSubscriptionsBoundary ensures the manager only accepts expanded subscriptions at the
// boundary and counts fan-out for capacity. Expansion happens once, before the subscribe call.
func TestToExpandedSubscriptionsBoundary(t *testing.T) {
	t.Parallel()

	t.Run("accepts an already expanded list without replacing instances", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		s := &subscription.Subscription{Channel: "ticker", QualifiedChannel: "ticker-spot"}
		x, err := m.toExpandedSubscriptions(subscription.List{s}, nil)
		require.NoError(t, err, "an expanded list must be accepted")
		require.Equal(t, 1, x.Len())
		assert.Same(t, s, x.List()[0], "the boundary should not replace subscription instances")
	})

	t.Run("rejects an unexpanded list", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		_, err := m.toExpandedSubscriptions(subscription.List{{Channel: "ticker"}}, nil)
		assert.ErrorIs(t, err, subscription.ErrNotExpanded, "an unexpanded list should not reach the boundary")
	})

	t.Run("expands before capacity is counted", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.expandSubscriptions = fanOutExpander(3)
		// One configuration template fans out to three wire subscriptions.
		x, err := m.toExpandedSubscriptions(subscription.List{{Channel: "ticker"}}, nil)
		require.NoError(t, err)
		require.Equal(t, 3, x.Len(), "capacity must be counted after fan-out, not from the template count")
	})

	t.Run("per-connection expander overrides the manager expander", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.expandSubscriptions = fanOutExpander(3)
		expand := func(subscription.List) (subscription.ExpandedList, error) {
			return subscription.NewExpandedList(subscription.List{{Channel: "one", QualifiedChannel: "one"}})
		}
		x, err := m.toExpandedSubscriptions(subscription.List{{Channel: "ticker"}}, expand)
		require.NoError(t, err)
		require.Equal(t, 1, x.Len(), "a per-connection expander must take precedence")
	})

	t.Run("setup adopts the manager expander", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		setup := newDefaultSetup()
		setup.ExpandSubscriptions = fanOutExpander(2)
		require.NoError(t, m.Setup(setup), "Setup must not error")
		x, err := m.toExpandedSubscriptions(subscription.List{{Channel: "x"}}, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, x.Len(), "Setup should adopt the manager-level expander")
	})
}

// fanOutExpander returns an expander which simulates template fan-out by expanding each input
// template into count qualified subscriptions.
func fanOutExpander(count int) func(subscription.List) (subscription.ExpandedList, error) {
	return func(l subscription.List) (subscription.ExpandedList, error) {
		expanded := make(subscription.List, 0, len(l)*count)
		for _, s := range l {
			for i := range count {
				c := s.Clone()
				c.QualifiedChannel = fmt.Sprintf("%s-%d", s.Channel, i)
				expanded = append(expanded, c)
			}
		}
		return subscription.NewExpandedList(expanded)
	}
}
