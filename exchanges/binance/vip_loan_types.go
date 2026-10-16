package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// VIPLoanBorrowInterestRate holds the flexible borrow interest rate of a VIP loan coin
type VIPLoanBorrowInterestRate struct {
	Asset                      currency.Code `json:"asset"`
	FlexibleDailyInterestRate  types.Number  `json:"flexibleDailyInterestRate"`
	FlexibleYearlyInterestRate types.Number  `json:"flexibleYearlyInterestRate"`
	Time                       types.Time    `json:"time"`
}

// VIPCollateralAssetDataResponse holds the collateral ratio tiers of VIP loan collateral assets
type VIPCollateralAssetDataResponse struct {
	Rows  []VIPCollateralAsset `json:"rows"`
	Total uint64               `json:"total"`
}

// VIPCollateralAsset holds the collateral ratio of each value tier of a collateral asset. Ratios are percentages such
// as "80%" and ranges are USD value bands such as "10000000-100000000" or ">10000000000"
type VIPCollateralAsset struct {
	CollateralCoin        currency.Code `json:"collateralCoin"`
	FirstCollateralRatio  string        `json:"_1stCollateralRatio"`
	FirstCollateralRange  string        `json:"_1stCollateralRange"`
	SecondCollateralRatio string        `json:"_2ndCollateralRatio"`
	SecondCollateralRange string        `json:"_2ndCollateralRange"`
	ThirdCollateralRatio  string        `json:"_3rdCollateralRatio"`
	ThirdCollateralRange  string        `json:"_3rdCollateralRange"`
	FourthCollateralRatio string        `json:"_4thCollateralRatio"`
	FourthCollateralRange string        `json:"_4thCollateralRange"`
}

// VIPLoanableAssetsDataResponse holds the interest rates and borrow limits of VIP loanable assets
type VIPLoanableAssetsDataResponse struct {
	Rows  []VIPLoanableAsset `json:"rows"`
	Total uint64             `json:"total"`
}

// VIPLoanableAsset holds the interest rates and USD borrow limits of a VIP loanable asset
type VIPLoanableAsset struct {
	LoanCoin                    currency.Code `json:"loanCoin"`
	FlexibleDailyInterestRate   types.Number  `json:"_flexibleDailyInterestRate"`
	FlexibleYearlyInterestRate  types.Number  `json:"_flexibleYearlyInterestRate"`
	ThirtyDayDailyInterestRate  types.Number  `json:"_30dDailyInterestRate"`
	ThirtyDayYearlyInterestRate types.Number  `json:"_30dYearlyInterestRate"`
	SixtyDayDailyInterestRate   types.Number  `json:"_60dDailyInterestRate"`
	SixtyDayYearlyInterestRate  types.Number  `json:"_60dYearlyInterestRate"`
	MinimumLimit                types.Number  `json:"minLimit"`
	MaximumLimit                types.Number  `json:"maxLimit"`
	VIPLevel                    uint64        `json:"vipLevel"`
}

// VIPLoanInterestRateHistoryRequest holds the parameters of GetVIPLoanInterestRateHistory
type VIPLoanInterestRateHistoryRequest struct {
	Coin      currency.Code
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Limit     uint64
}

// VIPLoanInterestRateHistoryResponse holds a page of VIP loan flexible interest rates
type VIPLoanInterestRateHistoryResponse struct {
	Rows  []VIPLoanInterestRate `json:"rows"`
	Total uint64                `json:"total"`
}

// VIPLoanInterestRate holds a VIP loan coin's annualised flexible interest rate at a point in time
type VIPLoanInterestRate struct {
	Coin                   currency.Code `json:"coin"`
	AnnualizedInterestRate types.Number  `json:"annualizedInterestRate"`
	Time                   types.Time    `json:"time"`
}

// VIPLoanBorrowRequest holds the parameters of VIPLoanBorrow
type VIPLoanBorrowRequest struct {
	LoanAccountID        uint64 // the account receiving the loan
	LoanCoin             currency.Code
	LoanAmount           float64
	CollateralAccountIDs []uint64
	CollateralCoins      []currency.Code
	IsFlexibleRate       bool
	LoanTerm             uint64 // days; required at a fixed rate
}

// VIPLoanBorrowResponse holds the application a VIP loan borrow made
type VIPLoanBorrowResponse struct {
	LoanAccountID       string        `json:"loanAccountId"`
	RequestID           string        `json:"requestId"`
	LoanCoin            currency.Code `json:"loanCoin"`
	IsFlexibleRate      string        `json:"isFlexibleRate"` // Yes or No
	LoanAmount          types.Number  `json:"loanAmount"`
	CollateralAccountID string        `json:"collateralAccountId"` // comma separated
	CollateralCoin      string        `json:"collateralCoin"`      // comma separated
	LoanTerm            string        `json:"loanTerm"`            // days
}

// VIPLoanRenewResponse holds a renewed VIP loan
type VIPLoanRenewResponse struct {
	LoanAccountID       string        `json:"loanAccountId"`
	LoanCoin            currency.Code `json:"loanCoin"`
	LoanAmount          types.Number  `json:"loanAmount"`
	CollateralAccountID string        `json:"collateralAccountId"` // comma separated
	CollateralCoin      string        `json:"collateralCoin"`      // comma separated
	LoanTerm            string        `json:"loanTerm"`            // days
}

// VIPLoanRepayResponse holds the outcome of a VIP loan repayment
type VIPLoanRepayResponse struct {
	LoanCoin           currency.Code `json:"loanCoin"`
	RepayAmount        types.Number  `json:"repayAmount"`
	RemainingPrincipal types.Number  `json:"remainingPrincipal"`
	RemainingInterest  types.Number  `json:"remainingInterest"`
	CollateralCoin     string        `json:"collateralCoin"` // comma separated
	CurrentLTV         types.Number  `json:"currentLTV"`
	RepayStatus        string        `json:"repayStatus"`
}

// VIPLoanCollateralAccountResponse holds VIP loan collateral accounts
type VIPLoanCollateralAccountResponse struct {
	Rows  []VIPLoanCollateralAccount `json:"rows"`
	Total uint64                     `json:"total"`
}

// VIPLoanCollateralAccount holds a VIP loan collateral account and its collateral coins
type VIPLoanCollateralAccount struct {
	CollateralAccountID string `json:"collateralAccountId"`
	CollateralCoin      string `json:"collateralCoin"` // comma separated
}

// VIPLoanHistoryRequest holds the parameters of GetVIPLoanAccruedInterest and GetVIPLoanRepaymentHistory
type VIPLoanHistoryRequest struct {
	OrderID   uint64
	LoanCoin  currency.Code
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Limit     uint64
}

// VIPLoanAccruedInterestResponse holds a page of VIP loan interest records
type VIPLoanAccruedInterestResponse struct {
	Rows  []VIPLoanAccruedInterest `json:"rows"`
	Total uint64                   `json:"total"`
}

// VIPLoanAccruedInterest holds an interest accrual on a VIP loan
type VIPLoanAccruedInterest struct {
	LoanCoin           currency.Code `json:"loanCoin"`
	PrincipalAmount    types.Number  `json:"principalAmount"`
	InterestAmount     types.Number  `json:"interestAmount"`
	AnnualInterestRate types.Number  `json:"annualInterestRate"`
	AccrualTime        types.Time    `json:"accrualTime"`
	OrderID            uint64        `json:"orderId"` // latest order ID of a renewed loan
}

// VIPLoanOngoingOrdersRequest holds the parameters of GetVIPLoanOngoingOrders
type VIPLoanOngoingOrdersRequest struct {
	OrderID             uint64
	CollateralAccountID uint64
	LoanCoin            currency.Code
	CollateralCoin      currency.Code
	Current             uint64
	Limit               uint64
}

// VIPLoanOngoingOrdersResponse holds a page of ongoing VIP loan orders
type VIPLoanOngoingOrdersResponse struct {
	Rows  []VIPLoanOngoingOrder `json:"rows"`
	Total uint64                `json:"total"`
}

// VIPLoanOngoingOrder holds an ongoing VIP loan order
type VIPLoanOngoingOrder struct {
	OrderID                          uint64        `json:"orderId"`
	LoanCoin                         currency.Code `json:"loanCoin"`
	TotalDebt                        types.Number  `json:"totalDebt"`
	LoanRate                         types.Number  `json:"loanRate"`
	ResidualInterest                 types.Number  `json:"residualInterest"`
	CollateralAccountID              string        `json:"collateralAccountId"` // comma separated
	CollateralCoin                   string        `json:"collateralCoin"`      // comma separated
	TotalCollateralValueAfterHaircut types.Number  `json:"totalCollateralValueAfterHaircut"`
	LockedCollateralValue            types.Number  `json:"lockedCollateralValue"`
	CurrentLTV                       types.Number  `json:"currentLTV"`
	ExpirationTime                   types.Time    `json:"expirationTime"` // zero for a flexible rate loan
	LoanDate                         types.Time    `json:"loanDate"`
	LoanTerm                         string        `json:"loanTerm"` // such as "30days", or "open term" for a flexible rate loan
}

// VIPLoanRepaymentHistoryResponse holds a page of VIP loan repayments
type VIPLoanRepaymentHistoryResponse struct {
	Rows  []VIPLoanRepayment `json:"rows"`
	Total uint64             `json:"total"`
}

// VIPLoanRepayment holds a VIP loan repayment
type VIPLoanRepayment struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	RepayAmount    types.Number  `json:"repayAmount"`
	CollateralCoin string        `json:"collateralCoin"` // comma separated
	RepayStatus    string        `json:"repayStatus"`
	LoanDate       types.Time    `json:"loanDate"`
	RepayTime      types.Time    `json:"repayTime"`
	OrderID        string        `json:"orderId"`
}

// VIPLoanApplicationStatusResponse holds a page of VIP loan applications
type VIPLoanApplicationStatusResponse struct {
	Rows  []VIPLoanApplication `json:"rows"`
	Total uint64               `json:"total"`
}

// VIPLoanApplication holds a VIP loan application and its status
type VIPLoanApplication struct {
	LoanAccountID       string        `json:"loanAccountId"`
	OrderID             string        `json:"orderId"`
	RequestID           string        `json:"requestId"`
	LoanCoin            currency.Code `json:"loanCoin"`
	LoanAmount          types.Number  `json:"loanAmount"`
	CollateralAccountID string        `json:"collateralAccountId"` // comma separated
	CollateralCoin      string        `json:"collateralCoin"`      // comma separated
	LoanTerm            string        `json:"loanTerm"`            // days
	Status              string        `json:"status"`
	LoanDate            types.Time    `json:"loanDate"`
}
