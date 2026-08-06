package kraken

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestBuildKrakenRateLimits(t *testing.T) {
	t.Parallel()

	rateLimits := buildKrakenRateLimits()
	for _, limit := range []request.EndpointLimit{
		krakenLimitPublic,
		krakenLimitFuturesPublic,
		krakenLimitFuturesAuth,
		krakenLimitAuth,
		krakenLimitHistory,
		krakenLimitTrading,
		krakenLimitCancel,
	} {
		assert.NotNilf(t, rateLimits[limit], "rate limit should exist for endpoint %d", limit)
	}

	requester, err := request.New("krakenRateLimits", http.DefaultClient, request.WithLimiter(rateLimits))
	require.NoError(t, err, "request.New must not error")

	ctx := request.WithDelayNotAllowed(t.Context())

	// History requests cost 2 on the private counter, so half the counter
	// maximum fills it exactly
	for range krakenSpotMaxCounter / 2 {
		require.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitHistory), "history requests must be allowed up to the counter maximum")
	}
	assert.ErrorIs(t, requester.InitiateRateLimit(ctx, krakenLimitHistory), request.ErrDelayNotAllowed, "history should be limited once the private counter is exhausted")
	assert.ErrorIs(t, requester.InitiateRateLimit(ctx, krakenLimitAuth), request.ErrDelayNotAllowed, "auth should share the exhausted private counter")

	assert.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitTrading), "trading should use a separate counter")
	assert.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitPublic), "public should use a separate limiter")
	assert.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitFuturesPublic), "futures public should not consume any budget")
	assert.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitFuturesAuth), "futures auth should use a separate limiter")

	// One trading unit is already consumed above; 7 worst-case cancels
	// consume another 56 of the 60 burst, and the next cannot fit
	for range 7 {
		require.NoError(t, requester.InitiateRateLimit(ctx, krakenLimitCancel), "cancels must be allowed within the trading counter")
	}
	assert.ErrorIs(t, requester.InitiateRateLimit(ctx, krakenLimitCancel), request.ErrDelayNotAllowed, "cancel weight should throttle rapid cancellations")
}
