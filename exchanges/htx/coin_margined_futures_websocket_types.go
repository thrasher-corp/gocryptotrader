package htx

import "github.com/thrasher-corp/gocryptotrader/types"

// WsSwapReqKline stores req kline data for swap websocket
type WsSwapReqKline struct {
	Rep  string `json:"rep"`
	ID   string `json:"id"`
	WsID int64  `json:"wsid"`
	Tick []struct {
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"tick"`
}

// WsSwapReqTradeDetail stores requested trade detail data for swap websocket
type WsSwapReqTradeDetail struct {
	Rep       string     `json:"rep"`
	ID        int64      `json:"id"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID        int64      `json:"id"`
		Price     float64    `json:"price"`
		Amount    float64    `json:"amount"`
		Direction string     `json:"direction"`
		Timestamp types.Time `json:"ts"`
	} `json:"data"`
}

// SwapWsSubPremiumKline stores subscribed premium kline data for futures websocket
type SwapWsSubPremiumKline struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"tick"`
}

// SwapWsReqPremiumKline stores requested premium kline data for futures websocket
type SwapWsReqPremiumKline struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"data"`
}

// SwapWsSubEstimatedFunding stores estimated funding rate data for swap websocket
type SwapWsSubEstimatedFunding struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64        `json:"id"`
		Volume types.Number `json:"vol"`
		Count  types.Number `json:"count"`
		Open   types.Number `json:"open"`
		Close  types.Number `json:"close"`
		Low    types.Number `json:"low"`
		High   types.Number `json:"high"`
		Amount types.Number `json:"amount"`
	} `json:"tick"`
}

// SwapWsReqEstimatedFunding stores requested estimated funding data for swap websocket
type SwapWsReqEstimatedFunding struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Volume types.Number `json:"vol"`
		Count  types.Number `json:"count"`
		ID     int64        `json:"id"`
		Open   types.Number `json:"open"`
		Close  types.Number `json:"close"`
		Low    types.Number `json:"low"`
		High   types.Number `json:"high"`
		Amount types.Number `json:"amount"`
	}
}

// SwapWsSubBasisData stores subscribed basis data for swap websocket
type SwapWsSubBasisData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      []struct {
		ID            int64        `json:"id"`
		ContractPrice types.Number `json:"contract_price"`
		IndexPrice    types.Number `json:"index_price"`
		Basis         types.Number `json:"basis"`
		BasisRate     types.Number `json:"basis_rate"`
	} `json:"tick"`
}

// SwapWsReqBasisData stores requested basis data for swap websocket
type SwapWsReqBasisData struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID            int64   `json:"id"`
		ContractPrice float64 `json:"contract_price"`
		IndexPrice    float64 `json:"index_price"`
		Basis         float64 `json:"basis"`
		BasisRate     float64 `json:"basis_rate"`
	}
}

// SwapWsSubOrderData stores subscribed order data for swap websocket
type SwapWsSubOrderData struct {
	Operation      string     `json:"op"`
	Topic          string     `json:"topic"`
	UID            string     `json:"uid"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	Pair           string     `json:"pair"`
	BusinessType   string     `json:"business_type"`
	MarginMode     string     `json:"margin_mode"`
	MarginAccount  string     `json:"margin_account"`
	Volume         float64    `json:"volume"`
	Price          float64    `json:"price"`
	OrderPriceType string     `json:"order_price_type"`
	Direction      string     `json:"direction"`
	Offset         string     `json:"offset"`
	Status         int64      `json:"status"`
	LeverageRate   float64    `json:"lever_rate"`
	OrderID        int64      `json:"order_id"`
	OrderIDString  string     `json:"order_id_str"`
	ClientOrderID  int64      `json:"client_order_id"`
	OrderSource    string     `json:"order_source"`
	OrderType      int64      `json:"order_type"`
	CreatedAt      int64      `json:"created_at"`
	CanceledAt     int64      `json:"canceled_at"`
	TradeVolume    float64    `json:"trade_volume"`
	TradeTurnover  float64    `json:"trade_turnover"`
	Fee            float64    `json:"fee"`
	FeeAsset       string     `json:"fee_asset"`
	TradeAvgPrice  float64    `json:"trade_avg_price"`
	MarginFrozen   float64    `json:"margin_frozen"`
	Profit         float64    `json:"profit"`
	RealProfit     float64    `json:"real_profit"`
	ReduceOnly     int64      `json:"reduce_only"`
	IsTPSL         int64      `json:"is_tpsl"`
	Trade          []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeFee      float64 `json:"trade_fee"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		FeeAsset      string  `json:"fee_asset"`
		Role          string  `json:"role"`
	} `json:"trade"`
	LiquidationType string `json:"liquidation_type"`
}

// SwapWsSubMatchOrderData stores subscribed match order data for swap websocket
type SwapWsSubMatchOrderData struct {
	Operation      string     `json:"op"`
	Topic          string     `json:"topic"`
	UID            string     `json:"uid"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	Pair           string     `json:"pair"`
	BusinessType   string     `json:"business_type"`
	MarginMode     string     `json:"margin_mode"`
	MarginAccount  string     `json:"margin_account"`
	Status         int64      `json:"status"`
	OrderID        int64      `json:"order_id"`
	OrderIDString  string     `json:"order_id_str"`
	ClientOrderID  int64      `json:"client_order_id"`
	OrderType      string     `json:"order_type"`
	TradeVolume    float64    `json:"trade_volume"`
	Volume         float64    `json:"volume"`
	Direction      string     `json:"direction"`
	Offset         string     `json:"offset"`
	LeverageRate   float64    `json:"lever_rate"`
	Price          float64    `json:"price"`
	OrderSource    string     `json:"order_source"`
	OrderPriceType string     `json:"order_price_type"`
	CreatedAt      int64      `json:"created_at"`
	ReduceOnly     int64      `json:"reduce_only"`
	IsTPSL         int64      `json:"is_tpsl"`
	Trade          []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		Role          string  `json:"role"`
	} `json:"trade"`
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

// SwapWsSubLiquidationOrders stores subscribed liquidation orders data for swap futures
type SwapWsSubLiquidationOrders struct {
	Operation  string     `json:"op"`
	Topic      string     `json:"topic"`
	Timestamp  types.Time `json:"ts"`
	OrdersData []struct {
		Symbol       string     `json:"symbol"`
		ContractCode string     `json:"contract_code"`
		Direction    string     `json:"direction"`
		Offset       string     `json:"offset"`
		Volume       float64    `json:"volume"`
		Price        float64    `json:"price"`
		CreatedAt    types.Time `json:"created_at"`
	} `json:"data"`
}

// SwapWsSubFundingData stores funding rate data for swap websocket
type SwapWsSubFundingData struct {
	Operation   string     `json:"op"`
	Topic       string     `json:"topic"`
	Timestamp   types.Time `json:"ts"`
	FundingData []struct {
		Symbol         string       `json:"symbol"`
		ContractCode   string       `json:"contract_code"`
		FeeAsset       string       `json:"fee_asset"`
		FundingTime    types.Time   `json:"funding_time"`
		FundingRate    types.Number `json:"funding_rate"`
		EstimatedRate  types.Number `json:"estimated_rate"`
		SettlementTime types.Time   `json:"settlement_time"`
	} `json:"data"`
}

// SwapWsSubContractInfo stores funding rate data for swap websocket
type SwapWsSubContractInfo struct {
	Operation    string     `json:"op"`
	Topic        string     `json:"topic"`
	Timestamp    types.Time `json:"ts"`
	Event        string     `json:"event"`
	ContractData []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractSize   float64 `json:"contract_size"`
		PriceTick      float64 `json:"price_tick"`
		SettlementDate string  `json:"settlement_date"`
		CreateDate     string  `json:"create_date"`
		ContractStatus int64   `json:"contract_status"`
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
