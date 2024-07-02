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

// TestExpandTemplates exercises ExpandTemplates
func TestExpandTemplates(t *testing.T) {
	t.Parallel()

	e := &mockEx{
		tpl: "subscriptions.tmpl",
	}

	// Functionality tests
	l := List{
		{Channel: "feature1"},
		{Channel: "feature2", Asset: asset.All, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Interval: kline.FifteenMin},
		{Channel: "feature3", Asset: asset.All, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Levels: 100},
		{Channel: "feature4", Authenticated: true},
		{Channel: "feature1", QualifiedChannel: "just one sub already processed"},
	}
	got, err := l.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	exp := List{
		{Channel: "feature1", QualifiedChannel: "feature1"},
		{Channel: "feature2", QualifiedChannel: "spot-feature2@15m", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Interval: kline.FifteenMin},
		{Channel: "feature2", QualifiedChannel: "future-feature2@15m", Asset: asset.Futures, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}, Interval: kline.FifteenMin},
		{Channel: "feature3", QualifiedChannel: "spot-USDTBTC-feature3@100", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Levels: 100},
		{Channel: "feature3", QualifiedChannel: "spot-USDCETH-feature3@100", Asset: asset.Spot, Pairs: currency.Pairs{ethusdcPair}, Levels: 100},
		{Channel: "feature3", QualifiedChannel: "future-USDTBTC-feature3@100", Asset: asset.Futures, Pairs: currency.Pairs{btcusdtPair}, Levels: 100},
		{Channel: "feature3", QualifiedChannel: "future-USDCETH-feature3@100", Asset: asset.Futures, Pairs: currency.Pairs{ethusdcPair}, Levels: 100},
		{Channel: "feature1", QualifiedChannel: "just one sub already processed"},
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

	_, err = List{{Channel: "feature2", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errAssetTemplateWithoutAll, "Should error correctly on xpand assets without All")

	e.tpl = "errors.tmpl"
	_, err = List{{Channel: "error1", Asset: asset.All}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on xpand pairs but not assets")

	_, err = List{{Channel: "error1"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errInvalidAssetExpandPairs, "Should error correctly on xpand pairs but not assets")

	_, err = List{{Channel: "error2"}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "wrong number of args for String", "Should error correctly with execution error")

	_, err = List{{Channel: "non-existent"}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errNoTemplateContent, "Should error correctly when no content generated")
	assert.ErrorContains(t, err, "non-existent", "Should error correctly when no content generated")

	_, err = List{{Channel: "error3", Asset: asset.All}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errAssetRecords, "Should error correctly when invalid number of asset entries")

	_, err = List{{Channel: "error4", Asset: asset.Spot}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, errPairRecords, "Should error correctly when invalid number of pair entries")

	_, err = List{{Channel: "error4", Asset: asset.Margin}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, asset.ErrInvalidAsset, "Should error correctly when invalid asset")

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
}
