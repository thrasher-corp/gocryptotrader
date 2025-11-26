package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WalletAccountTypes
const (
	Funding  WalletAccountType = "eb_convert_funding"
	UTA      WalletAccountType = "eb_convert_uta"
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
	AccuracyLength     uint8         `json:"accuracyLength"`
	CoinType           string        `json:"coinType"`
	Balance            types.Number  `json:"balance"`
	BalanceInUSDT      types.Number  `json:"uBalance"`
	SingleFromMinLimit types.Number  `json:"singleFromMinLimit"` // The minimum amount of fromCoin per transaction
	SingleFromMaxLimit types.Number  `json:"singleFromMaxLimit"` // The maximum amount of fromCoin per transaction
	DisableFrom        bool          `json:"disableFrom"`        // true: the coin is disabled to be fromCoin, false: the coin is allowed to be fromCoin
	DisableTo          bool          `json:"disableTo"`          // true: the coin is disabled to be toCoin, false: the coin is allowed to be toCoin
	TimePeriod         int64         `json:"timePeriod"`
	SingleToMinLimit   types.Number  `json:"singleToMinLimit"`
	SingleToMaxLimit   types.Number  `json:"singleToMaxLimit"`
	DailyFromMinLimit  types.Number  `json:"dailyFromMinLimit"`
	DailyFromMaxLimit  types.Number  `json:"dailyFromMaxLimit"`
	DailyToMinLimit    types.Number  `json:"dailyToMinLimit"`
	DailyToMaxLimit    types.Number  `json:"dailyToMaxLimit"`
}

// RequestQuoteRequest holds the parameters for requesting a quote
type RequestQuoteRequest struct {
	RequestID     string            `json:"requestId,omitempty"`
	AccountType   WalletAccountType `json:"accountType"`
	FromCoin      currency.Code     `json:"fromCoin"`
	ToCoin        currency.Code     `json:"toCoin"`
	RequestCoin   currency.Code     `json:"requestCoin"` // Must be same as FromCoin
	RequestAmount types.Number      `json:"requestAmount"`
	FromCoinType  string            `json:"fromCoinType,omitempty"` // "crypto"
	ToCoinType    string            `json:"toCoinType,omitempty"`   // "crypto"
	ParamType     string            `json:"paramType,omitempty"`    // "opFrom", mainly used for API broker user
	ParamValue    string            `json:"paramValue,omitempty"`   // Broker ID, mainly used for API broker user
}

// RequestQuoteResponse represents a response for a request a quote
type RequestQuoteResponse struct {
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
	ExtendedTaxAndFee  json.RawMessage `json:"extTaxAndFee"` // Compliance-related field. Currently returns an empty array, which may be used in the future
}

// ConfirmQuoteResponse represents a response for confirming a quote
type ConfirmQuoteResponse struct {
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
	ExtendedInfo          json.RawMessage   `json:"extInfo"` // Reserved field, ignored for now
	ConvertRate           types.Number      `json:"convertRate"`
	CreatedAt             types.Time        `json:"createdAt"`
}

// ConvertHistoryResponse represents a response for conversion history
type ConvertHistoryResponse struct {
	AccountType           WalletAccountType           `json:"accountType"`
	ExchangeTransactionID string                      `json:"exchangeTxId"`
	UserID                string                      `json:"userId"`
	FromCoin              currency.Code               `json:"fromCoin"`
	FromCoinType          string                      `json:"fromCoinType"`
	FromAmount            types.Number                `json:"fromAmount"`
	ToCoin                currency.Code               `json:"toCoin"`
	ToCoinType            string                      `json:"toCoinType"`
	ToAmount              types.Number                `json:"toAmount"`
	ExchangeStatus        string                      `json:"exchangeStatus"`
	ExtendedInfo          ExtendedInfoHistoryResponse `json:"extInfo"`
	ConvertRate           types.Number                `json:"convertRate"`
	CreatedAt             types.Time                  `json:"createdAt"`
}

// ExtendedInfoHistoryResponse represents the extended information for conversion history
type ExtendedInfoHistoryResponse struct {
	ParamType  string `json:"paramType"`
	ParamValue string `json:"paramValue"`
}
