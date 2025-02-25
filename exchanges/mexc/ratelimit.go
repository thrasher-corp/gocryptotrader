package mexc

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	tenSecondsInterval  = time.Second * 10
	fiveSecondsInterval = time.Second * 5
	twoSecondsInterval  = time.Second * 2
)

const (
	systemTimeEPL request.EndpointLimit = iota
	defaultSymbolsEPL
	getSymbolsEPL
	orderbooksEPL
	recentTradesListEPL
	aggregatedTradesEPL
	candlestickEPL
	currentAveragePriceEPL
	symbolTickerPriceChangeStatEPL
	symbolsTickerPriceChangeStatEPL
	symbolPriceTickerEPL
	symbolsPriceTickerEPL
	symbolOrderbookTickerEPL
	createSubAccountEPL
	subAccountListEPL
	createAPIKeyForSubAccountEPL
	getSubAccountAPIKeyEPL
	deleteSubAccountAPIKeyEPL
	subAccountUniversalTransferEPL
	getSubaccUnversalTransfersEPL
	getSubAccountAssetEPL
	getKYCStatusEPL
	selfSymbolsEPL
	newOrderEPL
	createBatchOrdersEPL
	cancelTradeOrderEPL
	cancelAllOpenOrdersBySymbolEPL
	getOrderByIDEPL
	getOpenOrdersEPL
	allOrdersEPL
	accountInformationEPL
	accountTradeListEPL
	enableMXDeductEPL
	getMXDeductStatusEPL
	getSymbolTradingFeeEPL
	getCurrencyInformationEPL
	withdrawCapitalEPL
	cancelWithdrawalEPL
	getFundDepositHistoryEPL
	getWithdrawalHistoryEPL
	generateDepositAddressEPL
	getDepositAddressEPL
	getWithdrawalAddressEPL
	userUniversalTransferEPL
	getUniversalTransferDetailByIDEPL
	getAssetConvertedMXEPL
	dustTransferEPL
	dustLogEPL
	internalTransferEPL
	getInternalTransferHistoryEPL
	capitalWithdrawalEPL
	contractsDetailEPL
	getTransferableCurrenciesEPL
	getContractDepthInfoEPL
	getDepthSnapshotOfContractEPL
	getContractIndexPriceEPL
	getContractFairPriceEPL
	getContractFundingPriceEPL
	getContractsCandlestickEPL
	getContractTransactionEPL
	getContractTickersEPL
	getAllContrRiskFundBalanceEPL
	contractRiskFundBalanceEPL
	contractFundingRateHistoryEPL
	allUserAssetsInfoEPL
	userSingleCurrencyAssetInfoEPL
	userAssetTransferRecordsEPL
	userPositionHistoryEPL
	usersCurrentHoldingPositionsEPL
	usersFundingRateDetailsEPL
	userCurrentPendingOrderEPL
	allUserHistoricalOrdersEPL
	getOrderBasedOnExternalNumberEPL
	orderByOrderNumberEPL
	batchOrdersByOrderIDEPL
	orderTransactionDetailsByOrderIDEPL
	userOrderAllTransactionDetailsEPL
	triggerOrderListEPL
	futuresStopLimitOrderListEPL
	futuresRiskLimitEPL
	futuresCurrentTradingFeeRateEPL
	increaseDecreaseMarginEPL
	contractLeverageEPL
	switchLeverageEPL
	getPositionModeEPL
	changePositionModeEPL
	placeFuturesOrderEPL
)

// GetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	ipModeRate := request.NewRateLimit(tenSecondsInterval, 500)
	uidModeRate := request.NewRateLimit(tenSecondsInterval, 500)

	return request.RateLimitDefinitions{
		systemTimeEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		defaultSymbolsEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolsEPL:          request.GetRateLimiterWithWeight(ipModeRate, 10),
		orderbooksEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		recentTradesListEPL:    request.GetRateLimiterWithWeight(ipModeRate, 5),
		aggregatedTradesEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		candlestickEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		currentAveragePriceEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),

		symbolTickerPriceChangeStatEPL:  request.GetRateLimiterWithWeight(ipModeRate, 1),
		symbolsTickerPriceChangeStatEPL: request.GetRateLimiterWithWeight(ipModeRate, 40),

		symbolPriceTickerEPL:              request.GetRateLimiterWithWeight(ipModeRate, 1),
		symbolsPriceTickerEPL:             request.GetRateLimiterWithWeight(ipModeRate, 2),
		symbolOrderbookTickerEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		createSubAccountEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountListEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		createAPIKeyForSubAccountEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAPIKeyEPL:            request.GetRateLimiterWithWeight(ipModeRate, 1),
		deleteSubAccountAPIKeyEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountUniversalTransferEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubaccUnversalTransfersEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAssetEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),
		getKYCStatusEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 1),
		selfSymbolsEPL:                    request.GetRateLimiterWithWeight(ipModeRate, 1),
		newOrderEPL:                       request.GetRateLimiterWithWeight(uidModeRate, 1), //
		createBatchOrdersEPL:              request.GetRateLimiterWithWeight(uidModeRate, 1), //
		cancelTradeOrderEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		cancelAllOpenOrdersBySymbolEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		getOrderByIDEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 2),
		getOpenOrdersEPL:                  request.GetRateLimiterWithWeight(ipModeRate, 3),
		allOrdersEPL:                      request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountInformationEPL:             request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountTradeListEPL:               request.GetRateLimiterWithWeight(ipModeRate, 10),
		enableMXDeductEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		getMXDeductStatusEPL:              request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolTradingFeeEPL:            request.GetRateLimiterWithWeight(ipModeRate, 20),
		getCurrencyInformationEPL:         request.GetRateLimiterWithWeight(ipModeRate, 10),
		withdrawCapitalEPL:                request.GetRateLimiterWithWeight(ipModeRate, 1),
		cancelWithdrawalEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		getFundDepositHistoryEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		getWithdrawalHistoryEPL:           request.GetRateLimiterWithWeight(ipModeRate, 1),
		generateDepositAddressEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		getDepositAddressEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),
		getWithdrawalAddressEPL:           request.GetRateLimiterWithWeight(ipModeRate, 10),
		userUniversalTransferEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		getUniversalTransferDetailByIDEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAssetConvertedMXEPL:            request.GetRateLimiterWithWeight(ipModeRate, 1),
		dustTransferEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 10),
		dustLogEPL:                        request.GetRateLimiterWithWeight(ipModeRate, 1),
		internalTransferEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		getInternalTransferHistoryEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		capitalWithdrawalEPL:              request.GetRateLimiterWithWeight(ipModeRate, 1),

		contractsDetailEPL:            request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		getTransferableCurrenciesEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractDepthInfoEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDepthSnapshotOfContractEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractIndexPriceEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractFairPriceEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractFundingPriceEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractsCandlestickEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractTransactionEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractTickersEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAllContrRiskFundBalanceEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		contractRiskFundBalanceEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		contractFundingRateHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		allUserAssetsInfoEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		userSingleCurrencyAssetInfoEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		userAssetTransferRecordsEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		userPositionHistoryEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		usersCurrentHoldingPositionsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		usersFundingRateDetailsEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		userCurrentPendingOrderEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		allUserHistoricalOrdersEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOrderBasedOnExternalNumberEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		orderByOrderNumberEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		batchOrdersByOrderIDEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		orderTransactionDetailsByOrderIDEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		userOrderAllTransactionDetailsEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		triggerOrderListEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresStopLimitOrderListEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresRiskLimitEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCurrentTradingFeeRateEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		increaseDecreaseMarginEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		contractLeverageEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		switchLeverageEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getPositionModeEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		changePositionModeEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		placeFuturesOrderEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
	}
}
