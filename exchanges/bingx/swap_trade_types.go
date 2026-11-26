package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapOrderRequest holds a USDT-M perpetual order for PlaceSwapOrder, PlaceSwapTestOrder and PlaceSwapBatchOrders.
// Which fields an order needs depends on its type: LIMIT takes Quantity and Price; MARKET Quantity; STOP, TAKE_PROFIT
// and TRIGGER_LIMIT Quantity, StopPrice and Price; STOP_MARKET, TAKE_PROFIT_MARKET and TRIGGER_MARKET Quantity and
// StopPrice, ClosePosition orders included; TRAILING_STOP_MARKET and TRAILING_TP_SL Price or PriceRate
type SwapOrderRequest struct {
	Symbol currency.Pair
	Type   OrderType
	Side   Side
	// PositionSide is BOTH in one-way position mode and LONG or SHORT in hedge mode; BingX takes an empty one as LONG
	PositionSide PositionSide
	// ReduceOnly applies in one-way position mode only
	ReduceOnly bool
	// Price is the trailing distance of TRAILING_STOP_MARKET and TRAILING_TP_SL orders
	Price float64
	// Quantity is in the contract's coin
	Quantity float64
	// QuoteOrderQuantity is the order size in the quote currency, which BingX ignores when Quantity is set
	QuoteOrderQuantity float64
	StopPrice          float64
	// PriceRate is the trailing rate of TRAILING_STOP_MARKET and TRAILING_TP_SL orders, at most 1
	PriceRate   float64
	WorkingType WorkingType
	// StopLoss attaches a STOP_MARKET or STOP order
	StopLoss *SwapTakeProfitStopLossRequest
	// TakeProfit attaches a TAKE_PROFIT_MARKET or TAKE_PROFIT order
	TakeProfit *SwapTakeProfitStopLossRequest
	// ClientOrderID is 1 to 40 characters, which BingX converts to lowercase; LIMIT and MARKET orders only
	ClientOrderID string
	TimeInForce   TimeInForce
	// ClosePosition closes the whole position once a STOP_MARKET or TAKE_PROFIT_MARKET order triggers, and cannot be
	// sent with ReduceOnly
	ClosePosition bool
	// ActivationPrice is the price that activates a TRAILING_STOP_MARKET or TRAILING_TP_SL order, the latest price by
	// default
	ActivationPrice float64
	// StopGuaranteed true or cutfee makes BingX ignore Quantity
	StopGuaranteed StopGuaranteed
	// PositionID is required only to close a position in separate isolated margin mode
	PositionID uint64
}

// SwapTakeProfitStopLossRequest is the takeProfit or stopLoss parameter of a USDT-M perpetual order, which BingX takes
// as a JSON object in a string
type SwapTakeProfitStopLossRequest struct {
	Type           OrderType      `json:"type"`
	StopPrice      float64        `json:"stopPrice"`
	Price          float64        `json:"price,omitempty"`
	WorkingType    WorkingType    `json:"workingType,omitempty"`
	Quantity       float64        `json:"quantity,omitempty"`
	StopGuaranteed StopGuaranteed `json:"stopGuaranteed,omitempty"`
}

// SwapBatchOrderRequest is one order of a batchOrders list, in the JSON form BingX takes; PlaceSwapBatchOrders builds it
// from a SwapOrderRequest
type SwapBatchOrderRequest struct {
	Symbol             string         `json:"symbol"`
	Type               OrderType      `json:"type"`
	Side               Side           `json:"side"`
	PositionSide       PositionSide   `json:"positionSide,omitempty"`
	ReduceOnly         string         `json:"reduceOnly,omitempty"`
	Price              float64        `json:"price,omitempty"`
	Quantity           float64        `json:"quantity,omitempty"`
	QuoteOrderQuantity float64        `json:"quoteOrderQty,omitempty"`
	StopPrice          float64        `json:"stopPrice,omitempty"`
	PriceRate          float64        `json:"priceRate,omitempty"`
	WorkingType        WorkingType    `json:"workingType,omitempty"`
	StopLoss           string         `json:"stopLoss,omitempty"`
	TakeProfit         string         `json:"takeProfit,omitempty"`
	ClientOrderID      string         `json:"clientOrderId,omitempty"`
	TimeInForce        TimeInForce    `json:"timeInForce,omitempty"`
	ClosePosition      string         `json:"closePosition,omitempty"`
	ActivationPrice    float64        `json:"activationPrice,omitempty"`
	StopGuaranteed     StopGuaranteed `json:"stopGuaranteed,omitempty"`
	PositionID         uint64         `json:"positionId,omitempty"`
}

// SwapNewOrderResponse holds an order Place Order, Test Order or Place Multiple Orders accepted
type SwapNewOrderResponse struct {
	Symbol           string         `json:"symbol"`
	OrderID          NumericID      `json:"orderId"`
	Side             Side           `json:"side"`
	PositionSide     PositionSide   `json:"positionSide"`
	Type             OrderType      `json:"type"`
	ClientOrderID    string         `json:"clientOrderId"`
	WorkingType      WorkingType    `json:"workingType"`
	ReduceOnly       types.Boolean  `json:"reduceOnly"`
	StopGuaranteed   StopGuaranteed `json:"stopGuaranteed"`
	Status           string         `json:"status"`
	AveragePrice     types.Number   `json:"avgPrice"` // MARKET orders only
	ExecutedQuantity types.Number   `json:"executedQty"`
}

// SwapNewOrderDataResponse holds the data member of a Place Order or Test Order reply
type SwapNewOrderDataResponse struct {
	Order *SwapNewOrderResponse `json:"order"`
}

// SwapNewOrdersDataResponse holds the data member of a Place Multiple Orders reply
type SwapNewOrdersDataResponse struct {
	Orders []SwapNewOrderResponse `json:"orders"`
}

// ModifySwapOrderRequest holds the parameters of Modify Order, which changes the quantity of a LIMIT order
type ModifySwapOrderRequest struct {
	Symbol currency.Pair
	// OrderID or ClientOrderID identifies the order
	OrderID       uint64
	ClientOrderID string
	Quantity      float64
}

// ModifySwapOrderResponse holds the order Modify Order changed
type ModifySwapOrderResponse struct {
	OrderID       NumericID `json:"orderId"`
	ClientOrderID string    `json:"clientOrderId"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
}

// CloseAllSwapPositionsResponse holds the outcome of Close All Positions
type CloseAllSwapPositionsResponse struct {
	// Success lists the orders that close the positions
	Success []NumericID `json:"success"`
	// Failed lists the positions BingX could not close, in a form its documentation leaves out
	Failed []json.RawMessage `json:"failed"`
}

// SwapOrderResponse holds a USDT-M perpetual order as the order queries and cancellations report it; fields an
// endpoint does not send stay zero
type SwapOrderResponse struct {
	Symbol       string       `json:"symbol"`
	OrderID      NumericID    `json:"orderId"`
	Side         Side         `json:"side"`
	PositionSide PositionSide `json:"positionSide"`
	Type         OrderType    `json:"type"`
	// OriginalQuantity is in the contract's coin
	OriginalQuantity types.Number `json:"origQty"`
	// Price is the trailing distance of TRAILING_STOP_MARKET and TRAILING_TP_SL orders
	Price            types.Number                   `json:"price"`
	ExecutedQuantity types.Number                   `json:"executedQty"`
	AveragePrice     types.Number                   `json:"avgPrice"`
	CumulativeQuote  types.Number                   `json:"cumQuote"`
	StopPrice        types.Number                   `json:"stopPrice"`
	Profit           types.Number                   `json:"profit"`
	Commission       types.Number                   `json:"commission"`
	Status           string                         `json:"status"`
	Time             types.Time                     `json:"time"`
	UpdateTime       types.Time                     `json:"updateTime"`
	ClientOrderID    string                         `json:"clientOrderId"`
	Leverage         string                         `json:"leverage"` // such as 20X
	TakeProfit       SwapTakeProfitStopLossResponse `json:"takeProfit"`
	StopLoss         SwapTakeProfitStopLossResponse `json:"stopLoss"`
	AdvanceAttribute uint64                         `json:"advanceAttr"`
	PositionID       NumericID                      `json:"positionID"`
	// TakeProfitEntrustPrice and StopLossEntrustPrice are the prices of the attached take profit and stop loss orders
	TakeProfitEntrustPrice float64        `json:"takeProfitEntrustPrice"`
	StopLossEntrustPrice   float64        `json:"stopLossEntrustPrice"`
	OrderType              string         `json:"orderType"`
	WorkingType            WorkingType    `json:"workingType"`
	StopGuaranteed         StopGuaranteed `json:"stopGuaranteed"`
	TriggerOrderID         NumericID      `json:"triggerOrderId"`
	IsTWAP                 bool           `json:"isTwap"`
	// MainOrderID is the TWAP order the order belongs to
	MainOrderID          string        `json:"mainOrderId"`
	ReduceOnly           types.Boolean `json:"reduceOnly"`
	PostOnly             bool          `json:"postOnly"`
	TrailingStopRate     float64       `json:"trailingStopRate"`
	TrailingStopDistance float64       `json:"trailingStopDistance"`
	// PriceRate is set only for TRAILING_STOP_MARKET and TRAILING_TP_SL orders placed with a rate
	PriceRate       types.Number  `json:"priceRate"`
	ActivationPrice types.Number  `json:"actPrice"`
	ClosePosition   types.Boolean `json:"closePosition"`
}

// SwapTakeProfitStopLossResponse holds the take profit or stop loss attached to a USDT-M perpetual order
type SwapTakeProfitStopLossResponse struct {
	Type           OrderType      `json:"type"`
	Quantity       float64        `json:"quantity"`
	StopPrice      float64        `json:"stopPrice"`
	Price          float64        `json:"price"`
	WorkingType    WorkingType    `json:"workingType"`
	StopGuaranteed StopGuaranteed `json:"stopGuaranteed"`
}

// SwapOrderDataResponse holds the data member of a reply carrying one order
type SwapOrderDataResponse struct {
	Order *SwapOrderResponse `json:"order"`
}

// SwapOrdersDataResponse holds the data member of a reply carrying a list of orders
type SwapOrdersDataResponse struct {
	Orders []SwapOrderResponse `json:"orders"`
}

// SwapCancelOrdersResponse holds the outcome of Cancel Multiple Orders or Cancel All Open Orders: the cancelled orders
// and the orders BingX could not cancel
type SwapCancelOrdersResponse struct {
	Success []SwapOrderResponse              `json:"success"`
	Failed  []SwapCancelOrderFailureResponse `json:"failed"`
}

// SwapCancelOrderFailureResponse holds an order BingX could not cancel and why
type SwapCancelOrderFailureResponse struct {
	OrderID       NumericID `json:"orderId"`
	ClientOrderID string    `json:"clientOrderId"`
	// ErrorCode is an int64 to match APIError.Code
	ErrorCode    int64  `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// SwapMarginTypeResponse holds the margin type of a USDT-M perpetual contract
type SwapMarginTypeResponse struct {
	MarginType MarginType `json:"marginType"`
}

// SwapLeverageResponse holds the leverage of a USDT-M perpetual contract and the positions it allows
type SwapLeverageResponse struct {
	LongLeverage          uint64       `json:"longLeverage"`
	ShortLeverage         uint64       `json:"shortLeverage"`
	MaxLongLeverage       uint64       `json:"maxLongLeverage"`
	MaxShortLeverage      uint64       `json:"maxShortLeverage"`
	AvailableLongVolume   types.Number `json:"availableLongVol"`
	AvailableShortVolume  types.Number `json:"availableShortVol"`
	AvailableLongValue    types.Number `json:"availableLongVal"`
	AvailableShortValue   types.Number `json:"availableShortVal"`
	MaxPositionLongValue  types.Number `json:"maxPositionLongVal"`
	MaxPositionShortValue types.Number `json:"maxPositionShortVal"`
}

// SetSwapLeverageResponse holds the leverage Set Leverage applied and the positions it allows
type SetSwapLeverageResponse struct {
	Leverage              uint64       `json:"leverage"`
	Symbol                string       `json:"symbol"`
	AvailableLongVolume   types.Number `json:"availableLongVol"`
	AvailableShortVolume  types.Number `json:"availableShortVol"`
	AvailableLongValue    types.Number `json:"availableLongVal"`
	AvailableShortValue   types.Number `json:"availableShortVal"`
	MaxPositionLongValue  types.Number `json:"maxPositionLongVal"`
	MaxPositionShortValue types.Number `json:"maxPositionShortVal"`
}

// SwapForceOrdersRequest holds the filters of User's Force Orders
type SwapForceOrdersRequest struct {
	Symbol currency.Pair
	// Currency is USDT or USDC
	Currency      currency.Code
	AutoCloseType AutoCloseType
	StartTime     time.Time
	EndTime       time.Time
	// Limit defaults to 50, at most 100
	Limit uint64
}

// SwapOrderHistoryRequest holds the filters of Query Order History, which spans at most seven days
type SwapOrderHistoryRequest struct {
	Symbol currency.Pair
	// Currency is USDT, the default, or USDC
	Currency currency.Code
	// OrderID returns the orders after it
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// Limit defaults to 500, at most 1000
	Limit uint64
}

// ModifySwapIsolatedPositionMarginRequest holds the parameters of Modify Isolated Position Margin
type ModifySwapIsolatedPositionMarginRequest struct {
	Symbol currency.Pair
	Amount float64
	Type   SwapMarginAdjustmentType
	// PositionSide is LONG or SHORT
	PositionSide PositionSide
	// PositionID takes precedence over PositionSide
	PositionID uint64
}

// ModifySwapIsolatedPositionMarginResponse holds the margin change Modify Isolated Position Margin made
type ModifySwapIsolatedPositionMarginResponse struct {
	Amount     float64                  `json:"amount"`
	Type       SwapMarginAdjustmentType `json:"type"`
	PositionID NumericID                `json:"positionId"`
}

// SwapHistoricalTransactionOrdersRequest holds the filters of Query Historical Transaction Orders
type SwapHistoricalTransactionOrdersRequest struct {
	Symbol currency.Pair
	// Currency is USDT, the default, or USDC
	Currency    currency.Code
	OrderID     uint64
	TradingUnit TradingUnit
	StartTime   time.Time
	EndTime     time.Time
}

// SwapHistoricalTransactionOrderResponse holds a filled order from Query Historical Transaction Orders
type SwapHistoricalTransactionOrderResponse struct {
	// FilledTimeUTC is the fill time BingX sends as filledTm, in UTC
	FilledTimeUTC         time.Time     `json:"filledTm"`
	Symbol                string        `json:"symbol"`
	Volume                types.Number  `json:"volume"`
	Price                 types.Number  `json:"price"`
	Amount                types.Number  `json:"amount"`
	Commission            types.Number  `json:"commission"`
	Currency              currency.Code `json:"currency"`
	OrderID               NumericID     `json:"orderId"`
	LiquidatedPrice       types.Number  `json:"liquidatedPrice"`       // liquidation orders only
	LiquidatedMarginRatio types.Number  `json:"liquidatedMarginRatio"` // liquidation orders only
	FilledTime            time.Time     `json:"filledTime"`
	WorkingType           WorkingType   `json:"workingType"`
	Side                  Side          `json:"side"`
	Type                  OrderType     `json:"type"`
	PositionSide          PositionSide  `json:"positionSide"`
	ClientOrderID         string        `json:"clientOrderID"`
	OnlyOnePosition       bool          `json:"onlyOnePosition"`
}

// SwapHistoricalTransactionOrdersDataResponse holds the data member of a Query Historical Transaction Orders reply
type SwapHistoricalTransactionOrdersDataResponse struct {
	FillOrders []SwapHistoricalTransactionOrderResponse `json:"fill_orders"`
}

// SwapPositionModeResponse holds the position mode of the USDT-M perpetual account
type SwapPositionModeResponse struct {
	// DualSidePosition is true in hedge mode and false in one-way mode
	DualSidePosition types.Boolean `json:"dualSidePosition"`
}

// CancelReplaceSwapOrderRequest holds the parameters of Cancel an Existing Order and Send a New Order: the order to
// cancel and the order to place, whose PositionSide is required. The documentation lists every order field for the new
// order but QuoteOrderQuantity
type CancelReplaceSwapOrderRequest struct {
	SwapOrderRequest
	CancelReplaceMode CancelReplaceMode
	// CancelOrderID or CancelClientOrderID identifies the order to cancel; CancelOrderID takes precedence
	CancelOrderID       uint64
	CancelClientOrderID string
	CancelRestrictions  CancelRestrictions
}

// SwapBatchCancelReplaceOrderRequest is one element of a batch cancel-replace list, in the JSON form BingX takes;
// CancelReplaceSwapBatchOrders builds it from a CancelReplaceSwapOrderRequest
type SwapBatchCancelReplaceOrderRequest struct {
	CancelReplaceMode   CancelReplaceMode  `json:"cancelReplaceMode"`
	CancelOrderID       uint64             `json:"cancelOrderId,omitempty"`
	CancelClientOrderID string             `json:"cancelClientOrderId,omitempty"`
	CancelRestrictions  CancelRestrictions `json:"cancelRestrictions,omitempty"`
	SwapBatchOrderRequest
}

// SwapCancelReplaceOrderResponse holds both halves of a cancel-replace, each with its own result, so a failed half is
// reported beside the other
type SwapCancelReplaceOrderResponse struct {
	CancelResult     types.Boolean                     `json:"cancelResult"`
	CancelMessage    string                            `json:"cancelMsg"`
	CancelResponse   SwapCancelReplaceCancelResponse   `json:"cancelResponse"`
	ReplaceResult    types.Boolean                     `json:"replaceResult"`
	ReplaceMessage   string                            `json:"replaceMsg"`
	NewOrderResponse SwapCancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// SwapCancelReplaceCancelResponse holds the order a cancel-replace cancelled
type SwapCancelReplaceCancelResponse struct {
	CancelClientOrderID string    `json:"cancelClientOrderId"`
	CancelOrderID       NumericID `json:"cancelOrderId"`
	SwapOrderResponse
	OnlyOnePosition bool `json:"onlyOnePosition"`
}

// SwapCancelReplaceNewOrderResponse holds the order a cancel-replace placed
type SwapCancelReplaceNewOrderResponse struct {
	OrderID       NumericID    `json:"orderId"`
	Symbol        string       `json:"symbol"`
	PositionSide  PositionSide `json:"positionSide"`
	Side          Side         `json:"side"`
	Type          OrderType    `json:"type"`
	Price         float64      `json:"price"`
	Quantity      float64      `json:"quantity"`
	StopPrice     float64      `json:"stopPrice"`
	WorkingType   WorkingType  `json:"workingType"`
	ClientOrderID string       `json:"clientOrderId"`
	TimeInForce   TimeInForce  `json:"timeInForce"`
	PriceRate     float64      `json:"priceRate"`
	// StopLoss and TakeProfit hold the attached orders as JSON objects in strings
	StopLoss   string        `json:"stopLoss"`
	TakeProfit string        `json:"takeProfit"`
	ReduceOnly types.Boolean `json:"reduceOnly"`
}

// SwapCancelAllAfterResponse holds the state of the USDT-M perpetual cancel all after countdown
type SwapCancelAllAfterResponse struct {
	TriggerTime types.Time `json:"triggerTime"`
	Status      string     `json:"status"`
	Note        string     `json:"note"`
}

// CloseSwapPositionResponse holds the order Close Position by Position ID placed
type CloseSwapPositionResponse struct {
	OrderID    NumericID `json:"orderId"`
	PositionID NumericID `json:"positionId"`
	Symbol     string    `json:"symbol"`
	// Side is Ask or Bid, and Type such as Market: these differ from the order side and type values
	Side             string       `json:"side"`
	Type             string       `json:"type"`
	PositionSide     PositionSide `json:"positionSide"`
	OriginalQuantity types.Number `json:"origQty"`
}

// SwapAllOrdersRequest holds the filters of All Orders
type SwapAllOrdersRequest struct {
	Symbol currency.Pair
	// OrderID returns the orders after it
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// Limit defaults to 500, at most 1000
	Limit uint64
}

// SwapMaintenanceMarginRatioResponse holds a position value tier of a USDT-M perpetual contract and its maintenance
// margin ratio
type SwapMaintenanceMarginRatioResponse struct {
	Tier                   string       `json:"tier"`
	Symbol                 string       `json:"symbol"`
	MinPositionValue       types.Number `json:"minPositionVal"`
	MaxPositionValue       types.Number `json:"maxPositionVal"`
	MaintenanceMarginRatio types.Number `json:"maintMarginRatio"`
	MaintenanceAmount      types.Number `json:"maintAmount"`
}

// SwapHistoricalTransactionDetailsRequest holds the filters of Query Historical Transaction Details. Pages hold the
// fills in ascending fill time; keep the other filters while incrementing PageIndex
type SwapHistoricalTransactionDetailsRequest struct {
	Symbol currency.Pair
	// Currency is USDT, the default, or USDC
	Currency currency.Code
	OrderID  uint64
	// PageIndex starts at 1, the default
	PageIndex uint64
	// PageSize defaults to 50, at most 1000
	PageSize uint64
	// StartTime is exclusive and EndTime inclusive; without StartTime, the window starts seven days ago, and
	// StartTime alone returns nothing
	StartTime time.Time
	EndTime   time.Time
}

// SwapHistoricalTransactionDetailsResponse holds a page of Query Historical Transaction Details
type SwapHistoricalTransactionDetailsResponse struct {
	FillHistoryOrders []SwapHistoricalTransactionDetailResponse `json:"fill_history_orders"`
	Total             uint64                                    `json:"total"`
}

// SwapHistoricalTransactionDetailResponse holds a fill from Query Historical Transaction Details
type SwapHistoricalTransactionDetailResponse struct {
	Symbol          string        `json:"symbol"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Role            string        `json:"role"` // taker or maker
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Price           types.Number  `json:"price"`
	OrderID         NumericID     `json:"orderId"`
	TradeID         string        `json:"tradeId"`
	FilledTime      time.Time     `json:"filledTime"`
	ClientOrderID   string        `json:"clientOrderID"`
	RealisedPNL     types.Number  `json:"realisedPNL"`
	Side            Side          `json:"side"`
	PositionSide    PositionSide  `json:"positionSide"`
}

// SwapPositionHistoryRequest holds the filters of Query Position History, which spans at most 90 days
type SwapPositionHistoryRequest struct {
	Symbol currency.Pair
	// Currency is USDT, the default, or USDC; a symbol's settlement currency takes precedence
	Currency   currency.Code
	PositionID uint64
	StartTime  time.Time
	EndTime    time.Time
	// PageIndex starts at 0, the default
	PageIndex uint64
	// PageSize defaults to 100, at most 1000
	PageSize uint64
}

// SwapPositionHistoryResponse holds a closed USDT-M perpetual position
type SwapPositionHistoryResponse struct {
	// PositionID is a string: the documented example masks some of its digits
	PositionID          string       `json:"positionId"`
	Symbol              string       `json:"symbol"`
	Isolated            bool         `json:"isolated"`
	PositionSide        PositionSide `json:"positionSide"`
	OpenTime            types.Time   `json:"openTime"`
	UpdateTime          types.Time   `json:"updateTime"`
	AveragePrice        types.Number `json:"avgPrice"`
	AverageClosePrice   types.Number `json:"avgClosePrice"`
	RealizedProfit      types.Number `json:"realizedProfit"`
	NetProfit           types.Number `json:"netProfit"`
	PositionAmount      types.Number `json:"positionAmt"`
	ClosePositionAmount types.Number `json:"closePositionAmt"`
	Leverage            uint64       `json:"leverage"`
	CloseAllPositions   bool         `json:"closeAllPositions"`
	PositionCommission  types.Number `json:"positionCommission"`
	TotalFunding        types.Number `json:"totalFunding"`
}

// SwapIsolatedMarginChangeHistoryRequest holds the filters of Isolated Margin Change History, which covers the last 30
// days
type SwapIsolatedMarginChangeHistoryRequest struct {
	Symbol     currency.Pair
	PositionID uint64
	StartTime  time.Time
	EndTime    time.Time
	// PageIndex starts at 1, the default
	PageIndex uint64
	// PageSize is at most 100
	PageSize uint64
}

// SwapIsolatedMarginChangeHistoryResponse holds a page of Isolated Margin Change History
type SwapIsolatedMarginChangeHistoryResponse struct {
	Records []SwapIsolatedMarginChangeResponse `json:"records"`
	Total   uint64                             `json:"total"`
}

// SwapIsolatedMarginChangeResponse holds a change to an isolated position's margin
type SwapIsolatedMarginChangeResponse struct {
	Symbol     string    `json:"symbol"`
	PositionID NumericID `json:"positionId"`
	// ChangeReason is such as ManualMarginAddition, OpenPosition, Liquidation or FundingFeeSettlement
	ChangeReason      string       `json:"changeReason"`
	MarginChange      types.Number `json:"marginChange"`
	MarginAfterChange types.Number `json:"marginAfterChange"`
	Time              types.Time   `json:"time"`
}

// ApplySwapVSTResponse holds the VST Apply VST added to or removed from the demo account
type ApplySwapVSTResponse struct {
	AdjustType VSTAdjustType `json:"adjustType"`
	Amount     types.Number  `json:"amount"`
}

// SwapTWAPOrderRequest holds the parameters of Place TWAP Order, which splits an order into child orders placed over
// up to 24 hours
type SwapTWAPOrderRequest struct {
	Symbol currency.Pair
	Side   Side
	// PositionSide is LONG or SHORT
	PositionSide PositionSide
	PriceType    TWAPPriceType
	// PriceVariance is a price difference in USDT for TWAPPriceTypeConstant and a slippage percentage for
	// TWAPPriceTypePercentage
	PriceVariance float64
	// TriggerPrice limits the child orders: a buy places them while the market price is below it, a sell while above
	TriggerPrice float64
	// Interval between child orders, 5 to 120 seconds, sent in whole seconds
	Interval time.Duration
	// AmountPerOrder is the largest child order
	AmountPerOrder float64
	TotalAmount    float64
}

// PlaceSwapTWAPOrderResponse holds the TWAP order Place TWAP Order created
type PlaceSwapTWAPOrderResponse struct {
	MainOrderID string `json:"mainOrderId"`
}

// SwapTWAPOrdersResponse holds a page of TWAP orders
type SwapTWAPOrdersResponse struct {
	List  []SwapTWAPOrderResponse `json:"list"`
	Total uint64                  `json:"total"`
}

// SwapTWAPOrderResponse holds a TWAP order
type SwapTWAPOrderResponse struct {
	Symbol         string        `json:"symbol"`
	MainOrderID    string        `json:"mainOrderId"`
	Side           Side          `json:"side"`
	PositionSide   PositionSide  `json:"positionSide"`
	PriceType      TWAPPriceType `json:"priceType"`
	PriceVariance  types.Number  `json:"priceVariance"`
	TriggerPrice   types.Number  `json:"triggerPrice"`
	Interval       uint64        `json:"interval"` // seconds
	AmountPerOrder types.Number  `json:"amountPerOrder"`
	TotalAmount    types.Number  `json:"totalAmount"`
	// OrderStatus is such as New, Running, Canceling, Filled, PartiallyFilled, Pending, Cancelled or Failed
	OrderStatus      string       `json:"orderStatus"`
	ExecutedQuantity types.Number `json:"executedQty"`
	Duration         uint64       `json:"duration"`    // seconds; the order is cancelled once they elapse
	MaxDuration      uint64       `json:"maxDuration"` // seconds
	CreatedTime      types.Time   `json:"createdTime"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// SwapTWAPHistoricalOrdersRequest holds the filters of Query TWAP Historical Orders
type SwapTWAPHistoricalOrdersRequest struct {
	Symbol currency.Pair
	// PageIndex starts at 1
	PageIndex uint64
	// PageSize is at most 1000
	PageSize  uint64
	StartTime time.Time
	EndTime   time.Time
}

// SwapMultiAssetsModeResponse holds the multi-assets mode of the USDT-M perpetual account
type SwapMultiAssetsModeResponse struct {
	AssetMode AssetMode `json:"assetMode"`
}

// SwapMultiAssetsRuleResponse holds an asset that can margin USDT-M perpetuals in multi-assets mode
type SwapMultiAssetsRuleResponse struct {
	MarginAssets currency.Code `json:"marginAssets"`
	// LTV and CollateralValueRatio are percentages such as 95.00%: the first values the asset for available margin,
	// the second for the risk rate
	LTV                  string `json:"ltv"`
	CollateralValueRatio string `json:"collateralValueRatio"`
	// MaxTransfer is the most that can be transferred in, empty without a limit
	MaxTransfer types.Number `json:"maxTransfer"`
	IndexPrice  types.Number `json:"indexPrice"` // in USD
}

// SwapMultiAssetsMarginResponse holds an asset margining the USDT-M perpetual account in multi-assets mode
type SwapMultiAssetsMarginResponse struct {
	Currency          currency.Code `json:"currency"`
	TotalAmount       types.Number  `json:"totalAmount"`
	AvailableTransfer types.Number  `json:"availableTransfer"`
	// LatestMortgageAmount is the latest collateral amount available
	LatestMortgageAmount types.Number `json:"latestMortgageAmount"`
}

// ReverseSwapPositionRequest holds the parameters of One-Click Reverse Position
type ReverseSwapPositionRequest struct {
	Type   ReverseType
	Symbol currency.Pair
	// TriggerPrice and WorkingType are required for ReverseTypeTriggerReverse
	TriggerPrice float64
	WorkingType  WorkingType
	// PositionID is the position to reverse, which the documented example sends although the parameter list leaves it
	// out
	PositionID uint64
}

// ReverseSwapPositionResponse holds the position One-Click Reverse Position opened
type ReverseSwapPositionResponse struct {
	Type          ReverseType `json:"type"`
	PositionID    NumericID   `json:"positionId"`
	NewPositionID NumericID   `json:"newPositionId"`
	Symbol        string      `json:"symbol"`
	// PositionSide is Long or Short, which differs from the position side values in case
	PositionSide     string       `json:"positionSide"`
	Isolated         bool         `json:"isolated"`
	PositionAmount   types.Number `json:"positionAmt"`
	AvailableAmount  types.Number `json:"availableAmt"`
	UnrealizedProfit types.Number `json:"unrealizedProfit"`
	// RealisedProfit is the documented field, which the documented example sends as RealizedProfit
	RealisedProfit   types.Number `json:"realisedProfit"`
	RealizedProfit   types.Number `json:"realizedProfit"`
	InitialMargin    types.Number `json:"initialMargin"`
	Margin           types.Number `json:"margin"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	AveragePrice     types.Number `json:"avgPrice"`
	Leverage         string       `json:"leverage"` // such as 10X
	PositionValue    types.Number `json:"positionValue"`
	MarkPrice        types.Number `json:"markPrice"`
	// RiskRate is a percentage; the position is reduced or liquidated at 100
	RiskRate           types.Number `json:"riskRate"`
	MaxMarginReduction types.Number `json:"maxMarginReduction"`
	PNLRatio           types.Number `json:"pnlRatio"`
	UpdateTime         types.Time   `json:"updateTime"`
}

// SwapAutoAddMarginRequest holds the parameters of Hedge Mode Position - Automatic Margin Addition
type SwapAutoAddMarginRequest struct {
	Symbol     currency.Pair
	PositionID uint64
	// FunctionSwitch turns automatic margin addition on or off
	FunctionSwitch bool
	// Amount is the margin to add in USDT, required to turn the feature on
	Amount float64
}

// SwapAutoAddMarginResponse holds the automatic margin addition setting of a hedge mode position
type SwapAutoAddMarginResponse struct {
	Symbol         string        `json:"symbol"`
	PositionID     NumericID     `json:"positionId"`
	FunctionSwitch types.Boolean `json:"functionSwitch"`
	Amount         types.Number  `json:"amount"`
}
