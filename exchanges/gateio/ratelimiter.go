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
		spotDefaultEPL:               request.NewRateLimit(oneSecondInterval, spotPublicRate, 1),
		spotPrivateEPL:               request.NewRateLimit(oneSecondInterval, spotPrivateRate, 1),
		spotPlaceOrdersEPL:           request.NewRateLimit(oneSecondInterval, spotPlaceOrdersRate, 1),
		spotCancelOrdersEPL:          request.NewRateLimit(oneSecondInterval, spotCancelOrdersRate, 1),
		perpetualSwapDefaultEPL:      request.NewRateLimit(oneSecondInterval, perpetualSwapPublicRate, 1),
		perpetualSwapPlaceOrdersEPL:  request.NewRateLimit(oneSecondInterval, perpetualSwapPlaceOrdersRate, 1),
		perpetualSwapPrivateEPL:      request.NewRateLimit(oneSecondInterval, perpetualSwapPrivateRate, 1),
		perpetualSwapCancelOrdersEPL: request.NewRateLimit(oneSecondInterval, perpetualSwapCancelOrdersRate, 1),
		walletEPL:                    request.NewRateLimit(oneSecondInterval, walletRate, 1),
		withdrawalEPL:                request.NewRateLimit(threeSecondsInterval, withdrawalRate, 1),
	}
}
