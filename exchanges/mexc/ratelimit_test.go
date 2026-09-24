package mexc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"systemTime":                         systemTimeEPL,
		"defaultSymbols":                     defaultSymbolsEPL,
		"getSymbols":                         getSymbolsEPL,
		"orderbooks":                         orderbooksEPL,
		"recentTradesList":                   recentTradesListEPL,
		"aggregatedTrades":                   aggregatedTradesEPL,
		"candlestick":                        candlestickEPL,
		"currentAveragePrice":                currentAveragePriceEPL,
		"symbolTickerPriceChangeStat":        symbolTickerPriceChangeStatEPL,
		"symbolsTickerPriceChangeStat":       symbolsTickerPriceChangeStatEPL,
		"symbolPriceTicker":                  symbolPriceTickerEPL,
		"symbolsPriceTicker":                 symbolsPriceTickerEPL,
		"symbolOrderbookTicker":              symbolOrderbookTickerEPL,
		"createSubAccount":                   createSubAccountEPL,
		"subAccountList":                     subAccountListEPL,
		"createAPIKeyForSubAccount":          createAPIKeyForSubAccountEPL,
		"getSubAccountAPIKey":                getSubAccountAPIKeyEPL,
		"deleteSubAccountAPIKey":             deleteSubAccountAPIKeyEPL,
		"subAccountUniversalTransfer":        subAccountUniversalTransferEPL,
		"getSubaccUnversalTransfers":         getSubaccUnversalTransfersEPL,
		"getSubAccountAsset":                 getSubAccountAssetEPL,
		"getKYCStatus":                       getKYCStatusEPL,
		"selfSymbols":                        selfSymbolsEPL,
		"newOrder":                           newOrderEPL,
		"createBatchOrders":                  createBatchOrdersEPL,
		"cancelTradeOrder":                   cancelTradeOrderEPL,
		"cancelAllOpenOrdersBySymbol":        cancelAllOpenOrdersBySymbolEPL,
		"getOrderByID":                       getOrderByIDEPL,
		"getOpenOrders":                      getOpenOrdersEPL,
		"allOrders":                          allOrdersEPL,
		"accountInformation":                 accountInformationEPL,
		"accountTradeList":                   accountTradeListEPL,
		"enableMXDeduct":                     enableMXDeductEPL,
		"getMXDeductStatus":                  getMXDeductStatusEPL,
		"getSymbolTradingFee":                getSymbolTradingFeeEPL,
		"getCurrencyInformation":             getCurrencyInformationEPL,
		"withdrawCapital":                    withdrawCapitalEPL,
		"cancelWithdrawal":                   cancelWithdrawalEPL,
		"getFundDepositHistory":              getFundDepositHistoryEPL,
		"getWithdrawalHistory":               getWithdrawalHistoryEPL,
		"generateDepositAddress":             generateDepositAddressEPL,
		"getDepositAddress":                  getDepositAddressEPL,
		"getWithdrawalAddress":               getWithdrawalAddressEPL,
		"userUniversalTransfer":              userUniversalTransferEPL,
		"getUniversalTransferDetailByID":     getUniversalTransferDetailByIDEPL,
		"getAssetConvertedMX":                getAssetConvertedMXEPL,
		"dustConvert":                        dustConvertEPL,
		"dustLog":                            dustLogEPL,
		"internalTransfer":                   internalTransferEPL,
		"getInternalTransferHistory":         getInternalTransferHistoryEPL,
		"getUniversalTransferhistory":        getUniversalTransferhistoryEPL,
		"getUserRebateHistory":               getUserRebateHistoryEPL,
		"getRebateRecordsDetail":             getRebateRecordsDetailEPL,
		"selfRebateRecordsDetails":           selfRebateRecordsDetailsEPL,
		"getReferCode":                       getReferCodeEPL,
		"getAffilateCommissionRecord":        getAffilateCommissionRecordEPL,
		"getAffilateWithdrawRecord":          getAffilateWithdrawRecordEPL,
		"getAffiliateConnissionDetail":       getAffiliateConnissionDetailEPL,
		"affiliateCampaignData":              affiliateCampaignDataEPL,
		"affiliateReferralData":              affiliateReferralDataEPL,
		"subAffiliateData":                   subAffiliateDataEPL,
		"getUID":                             getUIDEPL,
		"getAPIKeyInfo":                      getAPIKeyInfoEPL,
		"setAPIKeyInfo":                      setAPIKeyInfoEPL,
		"offlineSymbols":                     offlineSymbolsEPL,
		"announcements":                      announcementsEPL,
		"createSelfTradePreventionGroup":     createSelfTradePreventionGroupEPL,
		"getSelfTradePreventionGroup":        getSelfTradePreventionGroupEPL,
		"deleteSelfTradePreventionGroup":     deleteSelfTradePreventionGroupEPL,
		"addSelfTradePreventionGroupUIDs":    addSelfTradePreventionGroupUIDsEPL,
		"deleteSelfTradePreventionGroupUIDs": deleteSelfTradePreventionGroupUIDsEPL,
		"listenKey":                          listenKeyEPL,
		"broker":                             brokerEPL,
	}
	rl, err := request.New("rateLimitTest2", &http.Client{}, request.WithLimiter(rateLimits))
	require.NoError(t, err)
	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := rl.InitiateRateLimit(t.Context(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}

// TestRateLimitWeightsMatchDocumentation pins the endpoint weights re-derived from MEXC's current
// spot v3 documentation, so a regression to the retired 500-per-10-second table (every weight of
// which matched that page exactly) is caught. The budgets the weights draw on are pinned separately
// by TestRateLimitPoolBudgets.
func TestRateLimitWeightsMatchDocumentation(t *testing.T) {
	t.Parallel()
	rl := rateLimits
	for _, tc := range []struct {
		name   string
		epl    request.EndpointLimit
		weight request.Weight
	}{
		{"getSymbols", getSymbolsEPL, 25},
		{"orderbooks", orderbooksEPL, 3},
		{"symbolTickerPriceChangeStat", symbolTickerPriceChangeStatEPL, 25},
		{"symbolsTickerPriceChangeStat", symbolsTickerPriceChangeStatEPL, 25},
		{"symbolPriceTicker", symbolPriceTickerEPL, 10},
		{"symbolsPriceTicker", symbolsPriceTickerEPL, 10},
		{"symbolOrderbookTicker", symbolOrderbookTickerEPL, 10},
		{"newOrder", newOrderEPL, 1},
		{"createBatchOrders", createBatchOrdersEPL, 1},
		{"cancelTradeOrder", cancelTradeOrderEPL, 1},
		{"cancelAllOpenOrdersBySymbol", cancelAllOpenOrdersBySymbolEPL, 1},
		{"cancelAllOrders", cancelAllOrdersEPL, 1},
		{"withdrawCapital", withdrawCapitalEPL, 1},
		{"getUID", getUIDEPL, 1},
		{"getAPIKeyInfo", getAPIKeyInfoEPL, 1},
		{"setAPIKeyInfo", setAPIKeyInfoEPL, 1},
		{"offlineSymbols", offlineSymbolsEPL, 10},
		{"createSelfTradePreventionGroup", createSelfTradePreventionGroupEPL, 20},
		{"getSelfTradePreventionGroup", getSelfTradePreventionGroupEPL, 20},
		{"deleteSelfTradePreventionGroup", deleteSelfTradePreventionGroupEPL, 20},
		{"addSelfTradePreventionGroupUIDs", addSelfTradePreventionGroupUIDsEPL, 20},
		{"deleteSelfTradePreventionGroupUIDs", deleteSelfTradePreventionGroupUIDsEPL, 20},
		{"listenKey", listenKeyEPL, 1},
		{"broker", brokerEPL, 1},
	} {
		limiter, ok := rl[tc.epl]
		require.Truef(t, ok, "%s must have a rate limiter", tc.name)
		assert.Equalf(t, tc.weight, limiter.Weight(), "%s weight should match the documentation", tc.name)
	}
}

// TestRateLimitPoolBudgets pins the shared budgets themselves, which the weight table cannot express:
// the IP-weighted endpoints draw on 300 per 10 seconds, and place, batch, cancel and cancel-all draw
// on one shared 12 per second rather than 12 each.
func TestRateLimitPoolBudgets(t *testing.T) {
	t.Parallel()
	rl := rateLimits
	for _, tc := range []struct {
		name string
		epl  request.EndpointLimit
		rate rate.Limit
	}{
		{"IP pool at weight 1", systemTimeEPL, 30},
		{"shared order budget", newOrderEPL, 12},
		{"announcements at 5 per 2 seconds", announcementsEPL, 2.5},
	} {
		assert.Equalf(t, tc.rate, rl[tc.epl].Limit(), "%s should draw on a budget of %v actions per second", tc.name, tc.rate)
	}
	for _, epl := range []struct {
		name string
		epl  request.EndpointLimit
	}{
		{"createBatchOrders", createBatchOrdersEPL},
		{"cancelTradeOrder", cancelTradeOrderEPL},
		{"cancelAllOpenOrdersBySymbol", cancelAllOpenOrdersBySymbolEPL},
		{"cancelAllOrders", cancelAllOrdersEPL},
	} {
		assert.Truef(t, rl[epl.epl].SharesBudgetWith(rl[newOrderEPL]), "%s should draw on the same budget as newOrder, not an identical one of its own", epl.name)
	}
	assert.False(t, rl[announcementsEPL].SharesBudgetWith(rl[systemTimeEPL]), "announcements should be limited outside the weighted IP pool")
	assert.True(t, rl[listenKeyEPL].SharesBudgetWith(rl[systemTimeEPL]), "the listen key endpoints should draw on the weighted IP pool")
	assert.True(t, rl[brokerEPL].SharesBudgetWith(rl[systemTimeEPL]), "the broker endpoints should draw on the weighted IP pool")
}

// TestRateLimitsAreSharedAcrossInstances asserts two instances built by SetDefaults draw every endpoint on
// the same budget: the venue's budgets are per IP address and per account, so each instance carrying its
// own limiters would let two of them together exceed the venue's limit.
func TestRateLimitsAreSharedAcrossInstances(t *testing.T) {
	t.Parallel()
	first, second := new(Exchange), new(Exchange)
	first.SetDefaults()
	second.SetDefaults()
	firstLimits := first.Requester.GetRateLimiterDefinitions()
	secondLimits := second.Requester.GetRateLimiterDefinitions()
	require.Len(t, secondLimits, len(firstLimits), "both instances must define the same endpoints")
	for epl, limiter := range firstLimits {
		assert.Truef(t, limiter.SharesBudgetWith(secondLimits[epl]), "endpoint %d should draw on the same budget in both instances", epl)
	}
}
