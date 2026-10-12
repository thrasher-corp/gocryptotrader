package yobit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		wantError  error
		want       map[string]int
	}{
		{name: "payload", body: `{"success":1,"return":{"order_id":42}}`, want: map[string]int{"order_id": 42}},
		{name: "empty map", body: `{"success":1,"return":{}}`, want: map[string]int{}},
		{name: "rejected", body: `{"success":0,"error":"cancel rejected"}`, wantError: request.ErrAuthRequestFailed},
		{name: "missing success", body: `{"return":{"order_id":42}}`, wantError: request.ErrAuthRequestFailed},
		{name: "bare payload", body: `{"order_id":42}`, wantError: request.ErrAuthRequestFailed},
		{name: "missing payload", body: `{"success":1}`, wantError: common.ErrInvalidResponse},
		{name: "null payload", body: `{"success":1,"return":null}`, wantError: common.ErrInvalidResponse},
		{name: "inconsistent error", body: `{"success":1,"return":{},"error":"rejected"}`, wantError: request.ErrAuthRequestFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "test exchange must configure")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "trade API should use POST")
				assert.NotEmpty(t, r.Header.Get("Sign"), "trade API should sign its payload")
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "venue response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must install")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL), "mock endpoint must install")
			var got map[string]int
			err := ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpotSupplementary, privateCancelOrder, nil, &got)
			assert.ErrorIs(t, err, tc.wantError, "authentication should enforce the documented envelope")
			assert.Equal(t, tc.want, got, "the return payload should populate the caller's destination only after success")
			if tc.name == "rejected" {
				assert.ErrorContains(t, err, "cancel rejected", "failure should retain the venue reason")
			}
		})
	}
	t.Run("malformed payload", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "test exchange must configure")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"success":1,"return":{"order_id":"invalid"}}`))
			assert.NoError(t, err, "venue response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must install")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL), "mock endpoint must install")
		var got map[string]int
		assert.Error(t, ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpotSupplementary, privateCancelOrder, nil, &got), "malformed return payload should propagate its decoding error")
	})
	for method, payload := range map[string]string{
		privateAccountInfo:            `{"funds":{"btc":2}}`,
		privateTrade:                  `{"order_id":42}`,
		privateActiveOrders:           `{"42":{"amount":2}}`,
		privateOrderInfo:              `{"42":{"amount":2}}`,
		privateCancelOrder:            `{"order_id":42}`,
		privateTradeHistory:           `{"42":{"amount":2}}`,
		privateWithdrawCoinsToAddress: `{"server_time":1437146228}`,
		privateCreateCoupon:           `{"coupon":"coupon-value"}`,
		privateRedeemCoupon:           `{"couponAmount":"2","couponCurrency":"BTC"}`,
	} {
		t.Run(method+" caller", func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "test exchange must configure")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !assert.NoError(t, r.ParseForm(), "trade request should parse") {
					return
				}
				assert.Equal(t, method, r.Form.Get("method"), "caller should route its own operation")
				_, err := w.Write([]byte(`{"success":1,"return":` + payload + `}`))
				assert.NoError(t, err, "venue response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must install")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL), "mock endpoint must install")
			switch method {
			case privateAccountInfo:
				got, err := ex.GetAccountInformation(t.Context())
				require.NoError(t, err, "account payload must decode")
				assert.Equal(t, 2.0, got.Funds["btc"], "account return should retain funds")
			case privateTrade:
				got, err := ex.Trade(t.Context(), "btc_usd", "buy", 1, 1)
				require.NoError(t, err, "trade payload must decode")
				assert.Equal(t, int64(42), got, "trade return should retain order ID")
			case privateActiveOrders:
				got, err := ex.GetOpenOrders(t.Context(), "btc_usd")
				require.NoError(t, err, "active orders must decode")
				assert.Equal(t, 2.0, got["42"].Amount, "active order should retain amount")
			case privateOrderInfo:
				got, err := ex.GetOrderInformation(t.Context(), 42)
				require.NoError(t, err, "order payload must decode")
				assert.Equal(t, 2.0, got["42"].Amount, "order should retain amount")
			case privateCancelOrder:
				require.NoError(t, ex.CancelExistingOrder(t.Context(), 42), "cancellation envelope must decode")
			case privateTradeHistory:
				got, err := ex.GetTradeHistory(t.Context(), 0, 1, 0, 0, 0, "", "btc_usd")
				require.NoError(t, err, "trade history must decode once")
				assert.Equal(t, 2.0, got["42"].Amount, "trade history should retain amount")
			case privateWithdrawCoinsToAddress:
				got, err := ex.WithdrawCoinsToAddress(t.Context(), "BTC", 1, "address")
				require.NoError(t, err, "withdrawal must decode")
				assert.Equal(t, int64(1437146228), got.ServerTime.Time().Unix(), "withdrawal should retain timestamp")
			case privateCreateCoupon:
				got, err := ex.CreateCoupon(t.Context(), "BTC", 1)
				require.NoError(t, err, "coupon must decode")
				assert.Equal(t, "coupon-value", got.Coupon, "coupon return should retain value")
			case privateRedeemCoupon:
				got, err := ex.RedeemCoupon(t.Context(), "coupon-value")
				require.NoError(t, err, "redeemed coupon must decode")
				assert.Equal(t, 2.0, got.CouponAmount, "redeemed coupon should retain amount")
			}
		})
	}
}
