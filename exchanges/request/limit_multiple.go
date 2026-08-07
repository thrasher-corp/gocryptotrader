package request

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/time/rate"
)

var rateLimiterLockOrder atomic.Uint64

// AdditionalRateLimit describes an additional limiter, optional request-specific weight and non-sensitive diagnostic scope.
type AdditionalRateLimit struct {
	Limiter        *RateLimiterWithWeight
	WeightOverride Weight
	Scope          string
}

type rateLimiterLockSet []*RateLimiterWithWeight

// applyMultipleRateLimits coordinates request-scoped limiters so the request waits only for the longest reservation.
func (r *RateLimiterWithWeight) applyMultipleRateLimits(ctx context.Context, endpointWeightOverride Weight, additionalRateLimits []AdditionalRateLimit) error {
	rateLimits := make([]AdditionalRateLimit, 0, len(additionalRateLimits)+1)
	rateLimits = append(rateLimits, AdditionalRateLimit{
		Limiter:        r,
		WeightOverride: endpointWeightOverride,
		Scope:          "endpoint",
	})
	rateLimits = append(rateLimits, additionalRateLimits...)
	lockSet, err := newRateLimiterLockSet(rateLimits)
	if err != nil {
		return err
	}

	lockSet.lock()
	tn := time.Now()
	reserved := make([]*rate.Reservation, 0, len(rateLimits))
	var finalDelay time.Duration
	var limitingScope string
	for _, rateLimit := range rateLimits {
		reservations, delay, err := rateLimit.Limiter.reserve(tn, rateLimit.WeightOverride)
		if err != nil {
			cancelAll(reserved, tn)
			lockSet.unlock()
			return err
		}
		reserved = append(reserved, reservations...)
		if finalDelay < delay {
			finalDelay = delay
			limitingScope = rateLimit.Scope
		}
	}

	if hasDelayNotAllowed(ctx) {
		if finalDelay > 0 {
			cancelAll(reserved, tn)
			lockSet.unlock()
			return fmt.Errorf("%w for rate-limit scope %q", ErrDelayNotAllowed, limitingScope)
		}
		lockSet.unlock()
		return nil
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(tn.Add(finalDelay)) {
		cancelAll(reserved, tn)
		lockSet.unlock()
		return fmt.Errorf("rate limit delay of %s for scope %q will exceed deadline: %w", finalDelay, limitingScope, context.DeadlineExceeded)
	}

	if finalDelay == 0 {
		lockSet.unlock()
		return nil
	}
	lockSet.unlock()
	if IsVerbose(ctx, false) {
		log.Debugf(log.RequestSys, "Rate limit scope %q requires a %s delay", limitingScope, finalDelay)
	}

	select {
	case <-ctx.Done():
		lockSet.lock()
		cancelAll(reserved, time.Now())
		lockSet.unlock()
		return ctx.Err()
	case <-time.After(finalDelay):
		return nil
	}
}

func newRateLimiterLockSet(rateLimits []AdditionalRateLimit) (rateLimiterLockSet, error) {
	lockSet := make(rateLimiterLockSet, 0, len(rateLimits))
	for _, rateLimit := range rateLimits {
		if err := common.NilGuard(rateLimit.Limiter); err != nil {
			return nil, err
		}
		if !slices.Contains(lockSet, rateLimit.Limiter) {
			lockSet = append(lockSet, rateLimit.Limiter)
		}
	}
	// Stable ordering prevents deadlocks between requests with overlapping limiter sets.
	slices.SortFunc(lockSet, func(a, b *RateLimiterWithWeight) int {
		return cmp.Compare(a.lockOrderID, b.lockOrderID)
	})
	return lockSet, nil
}

func (l rateLimiterLockSet) lock() {
	for _, limiter := range l {
		limiter.m.Lock()
	}
}

func (l rateLimiterLockSet) unlock() {
	for i := len(l) - 1; i >= 0; i-- {
		l[i].m.Unlock()
	}
}
