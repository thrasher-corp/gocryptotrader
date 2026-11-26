package bingx

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetDualCurrencyProducts(t *testing.T) {
	t.Parallel()
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetDualCurrencyProducts(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "GetDualCurrencyProducts should reject a nil request")
	})
	t.Run("every filter sent", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			verifyQuerySignature(t, r.URL.RawQuery)
			q := r.URL.Query()
			q.Del("timestamp")
			q.Del("signature")
			exp := url.Values{
				"pageIndex": {"2"}, "pageSize": {"20"}, "beginDuration": {"1"}, "endDuration": {"7"}, "duration": {"3"},
				"investmentAsset": {"ETH"}, "exerciseAsset": {"USDT"}, "direction": {"PUT"}, "recvWindow": {"5000"},
			}
			assert.Equal(t, exp, q, "every filter should be sent")
			_, _ = io.WriteString(w, `{"code":0,"msg":"success","data":{"direction":"PUT"}}`)
		})
		result, err := ex.GetDualCurrencyProducts(t.Context(), &DualCurrencyProductsRequest{
			PageIndex:       2,
			PageSize:        20,
			BeginDuration:   1,
			EndDuration:     7,
			Duration:        3,
			InvestmentAsset: currency.ETH.Lower(),
			ExerciseAsset:   currency.USDT,
			Direction:       DualCurrencyDirectionPut,
		})
		require.NoError(t, err, "GetDualCurrencyProducts must not error")
		assert.Equal(t, &DualCurrencyProductsResponse{Direction: DualCurrencyDirectionPut}, result, "GetDualCurrencyProducts should decode the reply")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetDualCurrencyProducts(t.Context(), &DualCurrencyProductsRequest{
			PageSize:        10,
			InvestmentAsset: currency.BTC,
			ExerciseAsset:   currency.USDT,
			Direction:       DualCurrencyDirectionCall,
		})
		require.NoError(t, err, "GetDualCurrencyProducts must not error")
		require.NotNil(t, result, "GetDualCurrencyProducts must return a result")
		if !mockTests {
			return
		}
		exp := &DualCurrencyProductsResponse{
			InvestmentAsset: currency.BTC,
			ExerciseAsset:   currency.USDT,
			Direction:       DualCurrencyDirectionCall,
			RecommendVO:     DualCurrencyRecommendedProduct{Duration: 1},
			SubVOs: []DualCurrencyProductGroup{{
				Duration: 1,
				Items:    []DualCurrencyProduct{{Duration: 1}},
			}},
			DurationRangeList: []DualCurrencyDurationRange{{UpperLimit: 7}},
		}
		assert.Equal(t, exp, result, "GetDualCurrencyProducts should decode the documented example")
	})
}

func TestGetDualCurrencyPreOrderQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DualCurrencyPreOrderQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing SKU", arg: &DualCurrencyPreOrderQuoteRequest{StrikePrice: 77900, ProductID: 1234567890}, err: errDualCurrencySKURequired},
		{name: "missing strike price", arg: &DualCurrencyPreOrderQuoteRequest{SKU: "DUAL-BTC-USDT-CALL-8H", ProductID: 1234567890}, err: errDualCurrencyStrikePriceRequired},
		{name: "missing product ID", arg: &DualCurrencyPreOrderQuoteRequest{SKU: "DUAL-BTC-USDT-CALL-8H", StrikePrice: 77900}, err: errDualCurrencyProductIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDualCurrencyPreOrderQuote(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDualCurrencyPreOrderQuote should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetDualCurrencyPreOrderQuote(t.Context(), &DualCurrencyPreOrderQuoteRequest{
			SKU:         "DUAL-BTC-USDT-CALL-8H",
			StrikePrice: 77900,
			ProductID:   1234567890,
		})
		require.NoError(t, err, "GetDualCurrencyPreOrderQuote must not error")
		require.NotNil(t, result, "GetDualCurrencyPreOrderQuote must return a result")
		if !mockTests {
			return
		}
		assert.Equal(t, &DualCurrencyPreOrderQuoteResponse{}, result, "GetDualCurrencyPreOrderQuote should decode the documented example, whose values are all empty")
	})
}

func TestPlaceDualCurrencyOrder(t *testing.T) {
	t.Parallel()
	valid := DualCurrencyOrderRequest{
		InvestmentAsset:     currency.BTC,
		ExerciseAsset:       currency.USDT,
		Amount:              0.01,
		SelectAccounts:      []DualCurrencyAccount{DualCurrencyAccountFund},
		SKU:                 "DUAL-BTC-USDT-CALL-8H",
		ProductID:           1234567890,
		StrikePrice:         77900,
		QuoteID:             "quote_id_from_pre_order",
		SettleTargetAccount: DualCurrencyAccountFund,
	}
	for _, tc := range []struct {
		name   string
		modify func(*DualCurrencyOrderRequest)
		err    error
	}{
		{name: "missing amount", modify: func(r *DualCurrencyOrderRequest) { r.Amount = 0 }, err: order.ErrAmountIsInvalid},
		{name: "negative amount", modify: func(r *DualCurrencyOrderRequest) { r.Amount = -1 }, err: order.ErrAmountIsInvalid},
		{name: "missing SKU", modify: func(r *DualCurrencyOrderRequest) { r.SKU = "" }, err: errDualCurrencySKURequired},
		{name: "missing product ID", modify: func(r *DualCurrencyOrderRequest) { r.ProductID = 0 }, err: errDualCurrencyProductIDRequired},
		{name: "missing strike price", modify: func(r *DualCurrencyOrderRequest) { r.StrikePrice = 0 }, err: errDualCurrencyStrikePriceRequired},
		{name: "missing quote ID", modify: func(r *DualCurrencyOrderRequest) { r.QuoteID = "" }, err: errDualCurrencyQuoteIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			arg := valid
			tc.modify(&arg)
			_, err := e.PlaceDualCurrencyOrder(t.Context(), &arg)
			assert.ErrorIs(t, err, tc.err, "PlaceDualCurrencyOrder should reject the request")
		})
	}
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.PlaceDualCurrencyOrder(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "PlaceDualCurrencyOrder should reject a nil request")
	})
	t.Run("accounts sent as a JSON array", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			assert.Equal(t, "/openApi/wealth/v1/product/dual-currency/order", r.URL.Path, "path should be the place order path")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the body should not error")
			assert.Empty(t, body, "body should be empty")
			verifyQuerySignature(t, r.URL.RawQuery)
			q := r.URL.Query()
			assert.Equal(t, `["FUND","SPOT"]`, q.Get("selectAccounts"), "selectAccounts should be a JSON array")
			for k, v := range map[string]string{
				"amount": "0.01", "sku": "DUAL-BTC-USDT-CALL-8H", "productId": "1234567890", "strikePrice": "77900.5",
				"quoteId": "quote_id_from_pre_order", "settleTargetAccount": "SPOT",
			} {
				assert.Equalf(t, v, q.Get(k), "%s should be sent", k)
			}
			assert.False(t, q.Has("investmentAsset"), "an empty investment asset should not be sent")
			assert.False(t, q.Has("exerciseAsset"), "an empty exercise asset should not be sent")
			_, _ = io.WriteString(w, `{"code":0,"msg":"success","data":{"orderNo":"1","bizCode":0}}`)
		})
		result, err := ex.PlaceDualCurrencyOrder(t.Context(), &DualCurrencyOrderRequest{
			Amount:              0.01,
			SelectAccounts:      []DualCurrencyAccount{DualCurrencyAccountFund, DualCurrencyAccountSpot},
			SKU:                 "DUAL-BTC-USDT-CALL-8H",
			ProductID:           1234567890,
			StrikePrice:         77900.5,
			QuoteID:             "quote_id_from_pre_order",
			SettleTargetAccount: DualCurrencyAccountSpot,
		})
		require.NoError(t, err, "PlaceDualCurrencyOrder must not error")
		assert.Equal(t, &DualCurrencyOrderResponse{OrderNumber: "1"}, result, "PlaceDualCurrencyOrder should decode the reply")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		arg := valid
		result, err := e.PlaceDualCurrencyOrder(t.Context(), &arg)
		require.NoError(t, err, "PlaceDualCurrencyOrder must not error")
		require.NotNil(t, result, "PlaceDualCurrencyOrder must return a result")
		if !mockTests {
			return
		}
		assert.Equal(t, &DualCurrencyOrderResponse{}, result, "PlaceDualCurrencyOrder should decode the documented example, whose values are all empty")
	})
}

func TestGetDualCurrencyPositions(t *testing.T) {
	t.Parallel()
	t.Run("order number sent", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			assert.Equal(t, "1234", r.URL.Query().Get("orderNo"), "the order number should be sent")
			_, _ = io.WriteString(w, `{"code":0,"msg":"success","data":[{"orderNo":"1234","status":7}]}`)
		})
		result, err := ex.GetDualCurrencyPositions(t.Context(), "1234")
		require.NoError(t, err, "GetDualCurrencyPositions must not error")
		assert.Equal(t, []DualCurrencyPositionResponse{{OrderNumber: "1234", Status: DualCurrencyOrderStatusSettled}}, result, "GetDualCurrencyPositions should decode the reply")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetDualCurrencyPositions(t.Context(), "")
		require.NoError(t, err, "GetDualCurrencyPositions must not error")
		if !mockTests {
			return
		}
		exp := []DualCurrencyPositionResponse{{Status: DualCurrencyOrderStatusConfirmingShares}}
		assert.Equal(t, exp, result, "GetDualCurrencyPositions should decode the documented example")
	})
}

func TestGetDualCurrencyOrderRecords(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DualCurrencyOrderRecordsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing page ID", arg: &DualCurrencyOrderRecordsRequest{PageSize: 10}, err: errPageIndexRequired},
		{name: "missing page size", arg: &DualCurrencyOrderRecordsRequest{PageID: 1}, err: errPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDualCurrencyOrderRecords(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDualCurrencyOrderRecords should reject the request")
		})
	}
	t.Run("order number sent", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			for k, v := range map[string]string{"orderNo": "1234", "pageId": "2", "pageSize": "5"} {
				assert.Equalf(t, v, q.Get(k), "%s should be sent", k)
			}
			_, _ = io.WriteString(w, `{"code":0,"msg":"success","data":{"total":6,"pages":2,"current":2,"size":5,"records":[]}}`)
		})
		result, err := ex.GetDualCurrencyOrderRecords(t.Context(), &DualCurrencyOrderRecordsRequest{OrderNumber: "1234", PageID: 2, PageSize: 5})
		require.NoError(t, err, "GetDualCurrencyOrderRecords must not error")
		exp := &DualCurrencyOrderRecordsResponse{Total: 6, Pages: 2, Current: 2, Size: 5, Records: []DualCurrencyOrderRecord{}}
		assert.Equal(t, exp, result, "GetDualCurrencyOrderRecords should decode the reply")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetDualCurrencyOrderRecords(t.Context(), &DualCurrencyOrderRecordsRequest{PageID: 1, PageSize: 10})
		require.NoError(t, err, "GetDualCurrencyOrderRecords must not error")
		require.NotNil(t, result, "GetDualCurrencyOrderRecords must return a result")
		if !mockTests {
			return
		}
		exp := &DualCurrencyOrderRecordsResponse{
			Current: 1,
			Size:    10,
			Records: []DualCurrencyOrderRecord{{Status: DualCurrencyOrderStatusSettled, Duration: 1}},
		}
		assert.Equal(t, exp, result, "GetDualCurrencyOrderRecords should decode the documented example")
	})
}
