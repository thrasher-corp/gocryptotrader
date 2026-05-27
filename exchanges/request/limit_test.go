package request

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		err := (*RateLimiterWithWeight)(nil).RateLimit(t.Context())
		assert.ErrorContains(t, err, "nil pointer: *request.RateLimiterWithWeight")

		r := &RateLimiterWithWeight{limiter: rate.NewLimiter(rate.Limit(1), 1)}
		err = r.RateLimit(t.Context())
		assert.ErrorIs(t, err, errInvalidWeight, "should return errInvalidWeightCount for zero weight")

		r = NewRateLimitWithWeight(time.Second, 10, 1)
		start := time.Now()
		err = r.RateLimit(t.Context())
		elapsed := time.Since(start)
		require.NoError(t, err, "rate limit must not error")
		assert.Zero(t, elapsed, "first call should be immediate")

		r = NewRateLimitWithWeight(time.Second, 10, 5)
		start = time.Now()
		err = r.RateLimit(t.Context())
		elapsed = time.Since(start)
		require.NoError(t, err, "rate limit must not error")
		assert.Equal(t, 400*time.Millisecond, elapsed, "should wait 400ms (4 intervals) for weight 5")

		r = NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		start = time.Now()
		err = r.RateLimit(WithDelayNotAllowed(t.Context()))
		synctest.Wait()
		elapsed = time.Since(start)
		require.NoError(t, err, "first rate limit call must not error and must be immediate")
		assert.Zero(t, elapsed, "first call should be immediate")

		start = time.Now()
		err = r.RateLimit(t.Context())
		elapsed = time.Since(start)
		require.NoError(t, err, "second rate limit call must not error")
		assert.Equal(t, 100*time.Millisecond, elapsed, "second call should be delayed by exactly 100ms")

		err = r.RateLimit(WithDelayNotAllowed(t.Context()))
		assert.ErrorIs(t, err, ErrDelayNotAllowed, "should return correct error")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err = r.RateLimit(ctx)
		assert.ErrorIs(t, err, context.Canceled, "should return correct error when context is cancelled")

		// Rate limit is 100ms. Set deadline for 50ms.
		ctx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		err = r.RateLimit(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "should return correct error when context deadline exceeded")

		r = NewRateLimitWithWeight(time.Second, 10, 1)
		start = time.Now()
		err = r.RateLimit(WithRateLimitWeight(t.Context(), 5))
		elapsed = time.Since(start)
		require.NoError(t, err, "rate limit with override weight must not error")
		assert.Equal(t, 400*time.Millisecond, elapsed, "override weight should apply delay based on 5 requests")
	})
}

func TestRateLimitAll(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		require.NoError(t, RateLimitAll(t.Context(), nil, nil), "empty rate limit set must not error")

		short := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		long := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
		require.NoError(t, RateLimitAll(t.Context(), []*RateLimiterWithWeight{
			short,
			long,
		}, nil), "first reservation must not error")

		start := time.Now()
		err := RateLimitAll(t.Context(), []*RateLimiterWithWeight{
			short,
			long,
		}, nil)
		elapsed := time.Since(start)
		require.NoError(t, err, "parallel rate limit must not error")
		assert.Equal(t, 300*time.Millisecond, elapsed, "parallel rate limit should wait for the longest limiter only")

		err = RateLimitAll(t.Context(), []*RateLimiterWithWeight{short}, []Weight{0})
		assert.ErrorIs(t, err, errInvalidWeight, "zero weight should return errInvalidWeight")

		err = RateLimitAll(t.Context(), []*RateLimiterWithWeight{nil}, nil)
		assert.ErrorContains(t, err, "nil pointer: *request.RateLimiterWithWeight")

		err = RateLimitAll(t.Context(), []*RateLimiterWithWeight{short}, []Weight{1, 1})
		assert.ErrorContains(t, err, "rate limiter count 1 does not match weight count 2")
	})
}

func TestRateLimitAll_WithDelayNotAllowed(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		short := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		long := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
		require.NoError(t, RateLimitAll(t.Context(), []*RateLimiterWithWeight{
			short,
			long,
		}, nil), "first reservation must not error")

		err := RateLimitAll(WithDelayNotAllowed(t.Context()), []*RateLimiterWithWeight{
			short,
			long,
		}, nil)
		require.ErrorIs(t, err, ErrDelayNotAllowed, "delayed reservation must return ErrDelayNotAllowed")

		start := time.Now()
		err = RateLimitAll(t.Context(), []*RateLimiterWithWeight{
			short,
			long,
		}, nil)
		elapsed := time.Since(start)
		require.NoError(t, err, "cancelled reservations must be usable again")
		assert.Equal(t, 300*time.Millisecond, elapsed, "cancelled reservation should not add another delay window")
	})
}

func TestRateLimitWithParallel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		extra := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
		require.NoError(t, RateLimitWithParallel(t.Context(), endpoint, 0, []*RateLimiterWithWeight{extra}, []Weight{1}), "first reservation must not error")

		start := time.Now()
		err := RateLimitWithParallel(t.Context(), endpoint, 0, []*RateLimiterWithWeight{extra}, []Weight{1})
		elapsed := time.Since(start)
		require.NoError(t, err, "parallel rate limit must not error")
		assert.Equal(t, 300*time.Millisecond, elapsed, "parallel rate limit should wait for the longest limiter only")

		err = RateLimitWithParallel(t.Context(), endpoint, 0, []*RateLimiterWithWeight{extra}, nil)
		assert.ErrorContains(t, err, "rate limiter count 1 does not match weight count 0")

		err = RateLimitWithParallel(t.Context(), endpoint, 0, []*RateLimiterWithWeight{extra}, []Weight{0})
		assert.ErrorIs(t, err, errInvalidWeight, "zero parallel weight should return errInvalidWeight")

		weightedEndpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		weightedExtra := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
		start = time.Now()
		err = RateLimitWithParallel(t.Context(), weightedEndpoint, 3, []*RateLimiterWithWeight{weightedExtra}, []Weight{1})
		elapsed = time.Since(start)
		require.NoError(t, err, "explicit endpoint weight must not error")
		assert.Equal(t, 200*time.Millisecond, elapsed, "explicit endpoint weight should override endpoint default weight")
	})
}

func TestInitiateRateLimitWithParallel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		r, err := New("test", new(http.Client), WithLimiter(NewBasicRateLimit(100*time.Millisecond, 1, 1)))
		require.NoError(t, err, "requester must initialise")
		extra := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
		require.NoError(t, r.InitiateRateLimitWithParallel(t.Context(), Unset, 0, []*RateLimiterWithWeight{extra}, []Weight{1}), "first reservation must not error")

		start := time.Now()
		err = r.InitiateRateLimitWithParallel(t.Context(), Unset, 0, []*RateLimiterWithWeight{extra}, []Weight{1})
		elapsed := time.Since(start)
		require.NoError(t, err, "parallel rate limit must not error")
		assert.Equal(t, 300*time.Millisecond, elapsed, "parallel rate limit should wait for the longest limiter only")

		err = (*Requester)(nil).InitiateRateLimitWithParallel(t.Context(), Unset, 0, nil, nil)
		assert.ErrorIs(t, err, ErrRequestSystemIsNil, "nil requester should return ErrRequestSystemIsNil")
	})
}

func TestRateLimit_Concurrent_WithFailure(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		r := NewRateLimitWithWeight(time.Second, 10, 1)
		tn := time.Now()
		errs := common.ErrorCollector{}
		for i := range 10 {
			ctx := t.Context()
			if i%2 == 0 {
				ctx = WithDelayNotAllowed(ctx)
			}
			errs.Go(func() error { return r.RateLimit(ctx) })
		}

		require.ErrorContains(t, errs.Collect(), "delay not allowed, delay not allowed, delay not allowed, delay not allowed", "must return correct error")
		assert.Less(t, time.Since(tn), time.Millisecond*600, "should complete within reasonable time")
	})
}

func TestRateLimit_Concurrent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		r := NewRateLimitWithWeight(time.Second, 10, 1)
		tn := time.Now()
		errs := common.ErrorCollector{}
		for range 10 {
			errs.Go(func() error { return r.RateLimit(t.Context()) })
		}
		require.NoError(t, errs.Collect(), "rate limit must not error")
		assert.Less(t, time.Since(tn), time.Second, "should complete within reasonable time")
	})
}

func TestRateLimit_Linear_WithFailure(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		r := NewRateLimitWithWeight(time.Second, 10, 1)
		tn := time.Now()
		for i := range 10 {
			ctx := t.Context()
			if i%2 == 0 {
				ctx = WithDelayNotAllowed(ctx)
			}
			if err := r.RateLimit(ctx); err != nil {
				require.ErrorIs(t, err, ErrDelayNotAllowed, "must return correct error")
			}
		}
		assert.Less(t, time.Since(tn), time.Millisecond*600, "should complete within reasonable time")
	})
}

func TestRateLimit_Linear(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		r := NewRateLimitWithWeight(time.Second, 10, 1)
		tn := time.Now()
		for range 10 {
			require.NoError(t, r.RateLimit(t.Context()))
		}
		assert.Less(t, time.Since(tn), time.Second, "should complete within reasonable time")
	})
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
	require.Len(t, defs, 3, "must have 3 definitions")

	for _, key := range []EndpointLimit{Unset, Auth, UnAuth} {
		r, ok := defs[key]
		require.Truef(t, ok, "must have definition for %v", key)
		assert.Equalf(t, Weight(5), r.weight, "weight should be 5 for %v", key)
		assert.Equalf(t, rate.Limit(10), r.limiter.Limit(), "limit should be 10 per second for %v", key)
	}

	assert.Same(t, defs[Unset], defs[Auth], "Unset and Auth should be same instance")
	assert.Same(t, defs[Auth], defs[UnAuth], "Auth and UnAuth should be same instance")
}

func TestCancelAll(t *testing.T) {
	t.Parallel()

	reservations := make([]*rate.Reservation, 0, 2)
	cancelAll(reservations, time.Now())

	r := rate.NewLimiter(rate.Limit(1), 1)
	tn := time.Now()
	reservations = append(reservations, r.ReserveN(tn, 1))
	require.Equal(t, 0.0, r.TokensAt(tn), "must have zero tokens remaining")
	reservations = append(reservations, r.ReserveN(tn, 1))
	require.Equal(t, time.Second, reservations[1].DelayFrom(tn), "second reservation must have 1 second delay")
	require.Equal(t, -1.0, r.TokensAt(tn), "must have negative tokens remaining")
	cancelAll(reservations, tn)
	require.Equal(t, 1.0, r.TokensAt(tn), "must have 1 token remaining after cancellation")
}

func TestInitiateRateLimit(t *testing.T) {
	t.Parallel()

	var r *Requester
	err := r.InitiateRateLimit(t.Context(), Unset)
	assert.ErrorIs(t, err, ErrRequestSystemIsNil, "should return correct error")

	r = &Requester{}
	atomic.StoreInt32(&r.disableRateLimiter, 1)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.NoError(t, err, "should not error when rate limiter is disabled")

	atomic.StoreInt32(&r.disableRateLimiter, 0)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.ErrorContains(t, err, "nil pointer: request.RateLimitDefinitions", "should return correct error when limiter is nil")

	r.limiter = NewBasicRateLimit(time.Second, 10, 1)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.NoError(t, err, "should not error on valid rate limit initiation")
}
