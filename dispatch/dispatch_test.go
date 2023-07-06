package dispatch

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
)

var (
	errTest      = errors.New("test error")
	nonEmptyUUID = [uuid.Size]byte{108, 105, 99, 107, 77, 121, 72, 97, 105, 114, 121, 66, 97, 108, 108, 115}
)

func TestGlobalDispatcher(t *testing.T) {
	err := Start(0, 0)
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	running := IsRunning()
	if !running {
		t.Fatalf("received: '%v' but expected: '%v'", IsRunning(), true)
	}

	err = Stop()
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	running = IsRunning()
	if running {
		t.Fatalf("received: '%v' but expected: '%v'", IsRunning(), false)
	}
}

func TestStartStop(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	if d.isRunning() {
		t.Fatalf("received: '%v' but expected: '%v'", d.isRunning(), false)
	}

	err := d.stop()
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.start(10, 0)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = NewDispatcher()

	err = d.stop()
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotRunning)
	}

	if d.isRunning() {
		t.Fatalf("received: '%v' but expected: '%v'", d.isRunning(), false)
	}

	err = d.start(1, 100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !d.isRunning() {
		t.Fatalf("received: '%v' but expected: '%v'", d.isRunning(), true)
	}

	err = d.start(0, 0)
	if !errors.Is(err, errDispatcherAlreadyRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherAlreadyRunning)
	}

	// Add route option
	id, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Add pipe
	_, err = d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Max out jobs channel
	for x := 0; x < 99; x++ {
		err = d.publish(id, "woah-nelly")
		if !errors.Is(err, nil) {
			t.Fatalf("received: '%v' but expected: '%v'", err, nil)
		}
	}

	err = d.stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if d.isRunning() {
		t.Fatalf("received: '%v' but expected: '%v'", d.isRunning(), false)
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher
	_, err := d.subscribe(uuid.Nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = NewDispatcher()

	_, err = d.subscribe(uuid.Nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	_, err = d.subscribe(nonEmptyUUID)
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotRunning)
	}

	err = d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	id, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = d.subscribe(nonEmptyUUID)
	if !errors.Is(err, errDispatcherUUIDNotFoundInRouteList) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherUUIDNotFoundInRouteList)
	}

	d.outbound.New = func() interface{} { return "omg" }
	_, err = d.subscribe(id)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTypeAssertionFailure)
	}

	d.outbound.New = getChan
	ch, err := d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ch == nil {
		t.Fatal("expected channel value")
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.unsubscribe(uuid.Nil, nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = NewDispatcher()

	err = d.unsubscribe(uuid.Nil, nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	err = d.unsubscribe(nonEmptyUUID, nil)
	if !errors.Is(err, errChannelIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errChannelIsNil)
	}

	// will return nil if not running
	err = d.unsubscribe(nonEmptyUUID, make(chan interface{}))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.unsubscribe(nonEmptyUUID, make(chan interface{}))
	if !errors.Is(err, errDispatcherUUIDNotFoundInRouteList) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherUUIDNotFoundInRouteList)
	}

	id, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.unsubscribe(id, make(chan interface{}))
	if !errors.Is(err, errChannelNotFoundInUUIDRef) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errChannelNotFoundInUUIDRef)
	}

	// Skip over this when matching pipes
	_, err = d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ch, err := d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.unsubscribe(id, ch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ch2, err := d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.unsubscribe(id, ch2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestPublish(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	err := d.publish(uuid.Nil, nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = NewDispatcher()

	err = d.publish(nonEmptyUUID, "test")
	if !errors.Is(err, nil) { // If not running, don't send back an error.
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.start(2, 10)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish(uuid.Nil, nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	err = d.publish(nonEmptyUUID, nil)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	// demonstrate job limit error
	d.routes[nonEmptyUUID] = []chan interface{}{
		make(chan interface{}),
	}
	for x := 0; x < 200; x++ {
		err2 := d.publish(nonEmptyUUID, "test")
		if !errors.Is(err2, nil) {
			err = err2
			break
		}
	}
	if !errors.Is(err, errDispatcherJobsAtLimit) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherJobsAtLimit)
	}
}

func TestPublishReceive(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	if err := d.start(0, 0); !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	id, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	incoming, err := d.subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	go func(d *Dispatcher, id uuid.UUID) {
		for x := 0; x < 10; x++ {
			err2 := d.publish(id, "WOW")
			if !errors.Is(err2, nil) {
				panic(err2)
			}
		}
	}(d, id)

	data, ok := (<-incoming).(string)
	if !ok {
		t.Fatal("type assertion failure expected string")
	}

	if data != "WOW" {
		t.Fatal("unexpected value")
	}
}

func TestGetNewID(t *testing.T) {
	t.Parallel()
	var d *Dispatcher

	_, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = NewDispatcher()

	err = d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = d.getNewID(nil)
	if !errors.Is(err, errUUIDGeneratorFunctionIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUUIDGeneratorFunctionIsNil)
	}

	_, err = d.getNewID(func() (uuid.UUID, error) { return uuid.Nil, errTest })
	if !errors.Is(err, errTest) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTest)
	}

	_, err = d.getNewID(func() (uuid.UUID, error) { return [uuid.Size]byte{254}, nil })
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = d.getNewID(func() (uuid.UUID, error) { return [uuid.Size]byte{254}, nil })
	if !errors.Is(err, errUUIDCollision) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUUIDCollision)
	}
}

func TestMux(t *testing.T) {
	t.Parallel()
	var mux *Mux
	_, err := mux.Subscribe(uuid.Nil)
	if !errors.Is(err, errMuxIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMuxIsNil)
	}

	err = mux.Unsubscribe(uuid.Nil, nil)
	if !errors.Is(err, errMuxIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMuxIsNil)
	}

	err = mux.Publish(nil)
	if !errors.Is(err, errMuxIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMuxIsNil)
	}

	_, err = mux.GetID()
	if !errors.Is(err, errMuxIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMuxIsNil)
	}

	d := NewDispatcher()
	err = d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	mux = GetNewMux(d)

	err = mux.Publish(nil)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	err = mux.Publish("lol")
	if !errors.Is(err, errNoIDs) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoIDs)
	}

	id, err := mux.GetID()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = mux.Subscribe(uuid.Nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	pipe, err := mux.Subscribe(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	var errChan = make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	// Makes sure receiver is waiting for update
	go func(ch <-chan interface{}, errChan chan error, wg *sync.WaitGroup) {
		wg.Done()
		response, ok := (<-ch).(string)
		if !ok {
			errChan <- errors.New("type assertion failure")
			return
		}

		if response != "string" {
			errChan <- errors.New("unexpected return")
			return
		}
		errChan <- nil
	}(pipe.c, errChan, &wg)

	wg.Wait()

	payload := "string"
	go func(payload string) {
		err2 := mux.Publish(payload, id)
		if err2 != nil {
			fmt.Println(err2)
		}
	}(payload)

	err = <-errChan
	if err != nil {
		t.Fatal(err)
	}

	err = pipe.Release()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestMuxSubscribe(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	if err != nil {
		t.Fatal(err)
	}

	var pipes []Pipe
	for i := 0; i < 1000; i++ {
		newPipe, err := mux.Subscribe(itemID)
		if err != nil {
			t.Error(err)
		}
		pipes = append(pipes, newPipe)
	}

	for i := range pipes {
		err := pipes[i].Release()
		if err != nil {
			t.Error(err)
		}
	}
}

func TestMuxPublish(t *testing.T) {
	t.Parallel()
	d := NewDispatcher()
	err := d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	if err != nil {
		t.Fatal(err)
	}

	// demonstrate that jobs do not get published when the limit should be reached
	// but there is no listener associated with job
	for x := 0; x < 200; x++ {
		err2 := mux.Publish("test", itemID)
		if !errors.Is(err2, nil) {
			err = err2
			break
		}
	}
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	pipe, err := mux.Subscribe(itemID)
	if err != nil {
		t.Error(err)
	}

	go func(mux *Mux) {
		for i := 0; i < 100; i++ {
			errMux := mux.Publish(i, itemID)
			if errMux != nil {
				t.Error(errMux)
			}
		}
	}(mux)

	<-pipe.Channel()

	// demonstrate that jobs can be limited when subscribed
	for x := 0; x < 200; x++ {
		err2 := mux.Publish("test", itemID)
		if !errors.Is(err2, nil) {
			err = err2
			break
		}
	}
	if !errors.Is(err, errDispatcherJobsAtLimit) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherJobsAtLimit)
	}

	// demonstrate that jobs go back to not being sent after unsubscribing
	err = mux.Unsubscribe(itemID, pipe.c)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	for x := 0; x < 200; x++ {
		err2 := mux.Publish("test", itemID)
		if !errors.Is(err2, nil) {
			err = err2
			break
		}
	}
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Shut down dispatch system
	err = d.stop()
	if err != nil {
		t.Fatal(err)
	}
}

// 13636467	        84.26 ns/op	     141 B/op	       1 allocs/op
func BenchmarkSubscribe(b *testing.B) {
	d := NewDispatcher()
	err := d.start(0, 0)
	if !errors.Is(err, nil) {
		b.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	mux := GetNewMux(d)
	newID, err := mux.GetID()
	if err != nil {
		b.Error(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := mux.Subscribe(newID)
		if err != nil {
			b.Error(err)
		}
	}
}
