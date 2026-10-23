package gate

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// MultiCollateralLoanOrderRequest holds a multi-collateral loan order request
type MultiCollateralLoanOrderRequest struct {
	OrderID              string                      `json:"order_id,omitempty"`
	OrderType            string                      `json:"order_type,omitempty"` // current or fixed, defaulting to current
	FixedType            string                      `json:"fixed_type,omitempty"` // 7d or 30d, required for fixed rate orders
	FixedRate            types.Number                `json:"fixed_rate,omitempty"` // Required for fixed rate orders
	AutoRenew            bool                        `json:"auto_renew,omitempty"` // Fixed rate orders only
	AutoRepay            bool                        `json:"auto_repay,omitempty"` // Fixed rate orders only
	BorrowCurrency       currency.Code               `json:"borrow_currency"`
	BorrowAmount         types.Number                `json:"borrow_amount"`
	CollateralCurrencies []CurrencyAndAmountResponse `json:"collateral_currencies,omitempty"`
}

// MultiCollateralLoanOrderDetailResponse holds a multi-collateral loan order
type MultiCollateralLoanOrderDetailResponse struct {
	OrderID                 string                    `json:"order_id"`
	OrderType               string                    `json:"order_type"`
	FixedType               string                    `json:"fixed_type"`
	FixedRate               types.Number              `json:"fixed_rate"`
	ExpireTime              types.Time                `json:"expire_time"`
	AutoRenew               bool                      `json:"auto_renew"`
	AutoRepay               bool                      `json:"auto_repay"`
	CurrentLTV              types.Number              `json:"current_ltv"`
	Status                  string                    `json:"status"`
	BorrowTime              types.Time                `json:"borrow_time"`
	TotalLeftRepayUSDT      types.Number              `json:"total_left_repay_usdt"`
	TotalLeftCollateralUSDT types.Number              `json:"total_left_collateral_usdt"`
	BorrowCurrencies        []*BorrowCurrencyInfo     `json:"borrow_currencies"`
	CollateralCurrencies    []*CollateralCurrencyInfo `json:"collateral_currencies"`
}

// BorrowCurrencyInfo holds a multi-collateral loan order's outstanding borrowed currency
type BorrowCurrencyInfo struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	LeftRepayPrincipal types.Number  `json:"left_repay_principal"`
	LeftRepayInterest  types.Number  `json:"left_repay_interest"`
	LeftRepayUSDT      types.Number  `json:"left_repay_usdt"`
}

// CollateralCurrencyInfo holds a multi-collateral loan order's remaining collateral currency
type CollateralCurrencyInfo struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	LeftCollateral     types.Number  `json:"left_collateral"`
	LeftCollateralUSDT types.Number  `json:"left_collateral_usdt"`
}

// MultiCollateralLoanRepaymentRequest holds a multi-collateral loan repayment request
type MultiCollateralLoanRepaymentRequest struct {
	OrderID    uint64                                  `json:"order_id"`
	RepayItems []LoanRepaymentMultiCollateralAssetItem `json:"repay_items"`
}

// LoanRepaymentMultiCollateralAssetItem holds a currency to repay on a multi-collateral loan
type LoanRepaymentMultiCollateralAssetItem struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount,omitempty"` // Required for partial repayments
	// RepaidAll is true for a full repayment; it is always sent as the API requires it
	RepaidAll bool `json:"repaid_all"`
}

// MultiCollateralLoanRepaymentResponse holds a multi-collateral loan repayment result
type MultiCollateralLoanRepaymentResponse struct {
	OrderID          uint64                           `json:"order_id"`
	RepaidCurrencies []*MultiCollateralRepaidCurrency `json:"repaid_currencies"`
}

// MultiCollateralRepaidCurrency holds a currency's multi-collateral loan repayment result
type MultiCollateralRepaidCurrency struct {
	Succeeded       bool          `json:"succeeded"`
	Label           string        `json:"label"`   // Error label of a failed repayment
	Message         string        `json:"message"` // Error description of a failed repayment
	Currency        currency.Code `json:"currency"`
	RepaidPrincipal types.Number  `json:"repaid_principal"`
	RepaidInterest  types.Number  `json:"repaid_interest"`
}

// MultiCollateralRepaymentRecordsRequest holds multi-collateral loan repayment record filters
type MultiCollateralRepaymentRecordsRequest struct {
	Type           string // repay or liquidate
	BorrowCurrency currency.Code
	Page           uint64
	Limit          uint64
	From           time.Time
	To             time.Time
}

// MultiCurrencyCollateralRepaymentResponse holds a multi-collateral loan repayment record
type MultiCurrencyCollateralRepaymentResponse struct {
	OrderID               uint64                       `json:"order_id"`
	RecordID              uint64                       `json:"record_id"`
	InitLTV               types.Number                 `json:"init_ltv"`
	BeforeLTV             types.Number                 `json:"before_ltv"`
	AfterLTV              types.Number                 `json:"after_ltv"`
	BorrowTime            types.Time                   `json:"borrow_time"`
	RepayTime             types.Time                   `json:"repay_time"`
	BorrowCurrencies      []*CurrencyDetail            `json:"borrow_currencies"`
	CollateralCurrencies  []*CurrencyDetail            `json:"collateral_currencies"`
	RepaidCurrencies      []*RepayRecordRepaidCurrency `json:"repaid_currencies"`
	TotalInterestList     []*RepayRecordTotalInterest  `json:"total_interest_list"`
	LeftRepayInterestList []*CurrencyDetail            `json:"left_repay_interest_list"`
}

// RepayRecordRepaidCurrency holds a currency repaid by a multi-collateral loan repayment
type RepayRecordRepaidCurrency struct {
	Currency         currency.Code `json:"currency"`
	IndexPrice       types.Number  `json:"index_price"`
	RepaidAmount     types.Number  `json:"repaid_amount"`
	RepaidPrincipal  types.Number  `json:"repaid_principal"`
	RepaidInterest   types.Number  `json:"repaid_interest"`
	RepaidAmountUSDT types.Number  `json:"repaid_amount_usdt"`
}

// RepayRecordTotalInterest holds a currency's total interest on a multi-collateral loan
type RepayRecordTotalInterest struct {
	Currency   currency.Code `json:"currency"`
	IndexPrice types.Number  `json:"index_price"`
	Amount     types.Number  `json:"amount"`
	AmountUSDT types.Number  `json:"amount_usdt"`
}

// AddOrWithdrawCollateralRequest holds a request to add or withdraw multi-collateral loan collateral
type AddOrWithdrawCollateralRequest struct {
	OrderID       uint64                      `json:"order_id"`
	OperationType string                      `json:"type"` // append to add collateral or redeem to withdraw it
	Collaterals   []CurrencyAndAmountResponse `json:"collaterals,omitempty"`
}

// CollateralAddOrRemoveResponse holds the result of adding or withdrawing multi-collateral loan collateral
type CollateralAddOrRemoveResponse struct {
	OrderID              uint64                       `json:"order_id"`
	CollateralCurrencies []*CollateralOperationResult `json:"collateral_currencies"`
}

// CollateralOperationResult holds a currency's collateral adjustment result
type CollateralOperationResult struct {
	Succeeded bool          `json:"succeeded"`
	Label     string        `json:"label"`   // Error label of a failed adjustment
	Message   string        `json:"message"` // Error description of a failed adjustment
	Currency  currency.Code `json:"currency"`
	Amount    types.Number  `json:"amount"` // 0 when the adjustment failed
}

// MultiCollateralAdjustmentRecordsRequest holds multi-collateral loan collateral adjustment record filters
type MultiCollateralAdjustmentRecordsRequest struct {
	Page               uint64
	Limit              uint64
	From               time.Time
	To                 time.Time
	CollateralCurrency currency.Code
}

// MultiCollateralAdjustmentRecordResponse holds a multi collateral adjustment records
type MultiCollateralAdjustmentRecordResponse struct {
	OrderID              uint64            `json:"order_id"`
	RecordID             uint64            `json:"record_id"`
	BeforeLTV            types.Number      `json:"before_ltv"`
	AfterLTV             types.Number      `json:"after_ltv"`
	OperateTime          types.Time        `json:"operate_time"`
	BorrowCurrencies     []*CurrencyDetail `json:"borrow_currencies"`
	CollateralCurrencies []*CurrencyDetail `json:"collateral_currencies"`
}

// MultiCollateralCurrencyQuotaResponse holds currency quota information for multi-collateral
type MultiCollateralCurrencyQuotaResponse struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	MinQuota           types.Number  `json:"min_quota"`
	LeftQuota          types.Number  `json:"left_quota"`
	LeftQuotaUSDT      types.Number  `json:"left_quote_usdt"`
	LeftQuotaFixed     types.Number  `json:"left_quota_fixed"`
	LeftQuotaUSDTFixed types.Number  `json:"left_quote_usdt_fixed"`
}

// MultiCollateralSupportedCurrenciesResponse holds borrow and collateral currencies supported by multi-collateral
type MultiCollateralSupportedCurrenciesResponse struct {
	LoanCurrencies       []*MultiLoanItem       `json:"loan_currencies"`
	CollateralCurrencies []*MultiCollateralItem `json:"collateral_currencies"`
}

// MultiLoanItem represents a borrowable currency item in multi-collateral
type MultiLoanItem struct {
	Currency currency.Code `json:"currency"`
	Price    types.Number  `json:"price"`
}

// MultiCollateralItem represents a collateral currency item in multi-collateral
type MultiCollateralItem struct {
	Currency   currency.Code `json:"currency"`
	IndexPrice types.Number  `json:"index_price"`
	Discount   types.Number  `json:"discount"`
}

// MultiCollateralisationRatioResponse holds the collateralisation ratio information for multi-collateral
type MultiCollateralisationRatioResponse struct {
	InitLTV      types.Number `json:"init_ltv"`
	AlertLTV     types.Number `json:"alert_ltv"`
	LiquidateLTV types.Number `json:"liquidate_ltv"`
}

// CollateralFixRateResponse holds a currency's 7-day and 30-day fixed interest rates
type CollateralFixRateResponse struct {
	Currency   currency.Code `json:"currency"`
	Rate7Day   types.Number  `json:"rate_7d"`
	Rate30Day  types.Number  `json:"rate_30d"`
	UpdateTime types.Time    `json:"update_time"`
}

// CollateralCurrentRateResponse holds a currency's current interest rate
type CollateralCurrentRateResponse struct {
	Currency    currency.Code `json:"currency"`
	CurrentRate types.Number  `json:"current_rate"`
}

// CurrencyDetail holds a currency detail
type CurrencyDetail struct {
	Currency         currency.Code `json:"currency"`
	IndexPrice       types.Number  `json:"index_price"`
	BeforeAmount     types.Number  `json:"before_amount"`
	BeforeAmountUSDT types.Number  `json:"before_amount_usdt"`
	AfterAmount      types.Number  `json:"after_amount"`
	AfterAmountUSDT  types.Number  `json:"after_amount_usdt"`
}
