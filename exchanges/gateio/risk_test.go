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

	// mock HTTP response due to dynamically generated table IDs, which can only be retrieved via authenticated endpoint
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))
	require.NoError(t, testexch.MockHTTPInstance(e, "/"))

	got, err := e.GetFuturesRiskTable(t.Context(), currency.USDT, "BTC_USDT_202507040223")
	require.NoError(t, err)
	assert.NotEmpty(t, got)
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

	testexch.UpdatePairsOnce(t, e)
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

	testexch.UpdatePairsOnce(t, e)
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, e)
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, e)
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, e)
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
