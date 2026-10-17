package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PreTradeDataResponse holds the top 10 levels of a pair's order book, as published for regulatory transparency
type PreTradeDataResponse struct {
	Symbol      string        `json:"symbol"`
	Description string        `json:"description"`
	BaseAsset   currency.Code `json:"base_asset"`
	// BaseNotation is UNIT when quantities are expressed in units, or NOML in nominal value
	BaseNotation string `json:"base_notation"`
	// BaseDTICode is the base asset's Digital Token Identifier
	BaseDTICode      string        `json:"base_dti_code"`
	BaseDTIShortName string        `json:"base_dti_short_name"`
	QuoteAsset       currency.Code `json:"quote_asset"`
	// QuoteNotation is MONE for prices expressed in monetary value. Kraken documents no other value, but sends UNIT on
	// pairs quoted in a crypto asset
	QuoteNotation     string `json:"quote_notation"`
	QuoteDTICode      string `json:"quote_dti_code"`
	QuoteDTIShortName string `json:"quote_dti_short_name"`
	// Venue is the Market Identifier Code of the trading platform
	Venue string `json:"venue"`
	// System is CLOB, a central limit order book
	System string               `json:"system"`
	Bids   []PreTradePriceLevel `json:"bids"`
	Asks   []PreTradePriceLevel `json:"asks"`
}

// PreTradePriceLevel is an aggregated order book price level
type PreTradePriceLevel struct {
	// Side is BUY for a bid or SELL for an offer
	Side       string       `json:"side"`
	Price      types.Number `json:"price"`
	Quantity   types.Number `json:"qty"`
	OrderCount uint64       `json:"count"`
	// SubmissionTime is when an order at this level was submitted
	SubmissionTime time.Time `json:"submission_ts"`
	// PublicationTime is when the level was last updated and published
	PublicationTime time.Time `json:"publication_ts"`
}

// PostTradeDataRequest holds the parameters of Post-Trade Data
type PostTradeDataRequest struct {
	// Pair filters the trades to a pair; the last trades of every pair are returned when it is empty
	Pair currency.Pair
	// From returns the trades after it, for polling with the previous response's LastTime
	From time.Time
	// To returns the trades at or before it
	To time.Time
	// Count is the maximum number of trades, up to 1000, and 1000 when it is 0
	Count uint64
}

// PostTradeDataResponse holds trades as published for regulatory transparency. Kraken documents them in ascending time
// order, but sends the newest first
type PostTradeDataResponse struct {
	// LastTime is the time of the latest trade returned, to poll with as PostTradeDataRequest's From
	LastTime time.Time       `json:"last_ts"`
	Count    uint64          `json:"count"`
	Trades   []PostTradeData `json:"trades"`
}

// PostTradeData is a trade as published for regulatory transparency
type PostTradeData struct {
	TradeID           string        `json:"trade_id"`
	Price             types.Number  `json:"price"`
	Quantity          types.Number  `json:"quantity"`
	Symbol            string        `json:"symbol"`
	Description       string        `json:"description"`
	BaseAsset         currency.Code `json:"base_asset"`
	BaseNotation      string        `json:"base_notation"`
	BaseDTICode       string        `json:"base_dti_code"`
	BaseDTIShortName  string        `json:"base_dti_short_name"`
	QuoteAsset        currency.Code `json:"quote_asset"`
	QuoteNotation     string        `json:"quote_notation"`
	QuoteDTICode      string        `json:"quote_dti_code"`
	QuoteDTIShortName string        `json:"quote_dti_short_name"`
	// TradeVenue is the Market Identifier Code of the platform the trade executed on
	TradeVenue string    `json:"trade_venue"`
	TradeTime  time.Time `json:"trade_ts"`
	// PublicationVenue is the Market Identifier Code of the platform the trade was published on
	PublicationVenue string    `json:"publication_venue"`
	PublicationTime  time.Time `json:"publication_ts"`
}
