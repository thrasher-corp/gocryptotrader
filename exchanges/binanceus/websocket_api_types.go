package binanceus

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// WebsocketAPIRequest is the envelope of a WebSocket API request
type WebsocketAPIRequest struct {
	ID     string            `json:"id"`
	Method string            `json:"method"`
	Params map[string]string `json:"params,omitempty"`
}

// WebsocketAPIResponse is the envelope of a WebSocket API response; Result is present when the request succeeded and
// Error when it failed
type WebsocketAPIResponse struct {
	ID         string                  `json:"id"`
	Status     uint64                  `json:"status"`
	Result     json.RawMessage         `json:"result"`
	Error      *WebsocketAPIError      `json:"error"`
	RateLimits []RateLimitItemResponse `json:"rateLimits"`
}

// WebsocketAPIError is a failed WebSocket API request's error
type WebsocketAPIError struct {
	Code    int64  `json:"code"` // Binance.US's error codes are negative
	Message string `json:"msg"`
	// Data holds method specific details, such as retryAfter when rate limited or the outcome of each half of a
	// cancel-replace
	Data json.RawMessage `json:"data"`
}
