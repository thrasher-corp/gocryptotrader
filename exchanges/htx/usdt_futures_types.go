package htx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

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
	ContractCode     string       `json:"contract_code"`
	MarginMode       string       `json:"margin_mode"`
	PositionSide     string       `json:"position_side,omitempty"`
	Side             string       `json:"side"`
	Type             string       `json:"type"`
	PriceMatch       string       `json:"price_match,omitempty"`
	ClientOrderID    string       `json:"client_order_id,omitempty"`
	Price            types.Number `json:"price,omitempty"`
	Volume           types.Number `json:"volume"`
	ReduceOnly       int64        `json:"reduce_only,omitempty"`
	TimeInForce      string       `json:"time_in_force,omitempty"`
	SelfMatchPrevent string       `json:"self_match_prevent,omitempty"`
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
	ID                string        `json:"id"`
	ContractCode      string        `json:"contract_code"`
	Side              string        `json:"side"`
	PositionSide      string        `json:"position_side"`
	Type              string        `json:"type"`
	PriceMatch        string        `json:"price_match"`
	OrderID           string        `json:"order_id"`
	ClientOrderID     string        `json:"client_order_id"`
	MarginMode        string        `json:"margin_mode"`
	Price             types.Number  `json:"price"`
	Volume            types.Number  `json:"volume"`
	LeverageRate      types.Number  `json:"lever_rate"`
	State             string        `json:"state"`
	OrderSource       string        `json:"order_source"`
	ReduceOnly        types.Boolean `json:"reduce_only"`
	TimeInForce       string        `json:"time_in_force"`
	TradeAveragePrice types.Number  `json:"trade_avg_price"`
	TradeVolume       types.Number  `json:"trade_volume"`
	TradeTurnover     types.Number  `json:"trade_turnover"`
	FeeCurrency       string        `json:"fee_currency"`
	Fee               types.Number  `json:"fee"`
	Profit            types.Number  `json:"profit"`
	ContractType      string        `json:"contract_type"`
	CreatedTime       types.Time    `json:"created_time"`
	UpdatedTime       types.Time    `json:"updated_time"`
	CancelReason      string        `json:"cancel_reason"`
	SelfMatchPrevent  string        `json:"self_match_prevent"`
}
