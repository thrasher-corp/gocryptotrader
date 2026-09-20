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

	input := []byte(`{"gctscript":{"enabled":true},"logging":{"subloggers":[{"name":"GCTSCRIPT","level":"DEBUG","output":"stdout"},{"name":"DATABASE","level":"INFO","output":"file"},{"name":"gctscript","level":"ERROR","output":"stderr"}]},"keep":true}`)
	out, err := new(v16.Version).UpgradeConfig(t.Context(), input)
	require.NoError(t, err, "UpgradeConfig must not error")
	assert.JSONEq(t, `{"logging":{"subloggers":[{"name":"DATABASE","level":"INFO","output":"file"}]},"keep":true}`, string(out), "UpgradeConfig should remove only GCTScript configuration")
}

func TestUpgradeConfigWithoutLegacyConfiguration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"logging":{"subloggers":null},"keep":true}`)
	out, err := new(v16.Version).UpgradeConfig(t.Context(), input)
	require.NoError(t, err, "UpgradeConfig must not error")
	assert.JSONEq(t, string(input), string(out), "UpgradeConfig should preserve unrelated configuration")
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
