package mexc

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"systemTime":                       systemTimeEPL,
		"defaultSymbols":                   defaultSymbolsEPL,
		"getSymbols":                       getSymbolsEPL,
		"orderbooks":                       orderbooksEPL,
		"recentTradesList":                 recentTradesListEPL,
		"aggregatedTrades":                 aggregatedTradesEPL,
		"candlestick":                      candlestickEPL,
		"currentAveragePrice":              currentAveragePriceEPL,
		"symbolTickerPriceChangeStat":      symbolTickerPriceChangeStatEPL,
		"symbolsTickerPriceChangeStat":     symbolsTickerPriceChangeStatEPL,
		"symbolPriceTicker":                symbolPriceTickerEPL,
		"symbolsPriceTicker":               symbolsPriceTickerEPL,
		"symbolOrderbookTicker":            symbolOrderbookTickerEPL,
		"createSubAccount":                 createSubAccountEPL,
		"subAccountList":                   subAccountListEPL,
		"createAPIKeyForSubAccount":        createAPIKeyForSubAccountEPL,
		"getSubAccountAPIKey":              getSubAccountAPIKeyEPL,
		"deleteSubAccountAPIKey":           deleteSubAccountAPIKeyEPL,
		"subAccountUniversalTransfer":      subAccountUniversalTransferEPL,
		"getSubaccUnversalTransfers":       getSubaccUnversalTransfersEPL,
		"getSubAccountAsset":               getSubAccountAssetEPL,
		"getKYCStatus":                     getKYCStatusEPL,
		"selfSymbols":                      selfSymbolsEPL,
		"newOrder":                         newOrderEPL,
		"createBatchOrders":                createBatchOrdersEPL,
		"cancelTradeOrder":                 cancelTradeOrderEPL,
		"cancelAllOpenOrdersBySymbol":      cancelAllOpenOrdersBySymbolEPL,
		"getOrderByID":                     getOrderByIDEPL,
		"getOpenOrders":                    getOpenOrdersEPL,
		"allOrders":                        allOrdersEPL,
		"accountInformation":               accountInformationEPL,
		"accountTradeList":                 accountTradeListEPL,
		"enableMXDeduct":                   enableMXDeductEPL,
		"getMXDeductStatus":                getMXDeductStatusEPL,
		"getSymbolTradingFee":              getSymbolTradingFeeEPL,
		"getCurrencyInformation":           getCurrencyInformationEPL,
		"withdrawCapital":                  withdrawCapitalEPL,
		"cancelWithdrawal":                 cancelWithdrawalEPL,
		"getFundDepositHistory":            getFundDepositHistoryEPL,
		"getWithdrawalHistory":             getWithdrawalHistoryEPL,
		"generateDepositAddress":           generateDepositAddressEPL,
		"getDepositAddress":                getDepositAddressEPL,
		"getWithdrawalAddress":             getWithdrawalAddressEPL,
		"userUniversalTransfer":            userUniversalTransferEPL,
		"getUniversalTransferDetailByID":   getUniversalTransferDetailByIDEPL,
		"getAssetConvertedMX":              getAssetConvertedMXEPL,
		"dustTransfer":                     dustTransferEPL,
		"dustLog":                          dustLogEPL,
		"internalTransfer":                 internalTransferEPL,
		"getInternalTransferHistory":       getInternalTransferHistoryEPL,
		"capitalWithdrawal":                capitalWithdrawalEPL,
		"contractsDetail":                  contractsDetailEPL,
		"getTransferableCurrencies":        getTransferableCurrenciesEPL,
		"getContractDepthInfo":             getContractDepthInfoEPL,
		"getDepthSnapshotOfContract":       getDepthSnapshotOfContractEPL,
		"getContractIndexPrice":            getContractIndexPriceEPL,
		"getContractFairPrice":             getContractFairPriceEPL,
		"getContractFundingPrice":          getContractFundingPriceEPL,
		"getContractsCandlestick":          getContractsCandlestickEPL,
		"getContractTransaction":           getContractTransactionEPL,
		"getContractTickers":               getContractTickersEPL,
		"getAllContrRiskFundBalance":       getAllContrRiskFundBalanceEPL,
		"contractRiskFundBalance":          contractRiskFundBalanceEPL,
		"contractFundingRateHistory":       contractFundingRateHistoryEPL,
		"allUserAssetsInfo":                allUserAssetsInfoEPL,
		"userSingleCurrencyAssetInfo":      userSingleCurrencyAssetInfoEPL,
		"userAssetTransferRecords":         userAssetTransferRecordsEPL,
		"userPositionHistory":              userPositionHistoryEPL,
		"usersCurrentHoldingPositions":     usersCurrentHoldingPositionsEPL,
		"usersFundingRateDetails":          usersFundingRateDetailsEPL,
		"userCurrentPendingOrder":          userCurrentPendingOrderEPL,
		"allUserHistoricalOrders":          allUserHistoricalOrdersEPL,
		"getOrderBasedOnExternalNumber":    getOrderBasedOnExternalNumberEPL,
		"orderByOrderNumber":               orderByOrderNumberEPL,
		"batchOrdersByOrderID":             batchOrdersByOrderIDEPL,
		"orderTransactionDetailsByOrderID": orderTransactionDetailsByOrderIDEPL,
		"userOrderAllTransactionDetails":   userOrderAllTransactionDetailsEPL,
		"triggerOrderList":                 triggerOrderListEPL,
		"futuresStopLimitOrderList":        futuresStopLimitOrderListEPL,
		"futuresRiskLimit":                 futuresRiskLimitEPL,
		"futuresCurrentTradingFeeRate":     futuresCurrentTradingFeeRateEPL,
		"increaseDecreaseMargin":           increaseDecreaseMarginEPL,
		"contractLeverage":                 contractLeverageEPL,
		"switchLeverage":                   switchLeverageEPL,
		"getPositionMode":                  getPositionModeEPL,
		"changePositionMode":               changePositionModeEPL,
		"placeFuturesOrder":                placeFuturesOrderEPL,
		"cancelAllTriggerOrders":           cancelAllTriggerOrdersEPL,
		"cancelSLTriggerOrder":             cancelSLTriggerOrderEPL,
		"cancelALlSLPriceTriggerOrders":    cancelALlSLPriceTriggerOrdersEPL,
		"switchSLPriceOfTriggerPrice":      switchSLPriceOfTriggerPriceEPL,
		"switchSLLimitedOrderPrice":        switchSLLimitedOrderPriceEPL,
		"getUniversalTransferhistory":      getUniversalTransferhistoryEPL,
		"getUserRebateHistory":             getUserRebateHistoryEPL,
		"getRebateRecordsDetail":           getRebateRecordsDetailEPL,
		"selfRebateRecordsDetails":         selfRebateRecordsDetailsEPL,
		"getReferCode":                     getReferCodeEPL,
		"getAffilateCommissionRecord":      getAffilateCommissionRecordEPL,
		"getAffilateWithdrawRecord":        getAffilateWithdrawRecordEPL,
		"getAffiliateConnissionDetail":     getAffiliateConnissionDetailEPL,
		"affiliateCampaignData":            affiliateCampaignDataEPL,
		"affiliateReferralData":            affiliateReferralDataEPL,
		"subAffiliateData":                 subAffiliateDataEPL,
	}
	rl, err := request.New("rateLimitTest2", http.DefaultClient, request.WithLimiter(GetRateLimit()))
	require.NoError(t, err)
	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := rl.InitiateRateLimit(context.Background(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
