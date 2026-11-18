package websocket

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
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

// sendTimedRequest is a request that carries the time it is sent
type sendTimedRequest struct {
	RequestID int64     `json:"reqid"`
	SentAt    time.Time `json:"sentAt"`
}

// StampSendTime returns a copy of the request carrying the time it is sent
func (r *sendTimedRequest) StampSendTime(t time.Time) any {
	stamped := *r
	stamped.SentAt = t
	return &stamped
}

// unmarshalableSendTimedRequest is a request that carries the time it is sent and cannot be marshalled
type unmarshalableSendTimedRequest struct {
	RequestID int64    `json:"reqid"`
	Updates   chan int `json:"updates"`
}

// StampSendTime returns the request, which cannot be marshalled whatever its send time
func (r *unmarshalableSendTimedRequest) StampSendTime(time.Time) any {
	return r
}

func TestSendTimeStamper(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
		const wait = time.Second
		wc := &connection{
			URL:              "ws" + mock.URL[len("http"):] + "/ws",
			ResponseMaxLimit: time.Minute,
			Match:            NewMatch(),
			RateLimit:        request.NewWeightedRateLimitByDuration(wait),
		}
		require.NoError(t, wc.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
		// the bubble waits for every goroutine, including the reader below and the mock server's handler, which only
		// return once the connection closes
		t.Cleanup(func() { assert.NoError(t, wc.Shutdown(), "Shutdown should not error") })
		echoes := make(chan sendTimedRequest, 3)
		go func() {
			for {
				resp := wc.ReadMessage()
				if resp.Raw == nil {
					return
				}
				var echoed sendTimedRequest
				if assert.NoError(t, json.Unmarshal(resp.Raw, &echoed), "echoed request should decode") {
					wc.Match.IncomingWithData(echoed.RequestID, resp.Raw)
					echoes <- echoed
				}
			}
		}()

		// The first request takes the rate limit's only token, so each later one waits a second for the next
		first := &sendTimedRequest{RequestID: 1}
		_, err := wc.SendMessageReturnResponse(t.Context(), request.Unset, first.RequestID, first)
		require.NoError(t, err, "SendMessageReturnResponse must not error")
		second := &sendTimedRequest{RequestID: 2}
		_, err = wc.SendMessageReturnResponse(t.Context(), request.Unset, second.RequestID, second)
		require.NoError(t, err, "SendMessageReturnResponse must not error")
		third := &sendTimedRequest{RequestID: 3}
		require.NoError(t, wc.SendJSONMessage(t.Context(), request.Unset, third), "SendJSONMessage must not error")

		// The rate limit has no token to spare, so only a payload refused before its rate limit wait fails to marshal
		_, err = wc.SendMessageReturnResponse(request.WithDelayNotAllowed(t.Context()), request.Unset, int64(4), &unmarshalableSendTimedRequest{RequestID: 4})
		assert.ErrorContains(t, err, "error marshaling json", "SendMessageReturnResponse should refuse a payload it cannot marshal before its rate limit wait")

		// Every echo is read before the bubble ends, since the mock server sleeps before each one and the bubble's time
		// stops once this function returns
		sent := make(map[int64]time.Time, 3)
		for range 3 {
			echoed := <-echoes
			sent[echoed.RequestID] = echoed.SentAt
		}
		assert.WithinDuration(t, sent[1].Add(wait), sent[2], time.Millisecond, "SendMessageReturnResponse should send a payload stamped once its rate limit wait is over")
		assert.WithinDuration(t, sent[2].Add(wait), sent[3], time.Millisecond, "SendJSONMessage should send a payload stamped once its rate limit wait is over")
		assert.Equal(t, &sendTimedRequest{RequestID: 2}, second, "SendMessageReturnResponse should leave the payload it was given unchanged")
		assert.Equal(t, &sendTimedRequest{RequestID: 3}, third, "SendJSONMessage should leave the payload it was given unchanged")
	})
}

func TestDialNilDialer(t *testing.T) {
	t.Parallel()
	err := (&connection{}).Dial(t.Context(), nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Dial should error on a nil dialer")
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
