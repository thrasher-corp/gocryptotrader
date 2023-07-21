package bybit

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	errTypeAssert                = errors.New("type assertion failed")
	errStrParsing                = errors.New("parsing string failed")
	errInvalidSide               = errors.New("invalid side")
	errInvalidInterval           = errors.New("invalid interval")
	errInvalidPeriod             = errors.New("invalid period")
	errInvalidStartTime          = errors.New("startTime can't be zero or missing")
	errInvalidQuantity           = errors.New("quantity can't be zero or missing")
	errInvalidBasePrice          = errors.New("basePrice can't be empty or missing")
	errInvalidStopPrice          = errors.New("stopPrice can't be empty or missing")
	errInvalidTimeInForce        = errors.New("timeInForce can't be empty or missing")
	errInvalidTakeProfitStopLoss = errors.New("takeProfitStopLoss can't be empty or missing")
	errInvalidMargin             = errors.New("margin can't be empty")
	errInvalidLeverage           = errors.New("leverage can't be zero or less then it")
	errInvalidRiskID             = errors.New("riskID can't be zero or lesser")
	errInvalidPositionMode       = errors.New("position mode is invalid")
	errInvalidOrderType          = errors.New("orderType can't be empty or missing")
	errInvalidMode               = errors.New("mode can't be empty or missing")
	errInvalidBuyLeverage        = errors.New("buyLeverage can't be zero or less then it")
	errInvalidSellLeverage       = errors.New("sellLeverage can't be zero or less then it")
	errInvalidOrderRequest       = errors.New("order request param can't be nil")
	errInvalidOrderFilter        = errors.New("orderFilter can't be empty or missing")
	errInvalidCategory           = errors.New("invalid category")
	errInvalidCoin               = errors.New("coin can't be empty")

	errStopOrderOrOrderLinkIDMissing = errors.New("at least one should be present among stopOrderID and orderLinkID")
	errOrderOrOrderLinkIDMissing     = errors.New("at least one should be present among orderID and orderLinkID")

	errSymbolMissing    = errors.New("symbol missing")
	errEmptyOrderIDs    = errors.New("orderIDs can't be empty")
	errMissingPrice     = errors.New("price should be present for Limit and LimitMaker orders")
	errExpectedOneOrder = errors.New("expected one order")
)

var validCategory = []string{"spot", "linear", "inverse", "option"}

// UnmarshalTo acts as interface to exchange API response
type UnmarshalTo interface {
	GetError(isAuthRequest bool) error
}

// PairData stores pair data
type PairData struct {
	Name              string      `json:"name"`
	Alias             string      `json:"alias"`
	BaseCurrency      string      `json:"baseCurrency"`
	QuoteCurrency     string      `json:"quoteCurrency"`
	BasePrecision     bybitNumber `json:"basePrecision"`
	QuotePrecision    bybitNumber `json:"quotePrecision"`
	MinTradeQuantity  bybitNumber `json:"minTradeQuantity"`
	MinTradeAmount    bybitNumber `json:"minTradeAmount"`
	MinPricePrecision bybitNumber `json:"minPricePrecision"`
	MaxTradeQuantity  bybitNumber `json:"maxTradeQuantity"`
	MaxTradeAmount    bybitNumber `json:"maxTradeAmount"`
	Category          int64       `json:"category"`
	ShowStatus        bool        `json:"showStatus"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Bids   []orderbook.Item
	Asks   []orderbook.Item
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
	Time         bybitTime   `json:"time"`
	Symbol       string      `json:"symbol"`
	BestBidPrice bybitNumber `json:"bestBidPrice"`
	BestAskPrice bybitNumber `json:"bestAskPrice"`
	LastPrice    bybitNumber `json:"lastPrice"`
	OpenPrice    bybitNumber `json:"openPrice"`
	HighPrice    bybitNumber `json:"highPrice"`
	LowPrice     bybitNumber `json:"lowPrice"`
	Volume       bybitNumber `json:"volume"`
	QuoteVolume  bybitNumber `json:"quoteVolume"`
}

// LastTradePrice contains price for last trade
type LastTradePrice struct {
	Symbol string      `json:"symbol"`
	Price  bybitNumber `json:"price"`
}

// TickerData stores ticker data
type TickerData struct {
	Symbol      string      `json:"symbol"`
	BidPrice    bybitNumber `json:"bidPrice"`
	BidQuantity bybitNumber `json:"bidQty"`
	AskPrice    bybitNumber `json:"askPrice"`
	AskQuantity bybitNumber `json:"askQty"`
	Time        bybitTime   `json:"time"`
}

var (
	// BybitRequestParamsOrderLimit Limit order
	BybitRequestParamsOrderLimit = "LIMIT"

	// BybitRequestParamsOrderMarket Market order
	BybitRequestParamsOrderMarket = "MARKET"

	// BybitRequestParamsOrderLimitMaker Limit Maker
	BybitRequestParamsOrderLimitMaker = "LIMIT_MAKER"
)

var (
	// BybitRequestParamsTimeGTC Good Till Canceled
	BybitRequestParamsTimeGTC = "GTC"

	// BybitRequestParamsTimeFOK Fill or Kill
	BybitRequestParamsTimeFOK = "FOK"

	// BybitRequestParamsTimeIOC Immediate or Cancel
	BybitRequestParamsTimeIOC = "IOC"
)

// PlaceOrderRequest store new order request type
type PlaceOrderRequest struct {
	Symbol      string
	Quantity    float64
	Side        string
	TradeType   string
	TimeInForce string
	Price       float64
	OrderLinkID string
}

// PlaceOrderResponse store new order response type
type PlaceOrderResponse struct {
	OrderID     string      `json:"orderId"`
	OrderLinkID string      `json:"orderLinkId"`
	Symbol      string      `json:"symbol"`
	Time        bybitTime   `json:"transactTime"`
	Price       bybitNumber `json:"price"`
	Quantity    bybitNumber `json:"origQty"`
	TradeType   string      `json:"type"`
	Side        string      `json:"side"`
	Status      string      `json:"status"`
	TimeInForce string      `json:"timeInForce"`
	AccountID   string      `json:"accountId"`
	SymbolName  string      `json:"symbolName"`
	ExecutedQty bybitNumber `json:"executedQty"`
}

// QueryOrderResponse holds query order data
type QueryOrderResponse struct {
	AccountID           string      `json:"accountId"`
	ExchangeID          string      `json:"exchangeId"`
	Symbol              string      `json:"symbol"`
	SymbolName          string      `json:"symbolName"`
	OrderLinkID         string      `json:"orderLinkId"`
	OrderID             string      `json:"orderId"`
	Price               bybitNumber `json:"price"`
	Quantity            bybitNumber `json:"origQty"`
	ExecutedQty         bybitNumber `json:"executedQty"`
	CummulativeQuoteQty bybitNumber `json:"cummulativeQuoteQty"`
	AveragePrice        bybitNumber `json:"avgPrice"`
	Status              string      `json:"status"`
	TimeInForce         string      `json:"timeInForce"`
	TradeType           string      `json:"type"`
	Side                string      `json:"side"`
	StopPrice           bybitNumber `json:"stopPrice"`
	IcebergQty          bybitNumber `json:"icebergQty"`
	Time                bybitTime   `json:"time"`
	UpdateTime          bybitTime   `json:"updateTime"`
	IsWorking           bool        `json:"isWorking"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	OrderID     string      `json:"orderId"`
	OrderLinkID string      `json:"orderLinkId"`
	Symbol      string      `json:"symbol"`
	Status      string      `json:"status"`
	AccountID   string      `json:"accountId"`
	Time        bybitTime   `json:"transactTime"`
	Price       bybitNumber `json:"price"`
	Quantity    bybitNumber `json:"origQty"`
	ExecutedQty bybitNumber `json:"executedQty"`
	TimeInForce string      `json:"timeInForce"`
	TradeType   string      `json:"type"`
	Side        string      `json:"side"`
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	Symbol          string      `json:"symbol"`
	ID              string      `json:"id"`
	OrderID         string      `json:"orderId"`
	TicketID        string      `json:"ticketId"`
	Price           bybitNumber `json:"price"`
	Quantity        bybitNumber `json:"qty"`
	Commission      bybitNumber `json:"commission"`
	CommissionAsset bybitNumber `json:"commissionAsset"`
	Time            bybitTime   `json:"time"`
	IsBuyer         bool        `json:"isBuyer"`
	IsMaker         bool        `json:"isMaker"`
	SymbolName      string      `json:"symbolName"`
	MatchOrderID    string      `json:"matchOrderId"`
	Fee             FeeData     `json:"fee"`
	FeeTokenID      string      `json:"feeTokenId"`
	FeeAmount       bybitNumber `json:"feeAmount"`
	MakerRebate     bybitNumber `json:"makerRebate"`
}

// FeeData store fees data
type FeeData struct {
	FeeTokenID   int64       `json:"feeTokenId"`
	FeeTokenName string      `json:"feeTokenName"`
	Fee          bybitNumber `json:"fee"`
}

// Balance holds wallet balance
type Balance struct {
	Coin     string      `json:"coin"`
	CoinID   string      `json:"coinId"`
	CoinName string      `json:"coinName"`
	Total    bybitNumber `json:"total"`
	Free     bybitNumber `json:"free"`
	Locked   bybitNumber `json:"locked"`
}

type orderbookResponse struct {
	Data struct {
		Asks [][2]string `json:"asks"`
		Bids [][2]string `json:"bids"`
		Time bybitTime   `json:"time"`
	} `json:"result"`
	Error
}

// DepositWalletInfo stores wallet deposit info
type DepositWalletInfo struct {
	Coin   string      `json:"coin"`
	Chains []ChainInfo `json:"chains"`
}

// ChainInfo stores a coins chain info
type ChainInfo struct {
	ChainType      string `json:"chain_type"`
	DepositAddress string `json:"address_deposit"`
	DepositTag     string `json:"tag_deposit"`
	Chain          string `json:"chain"`
}

// Websocket Structures

// Authenticate stores authentication variables required
type Authenticate struct {
	Args      []interface{} `json:"args"`
	Operation string        `json:"op"`
}

// WsReq has the data used for ws request
type WsReq struct {
	Topic      string      `json:"topic"`
	Event      string      `json:"event"`
	Parameters interface{} `json:"params"`
}

// WsResp stores futures ws response
type WsResp struct {
	Success bool   `json:"success"`
	RetMsg  string `json:"ret_msg"`
	ConnID  string `json:"conn_id"`
	Request WsReq  `json:"request"`
}

// WsFuturesReq stores futures ws request
type WsFuturesReq struct {
	Topic string   `json:"op"`
	Args  []string `json:"args"`
}

// WsFuturesResp stores futures ws response
type WsFuturesResp struct {
	Success bool         `json:"success"`
	RetMsg  string       `json:"ret_msg"`
	ConnID  string       `json:"conn_id"`
	Request WsFuturesReq `json:"request"`
}

// WsParams store ws parameters
type WsParams struct {
	Symbol     string `json:"symbol"`
	IsBinary   bool   `json:"binary,string"`
	SymbolName string `json:"symbolName,omitempty"`
	KlineType  string `json:"klineType,omitempty"` // only present in kline ws stream
}

// WsSpotTickerData stores ws ticker data
type WsSpotTickerData struct {
	Symbol  string      `json:"symbol"`
	Bid     bybitNumber `json:"bidPrice"`
	Ask     bybitNumber `json:"askPrice"`
	BidSize bybitNumber `json:"bidQty"`
	AskSize bybitNumber `json:"askQty"`
	Time    bybitTime   `json:"time"`
}

// WsSpotTicker stores ws ticker data
type WsSpotTicker struct {
	Topic      string           `json:"topic"`
	Parameters WsParams         `json:"params"`
	Ticker     WsSpotTickerData `json:"data"`
}

// KlineStreamData stores ws kline stream data
type KlineStreamData struct {
	StartTime  bybitTime   `json:"t"`
	Symbol     string      `json:"s"`
	ClosePrice bybitNumber `json:"c"`
	HighPrice  bybitNumber `json:"h"`
	LowPrice   bybitNumber `json:"l"`
	OpenPrice  bybitNumber `json:"o"`
	Volume     bybitNumber `json:"v"`
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
	Time    bybitTime   `json:"t"`
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
	Time  bybitTime   `json:"t"`
	ID    string      `json:"v"`
	Price bybitNumber `json:"p"`
	Size  bybitNumber `json:"q"`
	Side  bool        `json:"m"`
}

// WsTrade stores ws trades data
type WsTrade struct {
	Topic      string      `json:"topic"`
	Parameters WsParams    `json:"params"`
	TradeData  WsTradeData `json:"data"`
}

// wsAccount defines websocket account info data
type wsAccount struct {
	EventType   string       `json:"e"`
	EventTime   string       `json:"E"`
	CanTrade    bool         `json:"T"`
	CanWithdraw bool         `json:"W"`
	CanDeposit  bool         `json:"D"`
	Balance     []Currencies `json:"B"`
}

// Currencies stores currencies data
type Currencies struct {
	Asset     string      `json:"a"`
	Available bybitNumber `json:"f"`
	Locked    bybitNumber `json:"l"`
}

// wsOrderUpdate defines websocket account order update data
type wsOrderUpdate struct {
	EventType                         string      `json:"e"`
	EventTime                         string      `json:"E"`
	Symbol                            string      `json:"s"`
	ClientOrderID                     string      `json:"c"`
	Side                              string      `json:"S"`
	OrderType                         string      `json:"o"`
	TimeInForce                       string      `json:"f"`
	Quantity                          bybitNumber `json:"q"`
	Price                             bybitNumber `json:"p"`
	OrderStatus                       string      `json:"X"`
	OrderID                           string      `json:"i"`
	OpponentOrderID                   string      `json:"M"`
	LastExecutedQuantity              bybitNumber `json:"l"`
	CumulativeFilledQuantity          bybitNumber `json:"z"`
	LastExecutedPrice                 bybitNumber `json:"L"`
	Commission                        bybitNumber `json:"n"`
	CommissionAsset                   string      `json:"N"`
	IsNormal                          bool        `json:"u"`
	IsOnOrderBook                     bool        `json:"w"`
	IsLimitMaker                      bool        `json:"m"`
	OrderCreationTime                 bybitTime   `json:"O"`
	CumulativeQuoteTransactedQuantity bybitNumber `json:"Z"`
	AccountID                         string      `json:"A"`
	IsClose                           bool        `json:"C"`
	Leverage                          bybitNumber `json:"v"`
}

// wsOrderFilled defines websocket account order filled data
type wsOrderFilled struct {
	EventType         string      `json:"e"`
	EventTime         string      `json:"E"`
	Symbol            string      `json:"s"`
	Quantity          bybitNumber `json:"q"`
	Timestamp         bybitTime   `json:"t"`
	Price             bybitNumber `json:"p"`
	TradeID           string      `json:"T"`
	OrderID           string      `json:"o"`
	UserGenOrderID    string      `json:"c"`
	OpponentOrderID   string      `json:"O"`
	AccountID         string      `json:"a"`
	OpponentAccountID string      `json:"A"`
	IsMaker           bool        `json:"m"`
	Side              string      `json:"S"`
}

// WsFuturesOrderbookData stores ws futures orderbook data
type WsFuturesOrderbookData struct {
	Price  bybitNumber `json:"price"`
	Symbol string      `json:"symbol"`
	ID     bybitNumber `json:"id"`
	Side   string      `json:"side"`
	Size   float64     `json:"size"`
}

// WsFuturesOrderbook stores ws futures orderbook
type WsFuturesOrderbook struct {
	Topic       string          `json:"topic"`
	Type        string          `json:"type"`
	Data        wsFuturesOBData `json:"data"`
	TimestampE6 bybitTime       `json:"timestamp_e6"`
}

type wsFuturesOBData []WsFuturesOrderbookData

// WsUSDTOrderbook stores ws usdt orderbook
type WsUSDTOrderbook struct {
	Topic       string          `json:"topic"`
	Type        string          `json:"type"`
	Data        wsFuturesOBData `json:"data"`
	TimestampE6 bybitTime       `json:"timestamp_e6"`
}

// WsFuturesDeltaOrderbook stores ws futures orderbook deltas
type WsFuturesDeltaOrderbook struct {
	Topic  string `json:"topic"`
	Type   string `json:"type"`
	OBData struct {
		Delete []WsFuturesOrderbookData `json:"delete"`
		Update []WsFuturesOrderbookData `json:"update"`
		Insert []WsFuturesOrderbookData `json:"insert"`
	} `json:"data"`
}

// WsFuturesTradeData stores ws future trade data
type WsFuturesTradeData struct {
	Time               time.Time   `json:"timestamp"`
	TimeInMilliseconds bybitTime   `json:"trade_time_ms"`
	Symbol             string      `json:"symbol"`
	Side               string      `json:"side"`
	Size               float64     `json:"size"`
	Price              bybitNumber `json:"price"`
	Direction          string      `json:"tick_direction"`
	ID                 string      `json:"trade_id"`
}

// WsFuturesTrade stores ws future trade
type WsFuturesTrade struct {
	Topic     string               `json:"topic"`
	TradeData []WsFuturesTradeData `json:"data"`
}

// WsFuturesKlineData stores ws future kline data
type WsFuturesKlineData struct {
	StartTime bybitTime   `json:"start"`
	EndTime   bybitTime   `json:"end"`
	Close     bybitNumber `json:"close"`
	Open      bybitNumber `json:"open"`
	High      bybitNumber `json:"high"`
	Low       bybitNumber `json:"low"`
	Volume    bybitNumber `json:"volume"`
	TurnOver  bybitNumber `json:"turnover"`
	Confirm   bool        `json:"confirm"`
	CrossSeq  bybitNumber `json:"cross_seq"`
	Timestamp bybitTime   `json:"timestamp"`
}

// WsFuturesKline stores ws future kline
type WsFuturesKline struct {
	Topic     string               `json:"topic"`
	KlineData []WsFuturesKlineData `json:"data"`
}

// WsInsuranceData stores ws insurance data
type WsInsuranceData struct {
	Currency      string    `json:"currency"`
	Timestamp     time.Time `json:"timestamp"`
	WalletBalance float64   `json:"wallet_balance"`
}

// WsInsurance stores ws insurance
type WsInsurance struct {
	Topic string            `json:"topic"`
	Data  []WsInsuranceData `json:"data"`
}

// WsTickerData stores ws ticker data
type WsTickerData struct {
	ID                    int64                   `json:"id"`
	Symbol                string                  `json:"symbol"`
	LastPrice             convert.StringToFloat64 `json:"last_price"`
	BidPrice              convert.StringToFloat64 `json:"bid1_price"`
	AskPrice              convert.StringToFloat64 `json:"ask1_price"`
	LastDirection         string                  `json:"last_tick_direction"`
	PrevPrice24h          convert.StringToFloat64 `json:"prev_price_24h"`
	Price24hPercentChange float64                 `json:"price_24h_pcnt_e6"`
	Price1hPercentChange  float64                 `json:"price_1h_pcnt_e6"`
	HighPrice24h          convert.StringToFloat64 `json:"high_price_24h"`
	LowPrice24h           convert.StringToFloat64 `json:"low_price_24h"`
	PrevPrice1h           convert.StringToFloat64 `json:"prev_price_1h"`
	MarkPrice             convert.StringToFloat64 `json:"mark_price"`
	IndexPrice            convert.StringToFloat64 `json:"index_price"`
	OpenInterest          float64                 `json:"open_interest"`
	OpenValue             float64                 `json:"open_value_e8"`
	TotalTurnOver         float64                 `json:"total_turnover_e8"`
	TurnOver24h           float64                 `json:"turnover_24h_e8"`
	TotalVolume           float64                 `json:"total_volume"`
	Volume24h             float64                 `json:"volume_24h"`
	FundingRate           float64                 `json:"funding_rate_e6"`
	PredictedFundingRate  float64                 `json:"predicted_funding_rate_e6"`
	CrossSeq              float64                 `json:"cross_seq"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdateAt              time.Time               `json:"updated_at"`
	NextFundingAt         time.Time               `json:"next_funding_time"`
	CountDownHour         float64                 `json:"countdown_hour"`
	FundingRateInterval   float64                 `json:"funding_rate_interval"`
}

// WsTicker stores ws ticker
type WsTicker struct {
	Topic     string       `json:"topic"`
	Ticker    WsTickerData `json:"data"`
	CrossSeq  float64      `json:"cross_seq"`
	Timestamp bybitTime    `json:"timestamp_e6"`
}

// WsDeltaTicker stores ws ticker
type WsDeltaTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Delete []WsTickerData `json:"delete"`
		Update []WsTickerData `json:"update"`
		Insert []WsTickerData `json:"insert"`
	} `json:"data"`
}

// WsFuturesTickerData stores ws future ticker data
type WsFuturesTickerData struct {
	ID                    int64       `json:"id"`     // Futures
	Symbol                string      `json:"symbol"` // Futures
	SymbolName            string      `json:"symbol_name"`
	SymbolYear            int64       `json:"symbol_year"`
	ContractType          string      `json:"contract_type"`
	Coin                  string      `json:"coin"`
	QuoteSymbol           string      `json:"quote_symbol"`
	Mode                  string      `json:"mode"`
	IsUpBorrowable        int64       `json:"is_up_borrowable"`
	ImportTime            bybitTime   `json:"import_time_e9"`
	StartTradingTime      bybitTime   `json:"start_trading_time_e9"`
	TimeToSettle          bybitTime   `json:"settle_time_e9"` // Futures
	SettleFeeRate         bybitNumber `json:"settle_fee_rate_e8"`
	ContractStatus        string      `json:"contract_status"`
	SystemSubsidy         bybitNumber `json:"system_subsidy_e8"`
	LastPrice             bybitNumber `json:"last_price"`
	BidPrice              bybitNumber `json:"bid1_price"`
	AskPrice              bybitNumber `json:"ask1_price"`
	LastDirection         string      `json:"last_tick_direction"`
	PrevPrice24h          bybitNumber `json:"prev_price_24h"`
	Price24hPercentChange bybitNumber `json:"price_24h_pcnt_e6"`
	Price1hPercentChange  bybitNumber `json:"price_1h_pcnt_e6"`
	HighPrice24h          bybitNumber `json:"high_price_24h"`
	LowPrice24h           bybitNumber `json:"low_price_24h"`
	PrevPrice1h           bybitNumber `json:"prev_price_1h"`
	MarkPrice             bybitNumber `json:"mark_price"`
	IndexPrice            bybitNumber `json:"index_price"`
	OpenInterest          bybitNumber `json:"open_interest"`
	OpenValue             bybitNumber `json:"open_value_e8"`
	TotalTurnOver         bybitNumber `json:"total_turnover_e8"`
	TurnOver24h           bybitNumber `json:"turnover_24h_e8"`
	TotalVolume           bybitNumber `json:"total_volume"`
	Volume24h             bybitNumber `json:"volume_24h"`
	Volume24hE8           bybitNumber `json:"volume_24h_e8"`
	FundingRate           bybitNumber `json:"funding_rate_e6"`
	PredictedFundingRate  bybitNumber `json:"predicted_funding_rate_e6"`
	FairBasis             bybitNumber `json:"fair_basis_e8"`
	FairBasisRate         bybitNumber `json:"fair_basis_rate_e8"`
	BasisInYear           bybitNumber `json:"basis_in_year_e8"`
	ExpectPrice           bybitNumber `json:"expect_price"`
	CrossSeq              bybitNumber `json:"cross_seq"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdateAt              time.Time   `json:"updated_at"`
	NextFundingTime       time.Time   `json:"next_funding_time"`
	CountDownHour         bybitNumber `json:"countdown_hour"`
	FundingRateInterval   bybitNumber `json:"funding_rate_interval"`
	DelistingStatus       string      `json:"delisting_status"`
}

// WsFuturesTicker stores ws future ticker
type WsFuturesTicker struct {
	Topic     string              `json:"topic"`
	Type      string              `json:"type"`
	Ticker    WsFuturesTickerData `json:"data"`
	CrossSeq  bybitNumber         `json:"cross_seq"`
	Timestamp bybitTime           `json:"timestamp_e6"`
}

// WsDeltaFuturesTicker stores ws delta future ticker
type WsDeltaFuturesTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Delete []WsFuturesTickerData `json:"delete"`
		Update []WsFuturesTickerData `json:"update"`
		Insert []WsFuturesTickerData `json:"insert"`
	} `json:"data"`
	CrossSeq  bybitNumber `json:"coss_seq"`
	Timestamp bybitTime   `json:"timestamp_e6"`
}

// WsLiquidationData stores ws liquidation data
type WsLiquidationData struct {
	Symbol    string      `json:"symbol"`
	Side      string      `json:"side"`
	Price     bybitNumber `json:"price"`
	Qty       bybitNumber `json:"qty"`
	Timestamp bybitTime   `json:"time"`
}

// WsFuturesLiquidation stores ws future liquidation
type WsFuturesLiquidation struct {
	Topic string            `json:"topic"`
	Data  WsLiquidationData `json:"data"`
}

// WsFuturesPositionData stores ws future position data
type WsFuturesPositionData struct {
	UserID              int64       `json:"user_id"`
	Symbol              string      `json:"symbol"`
	Side                string      `json:"side"`
	Size                float64     `json:"size"`
	PositionID          int64       `json:"position_idx"` // present in Futures position struct only
	Mode                int64       `json:"mode"`         // present in Futures position struct only
	Isolated            bool        `json:"isolated"`     // present in Futures position struct only
	PositionValue       bybitNumber `json:"position_value"`
	EntryPrice          bybitNumber `json:"entry_price"`
	LiquidPrice         bybitNumber `json:"liq_price"`
	BustPrice           bybitNumber `json:"bust_price"`
	Leverage            bybitNumber `json:"leverage"`
	OrderMargin         bybitNumber `json:"order_margin"`
	PositionMargin      bybitNumber `json:"position_margin"`
	AvailableBalance    bybitNumber `json:"available_balance"`
	TakeProfit          bybitNumber `json:"take_profit"`
	TakeProfitTriggerBy string      `json:"tp_trigger_by"`
	StopLoss            bybitNumber `json:"stop_loss"`
	StopLossTriggerBy   string      `json:"sl_trigger_by"`
	RealisedPNL         bybitNumber `json:"realised_pnl"`
	TrailingStop        bybitNumber `json:"trailing_stop"`
	TrailingActive      bybitNumber `json:"trailing_active"`
	WalletBalance       bybitNumber `json:"wallet_balance"`
	RiskID              int64       `json:"risk_id"`
	ClosingFee          bybitNumber `json:"occ_closing_fee"`
	FundingFee          bybitNumber `json:"occ_funding_fee"`
	AutoAddMargin       int64       `json:"auto_add_margin"`
	TotalPNL            bybitNumber `json:"cum_realised_pnl"`
	Status              string      `json:"position_status"`
	Version             int64       `json:"position_seq"`
}

// WsFuturesPosition stores ws future position
type WsFuturesPosition struct {
	Topic  string                  `json:"topic"`
	Action string                  `json:"action"`
	Data   []WsFuturesPositionData `json:"data"`
}

// WsFuturesExecutionData stores ws future execution data
type WsFuturesExecutionData struct {
	Symbol        string      `json:"symbol"`
	Side          string      `json:"side"`
	OrderID       string      `json:"order_id"`
	ExecutionID   string      `json:"exec_id"`
	OrderLinkID   string      `json:"order_link_id"`
	Price         bybitNumber `json:"price"`
	OrderQty      float64     `json:"order_qty"`
	ExecutionType string      `json:"exec_type"`
	ExecutionQty  float64     `json:"exec_qty"`
	ExecutionFee  bybitNumber `json:"exec_fee"`
	LeavesQty     float64     `json:"leaves_qty"`
	IsMaker       bool        `json:"is_maker"`
	Time          time.Time   `json:"trade_time"`
}

// WsFuturesExecution stores ws future execution
type WsFuturesExecution struct {
	Topic string                   `json:"topic"`
	Data  []WsFuturesExecutionData `json:"data"`
}

// WsOrderData stores ws order data
type WsOrderData struct {
	OrderID              string      `json:"order_id"`
	OrderLinkID          string      `json:"order_link_id"`
	Symbol               string      `json:"symbol"`
	Side                 string      `json:"side"`
	OrderType            string      `json:"order_type"`
	Price                bybitNumber `json:"price"`
	OrderQty             float64     `json:"qty"`
	TimeInForce          string      `json:"time_in_force"`
	CreateType           string      `json:"create_type"`
	CancelType           string      `json:"cancel_type"`
	OrderStatus          string      `json:"order_status"`
	LeavesQty            float64     `json:"leaves_qty"`
	CummulativeExecQty   float64     `json:"cum_exec_qty"`
	CummulativeExecValue bybitNumber `json:"cum_exec_value"`
	CummulativeExecFee   bybitNumber `json:"cum_exec_fee"`
	TakeProfit           bybitNumber `json:"take_profit"`
	StopLoss             bybitNumber `json:"stop_loss"`
	TrailingStop         bybitNumber `json:"trailing_stop"`
	TrailingActive       bybitNumber `json:"trailing_active"`
	LastExecPrice        bybitNumber `json:"last_exec_price"`
	ReduceOnly           bool        `json:"reduce_only"`
	CloseOnTrigger       bool        `json:"close_on_trigger"`
	Time                 time.Time   `json:"timestamp"`   // present in CoinMarginedFutures and Futures only
	CreateTime           time.Time   `json:"create_time"` // present in USDTMarginedFutures only
	UpdateTime           time.Time   `json:"update_time"` // present in USDTMarginedFutures only
}

// WsOrder stores ws order
type WsOrder struct {
	Topic string        `json:"topic"`
	Data  []WsOrderData `json:"data"`
}

// WsStopOrderData stores ws stop order data
type WsStopOrderData struct {
	OrderID        string      `json:"order_id"`
	OrderLinkID    string      `json:"order_link_id"`
	UserID         int64       `json:"user_id"`
	Symbol         string      `json:"symbol"`
	Side           string      `json:"side"`
	OrderType      string      `json:"order_type"`
	Price          bybitNumber `json:"price"`
	OrderQty       float64     `json:"qty"`
	TimeInForce    string      `json:"time_in_force"`
	CreateType     string      `json:"create_type"`
	CancelType     string      `json:"cancel_type"`
	OrderStatus    string      `json:"order_status"`
	StopOrderType  string      `json:"stop_order_type"`
	TriggerBy      string      `json:"trigger_by"`
	TriggerPrice   bybitNumber `json:"trigger_price"`
	ReduceOnly     bool        `json:"reduce_only"`
	Time           time.Time   `json:"timestamp"`
	CreateTime     time.Time   `json:"create_time"`
	UpdateTime     time.Time   `json:"update_time"`
	CloseOnTrigger bool        `json:"close_on_trigger"`
}

// WsFuturesStopOrder stores ws future stop order
type WsFuturesStopOrder struct {
	Topic string            `json:"topic"`
	Data  []WsStopOrderData `json:"data"`
}

// WsFuturesWalletData stores ws future wallet data
type WsFuturesWalletData struct {
	WalletBalance    float64 `json:"wallet_balance"`
	AvailableBalance float64 `json:"available_balance"`
}

// WsFuturesWallet stores ws future wallet
type WsFuturesWallet struct {
	Topic string                `json:"topic"`
	Data  []WsFuturesWalletData `json:"data"`
}

// WsFuturesParams stores futures ws subscription parameters
type WsFuturesParams struct {
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

// Ticker holds ticker information
type Ticker struct {
	// Spot fields
	Symbol            string      `json:"symbol"`
	TopBidPrice       bybitNumber `json:"bid1Price"`
	TopBidSize        bybitNumber `json:"bid1Size"`
	TopAskPrice       bybitNumber `json:"ask1Price"`
	TopAskSize        bybitNumber `json:"ask1Size"`
	LastPrice         bybitNumber `json:"lastPrice"`
	PreviousPrice24Hr bybitNumber `json:"prevPrice24h"`
	Price24HrPcnt     bybitNumber `json:"price24hPcnt"`
	HighPrice24Hr     bybitNumber `json:"highPrice24h"`
	LowPrice24Hr      bybitNumber `json:"lowPrice24h"`
	Turnover24Hr      bybitNumber `json:"turnover24h"`
	Volume24Hr        bybitNumber `json:"volume24h"`
	USDIndexPrice     bybitNumber `json:"usdIndexPrice"`

	// Option fields
	TopBidImpliedVolatility bybitNumber `json:"bid1Iv"`
	TopAskImpliedVolatility bybitNumber `json:"ask1Iv"`
	MarkPrice               bybitNumber `json:"markPrice"`
	IndexPrice              bybitNumber `json:"indexPrice"`
	MarkImpliedVolatility   bybitNumber `json:"markIv"`
	UnderlyingPrice         bybitNumber `json:"underlyingPrice"`
	OpenInterest            bybitNumber `json:"openInterest"`
	TotalVolume             bybitNumber `json:"totalVolume"`
	TotalTurnover           bybitNumber `json:"totalTurnover"`
	Delta                   bybitNumber `json:"delta"`
	Gamma                   bybitNumber `json:"gamma"`
	Vega                    bybitNumber `json:"vega"`
	Theta                   bybitNumber `json:"theta"`
	PredictedDeliveryPrice  bybitNumber `json:"predictedDeliveryPrice"`
	Change24h               bybitNumber `json:"change24h"`

	// Inverse/linear  fields
	PrevPrice1h       bybitNumber `json:"prevPrice1h"`
	OpenInterestValue bybitNumber `json:"openInterestValue"`
	FundingRate       bybitNumber `json:"fundingRate"`
	NextFundingTime   bybitNumber `json:"nextFundingTime"`
	BasisRate         bybitNumber `json:"basisRate"`
	DeliveryFeeRate   bybitNumber `json:"deliveryFeeRate"`
	DeliveryTime      bybitNumber `json:"deliveryTime"`
	Basis             bybitNumber `json:"basis"`
}

// Fee holds fee information
type Fee struct {
	BaseCoin string      `json:"baseCoin"`
	Symbol   string      `json:"symbol"`
	Taker    bybitNumber `json:"takerFeeRate"`
	Maker    bybitNumber `json:"makerFeeRate"`
}

// AccountFee holds account fee information
type AccountFee struct {
	Category string `json:"category"`
	List     []Fee  `json:"list"`
}

// ListOfTickers holds list of tickers
type ListOfTickers struct {
	Category string   `json:"category"`
	List     []Ticker `json:"list"`
}
