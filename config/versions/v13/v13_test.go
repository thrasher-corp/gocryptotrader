package v13_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v13 "github.com/thrasher-corp/gocryptotrader/config/versions/v13"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hobbyist",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist","apiKey":"keep"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder","apiKey":"keep"}}}`,
		},
		{
			name:     "normalised standard",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":" Standard "}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth"}}}`,
		},
		{
			name:     "legacy placeholder",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":" AccountPlan "}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"basic"}}}`,
		},
		{
			name:     "empty plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":""}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"basic"}}}`,
		},
		{
			name:     "current plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth"}}}`,
		},
		{
			name:     "unknown plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"custom"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"custom"}}}`,
		},
		{
			name:     "missing account plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"apiKey":"keep"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"apiKey":"keep"}}}`,
		},
		{
			name:     "non-string account plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v13.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should return the correct config")

			outAgain, err := new(v13.Version).UpgradeConfig(t.Context(), out)
			require.NoError(t, err, "UpgradeConfig must not error when repeated")
			assert.Equal(t, out, outAgain, "UpgradeConfig should not change an upgraded config")
		})
	}
}

func TestUpgradeConfigMalformed(t *testing.T) {
	t.Parallel()
	input := []byte(`{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":`)
	out, err := new(v13.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject malformed JSON")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "builder",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder","apiKey":"keep"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist","apiKey":"keep"}}}`,
		},
		{
			name:     "normalised growth",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":" Growth "}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"standard"}}}`,
		},
		{
			name:     "unaffected plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"startup"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"startup"}}}`,
		},
		{
			name:     "basic remains basic",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"basic"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"basic"}}}`,
		},
		{
			name:     "missing account plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"apiKey":"keep"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"apiKey":"keep"}}}`,
		},
		{
			name:     "non-string account plan",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
		},
		{
			name:    "malformed",
			input:   `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":`,
			wantErr: true,
		},
		{
			name:    "invalid string escape",
			input:   `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"\x"}}}`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(tc.input)
			out, err := new(v13.Version).DowngradeConfig(t.Context(), input)
			if tc.wantErr {
				require.Error(t, err, "DowngradeConfig must reject malformed input")
				assert.Equal(t, input, out, "DowngradeConfig should return the original config on error")
				return
			}
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "DowngradeConfig should return the correct config")

			outAgain, err := new(v13.Version).DowngradeConfig(t.Context(), out)
			require.NoError(t, err, "DowngradeConfig must not error when repeated")
			assert.Equal(t, out, outAgain, "DowngradeConfig should not change a downgraded config")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":12,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist","apiKey":"keep"}}}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 13)
	require.NoError(t, err, "Deploy must not error during the registered v13 upgrade")
	assert.JSONEq(t, `{"version":13,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder","apiKey":"keep"}}}`, string(out), "Deploy should return the correct v13 upgrade")

	input = []byte(`{"version":13,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth","apiKey":"keep"}}}`)
	out, err = versions.Manager.Deploy(t.Context(), bytes.Clone(input), 12)
	require.NoError(t, err, "Deploy must not error during the registered v13 downgrade")
	assert.JSONEq(t, `{"version":12,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"standard","apiKey":"keep"}}}`, string(out), "Deploy should return the correct v13 downgrade")

	input = []byte(`{"version":12,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"accountPlan","apiKey":"keep"}}}`)
	out, err = versions.Manager.Deploy(t.Context(), input, 13)
	require.NoError(t, err, "Deploy must not error when migrating the legacy account plan placeholder")
	assert.JSONEq(t, `{"version":13,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"basic","apiKey":"keep"}}}`, string(out), "Deploy should return the correct placeholder migration")
}
