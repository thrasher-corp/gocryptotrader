package request

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	err := (*RateLimiterWithWeight)(nil).RateLimit(t.Context())
	assert.ErrorIs(t, err, common.ErrNilPointer)

	r := &RateLimiterWithWeight{limiter: rate.NewLimiter(rate.Limit(1), 1)}
	err = r.RateLimit(t.Context())
	assert.ErrorIs(t, err, errInvalidWeightCount, "should return errInvalidWeightCount for zero weight")

	r = NewRateLimitWithWeight(time.Second, 10, 1)
	start := time.Now()
	err = r.RateLimit(t.Context())
	elapsed := time.Since(start)
	require.NoError(t, err, "rate limit must not error")
	assert.Less(t, elapsed, time.Millisecond*50, "should complete quickly for first request")

	r = NewRateLimitWithWeight(time.Second, 10, 5)
	start = time.Now()
	err = r.RateLimit(t.Context())
	elapsed = time.Since(start)
	require.NoError(t, err, "rate limit must not error")
	assert.Less(t, elapsed, time.Millisecond*600, "should complete within reasonable time for weight 5")

	r = NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
	start = time.Now()
	err = r.RateLimit(WithNoDelayPermitted(t.Context()))
	require.NoError(t, err, "first rate limit call must not error and must be immediate")
	firstElapsed := time.Since(start)
	assert.Less(t, firstElapsed, 50*time.Millisecond, "first call should be immediate")

	start = time.Now()
	err = r.RateLimit(t.Context())
	require.NoError(t, err, "second rate limit call must not error")
	secondElapsed := time.Since(start)
	assert.GreaterOrEqual(t, secondElapsed, 90*time.Millisecond, "second call should be delayed by approximately 100ms")
	assert.Less(t, secondElapsed, 150*time.Millisecond, "delay should not be excessive")

	err = r.RateLimit(WithNoDelayPermitted(t.Context()))
	assert.ErrorIs(t, err, ErrNoDelayPermitted, "should return correct error")

	var routineErr error
	wg := sync.WaitGroup{}
	wg.Go(func() { routineErr = r.RateLimit(t.Context()) })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	time.Sleep(10 * time.Millisecond)
	err = r.RateLimit(ctx)
	assert.ErrorIs(t, err, context.Canceled, "should return correct error")
	wg.Wait()
	assert.NoError(t, routineErr, "routine should complete successfully will providing friction for above context cancellation")

	wg.Go(func() { routineErr = r.RateLimit(t.Context()) })
	ctx, cancel = context.WithDeadline(t.Context(), time.Now())
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	err = r.RateLimit(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "should return correct error")
	wg.Wait()
	assert.NoError(t, routineErr, "routine should complete successfully will providing friction for above context deadline exceeded")
}

func TestNewRateLimit(t *testing.T) {
	t.Parallel()

	r := NewRateLimit(time.Second, 10)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, rate.Limit(10), r.Limit(), "limit should be 10 per second")

	r = NewRateLimit(time.Second, 0)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, rate.Inf, r.Limit(), "limit should be infinite on zero actions")

	r = NewRateLimit(time.Second, -1)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, rate.Inf, r.Limit(), "limit should be infinite on negative actions")

	r = NewRateLimit(0, 10)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, rate.Inf, r.Limit(), "limit should be infinite on zero interval")

	r = NewRateLimit(-time.Second, 10)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, rate.Inf, r.Limit(), "limit should be infinite on negative interval")
}

func TestNewRateLimitWithWeight(t *testing.T) {
	t.Parallel()

	r := NewRateLimitWithWeight(time.Second, 10, 5)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, Weight(5), r.weight, "weight should be 5")
	assert.Equal(t, rate.Limit(10), r.limiter.Limit(), "limit should be 10 per second")
}

func TestNewWeightedRateLimitByDuration(t *testing.T) {
	t.Parallel()

	r := NewWeightedRateLimitByDuration(time.Second)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Equal(t, Weight(1), r.weight, "weight should be 1")
	assert.Equal(t, rate.Limit(1), r.limiter.Limit(), "limit should be 1 per second")
}

func TestGetRateLimiterWithWeight(t *testing.T) {
	t.Parallel()

	r := rate.NewLimiter(rate.Limit(10), 1)
	weighted := GetRateLimiterWithWeight(r, 5)
	require.NotNil(t, weighted, "weighted limiter must not be nil")
	assert.Equal(t, Weight(5), weighted.weight, "weight should be 5")
	assert.Equal(t, r, weighted.limiter, "should reference same limiter")
}

func TestNewBasicRateLimit(t *testing.T) {
	t.Parallel()

	defs := NewBasicRateLimit(time.Second, 10, 5)
	require.NotNil(t, defs, "definitions must not be nil")
	require.Len(t, defs, 3, "should have 3 definitions")

	for _, key := range []EndpointLimit{Unset, Auth, UnAuth} {
		r, ok := defs[key]
		require.Truef(t, ok, "should have definition for %v", key)
		assert.Equalf(t, Weight(5), r.weight, "weight should be 5 for %v", key)
		assert.Equalf(t, rate.Limit(10), r.limiter.Limit(), "limit should be 10 per second for %v", key)
	}

	assert.Same(t, defs[Unset], defs[Auth], "Unset and Auth should be same instance")
	assert.Same(t, defs[Auth], defs[UnAuth], "Auth and UnAuth should be same instance")
}

func TestWithNoDelayPermitted(t *testing.T) {
	t.Parallel()

	assert.True(t, hasNoDelayPermitted(WithNoDelayPermitted(t.Context())))
	assert.False(t, hasNoDelayPermitted(t.Context()))
	assert.False(t, hasNoDelayPermitted(WithVerbose(t.Context())))
}
