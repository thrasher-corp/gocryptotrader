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
	err := Start(DefaultMaxWorkers)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	mux = GetNewMux()
	os.Exit(m.Run())
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
