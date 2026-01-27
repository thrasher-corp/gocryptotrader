package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketAPIResponse defines a general websocket response for api calls
type WebsocketAPIResponse struct {
	Header Header          `json:"header"`
	Data   json.RawMessage `json:"data"`
}

// Header defines a websocket header
type Header struct {
	ResponseTime types.Time `json:"response_time"`
	Status       int64      `json:"status,string"`
	Channel      string     `json:"channel"`
	Event        string     `json:"event"`
	ClientID     string     `json:"client_id"`
	ConnectionID string     `json:"conn_id"`
	TraceID      string     `json:"trace_id"`
}

// WebsocketRequest defines a websocket request
type WebsocketRequest struct {
	Time    int64            `json:"time,omitempty"`
	ID      int64            `json:"id,omitempty"`
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Payload WebsocketPayload `json:"payload"`
}

// WebsocketPayload defines an individualised websocket payload
type WebsocketPayload struct {
	RequestID string `json:"req_id,omitempty"`
	// APIKey and signature are only required in the initial login request
	// which is done when the connection is established.
	APIKey       string          `json:"api_key,omitempty"`
	Timestamp    string          `json:"timestamp,omitempty"`
	Signature    string          `json:"signature,omitempty"`
	RequestParam json.RawMessage `json:"req_param,omitempty"`
}

// WebsocketErrors defines a websocket error
type WebsocketErrors struct {
	Errors struct {
		Label   string `json:"label"`
		Message string `json:"message"`
	} `json:"errs"`
}

// WebsocketOrderBatchRequest defines a websocket order batch request
type WebsocketOrderBatchRequest struct {
	OrderID string        `json:"id"` // This requires id tag not order_id
	Pair    currency.Pair `json:"currency_pair"`
	Account string        `json:"account,omitempty"`
}

// WebsocketOrderRequest defines a websocket order request
type WebsocketOrderRequest struct {
	OrderID string `json:"order_id"` // This requires order_id tag
	Pair    string `json:"currency_pair"`
	Account string `json:"account,omitempty"`
}

// WebsocketCancellAllResponse defines a websocket order cancel response
type WebsocketCancellAllResponse struct {
	Pair      currency.Pair `json:"currency_pair"`
	Label     string        `json:"label"`
	Message   string        `json:"message"`
	Succeeded bool          `json:"succeeded"`
}

// WebsocketCancelParam is a struct to hold the parameters for cancelling orders
type WebsocketCancelParam struct {
	Pair    currency.Pair `json:"pair"`
	Side    string        `json:"side"`
	Account string        `json:"account,omitempty"`
}

// WebsocketAmendOrder defines a websocket amend order
type WebsocketAmendOrder struct {
	OrderID   string        `json:"order_id"`
	Pair      currency.Pair `json:"currency_pair"`
	Account   string        `json:"account,omitempty"`
	AmendText string        `json:"amend_text,omitempty"`
	Price     string        `json:"price,omitempty"`
	Amount    string        `json:"amount,omitempty"`
}

// WebsocketFuturesAmendOrder defines a websocket amend order
type WebsocketFuturesAmendOrder struct {
	OrderID   string        `json:"order_id"`
	Contract  currency.Pair `json:"-"` // Only used internally for routing
	Asset     asset.Item    `json:"-"` // Only used internally for routing
	AmendText string        `json:"amend_text,omitempty"`
	Price     string        `json:"price,omitempty"`
	Size      int64         `json:"size,omitempty"`
}

// WebsocketFutureOrdersList defines a websocket future orders list
type WebsocketFutureOrdersList struct {
	Contract currency.Pair `json:"contract,omitzero"`
	Asset    asset.Item    `json:"-"` // Only used internally for routing
	Status   string        `json:"status"`
	Limit    int64         `json:"limit,omitempty"`
	Offset   int64         `json:"offset,omitempty"`
	LastID   string        `json:"last_id,omitempty"`
}
