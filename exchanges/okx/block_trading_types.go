package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CounterpartiesResponse holds a counterparty the user is permitted to trade block trades with
type CounterpartiesResponse struct {
	TraderName string `json:"traderName"`
	TraderCode string `json:"traderCode"`
	Type       string `json:"type"`
}

// CreateRequestForQuoteRequest holds the parameters of an RFQ to create
type CreateRequestForQuoteRequest struct {
	Counterparties          []string                                 `json:"counterparties"`
	Anonymous               bool                                     `json:"anonymous,omitempty"`
	ClientRequestForQuoteID string                                   `json:"clRfqId,omitempty"`
	Tag                     string                                   `json:"tag,omitempty"`
	AllowPartialExecution   bool                                     `json:"allowPartialExecution,omitempty"`
	Legs                    []CreateRequestForQuoteLeg               `json:"legs"`
	AccountAllocations      []CreateRequestForQuoteAccountAllocation `json:"acctAlloc,omitempty"`
}

// CreateRequestForQuoteLeg holds one leg of an RFQ to create
type CreateRequestForQuoteLeg struct {
	InstrumentID string        `json:"instId"`
	TradeMode    string        `json:"tdMode,omitempty"`
	Currency     currency.Code `json:"ccy,omitzero"`
	Size         float64       `json:"sz,string"`
	// LimitPrice executes the RFQ automatically against a quote at that price or better; set it on every leg or on none
	LimitPrice         float64       `json:"lmtPx,string,omitempty"`
	Side               string        `json:"side"`
	PositionSide       string        `json:"posSide,omitempty"`
	TargetCurrency     string        `json:"tgtCcy,omitempty"` // base_ccy or quote_ccy, the unit of Size
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`
}

// CreateRequestForQuoteAccountAllocation allocates the legs of an RFQ to create to an account
type CreateRequestForQuoteAccountAllocation struct {
	Account string                               `json:"acct"`
	Legs    []CreateRequestForQuoteAllocationLeg `json:"legs"`
}

// CreateRequestForQuoteAllocationLeg holds one leg of an account allocation of an RFQ to create
type CreateRequestForQuoteAllocationLeg struct {
	InstrumentID string        `json:"instId"`
	Size         float64       `json:"sz,string"`
	TradeMode    string        `json:"tdMode,omitempty"`
	Currency     currency.Code `json:"ccy,omitzero"`
	PositionSide string        `json:"posSide,omitempty"`
}

// RequestForQuoteResponse holds an RFQ
type RequestForQuoteResponse struct {
	CreationTime            types.Time                         `json:"cTime"`
	UpdateTime              types.Time                         `json:"uTime"`
	ValidUntil              types.Time                         `json:"validUntil"`
	TraderCode              string                             `json:"traderCode"`
	RequestForQuoteID       string                             `json:"rfqId"`
	ClientRequestForQuoteID string                             `json:"clRfqId"`
	Tag                     string                             `json:"tag"`
	State                   string                             `json:"state"`
	FlowType                string                             `json:"flowType"`
	Counterparties          []string                           `json:"counterparties"`
	AllowPartialExecution   bool                               `json:"allowPartialExecution"`
	GroupID                 string                             `json:"groupId"`
	Legs                    []RequestForQuoteOrderLeg          `json:"legs"`
	AccountAllocations      []RequestForQuoteAccountAllocation `json:"acctAlloc"`
}

// RequestForQuoteOrderLeg holds one leg of an RFQ
type RequestForQuoteOrderLeg struct {
	InstrumentID       string        `json:"instId"`
	TradeMode          string        `json:"tdMode"`
	Currency           currency.Code `json:"ccy"`
	Size               types.Number  `json:"sz"`
	Side               string        `json:"side"`
	PositionSide       string        `json:"posSide"`
	TargetCurrency     string        `json:"tgtCcy"` // base_ccy or quote_ccy, the unit of Size
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// RequestForQuoteAccountAllocation holds an account's allocation of an RFQ or of its block trade
type RequestForQuoteAccountAllocation struct {
	Account       string                         `json:"acct"`
	BlockTradeID  string                         `json:"blockTdId"`
	StatusCode    string                         `json:"sCode"`
	StatusMessage string                         `json:"sMsg"`
	ErrorCode     string                         `json:"errorCode"`
	Legs          []RequestForQuoteAllocationLeg `json:"legs"`
}

// RequestForQuoteAllocationLeg holds one leg of an account allocation
type RequestForQuoteAllocationLeg struct {
	InstrumentID string        `json:"instId"`
	Size         types.Number  `json:"sz"`
	TradeMode    string        `json:"tdMode"`
	Currency     currency.Code `json:"ccy"`
	PositionSide string        `json:"posSide"`
	Fee          types.Number  `json:"fee"`
	FeeCurrency  currency.Code `json:"feeCcy"`
	TradeID      string        `json:"tradeId"`
}

// CancelRequestForQuoteRequest holds the RFQ to cancel; OKX uses RequestForQuoteID when both IDs are set
type CancelRequestForQuoteRequest struct {
	RequestForQuoteID       string `json:"rfqId,omitempty"`
	ClientRequestForQuoteID string `json:"clRfqId,omitempty"`
}

// CancelMultipleRequestForQuotesRequest holds the RFQs to cancel; OKX uses RequestForQuoteIDs when both lists are set
type CancelMultipleRequestForQuotesRequest struct {
	RequestForQuoteIDs       []string `json:"rfqIds,omitempty"`
	ClientRequestForQuoteIDs []string `json:"clRfqIds,omitempty"`
}

// CancelRequestForQuoteResponse holds the result of cancelling an RFQ
type CancelRequestForQuoteResponse struct {
	RequestForQuoteID       string `json:"rfqId"`
	ClientRequestForQuoteID string `json:"clRfqId"`
	StatusCode              string `json:"sCode"`
	StatusMessage           string `json:"sMsg"`
}

// BlockTradingTimestampResponse holds the time OKX processed a block trading request
type BlockTradingTimestampResponse struct {
	Timestamp types.Time `json:"ts"`
}

// ExecuteQuoteRequest holds the quote to execute
type ExecuteQuoteRequest struct {
	RequestForQuoteID string `json:"rfqId"`
	QuoteID           string `json:"quoteId"`
	// Legs sets the execution size of each leg for a partial execution, in the ratio of the RFQ's leg sizes
	Legs []ExecuteQuoteLeg `json:"legs,omitempty"`
}

// ExecuteQuoteLeg holds the execution size of one leg of a quote
type ExecuteQuoteLeg struct {
	InstrumentID string  `json:"instId"`
	Size         float64 `json:"sz,string"`
}

// ExecuteQuoteResponse holds the block trade an executed quote made
type ExecuteQuoteResponse struct {
	CreationTime            types.Time                         `json:"cTime"`
	RequestForQuoteID       string                             `json:"rfqId"`
	ClientRequestForQuoteID string                             `json:"clRfqId"`
	QuoteID                 string                             `json:"quoteId"`
	ClientQuoteID           string                             `json:"clQuoteId"`
	BlockTradeID            string                             `json:"blockTdId"`
	Tag                     string                             `json:"tag"`
	TakerTraderCode         string                             `json:"tTraderCode"`
	MakerTraderCode         string                             `json:"mTraderCode"`
	Legs                    []BlockTradeLeg                    `json:"legs"`
	AccountAllocations      []RequestForQuoteAccountAllocation `json:"acctAlloc"`
}

// BlockTradeLeg holds one leg of an executed block trade
type BlockTradeLeg struct {
	InstrumentID       string        `json:"instId"`
	Price              types.Number  `json:"px"`
	Size               types.Number  `json:"sz"`
	Side               string        `json:"side"`
	Fee                types.Number  `json:"fee"`
	FeeCurrency        currency.Code `json:"feeCcy"`
	TradeID            string        `json:"tradeId"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// QuoteProductResponse holds the products a maker quotes for one instrument type
type QuoteProductResponse struct {
	InstrumentType string               `json:"instType"`
	IncludeAll     bool                 `json:"includeAll"`
	Data           []QuoteProductDetail `json:"data"`
}

// QuoteProductDetail holds a product a maker quotes and its limits
type QuoteProductDetail struct {
	InstrumentFamily string       `json:"instFamily"`
	InstrumentID     string       `json:"instId"`
	MaxBlockSize     types.Number `json:"maxBlockSz"`
	MakerPriceBand   types.Number `json:"makerPxBand"` // in ticks from the mark price
}

// SetQuoteProductRequest holds the products a maker quotes for one instrument type
type SetQuoteProductRequest struct {
	InstrumentType string                   `json:"instType"`
	IncludeAll     bool                     `json:"includeAll,omitempty"`
	Data           []MakerInstrumentSetting `json:"data"`
}

// MakerInstrumentSetting holds a product a maker quotes and its limits; FUTURES, OPTION and SWAP take an instrument
// family and SPOT an instrument ID
type MakerInstrumentSetting struct {
	InstrumentFamily string  `json:"instFamily,omitempty"`
	InstrumentID     string  `json:"instId,omitempty"`
	MaxBlockSize     float64 `json:"maxBlockSz,string,omitempty"`
	MakerPriceBand   float64 `json:"makerPxBand,string,omitempty"` // in ticks from the mark price
}

// SetQuoteProductsResponse holds whether OKX applied the quote products
type SetQuoteProductsResponse struct {
	Result bool `json:"result"`
}

// SetRequestForQuoteMMPRequest holds the block trading maker MMP configuration to set
type SetRequestForQuoteMMPRequest struct {
	TimeInterval   uint64 `json:"timeInterval,string"`   // milliseconds of monitoring; 0 disables MMP
	FrozenInterval uint64 `json:"frozenInterval,string"` // milliseconds frozen; 0 freezes until ResetRequestForQuoteMMPStatus
	CountLimit     uint64 `json:"countLimit,string"`     // execution attempts allowed within TimeInterval
}

// SetRequestForQuoteMMPResponse holds the block trading maker MMP configuration OKX applied
type SetRequestForQuoteMMPResponse struct {
	TimeInterval   types.Number `json:"timeInterval"`   // milliseconds
	FrozenInterval types.Number `json:"frozenInterval"` // milliseconds
	CountLimit     types.Number `json:"countLimit"`
}

// RequestForQuoteMMPConfigResponse holds the block trading maker MMP configuration and state
type RequestForQuoteMMPConfigResponse struct {
	TimeInterval   types.Number `json:"timeInterval"`   // milliseconds; 0 when MMP is disabled
	FrozenInterval types.Number `json:"frozenInterval"` // milliseconds
	CountLimit     types.Number `json:"countLimit"`
	MMPFrozen      bool         `json:"mmpFrozen"`
	MMPFrozenUntil types.Number `json:"mmpFrozenUntil"` // milliseconds until MMP is no longer triggered
}

// CreateQuoteRequest holds the parameters of a quote to create in response to an RFQ
type CreateQuoteRequest struct {
	RequestForQuoteID string           `json:"rfqId"`
	ClientQuoteID     string           `json:"clQuoteId,omitempty"`
	Tag               string           `json:"tag,omitempty"`
	Anonymous         bool             `json:"anonymous,omitempty"`
	QuoteSide         string           `json:"quoteSide"`
	ExpiresIn         uint64           `json:"expiresIn,string,omitempty"` // seconds, from 10 to 120; OKX defaults to 60
	Legs              []CreateQuoteLeg `json:"legs"`
}

// CreateQuoteLeg holds one leg of a quote to create
type CreateQuoteLeg struct {
	InstrumentID       string        `json:"instId"`
	TradeMode          string        `json:"tdMode,omitempty"`
	Currency           currency.Code `json:"ccy,omitzero"`
	Size               float64       `json:"sz,string"`
	Price              float64       `json:"px,string"`
	Side               string        `json:"side"`
	PositionSide       string        `json:"posSide,omitempty"`
	TargetCurrency     string        `json:"tgtCcy,omitempty"` // base_ccy or quote_ccy, the unit of Size
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`
}

// QuoteResponse holds a quote
type QuoteResponse struct {
	CreationTime            types.Time `json:"cTime"`
	UpdateTime              types.Time `json:"uTime"`
	ValidUntil              types.Time `json:"validUntil"`
	QuoteID                 string     `json:"quoteId"`
	ClientQuoteID           string     `json:"clQuoteId"`
	RequestForQuoteID       string     `json:"rfqId"`
	ClientRequestForQuoteID string     `json:"clRfqId"`
	QuoteSide               string     `json:"quoteSide"`
	TraderCode              string     `json:"traderCode"`
	State                   string     `json:"state"`
	Reason                  string     `json:"reason"`
	Tag                     string     `json:"tag"`
	Legs                    []QuoteLeg `json:"legs"`
}

// CancelQuoteRequest holds the quote to cancel; OKX uses QuoteID when both quote IDs are set
type CancelQuoteRequest struct {
	QuoteID           string `json:"quoteId,omitempty"`
	ClientQuoteID     string `json:"clQuoteId,omitempty"`
	RequestForQuoteID string `json:"rfqId,omitempty"`
}

// CancelMultipleQuotesRequest holds the quotes to cancel; OKX uses QuoteIDs when both lists are set
type CancelMultipleQuotesRequest struct {
	QuoteIDs       []string `json:"quoteIds,omitempty"`
	ClientQuoteIDs []string `json:"clQuoteIds,omitempty"`
}

// CancelQuoteResponse holds the result of cancelling a quote
type CancelQuoteResponse struct {
	QuoteID       string `json:"quoteId"`
	ClientQuoteID string `json:"clQuoteId"`
	StatusCode    string `json:"sCode"`
	StatusMessage string `json:"sMsg"`
}

// CancelAllQuotesAfterCountdownRequest holds the countdown after which OKX cancels every quote
type CancelAllQuotesAfterCountdownRequest struct {
	Timeout uint64 `json:"timeOut,string"` // seconds; 0 disables the countdown
}

// CancelAllQuotesAfterCountdownResponse holds the state of the countdown after which OKX cancels every quote
type CancelAllQuotesAfterCountdownResponse struct {
	TriggerTime types.Time `json:"triggerTime"` // zero when the countdown is disabled
	Timestamp   types.Time `json:"ts"`
}

// RequestForQuotesRequest holds the filters of an RFQ listing
type RequestForQuotesRequest struct {
	RequestForQuoteID       string
	ClientRequestForQuoteID string
	State                   string
	BeginID                 string // returns RFQs newer than this RFQ ID
	EndID                   string // returns RFQs older than this RFQ ID
	Limit                   uint64
}

// QuotesRequest holds the filters of a quote listing
type QuotesRequest struct {
	RequestForQuoteID       string
	ClientRequestForQuoteID string
	QuoteID                 string
	ClientQuoteID           string
	State                   string
	BeginID                 string // returns quotes newer than this quote ID
	EndID                   string // returns quotes older than this quote ID
	Limit                   uint64
}

// RequestForQuoteTradesRequest holds the filters of a block trade listing
type RequestForQuoteTradesRequest struct {
	RequestForQuoteID       string
	ClientRequestForQuoteID string
	QuoteID                 string
	ClientQuoteID           string
	BlockTradeID            string
	BeginID                 string // returns block trades newer than this block trade ID
	EndID                   string // returns block trades older than this block trade ID
	BeginTime               time.Time
	EndTime                 time.Time
	Limit                   uint64
	// IsSuccessful selects successful or unsuccessful trades; OKX returns successful ones when it is nil
	IsSuccessful *bool
}

// RequestForQuoteTradeResponse holds a block trade the user is a counterparty to
type RequestForQuoteTradeResponse struct {
	CreationTime            types.Time                         `json:"cTime"`
	RequestForQuoteID       string                             `json:"rfqId"`
	ClientRequestForQuoteID string                             `json:"clRfqId"`
	QuoteID                 string                             `json:"quoteId"`
	ClientQuoteID           string                             `json:"clQuoteId"`
	BlockTradeID            string                             `json:"blockTdId"`
	Tag                     string                             `json:"tag"`
	TakerTraderCode         string                             `json:"tTraderCode"`
	MakerTraderCode         string                             `json:"mTraderCode"`
	IsSuccessful            bool                               `json:"isSuccessful"`
	ErrorCode               string                             `json:"errorCode"`
	Legs                    []BlockTradeLeg                    `json:"legs"`
	AccountAllocations      []RequestForQuoteAccountAllocation `json:"acctAlloc"`
}

// GetBlockTickersRequest holds the filters of a block ticker listing
type GetBlockTickersRequest struct {
	InstrumentType   string
	InstrumentFamily string // FUTURES, SWAP and OPTION only
}

// BlockTickerResponse holds an instrument's block trading volume over the last 24 hours
type BlockTickerResponse struct {
	InstrumentType                string       `json:"instType"`
	InstrumentID                  string       `json:"instId"`
	TradingVolume24HourInCurrency types.Number `json:"volCcy24h"`
	TradingVolume24HourInContract types.Number `json:"vol24h"`
	Timestamp                     types.Time   `json:"ts"`
}

// GetPublicRequestForQuoteTradesRequest holds the pagination of a public multi-leg block trade listing
type GetPublicRequestForQuoteTradesRequest struct {
	BeginID string // returns block trades newer than this block trade ID
	EndID   string // returns block trades older than this block trade ID
	Limit   uint64
}

// PublicTradesResponse holds a public multi-leg block trade
type PublicTradesResponse struct {
	BlockTradeID string                `json:"blockTdId"`
	GroupID      string                `json:"groupId"`
	Strategy     string                `json:"strategy"`
	Inverse      bool                  `json:"inverse"` // sent by OKX though its docs omit it
	CreationTime types.Time            `json:"cTime"`
	Legs         []PublicBlockTradeLeg `json:"legs"`
}

// PublicBlockTradeLeg holds one leg of a public multi-leg block trade
type PublicBlockTradeLeg struct {
	InstrumentID string       `json:"instId"`
	Side         string       `json:"side"`
	Size         types.Number `json:"sz"`
	Price        types.Number `json:"px"`
	TradeID      string       `json:"tradeId"`
}

// PublicBlockTradeResponse holds a public single-leg block trade
type PublicBlockTradeResponse struct {
	InstrumentID   string       `json:"instId"`
	TradeID        string       `json:"tradeId"`
	Price          types.Number `json:"px"`
	Size           types.Number `json:"sz"`
	Side           order.Side   `json:"side"`
	FillVolatility types.Number `json:"fillVol"` // implied volatility, options only
	ForwardPrice   types.Number `json:"fwdPx"`   // options only
	IndexPrice     types.Number `json:"idxPx"`
	MarkPrice      types.Number `json:"markPx"`
	GroupID        string       `json:"groupId"`
	Timestamp      types.Time   `json:"ts"`
}
