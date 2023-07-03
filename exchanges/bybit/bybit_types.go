package bybit

import (
	"encoding/json"
	"errors"
	"strconv"
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

// bybitTimeSec provides an internal conversion helper
type bybitTimeSec time.Time

// UnmarshalJSON is custom json unmarshaller for bybitTimeSec
func (b *bybitTimeSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*b = bybitTimeSec(time.Unix(timestamp, 0))
	return nil
}

// Time returns a time.Time object
func (b bybitTimeSec) Time() time.Time {
	return time.Time(b)
}

// bybitTimeSecStr provides an internal conversion helper
type bybitTimeSecStr time.Time

// UnmarshalJSON is custom json unmarshaller for bybitTimeSec
func (b *bybitTimeSecStr) UnmarshalJSON(data []byte) error {
	var timestamp string
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}

	t, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	*b = bybitTimeSecStr(time.Unix(t, 0))
	return nil
}

// Time returns a time.Time object
func (b bybitTimeSecStr) Time() time.Time {
	return time.Time(b)
}

// bybitTimeMilliSec provides an internal conversion helper
type bybitTimeMilliSec time.Time

// UnmarshalJSON is custom type json unmarshaller for bybitTimeMilliSec
func (b *bybitTimeMilliSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*b = bybitTimeMilliSec(time.UnixMilli(timestamp))
	return nil
}

// Time returns a time.Time object
func (b bybitTimeMilliSec) Time() time.Time {
	return time.Time(b)
}

// bybitTimeMilliSecStr provides an internal conversion helper
type bybitTimeMilliSecStr time.Time

// UnmarshalJSON is custom type json unmarshaller for bybitTimeMilliSec
func (b *bybitTimeMilliSecStr) UnmarshalJSON(data []byte) error {
	var timestamp string
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}

	t, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	*b = bybitTimeMilliSecStr(time.UnixMilli(t))
	return nil
}

// Time returns a time.Time object
func (b bybitTimeMilliSecStr) Time() time.Time {
	return time.Time(b)
}

// bybitTimeNanoSec provides an internal conversion helper
type bybitTimeNanoSec time.Time

// UnmarshalJSON is custom type json unmarshaller for bybitTimeNanoSec
func (b *bybitTimeNanoSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*b = bybitTimeNanoSec(time.Unix(0, timestamp))
	return nil
}

// Time returns a time.Time object
func (b bybitTimeNanoSec) Time() time.Time {
	return time.Time(b)
}

// UnmarshalTo acts as interface to exchange API response
type UnmarshalTo interface {
	GetError(isAuthRequest bool) error
}

// PairData stores pair data
type PairData struct {
	Name              string                  `json:"name"`
	Alias             string                  `json:"alias"`
	BaseCurrency      string                  `json:"baseCurrency"`
	QuoteCurrency     string                  `json:"quoteCurrency"`
	BasePrecision     convert.StringToFloat64 `json:"basePrecision"`
	QuotePrecision    convert.StringToFloat64 `json:"quotePrecision"`
	MinTradeQuantity  convert.StringToFloat64 `json:"minTradeQuantity"`
	MinTradeAmount    convert.StringToFloat64 `json:"minTradeAmount"`
	MinPricePrecision convert.StringToFloat64 `json:"minPricePrecision"`
	MaxTradeQuantity  convert.StringToFloat64 `json:"maxTradeQuantity"`
	MaxTradeAmount    convert.StringToFloat64 `json:"maxTradeAmount"`
	Category          int64                   `json:"category"`
	ShowStatus        bool                    `json:"showStatus"`
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
	Time         bybitTimeMilliSec       `json:"time"`
	Symbol       string                  `json:"symbol"`
	BestBidPrice convert.StringToFloat64 `json:"bestBidPrice"`
	BestAskPrice convert.StringToFloat64 `json:"bestAskPrice"`
	LastPrice    convert.StringToFloat64 `json:"lastPrice"`
	OpenPrice    convert.StringToFloat64 `json:"openPrice"`
	HighPrice    convert.StringToFloat64 `json:"highPrice"`
	LowPrice     convert.StringToFloat64 `json:"lowPrice"`
	Volume       convert.StringToFloat64 `json:"volume"`
	QuoteVolume  convert.StringToFloat64 `json:"quoteVolume"`
}

// LastTradePrice contains price for last trade
type LastTradePrice struct {
	Symbol string                  `json:"symbol"`
	Price  convert.StringToFloat64 `json:"price"`
}

// TickerData stores ticker data
type TickerData struct {
	Symbol      string                  `json:"symbol"`
	BidPrice    convert.StringToFloat64 `json:"bidPrice"`
	BidQuantity convert.StringToFloat64 `json:"bidQty"`
	AskPrice    convert.StringToFloat64 `json:"askPrice"`
	AskQuantity convert.StringToFloat64 `json:"askQty"`
	Time        bybitTimeMilliSec       `json:"time"`
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
	OrderID     string                  `json:"orderId"`
	OrderLinkID string                  `json:"orderLinkId"`
	Symbol      string                  `json:"symbol"`
	Time        bybitTimeMilliSecStr    `json:"transactTime"`
	Price       convert.StringToFloat64 `json:"price"`
	Quantity    convert.StringToFloat64 `json:"origQty"`
	TradeType   string                  `json:"type"`
	Side        string                  `json:"side"`
	Status      string                  `json:"status"`
	TimeInForce string                  `json:"timeInForce"`
	AccountID   string                  `json:"accountId"`
	SymbolName  string                  `json:"symbolName"`
	ExecutedQty convert.StringToFloat64 `json:"executedQty"`
}

// QueryOrderResponse holds query order data
type QueryOrderResponse struct {
	AccountID           string                  `json:"accountId"`
	ExchangeID          string                  `json:"exchangeId"`
	Symbol              string                  `json:"symbol"`
	SymbolName          string                  `json:"symbolName"`
	OrderLinkID         string                  `json:"orderLinkId"`
	OrderID             string                  `json:"orderId"`
	Price               convert.StringToFloat64 `json:"price"`
	Quantity            convert.StringToFloat64 `json:"origQty"`
	ExecutedQty         convert.StringToFloat64 `json:"executedQty"`
	CummulativeQuoteQty convert.StringToFloat64 `json:"cummulativeQuoteQty"`
	AveragePrice        convert.StringToFloat64 `json:"avgPrice"`
	Status              string                  `json:"status"`
	TimeInForce         string                  `json:"timeInForce"`
	TradeType           string                  `json:"type"`
	Side                string                  `json:"side"`
	StopPrice           convert.StringToFloat64 `json:"stopPrice"`
	IcebergQty          convert.StringToFloat64 `json:"icebergQty"`
	Time                bybitTimeMilliSecStr    `json:"time"`
	UpdateTime          bybitTimeMilliSecStr    `json:"updateTime"`
	IsWorking           bool                    `json:"isWorking"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	OrderID     string                  `json:"orderId"`
	OrderLinkID string                  `json:"orderLinkId"`
	Symbol      string                  `json:"symbol"`
	Status      string                  `json:"status"`
	AccountID   string                  `json:"accountId"`
	Time        bybitTimeMilliSecStr    `json:"transactTime"`
	Price       convert.StringToFloat64 `json:"price"`
	Quantity    convert.StringToFloat64 `json:"origQty"`
	ExecutedQty convert.StringToFloat64 `json:"executedQty"`
	TimeInForce string                  `json:"timeInForce"`
	TradeType   string                  `json:"type"`
	Side        string                  `json:"side"`
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	Symbol          string                  `json:"symbol"`
	ID              string                  `json:"id"`
	OrderID         string                  `json:"orderId"`
	TicketID        string                  `json:"ticketId"`
	Price           convert.StringToFloat64 `json:"price"`
	Quantity        convert.StringToFloat64 `json:"qty"`
	Commission      convert.StringToFloat64 `json:"commission"`
	CommissionAsset convert.StringToFloat64 `json:"commissionAsset"`
	Time            bybitTimeMilliSecStr    `json:"time"`
	IsBuyer         bool                    `json:"isBuyer"`
	IsMaker         bool                    `json:"isMaker"`
	SymbolName      string                  `json:"symbolName"`
	MatchOrderID    string                  `json:"matchOrderId"`
	Fee             FeeData                 `json:"fee"`
	FeeTokenID      string                  `json:"feeTokenId"`
	FeeAmount       convert.StringToFloat64 `json:"feeAmount"`
	MakerRebate     convert.StringToFloat64 `json:"makerRebate"`
}

// FeeData store fees data
type FeeData struct {
	FeeTokenID   int64                   `json:"feeTokenId"`
	FeeTokenName string                  `json:"feeTokenName"`
	Fee          convert.StringToFloat64 `json:"fee"`
}

// Balance holds wallet balance
type Balance struct {
	Coin     string                  `json:"coin"`
	CoinID   string                  `json:"coinId"`
	CoinName string                  `json:"coinName"`
	Total    convert.StringToFloat64 `json:"total"`
	Free     convert.StringToFloat64 `json:"free"`
	Locked   convert.StringToFloat64 `json:"locked"`
}

type orderbookResponse struct {
	Data struct {
		Asks [][2]string       `json:"asks"`
		Bids [][2]string       `json:"bids"`
		Time bybitTimeMilliSec `json:"time"`
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

// WsFuturesReq stores futures ws request
type WsFuturesReq struct {
	Topic string   `json:"op"`
	Args  []string `json:"args"`
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
	Symbol  string                  `json:"symbol"`
	Bid     convert.StringToFloat64 `json:"bidPrice"`
	Ask     convert.StringToFloat64 `json:"askPrice"`
	BidSize convert.StringToFloat64 `json:"bidQty"`
	AskSize convert.StringToFloat64 `json:"askQty"`
	Time    bybitTimeMilliSec       `json:"time"`
}

// WsSpotTicker stores ws ticker data
type WsSpotTicker struct {
	Topic      string           `json:"topic"`
	Parameters WsParams         `json:"params"`
	Ticker     WsSpotTickerData `json:"data"`
}

// KlineStreamData stores ws kline stream data
type KlineStreamData struct {
	StartTime  bybitTimeMilliSec       `json:"t"`
	Symbol     string                  `json:"s"`
	ClosePrice convert.StringToFloat64 `json:"c"`
	HighPrice  convert.StringToFloat64 `json:"h"`
	LowPrice   convert.StringToFloat64 `json:"l"`
	OpenPrice  convert.StringToFloat64 `json:"o"`
	Volume     convert.StringToFloat64 `json:"v"`
}

// KlineStream holds the kline stream data
type KlineStream struct {
	Topic      string          `json:"topic"`
	Parameters WsParams        `json:"params"`
	Kline      KlineStreamData `json:"data"`
}

// WsOrderbookData stores ws orderbook data
type WsOrderbookData struct {
	Symbol  string            `json:"s"`
	Time    bybitTimeMilliSec `json:"t"`
	Version string            `json:"v"`
	Bids    [][2]string       `json:"b"`
	Asks    [][2]string       `json:"a"`
}

// WsOrderbook stores ws orderbook data
type WsOrderbook struct {
	Topic      string          `json:"topic"`
	Parameters WsParams        `json:"params"`
	OBData     WsOrderbookData `json:"data"`
}

// WsTradeData stores ws trade data
type WsTradeData struct {
	Time  bybitTimeMilliSec       `json:"t"`
	ID    string                  `json:"v"`
	Price convert.StringToFloat64 `json:"p"`
	Size  convert.StringToFloat64 `json:"q"`
	Side  bool                    `json:"m"`
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
	Asset     string                  `json:"a"`
	Available convert.StringToFloat64 `json:"f"`
	Locked    convert.StringToFloat64 `json:"l"`
}

// wsOrderUpdate defines websocket account order update data
type wsOrderUpdate struct {
	EventType                         string                  `json:"e"`
	EventTime                         string                  `json:"E"`
	Symbol                            string                  `json:"s"`
	ClientOrderID                     string                  `json:"c"`
	Side                              string                  `json:"S"`
	OrderType                         string                  `json:"o"`
	TimeInForce                       string                  `json:"f"`
	Quantity                          convert.StringToFloat64 `json:"q"`
	Price                             convert.StringToFloat64 `json:"p"`
	OrderStatus                       string                  `json:"X"`
	OrderID                           string                  `json:"i"`
	OpponentOrderID                   string                  `json:"M"`
	LastExecutedQuantity              convert.StringToFloat64 `json:"l"`
	CumulativeFilledQuantity          convert.StringToFloat64 `json:"z"`
	LastExecutedPrice                 convert.StringToFloat64 `json:"L"`
	Commission                        convert.StringToFloat64 `json:"n"`
	CommissionAsset                   string                  `json:"N"`
	IsNormal                          bool                    `json:"u"`
	IsOnOrderBook                     bool                    `json:"w"`
	IsLimitMaker                      bool                    `json:"m"`
	OrderCreationTime                 bybitTimeMilliSecStr    `json:"O"`
	CumulativeQuoteTransactedQuantity convert.StringToFloat64 `json:"Z"`
	AccountID                         string                  `json:"A"`
	IsClose                           bool                    `json:"C"`
	Leverage                          convert.StringToFloat64 `json:"v"`
}

// wsOrderFilled defines websocket account order filled data
type wsOrderFilled struct {
	EventType         string                  `json:"e"`
	EventTime         string                  `json:"E"`
	Symbol            string                  `json:"s"`
	Quantity          convert.StringToFloat64 `json:"q"`
	Timestamp         bybitTimeMilliSecStr    `json:"t"`
	Price             convert.StringToFloat64 `json:"p"`
	TradeID           string                  `json:"T"`
	OrderID           string                  `json:"o"`
	UserGenOrderID    string                  `json:"c"`
	OpponentOrderID   string                  `json:"O"`
	AccountID         string                  `json:"a"`
	OpponentAccountID string                  `json:"A"`
	IsMaker           bool                    `json:"m"`
	Side              string                  `json:"S"`
}

// WsFuturesOrderbookData stores ws futures orderbook data
type WsFuturesOrderbookData struct {
	Price  convert.StringToFloat64 `json:"price"`
	Symbol string                  `json:"symbol"`
	ID     int64                   `json:"id"`
	Side   string                  `json:"side"`
	Size   float64                 `json:"size"`
}

// WsFuturesOrderbook stores ws futures orderbook
type WsFuturesOrderbook struct {
	Topic  string                   `json:"topic"`
	Type   string                   `json:"string"`
	OBData []WsFuturesOrderbookData `json:"data"`
}

// WsUSDTOrderbook stores ws usdt orderbook
type WsUSDTOrderbook struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		OBData []WsFuturesOrderbookData `json:"order_book"`
	} `json:"data"`
}

// WsCoinDeltaOrderbook stores ws coinmargined orderbook
type WsCoinDeltaOrderbook struct {
	Topic  string `json:"topic"`
	Type   string `json:"string"`
	OBData struct {
		Delete []WsFuturesOrderbookData `json:"delete"`
		Update []WsFuturesOrderbookData `json:"update"`
		Insert []WsFuturesOrderbookData `json:"insert"`
	} `json:"data"`
}

// WsFuturesTradeData stores ws future trade data
type WsFuturesTradeData struct {
	Time               time.Time         `json:"timestamp"`
	TimeInMilliseconds bybitTimeMilliSec `json:"trade_time_ms"`
	Symbol             string            `json:"symbol"`
	Side               string            `json:"side"`
	Size               float64           `json:"size"`
	Price              float64           `json:"price"`
	Direction          string            `json:"tick_direction"`
	ID                 string            `json:"trade_id"`
}

// WsFuturesTrade stores ws future trade
type WsFuturesTrade struct {
	Topic     string               `json:"topic"`
	TradeData []WsFuturesTradeData `json:"data"`
}

// WsFuturesKlineData stores ws future kline data
type WsFuturesKlineData struct {
	StartTime bybitTimeSec      `json:"start"`
	EndTime   bybitTimeSec      `json:"end"`
	Close     float64           `json:"close"`
	Open      float64           `json:"open"`
	High      float64           `json:"high"`
	Low       float64           `json:"low"`
	Volume    float64           `json:"volume"`
	TurnOver  float64           `json:"turnover"`
	Confirm   bool              `json:"confirm"`
	Timestamp bybitTimeMilliSec `json:"timestamp"`
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
	ID                    string                  `json:"id"`
	Symbol                string                  `json:"symbol"`
	LastPrice             convert.StringToFloat64 `json:"last_price"`
	BidPrice              float64                 `json:"bid1_price"`
	AskPrice              float64                 `json:"ask1_price"`
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
	FundingRate           int64                   `json:"funding_rate_e6"`
	PredictedFundingRate  float64                 `json:"predicted_funding_rate_e6"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdateAt              time.Time               `json:"updated_at"`
	NextFundingAt         time.Time               `json:"next_funding_time"`
	CountDownHour         int64                   `json:"countdown_hour"`
}

// WsTicker stores ws ticker
type WsTicker struct {
	Topic  string       `json:"topic"`
	Ticker WsTickerData `json:"data"`
}

// WsDeltaTicker stores ws ticker
type WsDeltaTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		Delete []WsTickerData `json:"delete"`
		Update []WsTickerData `json:"update"`
		Insert []WsTickerData `json:"insert"`
	} `json:"data"`
}

// WsFuturesTickerData stores ws future ticker data
type WsFuturesTickerData struct {
	ID                    string                  `json:"id"`
	Symbol                string                  `json:"symbol"`
	SymbolName            string                  `json:"symbol_name"`
	SymbolYear            int64                   `json:"symbol_year"`
	ContractType          string                  `json:"contract_type"`
	Coin                  string                  `json:"coin"`
	QuoteSymbol           string                  `json:"quote_symbol"`
	Mode                  string                  `json:"mode"`
	IsUpBorrowable        int64                   `json:"is_up_borrowable"`
	ImportTime            bybitTimeNanoSec        `json:"import_time_e9"`
	StartTradingTime      bybitTimeNanoSec        `json:"start_trading_time_e9"`
	TimeToSettle          bybitTimeNanoSec        `json:"settle_time_e9"`
	SettleFeeRate         int64                   `json:"settle_fee_rate_e8"`
	ContractStatus        string                  `json:"contract_status"`
	SystemSubsidy         int64                   `json:"system_subsidy_e8"`
	LastPrice             convert.StringToFloat64 `json:"last_price"`
	BidPrice              float64                 `json:"bid1_price"`
	AskPrice              float64                 `json:"ask1_price"`
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
	FairBasis             float64                 `json:"fair_basis_e8"`
	FairBasisRate         float64                 `json:"fair_basis_rate_e8"`
	BasisInYear           float64                 `json:"basis_in_year_e8"`
	ExpectPrice           convert.StringToFloat64 `json:"expect_price"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdateAt              time.Time               `json:"updated_at"`
}

// WsFuturesTicker stores ws future ticker
type WsFuturesTicker struct {
	Topic  string              `json:"topic"`
	Ticker WsFuturesTickerData `json:"data"`
}

// WsDeltaFuturesTicker stores ws delta future ticker
type WsDeltaFuturesTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"string"`
	Data  struct {
		Delete []WsFuturesTickerData `json:"delete"`
		Update []WsFuturesTickerData `json:"update"`
		Insert []WsFuturesTickerData `json:"insert"`
	} `json:"data"`
}

// WsLiquidationData stores ws liquidation data
type WsLiquidationData struct {
	Symbol    string                  `json:"symbol"`
	Side      string                  `json:"side"`
	Price     convert.StringToFloat64 `json:"price"`
	Qty       float64                 `json:"qty"`
	Timestamp bybitTimeMilliSec       `json:"time"`
}

// WsFuturesLiquidation stores ws future liquidation
type WsFuturesLiquidation struct {
	Topic string            `json:"topic"`
	Data  WsLiquidationData `json:"data"`
}

// WsFuturesPositionData stores ws future position data
type WsFuturesPositionData struct {
	UserID              int64                   `json:"user_id"`
	Symbol              string                  `json:"symbol"`
	Side                string                  `json:"side"`
	Size                float64                 `json:"size"`
	PositionID          int64                   `json:"position_idx"` // present in Futures position struct only
	Mode                int64                   `json:"mode"`         // present in Futures position struct only
	Isolated            bool                    `json:"isolated"`     // present in Futures position struct only
	PositionValue       convert.StringToFloat64 `json:"position_value"`
	EntryPrice          convert.StringToFloat64 `json:"entry_price"`
	LiquidPrice         convert.StringToFloat64 `json:"liq_price"`
	BustPrice           convert.StringToFloat64 `json:"bust_price"`
	Leverage            convert.StringToFloat64 `json:"leverage"`
	OrderMargin         convert.StringToFloat64 `json:"order_margin"`
	PositionMargin      convert.StringToFloat64 `json:"position_margin"`
	AvailableBalance    convert.StringToFloat64 `json:"available_balance"`
	TakeProfit          convert.StringToFloat64 `json:"take_profit"`
	TakeProfitTriggerBy string                  `json:"tp_trigger_by"`
	StopLoss            convert.StringToFloat64 `json:"stop_loss"`
	StopLossTriggerBy   string                  `json:"sl_trigger_by"`
	RealisedPNL         convert.StringToFloat64 `json:"realised_pnl"`
	TrailingStop        convert.StringToFloat64 `json:"trailing_stop"`
	TrailingActive      convert.StringToFloat64 `json:"trailing_active"`
	WalletBalance       convert.StringToFloat64 `json:"wallet_balance"`
	RiskID              int64                   `json:"risk_id"`
	ClosingFee          convert.StringToFloat64 `json:"occ_closing_fee"`
	FundingFee          convert.StringToFloat64 `json:"occ_funding_fee"`
	AutoAddMargin       int64                   `json:"auto_add_margin"`
	TotalPNL            convert.StringToFloat64 `json:"cum_realised_pnl"`
	Status              string                  `json:"position_status"`
	Version             int64                   `json:"position_seq"`
}

// WsFuturesPosition stores ws future position
type WsFuturesPosition struct {
	Topic  string                  `json:"topic"`
	Action string                  `json:"action"`
	Data   []WsFuturesPositionData `json:"data"`
}

// WsFuturesExecutionData stores ws future execution data
type WsFuturesExecutionData struct {
	Symbol        string                  `json:"symbol"`
	Side          string                  `json:"side"`
	OrderID       string                  `json:"order_id"`
	ExecutionID   string                  `json:"exec_id"`
	OrderLinkID   string                  `json:"order_link_id"`
	Price         convert.StringToFloat64 `json:"price"`
	OrderQty      float64                 `json:"order_qty"`
	ExecutionType string                  `json:"exec_type"`
	ExecutionQty  float64                 `json:"exec_qty"`
	ExecutionFee  convert.StringToFloat64 `json:"exec_fee"`
	LeavesQty     float64                 `json:"leaves_qty"`
	IsMaker       bool                    `json:"is_maker"`
	Time          time.Time               `json:"trade_time"`
}

// WsFuturesExecution stores ws future execution
type WsFuturesExecution struct {
	Topic string                   `json:"topic"`
	Data  []WsFuturesExecutionData `json:"data"`
}

// WsOrderData stores ws order data
type WsOrderData struct {
	OrderID              string                  `json:"order_id"`
	OrderLinkID          string                  `json:"order_link_id"`
	Symbol               string                  `json:"symbol"`
	Side                 string                  `json:"side"`
	OrderType            string                  `json:"order_type"`
	Price                convert.StringToFloat64 `json:"price"`
	OrderQty             float64                 `json:"qty"`
	TimeInForce          string                  `json:"time_in_force"`
	CreateType           string                  `json:"create_type"`
	CancelType           string                  `json:"cancel_type"`
	OrderStatus          string                  `json:"order_status"`
	LeavesQty            float64                 `json:"leaves_qty"`
	CummulativeExecQty   float64                 `json:"cum_exec_qty"`
	CummulativeExecValue convert.StringToFloat64 `json:"cum_exec_value"`
	CummulativeExecFee   convert.StringToFloat64 `json:"cum_exec_fee"`
	TakeProfit           convert.StringToFloat64 `json:"take_profit"`
	StopLoss             convert.StringToFloat64 `json:"stop_loss"`
	TrailingStop         convert.StringToFloat64 `json:"trailing_stop"`
	TrailingActive       convert.StringToFloat64 `json:"trailing_active"`
	LastExecPrice        convert.StringToFloat64 `json:"last_exec_price"`
	ReduceOnly           bool                    `json:"reduce_only"`
	CloseOnTrigger       bool                    `json:"close_on_trigger"`
	Time                 time.Time               `json:"timestamp"`   // present in CoinMarginedFutures and Futures only
	CreateTime           time.Time               `json:"create_time"` // present in USDTMarginedFutures only
	UpdateTime           time.Time               `json:"update_time"` // present in USDTMarginedFutures only
}

// WsOrder stores ws order
type WsOrder struct {
	Topic string        `json:"topic"`
	Data  []WsOrderData `json:"data"`
}

// WsStopOrderData stores ws stop order data
type WsStopOrderData struct {
	OrderID        string                  `json:"order_id"`
	OrderLinkID    string                  `json:"order_link_id"`
	UserID         int64                   `json:"user_id"`
	Symbol         string                  `json:"symbol"`
	Side           string                  `json:"side"`
	OrderType      string                  `json:"order_type"`
	Price          convert.StringToFloat64 `json:"price"`
	OrderQty       float64                 `json:"qty"`
	TimeInForce    string                  `json:"time_in_force"`
	CreateType     string                  `json:"create_type"`
	CancelType     string                  `json:"cancel_type"`
	OrderStatus    string                  `json:"order_status"`
	StopOrderType  string                  `json:"stop_order_type"`
	TriggerBy      string                  `json:"trigger_by"`
	TriggerPrice   convert.StringToFloat64 `json:"trigger_price"`
	Time           time.Time               `json:"timestamp"`
	CloseOnTrigger bool                    `json:"close_on_trigger"`
}

// WsFuturesStopOrder stores ws future stop order
type WsFuturesStopOrder struct {
	Topic string            `json:"topic"`
	Data  []WsStopOrderData `json:"data"`
}

// WsUSDTStopOrderData stores ws USDT stop order data
type WsUSDTStopOrderData struct {
	OrderID        string                  `json:"stop_order_id"`
	OrderLinkID    string                  `json:"order_link_id"`
	UserID         int64                   `json:"user_id"`
	Symbol         string                  `json:"symbol"`
	Side           string                  `json:"side"`
	OrderType      string                  `json:"order_type"`
	Price          convert.StringToFloat64 `json:"price"`
	OrderQty       float64                 `json:"qty"`
	TimeInForce    string                  `json:"time_in_force"`
	OrderStatus    string                  `json:"order_status"`
	StopOrderType  string                  `json:"stop_order_type"`
	TriggerBy      string                  `json:"trigger_by"`
	TriggerPrice   convert.StringToFloat64 `json:"trigger_price"`
	ReduceOnly     bool                    `json:"reduce_only"`
	CloseOnTrigger bool                    `json:"close_on_trigger"`
	CreateTime     time.Time               `json:"create_time"`
	UpdateTime     time.Time               `json:"update_time"`
}

// WsUSDTFuturesStopOrder stores ws USDT stop order
type WsUSDTFuturesStopOrder struct {
	Topic string                `json:"topic"`
	Data  []WsUSDTStopOrderData `json:"data"`
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

// Ticker holds ticker information
type Ticker struct {
	// Spot fields
	Symbol            string                  `json:"symbol"`
	TopBidPrice       convert.StringToFloat64 `json:"bid1Price"`
	TopBidSize        convert.StringToFloat64 `json:"bid1Size"`
	TopAskPrice       convert.StringToFloat64 `json:"ask1Price"`
	TopAskSize        convert.StringToFloat64 `json:"ask1Size"`
	LastPrice         convert.StringToFloat64 `json:"lastPrice"`
	PreviousPrice24Hr convert.StringToFloat64 `json:"prevPrice24h"`
	Price24HrPcnt     convert.StringToFloat64 `json:"price24hPcnt"`
	HighPrice24Hr     convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24Hr      convert.StringToFloat64 `json:"lowPrice24h"`
	Turnover24Hr      convert.StringToFloat64 `json:"turnover24h"`
	Volume24Hr        convert.StringToFloat64 `json:"volume24h"`
	USDIndexPrice     convert.StringToFloat64 `json:"usdIndexPrice"`

	// Option fields
	TopBidImpliedVolatility convert.StringToFloat64 `json:"bid1Iv"`
	TopAskImpliedVolatility convert.StringToFloat64 `json:"ask1Iv"`
	MarkPrice               convert.StringToFloat64 `json:"markPrice"`
	IndexPrice              convert.StringToFloat64 `json:"indexPrice"`
	MarkImpliedVolatility   convert.StringToFloat64 `json:"markIv"`
	UnderlyingPrice         convert.StringToFloat64 `json:"underlyingPrice"`
	OpenInterest            convert.StringToFloat64 `json:"openInterest"`
	TotalVolume             convert.StringToFloat64 `json:"totalVolume"`
	TotalTurnover           convert.StringToFloat64 `json:"totalTurnover"`
	Delta                   convert.StringToFloat64 `json:"delta"`
	Gamma                   convert.StringToFloat64 `json:"gamma"`
	Vega                    convert.StringToFloat64 `json:"vega"`
	Theta                   convert.StringToFloat64 `json:"theta"`
	PredictedDeliveryPrice  convert.StringToFloat64 `json:"predictedDeliveryPrice"`
	Change24h               convert.StringToFloat64 `json:"change24h"`

	// Inverse/linear  fields
	PrevPrice1h       convert.StringToFloat64 `json:"prevPrice1h"`
	OpenInterestValue convert.StringToFloat64 `json:"openInterestValue"`
	FundingRate       convert.StringToFloat64 `json:"fundingRate"`
	NextFundingTime   convert.StringToFloat64 `json:"nextFundingTime"`
	BasisRate         convert.StringToFloat64 `json:"basisRate"`
	DeliveryFeeRate   convert.StringToFloat64 `json:"deliveryFeeRate"`
	DeliveryTime      convert.StringToFloat64 `json:"deliveryTime"`
	Basis             convert.StringToFloat64 `json:"basis"`
}

// Fee holds fee information
type Fee struct {
	BaseCoin string                  `json:"baseCoin"`
	Symbol   string                  `json:"symbol"`
	Taker    convert.StringToFloat64 `json:"takerFeeRate"`
	Maker    convert.StringToFloat64 `json:"makerFeeRate"`
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
