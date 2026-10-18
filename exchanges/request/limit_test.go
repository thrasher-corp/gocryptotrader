package request

import (
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
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

		r = NewRateLimitWithWeight(time.Second, 10, 1)
		start = time.Now()
		err = r.RateLimit(WithWeight(t.Context(), 5))
		elapsed = time.Since(start)
		require.NoError(t, err, "rate limit must not error")
		assert.Equal(t, 400*time.Millisecond, elapsed, "should wait 400ms (4 intervals) for a context weight of 5 in place of the definition's 1")

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

		never := GetRateLimiterWithWeight(rate.NewLimiter(1, 0), 1)
		assert.ErrorIs(t, never.RateLimit(WithDelayNotAllowed(t.Context())), ErrDelayNotAllowed, "a limiter that never grants should refuse a request that cannot wait")
		ctx, cancel = context.WithTimeout(t.Context(), time.Hour)
		defer cancel()
		assert.ErrorIs(t, never.RateLimit(ctx), context.DeadlineExceeded, "a limiter that never grants should refuse a request with any deadline")
		ctx, cancel = context.WithCancel(t.Context())
		time.AfterFunc(time.Hour, cancel)
		assert.ErrorIs(t, never.RateLimit(ctx), context.Canceled, "a limiter that never grants should hold a request until it is cancelled")
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

func TestRateLimitBarrier(t *testing.T) {
	t.Parallel()

	t.Run("both immediate", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err)
		left := NewRateLimitWithWeight(time.Second, 1, 1)
		right := NewRateLimitWithWeight(time.Second, 1, 1)
		errs := make(chan error, 2)
		go func() { errs <- left.RateLimit(contexts[0]) }()
		go func() { errs <- right.RateLimit(contexts[1]) }()
		require.NoError(t, <-errs)
		require.NoError(t, <-errs)
	})

	t.Run("acceptance retains final reservations", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(WithDelayNotAllowed(t.Context()), 2)
		require.NoError(t, err)
		left := NewRateLimitWithWeight(time.Hour, 1, 1)
		right := NewRateLimitWithWeight(time.Hour, 1, 1)
		errs := make(chan error, 2)
		go func() { errs <- left.RateLimit(contexts[0]) }()
		go func() { errs <- right.RateLimit(contexts[1]) }()
		require.NoError(t, <-errs)
		require.NoError(t, <-errs)
		require.ErrorIs(t, left.RateLimit(WithDelayNotAllowed(t.Context())), ErrDelayNotAllowed,
			"accepted participant must retain its reservation")
		require.ErrorIs(t, right.RateLimit(WithDelayNotAllowed(t.Context())), ErrDelayNotAllowed,
			"accepted participant must retain its reservation")
	})

	t.Run("one delayed rejects both and restores reservations", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err)
		left := NewRateLimitWithWeight(time.Hour, 1, 1)
		right := NewRateLimitWithWeight(time.Hour, 1, 1)
		require.NoError(t, right.RateLimit(t.Context()))
		errs := make(chan error, 2)
		go func() { errs <- left.RateLimit(contexts[0]) }()
		go func() { errs <- right.RateLimit(contexts[1]) }()
		require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
		require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
		require.NoError(t, left.RateLimit(WithDelayNotAllowed(t.Context())))
	})

	t.Run("participant reuse", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err)
		left := NewRateLimitWithWeight(time.Second, 1, 1)
		right := NewRateLimitWithWeight(time.Second, 1, 1)
		errs := make(chan error, 2)
		go func() { errs <- left.RateLimit(contexts[0]) }()
		go func() { errs <- right.RateLimit(contexts[1]) }()
		require.NoError(t, <-errs)
		require.NoError(t, <-errs)
		require.NoError(t, left.RateLimit(contexts[0]), "post-acceptance calls must use ordinary rate limiting")
	})

	t.Run("cancellation rejects peer without consuming capacity", func(t *testing.T) {
		t.Parallel()
		parent, cancel := context.WithCancel(t.Context())
		contexts, err := NewRateLimitBarrierContexts(parent, 2)
		require.NoError(t, err)
		left := NewRateLimitWithWeight(time.Hour, 1, 1)
		errCh := make(chan error, 1)
		go func() { errCh <- left.RateLimit(contexts[0]) }()
		cancel()
		require.ErrorIs(t, <-errCh, context.Canceled)
		require.ErrorIs(t, WaitForRateLimitBarrier(contexts[1]), ErrRateLimitBarrierRejected)
		require.NoError(t, left.RateLimit(WithDelayNotAllowed(t.Context())))
	})

	t.Run("shared limiter without group burst rejects all participants", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
			require.NoError(t, err)
			limiter := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			errs := make(chan error, 2)
			go func() { errs <- limiter.RateLimit(contexts[0]) }()
			synctest.Wait()
			go func() { errs <- limiter.RateLimit(contexts[1]) }()

			require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
			require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
			assert.InDelta(t, 1, limiter.limiter.Tokens(), 0.001,
				"rejected group should restore shared limiter capacity")
		})
	})

	t.Run("contention before final arrival rejects group atomically", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
			require.NoError(t, err)
			left := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			right := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			errs := make(chan error, 2)
			go func() { errs <- left.RateLimit(contexts[0]) }()
			synctest.Wait()
			require.NoError(t, left.RateLimit(t.Context()), "competitor must consume left capacity")
			go func() { errs <- right.RateLimit(contexts[1]) }()

			require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
			require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
			assert.InDelta(t, 1, right.limiter.Tokens(), 0.001,
				"atomic rejection should restore the other participant's capacity")
		})
	})

	t.Run("weight exceeding burst rejects group", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err)
		weighted := NewRateLimitWithWeight(time.Second, 1, 2)
		peer := NewRateLimitWithWeight(time.Second, 1, 1)
		errs := make(chan error, 2)
		go func() { errs <- weighted.RateLimit(contexts[0]) }()
		go func() { errs <- peer.RateLimit(contexts[1]) }()
		require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
		require.ErrorIs(t, <-errs, ErrDelayNotAllowed)
	})

	t.Run("parked participant holds no capacity", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
			require.NoError(t, err)
			limiter := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			errCh := make(chan error, 1)
			go func() { errCh <- limiter.RateLimit(contexts[0]) }()
			synctest.Wait()

			require.NoError(t, limiter.RateLimit(WithDelayNotAllowed(t.Context())),
				"competitor must receive capacity unused by the parked participant")
			AbortRateLimitBarrier(contexts[1])
			require.ErrorIs(t, <-errCh, ErrRateLimitBarrierRejected)
			assert.InDelta(t, 0, limiter.limiter.Tokens(), 0.001,
				"rejected barrier should not consume capacity")
		})
	})
}

func TestRateLimitBarrierInvalidWeightReleasesParkedPeer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		invalid *RateLimiterWithWeight
		weight  func(context.Context) context.Context
		err     error
	}{
		{"zero weight", NewRateLimitWithWeight(time.Hour, 1, 0), func(ctx context.Context) context.Context { return ctx }, errInvalidWeight},
		{"a weight its window budget never admits", GetRateLimiterWithWindowBudget(NewWindowBudget(time.Hour, 2), 1), func(ctx context.Context) context.Context { return WithWeight(ctx, 3) }, ErrWeightExceedsBudget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				contexts, err := NewRateLimitBarrierContexts(t.Context(), 3)
				require.NoError(t, err, "barrier contexts must be created")
				defer AbortRateLimitBarrier(contexts[2])
				peer := NewRateLimitWithWeight(time.Hour, 1, 1)
				errors := make(chan error, 1)
				go func() { errors <- peer.RateLimit(contexts[0]) }()
				synctest.Wait()
				require.ErrorIs(t, tc.invalid.RateLimit(tc.weight(contexts[1])), tc.err, "an invalid weight must be rejected")
				synctest.Wait()
				select {
				case err := <-errors:
					assert.ErrorIs(t, err, ErrRateLimitBarrierRejected, "parked peer should observe rejection before the last arrival")
					assert.NotErrorIs(t, err, ErrDelayNotAllowed, "invalid peer should not report a capacity delay")
				default:
					t.Error("an invalid weight should release the parked peer without a final arrival")
				}
				assert.Equal(t, float64(1), peer.limiter.Tokens(), "rejected group should leave capacity untouched")
			})
		})
	}
}

func TestRateLimitBarrierCancellationWithoutFinalArrival(t *testing.T) {
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "barrier contexts must be created")
		defer AbortRateLimitBarrier(contexts[1])
		parked, cancel := context.WithCancel(contexts[0])
		defer cancel()
		errors := make(chan error, 1)
		go func() { errors <- WaitForRateLimitBarrier(parked) }()
		synctest.Wait()
		cancel()
		synctest.Wait()
		select {
		case err := <-errors:
			assert.ErrorIs(t, err, context.Canceled, "parked participant should observe cancellation without a final arrival")
		default:
			t.Error("cancelled participant should return without a final arrival")
		}
		assert.ErrorIs(t, WaitForRateLimitBarrier(contexts[1]), ErrRateLimitBarrierRejected, "late peer should observe group rejection")
	})
}

func TestRateLimitReservationLock(t *testing.T) {
	for _, coordinated := range []bool{false, true} {
		name := "ordinary reservation"
		if coordinated {
			name = "group admission"
		}
		t.Run(name, func(t *testing.T) {
			limiter := NewRateLimitWithWeight(time.Hour, 1, 1)
			limiter.m.Lock()
			ctx := t.Context()
			if coordinated {
				contexts, err := NewRateLimitBarrierContexts(ctx, 2)
				if !assert.NoError(t, err, "barrier contexts should be created") {
					limiter.m.Unlock()
					return
				}
				ctx = contexts[0]
				peerErrors := make(chan error, 1)
				go func() { peerErrors <- WaitForRateLimitBarrier(contexts[1]) }()
				defer func() { assert.NoError(t, <-peerErrors, "unlimited peer should be admitted") }()
			}
			errors := make(chan error, 1)
			go func() { errors <- limiter.RateLimit(ctx) }()
			assert.Eventually(t, func() bool {
				if !rateLimitReservationMu.TryLock() {
					return true
				}
				rateLimitReservationMu.Unlock()
				return false
			}, time.Second, time.Millisecond, "reservation should hold the accounting lock while waiting for the limiter")
			limiter.m.Unlock()
			assert.NoError(t, <-errors, "reservation should complete after the limiter unlocks")
		})
	}
}

func TestRateLimitBarrierCancellationDuringAdmission(t *testing.T) {
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err, "barrier contexts must be created")
	first, cancel := context.WithCancel(contexts[0])
	defer cancel()
	left := NewRateLimitWithWeight(time.Hour, 1, 1)
	right := NewRateLimitWithWeight(time.Hour, 1, 1)
	right.m.Lock()
	leftErrors, rightErrors := make(chan error, 1), make(chan error, 1)
	go func() { leftErrors <- left.RateLimit(first) }()
	go func() { rightErrors <- right.RateLimit(contexts[1]) }()
	assert.Eventually(t, func() bool {
		return left.limiter.Tokens() < 0.5
	}, time.Second, time.Millisecond, "admission should reserve the first participant before blocking on the second")
	cancel()
	right.m.Unlock()
	assert.ErrorIs(t, <-leftErrors, context.Canceled, "first participant should observe cancellation during admission")
	assert.ErrorIs(t, <-rightErrors, ErrRateLimitBarrierRejected, "peer should observe post-reservation rejection")
	assert.InDelta(t, 1, left.limiter.Tokens(), 0.001, "cancelled admission should restore the first reservation")
	assert.InDelta(t, 1, right.limiter.Tokens(), 0.001, "cancelled admission should restore the second reservation")
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

// TestRateLimitKeepsTokenBucketTiming checks that a definition drawing on a rate.Limiter alone, as every exchange's but
// Bybit's does, still sends each request when reserving its weight one token at a time from a twin limiter, at the
// moment the request asks, would
func TestRateLimitKeepsTokenBucketTiming(t *testing.T) {
	t.Parallel()
	arrivals := []time.Duration{0, 0, 0, 50 * time.Millisecond, 50 * time.Millisecond, time.Second, 1010 * time.Millisecond, 3 * time.Second, 3 * time.Second, 3 * time.Second, 7 * time.Second}
	for _, tc := range []struct {
		limit  rate.Limit
		burst  int
		weight Weight
	}{
		{10, 1, 1}, {10, 1, 5}, {3, 1, 2}, {100, 1, 40}, {30, 1, 50}, {2, 5, 3}, {0.5, 1, 1}, {1000, 10, 7}, {20, 1, 1},
	} {
		t.Run(fmt.Sprintf("%v a second with a burst of %d at weight %d", tc.limit, tc.burst, tc.weight), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				r, twin := GetRateLimiterWithWeight(rate.NewLimiter(tc.limit, tc.burst), tc.weight), rate.NewLimiter(tc.limit, tc.burst)
				start := time.Now()
				exp, sends := make([]time.Duration, len(arrivals)), make([]time.Duration, len(arrivals))
				var wg sync.WaitGroup
				for i, arrival := range arrivals {
					time.Sleep(arrival - time.Since(start))
					now := time.Now()
					var last *rate.Reservation
					for range tc.weight {
						last = twin.ReserveN(now, 1)
					}
					exp[i] = now.Add(last.DelayFrom(now)).Sub(start)
					wg.Go(func() {
						assert.NoError(t, r.RateLimit(t.Context()), "a request should be sent")
						sends[i] = time.Since(start)
					})
					synctest.Wait()
				}
				wg.Wait()
				assert.Equal(t, exp, sends, "each request should be sent when a plain token bucket reservation falls due")
			})
		})
	}
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

func TestRateLimiterWithWeightLimit(t *testing.T) {
	t.Parallel()
	assert.Zero(t, RateLimitNotRequired.Limit(), "the no-op limiter should report no budget")
	for _, tc := range []struct {
		name    string
		limiter *RateLimiterWithWeight
		want    rate.Limit
	}{
		{"10 per second at weight 1", NewRateLimitWithWeight(time.Second, 10, 1), 10},
		{"10 per second at weight 5", NewRateLimitWithWeight(time.Second, 10, 5), 10},
		{"one per 3 seconds", NewRateLimitWithWeight(3*time.Second, 1, 5), rate.Every(3 * time.Second)},
		{"3 per 2 seconds", NewRateLimitWithWeight(2*time.Second, 3, 1), 1.5},
		{"3 set on the limiter", GetRateLimiterWithWeight(rate.NewLimiter(3, 1), 1), 3},
		{"1.1 set on the limiter", GetRateLimiterWithWeight(rate.NewLimiter(1.1, 1), 1), 1.1},
		{"0.9 set on the limiter", GetRateLimiterWithWeight(rate.NewLimiter(0.9, 1), 3), 0.9},
		{"0.1 set on the limiter", GetRateLimiterWithWeight(rate.NewLimiter(0.1, 1), 3), 0.1},
		{"unrestricted", NewRateLimitWithWeight(time.Second, 0, 1), rate.Inf},
		{"a shared budget left out", GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, NewWindowBudget(time.Second, 1)), 10},
		{"a window budget of 10 a second", GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1, NewWindowBudget(time.Second, 1)), 10},
		{"a window budget of 570 in 5 seconds", GetRateLimiterWithWindowBudget(NewWindowBudget(5*time.Second, 570), 1), 114},
		{"a window budget of 12 in 100ms", GetRateLimiterWithWindowBudget(NewWindowBudget(100*time.Millisecond, 12), 1), 120},
		{"no window budget", GetRateLimiterWithWindowBudget(nil, 1, NewWindowBudget(time.Second, 1)), rate.Inf},
	} {
		assert.Equalf(t, tc.want, tc.limiter.Limit(), "Limit should report the budget of %s exactly", tc.name)
	}
}

func TestRateLimiterWithWeightSharesBudgetWith(t *testing.T) {
	t.Parallel()
	shared, unrestricted := NewRateLimit(time.Second, 10), NewRateLimit(time.Second, 0)
	orders, ip := NewWindowBudget(time.Second, 10), NewWindowBudget(5*time.Second, 570)
	// Each group holds the endpoints drawing on one budget of their own: an endpoint shares a budget with every
	// endpoint of its own group, itself included, and with no other, whatever shared budgets they also draw on.
	groups := [][]*RateLimiterWithWeight{
		{GetRateLimiterWithWeight(shared, 1), GetRateLimiterWithWeight(shared, 1), GetRateLimiterWithWeight(shared, 5), GetRateLimiterWithWeight(shared, 5), GetRateLimiterWithShared(shared, 1, ip)},
		{GetRateLimiterWithWeight(unrestricted, 1), GetRateLimiterWithWeight(unrestricted, 5)},
		{GetRateLimiterWithWindowBudget(orders, 1, ip), GetRateLimiterWithWindowBudget(orders, 1), GetRateLimiterWithWindowBudget(orders, 5, ip)},
		{GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1, ip)},
		{GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip)},
		{GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip)},
		{GetRateLimiterWithWindowBudget(nil, 1, ip)},
		{GetRateLimiterWithWindowBudget(nil, 1, ip)},
		{NewRateLimitWithWeight(time.Second, 10, 1)},
		{NewRateLimitWithWeight(time.Second, 10, 5)},
		{NewRateLimitWithWeight(time.Second, 3, 5)},
		{NewRateLimitWithWeight(time.Second, 20, 1)},
		{NewRateLimitWithWeight(time.Second, 20, 5)},
		{NewRateLimitWithWeight(time.Second, 0, 1)},
	}
	for i, group := range groups {
		for x, endpoint := range group {
			for j, others := range groups {
				for y, other := range others {
					assert.Equalf(t, i == j, endpoint.SharesBudgetWith(other), "endpoint %d of group %d sharing with endpoint %d of group %d should be %t", x, i, y, j, i == j)
				}
			}
		}
	}
	assert.False(t, groups[0][0].SharesBudgetWith(RateLimitNotRequired), "a limiter should not share a budget with the no-op limiter")
	assert.False(t, RateLimitNotRequired.SharesBudgetWith(groups[0][0]), "the no-op limiter should not share a budget with a limiter")
	assert.True(t, RateLimitNotRequired.SharesBudgetWith(RateLimitNotRequired), "the no-op limiter should share its absent budget with itself")
}

func TestRateLimiterWithWeightReadsLeaveTheBudget(t *testing.T) {
	t.Parallel()
	l := rate.NewLimiter(rate.Every(time.Hour), 1)
	r := GetRateLimiterWithWeight(l, 5)
	r.Limit()
	r.SharesBudgetWith(r)
	assert.Equal(t, rate.Every(time.Hour), l.Limit(), "reading the budget should not change its rate")
	assert.Equal(t, 1, l.Burst(), "reading the budget should not change its burst")
	assert.True(t, l.Allow(), "reading the budget should not spend it")
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

func TestGetRateLimiterWithShared(t *testing.T) {
	t.Parallel()

	own := rate.NewLimiter(rate.Limit(10), 1)
	shared, other := NewWindowBudget(time.Second, 2), NewWindowBudget(time.Second, 5)
	budgets := []*WindowBudget{nil, shared, other, shared, nil}
	r := GetRateLimiterWithShared(own, 3, budgets...)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Same(t, own, r.limiter, "the endpoint's own limiter should be kept, so that definitions can share it")
	assert.Equal(t, Weight(3), r.weight, "weight should be 3")
	assert.Equal(t, []*WindowBudget{shared, other}, r.shared, "shared budgets should be kept once each without nil entries")
	budgets[1] = NewWindowBudget(time.Second, 1)
	assert.Same(t, shared, r.shared[0], "shared budgets should not alias the caller's slice")
	assert.Empty(t, GetRateLimiterWithShared(own, 1).shared, "a limiter without shared budgets should hold none")
}

func TestGetRateLimiterWithWindowBudget(t *testing.T) {
	t.Parallel()
	own, shared, other := NewWindowBudget(time.Second, 10), NewWindowBudget(5*time.Second, 570), NewWindowBudget(100*time.Millisecond, 12)
	budgets := []*WindowBudget{nil, shared, own, other, shared}
	r := GetRateLimiterWithWindowBudget(own, 2, budgets...)
	require.NotNil(t, r, "limiter must not be nil")
	assert.Same(t, own, r.budget, "the endpoint's own budget should be kept, so that definitions can share it")
	assert.Equal(t, Weight(2), r.weight, "weight should be 2")
	assert.Equal(t, []*WindowBudget{shared, other}, r.shared, "shared budgets should be kept once each, without nil entries or the endpoint's own budget")
	assert.Equal(t, rate.Inf, r.limiter.Limit(), "the endpoint's limiter should be unrestricted, since its own budget limits it")
	budgets[1] = NewWindowBudget(time.Second, 1)
	assert.Same(t, shared, r.shared[0], "shared budgets should not alias the caller's slice")
	unbudgeted := GetRateLimiterWithWindowBudget(nil, 1, shared)
	assert.Nil(t, unbudgeted.budget, "a nil budget should leave the endpoint without one of its own")
	assert.Equal(t, []*WindowBudget{shared}, unbudgeted.shared, "a nil budget should leave the shared budgets to limit the endpoint")
}

func TestRateLimitWithShared(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		shared := NewWindowBudget(500*time.Millisecond, 1)
		first := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, shared)
		second := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, shared)

		start := time.Now()
		require.NoError(t, first.RateLimit(t.Context()), "first request must not error")
		assert.Zero(t, time.Since(start), "first request should not wait")

		start = time.Now()
		require.NoError(t, second.RateLimit(t.Context()), "second request must not error")
		assert.Equal(t, 500*time.Millisecond, time.Since(start), "a request on another endpoint should wait for the shared budget")

		slow := GetRateLimiterWithShared(NewRateLimit(time.Second, 1), 1, NewWindowBudget(500*time.Millisecond, 1))
		require.NoError(t, slow.RateLimit(t.Context()), "slow endpoint's first request must not error")
		start = time.Now()
		require.NoError(t, slow.RateLimit(t.Context()), "slow endpoint's second request must not error")
		assert.Equal(t, time.Second, time.Since(start), "a request should wait for the slowest of its limits")

		weighted := GetRateLimiterWithShared(NewRateLimit(time.Second, 100), 5, NewWindowBudget(time.Second, 1))
		start = time.Now()
		require.NoError(t, weighted.RateLimit(t.Context()), "weighted request must not error")
		assert.Equal(t, 40*time.Millisecond, time.Since(start), "a weighted request should take its weight from its own limiter but one request from a shared budget")

		time.Sleep(time.Second)
		require.NoError(t, first.RateLimit(t.Context()), "request must not error once the shared window has passed")
		err := second.RateLimit(WithDelayNotAllowed(t.Context()))
		assert.ErrorIs(t, err, ErrDelayNotAllowed, "a request the shared budget would delay should be refused when delays are not allowed")
		time.Sleep(500 * time.Millisecond)
		require.NoError(t, second.RateLimit(WithDelayNotAllowed(t.Context())), "a refused request must take nothing from the shared budget")
	})
}

func TestRateLimitWithSharedReleasesReservations(t *testing.T) {
	t.Parallel()

	t.Run("cancelled while waiting", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			shared := NewWindowBudget(time.Second, 1)
			own := rate.NewLimiter(rate.Limit(1), 1)
			first := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, shared)
			waiting := GetRateLimiterWithShared(own, 2, shared)
			require.NoError(t, first.RateLimit(t.Context()), "first request must take the shared budget")

			ctx, cancel := context.WithCancel(t.Context())
			errs := make(chan error, 1)
			go func() { errs <- waiting.RateLimit(ctx) }()
			synctest.Wait()
			cancel()
			require.ErrorIs(t, <-errs, context.Canceled, "a request cancelled while waiting must return the cancellation")

			// Without the release, the cancelled request's booking and tokens, due in one second, would keep both budgets
			// taken for another second
			time.Sleep(time.Second)
			require.NoError(t, first.RateLimit(WithDelayNotAllowed(t.Context())), "a cancelled request must give back its shared booking")
			require.NoError(t, GetRateLimiterWithWeight(own, 1).RateLimit(WithDelayNotAllowed(t.Context())), "a cancelled request must give back its own reservations")
		})
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			shared := NewWindowBudget(time.Second, 1)
			first := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, shared)
			second := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, shared)
			require.NoError(t, first.RateLimit(t.Context()), "first request must take the shared budget")

			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			require.ErrorIs(t, second.RateLimit(ctx), context.DeadlineExceeded, "a request the shared budget would delay past its deadline must be refused")
			time.Sleep(time.Second)
			require.NoError(t, second.RateLimit(WithDelayNotAllowed(t.Context())), "a request refused for its deadline must take nothing from the shared budget")
		})
	})
}

func TestRateLimitBarrierWithShared(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		requests int
		err      error
	}{
		{"a shared budget with room for one participant refuses the group", 1, ErrDelayNotAllowed},
		{"a shared budget with room for every participant admits the group", 2, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
			require.NoError(t, err, "barrier contexts must be created")
			shared := NewWindowBudget(time.Hour, tc.requests)
			left := GetRateLimiterWithShared(NewRateLimit(time.Hour, 1), 1, shared)
			right := GetRateLimiterWithShared(NewRateLimit(time.Hour, 1), 1, shared)
			errs := make(chan error, 2)
			go func() { errs <- left.RateLimit(contexts[0]) }()
			go func() { errs <- right.RateLimit(contexts[1]) }()
			for range 2 {
				assert.ErrorIs(t, <-errs, tc.err, "every participant should share the group's outcome")
			}
			rateLimitReservationMu.Lock()
			booked := len(shared.booked)
			rateLimitReservationMu.Unlock()
			if tc.err != nil {
				assert.Zero(t, booked, "a refused group should book nothing in the shared budget")
				assert.NoError(t, left.RateLimit(WithDelayNotAllowed(t.Context())), "a refused group should give back its own reservations")
				return
			}
			assert.Equal(t, 2, booked, "an admitted group should book every participant in the shared budget")
		})
	}
}

func TestRateLimitWithSharedConcurrent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		shared := NewWindowBudget(100*time.Millisecond, 1)
		start := time.Now()
		errs := common.ErrorCollector{}
		for range 10 {
			endpoint := GetRateLimiterWithShared(NewRateLimit(time.Second, 100), 1, shared)
			errs.Go(func() error { return endpoint.RateLimit(t.Context()) })
		}
		require.NoError(t, errs.Collect(), "concurrent requests must not error")
		assert.Equal(t, 900*time.Millisecond, time.Since(start), "requests on separate endpoints should queue on their shared budget")
	})
}

// weightedSend is when a request was sent and how many requests it counted as
type weightedSend struct {
	at     time.Duration
	weight int
}

// busiestWeightedWindow returns the most requests counted in any half-open window of the given length
func busiestWeightedWindow(sends []weightedSend, window time.Duration) int {
	sorted := slices.SortedFunc(slices.Values(sends), func(a, b weightedSend) int { return cmp.Compare(a.at, b.at) })
	var busiest int
	for i := range sorted {
		var held int
		for _, s := range sorted[i:] {
			if s.at >= sorted[i].at+window {
				break
			}
			held += s.weight
		}
		busiest = max(busiest, held)
	}
	return busiest
}

func TestRateLimitWithWindowBudget(t *testing.T) {
	t.Parallel()

	t.Run("a request after a batch of the whole budget waits a whole window", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			orders := GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1)
			start := time.Now()
			require.NoError(t, orders.RateLimit(WithWeight(t.Context(), 10)), "a batch of the whole budget must not error")
			assert.Zero(t, time.Since(start), "a batch of the whole budget should go at once on an idle budget")
			require.NoError(t, orders.RateLimit(t.Context()), "a request after the batch must not error")
			assert.Equal(t, time.Second, time.Since(start), "a request after a batch of the whole budget should wait until the batch leaves the window")
		})
	})

	t.Run("concurrent batches of the whole budget go a window apart", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			orders := GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1)
			start := time.Now()
			var mu sync.Mutex
			var sends []time.Duration
			var errs common.ErrorCollector
			for range 3 {
				errs.Go(func() error {
					if err := orders.RateLimit(WithWeight(t.Context(), 10)); err != nil {
						return err
					}
					mu.Lock()
					sends = append(sends, time.Since(start))
					mu.Unlock()
					return nil
				})
			}
			require.NoError(t, errs.Collect(), "concurrent batches must not error")
			slices.Sort(sends)
			assert.Equal(t, []time.Duration{0, time.Second, 2 * time.Second}, sends, "batches of the whole budget should go a window apart")
		})
	})

	for _, tc := range []struct {
		name    string
		singles int
		exp     time.Duration
	}{
		{"a batch fitting the room left in the window goes at once", 7, 600 * time.Millisecond},
		{"a batch larger than the room left waits until the window has room for all of it", 8, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				orders := GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1)
				start := time.Now()
				for i := range tc.singles {
					if i > 0 {
						time.Sleep(100 * time.Millisecond)
					}
					require.NoError(t, orders.RateLimit(WithDelayNotAllowed(t.Context())), "a single request must go at once")
				}
				require.NoError(t, orders.RateLimit(WithWeight(t.Context(), 3)), "a batch of three must not error")
				assert.Equal(t, tc.exp, time.Since(start), "a batch of three should go once no window holding it holds more than the budget")
			})
		})
	}

	t.Run("no window holds more than the budget under a mixed load", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			budget := NewWindowBudget(time.Second, 10)
			shared := NewWindowBudget(time.Second, 12)
			rest, websocket := GetRateLimiterWithWindowBudget(budget, 1, shared), GetRateLimiterWithWindowBudget(budget, 1)
			random := rand.New(rand.NewPCG(3, 4)) //nolint:gosec // A fixed seed keeps the load reproducible
			start := time.Now()
			var mu sync.Mutex
			var sends, restSends []weightedSend
			var errs common.ErrorCollector
			weights := []Weight{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			for range 120 {
				arrival, weight, overREST := time.Duration(random.IntN(5000))*time.Millisecond, weights[random.IntN(len(weights))], random.IntN(2) == 0
				if random.IntN(2) == 0 {
					weight = 1
				}
				errs.Go(func() error {
					time.Sleep(arrival)
					limiter := websocket
					if overREST {
						limiter = rest
					}
					if err := limiter.RateLimit(WithWeight(t.Context(), weight)); err != nil {
						return err
					}
					sent := weightedSend{time.Since(start), int(weight)}
					if sent.at < arrival {
						return fmt.Errorf("a request asked for at %s was sent at %s", arrival, sent.at)
					}
					mu.Lock()
					sends = append(sends, sent)
					if overREST {
						restSends = append(restSends, weightedSend{sent.at, 1})
					}
					mu.Unlock()
					return nil
				})
			}
			require.NoError(t, errs.Collect(), "every request must be sent, and none before it asked")
			require.Len(t, sends, 120, "every request must be sent")
			assert.Equal(t, 10, busiestWeightedWindow(sends, time.Second), "the busiest second should fill the budget without exceeding it")
			assert.LessOrEqual(t, busiestWeightedWindow(restSends, time.Second), 12, "the shared budget should count each request once, whatever its weight")
		})
	})

	t.Run("a request takes a moment ahead of one held back by a shared budget", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			budget, shared := NewWindowBudget(time.Second, 10), NewWindowBudget(5*time.Second, 1)
			require.NoError(t, sharedOnly(shared).RateLimit(t.Context()), "a request must fill the shared budget")
			start := time.Now()
			held := make(chan time.Duration, 1)
			go func() {
				assert.NoError(t, GetRateLimiterWithWindowBudget(budget, 1, shared).RateLimit(t.Context()), "the held back request should be sent")
				held <- time.Since(start)
			}()
			synctest.Wait()
			time.Sleep(10 * time.Millisecond)
			require.NoError(t, GetRateLimiterWithWindowBudget(budget, 1).RateLimit(WithDelayNotAllowed(t.Context())), "a request on the budget alone must not wait for one the shared budget holds back")
			assert.Equal(t, 5*time.Second, <-held, "the held back request should be sent when the shared budget has room")
		})
	})

	t.Run("refusals book nothing", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			budget := NewWindowBudget(time.Second, 10)
			orders := GetRateLimiterWithWindowBudget(budget, 1)
			require.NoError(t, orders.RateLimit(WithWeight(t.Context(), 8)), "a batch of eight must go at once")
			require.ErrorIs(t, orders.RateLimit(WithWeight(t.Context(), 11)), ErrWeightExceedsBudget, "a batch larger than the budget must be refused")
			require.ErrorIs(t, orders.RateLimit(WithWeight(WithDelayNotAllowed(t.Context()), 3)), ErrDelayNotAllowed, "a batch that would wait must be refused when delays are not allowed")
			ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
			defer cancel()
			require.ErrorIs(t, orders.RateLimit(WithWeight(ctx, 3)), context.DeadlineExceeded, "a batch that would wait past its deadline must be refused")
			rateLimitReservationMu.Lock()
			booked := len(budget.booked)
			rateLimitReservationMu.Unlock()
			assert.Equal(t, 8, booked, "refused batches should book nothing")
			assert.NoError(t, orders.RateLimit(WithWeight(WithDelayNotAllowed(t.Context()), 2)), "the room refused batches would have taken should be left")
		})
	})

	t.Run("a cancelled batch gives back every request it booked", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			budget := NewWindowBudget(time.Second, 10)
			orders := GetRateLimiterWithWindowBudget(budget, 1)
			require.NoError(t, orders.RateLimit(WithWeight(t.Context(), 10)), "a batch of the whole budget must go at once")
			ctx, cancel := context.WithCancel(t.Context())
			errs := make(chan error, 1)
			go func() { errs <- orders.RateLimit(WithWeight(ctx, 5)) }()
			synctest.Wait()
			time.Sleep(500 * time.Millisecond)
			cancel()
			require.ErrorIs(t, <-errs, context.Canceled, "a batch cancelled while waiting must return the cancellation")
			time.Sleep(500 * time.Millisecond)
			assert.NoError(t, orders.RateLimit(WithWeight(WithDelayNotAllowed(t.Context()), 10)), "a cancelled batch should leave the whole budget once the first batch leaves the window")
		})
	})
}

func TestRateLimitBarrierWithWindowBudget(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		weights []Weight
		err     error
		booked  int
	}{
		{"a budget with room for every participant's weight admits the group", []Weight{6, 4}, nil, 10},
		{"a budget without room for every participant's weight refuses the group", []Weight{6, 5}, ErrDelayNotAllowed, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			contexts, err := NewRateLimitBarrierContexts(t.Context(), uint(len(tc.weights)))
			require.NoError(t, err, "barrier contexts must be created")
			budget := NewWindowBudget(time.Hour, 10)
			errs := make(chan error, len(tc.weights))
			for i, weight := range tc.weights {
				orders := GetRateLimiterWithWindowBudget(budget, 1)
				go func() { errs <- orders.RateLimit(WithWeight(contexts[i], weight)) }()
			}
			for range tc.weights {
				assert.ErrorIs(t, <-errs, tc.err, "every participant should share the group's outcome")
			}
			rateLimitReservationMu.Lock()
			booked := len(budget.booked)
			rateLimitReservationMu.Unlock()
			assert.Equal(t, tc.booked, booked, "a group should book every participant's weight only when it is admitted")
		})
	}
}

func TestGetRateLimiterWithWeight(t *testing.T) {
	t.Parallel()

	r := rate.NewLimiter(rate.Limit(10), 1)
	weighted := GetRateLimiterWithWeight(r, 5)
	require.NotNil(t, weighted, "weighted limiter must not be nil")
	assert.Equal(t, Weight(5), weighted.weight, "weight should be 5")
	assert.Equal(t, r, weighted.limiter, "should reference same limiter")
	assert.Equal(t, Weight(5), weighted.Weight(), "weight should match the configured request cost")
	assert.Zero(t, (*RateLimiterWithWeight)(nil).Weight(), "nil limiter weight should be zero")
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

func TestInitiateRateLimit(t *testing.T) {
	t.Parallel()

	var r *Requester
	err := r.InitiateRateLimit(t.Context(), Unset)
	assert.ErrorIs(t, err, ErrRequestSystemIsNil, "should return correct error")

	r = &Requester{}
	r.disableRateLimiter.Store(true)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.NoError(t, err, "should not error when rate limiter is disabled")
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	go func() { errs <- r.InitiateRateLimit(contexts[0], Unset) }()
	require.Never(t, func() bool { return len(errs) != 0 }, 10*time.Millisecond, time.Millisecond,
		"disabled limiter participant must wait for its peer")
	go func() { errs <- r.InitiateRateLimit(contexts[1], Unset) }()
	require.NoError(t, <-errs, "disabled limiter barrier participant must not error")
	require.NoError(t, <-errs, "disabled limiter barrier participant must not error")

	r.disableRateLimiter.Store(false)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.ErrorContains(t, err, "nil pointer: request.RateLimitDefinitions", "should return correct error when limiter is nil")

	r.limiter = NewBasicRateLimit(time.Second, 10, 1)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.NoError(t, err, "should not error on valid rate limit initiation")
}

// reserveAt reserves a request of its definition's weight at now as RateLimit does, and returns when it is sent and
// what it reserved
func reserveAt(tb testing.TB, r *RateLimiterWithWeight, now time.Time) (time.Time, *rateLimitReservation) {
	tb.Helper()
	return reserveWeightAt(tb, r, r.weight, now)
}

// reserveWeightAt reserves a request of the given weight at now as RateLimit does, and returns when it is sent and what
// it reserved
func reserveWeightAt(tb testing.TB, r *RateLimiterWithWeight, weight Weight, now time.Time) (time.Time, *rateLimitReservation) {
	tb.Helper()
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	delay, plan := r.planLocked(now, weight)
	require.NotEqual(tb, rate.InfDuration, delay, "every limit must be able to take the request")
	return now.Add(delay), plan.reserveLocked(now)
}

// sharedOnly returns a definition drawing on a shared budget alone
func sharedOnly(b *WindowBudget) *RateLimiterWithWeight {
	return GetRateLimiterWithShared(NewRateLimit(time.Second, 0), 1, b)
}

// busiestWindow returns the most sends, given as offsets, in any half-open window of the given length
func busiestWindow(sends []time.Duration, window time.Duration) int {
	sorted := slices.Sorted(slices.Values(sends))
	var busiest int
	for i, start := range sorted {
		end, _ := slices.BinarySearch(sorted[i:], start+window)
		busiest = max(busiest, end)
	}
	return busiest
}

func TestRateLimitWithSharedSendTimes(t *testing.T) {
	t.Parallel()

	t.Run("an endpoint behind a full shared budget keeps its own rate", func(t *testing.T) {
		t.Parallel()
		ip := NewWindowBudget(5*time.Second, 600)
		endpoint := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip)
		start := time.Now()
		for range 600 {
			reserveAt(t, sharedOnly(ip), start.Add(-3*time.Second))
		}
		exp, sends := make([]time.Duration, 20), make([]time.Duration, 20)
		for i := range sends {
			exp[i] = 2*time.Second + time.Duration(i)*100*time.Millisecond
			send, _ := reserveAt(t, endpoint, start)
			sends[i] = send.Sub(start)
		}
		assert.Equal(t, exp, sends, "concurrent requests should leave at the endpoint's spacing once the shared budget has room")
		assert.Equal(t, 10, busiestWindow(sends, time.Second), "an endpoint of 10 a second should send no more than 10 requests in any second")
	})

	t.Run("a request on an idle endpoint does not wait for another endpoint's backlog", func(t *testing.T) {
		t.Parallel()
		ip := NewWindowBudget(5*time.Second, 600)
		busy := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip)
		start := time.Now()
		var last time.Time
		for range 30 {
			last, _ = reserveAt(t, busy, start)
		}
		require.Equal(t, 2900*time.Millisecond, last.Sub(start), "the busy endpoint's requests must queue on its own budget")
		send, _ := reserveAt(t, GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip), start)
		assert.Equal(t, start, send, "a request on an idle endpoint should be sent at once")
	})

	t.Run("a shared budget holds while an endpoint holds requests back", func(t *testing.T) {
		t.Parallel()
		ip := NewWindowBudget(5*time.Second, 600)
		endpoint := GetRateLimiterWithShared(NewRateLimit(time.Second, 5), 1, ip)
		start := time.Now()
		sends := make([]time.Duration, 0, 1250)
		for range 50 {
			send, _ := reserveAt(t, endpoint, start)
			sends = append(sends, send.Sub(start))
		}
		for range 1200 {
			send, _ := reserveAt(t, sharedOnly(ip), start)
			sends = append(sends, send.Sub(start))
		}
		assert.Zero(t, sends[50], "requests on the shared budget alone should not wait for the endpoint's backlog")
		assert.LessOrEqual(t, busiestWindow(sends, 5*time.Second), 600, "no five seconds should hold more than the shared budget's 600 requests")
	})

	t.Run("steady shared load does not starve an endpoint request", func(t *testing.T) {
		t.Parallel()
		ip := NewWindowBudget(5*time.Second, 600)
		endpoint := GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 1, ip)
		spacing := time.Second / 240
		start := time.Now()
		var lastBefore, firstAfter, endpointSend time.Time
		// Requests on the shared budget alone arrive at twice its rate for ten seconds, the endpoint's once it is full
		for i := range 2400 {
			now := start.Add(time.Duration(i) * spacing)
			if i == 720 {
				endpointSend, _ = reserveAt(t, endpoint, now)
			}
			send, _ := reserveAt(t, sharedOnly(ip), now)
			switch i {
			case 719:
				lastBefore = send
			case 720:
				firstAfter = send
			}
		}
		assert.True(t, endpointSend.After(lastBefore), "the endpoint request should be sent after the requests that came before it")
		assert.LessOrEqual(t, endpointSend.Sub(lastBefore), spacing, "the endpoint request should take the first moment with room after the requests that came before it")
		assert.True(t, firstAfter.After(endpointSend), "a request arriving after the endpoint request should be sent after it")
	})

	t.Run("a shared budget books a free moment ahead of later bookings", func(t *testing.T) {
		t.Parallel()
		ip := NewWindowBudget(time.Second, 1)
		own := rate.NewLimiter(rate.Every(2*time.Second), 1)
		start := time.Now()
		own.ReserveN(start, 1)
		later, _ := reserveAt(t, GetRateLimiterWithShared(own, 1, ip), start)
		require.Equal(t, 2*time.Second, later.Sub(start), "the endpoint's request must wait for its own budget")
		exp := []time.Duration{0, time.Second, 3 * time.Second}
		sends := make([]time.Duration, len(exp))
		for i := range sends {
			send, _ := reserveAt(t, sharedOnly(ip), start)
			sends[i] = send.Sub(start)
		}
		assert.Equal(t, exp, sends, "requests should fill the free moments around the booking for later, without taking it")
	})

	t.Run("a weighted request takes its tokens one by one up to its send time", func(t *testing.T) {
		t.Parallel()
		shared := NewWindowBudget(time.Second, 1)
		own := NewRateLimit(time.Second, 10)
		start := time.Now()
		reserveAt(t, sharedOnly(shared), start)
		send, reserved := reserveAt(t, GetRateLimiterWithShared(own, 5, shared), start)
		assert.Equal(t, time.Second, send.Sub(start), "a weighted request should wait for the shared budget")
		delays := make([]time.Duration, len(reserved.tokens))
		for i, r := range reserved.tokens {
			delays[i] = r.DelayFrom(start)
		}
		exp := []time.Duration{600 * time.Millisecond, 700 * time.Millisecond, 800 * time.Millisecond, 900 * time.Millisecond, time.Second}
		assert.Equal(t, exp, delays, "the endpoint's tokens should fall due one by one, the last when the request is sent")
		next, _ := reserveAt(t, GetRateLimiterWithWeight(own, 1), start)
		assert.Equal(t, 1100*time.Millisecond, next.Sub(start), "the endpoint's next request should wait a full interval after the weighted one is sent")
	})

	t.Run("idle limits send at once and take the request now", func(t *testing.T) {
		t.Parallel()
		own, shared := NewRateLimit(time.Second, 10), NewWindowBudget(time.Second, 10)
		start := time.Now()
		send, reserved := reserveAt(t, GetRateLimiterWithShared(own, 1, shared), start)
		assert.Equal(t, start, send, "a request on idle limits should be sent at once")
		assert.Len(t, reserved.tokens, 1, "a request should take one token from its own limiter")
		rateLimitReservationMu.Lock()
		defer rateLimitReservationMu.Unlock()
		assert.Equal(t, []time.Time{start}, shared.booked, "a request should be booked in the shared budget for now")
		assert.NotContains(t, futureReservations, own, "a request sent at once should leave its endpoint's clock at now")
	})
}

func TestRateLimitWithSharedRefusalsReserveNothing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ctx  func(context.Context) (context.Context, context.CancelFunc)
		err  error
	}{
		{"delay not allowed", func(ctx context.Context) (context.Context, context.CancelFunc) {
			return WithDelayNotAllowed(ctx), func() {}
		}, ErrDelayNotAllowed},
		{"deadline too soon", func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithTimeout(ctx, 50*time.Millisecond)
		}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				shared := NewWindowBudget(time.Second, 1)
				own := NewRateLimit(time.Second, 10)
				require.NoError(t, sharedOnly(shared).RateLimit(t.Context()), "a request must fill the shared budget")
				ctx, cancel := tc.ctx(t.Context())
				defer cancel()
				require.ErrorIs(t, GetRateLimiterWithShared(own, 1, shared).RateLimit(ctx), tc.err, "a request the shared budget would delay must be refused")
				assert.NoError(t, GetRateLimiterWithWeight(own, 1).RateLimit(WithDelayNotAllowed(t.Context())), "a refused request should leave its endpoint's clock and budget untouched")
				rateLimitReservationMu.Lock()
				booked := len(shared.booked)
				rateLimitReservationMu.Unlock()
				assert.Equal(t, 1, booked, "a refused request should book nothing in the shared budget")
			})
		})
	}
}

// bookedBudget returns a budget of the given requests a second holding n bookings at t
func bookedBudget(requests int, t time.Time, n int) *WindowBudget {
	return &WindowBudget{window: time.Second, requests: requests, booked: slices.Repeat([]time.Time{t}, n)}
}

func TestPlanLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	drained := func(l *rate.Limiter) *rate.Limiter {
		l.ReserveN(now, l.Burst())
		return l
	}
	full := func() *WindowBudget { return &WindowBudget{window: time.Second, requests: 1, booked: []time.Time{now}} }
	idleOwn, busyOwn, weightedOwn, burstOwn, contextOwn := NewRateLimit(time.Second, 10), drained(NewRateLimit(time.Second, 10)), NewRateLimit(time.Second, 10), rate.NewLimiter(10, 5), NewRateLimit(time.Second, 10)
	idleShared, waitingShared, fullShared, laterShared := NewWindowBudget(time.Second, 1), NewWindowBudget(time.Second, 1), full(), &WindowBudget{window: time.Second, requests: 1, booked: []time.Time{now.Add(time.Second)}}
	idleOrders, sevenOrders, eightOrders, ip := NewWindowBudget(time.Second, 10), bookedBudget(10, now, 7), bookedBudget(10, now, 8), NewWindowBudget(time.Second, 600)
	// Each has room only in the seconds the other is full, until both have room from the sixth second on
	oddSeconds := &WindowBudget{window: time.Second, requests: 1, booked: []time.Time{now.Add(time.Second), now.Add(3 * time.Second), now.Add(5 * time.Second)}}
	evenSeconds := &WindowBudget{window: time.Second, requests: 1, booked: []time.Time{now, now.Add(2 * time.Second), now.Add(4 * time.Second)}}
	for _, tc := range []struct {
		name    string
		limiter *RateLimiterWithWeight
		weight  Weight
		delay   time.Duration
		plan    *rateLimitPlan
	}{
		{"idle limits are taken now", GetRateLimiterWithShared(idleOwn, 1, idleShared), 1, 0, &rateLimitPlan{idleOwn, 1, now, now, []windowBooking{{idleShared, 1}}}},
		{"unrestricted limits are left out", GetRateLimiterWithShared(NewRateLimit(time.Second, 0), 3), 3, 0, &rateLimitPlan{sendAt: now}},
		{"a shared budget takes the request when the endpoint grants it", GetRateLimiterWithShared(busyOwn, 1, waitingShared), 1, 100 * time.Millisecond, &rateLimitPlan{busyOwn, 1, now, now.Add(100 * time.Millisecond), []windowBooking{{waitingShared, 1}}}},
		{"a weighted endpoint's last token falls due when the shared budget has room", GetRateLimiterWithShared(weightedOwn, 3, fullShared), 3, time.Second, &rateLimitPlan{weightedOwn, 3, now.Add(800 * time.Millisecond), now.Add(time.Second), []windowBooking{{fullShared, 1}}}},
		{"a weight within the burst is reserved for the send time", GetRateLimiterWithShared(burstOwn, 3, fullShared), 3, time.Second, &rateLimitPlan{burstOwn, 3, now.Add(time.Second), now.Add(time.Second), []windowBooking{{fullShared, 1}}}},
		{"the weight given takes the place of the definition's", GetRateLimiterWithWeight(contextOwn, 1), 3, 200 * time.Millisecond, &rateLimitPlan{contextOwn, 3, now, now.Add(200 * time.Millisecond), nil}},
		{"every shared budget has room at the send time", GetRateLimiterWithShared(NewRateLimit(time.Second, 0), 1, fullShared, laterShared), 1, 2 * time.Second, &rateLimitPlan{sendAt: now.Add(2 * time.Second), bookings: []windowBooking{{fullShared, 1}, {laterShared, 1}}}},
		{"budgets are checked again at each moment another moves the request to, until all have room", GetRateLimiterWithShared(NewRateLimit(time.Second, 0), 1, oddSeconds, evenSeconds), 1, 6 * time.Second, &rateLimitPlan{sendAt: now.Add(6 * time.Second), bookings: []windowBooking{{oddSeconds, 1}, {evenSeconds, 1}}}},
		{"a limiter that never grants plans nothing", GetRateLimiterWithShared(rate.NewLimiter(1, 0), 1, idleShared), 1, rate.InfDuration, &rateLimitPlan{}},
		{"a window budget of the endpoint's own books the weight, and a shared budget one request", GetRateLimiterWithWindowBudget(idleOrders, 1, ip), 10, 0, &rateLimitPlan{sendAt: now, bookings: []windowBooking{{idleOrders, 10}, {ip, 1}}}},
		{"a window budget with room for the weight takes it now", GetRateLimiterWithWindowBudget(sevenOrders, 1), 3, 0, &rateLimitPlan{sendAt: now, bookings: []windowBooking{{sevenOrders, 3}}}},
		{"a window budget without room for the weight waits until a window has passed", GetRateLimiterWithWindowBudget(eightOrders, 1), 3, time.Second, &rateLimitPlan{sendAt: now.Add(time.Second), bookings: []windowBooking{{eightOrders, 3}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rateLimitReservationMu.Lock()
			delay, plan := tc.limiter.planLocked(now, tc.weight)
			rateLimitReservationMu.Unlock()
			assert.Equal(t, tc.delay, delay, "planLocked should return the wait until every limit has room")
			assert.Equal(t, tc.plan, plan, "planLocked should reserve every limit for the send time")
		})
	}

	t.Run("a limiter reserved for later is planned from then", func(t *testing.T) {
		t.Parallel()
		l := NewRateLimit(time.Second, 10)
		rateLimitReservationMu.Lock()
		defer rateLimitReservationMu.Unlock()
		(&rateLimitPlan{limiter: l, tokens: 1, reserveAt: now.Add(2 * time.Second)}).reserveLocked(now)
		delay, plan := GetRateLimiterWithWeight(l, 1).planLocked(now, 1)
		assert.Equal(t, 2100*time.Millisecond, delay, "planLocked should wait a full interval after the later reservation")
		assert.Equal(t, &rateLimitPlan{l, 1, now.Add(2 * time.Second), now.Add(2100 * time.Millisecond), nil}, plan, "planLocked should not reserve a limiter before its later reservation")
	})

	t.Run("planning forgets bookings no window from now can reach", func(t *testing.T) {
		t.Parallel()
		shared, own := &WindowBudget{window: time.Second, requests: 1, booked: []time.Time{now.Add(-time.Second)}}, bookedBudget(1, now.Add(-time.Second), 1)
		rateLimitReservationMu.Lock()
		defer rateLimitReservationMu.Unlock()
		delay, _ := GetRateLimiterWithWindowBudget(own, 1, shared).planLocked(now, 1)
		assert.Zero(t, delay, "a booking a whole window before now should not delay a request")
		assert.Empty(t, shared.booked, "planLocked should forget a shared booking a whole window before now")
		assert.Empty(t, own.booked, "planLocked should forget an own booking a whole window before now")
	})
}

func TestRateLimiterWithWeightBookings(t *testing.T) {
	t.Parallel()
	own, shared, other := NewWindowBudget(time.Second, 10), NewWindowBudget(time.Second, 600), NewWindowBudget(time.Second, 12)
	for _, tc := range []struct {
		name    string
		limiter *RateLimiterWithWeight
		weight  Weight
		exp     []windowBooking
	}{
		{"a limiter alone books nothing", NewRateLimitWithWeight(time.Second, 10, 5), 5, nil},
		{"shared budgets book one request each whatever the weight", GetRateLimiterWithShared(NewRateLimit(time.Second, 10), 5, shared, other), 5, []windowBooking{{shared, 1}, {other, 1}}},
		{"a window budget of the endpoint's own books the weight", GetRateLimiterWithWindowBudget(own, 1), 7, []windowBooking{{own, 7}}},
		{"an own budget is booked before shared ones", GetRateLimiterWithWindowBudget(own, 1, shared, other), 3, []windowBooking{{own, 3}, {shared, 1}, {other, 1}}},
	} {
		assert.Equalf(t, tc.exp, tc.limiter.bookings(tc.weight), "bookings should return what %s books", tc.name)
	}
}

func TestRateLimiterWithWeightWeightFor(t *testing.T) {
	t.Parallel()
	limited, budgeted := NewRateLimitWithWeight(time.Second, 10, 2), GetRateLimiterWithWindowBudget(NewWindowBudget(time.Second, 10), 1)
	unweighted := func(ctx context.Context) context.Context { return ctx }
	weighted := func(weight Weight) func(context.Context) context.Context {
		return func(ctx context.Context) context.Context { return WithWeight(ctx, weight) }
	}
	for _, tc := range []struct {
		name    string
		limiter *RateLimiterWithWeight
		weigh   func(context.Context) context.Context
		exp     Weight
		err     error
	}{
		{"a context without a weight of its own", limited, unweighted, 2, nil},
		{"a context's weight in place of the definition's", limited, weighted(200), 200, nil},
		{"a definition of no weight", NewRateLimitWithWeight(time.Second, 10, 0), unweighted, 0, errInvalidWeight},
		{"a context of no weight", limited, weighted(0), 0, errInvalidWeight},
		{"a weight a window budget admits in one window", budgeted, weighted(10), 10, nil},
		{"a weight a window budget never admits", budgeted, weighted(11), 0, ErrWeightExceedsBudget},
		{"any weight without a window budget", GetRateLimiterWithWindowBudget(nil, 1), weighted(255), 255, nil},
	} {
		weight, err := tc.limiter.weightFor(tc.weigh(t.Context()))
		require.ErrorIsf(t, err, tc.err, "weightFor must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, weight, "weightFor should return the weight for %s", tc.name)
	}
}

func TestRateLimitPlanReserveLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	own, idle, shared, orders := NewRateLimit(time.Second, 10), NewRateLimit(time.Second, 10), NewWindowBudget(time.Second, 2), bookedBudget(10, now, 1)
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	plan := &rateLimitPlan{own, 2, now.Add(time.Second), now.Add(1100 * time.Millisecond), []windowBooking{{shared, 1}, {orders, 3}}}
	res := plan.reserveLocked(now)
	assert.Same(t, plan, res.plan, "a reservation should keep its plan to give it back")
	delays := make([]time.Duration, len(res.tokens))
	for i, r := range res.tokens {
		delays[i] = r.DelayFrom(now)
	}
	assert.Equal(t, []time.Duration{time.Second, 1100 * time.Millisecond}, delays, "reserveLocked should take each token one at a time at the planned time")
	assert.Equal(t, now.Add(time.Second), futureReservations[own], "a limiter reserved for later should keep its clock until then")
	assert.Equal(t, []time.Time{now.Add(1100 * time.Millisecond)}, shared.booked, "reserveLocked should book the send time once in a shared budget")
	sent := now.Add(1100 * time.Millisecond)
	assert.Equal(t, []time.Time{now, sent, sent, sent}, orders.booked, "reserveLocked should book the send time once for each request of the weight in an own budget")
	(&rateLimitPlan{limiter: idle, tokens: 1, reserveAt: now, sendAt: now}).reserveLocked(now)
	assert.NotContains(t, futureReservations, idle, "a limiter reserved now should not be kept as reserved for later")
	assert.Empty(t, (&rateLimitPlan{}).reserveLocked(now).tokens, "an empty plan should reserve nothing")
}

func TestRateLimitReservationCancelLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	(&rateLimitPlan{}).reserveLocked(now).cancelLocked(now)

	own, shared, orders := rate.NewLimiter(1, 1), NewWindowBudget(time.Second, 1), bookedBudget(10, now.Add(time.Second), 2)
	res := (&rateLimitPlan{own, 2, now, now.Add(time.Second), []windowBooking{{shared, 1}, {orders, 3}}}).reserveLocked(now)
	require.Equal(t, -1.0, own.TokensAt(now), "two reservations on a burst of one must leave a token owed")
	res.cancelLocked(now)
	assert.Equal(t, 1.0, own.TokensAt(now), "giving back tokens reserved now should restore the limiter")
	assert.Empty(t, shared.booked, "giving back should remove the request's booking")
	assert.Equal(t, []time.Time{now.Add(time.Second), now.Add(time.Second)}, orders.booked, "giving back should remove the weight's bookings and leave other requests' bookings for the same moment")

	later := rate.NewLimiter(1, 2)
	later.ReserveN(now, 2)
	(&rateLimitPlan{later, 1, now.Add(time.Second), now.Add(time.Second), nil}).reserveLocked(now).cancelLocked(now)
	delay, _ := GetRateLimiterWithWeight(later, 2).planLocked(now, 2)
	assert.Equal(t, 2*time.Second, delay, "giving back tokens reserved for later should not move the limiter's clock back, which would count the tokens accrued before it twice")
}

func TestCancelReservationsLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	l, shared, orders := rate.NewLimiter(1, 1), NewWindowBudget(time.Second, 2), NewWindowBudget(time.Second, 10)
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	cancelReservationsLocked(nil, now)
	first := (&rateLimitPlan{l, 1, now, now, []windowBooking{{shared, 1}, {orders, 4}}}).reserveLocked(now)
	second := (&rateLimitPlan{l, 1, now, now.Add(time.Second), []windowBooking{{shared, 1}, {orders, 6}}}).reserveLocked(now)
	cancelReservationsLocked([]*rateLimitReservation{first, second}, now)
	assert.Equal(t, 1.0, l.TokensAt(now), "giving back in reverse order should restore every token")
	assert.Empty(t, shared.booked, "giving back should remove every shared booking")
	assert.Empty(t, orders.booked, "giving back should remove every booking of every weight")
}

func TestReservableFromLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	l := rate.NewLimiter(1, 1)
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	assert.Equal(t, now, reservableFromLocked(l, now), "a limiter never reserved for later should be reservable now")
	futureReservations[l] = now.Add(time.Second)
	assert.Equal(t, now.Add(time.Second), reservableFromLocked(l, now), "a limiter reserved for later should be reservable from then")
	assert.Contains(t, futureReservations, l, "a reservation still to come should be kept")
	assert.Equal(t, now.Add(time.Second), reservableFromLocked(l, now.Add(time.Second)), "a limiter should be reservable at the time it was reserved for")
	assert.NotContains(t, futureReservations, l, "a reservation whose time has come should be forgotten")
}

func TestEarliestGrant(t *testing.T) {
	t.Parallel()
	from := time.Now().Round(0)
	drained := rate.NewLimiter(10, 1)
	drained.ReserveN(from, 1)
	stalled := rate.NewLimiter(0, 1)
	stalled.ReserveN(from, 1)
	for _, tc := range []struct {
		name    string
		limiter *rate.Limiter
		tokens  int
		grant   time.Time
		ok      bool
	}{
		{"no burst never grants", rate.NewLimiter(10, 0), 1, time.Time{}, false},
		{"held tokens grant at once", rate.NewLimiter(10, 2), 2, from, true},
		{"a missing token is waited for", drained, 1, from.Add(100 * time.Millisecond), true},
		{"a weight beyond the burst accrues one by one", rate.NewLimiter(10, 1), 3, from.Add(200 * time.Millisecond), true},
		{"no refill never grants", stalled, 1, time.Time{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grant, ok := earliestGrant(tc.limiter, from, tc.tokens)
			assert.Equal(t, tc.ok, ok, "earliestGrant should report whether the limiter ever grants")
			assert.Equal(t, tc.grant, grant, "earliestGrant should return when the tokens have accrued")
		})
	}
}

func TestDurationFromTokens(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		limit  rate.Limit
		tokens float64
		exp    time.Duration
	}{
		{0, 1, rate.InfDuration},
		{-1, 1, rate.InfDuration},
		{10, 0, 0},
		{10, 1, 100 * time.Millisecond},
		{3, 2, 666666666},
		{1e-12, 1e9, rate.InfDuration},
	} {
		assert.Equalf(t, tc.exp, durationFromTokens(tc.limit, tc.tokens), "durationFromTokens should convert %v tokens at %v a second", tc.tokens, tc.limit)
	}
	now := time.Now()
	l := rate.NewLimiter(3, 1)
	l.ReserveN(now, 1)
	assert.Equal(t, l.ReserveN(now, 1).DelayFrom(now), durationFromTokens(3, 1), "durationFromTokens should round as rate.Limiter rounds its reservations")
}

func TestAdmitLocked(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	close(done)
	admit := func(b *rateLimitBarrier) error {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.admitLocked()
	}
	for _, tc := range []struct {
		name        string
		peerDone    <-chan struct{}
		peerLimiter *RateLimiterWithWeight
		peerWeight  Weight
		err         error
	}{
		{"participants granted at once are admitted", make(chan struct{}), NewRateLimitWithWeight(time.Hour, 1, 1), 1, nil},
		{"a participant whose context is done rejects the group", done, NewRateLimitWithWeight(time.Hour, 1, 1), 1, ErrRateLimitBarrierRejected},
		{"a participant that would wait rejects the group", make(chan struct{}), NewRateLimitWithWeight(time.Hour, 1, 2), 2, ErrDelayNotAllowed},
		{"a participant is planned at its own weight rather than its definition's", make(chan struct{}), NewRateLimitWithWeight(time.Hour, 1, 1), 2, ErrDelayNotAllowed},
		{"a shared budget without room rejects the group", make(chan struct{}), GetRateLimiterWithShared(NewRateLimit(time.Hour, 1), 1, &WindowBudget{window: time.Hour, requests: 1, booked: []time.Time{time.Now()}}), 1, ErrDelayNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			first := NewRateLimitWithWeight(time.Hour, 1, 1)
			b := &rateLimitBarrier{participants: []*rateLimitBarrierParticipant{
				{done: make(chan struct{}), limiter: first, weight: 1},
				{done: make(chan struct{})},
				{done: tc.peerDone, limiter: tc.peerLimiter, weight: tc.peerWeight},
			}}
			require.ErrorIs(t, admit(b), tc.err, "admitLocked must resolve the group as one")
			if tc.err != nil {
				assert.InDelta(t, 1, first.limiter.Tokens(), 0.001, "a rejected group should give back the reservations already taken")
				return
			}
			assert.InDelta(t, 0, first.limiter.Tokens(), 0.001, "an admitted group should keep its reservations")
			assert.InDelta(t, 0, tc.peerLimiter.limiter.Tokens(), 0.001, "an admitted group should keep its reservations")
		})
	}

	for _, tc := range []struct {
		name   string
		booked int
		err    error
		exp    int
	}{
		{"a window budget with room for a participant's weight books it", 0, nil, 3},
		{"a window budget without room for a participant's weight rejects the group and books nothing", 1, ErrDelayNotAllowed, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			first, orders := NewRateLimitWithWeight(time.Hour, 1, 1), bookedBudget(3, time.Now(), tc.booked)
			b := &rateLimitBarrier{participants: []*rateLimitBarrierParticipant{
				{done: make(chan struct{}), limiter: first, weight: 1},
				{done: make(chan struct{}), limiter: GetRateLimiterWithWindowBudget(orders, 1), weight: 3},
			}}
			require.ErrorIs(t, admit(b), tc.err, "admitLocked must resolve the group as one")
			rateLimitReservationMu.Lock()
			booked := len(orders.booked)
			rateLimitReservationMu.Unlock()
			assert.Equal(t, tc.exp, booked, "admitLocked should book a participant's weight only when the group is admitted")
		})
	}

	t.Run("a participant cancelled during admission rejects the group", func(t *testing.T) {
		t.Parallel()
		cancelled := make(chan struct{})
		first, second := NewRateLimitWithWeight(time.Hour, 1, 1), NewRateLimitWithWeight(time.Hour, 1, 1)
		b := &rateLimitBarrier{participants: []*rateLimitBarrierParticipant{
			{done: cancelled, limiter: first, weight: 1},
			{done: make(chan struct{}), limiter: second, weight: 1},
		}}
		second.m.Lock()
		errs := make(chan error, 1)
		go func() { errs <- admit(b) }()
		require.Eventually(t, func() bool { return first.limiter.Tokens() < 0.5 }, time.Second, time.Millisecond, "admission must reserve the first participant before reaching the second")
		close(cancelled)
		second.m.Unlock()
		require.ErrorIs(t, <-errs, ErrRateLimitBarrierRejected, "a participant cancelled after its reservation must reject the group")
		assert.InDelta(t, 1, first.limiter.Tokens(), 0.001, "a rejected group should give back every reservation")
		assert.InDelta(t, 1, second.limiter.Tokens(), 0.001, "a rejected group should give back every reservation")
	})
}

func TestNewWindowBudget(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		window   time.Duration
		requests int
		exp      *WindowBudget
	}{
		{5 * time.Second, 600, &WindowBudget{window: 5 * time.Second, requests: 600}},
		{0, 600, nil},
		{-time.Second, 600, nil},
		{5 * time.Second, 0, nil},
		{5 * time.Second, -1, nil},
	} {
		assert.Equalf(t, tc.exp, NewWindowBudget(tc.window, tc.requests), "NewWindowBudget should return a budget of %d requests in %s, or none", tc.requests, tc.window)
	}
}

func TestWindowBudgetPruneLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	b := &WindowBudget{window: time.Second, requests: 2, booked: []time.Time{now.Add(-2 * time.Second), now.Add(-time.Second), now.Add(-time.Millisecond), now.Add(time.Second)}}
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	b.pruneLocked(now)
	assert.Equal(t, []time.Time{now.Add(-time.Millisecond), now.Add(time.Second)}, b.booked, "pruneLocked should forget send times a whole window or more before now")

	b = NewWindowBudget(5*time.Second, 600)
	var capacity int
	for window := range 10 {
		at := now.Add(time.Duration(window) * 5 * time.Second)
		b.pruneLocked(at)
		for range 600 {
			b.bookLocked(at, 1)
		}
		if window == 0 {
			capacity = cap(b.booked)
		}
	}
	assert.Equal(t, 600, len(b.booked), "only the last window's send times should be held")
	assert.Equal(t, capacity, cap(b.booked), "later windows should reuse the memory of the first")
}

func TestWindowBudgetEarliestLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	at := func(seconds float64) time.Time { return now.Add(time.Duration(seconds * float64(time.Second))) }
	for _, tc := range []struct {
		name     string
		requests int
		booked   []float64
		n        int
		from     float64
		exp      float64
	}{
		{"an empty budget has room at once", 1, nil, 1, 0, 0},
		{"a budget holding fewer than its requests has room at once", 3, []float64{0, 0}, 1, 0, 0},
		{"a booking a window ahead leaves room before it", 1, []float64{20}, 1, 0, 0},
		{"a booking within a window holds a later moment back", 1, []float64{20}, 1, 15, 30},
		{"a moment a whole window after a booking has room", 1, []float64{0}, 1, 10, 10},
		{"a moment just inside the window has none", 1, []float64{0}, 1, 9.999, 10},
		{"a full window holds every moment it can reach", 2, []float64{5, 12}, 1, 3, 15},
		{"bookings spanning a whole window leave room between them", 2, []float64{0, 10}, 1, 5, 5},
		{"bookings for one moment fill it", 3, []float64{0, 0, 0}, 1, 0, 10},
		{"back to back windows push a moment past them all", 1, []float64{0, 10, 20}, 1, 0, 30},
		{"a window with room for all of several requests takes them at once", 10, []float64{0, 1, 2, 3, 4, 5, 6}, 3, 6, 6},
		{"a window with room for fewer than several requests holds them until it has room for all", 10, []float64{0, 1, 2, 3, 4, 5, 6, 7}, 3, 7, 10},
		{"as many requests as the budget admits need a window clear of bookings", 10, []float64{0}, 10, 5, 10},
		{"as many requests as the budget admits fit a whole window after a booking", 10, []float64{0, 25}, 10, 5, 10},
		{"as many requests as the budget admits wait past bookings within a window of each other", 3, []float64{0, 9, 18}, 3, 1, 28},
		{"several requests take the room between bookings a window apart", 3, []float64{0, 0, 20, 20}, 2, 5, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := &WindowBudget{window: 10 * time.Second, requests: tc.requests}
			for _, seconds := range tc.booked {
				b.booked = append(b.booked, at(seconds))
			}
			rateLimitReservationMu.Lock()
			defer rateLimitReservationMu.Unlock()
			assert.Equal(t, at(tc.exp), b.earliestLocked(at(tc.from), tc.n), "earliestLocked should return the earliest moment with room for every request")
		})
	}

	t.Run("matches a search of every moment", func(t *testing.T) {
		t.Parallel()
		random := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // A fixed seed keeps the random cases reproducible
		// Bookings and the answer fall on whole seconds, so checking windows that start on them covers every window
		hasRoom := func(booked []int, requests, n, window, x int) bool {
			for start := x - window + 1; start <= x; start++ {
				held := n
				for _, b := range booked {
					if b >= start && b < start+window {
						held++
					}
				}
				if held > requests {
					return false
				}
			}
			return true
		}
		for range 4000 {
			requests, window := 1+random.IntN(6), 10
			n := 1 + random.IntN(requests)
			booked := make([]int, random.IntN(16))
			for i := range booked {
				booked[i] = random.IntN(40)
			}
			slices.Sort(booked)
			from := random.IntN(40)
			exp := from
			for !hasRoom(booked, requests, n, window, exp) {
				exp++
			}
			b := &WindowBudget{window: time.Duration(window) * time.Second, requests: requests}
			for _, seconds := range booked {
				b.booked = append(b.booked, at(float64(seconds)))
			}
			rateLimitReservationMu.Lock()
			got := b.earliestLocked(at(float64(from)), n)
			rateLimitReservationMu.Unlock()
			require.Equalf(t, at(float64(exp)), got, "earliestLocked must match a search of every moment for %d of %d requests booked at %v from %d", n, requests, booked, from)
		}
	})
}

func TestWindowBudgetBookLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	b := NewWindowBudget(time.Second, 10)
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	for _, tc := range []struct {
		offset time.Duration
		n      int
	}{{2 * time.Second, 1}, {0, 1}, {time.Second, 1}, {time.Second, 3}, {500 * time.Millisecond, 2}} {
		b.bookLocked(now.Add(tc.offset), tc.n)
	}
	exp := []time.Time{now, now.Add(500 * time.Millisecond), now.Add(500 * time.Millisecond), now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(2 * time.Second)}
	assert.Equal(t, exp, b.booked, "bookLocked should keep send times in order, once for each request")
}

func TestWindowBudgetCancelLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	b := &WindowBudget{window: time.Second, requests: 10, booked: []time.Time{now, now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(2 * time.Second), now.Add(2 * time.Second)}}
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	b.cancelLocked(now.Add(time.Second), 1)
	b.cancelLocked(now.Add(time.Second), 2)
	b.cancelLocked(now.Add(500*time.Millisecond), 1)
	b.cancelLocked(now.Add(-time.Second), 1)
	b.cancelLocked(now, 3)
	assert.Equal(t, []time.Time{now.Add(time.Second), now.Add(2 * time.Second), now.Add(2 * time.Second)}, b.booked, "cancelLocked should remove as many bookings of a time as it gives back, as many as there are, and nothing for a time not booked")
	(&WindowBudget{window: time.Second, requests: 1}).cancelLocked(now, 1)
}

func TestFirstAfter(t *testing.T) {
	t.Parallel()
	now := time.Now().Round(0)
	times := []time.Time{now, now.Add(time.Second), now.Add(time.Second), now.Add(2 * time.Second)}
	for _, tc := range []struct {
		at  time.Time
		exp int
	}{
		{now.Add(-time.Second), 0},
		{now, 1},
		{now.Add(time.Second), 3},
		{now.Add(2 * time.Second), 4},
		{now.Add(3 * time.Second), 4},
	} {
		assert.Equalf(t, tc.exp, firstAfter(times, tc.at), "firstAfter should return the index of the first time after %s", tc.at.Sub(now))
	}
	assert.Zero(t, firstAfter(nil, now), "firstAfter should return 0 for no times")
}

// BenchmarkRateLimitWithSharedSaturated times a request planned, booked and given back on a budget booked full from a
// window before now to three windows ahead, which it has to search past to find room
func BenchmarkRateLimitWithSharedSaturated(b *testing.B) {
	now := time.Now()
	budget := NewWindowBudget(5*time.Second, 600)
	spacing := 5 * time.Second / 600
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()
	for i := range 2400 {
		budget.bookLocked(now.Add(-5*time.Second+time.Duration(i)*spacing), 1)
	}
	endpoint := sharedOnly(budget)
	b.ReportAllocs()
	for b.Loop() {
		delay, plan := endpoint.planLocked(now, 1)
		if delay < 14*time.Second {
			b.Fatalf("a request must search past the bookings to find room, found it after %s", delay)
		}
		plan.reserveLocked(now).cancelLocked(now)
	}
}
