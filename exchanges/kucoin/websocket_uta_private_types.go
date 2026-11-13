package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAWebsocketOrder is a UTA order push, sent on each change to an order
type UTAWebsocketOrder struct {
	OrderID       string `json:"oi"`
	ClientOrderID string `json:"ci"`
	// OrderStatus is 0 not triggered, 1 triggered, 2 live, 3 filled, 4 partially filled, 5 cancelled or 6 partially
	// filled with the rest cancelled
	OrderStatus uint64 `json:"os"`
	EventType   string `json:"eT"`
	TradeType   string `json:"tT"`
	Symbol      string `json:"s"`
	Side        string `json:"S"`
	// PositionSide applies in futures hedge mode only
	PositionSide  string `json:"pS"`
	OrderType     string `json:"oT"` // codespell:ignore ot
	LiquidityRole string `json:"lR"`
	// OrderSource is USER, on UTA futures orders
	OrderSource string       `json:"oS"`
	Price       types.Number `json:"p"`
	MarginMode  string       `json:"mM"`
	TradeID     UTAID        `json:"ti"`
	Quantity    types.Number `json:"q"`
	// QuantityUnit is BASECCY, QUOTECCY or UNIT (contracts)
	QuantityUnit    string        `json:"qU"`
	FilledSize      types.Number  `json:"fS"`
	LastFilledSize  types.Number  `json:"lS"`
	LastFilledPrice types.Number  `json:"ls"`
	AveragePrice    types.Number  `json:"aP"`
	Fee             types.Number  `json:"f"`
	FeeCurrency     currency.Code `json:"fC"`
	Tax             types.Number  `json:"t"`
	CancelReason    string        `json:"cR"`
	// CancelledSize arrives as a string, or as a bare number in one of the docs' examples
	CancelledSize              types.Number `json:"cS"`
	RemainingSize              types.Number `json:"rS"`
	TriggerDirection           string       `json:"tD"`
	TriggerPrice               types.Number `json:"tP"`
	TriggerPriceType           string       `json:"tPT"`
	TakeProfitTriggerPrice     types.Number `json:"pP"`
	TakeProfitTriggerPriceType string       `json:"pPT"`
	TakeProfitOrderPrice       types.Number `json:"pOP"`
	StopLossTriggerPrice       types.Number `json:"lP"`
	StopLossTriggerPriceType   string       `json:"lPT"`
	StopLossOrderPrice         types.Number `json:"lOP"`
	// TriggerOrderID is the conditional order that placed this order
	TriggerOrderID      string `json:"toi"` // codespell:ignore toi
	SelfTradePrevention string `json:"stp"`
	ReduceOnly          bool   `json:"rO"`
	TimeInForce         string `json:"tIF"`
	PostOnly            bool   `json:"pO"`
	// CreationTimestamp is when the order left the gateway, before it was booked
	CreationTimestamp types.Time `json:"O"`
	UpdateTimestamp   types.Time `json:"U"`
	BrokerTag         string     `json:"bT"`
}

// UTAWebsocketExecutionLite is a UTA execution lite push, the execution push without its fee and fill details
type UTAWebsocketExecutionLite struct {
	TradeID       UTAID        `json:"ti"`
	Symbol        string       `json:"s"`
	Price         types.Number `json:"p"`
	Size          types.Number `json:"q"`
	Side          string       `json:"S"`
	OrderID       string       `json:"oi"`
	ClientOrderID string       `json:"ci"`
	OrderType     string       `json:"oT"` // codespell:ignore ot
	LiquidityRole string       `json:"lR"`
	// ExecutionTimestamp is when the match engine last updated the execution
	ExecutionTimestamp types.Time `json:"E"`
}

// UTAWebsocketExecution is a UTA execution push, sent when one of the account's orders trades
type UTAWebsocketExecution struct {
	UTAWebsocketExecutionLite
	Fee         types.Number  `json:"f"`
	FeeCurrency currency.Code `json:"fC"`
	// FillType is NORMAL, LIQUID, SETTLEMENT or ADL
	FillType string `json:"fT"`
	// FillPNL is the closed profit and loss of a futures fill closing a position; empty otherwise
	FillPNL types.Number `json:"fP"`
}

// UTAWebsocketBalance is a UTA balance push, sent when the unified account's balance of a currency changes
type UTAWebsocketBalance struct {
	Sequence        uint64        `json:"E"`
	UpdateTimestamp types.Time    `json:"U"`
	Currency        currency.Code `json:"c"`
	Equity          types.Number  `json:"e"`
	Balance         types.Number  `json:"b"`
	Available       types.Number  `json:"a"`
	Hold            types.Number  `json:"h"`
	// Liability is the amount borrowed, for UTA futures
	Liability                types.Number `json:"l"`
	TotalCrossMargin         types.Number `json:"tCM"`
	CrossPositionMargin      types.Number `json:"cPM"`
	CrossOrderMargin         types.Number `json:"cOM"`
	CrossUnrealisedPNL       types.Number `json:"cUP"`
	IsolatedPositionMargin   types.Number `json:"iPM"`
	IsolatedOrderMargin      types.Number `json:"iOM"`
	IsolatedFundingFeeMargin types.Number `json:"iFF"`
	IsolatedUnrealisedPNL    types.Number `json:"iUP"`
	// CollateralStatus is the platform-level collateral status of the currency: 1, 2 or 3
	CollateralStatus string `json:"cS"`
}

// UTAWebsocketPosition is a UTA futures position push, sent when an order or event such as ADL changes a position
type UTAWebsocketPosition struct {
	PositionID string `json:"pi"`
	Symbol     string `json:"s"`
	MarginMode string `json:"mM"`
	// Size is negative for a short position
	Size           types.Number `json:"q"`
	EntryPrice     types.Number `json:"eP"`
	PositionValue  types.Number `json:"pV"`
	PositionMargin types.Number `json:"pM"`
	MarkPrice      types.Number `json:"mP"`
	// LiquidationPrice is an estimate
	LiquidationPrice      types.Number `json:"lP"`
	BankruptPrice         types.Number `json:"bP"`
	Leverage              types.Number `json:"l"`
	UnrealisedPNL         types.Number `json:"uPL"`
	RealisedPNL           types.Number `json:"rPL"`
	InitialMargin         types.Number `json:"iM"`
	MaintenanceMarginRate types.Number `json:"mmr"`
	MaintenanceMargin     types.Number `json:"mtM"`
	// RiskRatio is the risk ratio of an isolated position; liquidation starts at 1
	RiskRatio types.Number `json:"r"`
	// ADL is the position's auto-deleveraging ranking percentile
	ADL               types.Number `json:"adl"`
	UpdateTimestamp   types.Time   `json:"U"`
	CreationTimestamp types.Time   `json:"O"`
}

// UTAWebsocketLiquidationWarning is a UTA liquidation warning push, sent while the account's risk ratio is 80% or
// more: on a change of risk event type, and every minute otherwise
type UTAWebsocketLiquidationWarning struct {
	// EventType is MARGIN_CALL, REDUCE_ONLY, LIQUIDATION_WARNING or FORCE_LIQUIDATION
	EventType string       `json:"eT"`
	RiskRatio types.Number `json:"r"`
	// AdjustedEquity and the margins, equity and liability are in USD
	AdjustedEquity    types.Number `json:"a"`
	InitialMargin     types.Number `json:"iM"`
	MaintenanceMargin types.Number `json:"mM"`
	AvailableMargin   types.Number `json:"aM"`
	Equity            types.Number `json:"e"`
	Liability         types.Number `json:"l"`
	UpdateTimestamp   types.Time   `json:"U"`
}

// UTAWebsocketLeverage is a UTA leverage push, sent when the account changes its leverage: of a futures symbol, or of
// a currency for cross margin
type UTAWebsocketLeverage struct {
	Symbol     string        `json:"s"`
	Leverage   types.Number  `json:"l"`
	Currency   currency.Code `json:"c"`
	MarginMode string        `json:"mM"`
	TradeType  string        `json:"tT"`
}
