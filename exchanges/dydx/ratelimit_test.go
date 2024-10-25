package dydx

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"Default V3":              defaultV3EPL,
		"Send Verification Email": sendVerificationEmailEPL,
		"Cancel Orders":           cancelOrdersEPL,
		"Cancel Single Order":     cancelSingleOrderEPL,
		"Post Orders":             postOrdersEPL,
		"Post Testnet Tokens":     postTestnetTokensEPL,
		"Cancel Active Orders":    cancelActiveOrdersEPL,
		"Get Active Orders":       getActiveOrdersEPL,
	}
	rl, err := request.New("rateLimitTest2", http.DefaultClient, request.WithLimiter(GetRateLimit()))
	require.NoError(t, err)
	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := rl.InitiateRateLimit(context.Background(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
