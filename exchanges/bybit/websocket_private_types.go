package bybit

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// errInvalidExtraFees is returned for extra fees that are neither a list nor a JSON encoded list
var errInvalidExtraFees = errors.New("extra fees must be a list or a JSON encoded list")

// WebsocketPositionUpdate is a push from the position topic or one of its categorised topics (Position)
type WebsocketPositionUpdate struct {
	ID           string              `json:"id"`
	Topic        string              `json:"topic"`
	CreationTime types.Time          `json:"creationTime"`
	Data         []WebsocketPosition `json:"data"`
}

// WebsocketPosition is a position. Its amounts keep their exact decimal value because they feed futures.Position
type WebsocketPosition struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	// Side is empty for an empty position
	Side          string              `json:"side"`
	Size          types.PreciseNumber `json:"size"`
	PositionIndex uint64              `json:"positionIdx"`
	PositionValue types.PreciseNumber `json:"positionValue"`
	// RiskID and RiskLimitValue are 0 in portfolio margin mode, where risk limits do not apply
	RiskID         uint64              `json:"riskId"`
	RiskLimitValue types.PreciseNumber `json:"riskLimitValue"`
	EntryPrice     types.PreciseNumber `json:"entryPrice"`
	MarkPrice      types.PreciseNumber `json:"markPrice"`
	// Leverage is empty in portfolio margin mode
	Leverage                  types.PreciseNumber `json:"leverage"`
	BreakEvenPrice            types.PreciseNumber `json:"breakEvenPrice"` // Linear and inverse only
	AutoAddMargin             types.Boolean       `json:"autoAddMargin"`
	PositionInitialMargin     types.PreciseNumber `json:"positionIM"`
	PositionMaintenanceMargin types.PreciseNumber `json:"positionMM"`
	// LiquidationPrice is an estimate under cross margin, and empty when beyond the price limits or in portfolio margin
	// mode
	LiquidationPrice    types.PreciseNumber `json:"liqPrice"`
	TakeProfit          types.PreciseNumber `json:"takeProfit"`
	StopLoss            types.PreciseNumber `json:"stopLoss"`
	TrailingStop        types.PreciseNumber `json:"trailingStop"`
	UnrealisedPNL       types.PreciseNumber `json:"unrealisedPnl"`
	CurrentRealisedPNL  types.PreciseNumber `json:"curRealisedPnl"`
	SessionAveragePrice types.PreciseNumber `json:"sessionAvgPrice"` // USDC contracts only
	Delta               types.PreciseNumber `json:"delta"`           // Option position pushes only
	Gamma               types.PreciseNumber `json:"gamma"`           // Option position pushes only
	Vega                types.PreciseNumber `json:"vega"`            // Option position pushes only
	Theta               types.PreciseNumber `json:"theta"`           // Option position pushes only
	// NetDeltaRatio is empty unless delta neutral mode is enabled
	NetDeltaRatio types.PreciseNumber `json:"netDeltaRatio"`
	// CumulativeRealisedPNL covers all time for perpetuals and futures, and the current holding for options
	CumulativeRealisedPNL types.PreciseNumber `json:"cumRealisedPnl"`
	PositionStatus        string              `json:"positionStatus"`
	ADLRankIndicator      uint64              `json:"adlRankIndicator"`
	IsReduceOnly          bool                `json:"isReduceOnly"`
	CreatedTime           types.Time          `json:"createdTime"`
	UpdatedTime           types.Time          `json:"updatedTime"`
	OpenTime              types.Time          `json:"openTime"`
	// Sequence is -1 for a symbol that was never traded
	Sequence                               int64               `json:"seq"`
	MaintenanceMarginRateSystemUpdatedTime types.Time          `json:"mmrSysUpdatedTime"`
	LeverageSystemUpdatedTime              types.Time          `json:"leverageSysUpdatedTime"`
	PositionInitialMarginByMarkPrice       types.PreciseNumber `json:"positionIMByMp"`
	PositionMaintenanceMarginByMarkPrice   types.PreciseNumber `json:"positionMMByMp"`
	// TPSLMode is deprecated and always Full
	TPSLMode string `json:"tpslMode"`
	// BustPrice is deprecated and always empty
	BustPrice types.PreciseNumber `json:"bustPrice"`
	// PositionBalance is deprecated; PositionInitialMargin replaces it
	PositionBalance types.PreciseNumber `json:"positionBalance"`
	// TradeMode is deprecated and always 0
	TradeMode uint64 `json:"tradeMode"`
}

// WebsocketExecutionUpdate is a push from the execution topic or one of its categorised topics (Execution)
type WebsocketExecutionUpdate struct {
	ID           string               `json:"id"`
	Topic        string               `json:"topic"`
	CreationTime types.Time           `json:"creationTime"`
	Data         []WebsocketExecution `json:"data"`
}

// WebsocketExecution is an execution of one of the account's orders
type WebsocketExecution struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	// IsLeverage is 1 for a borrowed spot order and 0 otherwise; Bybit sends an empty string for categories that cannot
	// borrow, which a boolean type would reject
	IsLeverage     string       `json:"isLeverage"`
	OrderID        string       `json:"orderId"`
	OrderLinkID    string       `json:"orderLinkId"`
	Side           string       `json:"side"`
	OrderPrice     types.Number `json:"orderPrice"`
	OrderQuantity  types.Number `json:"orderQty"`
	LeavesQuantity types.Number `json:"leavesQty"`
	// CreateType is not sent for spot and options
	CreateType        string       `json:"createType"`
	OrderType         string       `json:"orderType"`
	StopOrderType     string       `json:"stopOrderType"`
	ExecutionFee      types.Number `json:"execFee"`
	ExecutionID       string       `json:"execId"`
	ExecutionPrice    types.Number `json:"execPrice"`
	ExecutionQuantity types.Number `json:"execQty"`
	// ExecutionPNL is the profit and loss of a closing execution, matching cashFlow in the transaction log
	ExecutionPNL           types.Number `json:"execPnl"`
	ExecutionType          string       `json:"execType"`
	ExecutionValue         types.Number `json:"execValue"`
	ExecutionTime          types.Time   `json:"execTime"`
	IsMaker                bool         `json:"isMaker"`
	FeeRate                types.Number `json:"feeRate"`
	TradeImpliedVolatility types.Number `json:"tradeIv"`         // Options only
	MarkImpliedVolatility  types.Number `json:"markIv"`          // Options only
	MarkPrice              types.Number `json:"markPrice"`       // Options only
	IndexPrice             types.Number `json:"indexPrice"`      // Options only
	UnderlyingPrice        types.Number `json:"underlyingPrice"` // Options only
	BlockTradeID           string       `json:"blockTradeId"`
	ClosedSize             types.Number `json:"closedSize"`
	// ExtraFees holds tax fees charged to Indian, Indonesian site and EU site users
	ExtraFees ExtraFees `json:"extraFees"`
	Sequence  uint64    `json:"seq"`
	// MarketUnit is not documented but sent with each execution
	MarketUnit  string        `json:"marketUnit"`
	FeeCurrency currency.Code `json:"feeCurrency"`
}

// ExtraFees is a list of extra trading fees. Bybit sends an empty string when there are none
type ExtraFees []ExtraFee

// ExtraFee is a tax fee charged on top of the trading fee
type ExtraFee struct {
	FeeCoin    currency.Code `json:"feeCoin"`
	FeeType    string        `json:"feeType"`
	SubFeeType string        `json:"subFeeType"`
	FeeRate    types.Number  `json:"feeRate"`
	Fee        types.Number  `json:"fee"`
}

// WebsocketFastExecutionUpdate is a push from the execution.fast topic or one of its categorised topics (Fast
// Execution)
type WebsocketFastExecutionUpdate struct {
	Topic        string                   `json:"topic"`
	CreationTime types.Time               `json:"creationTime"`
	Data         []WebsocketFastExecution `json:"data"`
}

// WebsocketFastExecution is a trade execution of one of the account's orders, pushed with lower latency and fewer
// fields than an execution
type WebsocketFastExecution struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	OrderID  string `json:"orderId"`
	IsMaker  bool   `json:"isMaker"`
	// OrderLinkID is empty for a maker trade
	OrderLinkID       string       `json:"orderLinkId"`
	ExecutionID       string       `json:"execId"`
	ExecutionPrice    types.Number `json:"execPrice"`
	ExecutionQuantity types.Number `json:"execQty"`
	Side              string       `json:"side"`
	ExecutionTime     types.Time   `json:"execTime"`
	Sequence          uint64       `json:"seq"`
}

// WebsocketOrderUpdate is a push from the order topic or one of its categorised topics (Order)
type WebsocketOrderUpdate struct {
	ID           string           `json:"id"`
	Topic        string           `json:"topic"`
	CreationTime types.Time       `json:"creationTime"`
	Data         []WebsocketOrder `json:"data"`
}

// WebsocketOrder is an order
type WebsocketOrder struct {
	Category    string `json:"category"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	// ParentOrderLinkID links a take profit or stop loss order to the order it was attached to
	ParentOrderLinkID string `json:"parentOrderLinkId"`
	// IsLeverage is 1 for a borrowed spot order and 0 otherwise; Bybit sends an empty string for categories that cannot
	// borrow, which a boolean type would reject
	IsLeverage   string       `json:"isLeverage"`
	BlockTradeID string       `json:"blockTradeId"`
	Symbol       string       `json:"symbol"`
	Price        types.Number `json:"price"`
	// BrokerOrderPrice is for EU liquidity providers
	BrokerOrderPrice types.Number `json:"brokerOrderPrice"`
	Quantity         types.Number `json:"qty"`
	Side             string       `json:"side"`
	PositionIndex    uint64       `json:"positionIdx"`
	OrderStatus      string       `json:"orderStatus"`
	// CreateType is not sent for spot and options
	CreateType                 string       `json:"createType"`
	CancelType                 string       `json:"cancelType"`
	RejectReason               string       `json:"rejectReason"`
	AveragePrice               types.Number `json:"avgPrice"`
	LeavesQuantity             types.Number `json:"leavesQty"`
	LeavesValue                types.Number `json:"leavesValue"`
	CumulativeExecutedQuantity types.Number `json:"cumExecQty"`
	CumulativeExecutedValue    types.Number `json:"cumExecValue"`
	// CumulativeExecutedFee is deprecated for linear and spot orders, which report CumulativeFeeDetail instead
	CumulativeExecutedFee types.Number `json:"cumExecFee"`
	ClosedPNL             types.Number `json:"closedPnl"`
	// FeeCurrency is deprecated; CumulativeFeeDetail names each fee's currency
	FeeCurrency            currency.Code `json:"feeCurrency"`
	TimeInForce            string        `json:"timeInForce"`
	OrderType              string        `json:"orderType"`
	StopOrderType          string        `json:"stopOrderType"`
	OCOTriggerBy           string        `json:"ocoTriggerBy"`
	OrderImpliedVolatility types.Number  `json:"orderIv"`
	MarketUnit             string        `json:"marketUnit"`
	SlippageToleranceType  string        `json:"slippageToleranceType"`
	SlippageTolerance      types.Number  `json:"slippageTolerance"`
	// TriggerPrice is the activation price of a trailing stop order
	TriggerPrice       types.Number `json:"triggerPrice"`
	TakeProfit         types.Number `json:"takeProfit"`
	StopLoss           types.Number `json:"stopLoss"`
	TPSLMode           string       `json:"tpslMode"`
	TPLimitPrice       types.Number `json:"tpLimitPrice"`
	SLLimitPrice       types.Number `json:"slLimitPrice"`
	TPTriggerBy        string       `json:"tpTriggerBy"`
	SLTriggerBy        string       `json:"slTriggerBy"`
	TriggerDirection   uint64       `json:"triggerDirection"`
	TriggerBy          string       `json:"triggerBy"`
	LastPriceOnCreated types.Number `json:"lastPriceOnCreated"`
	ReduceOnly         bool         `json:"reduceOnly"`
	CloseOnTrigger     bool         `json:"closeOnTrigger"`
	PlaceType          string       `json:"placeType"`
	SMPType            string       `json:"smpType"`
	// SMPGroup changed from a number to a string in August 2026, so either is accepted
	SMPGroup    types.Number `json:"smpGroup"`
	SMPOrderID  string       `json:"smpOrderId"`
	CreatedTime types.Time   `json:"createdTime"`
	UpdatedTime types.Time   `json:"updatedTime"`
	// CumulativeFeeDetail holds the cumulative fees of linear and spot orders by currency
	CumulativeFeeDetail map[currency.Code]types.Number `json:"cumFeeDetail"`
	RPITakerAccess      bool                           `json:"rpiTakerAccess"`
	RPIMatchedQuantity  types.Number                   `json:"rpiMatchedQty"`
}

// WebsocketWalletUpdate is a push from the wallet topic (Wallet)
type WebsocketWalletUpdate struct {
	ID           string            `json:"id"`
	Topic        string            `json:"topic"`
	CreationTime types.Time        `json:"creationTime"`
	Data         []WebsocketWallet `json:"data"`
}

// WebsocketWallet is the balance of a unified trading account; its account wide values are in USD and do not apply to
// isolated margin
type WebsocketWallet struct {
	AccountType                  string       `json:"accountType"`
	AccountInitialMarginRate     types.Number `json:"accountIMRate"`
	AccountMaintenanceMarginRate types.Number `json:"accountMMRate"`
	TotalEquity                  types.Number `json:"totalEquity"`
	TotalWalletBalance           types.Number `json:"totalWalletBalance"`
	TotalMarginBalance           types.Number `json:"totalMarginBalance"`
	TotalAvailableBalance        types.Number `json:"totalAvailableBalance"`
	TotalPerpetualUnrealisedPNL  types.Number `json:"totalPerpUPL"`
	TotalInitialMargin           types.Number `json:"totalInitialMargin"`
	TotalMaintenanceMargin       types.Number `json:"totalMaintenanceMargin"`
	// The by mark price fields repeat the fields above, which Bybit now calculates the same way
	AccountInitialMarginRateByMarkPrice     types.Number `json:"accountIMRateByMp"`
	AccountMaintenanceMarginRateByMarkPrice types.Number `json:"accountMMRateByMp"`
	TotalInitialMarginByMarkPrice           types.Number `json:"totalInitialMarginByMp"`
	TotalMaintenanceMarginByMarkPrice       types.Number `json:"totalMaintenanceMarginByMp"`
	// AccountLTV is deprecated
	AccountLTV types.Number        `json:"accountLTV"`
	Coins      []WalletBalanceCoin `json:"coin"`
}

// WebsocketGreeksUpdate is a push from the greeks topic (Greek)
type WebsocketGreeksUpdate struct {
	ID           string            `json:"id"`
	Topic        string            `json:"topic"`
	CreationTime types.Time        `json:"creationTime"`
	Data         []WebsocketGreeks `json:"data"`
}

// WebsocketGreeks holds the account's option greeks for a base coin
type WebsocketGreeks struct {
	BaseCoin   currency.Code `json:"baseCoin"`
	TotalDelta types.Number  `json:"totalDelta"`
	TotalGamma types.Number  `json:"totalGamma"`
	TotalVega  types.Number  `json:"totalVega"`
	TotalTheta types.Number  `json:"totalTheta"`
}

// WebsocketStrategyUpdate is a push from the strategy topic (Strategy)
type WebsocketStrategyUpdate struct {
	ID           string              `json:"id"`
	Topic        string              `json:"topic"`
	CreationTime types.Time          `json:"creationTime"`
	Data         []WebsocketStrategy `json:"data"`
}

// WebsocketStrategy is a TWAP, iceberg, chase or POV strategy order
type WebsocketStrategy struct {
	StrategyID   string `json:"strategyId"`
	StrategyType string `json:"strategyType"`
	// Category is a strategy product type such as UTA_USDT, not an order category
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Size     types.Number `json:"size"`
	Side     string       `json:"side"`
	// Duration is in seconds, for TWAP strategies only
	Duration uint64 `json:"duration"`
	// Status is 2 running, 3 terminated, 4 terminated with orders unfilled, 5 paused or 6 not triggered yet
	Status uint64 `json:"status"`
	// TerminateType is 0 not terminated, 1 stopped by the user, 2 completed or 3 insufficient balance
	TerminateType   uint64 `json:"terminateType"`
	TerminateRemark string `json:"terminateRemark"`
	// ExecutedDuration is in seconds
	ExecutedDuration     uint64       `json:"executedDuration"`
	ExecutedSize         types.Number `json:"executedSize"`
	ExecutedAveragePrice types.Number `json:"executedAvgPrice"`
	ExecutedStartTime    types.Time   `json:"executedStartTimeE3"`
	// ExecutedEndTime is zero until the strategy ends
	ExecutedEndTime types.Time   `json:"executedEndTimeE3"`
	CreatedTime     types.Time   `json:"createdTimeE3"`
	UpdatedTime     types.Time   `json:"updatedTimeE3"`
	IsRandom        bool         `json:"isRandom"` // TWAP only
	ReduceOnly      bool         `json:"reduceOnly"`
	LimitPrice      types.Number `json:"limitPrice"`
	TriggerCount    uint64       `json:"triggerCount"`
	TradingCount    uint64       `json:"tradingCount"`
	ChaseDistance   types.Number `json:"chaseDistance"` // Chase and iceberg only
	// ChasePercentE4 is in basis points, so 100 is 1%; chase and iceberg only
	ChasePercentE4  uint64       `json:"ChasePercentE4"`
	MaxChasePrice   types.Number `json:"maxChasePrice"`   // Chase and iceberg only
	ChaseOrderPrice types.Number `json:"chaseOrderPrice"` // Chase only
	StrategyPrefer  string       `json:"strategyPrefer"`
	// Interval is the time between orders in seconds, for TWAP strategies only
	Interval uint64 `json:"interval"`
	// LeverageType is 0 normal or 1 margin, for spot only
	LeverageType uint64        `json:"leverageType"`
	PostOnly     types.Boolean `json:"postOnly"` // Iceberg only
	TriggerPrice types.Number  `json:"triggerPrice"`
	IsTriggered  bool          `json:"isTriggered"`
	StrategyTP   types.Number  `json:"strategyTp"`
	StrategySL   types.Number  `json:"strategySl"`
	// OrderType is 1 market or 2 limit
	OrderType        string       `json:"orderType"`
	OrderPriceOffset types.Number `json:"orderPriceOffset"`
	// PositionValue is the total value of a value based strategy, empty otherwise
	PositionValue       types.Number `json:"positionValue"`
	FilledPositionValue types.Number `json:"filledPositionValue"`
}

// UnmarshalJSON decodes a list of extra fees, which Bybit sends as an empty string when there are none and some
// endpoints send JSON encoded in a string
func (f *ExtraFees) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*f = nil
		return nil
	}
	if len(data) != 0 && data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		if encoded == "" {
			*f = nil
			return nil
		}
		data = []byte(encoded)
	}
	if len(data) == 0 || data[0] != '[' {
		return fmt.Errorf("%w: %s", errInvalidExtraFees, data)
	}
	var fees []ExtraFee
	if err := json.Unmarshal(data, &fees); err != nil {
		return err
	}
	*f = fees
	return nil
}
