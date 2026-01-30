package bitget

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type ConnectionFixture struct {
	websocket.Connection
	match *websocket.Match
}

func (c *ConnectionFixture) RequireMatchWithData(signature any, incoming []byte) error {
	return c.match.RequireMatchWithData(signature, incoming)
}

func TestWebsocketSpotPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{Pair: testPair})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{Pair: testPair, OrderType: "limit"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{
		Pair:      testPair,
		OrderType: "limit",
		Side:      "buy",
	})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	_, err = e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{
		Pair:      testPair,
		OrderType: "limit",
		Side:      "buy",
		Size:      1,
	})
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	_, err = e.WebsocketSpotPlaceOrder(t.Context(), &WebsocketSpotPlaceOrderRequest{
		Pair:        testPair,
		OrderType:   "limit",
		Side:        "buy",
		Size:        1,
		TimeInForce: "gtc",
	})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	e := newExchangeWithWebsocket(t, asset.Spot)

	got, err := e.WebsocketSpotPlaceOrder(request.WithVerbose(t.Context()), &WebsocketSpotPlaceOrderRequest{
		Pair:        testPair,
		OrderType:   "limit",
		Side:        "buy",
		Size:        0.01,
		TimeInForce: "post_only",
		Price:       100,
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{Contract: testPair})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES",
	})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy",
	})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
	})
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc",
	})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc", Price: 100,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc", Price: 100, MarginCoin: currency.USDT,
	})
	require.ErrorIs(t, err, errMarginModeUnset)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc", Price: 100, MarginCoin: currency.USDT, MarginMode: "isolated",
	})
	require.ErrorIs(t, err, errMissingValues)

	_, err = e.WebsocketFuturesPlaceOrder(t.Context(), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc", Price: 100, MarginCoin: currency.USDT, MarginMode: "isolated", ReduceOnly: "NO",
		TradeSide: "open",
	})
	require.ErrorIs(t, err, errValuesConflict)

	e := newExchangeWithWebsocket(t, asset.Futures)

	_, err = e.WebsocketFuturesPlaceOrder(request.WithVerbose(t.Context()), &WebsocketFuturesOrderRequest{
		Contract: testPair, InstrumentType: "USDT-FUTURES", OrderType: "limit", Side: "buy", ContractSize: 1,
		TimeInForce: "gtc", Price: 50, MarginCoin: currency.USDT, MarginMode: "isolated", TradeSide: "open",
	})
	require.NoError(t, err)
}

func TestWebsocketSpotCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSpotCancelOrder(t.Context(), currency.EMPTYPAIR, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketSpotCancelOrder(t.Context(), testPair, "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	e := newExchangeWithWebsocket(t, asset.Spot)

	got, err := e.WebsocketSpotCancelOrder(request.WithVerbose(t.Context()), testPair, "1376695893517410304", "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelOrder(t.Context(), currency.EMPTYPAIR, "", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), testPair, "", "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), testPair, "USD-FUTURES", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	e := newExchangeWithWebsocket(t, asset.Futures)
	_, err = e.WebsocketFuturesCancelOrder(request.WithVerbose(t.Context()), testPair, "USDT-FUTURES", "1376999173366317057", "")
	require.NoError(t, err)
}

func TestWebsocketOrder_WSHandleData_Regression(t *testing.T) {
	t.Parallel()

	conn := &ConnectionFixture{match: websocket.NewMatch()}
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{
			name:    "spot place order success",
			payload: []byte(`{"event":"trade","arg":[{"id":"1","instType":"SPOT","channel":"place-order","instId":"BTCUSDT","params":{"orderId":"1376695893517410304","clientOid":"a0e0a25b-bd51-445c-affe-29bf823220d0"}}],"code":0,"msg":"Success","ts":1763955799473}`),
		},
		{
			name:    "spot place order failure",
			payload: []byte(`{"event":"error","arg":[{"id":"1","instType":"SPOT","channel":"place-order","instId":"BTCUSDT","params":{"orderType":"limit","side":"buy","force":"post_only","price":"100","size":"0.0001"}}],"code":43027,"msg":"The minimum order value 1 is not met","ts":1763953814046}`),
		},
		{
			name:    "spot cancel order failure",
			payload: []byte(`{"event":"error","arg":[{"id":"1","instType":"SPOT","channel":"cancel-order","instId":"BTCUSDT","params":{"orderId":"12345"}}],"code":43001,"msg":"The order does not exist","ts":1763958366418}`),
		},
		{
			name:    "spot cancel order success",
			payload: []byte(`{"event":"trade","arg":[{"id":"1","instType":"SPOT","channel":"cancel-order","instId":"BTCUSDT","params":{"orderId":"1376695893517410304"}}],"code":0,"msg":"Success","ts":1763958928351}`),
		},
		{
			name:    "futures place order failure",
			payload: []byte(`{"event":"error","arg":[{"id":"1","instType":"USDT-FUTURES","channel":"place-order","instId":"BTCUSDT","params":{"orderType":"limit","side":"buy","force":"gtc","price":"5","marginCoin":"USDT","size":"1","marginMode":"isolated"}}],"code":40774,"msg":"The order type for unilateral position must also be the unilateral position type","ts":1764022640768}`),
		},
		{
			name:    "futures place order success",
			payload: []byte(`{"event":"trade","arg":[{"id":"1","instType":"USDT-FUTURES","channel":"place-order","instId":"BTCUSDT","params":{"orderId":"1376999173366317057","clientOid":"1376999173370511360"}}],"code":0,"msg":"Success","ts":1764028107021}`),
		},
		{
			name:    "futures cancel order failure",
			payload: []byte(`{"event":"error","arg":[{"id":"1","instType":"USDT-FUTURES","channel":"cancel-order","instId":"BTCUSDT","params":{"orderId":"1234"}}],"code":41101,"msg":"Param orderId=1234 error","ts":1764129464757}`),
		},
		{
			name:    "futures cancel order success",
			payload: []byte(`{"event":"trade","arg":[{"id":"1","instType":"USDT-FUTURES","channel":"cancel-order","instId":"BTCUSDT","params":{"orderId":"1376999173366317057"}}],"code":0,"msg":"Success","ts":1764129519844}`),
		},
	} {
		incoming, err := conn.match.Set("1", 1)
		require.NoError(t, err)
		err = e.wsHandleData(t.Context(), conn, tc.payload)
		require.NoErrorf(t, err, "must not error %s", tc.name)
		require.Lenf(t, incoming, 1, "must capture response %s", tc.name)
		require.Equalf(t, tc.payload, <-incoming, "must match payload %s", tc.name)
	}
}

// newExchangeWithWebsocket returns a websocket instance copy for testing.
// This restricts the pairs to a single asset type to reduce test time.
func newExchangeWithWebsocket(t *testing.T, a asset.Item) *Exchange {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e := new(Exchange)
	e.IsDemoTrading = testingInSandbox // called before setup for websocket connection testing
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, e)
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(apiKey, apiSecret, clientID, "", "", "")
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)

	// Disable all other asset types to ensure only the specified asset type is used for websocket tests.
	for _, enabled := range e.GetAssetTypes(true) {
		if enabled != a {
			require.NoError(t, e.CurrencyPairs.SetAssetEnabled(enabled, false))
		}
	}

	require.NoError(t, e.Websocket.Connect(t.Context()), "Test instance Websocket Connect must not error")
	return e
}
