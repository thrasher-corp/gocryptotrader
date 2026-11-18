package bithumb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestSynchroniseWebsocketOrderbook(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	// With no endpoints set, each job fails before it sends a request
	e.API.Endpoints = e.NewEndpoints()
	e.obm.jobs = make(chan job, maxWSOrderbookJobs)
	e.Websocket = sharedtestvalues.NewTestWebsocket()
	requireStopped := func(msg string) {
		t.Helper()
		stopped := make(chan struct{})
		go func() {
			e.Websocket.Wg.Wait()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(time.Second * 5):
			require.FailNow(t, msg)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	e.SynchroniseWebsocketOrderbook(ctx)
	for range 2 {
		e.obm.jobs <- job{}
		require.Eventually(t, func() bool { return len(e.obm.jobs) == 0 }, time.Second*5, time.Millisecond, "SynchroniseWebsocketOrderbook must keep taking jobs")
	}
	cancel()
	requireStopped("SynchroniseWebsocketOrderbook must stop when its context ends")

	for range cap(e.obm.jobs) {
		e.obm.jobs <- job{}
	}
	e.SynchroniseWebsocketOrderbook(ctx)
	requireStopped("SynchroniseWebsocketOrderbook must stop when its context has ended")
	require.Empty(t, e.obm.jobs, "SynchroniseWebsocketOrderbook must drain pending jobs when its context has ended")

	for range cap(e.obm.jobs) {
		e.obm.jobs <- job{}
	}
	close(e.Websocket.ShutdownC)
	e.SynchroniseWebsocketOrderbook(t.Context())
	requireStopped("SynchroniseWebsocketOrderbook must stop when the websocket shuts down")
	assert.Empty(t, e.obm.jobs, "SynchroniseWebsocketOrderbook should drain pending jobs when the websocket shuts down")
}
