package subscription

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// TestExpandTemplates exercises ExpandTemplates
func TestExpandTemplates(t *testing.T) {
	t.Parallel()

	e := newMockEx()
	e.tpl = "subscriptions.tmpl"

	// Functionality tests
	l := List{
		{Channel: "single-channel"},
		{Channel: "expand-assets", Asset: asset.All, Interval: kline.FifteenMin},
		{Channel: "expand-pairs", Asset: asset.All, Levels: 1},
		{Channel: "expand-pairs", Asset: asset.Spot, Levels: 2},
		{Channel: "single-channel", QualifiedChannel: "just one sub already processed"},
		{Channel: "update-asset-pairs", Asset: asset.All},
		{Channel: "expand-pairs", Asset: asset.Spot, Pairs: e.pairs[asset.Spot][0:2], Levels: 3},
		{Channel: "batching", Asset: asset.Spot},
		{Channel: "single-channel", Authenticated: true},
	}

	_, err := l.ExpandTemplates(&mockExWithSubValidator{mockEx: e, GenerateBadSubscription: true})
	require.ErrorIs(t, err, errValidateSubscriptionsTestError)
	_, err = l.ExpandTemplates(&mockExWithSubValidator{mockEx: e})
	require.NoError(t, err)
	_, err = l.ExpandTemplates(&mockExWithSubValidator{mockEx: e, FailGetSubscriptions: true})
	require.ErrorIs(t, err, ErrNotFound)

	got, err := l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp := List{
		{Channel: "single-channel", QualifiedChannel: "single-channel"},
		{Channel: "expand-assets", QualifiedChannel: "spot-expand-assets@15m", Asset: asset.Spot, Pairs: e.pairs[asset.Spot], Interval: kline.FifteenMin},
		{Channel: "expand-assets", QualifiedChannel: "future-expand-assets@15m", Asset: asset.Futures, Pairs: e.pairs[asset.Futures], Interval: kline.FifteenMin},
		{Channel: "single-channel", QualifiedChannel: "just one sub already processed"},
		{Channel: "update-asset-pairs", QualifiedChannel: "spot-btcusdt-update-asset-pairs", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}},
		{Channel: "expand-pairs", QualifiedChannel: "spot-USDTBTC-expand-pairs@3", Asset: asset.Spot, Pairs: e.pairs[asset.Spot][1:2], Levels: 3},
		{Channel: "expand-pairs", QualifiedChannel: "spot-USDCETH-expand-pairs@3", Asset: asset.Spot, Pairs: e.pairs[asset.Spot][:1], Levels: 3},
	}
	for a, pairs := range e.pairs {
		if a == asset.Index { // Not IsAssetWebsocketEnabled
			continue
		}
		for _, p := range common.SortStrings(pairs) {
			pStr := p.Swap().String()
			if a == asset.Spot {
				exp = append(exp, List{
					{Channel: "expand-pairs", QualifiedChannel: "spot-" + pStr + "-expand-pairs@1", Asset: a, Pairs: currency.Pairs{p}, Levels: 1},
					&Subscription{Channel: "expand-pairs", QualifiedChannel: "spot-" + pStr + "-expand-pairs@2", Asset: a, Pairs: currency.Pairs{p}, Levels: 2},
				}...)
			} else {
				exp = append(exp,
					&Subscription{Channel: "expand-pairs", QualifiedChannel: "future-" + pStr + "-expand-pairs@1", Asset: a, Pairs: currency.Pairs{p}, Levels: 1},
				)
			}
		}
	}
	for _, b := range common.Batch(common.SortStrings(e.pairs[asset.Spot]), 3) {
		exp = append(exp, &Subscription{Channel: "batching", QualifiedChannel: "spot-" + b.Join() + "-batching", Asset: asset.Spot, Pairs: b})
	}

	if !equalLists(t, exp, got) {
		t.FailNow() // If the first list isn't equal testing it again will duplicate test failures
	}

	e.auth = true
	got, err = l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp = append(exp, &Subscription{Channel: "single-channel", QualifiedChannel: "single-channel-authed"})
	equalLists(t, exp, got)

	// Test with just one asset to ensure asset.All works, and disabled assets don't error
	e.assets = e.assets[:1]
	l = List{
		{Channel: "expand-assets", Asset: asset.All, Interval: kline.OneHour},
		{Channel: "expand-pairs", Asset: asset.All, Levels: 4},
		{Channel: "single-channel", Asset: asset.Futures},
	}
	got, err = l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp = slices.Grow(List{
		{Channel: "expand-assets", QualifiedChannel: "spot-expand-assets@1h", Asset: asset.Spot, Pairs: e.pairs[asset.Spot], Interval: kline.OneHour},
	}, len(e.pairs[asset.Spot]))
	for _, p := range e.pairs[asset.Spot] {
		exp = append(exp, List{
			{Channel: "expand-pairs", QualifiedChannel: "spot-" + p.Swap().String() + "-expand-pairs@4", Asset: asset.Spot, Pairs: currency.Pairs{p}, Levels: 4},
		}...)
	}
	equalLists(t, exp, got)

	// Users can specify pairs which aren't available, even across diverse assets
	// Use-case: Coinbase user sub for futures BTC-USD would return all BTC pairs and all USD pairs, even though BTC-USD might not be enabled or available
	p := currency.Pairs{currency.NewPairWithDelimiter("BEAR", "PEAR", "🐻")}
	got, err = List{{Channel: "expand-pairs", Asset: asset.All, Pairs: p}}.ExpandTemplates(e)
	require.NoError(t, err, "Must not error with fictional pairs")
	exp = List{{Channel: "expand-pairs", QualifiedChannel: "spot-PEARBEAR-expand-pairs@0", Asset: asset.Spot, Pairs: p}}
	equalLists(t, exp, got)

	// Error cases
	_, err = List{{Channel: "nil"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidTemplate, "Should get correct error on nil template")

	e.tpl = "errors.tmpl"

	_, err = List{{Channel: "error1"}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "wrong number of args for String", "Should error correctly with execution error")

	_, err = List{{Channel: "empty-content", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errNoTemplateContent, "Should error correctly when no content generated")
	assert.ErrorContains(t, err, "empty-content", "Should error correctly when no content generated")

	_, err = List{{Channel: "error2", Asset: asset.All}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errAssetRecords, "Should error correctly when invalid number of asset entries")

	_, err = List{{Channel: "error3", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errPairRecords, "Should error correctly when invalid number of pair entries")

	_, err = List{{Channel: "error4", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errTooManyBatchSizePerAsset, "Should error correctly when too many BatchSize directives")

	_, err = List{{Channel: "error5", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "Should error correctly when batch size isn't an int")

	e.tpl = "parse-error.tmpl"
	e.tpl = "parse-error.tmpl"
	_, err = l.ExpandTemplates(e)
	assert.ErrorContains(t, err, "function \"explode\" not defined", "Should error correctly on unparsable template")

	e.errFormat = errors.New("the planet Krypton is gone")
	_, err = l.ExpandTemplates(e)
	assert.ErrorIs(t, err, e.errFormat, "Should error correctly on GetPairFormat")

	e.errPairs = errors.New("bad parenting from Jor-El")
	_, err = l.ExpandTemplates(e)
	assert.ErrorIs(t, err, e.errPairs, "Should error correctly on GetEnabledPairs")

	l = List{{Channel: "single-channel", QualifiedChannel: "already happy"}}
	got, err = l.ExpandTemplates(e)
	require.NoError(t, err)
	require.Len(t, got, 1, "Must get back the one sub")
	assert.Equal(t, "already happy", l[0].QualifiedChannel, "Should get back the one sub")
	assert.NotSame(t, &got, &l, "Should get back a different actual list")

	_, err = List{{Channel: "nil"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidTemplate, "Should get correct error on nil template")
}

// TestExpandTemplatesOwnership ensures callers can modify the result of ExpandTemplates without changing
// their inputs, for unexpanded, fully qualified and mixed lists, and that keys, state and reconciliation
// keep working. See the ownership contract on ExpandTemplates and README.md
func TestExpandTemplatesOwnership(t *testing.T) {
	t.Parallel()

	e := newMockEx()
	e.tpl = "subscriptions.tmpl"

	t.Run("Unexpanded", func(t *testing.T) {
		t.Parallel()
		in := &Subscription{Channel: "single-channel"}
		got, err := List{in}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 1, "Must get one subscription back")
		assert.NotSame(t, in, got[0], "Should not return the input subscription")
		got[0].Channel = "changed"
		assert.Equal(t, "single-channel", in.Channel, "Modifying the result should not change the input")
	})

	t.Run("FullyQualified", func(t *testing.T) {
		t.Parallel()
		in := &Subscription{
			Enabled:          true,
			Channel:          OrderbookChannel,
			QualifiedChannel: "orderbook:BTCUSDT",
			Pairs:            currency.Pairs{btcusdtPair},
			Params:           map[string]any{"depth": 20},
		}
		require.NoError(t, in.SetState(SubscribedState), "SetState must not error")

		got, err := List{in}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 1, "Must get one subscription back")
		assert.NotSame(t, in, got[0], "Should not return the input subscription")
		assert.Equal(t, SubscribedState, got[0].State(), "Should keep the state")
		assert.Equal(t, "orderbook:BTCUSDT", got[0].QualifiedChannel, "Should keep the qualified channel")
		assert.Nil(t, got[0].Key, "Should not invent a key for an unkeyed subscription")

		got[0].Channel = "changed"
		got[0].QualifiedChannel = "changed"
		got[0].Pairs[0].Delimiter = "🐳"
		got[0].Params["depth"] = 50
		assert.Equal(t, OrderbookChannel, in.Channel, "Modifying the result should not change the input channel")
		assert.Equal(t, "orderbook:BTCUSDT", in.QualifiedChannel, "Modifying the result should not change the input qualified channel")
		assert.Empty(t, in.Pairs[0].Delimiter, "Modifying the result should not change the input pairs")
		assert.Equal(t, 20, in.Params["depth"], "Modifying the result should not change the input params")
	})

	t.Run("Mixed", func(t *testing.T) {
		t.Parallel()
		qualified := &Subscription{Channel: "single-channel", QualifiedChannel: "already qualified"}
		unexpanded := &Subscription{Channel: "single-channel"}

		got, err := List{qualified, unexpanded}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 2, "Must get both subscriptions back")
		assert.NotSame(t, qualified, got[0], "Should not return the qualified input subscription")
		assert.NotSame(t, unexpanded, got[1], "Should not return the unexpanded input subscription")

		got[0].Channel, got[0].QualifiedChannel = "changed", "changed"
		got[1].Channel = "changed"
		assert.Equal(t, "single-channel", qualified.Channel, "Modifying the result should not change the qualified input")
		assert.Equal(t, "already qualified", qualified.QualifiedChannel, "Modifying the result should not change the qualified input")
		assert.Equal(t, "single-channel", unexpanded.Channel, "Modifying the result should not change the unexpanded input")
	})

	t.Run("CustomKey", func(t *testing.T) {
		t.Parallel()
		in := &Subscription{Channel: OrderbookChannel, QualifiedChannel: "orderbook:BTCUSDT", Key: "custom-key"}
		st := NewStore()
		require.NoError(t, st.Add(in), "Add must not error")

		got, err := List{in}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 1, "Must get one subscription back")
		assert.Equal(t, "custom-key", got[0].Key, "Should keep a custom key so the subscription stays reconcilable")
		assert.Same(t, in, st.Get(got[0]), "Should find the store entry through the returned subscription")
		require.NoError(t, st.Remove(got[0]), "Should remove the store entry through the returned subscription")
	})

	t.Run("DefaultKey", func(t *testing.T) {
		t.Parallel()
		in := &Subscription{Channel: OrderbookChannel, QualifiedChannel: "orderbook:BTCUSDT"}
		st := NewStore()
		require.NoError(t, st.Add(in), "Add must not error")
		require.NotNil(t, in.Key, "Add must set a key")

		got, err := List{in}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 1, "Must get one subscription back")
		key, ok := got[0].EnsureKeyed().(MatchableKey)
		if assert.True(t, ok, "Should have a matchable key") {
			assert.Same(t, got[0], key.GetSubscription(), "Built-in keys should reference the returned subscription")
		}
		assert.Same(t, in, st.Get(got[0]), "Should find the store entry through the returned subscription")
		require.NoError(t, st.Remove(got[0]), "Should remove the store entry through the returned subscription")
	})

	t.Run("NestedParams", func(t *testing.T) {
		t.Parallel()
		// Documented limit: only the Params map itself is copied; values nested inside it remain shared
		nested := map[string]any{"depth": 20}
		in := &Subscription{Channel: OrderbookChannel, QualifiedChannel: "orderbook:BTCUSDT", Params: map[string]any{"opts": nested}}

		got, err := List{in}.ExpandTemplates(e)
		require.NoError(t, err, "ExpandTemplates must not error")
		require.Len(t, got, 1, "Must get one subscription back")

		got[0].Params["extra"] = 1
		assert.NotContains(t, in.Params, "extra", "Params map should be copied")
		nested["depth"] = 50
		gotNested, ok := got[0].Params["opts"].(map[string]any)
		if assert.True(t, ok, "Should keep the nested value") {
			assert.Equal(t, 50, gotNested["depth"], "Values nested inside Params remain shared with the input")
		}
	})
}
