package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// IsolatedMarginAccountBalanceChangeInfoResponse represents margin account balance
type IsolatedMarginAccountBalanceChangeInfoResponse struct {
	ID            string        `json:"id"`
	Time          types.Time    `json:"time_ms"`
	Currency      currency.Code `json:"currency"`
	CurrencyPair  currency.Pair `json:"currency_pair"`
	AmountChanged types.Number  `json:"change"`
	Balance       types.Number  `json:"balance"`
	Type          string        `json:"type"`
}

// MarginFundingAccountItemResponse represents funding account list item.
type MarginFundingAccountItemResponse struct {
	Currency     currency.Code `json:"currency"`
	Available    types.Number  `json:"available"`
	LockedAmount types.Number  `json:"locked"`
	Lent         types.Number  `json:"lent"`       // Outstanding loan amount yet to be repaid
	TotalLent    types.Number  `json:"total_lent"` // Amount used for lending. total_lent = lent + locked
}

// OnOffStatusResponse represents on or off status response status
type OnOffStatusResponse struct {
	Status string `json:"status"`
}

// MaxTransferAndLoanAmountResponse represents the maximum amount to transfer, borrow, or lend for specific currency and currency pair
type MaxTransferAndLoanAmountResponse struct {
	Currency     currency.Code `json:"currency"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
}

// IsolatedMarginLendingMarketResponse represents an isolated margin lending market
type IsolatedMarginLendingMarketResponse struct {
	Pair                     currency.Pair `json:"currency_pair"`
	BaseMinimumBorrowAmount  types.Number  `json:"base_min_borrow_amount"`
	QuoteMinimumBorrowAmount types.Number  `json:"quote_min_borrow_amount"`
	PositionLeverage         types.Number  `json:"leverage"`
	Status                   string        `json:"status"` // "enabled" or "disabled"
	DelistedTime             types.Time    `json:"delisted_time"`
}

// IsTradable returns whether the market is enabled and has not completed delisting.
func (m *IsolatedMarginLendingMarketResponse) IsTradable(now time.Time) bool {
	delisted := m.DelistedTime.Time()
	return m.Status == "enabled" && (delisted.IsZero() || !delisted.Before(now))
}

// IsolatedMarginLendingTierResponse represents the lending tier information for isolated margin accounts.
type IsolatedMarginLendingTierResponse struct {
	MaximumBorrowingLimit      types.Number `json:"upper_limit"`
	MaintenanceMarginRate      types.Number `json:"mmr"`
	MaximumPermissibleLeverage types.Number `json:"leverage"`
}

// MarginMarketLeverageRequest holds the leverage multiplier to set for an isolated margin market
type MarginMarketLeverageRequest struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Leverage     types.Number  `json:"leverage"`
}

// IsolatedBorrowRepayRequest represents the request parameters for the isolated margin borrow-or-repay endpoint.
type IsolatedBorrowRepayRequest struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Type         string        `json:"type"` // Type is either "borrow" or "repay".
	Amount       types.Number  `json:"amount"`
	RepaidAll    bool          `json:"repaid_all,omitempty"` // If true, the amount is ignored and the entire outstanding loan is repaid.
}

// IsolatedMarginLoanResponse represents a record of borrowing or repaying in an isolated margin account.
type IsolatedMarginLoanResponse struct {
	Type         string        `json:"type"` // "borrow" or "repay" only returned in record response
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Amount       types.Number  `json:"amount"`
	CreateTime   types.Time    `json:"create_time"`
	UpdateTime   types.Time    `json:"update_time"`
}

// MaxBorrowableAmountResponse represents the max borrowable amount for specific margin currency
type MaxBorrowableAmountResponse struct {
	Currency   currency.Code `json:"currency"`
	Borrowable types.Number  `json:"borrowable"`
	Pair       currency.Pair `json:"currency_pair"`
}

// AccountBalanceInformation represents currency account balance information.
type AccountBalanceInformation struct {
	Available types.Number `json:"available"` // Includes borrowed funds while positive; interest deductions can make this negative.
	Borrowed  types.Number `json:"borrowed"`  // Outstanding borrowed principal, excluding accrued interest.
	// Interest is present in the response schema but is not populated for isolated margin accounts.
	// Gate.io instead applies interest deductions to Available.
	Interest     types.Number  `json:"interest"`
	Currency     currency.Code `json:"currency"`
	LockedAmount types.Number  `json:"locked"` // Funds reserved by active orders.
}

// MarginAccountItemResponse margin account item.
type MarginAccountItemResponse struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	AccountType  string        `json:"account_type"` // "mmr" (maintenance margin rate account) or "inactive" (market not activated)
	Leverage     types.Number  `json:"leverage"`
	Locked       bool          `json:"locked"`
	// Deprecated: Gate superseded it with MaintenanceMarginRate
	RiskRate              types.Number              `json:"risk"`
	MaintenanceMarginRate types.Number              `json:"mmr"`
	Base                  AccountBalanceInformation `json:"base"`
	Quote                 AccountBalanceInformation `json:"quote"`
}

// IsolatedMarginPoolLoanResponse holds the response from the API when fetching margin loan information, including a list of loans and VIP settings.
type IsolatedMarginPoolLoanResponse struct {
	Timestamp types.Time                 `json:"timestamp"`
	Method    string                     `json:"method"`
	Code      int64                      `json:"code"` // Signed because a failed request reports -1
	Message   string                     `json:"message"`
	Data      IsolatedMarginPoolLoanData `json:"data"`
}

// IsolatedMarginPoolLoanData holds a page of margin loan pools and the VIP level settings applied to them
type IsolatedMarginPoolLoanData struct {
	Total       uint64       `json:"total"`
	List        []Loan       `json:"list"`
	VIPSettings []VIPSetting `json:"vip_settings"`
}

// Loan represents an individual loan entry
type Loan struct {
	Market                      currency.Pair `json:"market"`
	Deal                        types.Number  `json:"deal"`
	MoneyLastTimeLoanRateHour   types.Number  `json:"money_last_time_loan_rate_hour"`
	MoneyLastTimeLoanRateYear   types.Number  `json:"money_last_time_loan_rate_year"`
	MoneyTotalLendAvailable     types.Number  `json:"money_total_lend_available"`
	MoneyUserMaxBorrowAmount    types.Number  `json:"money_user_max_borrow_amount"`
	MoneyTotalLendAvailableFiat types.Number  `json:"money_total_lend_available_fiat"`
	StockLastTimeLoanRateHour   types.Number  `json:"stock_last_time_loan_rate_hour"`
	StockLastTimeLoanRateYear   types.Number  `json:"stock_last_time_loan_rate_year"`
	StockTotalLendAvailable     types.Number  `json:"stock_total_lend_available"`
	StockTotalLendAvailableFiat types.Number  `json:"stock_total_lend_available_fiat"`
	StockUserMaxBorrowAmount    types.Number  `json:"stock_user_max_borrow_amount"`
	Leverage                    types.Number  `json:"leverage"`
}

// VIPSetting represents the VIP level settings for margin trading, including various rates and limits.
type VIPSetting struct {
	VIPLevel             uint8        `json:"vip_level"`
	LendDownRate         types.Number `json:"lend_down_rate"`
	BorrowUpRate         types.Number `json:"borrow_up_rate"`
	MarginQuotaRate      types.Number `json:"margin_quota_rate"`
	CrossMarginQuotaRate types.Number `json:"cross_margin_quota_rate"`
	MortgageQuotaRate    types.Number `json:"mortgage_quota_rate"`
	MarginMultiple       types.Number `json:"margin_multiple"`
	CrossMarginMultiple  types.Number `json:"cross_margin_multiple"`
}
