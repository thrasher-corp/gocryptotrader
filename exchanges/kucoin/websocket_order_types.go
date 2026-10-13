package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsOrderSession is the session message the websocket order API sends once connected, which the client signs to
// authenticate
type WsOrderSession struct {
	SessionID string     `json:"sessionId"`
	Timestamp types.Time `json:"timestamp"`
}

// WsOrderWelcome is the message the websocket order API sends once a session is authenticated
type WsOrderWelcome struct {
	SessionID    string `json:"sessionId"`
	Data         string `json:"data"`
	PingInterval int64  `json:"pingInterval"` // Milliseconds
	PingTimeout  int64  `json:"pingTimeout"`  // Milliseconds
}

// WsOrderMessage is a websocket order API request
type WsOrderMessage struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Args      any    `json:"args"`
}

// WsOrderResponse is a websocket order API reply: an operation's result, an error or a pong
type WsOrderResponse struct {
	Code          string           `json:"code"`
	Message       string           `json:"msg"`
	ID            string           `json:"id"`
	Operation     string           `json:"op"`
	InTime        types.Time       `json:"inTime"`
	OutTime       types.Time       `json:"outTime"`
	UserRateLimit WsOrderRateLimit `json:"userRateLimit"`
	RateLimit     WsOrderRateLimit `json:"rateLimit"` // Sent on errors
	Timestamp     types.Time       `json:"timestamp"` // Pong server time
	Data          json.RawMessage  `json:"data"`
}

// WsOrderRateLimit is the rate limit state a websocket order API reply reports
type WsOrderRateLimit struct {
	Limit     uint64 `json:"limit"`
	Remaining uint64 `json:"remaining"`
	Reset     uint64 `json:"reset"` // Milliseconds
}

// WsAddOrderResponse is the result of a spot.order or futures.order request
type WsAddOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
	OrderID       string `json:"orderId"`
}

// WsAddSpotOrderSyncResponse is the result of a spot.sync_order request
type WsAddSpotOrderSyncResponse struct {
	CanceledSize  types.Number `json:"canceledSize"`
	ClientOrderID string       `json:"clientOid"`
	DealSize      types.Number `json:"dealSize"`
	MatchTime     types.Time   `json:"matchTime"`
	OrderID       string       `json:"orderId"`
	OrderTime     types.Time   `json:"orderTime"`
	OriginSize    types.Number `json:"originSize"`
	RemainSize    types.Number `json:"remainSize"`
	Status        string       `json:"status"`
}

// WsAddMarginOrderResponse is the result of a margin.order request
type WsAddMarginOrderResponse struct {
	BorrowSize    types.Number `json:"borrowSize"`
	ClientOrderID string       `json:"clientOid"`
	LoanApplyID   string       `json:"loanApplyId"`
	OrderID       string       `json:"orderId"`
}

// WsFuturesOrderResult is the result of one order of a futures.multi_order or futures.multi_cancel request
type WsFuturesOrderResult struct {
	ClientOrderID string `json:"clientOid"`
	Code          string `json:"code"`
	Message       string `json:"msg"`
	OrderID       string `json:"orderId"`
	Symbol        string `json:"symbol"`
}

// WsCancelOrderRequest names the order a spot.cancel, spot.sync_cancel, margin.cancel or futures.cancel request cancels;
// when both IDs are given the client order ID takes effect
type WsCancelOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}

// WsCancelOrderResponse is the result of a spot.cancel or margin.cancel request: the ID the order was cancelled by
type WsCancelOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
	OrderID       string `json:"orderId"`
}

// WsCancelSpotOrderSyncResponse is the result of a spot.sync_cancel request
type WsCancelSpotOrderSyncResponse struct {
	CanceledSize  types.Number `json:"canceledSize"`
	ClientOrderID string       `json:"clientOid"`
	DealSize      types.Number `json:"dealSize"`
	OrderID       string       `json:"orderId"`
	OriginSize    types.Number `json:"originSize"`
	RemainSize    types.Number `json:"remainSize"`
	Status        string       `json:"status"`
}

// WsCancelFuturesOrderResponse is the result of a futures.cancel request: the cancelled order IDs when cancelled by
// order ID, or the client order ID
type WsCancelFuturesOrderResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
	ClientOrderID     string   `json:"clientOid"`
}

// WsBatchCancelFuturesOrdersRequest names the futures orders a futures.multi_cancel request cancels, by order ID or by
// symbol and client order ID
type WsBatchCancelFuturesOrdersRequest struct {
	OrderIDs     []string
	ClientOrders []WsFuturesClientOrder
}

// WsFuturesClientOrder names a futures order by its symbol and client order ID
type WsFuturesClientOrder struct {
	Symbol        string
	ClientOrderID string
}
