package okx

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

type tradeRateLimitClass string

type tradeRateLimiter struct {
	scopedLimitersLock sync.Mutex
	scopedLimiters     map[string]*request.RateLimiterWithWeight
	scopedLimiterKeys  []string
	subAccountLock     sync.Mutex
	subAccountLimiter  *request.RateLimiterWithWeight
}

const (
	maxOKXBatchOrders = 20
	// OKX scoped trade limiters are keyed by user-supplied instrument IDs or
	// option families. Keep the cache bounded in case callers pass many bad or
	// short-lived scope values.
	maxTradeScopedLimiters = 1024

	tradeRateLimitPlaceSingle  tradeRateLimitClass = "place-single"
	tradeRateLimitPlaceBatch   tradeRateLimitClass = "place-batch"
	tradeRateLimitCancelSingle tradeRateLimitClass = "cancel-single"
	tradeRateLimitCancelBatch  tradeRateLimitClass = "cancel-batch"
	tradeRateLimitAmendSingle  tradeRateLimitClass = "amend-single"
	tradeRateLimitAmendBatch   tradeRateLimitClass = "amend-batch"
)

func validateOKXBatchOrderCount(count int) error {
	if count > maxOKXBatchOrders {
		return fmt.Errorf("%w, cannot process more than %d orders", errExceedLimit, maxOKXBatchOrders)
	}
	return nil
}

func (l *tradeRateLimiter) getOrCreateScopedLimiter(class tradeRateLimitClass, scope string) *request.RateLimiterWithWeight {
	key := string(class) + "|" + strings.ToUpper(strings.TrimSpace(scope))
	l.scopedLimitersLock.Lock()
	defer l.scopedLimitersLock.Unlock()
	if l.scopedLimiters == nil {
		l.scopedLimiters = make(map[string]*request.RateLimiterWithWeight, maxTradeScopedLimiters)
	}
	if rl := l.scopedLimiters[key]; rl != nil {
		return rl
	}

	actions := 60
	if class == tradeRateLimitPlaceBatch ||
		class == tradeRateLimitCancelBatch ||
		class == tradeRateLimitAmendBatch {
		actions = 300
	}
	rl := request.NewRateLimitWithWeight(twoSecondsInterval, actions, 1)
	if len(l.scopedLimiterKeys) >= maxTradeScopedLimiters {
		// Drop the oldest limiter when the cache is full. These scopes come from
		// request values, so this stops bad or short-lived values growing the map
		// forever. If a real scope is evicted, it can be recreated cheaply.
		delete(l.scopedLimiters, l.scopedLimiterKeys[0])
		l.scopedLimiterKeys = l.scopedLimiterKeys[1:]
	}
	l.scopedLimiters[key] = rl
	l.scopedLimiterKeys = append(l.scopedLimiterKeys, key)
	return rl
}

func (l *tradeRateLimiter) additionalTradeRateLimits(class tradeRateLimitClass, counts map[string]int, subAccountOrderCount int) []request.RateLimitReservation {
	// OKX trade requests can be limited in three ways: the static REST or
	// websocket endpoint limit, an instrument/family limit, and the shared
	// subaccount limit. The request-specific limits are only returned when they
	// apply because their weights depend on the request contents.
	additionalRateLimits := l.additionalTradeScopeRateLimits(class, counts)
	if limit, ok := l.subAccountRateLimit(subAccountOrderCount); ok {
		additionalRateLimits = append(additionalRateLimits, limit)
	}
	return additionalRateLimits
}

func (l *tradeRateLimiter) additionalTradeScopeRateLimits(class tradeRateLimitClass, counts map[string]int) []request.RateLimitReservation {
	if len(counts) == 0 {
		return nil
	}

	additionalRateLimits := make([]request.RateLimitReservation, 0, len(counts))
	for scope, weight := range counts {
		if weight < 1 {
			continue
		}
		additionalRateLimits = append(additionalRateLimits, request.RateLimitReservation{
			Limiter: l.getOrCreateScopedLimiter(class, scope),
			Weight:  request.Weight(rateLimitWeight(weight)),
		})
	}
	return additionalRateLimits
}

func (l *tradeRateLimiter) subAccountRateLimit(orderCount int) (request.RateLimitReservation, bool) {
	if orderCount < 1 {
		return request.RateLimitReservation{}, false
	}
	l.subAccountLock.Lock()
	defer l.subAccountLock.Unlock()
	if l.subAccountLimiter == nil {
		l.subAccountLimiter = newTradeSubAccountRateLimiter()
	}
	return request.RateLimitReservation{
		Limiter: l.subAccountLimiter,
		Weight:  request.Weight(rateLimitWeight(orderCount)),
	}, true
}

func tradeScopeFromInstrumentID(instrumentID string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(instrumentID))
	if trimmed == "" {
		return ""
	}
	if isOptionInstrumentID(trimmed) {
		_, family := optionInstrumentSelectors(trimmed)
		return family
	}
	return trimmed
}

func optionInstrumentSelectors(instrumentID string) (underlying, family string) {
	parts, delimiter := strings.Split(instrumentID, "-"), "-"
	if len(parts) < 2 {
		parts, delimiter = strings.Split(instrumentID, "_"), "_"
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

func rateLimitWeight(value int) uint8 {
	if value <= 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}
