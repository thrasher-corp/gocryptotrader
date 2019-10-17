package coinbene

// TickerData stores ticker data
type TickerData struct {
	Symbol      string  `json:"symbol"`
	LatestPrice float64 `json:"latestPrice,string"`
	BestBid     float64 `json:"bestBid,string"`
	BestAsk     float64 `json:"bestAsk,string"`
	DailyHigh   float64 `json:"high24h,string"`
	DailyLow    float64 `json:"low24h,string"`
	DailyVol    float64 `json:"volume24h,string"`
}

// TickerResponse stores ticker response data
type TickerResponse struct {
	Code       int64 `json:"code"`
	TickerData `json:"data"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}

// OrderbookResponse stores data from fetched orderbooks
type OrderbookResponse struct {
	Code      int64 `json:"code"`
	Orderbook `json:"data"`
}

// TradeResponse stores trade data
type TradeResponse struct {
	Code   int64      `json:"code"`
	Trades [][]string `json:"data"`
}

// AllPairData stores pair data
type AllPairData struct {
	Symbol           string  `json:"symbol"`
	BaseAsset        string  `json:"baseAsset"`
	QuoteAsset       string  `json:"quoteAsset"`
	PricePrecision   int64   `json:"pricePrecision,string"`
	AmountPrecision  int64   `json:"amountPrecision,string"`
	TakerFeeRate     float64 `json:"takerFeeRate,string"`
	MakerFeeRate     float64 `json:"makerFeeRate,string"`
	MinAmount        float64 `json:"minAmount,string"`
	Site             string  `json:"site"`
	PriceFluctuation string  `json:"priceFluctuation"`
}

// AllPairResponse stores data for all pairs enabled on exchange
type AllPairResponse struct {
	Code int64         `json:"code"`
	Data []AllPairData `json:"data"`
}

// PairResponse stores data for a single queried pair
type PairResponse struct {
	Code int64       `json:"code"`
	Data AllPairData `json:"data"`
}

// UserBalanceData stores user balance data
type UserBalanceData struct {
	Asset     string  `json:"asset"`
	Available float64 `json:"available,string"`
	Reserved  float64 `json:"reserved,string"`
	Total     float64 `json:"total,string"`
}

// UserBalanceResponse stores user balance data
type UserBalanceResponse struct {
	Code int64             `json:"code"`
	Data []UserBalanceData `json:"data"`
}

// PlaceOrderResponse stores data for a placed order
type PlaceOrderResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	OrderID   string `json:"orderid"`
}

// OrderInfoData stores order info
type OrderInfoData struct {
	OrderID      string  `json:"orderId"`
	BaseAsset    string  `json:"baseAsset"`
	QuoteAsset   string  `json:"quoteAsset"`
	OrderType    string  `json:"orderDirection"`
	Quantity     float64 `json:"quntity,string"`
	Amount       float64 `json:"amout,string"`
	FilledAmount float64 `json:"filledAmount"`
	TakerRate    float64 `json:"takerFeeRate,string"`
	MakerRate    float64 `json:"makerRate,string"`
	AvgPrice     float64 `json:"avgPrice,string"`
	OrderStatus  string  `json:"orderStatus"`
	OrderTime    string  `json:"orderTime"`
	TotalFee     float64 `json:"totalFee"`
}

// OrderInfoResponse stores orderinfo data
type OrderInfoResponse struct {
	Order OrderInfoData `json:"data"`
	Code  int64         `json:"code"`
}

// RemoveOrderResponse stores data for the remove request
type RemoveOrderResponse struct {
	Code    int64  `json:"code"`
	OrderID string `json:"data"`
}

// OpenOrderResponse stores data for open orders
type OpenOrderResponse struct {
	Code       int64           `json:"code"`
	OpenOrders []OrderInfoData `json:"data"`
}

// ClosedOrderResponse stores data for closed orders
type ClosedOrderResponse struct {
	Code int64           `json:"code"`
	Data []OrderInfoData `json:"data"`
}

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
	Volume24h     float64 `json:"volume,string"`
	Timestamp     string  `json:"timestamp"`
}

// WsTicker stores websocket ticker
type WsTicker struct {
	Topic string         `json:"topic"`
	Data  []WsTickerData `json:"data"`
}

// WsTradeList stores websocket tradelist data
type WsTradeList struct {
	Topic string     `json:"topic"`
	Data  [][]string `json:"data"`
}

// WsOrderbook stores websocket orderbook data
type WsOrderbook struct {
	Topic     string      `json:"topic"`
	Action    string      `json:"action"`
	Data      []Orderbook `json:"data"`
	Version   int64       `json:"version,string"`
	Timestamp string      `json:"timestamp"`
}

// WsKline stores websocket kline data
type WsKline struct {
	Topic string          `json:"topic"`
	Data  [][]interface{} `json:"data"`
}

// WsUserData stores websocket user data
type WsUserData struct {
	Asset     string  `json:"string"`
	Available float64 `json:"availableBalance"`
	Locked    float64 `json:"frozenBalance"`
	Total     float64 `json:"balance"`
	Timestamp string  `json:"timestamp"`
}

// WsUserInfo stores websocket user info
type WsUserInfo struct {
	Topic string       `json:"topic"`
	Data  []WsUserData `json:"data"`
}

// WsPositionData stores websocket info on user's position
type WsPositionData struct {
	AvailableQuantity float64 `json:"availableQuantity"`
	AvgPrice          float64 `json:"avgPrice"`
	Leverage          float64 `json:"leverage"`
	LiquidationPrice  float64 `json:"liquidationPrice"`
	MarkPrice         float64 `json:"markPrice"`
	PositionMargin    float64 `json:"positionMargin"`
	Quantity          float64 `json:"quantity"`
	RealisedPNL       float64 `json:"realisedPnl"`
	Side              string  `json:"side"`
	Symbol            string  `json:"symbol"`
	MarginMode        int64   `json:"marginMode"`
	CreateTime        string  `json:"createTime"`
}

// WsPosition stores websocket info on user's positions
type WsPosition struct {
	Topic string           `json:"topic"`
	Data  []WsPositionData `json:"data"`
}

// WsOrderData stores websocket user order data
type WsOrderData struct {
	OrderID          string  `json:"orderId"`
	Direction        string  `json:"direction"`
	Leverage         float64 `json:"leverage"`
	Symbol           string  `json:"symbol"`
	OrderType        string  `json:"orderType"`
	Quantity         float64 `json:"quantity"`
	OrderPrice       float64 `json:"orderPrice"`
	OrderValue       float64 `json:"orderValue"`
	Fee              float64 `json:"fee"`
	FilledQuantity   float64 `json:"filledQuantity"`
	AveragePrice     float64 `json:"averagePrice"`
	OrderTime        string  `json:"orderTime"`
	Status           string  `json:"status"`
	LastFillQuantity float64 `json:"lastFillQuantity"`
	LastFillPrice    float64 `json:"lastFillPrice"`
	LastFillTime     string  `json:"lastFillTime"`
}

// WsUserOrders stores websocket user orders' data
type WsUserOrders struct {
	Topic string        `json:"topic"`
	Data  []WsOrderData `json:"data"`
}
