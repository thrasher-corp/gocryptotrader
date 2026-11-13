package adanos

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const populatedToken = `{
	"symbol":"BTC","name":"Synthetic token","found":true,
	"buzz_score":12.5,"mentions":8,"sentiment_score":-0.25,
	"positive_count":2,"negative_count":4,"neutral_count":2,
	"total_upvotes":-9,"unique_posts":3,"subreddit_count":2,
	"trend":"falling","bullish_pct":25,"bearish_pct":50,"period_days":3,
	"top_subreddits":[
		{"subreddit":"synthetic_one","mentions":5,"count":5,"sentiment_score":-0.5,"buzz_score":11.5},
		{"subreddit":"synthetic_two","mentions":3,"count":3,"sentiment_score":null,"buzz_score":0}
	],
	"daily_trend":[
		{"date":"2026-10-30","mentions":4,"sentiment_score":-0.5,"buzz_score":10.5,"bullish_pct":25,"bearish_pct":50},
		{"date":"2026-10-31","mentions":3,"sentiment_score":0,"buzz_score":null,"bullish_pct":0,"bearish_pct":null},
		{"date":"2026-11-01","mentions":1,"sentiment_score":null,"buzz_score":0,"bullish_pct":null,"bearish_pct":0}
	],
	"top_mentions":[
		{"text_snippet":"Synthetic discussion only.","sentiment_score":-0.75,"sentiment_label":"negative","upvotes":-12,"subreddit":"synthetic_one","created_utc":"2026-10-31T23:59:59Z"},
		{"text_snippet":"Another synthetic discussion.","sentiment_score":null,"sentiment_label":null,"upvotes":0,"subreddit":null,"created_utc":"2026-11-01T00:00:00Z"}
	]
}`

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *atomic.Int64) {
	t.Helper()
	var attempts atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		handler(w, r)
	}))
	server.Start()
	c, err := New("synthetic-test-key")
	require.NoError(t, err, "New must construct without HTTP")
	c.baseURL = server.URL
	transport, ok := c.httpClient.Transport.(*http.Transport)
	require.True(t, ok, "native client must use an HTTP transport")
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			return nil, errors.New("non-loopback test destination refused")
		}
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, address)
	}
	t.Cleanup(func() {
		assert.NoError(t, c.Shutdown(), "test client should release its connections and registration")
	})
	return c, &attempts
}

func TestNew(t *testing.T) {
	for _, key := range []string{"", " \t", "synthetic\r\nkey"} {
		c, err := New(key)
		assert.ErrorIs(t, err, ErrInvalidAPIKey, "invalid keys should fail locally")
		assert.Nil(t, c, "invalid keys should not construct a client")
	}
	c, err := New("synthetic-test-key")
	require.NoError(t, err, "New must not require network access")
	defer func() { assert.NoError(t, c.Shutdown(), "client should release its registration") }()
	assert.Equal(t, "https://api.adanos.com", c.baseURL, "client should use the canonical origin")
	assert.Positive(t, c.httpClient.Timeout, "requests should have a bounded timeout")
	assert.LessOrEqual(t, c.httpClient.Timeout, 30*time.Second, "default timeout should be bounded")
}

func TestShutdown(t *testing.T) {
	c, err := New("synthetic-test-key")
	require.NoError(t, err, "client must construct without HTTP")
	transport, ok := c.httpClient.Transport.(*http.Transport)
	require.True(t, ok, "native client must use an HTTP transport")
	// Exercise idle cleanup even though production requests disable keep-alives.
	transport.DisableKeepAlives = false
	transport.Proxy = nil
	closed := make(chan struct{}, 1)
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"BTC","found":false}`))
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			closed <- struct{}{}
		}
	}
	server.Start()
	c.baseURL = server.URL
	_, err = c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
	require.NoError(t, err, "request must finish before shutdown")
	duplicate, err := request.New("Adanos registration check", c.httpClient)
	assert.Error(t, err, "active client's HTTP client should already be registered")
	assert.Nil(t, duplicate, "active client's HTTP client should not register twice")
	require.NoError(t, c.Shutdown(), "retired client must release its registration")
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Error("shutdown should close the idle connection")
	}
	replacement, err := request.New("Adanos registration check", c.httpClient)
	require.NoError(t, err, "released HTTP client must be available for registration")
	assert.NoError(t, replacement.Shutdown(), "replacement requester should release its registration")
	t.Logf("lifecycle verification backend=%s registration_released=true idle_connection_closed=true external=0", json.Implementation)
}

func TestGetTokenPopulated(t *testing.T) {
	c, attempts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request should use GET")
		assert.Equal(t, "/reddit/crypto/v1/token/$bTc", r.URL.Path, "request should use only the token endpoint")
		assert.Equal(t, "synthetic-test-key", r.Header.Get("X-API-Key"), "key should travel in its header")
		assert.Equal(t, "application/json", r.Header.Get("Accept"), "request should accept JSON")
		assert.Empty(t, r.URL.RawQuery, "zero bounds should omit all query parameters")
		_, _ = w.Write([]byte(populatedToken))
	})
	got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "$bTc"})
	require.NoError(t, err, "populated report must decode")
	want := &TokenResponse{
		Symbol: "BTC", Name: new("Synthetic token"), Found: true,
		BuzzScore: new(12.5), Mentions: new(uint64(8)), SentimentScore: new(-0.25),
		PositiveCount: new(uint64(2)), NegativeCount: new(uint64(4)), NeutralCount: new(uint64(2)),
		TotalUpvotes: new(int64(-9)), UniquePosts: new(uint64(3)), SubredditCount: new(uint64(2)),
		Trend: new("falling"), BullishPercentage: new(uint64(25)), BearishPercentage: new(uint64(50)), PeriodDays: new(uint64(3)),
		TopSubreddits: []SubredditCount{
			{Subreddit: "synthetic_one", Mentions: 5, Count: 5, SentimentScore: new(-0.5), BuzzScore: new(11.5)},
			{Subreddit: "synthetic_two", Mentions: 3, Count: 3, BuzzScore: new(float64(0))},
		},
		DailyTrend: []DailyTrendItem{
			{Date: "2026-10-30", Mentions: 4, SentimentScore: new(-0.5), BuzzScore: new(10.5), BullishPercentage: new(uint64(25)), BearishPercentage: new(uint64(50))},
			{Date: "2026-10-31", Mentions: 3, SentimentScore: new(float64(0)), BullishPercentage: new(uint64(0))},
			{Date: "2026-11-01", Mentions: 1, BuzzScore: new(float64(0)), BearishPercentage: new(uint64(0))},
		},
		TopMentions: []TopMention{
			{TextSnippet: "Synthetic discussion only.", SentimentScore: new(-0.75), SentimentLabel: new("negative"), Upvotes: -12, Subreddit: new("synthetic_one"), CreatedUTC: time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)},
			{TextSnippet: "Another synthetic discussion.", CreatedUTC: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	assert.Equal(t, want, got, "every selected schema field should survive the public method")
	assert.EqualValues(t, 1, attempts.Load(), "one call should send exactly one request")
	t.Logf("native verification fixture=populatedToken backend=%s requests=%d external=0", json.Implementation, attempts.Load())
}

func TestGetTokenNoDataAndZero(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want *TokenResponse
	}{
		{
			name: "no data",
			body: `{"symbol":"BTC","found":false,"name":null,"buzz_score":null,"mentions":null,"sentiment_score":null,"positive_count":null,"negative_count":null,"neutral_count":null,"total_upvotes":null,"unique_posts":null,"subreddit_count":null,"trend":null,"bullish_pct":null,"bearish_pct":null,"period_days":null,"top_subreddits":null,"daily_trend":null,"top_mentions":null}`,
			want: &TokenResponse{Symbol: "BTC"},
		},
		{
			name: "zero data",
			body: `{"symbol":"BTC","found":true,"name":"","buzz_score":0,"mentions":0,"sentiment_score":0,"positive_count":0,"negative_count":0,"neutral_count":0,"total_upvotes":0,"unique_posts":0,"subreddit_count":0,"trend":"stable","bullish_pct":0,"bearish_pct":0,"period_days":0,"top_subreddits":[],"daily_trend":[],"top_mentions":[]}`,
			want: &TokenResponse{Symbol: "BTC", Found: true, Name: new(""), BuzzScore: new(float64(0)), Mentions: new(uint64(0)), SentimentScore: new(float64(0)), PositiveCount: new(uint64(0)), NegativeCount: new(uint64(0)), NeutralCount: new(uint64(0)), TotalUpvotes: new(int64(0)), UniquePosts: new(uint64(0)), SubredditCount: new(uint64(0)), Trend: new("stable"), BullishPercentage: new(uint64(0)), BearishPercentage: new(uint64(0)), PeriodDays: new(uint64(0)), TopSubreddits: []SubredditCount{}, DailyTrend: []DailyTrendItem{}, TopMentions: []TopMention{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, attempts := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) })
			got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
			require.NoError(t, err, "supported symbol must return a successful report")
			assert.Equal(t, tc.want, got, "nulls, zeros, and empty lists should retain their meaning")
			assert.EqualValues(t, 1, attempts.Load(), "report should use one request")
		})
	}
}

func TestGetTokenUTCDateBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		from time.Time
		to   time.Time
		want url.Values
	}{
		{name: "UTC rollover", from: time.Date(2026, 11, 1, 0, 30, 0, 0, time.FixedZone("east", 2*3600)), to: time.Date(2026, 11, 1, 23, 30, 0, 0, time.FixedZone("west", -2*3600)), want: url.Values{"from": {"2026-10-31"}, "to": {"2026-11-02"}}},
		{name: "equal UTC dates with reversed instants", from: time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC), to: time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC), want: url.Values{"from": {"2026-11-01"}, "to": {"2026-11-01"}}},
		{name: "from only", from: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), want: url.Values{"from": {"2026-11-01"}}},
		{name: "to only", to: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), want: url.Values{"to": {"2026-11-01"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, attempts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.want, r.URL.Query(), "bounds should be inclusive UTC dates without days")
				_, _ = w.Write([]byte(`{"symbol":"BTC","found":false}`))
			})
			_, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC", From: tc.from, To: tc.to})
			require.NoError(t, err, "valid UTC dates must be accepted")
			assert.EqualValues(t, 1, attempts.Load(), "valid dates should make one request")
		})
	}
}

func TestGetTokenValidation(t *testing.T) {
	c, attempts := newTestClient(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("invalid requests should not reach HTTP") })
	_, err := c.GetToken(t.Context(), nil)
	assert.ErrorIs(t, err, ErrInvalidRequest, "nil parameters should fail locally")
	var nilContext context.Context
	_, err = c.GetToken(nilContext, &TokenRequest{Symbol: "BTC"})
	assert.ErrorIs(t, err, ErrInvalidRequest, "nil context should fail locally")
	for _, symbol := range []string{"", "$", " BTC", "BTC\n", "BTC/USD", "../BTC", "BTC?key=x", "B-TC", "\u00e9", strings.Repeat("B", 21)} {
		_, err = c.GetToken(t.Context(), &TokenRequest{Symbol: symbol})
		assert.ErrorIs(t, err, ErrInvalidSymbol, "invalid symbols should fail locally")
	}
	_, err = c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC", From: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)})
	assert.ErrorIs(t, err, ErrInvalidPeriod, "reversed UTC dates should fail locally")
	_, err = c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC", From: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)})
	assert.ErrorIs(t, err, ErrInvalidPeriod, "non-date years should fail locally")
	assert.Zero(t, attempts.Load(), "all invalid inputs should make zero requests")
}

func TestGetTokenHTTPFailures(t *testing.T) {
	for _, status := range []int{401, 403, 404, 422, 429, 503, 504} {
		for _, detail := range []string{`"synthetic-test-key"`, `{"message":"synthetic-test-key"}`, `[{"msg":"synthetic-test-key"}]`} {
			t.Run(fmt.Sprintf("%d/%s", status, detail), func(t *testing.T) {
				c, attempts := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(status)
					_, _ = fmt.Fprintf(w, `{"detail":%s}`, detail)
				})
				got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
				require.ErrorIs(t, err, request.ErrBadStatus, "HTTP failure must preserve the native error")
				assert.NotContains(t, err.Error(), "synthetic-test-key", "returned errors should redact echoed keys")
				assert.Nil(t, got, "HTTP failures should not return successful data")
				assert.EqualValues(t, 1, attempts.Load(), "HTTP failures including 429 should never retry")
			})
		}
	}
}

func TestGetTokenPaddedKeyRedaction(t *testing.T) {
	for _, padding := range []string{"", " ", "\t", " \t"} {
		t.Run(fmt.Sprintf("padding=%q", padding), func(t *testing.T) {
			var received atomic.Int64
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received.Add(1)
				assert.Equal(t, "synthetic-padding-key", r.Header.Get("X-API-Key"), "server should receive the usable key")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				body, err := json.Marshal(map[string]string{"detail": r.Header.Get("X-API-Key")})
				assert.NoError(t, err, "echoed header should encode as JSON")
				_, _ = w.Write(body)
			}))
			server.Start()
			c, err := New(padding + "synthetic-padding-key" + padding)
			require.NoError(t, err, "padded keys must remain accepted")
			t.Cleanup(func() { assert.NoError(t, c.Shutdown(), "client should release registration") })
			c.baseURL = server.URL
			transport, ok := c.httpClient.Transport.(*http.Transport)
			require.True(t, ok, "native client must use an HTTP transport")
			transport.Proxy = nil
			got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
			require.ErrorIs(t, err, request.ErrBadStatus, "native error identity must survive redaction")
			assert.Nil(t, got, "HTTP failure should not return a report")
			assert.NotContains(t, err.Error(), "synthetic-padding-key", "wire key should be redacted")
			assert.Contains(t, err.Error(), "[REDACTED]", "echoed wire key should be replaced")
			assert.EqualValues(t, 1, received.Load(), "one call should send one request")
			t.Logf("backend=%s received_requests=%d wire_key_redacted=%t external=0", json.Implementation, received.Load(), !strings.Contains(err.Error(), "synthetic-padding-key"))
		})
	}
}

func TestGetTokenTLSProtocolOneShot(t *testing.T) {
	for _, control := range []bool{true, false} {
		t.Run(fmt.Sprintf("native_protocol_control=%t", control), func(t *testing.T) {
			var h1, h2 atomic.Int64
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h1.Add(1)
				assert.Equal(t, "HTTP/1.1", r.Proto, "restricted client should use HTTP1")
				assert.NotEqual(t, "h2", r.TLS.NegotiatedProtocol, "TLS should not negotiate HTTP2")
				_, _ = w.Write([]byte(`{"symbol":"BTC","found":false}`))
			}))
			server.EnableHTTP2 = true
			server.TLS = &tls.Config{NextProtos: []string{"h2", "http/1.1"}, MinVersion: tls.VersionTLS12}
			server.Config.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
				"h2": func(_ *http.Server, conn *tls.Conn, _ http.Handler) {
					defer conn.Close()
					assert.Equal(t, "h2", conn.ConnectionState().NegotiatedProtocol, "control should negotiate HTTP2")
					assert.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)), "TLS fixture should be bounded")
					preface := make([]byte, len(http2.ClientPreface))
					if _, err := io.ReadFull(conn, preface); !assert.NoError(t, err, "client should send HTTP2 preface") {
						return
					}
					assert.Equal(t, http2.ClientPreface, string(preface), "control should use real HTTP2 frames")
					framer := http2.NewFramer(conn, conn)
					if !assert.NoError(t, framer.WriteSettings(), "server should offer settings") {
						return
					}
					for {
						frame, err := framer.ReadFrame()
						if !assert.NoError(t, err, "server should read the request frames") {
							return
						}
						switch f := frame.(type) {
						case *http2.SettingsFrame:
							if !f.IsAck() && !assert.NoError(t, framer.WriteSettingsAck(), "server should acknowledge settings") {
								return
							}
						case *http2.HeadersFrame:
							if h2.Add(1) == 1 {
								assert.NoError(t, framer.WriteRSTStream(f.StreamID, http2.ErrCodeRefusedStream), "first stream should be refused")
								return
							}
							var headers bytes.Buffer
							assert.NoError(t, hpack.NewEncoder(&headers).WriteField(hpack.HeaderField{Name: ":status", Value: "200"}), "control response should encode")
							assert.NoError(t, framer.WriteHeaders(http2.HeadersFrameParam{StreamID: f.StreamID, BlockFragment: headers.Bytes(), EndHeaders: true}), "control should return headers")
							assert.NoError(t, framer.WriteData(f.StreamID, true, []byte(`{"symbol":"BTC","found":false}`)), "control replay should complete")
							return
						}
					}
				},
			}
			server.StartTLS()
			c, err := New("synthetic-tls-key")
			require.NoError(t, err, "client must construct")
			t.Cleanup(func() { assert.NoError(t, c.Shutdown(), "client should release registration") })
			c.baseURL = server.URL
			transport, ok := c.httpClient.Transport.(*http.Transport)
			require.True(t, ok, "native client must use an HTTP transport")
			transport.Proxy = nil
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
			// Custom test trust disables automatic H2 unless explicitly restored.
			transport.ForceAttemptHTTP2 = true
			if control {
				transport.Protocols = nil
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			got, err := c.GetToken(ctx, &TokenRequest{Symbol: "BTC"})
			require.NoError(t, err, "TLS report must complete")
			assert.Equal(t, &TokenResponse{Symbol: "BTC"}, got, "typed report should survive protocol selection")
			if control {
				assert.Zero(t, h1.Load(), "native protocol control should select H2")
				assert.EqualValues(t, 2, h2.Load(), "control should demonstrate refused-stream replay")
			} else {
				assert.EqualValues(t, 1, h1.Load(), "one public call should send exactly one HTTP1 request")
				assert.Zero(t, h2.Load(), "restricted client should not negotiate HTTP2 or replay")
			}
			t.Logf("backend=%s control=%t public_calls=1 h1_requests=%d h2_headers=%d external=0", json.Implementation, control, h1.Load(), h2.Load())
		})
	}
}

func TestGetTokenInvalidResponse(t *testing.T) {
	for _, body := range []string{`{`, `null`, `{}`, `{"symbol":"BTC","found":true,"mentions":"bad"}`} {
		t.Run(body, func(t *testing.T) {
			c, attempts := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
			got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
			assert.Error(t, err, "invalid responses should fail")
			if body == "null" || body == "{}" {
				assert.ErrorIs(t, err, ErrInvalidResponse, "missing report should retain its sentinel")
			}
			assert.Nil(t, got, "invalid responses should not invent data")
			assert.EqualValues(t, 1, attempts.Load(), "decode failures should not retry")
		})
	}
}

func TestGetTokenRejectsRedirects(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(strconv.FormatBool(external), func(t *testing.T) {
			var destinationCalls atomic.Int64
			destination := httptest.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { destinationCalls.Add(1) }))
			destination.Start()
			c, attempts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirected" {
					destinationCalls.Add(1)
					return
				}
				target := "/redirected"
				if external {
					target = destination.URL + "/redirected"
				}
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			})
			got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
			assert.ErrorIs(t, err, request.ErrBadStatus, "redirects should remain native HTTP failures")
			assert.Nil(t, got, "redirects should not return data")
			assert.EqualValues(t, 1, attempts.Load(), "redirects should not resend the key")
			assert.Zero(t, destinationCalls.Load(), "redirect destinations should receive no request")
		})
	}
}

func TestGetTokenCancellation(t *testing.T) {
	for _, mode := range []string{"already cancelled", "in flight", "deadline", "client timeout"} {
		t.Run(mode, func(t *testing.T) {
			started := make(chan struct{})
			c, attempts := newTestClient(t, func(_ http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-r.Context().Done():
				case <-time.After(2 * time.Second):
				}
			})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			want := context.Canceled
			switch mode {
			case "already cancelled":
				cancel()
			case "in flight":
				go func() {
					select {
					case <-started:
						cancel()
					case <-ctx.Done():
					}
				}()
			case "deadline":
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 100*time.Millisecond)
				defer stop()
				want = context.DeadlineExceeded
			case "client timeout":
				c.httpClient.Timeout = 100 * time.Millisecond
				want = context.DeadlineExceeded
			}
			got, err := c.GetToken(ctx, &TokenRequest{Symbol: "BTC"})
			assert.ErrorIs(t, err, want, "context errors should retain their identity")
			assert.Nil(t, got, "cancelled requests should not return data")
			if mode == "already cancelled" {
				assert.Zero(t, attempts.Load(), "pre-cancelled calls should send no request")
			} else {
				assert.EqualValues(t, 1, attempts.Load(), "cancelled in-flight calls should never retry")
			}
		})
	}
}

func TestGetTokenTransportFailure(t *testing.T) {
	c, attempts := newTestClient(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("failed dial should never reach server") })
	transport, ok := c.httpClient.Transport.(*http.Transport)
	require.True(t, ok, "native client must use an HTTP transport")
	var dials atomic.Int64
	transport.DialContext = func(context.Context, string, string) (net.Conn, error) {
		dials.Add(1)
		return nil, net.ErrClosed
	}
	got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
	assert.ErrorIs(t, err, net.ErrClosed, "transport failures should preserve their native cause")
	assert.Nil(t, got, "transport failures should not return data")
	assert.EqualValues(t, 1, dials.Load(), "transport failures should never retry")
	assert.Zero(t, attempts.Load(), "failed dial should make no HTTP requests")
}

func TestGetTokenConnectionClosedWithoutResponse(t *testing.T) {
	var received atomic.Int64
	c, attempts := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if received.Add(1) == 2 {
			hijacker, ok := w.(http.Hijacker)
			if !assert.True(t, ok, "HTTP/1 server should support hijacking") {
				return
			}
			conn, _, err := hijacker.Hijack()
			if !assert.NoError(t, err, "server should take ownership of the connection") {
				return
			}
			assert.NoError(t, conn.Close(), "server should close without a response")
			return
		}
		_, _ = w.Write([]byte(`{"symbol":"BTC","found":false}`))
	})
	_, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
	require.NoError(t, err, "first request must complete before the connection-close scenario")
	got, err := c.GetToken(t.Context(), &TokenRequest{Symbol: "BTC"})
	assert.Error(t, err, "connection closure should fail without transparent replay")
	assert.Nil(t, got, "connection closure should not return a replayed report")
	assert.EqualValues(t, 2, attempts.Load(), "two public calls should send only two requests")
	t.Logf("connection-close verification backend=%s public_calls=2 received_requests=%d error=%v external=0", json.Implementation, attempts.Load(), err)
}

func TestGetTokenVerboseRedactsKey(t *testing.T) {
	var mu sync.Mutex
	var output strings.Builder
	require.NoError(t, gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings()), "logging must initialise")
	gctlog.SetCustomLogHook(func(_, _ string, a ...any) bool {
		mu.Lock()
		defer mu.Unlock()
		_, _ = fmt.Fprintln(&output, a...)
		return true
	})
	t.Cleanup(func() {
		gctlog.SetCustomLogHook(nil)
		assert.NoError(t, gctlog.SetGlobalLogConfig(&gctlog.Config{}), "logging should reset")
	})
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"synthetic-test-key"}`))
	})
	_, err := c.GetToken(request.WithVerbose(t.Context()), &TokenRequest{Symbol: "BTC"})
	require.Error(t, err, "HTTP failure must return an error")
	assert.NotContains(t, err.Error(), "synthetic-test-key", "error should redact the key")
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, output.String(), "request header", "verbose request logging should actually run")
	assert.NotContains(t, output.String(), "synthetic-test-key", "logs should not expose the key")
}
