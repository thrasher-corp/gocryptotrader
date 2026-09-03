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
	// ErrRateLimitBarrierParticipantUsed is returned when a one-use participant context reaches a limiter more than once.
	ErrRateLimitBarrierParticipantUsed = errors.New("rate limit barrier participant already used")
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
	done      chan struct{}
	remaining uint
	rejected  bool
	closed    bool
	mu        sync.Mutex
}

type rateLimitBarrierParticipant struct {
	barrier *rateLimitBarrier
	used    atomic.Bool
}

// NewRateLimitBarrierContexts returns distinct one-use contexts whose rate-limit calls proceed only when every participant can proceed immediately.
// Each request owner must defer AbortRateLimitBarrier so a failure before reaching the limiter releases the other participants.
func NewRateLimitBarrierContexts(ctx context.Context, participants uint) ([]context.Context, error) {
	if participants < 2 {
		return nil, ErrInvalidRateLimitBarrierParticipants
	}
	barrier := &rateLimitBarrier{done: make(chan struct{}), remaining: participants}
	contexts := make([]context.Context, participants)
	for i := range contexts {
		contexts[i] = context.WithValue(ctx, rateLimitBarrierKey{}, &rateLimitBarrierParticipant{barrier: barrier}) //nolint:fatcontext // Participants are independent siblings of the same parent.
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
	return participant.wait(ctx, true)
}

func rateLimitBarrierParticipantFromContext(ctx context.Context) *rateLimitBarrierParticipant {
	participant, _ := ctx.Value(rateLimitBarrierKey{}).(*rateLimitBarrierParticipant)
	return participant
}

func (p *rateLimitBarrierParticipant) wait(ctx context.Context, immediate bool) error {
	if !p.used.CompareAndSwap(false, true) {
		return ErrRateLimitBarrierParticipantUsed
	}
	if !immediate {
		p.barrier.reject()
		return ErrDelayNotAllowed
	}
	if p.barrier.arrive() {
		return nil
	}
	select {
	case <-p.barrier.done:
		return p.barrier.result()
	case <-ctx.Done():
		if p.barrier.reject() {
			return ctx.Err()
		}
		return p.barrier.result()
	}
}

func (b *rateLimitBarrier) arrive() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return !b.rejected
	}
	b.remaining--
	if b.remaining == 0 {
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
	b.rejected = true
	b.closed = true
	close(b.done)
	return true
}

func (b *rateLimitBarrier) result() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.rejected {
		return ErrDelayNotAllowed
	}
	return nil
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
