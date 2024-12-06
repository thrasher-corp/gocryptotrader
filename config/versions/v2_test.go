package versions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion2Upgrade(t *testing.T) {
	t.Parallel()
	for _, tt := range [][]string{
		{"GDAX", "CoinbasePro"},
		{"Kraken", "Kraken"},
		{"CoinbasePro", "CoinbasePro"},
	} {
		out, err := new(Version2).UpgradeExchange(context.Background(), []byte(`{"name":"`+tt[0]+`"}`))
		require.NoError(t, err)
		require.NotEmpty(t, out)
		assert.Equalf(t, `{"name":"`+tt[1]+`"}`, string(out), "Test exchange name %s", tt[0])
	}
}

func TestVersion2Downgrade(t *testing.T) {
	t.Parallel()
	for _, tt := range [][]string{
		{"GDAX", "GDAX"},
		{"Kraken", "Kraken"},
		{"CoinbasePro", "GDAX"},
	} {
		out, err := new(Version2).DowngradeExchange(context.Background(), []byte(`{"name":"`+tt[0]+`"}`))
		require.NoError(t, err)
		require.NotEmpty(t, out)
		assert.Equalf(t, `{"name":"`+tt[1]+`"}`, string(out), "Test exchange name %s", tt[0])
	}
}
