package binance

import (
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UM_CM_Order represents a portfolio margin.
type UM_CM_Order struct {
	ClientOrderID string               `json:"clientOrderId"`
	CumQty        types.Number         `json:"cumQty"`
	ExecutedQty   types.Number         `json:"executedQty"`
	OrderID       int64                `json:"orderId"`
	AvgPrice      types.Number         `json:"avgPrice"`
	OrigQty       types.Number         `json:"origQty"`
	Price         types.Number         `json:"price"`
	ReduceOnly    bool                 `json:"reduceOnly"`
	Side          string               `json:"side"`
	PositionSide  string               `json:"positionSide"`
	Status        string               `json:"status"`
	Symbol        string               `json:"symbol"`
	TimeInForce   string               `json:"timeInForce"`
	Type          string               `json:"type"`
	UpdateTime    convert.ExchangeTime `json:"updateTime"`

	// Used By USDT Margined Futures only
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	GoodTillDate            convert.ExchangeTime `json:"goodTillDate"`
	CumQuote                types.Number         `json:"cumQuote"`

	// Used By Coin Margined Futures only
	Pair    string `json:"pair"`
	CumBase string `json:"cumBase"`
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
	Amount                  float64 `json:"quantity,omitempty"`
	QuoteOrderQty           float64 `json:"quoteOrderQty,omitempty"`
	Price                   float64 `json:"price,omitempty"`
	StopPrice               float64 `json:"stopPrice,omitempty"` // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	NewClientOrderID        string  `json:"newClientOrderId,omitempty"`
	NewOrderRespType        string  `json:"newOrderRespType,omitempty"`
	IcebergQuantity         float64 `json:"icebergQty,omitempty"`
	SideEffectType          string  `json:"sideEffectType,omitempty"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode,omitempty"`
}

// MarginOrderResp represents a margin order response.
type MarginOrderResp struct {
	Symbol                  string               `json:"symbol"`
	OrderID                 int64                `json:"orderId"`
	ClientOrderID           string               `json:"clientOrderId"`
	OrigClientOrderID       string               `json:"origClientOrderId"`
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

// MarginAccOrdersList represents a list of margin account order details.
type MarginAccOrdersList []struct {
	Symbol              string               `json:"symbol"`
	OrigClientOrderID   string               `json:"origClientOrderId,omitempty"`
	OrderID             int64                `json:"orderId,omitempty"`
	OrderListID         int64                `json:"orderListId"`
	ClientOrderID       string               `json:"clientOrderId,omitempty"`
	Price               types.Number         `json:"price,omitempty"`
	OrigQty             types.Number         `json:"origQty,omitempty"`
	ExecutedQty         types.Number         `json:"executedQty,omitempty"`
	CummulativeQuoteQty types.Number         `json:"cummulativeQuoteQty,omitempty"`
	Status              string               `json:"status,omitempty"`
	TimeInForce         string               `json:"timeInForce,omitempty"`
	Type                string               `json:"type,omitempty"`
	Side                string               `json:"side,omitempty"`
	ContingencyType     string               `json:"contingencyType,omitempty"`
	ListStatusType      string               `json:"listStatusType,omitempty"`
	ListOrderStatus     string               `json:"listOrderStatus,omitempty"`
	ListClientOrderID   string               `json:"listClientOrderId,omitempty"`
	TransactionTime     convert.ExchangeTime `json:"transactionTime,omitempty"`
	Orders              []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	OrderReports []struct {
		Symbol                  string       `json:"symbol"`
		OrigClientOrderID       string       `json:"origClientOrderId"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		StopPrice               types.Number `json:"stopPrice,omitempty"`
		IcebergQty              types.Number `json:"icebergQty"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"orderReports,omitempty"`
}

// ConditionalOrder represents a USDT/Coin margined conditional order instance.
type ConditionalOrder struct {
	NewClientStrategyID string               `json:"newClientStrategyId"`
	StrategyID          int                  `json:"strategyId"`
	StrategyStatus      string               `json:"strategyStatus"`
	StrategyType        string               `json:"strategyType"`
	OrigQty             types.Number         `json:"origQty"`
	Price               types.Number         `json:"price"`
	ReduceOnly          bool                 `json:"reduceOnly"`
	Side                string               `json:"side"`
	PositionSide        string               `json:"positionSide"`
	StopPrice           types.Number         `json:"stopPrice"`
	Symbol              string               `json:"symbol"`
	TimeInForce         string               `json:"timeInForce"`
	ActivatePrice       types.Number         `json:"activatePrice"` // activation price, only return with TRAILING_STOP_MARKET order
	PriceRate           types.Number         `json:"priceRate"`     // callback rate, only return with TRAILING_STOP_MARKET order
	BookTime            convert.ExchangeTime `json:"bookTime"`      // order place time
	UpdateTime          convert.ExchangeTime `json:"updateTime"`
	WorkingType         string               `json:"workingType"`
	PriceProtect        bool                 `json:"priceProtect"`

	// Returned for USDT Margined Futures orders only
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	GoodTillDate            convert.ExchangeTime `json:"goodTillDate"` //order pre-set auot cancel time for TIF GTD order

	Pair string `json:"pair"`
}

// ConditionalOrderParam represents a conditional order parameter for coin/usdt margined futures.
type ConditionalOrderParam struct {
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	PositionSide        string  `json:"positionSide,omitempty"` // Default BOTH for One-way Mode ; LONG or SHORT for Hedge Mode. It must be sent in Hedge Mode.
	StrategyType        string  `json:"strategyType"`           // "STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET", and "TRAILING_STOP_MARKET"
	TimeInForce         string  `json:"timeInForce,omitempty"`
	Quantity            float64 `json:"quantity,omitempty"`
	ReduceOnly          bool    `json:"reduceOnly,omitempty"`
	Price               float64 `json:"price,omitempty"`
	WorkingType         string  `json:"workingType,omitempty"`
	PriceProtect        bool    `json:"priceProtect,omitempty"`
	NewClientStrategyID string  `json:"newClientStrategyID,omitempty"`
	StopPrice           float64 `json:"stopPrice,omitempty"`
	ActivationPrice     float64 `json:"activationPrice,omitempty"`
	CallbackRate        float64 `json:"callbackRate,omitempty"`

	// User in USDT margined futures only
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`
	GoodTillDate            int64  `json:"goodTillDate,omitempty"`
}

// SuccessResponse represents a success code and message; used when cancelling orders in portfolio margin endpoints.
type SuccessResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// MarginOrder represents a margin account order
type MarginOrder struct {
	ClientOrderID           string               `json:"clientOrderId"`
	CummulativeQuoteQty     types.Number         `json:"cummulativeQuoteQty"`
	ExecutedQty             types.Number         `json:"executedQty"`
	IcebergQty              types.Number         `json:"icebergQty"`
	IsWorking               bool                 `json:"isWorking"`
	OrderID                 int                  `json:"orderId"`
	OrigQty                 types.Number         `json:"origQty"`
	Price                   types.Number         `json:"price"`
	Side                    string               `json:"side"`
	Status                  string               `json:"status"`
	StopPrice               types.Number         `json:"stopPrice"`
	Symbol                  string               `json:"symbol"`
	Time                    convert.ExchangeTime `json:"time"`
	TimeInForce             string               `json:"timeInForce"`
	Type                    string               `json:"type"`
	UpdateTime              convert.ExchangeTime `json:"updateTime"`
	AccountID               int64                `json:"accountId"`
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	PreventedMatchID        any                  `json:"preventedMatchId"`
	PreventedQuantity       any                  `json:"preventedQuantity"`
}

// MarginAccountTradeItem represents a margin account trade item.
type MarginAccountTradeItem struct {
	ID              int64                `json:"id"`
	Symbol          string               `json:"symbol"`
	Commission      string               `json:"commission"`
	CommissionAsset string               `json:"commissionAsset"`
	IsBestMatch     bool                 `json:"isBestMatch"`
	IsBuyer         bool                 `json:"isBuyer"`
	IsMaker         bool                 `json:"isMaker"`
	OrderID         int64                `json:"orderId"`
	Price           types.Number         `json:"price"`
	Qty             types.Number         `json:"qty"`
	Time            convert.ExchangeTime `json:"time"`
}
