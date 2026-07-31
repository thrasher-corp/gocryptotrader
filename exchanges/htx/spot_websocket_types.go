package htx

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsSubReq is a request to subscribe to or unubscribe from a topic for public channels (private channels use generic wsReq)
type wsSubReq struct {
	ID    string `json:"id,omitempty"`
	Sub   string `json:"sub,omitempty"`
	Unsub string `json:"unsub,omitempty"`
}

// WsHeartBeat defines a heartbeat request
type WsHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WsDepth defines market depth websocket response
type WsDepth struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Bids      [][]any    `json:"bids"`
		Asks      [][]any    `json:"asks"`
		Timestamp types.Time `json:"ts"`
		Version   int64      `json:"version"`
	} `json:"tick"`
}

// WsKline defines market kline websocket response
type WsKline struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  int64   `json:"count"`
	} `json:"tick"`
}

// WsTick stores websocket ticker data
type WsTick struct {
	Channel   string     `json:"ch"`
	Rep       string     `json:"rep"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Amount    float64    `json:"amount"`
		Close     float64    `json:"close"`
		Count     float64    `json:"count"`
		High      float64    `json:"high"`
		ID        float64    `json:"id"`
		Low       float64    `json:"low"`
		Open      float64    `json:"open"`
		Timestamp types.Time `json:"ts"`
		Volume    float64    `json:"vol"`
	} `json:"tick"`
}

// WsTrade defines market trade websocket response
type WsTrade struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID        int64      `json:"id"`
		Timestamp types.Time `json:"ts"`
		Data      []struct {
			Amount    float64    `json:"amount"`
			Timestamp types.Time `json:"ts"`
			TradeID   float64    `json:"tradeId"`
			Price     float64    `json:"price"`
			Direction string     `json:"direction"`
		} `json:"data"`
	}
}

// wsReq contains authentication login fields
type wsReq struct {
	Action  string `json:"action"`
	Channel string `json:"ch"`
	Params  any    `json:"params"`
}

// wsAuthReq contains authentication login fields
type wsAuthReq struct {
	AuthType         string `json:"authType"`
	AccessKey        string `json:"accessKey"`
	SignatureMethod  string `json:"signatureMethod"`
	SignatureVersion string `json:"signatureVersion"`
	Timestamp        string `json:"timestamp"`
	Signature        string `json:"signature"`
}

type wsAccountUpdateMsg struct {
	Data WsAccountUpdate `json:"data"`
}

// WsAccountUpdate contains account updates to balances
type WsAccountUpdate struct {
	Currency    string       `json:"currency"`
	AccountID   int64        `json:"accountId"`
	Balance     types.Number `json:"balance"`
	Available   types.Number `json:"available"`
	ChangeType  string       `json:"changeType"`
	AccountType string       `json:"accountType"`
	ChangeTime  types.Time   `json:"changeTime"`
	SeqNum      int64        `json:"seqNum"`
}

type wsOrderUpdateMsg struct {
	Data WsOrderUpdate `json:"data"`
}

// WsOrderUpdate contains updates to orders
type WsOrderUpdate struct {
	EventType       string       `json:"eventType"`
	Symbol          string       `json:"symbol"`
	AccountID       int64        `json:"accountId"`
	OrderID         int64        `json:"orderId"`
	TradeID         int64        `json:"tradeId"`
	ClientOrderID   string       `json:"clientOrderId"`
	Source          string       `json:"orderSource"`
	Price           types.Number `json:"orderPrice"`
	Size            types.Number `json:"orderSize"`
	Value           types.Number `json:"orderValue"`
	OrderType       string       `json:"type"`
	TradePrice      types.Number `json:"tradePrice"`
	TradeVolume     types.Number `json:"tradeVolume"`
	RemainingAmount types.Number `json:"remainAmt"`
	ExecutedAmount  types.Number `json:"execAmt"`
	IsTaker         bool         `json:"aggressor"`
	Side            order.Side   `json:"orderSide"`
	OrderStatus     string       `json:"orderStatus"`
	LastActTime     types.Time   `json:"lastActTime"`
	CreateTime      types.Time   `json:"orderCreateTime"`
	TradeTime       types.Time   `json:"tradeTime"`
	ErrCode         int64        `json:"errCode"`
	ErrMessage      string       `json:"errMessage"`
}

type wsTradeUpdateMsg struct {
	Data WsTradeUpdate `json:"data"`
}

// WsTradeUpdate contains trade updates to orders
type WsTradeUpdate struct {
	EventType       string       `json:"eventType"`
	Symbol          string       `json:"symbol"`
	OrderID         int64        `json:"orderId"`
	TradePrice      types.Number `json:"tradePrice"`
	TradeVolume     types.Number `json:"tradeVolume"`
	Side            order.Side   `json:"orderSide"`
	OrderType       string       `json:"orderType"`
	IsTaker         bool         `json:"aggressor"`
	TradeID         int64        `json:"tradeId"`
	TradeTime       types.Time   `json:"tradeTime"`
	TransactFee     types.Number `json:"transactFee"`
	FeeCurrency     string       `json:"feeCurrency"`
	FeeDeduct       string       `json:"feeDeduct"`
	FeeDeductType   string       `json:"feeDeductType"`
	AccountID       int64        `json:"accountId"`
	Source          string       `json:"orderSource"`
	OrderPrice      types.Number `json:"orderPrice"`
	OrderSize       types.Number `json:"orderSize"`
	Value           types.Number `json:"orderValue"`
	ClientOrderID   string       `json:"clientOrderId"`
	StopPrice       string       `json:"stopPrice"`
	Operator        string       `json:"operator"`
	OrderCreateTime types.Time   `json:"orderCreateTime"`
	OrderStatus     string       `json:"orderStatus"`
}
