package gateio

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var (
	BTCUSDT = currency.NewPairWithDelimiter("BTC", "USDT", "_")
	BTCUSD  = currency.NewPairWithDelimiter("BTC", "USD", "_")
)

func TestWebsocketFuturesSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, &ContractOrderCreateParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := &ContractOrderCreateParams{Contract: BTCUSDT}
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "40000"
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Size = 1 // 1 lovely long contract
	out.AutoSize = "silly_billies"
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)
	out.AutoSize = ""

	got, err := g.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errOrdersEmpty)

	out := &ContractOrderCreateParams{}
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Contract = BTCUSDT

	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidPrice)

	out.Price = "40000"
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAmount)

	out.Size = 1 // 1 lovely long contract
	out.AutoSize = "silly_billies"
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	out.AutoSize = "close_long"
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAmount)

	out.AutoSize = ""

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	// test single order
	got, err := g.WebsocketFuturesSubmitOrders(t.Context(), asset.CoinMarginedFutures, out)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketFuturesSubmitOrders(t.Context(), asset.CoinMarginedFutures, out, out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelOrder(t.Context(), "", currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), "42069", currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), "42069", BTCUSDT, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	got, err := g.WebsocketFuturesCancelOrder(t.Context(), "513160761072", BTCUSDT, asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesCancelAllOpenFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), currency.EMPTYPAIR, asset.Empty, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.Empty, "bruh")
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "bruh")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	got, err := g.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "bid")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketFuturesAmendOrder{}
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Contract = BTCUSDT
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	amend.Asset = asset.USDTMarginedFutures
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Size = 2

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	amend.OrderID = "513170215869"
	got, err := g.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	list := &WebsocketFutureOrdersList{}
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	list.Contract = BTCUSDT
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	list.Asset = asset.USDTMarginedFutures
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, errStatusNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	list.Status = statusOpen
	got, err := g.WebsocketFuturesOrderList(t.Context(), list)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesGetOrderStatus(t.Context(), currency.EMPTYPAIR, asset.Empty, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.Empty, "")
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures)

	got, err := g.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "513170215869")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
