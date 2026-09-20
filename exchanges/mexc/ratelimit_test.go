package mexc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"systemTime":                     systemTimeEPL,
		"defaultSymbols":                 defaultSymbolsEPL,
		"getSymbols":                     getSymbolsEPL,
		"orderbooks":                     orderbooksEPL,
		"recentTradesList":               recentTradesListEPL,
		"aggregatedTrades":               aggregatedTradesEPL,
		"candlestick":                    candlestickEPL,
		"currentAveragePrice":            currentAveragePriceEPL,
		"symbolTickerPriceChangeStat":    symbolTickerPriceChangeStatEPL,
		"symbolsTickerPriceChangeStat":   symbolsTickerPriceChangeStatEPL,
		"symbolPriceTicker":              symbolPriceTickerEPL,
		"symbolsPriceTicker":             symbolsPriceTickerEPL,
		"symbolOrderbookTicker":          symbolOrderbookTickerEPL,
		"createSubAccount":               createSubAccountEPL,
		"subAccountList":                 subAccountListEPL,
		"createAPIKeyForSubAccount":      createAPIKeyForSubAccountEPL,
		"getSubAccountAPIKey":            getSubAccountAPIKeyEPL,
		"deleteSubAccountAPIKey":         deleteSubAccountAPIKeyEPL,
		"subAccountUniversalTransfer":    subAccountUniversalTransferEPL,
		"getSubaccUnversalTransfers":     getSubaccUnversalTransfersEPL,
		"getSubAccountAsset":             getSubAccountAssetEPL,
		"getKYCStatus":                   getKYCStatusEPL,
		"selfSymbols":                    selfSymbolsEPL,
		"newOrder":                       newOrderEPL,
		"createBatchOrders":              createBatchOrdersEPL,
		"cancelTradeOrder":               cancelTradeOrderEPL,
		"cancelAllOpenOrdersBySymbol":    cancelAllOpenOrdersBySymbolEPL,
		"getOrderByID":                   getOrderByIDEPL,
		"getOpenOrders":                  getOpenOrdersEPL,
		"allOrders":                      allOrdersEPL,
		"accountInformation":             accountInformationEPL,
		"accountTradeList":               accountTradeListEPL,
		"enableMXDeduct":                 enableMXDeductEPL,
		"getMXDeductStatus":              getMXDeductStatusEPL,
		"getSymbolTradingFee":            getSymbolTradingFeeEPL,
		"getCurrencyInformation":         getCurrencyInformationEPL,
		"withdrawCapital":                withdrawCapitalEPL,
		"cancelWithdrawal":               cancelWithdrawalEPL,
		"getFundDepositHistory":          getFundDepositHistoryEPL,
		"getWithdrawalHistory":           getWithdrawalHistoryEPL,
		"generateDepositAddress":         generateDepositAddressEPL,
		"getDepositAddress":              getDepositAddressEPL,
		"getWithdrawalAddress":           getWithdrawalAddressEPL,
		"userUniversalTransfer":          userUniversalTransferEPL,
		"getUniversalTransferDetailByID": getUniversalTransferDetailByIDEPL,
		"getAssetConvertedMX":            getAssetConvertedMXEPL,
		"dustTransfer":                   dustTransferEPL,
		"dustLog":                        dustLogEPL,
		"internalTransfer":               internalTransferEPL,
		"getInternalTransferHistory":     getInternalTransferHistoryEPL,
		"capitalWithdrawal":              capitalWithdrawalEPL,
		"getUniversalTransferhistory":    getUniversalTransferhistoryEPL,
		"getUserRebateHistory":           getUserRebateHistoryEPL,
		"getRebateRecordsDetail":         getRebateRecordsDetailEPL,
		"selfRebateRecordsDetails":       selfRebateRecordsDetailsEPL,
		"getReferCode":                   getReferCodeEPL,
		"getAffilateCommissionRecord":    getAffilateCommissionRecordEPL,
		"getAffilateWithdrawRecord":      getAffilateWithdrawRecordEPL,
		"getAffiliateConnissionDetail":   getAffiliateConnissionDetailEPL,
		"affiliateCampaignData":          affiliateCampaignDataEPL,
		"affiliateReferralData":          affiliateReferralDataEPL,
		"subAffiliateData":               subAffiliateDataEPL,
	}
	rl, err := request.New("rateLimitTest2", &http.Client{}, request.WithLimiter(GetRateLimit()))
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
// which matched that page exactly) is caught. The IP pool and the shared order-endpoint budget are
// applied through the same limiter, and only the weight is exposed for inspection here.
func TestRateLimitWeightsMatchDocumentation(t *testing.T) {
	t.Parallel()
	rl := GetRateLimit()
	for _, tc := range []struct {
		name   string
		epl    request.EndpointLimit
		weight request.Weight
	}{
		{"getSymbols", getSymbolsEPL, 25},
		{"orderbooks", orderbooksEPL, 3},
		{"symbolTickerPriceChangeStat", symbolTickerPriceChangeStatEPL, 25},
		{"symbolsTickerPriceChangeStat", symbolsTickerPriceChangeStatEPL, 40},
		{"symbolPriceTicker", symbolPriceTickerEPL, 10},
		{"symbolsPriceTicker", symbolsPriceTickerEPL, 10},
		{"symbolOrderbookTicker", symbolOrderbookTickerEPL, 10},
		{"newOrder", newOrderEPL, 1},
		{"createBatchOrders", createBatchOrdersEPL, 1},
		{"cancelTradeOrder", cancelTradeOrderEPL, 1},
	} {
		limiter, ok := rl[tc.epl]
		require.Truef(t, ok, "%s must have a rate limiter", tc.name)
		assert.Equalf(t, tc.weight, limiter.Weight(), "%s weight should match the documentation", tc.name)
	}
}
