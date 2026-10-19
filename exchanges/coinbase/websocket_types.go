package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// websocketRequest is an Advanced Trade subscribe or unsubscribe request, for one channel
type websocketRequest struct {
	Type       string   `json:"type"`
	Channel    string   `json:"channel"`
	ProductIDs []string `json:"product_ids,omitempty"`
	// JWT authenticates the request; the user and futures_balance_summary channels require one
	JWT string `json:"jwt,omitempty"`
}

// websocketEnvelope is every Advanced Trade websocket message's envelope; an error carries only its type and message
type websocketEnvelope struct {
	Channel   string          `json:"channel"`
	ClientID  string          `json:"client_id"`
	Timestamp time.Time       `json:"timestamp"`
	Sequence  uint64          `json:"sequence_num"`
	Events    json.RawMessage `json:"events"`
	// Type is error for an error, which reports the failure in Message
	Type    string `json:"type"`
	Message string `json:"message"`
}

// websocketSubscriptionsEvent lists every channel a connection is subscribed to, and its products; Coinbase sends one
// after each subscribe and unsubscribe request
type websocketSubscriptionsEvent struct {
	Subscriptions map[string][]string `json:"subscriptions"`
}

// WebsocketHeartbeat is a heartbeats channel event, sent every second
type WebsocketHeartbeat struct {
	// CurrentTime is Coinbase's time in Go's time.Time String layout, with a monotonic reading appended
	CurrentTime string `json:"current_time"`
	// HeartbeatCounter increments with each heartbeat
	HeartbeatCounter uint64 `json:"heartbeat_counter"`
}

// WebsocketCandlesEvent holds candles channel candles, five minutes each
type WebsocketCandlesEvent struct {
	Type    string            `json:"type"`
	Candles []WebsocketCandle `json:"candles"`
}

// WebsocketCandle is a product's five minute candle
type WebsocketCandle struct {
	Start     types.Time   `json:"start"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Open      types.Number `json:"open"`
	Close     types.Number `json:"close"`
	Volume    types.Number `json:"volume"`
	ProductID string       `json:"product_id"`
}

// WebsocketStatusEvent holds status channel products
type WebsocketStatusEvent struct {
	Type     string                   `json:"type"`
	Products []WebsocketProductStatus `json:"products"`
}

// WebsocketProductStatus is a product's trading status
type WebsocketProductStatus struct {
	ProductType        string       `json:"product_type"`
	ID                 string       `json:"id"`
	BaseCurrency       string       `json:"base_currency"`
	QuoteCurrency      string       `json:"quote_currency"`
	BaseIncrement      types.Number `json:"base_increment"`
	QuoteIncrement     types.Number `json:"quote_increment"`
	DisplayName        string       `json:"display_name"`
	Status             string       `json:"status"`
	StatusMessage      string       `json:"status_message"`
	MinimumMarketFunds types.Number `json:"min_market_funds"`
}

// websocketTickerEvent holds ticker or ticker_batch channel tickers
type websocketTickerEvent struct {
	Type    string            `json:"type"`
	Tickers []WebsocketTicker `json:"tickers"`
}

// WebsocketTicker is a product's ticker; ticker_batch leaves out the best bid and ask
type WebsocketTicker struct {
	Type                        string       `json:"type"`
	ProductID                   string       `json:"product_id"`
	Price                       types.Number `json:"price"`
	Volume24Hour                types.Number `json:"volume_24_h"`
	Low24Hour                   types.Number `json:"low_24_h"`
	High24Hour                  types.Number `json:"high_24_h"`
	Low52Week                   types.Number `json:"low_52_w"`
	High52Week                  types.Number `json:"high_52_w"`
	PricePercentageChange24Hour types.Number `json:"price_percent_chg_24_h"`
	BestBid                     types.Number `json:"best_bid"`
	BestAsk                     types.Number `json:"best_ask"`
	BestBidQuantity             types.Number `json:"best_bid_quantity"`
	BestAskQuantity             types.Number `json:"best_ask_quantity"`
}

// websocketLevel2Event holds a product's book snapshot or the levels that changed since the last event
type websocketLevel2Event struct {
	// Type is snapshot or update
	Type      string                  `json:"type"`
	ProductID string                  `json:"product_id"`
	Updates   []websocketLevel2Update `json:"updates"`
}

// websocketLevel2Update is a book level's new quantity, which replaces the level's, and removes it when zero
type websocketLevel2Update struct {
	// Side is bid or offer
	Side        string       `json:"side"`
	EventTime   time.Time    `json:"event_time"`
	PriceLevel  types.Number `json:"price_level"`
	NewQuantity types.Number `json:"new_quantity"`
}

// websocketMarketTradesEvent holds trades batched over 250 milliseconds
type websocketMarketTradesEvent struct {
	Type   string                 `json:"type"`
	Trades []WebsocketMarketTrade `json:"trades"`
}

// WebsocketMarketTrade is a trade in a product's market
type WebsocketMarketTrade struct {
	TradeID   string       `json:"trade_id"`
	ProductID string       `json:"product_id"`
	Price     types.Number `json:"price"`
	Size      types.Number `json:"size"`
	// Side is the maker's side of the trade
	Side string    `json:"side"`
	Time time.Time `json:"time"`
}

// WebsocketUserEvent holds the account's orders and positions, all of them in a snapshot and the changed ones in an
// update
type WebsocketUserEvent struct {
	Type      string                 `json:"type"`
	Orders    []WebsocketUserOrder   `json:"orders"`
	Positions WebsocketUserPositions `json:"positions"`
}

// WebsocketUserOrder is an order of the account
type WebsocketUserOrder struct {
	AveragePrice         types.Number `json:"avg_price"`
	CancelReason         string       `json:"cancel_reason"`
	ClientOrderID        string       `json:"client_order_id"`
	CompletionPercentage types.Number `json:"completion_percentage"`
	ContractExpiryType   string       `json:"contract_expiry_type"`
	// CumulativeQuantity is how much has filled, in the base currency
	CumulativeQuantity types.Number `json:"cumulative_quantity"`
	FilledValue        types.Number `json:"filled_value"`
	// LeavesQuantity is how much remains, in the currency the order was placed in, base or quote
	LeavesQuantity        types.Number  `json:"leaves_quantity"`
	LimitPrice            types.Number  `json:"limit_price"`
	NumberOfFills         types.Number  `json:"number_of_fills"`
	OrderID               string        `json:"order_id"`
	OrderSide             string        `json:"order_side"`
	OrderType             string        `json:"order_type"`
	OutstandingHoldAmount types.Number  `json:"outstanding_hold_amount"`
	PostOnly              types.Boolean `json:"post_only"`
	ProductID             string        `json:"product_id"`
	ProductType           string        `json:"product_type"`
	RejectReason          string        `json:"reject_reason"`
	RetailPortfolioID     string        `json:"retail_portfolio_id"`
	RiskManagedBy         string        `json:"risk_managed_by"`
	// Status is PENDING, OPEN, FILLED, CANCEL_QUEUED, CANCELLED, EXPIRED or FAILED
	Status              string       `json:"status"`
	StopPrice           types.Number `json:"stop_price"`
	TimeInForce         string       `json:"time_in_force"`
	TotalFees           types.Number `json:"total_fees"`
	TotalValueAfterFees types.Number `json:"total_value_after_fees"`
	TriggerStatus       string       `json:"trigger_status"`
	CreationTime        time.Time    `json:"creation_time"`
	// EndTime and StartTime are 0001-01-01T00:00:00Z when they do not apply
	EndTime   time.Time `json:"end_time"`
	StartTime time.Time `json:"start_time"`
}

// WebsocketUserPositions holds the account's futures positions
type WebsocketUserPositions struct {
	PerpetualFuturesPositions []WebsocketPerpetualFuturesPosition `json:"perpetual_futures_positions"`
	ExpiringFuturesPositions  []WebsocketExpiringFuturesPosition  `json:"expiring_futures_positions"`
	// PredictionMarketPositions is a beta field Coinbase sends empty, so it is kept as sent
	PredictionMarketPositions json.RawMessage `json:"prediction_market_positions"`
}

// WebsocketPerpetualFuturesPosition is an INTX perpetual position, which the user channel still documents
type WebsocketPerpetualFuturesPosition struct {
	ProductID     string       `json:"product_id"`
	PortfolioUUID string       `json:"portfolio_uuid"`
	VWAP          types.Number `json:"vwap"`
	EntryVWAP     types.Number `json:"entry_vwap"`
	// PositionSide is Long or Short
	PositionSide string `json:"position_side"`
	// MarginType is Cross or Isolated
	MarginType                string       `json:"margin_type"`
	NetSize                   types.Number `json:"net_size"`
	BuyOrderSize              types.Number `json:"buy_order_size"`
	SellOrderSize             types.Number `json:"sell_order_size"`
	Leverage                  types.Number `json:"leverage"`
	MarkPrice                 types.Number `json:"mark_price"`
	LiquidationPrice          types.Number `json:"liquidation_price"`
	InitialMarginNotional     types.Number `json:"im_notional"`
	MaintenanceMarginNotional types.Number `json:"mm_notional"`
	PositionNotional          types.Number `json:"position_notional"`
	UnrealisedPNL             types.Number `json:"unrealized_pnl"`
	AggregatedPNL             types.Number `json:"aggregated_pnl"`
}

// WebsocketExpiringFuturesPosition is a US derivatives position
type WebsocketExpiringFuturesPosition struct {
	ProductID string `json:"product_id"`
	// Side is Long or Short
	Side              string       `json:"side"`
	NumberOfContracts types.Number `json:"number_of_contracts"`
	RealisedPNL       types.Number `json:"realized_pnl"`
	UnrealisedPNL     types.Number `json:"unrealized_pnl"`
	EntryPrice        types.Number `json:"entry_price"`
}

// WebsocketFuturesBalanceSummaryEvent holds the account's US derivatives balances
type WebsocketFuturesBalanceSummaryEvent struct {
	Type              string                         `json:"type"`
	FCMBalanceSummary WebsocketFuturesBalanceSummary `json:"fcm_balance_summary"`
}

// WebsocketFuturesBalanceSummary is the account's US derivatives balances and margin
type WebsocketFuturesBalanceSummary struct {
	FuturesBuyingPower           types.Number                 `json:"futures_buying_power"`
	TotalUSDBalance              types.Number                 `json:"total_usd_balance"`
	CBIUSDBalance                types.Number                 `json:"cbi_usd_balance"`
	CFMUSDBalance                types.Number                 `json:"cfm_usd_balance"`
	TotalOpenOrdersHoldAmount    types.Number                 `json:"total_open_orders_hold_amount"`
	UnrealisedPNL                types.Number                 `json:"unrealized_pnl"`
	DailyRealisedPNL             types.Number                 `json:"daily_realized_pnl"`
	InitialMargin                types.Number                 `json:"initial_margin"`
	AvailableMargin              types.Number                 `json:"available_margin"`
	LiquidationThreshold         types.Number                 `json:"liquidation_threshold"`
	LiquidationBufferAmount      types.Number                 `json:"liquidation_buffer_amount"`
	LiquidationBufferPercentage  types.Number                 `json:"liquidation_buffer_percentage"`
	IntradayMarginWindowMeasure  WebsocketMarginWindowMeasure `json:"intraday_margin_window_measure"`
	OvernightMarginWindowMeasure WebsocketMarginWindowMeasure `json:"overnight_margin_window_measure"`
}

// WebsocketMarginWindowMeasure is the margin a US derivatives margin window requires
type WebsocketMarginWindowMeasure struct {
	MarginWindowType            string       `json:"margin_window_type"`
	MarginLevel                 string       `json:"margin_level"`
	InitialMargin               types.Number `json:"initial_margin"`
	MaintenanceMargin           types.Number `json:"maintenance_margin"`
	LiquidationBufferPercentage types.Number `json:"liquidation_buffer_percentage"`
	TotalHold                   types.Number `json:"total_hold"`
	FuturesBuyingPower          types.Number `json:"futures_buying_power"`
}
