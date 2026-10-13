package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SmallBalanceCoinsResponse lists the small balance coins and the coins they can be converted to
type SmallBalanceCoinsResponse struct {
	SmallAssetCoins []SmallBalanceCoin `json:"smallAssetCoins"`
	// SupportToCoins lists the coins small balances can be converted to, such as MNT, USDT and USDC
	SupportToCoins []currency.Code `json:"supportToCoins"`
}

// SmallBalanceCoin describes converting one small balance coin. Get Small Balance Coins leaves the conversion fields
// empty, as reserved, and its fee fields null
type SmallBalanceCoin struct {
	FromCoin currency.Code `json:"fromCoin"`
	// SupportConvert is 1 when the coin can be converted and 2 when it cannot
	SupportConvert uint64 `json:"supportConvert"`
	// AvailableBalance may exceed the balance that can be converted
	AvailableBalance types.Number `json:"availableBalance"`
	// BaseValue is the balance valued in USDT
	BaseValue    types.Number            `json:"baseValue"`
	ToCoin       currency.Code           `json:"toCoin"`
	ToAmount     types.Number            `json:"toAmount"`
	ExchangeRate types.Number            `json:"exchangeRate"`
	FeeInfo      *SmallBalanceFeeInfo    `json:"feeInfo"`
	TaxFeeInfo   *SmallBalanceTaxFeeInfo `json:"taxFeeInfo"`
}

// SmallBalanceFeeInfo is a small balance conversion's exchange fee
type SmallBalanceFeeInfo struct {
	FeeCoin currency.Code `json:"feeCoin"`
	Amount  types.Number  `json:"amount"`
	FeeRate types.Number  `json:"feeRate"`
}

// SmallBalanceTaxFeeInfo is a small balance conversion's tax fee
type SmallBalanceTaxFeeInfo struct {
	TotalAmount types.Number  `json:"totalAmount"`
	FeeCoin     currency.Code `json:"feeCoin"`
	// TaxFeeItems are not documented yet
	TaxFeeItems []json.RawMessage `json:"taxFeeItems"`
}

// SmallBalanceQuoteRequest holds the small balance Request a Quote parameters, all of them required
type SmallBalanceQuoteRequest struct {
	// AccountType is the wallet; only eb_convert_uta is supported
	AccountType string
	// FromCoins lists up to 20 coins to convert
	FromCoins []currency.Code
	// ToCoin is MNT, USDT or USDC
	ToCoin currency.Code
}

// SmallBalanceQuoteResponse is a small balance conversion quote
type SmallBalanceQuoteResponse struct {
	QuoteID string            `json:"quoteId"`
	Result  SmallBalanceQuote `json:"result"`
}

// SmallBalanceQuote details a small balance conversion quote, which expires after 30 seconds
type SmallBalanceQuote struct {
	QuoteCreateTime types.Time             `json:"quoteCreateTime"`
	QuoteExpireTime types.Time             `json:"quoteExpireTime"`
	ExchangeCoins   []SmallBalanceCoin     `json:"exchangeCoins"`
	TotalFeeInfo    SmallBalanceFeeInfo    `json:"totalFeeInfo"`
	TotalTaxFeeInfo SmallBalanceTaxFeeInfo `json:"totalTaxFeeInfo"`
}

// SmallBalanceConfirmQuoteResponse is the status of a confirmed small balance quote
type SmallBalanceConfirmQuoteResponse struct {
	QuoteID string `json:"quoteId"`
	// ExchangeTransactionID is the quote ID
	ExchangeTransactionID string     `json:"exchangeTxId"`
	SubmitTime            types.Time `json:"submitTime"`
	// Status is init, processing, success, failure or partial_fulfillment
	Status  string `json:"status"`
	Message string `json:"msg"`
}

// SmallBalanceExchangeHistoryRequest holds the small balance Get Exchange History filters
type SmallBalanceExchangeHistoryRequest struct {
	// AccountType is eb_convert_uta or eb_convert_funding
	AccountType string
	// QuoteID takes priority over the other filters
	QuoteID   string
	StartTime time.Time
	EndTime   time.Time
	// Cursor is the page number
	Cursor uint64
	// Size is the page size, 50 by default and at most 100
	Size uint64
}

// SmallBalanceExchangeHistoryResponse is a page of small balance conversions
type SmallBalanceExchangeHistoryResponse struct {
	// Cursor is the page number
	Cursor     types.Number                 `json:"cursor"`
	Size       types.Number                 `json:"size"`
	LastPage   types.Number                 `json:"lastPage"`
	TotalCount types.Number                 `json:"totalCount"`
	Records    []SmallBalanceExchangeRecord `json:"records"`
}

// SmallBalanceExchangeRecord is one small balance conversion
type SmallBalanceExchangeRecord struct {
	// AccountType is eb_convert_uta for the Unified wallet or eb_convert_funding for the Funding wallet
	AccountType           string                          `json:"accountType"`
	ExchangeTransactionID string                          `json:"exchangeTxId"`
	ToCoin                currency.Code                   `json:"toCoin"`
	ToAmount              types.Number                    `json:"toAmount"`
	SubRecords            []SmallBalanceExchangeSubRecord `json:"subRecords"`
	// Status is init, processing, success, failure or partial_fulfillment
	Status    string     `json:"status"`
	CreatedAt types.Time `json:"createdAt"`
	// ExchangeSource is small_asset_uta or small_asset_funding
	ExchangeSource  string                 `json:"exchangeSource"`
	FeeCoin         currency.Code          `json:"feeCoin"`
	TotalFeeAmount  types.Number           `json:"totalFeeAmount"`
	TotalTaxFeeInfo SmallBalanceTaxFeeInfo `json:"totalTaxFeeInfo"`
}

// SmallBalanceExchangeSubRecord is the conversion of one coin within a small balance conversion
type SmallBalanceExchangeSubRecord struct {
	FromCoin   currency.Code `json:"fromCoin"`
	FromAmount types.Number  `json:"fromAmount"`
	ToCoin     currency.Code `json:"toCoin"`
	ToAmount   types.Number  `json:"toAmount"`
	FeeCoin    currency.Code `json:"feeCoin"`
	FeeAmount  types.Number  `json:"feeAmount"`
	// Status is init, processing, success, failure or partial_fulfillment
	Status     string                 `json:"status"`
	TaxFeeInfo SmallBalanceTaxFeeInfo `json:"taxFeeInfo"`
}
