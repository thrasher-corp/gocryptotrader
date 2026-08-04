package htx

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5WsTradeRequest wraps an authenticated V5 trade operation.
type V5WsTradeRequest struct {
	Operation string `json:"op"`
	CID       string `json:"cid"`
	Data      any    `json:"data"`
}

// V5WsRateLimit stores the rate-limit state returned by V5 trade operations.
type V5WsRateLimit struct {
	Limit     types.Number `json:"limit"`
	Interval  types.Number `json:"interval"`
	Remaining types.Number `json:"remaining"`
	Reset     types.Time   `json:"reset"`
}

// V5WsOrderResponse stores a single V5 WebSocket trade acknowledgement.
type V5WsOrderResponse struct {
	V5Response
	CID       string              `json:"cid"`
	Data      V5OrderResponseData `json:"data"`
	RateLimit V5WsRateLimit       `json:"rate_limit"`
}

// V5WsBatchOrderResponse stores batch V5 WebSocket trade acknowledgements.
type V5WsBatchOrderResponse struct {
	V5Response
	CID       string                `json:"cid"`
	Data      []V5OrderResponseData `json:"data"`
	RateLimit V5WsRateLimit         `json:"rate_limit"`
}

// V5WsOrderUpdate contains an authenticated V5 USDT-margined order update.
type V5WsOrderUpdate struct {
	Operation    string        `json:"op"`
	Topic        string        `json:"topic"`
	ContractCode string        `json:"contract_code"`
	Timestamp    types.Time    `json:"ts"`
	UID          string        `json:"uid"`
	Data         V5WsOrderData `json:"data"`
}

// V5WsOrderData contains the order fields published by the V5 notification API.
type V5WsOrderData struct {
	Side                 string       `json:"side"`
	PositionSide         string       `json:"position_side"`
	Type                 string       `json:"type"`
	PriceMatch           string       `json:"price_match"`
	OrderID              string       `json:"order_id"`
	ClientOrderID        string       `json:"client_order_id"`
	MarginMode           string       `json:"margin_mode"`
	Price                types.Number `json:"price"`
	Volume               types.Number `json:"volume"`
	LeverageRate         uint64       `json:"lever_rate"`
	State                string       `json:"state"`
	OrderSource          string       `json:"order_source"`
	CancelReason         string       `json:"cancel_reason"`
	ReduceOnly           bool         `json:"reduce_only"`
	TimeInForce          string       `json:"time_in_force"`
	TradeAveragePrice    types.Number `json:"trade_avg_price"`
	TradeVolume          types.Number `json:"trade_volume"`
	CancelVolume         types.Number `json:"cancel_volume"`
	TradeTurnover        types.Number `json:"trade_turnover"`
	FeeCurrency          string       `json:"fee_currency"`
	Fee                  types.Number `json:"fee"`
	Profit               types.Number `json:"profit"`
	ContractType         string       `json:"contract_type"`
	TakeProfitPrice      types.Number `json:"tp_trigger_price"`
	TakeProfitOrderPrice types.Number `json:"tp_order_price"`
	TakeProfitType       string       `json:"tp_type"`
	TakeProfitPriceType  string       `json:"tp_trigger_price_type"`
	StopLossPrice        types.Number `json:"sl_trigger_price"`
	StopLossOrderPrice   types.Number `json:"sl_order_price"`
	StopLossType         string       `json:"sl_type"`
	StopLossPriceType    string       `json:"sl_trigger_price_type"`
	CreatedTime          types.Number `json:"created_time"`
	UpdatedTime          types.Number `json:"updated_time"`
	SelfMatchPrevent     string       `json:"self_match_prevent"`
	AmendOriginalVolume  types.Number `json:"amend_origin_volume"`
	AmendSource          string       `json:"amend_source"`
	AmendResult          string       `json:"amend_result"`
}

// V5WsTradeUpdate contains an authenticated V5 USDT-margined execution update.
type V5WsTradeUpdate struct {
	Operation    string          `json:"op"`
	Topic        string          `json:"topic"`
	ContractCode string          `json:"contract_code"`
	Timestamp    types.Time      `json:"ts"`
	UID          string          `json:"uid"`
	Data         json.RawMessage `json:"data"`
}

// V5WsTradeDetailUpdate contains an authenticated V5 USDT-margined execution-detail update.
type V5WsTradeDetailUpdate struct {
	Operation    string          `json:"op"`
	Topic        string          `json:"topic"`
	ContractCode string          `json:"contract_code"`
	Timestamp    types.Time      `json:"ts"`
	UID          string          `json:"uid"`
	Data         json.RawMessage `json:"data"`
}

// V5WsPositionUpdate contains an authenticated V5 USDT-margined position update.
type V5WsPositionUpdate struct {
	Operation    string          `json:"op"`
	Topic        string          `json:"topic"`
	ContractCode string          `json:"contract_code"`
	Timestamp    types.Time      `json:"ts"`
	UID          string          `json:"uid"`
	Data         json.RawMessage `json:"data"`
}

// V5WsAccountUpdate contains an authenticated V5 USDT-margined account update.
type V5WsAccountUpdate struct {
	Operation string          `json:"op"`
	Topic     string          `json:"topic"`
	Timestamp types.Time      `json:"ts"`
	UID       string          `json:"uid"`
	Data      json.RawMessage `json:"data"`
}

// V5WsMatchOrderUpdate contains an authenticated V5 USDT-margined match-order update.
type V5WsMatchOrderUpdate struct {
	Operation    string          `json:"op"`
	Topic        string          `json:"topic"`
	ContractCode string          `json:"contract_code"`
	Timestamp    types.Time      `json:"ts"`
	UID          string          `json:"uid"`
	Data         json.RawMessage `json:"data"`
}

// V5WsAlgoOrderUpdate contains an authenticated V5 USDT-margined strategy-order update.
type V5WsAlgoOrderUpdate struct {
	Operation    string          `json:"op"`
	Topic        string          `json:"topic"`
	ContractCode string          `json:"contract_code"`
	Timestamp    types.Time      `json:"ts"`
	UID          string          `json:"uid"`
	Data         json.RawMessage `json:"data"`
}
