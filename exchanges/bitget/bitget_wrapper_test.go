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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.Margin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.CrossMargin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	err := e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.CoinMarginedFutures)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.USDCMarginedFutures)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.UpdateTickers(t.Context(), asset.Margin)
	assert.NoError(t, err)
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
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.USDCMarginedFutures)
	assert.NoError(t, err)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Margin)
	assert.NoError(t, err)
}
