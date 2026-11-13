package mexc

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	tenSecondsInterval  = time.Second * 10
	fiveSecondsInterval = time.Second * 5
	twoSecondsInterval  = time.Second * 2
	oneSecondInterval   = time.Second
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
	getSubAccountUniversalTransfersEPL
	getSubAccountAssetEPL
	getKYCStatusEPL
	selfSymbolsEPL
	newOrderEPL
	createBatchOrdersEPL
	cancelTradeOrderEPL
	cancelAllOpenOrdersBySymbolEPL
	cancelAllOrdersEPL
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
	dustConvertEPL
	dustLogEPL
	internalTransferEPL
	getInternalTransferHistoryEPL

	getUniversalTransferHistoryEPL
	getUserRebateHistoryEPL
	getRebateRecordsDetailEPL
	selfRebateRecordsDetailsEPL
	getReferCodeEPL
	getAffiliateCommissionRecordEPL
	getAffiliateWithdrawRecordEPL
	getAffiliateCommissionDetailEPL
	affiliateCampaignDataEPL
	affiliateReferralDataEPL
	subAffiliateDataEPL
	getUIDEPL
	getAPIKeyInfoEPL
	setAPIKeyInfoEPL
	offlineSymbolsEPL
	announcementsEPL
	createSelfTradePreventionGroupEPL
	getSelfTradePreventionGroupEPL
	deleteSelfTradePreventionGroupEPL
	addSelfTradePreventionGroupUIDsEPL
	deleteSelfTradePreventionGroupUIDsEPL
	listenKeyEPL
	brokerEPL

	futuresServerTimeEPL
	futuresContractsEPL
	futuresTransferableCurrenciesEPL
	futuresOrderbookEPL
	futuresDepthSnapshotsEPL
	futuresIndexPriceEPL
	futuresFairPriceEPL
	futuresFundingRateEPL
	futuresCandlesEPL
	futuresIndexPriceCandlesEPL
	futuresFairPriceCandlesEPL
	futuresRecentTradesEPL
	futuresTickersEPL
	futuresInsuranceFundEPL
	futuresInsuranceFundHistoryEPL
	futuresFundingRateHistoryEPL

	futuresAccountAssetsEPL
	futuresAccountAssetEPL
	futuresTransferRecordsEPL
	futuresProfitRateEPL
	futuresFeeDeductionConfigsEPL
	futuresDiscountConfigsEPL
	futuresFeeDeductionDetailsEPL
	futuresDiscountTypeEPL
	futuresFeeStatisticsEPL
	futuresHistoricalPositionsEPL
	futuresOpenPositionsEPL
	futuresFundingFeesEPL
	futuresRiskLimitsEPL
	futuresFeeDetailsEPL
	futuresChangeMarginEPL
	futuresAutoAddMarginEPL
	futuresLeverageEPL
	futuresChangeLeverageEPL
	futuresPositionModeEPL
	futuresChangePositionModeEPL
	futuresMultiAssetModeEPL
	futuresSelfTradePreventionGroupsEPL
	futuresSelfTradePreventionGroupEPL
	futuresCreateSelfTradePreventionGroupEPL
	futuresUpdateSelfTradePreventionGroupEPL
	futuresDeleteSelfTradePreventionGroupEPL

	futuresPlaceOrderEPL
	futuresBatchPlaceOrdersEPL
	futuresChaseOrderEPL
	futuresModifyOrderEPL
	futuresCancelOrdersEPL
	futuresBatchCancelByExternalIDEPL
	futuresCancelByExternalIDEPL
	futuresCancelAllOrdersEPL
	futuresReversePositionEPL
	futuresCloseAllPositionsEPL
	futuresOpenOrderCountsEPL
	futuresOrderByExternalIDEPL
	futuresOrderEPL
	futuresOrdersByIDEPL
	futuresOrdersByExternalIDEPL
	futuresOpenOrdersEPL
	futuresOrderHistoryEPL
	futuresOrderDealsEPL
	futuresOrderDealDetailsEPL
	futuresClosedOrdersEPL
	futuresPlanOrdersEPL
	futuresPlacePlanOrderEPL
	futuresModifyPlanOrderEPL
	futuresCancelPlanOrdersEPL
	futuresCancelAllPlanOrdersEPL
	futuresPlaceTakeProfitStopLossEPL
	futuresCancelTakeProfitStopLossEPL
	futuresCancelAllTakeProfitStopLossEPL
	futuresModifyLimitOrderTakeProfitStopLossEPL
	futuresModifyTakeProfitStopLossEPL
	futuresModifyPlanOrderTakeProfitStopLossEPL
	futuresTakeProfitStopLossOrdersEPL
	futuresOpenTakeProfitStopLossOrdersEPL
	futuresPlaceTrailingOrderEPL
	futuresCancelTrailingOrderEPL
	futuresModifyTrailingOrderEPL
	futuresTrailingOrdersEPL
)

// rateLimits holds the endpoint limits. The venue's budgets are per IP address and per account, not per
// Exchange instance, so every instance shares these limiters instead of building its own.
var rateLimits = func() request.RateLimitDefinitions {
	// IP-weighted endpoints share 300 weight per 10 seconds; the order-placement and cancel
	// endpoints are documented as a shared 12-requests-per-second budget, which is the binding
	// constraint on them (12/s is well inside the UID pool they also sit in), so they draw from one
	// per-second limiter rather than a weighted pool.
	ipModeRate := request.NewRateLimit(tenSecondsInterval, 300)
	orderRate := request.NewRateLimit(oneSecondInterval, 12)
	// Announcements are limited on their own at 5 requests per 2 seconds, outside the weighted pool.
	announcementsRate := request.NewRateLimit(twoSecondsInterval, 5)

	return request.RateLimitDefinitions{
		systemTimeEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		defaultSymbolsEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolsEPL:          request.GetRateLimiterWithWeight(ipModeRate, 25),
		orderbooksEPL:          request.GetRateLimiterWithWeight(ipModeRate, 3),
		recentTradesListEPL:    request.GetRateLimiterWithWeight(ipModeRate, 5),
		aggregatedTradesEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		candlestickEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		currentAveragePriceEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),

		// The 24hr ticker costs 25 whether one symbol or the whole catalogue is requested.
		symbolTickerPriceChangeStatEPL:  request.GetRateLimiterWithWeight(ipModeRate, 25),
		symbolsTickerPriceChangeStatEPL: request.GetRateLimiterWithWeight(ipModeRate, 25),

		symbolPriceTickerEPL:               request.GetRateLimiterWithWeight(ipModeRate, 10),
		symbolsPriceTickerEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),
		symbolOrderbookTickerEPL:           request.GetRateLimiterWithWeight(ipModeRate, 10),
		createSubAccountEPL:                request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountListEPL:                  request.GetRateLimiterWithWeight(ipModeRate, 1),
		createAPIKeyForSubAccountEPL:       request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAPIKeyEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),
		deleteSubAccountAPIKeyEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountUniversalTransferEPL:     request.GetRateLimiterWithWeight(ipModeRate, 50),
		getSubAccountUniversalTransfersEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAssetEPL:              request.GetRateLimiterWithWeight(ipModeRate, 1),
		getKYCStatusEPL:                    request.GetRateLimiterWithWeight(ipModeRate, 1),
		selfSymbolsEPL:                     request.GetRateLimiterWithWeight(ipModeRate, 1),
		newOrderEPL:                        request.GetRateLimiterWithWeight(orderRate, 1),
		createBatchOrdersEPL:               request.GetRateLimiterWithWeight(orderRate, 1),
		cancelTradeOrderEPL:                request.GetRateLimiterWithWeight(orderRate, 1),
		cancelAllOpenOrdersBySymbolEPL:     request.GetRateLimiterWithWeight(orderRate, 1),
		cancelAllOrdersEPL:                 request.GetRateLimiterWithWeight(orderRate, 1),
		getOrderByIDEPL:                    request.GetRateLimiterWithWeight(ipModeRate, 2),
		getOpenOrdersEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 3),
		allOrdersEPL:                       request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountInformationEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountTradeListEPL:                request.GetRateLimiterWithWeight(ipModeRate, 10),
		enableMXDeductEPL:                  request.GetRateLimiterWithWeight(ipModeRate, 1),
		getMXDeductStatusEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolTradingFeeEPL:             request.GetRateLimiterWithWeight(ipModeRate, 20),
		getCurrencyInformationEPL:          request.GetRateLimiterWithWeight(ipModeRate, 10),
		withdrawCapitalEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		cancelWithdrawalEPL:                request.GetRateLimiterWithWeight(ipModeRate, 1),
		getFundDepositHistoryEPL:           request.GetRateLimiterWithWeight(ipModeRate, 10),
		getWithdrawalHistoryEPL:            request.GetRateLimiterWithWeight(ipModeRate, 1),
		generateDepositAddressEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		getDepositAddressEPL:               request.GetRateLimiterWithWeight(ipModeRate, 10),
		getWithdrawalAddressEPL:            request.GetRateLimiterWithWeight(ipModeRate, 10),
		userUniversalTransferEPL:           request.GetRateLimiterWithWeight(ipModeRate, 50),
		getUniversalTransferDetailByIDEPL:  request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAssetConvertedMXEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),
		dustConvertEPL:                     request.GetRateLimiterWithWeight(ipModeRate, 10),
		dustLogEPL:                         request.GetRateLimiterWithWeight(ipModeRate, 1),
		internalTransferEPL:                request.GetRateLimiterWithWeight(ipModeRate, 1),
		getInternalTransferHistoryEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),

		getUniversalTransferHistoryEPL:  request.GetRateLimiterWithWeight(ipModeRate, 1),
		getUserRebateHistoryEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		getRebateRecordsDetailEPL:       request.GetRateLimiterWithWeight(ipModeRate, 1),
		selfRebateRecordsDetailsEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		getReferCodeEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffiliateCommissionRecordEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffiliateWithdrawRecordEPL:   request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffiliateCommissionDetailEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		affiliateCampaignDataEPL:        request.GetRateLimiterWithWeight(ipModeRate, 1),
		affiliateReferralDataEPL:        request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAffiliateDataEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),

		getUIDEPL:                             request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAPIKeyInfoEPL:                      request.GetRateLimiterWithWeight(ipModeRate, 1),
		setAPIKeyInfoEPL:                      request.GetRateLimiterWithWeight(ipModeRate, 1),
		offlineSymbolsEPL:                     request.GetRateLimiterWithWeight(ipModeRate, 10),
		announcementsEPL:                      request.GetRateLimiterWithWeight(announcementsRate, 1),
		createSelfTradePreventionGroupEPL:     request.GetRateLimiterWithWeight(ipModeRate, 20),
		getSelfTradePreventionGroupEPL:        request.GetRateLimiterWithWeight(ipModeRate, 20),
		deleteSelfTradePreventionGroupEPL:     request.GetRateLimiterWithWeight(ipModeRate, 20),
		addSelfTradePreventionGroupUIDsEPL:    request.GetRateLimiterWithWeight(ipModeRate, 20),
		deleteSelfTradePreventionGroupUIDsEPL: request.GetRateLimiterWithWeight(ipModeRate, 20),

		// The user data stream endpoints share one limit; their weight is undocumented, so they are
		// charged at 1 on the IP pool.
		listenKeyEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),

		// The broker endpoints share one limit; their weight is undocumented, so they are charged at 1
		// on the IP pool.
		brokerEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),

		// The futures API documents a budget for each endpoint on its own, rather than a shared weighted pool
		futuresServerTimeEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresContractsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		futuresTransferableCurrenciesEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderbookEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		futuresDepthSnapshotsEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresIndexPriceEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFairPriceEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFundingRateEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCandlesEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresIndexPriceCandlesEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFairPriceCandlesEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresRecentTradesEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresTickersEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		futuresInsuranceFundEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresInsuranceFundHistoryEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFundingRateHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		futuresAccountAssetsEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresAccountAssetEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresTransferRecordsEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresProfitRateEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFeeDeductionConfigsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresDiscountConfigsEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFeeDeductionDetailsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresDiscountTypeEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFeeStatisticsEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresHistoricalPositionsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOpenPositionsEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFundingFeesEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresRiskLimitsEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresFeeDetailsEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresChangeMarginEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresAutoAddMarginEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresLeverageEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresChangeLeverageEPL:                 request.NewRateLimitWithWeight(tenSecondsInterval, 10, 1),
		futuresPositionModeEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresChangePositionModeEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresMultiAssetModeEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresSelfTradePreventionGroupsEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresSelfTradePreventionGroupEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCreateSelfTradePreventionGroupEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresUpdateSelfTradePreventionGroupEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresDeleteSelfTradePreventionGroupEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		futuresPlaceOrderEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresBatchPlaceOrdersEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresChaseOrderEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresModifyOrderEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresCancelOrdersEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresBatchCancelByExternalIDEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCancelByExternalIDEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCancelAllOrdersEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresReversePositionEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresCloseAllPositionsEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresOpenOrderCountsEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderByExternalIDEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderEPL:                              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrdersByIDEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		futuresOrdersByExternalIDEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOpenOrdersEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderHistoryEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderDealsEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOrderDealDetailsEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresClosedOrdersEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresPlanOrdersEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresPlacePlanOrderEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresModifyPlanOrderEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresCancelPlanOrdersEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCancelAllPlanOrdersEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresPlaceTakeProfitStopLossEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		futuresCancelTakeProfitStopLossEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresCancelAllTakeProfitStopLossEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresModifyLimitOrderTakeProfitStopLossEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresModifyTakeProfitStopLossEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresModifyPlanOrderTakeProfitStopLossEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresTakeProfitStopLossOrdersEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresOpenTakeProfitStopLossOrdersEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresPlaceTrailingOrderEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresCancelTrailingOrderEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		futuresModifyTrailingOrderEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		futuresTrailingOrdersEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
	}
}()
