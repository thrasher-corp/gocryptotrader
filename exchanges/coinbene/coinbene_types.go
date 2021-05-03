package coinbene

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TickerData stores ticker data
type TickerData struct {
	Symbol      string  `json:"symbol"`
	LatestPrice float64 `json:"latestPrice,string"`
	BestBid     float64 `json:"bestBid,string"`
	BestAsk     float64 `json:"bestAsk,string"`
	DailyHigh   float64 `json:"high24h,string"`
	DailyLow    float64 `json:"low24h,string"`
	DailyVolume float64 `json:"volume24h,string"`
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
	Count  int64
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
	Volume       float64
	Direction    string
	TradeTime    time.Time
}

// Trades stores trade data
type Trades []TradeItem

// PairData stores pair data
type PairData struct {
	Symbol           string  `json:"symbol"`
	BaseAsset        string  `json:"baseAsset"`
	QuoteAsset       string  `json:"quoteAsset"`
	PricePrecision   int64   `json:"pricePrecision,string"`
	AmountPrecision  int64   `json:"amountPrecision,string"`
	TakerFeeRate     float64 `json:"takerFeeRate,string"`
	MakerFeeRate     float64 `json:"makerFeeRate,string"`
	MinAmount        float64 `json:"minAmount,string"`
	Site             string  `json:"site"`
	PriceFluctuation float64 `json:"priceFluctuation,string"`
}

// UserBalanceData stores user balance data
type UserBalanceData struct {
	Asset     string  `json:"asset"`
	Available float64 `json:"available,string"`
	Reserved  float64 `json:"reserved,string"`
	Total     float64 `json:"total,string"`
}

// PlaceOrderRequest places an order request
type PlaceOrderRequest struct {
	Price     float64
	Quantity  float64
	Symbol    string
	Direction string
	OrderType string
	ClientID  string
	Notional  int
}

// CancelOrdersResponse stores data for a cancelled order
type CancelOrdersResponse struct {
	OrderID string `json:"orderId"`
	Message string `json:"message"`
}

// OrderInfo stores order info
type OrderInfo struct {
	OrderID      string    `json:"orderId"`
	BaseAsset    string    `json:"baseAsset"`
	QuoteAsset   string    `json:"quoteAsset"`
	OrderType    string    `json:"orderDirection"`
	Quantity     float64   `json:"quntity,string"`
	Amount       float64   `json:"amout,string"`
	FilledAmount float64   `json:"filledAmount"`
	TakerRate    float64   `json:"takerFeeRate,string"`
	MakerRate    float64   `json:"makerFeeRate,string"`
	AvgPrice     float64   `json:"avgPrice,string"`
	OrderPrice   float64   `json:"orderPrice,string"`
	OrderStatus  string    `json:"orderStatus"`
	OrderTime    time.Time `json:"orderTime"`
	TotalFee     float64   `json:"totalFee"`
}

// OrderFills stores the fill info
type OrderFills struct {
	Price     float64   `json:"price,string"`
	Quantity  float64   `json:"quantity,string"`
	Amount    float64   `json:"amount,string"`
	Fee       float64   `json:"fee,string"`
	Direction string    `json:"direction"`
	TradeTime time.Time `json:"tradeTime"`
	FeeByConi float64   `json:"feeByConi,string"`
}

// OrdersInfo stores a collection of orders
type OrdersInfo []OrderInfo

// WsSub stores subscription data
type WsSub struct {
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

// WsTickerData stores websocket ticker data
type WsTickerData struct {
	Symbol        string  `json:"symbol"`
	LastPrice     float64 `json:"lastPrice,string"`
	MarkPrice     float64 `json:"markPrice,string"`
	BestAskPrice  float64 `json:"bestAskPrice,string"`
	BestBidPrice  float64 `json:"bestBidPrice,string"`
	BestAskVolume float64 `json:"bestAskVolume,string"`
	BestBidVolume float64 `json:"bestBidVolume,string"`
	High24h       float64 `json:"high24h,string"`
	Low24h        float64 `json:"low24h,string"`
	Volume24h     float64 `json:"volume24h,string"`
	Timestamp     int64   `json:"timestamp"`
}

// WsTicker stores websocket ticker
type WsTicker struct {
	Topic string         `json:"topic"`
	Data  []WsTickerData `json:"data"`
}

// WsTradeList stores websocket tradelist data
type WsTradeList struct {
	Topic string           `json:"topic"`
	Data  [][4]interface{} `json:"data"`
}

// WsTradeData stores trade data for websocket
type WsTradeData struct {
	BestAskPrice float64 `json:"bestAskPrice,string"`
	BestBidPrice float64 `json:"bestBidPrice,string"`
	High24h      float64 `json:"high24h,string"`
	LastPrice    float64 `json:"lastPrice,string"`
	Low24h       float64 `json:"low24h,string"`
	Open24h      float64 `json:"open24h,string"`
	OpenPrice    float64 `json:"openPrice,string"`
	Symbol       string  `json:"symbol"`
	Timestamp    int64   `json:"timestamp"`
	Volume24h    float64 `json:"volume24h,string"`
}

// WsKline stores websocket kline data
type WsKline struct {
	Topic string        `json:"topic"`
	Data  []WsKLineData `json:"data"`
}

// WsKLineData holds OHLCV data
type WsKLineData struct {
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
	Timestamp int64   `json:"t"`
}

// WsUserData stores websocket user data
type WsUserData struct {
	Asset     string    `json:"string"`
	Available float64   `json:"availableBalance,string"`
	Locked    float64   `json:"frozenBalance,string"`
	Total     float64   `json:"balance,string"`
	Timestamp time.Time `json:"timestamp"`
}

// WsUserInfo stores websocket user info
type WsUserInfo struct {
	Topic string       `json:"topic"`
	Data  []WsUserData `json:"data"`
}

// WsPositionData stores websocket info on user's position
type WsPositionData struct {
	AvailableQuantity float64   `json:"availableQuantity,string"`
	AveragePrice      float64   `json:"avgPrice,string"`
	Leverage          int64     `json:"leverage,string"`
	LiquidationPrice  float64   `json:"liquidationPrice,string"`
	MarkPrice         float64   `json:"markPrice,string"`
	PositionMargin    float64   `json:"positionMargin,string"`
	Quantity          float64   `json:"quantity,string"`
	RealisedPNL       float64   `json:"realisedPnl,string"`
	Side              string    `json:"side"`
	Symbol            string    `json:"symbol"`
	MarginMode        int64     `json:"marginMode,string"`
	CreateTime        time.Time `json:"createTime"`
}

// WsPosition stores websocket info on user's positions
type WsPosition struct {
	Topic string           `json:"topic"`
	Data  []WsPositionData `json:"data"`
}

// WsOrderbookData stores ws orderbook data
type WsOrderbookData struct {
	Topic  string `json:"topic"`
	Action string `json:"action"`
	Data   []struct {
		Bids      [][2]string `json:"bids"`
		Asks      [][2]string `json:"asks"`
		Version   int64       `json:"version"`
		Timestamp int64       `json:"timestamp"`
	} `json:"data"`
}

// WsOrderData stores websocket user order data
type WsOrderData struct {
	OrderID          string    `json:"orderId"`
	Direction        string    `json:"direction"`
	Leverage         int64     `json:"leverage,string"`
	Symbol           string    `json:"symbol"`
	OrderType        string    `json:"orderType"`
	Quantity         float64   `json:"quantity,string"`
	OrderPrice       float64   `json:"orderPrice,string"`
	OrderValue       float64   `json:"orderValue,string"`
	Fee              float64   `json:"fee,string"`
	FilledQuantity   float64   `json:"filledQuantity,string"`
	AveragePrice     float64   `json:"averagePrice,string"`
	OrderTime        time.Time `json:"orderTime"`
	Status           string    `json:"status"`
	LastFillQuantity float64   `json:"lastFillQuantity,string"`
	LastFillPrice    float64   `json:"lastFillPrice,string"`
	LastFillTime     string    `json:"lastFillTime"`
}

// WsUserOrders stores websocket user orders' data
type WsUserOrders struct {
	Topic string        `json:"topic"`
	Data  []WsOrderData `json:"data"`
}

// SwapTicker stores the swap ticker info
type SwapTicker struct {
	LastPrice      float64   `json:"lastPrice,string"`
	MarkPrice      float64   `json:"markPrice,string"`
	BestAskPrice   float64   `json:"bestAskPrice,string"`
	BestBidPrice   float64   `json:"bestBidPrice,string"`
	High24Hour     float64   `json:"high24h,string"`
	Low24Hour      float64   `json:"low24h,string"`
	Volume24Hour   float64   `json:"volume24h,string"`
	BestAskVolume  float64   `json:"bestAskVolume,string"`
	BestBidVolume  float64   `json:"bestBidVolume,string"`
	Turnover       float64   `json:"turnover,string"`
	Timestamp      time.Time `json:"timeStamp"`
	Change24Hour   float64   `json:"chg24h,string"`
	ChangeZeroHour float64   `json:"chg0h,string"`
}

// SwapTickers stores a map of swap tickers
type SwapTickers map[string]SwapTicker

// SwapKlineItem stores an individual kline data item
type SwapKlineItem struct {
	Time        time.Time
	Open        float64
	Close       float64
	High        float64
	Low         float64
	Volume      float64
	Turnover    float64
	BuyVolume   float64
	BuyTurnover float64
}

// SwapKlines stores an array of kline data
type SwapKlines []SwapKlineItem

// Instrument stores an individual tradable instrument
type Instrument struct {
	InstrumentID       currency.Pair `json:"instrumentId"`
	Multiplier         float64       `json:"multiplier,string"`
	MinimumAmount      float64       `json:"minAmount,string"`
	MaximumAmount      float64       `json:"maxAmount,string"`
	MinimumPriceChange float64       `json:"minPriceChange,string"`
	PricePrecision     int64         `json:"pricePrecision,string"`
}

// SwapTrade stores an individual trade
type SwapTrade struct {
	Price  float64
	Side   order.Side
	Volume float64
	Time   time.Time
}

// SwapTrades stores an array of swap trades
type SwapTrades []SwapTrade

// SwapAccountInfo returns the swap account balance info
type SwapAccountInfo struct {
	AvailableBalance        float64 `json:"availableBalance,string"`
	FrozenBalance           float64 `json:"frozenBalance,string"`
	MarginBalance           float64 `json:"marginBalance,string"`
	MarginRate              float64 `json:"marginRate,string"`
	Balance                 float64 `json:"balance,string"`
	UnrealisedProfitAndLoss float64 `json:"unrealisedPnl,string"`
}

// SwapPosition stores a single swap position's data
type SwapPosition struct {
	AvailableQuantity       float64   `json:"availableQuantity,string"`
	AveragePrice            float64   `json:"averagePrice,string"`
	CreateTime              time.Time `json:"createTime"`
	DeleveragePercentile    int64     `json:"deleveragePercentile,string"`
	Leverage                int64     `json:"leverage,string"`
	LiquidationPrice        float64   `json:"liquidationPrice,string"`
	MarkPrice               float64   `json:"markPrice,string"`
	PositionMargin          float64   `json:"positionMargin,string"`
	PositionValue           float64   `json:"positionValue,string"`
	Quantity                float64   `json:"quantity,string"`
	RateOfReturn            float64   `json:"roe,string"`
	Side                    string    `json:"side"`
	Symbol                  string    `json:"symbol"`
	UnrealisedProfitAndLoss float64   `json:"UnrealisedPnl,string"`
}

// SwapPositions stores a collection of swap positions
type SwapPositions []SwapPosition

// SwapPlaceOrderResponse stores the response data for placing a swap order
type SwapPlaceOrderResponse struct {
	OrderID  string `json:"orderId"`
	ClientID string `json:"clientId"`
}

// SwapOrder stores the swap order data
type SwapOrder struct {
	OrderID        string    `json:"orderId"`
	Direction      string    `json:"direction"`
	Leverage       int64     `json:"leverage,string"`
	OrderType      string    `json:"orderType"`
	Quantity       float64   `json:"quantity,string"`
	OrderPrice     float64   `json:"orderPrice,string"`
	OrderValue     float64   `json:"orderValue,string"`
	Fee            float64   `json:"fee"`
	FilledQuantity float64   `json:"filledQuantity,string"`
	AveragePrice   float64   `json:"averagePrice,string"`
	OrderTime      time.Time `json:"orderTime"`
	Status         string    `json:"status"`
}

// SwapOrders stores a collection of swap orders
type SwapOrders []SwapOrder

// OrderCancellationResponse returns a list of cancel order status
type OrderCancellationResponse struct {
	OrderID string `json:"orderId"`
	Code    int    `json:"code,string"`
	Message string `json:"message"`
}

// OrderPlacementResponse stores the order placement data
type OrderPlacementResponse OrderCancellationResponse

// SwapOrderFill stores a swap orders fill info
type SwapOrderFill struct {
	Symbol    string    `json:"symbol"`
	TradeTime time.Time `json:"tradeTime"`
	TradeID   int64     `json:"tradeId,string"`
	OrderID   int64     `json:"orderId,string"`
	Price     float64   `json:"price,string"`
	Fee       float64   `json:"fee,string"`
	ExecType  string    `json:"execType"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity,string"`
}

// SwapOrderFills stores a collection of swap order fills
type SwapOrderFills []SwapOrderFill

// SwapFundingRate stores a collection of funding rates
type SwapFundingRate struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	MarkPrice     float64 `json:"markPrice,string"`
	PositionValue float64 `json:"positionValue,string"`
	Fee           float64 `json:"fee,string"`
	FeeRate       float64 `json:"feeRate,string"`
	Leverage      int64   `json:"leverage"`
}

// CandleResponse stores returned kline data
type CandleResponse struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    [][]interface{} `json:"data"`
}
