package kucoin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceUTAOrderRequest holds the parameters of Place Order, which the unified trading account websocket Add Order
// operation takes as well
type PlaceUTAOrderRequest struct {
	// TradeType is SPOT, FUTURES or MARGIN
	TradeType string
	// ClientOrderID is required for futures and margin orders, which KuCoin rejects without one (error 110104); it is
	// at most 40 letters, digits, underscores and dashes
	ClientOrderID string
	Symbol        string
	Side          string
	// PositionSide is BOTH, LONG or SHORT; unified accounts do not support it yet
	PositionSide string
	OrderType    string
	// Size is in the unit SizeUnit names: BASECCY or QUOTECCY for spot market orders, BASECCY for spot limit orders,
	// and BASECCY or UNIT (contracts) for futures
	Size     float64
	SizeUnit string
	Price    float64
	// ReduceOnly applies to futures only
	ReduceOnly bool
	// MarginMode is CROSS or ISOLATED for futures and CROSS for margin; it does not apply to spot
	MarginMode          string
	SelfTradePrevention string
	TimeInForce         string
	// CancelAfter cancels a GTT order this many seconds after it reaches the matching engine
	CancelAfter uint64
	PostOnly    bool
	Tags        string
	// TriggerDirection (UP or DOWN), TriggerPrice and TriggerPriceType make the order conditional: KuCoin places it
	// once the price crosses TriggerPrice. A conditional order cannot carry take profit or stop loss trigger prices
	TriggerDirection           string
	TriggerPrice               float64
	TriggerPriceType           string
	TakeProfitTriggerPrice     float64
	TakeProfitTriggerPriceType string
	StopLossTriggerPrice       float64
	StopLossTriggerPriceType   string
	// CloseOrder closes the symbol's position: KuCoin ignores Side and Size, which the docs still require, and cancels
	// the pending close orders
	CloseOrder bool
}

// Validate checks the parameters Place Order requires and the conditional order rules the docs set
func (r *PlaceUTAOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.TradeType == "":
		return errTradeTypeMissing
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case r.Side == "":
		return order.ErrSideIsInvalid
	case r.OrderType == "":
		return order.ErrTypeIsInvalid
	case r.Size <= 0:
		return order.ErrAmountIsInvalid
	case r.SizeUnit == "":
		return errSizeUnitRequired
	case r.ClientOrderID == "" && (r.TradeType == "FUTURES" || r.TradeType == "MARGIN"):
		return order.ErrClientOrderIDMustBeSet
	case (r.TriggerDirection == "") != (r.TriggerPrice == 0):
		return errTriggerDirectionAndPriceRequired
	case r.TriggerDirection != "" && (r.TakeProfitTriggerPrice != 0 || r.StopLossTriggerPrice != 0):
		return errConditionalOrderWithTPSL
	}
	return nil
}

// Payload validates the request and returns the Place Order body, which the REST and websocket order endpoints share
func (r *PlaceUTAOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["tradeType"] = r.TradeType
	body["symbol"] = r.Symbol
	body["side"] = r.Side
	body["orderType"] = r.OrderType
	body["size"] = strconv.FormatFloat(r.Size, 'f', -1, 64)
	body["sizeUnit"] = r.SizeUnit
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
	}
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	if r.SelfTradePrevention != "" {
		body["stp"] = r.SelfTradePrevention
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.CancelAfter != 0 {
		body["cancelAfter"] = r.CancelAfter
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.Tags != "" {
		body["tags"] = r.Tags
	}
	if r.TriggerDirection != "" {
		body["triggerDirection"] = r.TriggerDirection
	}
	if r.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(r.TriggerPrice, 'f', -1, 64)
	}
	if r.TriggerPriceType != "" {
		body["triggerPriceType"] = r.TriggerPriceType
	}
	if r.TakeProfitTriggerPrice != 0 {
		body["tpTriggerPrice"] = strconv.FormatFloat(r.TakeProfitTriggerPrice, 'f', -1, 64)
	}
	if r.TakeProfitTriggerPriceType != "" {
		body["tpTriggerPriceType"] = r.TakeProfitTriggerPriceType
	}
	if r.StopLossTriggerPrice != 0 {
		body["slTriggerPrice"] = strconv.FormatFloat(r.StopLossTriggerPrice, 'f', -1, 64)
	}
	if r.StopLossTriggerPriceType != "" {
		body["slTriggerPriceType"] = r.StopLossTriggerPriceType
	}
	if r.CloseOrder {
		body["closeOrder"] = true
	}
	return body, nil
}

// UTAOrderResponse holds the order a Place Order or Cancel Order request acted on
type UTAOrderResponse struct {
	TradeType     string `json:"tradeType"`
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	// Timestamp is when KuCoin completed a REST request, and, in reply to a websocket Add Order or Cancel Order, when the
	// UTA risk engine received the order
	Timestamp types.Time `json:"ts"`
}

// AmendUTAOrderRequest holds the parameters of Amend Order, which the unified trading account websocket Amend Order
// operation takes as well. KuCoin amends futures orders only so far
type AmendUTAOrderRequest struct {
	Symbol        string
	NewSize       float64
	OrderID       string
	ClientOrderID string
	NewPrice      float64
	// SizeUnit is BASECCY or UNIT (contracts)
	SizeUnit string
	// CancelOnFail cancels the original order when the amendment fails
	CancelOnFail               bool
	StopLossTriggerPrice       float64
	TakeProfitTriggerPrice     float64
	StopLossTriggerPriceType   string
	TakeProfitTriggerPriceType string
}

// Validate checks the parameters Amend Order requires
func (r *AmendUTAOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case r.OrderID == "" && r.ClientOrderID == "":
		return order.ErrOrderIDNotSet
	}
	return nil
}

// Payload validates the request and returns the Amend Order body, which the REST and websocket amend endpoints share.
// Amend Order types its sizes and prices as JSON numbers, where Place Order takes strings
func (r *AmendUTAOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	if r.NewSize != 0 {
		body["newSize"] = r.NewSize
	}
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
	}
	if r.NewPrice != 0 {
		body["newPrice"] = r.NewPrice
	}
	if r.SizeUnit != "" {
		body["sizeUnit"] = r.SizeUnit
	}
	if r.CancelOnFail {
		body["cxlOnFail"] = true
	}
	if r.StopLossTriggerPrice != 0 {
		body["slTriggerPrice"] = r.StopLossTriggerPrice
	}
	if r.TakeProfitTriggerPrice != 0 {
		body["tpTriggerPrice"] = r.TakeProfitTriggerPrice
	}
	if r.StopLossTriggerPriceType != "" {
		body["slTriggerPriceType"] = r.StopLossTriggerPriceType
	}
	if r.TakeProfitTriggerPriceType != "" {
		body["tpTriggerPriceType"] = r.TakeProfitTriggerPriceType
	}
	return body, nil
}

// AmendUTAOrderResponse holds the order an Amend Order request amended. It only means KuCoin accepted the request:
// the websocket order channel pushes the outcome
type AmendUTAOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}

// CancelUTAOrderRequest holds the parameters of Cancel Order, which the unified trading account websocket Cancel Order
// operation takes as well. When both IDs are set, the REST Cancel Order uses OrderID while the websocket one uses
// ClientOrderID
type CancelUTAOrderRequest struct {
	TradeType     string
	Symbol        string
	OrderID       string
	ClientOrderID string
}

// Validate checks the parameters Cancel Order requires
func (r *CancelUTAOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.TradeType == "":
		return errTradeTypeMissing
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case r.OrderID == "" && r.ClientOrderID == "":
		return order.ErrOrderIDNotSet
	}
	return nil
}

// Payload validates the request and returns the Cancel Order body, which the REST and websocket cancel endpoints share
func (r *CancelUTAOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["tradeType"] = r.TradeType
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
	}
	return body, nil
}

// BatchCancelUTAOrdersByIDRequest holds the parameters of Batch Cancel Orders By ID: up to 20 orders of one trade type
type BatchCancelUTAOrdersByIDRequest struct {
	TradeType       string
	CancelOrderList []BatchCancelUTAOrderItem
}

// BatchCancelUTAOrderItem holds one order of a Batch Cancel Orders By ID request. KuCoin uses OrderID when both IDs
// are set
type BatchCancelUTAOrderItem struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}

// BatchCancelUTAOrdersByIDResponse holds the outcome of each order of a Batch Cancel Orders By ID request
type BatchCancelUTAOrdersByIDResponse struct {
	TradeType string                      `json:"tradeType"`
	Items     []BatchCancelUTAOrderResult `json:"items"`
}

// BatchCancelUTAOrderResult is the outcome of cancelling one order of a Batch Cancel Orders By ID request: a Code
// other than 200000 means KuCoin did not cancel the order, and Message says why
type BatchCancelUTAOrderResult struct {
	Code          string `json:"code"`
	Message       string `json:"msg"`
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	// Timestamp is when KuCoin completed the cancellation request
	Timestamp types.Time `json:"ts"`
}

// BatchCancelUTAOrdersBySymbolRequest holds the parameters of Batch Cancel Orders By Symbol
type BatchCancelUTAOrdersBySymbolRequest struct {
	TradeType string
	Symbol    string
	// MarginMode is CROSS (the default) or ISOLATED
	MarginMode string
	// OrderFilter is NORMAL for limit and market orders or ADVANCED for conditional orders
	OrderFilter string
}

// BatchCancelUTAOrdersBySymbolResponse holds the orders a Batch Cancel Orders By Symbol request cancelled, at most 100
type BatchCancelUTAOrdersBySymbolResponse struct {
	TradeType string              `json:"tradeType"`
	Timestamp types.Time          `json:"ts"`
	Items     []UTACancelledOrder `json:"items"`
}

// UTACancelledOrder holds an order a batch cancellation cancelled
type UTACancelledOrder struct {
	OrderID string `json:"orderId"`
}

// UTAOrderDetailsRequest holds the parameters of Get Order Details. KuCoin uses OrderID when both IDs are set, and
// only finds cancelled orders without fills and triggered or cancelled conditional orders for 2 hours, and orders with
// fills for 168 hours
type UTAOrderDetailsRequest struct {
	TradeType     string
	Symbol        string
	OrderID       string
	ClientOrderID string
}

// UTAOrderDetailsResponse holds an order of the unified trading account
type UTAOrderDetailsResponse struct {
	TradeType string `json:"tradeType"`
	UTAOrder
	// TakeProfitOrderPrice and StopLossOrderPrice are the prices of the take profit and stop loss orders, which execute
	// at the market price when empty
	TakeProfitOrderPrice types.Number `json:"tpOrderPrice"`
	StopLossOrderPrice   types.Number `json:"slOrderPrice"`
}

// UTAOrder holds an order of the unified trading account
type UTAOrder struct {
	OrderID string `json:"orderId"`
	// ClientOrderID is empty on an order a triggered spot or futures conditional order placed, which does not inherit it
	ClientOrderID string `json:"clientOid"`
	// Status is 0 not triggered (futures conditional orders), 1 triggered, 2 live, 3 filled, 4 partially filled,
	// 5 cancelled without fills or 6 partially filled and then cancelled
	Status     uint64       `json:"status"`
	FilledSize types.Number `json:"filledSize"`
	// AveragePrice is rounded to the tick size for futures
	AveragePrice types.Number  `json:"avgPrice"`
	Fee          types.Number  `json:"fee"`
	FeeCurrency  currency.Code `json:"feeCurrency"`
	Tax          types.Number  `json:"tax"`
	// TradeID is the order's last trade, which KuCoin may no longer return for spot trades older than 7 days
	TradeID      string       `json:"tradeId"`
	Symbol       string       `json:"symbol"`
	Side         string       `json:"side"`
	PositionSide string       `json:"positionSide"`
	OrderType    string       `json:"orderType"`
	Size         types.Number `json:"size"`
	// SizeUnit is BASECCY, QUOTECCY or UNIT (contracts)
	SizeUnit string       `json:"sizeUnit"`
	Price    types.Number `json:"price"`
	// ReduceOnly applies to futures orders only; KuCoin sends an empty string for other orders
	ReduceOnly          UTABoolean `json:"reduceOnly"`
	MarginMode          string     `json:"marginMode"`
	SelfTradePrevention string     `json:"stp"`
	TimeInForce         string     `json:"timeInForce"`
	// CancelReason is a cancellation reason code, such as USER or ORDER_EXPIRED
	CancelReason string `json:"cancelReason"`
	// CancelAfter is the number of seconds after which KuCoin cancels a GTT order
	CancelAfter                uint64       `json:"cancelAfter"`
	CancelSize                 types.Number `json:"cancelSize"`
	TriggerDirection           string       `json:"triggerDirection"`
	TriggerPrice               types.Number `json:"triggerPrice"`
	TriggerPriceType           string       `json:"triggerPriceType"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPrice"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPriceType"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPrice"`
	StopLossTriggerPriceType   string       `json:"slTriggerPriceType"`
	PostOnly                   bool         `json:"postOnly"`
	Tags                       string       `json:"tags"`
	// TriggerOrderID is the conditional order that placed this order
	TriggerOrderID string     `json:"triggerOrderId"`
	OrderTime      types.Time `json:"orderTime"`
	UpdatedTime    types.Time `json:"updatedTime"`
	BrokerTag      string     `json:"brokerTag"`
}

// UTABoolean is a boolean which the UTA order endpoints send as a JSON boolean where it applies, and as an empty string
// where it does not
type UTABoolean bool

// UnmarshalJSON decodes a JSON boolean; an empty string and null decode as false
func (b *UTABoolean) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true":
		*b = true
	case "false", `""`, "null":
		*b = false
	default:
		return fmt.Errorf("%w: boolean %s", common.ErrMalformedData, data)
	}
	return nil
}

// UTAOpenOrderListRequest holds the parameters of Get Open Order List
type UTAOpenOrderListRequest struct {
	TradeType string
	Symbol    string
	// OrderFilter is NORMAL (the default) for limit and market orders or ADVANCED for conditional orders
	OrderFilter string
	PageNumber  uint64
	PageSize    uint64
}

// UTAOpenOrderListResponse holds a page of the unified trading account's open orders, newest first
type UTAOpenOrderListResponse struct {
	PageNumber  uint64     `json:"pageNumber"`
	PageSize    uint64     `json:"pageSize"`
	TotalNumber uint64     `json:"totalNum"`
	TotalPage   uint64     `json:"totalPage"`
	TradeType   string     `json:"tradeType"`
	Items       []UTAOrder `json:"items"`
}

// UTAOrderHistoryRequest holds the parameters of Get Order History. Without a time window KuCoin returns the last 7
// days, and it only finds cancelled orders without fills and triggered or cancelled conditional orders for 2 hours,
// and orders with fills for 168 hours
type UTAOrderHistoryRequest struct {
	TradeType string
	Symbol    string
	Side      string
	// OrderFilter is NORMAL for limit and market orders or ADVANCED for conditional orders
	OrderFilter string
	StartAt     time.Time
	EndAt       time.Time
	// LastID is the previous page's LastID, to fetch the page after it
	LastID   uint64
	PageSize uint64
}

// UTAOrderHistoryResponse holds a page of the unified trading account's order history
type UTAOrderHistoryResponse struct {
	// LastID is the cursor of the next page
	LastID    uint64     `json:"lastId"`
	TradeType string     `json:"tradeType"`
	Items     []UTAOrder `json:"items"`
}

// UTATradeHistoryRequest holds the parameters of Get Trade History. Without a time window KuCoin returns the last 7
// days
type UTATradeHistoryRequest struct {
	TradeType string
	Symbol    string
	OrderID   string
	Side      string
	StartAt   time.Time
	EndAt     time.Time
	// LastID is the previous page's LastID, to fetch the page after it
	LastID uint64
	// PageSize is 50 by default and 200 at most
	PageSize uint64
	// FillType is NORMAL, ADL, LIQUID, SETTLEMENT or RECONCILIATION
	FillType string
}

// UTATradeHistoryResponse holds a page of the unified trading account's fills
type UTATradeHistoryResponse struct {
	// LastID is the cursor of the next page
	LastID    uint64    `json:"lastId"`
	TradeType string    `json:"tradeType"`
	Items     []UTAFill `json:"items"`
}

// UTAFill holds a fill of a unified trading account order
type UTAFill struct {
	TradeType     string `json:"tradeType"`
	Symbol        string `json:"symbol"`
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	OrderType     string `json:"orderType"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	// FillType is NORMAL, ADL, LIQUID, SETTLEMENT or RECONCILIATION
	FillType      string        `json:"fillType"`
	TradeID       string        `json:"tradeId"`
	Size          types.Number  `json:"size"`
	Value         types.Number  `json:"value"`
	Price         types.Number  `json:"price"`
	ExecutionTime types.Time    `json:"executionTime"`
	Fee           types.Number  `json:"fee"`
	FeeCurrency   currency.Code `json:"feeCurrency"`
	// LiquidityRole is MAKER or TAKER, and empty for ADL fills
	LiquidityRole string       `json:"liquidityRole"`
	MarginMode    string       `json:"marginMode"`
	Tax           types.Number `json:"tax"`
	// FillPNL is the profit or loss the fill closed; KuCoin leaves it empty for opening and non-futures fills and for
	// fills before 2026-09-05 (UTC-8)
	FillPNL types.Number `json:"fillPnl"`
}

// UTAMaxOrderQuantityRequest holds the parameters of Get Max Order Quantity
type UTAMaxOrderQuantityRequest struct {
	Symbol    string
	TradeType string
	// Price prices the quantities; KuCoin falls back to the market price when it is 0
	Price float64
}

// UTAMaxOrderQuantityResponse holds the most a unified trading account can buy and sell of a symbol
type UTAMaxOrderQuantityResponse struct {
	Symbol      string       `json:"symbol"`
	MaxBuySize  types.Number `json:"maxBuySize"`
	MaxSellSize types.Number `json:"maxSellSize"`
}
