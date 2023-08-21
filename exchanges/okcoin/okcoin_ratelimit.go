package okcoin

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Interval instances
const (
	oneSecondInterval   = time.Second
	twoSecondsInterval  = time.Second * 2
	fiveSecondsInterval = time.Second * 5
)

// RateLimit implementa a rate Limiter
type RateLimit struct {
	PlaceTradeOrder                                *rate.Limiter
	PlaceTradeMultipleOrders                       *rate.Limiter
	CancelTradeOrder                               *rate.Limiter
	CancelMultipleOrder                            *rate.Limiter
	AmendTradeOrder                                *rate.Limiter
	AmendMultipleOrders                            *rate.Limiter
	GetOrderDetails                                *rate.Limiter
	GetOrderList                                   *rate.Limiter
	GetOrderHistory                                *rate.Limiter
	GetOrderhistory3Months                         *rate.Limiter
	GetTransactionDetails3Days                     *rate.Limiter
	GetTransactionDetails3Months                   *rate.Limiter
	PlaceAlgoOrder                                 *rate.Limiter
	CancelAlgoOrder                                *rate.Limiter
	CancelAdvancedAlgoOrder                        *rate.Limiter
	GetAlgoOrderList                               *rate.Limiter
	GetAlgoOrderHistory                            *rate.Limiter
	GetFundingCurrencies                           *rate.Limiter
	GetFundingAccountBalance                       *rate.Limiter
	GetAccountAssetValuation                       *rate.Limiter
	FundingTransfer                                *rate.Limiter
	GetFundsTransferState                          *rate.Limiter
	AssetBillsDetail                               *rate.Limiter
	LightningDeposits                              *rate.Limiter
	GetAssetDepositAddress                         *rate.Limiter
	GetDepositHistory                              *rate.Limiter
	PostWithdrawal                                 *rate.Limiter
	PostLightningWithdrawal                        *rate.Limiter
	CancelWithdrawal                               *rate.Limiter
	GetAssetWithdrawalHistory                      *rate.Limiter
	GetAccountBalance                              *rate.Limiter
	GetBillsDetailLast3Month                       *rate.Limiter
	GetBillsDetail                                 *rate.Limiter
	GetAccountConfiguration                        *rate.Limiter
	GetMaxBuySellAmountOpenAmount                  *rate.Limiter
	GetMaxAvailableTradableAmount                  *rate.Limiter
	GetFeeRates                                    *rate.Limiter
	GetMaxWithdrawals                              *rate.Limiter
	GetAvailablePairs                              *rate.Limiter
	RequestQuotes                                  *rate.Limiter
	PlaceRFQOrder                                  *rate.Limiter
	GetRFQTradeOrderDetails                        *rate.Limiter
	GetRFQTradeOrderHistory                        *rate.Limiter
	FiatDepositRate                                *rate.Limiter
	FiatCancelDepositRate                          *rate.Limiter
	FiatDepositHistoryRate                         *rate.Limiter
	FiatWithdrawalRate                             *rate.Limiter
	FiatCancelWithdrawalRate                       *rate.Limiter
	FiatGetWithdrawalsRate                         *rate.Limiter
	FiatGetChannelInfoRate                         *rate.Limiter
	SubAccountsList                                *rate.Limiter
	GetAPIKeyOfASubAccount                         *rate.Limiter
	GetSubAccountTradingBalance                    *rate.Limiter
	GetSubAccountFundingBalance                    *rate.Limiter
	SubAccountTransferHistory                      *rate.Limiter
	MasterAccountsManageTransfersBetweenSubaccount *rate.Limiter
	GetTickers                                     *rate.Limiter
	GetTicker                                      *rate.Limiter
	GetOrderbook                                   *rate.Limiter
	GetCandlesticks                                *rate.Limiter
	GetCandlestickHistory                          *rate.Limiter
	GetPublicTrades                                *rate.Limiter
	GetPublicTradeHistrory                         *rate.Limiter
	Get24HourTradingVolume                         *rate.Limiter
	GetOracle                                      *rate.Limiter
	GetExchangeRate                                *rate.Limiter
	GetInstrumentsRate                             *rate.Limiter
	GetSystemTimeRate                              *rate.Limiter
	GetSystemStatusRate                            *rate.Limiter
}

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

// Limit implements an endpoint limit.
func (r *RateLimit) Limit(ctx context.Context, ep request.EndpointLimit) error {
	switch ep {
	case placeTradeOrderEPL:
		return r.PlaceTradeOrder.Wait(ctx)
	case placeTradeMultipleOrdersEPL:
		return r.PlaceTradeMultipleOrders.Wait(ctx)
	case cancelTradeOrderEPL:
		return r.CancelTradeOrder.Wait(ctx)
	case cancelMultipleOrderEPL:
		return r.CancelMultipleOrder.Wait(ctx)
	case amendTradeOrderEPL:
		return r.AmendTradeOrder.Wait(ctx)
	case amendMultipleOrdersEPL:
		return r.AmendMultipleOrders.Wait(ctx)
	case getOrderDetailsEPL:
		return r.GetOrderDetails.Wait(ctx)
	case getOrderListEPL:
		return r.GetOrderList.Wait(ctx)
	case getOrderHistoryEPL:
		return r.GetOrderHistory.Wait(ctx)
	case getOrderhistory3MonthsEPL:
		return r.GetOrderhistory3Months.Wait(ctx)
	case getTransactionDetails3DaysEPL:
		return r.GetTransactionDetails3Days.Wait(ctx)
	case getTransactionDetails3MonthsEPL:
		return r.GetTransactionDetails3Months.Wait(ctx)
	case placeAlgoOrderEPL:
		return r.PlaceAlgoOrder.Wait(ctx)
	case cancelAlgoOrderEPL:
		return r.CancelAlgoOrder.Wait(ctx)
	case cancelAdvancedAlgoOrderEPL:
		return r.CancelAdvancedAlgoOrder.Wait(ctx)
	case getAlgoOrderListEPL:
		return r.GetAlgoOrderList.Wait(ctx)
	case getAlgoOrderHistoryEPL:
		return r.GetAlgoOrderHistory.Wait(ctx)
	case getFundingCurrenciesEPL:
		return r.GetFundingCurrencies.Wait(ctx)
	case getFundingAccountBalanceEPL:
		return r.GetFundingAccountBalance.Wait(ctx)
	case getAccountAssetValuationEPL:
		return r.GetAccountAssetValuation.Wait(ctx)
	case fundingTransferEPL:
		return r.FundingTransfer.Wait(ctx)
	case getFundsTransferStateEPL:
		return r.GetFundsTransferState.Wait(ctx)
	case assetBillsDetailEPL:
		return r.AssetBillsDetail.Wait(ctx)
	case lightningDepositsEPL:
		return r.LightningDeposits.Wait(ctx)
	case getAssetDepositAddressEPL:
		return r.GetAssetDepositAddress.Wait(ctx)
	case getDepositHistoryEPL:
		return r.GetDepositHistory.Wait(ctx)
	case postWithdrawalEPL:
		return r.PostWithdrawal.Wait(ctx)
	case postLightningWithdrawalEPL:
		return r.PostLightningWithdrawal.Wait(ctx)
	case cancelWithdrawalEPL:
		return r.CancelWithdrawal.Wait(ctx)
	case getAssetWithdrawalHistoryEPL:
		return r.GetAssetWithdrawalHistory.Wait(ctx)
	case getAccountBalanceEPL:
		return r.GetAccountBalance.Wait(ctx)
	case getBillsDetailLast3MonthEPL:
		return r.GetBillsDetailLast3Month.Wait(ctx)
	case getBillsDetailEPL:
		return r.GetBillsDetail.Wait(ctx)
	case getAccountConfigurationEPL:
		return r.GetAccountConfiguration.Wait(ctx)
	case getMaxBuySellAmountOpenAmountEPL:
		return r.GetMaxBuySellAmountOpenAmount.Wait(ctx)
	case getMaxAvailableTradableAmountEPL:
		return r.GetMaxAvailableTradableAmount.Wait(ctx)
	case getFeeRatesEPL:
		return r.GetFeeRates.Wait(ctx)
	case getMaxWithdrawalsEPL:
		return r.GetMaxWithdrawals.Wait(ctx)
	case getAvailablePairsEPL:
		return r.GetAvailablePairs.Wait(ctx)
	case requestQuotesEPL:
		return r.RequestQuotes.Wait(ctx)
	case placeRFQOrderEPL:
		return r.PlaceRFQOrder.Wait(ctx)
	case getRFQTradeOrderDetailsEPL:
		return r.GetRFQTradeOrderDetails.Wait(ctx)
	case getRFQTradeOrderHistoryEPL:
		return r.GetRFQTradeOrderHistory.Wait(ctx)
	case fiatDepositEPL:
		return r.FiatDepositRate.Wait(ctx)
	case fiatCancelDepositEPL:
		return r.FiatCancelDepositRate.Wait(ctx)
	case fiatDepositHistoryEPL:
		return r.FiatDepositHistoryRate.Wait(ctx)
	case fiatWithdrawalEPL:
		return r.FiatWithdrawalRate.Wait(ctx)
	case fiatCancelWithdrawalEPL:
		return r.FiatCancelWithdrawalRate.Wait(ctx)
	case fiatGetWithdrawalsEPL:
		return r.FiatGetWithdrawalsRate.Wait(ctx)
	case fiatGetChannelInfoEPL:
		return r.FiatGetChannelInfoRate.Wait(ctx)
	case subAccountsListEPL:
		return r.SubAccountsList.Wait(ctx)
	case getAPIKeyOfASubAccountEPL:
		return r.GetAPIKeyOfASubAccount.Wait(ctx)
	case getSubAccountTradingBalanceEPL:
		return r.GetSubAccountTradingBalance.Wait(ctx)
	case getSubAccountFundingBalanceEPL:
		return r.GetSubAccountFundingBalance.Wait(ctx)
	case subAccountTransferHistoryEPL:
		return r.SubAccountTransferHistory.Wait(ctx)
	case masterAccountsManageTransfersBetweenSubaccountEPL:
		return r.MasterAccountsManageTransfersBetweenSubaccount.Wait(ctx)
	case getTickersEPL:
		return r.GetTickers.Wait(ctx)
	case getTickerEPL:
		return r.GetTicker.Wait(ctx)
	case getOrderbookEPL:
		return r.GetOrderbook.Wait(ctx)
	case getCandlesticksEPL:
		return r.GetCandlesticks.Wait(ctx)
	case getCandlestickHistoryEPL:
		return r.GetCandlestickHistory.Wait(ctx)
	case getPublicTradesEPL:
		return r.GetPublicTrades.Wait(ctx)
	case getPublicTradeHistroryEPL:
		return r.GetPublicTradeHistrory.Wait(ctx)
	case get24HourTradingVolumeEPL:
		return r.Get24HourTradingVolume.Wait(ctx)
	case getOracleEPL:
		return r.GetOracle.Wait(ctx)
	case getExchangeRateEPL:
		return r.GetExchangeRate.Wait(ctx)
	case getInstrumentsEPL:
		return r.GetInstrumentsRate.Wait(ctx)
	case getSystemTimeEPL:
		return r.GetSystemTimeRate.Wait(ctx)
	case getSystemStatusEPL:
		return r.GetSystemStatusRate.Wait(ctx)
	default:
		return errors.New("unknown endpoint limit")
	}
}

// SetRateLimit returns a new RateLimit instance which implements request.Limiter interface.
func SetRateLimit() *RateLimit {
	return &RateLimit{
		PlaceTradeOrder:                                request.NewRateLimit(twoSecondsInterval, placeTradeOrderRate),
		PlaceTradeMultipleOrders:                       request.NewRateLimit(twoSecondsInterval, placeTradeMultipleOrdersRate),
		CancelTradeOrder:                               request.NewRateLimit(twoSecondsInterval, cancelTradeOrderRate),
		CancelMultipleOrder:                            request.NewRateLimit(twoSecondsInterval, cancelMultipleOrderRate),
		AmendTradeOrder:                                request.NewRateLimit(twoSecondsInterval, amendTradeOrderRate),
		AmendMultipleOrders:                            request.NewRateLimit(twoSecondsInterval, amendMultipleOrdersRate),
		GetOrderDetails:                                request.NewRateLimit(twoSecondsInterval, getOrderDetailsRate),
		GetOrderList:                                   request.NewRateLimit(twoSecondsInterval, getOrderListRate),
		GetOrderHistory:                                request.NewRateLimit(twoSecondsInterval, getOrderHistoryRate),
		GetOrderhistory3Months:                         request.NewRateLimit(twoSecondsInterval, getOrderhistory3MonthsRate),
		GetTransactionDetails3Days:                     request.NewRateLimit(twoSecondsInterval, getTransactionDetails3DaysRate),
		GetTransactionDetails3Months:                   request.NewRateLimit(twoSecondsInterval, getTransactionDetails3MonthsRate),
		PlaceAlgoOrder:                                 request.NewRateLimit(twoSecondsInterval, placeAlgoOrderRate),
		CancelAlgoOrder:                                request.NewRateLimit(twoSecondsInterval, cancelAlgoOrderRate),
		CancelAdvancedAlgoOrder:                        request.NewRateLimit(twoSecondsInterval, cancelAdvancedAlgoOrderRate),
		GetAlgoOrderList:                               request.NewRateLimit(twoSecondsInterval, getAlgoOrderListRate),
		GetAlgoOrderHistory:                            request.NewRateLimit(twoSecondsInterval, getAlgoOrderHistoryRate),
		GetFundingCurrencies:                           request.NewRateLimit(oneSecondInterval, getFundingCurrenciesRate),
		GetFundingAccountBalance:                       request.NewRateLimit(oneSecondInterval, getFundingAccountBalanceRate),
		GetAccountAssetValuation:                       request.NewRateLimit(twoSecondsInterval, getAccountAssetValuationRate),
		FundingTransfer:                                request.NewRateLimit(oneSecondInterval, fundingTransferRate),
		GetFundsTransferState:                          request.NewRateLimit(oneSecondInterval, getFundsTransferStateRate),
		AssetBillsDetail:                               request.NewRateLimit(oneSecondInterval, assetBillsDetailRate),
		LightningDeposits:                              request.NewRateLimit(oneSecondInterval, lightningDepositsRate),
		GetAssetDepositAddress:                         request.NewRateLimit(oneSecondInterval, getAssetDepositAddressRate),
		GetDepositHistory:                              request.NewRateLimit(oneSecondInterval, getDepositHistoryRate),
		PostWithdrawal:                                 request.NewRateLimit(oneSecondInterval, postWithdrawalRate),
		PostLightningWithdrawal:                        request.NewRateLimit(oneSecondInterval, postLightningWithdrawalRate),
		CancelWithdrawal:                               request.NewRateLimit(oneSecondInterval, cancelWithdrawalRate),
		GetAssetWithdrawalHistory:                      request.NewRateLimit(oneSecondInterval, getAssetWithdrawalHistoryRate),
		GetAccountBalance:                              request.NewRateLimit(twoSecondsInterval, getAccountBalanceRate),
		GetBillsDetailLast3Month:                       request.NewRateLimit(oneSecondInterval, getBillsDetailLast3MonthRate),
		GetBillsDetail:                                 request.NewRateLimit(oneSecondInterval, getBillsDetailRate),
		GetAccountConfiguration:                        request.NewRateLimit(twoSecondsInterval, getAccountConfigurationRate),
		GetMaxBuySellAmountOpenAmount:                  request.NewRateLimit(twoSecondsInterval, getMaxBuySellAmountOpenAmountRate),
		GetMaxAvailableTradableAmount:                  request.NewRateLimit(twoSecondsInterval, getMaxAvailableTradableAmountRate),
		GetFeeRates:                                    request.NewRateLimit(twoSecondsInterval, getFeeRatesRate),
		GetMaxWithdrawals:                              request.NewRateLimit(twoSecondsInterval, getMaxWithdrawalsRate),
		GetAvailablePairs:                              request.NewRateLimit(oneSecondInterval, getAvailablePairsRate),
		RequestQuotes:                                  request.NewRateLimit(oneSecondInterval, requestQuotesRate),
		PlaceRFQOrder:                                  request.NewRateLimit(oneSecondInterval, placeRFQOrderRate),
		GetRFQTradeOrderDetails:                        request.NewRateLimit(oneSecondInterval, getRFQTradeOrderDetailsRate),
		GetRFQTradeOrderHistory:                        request.NewRateLimit(oneSecondInterval, getRFQTradeOrderHistoryRate),
		FiatDepositRate:                                request.NewRateLimit(oneSecondInterval, fiatDepositRate),
		FiatCancelDepositRate:                          request.NewRateLimit(twoSecondsInterval, fiatCancelDepositRate),
		FiatDepositHistoryRate:                         request.NewRateLimit(oneSecondInterval, fiatDepositHistoryRate),
		FiatWithdrawalRate:                             request.NewRateLimit(oneSecondInterval, fiatWithdrawalRate),
		FiatCancelWithdrawalRate:                       request.NewRateLimit(twoSecondsInterval, fiatCancelWithdrawalRate),
		FiatGetWithdrawalsRate:                         request.NewRateLimit(oneSecondInterval, fiatGetWithdrawalsRate),
		FiatGetChannelInfoRate:                         request.NewRateLimit(oneSecondInterval, fiatGetChannelInfoRate),
		SubAccountsList:                                request.NewRateLimit(twoSecondsInterval, subAccountsListRate),
		GetAPIKeyOfASubAccount:                         request.NewRateLimit(oneSecondInterval, getAPIKeyOfASubAccountRate),
		GetSubAccountTradingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubAccountTradingBalanceRate),
		GetSubAccountFundingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubAccountFundingBalanceRate),
		SubAccountTransferHistory:                      request.NewRateLimit(oneSecondInterval, subAccountTransferHistoryRate),
		MasterAccountsManageTransfersBetweenSubaccount: request.NewRateLimit(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate),
		GetTickers:                                     request.NewRateLimit(twoSecondsInterval, getTickersRate),
		GetTicker:                                      request.NewRateLimit(twoSecondsInterval, getTickerRate),
		GetOrderbook:                                   request.NewRateLimit(twoSecondsInterval, getOrderbookRate),
		GetCandlesticks:                                request.NewRateLimit(twoSecondsInterval, getCandlesticksRate),
		GetCandlestickHistory:                          request.NewRateLimit(twoSecondsInterval, getCandlestickHistoryRate),
		GetPublicTrades:                                request.NewRateLimit(twoSecondsInterval, getPublicTradesRate),
		GetPublicTradeHistrory:                         request.NewRateLimit(twoSecondsInterval, getPublicTradeHistroryRate),
		Get24HourTradingVolume:                         request.NewRateLimit(twoSecondsInterval, get24HourTradingVolumeRate),
		GetOracle:                                      request.NewRateLimit(fiveSecondsInterval, getOracleRate),
		GetExchangeRate:                                request.NewRateLimit(twoSecondsInterval, getExchangeRateRate),
		GetInstrumentsRate:                             request.NewRateLimit(twoSecondsInterval, getInstrumentsRate),
		GetSystemTimeRate:                              request.NewRateLimit(twoSecondsInterval, getSystemTimeRate),
		GetSystemStatusRate:                            request.NewRateLimit(fiveSecondsInterval, getSystemStatusRate),
	}
}
