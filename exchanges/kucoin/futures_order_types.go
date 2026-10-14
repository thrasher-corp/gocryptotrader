package kucoin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesOrderRequest holds the parameters of Add Order, which Add Order Test and Batch Add Orders take as well and
// the futures.order and futures.multi_order websocket operations send
type FuturesOrderRequest struct {
	// ClientOrderID is at most 40 numbers, letters, underscores and hyphens
	ClientOrderID string
	// Side is buy or sell; leave it empty with CloseOrder, which takes the side from the position
	Side   string
	Symbol string
	// Leverage applies to isolated margin orders; closing orders and cross margin orders leave it unset
	Leverage uint64
	// Type is limit, which KuCoin assumes when it is empty, or market
	Type   string
	Remark string
	// Stop is down or up, and makes the order a stop order that needs StopPriceType and StopPrice
	Stop string
	// StopPriceType is TP for the last traded price, MP for the mark price or IP for the index price
	StopPriceType string
	StopPrice     float64
	ReduceOnly    bool
	// CloseOrder closes the whole position, taking the side and size from it, and takes precedence over ReduceOnly
	CloseOrder bool
	ForceHold  bool
	// SelfTradePrevention is CN, CO or CB
	SelfTradePrevention string
	// MarginMode is ISOLATED, the default, or CROSS
	MarginMode string
	// Price is required for limit orders
	Price float64
	// Size is in lots. Size, Quantity and ValueQuantity are alternatives: set one of them
	Size uint64
	// TimeInForce is GTC, the default, or IOC; Add Order also takes FOK and RPI
	TimeInForce string
	PostOnly    bool
	// Quantity is in the base currency, a multiple of the contract multiplier; coin-margined contracts take Size only
	Quantity float64
	// ValueQuantity is in the quote currency; coin-margined contracts take Size only
	ValueQuantity float64
	// PositionSide is BOTH, the one-way mode default, LONG or SHORT; hedge mode requires LONG or SHORT
	PositionSide string
}

// Validate checks the parameters Add Order requires
func (r *FuturesOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.ClientOrderID == "":
		return order.ErrClientOrderIDMustBeSet
	case r.Side == "" && !r.CloseOrder:
		return order.ErrSideIsInvalid
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case r.Stop != "" && r.StopPriceType == "":
		return errInvalidStopPriceType
	case r.Stop != "" && r.StopPrice <= 0:
		return fmt.Errorf("%w: a stop order needs a stop price", limits.ErrPriceBelowMin)
	case (r.Type == "" || r.Type == order.Limit.Lower()) && r.Price <= 0:
		return fmt.Errorf("%w: a limit order needs a price", limits.ErrPriceBelowMin)
	case countSet(r.Size != 0, r.Quantity != 0, r.ValueQuantity != 0) > 1:
		return errSizeQuantityValueQuantityExclusive
	}
	return nil
}

// Payload validates the request and returns the Add Order body, which the REST endpoints and websocket order
// operations share
func (r *FuturesOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["symbol"] = r.Symbol
	if r.Side != "" {
		body["side"] = r.Side
	}
	if r.Leverage != 0 {
		body["leverage"] = r.Leverage
	}
	if r.Type != "" {
		body["type"] = r.Type
	}
	if r.Remark != "" {
		body["remark"] = r.Remark
	}
	if r.Stop != "" {
		body["stop"] = r.Stop
	}
	if r.StopPriceType != "" {
		body["stopPriceType"] = r.StopPriceType
	}
	if r.StopPrice != 0 {
		body["stopPrice"] = strconv.FormatFloat(r.StopPrice, 'f', -1, 64)
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOrder {
		body["closeOrder"] = true
	}
	if r.ForceHold {
		body["forceHold"] = true
	}
	if r.SelfTradePrevention != "" {
		body["stp"] = r.SelfTradePrevention
	}
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = r.Size
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.Quantity != 0 {
		body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	}
	if r.ValueQuantity != 0 {
		body["valueQty"] = strconv.FormatFloat(r.ValueQuantity, 'f', -1, 64)
	}
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	return body, nil
}

// FuturesOrderResponse holds the IDs of an order KuCoin accepted. Placing is asynchronous, so the order's status
// follows on the order endpoints and websocket channels
type FuturesOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}

// FuturesBatchOrderResult is the outcome of one order of a Batch Add Orders request. Code 200000 means KuCoin accepted
// the order; any other Code means it rejected it, for the reason in Message
type FuturesBatchOrderResult struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	Symbol        string `json:"symbol"`
	Code          string `json:"code"`
	Message       string `json:"msg"`
}

// FuturesTakeProfitStopLossOrderRequest holds the parameters of Add Take Profit And Stop Loss Order: an Add Order
// with take profit and stop loss trigger prices in place of a single stop
type FuturesTakeProfitStopLossOrderRequest struct {
	// ClientOrderID is at most 40 numbers, letters, underscores and hyphens
	ClientOrderID string
	// Side is buy or sell; leave it empty with CloseOrder, which takes the side from the position
	Side   string
	Symbol string
	// Leverage applies to isolated margin orders; closing orders and cross margin orders leave it unset
	Leverage uint64
	// Type is limit, which KuCoin assumes when it is empty, or market
	Type   string
	Remark string
	// StopPriceType is TP for the last traded price, MP for the mark price or IP for the index price
	StopPriceType string
	ReduceOnly    bool
	// CloseOrder closes the whole position, taking the side and size from it, and takes precedence over ReduceOnly
	CloseOrder bool
	ForceHold  bool
	// SelfTradePrevention is CN, CO or CB
	SelfTradePrevention string
	// MarginMode is ISOLATED, the default, or CROSS
	MarginMode string
	// Price is required for limit orders
	Price float64
	// Size is in lots. Size, Quantity and ValueQuantity are alternatives: set one of them
	Size uint64
	// TimeInForce is GTC, the default, or IOC
	TimeInForce string
	PostOnly    bool
	// TriggerStopUpPrice is the take profit price
	TriggerStopUpPrice float64
	// TriggerStopDownPrice is the stop loss price
	TriggerStopDownPrice float64
	// Quantity is in the base currency, a multiple of the contract multiplier; coin-margined contracts take Size only
	Quantity float64
	// ValueQuantity is in the quote currency; coin-margined contracts take Size only
	ValueQuantity float64
	// PositionSide is BOTH, the one-way mode default, LONG or SHORT; hedge mode requires LONG or SHORT
	PositionSide string
}

// Validate checks the parameters Add Take Profit And Stop Loss Order requires
func (r *FuturesTakeProfitStopLossOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.ClientOrderID == "":
		return order.ErrClientOrderIDMustBeSet
	case r.Side == "" && !r.CloseOrder:
		return order.ErrSideIsInvalid
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case (r.Type == "" || r.Type == order.Limit.Lower()) && r.Price <= 0:
		return fmt.Errorf("%w: a limit order needs a price", limits.ErrPriceBelowMin)
	case countSet(r.Size != 0, r.Quantity != 0, r.ValueQuantity != 0) > 1:
		return errSizeQuantityValueQuantityExclusive
	}
	return nil
}

// Payload validates the request and returns the Add Take Profit And Stop Loss Order body
func (r *FuturesTakeProfitStopLossOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["symbol"] = r.Symbol
	if r.Side != "" {
		body["side"] = r.Side
	}
	if r.Leverage != 0 {
		body["leverage"] = r.Leverage
	}
	if r.Type != "" {
		body["type"] = r.Type
	}
	if r.Remark != "" {
		body["remark"] = r.Remark
	}
	if r.StopPriceType != "" {
		body["stopPriceType"] = r.StopPriceType
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOrder {
		body["closeOrder"] = true
	}
	if r.ForceHold {
		body["forceHold"] = true
	}
	if r.SelfTradePrevention != "" {
		body["stp"] = r.SelfTradePrevention
	}
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = r.Size
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.TriggerStopUpPrice != 0 {
		body["triggerStopUpPrice"] = strconv.FormatFloat(r.TriggerStopUpPrice, 'f', -1, 64)
	}
	if r.TriggerStopDownPrice != 0 {
		body["triggerStopDownPrice"] = strconv.FormatFloat(r.TriggerStopDownPrice, 'f', -1, 64)
	}
	if r.Quantity != 0 {
		body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	}
	if r.ValueQuantity != 0 {
		body["valueQty"] = strconv.FormatFloat(r.ValueQuantity, 'f', -1, 64)
	}
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	return body, nil
}

// FuturesCancelledOrdersResponse holds the IDs of the orders a cancel request cancelled
type FuturesCancelledOrdersResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
}

// FuturesCancelledClientOrderResponse holds the client order ID of the order Cancel Order By ClientOid cancelled
type FuturesCancelledClientOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
}

// BatchCancelFuturesOrdersRequest holds the parameters of Batch Cancel Orders: up to 10 orders, by order ID or by
// symbol and client order ID. KuCoin cancels by OrderIDs when both lists are set, ignoring ClientOrders. The
// futures.multi_cancel websocket operation sends the same body
type BatchCancelFuturesOrdersRequest struct {
	OrderIDs     []string
	ClientOrders []BatchCancelFuturesClientOrder
}

// BatchCancelFuturesClientOrder identifies an order of a Batch Cancel Orders request by its symbol and client order ID
type BatchCancelFuturesClientOrder struct {
	Symbol        string
	ClientOrderID string
}

// Payload validates the request and returns the Batch Cancel Orders body, which the REST endpoint and websocket
// order operation share
func (r *BatchCancelFuturesOrdersRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if len(r.OrderIDs) == 0 && len(r.ClientOrders) == 0 {
		return nil, errBatchCancelOrderIDsRequired
	}
	body := make(map[string]any, 2)
	if len(r.OrderIDs) != 0 {
		for i := range r.OrderIDs {
			if r.OrderIDs[i] == "" {
				return nil, fmt.Errorf("%w: order ID %d", order.ErrOrderIDNotSet, i)
			}
		}
		body["orderIdsList"] = r.OrderIDs
	}
	if len(r.ClientOrders) != 0 {
		clientOrders := make([]map[string]string, len(r.ClientOrders))
		for i := range r.ClientOrders {
			switch {
			case r.ClientOrders[i].Symbol == "":
				return nil, fmt.Errorf("%w: client order %d", currency.ErrSymbolStringEmpty, i)
			case r.ClientOrders[i].ClientOrderID == "":
				return nil, fmt.Errorf("%w: client order %d", order.ErrClientOrderIDMustBeSet, i)
			}
			clientOrder := make(map[string]string, 2)
			clientOrder["symbol"] = r.ClientOrders[i].Symbol
			clientOrder["clientOid"] = r.ClientOrders[i].ClientOrderID
			clientOrders[i] = clientOrder
		}
		body["clientOidsList"] = clientOrders
	}
	return body, nil
}

// FuturesBatchCancelResult is the outcome of one order of a Batch Cancel Orders request, which echoes the order ID
// or the client order ID the request gave. Code 200 means KuCoin cancelled the order; any other Code means it did
// not, for the reason in Message
type FuturesBatchCancelResult struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	Code          string `json:"code"`
	Message       string `json:"msg"`
}

// FuturesOrder holds the details of a futures order, as the order and order list endpoints return them
type FuturesOrder struct {
	ID     string       `json:"id"`
	Symbol string       `json:"symbol"`
	Type   string       `json:"type"`
	Side   string       `json:"side"`
	Price  types.Number `json:"price"`
	// Size is in lots
	Size      uint64       `json:"size"`
	Value     types.Number `json:"value"`
	DealValue types.Number `json:"dealValue"`
	// DealSize is in lots
	DealSize            uint64 `json:"dealSize"`
	SelfTradePrevention string `json:"stp"`
	Stop                string `json:"stop"`
	StopPriceType       string `json:"stopPriceType"`
	StopTriggered       bool   `json:"stopTriggered"`
	// StopPrice is documented as a number, an integer or a string depending on the endpoint, and is null for orders
	// without a stop
	StopPrice   types.Number `json:"stopPrice"`
	TimeInForce string       `json:"timeInForce"`
	PostOnly    bool         `json:"postOnly"`
	Hidden      bool         `json:"hidden"`
	Iceberg     bool         `json:"iceberg"`
	Leverage    types.Number `json:"leverage"`
	ForceHold   bool         `json:"forceHold"`
	CloseOrder  bool         `json:"closeOrder"`
	// VisibleSize is in lots
	VisibleSize   uint64     `json:"visibleSize"`
	ClientOrderID string     `json:"clientOid"`
	Remark        string     `json:"remark"`
	Tags          string     `json:"tags"`
	IsActive      bool       `json:"isActive"`
	CancelExist   bool       `json:"cancelExist"`
	CreatedAt     types.Time `json:"createdAt"`
	UpdatedAt     types.Time `json:"updatedAt"`
	// EndAt is null while the order is active
	EndAt          types.Time    `json:"endAt"`
	OrderTime      types.Time    `json:"orderTime"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	MarginMode     string        `json:"marginMode"`
	// AverageDealPrice is the filled value over the filled quantity for linear contracts, and the filled quantity over
	// the filled value for inverse contracts, a quantity being lots times the contract multiplier
	AverageDealPrice types.Number `json:"avgDealPrice"`
	// FilledSize is in lots
	FilledSize   uint64       `json:"filledSize"`
	FilledValue  types.Number `json:"filledValue"`
	Status       string       `json:"status"`
	ReduceOnly   bool         `json:"reduceOnly"`
	PositionSide string       `json:"positionSide"`
}

// FuturesOrderDetailResponse holds the details of a futures order (Get Order By OrderId, Get Order By ClientOid)
type FuturesOrderDetailResponse FuturesOrder

// GetFuturesOrdersRequest holds the parameters of Get Order List. Done orders can be listed for a window of up to
// seven days
type GetFuturesOrdersRequest struct {
	// Status is active or done; KuCoin lists done orders when it is empty
	Status string
	Symbol string
	Side   string
	// Type is limit, market, limit_stop, market_stop, oco_limit or oco_stop
	Type        string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	// PageSize is at most 1000; KuCoin defaults to 50
	PageSize uint64
}

// FuturesOrdersResponse holds a page of futures orders
type FuturesOrdersResponse struct {
	CurrentPage uint64         `json:"currentPage"`
	PageSize    uint64         `json:"pageSize"`
	TotalNumber uint64         `json:"totalNum"`
	TotalPage   uint64         `json:"totalPage"`
	Items       []FuturesOrder `json:"items"`
}

// GetUntriggeredFuturesStopOrdersRequest holds the parameters of Get Stop Order List
type GetUntriggeredFuturesStopOrdersRequest struct {
	Symbol string
	Side   string
	// Type is limit or market
	Type        string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	// PageSize is at most 1000; KuCoin defaults to 50
	PageSize uint64
}

// FuturesOpenOrderStatsResponse holds the number and value of a contract's active orders
type FuturesOpenOrderStatsResponse struct {
	// OpenOrderBuySize and OpenOrderSellSize are in lots
	OpenOrderBuySize  uint64        `json:"openOrderBuySize"`
	OpenOrderSellSize uint64        `json:"openOrderSellSize"`
	OpenOrderBuyCost  types.Number  `json:"openOrderBuyCost"`
	OpenOrderSellCost types.Number  `json:"openOrderSellCost"`
	SettleCurrency    currency.Code `json:"settleCurrency"`
}

// FuturesRecentFill holds a fill of the last 24 hours, as Get Recent Trade History returns it
type FuturesRecentFill struct {
	Symbol  string `json:"symbol"`
	TradeID string `json:"tradeId"`
	OrderID string `json:"orderId"`
	Side    string `json:"side"`
	// Liquidity is maker or taker
	Liquidity  string       `json:"liquidity"`
	ForceTaker bool         `json:"forceTaker"`
	Price      types.Number `json:"price"`
	// Size is in lots
	Size        uint64       `json:"size"`
	Value       types.Number `json:"value"`
	OpenFeePay  types.Number `json:"openFeePay"`
	CloseFeePay types.Number `json:"closeFeePay"`
	Stop        string       `json:"stop"`
	FeeRate     types.Number `json:"feeRate"`
	// FixFee is deprecated and carries no meaning
	FixFee      types.Number  `json:"fixFee"`
	FeeCurrency currency.Code `json:"feeCurrency"`
	TradeTime   types.Time    `json:"tradeTime"`
	// SubTradeType is deprecated and carries no meaning
	SubTradeType   string        `json:"subTradeType"`
	MarginMode     string        `json:"marginMode"`
	DisplayType    string        `json:"displayType"`
	Fee            types.Number  `json:"fee"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	OrderType      string        `json:"orderType"`
	// TradeType is trade, liquid, cancel, adl or settlement
	TradeType    string     `json:"tradeType"`
	CreatedAt    types.Time `json:"createdAt"`
	PositionSide string     `json:"positionSide"`
}

// GetFuturesFillsRequest holds the parameters of Get Trade History. KuCoin keeps three months of fills and lists a
// window of up to seven days per request
type GetFuturesFillsRequest struct {
	// OrderID lists the fills of one order, and KuCoin then ignores the other parameters
	OrderID string
	Symbol  string
	Side    string
	// Type is limit, market, limit_stop or market_stop
	Type string
	// TradeTypes are trade, adl, liquid or settlement; KuCoin lists all of them when it is empty
	TradeTypes  []string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	// PageSize is at most 1000; KuCoin defaults to 50
	PageSize uint64
}

// FuturesFillsResponse holds a page of futures fills
type FuturesFillsResponse struct {
	CurrentPage uint64        `json:"currentPage"`
	PageSize    uint64        `json:"pageSize"`
	TotalNumber uint64        `json:"totalNum"`
	TotalPage   uint64        `json:"totalPage"`
	Items       []FuturesFill `json:"items"`
}

// FuturesFill holds a fill, as Get Trade History returns it
type FuturesFill struct {
	Symbol  string `json:"symbol"`
	TradeID string `json:"tradeId"`
	OrderID string `json:"orderId"`
	Side    string `json:"side"`
	// Liquidity is maker or taker
	Liquidity  string       `json:"liquidity"`
	ForceTaker bool         `json:"forceTaker"`
	Price      types.Number `json:"price"`
	// Size is in lots
	Size        uint64       `json:"size"`
	Value       types.Number `json:"value"`
	OpenFeePay  types.Number `json:"openFeePay"`
	CloseFeePay types.Number `json:"closeFeePay"`
	Stop        string       `json:"stop"`
	FeeRate     types.Number `json:"feeRate"`
	// FixFee is deprecated and carries no meaning
	FixFee      types.Number  `json:"fixFee"`
	FeeCurrency currency.Code `json:"feeCurrency"`
	TradeTime   types.Time    `json:"tradeTime"`
	// SubTradeType is deprecated and carries no meaning
	SubTradeType   string        `json:"subTradeType"`
	MarginMode     string        `json:"marginMode"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	DisplayType    string        `json:"displayType"`
	Fee            types.Number  `json:"fee"`
	OrderType      string        `json:"orderType"`
	// TradeType is trade, liquid, adl or settlement
	TradeType string     `json:"tradeType"`
	CreatedAt types.Time `json:"createdAt"`
	// OpenFeeTaxPay and CloseFeeTaxPay are set only for users whose KYC region levies tax
	OpenFeeTaxPay  types.Number `json:"openFeeTaxPay"`
	CloseFeeTaxPay types.Number `json:"closeFeeTaxPay"`
	PositionSide   string       `json:"positionSide"`
}
