package okx

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

type tradeRateLimitClass string

type tradeRateLimitKey struct {
	class tradeRateLimitClass
	scope string
}

type tradeRateLimiter struct {
	scopedLimitersLock sync.Mutex
	scopedLimiters     map[tradeRateLimitKey]*request.RateLimiterWithWeight
	subAccountLock     sync.Mutex
	subAccountLimiter  *request.RateLimiterWithWeight
}

var (
	errTradeRateLimiterNotInitialised = errors.New("trade rate limiter not initialised")
	errMissingTradeRateLimitScope     = errors.New("missing trade rate limit scope")
	errInvalidTradeRateLimitWeight    = errors.New("invalid trade rate limit weight")
)

const (
	maxBatchOrders = 20

	tradeRateLimitPlaceSingle  tradeRateLimitClass = "place-single"
	tradeRateLimitPlaceBatch   tradeRateLimitClass = "place-batch"
	tradeRateLimitCancelSingle tradeRateLimitClass = "cancel-single"
	tradeRateLimitCancelBatch  tradeRateLimitClass = "cancel-batch"
	tradeRateLimitAmendSingle  tradeRateLimitClass = "amend-single"
	tradeRateLimitAmendBatch   tradeRateLimitClass = "amend-batch"
)

func validateBatchOrderCount(count int) error {
	if count > maxBatchOrders {
		return fmt.Errorf("%w, cannot process more than %d orders", errExceedLimit, maxBatchOrders)
	}
	return nil
}

func (l *tradeRateLimiter) getOrCreateScopedLimiter(class tradeRateLimitClass, scope string) (*request.RateLimiterWithWeight, error) {
	normalisedScope := strings.ToUpper(strings.TrimSpace(scope))
	if normalisedScope == "" {
		return nil, errMissingTradeRateLimitScope
	}
	key := tradeRateLimitKey{
		class: class,
		scope: normalisedScope,
	}
	l.scopedLimitersLock.Lock()
	defer l.scopedLimitersLock.Unlock()
	if l.scopedLimiters == nil {
		return nil, errTradeRateLimiterNotInitialised
	}
	if rl := l.scopedLimiters[key]; rl != nil {
		return rl, nil
	}

	actions := 60
	if class == tradeRateLimitPlaceBatch ||
		class == tradeRateLimitCancelBatch ||
		class == tradeRateLimitAmendBatch {
		actions = 300
	}
	rl := request.NewRateLimitWithWeight(twoSecondsInterval, actions, 1)
	l.scopedLimiters[key] = rl
	return rl, nil
}

func (l *tradeRateLimiter) additionalTradeRateLimits(class tradeRateLimitClass, counts map[string]int, subAccountOrderCount int) ([]request.RateLimitWithWeightOverride, error) {
	// OKX trade requests can be limited in three ways: the static REST or
	// websocket endpoint limit, an instrument/family limit, and the shared
	// subaccount limit. The request-specific limits are only returned when they
	// apply because their weights depend on the request contents.
	additionalRateLimits, err := l.additionalTradeScopeRateLimits(class, counts)
	if err != nil {
		return nil, err
	}
	if limit, ok, err := l.subAccountRateLimit(subAccountOrderCount); err != nil {
		return nil, err
	} else if ok {
		additionalRateLimits = append(additionalRateLimits, limit)
	}
	return additionalRateLimits, nil
}

func (l *tradeRateLimiter) additionalTradeScopeRateLimits(class tradeRateLimitClass, counts map[string]int) ([]request.RateLimitWithWeightOverride, error) {
	if len(counts) == 0 {
		return nil, errMissingTradeRateLimitScope
	}

	additionalRateLimits := make([]request.RateLimitWithWeightOverride, 0, len(counts))
	for scope, weight := range counts {
		if weight < 1 {
			return nil, fmt.Errorf("%w: %s", errInvalidTradeRateLimitWeight, scope)
		}
		limiter, err := l.getOrCreateScopedLimiter(class, scope)
		if err != nil {
			return nil, err
		}
		additionalRateLimits = append(additionalRateLimits, request.RateLimitWithWeightOverride{
			Limiter:        limiter,
			WeightOverride: clampWeight(weight),
		})
	}
	return additionalRateLimits, nil
}

func (l *tradeRateLimiter) subAccountRateLimit(orderCount int) (request.RateLimitWithWeightOverride, bool, error) {
	if orderCount < 1 {
		return request.RateLimitWithWeightOverride{}, false, nil
	}
	l.subAccountLock.Lock()
	defer l.subAccountLock.Unlock()
	if l.subAccountLimiter == nil {
		return request.RateLimitWithWeightOverride{}, false, errTradeRateLimiterNotInitialised
	}
	return request.RateLimitWithWeightOverride{
		Limiter:        l.subAccountLimiter,
		WeightOverride: clampWeight(orderCount),
	}, true, nil
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
	delimiter := "-"
	parts := strings.Split(instrumentID, delimiter)
	if len(parts) < 2 {
		delimiter = "_"
		parts = strings.Split(instrumentID, delimiter)
		if len(parts) < 2 {
			return instrumentID, instrumentID
		}
	}
	underlying = strings.Join(parts[:2], delimiter)
	return underlying, underlying
}

func isOptionInstrumentID(instrumentID string) bool {
	return strings.Count(instrumentID, "-") >= 4 || strings.Count(instrumentID, "_") >= 4
}

func tradeScopeCountsFromPlaceOrders(args []PlaceOrderRequestParam) (map[string]int, error) {
	return tradeScopeCounts(args, func(arg PlaceOrderRequestParam) string { return arg.InstrumentID })
}

func tradeScopeCountsFromCancelOrders(args []CancelOrderRequestParam) (map[string]int, error) {
	return tradeScopeCounts(args, func(arg CancelOrderRequestParam) string { return arg.InstrumentID })
}

func tradeScopeCountsFromAmendOrders(args []AmendOrderRequestParams) (map[string]int, error) {
	return tradeScopeCounts(args, func(arg AmendOrderRequestParams) string { return arg.InstrumentID })
}

func tradeScopeCounts[T any](args []T, instrumentID func(T) string) (map[string]int, error) {
	counts := make(map[string]int)
	for _, arg := range args {
		scope := tradeScopeFromInstrumentID(instrumentID(arg))
		if scope == "" {
			return nil, errMissingTradeRateLimitScope
		}
		counts[scope]++
	}
	return counts, nil
}

func clampWeight(count int) request.Weight {
	return request.Weight(min(max(count, 0), math.MaxUint8))
}

func newTradeSubAccountRateLimiter() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(twoSecondsInterval, 1000, 1)
}
