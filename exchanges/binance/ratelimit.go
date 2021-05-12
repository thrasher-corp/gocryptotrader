package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	spotInterval    = time.Minute
	spotRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval        = 10 * time.Second
	spotOrderRequestRate     = 100
	cFuturesInterval         = time.Minute
	cFuturesRequestRate      = 6000
	cFuturesOrderInterval    = time.Minute
	cFuturesOrderRequestRate = 1200
	uFuturesInterval         = time.Minute
	uFuturesRequestRate      = 2400
	uFuturesOrderInterval    = time.Minute
	uFuturesOrderRequestRate = 1200
)

// Binance Spot rate limits
const (
	spotDefaultRate request.EndpointLimit = iota
	spotExchangeInfo
	spotHistoricalTradesRate
	spotOrderbookDepth500Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	spotOrderbookTickerAllRate
	spotPriceChangeAllRate
	spotSymbolPriceAllRate
	spotOpenOrdersAllRate
	spotOpenOrdersSpecificRate
	spotOrderRate
	spotOrderQueryRate
	spotAllOrdersRate
	spotAccountInformationRate
	uFuturesDefaultRate
	uFuturesHistoricalTradesRate
	uFuturesSymbolOrdersRate
	uFuturesPairOrdersRate
	uFuturesCurrencyForceOrdersRate
	uFuturesAllForceOrdersRate
	uFuturesIncomeHistoryRate
	uFuturesOrderbook50Rate
	uFuturesOrderbook100Rate
	uFuturesOrderbook500Rate
	uFuturesOrderbook1000Rate
	uFuturesKline100Rate
	uFuturesKline500Rate
	uFuturesKline1000Rate
	uFuturesKlineMaxRate
	uFuturesTickerPriceHistoryRate
	uFuturesOrdersDefaultRate
	uFuturesGetAllOrdersRate
	uFuturesAccountInformationRate
	uFuturesOrderbookTickerAllRate
	uFuturesCountdownCancelRate
	uFuturesBatchOrdersRate
	uFuturesGetAllOpenOrdersRate
	cFuturesDefaultRate
	cFuturesHistoricalTradesRate
	cFuturesTickerPriceHistoryRate
	cFuturesIncomeHistoryRate
	cFuturesOrderbook50Rate
	cFuturesOrderbook100Rate
	cFuturesOrderbook500Rate
	cFuturesOrderbook1000Rate
	cFuturesKline100Rate
	cFuturesKline500Rate
	cFuturesKline1000Rate
	cFuturesKlineMaxRate
	cFuturesIndexMarkPriceRate
	cFuturesBatchOrdersRate
	cFuturesCancelAllOrdersRate
	cFuturesGetAllOpenOrdersRate
	cFuturesAllForceOrdersRate
	cFuturesCurrencyForceOrdersRate
	cFuturesPairOrdersRate
	cFuturesSymbolOrdersRate
	cFuturesAccountInformationRate
	cFuturesOrderbookTickerAllRate
	cFuturesOrdersDefaultRate
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate           *rate.Limiter
	SpotOrdersRate     *rate.Limiter
	UFuturesRate       *rate.Limiter
	UFuturesOrdersRate *rate.Limiter
	CFuturesRate       *rate.Limiter
	CFuturesOrdersRate *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case spotDefaultRate:
		limiter, tokens = r.SpotRate, 1
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate:
		limiter, tokens = r.SpotRate, 2
	case spotHistoricalTradesRate,
		spotOrderbookDepth500Rate:
		limiter, tokens = r.SpotRate, 5
	case spotOrderbookDepth1000Rate,
		spotAccountInformationRate,
		spotExchangeInfo:
		limiter, tokens = r.SpotRate, 10
	case spotPriceChangeAllRate:
		limiter, tokens = r.SpotRate, 40
	case spotOrderbookDepth5000Rate:
		limiter, tokens = r.SpotRate, 50
	case spotOrderRate:
		limiter, tokens = r.SpotOrdersRate, 1
	case spotOrderQueryRate:
		limiter, tokens = r.SpotOrdersRate, 2
	case spotOpenOrdersSpecificRate:
		limiter, tokens = r.SpotOrdersRate, 3
	case spotAllOrdersRate:
		limiter, tokens = r.SpotOrdersRate, 10
	case spotOpenOrdersAllRate:
		limiter, tokens = r.SpotOrdersRate, 40
	case uFuturesDefaultRate,
		uFuturesKline100Rate:
		limiter, tokens = r.UFuturesRate, 1
	case uFuturesOrderbook50Rate,
		uFuturesKline500Rate,
		uFuturesOrderbookTickerAllRate:
		limiter, tokens = r.UFuturesRate, 2
	case uFuturesOrderbook100Rate,
		uFuturesKline1000Rate,
		uFuturesAccountInformationRate:
		limiter, tokens = r.UFuturesRate, 5
	case uFuturesOrderbook500Rate,
		uFuturesKlineMaxRate:
		limiter, tokens = r.UFuturesRate, 10
	case uFuturesOrderbook1000Rate,
		uFuturesHistoricalTradesRate:
		limiter, tokens = r.UFuturesRate, 20
	case uFuturesTickerPriceHistoryRate:
		limiter, tokens = r.UFuturesRate, 40
	case uFuturesOrdersDefaultRate:
		limiter, tokens = r.UFuturesOrdersRate, 1
	case uFuturesBatchOrdersRate,
		uFuturesGetAllOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 5
	case uFuturesCountdownCancelRate:
		limiter, tokens = r.UFuturesOrdersRate, 10
	case uFuturesCurrencyForceOrdersRate,
		uFuturesSymbolOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 20
	case uFuturesIncomeHistoryRate:
		limiter, tokens = r.UFuturesOrdersRate, 30
	case uFuturesPairOrdersRate,
		uFuturesGetAllOpenOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 40
	case uFuturesAllForceOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 50
	case cFuturesKline100Rate:
		limiter, tokens = r.CFuturesRate, 1
	case cFuturesKline500Rate,
		cFuturesOrderbookTickerAllRate:
		limiter, tokens = r.CFuturesRate, 2
	case cFuturesKline1000Rate,
		cFuturesAccountInformationRate:
		limiter, tokens = r.CFuturesRate, 5
	case cFuturesKlineMaxRate,
		cFuturesIndexMarkPriceRate:
		limiter, tokens = r.CFuturesRate, 10
	case cFuturesHistoricalTradesRate,
		cFuturesCurrencyForceOrdersRate:
		limiter, tokens = r.CFuturesRate, 20
	case cFuturesTickerPriceHistoryRate:
		limiter, tokens = r.CFuturesRate, 40
	case cFuturesAllForceOrdersRate:
		limiter, tokens = r.CFuturesRate, 50
	case cFuturesOrdersDefaultRate:
		limiter, tokens = r.CFuturesOrdersRate, 1
	case cFuturesBatchOrdersRate,
		cFuturesGetAllOpenOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 5
	case cFuturesCancelAllOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 10
	case cFuturesIncomeHistoryRate,
		cFuturesSymbolOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 20
	case cFuturesPairOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 40
	case cFuturesOrderbook50Rate:
		limiter, tokens = r.CFuturesRate, 2
	case cFuturesOrderbook100Rate:
		limiter, tokens = r.CFuturesRate, 5
	case cFuturesOrderbook500Rate:
		limiter, tokens = r.CFuturesRate, 10
	case cFuturesOrderbook1000Rate:
		limiter, tokens = r.CFuturesRate, 20
	case cFuturesDefaultRate:
		limiter, tokens = r.CFuturesRate, 1
	default:
		limiter, tokens = r.SpotRate, 1
	}

	var finalDelay time.Duration
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		finalDelay = limiter.Reserve().Delay()
	}
	time.Sleep(finalDelay)
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		SpotRate:           request.NewRateLimit(spotInterval, spotRequestRate),
		SpotOrdersRate:     request.NewRateLimit(spotOrderInterval, spotOrderRequestRate),
		UFuturesRate:       request.NewRateLimit(uFuturesInterval, uFuturesRequestRate),
		UFuturesOrdersRate: request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate),
		CFuturesRate:       request.NewRateLimit(cFuturesInterval, cFuturesRequestRate),
		CFuturesOrdersRate: request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate),
	}
}

func bestPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOrderbookTickerAllRate
	}

	return spotDefaultRate
}

func openOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOpenOrdersAllRate
	}

	return spotOpenOrdersSpecificRate
}

func orderbookLimit(depth int) request.EndpointLimit {
	switch {
	case depth <= 100:
		return spotDefaultRate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}

	return spotOrderbookDepth5000Rate
}
