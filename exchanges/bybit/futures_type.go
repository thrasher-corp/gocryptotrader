package bybit

import "time"

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
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
	Size   float64 `json:"size"`
	Side   string  `json:"side"`
}

// FuturesCandleStick holds kline data
type FuturesCandleStick struct {
	ID       int64   `json:"id"`
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

// FuturesCandleStickWithStringParam holds kline data
type FuturesCandleStickWithStringParam struct {
	ID       int64   `json:"id"`
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	OpenTime int64   `json:"open_time"`
	Open     float64 `json:"open,string"`
	High     float64 `json:"high,string"`
	Low      float64 `json:"low,string"`
	Close    float64 `json:"close,string"`
	Volume   float64 `json:"volume,string"`
	TurnOver float64 `json:"turnover,string"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol                 string  `json:"symbol"`
	BidPrice               float64 `json:"bid_price,string"`
	AskPrice               float64 `json:"ask_price,string"`
	LastPrice              float64 `json:"last_price,string"`
	LastTickDirection      string  `json:"last_tick_direction"`
	Price24hAgo            float64 `json:"prev_price_24h,string"`
	PricePcntChange24h     float64 `json:"price_24h_pcnt,string"`
	HighPrice24h           float64 `json:"high_price_24h,string"`
	LowPrice24h            float64 `json:"low_price_24h,string"`
	Price1hAgo             float64 `json:"prev_price_1h,string"`
	PricePcntChange1h      float64 `json:"price_1h_pcnt,string"`
	MarkPrice              float64 `json:"mark_price,string"`
	IndexPrice             float64 `json:"index_price,string"`
	OpenInterest           float64 `json:"open_interest"`
	OpenValue              float64 `json:"open_value,string"`
	TotalTurnover          float64 `json:"total_turnover,string"`
	Turnover24h            float64 `json:"turnover_24h,string"`
	TotalVolume            float64 `json:"total_volume"`
	Volume24h              float64 `json:"volume_24h"`
	FundingRate            float64 `json:"funding_rate,string"`
	PredictedFundingRate   float64 `json:"predicted_funding_rate,string"`
	NextFundingTime        string  `json:"next_funding_time"`
	CountdownHour          int64   `json:"countdown_hour"`
	DeliveryFeeRate        string  `json:"delivery_fee_rate"`        // type is string because it comes as empty string in API response sometime
	PredictedDeliveryPrice string  `json:"predicted_delivery_price"` // type is string because it comes as empty string in API response sometime
	DeliveryTime           string  `json:"delivery_time"`
}

// FuturesPublicTradesData stores recent public trades for futures
type FuturesPublicTradesData struct {
	Symbol         string    `json:"symbol"`
	Price          float64   `json:"price"`
	Qty            float64   `json:"qty"`
	Time           time.Time `json:"time"`
	Side           string    `json:"side"`
	TimeInMilliSec int64     `json:"trade_time_ms"`
}

// SymbolInfo stores symbol information for futures pair
type SymbolInfo struct {
	Name               string  `json:"name"`
	Alias              string  `json:"alias"`
	Status             string  `json:"status"`
	BaseCurrency       string  `json:"base_currency"`
	QuoteCurrency      string  `json:"quote_currency"`
	PriceScale         float64 `json:"price_scale"`
	TakerFee           string  `json:"taker_fee"`
	MakerFee           string  `json:"maker_fee"`
	FundingFeeInterval int64   `json:"funding_interval"`
	LeverageFilter     struct {
		MinLeverage  float64 `json:"min_leverage"`
		MaxLeverage  float64 `json:"max_leverage"`
		LeverageStep float64 `json:"leverage_step,string"`
	} `json:"leverage_filter"`
	PriceFilter struct {
		MinPrice float64 `json:"min_price,string"`
		MaxPrice float64 `json:"max_price,string"`
		TickSize float64 `json:"tick_size,string"`
	} `json:"price_filter"`
	LotSizeFilter struct {
		MinTradeQty float64 `json:"min_trading_qty"`
		MaxTradeQty float64 `json:"max_trading_qty"`
		QtyStep     float64 `json:"qty_step"`
	} `json:"lot_size_filter"`
}

// MarkPriceKlineData stores mark price kline data
type MarkPriceKlineData struct {
	ID       int64   `json:"id"`
	Symbol   string  `json:"symbol"`
	Interval string  `json:"period"`
	StartAt  int64   `json:"start_at"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
}

// IndexPriceKlineData stores index price kline data
type IndexPriceKlineData struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"period"`
	StartAt  int64   `json:"open_time"`
	Open     float64 `json:"open,string"`
	High     float64 `json:"high,string"`
	Low      float64 `json:"low,string"`
	Close    float64 `json:"close,string"`
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	OpenInterest float64 `json:"open_interest"`
	Symbol       string  `json:"symbol"`
	Time         int64   `json:"time"`
}

// BigDealData stores big deal data
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

// BaseFuturesOrder is base future order structure
type BaseFuturesOrder struct {
	UserID      int64   `json:"user_id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	OrderType   string  `json:"order_type"`
	Price       float64 `json:"price"`
	Qty         float64 `json:"qty"`
	TimeInForce string  `json:"time_in_force"`
}

// FuturesOrderData stores futures order data
type FuturesOrderData struct {
	BaseFuturesOrder
	OrderStatus     string    `json:"order_status"`
	OrderLinkID     string    `json:"order_link_id"`
	OrderID         string    `json:"order_id"`
	LeavesQty       float64   `json:"leaves_qty"`
	CumulativeQty   float64   `json:"cum_exec_qty"`
	CumulativeValue float64   `json:"cum_exec_value"`
	CumulativeFee   float64   `json:"cum_exec_fee"`
	RejectReason    string    `json:"reject_reason"`
	CreatedAt       time.Time `json:"create_at"`
}

// FuturesOrderCancelResp stores future order cancel response
type FuturesOrderCancelResp struct {
	FuturesOrderData
	LastExecutionTime  string  `json:"last_exec_time"`
	LastExecutionPrice float64 `json:"last_exec_price"`
	UpdateAt           string  `json:"updated_at"`
}

// FuturesOrderDataResp stores future order response
type FuturesOrderDataResp struct {
	FuturesOrderCancelResp
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

// FuturesActiveOrderData stores future active order data
type FuturesActiveOrderData struct {
	FuturesOrderData
	LeaveValue float64 `json:"leaves_value"`
}

// FuturesActiveOrderResp stores future active order response
type FuturesActiveOrderResp struct {
	FuturesActiveOrderData
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

// FuturesActiveOrder stores future active order
type FuturesActiveOrder struct {
	FuturesActiveOrderData
	PositionID int64  `json:"position_idx"`
	UpdatedAt  string `json:"updated_at"`
}

// FuturesRealtimeOrderData stores futures realtime order data
type FuturesRealtimeOrderData struct {
	BaseFuturesOrder
	OrderStatus         string  `json:"order_status"`
	OrderLinkID         string  `json:"order_link_id"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

// FuturesActiveRealtimeOrder stores future active realtime order
type FuturesActiveRealtimeOrder struct {
	FuturesRealtimeOrderData
	ExtensionField     map[string]interface{} `json:"ext_fields"`
	LastExecutionTime  string                 `json:"last_exec_time"`
	LastExecutionPrice float64                `json:"last_exec_price"`
	LeavesQty          float64                `json:"leaves_qty"`
	LeaveValue         float64                `json:"leaves_value,string"`
	CumulativeQty      float64                `json:"cum_exec_qty,string"`
	CumulativeValue    float64                `json:"cum_exec_value,string"`
	CumulativeFee      float64                `json:"cum_exec_fee,string"`
	RejectReason       string                 `json:"reject_reason"`
	CancelType         string                 `json:"cancel_type"`
	CreatedAt          time.Time              `json:"create_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	OrderID            string                 `json:"order_id"`
}

// CoinFuturesConditionalRealtimeOrder stores CMF future coinditional realtime order
type CoinFuturesConditionalRealtimeOrder struct {
	FuturesRealtimeOrderData
	ExtensionField  map[string]interface{} `json:"ext_fields"`
	LeavesQty       float64                `json:"leaves_qty"`
	LeaveValue      float64                `json:"leaves_value,string"`
	CumulativeQty   float64                `json:"cum_exec_qty,string"`
	CumulativeValue float64                `json:"cum_exec_value,string"`
	CumulativeFee   float64                `json:"cum_exec_fee,string"`
	RejectReason    string                 `json:"reject_reason"`
	CancelType      string                 `json:"cancel_type"`
	CreatedAt       string                 `json:"create_at"`
	UpdatedAt       string                 `json:"updated_at"`
	OrderID         string                 `json:"order_id"`
}

// FuturesConditionalRealtimeOrder stores future conditional realtime order
type FuturesConditionalRealtimeOrder struct {
	CoinFuturesConditionalRealtimeOrder
	PositionID int64 `json:"position_idx"`
}

// USDTFuturesConditionalRealtimeOrder stores USDT future conditional realtime order
type USDTFuturesConditionalRealtimeOrder struct {
	FuturesRealtimeOrderData
	StopOrderID    string  `json:"stop_order_id"`
	OrderStatus    string  `json:"order_status"`
	TriggerPrice   float64 `json:"trigger_price"`
	CreatedAt      string  `json:"created_time"`
	UpdatedAt      string  `json:"updated_time"`
	BasePrice      float64 `json:"base_price"`
	TriggerBy      string  `json:"trigger_by"`
	ReduceOnly     bool    `json:"reduce_only"`
	CloseOnTrigger bool    `json:"close_on_trigger"`
}

// FuturesConditionalOrderData stores futures conditional order data
type FuturesConditionalOrderData struct {
	BaseFuturesOrder
	TriggerBy           string  `json:"trigger_by"`
	BasePrice           float64 `json:"base_price"`
	StopOrderID         string  `json:"stop_order_id"`
	OrderLinkID         string  `json:"order_link_id"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
}

// FuturesConditionalOrderResp stores futures conditional order response
type FuturesConditionalOrderResp struct {
	FuturesConditionalOrderData
	Remark       string  `json:"remark"`
	RejectReason string  `json:"reject_reason"`
	StopPrice    float64 `json:"stop_px"`
	TakeProfit   float64 `json:"take_profit"`
	StopLoss     float64 `json:"stop_loss"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// USDTFuturesConditionalOrderResp stores USDT futures conditional order response
type USDTFuturesConditionalOrderResp struct {
	FuturesConditionalOrderData
	OrderStatus    string  `json:"order_status"`
	TriggerPrice   float64 `json:"trigger_price"`
	ReduceOnly     bool    `json:"reduce_only"`
	CloseOnTrigger bool    `json:"close_on_trigger"`
	CreatedAt      string  `json:"created_time"`
	UpdatedAt      string  `json:"updated_time"`
}

// CoinFuturesConditionalOrders stores CMF future conditional order
type CoinFuturesConditionalOrders struct {
	FuturesConditionalOrderData
	StopOrderStatus string  `json:"stop_order_status"`
	StopOrderType   string  `json:"stop_order_type"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	StopPrice       float64 `json:"stop_px"`
	StopOrderID     string  `json:"stop_order_id"`
	TakeProfit      float64 `json:"take_profit"`
	StopLoss        float64 `json:"stop_loss"`
}

// FuturesConditionalOrders stores future conditional order
type FuturesConditionalOrders struct {
	CoinFuturesConditionalOrders
	PositionID int64 `json:"position_idx"`
}

// USDTFuturesConditionalOrders stores USDT futures conditional order
type USDTFuturesConditionalOrders struct {
	FuturesConditionalOrderData
	OrderStatus  string  `json:"order_status"`
	TriggerPrice float64 `json:"trigger_price"`
	CreatedAt    string  `json:"created_time"`
	UpdatedAt    string  `json:"updated_time"`
	TakeProfit   float64 `json:"take_profit"`
	StopLoss     float64 `json:"stop_loss"`
}

// FuturesCancelOrderData stores future cancel order data
type FuturesCancelOrderData struct {
	CancelOrderID string `json:"clOrdID"`
	BaseFuturesOrder
	CreateType  string  `json:"create_type"`
	CancelType  string  `json:"cancel_type"`
	OrderStatus string  `json:"order_status"`
	LeavesQty   float64 `json:"leaves_qty"`
	LeavesValue float64 `json:"leaves_value"`
	CreatedAt   string  `json:"create_at"`
	UpdateAt    string  `json:"updated_at"`
	CrossStatus string  `json:"cross_status"`
	CrossSeq    int64   `json:"cross_seq"`
}

// FuturesCancelOrderResp stores future cancel order response
type FuturesCancelOrderResp struct {
	FuturesCancelOrderData
	StopOrderType     string  `json:"stop_order_type"`
	TriggerBy         string  `json:"trigger_by"`
	BasePrice         float64 `json:"base_price,string"`
	ExpectedDirection string  `json:"expected_direction"`
}

// RiskInfo stores risk information
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

// RiskInfoWithStringParam stores risk information where string params
type RiskInfoWithStringParam struct {
	ID             int64    `json:"id"`
	Symbol         string   `json:"symbol"`
	Limit          int64    `json:"limit"`
	MaintainMargin float64  `json:"maintain_margin,string"`
	StartingMargin float64  `json:"starting_margin,string"`
	Section        []string `json:"section"`
	IsLowestRisk   int64    `json:"is_lowest_risk"`
	CreatedAt      string   `json:"create_at"`
	UpdateAt       string   `json:"updated_at"`
	MaxLeverage    float64  `json:"max_leverage,string"`
}

// FundingInfo stores funding information
type FundingInfo struct {
	Symbol               string  `json:"symbol"`
	FundingRate          float64 `json:"funding_rate,string"`
	FundingRateTimestamp int64   `json:"funding_rate_timestamp"`
}

// USDTFundingInfo stores USDT funding information
type USDTFundingInfo struct {
	Symbol               string  `json:"symbol"`
	FundingRate          float64 `json:"funding_rate"`
	FundingRateTimestamp string  `json:"funding_rate_timestamp"`
}

// AnnouncementInfo stores announcement information
type AnnouncementInfo struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

// Position stores position
type Position struct {
	UserID                 int64   `json:"user_id"`
	Symbol                 string  `json:"symbol"`
	Side                   string  `json:"side"`
	Size                   float64 `json:"size"`
	PositionValue          float64 `json:"position_value"`
	EntryPrice             float64 `json:"entry_price"`
	LiquidationPrice       float64 `json:"liq_price"`
	BankruptcyPrice        float64 `json:"bust_price"`
	Leverage               float64 `json:"leverage"`
	PositionMargin         float64 `json:"position_margin"`
	OccupiedClosingFee     float64 `json:"occ_closing_fee"`
	RealisedPNL            float64 `json:"realised_pnl"`
	AccumulatedRealisedPNL float64 `json:"cum_realised_pnl"`
}

// PositionWithStringParam stores position with string params
type PositionWithStringParam struct {
	UserID                 int64   `json:"user_id"`
	Symbol                 string  `json:"symbol"`
	Side                   string  `json:"side"`
	Size                   float64 `json:"size"`
	PositionValue          float64 `json:"position_value,string"`
	EntryPrice             float64 `json:"entry_price,string"`
	LiquidationPrice       float64 `json:"liq_price,string"`
	BankruptcyPrice        float64 `json:"bust_price,string"`
	Leverage               float64 `json:"leverage,string"`
	PositionMargin         float64 `json:"position_margin,string"`
	OccupiedClosingFee     float64 `json:"occ_closing_fee,string"`
	RealisedPNL            float64 `json:"realised_pnl,string"`
	AccumulatedRealisedPNL float64 `json:"cum_realised_pnl,string"`
}

// PositionData stores position data
type PositionData struct {
	Position
	IsIsolated          bool    `json:"is_isolated"`
	AutoAddMargin       int64   `json:"auto_add_margin"`
	UnrealisedPNL       float64 `json:"unrealised_pnl"`
	DeleverageIndicator int64   `json:"deleverage_indicator"`
	RiskID              int64   `json:"risk_id"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	TrailingStop        float64 `json:"trailing_stop"`
}

// PositionDataWithStringParam stores position data with string params
type PositionDataWithStringParam struct {
	PositionWithStringParam
	IsIsolated          bool    `json:"is_isolated"`
	AutoAddMargin       int64   `json:"auto_add_margin"`
	UnrealisedPNL       float64 `json:"unrealised_pnl"`
	DeleverageIndicator int64   `json:"deleverage_indicator"`
	RiskID              int64   `json:"risk_id"`
	TakeProfit          float64 `json:"take_profit,string"`
	StopLoss            float64 `json:"stop_loss,string"`
	TrailingStop        float64 `json:"trailing_stop,string"`
}

// PositionResp stores position response
type PositionResp struct {
	PositionDataWithStringParam
	PositionID             int64   `json:"position_idx"`
	Mode                   int64   `json:"mode"`
	ID                     int64   `json:"id"`
	EffectiveLeverage      float64 `json:"effective_leverage,string"`
	OccupiedFundingFee     float64 `json:"occ_funding_fee,string"`
	PositionStatus         string  `json:"position_status"`
	CalculatedData         string  `json:"oc_calc_data"`
	OrderMargin            float64 `json:"order_margin,string"`
	WalletBalance          float64 `json:"wallet_balance,string"`
	CrossSequence          int64   `json:"cross_seq"`
	PositionSequence       int64   `json:"position_seq"`
	TakeProfitStopLossMode string  `json:"tp_sl_mode"`
	CreatedAt              string  `json:"created_at"`
	UpdateAt               string  `json:"updated_at"`
}

// SetTradingAndStopResp stores set trading and stop response
type SetTradingAndStopResp struct {
	PositionData
	ID                  int64                  `json:"id"`
	RiskID              int64                  `json:"risk_id"`
	AutoAddMargin       int64                  `json:"auto_add_margin"`
	OccupiedFundingFee  float64                `json:"occ_funding_fee,string"`
	TakeProfit          float64                `json:"take_profit,string"`
	StopLoss            float64                `json:"stop_loss,string"`
	PositionStatus      string                 `json:"position_status"`
	DeleverageIndicator int64                  `json:"deleverage_indicator"`
	CalculatedData      string                 `json:"oc_calc_data"`
	OrderMargin         float64                `json:"order_margin,string"`
	WalletBalance       float64                `json:"wallet_balance,string"`
	CrossSequence       int64                  `json:"cross_seq"`
	PositionSequence    int64                  `json:"position_seq"`
	CreatedAt           string                 `json:"created_at"`
	UpdateAt            string                 `json:"updated_at"`
	ExtensionField      map[string]interface{} `json:"ext_fields"`
}

// USDTPositionResp stores USDT position response
type USDTPositionResp struct {
	PositionData
	FreeQty                float64 `json:"free_qty"`
	TakeProfitStopLossMode string  `json:"tp_sl_mode"`
}

// UpdateMarginResp stores update margin response
type UpdateMarginResp struct {
	Position
	FreeQty float64 `json:"free_qty"`
}

// TradeData stores trade data
type TradeData struct {
	OrderID        string  `json:"order_id"`
	OrderLinkedID  string  `json:"order_link_id"`
	OrderSide      string  `json:"side"`
	Symbol         string  `json:"symbol"`
	ExecutionID    string  `json:"exec_id"`
	OrderPrice     float64 `json:"order_price"`
	OrderQty       float64 `json:"order_qty"`
	OrderType      string  `json:"order_type"`
	FeeRate        float64 `json:"fee_rate"`
	ExecutionFee   float64 `json:"exec_fee,string"`
	ExecutionPrice float64 `json:"exec_price,string"`
	ExecutionQty   float64 `json:"exec_qty"`
	ExecutionType  string  `json:"exec_type"`
	ExecutionValue float64 `json:"exec_value,string"`
	LeavesQty      float64 `json:"leaves_qty"`
	ClosedSize     float64 `json:"closed_size"`
	LastLiquidity  string  `json:"last_liquidity_ind"`
	TradeTimeMs    int64   `json:"trade_time_ms"`
}

// TradeResp stores trade response
type TradeResp struct {
	TradeData
	CrossSequence int64 `json:"cross_seq"`
	NthFill       int64 `json:"nth_fill"`
	UserID        int64 `json:"user_id"`
}

// ClosedTrades stores closed trades
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
	Leverage             float64 `json:"leverage"`
	CreatedAt            int64   `json:"created_at"`
}

// FundingFee stores funding fee
type FundingFee struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Size          float64 `json:"size"`
	FundingRate   float64 `json:"funding_rate"`
	ExecutionFee  float64 `json:"exec_fee"`
	ExecutionTime int64   `json:"exec_timestamp"`
}

// APIKeyData stores API key data
type APIKeyData struct {
	APIKey           string   `json:"api_key"`
	Type             string   `json:"type"`
	UserID           int64    `json:"user_id"`
	InviterID        int64    `json:"inviter_id"`
	IPs              []string `json:"ips"`
	Note             string   `json:"note"`
	Permission       []string `json:"permissions"`
	CreatedAt        string   `json:"created_at"`
	ExpiredAt        string   `json:"expired_at"`
	ReadOnly         bool     `json:"read_only"`
	VIPLevel         string   `json:"vip_level"`
	MarketMakerLevel string   `json:"mkt_maker_level"`
}

// LCPData stores LiquidityContributionPointsData data
type LCPData struct {
	Date          string  `json:"date"`
	SelfRatio     float64 `json:"self_ratio"`
	PlatformRatio float64 `json:"platform_ratio"`
	Score         float64 `json:"score"`
}

// WalletData stores wallet data
type WalletData struct {
	Equity                float64 `json:"equity"` // equity = wallet_balance + unrealised_pnl
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

// FundRecord stores funding records
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

// FundWithdrawalRecord stores funding withdrawal records
type FundWithdrawalRecord struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Coin        string    `json:"coin"`
	Status      string    `json:"status"`
	Amount      float64   `json:"amount,string"`
	Fee         float64   `json:"fee"`
	Address     string    `json:"address"`
	TxID        string    `json:"tx_id"`
	SubmittedAt time.Time `json:"submited_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssetExchangeRecord stores asset exchange records
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

// USDCOrderbookData stores orderbook data for USDCMarginedFutures
type USDCOrderbookData struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
	Side  string  `json:"side"`
}

// USDCContract stores contract data
type USDCContract struct {
	Symbol        string  `json:"symbol"`
	Status        string  `json:"status"`
	BaseCoin      string  `json:"baseCoin"`
	QuoteCoin     string  `json:"quoteCoin"`
	TakerFeeRate  float64 `json:"takerFeeRate,string"`
	MakerFeeRate  float64 `json:"makerFeeRate,string"`
	MinLeverage   float64 `json:"minLeverage,string"`
	MaxLeverage   float64 `json:"maxLeverage,string"`
	LeverageStep  float64 `json:"leverageStep,string"`
	MinPrice      float64 `json:"minPrice,string"`
	MaxPrice      float64 `json:"maxPrice,string"`
	TickSize      float64 `json:"tickSize,string"`
	MaxTradingQty float64 `json:"maxTradingQty,string"`
	MinTradingQty float64 `json:"minTradingQty,string"`
	QtyStep       float64 `json:"qtyStep,string"`
	DeliveryTime  float64 `json:"deliveryTime,string"`
}

// USDCSymbol stores symbol data
type USDCSymbol struct {
	Symbol               string  `json:"symbol"`
	NextFundingTime      string  `json:"nextFundingTime"`
	Bid                  float64 `json:"bid,string"`
	BidSize              float64 `json:"bidSize,string"`
	Ask                  float64 `json:"ask,string"`
	AskSize              float64 `json:"askSize,string"`
	LastPrice            float64 `json:"lastPrice,string"`
	OpenInterest         float64 `json:"openInterest,string"`
	IndexPrice           float64 `json:"indexPrice,string"`
	MarkPrice            float64 `json:"markPrice,string"`
	Change24h            float64 `json:"change24h,string"`
	High24h              float64 `json:"high24h,string"`
	Low24h               float64 `json:"low24h,string"`
	Volume24h            float64 `json:"volume24h,string"`
	Turnover24h          float64 `json:"turnover24h,string"`
	TotalVolume          float64 `json:"totalVolume,string"`
	TotalTurnover        float64 `json:"totalTurnover,string"`
	FundingRate          float64 `json:"fundingRate,string"`
	PredictedFundingRate float64 `json:"predictedFundingRate,string"`
	CountdownHour        float64 `json:"countdownHour,string"`
	UnderlyingPrice      string  `json:"underlyingPrice"`
}

// USDCKlineBase stores Kline Base
type USDCKlineBase struct {
	Symbol   string  `json:"symbol"`
	Period   string  `json:"period"`
	OpenTime string  `json:"openTime"`
	Open     float64 `json:"open,string"`
	High     float64 `json:"high,string"`
	Low      float64 `json:"low,string"`
	Close    float64 `json:"close,string"`
}

// USDCKline stores kline data
type USDCKline struct {
	USDCKlineBase
	Volume   float64 `json:"volume,string"`
	Turnover float64 `json:"turnover,string"`
}

// USDCOpenInterest stores open interest data
type USDCOpenInterest struct {
	Symbol       string  `json:"symbol"`
	Timestamp    string  `json:"timestamp"`
	OpenInterest float64 `json:"openInterest,string"`
}

// USDCLargeOrder stores large order data
type USDCLargeOrder struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// USDCAccountRatio stores long-short ratio data
type USDCAccountRatio struct {
	Symbol    string  `json:"symbol"`
	BuyRatio  float64 `json:"buyRatio"`
	SellRatio float64 `json:"sellRatio"`
	Timestamp string  `json:"timestamp"`
}

// USDCTrade stores trade data
type USDCTrade struct {
	ID         string  `json:"id"`
	Symbol     string  `json:"symbol"`
	OrderPrice float64 `json:"orderPrice,string"`
	OrderQty   float64 `json:"orderQty,string"`
	Side       string  `json:"side"`
	Timestamp  string  `json:"time"`
}

// USDCCreateOrderResp stores create order response
type USDCCreateOrderResp struct {
	ID          string  `json:"orderId"`
	OrderLinkID string  `json:"orderLinkId"`
	Symbol      string  `json:"symbol"`
	OrderPrice  float64 `json:"orderPrice,string"`
	OrderQty    float64 `json:"orderQty,string"`
	OrderType   string  `json:"orderType"`
	Side        string  `json:"side"`
}

// USDCOrder store order data
type USDCOrder struct {
	ID              string  `json:"orderId"`
	OrderLinkID     string  `json:"orderLinkId"`
	Symbol          string  `json:"symbol"`
	OrderType       string  `json:"orderType"`
	Side            string  `json:"side"`
	Qty             float64 `json:"qty,string"`
	Price           float64 `json:"price,string"`
	TimeInForce     string  `json:"timeInForce"`
	TotalOrderValue float64 `json:"cumExecValue,string"`
	TotalFilledQty  float64 `json:"cumExecQty,string"`
	TotalFee        float64 `json:"cumExecFee,string"`
	InitialMargin   string  `json:"orderIM"`
	OrderStatus     string  `json:"orderStatus"`
	TakeProfit      float64 `json:"takeProfit,string"`
	StopLoss        float64 `json:"stopLoss,string"`
	TPTriggerBy     string  `json:"tpTriggerBy"`
	SLTriggerBy     string  `json:"slTriggerBy"`
	LastExecPrice   float64 `json:"lastExecPrice"`
	BasePrice       string  `json:"basePrice"`
	TriggerPrice    float64 `json:"triggerPrice,string"`
	TriggerBy       string  `json:"triggerBy"`
	ReduceOnly      bool    `json:"reduceOnly"`
	StopOrderType   string  `json:"stopOrderType"`
	CloseOnTrigger  string  `json:"closeOnTrigger"`
	CreatedAt       string  `json:"createdAt"`
}

// USDCOrderHistory stores order history
type USDCOrderHistory struct {
	USDCOrder
	LeavesValue float64 `json:"leavesValue,string"` //Est. order value of unfilled orders
	CashFlow    string  `json:"cashFlow"`
	RealisedPnl float64 `json:"realisedPnl,string"`
	UpdatedAt   string  `json:"updatedAt"`
}

// USDCTradeHistory stores trade history
type USDCTradeHistory struct {
	ID               string  `json:"orderId"`
	OrderLinkID      string  `json:"orderLinkId"`
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	TradeID          string  `json:"tradeId"`
	ExecPrice        float64 `json:"execPrice,string"`
	ExecQty          float64 `json:"execQty,string"`
	ExecFee          float64 `json:"execFee,string"`
	FeeRate          float64 `json:"feeRate,string"`
	ExecType         string  `json:"execType"`
	ExecValue        float64 `json:"execValue,string"`
	TradeTime        string  `json:"tradeTime"`
	LastLiquidityInd string  `json:"lastLiquidityInd"`
}

// USDCTxLog stores transaction log data
type USDCTxLog struct {
	TxTime        string  `json:"transactionTime"`
	Symbol        string  `json:"symbol"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	Quantity      float64 `json:"qty,string"`
	Size          float64 `json:"size,string"`
	TradePrice    float64 `json:"tradePrice,string"`
	Funding       float64 `json:"funding,string"`
	Fee           float64 `json:"fee,string"`
	CashFlow      string  `json:"cashFlow"`
	Change        float64 `json:"change,string"`
	WalletBalance float64 `json:"walletBalance,string"`
	FeeRate       float64 `json:"feeRate,string"`
	TradeID       string  `json:"tradeId"`
	OrderID       string  `json:"orderId"`
	OrderLinkID   string  `json:"orderLinkId"`
	Info          string  `json:"info"`
}

type USDCWalletBalance struct {
	Equity           float64 `json:"equity,string"`
	WalletBalance    float64 `json:"walletBalance,string"`
	AvailableBalance float64 `json:"availableBalance,string"`
	AccountIM        float64 `json:"accountIM,string"`
	AccountMM        float64 `json:"accountMM,string"`
	TotalRPL         float64 `json:"totalRPL,string"`
	TotalSessionUPL  float64 `json:"totalSessionUPL,string"`
	TotalSessionRPL  float64 `json:"totalSessionRPL,string"`
}

type USDCAssetInfo struct {
	BaseCoin   string  `json:"baseCoin"`
	TotalDelta float64 `json:"totalDelta,string"`
	TotalGamma float64 `json:"totalGamma,string"`
	TotalVega  float64 `json:"totalVega,string"`
	TotalTheta float64 `json:"totalTheta,string"`
	TotalRPL   float64 `json:"totalRPL,string"`
	SessionUPL float64 `json:"sessionUPL,string"`
	SessionRPL float64 `json:"sessionRPL,string"`
	IM         float64 `json:"im,string"`
	MM         float64 `json:"mm,string"`
}
