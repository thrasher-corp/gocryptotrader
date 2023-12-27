package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
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
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
	Size   float64      `json:"size"`
	Side   string       `json:"side"`
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
	ID       int64        `json:"id"`
	Symbol   string       `json:"symbol"`
	Interval string       `json:"interval"`
	OpenTime int64        `json:"open_time"`
	Open     types.Number `json:"open"`
	High     types.Number `json:"high"`
	Low      types.Number `json:"low"`
	Close    types.Number `json:"close"`
	Volume   types.Number `json:"volume"`
	TurnOver types.Number `json:"turnover"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol                 string       `json:"symbol"`
	BidPrice               types.Number `json:"bid_price"`
	AskPrice               types.Number `json:"ask_price"`
	LastPrice              types.Number `json:"last_price"`
	LastTickDirection      string       `json:"last_tick_direction"`
	Price24hAgo            types.Number `json:"prev_price_24h"`
	PricePcntChange24h     types.Number `json:"price_24h_pcnt"`
	HighPrice24h           types.Number `json:"high_price_24h"`
	LowPrice24h            types.Number `json:"low_price_24h"`
	Price1hAgo             types.Number `json:"prev_price_1h"`
	PricePcntChange1h      types.Number `json:"price_1h_pcnt"`
	MarkPrice              types.Number `json:"mark_price"`
	IndexPrice             types.Number `json:"index_price"`
	OpenInterest           float64      `json:"open_interest"`
	OpenValue              types.Number `json:"open_value"`
	TotalTurnover          types.Number `json:"total_turnover"`
	Turnover24h            types.Number `json:"turnover_24h"`
	TotalVolume            float64      `json:"total_volume"`
	Volume24h              float64      `json:"volume_24h"`
	FundingRate            types.Number `json:"funding_rate"`
	PredictedFundingRate   types.Number `json:"predicted_funding_rate"`
	NextFundingTime        string       `json:"next_funding_time"`
	CountdownHour          int64        `json:"countdown_hour"`
	DeliveryFeeRate        types.Number `json:"delivery_fee_rate"`
	PredictedDeliveryPrice types.Number `json:"predicted_delivery_price"`
	DeliveryTime           string       `json:"delivery_time"`
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
		MinLeverage  float64      `json:"min_leverage"`
		MaxLeverage  float64      `json:"max_leverage"`
		LeverageStep types.Number `json:"leverage_step"`
	} `json:"leverage_filter"`
	PriceFilter struct {
		MinPrice types.Number `json:"min_price"`
		MaxPrice types.Number `json:"max_price"`
		TickSize types.Number `json:"tick_size"`
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
	Symbol   string       `json:"symbol"`
	Interval string       `json:"period"`
	StartAt  int64        `json:"open_time"`
	Open     types.Number `json:"open"`
	High     types.Number `json:"high"`
	Low      types.Number `json:"low"`
	Close    types.Number `json:"close"`
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
	ExtensionField     map[string]interface{} `json:"ext_fields"`
	LastExecutionTime  string                 `json:"last_exec_time"`
	LastExecutionPrice float64                `json:"last_exec_price"`
	LeavesQty          float64                `json:"leaves_qty"`
	LeaveValue         types.Number           `json:"leaves_value"`
	CumulativeQty      types.Number           `json:"cum_exec_qty"`
	CumulativeValue    types.Number           `json:"cum_exec_value"`
	CumulativeFee      types.Number           `json:"cum_exec_fee"`
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
	LeaveValue      types.Number           `json:"leaves_value"`
	CumulativeQty   types.Number           `json:"cum_exec_qty"`
	CumulativeValue types.Number           `json:"cum_exec_value"`
	CumulativeFee   types.Number           `json:"cum_exec_fee"`
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
	StopOrderType     string       `json:"stop_order_type"`
	TriggerBy         string       `json:"trigger_by"`
	BasePrice         types.Number `json:"base_price"`
	ExpectedDirection string       `json:"expected_direction"`
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
	ID             int64        `json:"id"`
	Symbol         string       `json:"symbol"`
	Limit          int64        `json:"limit"`
	MaintainMargin types.Number `json:"maintain_margin"`
	StartingMargin types.Number `json:"starting_margin"`
	Section        []string     `json:"section"`
	IsLowestRisk   int64        `json:"is_lowest_risk"`
	CreatedAt      string       `json:"create_at"`
	UpdateAt       string       `json:"updated_at"`
	MaxLeverage    types.Number `json:"max_leverage"`
}

// FundingInfo stores funding information
type FundingInfo struct {
	Symbol               string       `json:"symbol"`
	FundingRate          types.Number `json:"funding_rate"`
	FundingRateTimestamp int64        `json:"funding_rate_timestamp"`
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
	UserID                 int64        `json:"user_id"`
	Symbol                 string       `json:"symbol"`
	Side                   string       `json:"side"`
	Size                   float64      `json:"size"`
	PositionValue          types.Number `json:"position_value"`
	EntryPrice             types.Number `json:"entry_price"`
	LiquidationPrice       types.Number `json:"liq_price"`
	BankruptcyPrice        types.Number `json:"bust_price"`
	Leverage               types.Number `json:"leverage"`
	PositionMargin         types.Number `json:"position_margin"`
	OccupiedClosingFee     types.Number `json:"occ_closing_fee"`
	RealisedPNL            types.Number `json:"realised_pnl"`
	AccumulatedRealisedPNL types.Number `json:"cum_realised_pnl"`
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
	IsIsolated          bool         `json:"is_isolated"`
	AutoAddMargin       int64        `json:"auto_add_margin"`
	UnrealisedPNL       float64      `json:"unrealised_pnl"`
	DeleverageIndicator int64        `json:"deleverage_indicator"`
	RiskID              int64        `json:"risk_id"`
	TakeProfit          types.Number `json:"take_profit"`
	StopLoss            types.Number `json:"stop_loss"`
	TrailingStop        types.Number `json:"trailing_stop"`
}

// PositionResp stores position response
type PositionResp struct {
	PositionDataWithStringParam
	PositionID             int64        `json:"position_idx"`
	Mode                   int64        `json:"mode"`
	ID                     int64        `json:"id"`
	EffectiveLeverage      types.Number `json:"effective_leverage"`
	OccupiedFundingFee     types.Number `json:"occ_funding_fee"`
	PositionStatus         string       `json:"position_status"`
	CalculatedData         string       `json:"oc_calc_data"`
	OrderMargin            types.Number `json:"order_margin"`
	WalletBalance          types.Number `json:"wallet_balance"`
	CrossSequence          int64        `json:"cross_seq"`
	PositionSequence       int64        `json:"position_seq"`
	TakeProfitStopLossMode string       `json:"tp_sl_mode"`
	CreatedAt              string       `json:"created_at"`
	UpdateAt               string       `json:"updated_at"`
}

// SetTradingAndStopResp stores set trading and stop response
type SetTradingAndStopResp struct {
	PositionData
	ID                  int64                  `json:"id"`
	RiskID              int64                  `json:"risk_id"`
	AutoAddMargin       int64                  `json:"auto_add_margin"`
	OccupiedFundingFee  types.Number           `json:"occ_funding_fee"`
	TakeProfit          types.Number           `json:"take_profit"`
	StopLoss            types.Number           `json:"stop_loss"`
	PositionStatus      string                 `json:"position_status"`
	DeleverageIndicator int64                  `json:"deleverage_indicator"`
	CalculatedData      string                 `json:"oc_calc_data"`
	OrderMargin         types.Number           `json:"order_margin"`
	WalletBalance       types.Number           `json:"wallet_balance"`
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
	OrderID        string       `json:"order_id"`
	OrderLinkedID  string       `json:"order_link_id"`
	OrderSide      string       `json:"side"`
	Symbol         string       `json:"symbol"`
	ExecutionID    string       `json:"exec_id"`
	OrderPrice     float64      `json:"order_price"`
	OrderQty       float64      `json:"order_qty"`
	OrderType      string       `json:"order_type"`
	FeeRate        float64      `json:"fee_rate"`
	ExecutionFee   types.Number `json:"exec_fee"`
	ExecutionPrice types.Number `json:"exec_price"`
	ExecutionQty   float64      `json:"exec_qty"`
	ExecutionType  string       `json:"exec_type"`
	ExecutionValue types.Number `json:"exec_value"`
	LeavesQty      float64      `json:"leaves_qty"`
	ClosedSize     float64      `json:"closed_size"`
	LastLiquidity  string       `json:"last_liquidity_ind"`
	TradeTimeMs    int64        `json:"trade_time_ms"`
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
	ID            int64        `json:"id"`
	UserID        int64        `json:"user_id"`
	Coin          string       `json:"coin"`
	Type          string       `json:"type"`
	Amount        types.Number `json:"amount"`
	TxID          string       `json:"tx_id"`
	Address       string       `json:"address"`
	WalletBalance types.Number `json:"wallet_balance"`
	ExecutionTime string       `json:"exec_time"`
	CrossSequence int64        `json:"cross_seq"`
}

// FundWithdrawalRecord stores funding withdrawal records
type FundWithdrawalRecord struct {
	ID          int64        `json:"id"`
	UserID      int64        `json:"user_id"`
	Coin        string       `json:"coin"`
	Status      string       `json:"status"`
	Amount      types.Number `json:"amount"`
	Fee         float64      `json:"fee"`
	Address     string       `json:"address"`
	TxID        string       `json:"tx_id"`
	SubmittedAt time.Time    `json:"submited_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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
	Price types.Number `json:"price"`
	Size  types.Number `json:"size"`
	Side  string       `json:"side"`
}

// USDCContract stores contract data
type USDCContract struct {
	Symbol        string               `json:"symbol"`
	Status        string               `json:"status"`
	BaseCoin      string               `json:"baseCoin"`
	QuoteCoin     string               `json:"quoteCoin"`
	TakerFeeRate  types.Number         `json:"takerFeeRate"`
	MakerFeeRate  types.Number         `json:"makerFeeRate"`
	MinLeverage   types.Number         `json:"minLeverage"`
	MaxLeverage   types.Number         `json:"maxLeverage"`
	LeverageStep  types.Number         `json:"leverageStep"`
	MinPrice      types.Number         `json:"minPrice"`
	MaxPrice      types.Number         `json:"maxPrice"`
	TickSize      types.Number         `json:"tickSize"`
	MaxTradingQty types.Number         `json:"maxTradingQty"`
	MinTradingQty types.Number         `json:"minTradingQty"`
	QtyStep       types.Number         `json:"qtyStep"`
	DeliveryTime  bybitTimeMilliSecStr `json:"deliveryTime"`
}

// USDCSymbol stores symbol data
type USDCSymbol struct {
	Symbol               string       `json:"symbol"`
	NextFundingTime      string       `json:"nextFundingTime"`
	Bid                  types.Number `json:"bid"`
	BidSize              types.Number `json:"bidSize"`
	Ask                  types.Number `json:"ask"`
	AskSize              types.Number `json:"askSize"`
	LastPrice            types.Number `json:"lastPrice"`
	OpenInterest         types.Number `json:"openInterest"`
	IndexPrice           types.Number `json:"indexPrice"`
	MarkPrice            types.Number `json:"markPrice"`
	Change24h            types.Number `json:"change24h"`
	High24h              types.Number `json:"high24h"`
	Low24h               types.Number `json:"low24h"`
	Volume24h            types.Number `json:"volume24h"`
	Turnover24h          types.Number `json:"turnover24h"`
	TotalVolume          types.Number `json:"totalVolume"`
	TotalTurnover        types.Number `json:"totalTurnover"`
	FundingRate          types.Number `json:"fundingRate"`
	PredictedFundingRate types.Number `json:"predictedFundingRate"`
	CountdownHour        types.Number `json:"countdownHour"`
	UnderlyingPrice      string       `json:"underlyingPrice"`
}

// USDCKlineBase stores Kline Base
type USDCKlineBase struct {
	Symbol   string          `json:"symbol"`
	Period   string          `json:"period"`
	OpenTime bybitTimeSecStr `json:"openTime"`
	Open     types.Number    `json:"open"`
	High     types.Number    `json:"high"`
	Low      types.Number    `json:"low"`
	Close    types.Number    `json:"close"`
}

// USDCKline stores kline data
type USDCKline struct {
	USDCKlineBase
	Volume   types.Number `json:"volume"`
	Turnover types.Number `json:"turnover"`
}

// USDCOpenInterest stores open interest data
type USDCOpenInterest struct {
	Symbol       string               `json:"symbol"`
	Timestamp    bybitTimeMilliSecStr `json:"timestamp"`
	OpenInterest types.Number         `json:"openInterest"`
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
	ID         string               `json:"id"`
	Symbol     string               `json:"symbol"`
	OrderPrice types.Number         `json:"orderPrice"`
	OrderQty   types.Number         `json:"orderQty"`
	Side       string               `json:"side"`
	Timestamp  bybitTimeMilliSecStr `json:"time"`
}

// USDCCreateOrderResp stores create order response
type USDCCreateOrderResp struct {
	ID          string       `json:"orderId"`
	OrderLinkID string       `json:"orderLinkId"`
	Symbol      string       `json:"symbol"`
	OrderPrice  types.Number `json:"orderPrice"`
	OrderQty    types.Number `json:"orderQty"`
	OrderType   string       `json:"orderType"`
	Side        string       `json:"side"`
}

// USDCOrder store order data
type USDCOrder struct {
	ID              string               `json:"orderId"`
	OrderLinkID     string               `json:"orderLinkId"`
	Symbol          string               `json:"symbol"`
	OrderType       string               `json:"orderType"`
	Side            string               `json:"side"`
	Qty             types.Number         `json:"qty"`
	Price           types.Number         `json:"price"`
	TimeInForce     string               `json:"timeInForce"`
	TotalOrderValue types.Number         `json:"cumExecValue"`
	TotalFilledQty  types.Number         `json:"cumExecQty"`
	TotalFee        types.Number         `json:"cumExecFee"`
	InitialMargin   string               `json:"orderIM"`
	OrderStatus     string               `json:"orderStatus"`
	TakeProfit      types.Number         `json:"takeProfit"`
	StopLoss        types.Number         `json:"stopLoss"`
	TPTriggerBy     string               `json:"tpTriggerBy"`
	SLTriggerBy     string               `json:"slTriggerBy"`
	LastExecPrice   float64              `json:"lastExecPrice"`
	BasePrice       string               `json:"basePrice"`
	TriggerPrice    types.Number         `json:"triggerPrice"`
	TriggerBy       string               `json:"triggerBy"`
	ReduceOnly      bool                 `json:"reduceOnly"`
	StopOrderType   string               `json:"stopOrderType"`
	CloseOnTrigger  string               `json:"closeOnTrigger"`
	CreatedAt       bybitTimeMilliSecStr `json:"createdAt"`
}

// USDCOrderHistory stores order history
type USDCOrderHistory struct {
	USDCOrder
	LeavesQty   types.Number         `json:"leavesQty"` // Est. unfilled order qty
	CashFlow    string               `json:"cashFlow"`
	RealisedPnl types.Number         `json:"realisedPnl"`
	UpdatedAt   bybitTimeMilliSecStr `json:"updatedAt"`
}

// USDCTradeHistory stores trade history
type USDCTradeHistory struct {
	ID               string               `json:"orderId"`
	OrderLinkID      string               `json:"orderLinkId"`
	Symbol           string               `json:"symbol"`
	Side             string               `json:"side"`
	TradeID          string               `json:"tradeId"`
	ExecPrice        types.Number         `json:"execPrice"`
	ExecQty          types.Number         `json:"execQty"`
	ExecFee          types.Number         `json:"execFee"`
	FeeRate          types.Number         `json:"feeRate"`
	ExecType         string               `json:"execType"`
	ExecValue        types.Number         `json:"execValue"`
	TradeTime        bybitTimeMilliSecStr `json:"tradeTime"`
	LastLiquidityInd string               `json:"lastLiquidityInd"`
}

// USDCTxLog stores transaction log data
type USDCTxLog struct {
	TxTime        bybitTimeMilliSecStr `json:"transactionTime"`
	Symbol        string               `json:"symbol"`
	Type          string               `json:"type"`
	Side          string               `json:"side"`
	Quantity      types.Number         `json:"qty"`
	Size          types.Number         `json:"size"`
	TradePrice    types.Number         `json:"tradePrice"`
	Funding       types.Number         `json:"funding"`
	Fee           types.Number         `json:"fee"`
	CashFlow      string               `json:"cashFlow"`
	Change        types.Number         `json:"change"`
	WalletBalance types.Number         `json:"walletBalance"`
	FeeRate       types.Number         `json:"feeRate"`
	TradeID       string               `json:"tradeId"`
	OrderID       string               `json:"orderId"`
	OrderLinkID   string               `json:"orderLinkId"`
	Info          string               `json:"info"`
}

// USDCWalletBalance store USDC wallet balance
type USDCWalletBalance struct {
	Equity           types.Number `json:"equity"`
	WalletBalance    types.Number `json:"walletBalance"`
	AvailableBalance types.Number `json:"availableBalance"`
	AccountIM        types.Number `json:"accountIM"`
	AccountMM        types.Number `json:"accountMM"`
	TotalRPL         types.Number `json:"totalRPL"`
	TotalSessionUPL  types.Number `json:"totalSessionUPL"`
	TotalSessionRPL  types.Number `json:"totalSessionRPL"`
}

// USDCAssetInfo stores USDC asset data
type USDCAssetInfo struct {
	BaseCoin   string       `json:"baseCoin"`
	TotalDelta types.Number `json:"totalDelta"`
	TotalGamma types.Number `json:"totalGamma"`
	TotalVega  types.Number `json:"totalVega"`
	TotalTheta types.Number `json:"totalTheta"`
	TotalRPL   types.Number `json:"totalRPL"`
	SessionUPL types.Number `json:"sessionUPL"`
	SessionRPL types.Number `json:"sessionRPL"`
	IM         types.Number `json:"im"`
	MM         types.Number `json:"mm"`
}

// USDCPosition store USDC position data
type USDCPosition struct {
	Symbol              string               `json:"symbol"`
	Leverage            types.Number         `json:"leverage"`
	ClosingFee          types.Number         `json:"occClosingFee"`
	LiquidPrice         string               `json:"liqPrice"`
	Position            float64              `json:"positionValue"`
	TakeProfit          types.Number         `json:"takeProfit"`
	RiskID              string               `json:"riskId"`
	TrailingStop        types.Number         `json:"trailingStop"`
	UnrealisedPnl       types.Number         `json:"unrealisedPnl"`
	MarkPrice           types.Number         `json:"markPrice"`
	CumRealisedPnl      types.Number         `json:"cumRealisedPnl"`
	PositionMM          types.Number         `json:"positionMM"`
	PositionIM          types.Number         `json:"positionIM"`
	EntryPrice          types.Number         `json:"entryPrice"`
	Size                types.Number         `json:"size"`
	SessionRPL          types.Number         `json:"sessionRPL"`
	SessionUPL          types.Number         `json:"sessionUPL"`
	StopLoss            types.Number         `json:"stopLoss"`
	OrderMargin         types.Number         `json:"orderMargin"`
	SessionAvgPrice     types.Number         `json:"sessionAvgPrice"`
	CreatedAt           bybitTimeMilliSecStr `json:"createdAt"`
	UpdatedAt           bybitTimeMilliSecStr `json:"updatedAt"`
	TpSLMode            string               `json:"tpSLMode"`
	Side                string               `json:"side"`
	BustPrice           string               `json:"bustPrice"`
	PositionStatus      string               `json:"positionStatus"`
	DeleverageIndicator int64                `json:"deleverageIndicator"`
}

// USDCSettlementHistory store USDC settlement history data
type USDCSettlementHistory struct {
	Symbol          string               `json:"symbol"`
	Side            string               `json:"side"`
	Time            bybitTimeMilliSecStr `json:"time"`
	Size            types.Number         `json:"size"`
	SessionAvgPrice types.Number         `json:"sessionAvgPrice"`
	MarkPrice       types.Number         `json:"markPrice"`
	SessionRpl      types.Number         `json:"sessionRpl"`
}

// USDCRiskLimit store USDC risk limit data
type USDCRiskLimit struct {
	RiskID         string       `json:"riskId"`
	Symbol         string       `json:"symbol"`
	Limit          string       `json:"limit"`
	Section        []string     `json:"section"`
	StartingMargin types.Number `json:"startingMargin"`
	MaintainMargin types.Number `json:"maintainMargin"`
	IsLowestRisk   bool         `json:"isLowestRisk"`
	MaxLeverage    types.Number `json:"maxLeverage"`
}

// USDCFundingInfo store USDC funding data
type USDCFundingInfo struct {
	Symbol string               `json:"symbol"`
	Time   bybitTimeMilliSecStr `json:"fundingRateTimestamp"`
	Rate   types.Number         `json:"fundingRate"`
}

// CFuturesTradingFeeRate stores trading fee rate
type CFuturesTradingFeeRate struct {
	TakerFeeRate types.Number `json:"taker_fee_rate"`
	MakerFeeRate types.Number `json:"maker_fee_rate"`
	UserID       int64        `json:"user_id"`
}
