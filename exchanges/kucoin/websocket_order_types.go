package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketOrderSession is the session message the websocket order API sends once connected, which the client
// signs to authenticate
type WebsocketOrderSession struct {
	SessionID string     `json:"sessionId"`
	Timestamp types.Time `json:"timestamp"`
}

// WebsocketOrderWelcome is the message the websocket order API sends once a session is authenticated
type WebsocketOrderWelcome struct {
	SessionID    string `json:"sessionId"`
	Data         string `json:"data"`
	PingInterval uint64 `json:"pingInterval"` // Milliseconds
	PingTimeout  uint64 `json:"pingTimeout"`  // Milliseconds
}

// WebsocketOrderRequest is a websocket order API request
type WebsocketOrderRequest struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Args      any    `json:"args"`
}

// WebsocketOrderResponse is a websocket order API reply: an operation's result, an error or a pong
type WebsocketOrderResponse struct {
	Code          string                  `json:"code"`
	Message       string                  `json:"msg"`
	ID            string                  `json:"id"`
	Operation     string                  `json:"op"`
	InTime        types.Time              `json:"inTime"`
	OutTime       types.Time              `json:"outTime"`
	UserRateLimit WebsocketOrderRateLimit `json:"userRateLimit"`
	RateLimit     WebsocketOrderRateLimit `json:"rateLimit"` // Sent on errors
	Timestamp     types.Time              `json:"timestamp"` // Pong server time
	Data          json.RawMessage         `json:"data"`
}

// WebsocketOrderRateLimit is the rate limit state a websocket order API reply reports
type WebsocketOrderRateLimit struct {
	Limit     uint64 `json:"limit"`
	Remaining uint64 `json:"remaining"`
	Reset     uint64 `json:"reset"` // Milliseconds
}

// WebsocketAddOrderResponse is the result of a spot.order or futures.order request
type WebsocketAddOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
	OrderID       string `json:"orderId"`
}

// WebsocketAddSpotOrderSyncResponse is the result of a spot.sync_order request
type WebsocketAddSpotOrderSyncResponse struct {
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

// WebsocketAddMarginOrderResponse is the result of a margin.order request
type WebsocketAddMarginOrderResponse struct {
	BorrowSize    types.Number `json:"borrowSize"`
	ClientOrderID string       `json:"clientOid"`
	LoanApplyID   string       `json:"loanApplyId"`
	OrderID       string       `json:"orderId"`
}

// WebsocketFuturesOrderResponse is the result of one order of a futures.multi_order or futures.multi_cancel request
type WebsocketFuturesOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
	Code          string `json:"code"`
	Message       string `json:"msg"`
	OrderID       string `json:"orderId"`
	Symbol        string `json:"symbol"`
}

// WebsocketCancelOrderRequest names the order a spot.cancel, spot.sync_cancel, margin.cancel or futures.cancel request cancels;
// when both IDs are given the client order ID takes effect
type WebsocketCancelOrderRequest struct {
	Symbol        string
	OrderID       string
	ClientOrderID string
}

// WebsocketCancelOrderResponse is the result of a spot.cancel or margin.cancel request: the ID the order was cancelled by
type WebsocketCancelOrderResponse struct {
	ClientOrderID string `json:"clientOid"`
	OrderID       string `json:"orderId"`
}

// WebsocketCancelSpotOrderSyncResponse is the result of a spot.sync_cancel request
type WebsocketCancelSpotOrderSyncResponse struct {
	CanceledSize  types.Number `json:"canceledSize"`
	ClientOrderID string       `json:"clientOid"`
	DealSize      types.Number `json:"dealSize"`
	OrderID       string       `json:"orderId"`
	OriginSize    types.Number `json:"originSize"`
	RemainSize    types.Number `json:"remainSize"`
	Status        string       `json:"status"`
}

// WebsocketCancelFuturesOrderResponse is the result of a futures.cancel request: the cancelled order IDs when cancelled by
// order ID, or the client order ID
type WebsocketCancelFuturesOrderResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
	ClientOrderID     string   `json:"clientOid"`
}

// WebsocketBatchCancelFuturesOrdersRequest names the futures orders a futures.multi_cancel request cancels, by order ID or by
// symbol and client order ID
type WebsocketBatchCancelFuturesOrdersRequest struct {
	OrderIDs     []string
	ClientOrders []WebsocketFuturesClientOrder
}

// WebsocketFuturesClientOrder names a futures order by its symbol and client order ID
type WebsocketFuturesClientOrder struct {
	Symbol        string
	ClientOrderID string
}
