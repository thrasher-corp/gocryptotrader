package versions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion5ExchangeType(t *testing.T) {
	t.Parallel()
	assert.Implements(t, (*ExchangeVersion)(nil), new(Version5))
}

func TestVersion5Exchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GateIO"}, new(Version5).Exchanges())
}

func TestVersion5Upgrade(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"GateIO","currencyPairs":{}}`)
	_, err := new(Version5).UpgradeExchange(context.Background(), in)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":14}}`)
	_, err = new(Version5).UpgradeExchange(context.Background(), in)
	require.Error(t, err)
	var jsonErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &jsonErr, "UpgradeExchange should return a json.UnmarshalTypeError on bad type for pairs")

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":{"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"},"futures":{"assetEnabled":true,"enabled":"BTC_USD,BTC_USDT,ETH_USDT","available":"BTC_USD,BTC_USDT,ETH_USDT,LTC_USDT"}}}}`)
	out, err := new(Version5).UpgradeExchange(context.Background(), in)
	require.NoError(t, err)
	exp := `{"name":"GateIO","currencyPairs":{"pairs":{"coinmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USD","available":"BTC_USD"},"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"},"usdtmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USDT,ETH_USDT","available":"BTC_USDT,ETH_USDT,LTC_USDT"}}}}`
	assert.Equal(t, exp, string(out))

	out, err = new(Version5).UpgradeExchange(context.Background(), out)
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "UpgradeExchange without futures should not alter the new entries")
}

func TestVersion5Downgrade(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"GateIO","currencyPairs":{}}`)
	_, err := new(Version5).DowngradeExchange(context.Background(), in)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":14}}`)
	_, err = new(Version5).DowngradeExchange(context.Background(), in)
	require.Error(t, err)
	var jsonErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &jsonErr)

	in = []byte(`{"name":"GateIO","currencyPairs":{"pairs":{"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"},"coinmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USD","available":"BTC_USD"},"usdtmarginedfutures":{"assetEnabled":true,"enabled":"BTC_USDT,ETH_USDT","available":"BTC_USDT,ETH_USDT,LTC_USDT"}}}}`)
	out, err := new(Version5).DowngradeExchange(context.Background(), in)
	require.NoError(t, err)

	exp := `{"name":"GateIO","currencyPairs":{"pairs":{"futures":{"assetEnabled":true,"enabled":"BTC_USD,BTC_USDT,ETH_USDT","available":"BTC_USD,BTC_USDT,ETH_USDT,LTC_USDT"},"spot":{"assetEnabled":true,"enabled":"BTC-USDT","available":"BTC-USDT"}}}}`
	assert.Equal(t, exp, string(out))
}
