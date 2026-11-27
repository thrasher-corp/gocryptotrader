package okx

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// unlimitedEndpointLimits are the endpoint limits OKX documents no rate for: demo balance reductions are unlimited,
// and increases are limited by a daily quota the requester cannot hold
var unlimitedEndpointLimits = map[request.EndpointLimit]bool{increaseDemoAccountBalanceEPL: true, reduceDemoAccountBalanceEPL: true}

// orderWeightedLimit is the order budget and order count a limit orderWeightedEPL derives charges
type orderWeightedLimit struct {
	budget request.EndpointLimit
	orders request.Weight
}

// orderWeightedEndpointLimits returns every endpoint limit orderWeightedEPL derives, with what each charges
func orderWeightedEndpointLimits() map[request.EndpointLimit]orderWeightedLimit {
	derived := make(map[request.EndpointLimit]orderWeightedLimit)
	for budget, maxOrders := range orderBudgets {
		for orders := request.Weight(2); int(orders) <= maxOrders; orders++ {
			derived[orderWeightedEPL(budget, int(orders))] = orderWeightedLimit{budget, orders}
		}
	}
	return derived
}

func TestNewRateLimits(t *testing.T) {
	t.Parallel()
	definitions := newRateLimits()
	weighted := orderWeightedEndpointLimits()
	require.Len(t, definitions, int(endpointLimitCount-1)+len(weighted), "newRateLimits must define every endpoint limit once and nothing else")
	again := newRateLimits()
	for epl := request.EndpointLimit(1); epl < endpointLimitCount; epl++ {
		t.Run(strconv.FormatUint(uint64(epl), 10), func(t *testing.T) {
			t.Parallel()
			limiter, defined := definitions[epl]
			require.Truef(t, defined, "endpoint limit %d must have a rate limit definition", epl)
			if epl == websocketRequestEPL {
				assert.Equal(t, request.RateLimitNotRequired, limiter, "websocket requests should be left to the connection's own limiter")
				return
			}
			require.NotNil(t, limiter, "a REST endpoint limit must have a limiter")
			assert.Positive(t, float64(limiter.Limit()), "the limiter should admit requests")
			if unlimitedEndpointLimits[epl] {
				assert.Equal(t, rate.Inf, limiter.Limit(), "an endpoint OKX sets no rate for should be unrestricted")
			} else {
				assert.NotEqual(t, rate.Inf, limiter.Limit(), "the limiter should not be unrestricted, as a definition without actions or an interval is")
			}
			assert.Equal(t, request.Weight(1), limiter.Weight(), "each request should take one token")
			assert.False(t, limiter.SharesBudgetWith(again[epl]), "each call should return new limiters, so draining one set leaves another's budget alone")
		})
	}

	t.Run("order weighted limits", func(t *testing.T) {
		t.Parallel()
		require.Len(t, weighted, 3*(maxOrdersPerBatch-1)+maxCancelAlgosPerRequest-1, "orderWeightedEPL must derive a distinct limit for every order count of every order budget")
		for epl, derived := range weighted {
			assert.GreaterOrEqualf(t, epl, endpointLimitCount, "the limit for %d orders should not be a declared endpoint limit", derived.orders)
			limiter := definitions[epl]
			require.NotNilf(t, limiter, "the limit for %d orders must be defined", derived.orders)
			assert.Equalf(t, derived.orders, limiter.Weight(), "the limit for %d orders should take a token per order", derived.orders)
			assert.Truef(t, limiter.SharesBudgetWith(definitions[derived.budget]), "the limit for %d orders should draw on its budget's own limiter", derived.orders)
		}
	})

	t.Run("every endpoint limit admits OKX's documented rate", func(t *testing.T) {
		t.Parallel()
		documented := make(map[request.EndpointLimit]bool, len(documentedRates))
		for _, tc := range documentedRates {
			documented[tc.epl] = true
			refilled := tc.requests - orderBudgets[tc.epl]
			assert.InEpsilonf(t, float64(refilled)/tc.interval.Seconds(), float64(definitions[tc.epl].Limit()), 1e-9, "%s should hold to %d requests every %s", tc.route, tc.requests, tc.interval)
		}
		assert.InEpsilon(t, 3/1.2, float64(definitions[websocketConnectionEPL].Limit()), 1e-9, "websocket connections should be paced at 3 every 1.2 seconds, inside OKX's 3 a second per IP")
		for epl := request.EndpointLimit(1); epl < endpointLimitCount; epl++ {
			if epl == websocketRequestEPL || unlimitedEndpointLimits[epl] || epl == websocketConnectionEPL {
				continue
			}
			assert.Truef(t, documented[epl], "endpoint limit %d should be pinned to OKX's documented rate", epl)
		}
	})
}

func TestWebsocketControlLimiter(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		limiter := websocketControlLimiter()
		start := time.Now()
		for range 20 {
			require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must not error within the burst")
		}
		assert.Equal(t, start, time.Now(), "a connection should log in and subscribe without waiting")
		sent := 20
		for time.Since(start) < time.Hour {
			ctx, cancel := context.WithDeadline(t.Context(), start.Add(time.Hour))
			err := limiter.RateLimit(ctx)
			cancel()
			if err != nil {
				break
			}
			if time.Since(start) < time.Hour {
				sent++
			}
		}
		assert.LessOrEqual(t, sent, 480, "no hour should carry more than OKX's 480 control requests")
		assert.GreaterOrEqual(t, sent, 479, "the hour's budget should be spent rather than left idle")
	})
}

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	definitions := newRateLimits()
	rl, err := request.New("RateLimit_Static", new(http.Client), request.WithLimiter(definitions))
	require.NoError(t, err, "request.New must not error")
	for epl := request.EndpointLimit(1); epl < endpointLimitCount; epl++ {
		t.Run(strconv.FormatUint(uint64(epl), 10), func(t *testing.T) {
			t.Parallel()
			limiter, defined := definitions[epl]
			require.Truef(t, defined, "endpoint limit %d must have a rate limit definition", epl)
			if epl == websocketRequestEPL {
				assert.Equal(t, request.RateLimitNotRequired, limiter, "websocket requests should be left to the connection's own limiter")
				return
			}
			require.NoError(t, rl.InitiateRateLimit(t.Context(), epl), "InitiateRateLimit must not error")
		})
	}
}

func TestBatchLimitersTrackOrderBudgets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		epl       request.EndpointLimit
		budget    int
		maxOrders int
	}{
		{"batch-orders", placeMultipleOrdersEPL, 300, maxOrdersPerBatch},
		{"cancel-batch-orders", cancelMultipleOrdersEPL, 300, maxOrdersPerBatch},
		{"amend-batch-orders", amendMultipleOrdersEPL, 300, maxOrdersPerBatch},
		{"cancel-algos", cancelAlgoOrderEPL, 20, maxCancelAlgosPerRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.maxOrders, orderBudgets[tc.epl], "the budget should hold a request's most orders")
			refill := float64(tc.budget-tc.maxOrders) / twoSecondsInterval.Seconds()
			assert.InEpsilon(t, refill, float64(rateLimits[tc.epl].Limit()), 1e-9, "the budget should refill all but one request's orders every 2 seconds")
			synctest.Test(t, func(t *testing.T) {
				definitions := newRateLimits()
				type send struct {
					at     time.Duration
					orders int
				}
				var sends []send
				var total int
				start := time.Now()
				for range 4 {
					for _, orders := range []int{tc.maxOrders, tc.maxOrders, 2, tc.maxOrders - 1, tc.maxOrders, 3, tc.maxOrders} {
						require.NoError(t, definitions[orderWeightedEPL(tc.epl, orders)].RateLimit(t.Context()), "RateLimit must not error")
						sends = append(sends, send{time.Since(start), orders})
						total += orders
					}
				}
				assert.Zero(t, sends[0].at, "a first full request should go out at once")
				for i := range sends {
					var inWindow int
					for _, s := range sends {
						if s.at >= sends[i].at && s.at <= sends[i].at+twoSecondsInterval {
							inWindow += s.orders
						}
					}
					assert.LessOrEqualf(t, inWindow, tc.budget, "the 2 seconds from request %d should carry no more orders than OKX's budget", i)
				}
				expLast := time.Duration(float64(total-tc.maxOrders) / refill * float64(time.Second))
				assert.InDelta(t, float64(expLast), float64(sends[len(sends)-1].at), float64(time.Millisecond), "requests should go out as fast as the budget refills")
			})
		})
	}
}

// orderBudgetPoolSize is the tokens each counting pool holds, more than any
// request takes
const orderBudgetPoolSize = 100

// countingLimits returns rate limit definitions charging every endpoint limit to a pool of its own holding
// orderBudgetPoolSize tokens, the order weighted limits drawing on their budget's pool, with the pools. A pool never
// refills, so the tokens it has left count exactly the tokens taken, and a call wanting more than a pool holds fails
// at once under a context deadline. Websocket requests keep to the connection's own limiter, as in newRateLimits.
func countingLimits() (map[request.EndpointLimit]*rate.Limiter, request.RateLimitDefinitions) {
	pools := make(map[request.EndpointLimit]*rate.Limiter, endpointLimitCount)
	definitions := make(request.RateLimitDefinitions, endpointLimitCount)
	for epl := request.EndpointLimit(1); epl < endpointLimitCount; epl++ {
		if epl == websocketRequestEPL {
			definitions[epl] = request.RateLimitNotRequired
			continue
		}
		pools[epl] = rate.NewLimiter(0, orderBudgetPoolSize)
		definitions[epl] = request.GetRateLimiterWithWeight(pools[epl], 1)
	}
	addOrderWeightedLimits(definitions, pools)
	return pools, definitions
}

// chargeToPools makes ex's requester charge every endpoint limit to the pools
// of countingLimits, which it returns
func chargeToPools(t *testing.T, ex *Exchange) map[request.EndpointLimit]*rate.Limiter {
	t.Helper()
	pools, definitions := countingLimits()
	request.WithLimiter(definitions)(ex.Requester)
	if err := ex.Requester.EnableRateLimiter(); err != nil {
		require.ErrorIs(t, err, request.ErrRateLimiterAlreadyEnabled, "EnableRateLimiter must enable the rate limiter")
	}
	return pools
}

// newChargedWebsocketTestExchange returns a logged in websocket test exchange
// whose requester and websocket charge every endpoint limit to the pools of
// countingLimits, which it returns
func newChargedWebsocketTestExchange(t *testing.T, srv *websocketTestServer) (ex *Exchange, pools map[request.EndpointLimit]*rate.Limiter) {
	t.Helper()
	pools, definitions := countingLimits()
	return newWebsocketTestExchangeWithLimits(t, srv, true, nil, definitions), pools
}

// takenTokens returns the tokens each pool has given out, leaving out the
// pools nothing was taken from
func takenTokens(pools map[request.EndpointLimit]*rate.Limiter) map[request.EndpointLimit]float64 {
	taken := make(map[request.EndpointLimit]float64)
	for epl, pool := range pools {
		if n := orderBudgetPoolSize - pool.Tokens(); n != 0 {
			taken[epl] = n
		}
	}
	return taken
}

// chargeContext bounds a call charged to counting pools, which refuse at once a
// token they no longer hold when the context has a deadline
func chargeContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// orderRows returns a successful order action row per order, each carrying its
// ID under idKey
func orderRows(idKey string, orders int) []map[string]string {
	rows := make([]map[string]string, orders)
	for i := range rows {
		rows[i] = map[string]string{idKey: strconv.Itoa(i + 1), "sCode": "0", "sMsg": ""}
	}
	return rows
}

// orderRowsHandler answers a POST to path with orderRows
func orderRowsHandler(t *testing.T, path, idKey string, orders int) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, orderRows(idKey, orders))
	}
}

// newRetriedBatchExchange returns an exchange charging to the pools of
// countingLimits, which it returns, whose mock server answers the first POST to
// path with HTTP 429, which the requester retries at once, and the next with
// three orders' rows, and a count of the requests the server answers
func newRetriedBatchExchange(t *testing.T, path, idKey string) (ex *Exchange, pools map[request.EndpointLimit]*rate.Limiter, requests *atomic.Int32) {
	t.Helper()
	requests = new(atomic.Int32)
	answer := orderRowsHandler(t, path, idKey, 3)
	ex = newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		answer(w, r)
	}))
	pools = chargeToPools(t, ex)
	request.WithBackoff(func(int) time.Duration { return 0 })(ex.Requester)
	return ex, pools, requests
}

func TestBatchEndpointLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		orders int
		exp    request.EndpointLimit
	}{
		{"no orders", 0, placeMultipleOrdersEPL},
		{"a lone order", 1, placeOrderEPL},
		{"two orders", 2, orderWeightedEPL(placeMultipleOrdersEPL, 2)},
		{"a full batch", maxOrdersPerBatch, orderWeightedEPL(placeMultipleOrdersEPL, maxOrdersPerBatch)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, batchEndpointLimit(placeOrderEPL, placeMultipleOrdersEPL, tc.orders), "batchEndpointLimit should charge a lone order to the single order limit and any other count to the batch's order budget")
		})
	}
}

func TestOrderWeightedEPL(t *testing.T) {
	t.Parallel()
	require.LessOrEqual(t, endpointLimitCount, request.EndpointLimit(1)<<orderCountShift, "every declared endpoint limit must fit below the order count")
	for _, orders := range []int{0, 1} {
		assert.Equalf(t, cancelAlgoOrderEPL, orderWeightedEPL(cancelAlgoOrderEPL, orders), "%d orders should take the budget's own limit", orders)
	}
	seen := make(map[request.EndpointLimit]int)
	for orders := 2; orders <= maxOrdersPerBatch; orders++ {
		for budget := range orderBudgets {
			epl := orderWeightedEPL(budget, orders)
			assert.GreaterOrEqualf(t, epl, endpointLimitCount, "the limit for %d orders should not be a declared endpoint limit", orders)
			seen[epl]++
		}
	}
	for epl, n := range seen {
		assert.Equalf(t, 1, n, "endpoint limit %d should name one budget and order count", epl)
	}
}

// documentedRates pins every endpoint limit to the first rate limit OKX documents for its route, so a limit that
// drifts from the documentation fails TestNewRateLimits
var documentedRates = []struct {
	route    string
	epl      request.EndpointLimit
	requests int
	interval time.Duration
}{
	{"GET /api/v5/account/instruments", getAccountInstrumentsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/balance", getAccountBalanceEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/positions", getPositionsEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/positions-history", getPositionsHistoryEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/account-position-risk", getAccountAndPositionRiskEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/bills", getBillsDetailsEPL, 5, oneSecondInterval},
	{"GET /api/v5/account/bills-archive", getBillsDetailArchiveEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/bills-history-archive", billHistoryArchiveEPL, 12, 24 * time.Hour},
	{"GET /api/v5/account/bills-history-archive", getBillHistoryArchiveEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/subtypes", getBillTypesEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/config", getAccountConfigurationEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-position-mode", setPositionModeEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-leverage", setLeverageEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/max-size", getMaximumBuyOrSellAmountEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/max-avail-size", getMaximumAvailableTradableAmountEPL, 20, twoSecondsInterval},
	{"POST /api/v5/account/position/margin-balance", increaseOrDecreaseMarginEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/leverage-info", getLeverageEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/adjust-leverage-info", getLeverateEstimatedInfoEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/max-loan", getTheMaximumLoanOfInstrumentEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/trade-fee", getFeeRatesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/interest-accrued", getInterestAccruedDataEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/interest-rate", getInterestRateEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-fee-type", setFeeTypeEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-greeks", setGreeksEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-isolated-mode", isolatedMarginTradingSettingsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/max-withdrawal", getMaximumWithdrawalsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/account/risk-state", getAccountRiskStateEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/interest-limits", getBorrowInterestAndLimitEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/spot-manual-borrow-repay", manualBorrowOrRepayEPL, 1, threeSecondsInterval},
	{"POST /api/v5/account/set-auto-repay", setAutoRepayEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/spot-borrow-repay-history", getBorrowRepayHistoryEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/position-builder", newPositionBuilderEPL, 2, twoSecondsInterval},
	{"POST /api/v5/account/position-builder-graph", positionBuilderTrendGraphEPL, 1, fiveSecondsInterval},
	{"POST /api/v5/account/set-riskOffset-amt", setRiskOffsetAmountEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/greeks", getGreeksEPL, 10, twoSecondsInterval},
	{"GET /api/v5/account/position-tiers", getPMLimitationEPL, 10, twoSecondsInterval},
	{"POST /api/v5/account/activate-option", activateOptionEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/activate-feature", activateFeatureEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-auto-loan", setAutoLoanEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/account-level-switch-preset", presetAccountModeSwitchEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/set-account-switch-precheck", precheckAccountModeSwitchEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-account-level", setAccountLevelEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/set-collateral-assets", setCollateralAssetsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/account/collateral-assets", getCollateralAssetsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/mmp-reset", resetMMPStatusEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/mmp-config", setMMPEPL, 2, tenSecondsInterval},
	{"GET /api/v5/account/mmp-config", getMMPConfigEPL, 5, twoSecondsInterval},
	{"POST /api/v5/account/move-positions", movePositionsEPL, 1, oneSecondInterval},
	{"GET /api/v5/account/move-positions-history", getMovePositionsHistoryEPL, 2, twoSecondsInterval},
	{"POST /api/v5/account/set-auto-earn", setAutoEarnEPL, 2, twoSecondsInterval},
	{"POST /api/v5/account/set-settle-currency", setSettleCurrencyEPL, 20, twoSecondsInterval},
	{"POST /api/v5/account/set-trading-config", setTradingConfigEPL, 1, twoSecondsInterval},
	{"GET /api/v5/account/precheck-set-delta-neutral", precheckSetDeltaNeutralEPL, 1, twoSecondsInterval},
	{"GET /api/v5/users/glp/todayperformance", getGLPTodayPerformanceEPL, 5, twoSecondsInterval},
	{"GET /api/v5/users/glp/historicalperformance", getGLPHistoricalPerformanceEPL, 5, twoSecondsInterval},
	{"GET /api/v5/users/glp/v2/overview", getGLPV2OverviewEPL, 5, twoSecondsInterval},
	{"GET /api/v5/users/glp/v2/details", getGLPV2AssessmentDetailsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/users/glp/v2/history/performance", getGLPV2PerformanceHistoryEPL, 5, twoSecondsInterval},
	{"POST /api/v5/trade/order", placeOrderEPL, 60, twoSecondsInterval},
	{"POST /api/v5/trade/batch-orders", placeMultipleOrdersEPL, 300, twoSecondsInterval},
	{"POST /api/v5/trade/cancel-order", cancelOrderEPL, 60, twoSecondsInterval},
	{"POST /api/v5/trade/cancel-batch-orders", cancelMultipleOrdersEPL, 300, twoSecondsInterval},
	{"POST /api/v5/trade/amend-order", amendOrderEPL, 60, twoSecondsInterval},
	{"POST /api/v5/trade/amend-batch-orders", amendMultipleOrdersEPL, 300, twoSecondsInterval},
	{"POST /api/v5/trade/close-position", closePositionEPL, 20, twoSecondsInterval},
	{"GET /api/v5/trade/order", getOrderDetEPL, 60, twoSecondsInterval},
	{"GET /api/v5/trade/orders-pending", getOrderListEPL, 60, twoSecondsInterval},
	{"GET /api/v5/trade/orders-history", getOrderHistory7DaysEPL, 40, twoSecondsInterval},
	{"GET /api/v5/trade/orders-history-archive", getOrderHistory3MonthsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/trade/fills", getTransactionDetail3DaysEPL, 60, twoSecondsInterval},
	{"GET /api/v5/trade/fills-history", getTransactionDetail3MonthsEPL, 10, twoSecondsInterval},
	{"GET /api/v5/trade/easy-convert-currency-list", getEasyConvertCurrencyListEPL, 1, twoSecondsInterval},
	{"POST /api/v5/trade/easy-convert", placeEasyConvertEPL, 1, twoSecondsInterval},
	{"GET /api/v5/trade/easy-convert-history", getEasyConvertHistoryEPL, 1, twoSecondsInterval},
	{"GET /api/v5/trade/one-click-repay-currency-list", oneClickRepayCurrencyListEPL, 1, twoSecondsInterval},
	{"POST /api/v5/trade/one-click-repay", tradeOneClickRepayEPL, 1, twoSecondsInterval},
	{"GET /api/v5/trade/one-click-repay-history", getOneClickRepayHistoryEPL, 1, twoSecondsInterval},
	{"GET /api/v5/trade/one-click-repay-currency-list-v2", getOneClickRepayCurrencyListV2EPL, 1, twoSecondsInterval},
	{"POST /api/v5/trade/one-click-repay-v2", tradeOneClickRepayV2EPL, 1, twoSecondsInterval},
	{"GET /api/v5/trade/one-click-repay-history-v2", getOneClickRepayHistoryV2EPL, 1, twoSecondsInterval},
	{"POST /api/v5/trade/mass-cancel", massCancelMMPOrderEPL, 5, twoSecondsInterval},
	{"POST /api/v5/trade/cancel-all-after", cancelAllAfterCountdownEPL, 1, oneSecondInterval},
	{"GET /api/v5/trade/account-rate-limit", getTradeAccountRateLimitEPL, 1, oneSecondInterval},
	{"POST /api/v5/trade/order-precheck", orderPreCheckEPL, 5, twoSecondsInterval},
	{"POST /api/v5/trade/order-algo", placeAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/trade/cancel-algos", cancelAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/trade/amend-algos", amendAlgoOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/trade/order-algo", getAlgoOrderDetailEPL, 20, twoSecondsInterval},
	{"GET /api/v5/trade/orders-algo-pending", getAlgoOrderListEPL, 20, twoSecondsInterval},
	{"GET /api/v5/trade/orders-algo-history", getAlgoOrderHistoryEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/order-algo", gridTradingEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/amend-algo-basic-param", amendGridAlgoOrderBasicParametersEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/amend-order-algo", amendGridAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/stop-order-algo", stopGridAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/close-position", closePositionForForContractGridEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/cancel-close-order", cancelClosePositionOrderForContractGridEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/order-instant-trigger", instantTriggerGridAlgoOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/orders-algo-pending", getGridAlgoOrderListEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/orders-algo-history", getGridAlgoOrderHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/orders-algo-details", getGridAlgoOrderDetailsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/sub-orders", getGridAlgoSubOrdersEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/positions", getGridAlgoOrderPositionsEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/withdraw-income", spotGridWithdrawIncomeEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/compute-margin-balance", computeMarginBalanceEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/margin-balance", adjustMarginBalanceEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/adjust-investment", addGridInvestmentEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/ai-param", getGridAIParameterEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/min-investment", computeMinInvestmentEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/public/rsi-back-testing", rsiBackTestingEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/grid/grid-quantity", getMaxGridQuantityEPL, 5, twoSecondsInterval},
	{"POST /api/v5/tradingBot/grid/copy-order-algo", copyGridAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/create", placeDCAAlgoOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/amend-order-algo", amendSpotDCABasicParametersEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/stop", stopDCAAlgoOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/dca/ongoing-list", getDCAAlgoOrderListEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/dca/history-list", getDCAAlgoOrderHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/dca/orders", getDCASubOrdersEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/orders/manual-buy", addDCAInvestmentEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/settings/reinvestment", amendDCAReinvestmentEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/settings/take-profit", amendDCATakeProfitEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/dca/position-details", getDCAPositionDetailsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/dca/cycle-list", getDCACycleListEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/margin/add", addDCAMarginEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/dca/margin/reduce", reduceDCAMarginEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/create-signal", createSignalEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/signals", getSignalsEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/order-algo", createSignalBotEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/stop-order-algo", cancelSignalBotsEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/margin-balance", adjustSignalBotMarginBalanceEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/amendTPSL", amendSignalBotTakeProfitStopLossEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/set-instruments", setSignalBotInstrumentsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/orders-algo-details", signalBotOrderDetailsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/orders-algo-pending", getActiveSignalBotsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/orders-algo-history", getSignalBotHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/positions", signalBotOrderPositionsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/positions-history", getSignalBotPositionHistoryEPL, 10, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/close-position", closeSignalBotPositionEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/sub-order", placeSignalBotSubOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/signal/cancel-sub-order", cancelSignalBotSubOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/sub-orders", signalBotSubOrdersEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/signal/event-history", signalBotEventHistoryEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/order-algo", placeRecurringBuyOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/amend-order-algo", amendRecurringBuyOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/stop-order-algo", stopRecurringBuyOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/recurring/orders-algo-pending", getRecurringBuyOrderListEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/recurring/orders-algo-history", getRecurringBuyOrderHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/recurring/orders-algo-details", getRecurringBuyOrderDetailEPL, 20, twoSecondsInterval},
	{"GET /api/v5/tradingBot/recurring/sub-orders", getRecurringBuySubOrdersEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/amend-recurring-time", amendRecurringBuyTimeEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/amend-recurring-amount", amendRecurringBuyAmountEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/add-investment", addRecurringBuyInvestmentEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/pause", pauseRecurringBuyEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/restart", restartRecurringBuyEPL, 20, twoSecondsInterval},
	{"POST /api/v5/tradingBot/recurring/amend-price-range", amendRecurringBuyPriceRangeEPL, 20, twoSecondsInterval},
	{"GET /api/v5/copytrading/current-subpositions", getExistingLeadingPositionsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/copytrading/subpositions-history", getLeadingPositionHistoryEPL, 20, twoSecondsInterval},
	{"POST /api/v5/copytrading/algo-order", placeLeadingStopOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/copytrading/close-subposition", closeLeadingPositionEPL, 20, twoSecondsInterval},
	{"GET /api/v5/copytrading/instruments", getLeadingInstrumentsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/copytrading/set-instruments", amendLeadingInstrumentsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/profit-sharing-details", getProfitSharingLimitEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/total-profit-sharing", getTotalProfitSharingEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/unrealized-profit-sharing-details", getUnrealisedProfitSharingDetailsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/total-unrealized-profit-sharing", getTotalUnrealisedProfitSharingEPL, 5, twoSecondsInterval},
	{"POST /api/v5/copytrading/amend-profit-sharing-ratio", amendProfitSharingRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/config", getCopyTradingAccountConfigurationEPL, 5, twoSecondsInterval},
	{"POST /api/v5/copytrading/first-copy-settings", setFirstCopySettingsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/copytrading/amend-copy-settings", amendFirstCopySettingsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/copytrading/stop-copy-trading", stopCopyingEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/copy-settings", getCopySettingsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/current-lead-traders", getMyLeadTradersEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-config", getCopyTradingConfigurationEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-lead-traders", getLeadTraderRanksEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-weekly-pnl", getLeadTraderWeeklyProfitAndLossEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-pnl", getLeadTraderDailyProfitAndLossEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-stats", getLeadTraderStatsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-preference-currency", getLeadTraderCurrencyPreferencesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-current-subpositions", getTraderCurrentLeadPositionsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-subpositions-history", getLeadTraderLeadPositionHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/copytrading/public-copy-traders", getCopyTradersEPL, 5, twoSecondsInterval},
	{"GET /api/v5/market/tickers", getTickersEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/ticker", getTickerEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/books", getOrderBookEPL, 40, twoSecondsInterval},
	{"GET /api/v5/market/books-rpi", getRPIOrderBookEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/books-full", getFullOrderBookEPL, 10, twoSecondsInterval},
	{"GET /api/v5/market/candles", getCandlesticksEPL, 40, twoSecondsInterval},
	{"GET /api/v5/market/history-candles", getCandlestickHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/trades", getTradesRequestEPL, 100, twoSecondsInterval},
	{"GET /api/v5/market/history-trades", getTradesHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/option/instrument-family-trades", optionInstrumentTradeFamilyEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/option-trades", optionTradesEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/platform-24-volume", get24HTotalVolumeEPL, 2, twoSecondsInterval},
	{"GET /api/v5/market/call-auction-details", getCallAuctionDetailsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/rfq/counterparties", getCounterpartiesEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/create-rfq", createRequestForQuoteEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-rfq", cancelRequestForQuoteEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-batch-rfqs", cancelMultipleRequestForQuotesEPL, 2, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-all-rfqs", cancelAllRequestForQuotesEPL, 2, twoSecondsInterval},
	{"POST /api/v5/rfq/execute-quote", executeQuoteEPL, 2, threeSecondsInterval},
	{"GET /api/v5/rfq/maker-instrument-settings", getQuoteProductsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/maker-instrument-settings", setQuoteProductsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/mmp-reset", resetRequestForQuoteMMPEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/mmp-config", setRequestForQuoteMMPEPL, 1, tenSecondsInterval},
	{"GET /api/v5/rfq/mmp-config", getRequestForQuoteMMPConfigEPL, 5, twoSecondsInterval},
	{"POST /api/v5/rfq/create-quote", createQuoteEPL, 50, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-quote", cancelQuoteEPL, 50, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-batch-quotes", cancelMultipleQuotesEPL, 2, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-all-quotes", cancelAllQuotesEPL, 2, twoSecondsInterval},
	{"POST /api/v5/rfq/cancel-all-after", cancelAllQuotesAfterCountdownEPL, 1, oneSecondInterval},
	{"GET /api/v5/rfq/rfqs", getRequestForQuotesEPL, 2, twoSecondsInterval},
	{"GET /api/v5/rfq/quotes", getQuotesEPL, 2, twoSecondsInterval},
	{"GET /api/v5/rfq/trades", getTradesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/market/block-tickers", getBlockTickersEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/block-ticker", getBlockTickerEPL, 20, twoSecondsInterval},
	{"GET /api/v5/rfq/public-trades", getPublicTradesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/public/block-trades", getBlockTradesEPL, 20, twoSecondsInterval},
	{"POST /api/v5/sprd/order", placeSpreadOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/sprd/cancel-order", cancelSpreadOrderEPL, 20, twoSecondsInterval},
	{"POST /api/v5/sprd/mass-cancel", cancelAllSpreadOrderEPL, 10, twoSecondsInterval},
	{"POST /api/v5/sprd/amend-order", amendSpreadOrderEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/order", getSpreadOrderDetailsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/orders-pending", getActiveSpreadOrdersEPL, 10, twoSecondsInterval},
	{"GET /api/v5/sprd/orders-history", getSpreadOrders21DaysEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/orders-history-archive", getSpreadOrders3MonthsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/trades", getSpreadOrderTradesEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/spreads", getSpreadsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/books", getSpreadOrderbookEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/sprd-ticker", getSpreadTickerEPL, 20, twoSecondsInterval},
	{"GET /api/v5/sprd/public-trades", getSpreadPublicTradesEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/sprd-candles", getSpreadCandlesticksEPL, 40, twoSecondsInterval},
	{"GET /api/v5/market/sprd-history-candles", getSpreadCandlesticksHistoryEPL, 20, twoSecondsInterval},
	{"POST /api/v5/sprd/cancel-all-after", cancelAllSpreadOrdersAfterEPL, 1, oneSecondInterval},
	{"GET /api/v5/public/instruments", getInstrumentsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/event-contract/series", getEventContractSeriesEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/event-contract/events", getEventContractEventsEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/corporate-actions", getCorporateActionsEPL, 1, twoSecondsInterval},
	{"GET /api/v5/public/event-contract/markets", getEventContractMarketsEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/estimated-price", getEstimatedDeliveryExercisePriceEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/delivery-exercise-history", getDeliveryExerciseHistoryEPL, 40, twoSecondsInterval},
	{"GET /api/v5/public/estimated-settlement-info", getEstimatedSettlementPriceEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/settlement-history", getFuturesSettlementHistoryEPL, 40, twoSecondsInterval},
	{"GET /api/v5/public/funding-rate", getFundingEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/funding-rate-history", getFundingRateHistoryEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/open-interest", getOpenInterestEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/price-limit", getLimitPriceEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/opt-summary", getOptionMarketDateEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/discount-rate-interest-free-quota", getDiscountRateAndInterestFreeQuotaEPL, 2, twoSecondsInterval},
	{"GET /api/v5/public/time", getSystemTimeEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/mark-price", getMarkPriceEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/position-tiers", getPositionTiersEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/interest-rate-loan-quota", getInterestRateAndLoanQuotaEPL, 2, twoSecondsInterval},
	{"GET /api/v5/public/underlying", getUnderlyingEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/insurance-fund", getInsuranceFundEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/convert-contract-coin", unitConvertEPL, 10, twoSecondsInterval},
	{"GET /api/v5/public/instrument-tick-bands", optionTickBandsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/public/premium-history", getPremiumHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/index-tickers", getIndexTickersEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/index-candles", getIndexCandlesticksEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/history-index-candles", getIndexCandlesticksHistoryEPL, 10, twoSecondsInterval},
	{"GET /api/v5/market/mark-price-candles", getMarkPriceCandlesticksEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/history-mark-price-candles", getMarkPriceCandlesticksHistoryEPL, 20, twoSecondsInterval},
	{"GET /api/v5/market/exchange-rate", getExchangeRateRequestEPL, 1, twoSecondsInterval},
	{"GET /api/v5/market/index-components", getIndexComponentsEPL, 20, twoSecondsInterval},
	{"GET /api/v5/public/economic-calendar", getEconomicCalendarEPL, 1, fiveSecondsInterval},
	{"GET /api/v5/public/market-data-history", getHistoricalMarketDataEPL, 5, twoSecondsInterval},
	{"GET /api/v5/public/mm-instrument-types", getMMInstrumentTypesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/public/delta-hedge-currencies", getDeltaHedgeCurrenciesEPL, 20, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/trading-data/support-coin", getSupportCoinEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/open-interest-history", rubikGetContractOpenInterestHistoryEPL, 10, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/taker-volume", getTakerVolumeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/taker-volume-contract", rubikContractTakerVolumeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/margin/loan-ratio", getMarginLendingRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/long-short-account-ratio-contract-top-trader", rubikTopTradersContractLongShortRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/long-short-position-ratio-contract-top-trader", getTopTradersContractLongShortPositionRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/long-short-account-ratio-contract", getContractLongShortAccountRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/long-short-account-ratio", getLongShortRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/contracts/open-interest-volume", getContractsOpenInterestAndVolumeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/option/open-interest-volume", getOptionsOpenInterestAndVolumeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/option/open-interest-volume-ratio", getPutCallRatioEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/option/open-interest-volume-expiry", getOpenInterestAndVolumeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/option/open-interest-volume-strike", getOpenInterestAndVolumeStrikeEPL, 5, twoSecondsInterval},
	{"GET /api/v5/rubik/stat/option/taker-block-volume", getTakerFlowEPL, 5, twoSecondsInterval},
	{"GET /api/v5/asset/currencies", getCurrenciesEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/balances", getBalanceEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/non-tradable-assets", getNonTradableAssetsEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/asset-valuation", getAccountAssetValuationEPL, 1, oneSecondInterval},
	{"POST /api/v5/asset/transfer", fundsTransferEPL, 2, oneSecondInterval},
	{"GET /api/v5/asset/transfer-state", getFundsTransferStateEPL, 10, oneSecondInterval},
	{"GET /api/v5/asset/bills", assetBillsDetailsEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/bills-history", getAssetBillsHistoryEPL, 1, oneSecondInterval},
	{"GET /api/v5/asset/deposit-address", getDepositAddressEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/deposit-history", getDepositHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/travel-rule/get-locked", getLockedTravelRuleDepositsEPL, 3, oneSecondInterval},
	{"POST /api/v5/asset/travel-rule/unlock", unlockTravelRuleDepositEPL, 3, oneSecondInterval},
	{"POST /api/v5/asset/withdrawal", withdrawalEPL, 6, oneSecondInterval},
	{"POST /api/v5/asset/cancel-withdrawal", cancelWithdrawalEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/withdrawal-history", getWithdrawalHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/deposit-withdraw-status", getDepositWithdrawalStatusEPL, 1, twoSecondsInterval},
	{"GET /api/v5/asset/exchange-list", getPublicExchangeListEPL, 6, oneSecondInterval},
	{"POST /api/v5/asset/monthly-statement", applyForMonthlyStatementEPL, 20, 30 * 24 * time.Hour},
	{"GET /api/v5/asset/monthly-statement", getMonthlyStatementEPL, 10, twoSecondsInterval},
	{"GET /api/v5/asset/convert/currencies", getConvertCurrenciesEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/convert/currency-pair", getConvertCurrencyPairEPL, 6, oneSecondInterval},
	{"POST /api/v5/asset/convert/estimate-quote", estimateQuoteEPL, 10, oneSecondInterval},
	{"POST /api/v5/asset/convert/trade", convertTradeEPL, 10, oneSecondInterval},
	{"GET /api/v5/asset/convert/history", getConvertHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/fiat/deposit-payment-methods", getFiatDepositPaymentMethodsEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/withdrawal-payment-methods", getWithdrawalPaymentMethodsEPL, 3, oneSecondInterval},
	{"POST /api/v5/fiat/create-withdrawal", createWithdrawalOrderEPL, 3, oneSecondInterval},
	{"POST /api/v5/fiat/cancel-withdrawal", cancelWithdrawalOrderEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/withdrawal-order-history", getFiatWithdrawalOrderHistoryEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/withdrawal", getWithdrawalOrderDetailEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/deposit-order-history", getDepositOrderHistoryEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/deposit", getDepositOrderDetailEPL, 3, oneSecondInterval},
	{"GET /api/v5/fiat/buy-sell/currencies", getBuySellCurrenciesEPL, 6, oneSecondInterval},
	{"GET /api/v5/fiat/buy-sell/currency-pair", getBuySellCurrencyPairEPL, 6, oneSecondInterval},
	{"POST /api/v5/fiat/buy-sell/quote", getBuySellQuoteEPL, 10, oneSecondInterval},
	{"POST /api/v5/fiat/buy-sell/trade", placeBuySellTradeEPL, 1, fiveSecondsInterval},
	{"GET /api/v5/fiat/buy-sell/history", getBuySellTradeHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/users/subaccount/list", viewSubaccountListEPL, 20, twoSecondsInterval},
	{"POST /api/v5/users/subaccount/create-subaccount", createSubAccountEPL, 1, oneSecondInterval},
	{"POST /api/v5/users/subaccount/apikey", createSubAccountAPIKeyEPL, 1, oneSecondInterval},
	{"GET /api/v5/users/subaccount/apikey", getSubAccountAPIKeysEPL, 20, twoSecondsInterval},
	{"POST /api/v5/users/subaccount/modify-apikey", resetSubAccountAPIKeyEPL, 1, oneSecondInterval},
	{"POST /api/v5/users/subaccount/delete-apikey", deleteSubAccountAPIKeyEPL, 1, oneSecondInterval},
	{"GET /api/v5/account/subaccount/balances", getSubaccountTradingBalanceEPL, 6, twoSecondsInterval},
	{"GET /api/v5/asset/subaccount/balances", getSubaccountFundingBalanceEPL, 6, twoSecondsInterval},
	{"GET /api/v5/account/subaccount/max-withdrawal", getSubAccountMaxWithdrawalEPL, 20, twoSecondsInterval},
	{"GET /api/v5/asset/subaccount/bills", historyOfSubaccountTransferEPL, 6, oneSecondInterval},
	{"GET /api/v5/asset/subaccount/managed-subaccount-bills", managedSubAccountTransferEPL, 6, oneSecondInterval},
	{"POST /api/v5/asset/subaccount/transfer", masterAccountsManageTransfersBetweenSubaccountEPL, 1, oneSecondInterval},
	{"POST /api/v5/users/subaccount/set-transfer-out", setPermissionOfTransferOutEPL, 1, oneSecondInterval},
	{"GET /api/v5/users/entrust-subaccount-list", getCustodyTradingSubaccountListEPL, 1, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/offers", getOfferEPL, 3, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/purchase", purchaseEPL, 2, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/redeem", redeemEPL, 2, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/cancel", cancelPurchaseOrRedemptionEPL, 2, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/orders-active", getEarnActiveOrdersEPL, 3, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/orders-history", getFundingOrderHistoryEPL, 3, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/eth/product-info", getProductInfoEPL, 3, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/eth/purchase", purchaseETHStakingEPL, 2, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/eth/redeem", redeemETHStakingEPL, 2, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/eth/cancel-redeem", cancelETHStakingRedemptionEPL, 2, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/eth/balance", getBETHBalanceEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/eth/purchase-redeem-history", getPurchaseRedeemHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/eth/apy-history", getAPYHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/sol/product-info", getSOLStakingProductInfoEPL, 3, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/sol/purchase", purchaseSOLStakingEPL, 2, oneSecondInterval},
	{"POST /api/v5/finance/staking-defi/sol/redeem", redeemSOLStakingEPL, 2, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/sol/balance", getSOLStakingBalanceEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/sol/purchase-redeem-history", getSOLStakingPurchaseRedeemHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/staking-defi/sol/apy-history", getSOLStakingAPYHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/stable-rewards/product-info", getStableRewardsProductInfoEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/stable-rewards/balance", getStableRewardsBalanceEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/stable-rewards/apy-history", getStableRewardsAPYHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/okusd/limits", getOKUSDLimitsEPL, 2, twoSecondsInterval},
	{"POST /api/v5/finance/okusd/subscribe", subscribeOKUSDEPL, 1, twoSecondsInterval},
	{"POST /api/v5/finance/okusd/redeem", redeemOKUSDEPL, 1, twoSecondsInterval},
	{"GET /api/v5/finance/okusd/account", getOKUSDAccountEPL, 2, twoSecondsInterval},
	{"GET /api/v5/finance/okusd/subscribe/history", getOKUSDSubscriptionHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/okusd/redeem/history", getOKUSDRedemptionHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/okusd/rewards/history", getOKUSDRewardsHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/okusd/rate/history", getOKUSDAPRHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/savings/balance", getSavingBalanceEPL, 6, oneSecondInterval},
	{"POST /api/v5/finance/savings/purchase-redempt", savingsPurchaseRedemptionEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/savings/lending-history", getLendingHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/savings/lending-rate-summary", getPublicBorrowInfoEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/savings/lending-rate-history", getPublicBorrowHistoryEPL, 6, oneSecondInterval},
	{"GET /api/v5/finance/flexible-loan/borrow-currencies", getFlexibleLoanBorrowableCurrenciesEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/collateral-assets", getFlexibleLoanCollateralAssetsEPL, 5, twoSecondsInterval},
	{"POST /api/v5/finance/flexible-loan/max-loan", getFlexibleLoanMaximumLoanAmountEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/max-collateral-redeem-amount", getFlexibleLoanMaximumCollateralRedeemAmountEPL, 5, twoSecondsInterval},
	{"POST /api/v5/finance/flexible-loan/adjust-collateral", adjustFlexibleLoanCollateralEPL, 5, twoSecondsInterval},
	{"POST /api/v5/finance/flexible-loan/borrow", borrowFlexibleLoanEPL, 2, twoSecondsInterval},
	{"POST /api/v5/finance/flexible-loan/repay", repayFlexibleLoanEPL, 2, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/emode-info", getFlexibleLoanEModeInfoEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/loan-info", getFlexibleLoanInfoEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/loan-history", getFlexibleLoanHistoryEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/flexible-loan/interest-accrued", getFlexibleLoanAccruedInterestEPL, 5, twoSecondsInterval},
	{"GET /api/v5/finance/sfp/dcd/currency-pair", getDualInvestmentCurrencyPairsEPL, 1, oneSecondInterval},
	{"GET /api/v5/finance/sfp/dcd/products", getDualInvestmentProductsEPL, 1, oneSecondInterval},
	{"POST /api/v5/finance/sfp/dcd/quote", requestDualInvestmentQuoteEPL, 10, time.Minute},
	{"POST /api/v5/finance/sfp/dcd/trade", tradeDualInvestmentEPL, 2, time.Minute},
	{"POST /api/v5/finance/sfp/dcd/redeem-quote", requestDualInvestmentRedeemQuoteEPL, 10, time.Minute},
	{"POST /api/v5/finance/sfp/dcd/redeem", redeemDualInvestmentEPL, 2, time.Minute},
	{"GET /api/v5/finance/sfp/dcd/order-status", getDualInvestmentOrderStateEPL, 3, oneSecondInterval},
	{"GET /api/v5/finance/sfp/dcd/order-history", getDualInvestmentOrderHistoryEPL, 1, oneSecondInterval},
	{"GET /api/v5/affiliate/performance/summary", getAffiliatePerformanceSummaryEPL, 3, oneSecondInterval},
	{"GET /api/v5/affiliate/invitee/detail", getInviteeDetailEPL, 3, oneSecondInterval},
	{"GET /api/v5/affiliate/invitee/list", getInviteeListEPL, 3, oneSecondInterval},
	{"GET /api/v5/affiliate/link/list", getAffiliateLinkListEPL, 3, oneSecondInterval},
	{"GET /api/v5/affiliate/co-inviter/list", getCoInviterLinkListEPL, 3, oneSecondInterval},
	{"GET /api/v5/affiliate/sub-affiliate/list", getSubAffiliateListEPL, 3, oneSecondInterval},
	{"GET /api/v5/system/status", getEventStatusEPL, 1, fiveSecondsInterval},
	{"GET /api/v5/support/announcements", getAnnouncementsEPL, 5, twoSecondsInterval},
	{"GET /api/v5/support/announcement-types", getAnnouncementTypeEPL, 1, twoSecondsInterval},
	// No longer documented; superseded by GET /api/v5/affiliate/invitee/detail
	{"GET /api/v5/users/partner/if-rebate", getUserAffiliateRebateInformationEPL, 20, twoSecondsInterval},
}
