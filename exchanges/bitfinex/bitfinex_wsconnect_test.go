package bitfinex

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type wsConnectFixtureConnection struct {
	websocket.Connection
	dialErr error

	dialCalls     atomic.Int32
	readCalls     atomic.Int32
	sendJSONCalls atomic.Int32
}

func (f *wsConnectFixtureConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *wsConnectFixtureConnection) ReadMessage() websocket.Response {
	f.readCalls.Add(1)
	return websocket.Response{}
}

func (f *wsConnectFixtureConnection) SendJSONMessage(context.Context, request.EndpointLimit, any) error {
	f.sendJSONCalls.Add(1)
	return nil
}

func waitForWaitGroup(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for websocket waitgroup within %s", timeout)
	}
}

func TestWsConnectAuthDialFailureSkipsAuthReaderAndAuthSend(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials("key", "secret", "", "", "", "")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	publicConn := &wsConnectFixtureConnection{}
	authConn := &wsConnectFixtureConnection{dialErr: errors.New("auth dial failed")}
	ex.Websocket.Conn = publicConn
	ex.Websocket.AuthConn = authConn

	err := ex.WsConnect()
	require.NoError(t, err, "WsConnect must not error when auth dial fails")

	waitForWaitGroup(t, &ex.Websocket.Wg, 2*time.Second)

	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "auth endpoints should be disabled on auth dial failure")
	assert.Equal(t, int32(1), publicConn.dialCalls.Load(), "public conn should dial once")
	assert.Equal(t, int32(1), publicConn.readCalls.Load(), "public reader should run once")
	assert.Equal(t, int32(1), publicConn.sendJSONCalls.Load(), "ConfigureWS should send once on public connection")
	assert.Equal(t, int32(1), authConn.dialCalls.Load(), "auth conn should attempt dial once")
	assert.Equal(t, int32(0), authConn.readCalls.Load(), "auth reader should not start after failed auth dial")
	assert.Equal(t, int32(0), authConn.sendJSONCalls.Load(), "auth send should be skipped after failed auth dial")
}
