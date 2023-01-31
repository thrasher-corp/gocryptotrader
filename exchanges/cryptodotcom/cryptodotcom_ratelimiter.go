package cryptodotcom

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

const (
	hundredMilliSecondsInterval = 100 * time.Millisecond
	oneSecondInterval           = time.Second

	// number of requests per interval
	hundredPerInterval = 100
	fifteenPerIntrval  = 15
	thirtyPerInterval  = 30
	onePerInterval     = 1
	threePerInterval   = 3
)

const (
	publicAuthRate request.EndpointLimit = iota
	publicInstrumentsRate
	publicOrderbookRate
	publicCandlestickRate
	publicTickerRate
	publicTradesRate
	publicGetValuationsRate
	publicGetExpiredSettlementPriceRate
	publicGetInsuranceRate
	privateSetCancelOnDisconnectRate
	privateGetCancelOnDisconnectRate
	privateUserBalanceRate
	privateUserBalanceHistoryRate
	privateCreateSubAccountTransferRate
	privateGetSubAccountBalancesRate
	privateGetPositionsRate
	privateCreateOrderRate
	privateCancelOrderRate
	privateCreateOrderListRate
	privateCancelOrderListRate
	privateGetOrderListRate
	privateCancelAllOrdersRate
	privateClosePositionRate
	privateGetOrderHistoryRate
	privateGetOpenOrdersRate
	privateGetOrderDetailRate
	privateGetTradesRate
	privateChangeAccountLeverageRate
	privateGetTransactionsRate
	postWithdrawalRate
	privateGetCurrencyNetworksRate
	privategetDepositAddressRate
	privateGetAccountsRate
	privateGetOTCUserRate
	privateGetOTCInstrumentsRate
	privateOTCRequestQuoteRate
	privateOTCAcceptQuoteRate
	privateGetOTCQuoteHistoryRate
	privateGetOTCTradeHistoryRate
	privateGetWithdrawalHistoryRate
	privateGetDepositHistoryRate
	privateGetAccountSummaryRate
)

// RateLimiter represents the rate limiter struct for Crypto.com endpoints
type RateLimiter struct {
	PublicAuth                      *rate.Limiter
	PublicInstruments               *rate.Limiter
	PublicOrderbook                 *rate.Limiter
	PublicCandlestick               *rate.Limiter
	PublicTicker                    *rate.Limiter
	PublicTrades                    *rate.Limiter
	PublicGetValuations             *rate.Limiter
	PublicGetExpiredSettlementPrice *rate.Limiter
	PublicGetInsurance              *rate.Limiter
	PrivateSetCancelOnDisconnect    *rate.Limiter
	PrivateGetCancelOnDisconnect    *rate.Limiter
	PrivateUserBalance              *rate.Limiter
	PrivateUserBalanceHistory       *rate.Limiter
	PrivateCreateSubAccountTransfer *rate.Limiter
	PrivateGetSubAccountBalances    *rate.Limiter
	PrivateGetPositions             *rate.Limiter
	PrivateCreateOrder              *rate.Limiter
	PrivateCancelOrder              *rate.Limiter
	PrivateCreateOrderList          *rate.Limiter
	PrivateCancelOrderList          *rate.Limiter
	PrivateGetOrderList             *rate.Limiter
	PrivateCancelAllOrders          *rate.Limiter
	PrivateClosePosition            *rate.Limiter
	PrivateGetOrderHistory          *rate.Limiter
	PrivateGetOpenOrders            *rate.Limiter
	PrivateGetOrderDetail           *rate.Limiter
	PrivateGetTrades                *rate.Limiter
	PrivateChangeAccountLeverage    *rate.Limiter
	PrivateGetTransactions          *rate.Limiter
	PostWithdrawal                  *rate.Limiter
	PrivateGetCurrencyNetworks      *rate.Limiter
	PrivategetDepositAddress        *rate.Limiter
	PrivateGetAccounts              *rate.Limiter
	PrivateGetOTCUser               *rate.Limiter
	PrivateGetOTCInstruments        *rate.Limiter
	PrivateOTCRequestQuote          *rate.Limiter
	PrivateOTCAcceptQuote           *rate.Limiter
	PrivateGetOTCQuoteHistory       *rate.Limiter
	PrivateGetOTCTradeHistory       *rate.Limiter
	PrivateGetWithdrawalHistory     *rate.Limiter
	PrivateGetDepositHistory        *rate.Limiter
	PrivateGetAccountSummary        *rate.Limiter
}

// Limit limits the endpoint functionality
func (r *RateLimiter) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case publicAuthRate:
		return r.PublicAuth.Wait(ctx)
	case publicInstrumentsRate:
		return r.PublicInstruments.Wait(ctx)
	case publicOrderbookRate:
		return r.PublicOrderbook.Wait(ctx)
	case publicCandlestickRate:
		return r.PublicCandlestick.Wait(ctx)
	case publicTickerRate:
		return r.PublicTicker.Wait(ctx)
	case publicTradesRate:
		return r.PublicTrades.Wait(ctx)
	case publicGetValuationsRate:
		return r.PublicGetValuations.Wait(ctx)
	case publicGetExpiredSettlementPriceRate:
		return r.PublicGetExpiredSettlementPrice.Wait(ctx)
	case publicGetInsuranceRate:
		return r.PublicGetInsurance.Wait(ctx)
	case privateSetCancelOnDisconnectRate:
		return r.PrivateSetCancelOnDisconnect.Wait(ctx)
	case privateGetCancelOnDisconnectRate:
		return r.PrivateGetCancelOnDisconnect.Wait(ctx)
	case privateUserBalanceRate:
		return r.PrivateUserBalance.Wait(ctx)
	case privateUserBalanceHistoryRate:
		return r.PrivateUserBalanceHistory.Wait(ctx)
	case privateCreateSubAccountTransferRate:
		return r.PrivateCreateSubAccountTransfer.Wait(ctx)
	case privateGetSubAccountBalancesRate:
		return r.PrivateGetSubAccountBalances.Wait(ctx)
	case privateGetPositionsRate:
		return r.PrivateGetPositions.Wait(ctx)
	case privateCreateOrderRate:
		return r.PrivateCreateOrder.Wait(ctx)
	case privateCancelOrderRate:
		return r.PrivateCancelOrder.Wait(ctx)
	case privateCreateOrderListRate:
		return r.PrivateCreateOrderList.Wait(ctx)
	case privateCancelOrderListRate:
		return r.PrivateCancelOrderList.Wait(ctx)
	case privateGetOrderListRate:
		return r.PrivateGetOrderList.Wait(ctx)
	case privateCancelAllOrdersRate:
		return r.PrivateCancelAllOrders.Wait(ctx)
	case privateClosePositionRate:
		return r.PrivateClosePosition.Wait(ctx)
	case privateGetOrderHistoryRate:
		return r.PrivateGetOrderHistory.Wait(ctx)
	case privateGetOpenOrdersRate:
		return r.PrivateGetOpenOrders.Wait(ctx)
	case privateGetOrderDetailRate:
		return r.PrivateGetOrderDetail.Wait(ctx)
	case privateGetTradesRate:
		return r.PrivateGetTrades.Wait(ctx)
	case privateChangeAccountLeverageRate:
		return r.PrivateChangeAccountLeverage.Wait(ctx)
	case privateGetTransactionsRate:
		return r.PrivateGetTransactions.Wait(ctx)
	case postWithdrawalRate:
		return r.PostWithdrawal.Wait(ctx)
	case privateGetCurrencyNetworksRate:
		return r.PrivateGetCurrencyNetworks.Wait(ctx)
	case privategetDepositAddressRate:
		return r.PrivategetDepositAddress.Wait(ctx)
	case privateGetAccountsRate:
		return r.PrivateGetAccounts.Wait(ctx)
	case privateGetOTCUserRate:
		return r.PrivateGetOTCUser.Wait(ctx)
	case privateGetOTCInstrumentsRate:
		return r.PrivateGetOTCInstruments.Wait(ctx)
	case privateOTCRequestQuoteRate:
		return r.PrivateOTCRequestQuote.Wait(ctx)
	case privateOTCAcceptQuoteRate:
		return r.PrivateOTCAcceptQuote.Wait(ctx)
	case privateGetOTCQuoteHistoryRate:
		return r.PrivateGetOTCQuoteHistory.Wait(ctx)
	case privateGetOTCTradeHistoryRate:
		return r.PrivateGetOTCTradeHistory.Wait(ctx)
	case privateGetWithdrawalHistoryRate:
		return r.PrivateGetWithdrawalHistory.Wait(ctx)
	case privateGetDepositHistoryRate:
		return r.PrivateGetDepositHistory.Wait(ctx)
	case privateGetAccountSummaryRate:
		return r.PrivateGetAccountSummary.Wait(ctx)
	default:
		return errors.New("endpoint rate limit functionality not found")
	}
}

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() *RateLimiter {
	return &RateLimiter{
		PublicAuth:                      request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicInstruments:               request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicOrderbook:                 request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicCandlestick:               request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicTicker:                    request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicTrades:                    request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicGetValuations:             request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicGetExpiredSettlementPrice: request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PublicGetInsurance:              request.NewRateLimit(oneSecondInterval, hundredPerInterval),
		PrivateUserBalance:              request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateUserBalanceHistory:       request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateCreateSubAccountTransfer: request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetSubAccountBalances:    request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetPositions:             request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateCreateOrder:              request.NewRateLimit(hundredMilliSecondsInterval, fifteenPerIntrval),
		PrivateCancelOrder:              request.NewRateLimit(hundredMilliSecondsInterval, fifteenPerIntrval),
		PrivateCreateOrderList:          request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateCancelOrderList:          request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOrderList:             request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateCancelAllOrders:          request.NewRateLimit(hundredMilliSecondsInterval, fifteenPerIntrval),
		PrivateClosePosition:            request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOrderHistory:          request.NewRateLimit(oneSecondInterval, onePerInterval),
		PrivateGetOpenOrders:            request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOrderDetail:           request.NewRateLimit(hundredMilliSecondsInterval, thirtyPerInterval),
		PrivateGetTrades:                request.NewRateLimit(oneSecondInterval, onePerInterval),
		PrivateChangeAccountLeverage:    request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetTransactions:          request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PostWithdrawal:                  request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetCurrencyNetworks:      request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivategetDepositAddress:        request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetAccounts:              request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOTCUser:               request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOTCInstruments:        request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateOTCRequestQuote:          request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateOTCAcceptQuote:           request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOTCQuoteHistory:       request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetOTCTradeHistory:       request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetWithdrawalHistory:     request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetDepositHistory:        request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
		PrivateGetAccountSummary:        request.NewRateLimit(hundredMilliSecondsInterval, threePerInterval),
	}
}
