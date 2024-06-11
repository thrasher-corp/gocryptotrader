package poloniex

import (
	"context"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"auth Non Resource Intensive": authNonResourceIntensiveEPL,
		"auth Resource Intensive":     authResourceIntensiveEPL,
		"unauth":                      unauthEPL,
		"reference Data":              referenceDataEPL,
	}
	for name, tt := range testTable {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := SetRateLimit()
			if err := l.Limit(context.Background(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
