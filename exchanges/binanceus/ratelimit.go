package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// rateLimitPool is a Binance.US request budget. The endpoint limits drawing on a pool share its limiter, so their
// weights count against one budget the way Binance.US counts them
type rateLimitPool uint8

// Rate limit pools. The IP pool's limiter queues requests in turn, so a request drawing on it waits behind every market
// data request reserved before it, such as an order book synchronisation's snapshots; the cancel pool is the share of
// the IP budget kept for cancellations and the lookups a cancellation makes, which must not wait on market data
const (
	ipPool rateLimitPool = iota
	cancelPool
	orderPool
)

// rateLimitBudget is the weight a pool allows in each interval
type rateLimitBudget struct {
	interval time.Duration
	limit    int
}

// poolBudgets holds the budget of every pool, from exchangeInfo's rateLimits: REQUEST_WEIGHT 6000 a minute per IP and
// ORDERS 100 every 10 seconds per account. An order placement costs a request weight of 1 besides its order count, and
// the order pool admits at most 600 placements a minute, so the IP pool leaves that much of REQUEST_WEIGHT besides the
// cancel pool's share. ORDERS 200000 a day binds only an account placing more than 2.3 orders a second for hours, and a
// limiter spreading it evenly would slow every burst, so it is left to Binance.US, as is RAW_REQUESTS, which the weight
// limits keep requests far below
var poolBudgets = map[rateLimitPool]rateLimitBudget{
	ipPool:     {time.Minute, 4800},
	cancelPool: {time.Minute, 600},
	orderPool:  {10 * time.Second, 100},
}

// poolBursts holds the weight of the pools whose requests may go out at once, which the rest of the pool's budget
// refills over its interval, so no interval exceeds the budget. The cancel pool's burst takes the listing and
// cancellations of a cancellation of every symbol's orders
var poolBursts = map[rateLimitPool]int{
	cancelPool: 100,
}

// newPoolLimiter returns a limiter holding a pool to its budget
func newPoolLimiter(pool rateLimitPool) *rate.Limiter {
	budget := poolBudgets[pool]
	burst := poolBursts[pool]
	if burst <= 1 {
		return request.NewRateLimit(budget.interval, budget.limit)
	}
	return rate.NewLimiter(rate.Limit(float64(budget.limit-burst)/budget.interval.Seconds()), burst)
}

// Endpoint limits, by documentation section. They start after request.UnAuth, so a stray request.Unset, request.Auth or
// request.UnAuth finds no Binance.US limiter instead of silently using one. An endpoint whose weight depends on a
// parameter has a limit per tier, picked by a helper below. The WebSocket API's requests draw on the same limits as
// their REST equivalents
const (
	defaultRate request.EndpointLimit = request.UnAuth + 1 + iota
	cancelRate

	// General and market data
	exchangeInfoRate
	tradesRate
	aggregatedTradesRate
	orderbookDepth100Rate
	orderbookDepth500Rate
	orderbookDepth1000Rate
	orderbookDepth5000Rate
	symbolsTickerAllRate
	ticker24Hour20Rate
	ticker24Hour100Rate
	ticker24HourAllRate
	rollingWindowTicker1Rate
	rollingWindowTicker5Rate
	rollingWindowTicker10Rate
	rollingWindowTicker25Rate
	rollingWindowTickerMaxRate
	tradingDayTicker1Rate
	tradingDayTicker5Rate
	tradingDayTicker10Rate
	tradingDayTicker25Rate
	tradingDayTickerMaxRate

	// User account data
	accountInformationRate
	accountCommissionRate
	myFiltersRate

	// Trade orders
	orderRateLimitRate
	orderRate
	oneCancelsTheOtherOrderRate
	queryOrderRate
	openOrdersRate
	openOrdersAllRate
	cancelAllListingRate
	cancelLookupRate
	accountTradesRate
	accountTradesByOrderIDRate
	preventedMatchesRate
	preventedMatchesByOrderIDRate
	allOrdersRate
	queryOrderListRate
	allOrderListsRate
	openOrderListsRate

	// Wallet, convert dust and credit line endpoints heavier than 1
	convertDustRate
	creditLineAccountRate

	// WebSocket API connections and user data stream and session requests
	websocketAPIConnectRate
	userDataStreamSubscriptionRate

	// Stream subscriptions and pings, which Binance.US limits per connection
	wsConnectionMessageRate
)

// endpointCost is the pool an endpoint limit draws on and the weight each request costs
type endpointCost struct {
	pool   rateLimitPool
	weight request.Weight
}

// endpointCosts holds the documented weight of every endpoint limit except wsConnectionMessageRate. Order placements
// cost their order count on the order pool, which binds before their request weight of 1 does; an OCO counts as two
// orders
var endpointCosts = map[request.EndpointLimit]endpointCost{
	defaultRate: {ipPool, 1},
	cancelRate:  {cancelPool, 1},

	exchangeInfoRate:           {ipPool, 20},
	tradesRate:                 {ipPool, 25},
	aggregatedTradesRate:       {ipPool, 4},
	orderbookDepth100Rate:      {ipPool, 5},
	orderbookDepth500Rate:      {ipPool, 25},
	orderbookDepth1000Rate:     {ipPool, 50},
	orderbookDepth5000Rate:     {ipPool, 250},
	symbolsTickerAllRate:       {ipPool, 2},
	ticker24Hour20Rate:         {ipPool, 1},
	ticker24Hour100Rate:        {ipPool, 20},
	ticker24HourAllRate:        {ipPool, 40},
	rollingWindowTicker1Rate:   {ipPool, 2},
	rollingWindowTicker5Rate:   {ipPool, 10},
	rollingWindowTicker10Rate:  {ipPool, 20},
	rollingWindowTicker25Rate:  {ipPool, 50},
	rollingWindowTickerMaxRate: {ipPool, 100},
	tradingDayTicker1Rate:      {ipPool, 4},
	tradingDayTicker5Rate:      {ipPool, 20},
	tradingDayTicker10Rate:     {ipPool, 40},
	tradingDayTicker25Rate:     {ipPool, 100},
	tradingDayTickerMaxRate:    {ipPool, 200},

	accountInformationRate: {ipPool, 10},
	accountCommissionRate:  {ipPool, 20},
	myFiltersRate:          {ipPool, 40},

	orderRateLimitRate:            {ipPool, 20},
	orderRate:                     {orderPool, 1},
	oneCancelsTheOtherOrderRate:   {orderPool, 2},
	queryOrderRate:                {ipPool, 2},
	openOrdersRate:                {ipPool, 3},
	openOrdersAllRate:             {ipPool, 40},
	cancelAllListingRate:          {cancelPool, 40},
	cancelLookupRate:              {cancelPool, 2},
	accountTradesRate:             {ipPool, 20},
	accountTradesByOrderIDRate:    {ipPool, 5},
	preventedMatchesRate:          {ipPool, 1},
	preventedMatchesByOrderIDRate: {ipPool, 10},
	allOrdersRate:                 {ipPool, 10},
	queryOrderListRate:            {ipPool, 2},
	allOrderListsRate:             {ipPool, 10},
	openOrderListsRate:            {ipPool, 3},

	convertDustRate:       {ipPool, 10},
	creditLineAccountRate: {ipPool, 10},

	websocketAPIConnectRate:        {ipPool, 2},
	userDataStreamSubscriptionRate: {ipPool, 2},
}

// GetRateLimits returns Binance.US's endpoint limits, each with its pool's limiter and its weight
func GetRateLimits() request.RateLimitDefinitions {
	limiters := make(map[rateLimitPool]*rate.Limiter, len(poolBudgets))
	definitions := make(request.RateLimitDefinitions, len(endpointCosts)+1)
	for epl, cost := range endpointCosts {
		if limiters[cost.pool] == nil {
			limiters[cost.pool] = newPoolLimiter(cost.pool)
		}
		definitions[epl] = request.GetRateLimiterWithWeight(limiters[cost.pool], cost.weight)
	}
	// A nil limiter makes a websocket connection fall back to its own
	definitions[wsConnectionMessageRate] = request.RateLimitNotRequired
	return definitions
}

// orderbookLimit returns the limit of an order book request for its depth; 0 requests the default of 100
func orderbookLimit(depth uint64) request.EndpointLimit {
	switch {
	case depth <= 100:
		return orderbookDepth100Rate
	case depth <= 500:
		return orderbookDepth500Rate
	case depth <= 1000:
		return orderbookDepth1000Rate
	default:
		return orderbookDepth5000Rate
	}
}

// symbolsTickerLimit returns the limit of a price or book ticker request, which costs 1 for one symbol and 2 for several
// or every symbol
func symbolsTickerLimit(symbols int) request.EndpointLimit {
	if symbols == 1 {
		return defaultRate
	}
	return symbolsTickerAllRate
}

// ticker24HourLimit returns the limit of a 24hr ticker request for its number of symbols; none requests every symbol
func ticker24HourLimit(symbols int) request.EndpointLimit {
	switch {
	case symbols == 0, symbols > 100:
		return ticker24HourAllRate
	case symbols > 20:
		return ticker24Hour100Rate
	default:
		return ticker24Hour20Rate
	}
}

// rollingWindowTickerLimit returns the limit of a rolling window ticker request, which costs 2 for each symbol and 100
// from 50 symbols on. A limit has a fixed weight, so each tier charges for its largest symbol count
func rollingWindowTickerLimit(symbols int) request.EndpointLimit {
	switch {
	case symbols <= 1:
		return rollingWindowTicker1Rate
	case symbols <= 5:
		return rollingWindowTicker5Rate
	case symbols <= 10:
		return rollingWindowTicker10Rate
	case symbols <= 25:
		return rollingWindowTicker25Rate
	default:
		return rollingWindowTickerMaxRate
	}
}

// tradingDayTickerLimit returns the limit of a trading day ticker request, which costs 4 for each symbol and 200 from
// 50 symbols on. A limit has a fixed weight, so each tier charges for its largest symbol count
func tradingDayTickerLimit(symbols int) request.EndpointLimit {
	switch {
	case symbols <= 1:
		return tradingDayTicker1Rate
	case symbols <= 5:
		return tradingDayTicker5Rate
	case symbols <= 10:
		return tradingDayTicker10Rate
	case symbols <= 25:
		return tradingDayTicker25Rate
	default:
		return tradingDayTickerMaxRate
	}
}

// accountTradesLimit returns the limit of an account trade list query, which costs less for one order
func accountTradesLimit(orderID uint64) request.EndpointLimit {
	if orderID != 0 {
		return accountTradesByOrderIDRate
	}
	return accountTradesRate
}

// preventedMatchesLimit returns the limit of a prevented matches query, which costs more by order ID
func preventedMatchesLimit(orderID uint64) request.EndpointLimit {
	if orderID != 0 {
		return preventedMatchesByOrderIDRate
	}
	return preventedMatchesRate
}
