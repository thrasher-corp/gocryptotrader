package bitflyer

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Exchange specific rate limit consts
const (
	biflyerRateInterval                 = time.Minute * 5
	bitflyerPrivateRequestRate          = 500
	bitflyerPrivateLowVolumeRequestRate = 100
	bitflyerPrivateSendOrderRequestRate = 300
	bitflyerPublicRequestRate           = 500
)

// RateLimit implements the rate.Limiter interface
type RateLimit struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter

	// Send a New Order
	// Submit New Parent Order (Special order)
	// Cancel All Orders
	Order     *rate.Limiter
	LowVolume *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case request.Auth:
		time.Sleep(r.Auth.Reserve().Delay())
	case orders:
		res := r.Auth.Reserve()
		time.Sleep(r.Order.Reserve().Delay())
		time.Sleep(res.Delay())
	case lowVolume:
		authShell := r.Auth.Reserve()
		orderShell := r.Order.Reserve()
		time.Sleep(r.LowVolume.Reserve().Delay())
		time.Sleep(orderShell.Delay())
		time.Sleep(authShell.Delay())
	default:
		time.Sleep(r.UnAuth.Reserve().Delay())
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:      request.NewRateLimit(biflyerRateInterval, bitflyerPrivateRequestRate),
		UnAuth:    request.NewRateLimit(biflyerRateInterval, bitflyerPublicRequestRate),
		Order:     request.NewRateLimit(biflyerRateInterval, bitflyerPrivateSendOrderRequestRate),
		LowVolume: request.NewRateLimit(time.Minute, bitflyerPrivateLowVolumeRequestRate),
	}
}
