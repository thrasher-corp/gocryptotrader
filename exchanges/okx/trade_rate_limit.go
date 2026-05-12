package okx

import (
	"context"
	"fmt"
	"math"
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

func (e *Exchange) applyTradeScopeRateLimit(ctx context.Context, class tradeRateLimitClass, counts map[string]int) error {
	if len(counts) == 0 {
		return nil
	}

	for scope, weight := range counts {
		if weight < 1 {
			continue
		}
		rl := e.getOrCreateTradeScopedLimiter(class, scope)
		if err := rl.RateLimit(request.WithRateLimitWeight(ctx, toRateLimitWeight(weight))); err != nil {
			return fmt.Errorf("trade rate limit class=%s scope=%s: %w", class, scope, err)
		}
	}
	return nil
}

func (e *Exchange) applyTradeSubAccountRateLimit(ctx context.Context, orderCount int) error {
	if orderCount < 1 {
		return nil
	}
	rlAny, _ := e.tradeSubAccountLimiter.LoadOrStore(
		"structural-subaccount-limit",
		request.NewRateLimitWithWeight(twoSecondsInterval, 1000, 1),
	)
	rl, ok := rlAny.(*request.RateLimiterWithWeight)
	if !ok {
		return fmt.Errorf("invalid subaccount limiter type: %T", rlAny)
	}
	return rl.RateLimit(request.WithRateLimitWeight(ctx, toRateLimitWeight(orderCount)))
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

func isOptionInstrumentID(instrumentID string) bool {
	parts := strings.Split(instrumentID, "-")
	if len(parts) >= 5 {
		return true
	}
	parts = strings.Split(instrumentID, "_")
	return len(parts) >= 5
}

func tradeScopeCountsFromPlaceOrders(args []PlaceOrderRequestParam) map[string]int {
	counts := make(map[string]int)
	for i := range args {
		scope := tradeScopeFromInstrumentID(args[i].InstrumentID)
		if scope == "" {
			continue
		}
		counts[scope]++
	}
	return counts
}

func tradeScopeCountsFromCancelOrders(args []CancelOrderRequestParam) map[string]int {
	counts := make(map[string]int)
	for i := range args {
		scope := tradeScopeFromInstrumentID(args[i].InstrumentID)
		if scope == "" {
			continue
		}
		counts[scope]++
	}
	return counts
}

func tradeScopeCountsFromAmendOrders(args []AmendOrderRequestParams) map[string]int {
	counts := make(map[string]int)
	for i := range args {
		scope := tradeScopeFromInstrumentID(args[i].InstrumentID)
		if scope == "" {
			continue
		}
		counts[scope]++
	}
	return counts
}

func countByOrder[T any](args []T) int {
	return len(args)
}

func toRateLimitWeight(value int) uint8 {
	if value <= 0 {
		return 0
	}
	if value > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(value)
}
