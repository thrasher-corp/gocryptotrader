package v18_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v18.Version).Exchanges(), "Exchanges should apply the migration to every exchange")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"name":"Binance","api":{"urlEndpoints":{"RestOptionsURL":"https://eapi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com"}}}`,
		`{"name":"Binance","api":{"urlEndpoints":{"RestSpotSupplementaryURL":"https://api.binance.com"}}}`,
		`{"name":"Kraken"}`,
	} {
		out, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
		require.NoError(t, err, "UpgradeExchange must not error")
		assert.Equal(t, input, string(out), "UpgradeExchange should leave the configuration unchanged")
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "default endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestOptionsURL":"https://eapi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com"}}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{"RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com"}}}`,
		},
		{
			name:     "customised endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestOptionsURL":"https://options.example.com"},"authenticatedSupport":true}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{},"authenticatedSupport":true}}`,
		},
		{
			name:     "empty endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestOptionsURL":"","RestSpotURL":"https://sapi.binance.com"}}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
		},
		{
			name:     "null endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com","RestOptionsURL":null}}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
		},
		{
			name:     "repeated endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestOptionsURL":"https://eapi.binance.com","RestSpotURL":"https://sapi.binance.com","RestOptionsURL":"https://options.example.com"}}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
		},
		{
			name:     "other exchange",
			input:    `{"name":"Kraken","api":{"urlEndpoints":{"RestOptionsURL":"https://options.example.com","RestSpotURL":"https://api.kraken.com"}}}`,
			expected: `{"name":"Kraken","api":{"urlEndpoints":{"RestSpotURL":"https://api.kraken.com"}}}`,
		},
		{
			name:     "key outside the endpoints",
			input:    `{"name":"Binance","RestOptionsURL":"https://eapi.binance.com","api":{"RestOptionsURL":"https://eapi.binance.com","urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
			expected: `{"name":"Binance","RestOptionsURL":"https://eapi.binance.com","api":{"RestOptionsURL":"https://eapi.binance.com","urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
		},
		{
			name:     "no options endpoint",
			input:    `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
			expected: `{"name":"Binance","api":{"urlEndpoints":{"RestSpotURL":"https://sapi.binance.com"}}}`,
		},
		{
			name:     "no endpoints",
			input:    `{"name":"Binance","api":{}}`,
			expected: `{"name":"Binance","api":{}}`,
		},
		{
			name:     "no api",
			input:    `{"name":"Binance"}`,
			expected: `{"name":"Binance"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(out), "DowngradeExchange should remove only saved options REST endpoints")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	// The Binance entry holds every endpoint this version saves for Binance
	input := []byte(`{"version":17,"exchanges":[{"name":"Binance","api":{"urlEndpoints":{"EdgeCase1URL":"https://www.binance.com","RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestFuturesSupplementaryURL":"https://papi.binance.com","RestOptionsURL":"https://eapi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com","RestSpotURL":"https://sapi.binance.com","RestUSDTMarginedFuturesURL":"https://fapi.binance.com","WebsocketSpotURL":"wss://stream.binance.com:9443/stream","WebsocketTradeURL":"wss://ws-api.binance.com:443/ws-api/v3"}}},{"name":"Kraken","api":{"urlEndpoints":{"RestSpotURL":"https://api.kraken.com"}}}]}`)
	upgraded, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	expected := `{"version":18,"exchanges":[{"name":"Binance","api":{"urlEndpoints":{"EdgeCase1URL":"https://www.binance.com","RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestFuturesSupplementaryURL":"https://papi.binance.com","RestOptionsURL":"https://eapi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com","RestSpotURL":"https://sapi.binance.com","RestUSDTMarginedFuturesURL":"https://fapi.binance.com","WebsocketSpotURL":"wss://stream.binance.com:9443/stream","WebsocketTradeURL":"wss://ws-api.binance.com:443/ws-api/v3"}}},{"name":"Kraken","api":{"urlEndpoints":{"RestSpotURL":"https://api.kraken.com"}}}]}`
	assert.JSONEq(t, expected, string(upgraded), "Deploy should only advance the version to v18")

	downgraded, err := versions.Manager.Deploy(t.Context(), upgraded, 17)
	require.NoError(t, err, "Deploy must downgrade from v18")
	downgradedExpected := `{"version":17,"exchanges":[{"name":"Binance","api":{"urlEndpoints":{"EdgeCase1URL":"https://www.binance.com","RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestFuturesSupplementaryURL":"https://papi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com","RestSpotURL":"https://sapi.binance.com","RestUSDTMarginedFuturesURL":"https://fapi.binance.com","WebsocketSpotURL":"wss://stream.binance.com:9443/stream","WebsocketTradeURL":"wss://ws-api.binance.com:443/ws-api/v3"}}},{"name":"Kraken","api":{"urlEndpoints":{"RestSpotURL":"https://api.kraken.com"}}}]}`
	assert.JSONEq(t, downgradedExpected, string(downgraded), "Downgrade should remove only the options REST endpoint, which v17 rejects")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 18)
	require.NoError(t, err, "Deploy must reapply v18")
	reupgradedExpected := `{"version":18,"exchanges":[{"name":"Binance","api":{"urlEndpoints":{"EdgeCase1URL":"https://www.binance.com","RestCoinMarginedFuturesURL":"https://dapi.binance.com","RestFuturesSupplementaryURL":"https://papi.binance.com","RestSpotSupplementaryURL":"https://api.binance.com","RestSpotURL":"https://sapi.binance.com","RestUSDTMarginedFuturesURL":"https://fapi.binance.com","WebsocketSpotURL":"wss://stream.binance.com:9443/stream","WebsocketTradeURL":"wss://ws-api.binance.com:443/ws-api/v3"}}},{"name":"Kraken","api":{"urlEndpoints":{"RestSpotURL":"https://api.kraken.com"}}}]}`
	assert.JSONEq(t, reupgradedExpected, string(reupgraded), "Reapplying v18 should leave Binance to restore its options REST endpoint default at setup")
}
