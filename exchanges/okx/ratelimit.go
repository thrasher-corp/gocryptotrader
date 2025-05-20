package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	placeOrderEPL request.EndpointLimit = iota
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
	setTransactionDetail2YearIntervalEPL
	getTransactionDetailLast2YearsEPL
	cancelAllAfterCountdownEPL
	getTradeAccountRateLimitEPL
	orderPreCheckEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	amendAlgoOrderEPL
	cancelAdvanceAlgoOrderEPL
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
	createRFQEPL
	cancelRFQEPL
	cancelMultipleRFQEPL
	cancelAllRFQsEPL
	executeQuoteEPL
	getQuoteProductsEPL
	setQuoteProductsEPL
	resetRFQMMPEPL
	setMMPEPL
	getMMPConfigEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRFQsEPL
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
	lightningDepositsEPL
	getDepositAddressEPL
	getDepositHistoryEPL
	withdrawalEPL
	lightningWithdrawalsEPL
	cancelWithdrawalEPL
	getWithdrawalHistoryEPL
	getDepositWithdrawalStatusEPL
	smallAssetsConvertEPL
	getPublicExchangeListEPL
	getSavingBalanceEPL
	savingsPurchaseRedemptionEPL
	setLendingRateEPL
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
	manualBorrowAndRepayEPL
	getBorrowAndRepayHistoryEPL
	vipLoansBorrowAnsRepayEPL
	getBorrowAnsRepayHistoryHistoryEPL
	getVIPInterestAccruedDataEPL
	getVIPInterestDeductedDataEPL
	getVIPLoanOrderListEPL
	getVIPLoanOrderDetailEPL
	getBorrowInterestAndLimitEPL
	getFixedLoanBorrowLimitEPL
	getFixedLoanBorrowQuoteEPL
	placeFixedLoanBorrowingOrderEPL
	amendFixedLaonBorrowingOrderEPL
	manualRenewFixedLoanBorrowingOrderEPL
	repayFixedLoanBorrowingOrderEPL
	convertFixedLoanToMarketLoanEPL
	reduceLiabilitiesForFixedLoanEPL
	getFixedLoanBorrowOrderListEPL
	manualBorrowOrRepayEPL
	setAutoRepayEPL
	getBorrowRepayHistoryEPL
	newPositionBuilderEPL
	setRiskOffsetAmountEPL
	positionBuilderEPL
	getGreeksEPL
	getPMLimitationEPL
	setRiskOffsetLimiterEPL
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
	setSubAccountVIPLoanAllocationEPL
	getSubAccountBorrowInterestAndLimitEPL
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
	getMultipleLeveragesEPL
	setBatchLeverageEPL
	getMyLeadTradersEPL
	getLeadTraderRanksEPL
	getLeadTraderWeeklyPNLEPL
	getLeadTraderDailyPNLEPL
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
	getOrderBookLiteEPL
	getCandlesticksEPL
	getTradesRequestEPL
	get24HTotalVolumeEPL
	getOracleEPL
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
	getSpreadOrders7DaysEPL
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
	getLiquidationOrdersEPL
	getMarkPriceEPL
	getPositionTiersEPL
	getInterestRateAndLoanQuotaEPL
	getInterestRateAndLoanQuoteForVIPLoansEPL
	getUnderlyingEPL
	getInsuranceFundEPL
	unitConvertEPL
	optionTickBandsEPL
	getIndexTickerEPL
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
	getEstimatedDeliveryPriceEPL

	getAffilateInviteesDetailEPL
	getUserAffiliateRebateInformationEPL

	placeLendingOrderEPL
	amendLendingOrderEPL
	lendingOrderListEPL
	lendingSubOrderListEPL
	lendingPublicOfferEPL
	lendingAPYHistoryEPL
	lendingVolumeEPL

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
)

var rateLimits = func() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// Trade Endpoints
		placeOrderEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		placeMultipleOrdersEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		cancelOrderEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		cancelMultipleOrdersEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 300, 1),
		amendOrderEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		amendMultipleOrdersEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 4, 1),
		closePositionEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOrderDetEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getOrderListEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getOrderHistory7DaysEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getOrderHistory3MonthsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getTransactionDetail3DaysEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 60, 1),
		getTransactionDetail3MonthsEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		setTransactionDetail2YearIntervalEPL: request.NewRateLimitWithWeight(time.Hour*24, 5, 1),
		getTransactionDetailLast2YearsEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		cancelAllAfterCountdownEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getTradeAccountRateLimitEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		orderPreCheckEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		placeAlgoOrderEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAlgoOrderEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		amendAlgoOrderEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAdvanceAlgoOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderDetailEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderListEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAlgoOrderHistoryEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEasyConvertCurrencyListEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		placeEasyConvertEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getEasyConvertHistoryEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getOneClickRepayHistoryEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		oneClickRepayCurrencyListEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		tradeOneClickRepayEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		massCancelMMPOrderEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Block Trading endpoints
		getCounterpartiesEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		createRFQEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		cancelRFQEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		cancelMultipleRFQEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		cancelAllRFQsEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		executeQuoteEPL:                request.NewRateLimitWithWeight(threeSecondsInterval, 2, 1),
		getQuoteProductsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setQuoteProductsEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		resetMMPStatusEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		resetRFQMMPEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setMMPEPL:                      request.NewRateLimitWithWeight(tenSecondsInterval, 2, 1),
		getMMPConfigEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		createQuoteEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 50, 1),
		cancelQuoteEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 50, 1),
		cancelMultipleQuotesEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		cancelAllQuotesEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getRFQsEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getQuotesEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getTradesEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTradesHistoryEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		optionInstrumentTradeFamilyEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		optionTradesEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getPublicTradesEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Funding
		getCurrenciesEPL:              request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getBalanceEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getNonTradableAssetsEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getAccountAssetValuationEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		fundsTransferEPL:              request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getFundsTransferStateEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		assetBillsDetailsEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		lightningDepositsEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getDepositAddressEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getDepositHistoryEPL:          request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		withdrawalEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		lightningWithdrawalsEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		cancelWithdrawalEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getWithdrawalHistoryEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getDepositWithdrawalStatusEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		smallAssetsConvertEPL:         request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getPublicExchangeListEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getSavingBalanceEPL:           request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		savingsPurchaseRedemptionEPL:  request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		setLendingRateEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
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
		getAccountBalanceEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionsEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionsHistoryEPL:                request.NewRateLimitWithWeight(tenSecondsInterval, 1, 1),
		getAccountAndPositionRiskEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getBillsDetailsEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, 5, 1),
		getBillsDetailArchiveEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		billHistoryArchiveEPL:                 request.NewRateLimitWithWeight(time.Hour*24, 12, 1),
		getBillHistoryArchiveEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getAccountConfigurationEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setPositionModeEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setLeverageEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMaximumBuyOrSellAmountEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getMaximumAvailableTradableAmountEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		increaseOrDecreaseMarginEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeverageEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getLeverateEstimatedInfoEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getTheMaximumLoanOfInstrumentEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFeeRatesEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getInterestAccruedDataEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getInterestRateEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setGreeksEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		isolatedMarginTradingSettingsEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getMaximumWithdrawalsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getAccountRiskStateEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		manualBorrowAndRepayEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getBorrowAndRepayHistoryEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		vipLoansBorrowAnsRepayEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		getBorrowAnsRepayHistoryHistoryEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getVIPInterestAccruedDataEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getVIPInterestDeductedDataEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getVIPLoanOrderListEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getVIPLoanOrderDetailEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getBorrowInterestAndLimitEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFixedLoanBorrowLimitEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getFixedLoanBorrowQuoteEPL:            request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		placeFixedLoanBorrowingOrderEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		amendFixedLaonBorrowingOrderEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		manualRenewFixedLoanBorrowingOrderEPL: request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		repayFixedLoanBorrowingOrderEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		convertFixedLoanToMarketLoanEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		reduceLiabilitiesForFixedLoanEPL:      request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		getFixedLoanBorrowOrderListEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		manualBorrowOrRepayEPL:                request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		setAutoRepayEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getBorrowRepayHistoryEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		newPositionBuilderEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		setRiskOffsetAmountEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		positionBuilderEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getGreeksEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPMLimitationEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		setRiskOffsetLimiterEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		activateOptionEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setAutoLoanEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setAccountLevelEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),

		// Sub Account Endpoints

		viewSubaccountListEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		resetSubAccountAPIKeyEPL:                          request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getSubaccountTradingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getSubaccountFundingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getSubAccountMaxWithdrawalEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		historyOfSubaccountTransferEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		managedSubAccountTransferEPL:                      request.NewRateLimitWithWeight(oneSecondInterval, 6, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		setPermissionOfTransferOutEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		getCustodyTradingSubaccountListEPL:                request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),
		setSubAccountVIPLoanAllocationEPL:                 request.NewRateLimitWithWeight(oneSecondInterval, 5, 1),
		getSubAccountBorrowInterestAndLimitEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
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
		getMultipleLeveragesEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		setBatchLeverageEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getMyLeadTradersEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderRanksEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderWeeklyPNLEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getLeadTraderDailyPNLEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
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
		getOrderBookLiteEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, 6, 1),
		getCandlesticksEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getCandlestickHistoryEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getIndexCandlesticksEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getIndexCandlesticksHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getMarkPriceCandlesticksHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getEconomicCalendarEPL:             request.NewRateLimitWithWeight(oneSecondInterval, 5, 1),
		// getIndexCandlesticksEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEstimatedDeliveryPriceEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getTradesRequestEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, 100, 1),
		get24HTotalVolumeEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getOracleEPL:                 request.NewRateLimitWithWeight(fiveSecondsInterval, 1, 1),
		getExchangeRateRequestEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 1, 1),
		getIndexComponentsEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getBlockTickersEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getBlockTradesEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		// Spread trading related rate limiters
		placeSpreadOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelSpreadOrderEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAllSpreadOrderEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		amendSpreadOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderDetailsEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getActiveSpreadOrdersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getSpreadOrders7DaysEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderTradesEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadsEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadOrderbookEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadTickerEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadPublicTradesEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getSpreadCandlesticksEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getSpreadCandlesticksHistoryEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		cancelAllSpreadOrdersAfterEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 1, 1),

		// Public Data Endpoints
		getInstrumentsEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getDeliveryExerciseHistoryEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1),
		getOpenInterestEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFundingEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getFundingRateHistoryEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getLimitPriceEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getOptionMarketDateEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getEstimatedDeliveryExercisePriceEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getDiscountRateAndInterestFreeQuotaEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getSystemTimeEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getLiquidationOrdersEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, 40, 1), // Missing from documentation
		getMarkPriceEPL:                           request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getPositionTiersEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		getInterestRateAndLoanQuotaEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getInterestRateAndLoanQuoteForVIPLoansEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 2, 1),
		getUnderlyingEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getInsuranceFundEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		unitConvertEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, 10, 1),
		optionTickBandsEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, 5, 1),
		getIndexTickerEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

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
		getAffilateInviteesDetailEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),
		getUserAffiliateRebateInformationEPL: request.NewRateLimitWithWeight(twoSecondsInterval, 20, 1),

		placeLendingOrderEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		amendLendingOrderEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 2, 1),
		lendingOrderListEPL:    request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		lendingSubOrderListEPL: request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		lendingPublicOfferEPL:  request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		lendingAPYHistoryEPL:   request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),
		lendingVolumeEPL:       request.NewRateLimitWithWeight(oneSecondInterval, 3, 1),

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
	}
}()
