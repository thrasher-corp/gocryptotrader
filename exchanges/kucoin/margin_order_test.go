package kucoin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceMarginHFOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceMarginHFOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &PlaceMarginHFOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", req: &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb"}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "limit order without a price", req: &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Size: 0.00001}, err: limits.ErrPriceBelowMin},
		{name: "valid", req: &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Price: 50000, Size: 0.00001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestPlaceMarginHFOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*PlaceMarginHFOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&PlaceMarginHFOrderRequest{}).Payload()
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "Payload must validate the request")

	for _, tc := range []struct {
		name string
		req  *PlaceMarginHFOrderRequest
		exp  map[string]any
	}{
		{
			name: "every parameter",
			req: &PlaceMarginHFOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Remark: "order remarks",
				SelfTradePrevention: "CN", Price: 50000, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, CancelAfter: 600,
				Funds: 0.5, IsIsolated: true, AutoBorrow: true, AutoRepay: true,
			},
			exp: map[string]any{
				"clientOid": "5c52e11203aa677f33e493fb", "side": "buy", "symbol": "BTC-USDT", "type": "limit", "remark": "order remarks",
				"stp": "CN", "price": "50000", "size": "0.00001", "timeInForce": "GTT", "postOnly": true, "cancelAfter": uint64(600),
				"funds": "0.5", "isIsolated": true, "autoBorrow": true, "autoRepay": true,
			},
		},
		{
			name: "required parameters only",
			req:  &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "sell", Symbol: "BTC-USDT", Price: 70000, Size: 0.00001},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fc", "side": "sell", "symbol": "BTC-USDT", "price": "70000", "size": "0.00001"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := tc.req.Payload()
			require.NoError(t, err, "Payload must not error")
			assert.Equal(t, tc.exp, body, "Payload should send every set parameter as the type KuCoin documents")
		})
	}
}

func TestPlaceMarginHFOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMarginHFOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceMarginHFOrder must reject a nil request")
	_, err = e.PlaceMarginHFOrder(t.Context(), &PlaceMarginHFOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PlaceMarginHFOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceMarginHFOrderRequest
		exp  *PlaceMarginHFOrderResponse
	}{
		{
			name: "isolated limit order with auto borrow",
			req: &PlaceMarginHFOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Remark: "order remarks",
				SelfTradePrevention: "CN", Price: 50000, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, CancelAfter: 600,
				IsIsolated: true, AutoBorrow: true, AutoRepay: true,
			},
			exp: &PlaceMarginHFOrderResponse{
				OrderID:       "671663e02188630007e21c9c",
				LoanApplyID:   "600656d9a33ac90009de4f6f",
				BorrowSize:    0.5,
				ClientOrderID: "5c52e11203aa677f33e493fb",
			},
		},
		{
			name: "cross market order by funds",
			req:  &PlaceMarginHFOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "buy", Symbol: "BTC-USDT", Type: "market", Funds: 1},
			exp:  &PlaceMarginHFOrderResponse{OrderID: "671663e02188630007e21c9d", ClientOrderID: "5c52e11203aa677f33e493fc"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceMarginHFOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceMarginHFOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceMarginHFOrder should decode every field")
				return
			}
			require.NotNil(t, result, "PlaceMarginHFOrder must return a result")
			assert.NotEmpty(t, result.OrderID, "PlaceMarginHFOrder should return the order ID")
		})
	}
}

func TestPlaceMarginHFOrderTest(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMarginHFOrderTest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceMarginHFOrderTest must reject a nil request")
	_, err = e.PlaceMarginHFOrderTest(t.Context(), &PlaceMarginHFOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PlaceMarginHFOrderTest must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PlaceMarginHFOrderTest(t.Context(), &PlaceMarginHFOrderRequest{
		ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Remark: "order remarks",
		SelfTradePrevention: "CN", Price: 50000, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, CancelAfter: 600,
		IsIsolated: true, AutoBorrow: true, AutoRepay: true,
	})
	require.NoError(t, err, "PlaceMarginHFOrderTest must not error")
	if mockTests {
		exp := &PlaceMarginHFOrderResponse{
			OrderID:       "5bd6e9286d99522a52e458de",
			LoanApplyID:   "600656d9a33ac90009de4f6f",
			BorrowSize:    0.5,
			ClientOrderID: "5c52e11203aa677f33e493fb",
		}
		assert.Equal(t, exp, result, "PlaceMarginHFOrderTest should decode every field")
		return
	}
	require.NotNil(t, result, "PlaceMarginHFOrderTest must return a result")
	assert.NotEmpty(t, result.OrderID, "PlaceMarginHFOrderTest should return an order ID")
}

func TestCancelMarginHFOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginHFOrderByOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMarginHFOrderByOrderID must reject an empty order ID")
	_, err = e.CancelMarginHFOrderByOrderID(t.Context(), "671663e02188630007e21c9c", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelMarginHFOrderByOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginHFOrderByOrderID(t.Context(), "671663e02188630007e21c9c", "BTC-USDT")
	require.NoError(t, err, "CancelMarginHFOrderByOrderID must not error")
	exp := &CancelMarginHFOrderByOrderIDResponse{OrderID: "671663e02188630007e21c9c"}
	assert.Equal(t, exp, result, "CancelMarginHFOrderByOrderID should return the order ID")
}

func TestCancelMarginHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginHFOrderByClientOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelMarginHFOrderByClientOrderID must reject an empty client order ID")
	_, err = e.CancelMarginHFOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelMarginHFOrderByClientOrderID must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginHFOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb", "BTC-USDT")
	require.NoError(t, err, "CancelMarginHFOrderByClientOrderID must not error")
	exp := &CancelMarginHFOrderByClientOrderIDResponse{ClientOrderID: "5c52e11203aa677f33e493fb"}
	assert.Equal(t, exp, result, "CancelMarginHFOrderByClientOrderID should return the client order ID")
}

func TestCancelAllMarginHFOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllMarginHFOrdersBySymbol(t.Context(), "", "MARGIN_TRADE")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelAllMarginHFOrdersBySymbol must reject an empty symbol")
	_, err = e.CancelAllMarginHFOrdersBySymbol(t.Context(), "BTC-USDT", "")
	require.ErrorIs(t, err, errTradeTypeMissing, "CancelAllMarginHFOrdersBySymbol must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tradeType := range []string{"MARGIN_TRADE", "MARGIN_ISOLATED_TRADE"} {
		t.Run(tradeType, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelAllMarginHFOrdersBySymbol(t.Context(), "BTC-USDT", tradeType)
			require.NoError(t, err, "CancelAllMarginHFOrdersBySymbol must not error")
			assert.Equal(t, "success", result, "CancelAllMarginHFOrdersBySymbol should return success")
		})
	}
}

func TestGetActiveMarginHFOrderSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveMarginHFOrderSymbols(t.Context(), "")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetActiveMarginHFOrderSymbols must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveMarginHFOrderSymbols(t.Context(), "MARGIN_TRADE")
	require.NoError(t, err, "GetActiveMarginHFOrderSymbols must not error")
	if mockTests {
		exp := &ActiveMarginHFOrderSymbolsResponse{SymbolSize: 2, Symbols: []string{"BTC-USDT", "ETH-USDT"}}
		assert.Equal(t, exp, result, "GetActiveMarginHFOrderSymbols should decode every field")
		return
	}
	require.NotNil(t, result, "GetActiveMarginHFOrderSymbols must return a result")
	assert.Equal(t, result.SymbolSize, uint64(len(result.Symbols)), "GetActiveMarginHFOrderSymbols should return as many symbols as it counts")
}

func TestGetActiveMarginHFOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveMarginHFOrders(t.Context(), "", "MARGIN_TRADE")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetActiveMarginHFOrders must reject an empty symbol")
	_, err = e.GetActiveMarginHFOrders(t.Context(), "BTC-USDT", "")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetActiveMarginHFOrders must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveMarginHFOrders(t.Context(), "BTC-USDT", "MARGIN_TRADE")
	require.NoError(t, err, "GetActiveMarginHFOrders must not error")
	if !mockTests {
		return
	}
	exp := []MarginHFOrder{
		{
			ID:                  "671667306afcdb000723107f",
			Symbol:              "BTC-USDT",
			OperationType:       "DEAL",
			Type:                "limit",
			Side:                "buy",
			Price:               50000,
			Size:                0.00002,
			Funds:               1,
			DealSize:            0.00001,
			DealFunds:           0.5,
			Fee:                 0.0005,
			FeeCurrency:         currency.USDT,
			SelfTradePrevention: "CN",
			TimeInForce:         "GTT",
			PostOnly:            true,
			CancelAfter:         600,
			Channel:             "API",
			ClientOrderID:       "5c52e11203aa677f33e493fb",
			Remark:              "order remarks",
			Tags:                "partner",
			CreatedAt:           types.Time(time.UnixMilli(1729521456248)),
			LastUpdatedAt:       types.Time(time.UnixMilli(1729521460940)),
			TradeType:           "MARGIN_TRADE",
			InOrderBook:         true,
			RemainSize:          0.00001,
			RemainFunds:         0.5,
			Tax:                 0.00005,
			Active:              true,
		},
		{
			ID:                  "671667306afcdb0007231080",
			Symbol:              "BTC-USDT",
			OperationType:       "DEAL",
			Type:                "limit",
			Side:                "sell",
			Price:               70000,
			Size:                0.0001,
			Funds:               7,
			FeeCurrency:         currency.USDT,
			SelfTradePrevention: "DC",
			Stop:                "entry",
			StopTriggered:       true,
			StopPrice:           69000,
			TimeInForce:         "GTC",
			Hidden:              true,
			Iceberg:             true,
			VisibleSize:         0.00001,
			Channel:             "API",
			ClientOrderID:       "5c52e11203aa677f33e493fc",
			CancelExist:         true,
			CreatedAt:           types.Time(time.UnixMilli(1729521500000)),
			LastUpdatedAt:       types.Time(time.UnixMilli(1729521600000)),
			TradeType:           "MARGIN_TRADE",
			InOrderBook:         true,
			CancelledSize:       0.00005,
			CancelledFunds:      3.5,
			RemainSize:          0.00005,
			RemainFunds:         3.5,
			Active:              true,
		},
	}
	assert.Equal(t, exp, result, "GetActiveMarginHFOrders should decode every field")
}

func TestGetClosedMarginHFOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetClosedMarginHFOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetClosedMarginHFOrders must reject a nil request")
	_, err = e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{TradeType: "MARGIN_TRADE"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetClosedMarginHFOrders must reject an empty symbol")
	_, err = e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetClosedMarginHFOrders must reject an empty trade type")
	_, err = e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE", StartAt: time.UnixMilli(1729600000000), EndAt: time.UnixMilli(1729513000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetClosedMarginHFOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{
		Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE", Side: "buy", Type: "limit", LastID: 136112949352, Limit: 20,
		StartAt: time.UnixMilli(1729513000000), EndAt: time.UnixMilli(1729600000000),
	})
	require.NoError(t, err, "GetClosedMarginHFOrders must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetClosedMarginHFOrders should return a result")
		return
	}
	exp := &ClosedMarginHFOrdersResponse{
		LastID: 136112949351,
		Items: []MarginHFOrderResponse{
			{
				ID:                  "6716491f6afcdb00078365c8",
				Symbol:              "BTC-USDT",
				OperationType:       "DEAL",
				Type:                "limit",
				Side:                "buy",
				Price:               50000,
				Size:                0.00001,
				Funds:               0.5,
				FeeCurrency:         currency.USDT,
				SelfTradePrevention: "CN",
				TimeInForce:         "GTT",
				PostOnly:            true,
				CancelAfter:         600,
				Channel:             "API",
				ClientOrderID:       "5c52e11203aa677f33e493fb",
				Remark:              "order remarks",
				Tags:                "partner",
				CancelExist:         true,
				CreatedAt:           types.Time(time.UnixMilli(1729513759162)),
				LastUpdatedAt:       types.Time(time.UnixMilli(1729521126597)),
				TradeType:           "MARGIN_TRADE",
				CancelledSize:       0.00001,
				CancelledFunds:      0.5,
				CancelReason:        3,
			},
			{
				ID:            "6716491f6afcdb00078365c9",
				Symbol:        "BTC-USDT",
				OperationType: "DEAL",
				Type:          "limit",
				Side:          "buy",
				Price:         67150,
				Size:          0.0001,
				Funds:         6.715,
				DealSize:      0.0001,
				DealFunds:     6.715,
				Fee:           0.006715,
				FeeCurrency:   currency.USDT,
				Stop:          "entry",
				StopTriggered: true,
				StopPrice:     67100,
				TimeInForce:   "GTC",
				Hidden:        true,
				Iceberg:       true,
				VisibleSize:   0.00002,
				Channel:       "API",
				ClientOrderID: "5c52e11203aa677f33e493fc",
				CreatedAt:     types.Time(time.UnixMilli(1729560000000)),
				LastUpdatedAt: types.Time(time.UnixMilli(1729560001000)),
				TradeType:     "MARGIN_TRADE",
				Tax:           0.0006715,
			},
		},
	}
	assert.Equal(t, exp, result, "GetClosedMarginHFOrders should decode every field")
}

func TestGetMarginHFTradeFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginHFTradeFills(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginHFTradeFills must reject a nil request")
	_, err = e.GetMarginHFTradeFills(t.Context(), &GetMarginHFTradeFillsRequest{TradeType: "MARGIN_TRADE"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMarginHFTradeFills must reject an empty symbol")
	_, err = e.GetMarginHFTradeFills(t.Context(), &GetMarginHFTradeFillsRequest{Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetMarginHFTradeFills must reject an empty trade type")
	_, err = e.GetMarginHFTradeFills(t.Context(), &GetMarginHFTradeFillsRequest{Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE", StartAt: time.UnixMilli(1729680000000), EndAt: time.UnixMilli(1729600000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginHFTradeFills must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginHFTradeFills(t.Context(), &GetMarginHFTradeFillsRequest{
		Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE", OrderID: "671868085584bc0007d85f46", Side: "buy", Type: "limit",
		LastID: 137891621992, Limit: 20, StartAt: time.UnixMilli(1729600000000), EndAt: time.UnixMilli(1729680000000),
	})
	require.NoError(t, err, "GetMarginHFTradeFills must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetMarginHFTradeFills should return a result")
		return
	}
	exp := &MarginHFTradeFillsResponse{
		Items: []MarginHFTradeFill{
			{
				ID:             137891621991,
				Symbol:         "BTC-USDT",
				TradeID:        11040911994273792,
				OrderID:        "671868085584bc0007d85f46",
				CounterOrderID: "67186805b7cbdf00071621f9",
				Side:           "buy",
				Liquidity:      "taker",
				ForceTaker:     true,
				Price:          67141.6,
				Size:           0.00001,
				Funds:          0.671416,
				Fee:            0.000671416,
				FeeRate:        0.001,
				FeeCurrency:    currency.USDT,
				Stop:           "loss",
				TradeType:      "MARGIN_TRADE",
				Tax:            0.0000671416,
				TaxRate:        0.0001,
				Type:           "limit",
				CreatedAt:      types.Time(time.UnixMilli(1729652744998)),
			},
		},
		LastID: 137891621991,
	}
	assert.Equal(t, exp, result, "GetMarginHFTradeFills should decode every field")
}

// expMarginHFOrder is the margin order the Get Order By OrderId and Get Order By ClientOid fixtures return
var expMarginHFOrder = &MarginHFOrderResponse{
	ID:                  "671667306afcdb000723107f",
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
	Stop:                "loss",
	StopTriggered:       true,
	StopPrice:           50500,
	TimeInForce:         "GTT",
	PostOnly:            true,
	Hidden:              true,
	Iceberg:             true,
	VisibleSize:         0.00001,
	CancelAfter:         3600,
	Channel:             "API",
	ClientOrderID:       "5c52e11203aa677f33e493fb",
	Remark:              "order remarks",
	Tags:                "partner",
	CancelExist:         true,
	CreatedAt:           types.Time(time.UnixMilli(1729521456248)),
	LastUpdatedAt:       types.Time(time.UnixMilli(1729651011877)),
	TradeType:           "MARGIN_TRADE",
	InOrderBook:         true,
	CancelledSize:       0.00001,
	CancelledFunds:      0.5,
	RemainSize:          0.00001,
	RemainFunds:         0.5,
	Tax:                 0.00005,
	Active:              true,
	CancelReason:        5,
}

func TestGetMarginHFOrderDetailByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginHFOrderDetailByOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginHFOrderDetailByOrderID must reject an empty order ID")
	_, err = e.GetMarginHFOrderDetailByOrderID(t.Context(), "671667306afcdb000723107f", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMarginHFOrderDetailByOrderID must reject an empty symbol")

	orderID := "671667306afcdb000723107f"
	if mockTests {
		_, err = e.GetMarginHFOrderDetailByOrderID(cancelledContext(t), orderID, "BTC-USDT")
		require.ErrorIs(t, err, context.Canceled, "GetMarginHFOrderDetailByOrderID must return the request error")
		_, err = e.GetMarginHFOrderDetailByOrderID(t.Context(), "671667306afcdb0007231099", "BTC-USDT")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginHFOrderDetailByOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		closed, err := e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE"})
		require.NoError(t, err, "GetClosedMarginHFOrders must not error")
		if len(closed.Items) == 0 {
			t.Skip("no BTC-USDT cross margin order to query")
		}
		orderID = closed.Items[0].ID
	}
	result, err := e.GetMarginHFOrderDetailByOrderID(t.Context(), orderID, "BTC-USDT")
	require.NoError(t, err, "GetMarginHFOrderDetailByOrderID must not error")
	if mockTests {
		assert.Equal(t, expMarginHFOrder, result, "GetMarginHFOrderDetailByOrderID should decode every field")
		return
	}
	assert.Equal(t, orderID, result.ID, "GetMarginHFOrderDetailByOrderID should return the order")
}

func TestGetMarginHFOrderDetailByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginHFOrderDetailByClientOrderID(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetMarginHFOrderDetailByClientOrderID must reject an empty client order ID")
	_, err = e.GetMarginHFOrderDetailByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMarginHFOrderDetailByClientOrderID must reject an empty symbol")

	clientOrderID := "5c52e11203aa677f33e493fb"
	if mockTests {
		_, err = e.GetMarginHFOrderDetailByClientOrderID(cancelledContext(t), clientOrderID, "BTC-USDT")
		require.ErrorIs(t, err, context.Canceled, "GetMarginHFOrderDetailByClientOrderID must return the request error")
		_, err = e.GetMarginHFOrderDetailByClientOrderID(t.Context(), "5c52e11203aa677f33e49399", "BTC-USDT")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginHFOrderDetailByClientOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		closed, err := e.GetClosedMarginHFOrders(t.Context(), &GetClosedMarginHFOrdersRequest{Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE"})
		require.NoError(t, err, "GetClosedMarginHFOrders must not error")
		if len(closed.Items) == 0 {
			t.Skip("no BTC-USDT cross margin order to query")
		}
		clientOrderID = closed.Items[0].ClientOrderID
	}
	result, err := e.GetMarginHFOrderDetailByClientOrderID(t.Context(), clientOrderID, "BTC-USDT")
	require.NoError(t, err, "GetMarginHFOrderDetailByClientOrderID must not error")
	if mockTests {
		assert.Equal(t, expMarginHFOrder, result, "GetMarginHFOrderDetailByClientOrderID should decode every field")
		return
	}
	assert.Equal(t, clientOrderID, result.ClientOrderID, "GetMarginHFOrderDetailByClientOrderID should return the order")
}

func TestPlaceMarginStopOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceMarginStopOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &PlaceMarginStopOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", req: &PlaceMarginStopOrderRequest{ClientOrderID: "9e64g33505cc899h55g615hd"}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &PlaceMarginStopOrderRequest{ClientOrderID: "9e64g33505cc899h55g615hd", Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "no stop price", req: &PlaceMarginStopOrderRequest{ClientOrderID: "9e64g33505cc899h55g615hd", Side: "buy", Symbol: "BTC-USDT", Price: 60500, Size: 0.00001}, err: errStopPriceRequired},
		{name: "a limit order without a price", req: &PlaceMarginStopOrderRequest{ClientOrderID: "9e64g33505cc899h55g615hd", Side: "buy", Symbol: "BTC-USDT", Size: 0.00001, StopPrice: 60000}, err: limits.ErrPriceBelowMin},
	} {
		_, err := e.PlaceMarginStopOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "PlaceMarginStopOrder must reject a request with %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceMarginStopOrderRequest
		exp  *PlaceMarginStopOrderResponse
	}{
		{
			name: "isolated limit order with auto borrow",
			req: &PlaceMarginStopOrderRequest{
				ClientOrderID: "9e64g33505cc899h55g615hd", Side: "buy", Symbol: "BTC-USDT", Type: "limit", SelfTradePrevention: "CB",
				Price: 60500, Size: 0.00001, TimeInForce: "GTT", PostOnly: true, CancelAfter: 3600, IsIsolated: true,
				AutoBorrow: true, AutoRepay: true, Remark: "Test order", Stop: "entry", StopPrice: 60000,
			},
			exp: &PlaceMarginStopOrderResponse{OrderID: "vs8hoq2h6t43dlku003vff2r", ClientOrderID: "9e64g33505cc899h55g615hd"},
		},
		{
			name: "cross market order by funds",
			req:  &PlaceMarginStopOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "buy", Symbol: "BTC-USDT", Type: "market", Funds: 1, StopPrice: 60000},
			exp:  &PlaceMarginStopOrderResponse{OrderID: "vs8hoq2h6t43dlku003vff2s", ClientOrderID: "5c52e11203aa677f33e493fc"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceMarginStopOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceMarginStopOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceMarginStopOrder should decode every field")
				return
			}
			require.NotNil(t, result, "PlaceMarginStopOrder must return a result")
			assert.Equal(t, tc.req.ClientOrderID, result.ClientOrderID, "PlaceMarginStopOrder should return the client order ID")
		})
	}
}

func TestCancelMarginStopOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginStopOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMarginStopOrderByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginStopOrderByOrderID(t.Context(), "vs8hoq2h6t43dlku003vff2r")
	require.NoError(t, err, "CancelMarginStopOrderByOrderID must not error")
	exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs8hoq2h6t43dlku003vff2r"}}
	assert.Equal(t, exp, result, "CancelMarginStopOrderByOrderID should return the cancelled order ID")
}

func TestCancelMarginStopOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginStopOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelMarginStopOrderByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginStopOrderByClientOrderID(t.Context(), "9e64g33505cc899h55g615hd")
	require.NoError(t, err, "CancelMarginStopOrderByClientOrderID must not error")
	if mockTests {
		exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs8hoq2h6t43dlku003vff2r"}}
		assert.Equal(t, exp, result, "CancelMarginStopOrderByClientOrderID should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginStopOrderByClientOrderID should return a result")
}

func TestCancelMarginStopOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginStopOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMarginStopOrders must reject a nil request")
	_, err = e.CancelMarginStopOrders(t.Context(), &CancelMarginStopOrdersRequest{Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, errTradeTypeMissing, "CancelMarginStopOrders must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginStopOrders(t.Context(), &CancelMarginStopOrdersRequest{
		Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE", OrderIDs: []string{"vs8t6q2ig66rv9r3003i782s", "vs8t6q2ig68nhgf9003vcd8c"},
	})
	require.NoError(t, err, "CancelMarginStopOrders must not error")
	if mockTests {
		exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs8t6q2ig66rv9r3003i782s", "vs8t6q2ig68nhgf9003vcd8c"}}
		assert.Equal(t, exp, result, "CancelMarginStopOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginStopOrders should return a result")
}

func TestGetMarginStopOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginStopOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginStopOrders must reject a nil request")
	_, err = e.GetMarginStopOrders(t.Context(), &GetMarginStopOrdersRequest{StartAt: time.UnixMilli(1740627000000), EndAt: time.UnixMilli(1740626000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginStopOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginStopOrders(t.Context(), &GetMarginStopOrdersRequest{
		Symbol: "BTC-USDT", Side: "buy", Type: "limit", TradeType: "MARGIN_TRADE", StartAt: time.UnixMilli(1740626000000),
		EndAt: time.UnixMilli(1740627000000), CurrentPage: 1, OrderIDs: []string{"vs93gptvr9t2fsql003l8k5p", "vs93gptvr9t2fsql003l8k5q"},
		PageSize: 50, Stop: "stop",
	})
	require.NoError(t, err, "GetMarginStopOrders must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetMarginStopOrders should return a result")
		return
	}
	exp := &MarginStopOrderListResponse{
		CurrentPage: 1,
		PageSize:    50,
		TotalNum:    2,
		TotalPage:   1,
		Items: []MarginStopOrderListItem{
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
				SelfTradePrevention: "CB",
				TimeInForce:         "GTT",
				CancelAfter:         3600,
				PostOnly:            true,
				Hidden:              true,
				Iceberg:             true,
				VisibleSize:         0.000005,
				Channel:             "API",
				ClientOrderID:       "5c52e11203aa677f222233e493fb",
				Remark:              "order remarks",
				Tags:                "partner",
				DomainID:            "kucoin",
				TradeSource:         "USER",
				TradeType:           "MARGIN_TRADE",
				FeeCurrency:         currency.USDT,
				TakerFeeRate:        0.001,
				MakerFeeRate:        0.0008,
				CreatedAt:           types.Time(time.UnixMilli(1740626554884)),
				Stop:                "loss",
				StopTriggerTime:     types.Time(time.UnixMilli(1740626600000)),
				StopPrice:           60000,
				OrderTime:           types.Time(time.Unix(0, 1740626554883000024)),
				RelatedNumber:       "vs93gptvr9t2fsql003l8k5q",
				LimitPrice:          59900,
				Pop:                 "0.05",
				ActivateCondition:   "60000",
			},
			{
				ID:            "vs93gptvr9t2fsql003l8k5q",
				Symbol:        "BTC-USDT",
				UserID:        "633559791e1cbc0001f319bc",
				Status:        "NEW",
				Type:          "limit",
				Side:          "buy",
				Price:         40000,
				Size:          0.00002,
				TimeInForce:   "GTC",
				CancelAfter:   -1,
				Channel:       "API",
				ClientOrderID: "5c52e11203aa677f222233e493fc",
				DomainID:      "kucoin",
				TradeSource:   "MARGIN_SYSTEM",
				TradeType:     "MARGIN_TRADE",
				FeeCurrency:   currency.USDT,
				TakerFeeRate:  0.001,
				MakerFeeRate:  0.0008,
				CreatedAt:     types.Time(time.UnixMilli(1740626554991)),
				Stop:          "entry",
				StopPrice:     45000,
				OrderTime:     types.Time(time.UnixMilli(1740626554990)),
			},
		},
	}
	assert.Equal(t, exp, result, "GetMarginStopOrders should decode every field")
}

// expMarginStopOrder is the stop order the Get Stop Order By OrderId and Get Stop Order By ClientOid fixtures return
var expMarginStopOrder = MarginStopOrderResponse{
	ID:                  "vs8t6q2ovifc076m003hdbpp",
	Symbol:              "ETH-USDT",
	UserID:              "633559791e1cbc0001f319bc",
	Status:              "TRIGGERED",
	Type:                "limit",
	Side:                "buy",
	Price:               2000,
	Size:                0.005,
	Funds:               10,
	SelfTradePrevention: "CB",
	TimeInForce:         "GTC",
	CancelAfter:         -1,
	PostOnly:            true,
	Hidden:              true,
	Iceberg:             true,
	VisibleSize:         0.001,
	Channel:             "API",
	ClientOrderID:       "6d73f22304aa677g33e494fs",
	Remark:              "Test order",
	Tags:                "partner",
	DomainID:            "kucoin",
	TradeSource:         "USER",
	TradeType:           "MARGIN_TRADE",
	FeeCurrency:         currency.USDT,
	TakerFeeRate:        0.001,
	MakerFeeRate:        0.0008,
	CreatedAt:           types.Time(time.UnixMilli(1750662302773)),
	Stop:                "entry",
	StopTriggerTime:     types.Time(time.UnixMilli(1750662400000)),
	StopPrice:           2600,
	OrderTime:           types.Time(time.UnixMilli(1750662302772)),
}

func TestGetMarginStopOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginStopOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginStopOrderByOrderID must reject an empty order ID")

	orderID := "vs8t6q2ovifc076m003hdbpp"
	if mockTests {
		_, err = e.GetMarginStopOrderByOrderID(cancelledContext(t), orderID)
		require.ErrorIs(t, err, context.Canceled, "GetMarginStopOrderByOrderID must return the request error")
		_, err = e.GetMarginStopOrderByOrderID(t.Context(), "vs8t6q2ovifc076m003hdbpq")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginStopOrderByOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		stopOrders, err := e.GetMarginStopOrders(t.Context(), &GetMarginStopOrdersRequest{})
		require.NoError(t, err, "GetMarginStopOrders must not error")
		if len(stopOrders.Items) == 0 {
			t.Skip("no margin stop order to query")
		}
		orderID = stopOrders.Items[0].ID
	}
	result, err := e.GetMarginStopOrderByOrderID(t.Context(), orderID)
	require.NoError(t, err, "GetMarginStopOrderByOrderID must not error")
	if mockTests {
		assert.Equal(t, &expMarginStopOrder, result, "GetMarginStopOrderByOrderID should decode every field")
		return
	}
	assert.Equal(t, orderID, result.ID, "GetMarginStopOrderByOrderID should return the order")
}

func TestGetMarginStopOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginStopOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetMarginStopOrderByClientOrderID must reject an empty client order ID")

	clientOrderID := "6d73f22304aa677g33e494fs"
	if mockTests {
		_, err = e.GetMarginStopOrderByClientOrderID(cancelledContext(t), clientOrderID)
		require.ErrorIs(t, err, context.Canceled, "GetMarginStopOrderByClientOrderID must return the request error")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		stopOrders, err := e.GetMarginStopOrders(t.Context(), &GetMarginStopOrdersRequest{})
		require.NoError(t, err, "GetMarginStopOrders must not error")
		if len(stopOrders.Items) == 0 {
			t.Skip("no margin stop order to query")
		}
		clientOrderID = stopOrders.Items[0].ClientOrderID
	}
	result, err := e.GetMarginStopOrderByClientOrderID(t.Context(), clientOrderID)
	require.NoError(t, err, "GetMarginStopOrderByClientOrderID must not error")
	if mockTests {
		assert.Equal(t, []MarginStopOrderResponse{expMarginStopOrder}, result, "GetMarginStopOrderByClientOrderID should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetMarginStopOrderByClientOrderID must return the order")
	assert.Equal(t, clientOrderID, result[0].ClientOrderID, "GetMarginStopOrderByClientOrderID should return the order")
}

func TestMarginStopOrdersByClientOrderIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := MarginStopOrdersByClientOrderID{
		{ID: "vs8t6q2ovifc076m003hdbpp", ClientOrderID: "6d73f22304aa677g33e494fs", CancelAfter: -1, OrderTime: types.Time(time.Unix(0, 1750662302772000024))},
	}
	for _, tc := range []struct {
		name string
		data string
		exp  MarginStopOrdersByClientOrderID
	}{
		{name: "array", data: `[{"id":"vs8t6q2ovifc076m003hdbpp","clientOid":"6d73f22304aa677g33e494fs","cancelAfter":-1,"orderTime":1750662302772000024}]`, exp: exp},
		{name: "object", data: ` {"id":"vs8t6q2ovifc076m003hdbpp","clientOid":"6d73f22304aa677g33e494fs","cancelAfter":-1,"orderTime":1750662302772000024}`, exp: exp},
		{name: "null", data: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result MarginStopOrdersByClientOrderID
			require.NoError(t, json.Unmarshal([]byte(tc.data), &result), "Unmarshal must not error")
			assert.Equal(t, tc.exp, result, "UnmarshalJSON should decode the stop orders")
		})
	}
	var result MarginStopOrdersByClientOrderID
	assert.Error(t, json.Unmarshal([]byte(`{"cancelAfter":"soon"}`), &result), "UnmarshalJSON should reject an invalid stop order object")
	assert.Error(t, json.Unmarshal([]byte(`[{"cancelAfter":"soon"}]`), &result), "UnmarshalJSON should reject an invalid stop order array")
}

func TestPlaceMarginOCOOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceMarginOCOOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &PlaceMarginOCOOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb"}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "no price", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT"}, err: limits.ErrPriceBelowMin},
		{name: "no size", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Price: 50000}, err: limits.ErrAmountBelowMin},
		{name: "no stop price", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Price: 50000, Size: 0.00001}, err: errStopPriceRequired},
		{name: "no limit price", req: &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Price: 50000, Size: 0.00001, StopPrice: 70000}, err: errLimitPriceRequired},
	} {
		_, err := e.PlaceMarginOCOOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "PlaceMarginOCOOrder must reject a request with %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceMarginOCOOrderRequest
		exp  *PlaceMarginOCOOrderResponse
	}{
		{
			name: "isolated buy with auto borrow and repay",
			req: &PlaceMarginOCOOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "BTC-USDT", Price: 50000, Size: 0.00001,
				StopPrice: 70000, LimitPrice: 70100, IsIsolated: true, AutoRepay: true, AutoBorrow: true,
			},
			exp: &PlaceMarginOCOOrderResponse{OrderID: "674c316e688dea0007c7b986"},
		},
		{
			name: "cross sell",
			req:  &PlaceMarginOCOOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "sell", Symbol: "BTC-USDT", Price: 70000, Size: 0.00001, StopPrice: 50000, LimitPrice: 49900},
			exp:  &PlaceMarginOCOOrderResponse{OrderID: "674c316e688dea0007c7b987"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceMarginOCOOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceMarginOCOOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceMarginOCOOrder should decode every field")
				return
			}
			require.NotNil(t, result, "PlaceMarginOCOOrder must return a result")
			assert.NotEmpty(t, result.OrderID, "PlaceMarginOCOOrder should return the order ID")
		})
	}
}

func TestCancelMarginOCOOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginOCOOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMarginOCOOrderByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginOCOOrderByOrderID(t.Context(), "674c316e688dea0007c7b986")
	require.NoError(t, err, "CancelMarginOCOOrderByOrderID must not error")
	if mockTests {
		exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs93gpqc6kkmkk57003gok16", "vs93gpqc6kkmkk57003gok17"}}
		assert.Equal(t, exp, result, "CancelMarginOCOOrderByOrderID should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginOCOOrderByOrderID should return a result")
}

func TestCancelMarginOCOOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginOCOOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelMarginOCOOrderByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginOCOOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e493fb")
	require.NoError(t, err, "CancelMarginOCOOrderByClientOrderID must not error")
	if mockTests {
		exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs93gpqc6r0mkk57003gok3h", "vs93gpqc6r0mkk57003gok3i"}}
		assert.Equal(t, exp, result, "CancelMarginOCOOrderByClientOrderID should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginOCOOrderByClientOrderID should return a result")
}

func TestCancelMarginOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginOCOOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMarginOCOOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CancelMarginOCOOrdersRequest
	}{
		{name: "by order ID, symbol and trade type", req: &CancelMarginOCOOrdersRequest{OrderIDs: []string{"674c316e688dea0007c7b986", "674c316e688dea0007c7b987"}, Symbol: "BTC-USDT", TradeType: "MARGIN_TRADE"}},
		{name: "every OCO order", req: &CancelMarginOCOOrdersRequest{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelMarginOCOOrders(t.Context(), tc.req)
			require.NoError(t, err, "CancelMarginOCOOrders must not error")
			if mockTests {
				exp := &CancelledMarginOrdersResponse{CancelledOrderIDs: []string{"vs8t6q2q15qpbcik003ss820", "vs8t6q2q15qpbcik003ss821", "vs8t6q2q15ts076m003hdhfm", "vs8t6q2q15ts076m003hdhfn"}}
				assert.Equal(t, exp, result, "CancelMarginOCOOrders should decode every field")
				return
			}
			assert.NotNil(t, result, "CancelMarginOCOOrders should return a result")
		})
	}
}

// expMarginOCOOrder is the OCO order the Get OCO Order By OrderId and Get OCO Order By ClientOid fixtures return
var expMarginOCOOrder = &MarginOCOOrderResponse{
	Symbol:        "ETH-USDT",
	ClientOrderID: "5c52e11203aa677f33e4234fc",
	OrderID:       "685a0dc1c01cd6000716c615",
	OrderTime:     types.Time(time.UnixMilli(1750732225854)),
	Status:        "NEW",
}

func TestGetMarginOCOOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginOCOOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginOCOOrderByOrderID must reject an empty order ID")

	orderID := "685a0dc1c01cd6000716c615"
	if mockTests {
		_, err = e.GetMarginOCOOrderByOrderID(cancelledContext(t), orderID)
		require.ErrorIs(t, err, context.Canceled, "GetMarginOCOOrderByOrderID must return the request error")
		_, err = e.GetMarginOCOOrderByOrderID(t.Context(), "685a0dc1c01cd6000716c616")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginOCOOrderByOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ocoOrders, err := e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{PageSize: 10, CurrentPage: 1})
		require.NoError(t, err, "GetMarginOCOOrders must not error")
		if len(ocoOrders.Items) == 0 {
			t.Skip("no margin OCO order to query")
		}
		orderID = ocoOrders.Items[0].OrderID
	}
	result, err := e.GetMarginOCOOrderByOrderID(t.Context(), orderID)
	require.NoError(t, err, "GetMarginOCOOrderByOrderID must not error")
	if mockTests {
		assert.Equal(t, expMarginOCOOrder, result, "GetMarginOCOOrderByOrderID should decode every field")
		return
	}
	assert.Equal(t, orderID, result.OrderID, "GetMarginOCOOrderByOrderID should return the order")
}

func TestGetMarginOCOOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginOCOOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetMarginOCOOrderByClientOrderID must reject an empty client order ID")

	clientOrderID := "5c52e11203aa677f33e4234fc"
	if mockTests {
		_, err = e.GetMarginOCOOrderByClientOrderID(cancelledContext(t), clientOrderID)
		require.ErrorIs(t, err, context.Canceled, "GetMarginOCOOrderByClientOrderID must return the request error")
		_, err = e.GetMarginOCOOrderByClientOrderID(t.Context(), "5c52e11203aa677f33e4234fd")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginOCOOrderByClientOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ocoOrders, err := e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{PageSize: 10, CurrentPage: 1})
		require.NoError(t, err, "GetMarginOCOOrders must not error")
		if len(ocoOrders.Items) == 0 {
			t.Skip("no margin OCO order to query")
		}
		clientOrderID = ocoOrders.Items[0].ClientOrderID
	}
	result, err := e.GetMarginOCOOrderByClientOrderID(t.Context(), clientOrderID)
	require.NoError(t, err, "GetMarginOCOOrderByClientOrderID must not error")
	if mockTests {
		assert.Equal(t, expMarginOCOOrder, result, "GetMarginOCOOrderByClientOrderID should decode every field")
		return
	}
	assert.Equal(t, clientOrderID, result.ClientOrderID, "GetMarginOCOOrderByClientOrderID should return the order")
}

func TestGetMarginOCOOrderDetailByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginOCOOrderDetailByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginOCOOrderDetailByOrderID must reject an empty order ID")

	orderID := "685a0dc1c01cd6000716c615"
	if mockTests {
		_, err = e.GetMarginOCOOrderDetailByOrderID(cancelledContext(t), orderID)
		require.ErrorIs(t, err, context.Canceled, "GetMarginOCOOrderDetailByOrderID must return the request error")
		_, err = e.GetMarginOCOOrderDetailByOrderID(t.Context(), "685a0dc1c01cd6000716c616")
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetMarginOCOOrderDetailByOrderID must report an order KuCoin returns no data for")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ocoOrders, err := e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{PageSize: 10, CurrentPage: 1})
		require.NoError(t, err, "GetMarginOCOOrders must not error")
		if len(ocoOrders.Items) == 0 {
			t.Skip("no margin OCO order to query")
		}
		orderID = ocoOrders.Items[0].OrderID
	}
	result, err := e.GetMarginOCOOrderDetailByOrderID(t.Context(), orderID)
	require.NoError(t, err, "GetMarginOCOOrderDetailByOrderID must not error")
	if !mockTests {
		assert.Equal(t, orderID, result.OrderID, "GetMarginOCOOrderDetailByOrderID should return the order")
		return
	}
	exp := &MarginOCOOrderDetailResponse{
		MarginOCOOrderResponse: *expMarginOCOOrder,
		Orders: []MarginOCOOrderLeg{
			{
				ID:        "vs8t6q2q1n0s076m003hdhgm",
				Symbol:    "ETH-USDT",
				Side:      "buy",
				Price:     2300,
				StopPrice: 2300,
				Size:      0.01,
				Status:    "NEW",
			},
			{
				ID:        "vs8t6q2q1n0s076m003hdhgn",
				Symbol:    "ETH-USDT",
				Side:      "buy",
				Price:     2510,
				StopPrice: 2500,
				Size:      0.01,
				Status:    "NEW",
			},
		},
	}
	assert.Equal(t, exp, result, "GetMarginOCOOrderDetailByOrderID should decode every field")
}

func TestGetMarginOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginOCOOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginOCOOrders must reject a nil request")
	_, err = e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{CurrentPage: 1})
	require.ErrorIs(t, err, errPageSizeRequired, "GetMarginOCOOrders must reject an empty page size")
	_, err = e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{PageSize: 10})
	require.ErrorIs(t, err, errCurrentPageRequired, "GetMarginOCOOrders must reject an empty current page")
	_, err = e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{PageSize: 10, CurrentPage: 1, StartAt: time.UnixMilli(1733050000000), EndAt: time.UnixMilli(1733049000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginOCOOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginOCOOrders(t.Context(), &GetMarginOCOOrdersRequest{
		PageSize: 10, CurrentPage: 1, Symbol: "BTC-USDT", StartAt: time.UnixMilli(1733049000000), EndAt: time.UnixMilli(1733050000000),
		OrderIDs: []string{"674c3cfa72cf2800072ee7ce", "674c3cfa72cf2800072ee7cf"}, TradeType: "MARGIN_TRADE",
	})
	require.NoError(t, err, "GetMarginOCOOrders must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetMarginOCOOrders should return a result")
		return
	}
	exp := &MarginOCOOrderListResponse{
		CurrentPage: 1,
		PageSize:    10,
		TotalNum:    2,
		TotalPage:   1,
		Items: []MarginOCOOrderResponse{
			{
				Symbol:        "BTC-USDT",
				ClientOrderID: "5c52e1203aa6f3g7f1e493fb",
				OrderID:       "674c3cfa72cf2800072ee7ce",
				OrderTime:     types.Time(time.UnixMilli(1733049594803)),
				Status:        "NEW",
			},
			{
				Symbol:        "BTC-USDT",
				ClientOrderID: "5c52e1203aa6f3g7f1e493fc",
				OrderID:       "674c3cfa72cf2800072ee7cf",
				OrderTime:     types.Time(time.UnixMilli(1733049594900)),
				Status:        "TRIGGERED",
			},
		},
	}
	assert.Equal(t, exp, result, "GetMarginOCOOrders should decode every field")
}

func TestValidateMarginOrderAmounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		orderType          string
		price, size, funds float64
		err                error
	}{
		{name: "default type without a price", size: 0.00001, err: limits.ErrPriceBelowMin},
		{name: "limit without a price", orderType: "limit", size: 0.00001, err: limits.ErrPriceBelowMin},
		{name: "limit without a size", orderType: "limit", price: 50000, err: limits.ErrAmountBelowMin},
		{name: "limit", orderType: "limit", price: 50000, size: 0.00001},
		{name: "market without a size or funds", orderType: "market", err: errSizeOrFundIsRequired},
		{name: "market with a size and funds", orderType: "market", size: 0.00001, funds: 1, err: errSizeAndFundsExclusive},
		{name: "market by size", orderType: "market", size: 0.00001},
		{name: "market by funds", orderType: "market", funds: 1},
		{name: "type KuCoin validates", orderType: "stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateMarginOrderAmounts(tc.orderType, tc.price, tc.size, tc.funds)
			assert.ErrorIs(t, err, tc.err, "validateMarginOrderAmounts should return the expected error")
		})
	}
}
