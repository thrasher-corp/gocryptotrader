package binance

import (
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UMOrder represents a portfolio margin.
type UMOrder struct {
	ClientOrderID           string               `json:"clientOrderId"`
	CumQty                  types.Number         `json:"cumQty"`
	CumQuote                types.Number         `json:"cumQuote"`
	ExecutedQty             types.Number         `json:"executedQty"`
	OrderID                 int64                `json:"orderId"`
	AvgPrice                types.Number         `json:"avgPrice"`
	OrigQty                 types.Number         `json:"origQty"`
	Price                   types.Number         `json:"price"`
	ReduceOnly              bool                 `json:"reduceOnly"`
	Side                    string               `json:"side"`
	PositionSide            string               `json:"positionSide"`
	Status                  string               `json:"status"`
	Symbol                  string               `json:"symbol"`
	TimeInForce             string               `json:"timeInForce"`
	Type                    string               `json:"type"`
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	GoodTillDate            int64                `json:"goodTillDate"`
	UpdateTime              convert.ExchangeTime `json:"updateTime"`
}

// UMOrderParam request parameters for UM order
type UMOrderParam struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	PositionSide            string  `json:"positionSide,omitempty"`
	OrderType               string  `json:"type"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	Quantity                float64 `json:"quantity,omitempty"`
	ReduceOnly              bool    `json:"reduceOnly,omitempty"`
	Price                   float64 `json:"price,omitempty"`
	NewClientOrderID        string  `json:"newClientOrderId,omitempty"`
	NewOrderRespType        string  `json:"newOrderRespType,omitempty"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode,omitempty"`
	GoodTillDate            int64   `json:"goodTillDate,omitempty"`
}

// MarginOrderParam represents request parameter for margin trade order
type MarginOrderParam struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	OrderType               string  `json:"type"`
	Amount                  float64 `json:"quantity"`
	QuoteOrderQty           float64 `json:"quoteOrderQty"`
	Price                   float64 `json:"price"`
	StopPrice               float64 `json:"stopPrice"` // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	NewClientOrderID        string  `json:"newClientOrderId"`
	NewOrderRespType        string  `json:"newOrderRespType"`
	IcebergQuantity         float64 `json:"icebergQty"`
	SideEffectType          string  `json:"sideEffectType"`
	TimeInForce             string  `json:"timeInForce"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode"`
}

// MarginOrderResp represents a margin order response.
type MarginOrderResp struct {
	Symbol                  string               `json:"symbol"`
	OrderID                 int64                `json:"orderId"`
	ClientOrderID           string               `json:"clientOrderId"`
	TransactTime            convert.ExchangeTime `json:"transactTime"`
	Price                   types.Number         `json:"price"`
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	OrigQty                 types.Number         `json:"origQty"`
	ExecutedQty             types.Number         `json:"executedQty"`
	CummulativeQuoteQty     types.Number         `json:"cummulativeQuoteQty"`
	Status                  string               `json:"status"`
	TimeInForce             string               `json:"timeInForce"`
	Type                    string               `json:"type"`
	Side                    string               `json:"side"`
	MarginBuyBorrowAmount   float64              `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset    string               `json:"marginBuyBorrowAsset"`
	Fills                   []struct {
		Price           types.Number `json:"price"`
		Qty             types.Number `json:"qty"`
		Commission      types.Number `json:"commission"`
		CommissionAsset string       `json:"commissionAsset"`
	} `json:"fills"`
}

// UMConditionalOrder represents a USDT margined conditional order instance.
type UMConditionalOrder struct {
	NewClientStrategyID     string               `json:"newClientStrategyId"`
	StrategyID              int                  `json:"strategyId"`
	StrategyStatus          string               `json:"strategyStatus"`
	StrategyType            string               `json:"strategyType"`
	OrigQty                 types.Number         `json:"origQty"`
	Price                   types.Number         `json:"price"`
	ReduceOnly              bool                 `json:"reduceOnly"`
	Side                    string               `json:"side"`
	PositionSide            string               `json:"positionSide"`
	StopPrice               types.Number         `json:"stopPrice"`
	Symbol                  string               `json:"symbol"`
	TimeInForce             string               `json:"timeInForce"`
	ActivatePrice           types.Number         `json:"activatePrice"`
	PriceRate               types.Number         `json:"priceRate"`
	BookTime                convert.ExchangeTime `json:"bookTime"`
	UpdateTime              convert.ExchangeTime `json:"updateTime"`
	WorkingType             string               `json:"workingType"`
	PriceProtect            bool                 `json:"priceProtect"`
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	GoodTillDate            int64                `json:"goodTillDate"`
}

// UMConditionalOrderParam represents a conditional order parameter for unified margin
type UMConditionalOrderParam struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	PositionSide            string  `json:"positionSide"` // Default BOTH for One-way Mode ; LONG or SHORT for Hedge Mode. It must be sent in Hedge Mode.
	StrategyType            string  `json:"strategyType"`
	TimeInForce             string  `json:"timeInForce"`
	Quantity                float64 `json:"quantity"`
	ReduceOnly              bool    `json:"reduceOnly"`
	Price                   float64 `json:"price"`
	WorkingType             string  `json:"workingType"`
	PriceProtect            bool    `json:"priceProtect"`
	NewClientStrategyID     string  `json:"newClientStrategyID"`
	StopPrice               float64 `json:"stopPrice"`
	ActivationPrice         float64 `json:"activationPrice"`
	CallbackRate            float64 `json:"callbackRate"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode"`
	GoodTillDate            int64   `json:"goodTillDate"`
}
