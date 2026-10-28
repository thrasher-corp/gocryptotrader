package htx

import (
	"net/http"
	"strings"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// HTX rate limits per API Key
	htxSpotRateInterval = time.Second * 1
	htxSpotRequestRate  = 7

	htxFuturesRateInterval    = time.Second * 3
	htxFuturesAuthRequestRate = 30
	// Non market-request public interface rate
	htxFuturesUnAuthRequestRate    = 60
	htxFuturesTransferRateInterval = time.Second * 3
	htxFuturesTransferReqRate      = 10

	htxSwapRateInterval      = time.Second * 3
	htxSwapAuthRequestRate   = 30
	htxSwapUnauthRequestRate = 60

	htxFuturesAuth request.EndpointLimit = iota
	htxFuturesUnAuth
	htxFuturesTransfer
	htxSwapAuth
	htxSwapUnAuth
	htxSetAssetMode
	htxSwapSwitchLeverage
	htxUniversalTransfer
)

// Share rate-limit budgets across exchange instances and requester reinitialisation.
var rateLimits = request.RateLimitDefinitions{
	htxSwapSwitchLeverage: request.NewRateLimitWithWeight(3*time.Second, 1, 1),
	htxUniversalTransfer:  request.NewRateLimitWithWeight(2*time.Second, 5, 1),
	htxSetAssetMode:       request.NewRateLimitWithWeight(10*time.Second, 1, 1),
	request.Unset:         request.NewRateLimitWithWeight(htxSpotRateInterval, htxSpotRequestRate, 1),
	htxFuturesAuth:        request.NewRateLimitWithWeight(htxFuturesRateInterval, htxFuturesAuthRequestRate, 1),
	htxFuturesUnAuth:      request.NewRateLimitWithWeight(htxFuturesRateInterval, htxFuturesUnAuthRequestRate, 1),
	htxSwapAuth:           request.NewRateLimitWithWeight(htxSwapRateInterval, htxSwapAuthRequestRate, 1),
	htxSwapUnAuth:         request.NewRateLimitWithWeight(htxSwapRateInterval, htxSwapUnauthRequestRate, 1),
	htxFuturesTransfer:    request.NewRateLimitWithWeight(htxFuturesTransferRateInterval, htxFuturesTransferReqRate, 1),
}

func getRateLimitID(ep exchange.URL, path, method string, authenticated bool) request.EndpointLimit {
	if authenticated && ep == exchange.RestUSDTMargined && path == "/v5/account/asset_mode" && method == http.MethodPost {
		return htxSetAssetMode
	}
	if authenticated && method == http.MethodPost {
		if (ep == exchange.RestFutures || ep == exchange.RestCoinMargined) && path == "/swap-api/v1/swap_switch_lever_rate" {
			return htxSwapSwitchLeverage
		}
		if ep == exchange.RestSpot && path == "/v5/account/universal_transfer" {
			return htxUniversalTransfer
		}
	}
	if ep == exchange.RestSpot {
		return request.Unset
	}
	if authenticated && strings.Contains(path, "transfer") {
		return htxFuturesTransfer
	}
	isSwap := ep == exchange.RestUSDTMargined ||
		ep == exchange.RestCoinMargined ||
		strings.HasPrefix(path, "/swap-")
	if isSwap {
		if authenticated {
			return htxSwapAuth
		}
		return htxSwapUnAuth
	}
	if authenticated {
		return htxFuturesAuth
	}
	return htxFuturesUnAuth
}
