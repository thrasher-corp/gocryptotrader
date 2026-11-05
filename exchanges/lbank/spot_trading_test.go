package lbank

import (
	"context"
	"math"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PlaceOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "unknown type", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "buy_post", Price: 1, Amount: 1}, err: errInvalidOrderType},
		{name: "upper case type", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "BUY", Price: 1, Amount: 1}, err: errInvalidOrderType},
		{name: "limit without price", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "buy", Amount: 1}, err: limits.ErrPriceBelowMin},
		{name: "limit without amount", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "sell_maker", Price: 1}, err: limits.ErrAmountBelowMin},
		{name: "market buy without quote amount", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "buy_market", Amount: 1}, err: limits.ErrPriceBelowMin},
		{name: "market sell without amount", arg: &PlaceOrderRequest{Pair: spotTestPair, Type: "sell_market", Price: 1}, err: limits.ErrAmountBelowMin},
		{name: "no pair", arg: &PlaceOrderRequest{Type: "buy", Price: 1, Amount: 1}, err: currency.ErrCurrencyPairEmpty},
	} {
		_, err := e.placeOrderParams(tc.arg)
		assert.ErrorIsf(t, err, tc.err, "placeOrderParams should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name string
		arg  *PlaceOrderRequest
		exp  url.Values
	}{
		{
			name: "limit with every option",
			arg:  &PlaceOrderRequest{Pair: spotTestPair, Type: "buy_fok", Price: 50000.5, Amount: 0.00012, CustomID: "gct", Window: 1500 * time.Millisecond},
			exp:  url.Values{"symbol": {"btc_usdt"}, "type": {"buy_fok"}, "price": {"50000.5"}, "amount": {"0.00012"}, "custom_id": {"gct"}, "window": {"1500"}},
		},
		{
			name: "market buy spending a quote amount",
			arg:  &PlaceOrderRequest{Pair: spotTestPair, Type: "buy_market", Price: 100},
			exp:  url.Values{"symbol": {"btc_usdt"}, "type": {"buy_market"}, "price": {"100"}},
		},
		{
			name: "market sell",
			arg:  &PlaceOrderRequest{Pair: spotTestPair, Type: "sell_market", Amount: 0.5},
			exp:  url.Values{"symbol": {"btc_usdt"}, "type": {"sell_market"}, "amount": {"0.5"}},
		},
	} {
		params, err := e.placeOrderParams(tc.arg)
		require.NoErrorf(t, err, "placeOrderParams must not error for a %s", tc.name)
		assert.Equalf(t, tc.exp, params, "placeOrderParams should send the parameters of a %s", tc.name)
	}
}

// liveTestOrderRequest returns the order the live tests place: a buy of BTC-USDT at half its price, which rests on the
// book until cancelled
func liveTestOrderRequest(t *testing.T) *PlaceOrderRequest {
	t.Helper()
	prices, err := e.GetLatestPrices(t.Context(), spotTestPair)
	require.NoError(t, err, "GetLatestPrices must not error")
	require.NotEmpty(t, prices, "GetLatestPrices must return the pair's price")
	price := math.Round(prices[0].Price.Float64() / 2)
	require.Positive(t, price, "The live test order must have a price")
	return &PlaceOrderRequest{Pair: spotTestPair, Type: "buy", Price: price, Amount: 0.0001}
}

// cancelLiveTestOrderOnCleanup cancels an order the test placed once it ends, whether or not it reached its own
// cancellation; cancelling an order already cancelled fails harmlessly
func cancelLiveTestOrderOnCleanup(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = e.CancelExistingOrder(context.WithoutCancel(t.Context()), &OrderReferenceRequest{Pair: spotTestPair, OrderID: orderID})
	})
}

// placeLiveTestOrder places liveTestOrderRequest's order, to be cancelled once the test ends
func placeLiveTestOrder(t *testing.T) *PlaceOrderResponse {
	t.Helper()
	resp, err := e.PlaceOrder(t.Context(), liveTestOrderRequest(t))
	require.NoError(t, err, "PlaceOrder must not error")
	require.NotEmpty(t, resp.OrderID, "PlaceOrder must return the order placed")
	cancelLiveTestOrderOnCleanup(t, resp.OrderID)
	return resp
}

func TestPlaceTestOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTestOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceTestOrder must reject a nil request")

	// A test order is validated without being placed
	skipLiveWithoutCredentials(t)
	req := &PlaceOrderRequest{Pair: spotTestPair, Type: "buy", Price: 50000, Amount: 0.001, CustomID: "gct-test-order", Window: 5 * time.Second}
	if !mockTests {
		req = liveTestOrderRequest(t)
	}
	result, err := e.PlaceTestOrder(t.Context(), req)
	require.NoError(t, err, "PlaceTestOrder must not error")
	if mockTests {
		exp := &PlaceOrderResponse{Symbol: "btc_usdt", OrderID: "91dd58c4-370f-461d-0000-8cc3e83ddad3", CustomID: "gct-test-order"}
		assert.Equal(t, exp, result, "PlaceTestOrder should decode the order's IDs")
		return
	}
	assert.Equal(t, "btc_usdt", result.Symbol, "PlaceTestOrder should echo the pair")
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{Pair: spotTestPair, Type: "buy"})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceOrder must validate the order")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		result := placeLiveTestOrder(t)
		assert.Equal(t, "btc_usdt", result.Symbol, "PlaceOrder should return the order's pair")
		return
	}
	result, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{Pair: spotTestPair, Type: "buy", Price: 50000, Amount: 0.001, CustomID: "gct-order", Window: 5 * time.Second})
	require.NoError(t, err, "PlaceOrder must not error")
	exp := &PlaceOrderResponse{Symbol: "btc_usdt", OrderID: "8241275e-8666-4ab8-a84d-2e87f82e12ff", CustomID: "gct-order"}
	assert.Equal(t, exp, result, "PlaceOrder should decode the order's IDs")
}

func TestOrderReferenceParams(t *testing.T) {
	t.Parallel()
	_, err := e.orderReferenceParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "orderReferenceParams must reject a nil request")
	_, err = e.orderReferenceParams(&OrderReferenceRequest{Pair: spotTestPair})
	require.ErrorIs(t, err, errOrderIDRequired, "orderReferenceParams must require an order ID or client order ID")
	_, err = e.orderReferenceParams(&OrderReferenceRequest{OrderID: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "orderReferenceParams must require a pair")

	params, err := e.orderReferenceParams(&OrderReferenceRequest{Pair: spotTestPair, OrderID: "1", ClientOrderID: "c1"})
	require.NoError(t, err, "orderReferenceParams must not error")
	assert.Equal(t, url.Values{"symbol": {"btc_usdt"}, "orderId": {"1"}, "origClientOrderId": {"c1"}}, params, "orderReferenceParams should send both IDs")
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), &OrderReferenceRequest{Pair: spotTestPair})
	require.ErrorIs(t, err, errOrderIDRequired, "CancelExistingOrder must require an order ID or client order ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed := placeLiveTestOrder(t)
		result, err := e.CancelExistingOrder(t.Context(), &OrderReferenceRequest{Pair: spotTestPair, OrderID: placed.OrderID})
		require.NoError(t, err, "CancelExistingOrder must not error")
		assert.Equal(t, "buy", result.TradeType, "CancelExistingOrder should return the cancelled order")
		return
	}
	result, err := e.CancelExistingOrder(t.Context(), &OrderReferenceRequest{Pair: spotTestPair, OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554"})
	require.NoError(t, err, "CancelExistingOrder must not error")
	exp := &CancelOrderResponse{TradeType: "buy", Price: 50000, OriginalQuantity: 0.001}
	assert.Equal(t, exp, result, "CancelExistingOrder should decode the cancelled order")

	result, err = e.CancelExistingOrder(t.Context(), &OrderReferenceRequest{Pair: spotTestPair, ClientOrderID: "4b11c4ac-d54a-4bb5-86f9-06322174a036"})
	require.NoError(t, err, "CancelExistingOrder must not error by client order ID")
	exp = &CancelOrderResponse{TradeType: "buy_maker", Price: 50000, OriginalQuantity: 0.001, ExecutedQuantity: 0.0004, Status: 1}
	assert.Equal(t, exp, result, "CancelExistingOrder should decode the order cancelled by client order ID")
}

func TestCancelPairOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPairOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelPairOrders must require a pair")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("CancelPairOrders is not tested live, as it would cancel every open order on the pair, not only the test's")
	}
	result, err := e.CancelPairOrders(t.Context(), spotTestPair)
	require.NoError(t, err, "CancelPairOrders must not error")
	exp := []CancelOrderResponse{
		{OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554", ClientOrderID: "4b11c4ac-d54a-4bb5-86f9-06322174a036", TradeType: "buy", Price: 50000, OriginalQuantity: 0.001},
		{OrderID: "2539cee7-a0de-4caa-bbbe-993d39bc8efb", ClientOrderID: "d6f6dc30-5de9-4fe9-ba70-ce6e173e114e", TradeType: "buy", Price: 51000, OriginalQuantity: 0.001},
	}
	assert.Equal(t, exp, result, "CancelPairOrders should decode every cancelled order")
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.QueryOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QueryOrder must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.QueryOrder(t.Context(), &OrderReferenceRequest{Pair: spotTestPair, OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554"})
	if !mockTests {
		// The order is only recorded in the mock, so the live venue reports it missing
		assert.Error(t, err, "QueryOrder should report an order the account doesn't hold")
		return
	}
	require.NoError(t, err, "QueryOrder must not error")
	exp := &OrderResponse{
		Symbol: "btc_usdt", OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554", ClientOrderID: "4b11c4ac-d54a-4bb5-86f9-06322174a036", Type: "buy_maker",
		Price: 51000, OriginalQuantity: 0.001, OriginalQuoteOrderQuantity: 51, ExecutedQuantity: 0.0004, CumulativeQuoteQuantity: 20.4, Status: 3,
		CreateTime: types.Time(time.UnixMilli(1772335196123)), UpdateTime: types.Time(time.UnixMilli(1772335197000)),
	}
	assert.Equal(t, exp, result, "QueryOrder should decode every field")
}

func TestOrdersPageParams(t *testing.T) {
	t.Parallel()
	_, err := e.ordersPageParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ordersPageParams must reject a nil request")
	_, err = e.ordersPageParams(&OrdersPageRequest{Pair: spotTestPair, PageLength: maxOrdersPageLength + 1})
	require.ErrorIs(t, err, errLimitOutOfRange, "ordersPageParams must reject a page longer than LBank allows")
	_, err = e.ordersPageParams(&OrdersPageRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ordersPageParams must require a pair")

	params, err := e.ordersPageParams(&OrdersPageRequest{Pair: spotTestPair})
	require.NoError(t, err, "ordersPageParams must not error")
	assert.Equal(t, url.Values{"symbol": {"btc_usdt"}, "current_page": {"1"}, "page_length": {"200"}}, params, "ordersPageParams should default to the first page of 200 orders")
	params, err = e.ordersPageParams(&OrdersPageRequest{Pair: spotTestPair, CurrentPage: 3, PageLength: 10})
	require.NoError(t, err, "ordersPageParams must not error")
	assert.Equal(t, url.Values{"symbol": {"btc_usdt"}, "current_page": {"3"}, "page_length": {"10"}}, params, "ordersPageParams should send the page asked for")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOpenOrders(t.Context(), &OrdersPageRequest{Pair: spotTestPair})
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		exp := &OrdersPageResponse{
			Total: 1, PageLength: 200, CurrentPage: 1,
			Orders: []OrderResponse{
				{
					Symbol: "btc_usdt", OrderID: "11f0e0a7-eac3-436b-b2d3-779c4f910e31", ClientOrderID: "8e145780-033c-4f59-8cdc-c2e75856db65", Type: "buy",
					Price: 50000, OriginalQuantity: 0.01, OriginalQuoteOrderQuantity: 500,
					CreateTime: types.Time(time.UnixMilli(1772335195111)), UpdateTime: types.Time(time.UnixMilli(1772335196000)),
				},
				{
					Symbol: "btc_usdt", OrderID: "46e9dbe6-688b-4337-b7b8-e91cfb3e2cd5", Type: "sell_fok",
					Price: 100000, OriginalQuantity: 0.01, OriginalQuoteOrderQuantity: 1000, ExecutedQuantity: 0.003, CumulativeQuoteQuantity: 300, Status: 1,
					CreateTime: types.Time(time.UnixMilli(1772335195222)), UpdateTime: types.Time(time.UnixMilli(1772335199000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetOpenOrders should decode the page")
		return
	}
	assert.NotNil(t, result, "GetOpenOrders should return a page")
}

func TestQueryOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.QueryOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QueryOrderHistory must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.QueryOrderHistory(t.Context(), &OrdersPageRequest{Pair: spotTestPair})
	require.NoError(t, err, "QueryOrderHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "QueryOrderHistory should return a page")
		return
	}
	exp := &OrdersPageResponse{
		Total: 1, PageLength: 200, CurrentPage: 1,
		Orders: []OrderResponse{
			{
				Symbol: "btc_usdt", OrderID: "2539cee7-a0de-4caa-bbbe-993d39bc8efb", Type: "buy", Price: 51000, OriginalQuantity: 0.01,
				OriginalQuoteOrderQuantity: 510, Status: -1,
				CreateTime: types.Time(time.UnixMilli(1772335195333)), UpdateTime: types.Time(time.UnixMilli(1772335196000)),
			},
			{
				Symbol: "btc_usdt", OrderID: "5ca490f9-eede-45c7-82fe-89b6beb3fd7b", Type: "buy_market", OriginalQuoteOrderQuantity: 100,
				ExecutedQuantity: 0.00125, CumulativeQuoteQuantity: 100, Status: 2,
				CreateTime: types.Time(time.UnixMilli(1772335195444)), UpdateTime: types.Time(time.UnixMilli(1772335197000)),
			},
		},
	}
	assert.Equal(t, exp, result, "QueryOrderHistory should decode the page")

	result, err = e.QueryOrderHistory(t.Context(), &OrdersPageRequest{Pair: spotTestPair, CurrentPage: 2, PageLength: 50, Status: "2"})
	require.NoError(t, err, "QueryOrderHistory must not error with a status")
	exp = &OrdersPageResponse{Total: 2, PageLength: 50, CurrentPage: 2}
	assert.Equal(t, exp, result, "QueryOrderHistory should decode a page without orders")
}

func TestOrdersPageResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for name, payload := range map[string]string{
		"empty string": `{"total":1,"page_length":10,"current_page":1,"orders":""}`,
		"null":         `{"total":1,"page_length":10,"current_page":1,"orders":null}`,
		"absent":       `{"total":1,"page_length":10,"current_page":1}`,
		"empty list":   `{"total":1,"page_length":10,"current_page":1,"orders":[]}`,
	} {
		var resp OrdersPageResponse
		require.NoErrorf(t, resp.UnmarshalJSON([]byte(payload)), "UnmarshalJSON must accept orders as %s", name)
		assert.Equalf(t, OrdersPageResponse{Total: 1, PageLength: 10, CurrentPage: 1}, OrdersPageResponse{Total: resp.Total, PageLength: resp.PageLength, CurrentPage: resp.CurrentPage}, "UnmarshalJSON should decode the page fields with orders as %s", name)
		assert.Emptyf(t, resp.Orders, "UnmarshalJSON should decode no orders from %s", name)
	}
	resp := OrdersPageResponse{Orders: []OrderResponse{{OrderID: "stale"}}}
	require.NoError(t, resp.UnmarshalJSON([]byte(`{"orders":""}`)), "UnmarshalJSON must accept an empty string")
	assert.Nil(t, resp.Orders, "UnmarshalJSON should clear orders a reused page held")
	assert.Error(t, resp.UnmarshalJSON([]byte(`{"orders":7}`)), "UnmarshalJSON should reject orders that are neither a list nor empty")
	assert.Error(t, resp.UnmarshalJSON([]byte(`{"total":"x"}`)), "UnmarshalJSON should reject a malformed page")
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountInformation(t.Context())
	require.NoError(t, err, "GetAccountInformation must not error")
	if mockTests {
		exp := &AccountInformationResponse{
			UID: "LBA0B12679",
			Balances: []AccountBalance{
				{Asset: currency.NewCode("usdt"), Free: 1000},
				{Asset: currency.NewCode("btc"), Free: 0.01, Locked: 0.002},
				{Asset: currency.NewCode("eth"), Free: 0.1},
			},
			CanTrade: true, CanWithdraw: true, CanDeposit: true,
		}
		assert.Equal(t, exp, result, "GetAccountInformation should decode every field")
		return
	}
	assert.NotEmpty(t, result.UID, "GetAccountInformation should return the account's UID")
}

func TestGetTransactionHistory(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	_, err := e.GetTransactionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransactionHistory must reject a nil request")
	_, err = e.GetTransactionHistory(t.Context(), &TransactionHistoryRequest{Pair: spotTestPair, Limit: maxTransactionsLimit + 1})
	require.ErrorIs(t, err, errLimitOutOfRange, "GetTransactionHistory must reject a limit above LBank's")
	_, err = e.GetTransactionHistory(t.Context(), &TransactionHistoryRequest{Pair: spotTestPair, StartTime: start.Add(time.Hour), EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTransactionHistory must reject a start after the end")
	_, err = e.GetTransactionHistory(t.Context(), &TransactionHistoryRequest{Pair: spotTestPair, StartTime: start, EndTime: start.Add(transactionHistoryMaxWindow + time.Second)})
	require.ErrorIs(t, err, errTimeRangeTooLong, "GetTransactionHistory must reject a range longer than LBank allows")
	_, err = e.GetTransactionHistory(t.Context(), &TransactionHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTransactionHistory must require a pair")

	skipLiveWithoutCredentials(t)
	result, err := e.GetTransactionHistory(t.Context(), &TransactionHistoryRequest{
		Pair:      spotTestPair,
		StartTime: start,
		EndTime:   start.Add(24 * time.Hour),
		FromID:    "da612956ee81297529d695683559516b20fdeba0",
		Limit:     2,
	})
	require.NoError(t, err, "GetTransactionHistory must not error")
	if mockTests {
		exp := []TransactionResponse{
			{
				ID: "da612956ee81297529d695683559516b20fdeba1", Symbol: "btc_usdt", OrderID: "cd58b6a0-7afa-47ae-9bbb-3a212bdab3bb", Price: 2319.37,
				Quantity: 0.01, QuoteQuantity: 23.1937, Commission: 0.00051, IsBuyer: true, Time: types.Time(time.UnixMilli(1772335196000)),
			},
			{
				ID: "ada27f3c83ebffb7defd7a1699dd2f4d281fc033", Symbol: "btc_usdt", OrderID: "a4cd386d-6d8b-47fa-9a45-daa10c6dab7a", Price: 2319.96,
				Quantity: 0.01, QuoteQuantity: 23.1996, Commission: 1.18318, IsMaker: true, Time: types.Time(time.UnixMilli(1772335196001)),
			},
		}
		assert.Equal(t, exp, result, "GetTransactionHistory should decode every trade")
		return
	}
	assert.NotNil(t, result, "GetTransactionHistory should return the account's trades")
}
