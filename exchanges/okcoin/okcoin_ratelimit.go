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

// SetRateLimit returns a new RateLimit instance which implements request.Limiter interface.
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		placeTradeOrderEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, placeTradeOrderRate, 1),
		placeTradeMultipleOrdersEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, placeTradeMultipleOrdersRate, 1),
		cancelTradeOrderEPL:                               request.NewRateLimitWithToken(twoSecondsInterval, cancelTradeOrderRate, 1),
		cancelMultipleOrderEPL:                            request.NewRateLimitWithToken(twoSecondsInterval, cancelMultipleOrderRate, 1),
		amendTradeOrderEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, amendTradeOrderRate, 1),
		amendMultipleOrdersEPL:                            request.NewRateLimitWithToken(twoSecondsInterval, amendMultipleOrdersRate, 1),
		getOrderDetailsEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, getOrderDetailsRate, 1),
		getOrderListEPL:                                   request.NewRateLimitWithToken(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistoryEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, getOrderHistoryRate, 1),
		getOrderhistory3MonthsEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, getOrderhistory3MonthsRate, 1),
		getTransactionDetails3DaysEPL:                     request.NewRateLimitWithToken(twoSecondsInterval, getTransactionDetails3DaysRate, 1),
		getTransactionDetails3MonthsEPL:                   request.NewRateLimitWithToken(twoSecondsInterval, getTransactionDetails3MonthsRate, 1),
		placeAlgoOrderEPL:                                 request.NewRateLimitWithToken(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvancedAlgoOrderEPL:                        request.NewRateLimitWithToken(twoSecondsInterval, cancelAdvancedAlgoOrderRate, 1),
		getAlgoOrderListEPL:                               request.NewRateLimitWithToken(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:                            request.NewRateLimitWithToken(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getFundingCurrenciesEPL:                           request.NewRateLimitWithToken(oneSecondInterval, getFundingCurrenciesRate, 1),
		getFundingAccountBalanceEPL:                       request.NewRateLimitWithToken(oneSecondInterval, getFundingAccountBalanceRate, 1),
		getAccountAssetValuationEPL:                       request.NewRateLimitWithToken(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundingTransferEPL:                                request.NewRateLimitWithToken(oneSecondInterval, fundingTransferRate, 1),
		getFundsTransferStateEPL:                          request.NewRateLimitWithToken(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailEPL:                               request.NewRateLimitWithToken(oneSecondInterval, assetBillsDetailRate, 1),
		lightningDepositsEPL:                              request.NewRateLimitWithToken(oneSecondInterval, lightningDepositsRate, 1),
		getAssetDepositAddressEPL:                         request.NewRateLimitWithToken(oneSecondInterval, getAssetDepositAddressRate, 1),
		getDepositHistoryEPL:                              request.NewRateLimitWithToken(oneSecondInterval, getDepositHistoryRate, 1),
		postWithdrawalEPL:                                 request.NewRateLimitWithToken(oneSecondInterval, postWithdrawalRate, 1),
		postLightningWithdrawalEPL:                        request.NewRateLimitWithToken(oneSecondInterval, postLightningWithdrawalRate, 1),
		cancelWithdrawalEPL:                               request.NewRateLimitWithToken(oneSecondInterval, cancelWithdrawalRate, 1),
		getAssetWithdrawalHistoryEPL:                      request.NewRateLimitWithToken(oneSecondInterval, getAssetWithdrawalHistoryRate, 1),
		getAccountBalanceEPL:                              request.NewRateLimitWithToken(twoSecondsInterval, getAccountBalanceRate, 1),
		getBillsDetailLast3MonthEPL:                       request.NewRateLimitWithToken(oneSecondInterval, getBillsDetailLast3MonthRate, 1),
		getBillsDetailEPL:                                 request.NewRateLimitWithToken(oneSecondInterval, getBillsDetailRate, 1),
		getAccountConfigurationEPL:                        request.NewRateLimitWithToken(twoSecondsInterval, getAccountConfigurationRate, 1),
		getMaxBuySellAmountOpenAmountEPL:                  request.NewRateLimitWithToken(twoSecondsInterval, getMaxBuySellAmountOpenAmountRate, 1),
		getMaxAvailableTradableAmountEPL:                  request.NewRateLimitWithToken(twoSecondsInterval, getMaxAvailableTradableAmountRate, 1),
		getFeeRatesEPL:                                    request.NewRateLimitWithToken(twoSecondsInterval, getFeeRatesRate, 1),
		getMaxWithdrawalsEPL:                              request.NewRateLimitWithToken(twoSecondsInterval, getMaxWithdrawalsRate, 1),
		getAvailablePairsEPL:                              request.NewRateLimitWithToken(oneSecondInterval, getAvailablePairsRate, 1),
		requestQuotesEPL:                                  request.NewRateLimitWithToken(oneSecondInterval, requestQuotesRate, 1),
		placeRFQOrderEPL:                                  request.NewRateLimitWithToken(oneSecondInterval, placeRFQOrderRate, 1),
		getRFQTradeOrderDetailsEPL:                        request.NewRateLimitWithToken(oneSecondInterval, getRFQTradeOrderDetailsRate, 1),
		getRFQTradeOrderHistoryEPL:                        request.NewRateLimitWithToken(oneSecondInterval, getRFQTradeOrderHistoryRate, 1),
		fiatDepositEPL:                                    request.NewRateLimitWithToken(oneSecondInterval, fiatDepositRate, 1),
		fiatCancelDepositEPL:                              request.NewRateLimitWithToken(twoSecondsInterval, fiatCancelDepositRate, 1),
		fiatDepositHistoryEPL:                             request.NewRateLimitWithToken(oneSecondInterval, fiatDepositHistoryRate, 1),
		fiatWithdrawalEPL:                                 request.NewRateLimitWithToken(oneSecondInterval, fiatWithdrawalRate, 1),
		fiatCancelWithdrawalEPL:                           request.NewRateLimitWithToken(twoSecondsInterval, fiatCancelWithdrawalRate, 1),
		fiatGetWithdrawalsEPL:                             request.NewRateLimitWithToken(oneSecondInterval, fiatGetWithdrawalsRate, 1),
		fiatGetChannelInfoEPL:                             request.NewRateLimitWithToken(oneSecondInterval, fiatGetChannelInfoRate, 1),
		subAccountsListEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, subAccountsListRate, 1),
		getAPIKeyOfASubAccountEPL:                         request.NewRateLimitWithToken(oneSecondInterval, getAPIKeyOfASubAccountRate, 1),
		getSubAccountTradingBalanceEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getSubAccountTradingBalanceRate, 1),
		getSubAccountFundingBalanceEPL:                    request.NewRateLimitWithToken(twoSecondsInterval, getSubAccountFundingBalanceRate, 1),
		subAccountTransferHistoryEPL:                      request.NewRateLimitWithToken(oneSecondInterval, subAccountTransferHistoryRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimitWithToken(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		getTickersEPL:                                     request.NewRateLimitWithToken(twoSecondsInterval, getTickersRate, 1),
		getTickerEPL:                                      request.NewRateLimitWithToken(twoSecondsInterval, getTickerRate, 1),
		getOrderbookEPL:                                   request.NewRateLimitWithToken(twoSecondsInterval, getOrderbookRate, 1),
		getCandlesticksEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:                          request.NewRateLimitWithToken(twoSecondsInterval, getCandlestickHistoryRate, 1),
		getPublicTradesEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, getPublicTradesRate, 1),
		getPublicTradeHistroryEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, getPublicTradeHistroryRate, 1),
		get24HourTradingVolumeEPL:                         request.NewRateLimitWithToken(twoSecondsInterval, get24HourTradingVolumeRate, 1),
		getOracleEPL:                                      request.NewRateLimitWithToken(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateEPL:                                request.NewRateLimitWithToken(twoSecondsInterval, getExchangeRateRate, 1),
		getInstrumentsEPL:                                 request.NewRateLimitWithToken(twoSecondsInterval, getInstrumentsRate, 1),
		getSystemTimeEPL:                                  request.NewRateLimitWithToken(twoSecondsInterval, getSystemTimeRate, 1),
		getSystemStatusEPL:                                request.NewRateLimitWithToken(fiveSecondsInterval, getSystemStatusRate, 1),
	}
}
