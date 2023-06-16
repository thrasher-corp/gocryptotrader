package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

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
	Symbol string                  `json:"symbol"`
	Price  convert.StringToFloat64 `json:"price"`
	Size   float64                 `json:"size"`
	Side   string                  `json:"side"`
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
	ID       int64                   `json:"id"`
	Symbol   string                  `json:"symbol"`
	Interval string                  `json:"interval"`
	OpenTime int64                   `json:"open_time"`
	Open     convert.StringToFloat64 `json:"open"`
	High     convert.StringToFloat64 `json:"high"`
	Low      convert.StringToFloat64 `json:"low"`
	Close    convert.StringToFloat64 `json:"close"`
	Volume   convert.StringToFloat64 `json:"volume"`
	TurnOver convert.StringToFloat64 `json:"turnover"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol                 string                  `json:"symbol"`
	BidPrice               convert.StringToFloat64 `json:"bid_price"`
	AskPrice               convert.StringToFloat64 `json:"ask_price"`
	LastPrice              convert.StringToFloat64 `json:"last_price"`
	LastTickDirection      string                  `json:"last_tick_direction"`
	Price24hAgo            convert.StringToFloat64 `json:"prev_price_24h"`
	PricePcntChange24h     convert.StringToFloat64 `json:"price_24h_pcnt"`
	HighPrice24h           convert.StringToFloat64 `json:"high_price_24h"`
	LowPrice24h            convert.StringToFloat64 `json:"low_price_24h"`
	Price1hAgo             convert.StringToFloat64 `json:"prev_price_1h"`
	PricePcntChange1h      convert.StringToFloat64 `json:"price_1h_pcnt"`
	MarkPrice              convert.StringToFloat64 `json:"mark_price"`
	IndexPrice             convert.StringToFloat64 `json:"index_price"`
	OpenInterest           float64                 `json:"open_interest"`
	OpenValue              convert.StringToFloat64 `json:"open_value"`
	TotalTurnover          convert.StringToFloat64 `json:"total_turnover"`
	Turnover24h            convert.StringToFloat64 `json:"turnover_24h"`
	TotalVolume            float64                 `json:"total_volume"`
	Volume24h              float64                 `json:"volume_24h"`
	FundingRate            convert.StringToFloat64 `json:"funding_rate"`
	PredictedFundingRate   convert.StringToFloat64 `json:"predicted_funding_rate"`
	NextFundingTime        string                  `json:"next_funding_time"`
	CountdownHour          int64                   `json:"countdown_hour"`
	DeliveryFeeRate        convert.StringToFloat64 `json:"delivery_fee_rate"`
	PredictedDeliveryPrice convert.StringToFloat64 `json:"predicted_delivery_price"`
	DeliveryTime           string                  `json:"delivery_time"`
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
		MinLeverage  float64                 `json:"min_leverage"`
		MaxLeverage  float64                 `json:"max_leverage"`
		LeverageStep convert.StringToFloat64 `json:"leverage_step"`
	} `json:"leverage_filter"`
	PriceFilter struct {
		MinPrice convert.StringToFloat64 `json:"min_price"`
		MaxPrice convert.StringToFloat64 `json:"max_price"`
		TickSize convert.StringToFloat64 `json:"tick_size"`
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
	Symbol   string                  `json:"symbol"`
	Interval string                  `json:"period"`
	StartAt  int64                   `json:"open_time"`
	Open     convert.StringToFloat64 `json:"open"`
	High     convert.StringToFloat64 `json:"high"`
	Low      convert.StringToFloat64 `json:"low"`
	Close    convert.StringToFloat64 `json:"close"`
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	OpenInterest float64 `json:"open_interest"`
	Symbol       string  `json:"symbol"`
	Time         int64   `json:"time"`
}

// BigDealData stores big deal data
type BigDealData struct {
	ID     int64   `json:"id"`
	Symbol string  `json:"symbol"`
	Side   string  `json:"side"`
	Time   int64   `json:"timestamp"`
	Value  float64 `json:"value"`
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
	ExtensionField     map[string]interface{}  `json:"ext_fields"`
	LastExecutionTime  string                  `json:"last_exec_time"`
	LastExecutionPrice float64                 `json:"last_exec_price"`
	LeavesQty          float64                 `json:"leaves_qty"`
	LeaveValue         convert.StringToFloat64 `json:"leaves_value"`
	CumulativeQty      convert.StringToFloat64 `json:"cum_exec_qty"`
	CumulativeValue    convert.StringToFloat64 `json:"cum_exec_value"`
	CumulativeFee      convert.StringToFloat64 `json:"cum_exec_fee"`
	RejectReason       string                  `json:"reject_reason"`
	CancelType         string                  `json:"cancel_type"`
	CreatedAt          time.Time               `json:"create_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
	OrderID            string                  `json:"order_id"`
}

// CoinFuturesConditionalRealtimeOrder stores CMF future coinditional realtime order
type CoinFuturesConditionalRealtimeOrder struct {
	FuturesRealtimeOrderData
	ExtensionField  map[string]interface{}  `json:"ext_fields"`
	LeavesQty       float64                 `json:"leaves_qty"`
	LeaveValue      convert.StringToFloat64 `json:"leaves_value"`
	CumulativeQty   convert.StringToFloat64 `json:"cum_exec_qty"`
	CumulativeValue convert.StringToFloat64 `json:"cum_exec_value"`
	CumulativeFee   convert.StringToFloat64 `json:"cum_exec_fee"`
	RejectReason    string                  `json:"reject_reason"`
	CancelType      string                  `json:"cancel_type"`
	CreatedAt       string                  `json:"create_at"`
	UpdatedAt       string                  `json:"updated_at"`
	OrderID         string                  `json:"order_id"`
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
	StopOrderType     string                  `json:"stop_order_type"`
	TriggerBy         string                  `json:"trigger_by"`
	BasePrice         convert.StringToFloat64 `json:"base_price"`
	ExpectedDirection string                  `json:"expected_direction"`
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
	ID             int64                   `json:"id"`
	Symbol         string                  `json:"symbol"`
	Limit          int64                   `json:"limit"`
	MaintainMargin convert.StringToFloat64 `json:"maintain_margin"`
	StartingMargin convert.StringToFloat64 `json:"starting_margin"`
	Section        []string                `json:"section"`
	IsLowestRisk   int64                   `json:"is_lowest_risk"`
	CreatedAt      string                  `json:"create_at"`
	UpdateAt       string                  `json:"updated_at"`
	MaxLeverage    convert.StringToFloat64 `json:"max_leverage"`
}

// FundingInfo stores funding information
type FundingInfo struct {
	Symbol               string                  `json:"symbol"`
	FundingRate          convert.StringToFloat64 `json:"funding_rate"`
	FundingRateTimestamp int64                   `json:"funding_rate_timestamp"`
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
	UserID                 int64                   `json:"user_id"`
	Symbol                 string                  `json:"symbol"`
	Side                   string                  `json:"side"`
	Size                   float64                 `json:"size"`
	PositionValue          convert.StringToFloat64 `json:"position_value"`
	EntryPrice             convert.StringToFloat64 `json:"entry_price"`
	LiquidationPrice       convert.StringToFloat64 `json:"liq_price"`
	BankruptcyPrice        convert.StringToFloat64 `json:"bust_price"`
	Leverage               convert.StringToFloat64 `json:"leverage"`
	PositionMargin         convert.StringToFloat64 `json:"position_margin"`
	OccupiedClosingFee     convert.StringToFloat64 `json:"occ_closing_fee"`
	RealisedPNL            convert.StringToFloat64 `json:"realised_pnl"`
	AccumulatedRealisedPNL convert.StringToFloat64 `json:"cum_realised_pnl"`
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
	IsIsolated          bool                    `json:"is_isolated"`
	AutoAddMargin       int64                   `json:"auto_add_margin"`
	UnrealisedPNL       float64                 `json:"unrealised_pnl"`
	DeleverageIndicator int64                   `json:"deleverage_indicator"`
	RiskID              int64                   `json:"risk_id"`
	TakeProfit          convert.StringToFloat64 `json:"take_profit"`
	StopLoss            convert.StringToFloat64 `json:"stop_loss"`
	TrailingStop        convert.StringToFloat64 `json:"trailing_stop"`
}

// PositionResp stores position response
type PositionResp struct {
	PositionDataWithStringParam
	PositionID             int64                   `json:"position_idx"`
	Mode                   int64                   `json:"mode"`
	ID                     int64                   `json:"id"`
	EffectiveLeverage      convert.StringToFloat64 `json:"effective_leverage"`
	OccupiedFundingFee     convert.StringToFloat64 `json:"occ_funding_fee"`
	PositionStatus         string                  `json:"position_status"`
	CalculatedData         string                  `json:"oc_calc_data"`
	OrderMargin            convert.StringToFloat64 `json:"order_margin"`
	WalletBalance          convert.StringToFloat64 `json:"wallet_balance"`
	CrossSequence          int64                   `json:"cross_seq"`
	PositionSequence       int64                   `json:"position_seq"`
	TakeProfitStopLossMode string                  `json:"tp_sl_mode"`
	CreatedAt              string                  `json:"created_at"`
	UpdateAt               string                  `json:"updated_at"`
}

// SetTradingAndStopResp stores set trading and stop response
type SetTradingAndStopResp struct {
	PositionData
	ID                  int64                   `json:"id"`
	RiskID              int64                   `json:"risk_id"`
	AutoAddMargin       int64                   `json:"auto_add_margin"`
	OccupiedFundingFee  convert.StringToFloat64 `json:"occ_funding_fee"`
	TakeProfit          convert.StringToFloat64 `json:"take_profit"`
	StopLoss            convert.StringToFloat64 `json:"stop_loss"`
	PositionStatus      string                  `json:"position_status"`
	DeleverageIndicator int64                   `json:"deleverage_indicator"`
	CalculatedData      string                  `json:"oc_calc_data"`
	OrderMargin         convert.StringToFloat64 `json:"order_margin"`
	WalletBalance       convert.StringToFloat64 `json:"wallet_balance"`
	CrossSequence       int64                   `json:"cross_seq"`
	PositionSequence    int64                   `json:"position_seq"`
	CreatedAt           string                  `json:"created_at"`
	UpdateAt            string                  `json:"updated_at"`
	ExtensionField      map[string]interface{}  `json:"ext_fields"`
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
	OrderID        string                  `json:"order_id"`
	OrderLinkedID  string                  `json:"order_link_id"`
	OrderSide      string                  `json:"side"`
	Symbol         string                  `json:"symbol"`
	ExecutionID    string                  `json:"exec_id"`
	OrderPrice     float64                 `json:"order_price"`
	OrderQty       float64                 `json:"order_qty"`
	OrderType      string                  `json:"order_type"`
	FeeRate        float64                 `json:"fee_rate"`
	ExecutionFee   convert.StringToFloat64 `json:"exec_fee"`
	ExecutionPrice convert.StringToFloat64 `json:"exec_price"`
	ExecutionQty   float64                 `json:"exec_qty"`
	ExecutionType  string                  `json:"exec_type"`
	ExecutionValue convert.StringToFloat64 `json:"exec_value"`
	LeavesQty      float64                 `json:"leaves_qty"`
	ClosedSize     float64                 `json:"closed_size"`
	LastLiquidity  string                  `json:"last_liquidity_ind"`
	TradeTimeMs    int64                   `json:"trade_time_ms"`
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
	ID                   int64        `json:"id"`
	UserID               int64        `json:"user_id"`
	Symbol               string       `json:"symbol"`
	OrderID              string       `json:"order_id"`
	OrderSide            string       `json:"side"`
	Qty                  float64      `json:"qty"`
	OrderPrice           float64      `json:"order_price"`
	OrderType            string       `json:"order_type"`
	ExecutionType        string       `json:"exec_type"`
	ClosedSize           float64      `json:"closed_size"`
	CumulativeEntryValue float64      `json:"cum_entry_value"`
	AvgEntryPrice        float64      `json:"avg_entry_price"`
	CumulativeExitValue  float64      `json:"cum_exit_value"`
	AvgEntryValue        float64      `json:"avg_exit_price"`
	ClosedProfitLoss     float64      `json:"closed_pnl"`
	FillCount            int64        `json:"fill_count"`
	Leverage             float64      `json:"leverage"`
	CreatedAt            bybitTimeSec `json:"created_at"`
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
	ID            int64                   `json:"id"`
	UserID        int64                   `json:"user_id"`
	Coin          string                  `json:"coin"`
	Type          string                  `json:"type"`
	Amount        convert.StringToFloat64 `json:"amount"`
	TxID          string                  `json:"tx_id"`
	Address       string                  `json:"address"`
	WalletBalance convert.StringToFloat64 `json:"wallet_balance"`
	ExecutionTime string                  `json:"exec_time"`
	CrossSequence int64                   `json:"cross_seq"`
}

// FundWithdrawalRecord stores funding withdrawal records
type FundWithdrawalRecord struct {
	ID          int64                   `json:"id"`
	UserID      int64                   `json:"user_id"`
	Coin        string                  `json:"coin"`
	Status      string                  `json:"status"`
	Amount      convert.StringToFloat64 `json:"amount"`
	Fee         float64                 `json:"fee"`
	Address     string                  `json:"address"`
	TxID        string                  `json:"tx_id"`
	SubmittedAt time.Time               `json:"submited_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
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
	Price convert.StringToFloat64 `json:"price"`
	Size  convert.StringToFloat64 `json:"size"`
	Side  string                  `json:"side"`
}

// USDCContract stores contract data
type USDCContract struct {
	Symbol        string                  `json:"symbol"`
	Status        string                  `json:"status"`
	BaseCoin      string                  `json:"baseCoin"`
	QuoteCoin     string                  `json:"quoteCoin"`
	TakerFeeRate  convert.StringToFloat64 `json:"takerFeeRate"`
	MakerFeeRate  convert.StringToFloat64 `json:"makerFeeRate"`
	MinLeverage   convert.StringToFloat64 `json:"minLeverage"`
	MaxLeverage   convert.StringToFloat64 `json:"maxLeverage"`
	LeverageStep  convert.StringToFloat64 `json:"leverageStep"`
	MinPrice      convert.StringToFloat64 `json:"minPrice"`
	MaxPrice      convert.StringToFloat64 `json:"maxPrice"`
	TickSize      convert.StringToFloat64 `json:"tickSize"`
	MaxTradingQty convert.StringToFloat64 `json:"maxTradingQty"`
	MinTradingQty convert.StringToFloat64 `json:"minTradingQty"`
	QtyStep       convert.StringToFloat64 `json:"qtyStep"`
	DeliveryTime  bybitTimeMilliSecStr    `json:"deliveryTime"`
}

// USDCSymbol stores symbol data
type USDCSymbol struct {
	Symbol               string                  `json:"symbol"`
	NextFundingTime      string                  `json:"nextFundingTime"`
	Bid                  convert.StringToFloat64 `json:"bid"`
	BidSize              convert.StringToFloat64 `json:"bidSize"`
	Ask                  convert.StringToFloat64 `json:"ask"`
	AskSize              convert.StringToFloat64 `json:"askSize"`
	LastPrice            convert.StringToFloat64 `json:"lastPrice"`
	OpenInterest         convert.StringToFloat64 `json:"openInterest"`
	IndexPrice           convert.StringToFloat64 `json:"indexPrice"`
	MarkPrice            convert.StringToFloat64 `json:"markPrice"`
	Change24h            convert.StringToFloat64 `json:"change24h"`
	High24h              convert.StringToFloat64 `json:"high24h"`
	Low24h               convert.StringToFloat64 `json:"low24h"`
	Volume24h            convert.StringToFloat64 `json:"volume24h"`
	Turnover24h          convert.StringToFloat64 `json:"turnover24h"`
	TotalVolume          convert.StringToFloat64 `json:"totalVolume"`
	TotalTurnover        convert.StringToFloat64 `json:"totalTurnover"`
	FundingRate          convert.StringToFloat64 `json:"fundingRate"`
	PredictedFundingRate convert.StringToFloat64 `json:"predictedFundingRate"`
	CountdownHour        convert.StringToFloat64 `json:"countdownHour"`
	UnderlyingPrice      string                  `json:"underlyingPrice"`
}

// USDCKlineBase stores Kline Base
type USDCKlineBase struct {
	Symbol   string                  `json:"symbol"`
	Period   string                  `json:"period"`
	OpenTime bybitTimeSecStr         `json:"openTime"`
	Open     convert.StringToFloat64 `json:"open"`
	High     convert.StringToFloat64 `json:"high"`
	Low      convert.StringToFloat64 `json:"low"`
	Close    convert.StringToFloat64 `json:"close"`
}

// USDCKline stores kline data
type USDCKline struct {
	USDCKlineBase
	Volume   convert.StringToFloat64 `json:"volume"`
	Turnover convert.StringToFloat64 `json:"turnover"`
}

// USDCOpenInterest stores open interest data
type USDCOpenInterest struct {
	Symbol       string                  `json:"symbol"`
	Timestamp    bybitTimeMilliSecStr    `json:"timestamp"`
	OpenInterest convert.StringToFloat64 `json:"openInterest"`
}

// USDCLargeOrder stores large order data
type USDCLargeOrder struct {
	Symbol    string               `json:"symbol"`
	Side      string               `json:"side"`
	Timestamp bybitTimeMilliSecStr `json:"timestamp"`
	Value     float64              `json:"value"`
}

// USDCAccountRatio stores long-short ratio data
type USDCAccountRatio struct {
	Symbol    string               `json:"symbol"`
	BuyRatio  float64              `json:"buyRatio"`
	SellRatio float64              `json:"sellRatio"`
	Timestamp bybitTimeMilliSecStr `json:"timestamp"`
}

// USDCTrade stores trade data
type USDCTrade struct {
	ID         string                  `json:"id"`
	Symbol     string                  `json:"symbol"`
	OrderPrice convert.StringToFloat64 `json:"orderPrice"`
	OrderQty   convert.StringToFloat64 `json:"orderQty"`
	Side       string                  `json:"side"`
	Timestamp  bybitTimeMilliSecStr    `json:"time"`
}

// USDCCreateOrderResp stores create order response
type USDCCreateOrderResp struct {
	ID          string                  `json:"orderId"`
	OrderLinkID string                  `json:"orderLinkId"`
	Symbol      string                  `json:"symbol"`
	OrderPrice  convert.StringToFloat64 `json:"orderPrice"`
	OrderQty    convert.StringToFloat64 `json:"orderQty"`
	OrderType   string                  `json:"orderType"`
	Side        string                  `json:"side"`
}

// USDCOrder store order data
type USDCOrder struct {
	ID              string                  `json:"orderId"`
	OrderLinkID     string                  `json:"orderLinkId"`
	Symbol          string                  `json:"symbol"`
	OrderType       string                  `json:"orderType"`
	Side            string                  `json:"side"`
	Qty             convert.StringToFloat64 `json:"qty"`
	Price           convert.StringToFloat64 `json:"price"`
	TimeInForce     string                  `json:"timeInForce"`
	TotalOrderValue convert.StringToFloat64 `json:"cumExecValue"`
	TotalFilledQty  convert.StringToFloat64 `json:"cumExecQty"`
	TotalFee        convert.StringToFloat64 `json:"cumExecFee"`
	InitialMargin   string                  `json:"orderIM"`
	OrderStatus     string                  `json:"orderStatus"`
	TakeProfit      convert.StringToFloat64 `json:"takeProfit"`
	StopLoss        convert.StringToFloat64 `json:"stopLoss"`
	TPTriggerBy     string                  `json:"tpTriggerBy"`
	SLTriggerBy     string                  `json:"slTriggerBy"`
	LastExecPrice   float64                 `json:"lastExecPrice"`
	BasePrice       string                  `json:"basePrice"`
	TriggerPrice    convert.StringToFloat64 `json:"triggerPrice"`
	TriggerBy       string                  `json:"triggerBy"`
	ReduceOnly      bool                    `json:"reduceOnly"`
	StopOrderType   string                  `json:"stopOrderType"`
	CloseOnTrigger  string                  `json:"closeOnTrigger"`
	CreatedAt       bybitTimeMilliSecStr    `json:"createdAt"`
}

// USDCOrderHistory stores order history
type USDCOrderHistory struct {
	USDCOrder
	LeavesQty   convert.StringToFloat64 `json:"leavesQty"` // Est. unfilled order qty
	CashFlow    string                  `json:"cashFlow"`
	RealisedPnl convert.StringToFloat64 `json:"realisedPnl"`
	UpdatedAt   bybitTimeMilliSecStr    `json:"updatedAt"`
}

// USDCTradeHistory stores trade history
type USDCTradeHistory struct {
	ID               string                  `json:"orderId"`
	OrderLinkID      string                  `json:"orderLinkId"`
	Symbol           string                  `json:"symbol"`
	Side             string                  `json:"side"`
	TradeID          string                  `json:"tradeId"`
	ExecPrice        convert.StringToFloat64 `json:"execPrice"`
	ExecQty          convert.StringToFloat64 `json:"execQty"`
	ExecFee          convert.StringToFloat64 `json:"execFee"`
	FeeRate          convert.StringToFloat64 `json:"feeRate"`
	ExecType         string                  `json:"execType"`
	ExecValue        convert.StringToFloat64 `json:"execValue"`
	TradeTime        bybitTimeMilliSecStr    `json:"tradeTime"`
	LastLiquidityInd string                  `json:"lastLiquidityInd"`
}

// USDCTxLog stores transaction log data
type USDCTxLog struct {
	TxTime        bybitTimeMilliSecStr    `json:"transactionTime"`
	Symbol        string                  `json:"symbol"`
	Type          string                  `json:"type"`
	Side          string                  `json:"side"`
	Quantity      convert.StringToFloat64 `json:"qty"`
	Size          convert.StringToFloat64 `json:"size"`
	TradePrice    convert.StringToFloat64 `json:"tradePrice"`
	Funding       convert.StringToFloat64 `json:"funding"`
	Fee           convert.StringToFloat64 `json:"fee"`
	CashFlow      string                  `json:"cashFlow"`
	Change        convert.StringToFloat64 `json:"change"`
	WalletBalance convert.StringToFloat64 `json:"walletBalance"`
	FeeRate       convert.StringToFloat64 `json:"feeRate"`
	TradeID       string                  `json:"tradeId"`
	OrderID       string                  `json:"orderId"`
	OrderLinkID   string                  `json:"orderLinkId"`
	Info          string                  `json:"info"`
}

// USDCWalletBalance store USDC wallet balance
type USDCWalletBalance struct {
	Equity           convert.StringToFloat64 `json:"equity"`
	WalletBalance    convert.StringToFloat64 `json:"walletBalance"`
	AvailableBalance convert.StringToFloat64 `json:"availableBalance"`
	AccountIM        convert.StringToFloat64 `json:"accountIM"`
	AccountMM        convert.StringToFloat64 `json:"accountMM"`
	TotalRPL         convert.StringToFloat64 `json:"totalRPL"`
	TotalSessionUPL  convert.StringToFloat64 `json:"totalSessionUPL"`
	TotalSessionRPL  convert.StringToFloat64 `json:"totalSessionRPL"`
}

// USDCAssetInfo stores USDC asset data
type USDCAssetInfo struct {
	BaseCoin   string                  `json:"baseCoin"`
	TotalDelta convert.StringToFloat64 `json:"totalDelta"`
	TotalGamma convert.StringToFloat64 `json:"totalGamma"`
	TotalVega  convert.StringToFloat64 `json:"totalVega"`
	TotalTheta convert.StringToFloat64 `json:"totalTheta"`
	TotalRPL   convert.StringToFloat64 `json:"totalRPL"`
	SessionUPL convert.StringToFloat64 `json:"sessionUPL"`
	SessionRPL convert.StringToFloat64 `json:"sessionRPL"`
	IM         convert.StringToFloat64 `json:"im"`
	MM         convert.StringToFloat64 `json:"mm"`
}

// USDCPosition store USDC position data
type USDCPosition struct {
	Symbol              string                  `json:"symbol"`
	Leverage            convert.StringToFloat64 `json:"leverage"`
	ClosingFee          convert.StringToFloat64 `json:"occClosingFee"`
	LiquidPrice         string                  `json:"liqPrice"`
	Position            float64                 `json:"positionValue"`
	TakeProfit          convert.StringToFloat64 `json:"takeProfit"`
	RiskID              string                  `json:"riskId"`
	TrailingStop        convert.StringToFloat64 `json:"trailingStop"`
	UnrealisedPnl       convert.StringToFloat64 `json:"unrealisedPnl"`
	MarkPrice           convert.StringToFloat64 `json:"markPrice"`
	CumRealisedPnl      convert.StringToFloat64 `json:"cumRealisedPnl"`
	PositionMM          convert.StringToFloat64 `json:"positionMM"`
	PositionIM          convert.StringToFloat64 `json:"positionIM"`
	EntryPrice          convert.StringToFloat64 `json:"entryPrice"`
	Size                convert.StringToFloat64 `json:"size"`
	SessionRPL          convert.StringToFloat64 `json:"sessionRPL"`
	SessionUPL          convert.StringToFloat64 `json:"sessionUPL"`
	StopLoss            convert.StringToFloat64 `json:"stopLoss"`
	OrderMargin         convert.StringToFloat64 `json:"orderMargin"`
	SessionAvgPrice     convert.StringToFloat64 `json:"sessionAvgPrice"`
	CreatedAt           bybitTimeMilliSecStr    `json:"createdAt"`
	UpdatedAt           bybitTimeMilliSecStr    `json:"updatedAt"`
	TpSLMode            string                  `json:"tpSLMode"`
	Side                string                  `json:"side"`
	BustPrice           string                  `json:"bustPrice"`
	PositionStatus      string                  `json:"positionStatus"`
	DeleverageIndicator int64                   `json:"deleverageIndicator"`
}

// USDCSettlementHistory store USDC settlement history data
type USDCSettlementHistory struct {
	Symbol          string                  `json:"symbol"`
	Side            string                  `json:"side"`
	Time            bybitTimeMilliSecStr    `json:"time"`
	Size            convert.StringToFloat64 `json:"size"`
	SessionAvgPrice convert.StringToFloat64 `json:"sessionAvgPrice"`
	MarkPrice       convert.StringToFloat64 `json:"markPrice"`
	SessionRpl      convert.StringToFloat64 `json:"sessionRpl"`
}

// USDCRiskLimit store USDC risk limit data
type USDCRiskLimit struct {
	RiskID         string                  `json:"riskId"`
	Symbol         string                  `json:"symbol"`
	Limit          string                  `json:"limit"`
	Section        []string                `json:"section"`
	StartingMargin convert.StringToFloat64 `json:"startingMargin"`
	MaintainMargin convert.StringToFloat64 `json:"maintainMargin"`
	IsLowestRisk   bool                    `json:"isLowestRisk"`
	MaxLeverage    convert.StringToFloat64 `json:"maxLeverage"`
}

// USDCFundingInfo store USDC funding data
type USDCFundingInfo struct {
	Symbol string                  `json:"symbol"`
	Time   bybitTimeMilliSecStr    `json:"fundingRateTimestamp"`
	Rate   convert.StringToFloat64 `json:"fundingRate"`
}

// CFuturesTradingFeeRate stores trading fee rate
type CFuturesTradingFeeRate struct {
	TakerFeeRate convert.StringToFloat64 `json:"taker_fee_rate"`
	MakerFeeRate convert.StringToFloat64 `json:"maker_fee_rate"`
	UserID       int64                   `json:"user_id"`
}
