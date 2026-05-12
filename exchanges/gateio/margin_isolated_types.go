package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// IsolatedMarginAccountBalanceChangeInfo represents margin account balance
type IsolatedMarginAccountBalanceChangeInfo struct {
	ID            string     `json:"id"`
	Time          types.Time `json:"time_ms"`
	Currency      string     `json:"currency"`
	CurrencyPair  string     `json:"currency_pair"`
	AmountChanged string     `json:"change"`
	Balance       string     `json:"balance"`
}

// MarginFundingAccountItem represents funding account list item.
type MarginFundingAccountItem struct {
	Currency     string       `json:"currency"`
	Available    types.Number `json:"available"`
	LockedAmount types.Number `json:"locked"`
	Lent         string       `json:"lent"`       // Outstanding loan amount yet to be repaid
	TotalLent    string       `json:"total_lent"` // Amount used for lending. total_lent = lent + locked
}

// OnOffStatus represents on or off status response status
type OnOffStatus struct {
	Status string `json:"status"`
}

// MaxTransferAndLoanAmount represents the maximum amount to transfer, borrow, or lend for specific currency and currency pair
type MaxTransferAndLoanAmount struct {
	Currency     string       `json:"currency"`
	CurrencyPair string       `json:"currency_pair"`
	Amount       types.Number `json:"amount"`
}

// IsolatedMarginLendingMarket represents an isolated margin lending market
type IsolatedMarginLendingMarket struct {
	Pair                     currency.Pair `json:"currency_pair"`
	BaseMinimumBorrowAmount  types.Number  `json:"base_min_borrow_amount"`
	QuoteMinimumBorrowAmount types.Number  `json:"quote_min_borrow_amount"`
	PositionLeverage         types.Number  `json:"leverage"`
}

// IsolatedMarginLendingTier represents the lending tier information for isolated margin accounts.
type IsolatedMarginLendingTier struct {
	MaximumBorrowingLimit      types.Number `json:"upper_limit"`
	MaintenanceMarginRate      types.Number `json:"mmr"`
	MaximumPermissibleLeverage types.Number `json:"leverage"`
}

// IsolatedBorrowRepayRequest represents the request parameters for the isolated margin borrow-or-repay
// endpoint (POST margin/uni/loans).
type IsolatedBorrowRepayRequest struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Type         string        `json:"type"` // Type is either "borrow" or "repay".
	Amount       types.Number  `json:"amount"`
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

// MaxBorrowableAmount represents the max borrowable amount for specific margin currency
type MaxBorrowableAmount struct {
	Currency   currency.Code `json:"currency"`
	Borrowable types.Number  `json:"borrowable"`
	Pair       currency.Pair `json:"currency_pair"`
}

// AccountBalanceInformation represents currency account balance information.
type AccountBalanceInformation struct {
	Available    types.Number  `json:"available"`
	Borrowed     types.Number  `json:"borrowed"`
	Interest     types.Number  `json:"interest"`
	Currency     currency.Code `json:"currency"`
	LockedAmount types.Number  `json:"locked"`
}

// MarginAccountItem margin account item.
type MarginAccountItem struct {
	CurrencyPair          currency.Pair             `json:"currency_pair"`
	AccountType           string                    `json:"account_type"` // "risk" (risk rate account),"mmr" (maintenance margin rate account), or "inactive" (market not activated).
	Leverage              types.Number              `json:"leverage"`
	Locked                bool                      `json:"locked"`
	RiskRate              types.Number              `json:"risk"`
	MaintenanceMarginRate types.Number              `json:"mmr"`
	Base                  AccountBalanceInformation `json:"base"`
	Quote                 AccountBalanceInformation `json:"quote"`
}

// IsolatedMarginPoolLoanResponse holds the response from the API when fetching margin loan information, including a list of loans and VIP settings.
type IsolatedMarginPoolLoanResponse struct {
	Timestamp types.Time `json:"timestamp"`
	Method    string     `json:"method"`
	Code      int        `json:"code"`
	Message   string     `json:"message"`
	Data      struct {
		Total       int          `json:"total"`
		List        []Loan       `json:"list"`
		VIPSettings []VIPSetting `json:"vip_settings"`
	} `json:"data"`
}

// Loan represents an individual loan entry
type Loan struct {
	Market                      currency.Pair `json:"market"`
	Deal                        types.Number  `json:"deal"`
	MoneyLastTimeLoanRateHour   types.Number  `json:"money_last_time_loan_rate_hour"`
	MoneyLastTimeLoanRateYear   types.Number  `json:"money_last_time_loan_rate_year"`
	MoneyTotalLendAvailable     types.Number  `json:"money_total_lend_available"`
	MoneyUserMaxBorrowAmount    types.Number  `json:"money_user_max_borrow_amount"`
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
