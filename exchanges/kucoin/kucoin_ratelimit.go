package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	threeSecondsInterval = time.Second * 3
	oneMinuteInterval    = time.Minute
)

// rate of request per interval
const (
	retrieveAccountLedgerRate              = 18
	masterSubUserTransferRate              = 3
	retrieveDepositListRate                = 6
	retrieveV1HistoricalDepositListRate    = 6
	retrieveWithdrawalListRate             = 6
	retrieveV1HistoricalWithdrawalListRate = 6
	placeOrderRate                         = 45
	placeMarginOrdersRate                  = 45
	placeBulkOrdersRate                    = 3
	cancelOrderRate                        = 60
	cancelAllOrdersRate                    = 3
	listOrdersRate                         = 30
	listFillsRate                          = 9
	retrieveFullOrderbookRate              = 30
	retrieveMarginAccountRate              = 1

	futuresRetrieveAccountOverviewRate     = 30
	futuresRetrieveTransactionHistoryRate  = 9
	futuresPlaceOrderRate                  = 30
	futuresCancelAnOrderRate               = 40
	futuresLimitOrderMassCancelationRate   = 9
	futuresRetrieveOrderListRate           = 30
	futuresRetrieveFillsRate               = 9
	futuresRecentFillsRate                 = 9
	futuresRetrievePositionListRate        = 9
	futuresRetrieveFundingHistoryRate      = 9
	futuresRetrieveFullOrderbookLevel2Rate = 30

	defaultSpotRate    = 1200
	defaultFuturesRate = 1200
)

const (
	// for spot endpoints
	retrieveAccountLedgerEPL request.EndpointLimit = iota
	masterSubUserTransferEPL
	retrieveDepositListEPL
	retrieveV1HistoricalDepositListEPL
	retrieveWithdrawalListEPL
	retrieveV1HistoricalWithdrawalListEPL
	placeOrderEPL
	placeMarginOrdersEPL
	placeBulkOrdersEPL
	cancelOrderEPL
	cancelAllOrdersEPL
	listOrdersEPL
	listFillsEPL
	retrieveFullOrderbookEPL
	retrieveMarginAccountEPL
	defaultSpotEPL

	// for futures endpoints
	futuresRetrieveAccountOverviewEPL
	futuresRetrieveTransactionHistoryEPL
	futuresPlaceOrderEPL
	futuresCancelAnOrderEPL
	futuresLimitOrderMassCancelationEPL
	futuresRetrieveOrderListEPL
	futuresRetrieveFillsEPL
	futuresRecentFillsEPL
	futuresRetrievePositionListEPL
	futuresRetrieveFundingHistoryEPL
	futuresRetrieveFullOrderbookLevel2EPL
	defaultFuturesEPL
)

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// spot specific rate limiters
		retrieveAccountLedgerEPL:              request.NewRateLimitWithToken(threeSecondsInterval, retrieveAccountLedgerRate, 1),
		masterSubUserTransferEPL:              request.NewRateLimitWithToken(threeSecondsInterval, masterSubUserTransferRate, 1),
		retrieveDepositListEPL:                request.NewRateLimitWithToken(threeSecondsInterval, retrieveDepositListRate, 1),
		retrieveV1HistoricalDepositListEPL:    request.NewRateLimitWithToken(threeSecondsInterval, retrieveV1HistoricalDepositListRate, 1),
		retrieveWithdrawalListEPL:             request.NewRateLimitWithToken(threeSecondsInterval, retrieveWithdrawalListRate, 1),
		retrieveV1HistoricalWithdrawalListEPL: request.NewRateLimitWithToken(threeSecondsInterval, retrieveV1HistoricalWithdrawalListRate, 1),
		placeOrderEPL:                         request.NewRateLimitWithToken(threeSecondsInterval, placeOrderRate, 1),
		placeMarginOrdersEPL:                  request.NewRateLimitWithToken(threeSecondsInterval, placeMarginOrdersRate, 1),
		placeBulkOrdersEPL:                    request.NewRateLimitWithToken(threeSecondsInterval, placeBulkOrdersRate, 1),
		cancelOrderEPL:                        request.NewRateLimitWithToken(threeSecondsInterval, cancelOrderRate, 1),
		cancelAllOrdersEPL:                    request.NewRateLimitWithToken(threeSecondsInterval, cancelAllOrdersRate, 1),
		listOrdersEPL:                         request.NewRateLimitWithToken(threeSecondsInterval, listOrdersRate, 1),
		listFillsEPL:                          request.NewRateLimitWithToken(threeSecondsInterval, listFillsRate, 1),
		retrieveFullOrderbookEPL:              request.NewRateLimitWithToken(threeSecondsInterval, retrieveFullOrderbookRate, 1),
		retrieveMarginAccountEPL:              request.NewRateLimitWithToken(threeSecondsInterval, retrieveMarginAccountRate, 1),

		// default spot and futures rates
		defaultSpotEPL:    request.NewRateLimitWithToken(oneMinuteInterval, defaultSpotRate, 1),
		defaultFuturesEPL: request.NewRateLimitWithToken(oneMinuteInterval, defaultFuturesRate, 1),

		// futures specific rate limiters
		futuresRetrieveAccountOverviewEPL:     request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveAccountOverviewRate, 1),
		futuresRetrieveTransactionHistoryEPL:  request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveTransactionHistoryRate, 1),
		futuresPlaceOrderEPL:                  request.NewRateLimitWithToken(threeSecondsInterval, futuresPlaceOrderRate, 1),
		futuresCancelAnOrderEPL:               request.NewRateLimitWithToken(threeSecondsInterval, futuresCancelAnOrderRate, 1),
		futuresLimitOrderMassCancelationEPL:   request.NewRateLimitWithToken(threeSecondsInterval, futuresLimitOrderMassCancelationRate, 1),
		futuresRetrieveOrderListEPL:           request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveOrderListRate, 1),
		futuresRetrieveFillsEPL:               request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveFillsRate, 1),
		futuresRecentFillsEPL:                 request.NewRateLimitWithToken(threeSecondsInterval, futuresRecentFillsRate, 1),
		futuresRetrievePositionListEPL:        request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrievePositionListRate, 1),
		futuresRetrieveFundingHistoryEPL:      request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveFundingHistoryRate, 1),
		futuresRetrieveFullOrderbookLevel2EPL: request.NewRateLimitWithToken(threeSecondsInterval, futuresRetrieveFullOrderbookLevel2Rate, 1),
	}
}
