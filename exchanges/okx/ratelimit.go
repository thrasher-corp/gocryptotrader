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
	// Trade Endpoints
	placeOrderRate                  = 60
	placeMultipleOrdersRate         = 300
	cancelOrderRate                 = 60
	cancelMultipleOrdersRate        = 300
	amendOrderRate                  = 60
	amendMultipleOrdersRate         = 300
	closePositionsRate              = 20
	getOrderDetails                 = 60
	getOrderListRate                = 60
	getOrderHistory7DaysRate        = 40
	getOrderHistory3MonthsRate      = 20
	getTransactionDetail3DaysRate   = 60
	getTransactionDetail3MonthsRate = 10
	placeAlgoOrderRate              = 20
	cancelAlgoOrderRate             = 20
	cancelAdvanceAlgoOrderRate      = 20
	getAlgoOrderListRate            = 20
	getAlgoOrderHistoryRate         = 20
	getEasyConvertCurrencyListRate  = 1
	placeEasyConvert                = 1
	getEasyConvertHistory           = 1
	oneClickRepayCurrencyList       = 1
	tradeOneClickRepay              = 1
	getOneClickRepayHistory         = 1

	// Block Trading endpoints
	getCounterpartiesRate    = 5
	createRfqRate            = 5
	cancelRfqRate            = 5
	cancelMultipleRfqRate    = 2
	cancelAllRfqsRate        = 2
	executeQuoteRate         = 2
	setQuoteProducts         = 5
	restMMPStatus            = 5
	createQuoteRate          = 50
	cancelQuoteRate          = 50
	cancelMultipleQuotesRate = 2
	cancelAllQuotes          = 2
	getRfqsRate              = 2
	getQuotesRate            = 2
	getTradesRate            = 5
	getTradesHistoryRate     = 10
	getPublicTradesRate      = 5

	// Funding
	getCurrenciesRate            = 6
	getBalanceRate               = 6
	getAccountAssetValuationRate = 1
	fundsTransferRate            = 1
	getFundsTransferStateRate    = 1
	assetBillsDetailsRate        = 6
	lightningDepositsRate        = 2
	getDepositAddressRate        = 6
	getDepositHistoryRate        = 6
	withdrawalRate               = 6
	lightningWithdrawalsRate     = 2
	cancelWithdrawalRate         = 6
	getWithdrawalHistoryRate     = 6
	smallAssetsConvertRate       = 1

	// Savings
	getSavingBalanceRate          = 6
	savingsPurchaseRedemptionRate = 6
	setLendingRateRate            = 6
	getLendingHistoryRate         = 6
	getPublicBorrowInfoRate       = 6
	getPublicBorrowHistoryRate    = 6

	// Convert
	getConvertCurrenciesRate   = 6
	getConvertCurrencyPairRate = 6
	estimateQuoteRate          = 10
	convertTradeRate           = 10
	getConvertHistoryRate      = 6

	// Account
	getAccountBalanceRate                 = 10
	getPositionsRate                      = 10
	getPositionsHistoryRate               = 1
	getAccountAndPositionRiskRate         = 10
	getBillsDetailsRate                   = 6
	getAccountConfigurationRate           = 5
	setPositionModeRate                   = 5
	setLeverageRate                       = 20
	getMaximumBuyOrSellAmountRate         = 20
	getMaximumAvailableTradableAmountRate = 20
	increaseOrDecreaseMarginRate          = 20
	getLeverageRate                       = 20
	getTheMaximumLoanOfInstrumentRate     = 20
	getFeeRatesRate                       = 5
	getInterestAccruedDataRate            = 5
	getInterestRateRate                   = 5
	setGreeksRate                         = 5
	isolatedMarginTradingSettingsRate     = 5
	getMaximumWithdrawalsRate             = 20
	getAccountRiskStateRate               = 10
	vipLoansBorrowAndRepayRate            = 6
	getBorrowAnsRepayHistoryHistoryRate   = 5
	getBorrowInterestAndLimitRate         = 5
	positionBuilderRate                   = 2
	getGreeksRate                         = 10
	getPMLimitation                       = 10

	// Sub Account Endpoints
	viewSubaccountListRate                             = 2
	resetSubAccountAPIKey                              = 1
	getSubaccountTradingBalanceRate                    = 2
	getSubaccountFundingBalanceRate                    = 2
	historyOfSubaccountTransferRate                    = 6
	masterAccountsManageTransfersBetweenSubaccountRate = 1
	setPermissionOfTransferOutRate                     = 1
	getCustodyTradingSubaccountListRate                = 1
	gridTradingRate                                    = 20
	amendGridAlgoOrderRate                             = 20
	stopGridAlgoOrderRate                              = 20
	getGridAlgoOrderListRate                           = 20
	getGridAlgoOrderHistoryRate                        = 20
	getGridAlgoOrderDetailsRate                        = 20
	getGridAlgoSubOrdersRate                           = 20
	getGridAlgoOrderPositionsRate                      = 20
	spotGridWithdrawIncomeRate                         = 20
	computeMarginBalance                               = 20
	adjustMarginBalance                                = 20
	getGridAIParameter                                 = 20

	// Earn
	getOffer                   = 3
	purchase                   = 2
	redeem                     = 2
	cancelPurchaseOrRedemption = 2
	getEarnActiveOrders        = 3
	getFundingOrderHistory     = 3

	// Market Data
	getTickersRate               = 20
	getIndexTickersRate          = 20
	getOrderBookRate             = 20
	getCandlesticksRate          = 40
	getCandlesticksHistoryRate   = 20
	getIndexCandlesticksRate     = 20
	getMarkPriceCandlesticksRate = 20
	getTradesRequestRate         = 100
	get24HTotalVolumeRate        = 2
	getOracleRate                = 1
	getExchangeRateRequestRate   = 1
	getIndexComponentsRate       = 20
	getBlockTickersRate          = 20
	getBlockTradesRate           = 20

	// Public Data Endpoints
	getInstrumentsRate                         = 20
	getDeliveryExerciseHistoryRate             = 40
	getOpenInterestRate                        = 20
	getFundingRate                             = 20
	getFundingRateHistoryRate                  = 20
	getLimitPriceRate                          = 20
	getOptionMarketDateRate                    = 20
	getEstimatedDeliveryExercisePriceRate      = 10
	getDiscountRateAndInterestFreeQuotaRate    = 2
	getSystemTimeRate                          = 10
	getLiquidationOrdersRate                   = 40
	getMarkPriceRate                           = 10
	getPositionTiersRate                       = 10
	getInterestRateAndLoanQuotaRate            = 2
	getInterestRateAndLoanQuoteForVIPLoansRate = 2
	getUnderlyingRate                          = 20
	getInsuranceFundRate                       = 10
	unitConvertRate                            = 10

	// Trading Data Endpoints
	getSupportCoinRate                    = 5
	getTakerVolumeRate                    = 5
	getMarginLendingRatioRate             = 5
	getLongShortRatioRate                 = 5
	getContractsOpenInterestAndVolumeRate = 5
	getOptionsOpenInterestAndVolumeRate   = 5
	getPutCallRatioRate                   = 5
	getOpenInterestAndVolumeRate          = 5
	getTakerFlowRate                      = 5

	// Status Endpoints
	getEventStatusRate = 1
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
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	cancelAdvanceAlgoOrderEPL
	getAlgoOrderListEPL
	getAlgoOrderHistoryEPL
	getEasyConvertCurrencyListEPL
	placeEasyConvertEPL
	getEasyConvertHistoryEPL
	getOneClickRepayHistoryEPL
	oneClickRepayCurrencyListEPL
	tradeOneClickRepayEPL
	getCounterpartiesEPL
	createRfqEPL
	cancelRfqEPL
	cancelMultipleRfqEPL
	cancelAllRfqsEPL
	executeQuoteEPL
	setQuoteProductsEPL
	restMMPStatusEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRfqsEPL
	getQuotesEPL
	getTradesEPL
	getTradesHistoryEPL
	getPublicTradesEPL
	getCurrenciesEPL
	getBalanceEPL
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
	smallAssetsConvertEPL
	getSavingBalanceEPL
	savingsPurchaseRedemptionEPL
	setLendingRateEPL
	getLendingHistoryEPL
	getPublicBorrowInfoEPL
	getPublicBorrowHistoryEPL
	getConvertCurrenciesEPL
	getConvertCurrencyPairEPL
	estimateQuoteEPL
	convertTradeEPL
	getConvertHistoryEPL
	getAccountBalanceEPL
	getPositionsEPL
	getPositionsHistoryEPL
	getAccountAndPositionRiskEPL
	getBillsDetailsEPL
	getAccountConfigurationEPL
	setPositionModeEPL
	setLeverageEPL
	getMaximumBuyOrSellAmountEPL
	getMaximumAvailableTradableAmountEPL
	increaseOrDecreaseMarginEPL
	getLeverageEPL
	getTheMaximumLoanOfInstrumentEPL
	getFeeRatesEPL
	getInterestAccruedDataEPL
	getInterestRateEPL
	setGreeksEPL
	isolatedMarginTradingSettingsEPL
	getMaximumWithdrawalsEPL
	getAccountRiskStateEPL
	vipLoansBorrowAnsRepayEPL
	getBorrowAnsRepayHistoryHistoryEPL
	getBorrowInterestAndLimitEPL
	positionBuilderEPL
	getGreeksEPL
	getPMLimitationEPL
	viewSubaccountListEPL
	resetSubAccountAPIKeyEPL
	getSubaccountTradingBalanceEPL
	getSubaccountFundingBalanceEPL
	historyOfSubaccountTransferEPL
	masterAccountsManageTransfersBetweenSubaccountEPL
	setPermissionOfTransferOutEPL
	getCustodyTradingSubaccountListEPL
	gridTradingEPL
	amendGridAlgoOrderEPL
	stopGridAlgoOrderEPL
	getGridAlgoOrderListEPL
	getGridAlgoOrderHistoryEPL
	getGridAlgoOrderDetailsEPL
	getGridAlgoSubOrdersEPL
	getGridAlgoOrderPositionsEPL
	spotGridWithdrawIncomeEPL
	computeMarginBalanceEPL
	adjustMarginBalanceEPL
	getGridAIParameterEPL
	getOfferEPL
	purchaseEPL
	redeemEPL
	cancelPurchaseOrRedemptionEPL
	getEarnActiveOrdersEPL
	getFundingOrderHistoryEPL
	getTickersEPL
	getIndexTickersEPL
	getOrderBookEPL
	getCandlestickEPL
	getTradesRequestEPL
	get24HTotalVolumeEPL
	getOracleEPL
	getExchangeRateRequestEPL
	getIndexComponentsEPL
	getBlockTickersEPL
	getBlockTradesEPL
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
	getIndexCandlestickEPL
)

// GetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// Trade Endpoints
		placeOrderEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, placeOrderRate, 1),
		placeMultipleOrdersEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, placeMultipleOrdersRate, 1),
		cancelOrderEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, cancelOrderRate, 1),
		cancelMultipleOrdersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, cancelMultipleOrdersRate, 1),
		amendOrderEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, amendOrderRate, 1),
		amendMultipleOrdersEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, amendMultipleOrdersRate, 1),
		closePositionEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, closePositionsRate, 1),
		getOrderDetEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, getOrderDetails, 1),
		getOrderListEPL:                request.NewRateLimitWithWeight(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistory7DaysEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, getOrderHistory7DaysRate, 1),
		getOrderHistory3MonthsEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, getOrderHistory3MonthsRate, 1),
		getTransactionDetail3DaysEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, getTransactionDetail3DaysRate, 1),
		getTransactionDetail3MonthsEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getTransactionDetail3MonthsRate, 1),
		placeAlgoOrderEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvanceAlgoOrderEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, cancelAdvanceAlgoOrderRate, 1),
		getAlgoOrderListEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getEasyConvertCurrencyListEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, getEasyConvertCurrencyListRate, 1),
		placeEasyConvertEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, placeEasyConvert, 1),
		getEasyConvertHistoryEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, getEasyConvertHistory, 1),
		getOneClickRepayHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, getOneClickRepayHistory, 1),
		oneClickRepayCurrencyListEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, oneClickRepayCurrencyList, 1),
		tradeOneClickRepayEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, tradeOneClickRepay, 1),

		// Block Trading endpoints
		getCounterpartiesEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, getCounterpartiesRate, 1),
		createRfqEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, createRfqRate, 1),
		cancelRfqEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, cancelRfqRate, 1),
		cancelMultipleRfqEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, cancelMultipleRfqRate, 1),
		cancelAllRfqsEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, cancelAllRfqsRate, 1),
		executeQuoteEPL:         request.NewRateLimitWithWeight(threeSecondsInterval, executeQuoteRate, 1),
		setQuoteProductsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, setQuoteProducts, 1),
		restMMPStatusEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, restMMPStatus, 1),
		createQuoteEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, createQuoteRate, 1),
		cancelQuoteEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, cancelQuoteRate, 1),
		cancelMultipleQuotesEPL: request.NewRateLimitWithWeight(twoSecondsInterval, cancelMultipleQuotesRate, 1),
		cancelAllQuotesEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, cancelAllQuotes, 1),
		getRfqsEPL:              request.NewRateLimitWithWeight(twoSecondsInterval, getRfqsRate, 1),
		getQuotesEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, getQuotesRate, 1),
		getTradesEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, getTradesRate, 1),
		getTradesHistoryEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, getTradesHistoryRate, 1),
		getPublicTradesEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, getPublicTradesRate, 1),
		// Funding
		getCurrenciesEPL:             request.NewRateLimitWithWeight(oneSecondInterval, getCurrenciesRate, 1),
		getBalanceEPL:                request.NewRateLimitWithWeight(oneSecondInterval, getBalanceRate, 1),
		getAccountAssetValuationEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundsTransferEPL:             request.NewRateLimitWithWeight(oneSecondInterval, fundsTransferRate, 1),
		getFundsTransferStateEPL:     request.NewRateLimitWithWeight(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailsEPL:         request.NewRateLimitWithWeight(oneSecondInterval, assetBillsDetailsRate, 1),
		lightningDepositsEPL:         request.NewRateLimitWithWeight(oneSecondInterval, lightningDepositsRate, 1),
		getDepositAddressEPL:         request.NewRateLimitWithWeight(oneSecondInterval, getDepositAddressRate, 1),
		getDepositHistoryEPL:         request.NewRateLimitWithWeight(oneSecondInterval, getDepositHistoryRate, 1),
		withdrawalEPL:                request.NewRateLimitWithWeight(oneSecondInterval, withdrawalRate, 1),
		lightningWithdrawalsEPL:      request.NewRateLimitWithWeight(oneSecondInterval, lightningWithdrawalsRate, 1),
		cancelWithdrawalEPL:          request.NewRateLimitWithWeight(oneSecondInterval, cancelWithdrawalRate, 1),
		getWithdrawalHistoryEPL:      request.NewRateLimitWithWeight(oneSecondInterval, getWithdrawalHistoryRate, 1),
		smallAssetsConvertEPL:        request.NewRateLimitWithWeight(oneSecondInterval, smallAssetsConvertRate, 1),
		getSavingBalanceEPL:          request.NewRateLimitWithWeight(oneSecondInterval, getSavingBalanceRate, 1),
		savingsPurchaseRedemptionEPL: request.NewRateLimitWithWeight(oneSecondInterval, savingsPurchaseRedemptionRate, 1),
		setLendingRateEPL:            request.NewRateLimitWithWeight(oneSecondInterval, setLendingRateRate, 1),
		getLendingHistoryEPL:         request.NewRateLimitWithWeight(oneSecondInterval, getLendingHistoryRate, 1),
		getPublicBorrowInfoEPL:       request.NewRateLimitWithWeight(oneSecondInterval, getPublicBorrowInfoRate, 1),
		getPublicBorrowHistoryEPL:    request.NewRateLimitWithWeight(oneSecondInterval, getPublicBorrowHistoryRate, 1),

		// Convert
		getConvertCurrenciesEPL:   request.NewRateLimitWithWeight(oneSecondInterval, getConvertCurrenciesRate, 1),
		getConvertCurrencyPairEPL: request.NewRateLimitWithWeight(oneSecondInterval, getConvertCurrencyPairRate, 1),
		estimateQuoteEPL:          request.NewRateLimitWithWeight(oneSecondInterval, estimateQuoteRate, 1),
		convertTradeEPL:           request.NewRateLimitWithWeight(oneSecondInterval, convertTradeRate, 1),
		getConvertHistoryEPL:      request.NewRateLimitWithWeight(oneSecondInterval, getConvertHistoryRate, 1),

		// Account
		getAccountBalanceEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, getAccountBalanceRate, 1),
		getPositionsEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, getPositionsRate, 1),
		getPositionsHistoryEPL:               request.NewRateLimitWithWeight(tenSecondsInterval, getPositionsHistoryRate, 1),
		getAccountAndPositionRiskEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getAccountAndPositionRiskRate, 1),
		getBillsDetailsEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, getBillsDetailsRate, 1),
		getAccountConfigurationEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, getAccountConfigurationRate, 1),
		setPositionModeEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, setPositionModeRate, 1),
		setLeverageEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, setLeverageRate, 1),
		getMaximumBuyOrSellAmountEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getMaximumBuyOrSellAmountRate, 1),
		getMaximumAvailableTradableAmountEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getMaximumAvailableTradableAmountRate, 1),
		increaseOrDecreaseMarginEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, increaseOrDecreaseMarginRate, 1),
		getLeverageEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, getLeverageRate, 1),
		getTheMaximumLoanOfInstrumentEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, getTheMaximumLoanOfInstrumentRate, 1),
		getFeeRatesEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, getFeeRatesRate, 1),
		getInterestAccruedDataEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, getInterestAccruedDataRate, 1),
		getInterestRateEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, getInterestRateRate, 1),
		setGreeksEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, setGreeksRate, 1),
		isolatedMarginTradingSettingsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, isolatedMarginTradingSettingsRate, 1),
		getMaximumWithdrawalsEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, getMaximumWithdrawalsRate, 1),
		getAccountRiskStateEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, getAccountRiskStateRate, 1),
		vipLoansBorrowAnsRepayEPL:            request.NewRateLimitWithWeight(oneSecondInterval, vipLoansBorrowAndRepayRate, 1),
		getBorrowAnsRepayHistoryHistoryEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, getBorrowAnsRepayHistoryHistoryRate, 1),
		getBorrowInterestAndLimitEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getBorrowInterestAndLimitRate, 1),
		positionBuilderEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, positionBuilderRate, 1),
		getGreeksEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, getGreeksRate, 1),
		getPMLimitationEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, getPMLimitation, 1),

		// Sub Account Endpoints
		viewSubaccountListEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, viewSubaccountListRate, 1),
		resetSubAccountAPIKeyEPL:                          request.NewRateLimitWithWeight(oneSecondInterval, resetSubAccountAPIKey, 1),
		getSubaccountTradingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getSubaccountTradingBalanceRate, 1),
		getSubaccountFundingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getSubaccountFundingBalanceRate, 1),
		historyOfSubaccountTransferEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, historyOfSubaccountTransferRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithWeight(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		setPermissionOfTransferOutEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, setPermissionOfTransferOutRate, 1),
		getCustodyTradingSubaccountListEPL:                request.NewRateLimitWithWeight(oneSecondInterval, getCustodyTradingSubaccountListRate, 1),

		// Grid Trading Endpoints
		gridTradingEPL:               request.NewRateLimitWithWeight(twoSecondsInterval, gridTradingRate, 1),
		amendGridAlgoOrderEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, amendGridAlgoOrderRate, 1),
		stopGridAlgoOrderEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, stopGridAlgoOrderRate, 1),
		getGridAlgoOrderListEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, getGridAlgoOrderListRate, 1),
		getGridAlgoOrderHistoryEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, getGridAlgoOrderHistoryRate, 1),
		getGridAlgoOrderDetailsEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, getGridAlgoOrderDetailsRate, 1),
		getGridAlgoSubOrdersEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, getGridAlgoSubOrdersRate, 1),
		getGridAlgoOrderPositionsEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getGridAlgoOrderPositionsRate, 1),
		spotGridWithdrawIncomeEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, spotGridWithdrawIncomeRate, 1),
		computeMarginBalanceEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, computeMarginBalance, 1),
		adjustMarginBalanceEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, adjustMarginBalance, 1),
		getGridAIParameterEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, getGridAIParameter, 1),

		// Earn
		getOfferEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, getOffer, 1),
		purchaseEPL:                   request.NewRateLimitWithWeight(oneSecondInterval, purchase, 1),
		redeemEPL:                     request.NewRateLimitWithWeight(oneSecondInterval, redeem, 1),
		cancelPurchaseOrRedemptionEPL: request.NewRateLimitWithWeight(oneSecondInterval, cancelPurchaseOrRedemption, 1),
		getEarnActiveOrdersEPL:        request.NewRateLimitWithWeight(oneSecondInterval, getEarnActiveOrders, 1),
		getFundingOrderHistoryEPL:     request.NewRateLimitWithWeight(oneSecondInterval, getFundingOrderHistory, 1),

		// Market Data
		getTickersEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, getTickersRate, 1),
		getIndexTickersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, getIndexTickersRate, 1),
		getOrderBookEPL:           request.NewRateLimitWithWeight(twoSecondsInterval, getOrderBookRate, 1),
		getCandlestickEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:  request.NewRateLimitWithWeight(twoSecondsInterval, getCandlesticksHistoryRate, 1),
		getIndexCandlestickEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, getIndexCandlesticksRate, 1),
		getTradesRequestEPL:       request.NewRateLimitWithWeight(twoSecondsInterval, getTradesRequestRate, 1),
		get24HTotalVolumeEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, get24HTotalVolumeRate, 1),
		getOracleEPL:              request.NewRateLimitWithWeight(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateRequestEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getExchangeRateRequestRate, 1),
		getIndexComponentsEPL:     request.NewRateLimitWithWeight(twoSecondsInterval, getIndexComponentsRate, 1),
		getBlockTickersEPL:        request.NewRateLimitWithWeight(twoSecondsInterval, getBlockTickersRate, 1),
		getBlockTradesEPL:         request.NewRateLimitWithWeight(twoSecondsInterval, getBlockTradesRate, 1),

		// Public Data Endpoints
		getInstrumentsEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, getInstrumentsRate, 1),
		getDeliveryExerciseHistoryEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, getDeliveryExerciseHistoryRate, 1),
		getOpenInterestEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, getOpenInterestRate, 1),
		getFundingEPL:                             request.NewRateLimitWithWeight(twoSecondsInterval, getFundingRate, 1),
		getFundingRateHistoryEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, getFundingRateHistoryRate, 1),
		getLimitPriceEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, getLimitPriceRate, 1),
		getOptionMarketDateEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getOptionMarketDateRate, 1),
		getEstimatedDeliveryExercisePriceEPL:      request.NewRateLimitWithWeight(twoSecondsInterval, getEstimatedDeliveryExercisePriceRate, 1),
		getDiscountRateAndInterestFreeQuotaEPL:    request.NewRateLimitWithWeight(twoSecondsInterval, getDiscountRateAndInterestFreeQuotaRate, 1),
		getSystemTimeEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, getSystemTimeRate, 1),
		getLiquidationOrdersEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, getLiquidationOrdersRate, 1),
		getMarkPriceEPL:                           request.NewRateLimitWithWeight(twoSecondsInterval, getMarkPriceRate, 1),
		getPositionTiersEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, getPositionTiersRate, 1),
		getInterestRateAndLoanQuotaEPL:            request.NewRateLimitWithWeight(twoSecondsInterval, getInterestRateAndLoanQuotaRate, 1),
		getInterestRateAndLoanQuoteForVIPLoansEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getInterestRateAndLoanQuoteForVIPLoansRate, 1),
		getUnderlyingEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, getUnderlyingRate, 1),
		getInsuranceFundEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, getInsuranceFundRate, 1),
		unitConvertEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, unitConvertRate, 1),

		// Trading Data Endpoints
		getSupportCoinEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getSupportCoinRate, 1),
		getTakerVolumeEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getTakerVolumeRate, 1),
		getMarginLendingRatioEPL:             request.NewRateLimitWithWeight(twoSecondsInterval, getMarginLendingRatioRate, 1),
		getLongShortRatioEPL:                 request.NewRateLimitWithWeight(twoSecondsInterval, getLongShortRatioRate, 1),
		getContractsOpenInterestAndVolumeEPL: request.NewRateLimitWithWeight(twoSecondsInterval, getContractsOpenInterestAndVolumeRate, 1),
		getOptionsOpenInterestAndVolumeEPL:   request.NewRateLimitWithWeight(twoSecondsInterval, getOptionsOpenInterestAndVolumeRate, 1),
		getPutCallRatioEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, getPutCallRatioRate, 1),
		getOpenInterestAndVolumeEPL:          request.NewRateLimitWithWeight(twoSecondsInterval, getOpenInterestAndVolumeRate, 1),
		getTakerFlowEPL:                      request.NewRateLimitWithWeight(twoSecondsInterval, getTakerFlowRate, 1),

		// Status Endpoints
		getEventStatusEPL: request.NewRateLimitWithWeight(fiveSecondsInterval, getEventStatusRate, 1),
	}
}
