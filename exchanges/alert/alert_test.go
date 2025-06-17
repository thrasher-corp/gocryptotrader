package alert

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWait(t *testing.T) {
	wait := Notice{}
	var wg sync.WaitGroup

	// standard alert
	wg.Add(100)
	for range 100 {
		go func() {
			w := wait.Wait(nil)
			wg.Done()
			if <-w {
				log.Fatal("incorrect routine wait response for alert expecting false")
			}
			wg.Done()
		}()
	}

	wg.Wait()
	wg.Add(100)
	isLeaky(t, &wait, nil)
	wait.Alert()
	wg.Wait()
	isLeaky(t, &wait, nil)

	// use kick
	ch := make(chan struct{})
	wg.Add(100)
	for range 100 {
		go func() {
			w := wait.Wait(ch)
			wg.Done()
			if !<-w {
				log.Fatal("incorrect routine wait response for kick expecting true")
			}
			wg.Done()
		}()
	}
	wg.Wait()
	wg.Add(100)
	isLeaky(t, &wait, ch)
	close(ch)
	wg.Wait()
	ch = make(chan struct{})
	isLeaky(t, &wait, ch)

	// late receivers
	wg.Add(100)
	for x := range 100 {
		go func(x int) {
			bb := wait.Wait(ch)
			wg.Done()
			if x%2 == 0 {
				time.Sleep(time.Millisecond * 5)
			}
			b := <-bb
			if b {
				log.Fatal("incorrect routine wait response since we call alert below; expecting false")
			}
			wg.Done()
		}(x)
	}
	wg.Wait()
	wg.Add(100)
	isLeaky(t, &wait, ch)
	wait.Alert()
	wg.Wait()
	isLeaky(t, &wait, ch)
}

// isLeaky tests to see if the wait functionality is returning an abnormal
// channel that is operational when it shouldn't be.
func isLeaky(t *testing.T, a *Notice, ch chan struct{}) {
	t.Helper()
	check := a.Wait(ch)
	time.Sleep(time.Millisecond * 5) // When we call wait a routine for hold is
	// spawned, so for a test we need to add in a time for goschedular to allow
	// routine to actually wait on the forAlert and kick channels
	select {
	case <-check:
		t.Fatal("leaky waiter")
	default:
	}
}

// 120801772	         9.334 ns/op	       0 B/op	       0 allocs/op // PREV
// 146173060	         9.154 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkAlert(b *testing.B) {
	n := Notice{}
	for b.Loop() {
		n.Alert()
	}
}

// BenchmarkWait benchmark
//
// 150352	      9916 ns/op	     681 B/op	       4 allocs/op // PREV
// 87436	     14724 ns/op	     682 B/op	       4 allocs/op // CURRENT
func BenchmarkWait(b *testing.B) {
	n := Notice{}
	for b.Loop() {
		n.Wait(nil)
	}
}

// getSize checks the buffer size for testing purposes
func getSize() int {
	mu.RLock()
	defer mu.RUnlock()
	return preAllocBufferSize
}

func TestSetPreAllocationCommsBuffer(t *testing.T) {
	t.Parallel()
	err := SetPreAllocationCommsBuffer(-1)
	require.ErrorIs(t, err, errInvalidBufferSize)

	if getSize() != 5 {
		t.Fatal("unexpected amount")
	}

	err = SetPreAllocationCommsBuffer(7)
	require.NoError(t, err)

	if getSize() != 7 {
		t.Fatal("unexpected amount")
	}

	SetDefaultPreAllocationCommsBuffer()

	if getSize() != PreAllocCommsDefaultBuffer {
		t.Fatal("unexpected amount")
	}
}
