package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// MarginRiskLimitRequest holds the Get Margin Risk Limit filters
type MarginRiskLimitRequest struct {
	IsIsolated bool
	// Currency applies to cross margin
	Currency currency.Code
	// Symbol applies to isolated margin
	Symbol string
}

// MarginRiskLimitResponse is a currency's cross margin, or a symbol's isolated margin, risk limits and configuration. Cross
// margin fills Currency and the fields without a base or quote prefix; isolated margin fills Symbol and the base and
// quote fields
type MarginRiskLimitResponse struct {
	Timestamp types.Time    `json:"timestamp"`
	Currency  currency.Code `json:"currency"`
	// BorrowMaxAmount is the account's borrowing limit, shown even when the platform has nothing left to lend
	BorrowMaxAmount   types.Number `json:"borrowMaxAmount"`
	BuyMaxAmount      types.Number `json:"buyMaxAmount"`
	HoldMaxAmount     types.Number `json:"holdMaxAmount"`
	MarginCoefficient types.Number `json:"marginCoefficient"`
	// Precision is the currency's number of decimal places, which also sets the minimum repayment
	Precision       uint64       `json:"precision"`
	BorrowMinAmount types.Number `json:"borrowMinAmount"`
	// BorrowMinUnit is the unit a borrowed amount must be a multiple of
	BorrowMinUnit          types.Number `json:"borrowMinUnit"`
	BorrowEnabled          bool         `json:"borrowEnabled"`
	Symbol                 string       `json:"symbol"`
	BaseMaxBorrowAmount    types.Number `json:"baseMaxBorrowAmount"`
	QuoteMaxBorrowAmount   types.Number `json:"quoteMaxBorrowAmount"`
	BaseMaxBuyAmount       types.Number `json:"baseMaxBuyAmount"`
	QuoteMaxBuyAmount      types.Number `json:"quoteMaxBuyAmount"`
	BaseMaxHoldAmount      types.Number `json:"baseMaxHoldAmount"`
	QuoteMaxHoldAmount     types.Number `json:"quoteMaxHoldAmount"`
	BasePrecision          uint64       `json:"basePrecision"`
	QuotePrecision         uint64       `json:"quotePrecision"`
	BaseBorrowMinAmount    types.Number `json:"baseBorrowMinAmount"`
	QuoteBorrowMinAmount   types.Number `json:"quoteBorrowMinAmount"`
	BaseBorrowMinUnit      types.Number `json:"baseBorrowMinUnit"`
	QuoteBorrowMinUnit     types.Number `json:"quoteBorrowMinUnit"`
	BaseBorrowEnabled      bool         `json:"baseBorrowEnabled"`
	QuoteBorrowEnabled     bool         `json:"quoteBorrowEnabled"`
	BaseBorrowCoefficient  types.Number `json:"baseBorrowCoefficient"`
	QuoteBorrowCoefficient types.Number `json:"quoteBorrowCoefficient"`
	BaseMarginCoefficient  types.Number `json:"baseMarginCoefficient"`
	QuoteMarginCoefficient types.Number `json:"quoteMarginCoefficient"`
	// BorrowCoefficient is marked abandoned in KuCoin's docs, but its cross margin example still sends it
	BorrowCoefficient types.Number `json:"borrowCoefficient"`
}
