package deribit

// SubscriptionRequest contains the channels of a subscribe or unsubscribe request
type SubscriptionRequest struct {
	Channels []string `json:"channels"`
}

// PrivateSubscribeRequest subscribes an authenticated connection to channels
type PrivateSubscribeRequest struct {
	Channels []string `json:"channels"`
	// Label is added to private channel notifications, and may be up to 16 characters
	Label string `json:"label,omitempty"`
}
