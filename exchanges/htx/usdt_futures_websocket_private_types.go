package htx

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5WebsocketTradeRequest wraps an authenticated V5 trade operation.
type V5WebsocketTradeRequest struct {
	Operation  string `json:"op"`
	CID        string `json:"cid"`
	Parameters any    `json:"data"`
}

// V5WebsocketRateLimit stores the rate-limit state returned by V5 trade operations.
type V5WebsocketRateLimit struct {
	Limit     types.Number `json:"limit"`
	Interval  types.Number `json:"interval"`
	Remaining types.Number `json:"remaining"`
	Reset     types.Time   `json:"reset"`
}

// UnmarshalJSON accepts both the object schema and the JSON-encoded string used
// in HTX's trade acknowledgement examples.
func (r *V5WebsocketRateLimit) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return fmt.Errorf("decoding rate-limit string: %w", err)
		}
		data = []byte(encoded)
	}
	type rateLimit V5WebsocketRateLimit
	var decoded rateLimit
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("decoding rate-limit state: %w", err)
	}
	*r = V5WebsocketRateLimit(decoded)
	return nil
}

// V5WebsocketOrderResponse stores a single V5 WebSocket trade acknowledgement.
type V5WebsocketOrderResponse struct {
	V5Response
	CID             string                 `json:"cid"`
	Acknowledgement V5OrderAcknowledgement `json:"data"`
	RateLimit       V5WebsocketRateLimit   `json:"rate_limit"`
}

// V5WebsocketBatchOrderResponse stores batch V5 WebSocket trade acknowledgements.
type V5WebsocketBatchOrderResponse struct {
	V5Response
	CID              string                   `json:"cid"`
	Acknowledgements []V5OrderAcknowledgement `json:"data"`
	RateLimit        V5WebsocketRateLimit     `json:"rate_limit"`
}

// V5WebsocketOrderUpdate contains an authenticated V5 USDT-margined order update.
type V5WebsocketOrderUpdate struct {
	Operation    string           `json:"op"`
	Topic        string           `json:"topic"`
	ContractCode string           `json:"contract_code"`
	Timestamp    types.Time       `json:"ts"`
	UID          string           `json:"uid"`
	Order        V5WebsocketOrder `json:"data"`
}

// V5WebsocketOrder contains the order fields published by the V5 notification API.
type V5WebsocketOrder struct {
	Side                 string       `json:"side"`
	PositionSide         string       `json:"position_side"`
	Type                 string       `json:"type"`
	PriceMatch           string       `json:"price_match"`
	OrderID              string       `json:"order_id"`
	ClientOrderID        string       `json:"client_order_id"`
	MarginMode           string       `json:"margin_mode"`
	Price                types.Number `json:"price"`
	Volume               types.Number `json:"volume"`
	LeverageRate         types.Number `json:"lever_rate"`
	State                V5OrderState `json:"state"`
	OrderSource          string       `json:"order_source"`
	CancelReason         string       `json:"cancel_reason"`
	ReduceOnly           V5Boolean    `json:"reduce_only"`
	TimeInForce          string       `json:"time_in_force"`
	TradeAveragePrice    types.Number `json:"trade_avg_price"`
	TradeVolume          types.Number `json:"trade_volume"`
	CancelVolume         types.Number `json:"cancel_volume"`
	TradeTurnover        types.Number `json:"trade_turnover"`
	FeeCurrency          string       `json:"fee_currency"` // HTX may send a comma-separated list of fee currencies.
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
	CreatedTime          types.Time   `json:"created_time"`
	UpdatedTime          types.Time   `json:"updated_time"`
	SelfMatchPrevent     string       `json:"self_match_prevent"`
	AmendOriginalVolume  types.Number `json:"amend_origin_volume"`
	AmendSource          string       `json:"amend_source"`
	AmendResult          string       `json:"amend_result"`
}

// V5WebsocketTradeUpdate contains an authenticated V5 USDT-margined execution update.
type V5WebsocketTradeUpdate struct {
	Operation    string                 `json:"op"`
	Topic        string                 `json:"topic"`
	ContractCode string                 `json:"contract_code"`
	Timestamp    types.Time             `json:"ts"`
	UID          string                 `json:"uid"`
	Executions   []V5WebsocketExecution `json:"data"`
}

// V5WebsocketTradeDetailUpdate contains an authenticated V5 USDT-margined execution-detail update.
type V5WebsocketTradeDetailUpdate struct {
	Operation    string                 `json:"op"`
	Topic        string                 `json:"topic"`
	ContractCode string                 `json:"contract_code"`
	Timestamp    types.Time             `json:"ts"`
	UID          string                 `json:"uid"`
	Executions   []V5WebsocketExecution `json:"data"`
}

// V5WebsocketPositionUpdate contains an authenticated V5 USDT-margined position update.
type V5WebsocketPositionUpdate struct {
	Event        string                `json:"event"`
	Operation    string                `json:"op"`
	Topic        string                `json:"topic"`
	ContractCode string                `json:"contract_code"`
	Timestamp    types.Time            `json:"ts"`
	UID          string                `json:"uid"`
	Positions    []V5WebsocketPosition `json:"data"`
}

// V5WebsocketAccountUpdate contains an authenticated V5 USDT-margined account update.
type V5WebsocketAccountUpdate struct {
	Operation string           `json:"op"`
	Topic     string           `json:"topic"`
	Timestamp types.Time       `json:"ts"`
	UID       string           `json:"uid"`
	Balance   V5AccountBalance `json:"data"`
}

// V5WebsocketMatchOrderUpdate contains an authenticated V5 USDT-margined match-order update.
type V5WebsocketMatchOrderUpdate struct {
	Operation    string                  `json:"op"`
	Topic        string                  `json:"topic"`
	ContractCode string                  `json:"contract_code"`
	Timestamp    types.Time              `json:"ts"`
	UID          string                  `json:"uid"`
	Orders       []V5WebsocketMatchOrder `json:"data"`
}

// V5WebsocketAlgoOrderUpdate contains an authenticated V5 USDT-margined strategy-order update.
type V5WebsocketAlgoOrderUpdate struct {
	Operation    string        `json:"op"`
	Topic        string        `json:"topic"`
	ContractCode string        `json:"contract_code"`
	Timestamp    types.Time    `json:"ts"`
	UID          string        `json:"uid"`
	Orders       []V5AlgoOrder `json:"data"`
}

// V5WebsocketExecution describes a fill rather than a complete order snapshot.
type V5WebsocketExecution struct {
	TradeFee       types.Number        `json:"trade_fee"`
	FeeCurrency    currency.Code       `json:"fee_currency"`
	DeductionPrice types.Number        `json:"deduction_price"`
	ContractCode   string              `json:"contract_code"`
	ID             string              `json:"id"`
	OrderID        types.PreciseNumber `json:"order_id"`
	TradeID        types.PreciseNumber `json:"trade_id"`
	ClientOrderID  string              `json:"client_order_id"`
	Side           string              `json:"side"`
	Direction      string              `json:"direction"`
	PositionSide   string              `json:"position_side"`
	Role           string              `json:"role"`
	TradePrice     types.Number        `json:"trade_price"`
	TradeVolume    types.Number        `json:"trade_volume"`
	TradeTurnover  types.Number        `json:"trade_turnover"`
	CreatedTime    types.Time          `json:"created_time"`
	UpdatedTime    types.Time          `json:"updated_time"`
}

// V5WebsocketMatchOrder carries cumulative order state from the matching engine.
type V5WebsocketMatchOrder struct {
	V5WebsocketOrder
	ID               string       `json:"id"`
	ContractCode     string       `json:"contract_code"`
	TotalTradeVolume types.Number `json:"total_trade_volume"`
	TradePrice       types.Number `json:"trade_price"`
	MatchTime        types.Time   `json:"match_time"`
}

// V5WebsocketPosition contains the fields published by the V5 position channel.
type V5WebsocketPosition struct {
	ContractCode      string        `json:"contract_code"`
	Symbol            currency.Code `json:"symbol"`
	PositionMode      string        `json:"position_mode"`
	PositionSide      string        `json:"position_side"`
	Direction         string        `json:"direction"`
	MarginMode        string        `json:"margin_mode"`
	OpenAveragePrice  types.Number  `json:"open_avg_price"`
	Volume            types.Number  `json:"volume"`
	Available         types.Number  `json:"available"`
	Fee               types.Number  `json:"fee"`
	LeverageRate      types.Number  `json:"lever_rate"`
	ADLRiskPercent    uint64        `json:"adl_risk_percent"`
	LiquidationPrice  types.Number  `json:"liquidation_price"`
	InitialMargin     types.Number  `json:"initial_margin"`
	MaintenanceMargin types.Number  `json:"maintenance_margin"`
	Margin            types.Number  `json:"margin"`
	ProfitUnreal      types.Number  `json:"profit_unreal"`
	Profit            types.Number  `json:"profit"`
	ProfitRate        types.Number  `json:"profit_rate"`
	MarginRate        types.Number  `json:"margin_rate"`
	State             string        `json:"state"`
	FundingFee        types.Number  `json:"funding_fee"`
	LastPrice         types.Number  `json:"last_price"`
	MarkPrice         types.Number  `json:"mark_price"`
	ContractType      string        `json:"contract_type"`
	TotalTradeFee     types.Number  `json:"total_trade_fee"`
	BreakEvenPrice    types.Number  `json:"break_even_price"`
	CreatedTime       types.Time    `json:"created_time"`
	UpdatedTime       types.Time    `json:"updated_time"`
	Version           uint64        `json:"version"`
}
