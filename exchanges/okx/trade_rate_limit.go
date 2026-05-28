package okx

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

type tradeRateLimitClass string

const (
	tradeRateLimitPlaceSingle  tradeRateLimitClass = "place-single"
	tradeRateLimitPlaceBatch   tradeRateLimitClass = "place-batch"
	tradeRateLimitCancelSingle tradeRateLimitClass = "cancel-single"
	tradeRateLimitCancelBatch  tradeRateLimitClass = "cancel-batch"
	tradeRateLimitAmendSingle  tradeRateLimitClass = "amend-single"
	tradeRateLimitAmendBatch   tradeRateLimitClass = "amend-batch"
)

func (e *Exchange) getOrCreateTradeScopedLimiter(class tradeRateLimitClass, scope string) *request.RateLimiterWithWeight {
	key := string(class) + "|" + strings.ToUpper(strings.TrimSpace(scope))
	if rlAny, ok := e.tradeScopedLimiters.Load(key); ok {
		if rl, ok := rlAny.(*request.RateLimiterWithWeight); ok {
			return rl
		}
	}

	actions := 60
	if class == tradeRateLimitPlaceBatch ||
		class == tradeRateLimitCancelBatch ||
		class == tradeRateLimitAmendBatch {
		actions = 300
	}
	rl := request.NewRateLimitWithWeight(twoSecondsInterval, actions, 1)
	if existing, loaded := e.tradeScopedLimiters.LoadOrStore(key, rl); loaded {
		if got, ok := existing.(*request.RateLimiterWithWeight); ok {
			return got
		}
	}
	return rl
}

func (e *Exchange) tradeRateLimits(class tradeRateLimitClass, counts map[string]int, subAccountOrderCount int) []request.RateLimitReservation {
	limits := e.tradeScopeRateLimits(class, counts)
	if limit, ok := e.tradeSubAccountRateLimit(subAccountOrderCount); ok {
		limits = append(limits, limit)
	}
	return limits
}

func (e *Exchange) tradeScopeRateLimits(class tradeRateLimitClass, counts map[string]int) []request.RateLimitReservation {
	if len(counts) == 0 {
		return nil
	}

	limits := make([]request.RateLimitReservation, 0, len(counts))
	for scope, weight := range counts {
		if weight < 1 {
			continue
		}
		limits = append(limits, request.RateLimitReservation{
			Limiter: e.getOrCreateTradeScopedLimiter(class, scope),
			Weight:  request.Weight(toRateLimitWeight(weight)),
		})
	}
	return limits
}

func (e *Exchange) tradeSubAccountRateLimit(orderCount int) (request.RateLimitReservation, bool) {
	if orderCount < 1 {
		return request.RateLimitReservation{}, false
	}
	e.tradeSubAccountLock.Lock()
	defer e.tradeSubAccountLock.Unlock()
	if e.tradeSubAccountLimiter == nil {
		e.tradeSubAccountLimiter = newTradeSubAccountRateLimiter()
	}
	return request.RateLimitReservation{
		Limiter: e.tradeSubAccountLimiter,
		Weight:  request.Weight(toRateLimitWeight(orderCount)),
	}, true
}

func tradeScopeFromInstrumentID(instrumentID string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(instrumentID))
	if trimmed == "" {
		return ""
	}
	if isOptionInstrumentID(trimmed) {
		_, family := optionInstrumentSelectors(trimmed)
		return strings.ToUpper(strings.TrimSpace(family))
	}
	return trimmed
}

func optionInstrumentSelectors(instrumentID string) (underlying, family string) {
	parts := strings.Split(instrumentID, "-")
	delimiter := "-"
	if len(parts) < 2 {
		parts = strings.Split(instrumentID, "_")
		delimiter = "_"
	}
	if len(parts) < 2 {
		return instrumentID, instrumentID
	}
	underlying = strings.Join(parts[:2], delimiter)
	return underlying, underlying
}

func isOptionInstrumentID(instrumentID string) bool {
	return len(strings.Split(instrumentID, "-")) >= 5 || len(strings.Split(instrumentID, "_")) >= 5
}

func tradeScopeCountsFromPlaceOrders(args []PlaceOrderRequestParam) map[string]int {
	return tradeScopeCounts(args, func(arg PlaceOrderRequestParam) string { return arg.InstrumentID })
}

func tradeScopeCountsFromCancelOrders(args []CancelOrderRequestParam) map[string]int {
	return tradeScopeCounts(args, func(arg CancelOrderRequestParam) string { return arg.InstrumentID })
}

func tradeScopeCountsFromAmendOrders(args []AmendOrderRequestParams) map[string]int {
	return tradeScopeCounts(args, func(arg AmendOrderRequestParams) string { return arg.InstrumentID })
}

func tradeScopeCounts[T any](args []T, instrumentID func(T) string) map[string]int {
	counts := make(map[string]int)
	for _, arg := range args {
		if scope := tradeScopeFromInstrumentID(instrumentID(arg)); scope != "" {
			counts[scope]++
		}
	}
	return counts
}

func toRateLimitWeight(value int) uint8 {
	if value <= 0 {
		panic("rate limit weight must be positive")
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}
