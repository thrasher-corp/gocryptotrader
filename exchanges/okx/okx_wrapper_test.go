package okx

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

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

	// A push omitting instIdCode decodes to zero and must not clobber the
	// cached code for an instrument that already has one.
	fresh.cacheInstrumentIDCodes([]Instrument{
		{InstrumentID: mainPair},
		{InstrumentID: solUSDT, InstrumentIDCode: 22222},
	})
	assert.EqualValues(t, 12345, fresh.getInstrumentIDCode("BTC-USDT"), "a push without a code should keep the cached instrument ID code")
	assert.EqualValues(t, 22222, fresh.getInstrumentIDCode("SOL-USDT"), "a push with a code should still upsert it")

	assert.Zero(t, fresh.getInstrumentIDCode("DOGE-USDT"), "an uncached instrument ID should resolve a zero code")
}

func TestCancelAllOrdersMatchesOnlyRequestedOrders(t *testing.T) {
	// Subtests share the mock server's pending order queue, so they run
	// sequentially.
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var pending []map[string]string
	// pages, when set, serves the pending order list keyed by the after
	// pagination parameter, with "" naming the first page.
	var pages map[string][]map[string]string
	var cancelled []CancelOrderRequestParam
	var pendingQuery string

	writeOKXData := func(w http.ResponseWriter, data any) {
		encoded, err := json.Marshal(map[string]any{"code": "0", "msg": "", "data": data})
		if err != nil {
			t.Errorf("marshalling mock response should not error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			after := r.URL.Query().Get("after")
			mu.Lock()
			defer mu.Unlock()
			pendingQuery = r.URL.RawQuery
			if pages != nil {
				writeOKXData(w, pages[after])
				return
			}
			writeOKXData(w, pending)
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
			writeOKXData(w, respData)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	b := e.GetBase()
	b.SkipAuthCheck = true
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

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
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var amendBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(srv.Close)

	b := e.GetBase()
	b.SkipAuthCheck = true
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

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
