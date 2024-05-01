package okcoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Interval instances
const (
	oneSecondInterval   = time.Second
	twoSecondsInterval  = time.Second * 2
	fiveSecondsInterval = time.Second * 5
)

// Rate of requests per interval for each end point
const (
	placeTradeOrderRate                                = 60
	placeTradeMultipleOrdersRate                       = 300
	cancelTradeOrderRate                               = 60
	cancelMultipleOrderRate                            = 300
	amendTradeOrderRate                                = 60
	amendMultipleOrdersRate                            = 300
	getOrderDetailsRate                                = 60
	getOrderListRate                                   = 60
	getOrderHistoryRate                                = 40
	getOrderhistory3MonthsRate                         = 20
	getTransactionDetails3DaysRate                     = 60
	getTransactionDetails3MonthsRate                   = 10
	placeAlgoOrderRate                                 = 20
	cancelAlgoOrderRate                                = 20
	cancelAdvancedAlgoOrderRate                        = 20
	getAlgoOrderListRate                               = 20
	getAlgoOrderHistoryRate                            = 20
	getFundingCurrenciesRate                           = 6
	getFundingAccountBalanceRate                       = 6
	getAccountAssetValuationRate                       = 1
	fundingTransferRate                                = 1
	getFundsTransferStateRate                          = 1
	assetBillsDetailRate                               = 6
	lightningDepositsRate                              = 2
	getAssetDepositAddressRate                         = 6
	getDepositHistoryRate                              = 6
	postWithdrawalRate                                 = 6
	postLightningWithdrawalRate                        = 2
	cancelWithdrawalRate                               = 6
	getAssetWithdrawalHistoryRate                      = 6
	getAccountBalanceRate                              = 10
	getBillsDetailLast3MonthRate                       = 6
	getBillsDetailRate                                 = 6
	getAccountConfigurationRate                        = 5
	getMaxBuySellAmountOpenAmountRate                  = 20
	getMaxAvailableTradableAmountRate                  = 20
	getFeeRatesRate                                    = 5
	getMaxWithdrawalsRate                              = 20
	getAvailablePairsRate                              = 6
	requestQuotesRate                                  = 3
	placeRFQOrderRate                                  = 3
	getRFQTradeOrderDetailsRate                        = 6
	getRFQTradeOrderHistoryRate                        = 6
	fiatDepositRate                                    = 6
	fiatCancelDepositRate                              = 100
	fiatDepositHistoryRate                             = 6
	fiatWithdrawalRate                                 = 6
	fiatCancelWithdrawalRate                           = 100
	fiatGetWithdrawalsRate                             = 6
	fiatGetChannelInfoRate                             = 6
	subAccountsListRate                                = 2
	getAPIKeyOfASubAccountRate                         = 1
	getSubAccountTradingBalanceRate                    = 2
	getSubAccountFundingBalanceRate                    = 2
	subAccountTransferHistoryRate                      = 6
	masterAccountsManageTransfersBetweenSubaccountRate = 1
	getTickersRate                                     = 20
	getTickerRate                                      = 20
	getOrderbookRate                                   = 20
	getCandlesticksRate                                = 40
	getCandlestickHistoryRate                          = 20
	getPublicTradesRate                                = 100
	getPublicTradeHistroryRate                         = 10
	get24HourTradingVolumeRate                         = 2
	getOracleRate                                      = 1
	getExchangeRateRate                                = 1
	getInstrumentsRate                                 = 20
	getSystemTimeRate                                  = 10
	getSystemStatusRate                                = 1
)

const (
	placeTradeOrderEPL request.EndpointLimit = iota
	placeTradeMultipleOrdersEPL
	cancelTradeOrderEPL
	cancelMultipleOrderEPL
	amendTradeOrderEPL
	amendMultipleOrdersEPL
	getOrderDetailsEPL
	getOrderListEPL
	getOrderHistoryEPL
	getOrderhistory3MonthsEPL
	getTransactionDetails3DaysEPL
	getTransactionDetails3MonthsEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	cancelAdvancedAlgoOrderEPL
	getAlgoOrderListEPL
	getAlgoOrderHistoryEPL

	getFundingCurrenciesEPL
	getFundingAccountBalanceEPL
	getAccountAssetValuationEPL
	fundingTransferEPL
	getFundsTransferStateEPL
	assetBillsDetailEPL
	lightningDepositsEPL
	getAssetDepositAddressEPL
	getDepositHistoryEPL
	postWithdrawalEPL
	postLightningWithdrawalEPL
	cancelWithdrawalEPL
	getAssetWithdrawalHistoryEPL
	getAccountBalanceEPL
	getBillsDetailLast3MonthEPL
	getBillsDetailEPL
	getAccountConfigurationEPL
	getMaxBuySellAmountOpenAmountEPL
	getMaxAvailableTradableAmountEPL
	getFeeRatesEPL
	getMaxWithdrawalsEPL

	getAvailablePairsEPL
	requestQuotesEPL
	placeRFQOrderEPL
	getRFQTradeOrderDetailsEPL
	getRFQTradeOrderHistoryEPL

	fiatDepositEPL
	fiatCancelDepositEPL
	fiatDepositHistoryEPL
	fiatWithdrawalEPL
	fiatCancelWithdrawalEPL
	fiatGetWithdrawalsEPL
	fiatGetChannelInfoEPL

	subAccountsListEPL
	getAPIKeyOfASubAccountEPL
	getSubAccountTradingBalanceEPL
	getSubAccountFundingBalanceEPL
	subAccountTransferHistoryEPL
	masterAccountsManageTransfersBetweenSubaccountEPL

	getTickersEPL
	getTickerEPL
	getOrderbookEPL
	getCandlesticksEPL
	getCandlestickHistoryEPL
	getPublicTradesEPL
	getPublicTradeHistroryEPL
	get24HourTradingVolumeEPL
	getOracleEPL
	getExchangeRateEPL
	getInstrumentsEPL
	getSystemTimeEPL
	getSystemStatusEPL
)

// GetRateLimit returns a new RateLimit instance which implements request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		placeTradeOrderEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, placeTradeOrderRate, 1),
		placeTradeMultipleOrdersEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, placeTradeMultipleOrdersRate, 1),
		cancelTradeOrderEPL:                               request.NewRateLimitWithWeight(twoSecondsInterval, cancelTradeOrderRate, 1),
		cancelMultipleOrderEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, cancelMultipleOrderRate, 1),
		amendTradeOrderEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, amendTradeOrderRate, 1),
		amendMultipleOrdersEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, amendMultipleOrdersRate, 1),
		getOrderDetailsEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, getOrderDetailsRate, 1),
		getOrderListEPL:                                   request.NewRateLimitWithWeight(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistoryEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, getOrderHistoryRate, 1),
		getOrderhistory3MonthsEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, getOrderhistory3MonthsRate, 1),
		getTransactionDetails3DaysEPL:                     request.NewRateLimitWithWeight(twoSecondsInterval, getTransactionDetails3DaysRate, 1),
		getTransactionDetails3MonthsEPL:                   request.NewRateLimitWithWeight(twoSecondsInterval, getTransactionDetails3MonthsRate, 1),
		placeAlgoOrderEPL:                                 request.NewRateLimitWithWeight(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvancedAlgoOrderEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, cancelAdvancedAlgoOrderRate, 1),
		getAlgoOrderListEPL:                               request.NewRateLimitWithWeight(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:                            request.NewRateLimitWithWeight(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getFundingCurrenciesEPL:                           request.NewRateLimitWithWeight(oneSecondInterval, getFundingCurrenciesRate, 1),
		getFundingAccountBalanceEPL:                       request.NewRateLimitWithWeight(oneSecondInterval, getFundingAccountBalanceRate, 1),
		getAccountAssetValuationEPL:                       request.NewRateLimitWithWeight(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundingTransferEPL:                                request.NewRateLimitWithWeight(oneSecondInterval, fundingTransferRate, 1),
		getFundsTransferStateEPL:                          request.NewRateLimitWithWeight(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailEPL:                               request.NewRateLimitWithWeight(oneSecondInterval, assetBillsDetailRate, 1),
		lightningDepositsEPL:                              request.NewRateLimitWithWeight(oneSecondInterval, lightningDepositsRate, 1),
		getAssetDepositAddressEPL:                         request.NewRateLimitWithWeight(oneSecondInterval, getAssetDepositAddressRate, 1),
		getDepositHistoryEPL:                              request.NewRateLimitWithWeight(oneSecondInterval, getDepositHistoryRate, 1),
		postWithdrawalEPL:                                 request.NewRateLimitWithWeight(oneSecondInterval, postWithdrawalRate, 1),
		postLightningWithdrawalEPL:                        request.NewRateLimitWithWeight(oneSecondInterval, postLightningWithdrawalRate, 1),
		cancelWithdrawalEPL:                               request.NewRateLimitWithWeight(oneSecondInterval, cancelWithdrawalRate, 1),
		getAssetWithdrawalHistoryEPL:                      request.NewRateLimitWithWeight(oneSecondInterval, getAssetWithdrawalHistoryRate, 1),
		getAccountBalanceEPL:                              request.NewRateLimitWithWeight(twoSecondsInterval, getAccountBalanceRate, 1),
		getBillsDetailLast3MonthEPL:                       request.NewRateLimitWithWeight(oneSecondInterval, getBillsDetailLast3MonthRate, 1),
		getBillsDetailEPL:                                 request.NewRateLimitWithWeight(oneSecondInterval, getBillsDetailRate, 1),
		getAccountConfigurationEPL:                        request.NewRateLimitWithWeight(twoSecondsInterval, getAccountConfigurationRate, 1),
		getMaxBuySellAmountOpenAmountEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, getMaxBuySellAmountOpenAmountRate, 1),
		getMaxAvailableTradableAmountEPL:                  request.NewRateLimitWithWeight(twoSecondsInterval, getMaxAvailableTradableAmountRate, 1),
		getFeeRatesEPL:                                    request.NewRateLimitWithWeight(twoSecondsInterval, getFeeRatesRate, 1),
		getMaxWithdrawalsEPL:                              request.NewRateLimitWithWeight(twoSecondsInterval, getMaxWithdrawalsRate, 1),
		getAvailablePairsEPL:                              request.NewRateLimitWithWeight(oneSecondInterval, getAvailablePairsRate, 1),
		requestQuotesEPL:                                  request.NewRateLimitWithWeight(oneSecondInterval, requestQuotesRate, 1),
		placeRFQOrderEPL:                                  request.NewRateLimitWithWeight(oneSecondInterval, placeRFQOrderRate, 1),
		getRFQTradeOrderDetailsEPL:                        request.NewRateLimitWithWeight(oneSecondInterval, getRFQTradeOrderDetailsRate, 1),
		getRFQTradeOrderHistoryEPL:                        request.NewRateLimitWithWeight(oneSecondInterval, getRFQTradeOrderHistoryRate, 1),
		fiatDepositEPL:                                    request.NewRateLimitWithWeight(oneSecondInterval, fiatDepositRate, 1),
		fiatCancelDepositEPL:                              request.NewRateLimitWithWeight(twoSecondsInterval, fiatCancelDepositRate, 1),
		fiatDepositHistoryEPL:                             request.NewRateLimitWithWeight(oneSecondInterval, fiatDepositHistoryRate, 1),
		fiatWithdrawalEPL:                                 request.NewRateLimitWithWeight(oneSecondInterval, fiatWithdrawalRate, 1),
		fiatCancelWithdrawalEPL:                           request.NewRateLimitWithWeight(twoSecondsInterval, fiatCancelWithdrawalRate, 1),
		fiatGetWithdrawalsEPL:                             request.NewRateLimitWithWeight(oneSecondInterval, fiatGetWithdrawalsRate, 1),
		fiatGetChannelInfoEPL:                             request.NewRateLimitWithWeight(oneSecondInterval, fiatGetChannelInfoRate, 1),
		subAccountsListEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, subAccountsListRate, 1),
		getAPIKeyOfASubAccountEPL:                         request.NewRateLimitWithWeight(oneSecondInterval, getAPIKeyOfASubAccountRate, 1),
		getSubAccountTradingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getSubAccountTradingBalanceRate, 1),
		getSubAccountFundingBalanceEPL:                    request.NewRateLimitWithWeight(twoSecondsInterval, getSubAccountFundingBalanceRate, 1),
		subAccountTransferHistoryEPL:                      request.NewRateLimitWithWeight(oneSecondInterval, subAccountTransferHistoryRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithWeight(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		getTickersEPL:                                     request.NewRateLimitWithWeight(twoSecondsInterval, getTickersRate, 1),
		getTickerEPL:                                      request.NewRateLimitWithWeight(twoSecondsInterval, getTickerRate, 1),
		getOrderbookEPL:                                   request.NewRateLimitWithWeight(twoSecondsInterval, getOrderbookRate, 1),
		getCandlesticksEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:                          request.NewRateLimitWithWeight(twoSecondsInterval, getCandlestickHistoryRate, 1),
		getPublicTradesEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, getPublicTradesRate, 1),
		getPublicTradeHistroryEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, getPublicTradeHistroryRate, 1),
		get24HourTradingVolumeEPL:                         request.NewRateLimitWithWeight(twoSecondsInterval, get24HourTradingVolumeRate, 1),
		getOracleEPL:                                      request.NewRateLimitWithWeight(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateEPL:                                request.NewRateLimitWithWeight(twoSecondsInterval, getExchangeRateRate, 1),
		getInstrumentsEPL:                                 request.NewRateLimitWithWeight(twoSecondsInterval, getInstrumentsRate, 1),
		getSystemTimeEPL:                                  request.NewRateLimitWithWeight(twoSecondsInterval, getSystemTimeRate, 1),
		getSystemStatusEPL:                                request.NewRateLimitWithWeight(fiveSecondsInterval, getSystemStatusRate, 1),
	}
}
