package kraken

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Kraken applies two independent REST rate-limit systems:
//
//  1. Public API limit, applied per source IP. Covers unauthenticated
//     endpoints (Ticker, Depth, OHLC, Trades, Time, Assets, AssetPairs).
//
//  2. Private API limit, applied per account / API key. The private limit
//     is a stateful counter with continuous decay; each request increments
//     the counter, the counter decays at a fixed rate per second, and
//     exceeding the maximum triggers a temporary key ban.
//
// Within the private system trading endpoints (AddOrder, CancelOrder) are
// tracked under a separate per-pair counter that does not consume from the
// general private counter.
//
// Reference (verified 2026): https://support.kraken.com/articles/206548367
//
// Tiers for the private general counter:
//
//	Verified          : max 20, decay -0.5/sec
//	Verified Pro      : max 20, decay -1.0/sec
//
// Endpoint costs (private REST, general counter):
//
//	Account history (Ledgers, TradesHistory, ClosedOrders, QueryOrders,
//	                 QueryTrades, QueryLedgers)                        +4
//	Balance, TradeBalance, OpenOrders, OpenPositions, TradeVolume,
//	  WithdrawInfo, WithdrawStatus, DepositMethods, DepositAddresses,
//	  GetWebSocketsToken                                                +1
//	Trading (AddOrder, CancelOrder)                                     0
//	  (separate per-pair counter)
//	Withdraw, WithdrawCancel                                            +1
//
// We emulate each system with a separate golang.org/x/time/rate.Limiter
// where:
//
//	Kraken counter max  ↔  rate.Limiter burst
//	Kraken decay /sec   ↔  rate.Limiter refill rate
//
// Tunables. These are package-level vars rather than consts so they can be
// overridden at process start-up (for example to switch the private counter
// to the Verified Pro tier without rebuilding, or to relax limits for
// integration tests). They are not safe to mutate after the Requester has
// been constructed.
var (
	// Private general counter (Verified tier defaults).
	KrakenSpotMaxCounter  = 20  // documented maximum counter
	KrakenSpotDecayPerSec = 0.5 // tokens per second (Verified tier)

	// Private trading counter (AddOrder, CancelOrder).
	KrakenSpotOrderMaxBurst = 60
	KrakenSpotOrderRate     = 1.0 // tokens per second

	// Public API counter (per IP). Conservative defaults; Kraken
	// documents a generous public rate that varies by endpoint, but a
	// modest steady-state with healthy burst suits both bots and
	// occasional consumers.
	KrakenPublicMaxBurst = 15
	KrakenPublicRate     = 1.0 // tokens per second
)

// Kraken-specific EndpointLimit values. These extend the global Unset/Auth/
// UnAuth constants from the request package and let each endpoint declare its
// real cost so the limiter charges the correct number of tokens per call.
const (
	// krakenLimitDefault is used for unrecognised endpoints; weight 1 keeps
	// it conservative.
	krakenLimitDefault request.EndpointLimit = iota + 100

	// krakenLimitPublic — public REST endpoints (Time, Depth, Ticker,
	// Trades, OHLC, Assets, AssetPairs). Counted under the public-API
	// limit which is per-IP and independent of the private counter.
	krakenLimitPublic

	// krakenLimitFuturesPublic — public futures endpoints. Kraken documents
	// these as having no request cost, unlike authenticated futures paths.
	krakenLimitFuturesPublic

	// krakenLimitBalance — Balance, TradeBalance, OpenOrders,
	// OpenPositions, TradeVolume, withdraw/deposit info, WS token. +1
	// on the private general counter.
	krakenLimitBalance

	// krakenLimitHistory — Ledgers, TradesHistory, ClosedOrders,
	// QueryOrders, QueryTrades, QueryLedgers. +4 on the private general
	// counter; these are the expensive endpoints.
	krakenLimitHistory

	// krakenLimitTrading — AddOrder, CancelOrder. Counted under a
	// separate per-pair Kraken limiter, not on the general counter.
	krakenLimitTrading

	// krakenLimitWithdraw — Withdraw, WithdrawCancel. +1 on the private
	// general counter.
	krakenLimitWithdraw
)

// newKrakenSpotPrivateLimiter constructs the underlying rate.Limiter that
// emulates Kraken's private general counter (Verified tier defaults).
func newKrakenSpotPrivateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(KrakenSpotDecayPerSec), KrakenSpotMaxCounter)
}

// newKrakenSpotOrderLimiter constructs the underlying rate.Limiter for
// AddOrder/CancelOrder, which Kraken tracks under a separate per-pair
// counter independent of the private general counter.
func newKrakenSpotOrderLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(KrakenSpotOrderRate), KrakenSpotOrderMaxBurst)
}

// newKrakenPublicLimiter constructs the underlying rate.Limiter for the
// public REST API, which Kraken rate-limits per source IP independently of
// any private API key quota.
func newKrakenPublicLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(KrakenPublicRate), KrakenPublicMaxBurst)
}

// buildKrakenRateLimits returns the full RateLimitDefinitions map.
//
// Three independent underlying rate.Limiter instances are used:
//
//   - private: shared by all authenticated non-trading endpoints
//     (balance, info, history, withdraw). Different endpoint groups carry
//     different weights matching Kraken's documented costs.
//   - orders:  used only by AddOrder/CancelOrder.
//   - public:  used only by unauthenticated REST endpoints. Independent of
//     the private counter because Kraken's public limit is enforced per
//     IP, not per API key.
//
// The legacy request.Unset/Auth/UnAuth keys are kept and routed to
// `private` for backward compatibility — anything that still passes them
// will behave as before, just with the new tier-correct parameters.
func buildKrakenRateLimits() request.RateLimitDefinitions {
	private := newKrakenSpotPrivateLimiter()
	orders := newKrakenSpotOrderLimiter()
	public := newKrakenPublicLimiter()

	return request.RateLimitDefinitions{
		// Backward compatibility — anything still passing the global
		// constants gets the private limiter at weight 1.
		request.Unset:  request.GetRateLimiterWithWeight(private, 1),
		request.Auth:   request.GetRateLimiterWithWeight(private, 1),
		request.UnAuth: request.GetRateLimiterWithWeight(public, 1),

		// Kraken-specific endpoint costs.
		krakenLimitDefault:       request.GetRateLimiterWithWeight(private, 1),
		krakenLimitPublic:        request.GetRateLimiterWithWeight(public, 1),
		krakenLimitFuturesPublic: request.NewRateLimitWithWeight(0, 0, 1),
		krakenLimitBalance:       request.GetRateLimiterWithWeight(private, 1),
		krakenLimitHistory:       request.GetRateLimiterWithWeight(private, 4),
		krakenLimitTrading:       request.GetRateLimiterWithWeight(orders, 1),
		krakenLimitWithdraw:      request.GetRateLimiterWithWeight(private, 1),
	}
}
