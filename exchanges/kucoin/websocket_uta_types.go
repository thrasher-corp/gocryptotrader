package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAWebsocketAuthRequest is the authentication message a UTA private push or trade connection sends after connecting
type UTAWebsocketAuthRequest struct {
	// ID is echoed by the private push connection's reply; the trade connection's reply carries none
	ID         string `json:"id,omitempty"`
	Operation  string `json:"op"`
	APIKey     string `json:"kc-api-key"`
	Timestamp  int64  `json:"kc-api-timestamp"` // milliseconds
	Signature  string `json:"kc-api-sign"`
	Passphrase string `json:"kc-api-passphrase"`
}

// UTAWebsocketAckResponse is the reply to a subscribe, unsubscribe or private push authentication message
type UTAWebsocketAckResponse struct {
	ID string `json:"id"`
	// Result arrives as a JSON boolean, though the docs show it as a string
	Result  types.Boolean `json:"result"`
	Message string        `json:"message"`
	// Reason replaces Message on some rejections, such as an orderbook depth the RPI filter does not support
	Reason string `json:"reason"`
}

// UTAWebsocketSubscriptionRequest subscribes to or unsubscribes from a UTA websocket channel
type UTAWebsocketSubscriptionRequest struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Channel     string `json:"channel"`
	TradeType   string `json:"tradeType,omitempty"`
	AccountType string `json:"accountType,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
	Interval    string `json:"interval,omitempty"`
	Depth       string `json:"depth,omitempty"`
}

// UTAWebsocketMessage holds the fields that identify an incoming UTA push connection frame: pushes carry a topic and
// data, replies carry the request ID, and the welcome frame and pongs carry a message or operation
type UTAWebsocketMessage struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Message   string `json:"message"`
	// Topic is the channel followed by the trade or account type, such as ticker.SPOT or balance.UNIFIED
	Topic         string     `json:"T"`
	PushTimestamp types.Time `json:"P"`
	// UpdateType is snapshot or delta, on orderbook and call auction pushes
	UpdateType string `json:"t"`
	// Depth is the subscribed orderbook depth, on orderbook pushes
	Depth string          `json:"dp"`
	Data  json.RawMessage `json:"d"`
}
