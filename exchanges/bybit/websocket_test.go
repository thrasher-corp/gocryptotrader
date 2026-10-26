package bybit

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

var errUnexpectedRequest = errors.New("unexpected request type")

// FixtureConnection is a websocket connection that answers requests without a server
type FixtureConnection struct {
	url                               string
	dialError                         error
	sendMessageReturnResponseOverride []byte
	// reply answers each request sent through SendMessageReturnResponse when set
	reply func(request any) ([]byte, error)
	sent  []any
	match *websocket.Match
	websocket.Connection
}

func (d *FixtureConnection) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {}

func (d *FixtureConnection) GetURL() string { return d.url }

func (d *FixtureConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return d.dialError
}

func (d *FixtureConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, req any) ([]byte, error) {
	d.sent = append(d.sent, req)
	if d.reply != nil {
		return d.reply(req)
	}
	if d.sendMessageReturnResponseOverride != nil {
		return d.sendMessageReturnResponseOverride, nil
	}
	return []byte(`{"success":true,"ret_msg":"subscribe","conn_id":"5758770c-8152-4545-a84f-dae089e56499","req_id":"1","op":"subscribe"}`), nil
}

func (d *FixtureConnection) SendJSONMessage(context.Context, request.EndpointLimit, any) error {
	return nil
}

func (d *FixtureConnection) RequireMatchWithData(signature any, data []byte) error {
	return d.match.RequireMatchWithData(signature, data)
}

func (d *FixtureConnection) IncomingWithData(signature any, data []byte) bool {
	return d.match.IncomingWithData(signature, data)
}

// signatureConnection is a FixtureConnection that records the signature each request waits on, which decides whether
// the reply can reach it
type signatureConnection struct {
	*FixtureConnection
	signatures []any
}

func (c *signatureConnection) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, req any) ([]byte, error) {
	c.signatures = append(c.signatures, signature)
	return c.FixtureConnection.SendMessageReturnResponse(ctx, epl, signature, req)
}

// pingConnection is a FixtureConnection that records the heartbeat each SetupPingHandler call asks for
type pingConnection struct {
	*FixtureConnection
	limits   []request.EndpointLimit
	handlers []websocket.PingHandler
}

func (c *pingConnection) SetupPingHandler(epl request.EndpointLimit, handler websocket.PingHandler) {
	c.limits = append(c.limits, epl)
	c.handlers = append(c.handlers, handler)
}

// replySequence returns a reply func answering each request with the next of replies in turn; an empty reply fails
// the send with errFixtureSend
func replySequence(replies ...string) func(any) ([]byte, error) {
	return func(any) ([]byte, error) {
		if len(replies) == 0 {
			return nil, errUnexpectedRequest
		}
		reply := replies[0]
		replies = replies[1:]
		if reply == "" {
			return nil, errFixtureSend
		}
		return []byte(reply), nil
	}
}

// websocketTestInstance returns an exchange of its own for a websocket test, without the recorded REST responses
func websocketTestInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	return ex
}

// withMockCredentials gives ex the placeholder credentials, which account storage keys balances by
func withMockCredentials(ex *Exchange) *Exchange {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	return ex
}

// stopWebsocket disables and shuts down the websocket of ex, so its connection monitor does not reconnect to a closed
// mock server
func stopWebsocket(t *testing.T, ex *Exchange) {
	t.Helper()
	assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
	if err := ex.Websocket.Shutdown(); err != nil {
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail because the websocket is not connected")
	}
}

// fixtureFrames returns the frames a fixture file holds, one per line
func fixtureFrames(tb testing.TB, path string) [][]byte {
	tb.Helper()
	f, err := os.Open(path)
	require.NoErrorf(tb, err, "fixture %s must open", path)
	defer f.Close()
	var frames [][]byte
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		if line := bytes.TrimSpace(s.Bytes()); len(line) != 0 {
			frames = append(frames, bytes.Clone(line))
		}
	}
	require.NoErrorf(tb, s.Err(), "fixture %s must read", path)
	require.NotEmptyf(tb, frames, "fixture %s must hold frames", path)
	return frames
}

// decodeFramesStrictly decodes each frame into T, refusing keys T does not model
func decodeFramesStrictly[T any](tb testing.TB, frames [][]byte) []T {
	tb.Helper()
	decoded := make([]T, len(frames))
	for i, frame := range frames {
		d := json.NewDecoder(bytes.NewReader(frame))
		d.DisallowUnknownFields()
		require.NoErrorf(tb, d.Decode(&decoded[i]), "frame %d must decode into %T without unknown keys", i, decoded[i])
	}
	return decoded
}

// liveStreamFrames subscribes to topics on a Bybit stream and returns up to perTopic frames pushed for each topic, and
// the reply to the subscription, within the wait. Reads run in a goroutine so the window never needs a read deadline
func liveStreamFrames(tb testing.TB, streamURL string, wait time.Duration, perTopic int, topics ...string) (frames [][]byte, reply []byte) {
	tb.Helper()
	conn, resp, err := gws.DefaultDialer.DialContext(tb.Context(), streamURL, nil)
	if resp != nil {
		resp.Body.Close()
	}
	require.NoError(tb, err, "Dial must not error")
	defer conn.Close()
	read := make(chan []byte, 256)
	go func() {
		defer close(read)
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case read <- p:
			default: // A busy topic must not stall the reader; later frames are not needed
			}
		}
	}()
	req, err := json.Marshal(&SubscriptionRequest{RequestID: "1", Operation: subscribeOperation, Arguments: topics})
	require.NoError(tb, err, "Marshal must not error")
	require.NoError(tb, conn.WriteMessage(gws.TextMessage, req), "WriteMessage must not error")
	counts := make(map[string]int, len(topics))
	window := time.After(wait)
	heartbeat := time.NewTicker(websocketPingInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-heartbeat.C:
			require.NoError(tb, conn.WriteMessage(gws.TextMessage, []byte(`{"op":"ping"}`)), "WriteMessage must not error for a ping")
		case p, ok := <-read:
			require.True(tb, ok, "stream must stay open")
			topic, _ := jsonparser.GetString(p, "topic")
			switch {
			case topic == "" && reply == nil:
				reply = p
			case slices.Contains(topics, topic) && counts[topic] < perTopic:
				counts[topic]++
				frames = append(frames, p)
			}
			if reply != nil && len(counts) == len(topics) && !slices.ContainsFunc(slices.Collect(maps.Values(counts)), func(c int) bool { return c < perTopic }) {
				return frames, reply
			}
		case <-window:
			require.NotNil(tb, reply, "subscription must be answered")
			return frames, reply
		}
	}
}

func TestStreamOrigin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		url string
		exp string
		err error
	}{
		{url: spotPublic, exp: mainnetStreamOrigin},
		{url: "wss://stream-testnet.bybit.com/v5/public/spot", exp: "wss://stream-testnet.bybit.com"},
		{url: "wss://stream.bybit.eu/v5/public/spot", exp: "wss://stream.bybit.eu"},
		{url: "ws://127.0.0.1:8443/bybit/v5/public/spot?region=eu#spot", exp: "ws://127.0.0.1:8443"},
		{url: "", err: errNoStreamOrigin},
		{url: "/v5/public/spot", err: errNoStreamOrigin},
		{url: "//stream.bybit.com/v5/public/spot", err: errNoStreamOrigin},
		{url: "wss:///v5/public/spot", err: errNoStreamOrigin},
		{url: "wss:stream.bybit.com/v5/public/spot", err: errNoStreamOrigin},
		{url: "wss://stream.bybit.com:port/v5/public/spot", err: errNoStreamOrigin},
		{url: "wss://stream bybit.com/v5/public/spot", err: errNoStreamOrigin},
	} {
		got, err := streamOrigin(tc.url)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "streamOrigin should reject %q", tc.url)
			assert.ErrorContainsf(t, err, tc.url, "streamOrigin should name the URL %q it rejects", tc.url)
			assert.Emptyf(t, got, "streamOrigin should return no origin for %q", tc.url)
			continue
		}
		require.NoErrorf(t, err, "streamOrigin must not error for %q", tc.url)
		assert.Equalf(t, tc.exp, got, "streamOrigin should return the scheme and host of %q", tc.url)
	}
}

func TestAuxiliaryStreamURL(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		url string
		exp string
		// expStatus is the origin of the system status stream where it differs from the other streams'
		expStatus string
		err       error
	}{
		{url: spotPublic, exp: mainnetStreamOrigin},
		{url: "wss://stream-testnet.bybit.com/v5/public/spot", exp: "wss://stream-testnet.bybit.com"},
		{url: "ws://127.0.0.1:8443/v5/public/spot", exp: "ws://127.0.0.1:8443"},
		{url: "wss://stream.bybit.tr/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.bybit.kz/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.bybit.ae/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.bybit.eu/v5/public/spot", exp: mainnetStreamOrigin, expStatus: "wss://stream.bybit.eu"},
		{url: "wss://stream.bybitgeorgia.ge:443/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.bybit.id/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.manepa.jp/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://stream.spark-fintech.com/v5/public/spot", exp: mainnetStreamOrigin},
		// The regional sites' testnet hosts lack most of the same streams, which Bybit's testnet serves
		{url: "wss://stream-testnet.spark-fintech.com/v5/public/spot", exp: "wss://stream-testnet.bybit.com"},
		{url: "wss://stream-testnet.bybit.kz/v5/public/spot", exp: "wss://stream-testnet.bybit.com"},
		{url: "wss://stream-testnet.bybit.eu/v5/public/spot", exp: "wss://stream-testnet.bybit.com", expStatus: "wss://stream-testnet.bybit.eu"},
		{url: "wss://STREAM-TESTNET.BYBIT.AE/v5/public/spot", exp: "wss://stream-testnet.bybit.com"},
		// Hosts compare without regard to case, and a server Bybit does not name keeps its streams
		{url: "wss://STREAM.BYBIT.TR/v5/public/spot", exp: mainnetStreamOrigin},
		{url: "wss://STREAM.BYBIT.EU/v5/public/spot", exp: mainnetStreamOrigin, expStatus: "wss://STREAM.BYBIT.EU"},
		{url: "ws://stream.localhost:8443/v5/public/spot", exp: "ws://stream.localhost:8443"},
		{url: "wss://stream-testnet.bybits.org/v5/public/spot", exp: "wss://stream-testnet.bybits.org"},
		{url: "/v5/public/spot", err: errNoStreamOrigin},
	} {
		for _, path := range []string{systemStatusPath, advancedEarnPath, spreadPublicPath, rfqPublicPath, eventPublicPath} {
			got, err := auxiliaryStreamURL(tc.url, path)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "auxiliaryStreamURL should reject %q", tc.url)
				continue
			}
			require.NoErrorf(t, err, "auxiliaryStreamURL must not error for %q", tc.url)
			exp := tc.exp
			if path == systemStatusPath && tc.expStatus != "" {
				exp = tc.expStatus
			}
			assert.Equalf(t, exp+path, got, "auxiliaryStreamURL should place %s of %q on %s", path, tc.url, exp)
		}
	}
	// Every regional site's testnet host sends them to Bybit's testnet, apart from the EU testnet's status stream
	for _, host := range regionalStreamHosts {
		testnet := "wss://" + strings.Replace(host, "stream.", "stream-testnet.", 1)
		for _, path := range []string{systemStatusPath, advancedEarnPath, spreadPublicPath, rfqPublicPath, eventPublicPath} {
			got, err := auxiliaryStreamURL(testnet+"/v5/public/spot", path)
			require.NoErrorf(t, err, "auxiliaryStreamURL must not error for %q", testnet)
			exp := testnetStreamOrigin + path
			if host == euStreamHost && path == systemStatusPath {
				exp = testnet + path
			}
			assert.Equalf(t, exp, got, "auxiliaryStreamURL should place %s of %q on %s", path, testnet, exp)
		}
	}
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	err := e.WebsocketConnect(t.Context(), &FixtureConnection{dialError: nil})
	require.NoError(t, err, "WebsocketConnect must not error")
	exp := errors.New("dial error")
	err = e.WebsocketConnect(t.Context(), &FixtureConnection{dialError: exp})
	require.ErrorIs(t, err, exp, "WebsocketConnect must return the dial error")

	conn := &pingConnection{FixtureConnection: &FixtureConnection{}}
	require.NoError(t, e.WebsocketConnect(t.Context(), conn), "WebsocketConnect must not error")
	assert.Equal(t, []request.EndpointLimit{wsSubscriptionEPL}, conn.limits, "WebsocketConnect should start one heartbeat under the subscription limit")
	assert.Equal(t, []websocket.PingHandler{{MessageType: gws.TextMessage, Message: []byte(`{"op":"ping"}`), Delay: 20 * time.Second}}, conn.handlers, "WebsocketConnect should send a text ping every 20 seconds, as Bybit recommends")
	failing := &pingConnection{FixtureConnection: &FixtureConnection{dialError: exp}}
	require.ErrorIs(t, e.WebsocketConnect(t.Context(), failing), exp, "WebsocketConnect must return the dial error")
	assert.Empty(t, failing.handlers, "WebsocketConnect should start no heartbeat on a connection it could not dial")
}

func TestWebsocketAuthenticatePrivateConnection(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	err := ex.WebsocketAuthenticatePrivateConnection(t.Context(), &FixtureConnection{})
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "WebsocketAuthenticatePrivateConnection must error without authentication support")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	conn := &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"success":false,"ret_msg":"Invalid apikey","op":"auth","conn_id":"cejreaspqfh3sjdnldmg-p","req_id":"1"}`)}
	err = ex.WebsocketAuthenticatePrivateConnection(ctx, conn)
	require.ErrorIs(t, err, errWebsocketAuthFailed, "WebsocketAuthenticatePrivateConnection must report a rejected authentication")

	err = ex.WebsocketAuthenticatePrivateConnection(ctx, &FixtureConnection{reply: func(any) ([]byte, error) { return nil, errors.New("connection lost") }})
	require.ErrorContains(t, err, "connection lost", "WebsocketAuthenticatePrivateConnection must return a failed send")
	err = ex.WebsocketAuthenticatePrivateConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"success":"maybe"}`)})
	require.Error(t, err, "WebsocketAuthenticatePrivateConnection must error for an undecodable reply")
	err = ex.WebsocketAuthenticatePrivateConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`null`)})
	require.ErrorIs(t, err, errWebsocketAuthFailed, "WebsocketAuthenticatePrivateConnection must not accept a null reply")
	assert.ErrorIs(t, err, common.ErrNoResponse, "WebsocketAuthenticatePrivateConnection should report a null reply")

	conn = &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"success":true,"ret_msg":"","op":"auth","conn_id":"cejreaspqfh3sjdnldmg-p","req_id":"1"}`)}
	require.NoError(t, ex.WebsocketAuthenticatePrivateConnection(ctx, conn), "WebsocketAuthenticatePrivateConnection must not error")
	require.Len(t, conn.sent, 1, "WebsocketAuthenticatePrivateConnection must send one request")
	req, ok := conn.sent[0].(*AuthenticationRequest)
	require.True(t, ok, "WebsocketAuthenticatePrivateConnection must send an AuthenticationRequest")
	assert.Equal(t, authOperation, req.Operation, "request operation should be auth")
	assert.NotEmpty(t, req.RequestID, "request should carry a request ID to match the reply")
}

func TestGetAuthenticationPayload(t *testing.T) {
	t.Parallel()
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	start := time.Now()
	req, err := ex.GetAuthenticationPayload(ctx, "10001")
	require.NoError(t, err, "GetAuthenticationPayload must not error")
	require.Len(t, req.Arguments, 3, "arguments must hold the key, expiry and signature")
	assert.Equal(t, "10001", req.RequestID, "request ID should be kept")
	assert.Equal(t, authOperation, req.Operation, "operation should be auth")
	assert.Equal(t, "mock-api-key", req.Arguments[0], "first argument should be the API key")
	expires, ok := req.Arguments[1].(int64)
	require.True(t, ok, "expiry must be Unix milliseconds")
	assert.Greater(t, expires, start.UnixMilli(), "expiry should be in the future")
	sig, ok := req.Arguments[2].(string)
	require.True(t, ok, "signature must be a string")
	expSig := getSign("GET/realtime"+strconv.FormatInt(expires, 10), "mock-api-secret")
	assert.Equal(t, expSig, sig, "signature should sign GET/realtime and the expiry with the secret")
}

func TestWebsocketAuthenticationArguments(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	_, err := ex.websocketAuthenticationArguments(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "websocketAuthenticationArguments must error without authentication support")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	start := time.Now()
	args, err := ex.websocketAuthenticationArguments(ctx)
	end := time.Now()
	require.NoError(t, err, "websocketAuthenticationArguments must not error")
	require.Len(t, args, 3, "websocketAuthenticationArguments must return the key, expiry and signature")
	expires, ok := args[1].(int64)
	require.True(t, ok, "websocketAuthenticationArguments must return the expiry in Unix milliseconds")
	// Bybit refuses an expiry that has passed; the arguments stay valid for six hours
	assert.GreaterOrEqual(t, expires, start.Add(6*time.Hour).UnixMilli(), "websocketAuthenticationArguments should expire six hours ahead")
	assert.LessOrEqual(t, expires, end.Add(6*time.Hour).UnixMilli(), "websocketAuthenticationArguments should expire six hours ahead")
	mac := hmac.New(sha256.New, []byte("mock-api-secret"))
	_, err = mac.Write([]byte("GET/realtime" + strconv.FormatInt(expires, 10)))
	require.NoError(t, err, "Write must not error")
	exp := []any{"mock-api-key", expires, hex.EncodeToString(mac.Sum(nil))}
	assert.Equal(t, exp, args, "websocketAuthenticationArguments should return the key, the expiry and the HMAC-SHA256 of GET/realtime and the expiry")
}

func TestSendSubscriptions(t *testing.T) {
	t.Parallel()
	sub := func(topic string) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(topic, "USDT", "")}, QualifiedChannel: "tickers." + topic + "USDT"}
	}
	reject := func(named ...string) func(any) ([]byte, error) {
		return func(r any) ([]byte, error) {
			req, ok := r.(*SubscriptionRequest)
			if !ok {
				return nil, errUnexpectedRequest
			}
			for _, topic := range req.Arguments {
				if slices.Contains(named, topic) {
					return json.Marshal(&SubscriptionResponse{ReturnMessage: "error:handler not found,topic:" + topic, RequestID: req.RequestID, Operation: req.Operation})
				}
			}
			return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
		}
	}
	for _, tc := range []struct {
		name         string
		topics       []string
		reply        func(any) ([]byte, error)
		expSent      [][]string
		expStored    []string
		expRejected  []string
		expErrString string
	}{
		{
			name:      "all accepted",
			topics:    []string{"A", "B", "C"},
			reply:     reject(),
			expSent:   [][]string{{"tickers.AUSDT", "tickers.BUSDT", "tickers.CUSDT"}},
			expStored: []string{"tickers.AUSDT", "tickers.BUSDT", "tickers.CUSDT"},
		},
		{
			name:        "one bad topic in the first batch costs only itself",
			topics:      []string{"A", "BAD", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"},
			reply:       reject("tickers.BADUSDT"),
			expSent:     [][]string{{"tickers.AUSDT", "tickers.BADUSDT", "tickers.CUSDT", "tickers.DUSDT", "tickers.EUSDT", "tickers.FUSDT", "tickers.GUSDT", "tickers.HUSDT", "tickers.IUSDT", "tickers.JUSDT"}, {"tickers.AUSDT", "tickers.CUSDT", "tickers.DUSDT", "tickers.EUSDT", "tickers.FUSDT", "tickers.GUSDT", "tickers.HUSDT", "tickers.IUSDT", "tickers.JUSDT"}, {"tickers.KUSDT", "tickers.LUSDT"}},
			expStored:   []string{"tickers.AUSDT", "tickers.CUSDT", "tickers.DUSDT", "tickers.EUSDT", "tickers.FUSDT", "tickers.GUSDT", "tickers.HUSDT", "tickers.IUSDT", "tickers.JUSDT", "tickers.KUSDT", "tickers.LUSDT"},
			expRejected: []string{"tickers.BADUSDT"},
		},
		{
			name:        "each bad topic named in turn is dropped",
			topics:      []string{"A", "BAD", "WORSE"},
			reply:       reject("tickers.BADUSDT", "tickers.WORSEUSDT"),
			expSent:     [][]string{{"tickers.AUSDT", "tickers.BADUSDT", "tickers.WORSEUSDT"}, {"tickers.AUSDT", "tickers.WORSEUSDT"}, {"tickers.AUSDT"}},
			expStored:   []string{"tickers.AUSDT"},
			expRejected: []string{"tickers.BADUSDT", "tickers.WORSEUSDT"},
		},
		{
			name:   "a rejection naming no topic rejects its batch only",
			topics: []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"},
			reply: func(r any) ([]byte, error) {
				req, ok := r.(*SubscriptionRequest)
				if !ok {
					return nil, errUnexpectedRequest
				}
				if len(req.Arguments) == subscriptionBatchSize {
					return json.Marshal(&SubscriptionResponse{ReturnMessage: "args size >10", RequestID: req.RequestID, Operation: req.Operation})
				}
				return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
			},
			expSent:      [][]string{{"tickers.AUSDT", "tickers.BUSDT", "tickers.CUSDT", "tickers.DUSDT", "tickers.EUSDT", "tickers.FUSDT", "tickers.GUSDT", "tickers.HUSDT", "tickers.IUSDT", "tickers.JUSDT"}, {"tickers.KUSDT"}},
			expStored:    []string{"tickers.KUSDT"},
			expRejected:  []string{"tickers.AUSDT", "tickers.JUSDT"},
			expErrString: "args size >10",
		},
		{
			name:         "a failed send skips its batch only",
			topics:       []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"},
			reply:        func(r any) ([]byte, error) { return sendOrFail(r, "tickers.AUSDT") },
			expSent:      [][]string{{"tickers.AUSDT", "tickers.BUSDT", "tickers.CUSDT", "tickers.DUSDT", "tickers.EUSDT", "tickers.FUSDT", "tickers.GUSDT", "tickers.HUSDT", "tickers.IUSDT", "tickers.JUSDT"}, {"tickers.KUSDT"}},
			expStored:    []string{"tickers.KUSDT"},
			expRejected:  []string{"tickers.AUSDT"},
			expErrString: "connection lost",
		},
		{
			name:         "an undecodable reply skips its batch only",
			topics:       []string{"A"},
			reply:        func(any) ([]byte, error) { return []byte(`{"success":"maybe"}`), nil },
			expSent:      [][]string{{"tickers.AUSDT"}},
			expRejected:  []string{"tickers.AUSDT"},
			expErrString: "success",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			subs := make(subscription.List, len(tc.topics))
			for i, topic := range tc.topics {
				subs[i] = sub(topic)
			}
			conn := &FixtureConnection{reply: tc.reply}
			err := ex.sendSubscriptions(t.Context(), conn, subscribeOperation, subs)
			sent := make([][]string, len(conn.sent))
			for i, r := range conn.sent {
				req, ok := r.(*SubscriptionRequest)
				require.True(t, ok, "sendSubscriptions must send SubscriptionRequests")
				assert.Equal(t, subscribeOperation, req.Operation, "request operation should be subscribe")
				sent[i] = req.Arguments
			}
			assert.Equal(t, tc.expSent, sent, "sendSubscriptions should send every batch, resending a rejected one without the topics it names")
			assert.ElementsMatch(t, tc.expStored, ex.Websocket.GetSubscriptions().QualifiedChannels(), "sendSubscriptions should store only accepted subscriptions")
			if len(tc.expRejected) == 0 {
				require.NoError(t, err, "sendSubscriptions must not error")
				return
			}
			require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendSubscriptions must report rejected topics as subscription failures")
			for _, topic := range tc.expRejected {
				assert.ErrorContains(t, err, topic, "error should name each rejected topic")
			}
			if tc.expErrString != "" {
				assert.ErrorContains(t, err, tc.expErrString, "error should carry the cause")
			}
		})
	}

	ex := websocketTestInstance(t)
	subs := subscription.List{sub("A"), sub("B")}
	require.NoError(t, ex.sendSubscriptions(t.Context(), &FixtureConnection{reply: reject()}, subscribeOperation, subs), "sendSubscriptions must not error")
	conn := &FixtureConnection{reply: reject()}
	require.NoError(t, ex.sendSubscriptions(t.Context(), conn, unsubscribeOperation, subs), "sendSubscriptions must not error when unsubscribing")
	require.Len(t, conn.sent, 1, "unsubscribing must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "sendSubscriptions must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribed subscriptions should be removed")

	// Subscriptions sharing a topic are sent in one batch, so the topic is sent once wherever the batches fall
	shared := websocketTestInstance(t)
	topics := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	subs = make(subscription.List, 0, len(topics)+1)
	for _, topic := range topics {
		subs = append(subs, sub(topic))
	}
	alias := sub("A")
	alias.Channel = chanPublicTicker
	subs = append(subs, alias)
	conn = &FixtureConnection{reply: reject()}
	require.NoError(t, shared.sendSubscriptions(t.Context(), conn, subscribeOperation, subs), "sendSubscriptions must not error for subscriptions sharing a topic")
	require.Len(t, conn.sent, 1, "sendSubscriptions must send a topic shared across batches once")
	req, ok = conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "sendSubscriptions must send a SubscriptionRequest")
	assert.Len(t, req.Arguments, len(topics), "sendSubscriptions should send each topic once")
	assert.Len(t, shared.Websocket.GetSubscriptions(), len(subs), "sendSubscriptions should store every subscription sharing a topic")

	// A second orderbook depth of a symbol is neither sent nor stored, and the rest of the request goes ahead
	books := websocketTestInstance(t)
	book := func(levels int) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Levels: levels, QualifiedChannel: "orderbook." + strconv.Itoa(levels) + ".BTCUSDT"}
	}
	conn = &FixtureConnection{reply: reject()}
	err := books.sendSubscriptions(t.Context(), conn, subscribeOperation, subscription.List{book(50), book(1), sub("A")})
	require.ErrorIs(t, err, errOrderbookDepthConflict, "sendSubscriptions must refuse a second depth of a symbol asked for in one request")
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendSubscriptions must report a refusal as a subscription failure")
	assert.ErrorContains(t, err, "orderbook.1.BTCUSDT conflicts with depth 50", "error should name the refused topic and the depth asked for first")
	require.Len(t, conn.sent, 1, "sendSubscriptions must send the rest of the request")
	req, ok = conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "sendSubscriptions must send a SubscriptionRequest")
	assert.Equal(t, []string{"orderbook.50.BTCUSDT", "tickers.AUSDT"}, req.Arguments, "sendSubscriptions should send every topic but the refused one")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCUSDT", "tickers.AUSDT"}, books.Websocket.GetSubscriptions().QualifiedChannels(), "sendSubscriptions should store every subscription but the refused one")
	err = books.sendSubscriptions(t.Context(), conn, subscribeOperation, subscription.List{book(200)})
	require.ErrorIs(t, err, errOrderbookDepthConflict, "sendSubscriptions must refuse another depth of a symbol the connection holds")
	assert.ErrorContains(t, err, "orderbook.200.BTCUSDT conflicts with depth 50", "error should name the refused topic and the depth held")
	assert.Len(t, conn.sent, 1, "sendSubscriptions should send nothing for a request whose only subscription it refuses")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCUSDT", "tickers.AUSDT"}, books.Websocket.GetSubscriptions().QualifiedChannels(), "sendSubscriptions should not store a refused subscription")
}

// sendOrFail fails a request that holds topic and accepts the rest
func sendOrFail(r any, topic string) ([]byte, error) {
	req, ok := r.(*SubscriptionRequest)
	if !ok {
		return nil, errUnexpectedRequest
	}
	if slices.Contains(req.Arguments, topic) {
		return nil, errors.New("connection lost")
	}
	return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
}

func TestSendSubscriptionBatch(t *testing.T) {
	t.Parallel()
	const (
		subscribed   = `{"success":true,"ret_msg":"subscribe","conn_id":"2324d924-aa4d-45b0-a858-7b8be29ab52b","req_id":"10001","op":"subscribe"}`
		unsubscribed = `{"success":true,"ret_msg":"","conn_id":"3cd84cb1-4d06-4a05-930a-2efe5fc70f0f","req_id":"10002","op":"unsubscribe"}`
		rejectsBad   = `{"success":false,"ret_msg":"Invalid symbol :[tickers.BADUSDT]","conn_id":"d9avrat34tmf5pih98i0-gsmne","req_id":"10001","op":"subscribe"}`
		rejectsAll   = `{"success":false,"ret_msg":"args size >10","conn_id":"d9avrat34tmf5pih98i0-gsmne","req_id":"10001","op":"subscribe"}`
	)
	for _, tc := range []struct {
		name      string
		operation string
		stored    bool // the batch is stored before it is sent
		bases     []string
		replies   []string // the reply to each request in turn; an empty one fails the send
		expSent   [][]string
		expStored []string
		expErrs   []error
		expNamed  []string // topics the error names
	}{
		{name: "subscribed", operation: subscribeOperation, bases: []string{"A", "B"}, replies: []string{subscribed}, expSent: [][]string{{"tickers.AUSDT", "tickers.BUSDT"}}, expStored: []string{"tickers.AUSDT", "tickers.BUSDT"}},
		{name: "rejected topic is resent without it", operation: subscribeOperation, bases: []string{"A", "BAD", "C"}, replies: []string{rejectsBad, subscribed}, expSent: [][]string{{"tickers.AUSDT", "tickers.BADUSDT", "tickers.CUSDT"}, {"tickers.AUSDT", "tickers.CUSDT"}}, expStored: []string{"tickers.AUSDT", "tickers.CUSDT"}, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"tickers.BADUSDT"}},
		{name: "rejection naming no topic rejects the batch", operation: subscribeOperation, bases: []string{"A", "B"}, replies: []string{rejectsAll}, expSent: [][]string{{"tickers.AUSDT", "tickers.BUSDT"}}, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"tickers.AUSDT", "tickers.BUSDT", "args size >10"}},
		{name: "failed send", operation: subscribeOperation, bases: []string{"A"}, replies: []string{""}, expSent: [][]string{{"tickers.AUSDT"}}, expErrs: []error{websocket.ErrSubscriptionFailure, errFixtureSend}, expNamed: []string{"tickers.AUSDT"}},
		{name: "failed resend keeps the rejection", operation: subscribeOperation, bases: []string{"A", "BAD"}, replies: []string{rejectsBad, ""}, expSent: [][]string{{"tickers.AUSDT", "tickers.BADUSDT"}, {"tickers.AUSDT"}}, expErrs: []error{websocket.ErrSubscriptionFailure, errFixtureSend}, expNamed: []string{"tickers.BADUSDT", "tickers.AUSDT"}},
		{name: "undecodable reply", operation: subscribeOperation, bases: []string{"A"}, replies: []string{`{"success":"maybe"}`}, expSent: [][]string{{"tickers.AUSDT"}}, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"tickers.AUSDT"}},
		{name: "null reply", operation: subscribeOperation, bases: []string{"A", "B"}, replies: []string{`null`}, expSent: [][]string{{"tickers.AUSDT", "tickers.BUSDT"}}, expErrs: []error{websocket.ErrSubscriptionFailure, common.ErrNoResponse}, expNamed: []string{"tickers.AUSDT", "tickers.BUSDT"}},
		{name: "unsubscribed", operation: unsubscribeOperation, stored: true, bases: []string{"A", "B"}, replies: []string{unsubscribed}, expSent: [][]string{{"tickers.AUSDT", "tickers.BUSDT"}}},
		{name: "unsubscribed topic not stored", operation: unsubscribeOperation, bases: []string{"A"}, replies: []string{unsubscribed}, expSent: [][]string{{"tickers.AUSDT"}}, expErrs: []error{subscription.ErrNotFound}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			batch := make(subscription.List, len(tc.bases))
			for i, base := range tc.bases {
				batch[i] = &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(base, "USDT", "")}, QualifiedChannel: "tickers." + base + "USDT"}
			}
			conn := &signatureConnection{FixtureConnection: &FixtureConnection{reply: replySequence(tc.replies...)}}
			if tc.stored {
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, batch...), "AddSuccessfulSubscriptions must not error")
			}
			err := ex.sendSubscriptionBatch(t.Context(), conn, tc.operation, batch)
			sent := make([][]string, len(conn.sent))
			for i, r := range conn.sent {
				req, ok := r.(*SubscriptionRequest)
				require.True(t, ok, "sendSubscriptionBatch must send SubscriptionRequests")
				assert.Equal(t, tc.operation, req.Operation, "sendSubscriptionBatch should send the operation")
				assert.Equal(t, req.RequestID, conn.signatures[i], "sendSubscriptionBatch should wait on the request ID, which the reply echoes")
				sent[i] = req.Arguments
			}
			assert.Equal(t, tc.expSent, sent, "sendSubscriptionBatch should send the batch, and resend it without the topics a rejection names")
			assert.ElementsMatch(t, tc.expStored, ex.Websocket.GetSubscriptions().QualifiedChannels(), "sendSubscriptionBatch should store only the subscriptions Bybit accepted")
			if len(tc.expErrs) == 0 {
				require.NoError(t, err, "sendSubscriptionBatch must not error")
				return
			}
			for _, expErr := range tc.expErrs {
				require.ErrorIs(t, err, expErr, "sendSubscriptionBatch must return the expected error")
			}
			for _, named := range tc.expNamed {
				assert.ErrorContains(t, err, named, "sendSubscriptionBatch should name each failed topic and Bybit's reason")
			}
		})
	}
}

// TestSpotSubscribeRejectedBatch reproduces issue #2431 through the websocket manager: a server rejecting the batch
// that holds one invalid symbol must not stop the later batches, and only the invalid topic may go unsubscribed
func TestSpotSubscribeRejectedBatch(t *testing.T) {
	t.Parallel()
	const badTopic = "orderbook.50.ZZZQUSDT"
	var mu sync.Mutex
	var received [][]string
	handler := func(_ testing.TB, msg []byte, c *gws.Conn) error {
		var req SubscriptionRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		if req.Operation != subscribeOperation {
			return nil
		}
		mu.Lock()
		received = append(received, req.Arguments)
		mu.Unlock()
		resp := &SubscriptionResponse{Success: true, ReturnMessage: "subscribe", ConnectionID: "d9avrat34tmf5pih98i0-gsmne", RequestID: req.RequestID, Operation: subscribeOperation}
		if slices.Contains(req.Arguments, badTopic) {
			resp.Success = false
			resp.ReturnMessage = "Invalid symbol :[" + badTopic + "]"
		}
		return c.WriteJSON(resp)
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
	}
	bases := []string{"ZZZQ", "BTC", "ETH", "SOL", "XRP", "DOGE", "ADA", "LTC", "DOT", "LINK", "TRX", "AVAX"}
	pairs := make(currency.Pairs, 0, len(bases))
	for _, base := range bases {
		pairs = append(pairs, currency.NewPair(currency.NewCode(base), currency.USDT))
	}
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50, Pairs: pairs}}
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, handler))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")

	err := ex.Websocket.Connect(context.WithoutCancel(t.Context()))
	t.Cleanup(func() { stopWebsocket(t, ex) })
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Connect must report the rejected subscription")
	assert.ErrorContains(t, err, badTopic, "Connect should name the rejected topic")

	exp := make([]string, 0, len(pairs)-1)
	for _, p := range pairs[1:] {
		exp = append(exp, "orderbook.50."+p.Base.String()+"USDT")
	}
	assert.ElementsMatch(t, exp, ex.Websocket.GetSubscriptions().QualifiedChannels(), "every topic but the rejected one should be subscribed")
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 3, "the server must receive the rejected batch, its resend and the next batch")
	assert.Contains(t, received[0], badTopic, "the first batch should hold the invalid topic")
	assert.NotContains(t, received[1], badTopic, "the resent batch should drop the invalid topic")
	assert.Len(t, received[1], subscriptionBatchSize-1, "the resent batch should keep every other topic")
	assert.Len(t, received[2], len(pairs)-subscriptionBatchSize, "the later batch should still be sent")
}

// TestSpotSharedTopic subscribes and unsubscribes two configured spot subscriptions sharing a topic through the
// websocket manager, against a stream that, as Bybit's does, carries each topic once and ends it on one unsubscribe
func TestSpotSharedTopic(t *testing.T) {
	t.Parallel()
	var (
		mu       sync.Mutex
		topics   = make(map[string]bool)
		requests []string
	)
	handler := func(tb testing.TB, msg []byte, c *gws.Conn) error {
		tb.Helper()
		var req SubscriptionRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		if req.Operation != subscribeOperation && req.Operation != unsubscribeOperation {
			return nil
		}
		resp := &SubscriptionResponse{Success: true, ConnectionID: "d9avrat34tmf5pih98i0-gsmne", RequestID: req.RequestID, Operation: req.Operation}
		mu.Lock()
		for _, topic := range req.Arguments {
			requests = append(requests, req.Operation+" "+topic)
			switch {
			case req.Operation == unsubscribeOperation:
				delete(topics, topic)
			case topics[topic]:
				// The spot stream leaves a second subscribe to a topic unanswered; refusing it, as the linear stream
				// does, fails the test at once rather than at the reply timeout
				resp.Success = false
				resp.ReturnMessage = "error:already subscribed,topic:" + topic
			default:
				topics[topic] = true
			}
		}
		mu.Unlock()
		return c.WriteJSON(resp)
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
	}
	pairs := currency.Pairs{currency.NewBTCUSDT()}
	// The generic ticker channel and Bybit's own name for it both expand to the tickers.BTCUSDT topic
	generic := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: pairs}
	named := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: chanPublicTicker, Pairs: pairs}
	ex.Features.Subscriptions = subscription.List{generic, named}
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, handler))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	require.NoError(t, ex.Websocket.Connect(context.WithoutCancel(t.Context())), "Connect must not error")
	t.Cleanup(func() { stopWebsocket(t, ex) })
	state := func() (stored, sent, held []string) {
		for _, s := range ex.Websocket.GetSubscriptions() {
			stored = append(stored, s.Channel+" "+s.QualifiedChannel)
		}
		mu.Lock()
		defer mu.Unlock()
		for topic := range topics {
			held = append(held, topic)
		}
		return stored, slices.Clone(requests), held
	}
	stored, sent, held := state()
	assert.ElementsMatch(t, []string{"ticker tickers.BTCUSDT", "tickers tickers.BTCUSDT"}, stored, "Connect should store both subscriptions")
	assert.Equal(t, []string{"subscribe tickers.BTCUSDT"}, sent, "Connect should subscribe the shared topic once")
	assert.Equal(t, []string{"tickers.BTCUSDT"}, held, "the stream should carry the topic")

	ex.Features.Subscriptions = subscription.List{generic}
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
	stored, sent, held = state()
	assert.Equal(t, []string{"ticker tickers.BTCUSDT"}, stored, "FlushChannels should remove the dropped subscription")
	assert.Equal(t, []string{"subscribe tickers.BTCUSDT"}, sent, "FlushChannels should not unsubscribe a topic another subscription holds")
	assert.Equal(t, []string{"tickers.BTCUSDT"}, held, "the stream should still carry the topic")

	ex.Features.Subscriptions = subscription.List{generic, named}
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error when a subscription shares a held topic")
	stored, sent, _ = state()
	assert.ElementsMatch(t, []string{"ticker tickers.BTCUSDT", "tickers tickers.BTCUSDT"}, stored, "FlushChannels should store a subscription to a held topic")
	assert.Equal(t, []string{"subscribe tickers.BTCUSDT"}, sent, "FlushChannels should not subscribe a topic the stream already carries")

	ex.Features.Subscriptions = subscription.List{named}
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
	ex.Features.Subscriptions = nil
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
	stored, sent, held = state()
	assert.Empty(t, stored, "FlushChannels should remove the last subscription")
	assert.Equal(t, []string{"subscribe tickers.BTCUSDT", "unsubscribe tickers.BTCUSDT"}, sent, "FlushChannels should unsubscribe the topic with its last subscription")
	assert.Empty(t, held, "the stream should no longer carry the topic")
}

// TestSpotOrderbookDepthFlush checks through the websocket manager that a flush changing the depth of a spot symbol's
// orderbook unsubscribes the old depth before it subscribes the new one, and that a second configured depth is refused
func TestSpotOrderbookDepthFlush(t *testing.T) {
	t.Parallel()
	var (
		mu       sync.Mutex
		topics   = make(map[string]bool)
		requests []string
	)
	handler := func(_ testing.TB, msg []byte, c *gws.Conn) error {
		var req SubscriptionRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		if req.Operation != subscribeOperation && req.Operation != unsubscribeOperation {
			return nil
		}
		mu.Lock()
		for _, topic := range req.Arguments {
			requests = append(requests, req.Operation+" "+topic)
			topics[topic] = req.Operation == subscribeOperation
		}
		mu.Unlock()
		return c.WriteJSON(&SubscriptionResponse{Success: true, ConnectionID: "d9avrat34tmf5pih98i0-gsmne", RequestID: req.RequestID, Operation: req.Operation})
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
	}
	book := func(levels int) *subscription.Subscription {
		return &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: levels, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	}
	ex.Features.Subscriptions = subscription.List{book(50)}
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, handler))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	require.NoError(t, ex.Websocket.Connect(context.WithoutCancel(t.Context())), "Connect must not error")
	t.Cleanup(func() { stopWebsocket(t, ex) })
	state := func() (stored, sent, carried []string) {
		stored = ex.Websocket.GetSubscriptions().QualifiedChannels()
		mu.Lock()
		defer mu.Unlock()
		for topic, on := range topics {
			if on {
				carried = append(carried, topic)
			}
		}
		return stored, slices.Clone(requests), carried
	}

	ex.Features.Subscriptions = subscription.List{book(200)}
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error when the depth of a symbol changes")
	stored, sent, carried := state()
	assert.Equal(t, []string{"orderbook.200.BTCUSDT"}, stored, "FlushChannels should store the new depth only")
	assert.Equal(t, []string{"subscribe orderbook.50.BTCUSDT", "unsubscribe orderbook.50.BTCUSDT", "subscribe orderbook.200.BTCUSDT"}, sent, "FlushChannels should unsubscribe the old depth before it subscribes the new one")
	assert.Equal(t, []string{"orderbook.200.BTCUSDT"}, carried, "the stream should carry the new depth only")

	ex.Features.Subscriptions = subscription.List{book(200), book(1)}
	err := ex.Websocket.FlushChannels(t.Context())
	require.ErrorIs(t, err, errOrderbookDepthConflict, "FlushChannels must report a second depth of a symbol")
	assert.ErrorContains(t, err, "orderbook.1.BTCUSDT conflicts with depth 200", "error should name the refused topic and the depth held")
	stored, sent, carried = state()
	assert.Equal(t, []string{"orderbook.200.BTCUSDT"}, stored, "FlushChannels should not store the refused depth")
	assert.Equal(t, []string{"subscribe orderbook.50.BTCUSDT", "unsubscribe orderbook.50.BTCUSDT", "subscribe orderbook.200.BTCUSDT"}, sent, "FlushChannels should not send the refused depth")
	assert.Equal(t, []string{"orderbook.200.BTCUSDT"}, carried, "the stream should carry the held depth only")
}

func TestSplitRejectedSubscriptions(t *testing.T) {
	t.Parallel()
	a := &subscription.Subscription{QualifiedChannel: "orderbook.50.BTCUSDT"}
	b := &subscription.Subscription{QualifiedChannel: "orderbook.50.ZZZQUSDT"}
	c := &subscription.Subscription{QualifiedChannel: "orderbook.50.BTCUSD"}
	for _, tc := range []struct {
		message      string
		expRejected  subscription.List
		expRemaining subscription.List
	}{
		{message: "Invalid symbol :[orderbook.50.ZZZQUSDT]", expRejected: subscription.List{b}, expRemaining: subscription.List{a, c}},
		{message: "error:handler not found,topic:orderbook.50.ZZZQUSDT", expRejected: subscription.List{b}, expRemaining: subscription.List{a, c}},
		{message: "Invalid symbol :[orderbook.50.ZZZQUSDT, orderbook.50.BTCUSD]", expRejected: subscription.List{b, c}, expRemaining: subscription.List{a}},
		{message: "error:handler not found,topic:orderbook.50.BTCUSD", expRejected: subscription.List{c}, expRemaining: subscription.List{a, b}},
		{message: "args size >10", expRejected: subscription.List{a, b, c}},
	} {
		rejected, remaining := splitRejectedSubscriptions(subscription.List{a, b, c}, tc.message)
		assert.Equalf(t, tc.expRejected, rejected, "splitRejectedSubscriptions should reject the named topics for %q", tc.message)
		assert.Equalf(t, tc.expRemaining, remaining, "splitRejectedSubscriptions should keep the other topics for %q", tc.message)
	}
}

func TestSendCommandSubscriptions(t *testing.T) {
	t.Parallel()
	subs := subscription.List{
		{Channel: chanPublicTicker, Asset: asset.Options, QualifiedChannel: "tickers.BTC-24SEP27-170000-P-USDT"},
		{Channel: chanPublicTrade, Asset: asset.Options, QualifiedChannel: "publicTrade.BTC"},
		{Channel: chanPublicTrade, Asset: asset.Options, QualifiedChannel: "publicTrade.BTC"},
	}
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{reply: func(any) ([]byte, error) {
		return []byte(`{"success":true,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":["publicTrade.BTC"],"successTopics":["tickers.BTC-24SEP27-170000-P-USDT"]},"type":"COMMAND_RESP"}`), nil
	}}
	err := ex.sendCommandSubscriptions(t.Context(), conn, subscribeOperation, subs)
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendCommandSubscriptions must report failed topics")
	assert.ErrorContains(t, err, "publicTrade.BTC", "error should name the failed topic")
	require.Len(t, conn.sent, 1, "sendCommandSubscriptions must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "sendCommandSubscriptions must send a SubscriptionRequest")
	assert.Equal(t, []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}, req.Arguments, "a topic shared by subscriptions should be sent once")
	assert.Equal(t, []string{"tickers.BTC-24SEP27-170000-P-USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "only applied topics should be stored")

	ex = websocketTestInstance(t)
	conn = &FixtureConnection{reply: func(any) ([]byte, error) {
		return []byte(`{"success":false,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":[],"successTopics":[]},"type":"COMMAND_RESP"}`), nil
	}}
	err = ex.sendCommandSubscriptions(t.Context(), conn, subscribeOperation, subs[:1])
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendCommandSubscriptions must treat an unsuccessful reply as a failure")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "nothing should be stored after an unsuccessful reply")

	err = ex.sendCommandSubscriptions(t.Context(), &FixtureConnection{reply: func(any) ([]byte, error) { return nil, errors.New("connection lost") }}, subscribeOperation, subs[:1])
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendCommandSubscriptions must report a failed send")
	err = ex.sendCommandSubscriptions(t.Context(), &FixtureConnection{reply: func(any) ([]byte, error) { return []byte(`{"success":"maybe"}`), nil }}, subscribeOperation, subs[:1])
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "sendCommandSubscriptions must report an undecodable reply")

	fresh := subscription.List{{Channel: chanPublicTicker, Asset: asset.Options, QualifiedChannel: "tickers.BTC-24SEP27-170000-P-USDT"}}
	conn = &FixtureConnection{reply: func(any) ([]byte, error) {
		return []byte(`{"success":true,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":[],"successTopics":["tickers.BTC-24SEP27-170000-P-USDT"]},"type":"COMMAND_RESP"}`), nil
	}}
	require.NoError(t, ex.sendCommandSubscriptions(t.Context(), conn, subscribeOperation, fresh), "sendCommandSubscriptions must not error")
	require.NoError(t, ex.sendCommandSubscriptions(t.Context(), conn, unsubscribeOperation, fresh), "sendCommandSubscriptions must not error when unsubscribing")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribed topics should be removed")

	// Option books come in 25 and 100 levels, and an option takes one of them on a connection
	optionBook := func(levels int) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}, Levels: levels, QualifiedChannel: "orderbook." + strconv.Itoa(levels) + ".BTC-24SEP27-170000-P-USDT"}
	}
	conn = &FixtureConnection{reply: func(any) ([]byte, error) {
		return []byte(`{"success":true,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":[],"successTopics":["orderbook.25.BTC-24SEP27-170000-P-USDT"]},"type":"COMMAND_RESP"}`), nil
	}}
	err = ex.sendCommandSubscriptions(t.Context(), conn, subscribeOperation, subscription.List{optionBook(25), optionBook(100)})
	require.ErrorIs(t, err, errOrderbookDepthConflict, "sendCommandSubscriptions must refuse a second depth of an option")
	assert.ErrorContains(t, err, "orderbook.100.BTC-24SEP27-170000-P-USDT conflicts with depth 25", "error should name the refused topic and the depth asked for first")
	require.Len(t, conn.sent, 1, "sendCommandSubscriptions must send the rest of the request")
	req, ok = conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "sendCommandSubscriptions must send a SubscriptionRequest")
	assert.Equal(t, []string{"orderbook.25.BTC-24SEP27-170000-P-USDT"}, req.Arguments, "sendCommandSubscriptions should not send the refused depth")
	assert.Equal(t, []string{"orderbook.25.BTC-24SEP27-170000-P-USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "sendCommandSubscriptions should not store the refused depth")
}

func TestSendCommandSubscriptionBatch(t *testing.T) {
	t.Parallel()
	const (
		applied      = `{"success":true,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":[],"successTopics":["tickers.BTC-24SEP27-170000-P-USDT","publicTrade.BTC"]},"type":"COMMAND_RESP"}`
		tradesFailed = `{"success":true,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":["publicTrade.BTC"],"successTopics":["tickers.BTC-24SEP27-170000-P-USDT"]},"type":"COMMAND_RESP"}`
		unsuccessful = `{"success":false,"conn_id":"dat43eb07optjqfk6030-1k2ih","data":{"failTopics":[],"successTopics":[]},"type":"COMMAND_RESP"}`
	)
	for _, tc := range []struct {
		name      string
		operation string
		stored    bool // the batch is stored before it is sent
		reply     string
		expStored []string
		expErrs   []error
		expNamed  []string // topics the error names
	}{
		{name: "applied", operation: subscribeOperation, reply: applied, expStored: []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC", "publicTrade.BTC"}},
		{name: "failed topic", operation: subscribeOperation, reply: tradesFailed, expStored: []string{"tickers.BTC-24SEP27-170000-P-USDT"}, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"publicTrade.BTC"}},
		{name: "unsuccessful reply rejects every topic", operation: subscribeOperation, reply: unsuccessful, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}},
		{name: "failed send", operation: subscribeOperation, expErrs: []error{websocket.ErrSubscriptionFailure, errFixtureSend}, expNamed: []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}},
		{name: "undecodable reply", operation: subscribeOperation, reply: `{"success":"maybe"}`, expErrs: []error{websocket.ErrSubscriptionFailure}, expNamed: []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}},
		{name: "null reply", operation: subscribeOperation, reply: `null`, expErrs: []error{websocket.ErrSubscriptionFailure, common.ErrNoResponse}, expNamed: []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}},
		{name: "unsubscribed", operation: unsubscribeOperation, stored: true, reply: applied},
		{name: "unsubscribed topics not stored", operation: unsubscribeOperation, reply: applied, expErrs: []error{subscription.ErrNotFound}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			otherContract := currency.NewPairWithDelimiter("BTC", "10OCT26-82750-C-USDT", currency.DashDelimiter)
			batch := subscription.List{
				{Channel: chanPublicTicker, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}, QualifiedChannel: "tickers.BTC-24SEP27-170000-P-USDT"},
				{Channel: chanPublicTrade, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}, QualifiedChannel: "publicTrade.BTC"},
				{Channel: chanPublicTrade, Asset: asset.Options, Pairs: currency.Pairs{otherContract}, QualifiedChannel: "publicTrade.BTC"},
			}
			conn := &signatureConnection{FixtureConnection: &FixtureConnection{reply: replySequence(tc.reply)}}
			if tc.stored {
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, batch...), "AddSuccessfulSubscriptions must not error")
			}
			err := ex.sendCommandSubscriptionBatch(t.Context(), conn, tc.operation, batch)
			require.Len(t, conn.sent, 1, "sendCommandSubscriptionBatch must send one request")
			req, ok := conn.sent[0].(*SubscriptionRequest)
			require.True(t, ok, "sendCommandSubscriptionBatch must send a SubscriptionRequest")
			assert.Equal(t, tc.operation, req.Operation, "sendCommandSubscriptionBatch should send the operation")
			assert.Equal(t, []string{"tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}, req.Arguments, "sendCommandSubscriptionBatch should send a topic shared by subscriptions once")
			assert.Equal(t, []any{commandResponseType}, conn.signatures, "sendCommandSubscriptionBatch should wait on COMMAND_RESP, since the reply echoes no request ID")
			assert.ElementsMatch(t, tc.expStored, ex.Websocket.GetSubscriptions().QualifiedChannels(), "sendCommandSubscriptionBatch should store only the applied subscriptions")
			if len(tc.expErrs) == 0 {
				require.NoError(t, err, "sendCommandSubscriptionBatch must not error")
				return
			}
			for _, expErr := range tc.expErrs {
				require.ErrorIs(t, err, expErr, "sendCommandSubscriptionBatch must return the expected error")
			}
			for _, named := range tc.expNamed {
				assert.ErrorContains(t, err, named, "sendCommandSubscriptionBatch should name each failed topic")
			}
		})
	}
}

func TestRecordSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{}
	subs := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "tickers.BTCUSDT"},
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Levels: 50, QualifiedChannel: "orderbook.50.BTCUSDT"},
	}
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subs), "recordSubscriptions must not error")
	assert.ElementsMatch(t, []string{"tickers.BTCUSDT", "orderbook.50.BTCUSDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "recordSubscriptions should store the subscriptions")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "recordSubscriptions should mark %s subscribed", s.QualifiedChannel)
	}
	err := ex.recordSubscriptions(conn, subscribeOperation, subs[:1])
	require.ErrorIs(t, err, subscription.ErrDuplicate, "recordSubscriptions must not store a subscription twice")

	require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, subs), "recordSubscriptions must not error when unsubscribing")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "recordSubscriptions should remove unsubscribed subscriptions")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "recordSubscriptions should mark %s unsubscribed", s.QualifiedChannel)
	}
	err = ex.recordSubscriptions(conn, unsubscribeOperation, subs[:1])
	require.ErrorIs(t, err, subscription.ErrNotFound, "recordSubscriptions must report removing a subscription it does not hold")
}

func TestHandleNoTopicWebsocketResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		frame    string
		match    string
		err      error
		relayed  bool
		consumed bool
	}{
		{name: "spot subscribe reply", frame: `{"success":true,"ret_msg":"subscribe","conn_id":"2324d924-aa4d-45b0-a858-7b8be29ab52b","req_id":"10001","op":"subscribe"}`, match: "10001", consumed: true},
		{name: "unsubscribe reply", frame: `{"success":true,"ret_msg":"","conn_id":"3cd84cb1-4d06-4a05-930a-2efe5fc70f0f","req_id":"10002","op":"unsubscribe"}`, match: "10002", consumed: true},
		{name: "auth reply", frame: `{"success":true,"ret_msg":"","op":"auth","conn_id":"cejreaspqfh3sjdnldmg-p","req_id":"10003"}`, match: "10003", consumed: true},
		{name: "unmatched reply", frame: `{"success":true,"ret_msg":"","op":"auth","conn_id":"cejreaspqfh3sjdnldmg-p","req_id":"noMatch"}`, err: websocket.ErrSignatureNotMatched},
		{name: "reply without request ID", frame: `{"success":true,"ret_msg":"","op":"subscribe","conn_id":"cejreassvfrsfvb9v1a0-2m"}`},
		{name: "option subscribe reply", frame: `{"success":true,"conn_id":"aa01fbfffe80af37-00000001-000b37b9-7147f432704fd28c-00e1c172","data":{"failTopics":[],"successTopics":["orderbook.100.BTC-6JAN23-18000-C"]},"type":"COMMAND_RESP"}`, match: commandResponseType, consumed: true},
		{name: "unsolicited option reply", frame: `{"success":true,"conn_id":"aa01fbfffe80af37-00000001-000b37b9-7147f432704fd28c-00e1c172","data":{"failTopics":[],"successTopics":["orderbook.100.BTC-6JAN23-18000-C"]},"type":"COMMAND_RESP"}`},
		{name: "unsolicited option failure", frame: `{"success":true,"conn_id":"aa01fbfffe80af37-00000001-000b37b9-7147f432704fd28c-00e1c172","data":{"failTopics":["orderbook.100.BTC-6JAN23-18000-C"],"successTopics":[]},"type":"COMMAND_RESP"}`, err: websocket.ErrSubscriptionFailure},
		{name: "spot pong", frame: `{"success":true,"ret_msg":"pong","conn_id":"0970e817-426e-429a-a679-ff7f55e0b16a","op":"ping"}`},
		{name: "option pong", frame: `{"args":["1672916271846"],"op":"pong"}`},
		{name: "private pong", frame: `{"req_id":"test","op":"pong","args":["1675418560633"],"conn_id":"cfcb4ocsvfriu23r3er0-1b"}`},
		{name: "unknown", frame: `{"op":"welcome"}`, relayed: true},
		{name: "null", frame: `null`, relayed: true},
		{name: "undecodable", frame: `{"op":1}`, err: errors.New("op")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			conn := &FixtureConnection{match: websocket.NewMatch()}
			var ch <-chan []byte
			if tc.match != "" {
				var err error
				ch, err = conn.match.Set(tc.match, 1)
				require.NoError(t, err, "match.Set must not error")
			}
			err := ex.handleNoTopicWebsocketResponse(t.Context(), conn, []byte(tc.frame))
			switch {
			case tc.err == nil:
				require.NoError(t, err, "handleNoTopicWebsocketResponse must not error")
			case tc.err.Error() == "op":
				require.Error(t, err, "handleNoTopicWebsocketResponse must error for an undecodable frame")
			default:
				require.ErrorIs(t, err, tc.err, "handleNoTopicWebsocketResponse must return the expected error")
			}
			if tc.consumed {
				require.Len(t, ch, 1, "the reply must reach its request")
				assert.Equal(t, tc.frame, string(<-ch), "the reply should reach its request unchanged")
			}
			if tc.relayed {
				require.Len(t, ex.Websocket.DataHandler.C, 1, "an unknown frame must be relayed")
				payload := <-ex.Websocket.DataHandler.C
				assert.Equal(t, websocket.UnhandledMessageWarning{Message: tc.frame}, payload.Data, "an unknown frame should be relayed as a warning")
			} else {
				assert.Empty(t, ex.Websocket.DataHandler.C, "nothing should be relayed")
			}
		})
	}
}

func TestUniqueTopics(t *testing.T) {
	t.Parallel()
	subs := subscription.List{{QualifiedChannel: "publicTrade.BTC"}, {QualifiedChannel: "tickers.BTC-24SEP27-170000-P-USDT"}, {QualifiedChannel: "publicTrade.BTC"}}
	assert.Equal(t, []string{"publicTrade.BTC", "tickers.BTC-24SEP27-170000-P-USDT"}, uniqueTopics(subs), "uniqueTopics should list each topic once in order")
}

func TestShareTopics(t *testing.T) {
	t.Parallel()
	optionTrades := func(contract string) *subscription.Subscription {
		return &subscription.Subscription{Channel: chanPublicTrade, Asset: asset.Options, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", contract, currency.DashDelimiter)}, QualifiedChannel: "publicTrade.BTC"}
	}
	ex := websocketTestInstance(t)
	optionConn, spotConn, linearConn := &FixtureConnection{}, &FixtureConnection{}, &FixtureConnection{}
	put, call := optionTrades("24SEP27-170000-P-USDT"), optionTrades("24SEP27-170000-C-USDT")
	require.NoError(t, ex.recordSubscriptions(optionConn, subscribeOperation, subscription.List{put}), "recordSubscriptions must not error")
	require.NoError(t, ex.recordSubscriptions(spotConn, subscribeOperation, subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "tickers.BTCUSDT"}}), "recordSubscriptions must not error")

	// The spot and linear streams name their topics alike, and only a subscription on the same connection shares one,
	// whatever asset either names
	for _, linearTicker := range []*subscription.Subscription{
		{Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "tickers.BTCUSDT"},
		{Channel: chanPublicTicker, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "tickers.BTCUSDT"},
	} {
		pending, leaving, err := ex.shareTopics(linearConn, subscribeOperation, subscription.List{linearTicker})
		require.NoError(t, err, "shareTopics must not error")
		assert.Equalf(t, subscription.List{linearTicker}, pending, "shareTopics should leave a topic another connection holds to be sent for a %s subscription", linearTicker.Asset)
		assert.Empty(t, leaving.holders, "shareTopics should mark nothing leaving for a subscribe")
	}

	pending, _, err := ex.shareTopics(optionConn, subscribeOperation, subscription.List{call})
	require.NoError(t, err, "shareTopics must not error")
	assert.Empty(t, pending, "shareTopics should not send a topic another subscription on the connection holds")
	assert.Same(t, call, ex.Websocket.GetSubscription(call), "shareTopics should store a subscription to a topic another subscription holds")
	assert.Equal(t, subscription.SubscribedState, call.State(), "shareTopics should mark a subscription to a held topic subscribed")

	pending, leaving, err := ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{put})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	ex.keepHolders(leaving)
	assert.Empty(t, pending, "shareTopics should keep a topic another subscription still holds")
	assert.Empty(t, leaving.holders, "shareTopics should mark nothing leaving for an unsubscribe it records at once")
	assert.Nil(t, ex.Websocket.GetSubscription(put), "shareTopics should remove a subscription whose topic another still holds")
	assert.Equal(t, subscription.UnsubscribedState, put.State(), "shareTopics should mark that subscription unsubscribed")
	assert.Same(t, call, ex.Websocket.GetSubscription(call), "shareTopics should keep the subscription still holding the topic")

	_, _, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{put})
	assert.ErrorIs(t, err, subscription.ErrNotFound, "shareTopics should refuse to remove a subscription it does not hold")

	// An equal copy names the stored subscription, as the websocket manager's store does, and the holder it names is not
	// counted until its unsubscription is over
	copied := call.Clone()
	pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{copied})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	assert.Equal(t, subscription.List{copied}, pending, "shareTopics should leave the last subscription to a topic, named by an equal copy, to be unsubscribed")
	assert.Equal(t, subscription.List{call}, leaving.holders, "shareTopics should mark the stored subscription the copy names leaving")
	other := optionTrades("31DEC27-170000-P-USDT")
	pending, _, err = ex.shareTopics(optionConn, subscribeOperation, subscription.List{other})
	require.NoError(t, err, "shareTopics must not error")
	assert.Equal(t, subscription.List{other}, pending, "shareTopics should send a topic whose only holder is being unsubscribed")
	// The unsubscription was never sent, so the holder is still stored and counts again once it is over
	ex.keepHolders(leaving)
	pending, _, err = ex.shareTopics(optionConn, subscribeOperation, subscription.List{other})
	require.NoError(t, err, "shareTopics must not error")
	assert.Empty(t, pending, "shareTopics should count a holder whose unsubscription is over and which is still stored")

	// Subscriptions unsubscribed together leave their shared topic to be unsubscribed
	pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{other, call})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	assert.Equal(t, subscription.List{other, call}, pending, "shareTopics should leave a topic whose every subscription is unsubscribed to be unsubscribed")
	assert.ElementsMatch(t, subscription.List{other, call}, leaving.holders, "shareTopics should mark every subscription it unsubscribes leaving")
	require.NoError(t, ex.recordSubscriptions(optionConn, unsubscribeOperation, pending), "recordSubscriptions must not error")
	ex.keepHolders(leaving)

	// A subscription the manager no longer stores, as after a reconnect, which drops subscriptions without unsubscribing
	// them, holds no topic, even on the connection it was recorded on, and its record is dropped, as is that of one
	// recorded on a connection the reconnect replaced
	dropped, replaced := optionTrades("24SEP27-170000-C-USDT"), optionTrades("31DEC27-170000-C-USDT")
	oldConn := &FixtureConnection{}
	require.NoError(t, ex.recordSubscriptions(optionConn, subscribeOperation, subscription.List{dropped}), "recordSubscriptions must not error")
	require.NoError(t, ex.recordSubscriptions(oldConn, subscribeOperation, subscription.List{replaced}), "recordSubscriptions must not error")
	require.NoError(t, ex.Websocket.RemoveSubscriptions(optionConn, dropped), "RemoveSubscriptions must not error")
	require.NoError(t, ex.Websocket.RemoveSubscriptions(oldConn, replaced), "RemoveSubscriptions must not error")
	fresh := optionTrades("24SEP27-170000-P-USDT")
	pending, _, err = ex.shareTopics(optionConn, subscribeOperation, subscription.List{fresh})
	require.NoError(t, err, "shareTopics must not error")
	assert.Equal(t, subscription.List{fresh}, pending, "shareTopics should send a topic whose subscriptions the manager no longer stores")
	assert.NotContains(t, ex.topicHolders, dropped, "shareTopics should forget a subscription the manager no longer stores")
	assert.NotContains(t, ex.topicHolders, replaced, "shareTopics should forget a subscription the manager no longer stores on another connection")

	// A custom key names the stored subscription it matches, as the manager's store looks one up, so a subscription of
	// other pairs can name one keyed to ignore them; it takes one holder off the topic, and another keeps the topic
	broad := optionTrades("24SEP27-170000-P-USDT")
	broad.Key = subscription.IgnoringPairsKey{Subscription: broad}
	narrow := optionTrades("24SEP27-170000-C-USDT")
	narrow.Channel = chanPublicTicker
	narrow.Key = subscription.IgnoringPairsKey{Subscription: narrow}
	require.NoError(t, ex.recordSubscriptions(optionConn, subscribeOperation, subscription.List{broad, narrow}), "recordSubscriptions must not error")
	named := optionTrades("31DEC27-170000-C-USDT")
	named.Channel = chanPublicTicker
	named.Key = subscription.IgnoringPairsKey{Subscription: named}
	pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{named})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	ex.keepHolders(leaving)
	assert.Empty(t, pending, "shareTopics should keep a topic another custom keyed subscription holds")
	assert.Empty(t, leaving.holders, "shareTopics should mark nothing leaving for an unsubscribe it records at once")
	assert.Nil(t, ex.Websocket.GetSubscription(narrow), "shareTopics should remove the subscription the custom key names")
	assert.Same(t, broad, ex.Websocket.GetSubscription(broad), "shareTopics should keep the other custom keyed subscription")
	last := optionTrades("31DEC27-170000-P-USDT")
	last.Key = subscription.IgnoringPairsKey{Subscription: last}
	pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{last})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	ex.keepHolders(leaving)
	assert.Equal(t, subscription.List{last}, pending, "shareTopics should leave a topic whose last holder a custom key names to be unsubscribed")
	assert.Equal(t, subscription.List{broad}, leaving.holders, "shareTopics should mark the last holder the custom key names leaving")
	require.NoError(t, ex.recordSubscriptions(optionConn, unsubscribeOperation, pending), "recordSubscriptions must not error")

	// A key naming two holders leaves the manager to remove either, and the other still counts whichever it removes. The
	// removed holder's record no longer counts, though its key may still match the other's, so the topic is unsubscribed
	// once the other and a subscription sharing the topic go together
	for i := range 50 {
		wide := optionTrades("24SEP27-170000-C-USDT")
		wide.Key = subscription.IgnoringPairsKey{Subscription: wide}
		exact := optionTrades("24SEP27-170000-P-USDT")
		require.NoErrorf(t, ex.recordSubscriptions(optionConn, subscribeOperation, subscription.List{wide, exact}), "recordSubscriptions must not error, iteration %d", i)
		named := wide.Clone()
		named.Key = wide.Key
		pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, subscription.List{named})
		require.NoErrorf(t, err, "shareTopics must not error when unsubscribing, iteration %d", i)
		ex.keepHolders(leaving)
		require.Emptyf(t, pending, "shareTopics must keep a topic the other holder holds, iteration %d", i)
		sharer := optionTrades("31DEC27-170000-C-USDT")
		sharer.Channel = chanPublicTicker
		pending, _, err = ex.shareTopics(optionConn, subscribeOperation, subscription.List{sharer})
		require.NoErrorf(t, err, "shareTopics must not error, iteration %d", i)
		require.Emptyf(t, pending, "shareTopics must count the holder the manager kept, iteration %d", i)
		last := slices.DeleteFunc(ex.Websocket.GetSubscriptions(), func(s *subscription.Subscription) bool { return s.QualifiedChannel != "publicTrade.BTC" })
		require.Lenf(t, last, 2, "the manager must store the holder it kept and the sharer, iteration %d", i)
		pending, leaving, err = ex.shareTopics(optionConn, unsubscribeOperation, last)
		require.NoErrorf(t, err, "shareTopics must not error when unsubscribing, iteration %d", i)
		require.ElementsMatchf(t, last, pending, "shareTopics must unsubscribe a topic whose last holders go, iteration %d", i)
		require.NoErrorf(t, ex.recordSubscriptions(optionConn, unsubscribeOperation, pending), "recordSubscriptions must not error, iteration %d", i)
		ex.keepHolders(leaving)
		require.NoErrorf(t, ex.recordSubscriptions(optionConn, unsubscribeOperation, ex.Websocket.GetSubscriptions()), "recordSubscriptions must not error clearing iteration %d", i)
	}

	// Every orderbook depth of a symbol loads its one book, so a subscribe to another depth of a symbol the connection
	// holds is refused, while the same depth shares its topic and the RPI and full depth topics load no book
	book := func(base string, levels int, topic string) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(base, "USDT", "")}, Levels: levels, QualifiedChannel: topic}
	}
	bookConn := &FixtureConnection{}
	held := book("BTC", 50, "orderbook.50.BTCUSDT")
	require.NoError(t, ex.recordSubscriptions(bookConn, subscribeOperation, subscription.List{held}), "recordSubscriptions must not error")
	level1, level200 := book("BTC", 1, "orderbook.1.BTCUSDT"), book("BTC", 200, "orderbook.200.BTCUSDT")
	sharer, otherSymbol := book("BTC", 0, "orderbook.50.BTCUSDT"), book("ETH", 1, "orderbook.1.ETHUSDT")
	rpi := &subscription.Subscription{Channel: chanRPIOrderbook, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "orderbook.rpi.BTCUSDT"}
	full := &subscription.Subscription{Channel: chanFullOrderbook, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "orderbook.full.BTCUSDT"}
	pending, asked, err := ex.shareTopics(bookConn, subscribeOperation, subscription.List{level1, sharer, otherSymbol, rpi, full, level200})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse another depth of a symbol the connection holds")
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "shareTopics should report a refusal as a subscription failure")
	assert.ErrorContains(t, err, "orderbook.1.BTCUSDT conflicts with depth 50", "error should name each refused topic and the depth held")
	assert.ErrorContains(t, err, "orderbook.200.BTCUSDT conflicts with depth 50", "error should name each refused topic and the depth held")
	assert.Equal(t, subscription.List{otherSymbol, rpi, full}, pending, "shareTopics should send another symbol's depth and the topics that load no book")
	assert.Same(t, sharer, ex.Websocket.GetSubscription(sharer), "shareTopics should store a subscription sharing the topic of the depth held")
	for _, refused := range []*subscription.Subscription{level1, level200} {
		assert.Nilf(t, ex.Websocket.GetSubscription(refused), "shareTopics should not store refused %s", refused.QualifiedChannel)
		assert.Equalf(t, subscription.InactiveState, refused.State(), "shareTopics should leave refused %s inactive", refused.QualifiedChannel)
	}

	// A request asking for two depths of a symbol keeps the first, and the subscriptions sharing its topic
	solDefault, sol1, sol50 := book("SOL", 0, "orderbook.50.SOLUSDT"), book("SOL", 1, "orderbook.1.SOLUSDT"), book("SOL", 50, "orderbook.50.SOLUSDT")
	pending, asked, err = ex.shareTopics(bookConn, subscribeOperation, subscription.List{solDefault, sol1, sol50})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse a second depth of a symbol asked for in one request")
	assert.ErrorContains(t, err, "orderbook.1.SOLUSDT conflicts with depth 50", "error should name the refused topic and the depth asked for first")
	assert.Equal(t, subscription.List{solDefault, sol50}, pending, "shareTopics should send the first depth of a symbol asked for")

	// A depth whose holders are being unsubscribed still counts, since the unsubscribe can fail
	pending, leaving, err = ex.shareTopics(bookConn, unsubscribeOperation, subscription.List{held, sharer})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	require.Equal(t, subscription.List{held, sharer}, pending, "shareTopics must leave a depth whose every holder goes to be unsubscribed")
	pending, asked, err = ex.shareTopics(bookConn, subscribeOperation, subscription.List{level1})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse a depth while the holders of another are being unsubscribed")
	assert.Empty(t, pending, "shareTopics should not send a depth while the holders of another are being unsubscribed")
	ex.keepHolders(leaving)
	pending, asked, err = ex.shareTopics(bookConn, subscribeOperation, subscription.List{level1})
	ex.keepHolders(asked)
	assert.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics should refuse another depth once a failed unsubscribe leaves the depth held")
	assert.Empty(t, pending, "shareTopics should send nothing for a refused depth")

	// Two depths held at once, as concurrent requests can leave them, refuse a subscribe to either, but not an
	// unsubscribe from one
	require.NoError(t, ex.recordSubscriptions(bookConn, subscribeOperation, subscription.List{level1}), "recordSubscriptions must not error")
	for _, s := range []*subscription.Subscription{
		{Channel: subscription.OrderbookChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Levels: 1, QualifiedChannel: "orderbook.1.BTCUSDT"},
		{Channel: subscription.OrderbookChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Levels: 50, QualifiedChannel: "orderbook.50.BTCUSDT"},
	} {
		pending, asked, err = ex.shareTopics(bookConn, subscribeOperation, subscription.List{s})
		ex.keepHolders(asked)
		assert.ErrorIsf(t, err, errOrderbookDepthConflict, "shareTopics should refuse %s while the connection holds two depths of its symbol", s.QualifiedChannel)
		assert.Emptyf(t, pending, "shareTopics should not send %s while the connection holds two depths of its symbol", s.QualifiedChannel)
	}
	pending, leaving, err = ex.shareTopics(bookConn, unsubscribeOperation, subscription.List{level1})
	ex.keepHolders(leaving)
	require.NoError(t, err, "shareTopics must not refuse an unsubscribe from a depth")
	assert.Equal(t, subscription.List{level1}, pending, "shareTopics should leave the last holder of a depth to be unsubscribed")

	// Every connection to a stream loads the same books, so a depth another connection to the stream holds is refused,
	// while another stream's books are its own
	spotConn, sibling, linearConn := &FixtureConnection{url: "wss://spot"}, &FixtureConnection{url: "wss://spot"}, &FixtureConnection{url: "wss://linear"}
	require.NoError(t, ex.recordSubscriptions(spotConn, subscribeOperation, subscription.List{book("ADA", 50, "orderbook.50.ADAUSDT")}), "recordSubscriptions must not error")
	pending, asked, err = ex.shareTopics(sibling, subscribeOperation, subscription.List{book("ADA", 1, "orderbook.1.ADAUSDT")})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse a depth another connection to the stream holds")
	assert.Empty(t, pending, "shareTopics should not send a depth another connection to the stream holds another of")
	linearADA := book("ADA", 1, "orderbook.1.ADAUSDT")
	pending, asked, err = ex.shareTopics(linearConn, subscribeOperation, subscription.List{linearADA})
	ex.keepHolders(asked)
	require.NoError(t, err, "shareTopics must not refuse a depth of another stream's book")
	assert.Equal(t, subscription.List{linearADA}, pending, "shareTopics should send a depth of another stream's book")

	// A depth a request in flight asks for is held until the request is over, and then only by what it subscribed
	dot50, dot1 := book("DOT", 50, "orderbook.50.DOTUSDT"), book("DOT", 1, "orderbook.1.DOTUSDT")
	pending, inFlight, err := ex.shareTopics(spotConn, subscribeOperation, subscription.List{dot50})
	require.NoError(t, err, "shareTopics must not error")
	require.Equal(t, subscription.List{dot50}, pending, "shareTopics must send a depth no one holds")
	pending, asked, err = ex.shareTopics(sibling, subscribeOperation, subscription.List{dot1})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse a depth while a request in flight on the stream asks for another")
	assert.Empty(t, pending, "shareTopics should not send a depth while a request in flight on the stream asks for another")
	ex.keepHolders(inFlight)
	assert.Empty(t, ex.bookDepths, "keepHolders should release the depths a request asked for once it is over")
	pending, asked, err = ex.shareTopics(sibling, subscribeOperation, subscription.List{dot1})
	require.NoError(t, err, "shareTopics must not refuse a depth once a request for another fails")
	require.Equal(t, subscription.List{dot1}, pending, "shareTopics must send a depth once a request for another fails")
	require.NoError(t, ex.recordSubscriptions(sibling, subscribeOperation, pending), "recordSubscriptions must not error")
	ex.keepHolders(asked)
	pending, asked, err = ex.shareTopics(spotConn, subscribeOperation, subscription.List{dot50})
	ex.keepHolders(asked)
	assert.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics should refuse another depth once a request in flight subscribes its own")
	assert.Empty(t, pending, "shareTopics should not send another depth once a request in flight subscribes its own")

	// Two requests in flight asking for one depth hold it until both are over
	pending, first, err := ex.shareTopics(spotConn, subscribeOperation, subscription.List{book("XRP", 50, "orderbook.50.XRPUSDT")})
	require.NoError(t, err, "shareTopics must not error")
	require.Len(t, pending, 1, "shareTopics must send a depth no one holds")
	pending, second, err := ex.shareTopics(sibling, subscribeOperation, subscription.List{book("XRP", 50, "orderbook.50.XRPUSDT")})
	require.NoError(t, err, "shareTopics must not refuse the depth a request in flight asks for")
	require.Len(t, pending, 1, "shareTopics must send the depth a request in flight asks for")
	ex.keepHolders(first)
	pending, asked, err = ex.shareTopics(sibling, subscribeOperation, subscription.List{book("XRP", 1, "orderbook.1.XRPUSDT")})
	ex.keepHolders(asked)
	require.ErrorIs(t, err, errOrderbookDepthConflict, "shareTopics must refuse a depth while another request in flight still asks for another")
	assert.Empty(t, pending, "shareTopics should not send a depth while another request in flight still asks for another")
	ex.keepHolders(second)
	assert.Empty(t, ex.bookDepths, "keepHolders should release a depth once every request asking for it is over")
}

func TestBookDepthsLocked(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	spotConn, sibling, linearConn := &FixtureConnection{url: "wss://spot"}, &FixtureConnection{url: "wss://spot"}, &FixtureConnection{url: "wss://linear"}
	topic := func(qualified string) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: qualified}
	}
	ex.topicHolders = map[*subscription.Subscription]topicHolder{
		topic("orderbook.50.BTCUSDT"):  {conn: spotConn},
		topic("orderbook.1.BTCUSDT"):   {conn: sibling},
		topic("orderbook.50.BTCUSDT"):  {conn: sibling},
		topic("orderbook.200.SOLUSDT"): {conn: spotConn, leaving: []uint64{1}},
		topic("orderbook.50.ETHUSDT"):  {conn: linearConn},
		topic("tickers.ADAUSDT"):       {conn: spotConn},
		topic("orderbook.rpi.DOTUSDT"): {conn: spotConn},
	}
	ex.bookDepths = map[bookDepth]int{
		{stream: "wss://spot", symbol: "XRPUSDT", depth: "25"}:  1,
		{stream: "wss://spot", symbol: "BTCUSDT", depth: "50"}:  1,
		{stream: "wss://linear", symbol: "ADAUSDT", depth: "1"}: 2,
	}
	got := ex.bookDepthsLocked("wss://spot")
	for _, depths := range got {
		slices.Sort(depths)
	}
	exp := map[string][]string{"BTCUSDT": {"1", "50"}, "SOLUSDT": {"200"}, "XRPUSDT": {"25"}}
	assert.Equal(t, exp, got, "bookDepthsLocked should return each depth the stream's connections hold or its requests in flight ask for, once")
}

func TestOrderbookDepthTopic(t *testing.T) {
	t.Parallel()
	type parsed struct {
		depth, symbol string
		ok            bool
	}
	for topic, exp := range map[string]parsed{
		"orderbook.50.BTCUSDT":                   {depth: "50", symbol: "BTCUSDT", ok: true},
		"orderbook.1.BTCUSDT":                    {depth: "1", symbol: "BTCUSDT", ok: true},
		"orderbook.25.BTC-24SEP27-170000-P-USDT": {depth: "25", symbol: "BTC-24SEP27-170000-P-USDT", ok: true},
		"orderbook.50.BTC.USDT":                  {depth: "50", symbol: "BTC.USDT", ok: true},
		"orderbook.rpi.BTCUSDT":                  {},
		"orderbook.full.BTCUSDT":                 {},
		"orderbook.-50.BTCUSDT":                  {},
		"orderbook..BTCUSDT":                     {},
		"orderbook.50.":                          {},
		"orderbook.50":                           {},
		"orderbook":                              {},
		"kline.1.BTCUSDT":                        {},
		"tickers.BTCUSDT":                        {},
		"":                                       {},
	} {
		depth, symbol, ok := orderbookDepthTopic(topic)
		assert.Equalf(t, exp, parsed{depth: depth, symbol: symbol, ok: ok}, "orderbookDepthTopic should parse %q", topic)
	}
}

// TestTopicHolderMarks checks that each request clears only its own leaving marks, and all of them
func TestTopicHolderMarks(t *testing.T) {
	t.Parallel()
	ticker := func(channel, base string) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(base, "USDT", "")}, QualifiedChannel: "tickers." + base + "USDT"}
	}
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{}
	held := ticker(subscription.TickerChannel, "BTC")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{held}), "recordSubscriptions must not error")

	// Two requests unsubscribing one holder each mark it, so the first ending leaves it leaving for the second
	_, first, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{held})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	_, second, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{held.Clone()})
	require.NoError(t, err, "shareTopics must not error when unsubscribing again")
	assert.Equal(t, subscription.List{held}, second.holders, "shareTopics should mark a holder another request is unsubscribing")
	ex.keepHolders(first)
	sharer := ticker(chanPublicTicker, "BTC")
	pending, _, err := ex.shareTopics(conn, subscribeOperation, subscription.List{sharer})
	require.NoError(t, err, "shareTopics must not error")
	assert.Equal(t, subscription.List{sharer}, pending, "shareTopics should not count a holder another request is still unsubscribing")
	ex.keepHolders(second)
	pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{sharer})
	require.NoError(t, err, "shareTopics must not error")
	assert.Empty(t, pending, "shareTopics should count a holder once no request is unsubscribing it")
	require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, subscription.List{sharer}), "recordSubscriptions must not error")

	// A request clears every mark it set, and a request's late clearing leaves a later request's mark
	other := ticker(subscription.TickerChannel, "ETH")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{other}), "recordSubscriptions must not error")
	_, both, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{held, other})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	assert.ElementsMatch(t, subscription.List{held, other}, both.holders, "shareTopics should mark both holders it unsubscribes")
	ex.keepHolders(both)
	for _, sub := range []*subscription.Subscription{ticker(chanPublicTicker, "BTC"), ticker(chanPublicTicker, "ETH")} {
		pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{sub})
		require.NoError(t, err, "shareTopics must not error")
		assert.Emptyf(t, pending, "shareTopics should count %s's holder once its request clears its marks", sub.QualifiedChannel)
		require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, subscription.List{sub}), "recordSubscriptions must not error")
	}
	_, earlier, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{other})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, subscription.List{other}), "recordSubscriptions must not error")
	assert.NotContains(t, ex.topicHolders, other, "recordSubscriptions should forget a holder unsubscribed from its connection")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{other}), "recordSubscriptions must not error for the subscription again")
	_, later, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{other})
	require.NoError(t, err, "shareTopics must not error when unsubscribing again")
	ex.keepHolders(earlier)
	fresh := ticker(chanPublicTicker, "ETH")
	pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{fresh})
	require.NoError(t, err, "shareTopics must not error")
	assert.Equal(t, subscription.List{fresh}, pending, "shareTopics should keep the mark of a later request when an earlier one clears its own")
	ex.keepHolders(later)

	// Three requests unsubscribing one holder each mark it, and their marks clear in any order
	triple := ticker(subscription.TickerChannel, "DOT")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{triple}), "recordSubscriptions must not error")
	_, one, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{triple})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	_, two, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{triple.Clone()})
	require.NoError(t, err, "shareTopics must not error when unsubscribing again")
	_, three, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{triple.Clone()})
	require.NoError(t, err, "shareTopics must not error when unsubscribing a third time")
	assert.Equal(t, subscription.List{triple}, three.holders, "shareTopics should mark a holder two other requests are unsubscribing")
	ex.keepHolders(two)
	ex.keepHolders(one)
	dotSharer := ticker(chanPublicTicker, "DOT")
	pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{dotSharer})
	require.NoError(t, err, "shareTopics must not error")
	assert.Equal(t, subscription.List{dotSharer}, pending, "shareTopics should not count a holder whose third unsubscribing request is still in flight")
	ex.keepHolders(three)
	pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{dotSharer})
	require.NoError(t, err, "shareTopics must not error")
	assert.Empty(t, pending, "shareTopics should count a holder once every request unsubscribing it has cleared its mark")
	require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, subscription.List{dotSharer}), "recordSubscriptions must not error")

	// A holder being resubscribed is still stored, and holds its topic
	resubscribing := ticker(subscription.TickerChannel, "AVAX")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{resubscribing}), "recordSubscriptions must not error")
	require.NoError(t, resubscribing.SetState(subscription.ResubscribingState), "SetState must not error")
	pending, _, err = ex.shareTopics(conn, subscribeOperation, subscription.List{ticker(chanPublicTicker, "AVAX")})
	require.NoError(t, err, "shareTopics must not error")
	assert.Empty(t, pending, "shareTopics should count a holder being resubscribed")

	// The record follows a subscription to the connection it was recorded on last
	moved := ticker(subscription.TickerChannel, "LTC")
	latest := &FixtureConnection{}
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{moved}), "recordSubscriptions must not error")
	assert.ErrorIs(t, ex.recordSubscriptions(latest, subscribeOperation, subscription.List{moved}), subscription.ErrDuplicate, "recordSubscriptions should report the subscription the manager already stores")
	ex.topicMu.Lock()
	recorded := ex.topicHolders[moved].conn
	ex.topicMu.Unlock()
	assert.Same(t, latest, recorded, "recordSubscriptions should record the connection a subscription was recorded on last")

	// A request naming a holder another request is unsubscribing does not count it again, so the topic stays for a
	// holder subscribed meanwhile
	leavingHolder := ticker(subscription.TickerChannel, "ADA")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{leavingHolder}), "recordSubscriptions must not error")
	_, inFlight, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{leavingHolder})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{ticker(chanPublicTicker, "ADA")}), "recordSubscriptions must not error")
	pending, again, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{leavingHolder.Clone()})
	require.NoError(t, err, "shareTopics must not error when unsubscribing again")
	ex.keepHolders(again)
	ex.keepHolders(inFlight)
	assert.Empty(t, pending, "shareTopics should keep a topic a holder subscribed meanwhile holds")

	// A scalar key names the holder of an equal key
	scalar := ticker(subscription.TickerChannel, "SOL")
	scalar.Key = "sol ticker"
	require.NoError(t, ex.recordSubscriptions(conn, subscribeOperation, subscription.List{scalar}), "recordSubscriptions must not error")
	copied := ticker(subscription.TickerChannel, "SOL")
	copied.Key = "sol ticker"
	pending, marks, err := ex.shareTopics(conn, unsubscribeOperation, subscription.List{copied})
	require.NoError(t, err, "shareTopics must not error when unsubscribing")
	ex.keepHolders(marks)
	assert.Equal(t, subscription.List{copied}, pending, "shareTopics should leave the last holder an equal scalar key names to be unsubscribed")
	assert.Equal(t, subscription.List{scalar}, marks.holders, "shareTopics should mark the holder an equal scalar key names")
	require.NoError(t, ex.recordSubscriptions(conn, unsubscribeOperation, pending), "recordSubscriptions must not error")
	assert.NotContains(t, ex.topicHolders, scalar, "recordSubscriptions should forget the holder an equal scalar key unsubscribed")
}

// TestShareTopicsWhileUnsubscribing holds the reply to a topic's last unsubscribe, and checks that a subscribe to the
// topic meanwhile is sent, as the topic is being unsubscribed, that a request on another connection does not wait for
// the reply, and that a holder whose unsubscribe fails holds its topic again
func TestShareTopicsWhileUnsubscribing(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	ticker := func(channel, base string) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(base, "USDT", "")}, QualifiedChannel: "tickers." + base + "USDT"}
	}
	unsubscribing := make(chan struct{}, 1)
	release := make(chan struct{})
	held := &FixtureConnection{reply: func(r any) ([]byte, error) {
		req, ok := r.(*SubscriptionRequest)
		if !ok {
			return nil, errUnexpectedRequest
		}
		if req.Operation == unsubscribeOperation {
			unsubscribing <- struct{}{}
			<-release
		}
		return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
	}}
	first := ticker(subscription.TickerChannel, "BTC")
	require.NoError(t, ex.recordSubscriptions(held, subscribeOperation, subscription.List{first}), "recordSubscriptions must not error")
	errs := make(chan error, 1)
	go func() {
		errs <- ex.sendSubscriptions(t.Context(), held, unsubscribeOperation, subscription.List{first})
	}()
	select {
	case <-unsubscribing:
	case <-time.After(5 * time.Second):
		close(release)
		require.FailNow(t, "the unsubscribe must reach the connection")
	}
	other := &FixtureConnection{reply: func(r any) ([]byte, error) {
		req, ok := r.(*SubscriptionRequest)
		if !ok {
			return nil, errUnexpectedRequest
		}
		return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
	}}
	// A request that waited for the held reply would block until go test's timeout, so each is given five seconds
	unheld := func(send func() error) error {
		done := make(chan error, 1)
		go func() { done <- send() }()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			close(release)
			require.FailNow(t, "a request must not wait for the reply to another request")
			return nil
		}
	}
	assert.NoError(t, unheld(func() error {
		return ex.sendSubscriptions(t.Context(), other, subscribeOperation, subscription.List{ticker(subscription.TickerChannel, "ETH")})
	}), "a request on another connection should not wait for the reply")
	second := ticker(chanPublicTicker, "BTC")
	assert.NoError(t, unheld(func() error {
		return ex.sendSubscriptions(t.Context(), held, subscribeOperation, subscription.List{second})
	}), "a subscribe to a topic being unsubscribed should not error")
	close(release)
	select {
	case err := <-errs:
		require.NoError(t, err, "the unsubscribe must not error once answered")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the unsubscribe must be answered")
	}
	operations := make([]string, 0, len(held.sent))
	for _, r := range held.sent {
		req, ok := r.(*SubscriptionRequest)
		require.True(t, ok, "sendSubscriptions must send SubscriptionRequests")
		operations = append(operations, req.Operation+" "+strings.Join(req.Arguments, ","))
	}
	assert.Equal(t, []string{"unsubscribe tickers.BTCUSDT", "subscribe tickers.BTCUSDT"}, operations, "a subscribe to a topic being unsubscribed should be sent")

	// A holder whose unsubscribe fails is still stored, and holds its topic for the next subscription to it
	failing := &FixtureConnection{reply: func(r any) ([]byte, error) {
		req, ok := r.(*SubscriptionRequest)
		if !ok {
			return nil, errUnexpectedRequest
		}
		if req.Operation == unsubscribeOperation {
			return nil, errFixtureSend
		}
		return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
	}}
	kept := ticker(subscription.TickerChannel, "SOL")
	require.NoError(t, ex.recordSubscriptions(failing, subscribeOperation, subscription.List{kept}), "recordSubscriptions must not error")
	require.ErrorIs(t, ex.sendSubscriptions(t.Context(), failing, unsubscribeOperation, subscription.List{kept}), errFixtureSend, "sendSubscriptions must return a failed unsubscribe")
	require.NoError(t, ex.sendSubscriptions(t.Context(), failing, subscribeOperation, subscription.List{ticker(chanPublicTicker, "SOL")}), "sendSubscriptions must not error")
	assert.Len(t, failing.sent, 1, "a subscribe to a topic a holder still holds after a failed unsubscribe should not be sent")
	// The option and system status streams' sender does the same
	option := &subscription.Subscription{Channel: chanPublicTrade, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}, QualifiedChannel: "publicTrade.BTC"}
	require.NoError(t, ex.recordSubscriptions(failing, subscribeOperation, subscription.List{option}), "recordSubscriptions must not error")
	require.ErrorIs(t, ex.sendCommandSubscriptions(t.Context(), failing, unsubscribeOperation, subscription.List{option}), errFixtureSend, "sendCommandSubscriptions must return a failed unsubscribe")
	optionSharer := &subscription.Subscription{Channel: chanPublicTrade, Asset: asset.Options, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "24SEP27-170000-C-USDT", currency.DashDelimiter)}, QualifiedChannel: "publicTrade.BTC"}
	require.NoError(t, ex.sendCommandSubscriptions(t.Context(), failing, subscribeOperation, subscription.List{optionSharer}), "sendCommandSubscriptions must not error")
	assert.Len(t, failing.sent, 2, "a subscribe to an option topic a holder still holds after a failed unsubscribe should not be sent")

	// A subscription recorded at once that cannot be removed does not stop the rest of the request
	unstored := ticker(chanPublicTicker, "SOL")
	unstored.Channel = "orderbook"
	last := ticker(subscription.TickerChannel, "XRP")
	require.NoError(t, ex.recordSubscriptions(failing, subscribeOperation, subscription.List{last}), "recordSubscriptions must not error")
	sentBefore := len(failing.sent)
	err := ex.sendSubscriptions(t.Context(), failing, unsubscribeOperation, subscription.List{unstored, last})
	require.ErrorIs(t, err, subscription.ErrNotFound, "sendSubscriptions must report a subscription it does not hold")
	require.Len(t, failing.sent, sentBefore+1, "sendSubscriptions must still send the rest of the request")
	req, ok := failing.sent[sentBefore].(*SubscriptionRequest)
	require.True(t, ok, "sendSubscriptions must send a SubscriptionRequest")
	assert.Equal(t, []string{"tickers.XRPUSDT"}, req.Arguments, "sendSubscriptions should send the topic whose last holder it unsubscribes")
}

// TestSharedTopicsOnEveryStream shares a topic on the linear and private streams, which sendSubscriptions also serves
func TestSharedTopicsOnEveryStream(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		subs        subscription.List
		subscribe   func(*Exchange) func(context.Context, websocket.Connection, subscription.List) error
		unsubscribe func(*Exchange) func(context.Context, websocket.Connection, subscription.List) error
	}{
		{
			name: "linear",
			subs: subscription.List{
				{Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: chanPublicTicker, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
			},
			subscribe: func(ex *Exchange) func(context.Context, websocket.Connection, subscription.List) error {
				return func(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
					return ex.LinearSubscribe(ctx, conn, asset.USDTMarginedFutures, subs)
				}
			},
			unsubscribe: func(ex *Exchange) func(context.Context, websocket.Connection, subscription.List) error {
				return func(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
					return ex.LinearUnsubscribe(ctx, conn, asset.USDTMarginedFutures, subs)
				}
			},
		},
		{
			// Private topics take no symbol, so subscriptions of different assets can share one
			name: "private",
			subs: subscription.List{
				{Channel: chanOrder, Asset: asset.Spot, Authenticated: true},
				{Channel: chanOrder, Asset: asset.USDTMarginedFutures, Authenticated: true},
			},
			subscribe: func(ex *Exchange) func(context.Context, websocket.Connection, subscription.List) error {
				return ex.authSubscribe
			},
			unsubscribe: func(ex *Exchange) func(context.Context, websocket.Connection, subscription.List) error {
				return ex.authUnsubscribe
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			conn := &FixtureConnection{reply: func(r any) ([]byte, error) {
				req, ok := r.(*SubscriptionRequest)
				if !ok {
					return nil, errUnexpectedRequest
				}
				return json.Marshal(&SubscriptionResponse{Success: true, RequestID: req.RequestID, Operation: req.Operation})
			}}
			subscribe, unsubscribe := tc.subscribe(ex), tc.unsubscribe(ex)
			for _, s := range tc.subs {
				require.NoError(t, subscribe(t.Context(), conn, subscription.List{s}), "subscribing must not error")
			}
			require.Len(t, conn.sent, 1, "a topic subscriptions share must be subscribed once")
			require.NoError(t, unsubscribe(t.Context(), conn, tc.subs[:1]), "unsubscribing must not error")
			assert.Len(t, conn.sent, 1, "a topic another subscription holds should not be unsubscribed")
			require.NoError(t, unsubscribe(t.Context(), conn, tc.subs[1:]), "unsubscribing the last subscription must not error")
			require.Len(t, conn.sent, 2, "a topic's last subscription must unsubscribe it")
			req, ok := conn.sent[1].(*SubscriptionRequest)
			require.True(t, ok, "unsubscribing must send a SubscriptionRequest")
			assert.Equal(t, unsubscribeOperation, req.Operation, "the last request should unsubscribe")
			assert.Empty(t, ex.Websocket.GetSubscriptions(), "every subscription should be removed")
		})
	}
}

func TestTopicBatches(t *testing.T) {
	t.Parallel()
	a1 := &subscription.Subscription{Channel: "first", QualifiedChannel: "a"}
	b := &subscription.Subscription{Channel: "second", QualifiedChannel: "b"}
	a2 := &subscription.Subscription{Channel: "third", QualifiedChannel: "a"}
	c := &subscription.Subscription{Channel: "fourth", QualifiedChannel: "c"}
	d := &subscription.Subscription{Channel: "fifth", QualifiedChannel: "d"}
	subs := subscription.List{a1, b, a2, c, d}
	assert.Equal(t, []subscription.List{{a1, a2, b}, {c, d}}, topicBatches(subs, 2), "topicBatches should send at most size topics a batch, keeping the subscriptions that share one together")
	assert.Equal(t, []subscription.List{{a1, a2, b, c, d}}, topicBatches(subs, 0), "topicBatches should send every topic in one batch without a size")
	assert.Empty(t, topicBatches(nil, 2), "topicBatches should return no batches for no subscriptions")
}

func TestGenerateSpotSubscriptions(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, true), "StorePairs must not error")
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 200},
		{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.FiveMin},
		{Enabled: true, Asset: asset.Spot, Channel: chanRPIOrderbook},
		{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: chanInsurancePool},
		{Enabled: true, Channel: chanSystemStatus},
		// Channels with connections of their own are left to those, whatever asset they are configured for
		{Enabled: true, Asset: asset.Spot, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": "SOLUSDT_SOL/USDT"}},
		{Enabled: true, Asset: asset.All, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": "BTCUSDT-PRED"}},
		{Enabled: true, Asset: asset.Spot, Channel: ChannelRFQPublicTrades},
		{Enabled: true, Asset: asset.All, Channel: AdvancedEarnDualAssetsOffersChannel},
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	for _, s := range subs {
		assert.Equal(t, asset.Spot, s.Asset, "the spot connection should only generate spot subscriptions")
	}
	assert.ElementsMatch(t, []string{"orderbook.200.BTCUSDT", "kline.5.BTCUSDT", "orderbook.rpi.BTCUSDT"}, subs.QualifiedChannels(), "generateSubscriptions should expand the spot topics")

	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel}}
	_, err = ex.generateSubscriptions()
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generateSubscriptions must reject a subscription without a topic")
}

func TestDedicatedConnectionChannel(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{
		ChannelSpreadOrderbook, ChannelSpreadPublicTrade, ChannelSpreadTicker, ChannelSpreadOrder, ChannelSpreadExecution,
		ChannelEventOrderbook, ChannelEventPublicTrade, ChannelEventPosition, ChannelEventExecution, ChannelEventOrder,
		ChannelRFQPublicTrades, ChannelRFQQuotes, ChannelRFQTrades, ChannelRFQInquiries,
		AdvancedEarnDualAssetsOffersChannel, AdvancedEarnSmartLeverageOffersChannel, AdvancedEarnDiscountBuyOffersChannel,
		AdvancedEarnDoubleWinOffersChannel, chanSystemStatus,
	} {
		assert.Truef(t, dedicatedConnectionChannel(channel), "dedicatedConnectionChannel should report %s as subscribed on a connection of its own", channel)
	}
	for _, channel := range []string{chanOrderbook, chanRPIOrderbook, chanPublicTrade, chanPublicTicker, chanKline, subscription.TickerChannel} {
		assert.Falsef(t, dedicatedConnectionChannel(channel), "dedicatedConnectionChannel should leave %s to the spot connection", channel)
	}
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tpl, err := e.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	require.NotNil(t, tpl, "GetSubscriptionTemplate must return a template")
	// The context mirrors the one subscription.List.ExpandTemplates executes the template with
	templateContext := func(s *subscription.Subscription) map[string]any {
		return map[string]any{
			"S":              s,
			"AssetPairs":     map[asset.Item]currency.Pairs{asset.Spot: {currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}},
			"PairSeparator":  "|",
			"AssetSeparator": ";",
		}
	}
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
	}{
		{sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "tickers.BTCUSDT|tickers.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 200}, exp: "orderbook.200.BTCUSDT|orderbook.200.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "orderbook.50.BTCUSDT|orderbook.50.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: chanRPIOrderbook}, exp: "orderbook.rpi.BTCUSDT|orderbook.rpi.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "publicTrade.BTCUSDT|publicTrade.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneHour}, exp: "kline.60.BTCUSDT|kline.60.ETHUSDT|;"},
		{sub: &subscription.Subscription{Channel: subscription.MyWalletChannel}, exp: ";"},
	} {
		var buf bytes.Buffer
		require.NoErrorf(t, tpl.Execute(&buf, templateContext(tc.sub)), "Execute must not error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.exp, strings.TrimSpace(buf.String()), "GetSubscriptionTemplate should expand %s into one topic per pair", tc.sub.Channel)
	}
	err = tpl.Execute(new(bytes.Buffer), templateContext(&subscription.Subscription{Channel: subscription.CandlesChannel}))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetSubscriptionTemplate must refuse a candle subscription without an interval")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		exp     string
	}{
		{channel: subscription.TickerChannel, exp: "tickers"},
		{channel: subscription.OrderbookChannel, exp: "orderbook"},
		{channel: subscription.AllTradesChannel, exp: "publicTrade"},
		{channel: subscription.CandlesChannel, exp: "kline"},
		{channel: subscription.MyOrdersChannel, exp: "order"},
		{channel: subscription.MyWalletChannel, exp: "wallet"},
		{channel: subscription.MyTradesChannel, exp: "execution"},
		{channel: chanRPIOrderbook, exp: "orderbook.rpi"},
		{channel: "dcp.future", exp: "dcp.future"},
	} {
		assert.Equalf(t, tc.exp, channelName(&subscription.Subscription{Channel: tc.channel}), "channelName should name %s by its Bybit topic", tc.channel)
	}
}

func TestIsSymbolChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		exp  bool
	}{
		{name: chanPublicTicker, exp: true},
		{name: chanOrderbook, exp: true},
		{name: chanRPIOrderbook, exp: true},
		{name: chanFullOrderbook, exp: true},
		{name: chanPublicTrade, exp: true},
		{name: chanKline, exp: true},
		{name: chanAllLiquidation, exp: true},
		{name: chanPriceLimit, exp: true},
		{name: chanPositions},
		{name: chanExecution},
		{name: chanOrder},
		{name: chanWallet},
		{name: chanInsurancePool},
		{name: chanADLAlert},
		{name: chanSystemStatus},
	} {
		assert.Equalf(t, tc.exp, isSymbolChannel(tc.name), "isSymbolChannel should report whether a %s topic names a symbol", tc.name)
	}
}

func TestSymbol(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		pair  currency.Pair
		asset asset.Item
		exp   string
	}{
		{pair: currency.NewBTCUSDT(), asset: asset.Spot, exp: "BTCUSDT"},
		{pair: currency.NewPair(currency.BTC, currency.NewCode("USDT-16OCT26")), asset: asset.USDTMarginedFutures, exp: "BTCUSDT-16OCT26"},
		{pair: currency.NewPair(currency.BTC, currency.PERP), asset: asset.USDCMarginedFutures, exp: "BTCPERP"},
		{pair: currency.NewPair(currency.BTC, currency.NewCode("26DEC25")), asset: asset.USDCMarginedFutures, exp: "BTC-26DEC25"},
		{pair: currency.NewBTCUSD(), asset: asset.CoinMarginedFutures, exp: "BTCUSD"},
		{pair: currency.NewPair(currency.BTC, currency.NewCode("24SEP27-170000-P-USDT")), asset: asset.Options, exp: "BTC-24SEP27-170000-P-USDT"},
	} {
		symbol, err := e.symbol(tc.pair, tc.asset)
		require.NoErrorf(t, err, "symbol must not error for %s", tc.asset)
		assert.Equalf(t, tc.exp, symbol, "symbol should name %s as Bybit does", tc.asset)
	}
	_, err := e.symbol(currency.NewBTCUSDT(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "symbol must error for an unsupported asset")
}

func TestMatchPairAssetFromResponse(t *testing.T) {
	t.Parallel()
	ensureTradablePairs(t)

	noDelim := currency.PairFormat{Uppercase: true}
	for _, tc := range []struct {
		pair          string
		category      string
		expectedAsset asset.Item
		expectedPair  currency.Pair
		err           error
	}{
		{pair: noDelim.Format(spotTradablePair), category: cSpot, expectedAsset: asset.Spot, expectedPair: spotTradablePair},
		{pair: noDelim.Format(usdtMarginedTradablePair), category: cLinear, expectedAsset: asset.USDTMarginedFutures, expectedPair: usdtMarginedTradablePair},
		{pair: noDelim.Format(usdcMarginedTradablePair), category: cLinear, expectedAsset: asset.USDCMarginedFutures, expectedPair: usdcMarginedTradablePair},
		{pair: noDelim.Format(inverseTradablePair), category: cInverse, expectedAsset: asset.CoinMarginedFutures, expectedPair: inverseTradablePair},
		{pair: optionsTradablePair.String(), category: cOption, expectedAsset: asset.Options, expectedPair: optionsTradablePair},
		{pair: optionsTradablePair.String(), category: "silly", err: errUnsupportedCategory, expectedAsset: 0},
		{pair: "bad pair", category: cSpot, err: currency.ErrPairNotFound},
		{pair: "", category: cSpot, err: currency.ErrSymbolStringEmpty},
	} {
		t.Run(fmt.Sprintf("pair: %s, category: %s", tc.pair, tc.category), func(t *testing.T) {
			t.Parallel()
			p, a, err := e.matchPairAssetFromResponse(tc.category, tc.pair)
			require.ErrorIs(t, err, tc.err, "matchPairAssetFromResponse must return the expected error")
			assert.Equal(t, tc.expectedAsset, a, "matchPairAssetFromResponse should return the expected asset")
			assert.True(t, tc.expectedPair.Equal(p), "matchPairAssetFromResponse should return the expected pair")
		})
	}
}

func TestHasPotentialDelimiter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		exp   bool
	}{
		{asset: asset.Options, exp: true},             // BTC-24SEP27-170000-P-USDT
		{asset: asset.USDCMarginedFutures, exp: true}, // BTC-26DEC25
		{asset: asset.Spot},                           // BTCUSDT
		{asset: asset.USDTMarginedFutures},            // BTCUSDT-16OCT26, whose dash belongs to the quote
		{asset: asset.CoinMarginedFutures},            // BTCUSDH26
	} {
		assert.Equalf(t, tc.exp, hasPotentialDelimiter(tc.asset), "hasPotentialDelimiter should report whether %s symbols separate base and quote with a dash", tc.asset)
	}
}

func TestStringToInterval(t *testing.T) {
	t.Parallel()
	interval, err := stringToInterval("60")
	require.NoError(t, err, "stringToInterval must not error")
	assert.Equal(t, kline.OneHour, interval, "stringToInterval should parse minutes")
	interval, err = stringToInterval("720")
	require.NoError(t, err, "stringToInterval must not error")
	assert.Equal(t, kline.TwelveHour, interval, "stringToInterval should parse 720 minutes as 12 hours")
	_, err = stringToInterval("2")
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "stringToInterval must reject an interval Bybit does not offer")
}
