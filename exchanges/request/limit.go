package request

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

// Rate limiting errors.
var (
	ErrRateLimiterAlreadyDisabled = errors.New("rate limiter already disabled")
	ErrRateLimiterAlreadyEnabled  = errors.New("rate limiter already enabled")
	ErrDelayNotAllowed            = errors.New("delay not allowed")

	errInvalidWeight = errors.New("weight must be equal-or-greater than 1")
)

// RateLimitNotRequired is a no-op rate limiter.
var RateLimitNotRequired *RateLimiterWithWeight

// Const here define individual functionality sub types for rate limiting.
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// EndpointLimit defines individual endpoint rate limits.
type EndpointLimit uint16

// Weight defines the number of reservations to be used. This is a generalised weight for rate limiting.
// e.g. n weight = n request. i.e. 50 Weight = 50 requests.
type Weight uint8

// RateLimitDefinitions is a map of endpoint limits to rate limiters.
type RateLimitDefinitions map[any]*RateLimiterWithWeight

// RateLimiterWithWeight is a rate limiter coupled with a weight which refers to the number or weighting of the request.
// This is used to define the rate limit for a specific endpoint.
type RateLimiterWithWeight struct {
	limiter *rate.Limiter
	weight  Weight
	m       sync.Mutex
}

// NewRateLimit creates a new RateLimit based of time interval and how many actions allowed and breaks it down to an
// actions-per-second basis -- Burst rate is kept as one as this is not supported for out-bound requests.
func NewRateLimit(interval time.Duration, actions int) *rate.Limiter {
	if actions <= 0 || interval <= 0 {
		// Returns an un-restricted rate limiter
		return rate.NewLimiter(rate.Inf, 1)
	}

	i := 1 / interval.Seconds()
	rps := i * float64(actions)
	return rate.NewLimiter(rate.Limit(rps), 1)
}

// NewRateLimitWithWeight creates a new RateLimit based of time interval and how many actions allowed. This also has a
// weight count which refers to the number or weighting of the request. This is used to define the rate limit for a
// specific endpoint.
func NewRateLimitWithWeight(interval time.Duration, actions int, weight Weight) *RateLimiterWithWeight {
	return GetRateLimiterWithWeight(NewRateLimit(interval, actions), weight)
}

// NewWeightedRateLimitByDuration creates a new RateLimit based of time interval. This equates to 1 action per interval.
// The weight is set to 1.
func NewWeightedRateLimitByDuration(interval time.Duration) *RateLimiterWithWeight {
	return NewRateLimitWithWeight(interval, 1, 1)
}

// GetRateLimiterWithWeight couples a rate limiter with a weight count into an accepted defined rate limiter with weight
// struct.
func GetRateLimiterWithWeight(l *rate.Limiter, weight Weight) *RateLimiterWithWeight {
	return &RateLimiterWithWeight{limiter: l, weight: weight}
}

// NewBasicRateLimit returns an object that implements the limiter interface for basic rate limit.
func NewBasicRateLimit(interval time.Duration, actions int, weight Weight) RateLimitDefinitions {
	rl := NewRateLimitWithWeight(interval, actions, weight)
	return RateLimitDefinitions{Unset: rl, Auth: rl, UnAuth: rl}
}

// InitiateRateLimit sleeps for designated end point rate limits.
func (r *Requester) InitiateRateLimit(ctx context.Context, e EndpointLimit) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if atomic.LoadInt32(&r.disableRateLimiter) == 1 {
		return nil
	}
	if err := common.NilGuard(r.limiter); err != nil {
		return err
	}
	if err := r.limiter[e].RateLimit(ctx); err != nil {
		return fmt.Errorf("cannot rate limit request %w for endpoint %d", err, e)
	}
	return nil
}

// GetRateLimiterDefinitions returns the rate limiter definitions for the requester.
func (r *Requester) GetRateLimiterDefinitions() RateLimitDefinitions {
	if r == nil {
		return nil
	}
	return r.limiter
}

// RateLimit throttles a request based on weight, delaying the request.
// Errors if no delay is permitted via the context and a delay is required.
func (r *RateLimiterWithWeight) RateLimit(ctx context.Context) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}

	r.m.Lock()
	if r.weight == 0 {
		r.m.Unlock()
		return errInvalidWeight
	}

	tn := time.Now()
	reserved := make([]*rate.Reservation, 0, r.weight)
	for range r.weight {
		// This avoids needing burst capacity in the limiter, which would otherwise allow the rate limit to be exceeded over short periods
		reserved = append(reserved, r.limiter.ReserveN(tn, 1))
	}
	finalDelay := reserved[len(reserved)-1].DelayFrom(tn)

	if finalDelay == 0 {
		r.m.Unlock()
		return nil
	}

	if hasDelayNotAllowed(ctx) {
		cancelAll(reserved, tn)
		r.m.Unlock()
		return ErrDelayNotAllowed
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(tn.Add(finalDelay)) {
		cancelAll(reserved, tn)
		r.m.Unlock()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w", finalDelay, context.DeadlineExceeded)
	}
	r.m.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(finalDelay):
		return nil
	}
}

// cancelAll cancels all reservations at a specific time.
// Does not provide locking protection, so callers can maintain a single lock throughout.
func cancelAll(reservations []*rate.Reservation, at time.Time) {
	slices.Reverse(reservations) // cancel in reverse order for correct token reimbursement
	for _, r := range reservations {
		r.CancelAt(at)
	}
}

// DisableRateLimiter disables the rate limiting system for the exchange.
func (r *Requester) DisableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 0, 1) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyDisabled)
	}
	return nil
}

// EnableRateLimiter enables the rate limiting system for the exchange.
func (r *Requester) EnableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !atomic.CompareAndSwapInt32(&r.disableRateLimiter, 1, 0) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyEnabled)
	}
	return nil
}
