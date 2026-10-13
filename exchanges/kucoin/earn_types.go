package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EarnPurchaseRequest holds the parameters of SubscribeToEarnFixedIncomeProduct
type EarnPurchaseRequest struct {
	ProductID string
	Amount    float64
	// AccountType is the account paying for the subscription: MAIN (funding) or TRADE (spot trading)
	AccountType string
}

// EarnPurchaseResponse holds the result of an Earn product subscription
type EarnPurchaseResponse struct {
	// OrderID is the holding ID
	OrderID string `json:"orderId"`
	// OrderTransactionID is the subscription order ID
	OrderTransactionID string `json:"orderTxId"`
}

// EarnRedeemPreviewResponse holds what redeeming an Earn holding would pay out
type EarnRedeemPreviewResponse struct {
	Currency     currency.Code `json:"currency"`
	RedeemAmount types.Number  `json:"redeemAmount"`
	// PenaltyInterestAmount is charged for redeeming a fixed-term holding early
	PenaltyInterestAmount types.Number `json:"penaltyInterestAmount"`
	// RedeemPeriod is the redemption waiting period in days
	RedeemPeriod     uint64     `json:"redeemPeriod"`
	DeliverTime      types.Time `json:"deliverTime"`
	ManualRedeemable bool       `json:"manualRedeemable"`
	// RedeemAll reports whether the whole holding must be redeemed, as an early fixed-term redemption requires
	RedeemAll bool `json:"redeemAll"`
}

// EarnRedeemRequest holds the parameters of RedeemByEarnHoldingID
type EarnRedeemRequest struct {
	// OrderID is the holding ID
	OrderID string
	Amount  float64
	// FromAccountType is the account the ETH staking holding (ETH2) is redeemed from: MAIN (funding) or TRADE (spot
	// trading); other holdings ignore it
	FromAccountType string
	// ConfirmPunishRedeem accepts the penalty for redeeming a fixed-term holding early, which redeems all of it
	ConfirmPunishRedeem bool
}

// EarnRedeemResponse holds the result of an Earn holding redemption
type EarnRedeemResponse struct {
	// OrderTransactionID is the redemption order ID
	OrderTransactionID string     `json:"orderTxId"`
	DeliverTime        types.Time `json:"deliverTime"`
	// Status is SUCCESS or PENDING
	Status string       `json:"status"`
	Amount types.Number `json:"amount"`
}

// EarnProduct is an Earn product, as listed by the savings, promotion, staking, KCS staking and ETH staking product
// endpoints
type EarnProduct struct {
	ID                string        `json:"id"`
	Currency          currency.Code `json:"currency"`
	Category          string        `json:"category"`
	Type              string        `json:"type"`
	Precision         uint64        `json:"precision"`
	ProductUpperLimit types.Number  `json:"productUpperLimit"`
	UserUpperLimit    types.Number  `json:"userUpperLimit"`
	UserLowerLimit    types.Number  `json:"userLowerLimit"`
	// RedeemPeriod is the redemption waiting period in days
	RedeemPeriod   uint64     `json:"redeemPeriod"`
	LockStartTime  types.Time `json:"lockStartTime"`
	LockEndTime    types.Time `json:"lockEndTime"`
	ApplyStartTime types.Time `json:"applyStartTime"`
	ApplyEndTime   types.Time `json:"applyEndTime"`
	// ReturnRate is the annualised rate of return, where 0.035 is 3.5%
	ReturnRate           types.Number  `json:"returnRate"`
	IncomeCurrency       currency.Code `json:"incomeCurrency"`
	EarlyRedeemSupported types.Boolean `json:"earlyRedeemSupported"`
	ProductRemainAmount  types.Number  `json:"productRemainAmount"`
	Status               string        `json:"status"`
	RedeemType           string        `json:"redeemType"`
	IncomeReleaseType    string        `json:"incomeReleaseType"`
	// InterestDate is the most recent interest date
	InterestDate types.Time `json:"interestDate"`
	// Duration is the product duration in days
	Duration    uint64        `json:"duration"`
	NewUserOnly types.Boolean `json:"newUserOnly"`
}

// EarnAccountHoldingRequest holds the filters of GetEarnFixedIncomeCurrentHoldings
type EarnAccountHoldingRequest struct {
	Currency  currency.Code
	ProductID string
	// ProductCategory is DEMAND, ACTIVITY, STAKING, KCS_STAKING or ETH2
	ProductCategory string
	CurrentPage     uint64
	PageSize        uint64
}

// EarnAccountHoldingResponse is a page of the account's Earn holdings
type EarnAccountHoldingResponse struct {
	TotalNumber uint64               `json:"totalNum"`
	Items       []EarnAccountHolding `json:"items"`
	CurrentPage uint64               `json:"currentPage"`
	PageSize    uint64               `json:"pageSize"`
	TotalPage   uint64               `json:"totalPage"`
}

// EarnAccountHolding is an Earn holding of the account
type EarnAccountHolding struct {
	// OrderID is the holding ID
	OrderID         string        `json:"orderId"`
	ProductID       string        `json:"productId"`
	ProductCategory string        `json:"productCategory"`
	ProductType     string        `json:"productType"`
	Currency        currency.Code `json:"currency"`
	IncomeCurrency  currency.Code `json:"incomeCurrency"`
	// ReturnRate is the annualised rate of return, where 0.035 is 3.5%
	ReturnRate      types.Number `json:"returnRate"`
	HoldAmount      types.Number `json:"holdAmount"`
	RedeemedAmount  types.Number `json:"redeemedAmount"`
	RedeemingAmount types.Number `json:"redeemingAmount"`
	LockStartTime   types.Time   `json:"lockStartTime"`
	LockEndTime     types.Time   `json:"lockEndTime"`
	// PurchaseTime is the most recent subscription time
	PurchaseTime types.Time `json:"purchaseTime"`
	// RedeemPeriod is the redemption waiting period in days
	RedeemPeriod uint64 `json:"redeemPeriod"`
	// Status is LOCKED (held) or REDEEMING
	Status               string        `json:"status"`
	EarlyRedeemSupported types.Boolean `json:"earlyRedeemSupported"`
}

// StructuredProductPurchaseRequest holds the parameters of PurchaseStructuredProduct
type StructuredProductPurchaseRequest struct {
	ProductID      string
	InvestCurrency currency.Code
	InvestAmount   float64
	// AccountType is the account paying for the subscription: MAIN (funding) or TRADE (spot trading)
	AccountType string
}

// StructuredProductPurchaseResponse holds the result of a structured product subscription
type StructuredProductPurchaseResponse struct {
	// OrderID is the holding ID
	OrderID string `json:"orderId"`
}

// DualInvestmentProductsRequest holds the filters of GetDualInvestmentProducts
type DualInvestmentProductsRequest struct {
	// Category is DUAL_CLASSIC, DUAL_BOOSTER or DUAL_EXTRA
	Category       string
	StrikeCurrency currency.Code
	InvestCurrency currency.Code
	// Side is CALL or PUT
	Side string
}

// DualInvestmentProduct is a dual investment product
type DualInvestmentProduct struct {
	Category  string `json:"category"`
	ProductID string `json:"productId"`
	// SeriesNames, ProductName and UserAvailableBalance are in the docs' response example but not in its schema
	SeriesNames    string        `json:"seriesNames"`
	ProductName    string        `json:"productName"`
	TargetCurrency currency.Code `json:"targetCurrency"`
	QuoteCurrency  currency.Code `json:"quoteCurrency"`
	InvestCurrency currency.Code `json:"investCurrency"`
	// StrikeCurrency is the currency settled when the strike price is met
	StrikeCurrency currency.Code `json:"strikeCurrency"`
	StrikePrice    types.Number  `json:"strikePrice"`
	ProtectPrice   types.Number  `json:"protectPrice"`
	// AnnualRate is the annualised rate of return, where 0.05 is 5%
	AnnualRate       types.Number `json:"annualRate"`
	ExpirationTime   types.Time   `json:"expirationTime"`
	Side             string       `json:"side"`
	ExpectSettleTime types.Time   `json:"expectSettleTime"`
	// Duration is the product duration in days
	Duration             types.Number `json:"duration"`
	LowerLimit           types.Number `json:"lowerLimit"`
	UpperLimit           types.Number `json:"upperLimit"`
	AvailableScale       types.Number `json:"availableScale"`
	UserAvailableBalance types.Number `json:"userAvailableBalance"`
	// SoldStatus is AVAILABLE or SOLD_OUT
	SoldStatus string `json:"soldStatus"`
	// Increment is the step an investment amount must be a multiple of
	Increment types.Number `json:"increment"`
}

// StructuredProductOrdersRequest holds the filters of GetStructuredProductOrders
type StructuredProductOrdersRequest struct {
	// Categories are any of DUAL_CLASSIC, DUAL_BOOSTER and DUAL_EXTRA
	Categories     []string
	OrderID        string
	InvestCurrency currency.Code
	CurrentPage    uint64
	PageSize       uint64
}

// StructuredProductOrdersResponse is a page of the account's structured product orders
type StructuredProductOrdersResponse struct {
	CurrentPage uint64                   `json:"currentPage"`
	PageSize    uint64                   `json:"pageSize"`
	TotalNumber uint64                   `json:"totalNum"`
	TotalPage   uint64                   `json:"totalPage"`
	Items       []StructuredProductOrder `json:"items"`
}

// StructuredProductOrder is a structured product order of the account
type StructuredProductOrder struct {
	// OrderID is the holding's order ID
	OrderID  string `json:"orderId"`
	Category string `json:"category"`
	Side     string `json:"side"`
	// Duration is the product duration in days
	Duration types.Number `json:"duration"`
	// APR is the annual percentage rate
	APR            types.Number  `json:"apr"`
	InvestCurrency currency.Code `json:"investCurrency"`
	// StrikeCurrency is the currency settled when the strike price is met
	StrikeCurrency currency.Code `json:"strikeCurrency"`
	InvestAmount   types.Number  `json:"investAmount"`
	SettleAmount   types.Number  `json:"settleAmount"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	SettlePrice    types.Number  `json:"settlePrice"`
	// TargetPrice is the strike price that decides the settlement
	TargetPrice    types.Number `json:"targetPrice"`
	ExpirationTime types.Time   `json:"expirationTime"`
	// Status is PENDING, ONGOING, UNSOLD, SETTLING or FINISHED
	Status string `json:"status"`
}
