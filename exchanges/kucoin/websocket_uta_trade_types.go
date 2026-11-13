package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAWebsocketTradeAuthResponse is the reply to the UTA trade connection's authentication message: a welcome frame
// once signed in, or a code and message when rejected
type UTAWebsocketTradeAuthResponse struct {
	SessionID string `json:"sessionId"`
	// Data is welcome once signed in
	Data         string     `json:"data"`
	PingInterval uint64     `json:"pingInterval"` // milliseconds
	PingTimeout  uint64     `json:"pingTimeout"`  // milliseconds
	Code         string     `json:"code"`
	Message      string     `json:"msg"`
	InTime       types.Time `json:"inTime"`
	OutTime      types.Time `json:"outTime"`
}

// UTAWebsocketTradeRequest is an order operation sent over the UTA trade connection
type UTAWebsocketTradeRequest struct {
	// ID may be at most 32 bytes long
	ID        string         `json:"id"`
	Operation string         `json:"op"`
	Arguments map[string]any `json:"args"`
}

// UTAWebsocketTradeStatus holds the fields every reply on the UTA trade connection carries
type UTAWebsocketTradeStatus struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Code      string `json:"code"`
	Message   string `json:"msg"`
	// InTime and OutTime are when the gateway received and answered the request: in milliseconds, or nanoseconds for
	// connections which ask for them
	InTime        types.Time            `json:"inTime"`
	OutTime       types.Time            `json:"outTime"`
	UserRateLimit UTAWebsocketRateLimit `json:"userRateLimit"`
}

// utaTradeReply is a reply on the UTA trade connection, all of which carry the trade status
type utaTradeReply interface {
	tradeStatus() *UTAWebsocketTradeStatus
}

// tradeStatus returns the status a reply on the UTA trade connection carries
func (s *UTAWebsocketTradeStatus) tradeStatus() *UTAWebsocketTradeStatus {
	return s
}

// UTAWebsocketRateLimit is the account's rate limit state after a UTA trade connection request
type UTAWebsocketRateLimit struct {
	Limit     uint64 `json:"limit"`
	Remaining uint64 `json:"remaining"`
	// Reset is the time until the limit resets, in milliseconds
	Reset uint64 `json:"reset"`
}

// UTAWebsocketOrderResponse is the reply to an Add Order or Cancel Order operation
type UTAWebsocketOrderResponse struct {
	UTAWebsocketTradeStatus
	Data UTAOrderResponse `json:"data"`
}

// UTAWebsocketAmendOrderResponse is the reply to an Amend Order operation. It means KuCoin accepted the amendment: the
// order channel pushes its outcome
type UTAWebsocketAmendOrderResponse struct {
	UTAWebsocketTradeStatus
	Data AmendUTAOrderResponse `json:"data"`
}
