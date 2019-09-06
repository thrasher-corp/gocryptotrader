package dispatch

import (
	"os"
	"testing"

	"github.com/gofrs/uuid"
)

var mux *Mux

func TestMain(m *testing.M) {
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

	newPipe, err := mux.Subscribe(itemID)
	if err != nil {
		t.Error(err)
	}

	outgoing := make(chan interface{})

	// Set up dumb client
	go func(outgoing chan interface{}, p Pipe) {
		for {
			object := <-p.C
			if _, ok := (*object.(*interface{})).(struct{}); ok {
				outgoing <- p.Release()
			}
			outgoing <- object
		}
	}(outgoing, newPipe)

	mainPayload := "PAYLOAD"
	for i := 0; i < 1000; i++ {
		err := mux.Publish([]uuid.UUID{itemID}, &mainPayload)
		if err != nil {
			t.Error(err)
		}
		if data := <-outgoing; (*data.(*interface{})).(string) != mainPayload {
			t.Error("published object invalid")
		}
	}

	// Shut down dumb client
	err = mux.Publish([]uuid.UUID{itemID}, &struct{}{})
	if err != nil {
		t.Error(err)
	}

	if err := <-outgoing; err != nil {
		t.Error(err)
	}
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
