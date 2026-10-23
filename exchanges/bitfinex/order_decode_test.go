package bitfinex

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestOrderUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload string
		want          Order
	}{
		{
			name:    "v1_id_only_undelimited",
			payload: `{"id":9007199254740993,"symbol":"btcusd","exchange":"bitfinex","price":"26","avg_execution_price":"25","side":"sell","type":"exchange limit","timestamp":"1792648800.125","is_live":true,"is_cancelled":false,"is_hidden":true,"was_forced":true,"original_amount":"3","remaining_amount":"2","executed_amount":"1"}`,
			want:    Order{ID: 9007199254740993, Symbol: currency.NewBTCUSD().Lower(), Exchange: "bitfinex", Price: 26, AverageExecutionPrice: 25, Side: "sell", Type: "exchange limit", Timestamp: types.Time(time.UnixMilli(1792648800125)), IsLive: true, IsHidden: true, WasForced: true, OriginalAmount: 3, RemainingAmount: 2, ExecutedAmount: 1},
		},
		{
			name: "v1_legacy_order_id_colon_symbol", payload: `{"order_id":2,"symbol":"doge:usd","is_cancelled":true}`,
			want: Order{ID: 2, OrderID: 2, Symbol: currency.NewPairWithDelimiter("doge", "usd", ":"), IsCancelled: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got Order
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &got), "documented orders must decode")
			assert.Equal(t, tc.want, got, "all decoded fields should retain their own values")
		})
	}
}

func TestOrderUnmarshalJSONErrors(t *testing.T) {
	t.Parallel()
	for _, payload := range []string{
		`{"id":"not-an-id"}`, `{"symbol":"x"}`,
		`[]`, `[1]`,
		`[1,null,null,"tBTCUSD",1792648800123,1792648860456,5,3,"EXCHANGE LIMIT",null,null,null,0,"CANCELED",null,null,26,0]`,
	} {
		got := Order{ID: 99}
		assert.Error(t, json.Unmarshal([]byte(payload), &got), "invalid wire fields should fail decoding")
		assert.Equal(t, Order{ID: 99}, got, "failed decoding should not partially replace the order")
	}
}

func TestOrderWrappersWireFormats(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
			require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			ex.SkipAuthCheck = true
			ex.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "mock-secret"})
			pair := currency.NewBTCUSD()
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, a, pair)
			orderType := "limit"
			if a == asset.Spot {
				orderType = "exchange limit"
			}
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				assert.Equal(t, http.MethodPost, r.Method, "order endpoints should use POST")
				var payload string
				switch r.URL.Path {
				case "/v1/orders":
					payload = fmt.Sprintf(`[{"id":2,"symbol":"btcusd","side":"sell","type":%q,"timestamp":"1792648800.123","price":"26","avg_execution_price":"25","original_amount":"3","remaining_amount":"2","executed_amount":"1","is_live":true}]`, orderType)
				case "/v1/order/new":
					payload = fmt.Sprintf(`{"id":2,"symbol":"btcusd","side":"sell","type":%q,"timestamp":"1792648800.123","price":"26","avg_execution_price":"25","original_amount":"3","remaining_amount":"2","executed_amount":"1","is_live":true,"is_cancelled":false}`, orderType)
				case "/v1/order/cancel":
					payload = fmt.Sprintf(`{"id":2,"symbol":"btcusd","side":"sell","type":%q,"timestamp":"1792648800.123","price":"26","avg_execution_price":"25","original_amount":"3","remaining_amount":"2","executed_amount":"1","is_live":false,"is_cancelled":true}`, orderType)
				case "/v2/auth/r/orders/tBTCUSD/hist":
					var body struct {
						ID  []int64 `json:"id"`
						IDs []int64 `json:"ids"`
					}
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "history request should decode")
					assert.Empty(t, body.IDs, "v2 history should not send the v1 ids parameter")
					if len(body.ID) > 0 {
						assert.Equal(t, []int64{2}, body.ID, "ID lookup should send the requested order's documented id filter")
						payload = `[]`
					} else {
						// V2 history support is deferred to PR #2485, including amended amounts and terminal statuses.
						payload = `[[1,null,null,"tBTCUSD",1792648800123,1792648860456,5,3,"EXCHANGE LIMIT",null,null,null,0,"CANCELED",null,null,26,0]]`
					}
				default:
					t.Errorf("order wrappers should not query unexpected path %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				_, err := w.Write([]byte(payload))
				assert.NoError(t, err, "fixture response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "test HTTP client must be installed")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "fixture URL must be installed")
			want := order.Detail{AssetType: a, Exchange: ex.Name, OrderID: "2", Pair: pair.Lower(), Type: order.Limit, Side: order.Sell, Status: order.Active, Date: time.UnixMilli(1792648800123), Price: 26, AverageExecutedPrice: 25, Amount: 3, RemainingAmount: 2, ExecutedAmount: 1}
			req := &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{pair}, Side: order.AnySide, Type: order.AnyType}
			history, err := ex.GetOrderHistory(t.Context(), req)
			assert.Error(t, err, "unsupported v2 history should fail rather than invent fills or statuses")
			assert.Empty(t, history, "unsupported history should not produce misleading order details")
			got, err := ex.GetOrderInfo(t.Context(), "2", pair, a)
			require.NoError(t, err, "v1 ID-only open order lookup must succeed")
			assert.Equal(t, &want, got, "open order lookup should preserve the complete order mapping")
			active, err := ex.GetActiveOrders(t.Context(), req)
			require.NoError(t, err, "v1 active order list must succeed")
			assert.Equal(t, order.FilteredOrders{want}, active, "active order list should share the ID lookup mapping")
			submitted, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange: ex.Name, Pair: pair, AssetType: a, Type: order.Limit, Side: order.Sell, Price: 26, Amount: 3,
			})
			require.NoError(t, err, "REST submit must accept the documented v1 symbol reply")
			assert.Equal(t, "2", submitted.OrderID, "submit should preserve the exchange order ID")
			require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "2", Pair: pair, AssetType: a}), "REST cancel must accept the documented v1 symbol reply")
		})
	}
}
