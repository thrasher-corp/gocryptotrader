package lbank

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceOrderRequest holds an order's parameters
type PlaceOrderRequest struct {
	Pair currency.Pair
	// Type is buy or sell for a limit order, or the side with a suffix: _market, _maker (post only), _ioc or _fok
	Type string
	// Price is the limit price, or a market buy's quote amount to spend. A market sell takes none
	Price float64
	// Amount is the base amount. A market buy takes none
	Amount   float64
	CustomID string
	// Window is how long LBank may wait to match the order before cancelling it, which it reads in milliseconds
	Window time.Duration
}

// PlaceOrderResponse holds a placed order's IDs
type PlaceOrderResponse struct {
	Symbol   string `json:"symbol"`
	OrderID  string `json:"order_id"`
	CustomID string `json:"custom_id"`
}

// OrderReferenceRequest names an order on a pair by its order ID or client order ID
type OrderReferenceRequest struct {
	Pair          currency.Pair
	OrderID       string
	ClientOrderID string
}

// CancelOrderResponse holds a cancelled order. Status is the order's status when the cancellation was accepted
type CancelOrderResponse struct {
	OrderID          string       `json:"orderId"`
	ClientOrderID    string       `json:"origClientOrderId"`
	TradeType        string       `json:"tradeType"`
	Price            types.Number `json:"price"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	// Status is -1 cancelled, 0 open, 1 partially filled, 2 filled, 3 partially filled and cancelled or 4 cancelling
	Status int64 `json:"status"`
}

// OrdersPageRequest holds an orders page request's parameters
type OrdersPageRequest struct {
	Pair currency.Pair
	// CurrentPage defaults to the first page
	CurrentPage uint64
	// PageLength is 1 to 200 orders, defaulting to 200
	PageLength uint64
	// Status filters QueryOrderHistory to one status: -1 cancelled, 0 open, 1 partially filled, 2 filled, 3 partially
	// filled and cancelled or 4 cancelling
	Status string
}

// OrdersPageResponse holds a page of orders
type OrdersPageResponse struct {
	// Total is the number of pages
	Total       uint64          `json:"total"`
	PageLength  uint64          `json:"page_length"`
	CurrentPage uint64          `json:"current_page"`
	Orders      []OrderResponse `json:"orders"`
}

// ordersPage holds a page of orders before its orders are decoded
type ordersPage struct {
	Total       uint64          `json:"total"`
	PageLength  uint64          `json:"page_length"`
	CurrentPage uint64          `json:"current_page"`
	Orders      json.RawMessage `json:"orders"`
}

// UnmarshalJSON decodes a page of orders. LBank has sent an empty string in place of an empty list of orders
func (o *OrdersPageResponse) UnmarshalJSON(data []byte) error {
	var p ordersPage
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = OrdersPageResponse{Total: p.Total, PageLength: p.PageLength, CurrentPage: p.CurrentPage}
	if len(p.Orders) == 0 || bytes.Equal(p.Orders, []byte(`""`)) || bytes.Equal(p.Orders, []byte("null")) {
		return nil
	}
	return json.Unmarshal(p.Orders, &o.Orders)
}

// OrderResponse holds an order
type OrderResponse struct {
	Symbol        string `json:"symbol"`
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	// Type is the order type the order was placed with, such as buy or sell_market
	Type  string       `json:"type"`
	Price types.Number `json:"price"`
	// OriginalQuantity is the base amount ordered, which a market buy leaves as zero
	OriginalQuantity types.Number `json:"origQty"`
	// OriginalQuoteOrderQuantity is the quote amount ordered
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	// Status is -1 cancelled, 0 open, 1 partially filled, 2 filled, 3 partially filled and cancelled or 4 cancelling
	Status     int64      `json:"status"`
	CreateTime types.Time `json:"time"`
	UpdateTime types.Time `json:"updateTime"`
}

// AccountInformationResponse holds the account's spot balances and permissions
type AccountInformationResponse struct {
	UID         string           `json:"uid"`
	Balances    []AccountBalance `json:"balances"`
	CanTrade    bool             `json:"canTrade"`
	CanWithdraw bool             `json:"canWithdraw"`
	CanDeposit  bool             `json:"canDeposit"`
}

// AccountBalance holds the account's balance of a coin
type AccountBalance struct {
	Asset  currency.Code `json:"asset"`
	Free   types.Number  `json:"free"`
	Locked types.Number  `json:"locked"`
}

// TransactionHistoryRequest holds a transaction history request's parameters
type TransactionHistoryRequest struct {
	Pair      currency.Pair
	StartTime time.Time
	EndTime   time.Time
	FromID    string
	// Limit is 1 to 100 trades, defaulting to 100
	Limit uint64
}

// TransactionResponse holds one of the account's trades
type TransactionResponse struct {
	ID            string       `json:"id"`
	Symbol        string       `json:"symbol"`
	OrderID       string       `json:"orderId"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	// Commission is charged in the coin the trade buys
	Commission types.Number `json:"commission"`
	IsMaker    bool         `json:"isMaker"`
	IsBuyer    bool         `json:"isBuyer"`
	Time       types.Time   `json:"time"`
}
