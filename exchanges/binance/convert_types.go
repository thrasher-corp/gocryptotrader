package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ConvertPair is a convertible pair with the amount limits of each side; a maximum of 9E+24 means no practical limit
type ConvertPair struct {
	FromAsset          currency.Code `json:"fromAsset"`
	ToAsset            currency.Code `json:"toAsset"`
	FromAssetMinAmount types.Number  `json:"fromAssetMinAmount"`
	FromAssetMaxAmount types.Number  `json:"fromAssetMaxAmount"`
	ToAssetMinAmount   types.Number  `json:"toAssetMinAmount"`
	ToAssetMaxAmount   types.Number  `json:"toAssetMaxAmount"`
	// FromIsBase is returned although the schema omits it; Place limit order needs it to tell the base asset
	FromIsBase bool `json:"fromIsBase"`
}

// OrderQuantityPrecision is the number of decimal places an asset's convert order quantity accepts
type OrderQuantityPrecision struct {
	Asset    currency.Code `json:"asset"`
	Fraction uint64        `json:"fraction"`
}

// AcceptQuoteResponse is the order an accepted quote created
type AcceptQuoteResponse struct {
	OrderID     string     `json:"orderId"`
	CreateTime  types.Time `json:"createTime"`
	OrderStatus string     `json:"orderStatus"`
}

// ConvertLimitOrderResponse is a convert limit order's ID and status after placing or cancelling it
type ConvertLimitOrderResponse struct {
	OrderID uint64 `json:"orderId"`
	Status  string `json:"status"`
}

// ConvertTradeHistoryResponse is a page of convert trades
type ConvertTradeHistoryResponse struct {
	List      []ConvertTrade `json:"list"`
	StartTime types.Time     `json:"startTime"`
	EndTime   types.Time     `json:"endTime"`
	Limit     uint64         `json:"limit"`
	MoreData  bool           `json:"moreData"`
}

// ConvertTrade is a completed convert trade
type ConvertTrade struct {
	QuoteID      string        `json:"quoteId"`
	OrderID      uint64        `json:"orderId"`
	OrderStatus  string        `json:"orderStatus"`
	FromAsset    currency.Code `json:"fromAsset"`
	FromAmount   types.Number  `json:"fromAmount"`
	ToAsset      currency.Code `json:"toAsset"`
	ToAmount     types.Number  `json:"toAmount"`
	Ratio        types.Number  `json:"ratio"`
	InverseRatio types.Number  `json:"inverseRatio"`
	CreateTime   types.Time    `json:"createTime"`
}

// ConvertOrderStatusResponse is a convert order's status
type ConvertOrderStatusResponse struct {
	OrderID      uint64        `json:"orderId"`
	OrderStatus  string        `json:"orderStatus"`
	FromAsset    currency.Code `json:"fromAsset"`
	FromAmount   types.Number  `json:"fromAmount"`
	ToAsset      currency.Code `json:"toAsset"`
	ToAmount     types.Number  `json:"toAmount"`
	Ratio        types.Number  `json:"ratio"`
	InverseRatio types.Number  `json:"inverseRatio"`
	CreateTime   types.Time    `json:"createTime"`
}

// ConvertLimitOrderRequest holds the parameters of a convert limit order
type ConvertLimitOrderRequest struct {
	// BaseAsset is the side ConvertPair.FromIsBase names the base
	BaseAsset  currency.Code
	QuoteAsset currency.Code
	// LimitPrice is the price from BaseAsset to QuoteAsset
	LimitPrice float64
	// Exactly one of BaseAmount and QuoteAmount is required
	BaseAmount  float64
	QuoteAmount float64
	// Side is BUY or SELL
	Side string
	// WalletType is SPOT (default), FUNDING, EARN or a combination such as SPOT_FUNDING
	WalletType string
	// ExpiredType is 1_D, 3_D, 7_D or 30_D
	ExpiredType string
}

// ConvertLimitOpenOrdersResponse holds the open convert limit orders
type ConvertLimitOpenOrdersResponse struct {
	List []ConvertLimitOrder `json:"list"`
}

// ConvertLimitOrder is an open convert limit order
type ConvertLimitOrder struct {
	QuoteID          string        `json:"quoteId"`
	OrderID          uint64        `json:"orderId"`
	OrderStatus      string        `json:"orderStatus"`
	FromAsset        currency.Code `json:"fromAsset"`
	FromAmount       types.Number  `json:"fromAmount"`
	ToAsset          currency.Code `json:"toAsset"`
	ToAmount         types.Number  `json:"toAmount"`
	Ratio            types.Number  `json:"ratio"`
	InverseRatio     types.Number  `json:"inverseRatio"`
	CreateTime       types.Time    `json:"createTime"`
	ExpiredTimestamp types.Time    `json:"expiredTimestamp"`
}

// ConvertQuoteRequest holds the parameters of a convert quote request
type ConvertQuoteRequest struct {
	FromAsset currency.Code
	ToAsset   currency.Code
	// Exactly one of FromAmount, the amount debited, and ToAmount, the amount credited, is required
	FromAmount float64
	ToAmount   float64
	// WalletType is SPOT (default), FUNDING, EARN or a combination such as SPOT_FUNDING
	WalletType string
	// ValidTime is how long the quote is valid: 10s (default), 30s or 1m
	ValidTime string
}

// ConvertQuoteResponse is a convert quote; QuoteID is empty when the account cannot fund the conversion
type ConvertQuoteResponse struct {
	QuoteID        string       `json:"quoteId"`
	Ratio          types.Number `json:"ratio"`
	InverseRatio   types.Number `json:"inverseRatio"`
	ValidTimestamp types.Time   `json:"validTimestamp"`
	ToAmount       types.Number `json:"toAmount"`
	FromAmount     types.Number `json:"fromAmount"`
}
