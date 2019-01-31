package zb

import "encoding/json"

// Subscription defines an initial subscription type to be sent
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

// WsTicker defines websocket ticker data
type WsTicker struct {
	Date int64 `json:"date,string"`
	Data struct {
		Volume24Hr float64 `json:"vol,string"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		Last       float64 `json:"last,string"`
		Buy        float64 `json:"buy,string"`
		Sell       float64 `json:"sell,string"`
	} `json:"ticker"`
}

// WsDepth defines websocket orderbook data
type WsDepth struct {
	Timestamp int64         `json:"timestamp"`
	Asks      []interface{} `json:"asks"`
	Bids      []interface{} `json:"bids"`
}

// WsTrades defines websocket trade data
type WsTrades struct {
	Data []struct {
		Amount    float64 `json:"amount,string"`
		Price     float64 `json:"price,string"`
		TID       int64   `json:"tid"`
		Date      int64   `json:"date"`
		Type      string  `json:"type"`
		TradeType string  `json:"trade_type"`
	} `json:"data"`
}
