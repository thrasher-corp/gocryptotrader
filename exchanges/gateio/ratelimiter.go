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
	perpetualUpdateRiskEPL
	perpetualToggleDualModeEPL
	perpetualPositionsDualModeEPL
	perpetualUpdateMarginDualModeEPL
	perpetualUpdateLeverageDualModeEPL
	perpetualUpdateRiskDualModeEPL
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

	deliveryAccountEPL
	deliveryAccountBooksEPL
	deliveryPositionsEPL
	deliveryUpdateMarginEPL
	deliveryUpdateLeverageEPL
	deliveryUpdateRiskLimitEPL
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
	perpetualUpdateRiskEPL:             standardRateLimit(),
	perpetualToggleDualModeEPL:         standardRateLimit(),
	perpetualPositionsDualModeEPL:      standardRateLimit(),
	perpetualUpdateMarginDualModeEPL:   standardRateLimit(),
	perpetualUpdateLeverageDualModeEPL: standardRateLimit(),
	perpetualUpdateRiskDualModeEPL:     standardRateLimit(),
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

	deliveryAccountEPL:             standardRateLimit(),
	deliveryAccountBooksEPL:        standardRateLimit(),
	deliveryPositionsEPL:           standardRateLimit(),
	deliveryUpdateMarginEPL:        standardRateLimit(),
	deliveryUpdateLeverageEPL:      standardRateLimit(),
	deliveryUpdateRiskLimitEPL:     standardRateLimit(),
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
