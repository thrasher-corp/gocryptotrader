package bitget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.Futures)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.Margin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.CrossMargin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountBalances(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}
