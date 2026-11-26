package bingx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DualCurrencyProductsRequest holds the parameters of Dual-Currency Product List; durations are in days
type DualCurrencyProductsRequest struct {
	// PageIndex counts from 0
	PageIndex uint64
	// PageSize is 10 when zero, and at most 20
	PageSize uint64
	// BeginDuration and EndDuration bound the durations listed; without either, every duration is listed
	BeginDuration   uint64
	EndDuration     uint64
	Duration        uint64
	InvestmentAsset currency.Code
	ExerciseAsset   currency.Code
	Direction       DualCurrencyDirection
}

// DualCurrencyProductsResponse holds the dual-currency products of an investment and exercise asset pair
type DualCurrencyProductsResponse struct {
	DualTitle       string                `json:"dualTitle"`
	InvestmentAsset currency.Code         `json:"investmentAsset"`
	ExerciseAsset   currency.Code         `json:"exerciseAsset"`
	Direction       DualCurrencyDirection `json:"direction"`
	// RecommendVO is the recommended product, which only the unfiltered list may include
	RecommendVO       DualCurrencyRecommendedProduct `json:"recommendVo"`
	SubVOs            []DualCurrencyProductGroup     `json:"subVos"`
	DurationRangeList []DualCurrencyDurationRange    `json:"durationRangeList"`
}

// DualCurrencyRecommendedProduct is the dual-currency product BingX recommends
type DualCurrencyRecommendedProduct struct {
	SKU       string `json:"sku"`
	ProductID uint64 `json:"productId"`
	// Duration is in days
	Duration        uint64       `json:"duration"`
	TimeDescription string       `json:"timeDesc"`
	StrikePrice     types.Number `json:"strikePrice"`
	APY             types.Number `json:"apy"`
}

// DualCurrencyProductGroup holds the dual-currency products of a duration
type DualCurrencyProductGroup struct {
	// Duration is in days
	Duration        uint64 `json:"duration"`
	Description     string `json:"desc"`
	TimeDescription string `json:"timeDesc"`
	// ExpiryTime is in a format the docs leave undocumented
	ExpiryTime string                `json:"expiryTime"`
	Items      []DualCurrencyProduct `json:"items"`
}

// DualCurrencyProduct is a dual-currency product: a duration and strike price
type DualCurrencyProduct struct {
	SKU       string `json:"sku"`
	ProductID uint64 `json:"productId"`
	// Duration is in days
	Duration    uint64       `json:"duration"`
	StrikePrice types.Number `json:"strikePrice"`
	// TargetDeviationPercentage is the strike price's deviation from the target price, as a percentage
	TargetDeviationPercentage types.Number `json:"targetDeviationPct"`
	APY                       types.Number `json:"apy"`
	// SoldOut is undocumented; the documented product reports 0
	SoldOut uint64 `json:"soldOut"`
}

// DualCurrencyDurationRange is a range of durations the dual-currency products are grouped by, in days
type DualCurrencyDurationRange struct {
	LowerLimit  uint64 `json:"lowerLimit"`
	UpperLimit  uint64 `json:"upperLimit"`
	Description string `json:"desc"`
}

// DualCurrencyPreOrderQuoteRequest holds the parameters of Dual-Currency Pre-Order Quote, all required, which come from
// a product of the dual-currency product list
type DualCurrencyPreOrderQuoteRequest struct {
	SKU         string
	StrikePrice float64
	ProductID   uint64
}

// DualCurrencyPreOrderQuoteResponse holds a dual-currency product's quote, which an order needs
type DualCurrencyPreOrderQuoteResponse struct {
	DualTitle         string                `json:"dualTitle"`
	TimeDescription   string                `json:"timeDesc"`
	SKU               string                `json:"sku"`
	ProductID         uint64                `json:"productId"`
	InvestmentAsset   currency.Code         `json:"investmentAsset"`
	QuotaAsset        currency.Code         `json:"quotaAsset"`
	InvestPrecision   uint64                `json:"investPrecision"`
	ExercisePrecision uint64                `json:"exercisePrecision"`
	ExpiryPrecision   uint64                `json:"expiryPrecision"`
	ExerciseAsset     currency.Code         `json:"exerciseAsset"`
	Direction         DualCurrencyDirection `json:"direction"`
	// Duration is in days
	Duration uint64 `json:"duration"`
	// PurchaseEndTime is in a format the docs leave undocumented
	PurchaseEndTime string       `json:"purchaseEndTime"`
	EarnRate        types.Number `json:"earnRate"`
	APY             types.Number `json:"apy"`
	StrikePrice     types.Number `json:"strikePrice"`
	// TargetDeviationPercentage is the strike price's deviation from the target price, as a percentage
	TargetDeviationPercentage types.Number `json:"targetDeviationPct"`
	MinAmount                 types.Number `json:"minAmount"`
	MaxAmount                 types.Number `json:"maxAmount"`
	// ExpiryTime is in a format the docs leave undocumented
	ExpiryTime string `json:"expiryTime"`
	// SettledTime is an hour after ExpiryTime, in a format the docs leave undocumented
	SettledTime string `json:"settledTime"`
	// StartCloseTime is in a format the docs leave undocumented
	StartCloseTime  string       `json:"startCloseTime"`
	WealthUserLimit types.Number `json:"wealthUserLimit"`
	// UserLimitLeft is the user's remaining purchase quota, -1 when unlimited
	UserLimitLeft    types.Number `json:"userLimitLeft"`
	ProductLeftQuota types.Number `json:"productLeftQuota"`
	AvailableAmount  types.Number `json:"availableAmount"`
	// FundAvailable is the fund account's available balance
	FundAvailable types.Number `json:"fundAvailable"`
	// SpotAvailable is the spot account's available balance
	SpotAvailable types.Number             `json:"spotAvailable"`
	QuoteID       string                   `json:"quoteId"`
	Message       string                   `json:"msg"`
	BusinessCode  DualCurrencyBusinessCode `json:"bizCode"`
}

// DualCurrencyOrderRequest holds the parameters of Dual-Currency Place Order. Amount, SKU, ProductID, StrikePrice and
// QuoteID are required, the last four from a pre-order quote
type DualCurrencyOrderRequest struct {
	InvestmentAsset currency.Code
	ExerciseAsset   currency.Code
	Amount          float64
	// SelectAccounts are the accounts the order is paid from
	SelectAccounts []DualCurrencyAccount
	SKU            string
	ProductID      uint64
	StrikePrice    float64
	QuoteID        string
	// SettleTargetAccount is the account the order settles to, the fund account when empty
	SettleTargetAccount DualCurrencyAccount
}

// DualCurrencyOrderResponse holds a placed dual-currency order
type DualCurrencyOrderResponse struct {
	DualTitle       string                `json:"dualTitle"`
	InvestmentAsset currency.Code         `json:"investmentAsset"`
	LinkedAsset     currency.Code         `json:"linkedAsset"`
	ExerciseAsset   currency.Code         `json:"exerciseAsset"`
	Amount          types.Number          `json:"amount"`
	SKU             string                `json:"sku"`
	ProductID       uint64                `json:"productId"`
	QuoteID         string                `json:"quoteId"`
	Direction       DualCurrencyDirection `json:"direction"`
	// ExpiryTime is in a format the docs leave undocumented
	ExpiryTime string `json:"expiryTime"`
	// SettledTime is an hour after ExpiryTime, in a format the docs leave undocumented
	SettledTime  string                   `json:"settledTime"`
	Message      string                   `json:"msg"`
	BusinessCode DualCurrencyBusinessCode `json:"bizCode"`
	OrderNumber  string                   `json:"orderNo"`
	// StartCloseTime is in a format the docs leave undocumented
	StartCloseTime string        `json:"startCloseTime"`
	APY            types.Number  `json:"apy"`
	StrikePrice    types.Number  `json:"strikePrice"`
	QuotaAsset     currency.Code `json:"quotaAsset"`
}

// DualCurrencyPositionResponse is a dual-currency position
type DualCurrencyPositionResponse struct {
	OrderNumber     string                  `json:"orderNo"`
	UID             uint64                  `json:"uid"`
	InvestmentAsset currency.Code           `json:"investmentAsset"`
	ExerciseAsset   currency.Code           `json:"exerciseAsset"`
	Direction       DualCurrencyDirection   `json:"direction"`
	StrikePrice     float64                 `json:"strikePrice"`
	Amount          float64                 `json:"amount"`
	APY             float64                 `json:"apy"`
	Status          DualCurrencyOrderStatus `json:"status"`
	// ExpiryTime is in a format the docs leave undocumented
	ExpiryTime string `json:"expiryTime"`
	// CreateTime is the purchase time, in a format the docs leave undocumented
	CreateTime string        `json:"createTime"`
	DualTitle  string        `json:"dualTitle"`
	QuotaAsset currency.Code `json:"quotaAsset"`
}

// DualCurrencyOrderRecordsRequest holds the parameters of Dual-Currency Order Records
type DualCurrencyOrderRecordsRequest struct {
	// OrderNumber is every order when empty
	OrderNumber string
	// PageID is the page number
	PageID   uint64
	PageSize uint64
}

// DualCurrencyOrderRecordsResponse holds a page of dual-currency orders
type DualCurrencyOrderRecordsResponse struct {
	Total   uint64                    `json:"total"`
	Pages   uint64                    `json:"pages"`
	Current uint64                    `json:"current"`
	Size    uint64                    `json:"size"`
	Records []DualCurrencyOrderRecord `json:"records"`
}

// DualCurrencyOrderRecord is a dual-currency order and its settlement
type DualCurrencyOrderRecord struct {
	OrderNumber     string                  `json:"orderNo"`
	UID             uint64                  `json:"uid"`
	Status          DualCurrencyOrderStatus `json:"status"`
	InvestmentAsset currency.Code           `json:"investmentAsset"`
	ExerciseAsset   currency.Code           `json:"exerciseAsset"`
	Direction       DualCurrencyDirection   `json:"direction"`
	Amount          float64                 `json:"amount"`
	APY             float64                 `json:"apy"`
	StrikePrice     float64                 `json:"strikePrice"`
	SettlePrice     float64                 `json:"settlePrice"`
	SettleAmount    float64                 `json:"settleAmount"`
	SettleAssetName currency.Code           `json:"settleAssetName"`
	// ExpiryTime is in a format the docs leave undocumented
	ExpiryTime string `json:"expiryTime"`
	// SettledTime is in a format the docs leave undocumented
	SettledTime string `json:"settledTime"`
	// CreateTime is the purchase time, in a format the docs leave undocumented
	CreateTime string `json:"createTime"`
	// Duration is in days
	Duration  uint64 `json:"duration"`
	SKU       string `json:"sku"`
	ProductID uint64 `json:"productId"`
}
