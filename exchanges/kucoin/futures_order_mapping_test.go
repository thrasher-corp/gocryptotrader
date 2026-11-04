package kucoin

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

func TestGetOrderInfoFuturesExecutionMetadata(t *testing.T) {
	t.Parallel()
	for _, quote := range []string{"USDTM", "USDM"} {
		t.Run(quote, func(t *testing.T) {
			t.Parallel()
			for _, tc := range []struct {
				name, average       string
				filled, wantAverage float64
			}{
				{name: "quoted", average: `"59500"`, filled: 2, wantAverage: 59500},
				{name: "numeric", average: `59600`, filled: 2, wantAverage: 59600},
				{name: "unfilled", average: `"0"`},
				{name: "omitted", filled: 2},
				{name: "null", average: `null`, filled: 2},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					ex := new(Exchange)
					require.NoError(t, testexch.Setup(ex), "Setup must not error")
					require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
					pair := currency.NewPairWithDelimiter("XBT", quote, "")
					sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Futures, pair)
					average := ""
					if tc.average != "" {
						average = `,"avgDealPrice":` + tc.average
					}
					server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						_, err := fmt.Fprintf(w, `{"code":"200000","data":{"id":"1","symbol":%q,"type":"limit","side":"buy","price":"61000","size":3,"dealSize":%v,"isActive":true,"createdAt":1735720637000,"updatedAt":1735720638000,"endAt":null%s}}`, pair.String(), tc.filled, average)
						assert.NoError(t, err, "mock futures order should be written")
					}))
					require.NoError(t, ex.SetHTTPClient(server.Client()), "test client must be installed")
					require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "fixture URL must be installed")
					ex.API.AuthenticatedSupport = true
					ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
					ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"})
					got, err := ex.GetOrderInfo(t.Context(), "1", pair, asset.Futures)
					require.NoError(t, err, "futures order lookup must not error")
					assert.Equal(t, 61000.0, got.Price, "request price should remain distinct from execution price")
					assert.Equal(t, tc.wantAverage, got.AverageExecutedPrice, "average should use avgDealPrice or remain unknown")
					assert.Equal(t, 3.0, got.Amount, "order amount should retain contract count")
					assert.Equal(t, tc.filled, got.ExecutedAmount, "filled amount should retain executed contract count")
					assert.Zero(t, got.QuoteAmount, "contract count should not be exposed as a requested quote budget")
				})
			}
		})
	}
}

func TestFuturesActiveOrdersPreserveAverage(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
	pair := currency.NewPairWithDelimiter("XBT", "USDTM", "")
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Futures, pair)
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := `{"id":"1","symbol":"XBTUSDTM","type":"limit","side":"buy","price":"61000","size":3,"dealSize":1,"filledSize":1,"avgDealPrice":"59500","isActive":true,"status":"open","createdAt":1735720637000,"updatedAt":1735720638000}`
		if r.URL.Path == "/v1/orders" {
			payload = `{"currentPage":1,"totalPage":1,"items":[{"id":"1","symbol":"XBTUSDTM","type":"limit","side":"buy","price":"61000","size":3,"filledSize":2,"avgDealPrice":"59600","isActive":true,"status":"open","createdAt":1735720637000,"updatedAt":1735720639000}]}`
		}
		_, err := fmt.Fprintf(w, `{"code":"200000","data":%s}`, payload)
		assert.NoError(t, err, "futures order fixture should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "fixture client must be installed")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "fixture URL must be installed")
	ex.API.AuthenticatedSupport = true
	ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"})
	stored, err := ex.GetOrderInfo(t.Context(), "1", pair, asset.Futures)
	require.NoError(t, err, "initial lookup must succeed")
	assert.Equal(t, 59500.0, stored.AverageExecutedPrice, "initial lookup should report the first average")
	active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{pair}})
	require.NoError(t, err, "active futures sweep must succeed")
	require.Len(t, active, 1, "active sweep must return one order")
	assert.Equal(t, 59600.0, active[0].AverageExecutedPrice, "active sweep should retain avgDealPrice")
	require.NoError(t, stored.UpdateOrderFromDetail(&active[0]), "active sweep must merge into stored detail")
	assert.Equal(t, 2.0, stored.ExecutedAmount, "sweep should advance cumulative contracts")
	assert.Equal(t, 59600.0, stored.AverageExecutedPrice, "sweep should not clear the newly reported average")
}

func TestFuturesTradeOrderLifecycle(t *testing.T) {
	t.Parallel()
	for _, terminal := range []string{"filled", "canceled", "match"} {
		t.Run(terminal, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			pair := currency.NewPairWithDelimiter("XBT", "USDTM", "")
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Futures, pair)
			stored := order.Detail{Exchange: ex.Name, OrderID: "order-1", AssetType: asset.Futures, Pair: pair, Amount: 3}
			terminalFilled, terminalRemaining, terminalStatus := 3, 0, order.Filled
			terminalWireStatus := "done"
			if terminal == "match" {
				terminalWireStatus = "match"
			}
			if terminal == "canceled" {
				terminalFilled, terminalRemaining, terminalStatus = 2, 1, order.Cancelled
			}
			for i, step := range []struct {
				kind, status      string
				filled, remaining int
				want              order.Status
			}{
				{"open", "open", 0, 3, order.Open},
				{"match", "match", 1, 2, order.PartiallyFilled},
				{"match", "open", 2, 1, order.PartiallyFilled},
				{terminal, terminalWireStatus, terminalFilled, terminalRemaining, terminalStatus},
			} {
				payload := fmt.Appendf(nil, `{"orderId":"order-1","tradeId":"trade-%d","symbol":"XBTUSDTM","type":%q,"status":%q,"orderType":"limit","side":"buy","price":"61000","size":"3","filledSize":"%d","remainSize":"%d","clientOid":"client-1","orderTime":1735720637000000000,"ts":%d}`, i, step.kind, step.status, step.filled, step.remaining, int64(1735720638000000000)+int64(i)*1000000000)
				require.NoError(t, ex.processFuturesPrivateTradeOrders(t.Context(), payload), "futures lifecycle push must decode")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "lifecycle push must emit one order")
				got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
				require.True(t, ok, "lifecycle push must emit an order detail")
				assert.Equal(t, "order-1", got.OrderID, "push should retain orderId rather than tradeId")
				assert.Equal(t, step.want, got.Status, "push should distinguish partial executions from terminal events")
				assert.Equal(t, time.Unix(1735720637, 0), got.Date, "order creation time should remain separate from update time")
				assert.Equal(t, time.Unix(1735720638+int64(i), 0), got.LastUpdated, "push should retain its actual update timestamp")
				require.NoError(t, stored.UpdateOrderFromDetail(got), "lifecycle push must merge under the stored order ID")
				assert.Equal(t, step.want, stored.Status, "stored order should reflect the lifecycle status")
				assert.Equal(t, float64(step.filled), stored.ExecutedAmount, "stored order should retain cumulative execution")
			}
		})
	}
}
