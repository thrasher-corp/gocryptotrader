package htx

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5Boolean accepts the empty-string false value returned by some V5 order endpoints.
type V5Boolean types.Boolean

// UnmarshalJSON decodes documented V5 boolean representations.
func (b *V5Boolean) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte(`""`)) {
		*b = false
		return nil
	}
	var value types.Boolean
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*b = V5Boolean(value)
	return nil
}

// Bool returns the underlying boolean value.
func (b V5Boolean) Bool() bool { return bool(b) }

// V5OrderState accepts both named and legacy numeric order states.
type V5OrderState string

// UnmarshalJSON decodes both V5 order-state representations.
func (s *V5OrderState) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*s = V5OrderState(value)
		return nil
	}
	var state int64
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	orderVars, err := compatibleVars("buy", "limit", state)
	if err != nil {
		return err
	}
	*s = V5OrderState(orderVars.Status.String())
	return nil
}

// LinearSwapMarket stores USDT-margined contract metadata.
type LinearSwapMarket struct {
	Symbol            string       `json:"symbol"`
	ContractCode      string       `json:"contract_code"`
	ContractSize      types.Number `json:"contract_size"`
	PriceTick         types.Number `json:"price_tick"`
	SettlementDate    string       `json:"settlement_date"`
	SettlementPeriod  string       `json:"settlement_period"`
	DeliveryTime      string       `json:"delivery_time"`
	CreateDate        types.Time   `json:"create_date"`
	ContractStatus    int64        `json:"contract_status"`
	SupportMarginMode string       `json:"support_margin_mode"`
	ContractType      string       `json:"contract_type"`
	Pair              string       `json:"pair"`
	BusinessType      string       `json:"business_type"`
	DeliveryDate      string       `json:"delivery_date"`
	TradePartition    string       `json:"trade_partition"`
}

// V5Response stores HTX V5 response status metadata.
type V5Response struct {
	Code      int64      `json:"code"`
	Message   string     `json:"message"`
	Timestamp types.Time `json:"ts"`
}

// V5AccountBalanceResponse stores USDT-margined unified-margin balances.
type V5AccountBalanceResponse struct {
	V5Response
	Data V5AccountBalance `json:"data"`
}

// V5AccountBalance stores a USDT-margined unified-margin account balance.
type V5AccountBalance struct {
	State                 string       `json:"state"`
	Equity                types.Number `json:"equity"`
	InitialMargin         types.Number `json:"initial_margin"`
	MaintenanceMargin     types.Number `json:"maintenance_margin"`
	MaintenanceMarginRate types.Number `json:"maintenance_margin_rate"`
	ProfitUnreal          types.Number `json:"profit_unreal"`
	AvailableMargin       types.Number `json:"available_margin"`
	VoucherValue          types.Number `json:"voucher_value"`
	CreatedTime           types.Time   `json:"created_time"`
	UpdatedTime           types.Time   `json:"updated_time"`
	Details               []struct {
		Currency              string       `json:"currency"`
		Equity                types.Number `json:"equity"`
		IsolatedEquity        types.Number `json:"isolated_equity"`
		Available             types.Number `json:"available"`
		IsolatedAvailable     types.Number `json:"isolated_available"`
		WithdrawAvailable     types.Number `json:"withdraw_available"`
		ProfitUnreal          types.Number `json:"profit_unreal"`
		IsolatedProfitUnreal  types.Number `json:"isolated_profit_unreal"`
		InitialMargin         types.Number `json:"initial_margin"`
		MaintenanceMargin     types.Number `json:"maintenance_margin"`
		MaintenanceMarginRate types.Number `json:"maintenance_margin_rate"`
		InitialMarginRate     types.Number `json:"initial_margin_rate"`
		Voucher               types.Number `json:"voucher"`
		VoucherValue          types.Number `json:"voucher_value"`
		AvailableMargin       types.Number `json:"available_margin"`
		CrossOrderFrozen      types.Number `json:"cross_order_frozen"`
		IsolatedOrderFrozen   types.Number `json:"isolated_order_frozen"`
		CreatedTime           types.Time   `json:"created_time"`
		UpdatedTime           types.Time   `json:"updated_time"`
	} `json:"details"`
}

// V5OpenInterestResponse stores the current USDT-margined contract open interest.
type V5OpenInterestResponse struct {
	V5Response
	Success bool               `json:"success"`
	Data    V5OpenInterestData `json:"data"`
}

// V5OpenInterestData stores current USDT-margined contract open interest data.
type V5OpenInterestData struct {
	ContractCode  string       `json:"contract_code"`
	Amount        types.Number `json:"amount"`
	Volume        types.Number `json:"volume"`
	Value         types.Number `json:"value"`
	TradeAmount   types.Number `json:"trade_amount"`
	TradeVolume   types.Number `json:"trade_volume"`
	TradeTurnover types.Number `json:"trade_turnover"`
}

// V5OrderRequest stores a USDT-margined V5 order request.
type V5OrderRequest struct {
	ContractCode        string        `json:"contract_code"`
	MarginMode          string        `json:"margin_mode"`
	PositionSide        string        `json:"position_side,omitempty"`
	Side                string        `json:"side"`
	Type                string        `json:"type"`
	PriceMatch          string        `json:"price_match,omitempty"`
	ClientOrderID       string        `json:"client_order_id,omitempty"`
	Price               types.Number  `json:"price,omitempty"`
	Volume              types.Number  `json:"volume"`
	ReduceOnly          int64         `json:"reduce_only,omitempty"`
	TimeInForce         string        `json:"time_in_force,omitempty"`
	TakeProfitTrigger   types.Number  `json:"tp_trigger_price,omitempty"`
	TakeProfitPrice     types.Number  `json:"tp_order_price,omitempty"`
	TakeProfitType      string        `json:"tp_type,omitempty"`
	TakeProfitPriceType string        `json:"tp_trigger_price_type,omitempty"`
	StopLossTrigger     types.Number  `json:"sl_trigger_price,omitempty"`
	StopLossPrice       types.Number  `json:"sl_order_price,omitempty"`
	StopLossType        string        `json:"sl_type,omitempty"`
	StopLossPriceType   string        `json:"sl_trigger_price_type,omitempty"`
	PriceProtect        types.Boolean `json:"price_protect,omitempty"`
	SelfMatchPrevent    string        `json:"self_match_prevent,omitempty"`
}

// V5CancelOrderRequest stores a USDT-margined V5 cancel order request.
type V5CancelOrderRequest struct {
	ContractCode  string `json:"contract_code"`
	OrderID       string `json:"order_id,omitempty"`
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// V5CancelAllOrdersRequest stores a USDT-margined V5 cancel all orders request.
type V5CancelAllOrdersRequest struct {
	ContractCode string `json:"contract_code,omitempty"`
	Side         string `json:"side,omitempty"`
	PositionSide string `json:"position_side,omitempty"`
}

// V5OrderResponse stores a USDT-margined V5 order response.
type V5OrderResponse struct {
	V5Response
	Data V5OrderResponseData `json:"data"`
}

// V5OrderResponseData stores a USDT-margined V5 order acknowledgement.
type V5OrderResponseData struct {
	Code          int64  `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_order_id"`
}

// UnmarshalJSON preserves order identifiers regardless of whether HTX emits them as strings or bare numbers.
func (v *V5OrderResponseData) UnmarshalJSON(data []byte) error {
	var raw struct {
		Code          int64           `json:"code"`
		Message       string          `json:"message"`
		OrderID       json.RawMessage `json:"order_id"`
		ClientOrderID json.RawMessage `json:"client_order_id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decodeID := func(value json.RawMessage) (string, error) {
		if len(value) == 0 || bytes.Equal(value, []byte("null")) {
			return "", nil
		}
		if value[0] != '"' {
			return string(value), nil
		}
		var id string
		if err := json.Unmarshal(value, &id); err != nil {
			return "", fmt.Errorf("error decoding V5 order identifier: %w", err)
		}
		return id, nil
	}
	orderID, err := decodeID(raw.OrderID)
	if err != nil {
		return err
	}
	clientOrderID, err := decodeID(raw.ClientOrderID)
	if err != nil {
		return err
	}
	v.Code = raw.Code
	v.Message = raw.Message
	v.OrderID = orderID
	v.ClientOrderID = clientOrderID
	return nil
}

// V5CancelAllOrdersResponse stores USDT-margined V5 cancel all order acknowledgements.
type V5CancelAllOrdersResponse struct {
	V5Response
	Data []V5OrderResponseData `json:"data"`
}

// V5OrderQueryResponse stores a USDT-margined V5 order query response.
type V5OrderQueryResponse struct {
	V5Response
	Data V5OrderData `json:"data"`
}

// V5OrdersQueryResponse stores a USDT-margined V5 order list response.
type V5OrdersQueryResponse struct {
	V5Response
	Data []V5OrderData `json:"data"`
}

// V5OrderData stores USDT-margined V5 order details.
type V5OrderData struct {
	ID                  string       `json:"id"`
	ContractCode        string       `json:"contract_code"`
	Side                string       `json:"side"`
	PositionSide        string       `json:"position_side"`
	Type                string       `json:"type"`
	PriceMatch          string       `json:"price_match"`
	OrderID             string       `json:"order_id"`
	ClientOrderID       string       `json:"client_order_id"`
	MarginMode          string       `json:"margin_mode"`
	Price               types.Number `json:"price"`
	Volume              types.Number `json:"volume"`
	LeverageRate        types.Number `json:"lever_rate"`
	State               V5OrderState `json:"state"`
	OrderSource         string       `json:"order_source"`
	ReduceOnly          V5Boolean    `json:"reduce_only"`
	TimeInForce         string       `json:"time_in_force"`
	TakeProfitTrigger   types.Number `json:"tp_trigger_price"`
	TakeProfitPrice     types.Number `json:"tp_order_price"`
	TakeProfitType      string       `json:"tp_type"`
	TakeProfitPriceType string       `json:"tp_trigger_price_type"`
	StopLossTrigger     types.Number `json:"sl_trigger_price"`
	StopLossPrice       types.Number `json:"sl_order_price"`
	StopLossType        string       `json:"sl_type"`
	StopLossPriceType   string       `json:"sl_trigger_price_type"`
	TradeAveragePrice   types.Number `json:"trade_avg_price"`
	TradeVolume         types.Number `json:"trade_volume"`
	TradeTurnover       types.Number `json:"trade_turnover"`
	FeeCurrency         string       `json:"fee_currency"`
	Fee                 types.Number `json:"fee"`
	PriceProtect        V5Boolean    `json:"price_protect"`
	Profit              types.Number `json:"profit"`
	ContractType        string       `json:"contract_type"`
	CreatedTime         types.Time   `json:"created_time"`
	UpdatedTime         types.Time   `json:"updated_time"`
	CancelReason        string       `json:"cancel_reason"`
	SelfMatchPrevent    string       `json:"self_match_prevent"`
}
