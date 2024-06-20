package subscription

import (
	"errors"
	"maps"
	"strings"
	"testing"
	"text/template"

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
	expErr := errors.New("Krypton is gone")
	for _, a := range []asset.Item{asset.Spot, asset.All} {
		l := &List{{Channel: CandlesChannel, Asset: a}}
		_, err := l.AssetPairs(&mockEx{false, expErr, nil})
		assert.ErrorIs(t, err, expErr, "Should error correctly on GetEnabledPairs")
		_, err = l.AssetPairs(&mockEx{false, nil, expErr})
		assert.ErrorIs(t, err, expErr, "Should error correctly on GetPairFormat")
	}
}

// TestExpandTemplates exercises ExpandTemplates
func TestExpandTemplates(t *testing.T) {
	t.Parallel()
	l := List{
		{Channel: CandlesChannel,
			Template: "ohlc.{{$asset}}.{{$s.Interval.Short}}",
			Asset:    asset.All,
			Pairs:    currency.Pairs{btcusdtPair, ethusdcPair},
			Interval: kline.FifteenMin},
		{Channel: OrderbookChannel,
			Template: "book-{{$pair}}@{{$s.Levels}}",
			Asset:    asset.Spot,
			Pairs:    currency.Pairs{btcusdtPair, ethusdcPair},
			Levels:   100},
		{Channel: CandlesChannel,
			Template: "candles.{{assetName $asset}}.{{$pair.Swap.String}}.{{if eq $pair.String `BTCUSDT`}}{{$s.Params.color}}{{else}}red{{end}}.{{$s.Interval.Short}}",
			Asset:    asset.All,
			Pairs:    currency.Pairs{btcusdtPair, ethusdcPair},
			Params: map[string]any{
				"color": "green",
			},
			Interval: kline.FifteenMin},
		{Channel: MyTradesChannel,
			Template:      "trades.{{assetName $asset}}",
			Asset:         asset.All,
			Pairs:         currency.Pairs{btcusdtPair, ethusdcPair},
			Authenticated: true,
		},
		{Channel: AllTradesChannel,
			QualifiedChannel: "already happy",
			Template:         "{{breakThings}}"},
	}
	got, err := l.ExpandTemplates(&mockEx{false, nil, nil})
	require.NoError(t, err, "ExpandTemplates must not error")
	exp := List{
		{Channel: CandlesChannel, QualifiedChannel: "ohlc.spot.15m", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Interval: kline.FifteenMin},
		{Channel: CandlesChannel, QualifiedChannel: "ohlc.futures.15m", Asset: asset.Futures, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Interval: kline.FifteenMin},
		{Channel: OrderbookChannel, QualifiedChannel: "book-BTCUSDT@100", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Levels: 100},
		{Channel: OrderbookChannel, QualifiedChannel: "book-ETHUSDC@100", Asset: asset.Spot, Pairs: currency.Pairs{ethusdcPair}, Levels: 100},
		{Channel: CandlesChannel, QualifiedChannel: "candles.spot.USDTBTC.green.15m", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Interval: kline.FifteenMin,
			Params: map[string]any{"color": "green"}},
		{Channel: CandlesChannel, QualifiedChannel: "candles.spot.USDCETH.red.15m", Asset: asset.Spot, Pairs: currency.Pairs{ethusdcPair}, Interval: kline.FifteenMin,
			Params: map[string]any{"color": "green"}},
		{Channel: CandlesChannel, QualifiedChannel: "candles.future.USDTBTC.green.15m", Asset: asset.Futures, Pairs: currency.Pairs{btcusdtPair}, Interval: kline.FifteenMin,
			Params: map[string]any{"color": "green"}},
		{Channel: CandlesChannel, QualifiedChannel: "candles.future.USDCETH.red.15m", Asset: asset.Futures, Pairs: currency.Pairs{ethusdcPair}, Interval: kline.FifteenMin,
			Params: map[string]any{"color": "green"}},
		{Channel: AllTradesChannel, QualifiedChannel: "already happy"},
	}

	if !equalLists(t, exp, got) {
		t.FailNow() // If the first list isn't equal testing it again will duplicate test failures
	}

	got, err = l.ExpandTemplates(&mockEx{true, nil, nil})
	require.NoError(t, err, "ExpandTemplates must not error")
	exp = append(exp,
		&Subscription{Channel: MyTradesChannel, QualifiedChannel: "trades.spot", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}},
		&Subscription{Channel: MyTradesChannel, QualifiedChannel: "trades.future", Asset: asset.Futures, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}},
	)
	equalLists(t, exp, got)

	expErr := errors.New("the planet Krypton is gone")
	_, err = l.ExpandTemplates(&mockEx{false, nil, expErr})
	assert.ErrorIs(t, err, expErr, "Should error correctly on GetPairFormat")

	_, err = List{{Template: "Broken \x1E record"}}.ExpandTemplates(&mockEx{false, nil, expErr})
	assert.ErrorIs(t, err, errRecordSeparator, "Should error correctly on GetPairFormat")

	_, err = List{{Asset: asset.Spot, Template: "{{$asset}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorIs(t, err, errAssetTemplateWithoutAll, "Should error correctly on xpand Assets without All")

	_, err = List{{Asset: asset.All, Template: "{{$pair}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on xpand pair with All")

	_, err = List{{Asset: asset.All, Template: "{{gobble $pair}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on bad template")

	_, err = List{{Template: "{{breakThings}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.Error(t, err, "Should error correctly too many lines")

	got, err = List{{Asset: asset.Empty, Template: "allOrders"}}.ExpandTemplates(&mockEx{false, nil, nil})
	require.NoError(t, err, "Should not error on a template with nothing else in it")
	require.Len(t, got, 1, "Should get 1 subscription from an empty template")
	assert.Equal(t, "allOrders", got[0].QualifiedChannel, "Should expand a simple template")

	_, err = List{{Template: "{{brokeThings}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorContains(t, err, "not defined parsing", "Should error correctly on bad template")

	_, err = List{{Template: "{{breakThings 42}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorContains(t, err, "wrong number of args for breakThings", "Should error correctly on bad template")

	_, err = List{{Template: "{{breakThings}}"}}.ExpandTemplates(&mockEx{false, nil, nil})
	assert.ErrorContains(t, err, "did not generate the expected number of lines", "Should error correctly too many lines")

	l = List{{QualifiedChannel: "already happy", Template: "{{breakThings}}"}}
	got, err = l.ExpandTemplates(&mockEx{false, nil, nil})
	require.NoError(t, err, "did not generate the expected number of lines", "Must not error on a list of processed entries")
	require.Len(t, got, 1, "Must get back the one sub")
	assert.Equal(t, "already happy", l[0].QualifiedChannel, "Should get back the one sub")
	assert.NotSame(t, got, l, "Should get back a different actual list")
}

type mockEx struct {
	auth      bool
	errPairs  error
	errFormat error
}

func (m *mockEx) GetEnabledPairs(_ asset.Item) (currency.Pairs, error) {
	return currency.Pairs{btcusdtPair, ethusdcPair}, m.errPairs
}

func (m *mockEx) GetPairFormat(_ asset.Item, _ bool) (currency.PairFormat, error) {
	return currency.PairFormat{Uppercase: true}, m.errFormat
}

func (m *mockEx) GetSubscriptionTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"assetName": func(a asset.Item) string {
			if a == asset.Futures {
				return "future"
			}
			return a.String()
		},
		"breakThings": func() string {
			return "\x1E Too many zoos \x1E"
		},
	}
}

func (m *mockEx) GetAssetTypes(_ bool) asset.Items            { return asset.Items{asset.Spot, asset.Futures} }
func (m *mockEx) CanUseAuthenticatedWebsocketEndpoints() bool { return m.auth }

// equalLists is a utility function to compare subscription lists and show a pretty failure message
// It overcomes the verbose depth of assert.ElementsMatch spewConfig
// Duplicate of internal/testing/subscriptions/EqualLists
func equalLists(tb testing.TB, a, b List) bool {
	tb.Helper()
	for _, sub := range append(a, b...) {
		sub.Key = &StrictKey{&ExactKey{sub}}
	}
	s, err := NewStoreFromList(a)
	require.NoError(tb, err, "NewStoreFromList must not error")
	added, missing := s.Diff(b)
	if len(added) > 0 || len(missing) > 0 {
		fail := "Differences:"
		if len(added) > 0 {
			fail = fail + "\n + " + strings.Join(added.Strings(), "\n + ")
		}
		if len(missing) > 0 {
			fail = fail + "\n - " + strings.Join(missing.Strings(), "\n - ")
		}
		assert.Fail(tb, fail, "Subscriptions should be equal")
		return false
	}
	return true
}

// StrictKey is key type for subscriptions where all the pairs, QualifiedChannel and Params in a Subscription must match exactly
type StrictKey struct {
	*ExactKey
}

var _ MatchableKey = StrictKey{} // Enforce StrictKey must implement MatchableKey

// Match implements MatchableKey
// Returns true if the key fields exactly matches the subscription, including all Pairs, QualifiedChannel and Params
func (k StrictKey) Match(eachKey MatchableKey) bool {
	if !k.ExactKey.Match(eachKey) {
		return false
	}
	eachSub := eachKey.GetSubscription()
	return eachSub.QualifiedChannel == k.QualifiedChannel &&
		maps.Equal(eachSub.Params, k.Params)
}

// String implements Stringer; returns the Asset, Channel and Pairs
// Does not provide concurrency protection on the subscription it points to
func (k StrictKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised StrictKey"
	}
	return s.QualifiedChannel + " " + ExactKey{s}.String()
}
