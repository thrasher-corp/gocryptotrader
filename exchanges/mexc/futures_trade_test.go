package mexc

import (
	"net/http"
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
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The replies below are the venue's documented examples, as no credentials were available to record live ones. Some
// values are changed from the examples' zeros, so that more fields' decoding is checked

func TestValidateFuturesOrder(t *testing.T) {
	t.Parallel()
	valid := func() *FuturesOrderRequest {
		return &FuturesOrderRequest{Symbol: "BTC_USDT", Price: 50000, Volume: 1, Leverage: 20, Side: FuturesSideOpenLong, Type: FuturesOrderTypeLimit, OpenType: FuturesOpenTypeIsolated}
	}
	require.NoError(t, validateFuturesOrder(valid()), "a priced isolated open with leverage must be valid")
	require.ErrorIs(t, validateFuturesOrder(nil), common.ErrNilPointer)
	for name, tc := range map[string]struct {
		change func(*FuturesOrderRequest)
		err    error
	}{
		"no symbol":            {func(o *FuturesOrderRequest) { o.Symbol = "" }, currency.ErrSymbolStringEmpty},
		"no side":              {func(o *FuturesOrderRequest) { o.Side = 0 }, errInvalidOrderSide},
		"unknown side":         {func(o *FuturesOrderRequest) { o.Side = 5 }, errInvalidOrderSide},
		"unknown side type":    {func(o *FuturesOrderRequest) { o.SideType = "BUY" }, errInvalidSideType},
		"suffixed side type":   {func(o *FuturesOrderRequest) { o.SideType = "buyer" }, errInvalidSideType},
		"side type of a close": {func(o *FuturesOrderRequest) { o.Side, o.SideType = FuturesSideCloseLong, "sold" }, errInvalidSideType},
		"unknown type":         {func(o *FuturesOrderRequest) { o.Type = 7 }, order.ErrUnsupportedOrderType},
		"no open type":         {func(o *FuturesOrderRequest) { o.OpenType = 0 }, errInvalidOpenType},
		"no volume":            {func(o *FuturesOrderRequest) { o.Volume = 0 }, limits.ErrAmountBelowMin},
		"unpriced limit":       {func(o *FuturesOrderRequest) { o.Price = 0 }, limits.ErrPriceBelowMin},
		"isolated no leverage": {func(o *FuturesOrderRequest) { o.Leverage = 0 }, errInvalidLeverage},
		"long external ID":     {func(o *FuturesOrderRequest) { o.ExternalOrderID = strings.Repeat("x", futuresExternalOrderIDLimit+1) }, errExternalOrderIDTooLong},
		"unknown BBO":          {func(o *FuturesOrderRequest) { o.BBOTypeNumber = 5 }, order.ErrUnsupportedOrderType},
		"BBO with external ID": {func(o *FuturesOrderRequest) { o.BBOTypeNumber, o.ExternalOrderID = 1, "gct-1" }, order.ErrClientOrderIDNotSupported},
		"unknown loss trend":   {func(o *FuturesOrderRequest) { o.LossTrend = 4 }, errInvalidPriceType},
		"unknown profit trend": {func(o *FuturesOrderRequest) { o.ProfitTrend = 4 }, errInvalidPriceType},
		"unknown mode":         {func(o *FuturesOrderRequest) { o.PositionMode = 3 }, errInvalidPositionMode},
		"unknown STP mode":     {func(o *FuturesOrderRequest) { o.SelfTradePreventionMode = 4 }, errInvalidSelfTradePrevention},
	} {
		o := valid()
		tc.change(o)
		assert.ErrorIsf(t, validateFuturesOrder(o), tc.err, "an order with %s should be refused", name)
	}
	for name, change := range map[string]func(*FuturesOrderRequest){
		"unpriced market":         func(o *FuturesOrderRequest) { o.Type, o.Price = FuturesOrderTypeMarket, 0 },
		"BBO limit":               func(o *FuturesOrderRequest) { o.BBOTypeNumber, o.Price = 1, 0 },
		"cross open":              func(o *FuturesOrderRequest) { o.OpenType, o.Leverage = FuturesOpenTypeCross, 0 },
		"isolated close":          func(o *FuturesOrderRequest) { o.Side, o.Leverage = FuturesSideCloseLong, 0 },
		"longest external ID":     func(o *FuturesOrderRequest) { o.ExternalOrderID = strings.Repeat("x", futuresExternalOrderIDLimit) },
		"take-profit at fair":     func(o *FuturesOrderRequest) { o.ProfitTrend = FuturesPriceTypeFair },
		"one-way mode":            func(o *FuturesOrderRequest) { o.PositionMode = FuturesPositionModeOneWay },
		"cancel taker on a cross": func(o *FuturesOrderRequest) { o.SelfTradePreventionMode = 3 },
		"buy side type":           func(o *FuturesOrderRequest) { o.SideType = FuturesSideTypeBuy },
		"sell side type":          func(o *FuturesOrderRequest) { o.SideType = FuturesSideTypeSell },
	} {
		o := valid()
		change(o)
		assert.NoErrorf(t, validateFuturesOrder(o), "an order with %s should be valid", name)
	}
}

func TestParseFuturesOrderID(t *testing.T) {
	t.Parallel()
	_, err := parseFuturesOrderID("")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	for _, id := range []string{"0", "-1", "1.5", "abc", "18446744073709551616"} {
		_, err = parseFuturesOrderID(id)
		assert.ErrorIsf(t, err, errInvalidFuturesID, "%q should be refused", id)
	}
	id, err := parseFuturesOrderID("18446744073709551615")
	require.NoError(t, err, "parseFuturesOrderID must not error for the largest ID")
	assert.Equal(t, uint64(18446744073709551615), id, "the largest ID should be read exactly")
}

func TestJoinFuturesStates(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3,4,5", joinFuturesStates([]uint8{FuturesOrderStateFilled, FuturesOrderStateCancelled, FuturesOrderStateInvalid}), "the states should be comma separated")
	assert.Empty(t, joinFuturesStates(nil), "no states should join to nothing")
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesOrder(t.Context(), &FuturesOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	for _, side := range []uint8{FuturesSideOpenLong, FuturesSideCloseShort, FuturesSideOpenShort, FuturesSideCloseLong} {
		_, err = e.PlaceFuturesOrder(t.Context(), &FuturesOrderRequest{Symbol: "BTC_USDT", Volume: 1, Side: side, Type: FuturesOrderTypeMarket, OpenType: FuturesOpenTypeCross})
		require.ErrorIsf(t, err, errInvalidSideType, "an order of side %d without a side type must be refused before it is sent", side)
	}

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/create",
		body:   `{"symbol":"BTC_USDT","vol":1,"leverage":20,"side":3,"sideType":"sell","type":5,"openType":1,"stopLossPrice":120000,"lossTrend":2,"stpMode":1}`,
		data:   `{"orderId":"739113577038255616","ts":1761888808839}`,
	}), liveUnavailable)
	result, err := ex.PlaceFuturesOrder(t.Context(), &FuturesOrderRequest{
		Symbol:                  "BTC_USDT",
		Volume:                  1,
		Leverage:                20,
		Side:                    FuturesSideOpenShort,
		SideType:                FuturesSideTypeSell,
		Type:                    FuturesOrderTypeMarket,
		OpenType:                FuturesOpenTypeIsolated,
		StopLossPrice:           120000,
		LossTrend:               FuturesPriceTypeFair,
		SelfTradePreventionMode: 1,
	})
	require.NoError(t, err, "PlaceFuturesOrder must not error")
	assert.Equal(t, &FuturesOrderPlacementResponse{OrderID: "739113577038255616", Timestamp: types.Time(time.UnixMilli(1761888808839))}, result,
		"PlaceFuturesOrder should decode the order ID and the venue's time")
}

func TestPlaceFuturesBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PlaceFuturesBatchOrders(t.Context(), make([]FuturesOrderRequest, futuresBatchLimit+1))
	require.ErrorIs(t, err, errTooManyOrders)
	_, err = e.PlaceFuturesBatchOrders(t.Context(), []FuturesOrderRequest{{Symbol: "BTC_USDT"}})
	require.ErrorIs(t, err, errInvalidOrderSide, "an invalid order must fail the batch before it is sent")
	_, err = e.PlaceFuturesBatchOrders(t.Context(), []FuturesOrderRequest{{Symbol: "BTC_USDT", Volume: 1, Side: FuturesSideOpenShort, SideType: "short", Type: FuturesOrderTypeMarket, OpenType: FuturesOpenTypeCross}})
	require.ErrorIs(t, err, errInvalidSideType, "an order with an unknown side type must fail the batch before it is sent")

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/submit_batch",
		body: `[{"symbol":"BTC_USDT","price":50000,"vol":1,"side":1,"type":1,"openType":2,"externalOid":"gct-1","stpMode":0},` +
			`{"symbol":"BTC_USDT","price":150000,"vol":1,"side":3,"type":2,"openType":2,"stpMode":0}]`,
		data: `[{"orderId":768116802038827520,"externalOid":"gct-1","errorCode":0},{"orderId":0,"errorCode":2009,"errorMsg":"insufficient balance"}]`,
	}), liveUnavailable)
	result, err := ex.PlaceFuturesBatchOrders(t.Context(), []FuturesOrderRequest{
		{Symbol: "BTC_USDT", Price: 50000, Volume: 1, Side: FuturesSideOpenLong, Type: FuturesOrderTypeLimit, OpenType: FuturesOpenTypeCross, ExternalOrderID: "gct-1"},
		{Symbol: "BTC_USDT", Price: 150000, Volume: 1, Side: FuturesSideOpenShort, Type: FuturesOrderTypePostOnly, OpenType: FuturesOpenTypeCross},
	})
	require.NoError(t, err, "PlaceFuturesBatchOrders must not error when an order is refused")
	// A numeric order ID above 2^53 keeps every digit, and a refused order carries its error
	exp := []*FuturesBatchOrderResponse{
		{OrderID: "768116802038827520", ExternalOrderID: "gct-1"},
		{ErrorCode: 2009, ErrorMessage: "insufficient balance"},
	}
	assert.Equal(t, exp, result, "every order's outcome should be decoded")
}

func TestChaseFuturesOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ChaseFuturesOrder(t.Context(), 0), order.ErrOrderIDNotSet)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/chase_limit_order",
		body:   `{"orderId":739204819784386560}`,
	}), liveUnavailable)
	require.NoError(t, ex.ChaseFuturesOrder(t.Context(), 739204819784386560), "ChaseFuturesOrder must not error")
}

func TestModifyFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ModifyFuturesOrder(t.Context(), &FuturesModifyOrderRequest{Price: 1, Volume: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.ModifyFuturesOrder(t.Context(), &FuturesModifyOrderRequest{OrderID: 1, Volume: 1})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.ModifyFuturesOrder(t.Context(), &FuturesModifyOrderRequest{OrderID: 1, Price: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/change_limit_order",
		body:   `{"orderId":739204819784386560,"price":51000,"vol":2}`,
		data:   `"739204819784386560"`,
	}), liveUnavailable)
	result, err := ex.ModifyFuturesOrder(t.Context(), &FuturesModifyOrderRequest{OrderID: 739204819784386560, Price: 51000, Volume: 2})
	require.NoError(t, err, "ModifyFuturesOrder must not error")
	assert.Equal(t, FuturesID("739204819784386560"), result, "the order's ID should be returned")
}

func TestCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CancelFuturesOrders(t.Context(), make([]uint64, futuresBatchLimit+1))
	require.ErrorIs(t, err, errTooManyOrders)
	_, err = e.CancelFuturesOrders(t.Context(), []uint64{1, 0})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/cancel",
		body:   `[101716841474621953,108886241042563584]`,
		data: `[{"orderId":101716841474621953,"errorCode":2040,"errorMsg":"order not exist"},` +
			`{"orderId":108886241042563584,"errorCode":0,"errorMsg":"success"}]`,
	}), liveUnavailable)
	result, err := ex.CancelFuturesOrders(t.Context(), []uint64{101716841474621953, 108886241042563584})
	require.NoError(t, err, "CancelFuturesOrders must not error when an order is refused")
	exp := []*FuturesCancelOrderResponse{
		{OrderID: "101716841474621953", ErrorCode: 2040, ErrorMessage: "order not exist"},
		{OrderID: "108886241042563584", ErrorMessage: "success"},
	}
	assert.Equal(t, exp, result, "every order's outcome should be decoded")
}

func TestValidateExternalOrders(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, validateExternalOrders(nil), errExternalOrderIDRequired)
	require.ErrorIs(t, validateExternalOrders(make([]FuturesExternalOrderRequest, futuresBatchLimit+1)), errTooManyOrders)
	require.ErrorIs(t, validateExternalOrders([]FuturesExternalOrderRequest{{ExternalOrderID: "gct-1"}}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, validateExternalOrders([]FuturesExternalOrderRequest{{Symbol: "BTC_USDT"}}), errExternalOrderIDRequired)
	require.NoError(t, validateExternalOrders([]FuturesExternalOrderRequest{{Symbol: "BTC_USDT", ExternalOrderID: "gct-1"}}), "a named order must be valid")
}

func TestCancelFuturesOrdersByExternalID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrdersByExternalID(t.Context(), nil)
	require.ErrorIs(t, err, errExternalOrderIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/batch_cancel_with_external",
		body:   `[{"symbol":"BTC_USDT","externalOid":"ext_11"}]`,
		data:   `[{"orderId":108886241042563584,"externalOid":"ext_11","errorCode":0,"errorMsg":"success"}]`,
	}), liveUnavailable)
	result, err := ex.CancelFuturesOrdersByExternalID(t.Context(), []FuturesExternalOrderRequest{{Symbol: "BTC_USDT", ExternalOrderID: "ext_11"}})
	require.NoError(t, err, "CancelFuturesOrdersByExternalID must not error")
	assert.Equal(t, []*FuturesCancelOrderResponse{{OrderID: "108886241042563584", ExternalOrderID: "ext_11", ErrorMessage: "success"}}, result,
		"the order's outcome should be decoded")
}

func TestCancelFuturesOrderByExternalID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrderByExternalID(t.Context(), currency.EMPTYPAIR, "gct-1")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelFuturesOrderByExternalID(t.Context(), futuresTradablePair, "")
	require.ErrorIs(t, err, errExternalOrderIDRequired)

	// The venue documents a list reply to a request naming one order, so an object is read as well
	for _, data := range []string{
		`[{"orderId":108886241042563584,"errorCode":0,"errorMsg":"success"}]`,
		`{"orderId":108886241042563584,"errorCode":0,"errorMsg":"success"}`,
	} {
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
			method: http.MethodPost,
			path:   "private/order/cancel_with_external",
			body:   `{"symbol":"BTC_USDT","externalOid":"gct-1"}`,
			data:   data,
		}), liveUnavailable)
		result, err := ex.CancelFuturesOrderByExternalID(t.Context(), futuresTradablePair, "gct-1")
		require.NoErrorf(t, err, "CancelFuturesOrderByExternalID must not error for %s", data)
		assert.Equalf(t, []*FuturesCancelOrderResponse{{OrderID: "108886241042563584", ErrorMessage: "success"}}, result, "the outcome should be decoded from %s", data)
	}
}

func TestCancelAllFuturesOrders(t *testing.T) {
	t.Parallel()
	for pair, body := range map[currency.Pair]string{futuresTradablePair: `{"symbol":"BTC_USDT"}`, currency.EMPTYPAIR: `{"symbol":""}`} {
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel_all", body: body}), liveUnavailable)
		require.NoErrorf(t, ex.CancelAllFuturesOrders(t.Context(), pair), "CancelAllFuturesOrders must not error for %q", pair)
	}
}

func TestReverseFuturesPosition(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ReverseFuturesPosition(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ReverseFuturesPosition(t.Context(), &FuturesReversePositionRequest{PositionID: 1, Volume: 1}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, e.ReverseFuturesPosition(t.Context(), &FuturesReversePositionRequest{Symbol: "BTC_USDT", Volume: 1}), errPositionIDRequired)
	require.ErrorIs(t, e.ReverseFuturesPosition(t.Context(), &FuturesReversePositionRequest{Symbol: "BTC_USDT", PositionID: 1}), limits.ErrAmountBelowMin)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/reverse",
		body:   `{"symbol":"BTC_USDT","positionId":1109973831,"vol":5}`,
	}), liveUnavailable)
	require.NoError(t, ex.ReverseFuturesPosition(t.Context(), &FuturesReversePositionRequest{Symbol: "BTC_USDT", PositionID: 1109973831, Volume: 5}), "ReverseFuturesPosition must not error")
}

func TestCloseAllFuturesPositions(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/position/close_all", body: `{}`, data: `[]`}), liveUnavailable)
	require.NoError(t, ex.CloseAllFuturesPositions(t.Context()), "CloseAllFuturesPositions must not error")
}

func TestGetFuturesOpenOrderCounts(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/open_order_total_count",
		data:   `{"sumCount":3,"limitOrderCount":1,"stopOrderCount":1,"planOrderCount":1,"trackOrderCount":0}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesOpenOrderCounts(t.Context())
	require.NoError(t, err, "GetFuturesOpenOrderCounts must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, FuturesOpenOrderCountsResponse{SumCount: 3, LimitOrderCount: 1, StopOrderCount: 1, PlanOrderCount: 1}, *result, "the counts should be decoded")
}

func TestGetFuturesOrderByExternalID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderByExternalID(t.Context(), currency.EMPTYPAIR, "gct-1")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesOrderByExternalID(t.Context(), futuresTradablePair, "")
	require.ErrorIs(t, err, errExternalOrderIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/external/BTC_USDT/gct-1",
		data:   futuresFilledOrderReply,
	}), liveUnavailable)
	result, err := ex.GetFuturesOrderByExternalID(t.Context(), futuresTradablePair, "gct-1")
	require.NoError(t, err, "GetFuturesOrderByExternalID must not error")
	assert.Equal(t, futuresFilledOrder(), result, "the order named should be decoded")
}

func TestGetFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrder(t.Context(), "x")
	require.ErrorIs(t, err, errInvalidFuturesID)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/get/739106551624717312",
		data:   futuresFilledOrderReply,
	}), liveUnavailable)
	result, err := ex.GetFuturesOrder(t.Context(), "739106551624717312")
	require.NoError(t, err, "GetFuturesOrder must not error")
	assert.Equal(t, futuresFilledOrder(), result, "GetFuturesOrder should decode every field")
}

func TestGetFuturesOrdersByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrdersByID(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.GetFuturesOrdersByID(t.Context(), make([]string, futuresBatchLimit+1))
	require.ErrorIs(t, err, errTooManyOrders)
	_, err = e.GetFuturesOrdersByID(t.Context(), []string{"1", "x"})
	require.ErrorIs(t, err, errInvalidFuturesID)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/batch_query",
		query:  "order_ids=743829153601634818%2C739106551624717312",
		data:   "[" + futuresFilledOrderReply + "]",
	}), liveUnavailable)
	result, err := ex.GetFuturesOrdersByID(t.Context(), []string{"743829153601634818", "739106551624717312"})
	require.NoError(t, err, "GetFuturesOrdersByID must not error")
	assert.Equal(t, []*FuturesOrderResponse{futuresFilledOrder()}, result, "the orders found should be decoded")
}

func TestGetFuturesOrdersByExternalID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrdersByExternalID(t.Context(), []FuturesExternalOrderRequest{{Symbol: "BTC_USDT"}})
	require.ErrorIs(t, err, errExternalOrderIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/batch_query_with_external",
		body:   `[{"symbol":"BTC_USDT","externalOid":"gct-1"}]`,
		data:   "[" + futuresFilledOrderReply + "]",
	}), liveUnavailable)
	result, err := ex.GetFuturesOrdersByExternalID(t.Context(), []FuturesExternalOrderRequest{{Symbol: "BTC_USDT", ExternalOrderID: "gct-1"}})
	require.NoError(t, err, "GetFuturesOrdersByExternalID must not error")
	assert.Equal(t, []*FuturesOrderResponse{futuresFilledOrder()}, result, "the order named should be decoded")
}

func TestGetFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOpenOrders(t.Context(), &FuturesPageRequest{PageSize: futuresDefaultPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/open_orders",
		query:  "page_num=2&page_size=10",
		data:   `{"pageSize":10,"totalCount":11,"totalPage":2,"currentPage":2,"resultList":[` + futuresFilledOrderReply + `]}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesOpenOrders(t.Context(), &FuturesPageRequest{PageNumber: 2, PageSize: 10})
	require.NoError(t, err, "GetFuturesOpenOrders must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, []*FuturesOrderResponse{futuresFilledOrder()}, result, "the page's orders should be decoded")
}

func TestGetFuturesOrderHistory(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1761800000000)
	_, err := e.GetFuturesOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesOrderHistory(t.Context(), &FuturesOrderHistoryRequest{States: []uint8{FuturesOrderStateFilled}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesOrderHistory(t.Context(), &FuturesOrderHistoryRequest{Symbol: futuresTradablePair})
	require.ErrorIs(t, err, errOrderStatesRequired)
	_, err = e.GetFuturesOrderHistory(t.Context(), &FuturesOrderHistoryRequest{Symbol: futuresTradablePair, States: []uint8{FuturesOrderStateFilled}, StartTime: start, EndTime: start.Add(futuresHistoryWindow + time.Second)})
	require.ErrorIs(t, err, errHistoryWindowTooWide)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/history_orders",
		query:  "category=1&end_time=1761886400000&orderId=739106551624717312&page_num=1&page_size=5&start_time=1761800000000&states=3&symbol=BTC_USDT",
		data:   "[" + futuresFilledOrderReply + "]",
	}), liveUnavailable)
	result, err := ex.GetFuturesOrderHistory(t.Context(), &FuturesOrderHistoryRequest{
		Symbol:    futuresTradablePair,
		States:    []uint8{FuturesOrderStateFilled},
		Category:  FuturesOrderCategoryLimit,
		OrderID:   "739106551624717312",
		StartTime: start,
		EndTime:   start.Add(24 * time.Hour),
		PageSize:  5,
	})
	require.NoError(t, err, "GetFuturesOrderHistory must not error")
	assert.Equal(t, []*FuturesOrderResponse{futuresFilledOrder()}, result, "the order should be decoded")
}

func TestFuturesDealsParams(t *testing.T) {
	t.Parallel()
	_, err := futuresDealsParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = futuresDealsParams(&FuturesOrderDealsRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = futuresDealsParams(&FuturesOrderDealsRequest{Symbol: futuresTradablePair, PageSize: futuresDealsPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)
	params, err := futuresDealsParams(&FuturesOrderDealsRequest{Symbol: futuresTradablePair, PageSize: futuresDealsPageSizeLimit})
	require.NoError(t, err, "a page of 1000 deals must be accepted")
	assert.Equal(t, "page_num=1&page_size=1000&symbol=BTC_USDT", params.Encode(), "the page should be sent")
}

func TestGetFuturesOrderDeals(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/order_deals/v3",
		query:  "page_num=1&symbol=BTC_USDT",
		data: `[{"id":"9466823344","symbol":"BTC_USDT","side":3,"vol":1,"price":0.18695,"fee":0.0001,"feeCurrency":"USDT",` +
			`"profit":0,"category":1,"orderId":"739211854408619008","timestamp":1761912240000,"positionMode":1,"taker":true}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesOrderDeals(t.Context(), &FuturesOrderDealsRequest{Symbol: futuresTradablePair})
	require.NoError(t, err, "GetFuturesOrderDeals must not error")
	if !mockTests {
		return
	}
	// The venue's examples name a taker fill taker
	exp := []*FuturesOrderDealResponse{{
		ID:           "9466823344",
		Symbol:       "BTC_USDT",
		Side:         FuturesSideOpenShort,
		Volume:       1,
		Price:        0.18695,
		Fee:          0.0001,
		FeeCurrency:  currency.USDT,
		Taker:        true,
		Category:     FuturesOrderCategoryLimit,
		OrderID:      "739211854408619008",
		PositionMode: FuturesPositionModeHedge,
		Timestamp:    types.Time(time.UnixMilli(1761912240000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesOrderDeals should decode every field")
}

func TestFuturesOrderDealWasTaker(t *testing.T) {
	t.Parallel()
	assert.True(t, (&FuturesOrderDealResponse{IsTaker: true}).WasTaker(), "a fill sent as isTaker should be a taker's")
	assert.True(t, (&FuturesOrderDealResponse{Taker: true}).WasTaker(), "a fill sent as taker should be a taker's")
	assert.False(t, (&FuturesOrderDealResponse{}).WasTaker(), "a maker fill should not be a taker's")
}

func TestGetFuturesOrderDealDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderDealDetails(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/deal_details/102015012431820288",
		data: `[{"id":"159274416","symbol":"ETH_USDT","side":2,"vol":1,"price":1208.35,"feeCurrency":"USDT","fee":0.0072501,` +
			`"timestamp":1609992674000,"profit":0,"category":1,"orderId":"102015012431820288","isTaker":true}]`,
	}), liveUnavailable)
	result, err := ex.GetFuturesOrderDealDetails(t.Context(), "102015012431820288")
	require.NoError(t, err, "GetFuturesOrderDealDetails must not error")
	// The reply names a taker fill isTaker, as the venue's field table does
	exp := []*FuturesOrderDealResponse{{
		ID:          "159274416",
		Symbol:      "ETH_USDT",
		Side:        FuturesSideCloseShort,
		Volume:      1,
		Price:       1208.35,
		Fee:         0.0072501,
		FeeCurrency: currency.USDT,
		IsTaker:     true,
		Category:    FuturesOrderCategoryLimit,
		OrderID:     "102015012431820288",
		Timestamp:   types.Time(time.UnixMilli(1609992674000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesOrderDealDetails should decode every field")
}

func TestGetFuturesClosedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesClosedOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/close_orders",
		query:  "page_num=1&page_size=20&symbol=BTC_USDT",
		data:   "[" + futuresFilledOrderReply + "]",
	}), liveReadOnly)
	result, err := ex.GetFuturesClosedOrders(t.Context(), &FuturesOrderDealsRequest{Symbol: futuresTradablePair, PageSize: 20})
	require.NoError(t, err, "GetFuturesClosedOrders must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, []*FuturesOrderResponse{futuresFilledOrder()}, result, "the finished order should be decoded")
}

func TestGetFuturesPlanOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1761800000000)
	_, err := e.GetFuturesPlanOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesPlanOrders(t.Context(), &FuturesPlanOrdersRequest{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFuturesPlanOrders must refuse a window without both bounds, which the venue requires")
	_, err = e.GetFuturesPlanOrders(t.Context(), &FuturesPlanOrdersRequest{StartTime: start, EndTime: start.Add(time.Hour), Side: 9})
	require.ErrorIs(t, err, errInvalidOrderSide)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/planorder/list/orders",
		query:  "end_time=1761886400000&page_num=1&side=1&start_time=1761800000000&states=1%2C3&symbol=DOGE_USD",
		data: `[{"id":"739114860879304259","symbol":"DOGE_USD","leverage":2,"side":1,"triggerPrice":0.00001,"price":0.00001,` +
			`"vol":1,"openType":1,"triggerType":2,"state":1,"executeCycle":87600,"trend":1,"orderType":1,"errorCode":0,` +
			`"priceProtect":0,"createTime":1761889115000,"updateTime":1761889115000,"positionMode":1,"lossTrend":1,"profitTrend":1,` +
			`"reduceOnly":false}]`,
	}), liveUnavailable)
	result, err := ex.GetFuturesPlanOrders(t.Context(), &FuturesPlanOrdersRequest{
		Symbol:    currency.NewPairWithDelimiter("DOGE", "USD", currency.UnderscoreDelimiter),
		States:    []uint8{1, 3},
		Side:      FuturesSideOpenLong,
		StartTime: start,
		EndTime:   start.Add(24 * time.Hour),
	})
	require.NoError(t, err, "GetFuturesPlanOrders must not error")
	// The execution cycle is in hours
	exp := []*FuturesPlanOrderResponse{{
		ID:           "739114860879304259",
		Symbol:       "DOGE_USD",
		Leverage:     2,
		Side:         FuturesSideOpenLong,
		TriggerPrice: 0.00001,
		Price:        0.00001,
		Volume:       1,
		OpenType:     FuturesOpenTypeIsolated,
		TriggerType:  FuturesTriggerLessOrEqual,
		State:        1,
		ExecuteCycle: 87600,
		Trend:        FuturesPriceTypeLatest,
		OrderType:    FuturesOrderTypeLimit,
		PositionMode: FuturesPositionModeHedge,
		LossTrend:    FuturesPriceTypeLatest,
		ProfitTrend:  FuturesPriceTypeLatest,
		CreateTime:   types.Time(time.UnixMilli(1761889115000)),
		UpdateTime:   types.Time(time.UnixMilli(1761889115000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesPlanOrders should decode every field")
}

func TestPlaceFuturesPlanOrder(t *testing.T) {
	t.Parallel()
	valid := func() *FuturesPlanOrderRequest {
		return &FuturesPlanOrderRequest{
			Symbol: "BTC_USDT", Volume: 1, Leverage: 20, Side: FuturesSideOpenLong, OpenType: FuturesOpenTypeIsolated, TriggerPrice: 100000,
			TriggerType: FuturesTriggerGreaterOrEqual, ExecuteCycle: FuturesExecuteCycle24Hours, OrderType: FuturesOrderTypeMarket, Trend: FuturesPriceTypeLatest,
		}
	}
	_, err := e.PlaceFuturesPlanOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	for name, tc := range map[string]struct {
		change func(*FuturesPlanOrderRequest)
		err    error
	}{
		"no symbol":            {func(o *FuturesPlanOrderRequest) { o.Symbol = "" }, currency.ErrSymbolStringEmpty},
		"no side":              {func(o *FuturesPlanOrderRequest) { o.Side = 0 }, errInvalidOrderSide},
		"no open type":         {func(o *FuturesPlanOrderRequest) { o.OpenType = 0 }, errInvalidOpenType},
		"no volume":            {func(o *FuturesPlanOrderRequest) { o.Volume = 0 }, limits.ErrAmountBelowMin},
		"no leverage":          {func(o *FuturesPlanOrderRequest) { o.Leverage = 0 }, errInvalidLeverage},
		"no trigger price":     {func(o *FuturesPlanOrderRequest) { o.TriggerPrice = 0 }, limits.ErrPriceBelowMin},
		"no trigger type":      {func(o *FuturesPlanOrderRequest) { o.TriggerType = 0 }, errInvalidTriggerType},
		"no execution cycle":   {func(o *FuturesPlanOrderRequest) { o.ExecuteCycle = 3 }, errInvalidExecutionCycle},
		"market to limit":      {func(o *FuturesPlanOrderRequest) { o.OrderType = FuturesOrderTypeMarketToLimit }, order.ErrUnsupportedOrderType},
		"unpriced limit":       {func(o *FuturesPlanOrderRequest) { o.OrderType = FuturesOrderTypeLimit }, limits.ErrPriceBelowMin},
		"no trend":             {func(o *FuturesPlanOrderRequest) { o.Trend = 0 }, errInvalidPriceType},
		"unknown profit trend": {func(o *FuturesPlanOrderRequest) { o.ProfitTrend = 4 }, errInvalidPriceType},
	} {
		o := valid()
		tc.change(o)
		_, err := e.PlaceFuturesPlanOrder(t.Context(), o)
		assert.ErrorIsf(t, err, tc.err, "a plan order with %s should be refused", name)
	}

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/planorder/place/v2",
		body: `{"symbol":"BTC_USDT","vol":1,"leverage":20,"side":1,"openType":1,"triggerPrice":100000,"triggerType":1,` +
			`"executeCycle":1,"orderType":5,"trend":1}`,
		data: `"739206374277809664"`,
	}), liveUnavailable)
	result, err := ex.PlaceFuturesPlanOrder(t.Context(), valid())
	require.NoError(t, err, "PlaceFuturesPlanOrder must not error")
	assert.Equal(t, FuturesID("739206374277809664"), result, "the plan order's ID should be returned")
}

func TestModifyFuturesPlanOrder(t *testing.T) {
	t.Parallel()
	valid := func() *FuturesModifyPlanOrderRequest {
		return &FuturesModifyPlanOrderRequest{Symbol: "BTC_USDT", OrderID: 739206374277809664, TriggerPrice: 101000, Price: 101500, OrderType: FuturesOrderTypeLimit, TriggerType: FuturesTriggerGreaterOrEqual, Trend: FuturesPriceTypeFair}
	}
	require.ErrorIs(t, e.ModifyFuturesPlanOrder(t.Context(), nil), common.ErrNilPointer)
	for name, tc := range map[string]struct {
		change func(*FuturesModifyPlanOrderRequest)
		err    error
	}{
		"no symbol":        {func(o *FuturesModifyPlanOrderRequest) { o.Symbol = "" }, currency.ErrSymbolStringEmpty},
		"no order ID":      {func(o *FuturesModifyPlanOrderRequest) { o.OrderID = 0 }, order.ErrOrderIDNotSet},
		"no trigger price": {func(o *FuturesModifyPlanOrderRequest) { o.TriggerPrice = 0 }, limits.ErrPriceBelowMin},
		"no order type":    {func(o *FuturesModifyPlanOrderRequest) { o.OrderType = 0 }, order.ErrUnsupportedOrderType},
		"unpriced limit":   {func(o *FuturesModifyPlanOrderRequest) { o.Price = 0 }, limits.ErrPriceBelowMin},
		"no trigger type":  {func(o *FuturesModifyPlanOrderRequest) { o.TriggerType = 0 }, errInvalidTriggerType},
		"no trend":         {func(o *FuturesModifyPlanOrderRequest) { o.Trend = 0 }, errInvalidPriceType},
	} {
		o := valid()
		tc.change(o)
		assert.ErrorIsf(t, e.ModifyFuturesPlanOrder(t.Context(), o), tc.err, "a plan order change with %s should be refused", name)
	}

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/planorder/change_price",
		body:   `{"symbol":"BTC_USDT","orderId":739206374277809664,"triggerPrice":101000,"price":101500,"orderType":1,"triggerType":1,"trend":2}`,
	}), liveUnavailable)
	require.NoError(t, ex.ModifyFuturesPlanOrder(t.Context(), valid()), "ModifyFuturesPlanOrder must not error")
}

func TestCancelFuturesPlanOrders(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelFuturesPlanOrders(t.Context(), nil), order.ErrOrderIDNotSet)
	require.ErrorIs(t, e.CancelFuturesPlanOrders(t.Context(), make([]FuturesCancelPlanOrderRequest, futuresBatchLimit+1)), errTooManyOrders)
	require.ErrorIs(t, e.CancelFuturesPlanOrders(t.Context(), []FuturesCancelPlanOrderRequest{{OrderID: "1"}}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, e.CancelFuturesPlanOrders(t.Context(), []FuturesCancelPlanOrderRequest{{Symbol: "BTC_USDT"}}), order.ErrOrderIDNotSet)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/planorder/cancel",
		body:   `[{"symbol":"BTC_USDT","orderId":"1"},{"symbol":"ETH_USDT","orderId":"2"}]`,
	}), liveUnavailable)
	require.NoError(t, ex.CancelFuturesPlanOrders(t.Context(), []FuturesCancelPlanOrderRequest{{Symbol: "BTC_USDT", OrderID: "1"}, {Symbol: "ETH_USDT", OrderID: "2"}}),
		"CancelFuturesPlanOrders must not error")
}

func TestCancelAllFuturesPlanOrders(t *testing.T) {
	t.Parallel()
	for pair, body := range map[currency.Pair]string{futuresTradablePair: `{"symbol":"BTC_USDT"}`, currency.EMPTYPAIR: `{}`} {
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/planorder/cancel_all", body: body}), liveUnavailable)
		require.NoErrorf(t, ex.CancelAllFuturesPlanOrders(t.Context(), pair), "CancelAllFuturesPlanOrders must not error for %q", pair)
	}
}

func TestPlaceFuturesTakeProfitStopLossOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesTakeProfitStopLossOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossRequest{Volume: 1, StopLossPrice: 1, LossTrend: 1, ProfitTrend: 1})
	require.ErrorIs(t, err, errPositionIDRequired)
	_, err = e.PlaceFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossRequest{PositionID: 1, StopLossPrice: 1, LossTrend: 1, ProfitTrend: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.PlaceFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossRequest{PositionID: 1, Volume: 1, LossTrend: 1, ProfitTrend: 1})
	require.ErrorIs(t, err, errInvalidTakeProfitStopLoss)
	_, err = e.PlaceFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossRequest{PositionID: 1, Volume: 1, StopLossPrice: 1, ProfitTrend: 1})
	require.ErrorIs(t, err, errInvalidPriceType, "PlaceFuturesTakeProfitStopLossOrder must refuse a request without both price types, which the venue requires")

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/stoporder/place",
		body:   `{"positionId":1109973831,"vol":5,"lossTrend":1,"profitTrend":2,"stopLossPrice":100000,"takeProfitPrice":120000}`,
		data:   `"357859177"`,
	}), liveUnavailable)
	result, err := ex.PlaceFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossRequest{
		PositionID: 1109973831, Volume: 5, LossTrend: FuturesPriceTypeLatest, ProfitTrend: FuturesPriceTypeFair, StopLossPrice: 100000, TakeProfitPrice: 120000,
	})
	require.NoError(t, err, "PlaceFuturesTakeProfitStopLossOrder must not error")
	assert.Equal(t, FuturesID("357859177"), result, "the order's ID should be returned")
}

func TestCancelFuturesTakeProfitStopLossOrders(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelFuturesTakeProfitStopLossOrders(t.Context(), nil), errStopPlanOrderIDRequired)
	require.ErrorIs(t, e.CancelFuturesTakeProfitStopLossOrders(t.Context(), []uint64{1, 0}), errStopPlanOrderIDRequired)
	ids := make([]uint64, futuresBatchLimit+1)
	for i := range ids {
		ids[i] = uint64(i) + 1
	}
	require.ErrorIs(t, e.CancelFuturesTakeProfitStopLossOrders(t.Context(), ids), errTooManyOrders)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/stoporder/cancel",
		body:   `[{"stopPlanOrderId":1},{"stopPlanOrderId":2}]`,
	}), liveUnavailable)
	require.NoError(t, ex.CancelFuturesTakeProfitStopLossOrders(t.Context(), []uint64{1, 2}), "CancelFuturesTakeProfitStopLossOrders must not error")
}

func TestCancelAllFuturesTakeProfitStopLossOrders(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		positionID uint64
		pair       currency.Pair
		body       string
	}{
		"position": {positionID: 1109973831, body: `{"positionId":1109973831}`},
		"contract": {pair: futuresTradablePair, body: `{"symbol":"BTC_USDT"}`},
		"account":  {body: `{}`},
	} {
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/stoporder/cancel_all", body: tc.body}), liveUnavailable)
		require.NoErrorf(t, ex.CancelAllFuturesTakeProfitStopLossOrders(t.Context(), tc.positionID, tc.pair), "CancelAllFuturesTakeProfitStopLossOrders must not error for a %s", name)
	}
}

func TestModifyFuturesLimitOrderTakeProfitStopLoss(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ModifyFuturesLimitOrderTakeProfitStopLoss(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ModifyFuturesLimitOrderTakeProfitStopLoss(t.Context(), &FuturesModifyTakeProfitStopLossRequest{}), order.ErrOrderIDNotSet)
	require.ErrorIs(t, e.ModifyFuturesLimitOrderTakeProfitStopLoss(t.Context(), &FuturesModifyTakeProfitStopLossRequest{OrderID: 1, LossTrend: 4}), errInvalidPriceType)

	// The limit order is named by its order ID alone, so a take-profit and stop-loss order ID is not sent
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/stoporder/change_price",
		body:   `{"orderId":739204819784386560,"stopLossPrice":100000}`,
	}), liveUnavailable)
	require.NoError(t, ex.ModifyFuturesLimitOrderTakeProfitStopLoss(t.Context(), &FuturesModifyTakeProfitStopLossRequest{OrderID: 739204819784386560, StopPlanOrderID: 9, StopLossPrice: 100000}),
		"ModifyFuturesLimitOrderTakeProfitStopLoss must not error")
}

func TestModifyFuturesTakeProfitStopLossOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ModifyFuturesTakeProfitStopLossOrder(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ModifyFuturesTakeProfitStopLossOrder(t.Context(), &FuturesModifyTakeProfitStopLossRequest{StopLossPrice: 1}), errStopPlanOrderIDRequired)
	require.ErrorIs(t, e.ModifyFuturesTakeProfitStopLossOrder(t.Context(), &FuturesModifyTakeProfitStopLossRequest{StopPlanOrderID: 1}), errInvalidTakeProfitStopLoss)
	require.ErrorIs(t, e.ModifyFuturesTakeProfitStopLossOrder(t.Context(), &FuturesModifyTakeProfitStopLossRequest{StopPlanOrderID: 1, StopLossPrice: 1, ProfitTrend: 4}), errInvalidPriceType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/stoporder/change_plan_price",
		body:   `{"stopPlanOrderId":357859177,"takeProfitPrice":125000,"profitTrend":3}`,
	}), liveUnavailable)
	require.NoError(t, ex.ModifyFuturesTakeProfitStopLossOrder(t.Context(), &FuturesModifyTakeProfitStopLossRequest{OrderID: 7, StopPlanOrderID: 357859177, TakeProfitPrice: 125000, ProfitTrend: FuturesPriceTypeIndex}),
		"ModifyFuturesTakeProfitStopLossOrder must not error")
}

func TestModifyFuturesPlanOrderTakeProfitStopLoss(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ModifyFuturesPlanOrderTakeProfitStopLoss(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ModifyFuturesPlanOrderTakeProfitStopLoss(t.Context(), &FuturesModifyPlanOrderTakeProfitStopLossRequest{OrderID: 1}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, e.ModifyFuturesPlanOrderTakeProfitStopLoss(t.Context(), &FuturesModifyPlanOrderTakeProfitStopLossRequest{Symbol: "BTC_USDT"}), order.ErrOrderIDNotSet)
	require.ErrorIs(t, e.ModifyFuturesPlanOrderTakeProfitStopLoss(t.Context(), &FuturesModifyPlanOrderTakeProfitStopLossRequest{Symbol: "BTC_USDT", OrderID: 1, LossTrend: 9}), errInvalidPriceType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/planorder/change_stop_order",
		body:   `{"symbol":"BTC_USDT","orderId":739206374277809664,"lossTrend":1,"stopLossPrice":95000}`,
	}), liveUnavailable)
	require.NoError(t, ex.ModifyFuturesPlanOrderTakeProfitStopLoss(t.Context(), &FuturesModifyPlanOrderTakeProfitStopLossRequest{
		Symbol: "BTC_USDT", OrderID: 739206374277809664, LossTrend: FuturesPriceTypeLatest, StopLossPrice: 95000,
	}), "ModifyFuturesPlanOrderTakeProfitStopLoss must not error")
}

func TestGetFuturesTakeProfitStopLossOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTakeProfitStopLossOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesTakeProfitStopLossOrders(t.Context(), &FuturesTakeProfitStopLossOrdersRequest{PositionType: 3})
	require.ErrorIs(t, err, errInvalidPositionType)

	finished := false
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/stoporder/list/orders",
		query:  "is_finished=0&page_num=1&state=1&symbol=BTC_USDT&type=2",
		data: `{"pageSize":20,"totalCount":1,"totalPage":1,"currentPage":1,"resultList":[{"id":357859177,"orderId":0,"symbol":"BTC_USDT",` +
			`"positionId":"1027177055","stopLossPrice":0.0,"takeProfitPrice":0.0,"state":1,"triggerSide":0,"positionType":2,"vol":2,` +
			`"realityVol":0.0,"placeOrderId":0,"errorCode":0,"version":0,"isFinished":0,"createTime":"","updateTime":""}]}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesTakeProfitStopLossOrders(t.Context(), &FuturesTakeProfitStopLossOrdersRequest{Symbol: futuresTradablePair, IsFinished: &finished, State: 1, PositionType: FuturesPositionTypeShort})
	require.NoError(t, err, "GetFuturesTakeProfitStopLossOrders must not error")
	if !mockTests {
		return
	}
	// The position ID is sent as a string, and the order IDs of 0 and the empty times are read as unset
	exp := []*FuturesTakeProfitStopLossOrderResponse{{ID: "357859177", Symbol: "BTC_USDT", PositionID: "1027177055", State: 1, PositionType: FuturesPositionTypeShort, Volume: 2}}
	assert.Equal(t, exp, result, "GetFuturesTakeProfitStopLossOrders should decode every field")
}

func TestGetFuturesOpenTakeProfitStopLossOrders(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/stoporder/open_orders",
		query:  "symbol=BTC_USDT",
		data: `[{"id":357859177,"orderId":"720733527158642176","symbol":"BTC_USDT","positionId":"1027177055","lossTrend":1,` +
			`"profitTrend":1,"stopLossPrice":111325,"takeProfitPrice":113573.9,"state":3,"triggerSide":2,"positionType":1,"vol":2,` +
			`"realityVol":2,"placeOrderId":"720744238723187712","errorCode":0,"version":2,"isFinished":1,"priceProtect":0,` +
			`"profitLossVolType":"SEPARATE","takeProfitVol":2,"stopLossVol":2,"createTime":1757506663000,"updateTime":1757509217000,` +
			`"volType":1,"takeProfitReverse":2,"stopLossReverse":2,"closeTryTimes":0,"reverseTryTimes":0,"reverseErrorCode":0,"takeProfitType":0,` +
			`"profit_LOSS_VOL_TYPE_SAME":"SAME","profit_LOSS_VOL_TYPE_DIFFERENT":"SEPARATE"}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesOpenTakeProfitStopLossOrders(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesOpenTakeProfitStopLossOrders must not error")
	if !mockTests {
		return
	}
	// The example's profit_LOSS_VOL_TYPE fields name the volume types profitLossVolType takes, so they are not decoded
	exp := []*FuturesTakeProfitStopLossOrderResponse{{
		ID:                   "357859177",
		OrderID:              "720733527158642176",
		Symbol:               "BTC_USDT",
		PositionID:           "1027177055",
		LossTrend:            FuturesPriceTypeLatest,
		ProfitTrend:          FuturesPriceTypeLatest,
		StopLossPrice:        111325,
		TakeProfitPrice:      113573.9,
		State:                3,
		TriggerSide:          2,
		PositionType:         FuturesPositionTypeLong,
		Volume:               2,
		RealityVolume:        2,
		PlaceOrderID:         "720744238723187712",
		IsFinished:           1,
		Version:              2,
		ProfitLossVolumeType: "SEPARATE",
		TakeProfitVolume:     2,
		StopLossVolume:       2,
		VolumeType:           1,
		TakeProfitReverse:    2,
		StopLossReverse:      2,
		CreateTime:           types.Time(time.UnixMilli(1757506663000)),
		UpdateTime:           types.Time(time.UnixMilli(1757509217000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesOpenTakeProfitStopLossOrders should decode every field")
}

func TestValidateTrailing(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateTrailing(FuturesPriceTypeLatest, FuturesCallbackPercentage, 0.02), "a percentage callback must be valid")
	assert.ErrorIs(t, validateTrailing(0, FuturesCallbackPercentage, 0.02), errInvalidPriceType)
	assert.ErrorIs(t, validateTrailing(FuturesPriceTypeLatest, 3, 0.02), errInvalidCallbackType)
	assert.ErrorIs(t, validateTrailing(FuturesPriceTypeLatest, FuturesCallbackAbsolute, 0), limits.ErrPriceBelowMin)
}

func TestPlaceFuturesTrailingOrder(t *testing.T) {
	t.Parallel()
	valid := func() *FuturesTrailingOrderRequest {
		return &FuturesTrailingOrderRequest{
			Symbol: "DOGE_USDT", Leverage: 2, Side: FuturesSideOpenLong, Volume: 100, OpenType: FuturesOpenTypeIsolated,
			Trend: FuturesPriceTypeLatest, BackType: FuturesCallbackPercentage, BackValue: 0.02, PositionMode: FuturesPositionModeHedge,
		}
	}
	_, err := e.PlaceFuturesTrailingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	for name, tc := range map[string]struct {
		change func(*FuturesTrailingOrderRequest)
		err    error
	}{
		"no symbol":       {func(o *FuturesTrailingOrderRequest) { o.Symbol = "" }, currency.ErrSymbolStringEmpty},
		"no leverage":     {func(o *FuturesTrailingOrderRequest) { o.Leverage = 0 }, errInvalidLeverage},
		"no side":         {func(o *FuturesTrailingOrderRequest) { o.Side = 0 }, errInvalidOrderSide},
		"no volume":       {func(o *FuturesTrailingOrderRequest) { o.Volume = 0 }, limits.ErrAmountBelowMin},
		"no open type":    {func(o *FuturesTrailingOrderRequest) { o.OpenType = 0 }, errInvalidOpenType},
		"no mode":         {func(o *FuturesTrailingOrderRequest) { o.PositionMode = 0 }, errInvalidPositionMode},
		"no callback":     {func(o *FuturesTrailingOrderRequest) { o.BackType = 0 }, errInvalidCallbackType},
		"no price type":   {func(o *FuturesTrailingOrderRequest) { o.Trend = 0 }, errInvalidPriceType},
		"no callback gap": {func(o *FuturesTrailingOrderRequest) { o.BackValue = 0 }, limits.ErrPriceBelowMin},
	} {
		o := valid()
		tc.change(o)
		_, err := e.PlaceFuturesTrailingOrder(t.Context(), o)
		assert.ErrorIsf(t, err, tc.err, "a trailing order with %s should be refused", name)
	}

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/trackorder/place",
		body:   `{"symbol":"DOGE_USDT","leverage":2,"side":1,"vol":100,"openType":1,"trend":1,"backType":1,"backValue":0.02,"positionMode":1}`,
		data:   `"739218627261666816"`,
	}), liveUnavailable)
	result, err := ex.PlaceFuturesTrailingOrder(t.Context(), valid())
	require.NoError(t, err, "PlaceFuturesTrailingOrder must not error")
	assert.Equal(t, FuturesID("739218627261666816"), result, "the trailing order's ID should be returned")
}

func TestCancelFuturesTrailingOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelFuturesTrailingOrder(t.Context(), futuresTradablePair, 0), errTrailingOrderIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/trackorder/cancel",
		body:   `{"symbol":"DOGE_USDT","trackOrderId":739628779353703936}`,
	}), liveUnavailable)
	require.NoError(t, ex.CancelFuturesTrailingOrder(t.Context(), currency.NewPairWithDelimiter("DOGE", "USDT", currency.UnderscoreDelimiter), 739628779353703936),
		"CancelFuturesTrailingOrder must not error")
}

func TestModifyFuturesTrailingOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ModifyFuturesTrailingOrder(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ModifyFuturesTrailingOrder(t.Context(), &FuturesModifyTrailingOrderRequest{TrackOrderID: 1}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, e.ModifyFuturesTrailingOrder(t.Context(), &FuturesModifyTrailingOrderRequest{Symbol: "DOGE_USDT"}), errTrailingOrderIDRequired)
	require.ErrorIs(t, e.ModifyFuturesTrailingOrder(t.Context(), &FuturesModifyTrailingOrderRequest{Symbol: "DOGE_USDT", TrackOrderID: 1}), limits.ErrAmountBelowMin)
	require.ErrorIs(t, e.ModifyFuturesTrailingOrder(t.Context(), &FuturesModifyTrailingOrderRequest{Symbol: "DOGE_USDT", TrackOrderID: 1, Volume: 1}), errInvalidPriceType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/trackorder/change_order",
		body:   `{"symbol":"DOGE_USDT","trackOrderId":739628779353703936,"trend":2,"activePrice":0.19,"backType":2,"backValue":0.001,"vol":50}`,
	}), liveUnavailable)
	require.NoError(t, ex.ModifyFuturesTrailingOrder(t.Context(), &FuturesModifyTrailingOrderRequest{
		Symbol: "DOGE_USDT", TrackOrderID: 739628779353703936, Trend: FuturesPriceTypeFair, ActivePrice: 0.19, BackType: FuturesCallbackAbsolute, BackValue: 0.001, Volume: 50,
	}), "ModifyFuturesTrailingOrder must not error")
}

func TestGetFuturesTrailingOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTrailingOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesTrailingOrders(t.Context(), &FuturesTrailingOrdersRequest{})
	require.ErrorIs(t, err, errOrderStatesRequired)
	_, err = e.GetFuturesTrailingOrders(t.Context(), &FuturesTrailingOrdersRequest{States: []uint8{0}, Side: 7})
	require.ErrorIs(t, err, errInvalidOrderSide)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/trackorder/list/orders",
		query:  "pageIndex=1&pageSize=10&states=0%2C1",
		data: `[{"id":"739628779353703936","symbol":"DOGE_USDT","leverage":2,"side":1,"vol":100,"openType":1,"trend":1,` +
			`"activePrice":0,"markPrice":0.18657,"backType":1,"backValue":0.02,"triggerPrice":0.1903,"orderId":0,"errorCode":0,` +
			`"state":1,"createTime":1762011642623,"updateTime":1762011642623,"positionMode":1,"reduceOnly":false,"triggerType":1}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesTrailingOrders(t.Context(), &FuturesTrailingOrdersRequest{States: []uint8{0, 1}, PageIndex: 1, PageSize: 10})
	require.NoError(t, err, "GetFuturesTrailingOrders must not error")
	if !mockTests {
		return
	}
	// An untriggered order carries no placed order
	exp := []*FuturesTrailingOrderResponse{{
		ID:           "739628779353703936",
		Symbol:       "DOGE_USDT",
		Leverage:     2,
		Side:         FuturesSideOpenLong,
		Volume:       100,
		OpenType:     FuturesOpenTypeIsolated,
		Trend:        FuturesPriceTypeLatest,
		MarkPrice:    0.18657,
		BackType:     FuturesCallbackPercentage,
		BackValue:    0.02,
		TriggerPrice: 0.1903,
		TriggerType:  FuturesTriggerGreaterOrEqual,
		State:        1,
		PositionMode: FuturesPositionModeHedge,
		CreateTime:   types.Time(time.UnixMilli(1762011642623)),
		UpdateTime:   types.Time(time.UnixMilli(1762011642623)),
	}}
	assert.Equal(t, exp, result, "GetFuturesTrailingOrders should decode every field")
}

func TestFuturesOrderIDsAboveFloatPrecision(t *testing.T) {
	t.Parallel()
	// Order IDs pass 2^53, where a float64 would round 739106551624717313 to 739106551624717312
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/get/739106551624717313",
		data:   strings.Replace(futuresFilledOrderReply, `"orderId":"739106551624717312"`, `"orderId":739106551624717313`, 1),
	}), liveUnavailable)
	result, err := ex.GetFuturesOrder(t.Context(), strconv.FormatUint(739106551624717313, 10))
	require.NoError(t, err, "GetFuturesOrder must not error")
	exp := futuresFilledOrder()
	exp.OrderID = "739106551624717313"
	assert.Equal(t, exp, result, "an order ID sent as a number should keep every digit")
}
