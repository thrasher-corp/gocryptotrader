package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// TestEnsureKeyed logic test
func TestEnsureKeyed(t *testing.T) {
	t.Parallel()
	c := Subscription{
		Channel:  "candles",
		Asset:    asset.Spot,
		Currency: currency.NewPair(currency.BTC, currency.USDT),
	}
	k1, ok := c.EnsureKeyed().(DefaultKey)
	if assert.True(t, ok, "EnsureKeyed should return a DefaultKey") {
		assert.Exactly(t, k1, c.Key, "EnsureKeyed should set the same key")
		assert.Equal(t, k1.Channel, c.Channel, "DefaultKey channel should be correct")
		assert.Equal(t, k1.Asset, c.Asset, "DefaultKey asset should be correct")
		assert.Equal(t, k1.Currency, c.Currency, "DefaultKey currency should be correct")
	}
	type platypus string
	c = Subscription{
		Key:      platypus("Gerald"),
		Channel:  "orderbook",
		Asset:    asset.Margin,
		Currency: currency.NewPair(currency.ETH, currency.USDC),
	}
	k2, ok := c.EnsureKeyed().(platypus)
	if assert.True(t, ok, "EnsureKeyed should return a platypus") {
		assert.Exactly(t, k2, c.Key, "EnsureKeyed should set the same key")
		assert.EqualValues(t, "Gerald", k2, "key should have the correct value")
	}
}
