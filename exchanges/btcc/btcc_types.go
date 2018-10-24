package btcc

import "encoding/json"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// WsAllTickerData defines multiple ticker data
type WsAllTickerData []WsTicker

// WsOutgoing defines outgoing JSON
type WsOutgoing struct {
	Action string `json:"action"`
	Symbol string `json:"symbol,omitempty"`
	Count  int    `json:"count,omitempty"`
	Len    int    `json:"len,omitempty"`
}

// WsResponseMain defines the main websocket response
type WsResponseMain struct {
	MsgType string          `json:"MsgType"`
	CRID    string          `json:"CRID"`
	RC      interface{}     `json:"RC"`
	Reason  string          `json:"Reason"`
	Data    json.RawMessage `json:"data"`
}

// WsOrderbookSnapshot defines an orderbook from the websocket
type WsOrderbookSnapshot struct {
	Timestamp int64  `json:"Timestamp"`
	Symbol    string `json:"Symbol"`
	Version   int64  `json:"Version"`
	Type      string `json:"Type"`
	Content   string `json:"Content"`
	List      []struct {
		Side  string      `json:"Side"`
		Size  interface{} `json:"Size"`
		Price float64     `json:"Price"`
	} `json:"List"`
	MsgType string `json:"MsgType"`
}

// WsOrderbookSnapshotOld defines an old orderbook from the websocket connection
type WsOrderbookSnapshotOld struct {
	MsgType   string                   `json:"MsgType"`
	Symbol    string                   `json:"Symbol"`
	Data      map[string][]interface{} `json:"Data"`
	Timestamp int64                    `json:"Timestamp"`
}

// WsTrades defines trading data from the websocket
type WsTrades struct {
	Trades []struct {
		TID       int64   `json:"TID"`
		Timestamp int64   `json:"Timestamp"`
		Symbol    string  `json:"Symbol"`
		Side      string  `json:"Side"`
		Size      float64 `json:"Size"`
		Price     float64 `json:"Price"`
		MsgType   string  `json:"MsgType"`
	} `json:"Trades"`
	RC      int64  `json:"RC"`
	CRID    string `json:"CRID"`
	Reason  string `json:"Reason"`
	MsgType string `json:"MsgType"`
}

// WsTicker defines ticker data from the websocket
type WsTicker struct {
	Symbol             string  `json:"Symbol"`
	BidPrice           float64 `json:"BidPrice"`
	AskPrice           float64 `json:"AskPrice"`
	Open               float64 `json:"Open"`
	High               float64 `json:"High"`
	Low                float64 `json:"Low"`
	Last               float64 `json:"Last"`
	LastQuantity       float64 `json:"LastQuantity"`
	PrevCls            float64 `json:"PrevCls"`
	Volume             float64 `json:"Volume"`
	Volume24H          float64 `json:"Volume24H"`
	Timestamp          int64   `json:"Timestamp"`
	ExecutionLimitDown float64 `json:"ExecutionLimitDown"`
	ExecutionLimitUp   float64 `json:"ExecutionLimitUp"`
	MsgType            string  `json:"MsgType"`
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.USD:  0.005,
	symbol.USDT: 10,
	symbol.BTC:  0.001,
	symbol.ETH:  0.01,
	symbol.BCH:  0.0001,
	symbol.LTC:  0.001,
	symbol.DASH: 0.002,
}
