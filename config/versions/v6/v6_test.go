package v6_test

import (
	"bytes"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v6 "github.com/thrasher-corp/gocryptotrader/config/versions/v6"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	in := []byte(`
{"portfolioAddresses":{"addresses":[{"Address":"1JCe8z4jJVNXSjohjM4i9Hh813dLCNx2Sy","CoinType":"BTC","Balance":0.00108832,"Description":"","WhiteListed":false,"ColdStorage":false,"SupportedExchanges":""}]}}
	`)

	r, err := new(v6.Version).UpgradeConfig(t.Context(), in)
	require.NoError(t, err, "UpgradeConfig must not error")
	require.True(t, bytes.Contains(r, v6.DefaultConfig))

	r2, err := new(v6.Version).UpgradeConfig(t.Context(), r)
	require.NoError(t, err, "UpgradeConfig must not error")
	assert.Equal(t, r, r2, "UpgradeConfig should not affect an already upgraded config")
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	in := []byte(`
{"portfolioAddresses":{"addresses":[{"Address":"1JCe8z4jJVNXSjohjM4i9Hh813dLCNx2Sy","CoinType":"BTC","Balance":0.00108832,"Description":"","WhiteListed":false,"ColdStorage":false,"SupportedExchanges":""}],"providers":[{"name":"Ethplorer","enabled":true},{"name":"XRPScan","enabled":true},{"name":"CryptoID","enabled":false,"apiKey":"Key"}]}}
`)

	r, err := new(v6.Version).DowngradeConfig(t.Context(), in)
	require.NoError(t, err, "DowngradeConfig must not error")
	_, _, _, err = jsonparser.Get(r, "portfolioAddresses", "providers") //nolint:dogsled // Return values not needed
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "providers should be removed")
}
