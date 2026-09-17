package lbank

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testexchwebsocket "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	assert.NotEmpty(t, subs, "generateSubscriptions should return subscriptions")
}

func TestWebsocketSubscribeKeySuccess(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"result":true,"data":"4e9958623e6006bd","error_code":0,"ts":1705602277198}`,
		`{"result":"true","data":"4e9958623e6006bd","error_code":0,"ts":1705602277198}`,
	} {
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err, "GenerateKey must not error")
		ex.privateKey = pk
		require.NoError(t, ex.SetHTTPClient(&http.Client{Transport: &fakeRoundTripper{body: body}}), "SetHTTPClient must not error")
		key, err := ex.GetWebsocketSubscribeKey(t.Context())
		require.NoErrorf(t, err, "GetWebsocketSubscribeKey must accept %s", body)
		assert.Equalf(t, "4e9958623e6006bd", key, "GetWebsocketSubscribeKey should return data from %s", body)
		assert.NoErrorf(t, ex.RefreshWebsocketSubscribeKey(t.Context(), key), "RefreshWebsocketSubscribeKey should accept %s", body)
		assert.NoErrorf(t, ex.DestroyWebsocketSubscribeKey(t.Context(), key), "DestroyWebsocketSubscribeKey should accept %s", body)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{testPair}},
	}

	require.NoError(t, ex.Subscribe(subs), "Subscribe must not error")
	require.Len(t, conn.sentMessages, 1, "Subscribe must send one message")
	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, lbankWsSubscribe, req[lbankWsAction], "Subscribe should send subscribe action")

	require.NoError(t, ex.Unsubscribe(subs), "Unsubscribe must not error")
	require.Len(t, conn.sentMessages, 2, "Unsubscribe must send one more message")
	req, ok = conn.sentMessages[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, lbankWsUnsubscribe, req[lbankWsAction], "Unsubscribe should send unsubscribe action")
}

func TestManageSubsMockWsInstance(t *testing.T) {
	t.Parallel()

	handler := &lbankWsMockHandler{}
	ex := testexch.MockWsInstance[Exchange](t, testexchwebsocket.CurryWsMockUpgrader(t, handler.handle))

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{testPair}},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: currency.Pairs{testPair}, Levels: 100},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin, Pairs: currency.Pairs{testPair}},
	}

	require.NoError(t, ex.Subscribe(subs), "Subscribe must not error")

	waitForCondition(t, 2*time.Second, func() bool {
		return len(handler.captured()) == len(subs)
	})

	msgs := handler.captured()
	require.Len(t, msgs, 3, "manageSubs must send one message per subscription")

	var tickerReq, depthReq, kbarReq map[string]any
	for _, raw := range msgs {
		var req map[string]any
		require.NoError(t, json.Unmarshal(raw, &req), "sent message must be valid JSON")
		switch req["subscribe"] {
		case lbankWsTicker:
			tickerReq = req
		case lbankWsOrderbook:
			depthReq = req
		case lbankWsKbar:
			kbarReq = req
		}
	}

	require.NotNil(t, tickerReq, "ticker subscription must have been sent")
	assert.Equal(t, lbankWsSubscribe, tickerReq[lbankWsAction])
	assert.Equal(t, testPair.Lower().String(), tickerReq["pair"])

	require.NotNil(t, depthReq, "orderbook subscription must have been sent")
	assert.Equal(t, "100", depthReq["depth"])

	require.NotNil(t, kbarReq, "candles subscription must have been sent")
	assert.Equal(t, lbankWsKline1Min, kbarReq["kbar"])

	require.NoError(t, ex.Unsubscribe(subs), "Unsubscribe must not error")

	waitForCondition(t, 2*time.Second, func() bool {
		return len(handler.captured()) == len(subs)*2
	})

	msgs = handler.captured()
	require.Len(t, msgs, 6, "Unsubscribe must send one more message per subscription")

	var req map[string]any
	require.NoError(t, json.Unmarshal(msgs[len(msgs)-1], &req), "last sent message must be valid JSON")
	assert.Equal(t, lbankWsUnsubscribe, req[lbankWsAction], "last message should be an unsubscribe action")
}
