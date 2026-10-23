package gate

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OptionUnderlyingResponse represents an option underlying and its index price.
type OptionUnderlyingResponse struct {
	Name       string       `json:"name"`
	IndexPrice types.Number `json:"index_price"`
	IndexTime  types.Time   `json:"index_time"`
}

// OptionContractResponse represents an option contract detail.
type OptionContractResponse struct {
	Name        string       `json:"name"`
	Tag         string       `json:"tag"`
	IsCall      bool         `json:"is_call"`
	StrikePrice types.Number `json:"strike_price"`
	LastPrice   types.Number `json:"last_price"`
	MarkPrice   types.Number `json:"mark_price"`
	IndexPrice  types.Number `json:"index_price"`
	OrderbookID uint64       `json:"orderbook_id"`
	// Deprecated: Gate has deprecated it
	TradeID           uint64       `json:"trade_id"`
	TradeSize         types.Number `json:"trade_size"`
	PositionSize      types.Number `json:"position_size"`
	Underlying        string       `json:"underlying"`
	UnderlyingPrice   types.Number `json:"underlying_price"`
	Multiplier        types.Number `json:"multiplier"`
	OrderPriceRound   types.Number `json:"order_price_round"`
	MarkPriceRound    types.Number `json:"mark_price_round"`
	MakerFeeRate      types.Number `json:"maker_fee_rate"`
	TakerFeeRate      types.Number `json:"taker_fee_rate"`
	PriceLimitFeeRate types.Number `json:"price_limit_fee_rate"`
	RefDiscountRate   types.Number `json:"ref_discount_rate"`
	RefRebateRate     types.Number `json:"ref_rebate_rate"`
	// Deprecated: Gate has deprecated it
	OrderPriceDeviate types.Number `json:"order_price_deviate"`
	OrderSizeMin      types.Number `json:"order_size_min"`
	OrderSizeMax      types.Number `json:"order_size_max"`
	OrdersLimit       uint64       `json:"orders_limit"`
	CreateTime        types.Time   `json:"create_time"`
	ExpirationTime    types.Time   `json:"expiration_time"`
}

// OptionSettlementResponse list settlement history
type OptionSettlementResponse struct {
	Timestamp   types.Time   `json:"time"`
	Profit      types.Number `json:"profit"`
	Fee         types.Number `json:"fee"`
	SettlePrice types.Number `json:"settle_price"`
	Contract    string       `json:"contract"`
	StrikePrice types.Number `json:"strike_price"`
}

// MyOptionSettlementResponse represents option private settlement
type MyOptionSettlementResponse struct {
	Size         types.Number  `json:"size"`
	SettleProfit types.Number  `json:"settle_profit"`
	Contract     currency.Pair `json:"contract"`
	StrikePrice  types.Number  `json:"strike_price"`
	Time         types.Time    `json:"time"`
	SettlePrice  types.Number  `json:"settle_price"`
	Underlying   string        `json:"underlying"`
	RealisedPNL  types.Number  `json:"realised_pnl"`
	Fee          types.Number  `json:"fee"`
}

// OptionsTickerResponse represents tickers of options contracts
type OptionsTickerResponse struct {
	Name                  currency.Pair `json:"name"`
	LastPrice             types.Number  `json:"last_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	IndexPrice            types.Number  `json:"index_price"`
	PositionSize          types.Number  `json:"position_size"`
	Ask1Size              types.Number  `json:"ask1_size"`
	Ask1Price             types.Number  `json:"ask1_price"`
	Bid1Size              types.Number  `json:"bid1_size"`
	Bid1Price             types.Number  `json:"bid1_price"`
	Vega                  types.Number  `json:"vega"`
	Theta                 types.Number  `json:"theta"`
	Rho                   types.Number  `json:"rho"`
	Gamma                 types.Number  `json:"gamma"`
	Delta                 types.Number  `json:"delta"`
	MarkImpliedVolatility types.Number  `json:"mark_iv"`
	BidImpliedVolatility  types.Number  `json:"bid_iv"`
	AskImpliedVolatility  types.Number  `json:"ask_iv"`
	Leverage              types.Number  `json:"leverage"`
}

// OptionsUnderlyingTickerResponse represents underlying ticker
type OptionsUnderlyingTickerResponse struct {
	TradePut   uint64       `json:"trade_put"`
	TradeCall  uint64       `json:"trade_call"`
	IndexPrice types.Number `json:"index_price"`
}

// OptionTradeResponse represents an options market trade
type OptionTradeResponse struct {
	ID         uint64        `json:"id"`
	CreateTime types.Time    `json:"create_time"`
	Contract   currency.Pair `json:"contract"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
}

// OptionAccountResponse represents an option account.
type OptionAccountResponse struct {
	UserID                       uint64        `json:"user"`
	Total                        types.Number  `json:"total"`
	PositionValue                types.Number  `json:"position_value"`
	Equity                       types.Number  `json:"equity"`
	ShortEnabled                 bool          `json:"short_enabled"`
	MarketMakerProtectionEnabled bool          `json:"mmp_enabled"`
	LiquidationTriggered         bool          `json:"liq_triggered"`
	MarginMode                   uint8         `json:"margin_mode"`
	UnrealisedPNL                types.Number  `json:"unrealised_pnl"`
	InitialMargin                types.Number  `json:"init_margin"`
	MaintenanceMargin            types.Number  `json:"maint_margin"`
	OrderMargin                  types.Number  `json:"order_margin"`
	AskOrderMargin               types.Number  `json:"ask_order_margin"`
	BidOrderMargin               types.Number  `json:"bid_order_margin"`
	Available                    types.Number  `json:"available"`
	Point                        types.Number  `json:"point"`
	Currency                     currency.Code `json:"currency"`
	MaxOrderLimit                uint64        `json:"orders_limit"`
	PositionNotionalLimit        types.Number  `json:"position_notional_limit"`
}

// AccountBookResponse represents account changing history item
type AccountBookResponse struct {
	ChangeTime    types.Time   `json:"time"`
	AccountChange types.Number `json:"change"`
	Balance       types.Number `json:"balance"`
	CustomText    string       `json:"text"`
	ChangingType  string       `json:"type"`
}

// UsersPositionForUnderlyingResponse represents user's position for specified underlying.
type UsersPositionForUnderlyingResponse struct {
	User                  uint64        `json:"user"`
	Underlying            string        `json:"underlying"`
	UnderlyingPrice       types.Number  `json:"underlying_price"`
	Contract              currency.Pair `json:"contract"`
	Size                  int64         `json:"size"` // Negative for a short position
	EntryPrice            types.Number  `json:"entry_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	MarkImpliedVolatility types.Number  `json:"mark_iv"`
	RealisedPNL           types.Number  `json:"realised_pnl"`
	UnrealisedPNL         types.Number  `json:"unrealised_pnl"`
	PendingOrders         uint64        `json:"pending_orders"`
	CloseOrder            CloseOrder    `json:"close_order"`
	Delta                 types.Number  `json:"delta"`
	Gamma                 types.Number  `json:"gamma"`
	Vega                  types.Number  `json:"vega"`
	Theta                 types.Number  `json:"theta"`
}

// ContractClosePositionResponse represents user's liquidation history
type ContractClosePositionResponse struct {
	PositionCloseTime types.Time   `json:"time"`
	Contract          string       `json:"contract"`
	Side              string       `json:"side"` // Position side, long or short
	PNL               types.Number `json:"pnl"`
	CloseOrderText    string       `json:"text"`
	SettleSize        types.Number `json:"settle_size"`
}

// OptionOrderRequest represents option order request body
type OptionOrderRequest struct {
	Contract              currency.Pair `json:"contract"`
	OrderSize             types.Number  `json:"size"`              // Positive to buy, negative to sell, zero with Close set to close the position
	Iceberg               int64         `json:"iceberg,omitempty"` // Display size for iceberg order. 0 for non-iceberg. Note that you will have to pay the taker fee for the hidden size
	Price                 types.Number  `json:"price"`             // Zero with an IOC time in force places a market order
	Close                 bool          `json:"close,omitempty"`
	ReduceOnly            bool          `json:"reduce_only,omitempty"`
	MarketMakerProtection bool          `json:"mmp,omitempty"`
	TimeInForce           string        `json:"tif,omitempty"`
	Text                  string        `json:"text,omitempty"`
}

// MarshalJSON sends the size as a JSON integer and the price as a decimal string, since types.Number
// marshals both as strings and zero as "", whereas Gate expects a zero size to close a position and
// a "0" price for a market order
func (o OptionOrderRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // a value receiver keeps the encoding when a value rather than a pointer is marshalled
	type alias OptionOrderRequest
	return json.Marshal(struct {
		*alias
		OrderSize float64 `json:"size"`
		Price     string  `json:"price"`
	}{
		alias:     (*alias)(&o),
		OrderSize: o.OrderSize.Float64(),
		Price:     strconv.FormatFloat(o.Price.Float64(), 'f', -1, 64),
	})
}

// OptionOrderResponse represents option order response detail
type OptionOrderResponse struct {
	OptionOrderID           uint64       `json:"id"`
	User                    uint64       `json:"user"`
	CreateTime              types.Time   `json:"create_time"`
	FinishTime              types.Time   `json:"finish_time"`
	FinishAs                string       `json:"finish_as"`
	Status                  string       `json:"status"`
	Contract                string       `json:"contract"`
	Size                    types.Number `json:"size"`
	Iceberg                 uint64       `json:"iceberg"`
	Price                   types.Number `json:"price"`
	IsOrderPositionClose    bool         `json:"is_close"`
	IsReduceOnly            bool         `json:"is_reduce_only"`
	IsOrderLiquidation      bool         `json:"is_liq"`
	IsMarketMakerProtection bool         `json:"is_mmp"`
	TimeInForce             string       `json:"tif"`
	RemainingAmount         types.Number `json:"left"`
	FillPrice               types.Number `json:"fill_price"`
	Text                    string       `json:"text"`
	TakerFeeRate            types.Number `json:"tkfr"`
	MakerFeeRate            types.Number `json:"mkfr"`
	ReferenceUserID         uint64       `json:"refu"`
	ReferrerRebate          types.Number `json:"refr"`
}

// AmendOptionOrderRequest represents a request to amend the price and size of an open options order
type AmendOptionOrderRequest struct {
	Contract currency.Pair `json:"contract"`
	Price    types.Number  `json:"price"`
	Size     int64         `json:"size"` // Negative to sell
}

// CountdownCancelOptionOrdersRequest represents a request to set or clear the options order countdown, scoped
// to a contract or underlying when either is set
type CountdownCancelOptionOrdersRequest struct {
	Countdown  types.DurationSeconds `json:"timeout"` // Whole seconds of at least 5; zero clears the countdown
	Contract   currency.Pair         `json:"contract,omitzero"`
	Underlying string                `json:"underlying,omitempty"`
}

// OptionsMarketMakerProtectionRequest represents market maker protection settings for an options underlying
type OptionsMarketMakerProtectionRequest struct {
	Underlying    string
	Window        time.Duration // Up to 5 seconds in whole milliseconds; zero disables market maker protection
	FrozenPeriod  time.Duration // Whole milliseconds; zero keeps trading frozen until a reset
	QuantityLimit types.Number
	DeltaLimit    types.Number
}

// MarshalJSON sends the window and frozen period in milliseconds, keeping a zero value since each one
// carries meaning on the wire
func (o OptionsMarketMakerProtectionRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Underlying    string       `json:"underlying"`
		Window        int64        `json:"window"`
		FrozenPeriod  int64        `json:"frozen_period"`
		QuantityLimit types.Number `json:"qty_limit"`
		DeltaLimit    types.Number `json:"delta_limit"`
	}{
		Underlying:    o.Underlying,
		Window:        o.Window.Milliseconds(),
		FrozenPeriod:  o.FrozenPeriod.Milliseconds(),
		QuantityLimit: o.QuantityLimit,
		DeltaLimit:    o.DeltaLimit,
	})
}

// OptionsMarketMakerProtectionResponse represents the market maker protection settings and state of an options
// underlying
type OptionsMarketMakerProtectionResponse struct {
	Underlying    string                     `json:"underlying"`
	Window        types.DurationMilliseconds `json:"window"`
	FrozenPeriod  types.DurationMilliseconds `json:"frozen_period"`
	QuantityLimit types.Number               `json:"qty_limit"`
	DeltaLimit    types.Number               `json:"delta_limit"`
	TriggerTime   types.Time                 `json:"trigger_time_ms"`
	FrozenUntil   types.Time                 `json:"frozen_until_ms"`
}

// OptionTradingHistoryResponse list personal trading history
type OptionTradingHistoryResponse struct {
	ID              uint64        `json:"id"`
	UnderlyingPrice types.Number  `json:"underlying_price"`
	Size            types.Number  `json:"size"`
	Contract        currency.Pair `json:"contract"`
	TradeRole       string        `json:"role"`
	CreateTime      types.Time    `json:"create_time"`
	OrderID         uint64        `json:"order_id"`
	Price           types.Number  `json:"price"`
}
