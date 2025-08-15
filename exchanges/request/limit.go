package request

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/time/rate"
)

// Public error vars
var (
	ErrRateLimiterAlreadyDisabled = errors.New("rate limiter already disabled")
	ErrRateLimiterAlreadyEnabled  = errors.New("rate limiter already enabled")
)

var (
	errLimiterSystemIsNil = errors.New("limiter system is nil")
	errInvalidWeightCount = errors.New("invalid weight count must equal or greater than 1")
)

// Const here define individual functionality sub types for rate limiting
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// EndpointLimit defines individual endpoint rate limits that are set when new is called
type EndpointLimit uint16

// RateLimitDefinitions is a map of endpoint limits to rate limiters
type RateLimitDefinitions map[any]*RateLimiterWithWeight

// RateLimiterWithWeight is a rate limiter coupled with a weight count which refers to the number or weighting of the
// request. This is used to define the rate limit for a specific endpoint.
type RateLimiterWithWeight struct {
	endpoint *rate.Limiter
	global   *rate.Limiter
	weight   uint8
}

// Reservations is a slice of rate reservations
type Reservations []*rate.Reservation

// CancelAll cancels all potential reservations to free up rate limiter for context cancellations and deadline exceeded cases.
func (r Reservations) CancelAll() {
	for x := range r {
		r[x].Cancel()
	}
}

// NewRateLimit creates a new RateLimit based of time interval and how many actions allowed and breaks it down to an
// actions-per-second basis -- Burst rate is kept as one as this is not supported for out-bound requests.
func NewRateLimit(interval time.Duration, actions uint64) *rate.Limiter {
	if interval <= 0 || actions == 0 {
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
func NewRateLimitWithWeight(interval time.Duration, actions uint64, weight uint8, optionalGlobal ...*rate.Limiter) *RateLimiterWithWeight {
	return GetRateLimiterWithWeight(NewRateLimit(interval, actions), weight, optionalGlobal...)
}

// NewWeightedRateLimitByDuration creates a new RateLimit based of time interval. This equates to 1 action per interval.
// The weight is set to 1.
func NewWeightedRateLimitByDuration(interval time.Duration, optionalGlobal ...*rate.Limiter) *RateLimiterWithWeight {
	return NewRateLimitWithWeight(interval, 1, 1, optionalGlobal...)
}

// GetRateLimiterWithWeight couples a rate limiter with a weight count into an accepted defined rate limiter with weight struct
func GetRateLimiterWithWeight(l *rate.Limiter, weight uint8, optionalGlobal ...*rate.Limiter) *RateLimiterWithWeight {
	var global *rate.Limiter
	if len(optionalGlobal) > 0 {
		global = optionalGlobal[0]
	}
	return &RateLimiterWithWeight{l, global, weight}
}

// NewBasicRateLimit returns an object that implements the limiter interface for a basic rate limit
func NewBasicRateLimit(interval time.Duration, actions uint64, weight uint8) RateLimitDefinitions {
	rl := NewRateLimitWithWeight(interval, actions, weight)
	return RateLimitDefinitions{Unset: rl, Auth: rl, UnAuth: rl}
}

// RateLimit is a function that will rate limit a request based on the rate limiter provided. It will return an error if
// the context is cancelled or the deadline exceeded.
func RateLimit(ctx context.Context, rateLimiter *RateLimiterWithWeight, verbose bool) error {
	if err := common.NilGuard(rateLimiter); err != nil {
		return err
	}

	if rateLimiter.weight == 0 {
		return errInvalidWeightCount
	}

	var endpointDelay, globalDelay time.Duration
	endpointReservations, globalReservations := make(Reservations, rateLimiter.weight), make(Reservations, 0, rateLimiter.weight)
	// Consume 1 weight at a time as this avoids needing burst capacity in the limiter, which would otherwise allow
	// the rate limit to be exceeded over short periods.
	for i := range rateLimiter.weight {
		endpointReservation := rateLimiter.endpoint.Reserve()
		endpointReservations[i] = endpointReservation
		endpointDelay = endpointReservation.Delay()

		if rateLimiter.global != nil {
			globalReservation := rateLimiter.global.Reserve()
			globalReservations = append(globalReservations, globalReservation)
			globalDelay = globalReservation.Delay()
		}
	}

	finalDelay := endpointDelay
	var globalDelaySet bool
	if globalDelay > finalDelay {
		finalDelay = globalDelay
		globalDelaySet = true
	}

	if verbose {
		selected := "endpoint"
		if globalDelaySet {
			selected = "global"
		}
		log.Debugf(log.RequestSys, "rate limit: %s delay selected. endpointDelay=%s globalDelay=%s weight=%d endpoint{%s} global{%s}", selected, endpointDelay, globalDelay, rateLimiter.weight, describeLimiter(rateLimiter.endpoint), describeLimiter(rateLimiter.global))
	}

	if finalDelay == 0 {
		return nil
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		endpointReservations.CancelAll()
		globalReservations.CancelAll()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w", finalDelay, context.DeadlineExceeded)
	}

	select {
	case <-time.After(finalDelay):
		return nil
	case <-ctx.Done():
		endpointReservations.CancelAll()
		globalReservations.CancelAll()
		return ctx.Err()
	}
}

func describeLimiter(l *rate.Limiter) string {
	if l == nil {
		return "nil"
	}
	return fmt.Sprintf("limit=%.3f/s burst=%d", float64(l.Limit()), l.Burst())
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

// InitiateRateLimit sleeps for designated end point rate limits
func (r *Requester) InitiateRateLimit(ctx context.Context, e EndpointLimit, verbose bool) error {
	if atomic.LoadInt32(&r.disableRateLimiter) == 1 {
		return nil
	}
	if r.limiter == nil {
		return fmt.Errorf("cannot rate limit request %w", errLimiterSystemIsNil)
	}

	if err := RateLimit(ctx, r.limiter[e], verbose); err != nil {
		return fmt.Errorf("cannot rate limit request %w for endpoint %d", err, e)
	}

	return nil
}
