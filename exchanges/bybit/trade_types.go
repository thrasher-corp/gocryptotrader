package bybit

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

// PlaceOrderRequest holds the parameters of Place Order, which Pre Check Order takes as well
type PlaceOrderRequest struct {
	Category string
	Symbol   string
	// IsLeverage borrows to trade spot on margin
	IsLeverage bool
	Side       string
	OrderType  string
	// Quantity is in the base coin, except spot market buys, which are in the quote coin unless MarketUnit says
	// otherwise. A zero Quantity closes a perpetual or futures position when ReduceOnly and CloseOnTrigger are both set
	Quantity float64
	// MarketUnit selects the unit of Quantity for spot market orders: baseCoin or quoteCoin
	MarketUnit string
	// RPITakerAccess lets the order match RPI quotes; market orders and IOC or FOK limit orders only
	RPITakerAccess bool
	// SlippageToleranceType is TickSize or Percent, for market orders
	SlippageToleranceType string
	// SlippageTolerance is a number of ticks or a percentage, as SlippageToleranceType selects
	SlippageTolerance float64
	Price             float64
	// TriggerDirection is 1 when a conditional order triggers as the price rises to TriggerPrice and 2 as it falls
	TriggerDirection uint64
	OrderFilter      string
	TriggerPrice     float64
	TriggerBy        string
	// OrderImpliedVolatility is the option implied volatility as a fraction, 0.1 for 10%; it takes priority over Price
	OrderImpliedVolatility float64
	TimeInForce            string
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex  uint64
	OrderLinkID    string
	TakeProfit     float64
	StopLoss       float64
	TPTriggerBy    string
	SLTriggerBy    string
	ReduceOnly     bool
	CloseOnTrigger bool
	SMPType        string
	MMP            bool
	TPSLMode       string
	TPLimitPrice   float64
	SLLimitPrice   float64
	TPOrderType    string
	SLOrderType    string
	// BBOSideType prices the order from the best bid or offer: Queue for its own side of the book, Counterparty for the
	// other side
	BBOSideType string
	// BBOLevel is the orderbook level, 1 to 5, that BBOSideType prices from
	BBOLevel uint64
}

// Validate checks the parameters Place Order requires
func (r *PlaceOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	case r.Side == "":
		return order.ErrSideIsInvalid
	case r.OrderType == "":
		return order.ErrTypeIsInvalid
	case r.Quantity < 0, r.Quantity == 0 && (!r.ReduceOnly || !r.CloseOnTrigger):
		return limits.ErrAmountBelowMin
	}
	return nil
}

// Payload validates the request and returns the Place Order body, which the REST and websocket order endpoints share
func (r *PlaceOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	body["side"] = r.Side
	body["orderType"] = r.OrderType
	body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	if r.IsLeverage {
		body["isLeverage"] = 1
	}
	if r.MarketUnit != "" {
		body["marketUnit"] = r.MarketUnit
	}
	if r.RPITakerAccess {
		body["rpiTakerAccess"] = true
	}
	if r.SlippageToleranceType != "" {
		body["slippageToleranceType"] = r.SlippageToleranceType
	}
	if r.SlippageTolerance != 0 {
		body["slippageTolerance"] = strconv.FormatFloat(r.SlippageTolerance, 'f', -1, 64)
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.TriggerDirection != 0 {
		body["triggerDirection"] = r.TriggerDirection
	}
	if r.OrderFilter != "" {
		body["orderFilter"] = r.OrderFilter
	}
	if r.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(r.TriggerPrice, 'f', -1, 64)
	}
	if r.TriggerBy != "" {
		body["triggerBy"] = r.TriggerBy
	}
	if r.OrderImpliedVolatility != 0 {
		body["orderIv"] = strconv.FormatFloat(r.OrderImpliedVolatility, 'f', -1, 64)
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PositionIndex != 0 {
		body["positionIdx"] = r.PositionIndex
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	if r.TakeProfit != 0 {
		body["takeProfit"] = strconv.FormatFloat(r.TakeProfit, 'f', -1, 64)
	}
	if r.StopLoss != 0 {
		body["stopLoss"] = strconv.FormatFloat(r.StopLoss, 'f', -1, 64)
	}
	if r.TPTriggerBy != "" {
		body["tpTriggerBy"] = r.TPTriggerBy
	}
	if r.SLTriggerBy != "" {
		body["slTriggerBy"] = r.SLTriggerBy
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOnTrigger {
		body["closeOnTrigger"] = true
	}
	if r.SMPType != "" {
		body["smpType"] = r.SMPType
	}
	if r.MMP {
		body["mmp"] = true
	}
	if r.TPSLMode != "" {
		body["tpslMode"] = r.TPSLMode
	}
	if r.TPLimitPrice != 0 {
		body["tpLimitPrice"] = strconv.FormatFloat(r.TPLimitPrice, 'f', -1, 64)
	}
	if r.SLLimitPrice != 0 {
		body["slLimitPrice"] = strconv.FormatFloat(r.SLLimitPrice, 'f', -1, 64)
	}
	if r.TPOrderType != "" {
		body["tpOrderType"] = r.TPOrderType
	}
	if r.SLOrderType != "" {
		body["slOrderType"] = r.SLOrderType
	}
	if r.BBOSideType != "" {
		body["bboSideType"] = r.BBOSideType
	}
	if r.BBOLevel != 0 {
		body["bboLevel"] = strconv.FormatUint(r.BBOLevel, 10)
	}
	return body, nil
}

// OrderResponse holds the IDs of an order Bybit accepted. Placing, amending and cancelling are asynchronous, so the
// order's status follows on the order websocket stream
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// AmendOrderRequest holds the parameters of Amend Order. Zero values leave the order unchanged
type AmendOrderRequest struct {
	Category    string
	Symbol      string
	OrderID     string
	OrderLinkID string
	// OrderImpliedVolatility is the option implied volatility as a fraction, 0.1 for 10%
	OrderImpliedVolatility float64
	TriggerPrice           float64
	Quantity               float64
	Price                  float64
	TPSLMode               string
	// TakeProfit and StopLoss are left unchanged when nil, and cancelled when they point to zero
	TakeProfit   *float64
	StopLoss     *float64
	TPTriggerBy  string
	SLTriggerBy  string
	TriggerBy    string
	TPLimitPrice float64
	SLLimitPrice float64
}

// Validate checks the parameters Amend Order requires
func (r *AmendOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	case r.OrderID == "" && r.OrderLinkID == "":
		return errEitherOrderIDOROrderLinkIDRequired
	}
	return nil
}

// Payload validates the request and returns the Amend Order body, which the REST and websocket order endpoints share
func (r *AmendOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	if r.OrderImpliedVolatility != 0 {
		body["orderIv"] = strconv.FormatFloat(r.OrderImpliedVolatility, 'f', -1, 64)
	}
	if r.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(r.TriggerPrice, 'f', -1, 64)
	}
	if r.Quantity != 0 {
		body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.TPSLMode != "" {
		body["tpslMode"] = r.TPSLMode
	}
	if r.TakeProfit != nil {
		body["takeProfit"] = strconv.FormatFloat(*r.TakeProfit, 'f', -1, 64)
	}
	if r.StopLoss != nil {
		body["stopLoss"] = strconv.FormatFloat(*r.StopLoss, 'f', -1, 64)
	}
	if r.TPTriggerBy != "" {
		body["tpTriggerBy"] = r.TPTriggerBy
	}
	if r.SLTriggerBy != "" {
		body["slTriggerBy"] = r.SLTriggerBy
	}
	if r.TriggerBy != "" {
		body["triggerBy"] = r.TriggerBy
	}
	if r.TPLimitPrice != 0 {
		body["tpLimitPrice"] = strconv.FormatFloat(r.TPLimitPrice, 'f', -1, 64)
	}
	if r.SLLimitPrice != 0 {
		body["slLimitPrice"] = strconv.FormatFloat(r.SLLimitPrice, 'f', -1, 64)
	}
	return body, nil
}

// CancelOrderRequest holds the parameters of Cancel Order
type CancelOrderRequest struct {
	Category    string
	Symbol      string
	OrderID     string
	OrderLinkID string
	// OrderFilter is for spot orders only: Order (the default), tpslOrder or StopOrder
	OrderFilter string
}

// Validate checks the parameters Cancel Order requires
func (r *CancelOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	case r.OrderID == "" && r.OrderLinkID == "":
		return errEitherOrderIDOROrderLinkIDRequired
	}
	return nil
}

// Payload validates the request and returns the Cancel Order body, which the REST and websocket order endpoints share
func (r *CancelOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	if r.OrderFilter != "" {
		body["orderFilter"] = r.OrderFilter
	}
	return body, nil
}

// OpenOrdersRequest holds the parameters of Get Open & Closed Orders
type OpenOrdersRequest struct {
	Category   string
	Symbol     string
	BaseCoin   currency.Code
	SettleCoin currency.Code
	// OrderID and OrderLinkID take priority over Symbol and BaseCoin, and ignore OpenOnly
	OrderID     string
	OrderLinkID string
	// OpenOnly is 0 for open orders and 1 for the most recent 500 closed orders
	OpenOnly    uint64
	OrderFilter string
	Limit       uint64
	Cursor      string
}

// OpenOrdersResponse holds a page of orders from Get Open & Closed Orders
type OpenOrdersResponse struct {
	Category       string      `json:"category"`
	NextPageCursor string      `json:"nextPageCursor"`
	List           []OpenOrder `json:"list"`
}

// OpenOrder is an order from Get Open & Closed Orders
type OpenOrder struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	// ParentOrderLinkID links a futures or options take profit or stop loss order to the order it was attached to
	ParentOrderLinkID string       `json:"parentOrderLinkId"`
	BlockTradeID      string       `json:"blockTradeId"`
	Symbol            string       `json:"symbol"`
	Price             types.Number `json:"price"`
	Quantity          types.Number `json:"qty"`
	Side              string       `json:"side"`
	// IsLeverage is "1" for a spot margin order that borrows and "0" for one that does not; it is empty for derivatives
	IsLeverage                 string       `json:"isLeverage"`
	PositionIndex              uint64       `json:"positionIdx"`
	OrderStatus                string       `json:"orderStatus"`
	CreateType                 string       `json:"createType"`
	CancelType                 string       `json:"cancelType"`
	RejectReason               string       `json:"rejectReason"`
	AveragePrice               types.Number `json:"avgPrice"`
	LeavesQuantity             types.Number `json:"leavesQty"`
	LeavesValue                types.Number `json:"leavesValue"`
	CumulativeExecutedQuantity types.Number `json:"cumExecQty"`
	CumulativeExecutedValue    types.Number `json:"cumExecValue"`
	// CumulativeExecutedFee is deprecated for linear and spot orders, which report their fees in CumulativeFeeDetail
	CumulativeExecutedFee  types.Number `json:"cumExecFee"`
	TimeInForce            string       `json:"timeInForce"`
	OrderType              string       `json:"orderType"`
	StopOrderType          string       `json:"stopOrderType"`
	OrderImpliedVolatility types.Number `json:"orderIv"`
	MarketUnit             string       `json:"marketUnit"`
	// TriggerPrice is the activation price when StopOrderType is TrailingStop
	TriggerPrice     types.Number `json:"triggerPrice"`
	TakeProfit       types.Number `json:"takeProfit"`
	StopLoss         types.Number `json:"stopLoss"`
	TPSLMode         string       `json:"tpslMode"`
	OCOTriggerBy     string       `json:"ocoTriggerBy"`
	TPLimitPrice     types.Number `json:"tpLimitPrice"`
	SLLimitPrice     types.Number `json:"slLimitPrice"`
	TPTriggerBy      string       `json:"tpTriggerBy"`
	SLTriggerBy      string       `json:"slTriggerBy"`
	TriggerDirection uint64       `json:"triggerDirection"`
	TriggerBy        string       `json:"triggerBy"`
	// LastPriceOnCreated is the last price when the order was placed; spot orders report it in BasePrice instead
	LastPriceOnCreated types.Number `json:"lastPriceOnCreated"`
	BasePrice          types.Number `json:"basePrice"`
	ReduceOnly         bool         `json:"reduceOnly"`
	CloseOnTrigger     bool         `json:"closeOnTrigger"`
	PlaceType          string       `json:"placeType"`
	SMPType            string       `json:"smpType"`
	// SMPGroup is documented as a string but has been sent as a bare number
	SMPGroup    types.Number `json:"smpGroup"`
	SMPOrderID  string       `json:"smpOrderId"`
	CreatedTime types.Time   `json:"createdTime"`
	UpdatedTime types.Time   `json:"updatedTime"`
	// CumulativeFeeDetail holds the cumulative fees of linear and spot orders by fee coin
	CumulativeFeeDetail map[currency.Code]types.Number `json:"cumFeeDetail"`
}

// CancelAllOrdersRequest holds the parameters of Cancel All Orders. Linear and inverse orders need one of Symbol,
// BaseCoin or SettleCoin, in that order of priority
type CancelAllOrdersRequest struct {
	Category    string
	Symbol      string
	BaseCoin    currency.Code
	SettleCoin  currency.Code
	OrderFilter string
	// StopOrderType Stop, with OrderFilter StopOrder, cancels linear and inverse conditional orders but not their TP/SL
	// or trailing stop orders
	StopOrderType string
}

// CancelAllOrdersResponse holds the orders Cancel All Orders cancelled
type CancelAllOrdersResponse struct {
	List []OrderResponse `json:"list"`
	// Success is false when the cancellation failed; retired UTA 1.0 inverse accounts did not send it
	Success types.Boolean `json:"success"`
}

// OrderHistoryRequest holds the parameters of Get Order History. Without times it returns the last 7 days, and the
// window between StartTime and EndTime may not exceed 7 days
type OrderHistoryRequest struct {
	Category    string
	Symbol      string
	BaseCoin    currency.Code
	SettleCoin  currency.Code
	OrderID     string
	OrderLinkID string
	OrderFilter string
	OrderStatus string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// OrderHistoryResponse holds a page of orders from Get Order History
type OrderHistoryResponse struct {
	Category       string            `json:"category"`
	NextPageCursor string            `json:"nextPageCursor"`
	List           []HistoricalOrder `json:"list"`
}

// HistoricalOrder is an order from Get Order History, which reports the fields of an open order and these
type HistoricalOrder struct {
	OpenOrder
	SlippageToleranceType string       `json:"slippageToleranceType"`
	SlippageTolerance     types.Number `json:"slippageTolerance"`
	// ExtraFees lists, as a JSON string, the taxes charged on the Indonesian and EU sites; it is empty elsewhere
	ExtraFees string `json:"extraFees"`
	// RPITakerAccess is true when the order matched an RPI order
	RPITakerAccess     bool         `json:"rpiTakerAccess"`
	RPIMatchedQuantity types.Number `json:"rpiMatchedQty"`
}

// ExecutionRequest holds the parameters of Get Trade History. Without times it returns the last 7 days, and the
// window between StartTime and EndTime may not exceed 7 days
type ExecutionRequest struct {
	Category string
	Symbol   string
	// OrderID and OrderLinkID take priority over the other filters, which are then ignored
	OrderID       string
	OrderLinkID   string
	BaseCoin      currency.Code
	SettleCoin    currency.Code
	StartTime     time.Time
	EndTime       time.Time
	ExecutionType string
	Limit         uint64
	Cursor        string
}

// ExecutionResponse holds a page of executions from Get Trade History
type ExecutionResponse struct {
	Category       string      `json:"category"`
	NextPageCursor string      `json:"nextPageCursor"`
	List           []Execution `json:"list"`
}

// Execution is a fill from Get Trade History
type Execution struct {
	Symbol         string       `json:"symbol"`
	OrderID        string       `json:"orderId"`
	OrderLinkID    string       `json:"orderLinkId"`
	Side           string       `json:"side"`
	OrderPrice     types.Number `json:"orderPrice"`
	OrderQuantity  types.Number `json:"orderQty"`
	LeavesQuantity types.Number `json:"leavesQty"`
	CreateType     string       `json:"createType"`
	OrderType      string       `json:"orderType"`
	StopOrderType  string       `json:"stopOrderType"`
	ExecutionFee   types.Number `json:"execFee"`
	// ExecutionFeeV2 is the fee of the spot leg of a FutureSpread execution
	ExecutionFeeV2         types.Number  `json:"execFeeV2"`
	ExecutionID            string        `json:"execId"`
	ExecutionPrice         types.Number  `json:"execPrice"`
	ExecutionQuantity      types.Number  `json:"execQty"`
	ExecutionType          string        `json:"execType"`
	ExecutionValue         types.Number  `json:"execValue"`
	ExecutionTime          types.Time    `json:"execTime"`
	FeeCurrency            currency.Code `json:"feeCurrency"`
	IsMaker                bool          `json:"isMaker"`
	FeeRate                types.Number  `json:"feeRate"`
	TradeImpliedVolatility types.Number  `json:"tradeIv"`
	MarkImpliedVolatility  types.Number  `json:"markIv"`
	MarkPrice              types.Number  `json:"markPrice"`
	IndexPrice             types.Number  `json:"indexPrice"`
	UnderlyingPrice        types.Number  `json:"underlyingPrice"`
	BlockTradeID           string        `json:"blockTradeId"`
	ClosedSize             types.Number  `json:"closedSize"`
	// Sequence links fills with position updates; it is unique per symbol
	Sequence uint64 `json:"seq"`
	// ExtraFees lists, as a JSON string, the taxes charged to Indian users and on the Indonesian and EU sites; it is
	// empty elsewhere
	ExtraFees string `json:"extraFees"`
}

// BatchPlaceOrderRequest holds the parameters of Batch Place Order: up to 20 linear or inverse, 10 spot or 5 option
// orders of one category. Bybit counts each linear or inverse order against a budget of 10 a second, so the rate limit
// refuses a batch of more
type BatchPlaceOrderRequest struct {
	Category string
	Request  []BatchPlaceOrderItem
}

// Payload validates the request and returns the Batch Place Order body, which the REST and websocket batch order
// endpoints share
func (r *BatchPlaceOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if len(r.Request) == 0 {
		return nil, errNoOrderPassed
	}
	orders := make([]map[string]any, len(r.Request))
	for i := range r.Request {
		item, err := r.Request[i].Payload()
		if err != nil {
			return nil, fmt.Errorf("request %d: %w", i, err)
		}
		orders[i] = item
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["request"] = orders
	return body, nil
}

// BatchPlaceOrderItem holds the parameters of one order of a Batch Place Order request
type BatchPlaceOrderItem struct {
	Symbol string
	// IsLeverage borrows to trade spot on margin
	IsLeverage bool
	Side       string
	OrderType  string
	// Quantity is in the base coin, except spot market buys, which are in the quote coin unless MarketUnit says
	// otherwise. A zero Quantity closes a perpetual or futures position when ReduceOnly and CloseOnTrigger are both set
	Quantity float64
	// MarketUnit selects the unit of Quantity for spot market orders: baseCoin or quoteCoin
	MarketUnit string
	Price      float64
	// TriggerDirection is 1 when a conditional order triggers as the price rises to TriggerPrice and 2 as it falls
	TriggerDirection uint64
	OrderFilter      string
	TriggerPrice     float64
	TriggerBy        string
	// OrderImpliedVolatility is the option implied volatility as a fraction, 0.1 for 10%; it takes priority over Price
	OrderImpliedVolatility float64
	TimeInForce            string
	// PositionIndex is 0 in one-way mode, and 1 for the buy side or 2 for the sell side in hedge mode
	PositionIndex  uint64
	OrderLinkID    string
	TakeProfit     float64
	StopLoss       float64
	TPTriggerBy    string
	SLTriggerBy    string
	ReduceOnly     bool
	CloseOnTrigger bool
	SMPType        string
	MMP            bool
	TPSLMode       string
	TPLimitPrice   float64
	SLLimitPrice   float64
	TPOrderType    string
	SLOrderType    string
}

// Payload validates the order and returns its entry of a Batch Place Order body
func (r *BatchPlaceOrderItem) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Symbol == "":
		return nil, errSymbolMissing
	case r.Side == "":
		return nil, order.ErrSideIsInvalid
	case r.OrderType == "":
		return nil, order.ErrTypeIsInvalid
	case r.Quantity < 0, r.Quantity == 0 && (!r.ReduceOnly || !r.CloseOnTrigger):
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["side"] = r.Side
	body["orderType"] = r.OrderType
	body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	if r.IsLeverage {
		body["isLeverage"] = 1
	}
	if r.MarketUnit != "" {
		body["marketUnit"] = r.MarketUnit
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.TriggerDirection != 0 {
		body["triggerDirection"] = r.TriggerDirection
	}
	if r.OrderFilter != "" {
		body["orderFilter"] = r.OrderFilter
	}
	if r.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(r.TriggerPrice, 'f', -1, 64)
	}
	if r.TriggerBy != "" {
		body["triggerBy"] = r.TriggerBy
	}
	if r.OrderImpliedVolatility != 0 {
		body["orderIv"] = strconv.FormatFloat(r.OrderImpliedVolatility, 'f', -1, 64)
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PositionIndex != 0 {
		body["positionIdx"] = r.PositionIndex
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	if r.TakeProfit != 0 {
		body["takeProfit"] = strconv.FormatFloat(r.TakeProfit, 'f', -1, 64)
	}
	if r.StopLoss != 0 {
		body["stopLoss"] = strconv.FormatFloat(r.StopLoss, 'f', -1, 64)
	}
	if r.TPTriggerBy != "" {
		body["tpTriggerBy"] = r.TPTriggerBy
	}
	if r.SLTriggerBy != "" {
		body["slTriggerBy"] = r.SLTriggerBy
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOnTrigger {
		body["closeOnTrigger"] = true
	}
	if r.SMPType != "" {
		body["smpType"] = r.SMPType
	}
	if r.MMP {
		body["mmp"] = true
	}
	if r.TPSLMode != "" {
		body["tpslMode"] = r.TPSLMode
	}
	if r.TPLimitPrice != 0 {
		body["tpLimitPrice"] = strconv.FormatFloat(r.TPLimitPrice, 'f', -1, 64)
	}
	if r.SLLimitPrice != 0 {
		body["slLimitPrice"] = strconv.FormatFloat(r.SLLimitPrice, 'f', -1, 64)
	}
	if r.TPOrderType != "" {
		body["tpOrderType"] = r.TPOrderType
	}
	if r.SLOrderType != "" {
		body["slOrderType"] = r.SLOrderType
	}
	return body, nil
}

// BatchOrderExtInfo is the retExtInfo of a batch order request: Bybit's verdict on each order, in request order
type BatchOrderExtInfo struct {
	List []BatchOrderOutcome `json:"list"`
}

// BatchOrderOutcome is Bybit's verdict on one order of a batch order request
type BatchOrderOutcome struct {
	Code    int64  `json:"code"` // Bybit's error codes include negative ones, such as -1 and -2015
	Message string `json:"msg"`
}

// BatchPlaceOrderResponse holds the outcome of each order of a Batch Place Order request, in request order
type BatchPlaceOrderResponse struct {
	List []BatchPlacedOrder `json:"list"`
}

// BatchPlacedOrder is the outcome of one order of a Batch Place Order request. Code and Message are Bybit's verdict
// from retExtInfo: a non-zero Code means the order was rejected
type BatchPlacedOrder struct {
	Category    string     `json:"category"`
	Symbol      string     `json:"symbol"`
	OrderID     string     `json:"orderId"`
	OrderLinkID string     `json:"orderLinkId"`
	CreateAt    types.Time `json:"createAt"`
	Code        int64      `json:"code"` // Bybit's error codes include negative ones, such as -1 and -2015
	Message     string     `json:"msg"`
}

// BatchAmendOrderRequest holds the parameters of Batch Amend Order: up to 20 linear or inverse, 10 spot or 5 option
// orders of one category. Bybit counts each linear or inverse order against a budget of 10 a second, so the rate limit
// refuses a batch of more
type BatchAmendOrderRequest struct {
	Category string
	Request  []BatchAmendOrderItem
}

// Payload validates the request and returns the Batch Amend Order body, which the REST and websocket batch order
// endpoints share
func (r *BatchAmendOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if len(r.Request) == 0 {
		return nil, errNoOrderPassed
	}
	orders := make([]map[string]any, len(r.Request))
	for i := range r.Request {
		item, err := r.Request[i].Payload()
		if err != nil {
			return nil, fmt.Errorf("request %d: %w", i, err)
		}
		orders[i] = item
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["request"] = orders
	return body, nil
}

// BatchAmendOrderItem holds the parameters of one order of a Batch Amend Order request. Zero values leave the order
// unchanged
type BatchAmendOrderItem struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
	// OrderImpliedVolatility is the option implied volatility as a fraction, 0.1 for 10%
	OrderImpliedVolatility float64
	TriggerPrice           float64
	Quantity               float64
	Price                  float64
	TPSLMode               string
	// TakeProfit and StopLoss are left unchanged when nil, and cancelled when they point to zero
	TakeProfit   *float64
	StopLoss     *float64
	TPTriggerBy  string
	SLTriggerBy  string
	TriggerBy    string
	TPLimitPrice float64
	SLLimitPrice float64
}

// Payload validates the order and returns its entry of a Batch Amend Order body
func (r *BatchAmendOrderItem) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Symbol == "":
		return nil, errSymbolMissing
	case r.OrderID == "" && r.OrderLinkID == "":
		return nil, errEitherOrderIDOROrderLinkIDRequired
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	if r.OrderImpliedVolatility != 0 {
		body["orderIv"] = strconv.FormatFloat(r.OrderImpliedVolatility, 'f', -1, 64)
	}
	if r.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(r.TriggerPrice, 'f', -1, 64)
	}
	if r.Quantity != 0 {
		body["qty"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.TPSLMode != "" {
		body["tpslMode"] = r.TPSLMode
	}
	if r.TakeProfit != nil {
		body["takeProfit"] = strconv.FormatFloat(*r.TakeProfit, 'f', -1, 64)
	}
	if r.StopLoss != nil {
		body["stopLoss"] = strconv.FormatFloat(*r.StopLoss, 'f', -1, 64)
	}
	if r.TPTriggerBy != "" {
		body["tpTriggerBy"] = r.TPTriggerBy
	}
	if r.SLTriggerBy != "" {
		body["slTriggerBy"] = r.SLTriggerBy
	}
	if r.TriggerBy != "" {
		body["triggerBy"] = r.TriggerBy
	}
	if r.TPLimitPrice != 0 {
		body["tpLimitPrice"] = strconv.FormatFloat(r.TPLimitPrice, 'f', -1, 64)
	}
	if r.SLLimitPrice != 0 {
		body["slLimitPrice"] = strconv.FormatFloat(r.SLLimitPrice, 'f', -1, 64)
	}
	return body, nil
}

// BatchOrderResponse holds the outcome of each order of a Batch Amend Order or Batch Cancel Order request, in request
// order
type BatchOrderResponse struct {
	List []BatchOrderResult `json:"list"`
}

// BatchOrderResult is the outcome of one order of a Batch Amend Order or Batch Cancel Order request. Code and Message
// are Bybit's verdict from retExtInfo: a non-zero Code means the order was not amended or cancelled
type BatchOrderResult struct {
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	Code        int64  `json:"code"` // Bybit's error codes include negative ones, such as -1 and -2015
	Message     string `json:"msg"`
}

// BatchCancelOrderRequest holds the parameters of Batch Cancel Order: up to 20 linear or inverse, 10 spot or 5 option
// orders of one category. Bybit counts each linear or inverse order against a budget of 10 a second, so the rate limit
// refuses a batch of more
type BatchCancelOrderRequest struct {
	Category string
	Request  []BatchCancelOrderItem
}

// Payload validates the request and returns the Batch Cancel Order body, which the REST and websocket batch order
// endpoints share
func (r *BatchCancelOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if len(r.Request) == 0 {
		return nil, errNoOrderPassed
	}
	orders := make([]map[string]any, len(r.Request))
	for i := range r.Request {
		item, err := r.Request[i].Payload()
		if err != nil {
			return nil, fmt.Errorf("request %d: %w", i, err)
		}
		orders[i] = item
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["request"] = orders
	return body, nil
}

// BatchCancelOrderItem holds the parameters of one order of a Batch Cancel Order request
type BatchCancelOrderItem struct {
	Symbol      string
	OrderID     string
	OrderLinkID string
}

// Payload validates the order and returns its entry of a Batch Cancel Order body
func (r *BatchCancelOrderItem) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Symbol == "":
		return nil, errSymbolMissing
	case r.OrderID == "" && r.OrderLinkID == "":
		return nil, errEitherOrderIDOROrderLinkIDRequired
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.OrderLinkID != "" {
		body["orderLinkId"] = r.OrderLinkID
	}
	return body, nil
}

// BorrowQuotaRequest holds the parameters of Get Borrow Quota (Spot)
type BorrowQuotaRequest struct {
	Category string
	Symbol   string
	Side     string
}

// BorrowQuotaResponse holds the most a spot or spot margin order can trade, from Get Borrow Quota (Spot)
type BorrowQuotaResponse struct {
	Symbol string `json:"symbol"`
	Side   string `json:"side"`
	// MaxTradeQuantity and MaxTradeAmount include what can be borrowed when spot margin trading is on for a margin pair
	MaxTradeQuantity types.Number `json:"maxTradeQty"`
	MaxTradeAmount   types.Number `json:"maxTradeAmount"`
	// SpotMaxTradeQuantity and SpotMaxTradeAmount exclude what can be borrowed
	SpotMaxTradeQuantity types.Number  `json:"spotMaxTradeQty"`
	SpotMaxTradeAmount   types.Number  `json:"spotMaxTradeAmount"`
	BorrowCoin           currency.Code `json:"borrowCoin"`
}

// SetDisconnectCancelAllRequest holds the parameters of Set Disconnect Cancel All
type SetDisconnectCancelAllRequest struct {
	// Product is OPTIONS (the default), DERIVATIVES or SPOT
	Product string
	// TimeWindow is sent in whole seconds, from 3 to 300
	TimeWindow time.Duration
}

// PreCheckOrderResponse holds the account's margin rates before and after an order, from Pre Check Order. The rates
// are scaled by 1e4: 30 is a rate of 0.3%
type PreCheckOrderResponse struct {
	OrderID                     string       `json:"orderId"`
	OrderLinkID                 string       `json:"orderLinkId"`
	PreInitialMarginRateE4      types.Number `json:"preImrE4"`
	PreMaintenanceMarginRateE4  types.Number `json:"preMmrE4"`
	PostInitialMarginRateE4     types.Number `json:"postImrE4"`
	PostMaintenanceMarginRateE4 types.Number `json:"postMmrE4"`
}
