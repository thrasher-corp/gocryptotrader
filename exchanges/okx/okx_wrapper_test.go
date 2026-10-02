package okx

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
		{order.Market, order.ImmediateOrCancel, orderIOC},
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
				if r.URL.Path != "/sprd/orders-history" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				gotOrdType = r.URL.Query().Get("ordType")
				writeOKXData(t, w, []map[string]string{{
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderPostOnly,
					"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
				}})
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
		})
	}
}

// TestSpreadOrderHistoryAnyType guards the spread GetOrderHistory path: an
// AnyType request reaches OKX without an ordType filter instead of being
// rejected before sending, and its rows keep orderTypeFromString's typing.
func TestSpreadOrderHistoryAnyType(t *testing.T) {
	t.Parallel()
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sprd/orders-history" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		assert.Empty(t, r.URL.Query().Get("ordType"), "an AnyType request should not send an ordType filter")
		writeOKXData(t, w, []map[string]string{{"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderPostOnly, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100"}})
	}))
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetOrderHistory must not reject order.AnyType")
	require.Len(t, history, 1, "GetOrderHistory must return the spread order")
	assert.Equal(t, order.Limit, history[0].Type, "GetOrderHistory should read the order type")
	assert.Equal(t, order.PostOnly, history[0].TimeInForce, "GetOrderHistory should preserve the time in force")
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

// TestGetOrderHistoryCrawlsAcrossEndTimeBoundary guards the order history
// crawl: OKX can return the boundary millisecond's orders again at the head of
// the next page, so the crawl must skip the overlap instead of stopping at the
// first row on the boundary millisecond, which would drop the rest of that
// millisecond's orders and everything older.
func TestGetOrderHistoryCrawlsAcrossEndTimeBoundary(t *testing.T) {
	t.Parallel()
	base := time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond)
	endTime := base.Add(128 * time.Millisecond)
	// ORD-129 shares ORD-128's millisecond: several orders can share one
	// boundary millisecond.
	cTime := func(i int) time.Time {
		if i == 129 {
			return base.Add(128 * time.Millisecond)
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
		}
	}
	// The mock reads the end timestamp inclusively: the first page holds the
	// 100 newest orders at or before the end timestamp, and the second page
	// repeats the first page's last order before the older ones.
	page1 := make([]map[string]string, 0, orderListPageSize)
	for i := 129; i >= 30; i-- {
		page1 = append(page1, row(i))
	}
	page2 := make([]map[string]string, 0, 31)
	for i := 30; i >= 0; i-- {
		page2 = append(page2, row(i))
	}
	pages := map[string][]map[string]string{
		strconv.FormatInt(endTime.UnixMilli(), 10):                       page1,
		strconv.FormatInt(base.Add(30*time.Millisecond).UnixMilli(), 10): page2,
	}
	var mu sync.Mutex
	var endCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		end := r.URL.Query().Get("end")
		mu.Lock()
		repeated := slices.Contains(endCursors, end)
		endCursors = append(endCursors, end)
		page := pages[end]
		mu.Unlock()
		if repeated {
			// A repeat means the end timestamp cursor never advanced.
			t.Errorf("end %q requested again", end)
			http.Error(w, "repeated end", http.StatusBadRequest)
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
	cursors := slices.Clone(endCursors)
	mu.Unlock()
	assert.Equal(t, []string{
		strconv.FormatInt(endTime.UnixMilli(), 10),
		strconv.FormatInt(base.Add(30*time.Millisecond).UnixMilli(), 10),
	}, cursors, "the second page should be requested with the first page's last creation time")
	require.Len(t, history, 130, "a paginated crawl must return every order at or before the end timestamp")
	ids := make(map[string]struct{}, len(history))
	for x := range history {
		ids[history[x].OrderID] = struct{}{}
	}
	assert.Len(t, ids, len(history), "a paginated crawl should not duplicate orders")
	assert.Contains(t, ids, "ORD-129", "orders sharing the boundary millisecond should all be returned")
	assert.Contains(t, ids, "ORD-000", "orders past the first page should be returned")
}

// TestGetOrderHistoryStopsWhenAFullPageIsSeen guards the no-progress stop: a
// full page whose orders all repeat must not loop forever on the same end
// timestamp.
func TestGetOrderHistoryStopsWhenAFullPageIsSeen(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-24*time.Hour).Truncate(time.Millisecond).UnixMilli(), 10)
	page := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		// Every order shares the one creation millisecond, so a full page
		// cannot advance the end timestamp cursor.
		page = append(page, map[string]string{
			"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i),
			"cTime": created, "uTime": created, "state": "filled",
			"ordType": orderLimit, "side": "buy", "sz": "1",
			"px": "42000", "accFillSz": "1", "avgPx": "42000",
		})
	}
	var mu sync.Mutex
	var endCursors []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trade/orders-history-archive" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		end := r.URL.Query().Get("end")
		mu.Lock()
		repeated := slices.Contains(endCursors, end)
		endCursors = append(endCursors, end)
		mu.Unlock()
		if repeated {
			// A repeat means the end timestamp cursor never advanced.
			t.Errorf("end %q requested again", end)
			http.Error(w, "repeated end", http.StatusBadRequest)
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
	assert.Len(t, endCursors, 2, "the crawl should stop without a third request")
	assert.Len(t, history, orderListPageSize, "the first page's orders should be returned once")
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
