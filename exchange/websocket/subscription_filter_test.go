package websocket

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestSetSubscriptionFilter(t *testing.T) {
	t.Parallel()

	type filterCall struct {
		url      string
		channels []string
	}
	channels := func(subs subscription.List) []string {
		names := make([]string, 0, len(subs))
		for _, sub := range subs {
			names = append(names, sub.Channel)
		}
		slices.Sort(names)
		return names
	}
	recordFilter := func(calls *[]filterCall, keep map[string][]string) subscription.FilterHook {
		return func(connectionURL string, subs subscription.List) subscription.List {
			*calls = append(*calls, filterCall{connectionURL, channels(subs)})
			var kept subscription.List
			for _, sub := range subs {
				if slices.Contains(keep[connectionURL], sub.Channel) {
					kept = append(kept, sub)
				}
			}
			return kept
		}
	}

	t.Run("single connection keeps the default URL across refreshes", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		setup := newDefaultSetup()
		setup.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
		setup.Subscriber = currySimpleSub(m)
		setup.Unsubscriber = currySimpleUnsub(m)
		require.NoError(t, m.Setup(setup), "manager setup must succeed")
		require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
			URL: "wss://example.com/marketdata", ResponseCheckTimeout: time.Second, ResponseMaxLimit: time.Second,
		}), "connection setup must succeed")
		t.Cleanup(func() { cleanupManagerMonitors(t, m) })

		var calls []filterCall
		m.SetSubscriptionFilter(recordFilter(&calls, map[string][]string{setup.DefaultURL: {"TestSub", "TestSub2"}}))
		require.NoError(t, m.Connect(t.Context()), "Connect must subscribe")
		assert.Equal(t, []string{"TestSub", "TestSub2"}, channels(m.GetSubscriptions()), "Connect should keep the selected channels")

		require.NoError(t, m.SetWebsocketURL("wss://example.com/override", false, false), "updating the running URL must succeed")
		require.NoError(t, m.FlushChannels(t.Context()), "refresh must succeed with a different running URL")
		assert.Equal(t, []string{"TestSub", "TestSub2"}, channels(m.GetSubscriptions()), "refresh should keep the original selection")

		m.SetSubscriptionFilter(recordFilter(&calls, map[string][]string{setup.DefaultURL: {"TestSub2", "TestSub3"}}))
		require.NoError(t, m.FlushChannels(t.Context()), "refresh with a replacement filter must succeed")
		assert.Equal(t, []string{"TestSub2", "TestSub3"}, channels(m.GetSubscriptions()), "refresh should apply the replacement filter")

		m.SetSubscriptionFilter(nil)
		require.NoError(t, m.FlushChannels(t.Context()), "refresh without a filter must succeed")
		all := []string{"TestSub", "TestSub2", "TestSub3", "TestSub4"}
		assert.Equal(t, all, channels(m.GetSubscriptions()), "refresh should restore all generated subscriptions")
		assert.Equal(t, []filterCall{
			{setup.DefaultURL, all},
			{setup.DefaultURL, all},
			{setup.DefaultURL, all},
		}, calls, "each active filter call should receive the default URL and the full list")
	})

	t.Run("multi connection filters before splitting subscriptions", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		setup := newDefaultSetup()
		setup.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
		setup.UseMultiConnectionManagement = true
		setup.MaxWebsocketSubscriptionsPerConnection = 1
		require.NoError(t, m.Setup(setup), "manager setup must succeed")

		server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		t.Cleanup(func() { cleanupManagerMonitors(t, m) })
		url := "ws" + server.URL[len("http"):]
		publicURL, otherURL, requestURL := url+"/public", url+"/other", url+"/requests"
		for _, cfg := range []struct {
			url  string
			subs subscription.List
		}{
			{publicURL, subscription.List{{Channel: "book"}, {Channel: "trades"}}},
			{otherURL, subscription.List{{Channel: "ticker"}}},
			{requestURL, nil},
		} {
			require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
				URL:                      cfg.url,
				SubscriptionsNotRequired: cfg.subs == nil,
				Connector: func(ctx context.Context, conn Connection) error {
					return conn.Dial(ctx, dialer, nil, nil)
				},
				GenerateSubscriptions: func() (subscription.List, error) { return cfg.subs.Clone(), nil },
				Subscriber:            currySimpleSubConn(m),
				Unsubscriber:          currySimpleUnsubConn(m),
				Handler:               func(context.Context, Connection, []byte) error { return nil },
			}), "connection setup must succeed")
		}
		connectionCounts := func() map[string]int {
			m.connectionManagerMu.RLock()
			defer m.connectionManagerMu.RUnlock()
			counts := make(map[string]int, len(m.connectionManager))
			for _, ws := range m.connectionManager {
				counts[ws.setup.URL] = len(ws.connections)
			}
			return counts
		}

		var calls []filterCall
		m.SetSubscriptionFilter(recordFilter(&calls, map[string][]string{publicURL: {"book", "trades"}}))
		require.NoError(t, m.Connect(t.Context()), "Connect must subscribe")
		assert.Equal(t, []string{"book", "trades"}, channels(m.GetSubscriptions()), "Connect should keep the selected channels")
		assert.Equal(t, map[string]int{publicURL: 2, otherURL: 0, requestURL: 1}, connectionCounts(),
			"Connect should split retained subscriptions and open the request connection")

		m.SetSubscriptionFilter(recordFilter(&calls, map[string][]string{publicURL: {"trades"}, otherURL: {"ticker"}}))
		require.NoError(t, m.FlushChannels(t.Context()), "refresh must succeed with a replacement filter")
		assert.Equal(t, []string{"ticker", "trades"}, channels(m.GetSubscriptions()), "refresh should reconcile the changed selection")
		assert.Equal(t, 1, connectionCounts()[otherURL], "refresh should open a newly selected connection")
		assert.Equal(t, []filterCall{
			{publicURL, []string{"book", "trades"}},
			{otherURL, []string{"ticker"}},
			{publicURL, []string{"book", "trades"}},
			{otherURL, []string{"ticker"}},
		}, calls, "filter should receive one complete list per subscribing URL on both paths")
	})
}
