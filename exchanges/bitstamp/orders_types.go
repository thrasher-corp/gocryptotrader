package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Order subtypes used by derivatives markets
const (
	OrderSubtypeLimit                   = "LIMIT"
	OrderSubtypeMarket                  = "MARKET"
	OrderSubtypeInstant                 = "INSTANT"
	OrderSubtypeCash                    = "CASH"
	OrderSubtypeStopMarket              = "STOP_MARKET"
	OrderSubtypeStopLimit               = "STOP_LIMIT"
	OrderSubtypeStopLoss                = "STOP_LOSS"
	OrderSubtypeTakeProfit              = "TAKE_PROFIT"
	OrderSubtypeStopLossLimit           = "STOP_LOSS_LIMIT"
	OrderSubtypeTakeProfitLimit         = "TAKE_PROFIT_LIMIT"
	OrderSubtypeTrailingStopLoss        = "TRAILING_STOP_LOSS"
	OrderSubtypeTrailingTakeProfit      = "TRAILING_TAKE_PROFIT"
	OrderSubtypeTrailingStopLossLimit   = "TRAILING_STOP_LOSS_LIMIT"
	OrderSubtypeTrailingTakeProfitLimit = "TRAILING_TAKE_PROFIT_LIMIT"
)

// Stop order trigger price types
const (
	TriggerLastTradedPrice = "LAST_TRADED_PRICE"
	TriggerIndexPrice      = "INDEX_PRICE"
	TriggerMarkPrice       = "MARK_PRICE"
)

// Order event sources used for websocket gap recovery
const (
	OrderSourceOrderbook           = "orderbook"
	OrderSourceStopOrder           = "stop_order"
	OrderSourceTokenisedSecurities = "tokenized_securities" //nolint:gosec // G101 false positive; an order source rather than a credential
)

// Order statuses returned by the order status endpoint
const (
	OrderStatusOpen      = "Open"
	OrderStatusFinished  = "Finished"
	OrderStatusExpired   = "Expired"
	OrderStatusCancelled = "Canceled"
)

// Statuses of cancelled orders
const (
	CancelStatusCancelled = "Canceled"
	CancelStatusPending   = "Cancel pending"
)

// OrderStatusRequest holds the parameters for fetching an order's status
// Either OrderID or ClientOrderID must be supplied
type OrderStatusRequest struct {
	OrderID          uint64
	ClientOrderID    string
	OmitTransactions bool
}

// OrderStatusResponse holds the status of an order
type OrderStatusResponse struct {
	ID              uint64             `json:"id"`
	DateTime        types.DateTime     `json:"datetime"`
	Side            OrderSide          `json:"type"`
	Subtype         string             `json:"subtype"`
	Status          string             `json:"status"`
	Market          string             `json:"market"`
	Transactions    []OrderTransaction `json:"transactions"`
	AmountRemaining types.Number       `json:"amount_remaining"`
	ClientOrderID   string             `json:"client_order_id"`
	MarginMode      string             `json:"margin_mode"`
	Leverage        types.Number       `json:"leverage"`
	StopPrice       types.Number       `json:"stop_price"`
	Trigger         string             `json:"trigger"`
	ActivationPrice types.Number       `json:"activation_price"`
	TrailingDelta   uint64             `json:"trailing_delta"`
}

// OrderTransaction holds a transaction which executed part of an order
// Amounts holds the signed amount of each currency involved in the transaction, keyed by currency
type OrderTransaction struct {
	TradeID  uint64
	Price    float64
	Fee      float64
	DateTime time.Time
	Type     TransactionType
	Amounts  map[currency.Code]float64
}

// OrderEventsRequest holds the parameters for fetching order events for websocket gap recovery
type OrderEventsRequest struct {
	Pair    currency.Pair
	SinceID string
	UntilID string
	// OrderSource is only used when fetching account order events
	OrderSource string
}

// OrderEventResponse holds an order event used for websocket gap recovery
type OrderEventResponse struct {
	Event       string `json:"event"`
	EventID     string `json:"event_id"`
	OrderSource string `json:"order_source"`
	// TradeAccountID is only populated for account order events and is 0 for the main account
	TradeAccountID uint64         `json:"trade_account_id"`
	Data           OrderEventData `json:"data"`
}

// OrderEventData holds the order details of an order event, from the websocket order channels or their REST recovery
// RemainingAmount is the amount left to execute, and AmountTraded is the amount executed by the event rather than in total
type OrderEventData struct {
	ID              uint64       `json:"id"`
	Side            OrderSide    `json:"order_type"`
	OrderSubtype    uint8        `json:"order_subtype"`
	DateTime        types.Time   `json:"datetime"`
	Microtimestamp  types.Time   `json:"microtimestamp"`
	RemainingAmount types.Number `json:"amount_str"`
	AmountTraded    types.Number `json:"amount_traded"`
	AmountAtCreate  types.Number `json:"amount_at_create"`
	Price           types.Number `json:"price_str"`
	IsLiquidation   bool         `json:"is_liquidation"`
	// The following fields are only populated for account order events
	TradeAccountID  uint64       `json:"trade_account_id"`
	ClientOrderID   string       `json:"client_order_id"`
	OriginalOrderID uint64       `json:"orig_order_id"`
	ReduceOnly      bool         `json:"reduce_only"`
	StopPrice       types.Number `json:"stop_price"`
	ActivationPrice types.Number `json:"activation_price"`
	TrailingDelta   types.Number `json:"trailing_delta"`
}

// OpenOrderResponse holds an open order
type OpenOrderResponse struct {
	ID              uint64         `json:"id,string"`
	DateTime        types.DateTime `json:"datetime"`
	Side            OrderSide      `json:"type"`
	Subtype         string         `json:"subtype"`
	Price           types.Number   `json:"price"`
	RemainingAmount types.Number   `json:"amount"`
	AmountAtCreate  types.Number   `json:"amount_at_create"`
	Market          string         `json:"market"`
	LimitPrice      types.Number   `json:"limit_price"`
	ClientOrderID   string         `json:"client_order_id"`
	MarginMode      string         `json:"margin_mode"`
	Leverage        types.Number   `json:"leverage"`
	ReservedMargin  types.Number   `json:"reserved_margin"`
	StopPrice       types.Number   `json:"stop_price"`
	Trigger         string         `json:"trigger"`
	ActivationPrice types.Number   `json:"activation_price"`
	TrailingDelta   uint64         `json:"trailing_delta"`
	ReduceOnly      bool           `json:"reduce_only"`
}

// LimitOrderRequest holds the parameters for placing a limit order
// MarginMode and Leverage are required for derivatives markets. At most one of DailyOrder, ImmediateOrCancel,
// FillOrKill, MakerOrCancel and GoodTillDate may be set, and none of them may be combined with LimitPrice
type LimitOrderRequest struct {
	Pair              currency.Pair
	Side              order.Side
	Amount            float64
	Price             float64
	LimitPrice        float64
	DailyOrder        bool
	ImmediateOrCancel bool
	FillOrKill        bool
	MakerOrCancel     bool
	GoodTillDate      bool
	ExpireTime        time.Time
	ClientOrderID     string
	Subtype           string
	MarginMode        string
	Leverage          float64
	StopPrice         float64
	Trigger           string
	ActivationPrice   float64
	TrailingDelta     uint64
	ReduceOnly        bool
}

// InstantOrderRequest holds the parameters for placing an instant order
// Buy amounts are in the counter currency; sell amounts are in the base currency unless AmountInCounter is set
// MarginMode and Leverage are required for derivatives buy orders, and ReduceOnly is only supported by sell orders
// whose amount is in the base currency
type InstantOrderRequest struct {
	Pair            currency.Pair
	Side            order.Side
	Amount          float64
	AmountInCounter bool
	ClientOrderID   string
	MarginMode      string
	Leverage        float64
	ReduceOnly      bool
}

// MarketOrderRequest holds the parameters for placing a market order
// MarginMode and Leverage are required for derivatives markets
type MarketOrderRequest struct {
	Pair            currency.Pair
	Side            order.Side
	Amount          float64
	ClientOrderID   string
	Subtype         string
	MarginMode      string
	Leverage        float64
	StopPrice       float64
	Trigger         string
	ActivationPrice float64
	TrailingDelta   uint64
	ReduceOnly      bool
}

// OrderResponse holds the details of a newly placed order
type OrderResponse struct {
	ID              uint64         `json:"id,string"`
	Subtype         string         `json:"subtype"`
	Market          string         `json:"market"`
	DateTime        types.DateTime `json:"datetime"`
	Side            OrderSide      `json:"type"`
	Price           types.Number   `json:"price"`
	Amount          types.Number   `json:"amount"`
	ClientOrderID   string         `json:"client_order_id"`
	MarginMode      string         `json:"margin_mode"`
	Leverage        types.Number   `json:"leverage"`
	StopPrice       types.Number   `json:"stop_price"`
	Trigger         string         `json:"trigger"`
	ActivationPrice types.Number   `json:"activation_price"`
	TrailingDelta   uint64         `json:"trailing_delta"`
}

// CancelOrderRequest holds the parameters for cancelling an order
// Either OrderID or ClientOrderID must be supplied
type CancelOrderRequest struct {
	OrderID       uint64
	ClientOrderID string
}

// CancelOrderResponse holds the details of a cancelled order
type CancelOrderResponse struct {
	ID     uint64       `json:"id"`
	Amount types.Number `json:"amount"`
	Price  types.Number `json:"price"`
	Side   OrderSide    `json:"type"`
	Market string       `json:"market"`
	Status string       `json:"status"`
}

// CancelAllOrdersResponse holds the result of cancelling all open orders
type CancelAllOrdersResponse struct {
	Cancelled []CancelledOrder `json:"canceled"`
	Success   bool             `json:"success"`
}

// CancelledOrder holds an order cancelled by a cancel all orders request
type CancelledOrder struct {
	ID     uint64       `json:"id"`
	Amount types.Number `json:"amount"`
	Price  types.Number `json:"price"`
	Side   OrderSide    `json:"type"`
	Market string       `json:"market"`
}

// ReplaceOrderRequest holds the parameters for replacing an order
// Either OrderID or OriginalClientOrderID must be supplied
type ReplaceOrderRequest struct {
	OrderID               uint64
	OriginalClientOrderID string
	ClientOrderID         string
	Amount                float64
	Price                 float64
}

// ReplaceOrderResponse holds the details of a replacement order
type ReplaceOrderResponse struct {
	OrderID               uint64       `json:"order_id"`
	Side                  OrderSide    `json:"order_type"`
	Market                string       `json:"market"`
	Amount                types.Number `json:"amount"`
	Price                 types.Number `json:"price"`
	DateTime              DateTime     `json:"datetime"`
	OriginalOrderID       uint64       `json:"orig_order_id"`
	OriginalClientOrderID string       `json:"orig_client_order_id"`
	Status                string       `json:"status"`
}

// TradingMarketResponse holds a market which can be traded by the account
type TradingMarketResponse struct {
	Name      string `json:"name"`
	URLSymbol string `json:"url_symbol"`
}

// MaxOrderAmountRequest holds the parameters for estimating the maximum order amount for a derivatives market
// AdditionalCollateral holds collateral amounts to include in the estimate as if they were already posted
type MaxOrderAmountRequest struct {
	Pair                 currency.Pair
	MarginMode           string
	Leverage             float64
	OrderType            string
	Side                 order.Side
	Price                float64
	StopPrice            float64
	ActivationPrice      float64
	TrailingDelta        uint64
	AdditionalCollateral map[currency.Code]float64
}

// MaxOrderAmountResponse holds the maximum order amount and value for a derivatives market
type MaxOrderAmountResponse struct {
	MaximumOrderAmount         types.Number  `json:"maximum_order_amount"`
	MaximumOrderValue          types.Number  `json:"maximum_order_value"`
	MaximumOrderAmountCurrency currency.Code `json:"maximum_order_amount_currency"`
	MaximumOrderValueCurrency  currency.Code `json:"maximum_order_value_currency"`
}

// derivativesOrderParams holds the optional order parameters shared by limit, instant and market orders
type derivativesOrderParams struct {
	ClientOrderID   string
	Subtype         string
	MarginMode      string
	Leverage        float64
	StopPrice       float64
	Trigger         string
	ActivationPrice float64
	TrailingDelta   uint64
	ReduceOnly      bool
}

// maxOrderAmountBody is the JSON body of a maximum order amount request
type maxOrderAmountBody struct {
	Market               string            `json:"market"`
	MarginMode           string            `json:"margin_mode"`
	Leverage             float64           `json:"leverage"`
	OrderType            string            `json:"order_type"`
	Side                 string            `json:"side"`
	Price                float64           `json:"price,omitempty"`
	StopPrice            float64           `json:"stop_price,omitempty"`
	ActivationPrice      float64           `json:"activation_price,omitempty"`
	TrailingDelta        uint64            `json:"trailing_delta,omitempty"`
	AdditionalCollateral map[string]string `json:"additional_collateral,omitempty"`
}
