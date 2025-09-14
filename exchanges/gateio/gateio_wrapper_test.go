package gateio

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()

	testexch.UpdatePairsOnce(t, e)

	availSpot, err := e.GetAvailablePairs(asset.Spot)
	require.NoError(t, err)

	availMargin, err := e.GetAvailablePairs(asset.Margin)
	require.NoError(t, err)

	marginPairNotInSpot, err := availMargin.Remove(availSpot...).GetRandomPair()
	require.NoError(t, err)

	availOptions, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err)

	optionsPair, err := availOptions.GetRandomPair()
	require.NoError(t, err)

	availDelivery, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err)

	deliveryPair, err := availDelivery.GetRandomPair()
	require.NoError(t, err)

	for _, tc := range []struct {
		pair currency.Pair
		a    asset.Item
		err  error
	}{
		{pair: currency.EMPTYPAIR, a: asset.Spot, err: currency.ErrCurrencyPairEmpty},
		{pair: marginPairNotInSpot, a: asset.Margin, err: errNoOrderbookDataAvailable},
		{pair: marginPairNotInSpot, a: asset.Binary, err: asset.ErrNotSupported},
		{pair: currency.NewBTCUSDT(), a: asset.Spot},
		{pair: currency.NewBTCUSDT(), a: asset.USDTMarginedFutures},
		{pair: deliveryPair, a: asset.DeliveryFutures},
		{pair: optionsPair, a: asset.Options},
	} {
		t.Run(fmt.Sprintf("%s-%s: expected err:%v", tc.pair, tc.a, tc.err), func(t *testing.T) {
			t.Parallel()
			got, err := e.fetchOrderbook(t.Context(), tc.pair, tc.a, 1)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, e.Name, got.Exchange)
			assert.True(t, tc.pair.Equal(got.Pair))
			assert.Equal(t, tc.a, got.Asset)
			// options books are consistently empty but should not exceed limit 1
			assert.LessOrEqual(t, len(got.Asks), 1, "Asks count should not exceed limit, but may be empty especially for options")
			assert.LessOrEqual(t, len(got.Bids), 1, "Bids count should not exceed limit, but may be empty especially for options")
			assert.NotZero(t, got.LastUpdated)
			assert.NotZero(t, got.LastUpdateID)
			assert.NotZero(t, got.LastPushed)
			assert.LessOrEqual(t, got.LastUpdated, got.LastPushed)
		})
	}
}
