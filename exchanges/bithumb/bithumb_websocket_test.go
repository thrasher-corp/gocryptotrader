package bithumb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

var (
	wsTickerResp    = []byte(`{"type":"ticker","content":{"tickType":"24H","date":"20210811","time":"132017","openPrice":"33400","closePrice":"34010","lowPrice":"32660","highPrice":"34510","value":"45741663716.89916828275244531","volume":"1359398.496892086826189907","sellVolume":"198021.237915860451480504","buyVolume":"1161377.258976226374709403","prevClosePrice":"33530","chgRate":"1.83","chgAmt":"610","volumePower":"500","symbol":"UNI_KRW"}}`)
	wsTransResp     = []byte(`{"type":"transaction","content":{"list":[{"buySellGb":"1","contPrice":"1166","contQty":"125.2400","contAmt":"146029.8400","contDtm":"2021-08-13 15:23:42.911273","updn":"dn","symbol":"DAI_KRW"}]}}`)
	wsOrderbookResp = []byte(`{"type":"orderbookdepth","content":{"list":[{"symbol":"XLM_KRW","orderType":"ask","price":"401.2","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"ask","price":"401.6","quantity":"21277.735","total":"1"},{"symbol":"XLM_KRW","orderType":"ask","price":"403.3","quantity":"4000","total":"1"},{"symbol":"XLM_KRW","orderType":"bid","price":"399.5","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"bid","price":"398.2","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"bid","price":"399.8","quantity":"31416.8779","total":"1"},{"symbol":"XLM_KRW","orderType":"bid","price":"398.5","quantity":"34328.387","total":"1"}],"datetime":"1628835823604483"}}`)
)

func TestWsHandleData(t *testing.T) {
	t.Parallel()

	pairs := currency.Pairs{currency.NewBTCUSDT()}

	dummy := Exchange{
		location: time.Local,
		Name:     "dummy",
		Features: exchange.Features{
			Enabled: exchange.FeaturesEnabled{SaveTradeData: true},
		},
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					Available: pairs,
					Enabled:   pairs,
					ConfigFormat: &currency.PairFormat{
						Uppercase: true,
						Delimiter: currency.DashDelimiter,
					},
				},
			},
		},
		Websocket: websocket.NewManager(),
	}

	dummy.setupOrderbookManager(t.Context())
	dummy.API.Endpoints = e.NewEndpoints()

	welcomeMsg := []byte(`{"status":"0000","resmsg":"Connected Successfully"}`)
	err := dummy.wsHandleData(t.Context(), nil, welcomeMsg)
	require.NoError(t, err, "wsHandleData must not error for the welcome message")

	err = dummy.wsHandleData(t.Context(), nil, []byte(`{"status":"1336","resmsg":"Failed"}`))
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsHandleData must return a subscription failure for a failed status")

	err = dummy.wsHandleData(t.Context(), nil, wsTransResp)
	require.NoError(t, err, "wsHandleData must not error for a transaction")

	err = dummy.wsHandleData(t.Context(), nil, wsOrderbookResp)
	require.NoError(t, err, "wsHandleData must not error for an orderbook")

	err = dummy.wsHandleData(t.Context(), nil, wsTickerResp)
	require.NoError(t, err, "wsHandleData must not error for a ticker")
	tick, ok := (<-dummy.Websocket.DataHandler.C).Data.(*ticker.Price)
	require.True(t, ok, "ticker must send a price to the DataHandler")
	assert.Equal(t, 34010.0, tick.Last, "Last should be the latest close rather than the previous day's")
}

func TestSubToReq(t *testing.T) {
	t.Parallel()
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "KRW", "_"), currency.NewPairWithDelimiter("ETH", "KRW", "_")}
	r, err := subToReq(&subscription.Subscription{Channel: subscription.AllTradesChannel}, p)
	require.NoError(t, err, "trade request must be supported")
	assert.Equal(t, "transaction", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	r, err = subToReq(&subscription.Subscription{Channel: subscription.OrderbookChannel}, p)
	require.NoError(t, err, "orderbook request must be supported")
	assert.Equal(t, "orderbookdepth", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	r, err = subToReq(&subscription.Subscription{Channel: subscription.TickerChannel, Interval: kline.OneHour}, p)
	require.NoError(t, err, "ticker request must be supported")
	assert.Equal(t, "ticker", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	assert.Equal(t, []string{"1H"}, r.TickTypes)
	t.Run("unsupported channel", func(t *testing.T) {
		t.Parallel()
		_, err := subToReq(&subscription.Subscription{Channel: subscription.MyTradesChannel}, p)
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "unsupported channel should return its sentinel")
	})
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "KRW", "_"), currency.NewPairWithDelimiter("ETH", "KRW", "_")}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, true))
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{
		{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: p, QualifiedChannel: `{"type":"transaction","symbols":["BTC_KRW","ETH_KRW"]}`},
		{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: p, QualifiedChannel: `{"type":"orderbookdepth","symbols":["BTC_KRW","ETH_KRW"]}`},
		{
			Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: p, Interval: kline.ThirtyMin,
			QualifiedChannel: `{"type":"ticker","symbols":["BTC_KRW","ETH_KRW"],"tickTypes":["30M"]}`,
		},
	}
	testsubs.EqualLists(t, exp, subs)
	t.Run("unsupported channel", func(t *testing.T) {
		t.Parallel()
		_, err := subscription.List{{Channel: subscription.MyTradesChannel, Asset: asset.Spot, Pairs: p}}.ExpandTemplates(e)
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "template expansion should preserve the unsupported channel error")
	})
}

type migrationTestConnection struct {
	websocket.Connection
	err     error
	payload []byte
}

func (c *migrationTestConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return c.err
}

func (c *migrationTestConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.payload, _ = json.Marshal(payload)
	return c.err
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "send failure"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must succeed")
			base, err := ex.Websocket.CreateTestConnection(asset.Spot)
			require.NoError(t, err, "managed connection must be configured")
			conn := &migrationTestConnection{Connection: base}
			require.NoError(t, ex.Websocket.TrackTestConnection(asset.Spot, conn), "connection must be tracked")
			if fail {
				conn.err = errors.New("send failed")
			}
			sub := &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: `{"type":"ticker","symbols":["BTC_KRW"],"tickTypes":["30M"]}`}
			err = ex.Subscribe(t.Context(), conn, subscription.List{sub})
			assert.JSONEq(t, sub.QualifiedChannel, string(conn.payload), "wire request should retain the qualified payload")
			if fail {
				assert.ErrorIs(t, err, conn.err, "send failure should propagate")
				assert.Nil(t, ex.Websocket.GetSubscription(sub), "failed subscription should not be registered")
			} else {
				require.NoError(t, err, "subscription must succeed")
				assert.Same(t, sub, ex.Websocket.GetSubscription(sub), "successful subscription should be registered")
				assert.Equal(t, subscription.SubscribedState, sub.State(), "subscription should be active")
			}
		})
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	t.Run("disabled exchange", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		ex.SetEnabled(false)
		assert.ErrorIs(t, ex.WsConnect(), websocket.ErrWebsocketNotEnabled, "disabled exchange should not start a connection")
	})
	t.Run("disabled websocket", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		if ex.Websocket.IsEnabled() {
			require.NoError(t, ex.Websocket.Disable(), "websocket must be disabled")
		}
		assert.ErrorIs(t, ex.WsConnect(), websocket.ErrWebsocketNotEnabled, "disabled websocket should not start a connection")
	})
	t.Run("dial failure", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		errDial := errors.New("dial failed")
		err := ex.wsConnect(t.Context(), &migrationTestConnection{err: errDial})
		assert.ErrorIs(t, err, errDial, "dial error should propagate")
		assert.Nil(t, ex.obm.jobs, "failed dial should not start orderbook workers")
	})

	t.Run("managed reconnect", func(t *testing.T) {
		t.Parallel()
		var connections atomic.Int32
		requests := make(chan []byte, 2)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
			if !assert.NoError(t, err, "server should upgrade the connection") {
				return
			}
			defer conn.Close()
			connections.Add(1)
			for {
				_, payload, err := conn.ReadMessage()
				if err != nil {
					return
				}
				requests <- payload
				if err := conn.WriteMessage(gws.TextMessage, wsTickerResp); err != nil {
					return
				}
			}
		}))
		t.Cleanup(server.Close)
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
		ex.Config.API.Endpoints[exchange.WebsocketSpot.String()] = websocketURL
		ex.Config.Features.Enabled.Websocket = true
		ex.Websocket = websocket.NewManager()
		require.NoError(t, ex.Setup(ex.Config), "Setup must retain the custom endpoint")
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.ThirtyMin}}
		t.Cleanup(func() { _ = ex.Websocket.Disable(); _ = ex.Websocket.Shutdown() })
		for i := range 2 {
			require.NoError(t, ex.WsConnect(), "managed connection must connect")
			conn, err := ex.Websocket.GetConnection(asset.Spot)
			require.NoError(t, err, "spot route must exist")
			assert.Equal(t, websocketURL, conn.GetURL(), "managed route should use configured endpoint")
			select {
			case payload := <-requests:
				var subscribeRequest WsSubscribe
				require.NoError(t, json.Unmarshal(payload, &subscribeRequest), "subscription request must decode")
				assert.Equal(t, "ticker", subscribeRequest.Type, "request should preserve the ticker channel")
				assert.Equal(t, []string{"30M"}, subscribeRequest.TickTypes, "request should preserve the interval")
			case <-time.After(5 * time.Second):
				t.Fatal("subscription must reach server")
			}
			select {
			case msg := <-ex.Websocket.DataHandler.C:
				assert.IsType(t, &ticker.Price{}, msg.Data, "manager reader should deliver the ticker")
			case <-time.After(5 * time.Second):
				t.Fatal("manager reader must deliver data")
			}
			require.NoError(t, ex.Websocket.Shutdown(), "shutdown must stop the reader and orderbook workers")
			assert.EqualValues(t, i+1, connections.Load(), "each lifecycle should establish one connection")
		}
	})
}
