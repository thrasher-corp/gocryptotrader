package bitstamp

import (
	"net/http"
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
				RemainingAmount: preciseNumber(t, "0.5"),
				AmountAtCreate:  preciseNumber(t, "0.5"),
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
				RemainingAmount: preciseNumber(t, "0.5"),
				AmountAtCreate:  preciseNumber(t, "1.0"),
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

func TestGetOpenOrders(t *testing.T) { //nolint:tparallel // Cases run serially through the shared exchange
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	for _, tc := range []struct {
		name string
		pair currency.Pair
	}{
		{name: "all markets", pair: currency.EMPTYPAIR},
		{name: "spot market", pair: spotPair},
		{name: "perpetual market", pair: perpetualPair},
	} {
		// Serial and in order, as every case goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			orders, err := e.GetOpenOrders(t.Context(), tc.pair)
			require.NoErrorf(t, err, "GetOpenOrders must not error for %q", tc.pair)
			if !mockTests {
				return
			}
			require.NotEmptyf(t, orders, "GetOpenOrders must return orders for %q", tc.pair)
			for i := range orders {
				o := &orders[i]
				switch o.Market {
				case "BTC/USD":
					exp := OpenOrderResponse{
						ID:              1234123412341234,
						DateTime:        types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC)),
						Side:            OrderSide(order.Buy),
						Price:           100,
						RemainingAmount: preciseNumber(t, "0.40000000"),
						AmountAtCreate:  preciseNumber(t, "0.50000000"),
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
						RemainingAmount: preciseNumber(t, "0.01000"),
						AmountAtCreate:  preciseNumber(t, "0.01000"),
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
					assert.Failf(t, "GetOpenOrders should return only the mocked markets", "GetOpenOrders returned market %q", o.Market)
				}
			}
		})
	}
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *LimitOrderRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &LimitOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without a side", req: &LimitOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{name: "without a price", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "negative amount", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, Amount: -1}, err: order.ErrAmountIsInvalid},
		{name: "negative limit price", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: -1}, err: errInvalidPrice},
		{name: "good till date without an expiry time", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, GoodTillDate: true}, err: common.ErrDateUnset},
		{name: "daily and immediate or cancel", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, DailyOrder: true, ImmediateOrCancel: true}, err: errOrderFlagsConflict},
		{name: "fill or kill and maker or cancel", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, FillOrKill: true, MakerOrCancel: true}, err: errOrderFlagsConflict},
		{name: "limit price with a daily order", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: 2, DailyOrder: true}, err: errLimitPriceConflict},
		{name: "limit price with a good till date order", req: &LimitOrderRequest{Pair: spotPair, Side: order.Buy, Price: 1, LimitPrice: 2, GoodTillDate: true, ExpireTime: time.Now()}, err: errLimitPriceConflict},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.PlaceLimitOrder(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "PlaceLimitOrder should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
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
	})
}

func TestPlaceInstantOrder(t *testing.T) { //nolint:tparallel // Cases run serially through the shared exchange
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *InstantOrderRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &InstantOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without a side", req: &InstantOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{name: "without an amount", req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy}, err: order.ErrAmountIsInvalid},
		{name: "negative leverage", req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 1, Leverage: -1}, err: errInvalidLeverage},
		{name: "reduce only buy", req: &InstantOrderRequest{Pair: spotPair, Side: order.Buy, Amount: 1, ReduceOnly: true}, err: errReduceOnlyUnsupported},
		{name: "reduce only sell in the counter currency", req: &InstantOrderRequest{Pair: spotPair, Side: order.Sell, Amount: 1, AmountInCounter: true, ReduceOnly: true}, err: errReduceOnlyUnsupported},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.PlaceInstantOrder(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "PlaceInstantOrder should return the correct error")
		})
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	for side, id := range map[order.Side]uint64{order.Buy: 1234123412341239, order.Sell: 1234123412341240} {
		// Serial, as both orders go through the same exchange
		t.Run(side.Lower(), func(t *testing.T) {
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
		})
	}
}

func TestPlaceMarketOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *MarketOrderRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &MarketOrderRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without a side", req: &MarketOrderRequest{Pair: spotPair}, err: order.ErrSideIsInvalid},
		{name: "negative amount", req: &MarketOrderRequest{Pair: spotPair, Side: order.Buy, Amount: -1}, err: order.ErrAmountIsInvalid},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.PlaceMarketOrder(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "PlaceMarketOrder should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
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
	})
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

func TestCancelAllExistingOrders(t *testing.T) { //nolint:tparallel // Cases run serially through the shared exchange
	t.Parallel()
	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	spotOrder := CancelledOrder{ID: 1234123412341234, Amount: 0.4, Price: 100, Side: OrderSide(order.Buy), Market: "BTC/USD"}
	perpetualOrder := CancelledOrder{ID: 1234123412341235, Amount: 0.01, Price: 85000, Side: OrderSide(order.Sell), Market: "BTC/USD-PERP"}
	for name, tc := range map[string]struct {
		pair currency.Pair
		exp  *CancelAllOrdersResponse
	}{
		"all markets": {pair: currency.EMPTYPAIR, exp: &CancelAllOrdersResponse{Cancelled: []CancelledOrder{spotOrder, perpetualOrder}, Success: true}},
		"spot market": {pair: spotPair, exp: &CancelAllOrdersResponse{Cancelled: []CancelledOrder{spotOrder}, Success: true}},
	} {
		// Serial, as both cancellations go through the same exchange
		t.Run(name, func(t *testing.T) {
			resp, err := e.CancelAllExistingOrders(t.Context(), tc.pair)
			require.NoErrorf(t, err, "CancelAllExistingOrders must not error for %q", tc.pair)
			if mockTests {
				assert.Equalf(t, tc.exp, resp, "CancelAllExistingOrders should return the cancelled orders for %q", tc.pair)
			}
		})
	}
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ReplaceOrderRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without an order ID", req: &ReplaceOrderRequest{}, err: order.ErrOrderIDNotSet},
		{name: "without an amount", req: &ReplaceOrderRequest{OrderID: 1}, err: order.ErrAmountIsInvalid},
		{name: "without a price", req: &ReplaceOrderRequest{OrderID: 1, Amount: 1}, err: order.ErrPriceMustBeSetIfLimitOrder},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.ReplaceOrder(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "ReplaceOrder should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
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
	})
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
		name string
		req  *MaxOrderAmountRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &MaxOrderAmountRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without a margin mode", req: &MaxOrderAmountRequest{Pair: perpetualPair}, err: errMarginModeRequired},
		{name: "without a leverage", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated}, err: errLeverageRequired},
		{name: "without an order type", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3}, err: errOrderTypeRequired},
		{name: "leverage above the maximum", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 26}, err: errInvalidLeverage},
		{name: "without a side", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit}, err: order.ErrSideIsInvalid},
		{name: "negative stop price", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit, Side: order.Buy, StopPrice: -1}, err: errInvalidPrice},
		{name: "trailing delta above the maximum", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeTrailingStopLoss, Side: order.Buy, TrailingDelta: maxTrailingDelta + 1}, err: errInvalidTrailingDelta},
		{name: "negative collateral", req: &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeLimit, Side: order.Buy, AdditionalCollateral: map[currency.Code]float64{currency.USD: -1}}, err: errInvalidCollateral},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetMaxOrderAmount(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetMaxOrderAmount should return the correct error")
		})
	}
	t.Run("trailing delta at the maximum", func(t *testing.T) {
		t.Parallel()
		ex := newResponseTestExchange(t, http.StatusOK, `{"maximum_order_amount":"1","maximum_order_value":"1"}`)
		_, err := ex.GetMaxOrderAmount(t.Context(), &MaxOrderAmountRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 3, OrderType: OrderSubtypeTrailingStopLoss, Side: order.Buy, TrailingDelta: maxTrailingDelta})
		require.NoError(t, err, "GetMaxOrderAmount must accept the maximum trailing delta")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
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
	})
}

func TestValidateDerivativesOrderParams(t *testing.T) { //nolint:tparallel // Cases run serially, as they share the package subtype list
	t.Parallel()
	for _, tc := range []struct {
		name string
		pair currency.Pair
		p    *derivativesOrderParams
		err  error
	}{
		{name: "spot pair without parameters", pair: spotPair, p: &derivativesOrderParams{}},
		{name: "client order ID at the maximum length", pair: spotPair, p: &derivativesOrderParams{ClientOrderID: strings.Repeat("a", maxClientOrderIDLength)}},
		{name: "client order ID above the maximum length", pair: spotPair, p: &derivativesOrderParams{ClientOrderID: strings.Repeat("a", maxClientOrderIDLength+1)}, err: errClientOrderIDTooLong},
		{name: "unsupported subtype", pair: perpetualPair, p: &derivativesOrderParams{Subtype: OrderSubtypeStopLoss, MarginMode: MarginModeCross, Leverage: 2}, err: errInvalidOrderSubtype},
		{name: "negative leverage", pair: spotPair, p: &derivativesOrderParams{Leverage: -1}, err: errInvalidLeverage},
		{name: "perpetual pair without a margin mode", pair: perpetualPair, p: &derivativesOrderParams{Leverage: 2}, err: errMarginModeRequired},
		{name: "perpetual pair without a leverage", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross}, err: errLeverageRequired},
		{name: "negative stop price", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, StopPrice: -1}, err: errInvalidPrice},
		{name: "negative activation price", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, ActivationPrice: -1}, err: errInvalidPrice},
		{name: "trailing delta above the maximum", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, TrailingDelta: maxTrailingDelta + 1}, err: errInvalidTrailingDelta},
		{name: "activation price without a trailing delta", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, ActivationPrice: 1}, err: errTrailingDeltaRequired},
		{name: "trailing stop loss limit at the maximum trailing delta", pair: perpetualPair, p: &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, Subtype: OrderSubtypeTrailingStopLossLimit, ActivationPrice: 1, TrailingDelta: maxTrailingDelta}},
	} {
		// Serial and in order, as every case passes the same package subtype list
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, validateDerivativesOrderParams(tc.pair, tc.p, limitOrderSubtypes), tc.err, "validateDerivativesOrderParams should return the correct error for %s %+v", tc.pair, tc.p)
		})
	}
}

func TestValidateMarginParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		marginMode string
		leverage   float64
		err        error
	}{
		{name: "without a margin mode", leverage: 2, err: errMarginModeRequired},
		{name: "without a margin mode or leverage", err: errMarginModeRequired},
		{name: "without a leverage", marginMode: MarginModeCross, err: errLeverageRequired},
		{name: "negative leverage", marginMode: MarginModeIsolated, leverage: -1, err: errLeverageRequired},
		{name: "cross margin", marginMode: MarginModeCross, leverage: 2},
		{name: "isolated margin", marginMode: MarginModeIsolated, leverage: 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateMarginParams(tc.marginMode, tc.leverage), tc.err, "validateMarginParams should return the correct error")
		})
	}
}

func TestValidateClientOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		clientOrderID string
		err           error
	}{
		{name: "empty"},
		{name: "within the maximum length", clientOrderID: "my-order-123"},
		{name: "at the maximum length", clientOrderID: strings.Repeat("a", maxClientOrderIDLength)},
		{name: "above the maximum length", clientOrderID: strings.Repeat("a", maxClientOrderIDLength+1), err: errClientOrderIDTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateClientOrderID(tc.clientOrderID), tc.err, "validateClientOrderID should return the correct error")
		})
	}
}

func TestValidateLeverage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		leverage float64
		err      error
	}{
		{name: "zero", leverage: 0, err: errLeverageRequired},
		{name: "below the minimum", leverage: 0.5, err: errInvalidLeverage},
		{name: "at the minimum", leverage: 1, err: nil},
		{name: "at the maximum", leverage: 25, err: nil},
		{name: "above the maximum", leverage: 25.5, err: errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateLeverage(tc.leverage), tc.err, "validateLeverage should return the correct error for %v", tc.leverage)
		})
	}
}

func TestValidateCollateral(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateCollateral(nil), "validateCollateral should not error without collateral")
	assert.NoError(t, validateCollateral(map[currency.Code]float64{currency.USD: 0, currency.BTC: 1}), "validateCollateral should not error on amounts which are not negative")
	assert.ErrorIs(t, validateCollateral(map[currency.Code]float64{currency.BTC: 1, currency.USD: -1}), errInvalidCollateral, "validateCollateral should error on a negative amount")
}
