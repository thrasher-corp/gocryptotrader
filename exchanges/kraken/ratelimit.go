package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Endpoint groups for rate limiting; see buildKrakenRateLimits for the
// limits applied to each.
const (
	krakenLimitPublic request.EndpointLimit = iota
	krakenLimitFuturesPublic
	krakenLimitFuturesAuth
	krakenLimitAuth
	krakenLimitHistory
	krakenLimitTrading
	krakenLimitCancel
)

const (
	// Spot private counter, Verified tier: max 20, decays 0.5/sec.
	krakenSpotMaxCounter  = 20
	krakenSpotDecayPerSec = 0.5

	// Spot trading engine counter, Starter tier: max 60, decays 1/sec.
	// This counter covers all order management: placing, editing,
	// amending and cancelling orders. Higher verification tiers allow
	// more (Intermediate 125 @ 2.34/sec, Pro 180 @ 3.75/sec).
	krakenSpotOrderMaxBurst = 60
	krakenSpotOrderRate     = 1.0

	// Cancelling an order shortly after placement costs up to 8 counter
	// units depending on its age; cancels are weighted at the worst case
	// so the local counter cannot run ahead of Kraken's.
	krakenCancelWeight = 8

	// Kraken does not state a precise public burst limit; 15 @ 1/sec has
	// been tested and found safe.
	krakenPublicMaxBurst = 15
	krakenPublicRate     = 1.0
)

// buildKrakenRateLimits returns limiters matching Kraken's documented
// rate-limit behaviour: a spot private counter per API key with account
// history endpoints costing double, a separate trading engine counter with
// age-weighted cancels, a per-IP public limit, a futures cost pool of 500
// units per 10 seconds (order management endpoints cost up to 10, so 50
// requests per 10 seconds is a safe floor), and uncosted public futures
// endpoints.
//
// Reference: https://support.kraken.com/articles/206548367
func buildKrakenRateLimits() request.RateLimitDefinitions {
	private := rate.NewLimiter(rate.Limit(krakenSpotDecayPerSec), krakenSpotMaxCounter)
	trading := rate.NewLimiter(rate.Limit(krakenSpotOrderRate), krakenSpotOrderMaxBurst)
	public := rate.NewLimiter(rate.Limit(krakenPublicRate), krakenPublicMaxBurst)

	return request.RateLimitDefinitions{
		krakenLimitPublic:        request.GetRateLimiterWithWeight(public, 1),
		krakenLimitFuturesPublic: request.NewRateLimitWithWeight(0, 0, 1),
		krakenLimitFuturesAuth:   request.NewRateLimitWithWeight(10*time.Second, 50, 1),
		krakenLimitAuth:          request.GetRateLimiterWithWeight(private, 1),
		krakenLimitHistory:       request.GetRateLimiterWithWeight(private, 2),
		krakenLimitTrading:       request.GetRateLimiterWithWeight(trading, 1),
		krakenLimitCancel:        request.GetRateLimiterWithWeight(trading, krakenCancelWeight),
	}
}
