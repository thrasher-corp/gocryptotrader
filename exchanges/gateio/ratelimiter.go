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

// GetRateLimit returns the rate limiter for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		spotDefaultEPL:               request.NewRateLimitWithWeight(oneSecondInterval, spotPublicRate, 1),
		spotPrivateEPL:               request.NewRateLimitWithWeight(oneSecondInterval, spotPrivateRate, 1),
		spotPlaceOrdersEPL:           request.NewRateLimitWithWeight(oneSecondInterval, spotPlaceOrdersRate, 1),
		spotCancelOrdersEPL:          request.NewRateLimitWithWeight(oneSecondInterval, spotCancelOrdersRate, 1),
		perpetualSwapDefaultEPL:      request.NewRateLimitWithWeight(oneSecondInterval, perpetualSwapPublicRate, 1),
		perpetualSwapPlaceOrdersEPL:  request.NewRateLimitWithWeight(oneSecondInterval, perpetualSwapPlaceOrdersRate, 1),
		perpetualSwapPrivateEPL:      request.NewRateLimitWithWeight(oneSecondInterval, perpetualSwapPrivateRate, 1),
		perpetualSwapCancelOrdersEPL: request.NewRateLimitWithWeight(oneSecondInterval, perpetualSwapCancelOrdersRate, 1),
		walletEPL:                    request.NewRateLimitWithWeight(oneSecondInterval, walletRate, 1),
		withdrawalEPL:                request.NewRateLimitWithWeight(threeSecondsInterval, withdrawalRate, 1),
	}
}
