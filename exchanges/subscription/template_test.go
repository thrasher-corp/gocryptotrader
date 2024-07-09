package subscription

import (
	"errors"
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

	pairs := currency.Pairs{btcusdtPair, ethusdcPair}
	for _, b := range []currency.Code{currency.LTC, currency.XRP, currency.TRX} {
		for _, q := range []currency.Code{currency.USDT, currency.USDC} {
			pairs = append(pairs, currency.NewPair(b, q))
		}
	}

	e := &mockEx{
		pairs: pairs,
		tpl:   "subscriptions.tmpl",
	}

	// Functionality tests
	l := List{
		{Channel: "feature1"},
		{Channel: "feature2", Asset: asset.All, Interval: kline.FifteenMin},
		{Channel: "feature3", Asset: asset.All, Levels: 100},
		{Channel: "feature1", QualifiedChannel: "just one sub already processed"},
		{Channel: "feature5", Asset: asset.All},
		{Channel: "feature7", Asset: asset.Spot, Pairs: pairs[0:2]},
		{Channel: "feature6", Asset: asset.Spot},
		{Channel: "feature4", Authenticated: true},
	}
	got, err := l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp := List{
		{Channel: "feature1", QualifiedChannel: "feature1"},
		{Channel: "feature2", QualifiedChannel: "spot-feature2@15m", Asset: asset.Spot, Pairs: pairs, Interval: kline.FifteenMin},
		{Channel: "feature2", QualifiedChannel: "future-feature2@15m", Asset: asset.Futures, Pairs: pairs, Interval: kline.FifteenMin},
		{Channel: "feature1", QualifiedChannel: "just one sub already processed"},
		{Channel: "feature5", QualifiedChannel: "spot-btcusdt-feature5", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}},
		{Channel: "feature7", QualifiedChannel: "spot-BTCUSDT,ETHUSDC-feature7", Asset: asset.Spot, Pairs: pairs[0:2]},
	}
	for _, p := range pairs {
		exp = append(exp, List{
			{Channel: "feature3", QualifiedChannel: "spot-" + p.Swap().String() + "-feature3@100", Asset: asset.Spot, Pairs: currency.Pairs{p}, Levels: 100},
			{Channel: "feature3", QualifiedChannel: "future-" + p.Swap().String() + "-feature3@100", Asset: asset.Futures, Pairs: currency.Pairs{p}, Levels: 100},
		}...)
	}
	for _, b := range common.Batch(common.SortStrings(pairs), 3) {
		exp = append(exp, &Subscription{Channel: "feature6", QualifiedChannel: "spot-" + b.Join() + "-feature6", Asset: asset.Spot, Pairs: b})
	}

	if !equalLists(t, exp, got) {
		t.FailNow() // If the first list isn't equal testing it again will duplicate test failures
	}

	e.auth = true
	got, err = l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp = append(exp,
		&Subscription{Channel: "feature4", QualifiedChannel: "feature4-authed"},
	)
	equalLists(t, exp, got)

	_, err = List{{Channel: "nil"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidTemplate, "Should get correct error on nil template")

	_, err = List{{Channel: "feature1", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("NOPE", "POPE", "🐰")}}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, currency.ErrPairNotContainedInAvailablePairs, "Should error correctly when pair not available")

	e.tpl = "errors.tmpl"

	_, err = List{{Channel: "error1", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errAssetTemplateWithoutAll, "Should error correctly on xpand assets without All")

	_, err = List{{Channel: "error2", Asset: asset.All}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on xpand pairs but not assets")

	_, err = List{{Channel: "error2"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on xpand pairs but not assets")

	_, err = List{{Channel: "error3"}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "wrong number of args for String", "Should error correctly with execution error")

	_, err = List{{Channel: "empty-content", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errNoTemplateContent, "Should error correctly when no content generated")
	assert.ErrorContains(t, err, "empty-content", "Should error correctly when no content generated")

	_, err = List{{Channel: "error4", Asset: asset.All}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errAssetRecords, "Should error correctly when invalid number of asset entries")

	_, err = List{{Channel: "error5", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errPairRecords, "Should error correctly when invalid number of pair entries")

	_, err = List{{Channel: "error6", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errTooManyBatchSize, "Should error correctly when too many BatchSize directives")

	_, err = List{{Channel: "error7", Asset: asset.Spot}}.ExpandTemplates(e)
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

	l = List{{Channel: "feature1", QualifiedChannel: "already happy"}}
	got, err = l.ExpandTemplates(e)
	require.NoError(t, err)
	require.Len(t, got, 1, "Must get back the one sub")
	assert.Equal(t, "already happy", l[0].QualifiedChannel, "Should get back the one sub")
	assert.NotSame(t, got, l, "Should get back a different actual list")

	_, err = List{{Channel: "nil"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidTemplate, "Should get correct error on nil template")
}
