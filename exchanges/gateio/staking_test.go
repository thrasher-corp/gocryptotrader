package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetStakingCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingCoins(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwapStakingCoins(t *testing.T) {
	t.Parallel()
	_, err := e.SwapStakingCoins(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "nil arg must return ErrNilPointer")

	_, err = e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Side: 0, Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty coin must return ErrCurrencyCodeEmpty")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Coin: "ETH", Side: 0, Amount: 0.01})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingOrders(t.Context(), currency.EMPTYCODE, 0, -1, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingDividendRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingDividendRecords(t.Context(), currency.EMPTYCODE, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingAssets(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
