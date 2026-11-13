package mexc

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketFuturesRequest is a request sent over the futures websocket
type WebsocketFuturesRequest struct {
	Method string `json:"method"`
	Param  any    `json:"param,omitempty"`
}

// WebsocketFuturesChannelParams holds the parameters of a futures channel subscription
type WebsocketFuturesChannelParams struct {
	Symbol   string `json:"symbol,omitempty"`
	Interval string `json:"interval,omitempty"`
	Limit    uint64 `json:"limit,omitempty"`
}

// WebsocketFuturesLoginParams holds the parameters of a futures websocket login
type WebsocketFuturesLoginParams struct {
	APIKey      string `json:"apiKey"`
	RequestTime string `json:"reqTime"`
	Signature   string `json:"signature"`
}

// WebsocketFuturesResponse is a frame from the futures websocket: a push, a request's reply or a pong
type WebsocketFuturesResponse struct {
	Channel   string          `json:"channel"`
	Symbol    string          `json:"symbol"`
	Data      json.RawMessage `json:"data"`
	Timestamp types.Time      `json:"ts"`
}

// WebsocketFuturesTicker is a contract's ticker pushed over the futures websocket. Volume24Hour and HoldVolume are in
// contracts
type WebsocketFuturesTicker struct {
	Symbol          string     `json:"symbol"`
	LastPrice       float64    `json:"lastPrice"`
	BestBidPrice    float64    `json:"bid1"`
	BestAskPrice    float64    `json:"ask1"`
	Volume24Hour    float64    `json:"volume24"`
	Amount24Hour    float64    `json:"amount24"`
	HoldVolume      float64    `json:"holdVol"`
	Low24Hour       float64    `json:"lower24Price"`
	High24Hour      float64    `json:"high24Price"`
	PriceChangeRate float64    `json:"riseFallRate"`
	PriceChange     float64    `json:"riseFallValue"`
	IndexPrice      float64    `json:"indexPrice"`
	FairPrice       float64    `json:"fairPrice"`
	FundingRate     float64    `json:"fundingRate"`
	MaxBidPrice     float64    `json:"maxBidPrice"`
	MinAskPrice     float64    `json:"minAskPrice"`
	Timestamp       types.Time `json:"timestamp"`
}

// WebsocketFuturesCandle is a contract's forming candle pushed over the futures websocket. Volume is in contracts
type WebsocketFuturesCandle struct {
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	Amount    float64    `json:"a"`
	Volume    float64    `json:"q"`
	Open      float64    `json:"o"`
	Close     float64    `json:"c"`
	High      float64    `json:"h"`
	Low       float64    `json:"l"`
	RealOpen  float64    `json:"ro"`
	RealClose float64    `json:"rc"`
	RealHigh  float64    `json:"rh"`
	RealLow   float64    `json:"rl"`
	StartTime types.Time `json:"t"`
}

// WebsocketFuturesAsset is a futures account balance pushed over the futures websocket
type WebsocketFuturesAsset struct {
	Currency         currency.Code `json:"currency"`
	PositionMargin   float64       `json:"positionMargin"`
	FrozenBalance    float64       `json:"frozenBalance"`
	AvailableBalance float64       `json:"availableBalance"`
	Bonus            float64       `json:"bonus"`
}
