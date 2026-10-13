package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FiatTradingPairListResponse lists the fiat and crypto coins of fiat conversions
type FiatTradingPairListResponse struct {
	Fiats   []FiatConvertCoin `json:"fiats"`
	Cryptos []FiatConvertCoin `json:"cryptos"`
}

// FiatConvertCoin describes a coin of fiat conversions
type FiatConvertCoin struct {
	Coin      currency.Code `json:"coin"`
	FullName  string        `json:"fullName"`
	Icon      string        `json:"icon"`
	IconNight string        `json:"iconNight"`
	Precision uint64        `json:"precision"`
	// Disable is true when the coin cannot be used
	Disable bool `json:"disable"`
	// SingleFromMinLimit and SingleFromMaxLimit bound the amount sold in one conversion: fiat when buying, crypto when
	// selling
	SingleFromMinLimit types.Number `json:"singleFromMinLimit"`
	SingleFromMaxLimit types.Number `json:"singleFromMaxLimit"`
}

// FiatReferencePriceResponse holds a fiat and crypto pair's reference prices
type FiatReferencePriceResponse struct {
	Symbol    string                   `json:"symbol"`
	Fiat      currency.Code            `json:"fiat"`
	Crypto    currency.Code            `json:"crypto"`
	Timestamp types.Time               `json:"timestamp"`
	Buys      []FiatReferencePriceItem `json:"buys"`
	Sells     []FiatReferencePriceItem `json:"sells"`
}

// FiatReferencePriceItem is the reference price of one payment method
type FiatReferencePriceItem struct {
	// UnitPrice is the fiat price of one crypto coin
	UnitPrice     types.Number `json:"unitPrice"`
	PaymentMethod string       `json:"paymentMethod"`
}

// FiatQuoteRequest holds the fiat Request a Quote parameters; all but RequestCoinType are required
type FiatQuoteRequest struct {
	FromCoin currency.Code
	// FromCoinType and ToCoinType are fiat or crypto
	FromCoinType string
	ToCoin       currency.Code
	ToCoinType   string
	// RequestAmount is the amount to sell
	RequestAmount float64
	// RequestCoinType is the type of the coin sold, fiat or crypto, fiat by default
	RequestCoinType string
}

// FiatQuoteResponse is a fiat conversion quote
type FiatQuoteResponse struct {
	QuoteTransactionID string        `json:"quoteTxId"`
	ExchangeRate       types.Number  `json:"exchangeRate"`
	FromCoin           currency.Code `json:"fromCoin"`
	FromCoinType       string        `json:"fromCoinType"`
	ToCoin             currency.Code `json:"toCoin"`
	ToCoinType         string        `json:"toCoinType"`
	FromAmount         types.Number  `json:"fromAmount"`
	ToAmount           types.Number  `json:"toAmount"`
	ExpiredTime        types.Time    `json:"expiredTime"`
}

// FiatConfirmQuoteRequest holds the fiat Confirm a Quote parameters; QuoteTransactionID and SubUserID are required
type FiatConfirmQuoteRequest struct {
	QuoteTransactionID string
	// SubUserID is the user's sub user ID at Bybit
	SubUserID string
	// WebhookURL, at most 256 characters, is called when the conversion succeeds or fails
	WebhookURL string
	// MerchantRequestID is an optional ID of at most 36 characters for tracking the conversion
	MerchantRequestID string
}

// FiatConfirmQuoteResponse identifies a confirmed fiat conversion
type FiatConfirmQuoteResponse struct {
	TradeNumber       string `json:"tradeNo"`
	MerchantRequestID string `json:"merchantRequestId"`
}

// FiatConvertStatusResponse is one fiat conversion
type FiatConvertStatusResponse FiatTrade

// FiatTrade is one fiat conversion
type FiatTrade struct {
	TradeNumber string `json:"tradeNo"`
	// Status is processing, success or failed
	Status             string        `json:"status"`
	QuoteTransactionID string        `json:"quoteTxId"`
	ExchangeRate       types.Number  `json:"exchangeRate"`
	FromCoin           currency.Code `json:"fromCoin"`
	FromCoinType       string        `json:"fromCoinType"`
	ToCoin             currency.Code `json:"toCoin"`
	ToCoinType         string        `json:"toCoinType"`
	FromAmount         types.Number  `json:"fromAmount"`
	ToAmount           types.Number  `json:"toAmount"`
	CreatedAt          types.Time    `json:"createdAt"`
	SubUserID          string        `json:"subUserId"`
}

// FiatConvertHistoryRequest holds the fiat Get Convert History filters
type FiatConvertHistoryRequest struct {
	// Index is the page number, starting at 1
	Index uint64
	// Limit is the page size, 20 to 100
	Limit     uint64
	StartTime time.Time
	EndTime   time.Time
}

// FiatBalances holds fiat balances. Get Balance documents one object but its example returns an array of all the
// balances, so both forms decode
type FiatBalances []FiatBalance

// UnmarshalJSON decodes an array of balances, or a single balance object as a one-balance list
func (f *FiatBalances) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '{' {
		var balance FiatBalance
		if err := json.Unmarshal(data, &balance); err != nil {
			return err
		}
		*f = FiatBalances{balance}
		return nil
	}
	var balances []FiatBalance
	if err := json.Unmarshal(data, &balances); err != nil {
		return err
	}
	*f = balances
	return nil
}

// FiatBalance is the balance of one fiat currency
type FiatBalance struct {
	TotalBalance types.Number `json:"totalBalance"`
	// Balance is the available balance
	Balance       types.Number  `json:"balance"`
	FrozenBalance types.Number  `json:"frozenBalance"`
	Currency      currency.Code `json:"currency"`
}

// FiatTradeNotification is the body Bybit posts to the webhook URL of a confirmed fiat conversion when it ends; the
// webhook is authenticated by IP whitelisting (Trade Notify)
type FiatTradeNotification struct {
	TradeNumber string `json:"tradeNo"`
	// Status is processing, success or failed
	Status             string        `json:"status"`
	QuoteTransactionID string        `json:"quoteTxId"`
	ExchangeRate       types.Number  `json:"exchangeRate"`
	FromCoin           currency.Code `json:"fromCoin"`
	FromCoinType       string        `json:"fromCoinType"`
	ToCoin             currency.Code `json:"toCoin"`
	ToCoinType         string        `json:"toCoinType"`
	FromAmount         types.Number  `json:"fromAmount"`
	ToAmount           types.Number  `json:"toAmount"`
	CreatedAt          types.Time    `json:"createdAt"`
}
