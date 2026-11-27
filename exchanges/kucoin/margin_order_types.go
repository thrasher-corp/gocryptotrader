package kucoin

import (
	"bytes"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceMarginHFOrderRequest holds the parameters of Add Order and Add Order Test, which the websocket margin.order
// operation takes as well
type PlaceMarginHFOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	// Type is limit or market; KuCoin places a limit order when it is empty
	Type string
	// Remark is sent by the docs' request examples, though the body schema omits it
	Remark              string
	SelfTradePrevention string
	// Price and Size are required for limit orders
	Price float64
	// Size is in the base currency; a market order takes either Size or Funds
	Size        float64
	TimeInForce string
	PostOnly    bool
	// CancelAfter is in seconds, for GTT orders
	CancelAfter uint64
	// Funds is in the quote currency, for market orders
	Funds float64
	// IsIsolated places the order in the isolated margin account instead of the cross margin account
	IsIsolated bool
	AutoBorrow bool
	AutoRepay  bool
}

// Validate checks the parameters Add Order requires
func (r *PlaceMarginHFOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.ClientOrderID == "" {
		return order.ErrClientOrderIDMustBeSet
	}
	if r.Side == "" {
		return order.ErrSideIsInvalid
	}
	if r.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	return validateMarginOrderAmounts(r.Type, r.Price, r.Size, r.Funds)
}

// Payload validates the request and returns the Add Order body, which Add Order Test and the websocket margin.order
// operation share
func (r *PlaceMarginHFOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["side"] = r.Side
	body["symbol"] = r.Symbol
	if r.Type != "" {
		body["type"] = r.Type
	}
	if r.Remark != "" {
		body["remark"] = r.Remark
	}
	if r.SelfTradePrevention != "" {
		body["stp"] = r.SelfTradePrevention
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = strconv.FormatFloat(r.Size, 'f', -1, 64)
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.CancelAfter != 0 {
		body["cancelAfter"] = r.CancelAfter
	}
	if r.Funds != 0 {
		body["funds"] = strconv.FormatFloat(r.Funds, 'f', -1, 64)
	}
	if r.IsIsolated {
		body["isIsolated"] = true
	}
	if r.AutoBorrow {
		body["autoBorrow"] = true
	}
	if r.AutoRepay {
		body["autoRepay"] = true
	}
	return body, nil
}

// PlaceMarginHFOrderResponse holds the reply of Add Order and Add Order Test
type PlaceMarginHFOrderResponse struct {
	OrderID string `json:"orderId"`
	// LoanApplyID and BorrowSize are only returned for orders placed with AutoBorrow
	LoanApplyID string `json:"loanApplyId"`
	// BorrowSize is a quoted number in the Add Order docs and a bare one in the Add Order Test docs
	BorrowSize    types.Number `json:"borrowSize"`
	ClientOrderID string       `json:"clientOid"`
}

// CancelMarginHFOrderByOrderIDResponse holds the reply of Cancel Order By OrderId
type CancelMarginHFOrderByOrderIDResponse struct {
	OrderID string `json:"orderId"`
}

// CancelMarginHFOrderByClientOrderIDResponse holds the reply of Cancel Order By ClientOid
type CancelMarginHFOrderByClientOrderIDResponse struct {
	ClientOrderID string `json:"clientOid"`
}

// ActiveMarginHFOrderSymbolsResponse holds the reply of Get Symbols With Open Order
type ActiveMarginHFOrderSymbolsResponse struct {
	SymbolSize uint64   `json:"symbolSize"`
	Symbols    []string `json:"symbols"`
}

// MarginHFOrderResponse holds a margin HF order as Get Open Orders lists it
type MarginHFOrderResponse struct {
	ID                  string        `json:"id"`
	Symbol              string        `json:"symbol"`
	OperationType       string        `json:"opType"`
	Type                string        `json:"type"`
	Side                string        `json:"side"`
	Price               types.Number  `json:"price"`
	Size                types.Number  `json:"size"`
	Funds               types.Number  `json:"funds"`
	DealSize            types.Number  `json:"dealSize"`
	DealFunds           types.Number  `json:"dealFunds"`
	Fee                 types.Number  `json:"fee"`
	FeeCurrency         currency.Code `json:"feeCurrency"`
	SelfTradePrevention string        `json:"stp"`
	Stop                string        `json:"stop"`
	StopTriggered       bool          `json:"stopTriggered"`
	StopPrice           types.Number  `json:"stopPrice"`
	TimeInForce         string        `json:"timeInForce"`
	PostOnly            bool          `json:"postOnly"`
	Hidden              bool          `json:"hidden"`
	Iceberg             bool          `json:"iceberg"`
	VisibleSize         types.Number  `json:"visibleSize"`
	// CancelAfter is in seconds
	CancelAfter   int64      `json:"cancelAfter"`
	Channel       string     `json:"channel"`
	ClientOrderID string     `json:"clientOid"`
	Remark        string     `json:"remark"`
	Tags          string     `json:"tags"`
	CancelExist   bool       `json:"cancelExist"`
	CreatedAt     types.Time `json:"createdAt"`
	LastUpdatedAt types.Time `json:"lastUpdatedAt"`
	TradeType     string     `json:"tradeType"`
	InOrderBook   bool       `json:"inOrderBook"`
	// CancelledSize, CancelledFunds, RemainSize and RemainFunds split the order's size and funds into what was
	// cancelled and what is still open
	CancelledSize  types.Number `json:"cancelledSize"`
	CancelledFunds types.Number `json:"cancelledFunds"`
	RemainSize     types.Number `json:"remainSize"`
	RemainFunds    types.Number `json:"remainFunds"`
	Tax            types.Number `json:"tax"`
	Active         bool         `json:"active"`
}

// GetClosedMarginHFOrdersRequest holds the parameters of Get Closed Orders
type GetClosedMarginHFOrdersRequest struct {
	Symbol string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType string
	Side      string
	Type      string
	// LastID is the LastID of the previous page, for paging
	LastID uint64
	Limit  uint64
	// StartAt and EndAt bound the order update time; KuCoin keeps closed orders for seven days
	StartAt time.Time
	EndAt   time.Time
}

// MarginHFOrderDetailResponse holds a margin HF order as Get Closed Orders, Get Order By OrderId and Get Order By ClientOid
// return it
type MarginHFOrderDetailResponse struct {
	MarginHFOrderResponse
	// CancelReason is the vendor's cancellation reason code, 0 for a cancellation the user requested
	CancelReason uint64 `json:"cancelReason"`
}

// ClosedMarginHFOrdersResponse holds the reply of Get Closed Orders
type ClosedMarginHFOrdersResponse struct {
	LastID uint64                        `json:"lastId"`
	Items  []MarginHFOrderDetailResponse `json:"items"`
}

// GetMarginHFTradeFillsRequest holds the parameters of Get Trade History
type GetMarginHFTradeFillsRequest struct {
	Symbol string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType string
	// OrderID makes KuCoin ignore the other filters
	OrderID string
	Side    string
	Type    string
	// LastID is the LastID of the previous page, for paging
	LastID uint64
	Limit  uint64
	// StartAt and EndAt may span at most seven days
	StartAt time.Time
	EndAt   time.Time
}

// MarginHFTradeFillsResponse holds the reply of Get Trade History
type MarginHFTradeFillsResponse struct {
	Items  []MarginHFTradeFill `json:"items"`
	LastID uint64              `json:"lastId"`
}

// MarginHFTradeFill holds a margin HF trade fill
type MarginHFTradeFill struct {
	ID             uint64        `json:"id"`
	Symbol         string        `json:"symbol"`
	TradeID        uint64        `json:"tradeId"`
	OrderID        string        `json:"orderId"`
	CounterOrderID string        `json:"counterOrderId"`
	Side           string        `json:"side"`
	Liquidity      string        `json:"liquidity"`
	ForceTaker     bool          `json:"forceTaker"`
	Price          types.Number  `json:"price"`
	Size           types.Number  `json:"size"`
	Funds          types.Number  `json:"funds"`
	Fee            types.Number  `json:"fee"`
	FeeRate        types.Number  `json:"feeRate"`
	FeeCurrency    currency.Code `json:"feeCurrency"`
	Stop           string        `json:"stop"`
	TradeType      string        `json:"tradeType"`
	Tax            types.Number  `json:"tax"`
	TaxRate        types.Number  `json:"taxRate"`
	Type           string        `json:"type"`
	CreatedAt      types.Time    `json:"createdAt"`
}

// PlaceMarginStopOrderRequest holds the parameters of Add Stop Order
type PlaceMarginStopOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	// Type is limit or market; KuCoin places a limit order when it is empty
	Type                string
	SelfTradePrevention string
	// Price and Size are required for limit orders
	Price float64
	// Size is in the base currency; a market order takes either Size or Funds
	Size        float64
	TimeInForce string
	PostOnly    bool
	// CancelAfter is in seconds, for GTT orders
	CancelAfter uint64
	// Funds is in the quote currency, for market orders
	Funds float64
	// IsIsolated, AutoBorrow and AutoRepay are always sent, as the API requires them
	IsIsolated bool
	AutoBorrow bool
	AutoRepay  bool
	Remark     string
	// Stop is loss, which triggers at or below StopPrice, or entry, which triggers at or above it; KuCoin uses loss
	// when it is empty
	Stop      string
	StopPrice float64
}

// PlaceMarginStopOrderResponse holds the reply of Add Stop Order
type PlaceMarginStopOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}

// CancelledMarginOrdersResponse holds the order IDs a margin stop or OCO order cancellation cancelled
type CancelledMarginOrdersResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
}

// CancelMarginStopOrdersRequest holds the parameters of Batch Cancel Stop Orders
type CancelMarginStopOrdersRequest struct {
	Symbol string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType string
	OrderIDs  []string
}

// GetMarginStopOrdersRequest holds the parameters of Get Stop Orders List
type GetMarginStopOrdersRequest struct {
	Symbol string
	Side   string
	// Type is limit, market, limit_stop or market_stop
	Type string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType   string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	OrderIDs    []string
	PageSize    uint64
	// Stop is stop for stop orders or oco for OCO orders
	Stop string
}

// MarginStopOrderResponse holds a margin stop order as Get Stop Order By OrderId and Get Stop Order By ClientOid
// return it
type MarginStopOrderResponse struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	UserID string `json:"userId"`
	// Status is NEW or TRIGGERED
	Status              string       `json:"status"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	Price               types.Number `json:"price"`
	Size                types.Number `json:"size"`
	Funds               types.Number `json:"funds"`
	SelfTradePrevention string       `json:"stp"`
	TimeInForce         string       `json:"timeInForce"`
	// CancelAfter is in seconds, -1 when the order does not expire
	CancelAfter   int64        `json:"cancelAfter"`
	PostOnly      bool         `json:"postOnly"`
	Hidden        bool         `json:"hidden"`
	Iceberg       bool         `json:"iceberg"`
	VisibleSize   types.Number `json:"visibleSize"`
	Channel       string       `json:"channel"`
	ClientOrderID string       `json:"clientOid"`
	Remark        string       `json:"remark"`
	Tags          string       `json:"tags"`
	DomainID      string       `json:"domainId"`
	// TradeSource is USER for an order the user placed or MARGIN_SYSTEM for one the margin system placed
	TradeSource     string        `json:"tradeSource"`
	TradeType       string        `json:"tradeType"`
	FeeCurrency     currency.Code `json:"feeCurrency"`
	TakerFeeRate    types.Number  `json:"takerFeeRate"`
	MakerFeeRate    types.Number  `json:"makerFeeRate"`
	CreatedAt       types.Time    `json:"createdAt"`
	Stop            string        `json:"stop"`
	StopTriggerTime types.Time    `json:"stopTriggerTime"`
	StopPrice       types.Number  `json:"stopPrice"`
	// OrderTime is in nanoseconds
	OrderTime types.Time `json:"orderTime"`
}

// MarginStopOrderListItem holds a margin stop order as Get Stop Orders List returns it
type MarginStopOrderListItem struct {
	MarginStopOrderResponse
	// RelatedNumber, LimitPrice, Pop and ActivateCondition appear only in the Get Stop Orders List docs' example, as
	// null; the spot stop order docs type them as strings
	RelatedNumber     string       `json:"relatedNo"`
	LimitPrice        types.Number `json:"limitPrice"`
	Pop               string       `json:"pop"`
	ActivateCondition string       `json:"activateCondition"`
}

// MarginStopOrderListResponse holds the reply of Get Stop Orders List
type MarginStopOrderListResponse struct {
	CurrentPage uint64                    `json:"currentPage"`
	PageSize    uint64                    `json:"pageSize"`
	TotalNumber uint64                    `json:"totalNum"`
	TotalPage   uint64                    `json:"totalPage"`
	Items       []MarginStopOrderListItem `json:"items"`
}

// MarginStopOrdersByClientOrderID holds the stop orders Get Stop Order By ClientOid returns
type MarginStopOrdersByClientOrderID []MarginStopOrderResponse

// UnmarshalJSON decodes the stop orders as an array, as the docs' schema has them, or as a single object, as the docs'
// example has it
func (s *MarginStopOrdersByClientOrderID) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) != 0 && trimmed[0] == '{' {
		var stopOrder MarginStopOrderResponse
		if err := json.Unmarshal(trimmed, &stopOrder); err != nil {
			return err
		}
		*s = MarginStopOrdersByClientOrderID{stopOrder}
		return nil
	}
	return json.Unmarshal(data, (*[]MarginStopOrderResponse)(s))
}

// PlaceMarginOCOOrderRequest holds the parameters of Add OCO Order
type PlaceMarginOCOOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	Price         float64
	Size          float64
	// StopPrice is the trigger price
	StopPrice float64
	// LimitPrice is the price of the limit order placed once the order triggers
	LimitPrice float64
	// IsIsolated is always sent, as the API requires it
	IsIsolated bool
	AutoRepay  bool
	AutoBorrow bool
}

// PlaceMarginOCOOrderResponse holds the reply of Add OCO Order
type PlaceMarginOCOOrderResponse struct {
	OrderID string `json:"orderId"`
}

// CancelMarginOCOOrdersRequest holds the parameters of Batch Cancel OCO Order; KuCoin cancels every OCO order the
// set filters match, all of them when none is set
type CancelMarginOCOOrdersRequest struct {
	OrderIDs []string
	Symbol   string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType string
}

// MarginOCOOrderResponse holds a margin OCO order as Get OCO Order By OrderId, Get OCO Order By ClientOid and Get OCO
// Order List return it
type MarginOCOOrderResponse struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"clientOid"`
	OrderID       string `json:"orderId"`
	// OrderTime is in milliseconds
	OrderTime types.Time `json:"orderTime"`
	// Status is NEW, DONE, TRIGGERED or the cancelled status, which the docs spell both CANCELED and CANCELLED
	Status string `json:"status"`
}

// MarginOCOOrderDetailResponse holds the reply of Get OCO Order Detail By OrderId
type MarginOCOOrderDetailResponse struct {
	MarginOCOOrderResponse
	Orders []MarginOCOOrderLeg `json:"orders"`
}

// MarginOCOOrderLeg holds one of the two orders of a margin OCO order
type MarginOCOOrderLeg struct {
	ID        string       `json:"id"`
	Symbol    string       `json:"symbol"`
	Side      string       `json:"side"`
	Price     types.Number `json:"price"`
	StopPrice types.Number `json:"stopPrice"`
	Size      types.Number `json:"size"`
	Status    string       `json:"status"`
}

// GetMarginOCOOrdersRequest holds the parameters of Get OCO Order List
type GetMarginOCOOrdersRequest struct {
	// PageSize is 10 to 500
	PageSize uint64
	// CurrentPage starts at 1
	CurrentPage uint64
	Symbol      string
	StartAt     time.Time
	EndAt       time.Time
	// OrderIDs takes up to 500 order IDs
	OrderIDs []string
	// TradeType is MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin
	TradeType string
}

// MarginOCOOrderListResponse holds the reply of Get OCO Order List
type MarginOCOOrderListResponse struct {
	CurrentPage uint64                   `json:"currentPage"`
	PageSize    uint64                   `json:"pageSize"`
	TotalNumber uint64                   `json:"totalNum"`
	TotalPage   uint64                   `json:"totalPage"`
	Items       []MarginOCOOrderResponse `json:"items"`
}
