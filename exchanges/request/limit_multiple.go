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
	// reservationPlan groups repeated uses of a limiter and tracks when its first
	// and last tokens are available so all limiters finish at the same send time.
	type reservationPlan struct {
		limiter    *RateLimiterWithWeight
		weight     int
		scope      string
		firstDelay time.Duration
		finalDelay time.Duration
	}
	plans := make([]reservationPlan, 0, len(lockSet))
	planIndexes := make(map[*RateLimiterWithWeight]int, len(lockSet))
	for _, rateLimit := range rateLimits {
		weight := rateLimit.WeightOverride
		if weight == 0 {
			weight = rateLimit.Limiter.weight
		}
		if weight == 0 {
			return errInvalidWeight
		}
		if index, ok := planIndexes[rateLimit.Limiter]; ok {
			plans[index].weight += int(weight)
			continue
		}
		planIndexes[rateLimit.Limiter] = len(plans)
		plans = append(plans, reservationPlan{
			limiter: rateLimit.Limiter,
			weight:  int(weight),
			scope:   rateLimit.Scope,
		})
	}

	lockSet.lock()
	tn := time.Now()
	reservationCount := 0
	for i := range plans {
		reservationCount += plans[i].weight
	}
	reserved := make([]*rate.Reservation, 0, reservationCount)
	var finalDelay time.Duration
	var limitingScope string
	for i := range plans {
		for token := range plans[i].weight {
			reservation := plans[i].limiter.limiter.ReserveN(tn, 1)
			reserved = append(reserved, reservation)
			delay := reservation.DelayFrom(tn)
			if token == 0 {
				plans[i].firstDelay = delay
			}
			plans[i].finalDelay = delay
		}
		if finalDelay < plans[i].finalDelay {
			finalDelay = plans[i].finalDelay
			limitingScope = plans[i].scope
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
	if err := ctx.Err(); err != nil {
		cancelAll(reserved, tn)
		lockSet.unlock()
		return err
	}

	if finalDelay == 0 {
		lockSet.unlock()
		return nil
	}

	// The initial reservations find the common transmission time. For burst-one
	// limiters without competing reservations, re-reserving each limiter so its
	// final weighted token lands at that time prevents a non-binding limiter from
	// refilling before the request is actually sent.
	cancelAll(reserved, tn)
	reserved = reserved[:0]
	for i := range plans {
		startAt := tn.Add(finalDelay - (plans[i].finalDelay - plans[i].firstDelay))
		for range plans[i].weight {
			reserved = append(reserved, plans[i].limiter.limiter.ReserveN(startAt, 1))
		}
	}
	lockSet.unlock()
	if IsVerbose(ctx, false) {
		log.Debugf(log.RequestSys, "Rate limit scope %q requires a %s delay", limitingScope, finalDelay)
	}

	select {
	case <-ctx.Done():
		lockSet.lock()
		// A later reservation can prevent x/time/rate from fully rolling back an
		// earlier one, so cancellation after releasing the locks is best effort.
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
