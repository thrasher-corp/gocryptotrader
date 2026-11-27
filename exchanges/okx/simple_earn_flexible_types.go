package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SavingBalanceResponse holds the account's savings balance in one currency
type SavingBalanceResponse struct {
	Currency         currency.Code `json:"ccy"`
	Amount           types.Number  `json:"amt"`
	Earnings         types.Number  `json:"earnings"`
	Rate             types.Number  `json:"rate"`
	LoanAmount       types.Number  `json:"loanAmt"`
	PendingAmount    types.Number  `json:"pendingAmt"`
	RedemptionAmount types.Number  `json:"redemptAmt"` // deprecated by OKX
}

// SavingsPurchaseRedemptionRequest holds a savings purchase or redemption
type SavingsPurchaseRedemptionRequest struct {
	Currency currency.Code `json:"ccy"`
	Amount   float64       `json:"amt,string"`
	Side     string        `json:"side"`
	// Rate is the minimum annual lending rate of a purchase, which replaces the previous purchase's rate
	Rate float64 `json:"rate,string,omitempty"`
}

// SavingsPurchaseRedemptionResponse holds a savings purchase or redemption
type SavingsPurchaseRedemptionResponse struct {
	Currency currency.Code `json:"ccy"`
	Amount   types.Number  `json:"amt"`
	Side     string        `json:"side"`
	Rate     types.Number  `json:"rate"`
}

// LendingHistoryRequest holds the currency filter and lending time cursors for savings lending history
type LendingHistoryRequest struct {
	Currency currency.Code
	// After returns records earlier than this time
	After time.Time
	// Before returns records newer than this time
	Before time.Time
	Limit  uint64
}

// LendingHistoryResponse holds a record of the account's savings lending
type LendingHistoryResponse struct {
	Currency  currency.Code `json:"ccy"`
	Amount    types.Number  `json:"amt"`
	Earnings  types.Number  `json:"earnings"`
	Rate      types.Number  `json:"rate"`
	Timestamp types.Time    `json:"ts"`
}

// PublicBorrowInfoResponse holds a currency's market borrowing rates
type PublicBorrowInfoResponse struct {
	Currency         currency.Code `json:"ccy"`
	AverageAmount    types.Number  `json:"avgAmt"`    // deprecated by OKX, which sends it empty
	AverageAmountUSD types.Number  `json:"avgAmtUsd"` // deprecated by OKX, which sends it empty
	AverageRate      types.Number  `json:"avgRate"`
	PreviousRate     types.Number  `json:"preRate"`
	EstimatedRate    types.Number  `json:"estRate"`
}

// PublicBorrowHistoryRequest holds the currency filter and time cursors for the market's borrowing rate history
type PublicBorrowHistoryRequest struct {
	Currency currency.Code
	// After returns records earlier than this time
	After time.Time
	// Before returns records newer than this time
	Before time.Time
	Limit  uint64
}

// PublicBorrowHistoryResponse holds a currency's market borrowing and lending rates at a point in time
type PublicBorrowHistoryResponse struct {
	Currency    currency.Code `json:"ccy"`
	Amount      types.Number  `json:"amt"` // deprecated by OKX, which sends it empty
	Rate        types.Number  `json:"rate"`
	LendingRate types.Number  `json:"lendingRate"`
	Timestamp   types.Time    `json:"ts"`
}
