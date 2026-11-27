package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Ratelimit intervals.
const (
	oneSecondInterval    = time.Second
	twoSecondsInterval   = 2 * time.Second
	threeSecondsInterval = 3 * time.Second
	fiveSecondsInterval  = 5 * time.Second
	tenSecondsInterval   = 10 * time.Second
)

const (
	placeOrderEPL request.EndpointLimit = iota + 1
	placeMultipleOrdersEPL
	cancelOrderEPL
	cancelMultipleOrdersEPL
	amendOrderEPL
	amendMultipleOrdersEPL
	closePositionEPL
	getOrderDetEPL
	getOrderListEPL
	getOrderHistory7DaysEPL
	getOrderHistory3MonthsEPL
	getTransactionDetail3DaysEPL
	getTransactionDetail3MonthsEPL
	cancelAllAfterCountdownEPL
	getTradeAccountRateLimitEPL
	orderPreCheckEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	amendAlgoOrderEPL
	getAlgoOrderDetailEPL
	getAlgoOrderListEPL
	getAlgoOrderHistoryEPL
	getEasyConvertCurrencyListEPL
	placeEasyConvertEPL
	getEasyConvertHistoryEPL
	getOneClickRepayHistoryEPL
	oneClickRepayCurrencyListEPL
	tradeOneClickRepayEPL
	massCancelMMPOrderEPL
	getCounterpartiesEPL
	createRequestForQuoteEPL
	cancelRequestForQuoteEPL
	cancelMultipleRequestForQuotesEPL
	cancelAllRequestForQuotesEPL
	executeQuoteEPL
	getQuoteProductsEPL
	setQuoteProductsEPL
	resetRequestForQuoteMMPEPL
	setMMPEPL
	getMMPConfigEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRequestForQuotesEPL
	getQuotesEPL
	getTradesEPL
	getTradesHistoryEPL
	optionInstrumentTradeFamilyEPL
	optionTradesEPL
	getPublicTradesEPL
	getCurrenciesEPL
	getBalanceEPL
	getNonTradableAssetsEPL
	getAccountAssetValuationEPL
	fundsTransferEPL
	getFundsTransferStateEPL
	assetBillsDetailsEPL
	getDepositAddressEPL
	getDepositHistoryEPL
	withdrawalEPL
	cancelWithdrawalEPL
	getWithdrawalHistoryEPL
	getDepositWithdrawalStatusEPL
	getPublicExchangeListEPL
	getSavingBalanceEPL
	savingsPurchaseRedemptionEPL
	getLendingHistoryEPL
	getPublicBorrowInfoEPL
	getPublicBorrowHistoryEPL
	getConvertCurrenciesEPL
	getMonthlyStatementEPL
	applyForMonthlyStatementEPL
	getConvertCurrencyPairEPL
	estimateQuoteEPL
	convertTradeEPL
	getConvertHistoryEPL
	getAccountBalanceEPL
	getPositionsEPL
	getPositionsHistoryEPL
	getAccountAndPositionRiskEPL
	getBillsDetailsEPL
	getBillsDetailArchiveEPL
	billHistoryArchiveEPL
	getBillHistoryArchiveEPL
	getAccountConfigurationEPL
	setPositionModeEPL
	setLeverageEPL
	getMaximumBuyOrSellAmountEPL
	getMaximumAvailableTradableAmountEPL
	increaseOrDecreaseMarginEPL
	getLeverageEPL
	getLeverateEstimatedInfoEPL
	getTheMaximumLoanOfInstrumentEPL
	getFeeRatesEPL
	getInterestAccruedDataEPL
	getInterestRateEPL
	setGreeksEPL
	isolatedMarginTradingSettingsEPL
	getMaximumWithdrawalsEPL
	getAccountRiskStateEPL
	getBorrowInterestAndLimitEPL
	manualBorrowOrRepayEPL
	setAutoRepayEPL
	getBorrowRepayHistoryEPL
	newPositionBuilderEPL
	setRiskOffsetAmountEPL
	getGreeksEPL
	getPMLimitationEPL
	activateOptionEPL
	setAutoLoanEPL
	setAccountLevelEPL
	resetMMPStatusEPL
	viewSubaccountListEPL
	resetSubAccountAPIKeyEPL
	getSubaccountTradingBalanceEPL
	getSubaccountFundingBalanceEPL
	getSubAccountMaxWithdrawalEPL
	historyOfSubaccountTransferEPL
	managedSubAccountTransferEPL
	masterAccountsManageTransfersBetweenSubaccountEPL
	setPermissionOfTransferOutEPL
	getCustodyTradingSubaccountListEPL
	gridTradingEPL
	amendGridAlgoOrderEPL
	stopGridAlgoOrderEPL
	closePositionForForContractGridEPL
	cancelClosePositionOrderForContractGridEPL
	instantTriggerGridAlgoOrderEPL
	getGridAlgoOrderListEPL
	getGridAlgoOrderHistoryEPL
	getGridAlgoOrderDetailsEPL
	getGridAlgoSubOrdersEPL
	getGridAlgoOrderPositionsEPL
	spotGridWithdrawIncomeEPL
	computeMarginBalanceEPL
	adjustMarginBalanceEPL
	getGridAIParameterEPL
	computeMinInvestmentEPL
	rsiBackTestingEPL
	signalBotOrderDetailsEPL
	signalBotOrderPositionsEPL
	signalBotSubOrdersEPL
	signalBotEventHistoryEPL
	placeRecurringBuyOrderEPL
	amendRecurringBuyOrderEPL
	stopRecurringBuyOrderEPL
	getRecurringBuyOrderListEPL
	getRecurringBuyOrderHistoryEPL
	getRecurringBuyOrderDetailEPL
	getRecurringBuySubOrdersEPL
	getExistingLeadingPositionsEPL
	getLeadingPositionHistoryEPL
	placeLeadingStopOrderEPL
	closeLeadingPositionEPL
	getLeadingInstrumentsEPL
	getProfitSharingLimitEPL
	getTotalProfitSharingEPL
	setFirstCopySettingsEPL
	amendFirstCopySettingsEPL
	stopCopyingEPL
	getCopySettingsEPL
	getLeadTraderRanksEPL
	getLeadTraderWeeklyProfitAndLossEPL
	getLeadTraderDailyProfitAndLossEPL
	getLeadTraderStatsEPL
	getLeadTraderCurrencyPreferencesEPL
	getTraderCurrentLeadPositionsEPL
	getLeadTraderLeadPositionHistoryEPL
	getOfferEPL
	purchaseEPL
	redeemEPL
	cancelPurchaseOrRedemptionEPL
	getEarnActiveOrdersEPL
	getFundingOrderHistoryEPL
	getProductInfoEPL

	purchaseETHStakingEPL
	redeemETHStakingEPL
	getBETHBalanceEPL
	getPurchaseRedeemHistoryEPL
	getAPYHistoryEPL

	getTickersEPL
	getTickerEPL
	getPremiumHistoryEPL
	getIndexTickersEPL
	getOrderBookEPL
	getCandlesticksEPL
	getTradesRequestEPL
	get24HTotalVolumeEPL
	getExchangeRateRequestEPL
	getIndexComponentsEPL
	getBlockTickersEPL
	getBlockTradesEPL
	placeSpreadOrderEPL
	cancelSpreadOrderEPL
	cancelAllSpreadOrderEPL
	amendSpreadOrderEPL
	getSpreadOrderDetailsEPL
	getSpreadOrderTradesEPL
	getSpreadsEPL
	getSpreadOrderbookEPL
	getSpreadTickerEPL
	getSpreadPublicTradesEPL
	getSpreadCandlesticksEPL
	getSpreadCandlesticksHistoryEPL
	cancelAllSpreadOrdersAfterEPL
	getActiveSpreadOrdersEPL
	getSpreadOrders21DaysEPL
	getSpreadOrders3MonthsEPL
	getInstrumentsEPL
	getDeliveryExerciseHistoryEPL
	getOpenInterestEPL
	getFundingEPL
	getFundingRateHistoryEPL
	getLimitPriceEPL
	getOptionMarketDateEPL
	getEstimatedDeliveryExercisePriceEPL
	getDiscountRateAndInterestFreeQuotaEPL
	getSystemTimeEPL
	getMarkPriceEPL
	getPositionTiersEPL
	getInterestRateAndLoanQuotaEPL
	getUnderlyingEPL
	getInsuranceFundEPL
	unitConvertEPL
	optionTickBandsEPL
	getSupportCoinEPL
	getTakerVolumeEPL
	getMarginLendingRatioEPL
	getLongShortRatioEPL
	getContractsOpenInterestAndVolumeEPL
	getOptionsOpenInterestAndVolumeEPL
	getPutCallRatioEPL
	getOpenInterestAndVolumeEPL
	getTakerFlowEPL
	getEventStatusEPL
	getCandlestickHistoryEPL
	getIndexCandlesticksEPL
	getIndexCandlesticksHistoryEPL
	getMarkPriceCandlesticksHistoryEPL
	getEconomicCalendarEPL
	getMyLeadTradersEPL

	getUserAffiliateRebateInformationEPL

	rubikGetContractOpenInterestHistoryEPL
	rubikContractTakerVolumeEPL
	rubikTopTradersContractLongShortRatioEPL

	getAccountInstrumentsEPL
	getAnnouncementsEPL
	getAnnouncementTypeEPL

	getDepositOrderDetailEPL
	getDepositOrderHistoryEPL
	getWithdrawalOrderDetailEPL
	getFiatWithdrawalOrderHistoryEPL
	cancelWithdrawalOrderEPL
	createWithdrawalOrderEPL
	getWithdrawalPaymentMethodsEPL
	getFiatDepositPaymentMethodsEPL

	websocketRequestEPL

	// Financial Product: ETH staking, SOL staking, Stable Rewards, OKUSD, Flexible loan and Dual investment
	cancelETHStakingRedemptionEPL
	getSOLStakingProductInfoEPL
	purchaseSOLStakingEPL
	redeemSOLStakingEPL
	getSOLStakingBalanceEPL
	getSOLStakingPurchaseRedeemHistoryEPL
	getSOLStakingAPYHistoryEPL
	getStableRewardsProductInfoEPL
	getStableRewardsBalanceEPL
	getStableRewardsAPYHistoryEPL
	getOKUSDLimitsEPL
	subscribeOKUSDEPL
	redeemOKUSDEPL
	getOKUSDAccountEPL
	getOKUSDSubscriptionHistoryEPL
	getOKUSDRedemptionHistoryEPL
	getOKUSDRewardsHistoryEPL
	getOKUSDAPRHistoryEPL
	getFlexibleLoanBorrowableCurrenciesEPL
	getFlexibleLoanCollateralAssetsEPL
	getFlexibleLoanMaximumLoanAmountEPL
	getFlexibleLoanMaximumCollateralRedeemAmountEPL
	adjustFlexibleLoanCollateralEPL
	borrowFlexibleLoanEPL
	repayFlexibleLoanEPL
	getFlexibleLoanEModeInfoEPL
	getFlexibleLoanInfoEPL
	getFlexibleLoanHistoryEPL
	getFlexibleLoanAccruedInterestEPL
	getDualInvestmentCurrencyPairsEPL
	getDualInvestmentProductsEPL
	requestDualInvestmentQuoteEPL
	tradeDualInvestmentEPL
	requestDualInvestmentRedeemQuoteEPL
	redeemDualInvestmentEPL
	getDualInvestmentOrderStateEPL
	getDualInvestmentOrderHistoryEPL
	// Funding Account
	getAssetBillsHistoryEPL
	getLockedTravelRuleDepositsEPL
	unlockTravelRuleDepositEPL
	getBuySellCurrenciesEPL
	getBuySellCurrencyPairEPL
	getBuySellQuoteEPL
	placeBuySellTradeEPL
	getBuySellTradeHistoryEPL

	// Sub-account
	createSubAccountEPL
	createSubAccountAPIKeyEPL
	getSubAccountAPIKeysEPL
	deleteSubAccountAPIKeyEPL
	// Order Book Trading > Trade
	getOneClickRepayCurrencyListV2EPL
	tradeOneClickRepayV2EPL
	getOneClickRepayHistoryV2EPL
	// Copy Trading
	amendLeadingInstrumentsEPL
	getUnrealisedProfitSharingDetailsEPL
	getTotalUnrealisedProfitSharingEPL
	amendProfitSharingRatioEPL
	getCopyTradingAccountConfigurationEPL
	getCopyTradingConfigurationEPL
	getCopyTradersEPL

	// Affiliate
	getAffiliatePerformanceSummaryEPL
	getInviteeDetailEPL
	getInviteeListEPL
	getAffiliateLinkListEPL
	getCoInviterLinkListEPL
	getSubAffiliateListEPL
	// Market Data, Public Data and Trading Statistics
	getRPIOrderBookEPL
	getFullOrderBookEPL
	getCallAuctionDetailsEPL
	getEventContractSeriesEPL
	getEventContractEventsEPL
	getCorporateActionsEPL
	getEventContractMarketsEPL
	getEstimatedSettlementPriceEPL
	getFuturesSettlementHistoryEPL
	getMarkPriceCandlesticksEPL
	getHistoricalMarketDataEPL
	getMMInstrumentTypesEPL
	getDeltaHedgeCurrenciesEPL
	getContractLongShortAccountRatioEPL
	getTopTradersContractLongShortPositionRatioEPL
	getOpenInterestAndVolumeStrikeEPL
	// Trading Account
	getBillTypesEPL
	setFeeTypeEPL
	positionBuilderTrendGraphEPL
	activateFeatureEPL
	presetAccountModeSwitchEPL
	precheckAccountModeSwitchEPL
	setCollateralAssetsEPL
	getCollateralAssetsEPL
	movePositionsEPL
	getMovePositionsHistoryEPL
	setAutoEarnEPL
	setSettleCurrencyEPL
	setTradingConfigEPL
	precheckSetDeltaNeutralEPL
	increaseDemoAccountBalanceEPL
	reduceDemoAccountBalanceEPL
	getGLPTodayPerformanceEPL
	getGLPHistoricalPerformanceEPL
	getGLPV2OverviewEPL
	getGLPV2AssessmentDetailsEPL
	getGLPV2PerformanceHistoryEPL
	// Block Trading
	setRequestForQuoteMMPEPL
	getRequestForQuoteMMPConfigEPL
	cancelAllQuotesAfterCountdownEPL
	getBlockTickerEPL
	// Grid Trading
	amendGridAlgoOrderBasicParametersEPL
	addGridInvestmentEPL
	getMaxGridQuantityEPL
	copyGridAlgoOrderEPL

	// DCA Trading
	placeDCAAlgoOrderEPL
	amendSpotDCABasicParametersEPL
	stopDCAAlgoOrderEPL
	getDCAAlgoOrderListEPL
	getDCAAlgoOrderHistoryEPL
	getDCASubOrdersEPL
	addDCAInvestmentEPL
	amendDCAReinvestmentEPL
	amendDCATakeProfitEPL
	getDCAPositionDetailsEPL
	getDCACycleListEPL
	addDCAMarginEPL
	reduceDCAMarginEPL

	// Signal bot trading
	createSignalEPL
	getSignalsEPL
	createSignalBotEPL
	cancelSignalBotsEPL
	adjustSignalBotMarginBalanceEPL
	amendSignalBotTakeProfitStopLossEPL
	setSignalBotInstrumentsEPL
	getActiveSignalBotsEPL
	getSignalBotHistoryEPL
	getSignalBotPositionHistoryEPL
	closeSignalBotPositionEPL
	placeSignalBotSubOrderEPL
	cancelSignalBotSubOrderEPL

	// Recurring Buy
	amendRecurringBuyTimeEPL
	amendRecurringBuyAmountEPL
	addRecurringBuyInvestmentEPL
	pauseRecurringBuyEPL
	restartRecurringBuyEPL
	amendRecurringBuyPriceRangeEPL

	// Websocket connections
	websocketConnectionEPL

	// endpointLimitCount follows the last endpoint limit; keep it last
	endpointLimitCount
)

var rateLimits = newRateLimits()

// newRateLimits returns OKX's rate limit definitions
func newRateLimits() request.RateLimitDefinitions {
	placeOrders := orderBudgetLimiter(300, orderBudgets[placeMultipleOrdersEPL])
	cancelOrders := orderBudgetLimiter(300, orderBudgets[cancelMultipleOrdersEPL])
	amendOrders := orderBudgetLimiter(300, orderBudgets[amendMultipleOrdersEPL])
	cancelAlgos := orderBudgetLimiter(20, orderBudgets[cancelAlgoOrderEPL])
	defs := request.RateLimitDefinitions{
		// Trade Endpoints
		placeOrderEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		placeMultipleOrdersEPL:         request.GetRateLimiterWithWeight(placeOrders, 1), // Budgets orders: a request takes a token per order
		cancelOrderEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		cancelMultipleOrdersEPL:        request.GetRateLimiterWithWeight(cancelOrders, 1), // Budgets orders: a request takes a token per order
		amendOrderEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		amendMultipleOrdersEPL:         request.GetRateLimiterWithWeight(amendOrders, 1), // Budgets orders: a request takes a token per order
		closePositionEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOrderDetEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getOrderListEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getOrderHistory7DaysEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getOrderHistory3MonthsEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getTransactionDetail3DaysEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getTransactionDetail3MonthsEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		cancelAllAfterCountdownEPL:     request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getTradeAccountRateLimitEPL:    request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		orderPreCheckEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		placeAlgoOrderEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAlgoOrderEPL:             request.GetRateLimiterWithWeight(cancelAlgos, 1), // Budgets orders: a request takes a token per order
		amendAlgoOrderEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderDetailEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderListEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderHistoryEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEasyConvertCurrencyListEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		placeEasyConvertEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getEasyConvertHistoryEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getOneClickRepayHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		oneClickRepayCurrencyListEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		tradeOneClickRepayEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		massCancelMMPOrderEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Block Trading endpoints
		getCounterpartiesEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		createRequestForQuoteEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		cancelRequestForQuoteEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		cancelMultipleRequestForQuotesEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		cancelAllRequestForQuotesEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		executeQuoteEPL:                   request.NewRateLimitWithWeight(threeSecondsInterval, 2, 1),
		getQuoteProductsEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setQuoteProductsEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		resetMMPStatusEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		resetRequestForQuoteMMPEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setMMPEPL:                         request.NewRateLimitWithWeight(tenSecondsInterval, 2, 1),
		getMMPConfigEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		createQuoteEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 50, 1),
		cancelQuoteEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 50, 1),
		cancelMultipleQuotesEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		cancelAllQuotesEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getRequestForQuotesEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getQuotesEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getTradesEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTradesHistoryEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		optionInstrumentTradeFamilyEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		optionTradesEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getPublicTradesEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Funding
		getCurrenciesEPL:              request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getBalanceEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getNonTradableAssetsEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getAccountAssetValuationEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		fundsTransferEPL:              request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getFundsTransferStateEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 10, 1),
		assetBillsDetailsEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getDepositAddressEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getDepositHistoryEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		withdrawalEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		cancelWithdrawalEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getWithdrawalHistoryEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getDepositWithdrawalStatusEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getPublicExchangeListEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getSavingBalanceEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		savingsPurchaseRedemptionEPL:  request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getLendingHistoryEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getPublicBorrowInfoEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getPublicBorrowHistoryEPL:     request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		// Convert
		getMonthlyStatementEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		applyForMonthlyStatementEPL: request.NewRateLimitWithWeight(time.Hour*24*30, 20, 1),
		getConvertCurrenciesEPL:     request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getConvertCurrencyPairEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		estimateQuoteEPL:            request.NewRateLimitWithWeight(oneSecondInterval, 10, 1),
		convertTradeEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 10, 1),
		getConvertHistoryEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		// Account
		getAccountBalanceEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionsEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionsHistoryEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getAccountAndPositionRiskEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getBillsDetailsEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, 5, 1),
		getBillsDetailArchiveEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		billHistoryArchiveEPL:                request.NewRateLimitWithWeight(time.Hour*24, 12, 1),
		getBillHistoryArchiveEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getAccountConfigurationEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setPositionModeEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setLeverageEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMaximumBuyOrSellAmountEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMaximumAvailableTradableAmountEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		increaseOrDecreaseMarginEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeverageEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeverateEstimatedInfoEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTheMaximumLoanOfInstrumentEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFeeRatesEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getInterestAccruedDataEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getInterestRateEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setGreeksEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		isolatedMarginTradingSettingsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getMaximumWithdrawalsEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAccountRiskStateEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getBorrowInterestAndLimitEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		manualBorrowOrRepayEPL:               request.NewRateLimitWithWeight(threeSecondsInterval, 1, 1),
		setAutoRepayEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getBorrowRepayHistoryEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		newPositionBuilderEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		setRiskOffsetAmountEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getGreeksEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPMLimitationEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		activateOptionEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setAutoLoanEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setAccountLevelEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Sub Account Endpoints

		viewSubaccountListEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		resetSubAccountAPIKeyEPL:                          request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getSubaccountTradingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 6, 1),
		getSubaccountFundingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 6, 1),
		getSubAccountMaxWithdrawalEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		historyOfSubaccountTransferEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		managedSubAccountTransferEPL:                      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		setPermissionOfTransferOutEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getCustodyTradingSubaccountListEPL:                request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		// Grid Trading Endpoints

		gridTradingEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendGridAlgoOrderEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		stopGridAlgoOrderEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		closePositionForForContractGridEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelClosePositionOrderForContractGridEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		instantTriggerGridAlgoOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAlgoOrderListEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAlgoOrderHistoryEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAlgoOrderDetailsEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAlgoSubOrdersEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAlgoOrderPositionsEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		spotGridWithdrawIncomeEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		computeMarginBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		adjustMarginBalanceEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getGridAIParameterEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		computeMinInvestmentEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		rsiBackTestingEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Signal Bot Trading
		signalBotOrderDetailsEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		signalBotOrderPositionsEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		signalBotSubOrdersEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		signalBotEventHistoryEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Recurring Buy Order
		placeRecurringBuyOrderEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendRecurringBuyOrderEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		stopRecurringBuyOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getRecurringBuyOrderListEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getRecurringBuyOrderHistoryEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getRecurringBuyOrderDetailEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getRecurringBuySubOrdersEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getExistingLeadingPositionsEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeadingPositionHistoryEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		placeLeadingStopOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		closeLeadingPositionEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeadingInstrumentsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getProfitSharingLimitEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTotalProfitSharingEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setFirstCopySettingsEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		amendFirstCopySettingsEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		stopCopyingEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getCopySettingsEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderRanksEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderWeeklyProfitAndLossEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderDailyProfitAndLossEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderStatsEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderCurrencyPreferencesEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTraderCurrentLeadPositionsEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderLeadPositionHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Earn
		getOfferEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		purchaseEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		redeemEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		cancelPurchaseOrRedemptionEPL: request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getEarnActiveOrdersEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getFundingOrderHistoryEPL:     request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getProductInfoEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),

		// ETH Staking
		purchaseETHStakingEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		redeemETHStakingEPL:         request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getBETHBalanceEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getPurchaseRedeemHistoryEPL: request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getAPYHistoryEPL:            request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),

		// Market Data
		getTickersEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getTickerEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getPremiumHistoryEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getIndexTickersEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOrderBookEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getCandlesticksEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getCandlestickHistoryEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getIndexCandlesticksEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getIndexCandlesticksHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getMarkPriceCandlesticksHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEconomicCalendarEPL:             request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		// getIndexCandlesticksEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMyLeadTradersEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTradesRequestEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 100, 1),
		get24HTotalVolumeEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getExchangeRateRequestEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getIndexComponentsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getBlockTickersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getBlockTradesEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Spread trading related rate limiters
		placeSpreadOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelSpreadOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAllSpreadOrderEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		amendSpreadOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderDetailsEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getActiveSpreadOrdersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getSpreadOrders21DaysEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrders3MonthsEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderTradesEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadsEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderbookEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadTickerEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadPublicTradesEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadCandlesticksEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getSpreadCandlesticksHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAllSpreadOrdersAfterEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),

		// Public Data Endpoints
		getInstrumentsEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDeliveryExerciseHistoryEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getOpenInterestEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFundingEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getFundingRateHistoryEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getLimitPriceEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOptionMarketDateEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEstimatedDeliveryExercisePriceEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getDiscountRateAndInterestFreeQuotaEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getSystemTimeEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getMarkPriceEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionTiersEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getInterestRateAndLoanQuotaEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getUnderlyingEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getInsuranceFundEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		unitConvertEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		optionTickBandsEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Trading Data Endpoints

		getSupportCoinEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTakerVolumeEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getMarginLendingRatioEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLongShortRatioEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getContractsOpenInterestAndVolumeEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOptionsOpenInterestAndVolumeEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getPutCallRatioEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOpenInterestAndVolumeEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTakerFlowEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Status Endpoints

		getEventStatusEPL:                    request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		getUserAffiliateRebateInformationEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		rubikGetContractOpenInterestHistoryEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		rubikContractTakerVolumeEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		rubikTopTradersContractLongShortRatioEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		getAccountInstrumentsEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAnnouncementsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getAnnouncementTypeEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getDepositOrderDetailEPL:         request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getDepositOrderHistoryEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getWithdrawalOrderDetailEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getFiatWithdrawalOrderHistoryEPL: request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		cancelWithdrawalOrderEPL:         request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		createWithdrawalOrderEPL:         request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getWithdrawalPaymentMethodsEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getFiatDepositPaymentMethodsEPL:  request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),

		// This will fallback to the websocket connection specific rate limit
		websocketRequestEPL: request.RateLimitNotRequired,

		// Financial Product: ETH staking, SOL staking, Stable Rewards, OKUSD, Flexible loan and Dual investment
		cancelETHStakingRedemptionEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getSOLStakingProductInfoEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		purchaseSOLStakingEPL:                           request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		redeemSOLStakingEPL:                             request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getSOLStakingBalanceEPL:                         request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getSOLStakingPurchaseRedeemHistoryEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getSOLStakingAPYHistoryEPL:                      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getStableRewardsProductInfoEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getStableRewardsBalanceEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getStableRewardsAPYHistoryEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getOKUSDLimitsEPL:                               request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		subscribeOKUSDEPL:                               request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		redeemOKUSDEPL:                                  request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getOKUSDAccountEPL:                              request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getOKUSDSubscriptionHistoryEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOKUSDRedemptionHistoryEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOKUSDRewardsHistoryEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOKUSDAPRHistoryEPL:                           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanBorrowableCurrenciesEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanCollateralAssetsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanMaximumLoanAmountEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanMaximumCollateralRedeemAmountEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		adjustFlexibleLoanCollateralEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		borrowFlexibleLoanEPL:                           request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		repayFlexibleLoanEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getFlexibleLoanEModeInfoEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanInfoEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanHistoryEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFlexibleLoanAccruedInterestEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getDualInvestmentCurrencyPairsEPL:               request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getDualInvestmentProductsEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		requestDualInvestmentQuoteEPL:                   request.NewRateLimitWithWeight(time.Minute, 10, 1),
		tradeDualInvestmentEPL:                          request.NewRateLimitWithWeight(time.Minute, 2, 1),
		requestDualInvestmentRedeemQuoteEPL:             request.NewRateLimitWithWeight(time.Minute, 10, 1),
		redeemDualInvestmentEPL:                         request.NewRateLimitWithWeight(time.Minute, 2, 1),
		getDualInvestmentOrderStateEPL:                  request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getDualInvestmentOrderHistoryEPL:                request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		// Funding Account
		getAssetBillsHistoryEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getLockedTravelRuleDepositsEPL: request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		unlockTravelRuleDepositEPL:     request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getBuySellCurrenciesEPL:        request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getBuySellCurrencyPairEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getBuySellQuoteEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 10, 1),
		placeBuySellTradeEPL:           request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		getBuySellTradeHistoryEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),

		// Sub-account
		createSubAccountEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		createSubAccountAPIKeyEPL: request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getSubAccountAPIKeysEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		deleteSubAccountAPIKeyEPL: request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		// Order Book Trading > Trade
		getOneClickRepayCurrencyListV2EPL: request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		tradeOneClickRepayV2EPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getOneClickRepayHistoryV2EPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		// Copy Trading
		amendLeadingInstrumentsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getUnrealisedProfitSharingDetailsEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTotalUnrealisedProfitSharingEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		amendProfitSharingRatioEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getCopyTradingAccountConfigurationEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getCopyTradingConfigurationEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getCopyTradersEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Affiliate
		getAffiliatePerformanceSummaryEPL: request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getInviteeDetailEPL:               request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getInviteeListEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getAffiliateLinkListEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getCoInviterLinkListEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		getSubAffiliateListEPL:            request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		// Market Data, Public Data and Trading Statistics
		getRPIOrderBookEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFullOrderBookEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getCallAuctionDetailsEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEventContractSeriesEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getEventContractEventsEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getCorporateActionsEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getEventContractMarketsEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getEstimatedSettlementPriceEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getFuturesSettlementHistoryEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getMarkPriceCandlesticksEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getHistoricalMarketDataEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getMMInstrumentTypesEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getDeltaHedgeCurrenciesEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getContractLongShortAccountRatioEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTopTradersContractLongShortPositionRatioEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getOpenInterestAndVolumeStrikeEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		// Trading Account
		getBillTypesEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		setFeeTypeEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		positionBuilderTrendGraphEPL:   request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		activateFeatureEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		presetAccountModeSwitchEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		precheckAccountModeSwitchEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setCollateralAssetsEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getCollateralAssetsEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		movePositionsEPL:               request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getMovePositionsHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		setAutoEarnEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		setSettleCurrencyEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		setTradingConfigEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		precheckSetDeltaNeutralEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		increaseDemoAccountBalanceEPL:  request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1), // OKX enforces its daily quota of increases itself
		reduceDemoAccountBalanceEPL:    request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1), // OKX sets no limit on reductions
		getGLPTodayPerformanceEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getGLPHistoricalPerformanceEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getGLPV2OverviewEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getGLPV2AssessmentDetailsEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getGLPV2PerformanceHistoryEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		// Block Trading
		setRequestForQuoteMMPEPL:         request.NewRateLimitWithWeight(tenSecondsInterval, 1, 1),
		getRequestForQuoteMMPConfigEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		cancelAllQuotesAfterCountdownEPL: request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getBlockTickerEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		// Grid Trading
		amendGridAlgoOrderBasicParametersEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		addGridInvestmentEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMaxGridQuantityEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		copyGridAlgoOrderEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// DCA Trading
		placeDCAAlgoOrderEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendSpotDCABasicParametersEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		stopDCAAlgoOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDCAAlgoOrderListEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDCAAlgoOrderHistoryEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDCASubOrdersEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		addDCAInvestmentEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendDCAReinvestmentEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendDCATakeProfitEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDCAPositionDetailsEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDCACycleListEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		addDCAMarginEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		reduceDCAMarginEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Signal bot trading
		createSignalEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSignalsEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		createSignalBotEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelSignalBotsEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		adjustSignalBotMarginBalanceEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendSignalBotTakeProfitStopLossEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		setSignalBotInstrumentsEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getActiveSignalBotsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSignalBotHistoryEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSignalBotPositionHistoryEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		closeSignalBotPositionEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		placeSignalBotSubOrderEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelSignalBotSubOrderEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Recurring Buy
		amendRecurringBuyTimeEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendRecurringBuyAmountEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		addRecurringBuyInvestmentEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		pauseRecurringBuyEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		restartRecurringBuyEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendRecurringBuyPriceRangeEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Websocket connections: OKX allows 3 connection requests a second per IP, whatever the URL. Spacing them 400ms
		// apart rather than 333ms keeps four arrivals out of any one second despite network jitter.
		websocketConnectionEPL: request.NewRateLimitWithWeight(1200*time.Millisecond, 3, 1),
	}
	addOrderWeightedLimits(defs, map[request.EndpointLimit]*rate.Limiter{
		placeMultipleOrdersEPL:  placeOrders,
		cancelMultipleOrdersEPL: cancelOrders,
		amendMultipleOrdersEPL:  amendOrders,
		cancelAlgoOrderEPL:      cancelAlgos,
	})
	return defs
}

// orderBudgets holds the most orders a request carries to each endpoint limit
// OKX budgets by order rather than by request
var orderBudgets = map[request.EndpointLimit]int{
	placeMultipleOrdersEPL:  maxOrdersPerBatch,
	cancelMultipleOrdersEPL: maxOrdersPerBatch,
	amendMultipleOrdersEPL:  maxOrdersPerBatch,
	cancelAlgoOrderEPL:      maxCancelAlgosPerRequest,
}

// orderBudgetLimiter returns the limiter of an order budget OKX counts over 2
// seconds, of which a request spends a token per order, sending them all at
// once. Its burst holds one request's most orders and it refills the rest of
// the budget over the 2 seconds, so no 2 seconds carry more orders than the
// budget allows, however requests fall.
func orderBudgetLimiter(budget, maxOrders int) *rate.Limiter {
	return rate.NewLimiter(rate.Limit(float64(budget-maxOrders)/twoSecondsInterval.Seconds()), maxOrders)
}

// orderCountShift places an order count above every endpoint limit the const
// block declares, so the limits orderWeightedEPL derives are distinct from
// them and from each other
const orderCountShift = 10

// orderWeightedEPL returns the endpoint limit charging a request carrying the
// given number of orders to epl's order budget: a token per order, reserved
// together on every attempt, so a request its budget cannot cover takes none
// of it. A lone order takes epl's own limit.
func orderWeightedEPL(epl request.EndpointLimit, orders int) request.EndpointLimit {
	if orders <= 1 {
		return epl
	}
	return epl | request.EndpointLimit(orders)<<orderCountShift //nolint:gosec // Callers refuse more orders than a batch's documented maximum
}

// addOrderWeightedLimits defines the limit orderWeightedEPL derives for each
// order count of each order budget, drawing on the limiter of the budget's
// own endpoint limit
func addOrderWeightedLimits(defs request.RateLimitDefinitions, limiters map[request.EndpointLimit]*rate.Limiter) {
	for epl, maxOrders := range orderBudgets {
		for orders := request.Weight(2); int(orders) <= maxOrders; orders++ {
			defs[orderWeightedEPL(epl, int(orders))] = request.GetRateLimiterWithWeight(limiters[epl], orders)
		}
	}
}

// websocketControlLimiter returns a connection's limit on its login, subscribe
// and unsubscribe requests, which OKX caps at 480 an hour per connection: a
// burst of 20 lets a connection log in and subscribe at once, and the other
// 460 refill over the hour, so no hour carries more than 480
func websocketControlLimiter() *request.RateLimiterWithWeight {
	return request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Every(time.Hour/460), 20), 1)
}

// batchEndpointLimit returns the limit a batch place, cancel or amend request
// carrying the given number of orders is charged to: OKX charges a lone order
// sent to a batch endpoint to its single order endpoint's limit, and more
// orders to the batch endpoint's order budget
func batchEndpointLimit(single, batch request.EndpointLimit, orders int) request.EndpointLimit {
	if orders == 1 {
		return single
	}
	return orderWeightedEPL(batch, orders)
}
