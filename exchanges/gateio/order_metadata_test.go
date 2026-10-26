package gateio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestSpotOrderMetadata(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, tif, text string
		want            order.TimeInForce
		wantErr         bool
	}{
		{name: "gtc", tif: "gtc", text: "t-client", want: order.GoodTillCancel},
		{name: "ioc", tif: "ioc", text: "t-client", want: order.ImmediateOrCancel},
		{name: "poc", tif: "poc", text: "t-client", want: order.PostOnly},
		{name: "fok", tif: "fok", text: "t-client", want: order.FillOrKill},
		{name: "omitted", want: order.UnknownTIF},
		{name: "invalid", tif: "invalid", text: "t-client", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, true), "test pair must be enabled")
			metadata := ""
			if tc.tif != "" {
				metadata = fmt.Sprintf(`,"time_in_force":%q,"text":%q`, tc.tif, tc.text)
			}
			payload := `{"id":"1","currency_pair":"BTC_USDT","status":"open","type":"limit","side":"buy","amount":"3","left":"1","price":"60000"` + metadata + `}`
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				response := payload
				if strings.HasSuffix(r.URL.Path, "/spot/open_orders") {
					response = `[{"currency_pair":"BTC_USDT","total":1,"orders":[` + payload + `]}]`
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "mock order metadata should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "test client must be installed")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			detail, err := ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), asset.Spot)
			if tc.wantErr {
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported time in force should be rejected by detail lookup")
			} else {
				require.NoError(t, err, "detail lookup must accept reported metadata")
				assert.Equal(t, tc.want, detail.TimeInForce, "detail lookup should retain the reported time in force")
				assert.Equal(t, tc.text, detail.ClientOrderID, "detail lookup should retain the reported text without guessing")
			}
			active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Pairs: currency.Pairs{currency.NewBTCUSDT()}, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
			if tc.wantErr {
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported time in force should be rejected by active lookup")
			} else {
				require.NoError(t, err, "active lookup must accept reported metadata")
				require.Len(t, active, 1, "active lookup must return the fixture order")
				assert.Equal(t, tc.want, active[0].TimeInForce, "active lookup should retain the reported time in force")
				assert.Equal(t, tc.text, active[0].ClientOrderID, "active lookup should retain the reported text without guessing")
			}
		})
	}
}
