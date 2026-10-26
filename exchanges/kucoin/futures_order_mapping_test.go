package kucoin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
