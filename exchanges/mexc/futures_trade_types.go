package mexc

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Futures order sides
const (
	FuturesSideOpenLong   = 1
	FuturesSideCloseShort = 2
	FuturesSideOpenShort  = 3
	FuturesSideCloseLong  = 4
)

// Futures order directions, the sideType a single order names
const (
	FuturesSideTypeBuy  = "buy"
	FuturesSideTypeSell = "sell"
)

// Futures order types
const (
	FuturesOrderTypeLimit             = 1
	FuturesOrderTypePostOnly          = 2
	FuturesOrderTypeImmediateOrCancel = 3
	FuturesOrderTypeFillOrKill        = 4
	FuturesOrderTypeMarket            = 5
	FuturesOrderTypeMarketToLimit     = 6
)

// Futures order states
const (
	FuturesOrderStatePending   = 1
	FuturesOrderStateUnfilled  = 2
	FuturesOrderStateFilled    = 3
	FuturesOrderStateCancelled = 4
	FuturesOrderStateInvalid   = 5
)

// Futures order categories
const (
	FuturesOrderCategoryLimit                = 1
	FuturesOrderCategoryLiquidationCustody   = 2
	FuturesOrderCategoryCustodyClose         = 3
	FuturesOrderCategoryAutoDeleverage       = 4
	FuturesOrderCategoryDelivery             = 5
	FuturesOrderCategoryLiquidationReduction = 6
)

// Futures trigger price types, which name the price a trigger compares
const (
	FuturesPriceTypeLatest = 1
	FuturesPriceTypeFair   = 2
	FuturesPriceTypeIndex  = 3
)

// Futures trigger types
const (
	FuturesTriggerGreaterOrEqual = 1
	FuturesTriggerLessOrEqual    = 2
)

// Futures plan order execution cycles
const (
	FuturesExecuteCycle24Hours = 1
	FuturesExecuteCycle7Days   = 2
)

// Futures trailing order callback types
const (
	FuturesCallbackPercentage = 1
	FuturesCallbackAbsolute   = 2
)

// FuturesOrderRequest holds the parameters of a futures order. Price is unused by market orders, and Volume is in
// contracts
type FuturesOrderRequest struct {
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price,omitempty"`
	Volume   float64 `json:"vol"`
	Leverage uint64  `json:"leverage,omitempty"`
	Side     uint8   `json:"side"`
	// SideType is FuturesSideTypeBuy for an order opening a long or closing a short and FuturesSideTypeSell for one
	// opening a short or closing a long. The single order endpoint requires it; the batch endpoint doesn't list it
	SideType        string  `json:"sideType,omitempty"`
	Type            uint8   `json:"type"`
	OpenType        uint8   `json:"openType"`
	ExternalOrderID string  `json:"externalOid,omitempty"`
	PositionID      uint64  `json:"positionId,omitempty"`
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
	// LossTrend and ProfitTrend name the price the stop-loss and take-profit prices compare, FuturesPriceTypeLatest
	// by default
	LossTrend    uint8 `json:"lossTrend,omitempty"`
	ProfitTrend  uint8 `json:"profitTrend,omitempty"`
	PriceProtect uint8 `json:"priceProtect,omitempty"`
	PositionMode uint8 `json:"positionMode,omitempty"`
	ReduceOnly   bool  `json:"reduceOnly,omitempty"`
	// MarketCeiling opens a position with all of the available balance
	MarketCeiling bool `json:"marketCeiling,omitempty"`
	FlashClose    bool `json:"flashClose,omitempty"`
	// BBOTypeNumber prices a limit order at the best bid or offer: 1 the opposite side's best, 2 its fifth best, 3
	// the same side's best and 4 its fifth best; such an order takes no external order ID
	BBOTypeNumber uint8 `json:"bboTypeNum,omitempty"`
	// SelfTradePreventionMode is 0 for no action, 1 to cancel both orders, 2 the maker order and 3 the taker order
	SelfTradePreventionMode uint8 `json:"stpMode"`
}

// FuturesOrderPlacementResponse holds a placed order's ID
type FuturesOrderPlacementResponse struct {
	OrderID   FuturesID  `json:"orderId"`
	Timestamp types.Time `json:"ts"`
}

// FuturesBatchOrderResponse holds the outcome of one order of a batch
type FuturesBatchOrderResponse struct {
	OrderID         FuturesID `json:"orderId"`
	ExternalOrderID string    `json:"externalOid"`
	ErrorCode       uint64    `json:"errorCode"`
	ErrorMessage    string    `json:"errorMsg"`
}

// FuturesChaseOrderRequest names a limit order to move to the best price on its side of the book
type FuturesChaseOrderRequest struct {
	OrderID uint64 `json:"orderId"`
}

// FuturesModifyOrderRequest holds a limit order's new price and volume
type FuturesModifyOrderRequest struct {
	OrderID uint64  `json:"orderId"`
	Price   float64 `json:"price"`
	Volume  float64 `json:"vol"`
}

// FuturesCancelOrderResponse holds the outcome of one order's cancellation; a non-zero ErrorCode is a failure
type FuturesCancelOrderResponse struct {
	OrderID         FuturesID `json:"orderId"`
	ExternalOrderID string    `json:"externalOid"`
	ErrorCode       uint64    `json:"errorCode"`
	ErrorMessage    string    `json:"errorMsg"`
}

// FuturesExternalOrderRequest names an order by its contract and external order ID
type FuturesExternalOrderRequest struct {
	Symbol          string `json:"symbol"`
	ExternalOrderID string `json:"externalOid"`
}

// FuturesCancelAllOrdersRequest names the contract whose open orders to cancel; an empty symbol names every contract
type FuturesCancelAllOrdersRequest struct {
	Symbol string `json:"symbol"`
}

// FuturesReversePositionRequest holds the parameters of a position reversal
type FuturesReversePositionRequest struct {
	Symbol     string  `json:"symbol"`
	PositionID uint64  `json:"positionId"`
	Volume     float64 `json:"vol"`
}

// FuturesOpenOrderCountsResponse holds the numbers of the account's open orders by kind
type FuturesOpenOrderCountsResponse struct {
	SumCount        uint64 `json:"sumCount"`
	LimitOrderCount uint64 `json:"limitOrderCount"`
	StopOrderCount  uint64 `json:"stopOrderCount"`
	PlanOrderCount  uint64 `json:"planOrderCount"`
	TrackOrderCount uint64 `json:"trackOrderCount"`
}

// FuturesOrderResponse holds a futures order. Volumes are in contracts
type FuturesOrderResponse struct {
	OrderID                  FuturesID     `json:"orderId"`
	Symbol                   string        `json:"symbol"`
	PositionID               FuturesID     `json:"positionId"`
	Price                    float64       `json:"price"`
	PriceExact               types.Number  `json:"priceStr"`
	Volume                   float64       `json:"vol"`
	Leverage                 uint64        `json:"leverage"`
	Side                     uint8         `json:"side"`
	Category                 uint8         `json:"category"`
	OrderType                uint8         `json:"orderType"`
	DealAveragePrice         float64       `json:"dealAvgPrice"`
	DealAveragePriceExact    types.Number  `json:"dealAvgPriceStr"`
	DealVolume               float64       `json:"dealVol"`
	RemainVolume             float64       `json:"remainVol"`
	OrderMargin              float64       `json:"orderMargin"`
	UsedMargin               float64       `json:"usedMargin"`
	TakerFee                 float64       `json:"takerFee"`
	MakerFee                 float64       `json:"makerFee"`
	TotalFee                 float64       `json:"totalFee"`
	Profit                   float64       `json:"profit"`
	FeeCurrency              currency.Code `json:"feeCurrency"`
	OpenType                 uint8         `json:"openType"`
	State                    uint8         `json:"state"`
	ExternalOrderID          string        `json:"externalOid"`
	ErrorCode                uint64        `json:"errorCode"`
	CreateTime               types.Time    `json:"createTime"`
	UpdateTime               types.Time    `json:"updateTime"`
	PositionMode             uint8         `json:"positionMode"`
	ReduceOnly               bool          `json:"reduceOnly"`
	Version                  uint64        `json:"version"`
	ShowCancelReason         uint8         `json:"showCancelReason"`
	ShowProfitRateShare      uint8         `json:"showProfitRateShare"`
	BBOTypeNumber            uint8         `json:"bboTypeNum"`
	StopLossPrice            float64       `json:"stopLossPrice"`
	TakeProfitPrice          float64       `json:"takeProfitPrice"`
	LossTrend                uint8         `json:"lossTrend"`
	ProfitTrend              uint8         `json:"profitTrend"`
	PriceProtect             uint8         `json:"priceProtect"`
	PNLRate                  float64       `json:"pnlRate"`
	OpenAveragePrice         float64       `json:"openAvgPrice"`
	MakerFeeRate             float64       `json:"makerFeeRate"`
	TakerFeeRate             float64       `json:"takerFeeRate"`
	ZeroSaveTotalFeeBinance  float64       `json:"zeroSaveTotalFeeBinance"`
	ZeroTradeTotalFeeBinance float64       `json:"zeroTradeTotalFeeBinance"`
}

// FuturesOrderHistoryRequest holds the parameters of a futures order history request
type FuturesOrderHistoryRequest struct {
	Symbol currency.Pair
	// States selects orders by state, such as FuturesOrderStateFilled
	States   []uint8
	Category uint8
	// OrderID selects a single order
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	FuturesPageRequest
}

// FuturesOrderDealsRequest holds the parameters of a futures deals or closed orders request
type FuturesOrderDealsRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	FuturesPageRequest
}

// FuturesOrderDealResponse holds a fill of a futures order. Volume is in contracts and Fee is positive when paid
type FuturesOrderDealResponse struct {
	ID          FuturesID     `json:"id"`
	Symbol      string        `json:"symbol"`
	Side        uint8         `json:"side"`
	Volume      float64       `json:"vol"`
	Price       float64       `json:"price"`
	Fee         float64       `json:"fee"`
	FeeCurrency currency.Code `json:"feeCurrency"`
	Profit      float64       `json:"profit"`
	// IsTaker is documented and the venue's examples send Taker; WasTaker reads either
	IsTaker         bool       `json:"isTaker"`
	Taker           bool       `json:"taker"`
	Category        uint8      `json:"category"`
	OrderID         FuturesID  `json:"orderId"`
	OpponentOrderID FuturesID  `json:"opponentOrderId"`
	ExternalOrderID string     `json:"externalOid"`
	PositionMode    uint8      `json:"positionMode"`
	ReduceOnly      bool       `json:"reduceOnly"`
	IsSelf          bool       `json:"isSelf"`
	Timestamp       types.Time `json:"timestamp"`
}

// WasTaker reports whether the fill took liquidity
func (d *FuturesOrderDealResponse) WasTaker() bool {
	return d.IsTaker || d.Taker
}

// FuturesPlanOrdersRequest holds the parameters of a plan order list request; the venue documents both bounds of
// the window as required
type FuturesPlanOrdersRequest struct {
	Symbol    currency.Pair
	States    []uint8
	Side      uint8
	StartTime time.Time
	EndTime   time.Time
	FuturesPageRequest
}

// FuturesPlanOrderResponse holds a plan order, which places an order once its trigger price is reached
type FuturesPlanOrderResponse struct {
	ID           FuturesID `json:"id"`
	Symbol       string    `json:"symbol"`
	Leverage     uint64    `json:"leverage"`
	Side         uint8     `json:"side"`
	TriggerPrice float64   `json:"triggerPrice"`
	Price        float64   `json:"price"`
	Volume       float64   `json:"vol"`
	OpenType     uint8     `json:"openType"`
	TriggerType  uint8     `json:"triggerType"`
	State        uint8     `json:"state"`
	// ExecuteCycle is in hours
	ExecuteCycle     uint64     `json:"executeCycle"`
	Trend            uint8      `json:"trend"`
	ErrorCode        uint64     `json:"errorCode"`
	OrderID          FuturesID  `json:"orderId"`
	OrderType        uint8      `json:"orderType"`
	MarketOrderLevel uint64     `json:"marketOrderLevel"`
	PriceProtect     uint8      `json:"priceProtect"`
	PositionMode     uint8      `json:"positionMode"`
	LossTrend        uint8      `json:"lossTrend"`
	ProfitTrend      uint8      `json:"profitTrend"`
	StopLossPrice    float64    `json:"stopLossPrice"`
	TakeProfitPrice  float64    `json:"takeProfitPrice"`
	ReduceOnly       bool       `json:"reduceOnly"`
	CreateTime       types.Time `json:"createTime"`
	UpdateTime       types.Time `json:"updateTime"`
}

// FuturesPlanOrderRequest holds the parameters of a plan order. Price is unused by market orders, and Volume is in
// contracts
type FuturesPlanOrderRequest struct {
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price,omitempty"`
	Volume          float64 `json:"vol"`
	Leverage        uint64  `json:"leverage"`
	Side            uint8   `json:"side"`
	OpenType        uint8   `json:"openType"`
	TriggerPrice    float64 `json:"triggerPrice"`
	TriggerType     uint8   `json:"triggerType"`
	ExecuteCycle    uint8   `json:"executeCycle"`
	OrderType       uint8   `json:"orderType"`
	Trend           uint8   `json:"trend"`
	PriceProtect    uint8   `json:"priceProtect,omitempty"`
	PositionMode    uint8   `json:"positionMode,omitempty"`
	LossTrend       uint8   `json:"lossTrend,omitempty"`
	ProfitTrend     uint8   `json:"profitTrend,omitempty"`
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
	ReduceOnly      bool    `json:"reduceOnly,omitempty"`
}

// FuturesModifyPlanOrderRequest holds a plan order's new trigger and execution
type FuturesModifyPlanOrderRequest struct {
	Symbol       string  `json:"symbol"`
	OrderID      uint64  `json:"orderId"`
	TriggerPrice float64 `json:"triggerPrice"`
	Price        float64 `json:"price"`
	OrderType    uint8   `json:"orderType"`
	TriggerType  uint8   `json:"triggerType"`
	Trend        uint8   `json:"trend"`
	// From names the screen a modification came from: 1 the trade list and 2 the order list
	From uint8 `json:"from,omitempty"`
}

// FuturesCancelPlanOrderRequest names a plan order to cancel
type FuturesCancelPlanOrderRequest struct {
	Symbol  string `json:"symbol"`
	OrderID string `json:"orderId"`
}

// FuturesCancelAllPlanOrdersRequest names the contract whose plan orders to cancel; an empty symbol names every
// contract
type FuturesCancelAllPlanOrdersRequest struct {
	Symbol string `json:"symbol,omitempty"`
}

// FuturesTakeProfitStopLossRequest holds the parameters of a take-profit and stop-loss order on a position; at least
// one of the prices must be set, and Volume is in contracts
type FuturesTakeProfitStopLossRequest struct {
	PositionID      uint64  `json:"positionId"`
	Volume          float64 `json:"vol"`
	LossTrend       uint8   `json:"lossTrend"`
	ProfitTrend     uint8   `json:"profitTrend"`
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
	PriceProtect    uint8   `json:"priceProtect,omitempty"`
	// ProfitLossVolumeType is SAME for one volume or SEPARATE for TakeProfitVolume and StopLossVolume
	ProfitLossVolumeType string  `json:"profitLossVolType,omitempty"`
	TakeProfitVolume     float64 `json:"takeProfitVol,omitempty"`
	StopLossVolume       float64 `json:"stopLossVol,omitempty"`
	// VolumeType is 1 for a partial take-profit and stop-loss and 2 for the whole position
	VolumeType        uint8 `json:"volType,omitempty"`
	TakeProfitReverse uint8 `json:"takeProfitReverse,omitempty"`
	StopLossReverse   uint8 `json:"stopLossReverse,omitempty"`
	// TakeProfitType and StopLossType are 0 for a market order and 1 for a limit order at the order price
	TakeProfitType       uint8   `json:"takeProfitType,omitempty"`
	TakeProfitOrderPrice float64 `json:"takeProfitOrderPrice,omitempty"`
	StopLossType         uint8   `json:"stopLossType,omitempty"`
	StopLossOrderPrice   float64 `json:"stopLossOrderPrice,omitempty"`
}

// FuturesModifyTakeProfitStopLossRequest holds the new take-profit and stop-loss prices of a limit order, named by
// OrderID, or of a take-profit and stop-loss order, named by StopPlanOrderID
type FuturesModifyTakeProfitStopLossRequest struct {
	OrderID           uint64  `json:"orderId,omitempty"`
	StopPlanOrderID   uint64  `json:"stopPlanOrderId,omitempty"`
	StopLossPrice     float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice   float64 `json:"takeProfitPrice,omitempty"`
	LossTrend         uint8   `json:"lossTrend,omitempty"`
	ProfitTrend       uint8   `json:"profitTrend,omitempty"`
	TakeProfitReverse uint8   `json:"takeProfitReverse,omitempty"`
	StopLossReverse   uint8   `json:"stopLossReverse,omitempty"`
}

// FuturesModifyPlanOrderTakeProfitStopLossRequest holds a plan order's new take-profit and stop-loss prices; a zero
// price cancels that side
type FuturesModifyPlanOrderTakeProfitStopLossRequest struct {
	Symbol          string  `json:"symbol"`
	OrderID         uint64  `json:"orderId"`
	LossTrend       uint8   `json:"lossTrend,omitempty"`
	ProfitTrend     uint8   `json:"profitTrend,omitempty"`
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
}

// FuturesTakeProfitStopLossOrdersRequest holds the parameters of a take-profit and stop-loss order list request
type FuturesTakeProfitStopLossOrdersRequest struct {
	Symbol currency.Pair
	// IsFinished selects orders in a final state when true and the others when false
	IsFinished   *bool
	State        uint8
	PositionType uint8
	StartTime    time.Time
	EndTime      time.Time
	FuturesPageRequest
}

// FuturesCancelTakeProfitStopLossRequest names a take-profit and stop-loss order to cancel
type FuturesCancelTakeProfitStopLossRequest struct {
	StopPlanOrderID uint64 `json:"stopPlanOrderId"`
}

// FuturesCancelAllTakeProfitStopLossRequest names the position, or else the contract, whose take-profit and
// stop-loss orders to cancel; naming neither names every contract
type FuturesCancelAllTakeProfitStopLossRequest struct {
	PositionID uint64 `json:"positionId,omitempty"`
	Symbol     string `json:"symbol,omitempty"`
}

// FuturesTakeProfitStopLossOrderResponse holds a take-profit and stop-loss order
type FuturesTakeProfitStopLossOrderResponse struct {
	ID                   FuturesID  `json:"id"`
	OrderID              FuturesID  `json:"orderId"`
	Symbol               string     `json:"symbol"`
	PositionID           FuturesID  `json:"positionId"`
	LossTrend            uint8      `json:"lossTrend"`
	ProfitTrend          uint8      `json:"profitTrend"`
	StopLossPrice        float64    `json:"stopLossPrice"`
	TakeProfitPrice      float64    `json:"takeProfitPrice"`
	State                uint8      `json:"state"`
	TriggerSide          uint8      `json:"triggerSide"`
	PositionType         uint8      `json:"positionType"`
	Volume               float64    `json:"vol"`
	RealityVolume        float64    `json:"realityVol"`
	PlaceOrderID         FuturesID  `json:"placeOrderId"`
	ErrorCode            uint64     `json:"errorCode"`
	IsFinished           uint8      `json:"isFinished"`
	Version              uint64     `json:"version"`
	PriceProtect         uint8      `json:"priceProtect"`
	ProfitLossVolumeType string     `json:"profitLossVolType"`
	TakeProfitVolume     float64    `json:"takeProfitVol"`
	StopLossVolume       float64    `json:"stopLossVol"`
	VolumeType           uint8      `json:"volType"`
	TakeProfitReverse    uint8      `json:"takeProfitReverse"`
	StopLossReverse      uint8      `json:"stopLossReverse"`
	CloseTryTimes        uint64     `json:"closeTryTimes"`
	ReverseTryTimes      uint64     `json:"reverseTryTimes"`
	ReverseErrorCode     uint64     `json:"reverseErrorCode"`
	TakeProfitType       uint8      `json:"takeProfitType"`
	TakeProfitOrderPrice float64    `json:"takeProfitOrderPrice"`
	StopLossType         uint8      `json:"stopLossType"`
	StopLossOrderPrice   float64    `json:"stopLossOrderPrice"`
	CreateTime           types.Time `json:"createTime"`
	UpdateTime           types.Time `json:"updateTime"`
}

// FuturesTrailingOrderRequest holds the parameters of a trailing order; Volume is in contracts
type FuturesTrailingOrderRequest struct {
	Symbol      string  `json:"symbol"`
	Leverage    uint64  `json:"leverage"`
	Side        uint8   `json:"side"`
	Volume      float64 `json:"vol"`
	OpenType    uint8   `json:"openType"`
	Trend       uint8   `json:"trend"`
	ActivePrice float64 `json:"activePrice,omitempty"`
	BackType    uint8   `json:"backType"`
	BackValue   float64 `json:"backValue"`
	// PositionMode is FuturesPositionModeHedge or FuturesPositionModeOneWay
	PositionMode uint8 `json:"positionMode"`
	ReduceOnly   bool  `json:"reduceOnly,omitempty"`
}

// FuturesModifyTrailingOrderRequest holds a trailing order's new parameters
type FuturesModifyTrailingOrderRequest struct {
	Symbol       string  `json:"symbol"`
	TrackOrderID uint64  `json:"trackOrderId"`
	Trend        uint8   `json:"trend"`
	ActivePrice  float64 `json:"activePrice,omitempty"`
	BackType     uint8   `json:"backType"`
	BackValue    float64 `json:"backValue"`
	Volume       float64 `json:"vol"`
}

// FuturesCancelTrailingOrderRequest names a trailing order to cancel
type FuturesCancelTrailingOrderRequest struct {
	Symbol       string `json:"symbol,omitempty"`
	TrackOrderID uint64 `json:"trackOrderId"`
}

// FuturesTrailingOrdersRequest holds the parameters of a trailing order list request
type FuturesTrailingOrdersRequest struct {
	Symbol currency.Pair
	// States selects orders by state: 0 not activated, 1 activated, 2 triggered, 3 failed and 4 cancelled
	States    []uint8
	Side      uint8
	StartTime time.Time
	EndTime   time.Time
	// PageIndex and PageSize page the list; the venue names them differently to its other lists
	PageIndex uint64
	PageSize  uint64
}

// FuturesTrailingOrderResponse holds a trailing order
type FuturesTrailingOrderResponse struct {
	ID           FuturesID  `json:"id"`
	Symbol       string     `json:"symbol"`
	Leverage     uint64     `json:"leverage"`
	Side         uint8      `json:"side"`
	Volume       float64    `json:"vol"`
	OpenType     uint8      `json:"openType"`
	Trend        uint8      `json:"trend"`
	ActivePrice  float64    `json:"activePrice"`
	MarkPrice    float64    `json:"markPrice"`
	BackType     uint8      `json:"backType"`
	BackValue    float64    `json:"backValue"`
	TriggerPrice float64    `json:"triggerPrice"`
	TriggerType  uint8      `json:"triggerType"`
	OrderID      FuturesID  `json:"orderId"`
	ErrorCode    uint64     `json:"errorCode"`
	State        uint8      `json:"state"`
	PositionMode uint8      `json:"positionMode"`
	ReduceOnly   bool       `json:"reduceOnly"`
	CreateTime   types.Time `json:"createTime"`
	UpdateTime   types.Time `json:"updateTime"`
}
