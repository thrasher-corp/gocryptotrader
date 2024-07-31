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
	_, err := g.WebsocketLogin(context.Background(), nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = g.WebsocketLogin(context.Background(), &stream.WebsocketConnection{}, "")
	require.ErrorIs(t, err, errChannelEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	route, err := g.GetWebsocketRoute(asset.Spot)
	require.NoError(t, err)

	demonstrationConn, err := g.Websocket.GetOutboundConnection(route)
	require.NoError(t, err)

	got, err := g.WebsocketLogin(context.Background(), demonstrationConn, "spot.login")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

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

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// test single order
	got, err := g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketOrderPlace(context.Background(), []WebsocketOrder{out, out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

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

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketOrderCancel(context.Background(), "644913098758", btcusdt, "", asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

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

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out.OrderID = "644913101755"
	got, err := g.WebsocketOrderCancelAllByIDs(context.Background(), []WebsocketOrderCancelRequest{out}, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelAllByPair(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("LTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderCancelAllByPair(context.Background(), pair, 0, "", 0)
	require.ErrorIs(t, err, errEdgeCaseIssue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketOrderCancelAllByPair(context.Background(), currency.EMPTYPAIR, order.Buy, "", asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

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

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	amend.OrderID = "645029162673"
	got, err := g.WebsocketOrderAmend(context.Background(), amend, asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

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

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketGetOrderStatus(context.Background(), "644999650452", pair, "", asset.Spot)
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
