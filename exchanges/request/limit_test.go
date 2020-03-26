package request

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestStop(t *testing.T) {
	l := NewBasicRateLimit(time.Second, 1)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			err := l.Initiate(EndpointLimit(i))
			if err != nil {
				os.Exit(1)
			}
			wg.Done()
		}(i)
	}

	// Halt service 1 second.
	l.Lock()
	time.Sleep(time.Second * 2)
	l.Unlock()

	wg.Wait()

	l.Shutdown()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			err := l.Initiate(EndpointLimit(i + 10))
			if err == nil {
				os.Exit(1)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}
