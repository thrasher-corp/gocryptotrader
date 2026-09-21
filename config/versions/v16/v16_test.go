package v16_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v16 "github.com/thrasher-corp/gocryptotrader/config/versions/v16"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{name: "missing logging", input: `{"gctscript":{"enabled":true},"keep":true}`, expected: `{"keep":true}`},
		{name: "non-array subloggers", input: `{"logging":{"subloggers":null},"keep":true}`, expected: `{"logging":{"subloggers":null},"keep":true}`},
		{
			name:     "all GCTScript variants",
			input:    `{"gctscript":{"enabled":true},"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"DEBUG","output":"stdout"},{"name":"DATABASE","level":"INFO","output":"file"},{"name":"gctscript","level":"ERROR","output":"stderr"}]},"keep":true}`,
			expected: `{"logging":{"subloggers":[{"name":"DATABASE","level":"INFO","output":"file"}]},"keep":true}`,
		},
		{name: "unnamed sublogger preserved", input: `{"logging":{"subloggers":[{"level":"INFO"},{"name":"GCTSCRIPT"}]}}`, expected: `{"logging":{"subloggers":[{"level":"INFO"}]}}`},
		{name: "all subloggers removed", input: `{"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"gctscript"}]}}`, expected: `{"logging":{"subloggers":[]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v16.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should remove only GCTScript configuration")
		})
	}
}

func TestUpgradeConfigErrorsReturnOriginalConfig(t *testing.T) {
	t.Parallel()

	for _, inputString := range []string{
		`{"gctscript":{},"logging":{"subloggers":[{"name":"GCTSCRIPT"},]}}`,
		`{"gctscript":{},"logging":{"subloggers":[42]}}`,
		`{"gctscript":{},"logging":{"subloggers":`,
	} {
		input := []byte(inputString)
		out, err := new(v16.Version).UpgradeConfig(t.Context(), input)
		require.Error(t, err, "UpgradeConfig must reject malformed sublogger configuration")
		assert.Equal(t, input, out, "UpgradeConfig should return the original config on error")
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	out, err := new(v16.Version).DowngradeConfig(t.Context(), []byte(`{"keep":true}`))
	require.NoError(t, err, "DowngradeConfig must not error")
	assert.JSONEq(t, `{"keep":true,"gctscript":{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}}`, string(out), "DowngradeConfig should restore the legacy GCTScript defaults")
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":15,"gctscript":{"enabled":true},"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	assert.JSONEq(t, `{"version":16,"logging":{"subloggers":[{"name":"DATABASE"}]}}`, string(out), "Deploy should remove GCTScript configuration and set version 16")

	out, err = versions.Manager.Deploy(t.Context(), out, 15)
	require.NoError(t, err, "Deploy must apply the registered v16 downgrade")
	assert.JSONEq(t, `{"version":15,"logging":{"subloggers":[{"name":"DATABASE"}]},"gctscript":{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}}`, string(out), "Deploy should restore compatible GCTScript defaults and set version 15")
}

func TestRegisteredMultiHopUpgrade(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":12,"gctscript":{"enabled":true},"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist"}},"exchanges":[{"name":"Bitmex"},{"name":"Kraken"}],"logging":{"subloggers":[{"name":"GCTSCRIPT"},{"name":"DATABASE"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply registered migrations from version 12 through version 16")
	assert.JSONEq(t, `{"version":16,"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder"}},"exchanges":[{"name":"Kraken"}],"logging":{"subloggers":[{"name":"DATABASE"}]}}`, string(out), "Deploy should preserve earlier migrations while removing GCTScript")
}
