package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SOLStakingProductInfoResponse holds the SOL staking product's limits and yield
type SOLStakingProductInfoResponse struct {
	FastRedemptionDailyLimit types.Number `json:"fastRedemptionDailyLimit"`
	FastRedemptionAvailable  types.Number `json:"fastRedemptionAvail"`
	Rate                     types.Number `json:"rate"`
	RedemptionDays           types.Number `json:"redemptDays"`
	MinimumAmount            types.Number `json:"minAmt"`
}

// SOLStakingBalanceResponse holds the account's OKSOL holdings
type SOLStakingBalanceResponse struct {
	Currency              currency.Code `json:"ccy"`
	Amount                types.Number  `json:"amt"`
	LatestInterestAccrual types.Number  `json:"latestInterestAccrual"`
	TotalInterestAccrual  types.Number  `json:"totalInterestAccrual"`
}

// SOLStakingPurchaseRedeemHistoryResponse holds a SOL staking purchase or redemption
type SOLStakingPurchaseRedeemHistoryResponse struct {
	Type                   string       `json:"type"`
	Amount                 types.Number `json:"amt"`
	RedeemingAmount        types.Number `json:"redeemingAmt"`
	Status                 string       `json:"status"`
	RequestTime            types.Time   `json:"requestTime"`
	CompletedTime          types.Time   `json:"completedTime"`
	EstimatedCompletedTime types.Time   `json:"estCompletedTime"`
}
