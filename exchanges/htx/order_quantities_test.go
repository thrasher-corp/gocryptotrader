package htx

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSpotOrderQuantities(t *testing.T) {
	t.Parallel()
	for _, endpoint := range []string{"detail", "active", "history"} {
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()
			testSpotOrderQuantities(t, endpoint)
		})
	}
}

func testSpotOrderQuantities(t *testing.T, endpoint string) {
	t.Helper()
	for _, tc := range []struct {
		name, typ, state                             string
		amount, filled, cash, base, quote, remaining float64
		side                                         order.Side
		orderType                                    order.Type
		status                                       order.Status
	}{
		{"unfilled quote buy", "buy-market", "submitted", 100, 0, 0, 0, 100, 0, order.Buy, order.Market, order.New},
		{"partial quote buy", "buy-market", "partial-filled", 100, 0.001, 60, 0, 100, 0, order.Buy, order.Market, order.PartiallyFilled},
		{"filled quote buy", "buy-market", "filled", 100, 0.002, 100, 0, 100, 0, order.Buy, order.Market, order.Filled},
		{"base market sell", "sell-market", "partial-filled", 2, 1.5, 90, 2, 0, 0.5, order.Sell, order.Market, order.PartiallyFilled},
		{"base limit buy", "buy-limit", "partial-filled", 2, 1.5, 90, 2, 0, 0.5, order.Buy, order.Limit, order.PartiallyFilled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := fmt.Sprintf(`{"id":123,"symbol":"btcusdt","account-id":7,"amount":"%v","price":"61","created-at":1700000000000,"type":%q,"filled-amount":"%v","filled-cash-amount":"%v","filled-fees":"0.3","state":%q}`, tc.amount, tc.typ, tc.filled, tc.cash, tc.state)
			path, data := "/v1/order/orders/123", row
			switch endpoint {
			case "active":
				path, data = "/v1/order/openOrders", "["+row+"]"
			case "history":
				path, data = "/v1/order/orders", "["+row+"]"
			}
			ex := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, path, fmt.Sprintf(`{"status":"ok","data":%s}`, data), nil)
			pair := btcusdtPair
			req := &order.MultiOrderRequest{Pairs: currency.Pairs{pair}, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
			var got *order.Detail
			if endpoint == "detail" {
				detail, err := ex.GetOrderInfo(t.Context(), "123", pair, asset.Spot)
				require.NoError(t, err, "GetOrderInfo must not error")
				got = detail
			} else {
				fetch := ex.GetActiveOrders
				if endpoint == "history" {
					fetch = ex.GetOrderHistory
				}
				orders, err := fetch(t.Context(), req)
				require.NoError(t, err, "order lookup must not error")
				require.Len(t, orders, 1, "order lookup must return one order")
				got = &orders[0]
			}
			average := 0.0
			if tc.filled > 0 {
				average = tc.cash / tc.filled
			}
			expected := &order.Detail{
				Exchange: ex.Name, OrderID: "123", AccountID: "7", Pair: pair, AssetType: asset.Spot,
				Type: tc.orderType, Side: tc.side, Status: tc.status, Price: 61, Amount: tc.base, QuoteAmount: tc.quote,
				ExecutedAmount: tc.filled, ExecutedQuoteAmount: tc.cash, RemainingAmount: tc.remaining, AverageExecutedPrice: average,
				Fee: 0.3, Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000000000),
			}

			if endpoint == "detail" {
				expected.Pair = currency.NewBTCUSDT().Lower()
			}
			assert.Equal(t, expected, got, "order mapping should preserve identity and separate base quantities from quote budgets")
		})
	}
}
