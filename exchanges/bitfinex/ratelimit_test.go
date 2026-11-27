package bitfinex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) { //nolint:tparallel // the cases read the shared rateLimits map
	t.Parallel()
	limits := rateLimits
	// Serial, as the cases read the shared rateLimits map, which the checks after them read too
	for _, tc := range []struct {
		name      string
		epl       request.EndpointLimit
		perMinute float64
	}{
		{"platformStatus", platformStatusEPL, 30},
		{"ticker", tickerEPL, 90},
		{"tickers", tickersEPL, 30},
		{"tickersHistory", tickersHistoryEPL, 10},
		{"publicTrades", publicTradesEPL, 15},
		{"book", bookEPL, 240},
		{"stats", statsEPL, 15},
		{"candles", candlesEPL, 30},
		{"derivativesStatus", derivativesStatusEPL, 90},
		{"derivativesStatusHistory", derivativesStatusHistoryEPL, 90},
		{"liquidations", liquidationsEPL, 3},
		{"leaderboards", leaderboardsEPL, 90},
		{"fundingStatistics", fundingStatisticsEPL, 10},
		{"configs", configsEPL, 90},
		{"virtualAssetServiceProviders", virtualAssetServiceProvidersEPL, 10},
		{"marketAveragePrice", marketAveragePriceEPL, 90},
		{"foreignExchangeRate", foreignExchangeRateEPL, 90},
		{"wallets", walletsEPL, 90},
		{"retrieveOrders", retrieveOrdersEPL, 90},
		{"retrieveOrdersBySymbol", retrieveOrdersBySymbolEPL, 90},
		{"submitOrder", submitOrderEPL, 90},
		{"updateOrder", updateOrderEPL, 90},
		{"cancelOrder", cancelOrderEPL, 90},
		{"cancelOrders", cancelOrdersEPL, 90},
		{"orderMultiOp", orderMultiOpEPL, 90},
		{"ordersHistory", ordersHistoryEPL, 90},
		{"ordersHistoryBySymbol", ordersHistoryBySymbolEPL, 90},
		{"orderTrades", orderTradesEPL, 90},
		{"trades", tradesEPL, 90},
		{"tradesBySymbol", tradesBySymbolEPL, 90},
		{"overTheCounterOrdersHistory", overTheCounterOrdersHistoryEPL, 90},
		{"ledgers", ledgersEPL, 90},
		{"marginInfo", marginInfoEPL, 90},
		{"positions", positionsEPL, 90},
		{"claimPosition", claimPositionEPL, 90},
		{"increasePosition", increasePositionEPL, 90},
		{"increasePositionInfo", increasePositionInfoEPL, 90},
		{"positionsHistory", positionsHistoryEPL, 90},
		{"positionsSnapshot", positionsSnapshotEPL, 90},
		{"positionsAudit", positionsAuditEPL, 90},
		{"updatePositionFundingType", updatePositionFundingTypeEPL, 10},
		{"derivativePositionCollateral", derivativePositionCollateralEPL, 90},
		{"derivativePositionCollateralLimits", derivativePositionCollateralLimitsEPL, 90},
		{"fundingOffers", fundingOffersEPL, 90},
		{"submitFundingOffer", submitFundingOfferEPL, 90},
		{"cancelFundingOffer", cancelFundingOfferEPL, 90},
		{"cancelAllFundingOffers", cancelAllFundingOffersEPL, 90},
		{"fundingClose", fundingCloseEPL, 90},
		{"fundingAutoRenew", fundingAutoRenewEPL, 90},
		{"keepFunding", keepFundingEPL, 90},
		{"fundingOffersHistory", fundingOffersHistoryEPL, 90},
		{"fundingLoans", fundingLoansEPL, 90},
		{"fundingLoansHistory", fundingLoansHistoryEPL, 90},
		{"fundingCredits", fundingCreditsEPL, 90},
		{"fundingCreditsHistory", fundingCreditsHistoryEPL, 90},
		{"fundingTrades", fundingTradesEPL, 90},
		{"fundingInfo", fundingInfoEPL, 90},
		{"userInfo", userInfoEPL, 90},
		{"summary", summaryEPL, 90},
		{"loginHistory", loginHistoryEPL, 90},
		{"keyPermissions", keyPermissionsEPL, 90},
		{"generateToken", generateTokenEPL, 10},
		{"changelog", changelogEPL, 90},
		{"transfer", transferEPL, 10},
		{"depositAddress", depositAddressEPL, 10},
		{"depositAddressList", depositAddressListEPL, 10},
		{"generateInvoice", generateInvoiceEPL, 10},
		{"lightningNetworkInvoicePayments", lightningNetworkInvoicePaymentsEPL, 10},
		{"withdrawal", withdrawalEPL, 10},
		{"movements", movementsEPL, 90},
		{"movementInfo", movementInfoEPL, 90},
		{"alertList", alertListEPL, 45},
		{"alertSet", alertSetEPL, 90},
		{"alertDelete", alertDeleteEPL, 90},
		{"balanceAvailable", balanceAvailableEPL, 90},
		{"userSettingsWrite", userSettingsWriteEPL, 90},
		{"userSettingsRead", userSettingsReadEPL, 90},
		{"userSettingsDelete", userSettingsDeleteEPL, 90},
		{"thalexDeposit", thalexDepositEPL, 10},
		{"thalexWithdrawal", thalexWithdrawalEPL, 10},
		{"thalexFreeTransferCount", thalexFreeTransferCountEPL, 10},
		{"fiatWithdrawal", fiatWithdrawalEPL, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Containsf(t, limits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
			assert.InDeltaf(t, tc.perMinute/60, float64(limits[tc.epl].Limit()), 1e-9, "rateLimits should refill endpoint limit %d at its documented rate", tc.epl)
			assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d as one request", tc.epl)
		})
	}
	assert.Len(t, limits, 82, "rateLimits should define a limit for every endpoint and no more")
	for a, limitA := range limits {
		for b, limitB := range limits {
			if a != b {
				assert.Falsef(t, limitA.SharesBudgetWith(limitB), "Endpoint limits %v and %v should not share a budget, as Bitfinex limits each endpoint separately", a, b)
			}
		}
	}
}

func TestRateLimitsAreSharedAcrossInstances(t *testing.T) {
	t.Parallel()
	first, second := new(Exchange), new(Exchange)
	first.SetDefaults()
	second.SetDefaults()
	firstLimits := first.Requester.GetRateLimiterDefinitions()
	secondLimits := second.Requester.GetRateLimiterDefinitions()
	require.Len(t, secondLimits, len(firstLimits), "both instances must define the same endpoints")
	for epl, limiter := range firstLimits {
		assert.Truef(t, limiter.SharesBudgetWith(secondLimits[epl]), "endpoint %d should draw on the same budget in both instances, as Bitfinex limits each IP", epl)
	}
}
