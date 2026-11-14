package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// NewOrderRequest holds the parameters of a new order. Which quantity, price, stop and trailing parameters are
// mandatory depends on Type
type NewOrderRequest struct {
	Symbol             currency.Pair
	Side               Side
	Type               OrderType
	TimeInForce        TimeInForce
	Quantity           float64
	QuoteOrderQuantity float64 // Quote asset amount a MARKET order spends or receives, instead of Quantity
	Price              float64
	// NewClientOrderID identifies the order; API Partner Program members begin it with their partner code
	NewClientOrderID           string
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	IcebergQuantity            float64
	// NewOrderResponseType selects the ACK, RESULT or FULL response; MARKET and LIMIT orders default to FULL and the
	// others to ACK
	NewOrderResponseType    NewOrderResponseType
	SelfTradePreventionMode SelfTradePreventionMode
}

// OrderConditionalFields holds the order response fields that appear only when an order uses the feature behind them,
// such as an iceberg quantity, self-trade prevention, a stop price, a trailing stop or an expiry
type OrderConditionalFields struct {
	IcebergQuantity            types.Number `json:"icebergQty"`
	PreventedMatchID           uint64       `json:"preventedMatchId"`
	PreventedQuantity          types.Number `json:"preventedQuantity"`
	StopPrice                  types.Number `json:"stopPrice"`
	TrailingDeltaInBasisPoints uint64       `json:"trailingDelta"`
	// TrailingTime is the Unix time in milliseconds the trailing stop started tracking the price, -1 until then
	TrailingTime int64 `json:"trailingTime"`
	// ExpiryReason says why an expired order expired; it is sent by RESULT and FULL placements only
	ExpiryReason string `json:"expiryReason"`
}

// NewOrderResponse holds a new order in the ACK, RESULT or FULL format: ACK sets only the identifiers and transaction
// time, and FULL adds the fills to RESULT
type NewOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Fills                   []OrderFill             `json:"fills"`
	OrderConditionalFields
}

// OrderFill holds a fill of a new order
type OrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         uint64        `json:"tradeId"`
}

// TradeOrderResponse holds an order as queried
type TradeOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	Time                       types.Time   `json:"time"`
	UpdateTime                 types.Time   `json:"updateTime"`
	IsWorking                  bool         `json:"isWorking"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// CancelOrderRequest holds the parameters of an order cancellation; OrderID or OriginalClientOrderID is required, and
// when both are sent they must name the same order
type CancelOrderRequest struct {
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	NewClientOrderID      string             // Identifies the cancellation; generated when empty
	CancelRestrictions    CancelRestrictions // ONLY_NEW or ONLY_PARTIALLY_FILLED cancels only orders with that status
}

// CancelOrderResponse holds a cancelled order
type CancelOrderResponse struct {
	Symbol                     string                  `json:"symbol"`
	OriginalClientOrderID      string                  `json:"origClientOrderId"`
	OrderID                    uint64                  `json:"orderId"`
	OrderListID                int64                   `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string                  `json:"clientOrderId"`
	TransactTime               types.Time              `json:"transactTime"`
	Price                      types.Number            `json:"price"`
	OriginalQuantity           types.Number            `json:"origQty"`
	ExecutedQuantity           types.Number            `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number            `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number            `json:"cummulativeQuoteQty"`
	Status                     string                  `json:"status"`
	TimeInForce                TimeInForce             `json:"timeInForce"`
	Type                       OrderType               `json:"type"`
	Side                       Side                    `json:"side"`
	SelfTradePreventionMode    SelfTradePreventionMode `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// CancelledOpenOrderResponse holds an entry of a cancel open orders response: a cancelled order, or a cancelled order
// list with its list status, orders and order reports
type CancelledOpenOrderResponse struct {
	CancelOrderResponse
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// AccountTradeListRequest holds the parameters of an account trade list request. StartTime and EndTime can be at most
// 24 hours apart, and cannot be combined with OrderID or FromID
type AccountTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Trades from this trade ID on
	Limit     uint64 // 500 when zero, at most 1000
}

// AccountTradeResponse holds a trade of the account
type AccountTradeResponse struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is part of an order list
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsBestMatch     bool          `json:"isBestMatch"`
}

// CancelReplaceOrderRequest holds the parameters of a cancel-replace: the order to cancel, by CancelOrderID or
// CancelOriginalClientOrderID, and the new order, whose mandatory parameters depend on Type
type CancelReplaceOrderRequest struct {
	Symbol currency.Pair
	Side   Side
	Type   OrderType
	// CancelReplaceMode STOP_ON_FAILURE places the new order only if the cancellation succeeds, and ALLOW_FAILURE places
	// it either way
	CancelReplaceMode           CancelReplaceMode
	TimeInForce                 TimeInForce
	Quantity                    float64
	QuoteOrderQuantity          float64
	Price                       float64
	CancelNewClientOrderID      string // Identifies the cancellation; generated when empty
	CancelOriginalClientOrderID string
	CancelOrderID               uint64 // Takes precedence over CancelOriginalClientOrderID
	NewClientOrderID            string
	StrategyID                  uint64
	StrategyType                uint64 // At least 1000000; smaller values are reserved
	StopPrice                   float64
	TrailingDeltaInBasisPoints  uint64
	IcebergQuantity             float64
	NewOrderResponseType        NewOrderResponseType
	SelfTradePreventionMode     SelfTradePreventionMode
	CancelRestrictions          CancelRestrictions
}

// CancelAndReplaceResponse holds the outcome of a cancel-replace. Binance.US replies with it when both halves succeed
// and puts it in the error's data when either fails, so a failed half holds an error code and message instead of an
// order
type CancelAndReplaceResponse struct {
	CancelResult     string                        `json:"cancelResult"`   // SUCCESS or FAILURE
	NewOrderResult   string                        `json:"newOrderResult"` // SUCCESS, FAILURE or NOT_ATTEMPTED
	CancelResponse   CancelReplaceCancelResponse   `json:"cancelResponse"`
	NewOrderResponse CancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// CancelReplaceCancelResponse holds the cancellation half of a cancel-replace: the cancelled order, or the error code
// and message of a failed cancellation
type CancelReplaceCancelResponse struct {
	CancelOrderResponse
	Code    int64  `json:"code"` // Negative when the cancellation failed
	Message string `json:"msg"`
}

// CancelReplaceNewOrderResponse holds the new order half of a cancel-replace: the placed order, or the error code and
// message of a failed placement
type CancelReplaceNewOrderResponse struct {
	NewOrderResponse
	Code    int64  `json:"code"` // Negative when the placement failed
	Message string `json:"msg"`
}

// PreventedMatchesRequest holds the parameters of a prevented matches request, by PreventedMatchID or by OrderID,
// which FromPreventedMatchID and Limit page through
type PreventedMatchesRequest struct {
	Symbol               currency.Pair
	PreventedMatchID     uint64
	OrderID              uint64
	FromPreventedMatchID uint64
	Limit                uint64 // 500 when zero, at most 1000
}

// PreventedMatchResponse holds a match that self-trade prevention (STP) prevented, expiring the orders involved
type PreventedMatchResponse struct {
	Symbol                  string                  `json:"symbol"`
	PreventedMatchID        uint64                  `json:"preventedMatchId"`
	TakerOrderID            uint64                  `json:"takerOrderId"`
	MakerOrderID            uint64                  `json:"makerOrderId"`
	TradeGroupID            uint64                  `json:"tradeGroupId"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Price                   types.Number            `json:"price"`
	MakerPreventedQuantity  types.Number            `json:"makerPreventedQuantity"`
	TransactTime            types.Time              `json:"transactTime"`
}

// AllOrdersRequest holds the parameters of an all orders request. OrderID returns orders from that ID on, and
// StartTime and EndTime can be at most 24 hours apart
type AllOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// OneCancelsTheOtherOrderRequest holds the parameters of a one-cancels-the-other (OCO) order list: a LIMIT_MAKER order
// at Price and a STOP_LOSS order triggered at StopPrice, which is a STOP_LOSS_LIMIT order at StopLimitPrice when that is
// set. A sell's Price must be above the last price and its StopPrice below; a buy's the reverse
type OneCancelsTheOtherOrderRequest struct {
	Symbol               currency.Pair
	ListClientOrderID    string // Identifies the order list
	Side                 Side
	Quantity             float64
	LimitClientOrderID   string // Identifies the limit order
	Price                float64
	LimitIcebergQuantity float64
	// TrailingDeltaInBasisPoints makes the stop order a trailing stop
	TrailingDeltaInBasisPoints uint64
	StopClientOrderID          string // Identifies the stop order
	StopPrice                  float64
	StopLimitPrice             float64
	StopIcebergQuantity        float64
	// StopLimitTimeInForce is required with StopLimitPrice
	StopLimitTimeInForce    TimeInForce
	NewOrderResponseType    NewOrderResponseType
	SelfTradePreventionMode SelfTradePreventionMode // EXPIRE_TAKER, EXPIRE_MAKER or EXPIRE_BOTH
}

// OneCancelsTheOtherOrderResponse holds an order list; placements and cancellations add the order reports
type OneCancelsTheOtherOrderResponse struct {
	OrderListID       uint64           `json:"orderListId"`
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Symbol            string           `json:"symbol"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// OrderListOrder identifies an order of an order list
type OrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// OrderResponse holds an order report of an order list placement, in the placement's ACK, RESULT or FULL format, or of
// an order list cancellation, which alone sends OriginalClientOrderID
type OrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OriginalClientOrderID      string       `json:"origClientOrderId"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Fills                   []OrderFill             `json:"fills"`
	OrderConditionalFields
}

// AllOrderListsRequest holds the optional filters of an all order lists request. FromID cannot be combined with
// StartTime or EndTime
type AllOrderListsRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// QueryOrderListRequest identifies an order list by OrderListID or by its list client order ID, which the endpoint takes
// as origClientOrderId
type QueryOrderListRequest struct {
	// OrderListID is a pointer, as 0 is a valid order list ID
	OrderListID           *uint64
	OriginalClientOrderID string
}

// CancelOrderListRequest holds the parameters of an order list cancellation; OrderListID or ListClientOrderID is
// required
type CancelOrderListRequest struct {
	Symbol currency.Pair
	// OrderListID is a pointer, as 0 is a valid order list ID
	OrderListID       *uint64
	ListClientOrderID string
	NewClientOrderID  string // Identifies the cancellation; generated when empty
}
