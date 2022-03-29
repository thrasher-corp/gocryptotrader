package dispatch

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/gofrs/uuid"
)

var errTest = errors.New("test error")

func TestMain(m *testing.M) {
	err := Start(0, 0)
	if err != nil {
		fmt.Println("test main:", err)
		os.Exit(1)
	}

	running := IsRunning()
	if !running {
		fmt.Println("test main: Should be running")
		os.Exit(1)
	}

	err = DropWorker()
	if !errors.Is(err, nil) {
		fmt.Println("test main:", err)
		os.Exit(1)
	}

	err = SpawnWorker()
	if !errors.Is(err, nil) {
		fmt.Println("test main:", err)
		os.Exit(1)
	}

	err = Stop()
	if !errors.Is(err, nil) {
		fmt.Println("test main:", err)
		os.Exit(1)
	}

	err = Start(0, 0)
	if err != nil {
		fmt.Println("test main:", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestDispatcher(t *testing.T) {
	t.Parallel()
	var d *Dispatcher
	err := d.stop()
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.start(10, 0)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	if d.isRunning() {
		t.Fatal("should be false")
	}

	err = d.dropWorker()
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.spawnWorker()
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.publish(uuid.Nil, nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	_, err = d.subscribe(uuid.Nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.unsubscribe(uuid.Nil, nil)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	_, err = d.getNewID(uuid.NewV4)
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	d = newDispatcher()

	err = d.stop()
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotRunning)
	}

	d.count = 3
	err = d.start(10, 0)
	if !errors.Is(err, errLeakedWorkers) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errLeakedWorkers)
	}
	d.count = 0

	if d.isRunning() {
		t.Fatalf("received: '%v' but expected: '%v'", d.isRunning(), false)
	}

	err = d.dropWorker()
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotRunning)
	}

	err = d.spawnWorker()
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotRunning)
	}

	err = d.start(1, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.start(0, 0)
	if !errors.Is(err, errDispatcherAlreadyRunning) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherAlreadyRunning)
	}

	err = d.dropWorker()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.dropWorker()
	if !errors.Is(err, errNoWorkers) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoWorkers)
	}

	err = d.spawnWorker()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.spawnWorker()
	if !errors.Is(err, errWorkerCeilingReached) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWorkerCeilingReached)
	}

	err = d.stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish([uuid.Size]byte{255}, "lol")
	if !errors.Is(err, nil) { // If not running, don't send back an error.
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = d.subscribe(uuid.Nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	_, err = d.subscribe([uuid.Size]byte{255})
	if !errors.Is(err, errDispatcherNotInitialized) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherNotInitialized)
	}

	err = d.unsubscribe(uuid.Nil, nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	err = d.unsubscribe([uuid.Size]byte{255}, nil)
	if !errors.Is(err, errChannelIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errChannelIsNil)
	}

	// will return nil if not running
	err = d.unsubscribe([uuid.Size]byte{255}, make(<-chan interface{}))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.start(1, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish(uuid.Nil, nil)
	if !errors.Is(err, errIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDNotSet)
	}

	err = d.publish([uuid.Size]byte{255}, nil)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	err = d.dropWorker()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish([uuid.Size]byte{255}, "lol")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish([uuid.Size]byte{255}, "lol")
	if !errors.Is(err, errDispatcherJobsAtLimit) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherJobsAtLimit)
	}

	err = d.spawnWorker()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
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

	id, err := d.getNewID(uuid.NewV4)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = d.subscribe([uuid.Size]byte{255})
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

	go func() {
		err = d.publish(id, "lol")
		if err != nil {
			fmt.Println(err)
		}
	}()

	response, ok := (<-ch).(string)
	if !ok {
		t.Fatal("type assertion failure")
	}

	if response != "lol" {
		t.Fatal("unexpected return")
	}

	// publish no receiver
	err = d.publish(id, "lol")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.unsubscribe([uuid.Size]byte{255}, make(<-chan interface{}))
	if !errors.Is(err, errDispatcherUUIDNotFoundInRouteList) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDispatcherUUIDNotFoundInRouteList)
	}

	err = d.unsubscribe(id, make(<-chan interface{}))
	if !errors.Is(err, errChannelNotFoundInUUIDRef) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errChannelNotFoundInUUIDRef)
	}

	err = d.unsubscribe(id, ch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.dropWorker()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.publish(id, "lol") // publish no worker
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = d.start(1, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
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

	d := newDispatcher()
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

	payload := "string"
	go func() {
		err = mux.Publish(payload, id)
		if err != nil {
			fmt.Println(err)
		}
	}()

	response, ok := (<-pipe.C).(string)
	if !ok {
		t.Fatal("type assertion failure")
	}

	if response != payload {
		t.Fatalf("received: '%v' but expected: '%v'", response, payload)
	}

	err = pipe.Release()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	d := newDispatcher()
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

func TestPublish(t *testing.T) {
	t.Parallel()
	d := newDispatcher()
	err := d.start(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	mux := GetNewMux(d)
	itemID, err := mux.GetID()
	if err != nil {
		t.Fatal(err)
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

	<-pipe.C

	// Shut down dispatch system
	err = d.stop()
	if err != nil {
		t.Fatal(err)
	}
}

// //  8587209	       142.7 ns/op	     145 B/op	       1 allocs/op
// func BenchmarkSubscribe(b *testing.B) {
// 	d := newDispatcher()
// 	err := d.start(0, 0)
// 	if !errors.Is(err, nil) {
// 		b.Fatalf("received: '%v' but expected: '%v'", err, nil)
// 	}
// 	mux := GetNewMux(d)
// 	newID, err := mux.GetID()
// 	if err != nil {
// 		b.Error(err)
// 	}

// 	for n := 0; n < b.N; n++ {
// 		_, err := mux.Subscribe(newID)
// 		if err != nil {
// 			b.Error(err)
// 		}
// 	}
// }
