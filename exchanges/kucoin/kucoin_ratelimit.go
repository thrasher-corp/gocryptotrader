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
	RetrieveAccountLedger              *rate.Limiter
	MasterSubUserTransfer              *rate.Limiter
	RetrieveDepositList                *rate.Limiter
	RetrieveV1HistoricalDepositList    *rate.Limiter
	RetrieveWithdrawalList             *rate.Limiter
	RetrieveV1HistoricalWithdrawalList *rate.Limiter
	PlaceOrder                         *rate.Limiter
	PlaceMarginOrders                  *rate.Limiter
	PlaceBulkOrders                    *rate.Limiter
	CancelOrder                        *rate.Limiter
	CancelAllOrders                    *rate.Limiter
	ListOrders                         *rate.Limiter
	ListFills                          *rate.Limiter
	RetrieveFullOrderbook              *rate.Limiter
	RetrieveMarginAccount              *rate.Limiter
	SpotRate                           *rate.Limiter
	FuturesRate                        *rate.Limiter

	FRetrieveAccountOverviewRate     *rate.Limiter
	FRetrieveTransactionHistoryRate  *rate.Limiter
	FPlaceOrderRate                  *rate.Limiter
	FCancelAnOrderRate               *rate.Limiter
	FLimitOrderMassCancelationRate   *rate.Limiter
	FRetrieveOrderListRate           *rate.Limiter
	FRetrieveFillsRate               *rate.Limiter
	FRecentFillsRate                 *rate.Limiter
	FRetrievePositionListRate        *rate.Limiter
	FRetrieveFundingHistoryRate      *rate.Limiter
	FRetrieveFullOrderbookLevel2Rate *rate.Limiter
}

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

// Limit executes rate limiting functionality for Kucoin
func (r *RateLimit) Limit(ctx context.Context, epl request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch epl {
	case retrieveAccountLedgerEPL:
		return r.RetrieveAccountLedger.Wait(ctx)
	case masterSubUserTransferEPL:
		return r.MasterSubUserTransfer.Wait(ctx)
	case retrieveDepositListEPL:
		return r.RetrieveDepositList.Wait(ctx)
	case retrieveV1HistoricalDepositListEPL:
		return r.RetrieveV1HistoricalDepositList.Wait(ctx)
	case retrieveWithdrawalListEPL:
		return r.RetrieveWithdrawalList.Wait(ctx)
	case retrieveV1HistoricalWithdrawalListEPL:
		return r.RetrieveV1HistoricalWithdrawalList.Wait(ctx)
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
	case retrieveFullOrderbookEPL:
		return r.RetrieveFullOrderbook.Wait(ctx)
	case retrieveMarginAccountEPL:
		return r.RetrieveMarginAccount.Wait(ctx)
	case futuresRetrieveAccountOverviewEPL:
		return r.FRetrieveAccountOverviewRate.Wait(ctx)
	case futuresRetrieveTransactionHistoryEPL:
		return r.FRetrieveTransactionHistoryRate.Wait(ctx)
	case futuresPlaceOrderEPL:
		return r.FPlaceOrderRate.Wait(ctx)
	case futuresCancelAnOrderEPL:
		return r.FCancelAnOrderRate.Wait(ctx)
	case futuresLimitOrderMassCancelationEPL:
		return r.FLimitOrderMassCancelationRate.Wait(ctx)
	case futuresRetrieveOrderListEPL:
		return r.FRetrieveOrderListRate.Wait(ctx)
	case futuresRetrieveFillsEPL:
		return r.FRetrieveFillsRate.Wait(ctx)
	case futuresRecentFillsEPL:
		return r.FRecentFillsRate.Wait(ctx)
	case futuresRetrievePositionListEPL:
		return r.FRetrievePositionListRate.Wait(ctx)
	case futuresRetrieveFundingHistoryEPL:
		return r.FRetrieveFundingHistoryRate.Wait(ctx)
	case futuresRetrieveFullOrderbookLevel2EPL:
		return r.FRetrieveFullOrderbookLevel2Rate.Wait(ctx)
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
		RetrieveAccountLedger:              request.NewRateLimit(threeSecondsInterval, 18),
		MasterSubUserTransfer:              request.NewRateLimit(threeSecondsInterval, 3),
		RetrieveDepositList:                request.NewRateLimit(threeSecondsInterval, 6),
		RetrieveV1HistoricalDepositList:    request.NewRateLimit(threeSecondsInterval, 6),
		RetrieveWithdrawalList:             request.NewRateLimit(threeSecondsInterval, 6),
		RetrieveV1HistoricalWithdrawalList: request.NewRateLimit(threeSecondsInterval, 6),
		PlaceOrder:                         request.NewRateLimit(threeSecondsInterval, 45),
		PlaceMarginOrders:                  request.NewRateLimit(threeSecondsInterval, 45),
		PlaceBulkOrders:                    request.NewRateLimit(threeSecondsInterval, 3),
		CancelOrder:                        request.NewRateLimit(threeSecondsInterval, 60),
		CancelAllOrders:                    request.NewRateLimit(threeSecondsInterval, 3),
		ListOrders:                         request.NewRateLimit(threeSecondsInterval, 30),
		ListFills:                          request.NewRateLimit(threeSecondsInterval, 9),
		RetrieveFullOrderbook:              request.NewRateLimit(threeSecondsInterval, 30),
		RetrieveMarginAccount:              request.NewRateLimit(threeSecondsInterval, 1),

		// default spot and futures rates
		SpotRate:    request.NewRateLimit(oneMinuteInterval, 1200),
		FuturesRate: request.NewRateLimit(oneMinuteInterval, 1200),

		// futures specific rate limiters
		FRetrieveAccountOverviewRate:     request.NewRateLimit(threeSecondsInterval, 30),
		FRetrieveTransactionHistoryRate:  request.NewRateLimit(threeSecondsInterval, 9),
		FPlaceOrderRate:                  request.NewRateLimit(threeSecondsInterval, 30),
		FCancelAnOrderRate:               request.NewRateLimit(threeSecondsInterval, 40),
		FLimitOrderMassCancelationRate:   request.NewRateLimit(threeSecondsInterval, 9),
		FRetrieveOrderListRate:           request.NewRateLimit(threeSecondsInterval, 30),
		FRetrieveFillsRate:               request.NewRateLimit(threeSecondsInterval, 9),
		FRecentFillsRate:                 request.NewRateLimit(threeSecondsInterval, 9),
		FRetrievePositionListRate:        request.NewRateLimit(threeSecondsInterval, 9),
		FRetrieveFundingHistoryRate:      request.NewRateLimit(threeSecondsInterval, 9),
		FRetrieveFullOrderbookLevel2Rate: request.NewRateLimit(threeSecondsInterval, 30),
	}
}
