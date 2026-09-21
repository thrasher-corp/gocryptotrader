package v16_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v16 "github.com/thrasher-corp/gocryptotrader/config/versions/v16"
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Deribit adds account subscription",
			input:    `{"name":"Deribit","features":{"subscriptions":[{"enabled":true,"channel":"ticker"}]}}`,
			expected: `{"name":"Deribit","features":{"subscriptions":[{"enabled":true,"channel":"ticker"},{"enabled":true,"channel":"myAccount","authenticated":true}]}}`,
		},
		{
			name:     "OKX adds missing subscriptions",
			input:    `{"name":"Okx","features":{"subscriptions":[{"enabled":true,"channel":"tickers"}]}}`,
			expected: `{"name":"Okx","features":{"subscriptions":[{"enabled":true,"channel":"tickers"},{"enabled":true,"channel":"opt-summary","asset":"options"},{"enabled":true,"channel":"balance_and_position","authenticated":true},{"enabled":true,"channel":"account-greeks","authenticated":true}]}}`,
		},
		{
			name:     "custom disabled subscription is preserved",
			input:    `{"name":"Okx","features":{"subscriptions":[{"enabled":false,"channel":"opt-summary","asset":"options","custom":true}]}}`,
			expected: `{"name":"Okx","features":{"subscriptions":[{"enabled":false,"channel":"opt-summary","asset":"options","custom":true},{"enabled":true,"channel":"balance_and_position","authenticated":true},{"enabled":true,"channel":"account-greeks","authenticated":true}]}}`,
		},
		{
			name:     "missing subscription list uses runtime defaults",
			input:    `{"name":"Deribit","enabled":true}`,
			expected: `{"name":"Deribit","enabled":true}`,
		},
	}
	version := new(v16.Version)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := version.UpgradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(got), "UpgradeExchange should add only missing defaults")
			again, err := version.UpgradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"name":"Okx","features":{"subscriptions":[{"channel":"opt-summary"}]}}`)
	got, err := new(v16.Version).DowngradeExchange(t.Context(), input)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.Equal(t, input, got, "DowngradeExchange should preserve subscriptions")
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.ElementsMatch(t, []string{"Deribit", "Okx"}, new(v16.Version).Exchanges(), "Exchanges should include affected exchanges")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":15,"exchanges":[{"name":"Deribit","features":{"subscriptions":[]}},{"name":"Okx","features":{"subscriptions":[{"enabled":false,"channel":"opt-summary"}]}},{"name":"Kraken","enabled":true}]}`)
	got, err := versions.Manager.Deploy(t.Context(), input, versions.UseLatestVersion)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	assert.JSONEq(t, `{"version":16,"exchanges":[{"name":"Deribit","features":{"subscriptions":[{"enabled":true,"channel":"myAccount","authenticated":true}]}},{"name":"Okx","features":{"subscriptions":[{"enabled":false,"channel":"opt-summary"},{"enabled":true,"channel":"balance_and_position","authenticated":true},{"enabled":true,"channel":"account-greeks","authenticated":true}]}},{"name":"Kraken","enabled":true}]}`, string(got), "Deploy should add missing subscriptions while preserving explicit settings")
}
