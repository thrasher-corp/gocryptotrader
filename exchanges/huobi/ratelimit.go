package huobi

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// Huobi rate limits per API Key
	huobiSpotRateInterval = time.Second * 1
	huobiSpotRequestRate  = 7

	huobiFuturesRateInterval    = time.Second * 3
	huobiFuturesAuthRequestRate = 30
	// Non market-request public interface rate
	huobiFuturesUnAuthRequestRate    = 60
	huobiFuturesTransferRateInterval = time.Second * 3
	huobiFuturesTransferReqRate      = 10

	huobiSwapRateInterval      = time.Second * 3
	huobiSwapAuthRequestRate   = 30
	huobiSwapUnauthRequestRate = 60

	huobiFuturesAuth request.EndpointLimit = iota
	huobiFuturesUnAuth
	huobiFuturesTransfer
	huobiSwapAuth
	huobiSwapUnauth
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Spot          *rate.Limiter
	FuturesAuth   *rate.Limiter
	FuturesUnauth *rate.Limiter
	SwapAuth      *rate.Limiter
	SwapUnauth    *rate.Limiter
	FuturesXfer   *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, request.Tokens, error) {
	switch f {
	// TODO: Add futures and swap functionality
	case huobiFuturesAuth:
		return r.FuturesAuth, 1, nil
	case huobiFuturesUnAuth:
		return r.FuturesUnauth, 1, nil
	case huobiFuturesTransfer:
		return r.FuturesXfer, 1, nil
	case huobiSwapAuth:
		return r.SwapAuth, 1, nil
	case huobiSwapUnauth:
		return r.SwapUnauth, 1, nil
	default:
		// Spot calls
		return r.Spot, 1, nil
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Spot:          request.NewRateLimit(huobiSpotRateInterval, huobiSpotRequestRate),
		FuturesAuth:   request.NewRateLimit(huobiFuturesRateInterval, huobiFuturesAuthRequestRate),
		FuturesUnauth: request.NewRateLimit(huobiFuturesRateInterval, huobiFuturesUnAuthRequestRate),
		SwapAuth:      request.NewRateLimit(huobiSwapRateInterval, huobiSwapAuthRequestRate),
		SwapUnauth:    request.NewRateLimit(huobiSwapRateInterval, huobiSwapUnauthRequestRate),
		FuturesXfer:   request.NewRateLimit(huobiFuturesTransferRateInterval, huobiFuturesTransferReqRate),
	}
}
