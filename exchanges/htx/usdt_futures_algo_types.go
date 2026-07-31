package htx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5AlgoOrderRequest defines a strategy order.
type V5AlgoOrderRequest struct {
	ContractCode        string        `json:"contract_code"`
	Type                string        `json:"type"`
	PositionSide        string        `json:"position_side"`
	Side                string        `json:"side"`
	ClientOrderID       string        `json:"algo_client_order_id,omitempty"`
	MarginMode          string        `json:"margin_mode"`
	Volume              types.Number  `json:"volume,omitempty"`
	TakeProfitTrigger   types.Number  `json:"tp_trigger_price,omitempty"`
	TakeProfitPrice     types.Number  `json:"tp_order_price,omitempty"`
	TakeProfitType      string        `json:"tp_type,omitempty"`
	TakeProfitPriceType string        `json:"tp_trigger_price_type,omitempty"`
	StopLossTrigger     types.Number  `json:"sl_trigger_price,omitempty"`
	StopLossPrice       types.Number  `json:"sl_order_price,omitempty"`
	StopLossType        string        `json:"sl_type,omitempty"`
	StopLossPriceType   string        `json:"sl_trigger_price_type,omitempty"`
	Price               types.Number  `json:"price,omitempty"`
	PriceMatch          string        `json:"price_match,omitempty"`
	TriggerPrice        types.Number  `json:"trigger_price,omitempty"`
	TriggerPriceType    string        `json:"trigger_price_type,omitempty"`
	ActivationPrice     types.Number  `json:"active_price,omitempty"`
	OrderPriceType      string        `json:"order_price_type,omitempty"`
	CallbackRate        types.Number  `json:"callback_rate,omitempty"`
	ReduceOnly          types.Boolean `json:"reduce_only,omitempty"`
}

// V5CancelAlgoOrderRequest defines a strategy-order cancellation.
type V5CancelAlgoOrderRequest struct {
	AlgoID        string `json:"algo_id,omitempty"`
	ClientOrderID string `json:"algo_client_order_id,omitempty"`
	ContractCode  string `json:"contract_code"`
}

// V5AlgoAcknowledgementsResponse stores strategy-order acknowledgements.
type V5AlgoAcknowledgementsResponse struct {
	V5Response
	Data []V5AlgoAcknowledgement `json:"data"`
}

// V5AlgoAcknowledgement stores one strategy-order acknowledgement.
type V5AlgoAcknowledgement struct {
	Code          int64  `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	AlgoID        string `json:"algo_id"`
	ClientOrderID string `json:"algo_client_order_id"`
}

// V5OpenAlgoOrdersRequest defines open strategy-order filters.
type V5OpenAlgoOrdersRequest struct {
	ContractCode  string
	AlgoID        string
	ClientOrderID string
	Type          string
	From          uint64
	Limit         uint64
	Direction     string
}

// V5AlgoOrderHistoryRequest defines historical strategy-order filters.
type V5AlgoOrderHistoryRequest struct {
	ContractCode string
	MarginMode   string
	States       string
	Type         string
	StartTime    time.Time
	EndTime      time.Time
	From         uint64
	Limit        uint64
	Direction    string
}

// V5AlgoOrdersResponse stores strategy orders.
type V5AlgoOrdersResponse struct {
	V5Response
	Data []V5AlgoOrder `json:"data"`
}

// V5AlgoOrder stores a strategy order.
type V5AlgoOrder struct {
	ID                  types.Number  `json:"id"`
	AlgoID              string        `json:"algo_id"`
	ClientOrderID       string        `json:"algo_client_order_id"`
	ContractCode        string        `json:"contract_code"`
	Volume              types.Number  `json:"volume"`
	Type                string        `json:"type"`
	State               string        `json:"state"`
	PositionSide        string        `json:"position_side"`
	MarginMode          string        `json:"margin_mode"`
	Side                string        `json:"side"`
	TakeProfitTrigger   types.Number  `json:"tp_trigger_price"`
	TakeProfitPrice     types.Number  `json:"tp_order_price"`
	TakeProfitType      string        `json:"tp_type"`
	TakeProfitPriceType string        `json:"tp_trigger_price_type"`
	StopLossTrigger     types.Number  `json:"sl_trigger_price"`
	StopLossPrice       types.Number  `json:"sl_order_price"`
	StopLossType        string        `json:"sl_type"`
	StopLossPriceType   string        `json:"sl_trigger_price_type"`
	Price               types.Number  `json:"price"`
	PriceMatch          string        `json:"price_match"`
	TriggerPrice        types.Number  `json:"trigger_price"`
	TriggerPriceType    string        `json:"trigger_price_type"`
	ActivationPrice     types.Number  `json:"active_price"`
	OrderPriceType      string        `json:"order_price_type"`
	CallbackRate        types.Number  `json:"callback_rate"`
	ReduceOnly          types.Boolean `json:"reduce_only"`
	ActualVolume        types.Number  `json:"actual_volume"`
	ActualPrice         types.Number  `json:"actual_price"`
	ActualTime          types.Time    `json:"actual_time"`
	RelatedOrderID      string        `json:"relation_order_id"`
	CreatedTime         types.Time    `json:"created_time"`
	UpdatedTime         types.Time    `json:"updated_time"`
	OrderSource         string        `json:"order_source"`
}
