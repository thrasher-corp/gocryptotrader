package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PreUpgradeOrderHistoryRequest holds the parameters for GetPreUpgradeOrderHistory
type PreUpgradeOrderHistoryRequest struct {
	// Category is required
	Category string
	// Symbol defaults to the USDT contracts; USDC perpetual orders need it set
	Symbol string
	// BaseCoin applies to option
	BaseCoin    currency.Code
	OrderID     string
	OrderLinkID string
	// OrderFilter is Order for active orders or StopOrder for conditional ones
	OrderFilter string
	// OrderStatus does not apply to spot
	OrderStatus string
	// StartTime and EndTime go together, at most 7 days apart; without them the last 7 days are returned
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// PreUpgradeOrderHistoryResponse holds a page of pre-upgrade orders
type PreUpgradeOrderHistoryResponse struct {
	Category       string            `json:"category"`
	List           []PreUpgradeOrder `json:"list"`
	NextPageCursor string            `json:"nextPageCursor"`
}

// PreUpgradeOrder holds an order placed before the account was upgraded to a unified account
type PreUpgradeOrder struct {
	OrderID      string       `json:"orderId"`
	OrderLinkID  string       `json:"orderLinkId"`
	BlockTradeID string       `json:"blockTradeId"`
	Symbol       string       `json:"symbol"`
	Price        types.Number `json:"price"`
	Quantity     types.Number `json:"qty"`
	Side         string       `json:"side"`
	// IsLeverage means nothing for orders placed before the upgrade
	IsLeverage    string `json:"isLeverage"`
	PositionIndex uint64 `json:"positionIdx"`
	OrderStatus   string `json:"orderStatus"`
	CancelType    string `json:"cancelType"`
	RejectReason  string `json:"rejectReason"`
	// AveragePrice is empty for unfilled orders and for partially filled orders that were cancelled
	AveragePrice               types.Number `json:"avgPrice"`
	LeavesQuantity             types.Number `json:"leavesQty"`
	LeavesValue                types.Number `json:"leavesValue"`
	CumulativeExecutedQuantity types.Number `json:"cumExecQty"`
	CumulativeExecutedValue    types.Number `json:"cumExecValue"`
	CumulativeExecutedFee      types.Number `json:"cumExecFee"`
	TimeInForce                string       `json:"timeInForce"`
	OrderType                  string       `json:"orderType"`
	StopOrderType              string       `json:"stopOrderType"`
	OrderIV                    types.Number `json:"orderIv"`
	// TriggerPrice is the activation price of trailing stop orders
	TriggerPrice types.Number `json:"triggerPrice"`
	TakeProfit   types.Number `json:"takeProfit"`
	StopLoss     types.Number `json:"stopLoss"`
	TPTriggerBy  string       `json:"tpTriggerBy"`
	SLTriggerBy  string       `json:"slTriggerBy"`
	// TriggerDirection is 1 for a rise and 2 for a fall
	TriggerDirection   uint64       `json:"triggerDirection"`
	TriggerBy          string       `json:"triggerBy"`
	LastPriceOnCreated types.Number `json:"lastPriceOnCreated"`
	ReduceOnly         bool         `json:"reduceOnly"`
	CloseOnTrigger     bool         `json:"closeOnTrigger"`
	// PlaceType applies to option: iv or price
	PlaceType   string     `json:"placeType"`
	SMPType     string     `json:"smpType"`
	SMPGroup    uint64     `json:"smpGroup"`
	SMPOrderID  string     `json:"smpOrderId"`
	CreatedTime types.Time `json:"createdTime"`
	UpdatedTime types.Time `json:"updatedTime"`
}

// PreUpgradeTradeHistoryRequest holds the parameters for GetPreUpgradeTradeHistory
type PreUpgradeTradeHistoryRequest struct {
	// Category is required
	Category    string
	Symbol      string
	OrderID     string
	OrderLinkID string
	// BaseCoin applies to option
	BaseCoin currency.Code
	// StartTime and EndTime are at most 7 days apart; one alone covers the 7 days from or to it
	StartTime     time.Time
	EndTime       time.Time
	ExecutionType string
	Limit         uint64
	Cursor        string
}

// PreUpgradeTradeHistoryResponse holds a page of pre-upgrade executions
type PreUpgradeTradeHistoryResponse struct {
	Category       string                `json:"category"`
	List           []PreUpgradeExecution `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// PreUpgradeExecution holds an execution made before the account was upgraded to a unified account
type PreUpgradeExecution struct {
	Symbol         string       `json:"symbol"`
	OrderID        string       `json:"orderId"`
	OrderLinkID    string       `json:"orderLinkId"`
	Side           string       `json:"side"`
	OrderPrice     types.Number `json:"orderPrice"`
	OrderQuantity  types.Number `json:"orderQty"`
	LeavesQuantity types.Number `json:"leavesQty"`
	OrderType      string       `json:"orderType"`
	// StopOrderType is empty unless the order was a stop order
	StopOrderType     string       `json:"stopOrderType"`
	ExecutionFee      types.Number `json:"execFee"`
	ExecutionID       string       `json:"execId"`
	ExecutionPrice    types.Number `json:"execPrice"`
	ExecutionQuantity types.Number `json:"execQty"`
	ExecutionType     string       `json:"execType"`
	ExecutionValue    types.Number `json:"execValue"`
	ExecutionTime     types.Time   `json:"execTime"`
	IsMaker           bool         `json:"isMaker"`
	FeeRate           types.Number `json:"feeRate"`
	TradeIV           types.Number `json:"tradeIv"`
	MarkIV            types.Number `json:"markIv"`
	MarkPrice         types.Number `json:"markPrice"`
	IndexPrice        types.Number `json:"indexPrice"`
	UnderlyingPrice   types.Number `json:"underlyingPrice"`
	BlockTradeID      string       `json:"blockTradeId"`
	ClosedSize        types.Number `json:"closedSize"`
}

// PreUpgradeClosedPNLRequest holds the parameters for GetPreUpgradeClosedPNL
type PreUpgradeClosedPNLRequest struct {
	// Category and Symbol are required
	Category string
	Symbol   string
	// StartTime and EndTime are at most 7 days apart; one alone covers the 7 days from or to it
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// PreUpgradeClosedPNLResponse holds a page of pre-upgrade closed profit and loss records
type PreUpgradeClosedPNLResponse struct {
	Category       string                `json:"category"`
	List           []PreUpgradeClosedPNL `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// PreUpgradeClosedPNL holds a position's profit and loss closed before the account was upgraded to a unified account
type PreUpgradeClosedPNL struct {
	Symbol               string       `json:"symbol"`
	OrderID              string       `json:"orderId"`
	Side                 string       `json:"side"`
	Quantity             types.Number `json:"qty"`
	OrderPrice           types.Number `json:"orderPrice"`
	OrderType            string       `json:"orderType"`
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
	CreatedTime types.Time   `json:"createdTime"`
	UpdatedTime types.Time   `json:"updatedTime"`
}

// PreUpgradeTransactionLogRequest holds the parameters for GetPreUpgradeTransactionLog
type PreUpgradeTransactionLogRequest struct {
	// Category is required
	Category string
	BaseCoin currency.Code
	Type     string
	// StartTime and EndTime are at most 7 days apart; one alone covers the 7 days from or to it
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// PreUpgradeTransactionLogResponse holds a page of pre-upgrade transaction logs
type PreUpgradeTransactionLogResponse struct {
	List           []PreUpgradeTransactionLog `json:"list"`
	NextPageCursor string                     `json:"nextPageCursor"`
}

// PreUpgradeTransactionLog holds a USDC derivatives wallet transaction made before the account was upgraded to a
// unified account
type PreUpgradeTransactionLog struct {
	Symbol          string        `json:"symbol"`
	Category        string        `json:"category"`
	Side            string        `json:"side"`
	TransactionTime types.Time    `json:"transactionTime"`
	Type            string        `json:"type"`
	Quantity        types.Number  `json:"qty"`
	Size            types.Number  `json:"size"`
	Currency        currency.Code `json:"currency"`
	TradePrice      types.Number  `json:"tradePrice"`
	// Funding is positive when funding is received and negative when it is paid
	Funding types.Number `json:"funding"`
	// Fee is positive for an expense and negative for a rebate
	Fee         types.Number `json:"fee"`
	CashFlow    types.Number `json:"cashFlow"`
	Change      types.Number `json:"change"`
	CashBalance types.Number `json:"cashBalance"`
	// FeeRate is the trading fee rate of TRADE logs and the funding rate of SETTLEMENT logs
	FeeRate     types.Number `json:"feeRate"`
	BonusChange types.Number `json:"bonusChange"`
	TradeID     string       `json:"tradeId"`
	OrderID     string       `json:"orderId"`
	OrderLinkID string       `json:"orderLinkId"`
}

// PreUpgradeDeliveryRecordRequest holds the parameters for GetPreUpgradeOptionDeliveryRecord
type PreUpgradeDeliveryRecordRequest struct {
	// Category is required
	Category   string
	Symbol     string
	ExpiryDate time.Time
	Limit      uint64
	Cursor     string
}

// PreUpgradeDeliveryRecordResponse holds a page of pre-upgrade option deliveries
type PreUpgradeDeliveryRecordResponse struct {
	Category       string                     `json:"category"`
	List           []PreUpgradeDeliveryRecord `json:"list"`
	NextPageCursor string                     `json:"nextPageCursor"`
}

// PreUpgradeDeliveryRecord holds an option delivery made before the account was upgraded to a unified account
type PreUpgradeDeliveryRecord struct {
	DeliveryTime types.Time `json:"deliveryTime"`
	Symbol       string     `json:"symbol"`
	Side         string     `json:"side"`
	// Position is the delivered size
	Position            types.Number `json:"position"`
	DeliveryPrice       types.Number `json:"deliveryPrice"`
	Strike              types.Number `json:"strike"`
	Fee                 types.Number `json:"fee"`
	DeliveryRealisedPNL types.Number `json:"deliveryRpl"`
}

// PreUpgradeSessionSettlementRequest holds the parameters for GetPreUpgradeUSDCSessionSettlement
type PreUpgradeSessionSettlementRequest struct {
	// Category is required
	Category string
	Symbol   string
	Limit    uint64
	Cursor   string
}

// PreUpgradeSessionSettlementResponse holds a page of pre-upgrade USDC perpetual session settlements
type PreUpgradeSessionSettlementResponse struct {
	Category       string                 `json:"category"`
	List           []PreUpgradeSettlement `json:"list"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

// PreUpgradeSettlement holds a USDC perpetual session settlement made before the account was upgraded to a unified
// account
type PreUpgradeSettlement struct {
	Symbol string `json:"symbol"`
	Side   string `json:"side"`
	// Size is negative for a short position
	Size types.Number `json:"size"`
	// SessionAveragePrice is the settlement price
	SessionAveragePrice types.Number `json:"sessionAvgPrice"`
	MarkPrice           types.Number `json:"markPrice"`
	RealisedPNL         types.Number `json:"realisedPnl"`
	CreatedTime         types.Time   `json:"createdTime"`
}
