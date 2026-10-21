package kucoin

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	utaSpotTestPair    = currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter)
	utaFuturesTestPair = currency.NewPairWithDelimiter("XBT", "USDTM", currency.UnderscoreDelimiter)
)

// newUTATestExchange returns an exchange for UTA websocket tests. It is named after the test, as tickers and orderbooks
// are stored by exchange name across the test binary, carries placeholder credentials, and has the pairs the fixtures
// name enabled: BTC-USDT for spot and margin, and XBTUSDTM for futures
func newUTATestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := testInstance(t)
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: utaSpotTestPair, asset.Margin: utaSpotTestPair, asset.Futures: utaFuturesTestPair} {
		available, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		if !available.Contains(p, true) {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, append(available, p), false), "StorePairs must not error for %s", a)
		}
		if enabled, err := ex.IsPairEnabled(p, a); err != nil || !enabled {
			require.NoErrorf(t, ex.CurrencyPairs.EnablePair(a, p), "EnablePair must not error for %s", a)
		}
	}
	return ex
}

// newUTAWebsocket gives an exchange a fresh multi-connection websocket manager holding the given connections, with the
// trade and fill feeds enabled
func newUTAWebsocket(t *testing.T, ex *Exchange, setups ...*websocket.ConnectionSetup) {
	t.Helper()
	ex.Websocket = websocket.NewManager()
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         ex.Config,
		Features:                               &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		TradeFeed:                              true,
		FillsFeed:                              true,
		MaxWebsocketSubscriptionsPerConnection: 400,
	}), "Websocket Setup must not error")
	for _, s := range setups {
		require.NoError(t, ex.Websocket.SetupNewConnection(s), "SetupNewConnection must not error")
	}
	t.Cleanup(func() {
		// Disabling first stops the connection monitor reconnecting with the test's cancelled context
		assert.NoError(t, ex.Websocket.Disable(), "Websocket Disable should not error")
		if ex.Websocket.IsConnected() {
			assert.NoError(t, ex.Websocket.Shutdown(), "Websocket Shutdown should not error")
		}
	})
}

// newUTAMockServer serves a websocket on loopback which sends KuCoin's welcome frame on connecting when asked to, then
// answers each client message with the frames reply returns; frames are binary, as KuCoin's push hosts send them
func newUTAMockServer(t *testing.T, welcome bool, reply func(msg map[string]any) []string) string {
	t.Helper()
	upgrader := gws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		if welcome {
			if err := c.WriteMessage(gws.BinaryMessage, []byte(`{"sessionId":"81f56400-e7e3-4f2b-b0f6-5e649f48c302","message":"welcome","pingInterval":18000}`)); err != nil {
				return
			}
		}
		for {
			_, p, err := c.ReadMessage()
			if err != nil {
				return
			}
			var msg map[string]any
			if !assert.NoError(t, json.Unmarshal(p, &msg), "client message should be JSON") {
				return
			}
			for _, frame := range reply(msg) {
				if err := c.WriteMessage(gws.BinaryMessage, []byte(frame)); err != nil {
					return
				}
			}
		}
	}))
	s.Start()
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

// utaAck returns the reply KuCoin sends to a subscribe, unsubscribe or authentication message it accepts
func utaAck(msg map[string]any) string {
	id, _ := msg["id"].(string)
	return `{"id":"` + id + `","result":true}`
}

// utaFixtureLines returns the lines of a websocket fixture
func utaFixtureLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoErrorf(t, err, "opening %s must not error", path)
	defer f.Close()
	var lines []string
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	require.NoErrorf(t, s.Err(), "reading %s must not error", path)
	return lines
}

// drainUTARelay returns what the exchange has relayed so far
func drainUTARelay(ex *Exchange) []any {
	var out []any
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			out = append(out, p.Data)
		default:
			return out
		}
	}
}

// awaitUTARelay waits for the first relayed value of type T
func awaitUTARelay[T any](t *testing.T, ex *Exchange, timeout time.Duration) T {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			if v, ok := p.Data.(T); ok {
				return v
			}
			if err, ok := p.Data.(error); ok {
				t.Logf("relayed while awaiting %T: %v", *new(T), err)
			}
		case <-deadline:
			var zero T
			require.FailNowf(t, "relay timed out", "no %T must arrive later than %s", zero, timeout)
		}
	}
}

// utaFakeConnection is a websocket connection which records what it is given and answers from canned values
type utaFakeConnection struct {
	websocket.Connection
	mu          sync.Mutex
	dialer      *gws.Dialer
	dialErr     error
	pingHandler websocket.PingHandler
	sent        []any
	signatures  []any
	reply       []byte
	sendErr     error
	matched     []any
	matchErr    error
	subs        *subscription.Store
}

func (c *utaFakeConnection) Dial(_ context.Context, d *gws.Dialer, _ http.Header, _ url.Values) error {
	c.dialer = d
	return c.dialErr
}

func (c *utaFakeConnection) SetupPingHandler(_ request.EndpointLimit, h websocket.PingHandler) {
	c.pingHandler = h
}

func (c *utaFakeConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, signature, req any) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, req)
	c.signatures = append(c.signatures, signature)
	return c.reply, c.sendErr
}

func (c *utaFakeConnection) RequireMatchWithData(signature any, _ []byte) error {
	c.matched = append(c.matched, signature)
	return c.matchErr
}

func (c *utaFakeConnection) Subscriptions() *subscription.Store {
	return c.subs
}

func TestNewUTASpotPublicConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Millisecond, WebsocketResponseMaxLimit: time.Second}
	c := ex.newUTASpotPublicConnection(exch, kucoinUTASpotPublicWebsocketURL)
	require.NotNil(t, c, "newUTASpotPublicConnection must return a setup")
	assert.Equal(t, kucoinUTASpotPublicWebsocketURL, c.URL, "URL should be the spot public host")
	assert.Equal(t, time.Millisecond, c.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, time.Second, c.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.Equal(t, utaSpotPublicConnection, c.MessageFilter, "MessageFilter should name the spot public connection")
	assert.True(t, c.Optional, "the UTA spot public connection should be optional, so its failure leaves the others up")
	assert.Equal(t, newUTAPublicConnectionLimiter(), c.ConnectionRateLimiter(), "ConnectionRateLimiter should be the public limit")
	assert.NotNil(t, c.Connector, "Connector should be set")
	assert.NotNil(t, c.Subscriber, "Subscriber should be set")
	assert.NotNil(t, c.Unsubscriber, "Unsubscriber should be set")
	assert.NotNil(t, c.Handler, "Handler should be set")
	assert.Nil(t, c.Authenticate, "Authenticate should not be set")
	assert.False(t, c.SubscriptionsNotRequired, "SubscriptionsNotRequired should be false")
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
	}
	subs, err := c.GenerateSubscriptions()
	require.NoError(t, err, "GenerateSubscriptions must not error")
	require.Len(t, subs, 1, "GenerateSubscriptions must generate the spot ticker alone")
	assert.Equal(t, asset.Spot, subs[0].Asset, "GenerateSubscriptions should generate the spot ticker")
}

func TestNewUTAFuturesPublicConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Millisecond, WebsocketResponseMaxLimit: time.Second}
	c := ex.newUTAFuturesPublicConnection(exch, kucoinUTAFuturesPublicWebsocketURL)
	require.NotNil(t, c, "newUTAFuturesPublicConnection must return a setup")
	assert.Equal(t, kucoinUTAFuturesPublicWebsocketURL, c.URL, "URL should be the futures public host")
	assert.Equal(t, time.Millisecond, c.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, time.Second, c.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.Equal(t, utaFuturesPublicConnection, c.MessageFilter, "MessageFilter should name the futures public connection")
	assert.True(t, c.Optional, "the UTA futures public connection should be optional, so its failure leaves the others up")
	assert.Equal(t, newUTAPublicConnectionLimiter(), c.ConnectionRateLimiter(), "ConnectionRateLimiter should be the public limit")
	assert.NotNil(t, c.Connector, "Connector should be set")
	assert.NotNil(t, c.Subscriber, "Subscriber should be set")
	assert.NotNil(t, c.Unsubscriber, "Unsubscriber should be set")
	assert.NotNil(t, c.Handler, "Handler should be set")
	assert.Nil(t, c.Authenticate, "Authenticate should not be set")
	assert.False(t, c.SubscriptionsNotRequired, "SubscriptionsNotRequired should be false")
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
	}
	subs, err := c.GenerateSubscriptions()
	require.NoError(t, err, "GenerateSubscriptions must not error")
	require.Len(t, subs, 1, "GenerateSubscriptions must generate the futures ticker alone")
	assert.Equal(t, asset.Futures, subs[0].Asset, "GenerateSubscriptions should generate the futures ticker")
}

func TestNewUTAPrivateConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Millisecond, WebsocketResponseMaxLimit: time.Second}
	c := ex.newUTAPrivateConnection(exch, kucoinUTAPrivateWebsocketURL)
	require.NotNil(t, c, "newUTAPrivateConnection must return a setup")
	assert.Equal(t, kucoinUTAPrivateWebsocketURL, c.URL, "URL should be the private push host")
	assert.Equal(t, time.Millisecond, c.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, time.Second, c.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.Equal(t, utaPrivateConnection, c.MessageFilter, "MessageFilter should name the private connection")
	assert.True(t, c.Optional, "the UTA private connection should be optional, so its failure leaves the others up")
	assert.Equal(t, newUTAPrivateConnectionLimiter(), c.ConnectionRateLimiter(), "ConnectionRateLimiter should be the private limit")
	assert.NotNil(t, c.Connector, "Connector should be set")
	assert.NotNil(t, c.Authenticate, "Authenticate should be set")
	assert.NotNil(t, c.Subscriber, "Subscriber should be set")
	assert.NotNil(t, c.Unsubscriber, "Unsubscriber should be set")
	assert.NotNil(t, c.Handler, "Handler should be set")
	assert.False(t, c.SubscriptionsNotRequired, "SubscriptionsNotRequired should be false")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
		{Enabled: true, Channel: UTAAllOrdersChannel},
	}
	subs, err := c.GenerateSubscriptions()
	require.NoError(t, err, "GenerateSubscriptions must not error")
	require.Len(t, subs, 1, "GenerateSubscriptions must generate the order channel alone")
	assert.Equal(t, UTAAllOrdersChannel, subs[0].Channel, "GenerateSubscriptions should generate the order channel")
}

func TestNewUTATradeConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Millisecond, WebsocketResponseMaxLimit: time.Second}
	// KuCoin documents the UTA trade connection on the v2 path, apart from the Classic order API's v1
	assert.Equal(t, "wss://wsapi.kucoin.com/v2/private", kucoinUTATradeWebsocketURL, "the UTA trade connection should use the documented v2 host")
	c := ex.newUTATradeConnection(exch, kucoinUTATradeWebsocketURL)
	require.NotNil(t, c, "newUTATradeConnection must return a setup")
	assert.Equal(t, kucoinUTATradeWebsocketURL, c.URL, "URL should be the trade host")
	assert.Equal(t, time.Millisecond, c.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, time.Second, c.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.Equal(t, utaTradeConnection, c.MessageFilter, "MessageFilter should name the trade connection")
	assert.True(t, c.Optional, "the UTA trade connection should be optional, so its failure leaves the others up")
	assert.Nil(t, c.ConnectionRateLimiter, "ConnectionRateLimiter should not be set, as orders share the REST limit")
	assert.NotNil(t, c.Connector, "Connector should be set")
	assert.NotNil(t, c.Authenticate, "Authenticate should be set")
	assert.NotNil(t, c.Handler, "Handler should be set")
	assert.Nil(t, c.GenerateSubscriptions, "GenerateSubscriptions should not be set")
	assert.Nil(t, c.Subscriber, "Subscriber should not be set")
	assert.True(t, c.SubscriptionsNotRequired, "SubscriptionsNotRequired should be true")
}

func TestHasUTASubscriptions(t *testing.T) {
	t.Parallel()
	ex := &Exchange{}
	assert.False(t, ex.hasUTASubscriptions(), "hasUTASubscriptions should report no UTA channel without subscriptions")
	ex.Features.Subscriptions = defaultSubscriptions.Clone()
	assert.False(t, ex.hasUTASubscriptions(), "hasUTASubscriptions should report no UTA channel for the Classic defaults")
	ex.Features.Subscriptions = append(ex.Features.Subscriptions, &subscription.Subscription{Enabled: true, Channel: UTABalanceChannel})
	assert.True(t, ex.hasUTASubscriptions(), "hasUTASubscriptions should report a configured UTA channel")
}

func TestIsUTASubscription(t *testing.T) {
	t.Parallel()
	for channel := range utaChannels {
		assert.Truef(t, isUTASubscription(&subscription.Subscription{Channel: channel}), "isUTASubscription should report %s", channel)
	}
	for _, s := range defaultSubscriptions {
		assert.Falsef(t, isUTASubscription(s), "isUTASubscription should not report the Classic channel %s", s.Channel)
	}
}

// TestUTAConnectionsStayDownByDefault proves that registering the UTA push connections connects nothing new unless
// the configuration lists UTA channels
func TestUTAConnectionsStayDownByDefault(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	var mu sync.Mutex
	var dials int
	upgrader := gws.Upgrader{}
	s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dials++
		mu.Unlock()
		if c, err := upgrader.Upgrade(w, r, nil); err == nil {
			_ = c.Close()
		}
	}))
	s.Start()
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	newUTAWebsocket(
		t, ex,
		ex.newUTASpotPublicConnection(ex.Config, wsURL),
		ex.newUTAFuturesPublicConnection(ex.Config, wsURL+"/futures"),
		ex.newUTAPrivateConnection(ex.Config, wsURL+"/private"),
	)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = defaultSubscriptions.Clone()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	mu.Lock()
	defer mu.Unlock()
	assert.Zero(t, dials, "no UTA connection should be dialled without configured UTA channels")
}

func TestNewUTAPublicConnectionLimiter(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 300, 1), newUTAPublicConnectionLimiter(), "newUTAPublicConnectionLimiter should allow 300 messages per 10 seconds")
}

func TestNewUTAPrivateConnectionLimiter(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 100, 1), newUTAPrivateConnectionLimiter(), "newUTAPrivateConnectionLimiter should allow 100 messages per 10 seconds")
}

func TestUTAWebsocketConnect(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{dialErr: errors.New("dial failure")}
	require.ErrorIs(t, ex.UTAWebsocketConnect(t.Context(), conn), conn.dialErr, "UTAWebsocketConnect must return the dial error")
	assert.Zero(t, conn.pingHandler, "UTAWebsocketConnect should not start pinging a connection that failed to dial")

	conn = &utaFakeConnection{}
	require.NoError(t, ex.UTAWebsocketConnect(t.Context(), conn), "UTAWebsocketConnect must not error")
	require.NotNil(t, conn.dialer, "UTAWebsocketConnect must dial")
	assert.Equal(t, ex.Config.HTTPTimeout, conn.dialer.HandshakeTimeout, "UTAWebsocketConnect should bound the handshake by the HTTP timeout")
	assert.NotNil(t, conn.dialer.Proxy, "UTAWebsocketConnect should use the environment's proxy")
	exp := websocket.PingHandler{Delay: 18 * time.Second, Message: []byte(`{"id":"ping","op":"ping"}`), MessageType: gws.TextMessage}
	assert.Equal(t, exp, conn.pingHandler, "UTAWebsocketConnect should ping every 18 seconds")
}

func TestUTAAuthRequest(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	before := time.Now().UnixMilli()
	req, err := ex.utaAuthRequest(t.Context())
	require.NoError(t, err, "utaAuthRequest must not error")
	require.NotNil(t, req, "utaAuthRequest must return a request")
	assert.GreaterOrEqual(t, req.Timestamp, uint64(before), "utaAuthRequest should stamp the request with the current time")
	sign, err := crypto.GetHMAC(crypto.HashSHA256, []byte(strconv.FormatUint(req.Timestamp, 10)+"POST/api/websocket/users/verify"), []byte("mock-api-secret"))
	require.NoError(t, err, "GetHMAC must not error")
	passphrase, err := crypto.GetHMAC(crypto.HashSHA256, []byte("mock-passphrase"), []byte("mock-api-secret"))
	require.NoError(t, err, "GetHMAC must not error")
	exp := &UTAWebsocketAuthRequest{
		Operation:  "auth",
		APIKey:     "mock-api-key",
		Timestamp:  req.Timestamp,
		Signature:  base64.StdEncoding.EncodeToString(sign),
		Passphrase: base64.StdEncoding.EncodeToString(passphrase),
	}
	assert.Equal(t, exp, req, "utaAuthRequest should sign the timestamp and the verify path")

	ex.SkipAuthCheck = false // The mock instance skips credential checks
	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.utaAuthRequest(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "utaAuthRequest should error without credentials")
}

func TestAuthenticateUTAPrivateWebsocket(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{reply: []byte(`{"id":"71611efd-925e-4683-9363-367d479b5713","result":true}`)}
	require.NoError(t, ex.AuthenticateUTAPrivateWebsocket(t.Context(), conn), "AuthenticateUTAPrivateWebsocket must not error")
	require.Len(t, conn.sent, 1, "AuthenticateUTAPrivateWebsocket must send one message")
	req, ok := conn.sent[0].(*UTAWebsocketAuthRequest)
	require.True(t, ok, "AuthenticateUTAPrivateWebsocket must send an authentication request")
	assert.Len(t, req.ID, 32, "the authentication message should carry an ID")
	assert.Equal(t, req.ID, conn.signatures[0], "the reply should be matched by the authentication message's ID")
	assert.Equal(t, "auth", req.Operation, "the authentication message should be an auth operation")

	for _, tc := range []struct {
		name  string
		conn  *utaFakeConnection
		expIs error
	}{
		{name: "rejected", conn: &utaFakeConnection{reply: []byte(`{"id":"dfc046e6-35e7-4d34-80f2-2a51415b35b9","result":false,"message":"auth failed"}`)}, expIs: errUTAAuthenticationFailed},
		{name: "send failure", conn: &utaFakeConnection{sendErr: websocket.ErrSignatureTimeout}, expIs: websocket.ErrSignatureTimeout},
	} {
		err := ex.AuthenticateUTAPrivateWebsocket(t.Context(), tc.conn)
		assert.ErrorIsf(t, err, tc.expIs, "AuthenticateUTAPrivateWebsocket should error for %s", tc.name)
	}
	assert.Error(t, ex.AuthenticateUTAPrivateWebsocket(t.Context(), &utaFakeConnection{reply: []byte(`{`)}), "AuthenticateUTAPrivateWebsocket should error on a malformed reply")
	assert.ErrorIs(t, ex.AuthenticateUTAPrivateWebsocket(t.Context(), &utaFakeConnection{reply: []byte(`null`)}), common.ErrNoResponse, "AuthenticateUTAPrivateWebsocket should error on a null reply")

	ex.SkipAuthCheck = false // The mock instance skips credential checks
	ex.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, ex.AuthenticateUTAPrivateWebsocket(t.Context(), &utaFakeConnection{}), exchange.ErrCredentialsAreEmpty, "AuthenticateUTAPrivateWebsocket should error without credentials")
}

func TestGenerateUTASubscriptions(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.All},
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
		{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{utaSpotTestPair}, Levels: 5},
		{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Interval: kline.OneMin},
		{Enabled: true, Channel: UTAAllFundingFeeRatesChannel, Asset: asset.Futures},
		{Enabled: true, Channel: UTAAllOrdersChannel},
		{Enabled: true, Channel: UTAPositionChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
	}
	spot, err := ex.generateUTASubscriptions(utaSpotPublicConnection)
	require.NoError(t, err, "generateUTASubscriptions must not error for the spot public connection")
	exp := subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, QualifiedChannel: "ticker:BTC-USDT"},
		{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{utaSpotTestPair}, Levels: 5, QualifiedChannel: "obu:BTC-USDT"},
	}
	assert.Equal(t, exp, spot, "generateUTASubscriptions should expand the spot and margin public channels")

	futuresSubs, err := ex.generateUTASubscriptions(utaFuturesPublicConnection)
	require.NoError(t, err, "generateUTASubscriptions must not error for the futures public connection")
	exp = subscription.List{
		{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Interval: kline.OneMin, QualifiedChannel: "kline:XBTUSDTM"},
		{Enabled: true, Channel: UTAAllFundingFeeRatesChannel, Asset: asset.Futures, QualifiedChannel: "funding-fee-all-symbols"},
	}
	assert.Equal(t, exp, futuresSubs, "generateUTASubscriptions should expand the futures public channels")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	private, err := ex.generateUTASubscriptions(utaPrivateConnection)
	require.NoError(t, err, "generateUTASubscriptions must not error for the private connection")
	assert.Empty(t, private, "generateUTASubscriptions should generate no private channels without authentication")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	private, err = ex.generateUTASubscriptions(utaPrivateConnection)
	require.NoError(t, err, "generateUTASubscriptions must not error for the private connection")
	exp = subscription.List{
		{Enabled: true, Channel: UTAAllOrdersChannel, Authenticated: true, QualifiedChannel: "orderAll"},
		{Enabled: true, Channel: UTAPositionChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Authenticated: true, QualifiedChannel: "position:XBTUSDTM"},
	}
	assert.Equal(t, exp, private, "generateUTASubscriptions should expand the private channels")

	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTAMarkPriceChannel, Asset: asset.Spot}}
	_, err = ex.generateUTASubscriptions(utaSpotPublicConnection)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "generateUTASubscriptions should error for a channel the asset is not served")

	ex.Features.Subscriptions = subscription.List{}
	none, err := ex.generateUTASubscriptions(utaSpotPublicConnection)
	require.NoError(t, err, "generateUTASubscriptions must not error without UTA channels")
	assert.Empty(t, none, "generateUTASubscriptions should generate nothing without UTA channels")
}

// TestGenerateUTASubscriptionsAssets proves that UTA channels expand over the assets they stand for as the Classic
// channels do, without failing the websocket for a disabled asset, and that each KuCoin topic is subscribed to once
// TestGenerateUTASubscriptionsFatalErrors covers an error that is not about a subscription's interval or depth, which
// leaves the connection's whole generation failed rather than one subscription out
func TestGenerateUTASubscriptionsFatalErrors(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures},
		{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Futures, Interval: kline.OneMin},
	}
	ex.CurrencyPairs.Pairs[asset.Futures].RequestFormat = nil
	subs, err := ex.generateUTASubscriptions(utaFuturesPublicConnection)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "generateUTASubscriptions must return the pair format error")
	assert.NotErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateUTASubscriptions should not report a pair format error as a partial failure")
	assert.Nil(t, subs, "generateUTASubscriptions should generate nothing when a subscription cannot be expanded")
}

func TestGenerateUTASubscriptionsAssets(t *testing.T) {
	t.Parallel()
	spotPairs, futuresPairs := currency.Pairs{utaSpotTestPair}, currency.Pairs{utaFuturesTestPair}
	for _, tc := range []struct {
		name       string
		disabled   asset.Item
		subs       subscription.List
		connection string
		exp        subscription.List
		err        error
		// partial is whether the error leaves out the failing subscriptions alone, a partial failure
		partial bool
	}{
		{
			name:       "disabled asset without pairs",
			disabled:   asset.Margin,
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Margin}},
			connection: utaSpotPublicConnection,
		},
		{
			name:       "disabled asset with pairs",
			disabled:   asset.Margin,
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Margin, Pairs: spotPairs}},
			connection: utaSpotPublicConnection,
		},
		{
			name:       "all assets on the spot market",
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.All}},
			connection: utaSpotPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: spotPairs, QualifiedChannel: "ticker:BTC-USDT"}},
		},
		{
			name:       "all assets on the futures market",
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.All}},
			connection: utaFuturesPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: futuresPairs, QualifiedChannel: "ticker:XBTUSDTM"}},
		},
		{
			name:       "all assets with futures disabled",
			disabled:   asset.Futures,
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.All}},
			connection: utaFuturesPublicConnection,
		},
		{
			name:       "assetless futures channel",
			subs:       subscription.List{{Enabled: true, Channel: UTAAllFundingFeeRatesChannel}},
			connection: utaFuturesPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTAAllFundingFeeRatesChannel, Asset: asset.Futures, QualifiedChannel: "funding-fee-all-symbols"}},
		},
		{
			name:       "assetless futures channel on the spot market",
			subs:       subscription.List{{Enabled: true, Channel: UTAAllFundingFeeRatesChannel}},
			connection: utaSpotPublicConnection,
		},
		{
			name:       "assetless futures symbol channel",
			subs:       subscription.List{{Enabled: true, Channel: UTAMarkPriceChannel}},
			connection: utaFuturesPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTAMarkPriceChannel, Asset: asset.Futures, Pairs: futuresPairs, QualifiedChannel: "mark-price:XBTUSDTM"}},
		},
		{
			name:       "assetless spot symbol channel",
			subs:       subscription.List{{Enabled: true, Channel: UTACallAuctionInfoChannel}},
			connection: utaSpotPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTACallAuctionInfoChannel, Asset: asset.Spot, Pairs: spotPairs, QualifiedChannel: "callAuctionInfo:BTC-USDT"}},
		},
		{
			name:       "assetless channel serving both markets",
			subs:       subscription.List{{Enabled: true, Channel: UTATickerChannel}},
			connection: utaSpotPublicConnection,
		},
		{
			name: "spot and margin sharing a topic",
			subs: subscription.List{
				{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: spotPairs},
				{Enabled: true, Channel: UTATickerChannel, Asset: asset.Margin, Pairs: spotPairs},
			},
			connection: utaSpotPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: spotPairs, QualifiedChannel: "ticker:BTC-USDT"}},
		},
		{
			name: "orderbook depths of one symbol",
			subs: subscription.List{
				{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: spotPairs, Levels: 5},
				{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Margin, Pairs: spotPairs, Levels: 50},
				{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: spotPairs, Levels: 500},
			},
			connection: utaSpotPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: spotPairs, Levels: 5, QualifiedChannel: "obu:BTC-USDT"}},
		},
		{
			name: "kline intervals of one symbol",
			subs: subscription.List{
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Spot, Pairs: spotPairs, Interval: kline.OneMin},
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Margin, Pairs: spotPairs, Interval: kline.FiveMin},
			},
			connection: utaSpotPublicConnection,
			exp: subscription.List{
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Spot, Pairs: spotPairs, Interval: kline.OneMin, QualifiedChannel: "kline:BTC-USDT"},
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Margin, Pairs: spotPairs, Interval: kline.FiveMin, QualifiedChannel: "kline:BTC-USDT"},
			},
		},
		{
			name:       "private symbol channel over every asset",
			subs:       subscription.List{{Enabled: true, Channel: UTAOrderChannel, Asset: asset.All}},
			connection: utaPrivateConnection,
			exp: subscription.List{
				{Enabled: true, Channel: UTAOrderChannel, Asset: asset.Spot, Pairs: spotPairs, Authenticated: true, QualifiedChannel: "order:BTC-USDT"},
				{Enabled: true, Channel: UTAOrderChannel, Asset: asset.Futures, Pairs: futuresPairs, Authenticated: true, QualifiedChannel: "order:XBTUSDTM"},
			},
		},
		{
			name:       "unsupported interval",
			subs:       subscription.List{{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Futures, Interval: kline.SixHour}},
			connection: utaFuturesPublicConnection,
			err:        subscription.ErrInvalidInterval,
			partial:    true,
		},
		{
			// Futures klines have no 6 hour interval, which spot's have
			name: "interval one of every asset's markets lacks",
			subs: subscription.List{
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.All, Interval: kline.SixHour},
				{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures},
			},
			connection: utaFuturesPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: futuresPairs, QualifiedChannel: "ticker:XBTUSDTM"}},
			err:        subscription.ErrInvalidInterval,
			partial:    true,
		},
		{
			name: "interval one of every asset's markets lacks, on the market that has it",
			subs: subscription.List{
				{Enabled: true, Channel: UTAKlineChannel, Asset: asset.All, Interval: kline.SixHour},
				{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures},
			},
			connection: utaSpotPublicConnection,
			// Margin shares spot's topic, which is subscribed to once
			exp: subscription.List{{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Spot, Pairs: spotPairs, Interval: kline.SixHour, QualifiedChannel: "kline:BTC-USDT"}},
		},
		{
			name:       "unsupported interval on another market",
			subs:       subscription.List{{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Futures, Interval: kline.SixHour}},
			connection: utaSpotPublicConnection,
		},
		{
			name:       "unsupported depth",
			subs:       subscription.List{{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.All, Levels: 20}},
			connection: utaSpotPublicConnection,
			err:        subscription.ErrInvalidLevel,
			partial:    true,
		},
		{
			name: "unsupported depth beside a supported one",
			subs: subscription.List{
				{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 20},
				{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot},
			},
			connection: utaSpotPublicConnection,
			exp:        subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: spotPairs, QualifiedChannel: "ticker:BTC-USDT"}},
			err:        subscription.ErrInvalidLevel,
			partial:    true,
		},
		{
			name:       "asset the channel does not serve",
			subs:       subscription.List{{Enabled: true, Channel: UTAMarkPriceChannel, Asset: asset.Spot}},
			connection: utaSpotPublicConnection,
			err:        subscription.ErrNotSupported,
		},
		{
			name:       "asset the channel does not serve on another market",
			subs:       subscription.List{{Enabled: true, Channel: UTAMarkPriceChannel, Asset: asset.Spot}},
			connection: utaFuturesPublicConnection,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newUTATestExchange(t)
			for a, p := range map[asset.Item]currency.Pairs{asset.Spot: spotPairs, asset.Margin: spotPairs, asset.Futures: futuresPairs} {
				require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, p, true), "StorePairs must not error for %s", a)
			}
			if tc.disabled != asset.Empty {
				require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(tc.disabled, false), "SetAssetEnabled must not error")
			}
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = tc.subs
			subs, err := ex.generateUTASubscriptions(tc.connection)
			require.ErrorIs(t, err, tc.err, "generateUTASubscriptions must return the expected error")
			if tc.partial {
				require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateUTASubscriptions must report the subscriptions it leaves out as a partial failure")
			} else if err != nil {
				require.NotErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateUTASubscriptions must not report other errors as partial failures")
			}
			assert.Equal(t, tc.exp, subs, "generateUTASubscriptions should generate the expected subscriptions")
		})
	}
}

// TestUTAConnectWithDisabledAsset proves that a UTA channel configured for a disabled asset leaves the websocket's other
// connections to connect
func TestUTAConnectWithDisabledAsset(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	push := utaFixtureLines(t, "testdata/wsUTATicker.json")[1]
	futuresURL := newUTAMockServer(t, true, func(msg map[string]any) []string { return []string{utaAck(msg), push} })
	spotURL := newUTAMockServer(t, true, func(msg map[string]any) []string { return []string{utaAck(msg)} })
	newUTAWebsocket(t, ex, ex.newUTASpotPublicConnection(ex.Config, spotURL), ex.newUTAFuturesPublicConnection(ex.Config, futuresURL))
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Margin},
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
	}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error for a channel of a disabled asset")
	_, err := ex.Websocket.GetConnection(utaFuturesPublicConnection)
	assert.NoError(t, err, "the futures public connection should connect")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "only the futures ticker should be subscribed to")
}

func TestUTAConnectionFor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		a       asset.Item
		exp     string
	}{
		{channel: UTATickerChannel, a: asset.Spot, exp: utaSpotPublicConnection},
		{channel: UTATickerChannel, a: asset.Margin, exp: utaSpotPublicConnection},
		{channel: UTATickerChannel, a: asset.Futures, exp: utaFuturesPublicConnection},
		{channel: UTATickerChannel, a: asset.Empty, exp: utaSpotPublicConnection},
		{channel: UTAMarkPriceChannel, a: asset.Empty, exp: utaFuturesPublicConnection},
		{channel: UTACallAuctionInfoChannel, a: asset.Empty, exp: utaSpotPublicConnection},
		{channel: UTAOrderChannel, a: asset.Futures, exp: utaPrivateConnection},
		{channel: UTABalanceChannel, a: asset.Empty, exp: utaPrivateConnection},
	} {
		assert.Equalf(t, tc.exp, utaConnectionFor(utaChannels[tc.channel], tc.a), "utaConnectionFor should route %s %s", tc.channel, tc.a)
	}
}

func TestUTASubscriptionAssets(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  asset.Items
		err  error
	}{
		{name: "account channel", sub: &subscription.Subscription{Channel: UTABalanceChannel, Asset: asset.Margin}, exp: asset.Items{asset.Margin}},
		{name: "all assets", sub: &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.All}, exp: asset.Items{asset.Spot, asset.Futures}},
		{name: "assetless futures channel", sub: &subscription.Subscription{Channel: UTAMarkPriceChannel}, exp: asset.Items{asset.Futures}},
		{name: "assetless spot channel", sub: &subscription.Subscription{Channel: UTACallAuctionInfoChannel}, exp: asset.Items{asset.Spot}},
		{name: "assetless channel serving both markets", sub: &subscription.Subscription{Channel: UTATickerChannel}, exp: asset.Items{}},
		{name: "served asset", sub: &subscription.Subscription{Channel: UTAPositionChannel, Asset: asset.Futures}, exp: asset.Items{asset.Futures}},
		{name: "disabled asset", sub: &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Margin}, exp: asset.Items{}},
		{name: "spot mark price", sub: &subscription.Subscription{Channel: UTAMarkPriceChannel, Asset: asset.Spot}, err: subscription.ErrNotSupported},
		{name: "futures call auction", sub: &subscription.Subscription{Channel: UTACallAuctionInfoChannel, Asset: asset.Futures}, err: subscription.ErrNotSupported},
		{name: "spot position", sub: &subscription.Subscription{Channel: UTAPositionChannel, Asset: asset.Spot}, err: subscription.ErrNotSupported},
		{name: "options ticker", sub: &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Options}, err: subscription.ErrNotSupported},
	} {
		got, err := ex.utaSubscriptionAssets(tc.sub, utaChannels[tc.sub.Channel])
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "utaSubscriptionAssets should reject %s", tc.name)
			continue
		}
		require.NoErrorf(t, err, "utaSubscriptionAssets must not error for %s", tc.name)
		if len(tc.exp) == 0 {
			assert.Emptyf(t, got, "utaSubscriptionAssets should return no asset for %s", tc.name)
			continue
		}
		assert.Equalf(t, tc.exp, got, "utaSubscriptionAssets should return the assets of %s", tc.name)
	}
}

func TestUTAChannelAssets(t *testing.T) {
	t.Parallel()
	assert.Equal(t, asset.Items{asset.Spot, asset.Margin, asset.Futures}, utaChannels[UTATickerChannel].assets(), "assets should cover both markets of a ticker")
	assert.Equal(t, asset.Items{asset.Spot, asset.Margin}, utaChannels[UTACallAuctionInfoChannel].assets(), "assets should cover the spot market of a call auction")
	assert.Equal(t, asset.Items{asset.Futures}, utaChannels[UTAPositionChannel].assets(), "assets should cover the futures market of a position")
	assert.Empty(t, utaChannels[UTABalanceChannel].assets(), "assets should be empty for an account channel")
}

func TestExpandUTASubscription(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	s := &subscription.Subscription{Enabled: true, Channel: UTATradeChannel, Asset: asset.All}
	subs, err := ex.expandUTASubscription(s, utaChannels[UTATradeChannel], asset.Spot)
	require.NoError(t, err, "expandUTASubscription must not error")
	enabled, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.Len(t, subs, len(enabled), "expandUTASubscription must generate one subscription per enabled pair")
	for i, sub := range subs {
		assert.Equalf(t, asset.Spot, sub.Asset, "subscription %d should be for the asset expanded", i)
		assert.Equalf(t, currency.Pairs{enabled[i]}, sub.Pairs, "subscription %d should hold one enabled pair", i)
		assert.Equalf(t, "trade:"+enabled[i].Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(), sub.QualifiedChannel, "subscription %d should qualify the channel with its symbol", i)
	}

	s = &subscription.Subscription{Enabled: true, Channel: UTAExecutionChannel}
	subs, err = ex.expandUTASubscription(s, utaChannels[UTAExecutionChannel], asset.Empty)
	require.NoError(t, err, "expandUTASubscription must not error")
	exp := subscription.List{{Enabled: true, Channel: UTAExecutionChannel, Authenticated: true, QualifiedChannel: "execution"}}
	assert.Equal(t, exp, subs, "expandUTASubscription should keep an account channel whole")

	_, err = ex.expandUTASubscription(&subscription.Subscription{Channel: UTAKlineChannel, Asset: asset.Spot}, utaChannels[UTAKlineChannel], asset.Spot)
	assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "expandUTASubscription should error for a kline without an interval")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	_, err = ex.expandUTASubscription(&subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Margin}, utaChannels[UTATickerChannel], asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotEnabled, "expandUTASubscription should error for the pairs of a disabled asset")

	ex.CurrencyPairs.Delete(asset.Futures)
	_, err = ex.expandUTASubscription(&subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}}, utaChannels[UTATickerChannel], asset.Futures)
	assert.ErrorIs(t, err, currency.ErrAssetNotFound, "expandUTASubscription should error for an asset without a pair format")
}

// BenchmarkExpandUTASubscription expands a UTA subscription naming 1,000 pairs. Each pair's subscription is cloned
// without the configured pairs, so the bytes allocated grow linearly with them; cloning the pairs for every pair would
// allocate about 49 MB an expansion
func BenchmarkExpandUTASubscription(b *testing.B) {
	ex := testInstance(b)
	pairs := make(currency.Pairs, 1000)
	for i := range pairs {
		pairs[i] = currency.NewPair(currency.NewCode("C"+strconv.Itoa(i)), currency.USDT)
	}
	s := &subscription.Subscription{Enabled: true, Channel: UTATradeChannel, Asset: asset.Spot, Pairs: pairs}
	b.ReportAllocs()
	for b.Loop() {
		_, err := ex.expandUTASubscription(s, utaChannels[UTATradeChannel], asset.Spot)
		require.NoError(b, err, "expandUTASubscription must not error")
	}
}

func TestValidateUTASubscription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		a    asset.Item
		err  error
	}{
		{name: "spot ticker", sub: &subscription.Subscription{Channel: UTATickerChannel}, a: asset.Spot},
		{name: "margin orderbook", sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Levels: 500}, a: asset.Margin},
		{name: "futures kline", sub: &subscription.Subscription{Channel: UTAKlineChannel, Interval: kline.OneMonth}, a: asset.Futures},
		{name: "spot 6 hour kline", sub: &subscription.Subscription{Channel: UTAKlineChannel, Interval: kline.SixHour}, a: asset.Spot},
		{name: "futures 6 hour kline", sub: &subscription.Subscription{Channel: UTAKlineChannel, Interval: kline.SixHour}, a: asset.Futures, err: subscription.ErrInvalidInterval},
		{name: "orderbook depth 20", sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Levels: 20}, a: asset.Spot, err: subscription.ErrInvalidLevel},
	} {
		err := validateUTASubscription(tc.sub, tc.a)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "validateUTASubscription should reject %s", tc.name)
			continue
		}
		assert.NoErrorf(t, err, "validateUTASubscription should accept %s", tc.name)
	}
}

func TestUTAKlineInterval(t *testing.T) {
	t.Parallel()
	for i, exp := range utaKlineIntervals {
		got, err := utaKlineInterval(i, asset.Spot)
		require.NoErrorf(t, err, "utaKlineInterval must not error for %s", i)
		assert.Equalf(t, exp, got, "utaKlineInterval should name %s", i)
	}
	got, err := utaKlineInterval(kline.OneMonth, asset.Futures)
	require.NoError(t, err, "utaKlineInterval must not error for a futures month")
	assert.Equal(t, "1month", got, "utaKlineInterval should name a futures month")
	_, err = utaKlineInterval(kline.SixHour, asset.Futures)
	assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "utaKlineInterval should reject 6 hour futures klines")
	_, err = utaKlineInterval(kline.TenMin, asset.Spot)
	assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "utaKlineInterval should reject an interval KuCoin does not offer")
}

func TestUTAOrderbookDepth(t *testing.T) {
	t.Parallel()
	for levels, exp := range map[int]string{1: "1", 5: "5", 50: "50", 500: "increment@10ms"} {
		got, err := utaOrderbookDepth(levels)
		require.NoErrorf(t, err, "utaOrderbookDepth must not error for %d levels", levels)
		assert.Equalf(t, exp, got, "utaOrderbookDepth should select depth %s for %d levels", exp, levels)
	}
	_, err := utaOrderbookDepth(0)
	assert.ErrorIs(t, err, subscription.ErrInvalidLevel, "utaOrderbookDepth should reject levels KuCoin does not offer")
}

func TestUTASubscribe(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{reply: []byte(`{"id":"1545910660739","result":true}`)}
	subs := subscription.List{{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, QualifiedChannel: "ticker:BTC-USDT"}}
	require.NoError(t, ex.UTASubscribe(t.Context(), conn, subs), "UTASubscribe must not error")
	require.Len(t, conn.sent, 1, "UTASubscribe must send one message")
	req, ok := conn.sent[0].(*UTAWebsocketSubscriptionRequest)
	require.True(t, ok, "UTASubscribe must send a subscription request")
	assert.Equal(t, utaSubscribeAction, req.Action, "UTASubscribe should subscribe")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "UTASubscribe should mark the subscription subscribed")
}

func TestUTAUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{reply: []byte(`{"id":"1545910840805","result":"true"}`)}
	subs := subscription.List{{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, QualifiedChannel: "ticker:BTC-USDT"}}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.UTAUnsubscribe(t.Context(), conn, subs), "UTAUnsubscribe must not error")
	require.Len(t, conn.sent, 1, "UTAUnsubscribe must send one message")
	req, ok := conn.sent[0].(*UTAWebsocketSubscriptionRequest)
	require.True(t, ok, "UTAUnsubscribe must send a subscription request")
	assert.Equal(t, utaUnsubscribeAction, req.Action, "UTAUnsubscribe should unsubscribe")
	assert.Equal(t, subscription.UnsubscribedState, subs[0].State(), "UTAUnsubscribe should mark the subscription unsubscribed")
	assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "UTAUnsubscribe should remove the subscription")
}

func TestManageUTASubscriptions(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{reply: []byte(`{"id":"1545910660739","result":false,"message":"invalid request data"}`)}
	subs := subscription.List{
		{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, QualifiedChannel: "ticker:BTC-USDT"},
		{Channel: UTAMarkPriceChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, QualifiedChannel: "mark-price:XBTUSDTM"},
	}
	err := ex.manageUTASubscriptions(t.Context(), conn, subs, utaSubscribeAction)
	assert.ErrorIs(t, err, errUTARequestRejected, "manageUTASubscriptions should return the rejections")
	assert.Len(t, conn.sent, 2, "manageUTASubscriptions should send a message per subscription")
	assert.ErrorContains(t, err, "ticker:BTC-USDT", "manageUTASubscriptions should name the rejected ticker")
	assert.ErrorContains(t, err, "mark-price:XBTUSDTM", "manageUTASubscriptions should name the rejected mark price")
	assert.NoError(t, ex.manageUTASubscriptions(t.Context(), conn, nil, utaSubscribeAction), "manageUTASubscriptions should not error without subscriptions")
}

func TestManageUTASubscription(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	newSub := func() *subscription.Subscription {
		return &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, QualifiedChannel: "ticker:BTC-USDT"}
	}
	conn := &utaFakeConnection{reply: []byte(`{"id":"1545910660739","result":true}`)}
	s := newSub()
	require.NoError(t, ex.manageUTASubscription(t.Context(), conn, s, utaSubscribeAction), "manageUTASubscription must not error")
	assert.Equal(t, s, ex.Websocket.GetSubscription(s), "manageUTASubscription should store an accepted subscription")
	require.NoError(t, ex.manageUTASubscription(t.Context(), conn, s, utaUnsubscribeAction), "manageUTASubscription must not error unsubscribing")
	assert.Nil(t, ex.Websocket.GetSubscription(s), "manageUTASubscription should remove an unsubscribed subscription")

	for _, tc := range []struct {
		name  string
		conn  *utaFakeConnection
		sub   *subscription.Subscription
		expIs error
	}{
		{name: "an unknown channel", conn: conn, sub: &subscription.Subscription{Channel: "uta.unknown"}, expIs: subscription.ErrNotSupported},
		{name: "a send failure", conn: &utaFakeConnection{sendErr: websocket.ErrSignatureTimeout}, sub: newSub(), expIs: websocket.ErrSignatureTimeout},
		{name: "a rejection with a reason", conn: &utaFakeConnection{reply: []byte(`{"id":"bad","result":false,"reason":"topic Obu rpi_filter = 1 depth = 1 only support depth 5/50"}`)}, sub: newSub(), expIs: errUTARequestRejected},
	} {
		err := ex.manageUTASubscription(t.Context(), tc.conn, tc.sub, utaSubscribeAction)
		assert.ErrorIsf(t, err, tc.expIs, "manageUTASubscription should error for %s", tc.name)
	}
	assert.Error(t, ex.manageUTASubscription(t.Context(), &utaFakeConnection{reply: []byte(`{`)}, newSub(), utaSubscribeAction), "manageUTASubscription should error on a malformed reply")
	assert.ErrorIs(t, ex.manageUTASubscription(t.Context(), &utaFakeConnection{reply: []byte(`null`)}, newSub(), utaSubscribeAction), common.ErrNoResponse, "manageUTASubscription should error on a null reply")
	assert.Error(t, ex.manageUTASubscription(t.Context(), conn, newSub(), utaUnsubscribeAction), "manageUTASubscription should error unsubscribing from a subscription it does not hold")
}

func TestUTASubscriptionRequest(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp *UTAWebsocketSubscriptionRequest
	}{
		{
			sub: &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "ticker", TradeType: "SPOT", Symbol: "BTC-USDT"},
		},
		{
			sub: &subscription.Subscription{Channel: UTAKlineChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Interval: kline.FifteenMin},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "kline", TradeType: "FUTURES", Symbol: "XBTUSDTM", Interval: "15min"},
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{utaSpotTestPair}, Levels: 500},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "obu", TradeType: "SPOT", Symbol: "BTC-USDT", Depth: "increment@10ms"},
		},
		{
			sub: &subscription.Subscription{Channel: UTAMarkPriceChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "mark-price", Symbol: "XBTUSDTM"},
		},
		{
			sub: &subscription.Subscription{Channel: UTAAllFundingFeeRatesChannel, Asset: asset.Futures},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "funding-fee-all-symbols"},
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "order", TradeType: "UNIFIED", Symbol: "XBTUSDTM"},
		},
		{
			sub: &subscription.Subscription{Channel: UTABalanceChannel},
			exp: &UTAWebsocketSubscriptionRequest{Action: utaSubscribeAction, Channel: "balance", AccountType: "UNIFIED"},
		},
	} {
		req, err := ex.utaSubscriptionRequest(tc.sub, utaSubscribeAction)
		require.NoErrorf(t, err, "utaSubscriptionRequest must not error for %s", tc.sub.Channel)
		require.NotNilf(t, req, "utaSubscriptionRequest must return a request for %s", tc.sub.Channel)
		assert.Lenf(t, req.ID, 32, "utaSubscriptionRequest should give %s an ID", tc.sub.Channel)
		tc.exp.ID = req.ID
		assert.Equalf(t, tc.exp, req, "utaSubscriptionRequest should build the %s message", tc.sub.Channel)
	}

	rpi := map[string]any{"rpiFilter": true}
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
	}{
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 5, Params: rpi},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"5","rpiFilter":1}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 50, Params: rpi},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"50","rpiFilter":1}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 1, Params: rpi},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"1"}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 500, Params: rpi},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"increment@10ms"}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, Levels: 5, Params: rpi},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"SPOT","symbol":"BTC-USDT","depth":"5"}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 5},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"5"}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 5, Params: map[string]any{"rpiFilter": false}},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"5"}`,
		},
		{
			sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 50, Params: map[string]any{"rpiFilter": nil}},
			exp: `{"id":"","action":"SUBSCRIBE","channel":"obu","tradeType":"FUTURES","symbol":"XBTUSDTM","depth":"50"}`,
		},
	} {
		// An unsubscribe without the filter is acknowledged, but KuCoin keeps sending the RPI book
		for _, action := range []string{utaSubscribeAction, utaUnsubscribeAction} {
			req, err := ex.utaSubscriptionRequest(tc.sub, action)
			require.NoErrorf(t, err, "utaSubscriptionRequest must not error for %s depth %d", tc.sub.Asset, tc.sub.Levels)
			req.ID = ""
			msg, err := json.Marshal(req)
			require.NoError(t, err, "Marshal must not error")
			exp := strings.Replace(tc.exp, `"SUBSCRIBE"`, `"`+action+`"`, 1)
			assert.JSONEqf(t, exp, string(msg), "utaSubscriptionRequest should ask for RPI orders only on a futures orderbook of depth 5 or 50 whose params ask for them, %s %s depth %d params %v", action, tc.sub.Asset, tc.sub.Levels, tc.sub.Params)
		}
	}

	for _, tc := range []struct {
		name  string
		sub   *subscription.Subscription
		expIs error
	}{
		{name: "an unknown channel", sub: &subscription.Subscription{Channel: subscription.TickerChannel}, expIs: subscription.ErrNotSupported},
		{name: "several pairs", sub: &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair, currency.NewPairWithDelimiter("ETH", "USDT", "-")}}, expIs: subscription.ErrNotSinglePair},
		{name: "no levels", sub: &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}}, expIs: subscription.ErrInvalidLevel},
		{name: "an asset without pairs", sub: &subscription.Subscription{Channel: UTAOrderChannel, Asset: asset.Options, Pairs: currency.Pairs{utaSpotTestPair}}, expIs: asset.ErrNotSupported},
	} {
		_, err := ex.utaSubscriptionRequest(tc.sub, utaSubscribeAction)
		assert.ErrorIsf(t, err, tc.expIs, "utaSubscriptionRequest should error for %s", tc.name)
	}
}

func TestUTAMessageID(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	id := ex.utaMessageID()
	assert.Len(t, id, 32, "utaMessageID should fit the trade connection's 32 byte limit")
	assert.NotContains(t, id, "-", "utaMessageID should drop the UUID's dashes")
	assert.NotEqual(t, id, ex.utaMessageID(), "utaMessageID should differ on each call")
}

func TestUTAWebsocketAckResponseDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketAckResponse{
		{ID: "s1", Result: true},
		{ID: "1545910660739", Result: true},
		{ID: "bad1", Message: "invalid request data"},
		{ID: "bad", Reason: "topic Obu rpi_filter = 1 depth = 1 only support depth 5/50"},
	}
	got := make([]UTAWebsocketAckResponse, len(exp))
	for i, reply := range []string{
		`{"id":"s1","result":true}`,
		`{"id":"1545910660739","result":"true"}`,
		`{"id":"bad1","result":false,"message":"invalid request data"}`,
		`{"id":"bad","result":false,"reason":"topic Obu rpi_filter = 1 depth = 1 only support depth 5/50"}`,
	} {
		require.NoErrorf(t, json.Unmarshal([]byte(reply), &got[i]), "reply %d must decode", i)
	}
	assert.Equal(t, exp, got, "acknowledgements should decode a boolean or string result and either rejection field")
}

func TestUTAWebsocketMessageDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketMessage{
		{Message: "welcome"},
		{ID: "p1", Operation: "pong"},
		{
			Topic:         "obu.SPOT",
			PushTimestamp: types.Time(time.Unix(0, 1791901035798337832)),
			UpdateType:    "snapshot",
			Depth:         "5",
			Data:          json.RawMessage(`{"C":38627190018,"M":1791901035787000000,"O":38627190018,"a":[],"b":[],"s":"BTC-USDT"}`),
		},
	}
	got := make([]UTAWebsocketMessage, len(exp))
	for i, frame := range []string{
		`{"sessionId":"2742acdd-3c82-4d6e-bcf6-3f5c46906519","message":"welcome","pingInterval":18000}`,
		`{"op":"pong","id": "p1", "ts": 1791899342610902238}`,
		`{"T":"obu.SPOT", "dp":"5", "t":"snapshot", "P":1791901035798337832, "d":{"C":38627190018,"M":1791901035787000000,"O":38627190018,"a":[],"b":[],"s":"BTC-USDT"}}`,
	} {
		require.NoErrorf(t, json.Unmarshal([]byte(frame), &got[i]), "frame %d must decode", i)
	}
	assert.Equal(t, exp, got, "frames should decode the fields that route them, telling the topic and update type apart")
}

func TestWebsocketHandleUTAData(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{}
	require.NoError(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"sessionId":"81f56400-e7e3-4f2b-b0f6-5e649f48c302","message":"welcome","pingInterval":18000}`)), "wsHandleUTAData must ignore the welcome frame")
	require.NoError(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"op":"pong","id": "ping", "ts": 1791899342610902238}`)), "wsHandleUTAData must ignore a pong")
	assert.Empty(t, conn.matched, "wsHandleUTAData should not match the welcome frame or a pong to a request")

	require.NoError(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"id":"s1","result":true}`)), "wsHandleUTAData must route a reply")
	assert.Equal(t, []any{"s1"}, conn.matched, "wsHandleUTAData should match a reply by its ID")
	conn.matchErr = websocket.ErrSignatureNotMatched
	assert.ErrorIs(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"id":"s2","result":true}`)), websocket.ErrSignatureNotMatched, "wsHandleUTAData should return an unmatched reply's error")

	require.NoError(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"T":"funding-fee", "P":1791899460244168627, "d":{"s":"XBTUSDTM","fr":"0.000055","ft":1791878400000,"lfr":"0.000072","nt":1791907200000,"gl":28800000,"fc":"0.003","ff":"-0.003"}}`)), "wsHandleUTAData must route a push")
	assert.Len(t, drainUTARelay(ex), 1, "wsHandleUTAData should relay the push")

	assert.ErrorIs(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{"sessionId":"81f56400-e7e3-4f2b-b0f6-5e649f48c302"}`)), errUTAUnexpectedMessage, "wsHandleUTAData should error for a frame it cannot route")
	assert.Error(t, ex.wsHandleUTAData(t.Context(), conn, []byte(`{`)), "wsHandleUTAData should error for a malformed frame")
}

func TestProcessUTAPush(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	ex.SetTradeFeedStatus(true)
	ex.SetFillsFeedStatus(true)
	newUTAWebsocket(t, ex)
	for _, path := range []string{
		"testdata/wsUTATicker.json",
		"testdata/wsUTAKline.json",
		"testdata/wsUTATrade.json",
		"testdata/wsUTAOrderbook.json",
		"testdata/wsUTAMarkPrice.json",
		"testdata/wsUTAFundingFee.json",
		"testdata/wsUTACallAuctionInfo.json",
		"testdata/wsUTAOrder.json",
		"testdata/wsUTAExecution.json",
		"testdata/wsUTABalance.json",
		"testdata/wsUTAPosition.json",
		"testdata/wsUTALiquidationWarning.json",
		"testdata/wsUTALeverage.json",
	} {
		for i, line := range utaFixtureLines(t, path) {
			var msg UTAWebsocketMessage
			require.NoErrorf(t, json.Unmarshal([]byte(line), &msg), "%s line %d must decode", path, i+1)
			assert.NoErrorf(t, ex.processUTAPush(t.Context(), nil, &msg), "processUTAPush should route %s line %d", path, i+1)
		}
	}
	assert.NotEmpty(t, drainUTARelay(ex), "processUTAPush should relay the pushes")
	for _, topic := range []string{"unknown.SPOT", "ticker", "ticker.OPTIONS"} {
		msg := &UTAWebsocketMessage{Topic: topic, Data: []byte(`{"s":"BTC-USDT"}`)}
		assert.ErrorIsf(t, ex.processUTAPush(t.Context(), nil, msg), errUTAUnknownTopic, "processUTAPush should reject topic %q", topic)
	}
}

func TestUTAPairAndAssets(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	pair, assets, err := ex.utaPairAndAssets("BTC-USDT", "SPOT")
	require.NoError(t, err, "utaPairAndAssets must not error for an enabled spot pair")
	assert.Equal(t, utaSpotTestPair, pair, "utaPairAndAssets should split the spot symbol")
	assert.Equal(t, []asset.Item{asset.Spot, asset.Margin}, assets, "utaPairAndAssets should store a pair enabled on spot and margin under both")

	pair, assets, err = ex.utaPairAndAssets("GROVE-USDT", "SPOT")
	require.NoError(t, err, "utaPairAndAssets must not error for a pair enabled nowhere")
	assert.Equal(t, currency.NewPairWithDelimiter("GROVE", "USDT", "-"), pair, "utaPairAndAssets should split the spot symbol")
	assert.Equal(t, []asset.Item{asset.Spot}, assets, "utaPairAndAssets should store a pair enabled nowhere under spot")

	pair, assets, err = ex.utaPairAndAssets("XBTUSDTM", "FUTURES")
	require.NoError(t, err, "utaPairAndAssets must not error for a futures contract")
	assert.Equal(t, utaFuturesTestPair, pair, "utaPairAndAssets should match the futures contract")
	assert.Equal(t, []asset.Item{asset.Futures}, assets, "utaPairAndAssets should store a futures contract under futures")

	_, _, err = ex.utaPairAndAssets("NOTACONTRACTM", "FUTURES")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "utaPairAndAssets should error for an unknown futures contract")
	_, _, err = ex.utaPairAndAssets("", "SPOT")
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "utaPairAndAssets should error for an empty spot symbol")
	_, _, err = ex.utaPairAndAssets("BTC-USDT", "OPTIONS")
	assert.ErrorIs(t, err, errUTAUnknownTopic, "utaPairAndAssets should error for an unknown market")
}

// TestUTASpotPublicConnection subscribes to a spot ticker over the UTA spot public connection, against a mock server
// replaying a recorded push or, with -tags mock_test_off, against KuCoin
func TestUTASpotPublicConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	wsURL := kucoinUTASpotPublicWebsocketURL
	if mockTests {
		push := utaFixtureLines(t, "testdata/wsUTATicker.json")[0]
		wsURL = newUTAMockServer(t, true, func(msg map[string]any) []string {
			if msg["action"] == utaUnsubscribeAction {
				return []string{utaAck(msg)}
			}
			return []string{utaAck(msg), push}
		})
	}
	newUTAWebsocket(t, ex, ex.newUTASpotPublicConnection(ex.Config, wsURL))
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}}}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	tickers := awaitUTARelay[[]ticker.Price](t, ex, 30*time.Second)
	if mockTests {
		exp := []ticker.Price{
			{Last: 83399.7, LastSize: 0.02395178, Bid: 83403.3, BidSize: 0.68235477, Ask: 83403.4, AskSize: 0.53328968, Pair: utaSpotTestPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.Unix(0, 1791899343339000000)},
			{Last: 83399.7, LastSize: 0.02395178, Bid: 83403.3, BidSize: 0.68235477, Ask: 83403.4, AskSize: 0.53328968, Pair: utaSpotTestPair, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.Unix(0, 1791899343339000000)},
		}
		assert.Equal(t, exp, tickers, "the spot public connection should relay the stored tickers")
	} else {
		require.NotEmpty(t, tickers, "the spot public connection must relay tickers")
		assert.Positive(t, tickers[0].Bid, "the relayed ticker should carry a bid")
		assert.Positive(t, tickers[0].Ask, "the relayed ticker should carry an ask")
	}
	conn, err := ex.Websocket.GetConnection(utaSpotPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	subs := ex.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "the spot public connection must hold the ticker subscription")
	require.NoError(t, ex.UTAUnsubscribe(t.Context(), conn, subs), "UTAUnsubscribe must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "UTAUnsubscribe should remove the subscription")
}

// TestUTAFuturesPublicConnection subscribes to a futures ticker over the UTA futures public connection, against a
// mock server replaying a recorded push or, with -tags mock_test_off, against KuCoin
func TestUTAFuturesPublicConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	wsURL := kucoinUTAFuturesPublicWebsocketURL
	if mockTests {
		push := utaFixtureLines(t, "testdata/wsUTATicker.json")[1]
		wsURL = newUTAMockServer(t, true, func(msg map[string]any) []string { return []string{utaAck(msg), push} })
	}
	newUTAWebsocket(t, ex, ex.newUTAFuturesPublicConnection(ex.Config, wsURL))
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}}}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	// KuCoin pushes a futures ticker on a trade only, and quiet spells of half a minute occur
	tickers := awaitUTARelay[[]ticker.Price](t, ex, 2*time.Minute)
	if mockTests {
		exp := []ticker.Price{{Last: 83327, LastSize: 1, Bid: 83327, BidSize: 299, Ask: 83327.1, AskSize: 586, Pair: utaFuturesTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(0, 1791899402314000000)}}
		assert.Equal(t, exp, tickers, "the futures public connection should relay the stored ticker")
		return
	}
	require.NotEmpty(t, tickers, "the futures public connection must relay tickers")
	assert.Positive(t, tickers[0].Bid, "the relayed ticker should carry a bid")
}

// TestUTAPrivateConnection signs in and subscribes to the order channel over the UTA private connection, against a mock
// server replaying the docs' example push; with -tags mock_test_off it signs in to KuCoin when credentials are set
func TestUTAPrivateConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	wsURL := kucoinUTAPrivateWebsocketURL
	var mu sync.Mutex
	var received []map[string]any
	if mockTests {
		push := utaFixtureLines(t, "testdata/wsUTAOrder.json")[0]
		wsURL = newUTAMockServer(t, true, func(msg map[string]any) []string {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			if msg["op"] == "auth" {
				return []string{utaAck(msg)}
			}
			return []string{utaAck(msg), push}
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ex.SetCredentials(apiCredentials)
	}
	newUTAWebsocket(t, ex, ex.newUTAPrivateConnection(ex.Config, wsURL))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTAAllOrdersChannel}}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	if !mockTests {
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "the private connection should hold the order subscription")
		return
	}
	awaitUTARelay[*order.Detail](t, ex, 5*time.Second)
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 2, "the mock server must receive the authentication and subscribe messages")
	assert.Equal(t, "auth", received[0]["op"], "the private connection should sign in first")
	assert.Equal(t, "orderAll", received[1]["channel"], "the private connection should then subscribe")
	assert.Equal(t, "UNIFIED", received[1]["tradeType"], "the order subscription should name the unified account")
}
