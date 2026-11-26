package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Earn products, terms and settings
const (
	EarnTypeStaking   = "STAKING"
	EarnTypeLending   = "LENDING"
	EarnTermFlexible  = "FLEXIBLE"
	EarnTermFixed     = "FIXED"
	EarnSettingOptIn  = "OPT_IN"
	EarnSettingOptOut = "OPT_OUT"
)

// Earn transaction types and statuses
const (
	EarnTransactionTypeStake          = "STAKE"
	EarnTransactionTypeUnstake        = "UNSTAKE"
	EarnTransactionTypeRewardReceived = "REWARD_RECEIVED"
	EarnTransactionTypeOptIn          = "OPT_IN"
	EarnTransactionTypeOptOut         = "OPT_OUT"
	EarnTransactionTypeSubscribe      = "SUBSCRIBE"
	EarnTransactionTypeUnsubscribe    = "UNSUBSCRIBE"
	EarnTransactionTypeAllocation     = "ALLOCATION"
	EarnTransactionTypeDeallocation   = "DEALLOCATION"
	EarnTransactionStatusPending      = "PENDING"
	EarnTransactionStatusCompleted    = "COMPLETED"
	EarnTransactionStatusCancelled    = "CANCELED"
	EarnTransactionStatusFailed       = "FAILED"
)

// EarnSubscriptionRequest holds the parameters for subscribing to or unsubscribing from an Earn product
type EarnSubscriptionRequest struct {
	Currency currency.Code
	EarnType string
	EarnTerm string
	Amount   float64
}

// EarnSubscriptionSettingRequest holds the parameters for opting in to or out of an Earn product
type EarnSubscriptionSettingRequest struct {
	Setting  string
	Currency currency.Code
	EarnType string
}

// EarnTransactionsRequest holds the parameters for fetching Earn transactions
type EarnTransactionsRequest struct {
	Limit         uint64
	Offset        uint64
	Currency      currency.Code
	QuoteCurrency currency.Code
}

// EarnTransactionResponse holds an Earn transaction
type EarnTransactionResponse struct {
	DateTime      types.DateTime `json:"datetime"`
	Type          string         `json:"type"`
	Amount        types.Number   `json:"amount"`
	Currency      currency.Code  `json:"currency"`
	Value         types.Number   `json:"value"`
	QuoteCurrency currency.Code  `json:"quote_currency"`
	Status        string         `json:"status"`
}

// EarnSubscriptionResponse holds an Earn subscription
type EarnSubscriptionResponse struct {
	Currency                  currency.Code `json:"currency"`
	Type                      string        `json:"type"`
	Term                      string        `json:"term"`
	EstimatedAnnualYield      types.Number  `json:"estimated_annual_yield"`
	DistributionPeriod        string        `json:"distribution_period"`
	ActivationPeriod          string        `json:"activation_period"`
	MinimumSubscriptionAmount types.Number  `json:"minimum_subscription_amount"`
	Amount                    types.Number  `json:"amount"`
	AvailableAmount           types.Number  `json:"available_amount"`
	AmountEarned              types.Number  `json:"amount_earned"`
}

// earnSubscriptionBody is the JSON body of an Earn subscribe or unsubscribe request
type earnSubscriptionBody struct {
	Currency string  `json:"currency"`
	EarnType string  `json:"earn_type"`
	EarnTerm string  `json:"earn_term"`
	Amount   float64 `json:"amount"`
}

// earnSubscriptionSettingBody is the JSON body of an Earn subscription setting request
type earnSubscriptionSettingBody struct {
	Setting  string `json:"setting"`
	Currency string `json:"currency"`
	EarnType string `json:"earn_type"`
}
