package dispatch

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

var nonEmptyUUID = uuid.UUID{108, 105, 99, 107, 77, 121, 72, 97, 105, 114, 121, 66, 97, 108, 108, 115}

func TestGlobalDispatcher(t *testing.T) {
	err := Start(0, 0)
	require.NoError(t, err, "Start must not error")
	assert.True(t, IsRunning(), "IsRunning should return true")

	err = Stop()
	require.NoError(t, err, "Stop must not error")
	assert.False(t, IsRunning(), "IsRunning should return false")

	err = EnsureRunning(0, 0)
	require.NoError(t, err, "EnsureRunning must not error when starting")
	assert.True(t, IsRunning(), "IsRunning should return true after EnsureRunning")

	err = EnsureRunning(0, 0)
	require.NoError(t, err, "EnsureRunning must not error when called twice")

	assert.NoError(t, Stop(), "Stop should not error")
}

func TestStartStop(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	assert.False(t, d.isRunning(), "IsRunning should return false")

	err := d.stop()
	assert.ErrorIs(t, err, common.ErrNilPointer, "stop should error correctly")

	err = d.start(10, 0)
	assert.ErrorIs(t, err, common.ErrNilPointer, "start should error correctly")

	d = NewDispatcher()

	err = d.stop()
	assert.ErrorIs(t, err, ErrNotRunning, "stop should error correctly")
	assert.False(t, d.isRunning(), "IsRunning should return false")

	err = d.start(1, 100)
	assert.NoError(t, err, "start should not error")
	assert.True(t, d.isRunning(), "IsRunning should return true")

	err = d.start(0, 0)
	assert.ErrorIs(t, err, ErrDispatcherAlreadyRunning, "start should error correctly")

	// Add route option
	id, err := d.getNewID(uuid.NewV4)
	assert.NoError(t, err, "getNewID should not error")

	// Add pipe
	_, err = d.subscribe(id)
	assert.NoError(t, err, "subscribe should not error")

	// Max out jobs channel
	for range 99 {
		err = d.publish(id, "woah-nelly")
		assert.NoError(t, err, "publish should not error")
	}

	err = d.stop()
	assert.NoError(t, err, "stop should not error")
	assert.False(t, d.isRunning(), "IsRunning should return false")

	t.Run("stop counts out the pipes it closes", func(t *testing.T) {
		t.Parallel()
		d := NewDispatcher()
		require.NoError(t, d.start(0, 0), "start must not error")
		t.Cleanup(func() {
			if d.isRunning() {
				assert.NoError(t, d.stop(), "stop should not error")
			}
		})
		mux := GetNewMux(d)
		id, err := mux.GetID()
		require.NoError(t, err, "GetID must not error")
		pipes := make([]Pipe, 2)
		for i := range pipes {
			pipes[i], err = mux.Subscribe(id)
			require.NoError(t, err, "Subscribe must not error")
		}
		require.Equal(t, int32(2), d.subscriberCount.Load(), "both subscriptions must be counted")

		require.NoError(t, d.stop(), "stop must not error")
		for i := range pipes {
			select {
			case _, ok := <-pipes[i].c:
				assert.Falsef(t, ok, "stop should close pipe %d without sending to it", i)
			default:
				assert.Failf(t, "stop should close every pipe", "pipe %d is still open", i)
			}
		}
		assert.Zero(t, d.subscriberCount.Load(), "stop should count out every pipe it closes")
		require.NoError(t, pipes[0].Release(), "Release must not error while stopped")
		assert.Zero(t, d.subscriberCount.Load(), "a pipe released while stopped should stay counted out")

		require.NoError(t, d.start(0, 0), "start must not error")
		assert.ErrorIs(t, pipes[1].Release(), errChannelNotFoundInUUIDRef, "a pipe closed by stop should be unknown after a restart")
		assert.Zero(t, d.subscriberCount.Load(), "a pipe closed by stop should stay counted out after a restart")
		p, err := mux.Subscribe(id)
		require.NoError(t, err, "Subscribe must not error")
		assert.Equal(t, int32(1), d.subscriberCount.Load(), "a new subscription should be counted")
		require.NoError(t, p.Release(), "Release must not error")
		assert.Zero(t, d.subscriberCount.Load(), "a released subscription should be counted out")
	})
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher
	_, err := d.subscribe(uuid.Nil())
	assert.ErrorIs(t, err, common.ErrNilPointer, "subscribe should error correctly")

	d = NewDispatcher()

	_, err = d.subscribe(uuid.Nil())
	assert.ErrorIs(t, err, errIDNotSet, "subscribe should error correctly")

	_, err = d.subscribe(nonEmptyUUID)
	assert.ErrorIs(t, err, ErrNotRunning, "subscribe should error correctly")

	err = d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID must not error")

	_, err = d.subscribe(nonEmptyUUID)
	assert.ErrorIs(t, err, errDispatcherUUIDNotFoundInRouteList, "subscribe should error correctly")

	d.outbound.New = func() any { return "omg" }
	_, err = d.subscribe(id)
	assert.ErrorIs(t, err, errTypeAssertionFailure, "subscribe should error correctly")

	d.outbound.New = func() any { return make(chan any) }
	ch, err := d.subscribe(id)
	assert.NoError(t, err, "subscribe should not error")
	assert.NotNil(t, ch, "Channel should not be nil")
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.unsubscribe(uuid.Nil(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "unsubscribe should error correctly")

	d = NewDispatcher()

	err = d.unsubscribe(uuid.Nil(), nil)
	assert.ErrorIs(t, err, errIDNotSet, "unsubscribe should error correctly")

	err = d.unsubscribe(nonEmptyUUID, nil)
	assert.ErrorIs(t, err, errChannelIsNil, "unsubscribe should error correctly")

	// will return nil if not running
	err = d.unsubscribe(nonEmptyUUID, make(chan any))
	assert.NoError(t, err, "unsubscribe should not error")

	err = d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	err = d.unsubscribe(nonEmptyUUID, make(chan any))
	assert.ErrorIs(t, err, errDispatcherUUIDNotFoundInRouteList, "unsubscribe should error correctly")

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID must not error")

	err = d.unsubscribe(id, make(chan any))
	assert.ErrorIs(t, err, errChannelNotFoundInUUIDRef, "unsubscribe should error correctly")

	ch, err := d.subscribe(id)
	require.NoError(t, err, "subscribe must not error")

	err = d.unsubscribe(id, ch)
	assert.NoError(t, err, "unsubscribe should not error")

	ch2, err := d.subscribe(id)
	require.NoError(t, err, "subscribe must not error")

	err = d.unsubscribe(id, ch2)
	assert.NoError(t, err, "unsubscribe should not error")
}

func TestPublish(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.publish(uuid.Nil(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "publish should error correctly")

	d = NewDispatcher()

	err = d.publish(nonEmptyUUID, "test")
	assert.NoError(t, err, "publish should not error")

	err = d.start(2, 10)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	err = d.publish(uuid.Nil(), "test")
	assert.ErrorIs(t, err, errIDNotSet, "publish should error correctly")

	err = d.publish(nonEmptyUUID, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "publish should error correctly")

	// demonstrate job limit error
	d.routes[nonEmptyUUID] = []chan any{
		make(chan any),
	}
	for range 200 {
		if err = d.publish(nonEmptyUUID, "test"); err != nil {
			break
		}
	}
	assert.ErrorIs(t, err, errDispatcherJobsAtLimit, "publish should eventually error at limit")
}

func TestPublishReceive(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID must not error")

	incoming, err := d.subscribe(id)
	require.NoError(t, err, "subscribe must not error")

	publishResult := make(chan error, 1)
	go func(d *Dispatcher, id uuid.UUID) {
		for range 10 {
			if err := d.publish(id, "WOW"); err != nil {
				publishResult <- err
				return
			}
		}
		publishResult <- nil
	}(d, id)

	assert.NoError(t, <-publishResult, "publish should not error")
	data, ok := (<-incoming).(string)
	assert.True(t, ok, "Should get a string type from the pipe")
	assert.Equal(t, "WOW", data, "Should get correct value from the pipe")
}

func TestGetNewID(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	_, err := d.getNewID(uuid.NewV4)
	assert.ErrorIs(t, err, common.ErrNilPointer, "getNewID should error correctly")

	d = NewDispatcher()

	err = d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	_, err = d.getNewID(nil)
	assert.ErrorIs(t, err, errUUIDGeneratorFunctionIsNil, "getNewID should error correctly")

	_, err = d.getNewID(func() uuid.UUID { return uuid.UUID{254} })
	assert.NoError(t, err, "getNewID should not error")

	_, err = d.getNewID(func() uuid.UUID { return uuid.UUID{254} })
	assert.ErrorIs(t, err, errUUIDCollision, "getNewID should error correctly")
}

func TestMux(t *testing.T) {
	t.Parallel()
	var mux *Mux
	_, err := mux.Subscribe(uuid.Nil())
	assert.ErrorIs(t, err, common.ErrNilPointer, "Subscribe should error correctly")

	err = mux.Unsubscribe(uuid.Nil(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Unsubscribe should error correctly")

	err = mux.Publish(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Publish should error correctly")

	_, err = mux.GetID()
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetID should error correctly")

	d := NewDispatcher()
	err = d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })

	mux = GetNewMux(d)

	err = mux.Publish(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Publish should error correctly")

	err = mux.Publish("lol")
	assert.ErrorIs(t, err, errNoIDs, "Publish should error correctly")

	id, err := mux.GetID()
	require.NoError(t, err, "GetID must not error")

	_, err = mux.Subscribe(uuid.Nil())
	assert.ErrorIs(t, err, errIDNotSet, "Subscribe should error correctly")

	pipe, err := mux.Subscribe(id)
	require.NoError(t, err, "Subscribe must not error")

	ready := make(chan bool)

	payload := "string"

	readErr := make(chan error, 1)
	go func() {
		close(ready)
		response, ok := (<-pipe.c).(string)
		if !ok {
			readErr <- errors.New("Should get a string type value from Publish")
			return
		}
		if response != payload {
			readErr <- fmt.Errorf("Should get correct value from Publish: expected %q got %q", payload, response)
			return
		}
		readErr <- nil
	}()

	<-ready

	err = mux.Publish(payload, id)
	assert.NoError(t, err, "Publish should not error")
	assert.NoError(t, <-readErr, "pipe read should not error")

	err = pipe.Release()
	assert.NoError(t, err, "Release should not error")
}

func TestMuxSubscribe(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	require.NoError(t, err, "start must not error")
	t.Cleanup(func() { assert.NoError(t, d.stop(), "stop should not error") })
	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	require.NoError(t, err, "GetID must not error")

	pipes := make([]Pipe, 1000)
	for x := range 1000 {
		newPipe, err := mux.Subscribe(itemID)
		assert.NoError(t, err, "Subscribe should not error")
		pipes[x] = newPipe
	}

	for i := range pipes {
		err := pipes[i].Release()
		assert.NoError(t, err, "Release should not error")
	}
}

func TestMuxPublish(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	require.NoError(t, err, "start must not error")

	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	require.NoError(t, err, "GetID must not error")

	overloadCeiling := DefaultMaxWorkers * DefaultJobsLimit * 2

	for range overloadCeiling {
		err = mux.Publish("test", itemID)
		if !assert.NoError(t, err, "Publish should not error when over limit but no listeners") {
			break
		}
	}

	ready := make(chan any)
	demux := make(chan any, 1)
	pipe, err := mux.Subscribe(itemID)
	require.NoError(t, err, "Subscribe must not error")

	// Subscribers must be actively selecting in order to receive anything
	go func() {
		close(ready)
		i := <-pipe.c
		demux <- i
		close(demux)
	}()

	publishWithinLimitsErr := make(chan error, 1)
	go func() {
		<-ready // Ensure listener is ready before starting
		for i := range 100 {
			if errMux := mux.Publish(i, itemID); errMux != nil {
				publishWithinLimitsErr <- errMux
				return
			}
		}
		publishWithinLimitsErr <- nil
	}()

	assert.Eventually(t, func() bool { return len(demux) >= 1 }, time.Second, time.Millisecond*10, "Subscriber should eventually get at least one message")
	assert.NoError(t, <-publishWithinLimitsErr, "Publish should not error within limits")

	// demonstrate that jobs can be limited when subscribed
	// Published data gets consumed from .jobs to the worker channels, so we're looking to push more than it's consumed and prevent the select reading them too quickly
	runtime.LockOSThread()
	for range overloadCeiling {
		if err = mux.Publish("test", itemID); err != nil {
			break
		}
	}
	assert.ErrorIs(t, err, errDispatcherJobsAtLimit, "Publish should error when more published than expected")
	runtime.UnlockOSThread()

	err = mux.Unsubscribe(itemID, pipe.c)
	assert.NoError(t, err, "Unsubscribe should not error")

	for range overloadCeiling {
		if err = mux.Publish("test", itemID); err != nil {
			break
		}
	}
	assert.NoError(t, err, "Publish should not error after Unsubscribe when over limit")

	// Shut down dispatch system
	err = d.stop()
	assert.NoError(t, err, "stop should not error")
}

func TestPipeWait(t *testing.T) {
	t.Parallel()

	waitContext := func(t *testing.T) context.Context {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		t.Cleanup(cancel)
		return ctx
	}

	liveContext := func(t *testing.T) context.Context {
		t.Helper()
		return t.Context()
	}

	doneContext := func(t *testing.T) context.Context {
		t.Helper()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		return ctx
	}

	// waitBounded runs wait on another goroutine and requires it to return within five seconds
	waitBounded := func(t *testing.T, wait func() (string, error)) (string, error) {
		t.Helper()
		type result struct {
			got string
			err error
		}
		done := make(chan result, 1)
		go func() {
			got, err := wait()
			done <- result{got: got, err: err}
		}()
		select {
		case r := <-done:
			return r.got, r.err
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Wait must return within five seconds")
			return "", nil
		}
	}

	t.Run("returns the next value as the requested type", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any, 1)}
		p.c <- "update"
		got, err := p.Wait[string](waitContext(t))
		require.NoError(t, err, "Wait must not error")
		assert.Equal(t, "update", got, "Wait should return the next value")
	})

	t.Run("returns the next value as an interface it implements", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any, 1)}
		p.c <- time.Second
		got, err := p.Wait[fmt.Stringer](waitContext(t))
		require.NoError(t, err, "Wait must not error")
		assert.Equal(t, time.Second, got, "Wait should return the next value")
	})

	t.Run("returns values in the order received", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any, 2)}
		p.c <- "first"
		p.c <- "second"
		ctx := waitContext(t)
		for _, want := range []string{"first", "second"} {
			got, err := p.Wait[string](ctx)
			require.NoError(t, err, "Wait must not error")
			assert.Equal(t, want, got, "Wait should return the values in the order received")
		}
	})

	t.Run("value of another type than the interface requested", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any, 1)}
		p.c <- "update"
		got, err := p.Wait[fmt.Stringer](waitContext(t))
		assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "Wait should return a type assertion error")
		assert.EqualError(t, err, "type assert failure from string to fmt.Stringer", "Wait should name the value's type and the requested type")
		assert.Nil(t, got, "Wait should return the zero value")
	})

	t.Run("value of another type than the pointer requested", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any, 1)}
		p.c <- 1
		got, err := p.Wait[*int](waitContext(t))
		assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "Wait should return a type assertion error")
		assert.EqualError(t, err, "type assert failure from int to *int", "Wait should name the value's type and the requested type")
		assert.Nil(t, got, "Wait should return the zero value")
	})

	t.Run("closed pipe", func(t *testing.T) {
		t.Parallel()
		p := Pipe{c: make(chan any)}
		close(p.c)
		got, err := p.Wait[string](waitContext(t))
		assert.ErrorIs(t, err, ErrPipeClosed, "Wait should return ErrPipeClosed")
		assert.Same(t, ErrPipeClosed, err, "Wait should return ErrPipeClosed itself")
		assert.Empty(t, got, "Wait should return the zero value")
	})

	t.Run("context already done", func(t *testing.T) {
		t.Parallel()
		ctx := doneContext(t)
		p := Pipe{c: make(chan any)}
		_, err := waitBounded(t, func() (string, error) { return p.Wait[string](ctx) })
		assert.ErrorIs(t, err, context.Canceled, "Wait should return the context's error")
	})

	t.Run("deadline passes while waiting", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		p := Pipe{c: make(chan any)}
		_, err := waitBounded(t, func() (string, error) { return p.Wait[string](ctx) })
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Wait should return the context's error")
	})

	for _, tc := range []struct {
		name    string
		context func(*testing.T) context.Context
	}{
		{name: "live context", context: liveContext},
		{name: "context already done", context: doneContext},
	} {
		t.Run("pipe without a channel/"+tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := tc.context(t)
			var p Pipe
			got, err := waitBounded(t, func() (string, error) { return p.Wait[string](ctx) })
			assert.ErrorIs(t, err, errChannelIsNil, "Wait should return errChannelIsNil")
			assert.Empty(t, got, "Wait should return the zero value")
		})

		t.Run("nil pipe/"+tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := tc.context(t)
			var p *Pipe
			got, err := waitBounded(t, func() (string, error) { return p.Wait[string](ctx) })
			assert.ErrorIs(t, err, common.ErrNilPointer, "Wait should return common.ErrNilPointer")
			assert.Empty(t, got, "Wait should return the zero value")
		})
	}

	t.Run("published value then dispatcher stopped", func(t *testing.T) {
		t.Parallel()
		d := NewDispatcher()
		require.NoError(t, d.start(0, 0), "start must not error")
		t.Cleanup(func() {
			if d.isRunning() {
				assert.NoError(t, d.stop(), "stop should not error")
			}
		})
		mux := GetNewMux(d)
		id, err := mux.GetID()
		require.NoError(t, err, "GetID must not error")
		p, err := mux.Subscribe(id)
		require.NoError(t, err, "Subscribe must not error")
		require.NoError(t, mux.Publish("update", id), "Publish must not error")

		ctx := waitContext(t)
		got, err := p.Wait[string](ctx)
		require.NoError(t, err, "Wait must not error")
		assert.Equal(t, "update", got, "Wait should return the published value")

		require.NoError(t, d.stop(), "stop must not error")
		_, err = p.Wait[string](ctx)
		assert.ErrorIs(t, err, ErrPipeClosed, "Wait should return ErrPipeClosed once the dispatcher has stopped")
	})
}

func BenchmarkSubscribe(b *testing.B) {
	d := NewDispatcher()
	err := d.start(DefaultMaxWorkers, DefaultJobsLimit)
	require.NoError(b, err, "start must not error")
	// Each sample starts more relayers, which would otherwise stay blocked for the run
	b.Cleanup(func() { assert.NoError(b, d.stop(), "stop should not error") })
	mux := GetNewMux(d)
	newID, err := mux.GetID()
	require.NoError(b, err, "GetID must not error")

	// Unsubscribing keeps the subscriber slice a steady length, as its growth moves B/op between runs
	for b.Loop() {
		pipe, err := mux.Subscribe(newID)
		if err != nil {
			b.Error(err)
			continue
		}
		if err := pipe.Release(); err != nil {
			b.Error(err)
		}
	}
}
