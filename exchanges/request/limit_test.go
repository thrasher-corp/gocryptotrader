package request

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

func TestNewRateLimit(t *testing.T) {
	t.Parallel()

	require.Equal(t, rate.Inf, NewRateLimit(0, 0).Limit())
	require.Equal(t, 1, NewRateLimit(0, 0).Burst())
	require.Equal(t, 0.5, float64(NewRateLimit(time.Second*2, 1).Limit()))
}

func TestNewRateLimitWithWeight(t *testing.T) {
	t.Parallel()
	r := NewRateLimitWithWeight(time.Second*10, 5, 1)
	require.Equal(t, 0.5, float64(r.endpoint.Limit()))

	// Ensures rate limiting factor is the same
	r = NewRateLimitWithWeight(time.Second*2, 1, 1)
	require.Equal(t, 0.5, float64(r.endpoint.Limit()))

	// Test for open rate limit
	r = NewRateLimitWithWeight(time.Second*2, 0, 1)
	require.Equal(t, rate.Inf, r.endpoint.Limit())

	r = NewRateLimitWithWeight(0, 69, 1)
	require.Equal(t, rate.Inf, r.endpoint.Limit())
}

func TestNewWeightedRateLimitByDuration(t *testing.T) {
	t.Parallel()
	r := NewWeightedRateLimitByDuration(time.Second * 2)
	require.Equal(t, 0.5, float64(r.endpoint.Limit()))

	r = NewWeightedRateLimitByDuration(time.Second * 4)
	require.Equal(t, 0.25, float64(r.endpoint.Limit()))
}

func TestGetRateLimiterWithWeight(t *testing.T) {
	t.Parallel()
	r := GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1)
	require.Equal(t, rate.Inf, r.endpoint.Limit())
	require.Equal(t, uint8(1), r.weight)

	r = GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1, NewRateLimit(time.Second*2, 1))
	require.Equal(t, rate.Inf, r.endpoint.Limit())
	require.Equal(t, uint8(1), r.weight)
	require.Equal(t, 0.5, float64(r.global.Limit()))
}

func TestRateLimit(t *testing.T) {
	t.Parallel()

	err := RateLimit(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = RateLimit(t.Context(), &RateLimiterWithWeight{}, false)
	require.ErrorIs(t, err, errInvalidWeightCount)

	ctxWDL, cancelDL := context.WithDeadline(t.Context(), time.Now())
	defer cancelDL()
	r := GetRateLimiterWithWeight(NewRateLimit(time.Second, 1), 2)
	err = RateLimit(ctxWDL, r, false)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	ctxWCancel, cancel := context.WithCancel(t.Context())
	cancel()
	err = RateLimit(ctxWCancel, r, false)
	require.ErrorIs(t, err, context.Canceled)

	r = GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1)
	err = RateLimit(t.Context(), r, false)
	require.NoError(t, err, "must not error on fast path")

	r.global = NewRateLimit(time.Millisecond, 1)
	err = RateLimit(t.Context(), r, true)
	require.NoError(t, err, "must not error on global reservation fast path")

	err = RateLimit(t.Context(), r, true)
	require.NoError(t, err, "must not error on global reservation with time delay")
}

func TestDescribeLimiter(t *testing.T) {
	t.Parallel()
	require.Equal(t, "nil", describeLimiter(nil))
	require.Equal(t, "limit=1.000/s burst=1", describeLimiter(NewRateLimit(time.Second, 1)))
}

func TestDisableRateLimiter(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, (*Requester)(nil).DisableRateLimiter(), ErrRequestSystemIsNil)
	require.ErrorIs(t, (&Requester{disableRateLimiter: 1}).DisableRateLimiter(), ErrRateLimiterAlreadyDisabled)
	require.NoError(t, (&Requester{}).DisableRateLimiter())
}

func TestEnableRateLimiter(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, (*Requester)(nil).EnableRateLimiter(), ErrRequestSystemIsNil)
	require.ErrorIs(t, (&Requester{}).EnableRateLimiter(), ErrRateLimiterAlreadyEnabled)
	require.NoError(t, (&Requester{disableRateLimiter: 1}).EnableRateLimiter())
}

func TestInitiateRateLimit(t *testing.T) {
	t.Parallel()

	r := &Requester{disableRateLimiter: 1}
	require.NoError(t, r.InitiateRateLimit(t.Context(), Unset, false), "must not error when rate limiter is disabled")
	r.disableRateLimiter = 0
	require.ErrorIs(t, r.InitiateRateLimit(t.Context(), Unset, false), errLimiterSystemIsNil)
	r.limiter = NewBasicRateLimit(time.Second, 1, 1)
	require.ErrorIs(t, r.InitiateRateLimit(t.Context(), 1337, false), common.ErrNilPointer)
	require.NoError(t, r.InitiateRateLimit(t.Context(), UnAuth, false), "must not error on valid rate limit")
}
