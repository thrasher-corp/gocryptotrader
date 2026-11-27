package binanceus

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Side is the side parameter of an order
type Side string

// Order sides
const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType is the type parameter of an order
type OrderType string

// Order types. STOP_LOSS and TAKE_PROFIT orders place a market order when they trigger, and STOP_LOSS_LIMIT and
// TAKE_PROFIT_LIMIT orders a limit order. A symbol lists the types it takes in its exchange information
const (
	OrderTypeLimit           OrderType = "LIMIT"
	OrderTypeMarket          OrderType = "MARKET"
	OrderTypeStopLoss        OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"
)

// TimeInForce is how long an order stays on the order book, the timeInForce parameter of an order
type TimeInForce string

// Time in force values
const (
	TimeInForceGoodTillCancel    TimeInForce = "GTC"
	TimeInForceImmediateOrCancel TimeInForce = "IOC"
	TimeInForceFillOrKill        TimeInForce = "FOK"
)

// SelfTradePreventionMode is the selfTradePreventionMode parameter of an order
type SelfTradePreventionMode string

// Self-trade prevention modes. A symbol lists the modes it takes in its exchange information; OCO orders take the
// EXPIRE modes only
const (
	SelfTradePreventionModeNone        SelfTradePreventionMode = "NONE"
	SelfTradePreventionModeExpireTaker SelfTradePreventionMode = "EXPIRE_TAKER"
	SelfTradePreventionModeExpireMaker SelfTradePreventionMode = "EXPIRE_MAKER"
	SelfTradePreventionModeExpireBoth  SelfTradePreventionMode = "EXPIRE_BOTH"
	SelfTradePreventionModeDecrement   SelfTradePreventionMode = "DECREMENT"
	SelfTradePreventionModeTransfer    SelfTradePreventionMode = "TRANSFER"
)

// NewOrderResponseType is the newOrderRespType parameter of an order
type NewOrderResponseType string

// New order response types: ACK carries the order's identifiers, RESULT adds its state and FULL adds its fills. MARKET
// and LIMIT orders default to FULL and the other types to ACK
const (
	NewOrderResponseTypeAcknowledgement NewOrderResponseType = "ACK"
	NewOrderResponseTypeResult          NewOrderResponseType = "RESULT"
	NewOrderResponseTypeFull            NewOrderResponseType = "FULL"
)

// CancelReplaceMode is the cancelReplaceMode parameter of a cancel replace order
type CancelReplaceMode string

// Cancel replace modes: STOP_ON_FAILURE does not attempt the new order when the cancel fails, and ALLOW_FAILURE
// attempts it regardless
const (
	CancelReplaceModeStopOnFailure CancelReplaceMode = "STOP_ON_FAILURE"
	CancelReplaceModeAllowFailure  CancelReplaceMode = "ALLOW_FAILURE"
)

// CancelRestrictions is the cancelRestrictions parameter of a cancel or cancel replace order
type CancelRestrictions string

// Cancel restrictions, which cancel an order only while it is new or only while it is partially filled
const (
	CancelRestrictionsOnlyNew             CancelRestrictions = "ONLY_NEW"
	CancelRestrictionsOnlyPartiallyFilled CancelRestrictions = "ONLY_PARTIALLY_FILLED"
)

// SymbolStatus is the symbolStatus parameter of the order book and ticker endpoints, which leaves out symbols in
// another status; a single symbol in another status is refused with -1220
type SymbolStatus string

// Symbol statuses the symbolStatus parameter takes. A symbol's exchange information can also report PRE_TRADING,
// POST_TRADING, END_OF_DAY and AUCTION_MATCH
const (
	SymbolStatusTrading SymbolStatus = "TRADING"
	SymbolStatusHalt    SymbolStatus = "HALT"
	SymbolStatusBreak   SymbolStatus = "BREAK"
)

// TickerType is the type parameter of the ticker endpoints
type TickerType string

// Ticker types: MINI leaves out the price change, weighted average price and best bid and ask fields
const (
	TickerTypeFull TickerType = "FULL"
	TickerTypeMini TickerType = "MINI"
)

// Permission is an account permission a symbol is traded with, the permissions parameter of the exchange information
type Permission string

// PermissionSpot is the permission of every Binance.US symbol; trading groups are TRD_GRP_ followed by their number
const PermissionSpot Permission = "SPOT"

// CreditLineTransferType is the transferType parameter of a credit line transfer and of the credit line transfer
// history
type CreditLineTransferType string

// Credit line transfer types: a transfer into or out of the credit line account
const (
	CreditLineTransferTypeTransferIn  CreditLineTransferType = "TRANSFER_IN"
	CreditLineTransferTypeTransferOut CreditLineTransferType = "TRANSFER_OUT"
)

// CreditLineAlertType is the alertType parameter of the credit line alert history
type CreditLineAlertType string

// Credit line alert types: a liquidation call or a margin call
const (
	CreditLineAlertTypeLiquidationCall CreditLineAlertType = "LIQUIDATION_CALL"
	CreditLineAlertTypeMarginCall      CreditLineAlertType = "MARGIN_CALL"
)

// OverTheCounterOrderStatus is the status of an over-the-counter or OCBS order
type OverTheCounterOrderStatus string

// Over-the-counter order statuses, which OCBS orders report too. An order still being processed, as every newly placed
// over-the-counter order is, reports PROCESS until it settles as SUCCESS or FAIL
const (
	OverTheCounterOrderStatusProcess OverTheCounterOrderStatus = "PROCESS"
	OverTheCounterOrderStatusSuccess OverTheCounterOrderStatus = "SUCCESS"
	OverTheCounterOrderStatusFail    OverTheCounterOrderStatus = "FAIL"
)

// PaymentMethod is the paymentMethod parameter of a fiat withdrawal and of the fiat history filters
type PaymentMethod string

// Payment methods a fiat withdrawal takes. The fiat histories also report others, such as ACH and WIRE_INTERNATIONAL
const (
	PaymentMethodBitGo             PaymentMethod = "BITGO"
	PaymentMethodIntrabankTransfer PaymentMethod = "INTRABANK_TRANSFER"
	PaymentMethodWire              PaymentMethod = "WIRE"
)

// ErrorResponse is the body Binance.US replies with when a request fails: a negative code and a message, or on some
// /sapi endpoints a success flag of false with a message, which the staking envelope names message
type ErrorResponse struct {
	Code            types.Number    `json:"code"`
	Message         string          `json:"msg"`
	EnvelopeMessage string          `json:"message"`
	Success         *bool           `json:"success"`
	Data            json.RawMessage `json:"data"` // Only some endpoints add details, such as the outcome of a cancel-replace
}

// RateLimitItemResponse holds a rate limit. Count, the usage counted against it, is sent by the unfilled order count
// query and the WebSocket API but not by the exchange information
type RateLimitItemResponse struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// ExchangeFilter holds an exchange filter, whose type determines which limit it sets
type ExchangeFilter struct {
	FilterType             string `json:"filterType"`
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
}

// SymbolFilter holds a symbol filter, whose type determines which of the fields it sets
type SymbolFilter struct {
	FilterType string `json:"filterType"`
	// PRICE_FILTER
	MinPrice types.Number `json:"minPrice"`
	MaxPrice types.Number `json:"maxPrice"`
	TickSize types.Number `json:"tickSize"`
	// PERCENT_PRICE
	MultiplierUp   types.Number `json:"multiplierUp"`
	MultiplierDown types.Number `json:"multiplierDown"`
	// AveragePriceMinutes is the window of the average price the percent price and notional filters use; zero means the
	// last price
	AveragePriceMinutes uint64 `json:"avgPriceMins"`
	// PERCENT_PRICE_BY_SIDE
	BidMultiplierUp   types.Number `json:"bidMultiplierUp"`
	BidMultiplierDown types.Number `json:"bidMultiplierDown"`
	AskMultiplierUp   types.Number `json:"askMultiplierUp"`
	AskMultiplierDown types.Number `json:"askMultiplierDown"`
	// LOT_SIZE and MARKET_LOT_SIZE
	MinQuantity types.Number `json:"minQty"`
	MaxQuantity types.Number `json:"maxQty"`
	StepSize    types.Number `json:"stepSize"`
	// MIN_NOTIONAL and NOTIONAL
	MinNotional      types.Number `json:"minNotional"`
	ApplyToMarket    bool         `json:"applyToMarket"`
	ApplyMinToMarket bool         `json:"applyMinToMarket"`
	MaxNotional      types.Number `json:"maxNotional"`
	ApplyMaxToMarket bool         `json:"applyMaxToMarket"`
	// ICEBERG_PARTS
	Limit uint64 `json:"limit"`
	// MAX_NUM_ORDERS, MAX_NUM_ALGO_ORDERS, MAX_NUM_ICEBERG_ORDERS, MAX_NUM_ORDER_LISTS and MAX_NUM_ORDER_AMENDS
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
	MaxNumberOrderLists    uint64 `json:"maxNumOrderLists"`
	MaxNumberOrderAmends   uint64 `json:"maxNumOrderAmends"`
	// MAX_POSITION
	MaxPosition types.Number `json:"maxPosition"`
	// TRAILING_DELTA, in basis points
	MinTrailingAboveDelta uint64 `json:"minTrailingAboveDelta"`
	MaxTrailingAboveDelta uint64 `json:"maxTrailingAboveDelta"`
	MinTrailingBelowDelta uint64 `json:"minTrailingBelowDelta"`
	MaxTrailingBelowDelta uint64 `json:"maxTrailingBelowDelta"`
}

// AssetFilter holds an account's asset filter. MAX_ASSET limits the quantity of an asset one order may transact: an
// order's quantity when the asset is the symbol's base asset, and its notional value when it is the quote asset
type AssetFilter struct {
	FilterType string        `json:"filterType"`
	Asset      currency.Code `json:"asset"`
	Limit      types.Number  `json:"limit"`
}
