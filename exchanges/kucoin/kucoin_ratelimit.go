package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	thirtySecondsInterval = time.Second * 30
)

const (
	accountSummaryInfoEPL request.EndpointLimit = iota
	allAccountEPL
	accountDetailEPL
	accountLedgersEPL
	hfAccountLedgersEPL
	hfAccountLedgersMarginEPL
	futuresAccountLedgersEPL
	subAccountInfoV1EPL
	allSubAccountsInfoV2EPL
	createSubUserEPL
	subAccountsEPL
	subAccountBalancesEPL
	allSubAccountBalancesV2EPL
	subAccountSpotAPIListEPL
	createSpotAPIForSubAccountEPL
	modifySubAccountSpotAPIEPL
	deleteSubAccountSpotAPIEPL
	marginAccountDetailEPL
	crossMarginAccountsDetailEPL
	isolatedMarginAccountDetailEPL
	futuresAccountsDetailEPL
	tradingPairActualFeeEPL
	allFuturesSubAccountBalancesEPL
	futuresTradingPairFeeEPL
	futuresPositionHistoryEPL
	futuresMaxOpenPositionsSizeEPL
	futuresAllTickersInfoEPL
	createDepositAddressEPL
	depositAddressesV2EPL
	depositAddressesV1EPL
	depositListEPL
	historicDepositListEPL
	withdrawalListEPL
	retrieveV1HistoricalWithdrawalListEPL
	withdrawalQuotaEPL
	applyWithdrawalEPL
	cancelWithdrawalsEPL
	getTransferablesEPL
	flexiTransferEPL
	masterSubUserTransferEPL
	innerTransferEPL
	toMainOrTradeAccountEPL
	toFuturesAccountEPL
	futuresTransferOutRequestRecordsEPL
	basicFeesEPL
	tradeFeesEPL
	spotCurrenciesV3EPL
	spotCurrencyDetailEPL
	symbolsEPL
	tickersEPL
	allTickersEPL
	statistics24HrEPL
	marketListEPL
	partOrderbook20EPL
	partOrderbook100EPL
	fullOrderbookEPL
	tradeHistoryEPL
	klinesEPL
	fiatPriceEPL
	currentServerTimeEPL
	serviceStatusEPL
	hfPlaceOrderEPL
	hfSyncPlaceOrderEPL
	hfMultipleOrdersEPL
	hfSyncPlaceMultipleHFOrdersEPL
	hfModifyOrderEPL
	cancelHFOrderEPL
	hfSyncCancelOrderEPL
	hfCancelOrderByClientOrderIDEPL
	cancelSpecifiedNumberHFOrdersByOrderIDEPL
	hfCancelAllOrdersBySymbolEPL
	hfCancelAllOrdersEPL
	hfGetAllActiveOrdersEPL
	hfSymbolsWithActiveOrdersEPL
	hfCompletedOrderListEPL
	hfOrderDetailByOrderIDEPL
	autoCancelHFOrderSettingEPL
	autoCancelHFOrderSettingQueryEPL
	hfFilledListEPL
	placeOrderEPL
	placeBulkOrdersEPL
	cancelOrderEPL
	cancelOrderByClientOrderIDEPL
	cancelAllOrdersEPL
	listOrdersEPL
	recentOrdersEPL
	orderDetailByIDEPL
	getOrderByClientSuppliedOrderIDEPL
	listFillsEPL
	getRecentFillsEPL
	placeStopOrderEPL
	cancelStopOrderEPL
	cancelStopOrderByClientIDEPL
	cancelStopOrdersEPL
	listStopOrdersEPL
	getStopOrderDetailEPL
	getStopOrderByClientIDEPL
	placeOCOOrderEPL
	cancelOCOOrderByIDEPL
	cancelMultipleOCOOrdersEPL
	getOCOOrderByIDEPL
	getOCOOrderDetailsByOrderIDEPL
	getOCOOrdersEPL
	placeMarginOrderEPL
	cancelMarginHFOrderByIDEPL
	getMarginHFOrderDetailByID
	cancelAllMarginHFOrdersBySymbolEPL
	getActiveMarginHFOrdersEPL
	getFilledHFMarginOrdersEPL
	getMarginHFOrderDetailByOrderIDEPL
	getMarginHFTradeFillsEPL
	placeMarginOrdersEPL
	leveragedTokenInfoEPL
	getMarkPriceEPL
	getAllMarginMarkPriceEPL
	getMarginConfigurationEPL
	crossIsolatedMarginRiskLimitCurrencyConfigEPL
	isolatedMarginPairConfigEPL
	isolatedMarginAccountInfoEPL
	singleIsolatedMarginAccountInfoEPL
	postMarginBorrowOrderEPL
	postMarginRepaymentEPL
	getCrossIsolatedMarginInterestRecordsEPL
	marginBorrowingHistoryEPL
	marginRepaymentHistoryEPL
	lendingCurrencyInfoEPL
	interestRateEPL
	marginLendingSubscriptionEPL
	redemptionEPL
	modifySubscriptionEPL
	getRedemptionOrdersEPL
	getSubscriptionOrdersEPL
	futuresOpenContractsEPL
	futuresContractEPL
	futuresTickerEPL
	futuresOrderbookEPL
	futuresPartOrderbookDepth20EPL
	futuresPartOrderbookDepth100EPL
	futuresTransactionHistoryEPL
	futuresKlineEPL
	futuresInterestRateEPL
	futuresIndexListEPL
	futuresCurrentMarkPriceEPL
	futuresPremiumIndexEPL
	futuresTransactionVolumeEPL
	futuresServerTimeEPL
	futuresServiceStatusEPL
	multipleFuturesOrdersEPL
	futuresCancelAnOrderEPL
	futuresPlaceOrderEPL
	futuresLimitOrderMassCancelationEPL
	cancelUntriggeredFuturesStopOrdersEPL
	futuresCancelMultipleLimitOrdersEPL
	futuresRetrieveOrderListEPL
	futuresRecentCompletedOrdersEPL
	futuresOrdersByIDEPL
	futuresRetrieveFillsEPL
	futuresRecentFillsEPL
	futuresOpenOrderStatsEPL
	futuresPositionEPL
	futuresPositionListEPL
	setAutoDepositMarginEPL
	maxWithdrawMarginEPL
	removeMarginManuallyEPL
	futuresAddMarginManuallyEPL
	futuresRiskLimitLevelEPL
	futuresUpdateRiskLimitLevelEPL
	futuresCurrentFundingRateEPL
	futuresPublicFundingRateEPL
	futuresFundingHistoryEPL
	spotAuthenticationEPL
	futuresAuthenticationEPL
	futuresOrderDetailsByClientOrderIDEPL
	modifySubAccountAPIEPL
	allSubAccountsBalanceEPL
	allUserSubAccountsV2EPL
	futuresRetrieveTransactionHistoryEPL
	futuresAccountOverviewEPL
	createSubAccountAPIKeyEPL
	transferOutToMainEPL
	transferFundToFuturesAccountEPL
	futuresTransferOutListEPL

	subscribeToEarnEPL
	earnRedemptionEPL
	earnRedemptionPreviewEPL

	kucoinEarnSavingsProductsEPL
	kucoinEarnFixedIncomeCurrentHoldingEPL
	earnLimitedTimePromotionProductEPL
	earnKCSStakingProductEPL
	earnStakingProductEPL

	vipLendingEPL
	affilateUserRebateInfoEPL
	marginPairsConfigurationEPL
	modifyLeverageMultiplierEPL
	marginActiveHFOrdersEPL
)

// GetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	spotRate := request.NewRateLimit(thirtySecondsInterval, 4000)
	futuresRate := request.NewRateLimit(thirtySecondsInterval, 2000)
	managementRate := request.NewRateLimit(thirtySecondsInterval, 2000)
	publicRate := request.NewRateLimit(thirtySecondsInterval, 2000)

	return request.RateLimitDefinitions{
		// spot specific rate limiters
		accountSummaryInfoEPL:                         request.GetRateLimiterWithWeight(managementRate, 20),
		allAccountEPL:                                 request.GetRateLimiterWithWeight(managementRate, 5),
		accountDetailEPL:                              request.GetRateLimiterWithWeight(managementRate, 5),
		accountLedgersEPL:                             request.GetRateLimiterWithWeight(managementRate, 2),
		hfAccountLedgersEPL:                           request.GetRateLimiterWithWeight(spotRate, 2),
		hfAccountLedgersMarginEPL:                     request.GetRateLimiterWithWeight(spotRate, 2),
		futuresAccountLedgersEPL:                      request.GetRateLimiterWithWeight(spotRate, 2),
		subAccountInfoV1EPL:                           request.GetRateLimiterWithWeight(managementRate, 20),
		allSubAccountsInfoV2EPL:                       request.GetRateLimiterWithWeight(managementRate, 20),
		createSubUserEPL:                              request.GetRateLimiterWithWeight(managementRate, 20),
		subAccountsEPL:                                request.GetRateLimiterWithWeight(managementRate, 15),
		subAccountBalancesEPL:                         request.GetRateLimiterWithWeight(managementRate, 20),
		allSubAccountBalancesV2EPL:                    request.GetRateLimiterWithWeight(managementRate, 20),
		subAccountSpotAPIListEPL:                      request.GetRateLimiterWithWeight(managementRate, 20),
		createSpotAPIForSubAccountEPL:                 request.GetRateLimiterWithWeight(managementRate, 20),
		modifySubAccountSpotAPIEPL:                    request.GetRateLimiterWithWeight(managementRate, 30),
		deleteSubAccountSpotAPIEPL:                    request.GetRateLimiterWithWeight(managementRate, 30),
		marginAccountDetailEPL:                        request.GetRateLimiterWithWeight(spotRate, 40),
		crossMarginAccountsDetailEPL:                  request.GetRateLimiterWithWeight(spotRate, 15),
		isolatedMarginAccountDetailEPL:                request.GetRateLimiterWithWeight(spotRate, 15),
		futuresAccountsDetailEPL:                      request.GetRateLimiterWithWeight(futuresRate, 5),
		tradingPairActualFeeEPL:                       request.GetRateLimiterWithWeight(spotRate, 3),
		allFuturesSubAccountBalancesEPL:               request.GetRateLimiterWithWeight(futuresRate, 6),
		futuresTradingPairFeeEPL:                      request.GetRateLimiterWithWeight(futuresRate, 3),
		futuresPositionHistoryEPL:                     request.GetRateLimiterWithWeight(futuresRate, 2),
		futuresMaxOpenPositionsSizeEPL:                request.GetRateLimiterWithWeight(futuresRate, 2),
		futuresAllTickersInfoEPL:                      request.GetRateLimiterWithWeight(futuresRate, 5),
		createDepositAddressEPL:                       request.GetRateLimiterWithWeight(managementRate, 20),
		depositAddressesV2EPL:                         request.GetRateLimiterWithWeight(managementRate, 5),
		depositAddressesV1EPL:                         request.GetRateLimiterWithWeight(managementRate, 5),
		depositListEPL:                                request.GetRateLimiterWithWeight(managementRate, 5),
		historicDepositListEPL:                        request.GetRateLimiterWithWeight(managementRate, 5),
		withdrawalListEPL:                             request.GetRateLimiterWithWeight(managementRate, 20),
		retrieveV1HistoricalWithdrawalListEPL:         request.GetRateLimiterWithWeight(managementRate, 20),
		withdrawalQuotaEPL:                            request.GetRateLimiterWithWeight(managementRate, 20),
		applyWithdrawalEPL:                            request.GetRateLimiterWithWeight(managementRate, 5),
		cancelWithdrawalsEPL:                          request.GetRateLimiterWithWeight(managementRate, 20),
		getTransferablesEPL:                           request.GetRateLimiterWithWeight(managementRate, 20),
		flexiTransferEPL:                              request.GetRateLimiterWithWeight(managementRate, 4),
		masterSubUserTransferEPL:                      request.GetRateLimiterWithWeight(managementRate, 30),
		innerTransferEPL:                              request.GetRateLimiterWithWeight(managementRate, 10),
		toMainOrTradeAccountEPL:                       request.GetRateLimiterWithWeight(managementRate, 20),
		toFuturesAccountEPL:                           request.GetRateLimiterWithWeight(managementRate, 20),
		futuresTransferOutRequestRecordsEPL:           request.GetRateLimiterWithWeight(managementRate, 20),
		basicFeesEPL:                                  request.GetRateLimiterWithWeight(spotRate, 3),
		tradeFeesEPL:                                  request.GetRateLimiterWithWeight(spotRate, 3),
		spotCurrenciesV3EPL:                           request.GetRateLimiterWithWeight(publicRate, 3),
		spotCurrencyDetailEPL:                         request.GetRateLimiterWithWeight(publicRate, 3),
		symbolsEPL:                                    request.GetRateLimiterWithWeight(publicRate, 4),
		tickersEPL:                                    request.GetRateLimiterWithWeight(publicRate, 2),
		allTickersEPL:                                 request.GetRateLimiterWithWeight(publicRate, 15),
		statistics24HrEPL:                             request.GetRateLimiterWithWeight(publicRate, 15),
		marketListEPL:                                 request.GetRateLimiterWithWeight(publicRate, 3),
		partOrderbook20EPL:                            request.GetRateLimiterWithWeight(publicRate, 2),
		partOrderbook100EPL:                           request.GetRateLimiterWithWeight(publicRate, 4),
		fullOrderbookEPL:                              request.GetRateLimiterWithWeight(spotRate, 3),
		tradeHistoryEPL:                               request.GetRateLimiterWithWeight(publicRate, 3),
		klinesEPL:                                     request.GetRateLimiterWithWeight(publicRate, 3),
		fiatPriceEPL:                                  request.GetRateLimiterWithWeight(publicRate, 3),
		currentServerTimeEPL:                          request.GetRateLimiterWithWeight(publicRate, 3),
		serviceStatusEPL:                              request.GetRateLimiterWithWeight(publicRate, 3),
		hfPlaceOrderEPL:                               request.GetRateLimiterWithWeight(spotRate, 1),
		hfSyncPlaceOrderEPL:                           request.GetRateLimiterWithWeight(spotRate, 1),
		hfMultipleOrdersEPL:                           request.GetRateLimiterWithWeight(spotRate, 1),
		hfSyncPlaceMultipleHFOrdersEPL:                request.GetRateLimiterWithWeight(spotRate, 1),
		hfModifyOrderEPL:                              request.GetRateLimiterWithWeight(spotRate, 3),
		cancelHFOrderEPL:                              request.GetRateLimiterWithWeight(spotRate, 1),
		hfSyncCancelOrderEPL:                          request.GetRateLimiterWithWeight(spotRate, 1),
		hfCancelOrderByClientOrderIDEPL:               request.GetRateLimiterWithWeight(spotRate, 1),
		cancelSpecifiedNumberHFOrdersByOrderIDEPL:     request.GetRateLimiterWithWeight(spotRate, 2),
		hfCancelAllOrdersBySymbolEPL:                  request.GetRateLimiterWithWeight(spotRate, 2),
		hfCancelAllOrdersEPL:                          request.GetRateLimiterWithWeight(spotRate, 30),
		hfGetAllActiveOrdersEPL:                       request.GetRateLimiterWithWeight(spotRate, 2),
		hfSymbolsWithActiveOrdersEPL:                  request.GetRateLimiterWithWeight(spotRate, 2),
		hfCompletedOrderListEPL:                       request.GetRateLimiterWithWeight(spotRate, 2),
		hfOrderDetailByOrderIDEPL:                     request.GetRateLimiterWithWeight(spotRate, 2),
		autoCancelHFOrderSettingEPL:                   request.GetRateLimiterWithWeight(spotRate, 2),
		autoCancelHFOrderSettingQueryEPL:              request.GetRateLimiterWithWeight(spotRate, 2),
		hfFilledListEPL:                               request.GetRateLimiterWithWeight(spotRate, 2),
		placeOrderEPL:                                 request.GetRateLimiterWithWeight(spotRate, 2),
		placeBulkOrdersEPL:                            request.GetRateLimiterWithWeight(spotRate, 3),
		cancelOrderEPL:                                request.GetRateLimiterWithWeight(spotRate, 3),
		cancelOrderByClientOrderIDEPL:                 request.GetRateLimiterWithWeight(spotRate, 5),
		cancelAllOrdersEPL:                            request.GetRateLimiterWithWeight(spotRate, 20),
		listOrdersEPL:                                 request.GetRateLimiterWithWeight(spotRate, 2),
		recentOrdersEPL:                               request.GetRateLimiterWithWeight(spotRate, 3),
		orderDetailByIDEPL:                            request.GetRateLimiterWithWeight(spotRate, 2),
		getOrderByClientSuppliedOrderIDEPL:            request.GetRateLimiterWithWeight(spotRate, 3),
		listFillsEPL:                                  request.GetRateLimiterWithWeight(spotRate, 10),
		getRecentFillsEPL:                             request.GetRateLimiterWithWeight(spotRate, 20),
		placeStopOrderEPL:                             request.GetRateLimiterWithWeight(spotRate, 2),
		cancelStopOrderEPL:                            request.GetRateLimiterWithWeight(spotRate, 3),
		cancelStopOrderByClientIDEPL:                  request.GetRateLimiterWithWeight(spotRate, 5),
		cancelStopOrdersEPL:                           request.GetRateLimiterWithWeight(spotRate, 3),
		listStopOrdersEPL:                             request.GetRateLimiterWithWeight(spotRate, 8),
		getStopOrderDetailEPL:                         request.GetRateLimiterWithWeight(spotRate, 3),
		getStopOrderByClientIDEPL:                     request.GetRateLimiterWithWeight(spotRate, 3),
		placeOCOOrderEPL:                              request.GetRateLimiterWithWeight(spotRate, 2),
		cancelOCOOrderByIDEPL:                         request.GetRateLimiterWithWeight(spotRate, 3),
		cancelMultipleOCOOrdersEPL:                    request.GetRateLimiterWithWeight(spotRate, 3),
		getOCOOrderByIDEPL:                            request.GetRateLimiterWithWeight(spotRate, 2),
		getOCOOrderDetailsByOrderIDEPL:                request.GetRateLimiterWithWeight(spotRate, 2),
		getOCOOrdersEPL:                               request.GetRateLimiterWithWeight(spotRate, 2),
		placeMarginOrderEPL:                           request.GetRateLimiterWithWeight(spotRate, 5),
		cancelMarginHFOrderByIDEPL:                    request.GetRateLimiterWithWeight(spotRate, 5),
		getMarginHFOrderDetailByID:                    request.GetRateLimiterWithWeight(spotRate, 5),
		cancelAllMarginHFOrdersBySymbolEPL:            request.GetRateLimiterWithWeight(spotRate, 10),
		getActiveMarginHFOrdersEPL:                    request.GetRateLimiterWithWeight(spotRate, 4),
		getFilledHFMarginOrdersEPL:                    request.GetRateLimiterWithWeight(spotRate, 10),
		getMarginHFOrderDetailByOrderIDEPL:            request.GetRateLimiterWithWeight(spotRate, 4),
		getMarginHFTradeFillsEPL:                      request.GetRateLimiterWithWeight(spotRate, 5),
		placeMarginOrdersEPL:                          request.GetRateLimiterWithWeight(spotRate, 5),
		leveragedTokenInfoEPL:                         request.GetRateLimiterWithWeight(spotRate, 25),
		getMarkPriceEPL:                               request.GetRateLimiterWithWeight(publicRate, 2),
		getAllMarginMarkPriceEPL:                      request.GetRateLimiterWithWeight(publicRate, 10),
		getMarginConfigurationEPL:                     request.GetRateLimiterWithWeight(spotRate, 25),
		crossIsolatedMarginRiskLimitCurrencyConfigEPL: request.GetRateLimiterWithWeight(spotRate, 20),
		isolatedMarginPairConfigEPL:                   request.GetRateLimiterWithWeight(spotRate, 20),
		isolatedMarginAccountInfoEPL:                  request.GetRateLimiterWithWeight(spotRate, 50),
		singleIsolatedMarginAccountInfoEPL:            request.GetRateLimiterWithWeight(spotRate, 50),
		postMarginBorrowOrderEPL:                      request.GetRateLimiterWithWeight(spotRate, 15),
		postMarginRepaymentEPL:                        request.GetRateLimiterWithWeight(spotRate, 10),
		getCrossIsolatedMarginInterestRecordsEPL:      request.GetRateLimiterWithWeight(spotRate, 20),
		marginBorrowingHistoryEPL:                     request.GetRateLimiterWithWeight(spotRate, 15),
		marginRepaymentHistoryEPL:                     request.GetRateLimiterWithWeight(spotRate, 15),
		lendingCurrencyInfoEPL:                        request.GetRateLimiterWithWeight(spotRate, 10),
		interestRateEPL:                               request.GetRateLimiterWithWeight(publicRate, 5),
		marginLendingSubscriptionEPL:                  request.GetRateLimiterWithWeight(spotRate, 15),
		redemptionEPL:                                 request.GetRateLimiterWithWeight(spotRate, 15),
		modifySubscriptionEPL:                         request.GetRateLimiterWithWeight(spotRate, 10),
		getRedemptionOrdersEPL:                        request.GetRateLimiterWithWeight(spotRate, 10),
		getSubscriptionOrdersEPL:                      request.GetRateLimiterWithWeight(spotRate, 10),
		futuresOpenContractsEPL:                       request.GetRateLimiterWithWeight(publicRate, 3),
		futuresContractEPL:                            request.GetRateLimiterWithWeight(publicRate, 3),
		futuresTickerEPL:                              request.GetRateLimiterWithWeight(publicRate, 2),
		futuresOrderbookEPL:                           request.GetRateLimiterWithWeight(publicRate, 3),
		futuresPartOrderbookDepth20EPL:                request.GetRateLimiterWithWeight(publicRate, 5),
		futuresPartOrderbookDepth100EPL:               request.GetRateLimiterWithWeight(publicRate, 10),
		futuresTransactionHistoryEPL:                  request.GetRateLimiterWithWeight(publicRate, 5),
		futuresKlineEPL:                               request.GetRateLimiterWithWeight(publicRate, 3),
		futuresInterestRateEPL:                        request.GetRateLimiterWithWeight(publicRate, 5),
		futuresIndexListEPL:                           request.GetRateLimiterWithWeight(publicRate, 2),
		futuresCurrentMarkPriceEPL:                    request.GetRateLimiterWithWeight(publicRate, 3),
		futuresPremiumIndexEPL:                        request.GetRateLimiterWithWeight(publicRate, 3),
		futuresTransactionVolumeEPL:                   request.GetRateLimiterWithWeight(futuresRate, 3),
		futuresServerTimeEPL:                          request.GetRateLimiterWithWeight(publicRate, 2),
		futuresServiceStatusEPL:                       request.GetRateLimiterWithWeight(publicRate, 4),
		multipleFuturesOrdersEPL:                      request.GetRateLimiterWithWeight(futuresRate, 20),
		futuresCancelAnOrderEPL:                       request.GetRateLimiterWithWeight(futuresRate, 1),
		futuresPlaceOrderEPL:                          request.GetRateLimiterWithWeight(futuresRate, 2),
		futuresLimitOrderMassCancelationEPL:           request.GetRateLimiterWithWeight(futuresRate, 30),
		cancelUntriggeredFuturesStopOrdersEPL:         request.GetRateLimiterWithWeight(futuresRate, 15),
		futuresCancelMultipleLimitOrdersEPL:           request.GetRateLimiterWithWeight(futuresRate, 30),
		futuresRetrieveOrderListEPL:                   request.GetRateLimiterWithWeight(futuresRate, 2),
		futuresRecentCompletedOrdersEPL:               request.GetRateLimiterWithWeight(futuresRate, 5),
		futuresOrdersByIDEPL:                          request.GetRateLimiterWithWeight(futuresRate, 5),
		futuresRetrieveFillsEPL:                       request.GetRateLimiterWithWeight(futuresRate, 5),
		futuresRecentFillsEPL:                         request.GetRateLimiterWithWeight(futuresRate, 3),
		futuresOpenOrderStatsEPL:                      request.GetRateLimiterWithWeight(futuresRate, 10),
		futuresPositionEPL:                            request.GetRateLimiterWithWeight(futuresRate, 2),
		futuresPositionListEPL:                        request.GetRateLimiterWithWeight(futuresRate, 2),
		setAutoDepositMarginEPL:                       request.GetRateLimiterWithWeight(futuresRate, 4),
		maxWithdrawMarginEPL:                          request.GetRateLimiterWithWeight(futuresRate, 10),
		removeMarginManuallyEPL:                       request.GetRateLimiterWithWeight(futuresRate, 10),
		futuresAddMarginManuallyEPL:                   request.GetRateLimiterWithWeight(futuresRate, 4),
		futuresRiskLimitLevelEPL:                      request.GetRateLimiterWithWeight(futuresRate, 5),
		futuresUpdateRiskLimitLevelEPL:                request.GetRateLimiterWithWeight(futuresRate, 4),
		futuresCurrentFundingRateEPL:                  request.GetRateLimiterWithWeight(publicRate, 2),
		futuresPublicFundingRateEPL:                   request.GetRateLimiterWithWeight(publicRate, 5),
		futuresFundingHistoryEPL:                      request.GetRateLimiterWithWeight(futuresRate, 5),
		spotAuthenticationEPL:                         request.GetRateLimiterWithWeight(spotRate, 10),
		futuresAuthenticationEPL:                      request.GetRateLimiterWithWeight(futuresRate, 10),
		futuresOrderDetailsByClientOrderIDEPL:         request.GetRateLimiterWithWeight(futuresRate, 5),
		modifySubAccountAPIEPL:                        request.GetRateLimiterWithWeight(managementRate, 30),
		allSubAccountsBalanceEPL:                      request.GetRateLimiterWithWeight(managementRate, 20),
		allUserSubAccountsV2EPL:                       request.GetRateLimiterWithWeight(managementRate, 20),
		futuresRetrieveTransactionHistoryEPL:          request.GetRateLimiterWithWeight(managementRate, 2),
		futuresAccountOverviewEPL:                     request.GetRateLimiterWithWeight(futuresRate, 5),
		createSubAccountAPIKeyEPL:                     request.GetRateLimiterWithWeight(managementRate, 20),
		transferOutToMainEPL:                          request.GetRateLimiterWithWeight(managementRate, 20),
		transferFundToFuturesAccountEPL:               request.GetRateLimiterWithWeight(managementRate, 20),
		futuresTransferOutListEPL:                     request.GetRateLimiterWithWeight(managementRate, 20),

		subscribeToEarnEPL:       request.GetRateLimiterWithWeight(spotRate, 5),
		earnRedemptionEPL:        request.GetRateLimiterWithWeight(spotRate, 5),
		earnRedemptionPreviewEPL: request.GetRateLimiterWithWeight(spotRate, 5),

		kucoinEarnSavingsProductsEPL:           request.GetRateLimiterWithWeight(spotRate, 5),
		kucoinEarnFixedIncomeCurrentHoldingEPL: request.GetRateLimiterWithWeight(spotRate, 5),
		earnLimitedTimePromotionProductEPL:     request.GetRateLimiterWithWeight(spotRate, 5),

		earnKCSStakingProductEPL: request.GetRateLimiterWithWeight(spotRate, 5),
		earnStakingProductEPL:    request.GetRateLimiterWithWeight(spotRate, 5),

		vipLendingEPL:               request.GetRateLimiterWithWeight(spotRate, 1),
		affilateUserRebateInfoEPL:   request.GetRateLimiterWithWeight(spotRate, 30),
		marginPairsConfigurationEPL: request.GetRateLimiterWithWeight(spotRate, 5),
		modifyLeverageMultiplierEPL: request.GetRateLimiterWithWeight(spotRate, 5),
		marginActiveHFOrdersEPL:     request.GetRateLimiterWithWeight(spotRate, 2),
	}
}
