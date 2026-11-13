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

func TestWithWeight(t *testing.T) {
	t.Parallel()
	weight, ok := weightFromContext(t.Context())
	assert.False(t, ok, "a context without a weight should report none")
	assert.Zero(t, weight, "a context without a weight should report a zero weight")
	ctx := WithWeight(t.Context(), 10)
	weight, ok = weightFromContext(ctx)
	assert.True(t, ok, "a weighted context should report its weight")
	assert.Equal(t, Weight(10), weight, "a weighted context should report the weight it was given")
	weight, ok = weightFromContext(WithWeight(WithDelayNotAllowed(ctx), 0))
	assert.True(t, ok, "a context weighted zero should report its weight, which RateLimit refuses")
	assert.Zero(t, weight, "the latest weight should take the place of an earlier one")
	_, ok = weightFromContext(common.ThawContext(common.FreezeContext(ctx)))
	assert.False(t, ok, "a weight belongs to one request, so a frozen context should not carry it")
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
	_, err = rateLimitBarrierParticipantFromContext(contexts[1]).wait(t.Context(), nil, 0)
	require.ErrorIs(t, err, ErrRateLimitBarrierRejected)
	AbortRateLimitBarrier(contexts[0])
	AbortRateLimitBarrier(t.Context())
}

func TestRateLimitBarrierParticipantWait(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimitWithWeight(time.Hour, 1, 1)

	t.Run("a group is planned at the weights its participants arrived with", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "barrier contexts must be created")
		parked := rateLimitBarrierParticipantFromContext(contexts[0])
		errs := make(chan error, 1)
		go func() {
			admitted, err := parked.wait(contexts[0], nil, 0)
			assert.True(t, admitted, "a participant resolved by its barrier should report it")
			errs <- err
		}()
		require.Eventually(t, parked.used.Load, time.Second, time.Millisecond, "the first participant must reach the barrier")
		final := rateLimitBarrierParticipantFromContext(contexts[1])
		admitted, err := final.wait(contexts[1], limiter, 3)
		assert.True(t, admitted, "the final participant should resolve the barrier")
		assert.ErrorIs(t, err, ErrDelayNotAllowed, "a weight its limiter cannot grant at once should reject the group")
		assert.Equal(t, Weight(3), final.weight, "a participant should be admitted at the weight it arrived with")
		assert.Same(t, limiter, final.limiter, "a participant should be admitted on the limiter it arrived with")
		assert.ErrorIs(t, <-errs, ErrDelayNotAllowed, "a parked participant should share the group's outcome")
		admitted, err = final.wait(contexts[1], limiter, 1)
		assert.False(t, admitted, "a participant already resolved should not be resolved again")
		assert.ErrorIs(t, err, ErrDelayNotAllowed, "a participant already resolved should report its group's outcome")
	})

	t.Run("a participant of a closed barrier reports its outcome", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "barrier contexts must be created")
		AbortRateLimitBarrier(contexts[0])
		admitted, err := rateLimitBarrierParticipantFromContext(contexts[1]).wait(contexts[1], limiter, 1)
		assert.True(t, admitted, "a participant of a closed barrier should be resolved")
		assert.ErrorIs(t, err, ErrRateLimitBarrierRejected, "a participant of a rejected barrier should be rejected")
	})

	t.Run("a participant whose context is done rejects its peers", func(t *testing.T) {
		t.Parallel()
		parent, cancel := context.WithCancel(t.Context())
		contexts, err := NewRateLimitBarrierContexts(parent, 2)
		require.NoError(t, err, "barrier contexts must be created")
		cancel()
		admitted, err := rateLimitBarrierParticipantFromContext(contexts[0]).wait(contexts[0], limiter, 1)
		assert.True(t, admitted, "a cancelled participant should be resolved")
		assert.ErrorIs(t, err, context.Canceled, "a cancelled participant should report its cancellation")
		assert.ErrorIs(t, WaitForRateLimitBarrier(contexts[1]), ErrRateLimitBarrierRejected, "a cancelled participant should reject its peers")
	})

	t.Run("a participant cancelled while parked rejects its peers", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "barrier contexts must be created")
		ctx, cancel := context.WithCancel(contexts[0])
		parked := rateLimitBarrierParticipantFromContext(contexts[0])
		errs := make(chan error, 1)
		go func() {
			_, err := parked.wait(ctx, limiter, 1)
			errs <- err
		}()
		require.Eventually(t, parked.used.Load, time.Second, time.Millisecond, "the first participant must reach the barrier")
		cancel()
		assert.ErrorIs(t, <-errs, context.Canceled, "a participant cancelled while parked should report its cancellation")
		assert.ErrorIs(t, WaitForRateLimitBarrier(contexts[1]), ErrRateLimitBarrierRejected, "a participant cancelled while parked should reject its peers")
	})

	t.Run("a participant reaching a barrier closed meanwhile shares its outcome", func(t *testing.T) {
		t.Parallel()
		contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "barrier contexts must be created")
		arriving := rateLimitBarrierParticipantFromContext(contexts[0])
		arriving.barrier.mu.Lock()
		errs := make(chan error, 1)
		go func() {
			admitted, err := arriving.wait(contexts[0], limiter, 1)
			assert.True(t, admitted, "a participant of a closed barrier should be resolved")
			errs <- err
		}()
		require.Eventually(t, arriving.used.Load, time.Second, time.Millisecond, "the participant must start waiting before its barrier closes")
		arriving.barrier.rejection, arriving.barrier.closed = ErrRateLimitBarrierRejected, true
		close(arriving.barrier.done)
		arriving.barrier.mu.Unlock()
		assert.ErrorIs(t, <-errs, ErrRateLimitBarrierRejected, "a participant reaching a barrier closed meanwhile should share its outcome")
		assert.Nil(t, arriving.limiter, "a participant reaching a closed barrier should not be recorded")
	})
}

func TestRateLimitBarrierArrive(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimitWithWeight(time.Hour, 1, 1)
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err, "barrier contexts must be created")
	first, final := rateLimitBarrierParticipantFromContext(contexts[0]), rateLimitBarrierParticipantFromContext(contexts[1])
	b := first.barrier
	assert.False(t, b.arrive(contexts[0], first, limiter, 2), "a participant with peers still to come should not resolve the barrier")
	assert.Equal(t, Weight(2), first.weight, "arrive should record the participant's weight")
	assert.Same(t, limiter, first.limiter, "arrive should record the participant's limiter")
	assert.NotNil(t, first.done, "arrive should record the participant's cancellation")
	assert.True(t, b.arrive(contexts[1], final, nil, 0), "the final participant should resolve the barrier")
	assert.ErrorIs(t, b.result(), ErrDelayNotAllowed, "the final arrival should admit the group at the weights recorded, here more than the limiter grants at once")
	late := &rateLimitBarrierParticipant{barrier: b}
	assert.True(t, b.arrive(t.Context(), late, limiter, 1), "a participant of a closed barrier should be resolved")
	assert.Zero(t, late.weight, "a participant of a closed barrier should not be recorded")
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
