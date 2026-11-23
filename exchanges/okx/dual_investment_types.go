package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DualInvestmentCurrencyPairResponse holds a currency pair and option type dual investment products are offered on
type DualInvestmentCurrencyPairResponse struct {
	BaseCurrency  currency.Code `json:"baseCcy"`
	QuoteCurrency currency.Code `json:"quoteCcy"`
	OptionType    string        `json:"optType"`
	Underlying    string        `json:"uly"`
}

// DualInvestmentProductsRequest holds the currency pair and option type of the dual investment products to return
type DualInvestmentProductsRequest struct {
	BaseCurrency  currency.Code
	QuoteCurrency currency.Code
	OptionType    string
}

// DualInvestmentProductResponse holds a dual investment product
type DualInvestmentProductResponse struct {
	AbsoluteYield       types.Number  `json:"absYield"`
	AnnualisedYield     types.Number  `json:"annualizedYield"`
	BaseCurrency        currency.Code `json:"baseCcy"`
	QuoteCurrency       currency.Code `json:"quoteCcy"`
	NotionalCurrency    currency.Code `json:"notionalCcy"`
	ExpiryTime          types.Time    `json:"expTime"`
	InterestAccrualTime types.Time    `json:"interestAccrualTime"`
	ListTime            types.Time    `json:"listTime"`
	MinimumSize         types.Number  `json:"minSize"`
	MaximumSize         types.Number  `json:"maxSize"`
	OptionType          string        `json:"optType"`
	ProductID           string        `json:"productId"`
	QuoteTime           types.Time    `json:"quoteTime"`
	RedeemStartTime     types.Time    `json:"redeemStartTime"`
	RedeemEndTime       types.Time    `json:"redeemEndTime"`
	StepSize            types.Number  `json:"stepSz"`
	TradeEndTime        types.Time    `json:"tradeEndTime"`
	Underlying          string        `json:"uly"`
	StrikePrice         types.Number  `json:"strike"`
}

// DualInvestmentQuoteRequest holds a dual investment quote request
type DualInvestmentQuoteRequest struct {
	ProductID        string        `json:"productId"`
	NotionalSize     float64       `json:"notionalSz,string"`
	NotionalCurrency currency.Code `json:"notionalCcy"`
}

// DualInvestmentQuoteResponse holds a dual investment quote
type DualInvestmentQuoteResponse struct {
	AbsoluteYield       types.Number  `json:"absYield"`
	AnnualisedYield     types.Number  `json:"annualizedYield"`
	InterestAccrualTime types.Time    `json:"interestAccrualTime"`
	NotionalSize        types.Number  `json:"notionalSz"`
	NotionalCurrency    currency.Code `json:"notionalCcy"`
	ProductID           string        `json:"productId"`
	QuoteID             string        `json:"quoteId"`
	ValidUntil          types.Time    `json:"validUntil"`
	IndexPrice          types.Number  `json:"idxPx"`
}

// DualInvestmentTradeRequest holds the quote a dual investment order trades on
type DualInvestmentTradeRequest struct {
	QuoteID string `json:"quoteId"`
}

// DualInvestmentTradeResponse holds a dual investment order placed on a quote
type DualInvestmentTradeResponse struct {
	QuoteID string `json:"quoteId"`
	OrderID string `json:"ordId"`
	State   string `json:"state"`
}

// DualInvestmentRedeemQuoteRequest holds the dual investment order to quote an early redemption for
type DualInvestmentRedeemQuoteRequest struct {
	OrderID string `json:"ordId"`
}

// DualInvestmentRedeemQuoteResponse holds a dual investment early redemption quote
type DualInvestmentRedeemQuoteResponse struct {
	OrderID        string        `json:"ordId"`
	QuoteID        string        `json:"quoteId"`
	RedeemSize     types.Number  `json:"redeemSz"`
	RedeemCurrency currency.Code `json:"redeemCcy"`
	TermRate       types.Number  `json:"termRate"`
	ValidUntil     types.Time    `json:"validUntil"`
}

// DualInvestmentRedeemRequest holds a dual investment early redemption on a redeem quote
type DualInvestmentRedeemRequest struct {
	OrderID string `json:"ordId"`
	QuoteID string `json:"quoteId"`
}

// DualInvestmentRedeemResponse holds a dual investment order redeeming early
type DualInvestmentRedeemResponse struct {
	OrderID string `json:"ordId"`
	State   string `json:"state"`
}

// DualInvestmentOrderStateResponse holds a dual investment order's state
type DualInvestmentOrderStateResponse struct {
	OrderID string `json:"ordId"`
	State   string `json:"state"`
}

// DualInvestmentOrderHistoryRequest holds the filters, order ID cursors and time window for dual investment order
// history
type DualInvestmentOrderHistoryRequest struct {
	OrderID    string
	ProductID  string
	Underlying string
	State      string
	// BeginID returns orders newer than this order ID
	BeginID string
	// EndID returns orders earlier than this order ID
	EndID string
	Begin time.Time
	End   time.Time
	Limit uint64
}

// DualInvestmentOrderHistoryResponse holds a dual investment order
type DualInvestmentOrderHistoryResponse struct {
	OrderID            string        `json:"ordId"`
	QuoteID            string        `json:"quoteId"`
	State              string        `json:"state"`
	ProductID          string        `json:"productId"`
	BaseCurrency       currency.Code `json:"baseCcy"`
	QuoteCurrency      currency.Code `json:"quoteCcy"`
	Underlying         string        `json:"uly"`
	StrikePrice        types.Number  `json:"strike"`
	NotionalSize       types.Number  `json:"notionalSz"`
	NotionalCurrency   currency.Code `json:"notionalCcy"`
	AbsoluteYield      types.Number  `json:"absYield"`
	AnnualisedYield    types.Number  `json:"annualizedYield"`
	YieldSize          types.Number  `json:"yieldSz"`
	YieldCurrency      currency.Code `json:"yieldCcy"`
	SettlementSize     types.Number  `json:"settleSz"`
	SettlementCurrency currency.Code `json:"settleCcy"`
	SettlementPrice    types.Number  `json:"settlePx"`
	ExpiryTime         types.Time    `json:"expTime"`
	SettlementTime     types.Time    `json:"settleTime"`
	RedeemStartTime    types.Time    `json:"redeemStartTime"`
	RedeemEndTime      types.Time    `json:"redeemEndTime"`
	CreationTime       types.Time    `json:"cTime"`
	UpdateTime         types.Time    `json:"uTime"`
}
