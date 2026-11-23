package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// StableRewardsProductInfoResponse holds the settlement currencies a stablecoin subscribes and redeems through
type StableRewardsProductInfoResponse struct {
	Details   []StableRewardsProductDetail `json:"details"`
	Timestamp types.Time                   `json:"ts"`
}

// StableRewardsProductDetail holds a stablecoin's subscription and redemption terms for one settlement currency
type StableRewardsProductDetail struct {
	Currency                   currency.Code `json:"ccy"`
	SettlementCurrency         currency.Code `json:"settleCcy"`
	SubscriptionFeeRate        types.Number  `json:"subFeeRate"`
	RedemptionFeeRate          types.Number  `json:"redemptFeeRate"`
	MinimumSubscriptionAmount  types.Number  `json:"minSubAmt"`
	MinimumRedemptionAmount    types.Number  `json:"minRedeemAmt"`
	RemainingSubscriptionQuota types.Number  `json:"remainingSubQuota"`     // -1 when unlimited
	RemainingRedemptionQuota   types.Number  `json:"remainingRedemptQuota"` // -1 when unlimited
	CanRedeem                  bool          `json:"canRedeem"`
}

// StableRewardsBalanceResponse holds the account's Stable Rewards balances
type StableRewardsBalanceResponse struct {
	Details   []StableRewardsBalanceDetail `json:"details"`
	Timestamp types.Time                   `json:"ts"`
}

// StableRewardsBalanceDetail holds the account's balance and earnings in one stablecoin
type StableRewardsBalanceDetail struct {
	Currency         currency.Code `json:"ccy"`
	Amount           types.Number  `json:"amt"`
	TotalEarnAccrual types.Number  `json:"totalEarnAccrual"`
	State            string        `json:"state"`
}

// StableRewardsAPYHistoryRequest holds the stablecoin and number of days of APY history to return
type StableRewardsAPYHistoryRequest struct {
	Currency currency.Code
	Days     uint64
}
