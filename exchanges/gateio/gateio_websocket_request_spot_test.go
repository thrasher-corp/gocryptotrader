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

	demonstrationConn, err := g.Websocket.GetOutboundConnection(asset.Spot)
	require.NoError(t, err)

	got, err := g.WebsocketLogin(context.Background(), demonstrationConn, "spot.login")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderPlaceSpot(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderPlaceSpot(context.Background(), nil)
	require.ErrorIs(t, err, errBatchSliceEmpty)
	_, err = g.WebsocketOrderPlaceSpot(context.Background(), make([]CreateOrderRequestData, 1))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	out := CreateOrderRequestData{CurrencyPair: pair}
	_, err = g.WebsocketOrderPlaceSpot(context.Background(), []CreateOrderRequestData{out})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Buy.String())
	_, err = g.WebsocketOrderPlaceSpot(context.Background(), []CreateOrderRequestData{out})
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = 0.0003
	out.Type = "limit"
	_, err = g.WebsocketOrderPlaceSpot(context.Background(), []CreateOrderRequestData{out})
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = 20000

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// test single order
	got, err := g.WebsocketOrderPlaceSpot(context.Background(), []CreateOrderRequestData{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketOrderPlaceSpot(context.Background(), []CreateOrderRequestData{out, out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelSpot(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderCancelSpot(context.Background(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = g.WebsocketOrderCancelSpot(context.Background(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	btcusdt, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketOrderCancelSpot(context.Background(), "644913098758", btcusdt, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelAllByIDsSpot(t *testing.T) {
	t.Parallel()
	out := WebsocketOrderCancelRequest{}
	_, err := g.WebsocketOrderCancelAllByIDsSpot(context.Background(), []WebsocketOrderCancelRequest{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	out.OrderID = "1337"
	_, err = g.WebsocketOrderCancelAllByIDsSpot(context.Background(), []WebsocketOrderCancelRequest{out})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out.OrderID = "644913101755"
	got, err := g.WebsocketOrderCancelAllByIDsSpot(context.Background(), []WebsocketOrderCancelRequest{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelAllByPairSpot(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("LTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderCancelAllByPairSpot(context.Background(), pair, 0, "")
	require.ErrorIs(t, err, errEdgeCaseIssue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketOrderCancelAllByPairSpot(context.Background(), currency.EMPTYPAIR, order.Buy, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderAmendSpot(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketOrderAmendSpot(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketAmendOrder{}
	_, err = g.WebsocketOrderAmendSpot(context.Background(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = g.WebsocketOrderAmendSpot(context.Background(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderAmendSpot(context.Background(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Amount = "0.0004"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	amend.OrderID = "645029162673"
	got, err := g.WebsocketOrderAmendSpot(context.Background(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketGetOrderStatusSpot(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketGetOrderStatusSpot(context.Background(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = g.WebsocketGetOrderStatusSpot(context.Background(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketGetOrderStatusSpot(context.Background(), "644999650452", pair, "")
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

assetLoader:
	for _, a := range cpy.GetAssetTypes(true) {
		var avail currency.Pairs
		switch a {
		case asset.Spot:
			avail, err = cpy.GetAvailablePairs(a)
			require.NoError(t, err)
			if len(avail) > 1 { // reduce pairs to 1 to speed up tests
				avail = avail[:1]
			}
		case asset.Futures:
			avail, err = cpy.GetAvailablePairs(a)
			require.NoError(t, err)
			usdtPairs, err := avail.GetPairsByQuote(currency.USDT) // Get USDT margin pairs
			require.NoError(t, err)
			btcPairs, err := avail.GetPairsByQuote(currency.USD) // Get BTC margin pairs
			require.NoError(t, err)
			// below makes sure there is both a USDT and BTC pair available
			// so that allows two connections to be made.
			avail[0] = usdtPairs[0]
			avail[1] = btcPairs[0]
			avail = avail[:2]
		default:
			require.NoError(t, cpy.CurrencyPairs.SetAssetEnabled(a, false))
			continue assetLoader
		}
		require.NoError(t, cpy.SetPairs(avail, a, true))
	}
	require.NoError(t, cpy.Websocket.Connect())
	return cpy
}
