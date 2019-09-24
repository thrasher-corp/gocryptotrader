package coinbene

// TickerData stores ticker data
type TickerData struct {
	Symbol    string  `json:"symbol"`
	Last      float64 `json:"last,string"`
	Bid       float64 `json:"bid,string"`
	Ask       float64 `json:"ask,string"`
	DailyHigh float64 `json:"24hrHigh,string"`
	DailyLow  float64 `json:"24hrLow,string"`
	DailyVol  float64 `json:"24hrVol,string"`
	DailyAmt  float64 `json:"24hrAmt,string"`
}

// TickerResponse stores ticker response data
type TickerResponse struct {
	Status     string       `json:"status"`
	Timestamp  int64        `json:"timestamp"`
	TickerData []TickerData `json:"ticker"`
}

// OrderbookData stores data from orderbooks
type OrderbookData struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	Asks []OrderbookData `json:"asks"`
	Bids []OrderbookData `json:"bids"`
}

// OrderbookResponse stores data from fetched orderbooks
type OrderbookResponse struct {
	Orderbook Orderbook `json:"orderbook"`
	Status    string    `json:"status"`
	Symbol    string    `json:"symbol"`
	Timestamp int64     `json:"timestamp"`
}

// TradeData stores trade data
type TradeData struct {
	TradeID  string  `json:"tradeId"`
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"quantity,string"`
	Take     string  `json:"take"`
	Time     int64   `json:"time"`
}

// TradeResponse stores trade data
type TradeResponse struct {
	Status    string      `json:"status"`
	Timestamp int64       `json:"timestamp"`
	Symbol    string      `json:"symbol"`
	Trades    []TradeData `json:"trades"`
}

// AllPairData stores pair data
type AllPairData struct {
	Symbol      string  `json:"ticker"`
	BaseAsset   string  `json:"baseAsset"`
	QuoteAsset  string  `json:"quoteAsset"`
	TakerFee    float64 `json:"takerFee,string"`
	MakerFee    float64 `json:"makerFee,string"`
	TickSize    int     `json:"tickSize,string"`
	LotStepSize int     `json:"lotStepSize,string"`
	MinQuantity float64 `json:"minQuantity,string"`
}

// AllPairResponse stores data for all pairs enabled on exchange
type AllPairResponse struct {
	Status    string        `json:"status"`
	Timestamp int64         `json:"timestamp"`
	Symbol    []AllPairData `json:"symbol"`
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
	Account   string            `json:"account"`
	Balance   []UserBalanceData `json:"balance"`
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
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

// WithdrawResponse stores response for a withdraw request
type WithdrawResponse struct {
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
	WithdrawID string `json:"withdrawid"`
}
