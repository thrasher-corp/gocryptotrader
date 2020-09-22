package dispatch

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
)

var mux *Mux

func TestMain(m *testing.M) {
	err := Start(DefaultMaxWorkers, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cpyDispatch = dispatcher
	mux = GetNewMux()
	cpyMux = mux
	os.Exit(m.Run())
}

var cpyDispatch *Dispatcher
var cpyMux *Mux

func TestDispatcher(t *testing.T) {
	dispatcher = nil
	err := Stop()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = Start(10, 0)
	if err == nil {
		t.Error("error cannot be nil")
	}
	if IsRunning() {
		t.Error("should be false")
	}

	err = DropWorker()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = SpawnWorker()
	if err == nil {
		t.Error("error cannot be nil")
	}

	dispatcher = cpyDispatch

	if !IsRunning() {
		t.Error("should be true")
	}

	err = Start(10, 0)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = DropWorker()
	if err != nil {
		t.Error(err)
	}

	err = DropWorker()
	if err != nil {
		t.Error(err)
	}

	err = SpawnWorker()
	if err != nil {
		t.Error(err)
	}

	err = SpawnWorker()
	if err != nil {
		t.Error(err)
	}

	err = SpawnWorker()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = Stop()
	if err != nil {
		t.Error(err)
	}

	err = Stop()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = Start(0, 20)
	if err != nil {
		t.Error(err)
	}
	if cap(dispatcher.jobs) != 20 {
		t.Errorf("Expected jobs limit to be %v, is %v", 20, cap(dispatcher.jobs))
	}
	payload := "something"

	err = dispatcher.publish(uuid.UUID{}, &payload)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = dispatcher.publish(uuid.UUID{}, nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	id, err := dispatcher.getNewID()
	if err != nil {
		t.Error(err)
	}

	err = dispatcher.publish(id, &payload)
	if err != nil {
		t.Error(err)
	}

	err = dispatcher.stop()
	if err != nil {
		t.Error(err)
	}

	err = dispatcher.publish(id, &payload)
	if err != nil {
		t.Error(err)
	}

	_, err = dispatcher.subscribe(id)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = dispatcher.start(10, -1)
	if err != nil {
		t.Error(err)
	}
	if cap(dispatcher.jobs) != DefaultJobsLimit {
		t.Errorf("Expected jobs limit to be %v, is %v", DefaultJobsLimit, cap(dispatcher.jobs))
	}
	someID, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}

	_, err = dispatcher.subscribe(someID)
	if err == nil {
		t.Error("error cannot be nil")
	}

	randomChan := make(chan interface{})
	err = dispatcher.unsubscribe(someID, randomChan)
	if err == nil {
		t.Error("Expected error")
	}

	err = dispatcher.unsubscribe(id, randomChan)
	if err == nil {
		t.Error("Expected error")
	}

	close(randomChan)
	err = dispatcher.unsubscribe(id, randomChan)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestMux(t *testing.T) {
	mux = nil
	_, err := mux.Subscribe(uuid.UUID{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = mux.Unsubscribe(uuid.UUID{}, nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = mux.Publish(nil, nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = mux.GetID()
	if err == nil {
		t.Error("error cannot be nil")
	}
	mux = cpyMux

	err = mux.Publish(nil, nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	payload := "string"
	id, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}

	err = mux.Publish([]uuid.UUID{id}, &payload)
	if err != nil {
		t.Error(err)
	}

	_, err = mux.Subscribe(uuid.UUID{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = mux.Subscribe(id)
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestSubscribe(t *testing.T) {
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
	itemID, err := mux.GetID()
	if err != nil {
		t.Fatal(err)
	}

	pipe, err := mux.Subscribe(itemID)
	if err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		wg.Done()
		for {
			_, ok := <-pipe.C
			if !ok {
				pErr := pipe.Release()
				if pErr != nil {
					t.Error(pErr)
				}
				wg.Done()
				return
			}
		}
	}(&wg)
	wg.Wait()
	wg.Add(1)
	mainPayload := "PAYLOAD"
	for i := 0; i < 100; i++ {
		errMux := mux.Publish([]uuid.UUID{itemID}, &mainPayload)
		if errMux != nil {
			t.Error(errMux)
		}
	}

	// Shut down dispatch system
	err = Stop()
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func BenchmarkSubscribe(b *testing.B) {
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
