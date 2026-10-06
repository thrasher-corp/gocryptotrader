package v12_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v12 "github.com/thrasher-corp/gocryptotrader/config/versions/v12"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
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
			name:     "no EXMO",
			input:    `{"exchanges":[{"name":"EXMOO"},{"name":"Kraken","enabled":true}]}`,
			expected: `{"exchanges":[{"name":"EXMOO"},{"name":"Kraken","enabled":true}]}`,
		},
		{
			name:  "all EXMO variants",
			input: `{"name":"gocryptotrader","exchanges":[{"name":"EXMO","api":{"credentials":{"key":"one","secret":"two"}}},{"name":"Kraken","enabled":true,"custom":{"keep":"me"}},{"name":"exmo"},{"name":"ExMo"}]}`,
			expected: `{
				"name":"gocryptotrader",
				"exchanges":[{"name":"Kraken","enabled":true,"custom":{"keep":"me"}}]
			}`,
		},
		{
			name:     "unnamed exchange preserved",
			input:    `{"exchanges":[{"enabled":false},{"name":"EXMO"}]}`,
			expected: `{"exchanges":[{"enabled":false}]}`,
		},
		{
			name:     "all exchanges removed",
			input:    `{"exchanges":[{"name":"EXMO"},{"name":"exmo"}]}`,
			expected: `{"exchanges":[]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v12.Version).UpgradeConfig(t.Context(), []byte(tt.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tt.expected, string(out), "UpgradeConfig should remove only EXMO configurations")
		})
	}
}

func TestUpgradeConfigInvalidExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":[42]}`)
	out, err := new(v12.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject an invalid exchange entry")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestUpgradeConfigMalformedExchanges(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":`)
	out, err := new(v12.Version).UpgradeConfig(t.Context(), input)
	require.Error(t, err, "UpgradeConfig must reject malformed exchanges JSON")
	assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	input := []byte(`{"exchanges":[{"name":"Kraken"}]}`)
	out, err := new(v12.Version).DowngradeConfig(t.Context(), bytes.Clone(input))
	require.NoError(t, err, "DowngradeConfig must not error")
	assert.Equal(t, input, out, "DowngradeConfig should not change the config")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":11,"exchanges":[{"name":"EXMO"},{"name":"Kraken","enabled":true}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 12)
	require.NoError(t, err, "Deploy must apply the registered v12 upgrade")
	assert.JSONEq(t, `{"version":12,"exchanges":[{"name":"Kraken","enabled":true}]}`, string(out), "Deploy should remove EXMO and set version 12")
}
