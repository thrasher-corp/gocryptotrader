package deribit

import (
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
)

// GetRateLimits returns Deribit's default rate limits, which sub-accounts with more trading volume exceed
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		nonMatchingEPL:       request.NewRateLimitWithWeight(time.Second, nonMatchingRate, 1),
		matchingEPL:          request.NewRateLimitWithWeight(time.Second, matchingRate, 1),
		getInstrumentsEPL:    request.NewRateLimitWithWeight(time.Second, getInstrumentsRate, 1),
		subscribeEPL:         request.NewRateLimitWithWeight(subscribeInterval, subscribeRate, 1),
		movePositionsEPL:     request.NewRateLimitWithWeight(time.Minute, movePositionsRate, 1),
		transactionLogEPL:    request.NewRateLimitWithWeight(time.Second, transactionLogRate, 1),
		massQuoteEPL:         request.NewRateLimitWithWeight(time.Second, massQuoteRate, 1),
		simulatePortfolioEPL: request.NewRateLimitWithWeight(time.Second, simulatePortfolioRate, 1),
	}
}
