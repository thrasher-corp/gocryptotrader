package subscription

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// TestEnsureKeyed logic test
func TestEnsureKeyed(t *testing.T) {
	t.Parallel()
	c := Subscription{
		Channel: "candles",
		Asset:   asset.Spot,
		Pair:    currency.NewPair(currency.BTC, currency.USDT),
	}
	k1, ok := c.EnsureKeyed().(DefaultKey)
	if assert.True(t, ok, "EnsureKeyed should return a DefaultKey") {
		assert.Exactly(t, k1, c.Key, "EnsureKeyed should set the same key")
		assert.Equal(t, k1.Channel, c.Channel, "DefaultKey channel should be correct")
		assert.Equal(t, k1.Asset, c.Asset, "DefaultKey asset should be correct")
		assert.Equal(t, k1.Pair, c.Pair, "DefaultKey currency should be correct")
	}
	type platypus string
	c = Subscription{
		Key:     platypus("Gerald"),
		Channel: "orderbook",
		Asset:   asset.Margin,
		Pair:    currency.NewPair(currency.ETH, currency.USDC),
	}
	k2, ok := c.EnsureKeyed().(platypus)
	if assert.True(t, ok, "EnsureKeyed should return a platypus") {
		assert.Exactly(t, k2, c.Key, "EnsureKeyed should set the same key")
		assert.EqualValues(t, "Gerald", k2, "key should have the correct value")
	}
}

// TestMarshalling logic test
func TestMarshaling(t *testing.T) {
	t.Parallel()
	j, err := json.Marshal(&Subscription{Channel: CandlesChannel})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":false,"channel":"candles"}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"orderbook","interval":"5m","levels":4}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"orderbook","interval":"5m","levels":4,"pair":"BTCUSDT"}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: MyTradesChannel, Authenticated: true})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"myTrades","authenticated":true}`, string(j), "Marshalling should be clean and concise")
}
