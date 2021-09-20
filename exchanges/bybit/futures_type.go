package bybit

var (
	validFuturesIntervals = []string{
		"1", "3", "5", "15", "30", "60", "120", "240", "360", "720",
		"D", "M", "W", "d", "m", "w",
	}

	validFuturesPeriods = []string{
		"5min", "15min", "30min", "1h", "4h", "1d",
	}
)

// OrderbookData stores ob data for cmargined futures
type OrderbookData struct {
	Symbol int64  `json:"symbol"`
	Price  string `json:"price"`
	Size   int64  `json:"size"`
	Side   string `json:"side"`
}

// FuturesCandleStick holds kline data
type FuturesCandleStick struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	OpenTime int64   `json:"open_time"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
	TurnOver float64 `json:"turnover"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol                 string `json:"symbol"`
	BidPrice               string `json:"bid_price"`
	AskPrice               string `json:"ask_price"`
	LastPrice              string `json:"last_price"`
	LastTickDirection      string `json:"last_tick_direction"`
	Price24hAgo            string `json:"prev_price_24h"`
	PricePcntChange24h     string `json:"price_24h_pcnt"`
	HighPrice24h           string `json:"high_price_24h"`
	LowPrice24h            string `json:"low_price_24h"`
	Price1hAgo             string `json:"prev_price_1h"`
	PricePcntChange1h      string `json:"price_1h_pcnt"`
	MarkPrice              string `json:"mark_price"`
	IndexPrice             string `json:"index_price"`
	OpenInterest           int64  `json:"open_interest"`
	OpenValue              string `json:"open_value"`
	TotalTurnover          string `json:"total_turnover"`
	Turnover24h            string `json:"turnover_24h"`
	TotalVolume            int64  `json:"total_volume"`
	Volume24h              int64  `json:"volume_24h"`
	FundingRate            string `json:"funding_rate"`
	PredictedFundingRate   string `json:"predicted_funding_rate"`
	NextFundingTime        string `json:"next_funding_time"`
	CountdownHour          int64  `json:"countdown_hour"`
	DeliveryFeeRate        string `json:"delivery_fee_rate"`
	PredictedDeliveryPrice string `json:"predicted_delivery_price"`
	DeliveryTime           string `json:"delivery_time"`
}

// FuturesPublicTradesData stores recent public trades for futures
type FuturesPublicTradesData struct {
	ID     int64   `json:"id"`
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
	Qty    float64 `json:"qty,string"`
	Time   int64   `json:"time"`
	Side   string  `json:"side"`
}

// SymbolInfo stores symbol information for futures pair
type SymbolInfo struct {
	Name           string `json:"name"`
	Alias          string `json:"alias"`
	Status         string `json:"status"`
	BaseCurrency   string `json:"base_currency"`
	QuoteCurrency  string `json:"quote_currency"`
	PriceScale     int64  `json:"price_scale"`
	TakerFee       string `json:"taker_fee"`
	MakerFee       string `json:"maker_fee"`
	LeverageFilter struct {
		MinLeverage  int64   `json:"min_leverage"`
		MaxLeverage  int64   `json:"max_leverage"`
		LeverageStep float64 `json:"leverage_step,string"`
	} `json:"leverage_filter"`
	PriceFilter struct {
		MinPrice float64 `json:"min_price,string"`
		MaxPrice float64 `json:"max_price,string"`
		TickSize float64 `json:"tick_size,string"`
	} `json:"price_filter"`
	LotSizeFilter struct {
		MinTradeQty float64 `json:"min_trading_qty,string"`
		MaxTradeQty float64 `json:"max_trading_qty,string"`
		QtyStep     float64 `json:"qty_step,string"`
	} `json:"lot_size_filter"`
}

// AllLiquidationOrders gets all liquidation orders
type AllLiquidationOrders struct {
	ID     int64   `json:"id"`
	Qty    float64 `json:"origQty,string"`
	Side   string  `json:"side"`
	Time   int64   `json:"time"`
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type MarkPriceKlineData struct {
	ID       int64  `json:"id"`
	Symbol   string `json:"symbol"`
	Interval string `json:"period"`
	StartAt  int64  `json:"start_at"`
	Open     int64  `json:"open"`
	High     int64  `json:"high"`
	Low      int64  `json:"low"`
	Close    int64  `json:"close"`
}

type IndexPriceKlineData struct {
	Symbol   string `json:"symbol"`
	Interval string `json:"period"`
	StartAt  int64  `json:"open_time"`
	Open     int64  `json:"open"`
	High     int64  `json:"high"`
	Low      int64  `json:"low"`
	Close    int64  `json:"close"`
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	OpenInterest int64  `json:"open_interest"`
	Symbol       string `json:"symbol"`
	Time         int64  `json:"time"`
}

type BigDealData struct {
	Symbol string `json:"symbol"`
	Side   string `json:"side"`
	Time   int64  `json:"timestamp"`
	Value  int64  `json:"value"`
}

// AccountRatioData stores user accounts long short ratio
type AccountRatioData struct {
	Symbol    string  `json:"symbol"`
	BuyRatio  float64 `json:"buy_ratio"`
	SellRatio float64 `json:"sell_ratio"`
	Time      int64   `json:"timestamp"`
}

// FuturesOrderData stores futures order data
type FuturesOrderData struct {
	UserID              int64   `json:"user_id"`
	OrderID             string  `json:"order_id"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	OrderType           string  `json:"order_type"`
	Price               float64 `json:"price"`
	Qty                 float64 `json:"qty"`
	TimeInForce         string  `json:"time_in_force"`
	OrderStatus         string  `json:"order_status"`
	LastExecutionTime   string  `json:"last_exec_time"`
	LastExecutionPrice  string  `json:"last_exec_price"`
	LeavesQty           float64 `json:"leaves_qty"`
	CumulativeQty       float64 `json:"cum_exec_qty"`
	CumulativeValue     float64 `json:"cum_exec_value"`
	CumulativeFee       float64 `json:"cum_exec_fee"`
	RejectReason        string  `json:"reject_reason"`
	OrderLinkID         string  `json:"order_link_id"`
	CreatedAt           string  `json:"create_at"`
	UpdateAt            string  `json:"updated_at"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

type FuturesActiveOrders struct {
	Data []struct {
		UserID              int64   `json:"user_id"`
		Symbol              string  `json:"symbol"`
		Side                string  `json:"side"`
		OrderType           string  `json:"order_type"`
		Price               float64 `json:"price"`
		Qty                 float64 `json:"qty"`
		TimeInForce         string  `json:"time_in_force"`
		OrderStatus         string  `json:"order_status"`
		LeavesQty           float64 `json:"leaves_qty"`
		LeaveValue          float64 `json:"leaves_value"`
		CumulativeQty       float64 `json:"cum_exec_qty"`
		CumulativeValue     float64 `json:"cum_exec_value"`
		CumulativeFee       float64 `json:"cum_exec_fee"`
		RejectReason        string  `json:"reject_reason"`
		OrderLinkID         string  `json:"order_link_id"`
		CreatedAt           string  `json:"create_at"`
		OrderID             string  `json:"order_id"`
		TakeProfit          float64 `json:"take_profit"`
		StopLoss            float64 `json:"stop_loss"`
		TakeProfitTriggerBy string  `json:"tp_trigger_by"`
		StopLossTriggerBy   string  `json:"sl_trigger_by"`
	} `json:"data"`
	Cursor string `json:"cursor"`
}
