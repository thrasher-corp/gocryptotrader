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
			name:     "missing exchanges",
			input:    `{"name":"gocryptotrader"}`,
			expected: `{"name":"gocryptotrader"}`,
		},
		{
			name:     "non-array exchanges",
			input:    `{"exchanges":null}`,
			expected: `{"exchanges":null}`,
		},
		{
			name:     "no BitMEX",
			input:    `{"exchanges":[{"name":"BitMEXX"},{"name":"Kraken","enabled":true}]}`,
			expected: `{"exchanges":[{"name":"BitMEXX"},{"name":"Kraken","enabled":true}]}`,
		},
		{
			name:  "all BitMEX variants",
			input: `{"name":"gocryptotrader","exchanges":[{"name":"Bitmex","api":{"credentials":{"key":"one","secret":"two"}}},{"name":"Kraken","enabled":true,"custom":{"keep":"me"}},{"name":"bitmex"},{"name":"Gemini"},{"name":"BitMEX"}]}`,
			expected: `{
				"name":"gocryptotrader",
				"exchanges":[{"name":"Kraken","enabled":true,"custom":{"keep":"me"}},{"name":"Gemini"}]
			}`,
		},
		{
			name:     "unnamed exchange preserved",
			input:    `{"exchanges":[{"enabled":false},{"name":"Bitmex"}]}`,
			expected: `{"exchanges":[{"enabled":false}]}`,
		},
		{
			name:     "all exchanges removed",
			input:    `{"exchanges":[{"name":"Bitmex"},{"name":"bitmex"}]}`,
			expected: `{"exchanges":[]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v13.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should remove only BitMEX configurations")
		})
	}
}

func TestUpgradeConfigInvalidExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":[42]}`)
	out, err := new(v13.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject an invalid exchange entry")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestUpgradeConfigInvalidExchangesArray(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":[{"name":"Kraken"},]}`)
	out, err := new(v13.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject invalid exchanges JSON")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestUpgradeConfigMalformedExchanges(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":`)
	out, err := new(v13.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject malformed exchanges JSON")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":[{"name":"Kraken"}]}`)
	out, err := new(v13.Version).DowngradeConfig(t.Context(), bytes.Clone(input))
	require.NoError(t, err, "DowngradeConfig must not error")
	assert.Equal(t, input, out, "DowngradeConfig should not change the config")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":12,"exchanges":[{"name":"Bitmex"},{"name":"Kraken","enabled":true}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 13)
	require.NoError(t, err, "Deploy must apply the registered v13 upgrade")
	assert.JSONEq(t, `{"version":13,"exchanges":[{"name":"Kraken","enabled":true}]}`, string(out), "Deploy should remove BitMEX and set version 13")
}
