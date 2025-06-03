package v4_test

import (
	"bytes"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v4 "github.com/thrasher-corp/gocryptotrader/config/versions/v4"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v4.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	_, err := new(v4.Version).UpgradeExchange(t.Context(), []byte{})
	require.ErrorContains(t, err, `error upgrading assetTypes`)

	_, err = new(v4.Version).UpgradeExchange(t.Context(), []byte(`{}`))
	require.ErrorContains(t, err, `error upgrading currencyPairs.pairs`)

	in := []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"spot":{"enabled":"BTC-AUD","available":"BTC-AUD"},"futures":{"assetEnabled":true},"options":{},"margin":{"assetEnabled":null}}}}`)
	out, err := new(v4.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	_, _, _, err = jsonparser.Get(out, "currencyPairs", "assetTypes") //nolint:dogsled // Ignored return values really not needed
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "assetTypes should be removed")

	e, err := jsonparser.GetBoolean(out, "currencyPairs", "pairs", "spot", "assetEnabled")
	require.NoError(t, err, "Must find assetEnabled for spot")
	assert.True(t, e, "assetEnabled should be set to true")

	e, err = jsonparser.GetBoolean(out, "currencyPairs", "pairs", "futures", "assetEnabled")
	require.NoError(t, err, "Must find assetEnabled for futures")
	assert.True(t, e, "assetEnabled should be set to true")

	e, err = jsonparser.GetBoolean(out, "currencyPairs", "pairs", "options", "assetEnabled")
	require.NoError(t, err, "Must find assetEnabled for options")
	assert.False(t, e, "assetEnabled should be set to false")

	e, err = jsonparser.GetBoolean(out, "currencyPairs", "pairs", "margin", "assetEnabled")
	require.NoError(t, err, "Must find assetEnabled for margin")
	assert.False(t, e, "assetEnabled should be set to false")

	out2, err := new(v4.Version).UpgradeExchange(t.Context(), out)
	require.NoError(t, err, "Must not error on re-upgrading")
	assert.Equal(t, out, out2, "Should not affect an already upgraded config")

	in = []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"spot":{"assetEnabled":{}}}}}`)
	_, err = new(v4.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err)

	in = []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"margin":{"assetEnabled":{}}}}}`)
	_, err = new(v4.Version).UpgradeExchange(t.Context(), in)
	require.ErrorIs(t, err, jsonparser.UnknownValueTypeError)
	require.ErrorContains(t, err, "\"margin\"")
	require.ErrorContains(t, err, "\"object\"")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"Cracken","currencyPairs":{"pairs":{"spot":{"enabled":"BTC-AUD","available":"BTC-AUD","assetEnabled":true},"futures":{"assetEnabled":false},"options":{},"options_combo":{"assetEnabled":true}}}}`)
	out, err := new(v4.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	v, vT, _, err := jsonparser.Get(out, "currencyPairs", "assetTypes")
	require.NoError(t, err, "assetTypes must be found")
	require.Equal(t, jsonparser.Array, vT, "assetTypes must be an array")
	require.Equal(t, `["spot","options_combo"]`, string(v), "assetTypes must be correct")

	assetEnabledFn := func(k, v []byte, _ jsonparser.ValueType, _ int) error {
		_, err = jsonparser.GetBoolean(v, "assetEnabled")
		require.ErrorIsf(t, err, jsonparser.KeyPathNotFoundError, "assetEnabled must be removed from %s", k)
		return nil
	}
	err = jsonparser.ObjectEach(bytes.Clone(out), assetEnabledFn, "currencyPairs", "pairs")
	require.NoError(t, err, "Must not error visiting currencyPairs")
}
