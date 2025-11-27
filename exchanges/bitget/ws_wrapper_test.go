package bitget

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    "test",
		Pair:        testPair,
		AssetType:   asset.Binary,
		Side:        order.Long,
		Type:        order.Chase,
		TimeInForce: order.StopOrReduce,
		Amount:      0.001,
	})
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    "test",
		Pair:        testPair,
		AssetType:   asset.Binary,
		Side:        order.Long,
		Type:        order.Chase,
		TimeInForce: order.GoodTillCancel,
		Amount:      0.001,
	})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    "test",
		Pair:        testPair,
		AssetType:   asset.Binary,
		Side:        order.Long,
		Type:        order.Market,
		TimeInForce: order.GoodTillCancel,
		Amount:      0.001,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    "test",
		Pair:        testPair,
		AssetType:   asset.Spot,
		Side:        order.Long,
		Type:        order.Market,
		TimeInForce: order.GoodTillCancel,
		Amount:      0.001,
	})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    "test",
		Pair:        testPair,
		AssetType:   asset.USDTMarginedFutures,
		Side:        order.Long,
		Type:        order.Limit,
		TimeInForce: order.GoodTillCancel,
		Amount:      0.0001,
		Price:       50000,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:           "test",
		Pair:               testPair,
		AssetType:          asset.USDTMarginedFutures,
		Side:               order.Long,
		Type:               order.Limit,
		TimeInForce:        order.GoodTillCancel,
		Amount:             0.0001,
		Price:              50000,
		SettlementCurrency: currency.USDT,
	})
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	e := newExchangeWithWebsocket(t, asset.USDTMarginedFutures)
	got, err := e.WebsocketSubmitOrder(request.WithVerbose(t.Context()), &order.Submit{
		Exchange:           "test",
		Pair:               testPair,
		AssetType:          asset.USDTMarginedFutures,
		Side:               order.Long,
		Type:               order.Market,
		TimeInForce:        order.GoodTillCancel,
		Amount:             0.0001,
		SettlementCurrency: currency.USDT,
		MarginType:         margin.Isolated,
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	err := e.WebsocketCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spread})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	e := newExchangeWithWebsocket(t, asset.USDTMarginedFutures)
	err = e.WebsocketCancelOrder(request.WithVerbose(t.Context()), &order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      testPair,
		OrderID:   "1377803648385515520",
	})
	require.NoError(t, err)
}
