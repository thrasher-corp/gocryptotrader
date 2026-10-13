package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EventOrderbookFrame is a snapshot or delta of an event contract's payout ratio orderbook
type EventOrderbookFrame struct {
	Topic     string             `json:"topic"`
	Type      string             `json:"type"`
	Timestamp types.Time         `json:"ts"`
	Data      EventOrderbookData `json:"data"`
	// CreationTimestamp is when the matching engine produced the data
	CreationTimestamp types.Time `json:"cts"`
}

// EventOrderbookData holds the levels of an event orderbook frame; a delta entry with a zero order value is removed
type EventOrderbookData struct {
	Symbol  string                `json:"s"`
	Entries []EventOrderbookEntry `json:"r"`
	// UpdateID is 1 for a snapshot Bybit sends after a service restart, which replaces the local book
	UpdateID uint64 `json:"u"`
	Sequence uint64 `json:"seq"`
}

// EventPublicTradeFrame holds public trades of an event contract, oldest first
type EventPublicTradeFrame struct {
	Topic     string                `json:"topic"`
	Type      string                `json:"type"`
	Timestamp types.Time            `json:"ts"`
	Data      []EventWebsocketTrade `json:"data"`
}

// EventWebsocketTrade is a public trade of an event contract
type EventWebsocketTrade struct {
	// Timestamp is when the order filled
	Timestamp types.Time `json:"T"`
	Symbol    string     `json:"s"`
	// PayoutRatio is the trade price
	PayoutRatio types.Number `json:"r"`
	// Amount is the trade's order value
	Amount   types.Number `json:"a"`
	TradeID  string       `json:"i"`
	Sequence uint64       `json:"seq"`
}

// EventPositionFrame holds changes to the caller's event contract positions
type EventPositionFrame struct {
	ID           string          `json:"id"`
	Topic        string          `json:"topic"`
	CreationTime types.Time      `json:"creationTime"`
	Data         []EventPosition `json:"data"`
}

// EventExecutionFrame holds executions of the caller's event contract orders
type EventExecutionFrame struct {
	ID           string                 `json:"id"`
	Topic        string                 `json:"topic"`
	CreationTime types.Time             `json:"creationTime"`
	Data         []EventExecutionUpdate `json:"data"`
}

// EventExecutionUpdate is an execution of an event contract order. Fields prefixed EC describe the event contract
type EventExecutionUpdate struct {
	// Category is event
	Category      string        `json:"category"`
	TransactionID string        `json:"transId"`
	ExecutionID   string        `json:"execId"`
	OrderID       string        `json:"orderId"`
	OrderLinkID   string        `json:"orderLinkId"`
	Symbol        string        `json:"symbol"`
	SymbolID      uint64        `json:"symbolId"`
	BaseCoin      currency.Code `json:"baseCoin"`
	QuoteCoin     currency.Code `json:"quoteCoin"`
	SettleCoin    currency.Code `json:"settleCoin"`
	Side          string        `json:"side"`
	FeeCoin       currency.Code `json:"feeCoin"`
	ExecutionType string        `json:"execType"`
	// ExecutionPrice is the payout ratio
	ExecutionPrice   types.Number `json:"execPrice"`
	ExecutionValue   types.Number `json:"execValue"`
	ExecutionFee     types.Number `json:"execFee"`
	ExecutionFeeRate types.Number `json:"execFeeRate"`
	OrderPrice       types.Number `json:"orderPrice"`
	// ECContractType and ECDirection are numeric codes here, unlike the names the order stream sends
	ECContractType uint64 `json:"ecContractType"`
	ECDirection    uint64 `json:"ecDirection"`
	// ECDurationWindow is in seconds; a number here but quoted, and empty for contracts without one, over REST
	ECDurationWindow types.Number `json:"ecDurationWindow"`
	// ECTargetPrice is set for Target contracts only
	ECTargetPrice types.Number `json:"ecTargetPrice"`
	// ECLowerBound and ECUpperBound are set for Range contracts only
	ECLowerBound       types.Number    `json:"ecLowerBound"`
	ECUpperBound       types.Number    `json:"ecUpperBound"`
	ECOrderValue       types.Number    `json:"ecOrderValue"`
	ECIndexPrice       types.Number    `json:"ecIndexPrice"`
	ECPayout           types.Number    `json:"ecPayout"`
	EntryPrice         types.Number    `json:"entryPrice"`
	SettlePrice        types.Number    `json:"settlePrice"`
	GrossPayoutRatio   types.Number    `json:"grossPayoutRatio"`
	SessionRealisedPNL types.Number    `json:"sessionRpl"`
	TransactionTime    types.Time      `json:"transTime"`
	SettleTime         types.Time      `json:"settleTimeMs"`
	CrossSequence      types.Number    `json:"crossSeq"`
	TimeInForce        string          `json:"timeInForce"`
	ExtraFees          []EventExtraFee `json:"extraFees"`
}

// EventExtraFee is an extra fee charged on an event contract order or execution, such as a tax; the fields are those
// the private execution stream documents for its extra fees
type EventExtraFee struct {
	FeeCoin    currency.Code `json:"feeCoin"`
	FeeType    string        `json:"feeType"`
	SubFeeType string        `json:"subFeeType"`
	FeeRate    types.Number  `json:"feeRate"`
	Fee        types.Number  `json:"fee"`
}

// EventOrderFrame holds changes to the caller's event contract orders
type EventOrderFrame struct {
	ID           string             `json:"id"`
	Topic        string             `json:"topic"`
	CreationTime types.Time         `json:"creationTime"`
	Data         []EventOrderUpdate `json:"data"`
}

// EventOrderUpdate is the state of an event contract order. Fields prefixed EC describe the event contract
type EventOrderUpdate struct {
	// Category is event
	Category       string        `json:"category"`
	OrderID        string        `json:"orderId"`
	OrderLinkID    string        `json:"orderLinkId"`
	Symbol         string        `json:"symbol"`
	SymbolID       uint64        `json:"symbolId"`
	Side           string        `json:"side"`
	BaseCoin       currency.Code `json:"baseCoin"`
	QuoteCoin      currency.Code `json:"quoteCoin"`
	SettleCoin     currency.Code `json:"settleCoin"`
	OrderStatus    string        `json:"orderStatus"`
	ECContractType string        `json:"ecContractType"`
	ECDirection    string        `json:"ecDirection"`
	// ECDurationWindow is in seconds; a number here but quoted, and empty for contracts without one, over REST
	ECDurationWindow types.Number `json:"ecDurationWindow"`
	// ECTargetPrice is set for Target contracts only
	ECTargetPrice types.Number `json:"ecTargetPrice"`
	// ECLowerBound and ECUpperBound are set for Range contracts only
	ECLowerBound types.Number `json:"ecLowerBound"`
	ECUpperBound types.Number `json:"ecUpperBound"`
	ECOrderValue types.Number `json:"ecOrderValue"`
	ECIndexPrice types.Number `json:"ecIndexPrice"`
	ECSettleTime types.Time   `json:"ecSettleTime"`
	// PayoutRatio is the order's price
	PayoutRatio             types.Number `json:"payoutRatio"`
	OrderAveragePayoutRatio types.Number `json:"orderAvgPayoutRatio"`
	CumulativeExecutedValue types.Number `json:"cumExecValue"`
	CumulativeExecutedFee   types.Number `json:"cumExecFee"`
	LeavesValue             types.Number `json:"leavesValue"`
	CumulativePayout        types.Number `json:"cumPayout"`
	// CreateType is a numeric code here, unlike the name REST sends
	CreateType      uint64          `json:"createType"`
	CancelType      string          `json:"cancelType"`
	RejectReason    string          `json:"rejectReason"`
	ExtraFees       []EventExtraFee `json:"extraFees"`
	TransactionTime types.Time      `json:"transTime"`
	CreatedTime     types.Time      `json:"createdTime"`
	UpdatedTime     types.Time      `json:"updatedTime"`
}
