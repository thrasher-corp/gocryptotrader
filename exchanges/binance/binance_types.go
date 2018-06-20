package binance

import (
	"encoding/json"
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
	Code int
	Msg  string
	Bids []struct {
		Price    float64
		Quantity float64
	}
	Asks []struct {
		Price    float64
		Quantity float64
	}
}

// RecentTrade holds recent trade data
type RecentTrade struct {
	Code         int     `json:"code"`
	Msg          string  `json:"msg"`
	ID           int64   `json:"id"`
	Price        float64 `json:"price,string"`
	Quantity     float64 `json:"qty,string"`
	Time         int64   `json:"time"`
	IsBuyerMaker bool    `json:"isBuyerMaker"`
	IsBestMatch  bool    `json:"isBestMatch"`
}

type MultiStreamData struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

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
	// Symbol 交易对，必填项
	Symbol string
	// Side 交易方式，买或卖，必填写项
	Side BinanceRequestParamsSideType
	// TradeType 交易类型，市价或限价等
	TradeType BinanceRequestParamsOrderType
	// TimeInForce 不知道有毛用
	TimeInForce BinanceRequestParamsTimeForceType
	// Quantity 数量
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
	Time          int64   `json:"time"`
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

// BinanceInterval 要检索的时间段
type BinanceInterval string

var (
	// BinanceIntervalMinute 1m
	BinanceIntervalMinute         = BinanceInterval("1m")
	BinanceIntervalThreeMinutes   = BinanceInterval("3m")
	BinanceIntervalFiveMinutes    = BinanceInterval("5m")
	BinanceIntervalFifteenMinutes = BinanceInterval("15m")
	BinanceIntervalThirtyMinutes  = BinanceInterval("30m")
	BinanceIntervalHour           = BinanceInterval("1h")
	BinanceIntervalTwoHours       = BinanceInterval("2h")
	BinanceIntervalFourHours      = BinanceInterval("4h")
	BinanceIntervalSixHours       = BinanceInterval("6h")
	BinanceIntervalEightHours     = BinanceInterval("8h")
	BinanceIntervalTwelveHours    = BinanceInterval("12h")
	BinanceIntervalDay            = BinanceInterval("1d")
	BinanceIntervalThreeDays      = BinanceInterval("3d")
	BinanceIntervalWeek           = BinanceInterval("1w")
	BinanceIntervalMonth          = BinanceInterval("1M")
)

// BinanceRequestParamsSideType 交易类型
type BinanceRequestParamsSideType string

var (
	// BinanceRequestParamsSideBuy 买
	BinanceRequestParamsSideBuy = BinanceRequestParamsSideType("BUY")

	// BinanceRequestParamsSideSell 卖
	BinanceRequestParamsSideSell = BinanceRequestParamsSideType("SELL")
)

// BinanceRequestParamsTimeForceType Time in force
type BinanceRequestParamsTimeForceType string

var (
	// BinanceRequestParamsTimeGTC GTC
	BinanceRequestParamsTimeGTC = BinanceRequestParamsTimeForceType("GTC")

	// BinanceRequestParamsTimeIOC IOC
	BinanceRequestParamsTimeIOC = BinanceRequestParamsTimeForceType("IOC")

	// BinanceRequestParamsTimeFOK FOK
	BinanceRequestParamsTimeFOK = BinanceRequestParamsTimeForceType("FOK")
)

// BinanceRequestParamsOrderType 交易类型
type BinanceRequestParamsOrderType string

var (
	// BinanceRequestParamsOrderLimit 限价
	BinanceRequestParamsOrderLimit = BinanceRequestParamsOrderType("LIMIT")

	// BinanceRequestParamsOrderMarket 市场价
	BinanceRequestParamsOrderMarket = BinanceRequestParamsOrderType("MARKET")

	// BinanceRequestParamsOrderStopLoss STOP_LOSS
	BinanceRequestParamsOrderStopLoss = BinanceRequestParamsOrderType("STOP_LOSS")

	// BinanceRequestParamsOrderStopLossLimit STOP_LOSS_LIMIT
	BinanceRequestParamsOrderStopLossLimit = BinanceRequestParamsOrderType("STOP_LOSS_LIMIT")

	// BinanceRequestParamsOrderTakeProfit STOP_LOSS_LIMIT
	BinanceRequestParamsOrderTakeProfit = BinanceRequestParamsOrderType("TAKE_PROFIT")

	// BinanceRequestParamsOrderTakeProfitLimit STOP_LOSS_LIMIT
	BinanceRequestParamsOrderTakeProfitLimit = BinanceRequestParamsOrderType("TAKE_PROFIT_LIMIT")

	// BinanceRequestParamsOrderLimitMarker STOP_LOSS_LIMIT
	BinanceRequestParamsOrderLimitMarker = BinanceRequestParamsOrderType("LIMIT_MAKER")
)
