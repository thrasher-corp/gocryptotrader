package stream

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errExpectedTestErrorWhenProcessing = errors.New("test")

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	_, err := NewProcessor(0, nil)
	if !errors.Is(err, errMaxChanBufferSizeInvalid) {
		t.Fatalf("received: %v, expected: %v", err, errMaxChanBufferSizeInvalid)
	}

	_, err = NewProcessor(defaultChannelBufferSize, nil)
	if !errors.Is(err, errDataHandlerMustNotBeNil) {
		t.Fatalf("received: %v, expected: %v", err, errDataHandlerMustNotBeNil)
	}

	got, err := NewProcessor(defaultChannelBufferSize, make(chan interface{}))
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	if got == nil {
		t.Fatal("expected processor")
	}

	if got.chanBufferSize != defaultChannelBufferSize {
		t.Fatalf("received: %v, expected: %v", got.chanBufferSize, defaultChannelBufferSize)
	}

	if got.dataHandler == nil {
		t.Fatal("expected data handler")
	}

	if got.routes == nil {
		t.Fatal("expected routes")
	}
}

func TestProcessorQueueFunction(t *testing.T) {
	t.Parallel()

	happyDataHandler := make(chan interface{}) // unbuffered to block error
	proc, err := NewProcessor(defaultChannelBufferSize, happyDataHandler)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	err = proc.QueueFunction(Key{}, nil)
	if !errors.Is(err, errKeyEmpty) {
		t.Fatalf("received: %v, expected: %v", err, errKeyEmpty)
	}

	err = proc.QueueFunction(Key{Asset: asset.Spot}, nil)
	if !errors.Is(err, errUpdateTypeUnset) {
		t.Fatalf("received: %v, expected: %v", err, errUpdateTypeUnset)
	}

	err = proc.QueueFunction(Key{Type: 3, Asset: asset.Spot}, nil)
	if !errors.Is(err, errdUpdateTypeNotYetSupported) {
		t.Fatalf("received: %v, expected: %v", err, errdUpdateTypeNotYetSupported)
	}

	err = proc.QueueFunction(Key{Type: Book, Asset: asset.Spot}, nil)
	if !errors.Is(err, errNoFunctionalityToProcess) {
		t.Fatalf("received: %v, expected: %v", err, errNoFunctionalityToProcess)
	}

	// This will error and the routine will pause on send
	err = proc.QueueFunction(Key{Type: Book, Asset: asset.Spot}, func() error {
		return errExpectedTestErrorWhenProcessing
	})
	if err != nil {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	var wg sync.WaitGroup
	// This will back fill up the process channel
	for x := 0; x < defaultChannelBufferSize; x++ {
		wg.Add(1)
		err = proc.QueueFunction(Key{Type: Book, Asset: asset.Spot}, func() error {
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
		err = proc.QueueFunction(Key{Type: Book, Asset: asset.Spot}, func() error {
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
	if respErr, ok := resp.(error); ok && respErr != errExpectedTestErrorWhenProcessing {
		t.Fatalf("received: %v, expected: %v", respErr, errExpectedTestErrorWhenProcessing)
	}

	wg.Wait()

	// This will read the second error and then unblock processing
	err = <-defaultCapture
	if !errors.Is(err, errChannelFull) {
		t.Fatalf("received: %v, expected: %v", err, errChannelFull)
	}
}
