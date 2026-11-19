package binanceus

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

func TestGetRateLimits(t *testing.T) {
	t.Parallel()
	definitions := GetRateLimits()
	require.Len(t, definitions, len(endpointCosts)+1, "GetRateLimits must define each endpoint limit once")
	for name, epl := range map[string]request.EndpointLimit{"unset": request.Unset, "authenticated": request.Auth, "unauthenticated": request.UnAuth} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.NotContainsf(t, definitions, epl, "GetRateLimits should not define the generic endpoint limit %d", epl)
		})
	}

	pools := make(map[rateLimitPool]*request.RateLimiterWithWeight, len(poolBudgets))
	for epl := defaultRate; epl < wsConnectionMessageRate; epl++ {
		cost, ok := endpointCosts[epl]
		require.Truef(t, ok, "endpointCosts must hold endpoint limit %d", epl)
		require.Containsf(t, poolBudgets, cost.pool, "endpoint limit %d must draw on a pool with a budget", epl)
		definition := definitions[epl]
		require.NotNilf(t, definition, "GetRateLimits must give endpoint limit %d a limiter", epl)
		assert.Equalf(t, cost.weight, definition.Weight(), "endpoint limit %d should weigh its cost", epl)
		assert.InDeltaf(t, float64(newPoolLimiter(cost.pool).Limit()), float64(definition.Limit()), 1e-9, "endpoint limit %d should draw on its pool's budget", epl)
		if shared, ok := pools[cost.pool]; ok {
			assert.Truef(t, definition.SharesBudgetWith(shared), "endpoint limit %d should share its pool's limiter", epl)
			continue
		}
		pools[cost.pool] = definition
	}
	for pool, limiter := range pools {
		for other, otherLimiter := range pools {
			if pool != other {
				assert.Falsef(t, limiter.SharesBudgetWith(otherLimiter), "pools %d and %d should not share a limiter", pool, other)
			}
		}
	}

	require.Contains(t, definitions, wsConnectionMessageRate, "GetRateLimits must define the stream message rate")
	assert.Nil(t, definitions[wsConnectionMessageRate], "the stream message rate should have no limiter, so a stream connection uses its own")
}

func TestNewPoolLimiter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		pool  rateLimitPool
		rate  rate.Limit
		burst int
	}{
		{name: "IP", pool: ipPool, rate: rate.Limit(4800.0 / 60), burst: 1},
		{name: "cancel", pool: cancelPool, rate: rate.Limit((600.0 - 100) / 60), burst: 100},
		{name: "order", pool: orderPool, rate: rate.Limit(100.0 / 10), burst: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			limiter := newPoolLimiter(tc.pool)
			require.NotNil(t, limiter, "newPoolLimiter must return a limiter")
			assert.InDelta(t, float64(tc.rate), float64(limiter.Limit()), 1e-9, "newPoolLimiter should refill the pool's budget, less its burst, over its interval")
			assert.Equal(t, tc.burst, limiter.Burst(), "newPoolLimiter should let the pool's burst go out at once")
		})
	}

	t.Run("the IP, cancel and order pools share the request weight limit", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Minute, poolBudgets[ipPool].interval, "the IP pool should be budgeted by the minute")
		assert.Equal(t, time.Minute, poolBudgets[cancelPool].interval, "the cancel pool should be budgeted by the minute")
		// Each placement costs a request weight of 1, and the order pool admits its budget every interval
		placementWeight := poolBudgets[orderPool].limit * int(time.Minute/poolBudgets[orderPool].interval) * int(endpointCosts[orderRate].weight)
		assert.Equal(t, 6000, poolBudgets[ipPool].limit+poolBudgets[cancelPool].limit+placementWeight, "the IP and cancel pools and the request weight of the most placements the order pool admits should together hold REQUEST_WEIGHT's 6000 a minute")
	})
}

func TestOrderbookLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		depth  uint64
		exp    request.EndpointLimit
		weight request.Weight
	}{
		{depth: 0, exp: orderbookDepth100Rate, weight: 5},
		{depth: 1, exp: orderbookDepth100Rate, weight: 5},
		{depth: 100, exp: orderbookDepth100Rate, weight: 5},
		{depth: 101, exp: orderbookDepth500Rate, weight: 25},
		{depth: 500, exp: orderbookDepth500Rate, weight: 25},
		{depth: 501, exp: orderbookDepth1000Rate, weight: 50},
		{depth: 1000, exp: orderbookDepth1000Rate, weight: 50},
		{depth: 1001, exp: orderbookDepth5000Rate, weight: 250},
		{depth: 5000, exp: orderbookDepth5000Rate, weight: 250},
		{depth: 5001, exp: orderbookDepth5000Rate, weight: 250},
	} {
		t.Run("depth "+strconv.FormatUint(tc.depth, 10), func(t *testing.T) {
			t.Parallel()
			got := orderbookLimit(tc.depth)
			assert.Equalf(t, tc.exp, got, "orderbookLimit should pick the tier of depth %d", tc.depth)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "orderbookLimit should charge depth %d its documented weight", tc.depth)
		})
	}
}

func TestSymbolsTickerLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbols int
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{symbols: 0, exp: symbolsTickerAllRate, weight: 4},
		{symbols: 1, exp: symbolsTickerOneRate, weight: 2},
		{symbols: 2, exp: symbolsTickerAllRate, weight: 4},
		{symbols: 100, exp: symbolsTickerAllRate, weight: 4},
	} {
		t.Run(strconv.Itoa(tc.symbols)+" symbols", func(t *testing.T) {
			t.Parallel()
			got := symbolsTickerLimit(tc.symbols)
			assert.Equalf(t, tc.exp, got, "symbolsTickerLimit should pick the tier of %d symbols", tc.symbols)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "symbolsTickerLimit should charge %d symbols the weight Binance.US charges", tc.symbols)
		})
	}
}

func TestTicker24HourLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbols int
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{symbols: 0, exp: ticker24HourAllRate, weight: 80},
		{symbols: 1, exp: ticker24Hour20Rate, weight: 2},
		{symbols: 20, exp: ticker24Hour20Rate, weight: 2},
		{symbols: 21, exp: ticker24Hour100Rate, weight: 40},
		{symbols: 100, exp: ticker24Hour100Rate, weight: 40},
		{symbols: 101, exp: ticker24HourAllRate, weight: 80},
	} {
		t.Run(strconv.Itoa(tc.symbols)+" symbols", func(t *testing.T) {
			t.Parallel()
			got := ticker24HourLimit(tc.symbols)
			assert.Equalf(t, tc.exp, got, "ticker24HourLimit should pick the tier of %d symbols", tc.symbols)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "ticker24HourLimit should charge %d symbols the weight Binance.US charges", tc.symbols)
		})
	}
}

func TestRollingWindowTickerLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbols int
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{symbols: 0, exp: rollingWindowTicker1Rate, weight: 4},
		{symbols: 1, exp: rollingWindowTicker1Rate, weight: 4},
		{symbols: 2, exp: rollingWindowTicker5Rate, weight: 20},
		{symbols: 5, exp: rollingWindowTicker5Rate, weight: 20},
		{symbols: 6, exp: rollingWindowTicker10Rate, weight: 40},
		{symbols: 10, exp: rollingWindowTicker10Rate, weight: 40},
		{symbols: 11, exp: rollingWindowTicker25Rate, weight: 100},
		{symbols: 25, exp: rollingWindowTicker25Rate, weight: 100},
		{symbols: 26, exp: rollingWindowTickerMaxRate, weight: 200},
		{symbols: 50, exp: rollingWindowTickerMaxRate, weight: 200},
		{symbols: 100, exp: rollingWindowTickerMaxRate, weight: 200},
		{symbols: 101, exp: rollingWindowTickerMaxRate, weight: 200},
	} {
		t.Run(strconv.Itoa(tc.symbols)+" symbols", func(t *testing.T) {
			t.Parallel()
			got := rollingWindowTickerLimit(tc.symbols)
			assert.Equalf(t, tc.exp, got, "rollingWindowTickerLimit should pick the tier of %d symbols", tc.symbols)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "rollingWindowTickerLimit should charge %d symbols the weight of their tier's largest count", tc.symbols)
		})
	}
	for symbols := 1; symbols <= 100; symbols++ {
		charged := min(4*symbols, 200)
		assert.GreaterOrEqualf(t, int(endpointCosts[rollingWindowTickerLimit(symbols)].weight), charged, "rollingWindowTickerLimit should charge %d symbols at least the weight Binance.US charges", symbols)
	}
}

func TestTradingDayTickerLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbols int
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{symbols: 0, exp: tradingDayTicker1Rate, weight: 4},
		{symbols: 1, exp: tradingDayTicker1Rate, weight: 4},
		{symbols: 2, exp: tradingDayTicker5Rate, weight: 20},
		{symbols: 5, exp: tradingDayTicker5Rate, weight: 20},
		{symbols: 6, exp: tradingDayTicker10Rate, weight: 40},
		{symbols: 10, exp: tradingDayTicker10Rate, weight: 40},
		{symbols: 11, exp: tradingDayTicker25Rate, weight: 100},
		{symbols: 25, exp: tradingDayTicker25Rate, weight: 100},
		{symbols: 26, exp: tradingDayTickerMaxRate, weight: 200},
		{symbols: 50, exp: tradingDayTickerMaxRate, weight: 200},
		{symbols: 100, exp: tradingDayTickerMaxRate, weight: 200},
		{symbols: 101, exp: tradingDayTickerMaxRate, weight: 200},
	} {
		t.Run(strconv.Itoa(tc.symbols)+" symbols", func(t *testing.T) {
			t.Parallel()
			got := tradingDayTickerLimit(tc.symbols)
			assert.Equalf(t, tc.exp, got, "tradingDayTickerLimit should pick the tier of %d symbols", tc.symbols)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "tradingDayTickerLimit should charge %d symbols the weight of their tier's largest count", tc.symbols)
		})
	}
	for symbols := 1; symbols <= 100; symbols++ {
		documented := min(4*symbols, 200)
		assert.GreaterOrEqualf(t, int(endpointCosts[tradingDayTickerLimit(symbols)].weight), documented, "tradingDayTickerLimit should charge %d symbols at least their documented weight", symbols)
	}
}

func TestAccountTradesLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderID uint64
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{orderID: 0, exp: accountTradesRate, weight: 20},
		{orderID: 1, exp: accountTradesByOrderIDRate, weight: 5},
		{orderID: 100234, exp: accountTradesByOrderIDRate, weight: 5},
	} {
		t.Run("order ID "+strconv.FormatUint(tc.orderID, 10), func(t *testing.T) {
			t.Parallel()
			got := accountTradesLimit(tc.orderID)
			assert.Equalf(t, tc.exp, got, "accountTradesLimit should pick the tier of order ID %d", tc.orderID)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "accountTradesLimit should charge order ID %d its documented weight", tc.orderID)
		})
	}
}

func TestPreventedMatchesLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderID uint64
		exp     request.EndpointLimit
		weight  request.Weight
	}{
		{orderID: 0, exp: preventedMatchesRate, weight: 1},
		{orderID: 1, exp: preventedMatchesByOrderIDRate, weight: 10},
		{orderID: 35, exp: preventedMatchesByOrderIDRate, weight: 10},
	} {
		t.Run("order ID "+strconv.FormatUint(tc.orderID, 10), func(t *testing.T) {
			t.Parallel()
			got := preventedMatchesLimit(tc.orderID)
			assert.Equalf(t, tc.exp, got, "preventedMatchesLimit should pick the tier of order ID %d", tc.orderID)
			assert.Equalf(t, tc.weight, endpointCosts[got].weight, "preventedMatchesLimit should charge order ID %d its documented weight", tc.orderID)
		})
	}
}
