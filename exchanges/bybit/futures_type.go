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
}

// FuturesRealtimeOrderData stores futures realtime order data
type FuturesRealtimeOrderData struct {
	UserID              int64                  `json:"user_id"`
	Symbol              string                 `json:"symbol"`
	Side                string                 `json:"side"`
	OrderType           string                 `json:"order_type"`
	Price               float64                `json:"price"`
	Qty                 float64                `json:"qty"`
	TimeInForce         string                 `json:"time_in_force"`
	OrderStatus         string                 `json:"order_status"`
	ExtensionField      map[string]interface{} `json:"ext_fields"`
	LeavesQty           float64                `json:"leaves_qty"`
	LeaveValue          float64                `json:"leaves_value"`
	CumulativeQty       float64                `json:"cum_exec_qty"`
	CumulativeValue     float64                `json:"cum_exec_value"`
	CumulativeFee       float64                `json:"cum_exec_fee"`
	RejectReason        string                 `json:"reject_reason"`
	OrderLinkID         string                 `json:"order_link_id"`
	CreatedAt           string                 `json:"create_at"`
	UpdatedAt           string                 `json:"updated_at"`
	OrderID             string                 `json:"order_id"`
	TakeProfit          float64                `json:"take_profit"`
	StopLoss            float64                `json:"stop_loss"`
	TakeProfitTriggerBy string                 `json:"tp_trigger_by"`
	StopLossTriggerBy   string                 `json:"sl_trigger_by"`
}

type FuturesActiveRealtimeOrder struct {
	FuturesRealtimeOrderData
	CancelType string `json:"cancel_type"`
}

type FuturesConditionalRealtimeOrder struct {
	FuturesRealtimeOrderData
	BasePrice string `json:"base_price"`
	StopPrice string `json:"stop_px"`
	TriggerBy string `json:"trigger_by"`
}

// FuturesConditionalOrderData stores futures conditional order data
type FuturesConditionalOrderData struct {
	UserID              int64   `json:"user_id"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	OrderType           string  `json:"order_type"`
	Price               float64 `json:"price"`
	Qty                 float64 `json:"qty"`
	TimeInForce         string  `json:"time_in_force"`
	TriggerBy           string  `json:"trigger_by"`
	BasePrice           string  `json:"base_price"`
	Remark              string  `json:"remark"`
	RejectReason        string  `json:"reject_reason"`
	StopPrice           string  `json:"stop_px"`
	StopOrderID         string  `json:"stop_order_id"`
	OrderLinkID         string  `json:"order_link_id"`
	CreatedAt           string  `json:"create_at"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

type FuturesConditionalOrders struct {
	UserID              int64   `json:"user_id"`
	StopOrderStatus     string  `json:"stop_order_status"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	OrderType           string  `json:"order_type"`
	Price               float64 `json:"price"`
	Qty                 float64 `json:"qty"`
	TimeInForce         string  `json:"time_in_force"`
	StopOrderType       string  `json:"stop_order_type"`
	TriggerBy           string  `json:"trigger_by"`
	BasePrice           string  `json:"base_price"`
	OrderLinkID         string  `json:"order_link_id"`
	CreatedAt           string  `json:"create_at"`
	UpdatedAt           string  `json:"updated_at"`
	StopPrice           string  `json:"stop_px"`
	StopOrderID         string  `json:"stop_order_id"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

type FuturesCancelOrderData struct {
	CancelOrderID     string  `json:"clOrdID"`
	UserID            int64   `json:"user_id"`
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"`
	OrderType         string  `json:"order_type"`
	Price             float64 `json:"price"`
	Qty               float64 `json:"qty"`
	TimeInForce       string  `json:"time_in_force"`
	CreateType        string  `json:"create_type"`
	CancelType        string  `json:"cancel_type"`
	OrderStatus       string  `json:"order_status"`
	LeavesQty         float64 `json:"leaves_qty"`
	LeavesValue       float64 `json:"leaves_value"`
	CreatedAt         string  `json:"create_at"`
	UpdateAt          string  `json:"updated_at"`
	CrossStatus       string  `json:"cross_status"`
	CrossSeq          int64   `json:"cross_seq"`
	StopOrderType     string  `json:"stop_order_type"`
	TriggerBy         string  `json:"trigger_by"`
	BasePrice         float64 `json:"base_price,string"`
	ExpectedDirection string  `json:"expected_direction"`
}

type RiskInfo struct {
	ID             int64    `json:"id"`
	Symbol         string   `json:"symbol"`
	Limit          int64    `json:"limit"`
	MaintainMargin float64  `json:"maintain_margin"`
	StartingMargin float64  `json:"starting_margin"`
	Section        []string `json:"section"`
	IsLowestRisk   int64    `json:"is_lowest_risk"`
	CreatedAt      string   `json:"create_at"`
	UpdateAt       string   `json:"updated_at"`
	MaxLeverage    float64  `json:"max_leverage"`
}

type FundingInfo struct {
	Symbol               string  `json:"symbol"`
	FundingRate          float64 `json:"funding_rate,string"`
	FundingRateTimestamp int64   `json:"funding_rate_timestamp"`
}

type AnnouncementInfo struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

type Position struct {
	ID                     int64                  `json:"id"`
	PositionID             int64                  `json:"position_idx"`
	Mode                   int64                  `json:"mode"`
	UserID                 int64                  `json:"user_id"`
	RiskID                 int64                  `json:"risk_id"`
	Symbol                 string                 `json:"symbol"`
	Side                   string                 `json:"side"`
	Size                   int64                  `json:"size"`
	PositionValue          float64                `json:"position_value,string"`
	EntryPrice             float64                `json:"entry_price,string"`
	IsIsolated             bool                   `json:"is_isolated"`
	AutoAddMargin          int64                  `json:"auto_add_margin"`
	Leverage               int64                  `json:"leverage"`
	EffectiveLeverage      int64                  `json:"effective_leverage"`
	PositionMargin         float64                `json:"position_margin,string"`
	LiquidationPrice       int64                  `json:"liq_price"`
	BankruptcyPrice        int64                  `json:"bust_price"`
	OccupiedClosingFee     float64                `json:"occ_closing_fee,string"`
	OccupiedFundingFee     float64                `json:"occ_funding_fee"`
	TakeProfit             float64                `json:"take_profit,string"`
	StopLoss               float64                `json:"stop_loss,string"`
	TrailingStop           float64                `json:"trailing_stop,string"`
	PositionStatus         string                 `json:"position_status"`
	DeleverageIndicator    int64                  `json:"deleverage_indicator"`
	CalculatedData         string                 `json:"oc_calc_data"`
	OrderMargin            float64                `json:"order_margin,string"`
	WalletBalance          float64                `json:"wallet_balance,string"`
	RealisedPNL            float64                `json:"realised_pnl,string"`
	UnrealisedPNL          float64                `json:"unrealised_pnl"`
	AccumulatedRealisedPNL float64                `json:"cum_realised_pnl,string"`
	CrossSequence          int64                  `json:"cross_seq"`
	PositionSequence       int64                  `json:"position_seq"`
	CreatedAt              string                 `json:"created_at"`
	UpdateAt               string                 `json:"updated_at"`
	TakeProfitStopLossMode string                 `json:"tp_sl_mode"` // present in GetPositions API
	ExtensionField         map[string]interface{} `json:"ext_fields"` // present in SetTradingAndStop API
}

type Trade struct {
	ClosedSize     float64 `json:"closed_size"`
	CrossSequence  int64   `json:"cross_seq"`
	ExecutionFee   float64 `json:"exec_fee,string"`
	ExecutionID    string  `json:"exec_id"`
	ExecutionPrice float64 `json:"exec_price,string"`
	ExecutionQty   float64 `json:"exec_qty"`
	ExecutionType  string  `json:"exec_type"`
	ExecutionValue float64 `json:"exec_value,string"`
	FeeRate        float64 `json:"fee_rate"`
	LastLiquidilty string  `json:"last_liquidity_ind"`
	LeavesQty      float64 `json:"leaves_qty"`
	NthFill        int64   `json:"nth_fill"`
	OrderID        string  `json:"order_id"`
	OrderLinkedID  string  `json:"order_link_id"`
	OrderPrice     float64 `json:"order_price"`
	OrderQty       float64 `json:"order_qty"`
	OrderType      string  `json:"order_type"`
	OrderSide      string  `json:"side"`
	Symbol         string  `json:"symbol"`
	UserID         int64   `json:"user_id"`
	TradeTime      int64   `json:"trade_time_ms"`
}

type ClosedTrades struct {
	ID                   int64   `json:"id"`
	UserID               int64   `json:"user_id"`
	Symbol               string  `json:"symbol"`
	OrderID              string  `json:"order_id"`
	OrderSide            string  `json:"side"`
	Qty                  float64 `json:"qty"`
	OrderPrice           float64 `json:"order_price"`
	OrderType            string  `json:"order_type"`
	ExecutionType        string  `json:"exec_type"`
	ClosedSize           float64 `json:"closed_size"`
	CumulativeEntryValue float64 `json:"cum_entry_value"`
	AvgEntryPrice        float64 `json:"avg_entry_price"`
	CumulativeExitValue  float64 `json:"cum_exit_value"`
	AvgEntryValue        float64 `json:"avg_exit_price"`
	ClosedProfitLoss     float64 `json:"closed_pnl"`
	FillCount            int64   `json:"fill_count"`
	Leverage             int64   `json:"leverage"`
	CreatedAt            int64   `json:"created_at"`
}

type FundingFee struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Size          float64 `json:"qty"`
	FundingRate   float64 `json:"funding_rate"`
	ExecutionFee  float64 `json:"exec_fee"`
	ExecutionTime int64   `json:"exec_timestamp"`
}

type APIKeyData struct {
	APIKey     string   `json:"api_key"`
	Type       string   `json:"type"`
	UserID     int64    `json:"user_id"`
	InviterID  int64    `json:"inviter_id"`
	IPs        string   `json:"ips"`
	Note       string   `json:"note"`
	Permission []string `json:"permissions"`
	CreatedAt  string   `json:"created_at"`
	ExpiredAt  string   `json:"expired_at"`
	ReadOnly   bool     `json:"read_only"`
}

type LCPData struct {
	Date          string  `json:"date"`
	SelfRatio     float64 `json:"self_ratio"`
	PlatformRatio float64 `json:"platform_ratio"`
	Score         float64 `json:"score"`
}

type WalletData struct {
	Equity                float64 `json:"equity"` //equity = wallet_balance + unrealised_pnl
	AvailableBalance      float64 `json:"available_balance"`
	UserMargin            float64 `json:"used_margin"`
	OrderMargin           float64 `json:"order_margin"`
	PositionMargin        float64 `json:"position_margin"`
	PositionClosingFee    float64 `json:"occ_closing_fee"`
	PositionFundingFee    float64 `json:"occ_funding_fee"`
	WalletBalance         float64 `json:"wallet_balance"`
	RealisedPNL           float64 `json:"realised_pnl"`
	UnrealisedPNL         float64 `json:"unrealised_pnl"`
	CumulativeRealisedPNL float64 `json:"cum_realised_pnl"`
	GivenCash             float64 `json:"given_cash"`
	ServiceCash           float64 `json:"service_cash"`
}

type FundRecord struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"user_id"`
	Coin          string  `json:"coin"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount,string"`
	TxID          string  `json:"tx_id"`
	Address       string  `json:"address"`
	WalletBalance float64 `json:"wallet_balance,string"`
	ExecutionTime string  `json:"exec_time"`
	CrossSequence int64   `json:"cross_seq"`
}

type FundWithdrawalRecord struct {
	ID         int64   `json:"id"`
	UserID     int64   `json:"user_id"`
	Coin       string  `json:"coin"`
	Status     string  `json:"status"`
	Amount     float64 `json:"amount,string"`
	Fee        float64 `json:"fee"`
	Address    string  `json:"address"`
	TxID       string  `json:"tx_id"`
	SubmitedAt string  `json:"submited_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type AssetExchangeRecord struct {
	ID           int64   `json:"id"`
	FromCoin     string  `json:"from_coin"`
	FromAmount   float64 `json:"from_amount"`
	ToCoin       string  `json:"to_coin"`
	ToAmount     float64 `json:"to_amount"`
	ExchangeRate float64 `json:"exchange_rate"`
	FromFee      float64 `json:"from_fee"`
	CreatedAt    string  `json:"created_at"`
}
