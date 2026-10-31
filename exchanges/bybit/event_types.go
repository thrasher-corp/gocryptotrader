package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EventInstrumentsInfoRequest holds the filters for GetEventInstrumentsInfo
type EventInstrumentsInfoRequest struct {
	Symbol string
	Status string
	Limit  uint64
	Cursor string
}

// EventInstrumentsInfoResponse is a page of event contract specifications
type EventInstrumentsInfoResponse struct {
	NextPageCursor string            `json:"nextPageCursor"`
	List           []EventInstrument `json:"list"`
}

// EventInstrument is the specification of an event contract
type EventInstrument struct {
	Symbol            string        `json:"symbol"`
	SymbolID          uint64        `json:"symbolId"`
	BaseCoin          currency.Code `json:"baseCoin"`
	QuoteCoin         currency.Code `json:"quoteCoin"`
	SettleCoin        currency.Code `json:"settleCoin"`
	EventContractType string        `json:"eventContractType"`
	// DurationWindow is in seconds for Up_Down contracts; the docs type it as a string that is empty for the other
	// types, while Bybit sends a number
	DurationWindow types.Number `json:"durationWindow"`
	RVThreshold    types.Number `json:"rvThreshold"`
	Rebate         types.Number `json:"rebate"`
	TakerFeeRate   types.Number `json:"takerFeeRate"`
	MakerFeeRate   types.Number `json:"makerFeeRate"`
	LaunchTime     types.Time   `json:"launchTime"`
	DeliveryTime   types.Time   `json:"deliveryTime"`
	Status         string       `json:"status"`
	// TargetPrice is set for Target contracts only
	TargetPrice types.Number `json:"targetPrice"`
	// LowerBound and UpperBound exclusively bound the range of Range contracts
	LowerBound types.Number `json:"lowerBound"`
	UpperBound types.Number `json:"upperBound"`
	// CallOptionSymbolID and PutOptionSymbolID map the contract to the option symbols at its strikes; zero when the
	// contract has none
	CallOptionSymbolID uint64                `json:"callOptionSymbolId"`
	PutOptionSymbolID  uint64                `json:"putOptionSymbolId"`
	FeeRateRule        []EventFeeRateRule    `json:"feeRateRule"`
	LotSizeFilter      EventLotSizeFilter    `json:"lotSizeFilter"`
	QuotationLimit     []EventQuotationLimit `json:"quotationLimit"`
}

// EventFeeRateRule is the base markup that applies while settlement is a range of seconds away
type EventFeeRateRule struct {
	MinSecondsToDelivery uint64       `json:"minSecToDelivery"`
	MaxSecondsToDelivery uint64       `json:"maxSecToDelivery"`
	BaseMarkup           types.Number `json:"baseMarkup"`
}

// EventLotSizeFilter holds an event contract's order amount limits and quoting intervals
type EventLotSizeFilter struct {
	MinOrderAmount      types.Number `json:"minOrderAmount"`
	MaxOrderAmount      types.Number `json:"maxOrderAmount"`
	OrderAmountTickSize types.Number `json:"orderAmountTickSize"`
	// QuoteMaxValidMilliseconds is the longest a single quote stays valid
	QuoteMaxValidMilliseconds uint64 `json:"quoteMaxValidMs"`
	// QuoteMinIntervalPerUserMilliseconds is the shortest interval between a user's quote requests
	QuoteMinIntervalPerUserMilliseconds uint64 `json:"quoteMinIntervalPerUserMs"`
}

// EventQuotationLimit holds the quote amount limits that apply while settlement is a range of seconds away
type EventQuotationLimit struct {
	MinSecondsToDelivery  uint64       `json:"minSecToDelivery"`
	MaxSecondsToDelivery  uint64       `json:"maxSecToDelivery"`
	QuoteMaxAmountPerUser types.Number `json:"quoteMaxAmountPerUser"`
	QuoteMinAmountPerUser types.Number `json:"quoteMinAmountPerUser"`
	QuoteMinAmount        types.Number `json:"quoteMinAmount"`
}

// EventOrderbookResponse is an event contract's payout ratio orderbook
type EventOrderbookResponse struct {
	Symbol string `json:"s"`
	// Entries holds up to 25 levels
	Entries   []EventOrderbookEntry `json:"r"`
	Timestamp types.Time            `json:"ts"`
	// UpdateID and Sequence are signed as OrderbookResponse's are, since Bybit can send -1 for a book that has no orders,
	// such as a closed contract's
	UpdateID int64 `json:"u"`
	Sequence int64 `json:"seq"`
	// CreationTimestamp is when the data was created
	CreationTimestamp types.Time `json:"cts"`
}

// EventOrderbookEntry is a level of an event contract orderbook: the payout ratio is its price and the order value its
// size
type EventOrderbookEntry struct {
	PayoutRatio types.Number
	OrderValue  types.Number
}

// UnmarshalJSON decodes an entry from its [payoutRatio, orderValue] array
func (en *EventOrderbookEntry) UnmarshalJSON(data []byte) error {
	var entry [2]types.Number
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	en.PayoutRatio, en.OrderValue = entry[0], entry[1]
	return nil
}

// EventOrderHistoryRequest holds the filters for GetEventOrderHistory; Bybit queries the last 7 days by default
type EventOrderHistoryRequest struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// EventOrderHistoryResponse is a page of event contract orders in a final state
type EventOrderHistoryResponse struct {
	List   []EventOrder `json:"list"`
	Cursor string       `json:"cursor"`
}

// EventOrder is an event contract order in a final state
type EventOrder struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	OrderStatus string `json:"orderStatus"`
	Symbol      string `json:"symbol"`
	// SymbolID is quoted here but a number in Get Instrument Info and the order stream
	SymbolID               types.Number  `json:"symbolId"`
	BaseCoin               currency.Code `json:"baseCoin"`
	QuoteCoin              currency.Code `json:"quoteCoin"`
	SettleCoin             currency.Code `json:"settleCoin"`
	EventContractDirection string        `json:"ecDirection"`
	// EventContractDurationWindow is in seconds
	EventContractDurationWindow types.Number `json:"ecDurationWindow"`
	// EventContractTargetPrice is set for Target contracts only
	EventContractTargetPrice types.Number `json:"ecTargetPrice"`
	EventContractType        string       `json:"ecContractType"`
	// EventContractLowerBound and EventContractUpperBound are set for Range contracts only
	EventContractLowerBound types.Number `json:"ecLowerBound"`
	EventContractUpperBound types.Number `json:"ecUpperBound"`
	SettleTime              types.Time   `json:"settleTimeMs"`
	EventContractOrderValue types.Number `json:"ecOrderValue"`
	EventContractIndexPrice types.Number `json:"ecIndexPrice"`
	OrderAveragePayoutRatio types.Number `json:"orderAvgPayoutRatio"`
	CumulativeExecutedValue types.Number `json:"cumExecValue"`
	// PayoutRatio is the order's price
	PayoutRatio           types.Number `json:"payoutRatio"`
	TradeFee              types.Number `json:"tradeFee"`
	TransactionTime       types.Time   `json:"transTime"`
	CreateTime            types.Time   `json:"createTime"`
	UpdateTime            types.Time   `json:"updateTime"`
	CumulativePayout      types.Number `json:"cumPayout"`
	CumulativeExecutedFee types.Number `json:"cumExecFee"`
	CancelType            string       `json:"cancelType"`
	CreateType            string       `json:"createType"`
	RejectReason          string       `json:"rejectReason"`
	// ExtraFees is a JSON encoded fee list when there are extra fees
	ExtraFees string `json:"extraFees"`
}

// EventActiveOrdersRequest holds the filters for GetEventActiveOrders; Bybit uses OrderID when both IDs are given
type EventActiveOrdersRequest struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	Limit       uint64
	Cursor      string
}

// EventActiveOrdersResponse is a page of unfilled and partially filled event contract orders
type EventActiveOrdersResponse struct {
	List           []EventActiveOrder `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// EventActiveOrder is an unfilled or partially filled event contract order
type EventActiveOrder struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	Symbol      string `json:"symbol"`
	// SymbolID is quoted here but a number in Get Instrument Info and the order stream
	SymbolID    types.Number `json:"symbolId"`
	Side        string       `json:"side"`
	OrderStatus string       `json:"orderStatus"`
	// PayoutRatio is the order's price
	PayoutRatio             types.Number `json:"payoutRatio"`
	CumulativeExecutedValue types.Number `json:"cumExecValue"`
	CumulativeExecutedFee   types.Number `json:"cumExecFee"`
	LeavesValue             types.Number `json:"leavesValue"`
	EventContractType       string       `json:"ecContractType"`
	EventContractDirection  string       `json:"ecDirection"`
	EventContractOrderValue types.Number `json:"ecOrderValue"`
	EventContractSettleTime types.Time   `json:"ecSettleTime"`
	// EventContractTargetPrice is set for Target contracts only
	EventContractTargetPrice types.Number `json:"ecTargetPrice"`
	// EventContractLowerBound and EventContractUpperBound are set for Range contracts only
	EventContractLowerBound types.Number `json:"ecLowerBound"`
	EventContractUpperBound types.Number `json:"ecUpperBound"`
	// EventContractDurationWindow is in seconds
	EventContractDurationWindow types.Number `json:"ecDurationWindow"`
	EventContractIndexPrice     types.Number `json:"ecIndexPrice"`
	OrderAveragePayoutRatio     types.Number `json:"orderAvgPayoutRatio"`
	CreateType                  string       `json:"createType"`
	CreatedTime                 types.Time   `json:"createdTime"`
	UpdatedTime                 types.Time   `json:"updatedTime"`
}

// EventPositionInfoRequest holds the filters for GetEventPositionInfo; every position is returned when Symbol is empty
type EventPositionInfoRequest struct {
	Symbol     string
	BaseCoin   currency.Code
	SettleCoin currency.Code
	Limit      uint64
	Cursor     string
}

// EventPositionInfoResponse is a page of event contract positions
type EventPositionInfoResponse struct {
	List           []EventPosition `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

// EventPosition is an event contract position. The event position stream pushes the same fields
type EventPosition struct {
	Symbol                 string        `json:"symbol"`
	BaseCoin               currency.Code `json:"baseCoin"`
	SettleCoin             currency.Code `json:"settleCoin"`
	Side                   string        `json:"side"`
	EventContractType      string        `json:"ecContractType"`
	EventContractDirection string        `json:"ecDirection"`
	AveragePayoutRatio     types.Number  `json:"avgPayoutRatio"`
	// PositionValue is the cumulative executed value
	PositionValue           types.Number `json:"positionValue"`
	EventContractSettleTime types.Time   `json:"ecSettleTime"`
	CumulativeClosedPNL     types.Number `json:"cumClosedPnl"`
	CreatedTime             types.Time   `json:"createdTime"`
	UpdatedTime             types.Time   `json:"updatedTime"`
	// Sequence is the cross sequence that orders the updates
	Sequence uint64 `json:"seq"`
}

// EventSettlementRecordsRequest holds the filters for GetEventSettlementRecords; Bybit queries the last 7 days by
// default
type EventSettlementRecordsRequest struct {
	Symbol    string
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// EventSettlementRecordsResponse is a page of event contract settlements
type EventSettlementRecordsResponse struct {
	List   []EventSettlement `json:"list"`
	Cursor string            `json:"cursor"`
}

// EventSettlement is the settlement of an event contract position
type EventSettlement struct {
	TransactionID string `json:"transId"`
	Symbol        string `json:"symbol"`
	// SymbolID is quoted here but a number in Get Instrument Info and the order stream
	SymbolID               types.Number  `json:"symbolId"`
	BaseCoin               currency.Code `json:"baseCoin"`
	SettleCoin             currency.Code `json:"settleCoin"`
	Side                   string        `json:"side"`
	EventContractType      string        `json:"ecContractType"`
	EventContractDirection string        `json:"ecDirection"`
	SettlePrice            types.Number  `json:"settlePrice"`
	EntryPrice             types.Number  `json:"entryPrice"`
	GrossPayoutRatio       types.Number  `json:"grossPayoutRatio"`
	// EventContractDurationWindow is in seconds
	EventContractDurationWindow types.Number `json:"ecDurationWindow"`
	// EventContractTargetPrice is set for Target contracts only
	EventContractTargetPrice types.Number `json:"ecTargetPrice"`
	// EventContractLowerBound and EventContractUpperBound are set for Range contracts only
	EventContractLowerBound types.Number `json:"ecLowerBound"`
	EventContractUpperBound types.Number `json:"ecUpperBound"`
	TransactionTime         types.Time   `json:"transTime"`
	CrossSequence           types.Number `json:"crossSeq"`
	ExecutionFee            types.Number `json:"execFee"`
	ExecutionFeeRate        types.Number `json:"execFeeRate"`
	Payout                  types.Number `json:"payout"`
	SessionRealisedPNL      types.Number `json:"sessionRpl"`
	OrderID                 string       `json:"orderId"`
}

// EventTradeHistoryRequest holds the filters for GetEventTradeHistory; Bybit queries the last 7 days by default
type EventTradeHistoryRequest struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// EventTradeHistoryResponse is a page of event contract executions
type EventTradeHistoryResponse struct {
	List   []EventTrade `json:"list"`
	Cursor string       `json:"cursor"`
}

// EventTrade is an execution of an event contract order
type EventTrade struct {
	OrderID       string        `json:"orderId"`
	OrderLinkID   string        `json:"orderLinkId"`
	ExecutionID   string        `json:"execId"`
	Symbol        string        `json:"symbol"`
	BaseCoin      currency.Code `json:"baseCoin"`
	SettleCoin    currency.Code `json:"settleCoin"`
	Side          string        `json:"side"`
	ExecutionType string        `json:"execType"`
	// ExecutionPrice is the payout ratio
	ExecutionPrice   types.Number  `json:"execPrice"`
	ExecutionValue   types.Number  `json:"execValue"`
	ExecutionFee     types.Number  `json:"execFee"`
	ExecutionFeeRate types.Number  `json:"execFeeRate"`
	OrderPrice       types.Number  `json:"orderPrice"`
	FeeCoin          currency.Code `json:"feeCoin"`
	TransactionTime  types.Time    `json:"transTime"`
	TimeInForce      string        `json:"timeInForce"`
	CrossSequence    types.Number  `json:"crossSeq"`
	// EventContractPayout is the settlement payout of the execution
	EventContractPayout types.Number `json:"ecPayout"`
	// EventContractDurationWindow is in seconds
	EventContractDurationWindow types.Number `json:"ecDurationWindow"`
	SettleTime                  types.Time   `json:"settleTimeMs"`
	EventContractOrderValue     types.Number `json:"ecOrderValue"`
	EventContractType           string       `json:"ecContractType"`
	EventContractDirection      string       `json:"ecDirection"`
	// EventContractTargetPrice is set for Target contracts only
	EventContractTargetPrice types.Number `json:"ecTargetPrice"`
	// EventContractLowerBound and EventContractUpperBound are set for Range contracts only
	EventContractLowerBound types.Number `json:"ecLowerBound"`
	EventContractUpperBound types.Number `json:"ecUpperBound"`
	// ExtraFees is a JSON encoded fee list when there are extra fees
	ExtraFees string `json:"extraFees"`
}

// SubmitEventQuoteRequest holds the parameters for SubmitEventQuote
type SubmitEventQuoteRequest struct {
	Symbol      string
	OrderLinkID string
	// GrossPayoutRatio is the institution's gross payout ratio
	GrossPayoutRatio float64
	// Amount is in the settle coin
	Amount float64
}

// EventQuoteResponse holds the identifiers of a submitted or cancelled event contract quote
type EventQuoteResponse struct {
	OrderLinkID string `json:"orderLinkId"`
	OrderID     string `json:"orderId"`
}
