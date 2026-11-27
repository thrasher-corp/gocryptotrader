package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CancelSpreadOrderRequest holds the spread order to cancel; OKX uses OrderID when both IDs are set
type CancelSpreadOrderRequest struct {
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// CancelAllSpreadOrdersRequest holds the spread whose pending orders to cancel; an empty SpreadID cancels the pending
// orders of every spread
type CancelAllSpreadOrdersRequest struct {
	SpreadID string `json:"sprdId,omitempty"`
}

// CancelAllSpreadOrdersResponse holds whether OKX accepted a request to cancel pending spread orders
type CancelAllSpreadOrdersResponse struct {
	Result bool `json:"result"`
}

// AmendSpreadOrderRequest holds the spread order to amend; OKX uses OrderID when both IDs are set
type AmendSpreadOrderRequest struct {
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
	RequestID     string `json:"reqId,omitempty"`
	// NewSize includes any size already filled; zero leaves the size unchanged
	NewSize float64 `json:"newSz,string,omitempty"`
	// NewPrice is nil to leave the price unchanged, as a spread trades at zero and negative prices too
	NewPrice *float64 `json:"newPx,omitempty,string"`
}

// GetSpreadOrderDetailsRequest holds the spread order to retrieve; OKX uses OrderID when both IDs are set, and the
// latest order with ClientOrderID otherwise
type GetSpreadOrderDetailsRequest struct {
	OrderID       string
	ClientOrderID string
}

// SpreadOrderDetailResponse holds a spread order
type SpreadOrderDetailResponse struct {
	SpreadID            string         `json:"sprdId"`
	OrderID             string         `json:"ordId"`
	ClientOrderID       string         `json:"clOrdId"`
	Tag                 string         `json:"tag"`
	Price               types.Number   `json:"px"`
	Size                types.Number   `json:"sz"`
	OrderType           string         `json:"ordType"`
	Side                string         `json:"side"`
	FillSize            types.Number   `json:"fillSz"`
	FillPrice           types.Number   `json:"fillPx"`
	TradeID             string         `json:"tradeId"`
	AccumulatedFillSize types.Number   `json:"accFillSz"`
	PendingFillSize     OptionalNumber `json:"pendingFillSz"`
	PendingSettleSize   types.Number   `json:"pendingSettleSz"`
	CancelledSize       types.Number   `json:"canceledSz"`
	AveragePrice        types.Number   `json:"avgPx"`
	State               string         `json:"state"`
	CancelSource        string         `json:"cancelSource"`
	UpdateTime          types.Time     `json:"uTime"`
	CreationTime        types.Time     `json:"cTime"`
}

// GetActiveSpreadOrdersRequest holds the filters of a pending spread order listing
type GetActiveSpreadOrdersRequest struct {
	SpreadID  string
	OrderType string
	State     string
	BeginID   string // returns orders newer than this order ID
	EndID     string // returns orders older than this order ID
	Limit     uint64
}

// GetCompletedSpreadOrdersRequest holds the filters of a listing of the spread orders completed in the last 21 days
type GetCompletedSpreadOrdersRequest struct {
	SpreadID  string
	OrderType string
	State     string
	BeginID   string // returns orders newer than this order ID
	EndID     string // returns orders older than this order ID
	BeginTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// GetCompletedSpreadOrdersArchiveRequest holds the filters of a listing of the spread orders completed in the last 3
// months
type GetCompletedSpreadOrdersArchiveRequest struct {
	SpreadID         string
	OrderType        string
	State            string
	InstrumentType   string // selects spreads with a leg of this instrument type
	InstrumentFamily string // selects spreads with a leg of this instrument family
	BeginID          string // returns orders newer than this order ID
	EndID            string // returns orders older than this order ID
	BeginTime        time.Time
	EndTime          time.Time
	Limit            uint64
}

// GetSpreadTradesRequest holds the filters of a listing of the spread trades of the last 7 days
type GetSpreadTradesRequest struct {
	SpreadID  string
	TradeID   string
	OrderID   string
	BeginID   string // returns trades newer than this trade ID
	EndID     string // returns trades older than this trade ID
	BeginTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// SpreadTradeResponse holds a spread trade
type SpreadTradeResponse struct {
	SpreadID      string           `json:"sprdId"`
	TradeID       string           `json:"tradeId"`
	OrderID       string           `json:"ordId"`
	ClientOrderID string           `json:"clOrdId"`
	Tag           string           `json:"tag"`
	FillPrice     types.Number     `json:"fillPx"`
	FillSize      types.Number     `json:"fillSz"`
	Side          string           `json:"side"`
	State         string           `json:"state"`
	ExecutionType string           `json:"execType"`
	Timestamp     types.Time       `json:"ts"`
	Legs          []SpreadTradeLeg `json:"legs"`
	Code          string           `json:"code"`
	Message       string           `json:"msg"`
}

// SpreadTradeLeg holds one leg of a spread trade
type SpreadTradeLeg struct {
	InstrumentID    string        `json:"instId"`
	Price           types.Number  `json:"px"`
	Size            types.Number  `json:"sz"`
	SizeInContracts types.Number  `json:"szCont"`
	Side            string        `json:"side"`
	FillPNL         types.Number  `json:"fillPnl"`
	Fee             types.Number  `json:"fee"`
	FeeCurrency     currency.Code `json:"feeCcy"`
	TradeID         string        `json:"tradeId"`
}

// GetPublicSpreadsRequest holds the filters of a spread listing
type GetPublicSpreadsRequest struct {
	BaseCurrency currency.Code
	InstrumentID string // selects spreads with this instrument as a leg
	SpreadID     string
	State        string
}

// SpreadInstrumentResponse holds a spread
type SpreadInstrumentResponse struct {
	SpreadID      currency.Pair         `json:"sprdId"`
	SpreadType    string                `json:"sprdType"`
	State         string                `json:"state"`
	BaseCurrency  currency.Code         `json:"baseCcy"`
	SizeCurrency  currency.Code         `json:"szCcy"`
	QuoteCurrency currency.Code         `json:"quoteCcy"`
	TickSize      types.Number          `json:"tickSz"`
	MinSize       types.Number          `json:"minSz"`
	LotSize       types.Number          `json:"lotSz"`
	ListTime      types.Time            `json:"listTime"`
	ExpiryTime    types.Time            `json:"expTime"`
	UpdateTime    types.Time            `json:"uTime"`
	Legs          []SpreadInstrumentLeg `json:"legs"`
}

// SpreadInstrumentLeg holds one leg of a spread
type SpreadInstrumentLeg struct {
	InstrumentID string `json:"instId"`
	Side         string `json:"side"`
}

// GetPublicSpreadOrderBooksRequest holds the spread order book to retrieve
type GetPublicSpreadOrderBooksRequest struct {
	SpreadID string
	Depth    uint64 // levels per side; OKX defaults to 5 and allows 400
}

// SpreadOrderBookResponse holds a spread order book; each level holds its price, size and number of orders
type SpreadOrderBookResponse struct {
	Asks      [][]types.Number `json:"asks"`
	Bids      [][]types.Number `json:"bids"`
	Timestamp types.Time       `json:"ts"`
}

// SpreadTickerResponse holds a spread's latest trade, best bid and ask, and its trading over the last 24 hours
type SpreadTickerResponse struct {
	SpreadID            string       `json:"sprdId"`
	Last                types.Number `json:"last"`
	LastSize            types.Number `json:"lastSz"`
	AskPrice            types.Number `json:"askPx"`
	AskSize             types.Number `json:"askSz"`
	BidPrice            types.Number `json:"bidPx"`
	BidSize             types.Number `json:"bidSz"`
	OpenPrice24Hour     types.Number `json:"open24h"`
	HighestPrice24Hour  types.Number `json:"high24h"`
	LowestPrice24Hour   types.Number `json:"low24h"`
	TradingVolume24Hour types.Number `json:"vol24h"` // USD on an inverse spread, the base currency on a linear or hybrid one
	Timestamp           types.Time   `json:"ts"`
}

// GetPublicSpreadTradesRequest holds the spread whose public trades to retrieve; an empty SpreadID retrieves the trades
// of every spread
type GetPublicSpreadTradesRequest struct {
	SpreadID string
}

// SpreadPublicTradeResponse holds a public spread trade
type SpreadPublicTradeResponse struct {
	SpreadID  string       `json:"sprdId"`
	TradeID   string       `json:"tradeId"`
	Price     types.Number `json:"px"`
	Size      types.Number `json:"sz"`
	Side      string       `json:"side"` // the taker's side
	Timestamp types.Time   `json:"ts"`
}

// GetSpreadCandlesticksRequest holds the spread candlesticks to retrieve
type GetSpreadCandlesticksRequest struct {
	SpreadID string
	Interval kline.Interval
	Before   time.Time // returns candlesticks newer than this time
	After    time.Time // returns candlesticks older than this time
	Limit    uint64
}

// CancelAllSpreadOrdersAfterCountdownRequest holds the countdown after which OKX cancels every pending spread order
type CancelAllSpreadOrdersAfterCountdownRequest struct {
	Timeout uint64 `json:"timeOut,string"` // seconds; 0 disables the countdown
}

// CancelAllSpreadOrdersAfterCountdownResponse holds the state of the countdown after which OKX cancels every pending
// spread order
type CancelAllSpreadOrdersAfterCountdownResponse struct {
	TriggerTime types.Time `json:"triggerTime"` // zero when the countdown is disabled
	Timestamp   types.Time `json:"ts"`
}
