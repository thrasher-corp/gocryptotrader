package v1_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v1.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	v := &v1.Version{}
	in := []byte(`{"name":"Wibble","pairsLastUpdated":1566798411,"assetTypes":"spot","configCurrencyPairFormat":{"uppercase":true,"delimiter":"_"},"requestCurrencyPairFormat":{"uppercase":false,"delimiter":"_","separator":"-"},"enabledPairs":"LTC_BTC","availablePairs":"LTC_BTC,ETH_BTC,BTC_USD"}`)
	exp := []byte(`{"name":"Wibble","currencyPairs":{"bypassConfigFormatUpgrades":false,"requestFormat":{"uppercase":false,"delimiter":"_","separator":"-"},"configFormat":{"uppercase":true,"delimiter":"_"},"useGlobalFormat":true,"lastUpdated":1566798411,"pairs":{"spot":{"enabled":"LTC_BTC","available":"LTC_BTC,ETH_BTC,BTC_USD"}}}}`)

	out, err := v.UpgradeExchange(t.Context(), in)
	require.NoError(t, err)
	require.NotEmpty(t, out)
	assert.Equal(t, string(exp), string(out))
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte("just leave me alone, mkay?")
	out, err := new(v1.Version).DowngradeExchange(t.Context(), bytes.Clone(in))
	require.NoError(t, err)
	assert.Equal(t, out, in)
}
