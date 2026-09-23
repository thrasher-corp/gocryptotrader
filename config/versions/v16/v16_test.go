package v16_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v16 "github.com/thrasher-corp/gocryptotrader/config/versions/v16"
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
			out, err := new(v16.Version).UpgradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeExchange should remove only obsolete buffer settings")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":15,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferEnabled":true,"websocketBufferLimit":0}},{"name":"Gemini","orderbook":{"websocketBufferEnabled":false,"websocketBufferLimit":5}}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	expected := `{"version":16,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true}},{"name":"Gemini","orderbook":{}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should remove buffer settings from every exchange and advance to v16")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 15)
	require.NoError(t, err, "Deploy must downgrade from v16")
	assert.JSONEq(t, `{"version":15,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true}},{"name":"Gemini","orderbook":{}}]}`, string(downgraded), "Downgrade should leave unrecoverable buffer settings absent")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 16)
	require.NoError(t, err, "Deploy must reapply v16")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v16 should preserve the upgraded configuration")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"name":"Kraken","orderbook":{"verificationBypass":true}}`)
	out, err := new(v16.Version).DowngradeExchange(t.Context(), bytes.Clone(input))
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.Equal(t, input, out, "DowngradeExchange should not invent removed settings")
}
