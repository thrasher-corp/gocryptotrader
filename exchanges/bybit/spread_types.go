package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SpreadInstrumentsInfoRequest holds the filters for GetSpreadInstrumentsInfo
type SpreadInstrumentsInfoRequest struct {
	Symbol   string
	BaseCoin currency.Code
	Limit    uint64
	Cursor   string
}

// SpreadInstrumentsInfoResponse is a page of spread combination specifications
type SpreadInstrumentsInfoResponse struct {
	List           []SpreadInstrument `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// SpreadInstrument is the specification of a spread combination. Spread prices are the difference between the legs,
// so they can be zero or negative
type SpreadInstrument struct {
	Symbol       string        `json:"symbol"`
	ContractType string        `json:"contractType"`
	Status       string        `json:"status"`
	BaseCoin     currency.Code `json:"baseCoin"`
	QuoteCoin    currency.Code `json:"quoteCoin"`
	SettleCoin   currency.Code `json:"settleCoin"`
	TickSize     types.Number  `json:"tickSize"`
	MinPrice     types.Number  `json:"minPrice"`
	MaxPrice     types.Number  `json:"maxPrice"`
	LotSize      types.Number  `json:"lotSize"`
	MinSize      types.Number  `json:"minSize"`
	MaxSize      types.Number  `json:"maxSize"`
	LaunchTime   types.Time    `json:"launchTime"`
	// DeliveryTime is zero for combinations without a futures leg
	DeliveryTime types.Time            `json:"deliveryTime"`
	Legs         []SpreadInstrumentLeg `json:"legs"`
}

// SpreadInstrumentLeg is one leg of a spread combination
type SpreadInstrumentLeg struct {
	Symbol       string `json:"symbol"`
	ContractType string `json:"contractType"`
}

// SpreadOrderbookResponse is a spread combination's orderbook depth
type SpreadOrderbookResponse struct {
	Symbol    string                           `json:"s"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks      orderbook.LevelsArrayPriceAmount `json:"a"`
	Timestamp types.Time                       `json:"ts"`
	UpdateID  uint64                           `json:"u"`
	Sequence  uint64                           `json:"seq"`
	// CreationTimestamp is when the matching engine produced the depth, comparable with public trade times
	CreationTimestamp types.Time `json:"cts"`
}

// SpreadTickersResponse holds the ticker of a spread combination
type SpreadTickersResponse struct {
	List []SpreadTicker `json:"list"`
}

// SpreadTicker is a spread combination's best bid and ask, last price and 24 hour statistics. The spread websocket
// ticker topic pushes the same fields
type SpreadTicker struct {
	Symbol              string       `json:"symbol"`
	BidPrice            types.Number `json:"bidPrice"`
	BidSize             types.Number `json:"bidSize"`
	AskPrice            types.Number `json:"askPrice"`
	AskSize             types.Number `json:"askSize"`
	LastPrice           types.Number `json:"lastPrice"`
	HighPrice24Hour     types.Number `json:"highPrice24h"`
	LowPrice24Hour      types.Number `json:"lowPrice24h"`
	PreviousPrice24Hour types.Number `json:"prevPrice24h"`
	Volume24Hour        types.Number `json:"volume24h"`
}

// SpreadRecentPublicTradesResponse holds a spread combination's recent public trades
type SpreadRecentPublicTradesResponse struct {
	List []SpreadPublicTrade `json:"list"`
}

// SpreadPublicTrade is a public trade of a spread combination
type SpreadPublicTrade struct {
	ExecutionID string       `json:"execId"`
	Symbol      string       `json:"symbol"`
	Price       types.Number `json:"price"`
	Size        types.Number `json:"size"`
	Side        string       `json:"side"`
	Time        types.Time   `json:"time"`
	// Sequence is the quoted cross sequence
	Sequence types.Number `json:"seq"`
}

// CreateSpreadOrderRequest holds the parameters for CreateSpreadOrder
type CreateSpreadOrderRequest struct {
	Symbol    string
	Side      string
	OrderType string
	Quantity  float64
	// Price is sent when set; spread prices can be zero or negative
	Price       *float64
	OrderLinkID string
	TimeInForce string
}

// SpreadOrderResponse holds the identifiers of a spread order Bybit accepted for creation, amendment or cancellation
type SpreadOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// AmendSpreadOrderRequest holds the parameters for AmendSpreadOrder
type AmendSpreadOrderRequest struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	Quantity    float64
	// Price is sent when set; nil leaves the price unchanged and zero sets it to zero
	Price *float64
}

// CancelAllSpreadOrdersResponse lists the spread orders Bybit accepted for cancellation
type CancelAllSpreadOrdersResponse struct {
	List []SpreadOrderResponse `json:"list"`
	// Success is a flag Bybit documents as ignorable
	Success types.Boolean `json:"success"`
}

// SpreadOpenOrdersRequest holds the filters for GetSpreadOpenOrders
type SpreadOpenOrdersRequest struct {
	Symbol      string
	BaseCoin    currency.Code
	OrderID     string
	OrderLinkID string
	Limit       uint64
	Cursor      string
}

// SpreadOpenOrdersResponse is a page of unfilled and partially filled spread orders
type SpreadOpenOrdersResponse struct {
	List           []SpreadOpenOrder `json:"list"`
	NextPageCursor string            `json:"nextPageCursor"`
}

// SpreadOpenOrder is an unfilled or partially filled spread order
type SpreadOpenOrder struct {
	Symbol                     string        `json:"symbol"`
	BaseCoin                   currency.Code `json:"baseCoin"`
	OrderType                  string        `json:"orderType"`
	OrderLinkID                string        `json:"orderLinkId"`
	Side                       string        `json:"side"`
	TimeInForce                string        `json:"timeInForce"`
	OrderID                    string        `json:"orderId"`
	LeavesQuantity             types.Number  `json:"leavesQty"`
	OrderStatus                string        `json:"orderStatus"`
	CumulativeExecutedQuantity types.Number  `json:"cumExecQty"`
	Price                      types.Number  `json:"price"`
	Quantity                   types.Number  `json:"qty"`
	CreatedTime                types.Time    `json:"createdTime"`
	UpdatedTime                types.Time    `json:"updatedTime"`
}

// SpreadOrderHistoryRequest holds the filters for GetSpreadOrderHistory
type SpreadOrderHistoryRequest struct {
	Symbol      string
	BaseCoin    currency.Code
	OrderID     string
	OrderLinkID string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// SpreadOrderHistoryResponse is a page of spread orders in a final state
type SpreadOrderHistoryResponse struct {
	List           []SpreadHistoricOrder `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// SpreadHistoricOrder is a spread order in a final state with the orders of its two legs
type SpreadHistoricOrder struct {
	Symbol             string `json:"symbol"`
	OrderType          string `json:"orderType"`
	OrderLinkID        string `json:"orderLinkId"`
	OrderID            string `json:"orderId"`
	ContractType       string `json:"contractType"`
	CancelRejectReason string `json:"cxlRejReason"`
	OrderStatus        string `json:"orderStatus"`
	// Price is the combination's price, the difference between the legs, so it can be zero or negative
	Price         types.Number  `json:"price"`
	OrderQuantity types.Number  `json:"orderQty"`
	TimeInForce   string        `json:"timeInForce"`
	BaseCoin      currency.Code `json:"baseCoin"`
	CreatedAt     types.Time    `json:"createdAt"`
	UpdatedAt     types.Time    `json:"updatedAt"`
	Side          string        `json:"side"`
	// LeavesQuantity is meaningless for a cancelled order
	LeavesQuantity             types.Number  `json:"leavesQty"`
	SettleCoin                 currency.Code `json:"settleCoin"`
	CumulativeExecutedQuantity types.Number  `json:"cumExecQty"`
	Quantity                   types.Number  `json:"qty"`
	Leg1Symbol                 string        `json:"leg1Symbol"`
	Leg1ProductType            string        `json:"leg1ProdType"`
	Leg1OrderID                string        `json:"leg1OrderId"`
	Leg1Side                   string        `json:"leg1Side"`
	Leg2ProductType            string        `json:"leg2ProdType"`
	Leg2OrderID                string        `json:"leg2OrderId"`
	Leg2Symbol                 string        `json:"leg2Symbol"`
	Leg2Side                   string        `json:"leg2Side"`
}

// SpreadTradeHistoryRequest holds the filters for GetSpreadTradeHistory
type SpreadTradeHistoryRequest struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// SpreadTradeHistoryResponse is a page of spread combination executions
type SpreadTradeHistoryResponse struct {
	List           []SpreadExecution `json:"list"`
	NextPageCursor string            `json:"nextPageCursor"`
}

// SpreadExecution is an execution of a spread combination with the executions of its legs
type SpreadExecution struct {
	Symbol            string               `json:"symbol"`
	OrderLinkID       string               `json:"orderLinkId"`
	Side              string               `json:"side"`
	OrderID           string               `json:"orderId"`
	ExecutionPrice    types.Number         `json:"execPrice"`
	ExecutionTime     types.Time           `json:"execTime"`
	ExecutionType     string               `json:"execType"`
	ExecutionQuantity types.Number         `json:"execQty"`
	ExecutionID       string               `json:"execId"`
	Legs              []SpreadExecutionLeg `json:"legs"`
}

// SpreadExecutionLeg is the execution of one leg of a spread combination
type SpreadExecutionLeg struct {
	Symbol            string       `json:"symbol"`
	Side              string       `json:"side"`
	ExecutionPrice    types.Number `json:"execPrice"`
	ExecutionTime     types.Time   `json:"execTime"`
	ExecutionValue    types.Number `json:"execValue"`
	ExecutionType     string       `json:"execType"`
	Category          string       `json:"category"`
	ExecutionQuantity types.Number `json:"execQty"`
	// ExecutionFee is deprecated for the spot leg, whose fee is in ExecutionFeeV2
	ExecutionFee   types.Number  `json:"execFee"`
	ExecutionFeeV2 types.Number  `json:"execFeeV2"`
	FeeCurrency    currency.Code `json:"feeCurrency"`
	ExecutionID    string        `json:"execId"`
}

// SpreadMaxQuantityRequest holds the parameters for GetSpreadMaxQuantity
type SpreadMaxQuantityRequest struct {
	Symbol string
	// Side is "1" to buy or "2" to sell
	Side string
	// OrderPrice is always sent, since a spread price can be zero
	OrderPrice float64
}

// SpreadMaxQuantityResponse holds the maximum quantity a spread order can be placed for
type SpreadMaxQuantityResponse struct {
	// AvailableBalance is the maximum order quantity
	AvailableBalance types.Number `json:"ab"`
}
