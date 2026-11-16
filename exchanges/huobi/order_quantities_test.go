package huobi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	t.Run("quantity units", func(t *testing.T) {
		t.Parallel()
		testSpotOrderQuantities(t, false)
	})
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.GetOrderInfo(t.Context(), "123", currency.NewBTCUSDT(), asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
	})
}

func testSpotOrderQuantities(t *testing.T, active bool) {
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
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, endpoint := row, huobiGetOrder
				if active {
					data, endpoint = "["+row+"]", huobiGetOpenOrders
				}
				assert.Equal(t, "/v1"+endpoint, r.URL.Path, "request should use the correct endpoint")
				_, err := fmt.Fprintf(w, `{"status":"ok","data":%s}`, data)
				assert.NoError(t, err, "response should be written")
			}))
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "7"})
			pair := currency.NewBTCUSDT()
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Spot, pair)
			require.NoError(t, ex.Requester.DisableRateLimiter(), "DisableRateLimiter must not error")
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
			var got *order.Detail
			var err error
			if active {
				orders, fetchErr := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Pairs: currency.Pairs{pair}, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
				require.NoError(t, fetchErr, "GetActiveOrders must not error")
				require.Len(t, orders, 1, "GetActiveOrders must return one order")
				got = &orders[0]
			} else {
				got, err = ex.GetOrderInfo(t.Context(), "123", pair, asset.Spot)
				require.NoError(t, err, "GetOrderInfo must not error")
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
			if !active {
				expected.Pair = pair.Lower()
			}
			assert.Equal(t, expected, got, "order mapping should preserve identity and separate base quantities from quote budgets")
		})
	}
}
