package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GateIO endpoints limits. See: https://www.gate.io/docs/developers/apiv4/en/#frequency-limit-rule
const (
	publicTickersSpotEPL request.EndpointLimit = iota + 1
	publicOrderbookSpotEPL
	publicMarketTradesSpotEPL
	publicCandleStickSpotEPL
	publicCurrencyPairDetailSpotEPL
	publicListCurrencyPairsSpotEPL
	publicCurrenciesSpotEPL

	publicCurrencyPairsMarginEPL
	publicOrderbookMarginEPL

	publicInsuranceDeliveryEPL
	publicDeliveryContractsEPL
	publicOrderbookDeliveryEPL
	publicTradingHistoryDeliveryEPL
	publicCandleSticksDeliveryEPL
	publicTickersDeliveryEPL

	publicFuturesContractsEPL
	publicOrderbookFuturesEPL
	publicTradingHistoryFuturesEPL
	publicCandleSticksFuturesEPL
	publicPremiumIndexEPL
	publicTickersFuturesEPL
	publicFundingRatesEPL
	publicInsuranceFuturesEPL
	publicStatsFuturesEPL
	publicIndexConstituentsEPL
	publicLiquidationHistoryEPL

	publicUnderlyingOptionsEPL
	publicExpirationOptionsEPL
	publicContractsOptionsEPL
	publicSettlementOptionsEPL
	publicOrderbookOptionsEPL
	publicTickerOptionsEPL
	publicUnderlyingTickerOptionsEPL
	publicCandleSticksOptionsEPL
	publicMarkpriceCandleSticksOptionsEPL
	publicTradeHistoryOptionsEPL

	publicGetServerTimeEPL
	publicFlashSwapEPL
	publicListCurrencyChainEPL

	walletDepositAddressEPL
	walletWithdrawalRecordsEPL
	walletDepositRecordsEPL
	walletTransferCurrencyEPL
	walletSubAccountTransferEPL
	walletSubAccountTransferHistoryEPL
	walletSubAccountToSubAccountTransferEPL
	walletWithdrawStatusEPL
	walletSubAccountBalancesEPL
	walletSubAccountMarginBalancesEPL
	walletSubAccountFuturesBalancesEPL
	walletSubAccountCrossMarginBalancesEPL
	walletSavedAddressesEPL
	walletTradingFeeEPL
	walletTotalBalanceEPL
	walletConvertSmallBalancesEPL
	walletWithdrawEPL
	walletCancelWithdrawEPL

	subAccountEPL

	spotTradingFeeEPL
	spotAccountsEPL
	spotGetOpenOrdersEPL
	spotClosePositionEPL
	spotBatchOrdersEPL
	spotPlaceOrderEPL
	spotGetOrdersEPL
	spotCancelAllOpenOrdersEPL
	spotCancelBatchOrdersEPL
	spotGetOrderEPL
	spotAmendOrderEPL
	spotCancelSingleOrderEPL
	spotTradingHistoryEPL
	spotCountdownCancelEPL
	spotCreateTriggerOrderEPL
	spotGetTriggerOrderListEPL
	spotCancelTriggerOrdersEPL
	spotGetTriggerOrderEPL
	spotCancelTriggerOrderEPL

	marginAccountListEPL
	marginAccountBalanceEPL
	marginFundingAccountListEPL
	marginLendBorrowEPL
	marginAllLoansEPL
	marginMergeLendingLoansEPL
	marginGetLoanEPL
	marginModifyLoanEPL
	marginCancelLoanEPL
	marginRepayLoanEPL
	marginListLoansEPL
	marginRepaymentRecordEPL
	marginSingleRecordEPL
	marginModifyLoanRecordEPL
	marginAutoRepayEPL
	marginGetAutoRepaySettingsEPL
	marginGetMaxTransferEPL
	marginGetMaxBorrowEPL
	marginSupportedCurrencyCrossListEPL
	marginSupportedCurrencyCrossEPL
	marginAccountsEPL
	marginAccountHistoryEPL
	marginCreateCrossBorrowLoanEPL
	marginExecuteRepaymentsEPL
	marginGetCrossMarginRepaymentsEPL
	marginGetMaxTransferCrossEPL
	marginGetMaxBorrowCrossEPL
	marginGetCrossBorrowHistoryEPL
	marginGetBorrowEPL
	marginEstimateRateEPL

	flashSwapOrderEPL
	flashGetOrdersEPL
	flashGetOrderEPL
	flashOrderReviewEPL

	privateUnifiedSpotEPL

	perpetualAccountEPL
	perpetualAccountBooksEPL
	perpetualPositionsEPL
	perpetualPositionEPL
	perpetualUpdateMarginEPL
	perpetualUpdateLeverageEPL
	perpetualToggleDualModeEPL
	perpetualPositionsDualModeEPL
	perpetualUpdateMarginDualModeEPL
	perpetualUpdateLeverageDualModeEPL
	perpetualSubmitOrderEPL
	perpetualGetOrdersEPL
	perpetualSubmitBatchOrdersEPL
	perpetualFetchOrderEPL
	perpetualCancelOrderEPL
	perpetualAmendOrderEPL
	perpetualTradingHistoryEPL
	perpetualClosePositionEPL
	perpetualLiquidationHistoryEPL
	perpetualCancelTriggerOrdersEPL
	perpetualSubmitTriggerOrderEPL
	perpetualListOpenOrdersEPL
	perpetualCancelOpenOrdersEPL
	perpetualGetTriggerOrderEPL
	perpetualCancelTriggerOrderEPL
	perpetualCreateChaseOrderEPL
	perpetualStopChaseOrderEPL
	perpetualStopAllChaseOrdersEPL
	perpetualGetChaseOrdersEPL
	perpetualGetChaseOrderDetailEPL

	deliveryAccountEPL
	deliveryAccountBooksEPL
	deliveryPositionsEPL
	deliveryUpdateMarginEPL
	deliveryUpdateLeverageEPL
	deliverySubmitOrderEPL
	deliveryGetOrdersEPL
	deliveryCancelOrdersEPL
	deliveryGetOrderEPL
	deliveryCancelOrderEPL
	deliveryTradingHistoryEPL
	deliveryCloseHistoryEPL
	deliveryLiquidationHistoryEPL
	deliverySettlementHistoryEPL
	deliveryGetTriggerOrdersEPL
	deliveryAutoOrdersEPL
	deliveryCancelTriggerOrdersEPL
	deliveryGetTriggerOrderEPL
	deliveryCancelTriggerOrderEPL

	optionsSettlementsEPL
	optionsAccountsEPL
	optionsAccountBooksEPL
	optionsPositions
	optionsLiquidationHistoryEPL
	optionsSubmitOrderEPL
	optionsOrdersEPL
	optionsCancelOrdersEPL
	optionsOrderEPL
	optionsCancelOrderEPL
	optionsTradingHistoryEPL

	websocketRateLimitNotNeededEPL

	// Risk EPLs
	publicFuturesRiskTableEPL
	publicFuturesRiskLimitTiersEPL
	publicDeliveryRiskLimitTiersEPL
	unifiedUserRiskUnitDetailsEPL
	deliveryUpdateRiskLimitEPL
	perpetualUpdateRiskDualModeEPL
	perpetualUpdateRiskEPL

	// Alpha EPLs
	alphaAccountsEPL
	alphaAccountBookEPL
	alphaCreateQuoteEPL
	alphaPlaceOrderEPL
	alphaListOrdersEPL
	alphaGetOrderEPL
	alphaCurrenciesEPL
	alphaTickersEPL

	// Unified EPLs
	unifiedInterestRecordsEPL
	unifiedRiskUnitsEPL
	unifiedUpdateUnifiedModeEPL
	unifiedGetUnifiedModeEPL
	unifiedEstimateRateEPL
	unifiedCurrencyDiscountTiersEPL
	unifiedLoanMarginTiersEPL
	unifiedPortfolioCalculatorEPL
	unifiedGetLeverageUserCurrencySettingEPL
	unifiedCreateLeverageUserCurrencySettingEPL
	unifiedLeverageUserCurrencyConfigEPL
	unifiedCurrenciesEPL
	unifiedHistoryLoanRateEPL
	unifiedCollateralCurrenciesEPL
	unifiedEstimateQuickRepaymentEPL
	unifiedQuickRepaymentEPL

	// Earn fixed-term EPLs
	earnFixedTermProductEPL
	earnFixedTermProductListEPL

	// Loan multi-collateral EPLs
	loanMultiCollateralMortgageEPL
	loanMultiCollateralCurrencyQuotaEPL
	loanMultiCollateralCurrentRateEPL
	loanMultiCollateralCurrenciesEPL
	loanMultiCollateralLtvEPL
	loanMultiCollateralFixedRateEPL

	// Earn uni EPLs
	earnUniCreateLendsEPL
	earnUniGetLendsEPL
	earnUniUpdateLendsEPL
	earnUniLendRecordsEPL
	earnUniInterestsEPL
	earnUniInterestRecordsEPL
	earnUniInterestStatusEPL
	earnUniChartEPL
	earnUniRateEPL
	earnUniCurrenciesEPL
	earnUniCurrencyEPL

	// Margin uni EPLs
	marginUniCreateLoansEPL
	marginUniGetLoansEPL
	marginUniLoanRecordsEPL
	marginUniInterestRecordsEPL
	marginUniBorrowableEPL
	marginUniCurrencyPairsEPL
	marginUniCurrencyPairEPL

	// TradFi EPLs
	tradfiUsersMT5AccountEPL
	tradfiSymbolsDetailEPL
	tradfiKlinesEPL
	tradfiUsersEPL
	tradfiUsersAssetsEPL
	tradfiGetTransactionsEPL
	tradfiCreateTransactionsEPL
	tradfiGetOrdersEPL
	tradfiCreateOrdersEPL
	tradfiUpdateOrdersEPL
	tradfiDeleteOrdersEPL
	tradfiOrdersHistoryEPL
	tradfiGetPositionsEPL
	tradfiUpdatePositionsEPL
	tradfiCreatePositionsEPL
	tradfiPositionsHistoryEPL
	tradfiSymbolsCategoriesEPL
	tradfiSymbolsEPL
	tradfiTickersEPL

	// CrossEx EPLs
	crossexSymbolsEPL
	crossexRiskLimitEPL
	crossexTransfersCoinEPL
	crossexGetTransfersEPL
	crossexCreateTransfersEPL
	crossexCreateOrdersEPL
	crossexGetOrdersEPL
	crossexUpdateOrdersEPL
	crossexDeleteOrdersEPL
	crossexConvertQuoteEPL
	crossexConvertOrdersEPL
	crossexGetAccountsEPL
	crossexUpdateAccountsEPL
	crossexGetPositionsLeverageEPL
	crossexCreatePositionsLeverageEPL
	crossexGetMarginPositionsLeverageEPL
	crossexCreateMarginPositionsLeverageEPL
	crossexPositionEPL
	crossexInterestRateEPL
	crossexFeeEPL
	crossexPositionsEPL
	crossexMarginPositionsEPL
	crossexADLRankEPL
	crossexOpenOrdersEPL
	crossexHistoryOrdersEPL
	crossexHistoryPositionsEPL
	crossexHistoryMarginPositionsEPL
	crossexHistoryMarginInterestsEPL
	crossexHistoryTradesEPL
	crossexAccountBookEPL
	crossexCoinDiscountRateEPL

	// P2P EPLs
	p2pAccountInfoEPL
	p2pCounterpartyInfoEPL
	p2pPaymentMethodsEPL
	p2pSetWorkHoursEPL
	p2pPendingTransactionsEPL
	p2pCompletedTransactionsEPL
	p2pMyListEPL
	p2pMyHistoryListEPL
	p2pTransactionDetailsEPL
	p2pConfirmPaymentEPL
	p2pConfirmReceiptEPL
	p2pCancelTransactionEPL
	p2pPublishAdEPL
	p2pUpdateAdStatusEPL
	p2pAdDetailEPL
	p2pMyAdsListEPL
	p2pAdsListEPL
	p2pChatHistoryEPL
	p2pSendChatMessageEPL
	p2pUploadChatFileEPL

	// Bot EPLs
	botStrategyRecommendEPL
	botSpotGridCreateEPL
	botMarginGridCreateEPL
	botInfiniteGridCreateEPL
	botFuturesGridCreateEPL
	botSpotMartingaleCreateEPL
	botContractMartingaleCreateEPL
	botPortfolioRunningEPL
	botPortfolioDetailEPL
	botPortfolioStopEPL

	// Rebate EPLs
	rebateAgencyTransactionHistoryEPL
	rebateAgencyCommissionHistoryEPL
	rebatePartnerTransactionHistoryEPL
	rebatePartnerCommissionHistoryEPL
	rebatePartnerSubListEPL
	rebateBrokerCommissionHistoryEPL
	rebateBrokerTransactionHistoryEPL
	rebateUserInfoEPL
	rebateUserSubRelationEPL
	rebatePartnerApplicationsRecentEPL
	rebatePartnerEligibilityEPL
	rebatePartnerDataAggregatedEPL

	// OTC EPLs
	otcQuoteEPL
	otcOrderCreateEPL
	otcStablecoinOrderCreateEPL
	otcBankListEPL
	otcBankCreateEPL
	otcBankDeleteEPL
	otcBankSetDefaultEPL
	otcBankSupplementChecklistEPL
	otcBankPersonalSupplementEPL
	otcBankEnterpriseSupplementEPL
	otcOrderPaidEPL
	otcOrderCancelEPL
	otcOrderListEPL
	otcStablecoinOrderListEPL
	otcOrderDetailEPL
)

// package level rate limits for REST API
var packageRateLimits = request.RateLimitDefinitions{
	publicOrderbookSpotEPL:          standardRateLimit(),
	publicMarketTradesSpotEPL:       standardRateLimit(),
	publicCandleStickSpotEPL:        standardRateLimit(),
	publicTickersSpotEPL:            standardRateLimit(),
	publicCurrencyPairDetailSpotEPL: standardRateLimit(),
	publicListCurrencyPairsSpotEPL:  standardRateLimit(),
	publicCurrenciesSpotEPL:         standardRateLimit(),

	publicCurrencyPairsMarginEPL: standardRateLimit(),
	publicOrderbookMarginEPL:     standardRateLimit(),

	publicInsuranceDeliveryEPL:      standardRateLimit(),
	publicDeliveryContractsEPL:      standardRateLimit(),
	publicOrderbookDeliveryEPL:      standardRateLimit(),
	publicTradingHistoryDeliveryEPL: standardRateLimit(),
	publicCandleSticksDeliveryEPL:   standardRateLimit(),
	publicTickersDeliveryEPL:        standardRateLimit(),

	publicFuturesContractsEPL:      standardRateLimit(),
	publicOrderbookFuturesEPL:      standardRateLimit(),
	publicTradingHistoryFuturesEPL: standardRateLimit(),
	publicCandleSticksFuturesEPL:   standardRateLimit(),
	publicPremiumIndexEPL:          standardRateLimit(),
	publicTickersFuturesEPL:        standardRateLimit(),
	publicFundingRatesEPL:          standardRateLimit(),
	publicInsuranceFuturesEPL:      standardRateLimit(),
	publicStatsFuturesEPL:          standardRateLimit(),
	publicIndexConstituentsEPL:     standardRateLimit(),
	publicLiquidationHistoryEPL:    standardRateLimit(),

	publicUnderlyingOptionsEPL:            standardRateLimit(),
	publicExpirationOptionsEPL:            standardRateLimit(),
	publicContractsOptionsEPL:             standardRateLimit(),
	publicSettlementOptionsEPL:            standardRateLimit(),
	publicOrderbookOptionsEPL:             standardRateLimit(),
	publicTickerOptionsEPL:                standardRateLimit(),
	publicUnderlyingTickerOptionsEPL:      standardRateLimit(),
	publicCandleSticksOptionsEPL:          standardRateLimit(),
	publicMarkpriceCandleSticksOptionsEPL: standardRateLimit(),
	publicTradeHistoryOptionsEPL:          standardRateLimit(),

	publicGetServerTimeEPL:     standardRateLimit(),
	publicFlashSwapEPL:         standardRateLimit(),
	publicListCurrencyChainEPL: standardRateLimit(),

	walletDepositAddressEPL:                 standardRateLimit(),
	walletWithdrawalRecordsEPL:              standardRateLimit(),
	walletDepositRecordsEPL:                 standardRateLimit(),
	walletTransferCurrencyEPL:               personalAccountRateLimit(),
	walletSubAccountTransferEPL:             personalAccountRateLimit(),
	walletSubAccountTransferHistoryEPL:      standardRateLimit(),
	walletSubAccountToSubAccountTransferEPL: personalAccountRateLimit(),
	walletWithdrawStatusEPL:                 standardRateLimit(),
	walletSubAccountBalancesEPL:             personalAccountRateLimit(),
	walletSubAccountMarginBalancesEPL:       personalAccountRateLimit(),
	walletSubAccountFuturesBalancesEPL:      personalAccountRateLimit(),
	walletSubAccountCrossMarginBalancesEPL:  personalAccountRateLimit(),
	walletSavedAddressesEPL:                 standardRateLimit(),
	walletTradingFeeEPL:                     standardRateLimit(),
	walletTotalBalanceEPL:                   personalAccountRateLimit(),
	walletConvertSmallBalancesEPL:           personalAccountRateLimit(),
	walletWithdrawEPL:                       withdrawFromWalletRateLimit(),
	walletCancelWithdrawEPL:                 standardRateLimit(),

	subAccountEPL: personalAccountRateLimit(),

	spotTradingFeeEPL:          standardRateLimit(),
	spotAccountsEPL:            standardRateLimit(),
	spotGetOpenOrdersEPL:       standardRateLimit(),
	spotClosePositionEPL:       orderCloseRateLimit(),
	spotBatchOrdersEPL:         spotOrderPlacementRateLimit(),
	spotPlaceOrderEPL:          spotOrderPlacementRateLimit(),
	spotGetOrdersEPL:           standardRateLimit(),
	spotCancelAllOpenOrdersEPL: orderCloseRateLimit(),
	spotCancelBatchOrdersEPL:   orderCloseRateLimit(),
	spotGetOrderEPL:            standardRateLimit(),
	spotAmendOrderEPL:          spotOrderPlacementRateLimit(),
	spotCancelSingleOrderEPL:   orderCloseRateLimit(),
	spotTradingHistoryEPL:      standardRateLimit(),
	spotCountdownCancelEPL:     orderCloseRateLimit(),
	spotCreateTriggerOrderEPL:  spotOrderPlacementRateLimit(),
	spotGetTriggerOrderListEPL: standardRateLimit(),
	spotCancelTriggerOrdersEPL: orderCloseRateLimit(),
	spotGetTriggerOrderEPL:     standardRateLimit(),
	spotCancelTriggerOrderEPL:  orderCloseRateLimit(),

	marginAccountListEPL:                otherPrivateEndpointRateLimit(),
	marginAccountBalanceEPL:             otherPrivateEndpointRateLimit(),
	marginFundingAccountListEPL:         otherPrivateEndpointRateLimit(),
	marginLendBorrowEPL:                 otherPrivateEndpointRateLimit(),
	marginAllLoansEPL:                   otherPrivateEndpointRateLimit(),
	marginMergeLendingLoansEPL:          otherPrivateEndpointRateLimit(),
	marginGetLoanEPL:                    otherPrivateEndpointRateLimit(),
	marginModifyLoanEPL:                 otherPrivateEndpointRateLimit(),
	marginCancelLoanEPL:                 otherPrivateEndpointRateLimit(),
	marginRepayLoanEPL:                  otherPrivateEndpointRateLimit(),
	marginListLoansEPL:                  otherPrivateEndpointRateLimit(),
	marginRepaymentRecordEPL:            otherPrivateEndpointRateLimit(),
	marginSingleRecordEPL:               otherPrivateEndpointRateLimit(),
	marginModifyLoanRecordEPL:           otherPrivateEndpointRateLimit(),
	marginAutoRepayEPL:                  otherPrivateEndpointRateLimit(),
	marginGetAutoRepaySettingsEPL:       otherPrivateEndpointRateLimit(),
	marginGetMaxTransferEPL:             otherPrivateEndpointRateLimit(),
	marginGetMaxBorrowEPL:               otherPrivateEndpointRateLimit(),
	marginSupportedCurrencyCrossListEPL: otherPrivateEndpointRateLimit(),
	marginSupportedCurrencyCrossEPL:     otherPrivateEndpointRateLimit(),
	marginAccountsEPL:                   otherPrivateEndpointRateLimit(),
	marginAccountHistoryEPL:             otherPrivateEndpointRateLimit(),
	marginCreateCrossBorrowLoanEPL:      otherPrivateEndpointRateLimit(),
	marginExecuteRepaymentsEPL:          otherPrivateEndpointRateLimit(),
	marginGetCrossMarginRepaymentsEPL:   otherPrivateEndpointRateLimit(),
	marginGetMaxTransferCrossEPL:        otherPrivateEndpointRateLimit(),
	marginGetMaxBorrowCrossEPL:          otherPrivateEndpointRateLimit(),
	marginGetCrossBorrowHistoryEPL:      otherPrivateEndpointRateLimit(),
	marginGetBorrowEPL:                  otherPrivateEndpointRateLimit(),
	marginEstimateRateEPL:               otherPrivateEndpointRateLimit(),

	flashSwapOrderEPL:   otherPrivateEndpointRateLimit(),
	flashGetOrdersEPL:   otherPrivateEndpointRateLimit(),
	flashGetOrderEPL:    otherPrivateEndpointRateLimit(),
	flashOrderReviewEPL: otherPrivateEndpointRateLimit(),

	perpetualAccountEPL:                standardRateLimit(),
	perpetualAccountBooksEPL:           standardRateLimit(),
	perpetualPositionsEPL:              standardRateLimit(),
	perpetualPositionEPL:               standardRateLimit(),
	perpetualUpdateMarginEPL:           standardRateLimit(),
	perpetualUpdateLeverageEPL:         standardRateLimit(),
	perpetualToggleDualModeEPL:         standardRateLimit(),
	perpetualPositionsDualModeEPL:      standardRateLimit(),
	perpetualUpdateMarginDualModeEPL:   standardRateLimit(),
	perpetualUpdateLeverageDualModeEPL: standardRateLimit(),
	perpetualSubmitOrderEPL:            perpetualOrderplacementRateLimit(),
	perpetualGetOrdersEPL:              standardRateLimit(),
	perpetualSubmitBatchOrdersEPL:      perpetualOrderplacementRateLimit(),
	perpetualFetchOrderEPL:             standardRateLimit(),
	perpetualCancelOrderEPL:            orderCloseRateLimit(),
	perpetualAmendOrderEPL:             perpetualOrderplacementRateLimit(),
	perpetualTradingHistoryEPL:         standardRateLimit(),
	perpetualClosePositionEPL:          orderCloseRateLimit(),
	perpetualLiquidationHistoryEPL:     standardRateLimit(),
	perpetualCancelTriggerOrdersEPL:    orderCloseRateLimit(),
	perpetualSubmitTriggerOrderEPL:     perpetualOrderplacementRateLimit(),
	perpetualListOpenOrdersEPL:         standardRateLimit(),
	perpetualCancelOpenOrdersEPL:       orderCloseRateLimit(),
	perpetualGetTriggerOrderEPL:        standardRateLimit(),
	perpetualCancelTriggerOrderEPL:     orderCloseRateLimit(),
	perpetualCreateChaseOrderEPL:       perpetualOrderplacementRateLimit(),
	perpetualStopChaseOrderEPL:         orderCloseRateLimit(),
	perpetualStopAllChaseOrdersEPL:     orderCloseRateLimit(),
	perpetualGetChaseOrdersEPL:         standardRateLimit(),
	perpetualGetChaseOrderDetailEPL:    standardRateLimit(),

	deliveryAccountEPL:             standardRateLimit(),
	deliveryAccountBooksEPL:        standardRateLimit(),
	deliveryPositionsEPL:           standardRateLimit(),
	deliveryUpdateMarginEPL:        standardRateLimit(),
	deliveryUpdateLeverageEPL:      standardRateLimit(),
	deliverySubmitOrderEPL:         deliverySubmitCancelAmendRateLimit(),
	deliveryGetOrdersEPL:           standardRateLimit(),
	deliveryCancelOrdersEPL:        deliverySubmitCancelAmendRateLimit(),
	deliveryGetOrderEPL:            standardRateLimit(),
	deliveryCancelOrderEPL:         deliverySubmitCancelAmendRateLimit(),
	deliveryTradingHistoryEPL:      standardRateLimit(),
	deliveryCloseHistoryEPL:        standardRateLimit(),
	deliveryLiquidationHistoryEPL:  standardRateLimit(),
	deliverySettlementHistoryEPL:   standardRateLimit(),
	deliveryGetTriggerOrdersEPL:    standardRateLimit(),
	deliveryAutoOrdersEPL:          standardRateLimit(),
	deliveryCancelTriggerOrdersEPL: deliverySubmitCancelAmendRateLimit(),
	deliveryGetTriggerOrderEPL:     standardRateLimit(),
	deliveryCancelTriggerOrderEPL:  deliverySubmitCancelAmendRateLimit(),

	optionsSettlementsEPL:        standardRateLimit(),
	optionsAccountsEPL:           standardRateLimit(),
	optionsAccountBooksEPL:       standardRateLimit(),
	optionsPositions:             standardRateLimit(),
	optionsLiquidationHistoryEPL: standardRateLimit(),
	optionsSubmitOrderEPL:        optionsSubmitCancelAmendRateLimit(),
	optionsOrdersEPL:             standardRateLimit(),
	optionsCancelOrdersEPL:       optionsSubmitCancelAmendRateLimit(),
	optionsOrderEPL:              standardRateLimit(),
	optionsCancelOrderEPL:        optionsSubmitCancelAmendRateLimit(),
	optionsTradingHistoryEPL:     standardRateLimit(),

	privateUnifiedSpotEPL: standardRateLimit(),

	websocketRateLimitNotNeededEPL: nil, // no rate limit for certain websocket functions

	// Risk limits
	publicFuturesRiskTableEPL:       standardRateLimit(),
	publicFuturesRiskLimitTiersEPL:  standardRateLimit(),
	publicDeliveryRiskLimitTiersEPL: standardRateLimit(),
	unifiedUserRiskUnitDetailsEPL:   standardRateLimit(),
	deliveryUpdateRiskLimitEPL:      standardRateLimit(),
	perpetualUpdateRiskDualModeEPL:  standardRateLimit(),
	perpetualUpdateRiskEPL:          standardRateLimit(),

	// Alpha limits
	alphaAccountsEPL:    standardRateLimit(),
	alphaAccountBookEPL: standardRateLimit(),
	alphaCreateQuoteEPL: standardRateLimit(),
	alphaPlaceOrderEPL:  spotOrderPlacementRateLimit(),
	alphaListOrdersEPL:  standardRateLimit(),
	alphaGetOrderEPL:    standardRateLimit(),
	alphaCurrenciesEPL:  standardRateLimit(),
	alphaTickersEPL:     standardRateLimit(),

	// Unified limits
	unifiedInterestRecordsEPL:                   standardRateLimit(),
	unifiedRiskUnitsEPL:                         standardRateLimit(),
	unifiedUpdateUnifiedModeEPL:                 standardRateLimit(),
	unifiedGetUnifiedModeEPL:                    standardRateLimit(),
	unifiedEstimateRateEPL:                      standardRateLimit(),
	unifiedCurrencyDiscountTiersEPL:             standardRateLimit(),
	unifiedLoanMarginTiersEPL:                   standardRateLimit(),
	unifiedPortfolioCalculatorEPL:               standardRateLimit(),
	unifiedGetLeverageUserCurrencySettingEPL:    standardRateLimit(),
	unifiedCreateLeverageUserCurrencySettingEPL: standardRateLimit(),
	unifiedLeverageUserCurrencyConfigEPL:        standardRateLimit(),
	unifiedCurrenciesEPL:                        standardRateLimit(),
	unifiedHistoryLoanRateEPL:                   standardRateLimit(),
	unifiedCollateralCurrenciesEPL:              standardRateLimit(),
	unifiedEstimateQuickRepaymentEPL:            standardRateLimit(),
	unifiedQuickRepaymentEPL:                    standardRateLimit(),

	// Earn fixed-term limits
	earnFixedTermProductEPL:     standardRateLimit(),
	earnFixedTermProductListEPL: standardRateLimit(),

	// Loan multi-collateral limits
	loanMultiCollateralMortgageEPL:      standardRateLimit(),
	loanMultiCollateralCurrencyQuotaEPL: standardRateLimit(),
	loanMultiCollateralCurrentRateEPL:   standardRateLimit(),
	loanMultiCollateralCurrenciesEPL:    standardRateLimit(),
	loanMultiCollateralLtvEPL:           standardRateLimit(),
	loanMultiCollateralFixedRateEPL:     standardRateLimit(),

	// Earn uni limits
	earnUniCreateLendsEPL:     standardRateLimit(),
	earnUniGetLendsEPL:        standardRateLimit(),
	earnUniUpdateLendsEPL:     standardRateLimit(),
	earnUniLendRecordsEPL:     standardRateLimit(),
	earnUniInterestsEPL:       standardRateLimit(),
	earnUniInterestRecordsEPL: standardRateLimit(),
	earnUniInterestStatusEPL:  standardRateLimit(),
	earnUniChartEPL:           standardRateLimit(),
	earnUniRateEPL:            standardRateLimit(),
	earnUniCurrenciesEPL:      standardRateLimit(),
	earnUniCurrencyEPL:        standardRateLimit(),

	// Margin uni limits
	marginUniCreateLoansEPL:     standardRateLimit(),
	marginUniGetLoansEPL:        standardRateLimit(),
	marginUniLoanRecordsEPL:     standardRateLimit(),
	marginUniInterestRecordsEPL: standardRateLimit(),
	marginUniBorrowableEPL:      standardRateLimit(),
	marginUniCurrencyPairsEPL:   standardRateLimit(),
	marginUniCurrencyPairEPL:    standardRateLimit(),

	// TradFi limits
	tradfiUsersMT5AccountEPL:    standardRateLimit(),
	tradfiSymbolsDetailEPL:      standardRateLimit(),
	tradfiKlinesEPL:             standardRateLimit(),
	tradfiUsersEPL:              standardRateLimit(),
	tradfiUsersAssetsEPL:        standardRateLimit(),
	tradfiGetTransactionsEPL:    standardRateLimit(),
	tradfiCreateTransactionsEPL: standardRateLimit(),
	tradfiGetOrdersEPL:          standardRateLimit(),
	tradfiCreateOrdersEPL:       spotOrderPlacementRateLimit(),
	tradfiUpdateOrdersEPL:       spotOrderPlacementRateLimit(),
	tradfiDeleteOrdersEPL:       orderCloseRateLimit(),
	tradfiOrdersHistoryEPL:      standardRateLimit(),
	tradfiGetPositionsEPL:       standardRateLimit(),
	tradfiUpdatePositionsEPL:    standardRateLimit(),
	tradfiCreatePositionsEPL:    standardRateLimit(),
	tradfiPositionsHistoryEPL:   standardRateLimit(),
	tradfiSymbolsCategoriesEPL:  standardRateLimit(),
	tradfiSymbolsEPL:            standardRateLimit(),
	tradfiTickersEPL:            standardRateLimit(),

	// CrossEx limits
	crossexSymbolsEPL:                       standardRateLimit(),
	crossexRiskLimitEPL:                     standardRateLimit(),
	crossexTransfersCoinEPL:                 standardRateLimit(),
	crossexGetTransfersEPL:                  standardRateLimit(),
	crossexCreateTransfersEPL:               tenPer10SecondsRateLimit(),
	crossexCreateOrdersEPL:                  hundredPer10SecondsRateLimit(),
	crossexGetOrdersEPL:                     standardRateLimit(),
	crossexUpdateOrdersEPL:                  hundredPer10SecondsRateLimit(),
	crossexDeleteOrdersEPL:                  hundredPer10SecondsRateLimit(),
	crossexConvertQuoteEPL:                  hundredPerDayRateLimit(),
	crossexConvertOrdersEPL:                 tenPer10SecondsRateLimit(),
	crossexGetAccountsEPL:                   standardRateLimit(),
	crossexUpdateAccountsEPL:                hundredPer60SecondsRateLimit(),
	crossexGetPositionsLeverageEPL:          standardRateLimit(),
	crossexCreatePositionsLeverageEPL:       hundredPer10SecondsRateLimit(),
	crossexGetMarginPositionsLeverageEPL:    standardRateLimit(),
	crossexCreateMarginPositionsLeverageEPL: request.NewRateLimitWithWeight(time.Second*10, 100, 1),
	crossexPositionEPL:                      hundredPerDayRateLimit(),
	crossexInterestRateEPL:                  standardRateLimit(),
	crossexFeeEPL:                           standardRateLimit(),
	crossexPositionsEPL:                     standardRateLimit(),
	crossexMarginPositionsEPL:               standardRateLimit(),
	crossexADLRankEPL:                       standardRateLimit(),
	crossexOpenOrdersEPL:                    standardRateLimit(),
	crossexHistoryOrdersEPL:                 standardRateLimit(),
	crossexHistoryPositionsEPL:              standardRateLimit(),
	crossexHistoryMarginPositionsEPL:        standardRateLimit(),
	crossexHistoryMarginInterestsEPL:        standardRateLimit(),
	crossexHistoryTradesEPL:                 standardRateLimit(),
	crossexAccountBookEPL:                   standardRateLimit(),
	crossexCoinDiscountRateEPL:              standardRateLimit(),

	// P2P limits
	p2pAccountInfoEPL:           standardRateLimit(),
	p2pCounterpartyInfoEPL:      standardRateLimit(),
	p2pPaymentMethodsEPL:        standardRateLimit(),
	p2pSetWorkHoursEPL:          standardRateLimit(),
	p2pPendingTransactionsEPL:   standardRateLimit(),
	p2pCompletedTransactionsEPL: standardRateLimit(),
	p2pMyListEPL:                standardRateLimit(),
	p2pMyHistoryListEPL:         standardRateLimit(),
	p2pTransactionDetailsEPL:    standardRateLimit(),
	p2pConfirmPaymentEPL:        standardRateLimit(),
	p2pConfirmReceiptEPL:        standardRateLimit(),
	p2pCancelTransactionEPL:     standardRateLimit(),
	p2pPublishAdEPL:             standardRateLimit(),
	p2pUpdateAdStatusEPL:        standardRateLimit(),
	p2pAdDetailEPL:              standardRateLimit(),
	p2pMyAdsListEPL:             standardRateLimit(),
	p2pAdsListEPL:               standardRateLimit(),
	p2pChatHistoryEPL:           standardRateLimit(),
	p2pSendChatMessageEPL:       standardRateLimit(),
	p2pUploadChatFileEPL:        standardRateLimit(),

	// Bot limits
	botStrategyRecommendEPL:        standardRateLimit(),
	botSpotGridCreateEPL:           standardRateLimit(),
	botMarginGridCreateEPL:         standardRateLimit(),
	botInfiniteGridCreateEPL:       standardRateLimit(),
	botFuturesGridCreateEPL:        standardRateLimit(),
	botSpotMartingaleCreateEPL:     standardRateLimit(),
	botContractMartingaleCreateEPL: standardRateLimit(),
	botPortfolioRunningEPL:         standardRateLimit(),
	botPortfolioDetailEPL:          standardRateLimit(),
	botPortfolioStopEPL:            standardRateLimit(),

	// Rebate limits
	rebateAgencyTransactionHistoryEPL:  standardRateLimit(),
	rebateAgencyCommissionHistoryEPL:   standardRateLimit(),
	rebatePartnerTransactionHistoryEPL: standardRateLimit(),
	rebatePartnerCommissionHistoryEPL:  standardRateLimit(),
	rebatePartnerSubListEPL:            standardRateLimit(),
	rebateBrokerCommissionHistoryEPL:   standardRateLimit(),
	rebateBrokerTransactionHistoryEPL:  standardRateLimit(),
	rebateUserInfoEPL:                  standardRateLimit(),
	rebateUserSubRelationEPL:           standardRateLimit(),
	rebatePartnerApplicationsRecentEPL: standardRateLimit(),
	rebatePartnerEligibilityEPL:        standardRateLimit(),
	rebatePartnerDataAggregatedEPL:     standardRateLimit(),

	// OTC limits
	otcQuoteEPL:                    standardRateLimit(),
	otcOrderCreateEPL:              standardRateLimit(),
	otcStablecoinOrderCreateEPL:    standardRateLimit(),
	otcBankListEPL:                 standardRateLimit(),
	otcBankCreateEPL:               standardRateLimit(),
	otcBankDeleteEPL:               standardRateLimit(),
	otcBankSetDefaultEPL:           standardRateLimit(),
	otcBankSupplementChecklistEPL:  standardRateLimit(),
	otcBankPersonalSupplementEPL:   standardRateLimit(),
	otcBankEnterpriseSupplementEPL: standardRateLimit(),
	otcOrderPaidEPL:                standardRateLimit(),
	otcOrderCancelEPL:              standardRateLimit(),
	otcOrderListEPL:                standardRateLimit(),
	otcStablecoinOrderListEPL:      standardRateLimit(),
	otcOrderDetailEPL:              standardRateLimit(),
}

func hundredPerDayRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Hour*24, 100, 1)
}

func hundredPer10SecondsRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 100, 1)
}

func hundredPer60SecondsRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Minute, 100, 1)
}
func standardRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 200, 1)
}

func personalAccountRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 80, 1)
}

func orderCloseRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 200, 1)
}

func tenPer10SecondsRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 10, 1)
}

func spotOrderPlacementRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 10, 1)
}

func otherPrivateEndpointRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 150, 1)
}

func perpetualOrderplacementRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 100, 1)
}

func deliverySubmitCancelAmendRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 500, 1)
}

func optionsSubmitCancelAmendRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 200, 1)
}

func withdrawFromWalletRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*3, 1, 1)
}
