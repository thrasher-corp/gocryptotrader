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
	fifteenPerInterval = 15
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
	publicValuationRate
	publicTradesRate
	getValuationsRate
	getInsuranceRate
	expiredSettlementPriceRate
	getAnnouncementsRate
	publicGetValuationsRate
	publicGetExpiredSettlementPriceRate
	publicGetInsuranceRate
	privateUserBalanceRate
	privateUserBalanceHistoryRate
	privateCreateSubAccountTransferRate
	privateGetSubAccountBalancesRate
	privateGetPositionsRate
	privateCreateOrderRate
	privateAmendOrderRate
	privateCancelOrderRate
	privateCreateOrderListRate
	privateCancelOrderListRate
	privateGetOrderListRate
	privateCancelAllOrdersRate
	privateClosePositionRate
	changeAccountLeverageRate
	changeAccountSettingRate
	getAllExecutableTradesRate
	closePositionRate
	futuresOrderListRate
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
	privateCreateSubAccountRate
	privateGetOTCUserRate
	privateGetOTCInstrumentsRate
	privateOTCRequestQuoteRate
	privateOTCAcceptQuoteRate
	privateGetOTCQuoteHistoryRate
	privateGetOTCTradeHistoryRate
	privateCreateOTCOrderRate
	privateGetWithdrawalHistoryRate
	privateGetDepositHistoryRate
	privateGetAccountSummaryRate
	createExportRequestRate
	getExportRequestRate
	getPositionsRate
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		publicAuthRate:                      request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicInstrumentsRate:               request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicOrderbookRate:                 request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicCandlestickRate:               request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicTickerRate:                    request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicValuationRate:                 request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicTradesRate:                    request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		getValuationsRate:                   request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		getInsuranceRate:                    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		expiredSettlementPriceRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		getAnnouncementsRate:                request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		publicGetValuationsRate:             request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicGetExpiredSettlementPriceRate: request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		publicGetInsuranceRate:              request.NewRateLimitWithWeight(oneSecondInterval, hundredPerInterval, 1),
		privateUserBalanceRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateUserBalanceHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCreateSubAccountTransferRate: request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetSubAccountBalancesRate:    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetPositionsRate:             request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCreateOrderRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerInterval, 1),
		privateAmendOrderRate:               request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerInterval, 1),
		privateCancelOrderRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerInterval, 1),
		privateCreateOrderListRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCancelOrderListRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOrderListRate:             request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCancelAllOrdersRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, fifteenPerInterval, 1),
		privateClosePositionRate:            request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		changeAccountLeverageRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		changeAccountSettingRate:            request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		getAllExecutableTradesRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		closePositionRate:                   request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		futuresOrderListRate:                request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOrderHistoryRate:          request.NewRateLimitWithWeight(oneSecondInterval, onePerInterval, 1),
		privateGetOpenOrdersRate:            request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOrderDetailRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, thirtyPerInterval, 1),
		privateGetTradesRate:                request.NewRateLimitWithWeight(oneSecondInterval, onePerInterval, 1),
		privateChangeAccountLeverageRate:    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetTransactionsRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		postWithdrawalRate:                  request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetCurrencyNetworksRate:      request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privategetDepositAddressRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetAccountsRate:              request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCreateSubAccountRate:         request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOTCUserRate:               request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOTCInstrumentsRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateOTCRequestQuoteRate:          request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateOTCAcceptQuoteRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOTCQuoteHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetOTCTradeHistoryRate:       request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateCreateOTCOrderRate:           request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetWithdrawalHistoryRate:     request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetDepositHistoryRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		privateGetAccountSummaryRate:        request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		createExportRequestRate:             request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		getExportRequestRate:                request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
		getPositionsRate:                    request.NewRateLimitWithWeight(hundredMilliSecondsInterval, threePerInterval, 1),
	}
}
