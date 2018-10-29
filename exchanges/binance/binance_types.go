package binance

import (
	"encoding/json"

	"github.com/thrasher-/gocryptotrader/currency/symbol"
)

// Response holds basic binance api response data
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ExchangeInfo holds the full exchange information type
type ExchangeInfo struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Timezone   string `json:"timezone"`
	Servertime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []struct {
		Symbol             string   `json:"symbol"`
		Status             string   `json:"status"`
		BaseAsset          string   `json:"baseAsset"`
		BaseAssetPrecision int      `json:"baseAssetPrecision"`
		QuoteAsset         string   `json:"quoteAsset"`
		QuotePrecision     int      `json:"quotePrecision"`
		OrderTypes         []string `json:"orderTypes"`
		IcebergAllowed     bool     `json:"icebergAllowed"`
		Filters            []struct {
			FilterType  string  `json:"filterType"`
			MinPrice    float64 `json:"minPrice,string"`
			MaxPrice    float64 `json:"maxPrice,string"`
			TickSize    float64 `json:"tickSize,string"`
			MinQty      float64 `json:"minQty,string"`
			MaxQty      float64 `json:"maxQty,string"`
			StepSize    float64 `json:"stepSize,string"`
			MinNotional float64 `json:"minNotional,string"`
		} `json:"filters"`
	} `json:"symbols"`
}

// OrderBookDataRequestParams represents Klines request data.
type OrderBookDataRequestParams struct {
	Symbol string `json:"symbol"` // Required field; example LTCBTC,BTCUSDT
	Limit  int    `json:"limit"`  // Default 100; max 1000. Valid limits:[5, 10, 20, 50, 100, 500, 1000]
}

// OrderBookData is resp data from orderbook endpoint
type OrderBookData struct {
	Code         int           `json:"code"`
	Msg          string        `json:"msg"`
	LastUpdateID int64         `json:"lastUpdateId"`
	Bids         []interface{} `json:"bids"`
	Asks         []interface{} `json:"asks"`
}

// OrderBook actual structured data that can be used for orderbook
type OrderBook struct {
	LastUpdateID int64
	Code         int
	Msg          string
	Bids         []struct {
		Price    float64
		Quantity float64
	}
	Asks []struct {
		Price    float64
		Quantity float64
	}
}

// DepthUpdateParams is used as an embedded type for WebsocketDepthStream
type DepthUpdateParams []struct {
	PriceLevel float64
	Quantity   float64
	ingnore    []interface{}
}

// WebsocketDepthStream is the difference for the update depth stream
type WebsocketDepthStream struct {
	Event         string        `json:"e"`
	Timestamp     int64         `json:"E"`
	Pair          string        `json:"s"`
	FirstUpdateID int64         `json:"U"`
	LastUpdateID  int64         `json:"u"`
	UpdateBids    []interface{} `json:"b"`
	UpdateAsks    []interface{} `json:"a"`
}

// RecentTradeRequestParams represents Klines request data.
type RecentTradeRequestParams struct {
	Symbol string `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int    `json:"limit"`  // Default 500; max 500.
}

// RecentTrade holds recent trade data
type RecentTrade struct {
	Code         int     `json:"code"`
	Msg          string  `json:"msg"`
	ID           int64   `json:"id"`
	Price        float64 `json:"price,string"`
	Quantity     float64 `json:"qty,string"`
	Time         float64 `json:"time"`
	IsBuyerMaker bool    `json:"isBuyerMaker"`
	IsBestMatch  bool    `json:"isBestMatch"`
}

// MultiStreamData holds stream data
type MultiStreamData struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

// TradeStream holds the trade stream data
type TradeStream struct {
	EventType      string `json:"e"`
	EventTime      int64  `json:"E"`
	Symbol         string `json:"s"`
	TradeID        int64  `json:"t"`
	Price          string `json:"p"`
	Quantity       string `json:"q"`
	BuyerOrderID   int64  `json:"b"`
	SellerOrderID  int64  `json:"a"`
	TimeStamp      int64  `json:"T"`
	Maker          bool   `json:"m"`
	BestMatchPrice bool   `json:"M"`
}

// KlineStream holds the kline stream data
type KlineStream struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	Kline     struct {
		StartTime                int64  `json:"t"`
		CloseTime                int64  `json:"T"`
		Symbol                   string `json:"s"`
		Interval                 string `json:"i"`
		FirstTradeID             int64  `json:"f"`
		LastTradeID              int64  `json:"L"`
		OpenPrice                string `json:"o"`
		ClosePrice               string `json:"c"`
		HighPrice                string `json:"h"`
		LowPrice                 string `json:"l"`
		Volume                   string `json:"v"`
		NumberOfTrades           int64  `json:"n"`
		KlineClosed              bool   `json:"x"`
		Quote                    string `json:"q"`
		TakerBuyBaseAssetVolume  string `json:"V"`
		TakerBuyQuoteAssetVolume string `json:"Q"`
	} `json:"k"`
}

// TickerStream holds the ticker stream data
type TickerStream struct {
	EventType              string `json:"e"`
	EventTime              int64  `json:"E"`
	Symbol                 string `json:"s"`
	PriceChange            string `json:"p"`
	PriceChangePercent     string `json:"P"`
	WeightedAvgPrice       string `json:"w"`
	PrevDayClose           string `json:"x"`
	CurrDayClose           string `json:"c"`
	CloseTradeQuantity     string `json:"Q"`
	BestBidPrice           string `json:"b"`
	BestBidQuantity        string `json:"B"`
	BestAskPrice           string `json:"a"`
	BestAskQuantity        string `json:"A"`
	OpenPrice              string `json:"o"`
	HighPrice              string `json:"h"`
	LowPrice               string `json:"l"`
	TotalTradedVolume      string `json:"v"`
	TotalTradedQuoteVolume string `json:"q"`
	OpenTime               int64  `json:"O"`
	CloseTime              int64  `json:"C"`
	FirstTradeID           int64  `json:"F"`
	LastTradeID            int64  `json:"L"`
	NumberOfTrades         int64  `json:"n"`
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	Code         int     `json:"code"`
	Msg          string  `json:"msg"`
	ID           int64   `json:"id"`
	Price        float64 `json:"price,string"`
	Quantity     float64 `json:"qty,string"`
	Time         int64   `json:"time"`
	IsBuyerMaker bool    `json:"isBuyerMaker"`
	IsBestMatch  bool    `json:"isBestMatch"`
}

// AggregatedTrade holds aggregated trade information
type AggregatedTrade struct {
	ATradeID       int64   `json:"a"`
	Price          float64 `json:"p,string"`
	Quantity       float64 `json:"q,string"`
	FirstTradeID   int64   `json:"f"`
	LastTradeID    int64   `json:"l"`
	TimeStamp      int64   `json:"T"`
	Maker          bool    `json:"m"`
	BestMatchPrice bool    `json:"M"`
}

// CandleStick holds kline data
type CandleStick struct {
	OpenTime                 float64
	Open                     float64
	High                     float64
	Low                      float64
	Close                    float64
	Volume                   float64
	CloseTime                float64
	QuoteAssetVolume         float64
	TradeCount               float64
	TakerBuyAssetVolume      float64
	TakerBuyQuoteAssetVolume float64
}

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Symbol             string  `json:"symbol"`
	PriceChange        float64 `json:"priceChange,string"`
	PriceChangePercent float64 `json:"priceChangePercent,string"`
	WeightedAvgPrice   float64 `json:"weightedAvgPrice,string"`
	PrevClosePrice     float64 `json:"prevClosePrice,string"`
	LastPrice          float64 `json:"lastPrice,string"`
	LastQty            float64 `json:"lastQty,string"`
	BidPrice           float64 `json:"bidPrice,string"`
	AskPrice           float64 `json:"askPrice,string"`
	OpenPrice          float64 `json:"openPrice,string"`
	HighPrice          float64 `json:"highPrice,string"`
	LowPrice           float64 `json:"lowPrice,string"`
	Volume             float64 `json:"volume,string"`
	QuoteVolume        float64 `json:"quoteVolume,string"`
	OpenTime           int64   `json:"openTime"`
	CloseTime          int64   `json:"closeTime"`
	FirstID            int64   `json:"fristId"`
	LastID             int64   `json:"lastId"`
	Count              int64   `json:"count"`
}

// SymbolPrice holds basic symbol price
type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

// BestPrice holds best price data
type BestPrice struct {
	Symbol   string  `json:"symbol"`
	BidPrice float64 `json:"bidPrice,string"`
	BidQty   float64 `json:"bidQty,string"`
	AskPrice float64 `json:"askPrice,string"`
	AskQty   float64 `json:"askQty,string"`
}

// NewOrderRequest request type
type NewOrderRequest struct {
	// Symbol (currency pair to trade)
	Symbol string
	// Side Buy or Sell
	Side RequestParamsSideType
	// TradeType (market or limit order)
	TradeType RequestParamsOrderType
	// TimeInForce specifies how long the order remains in effect.
	// Examples are (Good Till Cancel (GTC), Immediate or Cancel (IOC) and Fill Or Kill (FOK))
	TimeInForce RequestParamsTimeForceType
	// Quantity
	Quantity         float64
	Price            float64
	NewClientOrderID string
	StopPrice        float64 //Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	IcebergQty       float64 //Used with LIMIT, STOP_LOSS_LIMIT, and TAKE_PROFIT_LIMIT to create an iceberg order.
	NewOrderRespType string
}

// NewOrderResponse is the return structured response from the exchange
type NewOrderResponse struct {
	Code            int     `json:"code"`
	Msg             string  `json:"msg"`
	Symbol          string  `json:"symbol"`
	OrderID         int64   `json:"orderId"`
	ClientOrderID   string  `json:"clientOrderId"`
	TransactionTime int64   `json:"transactTime"`
	Price           float64 `json:"price,string"`
	OrigQty         float64 `json:"origQty,string"`
	ExecutedQty     float64 `json:"executedQty,string"`
	Status          string  `json:"status"`
	TimeInForce     string  `json:"timeInForce"`
	Type            string  `json:"type"`
	Side            string  `json:"side"`
	Fills           []struct {
		Price           float64 `json:"price,string"`
		Qty             float64 `json:"qty,string"`
		Commission      float64 `json:"commission,string"`
		CommissionAsset float64 `json:"commissionAsset,string"`
	} `json:"fills"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	Symbol            string `json:"symbol"`
	OrigClientOrderID string `json:"origClientOrderId"`
	OrderID           int64  `json:"orderId"`
	ClientOrderID     string `json:"clientOrderId"`
}

// QueryOrderData holds query order data
type QueryOrderData struct {
	Code          int     `json:"code"`
	Msg           string  `json:"msg"`
	Symbol        string  `json:"symbol"`
	OrderID       int64   `json:"orderId"`
	ClientOrderID string  `json:"clientOrderId"`
	Price         float64 `json:"price,string"`
	OrigQty       float64 `json:"origQty,string"`
	ExecutedQty   float64 `json:"executedQty,string"`
	Status        string  `json:"status"`
	TimeInForce   string  `json:"timeInForce"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	StopPrice     float64 `json:"stopPrice,string"`
	IcebergQty    float64 `json:"icebergQty,string"`
	Time          float64 `json:"time"`
	IsWorking     bool    `json:"isWorking"`
}

// Balance holds query order data
type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// Account holds the account data
type Account struct {
	MakerCommission  int       `json:"makerCommission"`
	TakerCommission  int       `json:"takerCommission"`
	BuyerCommission  int       `json:"buyerCommission"`
	SellerCommission int       `json:"sellerCommission"`
	CanTrade         bool      `json:"canTrade"`
	CanWithdraw      bool      `json:"canWithdraw"`
	CanDeposit       bool      `json:"canDeposit"`
	UpdateTime       int64     `json:"updateTime"`
	Balances         []Balance `json:"balances"`
}

// RequestParamsSideType trade order side (buy or sell)
type RequestParamsSideType string

var (
	// BinanceRequestParamsSideBuy buy order type
	BinanceRequestParamsSideBuy = RequestParamsSideType("BUY")

	// BinanceRequestParamsSideSell sell order type
	BinanceRequestParamsSideSell = RequestParamsSideType("SELL")
)

// RequestParamsTimeForceType Time in force
type RequestParamsTimeForceType string

var (
	// BinanceRequestParamsTimeGTC GTC
	BinanceRequestParamsTimeGTC = RequestParamsTimeForceType("GTC")

	// BinanceRequestParamsTimeIOC IOC
	BinanceRequestParamsTimeIOC = RequestParamsTimeForceType("IOC")

	// BinanceRequestParamsTimeFOK FOK
	BinanceRequestParamsTimeFOK = RequestParamsTimeForceType("FOK")
)

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

var (
	// BinanceRequestParamsOrderLimit Limit order
	BinanceRequestParamsOrderLimit = RequestParamsOrderType("LIMIT")

	// BinanceRequestParamsOrderMarket Market order
	BinanceRequestParamsOrderMarket = RequestParamsOrderType("MARKET")

	// BinanceRequestParamsOrderStopLoss STOP_LOSS
	BinanceRequestParamsOrderStopLoss = RequestParamsOrderType("STOP_LOSS")

	// BinanceRequestParamsOrderStopLossLimit STOP_LOSS_LIMIT
	BinanceRequestParamsOrderStopLossLimit = RequestParamsOrderType("STOP_LOSS_LIMIT")

	// BinanceRequestParamsOrderTakeProfit TAKE_PROFIT
	BinanceRequestParamsOrderTakeProfit = RequestParamsOrderType("TAKE_PROFIT")

	// BinanceRequestParamsOrderTakeProfitLimit TAKE_PROFIT_LIMIT
	BinanceRequestParamsOrderTakeProfitLimit = RequestParamsOrderType("TAKE_PROFIT_LIMIT")

	// BinanceRequestParamsOrderLimitMarker LIMIT_MAKER
	BinanceRequestParamsOrderLimitMarker = RequestParamsOrderType("LIMIT_MAKER")
)

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol    string       // Required field; example LTCBTC, BTCUSDT
	Interval  TimeInterval // Time interval period
	Limit     int          // Default 500; max 500.
	StartTime int64
	EndTime   int64
}

// TimeInterval represents interval enum.
type TimeInterval string

// Vars related to time intervals
var (
	TimeIntervalMinute         = TimeInterval("1m")
	TimeIntervalThreeMinutes   = TimeInterval("3m")
	TimeIntervalFiveMinutes    = TimeInterval("5m")
	TimeIntervalFifteenMinutes = TimeInterval("15m")
	TimeIntervalThirtyMinutes  = TimeInterval("30m")
	TimeIntervalHour           = TimeInterval("1h")
	TimeIntervalTwoHours       = TimeInterval("2h")
	TimeIntervalFourHours      = TimeInterval("4h")
	TimeIntervalSixHours       = TimeInterval("6h")
	TimeIntervalEightHours     = TimeInterval("8h")
	TimeIntervalTwelveHours    = TimeInterval("12h")
	TimeIntervalDay            = TimeInterval("1d")
	TimeIntervalThreeDays      = TimeInterval("3d")
	TimeIntervalWeek           = TimeInterval("1w")
	TimeIntervalMonth          = TimeInterval("1M")
)

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.BNB:     0.13,
	symbol.BTC:     0.0005,
	symbol.NEO:     0,
	symbol.ETH:     0.01,
	symbol.LTC:     0.001,
	symbol.QTUM:    0.01,
	symbol.EOS:     0.1,
	symbol.SNT:     35,
	symbol.BNT:     1,
	symbol.GAS:     0,
	symbol.BCC:     0.001,
	symbol.BTM:     5,
	symbol.USDT:    3.4,
	symbol.HCC:     0.0005,
	symbol.OAX:     6.5,
	symbol.DNT:     54,
	symbol.MCO:     0.31,
	symbol.ICN:     3.5,
	symbol.ZRX:     1.9,
	symbol.OMG:     0.4,
	symbol.WTC:     0.5,
	symbol.LRC:     12.3,
	symbol.LLT:     67.8,
	symbol.YOYO:    1,
	symbol.TRX:     1,
	symbol.STRAT:   0.1,
	symbol.SNGLS:   54,
	symbol.BQX:     3.9,
	symbol.KNC:     3.5,
	symbol.SNM:     25,
	symbol.FUN:     86,
	symbol.LINK:    4,
	symbol.XVG:     0.1,
	symbol.CTR:     35,
	symbol.SALT:    2.3,
	symbol.MDA:     2.3,
	symbol.IOTA:    0.5,
	symbol.SUB:     11.4,
	symbol.ETC:     0.01,
	symbol.MTL:     2,
	symbol.MTH:     45,
	symbol.ENG:     2.2,
	symbol.AST:     14.4,
	symbol.DASH:    0.002,
	symbol.BTG:     0.001,
	symbol.EVX:     2.8,
	symbol.REQ:     29.9,
	symbol.VIB:     30,
	symbol.POWR:    8.2,
	symbol.ARK:     0.2,
	symbol.XRP:     0.25,
	symbol.MOD:     2,
	symbol.ENJ:     26,
	symbol.STORJ:   5.1,
	symbol.KMD:     0.002,
	symbol.RCN:     47,
	symbol.NULS:    0.01,
	symbol.RDN:     2.5,
	symbol.XMR:     0.04,
	symbol.DLT:     19.8,
	symbol.AMB:     8.9,
	symbol.BAT:     8,
	symbol.ZEC:     0.005,
	symbol.BCPT:    14.5,
	symbol.ARN:     3,
	symbol.GVT:     0.13,
	symbol.CDT:     81,
	symbol.GXS:     0.3,
	symbol.POE:     134,
	symbol.QSP:     36,
	symbol.BTS:     1,
	symbol.XZC:     0.02,
	symbol.LSK:     0.1,
	symbol.TNT:     47,
	symbol.FUEL:    79,
	symbol.MANA:    18,
	symbol.BCD:     0.01,
	symbol.DGD:     0.04,
	symbol.ADX:     6.3,
	symbol.ADA:     1,
	symbol.PPT:     0.41,
	symbol.CMT:     12,
	symbol.XLM:     0.01,
	symbol.CND:     58,
	symbol.LEND:    84,
	symbol.WABI:    6.6,
	symbol.SBTC:    0.0005,
	symbol.BCX:     0.5,
	symbol.WAVES:   0.002,
	symbol.TNB:     139,
	symbol.GTO:     20,
	symbol.ICX:     0.02,
	symbol.OST:     32,
	symbol.ELF:     3.9,
	symbol.AION:    3.2,
	symbol.CVC:     10.9,
	symbol.REP:     0.2,
	symbol.GNT:     8.9,
	symbol.DATA:    37,
	symbol.ETF:     1,
	symbol.BRD:     3.8,
	symbol.NEBL:    0.01,
	symbol.VIBE:    17.3,
	symbol.LUN:     0.36,
	symbol.CHAT:    60.7,
	symbol.RLC:     3.4,
	symbol.INS:     3.5,
	symbol.IOST:    105.6,
	symbol.STEEM:   0.01,
	symbol.NANO:    0.01,
	symbol.AE:      1.3,
	symbol.VIA:     0.01,
	symbol.BLZ:     10.3,
	symbol.SYS:     1,
	symbol.NCASH:   247.6,
	symbol.POA:     0.01,
	symbol.ONT:     1,
	symbol.ZIL:     37.2,
	symbol.STORM:   152,
	symbol.XEM:     4,
	symbol.WAN:     0.1,
	symbol.WPR:     43.4,
	symbol.QLC:     1,
	symbol.GRS:     0.2,
	symbol.CLOAK:   0.02,
	symbol.LOOM:    11.9,
	symbol.BCN:     1,
	symbol.TUSD:    1.35,
	symbol.ZEN:     0.002,
	symbol.SKY:     0.01,
	symbol.THETA:   24,
	symbol.IOTX:    90.5,
	symbol.QKC:     24.6,
	symbol.AGI:     29.81,
	symbol.NXS:     0.02,
	symbol.SC:      0.1,
	symbol.EON:     10,
	symbol.NPXS:    897,
	symbol.KEY:     223,
	symbol.NAS:     0.1,
	symbol.ADD:     100,
	symbol.MEETONE: 300,
	symbol.ATD:     100,
	symbol.MFT:     175,
	symbol.EOP:     5,
	symbol.DENT:    596,
	symbol.IQ:      50,
	symbol.ARDR:    2,
	symbol.HOT:     1210,
	symbol.VET:     100,
	symbol.DOCK:    68,
	symbol.POLY:    7,
	symbol.VTHO:    21,
	symbol.ONG:     0.1,
	symbol.PHX:     1,
	symbol.HC:      0.005,
	symbol.GO:      0.01,
	symbol.PAX:     1.4,
	symbol.EDO:     1.3,
	symbol.WINGS:   8.9,
	symbol.NAV:     0.2,
	symbol.TRIG:    49.1,
	symbol.APPC:    12.4,
	symbol.PIVX:    0.02,
}
