package bybit

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CryptoLoanBorrowableCoinsResponse holds the crypto loan borrowable coins
type CryptoLoanBorrowableCoinsResponse struct {
	List []CryptoLoanBorrowableCoin `json:"list"`
}

// CryptoLoanBorrowableCoin holds a crypto loan borrowable coin's loan types, precisions, limits and annualised
// interest rates. A fixed term rate is the lowest the market offers now, and is 0 when nobody lends for that term
type CryptoLoanBorrowableCoin struct {
	Currency                       currency.Code `json:"currency"`
	FixedBorrowable                bool          `json:"fixedBorrowable"`
	FixedBorrowingAccuracy         uint64        `json:"fixedBorrowingAccuracy"` // Decimal places
	FlexibleBorrowable             bool          `json:"flexibleBorrowable"`
	FlexibleBorrowingAccuracy      uint64        `json:"flexibleBorrowingAccuracy"` // Decimal places
	MaxBorrowingAmount             types.Number  `json:"maxBorrowingAmount"`
	MinFixedBorrowingAmount        types.Number  `json:"minFixedBorrowingAmount"`
	MinFlexibleBorrowingAmount     types.Number  `json:"minFlexibleBorrowingAmount"`
	VIPLevel                       string        `json:"vipLevel"`
	FlexibleAnnualizedInterestRate types.Number  `json:"flexibleAnnualizedInterestRate"`
	AnnualizedInterestRate7Day     types.Number  `json:"annualizedInterestRate7D"`
	AnnualizedInterestRate14Day    types.Number  `json:"annualizedInterestRate14D"`
	AnnualizedInterestRate30Day    types.Number  `json:"annualizedInterestRate30D"`
	AnnualizedInterestRate60Day    types.Number  `json:"annualizedInterestRate60D"`
	AnnualizedInterestRate90Day    types.Number  `json:"annualizedInterestRate90D"`
	AnnualizedInterestRate180Day   types.Number  `json:"annualizedInterestRate180D"`
}

// CryptoLoanCollateralCoinsResponse holds the crypto loan collateral ratio tiers and liquidation order of collateral
// coins
type CryptoLoanCollateralCoinsResponse struct {
	CollateralRatioConfigList []CryptoLoanCollateralRatioConfig `json:"collateralRatioConfigList"`
	CurrencyLiquidationList   []CryptoLoanCurrencyLiquidation   `json:"currencyLiquidationList"`
}

// CryptoLoanCollateralRatioConfig holds the collateral ratio tiers shared by a group of coins
type CryptoLoanCollateralRatioConfig struct {
	CollateralRatioList []CryptoLoanCollateralRatio `json:"collateralRatioList"`
	Currencies          CryptoLoanCurrencies        `json:"currencies"`
}

// CryptoLoanCollateralRatio holds the collateral ratio of a collateral quantity tier
type CryptoLoanCollateralRatio struct {
	CollateralRatio types.Number `json:"collateralRatio"`
	MaxValue        types.Number `json:"maxValue"`
	MinValue        types.Number `json:"minValue"`
}

// CryptoLoanCurrencies holds coins Bybit sends as one comma separated string, such as "BTC,ETH,XRP"
type CryptoLoanCurrencies []currency.Code

// UnmarshalJSON decodes a comma separated string of coins
func (c *CryptoLoanCurrencies) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*c = nil
		return nil
	}
	parts := strings.Split(s, ",")
	codes := make(CryptoLoanCurrencies, len(parts))
	for i := range parts {
		codes[i] = currency.NewCode(strings.TrimSpace(parts[i]))
	}
	*c = codes
	return nil
}

// CryptoLoanCurrencyLiquidation holds the order in which a collateral coin is liquidated
type CryptoLoanCurrencyLiquidation struct {
	Currency         currency.Code `json:"currency"`
	LiquidationOrder uint64        `json:"liquidationOrder"`
}

// CryptoLoanMaxCollateralReductionAmountResponse holds the maximum amount by which crypto loan collateral can be
// reduced
type CryptoLoanMaxCollateralReductionAmountResponse struct {
	MaxCollateralAmount types.Number `json:"maxCollateralAmount"`
}

// AdjustCryptoLoanCollateralAmountRequest holds the parameters of AdjustCryptoLoanCollateralAmount
type AdjustCryptoLoanCollateralAmountRequest struct {
	Currency  currency.Code
	Amount    float64
	Direction string // 0: add collateral, 1: reduce collateral
}

// AdjustCryptoLoanCollateralAmountResponse holds the ID of a crypto loan collateral adjustment
type AdjustCryptoLoanCollateralAmountResponse struct {
	AdjustID uint64 `json:"adjustId"`
}

// CryptoLoanCollateralAdjustmentHistoryRequest holds the filters of GetCryptoLoanCollateralAdjustmentHistory; every
// field is optional
type CryptoLoanCollateralAdjustmentHistoryRequest struct {
	AdjustID           string
	CollateralCurrency currency.Code
	Limit              uint64
	Cursor             string
}

// CryptoLoanCollateralAdjustmentHistoryResponse holds a page of crypto loan collateral adjustments
type CryptoLoanCollateralAdjustmentHistoryResponse struct {
	List           []CryptoLoanCollateralAdjustment `json:"list"`
	NextPageCursor string                           `json:"nextPageCursor"`
}

// CryptoLoanCollateralAdjustment holds a crypto loan collateral adjustment and the LTV before and after it
type CryptoLoanCollateralAdjustment struct {
	CollateralCurrency currency.Code `json:"collateralCurrency"`
	Amount             types.Number  `json:"amount"`
	AdjustID           uint64        `json:"adjustId"`
	AdjustTime         types.Time    `json:"adjustTime"`
	PreLTV             types.Number  `json:"preLTV"`
	AfterLTV           types.Number  `json:"afterLTV"`
	Direction          uint64        `json:"direction"` // 0: add collateral, 1: reduce collateral
	Status             uint64        `json:"status"`    // 1: success, 2: processing, 3: fail
}

// CryptoLoanPositionResponse holds the account's crypto loan debts, collateral and supplies
type CryptoLoanPositionResponse struct {
	BorrowList      []CryptoLoanBorrowPosition `json:"borrowList"`
	CollateralList  []CryptoLoanCoinAmount     `json:"collateralList"` // AmountUSD after tiered collateral ratios
	LTV             types.Number               `json:"ltv"`
	SupplyList      []CryptoLoanCoinAmount     `json:"supplyList"`
	TotalCollateral types.Number               `json:"totalCollateral"` // In USD
	TotalDebt       types.Number               `json:"totalDebt"`       // Fixed and flexible debt in USD
	TotalSupply     types.Number               `json:"totalSupply"`     // In USD
	// CollateralRestriction is the platform collateral restriction status: -1 unknown, 0 not restricted, 1 not
	// restricted but near the platform limit, 2 restricted, which rejects adding collateral
	CollateralRestriction string `json:"colRes"`
}

// CryptoLoanBorrowPosition holds the crypto loan debt of a loan coin
type CryptoLoanBorrowPosition struct {
	FixedTotalDebt             types.Number  `json:"fixedTotalDebt"`
	FixedTotalDebtUSD          types.Number  `json:"fixedTotalDebtUSD"`
	FlexibleHourlyInterestRate types.Number  `json:"flexibleHourlyInterestRate"`
	FlexibleTotalDebt          types.Number  `json:"flexibleTotalDebt"`
	FlexibleTotalDebtUSD       types.Number  `json:"flexibleTotalDebtUSD"`
	LoanCurrency               currency.Code `json:"loanCurrency"`
}

// CryptoLoanCoinAmount holds an amount of a coin and its USD value
type CryptoLoanCoinAmount struct {
	Amount    types.Number  `json:"amount"`
	AmountUSD types.Number  `json:"amountUSD"`
	Currency  currency.Code `json:"currency"`
}

// CryptoLoanCollateral holds a coin and amount to put up as crypto loan collateral
type CryptoLoanCollateral struct {
	Currency currency.Code
	Amount   float64
}

// CryptoLoanMaxLoanAmountResponse holds the maximum amount of a coin that can be borrowed against the current and any
// added collateral
type CryptoLoanMaxLoanAmountResponse struct {
	Currency       currency.Code `json:"currency"`
	MaxLoan        types.Number  `json:"maxLoan"`
	NotionalUSD    types.Number  `json:"notionalUsd"`
	RemainingQuota types.Number  `json:"remainingQuota"` // Platform limit left, shared by main and sub accounts
}

// FlexibleLoanAvailableInventoryResponse holds a coin's flexible loan pool inventory
type FlexibleLoanAvailableInventoryResponse struct {
	Currency           currency.Code `json:"currency"`
	AvailableInventory types.Number  `json:"availableInventory"`
	UpdateTime         types.Time    `json:"updateTime"`
}

// BorrowFlexibleLoanRequest holds the parameters of BorrowFlexibleLoan
type BorrowFlexibleLoanRequest struct {
	LoanCurrency   currency.Code
	LoanAmount     float64
	CollateralList []CryptoLoanCollateral // Up to 100 coins, taken from the funding wallet
}

// CryptoLoanOrderResponse holds the ID of a crypto loan borrow or supply order
type CryptoLoanOrderResponse struct {
	OrderID string `json:"orderId"`
}

// CryptoLoanRepayResponse holds the ID of a crypto loan repayment
type CryptoLoanRepayResponse struct {
	RepayID string `json:"repayId"`
}

// RepayFlexibleLoanWithCollateralRequest holds the parameters of RepayFlexibleLoanWithCollateral
type RepayFlexibleLoanWithCollateralRequest struct {
	LoanCurrency    currency.Code
	CollateralCoins []currency.Code
	Amount          float64
}

// FlexibleLoansResponse holds the account's ongoing flexible loans
type FlexibleLoansResponse struct {
	List []FlexibleLoan `json:"list"`
}

// FlexibleLoan holds an ongoing flexible loan's debt and latest hourly interest rate
type FlexibleLoan struct {
	HourlyInterestRate types.Number  `json:"hourlyInterestRate"`
	LoanCurrency       currency.Code `json:"loanCurrency"`
	TotalDebt          types.Number  `json:"totalDebt"` // Unpaid principal and interest
	UnpaidAmount       types.Number  `json:"unpaidAmount"`
	UnpaidInterest     types.Number  `json:"unpaidInterest"`
}

// FlexibleLoanBorrowingHistoryRequest holds the filters of GetFlexibleLoanBorrowingHistory; every field is optional
type FlexibleLoanBorrowingHistoryRequest struct {
	OrderID      string
	LoanCurrency currency.Code
	Limit        uint64
	Cursor       string
}

// FlexibleLoanBorrowingHistoryResponse holds a page of flexible loan borrows
type FlexibleLoanBorrowingHistoryResponse struct {
	List           []FlexibleLoanBorrow `json:"list"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// FlexibleLoanBorrow holds a flexible loan borrow
type FlexibleLoanBorrow struct {
	BorrowTime        types.Time    `json:"borrowTime"`
	InitialLoanAmount types.Number  `json:"initialLoanAmount"`
	LoanCurrency      currency.Code `json:"loanCurrency"`
	OrderID           string        `json:"orderId"`
	Status            uint64        `json:"status"` // 1: success, 2: processing, 3: fail
}

// FlexibleLoanRepaymentHistoryRequest holds the filters of GetFlexibleLoanRepaymentHistory; every field is optional
type FlexibleLoanRepaymentHistoryRequest struct {
	RepayID      string
	LoanCurrency currency.Code
	Limit        uint64
	Cursor       string
}

// FlexibleLoanRepaymentHistoryResponse holds a page of flexible loan repayments
type FlexibleLoanRepaymentHistoryResponse struct {
	List           []FlexibleLoanRepayment `json:"list"`
	NextPageCursor string                  `json:"nextPageCursor"`
}

// FlexibleLoanRepayment holds a flexible loan repayment
type FlexibleLoanRepayment struct {
	LoanCurrency currency.Code `json:"loanCurrency"`
	RepayAmount  types.Number  `json:"repayAmount"`
	RepayID      string        `json:"repayId"`
	RepayStatus  uint64        `json:"repayStatus"` // 1: success, 2: processing, 3: fail
	RepayTime    types.Time    `json:"repayTime"`
	// RepayType is 1 by the user, 2 by liquidation, 5 by delisting, 6 by delay liquidation or 7 by currency
	RepayType uint64 `json:"repayType"`
}

// FixedLoanMarketRequest holds the parameters of GetFixedLoanLendingMarket and GetFixedLoanBorrowingMarket
type FixedLoanMarketRequest struct {
	OrderCurrency currency.Code
	Term          uint64 // Days; optional
	OrderBy       string // apy, term or quantity
	Sort          uint64 // 0: ascending, the default; 1: descending
	Limit         uint64
}

// FixedLoanMarketResponse holds the fixed term crypto loan orders open in the market
type FixedLoanMarketResponse struct {
	List []FixedLoanMarketQuote `json:"list"`
}

// FixedLoanMarketQuote holds a fixed term crypto loan order open in the market
type FixedLoanMarketQuote struct {
	OrderCurrency currency.Code `json:"orderCurrency"`
	Term          uint64        `json:"term"` // Days
	AnnualRate    types.Number  `json:"annualRate"`
	Quantity      types.Number  `json:"qty"`
}

// FixedLoanAvailableInventoryRequest holds the parameters of GetFixedLoanAvailableInventory
type FixedLoanAvailableInventoryRequest struct {
	Currency   currency.Code
	Term       uint64  // Days
	AnnualRate float64 // 0.02 is 2%
}

// FixedLoanAvailableInventoryResponse holds a coin's fixed term loan pool inventory at a term and annual rate
type FixedLoanAvailableInventoryResponse struct {
	Currency           currency.Code `json:"currency"`
	Term               types.Number  `json:"term"` // Days
	AnnualRate         types.Number  `json:"annualRate"`
	AvailableInventory types.Number  `json:"availableInventory"`
	UpdateTime         types.Time    `json:"updateTime"`
}

// CreateFixedLoanBorrowOrderRequest holds the parameters of CreateFixedLoanBorrowOrder. The deprecated autoRepay
// parameter is left out in favour of RepayType
type CreateFixedLoanBorrowOrderRequest struct {
	OrderCurrency  currency.Code
	OrderAmount    float64
	AnnualRate     float64 // 0.02 is 2%
	Term           uint64  // Days
	RepayType      string  // 1: auto repayment, the default; 2: transfer to a flexible loan
	StrategyType   string  // PARTIAL: allow partial fills, the default; FULL: fill fully only
	CollateralList []CryptoLoanCollateral
}

// RenewFixedLoanBorrowOrderResponse holds the ID of a renewed fixed term borrow order
type RenewFixedLoanBorrowOrderResponse struct {
	OrderID uint64 `json:"orderId"`
}

// CreateFixedLoanSupplyOrderRequest holds the parameters of CreateFixedLoanSupplyOrder
type CreateFixedLoanSupplyOrderRequest struct {
	OrderCurrency   currency.Code
	OrderAmount     float64
	AnnualRate      float64 // 0.02 is 2%
	Term            uint64  // Days
	AvailableSource string  // 0: funding account, the default; 1: earn flexible account; 2: both
}

// FixedLoanBorrowContractInfoRequest holds the filters of GetFixedLoanBorrowContractInfo; every field is optional
type FixedLoanBorrowContractInfoRequest struct {
	OrderID       string
	LoanID        string
	OrderCurrency currency.Code
	Term          uint64 // Days
	Limit         uint64
	Cursor        string
}

// FixedLoanBorrowContractInfoResponse holds a page of fixed term borrow contracts
type FixedLoanBorrowContractInfoResponse struct {
	List           []FixedLoanBorrowContract `json:"list"`
	NextPageCursor string                    `json:"nextPageCursor"`
}

// FixedLoanBorrowContract holds a fixed term borrow contract
type FixedLoanBorrowContract struct {
	AnnualRate              types.Number  `json:"annualRate"`
	AutoRepay               types.Boolean `json:"autoRepay"` // Deprecated by Bybit in favour of RepayType
	BorrowCurrency          currency.Code `json:"borrowCurrency"`
	BorrowTime              types.Time    `json:"borrowTime"`
	InterestPaid            types.Number  `json:"interestPaid"`
	LoanID                  string        `json:"loanId"`
	OrderID                 string        `json:"orderId"`
	RepaymentTime           types.Time    `json:"repaymentTime"`
	ResidualPenaltyInterest types.Number  `json:"residualPenaltyInterest"`
	ResidualPrincipal       types.Number  `json:"residualPrincipal"`
	Status                  uint64        `json:"status"` // 1: unrepaid, 2: fully repaid, 3: overdue
	Term                    types.Number  `json:"term"`   // Days
	// RepayType is 1 auto repayment, 2 transfer to a flexible loan, or 0 no auto repayment for older orders
	RepayType string `json:"repayType"`
}

// FixedLoanSupplyContractInfoRequest holds the filters of GetFixedLoanSupplyContractInfo; every field is optional
type FixedLoanSupplyContractInfoRequest struct {
	OrderID        string
	SupplyID       string
	SupplyCurrency currency.Code
	Term           uint64 // Days
	Limit          uint64
	Cursor         string
}

// FixedLoanSupplyContractInfoResponse holds a page of fixed term supply contracts
type FixedLoanSupplyContractInfoResponse struct {
	List           []FixedLoanSupplyContract `json:"list"`
	NextPageCursor string                    `json:"nextPageCursor"`
}

// FixedLoanSupplyContract holds a fixed term supply contract
type FixedLoanSupplyContract struct {
	AnnualRate           types.Number  `json:"annualRate"`
	SupplyCurrency       currency.Code `json:"supplyCurrency"`
	SupplyTime           types.Time    `json:"supplyTime"`
	SupplyAmount         types.Number  `json:"supplyAmount"`
	InterestPaid         types.Number  `json:"interestPaid"`
	SupplyID             string        `json:"supplyId"`
	OrderID              string        `json:"orderId"`
	RedemptionTime       types.Time    `json:"redemptionTime"`  // Planned
	PenaltyInterest      types.Number  `json:"penaltyInterest"` // Overdue interest
	ActualRedemptionTime types.Time    `json:"actualRedemptionTime"`
	Status               uint64        `json:"status"` // 1: supplying, 2: redeemed
	Term                 types.Number  `json:"term"`   // Days
}

// UnmarshalJSON decodes a fixed term supply contract. The docs table names the paid interest interestPaid but the
// docs' response sample sends it as interest, so interest is read when interestPaid is absent
func (f *FixedLoanSupplyContract) UnmarshalJSON(data []byte) error {
	type alias FixedLoanSupplyContract
	type sampled struct {
		*alias
		Interest types.Number `json:"interest"`
	}
	aux := sampled{alias: (*alias)(f)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if f.InterestPaid == 0 {
		f.InterestPaid = aux.Interest
	}
	return nil
}

// FixedLoanOrderInfoRequest holds the filters of GetFixedLoanBorrowOrderInfo and GetFixedLoanSupplyOrderInfo; every
// field is optional
type FixedLoanOrderInfoRequest struct {
	OrderID       string
	OrderCurrency currency.Code
	State         string // 1: matching, 2: partially filled and cancelled, 3: fully filled, 4: cancelled
	Term          uint64 // Days
	Limit         uint64
	Cursor        string
}

// FixedLoanBorrowOrderInfoResponse holds a page of fixed term borrow orders
type FixedLoanBorrowOrderInfoResponse struct {
	List           []FixedLoanBorrowOrder `json:"list"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

// FixedLoanBorrowOrder holds a fixed term borrow order
type FixedLoanBorrowOrder struct {
	AnnualRate     types.Number  `json:"annualRate"`
	OrderID        uint64        `json:"orderId"`
	OrderTime      types.Time    `json:"orderTime"`
	FilledQuantity types.Number  `json:"filledQty"`
	OrderQuantity  types.Number  `json:"orderQty"`
	OrderCurrency  currency.Code `json:"orderCurrency"`
	// State is 1 matching, 2 partially filled and cancelled, 3 fully filled, 4 cancelled or 5 failed
	State uint64 `json:"state"`
	Term  uint64 `json:"term"` // Days
	// RepayType is 1 auto repayment, 2 transfer to a flexible loan, or 0 no auto repayment for older orders
	RepayType    string `json:"repayType"`
	StrategyType string `json:"strategyType"` // PARTIAL: partial fills allowed, FULL: full fill only
}

// FixedLoanRenewOrderInfoRequest holds the filters of GetFixedLoanRenewOrderInfo; every field is optional
type FixedLoanRenewOrderInfoRequest struct {
	OrderID       string
	OrderCurrency currency.Code
	Limit         uint64
	Cursor        string
}

// FixedLoanRenewOrderInfoResponse holds a page of fixed term borrow renewals
type FixedLoanRenewOrderInfoResponse struct {
	List           []FixedLoanRenewOrder `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// FixedLoanRenewOrder holds a fixed term borrow renewal
type FixedLoanRenewOrder struct {
	BorrowCurrency currency.Code `json:"borrowCurrency"`
	Amount         types.Number  `json:"amount"`
	// AutoRepay is the repayment type: 1 auto repayment, 2 transfer to a flexible loan, or 0 no auto repayment for
	// older orders
	AutoRepay       uint64     `json:"autoRepay"`
	ContractNumber  string     `json:"contractNo"`
	DueTime         types.Time `json:"dueTime"`
	OrderID         uint64     `json:"orderId"`
	LoanID          string     `json:"loanId"`
	RenewLoanNumber string     `json:"renewLoanNo"`
	Time            types.Time `json:"time"`
}

// FixedLoanSupplyOrderInfoResponse holds a page of fixed term supply orders
type FixedLoanSupplyOrderInfoResponse struct {
	List           []FixedLoanSupplyOrder `json:"list"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

// FixedLoanSupplyOrder holds a fixed term supply order
type FixedLoanSupplyOrder struct {
	AnnualRate     types.Number  `json:"annualRate"`
	OrderID        uint64        `json:"orderId"`
	OrderTime      types.Time    `json:"orderTime"`
	FilledQuantity types.Number  `json:"filledQty"`
	OrderQuantity  types.Number  `json:"orderQty"`
	OrderCurrency  currency.Code `json:"orderCurrency"`
	// State is 1 matching, 2 partially filled and cancelled, 3 fully filled, 4 cancelled or 5 failed
	State uint64 `json:"state"`
	Term  uint64 `json:"term"` // Days
}

// RepayFixedLoanWithCollateralRequest holds the parameters of RepayFixedLoanWithCollateral
type RepayFixedLoanWithCollateralRequest struct {
	LoanID          string // Optional; without it the loan currency's contracts are repaid in Bybit's offset order
	LoanCurrency    currency.Code
	CollateralCoins []currency.Code
	Amount          float64
}

// FixedLoanRepaymentHistoryRequest holds the filters of GetFixedLoanRepaymentHistory; every field is optional
type FixedLoanRepaymentHistoryRequest struct {
	RepayID      string
	LoanCurrency currency.Code
	Limit        uint64
	Cursor       string
}

// FixedLoanRepaymentHistoryResponse holds a page of fixed term loan repayments
type FixedLoanRepaymentHistoryResponse struct {
	List           []FixedLoanRepayment `json:"list"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// FixedLoanRepayment holds a fixed term loan repayment, which may cover several loan contracts
type FixedLoanRepayment struct {
	Details      []FixedLoanRepaymentDetail `json:"details"`
	LoanCurrency currency.Code              `json:"loanCurrency"`
	RepayAmount  types.Number               `json:"repayAmount"`
	RepayID      string                     `json:"repayId"`
	RepayStatus  uint64                     `json:"repayStatus"` // 1: success, 2: processing, 3: fail
	RepayTime    types.Time                 `json:"repayTime"`
	// RepayType is 1 by the user, 2 by liquidation, 3 auto repayment, 4 overdue repayment, 5 by delisting, 6 by
	// delay liquidation, 7 by currency or 8 transfer to a flexible loan
	RepayType uint64 `json:"repayType"`
}

// FixedLoanRepaymentDetail holds the part of a fixed term loan repayment applied to one loan contract
type FixedLoanRepaymentDetail struct {
	LoanCurrency currency.Code `json:"loanCurrency"`
	RepayAmount  types.Number  `json:"repayAmount"`
	LoanID       string        `json:"loanId"`
}
