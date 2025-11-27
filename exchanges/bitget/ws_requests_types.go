package bitget

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// WebsocketTradeRequest represents a general trade request
type WebsocketTradeRequest struct {
	ID             string `json:"id"`
	InstrumentType string `json:"instType"`
	InstrumentID   string `json:"instId"`
	Channel        string `json:"channel"`
	Params         any    `json:"params"`
}

// WebsocketTradeResponse represents a general trade response
type WebsocketTradeResponse struct {
	Event   string `json:"event"`
	Code    int64  `json:"code"`
	Result  any    `json:"arg"`
	Message string `json:"msg"`
}

// WebsocketSpotPlaceOrderRequest defines the parameters for placing a spot order via websocket
type WebsocketSpotPlaceOrderRequest struct {
	Pair                currency.Pair
	OrderType           string  // limit or market
	Side                string  // buy or sell
	Size                float64 // market orders quote size long, base size short. limit orders base size.
	TimeInForce         string  // gtc, post_only, fok or ioc
	Price               float64
	ClientOrderID       string
	SelfTradePrevention string // cancel_taker, cancel_maker, cancel_both
}

// WebsocketFuturesOrderRequest defines the parameters for placing a futures order via websocket
type WebsocketFuturesOrderRequest struct {
	Contract               currency.Pair
	InstrumentType         string
	OrderType              string // limit or market
	Side                   string // buy or sell
	ContractSize           float64
	TimeInForce            string // gtc, post_only, fok or ioc
	Price                  float64
	ClientOrderID          string
	MarginCoin             currency.Code
	MarginMode             string  // isolated or crossed
	TradeSide              string  // open or close: only required in hedge-mode
	ReduceOnly             string  // "YES" or "NO": only required in one-way-position mode
	PresetStopSurplusPrice float64 // take-profit value
	PresetStopLossPrice    float64 // stop-loss value
	SelfTradePrevention    string  // cancel_taker, cancel_maker, cancel_both
}

// WebsocketSpotOrderParams defines the order parameters for a websocket order request
type WebsocketSpotOrderParams struct {
	OrderType           string `json:"orderType"`
	Side                string `json:"side"`
	Size                string `json:"size"`
	Force               string `json:"force"`
	Price               string `json:"price,omitempty"`
	ClientOrderID       string `json:"clientOid,omitempty"`
	SelfTradePrevention string `json:"stp,omitempty"`
}

// WebsocketFuturesOrderParams defines the order parameters for a websocket futures order request
type WebsocketFuturesOrderParams struct {
	OrderType              string `json:"orderType"`
	Side                   string `json:"side"`
	Size                   string `json:"size"`
	Force                  string `json:"force"`
	Price                  string `json:"price,omitempty"`
	ClientOrderID          string `json:"clientOid,omitempty"`
	MarginCoin             string `json:"marginCoin"`
	MarginMode             string `json:"marginMode"`
	TradeSide              string `json:"tradeSide,omitempty"`
	ReduceOnly             string `json:"reduceOnly,omitempty"`
	PresetStopSurplusPrice string `json:"presetStopSurplusPrice,omitempty"`
	PresetStopLossPrice    string `json:"presetStopLossPrice,omitempty"`
	SelfTradePrevention    string `json:"stp,omitempty"`
}

// WebsocketSpotPlaceOrderResponse defines the response parameters for a placed order via websocket
type WebsocketSpotPlaceOrderResponse struct {
	ID             string                                `json:"id"`
	InstrumentType string                                `json:"instType"`
	Channel        string                                `json:"channel"`
	InstrumentID   string                                `json:"instId"`
	Params         WebsocketSpotPlaceOrderParamsResponse `json:"params"`
}

// WebsocketFuturesPlaceOrderResponse defines the response parameters for a placed order via websocket
type WebsocketFuturesPlaceOrderResponse struct {
	ID             string                                   `json:"id"`
	InstrumentType string                                   `json:"instType"`
	Channel        string                                   `json:"channel"`
	InstrumentID   string                                   `json:"instId"`
	Params         WebsocketFuturesPlaceOrderParamsResponse `json:"params"`
}

// WebsocketFuturesPlaceOrderParamsResponse defines the response parameters for a placed order via websocket
type WebsocketFuturesPlaceOrderParamsResponse struct {
	// Returned on success
	OrderID string `json:"orderId"`
	// Returned on error includes client order id
	WebsocketFuturesOrderParams
}

// WebsocketSpotPlaceOrderParamsResponse defines the response parameters for a placed order via websocket
type WebsocketSpotPlaceOrderParamsResponse struct {
	// Returned on success
	OrderID string `json:"orderId"`
	// Returned on error includes client order id
	WebsocketSpotOrderParams
}

// WebsocketCancelOrderResponse defines the response parameters for a cancelled order via websocket
type WebsocketCancelOrderResponse struct {
	ID             string       `json:"id"`
	InstrumentType string       `json:"instType"`
	Channel        string       `json:"channel"`
	InstrumentID   string       `json:"instId"`
	Params         WebsocketIDs `json:"params"`
}

// WebsocketIDs defines order identification parameters
type WebsocketIDs struct {
	OrderID       string `json:"orderId,omitempty"`
	ClientOrderID string `json:"clientOid,omitempty"`
}
