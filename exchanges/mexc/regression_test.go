package mexc

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
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
// once before the first attempt goes stale on the retry (recvWindow exceeded) and is rejected. group
// T defect #11a.
func TestAuthRequestReSignsOnRetry(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var timestamps []string
	var calls int
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timestamps = append(timestamps, r.URL.Query().Get("timestamp"))
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
	err := e.SendHTTPRequest(t.Context(), exchange.RestSpot, request.Auth, http.MethodPost, "broker/sub-account/apiKey", url.Values{"symbol": {"BTCUSDT"}}, arg, &result, true)
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
	end := time.Now()
	historic, err := e.GetHistoricTrades(t.Context(), btc, asset.Spot, end.Add(-time.Minute), end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.Len(t, historic, 2, "GetHistoricTrades must return both trades")
	assert.Equal(t, order.Sell, historic[0].Side, "a buyer-maker aggregate trade should be a taker sell")
	assert.Equal(t, order.Buy, historic[1].Side, "a seller-maker aggregate trade should be a taker buy")
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
