package coinbase

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// derivativesEnvelope identifies a gateway websocket message: the response to a request, a heartbeat or a subscription
// notification
type derivativesEnvelope struct {
	ID     int64                     `json:"id"`
	Method string                    `json:"method"`
	Params derivativesEnvelopeParams `json:"params"`
}

// derivativesEnvelopeParams identifies a notification's channel or a heartbeat's type
type derivativesEnvelopeParams struct {
	Channel string `json:"channel"`
	// Type is test_request when a heartbeat must be answered with a public/test request
	Type string `json:"type"`
}

// derivativesNotification decodes a subscription notification's data into the value Data holds
type derivativesNotification struct {
	Params derivativesNotificationParams `json:"params"`
}

// derivativesNotificationParams holds a subscription notification's channel and data
type derivativesNotificationParams struct {
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

// DerivativesWebsocketOrderbook is a book channel notification: the whole book, with no depth limit, in a snapshot, and
// the levels that changed since the previous notification in a change
type DerivativesWebsocketOrderbook struct {
	// Type is snapshot or change
	Type           string     `json:"type"`
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	ChangeID       uint64     `json:"change_id"`
	// PreviousChangeID is the change ID of the notification a change follows; a snapshot leaves it out
	PreviousChangeID uint64                                `json:"prev_change_id"`
	Bids             []DerivativesWebsocketOrderbookChange `json:"bids"`
	Asks             []DerivativesWebsocketOrderbookChange `json:"asks"`
}

// DerivativesWebsocketOrderbookChange is a book level change, sent as an [action, price, amount] tuple
type DerivativesWebsocketOrderbookChange struct {
	// Action is new, change or delete; a deleted level's amount is zero
	Action string
	Price  float64
	Amount float64
}

// UnmarshalJSON decodes an [action, price, amount] tuple
func (d *DerivativesWebsocketOrderbookChange) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &d.Action, &d.Price, &d.Amount)
}

// DerivativesWebsocketGroupedOrderbook is a grouped book channel notification, which sends the book's grouped levels
// whole, up to the channel's depth
type DerivativesWebsocketGroupedOrderbook struct {
	Timestamp      types.Time                  `json:"timestamp"`
	InstrumentName string                      `json:"instrument_name"`
	ChangeID       uint64                      `json:"change_id"`
	Bids           []DerivativesOrderBookLevel `json:"bids"`
	Asks           []DerivativesOrderBookLevel `json:"asks"`
}

// DerivativesWebsocketIncrementalTicker is an incremental_ticker notification. The first for an instrument is a snapshot
// carrying the whole ticker, and each change after it carries only the fields that moved. A change clears a field by
// sending it as null, which would leave a plain float64 as it was, so every field a change can leave out is a pointer,
// nil where a notification leaves it out or sends null
type DerivativesWebsocketIncrementalTicker struct {
	// Type is snapshot or change
	Type                   string                                      `json:"type"`
	Timestamp              types.Time                                  `json:"timestamp"`
	InstrumentName         string                                      `json:"instrument_name"`
	Stats                  DerivativesWebsocketIncrementalTickerStats  `json:"stats"`
	Greeks                 DerivativesWebsocketIncrementalTickerGreeks `json:"greeks"`
	State                  *string                                     `json:"state"`
	UnderlyingIndex        *string                                     `json:"underlying_index"`
	BestBidPrice           *float64                                    `json:"best_bid_price"`
	BestBidAmount          *float64                                    `json:"best_bid_amount"`
	BestAskPrice           *float64                                    `json:"best_ask_price"`
	BestAskAmount          *float64                                    `json:"best_ask_amount"`
	LastPrice              *float64                                    `json:"last_price"`
	MarkPrice              *float64                                    `json:"mark_price"`
	IndexPrice             *float64                                    `json:"index_price"`
	MinimumPrice           *float64                                    `json:"min_price"`
	MaximumPrice           *float64                                    `json:"max_price"`
	EstimatedDeliveryPrice *float64                                    `json:"estimated_delivery_price"`
	DeliveryPrice          *float64                                    `json:"delivery_price"`
	SettlementPrice        *float64                                    `json:"settlement_price"`
	UnderlyingPrice        *float64                                    `json:"underlying_price"`
	OpenInterest           *float64                                    `json:"open_interest"`
	BidImpliedVolatility   *float64                                    `json:"bid_iv"`
	AskImpliedVolatility   *float64                                    `json:"ask_iv"`
	MarkImpliedVolatility  *float64                                    `json:"mark_iv"`
	InterestRate           *float64                                    `json:"interest_rate"`
	InterestValue          *float64                                    `json:"interest_value"`
	CurrentFunding         *float64                                    `json:"current_funding"`
	Funding8Hour           *float64                                    `json:"funding_8h"`
}

// DerivativesWebsocketIncrementalTickerStats holds an incremental_ticker notification's 24 hour aggregates, each nil
// where the notification leaves it out or sends null
type DerivativesWebsocketIncrementalTickerStats struct {
	High *float64 `json:"high"`
	Low  *float64 `json:"low"`
	// PriceChange is a percentage
	PriceChange    *float64 `json:"price_change"`
	Volume         *float64 `json:"volume"`
	VolumeUSD      *float64 `json:"volume_usd"`
	VolumeNotional *float64 `json:"volume_notional"`
}

// DerivativesWebsocketIncrementalTickerGreeks holds an incremental_ticker notification's option greeks, each nil where
// the notification leaves it out or sends null
type DerivativesWebsocketIncrementalTickerGreeks struct {
	Delta *float64 `json:"delta"`
	Gamma *float64 `json:"gamma"`
	Rho   *float64 `json:"rho"`
	Theta *float64 `json:"theta"`
	Vega  *float64 `json:"vega"`
}

// DerivativesWebsocketPerpetualInterest is a perpetual's current interest rate
type DerivativesWebsocketPerpetualInterest struct {
	Timestamp  types.Time `json:"timestamp"`
	IndexPrice float64    `json:"index_price"`
	Interest   float64    `json:"interest"`
}

// DerivativesWebsocketQuote is an instrument's best bid and ask; a price is zero when its side is empty
type DerivativesWebsocketQuote struct {
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	BestBidPrice   float64    `json:"best_bid_price"`
	BestBidAmount  float64    `json:"best_bid_amount"`
	BestAskPrice   float64    `json:"best_ask_price"`
	BestAskAmount  float64    `json:"best_ask_amount"`
}

// DerivativesWebsocketPriceIndex is an index's current value
type DerivativesWebsocketPriceIndex struct {
	Timestamp types.Time `json:"timestamp"`
	IndexName string     `json:"index_name"`
	Price     float64    `json:"price"`
}

// DerivativesWebsocketPriceRanking is one component exchange's price of an index
type DerivativesWebsocketPriceRanking struct {
	Identifier string     `json:"identifier"`
	Enabled    bool       `json:"enabled"`
	Timestamp  types.Time `json:"timestamp"`
	// OriginalPrice is the exchange's price, and Price the adjusted price the index uses
	OriginalPrice float64 `json:"original_price"`
	Price         float64 `json:"price"`
	// Weight is a percentage
	Weight float64 `json:"weight"`
}

// DerivativesWebsocketPriceStatistics is an index's 24 hour statistics
type DerivativesWebsocketPriceStatistics struct {
	IndexName    string  `json:"index_name"`
	Low24Hour    float64 `json:"low24h"`
	High24Hour   float64 `json:"high24h"`
	Change24Hour float64 `json:"change24h"`
	// FastMarket is set from a drastic index change within an hour until two hours after prices calm down
	FastMarket bool `json:"fast_market"`
}

// DerivativesWebsocketVolatilityIndex is a volatility index's current value
type DerivativesWebsocketVolatilityIndex struct {
	Timestamp  types.Time `json:"timestamp"`
	IndexName  string     `json:"index_name"`
	Volatility float64    `json:"volatility"`
}

// DerivativesWebsocketMarkPriceOption is an option's mark price
type DerivativesWebsocketMarkPriceOption struct {
	Timestamp         types.Time `json:"timestamp"`
	InstrumentName    string     `json:"instrument_name"`
	MarkPrice         float64    `json:"mark_price"`
	ImpliedVolatility float64    `json:"iv"`
}

// DerivativesWebsocketEstimatedExpirationPrice is an index's current price, or its estimated expiration price when
// IsEstimated is set
type DerivativesWebsocketEstimatedExpirationPrice struct {
	// Timestamp is sent although the docs leave it out
	Timestamp types.Time `json:"timestamp"`
	// Seconds is the time left until the nearest expiry
	Seconds     uint64  `json:"seconds"`
	Price       float64 `json:"price"`
	IsEstimated bool    `json:"is_estimated"`
	// LeftTicks and TotalTicks count the expiry period's time ticks, while IsEstimated is set
	LeftTicks  uint64 `json:"left_ticks"`
	TotalTicks uint64 `json:"total_ticks"`
}

// DerivativesWebsocketCandle is a chart.trades candle, which may be sent several times as its period progresses
type DerivativesWebsocketCandle struct {
	Tick   types.Time `json:"tick"`
	Open   float64    `json:"open"`
	High   float64    `json:"high"`
	Low    float64    `json:"low"`
	Close  float64    `json:"close"`
	Volume float64    `json:"volume"`
	Cost   float64    `json:"cost"`
}

// DerivativesWebsocketInstrumentState is an instrument's new state
type DerivativesWebsocketInstrumentState struct {
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	// State is open, settlement, delivered, inactive, locked, halted or archivized
	State string `json:"state"`
}

// DerivativesWebsocketPlatformState is a platform state notification: an index locked or unlocked, or a maintenance
// break starting
type DerivativesWebsocketPlatformState struct {
	PriceIndex  string `json:"price_index"`
	Locked      bool   `json:"locked"`
	Maintenance bool   `json:"maintenance"`
	// Reasons is undocumented and has been sent empty, so it is kept as sent
	Reasons json.RawMessage `json:"reasons"`
}

// DerivativesWebsocketPublicMethodsState is whether the gateway allows unauthenticated public requests
type DerivativesWebsocketPublicMethodsState struct {
	AllowUnauthenticatedPublicRequests bool `json:"allow_unauthenticated_public_requests"`
}

// DerivativesWebsocketUserLock is a currency's lock state for the account; Currency is ALL when every currency changed
type DerivativesWebsocketUserLock struct {
	Currency currency.Code `json:"currency"`
	Locked   bool          `json:"locked"`
}

// DerivativesWebsocketPositionLock is whether the account's positions are locked, which stops opening or growing them
type DerivativesWebsocketPositionLock struct {
	Locked bool `json:"locked"`
}

// DerivativesWebsocketUserLiquidation is a liquidation, auto-deleveraging or liquidity support program event of one of
// the account's isolated margin subaccounts
type DerivativesWebsocketUserLiquidation struct {
	// State is liquidation_started, liquidation_completed, close_out_liquidation_started,
	// close_out_liquidation_completed, lsp_transfer, adl_transfer or adl_transfer_received
	State    string        `json:"state"`
	UserID   uint64        `json:"user_id"`
	Isolated bool          `json:"isolated"`
	Currency currency.Code `json:"currency"`
	// The margins and equity are sent for the liquidation states
	MarginBalance     float64 `json:"margin_balance"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	InitialMargin     float64 `json:"initial_margin"`
	CloseOutMargin    float64 `json:"close_out_margin"`
	Equity            float64 `json:"equity"`
	// The transfer fields are sent for the transfer states; Direction is this account's resulting direction
	InstrumentName string     `json:"instrument_name"`
	Direction      string     `json:"direction"`
	Amount         float64    `json:"amount"`
	Price          float64    `json:"price"`
	Commission     float64    `json:"commission"`
	Timestamp      types.Time `json:"timestamp"`
}

// DerivativesWebsocketIsolatedPortfolio is one of the account's isolated margin subaccounts' balances and margin in a
// currency
type DerivativesWebsocketIsolatedPortfolio struct {
	Currency          currency.Code `json:"currency"`
	Equity            float64       `json:"equity"`
	MarginBalance     float64       `json:"margin_balance"`
	AvailableFunds    float64       `json:"available_funds"`
	InitialMargin     float64       `json:"initial_margin"`
	MaintenanceMargin float64       `json:"maintenance_margin"`
	// MainUserID is the owning main account's user ID, and UserID the subaccount's
	MainUserID uint64 `json:"main_uid"`
	UserID     uint64 `json:"user_id"`
}

// DerivativesWebsocketTrailingTriggerPrices are the account's active trailing orders' trigger prices in a currency, by
// order
type DerivativesWebsocketTrailingTriggerPrices map[string]float64

// DerivativesWebsocketAnnouncement is an announcement the gateway published or deleted
type DerivativesWebsocketAnnouncement struct {
	DerivativesAnnouncement
	// Action is new for a published announcement and deleted for a deleted one
	Action string `json:"action"`
	// Unread counts the account's earlier unread announcements, which only an authenticated connection is sent
	Unread uint64 `json:"unread"`
}

// DerivativesWebsocketInstrumentCreation is an instrument the gateway created, which it notifies once
type DerivativesWebsocketInstrumentCreation struct {
	DerivativesInstrument
	Timestamp types.Time `json:"timestamp"`
}

// UnmarshalJSON decodes the instrument and the notification's timestamp, which the instrument's own decoder, promoted
// to this type, would leave out
func (d *DerivativesWebsocketInstrumentCreation) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &d.DerivativesInstrument); err != nil {
		return err
	}
	var notification struct {
		Timestamp types.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &notification); err != nil {
		return err
	}
	d.Timestamp = notification.Timestamp
	return nil
}

// DerivativesWebsocketUserChanges is the account's order, trade and position changes in an instrument
type DerivativesWebsocketUserChanges struct {
	InstrumentName string                 `json:"instrument_name"`
	Orders         []DerivativesOrder     `json:"orders"`
	Trades         []DerivativesUserTrade `json:"trades"`
	Positions      []DerivativesPosition  `json:"positions"`
}
