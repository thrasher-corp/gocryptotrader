package binanceus

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketUserDataStreamSubscriptionResponse is a User Data Stream subscription: the ID its events carry
type WebsocketUserDataStreamSubscriptionResponse struct {
	SubscriptionID uint64 `json:"subscriptionId"`
}

// WebsocketAccountPosition is the outboundAccountPosition event, sent whenever an account balance changes with the
// balances the change may have affected
type WebsocketAccountPosition struct {
	EventType             string                    `json:"e"`
	EventTime             types.Time                `json:"E"`
	LastAccountUpdateTime types.Time                `json:"u"`
	Balances              []WebsocketAccountBalance `json:"B"`
}

// WebsocketAccountBalance is an asset balance of an outboundAccountPosition event
type WebsocketAccountBalance struct {
	Asset  currency.Code `json:"a"`
	Free   types.Number  `json:"f"`
	Locked types.Number  `json:"l"`
}

// WebsocketBalanceUpdate is the balanceUpdate event, sent on deposits and withdrawals
type WebsocketBalanceUpdate struct {
	EventType    string        `json:"e"`
	EventTime    types.Time    `json:"E"`
	Asset        currency.Code `json:"a"`
	BalanceDelta types.Number  `json:"d"`
	ClearTime    types.Time    `json:"T"`
}

// WebsocketExecutionReport is the executionReport event, sent whenever an order changes; the conditional fields appear
// only for orders they apply to
type WebsocketExecutionReport struct {
	EventType                  string       `json:"e"`
	EventTime                  types.Time   `json:"E"`
	Symbol                     string       `json:"s"`
	ClientOrderID              string       `json:"c"`
	Side                       Side         `json:"S"`
	OrderType                  OrderType    `json:"o"`
	TimeInForce                TimeInForce  `json:"f"`
	OrderQuantity              types.Number `json:"q"`
	OrderPrice                 types.Number `json:"p"`
	StopPrice                  types.Number `json:"P"`
	TrailingDeltaInBasisPoints uint64       `json:"d"`
	IcebergQuantity            types.Number `json:"F"`
	OrderListID                int64        `json:"g"` // -1 unless the order is an order list leg
	// OriginalClientOrderID is the client order ID of the order being cancelled
	OriginalClientOrderID    string        `json:"C"`
	CurrentExecutionType     string        `json:"x"`
	CurrentOrderStatus       string        `json:"X"`
	OrderRejectReason        string        `json:"r"`
	OrderID                  uint64        `json:"i"`
	LastExecutedQuantity     types.Number  `json:"l"`
	CumulativeFilledQuantity types.Number  `json:"z"`
	LastExecutedPrice        types.Number  `json:"L"`
	CommissionAmount         types.Number  `json:"n"`
	CommissionAsset          currency.Code `json:"N"`
	TransactionTime          types.Time    `json:"T"`
	TradeID                  int64         `json:"t"` // -1 unless the event is a trade
	ExecutionID              uint64        `json:"I"`
	IsOrderOnBook            bool          `json:"w"`
	IsMaker                  bool          `json:"m"`
	Ignore                   bool          `json:"M"`
	OrderCreationTime        types.Time    `json:"O"`
	// CumulativeQuoteAssetTransactedQuantity divided by CumulativeFilledQuantity is the average price
	CumulativeQuoteAssetTransactedQuantity types.Number `json:"Z"`
	LastQuoteAssetTransactedQuantity       types.Number `json:"Y"`
	// The prevented execution fields carry a TRADE_PREVENTION execution's quantity, price and notional, which l, L and Y
	// report as 0
	PreventedExecutionQuantity      types.Number            `json:"pl"`
	PreventedExecutionPrice         types.Number            `json:"pL"`
	PreventedExecutionQuoteQuantity types.Number            `json:"pY"`
	QuoteOrderQuantity              types.Number            `json:"Q"`
	SelfTradePreventionMode         SelfTradePreventionMode `json:"V"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds
	TrailingTime          int64        `json:"D"`
	WorkingTime           types.Time   `json:"W"`
	TradeGroupID          uint64       `json:"u"`
	PreventedMatchID      uint64       `json:"v"`
	CounterOrderID        uint64       `json:"U"`
	PreventedQuantity     types.Number `json:"A"`
	LastPreventedQuantity types.Number `json:"B"`
	ExpiryReason          string       `json:"eR"`
}

// WebsocketListStatus is the listStatus event, sent with the executionReport events of an order list's legs
type WebsocketListStatus struct {
	EventType         string                     `json:"e"`
	EventTime         types.Time                 `json:"E"`
	Symbol            string                     `json:"s"`
	OrderListID       uint64                     `json:"g"`
	ContingencyType   string                     `json:"c"`
	ListStatusType    string                     `json:"l"`
	ListOrderStatus   string                     `json:"L"`
	ListRejectReason  string                     `json:"r"`
	ListClientOrderID string                     `json:"C"`
	TransactionTime   types.Time                 `json:"T"`
	Orders            []WebsocketListStatusOrder `json:"O"`
}

// WebsocketListStatusOrder is an order list leg of a listStatus event
type WebsocketListStatusOrder struct {
	Symbol        string `json:"s"`
	OrderID       uint64 `json:"i"`
	ClientOrderID string `json:"c"`
}

// WebsocketExternalLockUpdate is the externalLockUpdate event, sent when an external system locks or unlocks part of a
// spot balance
type WebsocketExternalLockUpdate struct {
	EventType       string        `json:"e"`
	EventTime       types.Time    `json:"E"`
	Asset           currency.Code `json:"a"`
	Delta           types.Number  `json:"d"`
	TransactionTime types.Time    `json:"T"`
}

// WebsocketEventStreamTerminated is the eventStreamTerminated event, sent when a User Data Stream subscription ends
type WebsocketEventStreamTerminated struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}
