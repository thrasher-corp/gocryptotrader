package bithumb

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestOrderTransactionType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side order.Side
		want string
		err  error
	}{
		{side: order.Buy, want: "bid"},
		{side: order.Bid, want: "bid"},
		{side: order.Long, want: "bid"},
		{side: order.Sell, want: "ask"},
		{side: order.Ask, want: "ask"},
		{side: order.Short, want: "ask"},
		{side: order.AnySide},
		{side: order.UnknownSide},
		{side: order.ClosePosition, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.side.String(), func(t *testing.T) {
			t.Parallel()
			got, err := orderTransactionType(tc.side)
			assert.ErrorIs(t, err, tc.err, "side conversion should validate recognised aliases")
			assert.Equal(t, tc.want, got, "side conversion should map to the venue filter")
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side order.Side
		want string
		err  error
	}{
		{side: order.Buy, want: "bid"},
		{side: order.Bid, want: "bid"},
		{side: order.Long, want: "bid"},
		{side: order.Sell, want: "ask"},
		{side: order.Ask, want: "ask"},
		{side: order.Short, want: "ask"},
		{side: order.AnySide},
		{side: order.UnknownSide},
		{side: order.ClosePosition, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.side.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "test exchange must configure")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			var requests atomic.Int32
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, "/trade/cancel", r.URL.Path, "single cancellation should use the cancellation endpoint")
				if !assert.NoError(t, r.ParseForm(), "cancellation form should parse") {
					return
				}
				assert.Equal(t, "Opaque-aB", r.Form.Get("order_id"), "opaque order identifier should remain unchanged")
				assert.Equal(t, "BTC", r.Form.Get("order_currency"), "base currency should be sent")
				assert.Equal(t, "KRW", r.Form.Get("payment_currency"), "quote currency should be sent")
				assert.NotContains(t, r.Form, "currency", "obsolete currency parameter should be absent")
				if tc.want == "" {
					assert.NotContains(t, r.Form, "type", "unknown side should omit type")
				} else {
					assert.Equal(t, tc.want, r.Form.Get("type"), "wire side should use venue spelling")
				}
				_, err := w.Write([]byte(`{"status":"0000"}`))
				assert.NoError(t, err, "cancellation response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must install")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "mock endpoint must install")
			err := ex.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: testPair, OrderID: "Opaque-aB", Side: tc.side})
			require.ErrorIs(t, err, tc.err, "single cancellation must validate its side")
			wantRequests := int32(1)
			if tc.err != nil {
				wantRequests = 0
			}
			assert.Equal(t, wantRequests, requests.Load(), "invalid cancellation should stop before an HTTP request")
		})
	}
}
