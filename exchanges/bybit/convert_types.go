package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WalletAccountTypes
const (
	Funding  WalletAccountType = "eb_convert_funding"
	Uta      WalletAccountType = "eb_convert_uta"
	Spot     WalletAccountType = "eb_convert_spot"
	Contract WalletAccountType = "eb_convert_contract"
	Inverse  WalletAccountType = "eb_convert_inverse"
)

// WalletAccountType represents the different types of wallet accounts
type WalletAccountType string

// ConvertCoinResponse represents a coin that can be converted
type ConvertCoinResponse struct {
	Coin               currency.Code `json:"coin"`
	FullName           string        `json:"fullName"`
	Icon               string        `json:"icon"`
	IconNight          string        `json:"iconNight"`
	AccuracyLength     int64         `json:"accuracyLength"`
	CoinType           string        `json:"coinType"`
	Balance            types.Number  `json:"balance"`
	BalanceInUSDT      types.Number  `json:"uBalance"`
	SingleFromMinLimit types.Number  `json:"singleFromMinLimit"` // The minimum amount of fromCoin per transaction
	SingleFromMaxLimit types.Number  `json:"singleFromMaxLimit"` // The maximum amount of fromCoin per transaction
	DisableFrom        bool          `json:"disableFrom"`        // true: the coin is disabled to be fromCoin, false: the coin is allowed to be fromCoin
	DisableTo          bool          `json:"disableTo"`          // true: the coin is disabled to be toCoin, false: the coin is allowed to be toCoin
	// Reserved fields, ignored for now
	TimePeriod        int64        `json:"timePeriod"`
	SingleToMinLimit  types.Number `json:"singleToMinLimit"`
	SingleToMaxLimit  types.Number `json:"singleToMaxLimit"`
	DailyFromMinLimit types.Number `json:"dailyFromMinLimit"`
	DailyFromMaxLimit types.Number `json:"dailyFromMaxLimit"`
	DailyToMinLimit   types.Number `json:"dailyToMinLimit"`
	DailyToMaxLimit   types.Number `json:"dailyToMaxLimit"`
}

// RequestAQuoteRequest holds the parameters for requesting a quote
type RequestAQuoteRequest struct {
	// Required fields
	AccountType WalletAccountType
	From        currency.Code // Convert from coin (coin to sell)
	To          currency.Code // Convert to coin (coin to buy)
	Amount      float64       // Convert amount
	// Optional fields
	RequestCoin  currency.Code // This will default to FromCoin
	FromCoinType string        // "crypto"
	ToCoinType   string        // "crypto"
	ParamType    string        // "opFrom", mainly used for API broker user
	ParamValue   string        // Broker ID, mainly used for API broker user
	RequestID    string
}

// RequestAQuoteResponse represents a response for a request a quote
type RequestAQuoteResponse struct {
	QuoteTransactionID string          `json:"quoteTxId"` // Quote transaction ID. It is system generated, and it is used to confirm quote and query the result of transaction
	ExchangeRate       types.Number    `json:"exchangeRate"`
	FromCoin           currency.Code   `json:"fromCoin"`
	FromCoinType       string          `json:"fromCoinType"`
	ToCoin             currency.Code   `json:"toCoin"`
	ToCoinType         string          `json:"toCoinType"`
	FromAmount         types.Number    `json:"fromAmount"`
	ToAmount           types.Number    `json:"toAmount"`
	ExpiredTime        types.Time      `json:"expiredTime"` // The expiry time for this quote (15 seconds)
	RequestID          string          `json:"requestId"`
	ExtTaxAndFee       json.RawMessage `json:"extTaxAndFee"` // Compliance-related field. Currently returns an empty array, which may be used in the future
}

// ConfirmAQuoteResponse represents a response for confirming a quote
type ConfirmAQuoteResponse struct {
	ExchangeStatus     string `json:"exchangeStatus"`
	QuoteTransactionID string `json:"quoteTxId"`
}

// ConvertStatusResponse represents the response for a conversion status query
type ConvertStatusResponse struct {
	AccountType           WalletAccountType `json:"accountType"`
	ExchangeTransactionID string            `json:"exchangeTxId"`
	UserID                string            `json:"userId"`
	FromCoin              currency.Code     `json:"fromCoin"`
	FromCoinType          string            `json:"fromCoinType"`
	FromAmount            types.Number      `json:"fromAmount"`
	ToCoin                currency.Code     `json:"toCoin"`
	ToCoinType            string            `json:"toCoinType"`
	ToAmount              types.Number      `json:"toAmount"`
	ExchangeStatus        string            `json:"exchangeStatus"`
	ExtInfo               json.RawMessage   `json:"extInfo"` // Reserved field, ignored for now
	ConvertRate           types.Number      `json:"convertRate"`
	CreatedAt             types.Time        `json:"createdAt"`
}

// ConvertHistoryResponse represents a response for conversion history
type ConvertHistoryResponse struct {
	AccountType           WalletAccountType `json:"accountType"`
	ExchangeTransactionID string            `json:"exchangeTxId"`
	UserID                string            `json:"userId"`
	FromCoin              currency.Code     `json:"fromCoin"`
	FromCoinType          string            `json:"fromCoinType"`
	FromAmount            types.Number      `json:"fromAmount"`
	ToCoin                currency.Code     `json:"toCoin"`
	ToCoinType            string            `json:"toCoinType"`
	ToAmount              types.Number      `json:"toAmount"`
	ExchangeStatus        string            `json:"exchangeStatus"`
	ExtInfo               json.RawMessage   `json:"extInfo"`
	ConvertRate           types.Number      `json:"convertRate"`
	CreatedAt             types.Time        `json:"createdAt"`
}
