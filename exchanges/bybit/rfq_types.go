package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateRFQRequest holds the parameters for CreateRFQ
type CreateRFQRequest struct {
	// Counterparties are the desk codes of the quoting parties the RFQ is sent to
	Counterparties []string
	RFQLinkID      string
	Anonymous      bool
	StrategyType   string
	List           []RFQLegRequest
	Hedge          []RFQHedgeRequest
}

// RFQLegRequest is a leg of an RFQ to create
type RFQLegRequest struct {
	Category string
	Symbol   string
	Side     string
	Quantity float64
}

// RFQHedgeRequest is the delta hedge leg of an RFQ to create
type RFQHedgeRequest struct {
	Category string
	Symbol   string
	Side     string
	Quantity float64
	// Price is the taker's reference price for the hedge, sent when set
	Price float64
}

// CreateRFQResponse holds the RFQ Bybit created
type CreateRFQResponse struct {
	RFQID     string     `json:"rfqId"`
	RFQLinkID string     `json:"rfqLinkId"`
	Status    string     `json:"status"`
	ExpiresAt types.Time `json:"expiresAt"`
	DeskCode  string     `json:"deskCode"`
}

// RFQConfigResponse holds the caller's desk code, RFQ limits and the quoting parties RFQs can be sent to
type RFQConfigResponse struct {
	DeskCode     string `json:"deskCode"`
	MaxLegs      uint64 `json:"maxLegs"`
	MaxLP        uint64 `json:"maxLP"`
	MaxActiveRFQ uint64 `json:"maxActiveRfq"`
	// RFQExpireTime is in minutes
	RFQExpireTime                 uint64            `json:"rfqExpireTime"`
	MinLimitQuantitySpotOrder     uint64            `json:"minLimitQtySpotOrder"`
	MinLimitQuantityContractOrder uint64            `json:"minLimitQtyContractOrder"`
	MinLimitQuantityOptionOrder   uint64            `json:"minLimitQtyOptionOrder"`
	StrategyTypes                 []RFQStrategyType `json:"strategyTypes"`
	Counterparties                []RFQCounterparty `json:"counterparties"`
}

// RFQStrategyType is a strategy an RFQ can be created for
type RFQStrategyType struct {
	StrategyName string `json:"strategyName"`
}

// RFQCounterparty is a quoting party an RFQ can be sent to
type RFQCounterparty struct {
	TraderName string `json:"traderName"`
	DeskCode   string `json:"deskCode"`
	// Type is LP for an automated market maker connected through the API, and empty for other quoting parties
	Type string `json:"type"`
}

// CancelRFQResponse holds the identifiers of the cancelled RFQ
type CancelRFQResponse struct {
	RFQID     string `json:"rfqId"`
	RFQLinkID string `json:"rfqLinkId"`
}

// RFQCancellation is the outcome of cancelling one RFQ
type RFQCancellation struct {
	RFQID     string `json:"rfqId"`
	RFQLinkID string `json:"rfqLinkId"`
	// Code is zero when the cancellation succeeded; the docs type it as a string but their example sends a number
	Code    types.Number `json:"code"`
	Message string       `json:"msg"`
}

// AcceptNonLPQuoteResponse holds the RFQ that now accepts quotes from parties other than liquidity providers
type AcceptNonLPQuoteResponse struct {
	RFQID string `json:"rfqId"`
}

// CreateRFQQuoteRequest holds the parameters for CreateRFQQuote
type CreateRFQQuoteRequest struct {
	RFQID       string
	QuoteLinkID string
	Anonymous   bool
	// ExpireIn is the quote's lifetime in seconds; Bybit defaults it to 60
	ExpireIn      uint64
	QuoteBuyList  []RFQQuoteLegRequest
	QuoteSellList []RFQQuoteLegRequest
}

// RFQQuoteLegRequest is the price of one leg of a quote to create
type RFQQuoteLegRequest struct {
	Category string
	Symbol   string
	// Price is sent when set; a hedge leg quoted without a price takes the hedge price in the RFQ
	Price float64
}

// CreateRFQQuoteResponse holds the quote Bybit created
type CreateRFQQuoteResponse struct {
	RFQID       string     `json:"rfqId"`
	QuoteID     string     `json:"quoteId"`
	QuoteLinkID string     `json:"quoteLinkId"`
	ExpiresAt   types.Time `json:"expiresAt"`
	DeskCode    string     `json:"deskCode"`
	Status      string     `json:"status"`
}

// ExecuteRFQQuoteRequest holds the parameters for ExecuteRFQQuote
type ExecuteRFQQuoteRequest struct {
	RFQID   string
	QuoteID string
	// QuoteSide is the side of the quote to execute: Buy executes the quoting party in the legs' direction and the
	// inquiring party opposite it
	QuoteSide string
	// IsHedge also executes the RFQ's hedge leg
	IsHedge bool
}

// ExecuteRFQQuoteResponse holds the quote execution Bybit accepted; the trade history and the execution stream confirm
// its outcome
type ExecuteRFQQuoteResponse struct {
	RFQID     string `json:"rfqId"`
	RFQLinkID string `json:"rfqLinkId"`
	QuoteID   string `json:"quoteId"`
	Status    string `json:"status"`
}

// CancelRFQQuoteRequest holds the identifiers for CancelRFQQuote; Bybit reads QuoteID first, then QuoteLinkID, then
// RFQID
type CancelRFQQuoteRequest struct {
	QuoteID     string
	RFQID       string
	QuoteLinkID string
}

// CancelRFQQuoteResponse holds the identifiers of the cancelled quote
type CancelRFQQuoteResponse struct {
	RFQID       string `json:"rfqId"`
	QuoteID     string `json:"quoteId"`
	QuoteLinkID string `json:"quoteLinkId"`
}

// RFQQuoteCancellation is the outcome of cancelling one quote
type RFQQuoteCancellation struct {
	RFQID       string `json:"rfqId"`
	QuoteID     string `json:"quoteId"`
	QuoteLinkID string `json:"quoteLinkId"`
	// Code is zero when the cancellation succeeded; the docs type it as a string but their example sends a number
	Code    types.Number `json:"code"`
	Message string       `json:"msg"`
}

// RFQsRealtimeRequest holds the filters for GetRFQsRealtime
type RFQsRealtimeRequest struct {
	RFQID string
	// RFQLinkID is ignored when TraderType is quote
	RFQLinkID  string
	TraderType string
}

// RFQsRealtimeResponse holds RFQs read without delay
type RFQsRealtimeResponse struct {
	List []RFQ `json:"list"`
}

// RFQ is a request for quote with its legs
type RFQ struct {
	RFQID          string     `json:"rfqId"`
	RFQLinkID      string     `json:"rfqLinkId"`
	Counterparties []string   `json:"counterparties"`
	ExpiresAt      types.Time `json:"expiresAt"`
	StrategyType   string     `json:"strategyType"`
	Status         string     `json:"status"`
	// AcceptOtherQuoteStatus is whether quotes from parties other than liquidity providers are accepted; these
	// endpoints quote it
	AcceptOtherQuoteStatus types.Boolean `json:"acceptOtherQuoteStatus"`
	// DeskCode is empty for an anonymous RFQ
	DeskCode  string        `json:"deskCode"`
	Anonymous bool          `json:"anonymous"`
	CreatedAt types.Time    `json:"createdAt"`
	UpdatedAt types.Time    `json:"updatedAt"`
	Legs      []RFQLeg      `json:"legs"`
	Hedge     []RFQHedgeLeg `json:"hedge"`
}

// RFQLeg is a leg of an RFQ
type RFQLeg struct {
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Side     string       `json:"side"`
	Quantity types.Number `json:"qty"`
}

// RFQHedgeLeg is the delta hedge leg of an RFQ
type RFQHedgeLeg struct {
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Side     string       `json:"side"`
	Quantity types.Number `json:"qty"`
	// Price is the taker's reference price for the hedge
	Price types.Number `json:"price"`
}

// RFQsRequest holds the filters for GetRFQs
type RFQsRequest struct {
	RFQID string
	// RFQLinkID only finds RFQs from the last 3 months and is ignored when TraderType is quote
	RFQLinkID  string
	TraderType string
	Status     string
	Limit      uint64
	Cursor     string
}

// RFQsResponse is a page of historical RFQs
type RFQsResponse struct {
	Cursor string `json:"cursor"`
	List   []RFQ  `json:"list"`
}

// RFQDetailsRequest holds the filters for GetRFQDetails
type RFQDetailsRequest struct {
	RFQID string
	// RFQLinkID only finds RFQs from the last 3 months
	RFQLinkID  string
	Status     string
	TraderType string
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
	Cursor     string
}

// RFQDetailsResponse is a page of historical RFQs with their quotes and trade legs
type RFQDetailsResponse struct {
	Cursor string      `json:"cursor"`
	List   []RFQDetail `json:"list"`
}

// RFQDetail is a historical RFQ with its active quotes and the caller's trade legs
type RFQDetail struct {
	RFQID string `json:"rfqId"`
	// RFQLinkID is empty for the quoting party
	RFQLinkID              string     `json:"rfqLinkId"`
	Counterparties         []string   `json:"counterparties"`
	ExpiresAt              types.Time `json:"expiresAt"`
	StrategyType           string     `json:"strategyType"`
	Status                 string     `json:"status"`
	AcceptOtherQuoteStatus bool       `json:"acceptOtherQuoteStatus"`
	// ExecutedQuoteID and ExecutedQuoteSide are set for a filled RFQ
	ExecutedQuoteID   string        `json:"execQuoteId"`
	ExecutedQuoteSide string        `json:"execQuoteSide"`
	DeskCode          string        `json:"deskCode"`
	Anonymous         bool          `json:"anonymous"`
	CreatedAt         types.Time    `json:"createdAt"`
	UpdatedAt         types.Time    `json:"updatedAt"`
	Legs              []RFQLeg      `json:"legs"`
	Hedge             []RFQHedgeLeg `json:"hedge"`
	// QuoteList is only returned to the inquiring party
	QuoteList []RFQDetailQuote    `json:"quoteList"`
	TradeLegs []RFQDetailTradeLeg `json:"tradeLegs"`
}

// RFQDetailQuote is an active quote on an RFQ
type RFQDetailQuote struct {
	QuoteID       string        `json:"quoteId"`
	DeskCode      string        `json:"deskCode"`
	Anonymous     bool          `json:"anonymous"`
	Status        string        `json:"status"`
	CreatedAt     types.Time    `json:"createdAt"`
	UpdatedAt     types.Time    `json:"updatedAt"`
	ExpiresAt     types.Time    `json:"expiresAt"`
	QuoteBuyList  []RFQQuoteLeg `json:"quoteBuyList"`
	QuoteSellList []RFQQuoteLeg `json:"quoteSellList"`
}

// RFQQuoteLeg is the price of one leg of a quote
type RFQQuoteLeg struct {
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	IsHedge  bool         `json:"isHedge"`
}

// RFQDetailTradeLeg is the execution of one of the caller's legs of an RFQ
type RFQDetailTradeLeg struct {
	Category string       `json:"category"`
	OrderID  string       `json:"orderId"`
	Symbol   string       `json:"symbol"`
	Side     string       `json:"side"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	// MarkPrice is the index price for spot and the underlying's mark price for options
	MarkPrice    types.Number `json:"markPrice"`
	ExecutionFee types.Number `json:"execFee"`
	ExtraFees    string       `json:"extraFees"`
	ExecutionID  string       `json:"execId"`
	// ResultCode is zero when the leg executed
	ResultCode    int64  `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
	// RejectParty is Taker or Maker for a rejected leg, bybit for an error on Bybit's side, and empty when filled
	RejectParty string `json:"rejectParty"`
}

// RFQQuotesRealtimeRequest holds the filters for GetRFQQuotesRealtime
type RFQQuotesRealtimeRequest struct {
	RFQID   string
	QuoteID string
	// QuoteLinkID is ignored when TraderType is request
	QuoteLinkID string
	TraderType  string
}

// RFQQuotesRealtimeResponse holds the quotes that are not in a final state, read without delay
type RFQQuotesRealtimeResponse struct {
	List []RFQQuote `json:"list"`
}

// RFQQuote is a quote on an RFQ
type RFQQuote struct {
	RFQID       string     `json:"rfqId"`
	RFQLinkID   string     `json:"rfqLinkId"`
	QuoteID     string     `json:"quoteId"`
	QuoteLinkID string     `json:"quoteLinkId"`
	ExpiresAt   types.Time `json:"expiresAt"`
	DeskCode    string     `json:"deskCode"`
	Anonymous   bool       `json:"anonymous"`
	Status      string     `json:"status"`
	// ExecutedQuoteSide is the side of the quote that was executed
	ExecutedQuoteSide string        `json:"execQuoteSide"`
	CreatedAt         types.Time    `json:"createdAt"`
	UpdatedAt         types.Time    `json:"updatedAt"`
	QuoteBuyList      []RFQQuoteLeg `json:"quoteBuyList"`
	QuoteSellList     []RFQQuoteLeg `json:"quoteSellList"`
}

// RFQQuotesRequest holds the filters for GetRFQQuotes
type RFQQuotesRequest struct {
	RFQID   string
	QuoteID string
	// QuoteLinkID is ignored when TraderType is request
	QuoteLinkID string
	TraderType  string
	Status      string
	Limit       uint64
	Cursor      string
}

// RFQQuotesResponse is a page of historical quotes
type RFQQuotesResponse struct {
	Cursor string     `json:"cursor"`
	List   []RFQQuote `json:"list"`
}

// RFQTradeHistoryRequest holds the filters for GetRFQTradeHistory
type RFQTradeHistoryRequest struct {
	RFQID string
	// RFQLinkID only finds trades from the last 3 months
	RFQLinkID string
	QuoteID   string
	// QuoteLinkID only finds trades from the last 3 months
	QuoteLinkID string
	TraderType  string
	Status      string
	Limit       uint64
	Cursor      string
}

// RFQTradeHistoryResponse is a page of the caller's RFQ trades
type RFQTradeHistoryResponse struct {
	Cursor string     `json:"cursor"`
	List   []RFQTrade `json:"list"`
}

// RFQTrade is the execution of an RFQ the caller took part in. The RFQ execution stream pushes the same fields
type RFQTrade struct {
	RFQID         string        `json:"rfqId"`
	RFQLinkID     string        `json:"rfqLinkId"`
	QuoteID       string        `json:"quoteId"`
	QuoteLinkID   string        `json:"quoteLinkId"`
	QuoteSide     string        `json:"quoteSide"`
	StrategyType  string        `json:"strategyType"`
	Status        string        `json:"status"`
	RFQDeskCode   string        `json:"rfqDeskCode"`
	QuoteDeskCode string        `json:"quoteDeskCode"`
	CreatedAt     types.Time    `json:"createdAt"`
	UpdatedAt     types.Time    `json:"updatedAt"`
	Legs          []RFQTradeLeg `json:"legs"`
}

// RFQTradeLeg is the execution of one leg of an RFQ trade
type RFQTradeLeg struct {
	Category string       `json:"category"`
	OrderID  string       `json:"orderId"`
	Symbol   string       `json:"symbol"`
	Side     string       `json:"side"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	// MarkPrice is the index price for spot and the underlying's mark price for options
	MarkPrice    types.Number `json:"markPrice"`
	ExecutionFee types.Number `json:"execFee"`
	ExecutionID  string       `json:"execId"`
	// ResultCode is zero when the leg executed
	ResultCode    int64  `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
	// RejectParty is Taker or Maker for a rejected leg, bybit for an error on Bybit's side, and empty when filled
	RejectParty string `json:"rejectParty"`
}

// RFQPublicTradesRequest holds the filters for GetRFQPublicTrades; the window cannot exceed 30 days
type RFQPublicTradesRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// RFQPublicTradesResponse is a page of recently executed RFQs
type RFQPublicTradesResponse struct {
	Cursor string           `json:"cursor"`
	List   []RFQPublicTrade `json:"list"`
}

// RFQPublicTrade is an executed RFQ without party information. The RFQ public trade stream pushes the same fields
type RFQPublicTrade struct {
	RFQID        string              `json:"rfqId"`
	StrategyType string              `json:"strategyType"`
	CreatedAt    types.Time          `json:"createdAt"`
	UpdatedAt    types.Time          `json:"updatedAt"`
	Legs         []RFQPublicTradeLeg `json:"legs"`
}

// RFQPublicTradeLeg is the execution of one leg of a public RFQ trade
type RFQPublicTradeLeg struct {
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Side     string       `json:"side"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	// MarkPrice is the index price for spot and the underlying's mark price for options
	MarkPrice types.Number `json:"markPrice"`
}
