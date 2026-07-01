package bitget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	ta := []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.Margin, asset.CrossMargin}
	for _, a := range ta {
		_, err = e.UpdateAccountBalances(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateAccountBalances should not error for asset type %s", a)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	err := e.UpdateTickers(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ta := []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.Margin}
	for _, a := range ta {
		err = e.UpdateTickers(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateTickers should not error for asset type %s", a)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Binary)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		assert.NoErrorf(t, err, "should not error for asset type %s", a)
		assert.NotEmptyf(t, pairs, "should not be empty for asset type %s", a)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ta := []asset.Item{asset.Spot, asset.USDCMarginedFutures, asset.Margin}
	for _, a := range ta {
		err = e.UpdateOrderExecutionLimits(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateOrderExecutionLimits should not error for asset type %s", a)
	}
}
