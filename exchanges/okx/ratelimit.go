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
func SetRateLimit(ok *Okx) request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// Trade Endpoints
		placeOrderEPL:                  request.NewRateLimit(twoSecondsInterval, placeOrderRate, 1),
		placeMultipleOrdersEPL:         request.NewRateLimit(twoSecondsInterval, placeMultipleOrdersRate, 1),
		cancelOrderEPL:                 request.NewRateLimit(twoSecondsInterval, cancelOrderRate, 1),
		cancelMultipleOrdersEPL:        request.NewRateLimit(twoSecondsInterval, cancelMultipleOrdersRate, 1),
		amendOrderEPL:                  request.NewRateLimit(twoSecondsInterval, amendOrderRate, 1),
		amendMultipleOrdersEPL:         request.NewRateLimit(twoSecondsInterval, amendMultipleOrdersRate, 1),
		ok.ClosePositions:              request.NewRateLimit(twoSecondsInterval, closePositionsRate, 1),
		getOrderDetEPL:                 request.NewRateLimit(twoSecondsInterval, getOrderDetails, 1),
		getOrderListEPL:                request.NewRateLimit(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistory7DaysEPL:        request.NewRateLimit(twoSecondsInterval, getOrderHistory7DaysRate, 1),
		getOrderHistory3MonthsEPL:      request.NewRateLimit(twoSecondsInterval, getOrderHistory3MonthsRate, 1),
		getTransactionDetail3DaysEPL:   request.NewRateLimit(twoSecondsInterval, getTransactionDetail3DaysRate, 1),
		getTransactionDetail3MonthsEPL: request.NewRateLimit(twoSecondsInterval, getTransactionDetail3MonthsRate, 1),
		placeAlgoOrderEPL:              request.NewRateLimit(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:             request.NewRateLimit(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvanceAlgoOrderEPL:      request.NewRateLimit(twoSecondsInterval, cancelAdvanceAlgoOrderRate, 1),
		getAlgoOrderListEPL:            request.NewRateLimit(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:         request.NewRateLimit(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getEasyConvertCurrencyListEPL:  request.NewRateLimit(twoSecondsInterval, getEasyConvertCurrencyListRate, 1),
		placeEasyConvertEPL:            request.NewRateLimit(twoSecondsInterval, placeEasyConvert, 1),
		getEasyConvertHistoryEPL:       request.NewRateLimit(twoSecondsInterval, getEasyConvertHistory, 1),
		getOneClickRepayHistoryEPL:     request.NewRateLimit(twoSecondsInterval, getOneClickRepayHistory, 1),
		oneClickRepayCurrencyListEPL:   request.NewRateLimit(twoSecondsInterval, oneClickRepayCurrencyList, 1),
		tradeOneClickRepayEPL:          request.NewRateLimit(twoSecondsInterval, tradeOneClickRepay, 1),

		// Block Trading endpoints
		getCounterpartiesEPL:    request.NewRateLimit(twoSecondsInterval, getCounterpartiesRate, 1),
		createRfqEPL:            request.NewRateLimit(twoSecondsInterval, createRfqRate, 1),
		cancelRfqEPL:            request.NewRateLimit(twoSecondsInterval, cancelRfqRate, 1),
		cancelMultipleRfqEPL:    request.NewRateLimit(twoSecondsInterval, cancelMultipleRfqRate, 1),
		cancelAllRfqsEPL:        request.NewRateLimit(twoSecondsInterval, cancelAllRfqsRate, 1),
		executeQuoteEPL:         request.NewRateLimit(threeSecondsInterval, executeQuoteRate, 1),
		setQuoteProductsEPL:     request.NewRateLimit(twoSecondsInterval, setQuoteProducts, 1),
		restMMPStatusEPL:        request.NewRateLimit(twoSecondsInterval, restMMPStatus, 1),
		createQuoteEPL:          request.NewRateLimit(twoSecondsInterval, createQuoteRate, 1),
		cancelQuoteEPL:          request.NewRateLimit(twoSecondsInterval, cancelQuoteRate, 1),
		cancelMultipleQuotesEPL: request.NewRateLimit(twoSecondsInterval, cancelMultipleQuotesRate, 1),
		cancelAllQuotesEPL:      request.NewRateLimit(twoSecondsInterval, cancelAllQuotes, 1),
		getRfqsEPL:              request.NewRateLimit(twoSecondsInterval, getRfqsRate, 1),
		getQuotesEPL:            request.NewRateLimit(twoSecondsInterval, getQuotesRate, 1),
		getTradesEPL:            request.NewRateLimit(twoSecondsInterval, getTradesRate, 1),
		getTradesHistoryEPL:     request.NewRateLimit(twoSecondsInterval, getTradesHistoryRate, 1),
		getPublicTradesEPL:      request.NewRateLimit(twoSecondsInterval, getPublicTradesRate, 1),
		// Funding
		getCurrenciesEPL:             request.NewRateLimit(oneSecondInterval, getCurrenciesRate, 1),
		getBalanceEPL:                request.NewRateLimit(oneSecondInterval, getBalanceRate, 1),
		getAccountAssetValuationEPL:  request.NewRateLimit(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundsTransferEPL:             request.NewRateLimit(oneSecondInterval, fundsTransferRate, 1),
		getFundsTransferStateEPL:     request.NewRateLimit(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailsEPL:         request.NewRateLimit(oneSecondInterval, assetBillsDetailsRate, 1),
		lightningDepositsEPL:         request.NewRateLimit(oneSecondInterval, lightningDepositsRate, 1),
		getDepositAddressEPL:         request.NewRateLimit(oneSecondInterval, getDepositAddressRate, 1),
		getDepositHistoryEPL:         request.NewRateLimit(oneSecondInterval, getDepositHistoryRate, 1),
		withdrawalEPL:                request.NewRateLimit(oneSecondInterval, withdrawalRate, 1),
		lightningWithdrawalsEPL:      request.NewRateLimit(oneSecondInterval, lightningWithdrawalsRate, 1),
		cancelWithdrawalEPL:          request.NewRateLimit(oneSecondInterval, cancelWithdrawalRate, 1),
		getWithdrawalHistoryEPL:      request.NewRateLimit(oneSecondInterval, getWithdrawalHistoryRate, 1),
		smallAssetsConvertEPL:        request.NewRateLimit(oneSecondInterval, smallAssetsConvertRate, 1),
		getSavingBalanceEPL:          request.NewRateLimit(oneSecondInterval, getSavingBalanceRate, 1),
		savingsPurchaseRedemptionEPL: request.NewRateLimit(oneSecondInterval, savingsPurchaseRedemptionRate, 1),
		setLendingRateEPL:            request.NewRateLimit(oneSecondInterval, setLendingRateRate, 1),
		getLendingHistoryEPL:         request.NewRateLimit(oneSecondInterval, getLendingHistoryRate, 1),
		getPublicBorrowInfoEPL:       request.NewRateLimit(oneSecondInterval, getPublicBorrowInfoRate, 1),
		getPublicBorrowHistoryEPL:    request.NewRateLimit(oneSecondInterval, getPublicBorrowHistoryRate, 1),

		// Convert
		getConvertCurrenciesEPL:   request.NewRateLimit(oneSecondInterval, getConvertCurrenciesRate, 1),
		getConvertCurrencyPairEPL: request.NewRateLimit(oneSecondInterval, getConvertCurrencyPairRate, 1),
		estimateQuoteEPL:          request.NewRateLimit(oneSecondInterval, estimateQuoteRate, 1),
		convertTradeEPL:           request.NewRateLimit(oneSecondInterval, convertTradeRate, 1),
		getConvertHistoryEPL:      request.NewRateLimit(oneSecondInterval, getConvertHistoryRate, 1),

		// Account
		getAccountBalanceEPL:                 request.NewRateLimit(twoSecondsInterval, getAccountBalanceRate, 1),
		getPositionsEPL:                      request.NewRateLimit(twoSecondsInterval, getPositionsRate, 1),
		getPositionsHistoryEPL:               request.NewRateLimit(tenSecondsInterval, getPositionsHistoryRate, 1),
		getAccountAndPositionRiskEPL:         request.NewRateLimit(twoSecondsInterval, getAccountAndPositionRiskRate, 1),
		getBillsDetailsEPL:                   request.NewRateLimit(oneSecondInterval, getBillsDetailsRate, 1),
		getAccountConfigurationEPL:           request.NewRateLimit(twoSecondsInterval, getAccountConfigurationRate, 1),
		setPositionModeEPL:                   request.NewRateLimit(twoSecondsInterval, setPositionModeRate, 1),
		setLeverageEPL:                       request.NewRateLimit(twoSecondsInterval, setLeverageRate, 1),
		getMaximumBuyOrSellAmountEPL:         request.NewRateLimit(twoSecondsInterval, getMaximumBuyOrSellAmountRate, 1),
		getMaximumAvailableTradableAmountEPL: request.NewRateLimit(twoSecondsInterval, getMaximumAvailableTradableAmountRate, 1),
		increaseOrDecreaseMarginEPL:          request.NewRateLimit(twoSecondsInterval, increaseOrDecreaseMarginRate, 1),
		getLeverageEPL:                       request.NewRateLimit(twoSecondsInterval, getLeverageRate, 1),
		getTheMaximumLoanOfInstrumentEPL:     request.NewRateLimit(twoSecondsInterval, getTheMaximumLoanOfInstrumentRate, 1),
		getFeeRatesEPL:                       request.NewRateLimit(twoSecondsInterval, getFeeRatesRate, 1),
		getInterestAccruedDataEPL:            request.NewRateLimit(twoSecondsInterval, getInterestAccruedDataRate, 1),
		getInterestRateEPL:                   request.NewRateLimit(twoSecondsInterval, getInterestRateRate, 1),
		setGreeksEPL:                         request.NewRateLimit(twoSecondsInterval, setGreeksRate, 1),
		isolatedMarginTradingSettingsEPL:     request.NewRateLimit(twoSecondsInterval, isolatedMarginTradingSettingsRate, 1),
		getMaximumWithdrawalsEPL:             request.NewRateLimit(twoSecondsInterval, getMaximumWithdrawalsRate, 1),
		getAccountRiskStateEPL:               request.NewRateLimit(twoSecondsInterval, getAccountRiskStateRate, 1),
		vipLoansBorrowAnsRepayEPL:            request.NewRateLimit(oneSecondInterval, vipLoansBorrowAndRepayRate, 1),
		getBorrowAnsRepayHistoryHistoryEPL:   request.NewRateLimit(twoSecondsInterval, getBorrowAnsRepayHistoryHistoryRate, 1),
		getBorrowInterestAndLimitEPL:         request.NewRateLimit(twoSecondsInterval, getBorrowInterestAndLimitRate, 1),
		positionBuilderEPL:                   request.NewRateLimit(twoSecondsInterval, positionBuilderRate, 1),
		getGreeksEPL:                         request.NewRateLimit(twoSecondsInterval, getGreeksRate, 1),
		getPMLimitationEPL:                   request.NewRateLimit(twoSecondsInterval, getPMLimitation, 1),

		// Sub Account Endpoints
		viewSubaccountListEPL:                             request.NewRateLimit(twoSecondsInterval, viewSubaccountListRate, 1),
		resetSubAccountAPIKeyEPL:                          request.NewRateLimit(oneSecondInterval, resetSubAccountAPIKey, 1),
		getSubaccountTradingBalanceEPL:                    request.NewRateLimit(twoSecondsInterval, getSubaccountTradingBalanceRate, 1),
		getSubaccountFundingBalanceEPL:                    request.NewRateLimit(twoSecondsInterval, getSubaccountFundingBalanceRate, 1),
		historyOfSubaccountTransferEPL:                    request.NewRateLimit(oneSecondInterval, historyOfSubaccountTransferRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimit(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		setPermissionOfTransferOutEPL:                     request.NewRateLimit(oneSecondInterval, setPermissionOfTransferOutRate, 1),
		getCustodyTradingSubaccountListEPL:                request.NewRateLimit(oneSecondInterval, getCustodyTradingSubaccountListRate, 1),

		// Grid Trading Endpoints
		gridTradingEPL:               request.NewRateLimit(twoSecondsInterval, gridTradingRate, 1),
		amendGridAlgoOrderEPL:        request.NewRateLimit(twoSecondsInterval, amendGridAlgoOrderRate, 1),
		stopGridAlgoOrderEPL:         request.NewRateLimit(twoSecondsInterval, stopGridAlgoOrderRate, 1),
		getGridAlgoOrderListEPL:      request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderListRate, 1),
		getGridAlgoOrderHistoryEPL:   request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderHistoryRate, 1),
		getGridAlgoOrderDetailsEPL:   request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderDetailsRate, 1),
		getGridAlgoSubOrdersEPL:      request.NewRateLimit(twoSecondsInterval, getGridAlgoSubOrdersRate, 1),
		getGridAlgoOrderPositionsEPL: request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderPositionsRate, 1),
		spotGridWithdrawIncomeEPL:    request.NewRateLimit(twoSecondsInterval, spotGridWithdrawIncomeRate, 1),
		computeMarginBalanceEPL:      request.NewRateLimit(twoSecondsInterval, computeMarginBalance, 1),
		adjustMarginBalanceEPL:       request.NewRateLimit(twoSecondsInterval, adjustMarginBalance, 1),
		getGridAIParameterEPL:        request.NewRateLimit(twoSecondsInterval, getGridAIParameter, 1),

		// Earn
		getOfferEPL:                   request.NewRateLimit(oneSecondInterval, getOffer, 1),
		purchaseEPL:                   request.NewRateLimit(oneSecondInterval, purchase, 1),
		redeemEPL:                     request.NewRateLimit(oneSecondInterval, redeem, 1),
		cancelPurchaseOrRedemptionEPL: request.NewRateLimit(oneSecondInterval, cancelPurchaseOrRedemption, 1),
		getEarnActiveOrdersEPL:        request.NewRateLimit(oneSecondInterval, getEarnActiveOrders, 1),
		getFundingOrderHistoryEPL:     request.NewRateLimit(oneSecondInterval, getFundingOrderHistory, 1),

		// Market Data
		getTickersEPL:             request.NewRateLimit(twoSecondsInterval, getTickersRate, 1),
		getIndexTickersEPL:        request.NewRateLimit(twoSecondsInterval, getIndexTickersRate, 1),
		getOrderBookEPL:           request.NewRateLimit(twoSecondsInterval, getOrderBookRate, 1),
		getCandlestickEPL:         request.NewRateLimit(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:  request.NewRateLimit(twoSecondsInterval, getCandlesticksHistoryRate, 1),
		getIndexCandlestickEPL:    request.NewRateLimit(twoSecondsInterval, getIndexCandlesticksRate, 1),
		getTradesRequestEPL:       request.NewRateLimit(twoSecondsInterval, getTradesRequestRate, 1),
		get24HTotalVolumeEPL:      request.NewRateLimit(twoSecondsInterval, get24HTotalVolumeRate, 1),
		getOracleEPL:              request.NewRateLimit(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateRequestEPL: request.NewRateLimit(twoSecondsInterval, getExchangeRateRequestRate, 1),
		getIndexComponentsEPL:     request.NewRateLimit(twoSecondsInterval, getIndexComponentsRate, 1),
		getBlockTickersEPL:        request.NewRateLimit(twoSecondsInterval, getBlockTickersRate, 1),
		getBlockTradesEPL:         request.NewRateLimit(twoSecondsInterval, getBlockTradesRate, 1),

		// Public Data Endpoints
		getInstrumentsEPL:                         request.NewRateLimit(twoSecondsInterval, getInstrumentsRate, 1),
		getDeliveryExerciseHistoryEPL:             request.NewRateLimit(twoSecondsInterval, getDeliveryExerciseHistoryRate, 1),
		getOpenInterestEPL:                        request.NewRateLimit(twoSecondsInterval, getOpenInterestRate, 1),
		getFundingEPL:                             request.NewRateLimit(twoSecondsInterval, getFundingRate, 1),
		getFundingRateHistoryEPL:                  request.NewRateLimit(twoSecondsInterval, getFundingRateHistoryRate, 1),
		getLimitPriceEPL:                          request.NewRateLimit(twoSecondsInterval, getLimitPriceRate, 1),
		getOptionMarketDateEPL:                    request.NewRateLimit(twoSecondsInterval, getOptionMarketDateRate, 1),
		getEstimatedDeliveryExercisePriceEPL:      request.NewRateLimit(twoSecondsInterval, getEstimatedDeliveryExercisePriceRate, 1),
		getDiscountRateAndInterestFreeQuotaEPL:    request.NewRateLimit(twoSecondsInterval, getDiscountRateAndInterestFreeQuotaRate, 1),
		getSystemTimeEPL:                          request.NewRateLimit(twoSecondsInterval, getSystemTimeRate, 1),
		getLiquidationOrdersEPL:                   request.NewRateLimit(twoSecondsInterval, getLiquidationOrdersRate, 1),
		getMarkPriceEPL:                           request.NewRateLimit(twoSecondsInterval, getMarkPriceRate, 1),
		getPositionTiersEPL:                       request.NewRateLimit(twoSecondsInterval, getPositionTiersRate, 1),
		getInterestRateAndLoanQuotaEPL:            request.NewRateLimit(twoSecondsInterval, getInterestRateAndLoanQuotaRate, 1),
		getInterestRateAndLoanQuoteForVIPLoansEPL: request.NewRateLimit(twoSecondsInterval, getInterestRateAndLoanQuoteForVIPLoansRate, 1),
		getUnderlyingEPL:                          request.NewRateLimit(twoSecondsInterval, getUnderlyingRate, 1),
		getInsuranceFundEPL:                       request.NewRateLimit(twoSecondsInterval, getInsuranceFundRate, 1),
		unitConvertEPL:                            request.NewRateLimit(twoSecondsInterval, unitConvertRate, 1),

		// Trading Data Endpoints
		getSupportCoinEPL:                    request.NewRateLimit(twoSecondsInterval, getSupportCoinRate, 1),
		getTakerVolumeEPL:                    request.NewRateLimit(twoSecondsInterval, getTakerVolumeRate, 1),
		getMarginLendingRatioEPL:             request.NewRateLimit(twoSecondsInterval, getMarginLendingRatioRate, 1),
		getLongShortRatioEPL:                 request.NewRateLimit(twoSecondsInterval, getLongShortRatioRate, 1),
		getContractsOpenInterestAndVolumeEPL: request.NewRateLimit(twoSecondsInterval, getContractsOpenInterestAndVolumeRate, 1),
		getOptionsOpenInterestAndVolumeEPL:   request.NewRateLimit(twoSecondsInterval, getOptionsOpenInterestAndVolumeRate, 1),
		getPutCallRatioEPL:                   request.NewRateLimit(twoSecondsInterval, getPutCallRatioRate, 1),
		getOpenInterestAndVolumeEPL:          request.NewRateLimit(twoSecondsInterval, getOpenInterestAndVolumeRate, 1),
		getTakerFlowEPL:                      request.NewRateLimit(twoSecondsInterval, getTakerFlowRate, 1),

		// Status Endpoints
		getEventStatusEPL: request.NewRateLimit(fiveSecondsInterval, getEventStatusRate, 1),
	}
}
