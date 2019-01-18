package zb

import "encoding/json"

// Subscription defines an intial subscription type to be sent
type Subscription struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

// Generic defines a generic fields associated with many return types
type Generic struct {
	Code    int64           `json:"code"`
	Success bool            `json:"success"`
	Channel string          `json:"channel"`
	Message string          `json:"message"`
	No      int64           `json:"no"`
	Data    json.RawMessage `json:"data"`
}

// Markets defines market data
type Markets map[string]struct {
	AmountScale int64 `json:"amountScale"`
	PriceScale  int64 `json:"priceScale"`
}
