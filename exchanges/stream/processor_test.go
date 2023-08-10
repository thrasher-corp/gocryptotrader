package stream

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	_, err := NewProcessor(0, nil)
	if !errors.Is(err, errMaxChanBufferSizeInvalid) {
		t.Fatalf("received: %v, expected: %v", err, errMaxChanBufferSizeInvalid)
	}

	_, err = NewProcessor(DefaultChannelBufferSize, nil)
	if !errors.Is(err, errDataHandlerMustNotBeNil) {
		t.Fatalf("received: %v, expected: %v", err, errDataHandlerMustNotBeNil)
	}

	got, err := NewProcessor(DefaultChannelBufferSize, make(chan interface{}))
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	if got == nil {
		t.Fatal("expected processor")
	}

	if got.chanBufferSize != DefaultChannelBufferSize {
		t.Fatalf("received: %v, expected: %v", got.chanBufferSize, DefaultChannelBufferSize)
	}

	if got.dataHandler == nil {
		t.Fatal("expected data handler")
	}

	if got.routes == nil {
		t.Fatal("expected routes")
	}
}

var errExpectedTestErrorWhenProcessing = errors.New("test")

func TestProcessorProcess(t *testing.T) {
	t.Parallel()

	happyDataHandler := make(chan interface{}) // unbuffered to block error
	proc, err := NewProcessor(DefaultChannelBufferSize, happyDataHandler)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	err = proc.Process(Key{}, nil)
	if !errors.Is(err, errKeyEmpty) {
		t.Fatalf("received: %v, expected: %v", err, errKeyEmpty)
	}

	err = proc.Process(Key{Asset: asset.Spot}, nil)
	if !errors.Is(err, errNoFunctionalityToProcess) {
		t.Fatalf("received: %v, expected: %v", err, errNoFunctionalityToProcess)
	}

	// This will error and the routine will pause on send
	err = proc.Process(Key{Asset: asset.Spot}, func() error {
		return errExpectedTestErrorWhenProcessing
	})
	if err != nil {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	var wg sync.WaitGroup
	// This will back fill up the process channel
	for x := 0; x < DefaultChannelBufferSize; x++ {
		wg.Add(1)
		err = proc.Process(Key{Asset: asset.Spot}, func() error {
			wg.Done()
			return nil
		})
		if err != nil {
			t.Fatalf("received: %v, expected: %v", err, nil)
		}
	}

	wg.Add(1)

	defaultCapture := make(chan error)
	// This will exceed the channel buffer size and block so needs to be in
	// a go routine
	go func() {
		err = proc.Process(Key{Asset: asset.Spot}, func() error {
			wg.Done()
			return nil
		})
		if err != nil {
			defaultCapture <- err
		}
	}()

	// Allows for above to wiggout
	time.Sleep(time.Millisecond * 100)

	// This will read the first error and then unblock processing
	resp := <-happyDataHandler
	if resp.(error) != errExpectedTestErrorWhenProcessing {
		t.Fatalf("received: %v, expected: %v", resp.(error), errExpectedTestErrorWhenProcessing)
	}

	wg.Wait()

	// This will read the second error and then unblock processing
	err = <-defaultCapture
	if !errors.Is(err, errChannelFull) {
		t.Fatalf("received: %v, expected: %v", err, errChannelFull)
	}
}
