package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CommandResponse is the reply the spread and event public connections send to a subscribe or unsubscribe request.
// It carries no request ID, so a request is matched to its reply by the topics it names
type CommandResponse struct {
	Success      bool                  `json:"success"`
	ConnectionID string                `json:"conn_id"`
	Data         CommandResponseTopics `json:"data"`
	Type         string                `json:"type"`
}

// CommandResponseTopics lists the topics a subscribe or unsubscribe request succeeded and failed for
type CommandResponseTopics struct {
	FailTopics    []string `json:"failTopics"`
	SuccessTopics []string `json:"successTopics"`
}

// WebsocketSpreadOrderbookUpdate is a snapshot or delta of a spread combination's orderbook
type WebsocketSpreadOrderbookUpdate struct {
	Topic     string              `json:"topic"`
	Type      string              `json:"type"`
	Timestamp types.Time          `json:"ts"`
	Data      SpreadOrderbookData `json:"data"`
	// CreationTimestamp is when the matching engine produced the depth, comparable with public trade times
	CreationTimestamp types.Time `json:"cts"`
}

// SpreadOrderbookData holds the levels of a spread orderbook frame; a delta level with a zero size is removed
type SpreadOrderbookData struct {
	Symbol string                           `json:"s"`
	Bids   orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks   orderbook.LevelsArrayPriceAmount `json:"a"`
	// UpdateID is 1 for a snapshot Bybit sends after a service restart, which replaces the local book
	UpdateID uint64 `json:"u"`
	Sequence uint64 `json:"seq"`
}

// WebsocketSpreadPublicTradeUpdate holds public trades of a spread combination, oldest first
type WebsocketSpreadPublicTradeUpdate struct {
	Topic     string                 `json:"topic"`
	Type      string                 `json:"type"`
	Timestamp types.Time             `json:"ts"`
	Data      []SpreadWebsocketTrade `json:"data"`
}

// SpreadWebsocketTrade is a public trade of a spread combination
type SpreadWebsocketTrade struct {
	OrderFillTimestamp types.Time `json:"T"`
	Symbol             string     `json:"s"`
	// Side is the taker's side
	Side          string       `json:"S"`
	Size          types.Number `json:"v"`
	Price         types.Number `json:"p"`
	TickDirection string       `json:"L"`
	TradeID       string       `json:"i"`
	Sequence      uint64       `json:"seq"`
}

// WebsocketSpreadTickerUpdate holds a spread combination's ticker
type WebsocketSpreadTickerUpdate struct {
	Topic     string       `json:"topic"`
	Type      string       `json:"type"`
	Timestamp types.Time   `json:"ts"`
	Data      SpreadTicker `json:"data"`
}

// WebsocketSpreadOrderUpdate holds changes to the caller's spread orders and their legs
type WebsocketSpreadOrderUpdate struct {
	ID           string              `json:"id"`
	Topic        string              `json:"topic"`
	CreationTime types.Time          `json:"creationTime"`
	Data         []SpreadOrderUpdate `json:"data"`
}

// SpreadOrderUpdate is the state of a spread combination order or of one of its leg orders
type SpreadOrderUpdate struct {
	// Category is combination, spot_leg or future_leg
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	// ParentOrderID is the combination order of a leg order
	ParentOrderID string       `json:"parentOrderId"`
	OrderID       string       `json:"orderId"`
	OrderLinkID   string       `json:"orderLinkId"`
	Side          string       `json:"side"`
	OrderStatus   string       `json:"orderStatus"`
	CancelType    string       `json:"cancelType"`
	RejectReason  string       `json:"rejectReason"`
	TimeInForce   string       `json:"timeInForce"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	AveragePrice  types.Number `json:"avgPrice"`
	// LeavesQuantity is the quantity not yet executed and LeavesValue its estimated value
	LeavesQuantity             types.Number `json:"leavesQty"`
	LeavesValue                types.Number `json:"leavesValue"`
	CumulativeExecutedQuantity types.Number `json:"cumExecQty"`
	CumulativeExecutedValue    types.Number `json:"cumExecValue"`
	// CumulativeExecutedFee is deprecated in favour of CumulativeFeeDetail
	CumulativeExecutedFee types.Number `json:"cumExecFee"`
	OrderType             string       `json:"orderType"`
	// IsLeverage is 1 for a spot leg when spot margin is on and 0 for a futures leg; it is empty for the combination
	IsLeverage  string     `json:"isLeverage"`
	CreatedTime types.Time `json:"createdTime"`
	UpdatedTime types.Time `json:"updatedTime"`
	// FeeCurrency is deprecated in favour of CumulativeFeeDetail
	FeeCurrency currency.Code `json:"feeCurrency"`
	CreateType  string        `json:"createType"`
	ClosedPNL   types.Number  `json:"closedPnl"`
	// CumulativeFeeDetail holds the cumulative trading fee by fee coin
	CumulativeFeeDetail map[currency.Code]types.Number `json:"cumFeeDetail"`
}

// WebsocketSpreadExecutionUpdate holds executions of the caller's spread orders and their legs
type WebsocketSpreadExecutionUpdate struct {
	ID           string                  `json:"id"`
	Topic        string                  `json:"topic"`
	CreationTime types.Time              `json:"creationTime"`
	Data         []SpreadExecutionUpdate `json:"data"`
}

// SpreadExecutionUpdate is an execution of a spread combination or of one of its legs
type SpreadExecutionUpdate struct {
	// Category is combination, spot_leg or future_leg
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	// IsLeverage is 1 for a spot leg when spot margin is on and 0 for a futures leg; it is empty for the combination
	IsLeverage string `json:"isLeverage"`
	// OrderID and OrderLinkID are empty for a leg
	OrderID        string       `json:"orderId"`
	OrderLinkID    string       `json:"orderLinkId"`
	Side           string       `json:"side"`
	OrderPrice     types.Number `json:"orderPrice"`
	OrderQuantity  types.Number `json:"orderQty"`
	LeavesQuantity types.Number `json:"leavesQty"`
	CreateType     string       `json:"createType"`
	OrderType      string       `json:"orderType"`
	// ExecutionFee is deprecated for the spot leg, whose fee is in ExecutionFeeV2
	ExecutionFee   types.Number  `json:"execFee"`
	ExecutionFeeV2 types.Number  `json:"execFeeV2"`
	FeeCoin        currency.Code `json:"feeCoin"`
	// ParentExecutionID is the combination execution of a leg execution
	ParentExecutionID string       `json:"parentExecId"`
	ExecutionID       string       `json:"execId"`
	ExecutionPrice    types.Number `json:"execPrice"`
	ExecutionQuantity types.Number `json:"execQty"`
	ExecutionPNL      types.Number `json:"execPnl"`
	ExecutionType     string       `json:"execType"`
	ExecutionValue    types.Number `json:"execValue"`
	ExecutionTime     types.Time   `json:"execTime"`
	IsMaker           bool         `json:"isMaker"`
	FeeRate           types.Number `json:"feeRate"`
	MarkPrice         types.Number `json:"markPrice"`
	ClosedSize        types.Number `json:"closedSize"`
	Sequence          uint64       `json:"seq"`
}
