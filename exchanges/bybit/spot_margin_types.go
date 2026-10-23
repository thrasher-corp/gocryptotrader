package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// VIPMarginDataResponse holds the spot margin borrowing terms of each VIP level, from Get VIP Margin Data
type VIPMarginDataResponse struct {
	VIPCoinList []VIPMarginLevel `json:"vipCoinList"`
}

// VIPMarginLevel holds the spot margin borrowing terms of one VIP level, from Get VIP Margin Data
type VIPMarginLevel struct {
	List     []VIPMarginCoin `json:"list"`
	VIPLevel string          `json:"vipLevel"`
}

// VIPMarginCoin holds a coin's spot margin borrowing terms at a VIP level, from Get VIP Margin Data
type VIPMarginCoin struct {
	Borrowable bool `json:"borrowable"`
	// CollateralRatio has been inaccurate since tiered collateral ratios replaced it on 19 February 2025; Get Tiered
	// Collateral Ratio returns those
	CollateralRatio    types.Number  `json:"collateralRatio"`
	Currency           currency.Code `json:"currency"`
	HourlyBorrowRate   types.Number  `json:"hourlyBorrowRate"`
	LiquidationOrder   types.Number  `json:"liquidationOrder"`
	MarginCollateral   bool          `json:"marginCollateral"`
	MaxBorrowingAmount types.Number  `json:"maxBorrowingAmount"`
}

// TieredCollateralRatioResponse holds the tiered collateral ratios of unified account loans, from Get Tiered
// Collateral Ratio
type TieredCollateralRatioResponse struct {
	List []TieredCollateralRatio `json:"list"`
}

// TieredCollateralRatio holds a coin's collateral ratio tiers, from Get Tiered Collateral Ratio
type TieredCollateralRatio struct {
	Currency            currency.Code         `json:"currency"`
	CollateralRatioList []CollateralRatioTier `json:"collateralRatioList"`
}

// CollateralRatioTier is the collateral ratio of an amount of a coin, from Get Tiered Collateral Ratio
type CollateralRatioTier struct {
	// MaxQuantity is empty, which decodes to zero, for the unbounded last tier
	MaxQuantity     types.Number `json:"maxQty"`
	MinQuantity     types.Number `json:"minQty"`
	CollateralRatio types.Number `json:"collateralRatio"`
}

// SpotMarginCurrencyDataResponse holds the manual borrowing settings of coins, from Get Currency Data
type SpotMarginCurrencyDataResponse struct {
	List []SpotMarginCurrency `json:"list"`
}

// SpotMarginCurrency holds a coin's manual borrowing settings, from Get Currency Data. A disabled kind of borrowing
// returns its settings empty
type SpotMarginCurrency struct {
	Currency                        currency.Code `json:"currency"`
	FlexibleManualBorrowable        bool          `json:"flexibleManualBorrowable"`
	MinFlexibleManualBorrowQuantity types.Number  `json:"minFlexibleManualBorrowQty"`
	// FlexibleManualBorrowAccuracy and FixedManualBorrowAccuracy are the decimal places of a borrowed quantity
	FlexibleManualBorrowAccuracy types.Number `json:"flexibleManualBorrowAccuracy"`
	FixedManualBorrowable        bool         `json:"fixedManualBorrowable"`
	MinFixedManualBorrowQuantity types.Number `json:"minFixedManualBorrowQty"`
	FixedManualBorrowAccuracy    types.Number `json:"fixedManualBorrowAccuracy"`
	// FixedInterestRateAccuracy is the decimal places of a fixed-rate borrowing's interest rate
	FixedInterestRateAccuracy types.Number `json:"fixedInterestRateAccuracy"`
	MinFixedInterestRate      types.Number `json:"minFixedInterestRate"`
	MaxFixedInterestRate      types.Number `json:"maxFixedInterestRate"`
}

// SpotMarginHistoricalInterestRateRequest holds the parameters of Get Historical Interest Rate. StartTime and EndTime
// go together and may be up to 30 days apart; without them the last 7 days are returned
type SpotMarginHistoricalInterestRateRequest struct {
	Currency currency.Code
	// VIPLevel defaults to the account's level; the base level is "No VIP"
	VIPLevel  string
	StartTime time.Time
	EndTime   time.Time
}

// SpotMarginHistoricalInterestRateResponse holds hourly borrowing rates, from Get Historical Interest Rate
type SpotMarginHistoricalInterestRateResponse struct {
	List []SpotMarginInterestRate `json:"list"`
}

// SpotMarginInterestRate is a coin's hourly borrowing rate at a VIP level, from Get Historical Interest Rate
type SpotMarginInterestRate struct {
	Timestamp        types.Time    `json:"timestamp"`
	Currency         currency.Code `json:"currency"`
	HourlyBorrowRate types.Number  `json:"hourlyBorrowRate"`
	VIPLevel         string        `json:"vipLevel"`
}

// SpotMarginModeResponse holds whether spot margin trading is on, from Toggle Margin Trade
type SpotMarginModeResponse struct {
	SpotMarginMode types.Boolean `json:"spotMarginMode"`
}

// SpotMarginStatusResponse holds the account's spot margin status and leverage, from Get Status And Leverage
type SpotMarginStatusResponse struct {
	// SpotLeverage is empty when spot margin trading is off
	SpotLeverage   types.Number  `json:"spotLeverage"`
	SpotMarginMode types.Boolean `json:"spotMarginMode"`
	// EffectiveLeverage is the actual leverage, truncated to 2 decimal places
	EffectiveLeverage types.Number `json:"effectiveLeverage"`
}

// MaxBorrowableAmountResponse holds the most of a coin the account can borrow, from Get Max Borrowable Amount
type MaxBorrowableAmountResponse struct {
	Currency currency.Code `json:"currency"`
	MaxLoan  types.Number  `json:"maxLoan"`
}

// SpotMarginPositionTiersResponse holds the borrowing risk tiers of coins, from Get Position Tiers
type SpotMarginPositionTiersResponse struct {
	List []SpotMarginPositionTiers `json:"list"`
}

// SpotMarginPositionTiers holds a coin's borrowing risk tiers, from Get Position Tiers
type SpotMarginPositionTiers struct {
	Currency               currency.Code            `json:"currency"`
	PositionTiersRatioList []SpotMarginPositionTier `json:"positionTiersRatioList"`
}

// SpotMarginPositionTier is one borrowing risk tier of a coin, from Get Position Tiers
type SpotMarginPositionTier struct {
	Tier types.Number `json:"tier"`
	// BorrowLimit is the borrowing limit accumulated up to this tier
	BorrowLimit                   types.Number `json:"borrowLimit"`
	PositionMaintenanceMarginRate types.Number `json:"positionMMR"`
	PositionInitialMarginRate     types.Number `json:"positionIMR"`
	MaxLeverage                   types.Number `json:"maxLeverage"`
}

// SpotMarginCoinStateResponse holds the spot margin leverage of coins, from Get Coin State
type SpotMarginCoinStateResponse struct {
	List []SpotMarginCoinState `json:"list"`
}

// SpotMarginCoinState is a coin's spot margin leverage, from Get Coin State
type SpotMarginCoinState struct {
	Currency currency.Code `json:"currency"`
	// SpotLeverage is documented as a string, empty when spot margin trading is off, but sent as a bare number
	SpotLeverage types.Number `json:"spotLeverage"`
}

// SpotMarginAvailableAmountToRepayResponse holds how much of a coin can be repaid without converting other assets, from
// Get Available Amount to Repay
type SpotMarginAvailableAmountToRepayResponse struct {
	Currency currency.Code `json:"currency"`
	// LossLessRepaymentAmount is the smaller of the coin's available spot balance and its borrowed amount
	LossLessRepaymentAmount types.Number `json:"lossLessRepaymentAmount"`
}

// SpotMarginAutoRepayModeResponse holds whether coins are repaid automatically, from Set Auto Repay Mode and Get Auto
// Repay Mode
type SpotMarginAutoRepayModeResponse struct {
	Data []SpotMarginAutoRepayMode `json:"data"`
}

// SpotMarginAutoRepayMode holds whether a coin's borrowing is repaid automatically, at 0 and 30 minutes past each
// hour, from its spot balance
type SpotMarginAutoRepayMode struct {
	Currency      currency.Code `json:"currency"`
	AutoRepayMode types.Boolean `json:"autoRepayMode"`
}

// SpotMarginFixedRateBorrowOrderQuoteRequest holds the parameters of Get Fixed-Rate Borrow Order Quote
type SpotMarginFixedRateBorrowOrderQuoteRequest struct {
	OrderCurrency currency.Code
	// Term is in days: 7, 14, 30, 90 or 180
	Term uint64
	// OrderBy sorts by apy, term or quantity
	OrderBy string
	// Sort is 0 for ascending, the default, and 1 for descending
	Sort  uint64
	Limit uint64
}

// SpotMarginFixedRateBorrowOrderQuoteResponse holds the fixed-rate lending offers of a coin, from Get Fixed-Rate Borrow
// Order Quote
type SpotMarginFixedRateBorrowOrderQuoteResponse struct {
	List []SpotMarginFixedRateBorrowOrderQuote `json:"list"`
}

// SpotMarginFixedRateBorrowOrderQuote is a fixed-rate lending offer, from Get Fixed-Rate Borrow Order Quote
type SpotMarginFixedRateBorrowOrderQuote struct {
	OrderCurrency currency.Code `json:"orderCurrency"`
	// Term is in days
	Term       uint64       `json:"term"`
	AnnualRate types.Number `json:"annualRate"`
	Quantity   types.Number `json:"qty"`
}

// SpotMarginFixedRateBorrowRequest holds the parameters of Fixed-Rate Borrow
type SpotMarginFixedRateBorrowRequest struct {
	OrderCurrency currency.Code
	OrderAmount   float64
	// AnnualRate is the interest rate to borrow at as a fraction, 0.02 for 2%
	AnnualRate float64
	// Term is in days: 7, 14, 30, 90 or 180
	Term uint64
	// RepayType is 1 to repay automatically from the account when the loan is due, the default, or 2 to turn it into a
	// flexible loan
	RepayType string
	// StrategyType is PARTIAL to allow a partial fill, the default, or FULL
	StrategyType string
}

// SpotMarginFixedRateBorrowResponse holds the loan order Fixed-Rate Borrow placed
type SpotMarginFixedRateBorrowResponse struct {
	OrderID string `json:"orderId"`
}

// SpotMarginFixedRateBorrowOrderInfoRequest holds the parameters of Get Fixed-Rate Borrow Order Info
type SpotMarginFixedRateBorrowOrderInfoRequest struct {
	OrderID       string
	OrderCurrency currency.Code
	// State is 1 matching, 2 partially filled and cancelled, 3 fully filled or 4 cancelled
	State uint64
	// Term is in days: 7, 14, 30, 90 or 180
	Term   uint64
	Limit  uint64
	Cursor string
}

// SpotMarginFixedRateBorrowOrderInfoResponse holds a page of fixed-rate loan orders, newest first, from Get Fixed-Rate
// Borrow Order Info
type SpotMarginFixedRateBorrowOrderInfoResponse struct {
	List           []SpotMarginFixedRateBorrowOrder `json:"list"`
	NextPageCursor string                           `json:"nextPageCursor"`
}

// SpotMarginFixedRateBorrowOrder is a fixed-rate loan order, from Get Fixed-Rate Borrow Order Info
type SpotMarginFixedRateBorrowOrder struct {
	AnnualRate types.Number `json:"annualRate"`
	// OrderID is documented as a number but sent as a string
	OrderID        string        `json:"orderId"`
	OrderTime      types.Time    `json:"orderTime"`
	FilledQuantity types.Number  `json:"filledQty"`
	OrderQuantity  types.Number  `json:"orderQty"`
	OrderCurrency  currency.Code `json:"orderCurrency"`
	// State is 1 matching, 2 partially filled and cancelled, 3 fully filled, 4 cancelled or 5 failed
	State uint64 `json:"state"`
	// Term is in days
	Term uint64 `json:"term"`
	// RepayType is 1 to repay automatically or 2 to turn the loan into a flexible loan when due
	RepayType string `json:"repayType"`
	// StrategyType is PARTIAL or FULL
	StrategyType string `json:"strategyType"`
}

// SpotMarginFixedRateBorrowContractInfoRequest holds the parameters of Get Fixed-Rate Borrow Contract Info
type SpotMarginFixedRateBorrowContractInfoRequest struct {
	OrderID       string
	OrderCurrency currency.Code
	// Term is in days: 7, 14, 30, 90 or 180
	Term   uint64
	Limit  uint64
	Cursor string
}

// SpotMarginFixedRateBorrowContractInfoResponse holds a page of fixed-rate loans, newest first, from Get Fixed-Rate
// Borrow Contract Info
type SpotMarginFixedRateBorrowContractInfoResponse struct {
	List           []SpotMarginFixedRateBorrowContract `json:"list"`
	NextPageCursor string                              `json:"nextPageCursor"`
}

// SpotMarginFixedRateBorrowContract is a fixed-rate loan, from Get Fixed-Rate Borrow Contract Info
type SpotMarginFixedRateBorrowContract struct {
	AnnualRate     types.Number  `json:"annualRate"`
	BorrowCurrency currency.Code `json:"borrowCurrency"`
	BorrowTime     types.Time    `json:"borrowTime"`
	InterestPaid   types.Number  `json:"interestPaid"`
	LoanID         string        `json:"loanId"`
	OrderID        string        `json:"orderId"`
	// RepaymentTime is when the loan is due
	RepaymentTime           types.Time   `json:"repaymentTime"`
	ResidualPenaltyInterest types.Number `json:"residualPenaltyInterest"`
	ResidualPrincipal       types.Number `json:"residualPrincipal"`
	// Status is 1 unrepaid, 2 fully repaid or 3 overdue
	Status uint64 `json:"status"`
	// Term is in days; it is a string here, unlike the number Get Fixed-Rate Borrow Order Info sends
	Term types.Number `json:"term"`
	// RepayType is 1 to repay automatically, 2 to turn the loan into a flexible loan when due, or 0 for loans made
	// before automatic repayment existed
	RepayType string `json:"repayType"`
	// StrategyType is PARTIAL or FULL
	StrategyType string `json:"strategyType"`
}

// SpotMarginLiabilityResponse holds the account's liability in a coin, from Get Liability Info
type SpotMarginLiabilityResponse struct {
	Currency             currency.Code `json:"currency"`
	TotalBorrowAmount    types.Number  `json:"totalBorrowAmount"`
	FixedBorrowAmount    types.Number  `json:"fixedBorrowAmount"`
	FlexibleBorrowAmount types.Number  `json:"flexibleBorrowAmount"`
	// SpotTotalBorrow includes the liability of open orders
	SpotTotalBorrow   types.Number `json:"spotTotalBorrow"`
	DerivativesBorrow types.Number `json:"derivativesBorrow"`
}

// SpotMarginFlexibleAvailableInventoryResponse holds how much of a coin the account can borrow at a flexible rate,
// from Get Flexible Available Inventory
type SpotMarginFlexibleAvailableInventoryResponse struct {
	Currency currency.Code `json:"currency"`
	// AvailableInventory is the smallest of the platform's lendable amount, the account's remaining borrowing limit and
	// the borrowing limit of the current leverage tier
	AvailableInventory types.Number `json:"availableInventory"`
	UpdateTime         types.Time   `json:"updateTime"`
}

// SpotMarginFixedRateAvailableInventoryRequest holds the parameters of Get Fixed-Rate Available Inventory
type SpotMarginFixedRateAvailableInventoryRequest struct {
	Currency currency.Code
	// Term is in days: 7, 14, 30, 90 or 180
	Term uint64
	// AnnualRate is the interest rate as a fraction, 0.02 for 2%
	AnnualRate float64
}

// SpotMarginFixedRateAvailableInventoryResponse holds how much of a coin the account can borrow for a term at an
// interest rate, from Get Fixed-Rate Available Inventory
type SpotMarginFixedRateAvailableInventoryResponse struct {
	Currency currency.Code `json:"currency"`
	// Term is in days
	Term       types.Number `json:"term"`
	AnnualRate types.Number `json:"annualRate"`
	// AvailableInventory is the smallest of the platform's lendable amount, the account's remaining borrowing limit and
	// the borrowing limit of the current leverage tier, rounded down to the borrowing precision
	AvailableInventory types.Number `json:"availableInventory"`
	UpdateTime         types.Time   `json:"updateTime"`
}
