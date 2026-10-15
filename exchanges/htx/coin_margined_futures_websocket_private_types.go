package htx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapWsSubOrderData stores subscribed order data for swap websocket
type SwapWsSubOrderData struct {
	Operation       string                 `json:"op"`
	Topic           string                 `json:"topic"`
	UID             string                 `json:"uid"`
	Timestamp       types.Time             `json:"ts"`
	Symbol          string                 `json:"symbol"`
	ContractCode    string                 `json:"contract_code"`
	ContractType    string                 `json:"contract_type"`
	Pair            string                 `json:"pair"`
	BusinessType    string                 `json:"business_type"`
	MarginMode      string                 `json:"margin_mode"`
	MarginAccount   string                 `json:"margin_account"`
	Volume          float64                `json:"volume"`
	Price           float64                `json:"price"`
	OrderPriceType  string                 `json:"order_price_type"`
	Direction       string                 `json:"direction"`
	Offset          string                 `json:"offset"`
	Status          int64                  `json:"status"`
	LeverageRate    float64                `json:"lever_rate"`
	OrderID         int64                  `json:"order_id"`
	OrderIDString   string                 `json:"order_id_str"`
	ClientOrderID   int64                  `json:"client_order_id"`
	OrderSource     string                 `json:"order_source"`
	OrderType       int64                  `json:"order_type"`
	CreatedAt       int64                  `json:"created_at"`
	CanceledAt      int64                  `json:"canceled_at"`
	TradeVolume     float64                `json:"trade_volume"`
	TradeTurnover   float64                `json:"trade_turnover"`
	Fee             float64                `json:"fee"`
	FeeAsset        string                 `json:"fee_asset"`
	TradeAvgPrice   float64                `json:"trade_avg_price"`
	MarginFrozen    float64                `json:"margin_frozen"`
	Profit          float64                `json:"profit"`
	RealProfit      float64                `json:"real_profit"`
	ReduceOnly      int64                  `json:"reduce_only"`
	IsTPSL          int64                  `json:"is_tpsl"`
	Trade           []LegacyFuturesWSTrade `json:"trade"`
	LiquidationType string                 `json:"liquidation_type"`
}

// SwapWsSubMatchOrderData stores subscribed match order data for swap websocket
type SwapWsSubMatchOrderData struct {
	Operation      string                 `json:"op"`
	Topic          string                 `json:"topic"`
	UID            string                 `json:"uid"`
	Timestamp      types.Time             `json:"ts"`
	Symbol         string                 `json:"symbol"`
	ContractCode   string                 `json:"contract_code"`
	ContractType   string                 `json:"contract_type"`
	Pair           string                 `json:"pair"`
	BusinessType   string                 `json:"business_type"`
	MarginMode     string                 `json:"margin_mode"`
	MarginAccount  string                 `json:"margin_account"`
	Status         int64                  `json:"status"`
	OrderID        int64                  `json:"order_id"`
	OrderIDString  string                 `json:"order_id_str"`
	ClientOrderID  int64                  `json:"client_order_id"`
	OrderType      int64                  `json:"order_type"`
	TradeVolume    float64                `json:"trade_volume"`
	Volume         float64                `json:"volume"`
	Direction      string                 `json:"direction"`
	Offset         string                 `json:"offset"`
	LeverageRate   float64                `json:"lever_rate"`
	Price          float64                `json:"price"`
	OrderSource    string                 `json:"order_source"`
	OrderPriceType string                 `json:"order_price_type"`
	CreatedAt      int64                  `json:"created_at"`
	ReduceOnly     int64                  `json:"reduce_only"`
	IsTPSL         int64                  `json:"is_tpsl"`
	Trade          []LegacyFuturesWSTrade `json:"trade"`
}

// SwapWsContractDetail stores per-contract margin data included in cross-margin account notifications.
type SwapWsContractDetail struct {
	Symbol           string  `json:"symbol"`
	ContractCode     string  `json:"contract_code"`
	MarginPosition   float64 `json:"margin_position"`
	MarginFrozen     float64 `json:"margin_frozen"`
	MarginAvailable  float64 `json:"margin_available"`
	ProfitUnreal     float64 `json:"profit_unreal"`
	LiquidationPrice float64 `json:"liquidation_price"`
	LeverageRate     float64 `json:"lever_rate"`
	AdjustFactor     float64 `json:"adjust_factor"`
	ContractType     string  `json:"contract_type"`
	Pair             string  `json:"pair"`
	BusinessType     string  `json:"business_type"`
}

// SwapWsSubEquityData stores subscribed account data for swap account equity updates through websocket
type SwapWsSubEquityData struct {
	Operation string     `json:"op"`
	Topic     string     `json:"topic"`
	Timestamp types.Time `json:"ts"`
	UID       string     `json:"uid"`
	Event     string     `json:"event"`
	Data      []struct {
		Symbol                string                 `json:"symbol"`
		ContractCode          string                 `json:"contract_code"`
		MarginMode            string                 `json:"margin_mode"`
		MarginAccount         string                 `json:"margin_account"`
		MarginAsset           string                 `json:"margin_asset"`
		PositionMode          string                 `json:"position_mode"`
		MarginBalance         float64                `json:"margin_balance"`
		MarginStatic          float64                `json:"margin_static"`
		MarginPosition        float64                `json:"margin_position"`
		MarginFrozen          float64                `json:"margin_frozen"`
		MarginAvailable       float64                `json:"margin_available"`
		ProfitReal            float64                `json:"profit_real"`
		ProfitUnreal          float64                `json:"profit_unreal"`
		WithdrawAvailable     float64                `json:"withdraw_available"`
		RiskRate              float64                `json:"risk_rate"`
		LiquidationPrice      float64                `json:"liquidation_price"`
		LeverageRate          float64                `json:"lever_rate"`
		AdjustFactor          float64                `json:"adjust_factor"`
		ContractDetail        []SwapWsContractDetail `json:"contract_detail"`
		FuturesContractDetail []SwapWsContractDetail `json:"futures_contract_detail"`
	} `json:"data"`
}

// SwapWsSubPositionUpdates stores subscribed position updates data for swap websocket
type SwapWsSubPositionUpdates struct {
	Operation string     `json:"op"`
	Topic     string     `json:"topic"`
	UID       string     `json:"uid"`
	Timestamp types.Time `json:"ts"`
	Event     string     `json:"event"`
	Data      []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractType   string  `json:"contract_type"`
		Pair           string  `json:"pair"`
		BusinessType   string  `json:"business_type"`
		MarginAsset    string  `json:"margin_asset"`
		MarginMode     string  `json:"margin_mode"`
		MarginAccount  string  `json:"margin_account"`
		PositionMode   string  `json:"position_mode"`
		Volume         float64 `json:"volume"`
		Available      float64 `json:"available"`
		Frozen         float64 `json:"frozen"`
		CostOpen       float64 `json:"cost_open"`
		CostHold       float64 `json:"cost_hold"`
		ProfitUnreal   float64 `json:"profit_unreal"`
		ProfitRate     float64 `json:"profit_rate"`
		Profit         float64 `json:"profit"`
		PositionMargin float64 `json:"position_margin"`
		LeverageRate   float64 `json:"lever_rate"`
		Direction      string  `json:"direction"`
		LastPrice      float64 `json:"last_price"`
	} `json:"data"`
}

// SwapWsSubTriggerOrderUpdates stores subscribed trigger order updates data for swap websocket
type SwapWsSubTriggerOrderUpdates struct {
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	UID       string `json:"uid"`
	Event     string `json:"event"`
	Data      []struct {
		Symbol          string  `json:"symbol"`
		ContractCode    string  `json:"contract_code"`
		ContractType    string  `json:"contract_type"`
		Pair            string  `json:"pair"`
		BusinessType    string  `json:"business_type"`
		MarginMode      string  `json:"margin_mode"`
		MarginAccount   string  `json:"margin_account"`
		Volume          float64 `json:"volume"`
		OrderType       int64   `json:"order_type"`
		Direction       string  `json:"direction"`
		Offset          string  `json:"offset"`
		LeverageRate    int64   `json:"lever_rate"`
		OrderID         int64   `json:"order_id"`
		OrderIDString   string  `json:"order_id_str"`
		RelationOrderID string  `json:"relation_order_id"`
		OrderPriceType  string  `json:"order_price_type"`
		Status          int64   `json:"status"`
		OrderSource     string  `json:"order_source"`
		TriggerPrice    float64 `json:"trigger_price"`
		TriggeredPrice  float64 `json:"triggered_price"`
		OrderPrice      float64 `json:"order_price"`
		CreatedAt       int64   `json:"created_at"`
		TriggeredAt     int64   `json:"triggered_at"`
		OrderInsertAt   int64   `json:"order_insert_at"`
		CancelledAt     int64   `json:"canceled_at"`
		FailCode        int64   `json:"fail_code"`
		FailReason      string  `json:"fail_reason"`
		ReduceOnly      int64   `json:"reduce_only"`
	} `json:"data"`
}
