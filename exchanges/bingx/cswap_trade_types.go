package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceCoinMarginedOrderRequest holds the parameters of a coin-M perpetual order. LIMIT orders need Quantity and Price,
// MARKET orders Quantity, and conditional orders StopPrice
type PlaceCoinMarginedOrderRequest struct {
	Symbol currency.Pair
	Type   OrderType
	Side   Side
	// PositionSide is BOTH in one-way mode, and LONG or SHORT in hedge mode
	PositionSide PositionSide
	Price        float64
	// Quantity is the number of contracts
	Quantity    float64
	StopPrice   float64
	WorkingType WorkingType
	// StopLoss places a STOP or STOP_MARKET order with the order, and TakeProfit a TAKE_PROFIT or TAKE_PROFIT_MARKET one
	StopLoss   *CoinMarginedTakeProfitStopLossRequest
	TakeProfit *CoinMarginedTakeProfitStopLossRequest
	// ClientOrderID is 1 to 40 characters, unique per order, which BingX converts to lowercase
	ClientOrderID string
	TimeInForce   TimeInForce
}

// CoinMarginedTakeProfitStopLossRequest holds a take profit or stop loss order placed with a coin-M perpetual order
type CoinMarginedTakeProfitStopLossRequest struct {
	Type        OrderType   `json:"type"`
	StopPrice   float64     `json:"stopPrice,omitempty"`
	Price       float64     `json:"price,omitempty"`
	WorkingType WorkingType `json:"workingType,omitempty"`
}

// CoinMarginedOrderData holds the order a coin-M order endpoint replies with
type CoinMarginedOrderData[T any] struct {
	Order *T `json:"order"`
}

// PlaceCoinMarginedOrderResponse holds a coin-M perpetual order as placed
type PlaceCoinMarginedOrderResponse struct {
	OrderID      NumericID    `json:"orderId"`
	Symbol       string       `json:"symbol"`
	PositionSide PositionSide `json:"positionSide"`
	Side         Side         `json:"side"`
	Type         OrderType    `json:"type"`
	Price        float64      `json:"price"`
	// Quantity is the number of contracts
	Quantity      float64     `json:"quantity"`
	StopPrice     float64     `json:"stopPrice"`
	WorkingType   WorkingType `json:"workingType"`
	ClientOrderID string      `json:"clientOrderId"`
	TimeInForce   TimeInForce `json:"timeInForce"`
}

// CoinMarginedTradeCommissionRateResponse holds the account's coin-M perpetual commission rates
type CoinMarginedTradeCommissionRateResponse struct {
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
	MakerCommissionRate types.Number `json:"makerCommissionRate"`
}

// CoinMarginedLeverageResponse holds a coin-M perpetual symbol's leverage for long and short positions
type CoinMarginedLeverageResponse struct {
	Symbol               string       `json:"symbol"`
	LongLeverage         uint64       `json:"longLeverage"`
	ShortLeverage        uint64       `json:"shortLeverage"`
	MaxLongLeverage      uint64       `json:"maxLongLeverage"`
	MaxShortLeverage     uint64       `json:"maxShortLeverage"`
	AvailableLongVolume  types.Number `json:"availableLongVol"`
	AvailableShortVolume types.Number `json:"availableShortVol"`
}

// CancelAllCoinMarginedOrdersResponse holds the orders a coin-M cancel all request cancelled and those it failed to
type CancelAllCoinMarginedOrdersResponse struct {
	Success []CoinMarginedOrderResponse      `json:"success"`
	Failed  []CoinMarginedCancelOrderFailure `json:"failed"`
}

// CoinMarginedCancelOrderFailure holds an order a coin-M cancel all request failed to cancel
type CoinMarginedCancelOrderFailure struct {
	OrderID NumericID `json:"orderId"`
	Code    uint64    `json:"code"`
	Message string    `json:"msg"`
}

// CloseAllCoinMarginedPositionsResponse holds the outcome of closing every coin-M perpetual position
type CloseAllCoinMarginedPositionsResponse struct {
	// Success holds the IDs of the orders closing the positions
	Success []NumericID                        `json:"success"`
	Failed  []CoinMarginedClosePositionFailure `json:"failed"`
}

// CoinMarginedClosePositionFailure holds a coin-M perpetual position that failed to close
type CoinMarginedClosePositionFailure struct {
	PositionID   NumericID `json:"positionId"`
	ErrorCode    uint64    `json:"errCode"`
	ErrorMessage string    `json:"errorMsg"`
}

// CoinMarginedPositionResponse holds a coin-M perpetual position
type CoinMarginedPositionResponse struct {
	Symbol       string       `json:"symbol"`
	PositionID   NumericID    `json:"positionId"`
	PositionSide PositionSide `json:"positionSide"`
	Isolated     bool         `json:"isolated"`
	// PositionAmount and AvailableAmount count contracts
	PositionAmount     types.Number `json:"positionAmt"`
	AvailableAmount    types.Number `json:"availableAmt"`
	UnrealisedProfit   types.Number `json:"unrealizedProfit"`
	InitialMargin      types.Number `json:"initialMargin"`
	LiquidationPrice   float64      `json:"liquidationPrice"`
	AveragePrice       types.Number `json:"avgPrice"`
	Leverage           uint64       `json:"leverage"`
	MarkPrice          types.Number `json:"markPrice"`
	RiskRate           types.Number `json:"riskRate"`
	MaxMarginReduction types.Number `json:"maxMarginReduction"`
	UpdateTime         types.Time   `json:"updateTime"`
}

// CoinMarginedAccountAssetResponse holds a coin-M perpetual account asset
type CoinMarginedAccountAssetResponse struct {
	Asset            currency.Code `json:"asset"`
	Balance          types.Number  `json:"balance"`
	Equity           types.Number  `json:"equity"`
	UnrealisedProfit types.Number  `json:"unrealizedProfit"`
	AvailableMargin  types.Number  `json:"availableMargin"`
	UsedMargin       types.Number  `json:"usedMargin"`
	FreezedMargin    types.Number  `json:"freezedMargin"`
	ShortUID         string        `json:"shortUid"`
}

// CoinMarginedForceOrdersRequest holds the parameters of a coin-M perpetual force order query
type CoinMarginedForceOrdersRequest struct {
	Symbol        currency.Pair
	AutoCloseType AutoCloseType
	StartTime     time.Time
	EndTime       time.Time
	// Limit is at most 100 and 50 by default
	Limit uint64
}

// CoinMarginedOrderTradeDetailRequest holds the parameters of a coin-M perpetual order trade detail query
type CoinMarginedOrderTradeDetailRequest struct {
	OrderID uint64
	// PageIndex starts at 1, the default
	PageIndex uint64
	// PageSize is at most 1000 and 100 by default
	PageSize uint64
}

// CoinMarginedOrderTradeDetailResponse holds a trade filling a coin-M perpetual order
type CoinMarginedOrderTradeDetailResponse struct {
	OrderID      NumericID    `json:"orderId"`
	Symbol       string       `json:"symbol"`
	Type         OrderType    `json:"type"`
	Side         Side         `json:"side"`
	PositionSide PositionSide `json:"positionSide"`
	TradeID      NumericID    `json:"tradeId"`
	// Volume counts contracts
	Volume       types.Number `json:"volume"`
	BaseQuantity float64      `json:"baseQty"`
	TradePrice   types.Number `json:"tradePrice"`
	// Amount is the trade's value in USD
	Amount      types.Number  `json:"amount"`
	RealisedPNL types.Number  `json:"realizedPnl"`
	Commission  types.Number  `json:"commission"`
	Currency    currency.Code `json:"currency"`
	Buyer       bool          `json:"buyer"`
	Maker       bool          `json:"maker"`
	TradeTime   types.Time    `json:"tradeTime"`
}

// CoinMarginedOrderListResponse holds the orders a coin-M order query replies with
type CoinMarginedOrderListResponse struct {
	Orders []CoinMarginedOrderResponse `json:"orders"`
}

// CoinMarginedOrderResponse holds a coin-M perpetual order as the cancel and order query endpoints report it. Its
// quantities count contracts
type CoinMarginedOrderResponse struct {
	Symbol  string    `json:"symbol"`
	OrderID NumericID `json:"orderId"`
	Side    Side      `json:"side"`
	// PositionSide is LONG, SHORT or BOTH, which some replies spell Long or Short
	PositionSide     string       `json:"positionSide"`
	Type             OrderType    `json:"type"`
	Quantity         float64      `json:"quantity"`
	OriginalQuantity types.Number `json:"origQty"`
	Price            types.Number `json:"price"`
	ExecutedQuantity types.Number `json:"executedQty"`
	AveragePrice     types.Number `json:"avgPrice"`
	CumulativeQuote  types.Number `json:"cumQuote"`
	StopPrice        types.Number `json:"stopPrice"`
	Profit           types.Number `json:"profit"`
	Commission       types.Number `json:"commission"`
	// Status is an order status such as PENDING, FILLED or CANCELLED, which some replies spell Pending
	Status        string     `json:"status"`
	Time          types.Time `json:"time"`
	UpdateTime    types.Time `json:"updateTime"`
	ClientOrderID string     `json:"clientOrderId"`
	// Leverage is the order's leverage followed by X, such as 15X, when set
	Leverage               string                         `json:"leverage"`
	TakeProfit             CoinMarginedTakeProfitStopLoss `json:"takeProfit"`
	StopLoss               CoinMarginedTakeProfitStopLoss `json:"stopLoss"`
	AdvanceAttr            uint64                         `json:"advanceAttr"`
	PositionID             NumericID                      `json:"positionID"`
	TakeProfitEntrustPrice float64                        `json:"takeProfitEntrustPrice"`
	StopLossEntrustPrice   float64                        `json:"stopLossEntrustPrice"`
	OrderType              string                         `json:"orderType"`
	WorkingType            WorkingType                    `json:"workingType"`
	PostOnly               bool                           `json:"postOnly"`
	ReduceOnly             types.Boolean                  `json:"reduceOnly"`
	TimeInForce            TimeInForce                    `json:"timeInForce"`
}

// CoinMarginedTakeProfitStopLoss holds the take profit or stop loss order attached to a coin-M perpetual order
type CoinMarginedTakeProfitStopLoss struct {
	Type           OrderType   `json:"type"`
	Quantity       float64     `json:"quantity"`
	StopPrice      float64     `json:"stopPrice"`
	Price          float64     `json:"price"`
	WorkingType    WorkingType `json:"workingType"`
	StopGuaranteed string      `json:"stopGuaranteed"`
}

// CoinMarginedOrderHistoryRequest holds the parameters of a coin-M perpetual order history query
type CoinMarginedOrderHistoryRequest struct {
	Symbol currency.Pair
	// OrderID returns the orders placed after it
	OrderID uint64
	// StartTime and EndTime span at most seven days; BingX returns the last seven days when both are zero
	StartTime time.Time
	EndTime   time.Time
	// Limit is required, and at most 1000
	Limit uint64
}

// CoinMarginedMarginTypeResponse holds a coin-M perpetual symbol's margin type
type CoinMarginedMarginTypeResponse struct {
	Symbol     string     `json:"symbol"`
	MarginType MarginType `json:"marginType"`
}

// AdjustCoinMarginedIsolatedMarginRequest holds the parameters of a coin-M isolated margin adjustment
type AdjustCoinMarginedIsolatedMarginRequest struct {
	Symbol currency.Pair
	// Amount is the margin to add or remove, in the contract's margin coin
	Amount float64
	Type   CoinMarginedMarginAdjustmentType
	// PositionSide is LONG or SHORT
	PositionSide PositionSide
}
