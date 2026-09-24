package mexc

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const (
	testCredentialKey    = "mock"
	testCredentialSecret = "tester"
)

func newSignedTestExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "test exchange Setup must not error")
	ex.SetCredentials(&accounts.Credentials{Key: testCredentialKey, Secret: testCredentialSecret})
	ex.GetBase().SkipAuthCheck = true
	// The limiters are shared by every instance and the server is local, so a test exchange does not
	// draw on the budget the other tests wait on.
	require.NoError(t, ex.Requester.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for k := range ex.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(k, server.URL), "SetRunningURL must not error for %s", k)
	}
	return ex
}

// TestSignatureIsHexEncoded pins the encoding of the request signature. MEXC's spot API follows the
// same scheme as Binance and expects the HMAC-SHA256 digest hex encoded; a base64 digest is rejected
// as an invalid signature, so the encoding is part of the contract rather than a free choice.
func TestSignatureIsHexEncoded(t *testing.T) {
	t.Parallel()
	var gotQuery url.Values
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	}))
	_, err := e.GetAccountInformation(t.Context())
	require.NoError(t, err, "GetAccountInformation must not error")

	signature := gotQuery.Get("signature")
	require.NotEmpty(t, signature, "signature query parameter must be set on an authenticated request")
	assert.Regexp(t, `^[0-9a-f]{64}$`, signature, "signature should be a hex encoded SHA256 digest, not base64")

	// Recompute the digest over the transmitted parameters, proving both that the signature covers
	// the request actually sent and that hex is the encoding applied to it.
	signed := url.Values{}
	for k, v := range gotQuery {
		if k != "signature" {
			signed[k] = v
		}
	}
	expected, err := crypto.GetHMAC(crypto.HashSHA256, []byte(signed.Encode()), []byte(testCredentialSecret))
	require.NoError(t, err, "GetHMAC must not error")
	assert.Equal(t, hex.EncodeToString(expected), signature, "signature should be the hex encoded HMAC-SHA256 of the transmitted query")
}

func TestPrivateEndpointRequestConstruction(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		call         func(ctx context.Context, e *Exchange) error
		method       string
		pathContains string
		body         string
	}{
		{"CreateBrokerSubAccount", func(ctx context.Context, e *Exchange) error {
			_, err := e.CreateBrokerSubAccount(ctx, &BrokerSubAccountCreationParams{SubAccount: "sub1", Note: "note"})
			return err
		}, http.MethodPost, "/broker/sub-account/virtualSubAccount", "{}"},
		{"GetBrokerAccountSubAccountList", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetBrokerAccountSubAccountList(ctx, "", 0, 0)
			return err
		}, http.MethodGet, "/broker/sub-account/list", "{}"},
		{"GetSubAccountStatus", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountStatus(ctx, "sub1")
			return err
		}, http.MethodGet, "/broker/sub-account/status", "{}"},
		{"GetSubAccountUniversalTransferHistory", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountUniversalTransferHistory(ctx, "", "", asset.Spot, asset.Spot, time.Time{}, time.Time{}, 0, 0)
			return err
		}, http.MethodGet, "/capital/sub-account/universalTransfer", "{}"},
		{"GetAccountInformation", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetAccountInformation(ctx)
			return err
		}, http.MethodGet, "/account", "{}"},
		{"GetOpenOrders", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetOpenOrders(ctx, spotTradablePair)
			return err
		}, http.MethodGet, "/openOrders", "[]"},
		{"GetOrderByID", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetOrderByID(ctx, spotTradablePair, "", "123")
			return err
		}, http.MethodGet, "/order", "{}"},
		{"CancelAllOpenOrdersBySymbol", func(ctx context.Context, e *Exchange) error {
			_, err := e.CancelAllOpenOrdersBySymbol(ctx, spotTradablePair)
			return err
		}, http.MethodDelete, "/openOrders", "[]"},
		{"GetSubAccountAsset", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountAsset(ctx, "sub1", asset.Spot)
			return err
		}, http.MethodGet, "/sub-account/asset", "{}"},
		{"GetDepositAddressOfCoin", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetDepositAddressOfCoin(ctx, currency.USDT, "")
			return err
		}, http.MethodGet, "/capital/deposit/address", "[]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotMethod, gotPath, gotAPIKey, gotSignature string
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAPIKey = r.Header.Get("X-MEXC-APIKEY")
				gotSignature = r.URL.Query().Get("signature")
				_, _ = w.Write([]byte(tc.body))
			}))
			err := tc.call(t.Context(), e)
			require.NoError(t, err)
			assert.Equal(t, tc.method, gotMethod, "request method should match the documented endpoint")
			assert.Contains(t, gotPath, tc.pathContains, "request path should target the documented endpoint")
			assert.Truef(t, strings.HasPrefix(gotPath, "/api/v3/"), "spot and broker endpoints should be versioned under /api/v3/, got %s", gotPath)
			assert.NotEmpty(t, gotAPIKey, "X-MEXC-APIKEY header should be set on an authenticated request")
			assert.NotEmpty(t, gotSignature, "signature query parameter should be set on an authenticated request")
		})
	}
}

// TestBatchOrderCreationParamMarshalsNumbersAsStrings pins the wire format of the batch order
// parameters. MEXC expects the numeric fields as quoted decimal strings; types.Number encodes them
// that way (and, unlike a float64 with json:",string", never falls back to exponent notation),
// while zero values stay omitted.
func TestBatchOrderCreationParamMarshalsNumbersAsStrings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		param    BatchOrderCreationParam
		expected string
	}{
		{
			name:     "populated",
			param:    BatchOrderCreationParam{OrderType: "LIMIT", Price: 1.23, Quantity: 0.5, Symbol: currency.NewBTCUSDT(), Side: "BUY"},
			expected: `{"type":"LIMIT","price":"1.23","quantity":"0.5","symbol":"BTCUSDT","side":"BUY"}`,
		},
		{
			name:     "zero numbers omitted",
			param:    BatchOrderCreationParam{OrderType: "LIMIT", QuoteOrderQty: 10, Symbol: currency.NewBTCUSDT(), Side: "BUY", NewClientOrderID: "7"},
			expected: `{"type":"LIMIT","quoteOrderQty":"10","symbol":"BTCUSDT","side":"BUY","newClientOrderId":"7"}`,
		},
		{
			name:     "large value stays decimal",
			param:    BatchOrderCreationParam{OrderType: "LIMIT", Price: 1e21, Quantity: 0.000001, Symbol: currency.NewBTCUSDT()},
			expected: `{"type":"LIMIT","price":"1000000000000000000000","quantity":"0.000001","symbol":"BTCUSDT"}`,
		},
		{
			name:     "self-trade prevention mode",
			param:    BatchOrderCreationParam{OrderType: "LIMIT", Price: 1, Quantity: 2, Symbol: currency.NewBTCUSDT(), Side: "SELL", SelfTradePreventionMode: "cancel_maker"},
			expected: `{"type":"LIMIT","price":"1","quantity":"2","symbol":"BTCUSDT","side":"SELL","stpMode":"cancel_maker"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.param)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.expected, string(got), "batch order parameters should marshal numeric fields as quoted decimal strings")
		})
	}
}

// TestGetOrderInfoAmountIsBaseQuantity pins the spot order amount mapping: origQty is the base
// quantity and belongs in Amount, while cummulativeQuoteQty is a quote figure and belongs in
// QuoteAmount. GetActiveOrders and GetOrderHistory already map the same payload this way.
func TestGetOrderInfoAmountIsBaseQuantity(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","orderId":"1","price":"20000","origQty":"0.5","executedQty":"0.2","cummulativeQuoteQty":"4000","status":"PARTIALLY_FILLED","type":"LIMIT","side":"BUY"}`))
	}))
	detail, err := e.GetOrderInfo(t.Context(), "1", spotTradablePair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, 0.5, detail.Amount, "Amount should carry origQty, the base quantity")
	assert.Equal(t, 4000.0, detail.QuoteAmount, "QuoteAmount should carry cummulativeQuoteQty")
	assert.Zero(t, detail.ContractAmount, "a spot order has no contract amount")
	assert.Equal(t, 0.2, detail.ExecutedAmount, "ExecutedAmount should carry executedQty")
	assert.InDelta(t, 0.3, detail.RemainingAmount, 1e-9, "RemainingAmount should be origQty minus executedQty")
}

// TestSubmitOrderPairFromRequest asserts SubmitOrder reports the pair the caller submitted rather than
// one re-parsed from the response symbol. The venue echoes the symbol concatenated without a
// delimiter, and a naive split mis-reads most MEXC symbols (METALUSDT read as MET/ALUSDT).
func TestSubmitOrderPairFromRequest(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"METALUSDT","orderId":"1","clientOrderId":"c1","price":"2.5","origQty":"10","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","transactTime":1736409765052}`))
	}))
	resp, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      currency.NewPair(currency.NewCode("METAL"), currency.USDT),
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    10,
		Price:     2.5,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, currency.NewCode("METAL"), resp.Pair.Base, "the response pair base should be METAL, not a mis-split of METALUSDT")
	assert.Equal(t, currency.USDT, resp.Pair.Quote, "the response pair quote should be USDT")
}

// TestSubmitOrderPlacedWhenStatusAbsent covers MEXC's create-order ACK, which carries an orderId but
// no status field (unlike Binance). A populated OrderID from a successful NewOrder means the order was
// placed, so the response must report a placed status and WasOrderPlaced() must be true — otherwise a
// filled market order is mis-read as never placed.
func TestSubmitOrderPlacedWhenStatusAbsent(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"KASUSDT","orderId":"C02__728298455591530497024","clientOrderId":"sbo000045","price":"0","origQty":"171.65","executedQty":"0","type":"MARKET","side":"BUY","transactTime":1736409765052}`))
	}))
	resp, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      currency.NewPair(currency.NewCode("KAS"), currency.USDT),
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Market,
		Amount:    171.65,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	require.NotEmpty(t, resp.OrderID, "the venue order id must be reported")
	assert.Equal(t, order.New, resp.Status, "an accepted order with an id should report a placed status, not UnknownStatus")
	// The engine derives its Detail from this SubmitResponse (DeriveDetail copies Status), and
	// order_placed on the gRPC boundary is Detail.WasOrderPlaced(): prove the status maps to placed.
	assert.True(t, (&order.Detail{Status: resp.Status}).WasOrderPlaced(), "the derived detail should report the order as placed")
}

// TestGetOrderInfoPairAndTimestamps asserts GetOrderInfo reports the requested pair (not one re-split
// from the concatenated response symbol) and both timestamps. The Query Order response carries time
// and updateTime but no transactTime (that field only exists on the New Order response), so reading
// LastUpdated from transactTime left both timestamps at the zero time.
func TestGetOrderInfoPairAndTimestamps(t *testing.T) {
	t.Parallel()
	metalUSDT := currency.NewPair(currency.NewCode("METAL"), currency.USDT)

	t.Run("both timestamps present", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"symbol":"METALUSDT","orderId":"1","clientOrderId":"c1","price":"2.5","origQty":"10","executedQty":"4","cummulativeQuoteQty":"10","type":"LIMIT","side":"BUY","status":"PARTIALLY_FILLED","time":1736409765000,"updateTime":1736409770000}`))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", metalUSDT, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Equal(t, currency.NewCode("METAL"), detail.Pair.Base, "Pair base should be METAL, not a mis-split of METALUSDT")
		assert.Equal(t, currency.USDT, detail.Pair.Quote, "Pair quote should be USDT")
		assert.Equal(t, int64(1736409765000), detail.Date.UnixMilli(), "Date should come from the order time")
		assert.Equal(t, int64(1736409770000), detail.LastUpdated.UnixMilli(), "LastUpdated should come from updateTime")
		assert.Equal(t, 10.0, detail.Cost, "Cost should carry the cumulative quote spent, not a zero")
	})

	t.Run("updateTime absent falls back to time", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"symbol":"METALUSDT","orderId":"1","price":"2.5","origQty":"10","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","time":1736409765000}`))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", metalUSDT, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.False(t, detail.LastUpdated.IsZero(), "LastUpdated should not be the zero time when updateTime is absent")
		assert.Equal(t, int64(1736409765000), detail.LastUpdated.UnixMilli(), "LastUpdated should fall back to the order time")
	})
}

// TestGetDepositAddressMultiNetwork asserts GetDepositAddress copes with the list the venue returns
// when no network is pinned (one address per network) by taking the first, and that the destination
// tag is read from memo with a fallback to tag. Rejecting anything but a single-element list dropped
// every multi-network coin.
func TestGetDepositAddressMultiNetwork(t *testing.T) {
	t.Parallel()

	t.Run("multi-network takes the first", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"coin":"USDT","network":"TRC20","address":"TAddr1"},{"coin":"USDT","network":"BEP20","address":"0xAddr2"},{"coin":"USDT","network":"ERC20","address":"0xAddr3"}]`))
		}))
		addr, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "")
		require.NoError(t, err, "GetDepositAddress must not reject a multi-network list")
		assert.Equal(t, "TAddr1", addr.Address, "the first address should be returned")
		assert.Equal(t, "TRC20", addr.Chain, "the chain should be the first entry's network")
	})

	t.Run("tag arrives as memo", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"coin":"EOS","network":"EOS","address":"eosaddr","memo":"MX10068"}]`))
		}))
		addr, err := e.GetDepositAddress(t.Context(), currency.NewCode("EOS"), "", "")
		require.NoError(t, err, "GetDepositAddress must not error")
		assert.Equal(t, "MX10068", addr.Tag, "the destination tag should be read from memo")
	})

	t.Run("empty list reports not found", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[]`))
		}))
		_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "")
		assert.ErrorIs(t, err, deposit.ErrAddressNotFound, "an empty address list should report not found")
	})
}

// TestGetActiveOrdersToleratesUncatalogedSymbol asserts that one order whose symbol is no longer in
// the available pairs (delisted with a working order, or a catalogue not yet refreshed) does not sink
// the whole listing. The catalogued order must still be returned.
func TestGetActiveOrdersToleratesUncatalogedSymbol(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"b1","price":"20000","origQty":"1","executedQty":"1","cummulativeQuoteQty":"20000","type":"LIMIT","side":"BUY","status":"FILLED","time":1736409765000},{"symbol":"DOGEUSDT","orderId":"d1","price":"0.1","origQty":"100","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","time":1736409765000}]`))
	}))
	btc := currency.NewBTCUSDT()
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, false), "storing available pairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, true), "storing enabled pairs must not error")

	orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not fail the whole listing because of one uncataloged symbol")
	require.NotEmpty(t, orders, "the catalogued order must still be returned")
	var found bool
	for i := range orders {
		if orders[i].OrderID == "b1" {
			found = true
			assert.Equal(t, 20000.0, orders[i].Cost, "Cost should carry the cumulative quote spent from the REST order")
		}
	}
	assert.True(t, found, "the catalogued BTCUSDT order should be present in the listing")
}

// TestUpdateOrderbookStampsVenueTime asserts the REST orderbook carries the venue's own timestamp and
// update id rather than being stamped with the local clock. The depth payload decodes both, but they
// were never copied onto the book, so Process fell back to time.Now() and a zero update id.
func TestUpdateOrderbookStampsVenueTime(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"lastUpdateId":123456,"bids":[["20000","1"]],"asks":[["20001","2"]],"timestamp":1736409765000}`))
	}))
	btc := currency.NewBTCUSDT()
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, false), "storing available pairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, true), "storing enabled pairs must not error")

	ob, err := e.UpdateOrderbook(t.Context(), btc, asset.Spot)
	require.NoError(t, err, "UpdateOrderbook must not error")
	assert.Equal(t, int64(123456), ob.LastUpdateID, "LastUpdateID should be read from the venue response")
	assert.Equal(t, int64(1736409765000), ob.LastUpdated.UnixMilli(), "LastUpdated should be the venue timestamp, not the local clock")
}

// TestCreateBatchOrderPartialRejection asserts a partially rejected batch does not report a rejected
// entry as a placed order. MEXC returns a mixed array where a rejected order carries code+msg in
// place of the order fields; decoding it into []*OrderDetail turned it into a zero-value order the
// caller could not tell from a success. A placed entry echoes the caller id as newClientOrderId, so
// the accepted order must still carry the client order id.
func TestCreateBatchOrderPartialRejection(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"ok1","newClientOrderId":"101","price":"20000","origQty":"1","type":"LIMIT","side":"BUY","status":"NEW"},{"newClientOrderId":"rej1","code":30002,"msg":"oversold"}]`))
	}))
	args := []BatchOrderCreationParam{
		{Symbol: currency.NewBTCUSDT(), Side: order.Buy.String(), OrderType: "LIMIT", Quantity: 1, Price: 20000, NewClientOrderID: "101"},
		{Symbol: currency.NewBTCUSDT(), Side: order.Sell.String(), OrderType: "LIMIT", Quantity: 1, Price: 21000},
	}
	orders, err := e.CreateBatchOrder(t.Context(), args)
	require.ErrorIs(t, err, errBatchOrderRejected, "a rejected batch entry must surface as an error")
	assert.Contains(t, err.Error(), "30002", "the rejection code should be reported")
	require.Len(t, orders, 1, "only the accepted order must be returned, not a zero-value stand-in for the rejected one")
	assert.Equal(t, "ok1", orders[0].OrderID, "the accepted order should be present")
	assert.Equal(t, "101", orders[0].ClientOrderID, "the accepted order should carry the echoed client order id")
}

// TestAuthRequestReSignsOnRetry asserts each attempt of an authenticated request signs a fresh
// timestamp. doRequest re-invokes the request builder on a rate-limit wait or 429; a timestamp minted
// once before the first attempt goes stale on the retry (recvWindow exceeded) and is rejected. It also
// pins that every attempt carries the API key header.
func TestAuthRequestReSignsOnRetry(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var timestamps, keys []string
	var calls int
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timestamps = append(timestamps, r.URL.Query().Get("timestamp"))
		keys = append(keys, r.Header.Get("X-MEXC-APIKEY"))
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	_, err := e.GetAccountInformation(t.Context())
	require.NoError(t, err, "the request must succeed after a retry")
	require.Len(t, timestamps, 2, "the 429 must have triggered exactly one retry")
	assert.NotEqual(t, timestamps[0], timestamps[1], "each attempt should sign a fresh timestamp, not reuse a stale one")
	assert.Equal(t, []string{testCredentialKey, testCredentialKey}, keys, "every attempt should carry the API key header")
}

// TestAuthRequestErrorWrapsTransport asserts an authenticated request failure keeps the underlying
// transport error matchable with errors.Is. Wrapping it with %v (and duplicating the wrap SendPayload
// already applies) severed the chain.
func TestAuthRequestErrorWrapsTransport(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":-1,"msg":"boom"}`))
	}))
	_, err := e.GetAccountInformation(t.Context())
	require.Error(t, err, "a 500 must surface as an error")
	assert.ErrorIs(t, err, request.ErrBadStatus, "the transport error should remain matchable with errors.Is")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "an authenticated request failure should still report as such")
}

// TestAuthRequestSignsQueryAndBody asserts the signature covers the query string plus the request
// body. MEXC signs totalParams = query string + body; signing the query alone means the three
// body-carrying broker callers sign something other than what they send.
func TestAuthRequestSignsQueryAndBody(t *testing.T) {
	t.Parallel()
	var (
		mu       sync.Mutex
		gotQuery url.Values
		gotBody  string
	)
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotQuery = r.URL.Query()
		gotBody = string(b)
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	arg := map[string]string{"note": "hello"}
	var result struct{}
	err := e.SendHTTPRequest(t.Context(), exchange.RestSpot, brokerEPL, http.MethodPost, "broker/sub-account/apiKey", url.Values{"symbol": {"BTCUSDT"}}, arg, &result, true)
	require.NoError(t, err, "SendHTTPRequest must not error")
	sig := gotQuery.Get("signature")
	require.NotEmpty(t, sig, "the signature must be present")
	require.NotEmpty(t, gotBody, "the request must carry a body")
	signed := url.Values{}
	for k, v := range gotQuery {
		if k != "signature" {
			signed[k] = v
		}
	}
	expected, err := crypto.GetHMAC(crypto.HashSHA256, []byte(signed.Encode()+gotBody), []byte(testCredentialSecret))
	require.NoError(t, err, "GetHMAC must not error")
	assert.Equal(t, hex.EncodeToString(expected), sig, "the signature should cover the query string plus the request body")
}

// TestTradeSideIsTakerSide pins trade sides to the taker. MEXC documents isBuyerMaker and m as "was
// the buyer the maker?", so true means the taker sold.
func TestTradeSideIsTakerSide(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/aggTrades") {
			_, _ = w.Write([]byte(`[{"p":"77234.92","q":"0.1","T":1789251250000,"m":true,"M":true},{"p":"77234.93","q":"0.1","T":1789251248000,"m":false,"M":true}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"price":"77234.92","qty":"0.1","time":1789251246575,"isBuyerMaker":true,"tradeType":"ASK"},{"price":"77234.93","qty":"0.1","time":1789251244778,"isBuyerMaker":false,"tradeType":"BID"}]`))
	}))
	btc := currency.NewBTCUSDT()
	recent, err := e.GetRecentTrades(t.Context(), btc, asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	require.Len(t, recent, 2, "GetRecentTrades must return both trades")
	assert.Equal(t, order.Sell, recent[0].Side, "a buyer-maker trade should be a taker sell")
	assert.Equal(t, order.Buy, recent[1].Side, "a seller-maker trade should be a taker buy")
	end := time.UnixMilli(1789251250000)
	historic, err := e.GetHistoricTrades(t.Context(), btc, asset.Spot, end.Add(-time.Minute), end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.Len(t, historic, 2, "GetHistoricTrades must return both trades")
	assert.Equal(t, order.Buy, historic[0].Side, "a seller-maker aggregate trade should be a taker buy")
	assert.Equal(t, order.Sell, historic[1].Side, "a buyer-maker aggregate trade should be a taker sell")
}

// TestGetHistoricTradesPagesTheWindow reads windows holding more trades than one request returns. The
// venue answers at most 1000 aggregated trades per request, newest first and stamped to the second, from
// a window of at most an hour whose bounds it reads to the second.
func TestGetHistoricTradesPagesTheWindow(t *testing.T) {
	t.Parallel()
	origin := time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)
	every := func(step time.Duration, per int, span time.Duration) []int64 {
		stamps := make([]int64, 0, int(span/step)*per)
		for at := origin; at.Before(origin.Add(span)); at = at.Add(step) {
			for range per {
				stamps = append(stamps, at.UnixMilli())
			}
		}
		return stamps
	}
	for _, tc := range []struct {
		name   string
		stamps []int64
		span   time.Duration
	}{
		// Pages end part-way through a second.
		{"three trades a second", every(time.Second, 3, 70*time.Minute), 70 * time.Minute},
		// Each hour fits in one page, so neighbouring hours meet at a second both could claim.
		{"one trade every ten seconds", every(10*time.Second, 1, 2*time.Hour), 2 * time.Hour},
		// The window starts part-way through a second holding more trades than a page, none of them in it.
		{"a full second before the window", append(every(time.Second, 1000, time.Second), origin.Add(time.Second).UnixMilli()), time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				from, _ := strconv.ParseInt(q.Get("startTime"), 10, 64)
				to, _ := strconv.ParseInt(q.Get("endTime"), 10, 64)
				limit, err := strconv.Atoi(q.Get("limit"))
				if err != nil {
					limit = 500
				}
				if to-from > time.Hour.Milliseconds() {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"msg":"More than 1 hours between startTime and endTime.","code":-1127}`))
					return
				}
				from, to = from/1000*1000, to/1000*1000
				var rows []string
				for i := len(tc.stamps) - 1; i >= 0 && len(rows) < limit; i-- {
					if tc.stamps[i] >= from && tc.stamps[i] <= to {
						rows = append(rows, `{"p":"`+strconv.Itoa(i)+`","q":"1","T":`+strconv.FormatInt(tc.stamps[i], 10)+`,"m":false,"M":true}`)
					}
				}
				_, _ = w.Write([]byte("[" + strings.Join(rows, ",") + "]"))
			}))
			// Starting and ending part-way through a second, as a caller's window may.
			start, end := origin.Add(250*time.Millisecond), origin.Add(tc.span+250*time.Millisecond)
			trades, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, start, end)
			require.NoError(t, err, "GetHistoricTrades must not error")
			var want []float64
			for i, ms := range tc.stamps {
				if ms >= start.UnixMilli() && ms <= end.UnixMilli() {
					want = append(want, float64(i))
				}
			}
			require.Equal(t, len(want), len(trades), "every trade in the window must be returned once")
			for i := range trades {
				if !assert.Equalf(t, want[i], trades[i].Price, "trade %d should be returned in order, oldest first", i) {
					break
				}
			}
			_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, time.Now().Add(-3*time.Minute), time.Now().Add(2*time.Hour))
			assert.NoError(t, err, "a window ending in the future should be read up to now")
		})
	}
}

// TestExtendListenKey asserts the user data stream keepalive is a PUT to userDataStream carrying the
// listen key. The private stream closes 60 minutes after creation unless a keepalive is sent, and
// nothing renewed it.
func TestExtendListenKey(t *testing.T) {
	t.Parallel()

	t.Run("builds the PUT request", func(t *testing.T) {
		t.Parallel()
		var (
			mu        sync.Mutex
			gotMethod string
			gotPath   string
			gotKey    string
			gotAPIKey string
		)
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotKey = r.URL.Query().Get("listenKey")
			gotAPIKey = r.Header.Get("X-MEXC-APIKEY")
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))
		}))
		require.NoError(t, e.ExtendListenKey(t.Context(), "LISTEN123"), "ExtendListenKey must not error")
		assert.Equal(t, http.MethodPut, gotMethod, "the keepalive should be a PUT")
		assert.Contains(t, gotPath, "/api/v3/userDataStream", "the keepalive should target userDataStream")
		assert.Equal(t, "LISTEN123", gotKey, "the keepalive should carry the listen key")
		assert.NotEmpty(t, gotAPIKey, "the keepalive should be authenticated")
	})

	t.Run("rejects an empty listen key", func(t *testing.T) {
		t.Parallel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		}))
		assert.ErrorIs(t, e.ExtendListenKey(t.Context(), ""), errListenKeyRequired, "an empty listen key should be rejected")
	})
}

// filledMarketOrderSDK77 is a captured Query Order response for a filled KASUSDT market buy: the price
// field (0.183503) is the last trade price, not the average, while cummulativeQuoteQty over executedQty
// (5.72522853 / 32.69) is the real average fill of 0.175137.
const filledMarketOrderSDK77 = `{"symbol":"KASUSDT","orderId":"C02__728298455591530497024","clientOrderId":"sbo000077","price":"0.183503","origQty":"32.69","executedQty":"32.69","cummulativeQuoteQty":"5.72522853","status":"FILLED","type":"MARKET","side":"BUY","time":1736409765000,"updateTime":1736409765000}`

// filledSpotOrderBody is a filled KASUSDT limit order (Query Order response) shared by the
// GetOrderInfo fee-enrichment subtests below.
const filledSpotOrderBody = `{"symbol":"KASUSDT","orderId":"1","price":"0.035","origQty":"200","executedQty":"200","cummulativeQuoteQty":"7","type":"LIMIT","side":"SELL","status":"FILLED","time":1736409765000,"updateTime":1736409770000}`

// TestAverageExecutedPrice pins the average fill derivation: cummulativeQuoteQty over executedQty, with
// a zero for an unfilled order rather than a divide by zero.
func TestAverageExecutedPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *OrderDetail
		want float64
	}{
		{"filled market order", &OrderDetail{ExecutedQty: 32.69, CummulativeQuoteQty: 5.72522853}, 0.175137},
		{"partial fill", &OrderDetail{ExecutedQty: 2, CummulativeQuoteQty: 0.34}, 0.17},
		{"unfilled order", &OrderDetail{ExecutedQty: 0, CummulativeQuoteQty: 0}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tc.want, averageExecutedPrice(tc.o), 1e-6, "average executed price mismatch")
		})
	}
}

// TestGetOrderInfoAverageExecutedPrice asserts GetOrderInfo reports the average fill (cummulativeQuoteQty
// over executedQty), not the price field, which on a filled market order is the last trade price.
func TestGetOrderInfoAverageExecutedPrice(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(filledMarketOrderSDK77))
	}))
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)
	detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.InDelta(t, 0.175137, detail.AverageExecutedPrice, 1e-6, "AverageExecutedPrice should be the average fill, not the price field")
	assert.Equal(t, 0.183503, detail.Price, "Price should still carry the reported price field")
}

// TestGetOrderInfoEnrichesVenueFee asserts GetOrderInfo reads the commission facts from myTrades for a
// filled order: the Query Order response carries no commission, so the fee amount and its currency come
// from the fills. The fee currency is taken from the fill (MEXC may charge in base, quote, or the MX
// token), never assumed, and is left unset when fills disagree.
func TestGetOrderInfoEnrichesVenueFee(t *testing.T) {
	t.Parallel()
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)

	// routeVenue serves the Query Order body on the order endpoint and the myTrades body on the trade
	// list endpoint, so one handler drives both REST calls GetOrderInfo makes for a filled order.
	routeVenue := func(order, trades string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "myTrades") {
				_, _ = w.Write([]byte(trades))
				return
			}
			_, _ = w.Write([]byte(order))
		}
	}

	t.Run("uniform commission asset aggregates fee and currency", func(t *testing.T) {
		t.Parallel()
		orderBody := filledSpotOrderBody
		// clientOrderId is a string on MEXC (e.g. "C02__…"), not a number — the fixture pins the
		// decode contract so a numeric field type would fail here.
		tradesBody := `[{"symbol":"KASUSDT","id":"t1","orderId":"1","clientOrderId":"C02__1","commission":"0.0035","commissionAsset":"USDT","isBuyer":false,"isMaker":true,"price":"0.035","qty":"100","quoteQty":"3.5","time":1736409770000},{"symbol":"KASUSDT","id":"t2","orderId":"1","clientOrderId":"C02__1","commission":"0.0035","commissionAsset":"USDT","isBuyer":false,"isMaker":false,"price":"0.035","qty":"100","quoteQty":"3.5","time":1736409770500}]`
		e := newSignedTestExchange(t, routeVenue(orderBody, tradesBody))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		require.Len(t, detail.Trades, 2, "both fills must be mapped onto the order")
		assert.InDelta(t, 0.007, detail.Fee, 1e-9, "Fee should be the sum of the fill commissions")
		assert.Equal(t, currency.USDT, detail.FeeAsset, "FeeAsset should be the fill commission asset")
		assert.Equal(t, "USDT", detail.Trades[0].FeeAsset, "the per-fill commission asset should be carried")
	})

	t.Run("mixed commission assets leave the aggregate currency unset", func(t *testing.T) {
		t.Parallel()
		orderBody := filledSpotOrderBody
		tradesBody := `[{"symbol":"KASUSDT","id":"t1","orderId":"1","commission":"0.0035","commissionAsset":"USDT","price":"0.035","qty":"100","quoteQty":"3.5","time":1736409770000},{"symbol":"KASUSDT","id":"t2","orderId":"1","commission":"0.1","commissionAsset":"MX","price":"0.035","qty":"100","quoteQty":"3.5","time":1736409770500}]`
		e := newSignedTestExchange(t, routeVenue(orderBody, tradesBody))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		require.Len(t, detail.Trades, 2, "both fills must be mapped even with mixed commission assets")
		assert.Zero(t, detail.Fee, "no aggregate fee should be reported when the fills are charged in different assets")
		assert.True(t, detail.FeeAsset.IsEmpty(), "the aggregate FeeAsset should be unset when fills disagree, never guessed")
		assert.InDelta(t, 0.0035, detail.Trades[0].Fee, 1e-9, "the per-fill commission should still be reported")
		assert.Equal(t, "USDT", detail.Trades[0].FeeAsset, "the per-fill commission asset should still be reported")
		assert.Equal(t, "MX", detail.Trades[1].FeeAsset, "the per-fill commission asset should still be reported")
	})

	t.Run("no fills leaves fee unenriched without a trade call", func(t *testing.T) {
		t.Parallel()
		orderBody := `{"symbol":"KASUSDT","orderId":"1","price":"0.035","origQty":"200","executedQty":"0","cummulativeQuoteQty":"0","type":"LIMIT","side":"SELL","status":"NEW","time":1736409765000}`
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.NotContains(t, r.URL.Path, "myTrades", "myTrades should not be called for an order with no fills")
			_, _ = w.Write([]byte(orderBody))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Empty(t, detail.Trades, "no trades should be attached when the order has not filled")
		assert.Zero(t, detail.Fee, "Fee should stay zero when there is nothing to enrich")
	})

	t.Run("myTrades failure does not sink the order lookup", func(t *testing.T) {
		t.Parallel()
		orderBody := filledSpotOrderBody
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "myTrades") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(orderBody))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "a myTrades failure must not fail the order lookup")
		assert.Empty(t, detail.Trades, "no trades should be attached when myTrades fails")
		assert.Equal(t, order.Filled, detail.Status, "the base order should still be reported")
	})

	t.Run("empty myTrades is retried before giving up", func(t *testing.T) {
		t.Parallel()
		// MEXC can mark an order filled a moment before its fills appear in myTrades. The first
		// lookup returns no fills; the retry finds them and the commission is materialised.
		orderBody := filledSpotOrderBody
		tradesBody := `[{"symbol":"KASUSDT","id":"t1","orderId":"1","commission":"0.007","commissionAsset":"USDT","price":"0.035","qty":"200","quoteQty":"7","time":1736409770000}]`
		var tradeCalls atomic.Int64
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "myTrades") {
				if tradeCalls.Add(1) == 1 {
					_, _ = w.Write([]byte(`[]`)) // first lookup: fills not visible yet
					return
				}
				_, _ = w.Write([]byte(tradesBody))
				return
			}
			_, _ = w.Write([]byte(orderBody))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.GreaterOrEqual(t, tradeCalls.Load(), int64(2), "an empty myTrades result should be retried")
		require.Len(t, detail.Trades, 1, "the retry must pick up the fill")
		assert.InDelta(t, 0.007, detail.Fee, 1e-9, "Fee should be materialised from the retried lookup")
		assert.Equal(t, currency.USDT, detail.FeeAsset, "FeeAsset should be materialised from the retried lookup")
	})

	t.Run("persistently empty myTrades retries once and gives up", func(t *testing.T) {
		t.Parallel()
		// The fills never surface. The lookup makes exactly one retry and then returns the base order
		// without commission rather than polling the venue repeatedly.
		orderBody := filledSpotOrderBody
		var tradeCalls atomic.Int64
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "myTrades") {
				tradeCalls.Add(1)
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(orderBody))
		}))
		detail, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
		require.NoError(t, err, "an empty myTrades result must not fail the order lookup")
		assert.Equal(t, int64(2), tradeCalls.Load(), "an empty lookup should be retried exactly once, not polled")
		assert.Empty(t, detail.Trades, "no trades should be attached when the fills never surface")
		assert.Zero(t, detail.Fee, "Fee should stay zero when there is nothing to enrich")
	})

	t.Run("cancellation during the retry wait abandons the lookup", func(t *testing.T) {
		t.Parallel()
		// The first lookup is empty; the context is cancelled during the retry wait, so the lookup
		// returns at once instead of running the wait out and does not make a second call.
		orderBody := filledSpotOrderBody
		var tradeCalls atomic.Int64
		served := make(chan struct{})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "myTrades") {
				tradeCalls.Add(1)
				_, _ = w.Write([]byte(`[]`))
				select {
				case <-served:
				default:
					close(served)
				}
				return
			}
			_, _ = w.Write([]byte(orderBody))
		}))
		var cancelledAt atomic.Int64
		go func() {
			<-served
			// Let the first lookup finish and enter the retry wait before cancelling it.
			time.Sleep(50 * time.Millisecond)
			cancelledAt.Store(time.Now().UnixNano())
			cancel()
		}()
		detail, err := e.GetOrderInfo(ctx, "1", kas, asset.Spot)
		// Measured from the cancellation rather than the start of the lookup: the absolute elapsed time
		// also carries the HTTP round trips and the 50ms hand-off, which a loaded runner inflates past
		// any ceiling. Time from cancel to return is ~0 when the wait is abandoned and the remainder of
		// the full second when it is not.
		sinceCancel := time.Since(time.Unix(0, cancelledAt.Load()))
		require.NoError(t, err, "a cancelled retry wait must not fail the order lookup")
		assert.Equal(t, int64(1), tradeCalls.Load(), "the lookup should stop at the first call when the wait is cancelled")
		assert.Less(t, sinceCancel, 500*time.Millisecond, "the retry wait should be abandoned as soon as the context is cancelled rather than run its full second")
		assert.Empty(t, detail.Trades, "no trades should be attached when the wait is cancelled")
	})
}

// TestGetActiveOrdersAverageExecutedPrice asserts the shared REST mapping reports the average fill for
// orders returned by GetActiveOrders.
func TestGetActiveOrdersAverageExecutedPrice(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[" + filledMarketOrderSDK77 + "]"))
	}))
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{kas}, false), "storing available pairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{kas}, true), "storing enabled pairs must not error")
	orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{kas}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, orders, 1, "the single order must be returned")
	assert.InDelta(t, 0.175137, orders[0].AverageExecutedPrice, 1e-6, "AverageExecutedPrice should be the average fill, not the price field")
}

// TestSubmitOrderReportsClientOrderID asserts SubmitOrder falls back to the request client id when the
// New Order ACK omits clientOrderId, which MEXC does not echo.
func TestSubmitOrderReportsClientOrderID(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"KASUSDT","orderId":"C02__728298455591530497024","price":"0","origQty":"171.65","executedQty":"0","type":"MARKET","side":"BUY","transactTime":1736409765052}`))
	}))
	resp, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      e.Name,
		Pair:          currency.NewPair(currency.NewCode("KAS"), currency.USDT),
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.Market,
		Amount:        171.65,
		ClientOrderID: "sbo000045",
	})
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, "sbo000045", resp.ClientOrderID, "the response should report the request client id when the ACK omits it")
}

// TestSubmitOrderMapsSideToVenueEnum asserts the order side is mapped to MEXC's BUY/SELL enum: a long
// side (Bid/Buy/Long) must reach the venue as BUY, not the order.Side string (BID/LONG), and an unset
// side must be rejected rather than sent as UNKNOWN.
func TestSubmitOrderMapsSideToVenueEnum(t *testing.T) {
	t.Parallel()
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)
	var sentSide string
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentSide = r.URL.Query().Get("side")
		_, _ = w.Write([]byte(`{"symbol":"KASUSDT","orderId":"1","clientOrderId":"c1","price":"0.03","origQty":"100","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","transactTime":1736409765052}`))
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange: e.Name, Pair: kas, AssetType: asset.Spot, Side: order.Bid, Type: order.Limit, Amount: 100, Price: 0.03,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, "BUY", sentSide, "a long side should reach MEXC as BUY, not BID")

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange: e.Name, Pair: kas, AssetType: asset.Spot, Side: order.UnknownSide, Type: order.Limit, Amount: 100, Price: 0.03,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "an unset side must be rejected, not sent as UNKNOWN")
}

// TestGetOrderInfoFillLookupAsksMaxPage asserts the commission fill lookup requests the documented
// maximum page so an order with more than the default ten fills does not report a short commission.
func TestGetOrderInfoFillLookupAsksMaxPage(t *testing.T) {
	t.Parallel()
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)
	var limit string
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "myTrades") {
			limit = r.URL.Query().Get("limit")
			_, _ = w.Write([]byte(`[{"symbol":"KASUSDT","id":"t1","orderId":"1","commission":"0.007","commissionAsset":"USDT","price":"0.035","qty":"200","quoteQty":"7","time":1736409770000}]`))
			return
		}
		_, _ = w.Write([]byte(filledSpotOrderBody))
	}))
	_, err := e.GetOrderInfo(t.Context(), "1", kas, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, "1000", limit, "the fill lookup should request the documented maximum page, not the default")
}

// TestGetOrderHistoryReportsTriggerPrice asserts the REST listing mapper reports stopPrice as the
// trigger price, the same as GetOrderInfo, so a stop order does not lose its trigger over the listing.
func TestGetOrderHistoryReportsTriggerPrice(t *testing.T) {
	t.Parallel()
	kas := currency.NewPair(currency.NewCode("KAS"), currency.USDT)
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"symbol":"KASUSDT","orderId":"1","price":"0.035","origQty":"100","executedQty":"0","stopPrice":"0.03","type":"LIMIT","side":"SELL","status":"NEW","time":1736409765000}]`))
	}))
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{kas}, false), "storing available pairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{kas}, true), "storing enabled pairs must not error")
	orders, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Pairs: currency.Pairs{kas}, Side: order.AnySide, Type: order.AnyType,
	})
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, orders, 1, "the order must be returned")
	assert.Equal(t, 0.03, orders[0].TriggerPrice, "the REST listing mapper should report stopPrice as the trigger price")
}

// TestCreateBrokerSubAccountRequestBody sends subAccount and note in the JSON body, as the broker docs
// require for virtualSubAccount.
func TestCreateBrokerSubAccountRequestBody(t *testing.T) {
	t.Parallel()
	var body []byte
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	_, err := e.CreateBrokerSubAccount(t.Context(), &BrokerSubAccountCreationParams{SubAccount: "sub1", Note: "note1"})
	require.NoError(t, err, "CreateBrokerSubAccount must not error")
	assert.JSONEq(t, `{"subAccount":"sub1","note":"note1"}`, string(body), "the body should carry subAccount and note")
}

// TestGenerateBrokerSubAccountDepositAddressRequestBody sends coin and network in the JSON body, as the
// broker docs require for deposit/subAddress.
func TestGenerateBrokerSubAccountDepositAddressRequestBody(t *testing.T) {
	t.Parallel()
	var body []byte
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	_, err := e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.USDT, Network: "TRC20"})
	require.NoError(t, err, "GenerateBrokerSubAccountDepositAddress must not error")
	assert.JSONEq(t, `{"coin":"USDT","network":"TRC20"}`, string(body), "the body should carry coin and network")
}

// TestKeepListenKeyAliveRenewsEachConnectionsOwnKey asserts that when subscriptions span more than one
// connection each connection's renewer renews its own listen key, not a shared last-minted slot, so no
// stream's key is left to expire. Reverting the renewer to a single shared key renews only one of the
// two here.
func TestKeepListenKeyAliveRenewsEachConnectionsOwnKey(t *testing.T) {
	prev := listenKeyKeepAliveInterval
	listenKeyKeepAliveInterval = 5 * time.Millisecond
	t.Cleanup(func() { listenKeyKeepAliveInterval = prev })

	var mu sync.Mutex
	renewed := map[string]int{}
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		renewed[r.URL.Query().Get("listenKey")]++
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go ex.keepListenKeyAlive(ctx, nil, "KEY_A")
	go ex.keepListenKeyAlive(ctx, nil, "KEY_B")
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return renewed["KEY_A"] > 0 && renewed["KEY_B"] > 0
	}, 2*time.Second, 5*time.Millisecond, "each connection's own listen key must be renewed")
	cancel()
	ex.Websocket.Wg.Wait()
}

// listenKeyTestConn stands in for a manager connection: WsConnect dials it, and its renewer asks whether
// it is still connected.
type listenKeyTestConn struct {
	websocket.Connection
	url    string
	closed atomic.Bool
}

func (c *listenKeyTestConn) SetURL(u string) { c.url = u }
func (c *listenKeyTestConn) GetURL() string  { return c.url }
func (c *listenKeyTestConn) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return nil
}
func (c *listenKeyTestConn) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {}
func (c *listenKeyTestConn) IsConnected() bool                                             { return !c.closed.Load() }

// TestKeepListenKeyAliveStopsWithItsConnection stops renewing a key once its own connection is closed. When
// a later connection fails, the manager rolls back the ones already made without closing ShutdownC, so a
// renewer watching only ShutdownC keeps the dead key alive until the next full shutdown.
func TestKeepListenKeyAliveStopsWithItsConnection(t *testing.T) {
	prev := listenKeyKeepAliveInterval
	listenKeyKeepAliveInterval = 5 * time.Millisecond
	t.Cleanup(func() { listenKeyKeepAliveInterval = prev })

	var renewed atomic.Int64
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"listenKey":"key-1"}`))
			return
		}
		renewed.Add(1)
		_, _ = w.Write([]byte(`{}`))
	}))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	conn := &listenKeyTestConn{url: "wss://one"}
	require.NoError(t, ex.WsConnect(t.Context(), conn), "WsConnect must not error")
	assert.Equal(t, "wss://one?listenKey=key-1", conn.url, "the connection should dial with its listen key")
	require.Eventually(t, func() bool { return renewed.Load() > 0 }, time.Second, 5*time.Millisecond, "the key must be renewed while its connection is open")

	conn.closed.Store(true)
	stopped := make(chan struct{})
	go func() {
		ex.Websocket.Wg.Wait()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		assert.Fail(t, "the renewer should stop once its connection is closed")
	}
}

// TestGenerateSubscriptionsExpandsConfiguredList asserts generateSubscriptions expands the configured
// subscription list rather than a hardcoded default: clearing the configured list yields none, where
// returning the package defaults would still yield some.
func TestGenerateSubscriptionsExpandsConfiguredList(t *testing.T) {
	t.Parallel()
	ex := newSignedTestExchange(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ex.Features.Subscriptions = subscription.List{}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	assert.Empty(t, subs, "an empty configured list should expand to no subscriptions, not the hardcoded defaults")
}

// TestDeleteAPIKeySubAccountSendsAPIKey pins the apiKey parameter: without it the request deletes by
// sub-account name alone, which is not what the caller asked for.
func TestDeleteAPIKeySubAccountSendsAPIKey(t *testing.T) {
	t.Parallel()
	var got url.Values
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(`{"subAccount":"SubAcc1"}`))
	}))
	_, err := ex.DeleteAPIKeySubAccount(t.Context(), "SubAcc1", "the-key")
	require.NoError(t, err, "DeleteAPIKeySubAccount must not error")
	assert.Equal(t, "the-key", got.Get("apiKey"), "the request should carry the apiKey being deleted")
	assert.Equal(t, "SubAcc1", got.Get("subAccount"), "the request should carry the sub-account name")
}

// TestOrderDetailDecodesSelfTradePrevention decodes the self-trade prevention fields the order
// queries carry: stpMode is the mode the order was placed with and cancelReason is stp_cancel when
// the venue cancelled it under that rule.
func TestOrderDetailDecodesSelfTradePrevention(t *testing.T) {
	t.Parallel()
	var o OrderDetail
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","orderId":"1","status":"CANCELED","type":"LIMIT","side":"BUY","stpMode":"cancel_taker","cancelReason":"stp_cancel"}`), &o), "Unmarshal must not error")
	assert.Equal(t, "cancel_taker", o.SelfTradePreventionMode, "SelfTradePreventionMode should carry the stpMode field")
	assert.Equal(t, "stp_cancel", o.CancelReason, "CancelReason should carry the cancelReason field")
}

// TestCreateBatchOrderAcceptsLimitOrderTypes sends the limit-family types the venue lists as order
// types (LIMIT_MAKER, IMMEDIATE_OR_CANCEL, FILL_OR_KILL) through the batch endpoint under the same
// quantity and price checks a single order gets.
func TestCreateBatchOrderAcceptsLimitOrderTypes(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{"LIMIT_MAKER", "IMMEDIATE_OR_CANCEL", "FILL_OR_KILL"} {
		t.Run(orderType, func(t *testing.T) {
			t.Parallel()
			var batch string
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				batch = r.URL.Query().Get("batchOrders")
				_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"1","orderListId":-1}]`))
			}))
			_, err := e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{{OrderType: orderType, Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1, Price: 2}})
			require.NoError(t, err, "CreateBatchOrder must not error")
			assert.Contains(t, batch, `"type":"`+orderType+`"`, "the batch should carry the order type on the wire")

			_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{{OrderType: orderType, Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1}})
			assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "a priceless limit-family order should be rejected before it is sent")
		})
	}
}

// TestNewOrderMarketParameters sends a market order with quantity or quoteOrderQty, never both and
// never a price: the venue takes one of the two amounts for a MARKET order and has no price for it.
func TestNewOrderMarketParameters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                           string
		orderType                      string
		quantity, quoteOrderQty, price float64
		expected                       url.Values
	}{
		{"market with both amounts", "MARKET", 1, 50, 0, url.Values{"quoteOrderQty": {"50"}}},
		{"market with a price", "MARKET", 1, 0, 25000, url.Values{"quantity": {"1"}}},
		{"market by quote", "MARKET", 0, 50, 25000, url.Values{"quoteOrderQty": {"50"}}},
		{"limit", "LIMIT", 1, 0, 25000, url.Values{"quantity": {"1"}, "price": {"25000"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got url.Values
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()
				_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","orderId":"1"}`))
			}))
			_, err := e.NewOrder(t.Context(), currency.NewBTCUSDT(), "", "BUY", tc.orderType, tc.quantity, tc.quoteOrderQty, tc.price)
			require.NoError(t, err, "NewOrder must not error")
			for _, field := range []string{"quantity", "quoteOrderQty", "price"} {
				assert.Equalf(t, tc.expected.Get(field), got.Get(field), "%s should be sent only when the order type takes it", field)
			}
		})
	}
}

// TestCancelAllOpenOrders cancels every open order of the account in one call. The venue confirms
// with code 200 and names no orders; any other code is a failure the caller must see.
func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	var method, path string
	body := `{"code":200,"msg":"success","timestamp":1778744778528}`
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(body))
	}))
	require.NoError(t, e.CancelAllOpenOrders(t.Context()), "CancelAllOpenOrders must not error on a confirmed cancel")
	assert.Equal(t, http.MethodDelete, method, "cancel-all should be a DELETE")
	assert.Equal(t, "/api/v3/order/all", path, "cancel-all should target the account-wide endpoint")

	body = `{"code":30000,"msg":"suspended","timestamp":1778744778528}`
	assert.ErrorIs(t, e.CancelAllOpenOrders(t.Context()), errCancelAllOrdersFailed, "an unconfirmed cancel-all should be reported as failed")
}

// TestCancelAllOrdersWithoutPairCancelsAccountWide routes a cancel-all with no pair to the
// account-wide endpoint, and one with a pair to the symbol endpoint.
func TestCancelAllOrdersWithoutPairCancelsAccountWide(t *testing.T) {
	t.Parallel()
	var paths []string
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/openOrders") {
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"9"}]`))
			return
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"success","timestamp":1778744778528}`))
	}))
	require.NoError(t, e.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error without a pair")
	assert.Empty(t, resp.Status, "the account-wide cancel names no orders")

	resp, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: spotTradablePair})
	require.NoError(t, err, "CancelAllOrders must not error with a pair")
	assert.Equal(t, map[string]string{"9": "cancelled"}, resp.Status, "the symbol cancel should report the cancelled order")
	assert.Equal(t, []string{"/api/v3/order/all", "/api/v3/openOrders"}, paths, "the empty pair should cancel account-wide and the pair by symbol")
}

// TestAccountPlatformAndSelfTradePreventionEndpoints pins the request and the decoding of the uid, API
// key, offline symbol, announcement, self-trade prevention group and listen key endpoints against the
// documented examples.
func TestAccountPlatformAndSelfTradePreventionEndpoints(t *testing.T) {
	t.Parallel()
	// liveChecks run instead of the recorded responses when built with -tags mock_test_off. They assert the
	// shape of the venue's answer. An endpoint without one changes account settings, so it is exercised
	// against its recorded response only.
	liveChecks := map[string]func(context.Context, *testing.T){
		"GetUID": func(ctx context.Context, t *testing.T) {
			t.Helper()
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			uid, err := e.GetUID(ctx)
			require.NoError(t, err, "GetUID must not error")
			assert.NotEmpty(t, uid, "GetUID should return the uid")
		},
		"GetAPIKeyInfo": func(ctx context.Context, t *testing.T) {
			t.Helper()
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			creds, err := e.GetCredentials(ctx)
			require.NoError(t, err, "GetCredentials must not error")
			info, err := e.GetAPIKeyInfo(ctx, creds.Key)
			require.NoError(t, err, "GetAPIKeyInfo must not error")
			assert.Equal(t, creds.Key, info.AccessKey, "AccessKey should be the key asked about")
			assert.NotEmpty(t, info.Status, "Status should be decoded")
			assert.NotEmpty(t, info.Permissions, "Permissions should be decoded")
			assert.False(t, info.CreateTime.IsZero(), "CreateTime should be decoded")
		},
		"GetOfflineSymbols": func(ctx context.Context, t *testing.T) {
			t.Helper()
			symbols, err := e.GetOfflineSymbols(ctx)
			require.NoError(t, err, "GetOfflineSymbols must not error")
			require.NotEmpty(t, symbols, "the venue must list offline symbols")
			for _, s := range symbols {
				assert.NotEmpty(t, s.Symbol, "Symbol should be decoded")
				assert.Containsf(t, []uint8{2, 3}, s.State, "%s State should be suspended or delisted", s.Symbol)
			}
		},
		"GetAnnouncements": func(ctx context.Context, t *testing.T) {
			t.Helper()
			pages, err := e.GetAnnouncements(ctx, "en-US", 1, 5)
			require.NoError(t, err, "GetAnnouncements must not error")
			require.NotEmpty(t, pages, "the announcement page must be decoded")
			assert.Positive(t, pages[0].TotalPage.Float64(), "TotalPage should be decoded")
			for _, d := range pages[0].Details {
				assert.NotEmpty(t, d.Title, "Title should be decoded")
				assert.False(t, d.PostTime.Time().IsZero(), "PostTime should be decoded")
			}
		},
		"GetListenKeys": func(ctx context.Context, t *testing.T) {
			t.Helper()
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			keys, err := e.GetListenKeys(ctx)
			require.NoError(t, err, "GetListenKeys must not error")
			assert.Positive(t, keys.Total, "Total should be decoded")
			assert.LessOrEqual(t, keys.Available, keys.Total, "Available should not exceed Total")
		},
		"CloseListenKey": func(ctx context.Context, t *testing.T) {
			t.Helper()
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			key, err := e.GenerateListenKey(ctx)
			require.NoError(t, err, "GenerateListenKey must not error")
			require.NoError(t, e.CloseListenKey(ctx, key), "CloseListenKey must not error")
		},
	}
	for _, tc := range []struct {
		name   string
		method string
		path   string
		query  url.Values
		body   string
		call   func(context.Context, *testing.T, *Exchange)
	}{
		{"GetUID", http.MethodGet, "/api/v3/uid", nil, `{"uid":"209302839"}`, func(ctx context.Context, t *testing.T, e *Exchange) {
			t.Helper()
			uid, err := e.GetUID(ctx)
			require.NoError(t, err, "GetUID must not error")
			assert.Equal(t, "209302839", uid, "GetUID should return the uid")
		}},
		{
			"GetAPIKeyInfo", http.MethodGet, "/api/v3/apiKeyInfo",
			url.Values{"accessKey": {"mx0npKfh57kEEVmyLa"}},
			`{"note":"note2","accessKey":"mx0npKfh57kEEVmyLa","createTime":"2026-09-14T18:53:36.000+00:00",` +
				`"permissions":"CONTRACT_ACCOUNT_READ,CONTRACT_DEAL_READ,SPOT_ACCOUNT_READ,SPOT_DEAL_WRITE,SPOT_DEAL_READ",` +
				`"remainingValidity":82,"ipWhiteList":"","status":"VALID"}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				info, err := e.GetAPIKeyInfo(ctx, "mx0npKfh57kEEVmyLa")
				require.NoError(t, err, "GetAPIKeyInfo must not error")
				assert.Equal(t, "mx0npKfh57kEEVmyLa", info.AccessKey, "AccessKey should be decoded from accessKey")
				assert.Equal(t, "VALID", info.Status, "Status should be decoded")
				assert.Equal(t, "CONTRACT_ACCOUNT_READ,CONTRACT_DEAL_READ,SPOT_ACCOUNT_READ,SPOT_DEAL_WRITE,SPOT_DEAL_READ", info.Permissions, "Permissions should be decoded")
				assert.Empty(t, info.IPWhiteList, "an empty IP whitelist should decode as empty")
				assert.Equal(t, 82.0, info.RemainingValidity.Float64(), "RemainingValidity should be decoded")
				assert.Equal(t, time.Date(2026, 9, 14, 18, 53, 36, 0, time.UTC), info.CreateTime.UTC(), "CreateTime should be decoded from the ISO 8601 timestamp")
			},
		},
		{
			"SetAPIKeyInfo", http.MethodPost, "/api/v3/apiKeyInfo",
			url.Values{"apiKey": {"mx0npKfh57kEEVmyLa"}, "ipWhiteList": {"1.1.1.1,2.2.2.2"}, "note": {"note2"}},
			`{"note":"note2","apikey":"mx0npKfh57kEEVmyLa","ipWhiteList":"1.1.1.1,2.2.2.2"}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				info, err := e.SetAPIKeyInfo(ctx, "mx0npKfh57kEEVmyLa", []string{"1.1.1.1", "2.2.2.2"}, "note2")
				require.NoError(t, err, "SetAPIKeyInfo must not error")
				assert.Equal(t, "1.1.1.1,2.2.2.2", info.IPWhiteList, "IPWhiteList should be decoded")
				assert.Equal(t, "mx0npKfh57kEEVmyLa", info.AccessKey, "AccessKey should be decoded from apikey")
			},
		},
		{
			"GetOfflineSymbols", http.MethodGet, "/api/v3/symbol/offline", nil, `{"data":[{"symbol":"LVNUSDT","state":3},{"symbol":"LOKAUSDT","state":3,"offlineTime":1724125694000}]}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				symbols, err := e.GetOfflineSymbols(ctx)
				require.NoError(t, err, "GetOfflineSymbols must not error")
				require.Len(t, symbols, 2, "both offline symbols must be decoded")
				assert.Equal(t, uint8(3), symbols[0].State, "State should be decoded")
				assert.True(t, symbols[0].OfflineTime.Time().IsZero(), "an absent offlineTime should decode to the zero time")
				assert.Equal(t, int64(1724125694000), symbols[1].OfflineTime.Time().UnixMilli(), "OfflineTime should be decoded")
			},
		},
		{
			"GetAnnouncements", http.MethodGet, "/api/v3/announcements",
			url.Values{"language": {"en-US"}, "page": {"2"}, "limit": {"5"}},
			`{"data":[{"details":[{"title":"t","url":"https://www.mexc.com/announcements/article/a","postTime":1790090755000,"language":"en-US"}],"totalPage":5549}],"code":0,"msg":"success"}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				pages, err := e.GetAnnouncements(ctx, "en-US", 2, 5)
				require.NoError(t, err, "GetAnnouncements must not error")
				require.Len(t, pages, 1, "the announcement page must be decoded")
				assert.Equal(t, 5549.0, pages[0].TotalPage.Float64(), "TotalPage should be decoded")
				require.Len(t, pages[0].Details, 1, "the announcement must be decoded")
				assert.Equal(t, int64(1790090755000), pages[0].Details[0].PostTime.Time().UnixMilli(), "PostTime should be decoded")
			},
		},
		{
			"CreateSelfTradePreventionGroup", http.MethodPost, "/api/v3/strategy/group",
			url.Values{"tradeGroupName": {"tradeGroupOne"}},
			`{"data":{"tradeGroupName":"tradeGroupOne","tradeGroupId":91,"createTime":1758043350000,"updateTime":1758043350000},"code":200,"msg":"success","timestamp":1758043350233}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				group, err := e.CreateSelfTradePreventionGroup(ctx, "tradeGroupOne")
				require.NoError(t, err, "CreateSelfTradePreventionGroup must not error")
				assert.Equal(t, int64(91), group.TradeGroupID.Int64(), "TradeGroupID should be decoded")
			},
		},
		{
			"GetSelfTradePreventionGroup", http.MethodGet, "/api/v3/strategy/group",
			url.Values{"tradeGroupName": {"tradeGroupOne"}},
			`{"data":[{"tradeGroupName":"tradeGroupOne","tradeGroupId":"91","tradeGroupUid":"1,2","createTime":1758043350000,"updateTime":1758043350000}],"code":200,"msg":"success","timestamp":1758044090972}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				groups, err := e.GetSelfTradePreventionGroup(ctx, "tradeGroupOne")
				require.NoError(t, err, "GetSelfTradePreventionGroup must not error")
				require.Len(t, groups, 1, "the group must be decoded")
				assert.Equal(t, int64(91), groups[0].TradeGroupID.Int64(), "a quoted TradeGroupID should be decoded")
				assert.Equal(t, "1,2", groups[0].TradeGroupUID, "TradeGroupUID should be decoded")
			},
		},
		{
			"DeleteSelfTradePreventionGroup", http.MethodDelete, "/api/v3/strategy/group",
			url.Values{"tradeGroupId": {"91"}},
			`{"data":true,"code":200,"msg":"success","timestamp":1758044399749}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				deleted, err := e.DeleteSelfTradePreventionGroup(ctx, "91")
				require.NoError(t, err, "DeleteSelfTradePreventionGroup must not error")
				assert.True(t, deleted, "DeleteSelfTradePreventionGroup should report the deletion")
			},
		},
		{
			"AddSelfTradePreventionGroupUIDs", http.MethodPost, "/api/v3/strategy/group/uid",
			url.Values{"tradeGroupId": {"92"}, "uid": {"49910594,49910595"}},
			`{"data":{"tradeGroupName":"1","tradeGroupId":92,"tradeGroupUid":"49910594,49910595","createTime":1758044671000,"updateTime":1758044777000},"code":200,"msg":"success","timestamp":1758044777023}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				group, err := e.AddSelfTradePreventionGroupUIDs(ctx, "92", []string{"49910594", "49910595"})
				require.NoError(t, err, "AddSelfTradePreventionGroupUIDs must not error")
				assert.Equal(t, "49910594,49910595", group.TradeGroupUID, "TradeGroupUID should be decoded")
			},
		},
		{
			"DeleteSelfTradePreventionGroupUIDs", http.MethodDelete, "/api/v3/strategy/group/uid",
			url.Values{"tradeGroupId": {"92"}, "uid": {"49910594"}},
			`{"data":true,"code":200,"msg":"success","timestamp":1758045403352}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				deleted, err := e.DeleteSelfTradePreventionGroupUIDs(ctx, "92", []string{"49910594"})
				require.NoError(t, err, "DeleteSelfTradePreventionGroupUIDs must not error")
				assert.True(t, deleted, "DeleteSelfTradePreventionGroupUIDs should report the removal")
			},
		},
		{
			"GetListenKeys", http.MethodGet, "/api/v3/userDataStream", nil, `{"total":200,"listenKey":["342e","c716"],"available":198}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				keys, err := e.GetListenKeys(ctx)
				require.NoError(t, err, "GetListenKeys must not error")
				assert.Equal(t, []string{"342e", "c716"}, keys.ListenKeys, "ListenKeys should be decoded")
				assert.Equal(t, uint64(198), keys.Available, "Available should be decoded")
			},
		},
		{
			"CloseListenKey", http.MethodDelete, "/api/v3/userDataStream",
			url.Values{"listenKey": {"KEY1"}},
			`{}`,
			func(ctx context.Context, t *testing.T, e *Exchange) {
				t.Helper()
				require.NoError(t, e.CloseListenKey(ctx, "KEY1"), "CloseListenKey must not error")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				live, ok := liveChecks[tc.name]
				if !ok {
					t.Skip("changes account settings, so it runs against the recorded response only")
				}
				live(t.Context(), t)
				return
			}
			var method, path string
			var query url.Values
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path, query = r.Method, r.URL.Path, r.URL.Query()
				_, _ = w.Write([]byte(tc.body))
			}))
			tc.call(t.Context(), t, e)
			assert.Equal(t, tc.method, method, "the request method should match the documented endpoint")
			assert.Equal(t, tc.path, path, "the request path should match the documented endpoint")
			for k := range tc.query {
				assert.Equalf(t, tc.query.Get(k), query.Get(k), "%s should be sent", k)
			}
		})
	}
}

// TestAccountPlatformAndSelfTradePreventionEndpointsRejectMissingParameters rejects a request the venue would refuse
// before it is sent.
func TestAccountPlatformAndSelfTradePreventionEndpointsRejectMissingParameters(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "no request should reach the venue")
	}))
	_, err := e.GetAPIKeyInfo(t.Context(), "")
	assert.ErrorIs(t, err, errAccessKeyRequired, "GetAPIKeyInfo should require an access key")
	_, err = e.SetAPIKeyInfo(t.Context(), "", []string{"1.1.1.1"}, "")
	assert.ErrorIs(t, err, errAPIKeyMissing, "SetAPIKeyInfo should require an API key")
	_, err = e.SetAPIKeyInfo(t.Context(), "k", nil, "")
	assert.ErrorIs(t, err, errIPWhiteListRequired, "SetAPIKeyInfo should require an IP address")
	_, err = e.SetAPIKeyInfo(t.Context(), "k", make([]string, 21), "")
	assert.ErrorIs(t, err, errTooManyIPAddresses, "SetAPIKeyInfo should refuse more than 20 IP addresses")
	for _, limit := range []int64{-5, 7, 105} {
		_, err = e.GetAnnouncements(t.Context(), "", 0, limit)
		assert.ErrorIsf(t, err, errInvalidPaginationLimit, "GetAnnouncements should refuse a limit of %d", limit)
	}
	_, err = e.CreateSelfTradePreventionGroup(t.Context(), "")
	assert.ErrorIs(t, err, errSelfTradePreventionGroupNameRequired, "CreateSelfTradePreventionGroup should require a name")
	_, err = e.GetSelfTradePreventionGroup(t.Context(), "")
	assert.ErrorIs(t, err, errSelfTradePreventionGroupNameRequired, "GetSelfTradePreventionGroup should require a name")
	_, err = e.DeleteSelfTradePreventionGroup(t.Context(), "")
	assert.ErrorIs(t, err, errSelfTradePreventionGroupIDRequired, "DeleteSelfTradePreventionGroup should require a group id")
	_, err = e.AddSelfTradePreventionGroupUIDs(t.Context(), "", []string{"1"})
	assert.ErrorIs(t, err, errSelfTradePreventionGroupIDRequired, "AddSelfTradePreventionGroupUIDs should require a group id")
	_, err = e.DeleteSelfTradePreventionGroupUIDs(t.Context(), "1", nil)
	assert.ErrorIs(t, err, errUIDRequired, "DeleteSelfTradePreventionGroupUIDs should require a uid")
	assert.ErrorIs(t, e.CloseListenKey(t.Context(), ""), errListenKeyRequired, "CloseListenKey should require a listen key")
}

// TestKeepListenKeyAliveClosesItsKey releases a connection's listen key once its renewer stops, so keys
// do not pile up against the account's limit across reconnects.
func TestKeepListenKeyAliveClosesItsKey(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var closed []string
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			mu.Lock()
			closed = append(closed, r.URL.Query().Get("listenKey"))
			mu.Unlock()
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		ex.keepListenKeyAlive(ctx, nil, "KEY_A")
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "the renewer must stop once its context is cancelled")
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"KEY_A"}, closed, "the renewer should close its own listen key when it stops")
}

// failingDialConn is a connection whose dial always fails
type failingDialConn struct {
	websocket.Connection
	url string
}

func (c *failingDialConn) GetURL() string  { return c.url }
func (c *failingDialConn) SetURL(u string) { c.url = u }
func (c *failingDialConn) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return errors.New("dial refused")
}

// TestWsConnectReleasesListenKeyOnFailedDial releases the listen key minted for a connection whose dial
// fails: no renewer owns it yet, and the monitor retries a failed connect every few seconds.
func TestWsConnectReleasesListenKeyOnFailedDial(t *testing.T) {
	t.Parallel()
	keys := []string{"KEY_A", "KEY_B", "KEY_C"}
	var mu sync.Mutex
	var minted, closed []string
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			key := keys[len(minted)]
			minted = append(minted, key)
			_, _ = w.Write([]byte(`{"listenKey":"` + key + `"}`))
		case http.MethodDelete:
			closed = append(closed, r.URL.Query().Get("listenKey"))
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	for range 3 {
		assert.Error(t, ex.WsConnect(t.Context(), &failingDialConn{url: spotWebsocketURL}), "WsConnect should report the failed dial")
	}
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, minted, 3, "each attempt must mint a listen key")
	assert.Equal(t, minted, closed, "each minted listen key should be released when its dial fails")
}

// TestCreateBatchOrderMarketParameters applies the single-order market rules to each batch entry: no
// price, and a quote amount sent alone when one is given.
func TestCreateBatchOrderMarketParameters(t *testing.T) {
	t.Parallel()
	var batch string
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		batch = r.URL.Query().Get("batchOrders")
		_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"1","orderListId":-1},{"symbol":"BTCUSDT","orderId":"2","orderListId":-1},{"symbol":"BTCUSDT","orderId":"3","orderListId":-1}]`))
	}))
	_, err := e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{
		{OrderType: "MARKET", Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1, QuoteOrderQty: 50, Price: 25000},
		{OrderType: "MARKET", Symbol: currency.NewBTCUSDT(), Side: "SELL", Quantity: 1, Price: 25000},
		{OrderType: "LIMIT", Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1, Price: 25000},
	})
	require.NoError(t, err, "CreateBatchOrder must not error")
	assert.JSONEq(t, `[`+
		`{"type":"MARKET","quoteOrderQty":"50","symbol":"BTCUSDT","side":"BUY"},`+
		`{"type":"MARKET","quantity":"1","symbol":"BTCUSDT","side":"SELL"},`+
		`{"type":"LIMIT","price":"25000","quantity":"1","symbol":"BTCUSDT","side":"BUY"}]`,
		batch, "market entries should carry no price and a quote amount alone")
}

// TestAPIKeyInfoCreateTime decodes the key creation time in either form it is published in: an
// RFC 3339 timestamp with milliseconds and an offset, or epoch milliseconds.
func TestAPIKeyInfoCreateTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, raw string
		expected  time.Time
	}{
		{"rfc3339", `"2026-09-14T18:53:36.000+00:00"`, time.Date(2026, 9, 14, 18, 53, 36, 0, time.UTC)},
		{"rfc3339 with offset", `"2026-09-15T02:53:36.250+08:00"`, time.Date(2026, 9, 14, 18, 53, 36, 250e6, time.UTC)},
		{"epoch milliseconds string", `"1758043350000"`, time.UnixMilli(1758043350000).UTC()},
		{"epoch milliseconds number", `1758043350000`, time.UnixMilli(1758043350000).UTC()},
		{"absent", `null`, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var info APIKeyInfo
			require.NoError(t, json.Unmarshal([]byte(`{"accessKey":"k","status":"VALID","createTime":`+tc.raw+`}`), &info), "Unmarshal must not error")
			assert.Truef(t, tc.expected.Equal(info.CreateTime), "CreateTime should be %v, got %v", tc.expected, info.CreateTime)
			assert.Equal(t, "k", info.AccessKey, "the other fields should still be decoded")
			assert.Equal(t, "VALID", info.Status, "the other fields should still be decoded")
		})
	}
	var info APIKeyInfo
	assert.Error(t, json.Unmarshal([]byte(`{"createTime":"yesterday"}`), &info), "an unreadable creation time should be reported")
}

// subscriptionTestConn answers each subscription request with the reply configured for its channel.
type subscriptionTestConn struct {
	websocket.Connection
	replies map[string]string
}

func (c *subscriptionTestConn) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, req any) ([]byte, error) {
	p, ok := req.(*WsSubscriptionPayload)
	if !ok || len(p.Params) != 1 {
		return nil, errors.New("unexpected subscription payload")
	}
	return []byte(c.replies[p.Params[0]]), nil
}

// TestHandleSubscriptionKeepsAcceptedWhenOneIsRejected registers the accepted subscriptions when another
// in the same request is rejected, and names the rejected one in the error. A rejected subscription was
// never stored, so removing it failed with subscription.ErrNotFound before the accepted ones were added.
func TestHandleSubscriptionKeepsAcceptedWhenOneIsRejected(t *testing.T) {
	t.Parallel()
	ex := newSignedTestExchange(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	accepted := &subscription.Subscription{Channel: channelLimitDepthV3, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Levels: 5, QualifiedChannel: "spot@public.limit.depth.v3.api.pb@BTCUSDT@5"}
	refused := &subscription.Subscription{Channel: channelLimitDepthV3, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}, Levels: 50, QualifiedChannel: "spot@public.limit.depth.v3.api.pb@ETHUSDT@50"}
	conn := &subscriptionTestConn{replies: map[string]string{
		accepted.QualifiedChannel: `{"id":0,"code":0,"msg":"` + accepted.QualifiedChannel + `"}`,
		refused.QualifiedChannel:  `{"id":0,"code":0,"msg":"Not Subscribed successfully! [` + refused.QualifiedChannel + `]. Reason： Blocked!"}`,
	}}
	err := ex.handleSubscription(t.Context(), conn, "SUBSCRIPTION", subscription.List{accepted, refused})
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "a rejected subscription must be reported")
	assert.ErrorContains(t, err, "ETH/USDT", "the error should name the rejected subscription")
	assert.NotContains(t, err.Error(), "BTC/USDT", "the error should not name the accepted subscription")
	got := ex.Websocket.GetSubscriptions()
	require.Len(t, got, 1, "only the accepted subscription must be stored")
	assert.Equal(t, accepted.QualifiedChannel, got[0].QualifiedChannel, "the accepted subscription should be stored")
	assert.Equal(t, subscription.SubscribedState, got[0].State(), "the accepted subscription should be marked subscribed")
}

// TestTickerListUnmarshalJSON decodes the 24hr ticker in both of the shapes the endpoint returns: an
// object for a single symbol and an array otherwise. The first byte decides which is decoded, so a
// malformed array reports its own fault rather than a failed retry as a single object.
func TestTickerListUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		want        []string
	}{
		{"object", `{"symbol":"BTCUSDT","lastPrice":"1"}`, []string{"BTCUSDT"}},
		{"object after whitespace", " \n\t{\"symbol\":\"BTCUSDT\"}", []string{"BTCUSDT"}},
		{"array", `[{"symbol":"BTCUSDT"},{"symbol":"ETHUSDT"}]`, []string{"BTCUSDT", "ETHUSDT"}},
		{"empty array", `[]`, []string{}},
		{"null", `null`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var list TickerList
			require.NoError(t, json.Unmarshal([]byte(tc.input), &list), "Unmarshal must not error")
			got := make([]string, 0, len(list))
			for i := range list {
				got = append(got, list[i].Symbol)
			}
			if tc.want == nil {
				assert.Empty(t, list, "null should decode to an empty list")
				return
			}
			assert.Equal(t, tc.want, got, "the decoded symbols should match")
		})
	}
	var list TickerList
	err := json.Unmarshal([]byte(`[{"symbol":1}]`), &list)
	require.Error(t, err, "a malformed array element must be reported")
	assert.NotContains(t, err.Error(), "array", "the error should describe the malformed element, not a retry as a single object")
}

// TestCurrencyFieldsDecodeAsCodes decodes the currency fields of the responses into currency codes, so
// callers compare them with Code.Equal rather than by string.
func TestCurrencyFieldsDecodeAsCodes(t *testing.T) {
	t.Parallel()
	var symbol SymbolDetail
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT"}`), &symbol), "Unmarshal must not error")
	assert.True(t, symbol.BaseAsset.Equal(currency.BTC), "BaseAsset should decode as BTC")
	assert.True(t, symbol.QuoteAsset.Equal(currency.USDT), "QuoteAsset should decode as USDT")

	var fill AccountTrade
	require.NoError(t, json.Unmarshal([]byte(`{"commissionAsset":"MX"}`), &fill), "Unmarshal must not error")
	assert.True(t, fill.CommissionAsset.Equal(currency.MX), "CommissionAsset should decode as MX")

	var info CurrencyInformation
	require.NoError(t, json.Unmarshal([]byte(`{"coin":"usdt","networkList":[{"coin":"USDT","netWork":"TRX"}]}`), &info), "Unmarshal must not error")
	assert.True(t, info.Coin.Equal(currency.USDT), "Coin should match regardless of case")
}

// TestSignedFieldsDecodeNegatives decodes the negative values the fields left signed carry: orderListId
// is -1 for an order outside an order list and the venue's error codes include -1121.
func TestSignedFieldsDecodeNegatives(t *testing.T) {
	t.Parallel()
	var o OrderDetail
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":"1","orderListId":-1}`), &o), "Unmarshal must not error")
	assert.Equal(t, int64(-1), o.OrderListID, "OrderDetail.OrderListID should decode -1")
	var fill AccountTrade
	require.NoError(t, json.Unmarshal([]byte(`{"id":"1","orderListId":-1}`), &fill), "Unmarshal must not error")
	assert.Equal(t, int64(-1), fill.OrderListID, "AccountTrade.OrderListID should decode -1")
	var r BatchOrderResult
	require.NoError(t, json.Unmarshal([]byte(`{"code":-1121,"msg":"Invalid symbol."}`), &r), "Unmarshal must not error")
	assert.Equal(t, int64(-1121), r.Code, "a negative error code should decode")
}

// TestAPIKeyInfoPermanentKey decodes the remaining validity of a key that never expires, which the venue
// documents as -999, whether it is sent as a number or as a string.
func TestAPIKeyInfoPermanentKey(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`-999`, `"-999"`} {
		var info APIKeyInfo
		require.NoErrorf(t, json.Unmarshal([]byte(`{"accessKey":"k","status":"VALID","remainingValidity":`+raw+`}`), &info), "Unmarshal must not error for %s", raw)
		assert.Equalf(t, -999.0, info.RemainingValidity.Float64(), "a permanent key's remaining validity should decode as -999 from %s", raw)
	}
}

// TestInternalTransferDecodesTransferID decodes the documented response, whose tranId is a string
func TestInternalTransferDecodesTransferID(t *testing.T) {
	t.Parallel()
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tranId":"c45d800a47ba4cbc876a5cd29388319"}`))
	}))
	resp, err := ex.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "", currency.USDT, 1)
	require.NoError(t, err, "InternalTransfer must not error")
	assert.Equal(t, "c45d800a47ba4cbc876a5cd29388319", resp.TransferID, "TransferID should be decoded")
}

// TestBrokerSubAccountDepositDetailDecodesDocumentedExample decodes the all-sub-accounts deposit history
// example MEXC documents, which names the sub-account and carries the deposit's time and memo.
func TestBrokerSubAccountDepositDetailDecodesDocumentedExample(t *testing.T) {
	t.Parallel()
	var d BrokerSubAccountDepositDetail
	require.NoError(t, json.Unmarshal([]byte(`{"subAccount":"2896a8a258b84b7fb27eb7f076dcd91c","amount":"4.990000000000000000000000000000","coin":"USDT-BSC","network":"BNB Smart Chain(BEP20)","status":5,"address":"0x08b65ab99e2576d5bf30f6553f55661af47329ce","txId":"0x8ac3ecac6e53201037dd6394697132ae2f9eef176274e4e7f87ac3005a72921a:68","unlockConfirm":"61","confirmTimes":"136","insertTime":1779361732000,"netWork":"BSC","memo":""}`), &d), "Unmarshal must not error")
	assert.Equal(t, "2896a8a258b84b7fb27eb7f076dcd91c", d.SubAccount, "SubAccount should name the sub-account the deposit went to")
	assert.Equal(t, int64(1779361732000), d.InsertTime.Time().UnixMilli(), "InsertTime should carry the deposit's time")
	assert.Empty(t, d.Memo, "Memo should carry the memo")
	assert.Equal(t, 4.99, d.Amount.Float64(), "Amount should be decoded")
}

// TestCampaignDataClickCount decodes clickTime as the click count MEXC documents it to be
func TestCampaignDataClickCount(t *testing.T) {
	t.Parallel()
	var c CampaignData
	require.NoError(t, json.Unmarshal([]byte(`{"campaign":"11kd","clickTime":7,"createTime":1695125287000}`), &c), "Unmarshal must not error")
	assert.Equal(t, uint64(7), c.ClickCount, "ClickCount should carry the clickTime count")
}
