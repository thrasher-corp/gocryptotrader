package request

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
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
	// ErrWeightExceedsBudget is returned for a request weighing more than its endpoint's window budget admits in a whole
	// window, which no wait would make room for
	ErrWeightExceedsBudget = errors.New("request weight exceeds its budget")

	errInvalidWeight       = errors.New("weight must be equal-or-greater than 1")
	rateLimitReservationMu sync.Mutex
	// futureReservations holds the latest time each limiter was reserved at while that time is still to come. Guarded
	// by rateLimitReservationMu. rate.Limiter moves its clock to the time of every reservation, so one made for an
	// earlier time would count the tokens accrued between the two twice and admit more than the limiter's rate. An
	// entry goes when its limiter is next planned rather than in a sweep, since a sweep on one clock would drop entries a
	// synctest bubble on another still needs; a limiter discarded while reserved ahead keeps its small entry
	futureReservations = make(map[*rate.Limiter]time.Time)
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
	// budget, when set, is the endpoint's own budget in place of limiter, which is then unrestricted. Each request books
	// its weight in it
	budget *WindowBudget
	// shared budgets, such as a venue's per-IP limit, count each request once whatever its weight, on top of the
	// endpoint's own
	shared []*WindowBudget
	weight Weight
	// Redundant under rateLimitReservationMu in production, but retained so
	// TestRateLimitReservationLock and TestRateLimitBarrierCancellationDuringAdmission
	// can pause admission deterministically to verify locking and cancellation safeguards.
	m sync.Mutex
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

// GetRateLimiterWithShared couples a rate limiter with a weight count like GetRateLimiterWithWeight, and also counts
// each request once in every distinct shared budget, for venues that count every request against a common budget,
// such as a per-IP limit, on top of each endpoint's own limit. A request is sent at the earliest moment its own limiter
// and every shared budget allow, and each counts it at that moment. A shared budget gives a request any moment it still
// has room at, even ahead of moments held for requests that came before, so a request waiting on its own endpoint holds
// up no other, and no later request can take its moment. Taking the limiter rather than a rate lets several
// definitions draw on one endpoint budget, with or without the shared ones
func GetRateLimiterWithShared(l *rate.Limiter, weight Weight, shared ...*WindowBudget) *RateLimiterWithWeight {
	r := GetRateLimiterWithWeight(l, weight)
	r.shared = make([]*WindowBudget, 0, len(shared))
	for _, s := range shared {
		if s != nil && !slices.Contains(r.shared, s) {
			r.shared = append(r.shared, s)
		}
	}
	return r
}

// GetRateLimiterWithWindowBudget returns a definition whose own budget is a WindowBudget, for venues that count an
// endpoint's requests over a rolling window, and which counts each request once in every distinct shared budget, as
// GetRateLimiterWithShared does. Each request books its weight in budget at its send time, so a request the venue
// counts once for each of its parts, such as a batch of orders, takes them all at the moment it is sent. Several
// definitions can draw on one budget, with or without the shared ones. A budget also given as shared is counted as the
// definition's own, and a nil budget leaves the shared ones alone to limit the definition
func GetRateLimiterWithWindowBudget(budget *WindowBudget, weight Weight, shared ...*WindowBudget) *RateLimiterWithWeight {
	r := GetRateLimiterWithShared(rate.NewLimiter(rate.Inf, 1), weight, shared...)
	r.budget = budget
	r.shared = slices.DeleteFunc(r.shared, func(s *WindowBudget) bool { return s == budget })
	return r
}

// WindowBudget admits at most a number of requests in any window of a given length, as venues count a rolling budget:
// one shared by every endpoint, such as a per-IP limit, or an endpoint's own, in which a request books its weight. It
// holds the send time of each request in the window before now and of each request booked for later, and gives a
// request the earliest moment with room, even ahead of moments booked for later requests, so that each budget stays
// exact without one endpoint's backlog holding up another's requests
type WindowBudget struct {
	window   time.Duration
	requests int
	// booked holds send times in order, once for each request they count. Guarded by rateLimitReservationMu
	booked []time.Time
}

// NewWindowBudget returns a budget admitting at most the given requests in any half-open window of the given length,
// so that a burst can use the whole budget at once. It returns nil, an unrestricted budget which
// GetRateLimiterWithShared and GetRateLimiterWithWindowBudget leave out, when either is not positive
func NewWindowBudget(window time.Duration, requests int) *WindowBudget {
	if window <= 0 || requests <= 0 {
		return nil
	}
	return &WindowBudget{window: window, requests: requests}
}

// pruneLocked forgets send times that no window holding now or a later moment can include. rateLimitReservationMu must
// be held
func (b *WindowBudget) pruneLocked(now time.Time) {
	b.booked = slices.Delete(b.booked, 0, firstAfter(b.booked, now.Add(-b.window)))
}

// earliestLocked returns the earliest moment from t on with room for n more requests, n being from 1 to the budget's
// requests. A moment has room while no window holding it holds more than requests-n bookings, so it has none when it is
// less than a window after the first, and less than a window before the last, of a run of requests-n+1 consecutive
// bookings that spans less than a window. Both bounds rise from run to run, so one pass from the first run that can
// reach t finds the answer, and stops at the first run ending a whole window after it. rateLimitReservationMu must be
// held
func (b *WindowBudget) earliestLocked(t time.Time, n int) time.Time {
	run := b.requests - n + 1
	at := t
	for i := firstAfter(b.booked, t.Add(-b.window)); i+run <= len(b.booked); i++ {
		last := b.booked[i+run-1]
		if !last.Before(at.Add(b.window)) {
			break
		}
		if end := b.booked[i].Add(b.window); last.Before(end) && at.Before(end) {
			at = end
		}
	}
	return at
}

// bookLocked holds a send time for n requests, after any held for the same moment. rateLimitReservationMu must be held
func (b *WindowBudget) bookLocked(at time.Time, n int) {
	i := firstAfter(b.booked, at)
	for range n {
		b.booked = slices.Insert(b.booked, i, at)
	}
}

// cancelLocked gives back a send time held for up to n requests that will not be sent. rateLimitReservationMu must be
// held
func (b *WindowBudget) cancelLocked(at time.Time, n int) {
	end := firstAfter(b.booked, at)
	start := end
	for start > 0 && end-start < n && b.booked[start-1].Equal(at) {
		start--
	}
	b.booked = slices.Delete(b.booked, start, end)
}

// firstAfter returns the index of the first of the ordered times that is after t
func firstAfter(times []time.Time, t time.Time) int {
	return sort.Search(len(times), func(i int) bool { return times[i].After(t) })
}

// Weight returns the number of reservations consumed by each request whose context sets no weight of its own.
func (r *RateLimiterWithWeight) Weight() Weight {
	if r == nil {
		return 0
	}
	return r.weight
}

// Limit returns the endpoint's own budget in actions per second, which is otherwise only observable by
// timing admissions. Exposed so an exchange's tests can pin the budget its endpoints draw on. A nil
// receiver is the RateLimitNotRequired sentinel and reports 0; an unrestricted limiter reports rate.Inf, and a
// WindowBudget its requests over its window, though it admits them all at once.
func (r *RateLimiterWithWeight) Limit() rate.Limit {
	if r == nil {
		return 0
	}
	if r.budget != nil {
		return rate.Limit(float64(r.budget.requests) / r.budget.window.Seconds())
	}
	return r.limiter.Limit()
}

// SharesBudgetWith reports whether both endpoints draw on the same budget of their own: the same rate.Limiter, or the
// same WindowBudget given to GetRateLimiterWithWindowBudget. Two endpoints given identical but separate budgets admit
// twice the documented rate between them, which comparing rates alone cannot detect. Shared budgets, such as a venue's
// per-IP limit, do not count, since every endpoint of the venue draws on them.
func (r *RateLimiterWithWeight) SharesBudgetWith(other *RateLimiterWithWeight) bool {
	if r == nil || other == nil {
		return r == other
	}
	if r.budget != nil || other.budget != nil {
		return r.budget == other.budget
	}
	return r.limiter == other.limiter
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

	weight, err := r.weightFor(ctx)
	if err != nil {
		AbortRateLimitBarrier(ctx)
		return err
	}

	if participant := rateLimitBarrierParticipantFromContext(ctx); participant != nil {
		admitted, err := participant.wait(ctx, r, weight)
		if err != nil || admitted {
			return err
		}
	}

	rateLimitReservationMu.Lock()
	r.m.Lock()
	tn := time.Now()
	finalDelay, plan := r.planLocked(tn, weight)

	if finalDelay == 0 {
		plan.reserveLocked(tn)
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return nil
	}

	if hasDelayNotAllowed(ctx) {
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return ErrDelayNotAllowed
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(tn.Add(finalDelay)) {
		r.m.Unlock()
		rateLimitReservationMu.Unlock()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w", finalDelay, context.DeadlineExceeded)
	}
	reserved := plan.reserveLocked(tn)
	r.m.Unlock()
	rateLimitReservationMu.Unlock()

	select {
	case <-ctx.Done():
		// A request that will not be sent gives back what it reserved, so that it holds up no later request on its own
		// budget or on a shared one
		rateLimitReservationMu.Lock()
		reserved.cancelLocked(time.Now())
		rateLimitReservationMu.Unlock()
		return ctx.Err()
	case <-time.After(finalDelay):
		return nil
	}
}

// weightFor returns the weight a request takes from its endpoint's own budget: its context's, or else its definition's
func (r *RateLimiterWithWeight) weightFor(ctx context.Context) (Weight, error) {
	weight, ok := weightFromContext(ctx)
	if !ok {
		weight = r.weight
	}
	if weight == 0 {
		return 0, errInvalidWeight
	}
	if r.budget != nil && int(weight) > r.budget.requests {
		return 0, fmt.Errorf("%w: weight %d, budget %d in %s", ErrWeightExceedsBudget, weight, r.budget.requests, r.budget.window)
	}
	return weight, nil
}

// windowBooking is how many requests a request counts as in a window budget
type windowBooking struct {
	budget   *WindowBudget
	requests int
}

// bookings returns what a request of the given weight books in each window budget: its weight in the endpoint's own,
// and one request in each shared budget
func (r *RateLimiterWithWeight) bookings(weight Weight) []windowBooking {
	if r.budget == nil && len(r.shared) == 0 {
		return nil
	}
	bookings := make([]windowBooking, 0, len(r.shared)+1)
	if r.budget != nil {
		bookings = append(bookings, windowBooking{r.budget, int(weight)})
	}
	for _, s := range r.shared {
		bookings = append(bookings, windowBooking{s, 1})
	}
	return bookings
}

// rateLimitPlan is when a request is sent, and when its own limiter is reserved so that its tokens fall due by then
type rateLimitPlan struct {
	limiter   *rate.Limiter
	tokens    int
	reserveAt time.Time
	sendAt    time.Time
	bookings  []windowBooking
}

// rateLimitReservation is what a request took, kept so that it can be given back
type rateLimitReservation struct {
	plan   *rateLimitPlan
	tokens []*rate.Reservation
}

// planLocked returns how long a request of the given weight waits until its own limiter and every window budget can
// take it, and how to reserve them for that moment. Tokens taken for an earlier moment would be spent while the request
// waits on a shared budget, letting the requests behind it on the endpoint go out together. A limiter that never grants
// the request gives rate.InfDuration and an empty plan. rateLimitReservationMu must be held
func (r *RateLimiterWithWeight) planLocked(now time.Time, weight Weight) (time.Duration, *rateLimitPlan) {
	p := &rateLimitPlan{bookings: r.bookings(weight)}
	grant := now
	if r.limiter.Limit() != rate.Inf {
		p.limiter, p.tokens, p.reserveAt = r.limiter, int(weight), reservableFromLocked(r.limiter, now)
		var ok bool
		if grant, ok = earliestGrant(r.limiter, p.reserveAt, p.tokens); !ok {
			return rate.InfDuration, &rateLimitPlan{}
		}
	}
	for _, b := range p.bookings {
		b.budget.pruneLocked(now)
	}
	p.sendAt = grant
	// A moment with room in one budget can fall in another's full window, so every budget checks each later moment
	for moved := true; moved; {
		moved = false
		for _, b := range p.bookings {
			if at := b.budget.earliestLocked(p.sendAt, b.requests); at.After(p.sendAt) {
				p.sendAt, moved = at, true
			}
		}
	}
	if p.limiter != nil && grant.Before(p.sendAt) {
		// Tokens beyond the burst accrue one by one before the last is taken at the send time
		p.reserveAt = p.sendAt.Add(-durationFromTokens(p.limiter.Limit(), float64(max(0, p.tokens-p.limiter.Burst()))))
	}
	return p.sendAt.Sub(now), p
}

// reserveLocked takes a plan's tokens one at a time, so that no burst capacity is needed for a request's weight, which
// would otherwise let the limit be exceeded over short periods, and books its send time in every window budget.
// rateLimitReservationMu must be held
func (p *rateLimitPlan) reserveLocked(now time.Time) *rateLimitReservation {
	res := &rateLimitReservation{plan: p, tokens: make([]*rate.Reservation, p.tokens)}
	for i := range res.tokens {
		res.tokens[i] = p.limiter.ReserveN(p.reserveAt, 1)
	}
	if p.reserveAt.After(now) {
		futureReservations[p.limiter] = p.reserveAt
	}
	for _, b := range p.bookings {
		b.budget.bookLocked(p.sendAt, b.requests)
	}
	return res
}

// cancelLocked gives a reservation back: its send time to every window budget, and its tokens in reverse order, for
// correct token reimbursement, at the earliest time their limiter accepts. rateLimitReservationMu must be held
func (res *rateLimitReservation) cancelLocked(now time.Time) {
	for _, b := range res.plan.bookings {
		b.budget.cancelLocked(res.plan.sendAt, b.requests)
	}
	if len(res.tokens) == 0 {
		return
	}
	at := reservableFromLocked(res.plan.limiter, now)
	for _, t := range slices.Backward(res.tokens) {
		t.CancelAt(at)
	}
}

// reservableFromLocked returns the earliest time l can be reserved or given back at without moving its clock back.
// rateLimitReservationMu must be held
func reservableFromLocked(l *rate.Limiter, now time.Time) time.Time {
	if at, ok := futureReservations[l]; ok {
		if at.After(now) {
			return at
		}
		delete(futureReservations, l)
	}
	return now
}

// earliestGrant returns the earliest time from 'from' on at which l has accrued n tokens taken one at a time, or false
// if it never will
func earliestGrant(l *rate.Limiter, from time.Time, n int) (time.Time, bool) {
	if l.Burst() < 1 {
		return time.Time{}, false
	}
	short := float64(n) - l.TokensAt(from)
	if short <= 0 {
		return from, true
	}
	wait := durationFromTokens(l.Limit(), short)
	if wait == rate.InfDuration {
		return time.Time{}, false
	}
	return from.Add(wait), true
}

// durationFromTokens returns how long a limit takes to accrue tokens, rounded as rate.Limiter rounds its reservations so
// that planned times match the reservations it makes
func durationFromTokens(limit rate.Limit, tokens float64) time.Duration {
	if limit <= 0 {
		return rate.InfDuration
	}
	duration := (tokens / float64(limit)) * float64(time.Second)
	if duration > float64(math.MaxInt64) {
		return rate.InfDuration
	}
	return time.Duration(duration)
}

// admitLocked reserves every participant's capacity as one transaction. The barrier lock must be held by the caller.
func (b *rateLimitBarrier) admitLocked() error {
	rateLimitReservationMu.Lock()
	defer rateLimitReservationMu.Unlock()

	at := time.Now()
	reservations := make([]*rateLimitReservation, 0, len(b.participants))
	for _, participant := range b.participants {
		if contextDone(participant.done) {
			cancelReservationsLocked(reservations, at)
			return ErrRateLimitBarrierRejected
		}
		if participant.limiter == nil {
			continue
		}
		participant.limiter.m.Lock()
		delay, plan := participant.limiter.planLocked(at, participant.weight)
		if delay != 0 {
			participant.limiter.m.Unlock()
			cancelReservationsLocked(reservations, at)
			return ErrDelayNotAllowed
		}
		reservations = append(reservations, plan.reserveLocked(at))
		participant.limiter.m.Unlock()
	}
	for _, participant := range b.participants {
		if contextDone(participant.done) {
			cancelReservationsLocked(reservations, at)
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

// cancelReservationsLocked gives reservations back in reverse order, for correct token reimbursement.
// rateLimitReservationMu must be held
func cancelReservationsLocked(reservations []*rateLimitReservation, now time.Time) {
	for _, r := range slices.Backward(reservations) {
		r.cancelLocked(now)
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
