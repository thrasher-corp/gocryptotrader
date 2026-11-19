package binanceus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketAPIGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetCurrentOrderCountUsage(t.Context())
	require.NoError(t, err, "GetCurrentOrderCountUsage must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "GetCurrentOrderCountUsage should return the order rate limits")
		return
	}
	exp := []RateLimitItemResponse{
		{RateLimitType: "ORDERS", Interval: "SECOND", IntervalNumber: 10, Limit: 50},
		{RateLimitType: "ORDERS", Interval: "DAY", IntervalNumber: 1, Limit: 160000},
	}
	assert.Equal(t, exp, result, "GetCurrentOrderCountUsage should decode every field")
}

func TestWebsocketAPINewOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().NewOrder(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 25, NewClientOrderID: "gct-market-1", NewOrderResponseType: NewOrderResponseTypeFull})
	require.NoError(t, err, "NewOrder must not error")
	require.NotNil(t, result, "NewOrder must return the order")
	if !mockTests {
		assert.Equal(t, "gct-market-1", result.ClientOrderID, "NewOrder should place the order under its client order ID")
		return
	}
	exp := &NewOrderResponse{
		Symbol:                     "BTCUSDT",
		OrderID:                    12569099453,
		OrderListID:                -1,
		ClientOrderID:              "4d96324ff9d44481926157ec08158a40",
		TransactTime:               types.Time(time.UnixMilli(1660801715793)),
		Price:                      23416.1,
		OriginalQuantity:           0.00847,
		ExecutedQuantity:           0.00847,
		CumulativeQuoteQuantity:    198.335215,
		Status:                     "FILLED",
		TimeInForce:                TimeInForceGoodTillCancel,
		Type:                       OrderTypeLimit,
		Side:                       SideSell,
		WorkingTime:                1660801715793,
		SelfTradePreventionMode:    SelfTradePreventionModeNone,
		StopPrice:                  23500,
		TrailingDeltaInBasisPoints: 10,
		TrailingTime:               -1,
		ExpiryReason:               "EXPIRED_IN_MATCH",
		Fills: []OrderFill{
			{Price: 23416.1, Quantity: 0.00635, CommissionAsset: currency.BNB, TradeID: 1650422481},
			{Price: 23416.5, Quantity: 0.00212, CommissionAsset: currency.BNB, TradeID: 1650422482},
		},
	}
	assert.Equal(t, exp, result, "NewOrder should decode every field")
}

func TestWebsocketAPINewOrderTest(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	err := ex.WebsocketAPI().NewOrderTest(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeLimit, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.001, Price: 80000})
	assert.NoError(t, err, "NewOrderTest should not error")
}

func TestWebsocketAPIQueryOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().QueryOrder(t.Context(), currency.NewBTCUSDT(), "myOrder1", 1)
	if !mockTests {
		require.ErrorIs(t, err, order.ErrOrderNotFound, "QueryOrder must report an order the account does not have")
		return
	}
	require.NoError(t, err, "QueryOrder must not error")
	exp := &TradeOrderResponse{
		Symbol:                     "BTCUSDT",
		OrderID:                    12569099453,
		OrderListID:                -1,
		ClientOrderID:              "4d96324ff9d44481926157",
		Price:                      23416.1,
		OriginalQuantity:           0.00847,
		ExecutedQuantity:           0.00847,
		CumulativeQuoteQuantity:    198.335215,
		Status:                     "FILLED",
		TimeInForce:                TimeInForceGoodTillCancel,
		Type:                       OrderTypeLimit,
		Side:                       SideSell,
		Time:                       types.Time(time.UnixMilli(1660801715639)),
		UpdateTime:                 types.Time(time.UnixMilli(1660801717945)),
		IsWorking:                  true,
		WorkingTime:                1660801715639,
		SelfTradePreventionMode:    SelfTradePreventionModeNone,
		PreventedQuantity:          1.2,
		TrailingDeltaInBasisPoints: 10,
		TrailingTime:               -1,
	}
	assert.Equal(t, exp, result, "QueryOrder should decode every field")
}

func TestWebsocketAPIOpenOrders(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().OpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "OpenOrders must not error")
	if !mockTests {
		return
	}
	exp := []TradeOrderResponse{{
		Symbol:                  "BTCUSDT",
		OrderID:                 12569099453,
		OrderListID:             -1,
		ClientOrderID:           "4d96324ff9d44481926157",
		Price:                   23416.1,
		OriginalQuantity:        0.00847,
		ExecutedQuantity:        0.0072,
		CumulativeQuoteQuantity: 172.43931,
		Status:                  "PARTIALLY_FILLED",
		TimeInForce:             TimeInForceGoodTillCancel,
		Type:                    OrderTypeLimit,
		Side:                    SideSell,
		Time:                    types.Time(time.UnixMilli(1660801715639)),
		UpdateTime:              types.Time(time.UnixMilli(1660801717945)),
		IsWorking:               true,
		WorkingTime:             1660801715639,
		SelfTradePreventionMode: SelfTradePreventionModeNone,
	}}
	assert.Equal(t, exp, result, "OpenOrders should decode every field")
}

func TestWebsocketAPICancelExistingOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().CancelExistingOrder(t.Context(), &CancelOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 4, OriginalClientOrderID: "myOrder1", NewClientOrderID: "cancelMyOrder1", CancelRestrictions: CancelRestrictionsOnlyNew})
	require.NoError(t, err, "CancelExistingOrder must not error")
	if !mockTests {
		return
	}
	exp := &CancelOrderResponse{
		Symbol:                     "BTCUSDT",
		OriginalClientOrderID:      "4d96324ff9d44481926157",
		OrderID:                    12569099453,
		OrderListID:                -1,
		ClientOrderID:              "91fe37ce9e69c90d6358c0",
		Price:                      23416.1,
		OriginalQuantity:           0.00847,
		ExecutedQuantity:           0.00001,
		OriginalQuoteOrderQuantity: 25,
		CumulativeQuoteQuantity:    0.234161,
		Status:                     "CANCELED",
		TimeInForce:                TimeInForceGoodTillCancel,
		Type:                       OrderTypeLimit,
		Side:                       SideSell,
		SelfTradePreventionMode:    SelfTradePreventionModeNone,
		TrailingTime:               -1,
	}
	assert.Equal(t, exp, result, "CancelExistingOrder should decode every field")
}

func TestWebsocketAPICancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().CancelAllOpenOrderOnSymbol(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "CancelAllOpenOrderOnSymbol must not error")
	if !mockTests {
		return
	}
	listOrderReport := func(origClientOrderID string, orderID uint64, orderType OrderType, price, stopPrice float64) OrderResponse {
		return OrderResponse{
			Symbol: "BTCUSDT", OriginalClientOrderID: origClientOrderID, OrderID: orderID, OrderListID: 19431, ClientOrderID: "OFFXQtxVFZ6Nbcg4PgE2DA",
			Price: types.Number(price), OriginalQuantity: 0.0085, Status: "CANCELED", TimeInForce: TimeInForceGoodTillCancel, Type: orderType,
			Side: SideBuy, SelfTradePreventionMode: SelfTradePreventionModeNone, StopPrice: types.Number(stopPrice),
		}
	}
	exp := []CancelledOpenOrderResponse{
		{
			Symbol: "BTCUSDT", OriginalClientOrderID: "4d96324ff9d44481926157", OrderID: 12569099453, OrderListID: -1, ClientOrderID: "91fe37ce9e69c90d6358c0",
			Price: 23416.1, OriginalQuantity: 0.00847, ExecutedQuantity: 0.00001, CumulativeQuoteQuantity: 0.234161, Status: "CANCELED",
			TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimit, Side: SideSell, SelfTradePreventionMode: SelfTradePreventionModeNone, TrailingTime: -1,
		},
		{
			Symbol:            "BTCUSDT",
			OrderListID:       19431,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "iuVNVJYYrByz6C4yGOPPK0",
			TransactionTime:   types.Time(time.UnixMilli(1660803702431)),
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 12569099453, ClientOrderID: "bX5wROblo6YeDwa9iTLeyY"}, {Symbol: "BTCUSDT", OrderID: 12569099454, ClientOrderID: "Tnu2IP0J5Y4mxw3IATBfmW"}},
			OrderReports: []OrderResponse{
				listOrderReport("bX5wROblo6YeDwa9iTLeyY", 12569099453, OrderTypeStopLossLimit, 23450.5, 23430),
				listOrderReport("Tnu2IP0J5Y4mxw3IATBfmW", 12569099454, OrderTypeLimitMaker, 23400, 0),
			},
		},
	}
	assert.Equal(t, exp, result, "CancelAllOpenOrderOnSymbol should decode cancelled orders and order lists")
}

func TestWebsocketAPIGetAccountTradeList(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), OrderID: 100234})
	require.NoError(t, err, "GetAccountTradeList must not error")
	if !mockTests {
		return
	}
	exp := []AccountTradeResponse{
		{Symbol: "BTCUSDT", ID: 1650422481, OrderID: 12569099453, OrderListID: -1, Price: 23416.1, Quantity: 0.00635, QuoteQuantity: 148.692235, CommissionAsset: currency.BNB, Time: types.Time(time.UnixMilli(1660801715793)), IsMaker: true, IsBestMatch: true},
		{Symbol: "BTCUSDT", ID: 1650422482, OrderID: 12569099453, OrderListID: -1, Price: 23416.5, Quantity: 0.00212, QuoteQuantity: 49.64298, CommissionAsset: currency.BNB, Time: types.Time(time.UnixMilli(1660801715793)), IsMaker: true, IsBestMatch: true},
	}
	assert.Equal(t, exp, result, "GetAccountTradeList should decode every field")
}

func TestWebsocketAPICancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	cancelled := CancelReplaceCancelResponse{
		Symbol: "BTCUSDT", OriginalClientOrderID: "4d96324ff9d44481926157", OrderID: 125690984230, OrderListID: -1, ClientOrderID: "91fe37ce9e69c90d6358c0",
		Price: 23450, OriginalQuantity: 0.00847, ExecutedQuantity: 0.00001, CumulativeQuoteQuantity: 0.2345, Status: "CANCELED",
		TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimit, Side: SideSell, SelfTradePreventionMode: SelfTradePreventionModeNone,
	}
	placed := func(workingTime int64) CancelReplaceNewOrderResponse {
		return CancelReplaceNewOrderResponse{
			Symbol: "BTCUSDT", OrderID: 12569099453, OrderListID: -1, ClientOrderID: "bX5wROblo6YeDwa9iTLeyY", TransactTime: types.Time(time.UnixMilli(1660813156959)),
			Price: 23416.1, OriginalQuantity: 0.00847, Status: "NEW", TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimit, Side: SideSell,
			WorkingTime: workingTime, SelfTradePreventionMode: SelfTradePreventionModeNone, Fills: []OrderFill{},
		}
	}

	t.Run("returns both halves' outcome with a partial failure", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("a partial failure cancels or places a real order")
		}
		ex := newWebsocketAPITestExchange(t, true)
		for _, tc := range []struct {
			name string
			req  *CancelReplaceOrderRequest
			exp  *CancelAndReplaceResponse
		}{
			{
				name: "failed cancellation",
				req:  &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, CancelReplaceMode: CancelReplaceModeAllowFailure, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.04, Price: 70000, CancelOrderID: 9},
				exp: &CancelAndReplaceResponse{
					CancelResult:     "FAILURE",
					NewOrderResult:   "SUCCESS",
					CancelResponse:   CancelReplaceCancelResponse{Code: -2011, Message: "Unknown order sent."},
					NewOrderResponse: placed(1669693344508),
				},
			},
			{
				name: "failed new order",
				req:  &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimitMaker, CancelReplaceMode: CancelReplaceModeAllowFailure, Quantity: 0.04, Price: 70000, CancelOrderID: 9},
				exp: &CancelAndReplaceResponse{
					CancelResult:     "SUCCESS",
					NewOrderResult:   "FAILURE",
					CancelResponse:   cancelled,
					NewOrderResponse: CancelReplaceNewOrderResponse{Code: -2010, Message: "Order would immediately match and take."},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				result, err := ex.WebsocketAPI().CancelExistingOrderAndSendNewOrder(t.Context(), tc.req)
				require.ErrorIs(t, err, errAPIResponse, "CancelExistingOrderAndSendNewOrder must return the API error")
				assert.ErrorIs(t, err, errWebsocketAPIRequestFailed, "CancelExistingOrderAndSendNewOrder should report a failed request")
				assert.ErrorContains(t, err, "code -2021", "CancelExistingOrderAndSendNewOrder should report the partial failure's code")
				assert.Equal(t, tc.exp, result, "CancelExistingOrderAndSendNewOrder should decode both halves from the error's data")
			})
		}
	})

	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, CancelReplaceMode: CancelReplaceModeStopOnFailure, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.04, Price: 70000, CancelOrderID: 9})
	require.NoError(t, err, "CancelExistingOrderAndSendNewOrder must not error")
	if !mockTests {
		return
	}
	exp := &CancelAndReplaceResponse{
		CancelResult:     "SUCCESS",
		NewOrderResult:   "SUCCESS",
		CancelResponse:   cancelled,
		NewOrderResponse: placed(1660813156959),
	}
	assert.Equal(t, exp, result, "CancelExistingOrderAndSendNewOrder should decode every field")
}

func TestWebsocketAPIGetPreventedMatches(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), PreventedMatchID: 1})
	require.NoError(t, err, "GetPreventedMatches must not error")
	if !mockTests {
		return
	}
	exp := []PreventedMatchResponse{{
		Symbol:                  "BTCUSDT",
		PreventedMatchID:        1,
		TakerOrderID:            5,
		MakerOrderID:            3,
		TradeGroupID:            1,
		SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
		Price:                   1.1,
		MakerPreventedQuantity:  1.3,
		TransactTime:            types.Time(time.UnixMilli(1669101687094)),
	}}
	assert.Equal(t, exp, result, "GetPreventedMatches should decode every field")
}

func TestWebsocketAPIAllOrders(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().AllOrders(t.Context(), &AllOrdersRequest{Symbol: currency.NewBTCUSDT()})
	require.NoError(t, err, "AllOrders must not error")
	if !mockTests {
		return
	}
	exp := []TradeOrderResponse{{
		Symbol:                  "BTCUSDT",
		OrderID:                 12569099453,
		OrderListID:             -1,
		ClientOrderID:           "4d96324ff9d44481926157",
		Price:                   23416.1,
		OriginalQuantity:        0.00847,
		ExecutedQuantity:        0.00847,
		CumulativeQuoteQuantity: 198.335215,
		Status:                  "FILLED",
		TimeInForce:             TimeInForceGoodTillCancel,
		Type:                    OrderTypeLimit,
		Side:                    SideSell,
		Time:                    types.Time(time.UnixMilli(1660801715639)),
		UpdateTime:              types.Time(time.UnixMilli(1660801717945)),
		IsWorking:               true,
		WorkingTime:             1660801715639,
		SelfTradePreventionMode: SelfTradePreventionModeNone,
		PreventedQuantity:       1.2,
	}}
	assert.Equal(t, exp, result, "AllOrders should decode every field")
}

func TestWebsocketAPINewOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	offline, cancel := context.WithCancel(t.Context())
	cancel()
	neither := &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Quantity: 0.002, Price: 80000}
	_, err := e.WebsocketAPI().NewOneCancelsTheOtherOrder(offline, neither)
	require.ErrorIs(t, err, errStopPriceOrTrailingDeltaRequired, "NewOneCancelsTheOtherOrder must require a stop price or a trailing delta over the WebSocket API")
	trailingDelta := &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Quantity: 0.002, Price: 80000, TrailingDeltaInBasisPoints: 100}
	_, err = e.NewOneCancelsTheOtherOrder(offline, trailingDelta)
	require.ErrorIs(t, err, errStopPriceRequired, "NewOneCancelsTheOtherOrder must still require a stop price over REST")

	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().NewOneCancelsTheOtherOrder(t.Context(), &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), ListClientOrderID: "gct-oco-1", Side: SideSell, Quantity: 0.002, Price: 80000, StopPrice: 70000, StopLimitPrice: 69900, StopLimitTimeInForce: TimeInForceGoodTillCancel})
	require.NoError(t, err, "NewOneCancelsTheOtherOrder must not error")
	trailingResult, err := ex.WebsocketAPI().NewOneCancelsTheOtherOrder(t.Context(), trailingDelta)
	require.NoError(t, err, "NewOneCancelsTheOtherOrder must take a trailing delta in place of a stop price over the WebSocket API")
	if !mockTests {
		return
	}
	stopLeg := OrderResponse{
		Symbol: "BTCUSDT", OrderID: 12569138901, OrderListID: 1274512, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU", TransactTime: types.Time(time.UnixMilli(1660801713793)),
		Price: 23410, OriginalQuantity: 0.0065, Status: "NEW", TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeStopLossLimit, Side: SideSell,
		WorkingTime: -1, SelfTradePreventionMode: SelfTradePreventionModeNone, StopPrice: 23405,
	}
	limitLeg := OrderResponse{
		Symbol: "BTCUSDT", OrderID: 12569138902, OrderListID: 1274512, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us", TransactTime: types.Time(time.UnixMilli(1660801713793)),
		Price: 23420, OriginalQuantity: 0.0065, Status: "NEW", TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimitMaker, Side: SideSell,
		WorkingTime: 1660801713793, SelfTradePreventionMode: SelfTradePreventionModeNone,
	}
	orderList := func(reports ...OrderResponse) *OneCancelsTheOtherOrderResponse {
		return &OneCancelsTheOtherOrderResponse{
			OrderListID:       1274512,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "08985fedd9ea2cf6b28996",
			TransactionTime:   types.Time(time.UnixMilli(1660801713793)),
			Symbol:            "BTCUSDT",
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 12569138901, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU"}, {Symbol: "BTCUSDT", OrderID: 12569138902, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us"}},
			OrderReports:      reports,
		}
	}
	assert.Equal(t, orderList(stopLeg, limitLeg), result, "NewOneCancelsTheOtherOrder should decode every field")

	partiallyFilledLimitLeg := limitLeg
	partiallyFilledLimitLeg.ExecutedQuantity = 0.001
	partiallyFilledLimitLeg.OriginalQuoteOrderQuantity = 25
	partiallyFilledLimitLeg.CumulativeQuoteQuantity = 23.42
	partiallyFilledLimitLeg.Status = "PARTIALLY_FILLED"
	partiallyFilledLimitLeg.Fills = []OrderFill{{Price: 23420, Quantity: 0.001, CommissionAsset: currency.BNB, TradeID: 1650422483}}
	assert.Equal(t, orderList(stopLeg, partiallyFilledLimitLeg), trailingResult, "NewOneCancelsTheOtherOrder should decode every field, the order reports' fills included")
}

func TestWebsocketAPIGetOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{OrderListID: new(uint64(27))})
	if !mockTests {
		require.ErrorIs(t, err, errAPIResponse, "GetOneCancelsTheOtherOrders must report an order list the account does not have")
		return
	}
	require.NoError(t, err, "GetOneCancelsTheOtherOrders must not error")
	exp := &OneCancelsTheOtherOrderResponse{
		OrderListID:       1274512,
		ContingencyType:   "OCO",
		ListStatusType:    "EXEC_STARTED",
		ListOrderStatus:   "EXECUTING",
		ListClientOrderID: "08985fedd9ea2cf6b28996",
		TransactionTime:   types.Time(time.UnixMilli(1660801713793)),
		Symbol:            "BTCUSDT",
		Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 12569138901, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU"}, {Symbol: "BTCUSDT", OrderID: 12569138902, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us"}},
	}
	assert.Equal(t, exp, result, "GetOneCancelsTheOtherOrders should decode every field")
}

func TestWebsocketAPIGetAllOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetAllOneCancelsTheOtherOrders(t.Context(), nil)
	require.NoError(t, err, "GetAllOneCancelsTheOtherOrders must not error")
	if !mockTests {
		return
	}
	exp := []OneCancelsTheOtherOrderResponse{{
		OrderListID:       1274512,
		ContingencyType:   "OCO",
		ListStatusType:    "EXEC_STARTED",
		ListOrderStatus:   "EXECUTING",
		ListClientOrderID: "08985fedd9ea2cf6b28996",
		TransactionTime:   types.Time(time.UnixMilli(1660801713793)),
		Symbol:            "BTCUSDT",
		Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 12569138901, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU"}, {Symbol: "BTCUSDT", OrderID: 12569138902, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us"}},
	}}
	assert.Equal(t, exp, result, "GetAllOneCancelsTheOtherOrders should decode every field")
}

func TestWebsocketAPIGetOpenOneCancelsTheOtherList(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetOpenOneCancelsTheOtherList(t.Context())
	require.NoError(t, err, "GetOpenOneCancelsTheOtherList must not error")
	if !mockTests {
		return
	}
	exp := []OneCancelsTheOtherOrderResponse{{
		ContingencyType:   "OCO",
		ListStatusType:    "EXEC_STARTED",
		ListOrderStatus:   "EXECUTING",
		ListClientOrderID: "08985fedd9ea2cf6b28996",
		TransactionTime:   types.Time(time.UnixMilli(1660801713793)),
		Symbol:            "BTCUSDT",
		Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 4, ClientOrderID: "CUhLgTXnX5n2c0gWiLpV4d"}, {Symbol: "BTCUSDT", OrderID: 5, ClientOrderID: "1ZqG7bBuYwaF4SU8CwnwHm"}},
	}}
	assert.Equal(t, exp, result, "GetOpenOneCancelsTheOtherList should decode every field")
}

func TestWebsocketAPICancelOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true, canManipulateRealOrders)
	result, err := ex.WebsocketAPI().CancelOneCancelsTheOtherOrder(t.Context(), &CancelOrderListRequest{Symbol: currency.NewBTCUSDT(), ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN"})
	require.NoError(t, err, "CancelOneCancelsTheOtherOrder must not error")
	if !mockTests {
		return
	}
	report := func(orderID uint64, clientOrderID string, orderType OrderType, price, stopPrice float64) OrderResponse {
		return OrderResponse{
			Symbol: "BTCUSDT", OrderID: orderID, OrderListID: 1274512, ClientOrderID: clientOrderID, TransactTime: types.Time(time.UnixMilli(1660801720215)),
			Price: types.Number(price), OriginalQuantity: 0.0065, Status: "CANCELED", TimeInForce: TimeInForceGoodTillCancel, Type: orderType, Side: SideSell,
			SelfTradePreventionMode: SelfTradePreventionModeNone, StopPrice: types.Number(stopPrice),
		}
	}
	exp := &OneCancelsTheOtherOrderResponse{
		OrderListID:       1274512,
		ContingencyType:   "OCO",
		ListStatusType:    "ALL_DONE",
		ListOrderStatus:   "ALL_DONE",
		ListClientOrderID: "6023531d7edaad348f5aff",
		TransactionTime:   types.Time(time.UnixMilli(1660801720215)),
		Symbol:            "BTCUSDT",
		Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 12569138901, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU"}, {Symbol: "BTCUSDT", OrderID: 12569138902, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us"}},
		OrderReports: []OrderResponse{
			report(12569138901, "BqtFCj5odMoWtSqGk2X9tU", OrderTypeStopLossLimit, 23410, 23405),
			report(12569138902, "jLnZpj5enfMXTuhKB1d0us", OrderTypeLimitMaker, 23420, 0),
		},
	}
	assert.Equal(t, exp, result, "CancelOneCancelsTheOtherOrder should decode every field")
}
