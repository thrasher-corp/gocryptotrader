package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	btcusdtPair = currency.NewBTCUSDT()
	ethusdcPair = currency.NewPair(currency.ETH, currency.USDC)
	ltcusdcPair = currency.NewPair(currency.LTC, currency.USDC)
)

// TestSubscriptionString exercises the String method
func TestSubscriptionString(t *testing.T) {
	t.Parallel()
	s := &Subscription{
		Channel: "candles",
		Asset:   asset.Spot,
		Pairs:   currency.Pairs{btcusdtPair, ethusdcPair.Format(currency.PairFormat{Delimiter: "/"})},
	}
	assert.Equal(t, "candles spot BTC/USDT,ETH/USDC", s.String(), "Subscription String should return correct value")
	s.Key = 42
	assert.Equal(t, "42: candles spot BTC/USDT,ETH/USDC", s.String(), "String with a non-MatchableKey")
}

// TestState exercises the state getter
func TestState(t *testing.T) {
	t.Parallel()
	s := &Subscription{}
	assert.Equal(t, InactiveState, s.State(), "State should return initial state")
	s.state = SubscribedState
	assert.Equal(t, SubscribedState, s.State(), "State should return correct state")
}

// TestSetState exercises the state setter
func TestSetState(t *testing.T) {
	t.Parallel()

	s := &Subscription{state: UnsubscribedState}

	for i := InactiveState; i <= UnsubscribedState; i++ {
		assert.NoErrorf(t, s.SetState(i), "State should not error setting state %s", i)
	}
	assert.ErrorIs(t, s.SetState(UnsubscribedState), ErrInStateAlready, "SetState should error on same state")
	assert.ErrorIs(t, s.SetState(UnsubscribedState+1), ErrInvalidState, "Setting an invalid state should error")
}

// TestEnsureKeyed exercises the key getter and ensures it sets a self-pointer key for non
func TestEnsureKeyed(t *testing.T) {
	t.Parallel()
	s := &Subscription{}
	k1, ok := s.EnsureKeyed().(MatchableKey)
	if assert.True(t, ok, "EnsureKeyed should return a MatchableKey") {
		assert.Same(t, s, k1.GetSubscription(), "Key should point to the same struct")
	}
	type platypus string
	s = &Subscription{
		Key:     platypus("Gerald"),
		Channel: "orderbook",
	}
	k2 := s.EnsureKeyed()
	assert.IsType(t, platypus(""), k2, "EnsureKeyed should return a platypus")
	assert.Equal(t, s.Key, k2, "Key should be the key provided")
}

// TestSubscriptionMarshaling ensures json Marshalling is clean and concise
// Since there is no UnmarshalJSON, this just exercises the json field tags of Subscription, and regressions in conciseness
func TestSubscriptionMarshaling(t *testing.T) {
	t.Parallel()
	j, err := json.Marshal(&Subscription{Key: 42, Channel: CandlesChannel})
	assert.NoError(t, err, "Marshalling should not error")
	assert.JSONEq(t, `{"enabled":false,"channel":"candles"}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4})
	assert.NoError(t, err, "Marshalling should not error")
	assert.JSONEq(t, `{"enabled":true,"channel":"orderbook","interval":"5m","levels":4}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4, Pairs: currency.Pairs{currency.NewBTCUSDT()}})
	assert.NoError(t, err, "Marshalling should not error")
	assert.JSONEq(t, `{"enabled":true,"channel":"orderbook","pairs":"BTCUSDT","interval":"5m","levels":4}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: MyTradesChannel, Authenticated: true})
	assert.NoError(t, err, "Marshalling should not error")
	assert.JSONEq(t, `{"enabled":true,"channel":"myTrades","authenticated":true}`, string(j), "Marshalling should be clean and concise")
}

func TestSubscriptionClone(t *testing.T) {
	t.Parallel()
	params := map[string]any{"a": 42}
	a := &Subscription{
		Channel:  TickerChannel,
		Interval: kline.OneHour,
		Pairs:    currency.Pairs{btcusdtPair},
		Params:   params,
	}
	a.EnsureKeyed()
	b := a.Clone()
	assert.IsType(t, new(Subscription), b, "Clone should return a Subscription pointer")
	assert.NotSame(t, a, b, "Clone should return a new Subscription")
	assert.Nil(t, b.Key, "Clone should have a nil key")
	b.Pairs[0].Delimiter = "🐳"
	assert.Empty(t, a.Pairs[0].Delimiter, "Pairs should be (relatively) deep copied")
	b.Params["a"] = 12
	assert.Equal(t, 42, a.Params["a"], "Params should be (relatively) deep copied")
	assert.NotEqual(t, params, b.Params, "Params should be cloned")
	assert.Equal(t, params, a.Params, "Original Params should be left alone")
	a.m.Lock()
	assert.True(t, b.m.TryLock(), "Clone should use a different Mutex")
}

// TestSetKey exercises SetKey
func TestSetKey(t *testing.T) {
	t.Parallel()
	s := &Subscription{}
	s.SetKey(14)
	assert.Equal(t, 14, s.Key, "SetKey should set a key correctly")
}

// TestSetPairs exercises SetPairs
func TestSetPairs(t *testing.T) {
	t.Parallel()
	s := &Subscription{}
	s.SetPairs(currency.Pairs{btcusdtPair})
	assert.Equal(t, "BTCUSDT", s.Pairs.Join(), "SetPairs should set a key correctly")
}

// TestAddPairs exercises AddPairs
func TestAddPairs(t *testing.T) {
	t.Parallel()
	s := &Subscription{}
	s.AddPairs()
	assert.Empty(t, s.Pairs, "Should not have added any pairs")
	s.AddPairs(btcusdtPair)
	assert.Len(t, s.Pairs, 1, "Should not have added any pairs")
}
