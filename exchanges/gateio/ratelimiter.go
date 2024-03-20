package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GateIO endpoints limits.
const (
	spotDefaultEPL request.EndpointLimit = iota
	spotPrivateEPL
	spotPlaceOrdersEPL
	spotCancelOrdersEPL
	perpetualSwapDefaultEPL
	perpetualSwapPlaceOrdersEPL
	perpetualSwapPrivateEPL
	perpetualSwapCancelOrdersEPL
	walletEPL
	withdrawalEPL

	// Request rates per interval

	spotPublicRate                = 900
	spotPrivateRate               = 900
	spotPlaceOrdersRate           = 10
	spotCancelOrdersRate          = 500
	perpetualSwapPublicRate       = 300
	perpetualSwapPlaceOrdersRate  = 100
	perpetualSwapPrivateRate      = 400
	perpetualSwapCancelOrdersRate = 400
	walletRate                    = 200
	withdrawalRate                = 1

	// interval
	oneSecondInterval    = time.Second
	threeSecondsInterval = time.Second * 3
)

// SetRateLimit returns the rate limiter for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		spotDefaultEPL:               request.NewRateLimitWithToken(oneSecondInterval, spotPublicRate, 1),
		spotPrivateEPL:               request.NewRateLimitWithToken(oneSecondInterval, spotPrivateRate, 1),
		spotPlaceOrdersEPL:           request.NewRateLimitWithToken(oneSecondInterval, spotPlaceOrdersRate, 1),
		spotCancelOrdersEPL:          request.NewRateLimitWithToken(oneSecondInterval, spotCancelOrdersRate, 1),
		perpetualSwapDefaultEPL:      request.NewRateLimitWithToken(oneSecondInterval, perpetualSwapPublicRate, 1),
		perpetualSwapPlaceOrdersEPL:  request.NewRateLimitWithToken(oneSecondInterval, perpetualSwapPlaceOrdersRate, 1),
		perpetualSwapPrivateEPL:      request.NewRateLimitWithToken(oneSecondInterval, perpetualSwapPrivateRate, 1),
		perpetualSwapCancelOrdersEPL: request.NewRateLimitWithToken(oneSecondInterval, perpetualSwapCancelOrdersRate, 1),
		walletEPL:                    request.NewRateLimitWithToken(oneSecondInterval, walletRate, 1),
		withdrawalEPL:                request.NewRateLimitWithToken(threeSecondsInterval, withdrawalRate, 1),
	}
}
