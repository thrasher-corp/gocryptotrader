package coinmarketcap

import "github.com/thrasher-corp/gocryptotrader/exchanges/request"

const (
	basicEPL request.EndpointLimit = iota
	builderEPL
	startupEPL
	growthEPL
	professionalEPL
	enterpriseEPL
)

func getRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		basicEPL:        request.NewRateLimitWithWeight(rateInterval, basicRequestRate, 1),
		builderEPL:      request.NewRateLimitWithWeight(rateInterval, builderRequestRate, 1),
		startupEPL:      request.NewRateLimitWithWeight(rateInterval, startupRequestRate, 1),
		growthEPL:       request.NewRateLimitWithWeight(rateInterval, growthRequestRate, 1),
		professionalEPL: request.NewRateLimitWithWeight(rateInterval, professionalRequestRate, 1),
		enterpriseEPL:   request.NewRateLimitWithWeight(rateInterval, enterpriseRequestRate, 1),
	}
}

func planRateLimit(plan uint8) request.EndpointLimit {
	switch plan {
	case Builder:
		return builderEPL
	case Startup:
		return startupEPL
	case Growth:
		return growthEPL
	case Professional:
		return professionalEPL
	case Enterprise:
		return enterpriseEPL
	default:
		return basicEPL
	}
}
