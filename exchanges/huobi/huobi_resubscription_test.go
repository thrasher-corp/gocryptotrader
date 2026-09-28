package huobi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// TestSubscribeQualifiedListReconciliation pins the manager's legacy connect sequence for an already
// fully qualified list: the manager expands the configured subscriptions, subscribes with them, and
// then checks that every one of those exact subscriptions is present in the subscription store.
// huobi keys subscriptions by their channel string, so the subscription handed to Subscribe has to be
// the one which lands in the store. If Subscribe re-expands the list it stores copies instead and the
// manager's check fails with ErrSubscriptionsNotAdded, breaking connect and reconnect. See #2372
func TestSubscribeQualifiedListReconciliation(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.Features.Subscriptions = subscription.List{{
		Enabled: true,
		Asset:   asset.Spot,
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{btcusdtPair},
	}}

	// The manager expands the configured subscriptions once before subscribing (manager.GenerateSubs)
	subs, err := h.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "Must get a single qualified subscription")
	require.NotEmpty(t, subs[0].QualifiedChannel, "Subscription must be fully qualified")

	// The manager subscribes with that list (manager.SubscribeToChannels) and then reconciles the very
	// same list against the subscription store (manager.go: m.subscriptions.Missing(subs))
	require.NoError(t, h.Websocket.SubscribeToChannels(t.Context(), nil, subs), "SubscribeToChannels must not error")

	var missing int
	for _, s := range subs {
		if h.Websocket.GetSubscription(s) == nil {
			missing++
		}
	}
	assert.Zerof(t, missing, "Every subscription the manager passed should reconcile against the store; got %d missing", missing)
}
