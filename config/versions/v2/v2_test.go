package v2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v2 "github.com/thrasher-corp/gocryptotrader/config/versions/v2"
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range [][]string{
		{"GDAX", "CoinbasePro"},
		{"Kraken", "Kraken"},
		{"CoinbasePro", "CoinbasePro"},
	} {
		out, err := new(v2.Version).UpgradeExchange(t.Context(), []byte(`{"name":"`+tt[0]+`"}`))
		require.NoError(t, err)
		require.NotEmpty(t, out)
		assert.Equalf(t, `{"name":"`+tt[1]+`"}`, string(out), "Test exchange name %s", tt[0])
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range [][]string{
		{"GDAX", "GDAX"},
		{"Kraken", "Kraken"},
		{"CoinbasePro", "GDAX"},
	} {
		out, err := new(v2.Version).DowngradeExchange(t.Context(), []byte(`{"name":"`+tt[0]+`"}`))
		require.NoError(t, err)
		require.NotEmpty(t, out)
		assert.Equalf(t, `{"name":"`+tt[1]+`"}`, string(out), "Test exchange name %s", tt[0])
	}
}
