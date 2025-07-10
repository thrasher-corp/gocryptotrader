package main

// SubmitOrderParams holds submit order parameters
type SubmitOrderParams struct {
	ExchangeName       string  `cli:"exchange,required"`
	CurrencyPair       string  `cli:"pair,required"`
	OrderSide          string  `cli:"side,required"`
	OrderType          string  `cli:"type,required"`
	Amount             float64 `cli:"amount,required"`
	AssetType          string  `cli:"asset,required"`
	Price              float64 `cli:"price"`
	Leverage           float64 `cli:"leverage"`
	ClientOrderID      string  `cli:"client_order_id"`
	MarginType         string  `cli:"margin_type"`
	TimeInForce        string  `cli:"time_in_force"`
	QuoteAmount        float64 `cli:"quote_amount"`
	ClientID           string  `cli:"client_id"`
	TriggerPrice       float64 `cli:"trigger_price"`
	TriggerLimitPrice  float64 `cli:"trigger_limit_price"`
	TriggerPriceType   string  `cli:"trigger_price_type"`
	TpPrice            float64 `cli:"tp_price"`
	TpLimitPrice       float64 `cli:"tp_limit_price"`
	TpPriceType        string  `cli:"tp_price_type"`
	SlPrice            float64 `cli:"sl_price"`
	SlLimitPrice       float64 `cli:"sl_limit_price"`
	SlPriceType        string  `cli:"sl_price_type"`
	TrackingMode       string  `cli:"tracking_mode"`
	TrackingValue      float64 `cli:"tracking_value"`
	Hidden             bool    `cli:"hidden"`
	Iceberg            bool    `cli:"iceberg"`
	AutoBorrow         bool    `cli:"auto_borrow"`
	ReduceOnly         bool    `cli:"reduce_only"`
	RetrieveFees       bool    `cli:"retrieve_fees"`
	RetrieveFeeDelayMs int64   `cli:"retrieve_fee_delay_ms"`
}

// ModifyOrderParams holds an order modification params
type ModifyOrderParams struct {
	ExchangeName      string  `cli:"exchange,required"`
	AssetType         string  `cli:"asset,required"`
	CurrencyPair      string  `cli:"pair,required"`
	OrderID           string  `cli:"order_id"`
	OrderType         string  `cli:"type"`
	OrderSide         string  `cli:"side"`
	Price             float64 `cli:"price"`
	Amount            float64 `cli:"amount"`
	ClientOrderID     string  `cli:"client_order_id"`
	TimeInForce       string  `cli:"time_in_force"`
	TriggerPrice      float64 `cli:"trigger_price"`
	TriggerLimitPrice float64 `cli:"trigger_limit_price"`
	TriggerPriceType  string  `cli:"trigger_price_type"`
	TpPrice           float64 `cli:"tp_price"`
	TpLimitPrice      float64 `cli:"tp_limit_price"`
	TpPriceType       string  `cli:"tp_price_type"`
	SlPrice           float64 `cli:"sl_price"`
	SlLimitPrice      float64 `cli:"sl_limit_price"`
	SlPriceType       string  `cli:"sl_price_type"`
}

// CancelOrderParams holds an order cancellation params
type CancelOrderParams struct {
	Exchange      string `cli:"exchange,required"`
	OrderID       string `cli:"order_id,required"`
	ClientOrderID string `cli:"client_order_id"`
	AccountID     string `cli:"account_id"`
	ClientID      string `cli:"client_id"`
	OrderType     string `cli:"type"`
	OrderSide     string `cli:"side"`
	AssetType     string `cli:"asset"`
	CurrencyPair  string `cli:"pair"`
	MarginType    string `cli:"margin_type"`
	TimeInForce   string `cli:"time_in_force"`
}
