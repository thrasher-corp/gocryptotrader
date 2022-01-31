package bybit

import (
	"errors"
	"time"
)

var (
	errTypeAssert = errors.New("type assertion failed")
	errStrParsing = errors.New("parsing string failed")
)

// PairData stores pair data
type PairData struct {
	Name              string  `json:"name"`
	Alias             string  `json:"alias"`
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	BasePrecision     float64 `json:"basePrecision,string"`
	QuotePrecision    float64 `json:"quotePrecision,string"`
	MinTradeQuantity  float64 `json:"minTradeQuantity,string"`
	MinTradeAmount    float64 `json:"minTradeAmount,string"`
	MinPricePrecision float64 `json:"minPricePrecision,string"`
	MaxTradeQuantity  float64 `json:"maxTradeQuantity,string"`
	MaxTradeAmount    float64 `json:"maxTradeAmount,string"`
	Category          int64   `json:"category"`
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Bids   []OrderbookItem
	Asks   []OrderbookItem
	Symbol string
	Time   time.Time
}

// TradeItem stores a single trade
type TradeItem struct {
	CurrencyPair string
	Price        float64
	Side         string
	Volume       float64
	Time         time.Time
}

// KlineItem stores an individual kline data item
type KlineItem struct {
	StartTime        time.Time
	EndTime          time.Time
	Open             float64
	Close            float64
	High             float64
	Low              float64
	Volume           float64
	QuoteAssetVolume float64
	TakerBaseVolume  float64
	TakerQuoteVolume float64
	TradesCount      int64
}

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Time         time.Time
	Symbol       string
	BestBidPrice float64
	BestAskPrice float64
	LastPrice    float64
	OpenPrice    float64
	HighPrice    float64
	LowPrice     float64
	Volume       float64
	QuoteVolume  float64
}

// LastTradePrice contains price for last trade
type LastTradePrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

// TickerData stores ticker data
type TickerData struct {
	Symbol      string
	BidPrice    float64
	BidQuantity float64
	AskPrice    float64
	AskQuantity float64
	Time        time.Time
}

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

var (
	// BybitRequestParamsOrderLimit Limit order
	BybitRequestParamsOrderLimit = RequestParamsOrderType("LIMIT")

	// BybitRequestParamsOrderMarket Market order
	BybitRequestParamsOrderMarket = RequestParamsOrderType("MARKET")

	// BybitRequestParamsOrderLimitMaker LIMIT_MAKER
	BybitRequestParamsOrderLimitMaker = RequestParamsOrderType("LIMIT_MAKER")
)

// RequestParamsTimeForceType Time in force
type RequestParamsTimeForceType string

var (
	// BybitRequestParamsTimeGTC GTC
	BybitRequestParamsTimeGTC = RequestParamsTimeForceType("GTC")

	// BybitRequestParamsTimeFOK FOK
	BybitRequestParamsTimeFOK = RequestParamsTimeForceType("FOK")

	// BybitRequestParamsTimeIOC IOC
	BybitRequestParamsTimeIOC = RequestParamsTimeForceType("IOC")
)

// PlaceOrderRequest request type
type PlaceOrderRequest struct {
	Symbol      string
	Quantity    float64
	Side        string
	TradeType   RequestParamsOrderType
	TimeInForce RequestParamsTimeForceType
	Price       float64
	OrderLinkID string
}

type PlaceOrderResponse struct {
	OrderID     int64                      `json:"orderId"`
	OrderLinkID string                     `json:"orderLinkId"`
	Symbol      string                     `json:"symbol"`
	Time        int64                      `json:"transactTime"`
	Price       float64                    `json:"price,string"`
	Quantity    float64                    `json:"origQty,string"`
	TradeType   RequestParamsOrderType     `json:"type"`
	Side        string                     `json:"side"`
	Status      string                     `json:"status"`
	TimeInForce RequestParamsTimeForceType `json:"timeInForce"`
	AccountID   int64                      `json:"accountId"`
	SymbolName  string                     `json:"symbolName"`
	ExecutedQty float64                    `json:"executedQty,string"`
}

// QueryOrderResponse holds query order data
type QueryOrderResponse struct {
	AccountID           int64                      `json:"accountId"`
	ExchangeID          int64                      `json:"exchangeId"`
	Symbol              string                     `json:"symbol"`
	SymbolName          string                     `json:"symbolName"`
	OrderLinkID         string                     `json:"orderLinkId"`
	OrderID             int64                      `json:"orderId"`
	Price               float64                    `json:"price,string"`
	Quantity            float64                    `json:"origQty,string"`
	ExecutedQty         string                     `json:"executedQty,string"`
	CummulativeQuoteQty string                     `json:"cummulativeQuoteQty,string"`
	AveragePrice        float64                    `json:"avgPrice,string"`
	Status              string                     `json:"status"`
	TimeInForce         RequestParamsTimeForceType `json:"timeInForce"`
	TradeType           RequestParamsOrderType     `json:"type"`
	Side                string                     `json:"side"`
	StopPrice           float64                    `json:"stopPrice,string"`
	IcebergQty          float64                    `json:"icebergQty,string"`
	Time                int64                      `json:"time"`
	UpdateTime          int64                      `json:"updateTime"`
	IsWorking           bool                       `json:"isWorking"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	OrderID     int64                      `json:"orderId"`
	OrderLinkID string                     `json:"orderLinkId"`
	Symbol      string                     `json:"symbol"`
	Status      string                     `json:"status"`
	AccountID   int64                      `json:"accountId"`
	Time        int64                      `json:"transactTime"`
	Price       float64                    `json:"price,string"`
	Quantity    float64                    `json:"origQty,string"`
	ExecutedQty string                     `json:"executedQty,string"`
	TimeInForce RequestParamsTimeForceType `json:"timeInForce"`
	TradeType   RequestParamsOrderType     `json:"type"`
	Side        string                     `json:"side"`
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	Symbol          string  `json:"symbol"`
	ID              int64   `json:"id"`
	OrderID         int64   `json:"orderId"`
	Price           float64 `json:"price,string"`
	Quantity        float64 `json:"qty,string"`
	Commission      float64 `json:"commission,string"`
	CommissionAsset float64 `json:"commissionAsset,string"`
	Time            int64   `json:"time"`
	IsBuyer         bool    `json:"isBuyer"`
	IsMaker         bool    `json:"isMaker"`
	SymbolName      string  `json:"symbolName"`
	MatchOrderId    int64   `json:"matchOrderId"`
	Fee             FeeData `json:""fee`
	FeeTokenId      string  `json:"feeTokenId"`
	FeeAmount       float64 `json:"feeAmount,string"`
	MakerRebate     float64 `json:"makerRebate,string"`
}

type FeeData struct {
	FeeTokenId   int64   `json:"feeTokenId"`
	FeeTokenName string  `json:"feeTokenName"`
	Fee          float64 `json:"fee,string"`
}

// Balance holds wallet balance
type Balance struct {
	Coin     string  `json:"coin"`
	CoinID   string  `json:"coinId"`
	CoinName string  `json:"coinName"`
	Total    float64 `json:"total,string"`
	Free     float64 `json:"free,string"`
	Locked   float64 `json:"locked,string"`
}

// Authenticate stores authentication variables required
type Authenticate struct {
	Args      []string `json:"args"`
	Operation string   `json:"op"`
}

// WsReq has the data used for ws request
type WsReq struct {
	Symbol     string      `json:"symbol"`
	Topic      string      `json:"topic"`
	Event      string      `json:"event"`
	Parameters interface{} `json:"params"`
}

type WsFuturesReq struct {
	Topic string   `json:"op"`
	Args  []string `json:"args"`
}

type WsParams struct {
	Symbol     string `json:"symbol"`
	IsBinary   bool   `json:"binary"`
	SymbolName string `json:"symbolName"`
	KlineType  string `json:"klineType"` // only present in kline ws stream
}

// WsSpotTickerData stores ws ticker data
type WsSpotTickerData struct {
	Symbol  string  `json:"symbol"`
	Bid     float64 `json:"bidPrice,string"`
	Ask     float64 `json:"askPrice,string"`
	BidSize float64 `json:"bidQty,string"`
	AskSize float64 `json:"askQty,string"`
	Time    int64   `json:"time"`
}

// WsSpotTicker stores ws ticker data
type WsSpotTicker struct {
	Topic      string           `json:"topic"`
	Parameters WsParams         `json:"params"`
	Ticker     WsSpotTickerData `json:"data"`
}

type KlineStreamData struct {
	StartTime  time.Time `json:"t"`
	Symbol     string    `json:"s"`
	ClosePrice float64   `json:"c,string"`
	HighPrice  float64   `json:"h,string"`
	LowPrice   float64   `json:"l,string"`
	OpenPrice  float64   `json:"o,string"`
	Volume     float64   `json:"vs,string"`
}

// KlineStream holds the kline stream data
type KlineStream struct {
	Topic      string          `json:"topic"`
	Parameters WsParams        `json:"params"`
	Kline      KlineStreamData `json:"data"`
}

// WsOrderbookData stores ws orderbook data
type WsOrderbookData struct {
	Symbol  string      `json:"s"`
	Time    int64       `json:"t"`
	Version string      `json:"v"`
	Bids    [][2]string `json:"b"`
	Asks    [][2]string `json:"a"`
}

// WsOrderbook stores ws orderbook data
type WsOrderbook struct {
	Topic      string          `json:"topic"`
	Parameters WsParams        `json:"params"`
	OBData     WsOrderbookData `json:"data"`
}

// WsTradeData stores ws trade data
type WsTradeData struct {
	Time  int64   `json:"t"`
	ID    string  `json:"v"`
	Price float64 `json:"p,string"`
	Size  float64 `json:"q,string"`
	Side  bool    `json:"m"`
}

// WsTrade stores ws trades data
type WsTrade struct {
	Topic      string      `json:"topic"`
	Parameters WsParams    `json:"params"`
	TradeData  WsTradeData `json:"data"`
}

// wsAccountInfo defines websocket account info data
type wsAccountInfo struct {
	EventType   string       `json:"e"`
	EventTime   time.Time    `json:"E"`
	CanTrade    bool         `json:"T"`
	CanWithdraw bool         `json:"W"`
	CanDeposit  bool         `json:"D"`
	Balance     []Currencies `json:"B"`
}

type Currencies struct {
	Asset     string  `json:"a"`
	Available float64 `json:"f,string"`
	Locked    float64 `json:"l,string"`
}

// wsOrderUpdate defines websocket account order update data
type wsOrderUpdate struct {
	EventType                         string    `json:"e"`
	EventTime                         time.Time `json:"E"`
	Symbol                            string    `json:"s"`
	ClientOrderID                     string    `json:"c"`
	Side                              string    `json:"S"`
	OrderType                         string    `json:"o"`
	TimeInForce                       string    `json:"f"`
	Quantity                          float64   `json:"q,string"`
	Price                             float64   `json:"p,string"`
	OrderStatus                       string    `json:"X"`
	OrderID                           int64     `json:"i"`
	OpponentOrderID                   string    `json:"M"`
	LastExecutedQuantity              float64   `json:"l,string"`
	CumulativeFilledQuantity          float64   `json:"z,string"`
	LastExecutedPrice                 float64   `json:"L,string"`
	Commission                        float64   `json:"n,string"`
	CommissionAsset                   string    `json:"N"`
	IsNormal                          bool      `json:"u"`
	IsOnOrderBook                     bool      `json:"w"`
	IsLimitMaker                      bool      `json:"m"`
	OrderCreationTime                 time.Time `json:"O"`
	CumulativeQuoteTransactedQuantity float64   `json:"Z,string"`
	AccountID                         string    `json:"A"`
	IsClose                           bool      `json:"C"`
	Leverage                          string    `json:"v"`
}

type WsFuturesOrderbookData struct {
	Price  string `json:"price"`
	Symbol string `json:"symbol"`
	ID     int64  `json:"id"`
	Side   string `json:"side"`
	Size   int    `json:"size"`
}

type WsFuturesOrderbook struct {
	Topic  string                   `json:"topic"`
	Type   string                   `json:"string"`
	OBData []WsFuturesOrderbookData `json:"data"`
}

type WsUSDTOrderbook struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		OBData []WsFuturesOrderbookData `json:"order_book"`
	} `json:"data"`
}

type WsCoinDeltaOrderbook struct {
	Topic  string `json:"topic"`
	Type   string `json:"string"`
	OBData struct {
		Delete []WsFuturesOrderbookData `json:"delete"`
		Update []WsFuturesOrderbookData `json:"update"`
		Insert []WsFuturesOrderbookData `json:"insert"`
	} `json:"data"`
}

type WsFuturesTradeData struct {
	Time               time.Time `json:"timestamp"`
	TimeInMilliseconds int64     `json:"trade_time_ms"`
	Symbol             string    `json:"symbol"`
	Side               string    `json:"side"`
	Size               int       `json:"size"`
	Price              float64   `json:"price"`
	Direction          string    `json:"tick_direction"`
	ID                 string    `json:"trade_id"`
}

type WsFuturesTrade struct {
	Topic     string               `json:"topic"`
	TradeData []WsFuturesTradeData `json:"data"`
}

type WsFuturesKlineData struct {
	StartTime int64   `json:"start"`
	EndTime   int64   `json:"end"`
	Close     float64 `json:"close"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	TurnOver  float64 `json:"turnover"`
	Confirm   bool    `json:"confirm"`
	Timestamp int64   `json:"timestamp"`
}

type WsFuturesKline struct {
	Topic     string               `json:"topic"`
	KlineData []WsFuturesKlineData `json:"data"`
}

type WsInsuranceData struct {
	Currency      string    `json:"currency"`
	Timestamp     time.Time `json:"timestamp"`
	WalletBalance float64   `json:"wallet_balance"`
}

type WsInsurance struct {
	Topic string            `json:"topic"`
	Data  []WsInsuranceData `json:"data"`
}

type WsTickerData struct {
	ID                    string    `json:"id"`
	Symbol                string    `json:"symbol"`
	LastPrice             float64   `json:"last_price,string"`
	BidPrice              float64   `json:"bid1_price"`
	AskPrice              float64   `json:"ask1_price"`
	LastDirection         string    `json:"last_tick_direction"`
	PrevPrice24h          float64   `json:"prev_price_24h,string"`
	Price24hPercentChange int64     `json:"price_24h_pcnt_e6"`
	Price1hPercentChange  int64     `json:"price_1h_pcnt_e6"`
	HighPrice24h          float64   `json:"high_price_24h,string"`
	LowPrice24h           float64   `json:"low_price_24h,string"`
	PrevPrice1h           float64   `json:"prev_price_1h,string"`
	MarkPrice             float64   `json:"mark_price,string"`
	IndexPrice            float64   `json:"index_price,string"`
	OpenInterest          int64     `json:"open_interest"`
	OpenValue             int64     `json:"open_value_e8"`
	TotalTurnOver         int64     `json:"total_turnover_e8"`
	TurnOver24h           int64     `json:"turnover_24h_e8"`
	TotalVolume           int64     `json:"total_volume"`
	Volume24h             int64     `json:"volume_24h"`
	FundingRate           int64     `json:"funding_rate_e6"`
	PredictedFundingRate  int64     `json:"predicted_funding_rate_e6"`
	CreatedAt             time.Time `json:"created_at"`
	UpdateAt              time.Time `json:"updated_at"`
	NextFundingAt         time.Time `json:"next_funding_time"`
	CountDownHour         int64     `json:"countdown_hour"`
}

type WsTicker struct {
	Topic  string       `json:"topic"`
	Ticker WsTickerData `json:"data"`
}

type WsDeltaTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		Delete []WsTickerData `json:"delete"`
		Update []WsTickerData `json:"update"`
		Insert []WsTickerData `json:"insert"`
	} `json:"data"`
}

type WsFuturesTickerData struct {
	ID                    string    `json:"id"`
	Symbol                string    `json:"symbol"`
	SymbolName            string    `json:"symbol_name"`
	SymbolYear            int64     `json:"symbol_year"`
	ContractType          string    `json:"contract_type"`
	Coin                  string    `json:"coin"`
	QuoteSymbol           string    `json:"quote_symbol"`
	Mode                  string    `json:"mode"`
	IsUpBorrowable        int64     `json:"is_up_borrowable"`
	ImportTime            int64     `json:"import_time_e9"`
	StartTradingTime      int64     `json:"start_trading_time_e9"`
	TimeToSettle          int64     `json:"settle_time_e9"`
	SettleFeeRate         int64     `json:"settle_fee_rate_e8"`
	ContractStatus        string    `json:"contract_status"`
	SystemSubsidy         int64     `json:"system_subsidy_e8"`
	LastPrice             float64   `json:"last_price,string"`
	BidPrice              float64   `json:"bid1_price"`
	AskPrice              float64   `json:"ask1_price"`
	LastDirection         string    `json:"last_tick_direction"`
	PrevPrice24h          float64   `json:"prev_price_24h,string"`
	Price24hPercentChange int64     `json:"price_24h_pcnt_e6"`
	Price1hPercentChange  int64     `json:"price_1h_pcnt_e6"`
	HighPrice24h          float64   `json:"high_price_24h,string"`
	LowPrice24h           float64   `json:"low_price_24h,string"`
	PrevPrice1h           float64   `json:"prev_price_1h,string"`
	MarkPrice             float64   `json:"mark_price,string"`
	IndexPrice            float64   `json:"index_price,string"`
	OpenInterest          int64     `json:"open_interest"`
	OpenValue             int64     `json:"open_value_e8"`
	TotalTurnOver         int64     `json:"total_turnover_e8"`
	TurnOver24h           int64     `json:"turnover_24h_e8"`
	TotalVolume           int64     `json:"total_volume"`
	Volume24h             int64     `json:"volume_24h"`
	FairBasis             int64     `json:"fair_basis_e8"`
	FairBasisRate         int64     `json:"fair_basis_rate_e8"`
	BasisInYear           int64     `json:"basis_in_year_e8"`
	ExpectPrice           float64   `json:"expect_price,string"`
	CreatedAt             time.Time `json:"created_at"`
	UpdateAt              time.Time `json:"updated_at"`
}

type WsFuturesTicker struct {
	Topic  string              `json:"topic"`
	Ticker WsFuturesTickerData `json:"data"`
}

type WsDeltaFuturesTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		Delete []WsFuturesTickerData `json:"delete"`
		Update []WsFuturesTickerData `json:"update"`
		Insert []WsFuturesTickerData `json:"insert"`
	} `json:"data"`
}

type WsLiquidationData struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price,string"`
	Qty       int64   `json:"qty"`
	Timestamp int64   `json:"time"`
}

type WsFuturesLiquidation struct {
	Topic string            `json:"topic"`
	Data  WsLiquidationData `json:"data"`
}

type WsFuturesPositionData struct {
	UserID              int64   `json:"user_id"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	Size                int64   `json:"size"`
	PositionID          int64   `json:"position_idx"` // present in Futures position struct only
	Mode                int64   `json:"mode"`         // present in Futures position struct only
	Isolated            bool    `json:"isolated"`     // present in Futures position struct only
	PositionValue       float64 `json:"position_value,string"`
	EntryPrice          float64 `json:"entry_price,string"`
	LiquidPrice         float64 `json:"liq_price,string"`
	BustPrice           float64 `json:"bust_price,string"`
	Leverage            float64 `json:"leverage,string"`
	OrderMargin         float64 `json:"order_margin,string"`
	PositionMargin      float64 `json:"position_margin,string"`
	AvailableBalance    float64 `json:"available_balance,string"`
	TakeProfit          float64 `json:"take_profit,string"`
	TakeProfitTriggerBy string  `json:"tp_trigger_by"`
	StopLoss            float64 `json:"stop_loss,string"`
	StopLossTriggerBy   string  `json:"sl_trigger_by"`
	RealisedPNL         float64 `json:"realised_pnl,string"`
	TrailingStop        float64 `json:"trailing_stop,string"`
	TrailingActive      float64 `json:"trailing_active,string"`
	WalletBalance       float64 `json:"wallet_balance,string"`
	RiskID              int64   `json:"risk_id"`
	ClosingFee          float64 `json:"occ_closing_fee,string"`
	FundingFee          float64 `json:"occ_funding_fee,string"`
	AutoAddMargin       int64   `json:"auto_add_margin"`
	TotalPNL            float64 `json:"cum_realised_pnl,string"`
	Status              string  `json:"position_status"`
	Version             int64   `json:"position_seq"`
}

type WsFuturesPosition struct {
	Topic  string                  `json:"topic"`
	Action string                  `json:"action"`
	Data   []WsFuturesPositionData `json:"data"`
}

type WsFuturesExecutionData struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	OrderID       string    `json:"order_id"`
	ExecutionID   string    `json:"exec_id"`
	OrderLinkID   string    `json:"order_link_id"`
	Price         float64   `json:"price,string"`
	OrderQty      int64     `json:"order_qty"`
	ExecutionType string    `json:"exec_type"`
	ExecutionQty  int64     `json:"exec_qty"`
	ExecutionFee  float64   `json:"exec_fee,string"`
	LeavesQty     int64     `json:"leaves_qty"`
	IsMaker       bool      `json:"is_maker"`
	Time          time.Time `json:"trade_time"`
}

type WsFuturesExecution struct {
	Topic string                   `json:"topic"`
	Data  []WsFuturesExecutionData `json:"data"`
}

type WsOrderData struct {
	OrderID              string    `json:"order_id"`
	OrderLinkID          string    `json:"order_link_id"`
	Symbol               string    `json:"symbol"`
	Side                 string    `json:"side"`
	OrderType            string    `json:"order_type"`
	Price                float64   `json:"price,string"`
	OrderQty             int64     `json:"qty"`
	TimeInForce          string    `json:"time_in_force"`
	CreateType           string    `json:"create_type"`
	CancelType           string    `json:"cancel_type"`
	OrderStatus          string    `json:"order_status"`
	LeavesQty            int64     `json:"leaves_qty"`
	CummulativeExecQty   int64     `json:"cum_exec_qty"`
	CummulativeExecValue float64   `json:"cum_exec_value,string"`
	CummulativeExecFee   float64   `json:"cum_exec_fee,string"`
	TakeProfit           float64   `json:"take_profit,string"`
	StopLoss             float64   `json:"stop_loss,string"`
	TrailingStop         float64   `json:"trailing_stop,string"`
	TrailingActive       float64   `json:"trailing_active,string"`
	LastExecPrice        float64   `json:"last_exec_price,string"`
	ReduceOnly           bool      `json:"reduce_only"`
	CloseOnTrigger       bool      `json:"close_on_trigger"`
	Time                 time.Time `json:"timestamp"`   // present in CoinMarginedFutures and Futures only
	CreateTime           time.Time `json:"create_time"` // present in USDTMarginedFutures only
	UpdateTime           time.Time `json:"update_time"` // present in USDTMarginedFutures only
}

type WsOrder struct {
	Topic string        `json:"topic"`
	Data  []WsOrderData `json:"data"`
}

type WsStopOrderData struct {
	OrderID        string    `json:"order_id"`
	OrderLinkID    string    `json:"order_link_id"`
	UserID         int64     `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          float64   `json:"price,string"`
	OrderQty       int64     `json:"qty"`
	TimeInForce    string    `json:"time_in_force"`
	CreateType     string    `json:"create_type"`
	CancelType     string    `json:"cancel_type"`
	OrderStatus    string    `json:"order_status"`
	StopOrderType  string    `json:"stop_order_type"`
	TriggerBy      string    `json:"trigger_by"`
	TriggerPrice   float64   `json:"trigger_price,string"`
	Time           time.Time `json:"timestamp"`
	CloseOnTrigger bool      `json:"close_on_trigger"`
}

type WsFuturesStopOrder struct {
	Topic string            `json:"topic"`
	Data  []WsStopOrderData `json:"data"`
}

type WsUSDTStopOrderData struct {
	OrderID        string    `json:"stop_order_id"`
	OrderLinkID    string    `json:"order_link_id"`
	UserID         int64     `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          float64   `json:"price,string"`
	OrderQty       int64     `json:"qty"`
	TimeInForce    string    `json:"time_in_force"`
	OrderStatus    string    `json:"order_status"`
	StopOrderType  string    `json:"stop_order_type"`
	TriggerBy      string    `json:"trigger_by"`
	TriggerPrice   float64   `json:"trigger_price,string"`
	ReduceOnly     bool      `json:"reduce_only"`
	CloseOnTrigger bool      `json:"close_on_trigger"`
	CreateTime     time.Time `json:"create_time"`
	UpdateTime     time.Time `json:"update_time"`
}

type WsUSDTFuturesStopOrder struct {
	Topic string                `json:"topic"`
	Data  []WsUSDTStopOrderData `json:"data"`
}

type WsFuturesWalletData struct {
	WalletBalance    float64 `json:"wallet_balance"`
	AvailableBalance float64 `json:"available_balance"`
}

type WsFuturesWallet struct {
	Topic string                `json:"topic"`
	Data  []WsFuturesWalletData `json:"data"`
}
