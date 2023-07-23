package bybit

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	errTypeAssert                 = errors.New("type assertion failed")
	errStrParsing                 = errors.New("parsing string failed")
	errInvalidSide                = errors.New("invalid side")
	errInvalidInterval            = errors.New("invalid interval")
	errInvalidPeriod              = errors.New("invalid period")
	errInvalidStartTime           = errors.New("startTime can't be zero or missing")
	errInvalidQuantity            = errors.New("quantity can't be zero or missing")
	errInvalidBasePrice           = errors.New("basePrice can't be empty or missing")
	errInvalidStopPrice           = errors.New("stopPrice can't be empty or missing")
	errInvalidTimeInForce         = errors.New("timeInForce can't be empty or missing")
	errInvalidTakeProfitStopLoss  = errors.New("takeProfitStopLoss can't be empty or missing")
	errInvalidMargin              = errors.New("margin can't be empty")
	errInvalidLeverage            = errors.New("leverage can't be zero or less then it")
	errInvalidRiskID              = errors.New("riskID can't be zero or lesser")
	errInvalidPositionMode        = errors.New("position mode is invalid")
	errInvalidOrderType           = errors.New("orderType can't be empty or missing")
	errInvalidMode                = errors.New("mode can't be empty or missing")
	errInvalidBuyLeverage         = errors.New("buyLeverage can't be zero or less then it")
	errInvalidSellLeverage        = errors.New("sellLeverage can't be zero or less then it")
	errInvalidOrderRequest        = errors.New("order request param can't be nil")
	errInvalidOrderFilter         = errors.New("invalid order filter")
	errInvalidCategory            = errors.New("invalid category")
	errEitherSymbolOrCoinRequired = errors.New("either symbol or coin required")
	errInvalidCoin                = errors.New("coin can't be empty")

	errStopOrderOrOrderLinkIDMissing = errors.New("at least one should be present among stopOrderID and orderLinkID")
	errOrderOrOrderLinkIDMissing     = errors.New("at least one should be present among orderID and orderLinkID")

	errOrderLinkIDMissing = errors.New("order link id missing")

	errSymbolMissing              = errors.New("symbol missing")
	errInvalidAutoAddMarginValue  = errors.New("invalid add auto margin value")
	errUnsupportedOrderType       = errors.New("unsupported order type")
	errEmptyOrderIDs              = errors.New("orderIDs can't be empty")
	errMissingPrice               = errors.New("price should be present for Limit and LimitMaker orders")
	errExpectedOneOrder           = errors.New("expected one order")
	errDisconnectTimeWindowNotSet = errors.New("disconnect time window not set")
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
	UpdateID       int64
	Bids           []orderbook.Item
	Asks           []orderbook.Item
	Symbol         string
	GenerationTime time.Time
}

// TradeItem stores a single trade
type TradeItem struct {
	CurrencyPair string
	Price        float64
	Side         string
	Volume       float64
	Time         time.Time
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

// // TickerData stores ticker data
// type TickerData struct {
// 	Symbol      string                  `json:"symbol"`
// 	BidPrice    convert.StringToFloat64 `json:"bidPrice"`
// 	BidQuantity convert.StringToFloat64 `json:"bidQty"`
// 	AskPrice    convert.StringToFloat64 `json:"askPrice"`
// 	AskQuantity convert.StringToFloat64 `json:"askQty"`
// 	Time        bybitTimeMilliSec       `json:"time"`
// }

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
	Symbol    string               `json:"s"`
	Asks      [][2]string          `json:"a"`
	Bids      [][2]string          `json:"b"`
	Timestamp convert.ExchangeTime `json:"ts"`
	UpdateID  int64                `json:"u"`
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

// ----------------------------------------------------------------------------

// InstrumentsInfo representa a category, page indicator, and list of instrument informations.
type InstrumentsInfo struct {
	Category       string           `json:"category"`
	List           []InstrumentInfo `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// InstrumentInfo represents detailed data for symbol.
type InstrumentInfo struct {
	Symbol          string                  `json:"symbol"`
	ContractType    string                  `json:"contractType"`
	Status          string                  `json:"status"`
	BaseCoin        string                  `json:"baseCoin"`
	QuoteCoin       string                  `json:"quoteCoin"`
	LaunchTime      convert.ExchangeTime    `json:"launchTime"`
	DeliveryTime    convert.ExchangeTime    `json:"deliveryTime"`
	DeliveryFeeRate convert.StringToFloat64 `json:"deliveryFeeRate"`
	PriceScale      convert.StringToFloat64 `json:"priceScale"`
	LeverageFilter  struct {
		MinLeverage  convert.StringToFloat64 `json:"minLeverage"`
		MaxLeverage  convert.StringToFloat64 `json:"maxLeverage"`
		LeverageStep convert.StringToFloat64 `json:"leverageStep"`
	} `json:"leverageFilter"`
	PriceFilter struct {
		MinPrice convert.StringToFloat64 `json:"minPrice"`
		MaxPrice convert.StringToFloat64 `json:"maxPrice"`
		TickSize convert.StringToFloat64 `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MaxOrderQty         convert.StringToFloat64 `json:"maxOrderQty"`
		MinOrderQty         convert.StringToFloat64 `json:"minOrderQty"`
		QtyStep             convert.StringToFloat64 `json:"qtyStep"`
		PostOnlyMaxOrderQty convert.StringToFloat64 `json:"postOnlyMaxOrderQty"`
	} `json:"lotSizeFilter"`
	UnifiedMarginTrade bool   `json:"unifiedMarginTrade"`
	FundingInterval    int64  `json:"fundingInterval"`
	SettleCoin         string `json:"settleCoin"`
}

// RestResponse represents a REST response instance.
type RestResponse struct {
	RetCode    int64                `json:"retCode"`
	RetMsg     string               `json:"retMsg"`
	Result     interface{}          `json:"result"`
	RetExtInfo json.RawMessage      `json:"retExtInfo"`
	Time       convert.ExchangeTime `json:"time"`
}

// KlineResponse represents a kline item list instance as an array of string.
type KlineResponse struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}

// KlineDatas represents a kline item list instance as an array of KlineItem.
type KlineDatas struct {
	Symbol   string      `json:"symbol"`
	Category string      `json:"category"`
	List     []KlineItem `json:"list"`
}

// KlineItem stores an individual kline data item
type KlineItem struct {
	StartTime time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64

	// not available for mark and index price kline data
	TradeVolume float64
	Turnover    float64
}

// MarkPriceKlineResponse represents a kline data item.
type MarkPriceKlineResponse struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}

// MarkPriceKlineItem represents a mark price kline item instance.
type MarkPriceKlineItem struct {
	StartTime  time.Time
	OpenPrice  float64
	HighPrice  float64
	LowPrice   float64
	ClosePrice float64
}

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	var (
		s = Orderbook{
			Symbol:         o.Symbol,
			UpdateID:       o.UpdateID,
			GenerationTime: o.Timestamp.Time(),
		}
		err error
	)
	s.Bids, err = processOB(o.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks, err = processOB(o.Asks)
	if err != nil {
		return nil, err
	}
	return &s, err
}

// TickerData represents a list of ticker detailed information.
type TickerData struct {
	Category string       `json:"category"`
	List     []TickerItem `json:"list"`
}

// TickerItem represents a ticker item detail.
type TickerItem struct {
	Symbol                 string                  `json:"symbol"`
	LastPrice              convert.StringToFloat64 `json:"lastPrice"`
	IndexPrice             convert.StringToFloat64 `json:"indexPrice"`
	MarkPrice              convert.StringToFloat64 `json:"markPrice"`
	PrevPrice24H           convert.StringToFloat64 `json:"prevPrice24h"`
	Price24HPcnt           convert.StringToFloat64 `json:"price24hPcnt"`
	HighPrice24H           convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H            convert.StringToFloat64 `json:"lowPrice24h"`
	PrevPrice1H            convert.StringToFloat64 `json:"prevPrice1h"`
	OpenInterest           convert.StringToFloat64 `json:"openInterest"`
	OpenInterestValue      convert.StringToFloat64 `json:"openInterestValue"`
	Turnover24H            convert.StringToFloat64 `json:"turnover24h"`
	Volume24H              convert.StringToFloat64 `json:"volume24h"`
	FundingRate            convert.StringToFloat64 `json:"fundingRate"`
	NextFundingTime        convert.StringToFloat64 `json:"nextFundingTime"`
	PredictedDeliveryPrice convert.StringToFloat64 `json:"predictedDeliveryPrice"`
	BasisRate              convert.StringToFloat64 `json:"basisRate"`
	DeliveryFeeRate        convert.StringToFloat64 `json:"deliveryFeeRate"`
	DeliveryTime           convert.ExchangeTime    `json:"deliveryTime"`
	Ask1Size               convert.StringToFloat64 `json:"ask1Size"`
	Bid1Price              convert.StringToFloat64 `json:"bid1Price"`
	Ask1Price              convert.StringToFloat64 `json:"ask1Price"`
	Bid1Size               convert.StringToFloat64 `json:"bid1Size"`
	Basis                  convert.StringToFloat64 `json:"basis"`
}

// FundingRateHistory represents a funding rate history for a category.
type FundingRateHistory struct {
	Category string        `json:"category"`
	List     []FundingRate `json:"list"`
}

// FundingRate represents a funding rate instance.
type FundingRate struct {
	Symbol               string                  `json:"symbol"`
	FundingRate          convert.StringToFloat64 `json:"fundingRate"`
	FundingRateTimestamp convert.ExchangeTime    `json:"fundingRateTimestamp"`
}

// TradingHistory represents a trading history list.
type TradingHistory struct {
	Category string               `json:"category"`
	List     []TradingHistoryItem `json:"list"`
}

// TradingHistoryItem represents a trading history item instance.
type TradingHistoryItem struct {
	ExecutionID  string                  `json:"execId"`
	Symbol       string                  `json:"symbol"`
	Price        convert.StringToFloat64 `json:"price"`
	Size         convert.StringToFloat64 `json:"size"`
	Side         string                  `json:"side"`
	TradeTime    convert.ExchangeTime    `json:"time"`
	IsBlockTrade bool                    `json:"isBlockTrade"`
}

// OpenInterest represents open interest of each symbol.
type OpenInterest struct {
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
	List     []struct {
		OpenInterest convert.StringToFloat64 `json:"openInterest"`
		Timestamp    convert.ExchangeTime    `json:"timestamp"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// HistoricVolatility represents option historical volatility
type HistoricVolatility struct {
	Period int64                   `json:"period"`
	Value  convert.StringToFloat64 `json:"value"`
	Time   convert.ExchangeTime    `json:"time"`
}

// InsuranceHistory represents an insurance list.
type InsuranceHistory struct {
	UpdatedTime convert.ExchangeTime `json:"updatedTime"`
	List        []struct {
		Coin    string                  `json:"coin"`
		Balance convert.StringToFloat64 `json:"balance"`
		Value   convert.StringToFloat64 `json:"value"`
	} `json:"list"`
}

// RiskLimitHistory represents risk limit history of a category.
type RiskLimitHistory struct {
	Category string `json:"category"`
	List     []struct {
		ID                int64                   `json:"id"`
		Symbol            string                  `json:"symbol"`
		RiskLimitValue    convert.StringToFloat64 `json:"riskLimitValue"`
		MaintenanceMargin convert.StringToFloat64 `json:"maintenanceMargin"`
		InitialMargin     convert.StringToFloat64 `json:"initialMargin"`
		IsLowestRisk      int64                   `json:"isLowestRisk"`
		MaxLeverage       convert.StringToFloat64 `json:"maxLeverage"`
	} `json:"list"`
}

// DeliveryPrice represents the delivery price information.
type DeliveryPrice struct {
	Category       string `json:"category"`
	NextPageCursor string `json:"nextPageCursor"`
	List           []struct {
		Symbol        string                  `json:"symbol"`
		DeliveryPrice convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryTime  convert.ExchangeTime    `json:"deliveryTime"`
	} `json:"list"`
}

// PlaceOrderParams represents
type PlaceOrderParams struct {
	Category               string        `json:"category"`
	Symbol                 currency.Pair `json:"symbol"`
	Side                   string        `json:"side"`
	OrderType              string        `json:"orderType"`  // Market, Limit
	OrderQuantity          float64       `json:"qty,string"` // Order quantity. For Spot Market Buy order, please note that qty should be quote curreny amount
	Price                  float64       `json:"price,string,omitempty"`
	TimeInForce            string        `json:"timeInForce,omitempty"`      // IOC and GTC
	OrderLinkID            string        `json:"orderLinkId,omitempty"`      // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	WhetherToBorrow        bool          `json:"-"`                          // '0' for default spot, '1' for Margin trading.
	IsLeverage             int64         `json:"isLeverage,omitempty"`       // '0' for default spot, '1' for Margin trading.
	OrderFilter            string        `json:"orderFilter,omitempty"`      // Valid for spot only. Order,tpslOrder. If not passed, Order by default
	TriggerDirection       int64         `json:"triggerDirection,omitempty"` // Conditional order param. Used to identify the expected direction of the conditional order. '1': triggered when market price rises to triggerPrice '2': triggered when market price falls to triggerPrice
	TriggerPrice           float64       `json:"triggerPrice,omitempty,string"`
	TriggerPriceType       string        `json:"triggerBy,omitempty"` // Conditional order param. Trigger price type. 'LastPrice', 'IndexPrice', 'MarkPrice'
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	PositionIdx            int64         `json:"positionIdx,omitempty"` // Under hedge-mode, this param is required '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
	ReduceOnly             bool          `json:"reduceOnly,omitempty"`
	TakeProfitPrice        float64       `json:"takeProfit,omitempty,string"`
	TakeProfitTriggerBy    string        `json:"tpTriggerBy,omitempty"` // The price type to trigger take profit. 'MarkPrice', 'IndexPrice', default: 'LastPrice'
	StopLossTriggerBy      string        `json:"slTriggerBy,omitempty"` // The price type to trigger stop loss. MarkPrice, IndexPrice, default: LastPrice
	StopLossPrice          float64       `json:"stopLoss,omitempty,string"`
	CloseOnTrigger         bool          `json:"closeOnTrigger,omitempty"`
	SMPExecutionType       string        `json:"smpType,omitempty"` // default: 'None', 'CancelMaker', 'CancelTaker', 'CancelBoth'
	MarketMakerProtection  bool          `json:"mmp,omitempty"`     // option only. true means set the order as a market maker protection order.

	// TP/SL mode
	// "Full": entire position for TP/SL. Then, tpOrderType or slOrderType must be Market
	// 'Partial': partial position tp/sl. Limit TP/SL order are supported. Note: When create limit tp/sl, tpslMode is required and it must be Partial
	TpslMode     string  `json:"tpslMode,omitempty"`
	TpOrderType  string  `json:"tpOrderType,omitempty"`
	SlOrderType  string  `json:"slOrderType,omitempty"`
	TpLimitPrice float64 `json:"tpLimitPrice,omitempty,string"`
	SlLimitPrice float64 `json:"slLimitPrice,omitempty,string"`
}

// OrderResponse holds newly placed order information.
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// AmendOrderParams represents a parameter for amending order.
type AmendOrderParams struct {
	Category               string        `json:"category"`
	Symbol                 currency.Pair `json:"symbol"`
	OrderID                string        `json:"orderId,omitempty"`
	OrderLinkID            string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	TriggerPrice           float64       `json:"triggerPrice,omitempty,string"`
	OrderQuantity          float64       `json:"qty,omitempty,string"` // Order quantity. For Spot Market Buy order, please note that qty should be quote curreny amount
	Price                  float64       `json:"price,string,omitempty"`

	TakeProfitPrice float64 `json:"takeProfit,omitempty,string"`
	StopLossPrice   float64 `json:"stopLoss,omitempty,string"`

	TakeProfitTriggerBy string `json:"tpTriggerBy,omitempty"` // The price type to trigger take profit. 'MarkPrice', 'IndexPrice', default: 'LastPrice'
	StopLossTriggerBy   string `json:"slTriggerBy,omitempty"` // The price type to trigger stop loss. MarkPrice, IndexPrice, default: LastPrice
	TriggerPriceType    string `json:"triggerBy,omitempty"`   // Conditional order param. Trigger price type. 'LastPrice', 'IndexPrice', 'MarkPrice'

	TpLimitPrice float64 `json:"tpLimitPrice,omitempty,string"`
	SlLimitPrice float64 `json:"slLimitPrice,omitempty,string"`
}

// CancelOrderParams represents a cancel order parameters.
type CancelOrderParams struct {
	Category    string        `json:"category,omitempty"`
	Symbol      currency.Pair `json:"symbol,omitempty"`
	OrderID     string        `json:"orderId,omitempty"`
	OrderLinkID string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:

	OrderFilter string `json:"orderFilter,omitempty"` // Valid for spot only. Order,tpslOrder. If not passed, Order by default
}

// TradeOrders represents category and list of trade orders of the category.
type TradeOrders struct {
	List           []TradeOrder `json:"list"`
	NextPageCursor string       `json:"nextPageCursor"`
	Category       string       `json:"category"`
}

// TradeOrder represents a trade order details.
type TradeOrder struct {
	OrderID                string                  `json:"orderId"`
	OrderLinkID            string                  `json:"orderLinkId"`
	BlockTradeID           string                  `json:"blockTradeId"`
	Symbol                 string                  `json:"symbol"`
	Price                  convert.StringToFloat64 `json:"price"`
	OrderQuantity          convert.StringToFloat64 `json:"qty"`
	Side                   string                  `json:"side"`
	IsLeverage             string                  `json:"isLeverage"`
	PositionIdx            int                     `json:"positionIdx"`
	OrderStatus            string                  `json:"orderStatus"`
	CancelType             string                  `json:"cancelType"`
	RejectReason           string                  `json:"rejectReason"`
	AveragePrice           convert.StringToFloat64 `json:"avgPrice"`
	LeavesQuantity         convert.StringToFloat64 `json:"leavesQty"`
	LeavesValue            string                  `json:"leavesValue"`
	CumulativeExecQuantity convert.StringToFloat64 `json:"cumExecQty"`
	CumulativeExecValue    convert.StringToFloat64 `json:"cumExecValue"`
	CumulativeExecFee      convert.StringToFloat64 `json:"cumExecFee"`
	TimeInForce            string                  `json:"timeInForce"`
	OrderType              string                  `json:"orderType"`
	StopOrderType          string                  `json:"stopOrderType"`
	OrderIv                string                  `json:"orderIv"`
	TriggerPrice           convert.StringToFloat64 `json:"triggerPrice"`
	TakeProfitPrice        convert.StringToFloat64 `json:"takeProfit"`
	StopLossPrice          convert.StringToFloat64 `json:"stopLoss"`
	TpTriggerBy            string                  `json:"tpTriggerBy"`
	SlTriggerBy            string                  `json:"slTriggerBy"`
	TriggerDirection       int                     `json:"triggerDirection"`
	TriggerBy              string                  `json:"triggerBy"`
	LastPriceOnCreated     string                  `json:"lastPriceOnCreated"`
	ReduceOnly             bool                    `json:"reduceOnly"`
	CloseOnTrigger         bool                    `json:"closeOnTrigger"`
	SmpType                string                  `json:"smpType"`
	SmpGroup               int                     `json:"smpGroup"`
	SmpOrderID             string                  `json:"smpOrderId"`
	TpslMode               string                  `json:"tpslMode"`
	TpLimitPrice           convert.StringToFloat64 `json:"tpLimitPrice"`
	SlLimitPrice           convert.StringToFloat64 `json:"slLimitPrice"`
	PlaceType              string                  `json:"placeType"`
	CreatedTime            convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime            convert.ExchangeTime    `json:"updatedTime"`
}

// CancelAllResponse represents a cancel all trade orders response.
type CancelAllResponse struct {
	List []OrderResponse `json:"list"`
}

// CancelAllOrdersParam request parameters for cancel all orders.
type CancelAllOrdersParam struct {
	Category    string        `json:"category"`
	Symbol      currency.Pair `json:"symbol"`
	OrderFilter string        `json:"orderFilter,omitempty"` // Valid for spot only. Order,tpslOrder. If not passed, Order by default
	BaseCoin    string        `json:"baseCoin,omitempty"`
	SettleCoin  string        `json:"settleCoin,omitempty"`
}

type PlaceBatchOrderParam struct {
	Category string                `json:"category"`
	Request  []BatchOrderItemParam `json:"request"`
}

// BatchOrderItemParam represents a batch order place parameter.
type BatchOrderItemParam struct {
	Category          string        `json:"category,omitempty"`
	Symbol            currency.Pair `json:"symbol,omitempty"`
	OrderType         string        `json:"orderType,omitempty"`
	Side              string        `json:"side,omitempty"`
	OrderQuantity     float64       `json:"qty,string,omitempty"`
	Price             float64       `json:"price,string,omitempty"`
	OrderIv           int64         `json:"orderIv,omitempty,string"`
	TimeInForce       string        `json:"timeInForce,omitempty"`
	OrderLinkID       string        `json:"orderLinkId,omitempty"`
	Mmp               bool          `json:"mmp,omitempty"`
	ReduceOnly        bool          `json:"reduceOnly,omitempty"`
	ImpliedVolatility string        `json:"iv,omitempty"`
	SMPType           string        `json:"smpType,omitempty"`
}

// BatchOrdersList represents a list trade orders.
type BatchOrdersList struct {
	List []BatchOrderResponse `json:"list"`
}

// BatchOrderResponse represents a batch trade order item response.
type BatchOrderResponse struct {
	Category    string               `json:"category"`
	Symbol      string               `json:"symbol"`
	OrderID     string               `json:"orderId"`
	OrderLinkID string               `json:"orderLinkId"`
	CreateAt    convert.ExchangeTime `json:"createAt"`
}

type errorMessages struct {
	List []ErrorMessage `json:"list"`
}

// ErrorMessage represents an error message item
type ErrorMessage struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// BatchAmendOrderParams request parameter for batch amend order.
type BatchAmendOrderParams struct {
	Category string                     `json:"category"`
	Request  []BatchAmendOrderParamItem `json:"request"`
}

// BatchAmendOrderParamItem represents a single order amend item in batch amend order
type BatchAmendOrderParamItem struct {
	Symbol                 currency.Pair `json:"symbol"`
	OrderID                string        `json:"orderId,omitempty"`
	OrderLinkID            string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	OrderQuantity          float64       `json:"qty,omitempty,string"` // Order quantity. For Spot Market Buy order, please note that qty should be quote curreny amount
	Price                  float64       `json:"price,string,omitempty"`
}

// CancelBatchOrder represents a batch cancel request parameters.
type CancelBatchOrder struct {
	Category string              `json:"category"`
	Request  []CancelOrderParams `json:"request"`
}

// CancelBatchResponseItem represents a batch cancel response item.
type CancelBatchResponseItem struct {
	Category    string `json:"category,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	OrderLinkID string `json:"orderLinkId,omitempty"`
}

type cancelBatchResponse struct {
	List []CancelBatchResponseItem `json:"list"`
}

// BorrowQuota represents
type BorrowQuota struct {
	Symbol         string `json:"symbol"`
	MaxTradeQty    string `json:"maxTradeQty"`
	Side           string `json:"side"`
	MaxTradeAmount string `json:"maxTradeAmount"`
	BorrowCoin     string `json:"borrowCoin"`
}

// SetDCPParams represents the set disconnect cancel all parameters.
type SetDCPParams struct {
	TimeWindow int64 `json:"timeWindow"`
}

// PositionInfoList represents a list of positions infos.
type PositionInfoList struct {
	Category       string         `json:"category"`
	NextPageCursor string         `json:"nextPageCursor"`
	List           []PositionInfo `json:"list"`
}

// PositionInfo represents a position info item.
type PositionInfo struct {
	PositionIndex    int64                   `json:"positionIdx"`
	RiskID           int64                   `json:"riskId"`
	RiskLimitValue   string                  `json:"riskLimitValue"`
	Symbol           string                  `json:"symbol"`
	Side             string                  `json:"side"`
	Size             convert.StringToFloat64 `json:"size"`
	AveragePrice     convert.StringToFloat64 `json:"avgPrice"`
	PositionValue    convert.StringToFloat64 `json:"positionValue"`
	TradeMode        int64                   `json:"tradeMode"`
	PositionStatus   string                  `json:"positionStatus"`
	AutoAddMargin    int64                   `json:"autoAddMargin"`
	AdlRankIndicator int64                   `json:"adlRankIndicator"`
	Leverage         convert.StringToFloat64 `json:"leverage"`
	PositionBalance  convert.StringToFloat64 `json:"positionBalance"`
	MarkPrice        convert.StringToFloat64 `json:"markPrice"`
	LiqPrice         convert.StringToFloat64 `json:"liqPrice"`
	BustPrice        convert.StringToFloat64 `json:"bustPrice"`
	PositionMM       convert.StringToFloat64 `json:"positionMM"`
	PositionIM       convert.StringToFloat64 `json:"positionIM"`
	TpslMode         string                  `json:"tpslMode"`
	TakeProfit       convert.StringToFloat64 `json:"takeProfit"`
	StopLoss         convert.StringToFloat64 `json:"stopLoss"`
	TrailingStop     convert.StringToFloat64 `json:"trailingStop"`
	UnrealisedPnl    convert.StringToFloat64 `json:"unrealisedPnl"`
	CumRealisedPnl   convert.StringToFloat64 `json:"cumRealisedPnl"`
	CreatedTime      convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime      convert.ExchangeTime    `json:"updatedTime"`
}

// SetLeverageParams parameters to set the leverage.
type SetLeverageParams struct {
	Category     string  `json:"category"`
	Symbol       string  `json:"symbol"`
	BuyLeverage  float64 `json:"buyLeverage,string"`  // [0, max leverage of corresponding risk limit]. Note: Under one-way mode, buyLeverage must be the same as sellLeverage
	SellLeverage float64 `json:"sellLeverage,string"` // [0, max leverage of corresponding risk limit]. Note: Under one-way mode, buyLeverage must be the same as sellLeverage
}

// SwitchTradeModeParams parameters to switch between cross margin and isolated margin trade mode.
type SwitchTradeModeParams struct {
	Category     string  `json:"category"`
	Symbol       string  `json:"symbol"`
	BuyLeverage  float64 `json:"buyLeverage,string"`
	SellLeverage float64 `json:"sellLeverage,string"`
	TradeMode    int64   `json:"tradeMode"` // 0: cross margin. 1: isolated margin
}

// TPSLModeParams paramaters for settle Take Profit(TP) or Stop Loss(SL) mode.
type TPSLModeParams struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	TpslMode string `json:"tpSlMode"` // TP/SL mode. Full,Partial
}

// TPSLModeResponse represents response for the take profit and stop loss mode change.
type TPSLModeResponse struct {
	TPSLMode string `json:"tpSlMode"`
}

// SwitchPositionModeParams represents a position switch mode parameters.
type SwitchPositionModeParams struct {
	Category     string        `json:"category"`
	Symbol       currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	Coin         currency.Code `json:"coin"`
	PositionMode int64         `json:"mode"` // Position mode. 0: Merged Single. 3: Both Sides
}

// SetRiskLimitParam represents a risk limit set parameter.
type SetRiskLimitParam struct {
	Category     string        `json:"category"`
	Symbol       currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	RiskID       int64         `json:"riskId"`
	PositionMode int64         `json:"positionIdx"` // Used to identify positions in different position modes. For hedge mode, it is required '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
}

// RiskLimitResponse represents a risk limit response.
type RiskLimitResponse struct {
	RiskID         int64  `json:"riskId"`
	RiskLimitValue string `json:"riskLimitValue"`
	Category       string `json:"category"`
}

// TradingStopParams take profit, stop loss or trailing stop for the position.
type TradingStopParams struct {
	Category                 string        `json:"category"`
	Symbol                   currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	TakeProfit               string        `json:"takeProfit,omitempty"`
	StopLoss                 string        `json:"stopLoss,omitempty"`
	TrailingStop             string        `json:"trailingStop,omitempty"`
	TakeProfitTriggerType    string        `json:"tpTriggerBy,omitempty"`
	StopLossTriggerType      string        `json:"slTriggerBy,omitempty"`
	ActivePrice              string        `json:"activePrice,omitempty"`
	TakeProfitOrStopLossMode string        `json:"tpslMode,omitempty"`
	TakeProfitOrderType      string        `json:"tpOrderType,omitempty"`
	StopLossOrderType        string        `json:"slOrderType,omitempty"`
	TakeProfitSize           float64       `json:"tpSize,string,omitempty"`
	StopLossSize             float64       `json:"slSize,string,omitempty"`
	TakeProfitLimitPrice     float64       `json:"tpLimitPrice,string,omitempty"`
	StopLossLimitPrice       float64       `json:"slLimitPrice,string,omitempty"`
	PositionIndex            int64         `json:"positionIdx,omitempty"`
}

// AddRemoveMarginParams represents parameters for auto add margin
type AddRemoveMarginParams struct {
	Category      string        `json:"category,omitempty"`
	Symbol        currency.Pair `json:"symbol,omitempty"`
	AutoAddmargin int64         `json:"autoAddmargin,string,omitempty"`
	PositionMode  int64         `json:"positionIdx,string,omitempty"`
}

// AddOrReduceMargin represents a add or reduce margin response.
type AddOrReduceMargin struct {
	Category                 string                  `json:"category"`
	Symbol                   string                  `json:"symbol"`
	PositionIndex            int64                   `json:"positionIdx"` // position mode index
	RiskID                   int64                   `json:"riskId"`
	RiskLimitValue           string                  `json:"riskLimitValue"`
	Size                     convert.StringToFloat64 `json:"size"`
	PositionValue            string                  `json:"positionValue"`
	AveragePrice             convert.StringToFloat64 `json:"avgPrice"`
	LiquidationPrice         convert.StringToFloat64 `json:"liqPrice"`
	BustPrice                convert.StringToFloat64 `json:"bustPrice"`
	MarkPrice                convert.StringToFloat64 `json:"markPrice"`
	Leverage                 string                  `json:"leverage"`
	AutoAddMargin            int64                   `json:"autoAddMargin"`
	PositionStatus           string                  `json:"positionStatus"`
	PositionIM               convert.StringToFloat64 `json:"positionIM"`
	PositionMM               convert.StringToFloat64 `json:"positionMM"`
	UnrealisedProfitAndLoss  convert.StringToFloat64 `json:"unrealisedPnl"`
	CumRealisedProfitAndLoss convert.StringToFloat64 `json:"cumRealisedPnl"`
	StopLoss                 convert.StringToFloat64 `json:"stopLoss"`
	TakeProfit               convert.StringToFloat64 `json:"takeProfit"`
	TrailingStop             convert.StringToFloat64 `json:"trailingStop"`
	CreatedTime              convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime              convert.ExchangeTime    `json:"updatedTime"`
}

// ExecutionResponse represents users order execution response
type ExecutionResponse struct {
	NextPageCursor string      `json:"nextPageCursor"`
	Category       string      `json:"category"`
	List           []Execution `json:"list"`
}

// Execution represents execution record
type Execution struct {
	Symbol                 string                  `json:"symbol"`
	OrderType              string                  `json:"orderType"`
	UnderlyingPrice        convert.StringToFloat64 `json:"underlyingPrice"`
	IndexPrice             convert.StringToFloat64 `json:"indexPrice"`
	OrderLinkID            string                  `json:"orderLinkId"`
	Side                   string                  `json:"side"`
	OrderID                string                  `json:"orderId"`
	StopOrderType          string                  `json:"stopOrderType"`
	LeavesQuantity         string                  `json:"leavesQty"`
	ExecTime               string                  `json:"execTime"`
	IsMaker                bool                    `json:"isMaker"`
	ExecFee                convert.StringToFloat64 `json:"execFee"`
	FeeRate                convert.StringToFloat64 `json:"feeRate"`
	ExecID                 string                  `json:"execId"`
	TradeImpliedVolatility string                  `json:"tradeIv"`
	BlockTradeID           string                  `json:"blockTradeId"`
	MarkPrice              convert.StringToFloat64 `json:"markPrice"`
	ExecPrice              convert.StringToFloat64 `json:"execPrice"`
	MarkIv                 string                  `json:"markIv"`
	OrderQuantity          string                  `json:"orderQty"`
	ExecValue              string                  `json:"execValue"`
	ExecType               string                  `json:"execType"`
	OrderPrice             convert.StringToFloat64 `json:"orderPrice"`
	ExecQuantity           convert.StringToFloat64 `json:"execQty"`
	ClosedSize             convert.StringToFloat64 `json:"closedSize"`
}

// ClosedProfitAndLossResponse represents list of closed profit and loss records
type ClosedProfitAndLossResponse struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol              string                  `json:"symbol"`
		OrderType           string                  `json:"orderType"`
		Leverage            string                  `json:"leverage"`
		UpdatedTime         convert.ExchangeTime    `json:"updatedTime"`
		Side                string                  `json:"side"`
		OrderID             string                  `json:"orderId"`
		ClosedPnl           string                  `json:"closedPnl"`
		AvgEntryPrice       convert.StringToFloat64 `json:"avgEntryPrice"`
		Quantity            convert.StringToFloat64 `json:"qty"`
		CumulatedEntryValue string                  `json:"cumEntryValue"`
		CreatedTime         convert.ExchangeTime    `json:"createdTime"`
		OrderPrice          convert.StringToFloat64 `json:"orderPrice"`
		ClosedSize          convert.StringToFloat64 `json:"closedSize"`
		AvgExitPrice        convert.StringToFloat64 `json:"avgExitPrice"`
		ExecutionType       string                  `json:"execType"`
		FillCount           convert.StringToFloat64 `json:"fillCount"`
		CumulatedExitValue  string                  `json:"cumExitValue"`
	} `json:"list"`
}

type paramsConfig struct {
	Spot            bool
	Option          bool
	Linear          bool
	Inverse         bool
	MendatorySymbol bool

	OptionalCategory bool
	OptionalBaseCoin bool
}

// TransactionLog represents a transaction log history.
type TransactionLog struct {
	NextPageCursor string               `json:"nextPageCursor"`
	List           []TransactionLogItem `json:"list"`
}

// TransactionLogItem represents a transaction log item information.
type TransactionLogItem struct {
	Symbol          string                  `json:"symbol"`
	Side            string                  `json:"side"`
	Funding         string                  `json:"funding"`
	OrderLinkID     string                  `json:"orderLinkId"`
	OrderID         string                  `json:"orderId"`
	Fee             convert.StringToFloat64 `json:"fee"`
	Change          string                  `json:"change"`
	CashFlow        string                  `json:"cashFlow"`
	TransactionTime convert.ExchangeTime    `json:"transactionTime"`
	Type            string                  `json:"type"`
	FeeRate         convert.StringToFloat64 `json:"feeRate"`
	BonusChange     convert.StringToFloat64 `json:"bonusChange"`
	Size            convert.StringToFloat64 `json:"size"`
	Qty             convert.StringToFloat64 `json:"qty"`
	CashBalance     convert.StringToFloat64 `json:"cashBalance"`
	Currency        string                  `json:"currency"`
	Category        string                  `json:"category"`
	TradePrice      convert.StringToFloat64 `json:"tradePrice"`
	TradeID         string                  `json:"tradeId"`
}

// PreUpdateOptionDeliveryRecord represents delivery records of Option
type PreUpdateOptionDeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol        string                  `json:"symbol"`
		Side          string                  `json:"side"`
		DeliveryTime  convert.ExchangeTime    `json:"deliveryTime"`
		ExercisePrice convert.StringToFloat64 `json:"strike"`
		Fee           convert.StringToFloat64 `json:"fee"`
		Position      string                  `json:"position"`
		DeliveryPrice convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryRpl   string                  `json:"deliveryRpl"` // Realized PnL of the delivery
	} `json:"list"`
}

// SettlementSession represents a USDC settlement session.
type SettlementSession struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		RealisedProfitAndLoss string                  `json:"realisedPnl"`
		Symbol                string                  `json:"symbol"`
		Side                  string                  `json:"side"`
		MarkPrice             convert.StringToFloat64 `json:"markPrice"`
		Size                  convert.StringToFloat64 `json:"size"`
		CreatedTime           convert.ExchangeTime    `json:"createdTime"`
		SessionAveragePrice   convert.StringToFloat64 `json:"sessionAvgPrice"`
	} `json:"list"`
}

// WalletBalance represents wallet balance
type WalletBalance struct {
	List []struct {
		TotalEquity            convert.StringToFloat64 `json:"totalEquity"`
		AccountIMRate          convert.StringToFloat64 `json:"accountIMRate"`
		TotalMarginBalance     convert.StringToFloat64 `json:"totalMarginBalance"`
		TotalInitialMargin     convert.StringToFloat64 `json:"totalInitialMargin"`
		AccountType            string                  `json:"accountType"`
		TotalAvailableBalance  convert.StringToFloat64 `json:"totalAvailableBalance"`
		AccountMMRate          convert.StringToFloat64 `json:"accountMMRate"`
		TotalPerpUPL           string                  `json:"totalPerpUPL"`
		TotalWalletBalance     convert.StringToFloat64 `json:"totalWalletBalance"`
		AccountLTV             string                  `json:"accountLTV"` // Account LTV: account total borrowed size / (account total equity + account total borrowed size).
		TotalMaintenanceMargin convert.StringToFloat64 `json:"totalMaintenanceMargin"`
		Coin                   []struct {
			AvailableToBorrow       convert.StringToFloat64 `json:"availableToBorrow"`
			Bonus                   convert.StringToFloat64 `json:"bonus"`
			AccruedInterest         string                  `json:"accruedInterest"`
			AvailableToWithdraw     convert.StringToFloat64 `json:"availableToWithdraw"`
			AvailableBalanceForSpot convert.StringToFloat64 `json:"free"`
			TotalOrderIM            string                  `json:"totalOrderIM"`
			Equity                  convert.StringToFloat64 `json:"equity"`
			TotalPositionMM         string                  `json:"totalPositionMM"`
			USDValue                convert.StringToFloat64 `json:"usdValue"`
			UnrealisedPnl           convert.StringToFloat64 `json:"unrealisedPnl"`
			BorrowAmount            convert.StringToFloat64 `json:"borrowAmount"`
			TotalPositionIM         string                  `json:"totalPositionIM"`
			WalletBalance           convert.StringToFloat64 `json:"walletBalance"`
			CummulativeRealisedPnl  convert.StringToFloat64 `json:"cumRealisedPnl"`
			Coin                    string                  `json:"coin"`
		} `json:"coin"`
	} `json:"list"`
}

// UnifiedAccountUpgradeResponse represents a response parameter for update to unified account.
type UnifiedAccountUpgradeResponse struct {
	UnifiedUpdateStatus string `json:"unifiedUpdateStatus"`
	UnifiedUpdateMsg    struct {
		Messages []string `json:"msg"`
	} `json:"unifiedUpdateMsg"`
}

// BorrowHistory represents interest records.
type BorrowHistory struct {
	NextPageCursor string `json:"nextPageCursor"`
	List           []struct {
		CreatedTime               convert.ExchangeTime    `json:"createdTime"`
		CostExemption             string                  `json:"costExemption"`
		InterestBearingBorrowSize string                  `json:"InterestBearingBorrowSize"`
		Currency                  string                  `json:"currency"`
		HourlyBorrowRate          convert.StringToFloat64 `json:"hourlyBorrowRate"`
		BorrowCost                convert.StringToFloat64 `json:"borrowCost"`
	} `json:"list"`
}

// CollateralInfo represents collateral information of the current unified margin account.
type CollateralInfo struct {
	List []struct {
		BorrowAmount        convert.StringToFloat64 `json:"borrowAmount"`
		AvailableToBorrow   string                  `json:"availableToBorrow"`
		FreeBorrowingAmount convert.StringToFloat64 `json:"freeBorrowingAmount"`
		Borrowable          bool                    `json:"borrowable"`
		Currency            string                  `json:"currency"`
		MaxBorrowingAmount  convert.StringToFloat64 `json:"maxBorrowingAmount"`
		HourlyBorrowRate    convert.StringToFloat64 `json:"hourlyBorrowRate"`
		MarginCollateral    bool                    `json:"marginCollateral"`
		CollateralRatio     convert.StringToFloat64 `json:"collateralRatio"`
	} `json:"list"`
}

// CoinGreeks represents current account greeks information.
type CoinGreeks struct {
	List []struct {
		BaseCoin   string                  `json:"baseCoin"`
		TotalDelta convert.StringToFloat64 `json:"totalDelta"`
		TotalGamma convert.StringToFloat64 `json:"totalGamma"`
		TotalVega  convert.StringToFloat64 `json:"totalVega"`
		TotalTheta convert.StringToFloat64 `json:"totalTheta"`
	} `json:"list"`
}

// FeeRate represents maker and taker fee rate information for a symbol.
type FeeRate struct {
	Symbol       string                  `json:"symbol"`
	TakerFeeRate convert.StringToFloat64 `json:"takerFeeRate"`
	MakerFeeRate convert.StringToFloat64 `json:"makerFeeRate"`
}

// AccountInfo represents margin mode account information.
type AccountInfo struct {
	MarginMode          string               `json:"marginMode"`
	UpdatedTime         convert.ExchangeTime `json:"updatedTime"`
	UnifiedMarginStatus int64                `json:"unifiedMarginStatus"`
	DcpStatus           string               `json:"dcpStatus"`
	TimeWindow          int64                `json:"timeWindow"`
	SmpGroup            int64                `json:"smpGroup"`
}

// SetMarginModeResponse represents a response for setting margin mode.
type SetMarginModeResponse struct {
	Reasons []struct {
		ReasonCode string `json:"reasonCode"`
		ReasonMsg  string `json:"reasonMsg"`
	} `json:"reasons"`
}

// MMPRequestParam represents an MMP request parameter.
type MMPRequestParam struct {
	BaseCoin           string                  `json:"baseCoin"`
	TimeWindowMS       int64                   `json:"window,string"`
	FrozenPeriod       int64                   `json:"frozenPeriod,string"`
	TradeQuantityLimit convert.StringToFloat64 `json:"qtyLimit"`
	DeltaLimit         convert.StringToFloat64 `json:"deltaLimit"`
}

// MMPStates represents an MMP states.
type MMPStates struct {
	Result []struct {
		BaseCoin           string                  `json:"baseCoin"`
		MmpEnabled         bool                    `json:"mmpEnabled"`
		Window             string                  `json:"window"`
		FrozenPeriod       string                  `json:"frozenPeriod"`
		TradeQuantityLimit convert.StringToFloat64 `json:"qtyLimit"`
		DeltaLimit         convert.StringToFloat64 `json:"deltaLimit"`
		MmpFrozenUntil     convert.StringToFloat64 `json:"mmpFrozenUntil"`
		MmpFrozen          bool                    `json:"mmpFrozen"`
	} `json:"result"`
}

// CoinExchangeRecords represents a coin exchange records.
type CoinExchangeRecords struct {
	OrderBody []struct {
		FromCoin              string                  `json:"fromCoin"`
		FromAmount            convert.StringToFloat64 `json:"fromAmount"`
		ToCoin                string                  `json:"toCoin"`
		ToAmount              convert.StringToFloat64 `json:"toAmount"`
		ExchangeRate          convert.StringToFloat64 `json:"exchangeRate"`
		CreatedTime           convert.ExchangeTime    `json:"createdTime"`
		ExchangeTransactionID string                  `json:"exchangeTxId"`
	} `json:"orderBody"`
	NextPageCursor string `json:"nextPageCursor"`
}

// DeliveryRecord represents delivery records of USDC futures and Options.
type DeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol                        string                  `json:"symbol"`
		Side                          string                  `json:"side"`
		DeliveryTime                  convert.ExchangeTime    `json:"deliveryTime"`
		ExercisePrice                 convert.StringToFloat64 `json:"strike"`
		Fee                           convert.StringToFloat64 `json:"fee"`
		Position                      convert.StringToFloat64 `json:"position"`
		DeliveryPrice                 convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryRealizedProfitAndLoss convert.StringToFloat64 `json:"deliveryRpl"`
	} `json:"list"`
}

// AccountInfos represents account type and account information
type AccountInfos map[string]struct {
	Status string `json:"status"`
	Assets []struct {
		Coin     string `json:"coin"`
		Frozen   string `json:"frozen"`
		Free     string `json:"free"`
		Withdraw string `json:"withdraw"`
	} `json:"assets"`
}

// CoinBalances represents coin balances for a specific asset type.
type CoinBalances struct {
	AccountType string `json:"accountType"`
	BizType     int    `json:"bizType"`
	AccountID   string `json:"accountId"`
	MemberID    string `json:"memberId"`
	Balance     struct {
		Coin               string                  `json:"coin"`
		WalletBalance      convert.StringToFloat64 `json:"walletBalance"`
		TransferBalance    convert.StringToFloat64 `json:"transferBalance"`
		Bonus              convert.StringToFloat64 `json:"bonus"`
		TransferSafeAmount string                  `json:"transferSafeAmount"`
	} `json:"balance"`
}

// TransferableCoins represents list of transferable coins.
type TransferableCoins struct {
	List []string `json:"list"`
}

// TransferParams represents parameters from internal coin transfer.
type TransferParams struct {
	TransferID      uuid.UUID               `json:"transferId"`
	Coin            currency.Code           `json:"coin"`
	Amount          convert.StringToFloat64 `json:"amount,string"`
	FromAccountType string                  `json:"fromAccountType"`
	ToAccountType   string                  `json:"toAccountType"`

	// Added for universal transfers
	FromMemberID int64 `json:"fromMemberId"`
	ToMemberID   int64 `json:"toMemberId"`
}

// TransferResponse represents a transfer response
type TransferResponse struct {
	List []struct {
		TransferID      string                  `json:"transferId"`
		Coin            string                  `json:"coin"`
		Amount          convert.StringToFloat64 `json:"amount"`
		FromAccountType string                  `json:"fromAccountType"`
		ToAccountType   string                  `json:"toAccountType"`
		Timestamp       convert.ExchangeTime    `json:"timestamp"`
		Status          string                  `json:"status"`

		// Returned with universal transfer IDs.
		FromMemberID string `json:"fromMemberId"`
		ToMemberID   string `json:"toMemberId"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// SubUID represents a sub-users ID
type SubUID struct {
	SubMemberIds             []string `json:"subMemberIds"`
	TransferableSubMemberIds []string `json:"transferableSubMemberIds"`
}

// AllowedDepositCoinInfo represents coin deposit information.
type AllowedDepositCoinInfo struct {
	ConfigList []struct {
		Coin               string `json:"coin"`
		Chain              string `json:"chain"`
		CoinShowName       string `json:"coinShowName"`
		ChainType          string `json:"chainType"`
		BlockConfirmNumber int    `json:"blockConfirmNumber"`
		MinDepositAmount   string `json:"minDepositAmount"`
	} `json:"configList"`
	NextPageCursor string `json:"nextPageCursor"`
}

// StatusResponse represents account information
type StatusResponse struct {
	Status int64 `json:"status"` // 1: SUCCESS 0: FAIL
}

// DepositRecords
type DepositRecords struct {
	Rows []struct {
		Coin          string `json:"coin"`
		Chain         string `json:"chain"`
		Amount        string `json:"amount"`
		TxID          string `json:"txID"`
		Status        int    `json:"status"`
		ToAddress     string `json:"toAddress"`
		Tag           string `json:"tag"`
		DepositFee    string `json:"depositFee"`
		SuccessAt     string `json:"successAt"`
		Confirmations string `json:"confirmations"`
		TxIndex       string `json:"txIndex"`
		BlockHash     string `json:"blockHash"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// InternalDepositRecords represents internal deposit records response instances
type InternalDepositRecords struct {
	Rows []struct {
		ID          string                  `json:"id"`
		Amount      convert.StringToFloat64 `json:"amount"`
		Type        int64                   `json:"type"`
		Coin        string                  `json:"coin"`
		Address     string                  `json:"address"`
		Status      int64                   `json:"status"`
		CreatedTime convert.ExchangeTime    `json:"createdTime"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// DepositAddresses represents deposit address information.
type DepositAddresses struct {
	Coin   string `json:"coin"`
	Chains []struct {
		ChainType      string `json:"chainType"`
		AddressDeposit string `json:"addressDeposit"`
		TagDeposit     string `json:"tagDeposit"`
		Chain          string `json:"chain"`
	} `json:"chains"`
}

// CoinInfo represents coin info information.
type CoinInfo struct {
	Rows []struct {
		Name         string                  `json:"name"`
		Coin         string                  `json:"coin"`
		RemainAmount convert.StringToFloat64 `json:"remainAmount"`
		Chains       []struct {
			ChainType             string                  `json:"chainType"`
			Confirmation          string                  `json:"confirmation"`
			WithdrawFee           convert.StringToFloat64 `json:"withdrawFee"`
			DepositMin            convert.StringToFloat64 `json:"depositMin"`
			WithdrawMin           convert.StringToFloat64 `json:"withdrawMin"`
			Chain                 string                  `json:"chain"`
			ChainDeposit          convert.StringToFloat64 `json:"chainDeposit"`
			ChainWithdraw         string                  `json:"chainWithdraw"`
			MinAccuracy           convert.StringToFloat64 `json:"minAccuracy"`
			WithdrawPercentageFee convert.StringToFloat64 `json:"withdrawPercentageFee"`
		} `json:"chains"`
	} `json:"rows"`
}

// WithdrawalRecords represents a list of withdrawal records.
type WithdrawalRecords struct {
	Rows []struct {
		Coin          string                  `json:"coin"`
		Chain         string                  `json:"chain"`
		Amount        convert.StringToFloat64 `json:"amount"`
		TransactionID string                  `json:"txID"`
		Status        string                  `json:"status"`
		ToAddress     string                  `json:"toAddress"`
		Tag           string                  `json:"tag"`
		WithdrawFee   convert.StringToFloat64 `json:"withdrawFee"`
		CreateTime    convert.ExchangeTime    `json:"createTime"`
		UpdateTime    convert.ExchangeTime    `json:"updateTime"`
		WithdrawID    string                  `json:"withdrawId"`
		WithdrawType  int                     `json:"withdrawType"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// WithdrawableAmount represents withdrawable amount information for each currency code
type WithdrawableAmount struct {
	LimitAmountUsd     string `json:"limitAmountUsd"`
	WithdrawableAmount map[string]struct {
		Coin               string                  `json:"coin"`
		WithdrawableAmount convert.StringToFloat64 `json:"withdrawableAmount"`
		AvailableBalance   convert.StringToFloat64 `json:"availableBalance"`
	} `json:"withdrawableAmount"`
}

// WithdrawalParam represents asset withdrawal request parameter.
type WithdrawalParam struct {
	Coin        currency.Code           `json:"coin,omitempty"`
	Chain       string                  `json:"chain,omitempty"`
	Address     string                  `json:"address,omitempty"`
	Tag         string                  `json:"tag,omitempty"`
	Amount      convert.StringToFloat64 `json:"amount,omitempty,string"`
	Timestamp   int64                   `json:"timestamp,omitempty"`
	ForceChain  int64                   `json:"forceChain,omitempty"`
	AccountType string                  `json:"accountType,omitempty"`
}

// CreateSubUserParams parameter to create a new sub user id. Use master user's api key only.
type CreateSubUserParams struct {
	Username   string `json:"username,omitempty"`   // Give a username of the new sub user id.
	Password   string `json:"password,omitempty"`   // Set the password for the new sub user id. 8-30 characters, must include numbers, capital and little letters.
	MemberType int64  `json:"memberType,omitempty"` // '1': normal sub account, '6': custodial sub account
	Switch     int64  `json:"switch,omitempty"`     // '0': turn off quick login (default) '1': turn on quick login
	IsUTC      bool   `json:"isUta,omitempty"`
	Note       string `json:"note,omitempty"`
}

// SubUserItem represents a sub user response instance.
type SubUserItem struct {
	UID        string `json:"uid"`
	Username   string `json:"username"`
	MemberType int64  `json:"memberType"`
	Status     int64  `json:"status"`
	Remark     string `json:"remark"`
}

// SubUIDAPIKeyParam represents a sub-user ID API key creation paramter.
type SubUIDAPIKeyParam struct {
	Subuid int64  `json:"subuid"`
	Note   string `json:"note"`

	ReadOnly int64 `json:"readOnly"`

	// Set the IP bind. example: ["192.168.0.1,192.168.0.2"]note:
	// don't pass ips or pass with ["*"] means no bind
	// No ip bound api key will be invalid after 90 days
	// api key will be invalid after 7 days once the account password is changed
	IPs []string `json:"ips"`

	// Tick the types of permission. one of below types must be passed, otherwise the error is thrown
	Permissions map[string][]string `json:"permissions,omitempty"`
}

// SubUIDAPIResponse represents sub UID API key response.
type SubUIDAPIResponse struct {
	ID          string              `json:"id"`
	Note        string              `json:"note"`
	APIKey      string              `json:"apiKey"`
	ReadOnly    int64               `json:"readOnly"`
	Secret      string              `json:"secret"`
	Permissions map[string][]string `json:"permissions"`

	IPS           []string  `json:"ips"`
	Type          int64     `json:"type"`
	DeadlineDay   int64     `json:"deadlineDay"`
	ExpiredAt     time.Time `json:"expiredAt"`
	CreatedAt     time.Time `json:"createdAt"`
	Unified       int64     `json:"unified"`
	Uta           int64     `json:"uta"`
	UserID        int64     `json:"userID"`
	InviterID     int64     `json:"inviterID"`
	VipLevel      string    `json:"vipLevel"`
	MktMakerLevel string    `json:"mktMakerLevel"`
	AffiliateID   int64     `json:"affiliateID"`
	RsaPublicKey  string    `json:"rsaPublicKey"`
	IsMaster      bool      `json:"isMaster"`
}

// WalletType represents available wallet types for the master account or sub account
type WalletType struct {
	Accounts []struct {
		UID         string   `json:"uid"`
		AccountType []string `json:"accountType"`
	} `json:"accounts"`
}

// SubUIDAPIKeyUpdateParam represents a sub-user ID API key update paramter.
type SubUIDAPIKeyUpdateParam struct {
	ReadOnly int64 `json:"readOnly"`
	// Set the IP bind. example: ["192.168.0.1,192.168.0.2"]note:
	// don't pass ips or pass with ["*"] means no bind
	// No ip bound api key will be invalid after 90 days
	// api key will be invalid after 7 days once the account password is changed
	IPs []string `json:"ips"`
	// Tick the types of permission. one of below types must be passed, otherwise the error is thrown
	Permissions map[string][]string `json:"permissions,omitempty"`
}

// AffiliateCustomerInfo represents user information
type AffiliateCustomerInfo struct {
	UID                 string                  `json:"uid"`
	TakerVol30Day       convert.StringToFloat64 `json:"takerVol30Day"`
	MakerVol30Day       convert.StringToFloat64 `json:"makerVol30Day"`
	TradeVol30Day       convert.StringToFloat64 `json:"tradeVol30Day"`
	DepositAmount30Day  convert.StringToFloat64 `json:"depositAmount30Day"`
	TakerVol365Day      convert.StringToFloat64 `json:"takerVol365Day"`
	MakerVol365Day      convert.StringToFloat64 `json:"makerVol365Day"`
	TradeVol365Day      convert.StringToFloat64 `json:"tradeVol365Day"`
	DepositAmount365Day convert.StringToFloat64 `json:"depositAmount365Day"`
	TotalWalletBalance  convert.StringToFloat64 `json:"totalWalletBalance"`
	DepositUpdateTime   time.Time               `json:"depositUpdateTime"`
	VipLevel            string                  `json:"vipLevel"`
	VolUpdateTime       time.Time               `json:"volUpdateTime"`
}

// LeverageTokenInfo represents leverage token information.
type LeverageTokenInfo struct {
	FundFee          convert.StringToFloat64 `json:"fundFee"`
	FundFeeTime      convert.ExchangeTime    `json:"fundFeeTime"`
	LtCoin           string                  `json:"ltCoin"`
	LtName           string                  `json:"ltName"`
	LtStatus         string                  `json:"ltStatus"`
	ManageFeeRate    convert.StringToFloat64 `json:"manageFeeRate"`
	ManageFeeTime    convert.ExchangeTime    `json:"manageFeeTime"`
	MaxPurchase      string                  `json:"maxPurchase"`
	MaxPurchaseDaily string                  `json:"maxPurchaseDaily"`
	MaxRedeem        string                  `json:"maxRedeem"`
	MaxRedeemDaily   string                  `json:"maxRedeemDaily"`
	MinPurchase      string                  `json:"minPurchase"`
	MinRedeem        string                  `json:"minRedeem"`
	NetValue         convert.StringToFloat64 `json:"netValue"`
	PurchaseFeeRate  convert.StringToFloat64 `json:"purchaseFeeRate"`
	RedeemFeeRate    convert.StringToFloat64 `json:"redeemFeeRate"`
	Total            convert.StringToFloat64 `json:"total"`
	Value            convert.StringToFloat64 `json:"value"`
}

// LeveragTokenMarket represents leverage token market details.
type LeveragedTokenMarket struct {
	Basket      convert.StringToFloat64 `json:"basket"`
	Circulation convert.StringToFloat64 `json:"circulation"`
	Leverage    convert.StringToFloat64 `json:"leverage"` // Real leverage calculated by last traded price
	LTCoin      string                  `json:"ltCoin"`
	NetValue    convert.StringToFloat64 `json:"nav"`
	NavTime     convert.ExchangeTime    `json:"navTime"` // Update time for net asset value (in milliseconds and UTC time zone)
}

// LeverageToken represents a response instance when purchasing a leverage token.
type LeverageToken struct {
	Amount        convert.StringToFloat64 `json:"amount"`
	ExecAmt       convert.StringToFloat64 `json:"execAmt"`
	ExecQty       convert.StringToFloat64 `json:"execQty"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	PurchaseID    string                  `json:"purchaseId"`
	SerialNo      string                  `json:"serialNo"`
	ValueCoin     string                  `json:"valueCoin"`
}

// RedeemToken represents leverage redeem token
type RedeemToken struct {
	ExecAmt       convert.StringToFloat64 `json:"execAmt"`
	ExecQty       convert.StringToFloat64 `json:"execQty"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	Quantity      convert.StringToFloat64 `json:"quantity"`
	RedeemID      string                  `json:"redeemId"`
	SerialNo      string                  `json:"serialNo"`
	ValueCoin     string                  `json:"valueCoin"`
}

// RedeemPurchaseRecord represents a purchase and redeem record instance.
type RedeemPurchaseRecord struct {
	Amount        convert.StringToFloat64 `json:"amount"`
	Fee           convert.StringToFloat64 `json:"fee"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	LtOrderType   string                  `json:"ltOrderType"`
	OrderID       string                  `json:"orderId"`
	OrderTime     convert.ExchangeTime    `json:"orderTime"`
	SerialNo      string                  `json:"serialNo"`
	UpdateTime    convert.ExchangeTime    `json:"updateTime"`
	Value         convert.StringToFloat64 `json:"value"`
	ValueCoin     string                  `json:"valueCoin"`
}

// SpotMarginMode represents data about whether spot margin trade is on / off
type SpotMarginMode struct {
	SpotMarginMode string `json:"spotMarginMode"`
}

// MarginCoinInfo represents margin coin information.
type MarginCoinInfo struct {
	Coin             string `json:"coin"`
	ConversionRate   string `json:"conversionRate"`
	LiquidationOrder int    `json:"liquidationOrder"`
}

// BorrowableCoinInfo represents borrowable coin information.
type BorrowableCoinInfo struct {
	BorrowingPrecision int64  `json:"borrowingPrecision"`
	Coin               string `json:"coin"`
	RepaymentPrecision int64  `json:"repaymentPrecision"`
}

// InterestAndQuota represents interest and quota information.
type InterestAndQuota struct {
	Coin           string                  `json:"coin"`
	InterestRate   string                  `json:"interestRate"`
	LoanAbleAmount convert.StringToFloat64 `json:"loanAbleAmount"`
	MaxLoanAmount  convert.StringToFloat64 `json:"maxLoanAmount"`
}

// AccountLoanInfo covers: Margin trade (Normal Account)
type AccountLoanInfo struct {
	AcctBalanceSum  convert.StringToFloat64 `json:"acctBalanceSum"`
	DebtBalanceSum  convert.StringToFloat64 `json:"debtBalanceSum"`
	LoanAccountList []struct {
		Interest     string                  `json:"interest"`
		Loan         string                  `json:"loan"`
		Locked       string                  `json:"locked"`
		TokenID      string                  `json:"tokenId"`
		Free         convert.StringToFloat64 `json:"free"`
		RemainAmount convert.StringToFloat64 `json:"remainAmount"`
		Total        convert.StringToFloat64 `json:"total"`
	} `json:"loanAccountList"`
	RiskRate     convert.StringToFloat64 `json:"riskRate"`
	Status       int64                   `json:"status"`
	SwitchStatus int64                   `json:"switchStatus"`
}

// BorrowResponse represents borrow response transaction id.
type BorrowResponse struct {
	TransactID string `json:"transactId"`
}

// LendArgument represents currency borrow and repay parameter.
type LendArgument struct {
	Coin           currency.Code `json:"coin"`
	AmountToBorrow float64       `json:"qty,string"`
}

// RepayResponse represents a repay id
type RepayResponse struct {
	RepayID string `json:"repayId"`
}

// BorrowOrderDetail represents a borrow order detail info.
type BorrowOrderDetail struct {
	ID              string                  `json:"id"`
	AccountID       string                  `json:"accountId"`
	Coin            string                  `json:"coin"`
	CreatedTime     convert.ExchangeTime    `json:"createdTime"`
	InterestAmount  convert.StringToFloat64 `json:"interestAmount"`
	InterestBalance convert.StringToFloat64 `json:"interestBalance"`
	LoanAmount      convert.StringToFloat64 `json:"loanAmount"`
	LoanBalance     convert.StringToFloat64 `json:"loanBalance"`
	RemainAmount    convert.StringToFloat64 `json:"remainAmount"`
	Status          int64                   `json:"status"`
	Type            int64                   `json:"type"`
}

// CoinRepaymentResponse represents a coin repayment detail.
type CoinRepaymentResponse struct {
	AccountID          string                  `json:"accountId"`
	Coin               string                  `json:"coin"`
	RepaidAmount       convert.StringToFloat64 `json:"repaidAmount"`
	RepayID            string                  `json:"repayId"`
	RepayMarginOrderID string                  `json:"repayMarginOrderId"`
	RepayTime          convert.ExchangeTime    `json:"repayTime"`
	TransactIds        []struct {
		RepaidAmount       convert.StringToFloat64 `json:"repaidAmount"`
		RepaidInterest     convert.StringToFloat64 `json:"repaidInterest"`
		RepaidPrincipal    convert.StringToFloat64 `json:"repaidPrincipal"`
		RepaidSerialNumber convert.StringToFloat64 `json:"repaidSerialNumber"`
		TransactID         string                  `json:"transactId"`
	} `json:"transactIds"`
}

// InstitutionalProductInfo represents institutional product info.
type InstitutionalProductInfo struct {
	MarginProductInfo []struct {
		ProductID           string   `json:"productId"`
		Leverage            string   `json:"leverage"`
		SupportSpot         int64    `json:"supportSpot"`
		SupportContract     int64    `json:"supportContract"`
		WithdrawLine        string   `json:"withdrawLine"`
		TransferLine        string   `json:"transferLine"`
		SpotBuyLine         string   `json:"spotBuyLine"`
		SpotSellLine        string   `json:"spotSellLine"`
		ContractOpenLine    string   `json:"contractOpenLine"`
		LiquidationLine     string   `json:"liquidationLine"`
		StopLiquidationLine string   `json:"stopLiquidationLine"`
		ContractLeverage    string   `json:"contractLeverage"`
		TransferRatio       string   `json:"transferRatio"`
		SpotSymbols         []string `json:"spotSymbols"`
		ContractSymbols     []string `json:"contractSymbols"`
	} `json:"marginProductInfo"`
}

// InstitutionalMarginCoinInfo represents margin coin info for institutional lending
// token and tokens convert information.
type InstitutionalMarginCoinInfo struct {
	MarginToken []struct {
		ProductID string `json:"productId"`
		TokenInfo []struct {
			Token            string `json:"token"`
			ConvertRatioList []struct {
				Ladder       string                  `json:"ladder"`
				ConvertRatio convert.StringToFloat64 `json:"convertRatio"`
			} `json:"convertRatioList"`
		} `json:"tokenInfo"`
	} `json:"marginToken"`
}

// LoanOrderDetails retrieves institutional loan order detail item.
type LoanOrderDetails struct {
	OrderID             string                  `json:"orderId"`
	OrderProductID      string                  `json:"orderProductId"`
	ParentUID           string                  `json:"parentUid"`
	LoanTime            convert.ExchangeTime    `json:"loanTime"`
	LoanCoin            string                  `json:"loanCoin"`
	LoanAmount          convert.StringToFloat64 `json:"loanAmount"`
	UnpaidAmount        convert.StringToFloat64 `json:"unpaidAmount"`
	UnpaidInterest      convert.StringToFloat64 `json:"unpaidInterest"`
	RepaidAmount        convert.StringToFloat64 `json:"repaidAmount"`
	RepaidInterest      convert.StringToFloat64 `json:"repaidInterest"`
	InterestRate        convert.StringToFloat64 `json:"interestRate"`
	Status              int64                   `json:"status"`
	Leverage            string                  `json:"leverage"`
	SupportSpot         int64                   `json:"supportSpot"`
	SupportContract     int64                   `json:"supportContract"`
	WithdrawLine        convert.StringToFloat64 `json:"withdrawLine"`
	TransferLine        convert.StringToFloat64 `json:"transferLine"`
	SpotBuyLine         convert.StringToFloat64 `json:"spotBuyLine"`
	SpotSellLine        convert.StringToFloat64 `json:"spotSellLine"`
	ContractOpenLine    convert.StringToFloat64 `json:"contractOpenLine"`
	LiquidationLine     convert.StringToFloat64 `json:"liquidationLine"`
	StopLiquidationLine convert.StringToFloat64 `json:"stopLiquidationLine"`
	ContractLeverage    convert.StringToFloat64 `json:"contractLeverage"`
	TransferRatio       convert.StringToFloat64 `json:"transferRatio"`
	SpotSymbols         []string                `json:"spotSymbols"`
	ContractSymbols     []string                `json:"contractSymbols"`
}

// OrderRepayInfo represents repaid information information.
type OrderRepayInfo struct {
	RepayOrderID string               `json:"repayOrderId"`
	RepaidTime   convert.ExchangeTime `json:"repaidTime"`
	Token        string               `json:"token"`
	Quantity     convert.ExchangeTime `json:"quantity"`
	Interest     convert.ExchangeTime `json:"interest"`
	BusinessType string               `json:"businessType"`
	Status       string               `json:"status"`
}

// LTVInfo represents institutional lending Loan-to-value(LTV)
type LTVInfo struct {
	LtvInfo []struct {
		LoanToValue    string                  `json:"ltv"`
		ParentUID      string                  `json:"parentUid"`
		SubAccountUids []string                `json:"subAccountUids"`
		UnpaidAmount   convert.StringToFloat64 `json:"unpaidAmount"`
		UnpaidInfo     []struct {
			Token          string                  `json:"token"`
			UnpaidQty      convert.StringToFloat64 `json:"unpaidQty"`
			UnpaidInterest convert.StringToFloat64 `json:"unpaidInterest"`
		} `json:"unpaidInfo"`
		Balance     string `json:"balance"`
		BalanceInfo []struct {
			Token           string                  `json:"token"`
			Price           convert.StringToFloat64 `json:"price"`
			Qty             convert.StringToFloat64 `json:"qty"`
			ConvertedAmount convert.StringToFloat64 `json:"convertedAmount"`
		} `json:"balanceInfo"`
	} `json:"ltvInfo"`
}

// C2CLendingCoinInfo represent contract-to-contract lending coin information.
type C2CLendingCoinInfo struct {
	Coin            string                  `json:"coin"`
	LoanToPoolRatio convert.StringToFloat64 `json:"loanToPoolRatio"`
	MaxRedeemQty    convert.StringToFloat64 `json:"maxRedeemQty"`
	MinPurchaseQty  convert.StringToFloat64 `json:"minPurchaseQty"`
	Precision       convert.StringToFloat64 `json:"precision"`
	Rate            convert.StringToFloat64 `json:"rate"`
}

// C2CLendingFundsParams represents deposit funds parameter
type C2CLendingFundsParams struct {
	Coin         currency.Code `json:"coin"`
	Quantity     float64       `json:"quantity,string"`
	SerialNumber string        `json:"serialNO"`
}

// C2CLendingFundResponse represents contract-to-contract deposit funds item.
type C2CLendingFundResponse struct {
	Coin         string                  `json:"coin"`
	OrderID      string                  `json:"orderId"`
	SerialNumber string                  `json:"serialNo"`
	Status       string                  `json:"status"`
	CreatedTime  convert.ExchangeTime    `json:"createdTime"`
	Quantity     convert.StringToFloat64 `json:"quantity"`
	UpdatedTime  convert.ExchangeTime    `json:"updatedTime"`

	// added for redeem funds
	PrincipalQty string `json:"principalQty"`

	// added for to distinguish between redeem funds and deposit funds
	OrderType string `json:"orderType"`
}

// LendingAccountInfo represents contract-to-contract lending account info item.
type LendingAccountInfo struct {
	Coin              string                  `json:"coin"`
	PrincipalInterest string                  `json:"principalInterest"`
	PrincipalQty      convert.StringToFloat64 `json:"principalQty"`
	PrincipalTotal    convert.StringToFloat64 `json:"principalTotal"`
	Quantity          convert.StringToFloat64 `json:"quantity"`
}

// BrokerEarningItem represents contract-to-contract broker earning item.
type BrokerEarningItem struct {
	UserID   string               `json:"userId"`
	BizType  string               `json:"bizType"`
	Symbol   string               `json:"symbol"`
	Coin     string               `json:"coin"`
	Earning  string               `json:"earning"`
	OrderID  string               `json:"orderId"`
	ExecTime convert.ExchangeTime `json:"execTime"`
}

// ServerTime represents server time
type ServerTime struct {
	TimeSecond convert.ExchangeTime `json:"timeSecond"`
	TimeNano   convert.ExchangeTime `json:"timeNano"`
}
