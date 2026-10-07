package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CancelAlgoOrderResponse is the result of cancelling an algo order
type CancelAlgoOrderResponse struct {
	AlgoID  uint64 `json:"algoId"`
	Success bool   `json:"success"`
	Code    int64  `json:"code"` // Signed because Binance error codes are negative
	Message string `json:"msg"`
}

// FuturesAlgoOrdersResponse is a page of futures algo orders
type FuturesAlgoOrdersResponse struct {
	Total  uint64             `json:"total"`
	Orders []FuturesAlgoOrder `json:"orders"`
}

// FuturesAlgoOrder is a futures algo order
type FuturesAlgoOrder struct {
	AlgoID           uint64       `json:"algoId"`
	Symbol           string       `json:"symbol"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	TotalQuantity    types.Number `json:"totalQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	ExecutedAmount   types.Number `json:"executedAmt"`
	AveragePrice     types.Number `json:"avgPrice"`
	ClientAlgoID     string       `json:"clientAlgoId"`
	BookTime         types.Time   `json:"bookTime"`
	EndTime          types.Time   `json:"endTime"`
	AlgoStatus       string       `json:"algoStatus"`
	AlgoType         string       `json:"algoType"`
	Urgency          string       `json:"urgency"`
}

// AlgoHistoricalOrdersRequest holds the parameters of the futures and spot historical algo order queries
type AlgoHistoricalOrdersRequest struct {
	Symbol currency.Pair
	// Side is BUY or SELL
	Side      string
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	// PageSize is between 1 and 100
	PageSize uint64
}

// AlgoSubOrdersResponse is a page of an algo order's sub orders
type AlgoSubOrdersResponse struct {
	Total            uint64         `json:"total"`
	ExecutedQuantity types.Number   `json:"executedQty"`
	ExecutedAmount   types.Number   `json:"executedAmt"`
	SubOrders        []AlgoSubOrder `json:"subOrders"`
}

// AlgoSubOrder is an order an algo order placed
type AlgoSubOrder struct {
	AlgoID           uint64        `json:"algoId"`
	OrderID          uint64        `json:"orderId"`
	OrderStatus      string        `json:"orderStatus"`
	ExecutedQuantity types.Number  `json:"executedQty"`
	ExecutedAmount   types.Number  `json:"executedAmt"`
	FeeAmount        types.Number  `json:"feeAmt"`
	FeeAsset         currency.Code `json:"feeAsset"`
	BookTime         types.Time    `json:"bookTime"`
	AveragePrice     types.Number  `json:"avgPrice"`
	Side             string        `json:"side"`
	Symbol           string        `json:"symbol"`
	SubID            uint64        `json:"subId"` // Execution sequence of the sub order
	TimeInForce      string        `json:"timeInForce"`
	OriginalQuantity types.Number  `json:"origQty"`
}

// FuturesTWAPOrderRequest holds the parameters of a futures TWAP order
type FuturesTWAPOrderRequest struct {
	Symbol currency.Pair
	// Side is BUY or SELL
	Side string
	// PositionSide is BOTH (default) in one-way mode, and LONG or SHORT, which it requires, in hedge mode
	PositionSide string
	Quantity     float64
	// Duration is sent in whole seconds and must be between 5 minutes and 24 hours
	Duration time.Duration
	// ClientAlgoID must be 32 characters long; the API generates one when it is not sent
	ClientAlgoID string
	// ReduceOnly cannot be sent in hedge mode or when opening a position
	ReduceOnly bool
	// LimitPrice is the order's limit price; the order executes at market price without it
	LimitPrice float64
}

// AlgoOrderResponse is the result of placing an algo order
type AlgoOrderResponse struct {
	ClientAlgoID string `json:"clientAlgoId"`
	Success      bool   `json:"success"`
	Code         int64  `json:"code"` // Signed because Binance error codes are negative
	Message      string `json:"msg"`
}

// VolumeParticipationOrderRequest holds the parameters of a futures volume participation order
type VolumeParticipationOrderRequest struct {
	Symbol currency.Pair
	// Side is BUY or SELL
	Side string
	// PositionSide is BOTH (default) in one-way mode, and LONG or SHORT, which it requires, in hedge mode
	PositionSide string
	Quantity     float64
	// Urgency is the relative speed of execution: LOW, MEDIUM or HIGH
	Urgency string
	// ClientAlgoID must be 32 characters long; the API generates one when it is not sent
	ClientAlgoID string
	// ReduceOnly cannot be sent in hedge mode or when opening a position
	ReduceOnly bool
	// LimitPrice is the order's limit price; the order executes at market price without it
	LimitPrice float64
}

// SpotAlgoOrdersResponse is a page of spot algo orders
type SpotAlgoOrdersResponse struct {
	Total  uint64          `json:"total"`
	Orders []SpotAlgoOrder `json:"orders"`
}

// SpotAlgoOrder is a spot algo order
type SpotAlgoOrder struct {
	AlgoID           uint64       `json:"algoId"`
	Symbol           string       `json:"symbol"`
	Side             string       `json:"side"`
	TotalQuantity    types.Number `json:"totalQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	ExecutedAmount   types.Number `json:"executedAmt"`
	AveragePrice     types.Number `json:"avgPrice"`
	ClientAlgoID     string       `json:"clientAlgoId"`
	BookTime         types.Time   `json:"bookTime"`
	EndTime          types.Time   `json:"endTime"`
	AlgoStatus       string       `json:"algoStatus"`
	AlgoType         string       `json:"algoType"`
	Urgency          string       `json:"urgency"`
}

// SpotTWAPOrderRequest holds the parameters of a spot TWAP order
type SpotTWAPOrderRequest struct {
	Symbol currency.Pair
	// Side is BUY or SELL
	Side     string
	Quantity float64
	// Duration is sent in whole seconds and must be between 5 minutes and 24 hours
	Duration time.Duration
	// ClientAlgoID must be 32 characters long; the API generates one when it is not sent
	ClientAlgoID string
	// LimitPrice is the order's limit price; the order executes at market price without it
	LimitPrice float64
}
