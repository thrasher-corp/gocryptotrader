package v7_test

import (
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v7 "github.com/thrasher-corp/gocryptotrader/config/versions/v7"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GateIO"}, new(v7.Version).Exchanges())
}

func TestUpgrade(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"GateIO","currencyPairs":{}}`)
	_, err := new(v7.Version).UpgradeExchange(t.Context(), in)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":14}}`)
	_, err = new(v7.Version).UpgradeExchange(t.Context(), in)
	require.Error(t, err)
	var jsonErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &jsonErr, "UpgradeExchange should return a json.UnmarshalTypeError on bad type for pairs")

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":{"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"},"futures":{"assetEnabled":true,"enabled":"BTC_USD,BTC_USDT,ETH_USDT","available":"BTC_USD,BTC_USDT,ETH_USDT,LTC_USDT"}}}}`)
	out, err := new(v7.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err)
	exp := `{"name":"GateIO","currencyPairs":{"pairs":{"coinmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USD","available":"BTC_USD"},"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"},"usdtmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USDT,ETH_USDT","available":"BTC_USDT,ETH_USDT,LTC_USDT"}}}}`
	assert.Equal(t, exp, string(out))

	out, err = new(v7.Version).UpgradeExchange(t.Context(), out)
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "UpgradeExchange without futures should not alter the new entries")
}

func TestDowngrade(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"GateIO","currencyPairs":{}}`)
	_, err := new(v7.Version).DowngradeExchange(t.Context(), in)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":14}}`)
	_, err = new(v7.Version).DowngradeExchange(t.Context(), in)
	require.Error(t, err)
	var jsonErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &jsonErr)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":{"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT,WIF-USDT"},"coinmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USD","available":"BTC_USD"},"usdtmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USDT,ETH_USDT","available":"BTC_USDT,ETH_USDT,LTC_USDT"}}}}`)
	out, err := new(v7.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err)

	exp := `{"name":"GateIO","currencyPairs":{"pairs":{"futures":{"assetEnabled":true,"enabled":"BTC_USD,BTC_USDT,ETH_USDT","available":"BTC_USD,BTC_USDT,ETH_USDT,LTC_USDT"},"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT,WIF-USDT"}}}}`
	assert.Equal(t, exp, string(out))
}
