package websocket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func TestMatchReturnResponses(t *testing.T) {
	t.Parallel()

	conn := connection{Match: NewMatch()}
	_, err := conn.MatchReturnResponses(t.Context(), nil, 0)
	require.ErrorIs(t, err, errInvalidBufferSize)

	ch, err := conn.MatchReturnResponses(t.Context(), nil, 1)
	require.NoError(t, err)

	require.ErrorIs(t, (<-ch).Err, ErrSignatureTimeout)
	conn.ResponseMaxLimit = time.Second

	ch, err = conn.MatchReturnResponses(t.Context(), nil, 1)
	require.NoError(t, err)

	exp := []byte("test")
	require.True(t, conn.Match.IncomingWithData(nil, exp))
	resp := <-ch
	require.NoError(t, resp.Err)
	require.NotEmpty(t, resp.Responses, "must have response data")
	assert.Equal(t, exp, resp.Responses[0])
}

func TestWebsocketConnectionRequireMatchWithData(t *testing.T) {
	t.Parallel()
	ws := connection{Match: NewMatch()}
	err := ws.RequireMatchWithData(0, nil)
	require.ErrorIs(t, err, ErrSignatureNotMatched)

	ch, err := ws.Match.Set(0, 1)
	require.NoError(t, err)

	err = ws.RequireMatchWithData(0, []byte("test"))
	require.NoError(t, err)
	require.Len(t, ch, 1, "must have one item in channel")
	assert.Equal(t, []byte("test"), <-ch)
}

func TestIncomingWithData(t *testing.T) {
	t.Parallel()
	ws := connection{Match: NewMatch()}
	require.False(t, ws.IncomingWithData(0, nil))

	ch, err := ws.Match.Set(0, 1)
	require.NoError(t, err)

	require.True(t, ws.IncomingWithData(0, []byte("test")))
	require.Len(t, ch, 1, "must have one item in channel")
	assert.Equal(t, []byte("test"), <-ch)
}

func TestConnectionSubscriptions(t *testing.T) {
	t.Parallel()
	ws := &connection{}
	require.Nil(t, ws.Subscriptions())
	ws.subscriptions = subscription.NewStore()
	require.NotNil(t, ws.Subscriptions())
	testsubs.EqualLists(t, ws.subscriptions.List(), ws.Subscriptions().List())
}

func TestDialNilDialer(t *testing.T) {
	t.Parallel()
	err := (&connection{}).Dial(t.Context(), nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Dial should error on a nil dialer")
}

func TestDialInvalidProxyURL(t *testing.T) {
	t.Parallel()
	err := (&connection{ProxyURL: "://proxy"}).Dial(t.Context(), &gws.Dialer{}, nil, nil)
	var urlErr *url.Error
	assert.ErrorAs(t, err, &urlErr, "Dial should return the error parsing an invalid proxy URL")
}

func TestDialRedactsQuery(t *testing.T) {
	t.Parallel()
	values := url.Values{"token": {"secret-token"}}

	err := (&connection{URL: "ws://bad host/ws"}).Dial(t.Context(), &gws.Dialer{}, nil, values)
	var urlErr *url.Error
	require.ErrorAs(t, err, &urlErr, "Dial must return the error parsing a malformed URL")
	assert.Equal(t, "ws://bad host/ws", urlErr.URL, "Dial should leave the query out of the parse error's URL")
	assert.NotContains(t, err.Error(), "secret-token", "Dial should leave the query out of a parse error")

	errDial := errors.New("dial failure")
	failingDialer := &gws.Dialer{NetDialContext: func(context.Context, string, string) (net.Conn, error) { return nil, errDial }}
	err = (&connection{URL: "ws://localhost/ws"}).Dial(t.Context(), failingDialer, nil, values)
	require.ErrorIs(t, err, errDial, "Dial must return the dial failure")
	assert.ErrorContains(t, err, "ws://localhost/ws", "Dial should name the URL it dialled")
	assert.NotContains(t, err.Error(), "secret-token", "Dial should leave the query out of a failed connection")

	refusing, refusingDialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	refusingURL := "ws" + strings.TrimPrefix(refusing.URL, "http") + "/ws"
	err = (&connection{URL: refusingURL}).Dial(t.Context(), refusingDialer, nil, values)
	require.ErrorIs(t, err, gws.ErrBadHandshake, "Dial must return a refused upgrade")
	assert.ErrorContains(t, err, refusingURL, "Dial should name the URL it dialled")
	assert.NotContains(t, err.Error(), "secret-token", "Dial should leave the query out of a refused upgrade")
}

// captureLogs collects log output until the test ends. Callers must not run in parallel: the log hook and log config are
// process-wide, and cleanup returns both to the package defaults.
func captureLogs(t *testing.T) func() string {
	t.Helper()
	var mu sync.Mutex
	var logged strings.Builder
	require.NoError(t, log.SetGlobalLogConfig(log.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	log.SetCustomLogHook(func(_, _ string, a ...any) bool {
		mu.Lock()
		fmt.Fprintln(&logged, a...)
		mu.Unlock()
		return true
	})
	t.Cleanup(func() {
		log.SetCustomLogHook(nil)
		assert.NoError(t, log.SetGlobalLogConfig(&log.Config{}), "SetGlobalLogConfig should not error")
	})
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		return logged.String()
	}
}

func TestDialVerboseLogRedactsQuery(t *testing.T) {
	// Not parallel: captureLogs changes process-wide log settings
	logged := captureLogs(t)

	queries := make(chan string, 1)
	var upgrader gws.Upgrader
	s, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.RawQuery
		if conn, err := upgrader.Upgrade(w, r, nil); err == nil {
			_ = conn.Close()
		}
	}))
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
	c := &connection{ExchangeName: "test", URL: wsURL, Verbose: true}
	require.NoError(t, c.Dial(t.Context(), dialer, nil, url.Values{"token": {"secret-token"}}), "Dial must connect")
	t.Cleanup(func() { _ = c.Connection.Close() })
	assert.Equal(t, "token=secret-token", <-queries, "Dial should send the query to the server")
	assert.Contains(t, logged(), "Websocket connected to "+wsURL+"\n", "Dial should log the URL it connected to")
	assert.NotContains(t, logged(), "secret-token", "Dial should leave the query out of its log")
}

func TestReadMessageError(t *testing.T) {
	t.Parallel()
	var upgrader gws.Upgrader
	s, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.WriteMessage(gws.TextMessage, []byte("hello"))
		_ = conn.Close()
	}))
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"

	readErrs := make(chan error, 1)
	c := &connection{URL: wsURL + "?listenKey=secret-key", readMessageErrors: readErrs, Traffic: make(chan struct{}, 1)}
	require.NoError(t, c.Dial(t.Context(), dialer, nil, nil), "Dial must connect")
	t.Cleanup(func() { _ = c.Connection.Close() })
	require.Len(t, c.Traffic, 1, "Dial must signal traffic")
	<-c.Traffic
	assert.Equal(t, []byte("hello"), c.ReadMessage().Raw, "ReadMessage should return the server's message")
	assert.Len(t, c.Traffic, 1, "ReadMessage should signal traffic")
	assert.Nil(t, c.ReadMessage().Raw, "ReadMessage should return no data from a closed connection")
	require.Len(t, readErrs, 1, "ReadMessage must relay the read error")
	err := <-readErrs
	require.ErrorIs(t, err, errConnectionFault, "ReadMessage must report a connection fault")
	assert.ErrorContains(t, err, strconv.Quote(wsURL), "ReadMessage should name the connection's URL")
	assert.NotContains(t, err.Error(), "secret-key", "ReadMessage should leave the query out of its error")

	unreceived := &connection{URL: wsURL}
	require.NoError(t, unreceived.Dial(t.Context(), dialer, nil, nil), "Dial must connect")
	t.Cleanup(func() { _ = unreceived.Connection.Close() })
	assert.Equal(t, []byte("hello"), unreceived.ReadMessage().Raw, "ReadMessage should return the server's message")
	assert.Nil(t, unreceived.ReadMessage().Raw, "ReadMessage should return no data when nothing receives its error")
}

func TestDialHandshakeTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// the server accepts connections but never answers a request, so only the handshake timeout can end the dial
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		wc := &connection{URL: "ws" + mock.URL[len("http"):] + "/ws"}

		start := time.Now()
		err := wc.Dial(t.Context(), dialer, nil, nil)
		require.ErrorIs(t, err, os.ErrDeadlineExceeded, "Dial must time out when the upgrade is never answered")
		assert.Equal(t, 45*time.Second, time.Since(start), "Dial should give up after the default handshake timeout")
		assert.Zero(t, dialer.HandshakeTimeout, "Dial should not set a handshake timeout on the caller's dialer")

		wc.ProxyURL = "http://proxy.invalid"
		dialer.HandshakeTimeout = time.Second
		var dialled []string
		dial := dialer.NetDialContext
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialled = append(dialled, addr)
			return dial(ctx, network, addr)
		}
		start = time.Now()
		err = wc.Dial(t.Context(), dialer, nil, nil)
		require.ErrorIs(t, err, os.ErrDeadlineExceeded, "Dial must time out when the proxy never answers")
		assert.Equal(t, time.Second, time.Since(start), "Dial should use the handshake timeout set on the dialer")
		assert.Nil(t, dialer.Proxy, "Dial should not set a proxy on the caller's dialer")
		assert.Equal(t, []string{"proxy.invalid:80"}, dialled, "Dial should connect through the configured proxy")
	})
}
