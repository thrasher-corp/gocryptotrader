package coinut

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketOrderFilledCumulativeExecution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, quantity, open, priorOpen, fill string
		executed, remaining, latest           float64
		status                                order.Status
	}{
		{"partial", "5.5", "1", "2.5", "1.5", 4.5, 1, 1.5, order.PartiallyFilled},
		{"final", "5.5", "0", "1.5", "1.5", 5.5, 0, 1.5, order.Filled},
		{"decimal", "0.03", "0.01", "0.02", "0.01", 0.02, 0.01, 0.01, order.PartiallyFilled},
		{"large exact", "200000000.00000003", "200000000.00000001", "200000000.00000002", "0.00000001", 0.00000002, 200000000.00000001, 0.00000001, order.PartiallyFilled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange setup must not error")
			ex.instrumentMap.Seed("BTCUSD", 123)
			pair := currency.NewBTCUSD()
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, false), "available pairs setup must not error")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, true), "enabled pairs setup must not error")
			ex.SkipAuthCheck = true
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := fmt.Fprintf(w, `{"status":["OK"],"orders":[{"order_id":42,"inst_id":123,"qty":%q,"open_qty":%q,"side":"BUY","price":"60000","timestamp":1735720637000000}]}`, tc.quantity, tc.priorOpen)
				assert.NoError(t, err, "mock earlier active order should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock REST client must be installed")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "mock REST endpoint must be installed")
			active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Pairs: currency.Pairs{pair}, AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide})
			require.NoError(t, err, "earlier cumulative execution must load through REST")
			require.Len(t, active, 1, "REST must return the earlier active order")
			stored := active[0].Copy()
			frame := fmt.Appendf(nil, `{"commission":{"amount":"60","currency":"USD"},"fill_price":"60000","fill_qty":%q,"nonce":1,"order":{"client_ord_id":1,"inst_id":123,"open_qty":%q,"order_id":42,"price":"60000","qty":%q,"side":"BUY","timestamp":1735720637000000},"reply":"order_filled","status":["OK"],"timestamp":1735720640000000,"trans_id":2002}`, tc.fill, tc.open, tc.quantity)
			require.NoError(t, ex.wsHandleData(t.Context(), frame), "order_filled push must decode")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "order_filled push must emit one order")
			got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
			require.True(t, ok, "order_filled push must emit an order detail")
			assert.Equal(t, tc.status, got.Status, "status should follow the nested order's open quantity")
			assert.Equal(t, tc.executed, got.ExecutedAmount, "execution should be the order's cumulative fill, not the latest fill")
			assert.Equal(t, tc.remaining, got.RemainingAmount, "remaining quantity should come from the order")
			require.Len(t, got.Trades, 1, "the latest fill must be its own trade")
			assert.Equal(t, tc.latest, got.Trades[0].Amount, "the trade should keep the latest fill's quantity")
			require.NoError(t, stored.UpdateOrderFromDetail(got), "cumulative push must merge into the REST snapshot")
			assert.Equal(t, tc.status, stored.Status, "the order manager should retain the push's lifecycle state")
			assert.Equal(t, tc.executed, stored.ExecutedAmount, "the order manager should retain cumulative execution")
		})
	}
}
