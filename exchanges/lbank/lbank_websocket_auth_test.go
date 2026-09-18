package lbank

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type pathRoundTripper struct {
	responses map[string]string // URL path substring -> response body
}

func (p *pathRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for path, body := range p.responses {
		if strings.Contains(req.URL.Path, path) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Header:     make(http.Header),
	}, nil
}

func TestDoRefreshSubscribeKeySuccess(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "old-key"
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{body: `{"result":"true","error_code":0}`},
	}), "SetHTTPClient must not error")

	ex.doRefreshSubscribeKey(t.Context())

	ex.ws.mu.RLock()
	defer ex.ws.mu.RUnlock()
	assert.Equal(t, "old-key", ex.ws.subscribeKey, "key should be unchanged after a successful refresh")
}

func TestDoRefreshSubscribeKeyRefreshAndFetchBothFail(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "old-key"
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	// RefreshWebsocketSubscribeKey and GetWebsocketSubscribeKey both hit the same
	// fake client here; both now key off error_code != 0 via ErrCapture, so a
	// nonzero error_code makes both the refresh and the fallback fetch fail.
	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{body: `{"result":"false","error_code":10005,"msg":"key expired"}`},
	}), "SetHTTPClient must not error")

	ex.doRefreshSubscribeKey(t.Context())
	// Since both calls use the same failing mock, GetWebsocketSubscribeKey also fails here —
	// this exercises the "refresh failed, then fetch also failed" path, logging both errors
	// and leaving the key unchanged.
	ex.ws.mu.RLock()
	defer ex.ws.mu.RUnlock()
	assert.Equal(t, "old-key", ex.ws.subscribeKey, "key should be unchanged when both refresh and re-fetch fail")
}

func TestWsRefreshSubscribeKeyLoopExitsOnContextCancel(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	old := wsRefreshInterval
	wsRefreshInterval = time.Millisecond
	t.Cleanup(func() { wsRefreshInterval = old })

	ctx, cancel := context.WithCancel(t.Context())
	ex.Websocket.Wg.Add(1)
	done := make(chan struct{})
	go func() {
		ex.wsRefreshSubscribeKey(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsRefreshSubscribeKey did not exit after context cancellation")
	}
}

func TestWsRefreshSubscribeKeyLoopExitsOnShutdown(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	old := wsRefreshInterval
	wsRefreshInterval = time.Hour // long enough that the ticker won't fire during this test
	t.Cleanup(func() { wsRefreshInterval = old })

	ex.Websocket.Wg.Add(1)
	done := make(chan struct{})
	go func() {
		ex.wsRefreshSubscribeKey(t.Context())
		close(done)
	}()

	close(ex.Websocket.ShutdownC)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsRefreshSubscribeKey did not exit after shutdown")
	}
}

func TestWsRefreshSubscribeKeyTickerFires(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "old-key"
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{body: `{"result":"true","error_code":0}`},
	}), "SetHTTPClient must not error")

	old := wsRefreshInterval
	wsRefreshInterval = 10 * time.Millisecond
	t.Cleanup(func() { wsRefreshInterval = old })

	ctx, cancel := context.WithCancel(t.Context())
	ex.Websocket.Wg.Add(1)
	done := make(chan struct{})
	go func() {
		ex.wsRefreshSubscribeKey(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // allow at least one tick to fire
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsRefreshSubscribeKey did not exit after context cancellation")
	}
}

func TestDoRefreshSubscribeKeyResubscribesOnNewKey(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "old-key"
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &pathRoundTripper{responses: map[string]string{
			lbankSubscribeRefreshKey: `{"result":"false","error_code":10005,"msg":"key expired"}`,
			lbankSubscribeGetKey:     `{"result":"true","data":"new-key","error_code":0}`,
		}},
	}), "SetHTTPClient must not error")

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	// GetSubscriptions() needs at least one authenticated subscription tracked
	// for doRefreshSubscribeKey's re-subscribe branch to have anything to act on.
	// Subscribe (not AddSuccessfulSubscriptions) registers it via the normal path,
	// so doRefreshSubscribeKey's later Subscribe call re-subscribes cleanly instead
	// of hitting a duplicate-registration error against the same tracked entry.
	require.NoError(t, ex.Subscribe(subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}), "initial Subscribe must not error")

	ex.doRefreshSubscribeKey(t.Context())

	ex.ws.mu.RLock()
	assert.Equal(t, "new-key", ex.ws.subscribeKey, "key should be updated")
	ex.ws.mu.RUnlock()

	// The initial Subscribe above sends the first message; doRefreshSubscribeKey's
	// re-subscribe sends the second, now carrying the new key.
	require.Len(t, conn.sentMessages, 2, "initial Subscribe plus doRefreshSubscribeKey's re-subscribe must send two messages")
	req, ok := conn.sentMessages[1].(map[string]any)
	require.True(t, ok, "second sent message must be a map[string]any")
	assert.Equal(t, lbankWsSubscribe, req[lbankWsAction])
	assert.Equal(t, lbankWsOrderUpdate, req["subscribe"])
	assert.Equal(t, "new-key", req["subscribeKey"], "re-subscribe should use the new key")
}

func TestDoRefreshSubscribeKeyResubscribeFailureIsLogged(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "old-key"
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &pathRoundTripper{responses: map[string]string{
			lbankSubscribeRefreshKey: `{"result":"false","error_code":10005,"msg":"key expired"}`,
			lbankSubscribeGetKey:     `{"result":"true","data":"new-key","error_code":0}`,
		}},
	}), "SetHTTPClient must not error")

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn
	require.NoError(t, ex.Subscribe(subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}), "initial Subscribe must not error")

	// Now make the connection fail sends, so doRefreshSubscribeKey's re-subscribe fails
	conn.sendErr = errors.New("send failed")

	// Should not panic; re-subscribe failure is only logged, not returned or observable
	// via a return value. This exercises the error branch inside the re-subscribe block.
	ex.doRefreshSubscribeKey(t.Context())

	ex.ws.mu.RLock()
	assert.Equal(t, "new-key", ex.ws.subscribeKey, "key should still be updated even if re-subscribe fails")
	ex.ws.mu.RUnlock()
}

func TestWsConnectAuthKeyFetchFailure(t *testing.T) {
	t.Parallel()

	ex := setupWithWebsocketEnabled(t)
	ex.Name = t.Name()
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{err: errors.New("network unreachable")},
	}), "SetHTTPClient must not error")

	conn := &wsConnectFixtureConnection{}
	ex.Websocket.Conn = conn

	err = ex.WsConnect()
	require.NoError(t, err, "WsConnect must not error even when auth key fetch fails")

	waitForWaitGroup(t, &ex.Websocket.Wg, 2*time.Second)

	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints should remain disabled when key fetch fails")
	assert.Equal(t, int32(1), conn.dialCalls.Load(), "conn should still dial once")
	assert.Equal(t, int32(1), conn.readCalls.Load(), "reader should still run once")
}

func TestWsConnectLoadPrivKeyFailure(t *testing.T) {
	t.Parallel()

	ex := setupWithWebsocketEnabled(t)
	ex.Name = t.Name()
	ex.API.AuthenticatedWebsocketSupport = true
	// No credentials set, no privateKey pre-seeded — loadPrivKey should fail

	conn := &wsConnectFixtureConnection{}
	ex.Websocket.Conn = conn

	err := ex.WsConnect()
	require.NoError(t, err, "WsConnect must not error even when loadPrivKey fails")

	waitForWaitGroup(t, &ex.Websocket.Wg, 2*time.Second)

	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints should remain disabled when loadPrivKey fails")
	assert.Equal(t, int32(1), conn.dialCalls.Load())
	assert.Equal(t, int32(1), conn.readCalls.Load())
}

func TestWsConnectAuthSuccess(t *testing.T) {
	t.Parallel()

	ex := setupWithWebsocketEnabled(t)
	ex.Name = t.Name()
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{body: `{"result":"true","data":"fake-subscribe-key","error_code":0}`},
	}), "SetHTTPClient must not error")

	conn := &wsConnectFixtureConnection{}
	ex.Websocket.Conn = conn
	t.Cleanup(func() { close(ex.Websocket.ShutdownC); waitForWaitGroup(t, &ex.Websocket.Wg, 2*time.Second) })

	err = ex.WsConnect()
	require.NoError(t, err, "WsConnect must not error")

	waitForCondition(t, 2*time.Second, func() bool {
		return conn.readCalls.Load() > 0
	})

	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints should be enabled after successful key fetch")
	assert.Equal(t, int32(1), conn.dialCalls.Load(), "conn should dial once")
	assert.Equal(t, int32(1), conn.readCalls.Load(), "reader should run once")

	ex.ws.mu.RLock()
	defer ex.ws.mu.RUnlock()
	assert.Equal(t, "fake-subscribe-key", ex.ws.subscribeKey, "subscribe key should match the fetched key")
}

func TestWsConnectNotEnabled(t *testing.T) {
	t.Parallel()
	ex := setupWithWebsocketEnabled(t)
	ex.Name = t.Name()
	ex.SetEnabled(false)

	err := ex.WsConnect()
	assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled, "WsConnect should return ErrWebsocketNotEnabled when exchange is disabled")
}

func TestWsConnectVerboseLogging(t *testing.T) {
	t.Parallel()
	ex := setupWithWebsocketEnabled(t)
	ex.Name = t.Name()
	ex.Verbose = true

	conn := &wsConnectFixtureConnection{}
	ex.Websocket.Conn = conn

	err := ex.WsConnect()
	require.NoError(t, err, "WsConnect must not error")

	waitForCondition(t, 2*time.Second, func() bool {
		return conn.readCalls.Load() > 0
	})
}
