package bitfinex

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

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

func TestWSConnectDialFailureSkipsConfigure(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &wsConnectFixtureConnection{dialErr: errors.New("dial failed")}

	err := ex.wsConnect(t.Context(), conn)
	require.Error(t, err, "wsConnect must return an error when dial fails")
	assert.Equal(t, int32(1), conn.dialCalls.Load(), "connection should dial once")
	assert.Equal(t, int32(0), conn.sendJSONCalls.Load(), "ConfigureWS should not send when dial fails")
}
