package deribit

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Rate limit keys; Deribit counts a sub-account's requests over HTTP and the websocket against the same credits
const (
	nonMatchingEPL request.EndpointLimit = iota + 1
	matchingEPL
	getInstrumentsEPL
	subscribeEPL
	movePositionsEPL
	transactionLogEPL
	massQuoteEPL
	simulatePortfolioEPL
	tickerOrderBookEPL
)

const (
	// Non-matching engine requests refill at 20 a second by default
	nonMatchingRate = 20
	// Matching engine requests refill at 5 a second in the lowest volume tier, which a sub-account starts in
	matchingRate = 5
	// public/get_instruments, private/get_transaction_log and private/simulate_portfolio refill at one a second
	getInstrumentsRate    = 1
	transactionLogRate    = 1
	simulatePortfolioRate = 1
	// public/subscribe and private/subscribe refill at about 3.3 a second, 10 every 3 seconds
	subscribeInterval = 3 * time.Second
	subscribeRate     = 10
	// private/move_positions refills at 6 a minute
	movePositionsRate = 6
	// private/mass_quote is limited by the account's maximum_mass_quotes rate, 10 a second by default
	massQuoteRate = 10
	// public/ticker, public/get_order_book and public/get_order_book_by_instrument_id share a limit of their own, which
	// Deribit charges on top of the non-matching engine one. Deribit documents 5 a second, but after the first few it
	// accepted unauthenticated requests, which it counts per IP address, at only about one a second, so they are limited
	// to one a second
	tickerOrderBookRate = 1
)

// rateLimits holds Deribit's default rate limits, which sub-accounts with more trading volume exceed. Deribit counts
// requests per sub-account, and unauthenticated ones per IP address, not per Exchange instance, so every instance
// shares these limiters rather than building its own
var rateLimits = request.RateLimitDefinitions{
	nonMatchingEPL:       request.NewRateLimitWithWeight(time.Second, nonMatchingRate, 1),
	matchingEPL:          request.NewRateLimitWithWeight(time.Second, matchingRate, 1),
	getInstrumentsEPL:    request.NewRateLimitWithWeight(time.Second, getInstrumentsRate, 1),
	subscribeEPL:         request.NewRateLimitWithWeight(subscribeInterval, subscribeRate, 1),
	movePositionsEPL:     request.NewRateLimitWithWeight(time.Minute, movePositionsRate, 1),
	transactionLogEPL:    request.NewRateLimitWithWeight(time.Second, transactionLogRate, 1),
	massQuoteEPL:         request.NewRateLimitWithWeight(time.Second, massQuoteRate, 1),
	simulatePortfolioEPL: request.NewRateLimitWithWeight(time.Second, simulatePortfolioRate, 1),
	tickerOrderBookEPL:   request.NewRateLimitWithWeight(time.Second, tickerOrderBookRate, 1),
}

// attemptRateLimits returns the endpoint limits each attempt of a request under epl waits on, in turn. A ticker or order
// book request waits on the non-matching engine limit, which Deribit also charges it to, and then on their shared limit,
// so the stricter limit is the last wait before each attempt is sent, and an attempt it refuses spends no more than a
// non-matching engine request
func attemptRateLimits(epl request.EndpointLimit) []request.EndpointLimit {
	if epl == tickerOrderBookEPL {
		return []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}
	}
	return []request.EndpointLimit{epl}
}

// initiateRateLimits waits on endpoint limits in turn, unless the requester has no rate limits, as it then sends
// requests without them
func (e *Exchange) initiateRateLimits(ctx context.Context, limits []request.EndpointLimit) error {
	if e.Requester.GetRateLimiterDefinitions() == nil {
		return nil
	}
	for _, epl := range limits {
		if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
			return err
		}
	}
	return nil
}
