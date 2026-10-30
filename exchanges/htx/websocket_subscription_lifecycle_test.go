package htx

import (
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestWebsocketSubscriptionLifecycle(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
	h.Name = t.Name()
	h.API.AuthenticatedWebsocketSupport = false
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.Spot, true), "test must enable only its fixture pair")
	h.Features.Subscriptions = subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 6},
	}
	releaseRejection := make(chan bool)
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(releaseRejection) }) })
	var requests atomic.Int32
	var writers sync.WaitGroup
	var writeMu sync.Mutex
	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, func(tb testing.TB, raw []byte, conn *gws.Conn) error {
		tb.Helper()
		requests.Add(1)
		channel, _ := jsonparser.GetString(raw, "sub")
		id, _ := jsonparser.GetString(raw, "id")
		if strings.Contains(channel, ".depth.") {
			writers.Go(func() {
				select {
				case <-releaseRejection:
				case <-t.Context().Done():
					return
				}
				writeMu.Lock()
				defer writeMu.Unlock()
				assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"error","err-code":"bad-topic","err-msg":"invalid depth"}`)), "rejected acknowledgement should write")
			})
			return nil
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := wsFixture(tb, raw, conn); err != nil {
			return err
		}
		if channel != "" {
			return conn.WriteMessage(gws.TextMessage, []byte(`{"ch":"market.btcusdt.detail","tick":{"close":100}}`))
		}
		return nil
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseRejection) })
		writers.Wait()
		if err := h.Websocket.Disable(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "cleanup should disable reconnection")
		}
		if err := h.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "cleanup should close owned connections")
		}
	})
	require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "connections must use the mock server")
	connected := make(chan error, 1)
	go func() { connected <- h.Websocket.Connect(t.Context()) }()
	select {
	case message := <-h.Websocket.DataHandler.C:
		if handlerErr, ok := message.Data.(error); ok {
			require.NoError(t, handlerErr, "reader must accept the immediate push")
		}
		require.IsType(t, &ticker.Price{}, message.Data, "immediate push must reach the handler before the batch finishes")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "immediate ticker must arrive while another subscription is pending")
	}
	select {
	case err := <-connected:
		require.FailNowf(t, "batch must still await its second acknowledgement", "Connect returned %v", err)
	default:
	}
	releaseOnce.Do(func() { close(releaseRejection) })
	select {
	case err := <-connected:
		require.ErrorIs(t, err, errAPIResponse, "partial rejection must remain visible")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "partial batch must finish after its last acknowledgement")
	}
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "accepted connection must remain available")
	require.Len(t, h.Websocket.GetSubscriptions(), 1, "only the accepted subscription must remain registered")
	require.Len(t, conn.Subscriptions().List(), 1, "manager must retain accepted connection ownership after a partial failure")
	previous := h.Websocket.GetSubscriptions()[0]
	h.Features.Subscriptions = h.Features.Subscriptions[:1]
	before := requests.Load()
	require.NoError(t, h.Websocket.FlushChannels(t.Context()), "equivalent refreshed subscriptions must remain usable")
	assert.Equal(t, before, requests.Load(), "equivalent refreshed settings should cause no subscribe or unsubscribe traffic")
	assert.Same(t, previous, h.Websocket.GetSubscriptions()[0], "refresh should retain the registered subscription")

	h.Features.Subscriptions = nil
	require.NoError(t, h.Websocket.FlushChannels(t.Context()), "removing subscriptions must unsubscribe the correct connection")
	assert.Empty(t, h.Websocket.GetSubscriptions(), "unsubscribe should clear manager registration")
	assert.Empty(t, conn.Subscriptions().List(), "unsubscribe should clear connection ownership")

	h.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	require.NoError(t, h.Websocket.FlushChannels(t.Context()), "restoring subscriptions must create a usable connection")
	reconnected, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "restored connection must be available")
	assert.NotSame(t, conn, reconnected, "restoring an emptied setup should use a new connection")
	assert.Same(t, h.Websocket.GetSubscriptions()[0], h.Websocket.GetSubscription(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "market.btcusdt.detail"}}, connection: reconnected}), "restored routing should use the new connection identity")
	assert.Nil(t, h.Websocket.GetSubscription(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "market.btcusdt.detail"}}, connection: conn}), "retired connection identity should not find the replacement subscription")
}

func TestWebsocketTradeConnectionLifecycle(t *testing.T) {
	t.Parallel()
	var tradeAuthentications atomic.Int32
	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
		tb.Helper()
		type requestPayload struct {
			Operation string `json:"op"`
		}
		var request requestPayload
		if err := json.Unmarshal(message, &request); err != nil {
			return err
		}
		if request.Operation == wsAuthChannel {
			tradeAuthentications.Add(1)
		}
		return wsFixture(tb, message, conn)
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "exchange defaults must load")
	cfg := h.Config
	cfg.API.AuthenticatedWebsocketSupport = true
	cfg.API.AuthenticatedSupport = true
	h.SetDefaults()
	require.NoError(t, h.Setup(cfg), "authenticated websocket setup must succeed")
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	h.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "USDT-margined asset must start disabled")
	require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "websocket endpoints must use the mock server")
	t.Cleanup(func() {
		if err := h.Websocket.Disable(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should not error")
		}
		if err := h.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error")
		}
	})
	require.NoError(t, h.Websocket.Connect(t.Context()), "spot must connect with the trade asset disabled")
	spot, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot connection must be available")
	_, err = h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "disabled trade asset must not create a connection")
	assert.Zero(t, tradeAuthentications.Load(), "disabled trade asset should not authenticate a socket")

	require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, true), "USDT-margined asset must enable")
	require.NoError(t, h.Websocket.FlushChannels(t.Context()), "enabling trade asset must create its connection")
	trade, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.NoError(t, err, "trade connection must be available after enabling the asset")
	assert.Equal(t, int32(1), tradeAuthentications.Load(), "trade socket should authenticate before use")
	spotAfterEnable, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot connection must remain available")
	assert.Same(t, spot, spotAfterEnable, "enabling trade should retain the spot connection")

	require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "USDT-margined asset must disable")
	require.NoError(t, h.Websocket.FlushChannels(t.Context()), "disabling trade asset must remove its connection")
	_, err = h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "disabled trade connection must be unavailable")
	assert.ErrorContains(t, trade.SendJSONMessage(t.Context(), 0, &V5WebsocketTradeRequest{}), "disconnected", "disabled trade connection should close")
	spotAfterDisable, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot connection must remain available after disabling trade")
	assert.Same(t, spot, spotAfterDisable, "disabling trade should retain the spot connection")
}
