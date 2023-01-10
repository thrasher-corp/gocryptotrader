package kucoin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	threeSecondsInterval = time.Second * 3
	oneMinuteInterval    = time.Minute
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	RetriveAccountLedger              *rate.Limiter
	MasterSubUserTransfer             *rate.Limiter
	RetriveDepositList                *rate.Limiter
	RetriveV1HistoricalDepositList    *rate.Limiter
	RetriveWithdrawalList             *rate.Limiter
	RetriveV1HistoricalWithdrawalList *rate.Limiter
	PlaceOrder                        *rate.Limiter
	PlaceMarginOrders                 *rate.Limiter
	PlaceBulkOrders                   *rate.Limiter
	CancelOrder                       *rate.Limiter
	CancelAllOrders                   *rate.Limiter
	ListOrders                        *rate.Limiter
	ListFills                         *rate.Limiter
	RetriveFullOrderbook              *rate.Limiter
	RetriveMarginAccount              *rate.Limiter
	SpotRate                          *rate.Limiter
	FuturesRate                       *rate.Limiter

	FRetriveAccountOverviewRate     *rate.Limiter
	FRetriveTransactionHistoryRate  *rate.Limiter
	FPlaceOrderRate                 *rate.Limiter
	FCancelAnOrderRate              *rate.Limiter
	FLimitOrderMassCancelationRate  *rate.Limiter
	FRetriveOrderListRate           *rate.Limiter
	FRetriveFillsRate               *rate.Limiter
	FRecentFillsRate                *rate.Limiter
	FRetrivePositionListRate        *rate.Limiter
	FRetriveFundingHistoryRate      *rate.Limiter
	FRetriveFullOrderbookLevel2Rate *rate.Limiter
}

// rate of request per interval
const (
	retriveAccountLedgerRate              = 18
	masterSubUserTransferRate             = 3
	retriveDepositListRate                = 6
	retriveV1HistoricalDepositListRate    = 6
	retriveWithdrawalListRate             = 6
	retriveV1HistoricalWithdrawalListRate = 6
	placeOrderRate                        = 45
	placeMarginOrdersRate                 = 45
	placeBulkOrdersRate                   = 3
	cancelOrderRate                       = 60
	cancelAllOrdersRate                   = 3
	listOrdersRate                        = 30
	listFillsRate                         = 9
	retriveFullOrderbookRate              = 30
	retriveMarginAccountRate              = 1

	futuresRetriveAccountOverviewRate     = 30
	futuresRetriveTransactionHistoryRate  = 9
	futuresPlaceOrderRate                 = 30
	futuresCancelAnOrderRate              = 40
	futuresLimitOrderMassCancelationRate  = 9
	futuresRetriveOrderListRate           = 30
	futuresRetriveFillsRate               = 9
	futuresRecentFillsRate                = 9
	futuresRetrivePositionListRate        = 9
	futuresRetriveFundingHistoryRate      = 9
	futuresRetriveFullOrderbookLevel2Rate = 30

	defaultSpotRate    = 1200
	defaultFuturesRate = 1200
)

const (
	// for spot endpoints
	retriveAccountLedgerEPL request.EndpointLimit = iota
	masterSubUserTransferEPL
	retriveDepositListEPL
	retriveV1HistoricalDepositListEPL
	retriveWithdrawalListEPL
	retriveV1HistoricalWithdrawalListEPL
	placeOrderEPL
	placeMarginOrdersEPL
	placeBulkOrdersEPL
	cancelOrderEPL
	cancelAllOrdersEPL
	listOrdersEPL
	listFillsEPL
	retriveFullOrderbookEPL
	retriveMarginAccountEPL
	defaultSpotEPL
	defaultFuturesEPL

	// for futures endpoints
	futuresRetriveAccountOverviewEPL
	futuresRetriveTransactionHistoryEPL
	futuresPlaceOrderEPL
	futuresCancelAnOrderEPL
	futuresLimitOrderMassCancelationEPL
	futuresRetriveOrderListEPL
	futuresRetriveFillsEPL
	futuresRecentFillsEPL
	futuresRetrivePositionListEPL
	futuresRetriveFundingHistoryEPL
	futuresRetriveFullOrderbookLevel2EPL
)

// Limit executes rate limiting functionality for Kucoin
func (r *RateLimit) Limit(ctx context.Context, epl request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch epl {
	case retriveAccountLedgerEPL:
		return r.RetriveAccountLedger.Wait(ctx)
	case masterSubUserTransferEPL:
		return r.MasterSubUserTransfer.Wait(ctx)
	case retriveDepositListEPL:
		return r.RetriveDepositList.Wait(ctx)
	case retriveV1HistoricalDepositListEPL:
		return r.RetriveV1HistoricalDepositList.Wait(ctx)
	case retriveWithdrawalListEPL:
		return r.RetriveWithdrawalList.Wait(ctx)
	case retriveV1HistoricalWithdrawalListEPL:
		return r.RetriveV1HistoricalWithdrawalList.Wait(ctx)
	case placeOrderEPL:
		return r.PlaceOrder.Wait(ctx)
	case placeMarginOrdersEPL:
		return r.PlaceMarginOrders.Wait(ctx)
	case placeBulkOrdersEPL:
		return r.PlaceBulkOrders.Wait(ctx)
	case cancelOrderEPL:
		return r.CancelOrder.Wait(ctx)
	case cancelAllOrdersEPL:
		return r.CancelAllOrders.Wait(ctx)
	case listOrdersEPL:
		return r.ListOrders.Wait(ctx)
	case listFillsEPL:
		return r.ListFills.Wait(ctx)
	case retriveFullOrderbookEPL:
		return r.RetriveFullOrderbook.Wait(ctx)
	case retriveMarginAccountEPL:
		return r.RetriveMarginAccount.Wait(ctx)
	case futuresRetriveAccountOverviewEPL:
		return r.FRetriveAccountOverviewRate.Wait(ctx)
	case futuresRetriveTransactionHistoryEPL:
		return r.FRetriveTransactionHistoryRate.Wait(ctx)
	case futuresPlaceOrderEPL:
		return r.FPlaceOrderRate.Wait(ctx)
	case futuresCancelAnOrderEPL:
		return r.FCancelAnOrderRate.Wait(ctx)
	case futuresLimitOrderMassCancelationEPL:
		return r.FLimitOrderMassCancelationRate.Wait(ctx)
	case futuresRetriveOrderListEPL:
		return r.FRetriveOrderListRate.Wait(ctx)
	case futuresRetriveFillsEPL:
		return r.FRetriveFillsRate.Wait(ctx)
	case futuresRecentFillsEPL:
		return r.FRecentFillsRate.Wait(ctx)
	case futuresRetrivePositionListEPL:
		return r.FRetrivePositionListRate.Wait(ctx)
	case futuresRetriveFundingHistoryEPL:
		return r.FRetriveFundingHistoryRate.Wait(ctx)
	case futuresRetriveFullOrderbookLevel2EPL:
		return r.FRetriveFullOrderbookLevel2Rate.Wait(ctx)
	case defaultSpotEPL:
		limiter, tokens = r.SpotRate, 1
	case defaultFuturesEPL:
		limiter, tokens = r.FuturesRate, 1
	default:
		return errors.New("endpoint rate limit functionality not found")
	}
	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		// Cancel all potential reservations to free up rate limiter if deadline
		// is exceeded.
		for x := range reserves {
			reserves[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	time.Sleep(finalDelay)
	return nil
}

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() *RateLimit {
	return &RateLimit{
		// spot specific rate limiters
		RetriveAccountLedger:              request.NewRateLimit(threeSecondsInterval, retriveAccountLedgerRate),
		MasterSubUserTransfer:             request.NewRateLimit(threeSecondsInterval, masterSubUserTransferRate),
		RetriveDepositList:                request.NewRateLimit(threeSecondsInterval, retriveDepositListRate),
		RetriveV1HistoricalDepositList:    request.NewRateLimit(threeSecondsInterval, retriveV1HistoricalDepositListRate),
		RetriveWithdrawalList:             request.NewRateLimit(threeSecondsInterval, retriveWithdrawalListRate),
		RetriveV1HistoricalWithdrawalList: request.NewRateLimit(threeSecondsInterval, retriveV1HistoricalWithdrawalListRate),
		PlaceOrder:                        request.NewRateLimit(threeSecondsInterval, placeOrderRate),
		PlaceMarginOrders:                 request.NewRateLimit(threeSecondsInterval, placeMarginOrdersRate),
		PlaceBulkOrders:                   request.NewRateLimit(threeSecondsInterval, placeBulkOrdersRate),
		CancelOrder:                       request.NewRateLimit(threeSecondsInterval, cancelOrderRate),
		CancelAllOrders:                   request.NewRateLimit(threeSecondsInterval, cancelAllOrdersRate),
		ListOrders:                        request.NewRateLimit(threeSecondsInterval, listOrdersRate),
		ListFills:                         request.NewRateLimit(threeSecondsInterval, listFillsRate),
		RetriveFullOrderbook:              request.NewRateLimit(threeSecondsInterval, retriveFullOrderbookRate),
		RetriveMarginAccount:              request.NewRateLimit(threeSecondsInterval, retriveMarginAccountRate),

		// default spot and futures rates
		SpotRate:    request.NewRateLimit(oneMinuteInterval, defaultSpotRate),
		FuturesRate: request.NewRateLimit(oneMinuteInterval, defaultFuturesRate),

		// futures specific rate limiters
		FRetriveAccountOverviewRate:     request.NewRateLimit(threeSecondsInterval, futuresRetriveAccountOverviewRate),
		FRetriveTransactionHistoryRate:  request.NewRateLimit(threeSecondsInterval, futuresRetriveTransactionHistoryRate),
		FPlaceOrderRate:                 request.NewRateLimit(threeSecondsInterval, futuresPlaceOrderRate),
		FCancelAnOrderRate:              request.NewRateLimit(threeSecondsInterval, futuresCancelAnOrderRate),
		FLimitOrderMassCancelationRate:  request.NewRateLimit(threeSecondsInterval, futuresLimitOrderMassCancelationRate),
		FRetriveOrderListRate:           request.NewRateLimit(threeSecondsInterval, futuresRetriveOrderListRate),
		FRetriveFillsRate:               request.NewRateLimit(threeSecondsInterval, futuresRetriveFillsRate),
		FRecentFillsRate:                request.NewRateLimit(threeSecondsInterval, futuresRecentFillsRate),
		FRetrivePositionListRate:        request.NewRateLimit(threeSecondsInterval, futuresRetrivePositionListRate),
		FRetriveFundingHistoryRate:      request.NewRateLimit(threeSecondsInterval, futuresRetriveFundingHistoryRate),
		FRetriveFullOrderbookLevel2Rate: request.NewRateLimit(threeSecondsInterval, futuresRetriveFullOrderbookLevel2Rate),
	}
}
