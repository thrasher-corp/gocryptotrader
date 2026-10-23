package gate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetUnifiedUserRiskUnitDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	got, err := e.GetUnifiedUserRiskUnitDetails(t.Context())
	require.NoError(t, err, "GetUnifiedUserRiskUnitDetails must not error")
	if mockTests {
		exp := &UserRiskUnitDetailsResponse{
			UserID:    10001,
			SpotHedge: true,
			RiskUnits: []RiskUnits{{
				Symbol:         currency.BTC,
				SpotInUse:      -13500.000001223,
				MaintainMargin: 2334.002,
				InitialMargin:  2334.002,
				Delta:          0.22,
				Gamma:          0.42,
				Theta:          0.29,
				Vega:           0.22,
			}},
		}
		assert.Equal(t, exp, got, "GetUnifiedUserRiskUnitDetails should decode every field")
		return
	}
	assert.NotNil(t, got, "GetUnifiedUserRiskUnitDetails should return the risk units")
}

func TestGetFuturesRiskTable(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskTable(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesRiskTable must reject an empty settlement currency")

	_, err = e.GetFuturesRiskTable(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errTableIDEmpty, "GetFuturesRiskTable must reject an empty table ID")

	// A risk limit table ID is only exposed on a position; no public endpoint returns one.
	// Mock runs use the recorded fixture ID; live runs source it from an open USDT futures position.
	tableID := "BTC_USDT_20260626" // matches the recorded mock fixture's table_id query
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		positions, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true, 0, 0)
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
	if mockTests {
		exp := []*RiskTableResponse{
			{
				Tier:            1,
				RiskLimit:       10000,
				InitialRate:     0.025,
				MaintenanceRate: 0.015,
				LeverageMax:     40,
			},
			{
				Tier:            2,
				RiskLimit:       30000,
				InitialRate:     0.03333,
				MaintenanceRate: 0.02,
				LeverageMax:     30,
				Deduction:       50,
			},
		}
		assert.Equal(t, exp, got, "GetFuturesRiskTable should decode every field")
		return
	}
	assert.NotEmpty(t, got, "GetFuturesRiskTable should return tiers")
}

func TestGetFuturesRiskLimitTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesRiskLimitTiers must reject an empty settlement currency")

	_, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 10, 10)
	require.ErrorIs(t, err, errPagingNotAllowed, "GetFuturesRiskLimitTiers must reject a limit and offset with a contract")

	_, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 0, 10)
	require.ErrorIs(t, err, errPagingNotAllowed, "GetFuturesRiskLimitTiers must reject an offset with a contract")

	got, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, 10)
	require.NoError(t, err, "GetFuturesRiskLimitTiers must not error for a page of markets")
	require.NotEmpty(t, got, "GetFuturesRiskLimitTiers must return tiers for a page of markets")

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	avail, err := e.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, avail, "GetAvailablePairs must return pairs")

	got, err = e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err, "GetFuturesRiskLimitTiers must not error for a contract")
	if mockTests {
		exp := []*RiskTableResponse{
			{
				Tier:            1,
				RiskLimit:       200000,
				InitialRate:     0.005,
				MaintenanceRate: 0.003,
				LeverageMax:     200,
			},
		}
		assert.Equal(t, exp, got, "GetFuturesRiskLimitTiers should decode every field")
		return
	}
	assert.NotEmpty(t, got, "GetFuturesRiskLimitTiers should return the contract's tiers")
}

func TestGetDeliveryRiskLimitTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryRiskLimitTiers must reject an empty settlement currency")

	_, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 10, 10)
	require.ErrorIs(t, err, errPagingNotAllowed, "GetDeliveryRiskLimitTiers must reject paging a contract's tiers")

	_, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.NewBTCUSDT(), 0, 10)
	require.ErrorIs(t, err, errPagingNotAllowed, "GetDeliveryRiskLimitTiers must reject an offset with a contract")

	got, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, 10)
	require.NoError(t, err, "GetDeliveryRiskLimitTiers must not error for a page of markets")
	require.NotEmpty(t, got, "GetDeliveryRiskLimitTiers must return tiers for a page of markets")
	// The recorded tiers are the documentation's example, which lists a ZTX_USDT market
	exp := &RiskTableResponse{
		Tier:            1,
		RiskLimit:       20000,
		InitialRate:     0.02,
		MaintenanceRate: 0.01,
		LeverageMax:     50,
		Contract:        currency.NewPairWithDelimiter("ZTX", "USDT", currency.UnderscoreDelimiter),
	}
	if mockTests {
		require.Len(t, got, 5, "GetDeliveryRiskLimitTiers must return every recorded tier")
		assert.Equal(t, exp, got[0], "GetDeliveryRiskLimitTiers should decode every field")
	} else {
		testexch.UpdatePairsOnce(t, e)
	}
	avail, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, avail, "GetAvailablePairs must return delivery pairs")

	got, err = e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err, "GetDeliveryRiskLimitTiers must not error for a contract")
	if mockTests {
		assert.Equal(t, []*RiskTableResponse{exp}, got, "GetDeliveryRiskLimitTiers should decode a contract's tiers")
		return
	}
	assert.NotEmpty(t, got, "GetDeliveryRiskLimitTiers should return a contract's tiers")
}

func TestDeliveryUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "DeliveryUpdatePositionRiskLimit must reject an empty settlement currency")

	_, err = e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "DeliveryUpdatePositionRiskLimit must reject an empty contract")

	_, err = e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.NewBTCUSD(), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit, "DeliveryUpdatePositionRiskLimit must reject a zero risk limit")

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	avail, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, avail, "GetAvailablePairs must return delivery pairs")

	tiers, err := e.GetDeliveryRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err, "GetDeliveryRiskLimitTiers must not error")
	require.NotEmpty(t, tiers, "GetDeliveryRiskLimitTiers must return tiers")

	lowestTierRiskLimit := tiers[0].RiskLimit.Float64()
	got, err := e.DeliveryUpdatePositionRiskLimit(t.Context(), currency.USDT, avail[0], lowestTierRiskLimit)
	require.NoError(t, err, "DeliveryUpdatePositionRiskLimit must not error")
	if mockTests {
		exp := deliveryPositionFixture(avail[0])
		exp.RiskLimit = 20000
		assert.Equal(t, exp, got, "DeliveryUpdatePositionRiskLimit should decode every field")
		return
	}
	assert.Equal(t, lowestTierRiskLimit, got.RiskLimit.Float64(), "DeliveryUpdatePositionRiskLimit should apply the requested risk limit")
}

func TestFuturesUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesUpdatePositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FuturesUpdatePositionRiskLimit must reject an empty settlement currency")

	_, err = e.FuturesUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesUpdatePositionRiskLimit must reject an empty contract")

	_, err = e.FuturesUpdatePositionRiskLimit(t.Context(), currency.USDT, currency.NewBTCUSD(), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit, "FuturesUpdatePositionRiskLimit must reject a zero risk limit")

	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	avail, err := e.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, avail, "GetAvailablePairs must return pairs")

	tiers, err := e.GetFuturesRiskLimitTiers(t.Context(), currency.USDT, avail[0], 0, 0)
	require.NoError(t, err, "GetFuturesRiskLimitTiers must not error")
	require.NotEmpty(t, tiers, "GetFuturesRiskLimitTiers must return the contract's tiers")

	lowestTierRiskLimit := tiers[0].RiskLimit.Float64()
	got, err := e.FuturesUpdatePositionRiskLimit(t.Context(), currency.USDT, avail[0], lowestTierRiskLimit)
	require.NoError(t, err, "FuturesUpdatePositionRiskLimit must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, got, "FuturesUpdatePositionRiskLimit should decode every field")
		return
	}
	require.NotNil(t, got, "FuturesUpdatePositionRiskLimit must return the position")
	assert.Equal(t, lowestTierRiskLimit, got.RiskLimit.Float64(), "FuturesUpdatePositionRiskLimit should apply the lowest tier's risk limit")
}
