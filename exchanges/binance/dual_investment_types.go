package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DualInvestmentProductListRequest holds the parameters of GetDualInvestmentProductList
type DualInvestmentProductListRequest struct {
	OptionType    string // CALL or PUT
	ExercisedCoin currency.Code
	InvestCoin    currency.Code
	PageSize      uint64
	PageIndex     uint64
}

// DualInvestmentProductListResponse holds a page of Dual Investment products
type DualInvestmentProductListResponse struct {
	Total uint64                  `json:"total"`
	List  []DualInvestmentProduct `json:"list"`
}

// DualInvestmentProduct holds a Dual Investment product
type DualInvestmentProduct struct {
	ID                   string        `json:"id"`
	InvestCoin           currency.Code `json:"investCoin"`
	ExercisedCoin        currency.Code `json:"exercisedCoin"`
	StrikePrice          types.Number  `json:"strikePrice"`
	Duration             uint64        `json:"duration"` // days
	SettleDate           types.Time    `json:"settleDate"`
	PurchaseDecimal      uint64        `json:"purchaseDecimal"`
	PurchaseEndTime      types.Time    `json:"purchaseEndTime"`
	CanPurchase          bool          `json:"canPurchase"`
	APR                  types.Number  `json:"apr"`
	OrderID              uint64        `json:"orderId"`
	MinimumAmount        types.Number  `json:"minAmount"`
	MaximumAmount        types.Number  `json:"maxAmount"`
	CreateTimestamp      types.Time    `json:"createTimestamp"`
	OptionType           string        `json:"optionType"`
	IsAutoCompoundEnable bool          `json:"isAutoCompoundEnable"`
	AutoCompoundPlanList []string      `json:"autoCompoundPlanList"`
}

// DualInvestmentAutoCompoundStatusResponse holds a position's auto-compound plan after a change
type DualInvestmentAutoCompoundStatusResponse struct {
	PositionID       string `json:"positionId"`
	AutoCompoundPlan string `json:"autoCompoundPlan"`
}

// DualInvestmentAccountsResponse holds the total value held in Dual Investment
type DualInvestmentAccountsResponse struct {
	TotalAmountInBTC  types.Number `json:"totalAmountInBTC"`
	TotalAmountInUSDT types.Number `json:"totalAmountInUSDT"`
}

// DualInvestmentPositionsResponse holds a page of Dual Investment positions
type DualInvestmentPositionsResponse struct {
	Total uint64                   `json:"total"`
	List  []DualInvestmentPosition `json:"list"`
}

// DualInvestmentPosition holds a Dual Investment position
type DualInvestmentPosition struct {
	ID                 string        `json:"id"` // position ID
	InvestCoin         currency.Code `json:"investCoin"`
	ExercisedCoin      currency.Code `json:"exercisedCoin"`
	SubscriptionAmount types.Number  `json:"subscriptionAmount"`
	StrikePrice        types.Number  `json:"strikePrice"`
	Duration           uint64        `json:"duration"` // days
	SettleDate         types.Time    `json:"settleDate"`
	PurchaseStatus     string        `json:"purchaseStatus"`
	APR                types.Number  `json:"apr"`
	OrderID            uint64        `json:"orderId"`
	PurchaseEndTime    types.Time    `json:"purchaseEndTime"`
	OptionType         string        `json:"optionType"`
	AutoCompoundPlan   string        `json:"autoCompoundPlan"`
	SubscriptionTime   types.Time    `json:"subscriptionTime"`
}

// SubscribeDualInvestmentProductsRequest holds the parameters of SubscribeDualInvestmentProducts
type SubscribeDualInvestmentProductsRequest struct {
	ID               string // product ID from GetDualInvestmentProductList
	OrderID          uint64 // order ID from GetDualInvestmentProductList
	DepositAmount    float64
	AutoCompoundPlan string // NONE, STANDARD or ADVANCED
}

// DualInvestmentSubscriptionResponse holds the position a Dual Investment subscription opened
type DualInvestmentSubscriptionResponse struct {
	PositionID         uint64        `json:"positionId"`
	InvestCoin         currency.Code `json:"investCoin"`
	ExercisedCoin      currency.Code `json:"exercisedCoin"`
	SubscriptionAmount types.Number  `json:"subscriptionAmount"`
	Duration           uint64        `json:"duration"`         // days
	AutoCompoundPlan   string        `json:"autoCompoundPlan"` // absent when the plan is NONE
	StrikePrice        types.Number  `json:"strikePrice"`
	SettleDate         types.Time    `json:"settleDate"`
	PurchaseStatus     string        `json:"purchaseStatus"`
	APR                types.Number  `json:"apr"`
	OrderID            uint64        `json:"orderId"`
	PurchaseTime       types.Time    `json:"purchaseTime"`
	OptionType         string        `json:"optionType"`
}
