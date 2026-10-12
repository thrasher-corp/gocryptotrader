package gemini

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		failure bool
	}{{"returned statuses", false}, {"request failure", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex))
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.failure {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"failure"}`))
					return
				}
				_, _ = w.Write([]byte(`{"result":"ok","details":{"cancelledOrders":["1","2"],"cancelRejects":["3"]}}`))
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL))
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			result, err := ex.CancelAllOrders(t.Context(), &order.Cancel{})
			if tc.failure {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, map[string]string{"1": order.Cancelled.String(), "2": order.Cancelled.String(), "3": "Could not cancel order"}, result.Status)
		})
	}
}
