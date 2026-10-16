package deribit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsEnvelope is the envelope of every websocket message: a response carries its request's ID, and a subscription
// notification or heartbeat its method
type wsEnvelope struct {
	ID     int64            `json:"id"`
	Method string           `json:"method"`
	Params wsEnvelopeParams `json:"params"`
}

// wsEnvelopeParams identifies a notification's channel or a heartbeat's type
type wsEnvelopeParams struct {
	Channel string `json:"channel"`
	// Type is test_request when a heartbeat must be answered with a public/test request
	Type string `json:"type"`
}

// wsNotification decodes a subscription notification's data into the value Data holds
type wsNotification struct {
	Params wsNotificationParams `json:"params"`
}

// wsNotificationParams contains a subscription notification's channel and data
type wsNotificationParams struct {
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

// WebsocketOrderbook contains a book channel notification: the whole book, with no depth limit, in a snapshot, and the
// levels that changed since the previous notification in a change
type WebsocketOrderbook struct {
	// Type is snapshot or change
	Type           string     `json:"type"`
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	ChangeID       uint64     `json:"change_id"`
	// PreviousChangeID is the change ID of the notification a change follows; a snapshot leaves it out
	PreviousChangeID uint64                     `json:"prev_change_id"`
	Bids             []WebsocketOrderbookChange `json:"bids"`
	Asks             []WebsocketOrderbookChange `json:"asks"`
}

// WebsocketOrderbookChange is a book level change, sent as an [action, price, amount] tuple
type WebsocketOrderbookChange struct {
	// Action is new, change or delete; a deleted level's amount is zero
	Action string
	Price  float64
	Amount float64
}

// UnmarshalJSON decodes an [action, price, amount] tuple
func (w *WebsocketOrderbookChange) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &w.Action, &w.Price, &w.Amount)
}

// WebsocketGroupedOrderbook contains a grouped book channel notification, which sends the book's grouped levels whole,
// up to the channel's depth
type WebsocketGroupedOrderbook struct {
	Timestamp      types.Time       `json:"timestamp"`
	InstrumentName string           `json:"instrument_name"`
	ChangeID       uint64           `json:"change_id"`
	Bids           []OrderBookLevel `json:"bids"`
	Asks           []OrderBookLevel `json:"asks"`
}

// WebsocketIncrementalTicker holds an incremental_ticker message. The first message for an instrument is a
// snapshot carrying the whole ticker, and each change after it carries only the fields that moved.
// A change clears the day's range by sending it as null, which would leave a plain float64 as it
// was, so every field a change can leave out is a pointer, nil where a message leaves it out or
// sends null
type WebsocketIncrementalTicker struct {
	Type                   string                           `json:"type"`
	Timestamp              types.Time                       `json:"timestamp"`
	InstrumentName         string                           `json:"instrument_name"`
	Stats                  WebsocketIncrementalTickerStats  `json:"stats"`
	Greeks                 WebsocketIncrementalTickerGreeks `json:"greeks"`
	State                  *string                          `json:"state"`
	ComboState             *string                          `json:"combo_state"`
	UnderlyingIndex        *string                          `json:"underlying_index"`
	BestBidPrice           *float64                         `json:"best_bid_price"`
	BestBidAmount          *float64                         `json:"best_bid_amount"`
	BestAskPrice           *float64                         `json:"best_ask_price"`
	BestAskAmount          *float64                         `json:"best_ask_amount"`
	LastPrice              *float64                         `json:"last_price"`
	MarkPrice              *float64                         `json:"mark_price"`
	IndexPrice             *float64                         `json:"index_price"`
	MinPrice               *float64                         `json:"min_price"`
	MaxPrice               *float64                         `json:"max_price"`
	EstimatedDeliveryPrice *float64                         `json:"estimated_delivery_price"`
	DeliveryPrice          *float64                         `json:"delivery_price"`
	SettlementPrice        *float64                         `json:"settlement_price"`
	UnderlyingPrice        *float64                         `json:"underlying_price"`
	OpenInterest           *float64                         `json:"open_interest"`
	// ImpliedBid and ImpliedAsk are served only for combos, priced off the legs
	ImpliedBid            *float64 `json:"implied_bid"`
	ImpliedAsk            *float64 `json:"implied_ask"`
	BidImpliedVolatility  *float64 `json:"bid_iv"`
	AskImpliedVolatility  *float64 `json:"ask_iv"`
	MarkImpliedVolatility *float64 `json:"mark_iv"`
	InterestRate          *float64 `json:"interest_rate"`
	InterestValue         *float64 `json:"interest_value"`
	CurrentFunding        *float64 `json:"current_funding"`
	Funding8Hour          *float64 `json:"funding_8h"`
}

// WebsocketIncrementalTickerStats holds the 24 hour aggregates an incremental_ticker message carries, each
// nil where the message leaves it out or sends null
type WebsocketIncrementalTickerStats struct {
	High           *float64 `json:"high"`
	Low            *float64 `json:"low"`
	PriceChange    *float64 `json:"price_change"`
	Volume         *float64 `json:"volume"`
	VolumeUSD      *float64 `json:"volume_usd"`
	VolumeNotional *float64 `json:"volume_notional"`
}

// WebsocketIncrementalTickerGreeks holds the option greeks an incremental_ticker message carries, each nil
// where the message leaves it out or sends null
type WebsocketIncrementalTickerGreeks struct {
	Delta *float64 `json:"delta"`
	Gamma *float64 `json:"gamma"`
	Rho   *float64 `json:"rho"`
	Theta *float64 `json:"theta"`
	Vega  *float64 `json:"vega"`
}

// WebsocketPerpetualInterest contains a perpetual's current interest rate
type WebsocketPerpetualInterest struct {
	Timestamp  types.Time `json:"timestamp"`
	IndexPrice float64    `json:"index_price"`
	Interest   float64    `json:"interest"`
}

// WebsocketQuote contains an instrument's best bid and ask
type WebsocketQuote struct {
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	BestBidPrice   float64    `json:"best_bid_price"`
	BestBidAmount  float64    `json:"best_bid_amount"`
	BestAskPrice   float64    `json:"best_ask_price"`
	BestAskAmount  float64    `json:"best_ask_amount"`
}

// WebsocketPriceIndex contains an index's current value
type WebsocketPriceIndex struct {
	Timestamp types.Time `json:"timestamp"`
	IndexName string     `json:"index_name"`
	Price     float64    `json:"price"`
}

// WebsocketPriceRanking contains one component exchange's price of an index
type WebsocketPriceRanking struct {
	Identifier string     `json:"identifier"`
	Enabled    bool       `json:"enabled"`
	Timestamp  types.Time `json:"timestamp"`
	// OriginalPrice is the exchange's price, and Price the adjusted price the index uses
	OriginalPrice float64 `json:"original_price"`
	Price         float64 `json:"price"`
	// Weight is a percentage
	Weight float64 `json:"weight"`
}

// WebsocketPriceStatistics contains an index's 24 hour statistics
type WebsocketPriceStatistics struct {
	IndexName    string  `json:"index_name"`
	Low24Hour    float64 `json:"low24h"`
	High24Hour   float64 `json:"high24h"`
	Change24Hour float64 `json:"change24h"`
	// FastMarket is set from a drastic index change within an hour until two hours after prices calm down
	FastMarket bool `json:"fast_market"`
}

// WebsocketVolatilityIndex contains a volatility index's current value
type WebsocketVolatilityIndex struct {
	Timestamp  types.Time `json:"timestamp"`
	IndexName  string     `json:"index_name"`
	Volatility float64    `json:"volatility"`
}

// WebsocketMarkPriceOption contains an option's mark price
type WebsocketMarkPriceOption struct {
	Timestamp         types.Time `json:"timestamp"`
	InstrumentName    string     `json:"instrument_name"`
	MarkPrice         float64    `json:"mark_price"`
	ImpliedVolatility float64    `json:"iv"`
}

// WebsocketEstimatedExpirationPrice contains an index's current price, or its estimated expiration price when
// IsEstimated is set
type WebsocketEstimatedExpirationPrice struct {
	Timestamp types.Time `json:"timestamp"`
	// Seconds is the time left until the nearest expiry
	Seconds     uint64  `json:"seconds"`
	Price       float64 `json:"price"`
	IsEstimated bool    `json:"is_estimated"`
	// LeftTicks and TotalTicks count the expiry period's time ticks, while IsEstimated is set
	LeftTicks  uint64 `json:"left_ticks"`
	TotalTicks uint64 `json:"total_ticks"`
}

// WebsocketCandle contains a chart.trades candle, which may be sent several times as its period progresses
type WebsocketCandle struct {
	Tick   types.Time `json:"tick"`
	Open   float64    `json:"open"`
	High   float64    `json:"high"`
	Low    float64    `json:"low"`
	Close  float64    `json:"close"`
	Volume float64    `json:"volume"`
	Cost   float64    `json:"cost"`
}

// WebsocketInstrumentState contains an instrument's new state
type WebsocketInstrumentState struct {
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	// State is open, settlement, delivered, inactive, locked, halted or archivized
	State string `json:"state"`
}

// WebsocketPlatformState contains a platform state notification: an index locked or unlocked, or a maintenance break
// starting
type WebsocketPlatformState struct {
	PriceIndex  string `json:"price_index"`
	Locked      bool   `json:"locked"`
	Maintenance bool   `json:"maintenance"`
	// Reasons is undocumented and has been sent empty, so it is kept as sent
	Reasons json.RawMessage `json:"reasons"`
}

// WebsocketPublicMethodsState contains whether Deribit allows unauthenticated public requests
type WebsocketPublicMethodsState struct {
	AllowUnauthenticatedPublicRequests bool `json:"allow_unauthenticated_public_requests"`
}

// WebsocketUserLock contains a currency's lock state for the account
type WebsocketUserLock struct {
	Currency currency.Code `json:"currency"`
	Locked   bool          `json:"locked"`
}

// WebsocketUserLiquidation contains a liquidation, auto-deleveraging or liquidity support program event of the account
// or, from user.isolated.liquidation, of one of its isolated margin subaccounts
type WebsocketUserLiquidation struct {
	// State is liquidation_started, liquidation_completed, close_out_liquidation_started,
	// close_out_liquidation_completed, lsp_transfer, adl_transfer or adl_transfer_received
	State             string        `json:"state"`
	UserID            uint64        `json:"user_id"`
	Isolated          bool          `json:"isolated"`
	Currency          currency.Code `json:"currency"`
	MarginBalance     float64       `json:"margin_balance"`
	MaintenanceMargin float64       `json:"maintenance_margin"`
	InitialMargin     float64       `json:"initial_margin"`
	CloseOutMargin    float64       `json:"close_out_margin"`
	Equity            float64       `json:"equity"`
	InstrumentName    string        `json:"instrument_name"`
	Direction         string        `json:"direction"`
	Amount            float64       `json:"amount"`
	Price             float64       `json:"price"`
	Commission        float64       `json:"commission"`
	Timestamp         types.Time    `json:"timestamp"`
}

// WebsocketMMPTrigger contains a market maker protection trigger
type WebsocketMMPTrigger struct {
	// FrozenUntil is when the account is unfrozen; zero means until it is reset
	FrozenUntil types.Time `json:"frozen_until"`
	// IndexName is the index triggered, or all for a Block RFQ trigger by trade count
	IndexName string `json:"index_name"`
	// MMPGroup is the mass quote group triggered
	MMPGroup string `json:"mmp_group"`
	BlockRFQ bool   `json:"block_rfq"`
}

// WebsocketAnnouncement contains an announcement Deribit published or deleted
type WebsocketAnnouncement struct {
	Announcement
	// Action is new for a published announcement and deleted for a deleted one
	Action string `json:"action"`
	// Unread counts the account's earlier unread announcements, which only an authenticated connection is sent
	Unread uint64 `json:"unread"`
}

// WebsocketInstrumentCreation contains an instrument Deribit created, which it notifies once
type WebsocketInstrumentCreation struct {
	Instrument
	Timestamp types.Time `json:"timestamp"`
}

// UnmarshalJSON decodes the instrument and the notification's timestamp, which the instrument's own decoder, promoted
// to this type, would leave out
func (w *WebsocketInstrumentCreation) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &w.Instrument); err != nil {
		return err
	}
	var notification struct {
		Timestamp types.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &notification); err != nil {
		return err
	}
	w.Timestamp = notification.Timestamp
	return nil
}

// WebsocketUserChanges contains the account's order, trade and position changes in an instrument
type WebsocketUserChanges struct {
	InstrumentName string      `json:"instrument_name"`
	Orders         []Order     `json:"orders"`
	Trades         []UserTrade `json:"trades"`
	// Positions is named as Deribit's examples send it, although its schema documents position
	Positions []Position `json:"positions"`
}

// WebsocketLSPEvent contains a Liquidity Support Program event of the participant subaccount the connection is
// authenticated as
type WebsocketLSPEvent struct {
	// Type is transfer for an assignment attempt, enabled or disabled when participation changes, or
	// configuration_changed whenever the effective configuration is written
	Type           string        `json:"type"`
	InstrumentName string        `json:"instrument_name"`
	Currency       currency.Code `json:"currency"`
	// Direction, Amount, Price and Success are a transfer's, as is ErrorReason when it fails: platform_locked,
	// lock_count_exceeded, timeout, temporarily_unavailable, not_enough_funds or internal_error
	Direction   string  `json:"direction"`
	Amount      float64 `json:"amount"`
	Price       float64 `json:"price"`
	Success     bool    `json:"success"`
	ErrorReason string  `json:"error_reason"`
	// StarbaseMatchID is a successful transfer's on Starbase
	StarbaseMatchID uint64 `json:"starbase_match_id"`
	// Automatic is a disable's, set when repeated transfer failures disabled participation, as FailureCount did within
	// WindowMinutes
	Automatic     bool   `json:"automatic"`
	FailureCount  uint64 `json:"failure_count"`
	WindowMinutes uint64 `json:"window_minutes"`
	// Configuration is the resolved effective configuration, sent for configuration_changed
	Configuration *LSPParticipantConfig `json:"configuration"`
	Timestamp     types.Time            `json:"timestamp"`
}

// WebsocketBlockRFQQuotes contains a maker's Block RFQ quotes; Deribit documents a single quote and sends a list in its
// example, so either is accepted
type WebsocketBlockRFQQuotes []BlockRFQQuote

// UnmarshalJSON decodes a list of quotes or a single quote
func (w *WebsocketBlockRFQQuotes) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, (*[]BlockRFQQuote)(w))
	}
	var quote *BlockRFQQuote
	if err := json.Unmarshal(data, &quote); err != nil {
		return err
	}
	if quote != nil {
		*w = WebsocketBlockRFQQuotes{*quote}
	}
	return nil
}
