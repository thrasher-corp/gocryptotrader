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

	errRequestSystemShutdown    = errors.New("request system has shutdown")
	errLimiterSystemIsNil       = errors.New("limiter system is nil")
	errInvalidTokenCount        = errors.New("invalid token count must equal or greater than 1")
	errSpecificRateLimiterIsNil = errors.New("specific rate limiter is nil")
)

// Const here define individual functionality sub types for rate limiting
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// BasicLimit denotes basic rate limit that implements the Limiter interface
// does not need to set endpoint functionality.
type BasicLimit struct {
	r *rate.Limiter
}

// Limit executes a single rate limit set by NewRateLimit
func (b *BasicLimit) Limit(ctx context.Context, _ EndpointLimit) (*rate.Limiter, int, error) {
	return b.r, 1, nil
}

// EndpointLimit defines individual endpoint rate limits that are set when
// New is called.
type EndpointLimit uint16

// Limiter interface groups rate limit functionality defined in the REST
// wrapper for extended rate limiting configuration i.e. Shells of rate
// limits with a global rate for sub rates.
type Limiter interface {
	Limit(context.Context, EndpointLimit) (*rate.Limiter, int, error)
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

// NewBasicRateLimit returns an object that implements the limiter interface
// for basic rate limit
func NewBasicRateLimit(interval time.Duration, actions int) Limiter {
	return &BasicLimit{NewRateLimit(interval, actions)}
}

// InitiateRateLimit sleeps for designated end point rate limits
func (r *Requester) initiateRateLimit(ctx context.Context, e EndpointLimit) error {
	if atomic.LoadInt32(&r.disableRateLimiter) == 1 {
		return nil
	}

	if r.limiter == nil {
		return fmt.Errorf("cannot rate limit request %w", errLimiterSystemIsNil)
	}

	rateLimiter, tokens, err := r.limiter.Limit(ctx, e)
	if err != nil {
		return err
	}

	if tokens <= 0 {
		return fmt.Errorf("cannot rate limit request %w", errInvalidTokenCount)
	}

	if rateLimiter == nil {
		return fmt.Errorf("cannot rate limit request %w", errSpecificRateLimiterIsNil)
	}

	var finalDelay time.Duration
	var reservations = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reservations[i] = rateLimiter.Reserve()
		finalDelay = reservations[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		// Cancel all potential reservations to free up rate limiter if deadline
		// is exceeded.
		for x := range reservations {
			reservations[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	tick := time.NewTimer(finalDelay)
	defer tick.Stop()
	select {
	case <-tick.C:
		return nil
	case <-ctx.Done():
		for x := range reservations {
			reservations[x].Cancel()
		}
		return ctx.Err()
	case <-r.shutdown:
		for x := range reservations {
			reservations[x].Cancel()
		}
		return fmt.Errorf("cannot rate limit request %w", errRequestSystemShutdown)
	}
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
