package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		publicAuthRate:                      request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicInstrumentsRate:               request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicOrderbookRate:                 request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicCandlestickRate:               request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicTickerRate:                    request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicTradesRate:                    request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicGetValuationsRate:             request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicGetExpiredSettlementPriceRate: request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		publicGetInsuranceRate:              request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, hundredPerInterval),
		privateUserBalanceRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateUserBalanceHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateCreateSubAccountTransferRate: request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetSubAccountBalancesRate:    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetPositionsRate:             request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateCreateOrderRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerIntrval, fifteenPerIntrval),
		privateCancelOrderRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerIntrval, fifteenPerIntrval),
		privateCreateOrderListRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateCancelOrderListRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOrderListRate:             request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateCancelAllOrdersRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerIntrval, fifteenPerIntrval),
		privateClosePositionRate:            request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOrderHistoryRate:          request.NewRateLimitWithWeight(oneSecondInterval, onePerInterval, onePerInterval),
		privateGetOpenOrdersRate:            request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOrderDetailRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, thirtyPerInterval, thirtyPerInterval),
		privateGetTradesRate:                request.NewRateLimitWithWeight(oneSecondInterval, onePerInterval, onePerInterval),
		privateChangeAccountLeverageRate:    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetTransactionsRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		postWithdrawalRate:                  request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetCurrencyNetworksRate:      request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privategetDepositAddressRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetAccountsRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOTCUserRate:               request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOTCInstrumentsRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateOTCRequestQuoteRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateOTCAcceptQuoteRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOTCQuoteHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetOTCTradeHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetWithdrawalHistoryRate:     request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetDepositHistoryRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
		privateGetAccountSummaryRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, threePerInterval),
	}
}
