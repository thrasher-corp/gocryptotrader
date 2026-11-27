package gate

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CrossMarginCurrenciesResponse represents a currency supported by cross margin
type CrossMarginCurrenciesResponse struct {
	Name                 currency.Code `json:"name"`
	Rate                 types.Number  `json:"rate"`
	CurrencyPrecision    types.Number  `json:"prec"`
	Discount             types.Number  `json:"discount"`
	MinBorrowAmount      types.Number  `json:"min_borrow_amount"`
	UserMaxBorrowAmount  types.Number  `json:"user_max_borrow_amount"`
	TotalMaxBorrowAmount types.Number  `json:"total_max_borrow_amount"`
	Price                types.Number  `json:"price"` // Price change between this currency and USDT
	Loanable             bool          `json:"loanable"`
	Status               uint64        `json:"status"`
}

// CrossMarginCurrencyBalance represents the currency detailed balance information for cross margin
type CrossMarginCurrencyBalance struct {
	Available types.Number `json:"available"`
	Freeze    types.Number `json:"freeze"`
	Borrowed  types.Number `json:"borrowed"`
	Interest  types.Number `json:"interest"`
}

// CrossMarginAccountResponse represents the account detail for cross margin account balance
type CrossMarginAccountResponse struct {
	UserID                      uint64                                `json:"user_id"`
	Locked                      bool                                  `json:"locked"`
	Balances                    map[string]CrossMarginCurrencyBalance `json:"balances"`
	Total                       types.Number                          `json:"total"`
	Borrowed                    types.Number                          `json:"borrowed"`
	Interest                    types.Number                          `json:"interest"`
	Risk                        types.Number                          `json:"risk"`
	TotalInitialMargin          types.Number                          `json:"total_initial_margin"`
	TotalMarginBalance          types.Number                          `json:"total_margin_balance"`
	TotalMaintenanceMargin      types.Number                          `json:"total_maintenance_margin"`
	TotalInitialMarginRate      types.Number                          `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate  types.Number                          `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin        types.Number                          `json:"total_available_margin"`
	TotalPortfolioMarginAccount types.Number                          `json:"portfolio_margin_total"`
}

// CrossMarginAccountHistoryItemResponse represents a cross margin account change history item
type CrossMarginAccountHistoryItemResponse struct {
	ID       string        `json:"id"`
	Time     types.Time    `json:"time"`
	Currency currency.Code `json:"currency"` // Currency changed
	Change   types.Number  `json:"change"`
	Balance  types.Number  `json:"balance"`
	Type     string        `json:"type"`
}

// CrossMarginBorrowLoanRequest represents a cross margin borrow loan parameters
type CrossMarginBorrowLoanRequest struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	Text     string        `json:"text,omitempty"`
}

// CrossMarginLoanResponse represents a cross margin borrow loan response
type CrossMarginLoanResponse struct {
	ID             string        `json:"id"`
	CreateTime     types.Time    `json:"create_time"`
	UpdateTime     types.Time    `json:"update_time"`
	Currency       currency.Code `json:"currency"`
	Amount         types.Number  `json:"amount"`
	Text           string        `json:"text"`
	Status         uint64        `json:"status"` // 1: failed, 2: borrowed, 3: repaid
	Repaid         types.Number  `json:"repaid"`
	RepaidInterest types.Number  `json:"repaid_interest"`
	UnpaidInterest types.Number  `json:"unpaid_interest"`
}
