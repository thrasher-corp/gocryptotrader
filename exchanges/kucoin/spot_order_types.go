package kucoin

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceHFOrderRequest holds the parameters of Add Order, which Add Order Sync, Add Order Test, the batch order
// endpoints and the Classic websocket spot.order and spot.sync_order operations take as well
type PlaceHFOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	// Type is limit, which needs Price and Size, or market, which needs Size or Funds
	Type                string
	Remark              string
	SelfTradePrevention string
	Price               float64
	// Size is in the base currency
	Size        float64
	TimeInForce string
	PostOnly    bool
	Tags        string
	// CancelAfter cancels a GTT order after this many seconds; zero keeps KuCoin's default of never
	CancelAfter uint64
	// Funds is the quote currency amount of a market order
	Funds float64
	// AllowMaxTimeWindow, in milliseconds, makes KuCoin reject the order when it arrives later than ClientTimestamp
	// plus the window
	AllowMaxTimeWindow uint64
	// ClientTimestamp is required with AllowMaxTimeWindow; KuCoin expects it to equal the request's KC-API-TIMESTAMP
	ClientTimestamp time.Time
}

// Validate checks the parameters Add Order requires
func (r *PlaceHFOrderRequest) Validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Symbol == "":
		return currency.ErrSymbolStringEmpty
	case r.Side == "":
		return order.ErrSideIsInvalid
	case r.Type == "":
		return order.ErrTypeIsInvalid
	case r.AllowMaxTimeWindow != 0 && r.ClientTimestamp.IsZero():
		return errClientTimestampRequired
	}
	return validateOrderAmounts(r.Type, r.Price, r.Size, r.Funds)
}

// validateOrderAmounts checks the amounts KuCoin requires of an order type: a limit order needs a price and a size,
// and a market order a size or funds but not both. Other types are left for KuCoin to validate
func validateOrderAmounts(orderType string, price, size, funds float64) error {
	switch orderType {
	case "limit":
		if price <= 0 {
			return limits.ErrPriceBelowMin
		}
		if size <= 0 {
			return limits.ErrAmountBelowMin
		}
	case "market":
		if size != 0 && funds != 0 {
			return errSizeAndFundsBothSet
		}
		if size <= 0 && funds <= 0 {
			return errSizeOrFundIsRequired
		}
	}
	return nil
}

// Payload validates the request and returns the Add Order body, which the REST and websocket order endpoints share
func (r *PlaceHFOrderRequest) Payload() (map[string]any, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["side"] = r.Side
	body["type"] = r.Type
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
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
	if r.Tags != "" {
		body["tags"] = r.Tags
	}
	if r.CancelAfter != 0 {
		body["cancelAfter"] = r.CancelAfter
	}
	if r.Funds != 0 {
		body["funds"] = strconv.FormatFloat(r.Funds, 'f', -1, 64)
	}
	if r.AllowMaxTimeWindow != 0 {
		body["allowMaxTimeWindow"] = r.AllowMaxTimeWindow
	}
	if !r.ClientTimestamp.IsZero() {
		body["clientTimestamp"] = r.ClientTimestamp.UnixMilli()
	}
	return body, nil
}

// PlaceHFOrderResponse holds the IDs of an order KuCoin accepted (Add Order, Add Order Test)
type PlaceHFOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}

// PlaceHFOrderSyncResponse holds an order's state once matching has finished (Add Order Sync)
type PlaceHFOrderSyncResponse struct {
	OrderID       string       `json:"orderId"`
	ClientOrderID string       `json:"clientOid"`
	OrderTime     types.Time   `json:"orderTime"`
	OriginSize    types.Number `json:"originSize"`
	DealSize      types.Number `json:"dealSize"`
	RemainSize    types.Number `json:"remainSize"`
	CanceledSize  types.Number `json:"canceledSize"`
	Status        string       `json:"status"`
	MatchTime     types.Time   `json:"matchTime"`
}

// BatchPlacedHFOrderResponse is the outcome of one order of a Batch Add Orders request. KuCoin rejected the order when
// Success is false, and FailMessage says why
type BatchPlacedHFOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	Success       bool   `json:"success"`
	FailMessage   string `json:"failMsg"`
}

// BatchPlacedHFOrderSyncResponse is the outcome of one order of a Batch Add Orders Sync request, with the order's state once
// matching has finished. KuCoin rejected the order when Success is false, and FailMessage says why
type BatchPlacedHFOrderSyncResponse struct {
	OrderID       string       `json:"orderId"`
	ClientOrderID string       `json:"clientOid"`
	OrderTime     types.Time   `json:"orderTime"`
	OriginSize    types.Number `json:"originSize"`
	DealSize      types.Number `json:"dealSize"`
	RemainSize    types.Number `json:"remainSize"`
	CanceledSize  types.Number `json:"canceledSize"`
	Status        string       `json:"status"`
	MatchTime     types.Time   `json:"matchTime"`
	Success       bool         `json:"success"`
	FailMessage   string       `json:"failMsg"`
}

// OrderIDResponse holds the ID of the order a request acted on (Cancel Order By OrderId, Add Stop Order,
// Add OCO Order)
type OrderIDResponse struct {
	OrderID string `json:"orderId"`
}

// CancelHFOrderByOrderIDSyncResponse holds an order's state once its cancellation has finished
// (Cancel Order By OrderId Sync)
type CancelHFOrderByOrderIDSyncResponse struct {
	OrderID      string       `json:"orderId"`
	OriginSize   types.Number `json:"originSize"`
	DealSize     types.Number `json:"dealSize"`
	RemainSize   types.Number `json:"remainSize"`
	CanceledSize types.Number `json:"canceledSize"`
	Status       string       `json:"status"`
}

// ClientOrderIDResponse holds the client order ID of the order a request acted on (Cancel Order By ClientOid)
type ClientOrderIDResponse struct {
	ClientOrderID string `json:"clientOid"`
}

// CancelHFOrderByClientOrderIDSyncResponse holds an order's state once its cancellation has finished
// (Cancel Order By ClientOid Sync)
type CancelHFOrderByClientOrderIDSyncResponse struct {
	ClientOrderID string       `json:"clientOid"`
	OriginSize    types.Number `json:"originSize"`
	DealSize      types.Number `json:"dealSize"`
	RemainSize    types.Number `json:"remainSize"`
	CanceledSize  types.Number `json:"canceledSize"`
	Status        string       `json:"status"`
}

// CancelPartialHFOrderResponse holds the size cancelled from an order (Cancel Partial Order)
type CancelPartialHFOrderResponse struct {
	OrderID    string       `json:"orderId"`
	CancelSize types.Number `json:"cancelSize"`
}

// CancelAllHFOrdersResponse holds the symbols whose orders were cancelled and those that failed (Cancel All Orders)
type CancelAllHFOrdersResponse struct {
	SucceedSymbols []string                        `json:"succeedSymbols"`
	FailedSymbols  []FailedHFOrderCancellationInfo `json:"failedSymbols"`
}

// FailedHFOrderCancellationInfo holds a symbol whose orders KuCoin failed to cancel, and why
type FailedHFOrderCancellationInfo struct {
	Symbol string `json:"symbol"`
	Error  string `json:"error"`
}

// ModifyHFOrderRequest holds the parameters of Modify Order: the order, by OrderID or ClientOrderID, and its new
// price or size
type ModifyHFOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
	NewPrice      float64
	NewSize       float64
}

// ModifyHFOrderResponse holds the ID of the order that replaced a modified order (Modify Order)
type ModifyHFOrderResponse struct {
	NewOrderID string `json:"newOrderId"`
	// ClientOrderID is the client order ID of the modified order
	ClientOrderID string `json:"clientOid"`
}

// HFOrderDetailResponse holds a spot order and why it was cancelled (Get Order By OrderId, Get Order By ClientOid,
// Get Closed Orders)
type HFOrderDetailResponse struct {
	HFOrderResponse
	// CancelReason is KuCoin's cancellation reason code; 0 means the user cancelled the order
	CancelReason uint64 `json:"cancelReason"`
}

// HFOrderResponse holds a spot order (Get Open Orders, Get Open Orders By Page)
type HFOrderResponse struct {
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
	TimeInForce         string        `json:"timeInForce"`
	PostOnly            bool          `json:"postOnly"`
	Hidden              bool          `json:"hidden"`
	Iceberg             bool          `json:"iceberg"`
	VisibleSize         types.Number  `json:"visibleSize"`
	// CancelAfter is the number of seconds after which a GTT order is cancelled
	CancelAfter    int64        `json:"cancelAfter"`
	Channel        string       `json:"channel"`
	ClientOrderID  string       `json:"clientOid"`
	Remark         string       `json:"remark"`
	Tags           string       `json:"tags"`
	CancelExist    bool         `json:"cancelExist"`
	CreatedAt      types.Time   `json:"createdAt"`
	LastUpdatedAt  types.Time   `json:"lastUpdatedAt"`
	TradeType      string       `json:"tradeType"`
	InOrderBook    bool         `json:"inOrderBook"`
	CancelledSize  types.Number `json:"cancelledSize"`
	CancelledFunds types.Number `json:"cancelledFunds"`
	RemainSize     types.Number `json:"remainSize"`
	RemainFunds    types.Number `json:"remainFunds"`
	Tax            types.Number `json:"tax"`
	Active         bool         `json:"active"`
}

// SymbolsWithOpenHFOrderResponse holds the symbols that have open spot orders (Get Symbols With Open Order)
type SymbolsWithOpenHFOrderResponse struct {
	Symbols []string `json:"symbols"`
}

// OpenHFOrdersPageResponse holds a page of open spot orders (Get Open Orders By Page)
type OpenHFOrdersPageResponse struct {
	CurrentPage uint64            `json:"currentPage"`
	PageSize    uint64            `json:"pageSize"`
	TotalNumber uint64            `json:"totalNum"`
	TotalPage   uint64            `json:"totalPage"`
	Items       []HFOrderResponse `json:"items"`
}

// ClosedHFOrdersRequest holds the parameters of Get Closed Orders
type ClosedHFOrdersRequest struct {
	Symbol string
	Side   string
	Type   string
	// LastID pages backwards from a previous response's LastID
	LastID  uint64
	Limit   uint64
	StartAt time.Time
	EndAt   time.Time
}

// ClosedHFOrdersResponse holds closed spot orders (Get Closed Orders)
type ClosedHFOrdersResponse struct {
	// LastID is passed as ClosedHFOrdersRequest.LastID to fetch the next page
	LastID uint64                  `json:"lastId"`
	Items  []HFOrderDetailResponse `json:"items"`
}

// HFTradeHistoryRequest holds the parameters of Get Trade History
type HFTradeHistoryRequest struct {
	Symbol  string
	OrderID string
	Side    string
	Type    string
	// LastID pages backwards from a previous response's LastID
	LastID  uint64
	Limit   uint64
	StartAt time.Time
	EndAt   time.Time
}

// HFTradeHistoryResponse holds spot fills (Get Trade History)
type HFTradeHistoryResponse struct {
	Items []HFTrade `json:"items"`
	// LastID is passed as HFTradeHistoryRequest.LastID to fetch the next page
	LastID uint64 `json:"lastId"`
}

// HFTrade holds a spot fill
type HFTrade struct {
	ID     uint64 `json:"id"`
	Symbol string `json:"symbol"`
	// TradeID increases per symbol
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
	TaxRate        types.Number  `json:"taxRate"`
	Tax            types.Number  `json:"tax"`
	Type           string        `json:"type"`
	CreatedAt      types.Time    `json:"createdAt"`
}

// DCPResponse holds the disconnection protection setting (Get DCP)
type DCPResponse struct {
	// Timeout is in seconds, -1 when unset
	Timeout int64 `json:"timeout"`
	// Symbols is a comma separated list, empty for every symbol
	Symbols     string     `json:"symbols"`
	CurrentTime types.Time `json:"currentTime"`
	TriggerTime types.Time `json:"triggerTime"`
}

// SetDCPResponse holds when the disconnection protection set triggers (Set DCP)
type SetDCPResponse struct {
	CurrentTime types.Time `json:"currentTime"`
	TriggerTime types.Time `json:"triggerTime"`
}

// PlaceStopOrderRequest holds the parameters of Add Stop Order
type PlaceStopOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	// Type is limit, which needs Price and Size, or market, which needs Size or Funds
	Type                string
	Remark              string
	SelfTradePrevention string
	Price               float64
	// Size is in the base currency
	Size        float64
	TimeInForce string
	PostOnly    bool
	// CancelAfter cancels a GTT order after this many seconds; zero keeps KuCoin's default of never
	CancelAfter uint64
	// Funds is the quote currency amount of a market order
	Funds     float64
	StopPrice float64
	// TradeType is TRADE (the default) for spot, MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated
	// margin
	TradeType string
	// Stop is loss (the default) to trigger as the price falls to StopPrice, or entry to trigger as it rises to it
	Stop string
}

// CancelledOrderIDsResponse holds the IDs of the orders a cancellation request cancelled (Batch Cancel Stop Orders,
// Cancel OCO Order By OrderId, Cancel OCO Order By ClientOid, Batch Cancel OCO Order)
type CancelledOrderIDsResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
}

// BatchCancelStopOrdersRequest holds the parameters of Batch Cancel Stop Orders; zero values do not filter
type BatchCancelStopOrdersRequest struct {
	Symbol    string
	TradeType string
	OrderIDs  []string
}

// StopOrderListRequest holds the parameters of Get Stop Orders List; zero values do not filter
type StopOrderListRequest struct {
	Symbol      string
	Side        string
	Type        string
	TradeType   string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	OrderIDs    []string
	PageSize    uint64
	// Stop lists stop orders (stop) or OCO orders (oco)
	Stop string
}

// StopOrderListResponse holds a page of untriggered stop orders (Get Stop Orders List)
type StopOrderListResponse struct {
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []StopOrderListItem `json:"items"`
}

// StopOrderListItem holds a stop order of Get Stop Orders List, which returns fields Get Stop Order By OrderId does
// not
type StopOrderListItem struct {
	StopOrderResponse
	RelatedNumber     string       `json:"relatedNo"`
	LimitPrice        types.Number `json:"limitPrice"`
	Pop               string       `json:"pop"`
	ActivateCondition string       `json:"activateCondition"`
}

// StopOrderResponse holds a stop order (Get Stop Order By OrderId, Get Stop Order By ClientOid)
type StopOrderResponse struct {
	ID                  string       `json:"id"`
	Symbol              string       `json:"symbol"`
	UserID              string       `json:"userId"`
	Status              string       `json:"status"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	Price               types.Number `json:"price"`
	Size                types.Number `json:"size"`
	Funds               types.Number `json:"funds"`
	SelfTradePrevention string       `json:"stp"`
	TimeInForce         string       `json:"timeInForce"`
	// CancelAfter is the number of seconds after which a GTT order is cancelled, -1 for never
	CancelAfter   int64         `json:"cancelAfter"`
	PostOnly      bool          `json:"postOnly"`
	Hidden        bool          `json:"hidden"`
	Iceberg       bool          `json:"iceberg"`
	VisibleSize   types.Number  `json:"visibleSize"`
	Channel       string        `json:"channel"`
	ClientOrderID string        `json:"clientOid"`
	Remark        string        `json:"remark"`
	Tags          string        `json:"tags"`
	DomainID      string        `json:"domainId"`
	TradeSource   string        `json:"tradeSource"`
	TradeType     string        `json:"tradeType"`
	FeeCurrency   currency.Code `json:"feeCurrency"`
	TakerFeeRate  types.Number  `json:"takerFeeRate"`
	MakerFeeRate  types.Number  `json:"makerFeeRate"`
	CreatedAt     types.Time    `json:"createdAt"`
	// Stop is loss when the order triggers as the price falls to StopPrice, entry as it rises to it
	Stop            string       `json:"stop"`
	StopTriggerTime types.Time   `json:"stopTriggerTime"`
	StopPrice       types.Number `json:"stopPrice"`
	OrderTime       types.Time   `json:"orderTime"`
}

// PlaceOCOOrderRequest holds the parameters of Add OCO Order: a limit order at Price, and a limit order at LimitPrice
// that StopPrice triggers; whichever executes first cancels the other
type PlaceOCOOrderRequest struct {
	ClientOrderID string
	Side          string
	Symbol        string
	Remark        string
	Price         float64
	// Size is in the base currency
	Size       float64
	StopPrice  float64
	LimitPrice float64
	// TradeType is TRADE, the only value KuCoin accepts
	TradeType string
}

// OCOOrderListRequest holds the parameters of Get OCO Order List; zero values do not filter and leave KuCoin's paging
// defaults
type OCOOrderListRequest struct {
	Symbol      string
	StartAt     time.Time
	EndAt       time.Time
	OrderIDs    []string
	PageSize    uint64
	CurrentPage uint64
}

// OCOOrderListResponse holds a page of OCO orders (Get OCO Order List)
type OCOOrderListResponse struct {
	CurrentPage uint64     `json:"currentPage"`
	PageSize    uint64     `json:"pageSize"`
	TotalNumber uint64     `json:"totalNum"`
	TotalPage   uint64     `json:"totalPage"`
	Items       []OCOOrder `json:"items"`
}

// OCOOrder holds an OCO order
type OCOOrder struct {
	OrderID       string     `json:"orderId"`
	Symbol        string     `json:"symbol"`
	ClientOrderID string     `json:"clientOid"`
	OrderTime     types.Time `json:"orderTime"`
	Status        string     `json:"status"`
}
