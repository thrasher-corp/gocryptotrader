package mexc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const (
	testCredentialKey    = "mock"
	testCredentialSecret = "tester"
)

func newSignedTestExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "test exchange Setup must not error")
	ex.SetCredentials(&accounts.Credentials{Key: testCredentialKey, Secret: testCredentialSecret})
	ex.GetBase().SkipAuthCheck = true
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	for k := range ex.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(k, server.URL), "SetRunningURL must not error for %s", k)
	}
	return ex
}

func TestPrivateEndpointRequestConstruction(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		call         func(ctx context.Context, e *Exchange) error
		method       string
		pathContains string
		body         string
	}{
		{"CreateBrokerSubAccount", func(ctx context.Context, e *Exchange) error {
			_, err := e.CreateBrokerSubAccount(ctx)
			return err
		}, http.MethodPost, "/broker/sub-account/virtualSubAccount", "{}"},
		{"GetBrokerAccountSubAccountList", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetBrokerAccountSubAccountList(ctx, "", 0, 0)
			return err
		}, http.MethodGet, "/broker/sub-account/list", "{}"},
		{"GetSubAccountStatus", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountStatus(ctx, "sub1")
			return err
		}, http.MethodGet, "/broker/sub-account/status", "{}"},
		{"GetSubAccountUnversalTransferHistory", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountUnversalTransferHistory(ctx, "", "", asset.Spot, asset.Spot, time.Time{}, time.Time{}, 0, 0)
			return err
		}, http.MethodGet, "/capital/sub-account/universalTransfer", "{}"},
		{"GetAccountInformation", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetAccountInformation(ctx)
			return err
		}, http.MethodGet, "/account", "{}"},
		{"GetOpenOrders", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetOpenOrders(ctx, spotTradablePair)
			return err
		}, http.MethodGet, "/openOrders", "[]"},
		{"GetOrderByID", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetOrderByID(ctx, spotTradablePair, "", "123")
			return err
		}, http.MethodGet, "/order", "{}"},
		{"CancelAllOpenOrdersBySymbol", func(ctx context.Context, e *Exchange) error {
			_, err := e.CancelAllOpenOrdersBySymbol(ctx, spotTradablePair)
			return err
		}, http.MethodDelete, "/openOrders", "[]"},
		{"GetSubAccountAsset", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetSubAccountAsset(ctx, "sub1", asset.Spot)
			return err
		}, http.MethodGet, "/sub-account/asset", "{}"},
		{"GetDepositAddressOfCoin", func(ctx context.Context, e *Exchange) error {
			_, err := e.GetDepositAddressOfCoin(ctx, currency.USDT, "")
			return err
		}, http.MethodGet, "/capital/deposit/address", "[]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotMethod, gotPath, gotAPIKey, gotSignature string
			e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAPIKey = r.Header.Get("X-MEXC-APIKEY")
				gotSignature = r.URL.Query().Get("signature")
				_, _ = w.Write([]byte(tc.body))
			}))
			err := tc.call(t.Context(), e)
			require.NoError(t, err)
			assert.Equal(t, tc.method, gotMethod, "request method should match the documented endpoint")
			assert.Contains(t, gotPath, tc.pathContains, "request path should target the documented endpoint")
			assert.Truef(t, strings.HasPrefix(gotPath, "/api/v3/"), "spot and broker endpoints should be versioned under /api/v3/, got %s", gotPath)
			assert.NotEmpty(t, gotAPIKey, "X-MEXC-APIKEY header should be set on an authenticated request")
			assert.NotEmpty(t, gotSignature, "signature query parameter should be set on an authenticated request")
		})
	}
}
