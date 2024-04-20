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
		PlaceTradeOrder:                                request.NewRateLimit(twoSecondsInterval, 60),
		PlaceTradeMultipleOrders:                       request.NewRateLimit(twoSecondsInterval, 300),
		CancelTradeOrder:                               request.NewRateLimit(twoSecondsInterval, 60),
		CancelMultipleOrder:                            request.NewRateLimit(twoSecondsInterval, 300),
		AmendTradeOrder:                                request.NewRateLimit(twoSecondsInterval, 60),
		AmendMultipleOrders:                            request.NewRateLimit(twoSecondsInterval, 300),
		GetOrderDetails:                                request.NewRateLimit(twoSecondsInterval, 60),
		GetOrderList:                                   request.NewRateLimit(twoSecondsInterval, 60),
		GetOrderHistory:                                request.NewRateLimit(twoSecondsInterval, 40),
		GetOrderhistory3Months:                         request.NewRateLimit(twoSecondsInterval, 20),
		GetTransactionDetails3Days:                     request.NewRateLimit(twoSecondsInterval, 60),
		GetTransactionDetails3Months:                   request.NewRateLimit(twoSecondsInterval, 10),
		PlaceAlgoOrder:                                 request.NewRateLimit(twoSecondsInterval, 20),
		CancelAlgoOrder:                                request.NewRateLimit(twoSecondsInterval, 20),
		CancelAdvancedAlgoOrder:                        request.NewRateLimit(twoSecondsInterval, 20),
		GetAlgoOrderList:                               request.NewRateLimit(twoSecondsInterval, 20),
		GetAlgoOrderHistory:                            request.NewRateLimit(twoSecondsInterval, 20),
		GetFundingCurrencies:                           request.NewRateLimit(time.Second, 6),
		GetFundingAccountBalance:                       request.NewRateLimit(time.Second, 6),
		GetAccountAssetValuation:                       request.NewRateLimit(twoSecondsInterval, 1),
		FundingTransfer:                                request.NewRateLimit(time.Second, 1),
		GetFundsTransferState:                          request.NewRateLimit(time.Second, 1),
		AssetBillsDetail:                               request.NewRateLimit(time.Second, 6),
		LightningDeposits:                              request.NewRateLimit(time.Second, 2),
		GetAssetDepositAddress:                         request.NewRateLimit(time.Second, 6),
		GetDepositHistory:                              request.NewRateLimit(time.Second, 6),
		PostWithdrawal:                                 request.NewRateLimit(time.Second, 6),
		PostLightningWithdrawal:                        request.NewRateLimit(time.Second, 2),
		CancelWithdrawal:                               request.NewRateLimit(time.Second, 6),
		GetAssetWithdrawalHistory:                      request.NewRateLimit(time.Second, 6),
		GetAccountBalance:                              request.NewRateLimit(twoSecondsInterval, 10),
		GetBillsDetailLast3Month:                       request.NewRateLimit(time.Second, 6),
		GetBillsDetail:                                 request.NewRateLimit(time.Second, 6),
		GetAccountConfiguration:                        request.NewRateLimit(twoSecondsInterval, 5),
		GetMaxBuySellAmountOpenAmount:                  request.NewRateLimit(twoSecondsInterval, 20),
		GetMaxAvailableTradableAmount:                  request.NewRateLimit(twoSecondsInterval, 20),
		GetFeeRates:                                    request.NewRateLimit(twoSecondsInterval, 5),
		GetMaxWithdrawals:                              request.NewRateLimit(twoSecondsInterval, 20),
		GetAvailablePairs:                              request.NewRateLimit(time.Second, 6),
		RequestQuotes:                                  request.NewRateLimit(time.Second, 3),
		PlaceRFQOrder:                                  request.NewRateLimit(time.Second, 3),
		GetRFQTradeOrderDetails:                        request.NewRateLimit(time.Second, 6),
		GetRFQTradeOrderHistory:                        request.NewRateLimit(time.Second, 6),
		FiatDepositRate:                                request.NewRateLimit(time.Second, 6),
		FiatCancelDepositRate:                          request.NewRateLimit(twoSecondsInterval, 100),
		FiatDepositHistoryRate:                         request.NewRateLimit(time.Second, 6),
		FiatWithdrawalRate:                             request.NewRateLimit(time.Second, 6),
		FiatCancelWithdrawalRate:                       request.NewRateLimit(twoSecondsInterval, 100),
		FiatGetWithdrawalsRate:                         request.NewRateLimit(time.Second, 6),
		FiatGetChannelInfoRate:                         request.NewRateLimit(time.Second, 6),
		SubAccountsList:                                request.NewRateLimit(twoSecondsInterval, 2),
		GetAPIKeyOfASubAccount:                         request.NewRateLimit(time.Second, 1),
		GetSubAccountTradingBalance:                    request.NewRateLimit(twoSecondsInterval, 2),
		GetSubAccountFundingBalance:                    request.NewRateLimit(twoSecondsInterval, 2),
		SubAccountTransferHistory:                      request.NewRateLimit(time.Second, 6),
		MasterAccountsManageTransfersBetweenSubaccount: request.NewRateLimit(time.Second, 1),
		GetTickers:                                     request.NewRateLimit(twoSecondsInterval, 20),
		GetTicker:                                      request.NewRateLimit(twoSecondsInterval, 20),
		GetOrderbook:                                   request.NewRateLimit(twoSecondsInterval, 20),
		GetCandlesticks:                                request.NewRateLimit(twoSecondsInterval, 40),
		GetCandlestickHistory:                          request.NewRateLimit(twoSecondsInterval, 20),
		GetPublicTrades:                                request.NewRateLimit(twoSecondsInterval, 100),
		GetPublicTradeHistrory:                         request.NewRateLimit(twoSecondsInterval, 10),
		Get24HourTradingVolume:                         request.NewRateLimit(twoSecondsInterval, 2),
		GetOracle:                                      request.NewRateLimit(fiveSecondsInterval, 1),
		GetExchangeRate:                                request.NewRateLimit(twoSecondsInterval, 1),
		GetInstrumentsRate:                             request.NewRateLimit(twoSecondsInterval, 20),
		GetSystemTimeRate:                              request.NewRateLimit(twoSecondsInterval, 10),
		GetSystemStatusRate:                            request.NewRateLimit(fiveSecondsInterval, 1),
	}
}
