package okx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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

// mockRequestLimit caps the requests one mock exchange answers. A pager whose
// stop regresses asks for the same page forever, so past the cap the mock fails
// the request and the test fails instead of hanging until go test's timeout.
// The busiest test sends 29.
const mockRequestLimit = 100

// limitMockRequests serves the first mockRequestLimit requests with h and fails
// the rest.
func limitMockRequests(t *testing.T, h http.Handler) http.Handler {
	t.Helper()
	var served atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if served.Add(1) > mockRequestLimit {
			t.Errorf("mock exchange received more than %d requests, the last for %s", mockRequestLimit, r.URL)
			http.Error(w, "mock request limit exceeded", http.StatusBadRequest)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// newMockExchange returns an exchange whose REST endpoints all point at a test
// server running h.
func newMockExchange(t *testing.T, h http.Handler) *Exchange {
	t.Helper()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.DisableRateLimiter(), "the limiter must disable: the mock answers at once, so the rate limiters every test in the package shares would only hold its requests back behind other tests")
	srv := httptest.NewTestServer(t, limitMockRequests(t, h))
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

func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

// TestGetOpenInterestSingleOptionsKey guards the options branch of the
// single-key open interest query: OKX rejects a family that doesn't match the
// instrument with 51002, so only the instrument's own family is queried, with
// the instrument pinned by ID.
func TestGetOpenInterestSingleOptionsKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		instID    string
		tickBands string
		family    string
		err       error
	}{
		{
			name:      "plain family option, _UM family listed first",
			instID:    "BTC-USD-260928-79000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]}]`,
			family:    "BTC-USD",
		},
		{
			name:      "_UM option, plain family listed first",
			instID:    "BTC-USD_UM-260928-79000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
			family:    "BTC-USD_UM",
		},
		{
			name:      "family missing from the tick bands",
			instID:    "ETH-USD-260928-2000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
			err:       errInstrumentFamilyRequired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

			var mu sync.Mutex
			var paths []string
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/public/instrument-tick-bands":
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.tickBands + `}`))
				case "/public/open-interest":
					if r.URL.Query().Get("instFamily") != tc.family {
						_, _ = w.Write([]byte(`{"code":"51002","msg":"Instrument ID doesn't match underlying index.","data":[]}`))
						return
					}
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"` + tc.instID + `","instType":"OPTION","oi":"100","oiCcy":"1","oiUsd":"10","ts":"1597026383085"}]}`))
				default:
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
				}
			}))
			b := e.GetBase()
			b.SkipAuthCheck = true
			require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
			for k := range b.API.Endpoints.GetURLMap() {
				require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
			}

			optionInstID, err := currency.NewPairFromString(tc.instID)
			require.NoError(t, err, "option instrument ID must parse")
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.Options, optionInstID)

			resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
				Base:  optionInstID.Base.Item,
				Quote: optionInstID.Quote.Item,
				Asset: asset.Options,
			})
			mu.Lock()
			defer mu.Unlock()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "GetOpenInterest must reject an option whose family isn't listed")
				assert.Equal(t, []string{"/public/instrument-tick-bands?instType=OPTION"}, paths, "no open interest query should be made without the family")
				return
			}
			require.NoError(t, err, "GetOpenInterest with an Options key must not error")
			require.Len(t, resp, 1, "GetOpenInterest with an Options key must return the requested instrument entry")
			assert.Equal(t, 100.0, resp[0].OpenInterest, "the entry should carry the mocked open interest")
			assert.Truef(t, resp[0].Key.Pair().Equal(optionInstID), "the entry should resolve to the requested pair, got %v", resp[0].Key.Pair())
			assert.Equal(t, []string{
				"/public/instrument-tick-bands?instType=OPTION",
				"/public/open-interest?instFamily=" + tc.family + "&instId=" + tc.instID + "&instType=OPTION",
			}, paths, "only the instrument's own family should be queried, with the instrument pinned by ID")
		})
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
	// a relisted instrument's websocket orders fail rather than go out with its
	// replaced code.
	fresh.cacheInstrumentIDCodes([]Instrument{
		{InstrumentID: mainPair},
		{InstrumentID: solUSDT, InstrumentIDCode: 22222},
	})
	assert.Zero(t, fresh.getInstrumentIDCode("BTC-USDT"), "a push without a code should drop the cached instrument ID code")
	assert.EqualValues(t, 22222, fresh.getInstrumentIDCode("SOL-USDT"), "a push with a code should still upsert it")

	assert.Zero(t, fresh.getInstrumentIDCode("DOGE-USDT"), "an uncached instrument ID should resolve a zero code")
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
		{order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
		{order.MarketMakerProtection, order.GoodTillCancel | order.PostOnly, orderMarketMakerProtectionAndPostOnly},
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

// TestMarketListingsKeepToMarketOrders guards a market listing with a time in
// force: the ioc and fok orders such a request places as read back as Limit,
// so the listings must ask for market orders instead.
func TestMarketListingsKeepToMarketOrders(t *testing.T) {
	t.Parallel()
	row := func(id, ordType, state string) map[string]string {
		return map[string]string{"instId": mainPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": state, "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": "1700000000000"}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := "filled"
		switch r.URL.Path {
		case "/trade/orders-pending":
			state = "live"
		case "/trade/orders-history-archive", "/trade/orders-history":
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		ordTypes := strings.Split(r.URL.Query().Get("ordType"), ",")
		rows := make([]map[string]string, 0, 4)
		for _, o := range []map[string]string{row("MARKET-1", orderMarket, state), row("LIMIT-1", orderLimit, state), row("IOC-1", orderIOC, state), row("FOK-1", orderFOK, state)} {
			if ordTypes[0] == "" || slices.Contains(ordTypes, o["ordType"]) {
				rows = append(rows, o)
			}
		}
		writeOKXData(t, w, rows)
	}))
	for _, tif := range []order.TimeInForce{order.ImmediateOrCancel, order.FillOrKill} {
		req := &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Market, TimeInForce: tif, Side: order.AnySide}
		active, err := e.GetActiveOrders(t.Context(), req)
		require.NoErrorf(t, err, "GetActiveOrders must not error for a market order with %s", tif)
		require.Lenf(t, active, 1, "GetActiveOrders must return the market order for a market order with %s", tif)
		assert.Equalf(t, "MARKET-1", active[0].OrderID, "GetActiveOrders should return the market order for a market order with %s", tif)
		history, err := e.GetOrderHistory(t.Context(), req)
		require.NoErrorf(t, err, "GetOrderHistory must not error for a market order with %s", tif)
		require.Lenf(t, history, 1, "GetOrderHistory must return the market order for a market order with %s", tif)
		assert.Equalf(t, "MARKET-1", history[0].OrderID, "GetOrderHistory should return the market order for a market order with %s", tif)
	}
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
		{order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
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

// TestGetSpreadOrderHistory guards the completed spread order history
// mapping and its pagination: the documented 3 month spread order
// history window means a start time older than the 21 day listing's
// window must also crawl sprd/orders-history-archive, and the overlap
// between the two listings must not duplicate orders. OKX confirmed the
// listing's begin filter reaches its full 21 day window, so windows
// inside it need no archive crawl.
func TestGetSpreadOrderHistory(t *testing.T) {
	t.Parallel()

	// Guards the spread order
	// history pagination: OKX caps the sprd/orders-history response at
	// orderListPageSize records and pages the remainder with the same endId
	// earlier-than order ID cursor as the pending spread order listing.
	t.Run("paginates with the endId cursor", func(t *testing.T) {
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
	})

	// Guards the
	// FromOrderID request field for spread order history: the endId cursor
	// returns records earlier than the requested order ID like the standard
	// archive's after cursor, so the first request must carry FromOrderID as the
	// endId cursor, and beginId, which returns records newer than an order ID,
	// must never be sent.
	t.Run("from order ID seeds the endId cursor", func(t *testing.T) {
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
			StartTime:   time.Now().Add(-24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds the spread crawl, with a recent start time keeping it inside the 21 day listing")

		mu.Lock()
		cursors := make([]string, len(endIDCursors))
		copy(cursors, endIDCursors)
		mu.Unlock()
		assert.Equal(t, []string{"SEED-1", "SPD-099"}, cursors, "the crawl should start from FromOrderID and advance with each page's last order ID")
		assert.Len(t, history, orderListPageSize+50, "a crawl seeded by FromOrderID should return every earlier spread order")
	})

	// Guards the no-progress
	// stop: a full page whose last order ID repeats the requested endId cursor
	// must not loop forever on the same page.
	t.Run("stops when a full page is seen", func(t *testing.T) {
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
			StartTime: time.Now().Add(-24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must stop when a full page holds no new spread order, with a recent start time keeping the crawl inside the 21 day listing")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"", "SPD-099"}, endIDCursors, "the crawl should stop when the endId cursor stops advancing")
		assert.Len(t, history, orderListPageSize, "the first page's orders should be returned once")
	})

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

	t.Run("archive pages with its endId cursor to a single remaining order", func(t *testing.T) {
		t.Parallel()
		archived := make([]map[string]string, 0, orderListPageSize+1)
		want := make([]string, 0, orderListPageSize+2)
		want = append(want, "SPD-NEW")
		for x := range orderListPageSize + 1 {
			id := fmt.Sprintf("SPD-ARC-%03d", x)
			archived = append(archived, map[string]string{
				"sprdId": "BTC-USDT_BTC-USDT", "ordId": id, "ordType": orderLimit,
				"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
				"cTime": strconv.FormatInt(time.Now().Add(-kline.OneMonth.Duration()).UnixMilli()-int64(x), 10),
			})
			want = append(want, id)
		}
		var mu sync.Mutex
		var archiveEndIDs []string
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sprd/orders-history":
				writeOKXData(t, w, []map[string]string{recent})
			case "/sprd/orders-history-archive":
				endID := r.URL.Query().Get("endId")
				mu.Lock()
				archiveEndIDs = append(archiveEndIDs, endID)
				mu.Unlock()
				switch endID {
				case "":
					writeOKXData(t, w, archived[:orderListPageSize])
				case archived[orderListPageSize-1]["ordId"]:
					writeOKXData(t, w, archived[orderListPageSize:])
				default:
					t.Errorf("unexpected archive endId %q", endID)
					writeOKXData(t, w, []map[string]string{})
				}
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		resp, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
			StartTime: time.Now().Add(-2 * kline.OneMonth.Duration()),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the archive spans two pages")
		ids := make([]string, 0, len(resp))
		for x := range resp {
			ids = append(ids, resp[x].OrderID)
		}
		assert.ElementsMatch(t, want, ids, "every archived order should be returned once, the single order on the last page included")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"", "SPD-ARC-099"}, archiveEndIDs, "the archive crawl should advance its endId cursor to each full page's last order ID")
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
			StartTime: time.Now().Add(-10 * 24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the ten-day-old start time sits inside the 21 day listing")
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
			StartTime: time.Now().Add(-40 * 24 * time.Hour),
			EndTime:   time.Now().Add(-30 * 24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the window sits entirely older than the 21 day listing and only the archive covers it")
		require.Len(t, resp, 1, "the archive must be the only source for a window older than the listing")
		assert.Equal(t, "SPD-OLD", resp[0].OrderID)
	})

	// Guards the
	// FromOrderID request field across both spread listings: when the requested
	// window reaches the archive, both crawls must start from FromOrderID as
	// their endId cursor, and the archive's overlapping row must not duplicate
	// the listing's.
	t.Run("from order ID seeds the archive crawl", func(t *testing.T) {
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
	})

	// Guards the archive
	// crawl's no-progress stop: a full archive page whose last order ID repeats
	// the requested endId cursor must not loop forever.
	t.Run("stops when the archive page stalls", func(t *testing.T) {
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
	})

	// Guards the spread GetOrderHistory ordType:
	// the documented single values are sent, and a plain limit query no longer
	// drops post_only and ioc history behind a limit-only filter.
	t.Run("narrows by the requested order type", func(t *testing.T) {
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
	})

	// Guards the spread GetOrderHistory path: an
	// AnyType request reaches OKX without an ordType filter instead of being
	// rejected before sending, and its rows keep orderTypeFromString's typing.
	// Guards an AnyType spread history request and
	// the spread order row mapping: the ordType filter is dropped, and every
	// order.Detail field comes from the listing row.
	t.Run("any type queries every listing", func(t *testing.T) {
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
		assert.Zero(t, first.Cost, "GetOrderHistory should leave a spread order's cost unset")
		assert.Equal(t, 1.0, first.Amount, "GetOrderHistory should read the order size")
		assert.Equal(t, 0.3, first.ExecutedAmount, "GetOrderHistory should read the accumulated fill size")
		assert.InDelta(t, 0.7, first.RemainingAmount, 1e-9, "an unfilled spread order should keep its remaining amount")
		assert.Equal(t, asset.Spread, first.AssetType, "GetOrderHistory should tag the spread asset type")
		assert.Equal(t, "BTC", first.Pair.Base.String(), "the pair base should be the spread ID head under the dash delimiter")
		assert.Equal(t, "USDT_BTC-USDT", first.Pair.Quote.String(), "the pair quote should keep the remainder of the spread ID")
		second := rows["2"]
		assert.Equal(t, order.Filled, second.Status, "GetOrderHistory should read the filled state")
		assert.Equal(t, 0.0, second.RemainingAmount, "a filled spread order should have nothing remaining")
		assert.Equal(t, 2.0, second.Amount, "GetOrderHistory should read the filled order size")
		assert.Equal(t, 2.0, second.ExecutedAmount, "GetOrderHistory should read the filled order's accumulated fill")
		assert.Equal(t, 88.0, second.AverageExecutedPrice, "GetOrderHistory should read the filled order's average price")
		assert.Zero(t, second.Cost, "GetOrderHistory should leave a filled spread order's cost unset")
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
		{order.Limit, order.GoodTillCancel | order.PostOnly, []string{"POST-1"}},
		{order.MarketMakerProtection, order.GoodTillCancel | order.PostOnly, []string{"MMP-PO-1"}},
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

// TestWebsocketSubmitOrder covers the explicit websocket submit path: the
// orders it refuses before anything transmits, the instrument code it needs
// cached first, the time in force and position mode each order sends.
func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("rejects an order before it transmits", func(t *testing.T) {
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
			Pair:      spreadPair,
			Side:      order.Buy,
			Type:      order.OptimalLimit,
			Amount:    1,
			AssetType: asset.Spread,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread order should not transmit on the private connection whatever its type")

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
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]Instrument)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		require.Error(t, err, "WebsocketSubmitOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})
	// Sends the ordType the REST placement path documents over the private
	// order frames.
	t.Run("orders send their documented ordType", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			orderType order.Type
			tif       order.TimeInForce
			exp       string
		}{
			{"plain limit", asset.Spot, order.Limit, order.UnknownTIF, orderLimit},
			{"post only", asset.Spot, order.Limit, order.PostOnly, orderPostOnly},
			{"good till cancel post only", asset.Spot, order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
			{"fill or kill", asset.Spot, order.Limit, order.FillOrKill, orderFOK},
			{"immediate or cancel", asset.Spot, order.Limit, order.ImmediateOrCancel, orderIOC},
			// OKX takes MMP orders for options only.
			{"market maker protection", asset.Options, order.MarketMakerProtection, order.UnknownTIF, orderMarketMakerProtection},
			{"market maker protection post only", asset.Options, order.MarketMakerProtection, order.PostOnly, orderMarketMakerProtectionAndPostOnly},
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
					AssetType:   tc.assetType,
					Side:        order.Buy,
					Type:        tc.orderType,
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
				assert.Equal(t, "1", sent[0]["px"], "the websocket order should send its price")
			})
		}
	})
	// Guards the contract
	// fields on the websocket transport: the private order frames must carry
	// OKX's required side, the position side and the reduce-only flag.
	t.Run("a contract order sends its side and reduce-only flag", func(t *testing.T) {
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
	})
	// Places a perpetual swap with the side the account's confirmed position
	// mode requires, over the private order frames.
	t.Run("a perpetual swap follows the account position mode", func(t *testing.T) {
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
	})
}

// TestWebsocketModifyOrder covers the amendments the explicit websocket
// path refuses before anything transmits.
func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("rejects an amend before it transmits", func(t *testing.T) {
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
}

// TestWebsocketCancelOrder covers the cancellations the explicit websocket
// path refuses, and the instrument code it needs cached first.
func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("rejects a cancel before it transmits", func(t *testing.T) {
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
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]Instrument)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		})
		require.Error(t, err, "WebsocketCancelOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})
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

// TestBatchCancelEndpointsRefuseOversizedLists guards the documented
// per-request maximum of the batch cancel endpoints: the caller supplies the
// list, so more than 20 orders, or 10 algo orders, is refused before any
// request transmits, over REST and the websocket alike.
func TestBatchCancelEndpointsRefuseOversizedLists(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	args := make([]CancelOrderRequestParam, 21)
	for i := range args {
		args[i] = CancelOrderRequestParam{InstrumentID: "BTC-USDT", OrderID: strconv.Itoa(i)}
	}
	_, err := e.CancelMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "CancelMultipleOrders should refuse more than 20 orders")
	_, err = e.WSCancelMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "WSCancelMultipleOrders should refuse more than 20 orders")
	algoArgs := make([]AlgoOrderCancelParams, maxCancelAlgosPerRequest+1)
	for i := range algoArgs {
		algoArgs[i] = AlgoOrderCancelParams{InstrumentID: "BTC-USDT", AlgoOrderID: strconv.Itoa(i)}
	}
	_, err = e.CancelAlgoOrder(t.Context(), algoArgs)
	assert.ErrorIs(t, err, errExceedLimit, "CancelAlgoOrder should refuse more than 10 algo orders")
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

// stopMockWebsocket stops the websocket manager MockWsInstance leaves
// running. Left alone, it redials the closed mock's port a minute later and
// logs in again, by when another test's server may hold that port. Registered
// after the server's cleanup, it runs first.
func stopMockWebsocket(t *testing.T, e *Exchange) {
	t.Helper()
	t.Cleanup(func() {
		if e.Websocket.IsEnabled() {
			assert.NoError(t, e.Websocket.Disable(), "Disable should not error")
		}
		// Shutdown may return an error if the connection is already closed, but
		// we must ensure the connection monitor goroutine is stopped regardless.
		_ = e.Websocket.Shutdown()
		// Give the connection monitor a moment to exit before the test cleanup
		// closes the mock server, preventing a redial attempt.
		time.Sleep(100 * time.Millisecond)
	})
}

// newMockWebsocketExchange returns an exchange whose authenticated websocket is
// available to the wrapper, with every REST endpoint pointed at the same test
// server and mainPair's instrument code cached. rest answers REST requests, and
// wsReply returns the code and data of OKX's reply to each websocket operation.
func newMockWebsocketExchange(t *testing.T, rest http.HandlerFunc, wsReply func(op string, args json.RawMessage) (code string, data any)) *Exchange {
	t.Helper()
	limitedRest := limitMockRequests(t, rest)
	e := testexch.MockWsInstance[Exchange](t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			limitedRest.ServeHTTP(w, r)
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
	stopMockWebsocket(t, e)
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error for RestSpot")
	for k := range e.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, e.API.Endpoints.SetRunningURL(k, mockURL+"/"), "SetRunningURL must not error for %s", k)
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must be available to the wrapper")
	e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
	require.NoError(t, e.DisableRateLimiter(), "DisableRateLimiter must not error")
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

// TestWebsocketSubmitOrders guards the batch submission of orders over the
// websocket: each order's own reply, the batches refused before anything
// transmits and the instrument code a batch needs cached.
func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	// Guards how batch submission reports OKX's
	// rows: each is matched to its order by position, a partial success keeps its
	// error beside the placed orders, and a reply without one row per order is
	// rejected rather than misattributed.
	t.Run("reports the replies", func(t *testing.T) {
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
			{"success with a null row", "0", []map[string]string{nil, second}, []string{"", "2 SELL"}, common.ErrInvalidResponse},
			{"success with a row without an order ID", "0", []map[string]string{{"ordId": "", "sCode": "0", "sMsg": ""}, second}, []string{"", "2 SELL"}, common.ErrInvalidResponse},
			{"success without order IDs", "0", []map[string]string{nil, {"ordId": "", "sCode": "0", "sMsg": ""}}, []string{"", ""}, common.ErrInvalidResponse},
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
	})

	t.Run("rejects a batch before it transmits", func(t *testing.T) {
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
	t.Run("sends nothing when one order in the batch is invalid", func(t *testing.T) {
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
}

// TestWebsocketCancelBatchOrders guards the websocket batch cancellation of
// orders: each order's own outcome beside a partial success, the batches
// refused before anything transmits and the code they need cached.
func TestWebsocketCancelBatchOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	// Guards the websocket batch cancel
	// results: a partial success records each order's own outcome beside the error.
	t.Run("partial success", func(t *testing.T) {
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
			"1": order.PendingCancel.String(),
			"2": "Order cancellation failed as the order has been filled, canceled or does not exist",
		}, resp.Status, "each order should report its own outcome")
	})

	t.Run("rejects a cancel batch before it transmits", func(t *testing.T) {
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
	t.Run("sends nothing when one cancel in the batch is invalid", func(t *testing.T) {
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
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]Instrument)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}})
		require.Error(t, err, "WebsocketCancelBatchOrders must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})
}

// TestWebsocketCancelAllOrders guards the websocket cancellation of all
// pending orders: the batches it sends, the orders it skips and the requests
// it refuses before anything transmits.
func TestWebsocketCancelAllOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	// Guards websocket
	// cancel-all: pending orders are cancelled over the websocket in batches of 20
	// with their instrument codes, and an order whose instrument has no cached code
	// is reported in the error rather than stopping the rest.
	t.Run("skips uncached instruments", func(t *testing.T) {
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
	})

	t.Run("rejects a cancel-all before it transmits", func(t *testing.T) {
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
	stopMockWebsocket(t, e)
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
	// Registered after the mock server's cleanup, so it runs first and frees
	// a handler still waiting on release.
	t.Cleanup(releaseOnce)
	type result struct {
		mode string
		err  error
	}
	fetched := make(chan result, 1)
	go func() {
		mode, err := e.contractPositionMode(t.Context())
		fetched <- result{mode, err}
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
	case got := <-fetched:
		require.NoError(t, got.err, "the in-flight fetch must not error")
		assert.Equal(t, positionModeLongShort, got.mode, "the in-flight lookup should return the confirmed mode rather than its older fetch")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the in-flight fetch must return")
	}
	mode, err := e.contractPositionMode(t.Context())
	require.NoError(t, err, "contractPositionMode must not error")
	assert.Equal(t, positionModeLongShort, mode, "a confirmed switch should outlive an older fetch")
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

// TestContractPositionModeErrorsWhenAFetchOverlapsASwitch guards the
// position mode against a first-use fetch through one API key that returns
// after a switch confirmed through another key of the same account: the
// fetched mode may predate the switch, so it is neither used nor cached, and
// the next order fetches again.
func TestContractPositionModeErrorsWhenAFetchOverlapsASwitch(t *testing.T) {
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
		require.ErrorIs(t, got.err, errPositionModeSwitchedDuringFetch, "the overlapped lookup must error rather than report a mode that predates the switch")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the in-flight lookup must return")
	}
	mode, err := e.contractPositionMode(ctxFor("KEY-1"))
	require.NoError(t, err, "contractPositionMode must not error once the switch settled")
	assert.Equal(t, positionModeLongShort, mode, "the next lookup should fetch the switched mode")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, fetches, "the overlapped fetch should be neither cached nor reused, so the next lookup fetches again")
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

// TestGetFuturesPositionOrders covers the orders the position report reads
// from the history listings: the fee sign, the contract costs, the execution
// units and which listing the crawl reads.
func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	// Guards the fee sign on the futures
	// position orders, which read the same order history rows as GetOrderHistory:
	// a fee OKX charged reads as a positive cost and a rebate as a negative one.
	t.Run("reads a charged fee as positive and a rebate as negative", func(t *testing.T) {
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
	})
	// Guards the cost of contract
	// fills: a linear contract's cost is its settlement-currency notional, an
	// inverse contract's is its base-currency one, which divides by the price
	// rather than multiplying, and an unfilled order costs nothing rather than
	// not-a-number.
	t.Run("reports contract costs by settlement type", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		inverseSwapPair := currency.NewPairWithDelimiter("BTC", "USD-SWAP", "-")
		for _, tc := range []struct {
			name       string
			instrument map[string]string
			pair       currency.Pair
			state      string
			avgPx      string
			expType    futures.ContractSettlementType
			expCost    float64
			expAsset   currency.Code
		}{
			{
				name:       "linear",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"},
				pair:       perpetualSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Linear,
				expCost:    60000,
				expAsset:   currency.USDT,
			},
			{
				name:       "inverse",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
				pair:       inverseSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Inverse,
				expCost:    100.0 * 100.0 / 60000.0,
				expAsset:   currency.BTC,
			},
			{
				name:       "suspended inverse",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "suspend"},
				pair:       inverseSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Inverse,
				expCost:    100.0 * 100.0 / 60000.0,
				expAsset:   currency.BTC,
			},
			{
				name:       "inverse unfilled",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
				pair:       inverseSwapPair,
				state:      "live",
				avgPx:      "0",
				expType:    futures.Inverse,
				expCost:    0,
				expAsset:   currency.BTC,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{tc.instrument})
					case "/trade/orders-history":
						writeOKXData(t, w, []map[string]string{{
							"instType": instTypeSwap, "instId": tc.instrument["instId"], "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": tc.state,
							"sz": "100", "accFillSz": map[string]string{"filled": "100", "live": "0"}[tc.state], "avgPx": tc.avgPx, "px": "60000", "cTime": created, "uTime": created,
						}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{tc.pair}, StartDate: time.Now().Add(-time.Hour), EndDate: time.Now(),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				assert.Equal(t, tc.expType, positions[0].ContractSettlementType, "the position should carry the contract's settlement type")
				require.Len(t, positions[0].Orders, 1, "GetFuturesPositionOrders must return the order")
				got := positions[0].Orders[0]
				assert.Equal(t, tc.expCost, got.Cost, "a contract fill's cost should follow its settlement type")
				assert.Equal(t, tc.expAsset, got.CostAsset, "the cost should be reported in the contract's settlement currency")
				assert.Equal(t, 100.0, got.ContractAmount, "the order amount should stay in contracts")
				assert.False(t, math.IsNaN(got.Cost), "an unfilled order's cost should never be not-a-number")
			})
		}
	})
	// Guards the unit of the
	// execution sizes: the amount is scaled by the contract value, so its executed
	// and remaining parts are scaled with it, while ContractAmount keeps the
	// contract count: 100 contracts of 0.01 BTC with 40 filled read as an amount
	// of 1 with 0.4 executed and 0.6 remaining.
	t.Run("scales execution amounts by the contract value", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{{
					"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "live",
					"sz": "100", "accFillSz": "40", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
				}})
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
		require.Len(t, positions[0].Orders, 1, "GetFuturesPositionOrders must return the order")
		got := positions[0].Orders[0]
		assert.Equal(t, 1.0, got.Amount, "the amount should be the contract count scaled by the contract value")
		assert.Equal(t, 0.4, got.ExecutedAmount, "the executed amount should share the amount's unit")
		assert.Equal(t, 0.6, got.RemainingAmount, "the remaining amount should share the amount's unit")
		assert.Equal(t, 100.0, got.ContractAmount, "the contract amount should keep the contract count")
	})
	// Guards the per instrument crawl: the
	// history listings page at 100 rows, so the window's orders must be followed
	// across pages with the after order ID cursor.
	t.Run("pages the history listing with the after cursor", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		row := func(id string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		first := make([]map[string]string, 0, orderListPageSize)
		for i := range orderListPageSize {
			first = append(first, row(fmt.Sprintf("ORD-%03d", i)))
		}
		last := make([]map[string]string, 0, 50)
		for i := range 50 {
			last = append(last, row(fmt.Sprintf("ORD-%03d", orderListPageSize+i)))
		}
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				after := r.URL.Query().Get("after")
				switch after {
				case "":
					writeOKXData(t, w, first)
				case "ORD-099":
					writeOKXData(t, w, last)
				default:
					t.Errorf("unexpected after cursor %q", after)
					http.Error(w, "unexpected cursor", http.StatusBadRequest)
				}
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
		require.Len(t, positions[0].Orders, 150, "GetFuturesPositionOrders must crawl every page of the instrument's window")
		assert.Equal(t, "ORD-000", positions[0].Orders[0].OrderID, "the first page's orders should come first")
		assert.Equal(t, "ORD-149", positions[0].Orders[149].OrderID, "the second page's orders should follow")
	})
	// Guards which history listing the
	// crawl reads: the 7 day listing for a window starting within 7 days, the 3
	// month archive otherwise, and one listing for the whole crawl, since the
	// archive leaves out cancelled orders without fills that the 7 day listing
	// keeps.
	t.Run("reads one history listing for the window", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		row := func(id string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		first := make([]map[string]string, 0, orderListPageSize)
		for i := range orderListPageSize {
			first = append(first, row(fmt.Sprintf("ORD-%03d", i)))
		}
		for _, tc := range []struct {
			name    string
			age     time.Duration
			listing string
		}{
			{name: "window within 7 days", age: 2 * 24 * time.Hour, listing: "/trade/orders-history"},
			{name: "window older than 7 days", age: 14 * 24 * time.Hour, listing: "/trade/orders-history-archive"},
			// StartDate turns 7 days old while the first page is served, so
			// either listing may be picked, but only one.
			{name: "window turning 7 days old during the crawl", age: 7*24*time.Hour - 100*time.Millisecond},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				listings := make(map[string]int)
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
					case "/trade/orders-history", "/trade/orders-history-archive":
						mu.Lock()
						listings[r.URL.Path]++
						mu.Unlock()
						if r.URL.Query().Get("after") == "" {
							// The first page outlasts the 100ms the last case's
							// StartDate has left before it is 7 days old.
							time.Sleep(200 * time.Millisecond)
							writeOKXData(t, w, first)
							return
						}
						writeOKXData(t, w, []map[string]string{row("ORD-100")})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair},
					StartDate: time.Now().Add(-tc.age), EndDate: time.Now(),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				assert.Len(t, positions[0].Orders, orderListPageSize+1, "GetFuturesPositionOrders should return both pages")
				mu.Lock()
				defer mu.Unlock()
				if tc.listing != "" {
					assert.Equal(t, map[string]int{tc.listing: 2}, listings, "both pages should come from the window's history listing")
					return
				}
				assert.Len(t, listings, 1, "every page of the crawl should come from one history listing")
			})
		}
	})
}

// TestWebsocketNullRowsDoNotPanic guards the single-row websocket operations
// against a success reply with a null row: the reply leaves the operation's
// outcome unknown, so each must report it rather than panic on the missing
// row.
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
				return e.WSMassCancelOrders(t.Context(), []CancelMassReqParam{
					{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"},
				})
			},
		},
		{
			name: "cancel all spread orders",
			call: func(e *Exchange) error {
				return e.WSCancelAllSpreadOrders(t.Context(), spreadPair.String())
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockWebsocketExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// TestGetFuturesPositionSummary covers the sizes a short position reports:
// net mode signs them itself, long/short mode carries the direction in posSide.
func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	// Guards the net mode
	// short: OKX reports it with a negative position size, and the summary types
	// carry short sizes signed, so only a zero size marks a closed position.
	t.Run("signs a net mode short", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/account/positions":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "-1", "mgnMode": "cross", "ccy": "USDT"}})
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"details": []map[string]string{{"ccy": "USDT", "eq": "100", "availEq": "100"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
			Asset: asset.PerpetualSwap, Pair: perpetualSwapPair,
		})
		require.NoError(t, err, "a net mode short must summarise rather than report no positions")
		assert.Equal(t, decimal.MustFromFloat(-0.01), summary.CurrentSize, "a short of one 0.01 contract should carry its negative size")
		assert.Equal(t, decimal.MustFromFloat(-1), summary.ContractSize, "the contract size should keep the signed position")
	})
	// Guards the
	// long/short mode short: OKX reports both sides' sizes positive and carries
	// the direction in posSide, so a posSide short must still read negative.
	t.Run("signs a long/short mode short", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/account/positions":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "1", "posSide": positionSideShort, "mgnMode": "cross", "ccy": "USDT"}})
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"details": []map[string]string{{"ccy": "USDT", "eq": "100", "availEq": "100"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeLongShort}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
			Asset: asset.PerpetualSwap, Pair: perpetualSwapPair,
		})
		require.NoError(t, err, "a long/short mode short must summarise rather than report no positions")
		assert.Equal(t, decimal.MustFromFloat(-0.01), summary.CurrentSize, "a posSide short of one 0.01 contract should read as a negative size")
		assert.Equal(t, decimal.MustFromFloat(-1), summary.ContractSize, "the contract size should carry the flipped short sign")
	})
}

// TestSetDefaults also pins the websocket base used to construct the public and private defaults.
func TestSetDefaults(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	t.Run("endpoint map", func(t *testing.T) {
		t.Parallel()
		defaults := ex.API.Endpoints.GetURLMap()
		require.Len(t, defaults, 4, "endpoint defaults must match the declared API scope")
		assert.Equal(t, "https://www.okx.com/api/v5/", defaults[exchange.RestSpot.String()], "REST default should target the v5 API")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/public", defaults[exchange.WebsocketSpot.String()], "public websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/private", defaults[exchange.WebsocketPrivate.String()], "private websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/business", defaults[exchange.WebsocketSpotSupplementary.String()], "business websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/", websocketURL, "websocket base should use port 443")
		for endpoint, url := range defaults {
			assert.NotContainsf(t, url, ":8443", "%s default should not use the retired port 8443", endpoint)
		}
	})
	for ep, want := range map[exchange.URL]string{
		exchange.WebsocketSpot:              "wss://ws.okx.com/ws/v5/public",
		exchange.WebsocketPrivate:           "wss://ws.okx.com/ws/v5/private",
		exchange.WebsocketSpotSupplementary: "wss://ws.okx.com/ws/v5/business",
	} {
		t.Run(ep.String(), func(t *testing.T) {
			t.Parallel()
			got, err := ex.API.Endpoints.GetURL(ep)
			require.NoErrorf(t, err, "GetURL must return the default %s endpoint", ep)
			assert.Equalf(t, want, got, "default %s endpoint should use port 443", ep)
		})
	}
}
