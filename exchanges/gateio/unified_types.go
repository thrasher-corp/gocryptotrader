package gateio

import (
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BalanceDetails represents a user's balance details
type BalanceDetails struct {
	Available                  types.Number `json:"available"`
	Freeze                     types.Number `json:"freeze"`
	Borrowed                   types.Number `json:"borrowed"`
	NegativeLiabilities        types.Number `json:"negative_liab"`
	FuturesPositionLiabilities types.Number `json:"futures_pos_liab"`
	Equity                     types.Number `json:"equity"`
	// Deprecated: Gate has deprecated it
	TotalFreeze           types.Number `json:"total_freeze"`
	TotalLiabilities      types.Number `json:"total_liab"`
	SpotInUse             types.Number `json:"spot_in_use"`
	Funding               types.Number `json:"funding"`
	FundingVersion        types.Number `json:"funding_version"`
	CrossBalance          types.Number `json:"cross_balance"`
	IsolatedBalance       types.Number `json:"iso_balance"`
	InitialMargin         types.Number `json:"im"`
	MaintenanceMargin     types.Number `json:"mm"`
	InitialMarginRate     types.Number `json:"imr"`
	MaintenanceMarginRate types.Number `json:"mmr"`
	MarginBalance         types.Number `json:"margin_balance"`
	AvailableMargin       types.Number `json:"available_margin"`
	EnabledCollateral     bool         `json:"enabled_collateral"`
	BalanceVersion        uint64       `json:"balance_version"`
}

// UnifiedUserAccountResponse represents a unified user account
type UnifiedUserAccountResponse struct {
	Mode        string                           `json:"mode"`
	UserID      uint64                           `json:"user_id"`
	RefreshTime types.Time                       `json:"refresh_time"`
	Locked      bool                             `json:"locked"`
	Balances    map[currency.Code]BalanceDetails `json:"balances"`
	// Deprecated: use UnifiedAccountTotal
	Total                          types.Number `json:"total"`
	Borrowed                       types.Number `json:"borrowed"`
	TotalInitialMargin             types.Number `json:"total_initial_margin"`
	TotalMarginBalance             types.Number `json:"total_margin_balance"`
	TotalMaintenanceMargin         types.Number `json:"total_maintenance_margin"`
	TotalInitialMarginRate         types.Number `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate     types.Number `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin           types.Number `json:"total_available_margin"`
	UnifiedAccountTotal            types.Number `json:"unified_account_total"`
	UnifiedAccountTotalLiabilities types.Number `json:"unified_account_total_liab"`
	UnifiedAccountTotalEquity      types.Number `json:"unified_account_total_equity"`
	// Deprecated: Gate sets leverage per currency
	Leverage         types.Number `json:"leverage"`
	SpotOrderLoss    types.Number `json:"spot_order_loss"`
	OptionsOrderLoss types.Number `json:"options_order_loss"`
	SpotHedge        bool         `json:"spot_hedge"`
	UseFunding       bool         `json:"use_funding"`
	IsAllCollateral  bool         `json:"is_all_collateral"`
}

// BorrowOrRepayRequest holds a request parameter for asset borrow and repay requests
type BorrowOrRepayRequest struct {
	Currency  currency.Code `json:"currency"`
	Type      string        `json:"type"` // Type: borrow - "borrow", repay - "repay"
	Amount    types.Number  `json:"amount"`
	RepaidAll bool          `json:"repaid_all,omitempty"` // Repays the whole loan, overriding Amount; repay only
	Text      string        `json:"text,omitempty"`
}

// BorrowOrRepayResponse holds the transaction created by a unified account borrow or repayment
type BorrowOrRepayResponse struct {
	TransactionID uint64 `json:"tran_id"`
}

// UnifiedLoansRequest holds the filters and paging of a unified account loan or loan record query
type UnifiedLoansRequest struct {
	Currency currency.Code
	Type     string // "platform" or "margin" for loans, "borrow" or "repay" for loan records
	Page     uint64
	Limit    uint64
}

// UnifiedLoanResponse holds an outstanding unified account loan
type UnifiedLoanResponse struct {
	Currency     currency.Code `json:"currency"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Type         string        `json:"type"` // "platform" or "margin"
	CreateTime   types.Time    `json:"create_time"`
	UpdateTime   types.Time    `json:"update_time"`
}

// UnifiedLoanRecordResponse holds a unified account borrow or repayment record
type UnifiedLoanRecordResponse struct {
	ID            uint64        `json:"id"`
	Type          string        `json:"type"` // "borrow" or "repay"
	RepaymentType string        `json:"repayment_type"`
	BorrowType    string        `json:"borrow_type"`
	MarginMode    string        `json:"margin_mode"`
	CurrencyPair  currency.Pair `json:"currency_pair"`
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	CreateTime    types.Time    `json:"create_time"`
}

// UnifiedInterestRecordsRequest holds the filters and paging of a unified account interest deduction record query
type UnifiedInterestRecordsRequest struct {
	Currency  currency.Code
	Type      string // "platform" or "margin", which Gate defaults to
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64
}

// UnifiedAccountModeRequest holds a request parameter for setting a unified account mode
type UnifiedAccountModeRequest struct {
	Mode     string                             `json:"mode"`
	Settings *UnifiedAccountModeSettingsRequest `json:"settings,omitempty"`
}

// UnifiedAccountModeSettingsRequest holds the unified account mode switches to change; nil switches are left unchanged
// so that a switch can be explicitly turned off
type UnifiedAccountModeSettingsRequest struct {
	USDTFutures *bool `json:"usdt_futures,omitempty"`
	SpotHedge   *bool `json:"spot_hedge,omitempty"`
	UseFunding  *bool `json:"use_funding,omitempty"`
	Options     *bool `json:"options,omitempty"`
}

// UnifiedAccountModeResponse holds the unified account mode and its switches
type UnifiedAccountModeResponse struct {
	Mode     string                     `json:"mode"`
	Settings UnifiedAccountModeSettings `json:"settings"`
}

// UnifiedAccountModeSettings holds the unified account mode switches
type UnifiedAccountModeSettings struct {
	USDTFutures bool `json:"usdt_futures"`
	SpotHedge   bool `json:"spot_hedge"`
	UseFunding  bool `json:"use_funding"`
	Options     bool `json:"options"`
}

// TierUpperLimit is a tier's upper boundary. Gate reports the unbounded final tier as "+" or an empty string, which
// decodes to positive infinity.
type TierUpperLimit float64

// UnmarshalJSON decodes a tier upper boundary
func (t *TierUpperLimit) UnmarshalJSON(data []byte) error {
	if s := string(data); s == `"+"` || s == `""` {
		*t = TierUpperLimit(math.Inf(1))
		return nil
	}
	var limit types.Number
	if err := limit.UnmarshalJSON(data); err != nil {
		return err
	}
	*t = TierUpperLimit(limit)
	return nil
}

// MarshalJSON encodes a tier upper boundary, using Gate's "+" for an unbounded tier
func (t TierUpperLimit) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(t), 1) {
		return []byte(`"+"`), nil
	}
	return types.Number(t).MarshalJSON()
}

// Float64 returns the boundary as a float64
func (t TierUpperLimit) Float64() float64 {
	return float64(t)
}

// UnifiedAccountTieredDetailResponse unified account tiered list
type UnifiedAccountTieredDetailResponse struct {
	Currency      currency.Code          `json:"currency"`
	DiscountTiers []*UnifiedDiscountTier `json:"discount_tiers"`
}

// UnifiedDiscountTier holds a unified account collateral discount tier
type UnifiedDiscountTier struct {
	Tier       string         `json:"tier"`
	Discount   types.Number   `json:"discount"`
	LowerLimit types.Number   `json:"lower_limit"`
	UpperLimit TierUpperLimit `json:"upper_limit"`
	Leverage   types.Number   `json:"leverage"`
}

// UnifiedAccountLoanMarginResponse holds unified account loan margin tiered detail
type UnifiedAccountLoanMarginResponse struct {
	Currency    currency.Code        `json:"currency"`
	MarginTiers []*UnifiedMarginTier `json:"margin_tiers"`
}

// UnifiedMarginTier holds a unified account borrowing margin tier
type UnifiedMarginTier struct {
	Tier       string         `json:"tier"`
	MarginRate types.Number   `json:"margin_rate"`
	LowerLimit types.Number   `json:"lower_limit"`
	UpperLimit TierUpperLimit `json:"upper_limit"`
	Leverage   types.Number   `json:"leverage"`
}

// PortfolioMarginCalculatorRequest holds a request parameters for a portfolio margin calculation
type PortfolioMarginCalculatorRequest struct {
	SpotBalances     []*SpotBalanceParams `json:"spot_balances,omitempty"`
	SpotOrders       []*SpotOrderParams   `json:"spot_orders,omitempty"`
	FuturesPositions []*FuturesOrderInfo  `json:"futures_positions,omitempty"`
	FuturesOrders    []*FuturesOrderInfo  `json:"futures_orders,omitempty"`
	OptionsPositions []*OptionsOrderInfo  `json:"options_positions,omitempty"`
	OptionsOrders    []*OptionsOrderInfo  `json:"options_orders,omitempty"`
	SpotHedge        bool                 `json:"spot_hedge,omitempty"`
}

// SpotBalanceParams holds a spot balance parameters
type SpotBalanceParams struct {
	Currency currency.Code `json:"currency"`
	Equity   types.Number  `json:"equity"` // Balance less borrowings, negative for a net short exposure
	Freeze   types.Number  `json:"freeze,omitempty"`
}

// SpotOrderParams represents a unified account margin calculator spot order parameters
type SpotOrderParams struct {
	CurrencyPair currency.Pair `json:"currency_pairs"`
	OrderPrice   types.Number  `json:"order_price"`
	Count        types.Number  `json:"count,omitempty"` // Initial order quantity, not used in the calculation
	Left         types.Number  `json:"left"`
	Side         order.Side    `json:"type"`
}

// MarshalJSON lowercases the order side, which order.Side otherwise marshals as "BUY"/"SELL" while Gate only accepts
// "buy"/"sell"
func (s *SpotOrderParams) MarshalJSON() ([]byte, error) {
	type Alias SpotOrderParams
	return json.Marshal(&struct {
		Side string `json:"type"`
		*Alias
	}{
		Side:  s.Side.Lower(),
		Alias: (*Alias)(s),
	})
}

// FuturesOrderInfo holds a USDT perpetual position or order for the portfolio margin calculator. Sizes are signed
// contract quantities, negative for short positions and sell orders.
type FuturesOrderInfo struct {
	Contract currency.Pair `json:"contract"`
	Size     types.Number  `json:"size"`
	Left     types.Number  `json:"left,omitempty"` // Orders only
}

// OptionsOrderInfo holds an options position or order for the portfolio margin calculator. Sizes are signed contract
// quantities, negative for short positions and sell orders.
type OptionsOrderInfo struct {
	OptionsName currency.Pair `json:"options_name"`
	Size        types.Number  `json:"size"`
	Left        types.Number  `json:"left,omitempty"` // Orders only
}

// PortfolioMarginCalculationResponse holds a portfolio margin calculation response
type PortfolioMarginCalculationResponse struct {
	MaintainMarginTotal types.Number               `json:"maintain_margin_total"`
	InitialMarginTotal  types.Number               `json:"initial_margin_total"`
	CalculateTime       types.Time                 `json:"calculate_time"`
	RiskUnits           []*PortfolioMarginRiskUnit `json:"risk_unit"`
}

// PortfolioMarginRiskUnit holds the portfolio margin calculation for one underlying
type PortfolioMarginRiskUnit struct {
	Symbol         currency.Code            `json:"symbol"`
	SpotInUse      types.Number             `json:"spot_in_use"`
	MaintainMargin types.Number             `json:"maintain_margin"`
	InitialMargin  types.Number             `json:"initial_margin"`
	MarginResults  []*PortfolioMarginResult `json:"margin_result"`
	Delta          types.Number             `json:"delta"`
	Gamma          types.Number             `json:"gamma"`
	Theta          types.Number             `json:"theta"`
	Vega           types.Number             `json:"vega"`
}

// PortfolioMarginResult holds the margin requirements of one position combination of a risk unit
type PortfolioMarginResult struct {
	Type                 string            `json:"type"`
	ProfitLossRanges     []*VolatilityInfo `json:"profit_loss_ranges"`
	MaxLoss              *VolatilityInfo   `json:"max_loss"`
	StressTest           types.Number      `json:"mr1"`
	BasisSpreadRisk      types.Number      `json:"mr2"`
	VolatilitySpreadRisk types.Number      `json:"mr3"`
	OptionShortRisk      types.Number      `json:"mr4"`
}

// VolatilityInfo represents a volatility info
type VolatilityInfo struct {
	PricePercentage             types.Number `json:"price_percentage"`
	ImpliedVolatilityPercentage types.Number `json:"implied_volatility_percentage"`
	ProfitLoss                  types.Number `json:"profit_loss"`
}

// UnifiedLeverageRequest holds the leverage to apply to every currency the user borrows
type UnifiedLeverageRequest struct {
	Leverage types.Number `json:"leverage"`
}

// LeverageFailedCurrencyResponse holds a currency whose leverage could not be updated
type LeverageFailedCurrencyResponse struct {
	Currency currency.Code `json:"currency"`
	Reason   string        `json:"reason"`
}

// UnifiedLeverageSettingResponse holds a unified account currency leverage multiplier setting.
type UnifiedLeverageSettingResponse struct {
	Currency currency.Code `json:"currency"`
	Leverage types.Number  `json:"leverage"`
}

// UnifiedCurrencyLeverageConfigResponse holds the maximum and minimum currency leverage configuration.
type UnifiedCurrencyLeverageConfigResponse struct {
	CurrentLeverage          types.Number `json:"current_leverage"`
	MinLeverage              types.Number `json:"min_leverage"`
	MaxLeverage              types.Number `json:"max_leverage"`
	Debit                    types.Number `json:"debit"`
	AvailableMargin          types.Number `json:"available_margin"`
	Borrowable               types.Number `json:"borrowable"`
	ExceptLeverageBorrowable types.Number `json:"except_leverage_borrowable"`
}

// UnifiedLoanCurrencyResponse holds detail for a loan currency supported by unified account.
type UnifiedLoanCurrencyResponse struct {
	Name                     currency.Code `json:"name"`
	Precision                types.Number  `json:"prec"`
	MinBorrowAmount          types.Number  `json:"min_borrow_amount"`
	UserMaxBorrowAmountUSDT  types.Number  `json:"user_max_borrow_amount"`
	TotalMaxBorrowAmountUSDT types.Number  `json:"total_max_borrow_amount"`
	LoanStatus               string        `json:"loan_status"` // "enable" or "disable"
}

// UnifiedHistoricalLendingRatesRequest holds the filters and paging of a unified account historical lending rate query
type UnifiedHistoricalLendingRatesRequest struct {
	Currency currency.Code
	Tier     string // VIP level of the floating rate
	Page     uint64
	Limit    uint64
}

// UnifiedHistoricalLendingRateResponse holds historical interest rate data for a currency.
type UnifiedHistoricalLendingRateResponse struct {
	Currency   currency.Code         `json:"currency"`
	Tier       string                `json:"tier"`
	TierUpRate types.Number          `json:"tier_up_rate"`
	Rates      []*UnifiedLendingRate `json:"rates"`
}

// UnifiedLendingRate holds the lending rate of an hour
type UnifiedLendingRate struct {
	Time types.Time   `json:"time"`
	Rate types.Number `json:"rate"`
}

// UnifiedSetCollateralRequest is the request body for setting collateral currency.
type UnifiedSetCollateralRequest struct {
	CollateralType uint64          `json:"collateral_type"` // 0: all currencies, 1: the enabled currencies only
	EnableList     []currency.Code `json:"enable_list,omitempty"`
	DisableList    []currency.Code `json:"disable_list,omitempty"`
}

// UnifiedSetCollateralResponse holds the result of setting collateral currency.
type UnifiedSetCollateralResponse struct {
	IsSuccess bool `json:"is_success"`
}

// UnifiedDebtCurrency holds debt currency detail for quick repayment estimation.
type UnifiedDebtCurrency struct {
	Currency        currency.Code `json:"currency"`
	DebtAmount      types.Number  `json:"debt_amount"`
	EstimatedUSD    types.Number  `json:"estimated_usd"`
	Borrowed        types.Number  `json:"borrowed"`
	NegativeBalance types.Number  `json:"neg_balance"`
}

// UnifiedAvailableCurrency holds available currency detail for quick repayment estimation.
type UnifiedAvailableCurrency struct {
	Currency     currency.Code `json:"currency"`
	Available    types.Number  `json:"available"`
	EstimatedUSD types.Number  `json:"estimated_usd"`
}

// UnifiedQuickEstimateRepaymentResponse holds the estimated quick repayment details.
type UnifiedQuickEstimateRepaymentResponse struct {
	DebtCurrencies      []*UnifiedDebtCurrency      `json:"debt_currencies"`
	AvailableCurrencies []*UnifiedAvailableCurrency `json:"available_currencies"`
}

// UnifiedQuickRepaymentRequest is the request body for quick repayment.
type UnifiedQuickRepaymentRequest struct {
	DebtCurrencies      []currency.Code `json:"debt_currencies"`
	AvailableCurrencies []currency.Code `json:"available_currencies"`
}

// UnifiedRepaidInfo holds repaid currency detail from a quick repayment.
type UnifiedRepaidInfo struct {
	Currency currency.Code `json:"currency"`
	Repaid   types.Number  `json:"repaid"`
	Left     types.Number  `json:"left"`
}

// UnifiedUsedInfo holds used currency detail from a quick repayment.
type UnifiedUsedInfo struct {
	Currency currency.Code `json:"currency"`
	Used     types.Number  `json:"used"`
}

// UnifiedQuickRepaymentResponse holds the result of a quick repayment.
type UnifiedQuickRepaymentResponse struct {
	OrderID     string               `json:"order_id"`
	RepaidInfos []*UnifiedRepaidInfo `json:"repaid_infos"`
	UsedInfos   []*UnifiedUsedInfo   `json:"used_infos"`
}

// DeltaNeutralRequest holds a request to enable or disable the account Delta-neutral strategy mode
type DeltaNeutralRequest struct {
	Enabled bool `json:"enabled"`
}

// DeltaNeutralResponse holds whether the account Delta-neutral strategy mode is enabled
type DeltaNeutralResponse struct {
	Enabled bool `json:"enabled"`
}
