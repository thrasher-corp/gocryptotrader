package versions

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion1Upgrade(t *testing.T) {
	t.Parallel()

	v := &Version1{}
	in := []byte(`{"name":"Wibble","pairsLastUpdated":1566798411,"assetTypes":"spot","configCurrencyPairFormat":{"uppercase":true,"delimiter":"_"},"requestCurrencyPairFormat":{"uppercase":false,"delimiter":"_","separator":"-"},"enabledPairs":"LTC_BTC","availablePairs":"LTC_BTC,ETH_BTC,BTC_USD"}`)
	exp := []byte(`{"name":"Wibble","currencyPairs":{"bypassConfigFormatUpgrades":false,"requestFormat":{"uppercase":false,"delimiter":"_","separator":"-"},"configFormat":{"uppercase":true,"delimiter":"_"},"useGlobalFormat":true,"lastUpdated":1566798411,"pairs":{"spot":{"enabled":"LTC_BTC","available":"LTC_BTC,ETH_BTC,BTC_USD"}}}}`)

	out, err := v.UpgradeExchange(context.Background(), in)
	require.NoError(t, err)
	require.NotEmpty(t, out)
	assert.Equal(t, string(exp), string(out))
}

func TestVersion1Downgrade(t *testing.T) {
	t.Parallel()
	in := []byte("just leave me alone, mkay?")
	out, err := new(Version1).DowngradeExchange(context.Background(), bytes.Clone(in))
	require.NoError(t, err)
	assert.Equal(t, out, in)
}
