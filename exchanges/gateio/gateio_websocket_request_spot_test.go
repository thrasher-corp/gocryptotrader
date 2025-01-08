package gateio

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	err := g.websocketLogin(context.Background(), nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = g.websocketLogin(context.Background(), &stream.WebsocketConnection{}, "")
	require.ErrorIs(t, err, errChannelEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	demonstrationConn, err := g.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err)

	err = g.websocketLogin(context.Background(), demonstrationConn, "spot.login")
	require.NoError(t, err)
}

func TestWebsocketSpotSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketSpotSubmitOrder(context.Background(), &WebsocketOrder{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := &WebsocketOrder{CurrencyPair: "BTC_USDT"}
	_, err = g.WebsocketSpotSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Buy.String())
	_, err = g.WebsocketSpotSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = "0.0003"
	out.Type = "limit"
	_, err = g.WebsocketSpotSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "20000"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketSpotSubmitOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketSpotSubmitOrders(context.Background(), nil)
	require.ErrorIs(t, err, errOrdersEmpty)
	_, err = g.WebsocketSpotSubmitOrders(context.Background(), make([]WebsocketOrder, 1))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := WebsocketOrder{CurrencyPair: "BTC_USDT"}
	_, err = g.WebsocketSpotSubmitOrders(context.Background(), []WebsocketOrder{out})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Buy.String())
	_, err = g.WebsocketSpotSubmitOrders(context.Background(), []WebsocketOrder{out})
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = "0.0003"
	out.Type = "limit"
	_, err = g.WebsocketSpotSubmitOrders(context.Background(), []WebsocketOrder{out})
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "20000"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// test single order
	got, err := g.WebsocketSpotSubmitOrders(context.Background(), []WebsocketOrder{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketSpotSubmitOrders(context.Background(), []WebsocketOrder{out, out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketSpotCancelOrder(context.Background(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = g.WebsocketSpotCancelOrder(context.Background(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	btcusdt, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketSpotCancelOrder(context.Background(), "644913098758", btcusdt, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelAllOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketSpotCancelAllOrdersByIDs(context.Background(), []WebsocketOrderBatchRequest{})
	require.ErrorIs(t, err, errNoOrdersToCancel)
	out := WebsocketOrderBatchRequest{}
	_, err = g.WebsocketSpotCancelAllOrdersByIDs(context.Background(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	out.OrderID = "1337"
	_, err = g.WebsocketSpotCancelAllOrdersByIDs(context.Background(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out.OrderID = "644913101755"
	got, err := g.WebsocketSpotCancelAllOrdersByIDs(context.Background(), []WebsocketOrderBatchRequest{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelAllOrdersByPair(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("LTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketSpotCancelAllOrdersByPair(context.Background(), pair, 0, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketSpotCancelAllOrdersByPair(context.Background(), currency.EMPTYPAIR, order.Buy, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketSpotAmendOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketAmendOrder{}
	_, err = g.WebsocketSpotAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = g.WebsocketSpotAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketSpotAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Amount = "0.0004"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	amend.OrderID = "645029162673"
	got, err := g.WebsocketSpotAmendOrder(context.Background(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketSpotGetOrderStatus(context.Background(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = g.WebsocketSpotGetOrderStatus(context.Background(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketSpotGetOrderStatus(context.Background(), "644999650452", pair, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

// getWebsocketInstance returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func getWebsocketInstance(t *testing.T, g *Gateio) *Gateio {
	t.Helper()

	cpy := new(Gateio)
	cpy.SetDefaults()
	gConf, err := config.GetConfig().GetExchangeConfig("GateIO")
	require.NoError(t, err)
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret

	require.NoError(t, cpy.Setup(gConf), "Test instance Setup must not error")
	cpy.CurrencyPairs.Load(&g.CurrencyPairs)

	for _, a := range cpy.GetAssetTypes(true) {
		if a != asset.Spot {
			require.NoError(t, cpy.CurrencyPairs.SetAssetEnabled(a, false))
			continue
		}
		avail, err := cpy.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, cpy.SetPairs(avail, a, true))
	}
	require.NoError(t, cpy.Websocket.Connect())
	return cpy
}
