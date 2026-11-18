package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ConvertSymbolRequest holds the Get Convert Symbol parameters; FromCurrency and ToCurrency are required
type ConvertSymbolRequest struct {
	FromCurrency currency.Code
	ToCurrency   currency.Code
	// OrderType is MARKET (the default) or LIMIT
	OrderType string
}

// ConvertSymbolResponse holds the trade size limits of converting one currency to another
type ConvertSymbolResponse struct {
	FromCurrency        currency.Code `json:"fromCurrency"`
	FromCurrencyMaxSize types.Number  `json:"fromCurrencyMaxSize"`
	FromCurrencyMinSize types.Number  `json:"fromCurrencyMinSize"`
	FromCurrencyStep    types.Number  `json:"fromCurrencyStep"`
	ToCurrency          currency.Code `json:"toCurrency"`
	ToCurrencyMaxSize   types.Number  `json:"toCurrencyMaxSize"`
	ToCurrencyMinSize   types.Number  `json:"toCurrencyMinSize"`
	ToCurrencyStep      types.Number  `json:"toCurrencyStep"`
}

// ConvertCurrenciesResponse lists the currencies that can be converted
type ConvertCurrenciesResponse struct {
	Currencies []ConvertCurrency `json:"currencies"`
	// USDTCurrencyLimit holds the USDT size limits of converting USDT with each currency
	USDTCurrencyLimit []ConvertUSDTCurrencyLimit `json:"usdtCurrencyLimit"`
}

// ConvertCurrency holds the size limits of one conversion of a currency
type ConvertCurrency struct {
	Currency currency.Code `json:"currency"`
	MaxSize  types.Number  `json:"maxSize"`
	MinSize  types.Number  `json:"minSize"`
	Step     types.Number  `json:"step"`
	// TradeDirection is the direction the currency can be converted in, such as ALL
	TradeDirection string `json:"tradeDirection"`
}

// ConvertUSDTCurrencyLimit holds the USDT size limits of one conversion of USDT with a currency
type ConvertUSDTCurrencyLimit struct {
	Currency currency.Code `json:"currency"`
	MaxSize  types.Number  `json:"maxSize"`
	MinSize  types.Number  `json:"minSize"`
	Step     types.Number  `json:"step"`
}

// ConvertQuoteRequest holds the Get Convert Quote and Get Convert Limit Quote parameters; FromCurrency, ToCurrency and
// exactly one of FromCurrencySize and ToCurrencySize are required
type ConvertQuoteRequest struct {
	FromCurrency     currency.Code
	ToCurrency       currency.Code
	FromCurrencySize float64
	ToCurrencySize   float64
}

// ConvertQuoteResponse is a quote for a convert market order
type ConvertQuoteResponse struct {
	QuoteID string `json:"quoteId"`
	// Price is the ToCurrency amount per unit of FromCurrency
	Price            types.Number `json:"price"`
	FromCurrencySize types.Number `json:"fromCurrencySize"`
	ToCurrencySize   types.Number `json:"toCurrencySize"`
	// ValidUntill is when the quote expires; the spelling is KuCoin's
	ValidUntill types.Time `json:"validUntill"`
}

// AddConvertOrderRequest holds the Add Convert Order parameters; ClientOrderID and QuoteID are required
type AddConvertOrderRequest struct {
	// ClientOrderID is at most 40 letters, digits, underscores and hyphens
	ClientOrderID string
	QuoteID       string
	// AccountType is BOTH (the default), FUNDING or TRADING
	AccountType string
}

// AddConvertOrderResponse identifies the convert order KuCoin accepted (Add Convert Order, Add Convert Limit Order)
type AddConvertOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// ConvertOrderDetailResponse is a convert market order (Get Convert Order Detail)
type ConvertOrderDetailResponse ConvertOrderResponse

// ConvertOrdersRequest holds the Get Convert Order History and Get Convert Limit Orders parameters, all optional
type ConvertOrdersRequest struct {
	StartAt time.Time
	EndAt   time.Time
	Page    uint64
	// PageSize is between 20 and 100
	PageSize uint64
	// Status is OPEN, SUCCESS or FAIL, or CANCELLED for limit orders
	Status string
}

// ConvertOrderHistoryResponse is a page of convert market orders
type ConvertOrderHistoryResponse struct {
	CurrentPage uint64                 `json:"currentPage"`
	PageSize    uint64                 `json:"pageSize"`
	TotalNumber uint64                 `json:"totalNum"`
	TotalPage   uint64                 `json:"totalPage"`
	Items       []ConvertOrderResponse `json:"items"`
}

// ConvertOrderResponse is a convert market order (Get Convert Order Detail, Get Convert Order History)
type ConvertOrderResponse struct {
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	// Status is OPEN, SUCCESS or FAIL
	Status           string        `json:"status"`
	FromCurrency     currency.Code `json:"fromCurrency"`
	ToCurrency       currency.Code `json:"toCurrency"`
	FromCurrencySize types.Number  `json:"fromCurrencySize"`
	ToCurrencySize   types.Number  `json:"toCurrencySize"`
	// AccountType is BOTH, FUNDING or TRADING
	AccountType string `json:"accountType"`
	// Price is the ToCurrency amount per unit of FromCurrency
	Price     types.Number `json:"price"`
	OrderTime types.Time   `json:"orderTime"`
}

// AddConvertLimitOrderRequest holds the Add Convert Limit Order parameters; every field but AccountType is required
type AddConvertLimitOrderRequest struct {
	// ClientOrderID is at most 40 letters, digits, underscores and hyphens
	ClientOrderID string
	// AccountType is BOTH (the default), FUNDING or TRADING
	AccountType      string
	FromCurrency     currency.Code
	ToCurrency       currency.Code
	FromCurrencySize float64
	ToCurrencySize   float64
}

// ConvertLimitQuoteResponse holds the protection price that a convert limit order's price must be at least
type ConvertLimitQuoteResponse struct {
	// Price is the ToCurrency amount per unit of FromCurrency
	Price types.Number `json:"price"`
	// ValidUntill is when the quote expires; the spelling is KuCoin's
	ValidUntill types.Time `json:"validUntill"`
}

// ConvertLimitOrderDetailResponse is a convert limit order (Get Convert Limit Order Detail)
type ConvertLimitOrderDetailResponse ConvertLimitOrderResponse

// ConvertLimitOrdersResponse is a page of active and historical convert limit orders
type ConvertLimitOrdersResponse struct {
	CurrentPage uint64                      `json:"currentPage"`
	PageSize    uint64                      `json:"pageSize"`
	TotalNumber uint64                      `json:"totalNum"`
	TotalPage   uint64                      `json:"totalPage"`
	Items       []ConvertLimitOrderResponse `json:"items"`
}

// ConvertLimitOrderResponse is a convert limit order (Get Convert Limit Order Detail, Get Convert Limit Orders)
type ConvertLimitOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	// Status is OPEN, SUCCESS, FAIL or CANCELLED
	Status           string        `json:"status"`
	FromCurrency     currency.Code `json:"fromCurrency"`
	ToCurrency       currency.Code `json:"toCurrency"`
	FromCurrencySize types.Number  `json:"fromCurrencySize"`
	ToCurrencySize   types.Number  `json:"toCurrencySize"`
	// AccountType is BOTH, FUNDING or TRADING
	AccountType string `json:"accountType"`
	// Price is the ToCurrency amount per unit of FromCurrency
	Price      types.Number `json:"price"`
	OrderTime  types.Time   `json:"orderTime"`
	ExpiryTime types.Time   `json:"expiryTime"`
	CancelTime types.Time   `json:"cancelTime"`
	FilledTime types.Time   `json:"filledTime"`
	// CancelType is the type or reason of a cancellation, which KuCoin does not enumerate
	CancelType uint64 `json:"cancelType"`
}
