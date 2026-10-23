package kucoin

import (
	"context"
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

func TestPlaceHFOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceHFOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceHFOrder must reject a nil request")
	_, err = e.PlaceHFOrder(t.Context(), &PlaceHFOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "PlaceHFOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceHFOrder(t.Context(), &PlaceHFOrderRequest{Type: "limit", Symbol: "BTC-USDT", Side: "buy", Price: 50000, Size: 0.00001, ClientOrderID: "5c52e11203aa677f33e493fb", Remark: "order remarks"})
	require.NoError(t, err, "PlaceHFOrder must not error")
	require.NotNil(t, result, "PlaceHFOrder must return a response")
	if mockTests {
		exp := &PlaceHFOrderResponse{
			OrderID:       "670fd33bf9406e0007ab3945",
			ClientOrderID: "5c52e11203aa677f33e493fb",
		}
		assert.Equal(t, exp, result, "PlaceHFOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceHFOrder should return the order ID")
}

func TestPlaceHFOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceHFOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &PlaceHFOrderRequest{}, err: currency.ErrSymbolStringEmpty},
		{name: "no side", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT"}, err: order.ErrSideIsInvalid},
		{name: "no type", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy"}, err: order.ErrTypeIsInvalid},
		{name: "max time window without client timestamp", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy", Type: "market", Funds: 1, AllowMaxTimeWindow: 1000}, err: errClientTimestampRequired},
		{name: "limit without price", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy", Type: "limit", Size: 0.00001}, err: limits.ErrPriceBelowMin},
		{name: "market without size or funds", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy", Type: "market"}, err: errSizeOrFundIsRequired},
		{name: "valid limit", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy", Type: "limit", Price: 50000, Size: 0.00001}},
		{name: "valid with max time window", req: &PlaceHFOrderRequest{Symbol: "BTC-USDT", Side: "buy", Type: "market", Funds: 1, AllowMaxTimeWindow: 1000, ClientTimestamp: time.UnixMilli(1729174140586)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestValidateOrderAmounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		orderType          string
		price, size, funds float64
		err                error
	}{
		{name: "limit without price", orderType: "limit", size: 0.00001, err: limits.ErrPriceBelowMin},
		{name: "limit without size", orderType: "limit", price: 50000, err: limits.ErrAmountBelowMin},
		{name: "limit", orderType: "limit", price: 50000, size: 0.00001},
		{name: "market with size and funds", orderType: "market", size: 0.00001, funds: 1, err: errSizeAndFundsBothSet},
		{name: "market without size or funds", orderType: "market", err: errSizeOrFundIsRequired},
		{name: "market with size", orderType: "market", size: 0.00001},
		{name: "market with funds", orderType: "market", funds: 1},
		{name: "other types are left to KuCoin", orderType: "limit_stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateOrderAmounts(tc.orderType, tc.price, tc.size, tc.funds), tc.err, "validateOrderAmounts should return the expected error")
		})
	}
}

func TestPlaceHFOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*PlaceHFOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&PlaceHFOrderRequest{}).Payload()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "Payload must validate the request")

	r := &PlaceHFOrderRequest{
		ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Remark: "order remarks",
		SelfTradePrevention: "CN", Price: 50000, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, Tags: "order tags",
		CancelAfter: 600, Funds: 0.5, AllowMaxTimeWindow: 1000, ClientTimestamp: time.UnixMilli(1729174140586),
	}
	exp := map[string]any{
		"clientOid": "5c52e11203aa677f33e493fb", "side": "buy", "symbol": "BTC-USDT", "type": "limit",
		"remark": "order remarks", "stp": "CN", "price": "50000", "size": "0.00001", "timeInForce": "GTT",
		"postOnly": true, "tags": "order tags", "cancelAfter": uint64(600), "funds": "0.5",
		"allowMaxTimeWindow": uint64(1000), "clientTimestamp": int64(1729174140586),
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")
}

func TestPlaceHFOrderSync(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceHFOrderSync(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceHFOrderSync must reject a nil request")
	_, err = e.PlaceHFOrderSync(t.Context(), &PlaceHFOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "PlaceHFOrderSync must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceHFOrderSync(t.Context(), &PlaceHFOrderRequest{Type: "limit", Symbol: "BTC-USDT", Side: "buy", Price: 50000, Size: 0.00003, ClientOrderID: "5c52e11203aa677f33e493fb", SelfTradePrevention: "DC"})
	require.NoError(t, err, "PlaceHFOrderSync must not error")
	require.NotNil(t, result, "PlaceHFOrderSync must return a response")
	if mockTests {
		exp := &PlaceHFOrderSyncResponse{
			OrderID:       "67111a7cb7cbdf000703e1f6",
			ClientOrderID: "5c52e11203aa677f33e493fb",
			OrderTime:     types.Time(time.UnixMilli(1729174140586)),
			OriginSize:    0.00003,
			DealSize:      0.00001,
			RemainSize:    0.00001,
			CanceledSize:  0.00001,
			Status:        "open",
			MatchTime:     types.Time(time.UnixMilli(1729174140588)),
		}
		assert.Equal(t, exp, result, "PlaceHFOrderSync should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceHFOrderSync should return the order ID")
}

func TestPlaceHFOrderTest(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceHFOrderTest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceHFOrderTest must reject a nil request")
	_, err = e.PlaceHFOrderTest(t.Context(), &PlaceHFOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "PlaceHFOrderTest must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PlaceHFOrderTest(t.Context(), &PlaceHFOrderRequest{Type: "market", Symbol: "BTC-USDT", Side: "buy", Funds: 1, ClientOrderID: "5c52e11203aa677f33e493fc", Remark: "order remarks"})
	require.NoError(t, err, "PlaceHFOrderTest must not error")
	require.NotNil(t, result, "PlaceHFOrderTest must return a response")
	if mockTests {
		exp := &PlaceHFOrderResponse{
			OrderID:       "670fd33bf9406e0007ab3946",
			ClientOrderID: "5c52e11203aa677f33e493fc",
		}
		assert.Equal(t, exp, result, "PlaceHFOrderTest should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceHFOrderTest should return the order ID")
}

var testBatchHFOrders = []PlaceHFOrderRequest{
	{ClientOrderID: "5c52e11203aa677f33e493fd", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Price: 30000, Size: 0.00001},
	{ClientOrderID: "5c52e11203aa677f33e493fe", Symbol: "ETH-USDT", Type: "limit", Side: "sell", Price: 2000, Size: 0.00001},
}

func TestBatchPlaceHFOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchPlaceHFOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "BatchPlaceHFOrders must reject an empty order list")
	_, err = e.BatchPlaceHFOrders(t.Context(), []PlaceHFOrderRequest{{}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "BatchPlaceHFOrders must validate every order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchPlaceHFOrders(t.Context(), testBatchHFOrders)
	require.NoError(t, err, "BatchPlaceHFOrders must not error")
	if mockTests {
		exp := []BatchPlacedHFOrderResponse{
			{
				OrderID:       "6710d8336afcdb0007319c27",
				ClientOrderID: "5c52e11203aa677f33e493fd",
				Success:       true,
			},
			{
				FailMessage: "The order funds should more then 0.1 USDT.",
			},
		}
		assert.Equal(t, exp, result, "BatchPlaceHFOrders should decode every order's outcome")
		return
	}
	assert.Len(t, result, len(testBatchHFOrders), "BatchPlaceHFOrders should return an outcome per order")
}

func TestBatchPlaceHFOrdersSync(t *testing.T) {
	t.Parallel()
	_, err := e.BatchPlaceHFOrdersSync(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "BatchPlaceHFOrdersSync must reject an empty order list")
	_, err = e.BatchPlaceHFOrdersSync(t.Context(), []PlaceHFOrderRequest{{}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "BatchPlaceHFOrdersSync must validate every order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchPlaceHFOrdersSync(t.Context(), testBatchHFOrders)
	require.NoError(t, err, "BatchPlaceHFOrdersSync must not error")
	if mockTests {
		exp := []BatchPlacedHFOrderSyncResponse{
			{
				OrderID:       "6711195e5584bc0007bd5aef",
				ClientOrderID: "5c52e11203aa677f33e493fd",
				OrderTime:     types.Time(time.UnixMilli(1729173854299)),
				OriginSize:    0.00001,
				DealSize:      0.000004,
				RemainSize:    0.000003,
				CanceledSize:  0.000003,
				Status:        "open",
				MatchTime:     types.Time(time.UnixMilli(1729173854326)),
				Success:       true,
			},
			{
				FailMessage: "The order funds should more then 0.1 USDT.",
			},
		}
		assert.Equal(t, exp, result, "BatchPlaceHFOrdersSync should decode every order's outcome")
		return
	}
	assert.Len(t, result, len(testBatchHFOrders), "BatchPlaceHFOrdersSync should return an outcome per order")
}

func TestHFOrderListBody(t *testing.T) {
	t.Parallel()
	_, err := hfOrderListBody(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "hfOrderListBody must reject an empty order list")
	_, err = hfOrderListBody([]PlaceHFOrderRequest{testBatchHFOrders[0], {Symbol: "BTC-USDT", Side: "buy", Type: "limit", Price: 30000}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "hfOrderListBody must validate every order")
	assert.ErrorContains(t, err, "order 1", "hfOrderListBody should name the invalid order")

	body, err := hfOrderListBody(testBatchHFOrders)
	require.NoError(t, err, "hfOrderListBody must not error")
	exp := map[string]any{
		"orderList": []map[string]any{
			{"clientOid": "5c52e11203aa677f33e493fd", "symbol": "BTC-USDT", "type": "limit", "side": "buy", "price": "30000", "size": "0.00001"},
			{"clientOid": "5c52e11203aa677f33e493fe", "symbol": "ETH-USDT", "type": "limit", "side": "sell", "price": "2000", "size": "0.00001"},
		},
	}
	assert.Equal(t, exp, body, "hfOrderListBody should list every order's body")
}

func TestCancelHFOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHFOrderByOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelHFOrderByOrderID must reject an empty order ID")
	_, err = e.CancelHFOrderByOrderID(t.Context(), "671124f9365ccb00073debd4", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelHFOrderByOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelHFOrderByOrderID(t.Context(), "671124f9365ccb00073debd4", "BTC-USDT")
	require.NoError(t, err, "CancelHFOrderByOrderID must not error")
	if mockTests {
		exp := &OrderIDResponse{
			OrderID: "671124f9365ccb00073debd4",
		}
		assert.Equal(t, exp, result, "CancelHFOrderByOrderID should decode every field")
	}
}

func TestCancelHFOrderByOrderIDSync(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHFOrderByOrderIDSync(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelHFOrderByOrderIDSync must reject an empty order ID")
	_, err = e.CancelHFOrderByOrderIDSync(t.Context(), "671128ee365ccb0007534d45", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelHFOrderByOrderIDSync must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelHFOrderByOrderIDSync(t.Context(), "671128ee365ccb0007534d45", "BTC-USDT")
	require.NoError(t, err, "CancelHFOrderByOrderIDSync must not error")
	require.NotNil(t, result, "CancelHFOrderByOrderIDSync must return a response")
	if mockTests {
		exp := &CancelHFOrderByOrderIDSyncResponse{
			OrderID:      "671128ee365ccb0007534d45",
			OriginSize:   0.00001,
			DealSize:     0.000004,
			RemainSize:   0.000003,
			CanceledSize: 0.000003,
			Status:       "done",
		}
		assert.Equal(t, exp, result, "CancelHFOrderByOrderIDSync should decode every field")
		return
	}
	assert.Equal(t, "671128ee365ccb0007534d45", result.OrderID, "CancelHFOrderByOrderIDSync should return the order ID")
}

func TestCancelHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHFOrderByClientOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelHFOrderByClientOrderID must reject an empty client order ID")
	_, err = e.CancelHFOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelHFOrderByClientOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelHFOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb", "BTC-USDT")
	require.NoError(t, err, "CancelHFOrderByClientOrderID must not error")
	if mockTests {
		exp := &ClientOrderIDResponse{
			ClientOrderID: "5c52e11203aa677f33e493fb",
		}
		assert.Equal(t, exp, result, "CancelHFOrderByClientOrderID should decode every field")
	}
}

func TestCancelHFOrderByClientOrderIDSync(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHFOrderByClientOrderIDSync(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelHFOrderByClientOrderIDSync must reject an empty client order ID")
	_, err = e.CancelHFOrderByClientOrderIDSync(t.Context(), "5c52e11203aa677f33e493fb", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelHFOrderByClientOrderIDSync must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelHFOrderByClientOrderIDSync(t.Context(), "5c52e11203aa677f33e493fb", "BTC-USDT")
	require.NoError(t, err, "CancelHFOrderByClientOrderIDSync must not error")
	require.NotNil(t, result, "CancelHFOrderByClientOrderIDSync must return a response")
	if mockTests {
		exp := &CancelHFOrderByClientOrderIDSyncResponse{
			ClientOrderID: "5c52e11203aa677f33e493fb",
			OriginSize:    0.00001,
			DealSize:      0.000004,
			RemainSize:    0.000003,
			CanceledSize:  0.000003,
			Status:        "done",
		}
		assert.Equal(t, exp, result, "CancelHFOrderByClientOrderIDSync should decode every field")
		return
	}
	assert.Equal(t, "5c52e11203aa677f33e493fb", result.ClientOrderID, "CancelHFOrderByClientOrderIDSync should return the client order ID")
}

func TestCancelPartialHFOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPartialHFOrder(t.Context(), "", "BTC-USDT", 0.00001)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPartialHFOrder must reject an empty order ID")
	_, err = e.CancelPartialHFOrder(t.Context(), "6711f73c1ef16c000717bb31", "", 0.00001)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelPartialHFOrder must reject an empty symbol")
	_, err = e.CancelPartialHFOrder(t.Context(), "6711f73c1ef16c000717bb31", "BTC-USDT", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CancelPartialHFOrder must reject a zero cancel size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelPartialHFOrder(t.Context(), "6711f73c1ef16c000717bb31", "BTC-USDT", 0.00001)
	require.NoError(t, err, "CancelPartialHFOrder must not error")
	if mockTests {
		exp := &CancelPartialHFOrderResponse{
			OrderID:    "6711f73c1ef16c000717bb31",
			CancelSize: 0.00001,
		}
		assert.Equal(t, exp, result, "CancelPartialHFOrder should decode every field")
	}
}

func TestCancelAllHFOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllHFOrdersBySymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelAllHFOrdersBySymbol must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllHFOrdersBySymbol(t.Context(), "BTC-USDT")
	require.NoError(t, err, "CancelAllHFOrdersBySymbol must not error")
	assert.Equal(t, "success", result, "CancelAllHFOrdersBySymbol should return KuCoin's acknowledgement")
}

func TestCancelAllHFOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllHFOrders(t.Context())
	require.NoError(t, err, "CancelAllHFOrders must not error")
	require.NotNil(t, result, "CancelAllHFOrders must return a response")
	if mockTests {
		exp := &CancelAllHFOrdersResponse{
			SucceedSymbols: []string{
				"ETH-USDT",
				"BTC-USDT",
			},
			FailedSymbols: []FailedHFOrderCancellationInfo{
				{
					Symbol: "KCS-USDT",
					Error:  "Order cancellation failed, please try again later.",
				},
			},
		}
		assert.Equal(t, exp, result, "CancelAllHFOrders should decode every field")
		return
	}
	assert.Empty(t, result.FailedSymbols, "CancelAllHFOrders should cancel every symbol's orders")
}

func TestModifyHFOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ModifyHFOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &ModifyHFOrderRequest{}, err: currency.ErrSymbolStringEmpty},
		{name: "no order ID", req: &ModifyHFOrderRequest{Symbol: "BTC-USDT"}, err: order.ErrOrderIDNotSet},
		{name: "no new price or size", req: &ModifyHFOrderRequest{Symbol: "BTC-USDT", OrderID: "670fd33bf9406e0007ab3945"}, err: errNewPriceOrNewSizeRequired},
	} {
		_, err := e.ModifyHFOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ModifyHFOrder must reject a request with %s", tc.name)
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyHFOrder(t.Context(), &ModifyHFOrderRequest{Symbol: "BTC-USDT", OrderID: "670fd33bf9406e0007ab3945", NewPrice: 30000, NewSize: 0.0001})
		return err
	})
	assert.JSONEq(t, `{"symbol":"BTC-USDT","orderId":"670fd33bf9406e0007ab3945","newPrice":"30000","newSize":"0.0001"}`, sent.body, "ModifyHFOrder should send the new price and size as the strings KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *ModifyHFOrderRequest
		exp  *ModifyHFOrderResponse
	}{
		{
			name: "by order ID",
			req:  &ModifyHFOrderRequest{Symbol: "BTC-USDT", OrderID: "670fd33bf9406e0007ab3945", NewPrice: 30000, NewSize: 0.0001},
			exp: &ModifyHFOrderResponse{
				NewOrderID:    "67112258f9406e0007408827",
				ClientOrderID: "5c52e11203aa677f33e493fb",
			},
		},
		{
			name: "by client order ID",
			req:  &ModifyHFOrderRequest{Symbol: "BTC-USDT", ClientOrderID: "5c52e11203aa677f33e493fb", NewSize: 0.0002},
			exp: &ModifyHFOrderResponse{
				NewOrderID:    "67112258f9406e0007408828",
				ClientOrderID: "5c52e11203aa677f33e493fb",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ModifyHFOrder(t.Context(), tc.req)
			require.NoError(t, err, "ModifyHFOrder must not error")
			require.NotNil(t, result, "ModifyHFOrder must return a response")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ModifyHFOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.NewOrderID, "ModifyHFOrder should return the new order ID")
		})
	}
}

func TestGetHFOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetHFOrderByOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetHFOrderByOrderID must reject an empty order ID")
	_, err = e.GetHFOrderByOrderID(t.Context(), "68a3e81d2a090d00070575f7", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetHFOrderByOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHFOrderByOrderID(t.Context(), "68a3e81d2a090d00070575f7", "BTC-USDT")
	require.NoError(t, err, "GetHFOrderByOrderID must not error")
	if mockTests {
		exp := &HFOrderDetailResponse{
			ID:                  "68a3e81d2a090d00070575f7",
			Symbol:              "BTC-USDT",
			OperationType:       "DEAL",
			Type:                "limit",
			Side:                "buy",
			Price:               50000,
			Size:                0.00003,
			Funds:               1.5,
			DealSize:            0.00001,
			DealFunds:           0.5,
			Fee:                 0.0005,
			FeeCurrency:         currency.USDT,
			SelfTradePrevention: "DC",
			TimeInForce:         "GTT",
			PostOnly:            true,
			Hidden:              true,
			Iceberg:             true,
			VisibleSize:         0.00001,
			CancelAfter:         3600,
			Channel:             "API",
			ClientOrderID:       "5c52e11203aa677f33e493fb",
			Remark:              "order remarks",
			Tags:                "order tags",
			CancelExist:         true,
			CreatedAt:           types.Time(time.UnixMilli(1755572253338)),
			LastUpdatedAt:       types.Time(time.UnixMilli(1755572263485)),
			TradeType:           "TRADE",
			InOrderBook:         true,
			CancelledSize:       0.00001,
			CancelledFunds:      0.5,
			RemainSize:          0.00001,
			RemainFunds:         0.5,
			Tax:                 0.00005,
			Active:              true,
			CancelReason:        5,
		}
		assert.Equal(t, exp, result, "GetHFOrderByOrderID should decode every field")

		_, err = e.GetHFOrderByOrderID(t.Context(), "68a3e81d2a090d00070575f8", "BTC-USDT")
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetHFOrderByOrderID should report an order KuCoin returns no data for")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = e.GetHFOrderByOrderID(ctx, "68a3e81d2a090d00070575f8", "BTC-USDT")
		assert.ErrorIs(t, err, context.Canceled, "GetHFOrderByOrderID should return the request error")
		return
	}
	assert.Equal(t, "68a3e81d2a090d00070575f7", result.ID, "GetHFOrderByOrderID should return the order")
}

func TestGetHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetHFOrderByClientOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetHFOrderByClientOrderID must reject an empty client order ID")
	_, err = e.GetHFOrderByClientOrderID(t.Context(), "9a1840ed0b5746459ede8e6f232daf2d", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetHFOrderByClientOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHFOrderByClientOrderID(t.Context(), "9a1840ed0b5746459ede8e6f232daf2d", "BTC-USDT")
	require.NoError(t, err, "GetHFOrderByClientOrderID must not error")
	if mockTests {
		exp := &HFOrderDetailResponse{
			ID:                  "695f59478203450007f77322",
			Symbol:              "BTC-USDT",
			OperationType:       "DEAL",
			Type:                "limit",
			Side:                "buy",
			Price:               50000,
			Size:                0.00003,
			Funds:               1.5,
			DealSize:            0.00001,
			DealFunds:           0.5,
			Fee:                 0.0005,
			FeeCurrency:         currency.USDT,
			SelfTradePrevention: "DC",
			TimeInForce:         "GTT",
			PostOnly:            true,
			Hidden:              true,
			Iceberg:             true,
			VisibleSize:         0.00001,
			CancelAfter:         3600,
			Channel:             "API",
			ClientOrderID:       "9a1840ed0b5746459ede8e6f232daf2d",
			Remark:              "order remarks",
			Tags:                "order tags",
			CancelExist:         true,
			CreatedAt:           types.Time(time.UnixMilli(1767856455976)),
			LastUpdatedAt:       types.Time(time.UnixMilli(1767856456017)),
			TradeType:           "TRADE",
			InOrderBook:         true,
			CancelledSize:       0.00001,
			CancelledFunds:      0.5,
			RemainSize:          0.00001,
			RemainFunds:         0.5,
			Tax:                 0.00005,
			Active:              true,
			CancelReason:        5,
		}
		assert.Equal(t, exp, result, "GetHFOrderByClientOrderID should decode every field")

		_, err = e.GetHFOrderByClientOrderID(t.Context(), "9a1840ed0b5746459ede8e6f232daf2e", "BTC-USDT")
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetHFOrderByClientOrderID should report an order KuCoin returns no data for")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = e.GetHFOrderByClientOrderID(ctx, "9a1840ed0b5746459ede8e6f232daf2e", "BTC-USDT")
		assert.ErrorIs(t, err, context.Canceled, "GetHFOrderByClientOrderID should return the request error")
		return
	}
	assert.Equal(t, "9a1840ed0b5746459ede8e6f232daf2d", result.ClientOrderID, "GetHFOrderByClientOrderID should return the order")
}

func TestGetSymbolsWithOpenHFOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSymbolsWithOpenHFOrder(t.Context())
	require.NoError(t, err, "GetSymbolsWithOpenHFOrder must not error")
	require.NotNil(t, result, "GetSymbolsWithOpenHFOrder must return a response")
	if mockTests {
		exp := &SymbolsWithOpenHFOrderResponse{
			Symbols: []string{
				"ETH-USDT",
				"BTC-USDT",
			},
		}
		assert.Equal(t, exp, result, "GetSymbolsWithOpenHFOrder should decode every field")
	}
}

func TestGetOpenHFOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenHFOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetOpenHFOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOpenHFOrders(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetOpenHFOrders must not error")
	if mockTests {
		exp := []HFOrderResponse{
			{
				ID:                  "67120bbef094e200070976f6",
				Symbol:              "BTC-USDT",
				OperationType:       "DEAL",
				Type:                "limit",
				Side:                "buy",
				Price:               50000,
				Size:                0.00003,
				Funds:               1.5,
				DealSize:            0.00001,
				DealFunds:           0.5,
				Fee:                 0.0005,
				FeeCurrency:         currency.USDT,
				SelfTradePrevention: "DC",
				TimeInForce:         "GTT",
				PostOnly:            true,
				Hidden:              true,
				Iceberg:             true,
				VisibleSize:         0.00001,
				CancelAfter:         3600,
				Channel:             "API",
				ClientOrderID:       "5c52e11203aa677f33e493fb",
				Remark:              "order remarks",
				Tags:                "order tags",
				CancelExist:         true,
				CreatedAt:           types.Time(time.UnixMilli(1729235902748)),
				LastUpdatedAt:       types.Time(time.UnixMilli(1729235909862)),
				TradeType:           "TRADE",
				InOrderBook:         true,
				CancelledSize:       0.00001,
				CancelledFunds:      0.5,
				RemainSize:          0.00001,
				RemainFunds:         0.5,
				Tax:                 0.00005,
				Active:              true,
			},
		}
		assert.Equal(t, exp, result, "GetOpenHFOrders should decode every field")
	}
}

func TestGetOpenHFOrdersByPage(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenHFOrdersByPage(t.Context(), "", 1, 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetOpenHFOrdersByPage must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOpenHFOrdersByPage(t.Context(), "BTC-USDT", 1, 20)
	require.NoError(t, err, "GetOpenHFOrdersByPage must not error")
	require.NotNil(t, result, "GetOpenHFOrdersByPage must return a response")
	if mockTests {
		exp := &OpenHFOrdersPageResponse{
			CurrentPage: 1,
			PageSize:    20,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []HFOrderResponse{
				{
					ID:                  "67c1437ea5226600071cc080",
					Symbol:              "BTC-USDT",
					OperationType:       "DEAL",
					Type:                "limit",
					Side:                "buy",
					Price:               50000,
					Size:                0.00003,
					Funds:               1.5,
					DealSize:            0.00001,
					DealFunds:           0.5,
					Fee:                 0.0005,
					FeeCurrency:         currency.USDT,
					SelfTradePrevention: "DC",
					TimeInForce:         "GTT",
					PostOnly:            true,
					Hidden:              true,
					Iceberg:             true,
					VisibleSize:         0.00001,
					CancelAfter:         3600,
					Channel:             "API",
					ClientOrderID:       "5c52e11203aa677f33e493fb",
					Remark:              "order remarks",
					Tags:                "order tags",
					CancelExist:         true,
					CreatedAt:           types.Time(time.UnixMilli(1740718974367)),
					LastUpdatedAt:       types.Time(time.UnixMilli(1741867658590)),
					TradeType:           "TRADE",
					InOrderBook:         true,
					CancelledSize:       0.00001,
					CancelledFunds:      0.5,
					RemainSize:          0.00001,
					RemainFunds:         0.5,
					Tax:                 0.00005,
					Active:              true,
				},
			},
		}
		assert.Equal(t, exp, result, "GetOpenHFOrdersByPage should decode every field")
		return
	}
	assert.Equal(t, uint64(1), result.CurrentPage, "GetOpenHFOrdersByPage should return the requested page")
}

func TestGetClosedHFOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetClosedHFOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetClosedHFOrders must reject a nil request")
	_, err = e.GetClosedHFOrders(t.Context(), &ClosedHFOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetClosedHFOrders must reject an empty symbol")
	_, err = e.GetClosedHFOrders(t.Context(), &ClosedHFOrdersRequest{Symbol: "BTC-USDT", StartAt: time.UnixMilli(1729577516000), EndAt: time.UnixMilli(1729577515000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetClosedHFOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetClosedHFOrders(t.Context(), &ClosedHFOrdersRequest{Symbol: "BTC-USDT", Side: "buy", Type: "limit", LastID: 19814995255306, Limit: 20, StartAt: time.UnixMilli(1729577515000), EndAt: time.UnixMilli(1729577516000)})
	require.NoError(t, err, "GetClosedHFOrders must not error")
	if mockTests {
		exp := &ClosedHFOrdersResponse{
			LastID: 19814995255305,
			Items: []HFOrderDetailResponse{
				{
					ID:                  "6717422bd51c29000775ea03",
					Symbol:              "BTC-USDT",
					OperationType:       "DEAL",
					Type:                "limit",
					Side:                "buy",
					Price:               50000,
					Size:                0.00003,
					Funds:               1.5,
					DealSize:            0.00001,
					DealFunds:           0.5,
					Fee:                 0.0005,
					FeeCurrency:         currency.USDT,
					SelfTradePrevention: "DC",
					TimeInForce:         "GTT",
					PostOnly:            true,
					Hidden:              true,
					Iceberg:             true,
					VisibleSize:         0.00001,
					CancelAfter:         3600,
					Channel:             "API",
					ClientOrderID:       "5c52e11203aa677f33e493fb",
					Remark:              "order remarks",
					Tags:                "order tags",
					CancelExist:         true,
					CreatedAt:           types.Time(time.UnixMilli(1729577515444)),
					LastUpdatedAt:       types.Time(time.UnixMilli(1729577515481)),
					TradeType:           "TRADE",
					CancelledSize:       0.00001,
					CancelledFunds:      0.5,
					RemainSize:          0.00001,
					RemainFunds:         0.5,
					Tax:                 0.00005,
					CancelReason:        1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetClosedHFOrders should decode every field")
	}
}

func TestGetHFTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetHFTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHFTradeHistory must reject a nil request")
	_, err = e.GetHFTradeHistory(t.Context(), &HFTradeHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetHFTradeHistory must reject an empty symbol")
	_, err = e.GetHFTradeHistory(t.Context(), &HFTradeHistoryRequest{Symbol: "BTC-USDT", StartAt: time.UnixMilli(1729577516000), EndAt: time.UnixMilli(1729577515000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHFTradeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHFTradeHistory(t.Context(), &HFTradeHistoryRequest{Symbol: "BTC-USDT", OrderID: "6717422bd51c29000775ea03", Side: "buy", Type: "limit", LastID: 19814995255306, Limit: 20, StartAt: time.UnixMilli(1729577515000), EndAt: time.UnixMilli(1729577516000)})
	require.NoError(t, err, "GetHFTradeHistory must not error")
	if mockTests {
		exp := &HFTradeHistoryResponse{
			Items: []HFTrade{
				{
					ID:             19814995255305,
					Symbol:         "BTC-USDT",
					TradeID:        11029373945659392,
					OrderID:        "6717422bd51c29000775ea03",
					CounterOrderID: "67174228135f9e000709da8c",
					Side:           "buy",
					Liquidity:      "taker",
					ForceTaker:     true,
					Price:          67717.6,
					Size:           0.00001,
					Funds:          0.677176,
					Fee:            0.000677176,
					FeeRate:        0.001,
					FeeCurrency:    currency.USDT,
					Stop:           "loss",
					TradeType:      "TRADE",
					TaxRate:        0.001,
					Tax:            0.000677176,
					Type:           "limit",
					CreatedAt:      types.Time(time.UnixMilli(1729577515473)),
				},
			},
			LastID: 19814995255305,
		}
		assert.Equal(t, exp, result, "GetHFTradeHistory should decode every field")
	}
}

func TestGetDCP(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDCP(t.Context())
	require.NoError(t, err, "GetDCP must not error")
	require.NotNil(t, result, "GetDCP must return a response")
	if mockTests {
		exp := &DCPResponse{
			Timeout:     5,
			Symbols:     "BTC-USDT,ETH-USDT",
			CurrentTime: types.Time(time.Unix(1729241305, 0)),
			TriggerTime: types.Time(time.Unix(1729241308, 0)),
		}
		assert.Equal(t, exp, result, "GetDCP should decode every field")
	}
}

func TestSetDCP(t *testing.T) {
	t.Parallel()
	_, err := e.SetDCP(t.Context(), 0, nil)
	require.ErrorIs(t, err, errTimeoutRequired, "SetDCP must reject a zero timeout")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SetDCP(t.Context(), 5, []string{"BTC-USDT", "ETH-USDT"})
		return err
	})
	assert.JSONEq(t, `{"timeout":5,"symbols":"BTC-USDT,ETH-USDT"}`, sent.body, "SetDCP should send the timeout as the integer KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetDCP(t.Context(), 5, []string{"BTC-USDT", "ETH-USDT"})
	require.NoError(t, err, "SetDCP must not error")
	require.NotNil(t, result, "SetDCP must return a response")
	if mockTests {
		exp := &SetDCPResponse{
			CurrentTime: types.Time(time.Unix(1729656588, 0)),
			TriggerTime: types.Time(time.Unix(1729656593, 0)),
		}
		assert.Equal(t, exp, result, "SetDCP should decode every field")
		return
	}
	assert.True(t, result.TriggerTime.Time().After(result.CurrentTime.Time()), "SetDCP should trigger after the current time")
}

func TestPlaceStopOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceStopOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no side", req: &PlaceStopOrderRequest{}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &PlaceStopOrderRequest{Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "no type", req: &PlaceStopOrderRequest{Side: "buy", Symbol: "BTC-USDT"}, err: order.ErrTypeIsInvalid},
		{name: "no stop price", req: &PlaceStopOrderRequest{Side: "buy", Symbol: "BTC-USDT", Type: "limit"}, err: errStopPriceRequired},
		{name: "limit without price", req: &PlaceStopOrderRequest{Side: "buy", Symbol: "BTC-USDT", Type: "limit", StopPrice: 50000}, err: limits.ErrPriceBelowMin},
	} {
		_, err := e.PlaceStopOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "PlaceStopOrder must reject a request with %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceStopOrderRequest
		exp  *OrderIDResponse
	}{
		{
			name: "limit",
			req:  &PlaceStopOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Remark: "order remarks", SelfTradePrevention: "CN", Price: 50000, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, CancelAfter: 3600, StopPrice: 50000, TradeType: "TRADE", Stop: "loss"},
			exp: &OrderIDResponse{
				OrderID: "670fd33bf9406e0007ab3945",
			},
		},
		{
			name: "market with funds",
			req:  &PlaceStopOrderRequest{Side: "sell", Symbol: "BTC-USDT", Type: "market", Funds: 1, StopPrice: 60000, Stop: "entry"},
			exp: &OrderIDResponse{
				OrderID: "670fd33bf9406e0007ab3946",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceStopOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceStopOrder must not error")
			require.NotNil(t, result, "PlaceStopOrder must return a response")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceStopOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceStopOrder should return the order ID")
		})
	}
}

func TestBatchCancelStopOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchCancelStopOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BatchCancelStopOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchCancelStopOrders(t.Context(), &BatchCancelStopOrdersRequest{Symbol: "ETH-BTC", TradeType: "TRADE", OrderIDs: []string{"5bd6e9286d99522a52e458de", "5bd6e9286d99522a52e458df"}})
	require.NoError(t, err, "BatchCancelStopOrders must not error")
	if mockTests {
		exp := &CancelledOrderIDsResponse{
			CancelledOrderIDs: []string{
				"5bd6e9286d99522a52e458de",
				"5bd6e9286d99522a52e458df",
			},
		}
		assert.Equal(t, exp, result, "BatchCancelStopOrders should decode every field")
	}
}

func TestGetStopOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetStopOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStopOrderList must reject a nil request")
	_, err = e.GetStopOrderList(t.Context(), &StopOrderListRequest{StartAt: time.UnixMilli(1740626560000), EndAt: time.UnixMilli(1740626550000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStopOrderList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStopOrderList(t.Context(), &StopOrderListRequest{Symbol: "BTC-USDT", Side: "buy", Type: "limit", TradeType: "TRADE", StartAt: time.UnixMilli(1740626550000), EndAt: time.UnixMilli(1740626560000), CurrentPage: 1, OrderIDs: []string{"vs93gptvr9t2fsql003l8k5p"}, PageSize: 50, Stop: "stop"})
	require.NoError(t, err, "GetStopOrderList must not error")
	require.NotNil(t, result, "GetStopOrderList must return a response")
	if mockTests {
		exp := &StopOrderListResponse{
			CurrentPage: 1,
			PageSize:    50,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []StopOrderListItem{
				{
					ID:                  "vs93gptvr9t2fsql003l8k5p",
					Symbol:              "BTC-USDT",
					UserID:              "633559791e1cbc0001f319bc",
					Status:              "NEW",
					Type:                "limit",
					Side:                "buy",
					Price:               50000,
					Size:                0.00001,
					Funds:               0.5,
					SelfTradePrevention: "CN",
					TimeInForce:         "GTT",
					CancelAfter:         3600,
					PostOnly:            true,
					Hidden:              true,
					Iceberg:             true,
					VisibleSize:         0.000005,
					Channel:             "API",
					ClientOrderID:       "5c52e11203aa677f222233e493fb",
					Remark:              "order remarks",
					Tags:                "order tags",
					DomainID:            "kucoin",
					TradeSource:         "USER",
					TradeType:           "TRADE",
					FeeCurrency:         currency.USDT,
					TakerFeeRate:        0.001,
					MakerFeeRate:        0.001,
					CreatedAt:           types.Time(time.UnixMilli(1740626554884)),
					Stop:                "loss",
					StopTriggerTime:     types.Time(time.Unix(1740626560, 0)),
					StopPrice:           60000,
					OrderTime:           types.Time(time.UnixMilli(1740626554883)),
					RelatedNumber:       "vs93gptvr9t2fsql003l8k5q",
					LimitPrice:          59000,
					Pop:                 "1",
					ActivateCondition:   "1",
				},
			},
		}
		assert.Equal(t, exp, result, "GetStopOrderList should decode every field")
		return
	}
	assert.Equal(t, uint64(1), result.CurrentPage, "GetStopOrderList should return the requested page")
}

func TestGetStopOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetStopOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetStopOrderByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStopOrderByOrderID(t.Context(), "vs8hoo8q2ceshiue003b67c0")
	require.NoError(t, err, "GetStopOrderByOrderID must not error")
	if mockTests {
		exp := &StopOrderResponse{
			ID:                  "vs8hoo8q2ceshiue003b67c0",
			Symbol:              "KCS-USDT",
			UserID:              "60fe4956c43cbc0006562c2c",
			Status:              "NEW",
			Type:                "limit",
			Side:                "buy",
			Price:               0.01,
			Size:                0.01,
			Funds:               0.0001,
			SelfTradePrevention: "CN",
			TimeInForce:         "GTT",
			CancelAfter:         3600,
			PostOnly:            true,
			Hidden:              true,
			Iceberg:             true,
			VisibleSize:         0.005,
			Channel:             "API",
			ClientOrderID:       "40e0eb9efe6311eb8e58acde48001122",
			Remark:              "order remarks",
			Tags:                "order tags",
			DomainID:            "kucoin",
			TradeSource:         "USER",
			TradeType:           "TRADE",
			FeeCurrency:         currency.USDT,
			TakerFeeRate:        0.002,
			MakerFeeRate:        0.002,
			CreatedAt:           types.Time(time.UnixMilli(1629098781128)),
			Stop:                "loss",
			StopTriggerTime:     types.Time(time.Unix(1629098790, 0)),
			StopPrice:           10,
			OrderTime:           types.Time(time.Unix(0, 1629098781127530345)),
		}
		assert.Equal(t, exp, result, "GetStopOrderByOrderID should decode every field")
	}
}

func TestGetStopOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetStopOrderByClientOrderID(t.Context(), "", "KCS-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetStopOrderByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStopOrderByClientOrderID(t.Context(), "2b700942b5db41cebe578cff48960e09", "KCS-USDT")
	require.NoError(t, err, "GetStopOrderByClientOrderID must not error")
	if mockTests {
		exp := []StopOrderResponse{
			{
				ID:                  "vs8hoo8os561f5np0032vngj",
				Symbol:              "KCS-USDT",
				UserID:              "60fe4956c43cbc0006562c2c",
				Status:              "NEW",
				Type:                "limit",
				Side:                "buy",
				Price:               0.01,
				Size:                0.01,
				Funds:               0.0001,
				SelfTradePrevention: "CN",
				TimeInForce:         "GTT",
				CancelAfter:         3600,
				PostOnly:            true,
				Hidden:              true,
				Iceberg:             true,
				VisibleSize:         0.005,
				Channel:             "API",
				ClientOrderID:       "2b700942b5db41cebe578cff48960e09",
				Remark:              "order remarks",
				Tags:                "order tags",
				DomainID:            "kucoin",
				TradeSource:         "USER",
				TradeType:           "TRADE",
				FeeCurrency:         currency.USDT,
				TakerFeeRate:        0.002,
				MakerFeeRate:        0.002,
				CreatedAt:           types.Time(time.UnixMilli(1629020492837)),
				Stop:                "loss",
				StopTriggerTime:     types.Time(time.Unix(1629020500, 0)),
				StopPrice:           1,
				OrderTime:           types.Time(time.Unix(0, 1629020492834532600)),
			},
		}
		assert.Equal(t, exp, result, "GetStopOrderByClientOrderID should decode every field")
	}
}

func TestPlaceOCOOrder(t *testing.T) {
	t.Parallel()
	valid := PlaceOCOOrderRequest{Symbol: "BTC-USDT", Side: "buy", Price: 94000, Size: 0.1, ClientOrderID: "5c52e11203aa67f1e493fb", StopPrice: 98000, LimitPrice: 96000, Remark: "this is remark", TradeType: "TRADE"}
	for _, tc := range []struct {
		name   string
		modify func(*PlaceOCOOrderRequest)
		err    error
	}{
		{name: "no client order ID", modify: func(r *PlaceOCOOrderRequest) { r.ClientOrderID = "" }, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", modify: func(r *PlaceOCOOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no symbol", modify: func(r *PlaceOCOOrderRequest) { r.Symbol = "" }, err: currency.ErrSymbolStringEmpty},
		{name: "no price", modify: func(r *PlaceOCOOrderRequest) { r.Price = 0 }, err: limits.ErrPriceBelowMin},
		{name: "no size", modify: func(r *PlaceOCOOrderRequest) { r.Size = 0 }, err: limits.ErrAmountBelowMin},
		{name: "no stop price", modify: func(r *PlaceOCOOrderRequest) { r.StopPrice = 0 }, err: errStopPriceRequired},
		{name: "no limit price", modify: func(r *PlaceOCOOrderRequest) { r.LimitPrice = 0 }, err: errLimitPriceRequired},
	} {
		req := valid
		tc.modify(&req)
		_, err := e.PlaceOCOOrder(t.Context(), &req)
		require.ErrorIsf(t, err, tc.err, "PlaceOCOOrder must reject a request with %s", tc.name)
	}
	_, err := e.PlaceOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOCOOrder must reject a nil request")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.PlaceOCOOrder(t.Context(), &valid)
		return err
	})
	assert.JSONEq(t, `{"clientOid":"5c52e11203aa67f1e493fb","side":"buy","symbol":"BTC-USDT","price":"94000","size":"0.1","stopPrice":"98000","limitPrice":"96000","remark":"this is remark","tradeType":"TRADE"}`, sent.body, "PlaceOCOOrder should send the prices and size as the strings KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceOCOOrder(t.Context(), &valid)
	require.NoError(t, err, "PlaceOCOOrder must not error")
	require.NotNil(t, result, "PlaceOCOOrder must return a response")
	if mockTests {
		exp := &OrderIDResponse{
			OrderID: "674c316e688dea0007c7b986",
		}
		assert.Equal(t, exp, result, "PlaceOCOOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceOCOOrder should return the order ID")
}

func TestCancelOCOOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOCOOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOCOOrderByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOCOOrderByOrderID(t.Context(), "674c3b6e688dea0007c7bab2")
	require.NoError(t, err, "CancelOCOOrderByOrderID must not error")
	if mockTests {
		exp := &CancelledOrderIDsResponse{
			CancelledOrderIDs: []string{
				"vs93gpqc6kkmkk57003gok16",
				"vs93gpqc6kkmkk57003gok17",
			},
		}
		assert.Equal(t, exp, result, "CancelOCOOrderByOrderID should decode every field")
	}
}

func TestCancelOCOOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOCOOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelOCOOrderByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOCOOrderByClientOrderID(t.Context(), "5c52e1203aa6f3g7f1e493fb")
	require.NoError(t, err, "CancelOCOOrderByClientOrderID must not error")
	if mockTests {
		exp := &CancelledOrderIDsResponse{
			CancelledOrderIDs: []string{
				"vs93gpqc6r0mkk57003gok3h",
				"vs93gpqc6r0mkk57003gok3i",
			},
		}
		assert.Equal(t, exp, result, "CancelOCOOrderByClientOrderID should decode every field")
	}
}

func TestCancelOCOMultipleOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOCOMultipleOrders(t.Context(), []string{"674c388172cf2800072ee746", "674c38bdfd8300000795167e"}, "BTC-USDT")
	require.NoError(t, err, "CancelOCOMultipleOrders must not error")
	if mockTests {
		exp := &CancelledOrderIDsResponse{
			CancelledOrderIDs: []string{
				"vs93gpqc750mkk57003gok6i",
				"vs93gpqc750mkk57003gok6j",
				"vs93gpqc75c39p83003tnriu",
				"vs93gpqc75c39p83003tnriv",
			},
		}
		assert.Equal(t, exp, result, "CancelOCOMultipleOrders should decode every field")
	}
}

func TestGetOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetOCOOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOCOOrderList must reject a nil request")
	_, err = e.GetOCOOrderList(t.Context(), &OCOOrderListRequest{StartAt: time.UnixMilli(1733049600000), EndAt: time.UnixMilli(1733049590000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOCOOrderList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOCOOrderList(t.Context(), &OCOOrderListRequest{Symbol: "BTC-USDT", StartAt: time.UnixMilli(1733049590000), EndAt: time.UnixMilli(1733049600000), OrderIDs: []string{"674c3cfa72cf2800072ee7ce"}, PageSize: 10, CurrentPage: 1})
	require.NoError(t, err, "GetOCOOrderList must not error")
	require.NotNil(t, result, "GetOCOOrderList must return a response")
	if mockTests {
		exp := &OCOOrderListResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []OCOOrder{
				{
					OrderID:       "674c3cfa72cf2800072ee7ce",
					Symbol:        "BTC-USDT",
					ClientOrderID: "5c52e1203aa6f3g7f1e493fb",
					OrderTime:     types.Time(time.UnixMilli(1733049594803)),
					Status:        "NEW",
				},
			},
		}
		assert.Equal(t, exp, result, "GetOCOOrderList should decode every field")
		return
	}
	assert.Equal(t, uint64(1), result.CurrentPage, "GetOCOOrderList should return the requested page")
}
