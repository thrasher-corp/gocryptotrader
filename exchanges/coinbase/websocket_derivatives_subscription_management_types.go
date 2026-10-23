package coinbase

// DerivativesSubscriptionWireRequest holds the params of a subscribe or unsubscribe request
type DerivativesSubscriptionWireRequest struct {
	Channels []string `json:"channels"`
}

// DerivativesPrivateSubscribeRequest holds the channels to subscribe a private connection to, and a label of up to 16
// characters the gateway adds to private channels' notifications
type DerivativesPrivateSubscribeRequest struct {
	Channels []string `json:"channels"`
	Label    string   `json:"label,omitempty"`
}
