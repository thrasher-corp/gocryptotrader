package coinbene

// TickerData stores ticker data
type TickerData struct {
	Symbol      string  `json:"symbol"`
	LatestPrice float64 `json:"latestPrice,string"`
	BestBid     float64 `json:"bestBid,string"`
	BestAsk     float64 `json:"bestAsk,string"`
	DailyHigh   float64 `json:"high24h,string"`
	DailyLow    float64 `json:"low24h,string"`
	DailyVol    float64 `json:"vol24h,string"`
}

// TickerResponse stores ticker response data
type TickerResponse struct {
	Code       int64      `json:"code"`
	TickerData TickerData `json:"data"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}

// OrderbookResponse stores data from fetched orderbooks
type OrderbookResponse struct {
	Code      int64     `json:"code"`
	Orderbook Orderbook `json:"data"`
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
	AvgPrice       float64 `json:"averagePrice,string"`
	CreateTime     string  `json:"createTime"`
	Fees           float64 `json:"fees"`
	FilledAmount   float64 `json:"filledamount"`
	FilledQuantity float64 `json:"filledquantity"`
	LastModified   string  `json:"lastmodified"`
	OrderID        string  `json:"orderid"`
	OrderQuantity  float64 `json:"orderquantity"`
	OrderStatus    string  `json:"orderstatus"`
	Price          float64 `json:"price"`
	Symbol         string  `json:"symbol"`
	OrderType      string  `json:"type"`
}

// OrderInfoResponse stores orderinfo data
type OrderInfoResponse struct {
	Order     OrderInfoData `json:"order"`
	Status    string        `json:"status"`
	Timestamp int64         `json:"timestamp"`
}

// RemoveOrderResponse stores data for the remove request
type RemoveOrderResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	OrderID   string `json:"orderid"`
}

// OpenOrderData stores data for open orders
type OpenOrderData struct {
	OrderID        string  `json:"orderid"`
	OrderStatus    string  `json:"orderstatus"`
	Symbol         string  `json:"symbol"`
	AvgPrice       float64 `json:"averagePrice,string"`
	CreateTime     string  `json:"createTime"`
	FilledAmount   float64 `json:"filledamount"`
	FilledQuantity float64 `json:"filledquantity"`
	LastModified   string  `json:"lastmodified"`
	OrderQuantity  float64 `json:"orderquantity"`
	Price          float64 `json:"price"`
	OrderType      string  `json:"type"`
}

// OpenOrderResponse stores data for open orders
type OpenOrderResponse struct {
	Status     string          `json:"status"`
	Timestamp  int64           `json:"timestamp"`
	OpenOrders []OpenOrderData `json:"orders"`
}
