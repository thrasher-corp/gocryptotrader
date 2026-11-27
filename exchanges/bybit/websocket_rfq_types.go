package bybit

import "github.com/thrasher-corp/gocryptotrader/types"

// WebsocketRFQPublicTradeUpdate holds executed RFQs without party information; all legs of a block trade arrive together
type WebsocketRFQPublicTradeUpdate struct {
	ID           string           `json:"id"`
	Topic        string           `json:"topic"`
	CreationTime types.Time       `json:"creationTime"`
	Data         []RFQPublicTrade `json:"data"`
}

// WebsocketRFQQuoteUpdate holds changes to the quotes the caller sent or received
type WebsocketRFQQuoteUpdate struct {
	ID           string           `json:"id"`
	Topic        string           `json:"topic"`
	CreationTime types.Time       `json:"creationTime"`
	Data         []RFQQuoteUpdate `json:"data"`
}

// RFQQuoteUpdate is the state of a quote the caller sent or received
type RFQQuoteUpdate struct {
	RFQID string `json:"rfqId"`
	// RFQLinkID and QuoteLinkID are hidden from the other party
	RFQLinkID   string     `json:"rfqLinkId"`
	QuoteID     string     `json:"quoteId"`
	QuoteLinkID string     `json:"quoteLinkId"`
	ExpiresAt   types.Time `json:"expiresAt"`
	// DeskCode is the quoting party's, hidden when the quote is anonymous
	DeskCode string `json:"deskCode"`
	Status   string `json:"status"`
	// ExecutedQuoteSide is the side of the quote that was executed
	ExecutedQuoteSide string              `json:"execQuoteSide"`
	CreatedAt         types.Time          `json:"createdAt"`
	UpdatedAt         types.Time          `json:"updatedAt"`
	QuoteBuyList      []RFQQuoteUpdateLeg `json:"quoteBuyList"`
	QuoteSellList     []RFQQuoteUpdateLeg `json:"quoteSellList"`
}

// RFQQuoteUpdateLeg is the price of one leg of a quote in the quote stream
type RFQQuoteUpdateLeg struct {
	Category string       `json:"category"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
}

// WebsocketRFQTradeUpdate holds executions of RFQs the caller took part in; all legs of a block trade arrive together
type WebsocketRFQTradeUpdate struct {
	ID           string     `json:"id"`
	Topic        string     `json:"topic"`
	CreationTime types.Time `json:"creationTime"`
	Data         []RFQTrade `json:"data"`
}

// WebsocketRFQInquiryUpdate holds changes to the RFQs the caller sent or received
type WebsocketRFQInquiryUpdate struct {
	ID           string             `json:"id"`
	Topic        string             `json:"topic"`
	CreationTime types.Time         `json:"creationTime"`
	Data         []RFQInquiryUpdate `json:"data"`
}

// RFQInquiryUpdate is the state of an RFQ the caller sent or received
type RFQInquiryUpdate struct {
	RFQID string `json:"rfqId"`
	// RFQLinkID is hidden from the quoting parties
	RFQLinkID      string     `json:"rfqLinkId"`
	Counterparties []string   `json:"counterparties"`
	ExpiresAt      types.Time `json:"expiresAt"`
	StrategyType   string     `json:"strategyType"`
	Status         string     `json:"status"`
	// AcceptOtherQuoteStatus is whether quotes from parties other than liquidity providers are accepted; the stream
	// quotes it
	AcceptOtherQuoteStatus types.Boolean `json:"acceptOtherQuoteStatus"`
	// DeskCode is the inquiring party's, hidden when the RFQ is anonymous
	DeskCode  string     `json:"deskCode"`
	CreatedAt types.Time `json:"createdAt"`
	UpdatedAt types.Time `json:"updatedAt"`
	Legs      []RFQLeg   `json:"legs"`
}
