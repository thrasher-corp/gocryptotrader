package binance

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	binanceGlobalInterval    = time.Minute
	binanceGlobalRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 10 requests per second and max 100000 requests per day.
	binanceOrderInterval      = time.Second
	binanceOrderRequestRate   = 10
	binanceRawRequestInterval = time.Minute
	binanceRawRequests        = 5000

	// Differntiates between different weights for endpoints
	WeightOne request.EndpointLimit = iota
	WeightTwo
	WeightFive
	WeightTen
	WeightFifty
	WeightOrder
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	GlobalRate        int
	Orders            int
	TotalRequestCount int

	global    time.Time
	globalMtx sync.Mutex
	Order     time.Time
	orderMtx  sync.Mutex
	Raw       time.Time
	m         sync.Mutex
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	// switch f {
	// case WeightOne, WeightTwo, WeightFive, WeightTen, WeightFifty:
	// 	return r.rateLimitMe(int(f))
	// case WeightOrder:
	// default:
	// 	return errors.New("unhandled endpoint limit")
	// }
	return r.rateLimitMe(int(f))
}

func (r *RateLimit) rateLimitMe(weight int) error {
	r.m.Lock()
	if time.Now().After(r.Raw) {
		r.Raw = time.Now().Truncate(binanceRawRequests)
		r.TotalRequestCount = 0
	}

	if r.TotalRequestCount+weight >= binanceRawRequests {
		time.Sleep(time.Until(r.Raw))
	}

	r.TotalRequestCount += weight

	if time.Now().After(r.global) {
		r.global = time.Now().Truncate(binanceGlobalInterval)
		r.GlobalRate = 0
	}

	if r.GlobalRate+weight >= binanceGlobalRequestRate {
		time.Sleep(time.Until(r.global))
	}

	r.GlobalRate += weight
	r.m.Unlock()
	return nil
}

// func (r *RateLimit) rateLimitOrderMe() error {

// }

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return new(RateLimit)
}
