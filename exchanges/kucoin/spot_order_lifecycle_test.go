package kucoin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestSpotOrderLifecycle(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind, status string
		want         order.Status
	}{
		{"open", "open", order.Open},
		{"match", "open", order.PartiallyFilled},
		{"match", "match", order.PartiallyFilled},
		{"filled", "done", order.Filled},
		{"canceled", "done", order.Cancelled},
		{"update", "done", order.Cancelled},
		{"update", "open", order.Open},
	} {
		t.Run(tc.kind+"/"+tc.status, func(t *testing.T) {
			t.Parallel()
			ex := testInstance(t)
			pair := currency.NewBTCUSDT()
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Spot, pair)
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "margin must be disabled to isolate the spot order")
			payload := fmt.Appendf(nil, `{"symbol":"BTC-USDT","orderId":"1","orderType":"limit","side":"buy","type":%q,"status":%q,"size":"3","filledSize":"2","remainSize":"1","price":"60000","orderTime":1735720637000000000,"ts":1735720638000000000}`, tc.kind, tc.status)
			require.NoError(t, ex.processOrderChangeEvent(t.Context(), payload, "/spotMarket/tradeOrdersV2"), "spot lifecycle event must decode")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "spot lifecycle event must emit one order")
			got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
			require.True(t, ok, "spot event must emit an order detail")
			assert.Equal(t, tc.want, got.Status, "spot status should distinguish cancellation, completion and maker matches")
			assert.Equal(t, 2.0, got.ExecutedAmount, "spot event should preserve cumulative execution")
		})
	}
}
