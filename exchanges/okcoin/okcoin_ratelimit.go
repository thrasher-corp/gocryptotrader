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
		placeTradeOrderEPL:                                request.NewRateLimit(twoSecondsInterval, placeTradeOrderRate, 1),
		placeTradeMultipleOrdersEPL:                       request.NewRateLimit(twoSecondsInterval, placeTradeMultipleOrdersRate, 1),
		cancelTradeOrderEPL:                               request.NewRateLimit(twoSecondsInterval, cancelTradeOrderRate, 1),
		cancelMultipleOrderEPL:                            request.NewRateLimit(twoSecondsInterval, cancelMultipleOrderRate, 1),
		amendTradeOrderEPL:                                request.NewRateLimit(twoSecondsInterval, amendTradeOrderRate, 1),
		amendMultipleOrdersEPL:                            request.NewRateLimit(twoSecondsInterval, amendMultipleOrdersRate, 1),
		getOrderDetailsEPL:                                request.NewRateLimit(twoSecondsInterval, getOrderDetailsRate, 1),
		getOrderListEPL:                                   request.NewRateLimit(twoSecondsInterval, getOrderListRate, 1),
		getOrderHistoryEPL:                                request.NewRateLimit(twoSecondsInterval, getOrderHistoryRate, 1),
		getOrderhistory3MonthsEPL:                         request.NewRateLimit(twoSecondsInterval, getOrderhistory3MonthsRate, 1),
		getTransactionDetails3DaysEPL:                     request.NewRateLimit(twoSecondsInterval, getTransactionDetails3DaysRate, 1),
		getTransactionDetails3MonthsEPL:                   request.NewRateLimit(twoSecondsInterval, getTransactionDetails3MonthsRate, 1),
		placeAlgoOrderEPL:                                 request.NewRateLimit(twoSecondsInterval, placeAlgoOrderRate, 1),
		cancelAlgoOrderEPL:                                request.NewRateLimit(twoSecondsInterval, cancelAlgoOrderRate, 1),
		cancelAdvancedAlgoOrderEPL:                        request.NewRateLimit(twoSecondsInterval, cancelAdvancedAlgoOrderRate, 1),
		getAlgoOrderListEPL:                               request.NewRateLimit(twoSecondsInterval, getAlgoOrderListRate, 1),
		getAlgoOrderHistoryEPL:                            request.NewRateLimit(twoSecondsInterval, getAlgoOrderHistoryRate, 1),
		getFundingCurrenciesEPL:                           request.NewRateLimit(oneSecondInterval, getFundingCurrenciesRate, 1),
		getFundingAccountBalanceEPL:                       request.NewRateLimit(oneSecondInterval, getFundingAccountBalanceRate, 1),
		getAccountAssetValuationEPL:                       request.NewRateLimit(twoSecondsInterval, getAccountAssetValuationRate, 1),
		fundingTransferEPL:                                request.NewRateLimit(oneSecondInterval, fundingTransferRate, 1),
		getFundsTransferStateEPL:                          request.NewRateLimit(oneSecondInterval, getFundsTransferStateRate, 1),
		assetBillsDetailEPL:                               request.NewRateLimit(oneSecondInterval, assetBillsDetailRate, 1),
		lightningDepositsEPL:                              request.NewRateLimit(oneSecondInterval, lightningDepositsRate, 1),
		getAssetDepositAddressEPL:                         request.NewRateLimit(oneSecondInterval, getAssetDepositAddressRate, 1),
		getDepositHistoryEPL:                              request.NewRateLimit(oneSecondInterval, getDepositHistoryRate, 1),
		postWithdrawalEPL:                                 request.NewRateLimit(oneSecondInterval, postWithdrawalRate, 1),
		postLightningWithdrawalEPL:                        request.NewRateLimit(oneSecondInterval, postLightningWithdrawalRate, 1),
		cancelWithdrawalEPL:                               request.NewRateLimit(oneSecondInterval, cancelWithdrawalRate, 1),
		getAssetWithdrawalHistoryEPL:                      request.NewRateLimit(oneSecondInterval, getAssetWithdrawalHistoryRate, 1),
		getAccountBalanceEPL:                              request.NewRateLimit(twoSecondsInterval, getAccountBalanceRate, 1),
		getBillsDetailLast3MonthEPL:                       request.NewRateLimit(oneSecondInterval, getBillsDetailLast3MonthRate, 1),
		getBillsDetailEPL:                                 request.NewRateLimit(oneSecondInterval, getBillsDetailRate, 1),
		getAccountConfigurationEPL:                        request.NewRateLimit(twoSecondsInterval, getAccountConfigurationRate, 1),
		getMaxBuySellAmountOpenAmountEPL:                  request.NewRateLimit(twoSecondsInterval, getMaxBuySellAmountOpenAmountRate, 1),
		getMaxAvailableTradableAmountEPL:                  request.NewRateLimit(twoSecondsInterval, getMaxAvailableTradableAmountRate, 1),
		getFeeRatesEPL:                                    request.NewRateLimit(twoSecondsInterval, getFeeRatesRate, 1),
		getMaxWithdrawalsEPL:                              request.NewRateLimit(twoSecondsInterval, getMaxWithdrawalsRate, 1),
		getAvailablePairsEPL:                              request.NewRateLimit(oneSecondInterval, getAvailablePairsRate, 1),
		requestQuotesEPL:                                  request.NewRateLimit(oneSecondInterval, requestQuotesRate, 1),
		placeRFQOrderEPL:                                  request.NewRateLimit(oneSecondInterval, placeRFQOrderRate, 1),
		getRFQTradeOrderDetailsEPL:                        request.NewRateLimit(oneSecondInterval, getRFQTradeOrderDetailsRate, 1),
		getRFQTradeOrderHistoryEPL:                        request.NewRateLimit(oneSecondInterval, getRFQTradeOrderHistoryRate, 1),
		fiatDepositEPL:                                    request.NewRateLimit(oneSecondInterval, fiatDepositRate, 1),
		fiatCancelDepositEPL:                              request.NewRateLimit(twoSecondsInterval, fiatCancelDepositRate, 1),
		fiatDepositHistoryEPL:                             request.NewRateLimit(oneSecondInterval, fiatDepositHistoryRate, 1),
		fiatWithdrawalEPL:                                 request.NewRateLimit(oneSecondInterval, fiatWithdrawalRate, 1),
		fiatCancelWithdrawalEPL:                           request.NewRateLimit(twoSecondsInterval, fiatCancelWithdrawalRate, 1),
		fiatGetWithdrawalsEPL:                             request.NewRateLimit(oneSecondInterval, fiatGetWithdrawalsRate, 1),
		fiatGetChannelInfoEPL:                             request.NewRateLimit(oneSecondInterval, fiatGetChannelInfoRate, 1),
		subAccountsListEPL:                                request.NewRateLimit(twoSecondsInterval, subAccountsListRate, 1),
		getAPIKeyOfASubAccountEPL:                         request.NewRateLimit(oneSecondInterval, getAPIKeyOfASubAccountRate, 1),
		getSubAccountTradingBalanceEPL:                    request.NewRateLimit(twoSecondsInterval, getSubAccountTradingBalanceRate, 1),
		getSubAccountFundingBalanceEPL:                    request.NewRateLimit(twoSecondsInterval, getSubAccountFundingBalanceRate, 1),
		subAccountTransferHistoryEPL:                      request.NewRateLimit(oneSecondInterval, subAccountTransferHistoryRate, 1),
		masterAccountsManageTransfersBetweenSubaccountEPL: request.NewRateLimit(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate, 1),
		getTickersEPL:                                     request.NewRateLimit(twoSecondsInterval, getTickersRate, 1),
		getTickerEPL:                                      request.NewRateLimit(twoSecondsInterval, getTickerRate, 1),
		getOrderbookEPL:                                   request.NewRateLimit(twoSecondsInterval, getOrderbookRate, 1),
		getCandlesticksEPL:                                request.NewRateLimit(twoSecondsInterval, getCandlesticksRate, 1),
		getCandlestickHistoryEPL:                          request.NewRateLimit(twoSecondsInterval, getCandlestickHistoryRate, 1),
		getPublicTradesEPL:                                request.NewRateLimit(twoSecondsInterval, getPublicTradesRate, 1),
		getPublicTradeHistroryEPL:                         request.NewRateLimit(twoSecondsInterval, getPublicTradeHistroryRate, 1),
		get24HourTradingVolumeEPL:                         request.NewRateLimit(twoSecondsInterval, get24HourTradingVolumeRate, 1),
		getOracleEPL:                                      request.NewRateLimit(fiveSecondsInterval, getOracleRate, 1),
		getExchangeRateEPL:                                request.NewRateLimit(twoSecondsInterval, getExchangeRateRate, 1),
		getInstrumentsEPL:                                 request.NewRateLimit(twoSecondsInterval, getInstrumentsRate, 1),
		getSystemTimeEPL:                                  request.NewRateLimit(twoSecondsInterval, getSystemTimeRate, 1),
		getSystemStatusEPL:                                request.NewRateLimit(fiveSecondsInterval, getSystemStatusRate, 1),
	}
}
