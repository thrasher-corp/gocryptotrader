package htx

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestLegacyFuturesWSFillLifecycle(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures} {
		for _, feeAsset := range []string{"", "BTC"} {
			t.Run(fmt.Sprintf("%s/fillFeeAsset=%s", a, feeAsset), func(t *testing.T) {
				t.Parallel()
				h := new(Exchange)
				require.NoError(t, testexch.Setup(h), "isolated exchange setup must succeed")
				sub := &subscription.Subscription{Asset: a, Channel: subscription.MyOrdersChannel, Authenticated: true}
				// Distinct global IDs can share the venue's non-unique match trade_id.
				raw := []byte(fmt.Sprintf(`{"op":"notify","topic":"orders.BTC-USD","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6,"order_id":1,"order_id_str":"order-1","volume":2,"price":110,"trade_volume":2,"trade_turnover":200,"trade_avg_price":100,"fee":-0.003,"fee_asset":"BTC","trade":[{"id":"9007199254740993","trade_id":42,"trade_volume":1,"trade_price":90,"trade_fee":-0.001,"trade_turnover":90,"created_at":1603878749908,"role":"maker","fee_asset":%q},{"id":"9007199254740994","trade_id":42,"trade_volume":1,"trade_price":110,"trade_fee":-0.002,"trade_turnover":110,"created_at":1603878749909,"role":"taker","fee_asset":%q}]}`, feeAsset, feeAsset))
				require.NoError(t, h.wsHandleFuturesPrivateMessage(t.Context(), sub, raw), "order notification must pass through its asset handler")
				read := func() *order.Detail {
					t.Helper()
					select {
					case message := <-h.Websocket.DataHandler.C:
						detail, ok := message.Data.(*order.Detail)
						require.True(t, ok, "handler must relay canonical order details")
						return detail
					case <-time.After(time.Second):
						require.FailNow(t, "handler must relay its notification before the deadline")
						return nil
					}
				}
				detail := read()
				assert.Equal(t, a, detail.AssetType, "handler should retain the derivative asset")
				assert.Equal(t, 100.0, detail.AverageExecutedPrice, "wire average should remain independent of the limit price")
				assert.Equal(t, 0.003, detail.Fee, "aggregate charged fee should be positive in GCT")
				assert.Equal(t, currency.BTC, detail.FeeAsset, "aggregate fee currency should be retained")
				require.Len(t, detail.Trades, 2, "both fills must be retained despite their shared match ID")
				for i, expected := range []struct {
					id         string
					price, fee float64
					maker      bool
				}{
					{id: "9007199254740993", price: 90, fee: 0.001, maker: true},
					{id: "9007199254740994", price: 110, fee: 0.002},
				} {
					fill := detail.Trades[i]
					assert.Equal(t, expected.id, fill.TID, "fill identity should preserve its global integer string")
					assert.Equal(t, expected.price, fill.Price, "fill execution price should be retained")
					assert.Equal(t, 1.0, fill.Amount, "fill contract volume should be retained")
					assert.Equal(t, expected.fee, fill.Fee, "fill charged fee should be positive in GCT")
					assert.Equal(t, "BTC", fill.FeeAsset, "fill fee currency should use the order currency when omitted")
					assert.Equal(t, expected.maker, fill.IsMaker, "fill role should distinguish maker and taker")
					assert.Equal(t, time.UnixMilli(1603878749908+int64(i)), fill.Timestamp, "fill timestamp should retain millisecond precision")
				}
				stored := &order.Detail{OrderID: detail.OrderID, Exchange: detail.Exchange, Pair: detail.Pair, AssetType: detail.AssetType}
				require.NoError(t, stored.UpdateOrderFromDetail(detail), "order store must accept the charged fee")
				assert.Equal(t, 0.003, stored.Fee, "order store should record the derivative fee")
				sub.Channel = subscription.MyTradesChannel
				match := []byte(`{"op":"notify","topic":"matchOrders.BTC-USD","contract_code":"BTC-USD","direction":"buy","order_type":1,"status":6,"order_id":1,"order_id_str":"order-1","volume":2,"price":110,"trade_volume":2,"trade":[{"id":"9007199254740993","trade_id":42,"trade_volume":1,"trade_price":90,"created_at":1603878749908,"role":"maker"},{"id":"9007199254740994","trade_id":42,"trade_volume":1,"trade_price":110,"created_at":1603878749909,"role":"taker"}]}`)
				require.NoError(t, h.wsHandleFuturesPrivateMessage(t.Context(), sub, match), "fee-less match notification must pass through its asset handler")
				matched := read()
				assert.Empty(t, matched.Trades, "fee-less matches should not overwrite fee-bearing fills")
				require.NoError(t, stored.UpdateOrderFromDetail(matched), "order management merge must accept the match update")
				assert.Equal(t, detail.Trades, stored.Trades, "subsequent fee-less match should preserve every recorded fill field")
				assert.Equal(t, detail.AverageExecutedPrice, stored.AverageExecutedPrice, "subsequent match should preserve the average execution price")
				assert.Equal(t, detail.Fee, stored.Fee, "subsequent match should preserve the aggregate fee")
				assert.Empty(t, h.Websocket.DataHandler.C, "each handler invocation should relay exactly one order")
			})
		}
	}
}
