package okx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsPlaceOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out := &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDT",
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        -1,
		Currency:     "USDT",
	}

	got, err := ok.WsPlaceOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsPlaceMultipleOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WsPlaceMultipleOrder(context.Background(), []PlaceOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out := PlaceOrderRequestParam{
		InstrumentID: "BTC-USDT",
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        -1, // Intentional fail
		Currency:     "USDT",
	}

	got, err := ok.WsPlaceMultipleOrder(context.Background(), []PlaceOrderRequestParam{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsCancelOrder(context.Background(), CancelOrderRequestParam{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WsCancelOrder(context.Background(), CancelOrderRequestParam{InstrumentID: "BTC-USDT"})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	c := CancelOrderRequestParam{InstrumentID: "BTC-USDT", OrderID: "1680136326338387968"}
	got, err := ok.WsCancelOrder(context.Background(), c)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsCancelMultipleOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{{InstrumentID: "BTC-USDT"}})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	c := CancelOrderRequestParam{InstrumentID: "BTC-USDT", OrderID: "1680136326338387968"}
	got, err := ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{c})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsAmendOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := &AmendOrderRequestParams{}
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = "BTC-USDT"
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	out.OrderID = "1680136326338387968"
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out.NewPrice = 20
	got, err := ok.WsAmendOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsAmendMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := ok.WsAmendMultipleOrders(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := AmendOrderRequestParams{}
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = "BTC-USDT"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	out.OrderID = "1680136326338387968"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	out.NewPrice = 20

	got, err := ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

// getWebsocketInstance returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func getWebsocketInstance(t *testing.T, g *Okx) *Okx {
	t.Helper()

	cpy := new(Okx)
	cpy.SetDefaults()
	gConf, err := config.GetConfig().GetExchangeConfig("Okx")
	require.NoError(t, err)
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret
	gConf.API.Credentials.ClientID = passphrase

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
