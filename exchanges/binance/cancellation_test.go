package binance

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestCancelMarginOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		id       int64
		client   string
		response string
		failure  bool
	}{
		{"order ID", 42, "", `{"orderId":42}`, false},
		{"client ID", 0, "client", `{"orderId":42}`, false},
		{"rejected", 42, "", `{"code":-2011,"msg":"Unknown order"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex))
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/sapi/v1/margin/order", r.URL.Path)
				assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))
				assert.Equal(t, tc.client, r.URL.Query().Get("origClientOrderId"))
				assert.NotEmpty(t, r.URL.Query().Get("signature"))
				assert.NotEmpty(t, r.URL.Query().Get("timestamp"))
				assert.Equal(t, "key", r.Header.Get("X-MBX-APIKEY"))
				_, _ = w.Write([]byte(tc.response))
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL))
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			result, err := ex.CancelMarginOrder(t.Context(), currency.NewBTCUSDT(), tc.id, tc.client)
			if tc.failure {
				require.ErrorContains(t, err, "Unknown order")
				return
			}
			require.NoError(t, err)
			assert.EqualValues(t, 42, result.OrderID)
		})
	}
}

func TestCancelAllMarginOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		count      int
		failure    bool
	}{
		{"plain and OCO", `[{"orderId":11,"status":"CANCELED"},{"orderListId":5,"orderReports":[{"orderId":20,"status":"CANCELED"},{"orderId":21,"status":"CANCELED"}]}]`, 2, false},
		{"empty", `[]`, 0, false},
		{"rejected", `{"code":-2011,"msg":"Unknown order"}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex))
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/sapi/v1/margin/openOrders", r.URL.Path)
				assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))
				assert.NotEmpty(t, r.URL.Query().Get("signature"))
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL))
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			result, err := ex.CancelAllMarginOrders(t.Context(), currency.NewBTCUSDT())
			if tc.failure {
				require.ErrorContains(t, err, "Unknown order")
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tc.count)
			if tc.count != 0 {
				assert.Len(t, result[1].OrderReports, 2)
			}
		})
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex))
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))
				if a == asset.Margin {
					assert.Equal(t, "/sapi/v1/margin/openOrders", r.URL.Path)
					assert.Equal(t, http.MethodDelete, r.Method)
					_, _ = w.Write([]byte(`[{"orderId":11,"status":"CANCELED"},{"orderListId":5,"orderReports":[{"orderId":20,"status":"CANCELED"},{"orderId":21,"status":"NEW"}]}]`))
					return
				}
				if r.Method == http.MethodGet {
					assert.Equal(t, "/api/v3/openOrders", r.URL.Path)
					_, _ = w.Write([]byte(`[{"orderId":11},{"orderId":20},{"orderId":21}]`))
				} else {
					assert.Equal(t, "/api/v3/order", r.URL.Path)
					if r.URL.Query().Get("orderId") == "20" {
						_, _ = w.Write([]byte(`{"code":-2011,"msg":"Unknown order"}`))
					} else {
						_, _ = w.Write([]byte(`{"orderId":11}`))
					}
				}
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL))
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			result, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: a})
			require.NoError(t, err)
			require.NotNil(t, result)
			if a == asset.Margin {
				assert.Equal(t, map[string]string{"11": order.Cancelled.String(), "20": order.Cancelled.String(), "21": "NEW"}, result.Status)
			} else {
				assert.Equal(t, map[string]string{"11": order.Cancelled.String(), "20": "Unknown order", "21": order.Cancelled.String()}, result.Status)
			}
		})
	}
}
