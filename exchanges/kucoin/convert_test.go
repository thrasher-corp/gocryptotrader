package kucoin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetConvertSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertSymbol(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertSymbol must reject a nil request")
	_, err = e.GetConvertSymbol(t.Context(), &ConvertSymbolRequest{ToCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetConvertSymbol must reject an empty from currency")
	_, err = e.GetConvertSymbol(t.Context(), &ConvertSymbolRequest{FromCurrency: currency.BTC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetConvertSymbol must reject an empty to currency")

	for _, tc := range []struct {
		name string
		req  *ConvertSymbolRequest
		exp  *ConvertSymbolResponse
	}{
		{
			name: "market",
			req:  &ConvertSymbolRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT},
			exp: &ConvertSymbolResponse{
				FromCurrency:        currency.BTC,
				FromCurrencyMaxSize: 10,
				FromCurrencyMinSize: 0.000001,
				FromCurrencyStep:    0.00000001,
				ToCurrency:          currency.USDT,
				ToCurrencyMaxSize:   900000,
				ToCurrencyMinSize:   0.01,
				ToCurrencyStep:      0.00000001,
			},
		},
		{
			name: "limit",
			req:  &ConvertSymbolRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT, OrderType: "LIMIT"},
			exp: &ConvertSymbolResponse{
				FromCurrency:        currency.BTC,
				FromCurrencyMaxSize: 5,
				FromCurrencyMinSize: 0.00001,
				FromCurrencyStep:    0.00000001,
				ToCurrency:          currency.USDT,
				ToCurrencyMaxSize:   500000,
				ToCurrencyMinSize:   1,
				ToCurrencyStep:      0.00000001,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertSymbol(t.Context(), tc.req)
			if !mockTests && err != nil && strings.Contains(err.Error(), "102431") {
				// KuCoin answers code 102431 (pair not available for Convert) for every pair in some regions
				t.Skipf("KuCoin does not offer this convert pair from here: %s", err)
			}
			require.NoError(t, err, "GetConvertSymbol must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetConvertSymbol should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertSymbol must return a result")
			assert.True(t, result.FromCurrency.Equal(currency.BTC), "GetConvertSymbol should return the from currency")
			assert.Positive(t, result.FromCurrencyMaxSize.Float64(), "GetConvertSymbol should return a maximum size")
		})
	}
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetConvertCurrencies(t.Context())
	require.NoError(t, err, "GetConvertCurrencies must not error")
	if mockTests {
		exp := &ConvertCurrenciesResponse{
			Currencies: []ConvertCurrency{
				{
					Currency:       currency.ETH,
					MaxSize:        70,
					MinSize:        0.000012,
					Step:           0.00000001,
					TradeDirection: "ALL",
				},
				{
					Currency:       currency.BTC,
					MaxSize:        10,
					MinSize:        0.0000012,
					Step:           0.00000001,
					TradeDirection: "ALL",
				},
			},
			USDTCurrencyLimit: []ConvertUSDTCurrencyLimit{
				{
					Currency: currency.BTC,
					MaxSize:  900000,
					MinSize:  0.012,
					Step:     0.00000001,
				},
				{
					Currency: currency.ETH,
					MaxSize:  250000,
					MinSize:  0.12,
					Step:     0.00000001,
				},
			},
		}
		assert.Equal(t, exp, result, "GetConvertCurrencies should decode every field")
		return
	}
	require.NotNil(t, result, "GetConvertCurrencies must return a result")
	require.NotEmpty(t, result.Currencies, "GetConvertCurrencies must return currencies")
	assert.NotEmpty(t, result.USDTCurrencyLimit, "GetConvertCurrencies should return USDT limits")
	assert.False(t, result.Currencies[0].Currency.IsEmpty(), "GetConvertCurrencies should return a currency code")
	assert.Positive(t, result.Currencies[0].MaxSize.Float64(), "GetConvertCurrencies should return a maximum size")
}

func TestGetConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertQuote must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ConvertQuoteRequest
		exp  *ConvertQuoteResponse
	}{
		{
			name: "to currency size",
			req:  &ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT, ToCurrencySize: 5},
			exp: &ConvertQuoteResponse{
				QuoteID:          "d6dvjbt19cpheg28fueg",
				Price:            65547.9811,
				FromCurrencySize: 0.00007628,
				ToCurrencySize:   5,
				ValidUntill:      types.Time(time.UnixMilli(1771829687707)),
			},
		},
		{
			name: "from currency size",
			req:  &ConvertQuoteRequest{FromCurrency: currency.USDT, ToCurrency: currency.BTC, FromCurrencySize: 5},
			exp: &ConvertQuoteResponse{
				QuoteID:          "d6dvjbt19cpheg28fuf0",
				Price:            0.000015256,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00007628,
				ValidUntill:      types.Time(time.UnixMilli(1771829687807)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertQuote(t.Context(), tc.req)
			require.NoError(t, err, "GetConvertQuote must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetConvertQuote should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertQuote must return a result")
			assert.NotEmpty(t, result.QuoteID, "GetConvertQuote should return a quote ID")
		})
	}
}

func TestAddConvertOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AddConvertOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddConvertOrder must reject a nil request")
	_, err = e.AddConvertOrder(t.Context(), &AddConvertOrderRequest{QuoteID: "d6dvkg519cpheg28h490"})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "AddConvertOrder must reject an empty client order ID")
	_, err = e.AddConvertOrder(t.Context(), &AddConvertOrderRequest{ClientOrderID: "client0001"})
	require.ErrorIs(t, err, errQuoteIDRequired, "AddConvertOrder must reject an empty quote ID")

	if !mockTests {
		t.Skip("the quote IDs are the docs' examples, and a convert order on a live quote trades at once, which a test cannot undo")
	}
	for _, tc := range []struct {
		name string
		req  *AddConvertOrderRequest
		exp  *AddConvertOrderResponse
	}{
		{
			name: "account type",
			req:  &AddConvertOrderRequest{ClientOrderID: "client0001", QuoteID: "d6dvkg519cpheg28h490", AccountType: "BOTH"},
			exp:  &AddConvertOrderResponse{OrderID: "12565022", ClientOrderID: "client0001"},
		},
		{
			name: "default account type",
			req:  &AddConvertOrderRequest{ClientOrderID: "client0002", QuoteID: "d6dvjbt19cpheg28fueg"},
			exp:  &AddConvertOrderResponse{OrderID: "12565023", ClientOrderID: "client0002"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddConvertOrder(t.Context(), tc.req)
			require.NoError(t, err, "AddConvertOrder must not error")
			assert.Equal(t, tc.exp, result, "AddConvertOrder should decode every field")
		})
	}
}

func TestGetConvertOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertOrderDetail(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetConvertOrderDetail must reject empty order IDs")

	orderID, clientOrderID := "10721315", "d0k3k4519cplvqk8um80"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		orders, err := e.GetConvertOrderHistory(t.Context(), &ConvertOrdersRequest{})
		require.NoError(t, err, "GetConvertOrderHistory must not error")
		require.NotNil(t, orders, "GetConvertOrderHistory must return a page")
		if len(orders.Items) == 0 {
			t.Skip("the account has no convert orders to look up")
		}
		orderID, clientOrderID = strconv.FormatUint(orders.Items[0].OrderID, 10), orders.Items[0].ClientOrderID
	}
	exp := &ConvertOrderDetailResponse{
		OrderID:          10721315,
		ClientOrderID:    "d0k3k4519cplvqk8um80",
		Status:           "SUCCESS",
		FromCurrency:     currency.USDT,
		ToCurrency:       currency.BTC,
		FromCurrencySize: 5,
		ToCurrencySize:   0.00004749,
		AccountType:      "TRADING",
		Price:            0.000009498,
		OrderTime:        types.Time(time.UnixMilli(1748939927000)),
	}
	for _, tc := range []struct {
		name, orderID, clientOrderID string
	}{
		{name: "order ID", orderID: orderID},
		{name: "client order ID", clientOrderID: clientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertOrderDetail(t.Context(), tc.orderID, tc.clientOrderID)
			require.NoError(t, err, "GetConvertOrderDetail must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetConvertOrderDetail should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertOrderDetail must return a result")
			assert.Equal(t, orderID, strconv.FormatUint(result.OrderID, 10), "GetConvertOrderDetail should return the order asked for")
		})
	}
}

func TestGetConvertOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertOrderHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &ConvertOrderHistoryResponse{
		CurrentPage: 1,
		PageSize:    20,
		TotalNumber: 1,
		TotalPage:   1,
		Items: []ConvertOrder{
			{
				OrderID:          10721315,
				ClientOrderID:    "d0k3k4519cplvqk8um80",
				Status:           "SUCCESS",
				FromCurrency:     currency.USDT,
				ToCurrency:       currency.BTC,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00004749,
				AccountType:      "TRADING",
				Price:            0.000009498,
				OrderTime:        types.Time(time.UnixMilli(1748939927000)),
			},
		},
	}
	for _, tc := range []struct {
		name string
		req  *ConvertOrdersRequest
	}{
		{name: "no filter", req: &ConvertOrdersRequest{}},
		{name: "every filter", req: &ConvertOrdersRequest{
			StartAt:  time.UnixMilli(1748900000000),
			EndAt:    time.UnixMilli(1749100000000),
			Page:     1,
			PageSize: 20,
			Status:   "SUCCESS",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetConvertOrderHistory must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetConvertOrderHistory should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertOrderHistory must return a result")
			assert.NotZero(t, result.PageSize, "GetConvertOrderHistory should return a page size")
		})
	}
}

func TestAddConvertLimitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AddConvertLimitOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddConvertLimitOrder must reject a nil request")
	arg := &AddConvertLimitOrderRequest{}
	_, err = e.AddConvertLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "AddConvertLimitOrder must reject an empty client order ID")
	arg.ClientOrderID = "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9"
	_, err = e.AddConvertLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AddConvertLimitOrder must reject an empty from currency")
	arg.FromCurrency = currency.USDT
	_, err = e.AddConvertLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AddConvertLimitOrder must reject an empty to currency")
	arg.ToCurrency = currency.BTC
	_, err = e.AddConvertLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AddConvertLimitOrder must reject a zero from currency size")
	arg.FromCurrencySize = 5
	_, err = e.AddConvertLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AddConvertLimitOrder must reject a zero to currency size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddConvertLimitOrderRequest
		exp  *AddConvertOrderResponse
	}{
		{
			name: "default account type",
			req: &AddConvertLimitOrderRequest{
				ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9",
				FromCurrency:     currency.USDT,
				ToCurrency:       currency.BTC,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00004762,
			},
			exp: &AddConvertOrderResponse{OrderID: "683ff823e66ccd0007ed881b", ClientOrderID: "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9"},
		},
		{
			name: "funding account",
			req: &AddConvertLimitOrderRequest{
				ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc012",
				AccountType:      "FUNDING",
				FromCurrency:     currency.USDT,
				ToCurrency:       currency.BTC,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00004761,
			},
			exp: &AddConvertOrderResponse{OrderID: "683ff9f39f47d70006dacb0f", ClientOrderID: "8dbfc2f1-f37e-4346-83be-b63ad11dc012"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddConvertLimitOrder(t.Context(), tc.req)
			require.NoError(t, err, "AddConvertLimitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddConvertLimitOrder should decode every field")
				return
			}
			require.NotNil(t, result, "AddConvertLimitOrder must return a result")
			assert.NotEmpty(t, result.OrderID, "AddConvertLimitOrder should return an order ID")
		})
	}
}

func TestGetConvertLimitQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertLimitQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertLimitQuote must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertLimitQuote(t.Context(), &ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT, FromCurrencySize: 0.001})
	require.NoError(t, err, "GetConvertLimitQuote must not error")
	if mockTests {
		exp := &ConvertLimitQuoteResponse{
			Price:       105019.9538,
			ValidUntill: types.Time(time.UnixMilli(1749023723018)),
		}
		assert.Equal(t, exp, result, "GetConvertLimitQuote should decode every field")
		return
	}
	require.NotNil(t, result, "GetConvertLimitQuote must return a result")
	assert.Positive(t, result.Price.Float64(), "GetConvertLimitQuote should return a protection price")
}

func TestGetConvertLimitOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertLimitOrderDetail(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetConvertLimitOrderDetail must reject empty order IDs")

	orderID, clientOrderID := "683ff823e66ccd0007ed881b", "8dbfc2f1-f37e-4346-83be-b63ad11dc012"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		orders, err := e.GetConvertLimitOrders(t.Context(), &ConvertOrdersRequest{})
		require.NoError(t, err, "GetConvertLimitOrders must not error")
		require.NotNil(t, orders, "GetConvertLimitOrders must return a page")
		if len(orders.Items) == 0 {
			t.Skip("the account has no convert limit orders to look up")
		}
		orderID, clientOrderID = orders.Items[0].OrderID, orders.Items[0].ClientOrderID
	}
	for _, tc := range []struct {
		name, orderID, clientOrderID string
		exp                          *ConvertLimitOrderDetailResponse
	}{
		{
			name:    "order ID",
			orderID: orderID,
			exp: &ConvertLimitOrderDetailResponse{
				OrderID:          "683ff823e66ccd0007ed881b",
				ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9",
				Status:           "CANCELLED",
				FromCurrency:     currency.USDT,
				ToCurrency:       currency.BTC,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00004762,
				AccountType:      "BOTH",
				Price:            0.000009524,
				OrderTime:        types.Time(time.UnixMilli(1749022755000)),
				ExpiryTime:       types.Time(time.UnixMilli(1749627556000)),
				CancelTime:       types.Time(time.UnixMilli(1749022900000)),
				CancelType:       1,
			},
		},
		{
			name:          "client order ID",
			clientOrderID: clientOrderID,
			exp: &ConvertLimitOrderDetailResponse{
				OrderID:          "683ff9f39f47d70006dacb0f",
				ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc012",
				Status:           "SUCCESS",
				FromCurrency:     currency.USDT,
				ToCurrency:       currency.BTC,
				FromCurrencySize: 5,
				ToCurrencySize:   0.00004761,
				AccountType:      "FUNDING",
				Price:            0.000009522,
				OrderTime:        types.Time(time.UnixMilli(1749023219000)),
				ExpiryTime:       types.Time(time.UnixMilli(1749628020000)),
				FilledTime:       types.Time(time.UnixMilli(1749045700000)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertLimitOrderDetail(t.Context(), tc.orderID, tc.clientOrderID)
			require.NoError(t, err, "GetConvertLimitOrderDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetConvertLimitOrderDetail should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertLimitOrderDetail must return a result")
			assert.Equal(t, orderID, result.OrderID, "GetConvertLimitOrderDetail should return the order asked for")
		})
	}
}

func TestGetConvertLimitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertLimitOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertLimitOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	filled := ConvertLimitOrder{
		OrderID:          "683ff9f39f47d70006dacb0f",
		ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc012",
		Status:           "SUCCESS",
		FromCurrency:     currency.USDT,
		ToCurrency:       currency.BTC,
		FromCurrencySize: 5,
		ToCurrencySize:   0.00004761,
		AccountType:      "FUNDING",
		Price:            0.000009522,
		OrderTime:        types.Time(time.UnixMilli(1749023219000)),
		ExpiryTime:       types.Time(time.UnixMilli(1749628020000)),
		FilledTime:       types.Time(time.UnixMilli(1749045700000)),
	}
	cancelled := ConvertLimitOrder{
		OrderID:          "683ff823e66ccd0007ed881b",
		ClientOrderID:    "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9",
		Status:           "CANCELLED",
		FromCurrency:     currency.USDT,
		ToCurrency:       currency.BTC,
		FromCurrencySize: 5,
		ToCurrencySize:   0.00004762,
		AccountType:      "BOTH",
		Price:            0.000009524,
		OrderTime:        types.Time(time.UnixMilli(1749022755000)),
		ExpiryTime:       types.Time(time.UnixMilli(1749627556000)),
		CancelTime:       types.Time(time.UnixMilli(1749022900000)),
		CancelType:       1,
	}
	for _, tc := range []struct {
		name string
		req  *ConvertOrdersRequest
		exp  *ConvertLimitOrdersResponse
	}{
		{
			name: "no filter",
			req:  &ConvertOrdersRequest{},
			exp:  &ConvertLimitOrdersResponse{CurrentPage: 1, PageSize: 20, TotalNumber: 2, TotalPage: 1, Items: []ConvertLimitOrder{filled, cancelled}},
		},
		{
			name: "every filter",
			req: &ConvertOrdersRequest{
				StartAt:  time.UnixMilli(1749000000000),
				EndAt:    time.UnixMilli(1749100000000),
				Page:     1,
				PageSize: 20,
				Status:   "CANCELLED",
			},
			exp: &ConvertLimitOrdersResponse{CurrentPage: 1, PageSize: 20, TotalNumber: 1, TotalPage: 1, Items: []ConvertLimitOrder{cancelled}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertLimitOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetConvertLimitOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetConvertLimitOrders should decode every field")
				return
			}
			require.NotNil(t, result, "GetConvertLimitOrders must return a result")
			assert.NotZero(t, result.PageSize, "GetConvertLimitOrders should return a page size")
		})
	}
}

func TestCancelConvertLimitOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelConvertLimitOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelConvertLimitOrder must reject empty order IDs")

	for _, tc := range []struct {
		name, orderID, clientOrderID, body string
	}{
		{name: "order ID", orderID: "683ff823e66ccd0007ed881b", body: `{"orderId":"683ff823e66ccd0007ed881b"}`},
		{name: "client order ID", clientOrderID: "8dbfc2f1-f37e-4346-83be-b63ad11dc0d9", body: `{"clientOrderId":"8dbfc2f1-f37e-4346-83be-b63ad11dc0d9"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The recorded mock matches DELETE requests by their query alone, so a local server checks the body
			sent := sentRequest(t, func(ex *Exchange) error {
				return ex.CancelConvertLimitOrder(t.Context(), tc.orderID, tc.clientOrderID)
			})
			assert.Equal(t, http.MethodDelete, sent.method, "CancelConvertLimitOrder should send a DELETE request")
			assert.Equal(t, "/api/v1/convert/limit/order/cancel", sent.requestURI, "CancelConvertLimitOrder should send no query")
			assert.JSONEq(t, tc.body, sent.body, "CancelConvertLimitOrder should send the order ID it cancels in the body")

			if !mockTests {
				t.Skip("the order IDs are the docs' examples, and cancelling one of the account's real orders is not safe in a test")
			}
			err := e.CancelConvertLimitOrder(t.Context(), tc.orderID, tc.clientOrderID)
			assert.NoError(t, err, "CancelConvertLimitOrder should not error")
		})
	}
}

func TestConvertQuoteParams(t *testing.T) {
	t.Parallel()
	_, err := convertQuoteParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "convertQuoteParams must reject a nil request")
	_, err = convertQuoteParams(&ConvertQuoteRequest{ToCurrency: currency.USDT, ToCurrencySize: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "convertQuoteParams must reject an empty from currency")
	_, err = convertQuoteParams(&ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrencySize: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "convertQuoteParams must reject an empty to currency")
	_, err = convertQuoteParams(&ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT})
	require.ErrorIs(t, err, errOneCurrencySizeRequired, "convertQuoteParams must reject a request without a size")
	_, err = convertQuoteParams(&ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT, FromCurrencySize: 0.001, ToCurrencySize: 5})
	require.ErrorIs(t, err, errOneCurrencySizeRequired, "convertQuoteParams must reject a request with both sizes")

	for _, tc := range []struct {
		name string
		req  *ConvertQuoteRequest
		exp  url.Values
	}{
		{
			name: "from currency size",
			req:  &ConvertQuoteRequest{FromCurrency: currency.NewCode("btc"), ToCurrency: currency.USDT, FromCurrencySize: 0.00000123},
			exp:  url.Values{"fromCurrency": {"BTC"}, "toCurrency": {"USDT"}, "fromCurrencySize": {"0.00000123"}},
		},
		{
			name: "to currency size",
			req:  &ConvertQuoteRequest{FromCurrency: currency.BTC, ToCurrency: currency.USDT, ToCurrencySize: 5},
			exp:  url.Values{"fromCurrency": {"BTC"}, "toCurrency": {"USDT"}, "toCurrencySize": {"5"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := convertQuoteParams(tc.req)
			require.NoError(t, err, "convertQuoteParams must not error")
			assert.Equal(t, tc.exp, params, "convertQuoteParams should set the documented parameters")
		})
	}
}

func TestConvertOrderIDParams(t *testing.T) {
	t.Parallel()
	_, err := convertOrderIDParams("", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "convertOrderIDParams must reject empty order IDs")

	for _, tc := range []struct {
		name, orderID, clientOrderID string
		exp                          url.Values
	}{
		{name: "order ID", orderID: "10721315", exp: url.Values{"orderId": {"10721315"}}},
		{name: "client order ID", clientOrderID: "client0001", exp: url.Values{"clientOrderId": {"client0001"}}},
		{name: "both", orderID: "10721315", clientOrderID: "client0001", exp: url.Values{"orderId": {"10721315"}, "clientOrderId": {"client0001"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := convertOrderIDParams(tc.orderID, tc.clientOrderID)
			require.NoError(t, err, "convertOrderIDParams must not error")
			assert.Equal(t, tc.exp, params, "convertOrderIDParams should set the IDs given")
		})
	}
}

func TestConvertOrdersParams(t *testing.T) {
	t.Parallel()
	_, err := convertOrdersParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "convertOrdersParams must reject a nil request")
	_, err = convertOrdersParams(&ConvertOrdersRequest{StartAt: time.UnixMilli(1749100000000), EndAt: time.UnixMilli(1749000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "convertOrdersParams must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *ConvertOrdersRequest
		exp  url.Values
	}{
		{name: "no filter", req: &ConvertOrdersRequest{}, exp: url.Values{}},
		{
			name: "every filter",
			req: &ConvertOrdersRequest{
				StartAt:  time.UnixMilli(1749000000000),
				EndAt:    time.UnixMilli(1749100000000),
				Page:     2,
				PageSize: 50,
				Status:   "OPEN",
			},
			exp: url.Values{
				"startAt":  {"1749000000000"},
				"endAt":    {"1749100000000"},
				"page":     {"2"},
				"pageSize": {"50"},
				"status":   {"OPEN"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := convertOrdersParams(tc.req)
			require.NoError(t, err, "convertOrdersParams must not error")
			assert.Equal(t, tc.exp, params, "convertOrdersParams should set the filters given")
		})
	}
}
