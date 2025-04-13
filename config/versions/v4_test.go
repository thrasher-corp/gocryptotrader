package versions

import (
	"bytes"
	"context"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion4ExchangeType(t *testing.T) {
	t.Parallel()
	assert.Implements(t, (*ExchangeVersion)(nil), new(Version4))
}

func TestVersion4Exchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(Version4).Exchanges())
}

func TestVersion4Upgrade(t *testing.T) {
	t.Parallel()

	_, err := new(Version4).UpgradeExchange(context.Background(), []byte{})
	require.ErrorIs(t, err, errUpgrading)
	require.ErrorContains(t, err, `assetTypes`)

	_, err = new(Version4).UpgradeExchange(context.Background(), []byte(`{}`))
	require.ErrorIs(t, err, errUpgrading)
	require.ErrorContains(t, err, `currencyPairs.pairs`)

	in := []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"spot":{"enabled":"BTC-AUD","available":"BTC-AUD"},"futures":{"assetEnabled":true},"options":{},"margin":{"assetEnabled":null}}}}`)
	out, err := new(Version4).UpgradeExchange(context.Background(), in)
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

	out2, err := new(Version4).UpgradeExchange(context.Background(), out)
	require.NoError(t, err, "Must not error on re-upgrading")
	assert.Equal(t, out, out2, "Should not affect an already upgraded config")

	in = []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"spot":{"assetEnabled":{}}}}}`)
	_, err = new(Version4).UpgradeExchange(context.Background(), in)
	require.NoError(t, err)

	in = []byte(`{"name":"Cracken","currencyPairs":{"assetTypes":["spot"],"pairs":{"margin":{"assetEnabled":{}}}}}`)
	_, err = new(Version4).UpgradeExchange(context.Background(), in)
	require.ErrorIs(t, err, jsonparser.UnknownValueTypeError)
	require.ErrorContains(t, err, "`margin`")
	require.ErrorContains(t, err, "`object`")
}

func TestVersion4Downgrade(t *testing.T) {
	t.Parallel()

	in := []byte(`{"name":"Cracken","currencyPairs":{"pairs":{"spot":{"enabled":"BTC-AUD","available":"BTC-AUD","assetEnabled":true},"futures":{"assetEnabled":false},"options":{},"options_combo":{"assetEnabled":true}}}}`)
	out, err := new(Version4).DowngradeExchange(context.Background(), in)
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
