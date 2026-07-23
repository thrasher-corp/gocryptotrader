package kraken

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestNewKrakenSpotPrivateLimiter(t *testing.T) {
	t.Parallel()

	limiter := newKrakenSpotPrivateLimiter()
	require.NotNil(t, limiter, "limiter must not be nil")
	assert.Equal(t, KrakenSpotDecayPerSec, float64(limiter.Limit()), "limit should match Kraken private decay")
	assert.Equal(t, KrakenSpotMaxCounter, limiter.Burst(), "burst should match Kraken private counter")
}

func TestNewKrakenSpotOrderLimiter(t *testing.T) {
	t.Parallel()

	limiter := newKrakenSpotOrderLimiter()
	require.NotNil(t, limiter, "limiter must not be nil")
	assert.Equal(t, KrakenSpotOrderRate, float64(limiter.Limit()), "limit should match Kraken order rate")
	assert.Equal(t, KrakenSpotOrderMaxBurst, limiter.Burst(), "burst should match Kraken order burst")
}

func TestNewKrakenPublicLimiter(t *testing.T) {
	t.Parallel()

	limiter := newKrakenPublicLimiter()
	require.NotNil(t, limiter, "limiter must not be nil")
	assert.Equal(t, KrakenPublicRate, float64(limiter.Limit()), "limit should match Kraken public rate")
	assert.Equal(t, KrakenPublicMaxBurst, limiter.Burst(), "burst should match Kraken public burst")
}

func TestBuildKrakenRateLimits(t *testing.T) {
	t.Parallel()

	rateLimits := buildKrakenRateLimits()
	require.NotEmpty(t, rateLimits, "rate limits must not be empty")

	expectedLimits := []request.EndpointLimit{
		request.Unset,
		request.Auth,
		request.UnAuth,
		krakenLimitDefault,
		krakenLimitPublic,
		krakenLimitFuturesPublic,
		krakenLimitBalance,
		krakenLimitHistory,
		krakenLimitTrading,
		krakenLimitWithdraw,
	}
	for _, limit := range expectedLimits {
		assert.NotNilf(t, rateLimits[limit], "rate limit should exist for endpoint %d", limit)
	}

	requester, err := request.New("krakenRateLimits", http.DefaultClient, request.WithLimiter(rateLimits))
	require.NoError(t, err, "request.New must not error")

	for range KrakenSpotMaxCounter / 4 {
		require.NoError(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitHistory), "history limit must allow burst usage")
	}
	assert.ErrorIs(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitHistory), request.ErrDelayNotAllowed, "history limit should consume private weight")
	assert.ErrorIs(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitBalance), request.ErrDelayNotAllowed, "balance limit should share the private limiter")
	assert.NoError(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitTrading), "trading limit should use a separate limiter")
	assert.NoError(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitPublic), "public limit should use a separate limiter")
	assert.NoError(t, requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), krakenLimitFuturesPublic), "futures public limit should not consume a rate budget")

	err = requester.InitiateRateLimit(request.WithDelayNotAllowed(t.Context()), request.UnAuth)
	if errors.Is(err, request.ErrDelayNotAllowed) {
		t.Fatal("unauthenticated limit must not share the exhausted private limiter")
	}
	require.NoError(t, err, "unauthenticated limit must use the public limiter")
}
