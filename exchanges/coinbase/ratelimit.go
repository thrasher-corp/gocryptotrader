package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Rate limit keys
const (
	// advancedTradePrivateEPL and advancedTradePublicEPL limit the Advanced Trade API's authenticated requests, per API
	// key, and its public ones, per IP
	advancedTradePrivateEPL request.EndpointLimit = iota + 1
	advancedTradePublicEPL
	// appEPL limits the Coinbase App API's requests, per API key or OAuth user
	appEPL
	// exchangeAPIPublicEPL limits Coinbase Exchange's public requests, per IP
	exchangeAPIPublicEPL
	// The Global Derivatives gateway runs Deribit's credit system: one limit for matching engine requests (orders), one
	// for the rest, and their own for a few costly methods
	derivativesNonMatchingEPL
	derivativesMatchingEPL
	derivativesGetInstrumentsEPL
	derivativesSubscribeEPL
	derivativesTransactionLogEPL
	derivativesSimulatePortfolioEPL
	// websocketConnectEPL limits connecting to the Advanced Trade websocket, and websocketMessageEPL the messages sent
	// on it, both per IP
	websocketConnectEPL
	websocketMessageEPL
)

const (
	// The Advanced Trade docs no longer state its REST limits; they were 30 private requests a second per key and 10
	// public ones a second per IP, so private requests keep a margin under that
	advancedTradePrivateRate = 27
	advancedTradePublicRate  = 10
	// The Coinbase App API allows 10,000 requests an hour per key
	appRate = 10000
	// Coinbase Exchange allows 10 public requests a second per IP
	exchangeAPIPublicRate = 10
	// The gateway's default credits refill 20 non-matching and 5 matching engine requests a second, one
	// public/get_instruments, private/get_transaction_log and private/simulate_portfolio request a second, and 10
	// subscriptions every 3 seconds; accounts with more trading volume get more
	derivativesNonMatchingRate       = 20
	derivativesMatchingRate          = 5
	derivativesGetInstrumentsRate    = 1
	derivativesTransactionLogRate    = 1
	derivativesSimulatePortfolioRate = 1
	derivativesSubscribeInterval     = 3 * time.Second
	derivativesSubscribeRate         = 10
	// The Advanced Trade websocket allows 8 connections and 8 unauthenticated messages a second per IP
	websocketConnectRate = 8
	websocketMessageRate = 8
)

// GetRateLimits returns Coinbase's rate limits
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		advancedTradePrivateEPL:         request.NewRateLimitWithWeight(time.Second, advancedTradePrivateRate, 1),
		advancedTradePublicEPL:          request.NewRateLimitWithWeight(time.Second, advancedTradePublicRate, 1),
		appEPL:                          request.NewRateLimitWithWeight(time.Hour, appRate, 1),
		exchangeAPIPublicEPL:            request.NewRateLimitWithWeight(time.Second, exchangeAPIPublicRate, 1),
		derivativesNonMatchingEPL:       request.NewRateLimitWithWeight(time.Second, derivativesNonMatchingRate, 1),
		derivativesMatchingEPL:          request.NewRateLimitWithWeight(time.Second, derivativesMatchingRate, 1),
		derivativesGetInstrumentsEPL:    request.NewRateLimitWithWeight(time.Second, derivativesGetInstrumentsRate, 1),
		derivativesSubscribeEPL:         request.NewRateLimitWithWeight(derivativesSubscribeInterval, derivativesSubscribeRate, 1),
		derivativesTransactionLogEPL:    request.NewRateLimitWithWeight(time.Second, derivativesTransactionLogRate, 1),
		derivativesSimulatePortfolioEPL: request.NewRateLimitWithWeight(time.Second, derivativesSimulatePortfolioRate, 1),
		websocketConnectEPL:             request.NewRateLimitWithWeight(time.Second, websocketConnectRate, 1),
		websocketMessageEPL:             request.NewRateLimitWithWeight(time.Second, websocketMessageRate, 1),
	}
}
