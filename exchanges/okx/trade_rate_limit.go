package okx

import (
	"fmt"
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

type tradeRateLimits struct {
	endpointWeight request.Weight
	limiters       []*request.RateLimiterWithWeight
	weights        []request.Weight
}

type tradeRateLimitParams struct {
	class                tradeRateLimitClass
	counts               map[string]int
	subAccountOrderCount int
	endpointWeight       int
}

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

func (e *Exchange) tradeRateLimits(params tradeRateLimitParams) (*tradeRateLimits, error) {
	limiters, weights := e.tradeScopeRateLimits(params.class, params.counts)
	if params.subAccountOrderCount > 0 {
		limit, err := e.tradeSubAccountRateLimit(params.subAccountOrderCount)
		if err != nil {
			return nil, err
		}
		limiters = append(limiters, limit)
		weights = append(weights, request.Weight(toRateLimitWeight(params.subAccountOrderCount)))
	}
	limits := &tradeRateLimits{
		limiters: limiters,
		weights:  weights,
	}
	if params.endpointWeight > 0 {
		limits.endpointWeight = request.Weight(toRateLimitWeight(params.endpointWeight))
	}
	return limits, nil
}

func (e *Exchange) tradeScopeRateLimits(class tradeRateLimitClass, counts map[string]int) ([]*request.RateLimiterWithWeight, []request.Weight) {
	if len(counts) == 0 {
		return nil, nil
	}

	limiters := make([]*request.RateLimiterWithWeight, 0, len(counts))
	weights := make([]request.Weight, 0, len(counts))
	for scope, weight := range counts {
		if weight < 1 {
			continue
		}
		limiters = append(limiters, e.getOrCreateTradeScopedLimiter(class, scope))
		weights = append(weights, request.Weight(toRateLimitWeight(weight)))
	}
	return limiters, weights
}

func (e *Exchange) tradeSubAccountRateLimit(orderCount int) (*request.RateLimiterWithWeight, error) {
	if orderCount < 1 {
		return nil, nil
	}
	rlAny, _ := e.tradeSubAccountLimiter.LoadOrStore(
		"structural-subaccount-limit",
		request.NewRateLimitWithWeight(twoSecondsInterval, 1000, 1),
	)
	rl, ok := rlAny.(*request.RateLimiterWithWeight)
	if !ok {
		return nil, fmt.Errorf("invalid subaccount limiter type: %T", rlAny)
	}
	return rl, nil
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
