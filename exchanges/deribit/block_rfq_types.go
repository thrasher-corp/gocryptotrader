package deribit

import (
	"bytes"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BlockRFQLeg contains one instrument's leg of a Block RFQ, a quote or a trade; the leg's size is its ratio times the
// amount
type BlockRFQLeg struct {
	InstrumentName string `json:"instrument_name"`
	// Direction is buy or sell; a quote's or an acceptance's leg must match the direction of the Block RFQ's leg
	Direction string `json:"direction"`
	Ratio     uint64 `json:"ratio"`
	// Price is a quote's or a trade's price for the leg, which a Block RFQ's own legs leave out; a zero price is not sent
	Price float64 `json:"price,omitempty"`
}

// BlockRFQHedge contains the hedge leg of a Block RFQ, a quote or a trade, of which there is at most one
type BlockRFQHedge struct {
	InstrumentName string  `json:"instrument_name"`
	Direction      string  `json:"direction"`
	Price          float64 `json:"price"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount float64 `json:"amount"`
}

// AcceptBlockRFQRequest accepts a Block RFQ's quotes as its taker with one crossing price
type AcceptBlockRFQRequest struct {
	BlockRFQID uint64 `json:"block_rfq_id"`
	// Legs must match the Block RFQ's legs
	Legs []BlockRFQLeg `json:"legs"`
	// Price is the worst price the taker accepts for the whole structure, which may be negative
	Price float64 `json:"price"`
	// Amount times a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
	// Direction is the taker's, buy or sell
	Direction string `json:"direction"`
	// TimeInForce is fill_or_kill, which fills at once or not at all, or good_til_cancelled, which leaves a trade
	// trigger active until it fills or is cancelled
	TimeInForce string         `json:"time_in_force"`
	Hedge       *BlockRFQHedge `json:"hedge,omitempty"`
}

// AcceptBlockRFQResponse contains the block trades an accepted Block RFQ executed, or the trade trigger left active
type AcceptBlockRFQResponse struct {
	TradeTrigger *BlockRFQTradeTrigger `json:"trade_trigger"`
	BlockTrades  []BlockTrade          `json:"block_trades"`
}

// BlockRFQTradeTrigger contains a taker's trade trigger, which accepts a Block RFQ's quotes once they cross its price;
// only the taker sees it
type BlockRFQTradeTrigger struct {
	// State is untriggered, triggered or cancelled
	State     string  `json:"state"`
	Price     float64 `json:"price"`
	Direction string  `json:"direction"`
	// CancelReason is set once the trigger is cancelled
	CancelReason string `json:"cancel_reason"`
}

// AddBlockRFQQuoteRequest quotes a Block RFQ as a maker
type AddBlockRFQQuoteRequest struct {
	BlockRFQID uint64 `json:"block_rfq_id"`
	// Amount times a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
	// Direction is the maker's, buy or sell
	Direction string `json:"direction"`
	// Legs carry each leg's price, unless Price quotes a future spread
	Legs  []BlockRFQLeg  `json:"legs"`
	Hedge *BlockRFQHedge `json:"hedge,omitempty"`
	// Label, of up to 64 characters, identifies the quote among the Block RFQ's quotes
	Label string `json:"label,omitempty"`
	// ExecutionInstruction is any_part_of, Deribit's default, which may fill partially, or all_or_none, which fills
	// completely or not at all and takes priority over any_part_of quotes at the same price
	ExecutionInstruction string `json:"execution_instruction,omitempty"`
	// Price is the aggregated price of a future spread quote; a zero price is not sent
	Price float64 `json:"price,omitempty"`
	// ExpiresAt is when the quote expires; a zero time is not sent
	ExpiresAt time.Time `json:"-"`
}

// addBlockRFQQuoteParams contains an AddBlockRFQQuoteRequest's params, with its expiry in Unix milliseconds
type addBlockRFQQuoteParams struct {
	*AddBlockRFQQuoteRequest
	ExpiresAt int64 `json:"expires_at,omitempty"`
}

// BlockRFQQuote contains a maker's quote on a Block RFQ
type BlockRFQQuote struct {
	BlockRFQQuoteID     uint64     `json:"block_rfq_quote_id"`
	BlockRFQID          uint64     `json:"block_rfq_id"`
	CreationTimestamp   types.Time `json:"creation_timestamp"`
	LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
	// QuoteState is the quote's state, such as open or cancelled, and QuoteStateReason why it was cancelled
	QuoteState       string `json:"quote_state"`
	QuoteStateReason string `json:"quote_state_reason"`
	// ExecutionInstruction is any_part_of or all_or_none
	ExecutionInstruction string  `json:"execution_instruction"`
	Price                float64 `json:"price"`
	Amount               float64 `json:"amount"`
	// Direction is the maker's
	Direction string `json:"direction"`
	// FilledAmount is in USD for perpetuals and futures, and in the base currency for options
	FilledAmount float64        `json:"filled_amount"`
	Legs         []BlockRFQLeg  `json:"legs"`
	Hedge        *BlockRFQHedge `json:"hedge"`
	// Replaced is set once the quote is edited
	Replaced bool   `json:"replaced"`
	Label    string `json:"label"`
	AppName  string `json:"app_name"`
}

// CancelAllBlockRFQQuotesRequest selects the account's Block RFQ quotes to cancel
type CancelAllBlockRFQQuotesRequest struct {
	// BlockRFQID cancels only one Block RFQ's quotes, and is left out to cancel the quotes of every Block RFQ
	BlockRFQID uint64 `json:"block_rfq_id,omitempty"`
	// Detailed asks for a report of the cancellations in place of their count
	Detailed bool `json:"detailed,omitempty"`
}

// CancelAllBlockRFQQuotesResponse contains the number of Block RFQ quotes cancelled, or a detailed request's reports
type CancelAllBlockRFQQuotesResponse struct {
	CancelledCount uint64
	// DetailedReports are kept as sent, as Deribit does not document their format
	DetailedReports []json.RawMessage
}

// UnmarshalJSON decodes the count of cancelled quotes, or a detailed request's list of reports
func (c *CancelAllBlockRFQQuotesResponse) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var reports []json.RawMessage
		if err := json.Unmarshal(trimmed, &reports); err != nil {
			return fmt.Errorf("%w: %w", errInvalidBlockRFQQuoteCancellation, err)
		}
		*c = CancelAllBlockRFQQuotesResponse{DetailedReports: reports}
		return nil
	}
	var count uint64
	if err := json.Unmarshal(data, &count); err != nil {
		return fmt.Errorf("%w: %w", errInvalidBlockRFQQuoteCancellation, err)
	}
	*c = CancelAllBlockRFQQuotesResponse{CancelledCount: count}
	return nil
}

// blockRFQIDParams contains the Block RFQ of a request
type blockRFQIDParams struct {
	BlockRFQID uint64 `json:"block_rfq_id"`
}

// BlockRFQ contains a Block RFQ as its taker or a maker sees it
type BlockRFQ struct {
	BlockRFQID          uint64     `json:"block_rfq_id"`
	CreationTimestamp   types.Time `json:"creation_timestamp"`
	ExpirationTimestamp types.Time `json:"expiration_timestamp"`
	// Role is the account's role, taker or maker
	Role string `json:"role"`
	// State is open, filled, cancelled or expired, although Deribit's example of a new Block RFQ shows created
	State string `json:"state"`
	// TakerRating buckets the ratio of the taker's Block RFQs over 3 months to those it filled as 1-2, 2-5, 5-20 or 20+,
	// and is empty while the taker has too few Block RFQs to rate
	TakerRating string `json:"taker_rating"`
	// Makers lists the targeted makers' aliases, which only the taker sees
	Makers []string `json:"makers"`
	// Amount times a leg's ratio is the leg's trade size
	Amount         float64              `json:"amount"`
	MinTradeAmount float64              `json:"min_trade_amount"`
	Asks           []BlockRFQPriceLevel `json:"asks"`
	Bids           []BlockRFQPriceLevel `json:"bids"`
	Legs           []BlockRFQLeg        `json:"legs"`
	Hedge          *BlockRFQHedge       `json:"hedge"`
	ComboID        string               `json:"combo_id"`
	Label          string               `json:"label"`
	// AppName is the application that created the Block RFQ for the taker, which only the taker sees
	AppName string `json:"app_name"`
	// MarkPrice and Trades are only reported for a filled Block RFQ
	MarkPrice float64 `json:"mark_price"`
	// Disclosed is set when the taker's and makers' aliases are visible to each other, Taker being the taker's alias
	Disclosed bool   `json:"disclosed"`
	Taker     string `json:"taker"`
	// IndexPrices maps price index names to their prices
	IndexPrices map[string]float64 `json:"index_prices"`
	// IncludedInTakerRating reports whether a closed Block RFQ counts towards the taker's rating, to its taker
	IncludedInTakerRating bool                  `json:"included_in_taker_rating"`
	Trades                []BlockRFQFill        `json:"trades"`
	TradeTrigger          *BlockRFQTradeTrigger `json:"trade_trigger"`
	// TradeAllocations split the taker's amount between accounts, and only the taker sees them
	TradeAllocations []TradeAllocation `json:"trade_allocations"`
}

// BlockRFQPriceLevel contains the quotes at one price on one side of a Block RFQ
type BlockRFQPriceLevel struct {
	// Makers lists the quoting makers' aliases
	Makers              []string   `json:"makers"`
	Price               float64    `json:"price"`
	Amount              float64    `json:"amount"`
	LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
	// ExecutionInstruction is any_part_of or all_or_none
	ExecutionInstruction string `json:"execution_instruction"`
	// ExpiresAt is the earliest expiry of the level's quotes
	ExpiresAt types.Time `json:"expires_at"`
}

// BlockRFQFill contains a fill of a Block RFQ
type BlockRFQFill struct {
	Direction string  `json:"direction"`
	Price     float64 `json:"price"`
	// Amount and HedgeAmount are in the base currency for options, linear futures, linear perpetuals and spot, and in
	// USD for inverse perpetuals and futures
	Amount      float64 `json:"amount"`
	HedgeAmount float64 `json:"hedge_amount"`
	// Maker is the alias of the maker filled, which a maker only sees on its own fills
	Maker string `json:"maker"`
}

// CancelBlockRFQQuoteRequest identifies a Block RFQ quote to cancel by its ID, or by its Block RFQ and label
type CancelBlockRFQQuoteRequest struct {
	BlockRFQQuoteID uint64 `json:"block_rfq_quote_id,omitempty"`
	BlockRFQID      uint64 `json:"block_rfq_id,omitempty"`
	Label           string `json:"label,omitempty"`
}

// CreateBlockRFQRequest creates a Block RFQ as its taker
type CreateBlockRFQRequest struct {
	Legs []CreateBlockRFQLeg `json:"legs"`
	// TradeAllocations split the amount between the taker's accounts, itself included, or a broker's clients
	TradeAllocations []BlockRFQTradeAllocation `json:"trade_allocations,omitempty"`
	Hedge            *BlockRFQHedge            `json:"hedge,omitempty"`
	// Label may be up to 64 characters
	Label string `json:"label,omitempty"`
	// Makers lists the aliases of the makers to notify; every available maker is notified when it is empty
	Makers []string `json:"makers,omitempty"`
	// Disclosed shows the taker's and makers' aliases to each other, which Deribit defaults to when it is nil; it can
	// only be false when at least 5 makers are targeted
	Disclosed *bool `json:"disclosed,omitempty"`
}

// CreateBlockRFQLeg contains one instrument's leg of a Block RFQ to create
type CreateBlockRFQLeg struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount    float64 `json:"amount"`
	Direction string  `json:"direction"`
}

// BlockRFQTradeAllocation allocates part of a Block RFQ's amount to a user, by UserID, or to a broker client, by
// ClientInfo
type BlockRFQTradeAllocation struct {
	UserID     uint64                    `json:"user_id,omitempty"`
	ClientInfo *BlockRFQAllocationClient `json:"client_info,omitempty"`
	Amount     float64                   `json:"amount"`
}

// BlockRFQAllocationClient identifies the broker client user a Block RFQ allocation goes to
type BlockRFQAllocationClient struct {
	ClientID     uint64 `json:"client_id"`
	ClientLinkID uint64 `json:"client_link_id"`
}

// EditBlockRFQQuoteRequest replaces a Block RFQ quote's amount, legs and prices; the quote is identified by its ID, or
// by its Block RFQ and label
type EditBlockRFQQuoteRequest struct {
	BlockRFQQuoteID uint64 `json:"block_rfq_quote_id,omitempty"`
	BlockRFQID      uint64 `json:"block_rfq_id,omitempty"`
	Label           string `json:"label,omitempty"`
	// Amount times a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
	// Legs carry each leg's price, unless Price quotes a future spread
	Legs  []BlockRFQLeg  `json:"legs"`
	Hedge *BlockRFQHedge `json:"hedge,omitempty"`
	// Price is the aggregated price of a future spread quote; a zero price is not sent
	Price float64 `json:"price,omitempty"`
}

// GetBlockRFQQuotesRequest selects the account's open Block RFQ quotes; every filter may be left out
type GetBlockRFQQuotesRequest struct {
	BlockRFQID      uint64 `json:"block_rfq_id,omitempty"`
	Label           string `json:"label,omitempty"`
	BlockRFQQuoteID uint64 `json:"block_rfq_quote_id,omitempty"`
}

// BlockRFQUserInfoResponse contains the Block RFQ identities and ratings of the account group and its users
type BlockRFQUserInfoResponse struct {
	// Parent is the identity of the account group, the main account and its subaccounts
	Parent BlockRFQParentIdentity `json:"parent"`
	Users  []BlockRFQUserIdentity `json:"users"`
}

// BlockRFQParentIdentity contains an account group's Block RFQ alias and whether it may quote as a maker
type BlockRFQParentIdentity struct {
	Identity string `json:"identity"`
	IsMaker  bool   `json:"is_maker"`
}

// BlockRFQUserIdentity contains an account's Block RFQ alias, its taker rating and whether it may quote as a maker
type BlockRFQUserIdentity struct {
	UserID      uint64  `json:"user_id"`
	TakerRating float64 `json:"taker_rating"`
	Identity    string  `json:"identity"`
	IsMaker     bool    `json:"is_maker"`
}

// GetBlockRFQsRequest selects the Block RFQs the account created or was asked to quote, newest first; every filter may
// be left out
type GetBlockRFQsRequest struct {
	// Count is the number of Block RFQs to return, up to 1000
	Count uint64 `json:"count,omitempty"`
	// State is open, filled, traded, cancelled, expired or closed
	State string `json:"state,omitempty"`
	// Role is taker, maker or any
	Role string `json:"role,omitempty"`
	// Continuation returns the Block RFQs older than the Block RFQ ID it holds
	Continuation uint64 `json:"continuation,omitempty"`
	BlockRFQID   uint64 `json:"block_rfq_id,omitempty"`
	// Currency selects one currency's Block RFQs; currency.NewCode("any") selects every currency
	Currency currency.Code `json:"currency,omitzero"`
}

// BlockRFQsResponse contains a page of Block RFQs
type BlockRFQsResponse struct {
	BlockRFQs []BlockRFQ `json:"block_rfqs"`
	// Continuation is the Continuation of the next page, and zero on the last; Deribit documents it as a string token
	// but takes it back as a Block RFQ ID, so a number or a numeric string is accepted
	Continuation types.Number `json:"continuation"`
}

// GetBlockRFQTradesRequest selects recent Block RFQ trades, newest first
type GetBlockRFQTradesRequest struct {
	// Currency is required; currency.NewCode("any") selects every currency
	Currency currency.Code `json:"currency"`
	// Continuation is the token of the next page, which the previous page returned
	Continuation string `json:"continuation,omitempty"`
	// Count is the number of Block RFQ trades to return, 20 by default; Deribit accepts 10 to 50, although it documents 1
	// to 1000
	Count uint64 `json:"count,omitempty"`
}

// BlockRFQTradesResponse contains a page of Block RFQ trades
type BlockRFQTradesResponse struct {
	BlockRFQs []BlockRFQTrade `json:"block_rfqs"`
	// Continuation is the token of the next page, a timestamp and Block RFQ ID, and is empty on the last
	Continuation string `json:"continuation"`
}

// BlockRFQTrade contains a filled Block RFQ, as public market data reports it
type BlockRFQTrade struct {
	// ID is the Block RFQ's ID
	ID        uint64     `json:"id"`
	Timestamp types.Time `json:"timestamp"`
	// Direction is the taker's
	Direction string `json:"direction"`
	// Amount times a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
	// MarkPrice is the structure's mark price at the trade, which may be negative
	MarkPrice float64        `json:"mark_price"`
	Legs      []BlockRFQLeg  `json:"legs"`
	ComboID   string         `json:"combo_id"`
	Hedge     *BlockRFQHedge `json:"hedge"`
	// IndexPrices maps price index names to their prices at the trade
	IndexPrices map[string]float64 `json:"index_prices"`
	Trades      []BlockRFQFill     `json:"trades"`
}
