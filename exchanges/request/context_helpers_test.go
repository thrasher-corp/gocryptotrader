package request

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestIsVerbose(t *testing.T) {
	t.Parallel()
	require.False(t, IsVerbose(t.Context(), false))
	require.True(t, IsVerbose(t.Context(), true))
	require.True(t, IsVerbose(WithVerbose(t.Context()), false))
	require.False(t, IsVerbose(context.WithValue(t.Context(), contextVerboseFlag, false), false))
	require.False(t, IsVerbose(context.WithValue(t.Context(), contextVerboseFlag, "bruh"), false))
	require.True(t, IsVerbose(context.WithValue(t.Context(), contextVerboseFlag, true), false))
}

func TestWithDelayNotAllowed(t *testing.T) {
	t.Parallel()
	assert.True(t, hasDelayNotAllowed(WithDelayNotAllowed(t.Context())))
	assert.False(t, hasDelayNotAllowed(t.Context()))
	assert.False(t, hasDelayNotAllowed(WithRetryNotAllowed(WithVerbose(t.Context()))))
}

func TestNewRateLimitBarrierContexts(t *testing.T) {
	t.Parallel()

	contexts, err := NewRateLimitBarrierContexts(t.Context(), 1)
	require.ErrorIs(t, err, ErrInvalidRateLimitBarrierParticipants)
	require.Nil(t, contexts)

	contexts, err = NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	require.Len(t, contexts, 2)
	require.NotSame(t, rateLimitBarrierParticipantFromContext(contexts[0]), rateLimitBarrierParticipantFromContext(contexts[1]))
}

func TestWaitForRateLimitBarrier(t *testing.T) {
	t.Parallel()
	require.NoError(t, WaitForRateLimitBarrier(t.Context()))

	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	go func() { errs <- WaitForRateLimitBarrier(contexts[0]) }()
	go func() { errs <- WaitForRateLimitBarrier(contexts[1]) }()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
}

func TestWaitForRateLimitBarrierRejectsCancelledFinalParticipant(t *testing.T) {
	t.Parallel()
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errCh := make(chan error, 1)
	go func() { errCh <- WaitForRateLimitBarrier(contexts[0]) }()
	require.Eventually(t, rateLimitBarrierParticipantFromContext(contexts[0]).used.Load,
		time.Second, time.Millisecond, "first participant must reach the barrier")

	cancelledContext, cancel := context.WithCancel(contexts[1])
	cancel()
	require.ErrorIs(t, WaitForRateLimitBarrier(cancelledContext), context.Canceled)
	require.ErrorIs(t, <-errCh, ErrRateLimitBarrierRejected)
}

func TestWaitForRateLimitBarrierRejectsParticipantCancelledWhileParked(t *testing.T) {
	t.Parallel()
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	parkedContext, cancel := context.WithCancel(contexts[0])
	errCh := make(chan error, 1)
	go func() { errCh <- WaitForRateLimitBarrier(parkedContext) }()
	require.Eventually(t, rateLimitBarrierParticipantFromContext(contexts[0]).used.Load,
		time.Second, time.Millisecond, "first participant must reach the barrier")

	cancel()
	require.ErrorIs(t, WaitForRateLimitBarrier(contexts[1]), ErrRateLimitBarrierRejected)
	require.ErrorIs(t, <-errCh, context.Canceled)
}

func TestWaitForRateLimitBarrierRejectsReuseBeforeResolution(t *testing.T) {
	t.Parallel()
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errCh := make(chan error, 1)
	go func() { errCh <- WaitForRateLimitBarrier(contexts[0]) }()
	require.Eventually(t, rateLimitBarrierParticipantFromContext(contexts[0]).used.Load,
		time.Second, time.Millisecond, "first participant must reach the barrier")

	require.ErrorIs(t, WaitForRateLimitBarrier(contexts[0]), ErrRateLimitBarrierParticipantUsed)
	AbortRateLimitBarrier(contexts[1])
	require.ErrorIs(t, <-errCh, ErrRateLimitBarrierRejected)
}

func TestAbortRateLimitBarrier(t *testing.T) {
	t.Parallel()

	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	AbortRateLimitBarrier(contexts[0])
	_, err = rateLimitBarrierParticipantFromContext(contexts[1]).wait(t.Context(), nil)
	require.ErrorIs(t, err, ErrRateLimitBarrierRejected)
	AbortRateLimitBarrier(contexts[0])
	AbortRateLimitBarrier(t.Context())
}

func TestRateLimitBarrierTerminalResults(t *testing.T) {
	t.Parallel()

	barrier := &rateLimitBarrier{done: make(chan struct{})}
	closed, err := barrier.closedResult()
	assert.False(t, closed, "open barrier should not report closed")
	assert.NoError(t, err, "open barrier should not have a result")
	assert.ErrorIs(t, barrier.reuseResult(), ErrRateLimitBarrierParticipantUsed)

	barrier.closed = true
	closed, err = barrier.closedResult()
	assert.True(t, closed, "accepted barrier should report closed")
	assert.NoError(t, err, "accepted barrier should not have an error")
	assert.NoError(t, barrier.reuseResult(), "accepted barrier should allow reuse")

	barrier.rejection = ErrRateLimitBarrierRejected
	assert.ErrorIs(t, barrier.result(), ErrRateLimitBarrierRejected)
	assert.ErrorIs(t, barrier.reuseResult(), ErrRateLimitBarrierRejected)
}

func TestWithHeaders(t *testing.T) {
	t.Parallel()
	headers := http.Header{"User-Agent": {"custom"}, "X-Values": {"one", "two"}}
	ctx := WithHeaders(t.Context(), headers)
	headers.Set("User-Agent", "mutated")

	got := headersFromContext(ctx)
	assert.Equal(t, "custom", got.Get("User-Agent"))
	assert.Equal(t, []string{"one", "two"}, got.Values("X-Values"))
	assert.Nil(t, headersFromContext(t.Context()))
	assert.Same(t, t.Context(), WithHeaders(t.Context(), nil))

	frozen := common.FreezeContext(ctx)
	thawed := common.ThawContext(frozen)
	assert.Equal(t, "custom", headersFromContext(thawed).Get("User-Agent"))
}

func TestWithRetryNotAllowed(t *testing.T) {
	t.Parallel()
	assert.True(t, hasRetryNotAllowed(WithRetryNotAllowed(t.Context())))
	assert.False(t, hasRetryNotAllowed(t.Context()))
	assert.False(t, hasRetryNotAllowed(WithDelayNotAllowed(WithVerbose(t.Context()))))
}
