package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errSubscriptionResponseMissing = errors.New("subscription response missing")
	errUnexpectedSubscriptionReply = errors.New("unexpected subscription response")
	errSystemNotOnline             = errors.New("trading engine not online")
	errChecksumMismatch            = errors.New("order book checksum mismatch")
	errWebsocketTokenEmpty         = errors.New("websocket token empty")
	errLevel3OutOfSync             = errors.New("level 3 order book out of sync")
)

// wsRequest is a websocket v2 request
type wsRequest struct {
	Method    string `json:"method"`
	Params    any    `json:"params,omitempty"`
	RequestID int64  `json:"req_id"`
}

// wsSubscriptionParams holds the parameters of a subscribe or unsubscribe request
type wsSubscriptionParams struct {
	Channel  string   `json:"channel"`
	Symbols  []string `json:"symbol,omitempty"`
	Depth    uint64   `json:"depth,omitempty"`
	Interval uint64   `json:"interval,omitempty"`
	// SnapOrders and SnapTrades select the executions snapshot; Kraken includes open orders and leaves out fills
	// when they are not sent
	SnapOrders *bool  `json:"snap_orders,omitempty"`
	SnapTrades *bool  `json:"snap_trades,omitempty"`
	Token      string `json:"token,omitempty"`
}

// wsResponse is the reply to a websocket v2 request. Symbol is set on the reply rejecting one of a request's symbols
type wsResponse struct {
	Method    string          `json:"method"`
	RequestID int64           `json:"req_id"`
	Result    json.RawMessage `json:"result"`
	Success   bool            `json:"success"`
	Error     string          `json:"error"`
	Symbol    string          `json:"symbol"`
	TimeIn    time.Time       `json:"time_in"`
	TimeOut   time.Time       `json:"time_out"`
}

// wsSubscriptionResult is the result of a successful subscribe or unsubscribe request, one for each symbol
type wsSubscriptionResult struct {
	Channel  string   `json:"channel"`
	Symbol   string   `json:"symbol"`
	Warnings []string `json:"warnings"`
}

// wsChannelMessage is a message pushed on a websocket v2 channel, whose data depends on the channel
type wsChannelMessage struct {
	Channel  string          `json:"channel"`
	Type     string          `json:"type"`
	Data     json.RawMessage `json:"data"`
	Sequence uint64          `json:"sequence"`
	// Timestamp is when an ohlc message was sent
	Timestamp time.Time `json:"timestamp"`
}

// WsStatus is the trading engine status the status channel pushes on connection and on every change
type WsStatus struct {
	System              string             `json:"system"`
	APIVersion          string             `json:"api_version"`
	ConnectionID        uint64             `json:"connection_id"`
	Version             string             `json:"version"`
	UpcomingMaintenance []MaintenanceEvent `json:"upcoming_maintenance"`
	Emergency           []EmergencyEvent   `json:"emergency"`
}

// WsTicker is a pair's top of book and 24 hour statistics, pushed on every trade or best price change
type WsTicker struct {
	Symbol                     string  `json:"symbol"`
	Bid                        float64 `json:"bid"`
	BidQuantity                float64 `json:"bid_qty"`
	Ask                        float64 `json:"ask"`
	AskQuantity                float64 `json:"ask_qty"`
	Last                       float64 `json:"last"`
	Volume                     float64 `json:"volume"`
	VolumeWeightedAveragePrice float64 `json:"vwap"`
	Low                        float64 `json:"low"`
	High                       float64 `json:"high"`
	Change                     float64 `json:"change"`
	ChangePercentage           float64 `json:"change_pct"`
	// Trades is the number of trades over the last 24 hours, which Kraken sends without documenting
	Trades    uint64    `json:"trades"`
	Timestamp time.Time `json:"timestamp"`
}

// WsBook is a pair's level 2 order book snapshot or update
type WsBook struct {
	Symbol string        `json:"symbol"`
	Bids   []WsBookLevel `json:"bids"`
	Asks   []WsBookLevel `json:"asks"`
	// Checksum is the CRC32 of the top 10 levels of each side after the update is applied
	Checksum  uint32    `json:"checksum"`
	Timestamp time.Time `json:"timestamp"`
}

// WsBookLevel is an order book price level. Its numbers keep the digits Kraken sent, which the checksum covers; a
// zero quantity removes the level
type WsBookLevel struct {
	Price    types.PreciseNumber `json:"price"`
	Quantity types.PreciseNumber `json:"qty"`
}

// WsLevel3Book is a pair's level 3 order book snapshot, holding the orders of each side in the order they queue, or an
// update, holding the events that change them
type WsLevel3Book struct {
	Symbol string          `json:"symbol"`
	Bids   []WsLevel3Order `json:"bids"`
	Asks   []WsLevel3Order `json:"asks"`
	// Checksum is the CRC32 of the orders of the top 10 price levels of each side after the update is applied
	Checksum  uint32    `json:"checksum"`
	Timestamp time.Time `json:"timestamp"`
}

// WsLevel3Order is an order resting in a level 3 order book, or an update's event for one. Its numbers keep the digits
// Kraken sent, which the checksum covers
type WsLevel3Order struct {
	// Event is add, modify or delete in an update; a modify event changes the quantity left after a fill
	Event      string              `json:"event"`
	OrderID    string              `json:"order_id"`
	LimitPrice types.PreciseNumber `json:"limit_price"`
	Quantity   types.PreciseNumber `json:"order_qty"`
	// Timestamp is when the order was inserted or amended
	Timestamp time.Time `json:"timestamp"`
}

// WsCandle is a candle the ohlc channel pushes on every trade
type WsCandle struct {
	Symbol                     string    `json:"symbol"`
	Open                       float64   `json:"open"`
	High                       float64   `json:"high"`
	Low                        float64   `json:"low"`
	Close                      float64   `json:"close"`
	VolumeWeightedAveragePrice float64   `json:"vwap"`
	Trades                     uint64    `json:"trades"`
	Volume                     float64   `json:"volume"`
	IntervalBegin              time.Time `json:"interval_begin"`
	// IntervalMinutes is the candle's interval in minutes
	IntervalMinutes uint64 `json:"interval"`
}

// WsTrade is a public trade
type WsTrade struct {
	Symbol string `json:"symbol"`
	// Side is the taker's side
	Side     string  `json:"side"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"qty"`
	// OrderType is the taker's order type
	OrderType string    `json:"ord_type"`
	TradeID   uint64    `json:"trade_id"`
	Timestamp time.Time `json:"timestamp"`
}

// WsInstruments holds the reference data of every active asset and pair
type WsInstruments struct {
	Assets []WsInstrumentAsset `json:"assets"`
	Pairs  []WsInstrumentPair  `json:"pairs"`
}

// WsInstrumentAsset is an asset's reference data
type WsInstrumentAsset struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	Precision        uint64  `json:"precision"`
	PrecisionDisplay uint64  `json:"precision_display"`
	Borrowable       bool    `json:"borrowable"`
	CollateralValue  float64 `json:"collateral_value"`
	MarginRate       float64 `json:"margin_rate"`
	// Multiplier is the fixed conversion rate of a tokenised asset
	Multiplier float64 `json:"multiplier"`
	Class      string  `json:"class"`
	// UnderlyingSymbol and MaximumCollateralUSDValue describe a tokenised asset, which Kraken sends without
	// documenting
	UnderlyingSymbol          string  `json:"underlying_symbol"`
	MaximumCollateralUSDValue float64 `json:"max_collateral_usd_value"`
}

// WsInstrumentPair is a pair's reference data and trading rules
type WsInstrumentPair struct {
	Symbol            string  `json:"symbol"`
	Base              string  `json:"base"`
	Quote             string  `json:"quote"`
	Status            string  `json:"status"`
	QuantityMinimum   float64 `json:"qty_min"`
	QuantityIncrement float64 `json:"qty_increment"`
	QuantityPrecision uint64  `json:"qty_precision"`
	PriceIncrement    float64 `json:"price_increment"`
	PricePrecision    uint64  `json:"price_precision"`
	CostMinimum       float64 `json:"cost_min"`
	CostPrecision     uint64  `json:"cost_precision"`
	// WebsocketDisplayPricePrecision is the recommended precision for displaying websocket prices
	WebsocketDisplayPricePrecision uint64 `json:"ws_display_price_precision"`
	HasIndex                       bool   `json:"has_index"`
	// MarginTradable is set when the pair can be traded on margin
	MarginTradable bool `json:"marginable"` //nolint:misspell // Kraken's field name
	// MarginInitial is the initial margin requirement, which Kraken documents in percent but sends as a fraction, such
	// as 0.1
	MarginInitial      float64 `json:"margin_initial"`
	PositionLimitLong  uint64  `json:"position_limit_long"`
	PositionLimitShort uint64  `json:"position_limit_short"`
}

// WsExecution is an execution report: an order's status change or, on a trade event, a fill. Which fields are set
// depends on ExecutionType
type WsExecution struct {
	OrderID            string  `json:"order_id"`
	OrderUserReference int32   `json:"order_userref"`
	ClientOrderID      string  `json:"cl_ord_id"`
	Symbol             string  `json:"symbol"`
	Side               string  `json:"side"`
	OrderType          string  `json:"order_type"`
	OrderQuantity      float64 `json:"order_qty"`
	// CashOrderQuantity is the order's quantity in the quote currency, when it was placed that way
	CashOrderQuantity float64 `json:"cash_order_qty"`
	LimitPrice        float64 `json:"limit_price"`
	LimitPriceType    string  `json:"limit_price_type"`
	// StopPrice is replaced by Triggers
	StopPrice   float64 `json:"stop_price"`
	TimeInForce string  `json:"time_in_force"`
	PostOnly    bool    `json:"post_only"`
	ReduceOnly  bool    `json:"reduce_only"`
	Margin      bool    `json:"margin"`
	// NoMarketPriceProtection is set when the order disabled market price protection
	NoMarketPriceProtection bool `json:"no_mpp"`
	// FeeCurrencyPreference is fcib to prefer fees in the base currency or fciq in the quote currency
	FeeCurrencyPreference string `json:"fee_ccy_pref"`
	// DisplayQuantity and DisplayQuantityRemaining describe an iceberg order's visible quantity
	DisplayQuantity          float64               `json:"display_qty"`
	DisplayQuantityRemaining float64               `json:"display_qty_remain"`
	EffectiveTime            time.Time             `json:"effective_time"`
	ExpireTime               time.Time             `json:"expire_time"`
	Triggers                 WsExecutionTriggers   `json:"triggers"`
	Contingent               WsExecutionContingent `json:"contingent"`
	ExecutionType            string                `json:"exec_type"`
	OrderStatus              string                `json:"order_status"`
	Amended                  bool                  `json:"amended"`
	Liquidated               bool                  `json:"liquidated"`
	Reason                   string                `json:"reason"`
	ExecutionID              string                `json:"exec_id"`
	TradeID                  uint64                `json:"trade_id"`
	// LastQuantity and LastPrice are the quantity and average price of this trade event
	LastQuantity float64 `json:"last_qty"`
	LastPrice    float64 `json:"last_price"`
	// LiquidityIndicator is t for a taker fill or m for a maker fill
	LiquidityIndicator string `json:"liquidity_ind"`
	// Cost is the value of this trade event
	Cost float64 `json:"cost"`
	// MarginBorrow is set when a fill increased or reduced margin borrowing
	MarginBorrow       bool             `json:"margin_borrow"`
	CumulativeQuantity float64          `json:"cum_qty"`
	CumulativeCost     float64          `json:"cum_cost"`
	AveragePrice       float64          `json:"avg_price"`
	Fees               []WsExecutionFee `json:"fees"`
	FeeUSDEquivalent   float64          `json:"fee_usd_equiv"`
	PositionStatus     string           `json:"position_status"`
	// OrderReferenceID is the transaction that created this order, such as the order whose fill created a close order
	OrderReferenceID string `json:"ord_ref_id"`
	// ExternalOrderID and ExternalExecutionID are a partner's identifiers
	ExternalOrderID     string `json:"ext_ord_id"`
	ExternalExecutionID string `json:"ext_exec_id"`
	// SenderSubID identifies an institutional sub-account or trader for self trade prevention
	SenderSubID string `json:"sender_sub_id"`
	// User identifies the user or sub-account the event belongs to
	User string `json:"user"`
	// RateCount is the trading rate-limit counter when the engine processed the event, sent when subscribed for it.
	// Kraken documents an integer, but the counter decays by fractions each second
	RateCount float64   `json:"ratecount"`
	Timestamp time.Time `json:"timestamp"`
	// Trigger and TriggeredPrice are a triggered order's reference price and the price that triggered it, which
	// Triggers replaces
	Trigger        string  `json:"trigger"`
	TriggeredPrice float64 `json:"triggered_price"`
	// CancelReason is replaced by Reason
	CancelReason string `json:"cancel_reason"`
}

// WsExecutionTriggers describes a triggered order's price trigger
type WsExecutionTriggers struct {
	Reference string  `json:"reference"`
	Price     float64 `json:"price"`
	// PriceType is static, pct for a percentage or quote for an offset in the quote currency
	PriceType string `json:"price_type"`
	// ActualPrice is the effective trigger price, for a relative or moving trigger
	ActualPrice float64 `json:"actual_price"`
	// PeakPrice is the highest or lowest price a trailing stop order follows
	PeakPrice float64 `json:"peak_price"`
	// LastPrice and Timestamp are the reference price and time when the trigger fired
	LastPrice float64   `json:"last_price"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// WsExecutionContingent describes the close orders each fill of an order creates
type WsExecutionContingent struct {
	OrderType        string  `json:"order_type"`
	TriggerPrice     float64 `json:"trigger_price"`
	TriggerPriceType string  `json:"trigger_price_type"`
	LimitPrice       float64 `json:"limit_price"`
	LimitPriceType   string  `json:"limit_price_type"`
}

// WsExecutionFee is a fee paid on a trade event
type WsExecutionFee struct {
	Asset    string  `json:"asset"`
	Quantity float64 `json:"qty"`
}

// WsBalance is an asset's balance in a balances snapshot
type WsBalance struct {
	Asset      string `json:"asset"`
	AssetClass string `json:"asset_class"`
	// Balance is held across every wallet
	Balance float64    `json:"balance"`
	Wallets []WsWallet `json:"wallets"`
}

// WsWallet is an asset's balance in one wallet
type WsWallet struct {
	// Type is spot or earn
	Type    string  `json:"type"`
	ID      string  `json:"id"`
	Balance float64 `json:"balance"`
}

// WsLedgerEntry is a ledger transaction a balances update pushes
type WsLedgerEntry struct {
	LedgerID string `json:"ledger_id"`
	// ReferenceID identifies what caused the entry, such as the trade of a trade entry
	ReferenceID string    `json:"ref_id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Subtype     string    `json:"subtype"`
	Category    string    `json:"category"`
	Asset       string    `json:"asset"`
	AssetClass  string    `json:"asset_class"`
	WalletType  string    `json:"wallet_type"`
	WalletID    string    `json:"wallet_id"`
	Amount      float64   `json:"amount"`
	Fee         float64   `json:"fee"`
	// Balance is the asset's balance in the account after the entry
	Balance float64 `json:"balance"`
	// User identifies the user or sub-account the entry belongs to
	User string `json:"user"`
}
