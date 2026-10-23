package websocket

import (
	"context"
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

// TestConnectDrivesTheExpandedBoundary ensures Connect validates generated subscriptions at the
// boundary before any connection is opened: an unexpanded list must be refused, an expanded one
// must reach the connector.
func TestConnectDrivesTheExpandedBoundary(t *testing.T) {
	t.Parallel()

	newManager := func(t *testing.T, gen func() (subscription.List, error), connected *bool) *Manager {
		t.Helper()
		m := NewManager()
		setup := newDefaultSetup()
		setup.UseMultiConnectionManagement = true
		require.NoError(t, m.Setup(setup), "Setup must not error")
		require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
			URL:                   "ws://boundary.test",
			Connector:             func(context.Context, Connection) error { *connected = true; return nil },
			Handler:               func(context.Context, Connection, []byte) error { return nil },
			Subscriber:            func(context.Context, Connection, subscription.List) error { return nil },
			Unsubscriber:          func(context.Context, Connection, subscription.List) error { return nil },
			GenerateSubscriptions: gen,
		}), "SetupNewConnection must not error")
		return m
	}

	t.Run("rejects an unexpanded generated list before connecting", func(t *testing.T) {
		t.Parallel()
		var connected bool
		m := newManager(t, func() (subscription.List, error) {
			return subscription.List{{Channel: "ticker"}}, nil
		}, &connected)
		err := m.Connect(t.Context())
		require.ErrorIs(t, err, ErrSubscriptionFailure, "an unexpanded list must fail the subscription generation stage")
		require.ErrorIs(t, err, subscription.ErrNotExpanded, "the boundary must report the unexpanded list")
		assert.False(t, connected, "Connect should not open a connection for an unexpanded list")
	})

	t.Run("accepts an expanded generated list and reaches the connector", func(t *testing.T) {
		t.Parallel()
		var connected bool
		m := newManager(t, func() (subscription.List, error) {
			return subscription.List{{Channel: "ticker", QualifiedChannel: "ticker-spot"}}, nil
		}, &connected)
		err := m.Connect(t.Context())
		require.NotErrorIs(t, err, subscription.ErrNotExpanded, "an expanded list must pass the boundary")
		assert.True(t, connected, "Connect should reach the connector for an expanded list")
	})
}

// TestFlushChannelsDrivesTheExpandedBoundary ensures a refresh regenerates and validates through
// the same boundary instead of trusting the generator.
func TestFlushChannelsDrivesTheExpandedBoundary(t *testing.T) {
	t.Parallel()

	newManager := func(t *testing.T, gen func() (subscription.List, error)) *Manager {
		t.Helper()
		m := NewManager()
		setup := newDefaultSetup()
		setup.UseMultiConnectionManagement = true
		require.NoError(t, m.Setup(setup), "Setup must not error")
		require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
			URL:                   "ws://boundary.test",
			Connector:             func(context.Context, Connection) error { return nil },
			Handler:               func(context.Context, Connection, []byte) error { return nil },
			Subscriber:            func(context.Context, Connection, subscription.List) error { return nil },
			Unsubscriber:          func(context.Context, Connection, subscription.List) error { return nil },
			GenerateSubscriptions: gen,
		}), "SetupNewConnection must not error")
		m.setState(connectedState)
		return m
	}

	t.Run("rejects an unexpanded regenerated list", func(t *testing.T) {
		t.Parallel()
		m := newManager(t, func() (subscription.List, error) {
			return subscription.List{{Channel: "ticker"}}, nil
		})
		err := m.FlushChannels(t.Context())
		require.ErrorIs(t, err, subscription.ErrNotExpanded, "FlushChannels must refuse an unexpanded regenerated list")
	})

	t.Run("accepts an expanded regenerated list", func(t *testing.T) {
		t.Parallel()
		m := newManager(t, func() (subscription.List, error) {
			return subscription.List{{Channel: "ticker", QualifiedChannel: "ticker-spot"}}, nil
		})
		err := m.FlushChannels(t.Context())
		require.NotErrorIs(t, err, subscription.ErrNotExpanded, "FlushChannels must accept an expanded regenerated list")
	})
}
