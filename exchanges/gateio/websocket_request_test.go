package gateio

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	out.Amount = "-1"
	out.Type = "limit"
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, 0)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "-1"
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

	out.Account = "spot"
	_, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, asset.Spot)
	require.NoError(t, err)

	// _, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out, out}, asset.Spot)
	// require.NoError(t, err)

	time.Sleep(time.Second * 5)
}
