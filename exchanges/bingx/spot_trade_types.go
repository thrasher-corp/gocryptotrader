package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// spotBatchOrdersLimit is the most orders a spot batch order request takes
const spotBatchOrdersLimit = 5

// PlaceSpotOrderRequest holds the parameters of a spot order
type PlaceSpotOrderRequest struct {
	Symbol currency.Pair
	Side   Side
	Type   OrderType
	// StopPrice triggers TAKE_STOP_LIMIT, TAKE_STOP_MARKET, TRIGGER_LIMIT and TRIGGER_MARKET orders
	StopPrice float64
	Quantity  float64
	// QuoteOrderQuantity is the order's amount in the quote currency; BingX uses Quantity when both are set
	QuoteOrderQuantity float64
	Price              float64
	NewClientOrderID   string // Letters, digits and underscores, 1 to 40 characters
	TimeInForce        TimeInForce
}

// SpotOrderResponse holds a spot order as the order placement and cancellation endpoints report it
type SpotOrderResponse struct {
	Symbol                   string          `json:"symbol"`
	OrderID                  NumericID       `json:"orderId"`
	TransactTime             types.Time      `json:"transactTime"`
	Price                    types.Number    `json:"price"`
	StopPrice                types.Number    `json:"stopPrice"`
	OriginalQuantity         types.Number    `json:"origQty"`
	ExecutedQuantity         types.Number    `json:"executedQty"`
	CummulativeQuoteQuantity types.Number    `json:"cummulativeQuoteQty"`
	Status                   SpotOrderStatus `json:"status"`
	Type                     OrderType       `json:"type"`
	Side                     Side            `json:"side"`
	ClientOrderID            string          `json:"clientOrderID"`
	CancelRestrictions       SpotOrderStatus `json:"cancelRestrictions"` // Documented for cancel-replace orders
}

// PlaceSpotBatchOrdersRequest holds the orders of a spot batch order request
type PlaceSpotBatchOrdersRequest struct {
	Orders []PlaceSpotOrderRequest // At most five
	// Sync places the orders one after another, so they may differ in symbol, side and type; otherwise they are placed
	// in parallel and must share them
	Sync bool
}

// SpotBatchOrderItemRequest is an order in the data JSON array of a spot batch order request
type SpotBatchOrderItemRequest struct {
	Symbol             string      `json:"symbol"`
	Side               Side        `json:"side"`
	Type               OrderType   `json:"type"`
	StopPrice          float64     `json:"stopPrice,omitempty"`
	Quantity           float64     `json:"quantity,omitempty"`
	QuoteOrderQuantity float64     `json:"quoteOrderQty,omitempty"`
	Price              float64     `json:"price,omitempty"`
	NewClientOrderID   string      `json:"newClientOrderId,omitempty"`
	TimeInForce        TimeInForce `json:"timeInForce,omitempty"`
}

// SpotOrdersResponse holds a list of spot orders as the order placement and cancellation endpoints report them
type SpotOrdersResponse struct {
	Orders []SpotOrderResponse `json:"orders"`
}

// CancelSpotOrderRequest holds the parameters of a spot order cancellation
type CancelSpotOrderRequest struct {
	Symbol  currency.Pair
	OrderID uint64
	// ClientOrderID finds orders placed within the last two hours
	ClientOrderID string
	// CancelRestrictions cancels the order only while it is NEW, PENDING or PARTIALLY_FILLED
	CancelRestrictions SpotOrderStatus
}

// CancelSpotBatchOrdersRequest holds the parameters of a spot batch order cancellation
type CancelSpotBatchOrdersRequest struct {
	Symbol currency.Pair
	// Process cancels the orders that can be cancelled and reports the rest as failures; without it, one order that
	// cannot be cancelled fails them all
	Process        bool
	OrderIDs       []uint64
	ClientOrderIDs []string
}

// CancelSpotBatchOrdersResponse holds the outcome of a spot batch order cancellation
type CancelSpotBatchOrdersResponse struct {
	Fails  []SpotCancelFailure `json:"fails"`
	Orders []SpotOrderResponse `json:"orders"`
}

// SpotCancelFailure holds an order a spot batch cancellation did not cancel, and why
type SpotCancelFailure struct {
	OrderID NumericID `json:"orderId"`
	Error   string    `json:"error"`
}

// CancelReplaceSpotOrderRequest holds the parameters of a spot cancel-replace, which cancels an order and places a new
// one on the same symbol
type CancelReplaceSpotOrderRequest struct {
	Symbol        currency.Pair
	CancelOrderID uint64
	// CancelClientOrderID finds orders placed within the last two hours
	CancelClientOrderID string
	// CancelRestrictions cancels the order only while it is NEW, PENDING or PARTIALLY_FILLED
	CancelRestrictions SpotOrderStatus
	CancelReplaceMode  CancelReplaceMode
	Side               Side
	Type               OrderType
	// StopPrice triggers TAKE_STOP_LIMIT, TAKE_STOP_MARKET, TRIGGER_LIMIT and TRIGGER_MARKET orders
	StopPrice          float64
	Quantity           float64
	QuoteOrderQuantity float64
	Price              float64
	NewClientOrderID   string
}

// CancelReplaceSpotOrderResponse holds the outcome of both halves of a spot cancel-replace
type CancelReplaceSpotOrderResponse struct {
	CancelResult        SpotCancelReplaceResult `json:"cancelResult"`
	OpenResult          SpotCancelReplaceResult `json:"openResult"`
	OrderOpenResponse   SpotOrderResponse       `json:"orderOpenResponse"`
	OrderCancelResponse SpotOrderResponse       `json:"orderCancelResponse"`
}

// SpotCancelReplaceResult holds the outcome of one half of a spot cancel-replace
type SpotCancelReplaceResult struct {
	Code    int64  `json:"code"` // An API error code, signed as APIError's
	Message string `json:"msg"`
	Result  bool   `json:"result"`
}

// SpotOrderDetailsRequest holds the parameters of a spot order query
type SpotOrderDetailsRequest struct {
	Symbol  currency.Pair
	OrderID uint64
	// ClientOrderID finds orders placed within the last two hours
	ClientOrderID string
}

// SpotOrderDetailResponse holds a spot order's details
type SpotOrderDetailResponse struct {
	Symbol                     string          `json:"symbol"`
	OrderID                    NumericID       `json:"orderId"`
	Price                      types.Number    `json:"price"`
	StopPrice                  types.Number    `json:"stopPrice"`
	OriginalQuantity           types.Number    `json:"origQty"`
	ExecutedQuantity           types.Number    `json:"executedQty"`
	CummulativeQuoteQuantity   types.Number    `json:"cummulativeQuoteQty"`
	Status                     SpotOrderStatus `json:"status"`
	Type                       OrderType       `json:"type"`
	Side                       Side            `json:"side"`
	Time                       types.Time      `json:"time"`
	UpdateTime                 types.Time      `json:"updateTime"`
	OriginalQuoteOrderQuantity types.Number    `json:"origQuoteOrderQty"`
	Fee                        types.Number    `json:"fee"`
	FeeAsset                   currency.Code   `json:"feeAsset"`
	ClientOrderID              string          `json:"clientOrderID"`
	AveragePrice               types.Number    `json:"avgPrice"`
}

// SpotOrderDetailsResponse holds a list of spot orders' details
type SpotOrderDetailsResponse struct {
	Orders []SpotOrderDetailResponse `json:"orders"`
}

// SpotOrderHistoryRequest holds the parameters of a spot order history request
type SpotOrderHistoryRequest struct {
	Symbol currency.Pair
	// OrderID returns orders with this ID or later; with neither it nor a time range, the most recent orders are
	// returned
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// PageIndex defaults to 1 and PageSize to 100, at most 100; PageIndex times PageSize is at most 10000
	PageIndex uint64
	PageSize  uint64
	Status    SpotOrderStatus // FILLED, CANCELED or FAILED
	Type      OrderType
}

// SpotTransactionDetailsRequest holds the parameters of a spot fill query
type SpotTransactionDetailsRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// FromID returns fills from this trade ID on; without it the most recent fills are returned
	FromID uint64
	Limit  uint64
}

// SpotFillsResponse holds a list of spot fills
type SpotFillsResponse struct {
	Fills []SpotFillResponse `json:"fills"`
}

// SpotFillResponse holds a fill of a spot order
type SpotFillResponse struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         NumericID     `json:"orderId"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      float64       `json:"commission"` // Negative when charged
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
}

// SpotCommissionRateResponse holds the account's spot commission rates for a symbol
type SpotCommissionRateResponse struct {
	TakerCommissionRate float64 `json:"takerCommissionRate"`
	MakerCommissionRate float64 `json:"makerCommissionRate"`
}

// CancelAllSpotOrdersAfterResponse holds the state of the countdown that cancels every open spot order
type CancelAllSpotOrdersAfterResponse struct {
	TriggerTime types.Time           `json:"triggerTime"` // Sent in seconds
	Status      CancelAllAfterStatus `json:"status"`
	Note        string               `json:"note"`
}

// PlaceSpotOCOOrderRequest holds the parameters of a spot one-cancels-the-other (OCO) order list: a limit order and a
// stop-limit order, either of which cancels the other when it executes
type PlaceSpotOCOOrderRequest struct {
	Symbol   currency.Pair
	Side     Side
	Quantity float64
	// LimitPrice is the limit order's price
	LimitPrice float64
	// OrderPrice is the price of the limit order the stop-limit order places when TriggerPrice is reached
	OrderPrice   float64
	TriggerPrice float64
	// ListClientOrderID, AboveClientOrderID and BelowClientOrderID, client IDs of the list, its limit order and its
	// stop-limit order, take digits only
	ListClientOrderID  string
	AboveClientOrderID string
	BelowClientOrderID string
}

// SpotOCOOrderResponse holds an order of a spot one-cancels-the-other (OCO) order list
type SpotOCOOrderResponse struct {
	TransactionTime types.Time      `json:"transactionTime"`
	OrderID         NumericID       `json:"orderId"`
	ClientOrderID   string          `json:"clientOrderId"`
	Symbol          string          `json:"symbol"`
	OrderType       OCOOrderType    `json:"orderType"`
	Side            Side            `json:"side"`
	TriggerPrice    float64         `json:"triggerPrice"`
	Price           float64         `json:"price"`
	Quantity        float64         `json:"quantity"`
	OrderListID     string          `json:"orderListId"`
	Status          SpotOrderStatus `json:"status"`
}

// CancelSpotOCOOrderListResponse identifies the order a spot one-cancels-the-other (OCO) order list cancellation found
// the list by
type CancelSpotOCOOrderListResponse struct {
	OrderID       NumericID `json:"orderId"`
	ClientOrderID string    `json:"clientOrderId"`
}

// SpotOCOOrderHistoryRequest holds the parameters of a spot one-cancels-the-other (OCO) order history request
type SpotOCOOrderHistoryRequest struct {
	PageIndex uint64
	PageSize  uint64
	StartTime time.Time
	EndTime   time.Time
}
