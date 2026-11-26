package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Rate limit of each endpoint. BingX budgets every signed endpoint per account (UID) at the rate its documentation
// gives, and its IP group 1, the market data endpoints and the signed deposit address endpoint, shares 500 requests
// each 10 seconds per IP. BingX removed the IP limits of the account groups 2 and 3 on 2025-12-16. The deposit address
// endpoint's own limit, one request a second or 10 each 10 seconds, is reserved out of group 1's budget, and the market
// data endpoints share the other 490
const (
	systemTimeRate request.EndpointLimit = iota
	swapContractsRate
	swapOrderBookRate
	swapRecentTradesRate
	swapMarkPriceAndFundingRateRate
	swapFundingRateHistoryRate
	swapKlinesRate
	swapOpenInterestRate
	swapTickerRate
	swapHistoricalTradesRate
	swapBookTickerRate
	swapMarkPriceKlinesRate
	swapPriceTickerRate
	swapTradingRulesRate
	swapTradingScheduleRate
	swapTestOrderRate
	swapPlaceOrderRate
	swapPlaceBatchOrdersRate
	swapModifyOrderRate
	swapCloseAllPositionsRate
	swapCancelOrderRate
	swapCancelBatchOrdersRate
	swapCancelAllOpenOrdersRate
	swapOpenOrdersRate
	swapPendingOrderStatusRate
	swapOrderDetailsRate
	swapMarginTypeRate
	swapChangeMarginTypeRate
	swapLeverageRate
	swapSetLeverageRate
	swapForceOrdersRate
	swapOrderHistoryRate
	swapModifyIsolatedPositionMarginRate
	swapFillOrdersRate
	swapSetPositionModeRate
	swapPositionModeRate
	swapCancelReplaceOrderRate
	swapBatchCancelReplaceOrdersRate
	swapCancelAllAfterRate
	swapClosePositionRate
	swapAllOrdersRate
	swapMaintenanceMarginRatioRate
	swapFillHistoryRate
	swapPositionHistoryRate
	swapIsolatedMarginHistoryRate
	swapApplyVSTRate
	swapPlaceTWAPOrderRate
	swapTWAPOpenOrdersRate
	swapTWAPHistoryOrdersRate
	swapTWAPOrderDetailRate
	swapCancelTWAPOrderRate
	swapSwitchMultiAssetsModeRate
	swapMultiAssetsModeRate
	swapMultiAssetsRulesRate
	swapMultiAssetsMarginRate
	swapReversePositionRate
	swapAutoAddMarginRate
	swapBalanceRate
	swapPositionsRate
	swapIncomeRate
	swapExportIncomeRate
	swapCommissionRateRate
	spotSymbolsRate
	spotRecentTradesRate
	spotOrderBookRate
	spotKlinesRate
	spotTickerRate
	spotAggregatedOrderBookRate
	spotPriceTickerRate
	spotBookTickerRate
	spotHistoricalKlinesRate
	spotHistoricalTradesRate
	spotBalanceRate
	assetTransferRecordsRate
	mainAccountInternalTransferRate
	assetTransferRate
	transferableCurrenciesRate
	assetTransferRecordsNewRate
	fundAccountBalanceRate
	mainAccountInternalTransferRecordsRate
	assetOverviewRate
	depositRecordsRate
	withdrawRecordsRate
	currencyConfigRate
	withdrawRate
	depositAddressRate
	depositRiskRecordsRate
	spotPlaceOrderRate
	spotPlaceBatchOrdersRate
	spotCancelOrderRate
	spotCancelBatchOrdersRate
	spotCancelOpenOrdersRate
	spotCancelReplaceOrderRate
	spotOrderDetailsRate
	spotOpenOrdersRate
	spotOrderHistoryRate
	spotTradeDetailsRate
	spotCommissionRateRate
	spotCancelAllAfterRate
	spotPlaceOCOOrderRate
	spotCancelOCOOrderRate
	spotOCOOrderListRate
	spotOpenOCOOrdersRate
	spotOCOOrderHistoryRate
	coinMarginedContractsRate
	coinMarginedPremiumIndexRate
	coinMarginedOpenInterestRate
	coinMarginedKlinesRate
	coinMarginedOrderBookRate
	coinMarginedTickerRate
	coinMarginedPlaceOrderRate
	coinMarginedCommissionRateRate
	coinMarginedLeverageRate
	coinMarginedSetLeverageRate
	coinMarginedCancelAllOpenOrdersRate
	coinMarginedCloseAllPositionsRate
	coinMarginedPositionsRate
	coinMarginedBalanceRate
	coinMarginedForceOrdersRate
	coinMarginedFillOrdersRate
	coinMarginedCancelOrderRate
	coinMarginedOpenOrdersRate
	coinMarginedOrderDetailRate
	coinMarginedOrderHistoryRate
	coinMarginedMarginTypeRate
	coinMarginedSetMarginTypeRate
	coinMarginedAdjustIsolatedMarginRate
	createSubAccountRate
	apiKeyPermissionsRate
	accountUIDRate
	subAccountListRate
	subAccountAssetsRate
	createSubAccountAPIKeyRate
	apiKeyInformationRate
	editSubAccountAPIKeyRate
	deleteSubAccountAPIKeyRate
	updateSubAccountStatusRate
	authoriseSubAccountInternalTransferRate
	subAccountInternalTransferRate
	querySubAccountDepositAddressRate
	subAccountDepositAddressRate
	subAccountDepositRecordsRate
	subAccountInternalTransferRecordsRate
	subAccountTransferHistoryRate
	subAccountTransferableAmountRate
	subAccountAssetTransferRate
	subAccountAssetOverviewRate
	agentInvitedUsersRate
	agentDailyCommissionRate
	agentUserInformationRate
	agentInvitedUserDepositsRate
	agentAPICommissionRate
	agentPartnerInformationRate
	agentReferralCodeCommissionRate
	agentSuperiorCheckRate
	agentChannelCommissionRate
	copyTradingSetTakeProfitStopLossRate
	copyTradingPerpetualTraderDetailRate
	copyTradingPerpetualProfitSummaryRate
	copyTradingPerpetualProfitDetailRate
	copyTradingSetCommissionRate
	copyTradingPerpetualTradingPairsRate
	copyTradingSpotSellOrderRate
	copyTradingSpotTraderDetailRate
	copyTradingSpotProfitSummaryRate
	copyTradingSpotProfitDetailRate
	copyTradingSpotOrderHistoryRate
	dualCurrencyProductsRate
	dualCurrencyPreOrderQuoteRate
	dualCurrencyPlaceOrderRate
	dualCurrencyPositionsRate
	dualCurrencyOrderRecordsRate
	createSignalRate
	activityListRate
	publicActivityListRate
	generateListenKeyRate
	extendListenKeyRate
	closeListenKeyRate

	// Websocket budgets: subscription and unsubscription requests share BingX's 10 a second per IP across every
	// connection, and heartbeat replies have a budget of their own, so they never wait behind subscription requests
	websocketSubscriptionRate
	websocketHeartbeatRate
)

// GetRateLimits returns the rate limit definitions for BingX
func GetRateLimits() request.RateLimitDefinitions {
	marketData := request.NewRateLimitWithWeight(10*time.Second, 490, 1)
	// undocumented is the conservative limit of an endpoint BingX documents no rate for
	undocumented := func() *request.RateLimiterWithWeight { return request.NewRateLimitWithWeight(time.Second, 5, 1) }
	return request.RateLimitDefinitions{
		systemTimeRate:                          marketData,
		swapContractsRate:                       marketData,
		swapOrderBookRate:                       marketData,
		swapRecentTradesRate:                    marketData,
		swapMarkPriceAndFundingRateRate:         marketData,
		swapFundingRateHistoryRate:              marketData,
		swapKlinesRate:                          marketData,
		swapOpenInterestRate:                    marketData,
		swapTickerRate:                          marketData,
		swapHistoricalTradesRate:                marketData,
		swapBookTickerRate:                      marketData,
		swapMarkPriceKlinesRate:                 marketData,
		swapPriceTickerRate:                     marketData,
		swapTradingRulesRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapTradingScheduleRate:                 request.NewRateLimitWithWeight(10*time.Second, 20, 1),
		swapTestOrderRate:                       request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapPlaceOrderRate:                      request.NewRateLimitWithWeight(time.Second, 10, 1),
		swapPlaceBatchOrdersRate:                request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapModifyOrderRate:                     request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapCloseAllPositionsRate:               request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapCancelOrderRate:                     request.NewRateLimitWithWeight(time.Second, 10, 1),
		swapCancelBatchOrdersRate:               request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapCancelAllOpenOrdersRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapOpenOrdersRate:                      request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapPendingOrderStatusRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapOrderDetailsRate:                    request.NewRateLimitWithWeight(time.Second, 30, 1),
		swapMarginTypeRate:                      request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapChangeMarginTypeRate:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapLeverageRate:                        request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapSetLeverageRate:                     request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapForceOrdersRate:                     request.NewRateLimitWithWeight(time.Second, 10, 1),
		swapOrderHistoryRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapModifyIsolatedPositionMarginRate:    request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapFillOrdersRate:                      request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapSetPositionModeRate:                 request.NewRateLimitWithWeight(time.Second, 4, 1),
		swapPositionModeRate:                    request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapCancelReplaceOrderRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapBatchCancelReplaceOrdersRate:        request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapCancelAllAfterRate:                  request.NewRateLimitWithWeight(time.Second, 1, 1),
		swapClosePositionRate:                   request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapAllOrdersRate:                       request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapMaintenanceMarginRatioRate:          request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapFillHistoryRate:                     request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapPositionHistoryRate:                 request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapIsolatedMarginHistoryRate:           request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapApplyVSTRate:                        request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapPlaceTWAPOrderRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapTWAPOpenOrdersRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapTWAPHistoryOrdersRate:               request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapTWAPOrderDetailRate:                 request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapCancelTWAPOrderRate:                 request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapSwitchMultiAssetsModeRate:           request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapMultiAssetsModeRate:                 request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapMultiAssetsRulesRate:                undocumented(),
		swapMultiAssetsMarginRate:               request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapReversePositionRate:                 request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapAutoAddMarginRate:                   request.NewRateLimitWithWeight(time.Second, 2, 1),
		swapBalanceRate:                         request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapPositionsRate:                       request.NewRateLimitWithWeight(time.Second, 10, 1),
		swapIncomeRate:                          request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapExportIncomeRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		swapCommissionRateRate:                  request.NewRateLimitWithWeight(time.Second, 5, 1),
		spotSymbolsRate:                         marketData,
		spotRecentTradesRate:                    marketData,
		spotOrderBookRate:                       marketData,
		spotKlinesRate:                          marketData,
		spotTickerRate:                          marketData,
		spotAggregatedOrderBookRate:             marketData,
		spotPriceTickerRate:                     marketData,
		spotBookTickerRate:                      marketData,
		spotHistoricalKlinesRate:                marketData,
		spotHistoricalTradesRate:                marketData,
		spotBalanceRate:                         request.NewRateLimitWithWeight(time.Second, 5, 1),
		assetTransferRecordsRate:                request.NewRateLimitWithWeight(time.Second, 3, 1),
		mainAccountInternalTransferRate:         request.NewRateLimitWithWeight(time.Second, 2, 1),
		assetTransferRate:                       request.NewRateLimitWithWeight(time.Second, 2, 1),
		transferableCurrenciesRate:              request.NewRateLimitWithWeight(time.Second, 2, 1),
		assetTransferRecordsNewRate:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		fundAccountBalanceRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		mainAccountInternalTransferRecordsRate:  request.NewRateLimitWithWeight(time.Second, 10, 1),
		assetOverviewRate:                       request.NewRateLimitWithWeight(time.Second, 5, 1),
		depositRecordsRate:                      request.NewRateLimitWithWeight(time.Second, 10, 1),
		withdrawRecordsRate:                     request.NewRateLimitWithWeight(time.Second, 10, 1),
		currencyConfigRate:                      request.NewRateLimitWithWeight(time.Second, 5, 1),
		withdrawRate:                            request.NewRateLimitWithWeight(time.Second, 2, 1),
		depositAddressRate:                      request.NewRateLimitWithWeight(time.Second, 1, 1),
		depositRiskRecordsRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotPlaceOrderRate:                      request.NewRateLimitWithWeight(time.Second, 10, 1),
		spotPlaceBatchOrdersRate:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotCancelOrderRate:                     request.NewRateLimitWithWeight(time.Second, 5, 1),
		spotCancelBatchOrdersRate:               request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotCancelOpenOrdersRate:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotCancelReplaceOrderRate:              request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotOrderDetailsRate:                    request.NewRateLimitWithWeight(time.Second, 10, 1),
		spotOpenOrdersRate:                      request.NewRateLimitWithWeight(time.Second, 10, 1),
		spotOrderHistoryRate:                    request.NewRateLimitWithWeight(time.Second, 10, 1),
		spotTradeDetailsRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		spotCommissionRateRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotCancelAllAfterRate:                  request.NewRateLimitWithWeight(time.Second, 1, 1),
		spotPlaceOCOOrderRate:                   request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotCancelOCOOrderRate:                  request.NewRateLimitWithWeight(time.Second, 2, 1),
		spotOCOOrderListRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		spotOpenOCOOrdersRate:                   request.NewRateLimitWithWeight(time.Second, 5, 1),
		spotOCOOrderHistoryRate:                 request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedContractsRate:               marketData,
		coinMarginedPremiumIndexRate:            marketData,
		coinMarginedOpenInterestRate:            marketData,
		coinMarginedKlinesRate:                  marketData,
		coinMarginedOrderBookRate:               marketData,
		coinMarginedTickerRate:                  marketData,
		coinMarginedPlaceOrderRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedCommissionRateRate:          request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedLeverageRate:                request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedSetLeverageRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedCancelAllOpenOrdersRate:     request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedCloseAllPositionsRate:       request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedPositionsRate:               request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedBalanceRate:                 request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedForceOrdersRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedFillOrdersRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedCancelOrderRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedOpenOrdersRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedOrderDetailRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedOrderHistoryRate:            request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedMarginTypeRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedSetMarginTypeRate:           request.NewRateLimitWithWeight(time.Second, 5, 1),
		coinMarginedAdjustIsolatedMarginRate:    request.NewRateLimitWithWeight(time.Second, 5, 1),
		createSubAccountRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		apiKeyPermissionsRate:                   request.NewRateLimitWithWeight(time.Second, 5, 1),
		accountUIDRate:                          request.NewRateLimitWithWeight(time.Second, 10, 1),
		subAccountListRate:                      request.NewRateLimitWithWeight(time.Second, 5, 1),
		subAccountAssetsRate:                    request.NewRateLimitWithWeight(time.Second, 5, 1),
		createSubAccountAPIKeyRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		apiKeyInformationRate:                   request.NewRateLimitWithWeight(time.Second, 5, 1),
		editSubAccountAPIKeyRate:                request.NewRateLimitWithWeight(time.Second, 5, 1),
		deleteSubAccountAPIKeyRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		updateSubAccountStatusRate:              request.NewRateLimitWithWeight(time.Second, 5, 1),
		authoriseSubAccountInternalTransferRate: request.NewRateLimitWithWeight(time.Second, 10, 1),
		subAccountInternalTransferRate:          request.NewRateLimitWithWeight(time.Second, 5, 1),
		querySubAccountDepositAddressRate:       request.NewRateLimitWithWeight(time.Second, 5, 1),
		subAccountDepositAddressRate:            request.NewRateLimitWithWeight(time.Second, 5, 1),
		subAccountDepositRecordsRate:            request.NewRateLimitWithWeight(time.Second, 5, 1),
		subAccountInternalTransferRecordsRate:   request.NewRateLimitWithWeight(time.Second, 10, 1),
		subAccountTransferHistoryRate:           undocumented(),
		subAccountTransferableAmountRate:        undocumented(),
		subAccountAssetTransferRate:             undocumented(),
		subAccountAssetOverviewRate:             request.NewRateLimitWithWeight(time.Second, 5, 1),
		agentInvitedUsersRate:                   request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentDailyCommissionRate:                request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentUserInformationRate:                request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentInvitedUserDepositsRate:            request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentAPICommissionRate:                  request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentPartnerInformationRate:             request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentReferralCodeCommissionRate:         request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentSuperiorCheckRate:                  request.NewRateLimitWithWeight(time.Second, 20, 1),
		agentChannelCommissionRate:              request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSetTakeProfitStopLossRate:    request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingPerpetualTraderDetailRate:    request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingPerpetualProfitSummaryRate:   request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingPerpetualProfitDetailRate:    request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSetCommissionRate:            request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingPerpetualTradingPairsRate:    request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSpotSellOrderRate:            request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSpotTraderDetailRate:         request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSpotProfitSummaryRate:        request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSpotProfitDetailRate:         request.NewRateLimitWithWeight(time.Second, 10, 1),
		copyTradingSpotOrderHistoryRate:         request.NewRateLimitWithWeight(time.Second, 10, 1),
		dualCurrencyProductsRate:                request.NewRateLimitWithWeight(time.Second, 10, 1),
		dualCurrencyPreOrderQuoteRate:           request.NewRateLimitWithWeight(time.Second, 10, 1),
		dualCurrencyPlaceOrderRate:              request.NewRateLimitWithWeight(time.Second, 10, 1),
		dualCurrencyPositionsRate:               request.NewRateLimitWithWeight(time.Second, 10, 1),
		dualCurrencyOrderRecordsRate:            request.NewRateLimitWithWeight(time.Second, 10, 1),
		createSignalRate:                        request.NewRateLimitWithWeight(time.Second, 1, 1),
		activityListRate:                        request.NewRateLimitWithWeight(10*time.Second, 100, 1),
		publicActivityListRate:                  request.NewRateLimitWithWeight(10*time.Second, 100, 1),
		generateListenKeyRate:                   request.NewRateLimitWithWeight(time.Second, 2, 1),
		extendListenKeyRate:                     request.NewRateLimitWithWeight(time.Second, 2, 1),
		closeListenKeyRate:                      request.NewRateLimitWithWeight(time.Second, 2, 1),
		websocketSubscriptionRate:               request.NewRateLimitWithWeight(time.Second, 10, 1),
		websocketHeartbeatRate:                  request.NewRateLimitWithWeight(time.Second, 10, 1),
	}
}
