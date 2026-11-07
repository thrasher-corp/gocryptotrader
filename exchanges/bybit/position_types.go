package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PositionInfoRequest holds the parameters of Get Position Info
type PositionInfoRequest struct {
	Category string
	// Symbol returns its position whether or not it is open, and takes priority over SettleCoin
	Symbol string
	// BaseCoin is for options only; without it every option position is returned
	BaseCoin currency.Code
	// SettleCoin returns the open positions of a settle coin; linear positions need it or Symbol
	SettleCoin currency.Code
	Limit      uint64
	Cursor     string
}

// PositionInfoResponse holds a page of positions from Get Position Info
type PositionInfoResponse struct {
	Category       string         `json:"category"`
	NextPageCursor string         `json:"nextPageCursor"`
	List           []PositionInfo `json:"list"`
}

// PositionInfo is a position from Get Position Info
type PositionInfo struct {
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex uint64 `json:"positionIdx"`
	// RiskID and RiskLimitValue are 0 in portfolio margin mode, where risk limits do not apply
	RiskID         uint64       `json:"riskId"`
	RiskLimitValue types.Number `json:"riskLimitValue"`
	Symbol         string       `json:"symbol"`
	// Side is Buy for a long position and Sell for a short one; it is empty when there is no position
	Side           string        `json:"side"`
	Size           types.Number  `json:"size"`
	AveragePrice   types.Number  `json:"avgPrice"`
	PositionValue  types.Number  `json:"positionValue"`
	AutoAddMargin  types.Boolean `json:"autoAddMargin"`
	PositionStatus string        `json:"positionStatus"`
	// Leverage is empty in portfolio margin mode
	Leverage       types.Number `json:"leverage"`
	BreakEvenPrice types.Number `json:"breakEvenPrice"`
	MarkPrice      types.Number `json:"markPrice"`
	// LiquidationPrice is an estimate for cross margin positions, and empty in portfolio margin mode or outside the
	// symbol's price limits
	LiquidationPrice types.Number `json:"liqPrice"`
	// LiquidationPriceByMarkPrice is not documented, but Bybit sends it
	LiquidationPriceByMarkPrice          types.Number `json:"liqPriceByMp"`
	PositionInitialMargin                types.Number `json:"positionIM"`
	PositionInitialMarginByMarkPrice     types.Number `json:"positionIMByMp"`
	PositionMaintenanceMargin            types.Number `json:"positionMM"`
	PositionMaintenanceMarginByMarkPrice types.Number `json:"positionMMByMp"`
	TakeProfit                           types.Number `json:"takeProfit"`
	StopLoss                             types.Number `json:"stopLoss"`
	// TrailingStop is the distance from the market price
	TrailingStop types.Number `json:"trailingStop"`
	// SessionAveragePrice is the average entry price of a USDC contract for the current settlement session
	SessionAveragePrice types.Number `json:"sessionAvgPrice"`
	Delta               types.Number `json:"delta"`
	Gamma               types.Number `json:"gamma"`
	Vega                types.Number `json:"vega"`
	Theta               types.Number `json:"theta"`
	// NetDeltaRatio is empty unless delta neutral mode is on
	NetDeltaRatio      types.Number `json:"netDeltaRatio"`
	UnrealisedPNL      types.Number `json:"unrealisedPnl"`
	CurrentRealisedPNL types.Number `json:"curRealisedPnl"`
	// CumulativeRealisedPNL is all time for perpetuals and futures, and meaningless for options
	CumulativeRealisedPNL types.Number `json:"cumRealisedPnl"`
	ADLRankIndicator      uint64       `json:"adlRankIndicator"`
	// CreatedTime is when the first position on the symbol was created
	CreatedTime types.Time `json:"createdTime"`
	UpdatedTime types.Time `json:"updatedTime"`
	OpenTime    types.Time `json:"openTime"`
	// Sequence links fills with position updates per symbol; it is -1 when the symbol has never been traded
	Sequence int64 `json:"seq"`
	// IsReduceOnly is true when Bybit lowered the risk limit and the position may only be reduced
	IsReduceOnly bool `json:"isReduceOnly"`
	// MaintenanceMarginRateSystemUpdatedTime and LeverageSystemUpdatedTime are when Bybit will lower, or lowered, the
	// position's maintenance margin rate and leverage
	MaintenanceMarginRateSystemUpdatedTime types.Time `json:"mmrSysUpdatedTime"`
	LeverageSystemUpdatedTime              types.Time `json:"leverageSysUpdatedTime"`
	// TPSLMode is deprecated and always Full
	TPSLMode string `json:"tpslMode"`
	// BustPrice is deprecated and always empty
	BustPrice types.Number `json:"bustPrice"`
	// PositionBalance is deprecated in favour of PositionInitialMargin
	PositionBalance types.Number `json:"positionBalance"`
	// TradeMode is deprecated and always 0; Get Account Info reports the margin mode
	TradeMode uint64 `json:"tradeMode"`
}

// SetLeverageRequest holds the parameters of Set Leverage. The buy and sell leverage must be equal in one-way mode and
// in cross margin mode
type SetLeverageRequest struct {
	Category     string
	Symbol       string
	BuyLeverage  float64
	SellLeverage float64
}

// FuturesLeverageResponse holds the leverage of futures symbols, from Get Futures Leverage
type FuturesLeverageResponse struct {
	Category string            `json:"category"`
	List     []FuturesLeverage `json:"list"`
}

// FuturesLeverage is a symbol's leverage, from Get Futures Leverage
type FuturesLeverage struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
	// Side is meaningless, as Bybit documents
	Side string `json:"side"`
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex uint64 `json:"positionIdx"`
}

// SwitchPositionModeRequest holds the parameters of Switch Position Mode
type SwitchPositionModeRequest struct {
	Category string
	// Symbol switches one symbol and takes priority over Coin, which switches every symbol it settles without a
	// position or open order
	Symbol string
	Coin   currency.Code
	// Mode is 0 for one-way (merged single) mode and 3 for hedge (both sides) mode
	Mode uint64
}

// TradingStopRequest holds the parameters of Set Trading Stop
type TradingStopRequest struct {
	Category string
	Symbol   string
	// TPSLMode is Full for the whole position, the only mode options support, or Partial
	TPSLMode string
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex uint64
	// TakeProfit, StopLoss and TrailingStop are left unchanged when nil, and cancelled when they point to zero.
	// TrailingStop is a distance from the market price
	TakeProfit   *float64
	StopLoss     *float64
	TrailingStop *float64
	TPTriggerBy  string
	SLTriggerBy  string
	// ActivePrice is the price that activates the trailing stop
	ActivePrice float64
	// TPSize and SLSize must be equal, in Partial mode
	TPSize       float64
	SLSize       float64
	TPLimitPrice float64
	SLLimitPrice float64
	TPOrderType  string
	SLOrderType  string
}

// SetAutoAddMarginRequest holds the parameters of Set Auto Add Margin, for isolated margin positions
type SetAutoAddMarginRequest struct {
	Category      string
	Symbol        string
	AutoAddMargin bool
	// PositionIndex is required in hedge mode: 1 for the buy side, 2 for the sell side
	PositionIndex uint64
}

// AddOrReduceMarginRequest holds the parameters of Add Or Reduce Margin, for isolated margin positions
type AddOrReduceMarginRequest struct {
	Category string
	Symbol   string
	// Margin is added when positive and removed when negative, to 4 decimal places
	Margin float64
	// PositionIndex is required in hedge mode: 1 for the buy side, 2 for the sell side
	PositionIndex uint64
}

// AddOrReduceMarginResponse holds the position after Add Or Reduce Margin
type AddOrReduceMarginResponse struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex             uint64        `json:"positionIdx"`
	RiskID                    uint64        `json:"riskId"`
	RiskLimitValue            types.Number  `json:"riskLimitValue"`
	Size                      types.Number  `json:"size"`
	AveragePrice              types.Number  `json:"avgPrice"`
	LiquidationPrice          types.Number  `json:"liqPrice"`
	BustPrice                 types.Number  `json:"bustPrice"`
	MarkPrice                 types.Number  `json:"markPrice"`
	PositionValue             types.Number  `json:"positionValue"`
	Leverage                  types.Number  `json:"leverage"`
	AutoAddMargin             types.Boolean `json:"autoAddMargin"`
	PositionStatus            string        `json:"positionStatus"`
	PositionInitialMargin     types.Number  `json:"positionIM"`
	PositionMaintenanceMargin types.Number  `json:"positionMM"`
	TakeProfit                types.Number  `json:"takeProfit"`
	StopLoss                  types.Number  `json:"stopLoss"`
	// TrailingStop is the distance from the market price
	TrailingStop          types.Number `json:"trailingStop"`
	UnrealisedPNL         types.Number `json:"unrealisedPnl"`
	CumulativeRealisedPNL types.Number `json:"cumRealisedPnl"`
	// CreatedTime is when the first position on the symbol was created
	CreatedTime types.Time `json:"createdTime"`
	UpdatedTime types.Time `json:"updatedTime"`
}

// ClosedPNLRequest holds the parameters of Get Closed PnL. Without times it returns the last 7 days, and the window
// between StartTime and EndTime may not exceed 7 days
type ClosedPNLRequest struct {
	Category  string
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// ClosedPNLResponse holds a page of closed profit and loss records from Get Closed PnL
type ClosedPNLResponse struct {
	Category       string      `json:"category"`
	NextPageCursor string      `json:"nextPageCursor"`
	List           []ClosedPNL `json:"list"`
}

// ClosedPNL is the profit and loss of an order that closed some or all of a position, from Get Closed PnL
type ClosedPNL struct {
	Symbol     string       `json:"symbol"`
	OrderID    string       `json:"orderId"`
	Side       string       `json:"side"`
	Quantity   types.Number `json:"qty"`
	OrderPrice types.Number `json:"orderPrice"`
	OrderType  string       `json:"orderType"`
	// ExecutionType is Trade, BustTrade, SessionSettlePnL, Settle, MovePosition, ForwardSplitSettle or
	// ReverseSplitSettle
	ExecutionType        string       `json:"execType"`
	ClosedSize           types.Number `json:"closedSize"`
	CumulativeEntryValue types.Number `json:"cumEntryValue"`
	AverageEntryPrice    types.Number `json:"avgEntryPrice"`
	CumulativeExitValue  types.Number `json:"cumExitValue"`
	AverageExitPrice     types.Number `json:"avgExitPrice"`
	ClosedPNL            types.Number `json:"closedPnl"`
	// FillCount is the number of fills of the order
	FillCount   types.Number `json:"fillCount"`
	Leverage    types.Number `json:"leverage"`
	OpenFee     types.Number `json:"openFee"`
	CloseFee    types.Number `json:"closeFee"`
	CreatedTime types.Time   `json:"createdTime"`
	UpdatedTime types.Time   `json:"updatedTime"`
}

// ClosedOptionsPositionsRequest holds the parameters of Get Closed Options Positions. Without times it returns the
// last day, and the window between StartTime and EndTime may not exceed 7 days
type ClosedOptionsPositionsRequest struct {
	Category  string
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// ClosedOptionsPositionsResponse holds a page of closed options positions from Get Closed Options Positions
type ClosedOptionsPositionsResponse struct {
	Category       string                  `json:"category"`
	NextPageCursor string                  `json:"nextPageCursor"`
	List           []ClosedOptionsPosition `json:"list"`
}

// ClosedOptionsPosition is a closed options position, from Get Closed Options Positions
type ClosedOptionsPosition struct {
	Symbol            string       `json:"symbol"`
	Side              string       `json:"side"`
	TotalOpenFee      types.Number `json:"totalOpenFee"`
	DeliveryFee       types.Number `json:"deliveryFee"`
	TotalCloseFee     types.Number `json:"totalCloseFee"`
	Quantity          types.Number `json:"qty"`
	CloseTime         types.Time   `json:"closeTime"`
	AverageExitPrice  types.Number `json:"avgExitPrice"`
	DeliveryPrice     types.Number `json:"deliveryPrice"`
	OpenTime          types.Time   `json:"openTime"`
	AverageEntryPrice types.Number `json:"avgEntryPrice"`
	TotalPNL          types.Number `json:"totalPnl"`
}

// MovePositionsRequest holds the parameters of Move Position, which moves positions between a master account and its
// sub-accounts, both unified trading accounts in one-way mode
type MovePositionsRequest struct {
	FromUID uint64
	ToUID   uint64
	// List holds up to 25 legs
	List []MovePositionLeg
}

// MovePositionLeg is one position move of a Move Position request
type MovePositionLeg struct {
	Category string
	Symbol   string
	// Price must be within 5% of the mark price for linear and inverse symbols
	Price float64
	// Side is the side FromUID trades
	Side     string
	Quantity float64
}

// MovePositionsResponse holds the block trade a Move Position request started
type MovePositionsResponse struct {
	BlockTradeID string `json:"blockTradeId"`
	// Status is Processing or Rejected
	Status string `json:"status"`
	// RejectParty is empty when the request passed validation, Taker or Maker when it was rejected, and bybit when
	// Bybit failed
	RejectParty string `json:"rejectParty"`
}

// MovePositionHistoryRequest holds the parameters of Get Move Position History. The window between StartTime and
// EndTime may not exceed 7 days
type MovePositionHistoryRequest struct {
	Category     string
	Symbol       string
	StartTime    time.Time
	EndTime      time.Time
	Status       string
	BlockTradeID string
	Limit        uint64
	Cursor       string
}

// MovePositionHistoryResponse holds a page of moved positions from Get Move Position History
type MovePositionHistoryResponse struct {
	List           []MovePositionRecord `json:"list"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// MovePositionRecord is one leg of a position move, from Get Move Position History
type MovePositionRecord struct {
	BlockTradeID string `json:"blockTradeId"`
	Category     string `json:"category"`
	OrderID      string `json:"orderId"`
	UserID       uint64 `json:"userId"`
	Symbol       string `json:"symbol"`
	// Side is from the taker's side
	Side     string       `json:"side"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	// ExecutionFee is in the base coin
	ExecutionFee types.Number `json:"execFee"`
	// Status is Processing, Filled or Rejected
	Status      string `json:"status"`
	ExecutionID string `json:"execId"`
	// ResultCode is 0 on success, when ResultMessage is empty
	ResultCode    int64      `json:"resultCode"`
	ResultMessage string     `json:"resultMessage"`
	CreatedAt     types.Time `json:"createdAt"`
	UpdatedAt     types.Time `json:"updatedAt"`
	// RejectParty is empty when the move filled, Taker or Maker when it was rejected, and bybit when Bybit failed
	RejectParty string `json:"rejectParty"`
}
