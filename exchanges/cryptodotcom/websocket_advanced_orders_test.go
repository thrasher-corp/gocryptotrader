package cryptodotcom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testWebsocketAPIAdvancedFailures checks a websocket method reports a null result, when it returns a result, and
// returns the error Crypto.com answers it with
func testWebsocketAPIAdvancedFailures(t *testing.T, venue *websocketVenue, name, method string, returnsResult bool, call func() error) {
	t.Helper()
	if returnsResult {
		venue.respond(method, `null`)
		assert.ErrorIsf(t, call(), common.ErrNoResponse, "%s should error for a null result", name)
	}
	venue.fail(method, &APIError{Code: 40001, Message: "BAD_REQUEST"})
	var apiErr *APIError
	require.ErrorAsf(t, call(), &apiErr, "%s must return the error Crypto.com answers with", name)
	assert.Equalf(t, &APIError{Code: 40001, Message: "BAD_REQUEST"}, apiErr, "%s should return Crypto.com's error", name)
}

// testWebsocketAPICancelContingency checks a websocket method cancelling an OCO, OTO or OTOCO sent the mock one request
// for listID, and that cancel returns the error the mock answers with
func testWebsocketAPICancelContingency(t *testing.T, venue *websocketVenue, name, method, listID string, cancel func() error) {
	t.Helper()
	requests := venue.received(method)
	require.Lenf(t, requests, 1, "%s must send one request", name)
	assert.JSONEqf(t, `{"list_id":"`+listID+`"}`, string(requests[0].Params), "%s should send the list ID", name)
	testWebsocketAPIAdvancedFailures(t, venue, name, method, false, cancel)
}

func TestWebsocketAPICreateAdvancedOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	_, err := refuser.CreateAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAdvancedOrder must refuse a nil request")
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range triggerOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			req := advancedTriggerOrder()
			tc.change(&req)
			_, err := refuser.CreateAdvancedOrder(t.Context(), &req)
			assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrder should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := advancedTriggerOrder()
	if !mockTests {
		req.ClientOrderID = newTradingTestClientOrderID()
	}
	result, err := ex.WebsocketAPI().CreateAdvancedOrder(t.Context(), &req)
	require.NoError(t, err, "CreateAdvancedOrder must not error")
	if !mockTests {
		require.NotEmpty(t, result.OrderID, "CreateAdvancedOrder must return the order ID")
		cancelAdvancedOrderAfterTest(t, result.OrderID)
		assert.Equal(t, req.ClientOrderID, result.ClientOrderID, "CreateAdvancedOrder should return the order's client order ID")
		return
	}
	exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
	assert.Equal(t, exp, result, "CreateAdvancedOrder should decode every field")
	requests := venue.received("private/advanced/create-order")
	require.Len(t, requests, 1, "CreateAdvancedOrder must send one request")
	assert.JSONEq(t, `{"instrument_name":"CRO_USD","side":"SELL","type":"STOP_LIMIT","quantity":"10","price":"0.0279","ref_price":"0.028","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`, string(requests[0].Params), "CreateAdvancedOrder should send the order")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateAdvancedOrder", "private/advanced/create-order", true, func() error {
		_, err := ex.WebsocketAPI().CreateAdvancedOrder(t.Context(), &req)
		return err
	})
}

func TestWebsocketAPIAmendAdvancedOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range amendAdvancedOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.AmendAdvancedOrder(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "AmendAdvancedOrder should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0279}
	if !mockTests {
		req.OrderID = placeAdvancedTestOrder(t).OrderID
		cancelAdvancedOrderAfterTest(t, req.OrderID)
	}
	result, err := ex.WebsocketAPI().AmendAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "AmendAdvancedOrder must not error")
	if !mockTests {
		assert.Equal(t, req.OrderID, result.OrderID, "AmendAdvancedOrder should return the amended order's ID")
		return
	}
	exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
	assert.Equal(t, exp, result, "AmendAdvancedOrder should decode every field")
	requests := venue.received("private/advanced/amend-order")
	require.Len(t, requests, 1, "AmendAdvancedOrder must send one request")
	assert.JSONEq(t, `{"order_id":"5755600460443882762","new_quantity":"10","new_price":"0","new_ref_price":"0.0279"}`, string(requests[0].Params), "AmendAdvancedOrder should send the amendment, its zero price as 0")
	testWebsocketAPIAdvancedFailures(t, venue, "AmendAdvancedOrder", "private/advanced/amend-order", true, func() error {
		_, err := ex.WebsocketAPI().AmendAdvancedOrder(t.Context(), req)
		return err
	})
}

func TestWebsocketAPICreateOneCancelsTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneCancelsTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneCancelsTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneCancelsTheOtherOrder should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(ocoOrders()))
	require.NoError(t, err, "CreateOneCancelsTheOtherOrder must not error")
	if !mockTests {
		require.NotEmpty(t, result.ListID, "CreateOneCancelsTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneCancelsTheOtherOrder)
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120100"}
	assert.Equal(t, exp, result, "CreateOneCancelsTheOtherOrder should decode every field")
	requests := venue.received("private/advanced/create-oco")
	require.Len(t, requests, 1, "CreateOneCancelsTheOtherOrder must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"LIMIT","quantity":"0.0001","price":"120000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"40000"}]}`, string(requests[0].Params), "CreateOneCancelsTheOtherOrder should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOneCancelsTheOtherOrder", "private/advanced/create-oco", true, func() error {
		_, err := ex.WebsocketAPI().CreateOneCancelsTheOtherOrder(t.Context(), ocoOrders())
		return err
	})
}

func TestWebsocketAPICancelOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	err := newAdvancedRefusalExchange(t).WebsocketAPI().CancelOneCancelsTheOtherOrder(t.Context(), "")
	require.ErrorIs(t, err, errListIDRequired, "CancelOneCancelsTheOtherOrder must refuse an empty list ID")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	placed, err := ex.WebsocketAPI().CreateOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(ocoOrders()))
	require.NoError(t, err, "CreateOneCancelsTheOtherOrder must place the OCO the test cancels")
	listID := string(placed.ListID)
	require.NoError(t, ex.WebsocketAPI().CancelOneCancelsTheOtherOrder(t.Context(), listID), "CancelOneCancelsTheOtherOrder must not error")
	if !mockTests {
		return
	}
	testWebsocketAPICancelContingency(t, venue, "CancelOneCancelsTheOtherOrder", "private/advanced/cancel-oco", listID, func() error {
		return ex.WebsocketAPI().CancelOneCancelsTheOtherOrder(t.Context(), listID)
	})
}

func TestWebsocketAPICreateOneTriggersTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneTriggersTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneTriggersTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneTriggersTheOtherOrder should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOneTriggersTheOtherOrder(t.Context(), contingencyTestOrders(otoOrders()))
	require.NoError(t, err, "CreateOneTriggersTheOtherOrder must not error")
	if !mockTests {
		require.NotEmpty(t, result.ListID, "CreateOneTriggersTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneTriggersTheOtherOrder)
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120199"}
	assert.Equal(t, exp, result, "CreateOneTriggersTheOtherOrder should decode every field")
	requests := venue.received("private/advanced/create-oto")
	require.Len(t, requests, 1, "CreateOneTriggersTheOtherOrder must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"}]}`, string(requests[0].Params), "CreateOneTriggersTheOtherOrder should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOneTriggersTheOtherOrder", "private/advanced/create-oto", true, func() error {
		_, err := ex.WebsocketAPI().CreateOneTriggersTheOtherOrder(t.Context(), otoOrders())
		return err
	})
}

func TestWebsocketAPICancelOneTriggersTheOtherOrder(t *testing.T) {
	t.Parallel()
	err := newAdvancedRefusalExchange(t).WebsocketAPI().CancelOneTriggersTheOtherOrder(t.Context(), "")
	require.ErrorIs(t, err, errListIDRequired, "CancelOneTriggersTheOtherOrder must refuse an empty list ID")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	placed, err := ex.WebsocketAPI().CreateOneTriggersTheOtherOrder(t.Context(), contingencyTestOrders(otoOrders()))
	require.NoError(t, err, "CreateOneTriggersTheOtherOrder must place the OTO the test cancels")
	listID := string(placed.ListID)
	require.NoError(t, ex.WebsocketAPI().CancelOneTriggersTheOtherOrder(t.Context(), listID), "CancelOneTriggersTheOtherOrder must not error")
	if !mockTests {
		return
	}
	testWebsocketAPICancelContingency(t, venue, "CancelOneTriggersTheOtherOrder", "private/advanced/cancel-oto", listID, func() error {
		return ex.WebsocketAPI().CancelOneTriggersTheOtherOrder(t.Context(), listID)
	})
}

func TestWebsocketAPICreateOneTriggersAOneCancelsTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneTriggersAOneCancelsTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneTriggersAOneCancelsTheOtherOrder should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(otocoOrders()))
	require.NoError(t, err, "CreateOneTriggersAOneCancelsTheOtherOrder must not error")
	if !mockTests {
		require.NotEmpty(t, result.ListID, "CreateOneTriggersAOneCancelsTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneTriggersAOneCancelsTheOtherOrder)
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120243"}
	assert.Equal(t, exp, result, "CreateOneTriggersAOneCancelsTheOtherOrder should decode every field")
	requests := venue.received("private/advanced/create-otoco")
	require.Len(t, requests, 1, "CreateOneTriggersAOneCancelsTheOtherOrder must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","quantity":"0.0001","ref_price":"120000"}]}`, string(requests[0].Params), "CreateOneTriggersAOneCancelsTheOtherOrder should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOneTriggersAOneCancelsTheOtherOrder", "private/advanced/create-otoco", true, func() error {
		_, err := ex.WebsocketAPI().CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), otocoOrders())
		return err
	})
}

func TestWebsocketAPICancelOneTriggersAOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	err := newAdvancedRefusalExchange(t).WebsocketAPI().CancelOneTriggersAOneCancelsTheOtherOrder(t.Context(), "")
	require.ErrorIs(t, err, errListIDRequired, "CancelOneTriggersAOneCancelsTheOtherOrder must refuse an empty list ID")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	placed, err := ex.WebsocketAPI().CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(otocoOrders()))
	require.NoError(t, err, "CreateOneTriggersAOneCancelsTheOtherOrder must place the OTOCO the test cancels")
	listID := string(placed.ListID)
	require.NoError(t, ex.WebsocketAPI().CancelOneTriggersAOneCancelsTheOtherOrder(t.Context(), listID), "CancelOneTriggersAOneCancelsTheOtherOrder must not error")
	if !mockTests {
		return
	}
	testWebsocketAPICancelContingency(t, venue, "CancelOneTriggersAOneCancelsTheOtherOrder", "private/advanced/cancel-otoco", listID, func() error {
		return ex.WebsocketAPI().CancelOneTriggersAOneCancelsTheOtherOrder(t.Context(), listID)
	})
}

func TestWebsocketAPICancelAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	_, err := refuser.CancelAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAdvancedOrder must refuse a nil request")
	_, err = refuser.CancelAdvancedOrder(t.Context(), &AdvancedOrderIDRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "CancelAdvancedOrder must refuse a request without an order ID or client order ID")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &AdvancedOrderIDRequest{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
	if !mockTests {
		req.ClientOrderID = placeAdvancedTestOrder(t).ClientOrderID
	}
	result, err := ex.WebsocketAPI().CancelAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "CancelAdvancedOrder must not error")
	if !mockTests {
		assert.Equal(t, req.ClientOrderID, result.ClientOrderID, "CancelAdvancedOrder should return the cancelled order's client order ID")
		return
	}
	exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
	assert.Equal(t, exp, result, "CancelAdvancedOrder should decode every field")
	requests := venue.received("private/advanced/cancel-order")
	require.Len(t, requests, 1, "CancelAdvancedOrder must send one request")
	assert.JSONEq(t, `{"client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`, string(requests[0].Params), "CancelAdvancedOrder should send the client order ID")
	testWebsocketAPIAdvancedFailures(t, venue, "CancelAdvancedOrder", "private/advanced/cancel-order", true, func() error {
		_, err := ex.WebsocketAPI().CancelAdvancedOrder(t.Context(), req)
		return err
	})
}

func TestWebsocketAPICancelAllAdvancedOrders(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range cancelAllAdvancedOrdersRefusals {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, refuser.CancelAllAdvancedOrders(t.Context(), tc.req), tc.err, "CancelAllAdvancedOrders should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeAll, PositionType: PositionTypeIsolatedMargin}
	require.NoError(t, ex.WebsocketAPI().CancelAllAdvancedOrders(t.Context(), req), "CancelAllAdvancedOrders must not error")
	if !mockTests {
		return
	}
	requests := venue.received("private/advanced/cancel-all-orders")
	require.Len(t, requests, 1, "CancelAllAdvancedOrders must send one request")
	assert.JSONEq(t, `{"instrument_name":"BTCUSD-PERP","type":"ALL","position_type":"ISOLATED_MARGIN"}`, string(requests[0].Params), "CancelAllAdvancedOrders should send the selection")
	testWebsocketAPIAdvancedFailures(t, venue, "CancelAllAdvancedOrders", "private/advanced/cancel-all-orders", false, func() error {
		return ex.WebsocketAPI().CancelAllAdvancedOrders(t.Context(), req)
	})
}

func TestWebsocketAPICreateAdvancedOrderList(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range createAdvancedOrderListRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateAdvancedOrderList(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrderList should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateAdvancedOrderList(t.Context(), advancedOrderListTestOrders())
	require.NoError(t, err, "CreateAdvancedOrderList must not error")
	if !mockTests {
		for i := range result {
			if result[i].Code == 0 {
				cancelAdvancedOrderAfterTest(t, result[i].OrderID)
			}
		}
		assertListOutcomes(t, result, len(advancedOrderListOrders()), "CreateAdvancedOrderList", "place")
		return
	}
	exp := []OrderListEntryResponse{
		{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"},
		{Index: 1, OrderID: "289991951579667815", ClientOrderID: "my_tp_order"},
	}
	assert.Equal(t, exp, result, "CreateAdvancedOrderList should decode every field")
	requests := venue.received("private/advanced/create-order-list")
	require.Len(t, requests, 1, "CreateAdvancedOrderList must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"40000","client_oid":"my_sl_order"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","quantity":"0.0001","ref_price":"120000","client_oid":"my_tp_order"}]}`, string(requests[0].Params), "CreateAdvancedOrderList should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateAdvancedOrderList", "private/advanced/create-order-list", true, func() error {
		_, err := ex.WebsocketAPI().CreateAdvancedOrderList(t.Context(), advancedOrderListOrders())
		return err
	})
}

func TestWebsocketAPICancelAdvancedOrderList(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range cancelAdvancedOrderListRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CancelAdvancedOrderList(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CancelAdvancedOrderList should refuse %s", tc.name)
		})
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	orders := []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}, {ClientOrderID: "my_tp_order"}}
	if !mockTests {
		first := placeAdvancedTestOrder(t)
		cancelAdvancedOrderIfTestFails(t, first.OrderID)
		second := placeAdvancedTestOrder(t)
		cancelAdvancedOrderIfTestFails(t, second.OrderID)
		orders = []AdvancedOrderIDRequest{{OrderID: first.OrderID}, {ClientOrderID: second.ClientOrderID}}
	}
	result, err := ex.WebsocketAPI().CancelAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelAdvancedOrderList must not error")
	require.Len(t, result, len(orders), "CancelAdvancedOrderList must return each cancel's outcome")
	if !mockTests {
		assertListOutcomes(t, result, len(orders), "CancelAdvancedOrderList", "cancel")
		return
	}
	exp := []OrderListEntryResponse{{}, {Index: 1}}
	assert.Equal(t, exp, result, "CancelAdvancedOrderList should decode every field")
	requests := venue.received("private/advanced/cancel-order-list")
	require.Len(t, requests, 1, "CancelAdvancedOrderList must send one request")
	assert.JSONEq(t, `{"order_list":[{"order_id":"289991951579667814"},{"client_oid":"my_tp_order"}]}`, string(requests[0].Params), "CancelAdvancedOrderList should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CancelAdvancedOrderList", "private/advanced/cancel-order-list", true, func() error {
		_, err := ex.WebsocketAPI().CancelAdvancedOrderList(t.Context(), orders)
		return err
	})
}
