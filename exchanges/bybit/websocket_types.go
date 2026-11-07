package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AuthenticationRequest authenticates a private stream connection (Connect: Authentication)
type AuthenticationRequest struct {
	RequestID string `json:"req_id,omitempty"`
	Operation string `json:"op"`
	// Arguments holds the API key, the expiry time in Unix milliseconds and the signature, in that order
	Arguments []any `json:"args"`
}

// SubscriptionRequest subscribes to or unsubscribes from topics on a public, private or system stream (Connect: How to
// Subscribe to Topics)
type SubscriptionRequest struct {
	RequestID string   `json:"req_id,omitempty"`
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

// SubscriptionResponse is the reply to a subscribe, unsubscribe, auth or ping request (Connect: Understanding the
// Subscription Response). Spot, linear, inverse and private streams echo the request ID and operation; option and system
// status streams reply with Type set to COMMAND_RESP and list the topics in Data instead, and answer pings with an
// operation of pong and the server time in Arguments
type SubscriptionResponse struct {
	Success       bool               `json:"success"`
	ReturnMessage string             `json:"ret_msg"`
	ConnectionID  string             `json:"conn_id"`
	RequestID     string             `json:"req_id"`
	Operation     string             `json:"op"`
	Type          string             `json:"type"`
	Data          SubscriptionTopics `json:"data"`
	Arguments     []types.Time       `json:"args"`
}

// SubscriptionTopics lists the topics an option or system status request applied and the ones it rejected
type SubscriptionTopics struct {
	FailTopics    []string `json:"failTopics"`
	SuccessTopics []string `json:"successTopics"`
}
