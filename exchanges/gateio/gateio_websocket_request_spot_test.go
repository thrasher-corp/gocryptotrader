package gateio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	err := e.websocketLogin(t.Context(), nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	c, err := g.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err)

	err = g.websocketLogin(t.Context(), c, "")
	require.ErrorIs(t, err, errChannelEmpty)

	err = g.websocketLogin(t.Context(), c, "spot.login")
	require.NoError(t, err)
}

func TestWebsocketSpotSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotSubmitOrder(t.Context(), &CreateOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := &CreateOrderRequest{CurrencyPair: currency.NewPair(currency.NewCode("GT"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})}
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Sell.String())
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = 1
	out.Type = "limit"
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = 100

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	got, err := g.WebsocketSpotSubmitOrder(t.Context(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotSubmitOrders(t.Context())
	require.ErrorIs(t, err, errOrdersEmpty)
	out := &CreateOrderRequest{}
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out.CurrencyPair = currency.NewBTCUSDT()
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	out.Side = strings.ToLower(order.Buy.String())
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Amount = 0.0003
	out.Type = "limit"
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = 20000

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	// test single order
	got, err := g.WebsocketSpotSubmitOrders(t.Context(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketSpotSubmitOrders(t.Context(), out, out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotCancelOrder(t.Context(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.WebsocketSpotCancelOrder(t.Context(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	got, err := g.WebsocketSpotCancelOrder(t.Context(), "644913098758", BTCUSDT, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelAllOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{})
	require.ErrorIs(t, err, errNoOrdersToCancel)
	out := WebsocketOrderBatchRequest{}
	_, err = e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	out.OrderID = "1337"
	_, err = e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Pair = BTCUSDT

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	out.OrderID = "644913101755"
	got, err := g.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotCancelAllOrdersByPair(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotCancelAllOrdersByPair(t.Context(), currency.NewPairWithDelimiter("LTC", "USDT", "_"), 0, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	got, err := g.WebsocketSpotCancelAllOrdersByPair(t.Context(), currency.EMPTYPAIR, order.Buy, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketAmendOrder{}
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair = BTCUSDT

	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Amount = "0.0004"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Spot)

	amend.OrderID = "645029162673"
	got, err := g.WebsocketSpotAmendOrder(t.Context(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSpotGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotGetOrderStatus(t.Context(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.WebsocketSpotGetOrderStatus(t.Context(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, e)
	g := newExchangeWithWebsocket(t, asset.Spot)

	got, err := g.WebsocketSpotGetOrderStatus(t.Context(), "644999650452", BTCUSDT, "")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

// getWebsocketInstance returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func newExchangeWithWebsocket(t *testing.T, a asset.Item) *Exchange {
	t.Helper()
	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}
	e := new(Exchange) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, e)
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	switch a {
	case asset.Spot:
		avail, err := e.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 { // reduce pairs to 1 to speed up tests
			avail = avail[:1]
		}
		require.NoError(t, e.SetPairs(avail, a, true))
	case asset.Futures:
		avail, err := e.GetAvailablePairs(a)
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
		require.NoError(t, e.SetPairs(avail, a, true))
	default:
		require.NoError(t, e.CurrencyPairs.SetAssetEnabled(a, false))
	}

	// Disable all other asset types to ensure only the specified asset type is used for websocket tests.
	for _, enabled := range e.GetAssetTypes(true) {
		if enabled != a {
			require.NoError(t, e.CurrencyPairs.SetAssetEnabled(enabled, false))
		}
	}

	require.NoError(t, e.Websocket.Connect())
	return e
}
