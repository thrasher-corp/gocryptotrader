package v18

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddUSDTMarginedPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		config   map[string]any
		expected bool
	}{
		{name: "missing currency pairs", config: map[string]any{}},
		{name: "missing pairs", config: map[string]any{"currencyPairs": map[string]any{}}},
		{
			name: "existing pair",
			config: map[string]any{
				"currencyPairs": map[string]any{
					"pairs": map[string]any{"usdtmarginedfutures": "existing"},
				},
			},
			expected: true,
		},
		{
			name: "add pair",
			config: map[string]any{
				"currencyPairs": map[string]any{"pairs": map[string]any{}},
			},
			expected: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			addUSDTMarginedPair(tc.config)
			currencyPairs, ok := tc.config["currencyPairs"].(map[string]any)
			if !tc.expected {
				if ok {
					pairs, _ := currencyPairs["pairs"].(map[string]any)
					assert.NotContains(t, pairs, "usdtmarginedfutures", "pair should not be added without the required parent configuration")
				}
				return
			}
			require.True(t, ok, "currency-pair configuration must exist")
			pairs, ok := currencyPairs["pairs"].(map[string]any)
			require.True(t, ok, "pair configuration must exist")
			pair, found := pairs["usdtmarginedfutures"]
			require.True(t, found, "USDT-margined pair configuration must exist")
			if tc.name == "existing pair" {
				assert.Equal(t, "existing", pair, "existing pair configuration should be preserved")
				return
			}
			pairConfig, ok := pair.(map[string]any)
			require.True(t, ok, "added pair configuration must use an object")
			assert.Equal(t, "BTC-USDT", pairConfig["enabled"], "enabled pair should match")
			assert.Equal(t, "BTC-USDT", pairConfig["available"], "available pair should match")
		})
	}
}

func TestAddDerivativeSubscriptions(t *testing.T) {
	t.Parallel()
	t.Run("missing features", func(t *testing.T) {
		t.Parallel()
		config := map[string]any{}
		addDerivativeSubscriptions(config)
		assert.NotContains(t, config, "features", "missing feature configuration should remain unchanged")
	})

	t.Run("add and preserve subscriptions", func(t *testing.T) {
		t.Parallel()
		config := map[string]any{
			"features": map[string]any{
				"subscriptions": []any{
					nil,
					map[string]any{"enabled": true, "channel": "myAccount", "authenticated": true},
					map[string]any{"enabled": true, "channel": "ticker", "asset": "futures"},
					map[string]any{"enabled": true, "channel": "myOrders", "asset": "futures", "authenticated": true},
				},
			},
		}
		addDerivativeSubscriptions(config)
		features, ok := config["features"].(map[string]any)
		require.True(t, ok, "features must remain an object")
		subscriptions, ok := features["subscriptions"].([]any)
		require.True(t, ok, "subscriptions must remain a list")
		require.Len(t, subscriptions, 31, "existing entries and missing derivative defaults must be retained")
		spotAccount, ok := subscriptions[1].(map[string]any)
		require.True(t, ok, "spot account subscription must remain an object")
		assert.Equal(t, "spot", spotAccount["asset"], "spot account subscription should gain its asset")
		privateCount := 0
		for _, sub := range subscriptions {
			entry, ok := sub.(map[string]any)
			if !ok {
				continue
			}
			assetName, _ := entry["asset"].(string)
			authenticated, _ := entry["authenticated"].(bool)
			if authenticated && assetName != "spot" {
				privateCount++
			}
		}
		assert.Equal(t, 17, privateCount, "all derivative private subscriptions should exist")

		addDerivativeSubscriptions(config)
		assert.Len(t, features["subscriptions"], 31, "repeated migration should not duplicate subscriptions")
	})
	for _, scope := range []string{"", "all", "All", "ALL"} {
		for _, channel := range []string{"ticker", "orderbook", "candles", "allTrades", "myOrders", "myTrades", "myAccount", "orders#*"} {
			t.Run(scope+"/"+channel, func(t *testing.T) {
				t.Parallel()
				entry := map[string]any{"enabled": false, "asset": scope, "channel": channel}
				cfg := map[string]any{"features": map[string]any{"subscriptions": []any{entry}}}
				addDerivativeSubscriptions(cfg)
				assert.Equal(t, "spot", entry["asset"], "legacy wildcard scope should preserve spot coverage on every channel")
				assert.Equal(t, false, entry["enabled"], "disabled user choices should remain disabled")
			})
		}
	}

	for _, scope := range []string{"futures", "Futures", "FUTURES"} {
		t.Run("preserve disabled "+scope, func(t *testing.T) {
			t.Parallel()
			entry := map[string]any{"enabled": false, "asset": scope, "channel": "ticker"}
			cfg := map[string]any{"features": map[string]any{"subscriptions": []any{entry}}}
			addDerivativeSubscriptions(cfg)
			features, ok := cfg["features"].(map[string]any)
			require.True(t, ok, "features must remain an object")
			subs, ok := features["subscriptions"].([]any)
			require.True(t, ok, "subscriptions must remain a list")
			count := 0
			for _, sub := range subs {
				item, ok := sub.(map[string]any)
				require.True(t, ok, "subscription must remain an object")
				if name, _ := item["asset"].(string); strings.EqualFold(name, "futures") && item["channel"] == "ticker" {
					count++
					assert.Equal(t, false, item["enabled"], "capitalised asset scopes should preserve explicit disabled choices")
				}
			}
			assert.Equal(t, 1, count, "equivalent asset spelling should not create a duplicate default")
		})
	}
}
