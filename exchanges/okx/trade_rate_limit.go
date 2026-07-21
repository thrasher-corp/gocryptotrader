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
	subAccountLimiter  *request.RateLimiterWithWeight
}

var (
	errTradeRateLimiterNotInitialised = errors.New("trade rate limiter not initialised")
	errMissingTradeRateLimitScope     = errors.New("missing trade rate limit scope")
	errInvalidTradeRateLimitClass     = errors.New("invalid trade rate limit class")
	errInvalidTradeRateLimitWeight    = errors.New("invalid trade rate limit weight")
)

const (
	maxBatchOrders                  = 20
	singleTradeRateLimitActions     = 60
	batchTradeRateLimitActions      = 300
	subAccountTradeRateLimitActions = 1000

	tradeRateLimitPlaceSingle  tradeRateLimitClass = "place-single"
	tradeRateLimitPlaceBatch   tradeRateLimitClass = "place-batch"
	tradeRateLimitCancelSingle tradeRateLimitClass = "cancel-single"
	tradeRateLimitCancelBatch  tradeRateLimitClass = "cancel-batch"
	tradeRateLimitAmendSingle  tradeRateLimitClass = "amend-single"
	tradeRateLimitAmendBatch   tradeRateLimitClass = "amend-batch"
)

var tradeRateLimitActions = map[tradeRateLimitClass]int{
	tradeRateLimitPlaceSingle:  singleTradeRateLimitActions,
	tradeRateLimitPlaceBatch:   batchTradeRateLimitActions,
	tradeRateLimitCancelSingle: singleTradeRateLimitActions,
	tradeRateLimitCancelBatch:  batchTradeRateLimitActions,
	tradeRateLimitAmendSingle:  singleTradeRateLimitActions,
	tradeRateLimitAmendBatch:   batchTradeRateLimitActions,
}

func batchOrderWeight(count int) (request.Weight, error) {
	if count > maxBatchOrders {
		return 0, fmt.Errorf("%w, cannot process more than %d orders", errExceedLimit, maxBatchOrders)
	}
	return rateLimitWeight(count)
}

func rateLimitWeight(count int) (request.Weight, error) {
	if count < 1 || count > math.MaxUint8 {
		return 0, errInvalidTradeRateLimitWeight
	}
	return request.Weight(count), nil
}

func (l *tradeRateLimiter) getOrCreateScopedLimiter(class tradeRateLimitClass, scope string) (*request.RateLimiterWithWeight, error) {
	if scope == "" {
		return nil, errMissingTradeRateLimitScope
	}
	actions, ok := tradeRateLimitActions[class]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errInvalidTradeRateLimitClass, class)
	}
	key := tradeRateLimitKey{
		class: class,
		scope: scope,
	}
	l.scopedLimitersLock.Lock()
	defer l.scopedLimitersLock.Unlock()
	if l.scopedLimiters == nil {
		return nil, errTradeRateLimiterNotInitialised
	}
	if cached := l.scopedLimiters[key]; cached != nil {
		return cached, nil
	}

	rl := request.NewRateLimitWithWeight(twoSecondsInterval, actions, 1)
	l.scopedLimiters[key] = rl
	return rl, nil
}

func (l *tradeRateLimiter) additionalTradeRateLimits(class tradeRateLimitClass, counts map[string]int) ([]request.RateLimitWithWeightOverride, error) {
	// OKX trade requests can be limited in three ways: the static REST or
	// websocket endpoint limit, an instrument/family limit, and the shared
	// subaccount limit. The request-specific limits are only returned when they
	// apply because their weights depend on the request contents.
	additionalRateLimits, orderCount, err := l.additionalTradeScopeRateLimits(class, counts)
	if err != nil {
		return nil, err
	}
	switch class {
	case tradeRateLimitPlaceSingle, tradeRateLimitPlaceBatch, tradeRateLimitAmendSingle, tradeRateLimitAmendBatch:
		if limit, ok, err := l.subAccountRateLimit(orderCount); err != nil {
			return nil, err
		} else if ok {
			additionalRateLimits = append(additionalRateLimits, limit)
		}
	case tradeRateLimitCancelSingle, tradeRateLimitCancelBatch:
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidTradeRateLimitClass, class)
	}
	return additionalRateLimits, nil
}

func (l *tradeRateLimiter) additionalTradeScopeRateLimits(class tradeRateLimitClass, counts map[string]int) ([]request.RateLimitWithWeightOverride, int, error) {
	if len(counts) == 0 {
		return nil, 0, errMissingTradeRateLimitScope
	}
	if _, ok := tradeRateLimitActions[class]; !ok {
		return nil, 0, fmt.Errorf("%w: %s", errInvalidTradeRateLimitClass, class)
	}

	weights := make(map[string]request.Weight, len(counts))
	orderCount := 0
	for scope, count := range counts {
		weight, err := rateLimitWeight(count)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: %s", err, scope)
		}
		weights[scope] = weight
		orderCount += count
	}
	// OKX documents that a one-order batch request consumes the corresponding
	// single-order bucket instead of the batch bucket:
	// https://www.okx.com/docs-v5/en/#order-book-trading-trade-post-place-multiple-orders
	if orderCount == 1 {
		switch class {
		case tradeRateLimitPlaceBatch:
			class = tradeRateLimitPlaceSingle
		case tradeRateLimitCancelBatch:
			class = tradeRateLimitCancelSingle
		case tradeRateLimitAmendBatch:
			class = tradeRateLimitAmendSingle
		}
	}
	additionalRateLimits := make([]request.RateLimitWithWeightOverride, 0, len(counts))
	for scope, weight := range weights {
		limiter, err := l.getOrCreateScopedLimiter(class, scope)
		if err != nil {
			return nil, 0, err
		}
		additionalRateLimits = append(additionalRateLimits, request.RateLimitWithWeightOverride{
			Limiter:        limiter,
			WeightOverride: weight,
		})
	}
	return additionalRateLimits, orderCount, nil
}

func (l *tradeRateLimiter) subAccountRateLimit(orderCount int) (request.RateLimitWithWeightOverride, bool, error) {
	if orderCount < 1 {
		return request.RateLimitWithWeightOverride{}, false, nil
	}
	weightOverride, err := rateLimitWeight(orderCount)
	if err != nil {
		return request.RateLimitWithWeightOverride{}, false, fmt.Errorf("%w: subaccount order count %d", err, orderCount)
	}
	if l.subAccountLimiter == nil {
		return request.RateLimitWithWeightOverride{}, false, errTradeRateLimiterNotInitialised
	}
	return request.RateLimitWithWeightOverride{
		Limiter:        l.subAccountLimiter,
		WeightOverride: weightOverride,
	}, true, nil
}

func tradeScopeFromInstrumentID(instrumentID string) (string, error) {
	if strings.TrimSpace(instrumentID) == "" {
		return "", errMissingTradeRateLimitScope
	}
	if isOptionInstrumentID(instrumentID) {
		return optionInstrumentFamily(instrumentID)
	}
	return instrumentID, nil
}

func optionInstrumentFamily(instrumentID string) (string, error) {
	delimiter := "-"
	parts := strings.Split(instrumentID, delimiter)
	if len(parts) < 2 {
		delimiter = "_"
		parts = strings.Split(instrumentID, delimiter)
		if len(parts) < 2 {
			return "", fmt.Errorf("%w: %s", errMissingTradeRateLimitScope, instrumentID)
		}
	}
	return strings.Join(parts[:2], delimiter), nil
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
		scope, err := tradeScopeFromInstrumentID(instrumentID(arg))
		if err != nil {
			return nil, err
		}
		counts[scope]++
	}
	return counts, nil
}
