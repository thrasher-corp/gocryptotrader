package request

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	// ErrInvalidRateLimitBarrierParticipants is returned when a barrier cannot coordinate at least two requests.
	ErrInvalidRateLimitBarrierParticipants = errors.New("rate limit barrier requires at least two participants")
	// ErrRateLimitBarrierParticipantUsed is returned when a participant context reaches a limiter again before its barrier resolves.
	ErrRateLimitBarrierParticipantUsed = errors.New("rate limit barrier participant already used")
	// ErrRateLimitBarrierRejected is returned when a peer aborts or is cancelled before group admission.
	ErrRateLimitBarrierRejected = errors.New("rate limit barrier rejected")
)

const contextVerboseFlag verbosity = "verbose"

type verbosity string

type headersKey struct{}

func init() {
	common.RegisterContextKey(headersKey{})
}

// WithVerbose adds verbosity to a request context so that specific requests
// can have distinct verbosity without impacting all requests.
func WithVerbose(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextVerboseFlag, true)
}

// IsVerbose checks main verbosity first then checks context verbose values
// for specific request verbosity.
func IsVerbose(ctx context.Context, verbose bool) bool {
	if !verbose {
		verbose, _ = ctx.Value(contextVerboseFlag).(bool)
	}
	return verbose
}

// WithHeaders adds outbound HTTP header overrides to the context. These values
// replace matching generated headers, including authentication headers.
func WithHeaders(ctx context.Context, headers http.Header) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return context.WithValue(ctx, headersKey{}, headers.Clone())
}

func headersFromContext(ctx context.Context) http.Header {
	headers, _ := ctx.Value(headersKey{}).(http.Header)
	return headers
}

type delayNotAllowedKey struct{}

// WithDelayNotAllowed adds a value to the context that indicates that no delay is allowed for rate limiting.
func WithDelayNotAllowed(ctx context.Context) context.Context {
	return context.WithValue(ctx, delayNotAllowedKey{}, struct{}{})
}

func hasDelayNotAllowed(ctx context.Context) bool {
	_, ok := ctx.Value(delayNotAllowedKey{}).(struct{})
	return ok
}

type rateLimitBarrierKey struct{}

type rateLimitBarrier struct {
	done         chan struct{}
	participants []*rateLimitBarrierParticipant
	remaining    uint
	rejection    error
	closed       bool
	mu           sync.Mutex
}

type rateLimitBarrierParticipant struct {
	barrier *rateLimitBarrier
	done    <-chan struct{}
	limiter *RateLimiterWithWeight
	used    atomic.Bool
}

// NewRateLimitBarrierContexts returns distinct contexts whose first rate-limit calls proceed only when every participant can proceed immediately.
// Each request owner must defer AbortRateLimitBarrier so a failure before reaching the limiter releases the other participants. The final arrival
// atomically reserves capacity for the group; shared limiters must have enough burst capacity for all participants, and each weight must fit its
// limiter's burst. After acceptance, subsequent rate-limit calls using a participant context proceed normally, including retries.
// Coordination covers only the first gate, not request execution: additional rate-limited calls under the same participant context can fail
// independently and lose the group's all-or-nothing admission guarantee. Cancellation or transport errors after admission can also split execution.
// NewRateLimit uses burst 1, so participants using its finite limiters must use distinct limiter instances and weight 1. Larger groups sharing
// a limiter or larger weights require sufficient burst capacity, for example via GetRateLimiterWithWeight with a custom limiter.
func NewRateLimitBarrierContexts(ctx context.Context, participants uint) ([]context.Context, error) {
	if participants < 2 {
		return nil, ErrInvalidRateLimitBarrierParticipants
	}
	barrier := &rateLimitBarrier{
		done:         make(chan struct{}),
		participants: make([]*rateLimitBarrierParticipant, participants),
		remaining:    participants,
	}
	contexts := make([]context.Context, participants)
	for i := range contexts {
		participant := &rateLimitBarrierParticipant{barrier: barrier}
		barrier.participants[i] = participant
		contexts[i] = context.WithValue(ctx, rateLimitBarrierKey{}, participant) //nolint:fatcontext // Participants are independent siblings of the same parent.
	}
	return contexts, nil
}

// AbortRateLimitBarrier rejects a participant that failed before reaching RateLimit. It is a no-op after the participant is consumed.
func AbortRateLimitBarrier(ctx context.Context) {
	participant, _ := ctx.Value(rateLimitBarrierKey{}).(*rateLimitBarrierParticipant)
	if participant == nil || !participant.used.CompareAndSwap(false, true) {
		return
	}
	participant.barrier.reject()
}

// WaitForRateLimitBarrier marks a request without an active limiter as immediately available and waits for its peers.
func WaitForRateLimitBarrier(ctx context.Context) error {
	participant := rateLimitBarrierParticipantFromContext(ctx)
	if participant == nil {
		return nil
	}
	_, err := participant.wait(ctx, nil)
	return err
}

func rateLimitBarrierParticipantFromContext(ctx context.Context) *rateLimitBarrierParticipant {
	participant, _ := ctx.Value(rateLimitBarrierKey{}).(*rateLimitBarrierParticipant)
	return participant
}

func (p *rateLimitBarrierParticipant) wait(ctx context.Context, limiter *RateLimiterWithWeight) (bool, error) {
	if !p.used.CompareAndSwap(false, true) {
		return false, p.barrier.reuseResult()
	}
	if closed, err := p.barrier.closedResult(); closed {
		return true, err
	}
	if err := ctx.Err(); err != nil {
		p.barrier.reject()
		return true, err
	}
	if p.barrier.arrive(ctx, p, limiter) {
		return true, p.barrier.resultFor(ctx)
	}
	select {
	case <-p.barrier.done:
		return true, p.barrier.resultFor(ctx)
	case <-ctx.Done():
		if p.barrier.reject() {
			return true, ctx.Err()
		}
		return true, p.barrier.resultFor(ctx)
	}
}

func (b *rateLimitBarrier) arrive(ctx context.Context, participant *rateLimitBarrierParticipant, limiter *RateLimiterWithWeight) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return true
	}
	participant.done = ctx.Done()
	participant.limiter = limiter
	b.remaining--
	if b.remaining == 0 {
		b.rejection = b.admitLocked()
		b.closed = true
		close(b.done)
		return true
	}
	return false
}

func (b *rateLimitBarrier) reject() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return false
	}
	b.rejection = ErrRateLimitBarrierRejected
	b.closed = true
	close(b.done)
	return true
}

func (b *rateLimitBarrier) result() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rejection
}

func (b *rateLimitBarrier) resultFor(ctx context.Context) error {
	err := b.result()
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func (b *rateLimitBarrier) closedResult() (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		return false, nil
	}
	return true, b.rejection
}

func (b *rateLimitBarrier) reuseResult() error {
	closed, err := b.closedResult()
	if !closed {
		return ErrRateLimitBarrierParticipantUsed
	}
	return err
}

type retryNotAllowedKey struct{}

// WithRetryNotAllowed adds a value to the context that indicates that no retries are allowed for requests.
func WithRetryNotAllowed(ctx context.Context) context.Context {
	return context.WithValue(ctx, retryNotAllowedKey{}, struct{}{})
}

func hasRetryNotAllowed(ctx context.Context) bool {
	_, ok := ctx.Value(retryNotAllowedKey{}).(struct{})
	return ok
}
