package hitbtc

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	hitbtcRateInterval      = time.Second
	hitbtcMarketDataReqRate = 100
	hitbtcTradingReqRate    = 300
	hitbtcAllOthers         = 10

	marketRequests request.EndpointLimit = iota
	tradingRequests
	otherRequests
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	MarketData *rate.Limiter
	Trading    *rate.Limiter
	Other      *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case marketRequests:
		time.Sleep(r.MarketData.Reserve().Delay())
	case tradingRequests:
		time.Sleep(r.Trading.Reserve().Delay())
	case otherRequests:
		time.Sleep(r.Other.Reserve().Delay())
	default:
		return errors.New("functionality not found")
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		MarketData: request.NewRateLimit(hitbtcRateInterval, hitbtcMarketDataReqRate),
		Trading:    request.NewRateLimit(hitbtcRateInterval, hitbtcTradingReqRate),
		Other:      request.NewRateLimit(hitbtcRateInterval, hitbtcAllOthers),
	}
}
