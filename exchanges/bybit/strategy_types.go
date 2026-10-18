package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Strategy results follow protobuf's JSON mapping: 64-bit integers such as timestamps and counts arrive quoted, and
// enums such as orderType arrive as names (UNKNOWN) rather than the numbers the docs list

// CreateStrategyOrderRequest holds a new strategy order; StrategyType decides which other fields apply
type CreateStrategyOrderRequest struct {
	// Category is UTA_USDT, UTA_USDC, UTA_SPOT, UTA_INVERSE, UTA_INVERSE_FUTURE or UTA_USDT_FUTURE
	Category string
	Symbol   string
	// Side is Buy or Sell
	Side string
	// StrategyType is twap, chaseOrder, iceberg or pov
	StrategyType string
	// Size and PositionValue are the total quantity in coin and in value: TWAP, chase and iceberg orders need one, and
	// POV orders take at most one
	Size          float64
	PositionValue float64
	// Duration is sent in whole seconds: TWAP orders need 5 minutes to 24 hours, divisible by Interval, and POV orders
	// take 15 minutes to 24 hours
	Duration   time.Duration
	ReduceOnly bool
	// PositionIndex is 0 one-way, 1 hedge-mode buy side or 2 hedge-mode sell side, which hedge mode requires
	PositionIndex uint64
	// LeverageType is 0 normal or 1 borrow to trade, for UTA_SPOT only
	LeverageType uint64
	// Interval is the sub-order interval, sent in whole seconds when set: 5, 10, 15, 30 (the default), 60 or 120 for
	// TWAP orders, and 0 for a single execution or 5 seconds to an hour for POV orders
	Interval *time.Duration
	// IsRandom varies each TWAP sub-order's quantity by up to 20%
	IsRandom     bool
	TriggerPrice float64
	// MaxChasePrice terminates a TWAP order, stops a chase order chasing and pauses an iceberg order's sub-orders
	MaxChasePrice float64
	// ChaseDistance is the absolute distance from the best bid or ask; -1 fills an iceberg order at the counterparty
	// price
	ChaseDistance float64
	// ChasePercentE4 is the offset from the best bid or ask in basis points: 100 is 1%
	ChasePercentE4 uint64
	// SubSize, OrderCount and SubPositionValue split an iceberg order into sub-orders
	SubSize          float64
	OrderCount       uint64
	SubPositionValue float64
	// PostOnly sets an iceberg order's maker-only mode when not nil. Create Strategy Order documents 0 as post-only
	// and 1 as taker allowed, but Get Strategy List documents the reverse
	PostOnly *uint64
	// LimitPrice applies to every iceberg sub-order
	LimitPrice float64
	// POVParameters is required by POV orders
	POVParameters *StrategyPOVParametersRequest
}

// StrategyPOVParametersRequest holds a POV order's execution parameters
type StrategyPOVParametersRequest struct {
	// Mode is TradedVolume, OppositeSideLiquidity or SameSideLiquidity
	Mode string
	// ParticipationRate is a percentage from 1 to 100 with one decimal place: 25.1 is 25.1%
	ParticipationRate float64
	// ReferenceWindow is the traded volume window TradedVolume needs, sent in whole seconds from 60 to 14400
	ReferenceWindow time.Duration
	// DepthReference is the number of book levels, from 1 to 10, the liquidity modes need
	DepthReference uint64
}

// CreateStrategyOrderResponse holds a new strategy order's ID and outcome
type CreateStrategyOrderResponse struct {
	StrategyID string `json:"strategyId"`
	// Result is null when the strategy was created, and otherwise describes why it was not
	Result string `json:"result"`
}

// GetStrategyListRequest holds the filters for listing strategies
type GetStrategyListRequest struct {
	StrategyID string
	Symbol     string
	// Status is 2 running, 3 terminated, 4 terminated with unfilled orders, 5 paused or 6 untriggered
	Status uint64
	// Category is UTA_USDT, UTA_USDC, UTA_USDC_FUTURE, UTA_SPOT, UTA_INVERSE, UTA_INVERSE_FUTURE or UTA_USDT_FUTURE
	Category string
	// StrategyType is twap, chaseOrder, iceberg or pov
	StrategyType string
	// BeginTime and EndTime are sent in whole seconds
	BeginTime time.Time
	EndTime   time.Time
	// PageSize defaults to 20 and is at most 50
	PageSize uint64
	Cursor   string
}

// StrategyListResponse holds a page of strategies
type StrategyListResponse struct {
	List []Strategy `json:"list"`
	// NextCursor is empty on the last page
	NextCursor     string `json:"nextCursor"`
	PreviousCursor string `json:"prevCursor"`
}

// Strategy holds a strategy order's settings and progress
type Strategy struct {
	StrategyID string       `json:"strategyId"`
	Category   string       `json:"category"`
	Symbol     string       `json:"symbol"`
	Side       string       `json:"side"`
	Size       types.Number `json:"size"`
	// StrategyType is twap, chaseOrder, iceberg or pov
	StrategyType string `json:"strategyType"`
	// Status is 2 running, 3 terminated, 4 terminated with unfilled orders, 5 paused or 6 untriggered
	Status               uint64       `json:"status"`
	ExecutedSize         types.Number `json:"executedSize"`
	ExecutedAveragePrice types.Number `json:"executedAvgPrice"`
	ExecutedStartTime    types.Time   `json:"executedStartTimeE3"`
	// ExecutedEndTime is zero until the strategy ends
	ExecutedEndTime types.Time   `json:"executedEndTimeE3"`
	CreatedTime     types.Time   `json:"createdTimeE3"`
	UpdatedTime     types.Time   `json:"updatedTimeE3"`
	ReduceOnly      bool         `json:"reduceOnly"`
	TriggerPrice    types.Number `json:"triggerPrice"`
	IsTriggered     bool         `json:"isTriggered"`
	// LeverageType is 0 normal or 1 margin
	LeverageType uint64 `json:"leverageType"`
	// TerminateType is the termination reason code, such as 1 user stop or 2 completed normally
	TerminateType   uint64       `json:"terminateType"`
	TerminateRemark string       `json:"terminateRemark"`
	TriggerCount    types.Number `json:"triggerCount"`
	TradingCount    types.Number `json:"tradingCount"`
	// RealisedPNL is set for futures only
	RealisedPNL  types.Number `json:"realizedPnl"`
	StrategyName string       `json:"strategyName"`
	// StrategyPrefer is limit, priceSpeedBalance, fastestExecution or quickExecution
	StrategyPrefer string `json:"strategyPrefer"`
	// Duration, ExecutedDuration and Interval are in seconds, for TWAP and POV orders
	Duration         uint64       `json:"duration"`
	ExecutedDuration uint64       `json:"executedDuration"`
	IsRandom         bool         `json:"isRandom"`
	Interval         uint64       `json:"interval"`
	LimitPrice       types.Number `json:"limitPrice"`
	// ChasePercentE4 is in basis points: 100 is 1%
	ChasePercentE4  types.Number `json:"chasePercentE4"`
	ChaseDistance   types.Number `json:"chaseDistance"`
	MaxChasePrice   types.Number `json:"maxChasePrice"`
	ChaseOrderPrice types.Number `json:"chaseOrderPrice"`
	// ChasePrice is the reference price side, such as Bid1 or Ask1
	ChasePrice string `json:"chasePrice"`
	// PostOnly is an iceberg order's maker-only mode: 0 taker allowed or 1 post-only
	PostOnly    uint64 `json:"postOnly"`
	IsRebalance bool   `json:"isRebalance"`
	// OrderType arrives as an enum name such as UNKNOWN, although the docs list 1 market and 2 limit
	OrderType string `json:"orderType"`
	// OrderPriceOffset is a percentage, for TWAP orders
	OrderPriceOffset types.Number `json:"orderPriceOffset"`
	StrategySL       types.Number `json:"strategySl"`
	StrategyTP       types.Number `json:"strategyTp"`
	// ArbitrageOrders holds the strategy's arbitrage orders, whose fields the docs do not describe
	ArbitrageOrders     []json.RawMessage `json:"arbitrageOrders"`
	PositionValue       types.Number      `json:"positionValue"`
	FilledPositionValue types.Number      `json:"filledPositionValue"`
	// Mode, ParticipationRate, ReferenceWindow and DepthReference are set for POV orders
	Mode              string       `json:"mode"`
	ParticipationRate types.Number `json:"participationRate"`
	ReferenceWindow   types.Number `json:"referenceWindow"`
	DepthReference    types.Number `json:"depthReference"`
}

// GetStrategyOrderListRequest holds the filters for listing a strategy's orders
type GetStrategyOrderListRequest struct {
	StrategyID string
	// Status is 2 created, 3 create failed, 4 partially filled, 5 fully filled, 6 rejected or 7 cancelled
	Status uint64
	Symbol string
	// StrategyType is twap, chaseOrder, iceberg or pov
	StrategyType string
	// BeginTime and EndTime are sent in whole seconds
	BeginTime time.Time
	EndTime   time.Time
	// PageSize defaults to 20
	PageSize uint64
	Cursor   string
}

// StrategyOrderListResponse holds a page of a strategy's orders
type StrategyOrderListResponse struct {
	List []StrategyOrder `json:"list"`
	// NextCursor is empty on the last page
	NextCursor     string `json:"nextCursor"`
	PreviousCursor string `json:"prevCursor"`
}

// StrategyOrder holds an order a strategy placed
type StrategyOrder struct {
	StrategyID string       `json:"strategyId"`
	OrderID    string       `json:"orderId"`
	Symbol     string       `json:"symbol"`
	Side       string       `json:"side"`
	Size       types.Number `json:"size"`
	Price      types.Number `json:"price"`
	// Status is 2 created, 3 create failed, 4 partially filled, 5 fully filled, 6 rejected or 7 cancelled
	Status       string       `json:"status"`
	ExecutedSize types.Number `json:"executedSize"`
	// DealTime is zero until the order fills
	DealTime types.Time `json:"dealTimeE3"`
	// ParentOrderID is the order a retried order replaces
	ParentOrderID string     `json:"parentOrderId"`
	CreatedTime   types.Time `json:"createdTimeE3"`
	UpdatedTime   types.Time `json:"updatedTimeE3"`
	// Code is 0 on success, and otherwise the order's error code
	Code     int64  `json:"code"`
	Message  string `json:"msg"`
	Category string `json:"category"`
	// PositionIndex is 0 one-way, 1 hedge-mode buy side or 2 hedge-mode sell side
	PositionIndex uint64 `json:"positionIdx"`
	// OrderType arrives as an enum name such as UNKNOWN, although the docs list 1 market and 2 limit
	OrderType string `json:"orderType"`
	// LeverageType is 0 normal or 1 margin
	LeverageType uint64       `json:"leverageType"`
	TPPrice      types.Number `json:"tpPrice"`
	SLPrice      types.Number `json:"slPrice"`
	// OrderPriceOffset is a percentage
	OrderPriceOffset    types.Number `json:"orderPriceOffset"`
	PositionValue       types.Number `json:"positionValue"`
	FilledPositionValue types.Number `json:"filledPositionValue"`
}

// StopStrategyResponse holds the ID of a stopped strategy
type StopStrategyResponse struct {
	StrategyID string `json:"strategyId"`
}
