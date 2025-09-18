package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketOrderDetails is the order details from the websocket response.
type WebsocketOrderDetails struct {
	Category                   string        `json:"category"`
	OrderID                    string        `json:"orderId"`
	OrderLinkID                string        `json:"orderLinkId"`
	IsLeverage                 string        `json:"isLeverage"` // Whether to borrow. Unified spot only. 0: false, 1: true; Classic spot is not supported, always 0
	BlockTradeID               string        `json:"blockTradeId"`
	Symbol                     string        `json:"symbol"` // Undelimited so inbuilt string used.
	Price                      types.Number  `json:"price"`
	Quantity                   types.Number  `json:"qty"`
	Side                       order.Side    `json:"side"`
	PositionIdx                int64         `json:"positionIdx"`
	OrderStatus                string        `json:"orderStatus"`
	CreateType                 string        `json:"createType"`
	CancelType                 string        `json:"cancelType"`
	RejectReason               string        `json:"rejectReason"` // Classic spot is not supported
	AveragePrice               types.Number  `json:"avgPrice"`
	LeavesQuantity             types.Number  `json:"leavesQty"`   // The remaining qty not executed. Classic spot is not supported
	LeavesValue                types.Number  `json:"leavesValue"` // The remaining value not executed. Classic spot is not supported
	CumulativeExecutedQuantity types.Number  `json:"cumExecQty"`
	CumulativeExecutedValue    types.Number  `json:"cumExecValue"`
	CumulativeExecutedFee      types.Number  `json:"cumExecFee"`
	ClosedPNL                  types.Number  `json:"closedPnl"`
	FeeCurrency                currency.Code `json:"feeCurrency"` // Trading fee currency for Spot only.
	TimeInForce                string        `json:"timeInForce"`
	OrderType                  string        `json:"orderType"`
	StopOrderType              string        `json:"stopOrderType"`
	OneCancelsOtherTriggerBy   string        `json:"ocoTriggerBy"` // UTA Spot: add new response field ocoTriggerBy, and the value can be OcoTriggerByUnknown, OcoTriggerByTp, OcoTriggerBySl
	OrderImpliedVolatility     types.Number  `json:"orderIv"`
	MarketUnit                 string        `json:"marketUnit"`   // The unit for qty when create Spot market orders for UTA account. baseCoin, quoteCoin
	TriggerPrice               types.Number  `json:"triggerPrice"` // Trigger price. If stopOrderType=TrailingStop, it is activate price. Otherwise, it is trigger price
	TakeProfit                 types.Number  `json:"takeProfit"`
	StopLoss                   types.Number  `json:"stopLoss"`
	TakeProfitStopLossMode     string        `json:"tpslMode"` // TP/SL mode, Full: entire position for TP/SL. Partial: partial position tp/sl. Spot does not have this field, and Option returns always ""
	TakeProfitLimitPrice       types.Number  `json:"tpLimitPrice"`
	StopLossLimitPrice         types.Number  `json:"slLimitPrice"`
	TakeProfitTriggerBy        string        `json:"tpTriggerBy"`
	StopLossTriggerBy          string        `json:"slTriggerBy"`
	TriggerDirection           int64         `json:"triggerDirection"` // Trigger direction. 1: rise, 2: fall
	TriggerBy                  string        `json:"triggerBy"`
	LastPriceOnCreated         types.Number  `json:"lastPriceOnCreated"`
	ReduceOnly                 bool          `json:"reduceOnly"`
	CloseOnTrigger             bool          `json:"closeOnTrigger"`
	PlaceType                  string        `json:"placeType"` // 	Place type, option used. iv, price
	SMPType                    string        `json:"smpType"`
	SMPGroup                   int           `json:"smpGroup"`
	SMPOrderID                 string        `json:"smpOrderId"`
	CreatedTime                types.Time    `json:"createdTime"`
	UpdatedTime                types.Time    `json:"updatedTime"`
}

// WebsocketConfirmation is the initial response from the websocket connection
type WebsocketConfirmation struct {
	RequestID              string            `json:"reqId"`
	RetCode                int64             `json:"retCode"`
	RetMsg                 string            `json:"retMsg"`
	Operation              string            `json:"op"`
	RequestAcknowledgement OrderResponse     `json:"data"`
	Header                 map[string]string `json:"header"`
	ConnectionID           string            `json:"connId"`
}

// WebsocketOrderResponse is the response from an order request through the websocket connection
type WebsocketOrderResponse struct {
	ID           string                  `json:"id"`
	Topic        string                  `json:"topic"`
	CreationTime types.Time              `json:"creationTime"`
	Data         []WebsocketOrderDetails `json:"data"`
}

// WebsocketGeneralPayload is the general payload for websocket requests
type WebsocketGeneralPayload struct {
	RequestID string            `json:"reqId"`
	Header    map[string]string `json:"header"`
	Operation string            `json:"op"`
	Arguments []any             `json:"args"`
}
