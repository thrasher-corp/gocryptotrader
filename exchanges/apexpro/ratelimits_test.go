package apexpro

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"publicEPL":      publicEPL,
		"privateGetEPL":  privateGetEPL,
		"privatePostEPL": privatePostEPL,
		"createOrderEPL": createOrderEPL,
	}
	rl, err := request.New("rateLimitTest2", http.DefaultClient, request.WithLimiter(rateLimits))
	require.NoError(t, err)
	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := rl.InitiateRateLimit(t.Context(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
