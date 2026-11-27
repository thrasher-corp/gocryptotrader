package okx

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// websocketOperationReply answers every operation with code and data
func websocketOperationReply(code string, data any) func(string, string, json.RawMessage) (string, any) {
	return func(string, string, json.RawMessage) (string, any) { return code, data }
}

// requireOneOperation returns the one request sent for an operation, checking
// it went out on a connection to path which had logged in
func requireOneOperation(t *testing.T, srv *websocketTestServer, operation, path string) websocketTestFrame {
	t.Helper()
	frames := srv.requests(operation)
	require.Lenf(t, frames, 1, "one %s request must be sent", operation)
	assert.Equalf(t, path, frames[0].path, "%s should be sent on its documented connection", operation)
	assert.Truef(t, frames[0].loggedIn, "%s should be sent on a connection which logged in", operation)
	return frames[0]
}

func TestWebsocketNullRowsDoNotPanic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		call func(e *Exchange) error
	}{
		{
			name: "submit order",
			call: func(e *Exchange) error {
				_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
					Pair: mainPair, AssetType: asset.Spot,
					Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1,
				})
				return err
			},
		},
		{
			name: "mass cancel",
			call: func(e *Exchange) error {
				return e.WebsocketAPI().CancelAllMMPOrders(t.Context(), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"})
			},
		},
		{
			name: "cancel all spread orders",
			call: func(e *Exchange) error {
				return e.WebsocketAPI().CancelAllSpreadOrders(t.Context(), spreadPair.String())
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockWebsocketExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
					return
				}
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
			}), func(string, json.RawMessage) (string, any) {
				return "0", []any{nil}
			})
			var err error
			require.NotPanics(t, func() {
				err = tc.call(e)
			}, "the operation must not panic on a success reply with a null row")
			require.ErrorIs(t, err, common.ErrInvalidResponse, "the operation must reject a success reply with a null row")
		})
	}
}

func TestWebsocketAPIPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().PlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOrder must reject a nil request")
	_, err = e.WebsocketAPI().PlaceOrder(t.Context(), &PlaceOrderRequest{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "PlaceOrder must reject an order without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		got, err := e.WebsocketAPI().PlaceOrder(request.WithVerbose(t.Context()), &PlaceOrderRequest{
			InstrumentID:     mainPair.String(),
			InstrumentIDCode: e.getInstrumentIDCode(mainPair.String()),
			TradeMode:        TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
			Side:             "Buy",
			OrderType:        "post_only",
			Size:             0.0001,
			Price:            20000,
			Currency:         currency.USDT,
		})
		require.NoError(t, err, "PlaceOrder must not error")
		assert.NotEmpty(t, got, "PlaceOrder should return the order")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "12345689", "tag": "", "ts": "1695190491421", "sCode": "0", "sMsg": "", "subCode": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().PlaceOrder(t.Context(), &PlaceOrderRequest{
			InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000,
		})
		require.NoError(t, err, "PlaceOrder must not error")
		exp := &OrderActionResponse{OrderID: "12345689", Timestamp: types.Time(time.UnixMilli(1695190491421))}
		assert.Equal(t, exp, got, "PlaceOrder should decode the reply")
		frame := requireOneOperation(t, srv, "order", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"limit","sz":"0.0001","px":"20000"}]`, string(frame.arguments), "PlaceOrder should send the order with its instrument ID code")
	})

	t.Run("instrument ID code alone", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "12345689", "tag": "", "ts": "1695190491421", "sCode": "0", "sMsg": "", "subCode": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().PlaceOrder(t.Context(), &PlaceOrderRequest{
			InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000,
		})
		require.NoError(t, err, "PlaceOrder must place an order its instrument ID code alone names")
		frame := requireOneOperation(t, srv, "order", "/ws/v5/private")
		assert.JSONEq(t, `[{"instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"limit","sz":"0.0001","px":"20000"}]`, string(frame.arguments), "PlaceOrder should send the order with its instrument ID code alone")
	})

	t.Run("refuses an account other than the connection's login", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"ordId": "12345689", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
		_, err := ex.WebsocketAPI().PlaceOrder(ctx, &PlaceOrderRequest{
			InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000,
		})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "PlaceOrder must refuse an order for another account than the connection logged in as")
		assert.Empty(t, srv.requests("order"), "no order should be sent")
	})

	t.Run("refuses an attached take profit or stop loss", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"ordId": "1001", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().PlaceOrder(t.Context(), &PlaceOrderRequest{
			InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1, Price: 100,
			AttachAlgoOrders: []AttachAlgoOrderRequest{{StopLossTriggerPrice: 90, StopLossOrderPrice: marketOrderPrice}},
		})
		require.ErrorIs(t, err, errAttachedOrderUnsupported, "PlaceOrder must refuse an attachment OKX's websocket order operation does not take")
		assert.Empty(t, srv.requests("order"), "no order should be sent")
	})
}

func TestWebsocketAPIPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().PlaceMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "PlaceMultipleOrders must reject an empty batch")
	_, err = e.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{{}})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceMultipleOrders must reject an order without its fields")
	_, err = e.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{{TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "PlaceMultipleOrders must reject an order without an instrument")
	_, err = e.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "PlaceMultipleOrders must reject an order without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		got, err := e.WebsocketAPI().PlaceMultipleOrders(request.WithVerbose(t.Context()), []PlaceOrderRequest{{
			InstrumentID:     mainPair.String(),
			InstrumentIDCode: e.getInstrumentIDCode(mainPair.String()),
			TradeMode:        TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
			Side:             "Buy",
			OrderType:        "post_only",
			Size:             0.0001,
			Price:            20000,
			Currency:         currency.USDT,
		}})
		require.NoError(t, err, "PlaceMultipleOrders must not error")
		assert.NotEmpty(t, got, "PlaceMultipleOrders should return the orders")
	})

	t.Run("limits the batch size", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := ex.WebsocketAPI().PlaceMultipleOrders(t.Context(), make([]PlaceOrderRequest, 21))
		assert.ErrorIs(t, err, errExceedLimit, "an oversized websocket batch should be rejected before transmission")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("2", []map[string]string{
			{"clOrdId": "", "ordId": "12345689", "tag": "", "ts": "1695190491421", "sCode": "0", "sMsg": "", "subCode": ""},
			{"clOrdId": "", "ordId": "", "tag": "", "ts": "1695190491421", "sCode": "51008", "sMsg": "Order failed. Insufficient USDT balance in account", "subCode": "1000"},
		})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		buy := PlaceOrderRequest{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000}
		sell := buy
		sell.Side, sell.Price = "sell", 30000
		got, err := ex.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{buy, sell})
		require.ErrorIs(t, err, errPartialSuccess, "PlaceMultipleOrders must report the partial success")
		exp := []*OrderActionResponse{
			{OrderID: "12345689", Timestamp: types.Time(time.UnixMilli(1695190491421))},
			{StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account", SubCode: "1000", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		}
		assert.Equal(t, exp, got, "PlaceMultipleOrders should return every order's outcome beside the error")
		frame := requireOneOperation(t, srv, "batch-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"limit","sz":"0.0001","px":"20000"},{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"sell","ordType":"limit","sz":"0.0001","px":"30000"}]`, string(frame.arguments), "PlaceMultipleOrders should send every order")
	})

	t.Run("instrument ID codes alone", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		buy := PlaceOrderRequest{InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000}
		sell := buy
		sell.Side, sell.Price = "sell", 30000
		_, err := ex.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{buy, sell})
		require.NoError(t, err, "PlaceMultipleOrders must place orders their instrument ID codes alone name")
		frame := requireOneOperation(t, srv, "batch-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"limit","sz":"0.0001","px":"20000"},{"instIdCode":12345,"tdMode":"cash","side":"sell","ordType":"limit","sz":"0.0001","px":"30000"}]`, string(frame.arguments), "PlaceMultipleOrders should send every order with its instrument ID code alone")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{placeOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{placeMultipleOrdersEPL: 2}},
			{"a full batch", maxWebsocketBatchOrders, map[request.EndpointLimit]float64{placeMultipleOrdersEPL: maxWebsocketBatchOrders}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", tc.orders))}
				ex, pools := newChargedWebsocketTestExchange(t, srv)
				args := slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000}}, tc.orders)
				_, err := ex.WebsocketAPI().PlaceMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "PlaceMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "PlaceMultipleOrders should take a token per order from the budget the websocket shares with REST")
				requireOneOperation(t, srv, "batch-orders", "/ws/v5/private")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.True(t, pools[placeMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.WebsocketAPI().PlaceMultipleOrders(chargeContext(t), slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000}}, 2))
		assert.ErrorIs(t, err, context.DeadlineExceeded, "PlaceMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Empty(t, srv.requests("batch-orders"), "PlaceMultipleOrders should send nothing when its budget cannot cover both orders")
		assert.Equal(t, 1.0, pools[placeMultipleOrdersEPL].Tokens(), "PlaceMultipleOrders should leave the budget it could not use")
	})

	t.Run("a disconnected websocket takes nothing from the order budget", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 3))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
		_, err := ex.WebsocketAPI().PlaceMultipleOrders(chargeContext(t), slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 0.0001, Price: 20000}}, 3))
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "PlaceMultipleOrders should fail without a connection")
		assert.Empty(t, takenTokens(pools), "PlaceMultipleOrders should take nothing from the budget for orders it cannot send")
	})

	t.Run("refuses an attached take profit or stop loss", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		plain := PlaceOrderRequest{InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1, Price: 100}
		attached := plain
		attached.AttachAlgoOrders = []AttachAlgoOrderRequest{{StopLossTriggerPrice: 90, StopLossOrderPrice: marketOrderPrice}}
		_, err := ex.WebsocketAPI().PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{plain, attached})
		require.ErrorIs(t, err, errAttachedOrderUnsupported, "PlaceMultipleOrders must refuse an attachment OKX's websocket order operation does not take")
		assert.Empty(t, srv.requests("batch-orders"), "no order should be sent")
	})
}

func TestValidateWebsocketPlace(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		arg    *PlaceOrderRequest
		expErr error
	}{
		{"nil", nil, common.ErrNilPointer},
		{"no side", &PlaceOrderRequest{InstrumentIDCode: 1, TradeMode: TradeModeCash, OrderType: orderLimit, Size: 1}, order.ErrSideIsInvalid},
		{"no instrument", &PlaceOrderRequest{TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}, errMissingInstrumentIDCode},
		{"no instrument ID code", &PlaceOrderRequest{InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}, errMissingInstrumentIDCode},
		{"instrument ID code alone", &PlaceOrderRequest{InstrumentIDCode: 1, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}, nil},
		{"both instrument identifiers", &PlaceOrderRequest{InstrumentID: "BTC-USDT", InstrumentIDCode: 1, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1}, nil},
		{"no attachments", &PlaceOrderRequest{InstrumentIDCode: 1, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1, AttachAlgoOrders: []AttachAlgoOrderRequest{}}, nil},
		{"an attached stop loss", &PlaceOrderRequest{InstrumentIDCode: 1, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1, AttachAlgoOrders: []AttachAlgoOrderRequest{{StopLossTriggerPrice: 90, StopLossOrderPrice: marketOrderPrice}}}, errAttachedOrderUnsupported},
		{"an attachment without values", &PlaceOrderRequest{InstrumentIDCode: 1, TradeMode: TradeModeCash, Side: "buy", OrderType: orderLimit, Size: 1, AttachAlgoOrders: []AttachAlgoOrderRequest{{}}}, errAttachedOrderUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateWebsocketPlace(tc.arg), tc.expErr, "validateWebsocketPlace should return the expected error")
		})
	}
}

func TestWebsocketAPICancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CancelSingleOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelSingleOrder must reject a nil request")
	_, err = e.WebsocketAPI().CancelSingleOrder(t.Context(), &CancelOrderRequest{OrderID: "1"})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "CancelSingleOrder must reject a request without an instrument")
	_, err = e.WebsocketAPI().CancelSingleOrder(t.Context(), &CancelOrderRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleOrder must reject a request without an order ID")
	_, err = e.WebsocketAPI().CancelSingleOrder(t.Context(), &CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: "1"})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "CancelSingleOrder must reject a request without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		got, err := e.WebsocketAPI().CancelSingleOrder(request.WithVerbose(t.Context()), &CancelOrderRequest{
			InstrumentID:     mainPair.String(),
			InstrumentIDCode: e.getInstrumentIDCode(mainPair.String()),
			OrderID:          "2341161427393388544",
		})
		require.NoError(t, err, "CancelSingleOrder must not error")
		assert.NotEmpty(t, got, "CancelSingleOrder should return the outcome")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "2510789768709120", "ts": "1695190491421", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().CancelSingleOrder(t.Context(), &CancelOrderRequest{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2510789768709120"})
		require.NoError(t, err, "CancelSingleOrder must not error")
		exp := &OrderActionResponse{OrderID: "2510789768709120", Timestamp: types.Time(time.UnixMilli(1695190491421))}
		assert.Equal(t, exp, got, "CancelSingleOrder should decode the reply")
		frame := requireOneOperation(t, srv, "cancel-order", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"2510789768709120"}]`, string(frame.arguments), "CancelSingleOrder should send the cancellation with its instrument ID code")
	})

	t.Run("instrument ID code alone", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "2510789768709120", "ts": "1695190491421", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().CancelSingleOrder(t.Context(), &CancelOrderRequest{InstrumentIDCode: 12345, OrderID: "2510789768709120"})
		require.NoError(t, err, "CancelSingleOrder must cancel an order its instrument ID code alone names")
		frame := requireOneOperation(t, srv, "cancel-order", "/ws/v5/private")
		assert.JSONEq(t, `[{"instIdCode":12345,"ordId":"2510789768709120"}]`, string(frame.arguments), "CancelSingleOrder should send the cancellation with its instrument ID code alone")
	})
}

func TestWebsocketAPICancelMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CancelMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "CancelMultipleOrders must reject an empty batch")
	_, err = e.WebsocketAPI().CancelMultipleOrders(t.Context(), make([]CancelOrderRequest, 21))
	require.ErrorIs(t, err, errExceedLimit, "CancelMultipleOrders must reject more than 20 cancellations")
	_, err = e.WebsocketAPI().CancelMultipleOrders(t.Context(), []CancelOrderRequest{{OrderID: "1"}})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "CancelMultipleOrders must reject a cancellation without an instrument")
	_, err = e.WebsocketAPI().CancelMultipleOrders(t.Context(), []CancelOrderRequest{{InstrumentID: mainPair.String()}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMultipleOrders must reject a cancellation without an order ID")
	_, err = e.WebsocketAPI().CancelMultipleOrders(t.Context(), []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1"}})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "CancelMultipleOrders must reject a cancellation without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		got, err := e.WebsocketAPI().CancelMultipleOrders(request.WithVerbose(t.Context()), []CancelOrderRequest{{
			InstrumentID:     mainPair.String(),
			InstrumentIDCode: e.getInstrumentIDCode(mainPair.String()),
			OrderID:          "2341184920998715392",
		}})
		require.NoError(t, err, "CancelMultipleOrders must not error")
		assert.NotEmpty(t, got, "CancelMultipleOrders should return the outcomes")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("2", []map[string]string{
			{"clOrdId": "oktswap6", "ordId": "2517748157541376", "ts": "1695190491421", "sCode": "0", "sMsg": ""},
			{"clOrdId": "oktswap7", "ordId": "2517748155771904", "ts": "1695190491421", "sCode": "51400", "sMsg": "order not exist"},
		})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().CancelMultipleOrders(t.Context(), []CancelOrderRequest{
			{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, ClientOrderID: "oktswap6"},
			{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, ClientOrderID: "oktswap7"},
		})
		require.ErrorIs(t, err, errPartialSuccess, "CancelMultipleOrders must report the partial success")
		assert.ErrorContains(t, err, "order not exist", "CancelMultipleOrders should report the failed cancellation")
		exp := []*OrderActionResponse{
			{OrderID: "2517748157541376", ClientOrderID: "oktswap6", Timestamp: types.Time(time.UnixMilli(1695190491421))},
			{OrderID: "2517748155771904", ClientOrderID: "oktswap7", StatusCode: 51400, StatusMessage: "order not exist", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		}
		assert.Equal(t, exp, got, "CancelMultipleOrders should return every cancellation's outcome beside the error")
		frame := requireOneOperation(t, srv, "batch-cancel-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"clOrdId":"oktswap6"},{"instId":"BTC-USDT","instIdCode":12345,"clOrdId":"oktswap7"}]`, string(frame.arguments), "CancelMultipleOrders should send every cancellation")
	})

	t.Run("instrument ID codes alone", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().CancelMultipleOrders(t.Context(), []CancelOrderRequest{{InstrumentIDCode: 12345, OrderID: "1"}, {InstrumentIDCode: 12345, ClientOrderID: "oktswap7"}})
		require.NoError(t, err, "CancelMultipleOrders must cancel orders their instrument ID codes alone name")
		frame := requireOneOperation(t, srv, "batch-cancel-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instIdCode":12345,"ordId":"1"},{"instIdCode":12345,"clOrdId":"oktswap7"}]`, string(frame.arguments), "CancelMultipleOrders should send every cancellation with its instrument ID code alone")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{cancelOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{cancelMultipleOrdersEPL: 2}},
			{"a full batch", maxWebsocketBatchOrders, map[request.EndpointLimit]float64{cancelMultipleOrdersEPL: maxWebsocketBatchOrders}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", tc.orders))}
				ex, pools := newChargedWebsocketTestExchange(t, srv)
				args := make([]CancelOrderRequest, tc.orders)
				for i := range args {
					args[i] = CancelOrderRequest{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: strconv.Itoa(i + 1)}
				}
				_, err := ex.WebsocketAPI().CancelMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "CancelMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "CancelMultipleOrders should take a token per order from the budget the websocket shares with REST")
				requireOneOperation(t, srv, "batch-cancel-orders", "/ws/v5/private")
			})
		}
	})

	t.Run("a disconnected websocket takes nothing from the order budget", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 3))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
		_, err := ex.WebsocketAPI().CancelMultipleOrders(chargeContext(t), []CancelOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "1"}, {InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2"}, {InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "3"}})
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "CancelMultipleOrders should fail without a connection")
		assert.Empty(t, takenTokens(pools), "CancelMultipleOrders should take nothing from the budget for orders it cannot send")
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.True(t, pools[cancelMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.WebsocketAPI().CancelMultipleOrders(chargeContext(t), []CancelOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "1"}, {InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2"}})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "CancelMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Empty(t, srv.requests("batch-cancel-orders"), "CancelMultipleOrders should send nothing when its budget cannot cover both orders")
		assert.Equal(t, 1.0, pools[cancelMultipleOrdersEPL].Tokens(), "CancelMultipleOrders should leave the budget it could not use")
	})
}

func TestValidateWebsocketCancel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		arg    CancelOrderRequest
		expErr error
	}{
		{"no instrument", CancelOrderRequest{OrderID: "1"}, errMissingInstrumentIDCode},
		{"instrument ID code alone", CancelOrderRequest{OrderID: "1", InstrumentIDCode: 1}, nil},
		{"no order ID", CancelOrderRequest{InstrumentID: "BTC-USDT", InstrumentIDCode: 1}, order.ErrOrderIDNotSet},
		{"no instrument ID code", CancelOrderRequest{InstrumentID: "BTC-USDT", OrderID: "1"}, errMissingInstrumentIDCode},
		{"order ID", CancelOrderRequest{InstrumentID: "BTC-USDT", OrderID: "1", InstrumentIDCode: 1}, nil},
		{"client order ID", CancelOrderRequest{InstrumentID: "BTC-USDT", ClientOrderID: "a", InstrumentIDCode: 1}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateWebsocketCancel(&tc.arg), tc.expErr, "validateWebsocketCancel should return the expected error")
		})
	}
}

func TestWebsocketAPIAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().AmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendOrder must reject a nil request")
	out := &AmendOrderRequest{}
	_, err = e.WebsocketAPI().AmendOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendOrder must reject an amendment without an order ID")
	out.OrderID = "2341200629875154944"
	_, err = e.WebsocketAPI().AmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation, "AmendOrder must reject an amendment without a new size or price")
	out.NewPrice = 21000
	_, err = e.WebsocketAPI().AmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "AmendOrder must reject an amendment without an instrument")
	out.InstrumentID = mainPair.String()
	_, err = e.WebsocketAPI().AmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "AmendOrder must reject an amendment without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		live := *out
		live.InstrumentIDCode = e.getInstrumentIDCode(mainPair.String())
		got, err := e.WebsocketAPI().AmendOrder(request.WithVerbose(t.Context()), &live)
		require.NoError(t, err, "AmendOrder must not error")
		assert.NotEmpty(t, got, "AmendOrder should return the outcome")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "2510789768709120", "ts": "1695190491421", "reqId": "b12344", "sCode": "0", "sMsg": "", "subCode": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().AmendOrder(t.Context(), &AmendOrderRequest{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2510789768709120", ClientRequestID: "b12344", NewPrice: 21000})
		require.NoError(t, err, "AmendOrder must not error")
		exp := &OrderActionResponse{OrderID: "2510789768709120", RequestID: "b12344", Timestamp: types.Time(time.UnixMilli(1695190491421))}
		assert.Equal(t, exp, got, "AmendOrder should decode the reply")
		frame := requireOneOperation(t, srv, "amend-order", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"2510789768709120","reqId":"b12344","newPx":"21000"}]`, string(frame.arguments), "AmendOrder should send the amendment with its instrument ID code")
	})

	// OKX documents an option's amendment in USD or implied volatility in
	// place of a new price, and the instrument by its ID code alone
	t.Run("option price in USD or implied volatility", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			arg      *AmendOrderRequest
			expFrame string
		}{
			{"USD", &AmendOrderRequest{InstrumentIDCode: 666, OrderID: "1001", NewPriceUSD: 200}, `[{"instIdCode":666,"ordId":"1001","newPxUsd":"200"}]`},
			{"implied volatility", &AmendOrderRequest{InstrumentIDCode: 666, OrderID: "1001", NewPriceVolatility: 0.7}, `[{"instIdCode":666,"ordId":"1001","newPxVol":"0.7"}]`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "1001", "ts": "1695190491421", "reqId": "", "sCode": "0", "sMsg": "", "subCode": ""}})}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				_, err := ex.WebsocketAPI().AmendOrder(t.Context(), tc.arg)
				require.NoError(t, err, "AmendOrder must amend an option's price")
				frame := requireOneOperation(t, srv, "amend-order", "/ws/v5/private")
				assert.JSONEq(t, tc.expFrame, string(frame.arguments), "AmendOrder should send the option's new price")
			})
		}
	})

	t.Run("refuses an attached take profit or stop loss", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"ordId": "1001", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendOrder(t.Context(), &AmendOrderRequest{
			InstrumentIDCode: 12345, OrderID: "1001", NewSize: 2,
			AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice}},
		})
		require.ErrorIs(t, err, errAttachedOrderUnsupported, "AmendOrder must refuse an attachment OKX's websocket amend operation does not take")
		assert.Empty(t, srv.requests("amend-order"), "no amendment should be sent")
	})

	t.Run("refuses more than one new price", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"ordId": "1001", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendOrder(t.Context(), &AmendOrderRequest{InstrumentIDCode: 666, OrderID: "1001", NewPriceUSD: 200, NewPriceVolatility: 0.7})
		require.ErrorIs(t, err, errConflictingAmendPrices, "AmendOrder must refuse an option repriced both in USD and as implied volatility")
		assert.Empty(t, srv.requests("amend-order"), "no amendment should be sent")
	})
}

func TestWebsocketAPIAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().AmendMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "AmendMultipleOrders must reject an empty batch")
	_, err = e.WebsocketAPI().AmendMultipleOrders(t.Context(), make([]AmendOrderRequest, 21))
	require.ErrorIs(t, err, errExceedLimit, "AmendMultipleOrders must reject more than 20 amendments")
	out := AmendOrderRequest{}
	_, err = e.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendMultipleOrders must reject an amendment without an order ID")
	out.OrderID = "2341200629875154944"
	_, err = e.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation, "AmendMultipleOrders must reject an amendment without a new size or price")
	out.NewPrice = 20000
	_, err = e.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{out})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "AmendMultipleOrders must reject an amendment without an instrument")
	out.InstrumentID = mainPair.String()
	_, err = e.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{out})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "AmendMultipleOrders must reject an amendment without its instrument ID code")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		_, err := e.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		live := out
		live.InstrumentIDCode = e.getInstrumentIDCode(mainPair.String())
		got, err := e.WebsocketAPI().AmendMultipleOrders(request.WithVerbose(t.Context()), []AmendOrderRequest{live})
		require.NoError(t, err, "AmendMultipleOrders must not error")
		assert.NotEmpty(t, got, "AmendMultipleOrders should return the outcomes")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("1", []map[string]string{
			{"clOrdId": "", "ordId": "12345689", "ts": "1695190491421", "reqId": "b12344", "sCode": "51503", "sMsg": "order not exist"},
			{"clOrdId": "oktswap7", "ordId": "", "ts": "1695190491421", "reqId": "b12345", "sCode": "51008", "sMsg": "Order failed. Insufficient USDT balance in account", "subCode": "1000"},
		})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "12345689", ClientRequestID: "b12344", NewSize: 2},
			{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, ClientOrderID: "oktswap7", ClientRequestID: "b12345", NewPrice: 20000},
		})
		require.ErrorIs(t, err, errOperationFailed, "AmendMultipleOrders must report the failed operation")
		assert.ErrorContains(t, err, "Insufficient USDT balance", "AmendMultipleOrders should report each amendment's failure")
		exp := []*OrderActionResponse{
			{OrderID: "12345689", RequestID: "b12344", StatusCode: 51503, StatusMessage: "order not exist", Timestamp: types.Time(time.UnixMilli(1695190491421))},
			{ClientOrderID: "oktswap7", RequestID: "b12345", StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account", SubCode: "1000", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		}
		assert.Equal(t, exp, got, "AmendMultipleOrders should return every amendment's outcome beside the error")
		frame := requireOneOperation(t, srv, "batch-amend-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"12345689","reqId":"b12344","newSz":"2"},{"instId":"BTC-USDT","instIdCode":12345,"clOrdId":"oktswap7","reqId":"b12345","newPx":"20000"}]`, string(frame.arguments), "AmendMultipleOrders should send every amendment")
	})

	// OKX documents an option's amendment in USD or implied volatility in
	// place of a new price, and the instrument by its ID code alone
	t.Run("option prices in USD and implied volatility", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentIDCode: 666, OrderID: "1", NewPriceUSD: 200},
			{InstrumentIDCode: 666, OrderID: "2", NewPriceVolatility: 0.7},
		})
		require.NoError(t, err, "AmendMultipleOrders must amend options' prices")
		frame := requireOneOperation(t, srv, "batch-amend-orders", "/ws/v5/private")
		assert.JSONEq(t, `[{"instIdCode":666,"ordId":"1","newPxUsd":"200"},{"instIdCode":666,"ordId":"2","newPxVol":"0.7"}]`, string(frame.arguments), "AmendMultipleOrders should send each option's new price")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{amendOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{amendMultipleOrdersEPL: 2}},
			{"a full batch", maxWebsocketBatchOrders, map[request.EndpointLimit]float64{amendMultipleOrdersEPL: maxWebsocketBatchOrders}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", tc.orders))}
				ex, pools := newChargedWebsocketTestExchange(t, srv)
				args := make([]AmendOrderRequest, tc.orders)
				for i := range args {
					args[i] = AmendOrderRequest{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: strconv.Itoa(i + 1), NewSize: 2}
				}
				_, err := ex.WebsocketAPI().AmendMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "AmendMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "AmendMultipleOrders should take a token per order from the budget the websocket shares with REST")
				requireOneOperation(t, srv, "batch-amend-orders", "/ws/v5/private")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.True(t, pools[amendMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.WebsocketAPI().AmendMultipleOrders(chargeContext(t), []AmendOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "1", NewSize: 2}, {InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2", NewSize: 2}})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "AmendMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Empty(t, srv.requests("batch-amend-orders"), "AmendMultipleOrders should send nothing when its budget cannot cover both orders")
		assert.Equal(t, 1.0, pools[amendMultipleOrdersEPL].Tokens(), "AmendMultipleOrders should leave the budget it could not use")
	})

	t.Run("a disconnected websocket takes nothing from the order budget", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex, pools := newChargedWebsocketTestExchange(t, srv)
		require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
		_, err := ex.WebsocketAPI().AmendMultipleOrders(chargeContext(t), []AmendOrderRequest{{InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "1", NewSize: 2}, {InstrumentID: mainPair.String(), InstrumentIDCode: 12345, OrderID: "2", NewSize: 2}})
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "AmendMultipleOrders should fail without a connection")
		assert.Empty(t, takenTokens(pools), "AmendMultipleOrders should take nothing from the budget for orders it cannot send")
	})

	t.Run("refuses an attached take profit or stop loss", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentIDCode: 12345, OrderID: "1", NewSize: 2},
			{InstrumentIDCode: 12345, OrderID: "2", NewSize: 2, AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice}}},
		})
		require.ErrorIs(t, err, errAttachedOrderUnsupported, "AmendMultipleOrders must refuse an attachment OKX's websocket amend operation does not take")
		assert.Empty(t, srv.requests("batch-amend-orders"), "no amendment should be sent")
	})

	t.Run("refuses more than one new price", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", orderRows("ordId", 2))}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentIDCode: 666, OrderID: "1", NewPriceUSD: 200},
			{InstrumentIDCode: 666, OrderID: "2", NewPriceUSD: 200, NewPriceVolatility: 0.7},
		})
		require.ErrorIs(t, err, errConflictingAmendPrices, "AmendMultipleOrders must refuse an option repriced both in USD and as implied volatility")
		assert.Empty(t, srv.requests("batch-amend-orders"), "no amendment should be sent")
	})
}

func TestValidateWebsocketAmend(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		arg    AmendOrderRequest
		expErr error
	}{
		{"no instrument", AmendOrderRequest{OrderID: "1", NewSize: 1}, errMissingInstrumentIDCode},
		{"instrument ID code alone", AmendOrderRequest{OrderID: "1", NewSize: 1, InstrumentIDCode: 1}, nil},
		{"no order ID", AmendOrderRequest{InstrumentID: "BTC-USDT", NewSize: 1, InstrumentIDCode: 1}, order.ErrOrderIDNotSet},
		{"no new size or price", AmendOrderRequest{InstrumentID: "BTC-USDT", OrderID: "1", InstrumentIDCode: 1}, errInvalidNewSizeOrPriceInformation},
		{"no instrument ID code", AmendOrderRequest{InstrumentID: "BTC-USDT", OrderID: "1", NewPrice: 1}, errMissingInstrumentIDCode},
		{"new size", AmendOrderRequest{InstrumentID: "BTC-USDT", OrderID: "1", NewSize: 1, InstrumentIDCode: 1}, nil},
		{"new price by client order ID", AmendOrderRequest{InstrumentID: "BTC-USDT", ClientOrderID: "a", NewPrice: 1, InstrumentIDCode: 1}, nil},
		{"new option price in USD", AmendOrderRequest{OrderID: "1", NewPriceUSD: 200, InstrumentIDCode: 1}, nil},
		{"new option implied volatility", AmendOrderRequest{OrderID: "1", NewPriceVolatility: 0.7, InstrumentIDCode: 1}, nil},
		{"new option price in USD and implied volatility", AmendOrderRequest{OrderID: "1", NewPriceUSD: 200, NewPriceVolatility: 0.7, InstrumentIDCode: 1}, errConflictingAmendPrices},
		{"no attachments", AmendOrderRequest{OrderID: "1", NewSize: 1, InstrumentIDCode: 1, AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{}}, nil},
		{"an attached stop loss", AmendOrderRequest{OrderID: "1", NewSize: 1, InstrumentIDCode: 1, AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice}}}, errAttachedOrderUnsupported},
		{"an attachment without values", AmendOrderRequest{OrderID: "1", NewSize: 1, InstrumentIDCode: 1, AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{}}}, errAttachedOrderUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateWebsocketAmend(&tc.arg), tc.expErr, "validateWebsocketAmend should return the expected error")
		})
	}
}

func TestWebsocketAPICancelAllMMPOrders(t *testing.T) {
	t.Parallel()
	err := e.WebsocketAPI().CancelAllMMPOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllMMPOrders must reject a nil request")
	err = e.WebsocketAPI().CancelAllMMPOrders(t.Context(), &MassCancelOrderRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType, "CancelAllMMPOrders must reject a request without an instrument type")
	err = e.WebsocketAPI().CancelAllMMPOrders(t.Context(), &MassCancelOrderRequest{InstrumentType: instTypeOption})
	require.ErrorIs(t, err, errInstrumentFamilyRequired, "CancelAllMMPOrders must reject a request without an instrument family")
	err = e.WebsocketAPI().CancelAllMMPOrders(t.Context(), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", LockInterval: 10001})
	require.ErrorIs(t, err, errLockIntervalOutOfRange, "CancelAllMMPOrders must reject a lock interval above 10 seconds")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		err := e.WebsocketAPI().CancelAllMMPOrders(request.WithVerbose(t.Context()), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: optionsPair.String()})
		require.NoError(t, err, "CancelAllMMPOrders must not error")
	})

	for _, tc := range []struct {
		name   string
		result bool
		expErr error
	}{
		{"mock websocket cancelled", true, nil},
		{"mock websocket not cancelled", false, errMassCancelFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]bool{{"result": tc.result}})}
			ex := newWebsocketTestExchange(t, srv, true, nil)
			err := ex.WebsocketAPI().CancelAllMMPOrders(t.Context(), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", LockInterval: 5000})
			require.ErrorIs(t, err, tc.expErr, "CancelAllMMPOrders must report the reply's result")
			frame := requireOneOperation(t, srv, "mass-cancel", "/ws/v5/private")
			assert.JSONEq(t, `[{"instType":"OPTION","instFamily":"BTC-USD","lockInterval":"5000"}]`, string(frame.arguments), "CancelAllMMPOrders should send the instrument family and lock interval")
		})
	}
}
