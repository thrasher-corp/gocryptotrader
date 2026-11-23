package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ETHStakingProductInfoResponse holds the ETH staking product's limits and yield
type ETHStakingProductInfoResponse struct {
	FastRedemptionDailyLimit types.Number `json:"fastRedemptionDailyLimit"`
	Rate                     types.Number `json:"rate"`
	RedemptionDays           types.Number `json:"redemptDays"`
	MinimumAmount            types.Number `json:"minAmt"`
}

// StakingAmountRequest holds the amount of an ETH or SOL staking purchase or redemption
type StakingAmountRequest struct {
	Amount float64 `json:"amt,string"`
}

// ETHStakingCancelRedemptionRequest holds the ETH staking redemption to cancel
type ETHStakingCancelRedemptionRequest struct {
	OrderID string `json:"ordId"`
}

// ETHStakingOrderIDResponse holds the ETH staking redemption a cancellation acted on
type ETHStakingOrderIDResponse struct {
	OrderID string `json:"ordId"`
}

// ETHStakingBalanceResponse holds the account's BETH holdings
type ETHStakingBalanceResponse struct {
	Currency              currency.Code `json:"ccy"`
	Amount                types.Number  `json:"amt"`
	LatestInterestAccrual types.Number  `json:"latestInterestAccrual"`
	TotalInterestAccrual  types.Number  `json:"totalInterestAccrual"`
	Timestamp             types.Time    `json:"ts"`
}

// StakingPurchaseRedeemHistoryRequest holds the filters and request time cursors for ETH and SOL staking purchase and
// redemption history
type StakingPurchaseRedeemHistoryRequest struct {
	Type   string
	Status string
	// After returns records requested earlier than this time
	After time.Time
	// Before returns records requested later than this time
	Before time.Time
	Limit  uint64
}

// ETHStakingPurchaseRedeemHistoryResponse holds an ETH staking purchase or redemption
type ETHStakingPurchaseRedeemHistoryResponse struct {
	Type                   string       `json:"type"`
	Amount                 types.Number `json:"amt"`
	RedeemingAmount        types.Number `json:"redeemingAmt"`
	Status                 string       `json:"status"`
	OrderID                string       `json:"ordId"`
	RequestTime            types.Time   `json:"requestTime"`
	CompletedTime          types.Time   `json:"completedTime"`
	EstimatedCompletedTime types.Time   `json:"estCompletedTime"`
}

// APYHistoryResponse holds a daily annual percentage yield of an ETH staking, SOL staking or Stable Rewards product
type APYHistoryResponse struct {
	Rate      types.Number `json:"rate"`
	Timestamp types.Time   `json:"ts"`
}
