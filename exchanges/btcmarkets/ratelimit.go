package btcmarkets

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// BTCMarkets Rate limit consts
const (
	btcmarketsRateInterval         = time.Second * 10
	btcmarketsGeneralLimit         = 25
	btcmarketsOrderLimit           = 30
	btcmarketsBatchOrderLimit      = 5
	btcmarketsWithdrawLimit        = 10
	btcmarketsCreateNewReportLimit = 1

	// Used to match endpints to rate limits
	orderFunc request.EndpointLimit = iota
	batchFunc
	withdrawFunc
	newReportFunc
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	General         *rate.Limiter
	OrderPlacement  *rate.Limiter
	BatchOrders     *rate.Limiter
	WithdrawRequest *rate.Limiter
	CreateNewReport *rate.Limiter
}

// Limit limits the outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case orderFunc:
		time.Sleep(r.OrderPlacement.Reserve().Delay())
	case batchFunc:
		time.Sleep(r.BatchOrders.Reserve().Delay())
	case withdrawFunc:
		time.Sleep(r.WithdrawRequest.Reserve().Delay())
	case newReportFunc:
		time.Sleep(r.CreateNewReport.Reserve().Delay())
	default:
		time.Sleep(r.General.Reserve().Delay())
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		General:         request.NewRateLimit(btcmarketsRateInterval, btcmarketsGeneralLimit),
		OrderPlacement:  request.NewRateLimit(btcmarketsRateInterval, btcmarketsOrderLimit),
		BatchOrders:     request.NewRateLimit(btcmarketsRateInterval, btcmarketsBatchOrderLimit),
		WithdrawRequest: request.NewRateLimit(btcmarketsRateInterval, btcmarketsWithdrawLimit),
		CreateNewReport: request.NewRateLimit(btcmarketsRateInterval, btcmarketsCreateNewReportLimit),
	}
}
