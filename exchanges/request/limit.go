package request

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Defines rate limiting errors
var (
	ErrRateLimiterAlreadyDisabled = errors.New("rate limiter already disabled")
	ErrRateLimiterAlreadyEnabled  = errors.New("rate limiter already enabled")

	errLimiterSystemIsNil       = errors.New("limiter system is nil")
	errInvalidWeightCount       = errors.New("invalid weight count must equal or greater than 1")
	errSpecificRateLimiterIsNil = errors.New("specific rate limiter is nil")
)

// Const here define individual functionality sub types for rate limiting
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// EndpointLimit defines individual endpoint rate limits that are set when
// New is called.
type EndpointLimit uint16

// Weight defines the number of reservations to be used. This is a generalised
// weight for rate limiting. e.g. n weight = n request. i.e. 50 Weight = 50
// requests.
type Weight uint8

// RateLimitDefinitions is a map of endpoint limits to rate limiters
type RateLimitDefinitions map[any]*RateLimiterWithWeight

// RateLimiterWithWeight is a rate limiter coupled with a weight count which
// refers to the number or weighting of the request. This is used to define
// the rate limit for a specific endpoint.
type RateLimiterWithWeight struct {
	*rate.Limiter
	Weight
}

// Reservations is a slice of rate reservations
type Reservations []*rate.Reservation

// CancelAll cancels all potential reservations to free up rate limiter for
// context cancellations and deadline exceeded cases.
func (r Reservations) CancelAll() {
	for x := range r {
		r[x].Cancel()
	}
}

// NewRateLimit creates a new RateLimit based of time interval and how many
// actions allowed and breaks it down to an actions-per-second basis -- Burst
// rate is kept as one as this is not supported for out-bound requests.
func NewRateLimit(interval time.Duration, actions int) *rate.Limiter {
	if actions <= 0 || interval <= 0 {
		// Returns an un-restricted rate limiter
		return rate.NewLimiter(rate.Inf, 1)
	}

	i := 1 / interval.Seconds()
	rps := i * float64(actions)
	return rate.NewLimiter(rate.Limit(rps), 1)
}

// NewRateLimitWithWeight creates a new RateLimit based of time interval and how
// many actions allowed. This also has a weight count which refers to the number
// or weighting of the request. This is used to define the rate limit for a
// specific endpoint.
func NewRateLimitWithWeight(interval time.Duration, actions int, weight Weight) *RateLimiterWithWeight {
	return GetRateLimiterWithWeight(NewRateLimit(interval, actions), weight)
}

// NewWeightedRateLimitByDuration creates a new RateLimit based of time
// interval. This equates to 1 action per interval. The weight is set to 1.
func NewWeightedRateLimitByDuration(interval time.Duration) *RateLimiterWithWeight {
	return NewRateLimitWithWeight(interval, 1, 1)
}

// GetRateLimiterWithWeight couples a rate limiter with a weight count into an
// accepted defined rate limiter with weight struct
func GetRateLimiterWithWeight(l *rate.Limiter, weight Weight) *RateLimiterWithWeight {
	return &RateLimiterWithWeight{l, weight}
}

// NewBasicRateLimit returns an object that implements the limiter interface
// for basic rate limit
func NewBasicRateLimit(interval time.Duration, actions int, weight Weight) RateLimitDefinitions {
	rl := NewRateLimitWithWeight(interval, actions, weight)
	return RateLimitDefinitions{Unset: rl, Auth: rl, UnAuth: rl}
}

// InitiateRateLimit sleeps for designated end point rate limits
func (r *Requester) InitiateRateLimit(ctx context.Context, e EndpointLimit) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if atomic.LoadInt32(&r.disableRateLimiter) == 1 {
		return nil
	}
	if r.limiter == nil {
		return fmt.Errorf("cannot rate limit request %w", errLimiterSystemIsNil)
	}

	rateLimiter := r.limiter[e]

	err := RateLimit(ctx, rateLimiter)
	if err != nil {
		return fmt.Errorf("cannot rate limit request %w for endpoint %d", err, e)
	}

	return nil
}

// GetRateLimiterDefinitions returns the rate limiter definitions for the
// requester
func (r *Requester) GetRateLimiterDefinitions() RateLimitDefinitions {
	if r == nil {
		return nil
	}
	return r.limiter
}

// RateLimit is a function that will rate limit a request based on the rate
// limiter provided. It will return an error if the context is cancelled or
// deadline exceeded.
func RateLimit(ctx context.Context, rateLimiter *RateLimiterWithWeight) error {
	if rateLimiter == nil {
		return errSpecificRateLimiterIsNil
	}

	if rateLimiter.Weight <= 0 {
		return errInvalidWeightCount
	}

	var finalDelay time.Duration
	reservations := make(Reservations, rateLimiter.Weight)
	for i := Weight(0); i < rateLimiter.Weight; i++ {
		// Consume 1 weight at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reservations[i] = rateLimiter.Reserve()
		finalDelay = reservations[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		reservations.CancelAll()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	tick := time.NewTimer(finalDelay)
	select {
	case <-tick.C:
		return nil
	case <-ctx.Done():
		tick.Stop()
		reservations.CancelAll()
		return ctx.Err()
	}
	// TODO: Shutdown case
}

// DisableRateLimiter disables the rate limiting system for the exchange
func (r *Requester) DisableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 0, 1) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyDisabled)
	}
	return nil
}

// EnableRateLimiter enables the rate limiting system for the exchange
func (r *Requester) EnableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 1, 0) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyEnabled)
	}
	return nil
}
