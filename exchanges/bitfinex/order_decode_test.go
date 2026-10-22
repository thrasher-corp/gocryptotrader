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
		{
			name:    "v2_partial_sell",
			payload: `[9007199254740993,null,null,"tBTCUSD",1792648800123,1792648860456,-2,-3,"EXCHANGE LIMIT",null,null,null,0,"PARTIALLY FILLED @ 25",null,null,26,25]`,
			want:    Order{ID: 9007199254740993, Symbol: currency.NewBTCUSD(), Price: 26, AverageExecutionPrice: 25, Side: "sell", Type: "EXCHANGE LIMIT", Timestamp: types.Time(time.UnixMilli(1792648800123)), LastUpdated: types.Time(time.UnixMilli(1792648860456)), Status: "PARTIALLY FILLED @ 25", OriginalAmount: 3, RemainingAmount: 2, ExecutedAmount: 1},
		},
		{
			name:    "v2_filled_buy_extended_array",
			payload: `[2,null,null,"tDOGE:USD",1792648800123,1792648860456,0,3,"LIMIT",null,null,null,0,"EXECUTED @ 25",null,null,26,25,null,null]`,
			want:    Order{ID: 2, Symbol: currency.NewPairWithDelimiter("DOGE", "USD", ":"), Price: 26, AverageExecutionPrice: 25, Side: "buy", Type: "LIMIT", Timestamp: types.Time(time.UnixMilli(1792648800123)), LastUpdated: types.Time(time.UnixMilli(1792648860456)), Status: "EXECUTED @ 25", OriginalAmount: 3, ExecutedAmount: 3},
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
	for _, payload := range []string{`[]`, `[1]`, `[1,null,null,"tBTCUSD",1,2,3,4,"LIMIT",null,null,null,0,"ACTIVE",null,null,26]`} {
		var got Order
		assert.ErrorIs(t, json.Unmarshal([]byte(payload), &got), errOrderArrayTooShort, "truncated arrays should return the length sentinel")
	}
	for _, payload := range []string{
		`{"id":"not-an-id"}`, `{"symbol":"x"}`,
		`[1,null,null,"tBTCUSD",1,2,3,4,"LIMIT",null,null,null,0,"ACTIVE",null,null,"bad",25]`,
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
				assert.Equal(t, http.MethodPost, r.Method, "order endpoints should use POST")
				var payload string
				switch r.URL.Path {
				case "/v1/orders":
					payload = fmt.Sprintf(`[{"id":2,"symbol":"btcusd","side":"sell","type":%q,"timestamp":"1792648800.123","price":"26","avg_execution_price":"25","original_amount":"3","remaining_amount":"2","executed_amount":"1","is_live":true}]`, orderType)
				case "/v2/auth/r/orders/tBTCUSD/hist":
					var body struct {
						ID  []int64 `json:"id"`
						IDs []int64 `json:"ids"`
					}
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "history request should decode")
					assert.Empty(t, body.IDs, "v2 history should not send the v1 ids parameter")
					if len(body.ID) > 0 {
						assert.Len(t, body.ID, 1, "ID lookup should send one documented id filter")
						assert.Contains(t, []int64{1, 2}, body.ID[0], "ID lookup should use the requested order ID")
					}
					payload = fmt.Sprintf(`[[1,null,null,"tBTCUSD",1792648800123,1792648860456,-2,-3,%q,null,null,null,0,"PARTIALLY FILLED @ 25",null,null,26,25]]`, orderType)
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
			want := order.Detail{AssetType: a, Exchange: ex.Name, OrderID: "1", Pair: pair, Type: order.Limit, Side: order.Sell, Status: order.PartiallyFilled, Date: time.UnixMilli(1792648800123), LastUpdated: time.UnixMilli(1792648860456), Price: 26, AverageExecutedPrice: 25, Amount: 3, RemainingAmount: 2, ExecutedAmount: 1}
			got, err := ex.GetOrderInfo(t.Context(), "1", pair, a)
			require.NoError(t, err, "v2 history lookup must succeed")
			assert.Equal(t, &want, got, "ID lookup should preserve the complete order mapping")
			req := &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{pair}, Side: order.AnySide, Type: order.AnyType}
			history, err := ex.GetOrderHistory(t.Context(), req)
			require.NoError(t, err, "v2 history list must succeed")
			assert.Equal(t, order.FilteredOrders{want}, history, "history should share the ID lookup mapping")
			want.OrderID, want.LastUpdated, want.Status = "2", time.Time{}, order.Active
			want.Pair = pair.Lower()
			got, err = ex.GetOrderInfo(t.Context(), "2", pair, a)
			require.NoError(t, err, "v1 ID-only open order lookup must succeed")
			assert.Equal(t, &want, got, "open order lookup should preserve the complete order mapping")
			active, err := ex.GetActiveOrders(t.Context(), req)
			require.NoError(t, err, "v1 active order list must succeed")
			assert.Equal(t, order.FilteredOrders{want}, active, "active order list should share the ID lookup mapping")
		})
	}
}

func TestParseOrderV2Status(t *testing.T) {
	t.Parallel()
	for status, want := range map[string]order.Status{
		"EXECUTED @ 25": order.Filled, "FORCED EXECUTED @ 25": order.Filled,
		"CANCELED": order.Cancelled, "POSTONLY CANCELED": order.Cancelled,
		"FILLORKILL CANCELED": order.Cancelled, "IOC CANCELED": order.Cancelled,
		"INSUFFICIENT BALANCE (G1)": order.InsufficientBalance,
	} {
		got, err := e.parseOrderToOrderDetail(&Order{Side: "buy", Type: "EXCHANGE LIMIT", Status: status})
		require.NoError(t, err, "documented v2 statuses must parse")
		assert.Equal(t, want, got.Status, "terminal status should not become an active order")
	}
}

func TestParseOrderV2TimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType string
		wantAsset asset.Item
		wantTIF   order.TimeInForce
	}{
		{"EXCHANGE FOK", asset.Spot, order.FillOrKill},
		{"IOC", asset.Margin, order.ImmediateOrCancel},
	} {
		got, err := e.parseOrderToOrderDetail(&Order{Side: "buy", Type: tc.orderType, Status: "ACTIVE"})
		require.NoError(t, err, "v2 execution instructions must parse")
		assert.Equal(t, order.Limit, got.Type, "execution instruction should preserve the underlying limit order")
		assert.Equal(t, tc.wantTIF, got.TimeInForce, "execution instruction should retain its time in force")
		assert.Equal(t, tc.wantAsset, got.AssetType, "execution instruction should retain its market")
	}
}
