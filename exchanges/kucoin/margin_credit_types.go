package kucoin

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// LendingCurrencyInfo is a currency's margin lending market configuration
type LendingCurrencyInfo struct {
	Currency       currency.Code `json:"currency"`
	PurchaseEnable bool          `json:"purchaseEnable"`
	RedeemEnable   bool          `json:"redeemEnable"`
	// Increment is the precision of purchase and redemption amounts
	Increment          types.Number `json:"increment"`
	MinPurchaseSize    types.Number `json:"minPurchaseSize"`
	MinInterestRate    types.Number `json:"minInterestRate"`
	MaxInterestRate    types.Number `json:"maxInterestRate"`
	InterestIncrement  types.Number `json:"interestIncrement"`
	MaxPurchaseSize    types.Number `json:"maxPurchaseSize"`
	MarketInterestRate types.Number `json:"marketInterestRate"`
	AutoPurchaseEnable bool         `json:"autoPurchaseEnable"`
}

// LoanMarketInterestRate is the margin lending market's interest rate for an hour
type LoanMarketInterestRate struct {
	Time               LoanMarketInterestRateTime `json:"time"`
	MarketInterestRate types.Number               `json:"marketInterestRate"`
}

// LoanMarketInterestRateTime is the start of the hour a loan market interest rate applies to, which KuCoin sends as
// YYYYMMDDHH00 in UTC+8
type LoanMarketInterestRateTime time.Time

// UnmarshalJSON decodes a YYYYMMDDHH00 time in UTC+8 into UTC
func (t *LoanMarketInterestRateTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%w %s: %w", errInvalidLoanMarketInterestRateTime, data, err)
	}
	if s == "" {
		*t = LoanMarketInterestRateTime{}
		return nil
	}
	tm, err := time.ParseInLocation("200601021504", s, time.FixedZone("UTC+8", 8*60*60))
	if err != nil {
		return fmt.Errorf("%w %q: %w", errInvalidLoanMarketInterestRateTime, s, err)
	}
	*t = LoanMarketInterestRateTime(tm.UTC())
	return nil
}

// Time returns the time as a time.Time
func (t LoanMarketInterestRateTime) Time() time.Time {
	return time.Time(t)
}

// MarginLendingSubscriptionRequest holds the Purchase parameters
type MarginLendingSubscriptionRequest struct {
	Currency     currency.Code
	Size         float64
	InterestRate float64
}

// OrderNumberResponse identifies a purchase or redemption order
type OrderNumberResponse struct {
	OrderNumber string `json:"orderNo"`
}

// ModifySubscriptionOrderRequest holds the Modify Purchase parameters
type ModifySubscriptionOrderRequest struct {
	Currency            currency.Code
	InterestRate        float64
	PurchaseOrderNumber string
}

// SubscriptionOrdersRequest holds the Get Purchase Orders filters
type SubscriptionOrdersRequest struct {
	// Status is DONE or PENDING
	Status              string
	Currency            currency.Code
	PurchaseOrderNumber string
	CurrentPage         uint64
	// PageSize is 1 to 50
	PageSize uint64
}

// SubscriptionOrdersResponse holds a page of margin lending purchase orders
type SubscriptionOrdersResponse struct {
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []SubscriptionOrder `json:"items"`
}

// SubscriptionOrder is a margin lending purchase order
type SubscriptionOrder struct {
	Currency            currency.Code `json:"currency"`
	PurchaseOrderNumber string        `json:"purchaseOrderNo"`
	PurchaseSize        types.Number  `json:"purchaseSize"`
	// MatchSize is the amount lent out
	MatchSize types.Number `json:"matchSize"`
	// InterestRate is the target annualised interest rate
	InterestRate types.Number `json:"interestRate"`
	// IncomeSize is the amount redeemed
	IncomeSize types.Number `json:"incomeSize"`
	ApplyTime  types.Time   `json:"applyTime"`
	// Status is DONE or PENDING
	Status string `json:"status"`
}

// RedemptionRequest holds the Redeem parameters
type RedemptionRequest struct {
	Currency            currency.Code
	Size                float64
	PurchaseOrderNumber string
}

// RedemptionOrdersRequest holds the Get Redeem Orders filters
type RedemptionOrdersRequest struct {
	// Status is DONE or PENDING
	Status            string
	Currency          currency.Code
	RedeemOrderNumber string
	CurrentPage       uint64
	// PageSize is 1 to 50
	PageSize uint64
}

// RedemptionOrdersResponse holds a page of margin lending redemption orders
type RedemptionOrdersResponse struct {
	CurrentPage uint64            `json:"currentPage"`
	PageSize    uint64            `json:"pageSize"`
	TotalNumber uint64            `json:"totalNum"`
	TotalPage   uint64            `json:"totalPage"`
	Items       []RedemptionOrder `json:"items"`
}

// RedemptionOrder is a margin lending redemption order
type RedemptionOrder struct {
	Currency            currency.Code `json:"currency"`
	PurchaseOrderNumber string        `json:"purchaseOrderNo"`
	RedeemOrderNumber   string        `json:"redeemOrderNo"`
	RedeemSize          types.Number  `json:"redeemSize"`
	// ReceiptSize is the amount redeemed
	ReceiptSize types.Number `json:"receiptSize"`
	// ApplyTime is typed as a string in KuCoin's docs, whose example shows null
	ApplyTime types.Time `json:"applyTime"`
	// Status is DONE or PENDING
	Status string `json:"status"`
}
