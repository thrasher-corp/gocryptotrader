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

func TestRegisteredExchangeMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":15,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferEnabled":true,"websocketBufferLimit":0}},{"name":"Gemini","orderbook":{"websocketBufferEnabled":false,"websocketBufferLimit":5}}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	expected := `{"version":16,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true}},{"name":"Gemini","orderbook":{}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should remove buffer settings from every exchange and advance to v16")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 15)
	require.NoError(t, err, "Deploy must downgrade from v16")
	assert.JSONEq(t, `{"version":15,"exchanges":[{"name":"Kraken","orderbook":{"verificationBypass":true,"websocketBufferLimit":5,"websocketBufferEnabled":false}},{"name":"Gemini","orderbook":{"websocketBufferLimit":5,"websocketBufferEnabled":false}}],"gctscript":{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}}`, string(downgraded), "Downgrade should restore the v15 buffer and GCTScript defaults")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 16)
	require.NoError(t, err, "Deploy must reapply v16")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v16 should preserve the upgraded configuration")
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
			out, err := new(v16.Version).DowngradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.expected, string(out))
		})
	}
}

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
		exact    bool
	}{
		{name: "missing logging", input: `{"gctscript":{"enabled":true},"keep":true}`, expected: `{"keep":true}`},
		{name: "non-array subloggers", input: `{"logging":{"subloggers":null},"keep":true}`, expected: `{"logging":{"subloggers":null},"keep":true}`},
		{
			name:     "subloggers preserved",
			input:    `{"gctscript":{"enabled":true},"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"DEBUG","output":"stdout"},{"name":"DATABASE","level":"INFO","output":"file"},{"name":"gctscript","level":"ERROR","output":"stderr"}]},"keep":true}`,
			expected: `{"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"DEBUG","output":"stdout"},{"name":"DATABASE","level":"INFO","output":"file"},{"name":"gctscript","level":"ERROR","output":"stderr"}]},"keep":true}`,
		},
		{
			name:     "duplicate subloggers preserved",
			input:    `{"gctscript":{},"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"ERROR","output":"console"},{"name":"DATABASE","level":"ERROR","output":"console"}],"subloggers":[{"name":"DATABASE"}]}}`,
			expected: `{"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"ERROR","output":"console"},{"name":"DATABASE","level":"ERROR","output":"console"}],"subloggers":[{"name":"DATABASE"}]}}`,
			exact:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v16.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			if tc.exact {
				assert.Equal(t, tc.expected, string(out), "UpgradeConfig should preserve duplicate sublogger arrays exactly")
			} else {
				assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should remove only the root GCTScript configuration")
			}
		})
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	out, err := new(v16.Version).DowngradeConfig(t.Context(), []byte(`{"keep":true}`))
	require.NoError(t, err, "DowngradeConfig must not error")
	assert.JSONEq(t, `{"keep":true,"gctscript":{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}}`, string(out), "DowngradeConfig should restore the legacy GCTScript defaults")
}

func TestRegisteredConfigMigration(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":15,"gctscript":{"enabled":true},"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	assert.JSONEq(t, `{"version":16,"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`, string(out), "Deploy should remove the root GCTScript configuration and set version 16")

	out, err = versions.Manager.Deploy(t.Context(), out, 15)
	require.NoError(t, err, "Deploy must apply the registered v16 downgrade")
	assert.JSONEq(t, `{"version":15,"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]},"gctscript":{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}}`, string(out), "Deploy should restore compatible GCTScript defaults and set version 15")
}

func TestRegisteredMultiHopUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":12,"gctscript":{"enabled":true},"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist"}},"exchanges":[{"name":"Bitmex"},{"name":"Kraken"}],"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply registered migrations from version 12 through version 16")
	assert.JSONEq(t, `{"version":16,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder"}},"exchanges":[{"name":"Kraken"}],"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`, string(out), "Deploy should preserve earlier migrations while removing the root GCTScript configuration")
}
