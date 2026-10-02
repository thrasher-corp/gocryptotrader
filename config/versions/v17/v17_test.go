package v17_test

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v17 "github.com/thrasher-corp/gocryptotrader/config/versions/v17"
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "both settings with explicit choices",
			input:    `{"name":"Kraken","enabled":true,"orderbook":{"verificationBypass":true,"websocketBufferEnabled":true,"websocketBufferLimit":0},"custom":{"keep":"me"}}`,
			expected: `{"name":"Kraken","enabled":true,"orderbook":{"verificationBypass":true},"custom":{"keep":"me"}}`,
		},
		{
			name:     "only limit",
			input:    `{"name":"Binance","orderbook":{"websocketBufferLimit":5,"verificationBypass":false}}`,
			expected: `{"name":"Binance","orderbook":{"verificationBypass":false}}`,
		},
		{
			name:     "only enabled",
			input:    `{"name":"Gemini","orderbook":{"websocketBufferEnabled":false}}`,
			expected: `{"name":"Gemini","orderbook":{}}`,
		},
		{
			name:     "no orderbook",
			input:    `{"name":"Coinbase","enabled":false}`,
			expected: `{"name":"Coinbase","enabled":false}`,
		},
		{
			name:     "null orderbook",
			input:    `{"name":"Bitstamp","orderbook":null}`,
			expected: `{"name":"Bitstamp","orderbook":null}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v17.Version).UpgradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeExchange should remove only obsolete buffer settings")
		})
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
			name:     "restore defaults",
			input:    `{"name":"Kraken","orderbook":{"verificationBypass":true}}`,
			expected: `{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferLimit":5,"websocketBufferEnabled":false}}`,
		},
		{
			name:     "preserve explicit settings",
			input:    `{"name":"Kraken","orderbook":{"websocketBufferLimit":0,"websocketBufferEnabled":true}}`,
			expected: `{"name":"Kraken","orderbook":{"websocketBufferLimit":0,"websocketBufferEnabled":true}}`,
		},
		{
			name:     "restore only missing setting",
			input:    `{"name":"Kraken","orderbook":{"websocketBufferEnabled":true}}`,
			expected: `{"name":"Kraken","orderbook":{"websocketBufferLimit":5,"websocketBufferEnabled":true}}`,
		},
		{
			name:     "no orderbook",
			input:    `{"name":"Coinbase"}`,
			expected: `{"name":"Coinbase"}`,
		},
		{
			name:     "null orderbook",
			input:    `{"name":"Bitstamp","orderbook":null}`,
			expected: `{"name":"Bitstamp","orderbook":null}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v17.Version).DowngradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(out), "DowngradeExchange should restore only missing legacy buffer defaults")
		})
	}
}

func TestDowngradeExchangeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "orderbook without a value", input: `{"name":"Kraken","orderbook":}`},
		{name: "setting without a value", input: `{"name":"Kraken","orderbook":{"websocketBufferLimit":}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := new(v17.Version).DowngradeExchange(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, jsonparser.UnknownValueTypeError, "DowngradeExchange must return the parser error")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":16,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferEnabled":true,"websocketBufferLimit":0}},{"name":"Gemini","orderbook":{"websocketBufferEnabled":false,"websocketBufferLimit":5}}]}`)
	unchanged, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must leave an existing v16 configuration valid")
	assert.Equal(t, input, unchanged, "Deploy should retain v16 buffer settings until the v17 upgrade")

	out, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must apply the registered v17 upgrade")
	expected := `{"version":17,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true}},{"name":"Gemini","orderbook":{}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should remove buffer settings from every exchange and advance to v17")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 16)
	require.NoError(t, err, "Deploy must downgrade from v17")
	assert.JSONEq(t, `{"version":16,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferLimit":5,"websocketBufferEnabled":false}},{"name":"Gemini","orderbook":{"websocketBufferLimit":5,"websocketBufferEnabled":false}}]}`, string(downgraded), "Downgrade should restore the v16 buffer defaults without changing other configuration")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 17)
	require.NoError(t, err, "Deploy must reapply v17")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v17 should preserve the upgraded configuration")
}
