package request

import (
	"context"
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

func TestRateLimitBarrierZeroWeightReleasesParkedPeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 3)
		require.NoError(t, err, "barrier contexts must be created")
		defer AbortRateLimitBarrier(contexts[2])
		peer := NewRateLimitWithWeight(time.Hour, 1, 1)
		errors := make(chan error, 1)
		go func() { errors <- peer.RateLimit(contexts[0]) }()
		synctest.Wait()
		invalid := NewRateLimitWithWeight(time.Hour, 1, 0)
		require.ErrorIs(t, invalid.RateLimit(contexts[1]), errInvalidWeight, "zero weight must be rejected")
		synctest.Wait()
		select {
		case err := <-errors:
			assert.ErrorIs(t, err, ErrRateLimitBarrierRejected, "parked peer should observe rejection before the last arrival")
			assert.NotErrorIs(t, err, ErrDelayNotAllowed, "invalid peer should not report a capacity delay")
		default:
			t.Error("zero weight should release the parked peer without a final arrival")
		}
		assert.Equal(t, float64(1), peer.limiter.Tokens(), "rejected group should leave capacity untouched")
	})
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
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	go func() { errs <- r.InitiateRateLimit(contexts[0], Unset) }()
	require.Never(t, func() bool { return len(errs) != 0 }, 10*time.Millisecond, time.Millisecond,
		"disabled limiter participant must wait for its peer")
	go func() { errs <- r.InitiateRateLimit(contexts[1], Unset) }()
	require.NoError(t, <-errs, "disabled limiter barrier participant must not error")
	require.NoError(t, <-errs, "disabled limiter barrier participant must not error")

	atomic.StoreInt32(&r.disableRateLimiter, 0)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.ErrorContains(t, err, "nil pointer: request.RateLimitDefinitions", "should return correct error when limiter is nil")

	r.limiter = NewBasicRateLimit(time.Second, 10, 1)
	err = r.InitiateRateLimit(t.Context(), Unset)
	assert.NoError(t, err, "should not error on valid rate limit initiation")
}
