package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FlexibleLoanBorrowableCurrencyResponse holds a currency a flexible loan can borrow
type FlexibleLoanBorrowableCurrencyResponse struct {
	BorrowCurrency currency.Code `json:"borrowCcy"`
}

// FlexibleLoanCollateralAssetsRequest holds the currency filter and flexible loan whose collateral to return
type FlexibleLoanCollateralAssetsRequest struct {
	Currency currency.Code
	OrderID  string
}

// FlexibleLoanCollateralAssetsResponse holds the collateral a flexible loan holds
type FlexibleLoanCollateralAssetsResponse struct {
	Assets []FlexibleLoanCollateralAsset `json:"assets"`
}

// FlexibleLoanCollateralAsset holds a flexible loan's collateral in one currency
type FlexibleLoanCollateralAsset struct {
	Currency    currency.Code `json:"ccy"`
	Amount      types.Number  `json:"amt"`
	NotionalUSD types.Number  `json:"notionalUsd"`
}

// FlexibleLoanMaximumLoanRequest holds the currency to borrow, the flexible loan and any supplementary collateral to
// size a maximum loan with
type FlexibleLoanMaximumLoanRequest struct {
	BorrowCurrency          currency.Code                `json:"borrowCcy"`
	OrderID                 string                       `json:"ordId,omitempty"`
	SupplementaryCollateral []FlexibleLoanCurrencyAmount `json:"supCollateral,omitempty"`
}

// FlexibleLoanCurrencyAmount holds a currency and amount a flexible loan request borrows or posts as collateral
type FlexibleLoanCurrencyAmount struct {
	Currency currency.Code `json:"ccy"`
	Amount   float64       `json:"amt,string"`
}

// FlexibleLoanMaximumLoanResponse holds the most a flexible loan can borrow of a currency
type FlexibleLoanMaximumLoanResponse struct {
	BorrowCurrency currency.Code `json:"borrowCcy"`
	MaximumLoan    types.Number  `json:"maxLoan"`
	NotionalUSD    types.Number  `json:"notionalUsd"`
	RemainingQuota types.Number  `json:"remainingQuota"`
}

// FlexibleLoanMaximumCollateralRedeemRequest holds the collateral currency and flexible loan to size a collateral
// release for
type FlexibleLoanMaximumCollateralRedeemRequest struct {
	Currency currency.Code
	OrderID  string
}

// FlexibleLoanMaximumCollateralRedeemResponse holds the most collateral of a currency a flexible loan can release
type FlexibleLoanMaximumCollateralRedeemResponse struct {
	Currency            currency.Code `json:"ccy"`
	MaximumRedeemAmount types.Number  `json:"maxRedeemAmt"`
}

// FlexibleLoanAdjustCollateralRequest holds a flexible loan collateral addition or reduction
type FlexibleLoanAdjustCollateralRequest struct {
	Type               string        `json:"type"`
	CollateralCurrency currency.Code `json:"collateralCcy"`
	CollateralAmount   float64       `json:"collateralAmt,string"`
	OrderID            string        `json:"ordId,omitempty"`
}

// FlexibleLoanBorrowRequest holds a new flexible loan, or a further borrow on the loan its order ID names
type FlexibleLoanBorrowRequest struct {
	OrderID        string                       `json:"ordId,omitempty"`
	LoanData       FlexibleLoanCurrencyAmount   `json:"loanData"`
	CollateralData []FlexibleLoanCurrencyAmount `json:"collateralData,omitempty"`
	EMode          string                       `json:"eMode,omitempty"`
	ClientOrderID  string                       `json:"clOrdId"`
}

// FlexibleLoanBorrowResponse holds the flexible loan a borrow created or added to
type FlexibleLoanBorrowResponse struct {
	OrderID       string `json:"ordId"`
	EMode         string `json:"eMode"`
	ClientOrderID string `json:"clOrdId"`
}

// FlexibleLoanRepayRequest holds a flexible loan repayment
type FlexibleLoanRepayRequest struct {
	OrderID       string        `json:"ordId"`
	Currency      currency.Code `json:"ccy"`
	Amount        float64       `json:"amt,string"`
	ClientOrderID string        `json:"clOrdId"`
}

// FlexibleLoanRepayResponse holds a flexible loan repayment
type FlexibleLoanRepayResponse struct {
	OrderID       string        `json:"ordId"`
	Currency      currency.Code `json:"ccy"`
	Amount        types.Number  `json:"amt"`
	ClientOrderID string        `json:"clOrdId"`
}

// FlexibleLoanEModeInfoResponse holds a collateral and borrow currency pair eligible for flexible loan E-Mode
type FlexibleLoanEModeInfoResponse struct {
	CollateralCurrency currency.Code `json:"collateralCcy"`
	BorrowCurrency     currency.Code `json:"borrowCcy"`
}

// FlexibleLoanInfoResponse holds a flexible loan's borrowing, collateral and risk
type FlexibleLoanInfoResponse struct {
	OrderID               string                  `json:"ordId"`
	LoanNotionalUSD       types.Number            `json:"loanNotionalUsd"`
	LoanData              []FlexibleLoanAsset     `json:"loanData"`
	CollateralNotionalUSD types.Number            `json:"collateralNotionalUsd"`
	CollateralData        []FlexibleLoanAsset     `json:"collateralData"`
	RiskWarningData       FlexibleLoanRiskWarning `json:"riskWarningData"`
	CurrentLTV            types.Number            `json:"curLTV"`
	MarginCallLTV         types.Number            `json:"marginCallLTV"`
	LiquidationLTV        types.Number            `json:"liqLTV"`
}

// FlexibleLoanAsset holds a currency and amount a flexible loan borrows or holds as collateral
type FlexibleLoanAsset struct {
	Currency currency.Code `json:"ccy"`
	Amount   types.Number  `json:"amt"`
}

// FlexibleLoanRiskWarning holds the liquidation instrument and price of a flexible loan with one collateral and one
// borrowed currency; OKX sends both empty otherwise
type FlexibleLoanRiskWarning struct {
	InstrumentID     string       `json:"instId"`
	LiquidationPrice types.Number `json:"liqPx"`
}

// FlexibleLoanHistoryRequest holds the filters and reference ID cursors for flexible loan history
type FlexibleLoanHistoryRequest struct {
	Type string
	// After returns records earlier than this reference ID
	After string
	// Before returns records newer than this reference ID
	Before  string
	Limit   uint64
	OrderID string
}

// FlexibleLoanHistoryResponse holds a flexible loan action
type FlexibleLoanHistoryResponse struct {
	ReferenceID string        `json:"refId"`
	Type        string        `json:"type"`
	Currency    currency.Code `json:"ccy"`
	Amount      types.Number  `json:"amt"`
	Timestamp   types.Time    `json:"ts"`
}

// FlexibleLoanAccruedInterestRequest holds the filters and reference ID cursors for flexible loan accrued interest
type FlexibleLoanAccruedInterestRequest struct {
	Currency currency.Code
	// After returns records earlier than this reference ID
	After string
	// Before returns records newer than this reference ID
	Before  string
	Limit   uint64
	OrderID string
}

// FlexibleLoanAccruedInterestResponse holds an interest accrual on a flexible loan
type FlexibleLoanAccruedInterestResponse struct {
	ReferenceID  string        `json:"refId"`
	Currency     currency.Code `json:"ccy"`
	Loan         types.Number  `json:"loan"`
	Interest     types.Number  `json:"interest"`
	InterestRate types.Number  `json:"interestRate"`
	Timestamp    types.Time    `json:"ts"`
}
