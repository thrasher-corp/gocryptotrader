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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
		{"GetSubAccountUnversalTransferHistory", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountUnversalTransferHistory(ctx, "", "", asset.Spot, asset.Spot, time.Time{}, time.Time{}, 0, 0)
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

	orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc}})
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
	orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{kas}})
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
