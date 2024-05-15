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

// GetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		// spot specific rate limiters
		retrieveAccountLedgerEPL:              request.NewRateLimitWithWeight(threeSecondsInterval, retrieveAccountLedgerRate, 1),
		masterSubUserTransferEPL:              request.NewRateLimitWithWeight(threeSecondsInterval, masterSubUserTransferRate, 1),
		retrieveDepositListEPL:                request.NewRateLimitWithWeight(threeSecondsInterval, retrieveDepositListRate, 1),
		retrieveV1HistoricalDepositListEPL:    request.NewRateLimitWithWeight(threeSecondsInterval, retrieveV1HistoricalDepositListRate, 1),
		retrieveWithdrawalListEPL:             request.NewRateLimitWithWeight(threeSecondsInterval, retrieveWithdrawalListRate, 1),
		retrieveV1HistoricalWithdrawalListEPL: request.NewRateLimitWithWeight(threeSecondsInterval, retrieveV1HistoricalWithdrawalListRate, 1),
		placeOrderEPL:                         request.NewRateLimitWithWeight(threeSecondsInterval, placeOrderRate, 1),
		placeMarginOrdersEPL:                  request.NewRateLimitWithWeight(threeSecondsInterval, placeMarginOrdersRate, 1),
		placeBulkOrdersEPL:                    request.NewRateLimitWithWeight(threeSecondsInterval, placeBulkOrdersRate, 1),
		cancelOrderEPL:                        request.NewRateLimitWithWeight(threeSecondsInterval, cancelOrderRate, 1),
		cancelAllOrdersEPL:                    request.NewRateLimitWithWeight(threeSecondsInterval, cancelAllOrdersRate, 1),
		listOrdersEPL:                         request.NewRateLimitWithWeight(threeSecondsInterval, listOrdersRate, 1),
		listFillsEPL:                          request.NewRateLimitWithWeight(threeSecondsInterval, listFillsRate, 1),
		retrieveFullOrderbookEPL:              request.NewRateLimitWithWeight(threeSecondsInterval, retrieveFullOrderbookRate, 1),
		retrieveMarginAccountEPL:              request.NewRateLimitWithWeight(threeSecondsInterval, retrieveMarginAccountRate, 1),

		// default spot and futures rates
		defaultSpotEPL:    request.NewRateLimitWithWeight(oneMinuteInterval, defaultSpotRate, 1),
		defaultFuturesEPL: request.NewRateLimitWithWeight(oneMinuteInterval, defaultFuturesRate, 1),

		// futures specific rate limiters
		futuresRetrieveAccountOverviewEPL:     request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveAccountOverviewRate, 1),
		futuresRetrieveTransactionHistoryEPL:  request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveTransactionHistoryRate, 1),
		futuresPlaceOrderEPL:                  request.NewRateLimitWithWeight(threeSecondsInterval, futuresPlaceOrderRate, 1),
		futuresCancelAnOrderEPL:               request.NewRateLimitWithWeight(threeSecondsInterval, futuresCancelAnOrderRate, 1),
		futuresLimitOrderMassCancelationEPL:   request.NewRateLimitWithWeight(threeSecondsInterval, futuresLimitOrderMassCancelationRate, 1),
		futuresRetrieveOrderListEPL:           request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveOrderListRate, 1),
		futuresRetrieveFillsEPL:               request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveFillsRate, 1),
		futuresRecentFillsEPL:                 request.NewRateLimitWithWeight(threeSecondsInterval, futuresRecentFillsRate, 1),
		futuresRetrievePositionListEPL:        request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrievePositionListRate, 1),
		futuresRetrieveFundingHistoryEPL:      request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveFundingHistoryRate, 1),
		futuresRetrieveFullOrderbookLevel2EPL: request.NewRateLimitWithWeight(threeSecondsInterval, futuresRetrieveFullOrderbookLevel2Rate, 1),
	}
}
