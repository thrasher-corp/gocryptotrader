package btse

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
				_, _ = w.Write([]byte(`[{"orderID":"1","status":6},{"orderID":"2","status":-1},{"orderID":"3","status":16},{"orderID":"4","status":17},{"orderID":"5","status":999}]`))
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL))
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			result, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot})
			if tc.failure {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, map[string]string{"1": order.Cancelled.String(), "2": "Cancellation status -1", "3": "Cancellation status 16", "4": "Cancellation status 17", "5": "Cancellation status 999"}, result.Status)
		})
	}
}
