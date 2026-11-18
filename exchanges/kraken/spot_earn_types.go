package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errEarnStrategyIDRequired = errors.New("earn strategy ID is required")

// EarnStrategiesRequest holds the parameters of List Earn Strategies
type EarnStrategiesRequest struct {
	// Ascending sorts the strategies in ascending rather than descending order
	Ascending bool
	// Asset filters the strategies to an asset
	Asset currency.Code
	// Cursor requests the page a previous page's NextCursor names
	Cursor string
	// Limit is the number of strategies a page holds, up to 65535, though Kraken may cap it lower
	Limit uint64
	// LockTypes filters the strategies to lock types: flex, bonded, timed or instant
	LockTypes []string
}

// EarnStrategiesResponse holds a page of earn strategies
type EarnStrategiesResponse struct {
	Items []EarnStrategy `json:"items"`
	// NextCursor requests the next page as EarnStrategiesRequest's Cursor, and is empty on the last page
	NextCursor string `json:"next_cursor"`
}

// EarnStrategy is an earn strategy and its parameters
type EarnStrategy struct {
	AllocationFee types.Number `json:"allocation_fee"`
	// AllocationRestrictionInfo lists why the account cannot allocate to the strategy, such as tier when its
	// verification tier is too low
	AllocationRestrictionInfo []string `json:"allocation_restriction_info"`
	// APREstimate is the yearly yield range Kraken estimates from the strategy's past revenue, and nil when it gives none
	APREstimate     *EarnAPREstimate `json:"apr_estimate"`
	Asset           currency.Code    `json:"asset"`
	AutoCompound    EarnAutoCompound `json:"auto_compound"`
	CanAllocate     bool             `json:"can_allocate"`
	CanDeallocate   bool             `json:"can_deallocate"`
	DeallocationFee types.Number     `json:"deallocation_fee"`
	ID              string           `json:"id"`
	LockType        EarnLockType     `json:"lock_type"`
	// UserCap is the most any user may allocate to the strategy, nil when there is no cap, and zero when the strategy
	// refuses new allocations, though auto compounding continues
	UserCap *types.Number `json:"user_cap"`
	// UserMinimumAllocation is the least an allocation or deallocation may be, in USD, and zero when there is no minimum
	UserMinimumAllocation types.Number    `json:"user_min_allocation"`
	YieldSource           EarnYieldSource `json:"yield_source"`
}

// EarnAPREstimate is an earn strategy's estimated yearly yield range, in percent
type EarnAPREstimate struct {
	High types.Number `json:"high"`
	Low  types.Number `json:"low"`
}

// EarnAutoCompound is whether an earn strategy compounds rewards. Type is disabled when it never does, enabled when it
// always does, or optional when it follows the account's preference, which defaults to Default
type EarnAutoCompound struct {
	Default bool   `json:"default"`
	Type    string `json:"type"`
}

// EarnLockType is how an earn strategy holds funds. Type is flex for Kraken Rewards, which earns on spot balances once
// enabled for the whole account and cannot be allocated to directly; bonded for a strategy with bonding and unbonding
// periods; or instant for a bonded strategy without them. Only bonded strategies set the bonding, unbonding and exit
// queue fields, and only bonded and instant strategies set PayoutFrequencySeconds
type EarnLockType struct {
	BondingPeriodSeconds uint64 `json:"bonding_period"`
	// BondingPeriodVariable is set when the bonding period's length varies
	BondingPeriodVariable bool `json:"bonding_period_variable"`
	// BondingRewards is set when funds earn while bonding, paid once bonding completes
	BondingRewards bool `json:"bonding_rewards"`
	// ExitQueuePeriodSeconds is how long deallocated funds wait in an exit queue, still earning, before they start
	// unbonding; there is no exit queue when it is 0
	ExitQueuePeriodSeconds uint64 `json:"exit_queue_period"`
	// PayoutFrequencySeconds is the interval at which rewards are credited to the account's ledger
	PayoutFrequencySeconds uint64 `json:"payout_frequency"`
	Type                   string `json:"type"`
	UnbondingPeriodSeconds uint64 `json:"unbonding_period"`
	// UnbondingPeriodVariable is set when the unbonding period's length varies
	UnbondingPeriodVariable bool `json:"unbonding_period_variable"`
	// UnbondingRewards is set when funds earn, and are paid, while unbonding
	UnbondingRewards bool `json:"unbonding_rewards"`
}

// EarnYieldSource is how an earn strategy generates yield. Type is staking for on-chain proof of stake, or off_chain
// for another mechanism
type EarnYieldSource struct {
	Type string `json:"type"`
}

// EarnAllocationsRequest holds the parameters of List Earn Allocations
type EarnAllocationsRequest struct {
	// Ascending sorts the allocations in ascending rather than descending order
	Ascending bool
	// ConvertedAsset is the asset allocations are valued in, USD when it is empty
	ConvertedAsset currency.Code
	// HideZeroAllocations omits the strategies the account used before but no longer holds anything in
	HideZeroAllocations bool
}

// EarnAllocationsResponse holds the account's earn allocations
type EarnAllocationsResponse struct {
	// ConvertedAsset is the asset every converted amount is valued in
	ConvertedAsset currency.Code    `json:"converted_asset"`
	Items          []EarnAllocation `json:"items"`
	// TotalAllocated is the amount allocated across every strategy, valued in ConvertedAsset
	TotalAllocated types.Number `json:"total_allocated"`
	// TotalRewarded is what every strategy has earned over the account's lifetime, valued in ConvertedAsset
	TotalRewarded types.Number `json:"total_rewarded"`
}

// EarnAllocation is the account's allocation to an earn strategy
type EarnAllocation struct {
	AmountAllocated EarnAmountAllocated `json:"amount_allocated"`
	NativeAsset     currency.Code       `json:"native_asset"`
	// Payout is the current payout period, and nil when there is none
	Payout     *EarnPayout `json:"payout"`
	StrategyID string      `json:"strategy_id"`
	// TotalRewarded is what the strategy has earned over the account's lifetime
	TotalRewarded EarnAmount `json:"total_rewarded"`
}

// EarnAmountAllocated is the amount allocated to an earn strategy, by state. Funds in Total that are not bonding, in
// the exit queue or unbonding are allocated and earning; whether bonding and unbonding funds earn depends on the
// strategy's lock type, while funds in the exit queue always earn. Bonding, ExitQueue, Pending and Unbonding are nil
// while they hold nothing
type EarnAmountAllocated struct {
	Bonding   *EarnAllocationState `json:"bonding"`
	ExitQueue *EarnAllocationState `json:"exit_queue"`
	// Pending is the amount being allocated, which is negative while a deallocation is pending
	Pending   *EarnAmount          `json:"pending"`
	Total     EarnAmount           `json:"total"`
	Unbonding *EarnAllocationState `json:"unbonding"`
}

// EarnAmount is an amount in its native asset and valued in the converted asset
type EarnAmount struct {
	ConvertedAmount types.Number `json:"converted"`
	NativeAmount    types.Number `json:"native"`
}

// EarnAllocationState holds the allocations to an earn strategy that are bonding, in the exit queue or unbonding
type EarnAllocationState struct {
	AllocationCount uint64                     `json:"allocation_count"`
	Allocations     []EarnAllocationTransition `json:"allocations"`
	ConvertedAmount types.Number               `json:"converted"`
	NativeAmount    types.Number               `json:"native"`
}

// EarnAllocationTransition is an allocation and when it moves to its next state. Estimates can be inaccurate for a
// minute or two after funds are allocated or deallocated
type EarnAllocationTransition struct {
	ConvertedAmount types.Number `json:"converted"`
	// CreatedAt is when the allocation or deallocation was received, or when deallocated funds joined the exit queue of
	// a strategy that has one
	CreatedAt time.Time `json:"created_at"`
	// Expires is when the allocation moves to its next state, or for ETH in the exit queue, when it finishes unbonding
	Expires      time.Time    `json:"expires"`
	NativeAmount types.Number `json:"native"`
}

// EarnPayout is an earn allocation's current payout period
type EarnPayout struct {
	// AccumulatedReward is what the period has earned so far
	AccumulatedReward EarnAmount `json:"accumulated_reward"`
	// EstimatedReward is what the period is estimated to earn from now until the payout
	EstimatedReward EarnAmount `json:"estimated_reward"`
	// PeriodEnd is the tentative time of the next payout
	PeriodEnd time.Time `json:"period_end"`
	// PeriodStart is the time of the last payout, or when the period was enabled
	PeriodStart time.Time `json:"period_start"`
}

// EarnAllocationStatusResponse holds the status of the last allocation to an earn strategy
type EarnAllocationStatusResponse struct {
	// Pending is set while an operation on the strategy is still in progress
	Pending bool `json:"pending"`
}

// EarnDeallocationStatusResponse holds the status of the last deallocation from an earn strategy
type EarnDeallocationStatusResponse struct {
	// Pending is set while an operation on the strategy is still in progress
	Pending bool `json:"pending"`
}
