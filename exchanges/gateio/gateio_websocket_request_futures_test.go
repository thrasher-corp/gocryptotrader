package gateio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestWebsocketFuturesSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesSubmitOrder(context.Background(), &ContractOrderCreateParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := &ContractOrderCreateParams{Contract: currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})}
	_, err = g.WebsocketFuturesSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidPrice)
	out.Price = "40000"
	_, err = g.WebsocketFuturesSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAmount)
	out.Size = 1 // 1 lovely long contract
	out.AutoSize = "silly_billies"
	_, err = g.WebsocketFuturesSubmitOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	out.AutoSize = ""

	got, err := g.WebsocketFuturesSubmitOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesSubmitOrders(context.Background())
	require.ErrorIs(t, err, errOrdersEmpty)

	out := &ContractOrderCreateParams{}
	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out)
	require.ErrorIs(t, err, errInvalidPrice)

	out.Price = "40000"
	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAmount)

	out.Size = 1 // 1 lovely long contract
	out.AutoSize = "silly_billies"
	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	out.AutoSize = "close_long"
	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out)
	require.ErrorIs(t, err, errInvalidAmount)

	out.AutoSize = ""
	outBad := *out
	outBad.Contract, err = currency.NewPairFromString("BTC_USD")
	require.NoError(t, err)

	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out, &outBad)
	require.ErrorIs(t, err, errSettlementCurrencyConflict)

	outBad.Contract, out.Contract = out.Contract, outBad.Contract // swapsies
	_, err = g.WebsocketFuturesSubmitOrders(context.Background(), out, &outBad)
	require.ErrorIs(t, err, errSettlementCurrencyConflict)

	outBad.Contract, out.Contract = out.Contract, outBad.Contract // swapsies back

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// test single order
	got, err := g.WebsocketFuturesSubmitOrders(request.WithVerbose(context.Background()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// test batch orders
	got, err = g.WebsocketFuturesSubmitOrders(context.Background(), out, out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesCancelOrder(context.Background(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = g.WebsocketFuturesCancelOrder(context.Background(), "42069", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	got, err := g.WebsocketFuturesCancelOrder(context.Background(), "513160761072", pair)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesCancelAllOpenFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesCancelAllOpenFuturesOrders(context.Background(), currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	_, err = g.WebsocketFuturesCancelAllOpenFuturesOrders(context.Background(), pair, "bruh")
	require.ErrorIs(t, err, errInvalidSide)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketFuturesCancelAllOpenFuturesOrders(context.Background(), pair, "bid")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesAmendOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketFuturesAmendOrder{}
	_, err = g.WebsocketFuturesAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = g.WebsocketFuturesAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketFuturesAmendOrder(context.Background(), amend)
	require.ErrorIs(t, err, errInvalidAmount)

	amend.Size = 2

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	amend.OrderID = "513170215869"
	got, err := g.WebsocketFuturesAmendOrder(context.Background(), amend)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesOrderList(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesOrderList(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	list := &WebsocketFutureOrdersList{}
	_, err = g.WebsocketFuturesOrderList(context.Background(), list)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	list.Contract, err = currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketFuturesOrderList(context.Background(), list)
	require.ErrorIs(t, err, errStatusNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	list.Status = statusOpen
	got, err := g.WebsocketFuturesOrderList(context.Background(), list)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketFuturesGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketFuturesGetOrderStatus(context.Background(), currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = g.WebsocketFuturesGetOrderStatus(context.Background(), pair, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	g := newExchangeWithWebsocket(t, asset.Futures) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	got, err := g.WebsocketFuturesGetOrderStatus(context.Background(), pair, "513170215869")
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
