package gemini

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// gemini limit rates
	geminiRateInterval = time.Minute
	geminiAuthRate     = 600
	geminiUnauthRate   = 120
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithToken(geminiRateInterval, geminiAuthRate, 1),
		request.UnAuth: request.NewRateLimitWithToken(geminiRateInterval, geminiUnauthRate, 1),
	}
}
