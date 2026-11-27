package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSupportedCoinPairs(t *testing.T) {
	t.Parallel()
	t.Run("sends the set coins", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name             string
			fromCoin, toCoin currency.Code
			exp              url.Values
		}{
			{name: "neither", exp: url.Values{}},
			{name: "from coin", fromCoin: currency.NewCode("shib"), exp: url.Values{"fromCoin": {"SHIB"}}},
			{name: "to coin", toCoin: currency.NewCode("KSHIB"), exp: url.Values{"toCoin": {"KSHIB"}}},
			{name: "both", fromCoin: currency.BTC, toCoin: currency.USDT, exp: url.Values{"fromCoin": {"BTC"}, "toCoin": {"USDT"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/otc/coinPairs", tc.exp, `[]`)
				_, err := ex.GetSupportedCoinPairs(t.Context(), tc.fromCoin, tc.toCoin)
				require.NoError(t, err, "GetSupportedCoinPairs must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetSupportedCoinPairs(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE)
	require.NoError(t, err, "GetSupportedCoinPairs must not error")
	if mockTests {
		exp := []CoinPairResponse{
			{FromCoin: currency.BTC, ToCoin: currency.USDT, FromCoinMinimumAmount: 0.1, FromCoinMaximumAmount: 100, ToCoinMinimumAmount: 1000, ToCoinMaximumAmount: 500000},
			{FromCoin: currency.USDT, ToCoin: currency.BNB, FromCoinMinimumAmount: 2000, FromCoinMaximumAmount: 600000, ToCoinMinimumAmount: 10, ToCoinMaximumAmount: 4000},
		}
		assert.Equal(t, exp, result, "GetSupportedCoinPairs should decode every field")
	}
}

func TestRequestForQuote(t *testing.T) {
	t.Parallel()
	_, err := e.RequestForQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestForQuote must reject a nil request")
	for _, tc := range []struct {
		name  string
		req   *QuoteRequest
		err   error
		param string
	}{
		{name: "no from coin", req: &QuoteRequest{ToCoin: currency.USDT, RequestCoin: currency.BTC, RequestAmount: 1}, err: currency.ErrCurrencyCodeEmpty, param: "fromCoin"},
		{name: "no to coin", req: &QuoteRequest{FromCoin: currency.BTC, RequestCoin: currency.BTC, RequestAmount: 1}, err: currency.ErrCurrencyCodeEmpty, param: "toCoin"},
		{name: "no request coin", req: &QuoteRequest{FromCoin: currency.BTC, ToCoin: currency.USDT, RequestAmount: 1}, err: currency.ErrCurrencyCodeEmpty, param: "requestCoin"},
		{name: "no request amount", req: &QuoteRequest{FromCoin: currency.BTC, ToCoin: currency.USDT, RequestCoin: currency.BTC}, err: order.ErrAmountIsInvalid, param: "amount"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RequestForQuote(t.Context(), tc.req)
			require.ErrorIs(t, err, tc.err, "RequestForQuote must reject an incomplete request")
			assert.ErrorContains(t, err, tc.param, "RequestForQuote should name the missing parameter")
		})
	}

	t.Run("sends a fractional amount in full", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"fromCoin": {"SHIB"}, "toCoin": {"KSHIB"}, "requestCoin": {"SHIB"}, "requestAmount": {"50000.12345678"}}
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/otc/quotes", exp, `{}`)
		_, err := ex.RequestForQuote(t.Context(), &QuoteRequest{FromCoin: currency.SHIB, ToCoin: currency.NewCode("KSHIB"), RequestCoin: currency.SHIB, RequestAmount: 50000.12345678})
		require.NoError(t, err, "RequestForQuote must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.RequestForQuote(t.Context(), &QuoteRequest{FromCoin: currency.USDC, ToCoin: currency.USD, RequestCoin: currency.USDC, RequestAmount: 5})
	require.NoError(t, err, "RequestForQuote must not error")
	require.NotNil(t, result, "RequestForQuote must return a quote")
	if mockTests {
		exp := &QuoteResponse{
			QuoteID:        "9ef49bbf386a4a1294350d3331c0bd34",
			Symbol:         "USDCUSD",
			Ratio:          1,
			InverseRatio:   1,
			ValidTimestamp: types.Time(time.UnixMilli(1778611657644)),
			FromAmount:     5,
			ToAmount:       5,
		}
		assert.Equal(t, exp, result, "RequestForQuote should decode every field of a quote the account can settle")
	}

	result, err = e.RequestForQuote(t.Context(), &QuoteRequest{FromCoin: currency.BTC, ToCoin: currency.USDT, RequestCoin: currency.BTC, RequestAmount: 1})
	require.NoError(t, err, "RequestForQuote must not error")
	require.NotNil(t, result, "RequestForQuote must return a quote")
	if mockTests {
		exp := &QuoteResponse{
			Symbol:         "BTCUSDT",
			Ratio:          50550.26,
			InverseRatio:   0.00001978,
			ValidTimestamp: types.Time(time.Unix(1641806714, 0)),
			FromAmount:     1,
			ToAmount:       50550.26,
		}
		assert.Equal(t, exp, result, "RequestForQuote should decode a quote the account cannot settle, which has no quote ID")
	}
}

func TestPlaceOverTheCounterTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOverTheCounterTradeOrder(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired, "PlaceOverTheCounterTradeOrder must reject an empty quote ID")

	t.Run("posts the quote ID", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/otc/orders", url.Values{"quoteId": {"15848701022"}}, `{}`)
		_, err := ex.PlaceOverTheCounterTradeOrder(t.Context(), "15848701022")
		require.NoError(t, err, "PlaceOverTheCounterTradeOrder must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live over-the-counter order")
	}
	skipIfCredentialsUnset(t)
	quoteID := "9ef49bbf386a4a1294350d3331c0bd34"
	if !mockTests {
		quote, err := e.RequestForQuote(t.Context(), &QuoteRequest{FromCoin: currency.USDC, ToCoin: currency.USD, RequestCoin: currency.USDC, RequestAmount: 5})
		require.NoError(t, err, "RequestForQuote must not error")
		require.NotNil(t, quote, "RequestForQuote must return a quote")
		if quote.QuoteID == "" {
			t.Skip("the account holds too little USDC for a quote it can place")
		}
		quoteID = quote.QuoteID
	}
	result, err := e.PlaceOverTheCounterTradeOrder(t.Context(), quoteID)
	require.NoError(t, err, "PlaceOverTheCounterTradeOrder must not error")
	require.NotNil(t, result, "PlaceOverTheCounterTradeOrder must return the order")
	if mockTests {
		exp := &PlaceOverTheCounterTradeOrderResponse{OrderID: "10002349", CreateTime: types.Time(time.Unix(1641906714, 0)), OrderStatus: OverTheCounterOrderStatusProcess}
		assert.Equal(t, exp, result, "PlaceOverTheCounterTradeOrder should decode every field")
		return
	}
	assert.Equal(t, OverTheCounterOrderStatusProcess, result.OrderStatus, "PlaceOverTheCounterTradeOrder should return an order still being processed")
}

func TestGetOverTheCounterTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOverTheCounterTradeOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOverTheCounterTradeOrder must reject an empty order ID")

	t.Run("decodes the order", func(t *testing.T) {
		t.Parallel()
		body := `{"quoteId":"15848645308","orderId":"10002380","orderStatus":"PROCESS","fromCoin":"SHIB","fromAmount":"10000","toCoin":"KSHIB","toAmount":"10","ratio":"0.001","inverseRatio":"1000","createTime":1641916714}`
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/otc/orders/10002380", url.Values{}, body)
		result, err := ex.GetOverTheCounterTradeOrder(t.Context(), "10002380")
		require.NoError(t, err, "GetOverTheCounterTradeOrder must not error")
		exp := &OverTheCounterTradeOrderResponse{
			QuoteID:      "15848645308",
			OrderID:      "10002380",
			OrderStatus:  OverTheCounterOrderStatusProcess,
			FromCoin:     currency.SHIB,
			FromAmount:   10000,
			ToCoin:       currency.NewCode("KSHIB"),
			ToAmount:     10,
			Ratio:        0.001,
			InverseRatio: 1000,
			CreateTime:   types.Time(time.Unix(1641916714, 0)),
		}
		assert.Equal(t, exp, result, "GetOverTheCounterTradeOrder should decode every field")
	})

	t.Run("escapes the order ID in the path", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/sapi/v1/otc/orders/"+url.PathEscape("1/2?3"), r.URL.EscapedPath(), "GetOverTheCounterTradeOrder should escape the order ID")
			assert.Equal(t, url.Values{}, signedQuery(t, r), "GetOverTheCounterTradeOrder should send no parameters besides the signed ones")
			writeResponse(t, w, `{}`)
		})
		_, err := ex.GetOverTheCounterTradeOrder(t.Context(), "1/2?3")
		require.NoError(t, err, "GetOverTheCounterTradeOrder must not error")
	})

	skipIfCredentialsUnset(t)
	orders, err := e.GetAllOverTheCounterTradeOrders(t.Context(), nil)
	require.NoError(t, err, "GetAllOverTheCounterTradeOrders must not error")
	require.NotNil(t, orders, "GetAllOverTheCounterTradeOrders must return a page of orders")
	if len(orders.Rows) == 0 {
		t.Skip("the account has no over-the-counter orders to query")
	}
	result, err := e.GetOverTheCounterTradeOrder(t.Context(), orders.Rows[0].OrderID)
	require.NoError(t, err, "GetOverTheCounterTradeOrder must not error")
	require.NotNil(t, result, "GetOverTheCounterTradeOrder must return the order")
	if mockTests {
		exp := &OverTheCounterTradeOrderResponse{
			QuoteID:      "4e5446f2cc6f44ab86ab02abf19a2fd2",
			OrderID:      "10002349",
			OrderStatus:  OverTheCounterOrderStatusSuccess,
			FromCoin:     currency.BTC,
			FromAmount:   1,
			ToCoin:       currency.USDT,
			ToAmount:     50550.26,
			Ratio:        50550.26,
			InverseRatio: 0.00001978,
			CreateTime:   types.Time(time.Unix(1641806714, 0)),
		}
		assert.Equal(t, exp, result, "GetOverTheCounterTradeOrder should decode every field")
		return
	}
	assert.Equal(t, orders.Rows[0].OrderID, result.OrderID, "GetOverTheCounterTradeOrder should return the requested order")
}

func TestGetAllOverTheCounterTradeOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1641806714000)
	_, err := e.GetAllOverTheCounterTradeOrders(t.Context(), &OverTheCounterTradeOrdersRequest{StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllOverTheCounterTradeOrders must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *OverTheCounterTradeOrdersRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{
				name: "every filter",
				req:  &OverTheCounterTradeOrdersRequest{OrderID: "10002349", FromCoin: currency.BTC, ToCoin: currency.USDT, StartTime: start, EndTime: start.Add(time.Hour), Page: 2, Limit: 100},
				exp:  url.Values{"orderId": {"10002349"}, "fromCoin": {"BTC"}, "toCoin": {"USDT"}, "startTime": {"1641806714000"}, "endTime": {"1641810314000"}, "page": {"2"}, "limit": {"100"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/otc/orders", tc.exp, `{}`)
				_, err := ex.GetAllOverTheCounterTradeOrders(t.Context(), tc.req)
				require.NoError(t, err, "GetAllOverTheCounterTradeOrders must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAllOverTheCounterTradeOrders(t.Context(), nil)
	require.NoError(t, err, "GetAllOverTheCounterTradeOrders must not error")
	require.NotNil(t, result, "GetAllOverTheCounterTradeOrders must return a page of orders")
	if mockTests {
		exp := &OverTheCounterTradeOrdersResponse{
			Total: 2,
			Rows: []OverTheCounterTradeOrderResponse{
				{
					QuoteID:      "4e5446f2cc6f44ab86ab02abf19a2fd2",
					OrderID:      "10002349",
					OrderStatus:  OverTheCounterOrderStatusSuccess,
					FromCoin:     currency.BTC,
					FromAmount:   1,
					ToCoin:       currency.USDT,
					ToAmount:     50550.26,
					Ratio:        50550.26,
					InverseRatio: 0.00001978,
					CreateTime:   types.Time(time.Unix(1641806714, 0)),
				},
				{
					QuoteID:      "15848645308",
					OrderID:      "10002380",
					OrderStatus:  OverTheCounterOrderStatusProcess,
					FromCoin:     currency.SHIB,
					FromAmount:   10000,
					ToCoin:       currency.NewCode("KSHIB"),
					ToAmount:     10,
					Ratio:        0.001,
					InverseRatio: 1000,
					CreateTime:   types.Time(time.Unix(1641916714, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllOverTheCounterTradeOrders should decode the total and every field of each order")
	}
}

func TestGetAllOneClickBuySellTradeOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1641806714000)
	_, err := e.GetAllOneClickBuySellTradeOrders(t.Context(), &OneClickBuySellTradeOrdersRequest{StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllOneClickBuySellTradeOrders must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *OneClickBuySellTradeOrdersRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{name: "end time alone", req: &OneClickBuySellTradeOrdersRequest{EndTime: start}, exp: url.Values{"endTime": {"1641806714000"}}},
			{
				name: "every filter",
				req:  &OneClickBuySellTradeOrdersRequest{OrderID: "1000238000", StartTime: start, EndTime: start.Add(time.Hour), Page: 2, Limit: 100},
				exp:  url.Values{"orderId": {"1000238000"}, "startTime": {"1641806714000"}, "endTime": {"1641810314000"}, "page": {"2"}, "limit": {"100"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/ocbs/orders", tc.exp, `{}`)
				_, err := ex.GetAllOneClickBuySellTradeOrders(t.Context(), tc.req)
				require.NoError(t, err, "GetAllOneClickBuySellTradeOrders must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAllOneClickBuySellTradeOrders(t.Context(), nil)
	require.NoError(t, err, "GetAllOneClickBuySellTradeOrders must not error")
	require.NotNil(t, result, "GetAllOneClickBuySellTradeOrders must return a page of orders")
	if mockTests {
		exp := &OneClickBuySellTradeOrdersResponse{
			Total: 2,
			DataList: []OneClickBuySellTradeOrder{
				{
					QuoteID:     "4e5446f2cc6f44ab86ab02abf19a7654",
					OrderID:     "10090821212",
					OrderStatus: OverTheCounterOrderStatusSuccess,
					FromCoin:    currency.BTC,
					FromAmount:  1,
					ToCoin:      currency.USD,
					ToAmount:    50550.26,
					FeeCoin:     currency.USD,
					FeeAmount:   0.26,
					Ratio:       50550.26,
					CreateTime:  types.Time(time.Unix(1641806714, 0)),
				},
				{
					QuoteID:     "4e5446f2cc6f44ab86ab02abf19abvd",
					OrderID:     "1000238000",
					OrderStatus: OverTheCounterOrderStatusFail,
					FromCoin:    currency.USD,
					FromAmount:  1000.5,
					ToCoin:      currency.ETH,
					ToAmount:    0.5,
					FeeCoin:     currency.USD,
					FeeAmount:   0.5,
					Ratio:       2000,
					CreateTime:  types.Time(time.Unix(1641916714, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllOneClickBuySellTradeOrders should decode the total and every field of each order")
	}
}
