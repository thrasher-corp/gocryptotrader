package subscription

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	btcusdtPair = currency.NewPair(currency.BTC, currency.USDT)
	ethusdcPair = currency.NewPair(currency.ETH, currency.USDC)
	ltcusdcPair = currency.NewPair(currency.LTC, currency.USDC)
)

// TestSubscriptionString exercises the String method
func TestSubscriptionString(t *testing.T) {
	s := &Subscription{
		Channel: "candles",
		Asset:   asset.Spot,
		Pairs:   currency.Pairs{btcusdtPair, ethusdcPair.Format(currency.PairFormat{Delimiter: "/"})},
	}
	assert.Equal(t, "candles spot BTC/USDT,ETH/USDC", s.String(), "Subscription String should return correct value")
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

	s := &Subscription{state: UnsubscribingState}

	for i := InactiveState; i <= UnsubscribingState; i++ {
		assert.NoErrorf(t, s.SetState(i), "State should not error setting state %s", i)
	}
	assert.ErrorIs(t, s.SetState(UnsubscribingState), ErrInStateAlready, "SetState should error on same state")
	assert.ErrorIs(t, s.SetState(UnsubscribingState+1), ErrInvalidState, "Setting an invalid state should error")
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

// TestSubscriptionMarshalling ensures json Marshalling is clean and concise
// Since there is no UnmarshalJSON, this just exercises the json field tags of Subscription, and regressions in conciseness
func TestSubscriptionMarshaling(t *testing.T) {
	t.Parallel()
	j, err := json.Marshal(&Subscription{Key: 42, Channel: CandlesChannel})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":false,"channel":"candles"}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"orderbook","interval":"5m","levels":4}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: OrderbookChannel, Interval: kline.FiveMin, Levels: 4, Pairs: currency.Pairs{currency.NewPair(currency.BTC, currency.USDT)}})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"orderbook","pairs":"BTCUSDT","interval":"5m","levels":4}`, string(j), "Marshalling should be clean and concise")

	j, err = json.Marshal(&Subscription{Enabled: true, Channel: MyTradesChannel, Authenticated: true})
	assert.NoError(t, err, "Marshalling should not error")
	assert.Equal(t, `{"enabled":true,"channel":"myTrades","authenticated":true}`, string(j), "Marshalling should be clean and concise")
}

// TestClone exercises Clone
func TestClone(t *testing.T) {
	a := &Subscription{
		Channel:  TickerChannel,
		Interval: kline.OneHour,
	}
	b := a.Clone()
	assert.IsType(t, new(Subscription), b, "Clone must return a Subscription pointer")
	assert.NotSame(t, a, b, "Clone must return a new Subscription")
	a.m.Lock()
	assert.True(t, b.m.TryLock(), "Clone must use a different Mutex")
}
