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

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// Trade Endpoints
		placeOrderEPL:                  request.NewRateLimitWithToken(twoSecondsInterval, placeOrderRate, 1),
		placeMultipleOrdersEPL:         request.NewRateLimitWithToken(twoSecondsInterval, placeMultipleOrdersRate, 1),
		cancelOrderEPL:                 request.NewRateLimitWithToken(twoSecondsInterval, cancelOrderRate, 1),
		cancelMultipleOrdersEPL:        request.NewRateLimitWithToken(twoSecondsInterval, cancelMultipleOrdersRate, 1),
		amendOrderEPL:                  request.NewRateLimitWithToken(twoSecondsInterval, amendOrderRate, 1),
		amendMultipleOrdersEPL:         request.NewRateLimitWithToken(twoSecondsInterval, amendMultipleOrdersRate, 1),
		closePositionEPL:               request.NewRateLimitWithToken(twoSecondsInterval, closePositionsRate, 1),
		getOrderDetEPL:                 request.NewRateLimitWithToken(twoSecondsInterval, getOrderDetails, 1),
		getOrderListEPL:                request.NewRateLimitWithToken(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistory7DaysEPL:        request.NewRateLimitWithToken(twoSecondsInterval, getOrderHistory7DaysRate, 1),
		getOrderHistory3MonthsEPL:      request.NewRateLimitWithToken(twoSecondsInterval, getOrderHistory3MonthsRate, 1),
		getTransactionDetail3DaysEPL:   request.NewRateLimitWithToken(twoSecondsInterval, getTransactionDetail3DaysRate, 1),
		getTransactionDetail3MonthsEPL: request.NewRateLimitWithToken(twoSecondsInterval, getTransactionDetail3MonthsRate, 1),
		placeAlgoOrderEPL:              request.NewRateLimitWithToken(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:             request.NewRateLimitWithToken(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvanceAlgoOrderEPL:      request.NewRateLimitWithToken(twoSecondsInterval, cancelAdvanceAlgoOrderRate, 1),
		getAlgoOrderListEPL:            request.NewRateLimitWithToken(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getEasyConvertCurrencyListEPL:  request.NewRateLimitWithToken(twoSecondsInterval, getEasyConvertCurrencyListRate, 1),
		placeEasyConvertEPL:            request.NewRateLimitWithToken(twoSecondsInterval, placeEasyConvert, 1),
		getEasyConvertHistoryEPL:       request.NewRateLimitWithToken(twoSecondsInterval, getEasyConvertHistory, 1),
		getOneClickRepayHistoryEPL:     request.NewRateLimitWithToken(twoSecondsInterval, getOneClickRepayHistory, 1),
		oneClickRepayCurrencyListEPL:   request.NewRateLimitWithToken(twoSecondsInterval, oneClickRepayCurrencyList, 1),
		tradeOneClickRepayEPL:          request.NewRateLimitWithToken(twoSecondsInterval, tradeOneClickRepay, 1),

		// Block Trading endpoints
		getCounterpartiesEPL:    request.NewRateLimitWithToken(twoSecondsInterval, getCounterpartiesRate, 1),
		createRfqEPL:            request.NewRateLimitWithToken(twoSecondsInterval, createRfqRate, 1),
		cancelRfqEPL:            request.NewRateLimitWithToken(twoSecondsInterval, cancelRfqRate, 1),
		cancelMultipleRfqEPL:    request.NewRateLimitWithToken(twoSecondsInterval, cancelMultipleRfqRate, 1),
		cancelAllRfqsEPL:        request.NewRateLimitWithToken(twoSecondsInterval, cancelAllRfqsRate, 1),
		executeQuoteEPL:         request.NewRateLimitWithToken(threeSecondsInterval, executeQuoteRate, 1),
		setQuoteProductsEPL:     request.NewRateLimitWithToken(twoSecondsInterval, setQuoteProducts, 1),
		restMMPStatusEPL:        request.NewRateLimitWithToken(twoSecondsInterval, restMMPStatus, 1),
		createQuoteEPL:          request.NewRateLimitWithToken(twoSecondsInterval, createQuoteRate, 1),
		cancelQuoteEPL:          request.NewRateLimitWithToken(twoSecondsInterval, cancelQuoteRate, 1),
		cancelMultipleQuotesEPL: request.NewRateLimitWithToken(twoSecondsInterval, cancelMultipleQuotesRate, 1),
		cancelAllQuotesEPL:      request.NewRateLimitWithToken(twoSecondsInterval, cancelAllQuotes, 1),
		getRfqsEPL:              request.NewRateLimitWithToken(twoSecondsInterval, getRfqsRate, 1),
		getQuotesEPL:            request.NewRateLimitWithToken(twoSecondsInterval, getQuotesRate, 1),
		getTradesEPL:            request.NewRateLimitWithToken(twoSecondsInterval, getTradesRate, 1),
		getTradesHistoryEPL:     request.NewRateLimitWithToken(twoSecondsInterval, getTradesHistoryRate, 1),
		getPublicTradesEPL:      request.NewRateLimitWithToken(twoSecondsInterval, getPublicTradesRate, 1),
		// Funding
		getCurrenciesEPL:             request.NewRateLimitWithToken(oneSecondInterval, getCurrenciesRate, 1),
		getBalanceEPL:                request.NewRateLimitWithToken(oneSecondInterval, getBalanceRate, 1),
		getAccountAssetValuationEPL:  request.NewRateLimitWithToken(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundsTransferEPL:             request.NewRateLimitWithToken(oneSecondInterval, fundsTransferRate, 1),
		getFundsTransferStateEPL:     request.NewRateLimitWithToken(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailsEPL:         request.NewRateLimitWithToken(oneSecondInterval, assetBillsDetailsRate, 1),
		lightningDepositsEPL:         request.NewRateLimitWithToken(oneSecondInterval, lightningDepositsRate, 1),
		getDepositAddressEPL:         request.NewRateLimitWithToken(oneSecondInterval, getDepositAddressRate, 1),
		getDepositHistoryEPL:         request.NewRateLimitWithToken(oneSecondInterval, getDepositHistoryRate, 1),
		withdrawalEPL:                request.NewRateLimitWithToken(oneSecondInterval, withdrawalRate, 1),
		lightningWithdrawalsEPL:      request.NewRateLimitWithToken(oneSecondInterval, lightningWithdrawalsRate, 1),
		cancelWithdrawalEPL:          request.NewRateLimitWithToken(oneSecondInterval, cancelWithdrawalRate, 1),
		getWithdrawalHistoryEPL:      request.NewRateLimitWithToken(oneSecondInterval, getWithdrawalHistoryRate, 1),
		smallAssetsConvertEPL:        request.NewRateLimitWithToken(oneSecondInterval, smallAssetsConvertRate, 1),
		getSavingBalanceEPL:          request.NewRateLimitWithToken(oneSecondInterval, getSavingBalanceRate, 1),
		savingsPurchaseRedemptionEPL: request.NewRateLimitWithToken(oneSecondInterval, savingsPurchaseRedemptionRate, 1),
		setLendingRateEPL:            request.NewRateLimitWithToken(oneSecondInterval, setLendingRateRate, 1),
		getLendingHistoryEPL:         request.NewRateLimitWithToken(oneSecondInterval, getLendingHistoryRate, 1),
		getPublicBorrowInfoEPL:       request.NewRateLimitWithToken(oneSecondInterval, getPublicBorrowInfoRate, 1),
		getPublicBorrowHistoryEPL:    request.NewRateLimitWithToken(oneSecondInterval, getPublicBorrowHistoryRate, 1),

		// Convert
		getConvertCurrenciesEPL:   request.NewRateLimitWithToken(oneSecondInterval, getConvertCurrenciesRate, 1),
		getConvertCurrencyPairEPL: request.NewRateLimitWithToken(oneSecondInterval, getConvertCurrencyPairRate, 1),
		estimateQuoteEPL:          request.NewRateLimitWithToken(oneSecondInterval, estimateQuoteRate, 1),
		convertTradeEPL:           request.NewRateLimitWithToken(oneSecondInterval, convertTradeRate, 1),
		getConvertHistoryEPL:      request.NewRateLimitWithToken(oneSecondInterval, getConvertHistoryRate, 1),

		// Account
		getAccountBalanceEPL:                 request.NewRateLimitWithToken(twoSecondsInterval, getAccountBalanceRate, 1),
		getPositionsEPL:                      request.NewRateLimitWithToken(twoSecondsInterval, getPositionsRate, 1),
		getPositionsHistoryEPL:               request.NewRateLimitWithToken(tenSecondsInterval, getPositionsHistoryRate, 1),
		getAccountAndPositionRiskEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getAccountAndPositionRiskRate, 1),
		getBillsDetailsEPL:                   request.NewRateLimitWithToken(oneSecondInterval, getBillsDetailsRate, 1),
		getAccountConfigurationEPL:           request.NewRateLimitWithToken(twoSecondsInterval, getAccountConfigurationRate, 1),
		setPositionModeEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, setPositionModeRate, 1),
		setLeverageEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, setLeverageRate, 1),
		getMaximumBuyOrSellAmountEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getMaximumBuyOrSellAmountRate, 1),
		getMaximumAvailableTradableAmountEPL: request.NewRateLimitWithToken(twoSecondsInterval, getMaximumAvailableTradableAmountRate, 1),
		increaseOrDecreaseMarginEPL:          request.NewRateLimitWithToken(twoSecondsInterval, increaseOrDecreaseMarginRate, 1),
		getLeverageEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, getLeverageRate, 1),
		getTheMaximumLoanOfInstrumentEPL:     request.NewRateLimitWithToken(twoSecondsInterval, getTheMaximumLoanOfInstrumentRate, 1),
		getFeeRatesEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, getFeeRatesRate, 1),
		getInterestAccruedDataEPL:            request.NewRateLimitWithToken(twoSecondsInterval, getInterestAccruedDataRate, 1),
		getInterestRateEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, getInterestRateRate, 1),
		setGreeksEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, setGreeksRate, 1),
		isolatedMarginTradingSettingsEPL:     request.NewRateLimitWithToken(twoSecondsInterval, isolatedMarginTradingSettingsRate, 1),
		getMaximumWithdrawalsEPL:             request.NewRateLimitWithToken(twoSecondsInterval, getMaximumWithdrawalsRate, 1),
		getAccountRiskStateEPL:               request.NewRateLimitWithToken(twoSecondsInterval, getAccountRiskStateRate, 1),
		vipLoansBorrowAnsRepayEPL:            request.NewRateLimitWithToken(oneSecondInterval, vipLoansBorrowAndRepayRate, 1),
		getBorrowAnsRepayHistoryHistoryEPL:   request.NewRateLimitWithToken(twoSecondsInterval, getBorrowAnsRepayHistoryHistoryRate, 1),
		getBorrowInterestAndLimitEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getBorrowInterestAndLimitRate, 1),
		positionBuilderEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, positionBuilderRate, 1),
		getGreeksEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, getGreeksRate, 1),
		getPMLimitationEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, getPMLimitation, 1),

		// Sub Account Endpoints
		viewSubaccountListEPL:                             request.NewRateLimitWithToken(twoSecondsInterval, viewSubaccountListRate, 1),
		resetSubAccountAPIKeyEPL:                          request.NewRateLimitWithToken(oneSecondInterval, resetSubAccountAPIKey, 1),
		getSubaccountTradingBalanceEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getSubaccountTradingBalanceRate, 1),
		getSubaccountFundingBalanceEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getSubaccountFundingBalanceRate, 1),
		historyOfSubaccountTransferEPL:                    request.NewRateLimitWithToken(oneSecondInterval, historyOfSubaccountTransferRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithToken(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		setPermissionOfTransferOutEPL:                     request.NewRateLimitWithToken(oneSecondInterval, setPermissionOfTransferOutRate, 1),
		getCustodyTradingSubaccountListEPL:                request.NewRateLimitWithToken(oneSecondInterval, getCustodyTradingSubaccountListRate, 1),

		// Grid Trading Endpoints
		gridTradingEPL:               request.NewRateLimitWithToken(twoSecondsInterval, gridTradingRate, 1),
		amendGridAlgoOrderEPL:        request.NewRateLimitWithToken(twoSecondsInterval, amendGridAlgoOrderRate, 1),
		stopGridAlgoOrderEPL:         request.NewRateLimitWithToken(twoSecondsInterval, stopGridAlgoOrderRate, 1),
		getGridAlgoOrderListEPL:      request.NewRateLimitWithToken(twoSecondsInterval, getGridAlgoOrderListRate, 1),
		getGridAlgoOrderHistoryEPL:   request.NewRateLimitWithToken(twoSecondsInterval, getGridAlgoOrderHistoryRate, 1),
		getGridAlgoOrderDetailsEPL:   request.NewRateLimitWithToken(twoSecondsInterval, getGridAlgoOrderDetailsRate, 1),
		getGridAlgoSubOrdersEPL:      request.NewRateLimitWithToken(twoSecondsInterval, getGridAlgoSubOrdersRate, 1),
		getGridAlgoOrderPositionsEPL: request.NewRateLimitWithToken(twoSecondsInterval, getGridAlgoOrderPositionsRate, 1),
		spotGridWithdrawIncomeEPL:    request.NewRateLimitWithToken(twoSecondsInterval, spotGridWithdrawIncomeRate, 1),
		computeMarginBalanceEPL:      request.NewRateLimitWithToken(twoSecondsInterval, computeMarginBalance, 1),
		adjustMarginBalanceEPL:       request.NewRateLimitWithToken(twoSecondsInterval, adjustMarginBalance, 1),
		getGridAIParameterEPL:        request.NewRateLimitWithToken(twoSecondsInterval, getGridAIParameter, 1),

		// Earn
		getOfferEPL:                   request.NewRateLimitWithToken(oneSecondInterval, getOffer, 1),
		purchaseEPL:                   request.NewRateLimitWithToken(oneSecondInterval, purchase, 1),
		redeemEPL:                     request.NewRateLimitWithToken(oneSecondInterval, redeem, 1),
		cancelPurchaseOrRedemptionEPL: request.NewRateLimitWithToken(oneSecondInterval, cancelPurchaseOrRedemption, 1),
		getEarnActiveOrdersEPL:        request.NewRateLimitWithToken(oneSecondInterval, getEarnActiveOrders, 1),
		getFundingOrderHistoryEPL:     request.NewRateLimitWithToken(oneSecondInterval, getFundingOrderHistory, 1),

		// Market Data
		getTickersEPL:             request.NewRateLimitWithToken(twoSecondsInterval, getTickersRate, 1),
		getIndexTickersEPL:        request.NewRateLimitWithToken(twoSecondsInterval, getIndexTickersRate, 1),
		getOrderBookEPL:           request.NewRateLimitWithToken(twoSecondsInterval, getOrderBookRate, 1),
		getCandlestickEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:  request.NewRateLimitWithToken(twoSecondsInterval, getCandlesticksHistoryRate, 1),
		getIndexCandlestickEPL:    request.NewRateLimitWithToken(twoSecondsInterval, getIndexCandlesticksRate, 1),
		getTradesRequestEPL:       request.NewRateLimitWithToken(twoSecondsInterval, getTradesRequestRate, 1),
		get24HTotalVolumeEPL:      request.NewRateLimitWithToken(twoSecondsInterval, get24HTotalVolumeRate, 1),
		getOracleEPL:              request.NewRateLimitWithToken(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateRequestEPL: request.NewRateLimitWithToken(twoSecondsInterval, getExchangeRateRequestRate, 1),
		getIndexComponentsEPL:     request.NewRateLimitWithToken(twoSecondsInterval, getIndexComponentsRate, 1),
		getBlockTickersEPL:        request.NewRateLimitWithToken(twoSecondsInterval, getBlockTickersRate, 1),
		getBlockTradesEPL:         request.NewRateLimitWithToken(twoSecondsInterval, getBlockTradesRate, 1),

		// Public Data Endpoints
		getInstrumentsEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, getInstrumentsRate, 1),
		getDeliveryExerciseHistoryEPL:             request.NewRateLimitWithToken(twoSecondsInterval, getDeliveryExerciseHistoryRate, 1),
		getOpenInterestEPL:                        request.NewRateLimitWithToken(twoSecondsInterval, getOpenInterestRate, 1),
		getFundingEPL:                             request.NewRateLimitWithToken(twoSecondsInterval, getFundingRate, 1),
		getFundingRateHistoryEPL:                  request.NewRateLimitWithToken(twoSecondsInterval, getFundingRateHistoryRate, 1),
		getLimitPriceEPL:                          request.NewRateLimitWithToken(twoSecondsInterval, getLimitPriceRate, 1),
		getOptionMarketDateEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getOptionMarketDateRate, 1),
		getEstimatedDeliveryExercisePriceEPL:      request.NewRateLimitWithToken(twoSecondsInterval, getEstimatedDeliveryExercisePriceRate, 1),
		getDiscountRateAndInterestFreeQuotaEPL:    request.NewRateLimitWithToken(twoSecondsInterval, getDiscountRateAndInterestFreeQuotaRate, 1),
		getSystemTimeEPL:                          request.NewRateLimitWithToken(twoSecondsInterval, getSystemTimeRate, 1),
		getLiquidationOrdersEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, getLiquidationOrdersRate, 1),
		getMarkPriceEPL:                           request.NewRateLimitWithToken(twoSecondsInterval, getMarkPriceRate, 1),
		getPositionTiersEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, getPositionTiersRate, 1),
		getInterestRateAndLoanQuotaEPL:            request.NewRateLimitWithToken(twoSecondsInterval, getInterestRateAndLoanQuotaRate, 1),
		getInterestRateAndLoanQuoteForVIPLoansEPL: request.NewRateLimitWithToken(twoSecondsInterval, getInterestRateAndLoanQuoteForVIPLoansRate, 1),
		getUnderlyingEPL:                          request.NewRateLimitWithToken(twoSecondsInterval, getUnderlyingRate, 1),
		getInsuranceFundEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, getInsuranceFundRate, 1),
		unitConvertEPL:                            request.NewRateLimitWithToken(twoSecondsInterval, unitConvertRate, 1),

		// Trading Data Endpoints
		getSupportCoinEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getSupportCoinRate, 1),
		getTakerVolumeEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getTakerVolumeRate, 1),
		getMarginLendingRatioEPL:             request.NewRateLimitWithToken(twoSecondsInterval, getMarginLendingRatioRate, 1),
		getLongShortRatioEPL:                 request.NewRateLimitWithToken(twoSecondsInterval, getLongShortRatioRate, 1),
		getContractsOpenInterestAndVolumeEPL: request.NewRateLimitWithToken(twoSecondsInterval, getContractsOpenInterestAndVolumeRate, 1),
		getOptionsOpenInterestAndVolumeEPL:   request.NewRateLimitWithToken(twoSecondsInterval, getOptionsOpenInterestAndVolumeRate, 1),
		getPutCallRatioEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, getPutCallRatioRate, 1),
		getOpenInterestAndVolumeEPL:          request.NewRateLimitWithToken(twoSecondsInterval, getOpenInterestAndVolumeRate, 1),
		getTakerFlowEPL:                      request.NewRateLimitWithToken(twoSecondsInterval, getTakerFlowRate, 1),

		// Status Endpoints
		getEventStatusEPL: request.NewRateLimitWithToken(fiveSecondsInterval, getEventStatusRate, 1),
	}
}
