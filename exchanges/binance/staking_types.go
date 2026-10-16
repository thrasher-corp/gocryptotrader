package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ETHStakingAccountResponse holds the ETH staking holdings and 30-day profit of an account
type ETHStakingAccountResponse struct {
	HoldingInETH          types.Number       `json:"holdingInETH"`
	Holdings              ETHStakingHoldings `json:"holdings"`
	ThirtyDaysProfitInETH types.Number       `json:"thirtyDaysProfitInETH"`
	Profit                ETHStakingProfit   `json:"profit"`
}

// ETHStakingHoldings holds the WBETH and BETH balances of an ETH staking account
type ETHStakingHoldings struct {
	WBETHAmount types.Number `json:"wbethAmount"`
	BETHAmount  types.Number `json:"bethAmount"`
}

// ETHStakingProfit holds the 30-day ETH staking profit by source
type ETHStakingProfit struct {
	AmountFromWBETH types.Number `json:"amountFromWBETH"` // accrued within WBETH
	AmountFromBETH  types.Number `json:"amountFromBETH"`  // distributed as BETH to the spot wallet
}

// EarnHistoryRequest holds the time range and paging of a staking or Auto-Invest history query
type EarnHistoryRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// BETHRewardsDistributionHistoryResponse holds a page of BETH rewards distributions
type BETHRewardsDistributionHistoryResponse struct {
	Rows  []BETHRewardsDistribution `json:"rows"`
	Total uint64                    `json:"total"`
}

// BETHRewardsDistribution holds a BETH rewards distribution
type BETHRewardsDistribution struct {
	Time                 types.Time    `json:"time"`
	Asset                currency.Code `json:"asset"`
	Holding              types.Number  `json:"holding"`
	Amount               types.Number  `json:"amount"`
	AnnualPercentageRate types.Number  `json:"annualPercentageRate"` // 0.5 is 50%
	Status               string        `json:"status"`
}

// ETHStakingQuotaResponse holds the ETH staking and redemption quotas and limits
type ETHStakingQuotaResponse struct {
	LeftStakingPersonalQuota    types.Number `json:"leftStakingPersonalQuota"`
	LeftRedemptionPersonalQuota types.Number `json:"leftRedemptionPersonalQuota"`
	MinimumStakeAmount          types.Number `json:"minStakeAmount"`
	MinimumRedeemAmount         types.Number `json:"minRedeemAmount"`
	RedeemPeriod                uint64       `json:"redeemPeriod"` // days
	Stakeable                   bool         `json:"stakeable"`
	Redeemable                  bool         `json:"redeemable"`
	CommissionFee               types.Number `json:"commissionFee"`
	Calculating                 bool         `json:"calculating"`
}

// StakingRedemptionHistoryRequest holds the parameters of GetETHRedemptionHistory and GetSOLRedemptionHistory
type StakingRedemptionHistoryRequest struct {
	RedeemID  uint64
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// ETHRedemptionHistoryResponse holds a page of ETH staking redemptions
type ETHRedemptionHistoryResponse struct {
	Rows  []ETHRedemption `json:"rows"`
	Total uint64          `json:"total"`
}

// ETHRedemption holds a WBETH or BETH redemption for ETH
type ETHRedemption struct {
	Time             types.Time    `json:"time"`
	ArrivalTime      types.Time    `json:"arrivalTime"`
	Asset            currency.Code `json:"asset"`
	Amount           types.Number  `json:"amount"`
	DistributeAsset  currency.Code `json:"distributeAsset"`
	DistributeAmount types.Number  `json:"distributeAmount"`
	ConversionRatio  types.Number  `json:"conversionRatio"`
	Status           string        `json:"status"`
}

// StakingSubscriptionHistoryRequest holds the parameters of GetETHStakingHistory and GetSOLStakingHistory
type StakingSubscriptionHistoryRequest struct {
	PurchaseID uint64
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64
}

// ETHStakingHistoryResponse holds a page of ETH staking subscriptions
type ETHStakingHistoryResponse struct {
	Rows  []ETHStakingSubscription `json:"rows"`
	Total uint64                   `json:"total"`
}

// ETHStakingSubscription holds an ETH staking subscription
type ETHStakingSubscription struct {
	Time             types.Time    `json:"time"`
	Asset            currency.Code `json:"asset"`
	Amount           types.Number  `json:"amount"`
	DistributeAsset  currency.Code `json:"distributeAsset"`
	DistributeAmount types.Number  `json:"distributeAmount"`
	ConversionRatio  types.Number  `json:"conversionRatio"`
	Status           string        `json:"status"`
}

// WBETHRateHistoryResponse holds a page of WBETH rates
type WBETHRateHistoryResponse struct {
	Rows  []WBETHRate  `json:"rows"`
	Total types.Number `json:"total"`
}

// WBETHRate holds the WBETH annual percentage rate and ETH exchange rate at a point in time
type WBETHRate struct {
	AnnualPercentageRate types.Number `json:"annualPercentageRate"`
	ExchangeRate         types.Number `json:"exchangeRate"`
	Time                 types.Time   `json:"time"`
}

// WBETHRewardHistoryResponse holds a page of WBETH rewards and the estimated total
type WBETHRewardHistoryResponse struct {
	EstimatedRewardsInETH types.Number  `json:"estRewardsInETH"`
	Rows                  []WBETHReward `json:"rows"`
	Total                 uint64        `json:"total"`
}

// WBETHReward holds the rewards accrued within held WBETH on a day
type WBETHReward struct {
	Time                 types.Time   `json:"time"`
	AmountInETH          types.Number `json:"amountInETH"`
	Holding              types.Number `json:"holding"`
	HoldingInETH         types.Number `json:"holdingInETH"`
	AnnualPercentageRate types.Number `json:"annualPercentageRate"`
}

// WBETHWrapHistoryResponse holds a page of BETH to WBETH wraps or WBETH to BETH unwraps
type WBETHWrapHistoryResponse struct {
	Rows  []WBETHWrap `json:"rows"`
	Total uint64      `json:"total"`
}

// WBETHWrap holds a BETH to WBETH wrap or WBETH to BETH unwrap
type WBETHWrap struct {
	Time         types.Time    `json:"time"`
	FromAsset    currency.Code `json:"fromAsset"`
	FromAmount   types.Number  `json:"fromAmount"`
	ToAsset      currency.Code `json:"toAsset"`
	ToAmount     types.Number  `json:"toAmount"`
	ExchangeRate types.Number  `json:"exchangeRate"` // BETH per WBETH
	Status       string        `json:"status"`
}

// ETHRedemptionResponse holds the outcome of an ETH redemption
type ETHRedemptionResponse struct {
	Success         bool         `json:"success"`
	ETHAmount       types.Number `json:"ethAmount"`
	RedeemID        uint64       `json:"redeemId"`
	ConversionRatio types.Number `json:"conversionRatio"`
	ArrivalTime     types.Time   `json:"arrivalTime"`
}

// ETHStakingSubscriptionResponse holds the outcome of an ETH staking subscription
type ETHStakingSubscriptionResponse struct {
	Success         bool         `json:"success"`
	WBETHAmount     types.Number `json:"wbethAmount"`
	PurchaseID      uint64       `json:"purchaseId"`
	ConversionRatio types.Number `json:"conversionRatio"` // ETH per WBETH
}

// WrapBETHResponse holds the outcome of a BETH wrap
type WrapBETHResponse struct {
	Success      bool         `json:"success"`
	WBETHAmount  types.Number `json:"wbethAmount"`
	ExchangeRate types.Number `json:"exchangeRate"`
}

// EarnSuccessResponse holds the outcome of an Earn request that reports only whether it succeeded
type EarnSuccessResponse struct {
	Success bool `json:"success"`
}

// BNSOLRateHistoryResponse holds a page of BNSOL rates
type BNSOLRateHistoryResponse struct {
	Rows  []BNSOLRate  `json:"rows"`
	Total types.Number `json:"total"`
}

// BNSOLRate holds the BNSOL annual percentage rate, SOL exchange rate and boost rewards at a point in time
type BNSOLRate struct {
	AnnualPercentageRate types.Number       `json:"annualPercentageRate"`
	ExchangeRate         types.Number       `json:"exchangeRate"`
	BoostRewards         []BNSOLBoostReward `json:"boostRewards"`
	Time                 types.Time         `json:"time"`
}

// BNSOLBoostReward holds a boost APR paid on BNSOL and its reward asset
type BNSOLBoostReward struct {
	BoostAPR     types.Number  `json:"boostAPR"`
	RewardsAsset currency.Code `json:"rewardsAsset"`
}

// BNSOLRewardsHistoryResponse holds a page of BNSOL rewards and the estimated total
type BNSOLRewardsHistoryResponse struct {
	EstimatedRewardsInSOL types.Number  `json:"estRewardsInSOL"`
	Rows                  []BNSOLReward `json:"rows"`
	Total                 uint64        `json:"total"`
}

// BNSOLReward holds the rewards accrued within held BNSOL on a day
type BNSOLReward struct {
	Time                 types.Time   `json:"time"`
	AmountInSOL          types.Number `json:"amountInSOL"`
	Holding              types.Number `json:"holding"`
	HoldingInSOL         types.Number `json:"holdingInSOL"`
	AnnualPercentageRate types.Number `json:"annualPercentageRate"`
}

// BoostRewardsHistoryRequest holds the parameters of GetBoostRewardsHistory
type BoostRewardsHistoryRequest struct {
	Type      string // CLAIM or DISTRIBUTE
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// BoostRewardsHistoryResponse holds a page of SOL staking boost rewards
type BoostRewardsHistoryResponse struct {
	Rows  []BoostReward `json:"rows"`
	Total uint64        `json:"total"`
}

// BoostReward holds a claimed or distributed SOL staking boost reward
type BoostReward struct {
	Time         types.Time    `json:"time"`
	Token        currency.Code `json:"token"`
	Amount       types.Number  `json:"amount"`
	BNSOLHolding types.Number  `json:"bnsolHolding"`
	Status       string        `json:"status"`
}

// SOLRedemptionHistoryResponse holds a page of SOL staking redemptions
type SOLRedemptionHistoryResponse struct {
	Rows  []SOLRedemption `json:"rows"`
	Total uint64          `json:"total"`
}

// SOLRedemption holds a BNSOL redemption for SOL
type SOLRedemption struct {
	Time             types.Time    `json:"time"`
	ArrivalTime      types.Time    `json:"arrivalTime"`
	Asset            currency.Code `json:"asset"`
	Amount           types.Number  `json:"amount"`
	DistributeAsset  currency.Code `json:"distributeAsset"`
	DistributeAmount types.Number  `json:"distributeAmount"`
	ExchangeRate     types.Number  `json:"exchangeRate"`
	Status           string        `json:"status"`
}

// SOLStakingHistoryResponse holds a page of SOL staking subscriptions
type SOLStakingHistoryResponse struct {
	Rows  []SOLStakingSubscription `json:"rows"`
	Total uint64                   `json:"total"`
}

// SOLStakingSubscription holds a SOL staking subscription
type SOLStakingSubscription struct {
	Time             types.Time    `json:"time"`
	Asset            currency.Code `json:"asset"`
	Amount           types.Number  `json:"amount"`
	DistributeAsset  currency.Code `json:"distributeAsset"`
	DistributeAmount types.Number  `json:"distributeAmount"`
	ExchangeRate     types.Number  `json:"exchangeRate"`
	Status           string        `json:"status"`
}

// SOLStakingQuotaResponse holds the SOL staking and redemption quotas and limits
type SOLStakingQuotaResponse struct {
	LeftStakingPersonalQuota    types.Number `json:"leftStakingPersonalQuota"`
	LeftRedemptionPersonalQuota types.Number `json:"leftRedemptionPersonalQuota"`
	MinimumStakeAmount          types.Number `json:"minStakeAmount"`
	MinimumRedeemAmount         types.Number `json:"minRedeemAmount"`
	RedeemPeriod                uint64       `json:"redeemPeriod"` // days
	Stakeable                   bool         `json:"stakeable"`
	Redeemable                  bool         `json:"redeemable"`
	SoldOut                     bool         `json:"soldOut"`
	CommissionFee               types.Number `json:"commissionFee"`
	NextEpochTime               types.Time   `json:"nextEpochTime"`
	Calculating                 bool         `json:"calculating"`
}

// SOLStakingUnclaimedReward holds an unclaimed SOL staking boost reward
type SOLStakingUnclaimedReward struct {
	Amount       types.Number  `json:"amount"`
	RewardsAsset currency.Code `json:"rewardsAsset"`
}

// SOLRedemptionResponse holds the outcome of a SOL redemption
type SOLRedemptionResponse struct {
	Success      bool         `json:"success"`
	SOLAmount    types.Number `json:"solAmount"`
	RedeemID     uint64       `json:"redeemId"`
	ExchangeRate types.Number `json:"exchangeRate"`
	ArrivalTime  types.Time   `json:"arrivalTime"`
}

// SOLStakingAccountResponse holds the BNSOL holdings and 30-day profit of an account
type SOLStakingAccountResponse struct {
	BNSOLAmount           types.Number `json:"bnsolAmount"`
	HoldingInSOL          types.Number `json:"holdingInSOL"`
	ThirtyDaysProfitInSOL types.Number `json:"thirtyDaysProfitInSOL"`
}

// SOLStakingSubscriptionResponse holds the outcome of a SOL staking subscription
type SOLStakingSubscriptionResponse struct {
	Success      bool         `json:"success"`
	BNSOLAmount  types.Number `json:"bnsolAmount"`
	PurchaseID   uint64       `json:"purchaseId"`
	ExchangeRate types.Number `json:"exchangeRate"`
}
