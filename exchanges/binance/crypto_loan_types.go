package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FlexibleLoanCollateralRepayRateResponse holds the collateral coin to loan coin rate of a flexible loan collateral
// repayment
type FlexibleLoanCollateralRepayRateResponse struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	Rate           types.Number  `json:"rate"`
}

// FlexibleLoanAdjustLTVRequest holds the parameters of FlexibleLoanAdjustLTV
type FlexibleLoanAdjustLTVRequest struct {
	LoanCoin         currency.Code
	CollateralCoin   currency.Code
	AdjustmentAmount float64
	Direction        string // ADDITIONAL or REDUCED
}

// FlexibleLoanAdjustLTVResponse holds the outcome of a flexible loan LTV adjustment
type FlexibleLoanAdjustLTVResponse struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	Direction        string        `json:"direction"`
	AdjustmentAmount types.Number  `json:"adjustmentAmount"`
	CurrentLTV       types.Number  `json:"currentLTV"`
	Status           string        `json:"status"`
}

// UnmarshalJSON decodes a flexible loan LTV adjustment. The v1 endpoint returned the adjusted amount as amount
// although its docs named it adjustedAmount, so amount is read when adjustmentAmount is absent
func (f *FlexibleLoanAdjustLTVResponse) UnmarshalJSON(data []byte) error {
	type alias FlexibleLoanAdjustLTVResponse
	aux := struct {
		*alias
		Amount types.Number `json:"amount"`
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if f.AdjustmentAmount == 0 {
		f.AdjustmentAmount = aux.Amount
	}
	return nil
}

// FlexibleLoanBorrowRequest holds the parameters of FlexibleLoanBorrow; one of LoanAmount and CollateralAmount is
// required, and giving both sets a custom LTV
type FlexibleLoanBorrowRequest struct {
	LoanCoin         currency.Code
	LoanAmount       float64
	CollateralCoin   currency.Code
	CollateralAmount float64
}

// FlexibleLoanBorrowResponse holds a flexible loan borrow
type FlexibleLoanBorrowResponse struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	LoanAmount       types.Number  `json:"loanAmount"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralAmount types.Number  `json:"collateralAmount"`
	Status           string        `json:"status"`
}

// FlexibleLoanRepayRequest holds the parameters of FlexibleLoanRepay
type FlexibleLoanRepayRequest struct {
	LoanCoin       currency.Code
	CollateralCoin currency.Code
	RepayAmount    float64
	// CollateralReturn returns excess collateral to the spot account; when false it stays in the loan and lowers the LTV.
	// It is sent only when set, and the endpoint returns excess collateral when it is left out
	CollateralReturn *bool
	FullRepayment    bool
	RepaymentType    uint64 // 1 repays with the loan asset (the default), 2 with the collateral
}

// FlexibleLoanRepayResponse holds the outcome of a flexible loan repayment
type FlexibleLoanRepayResponse struct {
	LoanCoin            currency.Code `json:"loanCoin"`
	CollateralCoin      currency.Code `json:"collateralCoin"`
	RemainingDebt       types.Number  `json:"remainingDebt"`
	RemainingCollateral types.Number  `json:"remainingCollateral"`
	FullRepayment       bool          `json:"fullRepayment"`
	CurrentLTV          types.Number  `json:"currentLTV"`
	RepayStatus         string        `json:"repayStatus"`
}

// FlexibleLoanAssetsDataResponse holds the flexible loanable assets
type FlexibleLoanAssetsDataResponse struct {
	Rows  []FlexibleLoanAsset `json:"rows"`
	Total uint64              `json:"total"`
}

// FlexibleLoanAsset holds the interest rate and USD borrow limits of a flexible loanable asset
type FlexibleLoanAsset struct {
	LoanCoin             currency.Code `json:"loanCoin"`
	FlexibleInterestRate types.Number  `json:"flexibleInterestRate"`
	FlexibleMinimumLimit types.Number  `json:"flexibleMinLimit"`
	FlexibleMaximumLimit types.Number  `json:"flexibleMaxLimit"`
}

// FlexibleLoanHistoryRequest holds the parameters of the flexible loan borrow, liquidation, LTV adjustment and
// repayment history endpoints
type FlexibleLoanHistoryRequest struct {
	LoanCoin       currency.Code
	CollateralCoin currency.Code
	StartTime      time.Time
	EndTime        time.Time
	Current        uint64
	Limit          uint64
}

// FlexibleLoanBorrowHistoryResponse holds a page of flexible loan borrows
type FlexibleLoanBorrowHistoryResponse struct {
	Rows  []FlexibleLoanBorrowRecord `json:"rows"`
	Total uint64                     `json:"total"`
}

// FlexibleLoanBorrowRecord holds a flexible loan borrow
type FlexibleLoanBorrowRecord struct {
	LoanCoin                currency.Code `json:"loanCoin"`
	InitialLoanAmount       types.Number  `json:"initialLoanAmount"`
	CollateralCoin          currency.Code `json:"collateralCoin"`
	InitialCollateralAmount types.Number  `json:"initialCollateralAmount"`
	BorrowTime              types.Time    `json:"borrowTime"`
	Status                  string        `json:"status"`
}

// FlexibleCollateralAssetsDataResponse holds the flexible loan collateral assets
type FlexibleCollateralAssetsDataResponse struct {
	Rows  []FlexibleLoanCollateralAsset `json:"rows"`
	Total uint64                        `json:"total"`
}

// FlexibleLoanCollateralAsset holds the LTV thresholds and USD collateral limit of a flexible loan collateral asset
type FlexibleLoanCollateralAsset struct {
	CollateralCoin currency.Code `json:"collateralCoin"`
	InitialLTV     types.Number  `json:"initialLTV"`
	MarginCallLTV  types.Number  `json:"marginCallLTV"`
	LiquidationLTV types.Number  `json:"liquidationLTV"`
	MaximumLimit   types.Number  `json:"maxLimit"`
}

// FlexibleLoanLiquidationHistoryResponse holds a page of flexible loan liquidations
type FlexibleLoanLiquidationHistoryResponse struct {
	Rows  []FlexibleLoanLiquidation `json:"rows"`
	Total uint64                    `json:"total"`
}

// FlexibleLoanLiquidation holds a flexible loan liquidation
type FlexibleLoanLiquidation struct {
	LoanCoin                    currency.Code `json:"loanCoin"`
	LiquidationDebt             types.Number  `json:"liquidationDebt"`
	CollateralCoin              currency.Code `json:"collateralCoin"`
	LiquidationCollateralAmount types.Number  `json:"liquidationCollateralAmount"`
	ReturnCollateralAmount      types.Number  `json:"returnCollateralAmount"`
	LiquidationFee              types.Number  `json:"liquidationFee"`
	LiquidationStartingPrice    types.Number  `json:"liquidationStartingPrice"`
	LiquidationStartingTime     types.Time    `json:"liquidationStartingTime"`
	Status                      string        `json:"status"`
}

// FlexibleLoanLTVAdjustmentHistoryResponse holds a page of flexible loan LTV adjustments
type FlexibleLoanLTVAdjustmentHistoryResponse struct {
	Rows  []FlexibleLoanLTVAdjustment `json:"rows"`
	Total uint64                      `json:"total"`
}

// FlexibleLoanLTVAdjustment holds a flexible loan LTV adjustment
type FlexibleLoanLTVAdjustment struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	Direction        string        `json:"direction"`
	CollateralAmount types.Number  `json:"collateralAmount"`
	PreLTV           types.Number  `json:"preLTV"`
	AfterLTV         types.Number  `json:"afterLTV"`
	AdjustTime       types.Time    `json:"adjustTime"`
}

// FlexibleLoanOngoingOrdersRequest holds the parameters of FlexibleLoanOngoingOrders
type FlexibleLoanOngoingOrdersRequest struct {
	LoanCoin       currency.Code
	CollateralCoin currency.Code
	Current        uint64
	Limit          uint64
}

// FlexibleLoanOngoingOrdersResponse holds a page of ongoing flexible loans
type FlexibleLoanOngoingOrdersResponse struct {
	Rows  []FlexibleLoanOngoingOrder `json:"rows"`
	Total uint64                     `json:"total"`
}

// FlexibleLoanOngoingOrder holds an ongoing flexible loan
type FlexibleLoanOngoingOrder struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	TotalDebt        types.Number  `json:"totalDebt"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralAmount types.Number  `json:"collateralAmount"`
	CurrentLTV       types.Number  `json:"currentLTV"`
}

// FlexibleLoanRepayHistoryResponse holds a page of flexible loan repayments
type FlexibleLoanRepayHistoryResponse struct {
	Rows  []FlexibleLoanRepayment `json:"rows"`
	Total uint64                  `json:"total"`
}

// FlexibleLoanRepayment holds a flexible loan repayment
type FlexibleLoanRepayment struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	RepayAmount      types.Number  `json:"repayAmount"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralReturn types.Number  `json:"collateralReturn"`
	RepayStatus      string        `json:"repayStatus"`
	RepayTime        types.Time    `json:"repayTime"`
}

// CryptoLoanIncomeHistoryRequest holds the parameters of CryptoLoanIncomeHistory
type CryptoLoanIncomeHistoryRequest struct {
	Asset     currency.Code
	Type      string // such as borrowIn or collateralSpent; every type when empty
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CryptoLoanIncome holds a stable rate crypto loan asset movement
type CryptoLoanIncome struct {
	Asset         currency.Code `json:"asset"`
	Type          string        `json:"type"`
	Amount        types.Number  `json:"amount"`
	Timestamp     types.Time    `json:"timestamp"`
	TransactionID string        `json:"tranId"`
}

// CryptoLoanHistoryRequest holds the parameters of the stable rate crypto loan borrow, LTV adjustment and repayment
// history endpoints
type CryptoLoanHistoryRequest struct {
	OrderID        uint64
	LoanCoin       currency.Code
	CollateralCoin currency.Code
	StartTime      time.Time
	EndTime        time.Time
	Current        uint64
	Limit          uint64
}

// CryptoLoanBorrowHistoryResponse holds a page of stable rate crypto loan borrows
type CryptoLoanBorrowHistoryResponse struct {
	Rows  []CryptoLoanBorrowRecord `json:"rows"`
	Total uint64                   `json:"total"`
}

// CryptoLoanBorrowRecord holds a stable rate crypto loan borrow
type CryptoLoanBorrowRecord struct {
	OrderID                 uint64        `json:"orderId"`
	LoanCoin                currency.Code `json:"loanCoin"`
	InitialLoanAmount       types.Number  `json:"initialLoanAmount"`
	HourlyInterestRate      types.Number  `json:"hourlyInterestRate"`
	LoanTerm                types.Number  `json:"loanTerm"` // days
	CollateralCoin          currency.Code `json:"collateralCoin"`
	InitialCollateralAmount types.Number  `json:"initialCollateralAmount"`
	BorrowTime              types.Time    `json:"borrowTime"`
	Status                  string        `json:"status"`
}

// CryptoLoanLTVAdjustmentHistoryResponse holds a page of stable rate crypto loan LTV adjustments
type CryptoLoanLTVAdjustmentHistoryResponse struct {
	Rows  []CryptoLoanLTVAdjustment `json:"rows"`
	Total uint64                    `json:"total"`
}

// CryptoLoanLTVAdjustment holds a stable rate crypto loan LTV adjustment
type CryptoLoanLTVAdjustment struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	Direction      string        `json:"direction"`
	Amount         types.Number  `json:"amount"`
	PreLTV         types.Number  `json:"preLTV"`
	AfterLTV       types.Number  `json:"afterLTV"`
	AdjustTime     types.Time    `json:"adjustTime"`
	OrderID        uint64        `json:"orderId"`
}

// CryptoLoanRepaymentHistoryResponse holds a page of stable rate crypto loan repayments
type CryptoLoanRepaymentHistoryResponse struct {
	Rows  []CryptoLoanRepayment `json:"rows"`
	Total uint64                `json:"total"`
}

// CryptoLoanRepayment holds a stable rate crypto loan repayment
type CryptoLoanRepayment struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	RepayAmount      types.Number  `json:"repayAmount"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralUsed   types.Number  `json:"collateralUsed"`
	CollateralReturn types.Number  `json:"collateralReturn"`
	RepayType        string        `json:"repayType"` // 1 repaid with the loan asset, 2 with collateral
	RepayStatus      string        `json:"repayStatus"`
	RepayTime        types.Time    `json:"repayTime"`
	OrderID          uint64        `json:"orderId"`
}
