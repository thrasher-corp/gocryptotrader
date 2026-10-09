package kraken

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
)

// ListEarnStrategies calls List Earn Strategies, returning the strategies available in the account's region. A
// strategy the account's verification tier is too low for has CanAllocate unset and tier among its
// AllocationRestrictionInfo. Kraken does not page strategies yet, so the first page holds them all. A nil request
// returns every strategy
func (e *Exchange) ListEarnStrategies(ctx context.Context, req *EarnStrategiesRequest) (*EarnStrategiesResponse, error) {
	body := make(map[string]any)
	if req != nil {
		if req.Limit > math.MaxUint16 {
			return nil, fmt.Errorf("%w: %d exceeds %d", errInvalidCount, req.Limit, math.MaxUint16)
		}
		if req.Ascending {
			body["ascending"] = true
		}
		if !req.Asset.IsEmpty() {
			body["asset"] = req.Asset.Upper().String()
		}
		if req.Cursor != "" {
			body["cursor"] = req.Cursor
		}
		if req.Limit != 0 {
			body["limit"] = req.Limit
		}
		if len(req.LockTypes) != 0 {
			body["lock_type"] = req.LockTypes
		}
	}
	var resp *EarnStrategiesResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/Strategies", nil, body, &resp)
}

// ListEarnAllocations calls List Earn Allocations, returning the account's allocation to each strategy it has used,
// including those it no longer holds anything in unless HideZeroAllocations is set. A nil request returns every
// allocation, valued in USD
func (e *Exchange) ListEarnAllocations(ctx context.Context, req *EarnAllocationsRequest) (*EarnAllocationsResponse, error) {
	body := make(map[string]any)
	if req != nil {
		if req.Ascending {
			body["ascending"] = true
		}
		if !req.ConvertedAsset.IsEmpty() {
			body["converted_asset"] = req.ConvertedAsset.Upper().String()
		}
		if req.HideZeroAllocations {
			body["hide_zero_allocations"] = true
		}
	}
	var resp *EarnAllocationsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/Allocations", nil, body, &resp)
}

// AllocateEarnFunds calls Allocate Earn Funds, allocating an amount of a strategy's asset to it. The allocation
// completes asynchronously, so GetEarnAllocationStatus must be polled for its outcome, and a strategy takes one
// allocation or deallocation at a time
func (e *Exchange) AllocateEarnFunds(ctx context.Context, strategyID string, amount float64) (bool, error) {
	body, err := earnOperationBody(strategyID, amount)
	if err != nil {
		return false, err
	}
	var resp bool
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/Allocate", nil, body, &resp)
}

// DeallocateEarnFunds calls Deallocate Earn Funds, deallocating an amount of a strategy's asset from it. The
// deallocation completes asynchronously, so GetEarnDeallocationStatus must be polled for its outcome, and a strategy
// takes one allocation or deallocation at a time
func (e *Exchange) DeallocateEarnFunds(ctx context.Context, strategyID string, amount float64) (bool, error) {
	body, err := earnOperationBody(strategyID, amount)
	if err != nil {
		return false, err
	}
	var resp bool
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/Deallocate", nil, body, &resp)
}

// GetEarnAllocationStatus calls Get Allocation Status, reporting whether the last allocation to a strategy is still in
// progress. An allocation that failed returns its error here, as if from the original request
func (e *Exchange) GetEarnAllocationStatus(ctx context.Context, strategyID string) (*EarnAllocationStatusResponse, error) {
	if strategyID == "" {
		return nil, errEarnStrategyIDRequired
	}
	var resp *EarnAllocationStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/AllocateStatus", nil, map[string]any{"strategy_id": strategyID}, &resp)
}

// GetEarnDeallocationStatus calls Get Deallocation Status, reporting whether the last deallocation from a strategy is
// still in progress. A deallocation that failed returns its error here, as if from the original request
func (e *Exchange) GetEarnDeallocationStatus(ctx context.Context, strategyID string) (*EarnDeallocationStatusResponse, error) {
	if strategyID == "" {
		return nil, errEarnStrategyIDRequired
	}
	var resp *EarnDeallocationStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Earn/DeallocateStatus", nil, map[string]any{"strategy_id": strategyID}, &resp)
}

// earnOperationBody returns the parameters Allocate Earn Funds and Deallocate Earn Funds share
func earnOperationBody(strategyID string, amount float64) (map[string]any, error) {
	if strategyID == "" {
		return nil, errEarnStrategyIDRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	body["strategy_id"] = strategyID
	return body, nil
}
