package exchange_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	shared "github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

type mockEx struct {
	shared.CustomEx
	flow chan int
}

func (m *mockEx) UpdateTradablePairs(_ context.Context, _ bool) error {
	m.flow <- 42
	return nil
}

func TestStart(t *testing.T) {
	m := &mockEx{
		shared.CustomEx{},
		make(chan int, 1),
	}
	m.Features.Enabled.AutoPairUpdates = true
	wg := sync.WaitGroup{}
	err := exchange.Start(context.TODO(), m, &wg)
	assert.NoError(t, err, "Start should not error")
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()
	assert.Eventually(t, func() bool { return len(done) == 1 }, time.Second, 100*time.Millisecond, "Start should resolve the waitgroup eventually")
	assert.Equal(t, 42, <-m.flow, "UpdateTradablePairs should be called on the exchange")
}
