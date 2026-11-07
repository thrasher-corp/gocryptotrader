package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// LegacyCryptoLoanCollateralCoinsResponse holds the legacy crypto loan collateral coins of each VIP level
type LegacyCryptoLoanCollateralCoinsResponse struct {
	VIPCoinList []LegacyCryptoLoanVIPCollateralCoins `json:"vipCoinList"`
}

// LegacyCryptoLoanVIPCollateralCoins holds the legacy crypto loan collateral coins of a VIP level
type LegacyCryptoLoanVIPCollateralCoins struct {
	List     []LegacyCryptoLoanCollateralCoin `json:"list"`
	VIPLevel string                           `json:"vipLevel"`
}

// LegacyCryptoLoanCollateralCoin holds a legacy crypto loan collateral coin's precision, LTV thresholds and limit
type LegacyCryptoLoanCollateralCoin struct {
	CollateralAccuracy uint64        `json:"collateralAccuracy"` // Decimal places
	Currency           currency.Code `json:"currency"`           // Sent although the docs table omits it
	InitialLTV         types.Number  `json:"initialLTV"`
	MarginCallLTV      types.Number  `json:"marginCallLTV"`
	LiquidationLTV     types.Number  `json:"liquidationLTV"`
	MaxLimit           types.Number  `json:"maxLimit"`
}

// LegacyCryptoLoanBorrowableCoinsResponse holds the legacy crypto loan borrowable coins of each VIP level
type LegacyCryptoLoanBorrowableCoinsResponse struct {
	VIPCoinList []LegacyCryptoLoanVIPBorrowableCoins `json:"vipCoinList"`
}

// LegacyCryptoLoanVIPBorrowableCoins holds the legacy crypto loan borrowable coins of a VIP level
type LegacyCryptoLoanVIPBorrowableCoins struct {
	List     []LegacyCryptoLoanBorrowableCoin `json:"list"`
	VIPLevel string                           `json:"vipLevel"`
}

// LegacyCryptoLoanBorrowableCoin holds a legacy crypto loan borrowable coin's precision, hourly interest rates and
// borrowing limits. A rate is 0 when the coin does not support that loan term
type LegacyCryptoLoanBorrowableCoin struct {
	BorrowingAccuracy          uint64        `json:"borrowingAccuracy"` // Decimal places
	Currency                   currency.Code `json:"currency"`
	FlexibleHourlyInterestRate types.Number  `json:"flexibleHourlyInterestRate"`
	HourlyInterestRate7Day     types.Number  `json:"hourlyInterestRate7D"`
	HourlyInterestRate14Day    types.Number  `json:"hourlyInterestRate14D"`
	HourlyInterestRate30Day    types.Number  `json:"hourlyInterestRate30D"`
	HourlyInterestRate90Day    types.Number  `json:"hourlyInterestRate90D"`
	HourlyInterestRate180Day   types.Number  `json:"hourlyInterestRate180D"`
	MaxBorrowingAmount         types.Number  `json:"maxBorrowingAmount"`
	MinBorrowingAmount         types.Number  `json:"minBorrowingAmount"`
}

// LegacyCryptoLoanBorrowableCollateralisableLimitResponse holds the amounts an account can borrow and put up as
// collateral for a legacy crypto loan coin pair
type LegacyCryptoLoanBorrowableCollateralisableLimitResponse struct {
	CollateralCurrency  currency.Code `json:"collateralCurrency"`
	LoanCurrency        currency.Code `json:"loanCurrency"`
	MaxCollateralAmount types.Number  `json:"maxCollateralAmount"`
	MaxLoanAmount       types.Number  `json:"maxLoanAmount"`
	MinCollateralAmount types.Number  `json:"minCollateralAmount"`
	MinLoanAmount       types.Number  `json:"minLoanAmount"`
}

// LegacyCryptoLoanUnpaidLoansRequest holds the filters of GetLegacyCryptoLoanUnpaidLoans; every field is optional
type LegacyCryptoLoanUnpaidLoansRequest struct {
	OrderID            string
	LoanCurrency       currency.Code
	CollateralCurrency currency.Code
	LoanTermType       string // 1: fixed term, which needs LoanTerm; 2: flexible term; empty: both
	LoanTerm           uint64 // Days, used when LoanTermType is 1
	Limit              uint64
	Cursor             string
}

// LegacyCryptoLoanUnpaidLoansResponse holds a page of unpaid legacy crypto loans
type LegacyCryptoLoanUnpaidLoansResponse struct {
	List           []LegacyCryptoLoanUnpaidLoan `json:"list"`
	NextPageCursor string                       `json:"nextPageCursor"`
}

// LegacyCryptoLoanUnpaidLoan holds an unpaid legacy crypto loan
type LegacyCryptoLoanUnpaidLoan struct {
	CollateralAmount   types.Number  `json:"collateralAmount"`
	CollateralCurrency currency.Code `json:"collateralCurrency"`
	CurrentLTV         types.Number  `json:"currentLTV"`
	ExpirationTime     types.Time    `json:"expirationTime"` // Zero for flexible loans
	// HourlyInterestRate is the real-time rate of a flexible loan or the fixed rate of a fixed term loan
	HourlyInterestRate      types.Number  `json:"hourlyInterestRate"`
	LoanCurrency            currency.Code `json:"loanCurrency"`
	LoanTerm                types.Number  `json:"loanTerm"` // Days; 0 for flexible loans
	OrderID                 string        `json:"orderId"`
	ResidualInterest        types.Number  `json:"residualInterest"` // Unpaid interest
	ResidualPenaltyInterest types.Number  `json:"residualPenaltyInterest"`
	TotalDebt               types.Number  `json:"totalDebt"` // Unpaid principal
}

// LegacyCryptoLoanRepaymentHistoryRequest holds the filters of GetLegacyCryptoLoanRepaymentHistory; every field is
// optional
type LegacyCryptoLoanRepaymentHistoryRequest struct {
	OrderID      string
	RepayID      string
	LoanCurrency currency.Code
	Limit        uint64
	Cursor       string
}

// LegacyCryptoLoanRepaymentHistoryResponse holds a page of legacy crypto loan repayments
type LegacyCryptoLoanRepaymentHistoryResponse struct {
	List           []LegacyCryptoLoanRepayment `json:"list"`
	NextPageCursor string                      `json:"nextPageCursor"`
}

// LegacyCryptoLoanRepayment holds a legacy crypto loan repayment
type LegacyCryptoLoanRepayment struct {
	CollateralCurrency currency.Code `json:"collateralCurrency"`
	CollateralReturn   types.Number  `json:"collateralReturn"` // Collateral returned by the final repayment, otherwise 0
	LoanCurrency       currency.Code `json:"loanCurrency"`
	LoanTerm           types.Number  `json:"loanTerm"` // Days; 0 for flexible loans
	OrderID            string        `json:"orderId"`
	RepayAmount        types.Number  `json:"repayAmount"`
	RepayID            string        `json:"repayId"`
	RepayStatus        uint64        `json:"repayStatus"` // 1: success, 2: processing
	RepayTime          types.Time    `json:"repayTime"`
	RepayType          string        `json:"repayType"` // 1: repaid by the user, 2: repaid by liquidation
}

// LegacyCryptoLoanCompletedLoanHistoryRequest holds the filters of GetLegacyCryptoLoanCompletedLoanHistory; every
// field is optional
type LegacyCryptoLoanCompletedLoanHistoryRequest struct {
	OrderID            string
	LoanCurrency       currency.Code
	CollateralCurrency currency.Code
	Limit              uint64
	Cursor             string
}

// LegacyCryptoLoanCompletedLoanHistoryResponse holds a page of fully repaid legacy crypto loans
type LegacyCryptoLoanCompletedLoanHistoryResponse struct {
	List           []LegacyCryptoLoanCompletedLoan `json:"list"`
	NextPageCursor string                          `json:"nextPageCursor"`
}

// LegacyCryptoLoanCompletedLoan holds a fully repaid legacy crypto loan
type LegacyCryptoLoanCompletedLoan struct {
	BorrowTime         types.Time    `json:"borrowTime"`
	CollateralCurrency currency.Code `json:"collateralCurrency"`
	ExpirationTime     types.Time    `json:"expirationTime"` // Zero for flexible loans
	// HourlyInterestRate is the real-time rate of a flexible loan or the fixed rate of a fixed term loan
	HourlyInterestRate      types.Number  `json:"hourlyInterestRate"`
	InitialCollateralAmount types.Number  `json:"initialCollateralAmount"`
	InitialLoanAmount       types.Number  `json:"initialLoanAmount"`
	LoanCurrency            currency.Code `json:"loanCurrency"`
	LoanTerm                types.Number  `json:"loanTerm"` // Days; 0 for flexible loans
	OrderID                 string        `json:"orderId"`
	RepaidInterest          types.Number  `json:"repaidInterest"`
	RepaidPenaltyInterest   types.Number  `json:"repaidPenaltyInterest"`
	Status                  uint64        `json:"status"` // 1: fully repaid manually, 2: fully repaid by liquidation
}

// AdjustLegacyCryptoLoanCollateralAmountRequest holds the parameters of AdjustLegacyCryptoLoanCollateralAmount
type AdjustLegacyCryptoLoanCollateralAmountRequest struct {
	OrderID   string
	Amount    float64
	Direction string // 0: add collateral, 1: reduce collateral
}

// AdjustLegacyCryptoLoanCollateralAmountResponse holds the ID of a legacy crypto loan collateral adjustment
type AdjustLegacyCryptoLoanCollateralAmountResponse struct {
	AdjustID string `json:"adjustId"`
}

// LegacyCryptoLoanLTVAdjustmentHistoryRequest holds the filters of GetLegacyCryptoLoanLTVAdjustmentHistory; every
// field is optional
type LegacyCryptoLoanLTVAdjustmentHistoryRequest struct {
	OrderID            string
	AdjustID           string
	CollateralCurrency currency.Code
	Limit              uint64
	Cursor             string
}

// LegacyCryptoLoanLTVAdjustmentHistoryResponse holds a page of legacy crypto loan collateral adjustments
type LegacyCryptoLoanLTVAdjustmentHistoryResponse struct {
	List           []LegacyCryptoLoanLTVAdjustment `json:"list"`
	NextPageCursor string                          `json:"nextPageCursor"`
}

// LegacyCryptoLoanLTVAdjustment holds a legacy crypto loan collateral adjustment and the LTV before and after it
type LegacyCryptoLoanLTVAdjustment struct {
	CollateralCurrency currency.Code `json:"collateralCurrency"`
	OrderID            string        `json:"orderId"`
	AdjustID           string        `json:"adjustId"`
	AdjustTime         types.Time    `json:"adjustTime"`
	PreLTV             types.Number  `json:"preLTV"`
	AfterLTV           types.Number  `json:"afterLTV"`
	Direction          uint64        `json:"direction"` // 0: add collateral, 1: reduce collateral
	Amount             types.Number  `json:"amount"`    // Sent although the docs table omits it
}
