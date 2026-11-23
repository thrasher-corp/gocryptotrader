package okx

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpreadOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpreadOrder must reject a nil request")

	_, err = e.PlaceSpreadOrder(contextGenerate(), &PlaceSpreadOrderRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID, "PlaceSpreadOrder must reject an empty request")

	arg := &PlaceSpreadOrderRequest{Tag: "tag-here"}
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID, "PlaceSpreadOrder must reject a request without a spread ID")

	arg.SpreadID = spreadPair.String()
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceSpreadOrder must reject a request without an order type")

	arg.OrderType = "limit"
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceSpreadOrder must reject a request without a size")

	arg.Size = 1
	arg.Price = 12345
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceSpreadOrder must reject a request without a side")

	t.Run("guards the price", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_, err := ex.PlaceSpreadOrder(t.Context(), &PlaceSpreadOrderRequest{
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
		_, err = ex.PlaceSpreadOrder(t.Context(), &PlaceSpreadOrderRequest{
			SpreadID:  spreadPair.String(),
			Side:      "buy",
			OrderType: orderLimit,
			Size:      1,
		})
		require.NoError(t, err, "a spread order at a zero price must place")
		mu.Lock()
		sent = nil
		require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
		mu.Unlock()
		assert.Equal(t, "0", sent["px"], "a zero spread price should be sent rather than left out")
		_, err = ex.PlaceSpreadOrder(t.Context(), &PlaceSpreadOrderRequest{
			SpreadID:  spreadPair.String(),
			Side:      "buy",
			OrderType: "MARKET",
			Size:      1,
		})
		require.NoError(t, err, "a market spread order given in upper case must place without a price")
		mu.Lock()
		sent = nil
		require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
		mu.Unlock()
		assert.Equal(t, orderMarket, sent["ordType"], "the spread order type should be sent in lower case")
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceSpreadOrderRequest
			body string
		}{
			{
				name: "every optional parameter",
				arg:  &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), ClientOrderID: "b15", Tag: "tag-here", Side: "BUY", OrderType: "Post_Only", Size: 0.5, Price: -41.2},
				body: `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","clOrdId":"b15","side":"buy","ordType":"post_only","sz":"0.5","px":"-41.2","tag":"tag-here"}`,
			},
			{
				name: "required parameters only",
				arg:  &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), Side: order.Sell.Lower(), OrderType: orderMarket, Size: 2},
				body: `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"sell","ordType":"market","sz":"2"}`,
			},
			{
				name: "limit order at a zero price",
				arg:  &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), Side: order.Buy.Lower(), OrderType: orderLimit, Size: 2},
				body: `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"buy","ordType":"limit","sz":"2","px":"0"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/order", tc.body, []map[string]string{{"ordId": "312269865356374016", "sCode": "0"}})
				_, err := ex.PlaceSpreadOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceSpreadOrder must not error")
			})
		}
	})

	t.Run("rejected order reports its row", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the rejection is a recorded reply")
		}
		_, err := e.PlaceSpreadOrder(contextGenerate(), &PlaceSpreadOrderRequest{
			SpreadID:      spreadPair.String(),
			ClientOrderID: "b15",
			Tag:           "tag-here",
			Side:          order.Sell.Lower(),
			OrderType:     orderLimit,
			Size:          1000,
			Price:         -30.5,
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "PlaceSpreadOrder must fail a rejected order")
		assert.ErrorContains(t, err, "51008", "PlaceSpreadOrder should report the rejected row's status code")
		assert.ErrorContains(t, err, "Insufficient USDT balance", "PlaceSpreadOrder should report the rejected row's status message")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceSpreadOrder(contextGenerate(), &PlaceSpreadOrderRequest{
		SpreadID:      spreadPair.String(),
		ClientOrderID: "12354123523",
		Side:          order.Buy.Lower(),
		OrderType:     orderLimit,
		Size:          1,
		Price:         12345,
	})
	require.NoError(t, err, "PlaceSpreadOrder must not error")
	if mockTests {
		exp := &SpreadOrderResponse{ClientOrderID: "12354123523", OrderID: "312269865356374016"}
		assert.Equal(t, exp, result, "PlaceSpreadOrder should decode every field")
	} else {
		require.NotNil(t, result, "PlaceSpreadOrder must return the order")
		assert.NotEmpty(t, result.OrderID, "PlaceSpreadOrder should return the order ID")
	}
}

func TestCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelSpreadOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no order ID", arg: &CancelSpreadOrderRequest{}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelSpreadOrder(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CancelSpreadOrder must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelSpreadOrderRequest
			body string
		}{
			{name: "order ID", arg: &CancelSpreadOrderRequest{OrderID: "12345"}, body: `{"ordId":"12345"}`},
			{name: "client order ID", arg: &CancelSpreadOrderRequest{ClientOrderID: "oktswap6"}, body: `{"clOrdId":"oktswap6"}`},
			{name: "both IDs", arg: &CancelSpreadOrderRequest{OrderID: "12345", ClientOrderID: "oktswap6"}, body: `{"ordId":"12345","clOrdId":"oktswap6"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/cancel-order", tc.body, []map[string]string{{"ordId": "12345", "sCode": "0"}})
				_, err := ex.CancelSpreadOrder(t.Context(), tc.arg)
				require.NoError(t, err, "CancelSpreadOrder must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpreadOrder(contextGenerate(), &CancelSpreadOrderRequest{OrderID: "12345"})
	require.NoError(t, err, "CancelSpreadOrder must not error")
	if mockTests {
		exp := &SpreadOrderResponse{ClientOrderID: "oktswap6", OrderID: "12345"}
		assert.Equal(t, exp, result, "CancelSpreadOrder should decode every field")
	} else {
		assert.NotNil(t, result, "CancelSpreadOrder should return a result")
	}
}

func TestCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllSpreadOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllSpreadOrders must reject a nil request")

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelAllSpreadOrdersRequest
			body string
		}{
			{name: "one spread", arg: &CancelAllSpreadOrdersRequest{SpreadID: spreadPair.String()}, body: `{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
			{name: "every spread", arg: &CancelAllSpreadOrdersRequest{}, body: `{}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/mass-cancel", tc.body, []map[string]bool{{"result": true}})
				_, err := ex.CancelAllSpreadOrders(t.Context(), tc.arg)
				require.NoError(t, err, "CancelAllSpreadOrders must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllSpreadOrders(contextGenerate(), &CancelAllSpreadOrdersRequest{SpreadID: spreadPair.String()})
	require.NoError(t, err, "CancelAllSpreadOrders must not error")
	if mockTests {
		assert.Equal(t, &CancelAllSpreadOrdersResponse{Result: true}, result, "CancelAllSpreadOrders should decode every field")
	} else {
		require.NotNil(t, result, "CancelAllSpreadOrders must return a result")
		assert.True(t, result.Result, "CancelAllSpreadOrders should report success")
	}
}

func TestAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendSpreadOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", arg: &AmendSpreadOrderRequest{}, err: order.ErrOrderIDNotSet},
		{name: "no order ID", arg: &AmendSpreadOrderRequest{NewSize: 2}, err: order.ErrOrderIDNotSet},
		{name: "no new size or price", arg: &AmendSpreadOrderRequest{OrderID: "2510789768709120"}, err: errSizeOrPriceIsRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendSpreadOrder(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "AmendSpreadOrder must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendSpreadOrderRequest
			body string
		}{
			{
				name: "every optional parameter",
				arg:  &AmendSpreadOrderRequest{OrderID: "2510789768709120", ClientOrderID: "b1", RequestID: "b12344", NewSize: 2, NewPrice: new(-30.5)},
				body: `{"ordId":"2510789768709120","clOrdId":"b1","reqId":"b12344","newSz":"2","newPx":"-30.5"}`,
			},
			{
				name: "required parameters only",
				arg:  &AmendSpreadOrderRequest{ClientOrderID: "b1", NewPrice: new(-30.5)},
				body: `{"clOrdId":"b1","newPx":"-30.5"}`,
			},
			{
				name: "a zero price alone",
				arg:  &AmendSpreadOrderRequest{OrderID: "2510789768709120", NewPrice: new(0.0)},
				body: `{"ordId":"2510789768709120","newPx":"0"}`,
			},
			{
				name: "a zero price beside a new size",
				arg:  &AmendSpreadOrderRequest{OrderID: "2510789768709120", NewSize: 2, NewPrice: new(0.0)},
				body: `{"ordId":"2510789768709120","newSz":"2","newPx":"0"}`,
			},
			{
				name: "a new size alone",
				arg:  &AmendSpreadOrderRequest{OrderID: "2510789768709120", NewSize: 2},
				body: `{"ordId":"2510789768709120","newSz":"2"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/amend-order", tc.body, []map[string]string{{"ordId": "2510789768709120", "sCode": "0"}})
				_, err := ex.AmendSpreadOrder(t.Context(), tc.arg)
				require.NoError(t, err, "AmendSpreadOrder must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendSpreadOrder(contextGenerate(), &AmendSpreadOrderRequest{OrderID: "2510789768709120", RequestID: "b12344", NewSize: 2})
	require.NoError(t, err, "AmendSpreadOrder must not error")
	if mockTests {
		exp := &SpreadOrderResponse{OrderID: "2510789768709120", RequestID: "b12344"}
		assert.Equal(t, exp, result, "AmendSpreadOrder should decode every field")
	} else {
		assert.NotNil(t, result, "AmendSpreadOrder should return a result")
	}
}

func TestAmendSpreadOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendSpreadOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no order ID", arg: &AmendSpreadOrderRequest{NewSize: 2}, err: order.ErrOrderIDNotSet},
		{name: "no new size or price", arg: &AmendSpreadOrderRequest{ClientOrderID: "b1"}, err: errSizeOrPriceIsRequired},
		{name: "negative new price", arg: &AmendSpreadOrderRequest{ClientOrderID: "b1", NewPrice: new(-30.5)}},
		{name: "zero new price", arg: &AmendSpreadOrderRequest{ClientOrderID: "b1", NewPrice: new(0.0)}},
		{name: "new size", arg: &AmendSpreadOrderRequest{OrderID: "2510789768709120", NewSize: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.err != nil {
				require.ErrorIs(t, tc.arg.Validate(), tc.err, "Validate must reject the request")
				return
			}
			require.NoError(t, tc.arg.Validate(), "Validate must accept the request")
		})
	}
}

func TestGetSpreadOrderDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetSpreadOrderDetailsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no order ID", arg: &GetSpreadOrderDetailsRequest{}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpreadOrderDetails(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetSpreadOrderDetails must reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadOrderDetails(contextGenerate(), &GetSpreadOrderDetailsRequest{OrderID: "1234567", ClientOrderID: "b1"})
	require.NoError(t, err, "GetSpreadOrderDetails must not error")
	if mockTests {
		exp := &SpreadOrderDetailResponse{
			SpreadID:            spreadPair.String(),
			OrderID:             "1234567",
			ClientOrderID:       "b1",
			Tag:                 "tag-here",
			Price:               -30.5,
			Size:                3,
			OrderType:           orderLimit,
			Side:                order.Buy.Lower(),
			FillSize:            1,
			FillPrice:           -30.5,
			TradeID:             "3647416334301888512",
			AccumulatedFillSize: 1,
			PendingFillSize:     OptionalNumber{Value: new(types.Number(0))},
			CancelledSize:       2,
			AveragePrice:        -30.5,
			State:               "canceled",
			CancelSource:        "1",
			UpdateTime:          types.Time(time.UnixMilli(1795239760000)),
			CreationTime:        types.Time(time.UnixMilli(1795239700000)),
		}
		assert.Equal(t, exp, result, "GetSpreadOrderDetails should decode every field")
	} else {
		assert.NotNil(t, result, "GetSpreadOrderDetails should return a result")
	}
}

func TestGetActiveSpreadOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveSpreadOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetActiveSpreadOrders must reject a nil request")

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
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		resp, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
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
	})

	t.Run("from order ID seeds the endId cursor", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide, FromOrderID: "SEED-1",
		})
		require.NoError(t, err, "GetActiveOrders must not error when FromOrderID seeds the spread crawl")
		require.Len(t, active, 1, "the seeded crawl must return the listing's orders")
		assert.Equal(t, "SPD-1", active[0].OrderID, "the pending spread order should be returned")
	})

	t.Run("reports execution values", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide,
		})
		require.NoError(t, err, "GetActiveOrders must not error")
		require.Len(t, active, 1, "the pending spread order must be returned")
		assert.Equal(t, 90.0, active[0].AverageExecutedPrice, "GetActiveOrders should read the average filled price")
		assert.Equal(t, 1.0, active[0].Amount, "GetActiveOrders should read the order size")
		assert.Equal(t, 0.3, active[0].ExecutedAmount, "GetActiveOrders should read the accumulated fill")
		assert.Zero(t, active[0].Cost, "GetActiveOrders should leave a spread order's cost unset")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveSpreadOrders(contextGenerate(), &GetActiveSpreadOrdersRequest{
		SpreadID:  spreadPair.String(),
		OrderType: orderPostOnly,
		State:     "partially_filled",
		BeginID:   "312269865356374015",
		EndID:     "312269865356374017",
		Limit:     10,
	})
	require.NoError(t, err, "GetActiveSpreadOrders must not error")
	if mockTests {
		exp := []SpreadOrderDetailResponse{{
			SpreadID:            spreadPair.String(),
			OrderID:             "312269865356374016",
			ClientOrderID:       "b1",
			Price:               -31.2,
			Size:                3,
			OrderType:           orderPostOnly,
			Side:                order.Buy.Lower(),
			FillSize:            0.5,
			FillPrice:           -31.2,
			TradeID:             "3647416334301888513",
			AccumulatedFillSize: 0.5,
			PendingFillSize:     OptionalNumber{Value: new(types.Number(2.5))},
			PendingSettleSize:   1,
			AveragePrice:        -31.2,
			State:               "partially_filled",
			UpdateTime:          types.Time(time.UnixMilli(1795239760000)),
			CreationTime:        types.Time(time.UnixMilli(1795239700000)),
		}}
		assert.Equal(t, exp, result, "GetActiveSpreadOrders should decode every field")
	} else {
		assert.NotNil(t, result, "GetActiveSpreadOrders should return a result")
	}
}

func TestGetCompletedSpreadOrdersLast21Days(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetCompletedSpreadOrdersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "begin after end", arg: &GetCompletedSpreadOrdersRequest{BeginTime: time.UnixMilli(1795300000000), EndTime: time.UnixMilli(1795200000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCompletedSpreadOrdersLast21Days(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetCompletedSpreadOrdersLast21Days must reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCompletedSpreadOrdersLast21Days(contextGenerate(), &GetCompletedSpreadOrdersRequest{
		SpreadID:  spreadPair.String(),
		OrderType: orderLimit,
		State:     "canceled",
		BeginID:   "312269865356374015",
		EndID:     "312269865356374017",
		BeginTime: time.UnixMilli(1795200000000),
		EndTime:   time.UnixMilli(1795300000000),
		Limit:     10,
	})
	require.NoError(t, err, "GetCompletedSpreadOrdersLast21Days must not error")
	if mockTests {
		exp := []SpreadOrderDetailResponse{{
			SpreadID:            spreadPair.String(),
			OrderID:             "312269865356374016",
			ClientOrderID:       "b1",
			Price:               -41.8,
			Size:                3,
			OrderType:           orderLimit,
			Side:                order.Sell.Lower(),
			FillSize:            0.0039,
			FillPrice:           -41.8,
			TradeID:             "3647286779197095937",
			AccumulatedFillSize: 0.0039,
			PendingFillSize:     OptionalNumber{Value: new(types.Number(0))},
			CancelledSize:       2.9961,
			AveragePrice:        -41.8,
			State:               "canceled",
			CancelSource:        "20",
			UpdateTime:          types.Time(time.UnixMilli(1795232032883)),
			CreationTime:        types.Time(time.UnixMilli(1795231800000)),
		}}
		assert.Equal(t, exp, result, "GetCompletedSpreadOrdersLast21Days should decode every field")
	} else {
		assert.NotNil(t, result, "GetCompletedSpreadOrdersLast21Days should return a result")
	}
}

func TestGetCompletedSpreadOrdersLast3Months(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetCompletedSpreadOrdersArchiveRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "begin after end", arg: &GetCompletedSpreadOrdersArchiveRequest{BeginTime: time.UnixMilli(1795300000000), EndTime: time.UnixMilli(1795200000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCompletedSpreadOrdersLast3Months(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetCompletedSpreadOrdersLast3Months must reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCompletedSpreadOrdersLast3Months(contextGenerate(), &GetCompletedSpreadOrdersArchiveRequest{
		SpreadID:         spreadPair.String(),
		OrderType:        orderLimit,
		State:            "canceled",
		InstrumentType:   instTypeSwap,
		InstrumentFamily: mainPair.String(),
		BeginID:          "312269865356374015",
		EndID:            "312269865356374017",
		BeginTime:        time.UnixMilli(1795200000000),
		EndTime:          time.UnixMilli(1795300000000),
		Limit:            10,
	})
	require.NoError(t, err, "GetCompletedSpreadOrdersLast3Months must not error")
	if mockTests {
		exp := []SpreadOrderDetailResponse{{
			SpreadID:            spreadPair.String(),
			OrderID:             "312269865356374016",
			ClientOrderID:       "b1",
			Price:               -41.8,
			Size:                3,
			OrderType:           orderLimit,
			Side:                order.Sell.Lower(),
			FillSize:            0.0039,
			FillPrice:           -41.8,
			TradeID:             "3647286779197095937",
			AccumulatedFillSize: 0.0039,
			PendingFillSize:     OptionalNumber{Value: new(types.Number(0))},
			CancelledSize:       2.9961,
			AveragePrice:        -41.8,
			State:               "canceled",
			CancelSource:        "15",
			UpdateTime:          types.Time(time.UnixMilli(1795232032883)),
			CreationTime:        types.Time(time.UnixMilli(1795231800000)),
		}}
		assert.Equal(t, exp, result, "GetCompletedSpreadOrdersLast3Months should decode every field")
	} else {
		assert.NotNil(t, result, "GetCompletedSpreadOrdersLast3Months should return a result")
	}
}

func TestSpreadOrderHistoryParams(t *testing.T) {
	t.Parallel()
	begin, end := time.UnixMilli(1795228800000), time.UnixMilli(1795232100000)
	for _, tc := range []struct {
		name                                       string
		spreadID, orderType, state, beginID, endID string
		beginTime, endTime                         time.Time
		limit                                      uint64
		exp                                        url.Values
		err                                        error
	}{
		{name: "begin after end", beginTime: end, endTime: begin, err: common.ErrStartAfterEnd},
		{name: "no filter", exp: url.Values{}},
		{name: "spread ID", spreadID: spreadPair.String(), exp: url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}}},
		{name: "order type", orderType: orderLimit, exp: url.Values{"ordType": {"limit"}}},
		{name: "state", state: "filled", exp: url.Values{"state": {"filled"}}},
		{name: "begin ID", beginID: "2510789768709120", exp: url.Values{"beginId": {"2510789768709120"}}},
		{name: "end ID", endID: "2510789768709121", exp: url.Values{"endId": {"2510789768709121"}}},
		{name: "begin time", beginTime: begin, exp: url.Values{"begin": {"1795228800000"}}},
		{name: "end time", endTime: end, exp: url.Values{"end": {"1795232100000"}}},
		{name: "equal begin and end", beginTime: begin, endTime: begin, exp: url.Values{"begin": {"1795228800000"}, "end": {"1795228800000"}}},
		{name: "limit", limit: 100, exp: url.Values{"limit": {"100"}}},
		{
			name: "every filter", spreadID: spreadPair.String(), orderType: orderLimit, state: "canceled", beginID: "2510789768709120", endID: "2510789768709121", beginTime: begin, endTime: end, limit: 100,
			exp: url.Values{
				"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}, "ordType": {"limit"}, "state": {"canceled"}, "beginId": {"2510789768709120"}, "endId": {"2510789768709121"},
				"begin": {"1795228800000"}, "end": {"1795232100000"}, "limit": {"100"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := spreadOrderHistoryParams(tc.spreadID, tc.orderType, tc.state, tc.beginID, tc.endID, tc.beginTime, tc.endTime, tc.limit)
			require.ErrorIs(t, err, tc.err, "spreadOrderHistoryParams must return the expected error")
			assert.Equal(t, tc.exp, params, "spreadOrderHistoryParams should encode every set filter")
		})
	}
}

func TestGetSpreadTradesOfLast7Days(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetSpreadTradesRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "begin after end", arg: &GetSpreadTradesRequest{BeginTime: time.UnixMilli(1795300000000), EndTime: time.UnixMilli(1795200000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpreadTradesOfLast7Days(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetSpreadTradesOfLast7Days must reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadTradesOfLast7Days(contextGenerate(), &GetSpreadTradesRequest{
		SpreadID:  spreadPair.String(),
		TradeID:   "3647286779197095937",
		OrderID:   "312269865356374016",
		BeginID:   "3647286779197095936",
		EndID:     "3647286779197095938",
		BeginTime: time.UnixMilli(1795200000000),
		EndTime:   time.UnixMilli(1795300000000),
		Limit:     10,
	})
	require.NoError(t, err, "GetSpreadTradesOfLast7Days must not error")
	if mockTests {
		exp := []SpreadTradeResponse{{
			SpreadID:      spreadPair.String(),
			TradeID:       "3647286779197095937",
			OrderID:       "312269865356374016",
			ClientOrderID: "b16",
			Tag:           "tag-here",
			FillPrice:     -41.8,
			FillSize:      0.0039,
			Side:          order.Sell.Lower(),
			State:         "rejected",
			ExecutionType: "M",
			Timestamp:     types.Time(time.UnixMilli(1795232032883)),
			Legs: []SpreadTradeLeg{
				{InstrumentID: perpetualSwapPair.String(), Price: 74420.1, Size: 0.0039, SizeInContracts: 0.39, Side: order.Buy.Lower(), FillPNL: -0.021, Fee: -0.0145, FeeCurrency: currency.USDT, TradeID: "1232342342"},
				{InstrumentID: mainPair.String(), Price: 74461.9, Size: 0.0039, Side: order.Sell.Lower(), Fee: -0.0000039, FeeCurrency: currency.BTC, TradeID: "5345646634"},
			},
			Code:    "51008",
			Message: "Order failed. Insufficient BTC balance in account.",
		}}
		assert.Equal(t, exp, result, "GetSpreadTradesOfLast7Days should decode every field")
	} else {
		assert.NotNil(t, result, "GetSpreadTradesOfLast7Days should return a result")
	}
}

func TestGetPublicSpreads(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicSpreads(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicSpreads must reject a nil request")

	for _, tc := range []struct {
		name string
		arg  *GetPublicSpreadsRequest
		exp  []SpreadInstrumentResponse
	}{
		{
			name: "every spread",
			arg:  &GetPublicSpreadsRequest{},
			exp: []SpreadInstrumentResponse{
				{
					SpreadID:      currency.NewPairWithDelimiter("BTC", "USD-261225_BTC-USD-270625", currency.DashDelimiter),
					SpreadType:    "inverse",
					State:         instrumentStateLive,
					BaseCurrency:  currency.BTC,
					SizeCurrency:  currency.USD,
					QuoteCurrency: currency.USD,
					TickSize:      0.5,
					MinSize:       10,
					LotSize:       10,
					ListTime:      types.Time(time.UnixMilli(1784620800000)),
					ExpiryTime:    types.Time(time.UnixMilli(1798185600000)),
					UpdateTime:    types.Time(time.UnixMilli(1794451805385)),
					Legs: []SpreadInstrumentLeg{
						{InstrumentID: "BTC-USD-261225", Side: order.Sell.Lower()},
						{InstrumentID: "BTC-USD-270625", Side: order.Buy.Lower()},
					},
				},
				{
					SpreadID:      currency.NewPairWithDelimiter("ETH", "USD-SWAP_ETH-USD-271231", currency.DashDelimiter),
					SpreadType:    "inverse",
					State:         instrumentStateLive,
					BaseCurrency:  currency.ETH,
					SizeCurrency:  currency.USD,
					QuoteCurrency: currency.USD,
					TickSize:      0.05,
					MinSize:       10,
					LotSize:       10,
					ListTime:      types.Time(time.UnixMilli(1793347800580)),
					ExpiryTime:    types.Time(time.UnixMilli(1830240000000)),
					UpdateTime:    types.Time(time.UnixMilli(1794451805416)),
					Legs: []SpreadInstrumentLeg{
						{InstrumentID: "ETH-USD-SWAP", Side: order.Sell.Lower()},
						{InstrumentID: "ETH-USD-271231", Side: order.Buy.Lower()},
					},
				},
			},
		},
		{
			name: "every filter",
			arg:  &GetPublicSpreadsRequest{BaseCurrency: currency.BTC, InstrumentID: perpetualSwapPair.String(), SpreadID: spreadPair.String(), State: instrumentStateLive},
			exp: []SpreadInstrumentResponse{{
				SpreadID:      spreadPair,
				SpreadType:    "linear",
				State:         instrumentStateLive,
				BaseCurrency:  currency.BTC,
				SizeCurrency:  currency.BTC,
				QuoteCurrency: currency.USDT,
				TickSize:      0.1,
				MinSize:       0.0001,
				LotSize:       0.0001,
				ListTime:      types.Time(time.UnixMilli(1606468572000)),
				UpdateTime:    types.Time(time.UnixMilli(1794451805535)),
				Legs: []SpreadInstrumentLeg{
					{InstrumentID: mainPair.String(), Side: order.Sell.Lower()},
					{InstrumentID: perpetualSwapPair.String(), Side: order.Buy.Lower()},
				},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPublicSpreads(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetPublicSpreads must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPublicSpreads should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetPublicSpreads must return spreads")
			for i := range result {
				assert.NotEmpty(t, result[i].Legs, "GetPublicSpreads should return each spread's legs")
			}
		})
	}
}

func TestGetPublicSpreadOrderBooks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetPublicSpreadOrderBooksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no spread ID", arg: &GetPublicSpreadOrderBooksRequest{}, err: errMissingInstrumentID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetPublicSpreadOrderBooks(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetPublicSpreadOrderBooks must reject the request")
		})
	}

	result, err := e.GetPublicSpreadOrderBooks(contextGenerate(), &GetPublicSpreadOrderBooksRequest{SpreadID: spreadPair.String()})
	require.NoError(t, err, "GetPublicSpreadOrderBooks must not error")
	if mockTests {
		exp := []SpreadOrderBookResponse{{
			Bids:      [][]types.Number{{-63.9, 0.2297, 1}},
			Asks:      [][]types.Number{{-8.7, 2.7457, 1}},
			Timestamp: types.Time(time.UnixMilli(1795406564451)),
		}}
		assert.Equal(t, exp, result, "GetPublicSpreadOrderBooks should decode every field")
		return
	}
	require.Len(t, result, 1, "GetPublicSpreadOrderBooks must return the spread's book")
	assert.NotZero(t, result[0].Timestamp, "GetPublicSpreadOrderBooks should return the book's time")
}

func TestGetPublicSpreadTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicSpreadTickers(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetPublicSpreadTickers must reject an empty spread ID")

	t.Run("overlays the book's top", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

		var mu sync.Mutex
		counts := map[string]int{}
		var booksQuery url.Values
		srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			counts[r.URL.Path]++
			if r.URL.Path == "/sprd/books" {
				booksQuery = r.URL.Query()
			}
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/market/sprd-ticker":
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","lastSz":"0.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","open24h":"4","high24h":"14.5","low24h":"-2.2","vol24h":"6.67","ts":"1715331406485"}]}`))
			case "/sprd/books":
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["15.2","3.3","1"]],"bids":[["14.8","4.4","2"]],"ts":"1715331407999"}]}`))
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

		result, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
		require.NoError(t, err, "GetPublicSpreadTickers must not error")
		require.NotEmpty(t, result, "GetPublicSpreadTickers must return a ticker")

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, counts["/sprd/books"], "the spread ticker should refresh the top of book from sprd/books")
		assert.Equal(t, "1", booksQuery.Get("sz"), "the top of book request should ask for a single level")
		assert.Equal(t, 1, counts["/market/sprd-ticker"], "the last and 24 hour figures should come from market/sprd-ticker")
		assert.Equal(t, 14.8, result[0].BidPrice.Float64(), "bidPx should come from the sprd/books top of book")
		assert.Equal(t, 4.4, result[0].BidSize.Float64(), "bidSz should come from the sprd/books top of book")
		assert.Equal(t, 15.2, result[0].AskPrice.Float64(), "askPx should come from the sprd/books top of book")
		assert.Equal(t, 3.3, result[0].AskSize.Float64(), "askSz should come from the sprd/books top of book")
		assert.Equal(t, 14.5, result[0].Last.Float64(), "last should come from market/sprd-ticker")
		assert.Equal(t, 4.0, result[0].OpenPrice24Hour.Float64(), "open24h should come from market/sprd-ticker")
		assert.Equal(t, 14.5, result[0].HighestPrice24Hour.Float64(), "high24h should come from market/sprd-ticker")
		assert.Equal(t, -2.2, result[0].LowestPrice24Hour.Float64(), "low24h should come from market/sprd-ticker")
		assert.Equal(t, 6.67, result[0].TradingVolume24Hour.Float64(), "vol24h should come from market/sprd-ticker")
		assert.Equal(t, int64(1715331407999), result[0].Timestamp.Time().UnixMilli(), "the timestamp should advance to the sprd/books generation time")
	})

	t.Run("a book failure fails the ticker", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

		var mu sync.Mutex
		var paths []string
		srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/sprd/books" {
				_, _ = w.Write([]byte(`{"code":"50026","msg":"System error, please try again later.","data":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","ts":"1715331406485"}]}`))
		}))
		b := e.GetBase()
		b.SkipAuthCheck = true
		require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
		for k := range b.API.Endpoints.GetURLMap() {
			require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
		}

		_, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
		require.ErrorContains(t, err, "50026", "a sprd/books failure must fail the spread ticker rather than return its cached top of book")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/market/sprd-ticker", "/sprd/books"}, paths, "the books refresh should fail only after the ticker request succeeds")
	})

	t.Run("a side the book lacks is cleared", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			books string
			exp   SpreadTickerResponse
		}{
			{
				name:  "bids only",
				books: `[{"asks":[],"bids":[["14.8","4.4","2"]],"ts":"1715331407999"}]`,
				exp:   SpreadTickerResponse{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, BidPrice: 14.8, BidSize: 4.4, Timestamp: types.Time(time.UnixMilli(1715331407999))},
			},
			{
				name:  "asks only",
				books: `[{"asks":[["15.2","3.3","1"]],"bids":[],"ts":"1715331407999"}]`,
				exp:   SpreadTickerResponse{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, AskPrice: 15.2, AskSize: 3.3, Timestamp: types.Time(time.UnixMilli(1715331407999))},
			},
			{
				name:  "empty book",
				books: `[{"asks":[],"bids":[],"ts":"1715331407999"}]`,
				exp:   SpreadTickerResponse{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, Timestamp: types.Time(time.UnixMilli(1715331407999))},
			},
			{
				name:  "older book",
				books: `[{"asks":[["15.2","3.3","1"]],"bids":[],"ts":"1715331400000"}]`,
				exp:   SpreadTickerResponse{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, AskPrice: 15.2, AskSize: 3.3, Timestamp: types.Time(time.UnixMilli(1715331406485))},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
				srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/market/sprd-ticker":
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","ts":"1715331406485"}]}`))
					case "/sprd/books":
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.books + `}`))
					default:
						assert.Failf(t, "unexpected request", "path %s should not be requested", r.URL.Path)
					}
				}))
				b := e.GetBase()
				require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
				for k := range b.API.Endpoints.GetURLMap() {
					require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
				}

				result, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
				require.NoError(t, err, "GetPublicSpreadTickers must not error")
				require.Len(t, result, 1, "GetPublicSpreadTickers must return the ticker")
				assert.Equal(t, tc.exp, result[0], "a side missing from sprd/books should be cleared, not left at the cached quote")
			})
		}
	})

	result, err := e.GetPublicSpreadTickers(contextGenerate(), spreadPair.String())
	require.NoError(t, err, "GetPublicSpreadTickers must not error")
	if mockTests {
		exp := []SpreadTickerResponse{{
			SpreadID:           spreadPair.String(),
			Last:               -30.5,
			LastSize:           0.0833,
			AskPrice:           -8.7,
			AskSize:            2.7457,
			BidPrice:           -63.9,
			BidSize:            0.2297,
			OpenPrice24Hour:    -30.5,
			HighestPrice24Hour: -30.5,
			LowestPrice24Hour:  -30.5,
			Timestamp:          types.Time(time.UnixMilli(1795406573867)),
		}}
		assert.Equal(t, exp, result, "GetPublicSpreadTickers should decode the ticker with the book's top")
		return
	}
	require.Len(t, result, 1, "GetPublicSpreadTickers must return the spread's ticker")
	assert.Equal(t, spreadPair.String(), result[0].SpreadID, "GetPublicSpreadTickers should return the requested spread")
}

func TestOverlaySpreadBook(t *testing.T) {
	t.Parallel()
	cached := types.Time(time.UnixMilli(1795231800000))
	ticker := func() *SpreadTickerResponse {
		return &SpreadTickerResponse{
			SpreadID: spreadPair.String(), Last: -42.2, LastSize: 0.0075, AskPrice: -41, AskSize: 3, BidPrice: -43, BidSize: 4,
			OpenPrice24Hour: -30.6, HighestPrice24Hour: -20, LowestPrice24Hour: -50, TradingVolume24Hour: 1.5, Timestamp: cached,
		}
	}
	overlaid := func(askPrice, askSize, bidPrice, bidSize types.Number, timestamp types.Time) *SpreadTickerResponse {
		tk := ticker()
		tk.AskPrice, tk.AskSize, tk.BidPrice, tk.BidSize, tk.Timestamp = askPrice, askSize, bidPrice, bidSize, timestamp
		return tk
	}
	newer, older := types.Time(time.UnixMilli(1795231900000)), types.Time(time.UnixMilli(1795231700000))
	asks, bids := [][]types.Number{{-41.6, 2.5, 1}, {-41.5, 1, 1}}, [][]types.Number{{-42.4, 1.25, 1}, {-42.5, 3, 2}}
	for _, tc := range []struct {
		name string
		book *SpreadOrderBookResponse
		exp  *SpreadTickerResponse
	}{
		{name: "newer book", book: &SpreadOrderBookResponse{Asks: asks, Bids: bids, Timestamp: newer}, exp: overlaid(-41.6, 2.5, -42.4, 1.25, newer)},
		{name: "older book keeps the ticker's time", book: &SpreadOrderBookResponse{Asks: asks, Bids: bids, Timestamp: older}, exp: overlaid(-41.6, 2.5, -42.4, 1.25, cached)},
		{name: "book as old as the ticker", book: &SpreadOrderBookResponse{Asks: asks, Bids: bids, Timestamp: cached}, exp: overlaid(-41.6, 2.5, -42.4, 1.25, cached)},
		{name: "book without asks clears the ask", book: &SpreadOrderBookResponse{Bids: bids, Timestamp: newer}, exp: overlaid(0, 0, -42.4, 1.25, newer)},
		{name: "book without bids clears the bid", book: &SpreadOrderBookResponse{Asks: asks, Timestamp: newer}, exp: overlaid(-41.6, 2.5, 0, 0, newer)},
		{name: "empty book clears both sides", book: &SpreadOrderBookResponse{}, exp: overlaid(0, 0, 0, 0, cached)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk := ticker()
			overlaySpreadBook(tk, tc.book)
			assert.Equal(t, tc.exp, tk, "overlaySpreadBook should replace the top of book and keep the ticker's other fields")
		})
	}
}

func TestSpreadBookTop(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		levels      [][]types.Number
		price, size types.Number
	}{
		{name: "no levels"},
		{name: "level without a size", levels: [][]types.Number{{-8.7}}},
		{name: "best level", levels: [][]types.Number{{-8.7, 2.7457, 1}, {-8.5, 1, 1}}, price: -8.7, size: 2.7457},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			price, size := spreadBookTop(tc.levels)
			assert.Equal(t, tc.price, price, "spreadBookTop should return the best level's price")
			assert.Equal(t, tc.size, size, "spreadBookTop should return the best level's size")
		})
	}
}

func TestGetPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicSpreadTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicSpreadTrades must reject a nil request")

	for _, tc := range []struct {
		name string
		arg  *GetPublicSpreadTradesRequest
		exp  []SpreadPublicTradeResponse
	}{
		{
			name: "every spread",
			arg:  &GetPublicSpreadTradesRequest{},
			exp: []SpreadPublicTradeResponse{
				{SpreadID: "BTC-USD-SWAP_BTC-USD-270924", TradeID: "3650204189554180096", Price: 2857.5, Size: 10, Side: order.Buy.Lower(), Timestamp: types.Time(time.UnixMilli(1795405924094))},
				{SpreadID: "BTC-USD-SWAP_BTC-USD-270625", TradeID: "3650202394425606144", Price: 1974.5, Size: 10, Side: order.Buy.Lower(), Timestamp: types.Time(time.UnixMilli(1795405817094))},
				{SpreadID: "BTC-USD-SWAP_BTC-USD-270924", TradeID: "3650200225668464640", Price: 2852, Size: 10, Side: order.Buy.Lower(), Timestamp: types.Time(time.UnixMilli(1795405687828))},
			},
		},
		{
			name: "one spread",
			arg:  &GetPublicSpreadTradesRequest{SpreadID: spreadPair.String()},
			exp: []SpreadPublicTradeResponse{
				{SpreadID: spreadPair.String(), TradeID: "3647416334301888512", Price: -30.5, Size: 0.0833, Side: order.Buy.Lower(), Timestamp: types.Time(time.UnixMilli(1795239755129))},
				{SpreadID: spreadPair.String(), TradeID: "3647288426451271680", Price: -32.9, Size: 0.0018, Side: order.Buy.Lower(), Timestamp: types.Time(time.UnixMilli(1795232131068))},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPublicSpreadTrades(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetPublicSpreadTrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPublicSpreadTrades should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetPublicSpreadTrades must return trades")
			if tc.arg.SpreadID != "" {
				for i := range result {
					assert.Equal(t, tc.arg.SpreadID, result[i].SpreadID, "GetPublicSpreadTrades should return the requested spread's trades")
				}
			}
		})
	}
}

func TestGetSpreadCandlesticks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetSpreadCandlesticksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no spread ID", arg: &GetSpreadCandlesticksRequest{Interval: kline.FiveMin, Limit: 10}, err: errMissingInstrumentID},
		{name: "window start after its end", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Before: time.UnixMilli(1795232100000), After: time.UnixMilli(1795228800000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpreadCandlesticks(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetSpreadCandlesticks must reject the request")
		})
	}

	before, after := time.UnixMilli(1795228800000), time.UnixMilli(1795232100000)
	if !mockTests {
		before, after = time.Now().AddDate(0, 0, -1), time.Now()
	}
	result, err := e.GetSpreadCandlesticks(contextGenerate(), &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.FiveMin, Before: before, After: after, Limit: 10})
	require.NoError(t, err, "GetSpreadCandlesticks must not error")
	if mockTests {
		require.Len(t, result, 10, "GetSpreadCandlesticks must return every candlestick in the window")
		exp := []SpreadCandlestickResponse{
			{Timestamp: types.Time(time.UnixMilli(1795231800000)), Open: -41.8, High: -41.8, Low: -42.2, Close: -42.2, Volume: 0.0075, Confirm: true},
			{Timestamp: types.Time(time.UnixMilli(1795231500000)), Open: -31.5, High: -30.6, Low: -31.6, Close: -30.6, Volume: 0.0275, Confirm: true},
		}
		assert.Equal(t, exp, result[:2], "GetSpreadCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSpreadCandlesticks should not return an empty result")
}

func TestGetSpreadCandlesticksHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetSpreadCandlesticksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no spread ID", arg: &GetSpreadCandlesticksRequest{Interval: kline.FiveMin, Limit: 10}, err: errMissingInstrumentID},
		{name: "window start after its end", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Before: time.UnixMilli(1795232100000), After: time.UnixMilli(1795228800000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpreadCandlesticksHistory(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetSpreadCandlesticksHistory must reject the request")
		})
	}

	before, after := time.UnixMilli(1795228800000), time.UnixMilli(1795232100000)
	if !mockTests {
		before, after = time.Now().AddDate(0, 0, -1), time.Now()
	}
	result, err := e.GetSpreadCandlesticksHistory(contextGenerate(), &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.FiveMin, Before: before, After: after, Limit: 10})
	require.NoError(t, err, "GetSpreadCandlesticksHistory must not error")
	if mockTests {
		require.Len(t, result, 10, "GetSpreadCandlesticksHistory must return every candlestick in the window")
		exp := []SpreadCandlestickResponse{
			{Timestamp: types.Time(time.UnixMilli(1795231800000)), Open: -41.8, High: -41.8, Low: -42.2, Close: -42.2, Volume: 0.0075, Confirm: true},
			{Timestamp: types.Time(time.UnixMilli(1795231500000)), Open: -31.5, High: -30.6, Low: -31.6, Close: -30.6, Volume: 0.0275, Confirm: true},
		}
		assert.Equal(t, exp, result[:2], "GetSpreadCandlesticksHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSpreadCandlesticksHistory should not return an empty result")
}

func TestSpreadCandlestickParams(t *testing.T) {
	t.Parallel()
	older, newer := time.UnixMilli(1795228800000), time.UnixMilli(1795232100000)
	spread := url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}}
	with := func(key, value string) url.Values {
		params := url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}}
		params.Set(key, value)
		return params
	}
	for _, tc := range []struct {
		name string
		arg  *GetSpreadCandlesticksRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing spread ID", arg: &GetSpreadCandlesticksRequest{Interval: kline.FiveMin, Limit: 10}, err: errMissingInstrumentID},
		{name: "before bound newer than the after bound", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Before: newer, After: older}, err: common.ErrStartAfterEnd},
		{name: "spread only", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String()}, exp: spread},
		{name: "five minute bar", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.FiveMin}, exp: with("bar", "5m")},
		{name: "daily bar opens at UTC", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.OneDay}, exp: with("bar", "1Dutc")},
		{name: "unsupported bar", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "before bound", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Before: older}, exp: with("before", "1795228800000")},
		{name: "after bound", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), After: newer}, exp: with("after", "1795232100000")},
		{
			name: "equal bounds",
			arg:  &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Before: older, After: older},
			exp:  url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}, "before": {"1795228800000"}, "after": {"1795228800000"}},
		},
		{name: "limit", arg: &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Limit: 10}, exp: with("limit", "10")},
		{
			name: "every parameter",
			arg:  &GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: kline.FiveMin, Before: older, After: newer, Limit: 10},
			exp:  url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}, "bar": {"5m"}, "before": {"1795228800000"}, "after": {"1795232100000"}, "limit": {"10"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := spreadCandlestickParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "spreadCandlestickParams must return the expected error")
			assert.Equal(t, tc.exp, params, "spreadCandlestickParams should encode every set parameter")
		})
	}

	t.Run("every bar the spread endpoints list", func(t *testing.T) {
		t.Parallel()
		for interval, bar := range map[kline.Interval]string{
			kline.OneMin: "1m", kline.ThreeMin: "3m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m",
			kline.OneHour: "1H", kline.TwoHour: "2H", kline.FourHour: "4H", kline.SixHour: "6Hutc", kline.TwelveHour: "12Hutc",
			kline.OneDay: "1Dutc", kline.TwoDay: "2Dutc", kline.ThreeDay: "3Dutc", kline.OneWeek: "1Wutc", kline.OneMonth: "1Mutc",
			kline.ThreeMonth: "3Mutc",
		} {
			t.Run(bar, func(t *testing.T) {
				t.Parallel()
				params, err := spreadCandlestickParams(&GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: interval})
				require.NoError(t, err, "spreadCandlestickParams must accept a bar the spread endpoints list")
				assert.Equal(t, with("bar", bar), params, "spreadCandlestickParams should send the bar")
			})
		}
	})

	t.Run("market bars the spread endpoints do not list", func(t *testing.T) {
		t.Parallel()
		for _, interval := range []kline.Interval{kline.ThousandMilliseconds, kline.EightHour, kline.FiveDay, kline.SixMonth, kline.OneYear} {
			t.Run(interval.Short(), func(t *testing.T) {
				t.Parallel()
				params, err := spreadCandlestickParams(&GetSpreadCandlesticksRequest{SpreadID: spreadPair.String(), Interval: interval})
				assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "spreadCandlestickParams should refuse a bar the spread endpoints do not list")
				assert.Nil(t, params, "spreadCandlestickParams should build no query for a bar the spread endpoints do not list")
			})
		}
	})
}

func TestCancelAllSpreadOrdersAfterCountdown(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{-10 * time.Second, 2 * time.Second, 10*time.Second - time.Millisecond, 120*time.Second + time.Millisecond} {
		t.Run(timeout.String(), func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelAllSpreadOrdersAfterCountdown(t.Context(), timeout)
			require.ErrorIs(t, err, errCountdownTimeoutRequired, "CancelAllSpreadOrdersAfterCountdown must reject the timeout")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			timeout time.Duration
			body    string
		}{
			{0, `{"timeOut":"0"}`},
			{10 * time.Second, `{"timeOut":"10"}`},
			{120 * time.Second, `{"timeOut":"120"}`},
		} {
			t.Run(tc.timeout.String(), func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/cancel-all-after", tc.body, []map[string]string{{"triggerTime": "0", "ts": "1587971400000"}})
				_, err := ex.CancelAllSpreadOrdersAfterCountdown(t.Context(), tc.timeout)
				require.NoError(t, err, "CancelAllSpreadOrdersAfterCountdown must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllSpreadOrdersAfterCountdown(contextGenerate(), 12*time.Second)
	require.NoError(t, err, "CancelAllSpreadOrdersAfterCountdown must not error")
	if mockTests {
		exp := &CancelAllSpreadOrdersAfterCountdownResponse{TriggerTime: types.Time(time.Unix(1587971412, 0)), Timestamp: types.Time(time.Unix(1587971400, 0))}
		assert.Equal(t, exp, result, "CancelAllSpreadOrdersAfterCountdown should decode every field")
	} else {
		require.NotNil(t, result, "CancelAllSpreadOrdersAfterCountdown must return a result")
		assert.True(t, result.TriggerTime.Time().After(result.Timestamp.Time()), "CancelAllSpreadOrdersAfterCountdown should trigger after the request")
	}
}

func TestPlaceSpreadOrderRequestValidate(t *testing.T) {
	t.Parallel()
	var p *PlaceSpreadOrderRequest
	require.ErrorIs(t, p.Validate(), common.ErrNilPointer, "Validate must reject a nil request")
	p = &PlaceSpreadOrderRequest{}
	require.ErrorIs(t, p.Validate(), errMissingInstrumentID, "Validate must reject a request without a spread ID")
	p.SpreadID = spreadPair.String()
	require.ErrorIs(t, p.Validate(), order.ErrTypeIsInvalid, "Validate must reject a request without an order type")
	p.OrderType = order.Limit.String()
	require.ErrorIs(t, p.Validate(), limits.ErrAmountBelowMin, "Validate must reject a request without a size")
	p.Size = 1
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid, "Validate must reject a request without a side")
	p.Side = order.Buy.String()
	require.NoError(t, p.Validate(), "a zero price must validate: spreads trade at zero")
	p.Price = 1
	require.NoError(t, p.Validate(), "Validate must accept a complete request")
	p.Price = -41.2
	require.NoError(t, p.Validate(), "a negative price must validate: the price of a spread order is the differential between its legs")
	p.OrderType = order.Market.String()
	p.Price = 0
	require.NoError(t, p.Validate(), "a market order must validate with no price at all")
}

func TestPlaceSpreadOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PlaceSpreadOrderRequest
		exp  string
	}{
		{"limit at a zero price", &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), Side: "buy", OrderType: orderLimit, Size: 1}, `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"buy","ordType":"limit","sz":"1","px":"0"}`},
		{"post only at a negative price", &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), ClientOrderID: "b15", Tag: "tag-here", Side: "sell", OrderType: orderPostOnly, Size: 1, Price: -41.2}, `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","clOrdId":"b15","side":"sell","ordType":"post_only","sz":"1","px":"-41.2","tag":"tag-here"}`},
		{"market given a price", &PlaceSpreadOrderRequest{SpreadID: spreadPair.String(), Side: "buy", OrderType: "MARKET", Size: 1, Price: 2}, `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"buy","ordType":"MARKET","sz":"1"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(tc.arg)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(body), "the spread order should send px with every order type but market")
			body, err = json.Marshal([]PlaceSpreadOrderRequest{*tc.arg})
			require.NoError(t, err, "Marshal must not error for a websocket argument list")
			assert.JSONEq(t, "["+tc.exp+"]", string(body), "a websocket argument list should send each order the same way")
		})
	}
}

func TestGetSpreadOrderHistoryDetails(t *testing.T) {
	t.Parallel()

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
			StartTime:   timeNow().Add(-24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds the spread crawl, with a recent start time keeping it inside the 21 day listing")

		mu.Lock()
		cursors := make([]string, len(endIDCursors))
		copy(cursors, endIDCursors)
		mu.Unlock()
		assert.Equal(t, []string{"SEED-1", "SPD-099"}, cursors, "the crawl should start from FromOrderID and advance with each page's last order ID")
		assert.Len(t, history, orderListPageSize+50, "a crawl seeded by FromOrderID should return every earlier spread order")
	})

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
			StartTime: timeNow().Add(-24 * time.Hour),
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
		"cTime": strconv.FormatInt(timeNow().UnixMilli(), 10),
	}
	tenDaysOld := map[string]string{
		"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-TEN", "ordType": orderLimit,
		"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
		"cTime": strconv.FormatInt(timeNow().Add(-10*24*time.Hour).UnixMilli(), 10),
	}
	old := map[string]string{
		"sprdId": "BTC-USDT_BTC-USDT", "ordId": "SPD-OLD", "ordType": orderLimit,
		"side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "px": "100",
		"cTime": strconv.FormatInt(timeNow().Add(-kline.OneMonth.Duration()).UnixMilli(), 10),
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
			StartTime: timeNow().Add(-kline.OneMonth.Duration()),
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
				"cTime": strconv.FormatInt(timeNow().Add(-kline.OneMonth.Duration()).UnixMilli()-int64(x), 10),
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
			StartTime: timeNow().Add(-2 * kline.OneMonth.Duration()),
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
			StartTime: timeNow().Add(-time.Hour),
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
			StartTime: timeNow().Add(-10 * 24 * time.Hour),
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

	t.Run("judges the 21 day mark by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		const day = 24 * time.Hour
		for _, tc := range []struct {
			name       string
			start, end time.Duration
			listings   map[string]int
		}{
			{name: "a start a minute inside the mark skips the archive", start: 21*day - time.Minute, listings: map[string]int{"/sprd/orders-history": 1}},
			{name: "a start a minute past the mark crawls the archive", start: 21*day + time.Minute, listings: map[string]int{"/sprd/orders-history": 1, "/sprd/orders-history-archive": 1}},
			{name: "an end a minute inside the mark reads the listing", start: 40 * day, end: 21*day - time.Minute, listings: map[string]int{"/sprd/orders-history": 1, "/sprd/orders-history-archive": 1}},
			{name: "an end a minute past the mark skips the listing", start: 40 * day, end: 21*day + time.Minute, listings: map[string]int{"/sprd/orders-history-archive": 1}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				listings := make(map[string]int)
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/sprd/orders-history", "/sprd/orders-history-archive":
						mu.Lock()
						listings[r.URL.Path]++
						mu.Unlock()
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				req := &order.MultiOrderRequest{AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide, StartTime: timeNow().Add(-tc.start)}
				if tc.end != 0 {
					req.EndTime = timeNow().Add(-tc.end)
				}
				_, err := e.GetOrderHistory(t.Context(), req)
				require.NoError(t, err, "GetOrderHistory must not error")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.listings, listings, "GetOrderHistory should choose the spread listings by the wrapper's clock")
			})
		}
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
			StartTime: timeNow().Add(-40 * 24 * time.Hour),
			EndTime:   timeNow().Add(-30 * 24 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when the window sits entirely older than the 21 day listing and only the archive covers it")
		require.Len(t, resp, 1, "the archive must be the only source for a window older than the listing")
		assert.Equal(t, "SPD-OLD", resp[0].OrderID)
	})

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
			StartTime:   timeNow().Add(-kline.OneMonth.Duration()),
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
			StartTime: timeNow().Add(-kline.OneMonth.Duration()),
		})
		require.NoError(t, err, "GetOrderHistory must stop when the archive page holds no new spread order")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"", "SPD-099"}, archiveEndIDs, "the archive crawl should stop when the endId cursor stops advancing")
		assert.Len(t, history, orderListPageSize, "the archive's orders should be recorded once")
	})

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
