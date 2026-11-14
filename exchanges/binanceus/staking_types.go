package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// StakingEnvelope holds the code, message and success flag the staking endpoints wrap their data in. A reply with a
// success flag of false is returned as an APIError, so a decoded envelope holds a successful reply
type StakingEnvelope[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Success bool   `json:"success"`
}

// StakingAssetInformationResponse holds the staking terms of an asset
type StakingAssetInformationResponse struct {
	StakingAsset currency.Code `json:"stakingAsset"`
	RewardAsset  currency.Code `json:"rewardAsset"`
	// AnnualPercentageRate is the annual rate of return without rewards restaked
	AnnualPercentageRate types.Number `json:"apr"`
	// AnnualPercentageYield is the annual rate of return with rewards restaked
	AnnualPercentageYield  types.Number `json:"apy"`
	UnstakingPeriodInHours uint64       `json:"unstakingPeriod"`
	MinStakingLimit        types.Number `json:"minStakingLimit"`
	MaxStakingLimit        types.Number `json:"maxStakingLimit"`
	AutoRestake            bool         `json:"autoRestake"`
}

// StakeAssetRequest holds the parameters of a stake
type StakeAssetRequest struct {
	StakingAsset currency.Code
	Amount       float64
	// AutoRestake false pays rewards out instead of restaking them; the endpoint defaults it to true
	AutoRestake *bool
}

// StakeAssetResponse holds the outcome of a stake
type StakeAssetResponse struct {
	Result           string `json:"result"`
	PurchaseRecordID string `json:"purchaseRecordId"`
}

// UnstakeAssetResponse holds the outcome of an unstake
type UnstakeAssetResponse struct {
	Result string `json:"result"`
}

// StakingBalanceResponse holds the staking balances and the statuses of the balances returned
type StakingBalanceResponse struct {
	StakingEnvelope[[]StakingBalance]
	Status []string `json:"status"`
}

// StakingBalance holds an asset's staked amount and its staking terms
type StakingBalance struct {
	Asset                 currency.Code `json:"asset"`
	StakingAmount         types.Number  `json:"stakingAmount"`
	RewardAsset           currency.Code `json:"rewardAsset"`
	AnnualPercentageRate  types.Number  `json:"apr"`
	AnnualPercentageYield types.Number  `json:"apy"`
	AutoRestake           bool          `json:"autoRestake"`
}

// StakingHistoryRequest holds the optional filters of a staking history or staking rewards history query
type StakingHistoryRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Page      uint64 // 1 when zero
	Limit     uint64 // 500 when zero, the most a page holds
}

// StakingRecordResponse holds a staking transaction
type StakingRecordResponse struct {
	Asset         currency.Code `json:"asset"`
	Amount        types.Number  `json:"amount"`
	Type          string        `json:"type"`
	InitiatedTime types.Time    `json:"initiatedTime"`
	Status        string        `json:"status"`
}

// StakingRewardsHistoryResponse holds a page of staking rewards and the number of rewards the query matched
type StakingRewardsHistoryResponse struct {
	StakingEnvelope[[]StakingRewardRecord]
	Total uint64 `json:"total"`
}

// StakingRewardRecord holds a staking reward
type StakingRewardRecord struct {
	Asset         currency.Code `json:"asset"`
	Amount        types.Number  `json:"amount"`
	USDValue      types.Number  `json:"usdValue"`
	Time          types.Time    `json:"time"`
	TransactionID uint64        `json:"tranId"`
	AutoRestaked  bool          `json:"autoRestaked"`
}
