package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
}

func TestGetRateLimitID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		endpoint      exchange.URL
		path          string
		authenticated bool
		expected      request.EndpointLimit
	}{
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
			assert.Equal(t, tc.expected, getRateLimitID(tc.endpoint, tc.path, tc.authenticated), "rate-limit endpoint should match")
		})
	}
}
