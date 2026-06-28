package mexc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func newSignedTestExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	te := new(Exchange)
	require.NoError(t, testexch.Setup(te), "test exchange Setup must not error")
	te.SetCredentials("mock", "tester", "", "", "", "")
	te.GetBase().SkipAuthCheck = true
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	for k := range te.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, te.API.Endpoints.SetRunningURL(k, server.URL), "SetRunningURL must not error for %s", k)
	}
	return te
}

func TestPrivateEndpointRequestConstruction(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		call         func(ctx context.Context, e *Exchange) error
		method       string
		pathContains string
	}{
		{"CreateBrokerSubAccount", func(ctx context.Context, e *Exchange) error {
			_, err := e.CreateBrokerSubAccount(ctx)
			return err
		}, http.MethodPost, "/broker/sub-account/virtualSubAccount"},
		{"GetBrokerAccountSubAccountList", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetBrokerAccountSubAccountList(ctx, "", 0, 0)
			return err
		}, http.MethodGet, "/broker/sub-account/list"},
		{"GetSubAccountStatus", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountStatus(ctx, "sub1")
			return err
		}, http.MethodGet, "/broker/sub-account/status"},
		{"GetSubAccountUnversalTransferHistory", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountUnversalTransferHistory(ctx, "", "", asset.Spot, asset.Spot, time.Time{}, time.Time{}, 0, 0)
			return err
		}, http.MethodGet, "/capital/sub-account/universalTransfer"},
		{"GetAllUserAssetsInformation", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetAllUserAssetsInformation(ctx)
			return err
		}, http.MethodGet, "/private/account/assets"},
		{"GetUserSingleCurrencyAssetInformation", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetUserSingleCurrencyAssetInformation(ctx, currency.USDT)
			return err
		}, http.MethodGet, "/private/account/asset/USDT"},
		{"GetOrderBasedOnExternalNumber", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetOrderBasedOnExternalNumber(ctx, futuresTradablePair, "ext123")
			return err
		}, http.MethodGet, "/private/order/external/"},
		{"GetPositionMode", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetPositionMode(ctx)
			return err
		}, http.MethodGet, "/private/position/position_mode"},
		{"IncreaseDecreaseMargin", func(ctx context.Context, e *Exchange) error {
			return e.IncreaseDecreaseMargin(ctx, 1, 1, "ADD")
		}, http.MethodPost, "/private/position/change_margin"},
		{"CancelOrderByClientOrderID", func(ctx context.Context, e *Exchange) error {
			_, err := e.CancelOrderByClientOrderID(ctx, futuresTradablePair, "ext123")
			return err
		}, http.MethodPost, "/private/order/cancel_with_external"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotMethod, gotPath, gotAPIKey, gotSignature string
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAPIKey = r.Header.Get("X-MEXC-APIKEY")
				gotSignature = r.URL.Query().Get("signature")
				_, _ = w.Write([]byte(`{}`))
			}))
			err := tc.call(t.Context(), e)
			require.NoError(t, err)
			assert.Equal(t, tc.method, gotMethod, "request method should match the documented endpoint")
			assert.Contains(t, gotPath, tc.pathContains, "request path should target the documented endpoint")
			assert.NotEmpty(t, gotAPIKey, "X-MEXC-APIKEY header should be set on an authenticated request")
			assert.NotEmpty(t, gotSignature, "signature query parameter should be set on an authenticated request")
		})
	}
}
