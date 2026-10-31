package cryptodotcom

import (
	"context"
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

// testWebsocketAPICancelContingency tests a websocket method cancelling an OCO, OTO or OTOCO by its list ID
func testWebsocketAPICancelContingency(t *testing.T, name, method, listID string, cancel func(WebsocketAPI, context.Context, string) error) {
	t.Helper()
	require.ErrorIsf(t, cancel(newAdvancedRefusalExchange(t).WebsocketAPI(), t.Context(), ""), errListIDRequired, "%s must refuse an empty list ID", name)
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	require.NoErrorf(t, cancel(ex.WebsocketAPI(), t.Context(), listID), "%s must not error", name)
	if !mockTests {
		return
	}
	requests := venue.received(method)
	require.Lenf(t, requests, 1, "%s must send one request", name)
	assert.JSONEqf(t, `{"list_id":"`+listID+`"}`, string(requests[0].Params), "%s should send the list ID", name)
	testWebsocketAPIAdvancedFailures(t, venue, name, method, false, func() error {
		return cancel(ex.WebsocketAPI(), t.Context(), listID)
	})
}

func TestWebsocketAPICreateAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	_, err := refuser.CreateAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAdvancedOrder must refuse a nil request")
	for _, tc := range triggerOrderRefusals {
		req := advancedTriggerOrder()
		tc.change(&req)
		_, err := refuser.CreateAdvancedOrder(t.Context(), &req)
		assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrder should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := advancedTriggerOrder()
	result, err := ex.WebsocketAPI().CreateAdvancedOrder(t.Context(), &req)
	require.NoError(t, err, "CreateAdvancedOrder must not error")
	if !mockTests {
		assert.NotEmpty(t, result.OrderID, "CreateAdvancedOrder should return the order ID")
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

func TestWebsocketAPIAmendAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range amendAdvancedOrderRefusals {
		_, err := refuser.AmendAdvancedOrder(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "AmendAdvancedOrder should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0279}
	result, err := ex.WebsocketAPI().AmendAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "AmendAdvancedOrder must not error")
	if !mockTests {
		assert.NotEmpty(t, result.OrderID, "AmendAdvancedOrder should return the order ID")
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

func TestWebsocketAPICreateOCO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range ocoRefusals() {
		_, err := refuser.CreateOCO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOCO should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOCO(t.Context(), ocoOrders())
	require.NoError(t, err, "CreateOCO must not error")
	if !mockTests {
		assert.NotEmpty(t, result.ListID, "CreateOCO should return the list ID")
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120100"}
	assert.Equal(t, exp, result, "CreateOCO should decode every field")
	requests := venue.received("private/advanced/create-oco")
	require.Len(t, requests, 1, "CreateOCO must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"LIMIT","quantity":"0.0001","price":"120000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"40000"}]}`, string(requests[0].Params), "CreateOCO should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOCO", "private/advanced/create-oco", true, func() error {
		_, err := ex.WebsocketAPI().CreateOCO(t.Context(), ocoOrders())
		return err
	})
}

func TestWebsocketAPICancelOCO(t *testing.T) {
	t.Parallel()
	testWebsocketAPICancelContingency(t, "CancelOCO", "private/advanced/cancel-oco", "6498090546073120100", WebsocketAPI.CancelOCO)
}

func TestWebsocketAPICreateOTO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range otoRefusals() {
		_, err := refuser.CreateOTO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOTO should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOTO(t.Context(), otoOrders())
	require.NoError(t, err, "CreateOTO must not error")
	if !mockTests {
		assert.NotEmpty(t, result.ListID, "CreateOTO should return the list ID")
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120199"}
	assert.Equal(t, exp, result, "CreateOTO should decode every field")
	requests := venue.received("private/advanced/create-oto")
	require.Len(t, requests, 1, "CreateOTO must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"}]}`, string(requests[0].Params), "CreateOTO should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOTO", "private/advanced/create-oto", true, func() error {
		_, err := ex.WebsocketAPI().CreateOTO(t.Context(), otoOrders())
		return err
	})
}

func TestWebsocketAPICancelOTO(t *testing.T) {
	t.Parallel()
	testWebsocketAPICancelContingency(t, "CancelOTO", "private/advanced/cancel-oto", "6498090546073120199", WebsocketAPI.CancelOTO)
}

func TestWebsocketAPICreateOTOCO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range otocoRefusals() {
		_, err := refuser.CreateOTOCO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOTOCO should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateOTOCO(t.Context(), otocoOrders())
	require.NoError(t, err, "CreateOTOCO must not error")
	if !mockTests {
		assert.NotEmpty(t, result.ListID, "CreateOTOCO should return the list ID")
		return
	}
	exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120243"}
	assert.Equal(t, exp, result, "CreateOTOCO should decode every field")
	requests := venue.received("private/advanced/create-otoco")
	require.Len(t, requests, 1, "CreateOTOCO must send one request")
	assert.JSONEq(t, `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","quantity":"0.0001","ref_price":"120000"}]}`, string(requests[0].Params), "CreateOTOCO should send the orders")
	testWebsocketAPIAdvancedFailures(t, venue, "CreateOTOCO", "private/advanced/create-otoco", true, func() error {
		_, err := ex.WebsocketAPI().CreateOTOCO(t.Context(), otocoOrders())
		return err
	})
}

func TestWebsocketAPICancelOTOCO(t *testing.T) {
	t.Parallel()
	testWebsocketAPICancelContingency(t, "CancelOTOCO", "private/advanced/cancel-otoco", "6498090546073120243", WebsocketAPI.CancelOTOCO)
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
	result, err := ex.WebsocketAPI().CancelAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "CancelAdvancedOrder must not error")
	if !mockTests {
		assert.NotEmpty(t, result.OrderID, "CancelAdvancedOrder should return the order ID")
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

func TestWebsocketAPICancelAllAdvancedOrders(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range cancelAllAdvancedOrdersRefusals {
		assert.ErrorIsf(t, refuser.CancelAllAdvancedOrders(t.Context(), tc.req), tc.err, "CancelAllAdvancedOrders should refuse %s", tc.name)
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

func TestWebsocketAPICreateAdvancedOrderList(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range createAdvancedOrderListRefusals() {
		_, err := refuser.CreateAdvancedOrderList(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrderList should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateAdvancedOrderList(t.Context(), advancedOrderListOrders())
	require.NoError(t, err, "CreateAdvancedOrderList must not error")
	if !mockTests {
		assert.Len(t, result, len(advancedOrderListOrders()), "CreateAdvancedOrderList should return each order's outcome")
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

func TestWebsocketAPICancelAdvancedOrderList(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t).WebsocketAPI()
	for _, tc := range cancelAdvancedOrderListRefusals() {
		_, err := refuser.CancelAdvancedOrderList(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CancelAdvancedOrderList should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	orders := []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}, {ClientOrderID: "my_tp_order"}}
	result, err := ex.WebsocketAPI().CancelAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelAdvancedOrderList must not error")
	if !mockTests {
		assert.Len(t, result, len(orders), "CancelAdvancedOrderList should return each cancel's outcome")
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
