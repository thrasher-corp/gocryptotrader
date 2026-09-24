package request

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

// Rate limiting errors.
var (
	ErrRateLimiterAlreadyDisabled = errors.New("rate limiter already disabled")
	ErrRateLimiterAlreadyEnabled  = errors.New("rate limiter already enabled")
	ErrDelayNotAllowed            = errors.New("delay not allowed")

	errInvalidWeight       = errors.New("weight must be equal-or-greater than 1")
	rateLimitReservationMu sync.Mutex
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
	limiter     *rate.Limiter
	weight      Weight
	m           sync.Mutex
	lockOrderID uint64
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
	return &RateLimiterWithWeight{
		limiter:     l,
		weight:      weight,
		lockOrderID: rateLimiterLockOrder.Add(1),
	}
}

// Weight returns the number of reservations consumed by each request.
func (r *RateLimiterWithWeight) Weight() Weight {
	if r == nil {
		return 0
	}
	return r.weight
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
	if r.disableRateLimiter.Load() {
		return WaitForRateLimitBarrier(ctx)
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
	additionalRateLimits := additionalRateLimitsFromContext(ctx)
	if len(additionalRateLimits) > 0 {
		if participant := rateLimitBarrierParticipantFromContext(ctx); participant != nil {
			AbortRateLimitBarrier(ctx)
			return ErrRateLimitBarrierRejected
		}
		return r.applyMultipleRateLimits(ctx, endpointRateLimitWeightFromContext(ctx), additionalRateLimits)
	}

	if r.weight == 0 {
		AbortRateLimitBarrier(ctx)
		return errInvalidWeight
	}

	if participant := rateLimitBarrierParticipantFromContext(ctx); participant != nil {
		admitted, err := participant.wait(ctx, r)
		if err != nil || admitted {
			return err
		}
	}

	rateLimitReservationMu.Lock()
	r.m.Lock()
	tn := time.Now()
	reservations, finalDelay := r.reserveLocked(tn)

	if finalDelay == 0 {
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return nil
	}

	if hasDelayNotAllowed(ctx) {
		cancelAll(reservations, tn)
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return ErrDelayNotAllowed
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(tn.Add(finalDelay)) {
		cancelAll(reservations, tn)
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w", finalDelay, context.DeadlineExceeded)
	}
	r.m.Unlock()
	rateLimitReservationMu.Unlock()

	select {
	case <-ctx.Done():
		r.m.Lock()
		cancelAll(reservations, time.Now())
		r.m.Unlock()
		return ctx.Err()
	case <-time.After(finalDelay):
		return nil
	}
}

// reserve keeps a weighted reservation contiguous while its caller holds the limiter mutex.
func (r *RateLimiterWithWeight) reserve(tn time.Time, weightOverride Weight) ([]*rate.Reservation, time.Duration, error) {
	weight := weightOverride
	if weight == 0 {
		weight = r.weight
	}
	if weight == 0 {
		return nil, 0, errInvalidWeight
	}

	reservations := make([]*rate.Reservation, 0, weight)
	for range weight {
		// Reserving one token at a time avoids requiring burst capacity.
		reservations = append(reservations, r.limiter.ReserveN(tn, 1))
	}
	return reservations, reservations[len(reservations)-1].DelayFrom(tn), nil
}

func (r *RateLimiterWithWeight) reserveLocked(at time.Time) ([]*rate.Reservation, time.Duration) {
	reserved, delay, _ := r.reserve(at, 0)
	return reserved, delay
}

// admitLocked reserves every participant's capacity as one transaction. The barrier lock must be held by the caller.
func (b *rateLimitBarrier) admitLocked() error {
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()

	at := time.Now()
	reservations := make([]*rate.Reservation, 0, len(b.participants))
	for _, participant := range b.participants {
		if contextDone(participant.done) {
			cancelAll(reservations, at)
			return ErrRateLimitBarrierRejected
		}
		if participant.limiter == nil {
			continue
		}
		participant.limiter.m.Lock()
		reserved, delay := participant.limiter.reserveLocked(at)
		participant.limiter.m.Unlock()
		reservations = append(reservations, reserved...)
		if delay != 0 {
			cancelAll(reservations, at)
			return ErrDelayNotAllowed
		}
	}
	for _, participant := range b.participants {
		if contextDone(participant.done) {
			cancelAll(reservations, at)
			return ErrRateLimitBarrierRejected
		}
	}
	return nil
}

func contextDone(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// cancelAll cancels all reservations at a specific time.
// Does not provide locking protection, so callers can maintain a single lock throughout.
func cancelAll(reservations []*rate.Reservation, at time.Time) {
	for _, reservation := range slices.Backward(reservations) {
		reservation.CancelAt(at)
	}
}

// DisableRateLimiter disables the rate limiting system for the exchange.
func (r *Requester) DisableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !r.disableRateLimiter.CompareAndSwap(false, true) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyDisabled)
	}
	return nil
}

// EnableRateLimiter enables the rate limiting system for the exchange.
func (r *Requester) EnableRateLimiter() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	if !r.disableRateLimiter.CompareAndSwap(true, false) {
		return fmt.Errorf("%s %w", r.name, ErrRateLimiterAlreadyEnabled)
	}
	return nil
}
