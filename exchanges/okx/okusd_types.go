package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OKUSDLimitsResponse holds the master account's remaining daily OKUSD subscription and redemption quotas
type OKUSDLimitsResponse struct {
	SubscriptionLimit       OKUSDSubscriptionLimit `json:"subLimit"`
	FastRedemptionLimit     OKUSDRedemptionLimit   `json:"fastRedeemLimit"`
	StandardRedemptionLimit OKUSDRedemptionLimit   `json:"stdRedeemLimit"`
	Timestamp               types.Time             `json:"ts"`
}

// OKUSDSubscriptionLimit holds the daily OKUSD subscription quota, in USDT
type OKUSDSubscriptionLimit struct {
	MaximumSubscriptionAmount types.Number `json:"maxSubAmt"`
	PersonalDailyLimit        types.Number `json:"personalDailyLimit"`
	PersonalUsedAmount        types.Number `json:"personalUsedAmt"`
	PlatformDailyLimit        types.Number `json:"platformDailyLimit"`
	PlatformUsedAmount        types.Number `json:"platformUsedAmt"`
}

// OKUSDRedemptionLimit holds the daily quota and fee rate of a kind of OKUSD redemption, in OKUSD
type OKUSDRedemptionLimit struct {
	PersonalDailyLimit types.Number `json:"personalDailyLimit"`
	PersonalUsedAmount types.Number `json:"personalUsedAmt"`
	PlatformDailyLimit types.Number `json:"platformDailyLimit"`
	PlatformUsedAmount types.Number `json:"platformUsedAmt"`
	FeeRate            types.Number `json:"feeRate"`
}

// OKUSDSubscribeRequest holds an OKUSD subscription
type OKUSDSubscribeRequest struct {
	Amount        float64 `json:"amt,string"`
	ClientOrderID string  `json:"clOrdId"`
}

// OKUSDSubscribeResponse holds an OKUSD subscription order
type OKUSDSubscribeResponse struct {
	OrderID       string        `json:"ordId"`
	ClientOrderID string        `json:"clOrdId"`
	Currency      currency.Code `json:"ccy"`
	Amount        types.Number  `json:"amt"`
	OKUSDAmount   types.Number  `json:"okusdAmt"`
	State         string        `json:"state"`
	Timestamp     types.Time    `json:"ts"`
}

// OKUSDRedeemTypeFast redeems OKUSD for USDT with real-time settlement
const OKUSDRedeemTypeFast = "1"

// OKUSDRedeemTypeStandard redeems OKUSD for USDT with standard settlement, five or six calendar days later
const OKUSDRedeemTypeStandard = "2"

// OKUSDRedeemRequest holds an OKUSD redemption
type OKUSDRedeemRequest struct {
	Amount        float64 `json:"amt,string"`
	RedeemType    string  `json:"redeemType"`
	ClientOrderID string  `json:"clOrdId"`
}

// OKUSDRedeemResponse holds an OKUSD redemption order
type OKUSDRedeemResponse struct {
	OrderID                 string        `json:"ordId"`
	ClientOrderID           string        `json:"clOrdId"`
	Currency                currency.Code `json:"ccy"`
	Amount                  types.Number  `json:"amt"`
	Fee                     types.Number  `json:"fee"`
	USDTAmount              types.Number  `json:"usdtAmt"`
	RedeemType              string        `json:"redeemType"`
	State                   string        `json:"state"`
	EstimatedSettlementTime types.Time    `json:"estSettlementTime"`
	Timestamp               types.Time    `json:"ts"`
}

// OKUSDAccountResponse holds the master account's OKUSD balance and accrued yield
type OKUSDAccountResponse struct {
	Currency         currency.Code `json:"ccy"`
	Amount           types.Number  `json:"amt"`
	TotalEarnAccrual types.Number  `json:"totalEarnAccrual"`
	Timestamp        types.Time    `json:"ts"`
}

// OKUSDHistoryRequest holds the page size and inclusive time window of an OKUSD history request
type OKUSDHistoryRequest struct {
	Limit uint64
	Begin time.Time
	End   time.Time
}

// OKUSDSubscriptionHistoryResponse holds a completed OKUSD subscription
type OKUSDSubscriptionHistoryResponse struct {
	OrderID                  string        `json:"ordId"`
	ClientOrderID            string        `json:"clOrdId"`
	Currency                 currency.Code `json:"ccy"`
	Amount                   types.Number  `json:"amt"`
	SettlementCurrency       currency.Code `json:"settleCcy"`
	SettlementCurrencyAmount types.Number  `json:"settleCcyAmt"`
	Status                   string        `json:"status"`
	Timestamp                types.Time    `json:"ts"`
}

// OKUSDRedemptionHistoryRequest holds the page size, inclusive time window and redemption type of an OKUSD
// redemption history request
type OKUSDRedemptionHistoryRequest struct {
	Limit uint64
	Begin time.Time
	End   time.Time
	Type  string
}

// OKUSDRedemptionHistoryResponse holds an OKUSD redemption
type OKUSDRedemptionHistoryResponse struct {
	OrderID                  string        `json:"ordId"`
	ClientOrderID            string        `json:"clOrdId"`
	Currency                 currency.Code `json:"ccy"`
	Amount                   types.Number  `json:"amt"`
	Fee                      types.Number  `json:"fee"`
	SettlementCurrency       currency.Code `json:"settleCcy"`
	SettlementCurrencyAmount types.Number  `json:"settleCcyAmt"`
	Type                     string        `json:"type"`
	Status                   string        `json:"status"`
	EstimatedSettlementTime  types.Time    `json:"estSettlementTime"`
	Timestamp                types.Time    `json:"ts"`
}

// OKUSDRewardsHistoryResponse holds a daily OKUSD yield distribution
type OKUSDRewardsHistoryResponse struct {
	Currency     currency.Code `json:"ccy"`
	EarnedAmount types.Number  `json:"earnAmt"`
	Amount       types.Number  `json:"amt"`
	APR          types.Number  `json:"apr"`
	Timestamp    types.Time    `json:"ts"`
}

// OKUSDAPRHistoryResponse holds an OKUSD APR snapshot
type OKUSDAPRHistoryResponse struct {
	APR       types.Number `json:"apr"`
	Timestamp types.Time   `json:"ts"`
}
