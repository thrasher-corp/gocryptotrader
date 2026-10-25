package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Staking request statuses: a stake request moves from NEW through PENDING to STAKED, an unstake request also through
// PENDING_UNSTAKING and PENDING_WITHDRAWAL, and each ends COMPLETED or REJECTED
const (
	StakingStatusNew               = "NEW"
	StakingStatusPending           = "PENDING"
	StakingStatusPendingWithdrawal = "PENDING_WITHDRAWAL"
	StakingStatusPendingUnstaking  = "PENDING_UNSTAKING"
	StakingStatusStaked            = "STAKED"
	StakingStatusCompleted         = "COMPLETED"
	StakingStatusRejected          = "REJECTED"
)

// Staking request sides
const (
	StakingSideStake   = "STAKE"
	StakingSideUnstake = "UNSTAKE"
)

// Staking convert request statuses: a convert request is NEW until it is COMPLETED or REJECTED
const (
	StakingConvertStatusNew       = "NEW"
	StakingConvertStatusCompleted = "COMPLETED"
	StakingConvertStatusRejected  = "REJECTED"
)

// Staking reward rate types, whether a staking instrument's estimated rewards are an annual percentage rate or an
// annual percentage yield, which compounds
const (
	StakingRewardRateTypeAPR = "APR"
	StakingRewardRateTypeAPY = "APY"
)

// stakingInstrumentParams is the params of a staking request naming one instrument
type stakingInstrumentParams struct {
	InstrumentName string `json:"instrument_name"`
}

// StakingConversionRateResponse is the conversion rate between a liquid staking token, such as CDCETH, and its staked
// token, such as ETH.staked
type StakingConversionRateResponse struct {
	InstrumentName string       `json:"instrument_name"`
	ConversionRate types.Number `json:"conversion_rate"`
}

// StakeRequest stakes a quantity of a staking instrument, such as SOL.staked
type StakeRequest struct {
	InstrumentName string       `json:"instrument_name"`
	Quantity       types.Number `json:"quantity,omitempty"`
}

// StakeResponse is a stake request as created, which Crypto.com processes asynchronously; the pre-stake charge is taken
// from the quantity staked
type StakeResponse struct {
	StakingID                     string              `json:"staking_id"`
	InstrumentName                string              `json:"instrument_name"`
	Status                        string              `json:"status"`
	Quantity                      types.PreciseNumber `json:"quantity"`
	UnderlyingCurrency            currency.Code       `json:"underlying_inst_name"`
	PreStakeChargeRateBasisPoints uint64              `json:"pre_stake_charge_rate_in_bps,string"`
	PreStakeCharge                types.PreciseNumber `json:"pre_stake_charge"`
	Reason                        string              `json:"reason"`
}

// UnstakeRequest unstakes a quantity of a staking instrument; for a yield-bearing instrument, such as TSTON.staked, the
// quantity is of the instrument itself, and the request completes in its underlying currency
type UnstakeRequest struct {
	InstrumentName string       `json:"instrument_name"`
	Quantity       types.Number `json:"quantity,omitempty"`
}

// UnstakeResponse is an unstake request as created, which Crypto.com processes asynchronously; for a yield-bearing
// instrument its quantity is of the underlying currency the request completes in
type UnstakeResponse struct {
	StakingID          string              `json:"staking_id"`
	InstrumentName     string              `json:"instrument_name"`
	Status             string              `json:"status"`
	Quantity           types.PreciseNumber `json:"quantity"`
	UnderlyingCurrency currency.Code       `json:"underlying_inst_name"`
	Reason             string              `json:"reason"`
}

// StakingPositionRequest filters the staking positions by instrument, returning every instrument's when unset
type StakingPositionRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
}

// StakingPositionResponse is the position in a staking instrument; for a yield-bearing instrument the staked, pending
// unstaked and reward eligible quantities are of the instrument held on-chain. The reward eligible quantity is what can
// be unstaked or converted
type StakingPositionResponse struct {
	InstrumentName          string              `json:"instrument_name"`
	UnderlyingCurrency      currency.Code       `json:"underlying_inst_name"`
	StakedQuantity          types.PreciseNumber `json:"staked_quantity"`
	PendingStakedQuantity   types.PreciseNumber `json:"pending_staked_quantity"`
	PendingUnstakedQuantity types.PreciseNumber `json:"pending_unstaked_quantity"`
	RewardEligibleQuantity  types.PreciseNumber `json:"reward_eligible_quantity"`
}

// StakingInstrumentResponse is an instrument that can be staked. OutOfStock is set while staking it is disabled and
// BlockUnstake while unstaking it is; its reward frequency and lock-up period are estimates in days, the frequency a
// fraction of a day for some instruments
type StakingInstrumentResponse struct {
	InstrumentName                string                    `json:"instrument_name"`
	UnderlyingCurrency            currency.Code             `json:"underlying_inst_name"`
	RewardInstrumentName          string                    `json:"reward_inst_name"`
	OutOfStock                    bool                      `json:"out_of_stock"`
	BlockUnstake                  bool                      `json:"block_unstake"`
	EstimatedRewards              types.Number              `json:"est_rewards"`
	RewardRateType                string                    `json:"apr_y"`
	MinimumStakeAmount            types.Number              `json:"min_stake_amt"`
	RewardFrequencyDays           types.Number              `json:"reward_frequency"`
	LockUpPeriodDays              types.Number              `json:"lock_up_period"`
	IsCompoundReward              bool                      `json:"is_compound_reward"`
	PreStakeChargeEnabled         bool                      `json:"pre_stake_charge_enable"`
	PreStakeChargeRateBasisPoints uint64                    `json:"pre_stake_charge_rate_in_bps,string"`
	IsRestaked                    bool                      `json:"is_restaked"`
	AdditionalRewards             []StakingAdditionalReward `json:"additional_rewards"`
}

// StakingAdditionalReward names an instrument a staking instrument also pays rewards in
type StakingAdditionalReward struct {
	RewardInstrumentName string `json:"reward_inst_name"`
}

// StakingHistoryRequest filters the open stake requests, the stake request history and the staking reward history: by
// instrument, and by a time window, both ends inclusive, that defaults to the 30 days before the end time, which
// defaults to now, and reaches back at most 180 days before it. Limit defaults to 20 and is at most 500
type StakingHistoryRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// stakingHistoryParams is the params of a staking history request, its times in Unix nanoseconds
type stakingHistoryParams struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	StartTime      uint64 `json:"start_time,omitempty,string"`
	EndTime        uint64 `json:"end_time,omitempty,string"`
	Limit          uint64 `json:"limit,omitempty,string"`
}

// StakingRequestResponse is a stake or unstake request as the open stake and stake history queries report it; an
// unstake request of a yield-bearing instrument reports its quantity in the underlying currency
type StakingRequestResponse struct {
	InstrumentName     string              `json:"instrument_name"`
	UnderlyingCurrency currency.Code       `json:"underlying_inst_name"`
	CycleID            string              `json:"cycle_id"`
	StakingID          string              `json:"staking_id"`
	Status             string              `json:"status"`
	AccountID          string              `json:"account"`
	Quantity           types.PreciseNumber `json:"quantity"`
	Side               string              `json:"side"`
	CreateTime         types.Time          `json:"create_timestamp_ms"`
}

// StakingRewardResponse is a staking reward paid, and the staked balance it was paid on
type StakingRewardResponse struct {
	StakingInstrumentName string              `json:"staking_inst_name"`
	UnderlyingCurrency    currency.Code       `json:"underlying_inst_name"`
	RewardInstrumentName  string              `json:"reward_inst_name"`
	RewardQuantity        types.PreciseNumber `json:"reward_quantity"`
	StakedBalance         types.PreciseNumber `json:"staked_balance"`
	EventTime             types.Time          `json:"event_timestamp_ms"`
}

// ConvertStakedTokenRequest converts between a staked token and its liquid staking token, ETH.staked to CDCETH or
// CDCETH to ETH.staked, at the rate GetStakingConversionRate returns; Crypto.com rejects the conversion if the rate
// moves from the expected rate by more than the slippage tolerance, which may be zero
type ConvertStakedTokenRequest struct {
	FromInstrumentName           string
	ToInstrumentName             string
	ExpectedRate                 types.Number
	FromQuantity                 types.Number
	SlippageToleranceBasisPoints types.Number
}

// convertStakedTokenParams is the params of a convert request; the slippage tolerance is a string so a zero tolerance is
// sent as "0", where types.Number would send it empty
type convertStakedTokenParams struct {
	FromInstrumentName           string       `json:"from_instrument_name"`
	ToInstrumentName             string       `json:"to_instrument_name"`
	ExpectedRate                 types.Number `json:"expected_rate"`
	FromQuantity                 types.Number `json:"from_quantity"`
	SlippageToleranceBasisPoints string       `json:"slippage_tolerance_bps"`
}

// ConvertStakedTokenResponse is a convert request as created, which Crypto.com processes asynchronously
type ConvertStakedTokenResponse struct {
	FromInstrumentName           string              `json:"from_instrument_name"`
	ToInstrumentName             string              `json:"to_instrument_name"`
	ExpectedRate                 types.Number        `json:"expected_rate"`
	FromQuantity                 types.PreciseNumber `json:"from_quantity"`
	SlippageToleranceBasisPoints types.Number        `json:"slippage_tolerance_bps"`
	ConvertID                    string              `json:"convert_id"`
	Reason                       string              `json:"reason"`
}

// StakingConvertHistoryRequest filters the open convert requests and the convert request history by a time window, as
// StakingHistoryRequest does
type StakingConvertHistoryRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// StakingConvertRequestResponse is a convert request as the open convert and convert history queries report it, with
// the rate it converted at and the quantity it converted to
type StakingConvertRequestResponse struct {
	FromInstrumentName           string              `json:"from_instrument_name"`
	ToInstrumentName             string              `json:"to_instrument_name"`
	ExpectedRate                 types.Number        `json:"expected_rate"`
	FromQuantity                 types.PreciseNumber `json:"from_quantity"`
	SlippageToleranceBasisPoints types.Number        `json:"slippage_tolerance_bps"`
	ActualRate                   types.Number        `json:"actual_rate"`
	ToQuantity                   types.PreciseNumber `json:"to_quantity"`
	ConvertID                    string              `json:"convert_id"`
	Status                       string              `json:"status"`
	CreateTime                   types.Time          `json:"create_timestamp_ms"`
}
