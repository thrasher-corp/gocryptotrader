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

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithWeight(geminiRateInterval, geminiAuthRate, 1),
		request.UnAuth: request.NewRateLimitWithWeight(geminiRateInterval, geminiUnauthRate, 1),
	}
}
