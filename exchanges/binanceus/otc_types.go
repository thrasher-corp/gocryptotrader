package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CoinPairResponse holds a coin pair over-the-counter orders convert between, and the amounts of each coin an order may
// convert
type CoinPairResponse struct {
	FromCoin              currency.Code `json:"fromCoin"`
	ToCoin                currency.Code `json:"toCoin"`
	FromCoinMinimumAmount types.Number  `json:"fromCoinMinAmount"`
	FromCoinMaximumAmount types.Number  `json:"fromCoinMaxAmount"`
	ToCoinMinimumAmount   types.Number  `json:"toCoinMinAmount"`
	ToCoinMaximumAmount   types.Number  `json:"toCoinMaxAmount"`
}

// QuoteRequest holds the parameters of a request for an over-the-counter quote
type QuoteRequest struct {
	FromCoin      currency.Code
	ToCoin        currency.Code
	RequestCoin   currency.Code // The coin RequestAmount is an amount of
	RequestAmount float64
}

// QuoteResponse holds an over-the-counter quote. QuoteID is sent only when the account holds enough of the coin the
// quote converts from to settle it; a quote without one is informational and cannot be placed
type QuoteResponse struct {
	QuoteID        string       `json:"quoteId"`
	Symbol         string       `json:"symbol"`
	Ratio          types.Number `json:"ratio"`          // Amount of the coin converted to for each unit converted from
	InverseRatio   types.Number `json:"inverseRatio"`   // Amount of the coin converted from for each unit converted to
	ValidTimestamp types.Time   `json:"validTimestamp"` // When the quote expires
	FromAmount     types.Number `json:"fromAmount"`
	ToAmount       types.Number `json:"toAmount"`
}

// PlaceOverTheCounterTradeOrderResponse holds an over-the-counter order as placed
type PlaceOverTheCounterTradeOrderResponse struct {
	OrderID     string                    `json:"orderId"`
	CreateTime  types.Time                `json:"createTime"`
	OrderStatus OverTheCounterOrderStatus `json:"orderStatus"` // Always PROCESS, as an order settles after it is placed
}

// OverTheCounterTradeOrderResponse holds an over-the-counter order
type OverTheCounterTradeOrderResponse struct {
	QuoteID      string                    `json:"quoteId"`
	OrderID      string                    `json:"orderId"`
	OrderStatus  OverTheCounterOrderStatus `json:"orderStatus"`
	FromCoin     currency.Code             `json:"fromCoin"`
	FromAmount   types.Number              `json:"fromAmount"`
	ToCoin       currency.Code             `json:"toCoin"`
	ToAmount     types.Number              `json:"toAmount"`
	Ratio        types.Number              `json:"ratio"`
	InverseRatio types.Number              `json:"inverseRatio"`
	CreateTime   types.Time                `json:"createTime"`
}

// OverTheCounterTradeOrdersRequest holds the optional filters of an over-the-counter orders request
type OverTheCounterTradeOrdersRequest struct {
	OrderID   string
	FromCoin  currency.Code
	ToCoin    currency.Code
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64 // Orders a page holds: 10 when zero, at most 100
}

// OverTheCounterTradeOrdersResponse holds a page of over-the-counter orders and the number of orders matching the
// request
type OverTheCounterTradeOrdersResponse struct {
	Total uint64                             `json:"total"`
	Rows  []OverTheCounterTradeOrderResponse `json:"rows"`
}

// OCBSTradeOrdersRequest holds the optional filters of an OCBS orders request
type OCBSTradeOrdersRequest struct {
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64 // Orders a page holds: 10 when zero, at most 100
}

// OCBSTradeOrdersResponse holds a page of OCBS orders and the number of orders matching the request
type OCBSTradeOrdersResponse struct {
	Total    uint64           `json:"total"`
	DataList []OCBSTradeOrder `json:"dataList"`
}

// OCBSTradeOrder holds an OCBS order, which buys crypto with fiat or sells it for fiat
type OCBSTradeOrder struct {
	QuoteID     string                    `json:"quoteId"`
	OrderID     string                    `json:"orderId"`
	OrderStatus OverTheCounterOrderStatus `json:"orderStatus"`
	FromCoin    currency.Code             `json:"fromCoin"` // Fiat for a buy and crypto for a sell
	FromAmount  types.Number              `json:"fromAmount"`
	ToCoin      currency.Code             `json:"toCoin"` // Crypto for a buy and fiat for a sell
	ToAmount    types.Number              `json:"toAmount"`
	FeeCoin     currency.Code             `json:"feeCoin"`
	FeeAmount   types.Number              `json:"feeAmount"` // The trading fee and the payment rail's fee
	Ratio       types.Number              `json:"ratio"`
	CreateTime  types.Time                `json:"createTime"`
}
