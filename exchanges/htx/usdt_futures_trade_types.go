package htx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5BatchOrderResponse stores per-order acknowledgements.
type V5BatchOrderResponse struct {
	V5Response
	Data []V5OrderResponseData `json:"data"`
}

// V5CancelBatchOrdersRequest defines a batch cancellation.
type V5CancelBatchOrdersRequest struct {
	ContractCode   string   `json:"contract_code"`
	OrderIDs       []string `json:"order_id,omitempty"`
	ClientOrderIDs []string `json:"client_order_id,omitempty"`
}

// V5CancelAfterRequest defines automatic order cancellation.
type V5CancelAfterRequest struct {
	Enabled string       `json:"on_off"`
	Timeout types.Number `json:"time_out,omitempty"`
}

// V5CancelAfterResponse stores the automatic-cancellation schedule.
type V5CancelAfterResponse struct {
	V5Response
	Data struct {
		CurrentTime types.Time `json:"current_time"`
		TriggerTime types.Time `json:"trigger_time"`
	} `json:"data"`
}

// V5ClosePositionRequest defines a market position close.
type V5ClosePositionRequest struct {
	ContractCode  string `json:"contract_code"`
	MarginMode    string `json:"margin_mode"`
	PositionSide  string `json:"position_side"`
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// V5OrderHistoryRequest defines historical-order filters.
type V5OrderHistoryRequest struct {
	ContractCode string
	MarginMode   string
	States       string
	Type         string
	PriceMatch   string
	TimeInForce  string
	StartTime    time.Time
	EndTime      time.Time
	From         uint64
	Limit        uint64
	Direction    string
}

// V5OrderDetailsRequest defines execution-detail filters.
type V5OrderDetailsRequest struct {
	ContractCode string
	OrderID      string
	StartTime    time.Time
	EndTime      time.Time
	From         string
	Limit        uint64
	Direction    string
}

// V5OrderDetailsResponse stores recent order executions.
type V5OrderDetailsResponse struct {
	V5Response
	Data []V5OrderDetail `json:"data"`
}

// V5OrderDetail stores one futures execution.
type V5OrderDetail struct {
	ID             string       `json:"id"`
	ContractCode   string       `json:"contract_code"`
	OrderID        string       `json:"order_id"`
	TradeID        string       `json:"trade_id"`
	Side           string       `json:"side"`
	PositionSide   string       `json:"position_side"`
	OrderType      string       `json:"order_type"`
	MarginMode     string       `json:"margin_mode"`
	Type           string       `json:"type"`
	Role           string       `json:"role"`
	TradePrice     types.Number `json:"trade_price"`
	TradeVolume    types.Number `json:"trade_volume"`
	TradeTurnover  types.Number `json:"trade_turnover"`
	CreatedTime    types.Time   `json:"created_time"`
	UpdatedTime    types.Time   `json:"updated_time"`
	OrderSource    string       `json:"order_source"`
	FeeCurrency    string       `json:"fee_currency"`
	TradeFee       types.Number `json:"trade_fee"`
	DeductionPrice types.Number `json:"deduction_price"`
	Profit         types.Number `json:"profit"`
	ContractType   string       `json:"contract_type"`
}

// V5OpenPositionsResponse stores current positions.
type V5OpenPositionsResponse struct {
	V5Response
	Data []V5OpenPosition `json:"data"`
}

// V5OpenPosition stores a current position.
type V5OpenPosition struct {
	ContractCode      string       `json:"contract_code"`
	PositionSide      string       `json:"position_side"`
	Direction         string       `json:"direction"`
	OpenAveragePrice  types.Number `json:"open_avg_price"`
	MarginMode        string       `json:"margin_mode"`
	Volume            types.Number `json:"volume"`
	Available         types.Number `json:"available"`
	LeverageRate      uint64       `json:"lever_rate"`
	ADLRiskPercent    types.Number `json:"adl_risk_percent"`
	LiquidationPrice  types.Number `json:"liquidation_price"`
	InitialMargin     types.Number `json:"initial_margin"`
	MaintenanceMargin types.Number `json:"maintenance_margin"`
	Margin            types.Number `json:"margin"`
	ProfitUnreal      types.Number `json:"profit_unreal"`
	ProfitRate        types.Number `json:"profit_rate"`
	MarginRate        types.Number `json:"margin_rate"`
	MarginCurrency    string       `json:"margin_currency"`
	LastPrice         types.Number `json:"last_price"`
	MarkPrice         types.Number `json:"mark_price"`
	ContractType      string       `json:"contract_type"`
	CreatedTime       types.Time   `json:"created_time"`
	UpdatedTime       types.Time   `json:"updated_time"`
}
