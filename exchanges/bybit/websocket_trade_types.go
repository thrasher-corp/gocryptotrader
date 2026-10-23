package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketTradeRequest is a request on the trade connection: authentication, or an order create, amend or cancel
// and their batch forms (Websocket Trade Guideline)
type WebsocketTradeRequest struct {
	RequestID string                       `json:"reqId,omitempty"`
	Header    *WebsocketTradeRequestHeader `json:"header,omitempty"`
	Operation string                       `json:"op"`
	Arguments []any                        `json:"args"`
}

// WebsocketTradeRequestHeader holds the headers of an order request
type WebsocketTradeRequestHeader struct {
	Timestamp string `json:"X-BAPI-TIMESTAMP"` // Unix milliseconds
	// ReceiveWindow is in milliseconds, 5000 when empty
	ReceiveWindow string `json:"X-BAPI-RECV-WINDOW,omitempty"`
	// Referer identifies an API broker
	Referer string `json:"Referer,omitempty"`
}

// WebsocketTradeResponse is the reply to a request on the trade connection; T is the type of its data, which is the
// REST endpoint's result for an order request and the server time for a pong
type WebsocketTradeResponse[T any] struct {
	RequestID          string                       `json:"reqId"`
	ReturnCode         int64                        `json:"retCode"` // Bybit's error codes include negative ones, such as -1 and -2015
	ReturnMessage      string                       `json:"retMsg"`
	Operation          string                       `json:"op"`
	Data               T                            `json:"data"`
	ReturnExtendedInfo WebsocketTradeExtraInfo      `json:"retExtInfo"`
	Header             WebsocketTradeResponseHeader `json:"header"`
	ConnectionID       string                       `json:"connId"`
}

// WebsocketTradeExtraInfo holds the outcome of each order of a batch request, in the order the request listed them;
// it is empty for other requests
type WebsocketTradeExtraInfo struct {
	List []WebsocketBatchOrderStatus `json:"list"`
}

// WebsocketBatchOrderStatus is the outcome of one order of a batch request; a code of 0 means it was accepted
type WebsocketBatchOrderStatus struct {
	Code    int64  `json:"code"` // Bybit's error codes include negative ones, such as -1 and -2015
	Message string `json:"msg"`
}

// WebsocketTradeResponseHeader holds the trace ID and rate limit state of a reply to an order request
type WebsocketTradeResponseHeader struct {
	TraceID string     `json:"Traceid"`
	TimeNow types.Time `json:"Timenow"`
	// Limit is the account's rate limit for the operation and LimitStatus what remains of it
	Limit       types.Number `json:"X-Bapi-Limit"`
	LimitStatus types.Number `json:"X-Bapi-Limit-Status"`
	// LimitResetTimestamp is when an exceeded limit resets, and the current time otherwise
	LimitResetTimestamp types.Time `json:"X-Bapi-Limit-Reset-Timestamp"`
}

// WebsocketBatchOrderResponse is the data of the reply to a batch order request
type WebsocketBatchOrderResponse struct {
	List []WebsocketBatchOrderAcknowledgement `json:"list"`
}

// WebsocketBatchOrderAcknowledgement is the result of one order of a batch request
type WebsocketBatchOrderAcknowledgement struct {
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	// CreateAt is set by batch create requests only
	CreateAt types.Time `json:"createAt"`
}

// WebsocketBatchOrderResult is the result and outcome of one order of a batch request
type WebsocketBatchOrderResult struct {
	Category    string
	Symbol      string
	OrderID     string
	OrderLinkID string
	CreateAt    time.Time
	// Code is 0 when Bybit accepted the order and Message explains a rejection
	Code    int64
	Message string
}
