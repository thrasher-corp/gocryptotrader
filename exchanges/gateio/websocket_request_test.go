package gateio

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var loginResponse = []byte(`{"header":{"response_time":"1722227146659","status":"200","channel":"spot.login","event":"api","client_id":"14.203.57.50-0xc11df96f20"},"data":{"result":{"api_key":"4960099442600b4cfefa48ac72dacca0","uid":"2365748"}},"request_id":"1722227146427268900"}`)

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketLogin(context.Background(), nil, "bro.Login")
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	got, err := g.WebsocketLogin(context.Background(), nil, "bro.Login")
	require.NoError(t, err)

	fmt.Println(got)
}

var orderError = []byte(`{"header":{"response_time":"1722392009059","status":"400","channel":"spot.order_place","event":"api","client_id":"14.203.57.50-0xc0b61a0840","conn_id":"b5cd175a189984a6","trace_id":"f56a31478d7c6ce4ddaea3b337263233"},"data":{"errs":{"label":"INVALID_ARGUMENT","message":"OrderPlace request params error"}},"request_id":"1722392008842968100"}`)
var orderAcceptedResp = []byte(`{"header":{"response_time":"1722393719499","status":"200","channel":"spot.order_place","event":"api","client_id":"14.203.57.50-0xc213dab340","conn_id":"bfcbe154b8520050","trace_id":"74fbfd701d54bfe207ec79b6d2736b3a"},"data":{"result":{"req_id":"1722393719287158300","api_key":"","timestamp":"","signature":"","trace_id":"0e30c04e4e7499bccde8f83990d7168a","req_header":{"trace_id":"0e30c04e4e7499bccde8f83990d7168a"},"req_param":[{"text":"apiv4-ws","currency_pair":"BTC_USDT","type":"limit","side":"BUY","amount":"-1","price":"-1"}]}},"request_id":"1722393719287158300","ack":true}`)
var orderSecondResponseError = []byte(`{"header":{"response_time":"1722400001367","status":"400","channel":"spot.order_place","event":"api","client_id":"14.203.57.50-0xc12e5e4f20","conn_id":"4ddf3b1b45523bc3","trace_id":"8cca91e29b405e334b1901463c36afe1"},"data":{"errs":{"label":"INVALID_PARAM_VALUE","message":"label: INVALID_PARAM_VALUE, message: Your order size 0.200000 USDT is too small. The minimum is 3 USDT"}},"request_id":"1722400001142974600"}`)
var orderSecondResponseSuccess = []byte(`{"header":{"response_time":"1722400187811","status":"200","channel":"spot.order_place","event":"api","client_id":"14.203.57.50-0xc1b81a7340"},"data":{"result":{"left":"0.0003","update_time":"1722400187","amount":"0.0003","create_time":"1722400187","price":"20000","finish_as":"open","time_in_force":"gtc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1722400187564025900","status":"open","iceberg":"0","filled_total":"0","id":"644865690097","fill_price":"0","update_time_ms":1722400187807,"create_time_ms":1722400187807}},"request_id":"1722400187564025900"}`)
var orderBatchSuccess = []byte(`{"header":{"response_time":"1722402442822","status":"200","channel":"spot.order_place","event":"api","client_id":"14.203.57.50-0xc0e372e580"},"data":{"result":[{"account":"spot","status":"open","side":"buy","amount":"0.0003","id":"644883514616","create_time":"1722402442","update_time":"1722402442","text":"t-1722402442588484600","left":"0.0003","currency_pair":"BTC_USDT","type":"limit","finish_as":"open","price":"20000","time_in_force":"gtc","iceberg":"0","filled_total":"0","fill_price":"0","create_time_ms":1722402442819,"update_time_ms":1722402442819,"succeeded":true},{"account":"spot","status":"open","side":"buy","amount":"0.0003","id":"644883514625","create_time":"1722402442","update_time":"1722402442","text":"t-1722402442588484601","left":"0.0003","currency_pair":"BTC_USDT","type":"limit","finish_as":"open","price":"20000","time_in_force":"gtc","iceberg":"0","filled_total":"0","fill_price":"0","create_time_ms":1722402442821,"update_time_ms":1722402442821,"succeeded":true}]},"request_id":"172240244
2588484600"}`)

func TestWebsocketOrderPlace(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderPlace(context.Background(), nil, 0)
	require.ErrorIs(t, err, errBatchSliceEmpty)
	_, err = g.WebsocketOrderPlace(context.Background(), make([]WebsocketOrder, 1), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := WebsocketOrder{CurrencyPair: "BTC_USDT"}
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, 0)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Buy.String())
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, 0)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = "0.0003"
	out.Type = "limit"
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, 0)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "20000"
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, 0)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	// test single order
	got, err := g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out, out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var orderCancelError = []byte(`{"header":{"response_time":"1722405878406","status":"400","channel":"spot.order_cancel","event":"api","client_id":"14.203.57.50-0xc1e68ac6e0","conn_id":"0378a86ff109ca9a","trace_id":"b05be4753e751dff9175215ee020b578"},"data":{"errs":{"label":"INVALID_CURRENCY_PAIR","message":"label: INVALID_CURRENCY_PAIR, message: Invalid currency pair BTCUSD"}},"request_id":"1722405878175928500"}`)
var orderCancelSuccess = []byte(`{"header":{"response_time":"1722406252471","status":"200","channel":"spot.order_cancel","event":"api","client_id":"14.203.57.50-0xc2397b9e40"},"data":{"result":{"left":"0.0003","update_time":"1722406252","amount":"0.0003","create_time":"1722406069","price":"20000","finish_as":"cancelled","time_in_force":"gtc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1722406069442994700","status":"cancelled","iceberg":"0","filled_total":"0","id":"644913098758","fill_price":"0","update_time_ms":1722406252467,"create_time_ms":1722406069667}},"request_id":"1722406252236528200"}`)

func TestWebsocketOrderCancel(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderCancel(context.Background(), "", currency.EMPTYPAIR, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = g.WebsocketOrderCancel(context.Background(), "1337", currency.EMPTYPAIR, "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	btcusdt, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderCancel(context.Background(), "1337", btcusdt, "", 0)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	got, err := g.WebsocketOrderCancel(context.Background(), "644913098758", btcusdt, "", asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var cancelAllfailed = []byte(`{"header":{"response_time":"1722407703038","status":"200","channel":"spot.order_cancel_ids","event":"api","client_id":"14.203.57.50-0xc36ba50dc0"},"data":{"result":[{"currency_pair":"BTC_USDT","id":"644913098758","label":"ORDER_NOT_FOUND","message":"Order not found"}]},"request_id":"1722407702811217700"}`)
var cancelAllSuccess = []byte(`{"header":{"response_time":"1722407800393","status":"200","channel":"spot.order_cancel_ids","event":"api","client_id":"14.203.57.50-0xc0ae1ed8c0"},"data":{"result":[{"currency_pair":"BTC_USDT","id":"644913101755","succeeded":true}]},"request_id":"1722407800174417400"}`)

func TestWebsocketOrderCancelAllByIDs(t *testing.T) {
	t.Parallel()
	out := WebsocketOrderCancelRequest{}
	_, err := g.WebsocketOrderCancelAllByIDs(context.Background(), []WebsocketOrderCancelRequest{out}, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	out.OrderID = "1337"
	_, err = g.WebsocketOrderCancelAllByIDs(context.Background(), []WebsocketOrderCancelRequest{out}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderCancelAllByIDs(context.Background(), []WebsocketOrderCancelRequest{out}, 0)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	out.OrderID = "644913101755"
	got, err := g.WebsocketOrderCancelAllByIDs(context.Background(), []WebsocketOrderCancelRequest{out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var cancelAllByPairSuccess = []byte(`{"header":{"response_time":"1722415590482","status":"200","channel":"spot.order_cancel_cp","event":"api","client_id":"58.169.146.133-0xc028f00b00"},"data":{"result":[{"left":"0.0003","update_time":"1722415590","amount":"0.0003","create_time":"1722406069","price":"20000","finish_as":"cancelled","time_in_force":"gtc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1722406069759058701","status":"cancelled","iceberg":"0","filled_total":"0","id":"644913101780","fill_price":"0","update_time_ms":1722415590471,"create_time_ms":1722406069992}]},"request_id":"1722415590230464500"}`)

func TestWebsocketOrderCancelAllByPair(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("LTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderCancelAllByPair(context.Background(), pair, 0, "", 0)
	require.ErrorIs(t, err, errEdgeCaseIssue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	got, err := g.WebsocketOrderCancelAllByPair(context.Background(), currency.EMPTYPAIR, order.Buy, "", asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var amendOrderError = []byte(`{"header":{"response_time":"1722420643127","status":"404","channel":"spot.order_amend","event":"api","client_id":"58.169.146.133-0xc1e615e6e0","conn_id":"71eb27ad8803a9bd","trace_id":"4d80b11b184b49bd540abd039f42a84d"},"data":{"errs":{"label":"ORDER_NOT_FOUND","message":"label: ORDER_NOT_FOUND, message: Order not found"}},"request_id":"1722420642896203600"}`)
var ammendOrderSuccess = []byte(`"header":{"response_time":"1722420772699","status":"200","channel":"spot.order_amend","event":"api","client_id":"58.169.146.133-0xc08c7c2f20"},"data":{"result":{"left":"0.0004","update_time":"1722420772","amount":"0.0004","create_time":"1722420733","price":"20000","finish_as":"open","time_in_force":"gtc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1722420733733908700","status":"open","iceberg":"0","filled_total":"0","id":"645029162673","fill_price":"0","update_time_ms":1722420772698,"create_time_ms":1722420733966}},"request_id":"1722420772476042600"}`)

func TestWebsocketOrderAmend(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketOrderAmend(context.Background(), nil, 0)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketAmendOrder{}
	_, err = g.WebsocketOrderAmend(context.Background(), amend, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = g.WebsocketOrderAmend(context.Background(), amend, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderAmend(context.Background(), amend, 0)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Amount = "0.0004"

	_, err = g.WebsocketOrderAmend(context.Background(), amend, 0)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	amend.OrderID = "645029162673"
	got, err := g.WebsocketOrderAmend(context.Background(), amend, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var getOrderStatusError = []byte(`{"header":{"response_time":"1722417357718","status":"404","channel":"spot.order_status","event":"api","client_id":"58.169.146.133-0xc0e6013600","conn_id":"8ae56147f8a55b08","trace_id":"127ac043f3a762ae88b746122aba5e3b"},"data":{"errs":{"label":"ORDER_NOT_FOUND","message":"label: ORDER_NOT_FOUND, message: Order with ID 644999648436 not found"}},"request_id":"1722417357478800700"}`)
var getOrderStatusSuccess = []byte(`{"header":{"response_time":"1722417915985","status":"200","channel":"spot.order_status","event":"api","client_id":"58.169.146.133-0xc06e7ff1e0"},"data":{"result":{"left":"0.0003","update_time":"1722417700","amount":"0.0003","create_time":"1722416858","price":"20000","finish_as":"cancelled","time_in_force":"gtc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1722416858697102100","status":"cancelled","iceberg":"0","filled_total":"0","id":"644999650452","fill_price":"0","update_time_ms":1722417700653,"create_time_ms":1722416858942}},"request_id":"1722417915744467800"}`)

func TestWebsocketGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketGetOrderStatus(context.Background(), "", currency.EMPTYPAIR, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = g.WebsocketGetOrderStatus(context.Background(), "1337", currency.EMPTYPAIR, "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("LTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketGetOrderStatus(context.Background(), "1337", pair, "", 0)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketGetOrderStatus(context.Background(), "644999650452", pair, "", asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
