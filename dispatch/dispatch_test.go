package dispatch

import (
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTest      = errors.New("test error")
	nonEmptyUUID = [uuid.Size]byte{108, 105, 99, 107, 77, 121, 72, 97, 105, 114, 121, 66, 97, 108, 108, 115}
)

func TestGlobalDispatcher(t *testing.T) {
	err := Start(0, 0)
	require.NoError(t, err, "Start should not error")
	assert.True(t, IsRunning(), "IsRunning should return true")

	err = Stop()
	assert.NoError(t, err, "Stop should not error")
	assert.False(t, IsRunning(), "IsRunning should return false")
}

func TestStartStop(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	assert.False(t, d.isRunning(), "IsRunning should return false")

	err := d.stop()
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "stop should error correctly")

	err = d.start(10, 0)
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "start should error correctly")

	d = NewDispatcher()

	err = d.stop()
	assert.ErrorIs(t, err, ErrNotRunning, "stop should error correctly")
	assert.False(t, d.isRunning(), "IsRunning should return false")

	err = d.start(1, 100)
	assert.NoError(t, err, "start should not error")
	assert.True(t, d.isRunning(), "IsRunning should return true")

	err = d.start(0, 0)
	assert.ErrorIs(t, err, errDispatcherAlreadyRunning, "start should error correctly")

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
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher
	_, err := d.subscribe(uuid.Nil)
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "subscribe should error correctly")

	d = NewDispatcher()

	_, err = d.subscribe(uuid.Nil)
	assert.ErrorIs(t, err, errIDNotSet, "subscribe should error correctly")

	_, err = d.subscribe(nonEmptyUUID)
	assert.ErrorIs(t, err, ErrNotRunning, "subscribe should error correctly")

	err = d.start(0, 0)
	require.NoError(t, err, "start should not error")

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID should not error")

	_, err = d.subscribe(nonEmptyUUID)
	assert.ErrorIs(t, err, errDispatcherUUIDNotFoundInRouteList, "subscribe should error correctly")

	d.outbound.New = func() interface{} { return "omg" }
	_, err = d.subscribe(id)
	assert.ErrorIs(t, err, errTypeAssertionFailure, "subscribe should error correctly")

	d.outbound.New = getChan
	ch, err := d.subscribe(id)
	assert.NoError(t, err, "subscribe should not error")
	assert.NotNil(t, ch, "Channel should not be nil")
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.unsubscribe(uuid.Nil, nil)
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "unsubscribe should error correctly")

	d = NewDispatcher()

	err = d.unsubscribe(uuid.Nil, nil)
	assert.ErrorIs(t, err, errIDNotSet, "unsubscribe should error correctly")

	err = d.unsubscribe(nonEmptyUUID, nil)
	assert.ErrorIs(t, err, errChannelIsNil, "unsubscribe should error correctly")

	// will return nil if not running
	err = d.unsubscribe(nonEmptyUUID, make(chan interface{}))
	assert.NoError(t, err, "unsubscribe should not error")

	err = d.start(0, 0)
	require.NoError(t, err, "start should not error")

	err = d.unsubscribe(nonEmptyUUID, make(chan interface{}))
	assert.ErrorIs(t, err, errDispatcherUUIDNotFoundInRouteList, "unsubscribe should error correctly")

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID should not error")

	err = d.unsubscribe(id, make(chan interface{}))
	assert.ErrorIs(t, err, errChannelNotFoundInUUIDRef, "unsubscribe should error correctly")

	ch, err := d.subscribe(id)
	require.NoError(t, err, "subscribe should not error")

	err = d.unsubscribe(id, ch)
	assert.NoError(t, err, "unsubscribe should not error")

	ch2, err := d.subscribe(id)
	require.NoError(t, err, "subscribe should not error")

	err = d.unsubscribe(id, ch2)
	assert.NoError(t, err, "unsubscribe should not error")
}

func TestPublish(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.publish(uuid.Nil, nil)
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "publish should error correctly")

	d = NewDispatcher()

	err = d.publish(nonEmptyUUID, "test")
	assert.NoError(t, err, "publish should not error")

	err = d.start(2, 10)
	require.NoError(t, err, "start should not error")

	err = d.publish(uuid.Nil, nil)
	assert.ErrorIs(t, err, errIDNotSet, "publish should error correctly")

	err = d.publish(nonEmptyUUID, nil)
	assert.ErrorIs(t, err, errNoData, "publish should error correctly")

	// demonstrate job limit error
	d.routes[nonEmptyUUID] = []chan interface{}{
		make(chan interface{}),
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
	require.NoError(t, err, "start should not error")

	id, err := d.getNewID(uuid.NewV4)
	require.NoError(t, err, "getNewID should not error")

	incoming, err := d.subscribe(id)
	require.NoError(t, err, "subscribe should not error")

	go func(d *Dispatcher, id uuid.UUID) {
		for range 10 {
			err := d.publish(id, "WOW")
			assert.NoError(t, err, "publish should not error")
		}
	}(d, id)

	data, ok := (<-incoming).(string)
	assert.True(t, ok, "Should get a string type from the pipe")
	assert.Equal(t, "WOW", data, "Should get correct value from the pipe")
}

func TestGetNewID(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	_, err := d.getNewID(uuid.NewV4)
	assert.ErrorIs(t, err, errDispatcherNotInitialized, "getNewID should error correctly")

	d = NewDispatcher()

	err = d.start(0, 0)
	require.NoError(t, err, "start should not error")

	_, err = d.getNewID(nil)
	assert.ErrorIs(t, err, errUUIDGeneratorFunctionIsNil, "getNewID should error correctly")

	_, err = d.getNewID(func() (uuid.UUID, error) { return uuid.Nil, errTest })
	assert.ErrorIs(t, err, errTest, "getNewID should error correctly")

	_, err = d.getNewID(func() (uuid.UUID, error) { return [uuid.Size]byte{254}, nil })
	assert.NoError(t, err, "getNewID should not error")

	_, err = d.getNewID(func() (uuid.UUID, error) { return [uuid.Size]byte{254}, nil })
	assert.ErrorIs(t, err, errUUIDCollision, "getNewID should error correctly")
}

func TestMux(t *testing.T) {
	t.Parallel()
	var mux *Mux
	_, err := mux.Subscribe(uuid.Nil)
	assert.ErrorIs(t, err, errMuxIsNil, "Subscribe should error correctly")

	err = mux.Unsubscribe(uuid.Nil, nil)
	assert.ErrorIs(t, err, errMuxIsNil, "Unsubscribe should error correctly")

	err = mux.Publish(nil)
	assert.ErrorIs(t, err, errMuxIsNil, "Publish should error correctly")

	_, err = mux.GetID()
	assert.ErrorIs(t, err, errMuxIsNil, "GetID should error correctly")

	d := NewDispatcher()
	err = d.start(0, 0)
	require.NoError(t, err, "start should not error")

	mux = GetNewMux(d)

	err = mux.Publish(nil)
	assert.ErrorIs(t, err, errNoData, "Publish should error correctly")

	err = mux.Publish("lol")
	assert.ErrorIs(t, err, errNoIDs, "Publish should error correctly")

	id, err := mux.GetID()
	require.NoError(t, err, "GetID should not error")

	_, err = mux.Subscribe(uuid.Nil)
	assert.ErrorIs(t, err, errIDNotSet, "Subscribe should error correctly")

	pipe, err := mux.Subscribe(id)
	require.NoError(t, err, "Subscribe should not error")

	var ready = make(chan bool)

	payload := "string"

	go func() {
		close(ready)
		response, ok := (<-pipe.c).(string)
		assert.True(t, ok, "Should get a string type value from Publish")
		assert.Equal(t, payload, response, "Should get correct value from Publish")
	}()

	<-ready

	err = mux.Publish(payload, id)
	assert.NoError(t, err, "Publish should not error")

	err = pipe.Release()
	assert.NoError(t, err, "Release should not error")
}

func TestMuxSubscribe(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	require.NoError(t, err, "start should not error")
	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	require.NoError(t, err, "GetID should not error")

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
	require.NoError(t, err, "start should not error")

	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	require.NoError(t, err, "GetID should not error")

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
	require.NoError(t, err, "Subscribe should not error")

	// Subscribers must be actively selecting in order to receive anything
	go func() {
		close(ready)
		i := <-pipe.c
		demux <- i
		close(demux)
	}()

	go func() {
		<-ready // Ensure listener is ready before starting
		for i := range 100 {
			errMux := mux.Publish(i, itemID)
			if !assert.NoError(t, errMux, "Publish should not error within limits") {
				return
			}
		}
	}()

	assert.Eventually(t, func() bool { return len(demux) >= 1 }, time.Second, time.Millisecond*10, "Subscriber should eventually get at least one message")

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

// 13636467	        84.26 ns/op	     141 B/op	       1 allocs/op
func BenchmarkSubscribe(b *testing.B) {
	d := NewDispatcher()
	err := d.start(0, 0)
	require.NoError(b, err, "start should not error")
	mux := GetNewMux(d)
	newID, err := mux.GetID()
	require.NoError(b, err, "GetID should not error")

	for n := 0; n < b.N; n++ {
		_, err := mux.Subscribe(newID)
		if err != nil {
			b.Error(err)
		}
	}
}
