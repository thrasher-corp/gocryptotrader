package request

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestRateLimiterWithWeightApplyMultipleRateLimits(t *testing.T) {
	t.Parallel()

	t.Run("multiple limits", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			short := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			long := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{
				{Limiter: short, Scope: "short"},
				{Limiter: long, Scope: "long"},
			}
			endpoint := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			start := time.Now()
			err := endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits)
			elapsed := time.Since(start)
			require.NoError(t, err, "rate limit set must not error")
			assert.Equal(t, 300*time.Millisecond, elapsed, "rate limit set should wait for the longest limiter only")

			err = endpoint.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{Limiter: NewRateLimitWithWeight(time.Second, 1, 0)}})
			assert.ErrorIs(t, err, errInvalidWeight, "zero weight should return errInvalidWeight")

			err = endpoint.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{WeightOverride: 1}})
			assert.ErrorContains(t, err, "nil pointer: *request.RateLimiterWithWeight")
		})
	})

	t.Run("delay not allowed", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			short := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			long := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{
				{Limiter: short, WeightOverride: 1, Scope: "short"},
				{Limiter: long, WeightOverride: 1, Scope: "long"},
			}
			endpoint := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			err := endpoint.applyMultipleRateLimits(WithDelayNotAllowed(t.Context()), 0, additionalRateLimits)
			require.ErrorIs(t, err, ErrDelayNotAllowed, "delayed reservation must return ErrDelayNotAllowed")
			assert.ErrorContains(t, err, `rate-limit scope "long"`, "delay rejection should identify the limiting scope")

			start := time.Now()
			err = endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits)
			elapsed := time.Since(start)
			require.NoError(t, err, "cancelled reservations must be usable again")
			assert.Equal(t, 300*time.Millisecond, elapsed, "cancelled reservation should not add another delay window")
		})
	})

	t.Run("delay not allowed without delay", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			extra := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			err := endpoint.applyMultipleRateLimits(
				WithDelayNotAllowed(t.Context()),
				0,
				[]AdditionalRateLimit{{Limiter: extra, Scope: "extra"}})
			require.NoError(t, err, "immediate reservations must not be rejected when delay is not allowed")
		})
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			extra := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{{Limiter: extra, Scope: "extra"}}
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()
			err := endpoint.applyMultipleRateLimits(ctx, 0, additionalRateLimits)
			require.ErrorIs(t, err, context.DeadlineExceeded, "reservation beyond the deadline must return context deadline exceeded")
			assert.ErrorContains(t, err, `scope "extra"`, "deadline error should identify the limiting scope")
		})
	})

	t.Run("verbose delay", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			extra := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{{Limiter: extra, Scope: "extra"}}
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			start := time.Now()
			require.NoError(t, endpoint.applyMultipleRateLimits(WithVerbose(t.Context()), 0, additionalRateLimits), "verbose delayed reservation must not error")
			assert.Equal(t, 200*time.Millisecond, time.Since(start), "verbose reservation should wait for the limiting scope")
		})
	})

	t.Run("additional limits", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			extra := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{{Limiter: extra, WeightOverride: 1}}
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			start := time.Now()
			err := endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits)
			elapsed := time.Since(start)
			require.NoError(t, err, "additional rate limit must not error")
			assert.Equal(t, 300*time.Millisecond, elapsed, "endpoint and additional rate limits should wait for the longest limiter only")

			err = endpoint.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{Limiter: NewRateLimitWithWeight(time.Second, 1, 0)}})
			assert.ErrorIs(t, err, errInvalidWeight, "zero additional weight should return errInvalidWeight")

			err = endpoint.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{WeightOverride: 1}})
			assert.ErrorContains(t, err, "nil pointer: *request.RateLimiterWithWeight", "nil additional limiter should return a nil guard error")
		})
	})

	t.Run("nested context limits", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			short := NewRateLimitWithWeight(200*time.Millisecond, 1, 1)
			long := NewRateLimitWithWeight(300*time.Millisecond, 1, 1)
			longest := NewRateLimitWithWeight(400*time.Millisecond, 1, 1)
			ctx := WithAdditionalRateLimits(t.Context(), AdditionalRateLimit{Limiter: short})
			ctx = WithAdditionalRateLimits(ctx, AdditionalRateLimit{Limiter: long})
			ctx = WithAdditionalRateLimits(ctx, AdditionalRateLimit{Limiter: longest})
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimitsFromContext(ctx)), "first reservation must not error")

			start := time.Now()
			err := endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimitsFromContext(ctx))
			elapsed := time.Since(start)
			require.NoError(t, err, "context rate limits must not error")
			assert.Equal(t, 400*time.Millisecond, elapsed, "context and explicit rate limits should wait for the longest limiter only")
		})
	})

	t.Run("weight override", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			weightedEndpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			weightedExtra := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			start := time.Now()
			err := weightedEndpoint.applyMultipleRateLimits(t.Context(), 3, []AdditionalRateLimit{{Limiter: weightedExtra, WeightOverride: 1}})
			elapsed := time.Since(start)
			require.NoError(t, err, "explicit endpoint weight must not error")
			assert.Equal(t, 200*time.Millisecond, elapsed, "explicit endpoint weight should override endpoint default weight")
		})
	})

	t.Run("duplicate limiter", func(t *testing.T) {
		t.Parallel()

		endpoint := NewRateLimitWithWeight(0, 0, 1)
		extra := NewRateLimitWithWeight(0, 0, 1)
		additionalRateLimits := []AdditionalRateLimit{
			{Limiter: extra, Scope: "extra"},
			{Limiter: extra, Scope: "extra"},
		}
		require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "duplicate limiter reservations must not deadlock or error")
	})

	t.Run("overlapping limiter sets", func(t *testing.T) {
		t.Parallel()

		first := NewRateLimitWithWeight(0, 0, 1)
		second := NewRateLimitWithWeight(0, 0, 1)
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			results <- first.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{Limiter: second}})
		}()
		go func() {
			<-start
			results <- second.applyMultipleRateLimits(t.Context(), 0, []AdditionalRateLimit{{Limiter: first}})
		}()
		close(start)
		for range 2 {
			select {
			case err := <-results:
				require.NoError(t, err, "overlapping limiter sets must not deadlock or error")
			case <-time.After(time.Second):
				require.Fail(t, "overlapping limiter sets must not deadlock")
			}
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			extra := NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
			additionalRateLimits := []AdditionalRateLimit{{Limiter: extra}}
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "first reservation must not error")

			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.ErrorIs(t, endpoint.applyMultipleRateLimits(ctx, 0, additionalRateLimits), context.Canceled, "cancelled wait must return context cancellation")

			start := time.Now()
			require.NoError(t, endpoint.applyMultipleRateLimits(t.Context(), 0, additionalRateLimits), "reservation after cancellation must not error")
			assert.Equal(t, 100*time.Millisecond, time.Since(start), "cancelled reservations should be released")
		})
	})

	t.Run("concurrent rejection remains atomic", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			endpoint := NewRateLimitWithWeight(time.Second, 10, 1)
			extra := NewRateLimitWithWeight(time.Second, 10, 1)
			ctx := WithAdditionalRateLimits(t.Context(), AdditionalRateLimit{Limiter: extra, Scope: "extra"})
			start := time.Now()
			errs := common.ErrorCollector{}
			for i := range 10 {
				requestContext := ctx
				if i%2 == 0 {
					requestContext = WithDelayNotAllowed(requestContext)
				}
				errs.Go(func() error { return endpoint.RateLimit(requestContext) })
			}

			require.ErrorContains(t, errs.Collect(), "delay not allowed", "concurrent rejected reservations must return the expected error")
			assert.Less(t, time.Since(start), 600*time.Millisecond, "rejected reservations should not delay accepted requests")
		})
	})
}

func TestNewRateLimiterLockSet(t *testing.T) {
	t.Parallel()

	first := NewRateLimitWithWeight(time.Second, 1, 1)
	second := NewRateLimitWithWeight(time.Second, 1, 1)
	testCases := []struct {
		name           string
		rateLimits     []AdditionalRateLimit
		expected       rateLimiterLockSet
		expectedErrMsg string
	}{
		{
			name:     "empty",
			expected: rateLimiterLockSet{},
		},
		{
			name: "deduplicated and ordered",
			rateLimits: []AdditionalRateLimit{
				{Limiter: second},
				{Limiter: first},
				{Limiter: second},
			},
			expected: rateLimiterLockSet{first, second},
		},
		{
			name:           "nil limiter",
			rateLimits:     []AdditionalRateLimit{{}},
			expectedErrMsg: "nil pointer: *request.RateLimiterWithWeight",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			lockSet, err := newRateLimiterLockSet(testCase.rateLimits)
			if testCase.expectedErrMsg != "" {
				require.ErrorContains(t, err, testCase.expectedErrMsg, "invalid limiter must return the expected error")
				assert.Nil(t, lockSet, "invalid limiter should not return a lock set")
				return
			}
			require.NoError(t, err, "valid limiters must create a lock set")
			assert.Equal(t, testCase.expected, lockSet, "lock set should contain each limiter once in stable order")
		})
	}
}

func TestRateLimiterLockSetLock(t *testing.T) {
	t.Parallel()

	first := NewRateLimitWithWeight(time.Second, 1, 1)
	second := NewRateLimitWithWeight(time.Second, 1, 1)
	lockSet := rateLimiterLockSet{first, second}
	lockSet.lock()
	defer lockSet.unlock()
	assert.False(t, first.m.TryLock(), "lock should acquire the first limiter")
	assert.False(t, second.m.TryLock(), "lock should acquire the second limiter")
}

func TestRateLimiterLockSetUnlock(t *testing.T) {
	t.Parallel()

	first := NewRateLimitWithWeight(time.Second, 1, 1)
	second := NewRateLimitWithWeight(time.Second, 1, 1)
	lockSet := rateLimiterLockSet{first, second}
	lockSet.lock()
	lockSet.unlock()
	require.True(t, first.m.TryLock(), "unlock must release the first limiter")
	first.m.Unlock()
	require.True(t, second.m.TryLock(), "unlock must release the second limiter")
	second.m.Unlock()
}
