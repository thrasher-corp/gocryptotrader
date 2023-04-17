package bitflyer

import (
	"context"
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
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, request.Tokens, error) {
	// TODO: Authenticated endpoints cannot be tested.
	switch f {
	case request.Auth:
		return r.Auth, 1, nil
	case orders:
		return r.Order, 1, nil
	case lowVolume:
		return r.LowVolume, 1, nil
	default:
		return r.UnAuth, 1, nil
	}
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
