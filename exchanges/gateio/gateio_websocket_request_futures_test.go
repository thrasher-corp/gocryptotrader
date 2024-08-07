package gateio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketOrderPlaceFutures(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderPlaceFutures(context.Background(), nil)
	require.ErrorIs(t, err, errBatchSliceEmpty)
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), make([]OrderCreateParams, 1))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out := OrderCreateParams{}
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.ErrorIs(t, err, errInvalidPrice)

	out.Price = "40000"
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.ErrorIs(t, err, errInvalidAmount)

	out.Size = 1 // 1 lovely long contract
	out.AutoSize = "silly_billies"
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.ErrorIs(t, err, errInvalidAutoSize)

	out.AutoSize = "close_long"
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.ErrorIs(t, err, errInvalidAmount)

	out.AutoSize = ""
	outBad := out
	outBad.Contract, err = currency.NewPairFromString("BTC_USD")
	require.NoError(t, err)

	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out, outBad})
	require.ErrorIs(t, err, errSettlementCurrencyConflict)

	outBad.Contract, out.Contract = out.Contract, outBad.Contract // swapsies
	_, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out, outBad})
	require.ErrorIs(t, err, errSettlementCurrencyConflict)

	outBad.Contract, out.Contract = out.Contract, outBad.Contract // swapsies back

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// test single order
	got, err := g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketOrderPlaceFutures(context.Background(), []OrderCreateParams{out, out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelFutures(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketOrderCancelFutures(context.Background(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = g.WebsocketOrderCancelFutures(context.Background(), "42069", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketOrderCancelFutures(context.Background(), "513160761072", pair)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderCancelAllOpenFuturesOrdersMatched(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketOrderCancelAllOpenFuturesOrdersMatched(context.Background(), currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	_, err = g.WebsocketOrderCancelAllOpenFuturesOrdersMatched(context.Background(), pair, "bruh")
	require.ErrorIs(t, err, errInvalidSide)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketOrderCancelAllOpenFuturesOrdersMatched(context.Background(), pair, "bid")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderAmendFutures(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketOrderAmendFutures(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketFuturesAmendOrder{}
	_, err = g.WebsocketOrderAmendFutures(context.Background(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = g.WebsocketOrderAmendFutures(context.Background(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderAmendFutures(context.Background(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Size = 2

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	amend.OrderID = "513170215869"
	got, err := g.WebsocketOrderAmendFutures(context.Background(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketOrderListFutures(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketOrderListFutures(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	list := &WebsocketFutureOrdersList{}
	_, err = g.WebsocketOrderListFutures(context.Background(), list)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	list.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketOrderListFutures(context.Background(), list)
	require.ErrorIs(t, err, errStatusNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	list.Status = "open"
	got, err := g.WebsocketOrderListFutures(context.Background(), list)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketGetOrderStatusFutures(t *testing.T) {
	t.Parallel()

	_, err := g.WebsocketGetOrderStatusFutures(context.Background(), currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketGetOrderStatusFutures(context.Background(), pair, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	testexch.UpdatePairsOnce(t, g)
	g := getWebsocketInstance(t, g) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketGetOrderStatusFutures(context.Background(), pair, "513170215869")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
