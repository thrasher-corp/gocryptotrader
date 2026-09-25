package okx

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// writeOKXData writes a successful OKX JSON response carrying data.
func writeOKXData(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{"code": "0", "msg": "", "data": data})
	if err != nil {
		t.Errorf("marshalling mock response should not error: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encoded)
}

// newMockExchange returns an exchange whose REST endpoints all point at a test
// server running h.
func newMockExchange(t *testing.T, h http.Handler) *Exchange {
	t.Helper()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	srv := httptest.NewTestServer(t, h)
	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}
	return e
}

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.Parse(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, byte(7), u[6]>>4, "MessageID must return a V7 uuid") // RFC 9562 version nibble
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

func TestGetInstrumentIDCode(t *testing.T) {
	t.Parallel()

	// A detached instance keeps the seeded cache isolated from parallel tests
	// sharing the package-level exchange.
	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]Instrument)
	fresh.instrumentIDCodeMap = map[string]uint64{
		"BTC-USDT":      12345,
		"BTC-USDT-SWAP": 67890,
	}

	testCases := []struct {
		instrumentID string
		expectedCode uint64
	}{
		{instrumentID: "BTC-USDT", expectedCode: 12345},
		{instrumentID: "BTC-USDT-SWAP", expectedCode: 67890},
		{instrumentID: "", expectedCode: 0},
		{instrumentID: "NOT-CACHED", expectedCode: 0},
	}
	for _, tc := range testCases {
		got := fresh.getInstrumentIDCode(tc.instrumentID)
		assert.Equalf(t, tc.expectedCode, got, "instrument ID code for %q should match the cached value", tc.instrumentID)
	}
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

func TestCacheInstrumentsIndexesIDCodes(t *testing.T) {
	t.Parallel()

	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]Instrument)
	fresh.instrumentIDCodeMap = make(map[string]uint64)

	ethUSDT := currency.NewPairWithDelimiter(currency.ETH.String(), currency.USDT.String(), currency.DashDelimiter)
	solUSDT := currency.NewPairWithDelimiter(currency.SOL.String(), currency.USDT.String(), currency.DashDelimiter)

	fresh.cacheInstruments(instTypeSpot, []Instrument{
		{InstrumentID: mainPair, InstrumentIDCode: 12345},
		{InstrumentID: ethUSDT, InstrumentIDCode: 67890},
	})
	assert.EqualValues(t, 12345, fresh.getInstrumentIDCode("BTC-USDT"), "cached BTC-USDT should resolve its instrument ID code")
	assert.EqualValues(t, 67890, fresh.getInstrumentIDCode("ETH-USDT"), "cached ETH-USDT should resolve its instrument ID code")

	// Instruments channel pushes carry deltas, so they upsert codes without
	// replacing the full instrument lists.
	fresh.cacheInstrumentIDCodes([]Instrument{
		{InstrumentID: solUSDT, InstrumentIDCode: 11111},
	})
	assert.EqualValues(t, 11111, fresh.getInstrumentIDCode("SOL-USDT"), "a pushed SOL-USDT should upsert its instrument ID code")
	assert.Len(t, fresh.instrumentsInfoMap[instTypeSpot], 2, "an instruments push should not replace the cached instrument list")

	// A push omitting instIdCode decodes to zero and drops the cached code, so
	// a relisted instrument's orders go over REST rather than with its replaced
	// code.
	fresh.cacheInstrumentIDCodes([]Instrument{
		{InstrumentID: mainPair},
		{InstrumentID: solUSDT, InstrumentIDCode: 22222},
	})
	assert.Zero(t, fresh.getInstrumentIDCode("BTC-USDT"), "a push without a code should drop the cached instrument ID code")
	assert.EqualValues(t, 22222, fresh.getInstrumentIDCode("SOL-USDT"), "a push with a code should still upsert it")

	assert.Zero(t, fresh.getInstrumentIDCode("DOGE-USDT"), "an uncached instrument ID should resolve a zero code")
}

func TestCancelAllOrdersMatchesOnlyRequestedOrders(t *testing.T) {
	// Subtests share the mock server's pending order queue, so they run
	// sequentially.
	var mu sync.Mutex
	var pending []map[string]string
	// pages, when set, serves the pending order list keyed by the after
	// pagination parameter, with "" naming the first page.
	var pages map[string][]map[string]string
	var cancelled []CancelOrderRequestParam
	var pendingQuery string

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			after := r.URL.Query().Get("after")
			mu.Lock()
			defer mu.Unlock()
			pendingQuery = r.URL.RawQuery
			if pages != nil {
				writeOKXData(t, w, pages[after])
				return
			}
			writeOKXData(t, w, pending)
		case "/trade/cancel-batch-orders":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading cancel request body should not error: %v", err)
				return
			}
			var reqs []CancelOrderRequestParam
			if err := json.Unmarshal(body, &reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			respData := make([]map[string]string, 0, len(reqs))
			for x := range reqs {
				respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			mu.Lock()
			cancelled = append(cancelled, reqs...)
			mu.Unlock()
			writeOKXData(t, w, respData)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))

	setPending := func(ords ...map[string]string) {
		mu.Lock()
		defer mu.Unlock()
		pending = ords
		pages = nil
	}
	setPages := func(p map[string][]map[string]string) {
		mu.Lock()
		defer mu.Unlock()
		pages = p
	}
	resetCancelled := func() {
		mu.Lock()
		defer mu.Unlock()
		cancelled = nil
		pendingQuery = ""
	}
	cancelledIDs := func() []string {
		mu.Lock()
		defer mu.Unlock()
		ids := make([]string, 0, len(cancelled))
		for x := range cancelled {
			ids = append(ids, cancelled[x].OrderID)
		}
		return ids
	}

	t.Run("by order ID", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "KEEP-ME"},
			map[string]string{"instId": "BTC-USDT", "ordId": "CANCEL-ME"},
		)
		resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "CANCEL-ME",
		})
		require.NoError(t, err, "CancelAllOrders must not error when cancelling by order ID")
		assert.Equal(t, []string{"CANCEL-ME"}, cancelledIDs(), "cancelling by order ID should cancel only the requested order")
		assert.Contains(t, resp.Status, "CANCEL-ME", "the cancelled order should be reported")
	})

	t.Run("by client order ID", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "OTHER", "clOrdId": "other"},
			map[string]string{"instId": "BTC-USDT", "ordId": "TARGET", "clOrdId": "target"},
		)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType:     asset.Spot,
			Pair:          mainPair,
			ClientOrderID: "target",
		})
		require.NoError(t, err, "CancelAllOrders must not error when cancelling by client order ID")
		assert.Equal(t, []string{"TARGET"}, cancelledIDs(), "cancelling by client order ID should cancel only the requested order")
	})

	t.Run("by both order ID and client order ID", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "MATCHES-CL-only", "clOrdId": "both"},
			map[string]string{"instId": "BTC-USDT", "ordId": "both", "clOrdId": "MATCHES-ORD-only"},
			map[string]string{"instId": "BTC-USDT", "ordId": "both", "clOrdId": "both"},
		)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType:     asset.Spot,
			Pair:          mainPair,
			OrderID:       "both",
			ClientOrderID: "both",
		})
		require.NoError(t, err, "CancelAllOrders must not error when cancelling by both order ID and client order ID")
		assert.Equal(t, []string{"both"}, cancelledIDs(), "supplying both IDs should only cancel the order matching both")
	})

	t.Run("order type filter parameter", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "LMT-1", "ordType": "limit"},
		)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      mainPair,
			Type:      order.Limit,
		})
		require.NoError(t, err, "CancelAllOrders must not error when filtering by order type")
		mu.Lock()
		query := pendingQuery
		mu.Unlock()
		require.NotEmpty(t, query, "the pending order list must have been requested")
		assert.Contains(t, query, "ordType=limit", "the order type filter should be sent as ordType")
		assert.NotContains(t, query, "orderType=", "OKX ignores an orderType parameter, so it should not be sent")
	})

	t.Run("formats the pair through the exchange's pair format", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "ANY"},
		)
		slashPair := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      slashPair,
		})
		require.NoError(t, err, "CancelAllOrders must not error for a differently delimited pair")
		mu.Lock()
		query := pendingQuery
		mu.Unlock()
		assert.Contains(t, query, "instId=BTC-USDT", "the instrument filter should use the exchange's pair format")
	})

	t.Run("by side", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"},
			map[string]string{"instId": "BTC-USDT", "ordId": "SELL-1", "side": "sell"},
		)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      mainPair,
			Side:      order.Buy,
		})
		require.NoError(t, err, "CancelAllOrders must not error when cancelling by side")
		assert.Equal(t, []string{"BUY-1"}, cancelledIDs(), "cancelling buys should not cancel sell orders")
	})

	t.Run("all", func(t *testing.T) {
		resetCancelled()
		setPending(
			map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"},
			map[string]string{"instId": "BTC-USDT", "ordId": "SELL-1", "side": "sell"},
		)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      mainPair,
		})
		require.NoError(t, err, "CancelAllOrders must not error when cancelling everything")
		assert.Equal(t, []string{"BUY-1", "SELL-1"}, cancelledIDs(), "an unfiltered cancel should cancel every open order")
	})

	t.Run("paginates the pending order list", func(t *testing.T) {
		resetCancelled()
		firstPage := make([]map[string]string, 0, orderListPageSize)
		for x := range orderListPageSize {
			firstPage = append(firstPage, map[string]string{"instId": "BTC-USDT", "ordId": fmt.Sprintf("PAGE1-%03d", x)})
		}
		lastOfFirstPage := firstPage[orderListPageSize-1]["ordId"]
		setPages(map[string][]map[string]string{
			"": firstPage,
			lastOfFirstPage: {
				{"instId": "BTC-USDT", "ordId": "PAGE2-0"},
				{"instId": "BTC-USDT", "ordId": "PAGE2-1"},
			},
		})
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Pair:      mainPair,
		})
		require.NoError(t, err, "CancelAllOrders must not error when the pending order list spans pages")
		got := cancelledIDs()
		assert.Len(t, got, orderListPageSize+2, "a paginated cancel should cancel every open order across pages")
		assert.Contains(t, got, "PAGE2-1", "orders past the first page should be cancelled")
	})
}

// TestModifyOrderPriceOnlyAmend guards the NewPrice population in ModifyOrder:
// before it, a price-only amend serialised neither newSz nor newPx and was
// rejected locally by errInvalidNewSizeOrPriceInformation.
func TestModifyOrderPriceOnlyAmend(t *testing.T) {
	t.Parallel()

	var amendBody []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/amend-order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var err error
		amendBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading amend request body should not error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"123","sCode":"0"}]}`))
	}))

	_, err := e.ModifyOrder(t.Context(), &order.Modify{
		Exchange:  e.Name,
		AssetType: asset.Spot,
		Pair:      mainPair,
		OrderID:   "123",
		Type:      order.Limit,
		Price:     42000,
	})
	require.NoError(t, err, "ModifyOrder must accept a price-only amend")
	assert.Contains(t, string(amendBody), `"newPx":"42000"`, "a price-only amend should serialise the new price")
}

// TestModifyOrderFractionalAmount guards fractional amend sizes: OKX allows
// them on most instruments, and OrderManager.Modify sends the stored size with
// every price-only amend.
func TestModifyOrderFractionalAmount(t *testing.T) {
	t.Parallel()

	var amendBody []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/amend-order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var err error
		amendBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading amend request body should not error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"123","sCode":"0"}]}`))
	}))

	_, err := e.ModifyOrder(t.Context(), &order.Modify{
		Exchange:  e.Name,
		AssetType: asset.Spot,
		Pair:      mainPair,
		OrderID:   "123",
		Type:      order.Limit,
		Amount:    0.001,
		Price:     42000,
	})
	require.NoError(t, err, "ModifyOrder must accept a fractional amount")
	assert.JSONEq(t, `{"instId":"BTC-USDT","ordId":"123","newSz":"0.001","newPx":"42000"}`, string(amendBody), "a fractional amend should reach OKX with its new size and price")
}

// TestCancelAllOrdersContinuesAfterFailedBatch guards the batch loop: a failed
// batch must not stop later batches, and the joined error is returned with the
// statuses collected from the batches that succeeded.
func TestCancelAllOrdersContinuesAfterFailedBatch(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var cancelRequests int

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			// 22 orders exceed one batch of 20, forming a failing first batch
			// and a succeeding second batch.
			ords := make([]map[string]string, 0, 22)
			for x := range 22 {
				ords = append(ords, map[string]string{"instId": "BTC-USDT", "ordId": fmt.Sprintf("ORD-%02d", x)})
			}
			writeOKXData(t, w, ords)
		case "/trade/cancel-batch-orders":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading cancel request body should not error: %v", err)
				return
			}
			var reqs []CancelOrderRequestParam
			if err := json.Unmarshal(body, &reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			mu.Lock()
			cancelRequests++
			first := cancelRequests == 1
			mu.Unlock()
			if first {
				http.Error(w, "batch one failed", http.StatusInternalServerError)
				return
			}
			respData := make([]map[string]string, 0, len(reqs))
			for x := range reqs {
				respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			writeOKXData(t, w, respData)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))

	resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
	require.Error(t, err, "CancelAllOrders must report the failed batch")
	mu.Lock()
	requests := cancelRequests
	mu.Unlock()
	assert.Equal(t, 2, requests, "a failed batch should not stop later batches from being sent")
	assert.Contains(t, resp.Status, "ORD-21", "orders from the later successful batch should still be cancelled")
	assert.NotContains(t, resp.Status, "ORD-00", "orders from the failed batch should not be reported as cancelled")
}

// TestCancelBatchOrdersRESTPartialSuccess guards the REST partial-success
// path: OKX answers a partly failed batch with code 2 and per-order results,
// which must be decoded and reported alongside the error.
func TestCancelBatchOrdersRESTPartialSuccess(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-batch-orders" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"2","msg":"","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`))
	}))
	resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "OK-1"},
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "FAIL-1"},
	})
	require.ErrorIs(t, err, errPartialSuccess, "a partially successful REST batch must report its error")
	require.NotNil(t, resp, "CancelBatchOrders must return the partial results")
	assert.Equal(t, map[string]string{
		"OK-1":   order.Cancelled.String(),
		"FAIL-1": "Order does not exist",
	}, resp.Status, "a partially successful REST batch should report every per-order result")
}

// TestCancelAllOrdersRESTPartialSuccess guards the REST partial-success path
// through CancelAllOrders: the per-order results of a code 2 batch reply must
// reach the status map alongside the error.
func TestCancelAllOrdersRESTPartialSuccess(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			writeOKXData(t, w, []map[string]string{
				{"instId": "BTC-USDT", "ordId": "OK-1"},
				{"instId": "BTC-USDT", "ordId": "FAIL-1"},
			})
		case "/trade/cancel-batch-orders":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"2","msg":"","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`))
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
	require.ErrorIs(t, err, errPartialSuccess, "a partially successful REST batch must report its error")
	assert.Equal(t, map[string]string{
		"OK-1":   order.Cancelled.String(),
		"FAIL-1": "Order does not exist",
	}, resp.Status, "a partially successful REST batch should report every per-order result")
}

// TestCancelBatchOrdersSpreadGuards covers the spread branch of
// CancelBatchOrders: cancels report a usable status key, the missing-ID guard
// rejects the batch, and no cancel is sent until the whole batch validates.
// Subtests share the mock server's request counter, so they run sequentially.
func TestCancelBatchOrdersSpreadGuards(t *testing.T) {
	const (
		spreadOrdID = "2341161427393388544"
		spotOrdID   = "2341161427393388545"
	)

	var mu sync.Mutex
	var spreadCancels int
	var batchFails bool

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sprd/cancel-order":
			mu.Lock()
			spreadCancels++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":{"sCode":"0","sMsg":"","ordId":"` + spreadOrdID + `","clOrdId":""}}`))
		case "/trade/cancel-batch-orders":
			mu.Lock()
			fails := batchFails
			mu.Unlock()
			if fails {
				http.Error(w, "batch failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":"","ordId":"` + spotOrdID + `","clOrdId":"spot-cl"}]}`))
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))

	setBatchFails := func(v bool) {
		mu.Lock()
		defer mu.Unlock()
		batchFails = v
	}
	resetSpreadCancels := func() {
		mu.Lock()
		defer mu.Unlock()
		spreadCancels = 0
	}
	spreadCancelsSent := func() int {
		mu.Lock()
		defer mu.Unlock()
		return spreadCancels
	}

	t.Run("client order ID status key", func(t *testing.T) {
		resetSpreadCancels()
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{
			AssetType:     asset.Spread,
			Pair:          spreadPair,
			ClientOrderID: "spread-cl",
		}})
		require.NoError(t, err, "CancelBatchOrders must not error for a client-ID-only spread cancel")
		assert.Equal(t, 1, spreadCancelsSent(), "the spread cancel should have been sent")
		assert.Equal(t, map[string]string{spreadOrdID: order.Cancelled.String()}, resp.Status,
			"a client-ID-only spread cancel should report the status under the order ID OKX returns")
	})

	t.Run("guard requires an ID", func(t *testing.T) {
		resetSpreadCancels()
		_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{
			AssetType: asset.Spread,
			Pair:      spreadPair,
		}})
		assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "a spread cancel without either ID should be rejected")
		assert.Zero(t, spreadCancelsSent(), "a rejected batch should send no cancels")
	})

	t.Run("invalid entry cancels nothing", func(t *testing.T) {
		resetSpreadCancels()
		_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "spread-cl"},
			{AssetType: asset.Spot, Pair: currency.Pair{}, OrderID: "spot-order"},
		})
		assert.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "an invalid entry should fail the batch")
		assert.Zero(t, spreadCancelsSent(), "no cancel should be sent before the whole batch validates")
	})

	t.Run("client order ID alone passes the ordinary order guard", func(t *testing.T) {
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{
			AssetType:     asset.Spot,
			Pair:          mainPair,
			ClientOrderID: "spot-cl",
		}})
		require.NoError(t, err, "CancelBatchOrders must accept a client-ID-only spot cancel")
		assert.Equal(t, map[string]string{spotOrdID: order.Cancelled.String()}, resp.Status,
			"a client-ID-only spot cancel should report the status under the order ID OKX returns")
	})

	t.Run("ordinary batch failure keeps spread statuses", func(t *testing.T) {
		resetSpreadCancels()
		setBatchFails(true)
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "spread-cl"},
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "spot-order"},
		})
		assert.Error(t, err, "the failed ordinary batch should be reported")
		assert.Equal(t, 1, spreadCancelsSent(), "the spread cancel before the failed batch should have been sent")
		assert.Equal(t, map[string]string{spreadOrdID: order.Cancelled.String()}, resp.Status,
			"a failed ordinary batch should not discard the spread statuses already recorded")
	})
}

// TestCancelBatchOrdersSpreadReplies guards spread cancel replies that confirm
// no order: each fails the batch, with a sentinel error when the reply names no
// order, and keeps the status of the spread order cancelled before it.
func TestCancelBatchOrdersSpreadReplies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		reply string
		err   error
	}{
		{"no data", `{"code":"0","msg":"","data":null}`, common.ErrNoResponse},
		{"no order ID", `{"code":"0","msg":"","data":[{"ordId":"","clOrdId":"S2","sCode":"0","sMsg":""}]}`, common.ErrInvalidResponse},
		{"cancel failed", `{"code":"1","msg":"All operations failed","data":[{"ordId":"","clOrdId":"S2","sCode":"51400","sMsg":"Order does not exist"}]}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var cancels int
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/cancel-order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				mu.Lock()
				cancels++
				first := cancels == 1
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if first {
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"SPRD-1","clOrdId":"S1","sCode":"0","sMsg":""}]}`))
					return
				}
				_, _ = w.Write([]byte(tc.reply))
			}))
			resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "S1"},
				{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "S2"},
			})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "CancelBatchOrders must return the sentinel for a reply that names no order")
			} else {
				require.Error(t, err, "CancelBatchOrders must return the failed cancel's error")
			}
			require.NotNil(t, resp, "CancelBatchOrders must return the statuses already recorded")
			assert.Equal(t, map[string]string{"SPRD-1": order.Cancelled.String()}, resp.Status,
				"the spread order cancelled before the failed reply should still be reported")
		})
	}
}

// TestCancelOrderAlgoEmptyReply guards the algo cancel reply that names no
// order, which would otherwise index an empty result.
func TestCancelOrderAlgoEmptyReply(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-advance-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []any{})
	}))
	err := e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-1", Type: order.Trigger})
	assert.ErrorIs(t, err, common.ErrNoResponse, "an algo cancel reply without results should return ErrNoResponse")
}

// TestApplyWebsocketInstrumentIDCodes guards the resolve-all-then-apply shape:
// an uncached instrument must report failure without touching the requests, so
// no half-applied instIdCode leaks into a REST request body.
func TestApplyWebsocketInstrumentIDCodes(t *testing.T) {
	t.Parallel()
	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]Instrument)
	fresh.instrumentIDCodeMap = map[string]uint64{"BTC-USDT": 12345}

	args := []CancelOrderRequestParam{
		{InstrumentID: "BTC-USDT"},
		{InstrumentID: "ETH-USDT"},
	}
	assert.False(t, fresh.applyWebsocketInstrumentIDCodes(args), "an uncached instrument should report failure")
	assert.Zero(t, args[0].InstrumentIDCode, "a failed apply should leave the resolved requests untouched")

	args = []CancelOrderRequestParam{
		{InstrumentID: "BTC-USDT"},
		{InstrumentID: "BTC-USDT"},
	}
	assert.True(t, fresh.applyWebsocketInstrumentIDCodes(args), "cached instruments should report success")
	assert.EqualValues(t, 12345, args[0].InstrumentIDCode, "a successful apply should assign the cached code")
	assert.EqualValues(t, 12345, args[1].InstrumentIDCode, "a successful apply should assign the cached code")
}

func TestWebsocketInstrumentIDCode(t *testing.T) {
	t.Parallel()
	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]Instrument)
	fresh.instrumentIDCodeMap = map[string]uint64{"BTC-USDT": 12345}

	code, ok := fresh.websocketInstrumentIDCode("BTC-USDT")
	assert.True(t, ok, "a cached instrument should report availability")
	assert.EqualValues(t, 12345, code, "a cached instrument should report its code")

	code, ok = fresh.websocketInstrumentIDCode("ETH-USDT")
	assert.False(t, ok, "an uncached instrument should report unavailability")
	assert.Zero(t, code, "an uncached instrument should report a zero code")
}

// TestCancelAllOrdersScopesSpreadMassCancel guards the spread branch: OKX's
// mass-cancel takes the spread pair as sprdId and cancels every spread order
// when it is omitted, so a populated pair must scope the cancel instead of the
// order ID leaking in as a bogus sprdId.
func TestCancelAllOrdersScopesSpreadMassCancel(t *testing.T) {
	var mu sync.Mutex
	var massCancelBodies []string

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/mass-cancel" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading mass-cancel body should not error: %v", err)
			return
		}
		mu.Lock()
		massCancelBodies = append(massCancelBodies, string(body))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"result":true}]}`))
	}))

	t.Run("pair scopes the mass cancel", func(t *testing.T) {
		resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spread,
			Pair:      spreadPair,
		})
		require.NoError(t, err, "CancelAllOrders must not error for a pair-scoped spread cancel")
		mu.Lock()
		last := massCancelBodies[len(massCancelBodies)-1]
		mu.Unlock()
		assert.Contains(t, last, `"sprdId":"BTC-USDT_BTC-USDT-SWAP"`, "a populated pair should scope the mass cancel to that spread")
		assert.Equal(t, map[string]string{"BTC-USDT_BTC-USDT-SWAP": "true"}, resp.Status, "the result should be reported under the spread it cancelled")
	})

	t.Run("no pair cancels every spread", func(t *testing.T) {
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spread})
		require.NoError(t, err, "CancelAllOrders must not error for an unscoped spread cancel")
		mu.Lock()
		last := massCancelBodies[len(massCancelBodies)-1]
		mu.Unlock()
		assert.Equal(t, "{}", last, "an unscoped cancel should omit sprdId so OKX cancels every spread order")
	})
}

// TestCancelBatchOrdersReportsAlgoResults guards the per-item algo cancel
// results: before the array decode, a response for more than one algo order
// failed to decode, so the batch errored, or dropped the algo statuses without
// an error when ordinary orders in the same batch were cancelled.
func TestCancelBatchOrdersReportsAlgoResults(t *testing.T) {
	t.Parallel()

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-advance-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[` +
			`{"algoId":"ALGO-OK","sCode":"0","sMsg":""},` +
			`{"algoId":"ALGO-BAD","sCode":"51000","sMsg":"The algo order does not exist"}]}`))
	}))

	resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-OK", Type: order.Trigger},
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-BAD", Type: order.Trigger},
	})
	require.NoError(t, err, "CancelBatchOrders must not error when the batch is accepted")
	assert.Equal(t, order.Cancelled.String(), resp.Status["ALGO-OK"], "a successfully cancelled algo order should report Cancelled")
	assert.Equal(t, "The algo order does not exist", resp.Status["ALGO-BAD"], "a failed algo cancel should report its status message")
}

// TestCancelResultsUsable guards the gate that decides whether per-order cancel
// results are recorded alongside a batch error: only a websocket partial
// success returns fully decoded results with its error.
func TestCancelResultsUsable(t *testing.T) {
	t.Parallel()
	assert.True(t, cancelResultsUsable(nil), "a successful batch should report usable results")
	assert.True(t, cancelResultsUsable(errPartialSuccess), "a partial success should report usable results")
	assert.True(t, cancelResultsUsable(common.AppendError(errPartialSuccess, errors.New("order 3: does not exist"))), "a joined partial success should still report usable results")
	assert.False(t, cancelResultsUsable(errors.New("batch one failed")), "any other error should report unusable results")
}
