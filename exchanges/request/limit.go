package request

import (
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
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
func (b *BasicLimit) Limit(_ EndpointLimit) error {
	time.Sleep(b.r.Reserve().Delay())
	return nil
}

// EndpointLimit defines individual endpoint rate limits that are set when
// New is called.
type EndpointLimit int

// Limiter interface groups rate limit functionality defined in the REST
// wrapper for extended rate limiting configuration i.e. Shells of rate
// limits with a global rate for sub rates.
type Limiter interface {
	Limit(EndpointLimit) error
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
func (r *Requester) InitiateRateLimit(e EndpointLimit) error {
	if atomic.LoadInt32(&r.disableRateLimiter) == 1 {
		return nil
	}

	if r.limiter != nil {
		return r.limiter.Limit(e)
	}

	return nil
}

// DisableRateLimiter disables the rate limiting system for the exchange
func (r *Requester) DisableRateLimiter() error {
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 0, 1) {
		return errors.New("rate limiter already disabled")
	}
	return nil
}

// EnableRateLimiter enables the rate limiting system for the exchange
func (r *Requester) EnableRateLimiter() error {
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 1, 0) {
		return errors.New("rate limiter already enabled")
	}
	return nil
}
