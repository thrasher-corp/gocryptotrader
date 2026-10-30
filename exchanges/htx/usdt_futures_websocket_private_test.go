package htx

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newV5TradeWebsocketTestExchange uses the production registration and authentication flow before testing trade requests.
func newV5TradeWebsocketTestExchange(t *testing.T, handler mockws.WsMockFunc) *Exchange {
	t.Helper()
	var authenticated atomic.Bool
	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
		tb.Helper()
		operation, _ := jsonparser.GetString(message, "op")
		if operation == wsAuthChannel {
			// Connect can return as soon as the client reads the reply.
			authenticated.Store(true)
			return wsFixture(tb, message, conn)
		}
		assert.True(tb, authenticated.Load(), "trade requests should follow authentication")
		return handler(tb, message, conn)
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
	h.Features.Subscriptions = nil
	require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "websocket endpoints must use the mock server")
	t.Cleanup(func() {
		if err := h.Websocket.Disable(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should not error")
		}
		if err := h.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error")
		}
	})
	require.NoError(t, h.Websocket.Connect(t.Context()), "trade connection must authenticate and connect")
	require.True(t, authenticated.Load(), "trade connection must authenticate before use")
	return h
}

func TestNewV5TradeWebsocketTestExchange(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
		tb.Helper()
		requests.Add(1)
		return wsFixture(tb, message, conn)
	})
	conn, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.NoError(t, err, "production trade connection must be available")
	subs := conn.Subscriptions().List()
	require.Len(t, subs, 1, "production trade connection must own its logical subscription")
	assert.Equal(t, "trade", subs[0].Channel, "logical subscription should identify the trade connection")
	assert.Equal(t, asset.USDTMarginedFutures, subs[0].Asset, "trade connection should belong to the USDT-margined asset")
	assert.True(t, subs[0].Authenticated, "logical subscription should require authentication")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "trade connection should finish production registration")
	assert.Zero(t, requests.Load(), "logical registration should send no subscription request to HTX")
	resp, err := h.WebsocketPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT"})
	require.NoError(t, err, "authenticated trade connection must complete an order request")
	require.NotNil(t, resp, "order acknowledgement must be returned")
	assert.Equal(t, "1", resp.Acknowledgement.OrderID, "order acknowledgement should decode")
	assert.Equal(t, int32(1), requests.Load(), "order request should use the production trade connection")
}

func TestSendV5TradeRequest(t *testing.T) {
	t.Parallel()
	t.Run("authentication required on connected socket", func(t *testing.T) {
		t.Parallel()
		h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
			tb.Helper()
			assert.Fail(tb, "unauthenticated trade request should not reach the socket")
			return wsFixture(tb, message, conn)
		})
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		var response *V5WebsocketOrderResponse
		err := h.sendV5TradeRequest(t.Context(), "place_order", &V5OrderRequest{ContractCode: "BTC-USDT"}, &response)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "trade request must require authentication even when connected")
		assert.Nil(t, response, "unauthenticated trade request should return no response")
	})
	type testCase struct {
		name        string
		setup       func(*testing.T) *Exchange
		expectedErr bool
		cancel      bool
	}
	for _, tc := range []testCase{
		{
			name: "success",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return newV5TradeWebsocketTestExchange(t, wsFixture)
			},
		},
		{
			name: "missing connection",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				h := new(Exchange)
				require.NoError(t, testexch.Setup(h), "HTX setup must not error")
				h.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return h
			},
			expectedErr: true,
		},
		{
			name: "cancelled context",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return newV5TradeWebsocketTestExchange(t, wsFixture)
			},
			expectedErr: true,
			cancel:      true,
		},
		{
			name: "exchange error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				fixture := func(tb testing.TB, message []byte, conn *gws.Conn) error {
					tb.Helper()
					cid, err := jsonparser.GetString(message, "cid")
					if err != nil {
						return err
					}
					return conn.WriteMessage(gws.TextMessage, []byte(`{"code":400,"message":"invalid request","cid":"`+cid+`"}`))
				}
				return newV5TradeWebsocketTestExchange(t, fixture)
			},
			expectedErr: true,
		},
		{
			name: "send error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				h := newV5TradeWebsocketTestExchange(t, wsFixture)
				conn, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
				require.NoError(t, err, "trade connection must be available")
				require.NoError(t, conn.Shutdown(), "trade connection must shut down")
				return h
			},
			expectedErr: true,
		},
		{
			name: "decode error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				fixture := func(tb testing.TB, message []byte, conn *gws.Conn) error {
					tb.Helper()
					cid, err := jsonparser.GetString(message, "cid")
					if err != nil {
						return err
					}
					return conn.WriteMessage(gws.TextMessage, []byte(`{"code":200,"cid":"`+cid+`","data":[]}`))
				}
				return newV5TradeWebsocketTestExchange(t, fixture)
			},
			expectedErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := tc.setup(t)
			var response *V5WebsocketOrderResponse
			ctx := t.Context()
			if tc.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := h.sendV5TradeRequest(ctx, "place_order", &V5OrderRequest{ContractCode: "BTC-USDT"}, &response)
			if tc.expectedErr {
				require.Error(t, err, "sendV5TradeRequest must return the expected error")
				return
			}
			require.NoError(t, err, "sendV5TradeRequest must not error")
			require.NotNil(t, response, "sendV5TradeRequest response must not be nil")
			assert.Equal(t, "1", response.Acknowledgement.OrderID, "order ID should decode")
		})
	}
}

func TestWebsocketPlaceV5Order(t *testing.T) {
	t.Parallel()
	for _, quoted := range []bool{false, true} {
		t.Run(map[bool]string{false: "object acknowledgement", true: "string acknowledgement"}[quoted], func(t *testing.T) {
			t.Parallel()
			h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, payload []byte, conn *gws.Conn) error {
				tb.Helper()
				cid, err := jsonparser.GetString(payload, "cid")
				if err != nil {
					return err
				}
				rate := `{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}`
				if quoted {
					encoded, err := json.Marshal(rate)
					if err != nil {
						return err
					}
					rate = string(encoded)
				}
				return conn.WriteMessage(gws.TextMessage, []byte(`{"code":200,"message":"Success","cid":"`+cid+`","data":{"order_id":"1"},"rate_limit":`+rate+`}`))
			})
			resp, err := h.WebsocketPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1})
			require.NoError(t, err, "trade acknowledgement must decode both documented shapes")
			require.NotNil(t, resp, "placed order acknowledgement must remain available")
			assert.Equal(t, "1", resp.Acknowledgement.OrderID, "placed order should retain its ID")
			assert.Equal(t, types.Number(19), resp.RateLimit.Remaining, "remaining limit should decode through actual transport")
		})
	}
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	_, err := h.WebsocketPlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketPlaceV5Order must reject nil request")
	resp, err := h.WebsocketPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1})
	require.NoError(t, err, "WebsocketPlaceV5Order must not error")
	assert.Equal(t, "1", resp.Acknowledgement.OrderID, "order ID should decode")
	assert.Equal(t, types.Number(19), resp.RateLimit.Remaining, "remaining rate limit should decode")

	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = h.WebsocketPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT"})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketPlaceV5Order must reject unauthenticated requests")
}

func TestWebsocketPlaceV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	_, err := h.WebsocketPlaceV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WebsocketPlaceV5BatchOrders must reject empty request")
	resp, err := h.WebsocketPlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1}})
	require.NoError(t, err, "WebsocketPlaceV5BatchOrders must not error")
	require.Len(t, resp.Acknowledgements, 1, "one order acknowledgement must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")

	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = h.WebsocketPlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT"}})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketPlaceV5BatchOrders must reject unauthenticated requests")
}

func TestWebsocketCancelV5Order(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	_, err := h.WebsocketCancelV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCancelV5Order must reject nil request")
	resp, err := h.WebsocketCancelV5Order(t.Context(), &V5CancelOrderRequest{ContractCode: "BTC-USDT", OrderID: "1"})
	require.NoError(t, err, "WebsocketCancelV5Order must not error")
	assert.Equal(t, "1", resp.Acknowledgement.OrderID, "order ID should decode")

	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = h.WebsocketCancelV5Order(t.Context(), &V5CancelOrderRequest{ContractCode: "BTC-USDT", OrderID: "1"})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketCancelV5Order must reject unauthenticated requests")
}

func TestWebsocketCancelV5BatchOrders(t *testing.T) {
	t.Parallel()
	eleven := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	type testCase struct {
		name    string
		request *V5CancelBatchOrdersRequest
		wantErr error
	}
	for _, tc := range []testCase{
		{name: "nil request", wantErr: common.ErrNilPointer},
		{name: "mixed identifiers", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{"1"}, ClientOrderIDs: []string{"2"}}, wantErr: errMixedOrderIdentifiers},
		{name: "no identifiers", request: &V5CancelBatchOrdersRequest{}, wantErr: order.ErrOrderIDNotSet},
		{name: "order one ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: eleven[:1]}},
		{name: "order ten IDs", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: eleven[:10]}},
		{name: "order eleven IDs", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: eleven}, wantErr: errCancelOrderLimitExceeded},
		{name: "order single empty ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{""}}, wantErr: order.ErrOrderIDNotSet},
		{name: "order empty first ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"", "2", "3"}}, wantErr: order.ErrOrderIDNotSet},
		{name: "order empty middle ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1", "", "3"}}, wantErr: order.ErrOrderIDNotSet},
		{name: "order empty last ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1", "2", ""}}, wantErr: order.ErrOrderIDNotSet},
		{name: "client order one ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: eleven[:1]}},
		{name: "client order ten IDs", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: eleven[:10]}},
		{name: "client order eleven IDs", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: eleven}, wantErr: errCancelOrderLimitExceeded},
		{name: "client order single empty ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: []string{""}}, wantErr: order.ErrOrderIDNotSet},
		{name: "client order empty first ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: []string{"", "2", "3"}}, wantErr: order.ErrOrderIDNotSet},
		{name: "client order empty middle ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: []string{"1", "", "3"}}, wantErr: order.ErrOrderIDNotSet},
		{name: "client order empty last ID", request: &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", ClientOrderIDs: []string{"1", "2", ""}}, wantErr: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var sent atomic.Int32
			h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, msg []byte, conn *gws.Conn) error {
				tb.Helper()
				if operation, _ := jsonparser.GetString(msg, "op"); operation == "cancel_batch_orders" {
					sent.Add(1)
				}
				return wsFixture(tb, msg, conn)
			})
			resp, err := h.WebsocketCancelV5BatchOrders(t.Context(), tc.request)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid batch must return its validation error")
				assert.Zero(t, sent.Load(), "invalid batch should never reach the trade socket")
				return
			}
			require.NoError(t, err, "valid batch must succeed")
			require.Len(t, resp.Acknowledgements, 1, "cancellation acknowledgement must decode")
			assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")
			assert.Equal(t, int32(1), sent.Load(), "valid batch should send exactly one request")
			h.Websocket.SetCanUseAuthenticatedEndpoints(false)
			_, err = h.WebsocketCancelV5BatchOrders(t.Context(), tc.request)
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "unauthenticated batch must fail")
			assert.Equal(t, int32(1), sent.Load(), "unauthenticated batch should not send another request")
		})
	}
}

func TestWebsocketCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	_, err := h.WebsocketCancelAllV5Orders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCancelAllV5Orders must reject nil request")
	resp, err := h.WebsocketCancelAllV5Orders(t.Context(), &V5CancelAllOrdersRequest{ContractCode: "BTC-USDT"})
	require.NoError(t, err, "WebsocketCancelAllV5Orders must not error")
	require.Len(t, resp.Acknowledgements, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")

	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = h.WebsocketCancelAllV5Orders(t.Context(), &V5CancelAllOrdersRequest{ContractCode: "BTC-USDT"})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketCancelAllV5Orders must reject unauthenticated requests")
}

func TestSendV5WSOrderUpdate(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		data := &V5WebsocketOrder{
			Side:              "buy",
			PositionSide:      "long",
			Type:              orderPriceTypePostOnly,
			OrderID:           "1",
			ClientOrderID:     "2",
			MarginMode:        "cross",
			Price:             100,
			Volume:            2,
			LeverageRate:      5,
			State:             "filled",
			ReduceOnly:        true,
			TimeInForce:       "rpi",
			TradeAveragePrice: 100,
			TradeVolume:       1,
			TradeTurnover:     100,
			FeeCurrency:       "HTX",
			Fee:               0.1,
			CreatedTime:       types.Time(time.UnixMilli(1000)),
			UpdatedTime:       types.Time(time.UnixMilli(2000)),
		}
		sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures}
		require.NoError(t, h.sendV5WSOrderUpdate(t.Context(), sub, "BTC-USDT", data), "sendV5WSOrderUpdate must convert and send the update")
		require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
		message := <-h.Websocket.DataHandler.C
		detail, ok := message.Data.(*order.Detail)
		require.True(t, ok, "update must use the canonical order type")
		assert.Equal(t, "1", detail.OrderID, "order ID should match")
		assert.Equal(t, "2", detail.ClientOrderID, "client order ID should match")
		assert.Equal(t, order.Limit, detail.Type, "order type should match")
		assert.Equal(t, order.PostOnly, detail.TimeInForce, "RPI should retain resting liquidity behaviour")
		assert.Equal(t, currency.USDT, detail.FeeAsset, "aggregate fee should remain USDT despite HTX payment currency")
		assert.Equal(t, 5.0, detail.Leverage, "leverage should match")
		assert.True(t, detail.ReduceOnly, "reduce-only should match")
		assert.Equal(t, time.UnixMilli(1000), detail.Date, "creation time should match")
		assert.Equal(t, time.UnixMilli(2000), detail.LastUpdated, "update time should match")
	})

	t.Run("invalid order", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		err := h.sendV5WSOrderUpdate(t.Context(), &subscription.Subscription{Asset: asset.USDTMarginedFutures}, "BTC-USDT", &V5WebsocketOrder{Side: "invalid"})
		require.Error(t, err, "sendV5WSOrderUpdate must reject an invalid order")
	})
}

func TestWSHandleUSDTMarginedPrivateMessage(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		channel  string
		expected any
		raw      string
	}
	for _, tt := range []testCase{
		{name: "orders", channel: subscription.MyOrdersChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":{"side":"buy","type":"limit","order_id":"1","margin_mode":"cross","state":"filled","time_in_force":"rpi","volume":"2","trade_volume":"1"}}`},
		{name: "trades", channel: wsTradeUpdatesChannel, expected: []fill.Data{}, raw: `{"contract_code":"BTC-USDT","data":[{"direction":"buy","order_id":"123","trade_id":456,"id":"456-123-1","trade_price":"100","trade_volume":"1","updated_time":1749468764315}]}`},
		{name: "trade details", channel: wsExecutionDetailsChannel, expected: []fill.Data{}, raw: `{"contract_code":"BTC-USDT","data":[{"direction":"buy","order_id":"123","trade_id":456,"id":"456-123-1","trade_price":"100","trade_volume":"1","updated_time":1749468764315}]}`},
		{name: "positions", channel: wsPositionsChannel, expected: &V5WebsocketPositionUpdate{}},
		{name: "account", channel: subscription.MyAccountChannel, expected: []accounts.Change{}, raw: `{"ts":1603878749908,"data":{"details":[{"currency":"USDT","equity":"2","isolated_equity":"3","available":"4","profit_unreal":"-2","initial_margin":"1","isolated_available":"0.9","available_margin":"1"}]}}`},
		{name: "matches", channel: subscription.MyTradesChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":[{"side":"buy","type":"limit","order_id":"1381717672952373249","state":"partially_filled","volume":"5","trade_volume":"1","total_trade_volume":"3","trade_price":"100","lever_rate":"","match_time":"1749468764315"}]}`},
		{name: "algo orders", channel: wsTriggerOrdersChannel, expected: &V5WebsocketAlgoOrderUpdate{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			raw := []byte(tt.raw)
			if len(raw) == 0 {
				raw = []byte(`{"op":"notify","topic":"private","ts":1603878749908,"uid":"123","contract_code":"BTC-USDT","data":[]}`)
			}
			require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "private USDT-margined notification must be decoded")
			require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
			if tt.name == "matches" {
				detail, ok := message.Data.(*order.Detail)
				require.True(t, ok, "match must produce an order update")
				assert.Equal(t, 3.0, detail.ExecutedAmount, "cumulative execution volume should be used")
				assert.Equal(t, 2.0, detail.RemainingAmount, "remaining volume should subtract cumulative executions")
				assert.Equal(t, time.UnixMilli(1749468764315), detail.LastUpdated, "match time should be retained")
				assert.True(t, detail.Date.IsZero(), "absent creation time should remain zero")
			}

			if changes, ok := message.Data.([]accounts.Change); ok {
				require.Len(t, changes, 1, "account change must be returned")
				assert.Equal(t, 5.0, changes[0].Balance.Total, "total should include isolated equity")
				assert.Equal(t, 1.0, changes[0].Balance.Free, "free should follow cross-margin availability")
				assert.Equal(t, 4.0, changes[0].Balance.Hold, "held should reconcile with total and free")
				stored, err := h.Accounts.GetBalance("", &accounts.Credentials{Key: "key", Secret: "secret"}, sub.Asset, changes[0].Balance.Currency)
				require.NoError(t, err, "account notification must reach the balance store")
				assert.Equal(t, changes[0].Balance.Total, stored.Total, "stored equity should match the notification")
				assert.Equal(t, changes[0].Balance.Free, stored.Free, "stored free margin should match the notification")
				assert.Equal(t, changes[0].Balance.Hold, stored.Hold, "stored held margin should match the notification")
				assert.Equal(t, time.UnixMilli(1603878749908), changes[0].Balance.UpdatedAt, "emitted changes should retain venue time")
				assert.WithinDuration(t, time.Now(), stored.UpdatedAt, time.Second, "stored balances should use local arrival time")
			}
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: "unsupported"}, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "unknown private USDT-margined channels must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel}, []byte(`{`))
	require.Error(t, err, "malformed private USDT-margined notifications must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "nil private USDT-margined subscriptions must be rejected")

	raw := []byte(`{
		"op":"notify",
		"topic":"orders",
		"contract_code":"SHIB-USDT",
		"ts":1640756528985,
		"uid":"502061937",
		"data":{
			"side":"buy",
			"position_side":"short",
			"type":"limit",
			"price":"0.0000124",
			"volume":"2",
			"lever_rate":30,
			"state":"new",
			"order_id":"1381668675223068672",
			"client_order_id":"1381668675223068672",
			"margin_mode":"cross",
			"reduce_only":true,
			"time_in_force":"gtc"
		}
	}`)
	sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}
	require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "V5 order notification must be decoded")
	require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
	message := <-h.Websocket.DataHandler.C
	orderUpdate, ok := message.Data.(*order.Detail)
	require.True(t, ok, "V5 order notification must use the canonical order type")
	assert.Equal(t, "1381668675223068672", orderUpdate.OrderID, "order ID should be retained")
	assert.Equal(t, 30.0, orderUpdate.Leverage, "leverage should be retained")
	assert.True(t, orderUpdate.ReduceOnly, "reduce-only should be retained")
}

func TestSendV5WSExecutionUpdates(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, raw string
		wantErr   bool
	}
	for _, tc := range []testCase{
		{name: "exact IDs and multiple fills", raw: `[{"contract_code":"BTC-USDT","side":"buy","order_id":1381717672952373249,"trade_id":"100005308993681","id":"first","trade_volume":"1","trade_price":"100","updated_time":1749468764315},{"direction":"sell","order_id":"1381717672952373251","trade_id":100005308993682,"id":"second","trade_volume":"2","trade_price":"101","created_time":"1749468764316"}]`},
		{name: "invalid side", raw: `[{"contract_code":"BTC-USDT","side":"invalid"}]`, wantErr: true},
		{name: "invalid symbol", raw: `[{"contract_code":"-","side":"buy"}]`, wantErr: true},
		{name: "empty", raw: `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "setup must succeed")
			var data []V5WebsocketExecution
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &data), "execution array must decode")
			err := h.sendV5WSExecutionUpdates(t.Context(), &subscription.Subscription{Asset: asset.USDTMarginedFutures}, "BTC-USDT", data)
			if tc.wantErr {
				require.Error(t, err, "invalid fill must fail")
				return
			}
			require.NoError(t, err, "execution conversion must succeed")
			if len(data) == 0 {
				assert.Empty(t, h.Websocket.DataHandler.C, "empty updates should emit nothing")
				return
			}
			require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
			message := <-h.Websocket.DataHandler.C
			fills, ok := message.Data.([]fill.Data)
			require.True(t, ok, "executions must use canonical fills")
			require.Len(t, fills, 2, "every execution must be retained")
			assert.Equal(t, "1381717672952373249", fills[0].OrderID, "numeric order ID should retain precision")
			assert.Equal(t, "1381717672952373251", fills[1].OrderID, "quoted order ID should retain precision")
			assert.Equal(t, "100005308993681", fills[0].TradeID, "trade ID should be retained")
			assert.Equal(t, order.Sell, fills[1].Side, "direction alias should decode")
			assert.Equal(t, btcusdtPair, fills[1].CurrencyPair, "envelope symbol should provide fallback")
			assert.Equal(t, time.UnixMilli(1749468764316), fills[1].Timestamp, "creation time should provide fallback")
			assert.Equal(t, 2.0, fills[1].Amount, "execution volume should be retained")
		})
	}
}
