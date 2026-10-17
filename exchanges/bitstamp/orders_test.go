package bitstamp

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSetDerivativesOrderParams(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setDerivativesOrderParams(params, &derivativesOrderParams{})
	assert.Empty(t, params, "setDerivativesOrderParams should not set unset parameters")

	setDerivativesOrderParams(params, &derivativesOrderParams{
		ClientOrderID:   "my-order-123",
		Subtype:         OrderSubtypeTrailingStopLoss,
		MarginMode:      MarginModeIsolated,
		Leverage:        5,
		StopPrice:       19000.5,
		Trigger:         TriggerMarkPrice,
		ActivationPrice: 21000,
		TrailingDelta:   150,
		ReduceOnly:      true,
	})
	assert.Equal(t, url.Values{
		"client_order_id":  {"my-order-123"},
		"subtype":          {OrderSubtypeTrailingStopLoss},
		"margin_mode":      {MarginModeIsolated},
		"leverage":         {"5"},
		"stop_price":       {"19000.5"},
		"trigger":          {TriggerMarkPrice},
		"activation_price": {"21000"},
		"trailing_delta":   {"150"},
		"reduce_only":      {trueValue},
	}, params, "setDerivativesOrderParams should set every parameter")
}

func TestOrderEventParams(t *testing.T) {
	t.Parallel()
	_, err := orderEventParams(&OrderEventsRequest{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "orderEventParams should error on an empty pair")

	params, err := orderEventParams(&OrderEventsRequest{Pair: perpetualPair})
	require.NoError(t, err, "orderEventParams must not error")
	assert.Equal(t, url.Values{"market": {"BTC/USD-PERP"}}, params, "orderEventParams should set the market")

	params, err = orderEventParams(&OrderEventsRequest{Pair: spotPair, SinceID: "019f4ac8-5e3a-11ef-af00-3e9012000020", UntilID: "019f4ac8-5e3b-11ef-af00-3e9012000021"})
	require.NoError(t, err, "orderEventParams must not error with event IDs")
	assert.Equal(t, url.Values{
		"market":   {"BTC/USD"},
		"since_id": {"019f4ac8-5e3a-11ef-af00-3e9012000020"},
		"until_id": {"019f4ac8-5e3b-11ef-af00-3e9012000021"},
	}, params, "orderEventParams should set the event ID range")
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderStatus(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetOrderStatus should error on a nil request")
	_, err = e.GetOrderStatus(t.Context(), &OrderStatusRequest{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderStatus should error without an order ID")

	skipIfLiveWithoutCredentials(t)
	status, err := e.GetOrderStatus(t.Context(), &OrderStatusRequest{OrderID: 1458532827766784})
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "GetOrderStatus should error for an unknown order")
		return
	}
	require.NoError(t, err, "GetOrderStatus must not error")
	exp := &OrderStatusResponse{
		ID:       1458532827766784,
		DateTime: types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC)),
		Side:     OrderSide(order.Buy),
		Status:   "Open",
		Market:   "BTC/USD",
		Transactions: []OrderTransaction{
			{TradeID: 209895701, Price: 20000, Fee: 8, DateTime: time.Date(2022, 1, 31, 14, 45, 15, 322000000, time.UTC), Type: TransactionTypeMarketTrade, Amounts: map[currency.Code]float64{currency.USD: -2000, currency.BTC: 0.1}},
			{TradeID: 209895702, Price: 20100, Fee: 8.04, DateTime: time.Date(2022, 1, 31, 14, 46, 15, 0, time.UTC), Type: TransactionTypeMarketTrade, Amounts: map[currency.Code]float64{currency.USD: -2010, currency.BTC: 0.1}},
		},
		AmountRemaining: 0.3,
		ClientOrderID:   "my-order-123",
	}
	assert.Equal(t, exp, status, "GetOrderStatus should return the order status")

	status, err = e.GetOrderStatus(t.Context(), &OrderStatusRequest{ClientOrderID: "my-order-123", OmitTransactions: true})
	require.NoError(t, err, "GetOrderStatus must not error with a client order ID")
	exp.Transactions = []OrderTransaction{}
	assert.Equal(t, exp, status, "GetOrderStatus should return the order status without transactions")
}

func TestGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderData(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetOrderData should error on a nil request")
	_, err = e.GetOrderData(t.Context(), &OrderEventsRequest{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderData should error on an empty pair")

	skipIfLiveWithoutCredentials(t)
	events, err := e.GetOrderData(t.Context(), &OrderEventsRequest{Pair: spotPair})
	require.NoError(t, err, "GetOrderData must not error")
	if mockTests {
		exp := []OrderEventResponse{{
			Event:       orderEventCreated,
			EventID:     "019f4ac8-1234-abcd-af00-3e9012000020",
			OrderSource: OrderSourceOrderbook,
			Data: OrderEventData{
				ID:              1458532827766784,
				Side:            OrderSide(order.Buy),
				DateTime:        types.Time(time.Unix(1643698764, 0)),
				Microtimestamp:  types.Time(time.UnixMicro(1643698764000000)),
				RemainingAmount: 0.5,
				AmountAtCreate:  0.5,
				Price:           50000,
			},
		}}
		assert.Equal(t, exp, events, "GetOrderData should decode the events")
	}
}

func TestGetAccountOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountOrderData(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetAccountOrderData should error on a nil request")
	_, err = e.GetAccountOrderData(t.Context(), &OrderEventsRequest{Pair: perpetualPair})
	assert.ErrorIs(t, err, errOrderSourceRequired, "GetAccountOrderData should error without an order source")
	_, err = e.GetAccountOrderData(t.Context(), &OrderEventsRequest{OrderSource: OrderSourceOrderbook})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAccountOrderData should error on an empty pair")

	skipIfLiveWithoutCredentials(t)
	events, err := e.GetAccountOrderData(t.Context(), &OrderEventsRequest{Pair: perpetualPair, OrderSource: OrderSourceOrderbook, SinceID: "019f4ac8-1234-abcd-af00-3e9012000020"})
	require.NoError(t, err, "GetAccountOrderData must not error")
	if mockTests {
		require.Len(t, events, 1, "GetAccountOrderData must return the events")
		exp := OrderEventResponse{
			Event:       "order_replaced",
			EventID:     "019f4ac8-1234-abcd-af00-3e9012000021",
			OrderSource: OrderSourceOrderbook,
			Data: OrderEventData{
				ID:              1458532827766786,
				Side:            OrderSide(order.Sell),
				OrderSubtype:    22,
				DateTime:        types.Time(time.Unix(1643698765, 0)),
				Microtimestamp:  types.Time(time.UnixMicro(1643698765000000)),
				RemainingAmount: 0.5,
				AmountAtCreate:  1,
				Price:           50000,
				ClientOrderID:   "my-order-123",
				OriginalOrderID: 1458532827766700,
				ReduceOnly:      true,
				StopPrice:       48000,
				ActivationPrice: 49000,
				TrailingDelta:   50,
			},
		}
		assert.Equal(t, exp, events[0], "GetAccountOrderData should return the event")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	for _, pair := range []currency.Pair{currency.EMPTYPAIR, spotPair, perpetualPair} {
		orders, err := e.GetOpenOrders(t.Context(), pair)
		require.NoErrorf(t, err, "GetOpenOrders must not error for %q", pair)
		if !mockTests {
			continue
		}
		require.NotEmptyf(t, orders, "GetOpenOrders must return orders for %q", pair)
		for i := range orders {
			o := &orders[i]
			switch o.Market {
			case "BTC/USD":
				exp := OpenOrderResponse{
					ID:              1234123412341234,
					DateTime:        types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC)),
					Side:            OrderSide(order.Buy),
					Price:           100,
					RemainingAmount: 0.4,
					AmountAtCreate:  0.5,
					Market:          "BTC/USD",
					LimitPrice:      110,
					ClientOrderID:   "my-order-123",
				}
				assert.Equal(t, exp, *o, "spot order should be correct")
			case "BTC/USD-PERP":
				exp := OpenOrderResponse{
					ID:              1234123412341235,
					DateTime:        types.DateTime(time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC)),
					Side:            OrderSide(order.Sell),
					Subtype:         OrderSubtypeStopLossLimit,
					Price:           85000,
					RemainingAmount: 0.01,
					AmountAtCreate:  0.01,
					Market:          "BTC/USD-PERP",
					MarginMode:      MarginModeCross,
					Leverage:        3,
					ReservedMargin:  283.33,
					StopPrice:       84500,
					Trigger:         TriggerLastTradedPrice,
					ReduceOnly:      true,
				}
				assert.Equal(t, exp, *o, "perpetual order should be correct")
			default:
				assert.Failf(t, "unexpected market", "GetOpenOrders returned market %q", o.Market)
			}
		}
	}
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *LimitOrderRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &LimitOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &LimitOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, Amount: -1}, err: order.ErrAmountIsInvalid},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: -1}, err: errInvalidPrice},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, GoodTillDate: true}, err: common.ErrDateUnset},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, DailyOrder: true, ImmediateOrCancel: true}, err: errOrderFlagsConflict},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, FillOrKill: true, MakerOrCancel: true}, err: errOrderFlagsConflict},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: 2, DailyOrder: true}, err: errLimitPriceConflict},
		{req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: 2, GoodTillDate: true, ExpireTime: time.Now()}, err: errLimitPriceConflict},
	} {
		_, err := e.PlaceLimitOrder(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "PlaceLimitOrder should return the correct error")
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.PlaceLimitOrder(t.Context(), &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 0.5, Price: 20000, ClientOrderID: "123456789"})
	require.NoError(t, err, "PlaceLimitOrder must not error")
	if mockTests {
		exp := &OrderResponse{
			ID:            1234123412341236,
			Market:        "BTC/USD",
			DateTime:      types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 796000000, time.UTC)),
			Side:          OrderSide(order.Buy),
			Price:         20000,
			Amount:        0.5,
			ClientOrderID: "123456789",
		}
		assert.Equal(t, exp, resp, "PlaceLimitOrder should return the order")

		resp, err = e.PlaceLimitOrder(t.Context(), &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 0.5, Price: 20000, GoodTillDate: true, ExpireTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		require.NoError(t, err, "PlaceLimitOrder must not error for a good till date order")
		exp.ID, exp.ClientOrderID = 1234123412341237, ""
		assert.Equal(t, exp, resp, "PlaceLimitOrder should return the good till date order")

		resp, err = e.PlaceLimitOrder(t.Context(), &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 0.25, Price: 20000, LimitPrice: 21000})
		require.NoError(t, err, "PlaceLimitOrder must not error with a limit price")
		exp.ID, exp.Amount = 1234123412341245, 0.25
		assert.Equal(t, exp, resp, "PlaceLimitOrder should return the order with a limit price")
	}
}

func TestPlaceInstantOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *InstantOrderRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &InstantOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &InstantOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy}, err: order.ErrAmountIsInvalid},
		{req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 1, Leverage: -1}, err: errInvalidLeverage},
		{req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 1, ReduceOnly: true}, err: errReduceOnlyUnsupported},
		{req: &InstantOrderRequest{Pair: spotPair, Side: order.Sell, Amount: 1, AmountInCounter: true, ReduceOnly: true}, err: errReduceOnlyUnsupported},
	} {
		_, err := e.PlaceInstantOrder(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "PlaceInstantOrder should return the correct error")
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	for side, id := range map[order.Side]uint64{order.Buy: 1234123412341239, order.Sell: 1234123412341240} {
		resp, err := e.PlaceInstantOrder(t.Context(), &InstantOrderRequest{Pair: spotPair, Side: side, Amount: 100, AmountInCounter: true})
		require.NoErrorf(t, err, "PlaceInstantOrder must not error for %s", side)
		if mockTests {
			exp := &OrderResponse{
				ID:       id,
				Market:   "BTC/USD",
				DateTime: types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 796000000, time.UTC)),
				Side:     OrderSide(side),
				Price:    84000,
				Amount:   0.00119048,
			}
			assert.Equalf(t, exp, resp, "PlaceInstantOrder should return the %s order", side)
		}
	}
}

func TestPlaceMarketOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *MarketOrderRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &MarketOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &MarketOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{req: &MarketOrderRequest{Pair: spotPair, Side: order.Buy, Amount: -1}, err: order.ErrAmountIsInvalid},
	} {
		_, err := e.PlaceMarketOrder(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "PlaceMarketOrder should return the correct error")
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.PlaceMarketOrder(t.Context(), &MarketOrderRequest{Pair: spotPair, Side: order.Sell, Amount: 0.1})
	require.NoError(t, err, "PlaceMarketOrder must not error")
	if mockTests {
		exp := &OrderResponse{
			ID:       1234123412341238,
			Market:   "BTC/USD",
			DateTime: types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 796000000, time.UTC)),
			Side:     OrderSide(order.Sell),
			Price:    84000,
			Amount:   0.1,
		}
		assert.Equal(t, exp, resp, "PlaceMarketOrder should return the order")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrder should error on a nil request")
	_, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelExistingOrder should error without an order ID")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.CancelExistingOrder(t.Context(), &CancelOrderRequest{OrderID: 1453282316578816})
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "CancelExistingOrder should error for an unknown order")
		return
	}
	require.NoError(t, err, "CancelExistingOrder must not error")
	exp := &CancelOrderResponse{ID: 1453282316578816, Amount: 0.02035278, Price: 2100.45, Side: OrderSide(order.Buy), Market: "BTC/USD", Status: "Canceled"}
	assert.Equal(t, exp, resp, "CancelExistingOrder should return the cancelled order")

	resp, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{ClientOrderID: "my-order-123"})
	require.NoError(t, err, "CancelExistingOrder must not error with a client order ID")
	exp = &CancelOrderResponse{ID: 1453282316578817, Amount: 0.5, Price: 2100.45, Side: OrderSide(order.Sell), Market: "BTC/USD-PERP", Status: CancelStatusPending}
	assert.Equal(t, exp, resp, "CancelExistingOrder should return the order pending cancellation")
}

func TestCancelAllExistingOrders(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	spotOrder := CancelledOrder{ID: 1234123412341234, Amount: 0.4, Price: 100, Side: OrderSide(order.Buy), Market: "BTC/USD"}
	perpetualOrder := CancelledOrder{ID: 1234123412341235, Amount: 0.01, Price: 85000, Side: OrderSide(order.Sell), Market: "BTC/USD-PERP"}
	for pair, exp := range map[currency.Pair]*CancelAllOrdersResponse{
		currency.EMPTYPAIR: {Cancelled: []CancelledOrder{spotOrder, perpetualOrder}, Success: true},
		spotPair:           {Cancelled: []CancelledOrder{spotOrder}, Success: true},
	} {
		resp, err := e.CancelAllExistingOrders(t.Context(), pair)
		require.NoErrorf(t, err, "CancelAllExistingOrders must not error for %q", pair)
		if mockTests {
			assert.Equalf(t, exp, resp, "CancelAllExistingOrders should return the cancelled orders for %q", pair)
		}
	}
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *ReplaceOrderRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &ReplaceOrderRequest{}, err: order.ErrOrderIDNotSet},
		{req: &ReplaceOrderRequest{OrderID: 1}, err: order.ErrAmountIsInvalid},
		{req: &ReplaceOrderRequest{OrderID: 1, Amount: 1}, err: order.ErrPriceMustBeSetIfLimitOrder},
	} {
		_, err := e.ReplaceOrder(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "ReplaceOrder should return the correct error")
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.ReplaceOrder(t.Context(), &ReplaceOrderRequest{OrderID: 1453282316578816, Amount: 0.02, Price: 2100.45})
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "ReplaceOrder should error for an unknown order")
		return
	}
	require.NoError(t, err, "ReplaceOrder must not error")
	exp := &ReplaceOrderResponse{
		OrderID:         1453282316578817,
		Side:            OrderSide(order.Buy),
		Market:          "BTC/USD",
		Amount:          0.02,
		Price:           2100.45,
		DateTime:        DateTime(time.Date(2025, 10, 17, 14, 23, 1, 725000000, time.UTC)),
		OriginalOrderID: 1453282316578816,
		Status:          "Open",
	}
	assert.Equal(t, exp, resp, "ReplaceOrder should return the replacement order")

	resp, err = e.ReplaceOrder(t.Context(), &ReplaceOrderRequest{OriginalClientOrderID: "my-order-123", ClientOrderID: "my-order-456", Amount: 0.02, Price: 2100.45})
	require.NoError(t, err, "ReplaceOrder must not error with a client order ID")
	exp.OriginalClientOrderID = "my-order-123"
	assert.Equal(t, exp, resp, "ReplaceOrder should return the replacement order with a client order ID")
}

func TestGetTradingMarkets(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	markets, err := e.GetTradingMarkets(t.Context())
	require.NoError(t, err, "GetTradingMarkets must not error")
	require.NotEmpty(t, markets, "GetTradingMarkets must return markets")
	if mockTests {
		assert.Equal(t, TradingMarketResponse{Name: "BTC/USD", URLSymbol: "btcusd"}, markets[0], "market should be correct")
	}
}

func TestGetMaxOrderAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *MaxOrderAmountRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &MaxOrderAmountRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair}, err: errMarginModeRequired},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated}, err: errLeverageRequired},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3}, err: errOrderTypeRequired},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 26}, err: errInvalidLeverage},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit}, err: order.ErrSideIsInvalid},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit, Side: order.Buy, StopPrice: -1}, err: errInvalidPrice},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeTrailingStopLoss, Side: order.Buy, TrailingDelta: maxTrailingDelta + 1}, err: errInvalidTrailingDelta},
		{req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit, Side: order.Buy, AdditionalCollateral: map[currency.Code]float64{currency.USD: -1}}, err: errInvalidCollateral},
	} {
		_, err := e.GetMaxOrderAmount(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetMaxOrderAmount should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	resp, err := e.GetMaxOrderAmount(t.Context(), &MaxOrderAmountRequest{
		Pair:                 perpetualPair,
		MarginMode:           MarginModeIsolated,
		Leverage:             3,
		OrderType:            OrderSubtypeLimit,
		Side:                 order.Buy,
		Price:                84000,
		AdditionalCollateral: map[currency.Code]float64{currency.USD: 120},
	})
	require.NoError(t, err, "GetMaxOrderAmount must not error")
	if mockTests {
		exp := &MaxOrderAmountResponse{
			MaximumOrderAmount:         1.23456,
			MaximumOrderValue:          103703.04,
			MaximumOrderAmountCurrency: currency.BTC,
			MaximumOrderValueCurrency:  currency.USD,
		}
		assert.Equal(t, exp, resp, "GetMaxOrderAmount should decode the maximum order")
	}
}

func TestValidateDerivativesOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		pair currency.Pair
		p    *derivativesOrderParams
		err  error
	}{
		{pair: spotPair, p: &derivativesOrderParams{}},
		{pair: spotPair, p: &derivativesOrderParams{ClientOrderID: strings.Repeat("a", maxClientOrderIDLength)}},
		{pair: spotPair, p: &derivativesOrderParams{ClientOrderID: strings.Repeat("a", maxClientOrderIDLength+1)}, err: errClientOrderIDTooLong},
		{pair: perpetualPair, p: &derivativesOrderParams{Subtype: OrderSubtypeStopLoss, MarginMode: MarginModeCross, Leverage: 2}, err: errInvalidOrderSubtype},
		{pair: spotPair, p: &derivativesOrderParams{Leverage: -1}, err: errInvalidLeverage},
		{pair: perpetualPair, p: &derivativesOrderParams{Leverage: 2}, err: errMarginModeRequired},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross}, err: errLeverageRequired},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, StopPrice: -1}, err: errInvalidPrice},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, ActivationPrice: -1}, err: errInvalidPrice},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, TrailingDelta: maxTrailingDelta + 1}, err: errInvalidTrailingDelta},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, ActivationPrice: 1}, err: errTrailingDeltaRequired},
		{pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, Subtype: OrderSubtypeTrailingStopLossLimit, ActivationPrice: 1, TrailingDelta: maxTrailingDelta}},
	} {
		assert.ErrorIsf(t, validateDerivativesOrderParams(tc.pair, tc.p, limitOrderSubtypes), tc.err, "validateDerivativesOrderParams should return the correct error for %s %+v", tc.pair, tc.p)
	}
}

func TestValidateLeverage(t *testing.T) {
	t.Parallel()
	for leverage, exp := range map[float64]error{0: errLeverageRequired, 0.5: errInvalidLeverage, 1: nil, 25: nil, 25.5: errInvalidLeverage} {
		assert.ErrorIsf(t, validateLeverage(leverage), exp, "validateLeverage should return the correct error for %v", leverage)
	}
}

func TestValidateCollateral(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateCollateral(nil), "validateCollateral should not error without collateral")
	assert.NoError(t, validateCollateral(map[currency.Code]float64{currency.USD: 0, currency.BTC: 1}), "validateCollateral should not error on amounts which are not negative")
	assert.ErrorIs(t, validateCollateral(map[currency.Code]float64{currency.BTC: 1, currency.USD: -1}), errInvalidCollateral, "validateCollateral should error on a negative amount")
}
