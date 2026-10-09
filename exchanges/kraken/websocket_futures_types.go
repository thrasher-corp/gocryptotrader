package kraken

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errFuturesChallengeEmpty     = errors.New("futures websocket challenge empty")
	errFuturesSequenceGap        = errors.New("futures order book sequence gap")
	errFuturesSubscriptionFailed = errors.New("futures websocket subscription failed")
	errFuturesUnexpectedReply    = errors.New("unexpected futures websocket reply")
	errFuturesUnsupportedFeed    = errors.New("futures websocket feed not supported")
	errFuturesOrderbookLevels    = errors.New("futures order book levels not supported")
)

// wsFuturesRequest is a futures websocket request: a challenge, or a subscription to a feed. A private feed's
// subscription carries the API key with a challenge and its signature
type wsFuturesRequest struct {
	Event             string   `json:"event"`
	Feed              string   `json:"feed,omitempty"`
	ProductIDs        []string `json:"product_ids,omitempty"`
	APIKey            string   `json:"api_key,omitempty"`
	OriginalChallenge string   `json:"original_challenge,omitempty"`
	SignedChallenge   string   `json:"signed_challenge,omitempty"`
}

// wsFuturesEvent is a futures websocket event: an acknowledgement, a challenge, or an alert or error, whose message
// says what failed but not which request it answers
type wsFuturesEvent struct {
	Event      string   `json:"event"`
	Feed       string   `json:"feed"`
	ProductIDs []string `json:"product_ids"`
	Message    string   `json:"message"`
}

// WsFuturesTicker is a futures market's ticker, which the ticker feed sends in full every second
type WsFuturesTicker struct {
	Time      types.Time `json:"time"`
	ProductID string     `json:"product_id"`
	// FundingRate is the current absolute funding rate, and RelativeFundingRate the same as a fraction of the spot price
	// when it was calculated. The funding fields are sent for perpetuals only
	FundingRate                   float64    `json:"funding_rate"`
	FundingRatePrediction         float64    `json:"funding_rate_prediction"`
	RelativeFundingRate           float64    `json:"relative_funding_rate"`
	RelativeFundingRatePrediction float64    `json:"relative_funding_rate_prediction"`
	NextFundingRateTime           types.Time `json:"next_funding_rate_time"`
	Feed                          string     `json:"feed"`
	Bid                           float64    `json:"bid"`
	Ask                           float64    `json:"ask"`
	BidSize                       float64    `json:"bid_size"`
	AskSize                       float64    `json:"ask_size"`
	// Volume is the size of the fills of the last 24 hours, and VolumeQuote the same in the quote currency for
	// multi-collateral futures
	Volume         float64 `json:"volume"`
	VolumeQuote    float64 `json:"volumeQuote"`
	DaysToMaturity uint64  `json:"dtm"`
	// Leverage is the market's leverage, such as 50x
	Leverage string  `json:"leverage"`
	Index    float64 `json:"index"`
	// Premium is the percentage difference between the futures price and the index price
	Premium float64 `json:"premium"`
	Last    float64 `json:"last"`
	// Change is the percentage change in price over the last 24 hours
	Change    float64 `json:"change"`
	Suspended bool    `json:"suspended"`
	// Tag groups the market by expiry, such as perpetual, month or quarter
	Tag string `json:"tag"`
	// Pair is the market's currency pair, such as XBT:USD
	Pair         string           `json:"pair"`
	OpenInterest float64          `json:"openInterest"`
	MarkPrice    float64          `json:"markPrice"`
	MaturityTime types.Time       `json:"maturityTime"`
	PostOnly     bool             `json:"post_only"`
	Open24Hour   float64          `json:"open"`
	High24Hour   float64          `json:"high"`
	Low24Hour    float64          `json:"low"`
	Greeks       *WsFuturesGreeks `json:"greeks"`
}

// WsFuturesGreeks holds an option's implied volatility and Greeks
type WsFuturesGreeks struct {
	ImpliedVolatility float64 `json:"iv"`
	Delta             float64 `json:"delta"`
	Theta             float64 `json:"theta"`
	Gamma             float64 `json:"gamma"`
	Vega              float64 `json:"vega"`
	Rho               float64 `json:"rho"`
}

// WsFuturesBookSnapshot is a futures market's whole order book, which updates follow from the next sequence number
type WsFuturesBookSnapshot struct {
	Feed      string               `json:"feed"`
	ProductID string               `json:"product_id"`
	Timestamp types.Time           `json:"timestamp"`
	Sequence  uint64               `json:"seq"`
	Bids      []WsFuturesBookLevel `json:"bids"`
	Asks      []WsFuturesBookLevel `json:"asks"`
}

// WsFuturesBookLevel is a futures order book price level
type WsFuturesBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"qty"`
}

// WsFuturesBookUpdate changes one futures order book price level; a zero quantity removes the level
type WsFuturesBookUpdate struct {
	Feed      string `json:"feed"`
	ProductID string `json:"product_id"`
	// Side is buy for a bid or sell for an ask
	Side      string     `json:"side"`
	Sequence  uint64     `json:"seq"`
	Price     float64    `json:"price"`
	Quantity  float64    `json:"qty"`
	Timestamp types.Time `json:"timestamp"`
}

// WsFuturesTradeSnapshot holds a futures market's recent trades, newest first
type WsFuturesTradeSnapshot struct {
	Feed      string           `json:"feed"`
	ProductID string           `json:"product_id"`
	Trades    []WsFuturesTrade `json:"trades"`
}

// WsFuturesTrade is a futures trade
type WsFuturesTrade struct {
	Feed      string `json:"feed"`
	ProductID string `json:"product_id"`
	UID       string `json:"uid"`
	// Side is the taker's side, buy or sell
	Side string `json:"side"`
	// Type is fill, liquidation, termination, or block for part of a block trade
	Type     string     `json:"type"`
	Sequence uint64     `json:"seq"`
	Time     types.Time `json:"time"`
	Quantity float64    `json:"qty"`
	Price    float64    `json:"price"`
}

// WsFuturesOpenOrdersSnapshot holds the account's open futures orders
type WsFuturesOpenOrdersSnapshot struct {
	Feed    string               `json:"feed"`
	Account string               `json:"account"`
	Orders  []WsFuturesOpenOrder `json:"orders"`
}

// WsFuturesOpenOrderUpdate reports a change to an open futures order. An order that is filled, cancelled or rejected
// is removed, with IsCancel set, and Kraken sends only its ID when it is cancelled
type WsFuturesOpenOrderUpdate struct {
	Feed     string              `json:"feed"`
	Order    *WsFuturesOpenOrder `json:"order"`
	OrderID  string              `json:"order_id"`
	IsCancel bool                `json:"is_cancel"`
	// Reason is why the order changed, such as new_placed_order_by_user, partial_fill, full_fill or cancelled_by_user
	Reason string `json:"reason"`
}

// WsFuturesOpenOrder is an open futures order
type WsFuturesOpenOrder struct {
	Instrument     string     `json:"instrument"`
	Time           types.Time `json:"time"`
	LastUpdateTime types.Time `json:"last_update_time"`
	Quantity       float64    `json:"qty"`
	Filled         float64    `json:"filled"`
	LimitPrice     float64    `json:"limit_price"`
	StopPrice      float64    `json:"stop_price"`
	// Type is limit, stop, take_profit or trailing_stop
	Type          string `json:"type"`
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cli_ord_id"`
	// Direction is 0 for a buy order and 1 for a sell order
	Direction  uint8 `json:"direction"`
	ReduceOnly bool  `json:"reduce_only"`
	// TriggerSignal is the price a triggered order follows: last, mark or spot
	TriggerSignal       string                        `json:"triggerSignal"`
	TrailingStopOptions *WsFuturesTrailingStopOptions `json:"trailing_stop_options"`
}

// WsFuturesTrailingStopOptions holds the deviation a trailing stop's trigger price follows the trigger signal at
type WsFuturesTrailingStopOptions struct {
	MaxDeviation float64 `json:"max_deviation"`
	// Unit is percent or quote_currency
	Unit string `json:"unit"`
}

// WsFuturesFills holds the account's futures fills: its recent fills on subscribing, then each new fill
type WsFuturesFills struct {
	Feed    string          `json:"feed"`
	Account string          `json:"account"`
	Fills   []WsFuturesFill `json:"fills"`
}

// WsFuturesFill is a fill of the account's futures order
type WsFuturesFill struct {
	Instrument             string     `json:"instrument"`
	Time                   types.Time `json:"time"`
	Price                  float64    `json:"price"`
	Sequence               uint64     `json:"seq"`
	Buy                    bool       `json:"buy"`
	Quantity               float64    `json:"qty"`
	RemainingOrderQuantity float64    `json:"remaining_order_qty"`
	OrderID                string     `json:"order_id"`
	ClientOrderID          string     `json:"cli_ord_id"`
	FillID                 string     `json:"fill_id"`
	// FillType is maker, taker, liquidation, assignee, assignor, unwindBankrupt, unwindCounterparty or takerAfterEdit
	FillType    string  `json:"fill_type"`
	FeePaid     float64 `json:"fee_paid"`
	FeeCurrency string  `json:"fee_currency"`
	// TakerOrderType is the order type of the taker's order, and OrderType that of the account's order, such as lmt,
	// ioc or post
	TakerOrderType string `json:"taker_order_type"`
	OrderType      string `json:"order_type"`
}

// WsFuturesBalances holds the account's futures balances: all of them on subscribing, then the wallets that changed
type WsFuturesBalances struct {
	Feed      string     `json:"feed"`
	Account   string     `json:"account"`
	Timestamp types.Time `json:"timestamp"`
	Sequence  uint64     `json:"seq"`
	// Holding is the cash wallet's balance of each currency
	Holding map[string]float64 `json:"holding"`
	// Futures holds the single-collateral margin accounts, keyed by name, such as F-XBT:USD
	Futures     map[string]WsFuturesMarginAccount `json:"futures"`
	FlexFutures *WsFuturesFlexAccount             `json:"flex_futures"`
}

// WsFuturesMarginAccount is a single-collateral margin account
type WsFuturesMarginAccount struct {
	Name string `json:"name"`
	// Pair is the account's currency pair, such as XBT/USD, and Unit the currency it holds as margin
	Pair              string  `json:"pair"`
	Unit              string  `json:"unit"`
	PortfolioValue    float64 `json:"portfolio_value"`
	Balance           float64 `json:"balance"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	InitialMargin     float64 `json:"initial_margin"`
	Available         float64 `json:"available"`
	UnrealizedFunding float64 `json:"unrealized_funding"`
	PNL               float64 `json:"pnl"`
}

// WsFuturesFlexAccount is the multi-collateral wallet
type WsFuturesFlexAccount struct {
	Currencies                 map[string]WsFuturesFlexCurrency       `json:"currencies"`
	BalanceValue               float64                                `json:"balance_value"`
	PortfolioValue             float64                                `json:"portfolio_value"`
	CollateralValue            float64                                `json:"collateral_value"`
	InitialMargin              float64                                `json:"initial_margin"`
	InitialMarginWithoutOrders float64                                `json:"initial_margin_without_orders"`
	MaintenanceMargin          float64                                `json:"maintenance_margin"`
	PNL                        float64                                `json:"pnl"`
	UnrealizedFunding          float64                                `json:"unrealized_funding"`
	TotalUnrealized            float64                                `json:"total_unrealized"`
	TotalUnrealizedAsMargin    float64                                `json:"total_unrealized_as_margin"`
	MarginEquity               float64                                `json:"margin_equity"`
	AvailableMargin            float64                                `json:"available_margin"`
	TotalPositionSize          float64                                `json:"total_position_size"`
	UnifiedBalances            bool                                   `json:"unified_balances"`
	Isolated                   map[string]WsFuturesFlexIsolatedMargin `json:"isolated"`
	Cross                      *WsFuturesFlexCrossMargin              `json:"cross"`
}

// WsFuturesFlexCurrency is a currency in the multi-collateral wallet
type WsFuturesFlexCurrency struct {
	Quantity        float64 `json:"quantity"`
	Value           float64 `json:"value"`
	CollateralValue float64 `json:"collateral_value"`
	// Available is the quantity available for margin, which unrealised profit can take beyond the quantity
	Available        float64 `json:"available"`
	Haircut          float64 `json:"haircut"`
	ConversionSpread float64 `json:"conversion_spread"`
}

// WsFuturesFlexIsolatedMargin is the margin of an isolated multi-collateral position
type WsFuturesFlexIsolatedMargin struct {
	InitialMargin              float64 `json:"initial_margin"`
	InitialMarginWithoutOrders float64 `json:"initial_margin_without_orders"`
	MaintenanceMargin          float64 `json:"maintenance_margin"`
	PNL                        float64 `json:"pnl"`
	UnrealizedFunding          float64 `json:"unrealized_funding"`
	TotalUnrealized            float64 `json:"total_unrealized"`
	TotalUnrealizedAsMargin    float64 `json:"total_unrealized_as_margin"`
}

// WsFuturesFlexCrossMargin is the margin of the multi-collateral cross margin positions
type WsFuturesFlexCrossMargin struct {
	BalanceValue               float64 `json:"balance_value"`
	PortfolioValue             float64 `json:"portfolio_value"`
	CollateralValue            float64 `json:"collateral_value"`
	InitialMargin              float64 `json:"initial_margin"`
	InitialMarginWithoutOrders float64 `json:"initial_margin_without_orders"`
	MaintenanceMargin          float64 `json:"maintenance_margin"`
	PNL                        float64 `json:"pnl"`
	UnrealizedFunding          float64 `json:"unrealized_funding"`
	TotalUnrealized            float64 `json:"total_unrealized"`
	TotalUnrealizedAsMargin    float64 `json:"total_unrealized_as_margin"`
	MarginEquity               float64 `json:"margin_equity"`
	AvailableMargin            float64 `json:"available_margin"`
	EffectiveLeverage          float64 `json:"effective_leverage"`
}

// WsFuturesPositions holds the account's open futures positions
type WsFuturesPositions struct {
	Feed      string              `json:"feed"`
	Account   string              `json:"account"`
	Sequence  uint64              `json:"seq"`
	Timestamp types.Time          `json:"timestamp"`
	Positions []WsFuturesPosition `json:"positions"`
}

// WsFuturesPosition is an open futures position
type WsFuturesPosition struct {
	Instrument string `json:"instrument"`
	// Balance is the position's size, negative when short
	Balance                 float64 `json:"balance"`
	PNL                     float64 `json:"pnl"`
	EntryPrice              float64 `json:"entry_price"`
	MarkPrice               float64 `json:"mark_price"`
	IndexPrice              float64 `json:"index_price"`
	LiquidationThreshold    float64 `json:"liquidation_threshold"`
	ReturnOnEquity          float64 `json:"return_on_equity"`
	UnrealizedFunding       float64 `json:"unrealized_funding"`
	EffectiveLeverage       float64 `json:"effective_leverage"`
	InitialMargin           float64 `json:"initial_margin"`
	InitialMarginWithOrders float64 `json:"initial_margin_with_orders"`
	MaintenanceMargin       float64 `json:"maintenance_margin"`
	PNLCurrency             string  `json:"pnl_currency"`
	MaxFixedLeverage        float64 `json:"max_fixed_leverage"`
	// ImpliedVolatility and the Greeks are sent for options positions
	ImpliedVolatility float64 `json:"iv"`
	Delta             float64 `json:"delta"`
	Theta             float64 `json:"theta"`
	Gamma             float64 `json:"gamma"`
	Vega              float64 `json:"vega"`
	Rho               float64 `json:"rho"`
}

// WsFuturesNotifications holds notifications for the account, such as of maintenance
type WsFuturesNotifications struct {
	Feed          string                  `json:"feed"`
	Notifications []WsFuturesNotification `json:"notifications"`
}

// WsFuturesNotification is a notification for the account
type WsFuturesNotification struct {
	ID uint64 `json:"id"`
	// Type is market, general, new_feature, bug_fix, maintenance or settlement
	Type string `json:"type"`
	// Priority is low, medium or high; a high priority maintenance notification means downtime at the effective time
	Priority                string     `json:"priority"`
	Note                    string     `json:"note"`
	EffectiveTime           types.Time `json:"effective_time"`
	ExpectedDowntimeMinutes uint64     `json:"expected_downtime_minutes"`
}
