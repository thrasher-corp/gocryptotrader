package bitflyer

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange specific rate limit consts
const (
	biflyerRateInterval                 = time.Minute * 5
	bitflyerPrivateRequestRate          = 500
	bitflyerPrivateLowVolumeRequestRate = 100
	bitflyerPrivateSendOrderRequestRate = 300
	bitflyerPublicRequestRate           = 500
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithToken(biflyerRateInterval, bitflyerPrivateRequestRate, 1),
		request.UnAuth: request.NewRateLimitWithToken(biflyerRateInterval, bitflyerPublicRequestRate, 1),
		// TODO: Below limits need to also take from auth rate limit. This
		// can not yet be tested and verified so is left not done for now.
		orders:    request.NewRateLimitWithToken(biflyerRateInterval, bitflyerPrivateSendOrderRequestRate, 1),
		lowVolume: request.NewRateLimitWithToken(time.Minute, bitflyerPrivateLowVolumeRequestRate, 1),
	}
}
