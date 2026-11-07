package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ConvertCoinListRequest holds the Get Convert Coin List parameters; AccountType is required
type ConvertCoinListRequest struct {
	// AccountType is the wallet: eb_convert_uta or eb_convert_funding
	AccountType string
	// Coin is the coin to convert from; with Side 1 the list holds the coins it converts to, and Side 0 ignores it
	Coin currency.Code
	// Side is 0 for the coins to convert from, with their balances, or 1 for the coins to convert to. It is always sent,
	// as the API documents no default and its example sends 0
	Side uint64
}

// ConvertCoinListResponse lists the coins that can be converted
type ConvertCoinListResponse struct {
	Coins []ConvertCoin `json:"coins"`
}

// ConvertCoin describes converting one coin
type ConvertCoin struct {
	Coin      currency.Code `json:"coin"`
	FullName  string        `json:"fullName"`
	Icon      string        `json:"icon"`
	IconNight string        `json:"iconNight"`
	// AccuracyLength is the coin's precision
	AccuracyLength uint64 `json:"accuracyLength"`
	CoinType       string `json:"coinType"`
	// Balance is the available balance; the exact convertible balance needs Side 1 and the coin
	Balance types.Number `json:"balance"`
	// USDTBalance is Balance valued in USDT
	USDTBalance types.Number `json:"uBalance"`
	// SingleFromMinLimit and SingleFromMaxLimit bound the amount converted from the coin in one conversion
	SingleFromMinLimit types.Number `json:"singleFromMinLimit"`
	SingleFromMaxLimit types.Number `json:"singleFromMaxLimit"`
	// DisableFrom and DisableTo are true when the coin cannot be converted from or to
	DisableFrom bool `json:"disableFrom"`
	DisableTo   bool `json:"disableTo"`
	// TimePeriod and the remaining limits are reserved for future use
	TimePeriod        uint64       `json:"timePeriod"`
	SingleToMinLimit  types.Number `json:"singleToMinLimit"`
	SingleToMaxLimit  types.Number `json:"singleToMaxLimit"`
	DailyFromMinLimit types.Number `json:"dailyFromMinLimit"`
	DailyFromMaxLimit types.Number `json:"dailyFromMaxLimit"`
	DailyToMinLimit   types.Number `json:"dailyToMinLimit"`
	DailyToMaxLimit   types.Number `json:"dailyToMaxLimit"`
}

// RequestQuoteRequest holds the Request a Quote parameters; AccountType, FromCoin, ToCoin and RequestAmount are
// required
type RequestQuoteRequest struct {
	// AccountType is the wallet: eb_convert_uta or eb_convert_funding
	AccountType string
	FromCoin    currency.Code
	ToCoin      currency.Code
	// RequestCoin must be FromCoin for now; empty sends FromCoin
	RequestCoin currency.Code
	// RequestAmount is the amount of RequestCoin to sell
	RequestAmount float64
	// FromCoinType and ToCoinType are crypto
	FromCoinType string
	ToCoinType   string
	// ParamType opFrom and ParamValue, a broker ID, are for API brokers
	ParamType  string
	ParamValue string
	// RequestID is an optional ID of at most 36 characters for tracking the quote
	RequestID string
}

// RequestQuoteResponse is a conversion quote
type RequestQuoteResponse struct {
	QuoteTransactionID string        `json:"quoteTxId"`
	ExchangeRate       types.Number  `json:"exchangeRate"`
	FromCoin           currency.Code `json:"fromCoin"`
	FromCoinType       string        `json:"fromCoinType"`
	ToCoin             currency.Code `json:"toCoin"`
	ToCoinType         string        `json:"toCoinType"`
	FromAmount         types.Number  `json:"fromAmount"`
	ToAmount           types.Number  `json:"toAmount"`
	// ExpiredTime is when the quote expires, 15 seconds after it was made
	ExpiredTime types.Time `json:"expiredTime"`
	RequestID   string     `json:"requestId"`
	// ExtendedTaxAndFee is a compliance field the API returns empty; its items are not documented yet
	ExtendedTaxAndFee []json.RawMessage `json:"extTaxAndFee"`
}

// ConfirmQuoteResponse is the status of a confirmed quote
type ConfirmQuoteResponse struct {
	QuoteTransactionID string `json:"quoteTxId"`
	// ExchangeStatus is init, processing, success or failure
	ExchangeStatus string `json:"exchangeStatus"`
}

// ConvertStatusResponse holds one conversion
type ConvertStatusResponse struct {
	Result ConvertRecord `json:"result"`
}

// ConvertRecord is one conversion
type ConvertRecord struct {
	// AccountType is the wallet: eb_convert_funding or eb_convert_uta for conversions via the API, or funding,
	// funding_fiat, funding_fbtc_convert or funding_block_trade for conversions on the web or app
	AccountType string `json:"accountType"`
	// ExchangeTransactionID is the quote transaction ID
	ExchangeTransactionID string        `json:"exchangeTxId"`
	UserID                string        `json:"userId"`
	FromCoin              currency.Code `json:"fromCoin"`
	FromCoinType          string        `json:"fromCoinType"`
	FromAmount            types.Number  `json:"fromAmount"`
	ToCoin                currency.Code `json:"toCoin"`
	ToCoinType            string        `json:"toCoinType"`
	ToAmount              types.Number  `json:"toAmount"`
	// ExchangeStatus is init, processing, success or failure
	ExchangeStatus string              `json:"exchangeStatus"`
	ExtendedInfo   ConvertExtendedInfo `json:"extInfo"`
	ConvertRate    types.Number        `json:"convertRate"`
	CreatedAt      types.Time          `json:"createdAt"`
}

// ConvertExtendedInfo holds the broker parameters a quote was requested with
type ConvertExtendedInfo struct {
	ParamType  string `json:"paramType"`
	ParamValue string `json:"paramValue"`
}

// ConvertHistoryRequest holds the Get Convert History filters
type ConvertHistoryRequest struct {
	// AccountTypes lists wallet types to return, all of them when empty: eb_convert_funding, eb_convert_uta, funding,
	// funding_fiat, funding_fbtc_convert or funding_block_trade
	AccountTypes []string
	// Index is the page number, starting at 1
	Index uint64
	// Limit is the page size, at most 100
	Limit uint64
}

// ConvertHistoryResponse is a page of conversions
type ConvertHistoryResponse struct {
	List []ConvertRecord `json:"list"`
}
