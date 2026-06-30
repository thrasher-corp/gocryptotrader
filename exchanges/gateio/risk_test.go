package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetUnifiedUserRiskUnitDetails(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	got, err := e.GetUnifiedUserRiskUnitDetails(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

func TestGetFuturesRiskTable(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskTable(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFuturesRiskTable(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errTableIDEmpty)

	// A risk limit table ID is only exposed on a Position; no public endpoint returns one.
	// Mock runs use the recorded fixture ID; live runs source it from an open USDT futures position.
	tableID := "BTC_USDT_20260626" // matches the recorded mock fixture's table_id query
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		positions, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true)
		require.NoError(t, err, "GetAllFuturesPositionsOfUsers must not error")
		tableID = ""
		for _, p := range positions {
			if p.RiskLimitTable != "" {
				tableID = p.RiskLimitTable
				break
			}
		}
		if tableID == "" {
			t.Skip("no open USDT futures position with a risk limit table to test against")
		}
	}

	got, err := e.GetFuturesRiskTable(t.Context(), currency.USDT, tableID)
	require.NoError(t, err, "GetFuturesRiskTable must not error")
	assert.NotEmpty(t, got, "GetFuturesRiskTable should return tiers")
}

func TestGetFuturesRiskLimitTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 10, 10)
	require.ErrorIs(t, err, errPagingNotAllowed)

	_, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 0, 10)
	require.ErrorIs(t, err, errPagingNotAllowed)

	got, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, 10)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	avail, err := e.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, avail)

	got, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestGetDeliveryRiskLimitTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 10, 10)
	require.ErrorIs(t, err, errPagingNotAllowed)

	_, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 0, 10)
	require.ErrorIs(t, err, errPagingNotAllowed)

	got, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, 10)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	avail, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err)
	require.NotEmpty(t, avail)

	got, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestDeliveryUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.NewBTCUSD(), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit)

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	avail, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err)
	require.NotEmpty(t, avail)

	tiers, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, tiers)

	lowestTierRiskLimit := float64(tiers[0].RiskLimit)
	got, err := e.DeliveryUpdatePositionRiskLimit(request.WithVerbose(t.Context()), currency.USDT, avail[0], lowestTierRiskLimit)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestFuturesUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesUpdatePositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.FuturesUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.FuturesUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.NewBTCUSD(), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit)

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	avail, err := e.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, avail)

	tiers, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, tiers)

	lowestTierRiskLimit := float64(tiers[0].RiskLimit)
	got, err := e.FuturesUpdatePositionRiskLimit(request.WithVerbose(t.Context()), currency.USDT, avail[0], lowestTierRiskLimit)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	assert.Equal(t, lowestTierRiskLimit, got.RiskLimit.Float64())
}

func TestFuturesUpdatePositionRiskLimitDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesUpdatePositionRiskLimitDualMode(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.FuturesUpdatePositionRiskLimitDualMode(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.FuturesUpdatePositionRiskLimitDualMode(t.Context(), currency.USDT, currency.NewBTCUSD(), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit)

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	avail, err := e.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, avail)

	tiers, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, tiers)

	lowestTierRiskLimit := float64(tiers[0].RiskLimit)
	got, err := e.FuturesUpdatePositionRiskLimitDualMode(t.Context(), currency.USDT, avail[0], lowestTierRiskLimit)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	assert.Equal(t, lowestTierRiskLimit, got.RiskLimit.Float64())
}
