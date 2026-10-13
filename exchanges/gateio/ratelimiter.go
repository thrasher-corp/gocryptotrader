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
	publicInsuranceHistorySpotEPL

	publicUniCurrencyPairsMarginEPL
	publicUniCurrencyPairDetailMarginEPL
	publicIsolatedMarginPoolLoansEPL

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
	publicFuturesContractsAllEPL
	publicFuturesADLRiskStatesEPL
	publicBatchFundingRatesEPL

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
	walletGetTransferEPL
	walletSubAccountTransferEPL
	walletSubAccountTransferHistoryEPL
	walletSubAccountToSubAccountTransferEPL
	walletTransferOrderStatusEPL
	walletWithdrawStatusEPL
	walletSubAccountBalancesEPL
	walletSubAccountMarginBalancesEPL
	walletSubAccountFuturesBalancesEPL
	walletSubAccountCrossMarginBalancesEPL
	walletSavedAddressesEPL
	walletTradingFeeEPL
	walletTotalBalanceEPL
	walletSmallBalancesEPL
	walletConvertSmallBalancesEPL
	walletSmallBalanceHistoryEPL
	walletUIDTransferHistoryEPL
	walletLowCapExchangeListEPL
	walletWithdrawEPL
	walletUIDTransferEPL
	walletCancelWithdrawEPL

	subAccountCreateEPL
	subAccountListEPL
	subAccountGetEPL
	subAccountCreateKeyEPL
	subAccountListKeysEPL
	subAccountUpdateKeyEPL
	subAccountDeleteKeyEPL
	subAccountGetKeyEPL
	subAccountLockEPL
	subAccountUnlockEPL
	subAccountUnifiedModeEPL

	accountDetailEPL
	accountMainKeysEPL
	accountRateLimitEPL
	accountCreateSTPGroupEPL
	accountListSTPGroupsEPL
	accountListSTPGroupUsersEPL
	accountAddSTPGroupUsersEPL
	accountDeleteSTPGroupUsersEPL
	accountSetDebitFeeEPL
	accountGetDebitFeeEPL

	spotTradingFeeEPL
	spotAccountsEPL
	spotAccountBookEPL
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
	spotAmendBatchOrdersEPL
	spotCreateTriggerOrderEPL
	spotGetTriggerOrderListEPL
	spotCancelTriggerOrdersEPL
	spotGetTriggerOrderEPL
	spotCancelTriggerOrderEPL
	spotCreatePOVOrderEPL
	spotGetPOVOrdersEPL
	spotCancelPOVOrdersEPL
	spotGetPOVOrderEPL
	spotCancelPOVOrderEPL

	marginIsolatedAccountsEPL
	marginAccountBalanceEPL
	marginFundingAccountListEPL
	marginAutoRepayEPL
	marginGetAutoRepaySettingsEPL
	marginGetMaxTransferEPL
	marginUniBorrowableEPL
	marginUserLoanMarginTiersEPL
	marginMarketLoanMarginTiersEPL
	marginSetUserMarketLeverageEPL
	marginUserAccountListEPL
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
	marginUniEstimateRateEPL
	marginUniLoansEPL
	marginCreateUniLoanEPL
	marginUniLoanRecordsEPL
	marginUniInterestRecordsEPL
	unifiedInterestRecordsEPL

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
	perpetualAmendTriggerOrderEPL
	perpetualPositionsTimeRangeEPL
	perpetualGetLeverageEPL
	perpetualSetLeverageEPL
	perpetualSwitchMarginModeEPL
	perpetualSwitchDualModeMarginModeEPL
	perpetualSetPositionModeEPL
	perpetualCancelOrdersEPL
	perpetualGetOrdersByTimeRangeEPL
	perpetualTradingHistoryByTimeRangeEPL
	perpetualAutoDeleveragesEPL
	perpetualTradingFeeEPL
	perpetualCancelBatchOrdersEPL
	perpetualAmendBatchOrdersEPL
	perpetualSubmitBBOOrderEPL
	perpetualCreateTrailOrderEPL
	perpetualStopTrailOrderEPL
	perpetualStopAllTrailOrdersEPL
	perpetualGetTrailOrdersEPL
	perpetualGetTrailOrderDetailEPL
	perpetualUpdateTrailOrderEPL
	perpetualGetTrailOrderChangeLogEPL
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
	deliverySubmitTriggerOrderEPL
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
	optionsAmendOrderEPL
	optionsCountdownCancelEPL
	optionsTradingHistoryEPL
	optionsSetMMPEPL
	optionsGetMMPEPL
	optionsResetMMPEPL

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
	alphaTokensEPL

	// Unified EPLs
	unifiedBorrowOrRepayEPL
	unifiedLoansEPL
	unifiedLoanRecordsEPL
	unifiedUpdateUnifiedModeEPL
	unifiedGetUnifiedModeEPL
	unifiedEstimateRateEPL
	unifiedCurrencyDiscountTiersEPL
	unifiedLoanMarginTiersEPL
	unifiedPortfolioCalculatorEPL
	unifiedLeverageUserSettingEPL
	unifiedGetLeverageUserCurrencySettingEPL
	unifiedCreateLeverageUserCurrencySettingEPL
	unifiedLeverageUserCurrencyConfigEPL
	unifiedCurrenciesEPL
	unifiedHistoryLoanRateEPL
	unifiedCollateralCurrenciesEPL
	unifiedEstimateQuickRepaymentEPL
	unifiedQuickRepaymentEPL
	unifiedSetDeltaNeutralEPL
	unifiedGetDeltaNeutralEPL

	// Earn fixed-term EPLs
	earnFixedTermProductEPL
	earnFixedTermProductListEPL
	earnFixedTermLendEPL
	earnFixedTermLendsEPL
	earnFixedTermPreRedeemEPL
	earnFixedTermHistoryEPL

	// Earn dual investment EPLs
	earnDualInvestmentPlanEPL
	earnDualOrdersEPL
	earnDualPlaceOrderEPL
	earnDualBalanceEPL
	earnDualOrderRefundPreviewEPL
	earnDualOrderRefundEPL
	earnDualModifyOrderReinvestEPL
	earnDualProjectRecommendEPL

	// Earn staking EPLs
	earnStakingCoinsEPL
	earnStakingSwapEPL
	earnStakingOrderListEPL
	earnStakingAwardListEPL
	earnStakingAssetsEPL

	// Earn auto invest EPLs
	earnAutoInvestCreatePlanEPL
	earnAutoInvestUpdatePlanEPL
	earnAutoInvestStopPlanEPL
	earnAutoInvestAddPositionEPL
	earnAutoInvestCoinsEPL
	earnAutoInvestMinAmountEPL
	earnAutoInvestPlanRecordsEPL
	earnAutoInvestOrdersEPL
	earnAutoInvestConfigEPL
	earnAutoInvestPlanDetailEPL
	earnAutoInvestPlanListEPL

	// Loan multi-collateral EPLs
	loanMultiCollateralMortgageEPL
	loanMultiCollateralCurrencyQuotaEPL
	loanMultiCollateralCurrentRateEPL
	loanMultiCollateralCurrenciesEPL
	loanMultiCollateralLtvEPL
	loanMultiCollateralFixedRateEPL
	loanMultiCollateralCreateOrderEPL
	loanMultiCollateralOrdersEPL
	loanMultiCollateralOrderDetailEPL
	loanMultiCollateralRepayEPL
	loanMultiCollateralRepayRecordsEPL
	loanMultiCollateralOperateEPL

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

	// TradFi EPLs
	tradfiUsersMT5AccountEPL
	tradfiSymbolsDetailEPL
	tradfiSymbolsCommissionsEPL
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
	tradfiOrderLogEPL
	tradfiGetPositionsEPL
	tradfiUpdatePositionsEPL
	tradfiCreatePositionsEPL
	tradfiPositionsHistoryEPL
	tradfiSymbolsCategoriesEPL
	tradfiSymbolsEPL
	tradfiTickersEPL

	// Stock EPLs
	stockUserAssetsEPL
	stockSymbolsEPL
	stockSymbolDetailsEPL
	stockOrderbookEPL
	stockCreateOrderEPL
	stockOpenOrdersEPL
	stockCancelAllOrdersEPL
	stockOrderHistoryEPL
	stockUpdateOrderEPL
	stockCancelOrderEPL
	stockPositionsEPL
	stockClosePositionEPL
	stockCreateTransactionEPL
	stockTransactionsEPL
	stockExchangesEPL
	stockFeeRateEPL

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
	crossexCancelBatchOrdersEPL
	crossexConvertQuoteEPL
	crossexConvertOrdersEPL
	crossexGetAccountsEPL
	crossexUpdateAccountsEPL
	crossexGetPositionsLeverageEPL
	crossexCreatePositionsLeverageEPL
	crossexGetMarginPositionsLeverageEPL
	crossexCreateMarginPositionsLeverageEPL
	crossexPositionEPL
	crossexUpdatePositionsMarginModeEPL
	crossexGetPositionsMarginModeEPL
	crossexUpdatePositionsMarginEPL
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
	crossexMarketTickersEPL
	crossexMarketFundingInfoEPL

	// P2P EPLs
	p2pAccountInfoEPL
	p2pCounterpartyInfoEPL
	p2pPaymentMethodsEPL
	p2pSetWorkHoursEPL
	p2pPendingTransactionsEPL
	p2pCompletedTransactionsEPL
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

	// Announcement EPLs
	publicAnnouncementArticlesEPL

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
	otcUploadPreUploadEPL
	otcOrderPaidEPL
	otcOrderCancelEPL
	otcOrderListEPL
	otcStablecoinOrderListEPL
	// Must remain last because TestRateLimits uses this as its inclusive upper bound.
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
	publicInsuranceHistorySpotEPL:   standardRateLimit(),

	publicUniCurrencyPairsMarginEPL:      standardRateLimit(),
	publicUniCurrencyPairDetailMarginEPL: standardRateLimit(),
	publicIsolatedMarginPoolLoansEPL:     standardRateLimit(),

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
	publicFuturesContractsAllEPL:   standardRateLimit(),
	publicFuturesADLRiskStatesEPL:  standardRateLimit(),
	publicBatchFundingRatesEPL:     standardRateLimit(),

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
	walletGetTransferEPL:                    standardRateLimit(),
	walletSubAccountTransferEPL:             personalAccountRateLimit(),
	walletSubAccountTransferHistoryEPL:      standardRateLimit(),
	walletSubAccountToSubAccountTransferEPL: personalAccountRateLimit(),
	walletTransferOrderStatusEPL:            standardRateLimit(),
	walletWithdrawStatusEPL:                 standardRateLimit(),
	walletSubAccountBalancesEPL:             personalAccountRateLimit(),
	walletSubAccountMarginBalancesEPL:       personalAccountRateLimit(),
	walletSubAccountFuturesBalancesEPL:      personalAccountRateLimit(),
	walletSubAccountCrossMarginBalancesEPL:  personalAccountRateLimit(),
	walletSavedAddressesEPL:                 standardRateLimit(),
	walletTradingFeeEPL:                     standardRateLimit(),
	walletTotalBalanceEPL:                   personalAccountRateLimit(),
	walletSmallBalancesEPL:                  standardRateLimit(),
	walletConvertSmallBalancesEPL:           standardRateLimit(),
	walletSmallBalanceHistoryEPL:            standardRateLimit(),
	walletUIDTransferHistoryEPL:             standardRateLimit(),
	walletLowCapExchangeListEPL:             standardRateLimit(),
	walletWithdrawEPL:                       withdrawFromWalletRateLimit(),
	walletUIDTransferEPL:                    uidTransferRateLimit(),
	walletCancelWithdrawEPL:                 standardRateLimit(),

	subAccountCreateEPL:      personalAccountRateLimit(),
	subAccountListEPL:        personalAccountRateLimit(),
	subAccountGetEPL:         personalAccountRateLimit(),
	subAccountCreateKeyEPL:   personalAccountRateLimit(),
	subAccountListKeysEPL:    personalAccountRateLimit(),
	subAccountUpdateKeyEPL:   personalAccountRateLimit(),
	subAccountDeleteKeyEPL:   personalAccountRateLimit(),
	subAccountGetKeyEPL:      personalAccountRateLimit(),
	subAccountLockEPL:        personalAccountRateLimit(),
	subAccountUnlockEPL:      personalAccountRateLimit(),
	subAccountUnifiedModeEPL: personalAccountRateLimit(),

	accountDetailEPL:              otherPrivateEndpointRateLimit(),
	accountMainKeysEPL:            otherPrivateEndpointRateLimit(),
	accountRateLimitEPL:           otherPrivateEndpointRateLimit(),
	accountCreateSTPGroupEPL:      otherPrivateEndpointRateLimit(),
	accountListSTPGroupsEPL:       otherPrivateEndpointRateLimit(),
	accountListSTPGroupUsersEPL:   otherPrivateEndpointRateLimit(),
	accountAddSTPGroupUsersEPL:    otherPrivateEndpointRateLimit(),
	accountDeleteSTPGroupUsersEPL: otherPrivateEndpointRateLimit(),
	accountSetDebitFeeEPL:         otherPrivateEndpointRateLimit(),
	accountGetDebitFeeEPL:         otherPrivateEndpointRateLimit(),

	spotTradingFeeEPL:          standardRateLimit(),
	spotAccountsEPL:            standardRateLimit(),
	spotAccountBookEPL:         standardRateLimit(),
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
	spotAmendBatchOrdersEPL:    spotOrderPlacementRateLimit(),
	spotCreateTriggerOrderEPL:  spotOrderPlacementRateLimit(),
	spotGetTriggerOrderListEPL: standardRateLimit(),
	spotCancelTriggerOrdersEPL: orderCloseRateLimit(),
	spotGetTriggerOrderEPL:     standardRateLimit(),
	spotCancelTriggerOrderEPL:  orderCloseRateLimit(),
	spotCreatePOVOrderEPL:      spotOrderPlacementRateLimit(),
	spotGetPOVOrdersEPL:        standardRateLimit(),
	spotCancelPOVOrdersEPL:     orderCloseRateLimit(),
	spotGetPOVOrderEPL:         standardRateLimit(),
	spotCancelPOVOrderEPL:      orderCloseRateLimit(),

	marginIsolatedAccountsEPL:           otherPrivateEndpointRateLimit(),
	marginAccountBalanceEPL:             otherPrivateEndpointRateLimit(),
	marginFundingAccountListEPL:         otherPrivateEndpointRateLimit(),
	marginAutoRepayEPL:                  otherPrivateEndpointRateLimit(),
	marginGetAutoRepaySettingsEPL:       otherPrivateEndpointRateLimit(),
	marginGetMaxTransferEPL:             otherPrivateEndpointRateLimit(),
	marginUniBorrowableEPL:              otherPrivateEndpointRateLimit(),
	marginUserLoanMarginTiersEPL:        otherPrivateEndpointRateLimit(),
	marginMarketLoanMarginTiersEPL:      standardRateLimit(),
	marginSetUserMarketLeverageEPL:      otherPrivateEndpointRateLimit(),
	marginUserAccountListEPL:            otherPrivateEndpointRateLimit(),
	marginSupportedCurrencyCrossListEPL: standardRateLimit(),
	marginSupportedCurrencyCrossEPL:     standardRateLimit(),
	marginAccountsEPL:                   otherPrivateEndpointRateLimit(),
	marginAccountHistoryEPL:             otherPrivateEndpointRateLimit(),
	marginCreateCrossBorrowLoanEPL:      otherPrivateEndpointRateLimit(),
	marginExecuteRepaymentsEPL:          otherPrivateEndpointRateLimit(),
	marginGetCrossMarginRepaymentsEPL:   otherPrivateEndpointRateLimit(),
	marginGetMaxTransferCrossEPL:        otherPrivateEndpointRateLimit(),
	marginGetMaxBorrowCrossEPL:          otherPrivateEndpointRateLimit(),
	marginGetCrossBorrowHistoryEPL:      otherPrivateEndpointRateLimit(),
	marginGetBorrowEPL:                  otherPrivateEndpointRateLimit(),
	marginUniEstimateRateEPL:            otherPrivateEndpointRateLimit(),
	marginUniLoansEPL:                   otherPrivateEndpointRateLimit(),
	marginCreateUniLoanEPL:              otherPrivateEndpointRateLimit(),
	marginUniLoanRecordsEPL:             otherPrivateEndpointRateLimit(),
	marginUniInterestRecordsEPL:         otherPrivateEndpointRateLimit(),
	unifiedInterestRecordsEPL:           otherPrivateEndpointRateLimit(),

	flashSwapOrderEPL:   otherPrivateEndpointRateLimit(),
	flashGetOrdersEPL:   otherPrivateEndpointRateLimit(),
	flashGetOrderEPL:    otherPrivateEndpointRateLimit(),
	flashOrderReviewEPL: otherPrivateEndpointRateLimit(),

	perpetualAccountEPL:                   standardRateLimit(),
	perpetualAccountBooksEPL:              standardRateLimit(),
	perpetualPositionsEPL:                 standardRateLimit(),
	perpetualPositionEPL:                  standardRateLimit(),
	perpetualUpdateMarginEPL:              standardRateLimit(),
	perpetualUpdateLeverageEPL:            standardRateLimit(),
	perpetualToggleDualModeEPL:            standardRateLimit(),
	perpetualPositionsDualModeEPL:         standardRateLimit(),
	perpetualUpdateMarginDualModeEPL:      standardRateLimit(),
	perpetualUpdateLeverageDualModeEPL:    standardRateLimit(),
	perpetualSubmitOrderEPL:               perpetualOrderplacementRateLimit(),
	perpetualGetOrdersEPL:                 standardRateLimit(),
	perpetualSubmitBatchOrdersEPL:         perpetualOrderplacementRateLimit(),
	perpetualFetchOrderEPL:                standardRateLimit(),
	perpetualCancelOrderEPL:               orderCloseRateLimit(),
	perpetualAmendOrderEPL:                perpetualOrderplacementRateLimit(),
	perpetualTradingHistoryEPL:            standardRateLimit(),
	perpetualClosePositionEPL:             standardRateLimit(),
	perpetualLiquidationHistoryEPL:        standardRateLimit(),
	perpetualCancelTriggerOrdersEPL:       orderCloseRateLimit(),
	perpetualSubmitTriggerOrderEPL:        perpetualOrderplacementRateLimit(),
	perpetualListOpenOrdersEPL:            standardRateLimit(),
	perpetualCancelOpenOrdersEPL:          orderCloseRateLimit(),
	perpetualGetTriggerOrderEPL:           standardRateLimit(),
	perpetualCancelTriggerOrderEPL:        orderCloseRateLimit(),
	perpetualAmendTriggerOrderEPL:         perpetualOrderplacementRateLimit(),
	perpetualPositionsTimeRangeEPL:        standardRateLimit(),
	perpetualGetLeverageEPL:               standardRateLimit(),
	perpetualSetLeverageEPL:               standardRateLimit(),
	perpetualSwitchMarginModeEPL:          standardRateLimit(),
	perpetualSwitchDualModeMarginModeEPL:  standardRateLimit(),
	perpetualSetPositionModeEPL:           standardRateLimit(),
	perpetualCancelOrdersEPL:              orderCloseRateLimit(),
	perpetualGetOrdersByTimeRangeEPL:      standardRateLimit(),
	perpetualTradingHistoryByTimeRangeEPL: standardRateLimit(),
	perpetualAutoDeleveragesEPL:           standardRateLimit(),
	perpetualTradingFeeEPL:                standardRateLimit(),
	perpetualCancelBatchOrdersEPL:         orderCloseRateLimit(),
	perpetualAmendBatchOrdersEPL:          perpetualOrderplacementRateLimit(),
	perpetualSubmitBBOOrderEPL:            perpetualOrderplacementRateLimit(),
	perpetualCreateTrailOrderEPL:          perpetualOrderplacementRateLimit(),
	perpetualStopTrailOrderEPL:            orderCloseRateLimit(),
	perpetualStopAllTrailOrdersEPL:        orderCloseRateLimit(),
	perpetualGetTrailOrdersEPL:            standardRateLimit(),
	perpetualGetTrailOrderDetailEPL:       standardRateLimit(),
	perpetualUpdateTrailOrderEPL:          perpetualOrderplacementRateLimit(),
	perpetualGetTrailOrderChangeLogEPL:    standardRateLimit(),
	perpetualCreateChaseOrderEPL:          perpetualOrderplacementRateLimit(),
	perpetualStopChaseOrderEPL:            orderCloseRateLimit(),
	perpetualStopAllChaseOrdersEPL:        orderCloseRateLimit(),
	perpetualGetChaseOrdersEPL:            standardRateLimit(),
	perpetualGetChaseOrderDetailEPL:       standardRateLimit(),

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
	deliverySubmitTriggerOrderEPL:  standardRateLimit(),
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
	optionsAmendOrderEPL:         optionsSubmitCancelAmendRateLimit(),
	optionsCountdownCancelEPL:    standardRateLimit(),
	optionsTradingHistoryEPL:     standardRateLimit(),
	optionsSetMMPEPL:             standardRateLimit(),
	optionsGetMMPEPL:             standardRateLimit(),
	optionsResetMMPEPL:           standardRateLimit(),

	privateUnifiedSpotEPL: otherPrivateEndpointRateLimit(),

	websocketRateLimitNotNeededEPL: nil, // no rate limit for certain websocket functions

	// Risk limits
	publicFuturesRiskTableEPL:       standardRateLimit(),
	publicFuturesRiskLimitTiersEPL:  standardRateLimit(),
	publicDeliveryRiskLimitTiersEPL: standardRateLimit(),
	unifiedUserRiskUnitDetailsEPL:   otherPrivateEndpointRateLimit(),
	deliveryUpdateRiskLimitEPL:      standardRateLimit(),
	perpetualUpdateRiskDualModeEPL:  standardRateLimit(),
	perpetualUpdateRiskEPL:          standardRateLimit(),

	// Alpha limits
	alphaAccountsEPL:    standardRateLimit(),
	alphaAccountBookEPL: standardRateLimit(),
	alphaCreateQuoteEPL: spotOrderPlacementRateLimit(),
	alphaPlaceOrderEPL:  fivePerSecondRateLimit(),
	alphaListOrdersEPL:  standardRateLimit(),
	alphaGetOrderEPL:    standardRateLimit(),
	alphaCurrenciesEPL:  standardRateLimit(),
	alphaTickersEPL:     standardRateLimit(),
	alphaTokensEPL:      standardRateLimit(),

	// Unified limits
	unifiedBorrowOrRepayEPL:                     fifteenPer10SecondsRateLimit(),
	unifiedLoansEPL:                             otherPrivateEndpointRateLimit(),
	unifiedLoanRecordsEPL:                       otherPrivateEndpointRateLimit(),
	unifiedUpdateUnifiedModeEPL:                 otherPrivateEndpointRateLimit(),
	unifiedGetUnifiedModeEPL:                    otherPrivateEndpointRateLimit(),
	unifiedEstimateRateEPL:                      otherPrivateEndpointRateLimit(),
	unifiedCurrencyDiscountTiersEPL:             standardRateLimit(),
	unifiedLoanMarginTiersEPL:                   standardRateLimit(),
	unifiedPortfolioCalculatorEPL:               standardRateLimit(),
	unifiedLeverageUserSettingEPL:               otherPrivateEndpointRateLimit(),
	unifiedGetLeverageUserCurrencySettingEPL:    otherPrivateEndpointRateLimit(),
	unifiedCreateLeverageUserCurrencySettingEPL: otherPrivateEndpointRateLimit(),
	unifiedLeverageUserCurrencyConfigEPL:        otherPrivateEndpointRateLimit(),
	unifiedCurrenciesEPL:                        standardRateLimit(),
	unifiedHistoryLoanRateEPL:                   standardRateLimit(),
	unifiedCollateralCurrenciesEPL:              otherPrivateEndpointRateLimit(),
	unifiedEstimateQuickRepaymentEPL:            otherPrivateEndpointRateLimit(),
	unifiedQuickRepaymentEPL:                    otherPrivateEndpointRateLimit(),
	unifiedSetDeltaNeutralEPL:                   otherPrivateEndpointRateLimit(),
	unifiedGetDeltaNeutralEPL:                   otherPrivateEndpointRateLimit(),

	// Earn fixed-term limits; private earning and collateral endpoints fall under the 150r/10s "Other Private endpoints" rule
	earnFixedTermProductEPL:     standardRateLimit(),
	earnFixedTermProductListEPL: standardRateLimit(),
	earnFixedTermLendEPL:        otherPrivateEndpointRateLimit(),
	earnFixedTermLendsEPL:       otherPrivateEndpointRateLimit(),
	earnFixedTermPreRedeemEPL:   otherPrivateEndpointRateLimit(),
	earnFixedTermHistoryEPL:     otherPrivateEndpointRateLimit(),

	// Earn dual investment limits
	earnDualInvestmentPlanEPL:      standardRateLimit(),
	earnDualOrdersEPL:              otherPrivateEndpointRateLimit(),
	earnDualPlaceOrderEPL:          otherPrivateEndpointRateLimit(),
	earnDualBalanceEPL:             otherPrivateEndpointRateLimit(),
	earnDualOrderRefundPreviewEPL:  otherPrivateEndpointRateLimit(),
	earnDualOrderRefundEPL:         otherPrivateEndpointRateLimit(),
	earnDualModifyOrderReinvestEPL: otherPrivateEndpointRateLimit(),
	earnDualProjectRecommendEPL:    otherPrivateEndpointRateLimit(),

	// Earn staking limits
	earnStakingCoinsEPL:     otherPrivateEndpointRateLimit(),
	earnStakingSwapEPL:      otherPrivateEndpointRateLimit(),
	earnStakingOrderListEPL: otherPrivateEndpointRateLimit(),
	earnStakingAwardListEPL: otherPrivateEndpointRateLimit(),
	earnStakingAssetsEPL:    otherPrivateEndpointRateLimit(),

	// Earn auto invest limits
	earnAutoInvestCreatePlanEPL:  otherPrivateEndpointRateLimit(),
	earnAutoInvestUpdatePlanEPL:  otherPrivateEndpointRateLimit(),
	earnAutoInvestStopPlanEPL:    otherPrivateEndpointRateLimit(),
	earnAutoInvestAddPositionEPL: otherPrivateEndpointRateLimit(),
	earnAutoInvestCoinsEPL:       otherPrivateEndpointRateLimit(),
	earnAutoInvestMinAmountEPL:   otherPrivateEndpointRateLimit(),
	earnAutoInvestPlanRecordsEPL: otherPrivateEndpointRateLimit(),
	earnAutoInvestOrdersEPL:      otherPrivateEndpointRateLimit(),
	earnAutoInvestConfigEPL:      otherPrivateEndpointRateLimit(),
	earnAutoInvestPlanDetailEPL:  otherPrivateEndpointRateLimit(),
	earnAutoInvestPlanListEPL:    otherPrivateEndpointRateLimit(),

	// Loan multi-collateral limits
	loanMultiCollateralMortgageEPL:      otherPrivateEndpointRateLimit(),
	loanMultiCollateralCurrencyQuotaEPL: otherPrivateEndpointRateLimit(),
	loanMultiCollateralCurrentRateEPL:   standardRateLimit(),
	loanMultiCollateralCurrenciesEPL:    standardRateLimit(),
	loanMultiCollateralLtvEPL:           standardRateLimit(),
	loanMultiCollateralFixedRateEPL:     standardRateLimit(),
	loanMultiCollateralCreateOrderEPL:   otherPrivateEndpointRateLimit(),
	loanMultiCollateralOrdersEPL:        otherPrivateEndpointRateLimit(),
	loanMultiCollateralOrderDetailEPL:   otherPrivateEndpointRateLimit(),
	loanMultiCollateralRepayEPL:         otherPrivateEndpointRateLimit(),
	loanMultiCollateralRepayRecordsEPL:  otherPrivateEndpointRateLimit(),
	loanMultiCollateralOperateEPL:       otherPrivateEndpointRateLimit(),

	// Earn uni limits
	earnUniCreateLendsEPL:     otherPrivateEndpointRateLimit(),
	earnUniGetLendsEPL:        otherPrivateEndpointRateLimit(),
	earnUniUpdateLendsEPL:     otherPrivateEndpointRateLimit(),
	earnUniLendRecordsEPL:     otherPrivateEndpointRateLimit(),
	earnUniInterestsEPL:       otherPrivateEndpointRateLimit(),
	earnUniInterestRecordsEPL: otherPrivateEndpointRateLimit(),
	earnUniInterestStatusEPL:  otherPrivateEndpointRateLimit(),
	earnUniChartEPL:           otherPrivateEndpointRateLimit(),
	earnUniRateEPL:            otherPrivateEndpointRateLimit(),
	earnUniCurrenciesEPL:      standardRateLimit(),
	earnUniCurrencyEPL:        standardRateLimit(),

	// TradFi limits
	tradfiUsersMT5AccountEPL:    standardRateLimit(),
	tradfiSymbolsDetailEPL:      standardRateLimit(),
	tradfiSymbolsCommissionsEPL: standardRateLimit(),
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
	tradfiOrderLogEPL:           standardRateLimit(),
	tradfiGetPositionsEPL:       standardRateLimit(),
	tradfiUpdatePositionsEPL:    standardRateLimit(),
	tradfiCreatePositionsEPL:    standardRateLimit(),
	tradfiPositionsHistoryEPL:   standardRateLimit(),
	tradfiSymbolsCategoriesEPL:  standardRateLimit(),
	tradfiSymbolsEPL:            standardRateLimit(),
	tradfiTickersEPL:            standardRateLimit(),

	// Stock limits, which the API documents as 5 requests per second for each endpoint
	stockUserAssetsEPL:        fivePerSecondRateLimit(),
	stockSymbolsEPL:           fivePerSecondRateLimit(),
	stockSymbolDetailsEPL:     fivePerSecondRateLimit(),
	stockOrderbookEPL:         fivePerSecondRateLimit(),
	stockCreateOrderEPL:       fivePerSecondRateLimit(),
	stockOpenOrdersEPL:        fivePerSecondRateLimit(),
	stockCancelAllOrdersEPL:   fivePerSecondRateLimit(),
	stockOrderHistoryEPL:      fivePerSecondRateLimit(),
	stockUpdateOrderEPL:       fivePerSecondRateLimit(),
	stockCancelOrderEPL:       fivePerSecondRateLimit(),
	stockPositionsEPL:         fivePerSecondRateLimit(),
	stockClosePositionEPL:     fivePerSecondRateLimit(),
	stockCreateTransactionEPL: fivePerSecondRateLimit(),
	stockTransactionsEPL:      fivePerSecondRateLimit(),
	stockExchangesEPL:         fivePerSecondRateLimit(),
	stockFeeRateEPL:           fivePerSecondRateLimit(),

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
	crossexCancelBatchOrdersEPL:             hundredPer10SecondsRateLimit(),
	crossexConvertQuoteEPL:                  hundredPerDayRateLimit(),
	crossexConvertOrdersEPL:                 tenPer10SecondsRateLimit(),
	crossexGetAccountsEPL:                   standardRateLimit(),
	crossexUpdateAccountsEPL:                hundredPer60SecondsRateLimit(),
	crossexGetPositionsLeverageEPL:          standardRateLimit(),
	crossexCreatePositionsLeverageEPL:       hundredPer10SecondsRateLimit(),
	crossexGetMarginPositionsLeverageEPL:    standardRateLimit(),
	crossexCreateMarginPositionsLeverageEPL: hundredPer10SecondsRateLimit(),
	crossexPositionEPL:                      hundredPerDayRateLimit(),
	crossexUpdatePositionsMarginModeEPL:     hundredPer10SecondsRateLimit(),
	crossexGetPositionsMarginModeEPL:        standardRateLimit(),
	crossexUpdatePositionsMarginEPL:         hundredPer10SecondsRateLimit(),
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
	crossexMarketTickersEPL:                 request.NewRateLimitWithWeight(time.Second, 1, 1),
	crossexMarketFundingInfoEPL:             request.NewRateLimitWithWeight(time.Second, 1, 1),

	// P2P limits
	p2pAccountInfoEPL:           otherPrivateEndpointRateLimit(),
	p2pCounterpartyInfoEPL:      otherPrivateEndpointRateLimit(),
	p2pPaymentMethodsEPL:        otherPrivateEndpointRateLimit(),
	p2pSetWorkHoursEPL:          otherPrivateEndpointRateLimit(),
	p2pPendingTransactionsEPL:   otherPrivateEndpointRateLimit(),
	p2pCompletedTransactionsEPL: otherPrivateEndpointRateLimit(),
	p2pTransactionDetailsEPL:    otherPrivateEndpointRateLimit(),
	p2pConfirmPaymentEPL:        otherPrivateEndpointRateLimit(),
	p2pConfirmReceiptEPL:        otherPrivateEndpointRateLimit(),
	p2pCancelTransactionEPL:     otherPrivateEndpointRateLimit(),
	p2pPublishAdEPL:             otherPrivateEndpointRateLimit(),
	p2pUpdateAdStatusEPL:        otherPrivateEndpointRateLimit(),
	p2pAdDetailEPL:              otherPrivateEndpointRateLimit(),
	p2pMyAdsListEPL:             otherPrivateEndpointRateLimit(),
	p2pAdsListEPL:               otherPrivateEndpointRateLimit(),
	p2pChatHistoryEPL:           otherPrivateEndpointRateLimit(),
	p2pSendChatMessageEPL:       otherPrivateEndpointRateLimit(),
	p2pUploadChatFileEPL:        otherPrivateEndpointRateLimit(),

	// Bot limits
	botStrategyRecommendEPL:        otherPrivateEndpointRateLimit(),
	botSpotGridCreateEPL:           otherPrivateEndpointRateLimit(),
	botMarginGridCreateEPL:         otherPrivateEndpointRateLimit(),
	botInfiniteGridCreateEPL:       otherPrivateEndpointRateLimit(),
	botFuturesGridCreateEPL:        otherPrivateEndpointRateLimit(),
	botSpotMartingaleCreateEPL:     otherPrivateEndpointRateLimit(),
	botContractMartingaleCreateEPL: otherPrivateEndpointRateLimit(),
	botPortfolioRunningEPL:         otherPrivateEndpointRateLimit(),
	botPortfolioDetailEPL:          otherPrivateEndpointRateLimit(),
	botPortfolioStopEPL:            otherPrivateEndpointRateLimit(),

	// Announcement limits
	publicAnnouncementArticlesEPL: standardRateLimit(),

	// Rebate limits
	rebateAgencyTransactionHistoryEPL:  otherPrivateEndpointRateLimit(),
	rebateAgencyCommissionHistoryEPL:   otherPrivateEndpointRateLimit(),
	rebatePartnerTransactionHistoryEPL: otherPrivateEndpointRateLimit(),
	rebatePartnerCommissionHistoryEPL:  otherPrivateEndpointRateLimit(),
	rebatePartnerSubListEPL:            otherPrivateEndpointRateLimit(),
	rebateBrokerCommissionHistoryEPL:   otherPrivateEndpointRateLimit(),
	rebateBrokerTransactionHistoryEPL:  otherPrivateEndpointRateLimit(),
	rebateUserInfoEPL:                  otherPrivateEndpointRateLimit(),
	rebateUserSubRelationEPL:           otherPrivateEndpointRateLimit(),
	rebatePartnerApplicationsRecentEPL: otherPrivateEndpointRateLimit(),
	rebatePartnerEligibilityEPL:        otherPrivateEndpointRateLimit(),
	rebatePartnerDataAggregatedEPL:     otherPrivateEndpointRateLimit(),

	// OTC limits
	otcQuoteEPL:                    otherPrivateEndpointRateLimit(),
	otcOrderCreateEPL:              otherPrivateEndpointRateLimit(),
	otcStablecoinOrderCreateEPL:    otherPrivateEndpointRateLimit(),
	otcBankListEPL:                 otherPrivateEndpointRateLimit(),
	otcBankCreateEPL:               otherPrivateEndpointRateLimit(),
	otcBankDeleteEPL:               otherPrivateEndpointRateLimit(),
	otcBankSetDefaultEPL:           otherPrivateEndpointRateLimit(),
	otcBankSupplementChecklistEPL:  otherPrivateEndpointRateLimit(),
	otcBankPersonalSupplementEPL:   otherPrivateEndpointRateLimit(),
	otcBankEnterpriseSupplementEPL: otherPrivateEndpointRateLimit(),
	otcUploadPreUploadEPL:          otherPrivateEndpointRateLimit(),
	otcOrderPaidEPL:                otherPrivateEndpointRateLimit(),
	otcOrderCancelEPL:              otherPrivateEndpointRateLimit(),
	otcOrderListEPL:                otherPrivateEndpointRateLimit(),
	otcStablecoinOrderListEPL:      otherPrivateEndpointRateLimit(),
	otcOrderDetailEPL:              otherPrivateEndpointRateLimit(),
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

func fifteenPer10SecondsRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 15, 1)
}

func spotOrderPlacementRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 10, 1)
}

func fivePerSecondRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second, 5, 1)
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

func uidTransferRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*10, 1, 1)
}

func withdrawFromWalletRateLimit() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(time.Second*3, 1, 1)
}
