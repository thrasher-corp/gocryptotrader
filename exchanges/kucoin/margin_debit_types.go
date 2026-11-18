package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BorrowInterestRateRequest holds the Get Borrow Interest Rate filters
type BorrowInterestRateRequest struct {
	// VIPLevel selects the VIP level, 0 to 12, whose rates are returned; nil returns the account's own level
	VIPLevel *uint64
	// Currencies takes up to 50 currencies
	Currencies []currency.Code
}

// BorrowInterestRateResponse holds the borrowing interest rates of a VIP level
type BorrowInterestRateResponse struct {
	VIPLevel uint64               `json:"vipLevel"`
	Items    []BorrowInterestRate `json:"items"`
}

// BorrowInterestRate is the interest rate charged on borrowing a currency
type BorrowInterestRate struct {
	Currency             currency.Code `json:"currency"`
	HourlyBorrowRate     types.Number  `json:"hourlyBorrowRate"`
	AnnualizedBorrowRate types.Number  `json:"annualizedBorrowRate"`
}

// MarginBorrowOrderRequest holds the Borrow parameters
type MarginBorrowOrderRequest struct {
	Currency currency.Code
	Size     float64
	// TimeInForce is IOC or FOK
	TimeInForce string
	// Symbol is required for the isolated margin account
	Symbol     string
	IsIsolated bool
	// IsHF borrows with high frequency borrowing rather than low frequency borrowing
	IsHF bool
}

// MarginBorrowOrderResponse identifies a borrow order
type MarginBorrowOrderResponse struct {
	OrderNumber string       `json:"orderNo"`
	ActualSize  types.Number `json:"actualSize"`
}

// MarginBorrowingHistoryRequest holds the Get Borrow History filters
type MarginBorrowingHistoryRequest struct {
	Currency   currency.Code
	IsIsolated bool
	// Symbol is required for the isolated margin account
	Symbol      string
	OrderNumber string
	StartTime   time.Time
	EndTime     time.Time
	CurrentPage uint64
	// PageSize is 10 to 500
	PageSize uint64
}

// MarginBorrowingHistoryResponse holds a page of borrow orders
type MarginBorrowingHistoryResponse struct {
	Timestamp   types.Time          `json:"timestamp"`
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []MarginBorrowOrder `json:"items"`
}

// MarginBorrowOrder is a borrow order
type MarginBorrowOrder struct {
	OrderNumber string `json:"orderNo"`
	// Symbol is empty for the cross margin account
	Symbol     string        `json:"symbol"`
	Currency   currency.Code `json:"currency"`
	Size       types.Number  `json:"size"`
	ActualSize types.Number  `json:"actualSize"`
	// Status is PENDING, SUCCESS or FAILED
	Status      string     `json:"status"`
	CreatedTime types.Time `json:"createdTime"`
}

// MarginRepaymentRequest holds the Repay parameters
type MarginRepaymentRequest struct {
	Currency currency.Code
	Size     float64
	// Symbol is required for the isolated margin account
	Symbol     string
	IsIsolated bool
	// IsHF repays high frequency borrowing rather than low frequency borrowing
	IsHF bool
}

// MarginRepaymentResponse identifies a repayment order
type MarginRepaymentResponse struct {
	Timestamp   types.Time   `json:"timestamp"`
	OrderNumber string       `json:"orderNo"`
	ActualSize  types.Number `json:"actualSize"`
}

// MarginRepaymentHistoryRequest holds the Get Repay History filters
type MarginRepaymentHistoryRequest struct {
	Currency   currency.Code
	IsIsolated bool
	// Symbol is required for the isolated margin account
	Symbol      string
	OrderNumber string
	StartTime   time.Time
	EndTime     time.Time
	CurrentPage uint64
	// PageSize is 10 to 500
	PageSize uint64
}

// MarginRepaymentHistoryResponse holds a page of repayment orders
type MarginRepaymentHistoryResponse struct {
	Timestamp   types.Time             `json:"timestamp"`
	CurrentPage uint64                 `json:"currentPage"`
	PageSize    uint64                 `json:"pageSize"`
	TotalNumber uint64                 `json:"totalNum"`
	TotalPage   uint64                 `json:"totalPage"`
	Items       []MarginRepaymentOrder `json:"items"`
}

// MarginRepaymentOrder is a repayment order
type MarginRepaymentOrder struct {
	OrderNumber string `json:"orderNo"`
	// Symbol is empty for the cross margin account
	Symbol    string        `json:"symbol"`
	Currency  currency.Code `json:"currency"`
	Size      types.Number  `json:"size"`
	Principal types.Number  `json:"principal"`
	Interest  types.Number  `json:"interest"`
	// Status is PENDING, SUCCESS or FAILED
	Status      string     `json:"status"`
	CreatedTime types.Time `json:"createdTime"`
}

// MarginInterestRecordsRequest holds the Get Interest History filters
type MarginInterestRecordsRequest struct {
	Currency   currency.Code
	IsIsolated bool
	// Symbol is required for the isolated margin account
	Symbol      string
	StartTime   time.Time
	EndTime     time.Time
	CurrentPage uint64
	// PageSize is 10 to 500
	PageSize uint64
}

// MarginInterestRecordsResponse holds a page of margin interest records
type MarginInterestRecordsResponse struct {
	Timestamp   types.Time             `json:"timestamp"`
	CurrentPage uint64                 `json:"currentPage"`
	PageSize    uint64                 `json:"pageSize"`
	TotalNumber uint64                 `json:"totalNum"`
	TotalPage   uint64                 `json:"totalPage"`
	Items       []MarginInterestRecord `json:"items"`
}

// MarginInterestRecord is the interest charged on a margin loan
type MarginInterestRecord struct {
	Currency currency.Code `json:"currency"`
	// DayRatio is the daily interest rate
	DayRatio       types.Number `json:"dayRatio"`
	InterestAmount types.Number `json:"interestAmount"`
	CreatedTime    types.Time   `json:"createdTime"`
}

// ModifyLeverageMultiplierRequest holds the Modify Leverage parameters
type ModifyLeverageMultiplierRequest struct {
	// Symbol is required for the isolated margin account
	Symbol     string
	IsIsolated bool
	// Leverage must exceed 1, have at most two decimal places and lie between the account's current debt leverage
	// and the system maximum
	Leverage float64
}
