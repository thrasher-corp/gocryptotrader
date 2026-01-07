package v11_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v11 "github.com/thrasher-corp/gocryptotrader/config/versions/v11"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Binance"}, new(v11.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	_, err := new(v11.Version).UpgradeExchange(t.Context(), []byte(`{"name":"Binance"}`))
	assert.NoError(t, err, "UpgradeExchange should not error with no subscriptions")

	in := []byte(`{"name":"Binance","features":{"subscriptions":[{"channel":"1","asset":4}]}}`)
	_, err = new(v11.Version).UpgradeExchange(t.Context(), in)
	assert.ErrorContains(t, err, "Value is not a string: 4", "UpgradeExchange should error correctly")

	in = []byte(`{"name":"Binance","features":{"subscriptions":[{"channel":"1"},{"channel":"2","asset":"all"}]}}`)
	out, err := new(v11.Version).UpgradeExchange(t.Context(), in)
	exp := `{"name":"Binance","features":{"subscriptions":[{"channel":"1","asset":"spot"},{"channel":"2","asset":"all"}]}}`
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "UpgradeExchange should modify the subscriptions correctly")

	out, err = new(v11.Version).UpgradeExchange(t.Context(), out)
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "Running UpgradeExchange twice should not make any changes")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"Binance","features":{"subscriptions":[{"channel":"1"},{"channel":"2","asset":"all"}]}}`)
	out, err := new(v11.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, string(in), string(out), "DowngradeExchange should not change input")
}
