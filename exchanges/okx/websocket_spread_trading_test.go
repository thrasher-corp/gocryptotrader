package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketAPIPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().PlaceSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpreadOrder must reject a nil request")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		result, err := e.WebsocketAPI().PlaceSpreadOrder(request.WithVerbose(t.Context()), &PlaceSpreadOrderRequest{
			SpreadID:      spreadPair.String(),
			ClientOrderID: "b15",
			Side:          order.Buy.Lower(),
			OrderType:     "limit",
			Price:         2.15,
			Size:          2,
		})
		require.NoError(t, err, "PlaceSpreadOrder must not error")
		assert.NotNil(t, result, "PlaceSpreadOrder should return the order")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "b15", "ordId": "12345689", "tag": "", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().PlaceSpreadOrder(t.Context(), &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), ClientOrderID: "b15", Side: "buy", OrderType: "limit", Price: -2.15, Size: 2})
		require.NoError(t, err, "PlaceSpreadOrder must not error")
		assert.Equal(t, &SpreadOrderResponse{OrderID: "12345689", ClientOrderID: "b15"}, got, "PlaceSpreadOrder should decode the reply")
		frame := requireOneOperation(t, srv, "sprd-order", "/ws/v5/business")
		assert.JSONEq(t, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","clOrdId":"b15","side":"buy","ordType":"limit","sz":"2","px":"-2.15"}]`, string(frame.arguments), "PlaceSpreadOrder should send the order with its negative price")
	})

	t.Run("mock websocket rejection", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("1", []map[string]string{{"clOrdId": "", "ordId": "", "tag": "", "sCode": "51000", "sMsg": "Parameter sz error"}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().PlaceSpreadOrder(t.Context(), &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), Side: "buy", OrderType: "limit", Price: 1, Size: 2})
		require.ErrorIs(t, err, errOperationFailed, "PlaceSpreadOrder must report the rejected order")
		assert.ErrorContains(t, err, "Parameter sz error", "PlaceSpreadOrder should report the row's message")
	})
}

func TestWebsocketAPIAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().AmendSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendSpreadOrder must reject a nil request")
	_, err = e.WebsocketAPI().AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendSpreadOrder must reject an amendment without an order ID")
	_, err = e.WebsocketAPI().AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired, "AmendSpreadOrder must reject an amendment without a new size or price")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		result, err := e.WebsocketAPI().AmendSpreadOrder(request.WithVerbose(t.Context()), &AmendSpreadOrderRequest{
			OrderID: "2510789768709120",
			NewSize: 2,
		})
		require.NoError(t, err, "AmendSpreadOrder must not error")
		assert.NotNil(t, result, "AmendSpreadOrder should return the outcome")
	})

	t.Run("mock websocket", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "2510789768709120", "reqId": "b12344", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		got, err := ex.WebsocketAPI().AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{OrderID: "2510789768709120", RequestID: "b12344", NewSize: 2})
		require.NoError(t, err, "AmendSpreadOrder must not error")
		assert.Equal(t, &SpreadOrderResponse{OrderID: "2510789768709120", RequestID: "b12344"}, got, "AmendSpreadOrder should decode the reply")
		frame := requireOneOperation(t, srv, "sprd-amend-order", "/ws/v5/business")
		assert.JSONEq(t, `[{"ordId":"2510789768709120","reqId":"b12344","newSz":"2"}]`, string(frame.arguments), "AmendSpreadOrder should send the amendment")
	})

	t.Run("reprices to zero", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": "", "ordId": "2510789768709120", "reqId": "", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketAPI().AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{OrderID: "2510789768709120", NewPrice: new(0.0)})
		require.NoError(t, err, "AmendSpreadOrder must amend a spread order's price to zero")
		frame := requireOneOperation(t, srv, "sprd-amend-order", "/ws/v5/business")
		assert.JSONEq(t, `[{"ordId":"2510789768709120","newPx":"0"}]`, string(frame.arguments), "AmendSpreadOrder should send a zero price rather than leave it out")
	})
}

func TestWebsocketAPICancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CancelSpreadOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpreadOrder must reject a cancellation without an order ID")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		result, err := e.WebsocketAPI().CancelSpreadOrder(request.WithVerbose(t.Context()), "1234", "")
		require.NoError(t, err, "CancelSpreadOrder must not error")
		assert.NotNil(t, result, "CancelSpreadOrder should return the outcome")
	})

	for _, tc := range []struct {
		name, orderID, clientOrderID, expArgs string
	}{
		{"mock websocket by order ID", "2510789768709120", "", `[{"ordId":"2510789768709120"}]`},
		{"mock websocket by client order ID", "", "b15", `[{"clOrdId":"b15"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"clOrdId": tc.clientOrderID, "ordId": "2510789768709120", "sCode": "0", "sMsg": ""}})}
			ex := newWebsocketTestExchange(t, srv, true, nil)
			got, err := ex.WebsocketAPI().CancelSpreadOrder(t.Context(), tc.orderID, tc.clientOrderID)
			require.NoError(t, err, "CancelSpreadOrder must not error")
			assert.Equal(t, &SpreadOrderResponse{OrderID: "2510789768709120", ClientOrderID: tc.clientOrderID}, got, "CancelSpreadOrder should decode the reply")
			frame := requireOneOperation(t, srv, "sprd-cancel-order", "/ws/v5/business")
			assert.JSONEq(t, tc.expArgs, string(frame.arguments), "CancelSpreadOrder should send only the ID given")
		})
	}

	t.Run("refuses an account other than the connection's login", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{{"ordId": "2510789768709120", "sCode": "0", "sMsg": ""}})}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
		_, err := ex.WebsocketAPI().CancelSpreadOrder(ctx, "2510789768709120", "")
		require.ErrorIs(t, err, errWebsocketOtherAccount, "CancelSpreadOrder must refuse a cancel for another account than the connection logged in as")
		assert.Empty(t, srv.requests("sprd-cancel-order"), "no cancel should be sent")
	})
}

func TestWebsocketAPICancelAllSpreadOrders(t *testing.T) {
	t.Parallel()

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWebsocket(t, e)
		require.NoError(t, e.WebsocketAPI().CancelAllSpreadOrders(request.WithVerbose(t.Context()), spreadPair.String()), "CancelAllSpreadOrders must not error")
	})

	for _, tc := range []struct {
		name     string
		spreadID string
		result   bool
		expArgs  string
		expErr   error
	}{
		{"mock websocket one spread", spreadPair.String(), true, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}]`, nil},
		{"mock websocket every spread", "", true, `[{}]`, nil},
		{"mock websocket not cancelled", spreadPair.String(), false, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}]`, errCancelAllSpreadOrdersFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]bool{{"result": tc.result}})}
			ex := newWebsocketTestExchange(t, srv, true, nil)
			require.ErrorIs(t, ex.WebsocketAPI().CancelAllSpreadOrders(t.Context(), tc.spreadID), tc.expErr, "CancelAllSpreadOrders must report the reply's result")
			frame := requireOneOperation(t, srv, "sprd-mass-cancel", "/ws/v5/business")
			assert.JSONEq(t, tc.expArgs, string(frame.arguments), "CancelAllSpreadOrders should send the spread given")
		})
	}
}
