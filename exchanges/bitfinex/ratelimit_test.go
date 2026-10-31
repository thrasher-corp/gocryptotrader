package bitfinex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetRateLimit(t *testing.T) {
	t.Parallel()
	limits := GetRateLimit()
	for _, tc := range []struct {
		epl       request.EndpointLimit
		perMinute float64
	}{
		{platformStatusEPL, 30},
		{tickerEPL, 90},
		{tickersEPL, 30},
		{tickersHistoryEPL, 10},
		{publicTradesEPL, 15},
		{bookEPL, 240},
		{statsEPL, 15},
		{candlesEPL, 30},
		{derivativesStatusEPL, 90},
		{derivativesStatusHistoryEPL, 90},
		{liquidationsEPL, 3},
		{leaderboardsEPL, 90},
		{fundingStatisticsEPL, 10},
		{configsEPL, 90},
		{virtualAssetServiceProvidersEPL, 10},
		{marketAveragePriceEPL, 90},
		{foreignExchangeRateEPL, 90},
		{walletsEPL, 90},
		{retrieveOrdersEPL, 90},
		{retrieveOrdersBySymbolEPL, 90},
		{submitOrderEPL, 90},
		{updateOrderEPL, 90},
		{cancelOrderEPL, 90},
		{cancelOrdersEPL, 90},
		{orderMultiOpEPL, 90},
		{ordersHistoryEPL, 90},
		{ordersHistoryBySymbolEPL, 90},
		{orderTradesEPL, 90},
		{tradesEPL, 90},
		{tradesBySymbolEPL, 90},
		{otcOrdersHistoryEPL, 90},
		{ledgersEPL, 90},
		{marginInfoEPL, 90},
		{positionsEPL, 90},
		{claimPositionEPL, 90},
		{increasePositionEPL, 90},
		{increasePositionInfoEPL, 90},
		{positionsHistoryEPL, 90},
		{positionsSnapshotEPL, 90},
		{positionsAuditEPL, 90},
		{updatePositionFundingTypeEPL, 10},
		{derivativePositionCollateralEPL, 90},
		{derivativePositionCollateralLimitsEPL, 90},
		{fundingOffersEPL, 90},
		{submitFundingOfferEPL, 90},
		{cancelFundingOfferEPL, 90},
		{cancelAllFundingOffersEPL, 90},
		{fundingCloseEPL, 90},
		{fundingAutoRenewEPL, 90},
		{keepFundingEPL, 90},
		{fundingOffersHistoryEPL, 90},
		{fundingLoansEPL, 90},
		{fundingLoansHistoryEPL, 90},
		{fundingCreditsEPL, 90},
		{fundingCreditsHistoryEPL, 90},
		{fundingTradesEPL, 90},
		{fundingInfoEPL, 90},
		{userInfoEPL, 90},
		{summaryEPL, 90},
		{loginHistoryEPL, 90},
		{keyPermissionsEPL, 90},
		{generateTokenEPL, 10},
		{changelogEPL, 90},
		{transferEPL, 10},
		{depositAddressEPL, 10},
		{depositAddressListEPL, 10},
		{generateInvoiceEPL, 10},
		{lnxInvoicePaymentsEPL, 10},
		{withdrawalEPL, 10},
		{movementsEPL, 90},
		{movementInfoEPL, 90},
		{alertListEPL, 45},
		{alertSetEPL, 90},
		{alertDeleteEPL, 90},
		{balanceAvailableEPL, 90},
		{userSettingsWriteEPL, 90},
		{userSettingsReadEPL, 90},
		{userSettingsDeleteEPL, 90},
		{thalexDepositEPL, 10},
		{thalexWithdrawalEPL, 10},
		{thalexFreeTransferCountEPL, 10},
		{fiatWithdrawalEPL, 10},
	} {
		require.Containsf(t, limits, tc.epl, "GetRateLimit must define endpoint limit %d", tc.epl)
		assert.InDeltaf(t, tc.perMinute/60, float64(limits[tc.epl].Limit()), 1e-9, "GetRateLimit should refill endpoint limit %d at its documented rate", tc.epl)
		assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "GetRateLimit should weigh endpoint limit %d as one request", tc.epl)
	}
	assert.Len(t, limits, 82, "GetRateLimit should define a limit for every endpoint and no more")
	for a, limitA := range limits {
		for b, limitB := range limits {
			if a != b {
				assert.Falsef(t, limitA.SharesBudgetWith(limitB), "Endpoint limits %v and %v should not share a budget, as Bitfinex limits each endpoint separately", a, b)
			}
		}
	}
}
