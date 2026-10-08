package okx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
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
	var afterCursors []string

	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			after := r.URL.Query().Get("after")
			mu.Lock()
			defer mu.Unlock()
			if slices.Contains(afterCursors, after) {
				// Nothing here fails a request, so a repeat means the cursor never advanced.
				t.Errorf("after %q requested again", after)
				http.Error(w, "repeated after", http.StatusBadRequest)
				return
			}
			afterCursors = append(afterCursors, after)
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
		afterCursors = nil
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
// path: OKX answers a partly failed batch with code 2, a message and per-order
// results, which are reported alongside the error only when they decode,
// whatever the message.
func TestCancelBatchOrdersRESTPartialSuccess(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		reply   string
		partial bool
		status  map[string]string
	}{
		{
			name:    "per-order results",
			reply:   `{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`,
			partial: true,
			status:  map[string]string{"OK-1": order.Cancelled.String(), "FAIL-1": "Order does not exist"},
		},
		{
			name:    "null result",
			reply:   `{"code":"2","msg":"Bulk operation partially succeeded.","data":[null,{"ordId":"OK-1","sCode":"0","sMsg":""}]}`,
			partial: true,
			status:  map[string]string{"OK-1": order.Cancelled.String()},
		},
		{
			name:   "undecodable result",
			reply:  `{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"bad","sMsg":"Order does not exist"}]}`,
			status: map[string]string{},
		},
		{
			name:   "undecodable result without a message",
			reply:  `{"code":"2","msg":"","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"bad","sMsg":"Order does not exist"}]}`,
			status: map[string]string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/trade/cancel-batch-orders" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.reply))
			}))
			resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.Spot, Pair: mainPair, OrderID: "OK-1"},
				{AssetType: asset.Spot, Pair: mainPair, OrderID: "FAIL-1"},
			})
			require.Error(t, err, "a partially successful REST batch must report its error")
			assert.Equal(t, tc.partial, errors.Is(err, errPartialSuccess), "only results that decode should be marked as a partial success")
			require.NotNil(t, resp, "CancelBatchOrders must return the partial results")
			assert.Equal(t, tc.status, resp.Status, "a partially successful REST batch should report every decoded per-order result")
		})
	}
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
			_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`))
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

// TestGetActiveOrdersPaginatesWithOrderIDCursor guards the pending order crawl:
// OKX pages /trade/orders-pending with the after order ID cursor, so paginating
// by a timestamp instead re-fetched the first page, losing every order past the
// first hundred.
func TestGetActiveOrdersPaginatesWithOrderIDCursor(t *testing.T) {
	t.Parallel()

	fillPage := func(first, count int) []map[string]string {
		ords := make([]map[string]string, 0, count)
		for x := range count {
			ords = append(ords, map[string]string{
				"instId":    "BTC-USDT",
				"ordId":     fmt.Sprintf("ORD-%03d", first+x),
				"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"state":     "live",
				"ordType":   "limit",
				"side":      "buy",
				"sz":        "1",
				"px":        "42000",
				"accFillSz": "0",
				"feeCcy":    "USDT",
				"fee":       "0",
			})
		}
		return ords
	}
	pages := map[string][]map[string]string{
		"":        fillPage(0, orderListPageSize),
		"ORD-099": fillPage(orderListPageSize, 50),
	}

	var mu sync.Mutex
	var afterCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-pending" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		after := r.URL.Query().Get("after")
		mu.Lock()
		repeated := slices.Contains(afterCursors, after)
		afterCursors = append(afterCursors, after)
		page := pages[after]
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("after %q requested again", after)
			http.Error(w, "repeated after", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))

	resp, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must not error when the pending order list spans pages")

	mu.Lock()
	cursors := make([]string, len(afterCursors))
	copy(cursors, afterCursors)
	mu.Unlock()
	assert.Equal(t, []string{"", "ORD-099"}, cursors, "the second page should be requested with the first page's last order ID")
	assert.Len(t, resp, orderListPageSize+50, "a paginated crawl should return every open order")
	ids := make(map[string]struct{}, len(resp))
	for x := range resp {
		ids[resp[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, orderListPageSize+50, "a paginated crawl should not duplicate orders")
	assert.Contains(t, ids, "ORD-149", "orders past the first page should be returned")
}

// TestOrderTypeFilter guards the ordType filter a type selects: every OKX order
// type orderTypeFromString reads back as the requested type and time in force,
// or orderTypeString's when none does.
func TestOrderTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
	}{
		{order.Limit, order.UnknownTIF, "limit,post_only,fok,ioc,op_fok,rpi"},
		{order.Limit, order.PostOnly, orderPostOnly},
		{order.Limit, order.FillOrKill, "fok,op_fok"},
		{order.Limit, order.ImmediateOrCancel, orderIOC},
		{order.Limit, order.GoodTillCancel, orderLimit},
		{order.MarketMakerProtection, order.UnknownTIF, "mmp,mmp_and_post_only"},
		{order.MarketMakerProtection, order.PostOnly, orderMarketMakerProtectionAndPostOnly},
		{order.Market, order.UnknownTIF, orderMarket},
		{order.Trigger, order.UnknownTIF, "trigger"},
	} {
		got, err := orderTypeFilter(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "orderTypeFilter must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "orderTypeFilter should select the OKX order types for %s %s", tc.orderType, tc.tif)
	}
	_, err := orderTypeFilter(order.Stop, order.UnknownTIF)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "orderTypeFilter should reject an order type OKX does not list")
}

// TestSpreadOrderTypeFilter guards the spread order type filter: the spread
// endpoints document only market, limit, post_only and ioc as single ordType
// values and reject the comma-separated lists the ordinary order endpoints
// accept with 51000.
func TestSpreadOrderTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
	}{
		// limit, post_only and ioc all read back as Limit: no filter is sent
		// and the request filter narrows by type instead.
		{order.Limit, order.UnknownTIF, ""},
		{order.Limit, order.GoodTillCancel, orderLimit},
		{order.Limit, order.PostOnly, orderPostOnly},
		{order.Limit, order.ImmediateOrCancel, orderIOC},
		{order.Market, order.UnknownTIF, orderMarket},
		// A market order with ImmediateOrCancel places as market.
		{order.Market, order.ImmediateOrCancel, orderMarket},
	} {
		got, err := spreadOrderTypeFilter(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "spreadOrderTypeFilter must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "spreadOrderTypeFilter should select a documented spread ordType for %s %s", tc.orderType, tc.tif)
	}
	_, err := spreadOrderTypeFilter(order.Market, order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject the fok a fill-or-kill market order maps to")
	_, err = spreadOrderTypeFilter(order.Limit, order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject the fok a fill-or-kill limit order maps to")
	_, err = spreadOrderTypeFilter(order.Trigger, order.UnknownTIF)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject an order type the spread endpoints do not list")
}

// TestSpreadOrderTypeFilterRequests guards the ordType the spread order
// queries send: only the documented single values reach the spread endpoints.
func TestSpreadOrderTypeFilterRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		exp       string
		expErr    error
	}{
		{"limit spans no filter", order.Limit, order.UnknownTIF, "", nil},
		{"good till cancel limit", order.Limit, order.GoodTillCancel, orderLimit, nil},
		{"post only limit", order.Limit, order.PostOnly, orderPostOnly, nil},
		{"immediate or cancel limit", order.Limit, order.ImmediateOrCancel, orderIOC, nil},
		{"market", order.Market, order.UnknownTIF, orderMarket, nil},
		{"fill or kill market rejected", order.Market, order.FillOrKill, "", order.ErrUnsupportedOrderType},
		{"fill or kill limit rejected", order.Limit, order.FillOrKill, "", order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotOrdType string
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/orders-pending" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				gotOrdType = r.URL.Query().Get("ordType")
				writeOKXData(t, w, []map[string]string{{
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderLimit,
					"side": "buy", "state": "live", "sz": "1", "accFillSz": "0", "px": "100",
				}})
			}))
			_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spread, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide,
			})
			if tc.expErr != nil {
				assert.ErrorIs(t, err, tc.expErr, "GetActiveOrders should reject the type before sending")
				return
			}
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s %s", tc.orderType, tc.tif)
			assert.Equal(t, tc.exp, gotOrdType, "the spread query should send the documented ordType")
		})
	}
}

// TestSpreadOrderHistoryTypeFilter guards the spread GetOrderHistory ordType:
// the documented single values are sent, and a plain limit query no longer
// drops post_only and ioc history behind a limit-only filter.
func TestSpreadOrderHistoryTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		exp       string
	}{
		{"limit spans no filter", order.Limit, order.UnknownTIF, ""},
		{"post only limit", order.Limit, order.PostOnly, orderPostOnly},
		{"market", order.Market, order.UnknownTIF, orderMarket},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotOrdType string
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/sprd/orders-history":
					gotOrdType = r.URL.Query().Get("ordType")
					writeOKXData(t, w, []map[string]string{{
						"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderPostOnly,
						"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
					}})
				case "/sprd/orders-history-archive":
					// A zero StartTime reaches back past the 21 day listing, so the archive is also crawled.
					writeOKXData(t, w, []map[string]string{})
				default:
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spread, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide,
			})
			require.NoErrorf(t, err, "GetOrderHistory must not error for %s %s", tc.orderType, tc.tif)
			assert.Equal(t, tc.exp, gotOrdType, "the spread history query should send the documented ordType")
		})
	}
}

// TestGetActiveSpreadOrdersPaginatesWithEndIDCursor guards the pending spread
// order pagination: OKX caps the sprd/orders-pending response at
// orderListPageSize records while an account can hold 500 pending spread
// orders, and pages the remainder with the endId cursor, which returns
// records earlier than the order ID.
func TestGetActiveSpreadOrdersPaginatesWithEndIDCursor(t *testing.T) {
	t.Parallel()

	fillPage := func(first, count int) []map[string]string {
		ords := make([]map[string]string, 0, count)
		for x := range count {
			ords = append(ords, map[string]string{
				"sprdId":    "BTC-USDT_BTC-USDT",
				"ordId":     fmt.Sprintf("SPD-%03d", first+x),
				"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"state":     "live",
				"ordType":   "limit",
				"side":      "buy",
				"sz":        "1",
				"px":        "42000",
				"accFillSz": "0",
			})
		}
		return ords
	}
	pages := map[string][]map[string]string{
		"":        fillPage(0, orderListPageSize),
		"SPD-099": fillPage(orderListPageSize, 50),
	}

	var mu sync.Mutex
	var endIDCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-pending" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		endID := r.URL.Query().Get("endId")
		mu.Lock()
		repeated := slices.Contains(endIDCursors, endID)
		endIDCursors = append(endIDCursors, endID)
		page := pages[endID]
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("endId %q requested again", endID)
			http.Error(w, "repeated endId", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))

	resp, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread,
		Type:      order.AnyType,
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must not error when the pending spread order list spans pages")

	mu.Lock()
	cursors := make([]string, len(endIDCursors))
	copy(cursors, endIDCursors)
	mu.Unlock()
	assert.Equal(t, []string{"", "SPD-099"}, cursors, "the second page should be requested with the first page's last order ID as the endId cursor")
	assert.Len(t, resp, orderListPageSize+50, "a paginated crawl should return every pending spread order")
	ids := make(map[string]struct{}, len(resp))
	for x := range resp {
		ids[resp[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, orderListPageSize+50, "a paginated crawl should not duplicate orders")
	assert.Contains(t, ids, "SPD-149", "orders past the first page should be returned")
}

// TestGetSpreadOrderHistoryPaginatesWithEndIDCursor guards the spread order
// history pagination: OKX caps the sprd/orders-history response at
// orderListPageSize records and pages the remainder with the same endId
// earlier-than order ID cursor as the pending spread order listing.
func TestGetSpreadOrderHistoryPaginatesWithEndIDCursor(t *testing.T) {
	t.Parallel()

	fillPage := func(first, count int) []map[string]string {
		ords := make([]map[string]string, 0, count)
		for x := range count {
			ords = append(ords, map[string]string{
				"sprdId":    "BTC-USDT_BTC-USDT",
				"ordId":     fmt.Sprintf("SPD-%03d", first+x),
				"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"state":     "filled",
				"ordType":   "limit",
				"side":      "buy",
				"sz":        "1",
				"px":        "42000",
				"accFillSz": "1",
				"avgPx":     "42000",
			})
		}
		return ords
	}
	pages := map[string][]map[string]string{
		"":        fillPage(0, orderListPageSize),
		"SPD-099": fillPage(orderListPageSize, 50),
	}

	var mu sync.Mutex
	var endIDCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sprd/orders-history-archive" {
			// A zero StartTime reaches back past the 21 day listing, so the archive is also crawled.
			writeOKXData(t, w, []map[string]string{})
			return
		}
		if r.URL.Path != "/sprd/orders-history" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		endID := r.URL.Query().Get("endId")
		mu.Lock()
		repeated := slices.Contains(endIDCursors, endID)
		endIDCursors = append(endIDCursors, endID)
		page := pages[endID]
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("endId %q requested again", endID)
			http.Error(w, "repeated endId", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))

	resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread,
		Type:      order.AnyType,
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetOrderHistory must not error when the spread order history spans pages")

	mu.Lock()
	cursors := make([]string, len(endIDCursors))
	copy(cursors, endIDCursors)
	mu.Unlock()
	assert.Equal(t, []string{"", "SPD-099"}, cursors, "the second page should be requested with the first page's last order ID as the endId cursor")
	assert.Len(t, resp, orderListPageSize+50, "a paginated crawl should return every spread order in the history")
	ids := make(map[string]struct{}, len(resp))
	for x := range resp {
		ids[resp[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, orderListPageSize+50, "a paginated crawl should not duplicate orders")
	assert.Contains(t, ids, "SPD-149", "orders past the first page should be returned")
}

// TestGetSpreadOrderHistoryFromOrderIDSeedsTheEndIDCursor guards the
// FromOrderID request field for spread order history: the endId cursor
// returns records earlier than the requested order ID like the standard
// archive's after cursor, so the first request must carry FromOrderID as the
// endId cursor, and beginId, which returns records newer than an order ID,
// must never be sent.
func TestGetSpreadOrderHistoryFromOrderIDSeedsTheEndIDCursor(t *testing.T) {
	t.Parallel()

	fillPage := func(first, count int) []map[string]string {
		ords := make([]map[string]string, 0, count)
		for x := range count {
			ords = append(ords, map[string]string{
				"sprdId":    "BTC-USDT_BTC-USDT",
				"ordId":     fmt.Sprintf("SPD-%03d", first+x),
				"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"state":     "filled",
				"ordType":   "limit",
				"side":      "buy",
				"sz":        "1",
				"px":        "42000",
				"accFillSz": "1",
				"avgPx":     "42000",
			})
		}
		return ords
	}
	pages := map[string][]map[string]string{
		"SEED-1":  fillPage(0, orderListPageSize),
		"SPD-099": fillPage(orderListPageSize, 50),
	}

	var mu sync.Mutex
	var endIDCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-history" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		assert.Empty(t, r.URL.Query().Get("beginId"), "beginId returns records newer than the order ID and should never be sent")
		endID := r.URL.Query().Get("endId")
		mu.Lock()
		repeated := slices.Contains(endIDCursors, endID)
		endIDCursors = append(endIDCursors, endID)
		page := pages[endID]
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("endId %q requested again", endID)
			http.Error(w, "repeated endId", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType:   asset.Spread,
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "SEED-1",
		// A recent start time keeps the crawl inside the 21 day listing.
		StartTime: time.Now().Add(-24 * time.Hour),
	})
	require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds the spread crawl")

	mu.Lock()
	cursors := make([]string, len(endIDCursors))
	copy(cursors, endIDCursors)
	mu.Unlock()
	assert.Equal(t, []string{"SEED-1", "SPD-099"}, cursors, "the crawl should start from FromOrderID and advance with each page's last order ID")
	assert.Len(t, history, orderListPageSize+50, "a crawl seeded by FromOrderID should return every earlier spread order")
}

// TestGetSpreadOrderHistoryStopsWhenAFullPageIsSeen guards the no-progress
// stop: a full page whose last order ID repeats the requested endId cursor
// must not loop forever on the same page.
func TestGetSpreadOrderHistoryStopsWhenAFullPageIsSeen(t *testing.T) {
	t.Parallel()

	page := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		page = append(page, map[string]string{
			"sprdId":    "BTC-USDT_BTC-USDT",
			"ordId":     fmt.Sprintf("SPD-%03d", i),
			"cTime":     strconv.FormatInt(1700000000000, 10),
			"uTime":     strconv.FormatInt(1700000000000, 10),
			"state":     "filled",
			"ordType":   "limit",
			"side":      "buy",
			"sz":        "1",
			"px":        "42000",
			"accFillSz": "1",
			"avgPx":     "42000",
		})
	}
	var mu sync.Mutex
	var endIDCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-history" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		endID := r.URL.Query().Get("endId")
		mu.Lock()
		repeated := slices.Contains(endIDCursors, endID)
		endIDCursors = append(endIDCursors, endID)
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("endId %q requested again", endID)
			http.Error(w, "repeated endId", http.StatusBadRequest)
			return
		}
		// The mock keeps returning the same full page whatever the cursor.
		writeOKXData(t, w, page)
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread,
		Type:      order.AnyType,
		Side:      order.AnySide,
		// A recent start time keeps the crawl inside the 21 day listing.
		StartTime: time.Now().Add(-24 * time.Hour),
	})
	require.NoError(t, err, "GetOrderHistory must stop when a full page holds no new spread order")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"", "SPD-099"}, endIDCursors, "the crawl should stop when the endId cursor stops advancing")
	assert.Len(t, history, orderListPageSize, "the first page's orders should be returned once")
}

// TestGetSpreadOrderHistoryFallsBackToArchive guards the documented 3 month
// spread order history window: a start time older than the 21 day listing's
// window must also crawl sprd/orders-history-archive, and the overlap between
// the two listings must not duplicate orders. OKX confirmed the listing's
// begin filter reaches its full 21 day window, so windows inside it need no
// archive crawl.
func TestGetSpreadOrderHistoryFallsBackToArchive(t *testing.T) {
	t.Parallel()

	recent := map[string]string{
		"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-NEW", "ordType": orderLimit,
		"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
		"cTime": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	tenDaysOld := map[string]string{
		"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-TEN", "ordType": orderLimit,
		"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
		"cTime": strconv.FormatInt(time.Now().Add(-10*24*time.Hour).UnixMilli(), 10),
	}
	old := map[string]string{
		"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-OLD", "ordType": orderLimit,
		"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
		"cTime": strconv.FormatInt(time.Now().Add(-kline.OneMonth.Duration()).UnixMilli(), 10),
	}

	t.Run("old start time also crawls the archive", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var archiveCrawls int
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sprd/orders-history":
				writeOKXData(t, w, []map[string]string{recent})
			case "/sprd/orders-history-archive":
				mu.Lock()
				archiveCrawls++
				mu.Unlock()
				// The archive listing overlaps the 21 day listing, since it lags it.
				writeOKXData(t, w, []map[string]string{recent, old})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
			StartTime: time.Now().Add(-kline.OneMonth.Duration()),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the archive is crawled")
		mu.Lock()
		crawls := archiveCrawls
		mu.Unlock()
		assert.Equal(t, 1, crawls, "the archive should be crawled when the start time passes the listing's 21 day window")
		ids := make([]string, 0, len(resp))
		for x := range resp {
			ids = append(ids, resp[x].OrderID)
		}
		assert.ElementsMatch(t, []string{"SPD-NEW", "SPD-OLD"}, ids, "the archive crawl should add the old order without duplicating the shared one")
	})

	t.Run("recent start time skips the archive", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sprd/orders-history" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{recent})
		}))
		resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
			StartTime: time.Now().Add(-time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the 21 day listing covers the start time")
		require.Len(t, resp, 1, "a recent start time must not need the archive")
		assert.Equal(t, "SPD-NEW", resp[0].OrderID)
	})

	t.Run("start time inside the listing's 21 day window skips the archive", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var archiveCrawls int
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sprd/orders-history":
				// OKX confirmed the listing's begin filter reaches its full
				// 21 day window, so ten day old orders come from the listing.
				writeOKXData(t, w, []map[string]string{recent, tenDaysOld})
			case "/sprd/orders-history-archive":
				mu.Lock()
				archiveCrawls++
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{old})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
			// Ten days back sits inside the listing's 21 day window, so the
			// listing alone must cover it and the archive must not be crawled.
			StartTime: time.Now().Add(-10 * 24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the 21 day listing covers the start time")
		mu.Lock()
		crawls := archiveCrawls
		mu.Unlock()
		assert.Equal(t, 0, crawls, "the archive should not be crawled when the start time sits inside the listing's 21 day window")
		ids := make([]string, 0, len(resp))
		for x := range resp {
			ids = append(ids, resp[x].OrderID)
		}
		assert.ElementsMatch(t, []string{"SPD-NEW", "SPD-TEN"}, ids, "the listing should carry the orders inside its 21 day window")
	})

	t.Run("end time older than the listing's window skips the listing", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sprd/orders-history":
				t.Error("the 21 day listing must not be crawled when the window ends before it")
				http.NotFound(w, r)
			case "/sprd/orders-history-archive":
				writeOKXData(t, w, []map[string]string{old})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
			// A window entirely older than the 21 day listing can only be
			// served by the archive.
			StartTime: time.Now().Add(-40 * 24 * time.Hour),
			EndTime:   time.Now().Add(-30 * 24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when only the archive covers the window")
		require.Len(t, resp, 1, "the archive must be the only source for a window older than the listing")
		assert.Equal(t, "SPD-OLD", resp[0].OrderID)
	})
}

// TestGetSpreadOrderHistoryFromOrderIDSeedsTheArchiveCrawl guards the
// FromOrderID request field across both spread listings: when the requested
// window reaches the archive, both crawls must start from FromOrderID as
// their endId cursor, and the archive's overlapping row must not duplicate
// the listing's.
func TestGetSpreadOrderHistoryFromOrderIDSeedsTheArchiveCrawl(t *testing.T) {
	t.Parallel()

	fillPage := func(first, count int) []map[string]string {
		ords := make([]map[string]string, 0, count)
		for x := range count {
			ords = append(ords, map[string]string{
				"sprdId":    "BTC-USDT_BTC-USDT",
				"ordId":     fmt.Sprintf("SPD-%03d", first+x),
				"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
				"state":     "filled",
				"ordType":   "limit",
				"side":      "buy",
				"sz":        "1",
				"px":        "42000",
				"accFillSz": "1",
				"avgPx":     "42000",
			})
		}
		return ords
	}
	pages := map[string][]map[string]string{
		"SEED-1":  fillPage(0, orderListPageSize),
		"SPD-099": fillPage(orderListPageSize, 50),
	}

	var mu sync.Mutex
	cursors := map[string][]string{}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endID := r.URL.Query().Get("endId")
		mu.Lock()
		pathCursors := cursors[r.URL.Path]
		repeated := slices.Contains(pathCursors, endID)
		cursors[r.URL.Path] = append(pathCursors, endID)
		mu.Unlock()
		if repeated {
			// Nothing here fails a request, so a repeat means the cursor never advanced.
			t.Errorf("%s endId %q requested again", r.URL.Path, endID)
			http.Error(w, "repeated endId", http.StatusBadRequest)
			return
		}
		assert.Empty(t, r.URL.Query().Get("beginId"), "beginId returns records newer than the order ID and should never be sent")
		switch r.URL.Path {
		case "/sprd/orders-history":
			writeOKXData(t, w, pages[endID])
		case "/sprd/orders-history-archive":
			if endID == "SEED-1" {
				// The archive repeats the listing's SPD-099 and adds the old order.
				writeOKXData(t, w, []map[string]string{pages["SEED-1"][orderListPageSize-1], {
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-OLD", "ordType": "limit",
					"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "42000",
					"cTime": strconv.FormatInt(1600000000000, 10),
					"uTime": strconv.FormatInt(1600000000000, 10),
				}})
				return
			}
			writeOKXData(t, w, []map[string]string{})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType:   asset.Spread,
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "SEED-1",
		StartTime:   time.Now().Add(-kline.OneMonth.Duration()),
	})
	require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds both spread crawls")

	mu.Lock()
	listing := slices.Clone(cursors["/sprd/orders-history"])
	archive := slices.Clone(cursors["/sprd/orders-history-archive"])
	mu.Unlock()
	assert.Equal(t, []string{"SEED-1", "SPD-099"}, listing, "the 21 day crawl should start from FromOrderID and advance with each page's last order ID")
	assert.Equal(t, []string{"SEED-1"}, archive, "the archive crawl should also start from FromOrderID")
	require.Len(t, history, orderListPageSize+51, "both crawls must return every earlier spread order")
	ids := make(map[string]struct{}, len(history))
	for x := range history {
		ids[history[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, len(history), "the archive overlap should not duplicate orders across the seeded crawls")
	assert.Contains(t, ids, "SPD-OLD", "the archive crawl should add the order past the listing's reach")
}

// TestGetSpreadOrderHistoryStopsWhenTheArchivePageStalls guards the archive
// crawl's no-progress stop: a full archive page whose last order ID repeats
// the requested endId cursor must not loop forever.
func TestGetSpreadOrderHistoryStopsWhenTheArchivePageStalls(t *testing.T) {
	t.Parallel()

	page := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		page = append(page, map[string]string{
			"sprdId":    "BTC-USDT_BTC-USDT",
			"ordId":     fmt.Sprintf("SPD-%03d", i),
			"cTime":     strconv.FormatInt(1700000000000, 10),
			"uTime":     strconv.FormatInt(1700000000000, 10),
			"state":     "filled",
			"ordType":   "limit",
			"side":      "buy",
			"sz":        "1",
			"px":        "42000",
			"accFillSz": "1",
			"avgPx":     "42000",
		})
	}
	var mu sync.Mutex
	var archiveEndIDs []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sprd/orders-history":
			writeOKXData(t, w, []map[string]string{})
		case "/sprd/orders-history-archive":
			endID := r.URL.Query().Get("endId")
			mu.Lock()
			repeated := slices.Contains(archiveEndIDs, endID)
			archiveEndIDs = append(archiveEndIDs, endID)
			mu.Unlock()
			if repeated {
				// Nothing here fails a request, so a repeat means the cursor never advanced.
				t.Errorf("endId %q requested again", endID)
				http.Error(w, "repeated endId", http.StatusBadRequest)
				return
			}
			// The mock keeps returning the same full page whatever the cursor.
			writeOKXData(t, w, page)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread,
		Type:      order.AnyType,
		Side:      order.AnySide,
		StartTime: time.Now().Add(-kline.OneMonth.Duration()),
	})
	require.NoError(t, err, "GetOrderHistory must stop when the archive page holds no new spread order")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"", "SPD-099"}, archiveEndIDs, "the archive crawl should stop when the endId cursor stops advancing")
	assert.Len(t, history, orderListPageSize, "the archive's orders should be recorded once")
}

// TestGetOrderHistoryIncludesZeroFillCanceledFromTheSevenDayListing guards
// the 7 day listing beside the archive: the archive does not contain canceled
// orders without any fills, so a zero-fill canceled order must come from the
// 7 day listing without duplicating the archive's rows.
func TestGetOrderHistoryIncludesZeroFillCanceledFromTheSevenDayListing(t *testing.T) {
	t.Parallel()

	start := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	filled := map[string]string{
		"instId": mainPair.String(), "ordId": "FILLED-1", "ordType": orderLimit, "side": "buy",
		"state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "42000", "px": "42000",
		"cTime": strconv.FormatInt(start.Add(time.Minute).UnixMilli(), 10),
	}
	zeroFillCanceled := map[string]string{
		"instId": mainPair.String(), "ordId": "CANCELED-1", "ordType": orderLimit, "side": "buy",
		"state": "canceled", "sz": "1", "accFillSz": "0", "avgPx": "0", "px": "42000",
		"cTime": strconv.FormatInt(start.Add(2*time.Minute).UnixMilli(), 10),
	}

	t.Run("the zero-fill canceled order comes from the 7 day listing", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-history-archive":
				writeOKXData(t, w, []map[string]string{filled})
			case "/trade/orders-history":
				// Only the 7 day listing keeps canceled orders without fills.
				writeOKXData(t, w, []map[string]string{zeroFillCanceled, filled})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			StartTime: start, Pairs: currency.Pairs{mainPair},
		})
		require.NoError(t, err, "GetOrderHistory must not error when the 7 day listing adds the zero-fill canceled order")
		ids := make([]string, 0, len(history))
		for i := range history {
			ids = append(ids, history[i].OrderID)
		}
		assert.ElementsMatch(t, []string{"FILLED-1", "CANCELED-1"}, ids, "the crawl should keep the zero-fill canceled order beside the archive's rows without duplicating them")
	})

	t.Run("a window closed past the 7 day listing still reads it", func(t *testing.T) {
		t.Parallel()
		// OKX lists an order there for 7 days after it completes, so an older
		// order cancelled without fills within the last 2 hours is only there.
		oldCanceled := map[string]string{
			"instId": mainPair.String(), "ordId": "OLD-CANCELED", "ordType": orderLimit, "side": "buy",
			"state": "canceled", "sz": "1", "accFillSz": "0", "avgPx": "0", "px": "42000",
			"cTime": strconv.FormatInt(time.Now().Add(-10*24*time.Hour).UnixMilli(), 10),
		}
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-history-archive":
				writeOKXData(t, w, []map[string]string{})
			case "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{oldCanceled})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			StartTime: time.Now().Add(-11 * 24 * time.Hour), EndTime: time.Now().Add(-9 * 24 * time.Hour),
			Pairs: currency.Pairs{mainPair},
		})
		require.NoError(t, err, "GetOrderHistory must not error when the window closes past the 7 day listing")
		require.Len(t, history, 1, "the 7 day listing must supply an older order cancelled without fills")
		assert.Equal(t, "OLD-CANCELED", history[0].OrderID, "the 7 day listing's order should be returned")
	})
}

// TestPendingOrderTypeFilter guards the pending order type filter through both
// wrappers: OKX returns the order types listed in ordType, which must cover
// every OKX order type read back as the requested type and time in force.
func TestPendingOrderTypeFilter(t *testing.T) {
	t.Parallel()
	pending := []map[string]string{
		{"instId": "BTC-USDT", "ordId": "LIMIT-1", "ordType": orderLimit, "side": "buy", "state": "live", "cTime": "1700000000008"},
		{"instId": "BTC-USDT", "ordId": "POST-1", "ordType": orderPostOnly, "side": "buy", "state": "live", "cTime": "1700000000007"},
		{"instId": "BTC-USDT", "ordId": "FOK-1", "ordType": orderFOK, "side": "buy", "state": "live", "cTime": "1700000000006"},
		{"instId": "BTC-USDT", "ordId": "IOC-1", "ordType": orderIOC, "side": "buy", "state": "live", "cTime": "1700000000005"},
		{"instId": "BTC-USDT", "ordId": "MMP-1", "ordType": orderMarketMakerProtection, "side": "buy", "state": "live", "cTime": "1700000000004"},
		{"instId": "BTC-USDT", "ordId": "MMP-PO-1", "ordType": orderMarketMakerProtectionAndPostOnly, "side": "buy", "state": "live", "cTime": "1700000000003"},
		{"instId": "BTC-USDT", "ordId": "OPTIMAL-1", "ordType": orderOptimalLimitIOC, "side": "buy", "state": "live", "cTime": "1700000000002"},
		{"instId": "BTC-USDT", "ordId": "RPI-1", "ordType": orderRPI, "side": "buy", "state": "live", "cTime": "1700000000001"},
		{"instId": "BTC-USDT", "ordId": "OPFOK-1", "ordType": orderOptionFOK, "side": "buy", "state": "live", "cTime": "1700000000000"},
	}
	var mu sync.Mutex
	var cancelled []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			ordTypes := strings.Split(r.URL.Query().Get("ordType"), ",")
			ords := make([]map[string]string, 0, len(pending))
			for _, o := range pending {
				if ordTypes[0] == "" || slices.Contains(ordTypes, o["ordType"]) {
					ords = append(ords, o)
				}
			}
			writeOKXData(t, w, ords)
		case "/trade/cancel-batch-orders":
			var reqs []CancelOrderRequestParam
			if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			respData := make([]map[string]string, 0, len(reqs))
			mu.Lock()
			for x := range reqs {
				cancelled = append(cancelled, reqs[x].OrderID)
				respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			mu.Unlock()
			writeOKXData(t, w, respData)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	cancelAll := func(c *order.Cancel) ([]string, error) {
		mu.Lock()
		cancelled = nil
		mu.Unlock()
		_, err := e.CancelAllOrders(t.Context(), c)
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(cancelled), err
	}

	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       []string
	}{
		{order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "FOK-1", "IOC-1", "RPI-1", "OPFOK-1"}},
		{order.Limit, order.PostOnly, []string{"POST-1"}},
		{order.Limit, order.FillOrKill, []string{"FOK-1", "OPFOK-1"}},
		{order.Limit, order.ImmediateOrCancel, []string{"IOC-1"}},
		{order.MarketMakerProtection, order.UnknownTIF, []string{"MMP-1", "MMP-PO-1"}},
		{order.MarketMakerProtection, order.PostOnly, []string{"MMP-PO-1"}},
	} {
		resp, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s %s", tc.orderType, tc.tif)
		ids := make([]string, 0, len(resp))
		for x := range resp {
			ids = append(ids, resp[x].OrderID)
		}
		assert.ElementsMatchf(t, tc.exp, ids, "GetActiveOrders should return every order read back as %s %s", tc.orderType, tc.tif)

		got, err := cancelAll(&order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: tc.orderType, TimeInForce: tc.tif})
		require.NoErrorf(t, err, "CancelAllOrders must not error for %s %s", tc.orderType, tc.tif)
		assert.ElementsMatchf(t, tc.exp, got, "CancelAllOrders should cancel every order read back as %s %s, and nothing else", tc.orderType, tc.tif)
	}

	got, err := cancelAll(&order.Cancel{Type: order.Limit})
	require.NoError(t, err, "CancelAllOrders must not error for a cancel scoped by type alone")
	assert.ElementsMatch(t, []string{"LIMIT-1", "POST-1", "FOK-1", "IOC-1", "RPI-1", "OPFOK-1"}, got, "a cancel scoped by type alone should cancel every limit order")

	got, err = cancelAll(&order.Cancel{Type: order.Limit, OrderID: "POST-1"})
	require.NoError(t, err, "CancelAllOrders must not error for an order ID within the requested type")
	assert.Equal(t, []string{"POST-1"}, got, "an order ID within the requested type should be cancelled")

	got, err = cancelAll(&order.Cancel{Type: order.Limit, OrderID: "OPTIMAL-1"})
	require.NoError(t, err, "CancelAllOrders must not error for an order ID outside the requested type")
	assert.Empty(t, got, "an order ID outside the requested type should not be cancelled")

	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Stop, Side: order.AnySide})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetActiveOrders should reject an order type OKX does not list")
	_, err = cancelAll(&order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Stop})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CancelAllOrders should reject an order type OKX does not list")
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
		if r.URL.Path != "/trade/cancel-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []any{})
	}))
	err := e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-1", Type: order.Trigger})
	assert.ErrorIs(t, err, common.ErrNoResponse, "an algo cancel reply without results should return ErrNoResponse")
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

func TestWebsocketTradingCapabilitiesDeclared(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	ws := e.Features.Supports.WebsocketCapabilities
	assert.True(t, ws.SubmitOrder, "WebsocketCapabilities should declare SubmitOrder")
	assert.True(t, ws.SubmitOrders, "WebsocketCapabilities should declare SubmitOrders")
	assert.True(t, ws.CancelOrder, "WebsocketCapabilities should declare CancelOrder")
	assert.True(t, ws.CancelOrders, "WebsocketCapabilities should declare CancelOrders")
	assert.True(t, ws.ModifyOrder, "WebsocketCapabilities should declare ModifyOrder")
}

// TestWebsocketOrderMethodsGuards covers the validation and unsupported-path
// behaviour of the explicit websocket order methods. Every check here must
// trigger before any websocket request transmits.
func TestWebsocketOrderMethodsGuards(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("submit order guards", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketSubmitOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrder must error for a nil submission")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo order type should not transmit over the websocket")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      spreadPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spread,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread order should not transmit on the private connection")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})

	t.Run("submit orders batch guards", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketSubmitOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrders must error for an empty batch")

		_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo order in the batch should stop the batch before transmission")
	})

	t.Run("modify order guards", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketModifyOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "WebsocketModifyOrder must error for a nil modification")

		_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:  e.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			OrderID:   "1234",
			Amount:    1,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread amend should not transmit on the private connection")

		_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
			Amount:    1,
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})

	t.Run("cancel order guards", func(t *testing.T) {
		t.Parallel()
		err := e.WebsocketCancelOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelOrder must error for a nil cancellation")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			OrderID:   "1234",
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread cancel should not transmit on the private connection")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "1234",
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel should not transmit over the websocket")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})

	t.Run("cancel batch orders guards", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketCancelBatchOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelBatchOrders must error for an empty batch")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), make([]order.Cancel, 21))
		require.ErrorIs(t, err, errExceedLimit, "WebsocketCancelBatchOrders must error for more than 20 orders")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread cancel should not transmit on the private connection")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel should not transmit over the websocket")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})

	t.Run("cancel all orders guards", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketCancelAllOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelAllOrders must error for a nil cancellation")

		_, err = e.WebsocketCancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spread,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread cancel-all should not transmit on the private connection")

		_, err = e.WebsocketCancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Type:      order.Trigger,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel-all should not transmit over the websocket")
	})

	t.Run("cached instrument code reaches the websocket transport", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		e.instrumentsInfoMap = make(map[string][]Instrument)
		e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		require.Error(t, err, "WebsocketSubmitOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		})
		require.Error(t, err, "WebsocketCancelOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}})
		require.Error(t, err, "WebsocketCancelBatchOrders must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})

	t.Run("mixed batch sends nothing", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		e.instrumentsInfoMap = make(map[string][]Instrument)
		e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}, {
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an invalid second order should fail the whole batch before transmission")
	})

	t.Run("mixed cancel batch sends nothing", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		e.instrumentsInfoMap = make(map[string][]Instrument)
		e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}, {
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "5678",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an invalid second order should fail the whole batch before transmission")
	})
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
		if r.URL.Path != "/trade/cancel-algos" {
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

// TestCancelBatchOrdersAlgoPartialSuccess guards the algo partial-success
// path: OKX answers a partly failed algo batch with code 2 and per-order
// results, which must be reported alongside the error, skipping any row that
// names no algo order.
func TestCancelBatchOrdersAlgoPartialSuccess(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[null,` +
			`{"algoId":"ALGO-OK","sCode":"0","sMsg":""},` +
			`{"algoId":"ALGO-BAD","sCode":"51000","sMsg":"The algo order does not exist"}]}`))
	}))
	resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-OK", Type: order.Trigger},
		{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-BAD", Type: order.Trigger},
	})
	require.ErrorIs(t, err, errPartialSuccess, "a partially successful algo batch must report its error")
	require.NotNil(t, resp, "CancelBatchOrders must return the partial results")
	assert.Equal(t, map[string]string{
		"ALGO-OK":  order.Cancelled.String(),
		"ALGO-BAD": "The algo order does not exist",
	}, resp.Status, "a partially successful algo batch should report every per-order result")
}

// TestCancelResultsUsable guards the gate that decides whether per-order cancel
// results are recorded alongside a batch error: only a partial success returns
// fully decoded results with its error.
func TestCancelResultsUsable(t *testing.T) {
	t.Parallel()
	assert.True(t, cancelResultsUsable(nil), "a successful batch should report usable results")
	assert.True(t, cancelResultsUsable(errPartialSuccess), "a partial success should report usable results")
	assert.True(t, cancelResultsUsable(common.AppendError(errPartialSuccess, errors.New("order 3: does not exist"))), "a joined partial success should still report usable results")
	assert.False(t, cancelResultsUsable(errors.New("batch one failed")), "any other error should report unusable results")
}

// TestOrderListsIncludeStartTime guards the StartTime bound of GetActiveOrders
// and GetOrderHistory: an order created exactly at StartTime is returned, and
// the listing stops at the first older order.
func TestOrderListsIncludeStartTime(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	row := func(id string, created time.Time) map[string]string {
		return map[string]string{"instId": mainPair.String(), "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "cTime": strconv.FormatInt(created.UnixMilli(), 10)}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeOKXData(t, w, []map[string]string{row("NEWER", start.Add(time.Minute)), row("AT-START", start), row("OLDER", start.Add(-time.Minute))})
	}))
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: start})
	require.NoError(t, err, "GetActiveOrders must not error")
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: start, Pairs: currency.Pairs{mainPair}})
	require.NoError(t, err, "GetOrderHistory must not error")
	for name, orders := range map[string]order.FilteredOrders{"GetActiveOrders": active, "GetOrderHistory": history} {
		ids := make([]string, 0, len(orders))
		for i := range orders {
			ids = append(ids, orders[i].OrderID)
		}
		assert.ElementsMatchf(t, []string{"NEWER", "AT-START"}, ids, "%s should return the orders created from StartTime on", name)
	}
}

// TestBatchAndGridReplies guards two REST replies: a partially successful
// batch placement returns its rows beside the error, and a failed grid amend
// reports OKX's status message.
func TestBatchAndGridReplies(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/batch-orders":
			_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"1","sCode":"0","sMsg":""},{"ordId":"","sCode":"51008","sMsg":"Order failed. Insufficient USDT balance in account."}]}`))
		case "/tradingBot/grid/amend-order-algo":
			_, _ = w.Write([]byte(`{"code":"1","msg":"Operation failed.","data":[{"algoId":"","sCode":"51000","sMsg":"Parameter algoId error"}]}`))
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	placed, err := e.PlaceMultipleOrders(t.Context(), []PlaceOrderRequestParam{
		{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderLimit, Amount: 1, Price: 1},
		{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderLimit, Amount: 1, Price: 1},
	})
	require.ErrorIs(t, err, errPartialSuccess, "PlaceMultipleOrders must return the partial success")
	require.Len(t, placed, 2, "PlaceMultipleOrders must return every row beside the error")
	assert.Equal(t, "1", placed[0].OrderID, "the placed order should keep its order ID")
	assert.Equal(t, int64(51008), placed[1].StatusCode, "the failed order should keep its status code")

	_, err = e.AmendGridAlgoOrder(t.Context(), &GridAlgoOrderAmend{AlgoID: "1", InstrumentID: "BTC-USDT-SWAP", StopLossTriggerPrice: 1})
	assert.ErrorContains(t, err, "Parameter algoId error", "AmendGridAlgoOrder should report OKX's status message")
}

// newMockWebsocketExchange returns an exchange whose authenticated websocket is
// available to the wrapper, with every REST endpoint pointed at the same test
// server and mainPair's instrument code cached. rest answers REST requests, and
// wsReply returns the code and data of OKX's reply to each websocket operation.
func newMockWebsocketExchange(t *testing.T, rest http.HandlerFunc, wsReply func(op string, args json.RawMessage) (code string, data any)) *Exchange {
	t.Helper()
	e := testexch.MockWsInstance[Exchange](t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			rest(w, r)
			return
		}
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			if string(msg) == "ping" {
				return nil
			}
			var req struct {
				ID        string          `json:"id"`
				Operation string          `json:"op"`
				Arguments json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			code, data := wsReply(req.Operation, req.Arguments)
			reply, err := json.Marshal(map[string]any{"id": req.ID, "op": req.Operation, "code": code, "msg": "", "data": data})
			if err != nil {
				return err
			}
			return c.WriteMessage(gws.TextMessage, reply)
		})
	}))
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error for RestSpot")
	for k := range e.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, e.API.Endpoints.SetRunningURL(k, mockURL+"/"), "SetRunningURL must not error for %s", k)
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must be available to the wrapper")
	e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
	return e
}

// TestOrderMethodTransports guards the transport contract: with an
// authenticated websocket available to the wrapper and the instrument code
// cached, the standard order methods still place, amend and cancel over REST,
// and only the Websocket* methods send websocket operations.
func TestOrderMethodTransports(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var restPaths, wsOps []string
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/trade/orders-pending" {
			writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1", "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"}})
			return
		}
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}, func(op string, _ json.RawMessage) (string, any) {
		mu.Lock()
		wsOps = append(wsOps, op)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	sent := func() (rest, ws []string) {
		mu.Lock()
		defer mu.Unlock()
		rest, ws = restPaths, wsOps
		restPaths, wsOps = nil, nil
		return rest, ws
	}
	submit := &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}
	modify := &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Amount: 2, Price: 1}
	cancel := &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"}

	_, err := e.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error")
	_, err = e.ModifyOrder(t.Context(), modify)
	require.NoError(t, err, "ModifyOrder must not error")
	require.NoError(t, e.CancelOrder(t.Context(), cancel), "CancelOrder must not error")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "CancelBatchOrders must not error")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	rest, ws := sent()
	assert.Equal(t, []string{"/trade/order", "/trade/amend-order", "/trade/cancel-order", "/trade/cancel-batch-orders", "/trade/orders-pending", "/trade/cancel-batch-orders"}, rest, "each standard order method should reach its REST endpoint")
	assert.Empty(t, ws, "no standard order method should send a websocket operation")

	_, err = e.WebsocketSubmitOrder(t.Context(), submit)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	_, err = e.WebsocketModifyOrder(t.Context(), modify)
	require.NoError(t, err, "WebsocketModifyOrder must not error")
	require.NoError(t, e.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must not error")
	rest, ws = sent()
	assert.Empty(t, rest, "no websocket order method should send a REST request")
	assert.Equal(t, []string{"order", "amend-order", "cancel-order"}, ws, "each websocket order method should send its websocket operation")
}

// TestWebsocketSubmitOrdersReplies guards how batch submission reports OKX's
// rows: each is matched to its order by position, a partial success keeps its
// error beside the placed orders, and a reply without one row per order is
// rejected rather than misattributed.
func TestWebsocketSubmitOrdersReplies(t *testing.T) {
	t.Parallel()
	first := map[string]string{"ordId": "1", "sCode": "0", "sMsg": ""}
	second := map[string]string{"ordId": "2", "sCode": "0", "sMsg": ""}
	failed := map[string]string{"ordId": "", "sCode": "51008", "sMsg": "Order failed. Insufficient USDT balance in account"}
	for _, tc := range []struct {
		name   string
		code   string
		data   []map[string]string
		exp    []string
		expErr error
	}{
		{"all placed", "0", []map[string]string{first, second}, []string{"1 BUY", "2 SELL"}, nil},
		{"partial success", "2", []map[string]string{first, failed}, []string{"1 BUY", ""}, errPartialSuccess},
		{"null row", "2", []map[string]string{nil, second}, []string{"", "2 SELL"}, errPartialSuccess},
		{"more rows than orders", "0", []map[string]string{first, second, second}, []string{}, common.ErrInvalidResponse},
		{"success without rows", "0", []map[string]string{}, []string{}, common.ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
			}, func(string, json.RawMessage) (string, any) {
				return tc.code, tc.data
			})
			resp, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{
				{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
				{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 2},
			})
			if tc.expErr == nil {
				require.NoError(t, err, "WebsocketSubmitOrders must not error")
			} else {
				require.ErrorIs(t, err, tc.expErr, "WebsocketSubmitOrders must return the expected error")
			}
			got := make([]string, 0, len(resp))
			for _, r := range resp {
				if r == nil {
					got = append(got, "")
					continue
				}
				got = append(got, r.OrderID+" "+r.Side.String())
			}
			assert.Equal(t, tc.exp, got, "WebsocketSubmitOrders should report each placed order ID against its own order")
		})
	}
}

// TestWebsocketCancelBatchOrdersPartialSuccess guards the websocket batch cancel
// results: a partial success records each order's own outcome beside the error.
func TestWebsocketCancelBatchOrdersPartialSuccess(t *testing.T) {
	t.Parallel()
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected REST request %s", r.URL.Path)
		http.NotFound(w, r)
	}, func(string, json.RawMessage) (string, any) {
		return "2", []map[string]string{
			{"ordId": "1", "sCode": "0", "sMsg": ""},
			{"ordId": "2", "sCode": "51400", "sMsg": "Order cancellation failed as the order has been filled, canceled or does not exist"},
		}
	})
	resp, err := e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{
		{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"},
		{Pair: mainPair, AssetType: asset.Spot, OrderID: "2"},
	})
	require.ErrorIs(t, err, errPartialSuccess, "WebsocketCancelBatchOrders must return the partial success")
	require.NotNil(t, resp, "WebsocketCancelBatchOrders must return the results beside the error")
	assert.Equal(t, map[string]string{
		"1": order.Cancelled.String(),
		"2": "Order cancellation failed as the order has been filled, canceled or does not exist",
	}, resp.Status, "each order should report its own outcome")
}

// TestWebsocketCancelAllOrdersSkipsUncachedInstruments guards websocket
// cancel-all: pending orders are cancelled over the websocket in batches of 20
// with their instrument codes, and an order whose instrument has no cached code
// is reported in the error rather than stopping the rest.
func TestWebsocketCancelAllOrdersSkipsUncachedInstruments(t *testing.T) {
	t.Parallel()
	pending := make([]map[string]string, 0, 26)
	for i := range 25 {
		pending = append(pending, map[string]string{"instId": mainPair.String(), "ordId": strconv.Itoa(i), "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"})
	}
	pending = append(pending, map[string]string{"instId": "XRP-USDT", "ordId": "UNCACHED-1", "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"})
	var mu sync.Mutex
	var batches [][]CancelOrderRequestParam
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-pending" {
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, pending)
	}, func(op string, args json.RawMessage) (string, any) {
		var reqs []CancelOrderRequestParam
		if err := json.Unmarshal(args, &reqs); err != nil {
			t.Errorf("decoding %s arguments should not error: %v", op, err)
		}
		mu.Lock()
		batches = append(batches, reqs)
		mu.Unlock()
		rows := make([]map[string]string, 0, len(reqs))
		for x := range reqs {
			rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0", "sMsg": ""})
		}
		return "0", rows
	})

	resp, err := e.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "WebsocketCancelAllOrders must report the order it cannot address")
	assert.ErrorContains(t, err, "UNCACHED-1", "the error should name the order left open")
	assert.Len(t, resp.Status, 25, "every addressable order should be cancelled")
	assert.NotContains(t, resp.Status, "UNCACHED-1", "the unaddressable order should not be reported cancelled")

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, batches, 2, "25 orders must be cancelled in two batches")
	assert.Len(t, batches[0], 20, "the first batch should hold 20 orders")
	assert.Len(t, batches[1], 5, "the second batch should hold the other 5 orders")
	for _, b := range batches {
		for x := range b {
			assert.EqualValues(t, 12345, b[x].InstrumentIDCode, "each cancel should carry the cached instrument code")
		}
	}
}

// TestOrderAmounts guards the order sizes GetOrderInfo reports: an order sized
// in the quote currency reports its amounts in the base currency, as the
// websocket order channel does for the captured order in testdata/wsOrders.json,
// and a filled order has nothing remaining.
func TestOrderAmounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                     string
		row                                      map[string]string
		amount, executed, remaining, quoteAmount float64
	}{
		{
			"quote sized, filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "sell", "cTime": "1694153250532", "sz": "10", "tgtCcy": "quote_ccy", "accFillSz": "0.00038128", "avgPx": "26228.1", "state": "filled"},
			0.00038128, 0.00038128, 0, 10,
		},
		{
			"quote sized, partially filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0.001", "avgPx": "25000", "state": "partially_filled"},
			0.004, 0.001, 0.003, 100,
		},
		{
			"quote sized, live",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0", "avgPx": "", "state": "live"},
			0, 0, 0, 100,
		},
		{
			"quote sized, partially filled, zero avgPx",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0", "avgPx": "", "state": "partially_filled"},
			0, 0, 0, 100,
		},
		{
			"base sized, partially filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "1", "tgtCcy": "base_ccy", "accFillSz": "0.5", "avgPx": "25000", "state": "partially_filled"},
			1, 0.5, 0.5, 0,
		},
		{
			"base sized, filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "sell", "cTime": "1694153250532", "sz": "1", "tgtCcy": "base_ccy", "accFillSz": "1", "avgPx": "25000", "state": "filled"},
			1, 1, 0, 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeOKXData(t, w, []map[string]string{tc.row})
			}))
			detail, err := e.GetOrderInfo(t.Context(), "1", mainPair, asset.Spot)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.amount, detail.Amount, "Amount should be in the base currency")
			assert.Equal(t, tc.executed, detail.ExecutedAmount, "ExecutedAmount should match accFillSz")
			assert.Equal(t, tc.remaining, detail.RemainingAmount, "RemainingAmount should be the unfilled base amount")
			assert.Equal(t, tc.quoteAmount, detail.QuoteAmount, "QuoteAmount should carry the quote-currency size when set")
		})
	}
}

// TestSpreadOrderDetails guards the GetOrderInfo spread order path: it
// preserves the time in force orderTypeFromString returns, and reports the
// accumulated fill size rather than the last fill.
func TestSpreadOrderDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		ordType             string
		state               string
		sz                  string
		accFillSz           string
		oType               order.Type
		tif                 order.TimeInForce
		executed, remaining float64
	}{
		{"limit", orderLimit, "live", "1", "0.3", order.Limit, order.UnknownTIF, 0.3, 0.7},
		{"post_only", orderPostOnly, "live", "1", "0.3", order.Limit, order.PostOnly, 0.3, 0.7},
		{"ioc", orderIOC, "live", "1", "0.3", order.Limit, order.ImmediateOrCancel, 0.3, 0.7},
		{"op_fok", orderOptionFOK, "live", "1", "0.3", order.Limit, order.FillOrKill, 0.3, 0.7},
		{"filled", orderLimit, "filled", "1", "1", order.Limit, order.UnknownTIF, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spreadRow := map[string]string{
				"sprdId":    "BTC-USDT_BTC-USDT",
				"ordId":     "1",
				"ordType":   tc.ordType,
				"side":      "buy",
				"state":     tc.state,
				"cTime":     "1700000000000",
				"sz":        tc.sz,
				"accFillSz": tc.accFillSz,
				"px":        "100",
				"avgPx":     "90",
			}
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeOKXData(t, w, spreadRow)
			}))

			detail, err := e.GetOrderInfo(t.Context(), "1", currency.Pair{}, asset.Spread)
			require.NoError(t, err, "GetOrderInfo must not error for a spread order")
			format, err := e.GetPairFormat(asset.Spread, true)
			require.NoError(t, err, "GetPairFormat must not error for the spread asset")
			expPair, err := currency.NewPairDelimiter("BTC-USDT_BTC-USDT", format.Delimiter)
			require.NoError(t, err, "the expected spread pair must parse")
			assert.Equal(t, expPair, detail.Pair, "GetOrderInfo should parse the pair from the spread ID like the other spread listings")
			assert.Equal(t, tc.oType, detail.Type, "GetOrderInfo should read the order type")
			assert.Equal(t, tc.tif, detail.TimeInForce, "GetOrderInfo should preserve the time in force")
			assert.Equal(t, tc.executed, detail.ExecutedAmount, "GetOrderInfo should report the accumulated fill")
			assert.Equal(t, tc.remaining, detail.RemainingAmount, "GetOrderInfo should report the unfilled remainder")
			assert.Equal(t, 90.0, detail.AverageExecutedPrice, "GetOrderInfo should read the average filled price")
			assert.InDelta(t, 90*tc.executed, detail.Cost, 1e-9, "the spread cost should be the average filled price times the accumulated fill, not the order price")
		})
	}
}

// TestSpreadOrderHistoryAnyType guards the spread GetOrderHistory path: an
// AnyType request reaches OKX without an ordType filter instead of being
// rejected before sending, and its rows keep orderTypeFromString's typing.
// TestSpreadOrderHistoryAnyType guards an AnyType spread history request and
// the spread order row mapping: the ordType filter is dropped, and every
// order.Detail field comes from the listing row.
func TestSpreadOrderHistoryAnyType(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sprd/orders-history":
			assert.Empty(t, r.URL.Query().Get("ordType"), "an AnyType request should not send an ordType filter")
			writeOKXData(t, w, []map[string]string{
				{
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderPostOnly, "side": "buy", "state": "live",
					"sz": "1", "accFillSz": "0.3", "px": "100", "avgPx": "90",
					"cTime": "1700000000000", "uTime": "1700000000000",
				},
				{
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "2", "ordType": orderLimit, "side": "sell", "state": "filled",
					"sz": "2", "accFillSz": "2", "px": "90", "avgPx": "88",
					"cTime": "1700000000001", "uTime": "1700000000001",
				},
			})
		case "/sprd/orders-history-archive":
			// A zero StartTime reaches back past the 21 day listing, so the archive is also crawled.
			writeOKXData(t, w, []map[string]string{})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetOrderHistory must not reject order.AnyType")
	require.Len(t, history, 2, "GetOrderHistory must return every spread order")
	rows := make(map[string]order.Detail, len(history))
	for i := range history {
		rows[history[i].OrderID] = history[i]
	}
	first := rows["1"]
	assert.Equal(t, order.Limit, first.Type, "GetOrderHistory should read the order type")
	assert.Equal(t, order.PostOnly, first.TimeInForce, "GetOrderHistory should preserve the time in force")
	assert.Equal(t, order.Buy, first.Side, "GetOrderHistory should read the order side")
	assert.Equal(t, order.Active, first.Status, "GetOrderHistory should read the order state")
	assert.Equal(t, 100.0, first.Price, "GetOrderHistory should read the limit price")
	assert.Equal(t, 90.0, first.AverageExecutedPrice, "GetOrderHistory should read the average filled price")
	assert.InDelta(t, 27.0, first.Cost, 1e-9, "the spread cost should be the average filled price times the accumulated fill")
	assert.Equal(t, 1.0, first.Amount, "GetOrderHistory should read the order size")
	assert.Equal(t, 0.3, first.ExecutedAmount, "GetOrderHistory should read the accumulated fill size")
	assert.InDelta(t, 0.7, first.RemainingAmount, 1e-9, "an unfilled spread order should keep its remaining amount")
	assert.Equal(t, asset.Spread, first.AssetType, "GetOrderHistory should tag the spread asset type")
	assert.Equal(t, "BTC", first.Pair.Base.String(), "the pair base should be the spread ID head under the dash delimiter")
	assert.Equal(t, "USDT_BTC-USDT", first.Pair.Quote.String(), "the pair quote should keep the remainder of the spread ID")
	second := rows["2"]
	assert.Equal(t, order.Filled, second.Status, "GetOrderHistory should read the filled state")
	assert.Equal(t, 0.0, second.RemainingAmount, "a filled spread order should have nothing remaining")
	assert.InDelta(t, 176.0, second.Cost, 1e-9, "the spread cost should be the average filled price times the accumulated fill")
}

// TestOCOStopLossTriggerPrice guards the OCO submit path: the stop loss leg
// triggers at the stop loss price, not the take profit price.
func TestOCOStopLossTriggerPrice(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/order-algo" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the algo order request body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:   e.Name,
		Pair:       mainPair,
		AssetType:  asset.Spot,
		Side:       order.Sell,
		Type:       order.OCO,
		Amount:     1,
		MarginType: margin.NoMargin,
		RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 110},
			StopLoss:   order.RiskManagement{Price: 90},
		},
	})
	require.NoError(t, err, "SubmitOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	var sent AlgoOrderParams
	require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
	assert.Equal(t, 110.0, sent.TakeProfitTriggerPrice, "the take profit leg should trigger at the take profit price")
	assert.Equal(t, 90.0, sent.StopLossTriggerPrice, "the stop loss leg should trigger at the stop loss price")
}

// TestSubmitOrderLimitTIFsRequestOrdTypes guards the ordType a limit order
// sends for each time in force: post_only, fok and ioc must map to their
// documented placement ordTypes instead of a plain resting limit.
func TestSubmitOrderLimitTIFsRequestOrdTypes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tif  order.TimeInForce
		exp  string
	}{
		{"plain limit", order.UnknownTIF, orderLimit},
		{"post only", order.PostOnly, orderPostOnly},
		{"fill or kill", order.FillOrKill, orderFOK},
		{"immediate or cancel", order.ImmediateOrCancel, orderIOC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/trade/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the place order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:    e.Name,
				Pair:        mainPair,
				AssetType:   asset.Spot,
				Side:        order.Buy,
				Type:        order.Limit,
				TimeInForce: tc.tif,
				Amount:      1,
				Price:       1,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
			assert.Equal(t, tc.exp, sent["ordType"], "the submitted limit order should send the documented ordType")
		})
	}
}

// TestWebsocketSubmitOrderLimitTIFsRequestOrdTypes guards the same ordType
// mapping on the websocket transport: the private order frames must carry the
// documented placement ordTypes too.
func TestWebsocketSubmitOrderLimitTIFsRequestOrdTypes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tif  order.TimeInForce
		exp  string
	}{
		{"plain limit", order.UnknownTIF, orderLimit},
		{"post only", order.PostOnly, orderPostOnly},
		{"fill or kill", order.FillOrKill, orderFOK},
		{"immediate or cancel", order.ImmediateOrCancel, orderIOC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var args string
			e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
			}, func(op string, raw json.RawMessage) (string, any) {
				if op != "order" {
					t.Errorf("unexpected websocket operation %s", op)
					return "1", nil
				}
				mu.Lock()
				args = string(raw)
				mu.Unlock()
				return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
			})
			_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
				Exchange:    e.Name,
				Pair:        mainPair,
				AssetType:   asset.Spot,
				Side:        order.Buy,
				Type:        order.Limit,
				TimeInForce: tc.tif,
				Amount:      1,
				Price:       1,
			})
			require.NoError(t, err, "WebsocketSubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent []map[string]any
			require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
			assert.Equal(t, tc.exp, sent[0]["ordType"], "the websocket limit order should send the documented ordType")
		})
	}
}

// TestSubmitOrderContractSendsSideAndReduceOnly guards the contract order
// body: OKX requires side for every instrument, and reduceOnly applies to
// MARGIN and FUTURES/SWAP orders only.
func TestSubmitOrderContractSendsSideAndReduceOnly(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		assetType     asset.Item
		side          order.Side
		reduceOnly    bool
		expSide       string
		expPosSide    any // nil when the field must be absent
		expReduceOnly any // nil when the field must be absent
	}{
		{"perpetual swap reduce only sell", asset.PerpetualSwap, order.Sell, true, "sell", positionSideNet, "true"},
		{"futures long", asset.Futures, order.Long, false, "buy", positionSideNet, nil},
		{"margin reduce only", asset.Margin, order.Sell, true, "sell", nil, "true"},
		{"options send side", asset.Options, order.Buy, false, "buy", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
					return
				}
				if r.URL.Path != "/trade/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the place order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:    e.Name,
				Pair:        mainPair,
				AssetType:   tc.assetType,
				Side:        tc.side,
				Type:        order.Limit,
				TimeInForce: order.GoodTillCancel,
				Amount:      1,
				Price:       1,
				MarginType:  margin.Multi,
				ReduceOnly:  tc.reduceOnly,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
			assert.Equal(t, tc.expSide, sent["side"], "the order should send OKX's required side")
			assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side")
			assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should match the documented applicability")
		})
	}
}

// TestWebsocketSubmitOrderContractSendsSideAndReduceOnly guards the contract
// fields on the websocket transport: the private order frames must carry
// OKX's required side, the position side and the reduce-only flag.
func TestWebsocketSubmitOrderContractSendsSideAndReduceOnly(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var args string
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/config" {
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
			return
		}
		t.Errorf("unexpected REST request %s", r.URL.Path)
		http.NotFound(w, r)
	}, func(op string, raw json.RawMessage) (string, any) {
		if op != "order" {
			t.Errorf("unexpected websocket operation %s", op)
			return "1", nil
		}
		mu.Lock()
		args = string(raw)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:   e.Name,
		Pair:       mainPair,
		AssetType:  asset.PerpetualSwap,
		Side:       order.Sell,
		Type:       order.Limit,
		Amount:     1,
		Price:      1,
		MarginType: margin.Multi,
		ReduceOnly: true,
	})
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	var sent []map[string]any
	require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
	assert.Equal(t, "sell", sent[0]["side"], "the websocket contract order should send OKX's required side")
	assert.Equal(t, positionSideNet, sent[0]["posSide"], "the websocket contract order should send the position side")
	assert.Equal(t, "true", sent[0]["reduceOnly"], "the websocket contract order should carry the reduce-only flag")
}

// TestPlaceOrderRequestParamValidateRequiresSideForContracts guards the
// documented requirement that side is set for every instrument type.
func TestPlaceOrderRequestParamValidateRequiresSideForContracts(t *testing.T) {
	t.Parallel()
	for _, assetType := range []asset.Item{asset.Futures, asset.PerpetualSwap, asset.Options} {
		arg := &PlaceOrderRequestParam{
			AssetType:    assetType,
			InstrumentID: mainPair.String(),
			TradeMode:    TradeModeCross,
			OrderType:    orderLimit,
			Amount:       1,
			PositionSide: "long",
		}
		err := arg.Validate()
		assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "Validate should reject a %s order without side", assetType)
		arg.Side = "buy"
		require.NoErrorf(t, arg.Validate(), "Validate must accept a %s order with side", assetType)
	}
}

// TestCancelBatchOrdersAlgoChunksPerTen guards the algo cancel transport: the
// documented cancel-algos endpoint accepts at most 10 algo orders per request,
// so a larger batch is split, and every chunk's results reach the status map.
func TestCancelBatchOrdersAlgoChunksPerTen(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var chunkSizes []int
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		raw, readErr := io.ReadAll(r.Body)
		assert.NoError(t, readErr, "reading the algo cancel body should not error")
		var sent []map[string]string
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Errorf("the algo cancel body should decode: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rows := make([]map[string]string, 0, len(sent))
		for _, a := range sent {
			rows = append(rows, map[string]string{"algoId": a["algoId"], "sCode": "0", "sMsg": ""})
		}
		mu.Lock()
		chunkSizes = append(chunkSizes, len(rows))
		mu.Unlock()
		writeOKXData(t, w, rows)
	}))
	cancels := make([]order.Cancel, 0, 15)
	for i := range 15 {
		cancels = append(cancels, order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-" + strconv.Itoa(i), Type: order.Trigger})
	}
	resp, err := e.CancelBatchOrders(t.Context(), cancels)
	require.NoError(t, err, "CancelBatchOrders must not error when every chunk succeeds")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{10, 5}, chunkSizes, "the algo cancels should be sent in requests of at most 10")
	assert.Len(t, resp.Status, 15, "every cancelled algo order should appear in the status map")
}

// TestCancelBatchOrdersAlgoChunkContinuesAfterPartialSuccess guards the chunk
// loop: a partially successful chunk reports its rows and still leaves the
// chunks after it to cancel.
func TestCancelBatchOrdersAlgoChunkContinuesAfterPartialSuccess(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var requests int
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/cancel-algos" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		raw, readErr := io.ReadAll(r.Body)
		assert.NoError(t, readErr, "reading the algo cancel body should not error")
		var sent []map[string]string
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Errorf("the algo cancel body should decode: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rows := make([]map[string]string, 0, len(sent))
		for _, a := range sent {
			rows = append(rows, map[string]string{"algoId": a["algoId"], "sCode": "0", "sMsg": ""})
		}
		mu.Lock()
		requests++
		partial := requests == 1
		mu.Unlock()
		if partial {
			// OKX answers one row per requested algo order, and the chunk's
			// second order failed.
			rows[1]["sCode"] = "51000"
			rows[1]["sMsg"] = "The algo order does not exist"
			body, mErr := json.Marshal(map[string]any{"code": "2", "msg": "Bulk operation partially succeeded.", "data": rows})
			if mErr != nil {
				t.Errorf("marshalling the partial success reply should not error: %v", mErr)
				http.Error(w, "bad reply", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
			return
		}
		writeOKXData(t, w, rows)
	}))
	cancels := make([]order.Cancel, 0, 12)
	for i := range 12 {
		cancels = append(cancels, order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-" + strconv.Itoa(i), Type: order.Trigger})
	}
	resp, err := e.CancelBatchOrders(t.Context(), cancels)
	require.ErrorIs(t, err, errPartialSuccess, "the partially successful chunk must report its error")
	require.NotNil(t, resp, "CancelBatchOrders must return the results beside the error")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, requests, "the second chunk should still be sent after a partial success")
	for i := range 12 {
		exp := order.Cancelled.String()
		if i == 1 {
			exp = "The algo order does not exist"
		}
		assert.Equalf(t, exp, resp.Status["ALGO-"+strconv.Itoa(i)], "the chunk results should record ALGO-%d's own outcome", i)
	}
}

// TestGetOrderHistoryPaginatesWithAfterCursor guards the order history
// crawl: OKX caps the archive response at orderListPageSize records and
// pages the remainder with the after cursor, which returns the records
// earlier than the requested order ID. Unlike the end timestamp, the order
// ID cursor is exclusive, so orders sharing one creation millisecond across
// the page boundary cannot repeat on the next page.
func TestGetOrderHistoryPaginatesWithAfterCursor(t *testing.T) {
	t.Parallel()
	base := time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond)
	endTime := base.Add(128 * time.Millisecond)
	// ORD-099 and ORD-100 share one creation millisecond across the page
	// boundary: the after cursor must page past them without overlap.
	cTime := func(i int) time.Time {
		if i == 100 {
			return base.Add(99 * time.Millisecond)
		}
		return base.Add(time.Duration(i) * time.Millisecond)
	}
	row := func(i int) map[string]string {
		created := strconv.FormatInt(cTime(i).UnixMilli(), 10)
		return map[string]string{
			"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i),
			"cTime": created, "uTime": created, "state": "filled",
			"ordType": orderLimit, "side": "buy", "sz": "1",
			"px": "42000", "accFillSz": "1", "avgPx": "42000",
			"fee": "-0.01", "feeCcy": "USDT",
			// The rebate currency is unrelated to the pair, so the cost asset
			// must come from the pair quote, not this field.
			"rebateCcy": "DOGE",
		}
	}
	// The mock pages by the after order ID cursor: the first page holds the
	// 100 newest orders, and the second page holds the 30 older ones.
	page1 := make([]map[string]string, 0, orderListPageSize)
	for i := 129; i >= 30; i-- {
		page1 = append(page1, row(i))
	}
	page2 := make([]map[string]string, 0, 30)
	for i := 29; i >= 0; i-- {
		page2 = append(page2, row(i))
	}
	pages := map[string][]map[string]string{
		"":        page1,
		"ORD-030": page2,
	}
	var mu sync.Mutex
	var afterCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/trade/orders-history" {
			// The window reaches the 7 day listing, which has nothing to add
			// to the archive's rows here.
			writeOKXData(t, w, []map[string]string{})
			return
		}
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		after := r.URL.Query().Get("after")
		mu.Lock()
		repeated := slices.Contains(afterCursors, after)
		afterCursors = append(afterCursors, after)
		page := pages[after]
		mu.Unlock()
		if repeated {
			// A repeat means the after cursor never advanced.
			t.Errorf("after %q requested again", after)
			http.Error(w, "repeated after", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
		StartTime: base.Add(-time.Second), EndTime: endTime,
		Pairs: currency.Pairs{mainPair},
	})
	require.NoError(t, err, "GetOrderHistory must not error when the archive spans pages")
	mu.Lock()
	cursors := slices.Clone(afterCursors)
	mu.Unlock()
	assert.Equal(t, []string{"", "ORD-030"}, cursors, "the second page should be requested with the first page's last order ID as the after cursor")
	require.Len(t, history, 130, "a paginated crawl must return every order")
	ids := make(map[string]struct{}, len(history))
	for x := range history {
		ids[history[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, len(history), "a paginated crawl should not duplicate orders")
	assert.Contains(t, ids, "ORD-100", "orders sharing a millisecond across the page boundary should all be returned")
	assert.Contains(t, ids, "ORD-000", "orders past the first page should be returned")
	assert.Equal(t, mainPair.Quote, history[0].CostAsset, "the cost asset should be the pair quote, not the order's rebate currency")
	assert.Equal(t, 0.01, history[0].Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
	assert.Equal(t, "USDT", history[0].FeeAsset.String(), "the fee asset should come from the response")
}

// TestGetOrderHistoryWithoutPairsReturnsEveryInstrument guards the official
// request semantics: the history endpoints require only the instrument type,
// so a request without pairs returns every instrument's orders instead of
// erroring.
func TestGetOrderHistoryWithoutPairsReturnsEveryInstrument(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-24*time.Hour).Truncate(time.Millisecond).UnixMilli(), 10)
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/trade/orders-history" {
			writeOKXData(t, w, []map[string]string{})
			return
		}
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{
			{
				"instId": mainPair.String(), "ordId": "ORD-A",
				"cTime": created, "uTime": created, "state": "filled",
				"ordType": orderLimit, "side": "buy", "sz": "1",
				"px": "42000", "accFillSz": "1", "avgPx": "42000",
			},
			{
				"instId": "ETH-USDT", "ordId": "ORD-B",
				"cTime": created, "uTime": created, "state": "filled",
				"ordType": orderLimit, "side": "sell", "sz": "2",
				"px": "3000", "accFillSz": "2", "avgPx": "3000",
			},
		})
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
		StartTime: time.Now().Add(-25 * time.Hour), EndTime: time.Now().Add(-23 * time.Hour),
	})
	require.NoError(t, err, "GetOrderHistory must not error when no pairs are set")
	require.Len(t, history, 2, "a request without pairs must return every instrument's orders")
	ids := make(map[string]struct{}, len(history))
	for x := range history {
		ids[history[x].OrderID] = struct{}{}
	}
	assert.Contains(t, ids, "ORD-A", "the first instrument's orders should be returned without pairs")
	assert.Contains(t, ids, "ORD-B", "the second instrument's orders should be returned without pairs")
}

// TestGetOrderHistoryStopsWhenAFullPageIsSeen guards the no-progress stop: a
// full page whose last order ID repeats the requested after cursor must not
// loop forever on the same page.
func TestGetOrderHistoryStopsWhenAFullPageIsSeen(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-24*time.Hour).Truncate(time.Millisecond).UnixMilli(), 10)
	page := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		// Every order shares the one creation millisecond, and the mock keeps
		// returning the page the after cursor already passed.
		page = append(page, map[string]string{
			"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i),
			"cTime": created, "uTime": created, "state": "filled",
			"ordType": orderLimit, "side": "buy", "sz": "1",
			"px": "42000", "accFillSz": "1", "avgPx": "42000",
		})
	}
	var mu sync.Mutex
	var afterCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/trade/orders-history" {
			// The window reaches the 7 day listing, which has nothing to add
			// to the archive's rows here.
			writeOKXData(t, w, []map[string]string{})
			return
		}
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		after := r.URL.Query().Get("after")
		mu.Lock()
		repeated := slices.Contains(afterCursors, after)
		afterCursors = append(afterCursors, after)
		mu.Unlock()
		if repeated {
			// A repeat means the after cursor never advanced.
			t.Errorf("after %q requested again", after)
			http.Error(w, "repeated after", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
		Pairs: currency.Pairs{mainPair},
	})
	require.NoError(t, err, "GetOrderHistory must stop when a full page holds no new order")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"", "ORD-099"}, afterCursors, "the crawl should stop when the after cursor stops advancing")
	assert.Len(t, history, orderListPageSize, "the first page's orders should be returned once")
}

// TestGetOrderHistoryFromOrderIDSeedsTheAfterCursor guards the FromOrderID
// request field: OKX pages the archive from an order ID with the after
// cursor, so the first request must carry FromOrderID as the after cursor.
func TestGetOrderHistoryFromOrderIDSeedsTheAfterCursor(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/trade/orders-history" {
			// The 7 day listing is crawled with the same seed and adds
			// nothing here.
			writeOKXData(t, w, []map[string]string{})
			return
		}
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, "SEED-1", r.URL.Query().Get("after"), "the first request should carry FromOrderID as the after cursor")
		writeOKXData(t, w, []map[string]string{})
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
		FromOrderID: "SEED-1",
		Pairs:       currency.Pairs{mainPair},
	})
	require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds the crawl")
	assert.Empty(t, history, "an empty archive page should return no orders")
}

// TestSubmitOrderPerpetualSwapPositionMode guards the placement of perpetual
// swap orders against the account's position mode: net mode pairs reduceOnly
// with posSide net, and long/short mode pairs the side with the position
// side so that a reduce-only sell closes the long position.
func TestSubmitOrderPerpetualSwapPositionMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		mode          string
		side          order.Side
		reduceOnly    bool
		expPosSide    string
		expReduceOnly any // nil when the field must be absent
	}{
		{"net mode open long", positionModeNet, order.Buy, false, positionSideNet, nil},
		{"net mode close long", positionModeNet, order.Sell, true, positionSideNet, "true"},
		{"long short open long", positionModeLongShort, order.Buy, false, positionSideLong, nil},
		{"long short open short", positionModeLongShort, order.Sell, false, positionSideShort, nil},
		{"long short close long", positionModeLongShort, order.Sell, true, positionSideLong, nil},
		{"long short close short", positionModeLongShort, order.Buy, true, positionSideShort, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
					return
				}
				if r.URL.Path != "/trade/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the place order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:   e.Name,
				Pair:       mainPair,
				AssetType:  asset.PerpetualSwap,
				Side:       tc.side,
				Type:       order.Limit,
				Amount:     1,
				Price:      1,
				MarginType: margin.Multi,
				ReduceOnly: tc.reduceOnly,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
			assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side the account's mode requires")
			assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should follow the position mode")
		})
	}
	t.Run("unknown mode", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"posMode": "weird_mode"}})
				return
			}
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:   e.Name,
			Pair:       mainPair,
			AssetType:  asset.PerpetualSwap,
			Side:       order.Buy,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			MarginType: margin.Multi,
		})
		require.ErrorIs(t, err, errInvalidPositionMode, "an undocumented position mode must stop the submit")
	})
	t.Run("fetch failure", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				http.Error(w, "account config unavailable", http.StatusInternalServerError)
				return
			}
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:   e.Name,
			Pair:       mainPair,
			AssetType:  asset.PerpetualSwap,
			Side:       order.Buy,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			MarginType: margin.Multi,
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "a failed position mode fetch must stop the submit")
	})
	t.Run("empty reply", func(t *testing.T) {
		t.Parallel()
		for _, data := range []any{nil, []any{nil}} {
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, data)
					return
				}
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:   e.Name,
				Pair:       mainPair,
				AssetType:  asset.PerpetualSwap,
				Side:       order.Buy,
				Type:       order.Limit,
				Amount:     1,
				Price:      1,
				MarginType: margin.Multi,
			})
			require.ErrorIsf(t, err, common.ErrNoResponse, "a %v account configuration reply must stop the submit", data)
		}
	})
}

// TestWebsocketSubmitOrderPerpetualSwapPositionMode guards the same
// mode-aware placement on the websocket transport.
func TestWebsocketSubmitOrderPerpetualSwapPositionMode(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var args string
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/config" {
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
			return
		}
		t.Errorf("unexpected REST request %s", r.URL.Path)
		http.NotFound(w, r)
	}, func(op string, raw json.RawMessage) (string, any) {
		if op != "order" {
			t.Errorf("unexpected websocket operation %s", op)
			return "1", nil
		}
		mu.Lock()
		args = string(raw)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:   e.Name,
		Pair:       mainPair,
		AssetType:  asset.PerpetualSwap,
		Side:       order.Sell,
		Type:       order.Limit,
		Amount:     1,
		Price:      1,
		MarginType: margin.Multi,
		ReduceOnly: true,
	})
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	var sent []map[string]any
	require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
	assert.Equal(t, "sell", sent[0]["side"], "the websocket contract order should send OKX's required side")
	assert.Equal(t, positionSideLong, sent[0]["posSide"], "a reduce-only sell should close the long position in long/short mode")
	_, has := sent[0]["reduceOnly"]
	assert.False(t, has, "long/short mode should not send reduceOnly")
}

// TestSubmitOrderDefaultsTradeMode guards the tdMode default: OKX requires
// tdMode on every order, and the zero margin type defaults to cash for spot
// and cross for perpetual swap.
func TestSubmitOrderDefaultsTradeMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		assetType asset.Item
		exp       string
	}{
		{"spot defaults to cash", asset.Spot, TradeModeCash},
		{"margin defaults to cross", asset.Margin, TradeModeCross},
		{"futures defaults to cross", asset.Futures, TradeModeCross},
		{"perpetual swap defaults to cross", asset.PerpetualSwap, TradeModeCross},
		{"options defaults to cross", asset.Options, TradeModeCross},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
					return
				}
				if r.URL.Path != "/trade/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the place order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:  e.Name,
				Pair:      mainPair,
				AssetType: tc.assetType,
				Side:      order.Buy,
				Type:      order.Limit,
				Amount:    1,
				Price:     1,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
			assert.Equal(t, tc.exp, sent["tdMode"], "the order should send the documented default trade mode")
		})
	}
}

// TestSubmitOrderMarketOmitsPrice guards the market order body: px is only
// applicable to the limit-style order types, so a market order carries no
// price even when the submit set one.
func TestSubmitOrderMarketOmitsPrice(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the place order body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      mainPair,
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Market,
		Amount:    1,
		Price:     999,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
	_, has := sent["px"]
	assert.False(t, has, "a market order should not send a price")
}

// TestSubmitOrderTriggerSendsOrderPrice guards the trigger order body: OKX
// requires orderPx, which the wrapper must take from the submitted price
// instead of dropping it.
func TestSubmitOrderTriggerSendsOrderPrice(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/config" {
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
			return
		}
		if r.URL.Path != "/trade/order-algo" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the algo order body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:     e.Name,
		Pair:         mainPair,
		AssetType:    asset.PerpetualSwap,
		Side:         order.Long,
		Type:         order.Trigger,
		Amount:       1,
		Price:        100,
		TriggerPrice: 110,
		MarginType:   margin.Multi,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	// Both submits run before the body is read under the mutex, so a second
	// request cannot deadlock against the handler's own lock.
	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:     e.Name,
		Pair:         mainPair,
		AssetType:    asset.PerpetualSwap,
		Side:         order.Long,
		Type:         order.Trigger,
		Amount:       1,
		TriggerPrice: 110,
		MarginType:   margin.Multi,
	})
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "a trigger order without an order price should be rejected before sending")

	mu.Lock()
	defer mu.Unlock()
	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
	assert.Equal(t, "100", sent["orderPx"], "the trigger order should carry the submitted order price")
	assert.Equal(t, "buy", sent["side"], "the trigger order should send OKX's required side")
	assert.Equal(t, positionSideNet, sent["posSide"], "the trigger order should send the position side the account's mode requires")
}

// TestSubmitOrderTWAPSendsTimeInterval guards the TWAP interval transport:
// OKX documents the field as timeInterval carrying seconds, not interval
// carrying a duration code.
func TestSubmitOrderTWAPSendsTimeInterval(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/order-algo" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the algo order body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      e.Name,
		Pair:          mainPair,
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.TWAP,
		Amount:        1,
		Price:         100,
		TrackingMode:  order.Percentage,
		TrackingValue: 0.5,
		MarginType:    margin.NoMargin,
	})
	require.NoError(t, err, "SubmitOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
	assert.Equal(t, "900", sent["timeInterval"], "the TWAP order should send its interval as documented seconds")
	_, has := sent["interval"]
	assert.False(t, has, "the TWAP order should not send the undocumented interval field")
}

// TestSubmitOrderChaseSendsDocumentedMaxChaseType guards the chase order
// mapping: OKX's maxChaseType accepts distance and ratio, not the percentage
// name the tracking mode string carries.
func TestSubmitOrderChaseSendsDocumentedMaxChaseType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		trackingMode order.TrackingMode
		exp          string
	}{
		{"percentage maps to ratio", order.Percentage, "ratio"},
		{"distance maps to distance", order.Distance, "distance"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
					return
				}
				if r.URL.Path != "/trade/order-algo" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the algo order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:      e.Name,
				Pair:          mainPair,
				AssetType:     asset.PerpetualSwap,
				Side:          order.Long,
				Type:          order.Chase,
				Amount:        1,
				Price:         100,
				TrackingMode:  tc.trackingMode,
				TrackingValue: 0.5,
				MarginType:    margin.Multi,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
			assert.Equal(t, tc.exp, sent["maxChaseType"], "the chase order should send the documented maximum chase type")
		})
	}
}

// TestPlaceMultipleOrdersLimitsBatchSize guards the documented batch limit:
// a maximum of 20 orders can be placed per request.
func TestPlaceMultipleOrdersLimitsBatchSize(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request path %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	args := make([]PlaceOrderRequestParam, 21)
	_, err := e.PlaceMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "an oversized REST batch should be rejected before transmission")
}

// TestWSPlaceMultipleOrdersLimitsBatchSize guards the same documented batch
// limit on the websocket transport.
func TestWSPlaceMultipleOrdersLimitsBatchSize(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request path %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	args := make([]PlaceOrderRequestParam, 21)
	_, err := e.WSPlaceMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "an oversized websocket batch should be rejected before transmission")
}

// TestSubmitOrderContractsFollowPositionMode guards the position side of
// futures and perpetual swap orders, plain and algo: OKX's position mode
// covers FUTURES and SWAP alike, and in long/short mode a close is expressed
// by the position side, since reduceOnly does not apply there.
func TestSubmitOrderContractsFollowPositionMode(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap} {
		for _, oType := range []order.Type{order.Limit, order.ConditionalStop} {
			for _, tc := range []struct {
				mode          string
				side          order.Side
				reduceOnly    bool
				expPosSide    string
				expReduceOnly any // nil when the field must be absent
			}{
				{positionModeNet, order.Buy, false, positionSideNet, nil},
				{positionModeNet, order.Sell, false, positionSideNet, nil},
				{positionModeNet, order.Sell, true, positionSideNet, "true"},
				{positionModeNet, order.Buy, true, positionSideNet, "true"},
				{positionModeLongShort, order.Buy, false, positionSideLong, nil},
				{positionModeLongShort, order.Sell, false, positionSideShort, nil},
				{positionModeLongShort, order.Sell, true, positionSideLong, nil},
				{positionModeLongShort, order.Buy, true, positionSideShort, nil},
			} {
				t.Run(fmt.Sprintf("%s %s %s %s reduce only %t", a, oType, tc.mode, tc.side, tc.reduceOnly), func(t *testing.T) {
					t.Parallel()
					var mu sync.Mutex
					var body []byte
					e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case "/account/config":
							writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
							return
						case "/trade/order", "/trade/order-algo":
						default:
							t.Errorf("unexpected request path %s", r.URL.Path)
							http.NotFound(w, r)
							return
						}
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the order body should not error")
						mu.Lock()
						body = b
						mu.Unlock()
						writeOKXData(t, w, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
					}))
					_, err := e.SubmitOrder(t.Context(), &order.Submit{
						Exchange:     e.Name,
						Pair:         mainPair,
						AssetType:    a,
						Side:         tc.side,
						Type:         oType,
						Amount:       1,
						Price:        1,
						TriggerPrice: 2,
						MarginType:   margin.Multi,
						ReduceOnly:   tc.reduceOnly,
					})
					require.NoError(t, err, "SubmitOrder must not error")
					mu.Lock()
					defer mu.Unlock()
					var sent map[string]any
					require.NoError(t, json.Unmarshal(body, &sent), "the order request body must decode")
					assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side the account's mode requires")
					if oType == order.Limit {
						assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should follow the position mode")
					}
				})
			}
		}
	}
}

// TestSetPositionModeRefreshesOrderPlacement guards the position mode cache
// refresh: a successful switch steers later contract order placement without
// another account configuration fetch, and an empty or unrecognised success
// reply reports an error rather than caching an unconfirmed mode.
func TestSetPositionModeRefreshesOrderPlacement(t *testing.T) {
	t.Parallel()
	t.Run("placement follows the confirmed mode", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/set-position-mode":
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
				return
			case "/trade/order":
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the order body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		}))
		_, err := e.SetPositionMode(t.Context(), positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		_, err = e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:   e.Name,
			Pair:       mainPair,
			AssetType:  asset.Futures,
			Side:       order.Buy,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			MarginType: margin.Multi,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent map[string]any
		require.NoError(t, json.Unmarshal(body, &sent), "the order request body must decode")
		assert.Equal(t, positionSideLong, sent["posSide"], "the order should follow the confirmed position mode without refetching it")
	})
	for _, tc := range []struct {
		name string
		data any
		err  error
	}{
		{"empty reply", nil, common.ErrNoResponse},
		{"null row", []any{nil}, common.ErrNoResponse},
		{"unrecognised mode", []map[string]string{{"posMode": "reverse"}}, errInvalidPositionMode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/account/set-position-mode" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				writeOKXData(t, w, tc.data)
			}))
			_, err := e.SetPositionMode(t.Context(), positionModeLongShort)
			require.ErrorIsf(t, err, tc.err, "SetPositionMode must report a %s", tc.name)
			assert.Empty(t, e.accountPositionModes, "an unconfirmed mode switch should not refresh the cache")
		})
	}
}

// TestLeverageFollowsContractPositionMode guards the leverage wrappers
// against OKX's documented posSide rule: isolated contract leverage takes a
// posSide only in long/short mode, where the side selects the position; net
// mode and cross margin leave it unset. Options carry no leverage setting.
func TestLeverageFollowsContractPositionMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		mode      string // empty when no account configuration fetch is expected
		margin    margin.Type
		side      order.Side
		expPos    string // expected posSide in the set-leverage body; "" means absent
		expGet    float64
		expSetErr error
	}{
		{"long/short mode isolated long", positionModeLongShort, margin.Isolated, order.Long, positionSideLong, 5, nil},
		{"long/short mode isolated short", positionModeLongShort, margin.Isolated, order.Short, positionSideShort, 10, nil},
		{"long/short mode isolated without a side", positionModeLongShort, margin.Isolated, order.UnknownSide, "", 0, order.ErrSideIsInvalid},
		{"net mode isolated without a side", positionModeNet, margin.Isolated, order.UnknownSide, "", 3, nil},
		{"cross sends no position side and fetches no mode", "", margin.Multi, order.UnknownSide, "", 7, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/account/config":
					if tc.mode == "" {
						t.Errorf("unexpected account configuration fetch for %s margin", tc.margin)
					}
					writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
				case "/account/set-leverage":
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the set leverage body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"lever": "5", "mgnMode": "isolated", "posSide": tc.expPos, "instId": perpetualSwapPair.String()}})
				case "/account/leverage-info":
					if r.URL.Query().Get("mgnMode") == TradeModeCross {
						writeOKXData(t, w, []map[string]string{
							{"instId": perpetualSwapPair.String(), "mgnMode": "cross", "posSide": "", "lever": "7"},
						})
						return
					}
					writeOKXData(t, w, []map[string]string{
						{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideLong, "lever": "5"},
						{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideShort, "lever": "10"},
						{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideNet, "lever": "3"},
					})
				default:
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			err := e.SetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, tc.margin, 5, tc.side)
			if tc.expSetErr != nil {
				require.ErrorIsf(t, err, tc.expSetErr, "SetLeverage must reject %s", tc.name)
				assert.Empty(t, body, "a rejected leverage setting should not reach OKX")
			} else {
				require.NoError(t, err, "SetLeverage must not error")
				mu.Lock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the set leverage body must decode")
				mu.Unlock()
				if tc.expPos == "" {
					_, has := sent["posSide"]
					assert.False(t, has, "posSide should be omitted")
				} else {
					assert.Equal(t, tc.expPos, sent["posSide"], "the leverage should target the position side the mode requires")
				}
			}

			got, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, tc.margin, tc.side)
			if tc.expSetErr != nil {
				require.ErrorIsf(t, err, tc.expSetErr, "GetLeverage must reject %s", tc.name)
			} else {
				require.NoError(t, err, "GetLeverage must not error")
				assert.Equalf(t, tc.expGet, got, "GetLeverage should return the row for %s", tc.name)
			}
		})
	}
	t.Run("options carry no leverage setting", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		err := e.SetLeverage(t.Context(), asset.Options, mainPair, margin.Multi, 5, order.UnknownSide)
		assert.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage should reject options, which OKX gives no leverage setting")
		_, err = e.GetLeverage(t.Context(), asset.Options, mainPair, margin.Multi, order.UnknownSide)
		assert.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage should reject options, which OKX gives no leverage setting")
	})
}

// TestLimitMakerOrdersAmendAndCancelByType guards the order types the amend
// and cancel switches accept: SubmitOrder places LimitMaker orders as
// post_only, so amending or cancelling one by the type it was submitted with
// must reach OKX on every transport.
func TestLimitMakerOrdersAmendAndCancelByType(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var restPaths, wsOps []string
	var submitBody []byte
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/trade/order" {
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the place order body should not error")
			mu.Lock()
			submitBody = b
			mu.Unlock()
		}
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}, func(op string, _ json.RawMessage) (string, any) {
		mu.Lock()
		wsOps = append(wsOps, op)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	sent := func() (rest, ws []string) {
		mu.Lock()
		defer mu.Unlock()
		rest, ws = restPaths, wsOps
		restPaths, wsOps = nil, nil
		return rest, ws
	}
	submit := &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.LimitMaker, Amount: 1, Price: 1}
	modify := &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Type: order.LimitMaker, Amount: 2, Price: 1}
	cancel := &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Type: order.LimitMaker}

	_, err := e.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error")
	_, err = e.ModifyOrder(t.Context(), modify)
	require.NoError(t, err, "ModifyOrder must accept a LimitMaker type")
	require.NoError(t, e.CancelOrder(t.Context(), cancel), "CancelOrder must accept a LimitMaker type")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "CancelBatchOrders must accept a LimitMaker type")
	rest, ws := sent()
	assert.Equal(t, []string{"/trade/order", "/trade/amend-order", "/trade/cancel-order", "/trade/cancel-batch-orders"}, rest, "each standard order method should reach its REST endpoint")
	assert.Empty(t, ws, "no standard order method should send a websocket operation")
	mu.Lock()
	var placed map[string]any
	require.NoError(t, json.Unmarshal(submitBody, &placed), "the place order request body must decode")
	mu.Unlock()
	assert.Equal(t, orderPostOnly, placed["ordType"], "a LimitMaker submit should place a post_only order")

	_, err = e.WebsocketModifyOrder(t.Context(), modify)
	require.NoError(t, err, "WebsocketModifyOrder must accept a LimitMaker type")
	require.NoError(t, e.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must accept a LimitMaker type")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "WebsocketCancelBatchOrders must accept a LimitMaker type")
	rest, ws = sent()
	assert.Empty(t, rest, "no websocket order method should send a REST request")
	assert.Equal(t, []string{"amend-order", "cancel-order", "batch-cancel-orders"}, ws, "each websocket order method should send its websocket operation")
}

// TestSpreadOrdersUseRESTWithAuthenticatedWebsocket guards the spread order
// flows: OKX accepts spread operations only on its business websocket, and the
// WS spread helpers send on the private one, so SubmitOrder, ModifyOrder and
// CancelOrder must place, amend and cancel spread orders over REST even when
// an authenticated websocket is available.
func TestSpreadOrdersUseRESTWithAuthenticatedWebsocket(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var wsOps []string
	var restPaths []string
	var spreadIDs []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
				// The connections outlive the test, and OKX's keepalive ping is not JSON
				if string(msg) == "ping" {
					return nil
				}
				var req struct {
					ID        string `json:"id"`
					Operation string `json:"op"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				mu.Lock()
				wsOps = append(wsOps, req.Operation)
				mu.Unlock()
				// The error live OKX returns when a spread operation reaches the
				// private websocket connection.
				return c.WriteMessage(gws.TextMessage,
					[]byte(`{"id":"`+req.ID+`","op":"`+req.Operation+`","code":"60028","msg":"The current operation is not supported by this URL. Please use the correct WebSocket URL for the operation."}`))
			})
			return
		}
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/sprd/order" {
			var body struct {
				SpreadID string `json:"sprdId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding spread order request: %v", err)
			}
			mu.Lock()
			spreadIDs = append(spreadIDs, body.SpreadID)
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":"","ordId":"SPRD-1","clOrdId":""}]}`))
	})

	e := testexch.MockWsInstance[Exchange](t, handler)
	// MockWsInstance points RestSpotURL at the server without a trailing
	// slash, which breaks the REST path join, so every endpoint is re-pointed
	// at the mock server; the websocket connections are already established.
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "the mock server URL must be resolvable")
	b := e.GetBase()
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, mockURL+"/"), "Setup must point endpoint %s at the mock server", k)
	}
	// A connected websocket with authenticated endpoints available is what the
	// broken code branches on before sending spread operations to the private
	// connection.
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must report availability for the wrapper")

	sub, err := e.SubmitOrder(t.Context(), &order.Submit{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err, "SubmitOrder must place a spread order over REST")
	require.NotEmpty(t, sub.OrderID, "SubmitOrder must return the spread order ID")

	_, err = e.ModifyOrder(t.Context(), &order.Modify{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
		Amount:    2,
		Price:     101,
	})
	require.NoError(t, err, "ModifyOrder must amend a spread order over REST")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
	})
	require.NoError(t, err, "CancelOrder must cancel a spread order over REST")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"/sprd/order", "/sprd/amend-order", "/sprd/cancel-order"}, restPaths,
		"spread submit, amend and cancel should reach their REST endpoints")
	assert.Equal(t, []string{"BTC-USDT_BTC-USDT-SWAP"}, spreadIDs,
		"SubmitOrder should send the spread ID OKX lists")
	assert.Empty(t, wsOps, "no spread operation should be sent over the websocket")
}

// TestSubmitSpreadOrderMarketPriceAndType guards the spread submit wire
// format: the ordType follows the order type and time in force, a market
// order carries no px, and rejected submits never reach OKX.
func TestSubmitSpreadOrderMarketPriceAndType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		orderType  order.Type
		tif        order.TimeInForce
		price      float64
		expOrdType string
		expPX      bool
	}{
		{"market carries no price", order.Market, order.UnknownTIF, 0, orderMarket, false},
		{"limit follows the maker type", order.LimitMaker, order.UnknownTIF, 2, orderPostOnly, true},
		{"ioc follows the time in force", order.Limit, order.ImmediateOrCancel, 2, orderIOC, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the spread order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:    e.Name,
				Pair:        spreadPair,
				AssetType:   asset.Spread,
				Side:        order.Buy,
				Type:        tc.orderType,
				TimeInForce: tc.tif,
				Amount:      1,
				Price:       tc.price,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
			assert.Equal(t, tc.expOrdType, sent["ordType"], "the spread order should send the documented ordType")
			_, has := sent["px"]
			assert.Equal(t, tc.expPX, has, "px should follow the documented applicability")
		})
	}
	// A limit order without a price is still rejected by the spread price
	// guard, and fok is refused outright rather than silently downgraded to
	// a resting limit order; both are refused client-side, so nothing
	// reaches OKX.
	for _, tc := range []struct {
		name   string
		tif    order.TimeInForce
		price  float64
		expErr error
	}{
		{"limit without a price", order.UnknownTIF, 0, limits.ErrPriceBelowMin},
		{"fok limit", order.FillOrKill, 2, order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("a rejected spread order should not reach OKX: %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:    e.Name,
				Pair:        spreadPair,
				AssetType:   asset.Spread,
				Side:        order.Buy,
				Type:        order.Limit,
				TimeInForce: tc.tif,
				Amount:      1,
				Price:       tc.price,
			})
			require.ErrorIs(t, err, tc.expErr, "the rejected spread order must return its own error")
		})
	}
}

// TestPlaceSpreadOrderPriceGuard guards the spread price validation on the
// REST helper: the px of a spread is the differential between its legs, which
// can be negative, while a price-bearing order without any price is rejected.
func TestPlaceSpreadOrderPriceGuard(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the spread order body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}))
	_, err := e.PlaceSpreadOrder(t.Context(), &SpreadOrderParam{
		SpreadID:  spreadPair.String(),
		Side:      "buy",
		OrderType: orderLimit,
		Size:      1,
		Price:     -41.2,
	})
	require.NoError(t, err, "a negative-priced spread order must place")
	mu.Lock()
	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
	mu.Unlock()
	assert.Equal(t, "-41.2", sent["px"], "a negative spread price should send verbatim")
	_, err = e.PlaceSpreadOrder(t.Context(), &SpreadOrderParam{
		SpreadID:  spreadPair.String(),
		Side:      "buy",
		OrderType: orderLimit,
		Size:      1,
	})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "a price-bearing spread order without a price must be rejected")
}

// TestSubmitSpreadOrderReportsRowError guards the spread order error shape:
// OKX's top-level reply only says all operations failed, so the failed row's
// own code and reason must be reported beside it.
func TestSubmitSpreadOrderReportsRowError(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"1","msg":"All operations failed","data":[{"sCode":"51008","sMsg":"Order placement failed due to insufficient balance.","ordId":"","clOrdId":""}]}`))
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange: e.Name, Pair: spreadPair, AssetType: asset.Spread,
		Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2,
	})
	require.Error(t, err, "a failed spread row must error")
	assert.ErrorContains(t, err, "51008", "the error should report the row's sCode")
	assert.ErrorContains(t, err, "insufficient balance", "the error should report the row's sMsg")
}

// TestGetLeverageRateSendsCurrency guards the optional ccy parameter: a set
// currency must be forwarded to the leverage-info endpoint.
func TestGetLeverageRateSendsCurrency(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var gotCcy string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/leverage-info" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		gotCcy = r.URL.Query().Get("ccy")
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{})
	}))
	_, err := e.GetLeverageRate(t.Context(), mainPair.String(), TradeModeCross, currency.BTC)
	require.NoError(t, err, "GetLeverageRate must not error with a currency set")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "BTC", gotCcy, "the currency should be sent as the ccy parameter")
}

// TestGetLeverageWithoutConfiguredLeverage guards the sentinel error for an
// instrument that has no configured leverage rows.
func TestGetLeverageWithoutConfiguredLeverage(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/leverage-info" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{})
	}))
	_, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "a query for an instrument without configured leverage must report the missing position")
}

// TestGetLeveragePropagatesRequestErrors guards the propagation of upstream
// leverage-info failures to the caller.
func TestGetLeveragePropagatesRequestErrors(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/leverage-info" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"code":"51000","data":[],"msg":"Parameter instId error"}`, http.StatusBadRequest)
	}))
	_, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
	require.ErrorContains(t, err, "51000", "upstream leverage-info errors must propagate to the caller")
}

// TestGetLeverageRequiresMatchingPositionSideRow guards the per-side row
// matching: a side without its own leverage row must not silently read the
// opposite side's or a stale row's leverage.
func TestGetLeverageRequiresMatchingPositionSideRow(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/account/config":
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
		case "/account/leverage-info":
			writeOKXData(t, w, []map[string]string{{
				"instId": "BTC-USDT-SWAP", "mgnMode": "isolated", "posSide": "short", "lever": "3",
			}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	_, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Isolated, order.Long)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "a per-side query without the requested side's row must not fall back to another side's leverage")
	lever, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Isolated, order.Short)
	require.NoError(t, err, "GetLeverage isolated short must not error")
	assert.Equal(t, 3.0, lever, "the short side should read its own leverage row")
}

// TestLeverageRejectsUnformattablePairs guards the pair format failure paths
// in the leverage wrappers: a store without request formats must surface the
// formatting error and must not reach the API.
func TestLeverageRejectsUnformattablePairs(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	err := e.CurrencyPairs.Store(asset.PerpetualSwap, &currency.PairStore{AssetEnabled: true})
	// The mock setup shares a global pair format; a per-asset store with nil
	// formats must be authoritative for the failure to surface.
	e.CurrencyPairs.UseGlobalFormat = false
	require.NoError(t, err, "storing a formatless pair store must not error")
	err = e.SetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, 3, order.AnySide)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "SetLeverage must surface the pair format failure")
	_, err = e.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetLeverage must surface the pair format failure")
}

// TestPendingOrderPagersStopOnARepeatedPage guards the stop on a page that
// does not advance: a venue answering every cursor with the same full page
// must not keep the pending order pagers, or cancel-all, requesting it until
// the context ends.
func TestPendingOrderPagersStopOnARepeatedPage(t *testing.T) {
	t.Parallel()
	page := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		page = append(page, map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i), "ordType": orderLimit, "side": "buy", "state": "live", "sz": "1", "px": "1", "cTime": "1700000000000"})
	}
	var mu sync.Mutex
	listings := make(map[string]int)
	var cancelled []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending", "/sprd/orders-pending":
			mu.Lock()
			listings[r.URL.Path]++
			mu.Unlock()
			writeOKXData(t, w, page)
		case "/trade/cancel-batch-orders":
			var reqs []CancelOrderRequestParam
			if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			rows := make([]map[string]string, 0, len(reqs))
			mu.Lock()
			for x := range reqs {
				cancelled = append(cancelled, reqs[x].OrderID)
				rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			mu.Unlock()
			writeOKXData(t, w, rows)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// A pager that never stops fails at this deadline instead of hanging.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for _, a := range []asset.Item{asset.Spot, asset.Spread} {
		active, err := e.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: a, Type: order.AnyType, Side: order.AnySide})
		require.NoErrorf(t, err, "GetActiveOrders must stop at a repeated page for %s", a)
		assert.Lenf(t, active, orderListPageSize, "GetActiveOrders should keep the repeated page once for %s", a)
	}
	_, err := e.CancelAllOrders(ctx, &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must stop at a repeated page")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, map[string]int{"/trade/orders-pending": 4, "/sprd/orders-pending": 2}, listings, "each pager should stop at the first repeated page")
	assert.Len(t, cancelled, orderListPageSize, "cancel-all should cancel each order once")
}

// TestListOrdersByLimitMakerType guards listing by the LimitMaker type that
// SubmitOrder places as post_only: OKX reads a post_only order back as a
// post-only Limit, which a LimitMaker query must still return.
func TestListOrdersByLimitMakerType(t *testing.T) {
	t.Parallel()
	row := func(id, ordType, state string) map[string]string {
		return map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": state, "sz": "1", "px": "1", "cTime": "1700000000000"}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := "canceled"
		switch r.URL.Path {
		case "/trade/orders-pending", "/sprd/orders-pending":
			state = "live"
		case "/trade/orders-history-archive", "/trade/orders-history", "/sprd/orders-history", "/sprd/orders-history-archive":
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{row("LIMIT-1", orderLimit, state), row("POST-1", orderPostOnly, state), row("IOC-1", orderIOC, state)})
	}))
	for _, a := range []asset.Item{asset.Spot, asset.Spread} {
		req := &order.MultiOrderRequest{AssetType: a, Type: order.LimitMaker, Side: order.AnySide}
		active, err := e.GetActiveOrders(t.Context(), req)
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		history, err := e.GetOrderHistory(t.Context(), req)
		require.NoErrorf(t, err, "GetOrderHistory must not error for %s", a)
		for _, got := range []order.FilteredOrders{active, history} {
			require.Lenf(t, got, 1, "a LimitMaker query must return the post-only order for %s", a)
			assert.Equalf(t, "POST-1", got[0].OrderID, "the post-only order should be returned for %s", a)
			assert.Equalf(t, order.LimitMaker, got[0].Type, "the order should report the requested type for %s", a)
		}
	}
}

// TestSubmitOrderRejectsUnsupportedMarginType guards the trade mode default:
// only the zero margin type defaults, so an explicit margin type OKX cannot
// express fails before any request instead of trading in another mode.
func TestSubmitOrderRejectsUnsupportedMarginType(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request path %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		for _, mt := range []margin.Type{
			margin.Unknown,
			margin.Isolated | margin.Multi,
			margin.Isolated | margin.NoMargin,
			margin.Isolated | margin.SpotIsolated,
			margin.NoMargin | margin.SpotIsolated,
			margin.Isolated | margin.NoMargin | margin.SpotIsolated,
		} {
			_, err := e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: a, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: mt})
			assert.ErrorIsf(t, err, margin.ErrMarginTypeUnsupported, "SubmitOrder should reject margin type %d for %s", mt, a)
		}
	}
}

// TestSetLeverageRateNormalisesPositionSide guards the posSide case
// normalisation: any case is accepted and sent lower case, as OKX requires.
func TestSetLeverageRateNormalisesPositionSide(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/set-leverage" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the set leverage body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"lever": "5", "mgnMode": TradeModeIsolated, "posSide": positionSideLong, "instId": perpetualSwapPair.String()}})
	}))
	_, err := e.SetLeverageRate(t.Context(), &SetLeverageInput{Leverage: 5, MarginMode: TradeModeIsolated, InstrumentID: perpetualSwapPair.String(), PositionSide: "LONG"})
	require.NoError(t, err, "SetLeverageRate must accept an upper-case position side")
	mu.Lock()
	var sent map[string]any
	err = json.Unmarshal(body, &sent)
	mu.Unlock()
	require.NoError(t, err, "the set leverage body must decode")
	assert.Equal(t, positionSideLong, sent["posSide"], "the position side should be sent in lower case")

	_, err = e.SetLeverageRate(t.Context(), &SetLeverageInput{Leverage: 5, MarginMode: TradeModeIsolated, Currency: currency.USDT})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "a currency-scoped isolated leverage must be rejected")
	assert.ErrorContains(t, err, `requires "cross" margin`, "the error should name the margin mode a currency-scoped leverage requires")
}

// TestContractPositionModeKeepsConfirmedSwitch guards the cached position mode
// against a first-use fetch that returns after SetPositionMode confirmed a
// switch: the older fetched mode must not replace the confirmed one.
func TestContractPositionModeKeepsConfirmedSwitch(t *testing.T) {
	t.Parallel()
	getStarted := make(chan struct{})
	startOnce := sync.OnceFunc(func() { close(getStarted) })
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/account/config":
			startOnce()
			<-release
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
		case "/account/set-position-mode":
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// Every test in the package shares the rate limiters, so a busy run could
	// otherwise hold the fetch back past the deadline below.
	require.NoError(t, e.DisableRateLimiter(), "DisableRateLimiter must not error")
	// Registered after the mock server's cleanup, so it runs first and frees
	// a handler still waiting on release.
	t.Cleanup(releaseOnce)
	fetched := make(chan error, 1)
	go func() {
		_, err := e.contractPositionMode(t.Context())
		fetched <- err
	}()
	select {
	case <-getStarted:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the account configuration fetch must start")
	}
	_, err := e.SetPositionMode(t.Context(), positionModeLongShort)
	require.NoError(t, err, "SetPositionMode must not error")
	releaseOnce()
	select {
	case err := <-fetched:
		require.NoError(t, err, "the in-flight fetch must not error")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the in-flight fetch must return")
	}
	mode, err := e.contractPositionMode(t.Context())
	require.NoError(t, err, "contractPositionMode must not error")
	assert.Equal(t, positionModeLongShort, mode, "a confirmed switch should outlive an older fetch")
}

// TestGetOrderHistoryWindowPaging guards the history crawl against OKX's
// documented pagination: begin and end are inclusive filters applied first,
// after then pages to older order IDs, and past the page limit OKX answers
// begin without end or after with the records closest to begin, the oldest.
// Each listing holds orders the other lacks, and older ones before the window.
func TestGetOrderHistoryWindowPaging(t *testing.T) {
	t.Parallel()
	base := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	created := func(id int64) time.Time { return base.Add(time.Duration(id) * time.Second) }
	// The archive holds orders 1 to 150 and the 7 day listing 151 to 300.
	listings := map[string][2]int64{"/trade/orders-history-archive": {1, 150}, "/trade/orders-history": {151, 300}}
	// venue serves one subtest, recording the cursors it has served in
	// requested.
	venue := func(t *testing.T, requested *sync.Map, w http.ResponseWriter, r *http.Request) {
		t.Helper()
		ids, ok := listings[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		// This venue never fails a well-formed request, so a repeated cursor
		// means the crawl stopped advancing: failing it ends the crawl instead
		// of serving the same page until go test's timeout.
		if _, repeated := requested.LoadOrStore(r.URL.Path+"?after="+q.Get("after"), struct{}{}); repeated {
			t.Errorf("%s after %q requested again", r.URL.Path, q.Get("after"))
			http.Error(w, "repeated after", http.StatusBadRequest)
			return
		}
		param := func(k string) (int64, bool) {
			v, err := strconv.ParseInt(q.Get(k), 10, 64)
			return v, err == nil
		}
		begin, hasBegin := param("begin")
		end, hasEnd := param("end")
		after, hasAfter := param("after")
		var page []int64
		for id := ids[0]; id <= ids[1]; id++ {
			c := created(id).UnixMilli()
			if (hasBegin && c < begin) || (hasEnd && c > end) || (hasAfter && id >= after) {
				continue
			}
			page = append(page, id)
		}
		if len(page) > orderListPageSize {
			if hasBegin && !hasEnd && !hasAfter {
				page = page[:orderListPageSize]
			} else {
				page = page[len(page)-orderListPageSize:]
			}
		}
		rows := make([]map[string]string, 0, len(page))
		for _, id := range slices.Backward(page) {
			c := strconv.FormatInt(created(id).UnixMilli(), 10)
			rows = append(rows, map[string]string{
				"instId": mainPair.String(), "ordId": strconv.FormatInt(id, 10), "ordType": orderLimit, "side": "buy",
				"state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": c, "uTime": c,
			})
		}
		writeOKXData(t, w, rows)
	}
	for _, tc := range []struct {
		name       string
		start, end time.Time
		first      int64
		last       int64
	}{
		{"start only", created(31), time.Time{}, 31, 300},
		{"end only", time.Time{}, created(120), 1, 120},
		{"start and end", created(31), created(200), 31, 200},
		{"neither", time.Time{}, time.Time{}, 1, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requested := new(sync.Map)
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				venue(t, requested, w, r)
			}))
			history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
				StartTime: tc.start, EndTime: tc.end, Pairs: currency.Pairs{mainPair},
			})
			require.NoError(t, err, "GetOrderHistory must not error")
			got := make([]int64, 0, len(history))
			for i := range history {
				id, err := strconv.ParseInt(history[i].OrderID, 10, 64)
				require.NoError(t, err, "order IDs must parse")
				got = append(got, id)
			}
			slices.Sort(got)
			exp := make([]int64, 0, tc.last-tc.first+1)
			for id := tc.first; id <= tc.last; id++ {
				exp = append(exp, id)
			}
			assert.Equal(t, exp, got, "GetOrderHistory should return exactly the orders in the window")
		})
	}
}

// TestGetOrderHistoryRejectsUnmappedStates guards the history crawl's state
// mapping: a state the wrapper cannot map must fail the request instead of
// reporting an order with an unknown status.
func TestGetOrderHistoryRejectsUnmappedStates(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10)
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-history-archive" && r.URL.Path != "/trade/orders-history" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{{
			"instId": mainPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy",
			"state": "bogus", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": created,
		}})
	}))
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{mainPair},
	})
	require.ErrorContains(t, err, "unrecognised order status", "an unmapped order state must fail the request")
}

// TestGetActiveSpreadOrdersFromOrderIDSeedsTheEndIDCursor guards the
// FromOrderID request field for pending spread orders: it must seed the
// endId cursor, orders earlier than the ID, the direction the spread
// history crawls and the standard listing's after cursor use, and beginId,
// which returns records newer than the ID, must never go out.
func TestGetActiveSpreadOrdersFromOrderIDSeedsTheEndIDCursor(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-pending" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, "SEED-1", r.URL.Query().Get("endId"), "the first request should carry FromOrderID as the endId cursor")
		assert.Empty(t, r.URL.Query().Get("beginId"), "beginId returns records newer than the order ID and should never be sent")
		writeOKXData(t, w, []map[string]string{{
			"sprdId": spreadPair.String(), "ordId": "SPD-1", "ordType": orderLimit,
			"side": "buy", "state": "live", "sz": "1", "px": "1", "accFillSz": "0", "cTime": "1700000000000",
		}})
	}))
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide, FromOrderID: "SEED-1",
	})
	require.NoError(t, err, "GetActiveOrders must not error when FromOrderID seeds the spread crawl")
	require.Len(t, active, 1, "the seeded crawl must return the listing's orders")
	assert.Equal(t, "SPD-1", active[0].OrderID, "the pending spread order should be returned")
}

// TestSubmitSpreadOrderNegativeLimitPrice guards the negative spread price
// round trip: a spread price is the differential between its legs, so a
// negative limit price must reach OKX as-is.
func TestSubmitSpreadOrderNegativeLimitPrice(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var body []byte
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the spread order body should not error")
		mu.Lock()
		body = b
		mu.Unlock()
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}))
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      spreadPair,
		AssetType: asset.Spread,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     -41.2,
	})
	require.NoError(t, err, "SubmitOrder must accept a negative spread limit price")
	mu.Lock()
	defer mu.Unlock()
	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
	assert.Equal(t, "-41.2", sent["px"], "the negative spread price should be sent as-is")
}

// TestGetOrderHistoryNarrowsByRequestedTimeInForce guards the history result
// set's time in force narrowing: the crawls send no ordType, so a post-only
// Limit query must not return IOC orders, and a GoodTillCancel one must keep
// the limit orders OKX reads back with no time in force, on both the
// standard and the spread listings.
func TestGetOrderHistoryNarrowsByRequestedTimeInForce(t *testing.T) {
	t.Parallel()
	row := func(id, ordType string) map[string]string {
		return map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": "1700000000000"}
	}
	rows := []map[string]string{
		row("LIMIT-1", orderLimit), row("POST-1", orderPostOnly), row("IOC-1", orderIOC), row("RPI-1", orderRPI), row("FOK-1", orderFOK),
		row("OPFOK-1", orderOptionFOK), row("MMP-1", orderMarketMakerProtection), row("MMPPO-1", orderMarketMakerProtectionAndPostOnly),
		// OKX sends ordType in lower case; the read-back accepts any case.
		row("POST-2", "POST_ONLY"),
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-history-archive", "/trade/orders-history":
			writeOKXData(t, w, rows)
		case "/sprd/orders-history", "/sprd/orders-history-archive":
			// The spread listings filter on the single ordType they are sent.
			filtered := make([]map[string]string, 0, len(rows))
			for _, o := range rows {
				if ordType := r.URL.Query().Get("ordType"); ordType == "" || o["ordType"] == ordType {
					filtered = append(filtered, o)
				}
			}
			writeOKXData(t, w, filtered)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	for _, tc := range []struct {
		name      string
		assetType asset.Item
		orderType order.Type
		tif       order.TimeInForce
		exp       []string
	}{
		{"post only query returns post only orders", asset.Spot, order.Limit, order.PostOnly, []string{"POST-1", "POST-2"}},
		{"immediate or cancel query returns ioc orders", asset.Spot, order.Limit, order.ImmediateOrCancel, []string{"IOC-1"}},
		{"no time in force returns every limit order", asset.Spot, order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "IOC-1", "RPI-1", "FOK-1", "OPFOK-1", "POST-2"}},
		{"fill or kill query returns both fill or kill types", asset.Spot, order.Limit, order.FillOrKill, []string{"FOK-1", "OPFOK-1"}},
		{"good till cancel query returns the orders it places", asset.Spot, order.Limit, order.GoodTillCancel, []string{"LIMIT-1"}},
		{"good till day query returns the orders it places", asset.Spot, order.Limit, order.GoodTillDay, []string{"LIMIT-1"}},
		{"good till cancel limit maker query returns post only orders", asset.Spot, order.LimitMaker, order.GoodTillCancel, []string{"POST-1", "POST-2"}},
		{"post only market maker protection query returns its post only orders", asset.Spot, order.MarketMakerProtection, order.PostOnly, []string{"MMPPO-1"}},
		{"any type query ignores the time in force as GetActiveOrders does", asset.Spot, order.AnyType, order.PostOnly, []string{"LIMIT-1", "POST-1", "IOC-1", "RPI-1", "FOK-1", "OPFOK-1", "MMP-1", "MMPPO-1", "POST-2"}},
		{"good till cancel spread query returns the orders it places", asset.Spread, order.Limit, order.GoodTillCancel, []string{"LIMIT-1"}},
		{"spread query without a time in force returns every limit order", asset.Spread, order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "IOC-1", "RPI-1", "FOK-1", "OPFOK-1", "POST-2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: tc.assetType, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide,
			})
			require.NoError(t, err, "GetOrderHistory must not error")
			ids := make([]string, 0, len(history))
			for i := range history {
				ids = append(ids, history[i].OrderID)
			}
			assert.ElementsMatch(t, tc.exp, ids, "the history should return exactly the orders carrying the requested time in force")
		})
	}
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.LimitMaker, TimeInForce: order.FillOrKill, Side: order.AnySide,
	})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "a time in force the order type cannot take should be refused, as GetActiveOrders refuses it")
}

// TestContractPositionModeIsPerAccount guards the cached position mode
// against requests acting for different accounts: credentials carried in a
// request's context select another account, whose own mode must decide its
// orders rather than the mode of the account that filled the cache first.
func TestContractPositionModeIsPerAccount(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	modes := map[string]string{"KEY-A": positionModeNet, "KEY-B": positionModeLongShort}
	var sent []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("OK-ACCESS-KEY")
		switch r.URL.Path {
		case "/account/config":
			mu.Lock()
			mode := modes[key]
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": mode}})
		case "/account/set-position-mode":
			var req PositionMode
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding the position mode body should not error: %v", err)
				return
			}
			mu.Lock()
			modes[key] = req.PositionMode
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
		case "/trade/order":
			var req PlaceOrderRequestParam
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding the order body should not error: %v", err)
				return
			}
			mu.Lock()
			sent = append(sent, key+" "+req.PositionSide)
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	ctxFor := func(key string) context.Context {
		return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
	}
	submit := func(key string) {
		t.Helper()
		_, err := e.SubmitOrder(ctxFor(key), &order.Submit{Exchange: e.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
		require.NoErrorf(t, err, "SubmitOrder must not error for %s", key)
	}
	submit("KEY-A")
	submit("KEY-B")
	_, err := e.SetPositionMode(ctxFor("KEY-B"), positionModeNet)
	require.NoError(t, err, "SetPositionMode must not error")
	submit("KEY-A")
	submit("KEY-B")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"KEY-A net", "KEY-B long", "KEY-A net", "KEY-B net"}, sent, "each account's orders should follow its own position mode")
}

// TestContractPositionModeRefetchesAfterAnotherKeySwitches guards the cached
// position mode against two API keys of one account: a switch confirmed
// through one key must not leave the other key placing for the old mode, so
// the other key fetches its mode again rather than keep the stale one.
func TestContractPositionModeRefetchesAfterAnotherKeySwitches(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	accountMode := positionModeNet
	var sent []string
	var fetches int
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("OK-ACCESS-KEY")
		switch r.URL.Path {
		case "/account/config":
			mu.Lock()
			fetches++
			mode := accountMode
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": mode}})
		case "/account/set-position-mode":
			var req PositionMode
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding the position mode body should not error: %v", err)
				return
			}
			mu.Lock()
			accountMode = req.PositionMode
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
		case "/trade/order":
			var req PlaceOrderRequestParam
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding the order body should not error: %v", err)
				return
			}
			mu.Lock()
			sent = append(sent, key+" "+req.PositionSide)
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	ctxFor := func(key string) context.Context {
		return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
	}
	submit := func(key string) {
		t.Helper()
		_, err := e.SubmitOrder(ctxFor(key), &order.Submit{Exchange: e.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
		require.NoErrorf(t, err, "SubmitOrder must not error for %s", key)
	}
	submit("KEY-1")
	_, err := e.SetPositionMode(ctxFor("KEY-2"), positionModeLongShort)
	require.NoError(t, err, "SetPositionMode must not error")
	submit("KEY-1")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"KEY-1 net", "KEY-1 long"}, sent, "a key should follow a switch its account confirmed through another key")
	assert.Equal(t, 2, fetches, "the second order should fetch its own key's mode again rather than keep the stale one")
}

// TestContractPositionModeRefetchesWhenAFetchOverlapsASwitch guards the
// position mode against a first-use fetch through one API key that returns
// after a switch confirmed through another key of the same account: the
// fetched mode may predate the switch, so the waiting caller fetches again
// rather than use or cache it.
func TestContractPositionModeRefetchesWhenAFetchOverlapsASwitch(t *testing.T) {
	t.Parallel()
	getStarted := make(chan struct{})
	startOnce := sync.OnceFunc(func() { close(getStarted) })
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	var mu sync.Mutex
	accountMode := positionModeNet
	var fetches int
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/account/config":
			mu.Lock()
			fetches++
			first := fetches == 1
			mode := accountMode
			mu.Unlock()
			if first {
				// The first fetch reads the mode before the switch and answers after it.
				startOnce()
				<-release
			}
			writeOKXData(t, w, []map[string]string{{"posMode": mode}})
		case "/account/set-position-mode":
			mu.Lock()
			accountMode = positionModeLongShort
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// Every test in the package shares the rate limiters, so a busy run could
	// otherwise hold the fetch back past the deadline below.
	require.NoError(t, e.DisableRateLimiter(), "DisableRateLimiter must not error")
	// Registered after the mock server's cleanup, so it runs first and frees
	// a handler still waiting on release.
	t.Cleanup(releaseOnce)
	ctxFor := func(key string) context.Context {
		return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
	}
	type result struct {
		mode string
		err  error
	}
	fetched := make(chan result, 1)
	go func() {
		mode, err := e.contractPositionMode(ctxFor("KEY-1"))
		fetched <- result{mode, err}
	}()
	select {
	case <-getStarted:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the account configuration fetch must start")
	}
	_, err := e.SetPositionMode(ctxFor("KEY-2"), positionModeLongShort)
	require.NoError(t, err, "SetPositionMode must not error")
	releaseOnce()
	select {
	case got := <-fetched:
		require.NoError(t, got.err, "the in-flight lookup must not error")
		assert.Equal(t, positionModeLongShort, got.mode, "the in-flight lookup should fetch again rather than use a mode that predates the switch")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the in-flight lookup must return")
	}
	mode, err := e.contractPositionMode(ctxFor("KEY-1"))
	require.NoError(t, err, "contractPositionMode must not error")
	assert.Equal(t, positionModeLongShort, mode, "the refetched mode should be cached")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, fetches, "the overlapping fetch should be repeated once and its answer cached")
}

// TestGetOrderInfoReportsExecutionValues guards the order detail mapping:
// the average executed price, the cost of the fills and the fee the account
// was charged come from the response rather than the order price, and the
// fee keeps the websocket stream's positive-for-a-charge sign.
func TestGetOrderInfoReportsExecutionValues(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/order" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, "ORD-1", r.URL.Query().Get("ordId"), "the order detail request should identify the order")
		assert.Equal(t, mainPair.String(), r.URL.Query().Get("instId"), "the order detail request should identify the instrument")
		writeOKXData(t, w, map[string]string{
			"instId": mainPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "filled",
			"sz": "2", "accFillSz": "1.5", "avgPx": "41000", "px": "40000", "fee": "-0.01", "feeCcy": "USDT",
			"cTime": "1700000000000", "uTime": "1700000000000",
		})
	}))
	detail, err := e.GetOrderInfo(t.Context(), "ORD-1", mainPair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, 41000.0, detail.AverageExecutedPrice, "GetOrderInfo should read the average filled price")
	assert.Equal(t, 41000.0*1.5, detail.Cost, "the cost should be the average filled price times the accumulated fill, not the order price")
	assert.Equal(t, "USDT", detail.CostAsset.String(), "the cost asset should be the pair quote")
	assert.Equal(t, 0.01, detail.Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
	assert.Equal(t, "USDT", detail.FeeAsset.String(), "the fee asset should come from the response")
}

// TestOrderCost guards the order cost mapping: OKX fills spot and margin
// orders in the base currency, so their cost is in the quote currency, while
// futures, perpetual swap and option orders fill in contracts and are left
// unset.
func TestOrderCost(t *testing.T) {
	t.Parallel()
	o := &OrderDetail{AveragePrice: 41000, AccumulatedFillSize: 1.5}
	for _, tc := range []struct {
		assetType asset.Item
		cost      float64
		costAsset currency.Code
	}{
		{asset.Spot, 61500, currency.USDT},
		{asset.Margin, 61500, currency.USDT},
		{asset.Futures, 0, currency.EMPTYCODE},
		{asset.PerpetualSwap, 0, currency.EMPTYCODE},
		{asset.Options, 0, currency.EMPTYCODE},
	} {
		cost, costAsset := orderCost(tc.assetType, o, currency.NewBTCUSDT())
		assert.Equalf(t, tc.cost, cost, "orderCost should report the expected cost for %s", tc.assetType)
		assert.Equalf(t, tc.costAsset, costAsset, "orderCost should report the expected cost asset for %s", tc.assetType)
	}
}

// TestContractOrdersLeaveCostUnset guards the order detail, pending and
// history mappings against reporting a contract order's average price times
// its contracts as a cost in the pair's quote, which for a perpetual swap is
// not a currency.
func TestContractOrdersLeaveCostUnset(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
	row := func(state string) map[string]string {
		return map[string]string{
			"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": state,
			"sz": "100", "accFillSz": "50", "avgPx": "60000", "px": "60000", "fee": "-3", "feeCcy": "USDT", "cTime": created, "uTime": created,
		}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/order", "/trade/orders-pending":
			writeOKXData(t, w, []map[string]string{row("partially_filled")})
		case "/trade/orders-history", "/trade/orders-history-archive":
			writeOKXData(t, w, []map[string]string{row("canceled")})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	info, err := e.GetOrderInfo(t.Context(), "ORD-1", perpetualSwapPair, asset.PerpetualSwap)
	require.NoError(t, err, "GetOrderInfo must not error")
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, active, 1, "GetActiveOrders must return the pending order")
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, history, 1, "GetOrderHistory must return the order")
	for name, d := range map[string]*order.Detail{"GetOrderInfo": info, "GetActiveOrders": &active[0], "GetOrderHistory": &history[0]} {
		assert.Zerof(t, d.Cost, "%s should leave a contract order's cost unset", name)
		assert.Truef(t, d.CostAsset.IsEmpty(), "%s should leave a contract order's cost asset unset", name)
		assert.Equalf(t, 60000.0, d.AverageExecutedPrice, "%s should still report the average price", name)
	}
}

// TestGetActiveOrdersReportExecutionValues guards the pending order mapping:
// a partially filled resting order reports its average executed price and
// the cost of its fills, and the fee keeps the websocket stream's
// positive-for-a-charge sign.
func TestGetActiveOrdersReportExecutionValues(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-pending" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{{
			"instId": "BTC-USDT", "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "live",
			"sz": "2", "accFillSz": "0.5", "avgPx": "41000", "px": "40000", "fee": "-0.01", "feeCcy": "USDT",
			"cTime": "1700000000000", "uTime": "1700000000000",
		}})
	}))
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, active, 1, "the pending order must be returned")
	assert.Equal(t, 41000.0, active[0].AverageExecutedPrice, "GetActiveOrders should read the average filled price")
	assert.Equal(t, 41000.0*0.5, active[0].Cost, "the cost should be the average filled price times the accumulated fill")
	assert.Equal(t, "USDT", active[0].CostAsset.String(), "the cost asset should be the pair quote")
	assert.Equal(t, 0.01, active[0].Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
	assert.Equal(t, "USDT", active[0].FeeAsset.String(), "the fee asset should come from the response")
}

// TestGetActiveSpreadOrdersReportExecutionValues guards the pending spread
// order mapping: it reports the average executed price and the cost of the
// fills like the spread history does.
func TestGetActiveSpreadOrdersReportExecutionValues(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-pending" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{{
			"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderLimit, "side": "buy", "state": "live",
			"sz": "1", "accFillSz": "0.3", "px": "100", "avgPx": "90", "cTime": "1700000000000",
		}})
	}))
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, active, 1, "the pending spread order must be returned")
	assert.Equal(t, 90.0, active[0].AverageExecutedPrice, "GetActiveOrders should read the average filled price")
	assert.InDelta(t, 27.0, active[0].Cost, 1e-9, "the cost should be the average filled price times the accumulated fill")
}

// TestGetFuturesPositionOrdersFeeSign guards the fee sign on the futures
// position orders, which read the same order history rows as GetOrderHistory:
// a fee OKX charged reads as a positive cost and a rebate as a negative one.
func TestGetFuturesPositionOrdersFeeSign(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
	row := func(id, fee string) map[string]string {
		return map[string]string{
			"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
			"sz": "100", "accFillSz": "100", "avgPx": "60000", "px": "60000", "fee": fee, "feeCcy": "USDT", "cTime": created, "uTime": created,
		}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/public/instruments":
			writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
		case "/trade/orders-history":
			writeOKXData(t, w, []map[string]string{row("CHARGED", "-3"), row("REBATED", "0.5")})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: time.Now().Add(-time.Hour), EndDate: time.Now(),
	})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
	fees := make(map[string]float64, len(positions[0].Orders))
	for i := range positions[0].Orders {
		fees[positions[0].Orders[i].OrderID] = positions[0].Orders[i].Fee
	}
	assert.Equal(t, map[string]float64{"CHARGED": 3, "REBATED": -0.5}, fees, "a fee OKX charged should read as a positive cost and a rebate as a negative one, as GetOrderHistory reports them")
}
