package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()

	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.EMPTYPAIR, AssetType: 1336})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: 1336})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Options,
		Side:      order.ClosePosition,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Long,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Short,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.AnySide,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			r := &order.Cancel{
				OrderID:   "1",
				AccountID: "1",
				AssetType: a,
				Pair:      currency.EMPTYPAIR,
			}
			_, err := e.CancelAllOrders(t.Context(), r)
			assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

			r.Pair = getPair(t, a)
			_, err = e.CancelAllOrders(t.Context(), r)
			assert.NoError(t, err)
		})
	}
}
