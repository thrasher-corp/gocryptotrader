package subscription

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// TestListStrings exercises List.Strings()
func TestListStrings(t *testing.T) {
	t.Parallel()
	l := List{
		&Subscription{
			Channel: TickerChannel,
			Asset:   asset.Spot,
			Pairs:   currency.Pairs{ethusdcPair, btcusdtPair},
		},
		&Subscription{
			Channel: OrderbookChannel,
			Pairs:   currency.Pairs{ethusdcPair},
		},
	}
	exp := []string{"orderbook  ETH/USDC", "ticker spot ETH/USDC,BTC/USDT"}
	assert.ElementsMatch(t, exp, l.Strings(), "String must return correct sorted list")
}

// TestQualifiedChannels exercises List.QualifiedChannels()
func TestQualifiedChannels(t *testing.T) {
	t.Parallel()
	l := List{
		&Subscription{
			QualifiedChannel: "ticker-btc",
		},
		&Subscription{
			QualifiedChannel: "candles-btc",
		},
	}
	exp := []string{"ticker-btc", "candles-btc"}
	assert.ElementsMatch(t, exp, l.QualifiedChannels(), "QualifiedChannels should return correct sorted list")
}

// TestListGroupPairs exercises List.GroupPairs()
func TestListGroupPairs(t *testing.T) {
	t.Parallel()
	l := List{
		{Asset: asset.Spot, Channel: TickerChannel, Pairs: currency.Pairs{ethusdcPair, btcusdtPair}},
	}
	for _, c := range []string{TickerChannel, OrderbookChannel} {
		for _, p := range []currency.Pair{ethusdcPair, btcusdtPair} {
			l = append(l, &Subscription{
				Channel: c,
				Asset:   asset.Spot,
				Pairs:   currency.Pairs{p},
			})
		}
	}
	n := l.GroupPairs()
	assert.Len(t, l, 5, "Orig list should not be changed")
	assert.Len(t, n, 2, "New list should be grouped")
	exp := []string{"ticker spot ETH/USDC,BTC/USDT", "orderbook spot ETH/USDC,BTC/USDT"}
	assert.ElementsMatch(t, exp, n.Strings(), "String must return correct sorted list")
}

// TestListSetStates exercises List.SetState()
func TestListSetStates(t *testing.T) {
	t.Parallel()
	l := List{{Channel: TickerChannel}, {Channel: OrderbookChannel}}
	assert.NoError(t, l.SetStates(SubscribingState), "SetStates should not error")
	assert.Equal(t, SubscribingState, l[1].State(), "SetStates should set State correctly")

	require.NoError(t, l[0].SetState(SubscribedState), "Individual SetState must not error")
	err := l.SetStates(SubscribedState)
	assert.ErrorIs(t, ErrInStateAlready, err, "SetStates should error when duplicate state")
	assert.Equal(t, SubscribedState, l[1].State(), "SetStates should set State correctly after the error")
}

// TestAssetPairs exercises AssetPairs error handling
// All other code is covered under TestExpandTemplates
func TestAssetPairs(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.All} {
		e := newMockEx()
		l := &List{{Channel: CandlesChannel, Asset: a}}
		e.errFormat = errors.New("Krypton is back")
		_, err := l.assetPairs(e)
		assert.ErrorIs(t, err, e.errFormat, "Should error correctly on GetPairFormat")
		e.errPairs = errors.New("Krypton is gone")
		_, err = l.assetPairs(e)
		assert.ErrorIs(t, err, e.errPairs, "Should error correctly on GetEnabledPairs")
	}
}

func TestListClone(t *testing.T) {
	t.Parallel()
	l := List{{Channel: TickerChannel}, {Channel: OrderbookChannel}}
	n := l.Clone()
	assert.NotSame(t, n, l, "Slices must not be the same")
	require.NotEmpty(t, n, "List must not be empty")
	assert.NotSame(t, n[0], l[0], "Subscriptions must be cloned")
	assert.Equal(t, n[0], l[0], "Subscriptions should be equal")
	l[0].Interval = kline.OneHour
	assert.NotEqual(t, n[0], l[0], "Subscriptions should be cloned")
}

var filterable = List{
	{Channel: "a", Enabled: true, Authenticated: false},
	{Channel: "b", Enabled: true, Authenticated: true},
	{Channel: "c", Enabled: false, Authenticated: true},
	{Channel: "d", Enabled: false, Authenticated: false},
}

func TestListEnabled(t *testing.T) {
	t.Parallel()
	l := filterable.Enabled()
	require.Len(t, l, 2)
	assert.Equal(t, filterable[:2], l)
	assert.Len(t, filterable, 4)
}

func TestListPublic(t *testing.T) {
	t.Parallel()
	l := filterable.Public()
	require.Len(t, l, 2)
	assert.Equal(t, List{filterable[0], filterable[3]}, l)
	assert.Len(t, filterable, 4)
}

func TestListPrivate(t *testing.T) {
	t.Parallel()
	l := filterable.Private()
	require.Len(t, l, 2)
	assert.Equal(t, filterable[1:3], l)
	assert.Len(t, filterable, 4)
}
