package gateio

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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

// RateLimitter represents a rate limiter structure for gateIO endpoints.
type RateLimitter struct {
	SpotDefault               *rate.Limiter
	SpotPrivate               *rate.Limiter
	SpotPlaceOrders           *rate.Limiter
	SpotCancelOrders          *rate.Limiter
	PerpetualSwapDefault      *rate.Limiter
	PerpetualSwapPlaceOrders  *rate.Limiter
	PerpetualSwapPrivate      *rate.Limiter
	PerpetualSwapCancelOrders *rate.Limiter
	Wallet                    *rate.Limiter
	Withdrawal                *rate.Limiter
}

// Limit executes rate limiting functionality
// implements the request.Limiter interface
func (r *RateLimitter) Limit(ctx context.Context, epl request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch epl {
	case spotDefaultEPL:
		limiter, tokens = r.SpotDefault, 1
	case spotPrivateEPL:
		return r.SpotPrivate.Wait(ctx)
	case spotPlaceOrdersEPL:
		return r.SpotPlaceOrders.Wait(ctx)
	case spotCancelOrdersEPL:
		return r.SpotCancelOrders.Wait(ctx)
	case perpetualSwapDefaultEPL:
		limiter, tokens = r.PerpetualSwapDefault, 1
	case perpetualSwapPlaceOrdersEPL:
		return r.PerpetualSwapPlaceOrders.Wait(ctx)
	case perpetualSwapPrivateEPL:
		return r.PerpetualSwapPrivate.Wait(ctx)
	case perpetualSwapCancelOrdersEPL:
		return r.PerpetualSwapCancelOrders.Wait(ctx)
	case walletEPL:
		return r.Wallet.Wait(ctx)
	case withdrawalEPL:
		return r.Withdrawal.Wait(ctx)
	default:
	}
	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}
	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		for x := range reserves {
			reserves[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	time.Sleep(finalDelay)
	return nil
}

// SetRateLimit returns the rate limiter for the exchange
func SetRateLimit() *RateLimitter {
	return &RateLimitter{
		SpotDefault:               request.NewRateLimit(oneSecondInterval, spotPublicRate),
		SpotPrivate:               request.NewRateLimit(oneSecondInterval, spotPrivateRate),
		SpotPlaceOrders:           request.NewRateLimit(oneSecondInterval, spotPlaceOrdersRate),
		SpotCancelOrders:          request.NewRateLimit(oneSecondInterval, spotCancelOrdersRate),
		PerpetualSwapDefault:      request.NewRateLimit(oneSecondInterval, perpetualSwapPublicRate),
		PerpetualSwapPlaceOrders:  request.NewRateLimit(oneSecondInterval, perpetualSwapPlaceOrdersRate),
		PerpetualSwapPrivate:      request.NewRateLimit(oneSecondInterval, perpetualSwapPrivateRate),
		PerpetualSwapCancelOrders: request.NewRateLimit(oneSecondInterval, perpetualSwapCancelOrdersRate),
		Wallet:                    request.NewRateLimit(oneSecondInterval, walletRate),
		Withdrawal:                request.NewRateLimit(threeSecondsInterval, withdrawalRate),
	}
}
