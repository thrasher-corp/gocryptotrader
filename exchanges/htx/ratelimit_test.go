package htx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetRateLimit(t *testing.T) {
	t.Parallel()
	definitions := GetRateLimit()
	assert.Contains(t, definitions, request.Unset, "spot limiter should be defined")
	assert.Contains(t, definitions, htxFuturesAuth, "delivery authenticated limiter should be defined")
	assert.Contains(t, definitions, htxFuturesUnAuth, "delivery public limiter should be defined")
	assert.Contains(t, definitions, htxSwapAuth, "swap authenticated limiter should be defined")
	assert.Contains(t, definitions, htxSwapUnAuth, "swap public limiter should be defined")
	assert.Contains(t, definitions, htxFuturesTransfer, "transfer limiter should be defined")
	type testCase struct {
		name              string
		endpoint          request.EndpointLimit
		requestsPerSecond float64
	}
	for _, tc := range []testCase{
		{name: "coin leverage switch", endpoint: htxSwapSwitchLeverage, requestsPerSecond: 1.0 / 3},
		{name: "universal transfer", endpoint: htxUniversalTransfer, requestsPerSecond: 2.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			limiter, ok := definitions[tc.endpoint]
			require.True(t, ok, "dedicated endpoint limiter must exist")
			require.NotNil(t, limiter, "dedicated endpoint limiter must be configured")
			assert.InDelta(t, tc.requestsPerSecond, float64(limiter.Limit()), 1e-12, "limiter should match HTX's documented rate")
			assert.Equal(t, request.Weight(1), limiter.Weight(), "one request should consume one token")
			for _, other := range []request.EndpointLimit{request.Unset, htxSwapAuth, htxFuturesTransfer, htxSetAssetMode} {
				assert.False(t, limiter.SharesBudgetWith(definitions[other]), "endpoint-specific allowance should not replace another operation's budget")
			}
		})
	}
}

func TestGetRateLimitID(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		endpoint      exchange.URL
		path          string
		method        string
		authenticated bool
		expected      request.EndpointLimit
	}
	for _, tc := range []testCase{
		{name: "set asset mode", endpoint: exchange.RestUSDTMargined, path: "/v5/account/asset_mode", method: http.MethodPost, authenticated: true, expected: htxSetAssetMode},
		{name: "get asset mode", endpoint: exchange.RestUSDTMargined, path: "/v5/account/asset_mode", method: http.MethodGet, authenticated: true, expected: htxSwapAuth},
		{name: "coin leverage switch", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_switch_lever_rate", method: http.MethodPost, authenticated: true, expected: htxSwapSwitchLeverage},
		{name: "dedicated coin endpoint", endpoint: exchange.RestCoinMargined, path: "/swap-api/v1/swap_switch_lever_rate", method: http.MethodPost, authenticated: true, expected: htxSwapSwitchLeverage},
		{name: "coin leverage public", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_switch_lever_rate", method: http.MethodPost, expected: htxSwapUnAuth},
		{name: "coin leverage wrong method", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_switch_lever_rate", method: http.MethodGet, authenticated: true, expected: htxSwapAuth},
		{name: "coin leverage wrong endpoint", endpoint: exchange.RestUSDTMargined, path: "/swap-api/v1/swap_switch_lever_rate", method: http.MethodPost, authenticated: true, expected: htxSwapAuth},
		{name: "universal transfer", endpoint: exchange.RestSpot, path: "/v5/account/universal_transfer", method: http.MethodPost, authenticated: true, expected: htxUniversalTransfer},
		{name: "universal transfer records", endpoint: exchange.RestSpot, path: "/v5/account/universal_transfer_records", method: http.MethodGet, authenticated: true, expected: request.Unset},
		{name: "universal transfer public", endpoint: exchange.RestSpot, path: "/v5/account/universal_transfer", method: http.MethodPost, expected: request.Unset},
		{name: "universal transfer wrong method", endpoint: exchange.RestSpot, path: "/v5/account/universal_transfer", method: http.MethodGet, authenticated: true, expected: request.Unset},
		{name: "universal transfer wrong endpoint", endpoint: exchange.RestFutures, path: "/v5/account/universal_transfer", method: http.MethodPost, authenticated: true, expected: htxFuturesTransfer},
		{name: "spot", endpoint: exchange.RestSpot, expected: request.Unset},
		{name: "delivery public", endpoint: exchange.RestFutures, path: "/api/v1/contract_info", expected: htxFuturesUnAuth},
		{name: "delivery private", endpoint: exchange.RestFutures, path: "/api/v1/contract_order", authenticated: true, expected: htxFuturesAuth},
		{name: "coin public", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_contract_info", expected: htxSwapUnAuth},
		{name: "coin private", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_order", authenticated: true, expected: htxSwapAuth},
		{name: "USDT public", endpoint: exchange.RestUSDTMargined, path: "/v5/market/open_interest", expected: htxSwapUnAuth},
		{name: "USDT private", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/order", authenticated: true, expected: htxSwapAuth},
		{name: "transfer", endpoint: exchange.RestFutures, path: "/api/v1/contract_master_sub_transfer", authenticated: true, expected: htxFuturesTransfer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, getRateLimitID(tc.endpoint, tc.path, tc.method, tc.authenticated), "rate-limit endpoint should match")
		})
	}
}
